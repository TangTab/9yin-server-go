package main

// GM 套路/内功发放目录（Java gm-web GmCatalog.taolus 的 Go 移植 + 内功
// 目录行扩展）：面板「武学套路」区块的「整套学习」下拉数据源。
//
//   - 套路表 = ini/ui/wuxue/school_taolu.ini + ini/shijia/taolu_info.ini
//     （均 GBK，节名=套路 key 如 cs_sl_djgq，skill=逗号分隔的招式 ConfigID
//     列表）。该列表是客户端套路面板的精选集；skill_static.ini 的 TaoLu
//     分组字段是客户端实际分组来源，但混有 wuji_*/_hide/_sky 变体行
//     （2026-11-12 交叉验证 124 套路全量：school 表恒为 static 分组的
//     子集）。发放按 school 表走；18 个 school 模板占位节（skill=
//     CS_sl_001,CS_sl_002）按 static 分组基座招式重建（见
//     rebuildPlaceholderTaolus）。
//   - 显示名链 = stringname.idres[key] → stringname.idres[book_key] →
//     key 原样（Java idresLookup 同构；如 cs_sl_djgq=太祖长拳）。
//   - 内功行 = neigong.ini 目录（modernNeiGongCatalog.books）+ stringname
//     显示名 + varprop 区间满级（目录缺席时为空表，面板显示「无」）。

import (
	"bufio"
	"bytes"
	"io"
	"sort"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var (
	modernSchoolTaoluPath = runtimeProjectPath("resources", "modern", "ini", "ui", "wuxue", "school_taolu.ini")
	modernShiJiaTaoluPath = runtimeProjectPath("resources", "modern", "ini", "shijia", "taolu_info.ini")
)

// gmTaoluCatalog 是套路目录：行序=文件装载序（school_taolu 在前、
// shijia 追加在后，重复 key 首见优先），byKey 小写索引供命令解析。
type gmTaoluCatalog struct {
	rows  []gmTaoluRow
	byKey map[string]gmTaoluRow
}

// gmTaolus 全局目录（gmItems 同款包级缓存；资源缺失=空目录）。
var gmTaolus = loadGMTaoluCatalog()

// loadGMTaoluCatalog 从现代资源树装载（文件缺失/读失败按空表降级）。
// 末段重建占位套路：school_taolu.ini 混有 18 个模板节（skill=
// CS_sl_001,CS_sl_002——两 ID 不存在于任何技能表，2026-11-12 真机树
// 全量核对），其中不乏太极拳(古谱)/苍阳诀等真套路——客户端实际按
// skill_static.ini 的 TaoLu 字段分组，故按静态分组重建招式表。
func loadGMTaoluCatalog() *gmTaoluCatalog {
	sources := make([]io.Reader, 0, 2)
	for _, path := range []string{modernSchoolTaoluPath, modernShiJiaTaoluPath} {
		data, err := readResourceBytes(path)
		if err != nil {
			continue
		}
		sources = append(sources, bytes.NewReader(data))
	}
	catalog := loadGMTaoluCatalogFrom(sceneNameTablesOf().nameByKey, sources...)
	catalog.rebuildPlaceholderTaolus(installedCombatSkillCatalog)
	return catalog
}

// rebuildPlaceholderTaolus 修补占位套路行：skill= 全部不在 skill_new.ini
// 的行（模板占位）改为按 skill_static TaoLu 分组的基座招式（剔除
// wuji_/xjskill_/wine_skill_ 前缀与 _hide/_sky/_npc 变体——真机树
// 337 个变体 ID 全数命中这三类）；分组也为空才整行剔除。
func (catalog *gmTaoluCatalog) rebuildPlaceholderTaolus(skillCatalog *combatSkillCatalog) {
	if catalog == nil || skillCatalog == nil || len(catalog.rows) == 0 {
		return
	}
	known := make(map[string]bool, len(skillCatalog.tables.skillNew))
	for id := range skillCatalog.tables.skillNew {
		known[strings.ToLower(id)] = true
	}
	groups := skillStaticTaoLuGroups(skillCatalog.tables)
	kept := make([]gmTaoluRow, 0, len(catalog.rows))
	byKey := make(map[string]gmTaoluRow, len(catalog.rows))
	for _, row := range catalog.rows {
		indexed := false
		for _, id := range row.Skills {
			if known[strings.ToLower(id)] {
				indexed = true
				break
			}
		}
		if indexed {
			kept = append(kept, row)
			byKey[strings.ToLower(row.Key)] = row
			continue
		}
		members := groups[strings.ToLower(row.Key)]
		if len(members) == 0 {
			continue // 真占位（无任何静态分组）：剔除死条目。
		}
		row.Skills = members
		kept = append(kept, row)
		byKey[strings.ToLower(row.Key)] = row
	}
	catalog.rows, catalog.byKey = kept, byKey
}

// skillStaticTaoLuGroups skill_static TaoLu 分组（套路 key 小写 →
// 基座招式 ConfigID 原大小写、按小写升序）：skill_new → StaticData →
// skill_static TaoLu，与客户端套路面板同一分组来源（skill_view.go 同
// 口径）。成员保留原大小写——已学表/View40 行以 ConfigID 为键，starter
// 集是混合大小写，改小写会造成重复行。
func skillStaticTaoLuGroups(tables skillResourceTables) map[string][]string {
	grouped := make(map[string]map[string]string)
	for configID, fields := range tables.skillNew {
		staticData := iniValue(fields, "StaticData")
		if staticData == "" {
			continue
		}
		taolu := strings.TrimSpace(iniValue(tables.skillStatic[staticData], "TaoLu"))
		if taolu == "" {
			continue
		}
		lower := strings.ToLower(configID)
		if gmTaoluVariantSkill(lower) {
			continue
		}
		key := strings.ToLower(taolu)
		if grouped[key] == nil {
			grouped[key] = make(map[string]string)
		}
		grouped[key][lower] = configID
	}
	groups := make(map[string][]string, len(grouped))
	for key, members := range grouped {
		keys := make([]string, 0, len(members))
		for lower := range members {
			keys = append(keys, lower)
		}
		sort.Strings(keys)
		list := make([]string, 0, len(keys))
		for _, lower := range keys {
			list = append(list, members[lower])
		}
		groups[key] = list
	}
	return groups
}

// gmTaoluVariantSkill 报告 skill_new 节名是否为套路变体（wuji_ 进阶/
// xjskill_ 心法演出/wine_skill_ 酒技门面/_hide/_sky/_npc 隐藏变体），
// 整套发放只发基座招式。
func gmTaoluVariantSkill(lowerID string) bool {
	if strings.HasPrefix(lowerID, "wuji_") || strings.HasPrefix(lowerID, "xjskill_") || strings.HasPrefix(lowerID, "wine_skill_") {
		return true
	}
	return strings.Contains(lowerID, "_hide") || strings.HasSuffix(lowerID, "_sky") || strings.HasSuffix(lowerID, "_npc")
}

// loadGMTaoluCatalogFrom 纯函数装载：GBK 解码 → 逐节取 skill= 逗号表 →
// idres 名链。注释节（//开头）与空 skill 行跳过；同 key 重复节首见优先。
func loadGMTaoluCatalogFrom(names map[string]string, sources ...io.Reader) *gmTaoluCatalog {
	catalog := &gmTaoluCatalog{byKey: make(map[string]gmTaoluRow)}
	for _, source := range sources {
		data, err := io.ReadAll(source)
		if err != nil {
			continue
		}
		if decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data); err == nil {
			data = decoded
		}
		section := ""
		skills := make([]string, 0, 8)
		flush := func() {
			if section == "" || len(skills) == 0 {
				return
			}
			if _, exists := catalog.byKey[strings.ToLower(section)]; exists {
				return
			}
			// 拷贝入行：skills 缓冲被下一节复用（skills[:0] 重置），直接
			// 引用会让已收录行被后续节的解析覆写（同底层数组别名）。
			stored := append([]string(nil), skills...)
			row := gmTaoluRow{Key: section, Label: gmTaoluLabel(section, names), Skills: stored}
			catalog.rows = append(catalog.rows, row)
			catalog.byKey[strings.ToLower(section)] = row
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
			if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				flush()
				section = strings.TrimSpace(line[1 : len(line)-1])
				skills = skills[:0]
				continue
			}
			if section == "" {
				continue
			}
			key, value, found := strings.Cut(line, "=")
			if !found || !strings.EqualFold(strings.TrimSpace(key), "skill") {
				continue
			}
			for _, item := range strings.Split(value, ",") {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				skills = append(skills, item)
			}
		}
		flush()
	}
	return catalog
}

// gmTaoluLabel 套路显示名链：stringname[key] → stringname[book_key] →
// key 原样（Java loadTaoluIni 同构）。
func gmTaoluLabel(key string, names map[string]string) string {
	if name := names[strings.ToLower(key)]; name != "" {
		return name
	}
	if name := names["book_"+strings.ToLower(key)]; name != "" {
		return name
	}
	return key
}

// resolve 按套路 key 解析（精确命中优先，小写兜底；面板下拉传回原 key）。
func (catalog *gmTaoluCatalog) resolve(key string) (gmTaoluRow, bool) {
	if catalog == nil || key == "" {
		return gmTaoluRow{}, false
	}
	if row, exists := catalog.byKey[strings.ToLower(strings.TrimSpace(key))]; exists {
		return row, true
	}
	return gmTaoluRow{}, false
}

// gmNeiGongRow 是 /api/neigongs 的行形态（「整套学习」内功下拉）：
// configId=neigong.ini 书 ConfigID（ng_*），name=stringname 显示名，
// maxLevel=varprop 区间满级（面板层数输入的参考上限）。
type gmNeiGongRow struct {
	ConfigID string `json:"configId"`
	Name     string `json:"name"`
	MaxLevel int32  `json:"maxLevel"`
	School   string `json:"school"`
}

// loadGMNeiGongRows 内功目录行：modernNeiGongCatalog.books 按 configID
// 排序；目录不可用（缺资源/夹具环境）返回空表，面板显示「无」。
func loadGMNeiGongRows() []gmNeiGongRow {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		return []gmNeiGongRow{}
	}
	names := sceneNameTablesOf().nameByKey
	rows := make([]gmNeiGongRow, 0, len(catalog.books))
	for _, book := range catalog.books {
		maxLevel := catalog.maxLevelOf(book.staticData)
		if maxLevel < 1 {
			maxLevel = 1
		}
		name := names[strings.ToLower(book.configID)]
		if name == "" {
			name = book.configID
		}
		rows = append(rows, gmNeiGongRow{
			ConfigID: book.configID, Name: name, MaxLevel: maxLevel, School: book.school,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ConfigID < rows[j].ConfigID })
	return rows
}
