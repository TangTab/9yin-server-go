package main

// 道具使用三目录（移植 Java ItemUseCatalog/SkillBookCatalog/SkillGateTable）：
//   - itemUseCatalog：giftbox.ini + tool_item.ini 箱类（script/DropID/
//     @CompensateItemRec 三键）+ item_use_skill_static.ini（itemType→技能
//     回血阶梯）；GBK 键/节名 ASCII 直读（名值 GBK 不消费）。
//   - skillBookCatalog：tool_item.ini 学习类（LearnSkillPack →
//     item_learn_skill_static.ini [packId] TragetID——官方拼写）+ 分层字段
//     SkillMaxLevel/AddLevel/AddMaxLevel/UseCondition。
//   - skillGateTable：rule/condition.ini 门槛（Type3 修炼上限区间 /
//     Type1 门派 / Type19 时代放行）。
// 文件缺失/脏行容错为空目录（使用链按 unknown 拒绝并记日志）。

import (
	"log"
	"path/filepath"
	"strconv"
	"strings"
)

// ---- itemUseCatalog ----

type itemUseKind uint8

const (
	itemUseBox itemUseKind = iota
	itemUseSkill
)

// itemUseEntry 解析后的使用语义：Box 携带 1..N 个掉落包 id（各 roll 一次）；
// UseSkill 携带技能档位（走 fallbackHealHp 文档化阶梯）。
type itemUseEntry struct {
	kind       itemUseKind
	dropIDs    []string
	skillID    string
	skillLevel int32
}

type itemUseCatalog struct {
	boxes     map[string]itemUseEntry // configId（小写）→ BOX
	useSkills map[int32]itemUseEntry  // itemType → USE_SKILL
}

// loadItemUseCatalogFromTables 由已解析 ini 表构建（测试注入面）。
func loadItemUseCatalogFromTables(giftbox, toolItem, useSkillStatic iniTable) *itemUseCatalog {
	catalog := &itemUseCatalog{
		boxes:     make(map[string]itemUseEntry),
		useSkills: make(map[int32]itemUseEntry),
	}
	// giftbox.ini（项目自建 UTF-8 表）：[configId] DropID=…
	for section, fields := range giftbox {
		if section == "" || strings.HasPrefix(section, "//") || strings.HasPrefix(section, "-") {
			continue
		}
		dropID := strings.TrimSpace(iniValue(fields, "DropID"))
		if dropID == "" {
			continue
		}
		catalog.boxes[lowerASCII(section)] = itemUseEntry{kind: itemUseBox, dropIDs: []string{dropID}}
	}
	// tool_item.ini：script=BoxItem(DropID) / script=CompensateItem(@Rec refs 全发)
	type toolFields struct {
		script, dropID string
		refs           []string
	}
	raw := make(map[string]*toolFields)
	for section, fields := range toolItem {
		if section == "" || strings.HasPrefix(section, "//") || strings.HasPrefix(section, ";") {
			continue
		}
		key := lowerASCII(section)
		entry, exists := raw[key]
		if !exists {
			entry = &toolFields{}
			raw[key] = entry
		}
		for _, field := range fields {
			switch lowerASCII(field.key) {
			case "script":
				entry.script = field.value
			case "dropid":
				entry.dropID = field.value
			default:
				if strings.HasPrefix(lowerASCII(field.key), "@compensateitemrec") {
					value := strings.TrimSpace(field.value)
					if comma := strings.Index(value, ","); comma >= 0 {
						value = strings.TrimSpace(value[comma+1:])
					}
					if value != "" {
						entry.refs = append(entry.refs, value)
					}
				}
			}
		}
	}
	for key, entry := range raw {
		if entry.script == "" {
			continue
		}
		script := strings.TrimSpace(entry.script)
		switch {
		case strings.EqualFold(script, "BoxItem") && strings.TrimSpace(entry.dropID) != "":
			catalog.boxes[key] = itemUseEntry{kind: itemUseBox, dropIDs: []string{strings.TrimSpace(entry.dropID)}}
		case strings.EqualFold(script, "CompensateItem") && len(entry.refs) > 0:
			refs := make([]string, 0, len(entry.refs))
			seen := make(map[string]bool, len(entry.refs))
			for _, ref := range entry.refs {
				ref = strings.TrimSpace(ref)
				if ref != "" && !seen[ref] {
					seen[ref] = true
					refs = append(refs, ref)
				}
			}
			if len(refs) > 0 {
				catalog.boxes[key] = itemUseEntry{kind: itemUseBox, dropIDs: refs}
			}
		}
	}
	// item_use_skill_static.ini：[itemType] SkillID/SkillLevel
	type skillFields struct{ id, level string }
	rawSkills := make(map[int32]*skillFields)
	for section, fields := range useSkillStatic {
		itemType, err := strconv.Atoi(strings.TrimSpace(section))
		if err != nil {
			continue // [-1] 字段说明节
		}
		entry, exists := rawSkills[int32(itemType)]
		if !exists {
			entry = &skillFields{}
			rawSkills[int32(itemType)] = entry
		}
		for _, field := range fields {
			switch {
			case strings.EqualFold(field.key, "SkillID"):
				entry.id = field.value
			case strings.EqualFold(field.key, "SkillLevel"):
				entry.level = field.value
			}
		}
	}
	for itemType, entry := range rawSkills {
		if strings.TrimSpace(entry.id) == "" {
			continue
		}
		level := int32(1)
		if parsed, err := strconv.Atoi(strings.TrimSpace(entry.level)); err == nil {
			if parsed < 1 {
				parsed = 1
			}
			if parsed > 9 {
				parsed = 9
			}
			level = int32(parsed)
		}
		catalog.useSkills[itemType] = itemUseEntry{
			kind: itemUseSkill, skillID: strings.TrimSpace(entry.id), skillLevel: level,
		}
	}
	return catalog
}

// resolve 按 configId（箱类）→ itemType（使用技能）顺序解析；未知返回 ok=false。
func (c *itemUseCatalog) resolve(configID string, itemType int32) (itemUseEntry, bool) {
	if configID != "" {
		if entry, exists := c.boxes[lowerASCII(configID)]; exists {
			return entry, true
		}
	}
	entry, exists := c.useSkills[itemType]
	return entry, exists
}

// fallbackHealHp 使用技能回血阶梯（yaopinhp* 无 varprop 数值时的文档化默认；
// L1 轻伤 → L9 大还，节奏对齐药品分度描述）。
func fallbackHealHp(skillLevel int32) int32 {
	if skillLevel < 1 {
		skillLevel = 1
	}
	if skillLevel > 9 {
		skillLevel = 9
	}
	switch skillLevel {
	case 1:
		return 100
	case 2:
		return 200
	case 3:
		return 400
	case 4:
		return 600
	case 5:
		return 800
	case 6:
		return 1000
	case 7:
		return 1500
	case 8:
		return 2000
	default:
		return 3000
	}
}

func fallbackHealMp(skillLevel int32) int32 {
	return fallbackHealHp(skillLevel) / 2
}

// ---- skillBookCatalog ----

// skillBookEntry 秘籍条目：pack 指向 + 分层字段（≤0=无该字段；纯学会型书）。
type skillBookEntry struct {
	pack          string
	targets       []string
	skillMaxLevel int32
	addLevel      int32
	addMaxLevel   int32
	useCondition  int32
}

type skillBookCatalog struct {
	entries map[string]skillBookEntry // configId（小写）→ 条目
}

// isLearnScript script 是否学习类（msg31 分派：三类都在 useItemEffect 之前拦截）。
func isLearnScript(script string) bool {
	switch lowerASCII(strings.TrimSpace(script)) {
	case "skillbook", "skillpage", "skillscroll":
		return true
	}
	return false
}

// loadSkillBookCatalogFromTables 由 tool_item.ini + item_learn_skill_static.ini 构建。
func loadSkillBookCatalogFromTables(toolItem, learnStatic iniTable) *skillBookCatalog {
	type bookFields struct{ script, pack, smax, addLevel, addMax, cond string }
	raw := make(map[string]*bookFields)
	for section, fields := range toolItem {
		if section == "" || strings.HasPrefix(section, "//") || strings.HasPrefix(section, ";") {
			continue
		}
		entry, exists := raw[section]
		if !exists {
			entry = &bookFields{}
			raw[section] = entry
		}
		for _, field := range fields {
			switch lowerASCII(field.key) {
			case "script":
				entry.script = field.value
			case "learnskillpack":
				entry.pack = field.value
			case "skillmaxlevel":
				entry.smax = field.value
			case "addlevel":
				entry.addLevel = field.value
			case "addmaxlevel":
				entry.addMax = field.value
			case "usecondition":
				entry.cond = field.value
			}
		}
	}
	targets := make(map[string][]string)
	for section, fields := range learnStatic {
		if section == "" || strings.HasPrefix(section, "-") {
			continue
		}
		value := strings.TrimSpace(iniValue(fields, "TragetID")) // 官方拼写
		if value == "" {
			continue
		}
		ids := make([]string, 0, 4)
		for _, token := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' }) {
			id := strings.TrimSpace(token)
			if id != "" {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			targets[section] = ids
		}
	}
	catalog := &skillBookCatalog{entries: make(map[string]skillBookEntry)}
	for section, entry := range raw {
		if !isLearnScript(entry.script) || strings.TrimSpace(entry.pack) == "" {
			continue
		}
		pack := strings.TrimSpace(entry.pack)
		target, exists := targets[pack]
		if !exists {
			continue // 无目标表行的书移除（使用链按未知拒绝）
		}
		catalog.entries[lowerASCII(section)] = skillBookEntry{
			pack: pack, targets: target,
			skillMaxLevel: atoiOrZero(entry.smax),
			addLevel:      atoiOrZero(entry.addLevel),
			addMaxLevel:   atoiOrZero(entry.addMax),
			useCondition:  atoiOrZero(entry.cond),
		}
	}
	return catalog
}

func (c *skillBookCatalog) resolve(configID string) (skillBookEntry, bool) {
	if configID == "" {
		return skillBookEntry{}, false
	}
	entry, exists := c.entries[lowerASCII(configID)]
	return entry, exists
}

// ---- skillGateTable ----

// skillGate 解析后的条件行（min/max 保留原文：Type=1 的 min 是 school_* 编码）。
type skillGate struct {
	gateType string
	name     string
	para1    string
	para2    string
	min      string
	max      string
}

// containerCapInRange 修炼上限类条件（Type=3 *Container MaxLevel）：cap ∈ [min,max]。
func (g skillGate) containerCapInRange(cap int32) bool {
	lo, err := strconv.Atoi(strings.TrimSpace(g.min))
	if err != nil {
		return false
	}
	hi, hiErr := strconv.Atoi(strings.TrimSpace(g.max))
	if cap < int32(lo) {
		return false
	}
	if hiErr == nil && cap > int32(hi) {
		return false
	}
	return true
}

type skillGateTable struct {
	gates map[int32]skillGate
}

func loadSkillGateTableFromTables(condition iniTable) *skillGateTable {
	table := &skillGateTable{gates: make(map[int32]skillGate)}
	for section, fields := range condition {
		id, err := strconv.Atoi(strings.TrimSpace(section))
		if err != nil || id <= 0 {
			continue // [-1] 字段说明节
		}
		gate := table.gates[int32(id)]
		for _, field := range fields {
			switch lowerASCII(field.key) {
			case "type":
				gate.gateType = field.value
			case "name":
				gate.name = field.value
			case "para1":
				gate.para1 = field.value
			case "para2":
				gate.para2 = field.value
			case "min":
				gate.min = field.value
			case "max":
				gate.max = field.value
			}
		}
		table.gates[int32(id)] = gate
	}
	return table
}

func (t *skillGateTable) gate(conditionID int32) (skillGate, bool) {
	gate, exists := t.gates[conditionID]
	return gate, exists
}

func atoiOrZero(value string) int32 {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return int32(parsed)
}

// ---- 装载（资源仓可选加载；夹具缺表时为空目录） ----

var (
	itemUseCatalogLoaded   *itemUseCatalog
	skillBookCatalogLoaded *skillBookCatalog
	skillGateTableLoaded   *skillGateTable
)

func loadOptionalINI(name string, parts ...string) iniTable {
	path := filepath.Join(append(parts, name)...)
	table, err := loadINISections(path)
	if err != nil {
		if errorsIsNotExist(err) {
			return make(iniTable)
		}
		log.Printf("item table %s unreadable (%v); degraded to empty", name, err)
		return make(iniTable)
	}
	return table
}

func currentItemUseCatalog() *itemUseCatalog {
	if itemCatalogOnce != nil {
		return itemCatalogOnce()
	}
	if itemUseCatalogLoaded == nil {
		itemRoot := filepath.Join(defaultModernShareRoot, "item")
		itemUseCatalogLoaded = loadItemUseCatalogFromTables(
			loadOptionalINI("giftbox.ini", itemRoot),
			loadOptionalINI("tool_item.ini", itemRoot),
			loadOptionalINI("item_use_skill_static.ini", itemRoot))
		log.Printf("item use catalog ready: boxes=%d useSkillTypes=%d",
			len(itemUseCatalogLoaded.boxes), len(itemUseCatalogLoaded.useSkills))
	}
	return itemUseCatalogLoaded
}

func currentSkillBookCatalog() *skillBookCatalog {
	if itemSkillBooksOnce != nil {
		return itemSkillBooksOnce()
	}
	if skillBookCatalogLoaded == nil {
		itemRoot := filepath.Join(defaultModernShareRoot, "item")
		skillBookCatalogLoaded = loadSkillBookCatalogFromTables(
			loadOptionalINI("tool_item.ini", itemRoot),
			loadOptionalINI("item_learn_skill_static.ini", itemRoot))
		log.Printf("skill book catalog ready: books=%d", len(skillBookCatalogLoaded.entries))
	}
	return skillBookCatalogLoaded
}

func currentSkillGateTable() *skillGateTable {
	if itemSkillGatesOnce != nil {
		return itemSkillGatesOnce()
	}
	if skillGateTableLoaded == nil {
		skillGateTableLoaded = loadSkillGateTableFromTables(
			loadOptionalINI("condition.ini", defaultModernShareRoot, "rule"))
		log.Printf("skill gate table ready: %d conditions", len(skillGateTableLoaded.gates))
	}
	return skillGateTableLoaded
}
