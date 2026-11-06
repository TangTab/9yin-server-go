package migrations

import (
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

func TestEmbeddedMigrationContainsNormalizedSchemaAndAtomicCutover(t *testing.T) {
	all, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].Version != 1 || len(all[0].SQL) < 10 {
		t.Fatalf("unexpected embedded migrations: %#v", all)
	}
	joined := strings.Join(all[0].SQL, "\n")
	for _, required := range []string{
		"CREATE TABLE accounts_m2", "CREATE TABLE roles_m2", "CREATE TABLE role_appearances_m2",
		"CREATE TABLE role_locations_m2", "uq_roles_account_slot", "RENAME TABLE accounts TO accounts_legacy_v1",
		"JSON_VALID(appearance)", "nineyin_m2_verify_copy",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("migration missing %q", required)
		}
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
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT version, checksum FROM schema_migrations`).
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

var _ *sql.DB

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
