package main

// 伤害公式引擎（Java script/CombatFormulas.java 逐行移植；Lua formula.lua
// 是权威，Java 兜底与本文件同输入必同输出）。
//
// 2026-11-07 战斗重写引入：此前 Go 把 damage_calculate.ini 的 r 段静态和
// 当作固定伤害直扣（无掷骰/无未命中/无暴击）。官方口径（Java rollDamage
// + 抓包 52905 实证：同招各段伤害相同、异招不同、有 miss）：
//
//	attack = 攻方面板区间掷值（空手 20-35 基础 + 装备外功 + 内功
//	         Min/MaxMeleeDamage；Go 装备/内功外功分量未移植，暂取基础区间）
//	ratio  = baseDamage / 100 / damageWaves（skill 编译口径）
//	miss   = 10% + 15%×(闪避−命中)/(闪避+1)，夹 24% 上限（守方缺省 0 → 10%）
//	crit   = 5% 基础，夹 15%；倍数 = 1 + max(0, 1.0 + 暴伤% − 暴减%)（缺省 ×2）
//	伤害   = floor(attack × (1−减伤) × ratio)，保底 1 点
//
// 掷骰顺序：miss → crit → 区间掷值（前两者不依赖区间跨度，同种子稳定）。
// 守方（NPC）侧防御/命中/闪避/暴抗当前全部缺省 0（Java rollDamage 同款
// 传参），外防对数减伤分支保留实现，接入守方面板后即可启用。

import (
	"math"
	"math/rand"
)

// 公式系数常量（formula.lua 一一对应，调参须两处同步）。
const (
	formulaDefenseSlope     = 20.0
	formulaDefenseIntercept = -44.0
	formulaDefenseLogScale  = 100.0
	formulaMaxReduction     = 0.75
	formulaCritBase         = 0.05
	formulaCritSlope        = 0.10
	formulaMaxBaseCrit      = 0.15
	formulaBaseCritDamage   = 1.0
	formulaMissBase         = 0.10
	formulaMissSlope        = 0.15
	formulaMaxDodge         = 0.24
	formulaReductionPerDmg  = 10
)

// combatFormulaInput 是公式入参（Java CombatFormulas.Input 同构）。当前
// 战斗链只填 attack/ratio 与两枚掷骰值；防御侧字段留给守方面板接入。
type combatFormulaInput struct {
	attack         float64
	defense        float64
	pierce         float64
	accuracy       float64
	evasion        float64
	hitRateBonus   float64
	dodgeRateBonus float64
	critValue      float64
	resilience     float64
	ratio          float64
	parrying       bool
	missRoll       float64
	critRoll       float64
}

// combatFormulaResult 对齐 Java Result：damage=0 且 miss=true 表示未命中
// （整段不发帧）。
type combatFormulaResult struct {
	damage       int32
	miss         bool
	crit         bool
	parried      bool
	reductionPct float64
}

func combatFormulasCalc(c combatFormulaInput) combatFormulaResult {
	miss := combatMissChance(c)
	if miss > 0 && c.missRoll < miss {
		return combatFormulaResult{damage: 0, miss: true, reductionPct: combatDefenseReduction(c.defense)}
	}
	reduction := combatDefenseReduction(c.defense)
	ratio := c.ratio
	if ratio <= 0 {
		ratio = 1.0
	}
	// Java：raw = (attack × (1−减伤) + pierce) × ratio（pierce 当前缺省 0）。
	raw := (c.attack * (1 - reduction)) * ratio
	crit := false
	if chance := combatCritChance(c); chance > 0 && c.critRoll < chance {
		raw *= combatCritMultiplier()
		crit = true
	}
	parried := false
	if c.parrying {
		parried = true
	}
	damage := int32(math.Floor(raw))
	if damage < 1 {
		damage = 1
	}
	return combatFormulaResult{damage: damage, crit: crit, parried: parried, reductionPct: reduction}
}

// combatDefenseReduction：外防 → 减伤比例 [0, 0.75]；外防 ≤ 0 不减伤。
func combatDefenseReduction(defense float64) float64 {
	if defense <= 0 {
		return 0
	}
	pct := (formulaDefenseSlope*math.Log(defense/formulaDefenseLogScale) + formulaDefenseIntercept) / 100.0
	return clampFloat64(pct, 0, formulaMaxReduction)
}

// combatMissChance：10% + 15%×(闪避−命中)/(闪避+1)，合计夹 24% 后抵扣命中率。
func combatMissChance(c combatFormulaInput) float64 {
	ratio := 0.0
	if denominator := c.evasion + 1; denominator != 0 {
		ratio = (c.evasion - c.accuracy) / denominator
	}
	dodge := math.Min(formulaMaxDodge, formulaMissBase+formulaMissSlope*ratio+c.dodgeRateBonus)
	return clampFloat64(dodge-c.hitRateBonus, 0, 1)
}

// combatCritChance：5% + 10%×(暴击值−坚韧)/(暴击值+1)，夹 15% 基础上限。
func combatCritChance(c combatFormulaInput) float64 {
	ratio := 0.0
	if denominator := c.critValue + 1; denominator != 0 {
		ratio = (c.critValue - c.resilience) / denominator
	}
	base := clampFloat64(formulaCritBase+formulaCritSlope*ratio, 0, formulaMaxBaseCrit)
	return clampFloat64(base, 0, 1)
}

// combatCritMultiplier：1 + max(0, 基础 100% + 暴伤% − 暴减%)（缺省 2.0）。
func combatCritMultiplier() float64 {
	return 1 + math.Max(0, formulaBaseCritDamage)
}

func clampFloat64(value, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, value))
}

// combatRandomFloat64 是掷骰随机源（包级变量便于测试确定性注入）。
var combatRandomFloat64 = rand.Float64

func randFloat64() float64 { return combatRandomFloat64() }

// combatRoll 是一次掷骰产物（Java RollResult 同构）：damage=0 即未命中。
type combatRoll struct {
	damage int32
	crit   bool
}

// rollSkillDamage 出招掷值（Java SkillCastHandler.rollDamage 同序）：
// missRoll/critRoll 先掷，攻方区间掷值后掷。miss → damage=0（发帧阶段
// 整段跳过）；命中保底 1 点。
func rollSkillDamage(attackMin, attackMax int32, ratio float64) combatRoll {
	missRoll := randFloat64()
	critRoll := randFloat64()
	attack := float64(attackMin)
	if attackMax > attackMin {
		attack += math.Floor(randFloat64() * float64(attackMax-attackMin+1))
	}
	result := combatFormulasCalc(combatFormulaInput{
		attack: attack, ratio: ratio,
		missRoll: missRoll, critRoll: critRoll,
	})
	if result.miss {
		return combatRoll{}
	}
	return combatRoll{damage: result.damage, crit: result.crit}
}
