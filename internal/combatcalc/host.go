// Package combatcalc 见 transpile.go 包注释。
//
// host.go：单实例长存 VM（stdlib.Open 沙盒无 provider）+ 叶子锁串行化，
// 装载期把 formula.ini 规格转译为 Lua 闭包、经 M.install_formulas 注入
// 引擎；运行期 CalcHit 走 ProtectedCall（500ms 超时 + 按调用重置的指令
// 预算 + panic 恢复）。运行期出错只禁用整引擎（WARN 一次），调用方全部
// 回退 legacy 面板×倍率掷骰——战斗链永不被脚本层阻断。
package combatcalc

import (
	"context"
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
	// DefaultTimeout 是单次 calc_hit 调用的默认 wall-clock 预算。
	DefaultTimeout = 500 * time.Millisecond
	// maxInstructions 是单次调用的指令预算（VM 长存不重建，按调用重置）。
	maxInstructions = 5_000_000
	// engineFileName 是脚本目录下的引擎文件名（scripts/module/combat/formula.lua）。
	engineFileName = "formula.lua"
	// maxFormulaParam 是 formula.ini 单公式的最大参数位（Prop1..Prop12）。
	maxFormulaParam = 12
)

// unsupportedSrc 是引擎不支持的参数来源码（buff 值 7/8、buff 释放者 11、
// 距离 9、主角保留 4、杂项 10/12/13）——含这些来源的公式整体跳过，对应
// 技能走 legacy。客户端 Src 直方图：0/5/6/1/2 已覆盖绝大多数玩家公式。
var unsupportedSrc = map[int]bool{4: true, 7: true, 8: true, 9: true, 10: true, 11: true, 12: true, 13: true}

// FormulaSpec 是 formula.ini 单节规格（宿主侧解析后传入，包不读 ini）。
type FormulaSpec struct {
	ID      int32
	Formula string
	Src     map[int]int    // 参数位 1..12 → Src 码
	Prop    map[int]string // 参数位 → Prop 名（Src=0/1/2 时有效）
}

// Engine 管理公式引擎的单实例 Lua VM。零值不可用，经 New 创建。
type Engine struct {
	mu      sync.Mutex
	dir     string
	logger  Logger
	timeout time.Duration

	vm         *vm.VM
	calcFn     vm.Value
	supported  map[int32]bool
	broken     bool
	loaded     int
	skipped    int
	skipReason map[string]int // 不支持原因 → 条数（启动日志汇总）
}

// New 创建公式引擎宿主。dir 为空表示禁用：Ready 恒 false（全走 legacy），
// Load 为空操作。logger 为 nil 时告警静默丢弃。
func New(dir string, logger Logger) *Engine {
	return &Engine{dir: dir, logger: logger, timeout: DefaultTimeout}
}

// Enabled 报告宿主是否配置了脚本目录（-combat-scripts 非空）。
func (e *Engine) Enabled() bool { return e.dir != "" }

// Ready 报告引擎当前是否可服务 CalcHit（已装载且未因运行期错误禁用）。
func (e *Engine) Ready() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.vm != nil && !e.broken
}

// Loaded 返回装载成功的公式数；Skipped 返回跳过（转译失败/不支持来源）数。
func (e *Engine) Loaded() (installed, skipped int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.loaded, e.skipped
}

// Supports 报告指定公式号是否已注入引擎（技能 damageRows 据此选路径）。
func (e *Engine) Supports(formulaID int32) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.vm != nil && !e.broken && e.supported[formulaID]
}

// Close 释放 VM。之后 CalcHit 一律回退。
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.closeLocked()
}

func (e *Engine) closeLocked() error {
	if e.vm == nil {
		return nil
	}
	err := e.vm.Close(context.Background())
	e.vm = nil
	e.calcFn = vm.Nil
	e.supported = nil
	e.loaded = 0
	e.skipped = 0
	e.skipReason = nil
	e.broken = false
	return err
}

// Load （重新）创建 VM、装载引擎脚本并注入全部公式闭包。单个公式坏不
// 中断其余（跳过计数）；引擎文件缺失/坏即整体不可用（返回错误，调用方
// 全走 legacy）。
func (e *Engine) Load(specs []FormulaSpec) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.loadLocked(specs)
}

// Reload 重新装载（GM lua_reload 联动）。
func (e *Engine) Reload(specs []FormulaSpec) (int, error) { return e.Load(specs) }

func (e *Engine) loadLocked(specs []FormulaSpec) (int, error) {
	if e.dir == "" {
		return 0, nil
	}
	e.closeLocked()

	path := filepath.Join(e.dir, engineFileName)
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("combatcalc: read %s: %w", path, err)
	}
	block, err := parser.Parse(engineFileName, string(src))
	if err != nil {
		return 0, fmt.Errorf("combatcalc: parse %s: %w", engineFileName, err)
	}
	proto, err := compiler.Compile(engineFileName, block)
	if err != nil {
		return 0, fmt.Errorf("combatcalc: compile %s: %w", engineFileName, err)
	}

	e.vm = vm.New(vm.WithContext(context.Background()), vm.WithLimits(vm.Limits{MaxInstructions: maxInstructions}))
	stdlib.Open(e.vm) // 无 provider：io/os/exec/debug/chan/time 不注册
	e.supported = make(map[int32]bool)
	e.skipReason = make(map[string]int)

	out, err := e.invokeLocked("engine.run", func() ([]vm.Value, error) { return e.vm.Run(proto) })
	if err != nil {
		e.closeLocked()
		return 0, fmt.Errorf("combatcalc: run %s: %w", engineFileName, err)
	}
	if len(out) == 0 || !out[0].IsTable() {
		e.closeLocked()
		return 0, fmt.Errorf("combatcalc: %s must return a table", engineFileName)
	}
	module := out[0].AsTable()
	calcFn := tableGet(module, "calc_hit")
	installFn := tableGet(module, "install_formulas")
	if !calcFn.IsCallable() || !installFn.IsCallable() {
		e.closeLocked()
		return 0, fmt.Errorf("combatcalc: %s needs M.calc_hit and M.install_formulas", engineFileName)
	}
	e.calcFn = calcFn

	formulas := vm.NewEmptyTable()
	bindings := vm.NewEmptyTable()
	for _, spec := range specs {
		closure, binding, err := e.compileFormulaLocked(spec)
		if err != nil {
			e.skipped++
			e.skipReason[err.Error()]++
			continue
		}
		formulas.SetInt(int(spec.ID), closure)
		bindings.SetInt(int(spec.ID), vm.NewTable(binding))
		e.supported[spec.ID] = true
		e.loaded++
	}
	if _, err := e.invokeLocked("engine.install", func() ([]vm.Value, error) {
		return e.vm.ProtectedCall(installFn, []vm.Value{vm.NewTable(formulas), vm.NewTable(bindings)})
	}); err != nil {
		e.closeLocked()
		return 0, fmt.Errorf("combatcalc: install_formulas: %w", err)
	}
	if e.loaded == 0 {
		e.warnf("公式引擎装载成功但无可用公式（%d 条规格全部跳过）", len(specs))
	}
	return e.loaded, nil
}

// compileFormulaLocked 转译并编译单条公式，返回闭包与参数绑定表。
func (e *Engine) compileFormulaLocked(spec FormulaSpec) (vm.Value, *vm.Table, error) {
	if strings.TrimSpace(spec.Formula) == "" {
		return vm.Nil, nil, fmt.Errorf("empty formula")
	}
	for _, src := range spec.Src {
		if unsupportedSrc[src] {
			return vm.Nil, nil, fmt.Errorf("unsupported src %d", src)
		}
	}
	body, err := transpileFormula(spec.Formula)
	if err != nil {
		return vm.Nil, nil, fmt.Errorf("transpile: %v", err)
	}
	closureProto, err := compileClosure(fmt.Sprintf("formula_%d", spec.ID), closureSource(body))
	if err != nil {
		return vm.Nil, nil, fmt.Errorf("compile: %v", err)
	}
	out, err := e.invokeLocked(fmt.Sprintf("formula_%d", spec.ID), func() ([]vm.Value, error) {
		return e.vm.Run(closureProto)
	})
	if err != nil {
		return vm.Nil, nil, fmt.Errorf("run: %v", err)
	}
	if len(out) == 0 || !out[0].IsCallable() {
		return vm.Nil, nil, fmt.Errorf("closure not callable")
	}

	maxParam := 0
	for n := range spec.Src {
		if n > maxParam {
			maxParam = n
		}
	}
	if maxParam > maxFormulaParam {
		maxParam = maxFormulaParam
	}
	binding := vm.NewEmptyTable()
	for n := 1; n <= maxParam; n++ {
		entry := vm.NewEmptyTable()
		entry.SetString("src", vm.NewInt(int64(spec.Src[n])))
		if prop, ok := spec.Prop[n]; ok {
			entry.SetString("prop", vm.NewString(prop))
		}
		binding.SetInt(n, vm.NewTable(entry))
	}
	return out[0], binding, nil
}

// compileClosure 解析编译一段 Lua 函数源。
func compileClosure(name, source string) (*compiler.Proto, error) {
	block, err := parser.Parse(name, source)
	if err != nil {
		return nil, err
	}
	return compiler.Compile(name, block)
}

// CalcHit 结算一次单击。第二返回值 false 表示引擎未接手（未装载/公式
// 不支持/运行期出错已禁用/脚本请求回退）——调用方走 legacy 掷骰。
func (e *Engine) CalcHit(in *HitInput) (HitResult, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vm == nil || e.broken || !e.calcFn.IsCallable() {
		return HitResult{}, false
	}
	if !e.supported[in.FormulaID] {
		return HitResult{}, false
	}
	if in.HurtCount <= 0 {
		in.HurtCount = 1
	}
	if in.CurTargetCount <= 0 {
		in.CurTargetCount = 1
	}
	ctxTable := vm.NewTable(buildCtxTable(in))
	out, err := e.invokeLocked("engine.calc_hit", func() ([]vm.Value, error) {
		return e.vm.ProtectedCall(e.calcFn, []vm.Value{ctxTable})
	})
	if err != nil {
		e.broken = true
		e.warnf("禁用公式引擎（calc_hit 出错：%v），伤害全部回退 legacy 掷骰", err)
		return HitResult{}, false
	}
	if len(out) == 0 || !out[0].IsTable() {
		return HitResult{}, false
	}
	result := out[0].AsTable()
	if flag := tableGet(result, "fallback"); flag.IsBool() && flag.AsBool() {
		return HitResult{}, false
	}
	damage := tableGet(result, "damage")
	if !damage.IsNumber() {
		return HitResult{}, false
	}
	return HitResult{
		Damage: int32(damage.AsInt()),
		Miss:   boolValue(result, "miss"),
		Crit:   boolValue(result, "crit"),
	}, true
}

// invokeLocked 套上超时 context、按调用重置的指令预算与 panic 恢复。
// 调用方持有 mu。
func (e *Engine) invokeLocked(label string, run func() ([]vm.Value, error)) (out []vm.Value, err error) {
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

// SkipSummary 返回跳过原因汇总（启动日志用），按条数降序。
func (e *Engine) SkipSummary() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.skipReason) == 0 {
		return ""
	}
	type pair struct {
		reason string
		count  int
	}
	pairs := make([]pair, 0, len(e.skipReason))
	for reason, count := range e.skipReason {
		pairs = append(pairs, pair{reason, count})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, fmt.Sprintf("%s×%d", p.reason, p.count))
	}
	return strings.Join(parts, ", ")
}

func tableGet(t vm.LuaTable, key string) vm.Value {
	return t.Get(vm.NewString(key))
}

func boolValue(t vm.LuaTable, key string) bool {
	v := tableGet(t, key)
	return v.IsBool() && v.AsBool()
}

func (e *Engine) warnf(format string, args ...any) {
	if e.logger == nil {
		return
	}
	e.logger("combatcalc: "+format, args...)
}
