package main

// GM 套路/内功发放目录单测：GBK 解析、skill= 逗号表、显示名链
// （key → book_key → 原样）、注释节跳过、重复节首见优先、resolve
// 小写兜底；内功行目录缺席时为空表（资源环境另有真实树冒烟）。

import (
	"strings"
	"testing"
)

func TestLoadGMTaoluCatalogFrom(t *testing.T) {
	names := map[string]string{
		"cs_wd_tjq":     "太极拳",
		"book_cs_dummy": "书名兜底",
	}
	school := `
[//注释节不进目录]
[cs_wd_tjq]
skill=CS_wd_tjq01, CS_wd_tjq02,,CS_wd_tjq03
attack=9

[cs_dummy]
skill=dmy01,dmy02
`
	shijia := `
[CS_dummy]
skill=CS_dummy_A
`
	catalog := loadGMTaoluCatalogFrom(names,
		strings.NewReader(string(gbkBytes(school))),
		strings.NewReader(string(gbkBytes(shijia))))
	if len(catalog.rows) != 2 {
		t.Fatalf("rows=%d want 2: %+v", len(catalog.rows), catalog.rows)
	}
	first := catalog.rows[0]
	if first.Key != "cs_wd_tjq" || first.Label != "太极拳" {
		t.Fatalf("first row key/label=%q/%q", first.Key, first.Label)
	}
	if len(first.Skills) != 3 || first.Skills[0] != "CS_wd_tjq01" || first.Skills[2] != "CS_wd_tjq03" {
		t.Fatalf("skills=%v want trimmed 3-item list", first.Skills)
	}
	// shijia 的重复 key（大小写不同）首见优先：保持 school 版两招。
	second := catalog.rows[1]
	if second.Key != "cs_dummy" || second.Label != "书名兜底" || len(second.Skills) != 2 {
		t.Fatalf("second row=%+v want school first-seen with book_ label", second)
	}
	// resolve：精确与小写均可命中；未知 key 拒绝。
	if row, ok := catalog.resolve("CS_WD_TJQ"); !ok || row.Key != "cs_wd_tjq" {
		t.Fatalf("case-insensitive resolve failed: %+v %t", row, ok)
	}
	if _, ok := catalog.resolve("no_such"); ok {
		t.Fatal("unknown key must not resolve")
	}
	if _, ok := catalog.resolve(""); ok {
		t.Fatal("empty key must not resolve")
	}
}

func TestLoadGMTaoluCatalogLabelFallback(t *testing.T) {
	catalog := loadGMTaoluCatalogFrom(map[string]string{},
		strings.NewReader("[cs_x]\nskill=a,b\n"))
	if len(catalog.rows) != 1 || catalog.rows[0].Label != "cs_x" {
		t.Fatalf("label fallback to raw key failed: %+v", catalog.rows)
	}
}

func TestLoadGMTaoluCatalogSkipsEmptySkillSections(t *testing.T) {
	catalog := loadGMTaoluCatalogFrom(nil,
		strings.NewReader("[no_skills]\nattack=5\n[has_skills]\nskill=a\n"))
	if len(catalog.rows) != 1 || catalog.rows[0].Key != "has_skills" {
		t.Fatalf("empty-skill section must be skipped: %+v", catalog.rows)
	}
}

func TestLoadGMNeiGongRows(t *testing.T) {
	rows := loadGMNeiGongRows()
	if len(rows) == 0 {
		t.Skip("neigong catalog unavailable in fixture")
	}
	// 排序 + 字段完整性：configId 单调、满层 ≥1、显示名非空。
	for index := 1; index < len(rows); index++ {
		if rows[index-1].ConfigID >= rows[index].ConfigID {
			t.Fatalf("rows not sorted at %d: %s >= %s", index, rows[index-1].ConfigID, rows[index].ConfigID)
		}
	}
	for _, row := range rows {
		if !strings.HasPrefix(row.ConfigID, "ng_") {
			t.Fatalf("non ng_ row: %+v", row)
		}
		if row.MaxLevel < 1 {
			t.Fatalf("maxLevel<1: %+v", row)
		}
		if row.Name == "" {
			t.Fatalf("empty name: %+v", row)
		}
	}
}

func TestRebuildPlaceholderTaolus(t *testing.T) {
	catalog := loadGMTaoluCatalogFrom(nil,
		strings.NewReader("[cs_real]\nskill=Real_A,real_b\n[cs_tjq]\nskill=CS_sl_001,CS_sl_002\n[cs_dead]\nskill=CS_sl_001\n"))
	skillCatalog := &combatSkillCatalog{tables: skillResourceTables{
		skillNew: iniTable{
			"real_a": {{key: "StaticData", value: "10"}},
			"REAL_B": {{key: "StaticData", value: "11"}},
			// tjq 静态分组：基座两招 + wuji/_hide/_sky/_npc 变体全被剔。
			"CS_tjq01":      {{key: "StaticData", value: "20"}},
			"CS_tjq02":      {{key: "StaticData", value: "21"}},
			"wuji_CS_tjq01": {{key: "StaticData", value: "22"}},
			"CS_tjq02_hide": {{key: "StaticData", value: "23"}},
			"CS_tjq02_sky":  {{key: "StaticData", value: "24"}},
			"CS_tjq02_npc":  {{key: "StaticData", value: "25"}},
		},
		skillStatic: iniTable{
			"10": {{key: "TaoLu", value: "cs_other"}},
			"11": {{key: "TaoLu", value: "cs_other"}},
			"20": {{key: "TaoLu", value: "cs_tjq"}},
			"21": {{key: "TaoLu", value: "cs_tjq"}},
			"22": {{key: "TaoLu", value: "cs_tjq"}},
			"23": {{key: "TaoLu", value: "cs_tjq"}},
			"24": {{key: "TaoLu", value: "cs_tjq"}},
			"25": {{key: "TaoLu", value: "cs_tjq"}},
		},
	}}
	catalog.rebuildPlaceholderTaolus(skillCatalog)
	if len(catalog.rows) != 2 {
		t.Fatalf("rows=%d want 2 (real + rebuilt, dead dropped): %+v", len(catalog.rows), catalog.rows)
	}
	if catalog.rows[0].Key != "cs_real" || len(catalog.rows[0].Skills) != 2 {
		t.Fatalf("indexed row must keep school list: %+v", catalog.rows[0])
	}
	// 占位行按静态分组重建：基座招式（原大小写）+ 变体剔除。
	rebuilt := catalog.rows[1]
	if rebuilt.Key != "cs_tjq" {
		t.Fatalf("rebuilt row=%+v", rebuilt)
	}
	if len(rebuilt.Skills) != 2 || rebuilt.Skills[0] != "CS_tjq01" || rebuilt.Skills[1] != "CS_tjq02" {
		t.Fatalf("rebuilt skills=%v want base members in canonical case", rebuilt.Skills)
	}
	if _, ok := catalog.resolve("cs_dead"); ok {
		t.Fatal("placeholder without static group must be dropped")
	}
}
