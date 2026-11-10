package main

import (
	"bytes"
	"encoding/binary"
	"testing"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// TestNPCDetail0x28Generator 生成器闸：身份串重写后帧仍自洽
// （len-1 更新、idx 字段可再定位、obj/owner/变换落位），模板常量段不动。
func TestNPCDetail0x28Generator(t *testing.T) {
	transform := worldcore.Transform{X: 12.5, Y: 2, Z: -7.25, Orient: 1.5}
	frame := npcDetail0x28Frame(0x0A0B0C0D, 0x11223344, transform, `npc\worldnpc059`, "gnnpc_2021zyb_001")
	if frame[0] != 0x28 || frame[1] != 0x47 {
		t.Fatalf("header broken: %x %x", frame[0], frame[1])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != uint32(len(frame)-1) {
		t.Fatalf("len-1=%d want %d", got, len(frame)-1)
	}
	if got := binary.LittleEndian.Uint32(frame[14:]); got != 0x0A0B0C0D {
		t.Fatalf("obj=%#x", got)
	}
	if got := binary.LittleEndian.Uint32(frame[18:]); got != 0x11223344 {
		t.Fatalf("owner=%#x", got)
	}
	if !bytes.Contains(frame, []byte("gnnpc_2021zyb_001\x00")) {
		t.Fatal("ConfigID identity missing")
	}
	if !bytes.Contains(frame, []byte("npc\\worldnpc059\x00")) {
		t.Fatal("Resource identity missing")
	}
	if !bytes.Contains(frame, []byte("icon\\funcnpc\\worldnpc059.png\x00")) {
		t.Fatal("Photo derived path missing")
	}
	if !bytes.Contains(frame, []byte("stand\x00")) {
		t.Fatal("State stand missing")
	}
	if bytes.Contains(frame, []byte("ty_bukuai_guard")) {
		t.Fatal("template NPC identity leaked (=3 崩溃根因)")
	}
	// 再定位：重写后的字段应仍可按模式找到（幂等可用）。
	again := rewrite0x28StringField(frame, 183, "walk")
	if !bytes.Contains(again, []byte("walk\x00")) || bytes.Contains(again, []byte("stand\x00")) {
		t.Fatal("re-locate/rewrite State failed")
	}
}
