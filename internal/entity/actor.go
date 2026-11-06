// Package entity 包含世界对象的权威运行时状态。
package entity

import (
	"math"
	"sync"
	"time"
)

const (
	LogicStateNormal   uint8 = 0
	LogicStateFighting uint8 = 1
	// LogicStateSitcross 是新版原生的 LS_SITCROSS 状态。动画由客户端
	// 负责，而何时允许进入该状态由服务器决定。
	LogicStateSitcross uint8 = 102
	LogicStateDied     uint8 = 120
)

// Actor 是角色战斗状态的服务器权威。客户端可以从
// ServerObjectProperty (0x10) 渲染这些数值，但绝不拥有它们。
type Actor struct {
	mu sync.RWMutex

	objectID           uint32
	ownerID            uint32
	name               string
	hp                 int32
	baseMaxHP          int32
	maxHP              int32
	hitHP              int32
	mp                 int32
	baseMaxMP          int32
	maxMP              int32
	qingGong           int32
	maxQingGong        int32
	maxQingGongAdd     int32
	qingGongConsumeMul float32
	qingGongUpdatedAt  time.Time
	speedRatio         int32
	logic              uint8
}

type ActorState struct {
	ObjectID, OwnerID    uint32
	Name                 string
	HP, BaseMaxHP, MaxHP int32
	HitHP                int32
	MP, BaseMaxMP, MaxMP int32
	QingGongPoint        int32
	MaxQingGongPoint     int32
	MaxQingGongPointAdd  int32
	QingGongConsumeMul   float32
	SpeedRatio           int32
	LogicState           uint8
}

func NewActor(objectID, ownerID uint32, name string, maxHP, maxMP int32) *Actor {
	if maxHP < 1 {
		maxHP = 1
	}
	if maxMP < 1 {
		maxMP = 1
	}
	// 现行正式服玩法文档将正常的满条描述为
	// 150。装备/buff 加成在下方单独表示。
	const defaultMaxQingGong = 150
	return &Actor{
		objectID: objectID, ownerID: ownerID, name: name,
		hp: maxHP, baseMaxHP: maxHP, maxHP: maxHP, hitHP: maxHP,
		mp: maxMP, baseMaxMP: maxMP, maxMP: maxMP,
		qingGong: defaultMaxQingGong, maxQingGong: defaultMaxQingGong,
		qingGongUpdatedAt: time.Now(), speedRatio: 100,
	}
}

// NewPlayer 构造新场景会话使用的默认存活玩家状态。
// 把这些数值放在这里，可以避免出生（spawn）与后续 0x10
// 更新各自为政、变成多份事实来源。
func NewPlayer(objectID, ownerID uint32, name string) *Actor {
	return NewActor(objectID, ownerID, name, 1000, 500)
}

func (a *Actor) Snapshot() ActorState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return ActorState{
		ObjectID: a.objectID, OwnerID: a.ownerID, Name: a.name,
		HP: a.hp, BaseMaxHP: a.baseMaxHP, MaxHP: a.maxHP, HitHP: a.hitHP,
		MP: a.mp, BaseMaxMP: a.baseMaxMP, MaxMP: a.maxMP,
		QingGongPoint: a.qingGong, MaxQingGongPoint: a.maxQingGong,
		MaxQingGongPointAdd: a.maxQingGongAdd, QingGongConsumeMul: a.qingGongConsumeMul,
		SpeedRatio: a.speedRatio, LogicState: a.logic,
	}
}

// ApplyDamage 在 actor 状态发生变化时返回 true。死亡是一次
// 状态迁移，而不是客户端对 HP == 0 的解释。
func (a *Actor) ApplyDamage(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.hp == 0 {
		return false
	}
	if a.hitHP < a.hp {
		a.hitHP = a.hp
	}
	a.hp -= amount
	if a.hp <= 0 {
		a.hp = 0
		a.hitHP = 0
		a.logic = LogicStateDied
	} else {
		a.logic = LogicStateFighting
	}
	return true
}

// ApplyHeal 恢复 HP，上限为 MaxHP。治疗有意不清除
// 死亡状态：未来的复活流程必须显式完成那次状态迁移。
func (a *Actor) ApplyHeal(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic == LogicStateDied || a.hp >= a.maxHP {
		return false
	}
	before := a.hp
	a.hp += amount
	if a.hp > a.maxHP {
		a.hp = a.maxHP
	}
	if a.hitHP < a.hp {
		a.hitHP = a.hp
	}
	return a.hp != before
}

// RecoverBruisedHPWhileFighting 只结算战斗期间的瘀血层。
// HitHP 是受击前/第二根血条，因此让它向 HP 回落可以使可见的
// 瘀血（HitHP-HP）消散，而不给予任何真实 HP。
//
// 在其他 LogicState 值下该方法有意不做任何事：真实 HP 的恢复
// 由打坐和显式治疗负责，而死亡的 actor 无法恢复。
func (a *Actor) RecoverBruisedHPWhileFighting(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic != LogicStateFighting || a.hp <= 0 || a.hitHP <= a.hp {
		return false
	}
	a.hitHP -= amount
	if a.hitHP < a.hp {
		a.hitHP = a.hp
	}
	return true
}

// RestoreWhileSit 执行一次权威的打坐恢复结算。它绝不会
// 复活死亡的 actor，在 LS_SITCROSS 之外也不做任何事；因此 UI 的
// 恢复速率属性绝不会被误当作实际治疗。
func (a *Actor) RestoreWhileSit(hpAmount, mpAmount int32) bool {
	if hpAmount < 0 || mpAmount < 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic != LogicStateSitcross || a.hp <= 0 {
		return false
	}
	changed := false
	if hpAmount > 0 && a.hp < a.maxHP {
		a.hp += hpAmount
		if a.hp > a.maxHP {
			a.hp = a.maxHP
		}
		if a.hitHP < a.hp {
			a.hitHP = a.hp
		}
		changed = true
	}
	if mpAmount > 0 && a.mp < a.maxMP {
		a.mp += mpAmount
		if a.mp > a.maxMP {
			a.mp = a.maxMP
		}
		changed = true
	}
	return changed
}

// SetDerivedMaxima 在角色的已装备内功评定完成后，
// 设置权威的资源总量上限。基础上限永不改变；
// 它们仍可供客户端协议的 MaxHP/MaxMP + Add 显示使用。
// 上限增大时，已完全恢复的 actor 保持满值；而部分消耗的
// actor 保留其当前的绝对资源量，仅在上限
// 缩小时才会被钳制。
func (a *Actor) SetDerivedMaxima(maxHP, maxMP int32) bool {
	if maxHP < 1 {
		maxHP = 1
	}
	if maxMP < 1 {
		maxMP = 1
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.maxHP == maxHP && a.maxMP == maxMP {
		return false
	}
	hpWasFull := a.hp >= a.maxHP
	mpWasFull := a.mp >= a.maxMP
	a.maxHP = maxHP
	a.maxMP = maxMP
	if hpWasFull {
		a.hp = maxHP
	} else if a.hp > maxHP {
		a.hp = maxHP
	}
	if a.hitHP > maxHP {
		a.hitHP = maxHP
	}
	if mpWasFull {
		a.mp = maxMP
	} else if a.mp > maxMP {
		a.mp = maxMP
	}
	return true
}

// SpendMP 是后续技能处理的权威资源闸门。
func (a *Actor) SpendMP(cost int32) bool {
	if cost < 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.mp < cost {
		return false
	}
	a.mp -= cost
	return true
}

// UseQingGong 施加当前由服务器权威判定的轻功资源
// 消耗。乘数是一个小数修正：-0.20 表示 80% 消耗，
// +0.20 表示 120% 消耗。消耗不会变为负数。
func (a *Actor) UseQingGong(baseCost int32, now time.Time) (cost int32, ok bool) {
	if baseCost < 0 {
		return 0, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.restoreQingGongLocked(now)
	multiplier := float64(1 + a.qingGongConsumeMul)
	if multiplier < 0 {
		multiplier = 0
	}
	// qingGongConsumeMul 以 float32 持久化；取整前先消除表示误差，
	// 这样 25 点的技能在 -0.2 之下消耗仍是 20 而不是 21。
	cost = int32(math.Ceil(float64(baseCost)*multiplier - 1e-6))
	if a.qingGong < cost {
		return cost, false
	}
	a.qingGong -= cost
	// 本地测试规则：不要让反复的移动校验把玩家困在
	// 无法使用的轻功条上。回满被有意放在权威扣费/判定之后，
	// 因此动作仍须支付得起才能
	// 开始；只有成功后剩余不足 50 点的动作才触发回满。
	if a.qingGong < 50 {
		a.qingGong = a.maxQingGong + a.maxQingGongAdd
		a.qingGongUpdatedAt = now
	}
	return cost, true
}

// RestoreQingGong 推进本地测试的资源计时器。恢复速率保留在这里
// （而不是客户端 Lua），以便后续 buff 能够权威地修改它。
func (a *Actor) RestoreQingGong(now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.restoreQingGongLocked(now)
}

func (a *Actor) restoreQingGongLocked(now time.Time) bool {
	if now.Before(a.qingGongUpdatedAt) {
		return false
	}
	// 本地基准：每秒 10 点。各技能的消耗由官方配置提供；
	// 回复修正属于未来的 buff 效果。
	points := int32(now.Sub(a.qingGongUpdatedAt) / (100 * time.Millisecond))
	if points <= 0 {
		return false
	}
	a.qingGongUpdatedAt = a.qingGongUpdatedAt.Add(time.Duration(points) * 100 * time.Millisecond)
	before := a.qingGong
	a.qingGong += points
	max := a.maxQingGong + a.maxQingGongAdd
	if a.qingGong > max {
		a.qingGong = max
	}
	return a.qingGong != before
}

func (a *Actor) SetQingGongConsumeMultiplier(multiplier float32) {
	a.mu.Lock()
	a.qingGongConsumeMul = multiplier
	a.mu.Unlock()
}

func (a *Actor) SetSpeedRatio(ratio int32) {
	if ratio < 0 {
		ratio = 0
	}
	a.mu.Lock()
	a.speedRatio = ratio
	a.mu.Unlock()
}

// SetLogicState 只改变当前状态；HP 和 MP 仍由同一个 actor 持有，
// 并由玩家适配器一并进行同步。
func (a *Actor) SetLogicState(state uint8) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logic = state
}

// Revive 是最小化的权威复活状态迁移。
func (a *Actor) Revive() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.hp = a.maxHP
	a.hitHP = a.maxHP
	a.mp = a.maxMP
	a.qingGong = a.maxQingGong + a.maxQingGongAdd
	a.qingGongUpdatedAt = time.Now()
	a.speedRatio = 100
	a.logic = LogicStateNormal
}
