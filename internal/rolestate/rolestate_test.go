package rolestate

import (
	"strings"
	"testing"
)

// TestBuildRoleStateBorn03 钉死默认建号状态（官方口径 2026-11-26 建号.pcapng：
// 首登新手副本 born_single_01（BornCJ000100 锚）+ book2 出生地烟雨庄
// ChushengCJ000002；RoleBaseProp 五维=3/HP70/MP40 + StarterSet 授予集）。
// SP（怒气）例外：官服建号档 80、登录即满，本线 2026-11-26 定案改 0——
// 怒气是会话值，不随档。
func TestBuildRoleStateBorn03(t *testing.T) {
	state, err := BuildRoleState(DefaultBook, 1, "icon\\players\\boy01_create.png,obj\\char\\b_hair\\b_hair1", 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if state.Sex != 1 || state.Str != 3 || state.Dex != 3 || state.Ing != 3 || state.Spi != 3 || state.Sta != 3 {
		t.Fatalf("五维官方口径 =3（初始号实证）： %+v", state)
	}
	if state.HP != 70 || state.MP != 40 || state.SP != 0 || state.Parry != 100 {
		t.Fatalf("资源初值 HP70/MP40/SP0/Parry100： %+v", state)
	}
	// 首登场景 = 新手副本（官服首帧 0x0B 实证）；出生地入 Birth 字段。
	if state.Scene != `ini\scene\born_single_01` || state.Resource != GuideScene {
		t.Fatalf("首登场景 = %q resource = %q", state.Scene, state.Resource)
	}
	if state.Position != [4]float64{197.626, 4.746, -108.829, -2.746} {
		t.Fatalf("副本首登锚（BornCJ000100） = %v", state.Position)
	}
	if state.Book != DefaultBook || state.BirthSection != "ChushengCJ000002" ||
		state.BirthScene != `ini\scene\born03_YanYuZhuang` || state.BirthResource != "born03" {
		t.Fatalf("出生地字段： %+v", state)
	}
	if state.BirthPosition != [4]float64{371.67, 23.57, 1058.48, 2.23} {
		t.Fatalf("ChushengCJ000002 出生点 = %v", state.BirthPosition)
	}
	if state.NewbieSkill != 1 || state.InitTask != 22001 || state.BaishiTask != 49 {
		t.Fatalf("NewbieSkill/InitTask/BaishiTask = %d/%d/%d", state.NewbieSkill, state.InitTask, state.BaishiTask)
	}
	if len(state.Skills) != 27 {
		t.Fatalf("starter 武学 27 项（5 基本+7 花神+7 心佛掌+8 太极），得 %d", len(state.Skills))
	}
	if len(state.NeiGong) != 6 || state.CurNeiGong != "ng_wd_001" {
		t.Fatalf("初始内功 6 本/首本 ng_wd_001，得 %d 本 cur=%s", len(state.NeiGong), state.CurNeiGong)
	}
	if state.NeiGong[0].Total != 273200 || state.NeiGong[0].Power != 4550 || state.NeiGong[0].MaxHPAdd != 1630 {
		t.Fatalf("首本内功 30 级数值： %+v", state.NeiGong[0])
	}
	if len(state.QingGongCategories) != 18 || len(state.QingGongSkills) != 52 {
		t.Fatalf("轻功类别 18/实例 52（b-line starterQingGongSkillIDs 全集），得 %d/%d", len(state.QingGongCategories), len(state.QingGongSkills))
	}
	if len(state.CollectSkills) != 204 {
		t.Fatalf("采集技能 204 项，得 %d", len(state.CollectSkills))
	}
	// 任务门：Guide 1069..1073(5) + Alt 1344 + SchoolLead 1555..1562(8) +
	// Act 1140..1147(8) + ActAlt 51002 = 23 行（b-line facultyAcceptedTaskIDs）。
	if len(state.TaskAccepted) != 23 || state.TaskAccepted[0] != 1069 || state.TaskAccepted[len(state.TaskAccepted)-1] != 51002 {
		t.Fatalf("Task_Accepted 23 行： %v", state.TaskAccepted)
	}
	// Task_Completed：Unlock 1346 + OpenGate 49634 + FuncGate 19 项 = 21。
	if len(state.TaskCompleted) != 21 || state.TaskCompleted[0] != 1346 || state.TaskCompleted[1] != 49634 {
		t.Fatalf("Task_Completed 21 项： %v", state.TaskCompleted)
	}
}

// TestBuildRoleStateBooks 全剧本 → 出生地（books.ini + location.ini 数据
// 驱动；四出生地用户口径 + 缺节/零坐标回落链）。
func TestBuildRoleStateBooks(t *testing.T) {
	cases := []struct {
		book     string
		section  string
		resource string
		scene    string
		pos      [4]float64
	}{
		// 直取：ChushengCJ 节坐标非 0。
		{"book1", "ChushengCJ000001", "born04", `ini\scene\born04_QianDengZhen`, [4]float64{888, -23.68, 623, 0}},
		{"book2", "ChushengCJ000002", "born03", `ini\scene\born03_YanYuZhuang`, [4]float64{371.67, 23.57, 1058.48, 2.23}},
		// 坐标 0：同场景 BornCJ 节回落（裸目录/带后缀 resource 等价匹配）。
		{"book4", "ChushengCJ000009", "city05", `ini\scene\city05`, [4]float64{418, 22.98, 607, 1.59}},
		{"book5", "ChushengCJ000003", "born01", `ini\scene\born01`, [4]float64{990, 6.66, 532, 4.7}},
		{"book6", "ChushengCJ000008", "city04", `ini\scene\city04`, [4]float64{1209.7, -17.06, 705.26, 4.75}},
		// 节缺席：FallbackBornName 显式回落。
		{"book3", "ChushengCJ000001", "born04", `ini\scene\born04_QianDengZhen`, [4]float64{888, -23.68, 623, 0}},
		{"book8", "ChushengCJ000002", "born03", `ini\scene\born03_YanYuZhuang`, [4]float64{371.67, 23.57, 1058.48, 2.23}},
		// 空剧本/未知剧本 → DefaultBook。
		{"", "ChushengCJ000002", "born03", `ini\scene\born03_YanYuZhuang`, [4]float64{371.67, 23.57, 1058.48, 2.23}},
		{"book99", "ChushengCJ000002", "born03", `ini\scene\born03_YanYuZhuang`, [4]float64{371.67, 23.57, 1058.48, 2.23}},
	}
	for _, c := range cases {
		place := BirthplaceOf(c.book)
		if place.Section != c.section || place.Resource != c.resource ||
			place.Scene != c.scene || place.Position != c.pos {
			t.Fatalf("BirthplaceOf(%q) = %+v，want %+v", c.book, place, c)
		}
	}
	// 首登恒副本：任意剧本的 Scene/Resource 都是 born_single_01。
	for _, book := range []string{"book1", "book4", "book5", ""} {
		state, err := BuildRoleState(book, 1, "", 0)
		if err != nil {
			t.Fatalf("BuildRoleState(%q): %v", book, err)
		}
		if state.Resource != GuideScene || state.Scene != `ini\scene\born_single_01` {
			t.Fatalf("book %q 首登应落副本： %q/%q", book, state.Scene, state.Resource)
		}
	}
	// 剧本/出生地查询面。
	if got := BirthSectionOf("book1"); got != "ChushengCJ000001" {
		t.Fatalf("BirthSectionOf(book1) = %q", got)
	}
	if got := InferBookFromPara("icon\\players\\boy01_create.png,book2,obj\\char\\b_hair\\b_hair1,school_kunlun"); got != "book2" {
		t.Fatalf("InferBookFromPara = %q", got)
	}
	if got := InferBookFromPara("school_kunlun,bookmark"); got != "" {
		t.Fatalf("非剧本词不得误判： %q", got)
	}
	// 属地复活点（ReliveCJ 节表组串；鸡鸣驿=官服男号 ui 键）。
	if got, ok := ReliveOfScene(`ini\scene\born01_MoHeZhen`); !ok || got != "4,584.97,24.57,298.65,0.6,ui_pos_ReliveCJ100101" {
		t.Fatalf("ReliveOfScene(born01) = %q ok=%v", got, ok)
	}
	if _, ok := ReliveOfScene(`ini\scene\no_such_scene`); ok {
		t.Fatal("未知场景不应有复活点")
	}
}

// TestRoleStateJSONRoundTrip JSON 序列化回读（nx_roles.role_state_json 通道）。
func TestRoleStateJSONRoundTrip(t *testing.T) {
	state, err := BuildRoleState(DefaultBook, 2, "icon\\players\\girl01_create.png", 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	raw, err := MarshalRoleState(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(raw, `"curNeiGong":"ng_wd_001"`) {
		t.Fatalf("json 缺内功字段: %s", raw[:120])
	}
	back, err := UnmarshalRoleState(raw)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Sex != 2 || len(back.Skills) != 27 || back.Position != state.Position {
		t.Fatalf("回读不一致： %+v", back)
	}
	empty, err := UnmarshalRoleState("")
	if err != nil || empty.Sex != 0 {
		t.Fatalf("空档兼容： %v %+v", err, empty)
	}
}

// TestInferSexFromPara 性别推断（world 侧 inferSex 同口径）。
func TestInferSexFromPara(t *testing.T) {
	cases := map[string]int{
		"icon\\players\\girl01_create.png": 2,
		"icon\\players\\boy01_create.png":  1,
		"photo,obj\\char\\g_hair\\g_hair1": 2,
		"photo,obj\\char\\b_hair\\b_hair1": 1,
		"":                                 1,
	}
	for para, want := range cases {
		if got := InferSexFromPara(para); got != want {
			t.Fatalf("InferSexFromPara(%q) = %d, want %d", para, got, want)
		}
	}
}

// TestMoneyRoundTrip 碎银字段（CapitalType1 位）marshal 往返与建号初值。
func TestMoneyRoundTrip(t *testing.T) {
	st, err := BuildRoleState("", 0, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if st.Money != starterMoney {
		t.Fatalf("starter money %d", st.Money)
	}
	st.Money = 123456789
	js, err := MarshalRoleState(st)
	if err != nil {
		t.Fatal(err)
	}
	back, err := UnmarshalRoleState(js)
	if err != nil {
		t.Fatal(err)
	}
	if back.Money != 123456789 {
		t.Fatalf("money round trip %d", back.Money)
	}
	// 旧档（无 money 键）反序列化 → 0（omitempty 兼容）
	legacy, err := UnmarshalRoleState(`{"sex":1,"scene":"s","resource":"r","position":[0,0,0,1]}`)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Money != 0 {
		t.Fatalf("legacy money %d", legacy.Money)
	}
}

// TestProgressAndItemDomainRoundTrip 进度域（OpenMaps）与物品域
// （BagItems/Equips，2433 线 bagRow/equipRow 口径）JSON 往返 + 槽位/行号
// 分配助手。
func TestProgressAndItemDomainRoundTrip(t *testing.T) {
	state, err := BuildRoleState(DefaultBook, 1, "", 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	state.OpenMaps = make([]uint32, 24)
	state.OpenMaps[11] = 1 << 14
	state.BagItems = []BagItem{
		{Slot: 1, ConfigID: "yaopin_00100", ItemType: 8, Amount: 5, ViewID: 1, Seq: 1, UID: "aabbccdd"},
		{Slot: 2, ConfigID: "wuqi_001", ItemType: 16, ViewID: 2, Seq: 2},
	}
	state.Equips = []EquipItem{
		{Slot: 22, ConfigID: "wuqi_001", ItemType: 16, EquipType: "Weapon", Seq: 2, UID: "aabbccdd"},
	}
	raw, err := MarshalRoleState(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, err := UnmarshalRoleState(raw)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.OpenMaps) != 24 || back.OpenMaps[11] != 1<<14 {
		t.Fatalf("OpenMaps 回读不一致: %v", back.OpenMaps)
	}
	if len(back.BagItems) != 2 || back.BagItems[0].UID != "aabbccdd" ||
		len(back.Equips) != 1 || back.Equips[0].EquipType != "Weapon" {
		t.Fatalf("物品域回读不一致: %+v %+v", back.BagItems, back.Equips)
	}

	// 槽位/行号助手：分类页首空格、跨表 max+1。
	if got := back.FreeBagSlot(1); got != 2 {
		t.Fatalf("FreeBagSlot(1)=%d, want 2", got)
	}
	if got := back.FreeBagSlot(3); got != 1 {
		t.Fatalf("FreeBagSlot(3)=%d, want 1（空分类页）", got)
	}
	if got := back.NextItemSeq(); got != 3 {
		t.Fatalf("NextItemSeq()=%d, want 3", got)
	}

	// omitempty：空进度域不落 JSON（旧档兼容）。
	fresh, err := BuildRoleState(DefaultBook, 1, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	rawFresh, err := MarshalRoleState(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rawFresh, "openMaps") || strings.Contains(rawFresh, "bagItems") {
		t.Fatalf("空进度域应省略: %s", rawFresh)
	}
}
// TestCustomizingBinaryRoundTrip 个性配置串字节保真：含 ≥0x80 的二进制串
// JSON 往返不得被 UTF-8 清洗（旧实现经 encoding/json 把高字节换成
// U+FFFD，一个字节膨胀成三个，回推客户端后画面配置整体错位——
// 2026-11-23 重登卡顿根因）。
func TestCustomizingBinaryRoundTrip(t *testing.T) {
	// 11-22 真机首传的尾部形态：161/129/145/229 等取值。
	config := "\x01e\x01\x01\x01\x10\x02\x02\x02\xa1\x02\x02\x01\x02\xe5\x02\x81\x01"
	state, err := BuildRoleState(DefaultBook, 1, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	state.Customizing = BinaryString(config)
	raw, err := MarshalRoleState(state)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, replacementBytes) {
		t.Fatalf("marshal 清洗了二进制串: %q", raw)
	}
	back, err := UnmarshalRoleState(raw)
	if err != nil {
		t.Fatal(err)
	}
	if string(back.Customizing) != config {
		t.Fatalf("回读不一致: %q want %q", back.Customizing, config)
	}

	// 旧档裸串：完好的照收，被清洗过（含 U+FFFD）的丢弃。
	const base = `{"sex":1,"scene":"s","resource":"r","position":[0,0,0,1],"customizing":`
	legacy, err := UnmarshalRoleState(base + `"\u0001e\u0001\u0002"}`)
	if err != nil {
		t.Fatal(err)
	}
	if string(legacy.Customizing) != "\x01e\x01\x02" {
		t.Fatalf("旧档完好串: %q", legacy.Customizing)
	}
	poisoned, err := UnmarshalRoleState(base + `"\u0001e\u0001\ufffd\u0002\ufffd\ufffd"}`)
	if err != nil {
		t.Fatal(err)
	}
	if poisoned.Customizing != "" {
		t.Fatalf("旧档坏串应丢弃: %q", poisoned.Customizing)
	}
	if !ValidCustomizing(config) || ValidCustomizing("a\ufffdb") {
		t.Fatal("ValidCustomizing 口径")
	}
}

// TestNormalizeAppearance 捏脸串含逗号的槽位归位（2026-11-27 真机「没头发」
// 根因：para 逗号往返把捏脸炸碎、Hair/Face 属性缺席）。
func TestNormalizeAppearance(t *testing.T) {
	fields := strings.Split("book5,icon\\players\\boy02_create.png,\x0c===/=2=6I/@64.6'\x1f5//6,0\x02J\x02V,1,obj\\char\\b_hair\\b_hair4,cloth_b0001,pants_b0001,shoes_b0003,school_wudang,", ",")
	got := NormalizeAppearance(fields)
	want := []string{
		"book5",
		"icon\\players\\boy02_create.png",
		"\x0c===/=2=6I/@64.6'\x1f5//6,0\x02J\x02V", // 捏脸碎片并回一槽
		"1",
		"obj\\char\\b_hair\\b_hair4",
		"cloth_b0001", "pants_b0001", "shoes_b0003",
		"school_wudang",
		"",
	}
	if len(got) != len(want) {
		t.Fatalf("槽位数 = %d want %d: %q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("槽 %d = %q want %q", i, got[i], want[i])
		}
	}
	// 无逗号捏脸（官服样本形态）不受影响。
	plain := NormalizeAppearance(strings.Split("book2,icon\\players\\boy01_create.png,333333X38,1,obj\\char\\b_hair\\b_hair1,cloth_b0001,", ","))
	if plain[2] != "333333X38" || plain[4] != "obj\\char\\b_hair\\b_hair1" {
		t.Fatalf("无逗号捏脸被改写: %q", plain)
	}
}
