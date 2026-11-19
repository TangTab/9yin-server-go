package fxcore

import (
	"fmt"
	"sort"
	"sync"
)

// Event 复刻官方事件对象（nx_gen_event 产物）：目标实体 + 携带参数。
// 异步过程经 AddWaitEvent 挂起等待，GenEvent 唤醒全部等待者（泵内派发，
// 单线程——官方协程语义，Go 侧不做 goroutine，保泵序确定性）。
type Event struct {
	Object EntityID  // 目标实体（官方 nx_gen_event(entity)）
	Name   string    // 事件名（可选细分）
	Args   []any     // 携带参数
}

// AsyncProc 复刻官方 CAsyncProc：可挂起的异步过程。
// Body 每次 Continue 被调用：返回 true=继续运行（下一遍再跑），
// false=挂起等待事件（waitEvents 非空）或结束（Done）。
type AsyncProc struct {
	ID   int
	Name string // "script func" 定位面（官方 CreateAsyncProc(script, func)）

	Body     func(proc *AsyncProc, ev *Event) bool
	waitObjs []EntityID // 官方 AddWaitEvent 注册的等待目标
	paused   bool
	done     bool
	pausing  bool // 官方 Pause/PauseAll 与 strict_pause 语义
}

// AddWaitEvent 官方等待注册（等待目标实体上的事件）。
func (p *AsyncProc) AddWaitEvent(id EntityID) { p.waitObjs = append(p.waitObjs, id) }

// FindWaitEvent 官方查询：是否在等待该目标。
func (p *AsyncProc) FindWaitEvent(id EntityID) bool {
	for _, w := range p.waitObjs {
		if w == id {
			return true
		}
	}
	return false
}

// GetWaitEventObject 官方取首个等待目标（0=无）。
func (p *AsyncProc) GetWaitEventObject() EntityID {
	if len(p.waitObjs) == 0 {
		return 0
	}
	return p.waitObjs[0]
}

// Pause 官方暂停（strictPause=true 时严格挂起，事件不唤醒）。
func (p *AsyncProc) Pause(strict bool) { p.paused = true; p.pausing = strict }

// Done 过程终止位。
func (p *AsyncProc) Done() bool { return p.done }

// AsyncManager 复刻官方 CAsyncManager：异步过程表 + 事件派发。
// 泵每拍两遍：Pass1 运行/唤醒，Pass2 清理终止过程（官方 DeleteNode/EraseNode）。
type AsyncManager struct {
	mu     sync.Mutex
	procs  map[int]*AsyncProc
	events []*Event // 待派发事件队列（GenEvent 入队，Pass1 消费）
	nextID int
}

// NewAsyncManager 创建空管理器。
func NewAsyncManager() *AsyncManager {
	return &AsyncManager{procs: make(map[int]*AsyncProc)}
}

// Add 复刻 CAsyncManager::Add：注册异步过程并返回句柄。
func (m *AsyncManager) Add(name string, body func(proc *AsyncProc, ev *Event) bool) *AsyncProc {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	p := &AsyncProc{ID: m.nextID, Name: name, Body: body}
	m.procs[p.ID] = p
	return p
}

// GenEvent 复刻官方 GenEvent：事件入队（Pass1 派发给等待者）。
func (m *AsyncManager) GenEvent(ev *Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, ev)
}

// Kill 复刻官方 Kill：标记终止（Pass2 清除）。
func (m *AsyncManager) Kill(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p := m.procs[id]; p != nil {
		p.done = true
	}
}

// DeleteAsyncProc 官方立即移除。
func (m *AsyncManager) DeleteAsyncProc(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.procs, id)
}

// Find 句柄查找。
func (m *AsyncManager) Find(id int) *AsyncProc {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.procs[id]
}

// Count 存活过程数。
func (m *AsyncManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.procs)
}

// ExecutePass1 复刻泵第一遍（0x100fcef0 位）：先派发事件唤醒等待者，
// 再运行全部未挂起过程。事件派发序=入队序（FIFO）。
func (m *AsyncManager) ExecutePass1() {
	// 事件派发：等待目标匹配的过程收事件并清等待位。
	m.mu.Lock()
	events := m.events
	m.events = nil
	procs := make([]*AsyncProc, 0, len(m.procs))
	for _, p := range m.procs {
		procs = append(procs, p)
	}
	m.mu.Unlock()

	for _, ev := range events {
		for _, p := range procs {
			if p.done || p.paused && p.pausing {
				continue
			}
			if p.FindWaitEvent(ev.Object) {
				p.waitObjs = nil
				if p.Body != nil {
					p.Body(p, ev)
				}
			}
		}
	}
	// 运行面：无等待且未暂停的过程逐个 Continue（nil 事件=常规步进）。
	for _, p := range procs {
		if p.done || p.paused || len(p.waitObjs) > 0 || p.Body == nil {
			continue
		}
		if !p.Body(p, nil) {
			if len(p.waitObjs) == 0 && !p.done {
				// Body 返回 false 且未注册等待=自然结束（官方脚本 return）。
				p.done = true
			}
		}
	}
}

// ExecutePass2 复刻泵第二遍（0x100fcfc0 位）：清除终止过程
//（官方 DeleteNode/EraseNode），随后调用方帧尾 FlushDeleted。
func (m *AsyncManager) ExecutePass2() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for id, p := range m.procs {
		if p.done {
			delete(m.procs, id)
			n++
		}
	}
	return n
}

// Snapshot 过程名快照（Dump 面）。
func (m *AsyncManager) Snapshot() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.procs))
	for _, p := range m.procs {
		out = append(out, fmt.Sprintf("%d:%s", p.ID, p.Name))
	}
	sort.Strings(out)
	return out
}
