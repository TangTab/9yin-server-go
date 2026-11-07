package main

// 风物志皮肤目录（Java SkinCatalog 的 Go 移植）：客户端
// resources/modern/share/rule/card.ini，节号 = 皮肤号（1..1239 量级：
// Weapon 348 / Cloth 247 / Hat 133 / Waist 101 / Cloak 57 / Mount 31 …）。
// ModelData 形如 "Part,model;Part,model" 多段（套装皮肤多部件，单件
// 如 Weapon,blade_card_001）。configid 均为 ASCII，原始字节读（同
// equipment.ini 约定）；目录缺失时空目录优雅降级。

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// skinPartModel 是 ModelData 的一段（部件名 + 模型名）。
type skinPartModel struct {
	part  string
	model string
}

// skin 是 card.ini 一个数字节的皮肤。
type skin struct {
	id       int
	configID string
	male     []skinPartModel
	female   []skinPartModel
	itemType int32
	mainType int32
	subType  int32
	level    int32
}

// models 按性别取部件模型表（sex==1 且有女模用女模，否则男模）。
func (s skin) models(sex byte) []skinPartModel {
	if sex == 1 && len(s.female) > 0 {
		return s.female
	}
	return s.male
}

// parts 该性别下的部件名集合（小写）。
func (s skin) parts(sex byte) map[string]bool {
	parts := make(map[string]bool)
	for _, entry := range s.models(sex) {
		parts[strings.ToLower(entry.part)] = true
	}
	return parts
}

// hairOnly 是否纯发型皮肤（有部件且全部为 Hat——GM equip_hair 专用口径）。
func (s skin) hairOnly() bool {
	if len(s.male) == 0 && len(s.female) == 0 {
		return false
	}
	models := s.male
	if len(models) == 0 {
		models = s.female
	}
	for _, entry := range models {
		if !strings.EqualFold(entry.part, "Hat") {
			return false
		}
	}
	return true
}

// skinCatalog 是升序皮肤表 + configId 反查。
type skinCatalog struct {
	byID  map[int]skin
	ids   []int // 升序（CardRec 位图行上界）
	byCfg map[string]int
}

// loadSkinCatalog 解析 card.ini（root = share/rule 目录）。非数字节防御
// 跳过（card.ini 有少量注释节）。
func loadSkinCatalog(ruleRoot string) *skinCatalog {
	catalog := &skinCatalog{byID: map[int]skin{}, byCfg: map[string]int{}}
	parseINI(filepath.Join(ruleRoot, "card.ini"), func(section, key, value string) {
		id, err := strconv.Atoi(section)
		if err != nil {
			return
		}
		entry := catalog.byID[id]
		entry.id = id
		switch strings.ToLower(key) {
		case "configid":
			entry.configID = value
		case "malemodeldata":
			entry.male = parseSkinModels(value)
		case "femalemodeldata":
			entry.female = parseSkinModels(value)
		case "itemtype":
			entry.itemType = atoi32(value)
		case "maintype":
			entry.mainType = atoi32(value)
		case "subtype":
			entry.subType = atoi32(value)
		case "level":
			entry.level = atoi32(value)
		}
		catalog.byID[id] = entry
	})
	for id, entry := range catalog.byID {
		catalog.ids = append(catalog.ids, id)
		if entry.configID != "" {
			catalog.byCfg[strings.ToLower(entry.configID)] = id
		}
	}
	sort.Ints(catalog.ids)
	return catalog
}

// parseSkinModels 解析 "Part,model;Part,model" 多段；缺件或缺模型段跳过。
func parseSkinModels(data string) []skinPartModel {
	if strings.TrimSpace(data) == "" {
		return nil
	}
	var models []skinPartModel
	for _, segment := range strings.Split(data, ";") {
		part, model, ok := strings.Cut(segment, ",")
		if !ok || part == "" || model == "" {
			continue
		}
		models = append(models, skinPartModel{part: strings.TrimSpace(part), model: strings.TrimSpace(model)})
	}
	return models
}

func atoi32(value string) int32 {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return int32(parsed)
}

// defaultSkinRuleRoot card.ini 所在目录（NINEYIN_CARD_ROOT 可覆盖）。
func defaultSkinRuleRoot() string {
	if custom := strings.TrimSpace(os.Getenv("NINEYIN_CARD_ROOT")); custom != "" {
		return custom
	}
	return runtimeProjectPath("resources", "modern", "share", "rule")
}

// skinCards 全局目录（equipmentModels 同款包级缓存；测试经
// injectStubSkinCatalog 换表）。
var skinCards = loadSkinCatalog(defaultSkinRuleRoot())

func (c *skinCatalog) isEmpty() bool { return len(c.ids) == 0 }

// lookup 按皮肤号取皮肤。
func (c *skinCatalog) lookup(id int) (skin, bool) {
	entry, ok := c.byID[id]
	return entry, ok
}

// allIDs 升序全量（grant_all_cards / 位图行上界）。
func (c *skinCatalog) allIDs() []int { return c.ids }

// idOfConfig 按 configId 反查皮肤号（GM equip_card 接受 configId 形态时用）。
func (c *skinCatalog) idOfConfig(configID string) (int, bool) {
	id, ok := c.byCfg[strings.ToLower(strings.TrimSpace(configID))]
	return id, ok
}

// skinPartSummary 日志用的部件摘要（"Weapon blade_card_001;…"）。
func skinPartSummary(entry skin, sex byte) string {
	var segments []string
	for _, entry := range entry.models(sex) {
		segments = append(segments, entry.part+" "+entry.model)
	}
	return strings.Join(segments, ";")
}
