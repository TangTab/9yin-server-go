package entergen

// alldest_test.go 巡逻广播帧族字节形态：0x21 批量（44B 记录布局/
// rotate 槽）、0x10 State 翻转、0x0E 移除。

import (
	"encoding/binary"
	"math"
	"testing"
)

// TestServerAllDestFrameLayout 0x21 帧：op | u16 count | count×44B；槽序
// id/own/xyz/orient/speed/rotate/motion2/motion3/flags。
func TestServerAllDestFrameLayout(t *testing.T) {
	if got := ServerAllDestFrame(nil); got != nil {
		t.Fatal("空集应返回 nil")
	}
	entries := []AllDestEntry{{
		ObjectID: 0x11000005, OwnerID: 0x00660005,
		DestX: 100.5, DestY: 2, DestZ: 200.25, DestOrient: 1.5,
		Speed: 1.25, Rotate: float32(2 * math.Pi), MoveFlags: 1,
	}, {
		ObjectID: 0x11000006, OwnerID: 0x00660006,
		DestX: 1, DestY: 2, DestZ: 3, DestOrient: 0,
		Speed: 0, MoveFlags: 0, // 停留条目：rotate 恒 0
	}}
	frame := ServerAllDestFrame(entries)
	if len(frame) != 3+2*44 {
		t.Fatalf("帧长 %d ≠ %d", len(frame), 3+2*44)
	}
	if frame[0] != 0x21 {
		t.Fatalf("op=%#x", frame[0])
	}
	if n := binary.LittleEndian.Uint16(frame[1:]); n != 2 {
		t.Fatalf("count=%d", n)
	}
	e0 := frame[3:]
	if id := binary.LittleEndian.Uint32(e0[0:]); id != 0x11000005 {
		t.Fatalf("slot0 id=%#x", id)
	}
	if own := binary.LittleEndian.Uint32(e0[4:]); own != 0x00660005 {
		t.Fatalf("slot1 own=%#x", own)
	}
	if x := math.Float32frombits(binary.LittleEndian.Uint32(e0[8:])); x != 100.5 {
		t.Fatalf("slot2 x=%v", x)
	}
	if speed := math.Float32frombits(binary.LittleEndian.Uint32(e0[24:])); speed != 1.25 {
		t.Fatalf("slot6 speed=%v", speed)
	}
	if rotate := math.Float32frombits(binary.LittleEndian.Uint32(e0[28:])); math.Abs(float64(rotate-2*math.Pi)) > 1e-6 {
		t.Fatalf("slot7 rotate=%v（应 2π）", rotate)
	}
	if flags := binary.LittleEndian.Uint32(e0[40:]); flags != 1 {
		t.Fatalf("slot10 flags=%d", flags)
	}
	e1 := frame[3+44:]
	if rotate := math.Float32frombits(binary.LittleEndian.Uint32(e1[28:])); rotate != 0 {
		t.Fatalf("停留条目 rotate=%v（应 0）", rotate)
	}
	if flags := binary.LittleEndian.Uint32(e1[40:]); flags != 0 {
		t.Fatalf("停留条目 flags=%d", flags)
	}
}

// TestNpcStateFrame 0x10 State 翻转帧：单属性 State=传入的动作名（walk /
// stand / 姿态动作名）；空状态拒绝。
func TestNpcStateFrame(t *testing.T) {
	walk, err := NpcStateFrame(0x11000005, 0x00660005, "walk")
	if err != nil {
		t.Fatal(err)
	}
	if walk[0] != 0x10 || walk[1] != 0 {
		t.Fatalf("op=%#x isView=%d", walk[0], walk[1])
	}
	if id := binary.LittleEndian.Uint32(walk[2:]); id != 0x11000005 {
		t.Fatalf("obj=%#x", id)
	}
	if n := binary.LittleEndian.Uint16(walk[10:]); n != 1 {
		t.Fatalf("属性数=%d", n)
	}
	if !contains(walk, "walk") {
		t.Fatal("walk 帧缺 walk 串")
	}
	stand, _ := NpcStateFrame(0x11000005, 0x00660005, "stand")
	if !contains(stand, "stand") {
		t.Fatal("stand 帧缺 stand 串")
	}
	pose, err := NpcStateFrame(0x11000005, 0x00660005, "ty_sitchair_drinkloop")
	if err != nil || !contains(pose, "ty_sitchair_drinkloop") {
		t.Fatalf("姿态 State 帧应带动作名: err=%v bytes=% x", err, pose)
	}
	if _, err := NpcStateFrame(0x11000005, 0x00660005, ""); err == nil {
		t.Fatal("空状态应拒绝（会把 NPC 置成空动作）")
	}
}

// TestSceneRemoveObjectFrame 0x0E 移除帧（9B）。
func TestSceneRemoveObjectFrame(t *testing.T) {
	frame := SceneRemoveObjectFrame(0x11000005, 0x00660005)
	if len(frame) != 9 || frame[0] != 0x0E {
		t.Fatalf("帧形态异常 len=%d op=%#x", len(frame), frame[0])
	}
	if id := binary.LittleEndian.Uint32(frame[1:]); id != 0x11000005 {
		t.Fatalf("obj=%#x", id)
	}
	if own := binary.LittleEndian.Uint32(frame[5:]); own != 0x00660005 {
		t.Fatalf("own=%#x", own)
	}
}

func contains(haystack []byte, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
