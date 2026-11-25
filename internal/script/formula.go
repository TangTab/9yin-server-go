package script

// formula.go 伤害公式引擎：把公式（数据层转译的 Lua 函数体）装成闭包注入
// 内置引擎脚本，战斗链每击×每目标调用一次 M.calc_hit。
//
// 错误策略（2433 线 internal/combatcalc 同款）：装载期坏公式跳过并计数
// （调用该公式的击走回退面）；运行期引擎出错只禁用引擎脚本一次，之后全部
// 调用返回 Fallback——服务器永不因脚本层停摆。

import (
	_ "embed"
	"math"
	"sync/atomic"

	lua "github.com/yuin/gopher-lua"
)

//go:embed lua/combat_formula.lua
var engineSource string

// engineKey 引擎脚本的句柄键（内置，不走脚本根）。
const engineKey = "__combat_formula__"

// FormulaParam 公式第 n 个参数（P<n>）的来源绑定（来源码见 skilldata 的
// Src* 常量；本包只透传）。
type FormulaParam struct {
	Source int
	Prop   string
}

// FormulaSpec 一条可装载公式。
type FormulaSpec struct {
	ID     int
	Body   string // Lua 函数体（表达式转译产物；调用环境 PV/R/Q/MIN/MAX）
	Params [12]FormulaParam
}

// HitContext 每击求值输入（宿主预填；随机数由宿主预掷，脚本不掷骰）。
type HitContext struct {
	FormulaID       int
	Params          []string // 伤害计算行的位置实参
	HurtCount       int      // 分母击数（HurtCount）
	TargetCount     int      // 当前目标数（CurTargetCount）
	SkillLevel      int
	Attacker        map[string]float64 // 攻击者属性
	Target          map[string]float64 // 目标属性
	Distance        float64
	SelfBuffLevel   func(configID string) int
	TargetBuffLevel func(configID string) int
	NamedVar        func(name string) float64
	// 对抗值：攻方命中/暴击值按伤害系预选；守方防御/闪避/坚韧。
	Accuracy, CritValue          float64
	Defense, Evasion, Resilience float64
	Parrying                     bool
	Rand                         []float64 // [0,1) 预掷序列：[1] 未命中、[2] 暴击、其后区间掷值
}

// HitResult 每击结算结果。
type HitResult struct {
	Damage   float64
	Miss     bool
	Crit     bool
	Parried  bool
	Fallback bool // 公式缺席/不可求值——调用方走回退面
}

// Engine 公式引擎。
type Engine struct {
	vm          *VM
	calc        *Func
	installed   int
	unsupported int
	broken      atomic.Bool
}

// NewEngine 装载引擎脚本并注入公式闭包。
func NewEngine(vm *VM, specs []FormulaSpec) *Engine {
	e := &Engine{vm: vm}
	if vm == nil {
		return e
	}
	vm.lock()
	defer vm.unlock()
	if err := vm.runSource(sourceID(globalEnv, engineKey), engineKey, []byte(engineSource), nil); err != nil {
		return e
	}
	module, ok := vm.l.GetGlobal("__nx_combat").(*lua.LTable)
	if !ok {
		return e
	}
	L := vm.l
	formulas, bindings := L.NewTable(), L.NewTable()
	for _, spec := range specs {
		if spec.Body == "" {
			e.unsupported++
			continue
		}
		fn, err := L.LoadString("return function(PV, R, Q, MIN, MAX) return " + spec.Body + " end")
		if err != nil {
			e.unsupported++
			continue
		}
		L.Push(fn)
		if err := L.PCall(0, 1, nil); err != nil {
			e.unsupported++
			L.SetTop(0)
			continue
		}
		closure := L.Get(-1)
		L.SetTop(0)
		key := lua.LNumber(spec.ID)
		formulas.RawSet(key, closure)
		bind := L.NewTable()
		for n, param := range spec.Params {
			if param.Source == 0 && param.Prop == "" {
				continue
			}
			row := L.NewTable()
			row.RawSetString("src", lua.LNumber(param.Source))
			row.RawSetString("prop", lua.LString(param.Prop))
			bind.RawSetInt(n+1, row)
		}
		bindings.RawSet(key, bind)
		e.installed++
	}
	module.RawSetString("__formulas", formulas)
	module.RawSetString("__bindings", bindings)
	install, _ := module.RawGetString("install").(*lua.LFunction)
	if install == nil {
		return e
	}
	L.Push(install)
	L.Push(formulas)
	L.Push(bindings)
	if err := L.PCall(2, 1, nil); err != nil {
		L.SetTop(0)
		return e
	}
	L.SetTop(0)
	if fn := vm.l.GetGlobal("__nx_combat"); fn != lua.LNil {
		module = fn.(*lua.LTable)
		if calc, ok := module.RawGetString("calc_hit").(*lua.LFunction); ok {
			e.calc = &Func{vm: vm, key: engineKey, id: sourceID(globalEnv, engineKey), fn: calc}
		}
	}
	return e
}

// Installed 已装载公式数 / 装载期跳过数。
func (e *Engine) Installed() (int, int) { return e.installed, e.unsupported }

// Enabled 引擎可用（已装载且未因运行期出错禁用）。
func (e *Engine) Enabled() bool { return e != nil && e.calc != nil && !e.broken.Load() }

// CalcHit 每击结算；引擎不可用或出错返回 Fallback。
func (e *Engine) CalcHit(ctx HitContext) HitResult {
	if !e.Enabled() {
		return HitResult{Fallback: true}
	}
	calc := e.calc
	calc.vm.lock()
	out, err := calc.vm.callTable(calc, hitTable(calc.vm.l, ctx))
	calc.vm.unlock()
	if err != nil {
		if !e.broken.Swap(true) {
			e.calc.vm.trace("script: 伤害公式引擎运行期出错，引擎已禁用: %v", err)
		}
		return HitResult{Fallback: true}
	}
	return out
}

// Settle 只走结算管线（未命中 → 取整 → 暴击 → 招架 → 保底）：原值来自
// 技能脚本的 get_damage 时用。引擎不可用返回 Fallback。
func (e *Engine) Settle(ctx HitContext, raw float64) HitResult {
	if !e.Enabled() {
		return HitResult{Fallback: true, Damage: raw}
	}
	calc := e.calc
	calc.vm.lock()
	defer calc.vm.unlock()
	settle, _ := calc.vm.l.GetGlobal("__nx_combat").(*lua.LTable)
	if settle == nil {
		return HitResult{Fallback: true, Damage: raw}
	}
	fn, ok := settle.RawGetString("settle_hit").(*lua.LFunction)
	if !ok {
		return HitResult{Fallback: true, Damage: raw}
	}
	result, err := calc.vm.callTable(&Func{vm: calc.vm, key: engineKey, id: sourceID(globalEnv, engineKey), fn: fn}, hitTable(calc.vm.l, ctx), lua.LNumber(raw))
	if err != nil {
		e.broken.Store(true)
		return HitResult{Fallback: true, Damage: raw}
	}
	return result
}

// callTable 调用函数（首参为上下文表，extra 为其余实参），结果表 →
// HitResult。
func (vm *VM) callTable(f *Func, table *lua.LTable, extra ...lua.LValue) (HitResult, error) {
	var result HitResult
	err := f.callUnlocked(func(L *lua.LState) {
		L.Push(table)
		for _, v := range extra {
			L.Push(v)
		}
	}, 1+len(extra), func(L *lua.LState, v lua.LValue) {
		row, ok := v.(*lua.LTable)
		if !ok {
			result.Fallback = true
			return
		}
		if flag(L, row, "fallback") {
			result.Fallback = true
			return
		}
		result.Damage = number(L, row, "damage")
		result.Miss, result.Crit, result.Parried = flag(L, row, "miss"), flag(L, row, "crit"), flag(L, row, "parried")
	})
	return result, err
}

// hitTable HitContext → Lua 表。
func hitTable(L *lua.LState, ctx HitContext) *lua.LTable {
	t := L.NewTable()
	t.RawSetString("formula_id", lua.LNumber(ctx.FormulaID))
	t.RawSetString("hurt_count", lua.LNumber(ctx.HurtCount))
	t.RawSetString("cur_target_count", lua.LNumber(ctx.TargetCount))
	t.RawSetString("skill_level", lua.LNumber(ctx.SkillLevel))
	t.RawSetString("distance", lua.LNumber(ctx.Distance))
	t.RawSetString("accuracy", lua.LNumber(ctx.Accuracy))
	t.RawSetString("crit_value", lua.LNumber(ctx.CritValue))
	t.RawSetString("parrying", lua.LBool(ctx.Parrying))
	if len(ctx.Params) > 0 {
		params := L.NewTable()
		for i, p := range ctx.Params {
			params.RawSetInt(i+1, lua.LString(p))
		}
		t.RawSetString("params", params)
	}
	if len(ctx.Rand) > 0 {
		rand := L.NewTable()
		for i, r := range ctx.Rand {
			rand.RawSetInt(i+1, lua.LNumber(r))
		}
		t.RawSetString("rand", rand)
	}
	t.RawSetString("props", numberTable(L, ctx.Attacker))
	t.RawSetString("target", numberTable(L, ctx.Target))
	defender := L.NewTable()
	defender.RawSetString("defense", lua.LNumber(ctx.Defense))
	defender.RawSetString("evasion", lua.LNumber(ctx.Evasion))
	defender.RawSetString("resilience", lua.LNumber(ctx.Resilience))
	t.RawSetString("defender", defender)
	if ctx.SelfBuffLevel != nil {
		t.RawSetString("self_buff", L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LNumber(ctx.SelfBuffLevel(L.CheckString(1))))
			return 1
		}))
	}
	if ctx.TargetBuffLevel != nil {
		t.RawSetString("target_buff", L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LNumber(ctx.TargetBuffLevel(L.CheckString(1))))
			return 1
		}))
	}
	if ctx.NamedVar != nil {
		t.RawSetString("named", L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LNumber(ctx.NamedVar(L.CheckString(1))))
			return 1
		}))
	}
	return t
}

func numberTable(L *lua.LState, props map[string]float64) *lua.LTable {
	t := L.NewTable()
	for name, value := range props {
		if !math.IsNaN(value) && !math.IsInf(value, 0) {
			t.RawSetString(name, lua.LNumber(value))
		}
	}
	return t
}

func number(L *lua.LState, row *lua.LTable, key string) float64 {
	if v, ok := row.RawGetString(key).(lua.LNumber); ok {
		return float64(v)
	}
	return 0
}

func flag(L *lua.LState, row *lua.LTable, key string) bool {
	v, _ := row.RawGetString(key).(lua.LBool)
	return bool(v)
}
