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

// Vitals 是 NPC 战斗数值的组件化形态，字段与 entity.Actor 中
// NPC 实际用到的子集一一对应。玩家侧暂时仍使用 entity.Actor；
// 两处逻辑必须保持等价，直到玩家也迁入组件存储。
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
// NPC 没有轻功条与速度比）。
func (v *Vitals) Revive() {
	v.HP = v.MaxHP
	v.HitHP = v.MaxHP
	v.MP = v.MaxMP
	v.Logic = entity.LogicStateNormal
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
	NextAttack      time.Time
	RespawnAt       time.Time
	ControlledUntil time.Time
}
