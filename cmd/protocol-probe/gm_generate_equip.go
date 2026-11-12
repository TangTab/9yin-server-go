package main

// GM 模板生成（带词条装备）三动作（面板「装备与铸造」组；Java LoginFlow
// applyGenerateWeapon/Armor/Bracer L5615-5718 蓝本）：
//
//	gm_generate_weapon  weaponType(101刀/102剑/103匕首/104双刀/105双剑/
//	                   106双刺/107长棍/108短棍/122重剑)+crafter(cook/smith)
//	                   → 挑 _60501 家族（51 级金色生活技能装）模板，词条抽
//	                   官方洗练池 senior_weapon_rand.ini（厨师 6 条混合池；
//	                   铁匠 3 条混合池 + 3 条招式池），立即穿戴。
//	gm_generate_armor  armorSlot(9 部位) → 固定 5 条红词条
//	                   （外功防御+46/+47、体魄+29、内息+39、臂力+35）。
//	gm_generate_bracer 无参 → 护腕模板 + 固定 3 条臂力（+33/+35/+35）
//	                   + 3 条招式词条（按身上已穿武器取池；未穿省略）。
//
// 词条通道与武器重铸（weapon_recast.go）同链：affixes JSON（kind 1|2 按
// proppack/skillpack 归属）+ PropBuilt@1954 穿戴行帧；模板挑选遍历 gmItems
// 目录（equipment.ini 全量），60501 家族优先、品质降序头部 8 窗口回退。

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// gmWeaponPrefix 武器类型 → 模型前缀（Java LoginFlow.WEAPON_PREFIX 同表）。
var gmWeaponPrefix = map[int32]string{
	101: "blade_", 102: "sword_", 103: "thorn_", 104: "sblade_", 105: "ssword_",
	106: "sthorn_", 107: "lstuff_", 108: "cosh_", 122: "claymore_",
}

// gmArmorSlotEquipTypes 防具部位名（GM 面板口径）→ EquipType（Java
// LoginFlow.ARMOR_SLOT 同表；wrist 归 bracer 专用但保留同域）。
var gmArmorSlotEquipTypes = map[string]string{
	"helmet": "Hat", "amulet": "Necklace", "cloth": "Cloth", "pants": "Pants",
	"leg": "Leg", "shoes": "Shoes", "wrist": "Wrist", "ear": "Earring", "ring": "Fingerring",
}

// gmTemplateFamilyBase 模板家族基座后缀（面板规格：51 级金色生活技能装
// 60501 家族优先；Java TEMPLATE_FAMILY_BASE 同值）。
const gmTemplateFamilyBase = "_60501"

// gmFixedAffix 固定词条规格：(属性名, 数值) 对。
type gmFixedAffix struct {
	prop  string
	value int
}

// gmArmorRedAffixes 防具固定红词条（面板规格）：外功防御+46/+47、体魄+29、
// 内息+39、臂力+35（Java ARMOR_RED_AFFIXES 同表）。
var gmArmorRedAffixes = []gmFixedAffix{
	{"PhyDefAdd", 46}, {"PhyDefAdd", 47},
	{"StaAdd", 29}, {"MaxMPAdd", 39}, {"StrAdd", 35},
}

// gmBracerStrAffixes 护腕固定阳刚臂力词条数值（面板规格：+33/+35/+35；
// Java BRACER_STR_AFFIXES 同值）。
var gmBracerStrAffixes = []int{33, 35, 35}

// applyGMGenerateEquip 模板生成三动作总入口（processGMCommand 分派）。
func applyGMGenerateEquip(h *itemSession, action string, weaponType int32, crafter, armorSlot string) (string, error) {
	switch action {
	case "gm_generate_weapon":
		return applyGMGenerateWeapon(h, weaponType, crafter)
	case "gm_generate_armor":
		return applyGMGenerateArmor(h, armorSlot)
	case "gm_generate_bracer":
		return applyGMGenerateBracer(h)
	}
	return "", fmt.Errorf("unsupported generate action %q", action)
}

// applyGMGenerateWeapon gm_generate_weapon：按武器类型挑模板 + 官方洗练
// 池词条（厨师 6 属性；铁匠 3 属性 + 3 招式）并立即穿戴。
func applyGMGenerateWeapon(h *itemSession, weaponType int32, crafter string) (string, error) {
	prefix := gmWeaponPrefix[weaponType]
	if prefix == "" {
		return "", fmt.Errorf("weaponType %d 无模型前缀", weaponType)
	}
	template := pickGMTemplate(h, "Weapon", prefix)
	if template == nil {
		return "", fmt.Errorf("没有武器类型 %d 的可穿戴模板", weaponType)
	}
	// 面板规格：词条取 senior 官方洗练池（厨师 6 条混合池；铁匠 3 条混合
	// 池 + 3 条招式池）；池缺失回退全池随机并在回执注明。混合池不滤词条
	// 归属（Java drawPacks 语义：池内重复=权重）。
	cook := crafter == "cook"
	catalog := currentWeaponRecastCatalog()
	packs := gmDrawPacks(gmTemplateRandPool(catalog, template.ConfigID), gmAffixCount(cook))
	note := ""
	if len(packs) == 0 {
		packs = gmRandomPropPacks(catalog, gmAffixCount(cook))
		note = "官方洗练池缺失，词条回退全池随机"
	} else if !cook {
		skills := gmDrawPacks(gmTemplateWeaponPool(catalog, template.ConfigID), 3)
		packs = append(packs, skills...)
	}
	return wearGMTemplate(h, *template, packs, "武器", note)
}

// applyGMGenerateArmor gm_generate_armor：按防具部位挑模板 + 固定 5 条红
// 词条（可穿戴部位直接穿，饰品入背包）。
func applyGMGenerateArmor(h *itemSession, armorSlot string) (string, error) {
	equipType := gmArmorSlotEquipTypes[armorSlot]
	if equipType == "" {
		return "", fmt.Errorf("armorSlot %q 无对应部位", armorSlot)
	}
	template := pickGMTemplate(h, equipType, "")
	if template == nil {
		return "", fmt.Errorf("没有部位 %s 的可穿戴模板", armorSlot)
	}
	// 固定词条按 (属性名,数值) 从 proppack 定位最小包号，未命中跳过计数。
	packs := make([]int, 0, len(gmArmorRedAffixes))
	missing := 0
	for _, affix := range gmArmorRedAffixes {
		if pack := fixedAffixPack(affix.prop, float64(affix.value)); pack > 0 {
			packs = append(packs, pack)
		} else {
			missing++
		}
	}
	note := ""
	if missing > 0 {
		note = fmt.Sprintf("红词条缺 %d 条（proppack 未命中）", missing)
	}
	return wearGMTemplate(h, *template, packs, "防具", note)
}

// applyGMGenerateBracer gm_generate_bracer：护腕模板 + 固定 3 条臂力
// （+33/+35/+35）+ 3 条招式词条（按身上武器类型取池）。
func applyGMGenerateBracer(h *itemSession) (string, error) {
	template := pickGMTemplate(h, "Wrist", "")
	if template == nil {
		return "", fmt.Errorf("没有护腕（Wrist）的可穿戴模板")
	}
	packs := make([]int, 0, 6)
	missing := 0
	for _, value := range gmBracerStrAffixes {
		if pack := fixedAffixPack("StrAdd", float64(value)); pack > 0 {
			packs = append(packs, pack)
		} else {
			missing++
		}
	}
	note := ""
	if missing > 0 {
		note = fmt.Sprintf("臂力词条缺 %d 条（proppack 未命中）", missing)
	}
	// 招式词条：按身上已穿武器的官方池随机（洗练「招式开光」同语义）；
	// 未穿武器则省略。
	weaponConfig := gmEquippedWeaponConfig(h)
	if weaponConfig == "" {
		note = joinGMNote(note, "未穿武器，招式词条省略")
	} else {
		catalog := currentWeaponRecastCatalog()
		if skills := gmDrawPacks(gmTemplateWeaponPool(catalog, weaponConfig), 3); len(skills) > 0 {
			packs = append(packs, skills...)
		} else {
			note = joinGMNote(note, "官方招式池缺失，招式词条省略")
		}
	}
	return wearGMTemplate(h, *template, packs, "护腕", note)
}

// gmAffixCount 厨师/铁匠的混合池词条数（6/3）。
func gmAffixCount(cook bool) int {
	if cook {
		return 6
	}
	return 3
}

// joinGMNote 拼接回执附注段（空段跳过，分号连接）。
func joinGMNote(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "；")
}

// gmEquippedWeaponConfig 身上已穿武器 ConfigID（招式词条按其武器类型取
// 池）；未穿返回空串。
func gmEquippedWeaponConfig(h *itemSession) string {
	for _, worn := range h.player.sessionEquips() {
		switch strings.ToLower(worn.equipType) {
		case "weapon", "shotweapon", "innerweapon":
			return worn.configID
		}
	}
	return ""
}

// pickGMTemplate 模板挑选：EquipType 匹配（武器再加模型前缀）+ 性别过滤
// + 60501 家族优先，回退品质窗口随机（Java pickTemplate 同语义；候选遍
// 历 gmItems 目录 = equipment.ini 全量）。
func pickGMTemplate(h *itemSession, equipType, modelPrefix string) *gmItemRow {
	sex := h.player.visualSexOf()
	candidates := make([]*gmItemRow, 0, 16)
	for index := range gmItems.rows {
		row := &gmItems.rows[index]
		if !strings.EqualFold(row.EquipType, equipType) {
			continue
		}
		if row.NeedSex != 2 && row.NeedSex != int(sex) {
			continue // NeedSex 0男/1女/2通用
		}
		if modelPrefix != "" &&
			!strings.HasPrefix(equipmentModels.resolve(row.ConfigID, sex), modelPrefix) {
			continue
		}
		candidates = append(candidates, row)
	}
	if len(candidates) == 0 {
		return nil
	}
	family := make([]*gmItemRow, 0, len(candidates))
	for _, row := range candidates {
		if strings.HasSuffix(row.ConfigID, gmTemplateFamilyBase) {
			family = append(family, row)
		}
	}
	if len(family) > 0 {
		return family[recastRandomSource.Intn(len(family))]
	}
	sort.SliceStable(candidates, func(a, b int) bool {
		return candidates[a].ColorLevel > candidates[b].ColorLevel
	})
	window := len(candidates)
	if window > 8 {
		window = 8 // 头部品质窗口内随机
	}
	return candidates[recastRandomSource.Intn(window)]
}

// ---- 洗练池模板方言（Java RecastCatalog.randPool/weaponPool 同语义） ----

// gmTemplateFamilyPrefix 家族前缀：去尾部 `_数字` 段
// （cloth_liangu_dragon_kf_60701 → cloth_liangu_dragon_kf）。
func gmTemplateFamilyPrefix(configID string) string {
	us := strings.LastIndexByte(configID, '_')
	if us <= 0 || us == len(configID)-1 {
		return configID
	}
	for i := us + 1; i < len(configID); i++ {
		if c := configID[i]; c < '0' || c > '9' {
			return configID
		}
	}
	return configID[:us]
}

// gmTemplateRandPool 随机洗练池：精确节 → 家族前缀节并集 → 整池并集兜底
//（Java randPool(configId, weaponItem=true) 在武器模板场景的可达路径；
// 并集按节名排序保证确定性，同 Java sectionPool 的 sorted 合并）。
func gmTemplateRandPool(catalog *weaponRecastCatalog, configID string) []int {
	return gmSectionPoolFallback(catalog, configID)
}

// gmTemplateWeaponPool 招式开光池：恒走 weapon 文件（精确 → 前缀 → 整池；
// Go 洗练目录只装 weapon 两表，rand/weapon 池同源——防具/护腕固定词条
// 不走随机池，senior_prop_rand 的交叉查表路径在本域不可达）。
func gmTemplateWeaponPool(catalog *weaponRecastCatalog, configID string) []int {
	return gmSectionPoolFallback(catalog, configID)
}

// gmSectionPoolFallback 精确节 → 家族并集 → 整池。
func gmSectionPoolFallback(catalog *weaponRecastCatalog, configID string) []int {
	if catalog == nil {
		return nil
	}
	if exact := catalog.randPools[configID]; len(exact) > 0 {
		return exact
	}
	prefix := gmTemplateFamilyPrefix(configID)
	if prefix != configID {
		sections := make([]string, 0, 8)
		for section, pool := range catalog.randPools {
			if len(pool) > 0 && strings.HasPrefix(section, prefix+"_") {
				sections = append(sections, section)
			}
		}
		if len(sections) > 0 {
			sort.Strings(sections)
			merged := make([]int, 0, len(sections)*8)
			for _, section := range sections {
				merged = append(merged, catalog.randPools[section]...)
			}
			return merged
		}
	}
	// 整池兜底（节名排序合并；Go randPools 无表序记录，排序即确定性）。
	sections := make([]string, 0, len(catalog.randPools))
	for section, pool := range catalog.randPools {
		if len(pool) > 0 {
			sections = append(sections, section)
		}
	}
	sort.Strings(sections)
	merged := make([]int, 0, len(sections)*8)
	for _, section := range sections {
		merged = append(merged, catalog.randPools[section]...)
	}
	return merged
}

// gmDrawPacks 官方洗练池放回均匀抽 count 条包号（池内重复即权重；Java
// drawPacks 同语义）；池空返回 nil。
func gmDrawPacks(pool []int, count int) []int {
	if len(pool) == 0 || count <= 0 {
		return nil
	}
	drawn := make([]int, 0, count)
	for i := 0; i < count; i++ {
		drawn = append(drawn, pool[recastRandomSource.Intn(len(pool))])
	}
	return drawn
}

// gmRandomPropPacks 回退词条：全 proppack 节号集均匀抽 count 条（Java
// randomPropBuilt 同语义，返回包号列表走统一 kind 归属链）。
func gmRandomPropPacks(catalog *weaponRecastCatalog, count int) []int {
	if catalog == nil || len(catalog.propIDs) == 0 || count <= 0 {
		return nil
	}
	packIDs := make([]int, 0, len(catalog.propIDs))
	for id := range catalog.propIDs {
		packIDs = append(packIDs, id)
	}
	sort.Ints(packIDs)
	drawn := make([]int, 0, count)
	for i := 0; i < count; i++ {
		drawn = append(drawn, packIDs[recastRandomSource.Intn(len(packIDs))])
	}
	return drawn
}

// ---- proppack (属性名,数值) → 最小包号 索引（Java EquipAffixCatalog +
// fixedAffixPack 同语义；多包命中取最小号——基础包低号，条件包不竞争） ----

type proppackFixedIndex struct {
	byPropValue map[string]map[float64]int // propName → value → 最小包号
}

var (
	proppackFixedOnce sync.Once
	proppackFixed     *proppackFixedIndex
)

// currentProppackFixedIndex 懒加载全局索引（currentWeaponRecastCatalog
// 同模式）。
func currentProppackFixedIndex() *proppackFixedIndex {
	proppackFixedOnce.Do(func() {
		if proppackFixed != nil {
			return // 测试注入优先
		}
		proppackFixed = loadProppackFixedIndex(
			filepath.Join(defaultModernShareRoot, "modifypack", "proppack.ini"))
	})
	return proppackFixed
}

// setProppackFixedIndexForTest 单测注入合成索引（生产代码不得调用）。
func setProppackFixedIndexForTest(index *proppackFixedIndex) {
	proppackFixedOnce.Do(func() {})
	proppackFixed = index
}

// loadProppackFixedIndex 全量扫 proppack.ini：r=<序>,<PropName>,<值>,<flag>
// [,<条件ID>] → (propName,value) → 最小包号。
func loadProppackFixedIndex(path string) *proppackFixedIndex {
	index := &proppackFixedIndex{byPropValue: map[string]map[float64]int{}}
	file, err := openResourceReader(path)
	if err != nil {
		log.Printf("gm generate: proppack fixed index unavailable: %v", err)
		return index
	}
	defer file.Close()
	parseINISections(file, func(section, key, value string) {
		if !strings.EqualFold(key, "r") {
			return
		}
		pack, ok := parseUintSection(section)
		if !ok || pack <= 0 {
			return
		}
		parts := strings.Split(value, ",")
		if len(parts) < 3 {
			return
		}
		propName := strings.TrimSpace(parts[1])
		if propName == "" {
			return
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			return
		}
		values, seen := index.byPropValue[propName]
		if !seen {
			values = map[float64]int{}
			index.byPropValue[propName] = values
		}
		if best, ok := values[parsed]; !ok || int(pack) < best {
			values[parsed] = int(pack)
		}
	})
	return index
}

// fixedAffixPack 按 (属性名,数值) 在 proppack 定位词缀包：多包命中取最小
// 包号，未命中返回 0。
func fixedAffixPack(prop string, value float64) int {
	index := currentProppackFixedIndex()
	if index == nil {
		return 0
	}
	if values, ok := index.byPropValue[prop]; ok {
		if pack, ok := values[value]; ok {
			return pack
		}
	}
	return 0
}

// ---- 生成落地（grant_item wear 分支 + recastConfirm 词条收尾的合链） ----

// wearGMTemplate 生成模板落地：可穿戴直接穿（异款占用先让位防 uid 串号
// ——grant_item wear 分支同款，比 Java 直接 upsert 覆盖多一层保护）；
// 槽位未证类型兜底入背包（词条不随行）。
func wearGMTemplate(h *itemSession, template gmItemRow, packs []int, kind, note string) (string, error) {
	slot := equipSlotFor(template.EquipType, h.player.sessionEquips())
	if slot <= 0 {
		return gmGrantTemplateToBag(h, template, kind)
	}
	if old, _, taken := h.player.equipAtSlot(slot); taken && old.configID != template.ConfigID {
		// 目标槽被异款占用：先让位旧装备入包（unequipFromDoll 一体完成
		// DB uid 携带 + 内存 + 帧 + 随机属性 record 重推）。
		unequipFromDoll(h, slot, int32(viewEquipBag), 0)
		if _, _, still := h.player.equipAtSlot(slot); still {
			return "", fmt.Errorf("槽位 %d 让位失败（背包满或落库失败），已占用 %s", slot, old.configID)
		}
	}
	worn, wearErr := h.store.upsertEquipSlot(h.ctx, h.roleID, slot,
		template.ConfigID, int32(template.ItemType), template.EquipType)
	if wearErr != nil {
		return "", wearErr
	}
	// 词条落库（kind 按 proppack/skillpack 归属）+ 内存行/帧同步。
	affixes := "[]"
	if len(packs) > 0 {
		affixes = recastAffixesJSON(currentWeaponRecastCatalog(), packs)
		if err := h.store.updateEquipAffixes(h.ctx, h.roleID, slot, affixes); err != nil {
			return "", fmt.Errorf("词条落库失败: %w", err)
		}
	}
	worn.propBuilt = propBuiltOfAffixes(affixes)
	h.player.replaceEquipSlot(worn)
	if framesErr := pushWearFrames(h.link, h.roleID, h.player.visualSexOf(),
		effectiveShowEquipType(h.player), worn, h.player.sessionEquips()); framesErr != nil {
		return "", framesErr
	}
	// 模板包 record（镇派兵器词条官方通道 394/395）+ 宝物/娃娃随机属性
	// 掷值存档整推（helper 内部分拣；60501 家族无模板包则内部跳过）。
	h.pushRandomPropsForEquip(template.ConfigID, worn.uid, slot)
	if note != "" {
		note = "，" + note
	}
	return fmt.Sprintf("已生成并穿戴%s %s（词条 %d 条%s）外观与加成已同步",
		kind, template.ConfigID, len(packs), note), nil
}

// gmGrantTemplateToBag 槽位未证类型兜底：生成后入背包（applyGrantItem
// 背包分支同款，词条不随行）。
func gmGrantTemplateToBag(h *itemSession, template gmItemRow, kind string) (string, error) {
	entry := grantableItem{
		configID:  template.ConfigID,
		itemType:  int32(template.ItemType),
		viewID:    int32(template.ViewID),
		name:      template.Name,
		equipType: template.EquipType,
		script:    template.Script,
	}
	if entry.viewID <= 0 {
		entry.viewID = 1
	}
	inserted, landed := h.grantBagItemRow(entry, 1)
	if !landed || inserted.slot <= 0 {
		return "", fmt.Errorf("背包已满（1..90）")
	}
	h.pushRandomPropsForBag(template.ConfigID, inserted.uid, inserted.viewID, inserted.slot)
	return fmt.Sprintf("已生成%s %s 入背包第 %d 格（饰品穿戴槽位未证，词条不随行）",
		kind, template.ConfigID, inserted.slot), nil
}
