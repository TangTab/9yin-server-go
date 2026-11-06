// Package neigong 装载内功特效模块脚本（scripts/module/neigong/buf_ng_*.lua，
// 自参考服拷贝的 155 本），并向未来的战斗分发层提供钩子调用桥。
//
// 与 internal/luascript（事件型宿主：game.on 注册、npc_service 事件、出错整机
// 重建）的契约不同：脚本形态是模块（local M = {} … return M），暴露 M.meta /
// M.levels 与运行期钩子（M.on_attack(ctx) 等，返回效果指令表）。因此本包自建
// Lua VM，错误策略按战斗热路径要求收窄爆炸半径：
//
//   - 装载期坏文件：跳过该本并汇总错误（软语义，同 luascript.Host.LoadDir）；
//   - 运行期坏脚本：**只禁用该 buffId**（WARN 一次），VM 不重建——一次坏调用
//     最坏造成一次 ≤timeout 的持锁停顿，之后该本静默零效果，其余各本与出招
//     链路完全不受影响；
//   - 无书 / 无模块 / 无钩子 / 已禁用 / 钩子返回 nil：一律返回 nil 效果，
//     调用方（战斗二期）据此走零效果路径，永不因脚本层缺失而阻断。
//
// 沙盒护栏与 luascript.Host 相同：stdlib.Open 后无 provider（io/os/exec/debug/
// chan/time 不存在）；每次调用有 wall-clock 超时（默认 500ms）与指令预算
// （每次调用前重置计数器，VM 长存不重建也不累积）；引擎 panic 被 recover 成
// 该本的禁用事件。脚本为纯函数：模块无跨玩家共享可变状态，特效状态由调用方
// 持有、经 Ctx.State 传入传出。
package neigong

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

const (
	// DefaultTimeout 是单次钩子调用的默认 wall-clock 预算。
	DefaultTimeout = 500 * time.Millisecond
	// maxInstructions 是单次调用的指令预算（每次调用前重置），与超时互为
	// 冗余护栏；内功 VM 从不重建，必须按调用重置以免预算跨调用累积耗尽。
	maxInstructions = 5_000_000
)

// hookNames 是装载期探测的钩子函数全集（M.<name>）。on_hit/on_be_hit/on_tick
// 是生成桩对 on_attack/on_victim/on_periodic 的别名，经 hookChain 兜底。
var hookNames = []string{
	"on_attack", "on_victim", "on_parry", "on_periodic",
	"on_tick", "on_hit", "on_be_hit", "passive",
}

// hookChain 把战斗事件名解析为按序探测的钩子名：主钩子缺失时尝试别名
// （对齐参考服 NeiGongEffects 的分发语义）。不在映射内的名字按原样直连，
// 因此调用方也可以直接传裸钩子名（on_hit 等）。
func hookChain(event string) []string {
	switch event {
	case "on_attack":
		return []string{"on_attack", "on_hit"}
	case "on_victim":
		return []string{"on_victim", "on_be_hit"}
	case "on_periodic":
		return []string{"on_periodic", "on_tick"}
	default:
		return []string{event}
	}
}

// Book 是一本内功特效脚本的目录项（装载时读自 M.meta）。
type Book struct {
	ID       string // buff id，即模块键（buf_ng_bg_001）
	Name     string // 内功名（大巧若拙经）
	Book     string // 书 id（ng_bg_001）
	Category string // 分类（hit/passive/…）
	MaxTier  int    // 层数
}

// module 是装载后的一本脚本：模块表引用、目录项与已存在的钩子函数。
type module struct {
	table vm.Value
	meta  Book
	hooks map[string]vm.Value
}

// Effects 管理内功特效模块的单实例 Lua VM 与脚本目录。零值不可用，经 New
// 创建。单 VM 由 mu 串行化；mu 是叶子锁（同 luascript.Host 锁纪律）。
type Effects struct {
	mu      sync.Mutex
	dir     string
	logger  Logger
	timeout time.Duration

	vm      *vm.VM
	modules map[string]*module
	byBook  map[string]string // 书 id → buff id
	order   []string          // 排序后的 buff id（Books 的稳定序）
	broken  map[string]bool   // 运行期出错被禁用的 buff id
	warned  map[string]bool   // 已告警的 (buff:键) 组合，防热路径日志刷屏
	loaded  int
}

// New 创建内功特效宿主。dir 为空表示禁用：所有调用返回零效果，Load 为空
// 操作。logger 为 nil 时告警静默丢弃（测试可用）。
func New(dir string, logger Logger) *Effects {
	return &Effects{dir: dir, logger: logger, timeout: DefaultTimeout}
}

// Enabled 报告宿主是否配置了脚本目录（-neigong-scripts 非空）。
func (e *Effects) Enabled() bool { return e.dir != "" }

// Loaded 返回最近一次装载成功的脚本数。
func (e *Effects) Loaded() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.loaded
}

// Count 是 Loaded 的别名，与其余目录型注册表（npcfunc 等）的命名对齐。
func (e *Effects) Count() int { return e.Loaded() }

// Broken 返回运行期出错被禁用的 buff id（排序后）。
func (e *Effects) Broken() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]string, 0, len(e.broken))
	for id := range e.broken {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Books 返回全部已装载内功的目录项（按 buff id 排序）。
func (e *Effects) Books() []Book {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Book, 0, len(e.order))
	for _, id := range e.order {
		out = append(out, e.modules[id].meta)
	}
	return out
}

// Book 按 buff id 查目录项。
func (e *Effects) Book(id string) (Book, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	m, ok := e.modules[id]
	if !ok {
		return Book{}, false
	}
	return m.meta, true
}

// ByBook 按书 id（ng_jh_001）反查 buff id（buf_ng_jh_001）。
func (e *Effects) ByBook(book string) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	id, ok := e.byBook[book]
	return id, ok
}

// Load （重新）创建 VM 并按文件名序装载脚本目录下的全部 *.lua 模块。
// 单个脚本坏（语法错 / 未返回表 / meta 缺失）不中断其余，错误汇总返回；
// 返回值为成功装载的模块数。
func (e *Effects) Load() (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.loadLocked()
}

// Reload 重新装载脚本目录（GM 热重载）。
func (e *Effects) Reload() (int, error) { return e.Load() }

// Close 释放 VM。之后的 Hook/Passive 返回零效果。
func (e *Effects) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.closeLocked()
}

func (e *Effects) closeLocked() error {
	if e.vm == nil {
		return nil
	}
	err := e.vm.Close(context.Background())
	e.vm = nil
	e.modules = nil
	e.byBook = nil
	e.order = nil
	e.broken = nil
	e.warned = nil
	e.loaded = 0
	return err
}

func (e *Effects) loadLocked() (int, error) {
	if e.dir == "" {
		return 0, nil
	}
	e.closeLocked()
	entries, err := os.ReadDir(e.dir)
	if err != nil {
		return 0, fmt.Errorf("neigong: read %s: %w", e.dir, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	e.vm = vm.New(vm.WithContext(context.Background()), vm.WithLimits(vm.Limits{MaxInstructions: maxInstructions}))
	stdlib.Open(e.vm) // 无 provider：io/os/exec/debug/chan/time 不注册
	e.modules = make(map[string]*module)
	e.byBook = make(map[string]string)
	e.broken = make(map[string]bool)
	e.warned = make(map[string]bool)

	var failures []error
	for _, name := range names {
		mod, err := e.loadModuleLocked(name)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		id := mod.meta.ID
		e.modules[id] = mod
		e.order = append(e.order, id)
		if mod.meta.Book != "" {
			if prev, dup := e.byBook[mod.meta.Book]; !dup || id < prev {
				e.byBook[mod.meta.Book] = id
			}
		}
	}
	sort.Strings(e.order)
	e.loaded = len(e.modules)
	return e.loaded, errors.Join(failures...)
}

// loadModuleLocked 装载单个模块文件：parse→compile→run，捕获 chunk 返回的
// 模块表，读 M.meta 建目录项。键取文件名主干；meta.id 不一致时以文件名为
// 准并告警（脚本名即 buff id 的仓内约定优先）。
func (e *Effects) loadModuleLocked(name string) (*module, error) {
	path := filepath.Join(e.dir, name)
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("neigong: read %s: %w", name, err)
	}
	block, err := parser.Parse(name, string(src))
	if err != nil {
		return nil, fmt.Errorf("neigong: parse %s: %w", name, err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		return nil, fmt.Errorf("neigong: compile %s: %w", name, err)
	}
	out, runErr := e.invokeLocked(name, func() ([]vm.Value, error) { return e.vm.Run(proto) })
	if runErr != nil {
		return nil, fmt.Errorf("neigong: run %s: %w", name, runErr)
	}
	if len(out) == 0 || !out[0].IsTable() {
		return nil, fmt.Errorf("neigong: %s: module must return a table (got %s)", name, valueTypeName(out))
	}
	table := out[0].AsTable()

	stem := strings.TrimSuffix(name, ".lua")
	meta, metaErr := readMeta(table, stem)
	if metaErr != nil {
		return nil, fmt.Errorf("neigong: %s: %w", name, metaErr)
	}
	if meta.ID != stem {
		e.warnf("%s: meta.id %q 与文件名不符，以文件名为键 %q", stem, meta.ID, stem)
		meta.ID = stem
	}
	hooks := make(map[string]vm.Value)
	for _, hook := range hookNames {
		if fn := tableGet(table, hook); fn.IsCallable() {
			hooks[hook] = fn
		}
	}
	return &module{table: out[0], meta: meta, hooks: hooks}, nil
}

// tableGet 按字符串键取值。LuaTable 接口没有 GetString（那在具体类型
// *Table 上），统一经 Get(Value) 取以兼容任意表实现。
func tableGet(t vm.LuaTable, key string) vm.Value {
	return t.Get(vm.NewString(key))
}

// readMeta 从模块表读 M.meta{id,name,book,category,max_tier}。meta 表或
// id/name 缺失视为坏脚本（155 本仓内脚本均有完整 meta）。
func readMeta(table vm.LuaTable, stem string) (Book, error) {
	metaValue := tableGet(table, "meta")
	if !metaValue.IsTable() {
		return Book{}, fmt.Errorf("missing M.meta table")
	}
	meta := metaValue.AsTable()
	book := Book{
		ID:       tableGet(meta, "id").AsString(),
		Name:     tableGet(meta, "name").AsString(),
		Book:     tableGet(meta, "book").AsString(),
		Category: tableGet(meta, "category").AsString(),
		MaxTier:  1,
	}
	if book.ID == "" || book.Name == "" {
		return Book{}, fmt.Errorf("M.meta needs id and name")
	}
	if tier := tableGet(meta, "max_tier"); tier.IsNumber() {
		if t := tier.AsInt(); t > 0 {
			book.MaxTier = int(t)
		}
	}
	return book, nil
}

// Hook 调用一本内功的事件钩子。event 为战斗事件名（on_attack/on_victim/
// on_parry/on_periodic）或裸钩子名，别名链见 hookChain。返回效果指令；
// nil 表示零效果（无模块/无钩子/已禁用/脚本返回 nil/返回值不可用）。
// 脚本运行期出错只禁用该本，不上抛错误——战斗链永不被脚本层阻断。
func (e *Effects) Hook(buffID, event string, ctx *Ctx) *Effect {
	result, ok := e.callHook(buffID, event, ctx)
	if !ok || result.IsNil() || (result.IsBool() && !result.AsBool()) {
		return nil
	}
	if !result.IsTable() {
		e.warnOnce(buffID+":return", "%s: %s 返回非表值 %s，按零效果处理", buffID, event, valueString(result))
		return nil
	}
	return e.convertEffect(buffID, event, result.AsTable())
}

// Passive 调用 M.passive，返回扁平数值属性表（max_mp_add/attack_add 等，
// 键开放、值为数值）。无钩子/坏脚本返回 nil。
func (e *Effects) Passive(buffID string, ctx *Ctx) map[string]float64 {
	result, ok := e.callHook(buffID, "passive", ctx)
	if !ok || result.IsNil() || !result.IsTable() {
		return nil
	}
	table := result.AsTable()
	out := make(map[string]float64)
	key := vm.Nil
	for {
		nextKey, value, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		if !nextKey.IsString() || !value.IsNumber() {
			e.warnOnce(buffID+":passive", "passive 非数值条目 %s=%s 已丢弃", valueString(nextKey), valueString(value))
			continue
		}
		out[nextKey.AsString()] = value.AsFloat()
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// callHook 解析钩子并以只读 ctx 表调用（内部持锁）。第二返回值 false 表示
// 无事发生（无模块/无钩子/已禁用），调用方直接走零效果路径。
func (e *Effects) callHook(buffID, event string, ctx *Ctx) (vm.Value, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vm == nil {
		return vm.Nil, false
	}
	mod, ok := e.modules[buffID]
	if !ok || e.broken[buffID] {
		return vm.Nil, false
	}
	var fn vm.Value
	for _, hook := range hookChain(event) {
		if candidate, exists := mod.hooks[hook]; exists {
			fn = candidate
			break
		}
	}
	if !fn.IsCallable() {
		return vm.Nil, false
	}
	if ctx == nil {
		ctx = &Ctx{}
	}
	ctxTable := vm.NewTable(buildCtxTable(e.vm, ctx))
	out, err := e.invokeLocked(buffID+"."+event, func() ([]vm.Value, error) {
		return e.vm.ProtectedCall(fn, []vm.Value{ctxTable})
	})
	if err != nil {
		e.broken[buffID] = true
		e.warnf("禁用 %s（%s 钩子出错：%v）", buffID, event, err)
		return vm.Nil, false
	}
	if len(out) == 0 {
		return vm.Nil, true
	}
	return out[0], true
}

// invokeLocked 套上超时 context、按调用重置的指令预算与 panic 恢复执行
// run。调用方持有 mu。
func (e *Effects) invokeLocked(label string, run func() ([]vm.Value, error)) (out []vm.Value, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()
	e.vm.SetContext(ctx)
	e.vm.ResetInstructionCount()
	defer func() {
		if r := recover(); r != nil {
			out = nil
			err = fmt.Errorf("%s: vm panic: %v", label, r)
		}
	}()
	out, err = run()
	if ctx.Err() != nil && err == nil {
		err = ctx.Err()
	}
	return out, err
}

// warnf 打 WARN 日志；nil logger 静默。
func (e *Effects) warnf(format string, args ...any) {
	if e.logger == nil {
		return
	}
	e.logger("neigong: "+format, args...)
}

// warnOnce 对同一 (buff:键) 组合只告警一次，防止战斗热路径日志刷屏。
func (e *Effects) warnOnce(key, format string, args ...any) {
	if e.logger == nil {
		return
	}
	e.mu.Lock()
	if e.warned == nil {
		e.warned = make(map[string]bool)
	}
	if e.warned[key] {
		e.mu.Unlock()
		return
	}
	e.warned[key] = true
	e.mu.Unlock()
	e.logger("neigong: "+format, args...)
}

func valueTypeName(out []vm.Value) string {
	if len(out) == 0 {
		return "no value"
	}
	return out[0].Type()
}
