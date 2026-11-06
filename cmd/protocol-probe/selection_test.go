package main

import (
	"encoding/binary"
	"testing"
)

// msg431 选择目标（fork selection.go 还原）：选中回
// LastObject(100)+AutoSelectObject(230) 双帧并镜像选择态；
// obj(0,0) 清空；未知对象报错不回帧。
func TestHandleSelectTargetCustom(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_test"
	npc.resolved.ScriptClass = "AttackNpc"
	if err := lifecycle.registerNPCOnRun(4456, npc); err != nil {
		t.Fatal(err)
	}

	selectMsg := func(targetID, targetOwner uint32) clientCustomMessage {
		packed := uint64(targetID) | uint64(targetOwner)<<32
		var raw [8]byte
		binary.LittleEndian.PutUint64(raw[:], packed)
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomSelectTarget},
			{Type: 8, Raw: raw},
		}}
	}

	if err := handleSelectTargetCustom(lifecycle, selectMsg(4456, 1), "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 2 {
		t.Fatalf("select target frames=%x, want LastObject+AutoSelectObject", frames)
	}
	lastObject, autoSelect := frames[0], frames[1]
	if lastObject[0] != 0x10 || binary.LittleEndian.Uint16(lastObject[12:]) != 414 ||
		binary.LittleEndian.Uint64(lastObject[14:]) != uint64(4456)|uint64(1)<<32 {
		t.Fatalf("LastObject frame=%x", lastObject)
	}
	if autoSelect[0] != 0x10 || binary.LittleEndian.Uint16(autoSelect[12:]) != propAutoSelectObject ||
		binary.LittleEndian.Uint64(autoSelect[14:]) != uint64(4456)|uint64(1)<<32 {
		t.Fatalf("AutoSelectObject frame=%x", autoSelect)
	}
	if targetID, targetOwner, ok := lifecycle.selectedCombatTarget(); !ok || targetID != 4456 || targetOwner != 1 {
		t.Fatalf("selected target=%d-%d ok=%t", targetID, targetOwner, ok)
	}

	// 取消选择（msgId 440 之后客户端跟发 431 obj(0,0)）。
	if err := handleSelectTargetCustom(lifecycle, selectMsg(0, 0), "test"); err != nil {
		t.Fatal(err)
	}
	frames = conn.Frames()
	if len(frames) != 4 || binary.LittleEndian.Uint64(frames[3][14:]) != 0 {
		t.Fatalf("clear target frames=%x, want zeroed AutoSelectObject tail", frames)
	}
	if _, _, ok := lifecycle.selectedCombatTarget(); ok {
		t.Fatal("clear target kept a selection")
	}

	// 未知对象：报错且不追加帧。
	if err := handleSelectTargetCustom(lifecycle, selectMsg(9999, 1), "test"); err == nil {
		t.Fatal("unknown target accepted")
	}
	if got := len(conn.Frames()); got != 4 {
		t.Fatalf("unknown target appended frames: %d", got)
	}

	// 形状校验：第二个值必须是 raw16 对象。
	bad := clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomSelectTarget}, {Type: 2, Int32: 4456}}}
	if err := handleSelectTargetCustom(lifecycle, bad, "test"); err == nil {
		t.Fatal("non-object target accepted")
	}
}
