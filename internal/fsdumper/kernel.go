package fsdumper

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// DB 是 CAccess 的 Go 等价物：官方经 Store 服转发，本复刻
// database/sql 直连（差异见包文档「与官方的差异」）。
type DB interface {
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
}

// dumpTimerEntry 对应官方 CDumpTimer beat：name+param 定位，
// first 首拍延迟、interval 轮转周期（0=单发）。
type dumpTimerEntry struct {
	name     string
	param    any
	fn       func(k *Kernel, param any, beat int)
	first    time.Duration
	interval time.Duration
	deadline time.Time
	beat     int
}

// Kernel 对应官方 CDumpKernel（IDumpKernel 17 槽虚面的 Go 投影）。
// 由 Server 持有，业务模块（DumpRole/DumpRes）只经它触达
// 定时器/DB/Echo/SendServerMsg。
type Kernel struct {
	mu     sync.Mutex
	timers []*dumpTimerEntry
	seq    int64

	db  map[string]DB // access 名 → 连接（role/public/manage/output）
	srv *Server

	// ResourcePath 对应 IDumpKernel::GetResourcePath（Res 目录）。
	ResourcePath string
}

func newKernel(srv *Server) *Kernel {
	return &Kernel{db: map[string]DB{}, srv: srv}
}

// AddAccess 对应 GameDumper::AddAccess(name, connstr) → CAccess 注册。
func (k *Kernel) AddAccess(name string, handle DB) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.db[name] = handle
}

// Access 取库名句柄；不存在报错（官方 AccessNotFound 断言日志同义）。
func (k *Kernel) Access(name string) (DB, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	h, ok := k.db[name]
	if !ok {
		return nil, fmt.Errorf("fsdumper: access %q 未注册", name)
	}
	return h, nil
}

// AddTimer 对应 IDumpKernel::AddTimer(name, cb, param, first, interval)。
// first/interval 单位毫秒，与官方签名一致。
func (k *Kernel) AddTimer(name string, param any, first, interval time.Duration,
	fn func(k *Kernel, param any, beat int)) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.timers = append(k.timers, &dumpTimerEntry{
		name: name, param: param, fn: fn,
		first: first, interval: interval,
		deadline: time.Now().Add(first),
	})
}

// FindTimer / RemoveTimer 对应虚表 +0x04/+0x0C。
func (k *Kernel) FindTimer(name string, param any) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, t := range k.timers {
		if t.name == name && (param == nil || t.param == param) {
			return true
		}
	}
	return false
}

func (k *Kernel) RemoveTimer(name string, param any) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	for i, t := range k.timers {
		if t.name == name && (param == nil || t.param == param) {
			k.timers = append(k.timers[:i], k.timers[i+1:]...)
			return true
		}
	}
	return false
}

// runDue 收割到期节拍（TimerCheck 主泵每 tick 调用）。
// 返回是否仍有存活定时器。
func (k *Kernel) runDue() bool {
	k.mu.Lock()
	now := time.Now()
	var due []*dumpTimerEntry
	kept := k.timers[:0]
	for _, t := range k.timers {
		if now.After(t.deadline) || now.Equal(t.deadline) {
			t.beat++
			if t.interval > 0 {
				t.deadline = now.Add(t.interval)
				kept = append(kept, t)
			}
			due = append(due, t)
		} else {
			kept = append(kept, t)
		}
	}
	k.timers = kept
	k.mu.Unlock()
	for _, t := range due {
		t.fn(k, t.param, t.beat)
	}
	return len(due) > 0 || len(k.timers) > 0
}

// SendServerMsg 对应虚表 +0x30：向 :4006 指定客户端（或 -1 全体）发
// text 方言行。
func (k *Kernel) SendServerMsg(clientID int, msg *textVarList) bool {
	if k.srv == nil {
		return false
	}
	return k.srv.SendTo(clientID, msg.AppendLine())
}

// SendResponse 对应虚表 +0x10：回 EchoSock 维护口一行。
func (k *Kernel) SendResponse(line string) {
	if k.srv != nil {
		k.srv.NotifyEcho(line)
	}
}

// NotifyEcho 维护口日志镜像（官方 SendResponse 走 EchoSock console）。
func (k *Kernel) NotifyEcho(line string) {
	if k.srv != nil {
		k.srv.NotifyEcho(line)
	}
}

// Quit 对应虚表 +0x40：触发关停（State→Closing）。
func (k *Kernel) Quit() {
	if k.srv != nil {
		k.srv.postEvent(stateEventClose)
	}
}

// SQLDB 把 *sql.DB 适配到 DB 接口（ctx 签名桥）。
type SQLDB struct{ DBHandle *sql.DB }

func (s SQLDB) Exec(ctx context.Context, q string, args ...any) (sql.Result, error) {
	return s.DBHandle.ExecContext(ctx, q, args...)
}

func (s SQLDB) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return s.DBHandle.QueryContext(ctx, q, args...)
}

func (s SQLDB) QueryRow(ctx context.Context, q string, args ...any) *sql.Row {
	return s.DBHandle.QueryRowContext(ctx, q, args...)
}
