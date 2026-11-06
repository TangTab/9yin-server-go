// Package migrations 以 advisory lock 和不可变的 SHA-256 校验和
// 来嵌入并应用数据库 schema。
package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed *.sql
var files embed.FS

var (
	ErrLockUnavailable  = errors.New("migration: schema lock unavailable")
	ErrChecksumMismatch = errors.New("migration: applied checksum differs")
)

type Migration struct {
	Version  uint64
	Name     string
	Checksum [32]byte
	SQL      []string
}

// Embedded 返回编译时嵌入的迁移集合。
func Embedded() ([]Migration, error) {
	return Load(files)
}

// Load 从任意文件系统解析迁移集合（文件名形如 NNNN_name.sql，
// 语句以 “-- migrate:statement” 分隔），按版本升序返回。
func Load(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, err
	}
	result := make([]Migration, 0, len(entries))
	for _, name := range entries {
		base := filepath.Base(name)
		separator := strings.IndexByte(base, '_')
		if separator < 1 {
			return nil, fmt.Errorf("migration: invalid filename %q", base)
		}
		version, err := strconv.ParseUint(base[:separator], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("migration: invalid version in %q: %w", base, err)
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		statements, err := splitStatements(string(data))
		if err != nil {
			return nil, fmt.Errorf("migration %s: %w", base, err)
		}
		result = append(result, Migration{Version: version, Name: base, Checksum: sha256.Sum256(data), SQL: statements})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Version < result[j].Version })
	for index := 1; index < len(result); index++ {
		if result[index-1].Version == result[index].Version {
			return nil, fmt.Errorf("migration: duplicate version %d", result[index].Version)
		}
	}
	return result, nil
}

func splitStatements(data string) ([]string, error) {
	const marker = "-- migrate:statement"
	parts := strings.Split(data, marker)
	statements := make([]string, 0, len(parts)-1)
	if strings.TrimSpace(parts[0]) != "" {
		return nil, errors.New("content before first statement marker")
	}
	for _, part := range parts[1:] {
		statement := strings.TrimSpace(part)
		if statement == "" {
			return nil, errors.New("empty statement")
		}
		statements = append(statements, statement)
	}
	if len(statements) == 0 {
		return nil, errors.New("no statements")
	}
	return statements, nil
}

type Runner struct {
	DB       *sql.DB
	LockName string
	Timeout  int
	// Migrations 为待应用集合；nil 表示使用编译时嵌入的 Embedded()。
	Migrations []Migration
}

// source 返回待应用的迁移集合。
func (runner Runner) source() ([]Migration, error) {
	if runner.Migrations != nil {
		return runner.Migrations, nil
	}
	return Embedded()
}

func (runner Runner) Up(ctx context.Context) error {
	migrations, err := runner.source()
	if err != nil {
		return err
	}
	conn, err := runner.DB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migration: connect: %w", err)
	}
	defer conn.Close()
	lockName := runner.LockName
	if lockName == "" {
		lockName = "nineyin_schema_migration"
	}
	timeout := runner.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	var locked sql.NullInt64
	if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, ?)`, lockName, timeout).Scan(&locked); err != nil {
		return fmt.Errorf("migration: acquire lock: %w", err)
	}
	if !locked.Valid || locked.Int64 != 1 {
		return ErrLockUnavailable
	}
	defer conn.ExecContext(context.Background(), `SELECT RELEASE_LOCK(?)`, lockName)

	if _, err := conn.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  checksum BINARY(32) NOT NULL,
  applied_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("migration: create ledger: %w", err)
	}

	applied, err := loadApplied(ctx, conn)
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		if checksum, exists := applied[migration.Version]; exists {
			if checksum != migration.Checksum {
				return fmt.Errorf("%w: version %d (%s)", ErrChecksumMismatch, migration.Version, migration.Name)
			}
			continue
		}
		for index, statement := range migration.SQL {
			if _, err := conn.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migration %d (%s), statement %d: %w", migration.Version, migration.Name, index+1, err)
			}
		}
		if _, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations(version, name, checksum) VALUES (?, ?, ?)`,
			migration.Version, migration.Name, migration.Checksum[:]); err != nil {
			return fmt.Errorf("migration %d: record checksum: %w", migration.Version, err)
		}
	}
	return nil
}

func loadApplied(ctx context.Context, conn *sql.Conn) (map[uint64][32]byte, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version, checksum FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("migration: read ledger: %w", err)
	}
	defer rows.Close()
	result := make(map[uint64][32]byte)
	for rows.Next() {
		var version uint64
		var raw []byte
		if err := rows.Scan(&version, &raw); err != nil {
			return nil, fmt.Errorf("migration: scan ledger: %w", err)
		}
		if len(raw) != sha256.Size {
			return nil, fmt.Errorf("migration: invalid checksum length for version %d", version)
		}
		var checksum [32]byte
		copy(checksum[:], raw)
		result[version] = checksum
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migration: read ledger: %w", err)
	}
	return result, nil
}

// MigrationStatus 是 Status 对照台账得出的单条迁移状态。
type MigrationStatus struct {
	Version    uint64
	Name       string
	Checksum   [32]byte
	Applied    bool
	ChecksumOK bool
	// AppliedAt 为台账记录的应用时刻（未应用时为零值）。
	AppliedAt time.Time
}

type ledgerEntry struct {
	checksum  [32]byte
	appliedAt time.Time
}

// Status 对照 schema_migrations 台账列出迁移状态；只读、不加锁、不修改。
// 台账表尚不存在时，所有迁移均报告为未应用。
func (runner Runner) Status(ctx context.Context, list []Migration) ([]MigrationStatus, error) {
	ledger, err := readLedger(ctx, runner.DB)
	if err != nil {
		return nil, err
	}
	result := make([]MigrationStatus, 0, len(list))
	for _, migration := range list {
		status := MigrationStatus{Version: migration.Version, Name: migration.Name, Checksum: migration.Checksum}
		if entry, exists := ledger[migration.Version]; exists {
			status.Applied = true
			status.ChecksumOK = entry.checksum == migration.Checksum
			status.AppliedAt = entry.appliedAt
		}
		result = append(result, status)
	}
	return result, nil
}

// readLedger 读取台账（表不存在时返回空映射）。UNIX_TIMESTAMP 取 epoch，
// 避免依赖 DSN 的 parseTime 参数。
func readLedger(ctx context.Context, db *sql.DB) (map[uint64]ledgerEntry, error) {
	var exists int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = 'schema_migrations'`).Scan(&exists); err != nil {
		return nil, fmt.Errorf("migration: probe ledger: %w", err)
	}
	if exists == 0 {
		return map[uint64]ledgerEntry{}, nil
	}
	rows, err := db.QueryContext(ctx, `SELECT version, checksum, CAST(UNIX_TIMESTAMP(applied_at) AS SIGNED) FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("migration: read ledger: %w", err)
	}
	defer rows.Close()
	result := make(map[uint64]ledgerEntry)
	for rows.Next() {
		var version uint64
		var raw []byte
		var seconds sql.NullInt64
		if err := rows.Scan(&version, &raw, &seconds); err != nil {
			return nil, fmt.Errorf("migration: scan ledger: %w", err)
		}
		if len(raw) != sha256.Size {
			return nil, fmt.Errorf("migration: invalid checksum length for version %d", version)
		}
		entry := ledgerEntry{appliedAt: time.Unix(seconds.Int64, 0)}
		copy(entry.checksum[:], raw)
		result[version] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migration: read ledger: %w", err)
	}
	return result, nil
}
