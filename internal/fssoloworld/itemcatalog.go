package fssoloworld

// itemcatalog.go 物品目录解析（进程级缓存）：ini/Item 构建产物优先，产物
// 缺席回退设计态 share/item。
//
// 解析序（显式环境变量一律压过自动探测）：
//
//	NINEYIN_ITEM_XML  构建产物目录（含 path_config.xml）显式覆盖
//	NINEYIN_ITEM_DIR  设计态 ini 目录显式覆盖（旧口径，原样保留）
//	数据根 ini/Item   构建产物（tools/datagen item；候选根同技能产物
//	                  skillDataRoots：npcfeed.ResolveRoot → 仓库 testdata）
//	数据根 share/item 设计态 ini 回退（只有目录面，无模板行/静态包）
//
// 缺档负面缓存（进程级，不重试——物品数据静态）；缓存键 = 数据根 + 两个
// 环境变量。

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/itemdata"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

var (
	itemCatalogMu     sync.Mutex
	itemCatalogTried  string
	itemCatalogCached *itemdata.Catalog
)

// itemCatalogOf 物品目录（进程级缓存，shopCatalogOf 同款模式）。
func (s *Server) itemCatalogOf() *itemdata.Catalog {
	itemXML, itemDir := strings.TrimSpace(os.Getenv("NINEYIN_ITEM_XML")), strings.TrimSpace(os.Getenv("NINEYIN_ITEM_DIR"))
	root := npcfeed.ResolveRoot()
	key := root + "\x00" + itemXML + "\x00" + itemDir
	itemCatalogMu.Lock()
	defer itemCatalogMu.Unlock()
	if key == itemCatalogTried {
		return itemCatalogCached
	}
	itemCatalogTried, itemCatalogCached = key, nil
	if itemXML != "" {
		catalog, err := itemdata.LoadXML(filepath.Clean(itemXML))
		return s.adoptItemCatalog(catalog, err, "NINEYIN_ITEM_XML")
	}
	if itemDir != "" {
		catalog, err := itemdata.LoadDir(filepath.Clean(itemDir))
		return s.adoptItemCatalog(catalog, err, "NINEYIN_ITEM_DIR")
	}
	for _, candidate := range skillDataRoots() {
		catalog, err := itemdata.LoadXML(filepath.Join(candidate, "ini", "Item"))
		if err == nil {
			return s.adoptItemCatalog(catalog, nil, candidate+string(filepath.Separator)+"ini"+string(filepath.Separator)+"Item")
		}
	}
	if root == "" {
		return nil
	}
	catalog, err := itemdata.LoadDir(filepath.Join(root, "share", "item"))
	return s.adoptItemCatalog(catalog, err, "share/item")
}

// resetItemCatalog 清目录缓存（测试缝：换环境变量/夹具后强制重解析）。
func resetItemCatalog() {
	itemCatalogMu.Lock()
	itemCatalogTried, itemCatalogCached = "", nil
	itemCatalogMu.Unlock()
}

// adoptItemCatalog 装载结果入缓存并记一次来源日志。
func (s *Server) adoptItemCatalog(catalog *itemdata.Catalog, err error, source string) *itemdata.Catalog {
	itemCatalogCached = catalog
	switch {
	case catalog == nil:
		s.trace("item catalog unavailable（%s）: %v", source, err)
	case catalog.FromXML():
		s.trace("item catalog loaded: %s（%d 行，黑金格式构建产物；缺席静态包 %v）",
			catalog.Source(), catalog.Count(), catalog.MissingPacks())
	default:
		s.trace("item catalog loaded: %s（%d 行，设计态 ini 回退——go run ./tools/datagen item 构建产物可启用模板行/静态包）",
			catalog.Source(), catalog.Count())
	}
	return catalog
}

// bagRowIndex 背包分类页 slot（1 起）上的行下标（无行 -1）。
func bagRowIndex(st *rolestate.RoleState, page, slot int) int {
	for i := range st.BagItems {
		if st.BagItems[i].ViewID == page && st.BagItems[i].Slot == slot {
			return i
		}
	}
	return -1
}
