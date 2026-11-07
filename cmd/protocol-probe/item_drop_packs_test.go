package main

import (
	"math/rand"
	"testing"
)

// 掉落三源：DropPacks 权重 roll / 缺包兜底阶梯 / box_contents 描述对照
//（Java DropPacks/BoxContents 同构）。

func testDropPacksJSON() []byte {
	return []byte(`{
		"packages": {
			"pack_single": [{"item": "yaopin_00100", "min": 2, "max": 2, "weight": 10}],
			"pack_weighted": [
				{"item": "item_a", "min": 1, "max": 1, "weight": 7},
				{"item": "item_b", "min": 1, "max": 1, "weight": 3}
			]
		}
	}`)
}

func mustDropPacks(t *testing.T, data []byte) *dropPacks {
	t.Helper()
	packs, err := loadDropPacks(data)
	if err != nil {
		t.Fatal(err)
	}
	return packs
}

func TestDropPackRollPicksByWeightAndClampsRolls(t *testing.T) {
	packs := mustDropPacks(t, testDropPacksJSON())
	random := rand.New(rand.NewSource(42))

	// 单条包：rolls 超 10 也只 roll 10 次；min=max=2 恒 2/次 → 合并 20。
	grants := packs.roll("pack_single", 99, random)
	if len(grants) != 1 || grants[0].configID != "yaopin_00100" || grants[0].amount != 20 {
		t.Fatalf("single pack grants=%v, want yaopin_00100 x20", grants)
	}

	// 权重边界：r=0 → 首条（权重 7）；r→1⁻ → 末条。
	entries := []dropPackEntry{
		{item: "item_a", min: 1, max: 1, weight: 7},
		{item: "item_b", min: 1, max: 1, weight: 3},
	}
	if got := pickDropEntry(entries, 0); got.item != "item_a" {
		t.Fatalf("pick(r=0)=%+v, want item_a", got)
	}
	if got := pickDropEntry(entries, 0.99); got.item != "item_b" {
		t.Fatalf("pick(r=0.99)=%+v, want item_b", got)
	}
	if got := pickDropEntry([]dropPackEntry{{item: "x", min: 1, max: 1, weight: 0}}, 0.5); got != nil {
		t.Fatalf("zero-weight pick=%+v, want nil", got)
	}

	// rolls<1 钳 1；空 dropId 返回空。
	if grants = packs.roll("", 3, random); len(grants) != 0 {
		t.Fatalf("empty dropId grants=%v", grants)
	}
	if !packs.has("pack_single") || packs.has("missing") {
		t.Fatal("has() 只认 authored 包")
	}
}

func TestDropPackMissingFallsBackToLadder(t *testing.T) {
	packs := mustDropPacks(t, []byte(`{"packages": {}}`))
	random := rand.New(rand.NewSource(1))
	grants := packs.roll("box_totally_missing", 1, random)
	if len(grants) != 1 {
		t.Fatalf("fallback ladder grants=%v", grants)
	}
	config := grants[0].configID
	if config != "yaopin_00100" && config != "yaopin_00300" && config != "yaopin_00500" && config != "yaopin_00700" {
		t.Fatalf("fallback grant=%s, want yaopin ladder", config)
	}
}

func TestBoxContentsFixedGrants(t *testing.T) {
	contents, err := loadBoxContents([]byte(`{
		"boxes": {
			"box_prize_test_001": [
				{"item": "item_dmp_ptz", "min": 40, "max": 40},
				{"item": "item_dmp_fjsj", "min": 1, "max": 1}
			],
			"box_bad": [{"item": "x", "min": 1, "max": 5000}]
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	grants := contents.byBox("Box_Prize_Test_001") // 大小写不敏感
	if len(grants) != 2 || grants[0].configID != "item_dmp_ptz" || grants[0].amount != 40 ||
		grants[1].configID != "item_dmp_fjsj" || grants[1].amount != 1 {
		t.Fatalf("box grants=%v", grants)
	}
	if contents.byBox("box_bad") != nil {
		t.Fatal("max>999 条目应整包剔除") // Java 同款剔除（min<1/max<min/>999）
	}
	if contents.byBox("box_unknown") != nil {
		t.Fatal("未知箱应返回 nil 回落 DropPacks")
	}
}

func TestItemUseCatalogResolvesBoxesCompensateAndSkills(t *testing.T) {
	toolItem := iniTable{
		"box_item_test": {
			{key: "script", value: "BoxItem"},
			{key: "DropID", value: "qy_chengdu_009"},
		},
		"box_qycdu_009": {
			{key: "script", value: "CompensateItem"},
			{key: "@CompensateItemRec", value: "18658,qy_chengdu_009"},
			{key: "@CompensateItemRec", value: "18659,box_qycdu_009"},
		},
	}
	useStatic := iniTable{
		"2101": {{key: "SkillID", value: "yaopinhp1"}, {key: "SkillLevel", value: "1"}},
		"2104": {{key: "SkillID", value: "yaopinhp4"}, {key: "SkillLevel", value: "9"}},
		"9999": {{key: "SkillLevel", value: "3"}}, // 无 SkillID → 剔除
	}
	catalog := loadItemUseCatalogFromTables(nil, toolItem, useStatic)

	entry, ok := catalog.resolve("BOX_ITEM_TEST", 0) // configId 大小写不敏感
	if !ok || entry.kind != itemUseBox || len(entry.dropIDs) != 1 || entry.dropIDs[0] != "qy_chengdu_009" {
		t.Fatalf("BoxItem entry=%+v ok=%v", entry, ok)
	}
	entry, ok = catalog.resolve("box_qycdu_009", 0)
	if !ok || len(entry.dropIDs) != 2 { // CompensateItem ref 全发（func-56 未接）
		t.Fatalf("CompensateItem entry=%+v ok=%v", entry, ok)
	}
	entry, ok = catalog.resolve("unknown_box", 2101)
	if !ok || entry.kind != itemUseSkill || entry.skillID != "yaopinhp1" || entry.skillLevel != 1 {
		t.Fatalf("use-skill entry=%+v ok=%v", entry, ok)
	}
	entry, ok = catalog.resolve("unknown_box", 2104)
	if !ok || fallbackHealHp(entry.skillLevel) != 3000 || fallbackHealMp(entry.skillLevel) != 1500 {
		t.Fatalf("ladder L9 entry=%+v hp=%d", entry, fallbackHealHp(entry.skillLevel))
	}
	if _, ok = catalog.resolve("unknown", 9999); ok {
		t.Fatal("无 SkillID 的档位应剔除")
	}
}

func TestSkillBookCatalogParsesTargetsAndLayers(t *testing.T) {
	toolItem := iniTable{
		"book_page2": {
			{key: "script", value: "SkillPage"},
			{key: "LearnSkillPack", value: "pack_page2"},
			{key: "SkillMaxLevel", value: "2"},
			{key: "UseCondition", value: "12193"},
		},
		"book_scroll": {
			{key: "script", value: "SkillScroll"},
			{key: "LearnSkillPack", value: "pack_scroll"},
			{key: "AddLevel", value: "1"},
			{key: "AddMaxLevel", value: "9"},
		},
		"book_no_targets": {
			{key: "script", value: "SkillBook"},
			{key: "LearnSkillPack", value: "pack_missing"},
		},
	}
	learnStatic := iniTable{
		"pack_page2":    {{key: "TragetID", value: "CS_em_jdmz01"}},                 // 官方拼写
		"pack_scroll":   {{key: "TragetID", value: "ng_em_001,ng_em_002;QG_JH_01"}}, // ',' ';' 混合
		"pack_qinggong": {{key: "TragetID", value: "qinggong_20,qinggong_25"}},
	}
	catalog := loadSkillBookCatalogFromTables(toolItem, learnStatic)

	entry, ok := catalog.resolve("book_page2")
	if !ok || len(entry.targets) != 1 || entry.targets[0] != "CS_em_jdmz01" ||
		entry.skillMaxLevel != 2 || entry.useCondition != 12193 {
		t.Fatalf("page book=%+v ok=%v", entry, ok)
	}
	entry, ok = catalog.resolve("BOOK_SCROLL")
	if !ok || entry.addLevel != 1 || entry.addMaxLevel != 9 ||
		len(entry.targets) != 3 || entry.targets[2] != "QG_JH_01" {
		t.Fatalf("scroll book=%+v ok=%v", entry, ok)
	}
	if _, ok = catalog.resolve("book_no_targets"); ok {
		t.Fatal("无目标表行的书应剔除")
	}
	if !isLearnScript("skillpage") || isLearnScript("ToolItem") {
		t.Fatal("isLearnScript 三类判定")
	}
}

func TestSkillGateTableType3RangeAndSchool(t *testing.T) {
	condition := iniTable{
		"12193": {
			{key: "Type", value: "3"}, {key: "Name", value: "Skill"},
			{key: "Para1", value: "CS_em_jdmz01"}, {key: "Para2", value: "MaxLevel"},
			{key: "min", value: "1"}, {key: "max", value: "1"},
		},
		"13563": {
			{key: "Type", value: "3"}, {key: "Name", value: "NeiGongContainer"},
			{key: "Para1", value: "ng_em_001"}, {key: "Para2", value: "MaxLevel"},
			{key: "min", value: "9"}, {key: "max", value: "17"},
		},
		"20001": {
			{key: "Type", value: "1"}, {key: "min", value: "school_wd"},
		},
		"30001": {
			{key: "Type", value: "19"},
		},
	}
	table := loadSkillGateTableFromTables(condition)

	gate, ok := table.gate(12193)
	if !ok || !gate.containerCapInRange(1) || gate.containerCapInRange(2) || gate.containerCapInRange(0) {
		t.Fatalf("gate12193=%+v range broken", gate)
	}
	gate, _ = table.gate(13563)
	if !gate.containerCapInRange(9) || !gate.containerCapInRange(17) || gate.containerCapInRange(18) || gate.containerCapInRange(8) {
		t.Fatalf("gate13563=%+v range broken", gate)
	}
	if gate, _ = table.gate(20001); gate.min != "school_wd" {
		t.Fatalf("school gate=%+v", gate)
	}
	if gate, ok = table.gate(99999); ok {
		t.Fatal("缺席条件应返回 !ok（放行）")
	}
}
