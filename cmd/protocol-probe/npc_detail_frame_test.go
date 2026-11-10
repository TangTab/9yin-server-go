package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// TestNPCDetail0x28TemplateHeaderInvariants 钉死官服 0x28 模板帧头布局
// （c.pcapng 21 帧双模板逐字节定版，2026-11-10）：[0]=0x28、[1]=0x47
// 家族标记、[2:6]=len-1。=1 探针三轮真机被拒的根因正是把 serial 写进
// [1:5] 覆写 0x47 标记——本测试防止任何改动再破坏这些不变量。
func TestNPCDetail0x28TemplateHeaderInvariants(t *testing.T) {
	if len(npcDetail0x28Template) < 0x26 {
		t.Fatalf("template too short: %d", len(npcDetail0x28Template))
	}
	if npcDetail0x28Template[0] != 0x28 {
		t.Fatalf("template [0]=%#x, want 0x28", npcDetail0x28Template[0])
	}
	if npcDetail0x28Template[1] != 0x47 {
		t.Fatalf("template [1]=%#x, want 0x47 family marker", npcDetail0x28Template[1])
	}
	if got := binary.LittleEndian.Uint32(npcDetail0x28Template[2:]); got != uint32(len(npcDetail0x28Template)-1) {
		t.Fatalf("template [2:6]=%d, want len-1=%d", got, len(npcDetail0x28Template)-1)
	}
}

// TestNPCDetail0x28PatchFrameOnlyTouchesObjectAndTransform 替换版只允许
// 改 [14:22] obj/owner 与 [22:38] 变换四浮点；帧头与记录块必须与官服
// 模板逐字节一致（记录块语义待后续按 NPC schema 生成，当前为诊断形态）。
func TestNPCDetail0x28PatchFrameOnlyTouchesObjectAndTransform(t *testing.T) {
	transform := worldcore.Transform{X: 100.5, Y: 2.25, Z: -88.75, Orient: 3.5}
	frame := npcDetail0x28PatchFrame(0x0A0B0C0D, 0x11223344, transform)
	if len(frame) != len(npcDetail0x28Template) {
		t.Fatalf("patched frame len=%d, want %d", len(frame), len(npcDetail0x28Template))
	}
	if got := binary.LittleEndian.Uint32(frame[14:]); got != 0x0A0B0C0D {
		t.Fatalf("patched obj=%#x", got)
	}
	if got := binary.LittleEndian.Uint32(frame[18:]); got != 0x11223344 {
		t.Fatalf("patched owner=%#x", got)
	}
	for i, want := range []float32{100.5, 2.25, -88.75, 3.5} {
		if got := math.Float32frombits(binary.LittleEndian.Uint32(frame[22+4*i:])); got != want {
			t.Fatalf("patched transform[%d]=%v, want %v", i, got, want)
		}
	}
	// 帧头（[0:14]）与记录块（[38:]）逐字节保持模板原样。
	if !bytes.Equal(frame[:14], npcDetail0x28Template[:14]) {
		t.Fatal("patched frame modified header [0:14]")
	}
	if !bytes.Equal(frame[38:], npcDetail0x28Template[38:]) {
		t.Fatal("patched frame modified record block [38:]")
	}
}
