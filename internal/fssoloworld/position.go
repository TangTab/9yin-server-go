package fssoloworld

// 位置写回链（2433 线 rush_cast.go/motion.go 口径）：客户端移动上报
//（0x0A 67B 位移包 + 自定义消息 15 TVarList 移动上传）→ roleState.Position
// 内存权威 → 2s 节流 store 0x04 存档 + 下线兜底（entryClientClosed 既有）。
// 重登进场 0x1F/0x0D 即用存档位（enterContext State.Position）——
// 「走到哪重登回哪」，此前每次进场恒回出生点。

import (
	"math"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// clientCustomMovementUpload C2S 15 移动上传（CLIENT_CUSTOMMSG_MOVE_SCENE，
// 2433 线 rush_cast.go:37）：[15, intent, state, x, y, z, destX, destZ,
// speed, orient, clock]；state 7=行走（长式 idx9=绝对朝向）/4=停/5=跳。
const clientCustomMovementUpload int32 = 15

// clientCustomTaskMsg C2S 327 任务消息（CLIENT_CUSTOMMSG_TASK_MSG，
// 2433 线 official_login_align.go:54）：sub 6=已接任务列表同步。
const clientCustomTaskMsg int32 = 327

// positionSaveInterval 位置节流存档窗口（2433 线 saveLocation 2s 同款；
// 逐帧落库不可取，下线兜底覆盖窗口内丢失）。
const positionSaveInterval = 2 * time.Second

// validPlayerPosition 位移包同款有限性/界值校验（防畸形坐标污染存档；
// parseMotionPosition 界值同源：|x|/|z|≤10000、|y|≤3000、|orient|≤20）。
func validPlayerPosition(pos [4]float64) bool {
	for i, limit := range [4]float64{10000, 3000, 10000, 20} {
		v := pos[i]
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > limit {
			return false
		}
	}
	return true
}

// absorbPlayerPosition 移动上报收编：roleState.Position 内存权威更新 +
// NPC 视口对账 + 2s 节流存档。无 roleState 会话（未进场/replay 回退）
// 只做 NPC 对账不落库。切图在途（sceneSwitch）全丢弃——0x0C 后首个
// 0x0A 是旧场景过期活动（2433 定版坑）；传送门圆判定挂尾部。
func (s *Server) absorbPlayerPosition(pl *Player, pos [4]float64) {
	if !validPlayerPosition(pos) {
		return
	}
	if s.sceneSwitchInFlight(pl) {
		return
	}
	s.npcReconcile(pl, pos)
	// 冲锋暂存中：位置上报驱动进带判定（无暂存即时返回）。
	s.rushOnReport(pl, pos[0], pos[2])
	if !pl.haveRoleState {
		return
	}
	pl.roleState.Position = pos
	if now := time.Now(); now.Sub(pl.posSaveAt) >= positionSaveInterval {
		pl.posSaveAt = now
		s.sendRoleSave(pl)
	}
	s.checkPortal(pl, pos)
}

// handleMovementUpload C2S 15 TVarList 移动上传：idx3..5 恒为 x/y/z（走/
// 停双态实证，2433 线 movement_echo_test.go 定版）；state=7 长式 idx9 为
// 绝对朝向，其余态沿用存档朝向（停帧 idx6 是朝向增量不可用）。
func (s *Server) handleMovementUpload(pl *Player, message entergen.CustomInbound) {
	values := message.Values
	if len(values) < 6 {
		return
	}
	state, ok := values[2].ExactInt32()
	if !ok {
		return
	}
	var pos [4]float64
	for i := 0; i < 3; i++ {
		coordinate, ok := customFloat(values[3+i])
		if !ok {
			return
		}
		pos[i] = coordinate
	}
	pos[3] = pl.roleState.Position[3]
	if state == 7 && len(values) >= 10 {
		if orient, ok := customFloat(values[9]); ok {
			pos[3] = orient
		}
	}
	s.absorbPlayerPosition(pl, pos)
}

// handleTaskMsg C2S 327 任务消息：sub 6=已接任务列表同步 → 幂等重放
// Task_Accepted 0x29 批量帧（与进场挂载行推同源编码）；其余子命令仅
// 簿记——接/交任务受理链未落地（2433 线同样仅 GM 写入口）。
func (s *Server) handleTaskMsg(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 {
		return
	}
	sub, ok := message.Values[1].ExactInt32()
	if !ok {
		return
	}
	if sub != 6 {
		s.trace("task msg sub=%d client=%d (unhandled)", sub, pl.Conn1)
		return
	}
	if !pl.haveRoleState {
		return
	}
	frame, err := entergen.TaskAcceptedBatchFrame(entergen.Context{
		Identity: pl.entergenIdentity(),
		State:    pl.roleState,
	})
	if err != nil {
		s.trace("task repush failed client=%d: %v", pl.Conn1, err)
		return
	}
	if frame != nil {
		s.sendToClient(pl, frame)
	}
}

// customFloat TVarList 数值取 float（f32/f64 双型；int 型不取——坐标域
// 无整型方言，宽容即污染）。
func customFloat(value entergen.CustomInboundValue) (float64, bool) {
	switch value.Type {
	case 4:
		return float64(value.Float32), true
	case 5:
		return value.Float64, true
	}
	return 0, false
}
