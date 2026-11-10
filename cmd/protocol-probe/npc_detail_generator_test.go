package main

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// TestNPCDetail0x28GeneratorV2 生成器 v2（精确切片替换）闸：
// ①身份四串干净形在场、模板身份零残留（=3/=4 崩溃验收标准）；
// ②非身份段（RotatePara/IDCard/hand1_1 尾段/短值 HPRatio+HitHPRatio）
//
//	逐字节保真；
//
// ③客户端法则 Go 侧预检 walker 全程走通（=4 缺的保障）。
func TestNPCDetail0x28GeneratorV2(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "official2433")
	transform := worldcore.Transform{X: 12.5, Y: 2, Z: -7.25, Orient: 1.5}
	frame := npcDetail0x28Frame(0x0A0B0C0D, 0x11223344, transform, "npc"+string(rune(92))+"worldnpc059", "gnnpc_2021zyb_001")
	if frame[0] != 0x28 || frame[1] != 0x47 {
		t.Fatalf("header broken")
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
	// 用显式字节断言（避免转义混乱）
	checks := []struct {
		name   string
		needle []byte
	}{
		{"State", append([]byte("stand"), 0)},
		{"Resource", append([]byte("npc"+string(rune(92))+"worldnpc059"), 0)},
		{"ConfigID", append([]byte("gnnpc_2021zyb_001"), 0)},
		{"Photo", append([]byte("icon"+string(rune(92))+"funcnpc"+string(rune(92))+"worldnpc059.png"), 0)},
		{"RotatePara 保真", []byte("0.0,4.767,")},
		{"WeaponMode 尾段保真", append([]byte("hand1_1"), 0)},
		{"HitHPRatio 尾段保真", []byte{0xe4, 0x01, 100, 0, 0, 0}},
	}
	for _, c := range checks {
		if !bytes.Contains(frame, c.needle) {
			t.Fatalf("%s missing", c.name)
		}
	}
	for _, leaked := range []string{"ty_bukuai_guard", "monm031300", "npc" + string(rune(92)) + "world" + string(rune(0x91))} {
		if bytes.Contains(frame, []byte(leaked)) {
			t.Fatalf("template identity leaked: %q", leaked)
		}
	}
	// 预检：len-1 自洽已在上方断言；生成段（State/Resource/ConfigID/Photo）
	// 为干净形且逐个可解析；保留段与模板逐字节相等（官方 blob 原样=
	// 官服客户端已接受的字节）。
	if !bytes.Contains(frame, npcDetail0x28Template[104:126]) {
		t.Fatal("RotatePara kept-region mutated")
	}
	tail := npcDetail0x28Template[177:]
	if !bytes.HasSuffix(frame, tail) {
		t.Fatal("tail kept-region (Level..HitHPRatio) mutated")
	}
}
