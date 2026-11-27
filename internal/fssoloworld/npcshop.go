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
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
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

// ---- 购销受理（msg70/71，2433 shop_buy.go fork 同名 handler 复刻） ----

const (
	clientCustomShopBuy  int32 = 70
	clientCustomShopSell int32 = 71
	// shopBuyMaxCount 购买数量上限（fork CMPL count,0x62=98；View61 行
	// MaxAmount=99 的服务端闸）。
	shopBuyMaxCount int32 = 98
	// shopSellUnitSilver fork 固定出售单价（碎银/件，无逐物品价格表）。
	shopSellUnitSilver int64 = 10
)

// 物品目录解析见 itemcatalog.go（ini/Item 构建产物优先，设计态 ini 回退）。

// handleShopBuy msg70 [70, shopId str, page, pos, count]：货位匹配
// （item.page==page+1 && item.position==pos-1，与开窗 objectIndex 同坐标
// 系）→ 按 capitalType 扣款（本线仅碎银 CapitalType1=roleState.Money；
// 0/2 未建模货币、3 兑换链待接——静默拒 + trace，fork 同口径）→
// 入包（同类同页行内叠堆优先，否则新行）→ 三币回帧（碎银 0x10）→
// 落档。
func (s *Server) handleShopBuy(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 5 {
		s.trace("npc shop buy: 畸形 values=%d client=%d", len(message.Values), pl.Conn1)
		return
	}
	shopID, ok := customStringAt(message, 1)
	if !ok {
		s.trace("npc shop buy: shopId 非串 client=%d", pl.Conn1)
		return
	}
	page, pageOK := message.Values[2].ExactInt32()
	pos, posOK := message.Values[3].ExactInt32()
	count, countOK := message.Values[4].ExactInt32()
	if !pageOK || !posOK || !countOK || page < 0 || pos < 1 || count < 1 || count > shopBuyMaxCount {
		s.trace("npc shop buy: %s 越界 page=%d pos=%d count=%d client=%d", shopID, page, pos, count, pl.Conn1)
		return
	}
	catalog := s.shopCatalogOf()
	section, ok := catalog.Section(shopID)
	if !ok {
		s.trace("npc shop buy: %s 无目录节 client=%d", shopID, pl.Conn1)
		return
	}
	var listing *npcshop.Item
	for i := range section.Items {
		if section.Items[i].Page == page+1 && section.Items[i].Position == pos-1 {
			listing = &section.Items[i]
			break
		}
	}
	if listing == nil {
		s.trace("npc shop buy: %s 无货位 page=%d pos=%d client=%d", shopID, page, pos, pl.Conn1)
		return
	}
	units := listing.Amount
	if units < 1 {
		units = 1
	}
	switch listing.CapitalType {
	case 1:
		cost := int64(listing.Price) * int64(count)
		if pl.roleState.Money < cost {
			s.trace("npc shop buy: %s %s x%d 需碎银 %d 现有 %d client=%d",
				shopID, listing.ConfigID, count, cost, pl.roleState.Money, pl.Conn1)
			return
		}
		if !s.grantShopItemUnits(pl, listing.ConfigID, units*count) {
			return // 满包/物品未知（已 trace）；不扣款
		}
		pl.roleState.Money -= cost
	case 0, 2:
		s.trace("npc shop buy: %s %s capitalType=%d 未建模货币拒收 client=%d",
			shopID, listing.ConfigID, listing.CapitalType, pl.Conn1)
		return
	default: // 3 兑换链（exchangeitem.ini）待接
		s.trace("npc shop buy: %s %s 兑换类（exchange）待接 client=%d", shopID, listing.ConfigID, pl.Conn1)
		return
	}
	s.pushMoneyFrame(pl)
	s.sendRoleSave(pl)
	s.trace("npc shop buy: %s %s x%d（单价 %d）→ 碎银余 %d client=%d",
		shopID, listing.ConfigID, count, listing.Price, pl.roleState.Money, pl.Conn1)
}

// grantShopItemUnits 商店购货入包（行变化即时推送）。
func (s *Server) grantShopItemUnits(pl *Player, configID string, total int32) bool {
	return s.grantItemUnits(pl, configID, int(total), "npc shop buy", func(item rolestate.BagItem, id string) {
		s.pushBagRow(pl, item, id)
	})
}

// grantItemUnits 发放 total 件：同 configID 同分类页的未满行先叠堆
// （上限 = 模板行 MaxAmount；目录缺席/未标不设限），满了开新行——不可堆叠
// 物（MaxAmount=1）逐件成行；新行查物品目录取 ItemType 真值、页标与
// MaxAmount 入档。先做容量预检：放不下整单则一件不发、返回 false（调用方
// 据此不扣款——避免「部分入包不收费」）。每条变化的背包行交给 sink（what
// 是日志前缀）：购销链即时推送，任务奖励链先攒帧、按官服帧序下发。
func (s *Server) grantItemUnits(pl *Player, configID string, total int, what string, sink func(rolestate.BagItem, string)) bool {
	st := &pl.roleState
	entry, ok := s.itemCatalogOf().Lookup(configID)
	if !ok {
		s.trace("%s: 物品 %s 无目录条目拒发", what, configID)
		return false
	}
	// 预检：现有行还能叠多少、余量要开几个新行。
	remaining := int(total)
	for i := range st.BagItems {
		row := &st.BagItems[i]
		if remaining == 0 || row.ConfigID != configID || row.ViewID != entry.ViewID {
			continue
		}
		capacity := row.MaxAmount
		if capacity <= 0 {
			capacity = entry.MaxAmount
		}
		room := remaining
		if capacity > 0 {
			room = max(min(room, capacity-row.Amount), 0)
		}
		remaining -= room
	}
	newRows := 0
	if remaining > 0 {
		newRows = 1
		if entry.MaxAmount > 0 {
			newRows = (remaining + entry.MaxAmount - 1) / entry.MaxAmount
		}
	}
	if free := st.FreeBagSlots(entry.ViewID); free < newRows {
		s.trace("%s: %s x%d 分类页 %d 空位不足（需 %d 行，余 %d）",
			what, configID, total, entry.ViewID, newRows, free)
		return false
	}
	// 落包：先补未满行，再开新行。
	remaining = int(total)
	for i := range st.BagItems {
		row := &st.BagItems[i]
		if remaining == 0 || row.ConfigID != configID || row.ViewID != entry.ViewID {
			continue
		}
		take := remaining
		if entry.MaxAmount > 0 {
			take = max(min(take, entry.MaxAmount-row.Amount), 0)
		}
		row.Amount += take
		row.MaxAmount = entry.MaxAmount
		remaining -= take
		if take > 0 {
			sink(*row, configID)
		}
	}
	for remaining > 0 {
		take := remaining
		if entry.MaxAmount > 0 {
			take = min(take, entry.MaxAmount)
		}
		item := rolestate.BagItem{
			Slot:       st.FreeBagSlot(entry.ViewID),
			ConfigID:   configID,
			ItemType:   entry.ItemType,
			Amount:     take,
			ViewID:     entry.ViewID,
			Seq:        st.NextItemSeq(),
			EquipType:  entry.EquipType,
			ColorLevel: entry.ColorLevel,
			Photo:      entry.Photo,
			MaxAmount:  entry.MaxAmount,
		}
		st.BagItems = append(st.BagItems, item)
		remaining -= take
		sink(item, configID)
	}
	return true
}

// pushBagRow 背包行成对推送（0x19 预清 + 0x18 行；编码失败只记日志）。
func (s *Server) pushBagRow(pl *Player, item rolestate.BagItem, configID string) {
	push, err := entergen.BagItemPushFrames(item)
	if err != nil {
		s.trace("bag row: 帧编码失败 %s: %v", configID, err)
		return
	}
	for _, f := range push {
		s.sendToClient(pl, f)
	}
}

// handleShopSell msg71 [71, 类别页, 槽位(1 起), 数量, configID str]：
// 槽位打中行 → 校验 configID/数量 → 扣件（清行 0x19 / 减堆 0x19+0x18）→
// +10 碎银/件 → 三币回帧 → 落档。
func (s *Server) handleShopSell(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 5 {
		s.trace("npc shop sell: 畸形 values=%d client=%d", len(message.Values), pl.Conn1)
		return
	}
	category, catOK := message.Values[1].ExactInt32()
	idx, idxOK := message.Values[2].ExactInt32()
	count, countOK := message.Values[3].ExactInt32()
	if !catOK || !idxOK || !countOK || idx < 1 || count < 1 {
		s.trace("npc shop sell: 越界 category=%d idx=%d count=%d client=%d", category, idx, count, pl.Conn1)
		return
	}
	if category < 1 || category > 4 {
		category = 1 // fork 缺省分支：落道具页
	}
	configID, _ := customStringAt(message, 4) // 校验用，缺席放行
	st := &pl.roleState
	rowIdx := -1
	for i := range st.BagItems {
		if st.BagItems[i].ViewID == int(category) && st.BagItems[i].Slot == int(idx) {
			rowIdx = i
			break
		}
	}
	if rowIdx < 0 {
		s.trace("npc shop sell: 无行 view=%d idx=%d client=%d", category, idx, pl.Conn1)
		return
	}
	row := &st.BagItems[rowIdx]
	if configID != "" && configID != row.ConfigID {
		s.trace("npc shop sell: 过期格 configID=%s 应为 %s view=%d idx=%d", configID, row.ConfigID, category, idx)
		return
	}
	if row.Amount < int(count) {
		s.trace("npc shop sell: %s x%d 超堆 %d", row.ConfigID, count, row.Amount)
		return
	}
	row.Amount -= int(count)
	wire := entergen.BagViewOf(int(category))
	if row.Amount == 0 {
		st.BagItems = append(st.BagItems[:rowIdx], st.BagItems[rowIdx+1:]...)
		for _, frame := range entergen.BagRowRemoveFrames(wire, uint16(idx)) {
			s.sendToClient(pl, frame)
		}
	} else {
		push, err := entergen.BagItemPushFrames(*row)
		if err != nil {
			s.trace("npc shop sell: 减堆帧编码失败 %s: %v", row.ConfigID, err)
			return
		}
		for _, f := range push {
			s.sendToClient(pl, f)
		}
	}
	proceeds := shopSellUnitSilver * int64(count)
	pl.roleState.Money += proceeds
	s.pushMoneyFrame(pl)
	s.sendRoleSave(pl)
	s.trace("npc shop sell: %s x%d → 碎银 +%d（余 %d）client=%d",
		row.ConfigID, count, proceeds, pl.roleState.Money, pl.Conn1)
}

// pushMoneyFrame 碎银三币回帧（CapitalType1 0x10 单属性——gmcmd
// moneyFrame 同帧型，购销共用）。
func (s *Server) pushMoneyFrame(pl *Player) {
	frame, err := entergen.ObjectPropertyFrame(1, 1, 0, []entergen.IndexedProperty{
		entergen.NamedProperty("CapitalType1", entergen.Int64Value(pl.roleState.Money)),
	})
	if err != nil {
		s.trace("npc shop: 碎银帧编码失败: %v", err)
		return
	}
	s.sendToClient(pl, frame)
}
