package entergen

import (
	"encoding/binary"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// TestMountPlanShape 钉死挂载计划（golden 0x14 帧提取）：48 项、4 全局 +
// 44 玩家域、顺序首尾锚、全部记录名在当前方言表可解析。
func TestMountPlanShape(t *testing.T) {
	if len(MountPlan) != 48 {
		t.Fatalf("挂载计划 48 项（golden 实证），得 %d", len(MountPlan))
	}
	global, player := 0, 0
	for _, spec := range MountPlan {
		switch spec.Sub {
		case 2:
			global++
		case 0:
			player++
		default:
			t.Fatalf("sub 域值非法 %d（%s）", spec.Sub, spec.Name)
		}
		clientdata.MustRecordIndexByName(spec.Name) // 缺名 panic
	}
	if global != 4 || player != 44 {
		t.Fatalf("sub 分布 4 全局 + 44 玩家，得 %d/%d", global, player)
	}
	anchors := []struct {
		idx  int
		name string
	}{
		{0, "WeatherRec"}, {1, "Time_Limit_Form_Rec"}, {2, "scene_stall_rec"}, {3, "prep_show_npc_rec"},
		{4, "TiGuanRec"}, {5, "Task_Accepted"}, {7, "Task_Completed"},
		{47, "OffLineJob_Log"},
	}
	for _, a := range anchors {
		if MountPlan[a.idx].Name != a.name {
			t.Fatalf("顺序锚 #%d = %s，应 %s", a.idx, MountPlan[a.idx].Name, a.name)
		}
	}
}

// TestMountSequenceFrames 行推序列（2026-11-22 晚定案：0x14 挂载帧不发
// ——aow/golden 方言遗物，2433 fork 官服对拍零 0x14）：Task_Completed
// 位图行、Task_Accepted 0x29 批量、QingGongRec 18 行、无 0x14 帧。
func TestMountSequenceFrames(t *testing.T) {
	ctx := testContext(t)
	frames, err := MountSequenceFrames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var mounts, bitmapRows, qgRows int
	var accepted29 bool
	for _, f := range frames {
		switch f[0] {
		case 0x14:
			mounts++
		case 0x11:
			if f[1] != 0 {
				break
			}
			switch rec := binary.LittleEndian.Uint16(f[10:]); rec {
			case clientdata.MustRecordIndexByName("Task_Completed"):
				bitmapRows++
			case clientdata.MustRecordIndexByName("QingGongRec"):
				qgRows++
			}
		case 0x29:
			accepted29 = true
		}
	}
	if mounts != 0 {
		t.Fatalf("0x14 挂载帧应为 0（fork 方言不发），得 %d", mounts)
	}
	if !accepted29 {
		t.Fatal("Task_Accepted 0x29 批量缺席")
	}
	if qgRows != 18 {
		t.Fatalf("QingGongRec 18 行，得 %d", qgRows)
	}
	// Task_Completed 位图：OpenGate 49634 → final row 1551 → 1552 行。
	if bitmapRows != 1552 {
		t.Fatalf("位图 0x11 行 1552（final-row 前全零行），得 %d", bitmapRows)
	}
}

// mountName 从 0x14 帧反查记录名（测试用）。
func mountName(f []byte) string {
	idx := binary.LittleEndian.Uint16(f[10:])
	schema, _ := clientdata.RecordSchemaByIndex(int(idx))
	return schema.Name
}

// countFollowingRows 从 from 起数连续同记录 0x11 行数。
func countFollowingRows(frames [][]byte, from int) int {
	n := 0
	for i := from; i < len(frames) && frames[i][0] == 0x11; i++ {
		n++
	}
	return n
}
