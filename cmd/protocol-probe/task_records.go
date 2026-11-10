package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// 任务域（数据侧第一块）：角色任务状态快照持久化到 role_tasks
// （migrations/0009，role_id PK / snapshot JSON / version / updated_at，
// 与 role_meridians/role_faculty 同族）。快照形态由 0009 注释定死：
// {"accepted":[{"task_id":int,"status":int,"accepted_at":int64}],
// "completed":[int,...]}。0009 注释里 "TaskStore.attachDatabase 运行期
// 兜底建表" 是 Java 参考服行为——本服不做运行期 DDL（仓库铁律），缺表/
// 缺库一律降级会话态只记日志，建表权威在 migrations/ 重放。GM
// authored_task_* 四动作是当前唯一写入口（协议帧/目标触发器后续轮次
// 接入后复用同一内存域）。
const (
	taskDBQueryTimeout = 5 * time.Second

	// 任务状态（snapshot.accepted[].status）：1=已接进行中，2=可交付
	//（GM authored_task_submit_ready 把任务推进到可交付；交付后任务
	// 移出 accepted、进 completed 集合）。
	taskStatusActive      = 1
	taskStatusSubmitReady = 2
)

// taskProgressSnapshot 是 role_tasks.snapshot 的持久化形态；Accepted 与
// Completed 均按任务 ID 升序（确定性输出，存档对比稳定）。
type taskProgressSnapshot struct {
	Accepted  []taskAcceptedRow `json:"accepted"`
	Completed []int             `json:"completed"`
}

// taskAcceptedRow 单条进行中任务；字段名与 migrations/0009 注释逐字对齐。
type taskAcceptedRow struct {
	TaskID     int   `json:"task_id"`
	Status     int   `json:"status"`
	AcceptedAt int64 `json:"accepted_at"`
}

type taskDBStore struct {
	db *sql.DB
}

func newTaskDBStore(db *sql.DB) *taskDBStore {
	return &taskDBStore{db: db}
}

// Load 读取角色任务状态；无行、DB 错误或坏快照都按“无存档”处理并
// 记日志（缺表降级，登录链走空态，不阻塞登录）。
func (store *taskDBStore) Load(roleID role.RoleID) (taskProgressSnapshot, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return taskProgressSnapshot{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), taskDBQueryTimeout)
	defer cancel()
	var raw []byte
	err := store.db.QueryRowContext(ctx,
		`SELECT snapshot FROM role_tasks WHERE role_id = ?`, roleID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return taskProgressSnapshot{}, false
	}
	if err != nil {
		log.Printf("task store: load role=%d: %v", roleID, err)
		return taskProgressSnapshot{}, false
	}
	var snapshot taskProgressSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		log.Printf("task store: decode role=%d snapshot: %v", roleID, err)
		return taskProgressSnapshot{}, false
	}
	return snapshot, true
}

// Save upsert 快照（version 自增、updated_at 由库维护；faculty/jingMai
// 同款 SQL）。
func (store *taskDBStore) Save(roleID role.RoleID, value taskProgressSnapshot) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("task store: no database")
	}
	if roleID == 0 {
		return errors.New("task store: zero role id")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode task snapshot: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), taskDBQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx, `INSERT INTO role_tasks (role_id, snapshot, version, updated_at)
VALUES (?, ?, 1, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE snapshot = VALUES(snapshot), version = version + 1, updated_at = UTC_TIMESTAMP(6)`,
		roleID, raw); err != nil {
		return fmt.Errorf("persist task role=%d: %w", roleID, err)
	}
	return nil
}

// playerTaskDomain 是 playerActor 的任务内存域（p.mu 保护）：accepted
// 键为任务 ID，completed 为已完成集合。迁移方法返回 bool 表示是否
// 发生变更，GM 回执据此区分「成功/已在目标状态」。
type playerTaskDomain struct {
	accepted  map[int]playerTaskEntry
	completed map[int]struct{}
}

type playerTaskEntry struct {
	status     int
	acceptedAt int64
}

// acceptTask 接取任务（status=进行中）；非正任务 ID、已接或已完成
// 返回 false。
func (p *playerActor) acceptTask(taskID int, now time.Time) bool {
	if taskID <= 0 {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tasks.accepted == nil {
		p.tasks.accepted = make(map[int]playerTaskEntry)
	}
	if p.tasks.completed == nil {
		p.tasks.completed = make(map[int]struct{})
	}
	if _, ok := p.tasks.accepted[taskID]; ok {
		return false
	}
	if _, ok := p.tasks.completed[taskID]; ok {
		return false
	}
	p.tasks.accepted[taskID] = playerTaskEntry{status: taskStatusActive, acceptedAt: now.Unix()}
	return true
}

// markTaskSubmitReady 把已接任务推进到可交付；未接或已在可交付态
// 返回 false。
func (p *playerActor) markTaskSubmitReady(taskID int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.tasks.accepted[taskID]
	if !ok || entry.status == taskStatusSubmitReady {
		return false
	}
	entry.status = taskStatusSubmitReady
	p.tasks.accepted[taskID] = entry
	return true
}

// removeTask 放弃/清理一条已接任务（不动 completed）；未接返回 false。
func (p *playerActor) removeTask(taskID int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.tasks.accepted[taskID]; !ok {
		return false
	}
	delete(p.tasks.accepted, taskID)
	return true
}

// completeTask 交付任务：accepted 移除、进 completed 集合（GM
// authored_task_reward 语义——奖励发放属协议侧，此处只落状态）；未接
// 返回 false。
func (p *playerActor) completeTask(taskID int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.tasks.accepted[taskID]; !ok {
		return false
	}
	delete(p.tasks.accepted, taskID)
	if p.tasks.completed == nil {
		p.tasks.completed = make(map[int]struct{})
	}
	p.tasks.completed[taskID] = struct{}{}
	return true
}

// earliestActiveTaskID 返回最早接取（acceptedAt 最小，同刻取小 ID）
// 的进行中任务；无进行中任务返回 0。GM 面板任务 ID 留空时的默认
// 作用对象（对应前端「留空默认作用于最早的活跃任务」文案）。
func (p *playerActor) earliestActiveTaskID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	best := 0
	var bestAt int64
	for id, entry := range p.tasks.accepted {
		if best == 0 || entry.acceptedAt < bestAt || (entry.acceptedAt == bestAt && id < best) {
			best, bestAt = id, entry.acceptedAt
		}
	}
	return best
}

// taskCounts 回执/日志用计数：进行中、其中可交付、已完成。
func (p *playerActor) taskCounts() (active, ready, completed int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, entry := range p.tasks.accepted {
		if entry.status == taskStatusSubmitReady {
			ready++
		} else {
			active++
		}
	}
	return active, ready, len(p.tasks.completed)
}

// taskSnapshotOf 拷贝任务域快照（Accepted/Completed 按任务 ID 升序）。
func (p *playerActor) taskSnapshotOf() taskProgressSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	snapshot := taskProgressSnapshot{
		Accepted:  make([]taskAcceptedRow, 0, len(p.tasks.accepted)),
		Completed: make([]int, 0, len(p.tasks.completed)),
	}
	for id, entry := range p.tasks.accepted {
		snapshot.Accepted = append(snapshot.Accepted, taskAcceptedRow{
			TaskID: id, Status: entry.status, AcceptedAt: entry.acceptedAt,
		})
	}
	sort.Slice(snapshot.Accepted, func(i, j int) bool { return snapshot.Accepted[i].TaskID < snapshot.Accepted[j].TaskID })
	for id := range p.tasks.completed {
		snapshot.Completed = append(snapshot.Completed, id)
	}
	sort.Ints(snapshot.Completed)
	return snapshot
}

// restoreTaskState 登录装载：重建两集合（task_id ≤ 0 丢弃，status 保真
// 保留——未来状态枚举扩容不丢数据；completed 去重）。
func (p *playerActor) restoreTaskState(snapshot taskProgressSnapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tasks.accepted = make(map[int]playerTaskEntry, len(snapshot.Accepted))
	for _, row := range snapshot.Accepted {
		if row.TaskID <= 0 {
			continue
		}
		p.tasks.accepted[row.TaskID] = playerTaskEntry{status: row.Status, acceptedAt: row.AcceptedAt}
	}
	p.tasks.completed = make(map[int]struct{}, len(snapshot.Completed))
	for _, id := range snapshot.Completed {
		if id > 0 {
			p.tasks.completed[id] = struct{}{}
		}
	}
}
