package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// native 62311 #548-#751：204 项采集技能全 1 级，行形 (id, 1)。
func TestGrantCollectSkillsMirrorsNative(t *testing.T) {
	if len(collectSkillIDs) != 204 {
		t.Fatalf("collect skills=%d want=204", len(collectSkillIDs))
	}
	conn := &captureMessageConnection{}
	if err := grantCollectSkills(conn); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 204 {
		t.Fatalf("frames=%d want=204", len(frames))
	}
	first := frames[0]
	if got := binary.LittleEndian.Uint16(first[10:]); got != recordCollectSkill {
		t.Fatalf("record=%d want=67", got)
	}
	if !bytes.Contains(first, []byte("CS_by_xhmdl01")) {
		t.Fatalf("first row missing id: % X", first)
	}
	// 行形: 16 头 + u32 len(14) + "CS_by_xhmdl01\0" + i32(1) = 38B（native #548 同长）
	if len(first) != 38 {
		t.Fatalf("first row len=%d want=38 (native #548)", len(first))
	}
	if level := binary.LittleEndian.Uint32(first[34:]); level != 1 {
		t.Fatalf("level=%d want=1", level)
	}
}

// native #1680-#1682 quest49 簇逐列形状。
func TestPushJyzQuestClusterMirrorsNative(t *testing.T) {
	conn := &captureMessageConnection{}
	if err := pushJyzQuestCluster(conn, nil); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 4 {
		t.Fatalf("frames=%d want=4", len(frames))
	}
	task49 := frames[0]
	if len(task49) != 189 {
		t.Fatalf("task49 row len=%d want=189 (native #1680)", len(task49))
	}
	trace := frames[1]
	if len(trace) != 131 {
		t.Fatalf("trace row len=%d want=131 (native #1681)", len(trace))
	}
	// NewRideRec: 两行单帧 50B(native) → 拆两单行帧(各 24B) 行值一致
	for k, want := range []uint32{9513, 41523} {
		f := frames[2+k]
		if got := binary.LittleEndian.Uint16(f[10:]); got != recordNewRide {
			t.Fatalf("ride %d record=%d", k, got)
		}
		if pack := binary.LittleEndian.Uint32(f[16:]); pack != want {
			t.Fatalf("ride %d pack=%d want=%d", k, pack, want)
		}
	}
}
