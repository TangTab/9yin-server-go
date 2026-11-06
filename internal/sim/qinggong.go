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

// Use 施加当前由服务器权威判定的轻功资源消耗。本本地测试规则：
// 回满被有意放在权威扣费/判定之后，因此动作仍须支付得起才能开始；
// 只有成功后剩余不足 50 点的动作才触发回满。
func (q *QingGong) Use(baseCost int32, now time.Time) (cost int32, ok bool) {
	if baseCost < 0 {
		return 0, false
	}
	q.Restore(now)
	multiplier := float64(1 + q.ConsumeMul)
	if multiplier < 0 {
		multiplier = 0
	}
	// ConsumeMul 以 float32 持久化；取整前先消除表示误差，
	// 这样 25 点的技能在 -0.2 之下消耗仍是 20 而不是 21。
	cost = int32(math.Ceil(float64(baseCost)*multiplier - 1e-6))
	if q.Point < cost {
		return cost, false
	}
	q.Point -= cost
	if q.Point < 50 {
		q.Point = q.Max + q.MaxAdd
		q.UpdatedAt = now
	}
	return cost, true
}

// Restore 推进本地测试的资源计时器。本地基准：每秒 10 点
// （100ms 一点）；各技能的消耗由官方配置提供。
func (q *QingGong) Restore(now time.Time) bool {
	if now.Before(q.UpdatedAt) {
		return false
	}
	points := int32(now.Sub(q.UpdatedAt) / (100 * time.Millisecond))
	if points <= 0 {
		return false
	}
	q.UpdatedAt = q.UpdatedAt.Add(time.Duration(points) * 100 * time.Millisecond)
	before := q.Point
	q.Point += points
	max := q.Max + q.MaxAdd
	if q.Point > max {
		q.Point = max
	}
	return q.Point != before
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
