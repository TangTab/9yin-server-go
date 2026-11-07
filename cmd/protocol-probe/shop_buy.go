package main

import (
	"fmt"
	"log"
	"math"
)

// 商店购销（fork shop_buy.go 同名 handler 复刻；docs/fork-p2-shop-title-jingmai.md）。
// 70 = handleShopBuyCustom：[70, str shopId, i32 page, i32 pos, i32 count]，
//
//	货位匹配 item.page==page+1 && item.position==pos-1（与 openShopLocked
//	的 objectIndex=(page-1)*10+position+1 同一坐标系）；fork 不查开店态，
//	直接以消息里的 shopId 装载 share/trade/shop.ini 小节。
//
// 71 = handleShopSellCustom：[71, i32 类别页, i32 槽位(1 起), i32 数量,
//
//	str configID]，槽位取 idx-1；fork 出售价为固定 10 碎银/件
//	（0x140455cd9：silver = 10 * count，无逐物品价格表）。
const (
	clientCustomShopBuy  int32 = 70
	clientCustomShopSell int32 = 71

	// shopBuyMaxCount 是 fork 的购买数量上限（handleShopBuyCustom
	// CMPL count, 0x62=98；View61 行 MaxAmount=99 的服务端闸）。
	shopBuyMaxCount = 98
	// shopSellUnitSilver 是 fork 的固定出售单价（碎银/件）。
	shopSellUnitSilver = 10
)

// shopBuyCharge 按 capitalType 扣款；余额不足返回 false（调用方静默拒，
// 不回帧——fork "needs silver %d, has %d" 同口径）。
func shopBuyCharge(player *playerActor, capitalType int32, cost int32, configID string, count int32, remote string) bool {
	switch capitalType {
	case 0:
		if !player.trySpendGold(cost) {
			log.Printf("%s: shop buy item %s x%d needs gold %d, has %d", remote, configID, count, cost, player.goldBalance())
			return false
		}
	case 1:
		if !player.trySpendSilver(cost) {
			log.Printf("%s: shop buy item %s needs silver %d, has %d", remote, configID, cost, player.silverBalance())
			return false
		}
	case 2:
		if !player.trySpendSilverCard(cost) {
			log.Printf("%s: shop buy item %s needs silverCard %d, has %d", remote, configID, cost, player.silverCardBalance())
			return false
		}
	default:
		log.Printf("%s: shop buy item %s unsupported capital type %d", remote, configID, capitalType)
		return false
	}
	return true
}

// shopBuyRefund 扣款后发放失败时的回滚（fork 无此路径，纯内存包满时
// 避免吞钱）。
func shopBuyRefund(player *playerActor, capitalType, cost int32) {
	switch capitalType {
	case 0:
		player.setGold(player.goldBalance() + cost)
	case 1:
		player.addSilver(cost)
	case 2:
		player.addSilverCard(cost)
	}
}

func maxI32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// handleShopBuyCustom 处理 msg70 购买：定位货位 → 按 capitalType 扣款 →
// 入包（同类同页行内叠堆，否则新行）→ 三币回帧 → 货币落库。
func handleShopBuyCustom(h *itemSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomShopBuy {
		return false, nil
	}
	if h.player == nil {
		log.Printf("%s: reject shop buy before player spawn", h.remote)
		return true, nil
	}
	if len(custom.Values) < 5 {
		log.Printf("%s: reject malformed shop buy values=%v", h.remote, custom.Values)
		return true, nil
	}
	if custom.Values[1].Type != 6 && custom.Values[1].Type != 7 {
		log.Printf("%s: reject malformed shop buy values=%v", h.remote, custom.Values)
		return true, nil
	}
	shopID := custom.Values[1].Text
	page, pageOK := custom.Values[2].exactInt32()
	pos, posOK := custom.Values[3].exactInt32()
	count, countOK := custom.Values[4].exactInt32()
	if !pageOK || !posOK || !countOK || page < 0 || pos < 1 || count < 1 || count > shopBuyMaxCount {
		log.Printf("%s: shop buy %s malformed page=%d pos=%d amount=%d", h.remote, shopID, page, pos, count)
		return true, nil
	}
	items, _, _, err := loadShopCatalogSection(defaultShopINIPath, shopID)
	if err != nil {
		log.Printf("%s: shop buy %s catalog: %v", h.remote, shopID, err)
		return true, nil
	}
	var listing *shopCatalogItem
	for index := range items {
		if items[index].page == page+1 && items[index].position == pos-1 {
			listing = &items[index]
			break
		}
	}
	if listing == nil {
		log.Printf("%s: shop buy %s no listing at page=%d pos=%d", h.remote, shopID, page, pos)
		return true, nil
	}
	units := listing.amount
	if units < 1 {
		units = 1
	}
	// 兑换类（capitalType=3）：走 exchangeitem.ini 计价分支
	//（fork shop_buy.go:148 case 3 → exchangeCatalogFor）。
	if listing.capitalType == 3 {
		return h.shopBuyExchange(shopID, *listing, units, count)
	}
	cost := int64(listing.price) * int64(count)
	if cost > math.MaxInt32 {
		cost = math.MaxInt32
	}
	if !shopBuyCharge(h.player, listing.capitalType, int32(cost), listing.configID, count, h.remote) {
		return true, nil
	}
	granted := h.grantShopItemUnits(listing.configID, units*count)
	if granted == 0 {
		shopBuyRefund(h.player, listing.capitalType, int32(cost))
		log.Printf("%s: shop buy %s item %s x%d grant failed (bag full?)", h.remote, shopID, listing.configID, count)
		return true, nil
	}
	frames, frameErr := h.player.currenciesUpdate()
	if frameErr != nil {
		log.Printf("%s: shop buy currency frames: %v", h.remote, frameErr)
	} else {
		for _, frame := range frames {
			if err := h.link.WriteFrame(frame); err != nil {
				return true, fmt.Errorf("write shop buy currency frame: %w", err)
			}
		}
	}
	h.persistCurrency("shop buy")
	log.Printf("%s: shop buy %s item %s x%d capital=%d price=%d -> slot=%d", h.remote, shopID, listing.configID, count, listing.capitalType, listing.price, granted)
	return true, nil
}

// exchangePropBudget 把 Prop=CapitalTypeN 映射到会话货币余额检查
// （fork 只认服务端跟踪的货币，否则 "uses untracked currency (refused)"）。
// 返回按币种的待扣额；第二返回值 false = 拒绝。
func exchangePropBudget(player *playerActor, props []exchangePropCost, remote string, exchangeID int32) (map[int32]int32, bool) {
	budget := make(map[int32]int32, len(props))
	for _, prop := range props {
		var capital, balance int32
		switch prop.name {
		case "CapitalType0":
			capital, balance = 0, player.goldBalance()
		case "CapitalType1":
			capital, balance = 1, player.silverBalance()
		case "CapitalType2":
			capital, balance = 2, player.silverCardBalance()
		default:
			log.Printf("%s: shop buy exchange %d uses untracked currency %s (refused)", remote, exchangeID, prop.name)
			return nil, false
		}
		budget[capital] += prop.cost
		if budget[capital] > balance {
			log.Printf("%s: shop buy exchange %d needs %s %d, has %d", remote, exchangeID, prop.name, budget[capital], balance)
			return nil, false
		}
	}
	return budget, true
}

// shopBuyExchange 兑换购买：全部校验通过后才动账——物品数量
// （countBagConfig）→ Prop 货币余额 → 包容量预检（杜绝扣完发不出的
// 回滚窗口）→ 跨行扣物 → 扣币 → 发放。Condition 条件门不评估仅记日志
// （fork "conditions not evaluated" 同口径）。
func (h *itemSession) shopBuyExchange(shopID string, listing shopCatalogItem, units, count int32) (bool, error) {
	catalog, catalogErr := exchangeCatalogFor(defaultExchangeINIPath)
	if catalogErr != nil {
		log.Printf("%s: shop buy %s item %s exchange catalog: %v", h.remote, shopID, listing.configID, catalogErr)
		return true, nil
	}
	entry, ok := catalog.entry(listing.exchangeID)
	if !ok || entry.empty() {
		log.Printf("%s: shop buy %s item %s no exchange entry %d", h.remote, shopID, listing.configID, listing.exchangeID)
		return true, nil
	}
	if entry.condition != "" {
		log.Printf("%s: shop buy %s item %s exchange %d conditions not evaluated (%s)", h.remote, shopID, listing.configID, listing.exchangeID, entry.condition)
	}
	for _, cost := range entry.items {
		if has := h.player.countBagConfig(cost.configID); has < cost.count {
			log.Printf("%s: shop buy %s item %s needs %d x%s, has %d", h.remote, shopID, listing.configID, cost.count, cost.configID, has)
			return true, nil
		}
	}
	budget, budgetOK := exchangePropBudget(h.player, entry.props, h.remote, listing.exchangeID)
	if !budgetOK {
		return true, nil
	}
	// 容量预检：可叠堆直接放行；否则按发放目录的类别页查空位
	//（无容量先拒，扣费零回滚；查不到发放目录同样先拒）。
	if _, stacked := h.player.stackTargetFor(listing.configID); !stacked {
		grantable, known := h.store.lookupGrantable(h.ctx, listing.configID)
		if !known {
			log.Printf("%s: shop buy %s item %s no grantable entry", h.remote, shopID, listing.configID)
			return true, nil
		}
		if h.player.freeBagSlot(grantable.viewID) == 0 {
			log.Printf("%s: shop buy %s item %s x%d bag full", h.remote, shopID, listing.configID, count)
			return true, nil
		}
	}
	for _, cost := range entry.items {
		if !h.consumeBagConfig(cost.configID, cost.count) {
			log.Printf("%s: shop buy %s item %s exchange material lost after check", h.remote, shopID, listing.configID)
			return true, nil
		}
	}
	for capital, cost := range budget {
		switch capital {
		case 0:
			h.player.trySpendGold(cost)
		case 1:
			h.player.trySpendSilver(cost)
		case 2:
			h.player.trySpendSilverCard(cost)
		}
	}
	granted := h.grantShopItemUnits(listing.configID, units*count)
	if granted == 0 {
		// 预检后仍失败（极端竞态）：退币，材料损耗记错误日志。
		for capital, cost := range budget {
			shopBuyRefund(h.player, capital, cost)
		}
		log.Printf("%s: shop buy %s item %s exchange grant failed after charge (materials lost)", h.remote, shopID, listing.configID)
		return true, nil
	}
	if len(budget) > 0 {
		frames, frameErr := h.player.currenciesUpdate()
		if frameErr != nil {
			log.Printf("%s: shop buy exchange currency frames: %v", h.remote, frameErr)
		} else {
			for _, frame := range frames {
				if err := h.link.WriteFrame(frame); err != nil {
					return true, fmt.Errorf("write shop buy exchange currency frame: %w", err)
				}
			}
		}
		h.persistCurrency("shop buy")
	}
	log.Printf("%s: shop buy %s item %s x%d via exchange %d (items=%d props=%d) -> slot=%d", h.remote, shopID, listing.configID, count, listing.exchangeID, len(entry.items), len(entry.props), granted)
	return true, nil
}

// grantShopItemUnits 发放 total 件：同页同类同 configID 行内叠堆优先，
// 否则 lookupGrantable + grantBagItem 新行。返回落包槽位（0=失败）。
func (h *itemSession) grantShopItemUnits(configID string, total int32) int32 {
	if row, ok := h.player.stackTargetFor(configID); ok && h.addToBagItemCount(row, total) {
		return row.slot
	}
	entry, ok := h.store.lookupGrantable(h.ctx, configID)
	if !ok {
		log.Printf("%s: no grantable item %s", h.remote, configID)
		return 0
	}
	return h.grantBagItem(entry, total)
}

// handleShopSellCustom 处理 msg71 出售：类别页映射同 bagViewOf
// （2→121/3→123/4→125/缺省工具页 2），槽位 idx-1；扣件 → +10×count
// 碎银 → 三币回帧 → 货币落库。
func handleShopSellCustom(h *itemSession, custom clientCustomMessage) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomShopSell {
		return false, nil
	}
	if h.player == nil {
		log.Printf("%s: reject shop sell before player spawn", h.remote)
		return true, nil
	}
	if len(custom.Values) < 5 {
		log.Printf("%s: reject malformed shop sell values=%v", h.remote, custom.Values)
		return true, nil
	}
	category, categoryOK := custom.Values[1].exactInt32()
	idx, idxOK := custom.Values[2].exactInt32()
	count, countOK := custom.Values[3].exactInt32()
	if !categoryOK || !idxOK || !countOK || category < 0 || idx < 1 || count < 1 {
		log.Printf("%s: reject shop sell view=%d idx=%d count=%d", h.remote, category, idx, count)
		return true, nil
	}
	configID := ""
	if custom.Values[4].Type == 6 || custom.Values[4].Type == 7 {
		configID = custom.Values[4].Text
	}
	if category < 1 || category > 4 {
		category = 1 // fork 缺省分支：落工具页（bagViewOf 同映射）
	}
	slot := idx - 1
	view := bagViewOf(category)
	row, _, ok := h.player.bagItemAt(view, slot)
	if !ok {
		log.Printf("%s: shop sell no item at view=%d idx=%d", h.remote, view, idx)
		return true, nil
	}
	if configID != "" && configID != row.configID {
		log.Printf("%s: shop sell stale cell configID=%s want %s at view=%d idx=%d", h.remote, configID, row.configID, view, idx)
		return true, nil
	}
	if row.amount < count {
		log.Printf("%s: shop sell %s x%d exceeds stack %d", h.remote, row.configID, count, row.amount)
		return true, nil
	}
	if !h.consumeBagItemCount(row, count) {
		return true, nil
	}
	proceeds := shopSellUnitSilver * count
	silver := h.player.addSilver(proceeds)
	frames, frameErr := h.player.currenciesUpdate()
	if frameErr != nil {
		log.Printf("%s: shop sell currency frames: %v", h.remote, frameErr)
	} else {
		for _, frame := range frames {
			if err := h.link.WriteFrame(frame); err != nil {
				return true, fmt.Errorf("write shop sell currency frame: %w", err)
			}
		}
	}
	h.persistCurrency("shop sell")
	log.Printf("%s: shop sell item=%s x%d -> silver +%d (now %d)", h.remote, row.configID, count, proceeds, silver)
	return true, nil
}
