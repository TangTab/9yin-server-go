package role

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

func newMockRepository(t *testing.T) (*MySQLRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		_ = db.Close()
	})
	return NewMySQLRepository(db, MySQLConfig{}), mock
}

func TestMySQLCreateIsAtomicAndReturnsVersionedSnapshot(t *testing.T) {
	repository, mock := newMockRepository(t)
	draft := NewRole{
		Slot:       0,
		Name:       "邢军",
		Appearance: Appearance{FormatVersion: 1, Values: []string{"book5", "cloth_b0001"}},
		Location:   Location{Scene: Scene{Config: `ini\scene\school07_WuDang`, Resource: "school07"}, Position: Position{}},
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM accounts WHERE account_id = \? FOR UPDATE`).WithArgs(AccountID(7)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(AccountActive))
	mock.ExpectExec(`INSERT INTO roles\(account_id, slot, name\) VALUES \(\?, \?, \?\)`).
		WithArgs(AccountID(7), uint16(0), "邢军").WillReturnResult(sqlmock.NewResult(42, 1))
	mock.ExpectExec(`INSERT INTO role_appearances`).WithArgs(RoleID(42), uint16(1), []byte(`["book5","cloth_b0001"]`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO role_locations`).WithArgs(RoleID(42), draft.Location.Scene.Config, "school07", float32(0), float32(0), float32(0), float32(0)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	created, err := repository.Create(context.Background(), 7, draft)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 42 || created.AccountID != 7 || created.Location.Version != 1 || created.Appearance.Version != 1 {
		t.Fatalf("unexpected snapshot: %#v", created)
	}
	if created.Location.Position != (Position{}) {
		t.Fatalf("legal origin changed: %#v", created.Location.Position)
	}
}

func TestMySQLCreateRollsBackWhenChildInsertFails(t *testing.T) {
	repository, mock := newMockRepository(t)
	draft := NewRole{
		Name:       "甲",
		Appearance: Appearance{FormatVersion: 1},
		Location:   Location{Scene: Scene{Config: "config", Resource: "scene"}},
	}
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT status FROM accounts`).WithArgs(AccountID(1)).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(AccountActive))
	mock.ExpectExec(`INSERT INTO roles`).WillReturnResult(sqlmock.NewResult(8, 1))
	mock.ExpectExec(`INSERT INTO role_appearances`).WillReturnError(errors.New("disk full"))
	mock.ExpectRollback()
	_, err := repository.Create(context.Background(), 1, draft)
	if !errors.Is(err, ErrStorage) {
		t.Fatalf("error = %v, want ErrStorage", err)
	}
}

func TestMySQLCreateMapsNamedUniqueConstraints(t *testing.T) {
	for name, testCase := range map[string]struct {
		message string
		want    error
	}{
		"name": {"Duplicate entry '甲' for key 'roles.uq_roles_name'", ErrRoleNameTaken},
		"slot": {"Duplicate entry '1-0' for key 'roles.uq_roles_account_slot'", ErrRoleSlotTaken},
	} {
		t.Run(name, func(t *testing.T) {
			err := classifyCreateError(&mysql.MySQLError{Number: 1062, Message: testCase.message})
			if !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestMySQLLoadOwnedDoesNotLeakAnotherAccountRole(t *testing.T) {
	repository, mock := newMockRepository(t)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE r.account_id = ? AND r.role_id = ?")).WithArgs(AccountID(1), RoleID(9)).
		WillReturnRows(sqlmock.NewRows([]string{"role_id"}))
	mock.ExpectQuery(`SELECT account_id FROM roles WHERE role_id = \?`).WithArgs(RoleID(9)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id"}).AddRow(2))
	_, err := repository.LoadOwned(context.Background(), 1, 9)
	if !errors.Is(err, ErrNotOwned) {
		t.Fatalf("error = %v, want ErrNotOwned", err)
	}
}

func TestMySQLLoadOwnedRejectsMissingAppearanceOrLocation(t *testing.T) {
	repository, mock := newMockRepository(t)
	columns := []string{"role_id", "account_id", "slot", "name", "delete_requested_at", "role_version",
		"format_version", "appearance", "appearance_version", "scene_config", "scene_resource", "pos_x", "pos_y", "pos_z", "orient", "location_version"}
	mock.ExpectQuery(regexp.QuoteMeta("WHERE r.account_id = ? AND r.role_id = ?")).WithArgs(AccountID(1), RoleID(9)).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(9, 1, 0, "甲", nil, 1, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))
	_, err := repository.LoadOwned(context.Background(), 1, 9)
	if !errors.Is(err, ErrCorruptSnapshot) {
		t.Fatalf("error = %v, want ErrCorruptSnapshot", err)
	}
}

func TestMySQLLocationCASUpdatesWholeSnapshotAndReportsConflict(t *testing.T) {
	repository, mock := newMockRepository(t)
	location := Location{
		Scene:    Scene{Config: "new-config", Resource: "new-scene"},
		Position: Position{X: 1, Y: 2, Z: 3, Orient: 4},
	}
	mock.ExpectExec(`UPDATE role_locations`).WithArgs("new-config", "new-scene", float32(1), float32(2), float32(3), float32(4), RoleID(10), uint64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	next, err := repository.SaveLocation(context.Background(), 10, 3, location)
	if err != nil || next != 4 {
		t.Fatalf("success: next=%d err=%v", next, err)
	}

	mock.ExpectExec(`UPDATE role_locations`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT version FROM role_locations WHERE role_id = \?`).WithArgs(RoleID(10)).
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(4))
	current, err := repository.SaveLocation(context.Background(), 10, 3, location)
	if !errors.Is(err, ErrVersionConflict) || current != 4 {
		t.Fatalf("conflict: current=%d err=%v", current, err)
	}
}

func TestMySQLRequestDeleteEnforcesOwnership(t *testing.T) {
	repository, mock := newMockRepository(t)
	mock.ExpectExec(`UPDATE roles SET delete_requested_at`).WithArgs(AccountID(1), RoleID(5)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT account_id FROM roles WHERE role_id = \?`).WithArgs(RoleID(5)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id"}).AddRow(2))
	if err := repository.RequestDelete(context.Background(), 1, 5); !errors.Is(err, ErrNotOwned) {
		t.Fatalf("error = %v, want ErrNotOwned", err)
	}
}

func TestValidateAccountKeyRejectsControlsButPreservesCaseAndUnicode(t *testing.T) {
	for _, invalid := range []AccountKey{"", "a\x00b", "a\nb"} {
		if err := ValidateAccountKey(invalid); !errors.Is(err, ErrInvalidAccountKey) {
			t.Fatalf("key %q error = %v", invalid, err)
		}
	}
	for _, valid := range []AccountKey{"Alice", "alice", "玩家一"} {
		if err := ValidateAccountKey(valid); err != nil {
			t.Fatalf("key %q: %v", valid, err)
		}
	}
}

func TestProductionDSNAppliesRequiredSafetyParameters(t *testing.T) {
	dsn, err := ProductionDSN("user:pass@tcp(localhost:3306)/nineyin")
	if err != nil {
		t.Fatal(err)
	}
	for _, parameter := range []string{"charset=utf8mb4", "parseTime=true", "timeout=5s", "readTimeout=5s", "writeTimeout=5s"} {
		if !strings.Contains(dsn, parameter) {
			t.Fatalf("normalized DSN %q missing %q", dsn, parameter)
		}
	}
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil || parsed.Loc.String() != "UTC" {
		t.Fatalf("normalized location = %v, err=%v", parsed.Loc, err)
	}
	if _, err := ProductionDSN("user:pass@tcp(localhost:3306)/"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("missing database error = %v", err)
	}
}

var _ *sql.DB // retain database/sql compile coverage for injected DB contract
