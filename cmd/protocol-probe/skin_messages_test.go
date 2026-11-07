package main

import (
	"encoding/binary"
	"testing"
)

// 镜像 Java SkinMessagesTest：位图数学（serial=皮肤号+1）、显式行号、
// 倒序清行、UseCardRec Word 双列、显示开关帧、无裸 custom 180。

func TestSkinBitmapBitMath(t *testing.T) {
	cases := []struct {
		id       uint32
		row      uint32
		expected uint32
	}{
		{0, 0, 1 << 0},   // serial 1
		{1, 0, 1 << 1},   // serial 2（真机解锁修复锚点）
		{30, 0, 1 << 30}, // serial 31
		{31, 0, 1 << 31}, // serial 32 → 行 0 位 32
		{32, 1, 1 << 0},  // serial 33 → 行 1 位 1
	}
	for _, c := range cases {
		row, mask := skinBitmapBit(c.id)
		if row != c.row || mask != c.expected {
			t.Fatalf("skinBitmapBit(%d)=(%d,0x%X), want (%d,0x%X)", c.id, row, mask, c.row, c.expected)
		}
	}
}

func TestSkinBitmapRowsShape(t *testing.T) {
	// 空集也产出 1 行 0 值（建立通道清空旧位图）。
	rows, count := skinBitmapRows(nil)
	if count != 1 || rows[0] != 0 {
		t.Fatalf("empty rows=%v count=%d, want 1 行 0 值", rows, count)
	}
	// {1,31,32}：行 0 = bit2|bit32；行 1 = bit1。
	rows, count = skinBitmapRows([]int32{1, 31, 32})
	if count != 2 {
		t.Fatalf("count=%d, want 2", count)
	}
	if rows[0] != (1<<1)|(1<<31) || rows[1] != 1<<0 {
		t.Fatalf("rows=%v", rows)
	}
	// 负数忽略。
	rows, count = skinBitmapRows([]int32{-5})
	if count != 1 || rows[0] != 0 {
		t.Fatalf("negative rows=%v count=%d", rows, count)
	}
}

func TestSkinCollectedFramesExplicitRowNumbers(t *testing.T) {
	rows := map[uint32]uint32{0: 7, 1: 9}
	frames := skinCollectedFrames(1, 1, rows, 2)
	if len(frames) != 2 {
		t.Fatalf("frames=%d, want 2（含零值行全推）", len(frames))
	}
	for index, frame := range frames {
		if frame[0] != 0x11 {
			t.Fatalf("frame[%d] opcode=%#x", index, frame[0])
		}
		if row := binary.LittleEndian.Uint16(frame[12:]); row != uint16(index) {
			t.Fatalf("frame[%d] row=%d, want 显式行号 %d", index, row, index)
		}
		if value := binary.LittleEndian.Uint32(frame[16:]); value != rows[uint32(index)] {
			t.Fatalf("frame[%d] value=%d, want %d", index, value, rows[uint32(index)])
		}
	}
}

func TestSkinCollectedClearFramesDescending(t *testing.T) {
	frames := skinCollectedClearFrames(1, 1, 3)
	if len(frames) != 3 {
		t.Fatalf("frames=%d, want 3", len(frames))
	}
	want := []uint16{2, 1, 0}
	for index, frame := range frames {
		if frame[0] != 0x12 {
			t.Fatalf("frame[%d] opcode=%#x", index, frame[0])
		}
		if row := binary.LittleEndian.Uint16(frame[12:]); row != want[index] {
			t.Fatalf("frame[%d] row=%d, want 倒序 %d", index, row, want[index])
		}
	}
}

func TestSkinMountedFramesWordColumns(t *testing.T) {
	frames, err := skinMountedFrames(1, 1, []int32{7})
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 {
		t.Fatalf("frames=%d, want 1", len(frames))
	}
	frame := frames[0]
	if frame[0] != 0x11 {
		t.Fatalf("opcode=%#x", frame[0])
	}
	if index := binary.LittleEndian.Uint16(frame[10:]); index != recordUseCard {
		t.Fatalf("record index=%d, want %d", index, recordUseCard)
	}
	// 16 字节头 + 两个 u16 列；首列 = 皮肤号。
	if len(frame) != 16+4 {
		t.Fatalf("frame len=%d, want 20（Word 双列）", len(frame))
	}
	if id := binary.LittleEndian.Uint16(frame[16:]); id != 7 {
		t.Fatalf("skin id column=%d, want 7", id)
	}
	if displayType := binary.LittleEndian.Uint16(frame[18:]); displayType != 0 {
		t.Fatalf("display type column=%d, want 0", displayType)
	}
}

func TestSkinShowEquipTypeFrame(t *testing.T) {
	frame, err := skinShowEquipTypeFrame(1, 1, showEquipFashion, 42)
	if err != nil {
		t.Fatal(err)
	}
	if frame[0] != 0x10 {
		t.Fatalf("opcode=%#x", frame[0])
	}
	if count := binary.LittleEndian.Uint16(frame[10:]); count != 2 {
		t.Fatalf("property count=%d, want 2（ShowEquipType+CardID）", count)
	}
	if index := binary.LittleEndian.Uint16(frame[12:]); index != propShowEquipType {
		t.Fatalf("first property=%d, want ShowEquipType(408)", index)
	}
}

// 整刷编排帧序：清位图 → 推位图 → 清挂载 → 推挂载 → 显示开关；
// 镜像计数回写。
func TestPushSkinRefreshFramesOrchestration(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	player.applySkins(skinState{collected: []int32{1}, mounted: []int32{1}})
	// 预置旧推计数（模拟首推后整刷：先删旧行）。
	player.mu.Lock()
	player.skinRows, player.skinMountedRows = 1, 1
	player.mu.Unlock()

	if err := pushSkinRefreshFrames(conn, player); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	// 清位图 1（0x12 row0）→ 推位图 1（0x11 row0）→ 清挂载 1（0x12 row0）
	// → 推挂载 1（0x11）→ 显示开关 1（0x10）。
	if len(frames) != 5 {
		t.Fatalf("frames=%d: %x", len(frames), frames)
	}
	if frames[0][0] != 0x12 || frames[1][0] != 0x11 || frames[2][0] != 0x12 ||
		frames[3][0] != 0x11 || frames[4][0] != 0x10 {
		t.Fatalf("frame opcodes=%x, want 12 11 12 11 10", [5]byte{frames[0][0], frames[1][0], frames[2][0], frames[3][0], frames[4][0]})
	}
	player.mu.Lock()
	rows, mounted := player.skinRows, player.skinMountedRows
	player.mu.Unlock()
	if rows != 1 || mounted != 1 {
		t.Fatalf("mirrors=(%d,%d), want (1,1)", rows, mounted)
	}
}

// 空态登录恢复零帧（客户端保持未收录）。
func TestPushSavedSkinsEmptyStateZeroFrames(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("测试", 0)
	if err := pushSavedSkins(conn, player); err != nil {
		t.Fatal(err)
	}
	if len(conn.Frames()) != 0 {
		t.Fatalf("frames=%d, want 0", len(conn.Frames()))
	}
}

// 0x0C 重入镜像归零：清行不针对不存在的旧行。
func TestResetSkinRowMirrors(t *testing.T) {
	player := newPlayerActor("测试", 0)
	player.mu.Lock()
	player.skinRows, player.skinMountedRows = 5, 2
	player.mu.Unlock()
	player.resetSkinRowMirrors()
	player.mu.Lock()
	defer player.mu.Unlock()
	if player.skinRows != 0 || player.skinMountedRows != 0 {
		t.Fatalf("mirrors not reset: %d/%d", player.skinRows, player.skinMountedRows)
	}
}
