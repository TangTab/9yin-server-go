package fssoloworld

// 位置写回链测试：msg15 移动上传（走/停双态朝向口径）、界值拒收、
// 2s 节流存档水位、m327 任务列表同步。下行帧正确性由 entergen
// TaskAcceptedBatchFrame/MapRecords 字节级测试覆盖（nil inner 下
// sendToClient 静默）。

import (
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

func customFloat32(v float32) entergen.CustomInboundValue {
	return entergen.CustomInboundValue{Type: 4, Float32: v}
}

// TestMovementUploadUpdatesPosition msg15 双态：走帧 idx9 绝对朝向，
// 停帧保持存档朝向（idx6 是朝向增量不可用）；界值外拒收。
func TestMovementUploadUpdatesPosition(t *testing.T) {
	s, pl := newCustomTestServer(t)
	pl.roleState.Position = [4]float64{1, 2, 3, 0.5}

	// 走帧 state7：[15, intent, 7, x, y, z, destX, destZ, speed, orient, clock]。
	s.handleMovementUpload(pl, customMsg(15,
		customInt(0xA000), customInt(7),
		customFloat32(10.5), customFloat32(20.25), customFloat32(30.125),
		customFloat32(11), customFloat32(31), customFloat32(5), customFloat32(1.25), customInt(123)))
	if got := pl.roleState.Position; got != [4]float64{10.5, 20.25, 30.125, 1.25} {
		t.Fatalf("走帧后 Position=%v", got)
	}

	// 停帧 state4：坐标收编，朝向保持。
	s.handleMovementUpload(pl, customMsg(15,
		customInt(1), customInt(4),
		customFloat32(12), customFloat32(22), customFloat32(32), customFloat32(0.1)))
	if got := pl.roleState.Position; got != [4]float64{12, 22, 32, 1.25} {
		t.Fatalf("停帧后 Position=%v（朝向应保持 1.25）", got)
	}

	// 界值外拒收（|x|>10000）。
	s.handleMovementUpload(pl, customMsg(15,
		customInt(2), customInt(4),
		customFloat32(99999), customFloat32(22), customFloat32(32)))
	if got := pl.roleState.Position[0]; got != 12 {
		t.Fatalf("越界坐标被收编: %v", pl.roleState.Position)
	}

	// 节流窗口内：内存权威位仍更新（不重发存档）。
	pl.posSaveAt = time.Now()
	s.handleMovementUpload(pl, customMsg(15,
		customInt(3), customInt(4),
		customFloat32(13), customFloat32(23), customFloat32(33)))
	if got := pl.roleState.Position; got != [4]float64{13, 23, 33, 1.25} {
		t.Fatalf("窗口内 Position=%v（内存仍应更新）", got)
	}

	// 无 roleState 会话：拒收不崩（replay 回退路径）。
	bare := s.Players().AddPlayer(8, 1, "")
	s.handleMovementUpload(bare, customMsg(15,
		customInt(4), customInt(7),
		customFloat32(1), customFloat32(2), customFloat32(3), customFloat32(1),
		customFloat32(3), customFloat32(5), customFloat32(2.5), customInt(9)))
	if bare.roleState.Position != [4]float64{} {
		t.Fatalf("无档会话被写位: %v", bare.roleState.Position)
	}
}

// TestAbsorbPositionThrottleSave 首次收编即过水位（水位零值），窗口后
// 再收编再次过水位——存档节流的两次跨越点。
func TestAbsorbPositionThrottleSave(t *testing.T) {
	s, pl := newCustomTestServer(t)
	s.absorbPlayerPosition(pl, [4]float64{5, 6, 7, 0.5})
	if pl.roleState.Position != [4]float64{5, 6, 7, 0.5} {
		t.Fatalf("首次收编失败: %v", pl.roleState.Position)
	}
	first := pl.posSaveAt
	if first.IsZero() {
		t.Fatal("首次收编未过存档水位")
	}
	s.absorbPlayerPosition(pl, [4]float64{8, 9, 10, 0.5})
	if pl.roleState.Position != [4]float64{8, 9, 10, 0.5} {
		t.Fatalf("窗口内收编失败: %v", pl.roleState.Position)
	}
	pl.posSaveAt = time.Now().Add(-positionSaveInterval)
	backdated := pl.posSaveAt
	s.absorbPlayerPosition(pl, [4]float64{11, 12, 13, 0.5})
	// 注：Windows 时钟粒度下连续 time.Now() 可能同 tick 相等——断言
	// 「水位不再等于回拨值且已刷新为当前窗口内」而非严格递增。
	if pl.posSaveAt.Equal(backdated) {
		t.Fatal("窗口到期未重过水位")
	}
	if time.Since(pl.posSaveAt) >= positionSaveInterval {
		t.Fatalf("水位未刷新为当前时刻: %v", pl.posSaveAt)
	}
}

// TestTaskMsgSyncRepush m327 sub6 幂等重放（帧内容 entergen 侧覆盖）；
// 未接子命令静默不崩。
func TestTaskMsgSyncRepush(t *testing.T) {
	s, pl := newCustomTestServer(t)
	pl.roleState.TaskAccepted = []uint32{1069}
	s.handleTaskMsg(pl, customMsg(327, customInt(6), customStr("1069")))
	s.handleTaskMsg(pl, customMsg(327, customInt(1)))
}
