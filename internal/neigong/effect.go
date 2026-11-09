package neigong

import (
	"github.com/iceisfun/golua/v2/vm"
	"github.com/local/9yin-go-server/internal/modulevm"
)

// Effect 是钩子返回指令白名单的 Go 形态。nil 字段/零值表示该指令未下发。
// 白名单外的键与畸形值在 convertEffect 处 WARN 丢弃，不打断战斗链（对齐
// 参考服 NeiGongEffects 契约；形态按 155 本仓内脚本实测勘定）。
type Effect struct {
	ExtraDamage *ExtraDamage   // extra_damage：并入当次落血的附伤
	ReducePct   float64        // reduce_pct：承伤折减系数 0..1（0=未下发）
	Shield      *Shield        // shield：护盾（FIFO 吸收，到期即弃）
	DodgeAdd    *DodgeAdd      // dodge_add：闪避加成窗（本仓扩展，料敌先机）
	Heal        int64          // heal：回复气血
	MPRestore   int64          // mp_restore：回复内力
	Reflect     *Reflect       // reflect：反打（数值或 {amount,kind} 双形态）
	ParryReduce *ParryReduce   // parry_reduce：格挡减伤拆分（解析占位）
	State       map[string]any // state：扁平状态回写，下次经 Ctx.State 传回
}

// ExtraDamage 是附加伤害指令（kind 现管线统一外功，透传）。
type ExtraDamage struct {
	Amount int64
	Kind   string
}

// DodgeAdd 是闪避加成窗指令（本仓扩展）：期间受击闪避 +Amount 个百分点，
// 持续 LifeMS。表达参考服脚本中的 TODO 机制"料敌先机"（buf_ng_yh_001）。
type DodgeAdd struct {
	Amount int64
	LifeMS int64
}

// Shield 是护盾指令。
type Shield struct {
	Amount int64
	LifeMS int64
}

// Reflect 是反打指令。
type Reflect struct {
	Amount int64
	Kind   string
}

// ParryReduce 是格挡减伤拆分（内功/外功分量；当前管线格挡零伤，解析占位）。
type ParryReduce struct {
	Internal float64
	External float64
}

// convertEffect 把钩子返回表按白名单转成 Effect。返回 nil 表示没有任何
// 有效指令（空表/全部被丢弃）。
func (e *Effects) convertEffect(buffID, event string, table vm.LuaTable) *Effect {
	effect := &Effect{}
	set := false
	key := vm.Nil
	for {
		nextKey, value, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		if !nextKey.IsString() {
			e.host.WarnOnce(buffID+":key", "neigong: %s: %s 返回非字符串键 %s，已丢弃", buffID, event, modulevm.ValueString(nextKey))
			continue
		}
		name := nextKey.AsString()
		switch name {
		case "extra_damage":
			if amount, kind, ok := e.convertAmountKind(buffID, name, value); ok {
				effect.ExtraDamage, set = &ExtraDamage{Amount: amount, Kind: kind}, true
			}
		case "reflect":
			if value.IsNumber() {
				if amount, ok := modulevm.LuaInt(value); ok && amount > 0 {
					effect.Reflect, set = &Reflect{Amount: amount}, true
					continue
				}
				e.host.WarnOnce(buffID+":"+name, "neigong: %s: %s reflect 数值 %s 非正/畸形，已丢弃", buffID, event, modulevm.ValueString(value))
				continue
			}
			if amount, kind, ok := e.convertAmountKind(buffID, name, value); ok {
				effect.Reflect, set = &Reflect{Amount: amount, Kind: kind}, true
			}
		case "reduce_pct":
			if value.IsNumber() {
				if pct := value.AsFloat(); pct >= 0 && pct <= 1 {
					effect.ReducePct, set = pct, true
					continue
				}
			}
			e.host.WarnOnce(buffID+":"+name, "neigong: %s: %s reduce_pct %s 越界 [0,1]，已丢弃", buffID, event, modulevm.ValueString(value))
		case "shield":
			if s, ok := e.convertShield(buffID, event, value); ok {
				effect.Shield, set = s, true
			}
		case "dodge_add":
			if d, ok := e.convertDodgeAdd(buffID, event, value); ok {
				effect.DodgeAdd, set = d, true
			}
		case "heal", "mp_restore":
			amount, ok := modulevm.LuaInt(value)
			if !ok || amount <= 0 {
				e.host.WarnOnce(buffID+":"+name, "neigong: %s: %s %s %s 非正/畸形，已丢弃", buffID, event, name, modulevm.ValueString(value))
				continue
			}
			if name == "heal" {
				effect.Heal = amount
			} else {
				effect.MPRestore = amount
			}
			set = true
		case "parry_reduce":
			if pr, ok := e.convertParryReduce(buffID, event, value); ok {
				effect.ParryReduce, set = pr, true
			}
		case "state":
			if value.IsTable() {
				effect.State = e.convertState(buffID, event, value.AsTable())
				set = true
			} else {
				e.host.WarnOnce(buffID+":"+name, "neigong: %s: %s state %s 非表，已丢弃", buffID, event, modulevm.ValueString(value))
			}
		default:
			e.host.WarnOnce(buffID+":"+name, "neigong: %s: %s 返回白名单外指令 %s=%s，已丢弃", buffID, event, name, modulevm.ValueString(value))
		}
	}
	if !set {
		return nil
	}
	return effect
}

// convertAmountKind 解析 {amount, kind?} 形态（extra_damage/reflect 共用）。
// amount 须为正数；kind 缺省空串。
func (e *Effects) convertAmountKind(buffID, name string, value vm.Value) (int64, string, bool) {
	table, ok := e.tableOf(buffID, name, value)
	if !ok {
		return 0, "", false
	}
	amountValue := modulevm.Field(table, "amount")
	amount, ok := modulevm.LuaInt(amountValue)
	if !ok || amount <= 0 {
		e.host.WarnOnce(buffID+":"+name, "neigong: %s: %s.amount %s 非正/畸形，整条丢弃", buffID, name, modulevm.ValueString(amountValue))
		return 0, "", false
	}
	return amount, modulevm.Field(table, "kind").AsString(), true
}

func (e *Effects) convertShield(buffID, event string, value vm.Value) (*Shield, bool) {
	table, ok := e.tableOf(buffID, "shield", value)
	if !ok {
		return nil, false
	}
	amount, amountOK := modulevm.LuaInt(modulevm.Field(table, "amount"))
	life, lifeOK := modulevm.LuaInt(modulevm.Field(table, "life_ms"))
	if !amountOK || amount <= 0 || !lifeOK || life <= 0 {
		e.host.WarnOnce(buffID+":shield", "neigong: %s: %s shield {amount=%s, life_ms=%s} 畸形，整条丢弃",
			buffID, event, modulevm.ValueString(modulevm.Field(table, "amount")), modulevm.ValueString(modulevm.Field(table, "life_ms")))
		return nil, false
	}
	return &Shield{Amount: amount, LifeMS: life}, true
}

// convertDodgeAdd 解析 dodge_add{amount, life_ms}（本仓扩展指令）。
func (e *Effects) convertDodgeAdd(buffID, event string, value vm.Value) (*DodgeAdd, bool) {
	table, ok := e.tableOf(buffID, "dodge_add", value)
	if !ok {
		return nil, false
	}
	amount, amountOK := modulevm.LuaInt(modulevm.Field(table, "amount"))
	life, lifeOK := modulevm.LuaInt(modulevm.Field(table, "life_ms"))
	if !amountOK || amount <= 0 || !lifeOK || life <= 0 {
		e.host.WarnOnce(buffID+":dodge_add", "neigong: %s: %s dodge_add {amount=%s, life_ms=%s} 畸形，整条丢弃",
			buffID, event, modulevm.ValueString(modulevm.Field(table, "amount")), modulevm.ValueString(modulevm.Field(table, "life_ms")))
		return nil, false
	}
	return &DodgeAdd{Amount: amount, LifeMS: life}, true
}

func (e *Effects) convertParryReduce(buffID, event string, value vm.Value) (*ParryReduce, bool) {
	table, ok := e.tableOf(buffID, "parry_reduce", value)
	if !ok {
		return nil, false
	}
	internal, internalOK := modulevm.LuaFloat(modulevm.Field(table, "internal"))
	external, externalOK := modulevm.LuaFloat(modulevm.Field(table, "external"))
	if !internalOK || !externalOK {
		e.host.WarnOnce(buffID+":parry_reduce", "neigong: %s: %s parry_reduce {internal=%s, external=%s} 畸形，整条丢弃",
			buffID, event, modulevm.ValueString(modulevm.Field(table, "internal")), modulevm.ValueString(modulevm.Field(table, "external")))
		return nil, false
	}
	return &ParryReduce{Internal: internal, External: external}, true
}

// convertState 转换扁平状态表（键任意、值限标量；脚本用它做互斥窗/层数/
// 冷却等会话内状态）。非标量条目 WARN 丢弃；空表合法（清窗）。
func (e *Effects) convertState(buffID, event string, table vm.LuaTable) map[string]any {
	out := make(map[string]any)
	key := vm.Nil
	for {
		nextKey, value, err := table.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		key = nextKey
		if !nextKey.IsString() {
			e.host.WarnOnce(buffID+":state-key", "neigong: %s: %s state 非字符串键 %s，已丢弃", buffID, event, modulevm.ValueString(nextKey))
			continue
		}
		switch {
		case value.IsInt():
			out[nextKey.AsString()] = value.AsInt()
		case value.IsFloat():
			out[nextKey.AsString()] = value.AsFloat()
		case value.IsString():
			out[nextKey.AsString()] = value.AsString()
		case value.IsBool():
			out[nextKey.AsString()] = value.AsBool()
		default:
			e.host.WarnOnce(buffID+":state:"+nextKey.AsString(), "neigong: %s: %s state[%s] 非标量（%s），已丢弃",
				buffID, event, nextKey.AsString(), value.Type())
		}
	}
	return out
}

// tableOf 校验值是表并取表，否则 WARN 一次。
func (e *Effects) tableOf(buffID, name string, value vm.Value) (vm.LuaTable, bool) {
	if value.IsTable() {
		return value.AsTable(), true
	}
	e.host.WarnOnce(buffID+":"+name, "neigong: %s: %s 非表（%s），整条丢弃", buffID, name, modulevm.ValueString(value))
	return nil, false
}
