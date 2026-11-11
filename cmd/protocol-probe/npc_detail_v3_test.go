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
	frame := npcDetail0x28V3Frame(0x0A0B0C0D, 0x11223344, transform, nil, props)

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
	if got := byName("CollideRadius").f32; math.Abs(float64(got)-0.905) > 1e-6 {
		t.Fatalf("CollideRadius=%v（schema 缺席应回落 0.905）", got)
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
	// #0 形官服常量族（官方 #0 帧 26 字段逐位对齐）。
	for name, want := range map[string]int32{
		"CursorShape": 2, "MPRatio": 11, "HasKarma": 1, "Character": 1,
		"KarmaLevel": 3, "OriginPara1": 7, "OriginPara2": 1, "OriginPara3": 5,
	} {
		if got := byName(name).i32; got != want {
			t.Fatalf("%s=%d want %d", name, got, want)
		}
	}
	// #1 形专属行不得出现在普通形态。
	for _, name := range []string{"WeaponName", "Fixed", "FixedBeHitAct", "SelfForceID"} {
		entry := clientdata.MustPlayerProperty(name)
		if _, ok := fields[entry.Index]; ok {
			t.Fatalf("普通形态不应下发 %s", name)
		}
	}
	if got := binary.LittleEndian.Uint16(frame[70:]); got != 26 {
		t.Fatalf("fieldcount=%d want 26（官方 #0 形）", got)
	}
}

// TestNPCDetail0x28V3FrameArmed #1 形（带武器 NPC，官方 #1 帧方言）：
// 判据=schema WeaponName 非空；Fixed 紧随 CollideRadius、WeaponName 在
// WeaponMode 前、FixedBeHitAct/SelfForceID 在后；Photo/CursorShape/
// MPRatio/karma/origin 族不发；BufferInfo×2 活 buff 槽省略（官方 23 字段
// −2=21）。
func TestNPCDetail0x28V3FrameArmed(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "official2433")
	schema := []clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("Level").Index, Name: "Level", Value: clientdata.Int32Value(160)},
		{Index: clientdata.MustPlayerProperty("NpcType").Index, Name: "NpcType", Value: clientdata.Int32Value(4)},
		{Index: clientdata.MustPlayerProperty("WeaponName").Index, Name: "WeaponName", Value: clientdata.StringValue("s_r_255")},
		{Index: clientdata.MustPlayerProperty("Fixed").Index, Name: "Fixed", Value: clientdata.Int32Value(1)},
	}
	props := officialNPCEntityProperties(schema, "npc793", "Boss_weibing_kill_01")
	frame := npcDetail0x28V3Frame(0x0E00431B, 0x4422, worldcore.Transform{X: 430.7, Y: 0.5637, Z: 693.2, Orient: 5.643}, nil, props)
	fields := parseDetail0x28V3Fields(t, frame)

	wantText := map[string]string{
		"Resource":   "npc\\worldnpc793",
		"ConfigID":   "Boss_weibing_kill_01",
		"WeaponName": "s_r_255",
		"WeaponMode": "hand1_1",
		"State":      "stand",
	}
	wantInt := map[string]int32{
		"Type": 4, "NpcType": 4, "Level": 160, "BossLevel": 1,
		"Fixed": 1, "FixedBeHitAct": 1, "SelfForceID": 999,
		"HPRatio": 100, "ParryRatio": 100, "HitHPRatio": 100,
	}
	for name, want := range wantText {
		entry := clientdata.MustPlayerProperty(name)
		f, ok := fields[entry.Index]
		if !ok || !f.valid {
			t.Fatalf("字段 %s 缺席", name)
		}
		if f.text != want {
			t.Fatalf("%s=%q want %q", name, f.text, want)
		}
	}
	for name, want := range wantInt {
		entry := clientdata.MustPlayerProperty(name)
		f, ok := fields[entry.Index]
		if !ok || !f.valid {
			t.Fatalf("字段 %s 缺席", name)
		}
		if f.i32 != want {
			t.Fatalf("%s=%d want %d", name, f.i32, want)
		}
	}
	// #0 形专属行不得出现在带武器形态。
	for _, name := range []string{"Photo", "CursorShape", "MPRatio", "HasKarma", "KarmaLevel", "OriginPara1"} {
		entry := clientdata.MustPlayerProperty(name)
		if _, ok := fields[entry.Index]; ok {
			t.Fatalf("带武器形态不应下发 %s", name)
		}
	}
	if got := binary.LittleEndian.Uint16(frame[70:]); got != 21 {
		t.Fatalf("fieldcount=%d want 21（官方 #1 形 23 − BufferInfo×2）", got)
	}
}

func TestNPCDetail0x28V3FrameForkTable(t *testing.T) {
	// 2026-11-11 真机回归：fork 表 + NINEYIN_NPC_DETAIL_0X28=1 曾致首帧
	// NPC 0x0D 后会话静默关闭（疑 v3 属性集构造 panic）。双方言都必须走通；
	// 2026-11-11 晚补形态方言后再加全帧回走（fork 线型如 u8 也要零失步）。
	clientdata.SwapPropertyTableForTest(t, "fork")
	props := officialNPCEntityProperties([]clientdata.IndexedProperty{
		{Index: clientdata.MustPlayerProperty("Level").Index, Name: "Level", Value: clientdata.Int32Value(5)},
	}, "npc449", "sz447")
	frame := npcDetail0x28V3Frame(1, 2, worldcore.Transform{}, nil, props)
	if frame[1] != 0x46 {
		t.Fatalf("mode=%#x", frame[1])
	}
	fields := parseDetail0x28V3Fields(t, frame)
	for _, name := range []string{"CollideRadius", "State", "Resource", "ConfigID", "Photo", "WeaponMode", "CursorShape", "MPRatio", "OriginPara3"} {
		entry := clientdata.MustPlayerProperty(name)
		if _, ok := fields[entry.Index]; !ok {
			t.Fatalf("fork 方言字段 %s 缺席", name)
		}
	}
}

func TestNPCDetail0x28V3FrameNoIdentity(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "official2433")
	props := officialNPCEntityProperties(nil, "", "")
	frame := npcDetail0x28V3Frame(1, 2, worldcore.Transform{}, nil, props)
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

// TestNPCDetail0x28V3CollideSchemaPassThrough 怪物/Boss 形 CollideRadius
// 按体型的 schema 真值直通（npc.pcapng 0x46 定版：怪物 0.915/Boss 1.0）。
func TestNPCDetail0x28V3CollideSchemaPassThrough(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "official2433")
	for _, want := range []float32{0.5, 0.915, 1.0} {
		schema := []clientdata.IndexedProperty{
			{Index: clientdata.MustPlayerProperty("CollideRadius").Index, Name: "CollideRadius", Value: clientdata.Float32Value(want)},
		}
		props := officialNPCEntityProperties(schema, "npc793", "Boss_weibing_kill_01")
		found := false
		for _, p := range props {
			if p.Name == "CollideRadius" {
				found = true
				if p.Value.F32 != want {
					t.Fatalf("CollideRadius=%v want schema 真值 %v", p.Value.F32, want)
				}
			}
		}
		if !found {
			t.Fatalf("CollideRadius 行缺席（want=%v）", want)
		}
	}
}

// TestNPCDetail0x28V3MovingTarget 走动物化 posB=目标位（npc.pcapng 官服
// 0x46 帧 8/8 A≠B，B=段末目的地）；nil target 保持 A=B（静立/停留方言）。
func TestNPCDetail0x28V3MovingTarget(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "official2433")
	props := officialNPCEntityProperties(nil, "", "")
	transform := worldcore.Transform{X: 100, Y: 1, Z: 200, Orient: 0.5}
	slot := func(frame []byte, i int) float32 {
		return math.Float32frombits(binary.LittleEndian.Uint32(frame[18+i*4:]))
	}

	moving := npcDetail0x28V3Frame(7, 8, transform,
		&worldcore.Transform{X: 110, Y: 1.5, Z: 220, Orient: 0.9}, props)
	for i, want := range []float32{100, 1, 200, 0.5} {
		if got := slot(moving, i); got != want {
			t.Fatalf("走动 posA 槽 %d=%v want %v", i, got, want)
		}
	}
	for i, want := range []float32{110, 1.5, 220, 0.9} {
		if got := slot(moving, 4+i); got != want {
			t.Fatalf("走动 posB 槽 %d=%v want %v", i, got, want)
		}
	}

	staticFrame := npcDetail0x28V3Frame(7, 8, transform, nil, props)
	for i, want := range []float32{100, 1, 200, 0.5} {
		if got := slot(staticFrame, 4+i); got != want {
			t.Fatalf("静立 posB 槽 %d=%v want 与 posA 同值 %v", i, got, want)
		}
	}
	if slot(staticFrame, 8) != npcDetail0x28BaseScale {
		t.Fatalf("scale 槽=%v want 基线 %v", slot(staticFrame, 8), npcDetail0x28BaseScale)
	}
}
