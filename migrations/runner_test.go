package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"

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
