package main

import (
	"encoding/binary"
	"testing"
)

// TestSwitchSyncFrame 钉死 0x1E 414("sync") 批量开关帧形：外壳/消息号/
// 命令串与 Java StarterGrants.switchSyncFrame 同源；2919（ST_FUNCTION_
// NEW_C_FORM，人物面板「切换新版」按钮门槛）必须在列，1223（技能动作
// 开关）必须不在列——"sync" 会清空整棵开关树后重设，1223 走 "reset" 单开。
func TestSwitchSyncFrame(t *testing.T) {
	if len(switchSyncIDs) != 203 {
		t.Fatalf("switchSyncIDs len=%d, want 203（Java SWITCH_SYNC_IDS 同数）", len(switchSyncIDs))
	}
	for i := 1; i < len(switchSyncIDs); i++ {
		if switchSyncIDs[i-1] >= switchSyncIDs[i] {
			t.Fatalf("switchSyncIDs 非严格升序 @%d: %d >= %d", i, switchSyncIDs[i-1], switchSyncIDs[i])
		}
	}
	inList := func(id int32) bool {
		for _, v := range switchSyncIDs {
			if v == id {
				return true
			}
		}
		return false
	}
	if !inList(2919) {
		t.Fatal("switchSyncIDs 缺 2919（ST_FUNCTION_NEW_C_FORM），切换新版按钮不出现")
	}
	if !inList(0) || !inList(2925) {
		t.Fatal("switchSyncIDs 首尾漂移（Java 表以 0 起 2925 止）")
	}
	if inList(1223) {
		t.Fatal("switchSyncIDs 不应含 1223（技能动作开关走 reset 单开）")
	}

	frame, err := switchSyncFrame()
	if err != nil {
		t.Fatalf("switchSyncFrame: %v", err)
	}
	if frame[0] != 0x1E {
		t.Fatalf("opcode=%#x, want 0x1E（旧版外壳，Java pushSwitchSync 同形）", frame[0])
	}
	argc := binary.LittleEndian.Uint16(frame[1:3])
	if want := uint16(len(switchSyncIDs) + 2); argc != want {
		t.Fatalf("argc=%d, want %d（消息号 + sync + 203 开关号）", argc, want)
	}
	// TVarList[0] = int32 消息号 414（tag2），[1] = 窄字符串 "sync"（tag6）。
	if frame[3] != 2 {
		t.Fatalf("TVarList[0] tag=%d, want 2（int32 消息号）", frame[3])
	}
	if got := binary.LittleEndian.Uint32(frame[4:8]); got != uint32(serverSwitchControlMessage) {
		t.Fatalf("消息号=%d, want %d", got, serverSwitchControlMessage)
	}
	if frame[8] != 6 {
		t.Fatalf("TVarList[1] tag=%d, want 6（窄字符串）", frame[8])
	}
	if got := binary.LittleEndian.Uint32(frame[9:13]); got != 5 { // len("sync")+1
		t.Fatalf("sync 串长=%d, want 5", got)
	}
	if string(frame[13:18]) != "sync\x00" {
		t.Fatalf("命令串=%q, want \"sync\"", string(frame[13:18]))
	}
}
