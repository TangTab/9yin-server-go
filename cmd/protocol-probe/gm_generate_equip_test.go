package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// GM 模板生成（带词条装备）三动作域：模板挑选、洗练池模板方言、proppack
// 固定词条索引与三动作端到端（Java LoginFlow applyGenerate* 对齐）。

// stubGMGenerateCatalog 合成洗练目录（setWeaponRecastCatalogForTest 注入）：
// sword_zztest_60501 精确节（混合 prop/skill）+ blade 家族节（前缀并集路径）
// + 无关节（整池兜底时也并入）。
func stubGMGenerateCatalog(t *testing.T) *weaponRecastCatalog {
	t.Helper()
	dir := t.TempDir()
	randINI := "[sword_zztest_60501]\nr=13040,13041,13234\n\n" +
		"[blade_family_1]\nr=13041\n\n[blade_family_2]\nr=13040,13236\n"
	propPackINI := "[13040]\nr=,MinMeleeDamage,7,0\n\n[13041]\nr=,PhyDefAdd,46,0\n\n[13042]\nr=,StrAdd,35,0\n"
	skillPackINI := "[13234]\nr=CS_a,PhyDamageMul,0.7,0\n\n[13236]\nr=CS_b,StiDamageMul,0.8,0\n"
	for name, content := range map[string]string{
		"senior_weapon_rand.ini": randINI,
		"proppack.ini":           propPackINI,
		"skillpack.ini":          skillPackINI,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return loadWeaponRecastCatalog(
		filepath.Join(dir, "senior_weapon_rand.ini"),
		filepath.Join(dir, "senior_weapon_dire.ini"),
		filepath.Join(dir, "proppack.ini"),
		filepath.Join(dir, "skillpack.ini"))
}

// stubGMFixedIndex 合成 proppack (属性,数值) 索引（setProppackFixedIndexForTest
// 注入）：StrAdd 33/35 与防具 5 红词条全命中（多包命中取最小号验证）。
func stubGMFixedIndex(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	ini := "[9]\nr=,PhyDefAdd,46,0\n\n[5]\nr=,PhyDefAdd,46,0\n\n[7]\nr=,PhyDefAdd,47,0\n" +
		"[11]\nr=,StaAdd,29,0\n\n[13]\nr=,MaxMPAdd,39,0\n\n[42139]\nr=,StrAdd,35,0,20095\n" +
		"[35]\nr=,StrAdd,35,0\n\n[10]\nr=,StrAdd,33,0\n"
	path := filepath.Join(dir, "proppack.ini")
	if err := os.WriteFile(path, []byte(ini), 0o644); err != nil {
		t.Fatalf("write proppack: %v", err)
	}
	setProppackFixedIndexForTest(loadProppackFixedIndex(path))
	t.Cleanup(func() { setProppackFixedIndexForTest(nil) })
}

// newGMGenerateSession 合成会话：可选已穿武器占 22 槽。
func newGMGenerateSession(t *testing.T, withWeapon bool) (*itemSession, *playerActor) {
	t.Helper()
	player := &playerActor{}
	if withWeapon {
		player.equip = []equipRow{{
			configID: "blade_test", itemType: 200, equipType: "weapon",
			slot: 22, hardiness: 100, maxHardiness: 100,
		}}
	}
	return &itemSession{link: &captureMessageConnection{}, player: player, remote: "test"}, player
}

func TestGMTemplateFamilyPrefix(t *testing.T) {
	cases := map[string]string{
		"sword_zztest_60501":     "sword_zztest",
		"cloth_dragon_kf_60701":  "cloth_dragon_kf",
		"blade_plain":            "blade_plain", // 尾段非数字
		"blade_":                 "blade_",      // 空尾段
		"60501":                  "60501",       // 无下划线
	}
	for configID, want := range cases {
		if got := gmTemplateFamilyPrefix(configID); got != want {
			t.Fatalf("familyPrefix(%q)=%q want %q", configID, got, want)
		}
	}
}

func TestGMSectionPoolFallback(t *testing.T) {
	catalog := stubGMGenerateCatalog(t)
	// 精确节。
	if pool := gmTemplateRandPool(catalog, "sword_zztest_60501"); len(pool) != 3 {
		t.Fatalf("exact pool=%v", pool)
	}
	// 家族前缀并集（节名排序合并：blade_family_1 先于 _2）。
	pool := gmTemplateRandPool(catalog, "blade_family_888")
	if len(pool) != 3 || pool[0] != 13041 || pool[1] != 13040 || pool[2] != 13236 {
		t.Fatalf("family pool=%v", pool)
	}
	// 前缀无匹配（blade_test 家族不在册）→ 整池兜底（rand 全节并集：
	// sword 精确节 3 + blade_family 两节 3）。
	pool = gmTemplateRandPool(catalog, "blade_test_999")
	if len(pool) != 6 {
		t.Fatalf("fallback pool=%v", pool)
	}
}

func TestGMDrawPacksReplacement(t *testing.T) {
	// 放回语义：单元素池抽 3 → 3 条同值（可重复）。
	drawn := gmDrawPacks([]int{13040}, 3)
	if len(drawn) != 3 || drawn[0] != 13040 || drawn[2] != 13040 {
		t.Fatalf("drawn=%v", drawn)
	}
	if gmDrawPacks(nil, 3) != nil {
		t.Fatalf("empty pool should return nil")
	}
	if gmDrawPacks([]int{13040}, 0) != nil {
		t.Fatalf("zero count should return nil")
	}
}

func TestLoadProppackFixedIndex(t *testing.T) {
	stubGMFixedIndex(t)
	// 多包命中取最小号：PhyDefAdd+46 在 [5]/[9] → 5；StrAdd+35 在
	// [35]/[42139]（条件包）→ 35。
	if pack := fixedAffixPack("PhyDefAdd", 46); pack != 5 {
		t.Fatalf("PhyDefAdd+46 pack=%d want 5", pack)
	}
	if pack := fixedAffixPack("StrAdd", 35); pack != 35 {
		t.Fatalf("StrAdd+35 pack=%d want 35", pack)
	}
	if pack := fixedAffixPack("StrAdd", 33); pack != 10 {
		t.Fatalf("StrAdd+33 pack=%d want 10", pack)
	}
	if pack := fixedAffixPack("Missing", 1); pack != 0 {
		t.Fatalf("missing pack=%d want 0", pack)
	}
}

func stubGMGenerateItems(t *testing.T) {
	t.Helper()
	injectStubGMItemCatalog(t, &gmItemCatalog{rows: []gmItemRow{
		{ConfigID: "sword_zztest_60501", Name: "测试单剑", ViewID: 2, Wearable: true,
			EquipType: "Weapon", ItemType: 2101, ColorLevel: 5, NeedSex: 2},
		{ConfigID: "sword_zztest_male_60501", ViewID: 2, EquipType: "Weapon",
			ItemType: 2101, ColorLevel: 5, NeedSex: 0},
		{ConfigID: "sword_zztest_low", ViewID: 2, EquipType: "Weapon",
			ItemType: 2101, ColorLevel: 1, NeedSex: 2},
		{ConfigID: "cloth_zztest_60501", Name: "测试衣服", ViewID: 2, EquipType: "Cloth",
			ItemType: 2101, ColorLevel: 5, NeedSex: 2},
		{ConfigID: "wrist_zztest_60501", ViewID: 2, EquipType: "Wrist",
			ItemType: 2101, ColorLevel: 5, NeedSex: 2},
		{ConfigID: "special_zztest", ViewID: 2, EquipType: "Special",
			ItemType: 2101, ColorLevel: 5, NeedSex: 2},
	}})
}

func TestPickGMTemplate(t *testing.T) {
	stubGMGenerateItems(t)
	h, _ := newGMGenerateSession(t, false)

	// 60501 家族优先（低品质非家族候选不参与）。
	picked := pickGMTemplate(h, "Weapon", "sword_")
	if picked == nil || !strings.HasSuffix(picked.ConfigID, "_60501") {
		t.Fatalf("weapon template=%+v", picked)
	}
	// 性别过滤：女号排除 NeedSex=0 男装。
	h.player.visualSex = 1
	for i := 0; i < 20; i++ {
		if picked = pickGMTemplate(h, "Weapon", "sword_"); picked.NeedSex == 0 {
			t.Fatalf("female picked male-only %s", picked.ConfigID)
		}
	}
	// 模型前缀过滤：thorn_ 无候选。
	if picked := pickGMTemplate(h, "Weapon", "thorn_"); picked != nil {
		t.Fatalf("thorn template=%s should be nil", picked.ConfigID)
	}
	// 无家族时回退品质窗口随机（cloth 家族唯一存在，走另一部位验证：
	// Special 无槽类型仅验证挑选不炸）。
	if picked := pickGMTemplate(h, "Special", ""); picked == nil || picked.ConfigID != "special_zztest" {
		t.Fatalf("special template=%+v", picked)
	}
}

func TestApplyGMGenerateWeapon(t *testing.T) {
	setWeaponRecastCatalogForTest(stubGMGenerateCatalog(t))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	stubGMGenerateItems(t)
	h, player := newGMGenerateSession(t, false)

	text, err := applyGMGenerateWeapon(h, 102, "cook")
	if err != nil {
		t.Fatalf("cook weapon: %v", err)
	}
	// 家族有 NeedSex=2/0 两个合法候选，随机二选一——断言家族命中与词条数。
	if !strings.Contains(text, "已生成并穿戴武器 sword_zztest_") || !strings.Contains(text, "（词条 6 条）") {
		t.Fatalf("text=%q", text)
	}
	row, ok := player.wornItemAt(22)
	if !ok || !strings.HasSuffix(row.configID, "_60501") {
		t.Fatalf("worn=%+v", row)
	}
	if got := len(strings.Split(row.propBuilt, ";")); got != 6 {
		t.Fatalf("propBuilt=%q segments=%d", row.propBuilt, got)
	}

	// 未知类型拒绝。
	if _, err := applyGMGenerateWeapon(h, 99, "cook"); err == nil {
		t.Fatalf("weaponType 99 should fail")
	}

	// 铁匠：3 混合 + 3 招式（weaponPool=精确节混合池，招式段计入总数）。
	h2, player2 := newGMGenerateSession(t, false)
	if _, err := applyGMGenerateWeapon(h2, 102, "smith"); err != nil {
		t.Fatalf("smith weapon: %v", err)
	}
	row2, _ := player2.wornItemAt(22)
	if got := len(strings.Split(row2.propBuilt, ";")); got != 6 {
		t.Fatalf("smith propBuilt=%q segments=%d", row2.propBuilt, got)
	}
}

func TestApplyGMGenerateWeaponPoolFallback(t *testing.T) {
	// 池表空（loadWeaponRecastCatalog 全 miss）→ 回退全池随机不可用时
	// 词条为 0 并注明（合成目录 proppack 也空 → propIDs 空 → 回退空）。
	dir := t.TempDir()
	setWeaponRecastCatalogForTest(loadWeaponRecastCatalog(
		filepath.Join(dir, "miss.ini"), filepath.Join(dir, "miss.ini"),
		filepath.Join(dir, "miss.ini"), filepath.Join(dir, "miss.ini")))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	stubGMGenerateItems(t)
	h, _ := newGMGenerateSession(t, false)

	text, err := applyGMGenerateWeapon(h, 102, "cook")
	if err != nil {
		t.Fatalf("fallback weapon: %v", err)
	}
	if !strings.Contains(text, "词条 0 条") || !strings.Contains(text, "官方洗练池缺失") {
		t.Fatalf("text=%q", text)
	}
}

func TestApplyGMGenerateArmor(t *testing.T) {
	stubGMFixedIndex(t)
	stubGMGenerateItems(t)
	h, player := newGMGenerateSession(t, false)

	text, err := applyGMGenerateArmor(h, "cloth")
	if err != nil {
		t.Fatalf("armor: %v", err)
	}
	if !strings.Contains(text, "已生成并穿戴防具 cloth_zztest_60501（词条 5 条）") {
		t.Fatalf("text=%q", text)
	}
	row, ok := player.wornItemAt(3) // cloth 槽
	if !ok || row.configID != "cloth_zztest_60501" {
		t.Fatalf("worn=%+v", row)
	}
	if got := strings.Split(row.propBuilt, ";"); len(got) != 5 {
		t.Fatalf("propBuilt=%q", row.propBuilt)
	}

	// 未穿武器场景的未知部位拒绝。
	if _, err := applyGMGenerateArmor(h, "belt"); err == nil {
		t.Fatalf("armorSlot belt should fail")
	}
}

func TestApplyGMGenerateArmorAffixMiss(t *testing.T) {
	// 索引空 → 5 条全 miss → 回执注明缺 5 条。
	setProppackFixedIndexForTest(&proppackFixedIndex{byPropValue: map[string]map[float64]int{}})
	t.Cleanup(func() { setProppackFixedIndexForTest(nil) })
	stubGMGenerateItems(t)
	h, player := newGMGenerateSession(t, false)

	text, err := applyGMGenerateArmor(h, "cloth")
	if err != nil {
		t.Fatalf("armor: %v", err)
	}
	if !strings.Contains(text, "词条 0 条，红词条缺 5 条") {
		t.Fatalf("text=%q", text)
	}
	row, _ := player.wornItemAt(3)
	if row.propBuilt != "" {
		t.Fatalf("propBuilt=%q want empty", row.propBuilt)
	}
}

func TestApplyGMGenerateBracer(t *testing.T) {
	setWeaponRecastCatalogForTest(stubGMGenerateCatalog(t))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	stubGMFixedIndex(t)
	stubGMGenerateItems(t)

	// 未穿武器：3 臂力，招式省略。
	h, player := newGMGenerateSession(t, false)
	text, err := applyGMGenerateBracer(h)
	if err != nil {
		t.Fatalf("bracer: %v", err)
	}
	if !strings.Contains(text, "护腕 wrist_zztest_60501（词条 3 条，未穿武器，招式词条省略）") {
		t.Fatalf("text=%q", text)
	}
	row, _ := player.wornItemAt(6) // wrist 槽
	if row.propBuilt != "1,10;1,35;1,35" {
		t.Fatalf("propBuilt=%q want 1,10;1,35;1,35", row.propBuilt)
	}

	// 已穿武器：+3 招式（blade_test 家族前缀节并集 [13041,13040] 均为
	// prop/skill 混合兜底——合成目录 blade_family_* 命中）。
	h2, player2 := newGMGenerateSession(t, true)
	text, err = applyGMGenerateBracer(h2)
	if err != nil {
		t.Fatalf("bracer with weapon: %v", err)
	}
	row2, _ := player2.wornItemAt(6)
	if got := len(strings.Split(row2.propBuilt, ";")); got != 6 {
		t.Fatalf("propBuilt=%q segments=%d", row2.propBuilt, got)
	}
	if !strings.Contains(text, "词条 6 条") {
		t.Fatalf("text=%q", text)
	}
}

func TestWearGMTemplateDisplace(t *testing.T) {
	setWeaponRecastCatalogForTest(stubGMGenerateCatalog(t))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	stubGMGenerateItems(t)
	h, player := newGMGenerateSession(t, true) // blade_test 占 22

	if _, err := applyGMGenerateWeapon(h, 102, "cook"); err != nil {
		t.Fatalf("displace weapon: %v", err)
	}
	// 22 槽换新（家族随机二选一），旧武器让位入包。
	row, _ := player.wornItemAt(22)
	if !strings.HasSuffix(row.configID, "_60501") {
		t.Fatalf("slot 22=%s", row.configID)
	}
	moved := false
	for _, bagRowValue := range player.sessionBag() {
		if bagRowValue.configID == "blade_test" && bagRowValue.slot > 0 {
			moved = true
		}
	}
	if !moved {
		t.Fatalf("blade_test should be displaced to bag")
	}
}

func TestWearGMTemplateBagFallback(t *testing.T) {
	// 槽位未证类型（Special 不在 equipSlotsByType）→ 兜底入包，词条不随行。
	stubGMGenerateItems(t)
	h, player := newGMGenerateSession(t, false)

	text, err := wearGMTemplate(h, gmItems.rows[5], []int{13040}, "防具", "")
	if err != nil {
		t.Fatalf("bag fallback: %v", err)
	}
	if !strings.Contains(text, "入背包第") || !strings.Contains(text, "词条不随行") {
		t.Fatalf("text=%q", text)
	}
	found := false
	for _, bagRowValue := range player.sessionBag() {
		if bagRowValue.configID == "special_zztest" {
			found = bagRowValue.propBuilt == ""
		}
	}
	if !found {
		t.Fatalf("special_zztest should be in bag without propBuilt")
	}
}

func TestApplyGMGenerateEquipDispatch(t *testing.T) {
	stubGMGenerateItems(t)
	h, _ := newGMGenerateSession(t, false)
	if _, err := applyGMGenerateEquip(h, "gm_generate_bracer", 0, "", ""); err != nil {
		t.Fatalf("dispatch bracer: %v", err)
	}
	if _, err := applyGMGenerateEquip(h, "gm_generate_unknown", 0, "", ""); err == nil {
		t.Fatalf("unknown action should fail")
	}
}
