package fssoloworld

// gmitemsearch.go GM 物品检索（2433 线 gm_item_catalog.go 的目录面移植，
// 数据源改用本仓 itemdata 设计态装载）：share/item 87 档进程缓存一次，
// 名称/ConfigID 子串匹配（不分大小写）+ 分类页过滤。行协议
// `configID|name|view|equipType|maxAmount`——gm items（echo）与面板
// 「货币与物品」子页的发放搜索共用；数据根缺 share/item 时空目录
//（搜索零行，回执带提示）。

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/itemdata"
)

// gmItemRow 一条检索结果（echo 行协议的取值面）。
type gmItemRow struct {
	ConfigID  string
	Name      string
	ViewID    int
	EquipType string
	MaxAmount int
}

var (
	gmItemSearchMu     sync.Mutex
	gmItemSearchTried  string
	gmItemSearchCached []gmItemRow // 按 ConfigID 排序（输出确定序）
)

// gmItemSearchIndex 检索目录（进程级缓存，skillCatalog 同款数据根锚定）。
func (s *Server) gmItemSearchIndex() []gmItemRow {
	roots := skillDataRoots()
	key := strings.Join(roots, "\x00")
	gmItemSearchMu.Lock()
	defer gmItemSearchMu.Unlock()
	if key == gmItemSearchTried {
		return gmItemSearchCached
	}
	gmItemSearchTried, gmItemSearchCached = key, nil
	for _, root := range roots {
		catalog, err := itemdata.LoadDir(filepath.Join(root, "share", "item"))
		if err != nil {
			continue
		}
		rows := make([]gmItemRow, 0, 1024)
		catalog.Each(func(entry itemdata.Entry) {
			if strings.HasPrefix(entry.ConfigID, "//") || entry.ViewID < 1 || entry.ViewID > 4 {
				return // 注释节与身份装内件不进检索（2433 线同口径）
			}
			rows = append(rows, gmItemRow{
				ConfigID:  entry.ConfigID,
				Name:      entry.Name,
				ViewID:    entry.ViewID,
				EquipType: entry.EquipType,
				MaxAmount: entry.MaxAmount,
			})
		})
		sort.Slice(rows, func(i, j int) bool { return rows[i].ConfigID < rows[j].ConfigID })
		gmItemSearchCached = rows
		s.trace("gm item search catalog: %d 行（%s/share/item）", len(rows), root)
		break
	}
	if gmItemSearchCached == nil {
		s.trace("gm item search catalog unavailable（候选根 %v）：检索空结果", roots)
	}
	return gmItemSearchCached
}

// resetGMItemSearch 清检索缓存（测试缝）。
func resetGMItemSearch() {
	gmItemSearchMu.Lock()
	gmItemSearchTried, gmItemSearchCached = "", nil
	gmItemSearchMu.Unlock()
}

// gmItemSearch 纯函数检索：query 对 Name/ConfigID 子串匹配（不分大小
// 写），viewID 0=全部页；limit 钳 [1, 300]。
func gmItemSearch(rows []gmItemRow, query string, viewID, limit int) []gmItemRow {
	if limit < 1 {
		limit = 50
	}
	if limit > 300 {
		limit = 300
	}
	query = strings.ToLower(strings.TrimSpace(query))
	matched := make([]gmItemRow, 0, limit)
	for _, row := range rows {
		if viewID != 0 && row.ViewID != viewID {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(row.Name), query) &&
			!strings.Contains(strings.ToLower(row.ConfigID), query) {
			continue
		}
		matched = append(matched, row)
		if len(matched) >= limit {
			break
		}
	}
	return matched
}
