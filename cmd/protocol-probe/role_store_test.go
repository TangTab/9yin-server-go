package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/auth"

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
	firstAccount, firstRole, err := store.login(ctx, "acct:v1:first", nil)
	if err != nil || firstRole != nil {
		t.Fatalf("first login role=%v error=%v", firstRole, err)
	}
	secondAccount, secondRole, err := store.login(ctx, "acct:v1:second", nil)
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

	_, reloaded, err := store.login(ctx, firstAccount.Key, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded == nil || reloaded.Location.Position != location.Position {
		t.Fatalf("reloaded role = %#v", reloaded)
	}
	_, other, err := store.login(ctx, secondAccount.Key, nil)
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

// TestRoleStoreRegisterAndPasswordVerification 注册→验密全链：
// 存量账号（无校验子）免密放行；注册后错误密码拒绝、正确密码（客户端
// 密码槽密文形态）放行；重复注册＝改密（fork 实证 upsert 语义）。
func TestRoleStoreRegisterAndPasswordVerification(t *testing.T) {
	ctx := context.Background()
	store := testRoleStore(t)

	const accountName = "regtest01"
	key, err := auth.AccountKeyFor(accountName)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "acct:v1:") || len(key) != len("acct:v1:")+64 {
		t.Fatalf("unexpected key shape: %q", key)
	}

	// 未注册：默认模式自动建号，免口令放行。
	account, _, err := store.login(ctx, role.AccountKey(key), nil)
	if err != nil {
		t.Fatalf("legacy auto-provision login: %v", err)
	}
	if len(account.PasswordHash) != 0 {
		t.Fatalf("unregistered account has verifier: %x", account.PasswordHash)
	}

	// 注册：校验子落库。
	verifier := auth.ComputePasswordVerifier(accountName, "first-pass")
	if _, err := store.accounts.RegisterAccount(ctx, account.Key, verifier); err != nil {
		t.Fatal(err)
	}
	// 错误密码（另一口令的密文）拒绝。
	if _, _, err := store.login(ctx, account.Key, auth.ComputePasswordVerifier(accountName, "wrong-pass")); !errors.Is(err, errPasswordMismatch) {
		t.Fatalf("wrong password: want errPasswordMismatch, got %v", err)
	}
	// 正确密码放行。
	if _, _, err := store.login(ctx, account.Key, verifier); err != nil {
		t.Fatalf("correct password rejected: %v", err)
	}

	// 重复注册＝改密。
	rekeyed := auth.ComputePasswordVerifier(accountName, "second-pass")
	if _, err := store.accounts.RegisterAccount(ctx, account.Key, rekeyed); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.login(ctx, account.Key, verifier); !errors.Is(err, errPasswordMismatch) {
		t.Fatalf("old password after re-register: want mismatch, got %v", err)
	}
	if _, _, err := store.login(ctx, account.Key, rekeyed); err != nil {
		t.Fatalf("new password rejected: %v", err)
	}
}

// TestRoleStoreRequireRegistration 要求注册模式下未知账号拒绝建号。
func TestRoleStoreRequireRegistration(t *testing.T) {
	t.Setenv("NINEYIN_REQUIRE_REGISTRATION", "1")
	ctx := context.Background()
	store := testRoleStore(t)
	key, err := auth.AccountKeyFor("gated01")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.login(ctx, role.AccountKey(key), nil); !errors.Is(err, errAccountNotRegistered) {
		t.Fatalf("want errAccountNotRegistered, got %v", err)
	}
	// 注册后放行。
	if _, err := store.accounts.RegisterAccount(ctx, role.AccountKey(key), auth.ComputePasswordVerifier("gated01", "pw")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.login(ctx, role.AccountKey(key), auth.ComputePasswordVerifier("gated01", "pw")); err != nil {
		t.Fatalf("registered login rejected: %v", err)
	}
}
