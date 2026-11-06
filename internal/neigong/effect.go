package neigong

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/iceisfun/golua/v2/vm"
)

// Effect 是钩子返回指令白名单的 Go 形态。nil 字段/零值表示该指令未下发。
// 白名单外的键与畸形值在 convertEffect 处 WARN 丢弃，不打断战斗链（对齐
// 参考服 NeiGongEffects 契约；形态按 155 本仓内脚本实测勘定）。
type Effect struct {
	ExtraDamage *ExtraDamage // extra_damage：并入当次落血的附伤
	ReducePct   float64      // reduce_pct：承伤折减系数 0..1（0=未下发）
	Shield      *Shield      // shield：护盾（FIFO 吸收，到期即弃）
	Heal        int64        // heal：回复气血
	MPRestore   int64        // mp_restore：回复内力
	Reflect     *Reflect     // reflect：反打（数值或 {amount,kind} 双形态）
	ParryReduce *ParryReduce // parry_reduce：格挡减伤拆分（解析占位）
	State       map[string]any // state：扁平状态回写，下次经 Ctx.State 传回
}

// ExtraDamage 是附加伤害指令（kind 现管线统一外功，透传）。
type ExtraDamage struct {
	Amount int64
	Kind   string
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
			e.warnOnce(buffID+":key", "%s: %s 返回非字符串键 %s，已丢弃", buffID, event, valueString(nextKey))
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
				if amount, ok := luaInt(value); ok && amount > 0 {
					effect.Reflect, set = &Reflect{Amount: amount}, true
					continue
				}
				e.warnOnce(buffID+":"+name, "%s: %s reflect 数值 %s 非正/畸形，已丢弃", buffID, event, valueString(value))
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
			e.warnOnce(buffID+":"+name, "%s: %s reduce_pct %s 越界 [0,1]，已丢弃", buffID, event, valueString(value))
		case "shield":
			if s, ok := e.convertShield(buffID, event, value); ok {
				effect.Shield, set = s, true
			}
		case "heal", "mp_restore":
			amount, ok := luaInt(value)
			if !ok || amount <= 0 {
				e.warnOnce(buffID+":"+name, "%s: %s %s %s 非正/畸形，已丢弃", buffID, event, name, valueString(value))
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
				e.warnOnce(buffID+":"+name, "%s: %s state %s 非表，已丢弃", buffID, event, valueString(value))
			}
		default:
			e.warnOnce(buffID+":"+name, "%s: %s 返回白名单外指令 %s=%s，已丢弃", buffID, event, name, valueString(value))
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
	amountValue := tableGet(table, "amount")
	amount, ok := luaInt(amountValue)
	if !ok || amount <= 0 {
		e.warnOnce(buffID+":"+name, "%s: %s.amount %s 非正/畸形，整条丢弃", buffID, name, valueString(amountValue))
		return 0, "", false
	}
	return amount, tableGet(table, "kind").AsString(), true
}

func (e *Effects) convertShield(buffID, event string, value vm.Value) (*Shield, bool) {
	table, ok := e.tableOf(buffID, "shield", value)
	if !ok {
		return nil, false
	}
	amount, amountOK := luaInt(tableGet(table, "amount"))
	life, lifeOK := luaInt(tableGet(table, "life_ms"))
	if !amountOK || amount <= 0 || !lifeOK || life <= 0 {
		e.warnOnce(buffID+":shield", "%s: %s shield {amount=%s, life_ms=%s} 畸形，整条丢弃",
			buffID, event, valueString(tableGet(table, "amount")), valueString(tableGet(table, "life_ms")))
		return nil, false
	}
	return &Shield{Amount: amount, LifeMS: life}, true
}

func (e *Effects) convertParryReduce(buffID, event string, value vm.Value) (*ParryReduce, bool) {
	table, ok := e.tableOf(buffID, "parry_reduce", value)
	if !ok {
		return nil, false
	}
	internal, internalOK := luaFloat(tableGet(table, "internal"))
	external, externalOK := luaFloat(tableGet(table, "external"))
	if !internalOK || !externalOK {
		e.warnOnce(buffID+":parry_reduce", "%s: %s parry_reduce {internal=%s, external=%s} 畸形，整条丢弃",
			buffID, event, valueString(tableGet(table, "internal")), valueString(tableGet(table, "external")))
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
			e.warnOnce(buffID+":state-key", "%s: %s state 非字符串键 %s，已丢弃", buffID, event, valueString(nextKey))
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
			e.warnOnce(buffID+":state:"+nextKey.AsString(), "%s: %s state[%s] 非标量（%s），已丢弃",
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
	e.warnOnce(buffID+":"+name, "%s: %s 非表（%s），整条丢弃", buffID, name, valueString(value))
	return nil, false
}

// luaInt 把 Lua 数值安全转 int64（NaN/Inf 拒绝；浮点取最近整数）。
func luaInt(v vm.Value) (int64, bool) {
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

// luaFloat 把 Lua 数值安全转 float64（NaN/Inf 拒绝）。
func luaFloat(v vm.Value) (float64, bool) {
	if !v.IsNumber() {
		return 0, false
	}
	f := v.AsFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// valueString 把 Lua 值序列化为可读文本（仅用于 WARN 日志；表浅层展开）。
func valueString(v vm.Value) string {
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
				parts = append(parts, valueString(nextKey)+"="+valueString(value))
				key = nextKey
			}
			return "{" + strings.Join(parts, ", ") + "}"
		}
		return "table"
	default:
		return fmt.Sprintf("%s(%s)", v.Type(), v.String())
	}
}
