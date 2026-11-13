package main

// 招式威力词条目录：skillpack.ini 威力包（PackCategory=5）→ 技能 key →
// 套路分组 + 招式类型（实/虚/架/怒）。供两条链消费：
//
//   - /api/skill-affixes（面板「模板生成」技能词条的套路 → 招式两级选择）
//   - gm_generate_weapon 的 skillTaolu/skillKeys 定向构造（gm_generate_equip.go）
//
// 数据链（GBK ini + UTF-8 idres）：
//
//	skillpack.ini [pack] PackCategory=5，r=<技能key>,*DamageMul,…
//	  → packsBySkill（威力包按技能 key 归集；同包对基座与 _hide 变体
//	    常是多行并列——[13221] 同时列 CS_jh_sfj04 与 CS_jh_sfj04_hide，
//	    按基座 key 取包即全覆盖变体）
//	skill_new.ini [key] StaticData → skill_static.ini [static]
//	  TaoLu（套路分组，客户端套路面板同源）/ EffectType（1实/2虚/3架，
//	  tips_func_equip.lua 636-646 定版）/ PackCategory（==2 怒招，
//	  skill_combat.go 武技连击段同口径）
//	text/chineses/string_skill_desc.idres → 招式中文名
//
// 目录只列基座招式：变体经 gmTaoluVariantSkill（_hide/_sky/_npc/
// wuji_ 等前缀）+ LevelupRelevance 引用集（_1/_2 分层变体，如
// CS_jh_sfj01_1 挂在基座 12113 的 LevelupRelevance）双过滤。

import (
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// skillAffixStatic 技能 key 的套路/类型静态字段（skill_static 侧）。
type skillAffixStatic struct {
	Taolu      string // 套路 key（skill_static.TaoLu 原值）
	EffectType int    // 1=实招 2=虚招 3=架招
	Rage       bool   // 怒招（varprop→consume AConsumeSP>0，机制口径）
}

// skillAffixCatalog 威力词条目录（键统一小写归一——skillpack r= 行与
// skill_new 节名大小写一致，归一后手输小写也能命中）。
type skillAffixCatalog struct {
	packsBySkill map[string][]int            // 技能 key 小写 → 威力包号升序去重
	skillStatic  map[string]skillAffixStatic // 技能 key 小写 → 套路/类型（仅威力 key）
	taoluSkills  map[string][]string         // 套路 key 小写 → 基座威力招式 key（小写升序）
}

var (
	skillAffixOnce sync.Once
	skillAffixes   *skillAffixCatalog
)

// currentSkillAffixCatalog 懒加载全局目录（currentWeaponRecastCatalog
// 同模式；测试注入 setSkillAffixCatalogForTest 优先）。
func currentSkillAffixCatalog() *skillAffixCatalog {
	skillAffixOnce.Do(func() {
		if skillAffixes != nil {
			return
		}
		skillAffixes = loadSkillAffixCatalog(
			filepath.Join(defaultModernShareRoot, "modifypack", "skillpack.ini"),
			installedCombatSkillCatalog.tables)
	})
	return skillAffixes
}

// setSkillAffixCatalogForTest 单测注入合成目录（生产代码不得调用）。
func setSkillAffixCatalogForTest(catalog *skillAffixCatalog) {
	skillAffixOnce.Do(func() {})
	skillAffixes = catalog
}

// loadSkillAffixCatalog 纯函数装载：skillpack 威力包 + 技能静态表 →
// 目录。skillpack 缺失按空目录降级（nil 表容错）。
func loadSkillAffixCatalog(skillPackPath string, tables skillResourceTables) *skillAffixCatalog {
	catalog := &skillAffixCatalog{
		packsBySkill: map[string][]int{},
		skillStatic:  map[string]skillAffixStatic{},
		taoluSkills:  map[string][]string{},
	}
	data, err := gmReadINI(skillPackPath)
	if err != nil {
		log.Printf("skill affix catalog: skillpack unavailable: %v", err)
		return catalog
	}
	seen := map[string]map[int]bool{}
	sectionPack, sectionPower := "", false
	gmScanINI(data, func(section, key, value string) {
		if section != sectionPack {
			sectionPack, sectionPower = section, false
		}
		switch key {
		case "packcategory":
			sectionPower = strings.TrimSpace(value) == "5"
		case "r":
			if !sectionPower {
				return
			}
			pack, convErr := strconv.Atoi(section)
			if convErr != nil || pack <= 0 {
				return
			}
			skillKey := strings.ToLower(strings.TrimSpace(strings.Split(value, ",")[0]))
			if skillKey == "" {
				return
			}
			if seen[skillKey] == nil {
				seen[skillKey] = map[int]bool{}
			}
			if !seen[skillKey][pack] {
				seen[skillKey][pack] = true
				catalog.packsBySkill[skillKey] = append(catalog.packsBySkill[skillKey], pack)
			}
		}
	})
	for key := range catalog.packsBySkill {
		sort.Ints(catalog.packsBySkill[key])
	}

	// 套路/类型归组：skill_new → StaticData → skill_static；变体过滤后
	// 按套路分组（小写升序）。skillNew 节名原大小写（CS_jh_sfj04），
	// packsBySkill 键小写——建归一索引反查。
	skillNewLower := make(map[string][]iniField, len(tables.skillNew))
	for id, fields := range tables.skillNew {
		skillNewLower[strings.ToLower(id)] = fields
	}
	rageConsume := rageConsumeSet(tables)
	variants := levelupVariantKeys(tables)
	grouped := map[string]map[string]bool{}
	for key := range catalog.packsBySkill {
		fields, ok := skillNewLower[key]
		if !ok {
			continue
		}
		staticData := strings.TrimSpace(iniValue(fields, "StaticData"))
		staticFields := tables.skillStatic[staticData]
		taolu := strings.ToLower(strings.TrimSpace(iniValue(staticFields, "TaoLu")))
		if taolu == "" {
			continue
		}
		catalog.skillStatic[key] = skillAffixStatic{
			Taolu:      taolu,
			EffectType: int(iniInt(staticFields, "EffectType")),
			Rage:       rageSkillFlag(staticFields, iniValue(fields, "script"), tables, rageConsume),
		}
		if gmTaoluVariantSkill(key) || variants[key] {
			continue
		}
		if grouped[taolu] == nil {
			grouped[taolu] = map[string]bool{}
		}
		grouped[taolu][key] = true
	}
	for taolu, members := range grouped {
		keys := make([]string, 0, len(members))
		for key := range members {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		catalog.taoluSkills[taolu] = keys
	}
	return catalog
}

// levelupVariantKeys 收集 skill_static 全表 LevelupRelevance 引用的
// 变体 key（逗号列表，如 CS_jh_sfj01_1——基座升级分层/反打变体挂在
// 基座字段上，表自带挂钩，比后缀猜测可靠）。
func levelupVariantKeys(tables skillResourceTables) map[string]bool {
	variants := map[string]bool{}
	for _, fields := range tables.skillStatic {
		for _, ref := range strings.Split(iniValue(fields, "LevelupRelevance"), ",") {
			if ref = strings.ToLower(strings.TrimSpace(ref)); ref != "" {
				variants[ref] = true
			}
		}
	}
	return variants
}

// rageConsumeSet 怒气消耗行集合：consume 表 AConsumeSP>0 的节号
//（skill_combat.go spCost 同源数据）。
func rageConsumeSet(tables skillResourceTables) map[string]bool {
	rage := make(map[string]bool, len(tables.consume))
	for id, fields := range tables.consume {
		if iniInt(fields, "AConsumeSP") > 0 {
			rage[id] = true
		}
	}
	return rage
}

// rageSkillFlag 怒招判定（机制口径：怒气驱动）：static 的 varprop 区间
// [MinVarPropNo,MaxVarPropNo] 内任一行 Consume 指向 AConsumeSP>0 的
// consume 行。位置特征不可靠——skill_static.PackCategory=2 是普通招式
// 公共值、wuji_ 前缀只覆盖古谱武技族、TargetType=8 虚招也大量命中
//（2026-11-13 全量核对）。
func rageSkillFlag(static []iniField, script string, tables skillResourceTables, rageConsume map[string]bool) bool {
	var table iniTable
	if strings.EqualFold(script, "SkillLock") {
		table = tables.lockVar
	} else {
		table = tables.normalVar
	}
	minimum, maximum := iniInt(static, "MinVarPropNo"), iniInt(static, "MaxVarPropNo")
	for id := minimum; id <= maximum && id > 0; id++ {
		consume := strings.TrimSpace(iniValue(table[strconv.Itoa(int(id))], "Consume"))
		if consume != "" && rageConsume[consume] {
			return true
		}
	}
	return false
}

// skillAffixTypeLabel 招式类型文案：EffectType 1=实(红) 2=虚(蓝) 3=架
//（绿）（tips_func_equip.lua 641-646 / form_wuxue_skill.lua 2211-2221
// 定版）；怒招按 AConsumeSP 机制判定，优先于 EffectType 显示。
func skillAffixTypeLabel(effectType int, rage bool) string {
	if rage {
		return "怒招"
	}
	switch effectType {
	case 1:
		return "实招"
	case 2:
		return "虚招"
	case 3:
		return "架招"
	}
	return "招式"
}

// ---- 招式中文名表（string_skill_desc.idres，UTF-8） ----

var (
	skillDescNamesOnce sync.Once
	skillDescNames     map[string]string
)

// currentSkillDescNames 懒加载招式名表（loadStringNameIDRes 同解析：
// 键小写、重复取首值；表缺失按空表降级）。
func currentSkillDescNames() map[string]string {
	skillDescNamesOnce.Do(func() {
		skillDescNames = map[string]string{}
		tables := &sceneNameTables{nameByKey: map[string]string{}}
		path := filepath.Join(defaultModernTextRoot, "chineses", "string_skill_desc.idres")
		if err := loadStringNameIDRes(path, tables); err != nil {
			log.Printf("skill affix catalog: string_skill_desc unavailable: %v", err)
			return
		}
		skillDescNames = tables.nameByKey
	})
	return skillDescNames
}

// skillAffixDisplayName 招式显示名：string_skill_desc[key] → key。
func skillAffixDisplayName(key string) string {
	if name := currentSkillDescNames()[strings.ToLower(key)]; name != "" {
		return name
	}
	return key
}
