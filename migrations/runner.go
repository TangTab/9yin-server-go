// Package migrations 以 advisory lock 和不可变的 SHA-256 校验和来装载并
// 重放数据库迁移。两种文件形态（见 migrations/README.md）：
//
//	NNNN_*.sql     本仓库手写增量，语句以 “-- migrate:statement” 分隔，
//	               校验和为文件字节；
//	NNNN_*.sql.gz  Java 服务端沿用的 tar+gzip 权威存档（mysqldump 风格
//	               多语句，文件自身幂等），按词法切分语句，校验和为内部
//	               SQL 文本（重打包不漂移）。
//
// 台账使用独立的 sql_replay_ledger 表：schema_migrations 在权威存档里是
// 随 0002 配置数据 REPLACE INTO 的普通表，不能承载本包的版本判定。
package migrations

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed *.sql*
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

// Load 从任意文件系统解析迁移集合（文件名形如 NNNN_name.sql[.gz]），
// 按版本升序返回，版本不得重复。
func Load(fsys fs.FS) ([]Migration, error) {
	var names []string
	for _, pattern := range []string{"*.sql", "*.sql.gz"} {
		entries, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, err
		}
		names = append(names, entries...)
	}
	result := make([]Migration, 0, len(names))
	for _, name := range names {
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
		var checksum [32]byte
		var statements []string
		if strings.HasSuffix(base, ".sql.gz") {
			inner, err := gunzipSQLArchive(data)
			if err != nil {
				return nil, fmt.Errorf("migration %s: %w", base, err)
			}
			checksum = sha256.Sum256(inner)
			statements = splitPlainStatements(string(inner))
		} else {
			checksum = sha256.Sum256(data)
			statements, err = splitStatements(string(data))
			if err != nil {
				return nil, fmt.Errorf("migration %s: %w", base, err)
			}
		}
		if len(statements) == 0 {
			return nil, fmt.Errorf("migration %s: no statements", base)
		}
		result = append(result, Migration{Version: version, Name: base, Checksum: checksum, SQL: statements})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Version < result[j].Version })
	for index := 1; index < len(result); index++ {
		if result[index-1].Version == result[index].Version {
			return nil, fmt.Errorf("migration: duplicate version %d", result[index].Version)
		}
	}
	return result, nil
}

// gunzipSQLArchive 解出 .sql.gz 内的 SQL 文本：优先按 tar 单成员归档解包
// （export-sql.bat 的打包约定），解不出 tar 结构时按裸 gzip 的解压结果对待。
func gunzipSQLArchive(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gunzip: %w", err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("gunzip: %w", err)
	}
	tr := tar.NewReader(bytes.NewReader(raw))
	if header, err := tr.Next(); err == nil && header.Typeflag == tar.TypeReg {
		return io.ReadAll(tr)
	}
	return raw, nil
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

// splitPlainStatements 把 mysqldump 风格的多语句 SQL 按词法切成单条语句：
// '…'/"…"/`…` 引号串内（'…" 反斜杠转义、` 用 `` 转义）、-- / # 行注释和
// /*…*/ 块注释内的分号不作为语句结尾；只含注释与空白的片段被丢弃。
func splitPlainStatements(data string) []string {
	var statements []string
	start, i, n := 0, 0, len(data)
	flush := func(end int) {
		if statement := strings.TrimSpace(data[start:end]); !commentOnly(statement) {
			statements = append(statements, statement)
		}
		start = end + 1
	}
	for i < n {
		switch c := data[i]; {
		case c == '\'' || c == '"' || c == '`':
			i = skipQuoted(data, i)
		case c == '-' && strings.HasPrefix(data[i:], "--"):
			i = skipLine(data, i)
		case c == '#':
			i = skipLine(data, i)
		case c == '/' && strings.HasPrefix(data[i:], "/*"):
			i = skipBlock(data, i)
		case c == ';':
			flush(i)
			i++
		default:
			i++
		}
	}
	if start < n {
		flush(n)
	}
	return statements
}

// skipQuoted 跳过一个引号串（含转义），返回紧随其后的下标；未闭合则到串尾。
func skipQuoted(data string, index int) int {
	quote := data[index]
	for index++; index < len(data); index++ {
		switch data[index] {
		case '\\':
			if quote != '`' {
				index++ // 反斜杠转义（反引号串内不生效）
			}
		case quote:
			if quote == '`' && index+1 < len(data) && data[index+1] == '`' {
				index++ // `` 转义
				continue
			}
			return index + 1
		}
	}
	return len(data)
}

func skipLine(data string, index int) int {
	if next := strings.IndexByte(data[index:], '\n'); next >= 0 {
		return index + next + 1
	}
	return len(data)
}

func skipBlock(data string, index int) int {
	if end := strings.Index(data[index+2:], "*/"); end >= 0 {
		return index + 2 + end + 2
	}
	return len(data)
}

// commentOnly 判断片段是否只含注释与空白（这样的“语句”不发给服务端）。
// /*!…*/ 是可执行版本注释（SET NAMES / SET TIME_ZONE 等），不算纯注释。
func commentOnly(statement string) bool {
	for i, n := 0, len(statement); i < n; {
		switch c := statement[i]; {
		case c == '\'' || c == '"' || c == '`':
			return false
		case c == '-' && strings.HasPrefix(statement[i:], "--"), c == '#':
			i = skipLine(statement, i)
		case c == '/' && strings.HasPrefix(statement[i:], "/*!"):
			return false
		case c == '/' && strings.HasPrefix(statement[i:], "/*"):
			i = skipBlock(statement, i)
		case c == ' ', c == '\t', c == '\r', c == '\n':
			i++
		default:
			return false
		}
	}
	return true
}

type Runner struct {
	DB       *sql.DB
	LockName string
	Timeout  int
	// Migrations 为待应用集合；nil 表示使用编译时嵌入的 Embedded()。
	Migrations []Migration
	// LedgerTable 为台账表名，缺省 sql_replay_ledger。
	LedgerTable string
	// Force 时无视台账跳过与校验和差异，全量重放（存档文件按约定幂等，
	// 用于缺表补建、按存档恢复配置）。
	Force bool
}

// source 返回待应用的迁移集合。
func (runner Runner) source() ([]Migration, error) {
	if runner.Migrations != nil {
		return runner.Migrations, nil
	}
	return Embedded()
}

var ledgerIdent = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func (runner Runner) ledgerTable() (string, error) {
	name := runner.LedgerTable
	if name == "" {
		name = "sql_replay_ledger"
	}
	if !ledgerIdent.MatchString(name) {
		return "", fmt.Errorf("migration: invalid ledger table %q", name)
	}
	return name, nil
}

func (runner Runner) Up(ctx context.Context) error {
	migrations, err := runner.source()
	if err != nil {
		return err
	}
	table, err := runner.ledgerTable()
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

	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  version BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  checksum BINARY(32) NOT NULL,
  applied_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`, table)); err != nil {
		return fmt.Errorf("migration: create ledger: %w", err)
	}

	applied, err := loadApplied(ctx, conn, table)
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		if checksum, exists := applied[migration.Version]; exists && !runner.Force {
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
		if _, err := conn.ExecContext(ctx, fmt.Sprintf(`REPLACE INTO %s(version, name, checksum) VALUES (?, ?, ?)`, table),
			migration.Version, migration.Name, migration.Checksum[:]); err != nil {
			return fmt.Errorf("migration %d: record checksum: %w", migration.Version, err)
		}
	}
	return nil
}

func loadApplied(ctx context.Context, conn *sql.Conn, table string) (map[uint64][32]byte, error) {
	rows, err := conn.QueryContext(ctx, fmt.Sprintf(`SELECT version, checksum FROM %s ORDER BY version`, table))
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

// Status 对照台账列出迁移状态；只读、不加锁、不修改。
// 台账表尚不存在时，所有迁移均报告为未应用。
func (runner Runner) Status(ctx context.Context, list []Migration) ([]MigrationStatus, error) {
	table, err := runner.ledgerTable()
	if err != nil {
		return nil, err
	}
	ledger, err := readLedger(ctx, runner.DB, table)
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
func readLedger(ctx context.Context, db *sql.DB, table string) (map[uint64]ledgerEntry, error) {
	var exists int
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&exists); err != nil {
		return nil, fmt.Errorf("migration: probe ledger: %w", err)
	}
	if exists == 0 {
		return map[uint64]ledgerEntry{}, nil
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`SELECT version, checksum, CAST(UNIX_TIMESTAMP(applied_at) AS SIGNED) FROM %s ORDER BY version`, table))
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
