// Package sim 以 ECS 组件的形式持有场景内 NPC 的服务器权威
// 模拟态。它是 sceneLifecycle 原先两张平行 map（combatActors/
// combatStates）的替代存储；协议编解码仍属于调用方，本包
// 不感知线上格式。
package sim

import (
	"time"

	"github.com/local/9yin-go-server/internal/entity"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// Vitals 是角色战斗数值的组件化形态，玩家与 NPC 共用同一份
// 原语逻辑。NPC 经 ECS 世界存储；玩家作为 playerActor 的值
// 字段由其互斥锁保护。
type Vitals struct {
	HP        int32
	HitHP     int32
	BaseMaxHP int32
	MaxHP     int32
	MP        int32
	BaseMaxMP int32
	MaxMP     int32
	Logic     uint8
}

// NewVitals 按 entity.NewActor 的默认值构造满状态的 NPC 数值。
// maxHP/maxMP 的钳制规则与 Actor 保持一致。
func NewVitals(maxHP, maxMP int32) Vitals {
	if maxHP < 1 {
		maxHP = 1
	}
	if maxMP < 1 {
		maxMP = 1
	}
	return Vitals{
		HP: maxHP, HitHP: maxHP, BaseMaxHP: maxHP, MaxHP: maxHP,
		MP: maxMP, BaseMaxMP: maxMP, MaxMP: maxMP,
		Logic: entity.LogicStateNormal,
	}
}

// ApplyDamage 与 entity.Actor.ApplyDamage 的语义完全一致：
// 瘀血层簿记、死亡状态迁移与"死亡后不再受伤"的闸门。
// 状态发生变化时返回 true。
func (v *Vitals) ApplyDamage(amount int32) bool {
	if amount <= 0 {
		return false
	}
	if v.HP == 0 {
		return false
	}
	if v.HitHP < v.HP {
		v.HitHP = v.HP
	}
	v.HP -= amount
	if v.HP <= 0 {
		v.HP = 0
		v.HitHP = 0
		v.Logic = entity.LogicStateDied
	} else {
		v.Logic = entity.LogicStateFighting
	}
	return true
}

// Revive 是最小化的权威复活迁移（Actor.Revive 的 NPC 子集：
// NPC 没有轻功条与速度比）。玩家侧的完整复活（轻功回满、速度
// 比重置）由 playerActor 跨组件编排。
func (v *Vitals) Revive() {
	v.HP = v.MaxHP
	v.HitHP = v.MaxHP
	v.MP = v.MaxMP
	v.Logic = entity.LogicStateNormal
}

// SetLogicState 只改变当前状态；HP 和 MP 仍由同一份 Vitals 持有，
// 并由发布方一并进行同步。
func (v *Vitals) SetLogicState(state uint8) {
	v.Logic = state
}

// ApplyHeal 恢复 HP，上限为 MaxHP。治疗有意不清除死亡状态：
// 复活必须显式完成那次状态迁移。与 entity.Actor.ApplyHeal 逐行等价。
func (v *Vitals) ApplyHeal(amount int32) bool {
	if amount <= 0 {
		return false
	}
	if v.Logic == entity.LogicStateDied || v.HP >= v.MaxHP {
		return false
	}
	before := v.HP
	v.HP += amount
	if v.HP > v.MaxHP {
		v.HP = v.MaxHP
	}
	if v.HitHP < v.HP {
		v.HitHP = v.HP
	}
	return v.HP != before
}

// RecoverBruisedHPWhileFighting 只结算战斗期间的瘀血层。
// HitHP 是受击前/第二根血条，让它向 HP 回消散可见瘀血而不给
// 真实 HP。其他 LogicState 下有意不做任何事。
func (v *Vitals) RecoverBruisedHPWhileFighting(amount int32) bool {
	if amount <= 0 {
		return false
	}
	if v.Logic != entity.LogicStateFighting || v.HP <= 0 || v.HitHP <= v.HP {
		return false
	}
	v.HitHP -= amount
	if v.HitHP < v.HP {
		v.HitHP = v.HP
	}
	return true
}

// RestoreWhileSit 执行一次权威的打坐恢复结算。它绝不会复活
// 死亡状态，在 LS_SITCROSS 之外也不做任何事；因此 UI 的恢复
// 速率属性绝不会被误当作实际治疗。
func (v *Vitals) RestoreWhileSit(hpAmount, mpAmount int32) bool {
	if hpAmount < 0 || mpAmount < 0 {
		return false
	}
	if v.Logic != entity.LogicStateSitcross || v.HP <= 0 {
		return false
	}
	changed := false
	if hpAmount > 0 && v.HP < v.MaxHP {
		v.HP += hpAmount
		if v.HP > v.MaxHP {
			v.HP = v.MaxHP
		}
		if v.HitHP < v.HP {
			v.HitHP = v.HP
		}
		changed = true
	}
	if mpAmount > 0 && v.MP < v.MaxMP {
		v.MP += mpAmount
		if v.MP > v.MaxMP {
			v.MP = v.MaxMP
		}
		changed = true
	}
	return changed
}

// SpendMP 是技能处理的权威资源闸门。
func (v *Vitals) SpendMP(cost int32) bool {
	if cost < 0 {
		return false
	}
	if v.MP < cost {
		return false
	}
	v.MP -= cost
	return true
}

// SetDerivedMaxima 在已装备内功评定完成后设置权威的资源总量
// 上限。基础上限永不改变，仍供客户端协议的 Max/Max+Add 显示
// 使用。上限增大时满值保持满值；部分消耗的保留绝对量，仅在上限
// 缩小时钳制。
func (v *Vitals) SetDerivedMaxima(maxHP, maxMP int32) bool {
	if maxHP < 1 {
		maxHP = 1
	}
	if maxMP < 1 {
		maxMP = 1
	}
	if v.MaxHP == maxHP && v.MaxMP == maxMP {
		return false
	}
	hpWasFull := v.HP >= v.MaxHP
	mpWasFull := v.MP >= v.MaxMP
	v.MaxHP = maxHP
	v.MaxMP = maxMP
	if hpWasFull {
		v.HP = maxHP
	} else if v.HP > maxHP {
		v.HP = maxHP
	}
	if v.HitHP > maxHP {
		v.HitHP = maxHP
	}
	if mpWasFull {
		v.MP = maxMP
	} else if v.MP > maxMP {
		v.MP = maxMP
	}
	return true
}

// ActorState 把组件值投影为 entity.ActorState，让协议层
// （npcCombatProperties 等）无需感知组件存储即可编码，
// 保证迁移前后线上字节不变。
func (v Vitals) ActorState(objectID, ownerID uint32) entity.ActorState {
	return entity.ActorState{
		ObjectID: objectID, OwnerID: ownerID,
		HP: v.HP, HitHP: v.HitHP, BaseMaxHP: v.BaseMaxHP, MaxHP: v.MaxHP,
		MP: v.MP, BaseMaxMP: v.BaseMaxMP, MaxMP: v.MaxMP,
		LogicState: v.Logic,
	}
}

// Brain 是敌对 NPC 的运行期战斗决策态，即原 npcCombatState。
// 模板参数（aggro/chase/range/speed/damage/cadence/respawn）
// 由注册方在 spawn 时一次性写入；threat/nextAttack/respawnAt/
// controlledUntil 是 tick 期间的决策字段。创建者/模板决定哪个
// 对象敌对——只有 attacknpc/bossnpc 会携带本组件，平民 NPC
// 只有 Vitals，两者的组件布局（archetype）天然分离。
type Brain struct {
	Home         worldcore.Transform
	AggroRange   float32
	ChaseRange   float32
	AttackRange  float32
	MoveSpeed    float32
	Damage       int32
	AttackEvery  time.Duration
	RespawnAfter time.Duration

	Threat          int32
	TargetMember    uint32
	NextAttack      time.Time
	RespawnAt       time.Time
	ControlledUntil time.Time
}
