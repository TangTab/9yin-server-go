package clientdata

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestOfficialPropertyTableShape 校验嵌入表规模与关键锚点。
// 锚点来自 F 盘新版 exe 抓包的三方比对(2026-11-06):下发表 == 官方表,
// SP=467、玩家域 HP=465/MPRatio=470/InParry=471 等是位置绑定的硬语义。
func TestOfficialPropertyTableShape(t *testing.T) {
	if got := OfficialPropertyCount(); got != 2370 {
		t.Fatalf("official property table entries = %d, want 2370", got)
	}
	anchors := []struct {
		index int
		name  string
		code  int
	}{
		{34, "NpcType", 3},
		{36, "Prop36", 8},
		{88, "InParry", 3},
		{100, "LastObject", 9},
		{178, "Name", 8},
		{390, "SpeedRatio", 3},
		{414, "LastObject", 9},
		{415, "AutoSelectObject", 9},
		{463, "LogicState", 1},
		{465, "HP", 3},
		{467, "SP", 3},
		{470, "MPRatio", 3},
		{471, "InParry", 3},
		{474, "InManualParry", 3},
		{475, "ParryValue", 3},
		{580, "MaxSP", 3},
		{583, "MaxSPAdd", 3},
		{629, "BufferInfo1", 7},
		{652, "BufferInfo24", 7},
	}
	for _, anchor := range anchors {
		entry, ok := PropertyByIndex(anchor.index)
		if !ok {
			t.Fatalf("index %d absent from official table", anchor.index)
		}
		if entry.Name != anchor.name || entry.TypeCode != anchor.code {
			t.Errorf("official table[%d] = (%q, %d), want (%q, %d)", anchor.index, entry.Name, entry.TypeCode, anchor.name, anchor.code)
		}
	}
}

// TestOfficialPropertyTableDomainResolution 校验同名多处的域解析:
// 场景域首现、玩家域末现(镜像 Java PropertyTable 语义)。
func TestOfficialPropertyTableDomainResolution(t *testing.T) {
	cases := []struct {
		name        string
		sceneIndex  int
		playerIndex int
	}{
		{"LastObject", 100, 414},
		{"InParry", 88, 471},
		{"HP", 32, 465},
		{"MPRatio", 29, 470},
		{"Name", 178, 178},       // 全表唯一
		{"LogicState", 463, 463}, // 仅玩家块出现
		{"SP", 467, 467},
	}
	for _, tc := range cases {
		scene, ok := SceneProperty(tc.name)
		if !ok {
			t.Fatalf("scene property %q absent", tc.name)
		}
		if int(scene.Index) != tc.sceneIndex {
			t.Errorf("scene %q index = %d, want %d", tc.name, scene.Index, tc.sceneIndex)
		}
		player, ok := PlayerProperty(tc.name)
		if !ok {
			t.Fatalf("player property %q absent", tc.name)
		}
		if int(player.Index) != tc.playerIndex {
			t.Errorf("player %q index = %d, want %d", tc.name, player.Index, tc.playerIndex)
		}
	}
	if _, ok := SceneProperty("NoSuchPropertyName"); ok {
		t.Error("unknown name resolved in scene domain")
	}
	if _, ok := PlayerProperty("NoSuchPropertyName"); ok {
		t.Error("unknown name resolved in player domain")
	}
}

// TestOfficialPropertyTableFrameReassembly 按表重组 0x09 帧体:
// [0x09][u16 count](name\0 u8 type)*,与导出时的捕获帧逐字节一致性的 Go 侧回归。
func TestOfficialPropertyTableFrameReassembly(t *testing.T) {
	entries := OfficialPropertyEntries()
	frame := make([]byte, 3, 64*1024)
	frame[0] = 0x09
	binary.LittleEndian.PutUint16(frame[1:], uint16(len(entries)))
	for _, entry := range entries {
		frame = append(frame, entry.Name...)
		frame = append(frame, 0, byte(entry.TypeCode))
	}
	if count := binary.LittleEndian.Uint16(frame[1:]); int(count) != len(entries) {
		t.Fatalf("frame count %d != entries %d", count, len(entries))
	}
	// 逐行回读:重组体必须能精确还原每一行的 (name, type)。
	offset := 3
	for i, entry := range entries {
		end := bytes.IndexByte(frame[offset:], 0)
		if end < 0 {
			t.Fatalf("entry %d: no NUL terminator", i)
		}
		if string(frame[offset:offset+end]) != entry.Name || frame[offset+end+1] != byte(entry.TypeCode) {
			t.Fatalf("entry %d roundtrip mismatch", i)
		}
		offset = offset + end + 2
	}
	if offset != len(frame) {
		t.Fatalf("reassembly consumed %d of %d bytes", offset, len(frame))
	}
}
