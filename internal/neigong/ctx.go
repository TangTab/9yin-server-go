package neigong

import (
	"github.com/iceisfun/golua/v2/vm"
)

// SelfStats 是脚本可见的自身快照（ctx.self）。
type SelfStats struct {
	HP, MaxHP int64
	MP, MaxMP int64
	Attack    int64
}

// TargetInfo 是脚本可见的目标快照（ctx.target；nil 表示本次事件无目标）。
type TargetInfo struct {
	OID       uint32
	HP, MaxHP int64
}

// Ctx 是一次钩子调用的环境快照（只读，Go 侧预填）。字段面按 155 本仓内
// 脚本的实际消费集勘定：book_level/book_max_level/story_stage/now/damage/
// incoming/parried/crit/controlled/since_last_ms + self + target + attacker +
// state + rand.r_proc。随机数由 Go 预掷（r_proc），脚本不掷骰——可测试、
// 与参考服契约一致。
//
// 零值即合法：脚本侧各字段均有 or 兜底（如 ctx.book_level or 1），档位门
// tier_of 亦会把 0 层夹回 1。
type Ctx struct {
	BookLevel    int // 装备书修炼层数
	BookMaxLevel int // 书满级（层位换算分母）
	StoryStage   int // 全服九阴志阶段（档位门 cap = 3 + stage - 1）

	NowMS int64   // 服时钟（毫秒）；脚本 state 窗口用它算 until_at
	RProc float64 // 预掷_proc 随机数 [0,1)

	Damage     int64 // 本次事件造成的伤害（on_attack 侧）
	Incoming   int64 // 承受的伤害（on_victim/on_parry 侧，格挡下为格挡前值）
	Parried    bool  // 本次事件被目标格挡（on_attack 侧）
	Crit       bool  // 本次事件暴击
	Controlled bool  // 自身处于禁轻功/定身/昏迷类控制

	SinceLastMS int64 // 距上次有效触发的毫秒数；<0 表示未知/无上次触发

	Self        SelfStats
	Target      *TargetInfo    // nil = 无目标键
	AttackerOID uint32         // 0 = 无 attacker 键
	State       map[string]any // 上一次有效触发回写的扁平状态；nil = 不设 state 键
}

// buildCtxTable 构建传给钩子的只读 ctx 表。约定：可选段（target/attacker/
// state）缺失时不设键，脚本侧以 ctx.state or {} 等形式兜底。
func buildCtxTable(v *vm.VM, ctx *Ctx) *vm.Table {
	t := vm.NewEmptyTable()
	t.SetString("book_level", vm.NewInt(int64(ctx.BookLevel)))
	t.SetString("book_max_level", vm.NewInt(int64(ctx.BookMaxLevel)))
	t.SetString("story_stage", vm.NewInt(int64(ctx.StoryStage)))
	t.SetString("now", vm.NewInt(ctx.NowMS))
	t.SetString("damage", vm.NewInt(ctx.Damage))
	t.SetString("incoming", vm.NewInt(ctx.Incoming))
	t.SetString("parried", vm.NewBool(ctx.Parried))
	t.SetString("crit", vm.NewBool(ctx.Crit))
	t.SetString("controlled", vm.NewBool(ctx.Controlled))
	t.SetString("since_last_ms", vm.NewInt(ctx.SinceLastMS))

	self := vm.NewEmptyTable()
	self.SetString("hp", vm.NewInt(ctx.Self.HP))
	self.SetString("max_hp", vm.NewInt(ctx.Self.MaxHP))
	self.SetString("mp", vm.NewInt(ctx.Self.MP))
	self.SetString("max_mp", vm.NewInt(ctx.Self.MaxMP))
	self.SetString("attack", vm.NewInt(ctx.Self.Attack))
	t.SetString("self", vm.NewTable(self))

	if ctx.Target != nil {
		target := vm.NewEmptyTable()
		target.SetString("oid", vm.NewInt(int64(ctx.Target.OID)))
		target.SetString("hp", vm.NewInt(ctx.Target.HP))
		target.SetString("max_hp", vm.NewInt(ctx.Target.MaxHP))
		t.SetString("target", vm.NewTable(target))
	}
	if ctx.AttackerOID != 0 {
		attacker := vm.NewEmptyTable()
		attacker.SetString("oid", vm.NewInt(int64(ctx.AttackerOID)))
		t.SetString("attacker", vm.NewTable(attacker))
	}
	if ctx.State != nil {
		state := vm.NewEmptyTable()
		for key, value := range ctx.State {
			state.SetString(key, stateValue(value))
		}
		t.SetString("state", vm.NewTable(state))
	}

	rand := vm.NewEmptyTable()
	rand.SetString("r_proc", vm.NewFloat(ctx.RProc))
	t.SetString("rand", vm.NewTable(rand))
	return t
}

// stateValue 把 Go 侧状态值转回 Lua：支持 int64/float64/string/bool，其余
// 类型（装载期转换保证不会出现）落为 nil。
func stateValue(value any) vm.Value {
	switch v := value.(type) {
	case int64:
		return vm.NewInt(v)
	case int:
		return vm.NewInt(int64(v))
	case float64:
		return vm.NewFloat(v)
	case string:
		return vm.NewString(v)
	case bool:
		return vm.NewBool(v)
	default:
		return vm.Nil
	}
}
