package sim

import (
	"math"
	"time"
)

// 现行正式服玩法文档将正常的满条描述为 150。
// 装备/buff 加成在 MaxAdd 中单独表示。
const defaultMaxQingGong = 150

// QingGong 是轻功资源条组件，含时间戳驱动的被动回复。逻辑自
// entity.Actor 的 qingGong 字段组逐行移植：恢复速率保留在服务端
// （而不是客户端 Lua），以便后续 buff 能权威地修改它。
type QingGong struct {
	Point      int32
	Max        int32
	MaxAdd     int32
	ConsumeMul float32
	UpdatedAt  time.Time
}

func NewQingGong(now time.Time) QingGong {
	return QingGong{
		Point: defaultMaxQingGong, Max: defaultMaxQingGong,
		UpdatedAt: now,
	}
}

// Reset 是复活路径的权威回满。
func (q *QingGong) Reset(now time.Time) {
	q.Point = q.Max + q.MaxAdd
	q.UpdatedAt = now
}

// SetConsumeMultiplier 施加小数修正：-0.20 表示 80% 消耗，
// +0.20 表示 120% 消耗。
func (q *QingGong) SetConsumeMultiplier(multiplier float32) {
	q.ConsumeMul = multiplier
}

// Use 施加服务器权威判定的轻功一次性消耗（216 begin 的 qgdefine
// Consume / 技能 AConsumeQGP）。2026-11-07 对齐 Java Actor.useQingGong
// 去掉「余量 <50 自动回满」本地测试规则——官方轻功回复是分档慢回复
// （见 RecoverTiered），回满会让消耗经济失去意义（真机「无限跳」根因）。
// 消耗系数档保留：ConsumeMul 以 float32 持久化，取整前先消除表示误差，
// 这样 25 点的技能在 -0.2 之下消耗仍是 20 而不是 21。
func (q *QingGong) Use(baseCost int32, now time.Time) (cost int32, ok bool) {
	if baseCost < 0 {
		return 0, false
	}
	multiplier := float64(1 + q.ConsumeMul)
	if multiplier < 0 {
		multiplier = 0
	}
	cost = int32(math.Ceil(float64(baseCost)*multiplier - 1e-6))
	if q.Point < cost {
		return cost, false
	}
	q.Point -= cost
	q.UpdatedAt = now
	return cost, true
}

// Drain 持续态扣点（缓降/踏波行等租约的每秒消耗）：不触发任何回满，
// 付不起返回 false（调用方强摘租约）。Java Actor.drainQingGong 同口径。
func (q *QingGong) Drain(cost int32) bool {
	if cost < 0 {
		cost = 0
	}
	if q.Point < cost {
		q.Point = 0
		return false
	}
	q.Point -= cost
	return true
}

// RecoverTiered 官方分档回复一步（玩家资料实测口径，Java
// Actor.recoverQingGongTiered 同款）：按回复前余量取档——
// 101..上限 → +10、51..100 → +5、0..50 → +3；封顶 max+add。
// 由 1s ticker 每 3s 推进一步，消耗期（缓降/持续轻功租约）暂停。
func (q *QingGong) RecoverTiered() bool {
	cap := q.Max + q.MaxAdd
	if q.Point >= cap {
		return false
	}
	gain := int32(3)
	switch {
	case q.Point >= 101:
		gain = 10
	case q.Point >= 51:
		gain = 5
	}
	q.Point += gain
	if q.Point > cap {
		q.Point = cap
	}
	return true
}

// Motion 是移动修正组件。SpeedRatio 当前恒为 100（百分比），
// 保留组件形态供后续 buff 权威修改；复活时重置。
type Motion struct {
	SpeedRatio int32
}

func NewMotion() Motion {
	return Motion{SpeedRatio: 100}
}

// SetSpeedRatio 以百分比设置速度比，负值钳为 0。
func (m *Motion) SetSpeedRatio(ratio int32) {
	if ratio < 0 {
		ratio = 0
	}
	m.SpeedRatio = ratio
}
