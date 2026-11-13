// Package luascript 在服务端内嵌 Lua 5.5 脚本宿主（third_party/golua，
// 纯 Go 实现）。脚本从单一目录加载，通过 game.on(event, handler) 声明
// 事件处理器；玩法侧经 Call 把调用环境快照递给脚本。
//
// 隔离与安全约定：
//   - 沙盒：不注册任何 provider，io/os/exec/debug/chan/time 模块不存在；
//   - 每次 Call/Eval 都有 wall-clock 超时（默认 500ms）与指令预算，
//     死循环脚本会以错误返回而不是挂死连接；
//   - 引擎自身的 panic 被 recover 成错误；运行期出错后 VM 整体重建
//     （重建即重新加载脚本目录），脏状态不会跨调用存活；
//   - 单 VM 由 mu 串行化。锁纪律：mu 是叶子锁——调用方可以在已持有
//     场景锁的状态下调用本包，但本包的宿主函数绝不反向获取场景锁，
//     副作用一律经 EventContext 里的回调由调用方执行。
package luascript

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// Logger 与标准库 log.Printf 兼容；宿主注入，包内不直接打日志。
type Logger func(format string, args ...any)

// PlayerSnapshot 是脚本可见的玩家快照（ctx:player()）。
type PlayerSnapshot struct {
	ID      uint32
	Name    string
	Faction string
	Silver  int64
	SP      int64
	Scene   string
	X, Y, Z float64
}

// NPCSnapshot 是脚本可见的 NPC 快照（ctx:npc()）。
type NPCSnapshot struct {
	ID       uint32
	ConfigID string
	Name     string
}

// ServiceSnapshot 是脚本可见的 NPC 服务快照（ctx:service()）。
type ServiceSnapshot struct {
	Mark   uint16
	Label  string
	Value  string
	Source string
}

// TaskSnapshot 是脚本可见的任务快照（ctx:tasks() 的元素）。文本为
// 去除富文本标记后的纯文本。
type TaskSnapshot struct {
	ID            int
	Title         string
	Target        string
	CanAcceptMenu string
}

// DungeonSnapshot 是脚本可见的副本实例快照（ctx:dungeon()）。
type DungeonSnapshot struct {
	Config     string
	Resource   string
	Instance   string
	Difficulty int
	Wave       int // 当前在场波次值（NpcProgress 刻度；0=无波次数据）
	NextWave   int // 下一个未注册波次值（0=已是末波）
	Killed     int // 副本内击杀累计
}

// EventContext 描述一次事件调用的环境数据与副作用回调。回调由调用方
// 提供，并在调用方的锁纪律下执行（例如已持 run.mu 时直接写帧）。
type EventContext struct {
	Player  PlayerSnapshot
	NPC     NPCSnapshot
	Service ServiceSnapshot
	// Dungeon 是副本事件（dungeon_*）携带的实例快照；非副本事件为零值。
	Dungeon DungeonSnapshot
	// Tasks 是本次事件 NPC 绑定且已加载文本的任务（npc_service 事件）。
	Tasks []TaskSnapshot
	// Notify 发送剧情提示卡（服务端 custom 383 路线）。
	Notify func(brief, chapterName string) error
	// Bubble 发送 NPC 说话气泡（服务端 custom 43 路线）。
	Bubble func(textID string) error
	// SpawnWave 物化一个副本波次组（dungeon_wave 引擎；回调在调用方
	// 场景锁纪律下执行）。
	SpawnWave func(wave int) error
	// After 声明一个延时副本事件回调（定时器由引擎持有，秒 ∈ (0,600]）。
	After func(seconds float64, event string) error
}

const (
	// DefaultTimeout 是单次脚本调用的默认 wall-clock 预算。
	DefaultTimeout = 500 * time.Millisecond
	// maxInstructions 是单次调用的指令预算，与超时互为冗余护栏。
	maxInstructions = 5_000_000
	// maxEvalSource 限制 GM lua_eval 的输入长度。
	maxEvalSource = 16 << 10
	// maxResultText 限制 Eval 结果文本长度。
	maxResultText = 4 << 10
)

// Host 管理单实例 Lua VM 与脚本目录。零值不可用，经 New 创建。
type Host struct {
	mu      sync.Mutex
	dir     string
	logger  Logger
	timeout time.Duration

	vm          *vm.VM
	handlers    map[string]vm.Value
	loadedCount int
}

// New 创建脚本宿主。dir 为空表示禁用：所有 Call 返回未处理，LoadDir
// 为空操作。logger 为 nil 时脚本 game.log 静默丢弃。
func New(dir string, logger Logger) *Host {
	return &Host{dir: dir, logger: logger, timeout: DefaultTimeout}
}

// LoadDir （重新）创建 VM 并按文件名序执行脚本目录下的全部 *.lua。
// 软语义：单个脚本出错不中断其余脚本，错误汇总返回；返回值为成功
// 加载的脚本数。Reload 是它的别名，供 GM 热重载使用。
func (h *Host) LoadDir() (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.loadDirLocked()
}

// Reload 重新加载脚本目录。
func (h *Host) Reload() (int, error) { return h.LoadDir() }

// Loaded 返回最近一次成功加载的脚本数。
func (h *Host) Loaded() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.loadedCount
}

// Enabled 报告宿主是否配置了脚本目录（-scripts 非空）。
func (h *Host) Enabled() bool { return h.dir != "" }

// HasHandler 报告事件是否已有脚本处理器。无 VM（未加载/已关闭/禁用）
// 时恒为 false。
func (h *Host) HasHandler(event string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	fn, ok := h.handlers[event]
	return ok && fn.IsCallable()
}

// Close 释放 VM。之后的 Call 返回未处理，Eval 返回错误。
func (h *Host) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closeLocked()
}

func (h *Host) closeLocked() error {
	if h.vm == nil {
		return nil
	}
	err := h.vm.Close(context.Background())
	h.vm = nil
	h.handlers = nil
	h.loadedCount = 0
	return err
}

// loadDirLocked 重建 VM 并执行目录内脚本。调用方持有 mu。
func (h *Host) loadDirLocked() (int, error) {
	if h.dir == "" {
		return 0, nil
	}
	h.closeLocked()
	entries, err := os.ReadDir(h.dir)
	if err != nil {
		return 0, fmt.Errorf("luascript: read %s: %w", h.dir, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	h.vm = vm.New(vm.WithContext(context.Background()), vm.WithLimits(vm.Limits{MaxInstructions: maxInstructions}))
	stdlib.Open(h.vm) // 无 provider：io/os/exec/debug/chan/time 不注册
	h.handlers = make(map[string]vm.Value)
	h.installGameLocked()

	var failures []error
	loaded := 0
	for _, name := range names {
		if err := h.runFileLocked(name); err != nil {
			failures = append(failures, err)
			continue
		}
		loaded++
	}
	h.loadedCount = loaded
	return loaded, errors.Join(failures...)
}

func (h *Host) runFileLocked(name string) error {
	path := filepath.Join(h.dir, name)
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("luascript: read %s: %w", name, err)
	}
	block, err := parser.Parse(name, string(src))
	if err != nil {
		return fmt.Errorf("luascript: parse %s: %w", name, err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		return fmt.Errorf("luascript: compile %s: %w", name, err)
	}
	if _, err := h.vm.Run(proto); err != nil {
		return fmt.Errorf("luascript: run %s: %w", name, err)
	}
	return nil
}

// installGameLocked 注册 game 表（on/log）。调用方持有 mu。
func (h *Host) installGameLocked() {
	game := vm.NewEmptyTable()
	h.vm.SetGlobal("game", vm.NewTable(game))
	game.SetString("on", vm.NewNativeFunc(func(l *vm.VM) int {
		if l.ArgCount() != 2 {
			panic(&vm.LuaError{Value: vm.NewString("game.on(event, handler) 需要两个参数")})
		}
		event := l.Get(1)
		handler := l.Get(2)
		if !event.IsString() || !handler.IsCallable() {
			panic(&vm.LuaError{Value: vm.NewString("game.on 的参数应为事件名与可调用处理器")})
		}
		h.handlers[event.AsString()] = handler
		return 0
	}))
	game.SetString("log", vm.NewNativeFunc(func(l *vm.VM) int {
		n := l.ArgCount()
		if n == 0 {
			return 0
		}
		args := make([]any, 0, n-1)
		for i := 2; i <= n; i++ {
			args = append(args, goValue(l.Get(i)))
		}
		if h.logger == nil {
			return 0
		}
		format := l.Get(1)
		if format.IsString() {
			h.logger("lua: "+format.AsString(), args...)
			return 0
		}
		// 首参不是格式串时退化为空格拼接，避免把数字静默吞成空格式。
		parts := append([]any{goValue(format)}, args...)
		h.logger("lua: "+strings.TrimRight(strings.Repeat("%v ", len(parts)), " "), parts...)
		return 0
	}))
}

// Call 执行事件处理器。没有处理器时返回 (false, nil)，调用方走默认
// 路径。处理器返回 false（显式）表示拒绝处理；无返回值或真值视为已
// 处理。处理器出错时返回 (true, err) 且 VM 随即重建。
func (h *Host) Call(event string, ec *EventContext) (handled bool, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.vm == nil {
		return false, nil
	}
	fn, ok := h.handlers[event]
	if !ok || !fn.IsCallable() {
		return false, nil
	}
	ctxTable := newEventTable(h.vm, ec)
	out, callErr := h.invokeLocked(event, func() ([]vm.Value, error) {
		return h.vm.ProtectedCall(fn, []vm.Value{vm.NewTable(ctxTable)})
	})
	if callErr != nil {
		h.rebuildLocked(callErr)
		return true, fmt.Errorf("luascript: %s handler: %w", event, callErr)
	}
	handled = len(out) == 0 || out[0].ToBool()
	return handled, nil
}

// Eval 在沙盒内执行一段 Lua 代码（GM 调试入口），返回结果的序列化
// 文本。出错时 VM 随即重建。
func (h *Host) Eval(src string) (string, error) {
	if len(src) > maxEvalSource {
		return "", fmt.Errorf("luascript: eval source exceeds %d bytes", maxEvalSource)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.vm == nil {
		return "", fmt.Errorf("luascript: host is not loaded")
	}
	block, err := parser.Parse("gm_eval", src)
	if err != nil {
		return "", fmt.Errorf("luascript: parse: %w", err)
	}
	proto, err := compiler.Compile("gm_eval", block)
	if err != nil {
		return "", fmt.Errorf("luascript: compile: %w", err)
	}
	// 注意不能把 proto 手工包成 Closure 再 ProtectedCall：那会丢失 _ENV
	// upvalue 绑定。Run 负责绑定并在内部恢复脚本错误。
	out, err := h.invokeLocked("gm_eval", func() ([]vm.Value, error) {
		h.vm.SetChunkName("gm_eval")
		return h.vm.Run(proto)
	})
	if err != nil {
		h.rebuildLocked(err)
		return "", err
	}
	parts := make([]string, 0, len(out))
	for _, value := range out {
		parts = append(parts, valueString(value, 0))
	}
	text := strings.Join(parts, "\t")
	if len(text) > maxResultText {
		text = text[:maxResultText] + "…"
	}
	return text, nil
}

// invokeLocked 套上超时 context 与 panic 恢复执行 run。调用方持有 mu。
func (h *Host) invokeLocked(event string, run func() ([]vm.Value, error)) (out []vm.Value, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()
	h.vm.SetContext(ctx)
	defer func() {
		if r := recover(); r != nil {
			out = nil
			err = fmt.Errorf("%s: vm panic: %v", event, r)
		}
	}()
	out, err = run()
	if ctx.Err() != nil && err == nil {
		err = ctx.Err()
	}
	return out, err
}

// rebuildLocked 在运行期错误后重建 VM 并重新加载脚本目录，避免脏
// 状态跨调用存活。重建失败仅记录，不影响返回给调用方的原始错误。
func (h *Host) rebuildLocked(cause error) {
	if h.logger != nil {
		h.logger("luascript: rebuilding vm after error: %v", cause)
	}
	if _, err := h.loadDirLocked(); err != nil && h.logger != nil {
		h.logger("luascript: reload after error: %v", err)
	}
}

// newEventTable 构建传给处理器的 ctx 表：快照方法 + 副作用方法。
// 方法均以 ctx:method() 形式调用（self 位于参数 1，结果写入槽 0）。
func newEventTable(v *vm.VM, ec *EventContext) *vm.Table {
	ctx := vm.NewEmptyTable()
	ctx.SetString("player", vm.NewNativeFunc(func(l *vm.VM) int {
		t := vm.NewEmptyTable()
		t.SetString("id", vm.NewInt(int64(ec.Player.ID)))
		setString(t, "name", ec.Player.Name)
		setString(t, "faction", ec.Player.Faction)
		t.SetString("silver", vm.NewInt(ec.Player.Silver))
		t.SetString("sp", vm.NewInt(ec.Player.SP))
		setString(t, "scene", ec.Player.Scene)
		t.SetString("x", vm.NewFloat(ec.Player.X))
		t.SetString("y", vm.NewFloat(ec.Player.Y))
		t.SetString("z", vm.NewFloat(ec.Player.Z))
		l.Set(0, vm.NewTable(t))
		return 1
	}))
	ctx.SetString("npc", vm.NewNativeFunc(func(l *vm.VM) int {
		t := vm.NewEmptyTable()
		t.SetString("id", vm.NewInt(int64(ec.NPC.ID)))
		setString(t, "config_id", ec.NPC.ConfigID)
		setString(t, "name", ec.NPC.Name)
		l.Set(0, vm.NewTable(t))
		return 1
	}))
	ctx.SetString("service", vm.NewNativeFunc(func(l *vm.VM) int {
		t := vm.NewEmptyTable()
		t.SetString("mark", vm.NewInt(int64(ec.Service.Mark)))
		setString(t, "label", ec.Service.Label)
		setString(t, "value", ec.Service.Value)
		setString(t, "source", ec.Service.Source)
		l.Set(0, vm.NewTable(t))
		return 1
	}))
	ctx.SetString("dungeon", vm.NewNativeFunc(func(l *vm.VM) int {
		t := vm.NewEmptyTable()
		setString(t, "config", ec.Dungeon.Config)
		setString(t, "resource", ec.Dungeon.Resource)
		setString(t, "instance", ec.Dungeon.Instance)
		if ec.Dungeon.Difficulty != 0 {
			t.SetString("difficulty", vm.NewInt(int64(ec.Dungeon.Difficulty)))
		}
		if ec.Dungeon.Wave != 0 {
			t.SetString("wave", vm.NewInt(int64(ec.Dungeon.Wave)))
		}
		if ec.Dungeon.NextWave != 0 {
			t.SetString("next_wave", vm.NewInt(int64(ec.Dungeon.NextWave)))
		}
		if ec.Dungeon.Killed != 0 {
			t.SetString("killed", vm.NewInt(int64(ec.Dungeon.Killed)))
		}
		l.Set(0, vm.NewTable(t))
		return 1
	}))
	ctx.SetString("spawn_wave", vm.NewNativeFunc(func(l *vm.VM) int {
		// ctx:spawn_wave(n)：1=self。
		if l.ArgCount() < 2 || ec.SpawnWave == nil {
			panic(&vm.LuaError{Value: vm.NewString("ctx:spawn_wave(n) 不可用")})
		}
		wave := l.Get(2)
		if !wave.IsNumber() {
			panic(&vm.LuaError{Value: vm.NewString("ctx:spawn_wave 需要波次数值（NpcProgress 刻度）")})
		}
		if err := ec.SpawnWave(int(wave.AsInt())); err != nil {
			panic(&vm.LuaError{Value: vm.NewString("spawn_wave: " + err.Error())})
		}
		l.Set(0, vm.True)
		return 1
	}))
	ctx.SetString("after", vm.NewNativeFunc(func(l *vm.VM) int {
		// ctx:after(seconds, event)：1=self。
		if l.ArgCount() < 3 || ec.After == nil {
			panic(&vm.LuaError{Value: vm.NewString("ctx:after(seconds, event) 不可用")})
		}
		seconds := l.Get(2)
		event := l.Get(3)
		if !seconds.IsNumber() || !event.IsString() {
			panic(&vm.LuaError{Value: vm.NewString("ctx:after 需要秒数与事件名")})
		}
		if err := ec.After(seconds.AsFloat(), event.AsString()); err != nil {
			panic(&vm.LuaError{Value: vm.NewString("after: " + err.Error())})
		}
		l.Set(0, vm.True)
		return 1
	}))
	ctx.SetString("tasks", vm.NewNativeFunc(func(l *vm.VM) int {
		tasks := vm.NewEmptyTable()
		for i, task := range ec.Tasks {
			entry := vm.NewEmptyTable()
			entry.SetString("id", vm.NewInt(int64(task.ID)))
			setString(entry, "title", task.Title)
			setString(entry, "target", task.Target)
			setString(entry, "can_accept_menu", task.CanAcceptMenu)
			tasks.SetInt(i+1, vm.NewTable(entry))
		}
		l.Set(0, vm.NewTable(tasks))
		return 1
	}))
	ctx.SetString("notify", vm.NewNativeFunc(func(l *vm.VM) int {
		// 方法调用 ctx:notify(brief, chapter?) 的参数布局：1=self。
		if l.ArgCount() < 2 || ec.Notify == nil {
			panic(&vm.LuaError{Value: vm.NewString("ctx:notify(brief, chapter_name?) 不可用")})
		}
		brief := l.Get(2).AsString()
		chapter := "ui_shop"
		if l.ArgCount() >= 3 && !l.Get(3).IsNil() {
			chapter = l.Get(3).AsString()
		}
		if err := ec.Notify(brief, chapter); err != nil {
			panic(&vm.LuaError{Value: vm.NewString("notify: " + err.Error())})
		}
		l.Set(0, vm.True)
		return 1
	}))
	ctx.SetString("bubble", vm.NewNativeFunc(func(l *vm.VM) int {
		if l.ArgCount() < 2 || ec.Bubble == nil {
			panic(&vm.LuaError{Value: vm.NewString("ctx:bubble(text_id) 不可用")})
		}
		if err := ec.Bubble(l.Get(2).AsString()); err != nil {
			panic(&vm.LuaError{Value: vm.NewString("bubble: " + err.Error())})
		}
		l.Set(0, vm.True)
		return 1
	}))
	return ctx
}

func setString(t *vm.Table, key, value string) {
	if value != "" {
		t.SetString(key, vm.NewString(value))
	}
}
