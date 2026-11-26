package rolestate

import (
	"strings"
	"testing"
)

// TestBuildRoleStateBorn03 钉死默认建号状态（官方口径：born03 烟雨庄 +
// RoleBaseProp 五维=3/HP70/MP40 + StarterSet 授予集 + location.ini
// BornCJ000003 出生点）。SP（怒气）例外：官服建号档 80、登录即满，本线
// 2026-11-26 定案改 0——怒气是会话值，不随档。
func TestBuildRoleStateBorn03(t *testing.T) {
	state, err := BuildRoleState(DefaultBirthScene, 1, "icon\\players\\boy01_create.png,obj\\char\\b_hair\\b_hair1", 0)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if state.Sex != 1 || state.Str != 3 || state.Dex != 3 || state.Ing != 3 || state.Spi != 3 || state.Sta != 3 {
		t.Fatalf("五维官方口径 =3（初始号实证）： %+v", state)
	}
	if state.HP != 70 || state.MP != 40 || state.SP != 0 || state.Parry != 100 {
		t.Fatalf("资源初值 HP70/MP40/SP0/Parry100： %+v", state)
	}
	if state.Scene != `ini\scene\born03_YanYuZhuang` || state.Resource != "born03" {
		t.Fatalf("出生场景 = %q resource = %q", state.Scene, state.Resource)
	}
	if state.Position != [4]float64{482.5, 13.7, 1008, 1.56} {
		t.Fatalf("BornCJ000003 出生点 = %v", state.Position)
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

// TestRoleStateJSONRoundTrip JSON 序列化回读（nx_roles.role_state_json 通道）。
func TestRoleStateJSONRoundTrip(t *testing.T) {
	state, err := BuildRoleState(DefaultBirthScene, 2, "icon\\players\\girl01_create.png", 0)
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
	state, err := BuildRoleState(DefaultBirthScene, 1, "", 0)
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
	fresh, err := BuildRoleState(DefaultBirthScene, 1, "", 0)
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
	state, err := BuildRoleState(DefaultBirthScene, 1, "", 0)
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
