// Package combatformula 装载伤害公式模块脚本（scripts/module/combat/formula.lua，
// 自参考服拷贝）：M.calc(c) 是伤害公式的权威实现（未命中 → 暴击（招架不免疫）
// → 招架穿透 → 平减 → 保底 1 点）。
//
// 宿主机制由 internal/modulevm 提供（坏脚本只禁用该模块、VM 不重建）。
// Calc 返回 nil（无脚本/坏脚本/缺 calc）时调用方应回退内置公式实现——
// 战斗管线落地后由其保证回退顺序，脚本层永不让出招链拒绝。
//
// 契约要点（formula.lua 文件头）：随机数由调用方预掷（c.rand[1]=未命中掷、
// c.rand[2]=暴击掷，脚本不掷骰——可测试、与参考服 Java 兜底逐位一致）；
// ratio ≤ 0 按 1.0。
package combatformula

import (
	"fmt"
	"time"

	"github.com/iceisfun/golua/v2/vm"
	"github.com/local/9yin-go-server/internal/modulevm"
)

// Logger 与标准库 log.Printf 兼容；宿主注入，包内不直接打日志。
type Logger = modulevm.Logger

// moduleName 是公式模块名（combat/formula.lua 的文件名主干）。
const moduleName = "formula"

// Input 是 M.calc 的只读输入 c。字段与 formula.lua 文件头一一对应；
// 全零值合法（守方侧属性缺省 0 的现状即如此）。
type Input struct {
	// 攻方
	Attack             float64 // 攻击力
	Defense            float64 // 守方外防（对数减伤曲线）
	Pierce             float64 // 穿刺（不吃外防折减）
	InternalConversion float64 // 化劲（招架穿透分母）
	// 命中
	Accuracy       float64
	Evasion        float64
	HitRateBonus   float64 // 命中率% 加成（直接抵扣未命中率）
	DodgeRateBonus float64 // 闪避率% 加成
	// 暴击
	CritValue           float64 // 暴击值
	Resilience          float64 // 守方坚韧
	CritRateBonus       float64
	CritRateReduction   float64
	CritDamageBonus     float64 // 暴伤% 加成
	CritDamageReduction float64 // 守方暴伤减免%
	// 守方其他
	DamageReduction float64 // 减免属性（每 10 点折 1 点平减）
	BlockRating     float64 // 格挡值
	// 技能与状态
	Ratio    float64 // 技能倍率（编译伤害/100/波数；<=0 按 1.0）
	Parrying bool    // 守方是否招架
	// 预掷随机数（脚本不掷骰）
	RMiss float64 // c.rand[1]
	RCrit float64 // c.rand[2]
}

// Output 是 M.calc 的返回（字段名与脚本一致）。
type Output struct {
	Damage       int64 // 落血值（完全招架为 0，否则保底 1）
	Miss         bool
	Crit         bool
	Parried      bool
	ReductionPct float64 // 外防减伤比例 0..0.75
}

// Formula 是公式模块宿主的域包装。零值不可用，经 New 创建。
type Formula struct {
	host *modulevm.Host
}

// New 创建公式宿主。dir 为空表示禁用：Calc 恒返回 nil。
func New(dir string, logger Logger) *Formula {
	return &Formula{host: modulevm.New(dir, logger)}
}

// SetTimeout 调整单次调用预算（测试用）。
func (f *Formula) SetTimeout(d time.Duration) { f.host.SetTimeout(d) }

// Enabled 报告宿主是否配置了脚本目录（-combat-scripts 非空）。
func (f *Formula) Enabled() bool { return f.host.Enabled() }

// Loaded 返回最近一次装载成功的模块数。
func (f *Formula) Loaded() int { return f.host.Loaded() }

// Count 是 Loaded 的别名，与目录型注册表的命名对齐。
func (f *Formula) Count() int { return f.Loaded() }

// Broken 返回被禁用的模块名（正常至多 ["formula"]）。
func (f *Formula) Broken() []string { return f.host.Broken() }

// Reload 重新装载脚本目录（GM 热重载）。
func (f *Formula) Reload() (int, error) { return f.Load() }

// Close 释放 VM。之后的 Calc 返回 nil。
func (f *Formula) Close() error { return f.host.Close() }

// Load 装载脚本目录并做域校验：formula 模块缺 calc 时禁用并返回错误。
func (f *Formula) Load() (int, error) {
	loaded, loadErr := f.host.Load()
	if !f.host.Has(moduleName) {
		if f.host.Enabled() && loadErr == nil {
			return loaded, fmt.Errorf("combatformula: %s module missing in directory", moduleName)
		}
		return loaded, loadErr
	}
	if !f.host.Func(moduleName, "calc").IsCallable() {
		f.host.MarkBroken(moduleName, "combatformula: calc 函数缺失")
		return loaded, fmt.Errorf("combatformula: %s.calc missing", moduleName)
	}
	return loaded, loadErr
}

// Calc 调用 M.calc。nil 表示脚本层不可用（无模块/已禁用/运行期出错——出错
// 时该模块被禁用，WARN 一次）或返回值畸形，调用方回退内置公式。
func (f *Formula) Calc(in *Input) *Output {
	if in == nil {
		in = &Input{}
	}
	result, ok := f.host.Invoke(moduleName, []string{"calc"}, func(v *vm.VM) vm.Value {
		return vm.NewTable(buildInputTable(v, in))
	})
	if !ok || !result.IsTable() {
		return nil
	}
	return convertOutput(result.AsTable())
}

// buildInputTable 构建只读输入表 c（字段名与 formula.lua 文件头一致）。
func buildInputTable(v *vm.VM, in *Input) *vm.Table {
	c := vm.NewEmptyTable()
	c.SetString("attack", vm.NewFloat(in.Attack))
	c.SetString("defense", vm.NewFloat(in.Defense))
	c.SetString("pierce", vm.NewFloat(in.Pierce))
	c.SetString("internal_conversion", vm.NewFloat(in.InternalConversion))
	c.SetString("accuracy", vm.NewFloat(in.Accuracy))
	c.SetString("evasion", vm.NewFloat(in.Evasion))
	c.SetString("hit_rate_bonus", vm.NewFloat(in.HitRateBonus))
	c.SetString("dodge_rate_bonus", vm.NewFloat(in.DodgeRateBonus))
	c.SetString("crit_value", vm.NewFloat(in.CritValue))
	c.SetString("resilience", vm.NewFloat(in.Resilience))
	c.SetString("crit_rate_bonus", vm.NewFloat(in.CritRateBonus))
	c.SetString("crit_rate_reduction", vm.NewFloat(in.CritRateReduction))
	c.SetString("crit_damage_bonus", vm.NewFloat(in.CritDamageBonus))
	c.SetString("crit_damage_reduction", vm.NewFloat(in.CritDamageReduction))
	c.SetString("damage_reduction", vm.NewFloat(in.DamageReduction))
	c.SetString("block_rating", vm.NewFloat(in.BlockRating))
	c.SetString("ratio", vm.NewFloat(in.Ratio))
	c.SetString("parrying", vm.NewBool(in.Parrying))
	rand := vm.NewEmptyTable()
	rand.SetInt(1, vm.NewFloat(in.RMiss))
	rand.SetInt(2, vm.NewFloat(in.RCrit))
	c.SetString("rand", vm.NewTable(rand))
	return c
}

// convertOutput 转换返回表；damage/miss/crit/parried 任一畸形视为整体不可用
// （返回 nil，公式契约字段是全量的）。
func convertOutput(table vm.LuaTable) *Output {
	damage, damageOK := modulevm.LuaInt(modulevm.Field(table, "damage"))
	miss := modulevm.Field(table, "miss")
	crit := modulevm.Field(table, "crit")
	parried := modulevm.Field(table, "parried")
	if !damageOK || !miss.IsBool() || !crit.IsBool() || !parried.IsBool() {
		return nil
	}
	reduction := 0.0
	if value := modulevm.Field(table, "reduction_pct"); value.IsNumber() {
		reduction = value.AsFloat()
	}
	return &Output{
		Damage:       damage,
		Miss:         miss.AsBool(),
		Crit:         crit.AsBool(),
		Parried:      parried.AsBool(),
		ReductionPct: reduction,
	}
}
