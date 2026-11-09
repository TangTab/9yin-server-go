// Package modulevm 提供模块式 Lua 脚本宿主的通用内核：装载（parse→compile→
// run，捕获 chunk 返回的模块表）、按名索引、函数调用桥与坏脚本隔离。域包
// （neigong/itemuse/skillscript/combatformula）以薄封装复用，各自实现域校验
// （EachModule 回调内读 M.meta 建目录）、ctx 表构建与返回值白名单转换。
//
// 与 internal/luascript（事件型宿主：game.on 注册、出错整机重建）的分工：
// 模块脚本是 `local M = {} … return M` 的纯函数库，错误策略按热路径要求收窄
// 爆炸半径——
//
//   - 装载期坏文件：跳过该模块并汇总错误（软语义），不中断其余；
//   - 运行期坏脚本：**只禁用该模块**（WARN 一次），VM 不重建——一次坏调用
//     最坏造成一次 ≤timeout 的持锁停顿，之后该模块静默零效果，其余模块与
//     调用方链路完全不受影响；
//   - 无模块 / 无函数 / 已禁用 / 返回 nil：Invoke 返回 ok=false，调用方走
//     各自的回退路径，永不因脚本层缺失而阻断。
//
// 沙盒护栏与 internal/luascript 相同：stdlib.Open 后无 provider（io/os/exec/
// debug/chan/time 不存在）；每次调用有 wall-clock 超时（默认 500ms）与指令
// 预算（每次调用前重置计数器，VM 长存不重建也不累积）；引擎 panic 被 recover
// 成该模块的禁用事件。单 VM 由 mu 串行化，mu 是叶子锁（同 luascript 锁纪律）。
package modulevm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	// DefaultTimeout 是单次模块函数调用的默认 wall-clock 预算。
	DefaultTimeout = 500 * time.Millisecond
	// maxInstructions 是单次调用的指令预算（每次调用前重置），与超时互为
	// 冗余护栏；模块 VM 从不重建，必须按调用重置以免预算跨调用累积耗尽。
	maxInstructions = 5_000_000
)

// Module 是装载后的一个模块脚本：模块名（文件名主干）与模块表引用。
type Module struct {
	Name  string
	table vm.Value
}

// Table 返回模块表引用（M）。
func (m *Module) Table() vm.Value { return m.table }

// Host 管理单实例 Lua VM 与一个模块脚本目录。零值不可用，经 New 创建。
// 域包不直接暴露本类型，经各自 New 包装并补域目录。
type Host struct {
	mu      sync.Mutex
	dir     string
	logger  Logger
	timeout time.Duration

	vm         *vm.VM
	modules    map[string]*Module
	order      []string // 排序后的模块名
	broken     map[string]bool
	warned     map[string]bool
	loaded     int
	generation int64 // 每次成功 Load 递增；域侧缓存据此失效
}

// New 创建模块宿主。dir 为空表示禁用：所有 Invoke 返回未命中，Load 为空
// 操作。logger 为 nil 时告警静默丢弃。
func New(dir string, logger Logger) *Host {
	return &Host{dir: dir, logger: logger, timeout: DefaultTimeout}
}

// SetTimeout 调整单次调用预算（测试用）。
func (h *Host) SetTimeout(d time.Duration) { h.timeout = d }

// Enabled 报告宿主是否配置了脚本目录。
func (h *Host) Enabled() bool { return h.dir != "" }

// Loaded 返回最近一次装载成功的模块数。
func (h *Host) Loaded() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.loaded
}

// Generation 返回装载代数：每次成功 Load/Reload 递增。域侧的静态快照
//（如 M.passive 属性缓存）据此在热重载后失效。
func (h *Host) Generation() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.generation
}

// Names 返回全部已装载模块名（排序后；含被禁用的）。
func (h *Host) Names() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.order...)
}

// Broken 返回运行期出错被禁用的模块名（排序后）。
func (h *Host) Broken() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.broken))
	for name := range h.broken {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Has 报告模块存在且未被禁用。
func (h *Host) Has(name string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.modules[name]
	return ok && !h.broken[name]
}

// Load （重新）创建 VM 并按文件名序装载脚本目录下的全部 *.lua 模块：
// 捕获 chunk 返回值，非表的模块计为装载失败（软语义，错误汇总返回）。
// 返回值为成功装载的模块数。域校验（meta 读取等）由域包经 EachModule 追加。
func (h *Host) Load() (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.loadLocked()
}

// Reload 重新装载脚本目录（GM 热重载）。
func (h *Host) Reload() (int, error) { return h.Load() }

// Close 释放 VM。之后的 Invoke 返回未命中。
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
	h.modules = nil
	h.order = nil
	h.broken = nil
	h.warned = nil
	h.loaded = 0
	return err
}

func (h *Host) loadLocked() (int, error) {
	if h.dir == "" {
		return 0, nil
	}
	h.closeLocked()
	entries, err := os.ReadDir(h.dir)
	if err != nil {
		return 0, fmt.Errorf("modulevm: read %s: %w", h.dir, err)
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
	h.modules = make(map[string]*Module)
	h.broken = make(map[string]bool)
	h.warned = make(map[string]bool)

	var failures []error
	for _, name := range names {
		mod, err := h.loadModuleLocked(name)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		h.modules[mod.Name] = mod
	}
	h.rebuildOrderLocked()
	h.loaded = len(h.modules)
	h.generation++
	return h.loaded, errors.Join(failures...)
}

func (h *Host) rebuildOrderLocked() {
	h.order = make([]string, 0, len(h.modules))
	for name := range h.modules {
		h.order = append(h.order, name)
	}
	sort.Strings(h.order)
}

// loadModuleLocked 装载单个模块文件：parse→compile→run，捕获 chunk 返回的
// 模块表。键取文件名主干。
func (h *Host) loadModuleLocked(name string) (*Module, error) {
	path := filepath.Join(h.dir, name)
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("modulevm: read %s: %w", name, err)
	}
	block, err := parser.Parse(name, string(src))
	if err != nil {
		return nil, fmt.Errorf("modulevm: parse %s: %w", name, err)
	}
	proto, err := compiler.Compile(name, block)
	if err != nil {
		return nil, fmt.Errorf("modulevm: compile %s: %w", name, err)
	}
	out, runErr := h.invokeLocked(name, func() ([]vm.Value, error) { return h.vm.Run(proto) })
	if runErr != nil {
		return nil, fmt.Errorf("modulevm: run %s: %w", name, runErr)
	}
	if len(out) == 0 || !out[0].IsTable() {
		return nil, fmt.Errorf("modulevm: %s: module must return a table (got %s)", name, ValueString(firstOr(out)))
	}
	return &Module{Name: strings.TrimSuffix(name, ".lua"), table: out[0]}, nil
}

// EachModule 在持锁状态下遍历已装载模块，供域包装载后读取 M.meta 建域
// 目录 / 做域校验。回调返回错误的模块由域包决定处置（通常 MarkBroken），
// 错误汇总返回（与 Load 的软语义衔接）。
func (h *Host) EachModule(fn func(name string, table vm.LuaTable) error) []error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.vm == nil {
		return nil
	}
	var failures []error
	for _, name := range h.order {
		table := h.modules[name].table
		if err := fn(name, table.AsTable()); err != nil {
			failures = append(failures, err)
		}
	}
	return failures
}

// Func 在模块上按序探测候选函数名，返回首个可调用的；模块缺失/已禁用/
// 无候选时返回 Nil。
func (h *Host) Func(name string, candidates ...string) vm.Value {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.funcLocked(name, candidates)
}

func (h *Host) funcLocked(name string, candidates []string) vm.Value {
	mod, ok := h.modules[name]
	if !ok || h.broken[name] {
		return vm.Nil
	}
	table := mod.table.AsTable()
	for _, candidate := range candidates {
		if fn := Field(table, candidate); fn.IsCallable() {
			return fn
		}
	}
	return vm.Nil
}

// Invoke 调用模块函数：按 candidates 依次探测（别名兜底由调用方把候选序
// 传入），arg 在持锁下以 VM 引用构建参数表。ok=false 表示未命中（无模块/
// 无函数/已禁用）或运行期出错——出错时该模块被禁用（WARN 一次），VM 不
// 重建。返回值为首个结果（无返回值时 Nil）。
func (h *Host) Invoke(name string, candidates []string, arg func(v *vm.VM) vm.Value) (vm.Value, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.vm == nil {
		return vm.Nil, false
	}
	fn := h.funcLocked(name, candidates)
	if !fn.IsCallable() {
		return vm.Nil, false
	}
	var argValue vm.Value = vm.Nil
	if arg != nil {
		argValue = arg(h.vm)
	}
	out, err := h.invokeLocked(name+"."+candidates[0], func() ([]vm.Value, error) {
		return h.vm.ProtectedCall(fn, []vm.Value{argValue})
	})
	if err != nil {
		h.broken[name] = true
		h.Warnf("modulevm: 禁用 %s（%s 钩子出错：%v）", name, candidates[0], err)
		return vm.Nil, false
	}
	if len(out) == 0 {
		return vm.Nil, true
	}
	return out[0], true
}

// invokeLocked 套上超时 context、按调用重置的指令预算与 panic 恢复执行
// run。调用方持有 mu。
func (h *Host) invokeLocked(label string, run func() ([]vm.Value, error)) (out []vm.Value, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()
	h.vm.SetContext(ctx)
	h.vm.ResetInstructionCount()
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

// MarkBroken 手动禁用一个模块（域校验失败等场景）并告警；format 描述原因。
func (h *Host) MarkBroken(name, format string, args ...any) {
	h.mu.Lock()
	if h.broken == nil {
		h.mu.Unlock()
		return
	}
	h.broken[name] = true
	h.mu.Unlock()
	h.Warnf("modulevm: 禁用 %s: "+format, append([]any{name}, args...)...)
}

// Warnf 打 WARN 日志；nil logger 静默。
func (h *Host) Warnf(format string, args ...any) {
	if h.logger == nil {
		return
	}
	h.logger(format, args...)
}

// WarnOnce 对同一键只告警一次，防止热路径日志刷屏。
func (h *Host) WarnOnce(key, format string, args ...any) {
	if h.logger == nil {
		return
	}
	h.mu.Lock()
	if h.warned == nil {
		h.warned = make(map[string]bool)
	}
	if h.warned[key] {
		h.mu.Unlock()
		return
	}
	h.warned[key] = true
	h.mu.Unlock()
	h.logger(format, args...)
}

// Timeout 返回当前单次调用预算。
func (h *Host) Timeout() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.timeout
}

// Field 按字符串键取表字段。LuaTable 接口没有 GetString（那在具体类型
// *Table 上），统一经 Get(Value) 取以兼容任意表实现。
func Field(t vm.LuaTable, key string) vm.Value {
	return t.Get(vm.NewString(key))
}

// Index 按数组下标（1 起）取表元素，用途同 Field。
func Index(t vm.LuaTable, i int) vm.Value {
	return t.Get(vm.NewInt(int64(i)))
}

// ValueString 把 Lua 值序列化为可读文本（WARN 日志用；表浅层展开至 8 项）。
func ValueString(v vm.Value) string {
	switch {
	case v.IsNil():
		return "nil"
	case v.IsBool():
		return strconv.FormatBool(v.AsBool())
	case v.IsInt():
		return strconv.FormatInt(v.AsInt(), 10)
	case v.IsFloat():
		return strconv.FormatFloat(v.AsFloat(), 'g', -1, 64)
	case v.IsString():
		return v.AsString()
	case v.IsTable():
		if table := v.AsTable(); table != nil {
			var parts []string
			count := 0
			key := vm.Nil
			for {
				nextKey, value, err := table.Next(key)
				if err != nil || nextKey.IsNil() {
					break
				}
				if count >= 8 {
					parts = append(parts, "...")
					break
				}
				count++
				parts = append(parts, ValueString(nextKey)+"="+ValueString(value))
				key = nextKey
			}
			return "{" + strings.Join(parts, ", ") + "}"
		}
		return "table"
	default:
		return fmt.Sprintf("%s(%s)", v.Type(), v.String())
	}
}

func firstOr(out []vm.Value) vm.Value {
	if len(out) == 0 {
		return vm.Nil
	}
	return out[0]
}

// LuaInt 把 Lua 数值安全转 int64（NaN/Inf 拒绝；浮点取最近整数）。域包的
// 返回值白名单转换共用。
func LuaInt(v vm.Value) (int64, bool) {
	switch {
	case v.IsInt():
		return v.AsInt(), true
	case v.IsFloat():
		f := v.AsFloat()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false
		}
		return int64(math.Round(f)), true
	default:
		return 0, false
	}
}

// LuaFloat 把 Lua 数值安全转 float64（NaN/Inf 拒绝）。
func LuaFloat(v vm.Value) (float64, bool) {
	if !v.IsNumber() {
		return 0, false
	}
	f := v.AsFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// LuaBool 把任意 Lua 值按真值转 bool（nil/false 为假，其余为真）。
func LuaBool(v vm.Value) bool { return v.ToBool() }
