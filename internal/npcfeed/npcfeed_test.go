package npcfeed

// npcfeed_test.go 真档集成测试（构建产物在仓根 testdata/ini，cwd 探测
// ../../testdata 命中）+ 帧配方/闸门单测。

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// testRoot 解析测试数据根（仓内构建产物）。
func testRoot(t *testing.T) string {
	t.Helper()
	root := ResolveRoot()
	if root == "" {
		t.Skip("数据根缺席（testdata/ini 不在探测链上）")
	}
	return root
}

func TestBuildCatalogBorn03(t *testing.T) {
	root := testRoot(t)
	spawns, stats, err := BuildCatalog(root, "born03")
	if err != nil {
		t.Fatalf("BuildCatalog: %v", err)
	}
	if len(spawns) == 0 {
		t.Fatal("born03 目录为空（真档缺席或联查全败——见 stats 样本）")
	}
	seen := map[uint32]bool{}
	for i, s := range spawns {
		if s.ObjectID>>24 != 0x11 {
			t.Fatalf("条 %d objectID %x 不在 0x11 段", i, s.ObjectID)
		}
		if s.OwnerID>>16 != 0x0066 {
			t.Fatalf("条 %d ownerID %x 不在 0x66xxxx 段", i, s.OwnerID)
		}
		if seen[s.ObjectID] {
			t.Fatalf("objectID %x 重复", s.ObjectID)
		}
		seen[s.ObjectID] = true
		if s.Script == "EventNpc" {
			t.Fatalf("条 %d (%s) EventNpc 漏闸", i, s.ConfigID)
		}
		if s.Template["Resource"] == "" {
			t.Fatalf("条 %d (%s) 模板无 Resource", i, s.ConfigID)
		}
	}
	if stats.CombatHit == 0 {
		t.Error("born03 战斗联查 0 命中（NpcTemplate.xml/FightID 链断裂？）")
	}
	t.Logf("born03: 摆放 %d 联查 %d 战斗命中 %d 闸拦 %d；样本 %v",
		stats.Placements, stats.Resolved, stats.CombatHit, stats.Skipped, stats.Samples)
}

func TestBuildCatalogUnknownScene(t *testing.T) {
	root := testRoot(t)
	if _, _, err := BuildCatalog(root, "scene_no_such"); err == nil {
		t.Fatal("未知场景应报错")
	}
}

func TestCombatLayerOverrides(t *testing.T) {
	ResetStore()
	defer ResetStore()
	s := Spawn{
		ObjectID: ObjectIDAt(0), OwnerID: OwnerIDOf(ObjectIDAt(0)),
		ConfigID: "test",
		Template: map[string]string{"ID": "test", "Resource": "npc\\t", "Level": "2"},
		Combat:   &CombatRow{ID: "114", Attrs: map[string]string{"ID": "114", "MaxLife": "108678", "MaxMana": "97810", "Level": "70"}},
	}
	props := s.OrderedProperties()
	byName := map[string]PropValue{}
	for _, p := range props {
		byName[p.Name] = p.Value
	}
	lv, ok := byName["Level"]
	if !ok {
		t.Fatal("Level 缺席官方表场景域")
	}
	var levelGot int32
	switch lv.Type {
	case clientdata.WireByte:
		levelGot = int32(lv.U8)
	case clientdata.WireWord:
		levelGot = int32(lv.U16)
	case clientdata.WireInt32:
		levelGot = lv.I32
	}
	if levelGot != 70 {
		t.Errorf("Level 战斗真值未生效: %d (type=%v), want 70", levelGot, lv.Type)
	}
	maxHP, ok := byName["MaxHP"]
	if !ok {
		t.Fatal("MaxHP 缺席官方表（默认层未注入？）")
	}
	if maxHP.I32 != 108678 {
		t.Errorf("MaxHP = %d, want 108678", maxHP.I32)
	}
	if hp := byName["HP"]; hp.I32 != 108678 {
		t.Errorf("HP = %d, want 108678", hp.I32)
	}
}

func TestEligibleGates(t *testing.T) {
	cases := []struct {
		name   string
		p      Placement
		script string
		want   bool
	}{
		{"普通", Placement{Attrs: map[string]string{}}, "CommonNpc", true},
		{"EventNpc 类", Placement{Attrs: map[string]string{}}, "EventNpc", false},
		{"任务闸", Placement{Attrs: map[string]string{"TaskSubId": "11034"}}, "AttackNpc", false},
		{"子场景+战斗类", Placement{Attrs: map[string]string{"SubSceneNo": "8961"}}, "AttackNpc", false},
		{"子场景+平民", Placement{Attrs: map[string]string{"SubSceneNo": "8961"}}, "CommonNpc", true},
		{"类名大小写", Placement{Attrs: map[string]string{}}, " eventnpc ", false},
	}
	for _, c := range cases {
		if got := eligible(c.p, c.script); got != c.want {
			t.Errorf("%s: eligible = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestFrame0DShape(t *testing.T) {
	s := Spawn{
		ObjectID: ObjectIDAt(1), OwnerID: OwnerIDOf(ObjectIDAt(1)),
		X: 1.5, Y: 2.5, Z: 3.5, Orient: 0.25,
		ConfigID: "t1",
		Template: map[string]string{"ID": "t1", "Resource": `npc\worldnpc001`, "NpcType": "62", "CollideRadius": "0.9"},
	}
	frame, err := s.Frame0D()
	if err != nil {
		t.Fatalf("Frame0D: %v", err)
	}
	if frame[0] != 0x0D {
		t.Fatalf("op = %#x, want 0x0D", frame[0])
	}
	if got := binary.LittleEndian.Uint32(frame[1:]); got != s.ObjectID {
		t.Errorf("objectID = %x", got)
	}
	if got := binary.LittleEndian.Uint32(frame[5:]); got != s.OwnerID {
		t.Errorf("ownerID = %x", got)
	}
	count := binary.LittleEndian.Uint16(frame[61:63])
	if int(count) < 16 {
		t.Fatalf("属性数 %d < 16（官方块缺席？）", count)
	}
	// 官方块必备行按名在场：解码全部值对找 State/Type。
	hasState, hasType := false, false
	off := 63
	for i := 0; i < int(count); i++ {
		idx := binary.LittleEndian.Uint16(frame[off:])
		off += 2
		entry, ok := clientdata.PropertyByIndex(int(idx))
		if !ok {
			t.Fatalf("值对 %d 索引 %d 不在官方表", i, idx)
		}
		switch entry.Name {
		case "State":
			hasState = true
		case "Type":
			hasType = true
		}
		n := 0
		switch entry.Type {
		case clientdata.WireByte:
			n = 1
		case clientdata.WireWord:
			n = 2
		case clientdata.WireInt32, clientdata.WireFloat32:
			n = 4
		case clientdata.WireInt64, clientdata.WireObject:
			n = 8
		case clientdata.WireString, clientdata.WireWideString:
			n = int(binary.LittleEndian.Uint32(frame[off:]))
			off += 4
		}
		off += n
	}
	if !hasState || !hasType {
		t.Errorf("官方块 State/Type 缺席（state=%v type=%v）", hasState, hasType)
	}
	if off != len(frame) {
		t.Errorf("值对走查终点 %d ≠ 帧长 %d（编码不一致）", off, len(frame))
	}
}

func TestFrame0x28Shape(t *testing.T) {
	s := Spawn{
		ObjectID: ObjectIDAt(2), OwnerID: OwnerIDOf(ObjectIDAt(2)),
		X: 10, Y: 20, Z: 30, Orient: 1,
		ConfigID: "t2",
		Template: map[string]string{"ID": "t2", "Resource": `npc\funcnpc001`, "NpcType": "62"},
		Combat:   &CombatRow{ID: "9", Attrs: map[string]string{"ID": "9", "Level": "6", "MaxLife": "77"}},
	}
	frame := s.Frame0x28()
	if frame[0] != 0x28 || frame[1] != 0x46 {
		t.Fatalf("op/mode = %#x/%#x, want 0x28/0x46", frame[0], frame[1])
	}
	if got := binary.LittleEndian.Uint32(frame[6:]); int(got) != len(frame)-10 {
		t.Errorf("体长 = %d, want %d", got, len(frame)-10)
	}
	if seg := binary.LittleEndian.Uint32(frame[10:]); seg>>24 != 0x11 {
		t.Errorf("obj 段异常 %x", seg)
	}
}

func TestOfficialPropsBoxDialect(t *testing.T) {
	box := Spawn{Template: map[string]string{"ID": "b", "NpcType": "161"}}
	props := officialProps(box)
	found := map[string]bool{}
	for _, p := range props {
		found[p.Name] = true
	}
	if found["CollideRadius"] {
		t.Error("宝箱族不应发 CollideRadius")
	}
	if !found["OpenRange"] {
		t.Error("宝箱族尾行应发 OpenRange=2.0")
	}

	gather := Spawn{Template: map[string]string{"ID": "g", "ConfigID": "GatherClone001"}}
	props = officialProps(gather)
	found = map[string]bool{}
	for _, p := range props {
		found[p.Name] = true
	}
	if found["CollideRadius"] {
		t.Error("采集族不应发 CollideRadius")
	}
	if !found["GatherRange"] {
		t.Error("采集族尾行应发 GatherRange=3.0")
	}
}

func TestFilterByRadius(t *testing.T) {
	spawns := []Spawn{
		{ObjectID: ObjectIDAt(0), X: 0, Z: 0},
		{ObjectID: ObjectIDAt(1), X: 10, Z: 0},
		{ObjectID: ObjectIDAt(2), X: 100, Z: 0},
	}
	got := FilterByRadius(spawns, 0, 0, 50, 0)
	if len(got) != 2 {
		t.Fatalf("半径 50 命中 %d, want 2", len(got))
	}
	got = FilterByRadius(spawns, 0, 0, 0, 1)
	if len(got) != 1 {
		t.Fatalf("上限 1 截断 %d, want 1", len(got))
	}
	if got = FilterByRadius(spawns, 0, 0, 0, 0); len(got) != 3 {
		t.Fatalf("全量 %d, want 3", len(got))
	}
}

func TestFramesBothForms(t *testing.T) {
	spawns := []Spawn{{
		ObjectID: ObjectIDAt(3), OwnerID: OwnerIDOf(ObjectIDAt(3)),
		ConfigID: "f", Template: map[string]string{"ID": "f", "Resource": `npc\x`},
	}}
	as28, err := Frames(spawns, true)
	if err != nil || as28[0][0] != 0x28 {
		t.Fatalf("0x28 形态异常: %v", err)
	}
	as0d, err := Frames(spawns, false)
	if err != nil || as0d[0][0] != 0x0D {
		t.Fatalf("0x0D 形态异常: %v", err)
	}
}

// decode0x28Fields 解 0x28 mode=0x46 体值对（idx 按官方表反查名）。
func decode0x28Fields(t *testing.T, frame []byte) map[string][]byte {
	t.Helper()
	if frame[0] != 0x28 || frame[1] != 0x46 {
		t.Fatalf("非 0x28/0x46 帧")
	}
	body := frame[10:]
	count := int(binary.LittleEndian.Uint16(body[60:62]))
	fields := map[string][]byte{}
	off := 62
	for i := 0; i < count; i++ {
		idx := binary.LittleEndian.Uint16(body[off:])
		off += 2
		entry, ok := clientdata.PropertyByIndex(int(idx))
		if !ok {
			t.Fatalf("值对 %d 索引 %d 不在官方表", i, idx)
		}
		n := 0
		switch entry.Type {
		case clientdata.WireByte:
			n = 1
		case clientdata.WireWord:
			n = 2
		case clientdata.WireInt32, clientdata.WireFloat32:
			n = 4
		case clientdata.WireInt64, clientdata.WireObject:
			n = 8
		case clientdata.WireString, clientdata.WireWideString:
			n = int(binary.LittleEndian.Uint32(body[off:]))
			off += 4
		}
		fields[entry.Name] = body[off : off+n]
		off += n
	}
	if off != len(body) {
		t.Fatalf("值对走查终点 %d ≠ 体长 %d", off, len(body))
	}
	return fields
}

func TestFrame0x28V3Dialect(t *testing.T) {
	// #0 普通形（WeaponName 空）：CursorShape/MPRatio/karma/origin 族在场。
	civilian := Spawn{
		ObjectID: ObjectIDAt(0), OwnerID: OwnerIDOf(ObjectIDAt(0)),
		ConfigID: "FuncNpc00051",
		Template: map[string]string{
			"ID": "FuncNpc00051", "Resource": `npc\funcnpc001`, "NpcType": "62",
			"Level": "2", "CollideRadius": "0.905", "CursorShape": "2",
		},
		Combat: &CombatRow{ID: "1", Attrs: map[string]string{"ID": "1", "Level": "1", "MaxLife": "51"}},
	}
	fields := decode0x28Fields(t, civilian.Frame0x28())
	for _, name := range []string{
		"CollideRadius", "State", "Type", "Resource", "RotatePara", "ConfigID", "Photo",
		"NpcType", "CursorShape", "ActionSpeed", "AIEffect", "Level", "BossLevel",
		"IDCard", "BornAction", "WeaponMode", "HPRatio", "MPRatio",
		"ParryRatio", "HitHPRatio", "HasKarma", "Character", "KarmaLevel",
		"OriginPara1", "OriginPara2", "OriginPara3",
	} {
		if _, ok := fields[name]; !ok {
			t.Errorf("#0 普通形缺字段 %s（2433 v3 官方 #0 样本定版行）", name)
		}
	}
	if _, ok := fields["NotPositive"]; ok {
		t.Error("#0 形不应有 NotPositive（op28_addobject 实验形残留）")
	}
	if got := strings.TrimRight(string(fields["RotatePara"]), "\x00"); got != "0.0,1.000,0.0" {
		t.Errorf("RotatePara = %q, want 0.0,1.000,0.0（2433 v3 缺省）", got)
	}

	// #1 带武器形：WeaponName/FixedBeHitAct/SelfForceID 在场，
	// Photo/CursorShape/MPRatio/karma 族缺席。
	armed := Spawn{
		ObjectID: ObjectIDAt(1), OwnerID: OwnerIDOf(ObjectIDAt(1)),
		ConfigID: "monv010200",
		Template: map[string]string{
			"ID": "monv010200", "Resource": `npc\monster500`, "NpcType": "14",
			"Level": "1", "WeaponName": `npc\w_qiang01`, "WeaponMode": "hand1_1",
		},
	}
	fields = decode0x28Fields(t, armed.Frame0x28())
	for _, name := range []string{"WeaponName", "FixedBeHitAct", "SelfForceID"} {
		if _, ok := fields[name]; !ok {
			t.Errorf("#1 带武器形缺字段 %s", name)
		}
	}
	for _, name := range []string{"Photo", "CursorShape", "MPRatio", "HasKarma", "Character", "KarmaLevel", "OriginPara1"} {
		if _, ok := fields[name]; ok {
			t.Errorf("#1 带武器形不应有 %s（官方 #1 样本无此行）", name)
		}
	}
}
