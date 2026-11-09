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

// ResetAfterLeash 是活怪栓绳归位的权威完全重置：HP/瘀血层/MP 回满
// 并退回 Normal（数值迁移与 Revive 一致；调用场景是脱战回家而非
// 尸体复活，帧形对齐官方重置帧「HP 回满 + 463=0」）。
func (v *Vitals) ResetAfterLeash() {
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

// SitcrossSettlement 记录一次打坐恢复的逐层结算结果，供出帧层
// 按官方形态拆帧：HP 层产 n=1 属性帧对 + msg24 c1=16 治疗飘字、
// MP 层产属性帧对 + c1=18 回蓝飘字（官服补满末跳 (实际, 溢出)
// 拆分实证），HitHP 层产自己的属性帧对。
type SitcrossSettlement struct {
	// HPApplied 是本跳实际落入 HP 的量（飘字帧第 2 值；未触顶跳
	// 等于名义回复量）。
	HPApplied int32
	// HPWasted 是被上限截断的溢出量（飘字帧第 3 值；官服末跳
	// 162+1478=名义1640）。
	HPWasted int32
	// HitHPChanged 表示瘀血层 HitHP 被抬升到新 HP（回复追平受击
	// 血条的那一跳，官服 00:11:54.833 双 Ratio+双值四帧实证）。
	HitHPChanged bool
	// MPApplied/MPWasted 与 HP 层同口径（官服 c1=18 回蓝飘字：
	// 49.4.14.150:7074 实时实证每跳 +519 飘字，心搏 +100 无飘字）。
	MPApplied int32
	MPWasted  int32
}

// Changed 汇总本跳是否有任何层落到状态变化。
func (s SitcrossSettlement) Changed() bool {
	return s.HPApplied > 0 || s.HitHPChanged || s.MPApplied > 0
}

// RestoreWhileSit 执行一次权威的打坐恢复结算。它绝不会复活
// 死亡状态，在 LS_SITCROSS 之外也不做任何事；因此 UI 的恢复
// 速率属性绝不会被误当作实际治疗。
func (v *Vitals) RestoreWhileSit(hpAmount, mpAmount int32) SitcrossSettlement {
	var settlement SitcrossSettlement
	if hpAmount < 0 || mpAmount < 0 {
		return settlement
	}
	if v.Logic != entity.LogicStateSitcross || v.HP <= 0 {
		return settlement
	}
	if hpAmount > 0 && v.HP < v.MaxHP {
		before := v.HP
		v.HP += hpAmount
		if v.HP > v.MaxHP {
			v.HP = v.MaxHP
		}
		settlement.HPApplied = v.HP - before
		settlement.HPWasted = hpAmount - settlement.HPApplied
		if v.HitHP < v.HP {
			v.HitHP = v.HP
			settlement.HitHPChanged = true
		}
	}
	if mpAmount > 0 && v.MP < v.MaxMP {
		before := v.MP
		v.MP += mpAmount
		if v.MP > v.MaxMP {
			v.MP = v.MaxMP
		}
		settlement.MPApplied = v.MP - before
		settlement.MPWasted = mpAmount - settlement.MPApplied
	}
	return settlement
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

// RestoreMP 回复内力（内功特效 mp_restore 指令的落点），钳到上限。
func (v *Vitals) RestoreMP(amount int32) bool {
	if amount <= 0 || v.MP >= v.MaxMP {
		return false
	}
	before := v.MP
	v.MP += amount
	if v.MP > v.MaxMP {
		v.MP = v.MaxMP
	}
	return v.MP != before
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
	// Passive 镜像模板/creator 字段 NotPositive（官方属性表 idx 62）：
	// 城内 AttackNpc 大多置 1（掷斧监工/背柴工人/剧情人物等白名
	// NPC），官方语义为不主动攻击、仅受击反击（Java NpcCombatAi 全量
	// 即此模型）。置位时 tick 不做近身主动仇恨，目标只能由受击路径
	//（applySkillDamageToNPC）锁定。
	Passive bool

	Threat          int32
	TargetMember    uint32
	NextAttack      time.Time
	RespawnAt       time.Time
	ControlledUntil time.Time
}
