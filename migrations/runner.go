// Package migrations embeds and applies the database schema with an advisory
// lock and immutable SHA-256 checksums.
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

func Embedded() ([]Migration, error) {
	entries, err := fs.Glob(files, "*.sql")
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
		data, err := files.ReadFile(name)
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
}

func (runner Runner) Up(ctx context.Context) error {
	migrations, err := Embedded()
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
