package main

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// parseDetail0x28V3Fields 按官服 0x09 表列型回走 v3 明文帧字段区，
// 返回 idx→(名字, 解码值)。帧语法见 npc_detail_v3.go 头注。
func parseDetail0x28V3Fields(t *testing.T, frame []byte) map[uint16]struct {
	name  string
	i32   int32
	f32   float32
	text  string
	valid bool
} {
	t.Helper()
	if frame[0] != 0x28 || frame[1] != 0x46 {
		t.Fatalf("header = %02X %02X, want 28 46", frame[0], frame[1])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != uint32(len(frame)-1) {
		t.Fatalf("len-1=%d want %d", got, len(frame)-1)
	}
	if got := binary.LittleEndian.Uint32(frame[6:]); got != uint32(len(frame)-10) {
		t.Fatalf("outlen=%d want %d", got, len(frame)-10)
	}
	body := frame[10:]
	count := binary.LittleEndian.Uint16(body[60:])
	fields := make(map[uint16]struct {
		name  string
		i32   int32
		f32   float32
		text  string
		valid bool
	}, count)
	off := 62
	for i := 0; i < int(count); i++ {
		idx := binary.LittleEndian.Uint16(body[off:])
		entry, ok := clientdata.PropertyByIndex(int(idx))
		if !ok {
			t.Fatalf("field#%d idx=%d 不在当前表", i, idx)
		}
		off += 2
		var decoded struct {
			name  string
			i32   int32
			f32   float32
			text  string
			valid bool
		}
		decoded.name = entry.Name
		switch entry.Type {
		case clientdata.WireByte:
			decoded.i32 = int32(body[off])
			off++
		case clientdata.WireWord:
			decoded.i32 = int32(binary.LittleEndian.Uint16(body[off:]))
			off += 2
		case clientdata.WireInt32:
			decoded.i32 = int32(binary.LittleEndian.Uint32(body[off:]))
			off += 4
		case clientdata.WireFloat32:
			decoded.f32 = math.Float32frombits(binary.LittleEndian.Uint32(body[off:]))
			off += 4
		case clientdata.WireString:
			slen := binary.LittleEndian.Uint32(body[off:])
			off += 4
			raw := body[off : off+int(slen)]
			if slen == 0 || raw[slen-1] != 0 {
				t.Fatalf("field %s 串缺 NUL 尾", entry.Name)
			}
			decoded.text = string(raw[:slen-1])
			off += int(slen)
		default:
			t.Fatalf("field %s 列型 %d 未覆盖", entry.Name, entry.Type)
		}
		decoded.valid = true
		fields[idx] = decoded
	}
	if off != len(body) {
		t.Fatalf("字段区终点 %d ≠ 体长 %d（语法失步）", off, len(body))
	}
	return fields
}

// TestNPCDetail0x28V3Frame v3 明文方言全量闸：
// ①头/长度不变量（len-1、明文体长）；②变换块 13 槽（posA=posB、scale=1.25、
// 末槽原始 u32 1）；③字段区按表列型回走零失步；④身份/常量值逐项；
// ⑤model 空时身份三串+RotatePara 省略仍零失步。
func TestNPCDetail0x28V3Frame(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "official2433")
	schema := []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("Level").Index, Name: "Level", Value: clientdata.Int32Value(5)},
		{Index: clientdata.MustPlayerProperty("NpcType").Index, Name: "NpcType", Value: clientdata.Int32Value(14)},
	}
	transform := worldcore.Transform{X: 12.5, Y: 2, Z: -7.25, Orient: 1.5}
	props := officialNPCEntityProperties(schema, "npc449", "sz447")
	frame := npcDetail0x28V3Frame(0x0A0B0C0D, 0x11223344, transform, props)

	if got := binary.LittleEndian.Uint32(frame[10:]); got != 0x0A0B0C0D {
		t.Fatalf("obj=%#x", got)
	}
	if got := binary.LittleEndian.Uint32(frame[14:]); got != 0x11223344 {
		t.Fatalf("own=%#x", got)
	}
	for slot, want := range map[int]float32{0: 12.5, 1: 2, 2: -7.25, 3: 1.5, 4: 12.5, 5: 2, 6: -7.25, 7: 1.5, 8: 1.25, 9: 0, 10: 0, 11: 0} {
		base := 18 + slot*4
		if got := math.Float32frombits(binary.LittleEndian.Uint32(frame[base:])); got != want {
			t.Fatalf("变换槽 %d = %v want %v", slot, got, want)
		}
	}
	if got := binary.LittleEndian.Uint32(frame[66:]); got != 1 {
		t.Fatalf("末槽原始 u32 = %#x want 1", got)
	}

	fields := parseDetail0x28V3Fields(t, frame)
	byName := func(name string) struct {
		name  string
		i32   int32
		f32   float32
		text  string
		valid bool
	} {
		entry := clientdata.MustPlayerProperty(name)
		f, ok := fields[entry.Index]
		if !ok || !f.valid {
			t.Fatalf("字段 %s 缺席", name)
		}
		return f
	}
	if got := byName("CollideRadius").f32; math.Abs(float64(got)-0.9) > 1e-6 {
		t.Fatalf("CollideRadius=%v", got)
	}
	if got := byName("State").text; got != "stand" {
		t.Fatalf("State=%q", got)
	}
	if got := byName("Type").i32; got != 4 {
		t.Fatalf("Type=%d", got)
	}
	if got := byName("Resource").text; got != "npc\\worldnpc449" {
		t.Fatalf("Resource=%q", got)
	}
	if got := byName("ConfigID").text; got != "sz447" {
		t.Fatalf("ConfigID=%q", got)
	}
	if got := byName("Photo").text; got != "icon\\funcnpc\\worldnpc449.png" {
		t.Fatalf("Photo=%q", got)
	}
	if got := byName("NpcType").i32; got != 14 {
		t.Fatalf("NpcType=%d", got)
	}
	if got := byName("ActionSpeed").f32; got != 1.0 {
		t.Fatalf("ActionSpeed=%v", got)
	}
	if got := byName("AIEffect").text; got != "nil" {
		t.Fatalf("AIEffect=%q", got)
	}
	if got := byName("Level").i32; got != 5 {
		t.Fatalf("Level=%d", got)
	}
	if got := byName("IDCard").text; got != "null,null,null,null,null" {
		t.Fatalf("IDCard=%q", got)
	}
	if got := byName("WeaponMode").text; got != "hand1_1" {
		t.Fatalf("WeaponMode=%q", got)
	}
	for _, name := range []string{"HPRatio", "ParryRatio", "HitHPRatio"} {
		if got := byName(name).i32; got != 100 {
			t.Fatalf("%s=%d", name, got)
		}
	}
}

func TestNPCDetail0x28V3FrameNoIdentity(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "official2433")
	props := officialNPCEntityProperties(nil, "", "")
	frame := npcDetail0x28V3Frame(1, 2, worldcore.Transform{}, props)
	fields := parseDetail0x28V3Fields(t, frame)
	for _, name := range []string{"Resource", "ConfigID", "Photo", "RotatePara"} {
		entry := clientdata.MustPlayerProperty(name)
		if _, ok := fields[entry.Index]; ok {
			t.Fatalf("空身份时 %s 不应下发", name)
		}
	}
	if got := binary.LittleEndian.Uint16(frame[70:]); got != uint16(len(props)) {
		t.Fatalf("fieldcount=%d want %d", got, len(props))
	}
}
