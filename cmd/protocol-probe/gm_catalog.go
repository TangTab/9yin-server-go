package main

// GM 面板目录端点（console.html「时装/身份装发放」「风物志卡牌」「发型
// 卡牌」区块的下拉数据源；Java gm-web 网关对应物）。条目中文名经
// hub.itemNames（game_items 批查）解析，无 DB 时回退 ConfigID——
// httptest 驱动不依赖运行库。

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type gmCatalogItem struct {
	ConfigID string `json:"configId"`
	Name     string `json:"name"`
}

type gmCatalogCategory struct {
	Key   string          `json:"key"`
	Label string          `json:"label"`
	Items []gmCatalogItem `json:"items"`
}

// resolveItemNames 批量解析中文名（钩子缺席/未命中回退 ConfigID）。
func (hub *gmHub) resolveItemNames(configIDs []string) map[string]string {
	names := make(map[string]string, len(configIDs))
	if hub.itemNames == nil {
		return names
	}
	return hub.itemNames(configIDs)
}

func attachNames(items []gmCatalogItem, names map[string]string) {
	for index := range items {
		if name := names[items[index].ConfigID]; name != "" {
			items[index].Name = name
		} else {
			items[index].Name = items[index].ConfigID
		}
	}
}

// fashionCatalogHandler GET /api/fashion-catalog：身份装 7 组（shengfen）
// + 时装 4 段（fashion），条目按段内 r= 行序。
func (hub *gmHub) fashionCatalogHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	collect := func(key, label string, sections ...string) gmCatalogCategory {
		category := gmCatalogCategory{Key: key, Label: label}
		seen := map[string]bool{}
		for _, section := range sections {
			for _, configID := range fashionSuits.sections[section] {
				if !seen[configID] {
					seen[configID] = true
					category.Items = append(category.Items, gmCatalogItem{ConfigID: configID})
				}
			}
		}
		if len(category.Items) == 0 {
			category.Items = []gmCatalogItem{}
		}
		return category
	}
	collectPrefixed := func(key, label, prefix string) gmCatalogCategory {
		var sections []string
		for _, section := range fashionSuits.order {
			if strings.HasPrefix(strings.ToLower(section), prefix) {
				sections = append(sections, section)
			}
		}
		return collect(key, label, sections...)
	}
	categories := []gmCatalogCategory{
		collectPrefixed("identity_school", "身份装·门派", "school_"),
		collectPrefixed("identity_force", "身份装·势力", "force_"),
		collectPrefixed("identity_newschool", "身份装·新门派", "newschool_"),
		collect("identity_shijia", "身份装·世家", "shijia"),
		collect("identity_guild", "身份装·帮会", "guild"),
		collect("identity_life", "身份装·生活", "life"),
		collect("identity_activity", "身份装·活动", "activity"),
		collect("fashion_hat", "时装·帽子", "hat"),
		collect("fashion_coat", "时装·上衣", "coat"),
		collect("fashion_shoes", "时装·鞋子", "shoes"),
		collect("fashion_suit", "时装·套装", "suit"),
	}
	var allIDs []string
	for _, category := range categories {
		for _, item := range category.Items {
			allIDs = append(allIDs, item.ConfigID)
		}
	}
	names := hub.resolveItemNames(allIDs)
	for index := range categories {
		attachNames(categories[index].Items, names)
	}
	writeGMJSON(writer, http.StatusOK, categories)
}

// itemsHandler GET /api/items?q=<中文名或ConfigID>&view=<0..4>：「所有
// 道具与换装」搜索（Java GmCatalog.items 同口径；目录装载见
// gm_item_catalog.go）。
func (hub *gmHub) itemsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	view := 0
	if parsed, err := strconv.Atoi(request.URL.Query().Get("view")); err == nil && parsed >= 0 {
		view = parsed
	}
	writeGMJSON(writer, http.StatusOK, gmItems.search(request.URL.Query().Get("q"), view))
}

// gmCardRow 是 /api/cards 与 /api/hair-cards 的行形态（console-features.js
// 消费字段：id / name / config_id / category）。
type gmCardRow struct {
	ID       int32  `json:"id"`
	Name     string `json:"name"`
	ConfigID string `json:"config_id"`
	Category string `json:"category"`
}

// skinCategoryOf 皮肤分类（部件名首件；套装多部件取第一段——
// Weapon/Cloth/Hat/Waist/Cloak 等与客户端衣橱分组一致）。
func skinCategoryOf(entry skin) string {
	models := entry.male
	if len(models) == 0 {
		models = entry.female
	}
	if len(models) == 0 || models[0].part == "" {
		return "Misc"
	}
	return models[0].part
}

func (hub *gmHub) cardsCore(hairOnly bool) []gmCardRow {
	rows := make([]gmCardRow, 0, len(skinCards.ids))
	var configIDs []string
	for _, id := range skinCards.allIDs() {
		entry, ok := skinCards.lookup(id)
		if !ok {
			continue
		}
		if hairOnly && !entry.hairOnly() {
			continue
		}
		rows = append(rows, gmCardRow{
			ID:       int32(id),
			ConfigID: entry.configID,
			Category: skinCategoryOf(entry),
		})
		configIDs = append(configIDs, entry.configID)
	}
	names := hub.resolveItemNames(configIDs)
	for index := range rows {
		if name := names[rows[index].ConfigID]; name != "" {
			rows[index].Name = name
		} else {
			rows[index].Name = rows[index].ConfigID
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// cardsHandler GET /api/cards：全量皮肤（含卡片道具名/分类）。
func (hub *gmHub) cardsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeGMJSON(writer, http.StatusOK, hub.cardsCore(false))
}

// hairCardsHandler GET /api/hair-cards：纯发型皮肤（equip_hair 专用）。
func (hub *gmHub) hairCardsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeGMJSON(writer, http.StatusOK, hub.cardsCore(true))
}
