package fssoloworld

// npcshop.go 商店/仓库 View 受理（菜单 mark 805000000/807000000）：
//
//	商店  关对话（m8×2）→ 0x15 建 View61（ShopID/PageNum/PageCount/
//	     ShopType 四属性）→ 0x18 逐物品行（ConfigID/Amount/MaxAmount/
//	     SellPrice0..2 按币种入槽）——2433 openShopLocked 帧配方移植，
//	     属性索引按名解析（clientdata 0x09 表）。
//	仓库  关对话 → 0x15 建 View4（容量 18；form_depot 无 Lua 层接收器，
//	     表单由客户端对 SERVER_CREATE_VIEW 原生处理打开——2433 定案）。
//
// 数据链：商店目录 share/trade/shop.ini（npcshop 包，GBK）；ShopID 来自
// NpcConfig 模板行业务列（npcgen 2026-11-23 折叠：商店ID/仓库ID 等七列
// 直通）与 NpcType 结构域回落（npcinteract.go 两级提取）。

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/npcshop"
)

const (
	npcShopViewport  uint16 = 61 // form_shop.lua 从 View61 createview 回调取商店设置
	npcShopCapacity  uint16 = 100
	npcShopPageSize  int32  = 10
	npcShopMaxBuy    int32  = 99
	npcDepotViewport uint16 = 4
	npcDepotCapacity uint16 = 18 // share/Rule/depotbox.ini StartCap（2433 同值）
)

// ---- 商店目录（进程级缓存，npcTaskCatalogOf 同款 root 锚定模式） ----

var (
	npcShopMu        sync.Mutex
	npcShopRootTried string
	npcShopCatalog   *npcshop.Catalog
)

// shopCatalogOf 商店目录（npcTaskCatalogOf 同款 root 锚定缓存）：
// env NINEYIN_SHOP_INI 显式档 > 数据根 share/trade/shop.ini。缺档负面
// 缓存（进程级，不重试——商店数据静态）。
func (s *Server) shopCatalogOf() *npcshop.Catalog {
	npcShopMu.Lock()
	defer npcShopMu.Unlock()
	root := npcfeed.ResolveRoot()
	if root == npcShopRootTried {
		return npcShopCatalog
	}
	npcShopRootTried = root
	npcShopCatalog = nil
	if root == "" {
		return nil
	}
	if v := strings.TrimSpace(os.Getenv("NINEYIN_SHOP_INI")); v != "" {
		if catalog, err := npcshop.Load(filepath.Clean(v)); err == nil {
			npcShopCatalog = catalog
			return catalog
		}
	}
	if catalog, err := npcshop.Load(filepath.Join(root, "share", "trade", "shop.ini")); err == nil {
		npcShopCatalog = catalog
	}
	return npcShopCatalog
}

// openNPCShopView 商店开窗（mark 805000000）。返回是否成功开窗。
func (s *Server) openNPCShopView(pl *Player, shopID string) bool {
	catalog := s.shopCatalogOf()
	if catalog == nil {
		s.trace("npc shop: 商店目录缺席（NINEYIN_SHOP_INI / share/trade/shop.ini），ShopID=%s 不开窗", shopID)
		return false
	}
	section, ok := catalog.Section(shopID)
	if !ok {
		s.trace("npc shop: ShopID=%s 无目录节（%d 节），不开窗", shopID, catalog.Count())
		return false
	}
	frames, err := npcShopViewFrames(shopID, section)
	if err != nil {
		s.trace("npc shop: %s 帧编码失败: %v", shopID, err)
		return false
	}
	for _, frame := range frames {
		s.sendToClient(pl, frame)
	}
	s.trace("npc shop: %s 开窗（%d 品，%d 页，type=%d）client=%d",
		shopID, len(section.Items), section.Pages, section.Type, pl.Conn1)
	return true
}

// openNPCDepotView 仓库开窗（mark 807000000）。
func (s *Server) openNPCDepotView(pl *Player) {
	frame, err := entergen.CreateViewFrame(npcDepotViewport, npcDepotCapacity, nil)
	if err != nil {
		s.trace("npc depot: 帧编码失败: %v", err)
		return
	}
	s.sendToClient(pl, frame)
	s.trace("npc depot: 开窗 client=%d", pl.Conn1)
}

// npcShopViewFrames 商店视口帧族（纯函数，测试面）：0x15 建视图带四
// 属性 + 每物品一条 0x18。objectIndex=(page-1)*每页10+position+1（2433
// 同式）。capitalType 3=兑换类三价全 0（成本走 exchangeitem 校验）。
func npcShopViewFrames(shopID string, section *npcshop.Section) ([][]byte, error) {
	propIdx := func(name string) uint16 { return clientdata.MustPlayerProperty(name).Index }
	create, err := entergen.CreateViewFrame(npcShopViewport, npcShopCapacity, []entergen.ViewProperty{
		{Index: propIdx("ShopID"), Value: entergen.StringValue(shopID)},
		{Index: propIdx("PageNum"), Value: entergen.Int32Value(npcShopPageSize)},
		{Index: propIdx("PageCount"), Value: entergen.Int32Value(section.Pages)},
		{Index: propIdx("ShopType"), Value: entergen.Int32Value(section.Type)},
	})
	if err != nil {
		return nil, err
	}
	frames := make([][]byte, 0, 1+len(section.Items))
	frames = append(frames, create)
	for _, item := range section.Items {
		props := []entergen.ViewProperty{
			{Index: propIdx("ConfigID"), Value: entergen.StringValue(item.ConfigID)},
			{Index: propIdx("Amount"), Value: entergen.Int32Value(item.Amount)},
			{Index: propIdx("SellPrice0"), Value: entergen.Int32Value(0)},
			{Index: propIdx("SellPrice1"), Value: entergen.Int32Value(0)},
			{Index: propIdx("SellPrice2"), Value: entergen.Int32Value(0)},
			{Index: propIdx("MaxAmount"), Value: entergen.Int32Value(npcShopMaxBuy)},
		}
		switch item.CapitalType {
		case 0:
			props[2].Value = entergen.Int32Value(item.Price)
		case 1:
			props[3].Value = entergen.Int32Value(item.Price)
		case 2:
			props[4].Value = entergen.Int32Value(item.Price)
		case 3:
			// 兑换类：三价全 0 物化（Java Shop_menpai_xtc_01 同款零价
			// 形态，客户端兼容已验证）。
		}
		objectIndex := uint16((item.Page-1)*npcShopPageSize + item.Position + 1)
		row, err := entergen.ViewAddFrame(npcShopViewport, objectIndex, props)
		if err != nil {
			return nil, err
		}
		frames = append(frames, row)
	}
	return frames, nil
}
