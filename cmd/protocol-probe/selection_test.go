package main

import (
	"bytes"
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
		targetID = forkNPCID(targetID)
		packed := uint64(targetID) | uint64(targetOwner)<<32
		var raw [8]byte
		binary.LittleEndian.PutUint64(raw[:], packed)
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomSelectTarget},
			{Type: 8, Raw: raw},
		}}
	}

	if err := handleSelectTargetCustom(lifecycle, selectMsg(4456, npcTestOwner(4456)), "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	// 431 ack = 选中双帧（416/417）+ msg25[30000] 暂存授权（2026-11-12
	// 深夜恢复，官服 b.pcapng 双样本 + 55981/62096 会话实证：客户端等
	// 这帧才启动自动接近；7072 线裸双帧方言对本 fork 模式客户端不适用，
	// 见 selection.go 注释）。
	if len(frames) != 3 {
		t.Fatalf("select target frames=%x, want LastObject+AutoSelectObject+msg25", frames)
	}
	lastObject, autoSelect, pauseAck := frames[0], frames[1], frames[2]
	if lastObject[0] != 0x10 || binary.LittleEndian.Uint16(lastObject[12:]) != 414 ||
		binary.LittleEndian.Uint64(lastObject[14:]) != uint64(forkNPCID(4456))|uint64(npcTestOwner(4456))<<32 {
		t.Fatalf("LastObject frame=%x", lastObject)
	}
	if autoSelect[0] != 0x10 || binary.LittleEndian.Uint16(autoSelect[12:]) != propAutoSelectObject ||
		binary.LittleEndian.Uint64(autoSelect[14:]) != uint64(forkNPCID(4456))|uint64(npcTestOwner(4456))<<32 {
		t.Fatalf("AutoSelectObject frame=%x", autoSelect)
	}
	if pauseAck[0] != 0x1E || !bytes.Contains(pauseAck, []byte("skill_pause")) {
		t.Fatalf("select pause ack frame=%x, want msg25 skill_pause", pauseAck)
	}
	if targetID, targetOwner, ok := lifecycle.selectedCombatTarget(); !ok || targetID != forkNPCID(4456) || targetOwner != npcTestOwner(4456) {
		t.Fatalf("selected target=%d-%d ok=%t", targetID, targetOwner, ok)
	}

	// 取消选择（msgId 440 之后客户端跟发 431 obj(0,0)）。
	if err := handleSelectTargetCustom(lifecycle, selectMsg(0, 0), "test"); err != nil {
		t.Fatal(err)
	}
	frames = conn.Frames()
	// 选中三帧 + 清除双帧 = 5（清除路径不带 msg25）。
	if len(frames) != 5 || binary.LittleEndian.Uint64(frames[4][14:]) != 0 {
		t.Fatalf("clear target frames=%x, want zeroed AutoSelectObject tail", frames)
	}
	if _, _, ok := lifecycle.selectedCombatTarget(); ok {
		t.Fatal("clear target kept a selection")
	}

	// 未知对象：报错且不追加帧。
	if err := handleSelectTargetCustom(lifecycle, selectMsg(9999, 1), "test"); err == nil {
		t.Fatal("unknown target accepted")
	}
	if got := len(conn.Frames()); got != 5 {
		t.Fatalf("unknown target appended frames: %d", got)
	}

	// 形状校验：第二个值必须是 raw16 对象。
	bad := clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomSelectTarget}, {Type: 2, Int32: 4456}}}
	if err := handleSelectTargetCustom(lifecycle, bad, "test"); err == nil {
		t.Fatal("non-object target accepted")
	}
}
