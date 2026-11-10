package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/local/9yin-go-server/internal/role"
)

const taskTestRoleID = role.RoleID(43)

func newMockTaskDB(t *testing.T) (*taskDBStore, sqlmock.Sqlmock) {
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
	return newTaskDBStore(db), mock
}

func TestTaskDBLoadDecodesSnapshot(t *testing.T) {
	store, mock := newMockTaskDB(t)
	snapshot := taskProgressSnapshot{
		Accepted: []taskAcceptedRow{
			{TaskID: 791, Status: taskStatusActive, AcceptedAt: 1790000000},
			{TaskID: 802, Status: taskStatusSubmitReady, AcceptedAt: 1790000100},
		},
		Completed: []int{42, 100},
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT snapshot FROM role_tasks WHERE role_id = \?`).WithArgs(taskTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow(raw))
	loaded, exists := store.Load(taskTestRoleID)
	if !exists {
		t.Fatal("expected snapshot to exist")
	}
	if len(loaded.Accepted) != 2 || loaded.Accepted[1].TaskID != 802 ||
		loaded.Accepted[1].Status != taskStatusSubmitReady || loaded.Accepted[1].AcceptedAt != 1790000100 {
		t.Fatalf("accepted=%+v", loaded.Accepted)
	}
	if len(loaded.Completed) != 2 || loaded.Completed[0] != 42 {
		t.Fatalf("completed=%v", loaded.Completed)
	}
}

func TestTaskDBLoadMissingAndCorruptRows(t *testing.T) {
	store, mock := newMockTaskDB(t)
	mock.ExpectQuery(`SELECT snapshot FROM role_tasks WHERE role_id = \?`).WithArgs(taskTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}))
	if _, exists := store.Load(taskTestRoleID); exists {
		t.Fatal("missing row must read as no archive")
	}

	store2, mock2 := newMockTaskDB(t)
	mock2.ExpectQuery(`SELECT snapshot FROM role_tasks WHERE role_id = \?`).WithArgs(taskTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot"}).AddRow([]byte("not json")))
	if _, exists := store2.Load(taskTestRoleID); exists {
		t.Fatal("corrupt snapshot must read as no archive")
	}
}

// Save 的 upsert SQL 与 faculty/jingMai 家族逐字同形（migrations/0009
// 表结构），字段名按 0009 注释定版。
func TestTaskDBSaveUpsertsSQL(t *testing.T) {
	store, mock := newMockTaskDB(t)
	snapshot := taskProgressSnapshot{
		Accepted:  []taskAcceptedRow{{TaskID: 791, Status: taskStatusActive, AcceptedAt: 1790000000}},
		Completed: []int{42},
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(`INSERT INTO role_tasks \(role_id, snapshot, version, updated_at\) VALUES \(\?, \?, 1, UTC_TIMESTAMP\(6\)\)`+
		` ON DUPLICATE KEY UPDATE snapshot = VALUES\(snapshot\), version = version \+ 1, updated_at = UTC_TIMESTAMP\(6\)`).
		WithArgs(taskTestRoleID, raw).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.Save(taskTestRoleID, snapshot); err != nil {
		t.Fatal(err)
	}
}

// 快照 JSON 形态与 migrations/0009 注释逐字对齐（字段名/键序由该
// 注释定版，前端与 Java 端互通依赖）。
func TestTaskSnapshotJSONShape(t *testing.T) {
	raw, err := json.Marshal(taskProgressSnapshot{
		Accepted:  []taskAcceptedRow{{TaskID: 791, Status: 1, AcceptedAt: 1790000000}},
		Completed: []int{42},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"accepted":[{"task_id":791,"status":1,"accepted_at":1790000000}],"completed":[42]}`
	if string(raw) != want {
		t.Fatalf("snapshot json=%s want %s", raw, want)
	}
}

// 接取 → 可交付 → 交付全链 + 重复接取/已完成不可再接。
func TestPlayerTaskLifecycle(t *testing.T) {
	player := &playerActor{}
	now := time.Unix(1790000000, 0)
	if !player.acceptTask(791, now) {
		t.Fatal("accept 791 should apply")
	}
	if player.acceptTask(791, now.Add(time.Minute)) {
		t.Fatal("duplicate accept must be rejected")
	}
	active, ready, done := player.taskCounts()
	if active != 1 || ready != 0 || done != 0 {
		t.Fatalf("counts after accept=%d/%d/%d", active, ready, done)
	}
	if !player.markTaskSubmitReady(791) {
		t.Fatal("mark ready should apply")
	}
	if player.markTaskSubmitReady(791) {
		t.Fatal("mark ready twice must be rejected")
	}
	active, ready, done = player.taskCounts()
	if active != 0 || ready != 1 || done != 0 {
		t.Fatalf("counts after ready=%d/%d/%d", active, ready, done)
	}
	if !player.completeTask(791) {
		t.Fatal("complete should apply")
	}
	if player.completeTask(791) {
		t.Fatal("complete twice must be rejected")
	}
	if player.markTaskSubmitReady(791) || player.removeTask(791) {
		t.Fatal("completed task must not be mutable via ready/remove")
	}
	if player.acceptTask(791, now) {
		t.Fatal("completed task must not be re-acceptable")
	}
	active, ready, done = player.taskCounts()
	if active != 0 || ready != 0 || done != 1 {
		t.Fatalf("counts after complete=%d/%d/%d", active, ready, done)
	}
}

// removeTask 只清 accepted，不动 completed。
func TestPlayerTaskRemove(t *testing.T) {
	player := &playerActor{}
	now := time.Unix(1790000000, 0)
	player.acceptTask(700, now)
	player.acceptTask(800, now)
	if player.removeTask(999) {
		t.Fatal("remove unknown task must be rejected")
	}
	if !player.removeTask(700) {
		t.Fatal("remove 700 should apply")
	}
	if active, _, done := player.taskCounts(); active != 1 || done != 0 {
		t.Fatalf("counts after remove=%d/%d", active, done)
	}
	if player.removeTask(700) {
		t.Fatal("remove twice must be rejected")
	}
}

// 快照按任务 ID 升序（接取乱序时仍确定性输出）；恢复往返保真
// （status 原值保留、非法 ID 丢弃、completed 去重）。
func TestPlayerTaskSnapshotOrderAndRestoreRoundtrip(t *testing.T) {
	player := &playerActor{}
	base := time.Unix(1790000000, 0)
	player.acceptTask(802, base)
	player.acceptTask(791, base.Add(time.Second))
	player.acceptTask(900, base)
	snapshot := player.taskSnapshotOf()
	if len(snapshot.Accepted) != 3 ||
		snapshot.Accepted[0].TaskID != 791 || snapshot.Accepted[1].TaskID != 802 || snapshot.Accepted[2].TaskID != 900 {
		t.Fatalf("accepted not ascending by task id: %+v", snapshot.Accepted)
	}

	restored := &playerActor{}
	restored.restoreTaskState(taskProgressSnapshot{
		Accepted: []taskAcceptedRow{
			{TaskID: 0, Status: 1, AcceptedAt: 1},    // 非法 ID 丢弃
			{TaskID: -7, Status: 1, AcceptedAt: 1},   // 同上
			{TaskID: 802, Status: 9, AcceptedAt: 42}, // 未知 status 保真保留
		},
		Completed: []int{42, 42, 0}, // 去重 + 非法丢弃
	})
	snapshot = restored.taskSnapshotOf()
	if len(snapshot.Accepted) != 1 || snapshot.Accepted[0].TaskID != 802 || snapshot.Accepted[0].Status != 9 || snapshot.Accepted[0].AcceptedAt != 42 {
		t.Fatalf("restored accepted=%+v", snapshot.Accepted)
	}
	if len(snapshot.Completed) != 1 || snapshot.Completed[0] != 42 {
		t.Fatalf("restored completed=%v", snapshot.Completed)
	}
	if active, _, done := restored.taskCounts(); active != 1 || done != 1 {
		t.Fatalf("restored counts=%d/%d", active, done)
	}
	// 恢复后的行为语义与直接迁移一致：802 已接不可重复接，42 已完成不可接。
	if restored.acceptTask(802, base) {
		t.Fatal("restored accepted task must not be re-acceptable")
	}
	if restored.acceptTask(42, base) {
		t.Fatal("restored completed task must not be acceptable")
	}
}

// earliestActiveTaskID：最早接取优先，同刻取小 ID（GM 留空默认对象）。
func TestPlayerTaskEarliestActive(t *testing.T) {
	player := &playerActor{}
	if id := player.earliestActiveTaskID(); id != 0 {
		t.Fatalf("empty domain earliest=%d want 0", id)
	}
	base := time.Unix(1790000000, 0)
	player.acceptTask(900, base)
	player.acceptTask(700, base)                  // 同刻，ID 更小
	player.acceptTask(600, base.Add(time.Minute)) // 更晚
	if id := player.earliestActiveTaskID(); id != 700 {
		t.Fatalf("earliest=%d want 700", id)
	}
	player.completeTask(700)
	if id := player.earliestActiveTaskID(); id != 900 {
		t.Fatalf("earliest after complete=%d want 900", id)
	}
}
