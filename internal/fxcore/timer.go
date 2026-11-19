package fxcore

import (
	"math"
	"time"
)

// GameTimer 复刻官方 CGameTimer：帧时钟。真实时间取单调钟
//（官方 QueryPerformanceCounter），Execute(step) 以定步步长推进
// 模拟时间视图——泵在 fpsCap>0 时按 1/fpsCap 步进，全速时按实测 dt。
type GameTimer struct {
	startTime time.Time
	// simTime 定步推进的模拟时间秒（Execute 累加 step）。
	simTime float64
	// simLast 上次 Execute 时刻的模拟时间（ElapseSince 基准）。
	simLast float64
}

// NewGameTimer 创建并启动时钟（官方 ctor 即 Start）。
func NewGameTimer() *GameTimer {
	t := &GameTimer{}
	t.Reset()
	return t
}

// Reset 复位起点（官方 CGameTimer::Start，Initialize 调用位）。
func (t *GameTimer) Reset() {
	t.startTime = time.Now()
	t.simTime = 0
	t.simLast = 0
}

// Now 真实流逝秒（官方 GetTime：CCore::Initialize 处取值存 logClock）。
func (t *GameTimer) Now() float64 { return time.Since(t.startTime).Seconds() }

// Execute 复刻官方 Execute(elapsed)：推进模拟时钟并返回步进秒。
// 官方语义：定步模式传入 1/fpsCap，全速模式传入实测 dt；
// 返回值即本步长（泵累计入 virtualTime）。
func (t *GameTimer) Execute(step float64) float64 {
	if step < 0 {
		step = 0
	}
	t.simLast = t.simTime
	t.simTime += step
	return step
}

// SimTime 模拟时间秒（定步累计；全速模式=真实累计）。
func (t *GameTimer) SimTime() float64 { return t.simTime }

// SimElapsed 上次 Execute 的步长。
func (t *GameTimer) SimElapsed() float64 { return t.simTime - t.simLast }

// u64Tick 官方全局 u64Tick 等价：毫秒Tick 计数（监控/日志时间戳面）。
func u64Tick() uint64 { return uint64(time.Now().UnixMilli()) }

// clampFrame 官方 dt 钳制语义（防大跳；暂停恢复/断点后不追帧）。
func clampFrame(dt float64) float64 { return math.Min(dt, 0.25) }
