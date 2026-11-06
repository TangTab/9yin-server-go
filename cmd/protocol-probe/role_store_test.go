package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/local/9yin-go-server/internal/role"
)

func testRoleStore(t *testing.T) *roleStore {
	t.Helper()
	repository, err := role.OpenJSONRepository(filepath.Join(t.TempDir(), "roles.json"))
	if err != nil {
		t.Fatal(err)
	}
	return &roleStore{accounts: repository, roles: repository, close: func() error { return nil }}
}

func TestRoleStoreIsolatesAccountsAndPersistsLocationVersion(t *testing.T) {
	ctx := context.Background()
	store := testRoleStore(t)
	firstAccount, firstRole, err := store.login(ctx, "acct:v1:first")
	if err != nil || firstRole != nil {
		t.Fatalf("first login role=%v error=%v", firstRole, err)
	}
	secondAccount, secondRole, err := store.login(ctx, "acct:v1:second")
	if err != nil || secondRole != nil {
		t.Fatalf("second login role=%v error=%v", secondRole, err)
	}
	firstRole, err = store.create(ctx, firstAccount, "甲", []string{"book5"})
	if err != nil {
		t.Fatal(err)
	}
	if got := firstRole.Appearance.Values[8]; got != defaultRoleFaction {
		t.Fatalf("default faction=%q want=%q", got, defaultRoleFaction)
	}
	if got, want := firstRole.Location, defaultRoleLocation(); got.Scene != want.Scene || got.Position != want.Position {
		t.Fatalf("default location=%+v want=%+v", got, want)
	}
	secondRole, err = store.create(ctx, secondAccount, "乙", []string{"book4"})
	if err != nil {
		t.Fatal(err)
	}
	if firstRole.ID == secondRole.ID || firstRole.AccountID == secondRole.AccountID {
		t.Fatalf("identities overlap: first=%#v second=%#v", firstRole, secondRole)
	}

	location := firstRole.Location
	previousVersion := location.Version
	location.Position = role.Position{X: 0, Y: 0, Z: 0, Orient: 0}
	if err := store.saveLocation(ctx, firstRole, location); err != nil {
		t.Fatal(err)
	}
	if firstRole.Location.Version != previousVersion+1 || firstRole.Location.Position != location.Position {
		t.Fatalf("saved location = %#v", firstRole.Location)
	}

	_, reloaded, err := store.login(ctx, firstAccount.Key)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded == nil || reloaded.Location.Position != location.Position {
		t.Fatalf("reloaded role = %#v", reloaded)
	}
	_, other, err := store.login(ctx, secondAccount.Key)
	if err != nil {
		t.Fatal(err)
	}
	if other == nil || other.Name != "乙" {
		t.Fatalf("second account role = %#v", other)
	}
}

func TestOpenRoleStoreRequiresDSN(t *testing.T) {
	t.Setenv("NINEYIN_MYSQL_DSN", "")
	_, err := openRoleStore()
	if err == nil || !strings.Contains(err.Error(), "NINEYIN_MYSQL_DSN") {
		t.Fatalf("expected DSN requirement error, got %v", err)
	}
}

func TestVerifyRequiredTablesFailsFastOnMissing(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	// 仅返回 accounts/roles；缺 role_appearances/role_locations/role_faculty。
	rows := sqlmock.NewRows([]string{"table_name"}).
		AddRow("accounts").
		AddRow("roles")
	mock.ExpectQuery(`SELECT table_name FROM information_schema\.tables`).WillReturnRows(rows)
	err = verifyRequiredTables(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "role_faculty") || !strings.Contains(err.Error(), "upgrade-db.bat") {
		t.Fatalf("expected missing-table error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
