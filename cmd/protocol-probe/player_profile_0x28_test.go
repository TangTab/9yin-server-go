package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// profileFieldWireSize 官方表列型 → 线宽（与解码器方言一致；t6=8B f64）。
func profileFieldWireSize(typeCode int) int {
	switch typeCode {
	case 1:
		return 1
	case 2:
		return 2
	case 3:
		return 4
	case 4, 6:
		return 8
	case 5:
		return 4
	default:
		return -1
	}
}

// TestPlayerProfile0x28Envelope 帧头不变量 + 逐字段走查精确落在帧尾
//（与官服样本同一校验口径：len-1/bodylen/count/字段区按表列型走完恰尽）。
func TestPlayerProfile0x28Envelope(t *testing.T) {
	// 官服方言下 41 字段全量解析（fork 表家园/VIP/节日域头行缺席=11 字段
	// 稀疏形，另由 TestPlayerProfile0x28ForkSparseSmoke 覆盖）。
	clientdata.SwapPropertyTableForTest(t, "official2433")
	pos := role.Position{X: 660.6, Y: 33.6, Z: 299.5, Orient: 1.578}
	frame, count := playerProfile0x28Frame(0x1105C0E3, 0x03DEfBAE, pos, nil)
	if frame[0] != 0x28 || frame[1] != 0x46 {
		t.Fatalf("header = %#x %#x, want 28 46", frame[0], frame[1])
	}
	if got := binary.LittleEndian.Uint32(frame[2:]); got != uint32(len(frame)-1) {
		t.Fatalf("len-1 = %d, want %d (len=%d)", got, len(frame)-1, len(frame))
	}
	body := frame[10:]
	if got := binary.LittleEndian.Uint32(frame[6:]); got != uint32(len(body)) {
		t.Fatalf("bodylen = %d, want %d", got, len(body))
	}
	if obj, own := binary.LittleEndian.Uint32(body[0:]), binary.LittleEndian.Uint32(body[4:]); obj != 0x1105C0E3 || own != 0x03DEfBAE {
		t.Fatalf("obj/own = %#x/%#x", obj, own)
	}
	// 变换块：posA=落点、posB/scale 全零（玩家形官服方言）。
	for i, want := range []float32{660.6, 33.6, 299.5, 1.578} {
		if got := math.Float32frombits(binary.LittleEndian.Uint32(body[8+4*i:])); got != want {
			t.Fatalf("posA[%d] = %g, want %g", i, got, want)
		}
	}
	for i := 4; i < 12; i++ {
		if got := binary.LittleEndian.Uint32(body[8+4*i:]); got != 0 {
			t.Fatalf("transform slot %d = %#x, want 0 (posB/scale/保留)", i, got)
		}
	}
	if got := binary.LittleEndian.Uint32(body[8+4*12:]); got != 1 {
		t.Fatalf("transform slot 12 = %#x, want 1 (尾 u32 标志位)", got)
	}
	// 字段区走查：count 字段按表列型走完恰落帧尾（官服样本同口径）。
	got := binary.LittleEndian.Uint16(body[60:])
	if int(got) != count {
		t.Fatalf("count = %d, want %d", got, count)
	}
	if count != len(playerProfilePropSpecs) {
		t.Fatalf("official2433 field count = %d, want %d（全量清单应无缺席）", count, len(playerProfilePropSpecs))
	}
	off := 62
	seen := map[string]bool{}
	for i := 0; i < count; i++ {
		idx := binary.LittleEndian.Uint16(body[off:])
		entry, ok := clientdata.PropertyByIndex(int(idx))
		if !ok {
			t.Fatalf("field %d idx=%d 不在当前 0x09 表", i, idx)
		}
		off += 2
		seen[entry.Name] = true
		var size int
		if entry.TypeCode == 7 || entry.TypeCode == 8 {
			slen := int(binary.LittleEndian.Uint32(body[off:]))
			off += 4
			if body[off+slen-1] != 0 {
				t.Fatalf("field %d %s 串值未按官方方言收 NUL", i, entry.Name)
			}
			size = slen
		} else {
			size = profileFieldWireSize(entry.TypeCode)
		}
		if size < 0 {
			t.Fatalf("field %d %s 列型 %d 不可走查", i, entry.Name, entry.TypeCode)
		}
		off += size
	}
	if off != len(body) {
		t.Fatalf("walk end %d, want %d", off, len(body))
	}
	// v1 语义锚（当前表缺席时跳过对应断言，不因方言档位失败）。
	if entry, ok := clientdata.PlayerProperty("Name"); ok && seen["Name"] {
		off := bytes.Index(body, binary.LittleEndian.AppendUint16(nil, entry.Index))
		slen := int(binary.LittleEndian.Uint32(body[off+2:]))
		if !bytes.Equal(body[off+6:off+6+slen], playerProfileNameWide) {
			t.Fatalf("Name 官服遮蔽常量不符")
		}
	}
	if entry, ok := clientdata.PlayerProperty("HomeFengShuiDate"); ok && seen["HomeFengShuiDate"] {
		off := bytes.Index(body, binary.LittleEndian.AppendUint16(nil, entry.Index))
		if got := math.Float64frombits(binary.LittleEndian.Uint64(body[off+2:])); got != 42952.0 {
			t.Fatalf("HomeFengShuiDate = %g, want 42952 (8B f64 官服方言)", got)
		}
	}
	if entry, ok := clientdata.PlayerProperty("GroupID"); ok && seen["GroupID"] {
		off := bytes.Index(body, binary.LittleEndian.AppendUint16(nil, entry.Index))
		if got := binary.LittleEndian.Uint32(body[off+2:]); got != 0xFFFFFFFF {
			t.Fatalf("GroupID = %#x, want -1", got)
		}
	}
}

// TestPlayerProfile0x28FollowsActiveTable 双方言换表跟随：official2433
// 下 Name 落官服原号 178、HomeFengShuiDate 落 6（登陆-all #132 实证位）。
func TestPlayerProfile0x28FollowsActiveTable(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "official2433")
	frame, _ := playerProfile0x28Frame(1, 2, role.Position{}, nil)
	body := frame[10:]
	count := int(binary.LittleEndian.Uint16(body[60:]))
	off := 62
	got := map[string]uint16{}
	for i := 0; i < count; i++ {
		idx := binary.LittleEndian.Uint16(body[off:])
		entry, _ := clientdata.PropertyByIndex(int(idx))
		got[entry.Name] = idx
		off += 2
		if entry.TypeCode == 7 || entry.TypeCode == 8 {
			off += 4 + int(binary.LittleEndian.Uint32(body[off:]))
		} else {
			off += profileFieldWireSize(entry.TypeCode)
		}
	}
	for name, want := range map[string]uint16{
		"Name": 178, "GroupID": 180, "HomeFengShuiDate": 6, "Type": 181,
		"Photo": 184, "Face": 185, "Hair": 188, "RelivePositon": 195,
	} {
		if got[name] != want {
			t.Fatalf("official2433 %s=%d, want %d", name, got[name], want)
		}
	}
}

// TestPlayerProfile0x28ForkSparseSmoke fork 表稀疏形：家园/VIP/节日域头行
// 缺席时按官服稀疏语义跳过，帧不变量（len-1/bodylen/走查恰尽）仍成立。
func TestPlayerProfile0x28ForkSparseSmoke(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "fork")
	frame, count := playerProfile0x28Frame(1, 2, role.Position{}, nil)
	if count == 0 || count >= len(playerProfilePropSpecs) {
		t.Fatalf("fork count = %d, want 1..%d 稀疏形", count, len(playerProfilePropSpecs)-1)
	}
	body := frame[10:]
	got := int(binary.LittleEndian.Uint16(body[60:]))
	if got != count {
		t.Fatalf("count hdr = %d, want %d", got, count)
	}
	off := 62
	for i := 0; i < count; i++ {
		idx := int(binary.LittleEndian.Uint16(body[off:]))
		entry, ok := clientdata.PropertyByIndex(idx)
		if !ok {
			t.Fatalf("field %d idx=%d 不在 fork 表", i, idx)
		}
		off += 2
		if entry.TypeCode == 7 || entry.TypeCode == 8 {
			off += 4 + int(binary.LittleEndian.Uint32(body[off:]))
		} else {
			off += profileFieldWireSize(entry.TypeCode)
		}
	}
	if off != len(body) {
		t.Fatalf("fork walk end %d, want %d", off, len(body))
	}
}

// TestPlayerProfile0x28SkillRows 技能行 v2：每技能 17 列（当前表缺席列
// 稀疏跳过），帧走查仍恰尽；ConfigID 行串回读、重复 idx 合法。
func TestPlayerProfile0x28SkillRows(t *testing.T) {
	clientdata.SwapPropertyTableForTest(t, "official2433")
	skills := []learnedSkillView{
		{configID: "CS_sl_djgq04", staticData: 3932, level: 1, maxLevel: 4, total: 20400, pauseTime: 30000},
		{configID: "zs_normal_02", staticData: 77, level: 3, maxLevel: 9, total: 500, pauseTime: 0},
	}
	frame, count := playerProfile0x28Frame(1, 2, role.Position{}, skills)
	body := frame[10:]
	if int(binary.LittleEndian.Uint16(body[60:])) != count {
		t.Fatalf("count hdr mismatch")
	}
	off := 62
	type seenField struct {
		idx  uint16
		name string
		val  string
	}
	var fields []seenField
	for i := 0; i < count; i++ {
		idx := binary.LittleEndian.Uint16(body[off:])
		entry, ok := clientdata.PropertyByIndex(int(idx))
		if !ok {
			t.Fatalf("field %d idx=%d 不在表", i, idx)
		}
		off += 2
		val := ""
		if entry.TypeCode == 7 || entry.TypeCode == 8 {
			slen := int(binary.LittleEndian.Uint32(body[off:]))
			off += 4
			val = string(body[off : off+slen])
			off += slen
		} else {
			off += profileFieldWireSize(entry.TypeCode)
		}
		fields = append(fields, seenField{idx, entry.Name, val})
	}
	if off != len(body) {
		t.Fatalf("walk end %d, want %d", off, len(body))
	}
	if count != len(playerProfilePropSpecs)+2*len(playerProfileSkillRowNames) {
		t.Fatalf("count=%d, want %d（41 基础 + 2 行 ×17）", count, len(playerProfilePropSpecs)+2*len(playerProfileSkillRowNames))
	}
	// 两行 ConfigID 均在且串值含尾 NUL。
	configs := 0
	for _, f := range fields {
		if f.name == "ConfigID" && (f.val == "CS_sl_djgq04\x00" || f.val == "zs_normal_02\x00") {
			configs++
		}
	}
	if configs != 2 {
		t.Fatalf("ConfigID rows=%d, want 2", configs)
	}
}
