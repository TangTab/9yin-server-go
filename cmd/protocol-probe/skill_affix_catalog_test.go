package main

import (
	"strings"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// 招式威力词条目录域（skill_affix_catalog.go）：skillpack 威力包装载、
// 变体过滤、套路分组、类型文案与两级目录组装。

// stubSkillAffixTables 合成技能静态表：神风诀 5 个 key（实招 sfj04 /
// 虚招 sfj03 / 怒招 sfj08——varprop→consume AConsumeSP=50 / _hide 变体 /
// _1 分层变体——后者挂在基座 LevelupRelevance）。
func stubSkillAffixTables() skillResourceTables {
	fields := func(pairs ...string) []iniField {
		out := make([]iniField, 0, len(pairs)/2)
		for i := 0; i+1 < len(pairs); i += 2 {
			out = append(out, iniField{key: pairs[i], value: pairs[i+1]})
		}
		return out
	}
	return skillResourceTables{
		skillNew: iniTable{
			"CS_jh_sfj04":      fields("StaticData", "12118", "script", "SkillLock"),
			"CS_jh_sfj03":      fields("StaticData", "12117", "script", "SkillNormal"),
			"CS_jh_sfj08":      fields("StaticData", "12128", "script", "SkillNormal"),
			"CS_jh_sfj04_hide": fields("StaticData", "12133"),
			"CS_jh_sfj01_1":    fields("StaticData", "12114"),
		},
		skillStatic: iniTable{
			"12113": fields("TaoLu", "CS_jh_sfj", "EffectType", "1", "LevelupRelevance", "CS_jh_sfj01_1,CS_jh_sfj01_2"),
			"12117": fields("TaoLu", "CS_jh_sfj", "EffectType", "2", "MinVarPropNo", "242321", "MaxVarPropNo", "242332"),
			"12118": fields("TaoLu", "CS_jh_sfj", "EffectType", "1", "MinVarPropNo", "242341", "MaxVarPropNo", "242352"),
			"12128": fields("TaoLu", "CS_jh_sfj", "EffectType", "1", "MinVarPropNo", "242541", "MaxVarPropNo", "242552"),
			"12133": fields("TaoLu", "CS_jh_sfj", "EffectType", "1"),
			"12114": fields("TaoLu", "CS_jh_sfj", "EffectType", "1"),
		},
		normalVar: iniTable{
			"242321": fields("Level", "1", "Consume", "5500"), // 虚招：无怒气消耗
			"242541": fields("Level", "1", "Consume", "5501"), // 怒招：AConsumeSP=50
		},
		lockVar: iniTable{
			"242341": fields("Level", "1", "Consume", "5500"), // 实招(SkillLock)：无怒气消耗
		},
		consume: iniTable{
			"5500": fields("AConsumeSP", "0"),
			"5501": fields("AConsumeSP", "50"),
		},
	}
}

// stubSkillAffixCatalog 合成威力词条目录（临时 skillpack.ini 装载）：
// 13221 威力包同包覆盖基座+hide；13222 只 sfj04 高档；13224 只 sfj03；
// 13226 只 sfj08（怒招）；13117 是非威力包（BelongCategory 冲锋距离）不收。
func stubSkillAffixCatalog(t *testing.T) *skillAffixCatalog {
	t.Helper()
	dir := t.TempDir()
	ini := "[13221]\nPackCategory=5\nr=CS_jh_sfj04,PhyDamageMul,0.05,0\nr=CS_jh_sfj04_hide,PhyDamageMul,0.05,0\n\n" +
		"[13222]\nPackCategory=5\nr=CS_jh_sfj04,PhyDamageMul,0.1,0\n\n" +
		"[13224]\nPackCategory=5\nr=CS_jh_sfj03,PhyDamageMul,0.3,0\n\n" +
		"[13226]\nPackCategory=5\nr=CS_jh_sfj08,PhyDamageMul,0.4,0\n\n" +
		"[13117]\nBelongCategory=2\nr=CS_jh_sfj04,GRushRange,10,0\n\n" +
		"[13223]\nPackCategory=5\nr=CS_jh_sfj01_1,PhyDamageMul,0.2,0\n"
	path := filepath.Join(dir, "skillpack.ini")
	if err := os.WriteFile(path, []byte(ini), 0o644); err != nil {
		t.Fatalf("write skillpack: %v", err)
	}
	return loadSkillAffixCatalog(path, stubSkillAffixTables())
}

func TestLoadSkillAffixCatalog(t *testing.T) {
	catalog := stubSkillAffixCatalog(t)
	// 威力包按 key 归集（升序去重；非威力包 13117 排除；hide 同包收录）。
	if got := catalog.packsBySkill["cs_jh_sfj04"]; !reflect.DeepEqual(got, []int{13221, 13222}) {
		t.Fatalf("sfj04 packs=%v", got)
	}
	if got := catalog.packsBySkill["cs_jh_sfj04_hide"]; !reflect.DeepEqual(got, []int{13221}) {
		t.Fatalf("sfj04_hide packs=%v", got)
	}
	if got := catalog.packsBySkill["cs_jh_sfj03"]; !reflect.DeepEqual(got, []int{13224}) {
		t.Fatalf("sfj03 packs=%v", got)
	}
	// 套路分组：_hide（gmTaoluVariantSkill）与 _1（LevelupRelevance 变体
	// 集）均剔除，只留基座招式（小写升序）。
	if got := catalog.taoluSkills["cs_jh_sfj"]; !reflect.DeepEqual(got, []string{"cs_jh_sfj03", "cs_jh_sfj04", "cs_jh_sfj08"}) {
		t.Fatalf("taolu skills=%v", got)
	}
	// 类型静态：虚招 sfj03 EffectType=2；怒招 sfj08 走 varprop→consume
	// AConsumeSP=50 判定（EffectType=1 红框但机制上怒气驱动）。
	if static := catalog.skillStatic["cs_jh_sfj03"]; static.Taolu != "cs_jh_sfj" || static.EffectType != 2 || static.Rage {
		t.Fatalf("sfj03 static=%+v", static)
	}
	if static := catalog.skillStatic["cs_jh_sfj08"]; !static.Rage {
		t.Fatalf("sfj08 static=%+v want rage", static)
	}
	if static := catalog.skillStatic["cs_jh_sfj04"]; static.Rage {
		t.Fatalf("sfj04 static=%+v want non-rage", static)
	}
}

func TestSkillAffixTypeLabel(t *testing.T) {
	cases := []struct {
		effectType int
		rage       bool
		want       string
	}{
		{1, false, "实招"}, {2, false, "虚招"}, {3, false, "架招"},
		{1, true, "怒招"}, // 机制口径优先于 EffectType 显示
		{0, false, "招式"},
	}
	for _, c := range cases {
		if got := skillAffixTypeLabel(c.effectType, c.rage); got != c.want {
			t.Fatalf("typeLabel(%d,%v)=%q want %q", c.effectType, c.rage, got, c.want)
		}
	}
}

func TestBuildGMSkillAffixes(t *testing.T) {
	catalog := stubSkillAffixCatalog(t)
	taolus := &gmTaoluCatalog{
		rows:  []gmTaoluRow{{Key: "cs_wd_tjq", Label: "太极剑法"}, {Key: "cs_jh_sfj", Label: "神风诀"}},
		byKey: map[string]gmTaoluRow{
			"cs_wd_tjq": {Key: "cs_wd_tjq", Label: "太极剑法"},
			"cs_jh_sfj": {Key: "cs_jh_sfj", Label: "神风诀"},
		},
	}
	names := map[string]string{"cs_jh_sfj04": "弄月吟风"}
	result := buildGMSkillAffixes(catalog, taolus, names)
	if !result.Loaded || len(result.Taolus) != 1 {
		t.Fatalf("taolus=%+v", result.Taolus)
	}
	// 行序跟 /api/taolus（太极剑法无威力招式跳过，神风诀保留）。
	row := result.Taolus[0]
	if row.Key != "cs_jh_sfj" || row.Label != "神风诀" || len(row.Skills) != 3 {
		t.Fatalf("row=%+v", row)
	}
	if row.Skills[0].Key != "cs_jh_sfj03" || row.Skills[0].Type != "虚招" || row.Skills[0].Name != "cs_jh_sfj03" {
		t.Fatalf("skill[0]=%+v（名缺 idres 回退 key）", row.Skills[0])
	}
	if row.Skills[1].Key != "cs_jh_sfj04" || row.Skills[1].Name != "弄月吟风" || row.Skills[1].Type != "实招" {
		t.Fatalf("skill[1]=%+v", row.Skills[1])
	}
	if row.Skills[2].Key != "cs_jh_sfj08" || row.Skills[2].Type != "怒招" {
		t.Fatalf("skill[2]=%+v", row.Skills[2])
	}
	// 空目录 loaded=false。
	if empty := buildGMSkillAffixes(&skillAffixCatalog{}, taolus, nil); empty.Loaded || len(empty.Taolus) != 0 {
		t.Fatalf("empty=%+v", empty)
	}
}

func TestGMTaoluSkillEntries(t *testing.T) {
	setSkillAffixCatalogForTest(stubSkillAffixCatalog(t))
	t.Cleanup(func() { setSkillAffixCatalogForTest(nil) })

	// 指定招式（maxTier）：每招最高档包，kind=2。
	entries, note := gmTaoluSkillEntries(gmAffixPlan{
		SkillTaolu: "cs_jh_sfj", SkillKeys: []string{"CS_jh_sfj04", "cs_jh_sfj03"}, MaxTier: true,
	})
	if note != "" || len(entries) != 2 ||
		entries[0] != (gmAffixEntry{Pack: 13222, Kind: 2}) || entries[1] != (gmAffixEntry{Pack: 13224, Kind: 2}) {
		t.Fatalf("entries=%v note=%q", entries, note)
	}

	// 指定招式含 miss：有效招式照发，miss 计数注明。
	entries, note = gmTaoluSkillEntries(gmAffixPlan{SkillTaolu: "cs_jh_sfj", SkillKeys: []string{"CS_missing"}, MaxTier: true})
	if len(entries) != 0 || note != "1 个招式无威力词条省略" {
		t.Fatalf("entries=%v note=%q", entries, note)
	}

	// 套路随机：2 招全抽（不放回、互不相同），档位取各招包。
	entries, _ = gmTaoluSkillEntries(gmAffixPlan{SkillTaolu: "cs_jh_sfj", Skill: 2, MaxTier: true})
	packs := map[int]bool{}
	for _, entry := range entries {
		if entry.Kind != 2 {
			t.Fatalf("entry=%v want kind 2", entry)
		}
		packs[entry.Pack] = true
	}
	if len(entries) != 2 || len(packs) != 2 {
		t.Fatalf("entries=%v（应 2 条不同招式）", entries)
	}

	// 套路无威力招式 / 表未加载。
	if _, note = gmTaoluSkillEntries(gmAffixPlan{SkillTaolu: "cs_unknown", Skill: 2}); note != "套路无威力词条招式，技能词条省略" {
		t.Fatalf("unknown taolu note=%q", note)
	}
	setSkillAffixCatalogForTest(&skillAffixCatalog{})
	if _, note = gmTaoluSkillEntries(gmAffixPlan{SkillTaolu: "cs_jh_sfj", Skill: 2}); note != "招式威力词条表未加载，技能词条省略" {
		t.Fatalf("unloaded note=%q", note)
	}
	setSkillAffixCatalogForTest(nil)
}

func TestApplyGMGenerateWeaponTaoluSkills(t *testing.T) {
	setWeaponRecastCatalogForTest(stubGMGenerateCatalog(t))
	t.Cleanup(func() { setWeaponRecastCatalogForTest(nil) })
	setSkillAffixCatalogForTest(stubSkillAffixCatalog(t))
	t.Cleanup(func() { setSkillAffixCatalogForTest(nil) })
	stubPlannedItems(t)
	h, player := newGMGenerateSession(t, false)

	// 端到端：套路定向 + 精确招式（sfj04 最高档 13222 / sfj03 13224，
	// kind=2 段）+ 伤害 1 条（dire 池最高档 13042）。
	text, err := applyGMGenerateWeapon(h, 102, "cook", gmAffixPlan{
		Damage: 1, MaxTier: true,
		SkillTaolu: "cs_jh_sfj", SkillKeys: []string{"CS_jh_sfj04", "CS_jh_sfj03"},
	})
	if err != nil {
		t.Fatalf("taolu weapon: %v", err)
	}
	if !strings.Contains(text, "词条 3 条") {
		t.Fatalf("text=%q", text)
	}
	row, _ := player.wornItemAt(22)
	got := map[string]bool{}
	for _, segment := range strings.Split(row.propBuilt, ";") {
		got[segment] = true
	}
	if !got["1,13042"] || !got["2,13222"] || !got["2,13224"] || len(got) != 3 {
		t.Fatalf("propBuilt=%q", row.propBuilt)
	}
}
