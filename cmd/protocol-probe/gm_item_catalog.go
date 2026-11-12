package main

// GM「所有道具与换装」目录（Java gm-web GmCatalog.items 的 Go 移植）：
// share/item/equipment.ini（装备，EquipType 随行）+ tool_item.ini（道具）
// 合并检索。前端约定（console.html 该节说明）：
//   - 中文名以 stringname.idres 为准（与游戏内显示同源，可纠正表内抄错/
//     占位名）；idres 无键回落表内 Name，仍无（或纯数字占位）则空，
//     页面回落显示 ConfigID；
//   - 注释节（//包裹、//修炼药剂…）与身份装内件（ViewID≠1..4，走
//     「时装/身份装发放」的套装链）不进本列表；
//   - ViewID 把道具分入普通(1)/装备(2)/材料(3)/任务(4)背包页。
//
// 两表均为 GBK 且 10 万+ 节，启动装载一次（fashionSuits/equipmentModels
// 同款包级缓存；测试经 injectStubGMItemCatalog 换表）。目录缺失时为空
// 目录：搜索返回空数组，面板提示"没有找到匹配项"。

import (
	"bufio"
	"bytes"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// gmItemSearchLimit 单次搜索返回上限（Java GmCatalog.ITEM_SEARCH_LIMIT；
// 全量 10 万+ 行，无上限会把下拉框与 JSON 都撑爆）。
const gmItemSearchLimit = 300

// gmItemRow 是 /api/items 的行形态（console-features.js searchItems 消费
// 字段：configId/name/viewId/wearable/equipType/script/maxAmount/
// colorLevel/needSex）。
type gmItemRow struct {
	ConfigID   string `json:"configId"`
	Name       string `json:"name"`
	ViewID     int    `json:"viewId"`
	Wearable   bool   `json:"wearable"`
	EquipType  string `json:"equipType"`
	Script     string `json:"script"`
	MaxAmount  int    `json:"maxAmount"`
	ColorLevel int    `json:"colorLevel"`
	NeedSex    int    `json:"needSex"`
	// ItemType 仅供目录内过滤（/api/giftboxes 取 2102/2103 开箱类），
	// 不属于面板检索行字段，不参与 JSON 输出。
	ItemType int `json:"-"`
	// ArtPack 同为目录内过滤数据（/api/mounts 取坐骑外观 ID），
	// 不参与 JSON 输出。
	ArtPack string `json:"-"`
}

type gmItemCatalog struct {
	rows []gmItemRow
}

// gmItemSection 是单节聚合中间态：消费过的键记位（同节重复键首见优先，
// Java IniFile 键列表 + first() 同口径）；row 先持 INI 原始 Name，装载
// 尾段再过 idres 名链。
type gmItemSection struct {
	row  gmItemRow
	seen uint16
}

const (
	gmItemKeyName uint16 = 1 << iota
	gmItemKeyScript
	gmItemKeyEquipType
	gmItemKeyViewID
	gmItemKeyMaxAmount
	gmItemKeyColorLevel
	gmItemKeyNeedSex
	gmItemKeyItemType
	gmItemKeyArtPack
)

// gmItemIniSource 单张 ini 表（equipment=true / tool_item=false）。
type gmItemIniSource struct {
	reader   io.Reader
	wearable bool
}

// loadGMItemCatalog 从现代资源树装载（item/equipment.ini + tool_item.ini）。
// 中文名查 stringname.idres（sceneNameTables 复用同一张小写键表，选角
// 场景名与道具名同源同缓存）。文件缺失/读失败按空表降级，不报错。
func loadGMItemCatalog(shareRoot string) *gmItemCatalog {
	itemRoot := filepath.Join(shareRoot, "item")
	sources := make([]gmItemIniSource, 0, 2)
	for _, entry := range []struct {
		name     string
		wearable bool
	}{
		{"equipment.ini", true},
		{"tool_item.ini", false},
	} {
		data, err := readResourceBytes(filepath.Join(itemRoot, entry.name))
		if err != nil {
			continue
		}
		sources = append(sources, gmItemIniSource{reader: bytes.NewReader(data), wearable: entry.wearable})
	}
	return loadGMItemCatalogFrom(sceneNameTablesOf().nameByKey, sources...)
}

// loadGMItemCatalogFrom 纯函数装载：GBK 解码 → 逐节聚合 → ViewID 1..4
// 过滤 → idres/INI 名链。装备表行序在前、道具表在后（Java loadItemIni
// 顺序）；同文件重复节并回首次出现位置。
func loadGMItemCatalogFrom(names map[string]string, sources ...gmItemIniSource) *gmItemCatalog {
	catalog := &gmItemCatalog{}
	for _, source := range sources {
		data, err := io.ReadAll(source.reader)
		if err != nil {
			continue
		}
		if decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data); err == nil {
			data = decoded
		}
		var (
			order   []string
			byID    = map[string]*gmItemSection{}
			current *gmItemSection
		)
		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
			if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				section := strings.TrimSpace(line[1 : len(line)-1])
				// 注释节（[//拜师入门专用] 等）不进目录。
				if strings.HasPrefix(section, "//") || strings.HasPrefix(section, ";") {
					current = nil
					continue
				}
				if existing, ok := byID[section]; ok {
					current = existing
				} else {
					current = &gmItemSection{row: gmItemRow{
						ConfigID: section, ViewID: 1, NeedSex: 2, Wearable: source.wearable,
					}}
					byID[section] = current
					order = append(order, section)
				}
				continue
			}
			if current == nil {
				continue
			}
			key, value, found := strings.Cut(line, "=")
			if !found {
				continue
			}
			value = strings.TrimSpace(value)
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "name":
				if current.seen&gmItemKeyName == 0 {
					current.seen |= gmItemKeyName
					current.row.Name = value
				}
			case "script":
				if current.seen&gmItemKeyScript == 0 {
					current.seen |= gmItemKeyScript
					current.row.Script = value
				}
			case "equiptype":
				if current.seen&gmItemKeyEquipType == 0 {
					current.seen |= gmItemKeyEquipType
					if source.wearable {
						current.row.EquipType = value
					}
				}
			case "viewid":
				if current.seen&gmItemKeyViewID == 0 {
					current.seen |= gmItemKeyViewID
					setGMItemInt(value, &current.row.ViewID)
				}
			case "maxamount":
				if current.seen&gmItemKeyMaxAmount == 0 {
					current.seen |= gmItemKeyMaxAmount
					setGMItemInt(value, &current.row.MaxAmount)
				}
			case "colorlevel":
				if current.seen&gmItemKeyColorLevel == 0 {
					current.seen |= gmItemKeyColorLevel
					setGMItemInt(value, &current.row.ColorLevel)
				}
			case "needsex":
				if current.seen&gmItemKeyNeedSex == 0 {
					current.seen |= gmItemKeyNeedSex
					setGMItemInt(value, &current.row.NeedSex)
				}
			case "itemtype":
				if current.seen&gmItemKeyItemType == 0 {
					current.seen |= gmItemKeyItemType
					setGMItemInt(value, &current.row.ItemType)
				}
			case "artpack":
				if current.seen&gmItemKeyArtPack == 0 {
					current.seen |= gmItemKeyArtPack
					current.row.ArtPack = value
				}
			}
		}
		for _, configID := range order {
			row := byID[configID].row
			// 身份装内件（ViewID 11 等）走「时装/身份装发放」套装链，
			// 发进背包只会成为隐形行，整组跳过（Java 同口径）。
			if row.ViewID < 1 || row.ViewID > 4 {
				continue
			}
			row.Name = gmItemDisplayName(row.ConfigID, row.Name, names)
			catalog.rows = append(catalog.rows, row)
		}
	}
	return catalog
}

// setGMItemInt 宽容解析（非法值保持默认，Java parseInt 同口径）。
func setGMItemInt(value string, target *int) {
	if parsed, err := strconv.Atoi(value); err == nil {
		*target = parsed
	}
}

// gmItemDisplayName 显示名解析链：stringname.idres（小写键）→ 有效 INI
// Name（非空且非纯数字占位）→ 空串（页面回落 ConfigID）。
func gmItemDisplayName(configID, iniName string, names map[string]string) string {
	if name := names[strings.ToLower(configID)]; name != "" {
		return name
	}
	if iniName != "" && !gmNumericPlaceholder(iniName) {
		return iniName
	}
	return ""
}

// gmNumericPlaceholder 纯数字占位名（"10101"/"8"——原表把编号写进了
// Name 列）。
func gmNumericPlaceholder(name string) bool {
	for _, r := range name {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// search 检索：query 对 Name/ConfigID 不区分大小写包含；view>0 时按
// ViewID 过滤；上限 gmItemSearchLimit（行序=文件序）。
func (c *gmItemCatalog) search(query string, view int) []gmItemRow {
	if c == nil {
		return []gmItemRow{}
	}
	needle := strings.ToLower(strings.TrimSpace(query))
	out := make([]gmItemRow, 0, 32)
	for _, row := range c.rows {
		if needle != "" &&
			!strings.Contains(strings.ToLower(row.Name), needle) &&
			!strings.Contains(strings.ToLower(row.ConfigID), needle) {
			continue
		}
		if view > 0 && row.ViewID != view {
			continue
		}
		out = append(out, row)
		if len(out) >= gmItemSearchLimit {
			break
		}
	}
	return out
}

// searchPage 分页检索（与 search 同一匹配口径）：返回 [offset, offset+limit)
// 的命中行与命中总数。目录 10 万+ 行，旧「一次截 300 条」让前端把截断值
// 当总量、且永远翻不到 300 名以外——分页信封把真实 total 交回前端。
// 全表线性扫一遍（含 total 计数），毫秒级，无需索引。
func (c *gmItemCatalog) searchPage(query string, view, offset, limit int) ([]gmItemRow, int) {
	if c == nil {
		return []gmItemRow{}, 0
	}
	needle := strings.ToLower(strings.TrimSpace(query))
	out := make([]gmItemRow, 0, min(limit, 32))
	total := 0
	for _, row := range c.rows {
		if needle != "" &&
			!strings.Contains(strings.ToLower(row.Name), needle) &&
			!strings.Contains(strings.ToLower(row.ConfigID), needle) {
			continue
		}
		if view > 0 && row.ViewID != view {
			continue
		}
		if total >= offset && len(out) < limit {
			out = append(out, row)
		}
		total++
	}
	return out, total
}

// gmItems 全局目录（fashionSuits 同款包级缓存）。
var gmItems = loadGMItemCatalog(defaultModernShareRoot)
