package combatcalc

import (
	"github.com/iceisfun/golua/v2/vm"
)

// HitInput 是一次单击伤害结算的输入快照（Go 侧预填，脚本不掷骰——与
// internal/neigong、参考服 formula.lua 契约一致）。
type HitInput struct {
	FormulaID      int32             // damage_calculate r 行第 1 段（formula.ini 段号）
	Params         []string          // r 行第 3 段起原始串（P1..Pn 按位对应）
	HurtCount      int               // 段数（zhaoshi 判定帧数 > varprop HitCount > 正 r 段数）
	CurTargetCount int               // 本次出招目标快照数
	Props          map[string]int64  // 攻方属性快照（MeleePower/Str/MinMeleeDamage…）
	Vitals         map[string]int64  // 攻方生存值（HP/MaxHP/MP/MaxMP）
	CritValue      int64             // 攻方暴击值（PhyDamage→PhyVa，内功系→MagicVa）
	Accuracy       int64             // 攻方命中值（PhyDamage→PhyHit，内功系→MagicHit）
	Defender       *DefenderStats    // 守方快照；nil＝零值（NPC 属性块迁移前）
	MissRoll       float64           // 预掷 [1]：未命中判定
	CritRoll       float64           // 预掷 [2]：暴击判定
	ExtraRolls     []float64         // 预掷 [3..]：R[a,b] 与 Src=6 区间按求值顺序消费
}

// DefenderStats 是守方对抗快照（§十六 4–7 条管线输入）。NPC 侧属性数据
// 源（官方 base block 0..98 迁移）就位前调用方传 nil，引擎按零值结算——
// 与参考服"守方侧缺省"口径一致，但管线已铺好，迁移后只改数据不改公式。
type DefenderStats struct {
	Defense   int64 // 对应伤害系防御（外功→PhyDef，阴/阳/刚/柔→分系 Def）
	Evasion   int64 // 闪避值（missChance 输入）
	Resilience int64 // 封劲（critChance 输入）
}

// HitResult 是 M.calc_hit 的结算产物。
type HitResult struct {
	Damage int32
	Miss   bool
	Crit   bool
}

// randExtraMinimum 是 Go 侧保证的区间掷随机数下限——公式内的 R[] 与 Src=6
// 消费超出时引擎复用末位（确定性可测）。
const randExtraMinimum = 6

// buildCtxTable 构建传给 M.calc_hit 的只读 ctx 表。字段面与 formula.lua
// 引擎约定一一对应：formula_id/params/hurt_count/cur_target_count/props/
// vitals/rand（[1]=miss、[2]=crit、[3..]=区间掷序列）。
func buildCtxTable(in *HitInput) *vm.Table {
	t := vm.NewEmptyTable()
	t.SetString("formula_id", vm.NewInt(int64(in.FormulaID)))
	t.SetString("hurt_count", vm.NewInt(int64(in.HurtCount)))
	t.SetString("cur_target_count", vm.NewInt(int64(in.CurTargetCount)))
	t.SetString("crit_value", vm.NewInt(in.CritValue))
	t.SetString("accuracy", vm.NewInt(in.Accuracy))
	if in.Defender != nil {
		defender := vm.NewEmptyTable()
		defender.SetString("defense", vm.NewInt(in.Defender.Defense))
		defender.SetString("evasion", vm.NewInt(in.Defender.Evasion))
		defender.SetString("resilience", vm.NewInt(in.Defender.Resilience))
		t.SetString("defender", vm.NewTable(defender))
	}

	params := vm.NewEmptyTable()
	for i, raw := range in.Params {
		params.SetInt(i+1, vm.NewString(raw))
	}
	t.SetString("params", vm.NewTable(params))

	props := vm.NewEmptyTable()
	for key, value := range in.Props {
		props.SetString(key, vm.NewInt(value))
	}
	t.SetString("props", vm.NewTable(props))

	vitals := vm.NewEmptyTable()
	for key, value := range in.Vitals {
		vitals.SetString(key, vm.NewInt(value))
	}
	t.SetString("vitals", vm.NewTable(vitals))

	rand := vm.NewEmptyTable()
	rand.SetInt(1, vm.NewFloat(in.MissRoll))
	rand.SetInt(2, vm.NewFloat(in.CritRoll))
	for i := 0; i < randExtraMinimum; i++ {
		roll := 0.5
		if i < len(in.ExtraRolls) {
			roll = in.ExtraRolls[i]
		}
		rand.SetInt(i+3, vm.NewFloat(roll))
	}
	t.SetString("rand", vm.NewTable(rand))
	return t
}
