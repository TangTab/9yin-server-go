package migrations

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestEmbeddedJavaArchivesLoad(t *testing.T) {
	all, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 13 {
		t.Fatalf("got %d migrations, want at least 13", len(all))
	}
	for index, migration := range all {
		if !strings.HasSuffix(migration.Name, ".sql.gz") {
			t.Fatalf("migration %d: unexpected name %q", index, migration.Name)
		}
		if len(migration.SQL) == 0 {
			t.Fatalf("migration %d (%s): no statements", index, migration.Name)
		}
		if index > 0 && all[index-1].Version >= migration.Version {
			t.Fatalf("versions not strictly increasing at %d", index)
		}
	}
	if !strings.Contains(strings.Join(all[0].SQL, "\n"), "CREATE TABLE IF NOT EXISTS") {
		t.Fatal("0001 schema archive lacks CREATE TABLE IF NOT EXISTS")
	}
	if !strings.Contains(strings.Join(all[1].SQL, "\n"), "REPLACE INTO") {
		t.Fatal("0002 config archive lacks REPLACE INTO")
	}
}

func TestLoadSqlGzTarArchive(t *testing.T) {
	inner := "-- note\nSELECT 1;\n/*!50503 SET NAMES utf8mb4 */;\nSELECT 'a;''b', `x;y`;\n"
	var tarBuf bytes.Buffer
	tw := tar.NewWriter(&tarBuf)
	if err := tw.WriteHeader(&tar.Header{Name: "0002_demo.sql", Mode: 0o644, Size: int64(len(inner)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(inner)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	var gzBuf bytes.Buffer
	zw := gzip.NewWriter(&gzBuf)
	if _, err := zw.Write(tarBuf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"0002_demo.sql.gz": &fstest.MapFile{Data: gzBuf.Bytes()}}
	all, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Version != 2 {
		t.Fatalf("unexpected migrations: %#v", all)
	}
	if want := sha256.Sum256([]byte(inner)); all[0].Checksum != want {
		t.Fatalf("checksum = %x, want sha256 of inner sql %x", all[0].Checksum, want)
	}
	wantStatements := []string{"-- note\nSELECT 1", "/*!50503 SET NAMES utf8mb4 */", "SELECT 'a;''b', `x;y`"}
	if len(all[0].SQL) != len(wantStatements) {
		t.Fatalf("statements = %#v", all[0].SQL)
	}
	for index, statement := range wantStatements {
		if all[0].SQL[index] != statement {
			t.Fatalf("statement %d = %q, want %q", index+1, all[0].SQL[index], statement)
		}
	}
}

func TestLoadBareGzipFallsBackToRawSQL(t *testing.T) {
	var gzBuf bytes.Buffer
	zw := gzip.NewWriter(&gzBuf)
	if _, err := zw.Write([]byte("SELECT 9;\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"0003_bare.sql.gz": &fstest.MapFile{Data: gzBuf.Bytes()}}
	all, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || len(all[0].SQL) != 1 || all[0].SQL[0] != "SELECT 9" {
		t.Fatalf("unexpected migrations: %#v", all)
	}
}

func TestSplitPlainStatements(t *testing.T) {
	statements := splitPlainStatements("INSERT INTO t VALUES ('it\\'s;ok', \"d\\\"q;\");\n-- tail comment only\nSELECT 2;\n/* nothing */;")
	if len(statements) != 2 || statements[0] != "INSERT INTO t VALUES ('it\\'s;ok', \"d\\\"q;\")" || statements[1] != "-- tail comment only\nSELECT 2" {
		t.Fatalf("statements = %#v", statements)
	}
	if got := splitPlainStatements("/* ; */ SELECT `x;y`;"); len(got) != 1 || got[0] != "/* ; */ SELECT `x;y`" {
		t.Fatalf("backtick statements = %#v", got)
	}
	if got := splitPlainStatements("# only a comment\n;\n"); len(got) != 0 {
		t.Fatalf("comment-only statements = %#v", got)
	}
}

func TestLoadFromArbitraryFS(t *testing.T) {
	fsys := fstest.MapFS{
		"0002_add_x.sql": &fstest.MapFile{Data: []byte("-- migrate:statement\nSELECT 1\n\n-- migrate:statement\nSELECT 2\n")},
		"0001_init.sql":  &fstest.MapFile{Data: []byte("-- migrate:statement\nSELECT 0\n")},
	}
	all, err := Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d migrations, want 2", len(all))
	}
	if all[0].Version != 1 || all[1].Version != 2 {
		t.Fatalf("versions not sorted: %d, %d", all[0].Version, all[1].Version)
	}
	if len(all[1].SQL) != 2 || all[1].SQL[0] != "SELECT 1" || all[1].SQL[1] != "SELECT 2" {
		t.Fatalf("unexpected statements: %#v", all[1].SQL)
	}

	bad := fstest.MapFS{"oops.sql": &fstest.MapFile{Data: []byte("-- migrate:statement\nSELECT 1\n")}}
	if _, err := Load(bad); err == nil || !strings.Contains(err.Error(), "invalid filename") {
		t.Fatalf("error = %v, want invalid filename", err)
	}
}

func TestRunnerRejectsChangedAppliedMigrationUnderLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	all, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	wrong := sha256.Sum256([]byte("changed"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")).WithArgs("nineyin_schema_migration", 30).
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(1))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS sql_replay_ledger`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT version, checksum FROM sql_replay_ledger`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}).AddRow(all[0].Version, wrong[:]))
	mock.ExpectExec(regexp.QuoteMeta("SELECT RELEASE_LOCK(?)")).WithArgs("nineyin_schema_migration").
		WillReturnResult(sqlmock.NewResult(0, 0))
	err = (Runner{DB: db}).Up(context.Background())
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("error = %v, want ErrChecksumMismatch", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRunnerFailsWhenAdvisoryLockIsUnavailable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")).WithArgs("nineyin_schema_migration", 30).
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(0))
	err = (Runner{DB: db}).Up(context.Background())
	if !errors.Is(err, ErrLockUnavailable) {
		t.Fatalf("error = %v, want ErrLockUnavailable", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpAppliesOnlyPendingMigrations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	applied := sha256.Sum256([]byte("applied-v1"))
	list := []Migration{
		{Version: 1, Name: "0001_a.sql.gz", Checksum: applied, SQL: []string{"SELECT 1"}},
		{Version: 2, Name: "0002_b.sql.gz", Checksum: sha256.Sum256([]byte("v2")), SQL: []string{"SELECT 2"}},
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")).WithArgs("nineyin_schema_migration", 30).
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(1))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS sql_replay_ledger`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT version, checksum FROM sql_replay_ledger`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}).AddRow(uint64(1), applied[:]))
	// 只允许执行 v2 的语句并记账；v1 不得出现任何 exec/REPLACE。
	mock.ExpectExec("SELECT 2").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`REPLACE INTO sql_replay_ledger`).WithArgs(2, "0002_b.sql.gz", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("SELECT RELEASE_LOCK(?)")).WithArgs("nineyin_schema_migration").
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := (Runner{DB: db, Migrations: list}).Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestForceReplaysAppliedMigrations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	applied := sha256.Sum256([]byte("applied-v1"))
	list := []Migration{
		{Version: 1, Name: "0001_a.sql.gz", Checksum: applied, SQL: []string{"SELECT 1"}},
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT GET_LOCK(?, ?)")).WithArgs("nineyin_schema_migration", 30).
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(1))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS sql_replay_ledger`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT version, checksum FROM sql_replay_ledger`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}).AddRow(uint64(1), applied[:]))
	// Force：已应用的 v1 也要重放并重新记账。
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`REPLACE INTO sql_replay_ledger`).WithArgs(1, "0001_a.sql.gz", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("SELECT RELEASE_LOCK(?)")).WithArgs("nineyin_schema_migration").
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := (Runner{DB: db, Migrations: list, Force: true}).Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStatusReportsPendingAndApplied(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM information_schema.tables").
		WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	rows := sqlmock.NewRows([]string{"version", "checksum", "applied"}).
		AddRow(1, make([]byte, 32), 1730000000)
	mock.ExpectQuery("SELECT version, checksum, CAST\\(UNIX_TIMESTAMP").WillReturnRows(rows)
	list := []Migration{
		{Version: 1, Name: "0001_init.sql", Checksum: [32]byte{}},
		{Version: 2, Name: "0002_add_x.sql", Checksum: [32]byte{}},
	}
	states, err := (Runner{DB: db}).Status(context.Background(), list)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 2 {
		t.Fatalf("got %d states, want 2", len(states))
	}
	if !states[0].Applied || !states[0].ChecksumOK {
		t.Fatalf("version 1 should be applied with matching checksum: %+v", states[0])
	}
	if states[1].Applied {
		t.Fatalf("version 2 should be pending: %+v", states[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

var _ *sql.DB
