package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/local/9yin-go-server/internal/role"
)

const facultyTestRoleID = role.RoleID(42)

func newMockFacultyDB(t *testing.T) (*facultyDBStore, sqlmock.Sqlmock) {
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
	return newFacultyDBStore(db), mock
}

func TestFacultyDBLoadDecodesSnapshot(t *testing.T) {
	store, mock := newMockFacultyDB(t)
	snapshot := facultyProgressSnapshot{
		CurNeiGong:     "buf_ng_yihua_01",
		Faculty:        12,
		FacultyState:   1,
		FacultyStyle:   2,
		FacultyName:    "移花宫",
		FillSpeed:      7,
		LastAdvanceUTC: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC),
		Books:          []facultyBookSnapshot{{ConfigID: "buf_ng_yihua_01", Level: 3, Total: 100, Fill: 42}},
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT snapshot FROM role_faculty WHERE role_id = \?`).WithArgs(facultyTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow(raw))
	loaded, exists := store.Load(facultyTestRoleID)
	if !exists {
		t.Fatal("expected snapshot to exist")
	}
	if loaded.CurNeiGong != snapshot.CurNeiGong || loaded.Faculty != snapshot.Faculty ||
		loaded.FacultyName != snapshot.FacultyName || len(loaded.Books) != 1 || loaded.Books[0].Fill != 42 {
		t.Fatalf("loaded=%+v", loaded)
	}
	if !loaded.LastAdvanceUTC.Equal(snapshot.LastAdvanceUTC) {
		t.Fatalf("last_advance_utc=%v want %v", loaded.LastAdvanceUTC, snapshot.LastAdvanceUTC)
	}
}

func TestFacultyDBLoadMissingAndCorruptRows(t *testing.T) {
	store, mock := newMockFacultyDB(t)
	mock.ExpectQuery(`SELECT snapshot FROM role_faculty WHERE role_id = \?`).WithArgs(facultyTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}))
	if _, exists := store.Load(facultyTestRoleID); exists {
		t.Fatal("missing row must read as no archive")
	}

	store2, mock2 := newMockFacultyDB(t)
	mock2.ExpectQuery(`SELECT snapshot FROM role_faculty WHERE role_id = \?`).WithArgs(facultyTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte("not json")))
	if _, exists := store2.Load(facultyTestRoleID); exists {
		t.Fatal("corrupt snapshot must read as no archive")
	}
}

func TestFacultyDBSaveUpsertsJavaParitySQL(t *testing.T) {
	store, mock := newMockFacultyDB(t)
	snapshot := facultyProgressSnapshot{CurNeiGong: "buf_ng_yihua_01", Faculty: 2, FacultyName: "移花宫"}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(`INSERT INTO role_faculty \(role_id, snapshot, version, updated_at\) VALUES \(\?, \?, 1, UTC_TIMESTAMP\(6\)\)`+
		` ON DUPLICATE KEY UPDATE snapshot = VALUES\(snapshot\), version = version \+ 1, updated_at = UTC_TIMESTAMP\(6\)`).
		WithArgs(facultyTestRoleID, raw).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.Save(facultyTestRoleID, snapshot); err != nil {
		t.Fatal(err)
	}
}

// 超集字段 passthrough：Java/新版服快照的额外字段解码后原样带回，
// Go 子集写回不抹数据。
func TestFacultySnapshotPassthroughSupersetFields(t *testing.T) {
	source := `{"cur_neigong":"buf_ng_yihua_01","faculty":5,"faculty_state":1,"faculty_style":0,` +
		`"faculty_name":"移花宫","fill_speed":3,"last_advance_utc":"2026-11-06T12:00:00Z",` +
		`"books":[{"config_id":"ng_wd_001","level":36,"power":4670,"total":274400,"fill":0,"max_power":4670,"max_level":36}],` +
		`"vitals":{"ene":100,"max_ene":200},"live_groove_1":7,"fill_speed_multiplier":1.5,"base_str":12}`
	var loaded facultyProgressSnapshot
	if err := json.Unmarshal([]byte(source), &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Faculty != 5 || loaded.FacultyName != "移花宫" || len(loaded.Books) != 1 || loaded.Books[0].Level != 36 {
		t.Fatalf("known fields=%+v", loaded)
	}
	rendered, err := json.Marshal(cloneFacultyProgress(loaded))
	if err != nil {
		t.Fatal(err)
	}
	var round map[string]json.RawMessage
	if err := json.Unmarshal(rendered, &round); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"vitals", "live_groove_1", "fill_speed_multiplier", "base_str"} {
		if _, exists := round[key]; !exists {
			t.Fatalf("passthrough field %q lost in %s", key, rendered)
		}
	}
	if string(round["vitals"]) != `{"ene":100,"max_ene":200}` {
		t.Fatalf("vitals passthrough=%s", round["vitals"])
	}
	var books []map[string]json.RawMessage
	if err := json.Unmarshal(round["books"], &books); err != nil {
		t.Fatal(err)
	}
	if len(books) != 1 || string(books[0]["max_level"]) != "36" {
		t.Fatalf("book max_level passthrough lost: %s", round["books"])
	}
	// 往返后再解码仍是同一快照。
	var again facultyProgressSnapshot
	if err := json.Unmarshal(rendered, &again); err != nil {
		t.Fatal(err)
	}
	if again.CurNeiGong != loaded.CurNeiGong || again.Faculty != loaded.Faculty || again.Books[0].Power != 4670 {
		t.Fatalf("roundtrip=%+v", again)
	}
}

// 纯 Go 子集快照（无 extras）输出与旧文件存档格式兼容，
// Java FacultyStore 宽松解析可读。
func TestFacultySnapshotSubsetMarshalStable(t *testing.T) {
	snapshot := facultyProgressSnapshot{CurNeiGong: "buf_ng_x", Faculty: 1}
	rendered, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rendered, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != len(facultySnapshotJSONKeys) {
		t.Fatalf("field count=%d want %d: %s", len(fields), len(facultySnapshotJSONKeys), rendered)
	}
}
