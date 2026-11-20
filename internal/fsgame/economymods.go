package fsgame

import (
	"fmt"
	"strings"
)

// 六期：经济域五模块（docs/official/FsGame全解与Go复刻.md §8 经济）。
// 官方源码树：fsgame\trademodule\（shopmodule/auctionhousemodule）。

// ---- ShopModule ----

// ShopModule 复刻官方 trademodule\shopmodule（Init @0x4425C0 /
// LoadResource @0x442D00，玩法表 Init 序 73；符号 1147，7 心跳）——商店。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnRecover",@0x446270) / ("player","OnReady",
//	    @0x446510) / ("player","OnDoSelect",@0x446630 OnPlayerDoSelect)
//	AddIntCommandHook ("player",22,@0x4477F0 OnMenuSelect) / (625,@0x4477A0
//	    OnBeAttacked) / (616,@0x447750 OnBeKilled)
//	AddIntCustomHook ("player",70=0x46,@0x466F0) 购物命令多路
//	AddClassCallback ("ShopBox","OnCreateClass",@0x445F20) / AddEventCallback
//	    ("ShopBox","OnCreate",@0x445F70)
//	AddClassCallback ("ReclaimBox","OnCreateClass",@0x445FD0) /
//	    ("ReclaimBox","OnCreate",@0x446020)
//	AddEventCallback ("CommonNpc","OnLoad",@0x446680) / ("CommonNpc",
//	    "OnSelect",@0x447470 OnNpcSelect——开商店)
//	DeclareHeartBeat：BaseShop::H_ArriveRefreshTime@0x461E50 /
//	    H_RefreshItem@0x461F10 / H_ShopItemCoolDown@0x461D70 /
//	    ShopModule::H_ClearReclaim
//
// LoadResource @0x442D00（失败串定案）：ini/trade/Shop/Shop.xml
//（76 店：ID/ShopType/RepairType/RefreshItemTime="12,30"/SellID 逗号商品表/
// Camp）+ ini/trade/Shop/ShopItem.xml（2755 品：ItemID/Maxamount/
// CurrencyType/BuyPrice(-1 非卖)/isbind/SellNum(-1 不限)/CoolTime）。
type ShopModule struct {
	BaseModule

	// Shops 店表；Items 商品表。
	Shops map[string]map[string]string
	Items map[string]map[string]string

	// CostOut 货币扣减出口（官方经 CapitalModule；Go 面 sync 挂接）。
	CostOut func(player PersistID, currencyType int32, amount int64) bool
	// BuyOut 购买到货出口（官方经 ShopBox 容器；接 ToolBox.AddItem）。
	BuyOut func(player PersistID, itemID string, count int32)
	// sold 售卖流水（测试面）。
	sold []string
}

// Name 模块名。
func (m *ShopModule) Name() string { return "ShopModule" }

// Init 注册期（对齐官方 0x4425C0）。
func (m *ShopModule) Init(k *Kernel) error {
	m.Shops = make(map[string]map[string]string)
	m.Items = make(map[string]map[string]string)
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnDoSelect", m.onPlayerDoSelect)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	b.OnIntCommand("player", 625, m.onBeAttacked)
	b.OnIntCommand("player", 616, m.onBeKilled)
	b.OnCommand("player", 70, m.onCustomMsg)
	b.OnClassEvent("ShopBox", "OnCreateClass", m.onShopBoxCreateClass)
	b.OnEvent("ShopBox", "OnCreate", m.onShopBoxCreate)
	b.OnClassEvent("ReclaimBox", "OnCreateClass", m.onReclaimBoxCreateClass)
	b.OnEvent("ReclaimBox", "OnCreate", m.onReclaimBoxCreate)
	b.OnEvent("CommonNpc", "OnLoad", m.onNpcLoad)
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.HeartBeat("BaseShop::H_ArriveRefreshTime", m.hbArriveRefreshTime)
	b.HeartBeat("BaseShop::H_RefreshItem", m.hbRefreshItem)
	b.HeartBeat("BaseShop::H_ShopItemCoolDown", m.hbShopItemCoolDown)
	b.HeartBeat("ShopModule::H_ClearReclaim", m.hbClearReclaim)
	return nil
}

// LoadResource 官方 @0x442D00：ini/trade/Shop/Shop.xml + ShopItem.xml。
func (m *ShopModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: ShopModule.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/trade/Shop/Shop.xml")
	if err != nil {
		return fmt.Errorf("fsgame: [Error]ini/trade/Shop.xml Load Failed: %w", err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	for id, row := range ruleByID(rows) {
		m.Shops[id] = row
	}
	data, err = readResource(k.ResourcePath, "ini/trade/Shop/ShopItem.xml")
	if err != nil {
		return fmt.Errorf("fsgame: [Error]ini/trade/ShopItem.xml Load Failed: %w", err)
	}
	if rows, err = ParsePropTable(data); err != nil {
		return err
	}
	for id, row := range ruleByID(rows) {
		m.Items[id] = row
	}
	k.Tracef("ShopModule: LoadResource 店 %d / 商品 %d", len(m.Shops), len(m.Items))
	return nil
}

// Buy 购买闭环：店表 SellID 含商品门 → BuyPrice>0（-1 非卖）→ Maxamount
// 限购（SoldOut 面记）→ 货币扣减（CostOut）→ 到货（BuyOut）。
func (m *ShopModule) Buy(k *Kernel, self PersistID, shopID, itemID string, count int32) bool {
	shop := m.Shops[shopID]
	item := m.Items[itemID]
	if shop == nil || item == nil {
		return false
	}
	// 店内商品门（SellID 逗号表）。
	inShop := false
	for _, sid := range strings.Split(shop["SellID"], ",") {
		if strings.TrimSpace(sid) == itemID {
			inShop = true
			break
		}
	}
	if !inShop {
		return false
	}
	price := int64(atoi32(item["BuyPrice"]))
	if price < 0 {
		return false // -1 非卖品
	}
	if max := atoi32(item["Maxamount"]); max > 0 && count > max {
		count = max
	}
	if count <= 0 {
		return false
	}
	if m.CostOut != nil && !m.CostOut(self, atoi32(item["CurrencyType"]), price*int64(count)) {
		return false
	}
	if m.BuyOut != nil {
		m.BuyOut(self, itemID, count)
	}
	m.sold = append(m.sold, fmt.Sprintf("%s×%d@%s", itemID, count, shopID))
	return true
}

// onCustomMsg 官方 @0x466F0（cust 70=0x46）：购物命令
//（args: 子命令, 店ID, 商品ID, 数量——0 购买族；ConvoyPointRec 联动面）。
func (m *ShopModule) onCustomMsg(k *Kernel, self, from PersistID, args VarList) {
	switch args.Int(0) {
	case 0: // 购买
		m.Buy(k, self, args.String(1), args.String(2), args.Int(3))
	default:
		k.Tracef("ShopModule: subcmd %d", args.Int(0))
	}
}

// onNpcSelect 官方 @0x447470：NPC 选中开店（NPC→ShopID 绑定经商店配置）。
func (m *ShopModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("ShopModule: OnNpcSelect (open shop)")
}

// onNpcLoad 官方 @0x446680。
func (m *ShopModule) onNpcLoad(k *Kernel, self, from PersistID, args VarList) {}

// onRecover 官方 @0x446270：限购/冷却计数回载。
func (m *ShopModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}

// onReady 官方 @0x446510。
func (m *ShopModule) onReady(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerDoSelect 官方 @0x446630。
func (m *ShopModule) onPlayerDoSelect(k *Kernel, self, from PersistID, args VarList) {}

// onMenuSelect 官方 @0x4477F0（命令 22）。
func (m *ShopModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {}

// onBeAttacked 官方 @0x4477A0（命令 625）。
func (m *ShopModule) onBeAttacked(k *Kernel, self, from PersistID, args VarList) {}

// onBeKilled 官方 @0x447750（命令 616）。
func (m *ShopModule) onBeKilled(k *Kernel, self, from PersistID, args VarList) {}

// onShopBoxCreateClass/OnCreate 官方 @0x445F20/@0x445F70：商店容器。
func (m *ShopModule) onShopBoxCreateClass(k *Kernel, class string)            {}
func (m *ShopModule) onShopBoxCreate(k *Kernel, self, from PersistID, args VarList) {}

// onReclaimBoxCreateClass/OnCreate 官方 @0x445FD0/@0x446020：回购容器。
func (m *ShopModule) onReclaimBoxCreateClass(k *Kernel, class string)              {}
func (m *ShopModule) onReclaimBoxCreate(k *Kernel, self, from PersistID, args VarList) {}

// H_ArriveRefreshTime 官方 @0x461E50：到点刷新（RefreshItemTime="12,30"——
// 每日 12:30 类时刻表）。
func (m *ShopModule) hbArriveRefreshTime(k *Kernel, self PersistID, beats int32) {}

// H_RefreshItem 官方 @0x461F10：限购/冷却刷新。
func (m *ShopModule) hbRefreshItem(k *Kernel, self PersistID, beats int32) {}

// H_ShopItemCoolDown 官方 @0x461D70：商品购买冷却。
func (m *ShopModule) hbShopItemCoolDown(k *Kernel, self PersistID, beats int32) {}

// H_ClearReclaim 官方：回购栏清理。
func (m *ShopModule) hbClearReclaim(k *Kernel, self PersistID, beats int32) {}

// Sold 售卖流水（测试面）。
func (m *ShopModule) Sold() []string { return m.sold }

// ---- VipModule ----

// VipModule 复刻官方 vipmodule（Init @0x216780，玩法表 Init 序 123；
// 符号 555）——VIP 面。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnReady",@0x218E00)
//	AddIntCustomHook ("player",65=0x41,@0x217660 OnCustomRequestGift——
//	    VipGiftTime 记录 + vip_get_gift 串) / ("player",64=0x40,
//	    @0x219630 OnCustomOpenVip)
//	AddIntCommandHook ("player",321=0x141,@0x21A150 OnVipDecTicketRes)
//	DeclareHeartBeat ("VipModule::HB_VipValidTime",@0x217480——IsVip
//	    有效期检查)
//
// 属性（player schema）：IsVip(byte, save) / VipGiftTime(string, save)。
type VipModule struct {
	BaseModule

	// GiftOut 礼包发放出口（接 GiftModule/ToolBox）。
	GiftOut func(player PersistID, giftID string, count int32)
	// ops 命令流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *VipModule) Name() string { return "VipModule" }

// Init 注册期（对齐官方 0x216780）。
func (m *VipModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnReady", m.onPlayerReady)
	b.OnCommand("player", 65, m.onCustomRequestGift)
	b.OnCommand("player", 64, m.onCustomOpenVip)
	b.OnIntCommand("player", 321, m.onVipDecTicketRes)
	b.HeartBeat("VipModule::HB_VipValidTime", m.hbVipValidTime)
	return nil
}

// onPlayerReady 官方 @0x218E00：VIP 状态同步。
func (m *VipModule) onPlayerReady(k *Kernel, self, from PersistID, args VarList) {}

// onCustomRequestGift 官方 @0x217660（cust 65=0x41）：领 VIP 礼包
//（IsVip 门 + VipGiftTime 每日一次 → vip_get_gift）。
func (m *VipModule) onCustomRequestGift(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil || o.GetInt("IsVip") == 0 {
		return
	}
	today := fmt.Sprintf("%d", nowUnix()/86400)
	if o.GetString("VipGiftTime") == today {
		return // 今日已领
	}
	_ = k.SetString(self, "VipGiftTime", today)
	if m.GiftOut != nil {
		m.GiftOut(self, "vip_get_gift", 1)
	}
	m.ops = append(m.ops, "gift")
}

// onCustomOpenVip 官方 @0x219630（cust 64=0x40）：VIP 面板。
func (m *VipModule) onCustomOpenVip(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "open")
}

// onVipDecTicketRes 官方 @0x21A150（命令 321=0x141）：VIP 购券回执。
func (m *VipModule) onVipDecTicketRes(k *Kernel, self, from PersistID, args VarList) {}

// HB_VipValidTime 官方 @0x217480：VIP 有效期检查（到期清 IsVip）。
func (m *VipModule) hbVipValidTime(k *Kernel, self PersistID, beats int32) {}

// Ops 命令流水（测试面）。
func (m *VipModule) Ops() []string { return m.ops }

// ---- GiftModule ----

// GiftModule 复刻官方 giftmodule（Init @0xB531D0，玩法表 Init 序 130；
// 符号 43）——充值/活动礼包。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnRecover",@0xB53370)
//	AddEventCallback ("scene","OnQueryGift",@0xB533F0 OnQueryGift——
//	    充值中心查询（CapitalType3 面）)
//	AddEventCallback ("scene","OnGetGift",@0xB53570 OnGetGift——
//	    日志串定案 "ongetgift:result=%d,giftid=%d,gift=%s,count=%d,sendmail=%d"）
//	AddIntCommandHook ("player",190=0xBE,@0xB537B0 OnPlayerCmd)
//	DeclareHeartBeat ("GiftModule::HB_QueryGift",@0xB53C60)
//
// 落账：GiftWaitPostGetRec（struct/player/letter.xml 账号领奖等待收取表，
// 3 列：礼物id/名称/数量）或直发（sendmail 分支）。
type GiftModule struct {
	BaseModule

	// GiftOut 礼包到货出口（sendmail=0 直发面；接 ToolBox）。
	GiftOut func(player PersistID, giftID, giftName string, count int32)
	// grants 发放流水（测试面）。
	grants []string
}

// Name 模块名。
func (m *GiftModule) Name() string { return "GiftModule" }

// Init 注册期（对齐官方 0xB531D0）。
func (m *GiftModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("scene", "OnQueryGift", m.onQueryGift)
	b.OnEvent("scene", "OnGetGift", m.onGetGift)
	b.OnIntCommand("player", 190, m.onPlayerCmd)
	b.HeartBeat("GiftModule::HB_QueryGift", m.hbQueryGift)
	return nil
}

// onQueryGift 官方 @0xB533F0：充值中心礼包查询（CapitalType3）。
func (m *GiftModule) onQueryGift(k *Kernel, self, from PersistID, args VarList) {}

// onGetGift 官方 @0xB53570：礼包发放
//（args: result, giftid, gift, count, sendmail——日志串五元组定案）。
// sendmail=1 → GiftWaitPostGetRec 邮件待收；0 → 直发出口。
func (m *GiftModule) onGetGift(k *Kernel, self, from PersistID, args VarList) {
	giftID, giftName := args.String(1), args.String(2)
	count := args.Int(3)
	if giftID == "" || count <= 0 {
		return
	}
	if args.Int(4) != 0 { // sendmail：邮件待收
		if o := k.GetGameObj(from); o != nil {
			if r := o.Record("GiftWaitPostGetRec"); r != nil && r.Rows() < r.Def().MaxRows {
				r.AppendRow(VarList{atoi32(giftID), giftName, count})
			}
		}
	} else if m.GiftOut != nil {
		m.GiftOut(from, giftID, giftName, count)
	}
	m.grants = append(m.grants, fmt.Sprintf("%s×%d(mail=%d)", giftID, count, args.Int(4)))
	k.Tracef("GiftModule: ongetgift:result=%d,giftid=%s,gift=%s,count=%d,sendmail=%d",
		args.Int(0), giftID, giftName, count, args.Int(4))
}

// onPlayerCmd 官方 @0xB537B0（命令 190=0xBE）：礼包面板命令。
func (m *GiftModule) onPlayerCmd(k *Kernel, self, from PersistID, args VarList) {}

// onRecover 官方 @0xB53370：待收礼包回载。
func (m *GiftModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}

// HB_QueryGift 官方 @0xB53C60：礼包轮询（充值中心异步）。
func (m *GiftModule) hbQueryGift(k *Kernel, self PersistID, beats int32) {}

// Grants 发放流水（测试面）。
func (m *GiftModule) Grants() []string { return m.grants }

// ---- AuctionHouseModule ----

// AuctionHouseModule 复刻官方 trademodule\auctionhousemodule
// （Init @0x41DD40 / LoadResource @0x41E630，玩法表 Init 序 71；符号 596）
// ——拍卖行会员侧桥（拍卖撮合本体在 FsAuctionLogic/FsSoloAuction 服）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddClassCallback ("player","OnCreateClass",@0x41DF80——auction.xml
//	    schema：rec_consign_args(4×300 寄卖参数)/rec_search_args(13×1)/
//	    rec_my_consign(1×150)/rec_my_bid(2×200)）
//	AddIntCustomHook ("player",197=0xC5,@0x42AEC0 OnCustomMsg——
//	    get_auction_msg 多路）
//	DeclareLuaExt ×2：nx_rebuild_ah_recs@0x41DB10 / nx_clear_ah_data@0x41DC40
//
// LoadResource @0x41E630：ini/trade/AuctionHouse/ItemType.xml
//（80 分类：ID/BigType/Father 树）。
type AuctionHouseModule struct {
	BaseModule

	// AuctionOut 拍卖行上行出口（官方 IKernel::SendAuctionMessage →
	// FsAuctionLogic 服，已复刻 internal/ 侧）。
	AuctionOut func(player PersistID, op string, args VarList)

	// ItemTypes 拍卖分类树。
	ItemTypes map[string]map[string]string
	// ops 命令流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *AuctionHouseModule) Name() string { return "AuctionHouseModule" }

// Init 注册期（对齐官方 0x41DD40）。
func (m *AuctionHouseModule) Init(k *Kernel) error {
	m.ItemTypes = make(map[string]map[string]string)
	b := k.Bus()
	b.OnClassEvent("player", "OnCreateClass", m.onCreateClass)
	b.OnCommand("player", 197, m.onCustomMsg)
	return nil
}

// LoadResource 官方 @0x41E630：ini/trade/AuctionHouse/ItemType.xml
//（失败串 "[Error]ini/AuctionHouse/ItemType.xml Load Failed"）。
// 另按官方 OnCreateClass @0x41DF80 面：auction.xml 记录族（rec_consign_args/
// rec_my_consign/rec_my_bid/rec_search_args）并入 player 类
//（不触发 OnCreateClass 二次回调）。
func (m *AuctionHouseModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: AuctionHouseModule.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/trade/AuctionHouse/ItemType.xml")
	if err != nil {
		return fmt.Errorf("fsgame: [Error]ini/AuctionHouse/ItemType.xml Load Failed: %w", err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	for id, row := range ruleByID(rows) {
		m.ItemTypes[id] = row
	}
	// auction.xml 记录并入 player 类（官方 createclass_from_xml 动态面）。
	if ax, err := readResource(k.ResourcePath, "struct/player/auction.xml"); err == nil {
		if cd := k.classes["player"]; cd != nil {
			if _, err := ParseStructXML(bytesReader(ax), cd); err == nil {
				delete(k.merged, "player")
			}
		}
	}
	k.Tracef("AuctionHouseModule: LoadResource ItemType %d 分类", len(m.ItemTypes))
	return nil
}

// onCreateClass 官方 @0x41DF80：player schema 就绪（auction 记录族登记面）。
func (m *AuctionHouseModule) onCreateClass(k *Kernel, class string) {
	k.Tracef("AuctionHouseModule: OnCreateClass player (auction.xml recs)")
}

// onCustomMsg 官方 @0x42AEC0（cust 197=0xC5）：拍卖命令多路
//（get_auction_msg 串定案；子命令：查询/寄卖/竞价/取回族——
// rec_consign_args/rec_my_consign/rec_my_bid 记录协作）。
func (m *AuctionHouseModule) onCustomMsg(k *Kernel, self, from PersistID, args VarList) {
	op := args.String(0)
	switch op {
	case "consign": // 寄卖（rec_consign_args：ConfigID/起拍/一口价/数量）
		if o := k.GetGameObj(self); o != nil {
			if r := o.Record("rec_consign_args"); r != nil && r.Rows() < r.Def().MaxRows {
				r.AppendRow(VarList{args.String(1), int64(args.Int64(2)), int64(args.Int64(3)), args.Int(4)})
			}
		}
	case "bid": // 竞价（rec_my_bid：Uid/时间）
		if o := k.GetGameObj(self); o != nil {
			if r := o.Record("rec_my_bid"); r != nil && r.Rows() < r.Def().MaxRows {
				r.AppendRow(VarList{args.String(1), unixNow()})
			}
		}
	default: // 查询/取回等：上行拍卖服
	}
	if m.AuctionOut != nil {
		m.AuctionOut(self, op, args)
	}
	m.ops = append(m.ops, op)
}

// ItemTypeChildren 分类子树（Father 键）。
func (m *AuctionHouseModule) ItemTypeChildren(father string) []string {
	var out []string
	for id, row := range m.ItemTypes {
		if row["Father"] == father {
			out = append(out, id)
		}
	}
	return out
}

// NxRebuildAhRecs nx_rebuild_ah_recs 的 Go 等价（本地寄卖/竞价记录重建）。
func (m *AuctionHouseModule) NxRebuildAhRecs(k *Kernel, self PersistID) {}

// NxClearAhData nx_clear_ah_data 的 Go 等价（清拍卖记录）。
func (m *AuctionHouseModule) NxClearAhData(k *Kernel, self PersistID) {
	if o := k.GetGameObj(self); o != nil {
		for _, rec := range []string{"rec_consign_args", "rec_my_consign", "rec_my_bid", "rec_search_args"} {
			if r := o.Record(rec); r != nil {
				r.Clear()
			}
		}
	}
}

// Ops 命令流水（测试面）。
func (m *AuctionHouseModule) Ops() []string { return m.ops }

// ---- IncomeReduceModule ----

// IncomeReduceModule 复刻官方 incomereducemodule（Init @0x275A90，玩法表
// Init 序 118；符号 76）——收益递减。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnRecover",@0x275E30)
//	DeclareHeartBeat ("IncomeReduceModule::HB_ReduceReset",@0x276320——
//	    串定案：Level + sys_eventment_0002/0003）
//
// 记录（struct/rule/income_reduce.xml）：ReduceCountRec(3×1 事件/战场/
// 竞技场计数) + TomplimitRec(4×1 战场上限族)。
type IncomeReduceModule struct {
	BaseModule

	// notices 公告观测（sys_eventment_0002 递减生效/0003 重置）。
	notices []string
}

// Name 模块名。
func (m *IncomeReduceModule) Name() string { return "IncomeReduceModule" }

// Init 注册期（对齐官方 0x275A90）。
func (m *IncomeReduceModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.HeartBeat("IncomeReduceModule::HB_ReduceReset", m.hbReduceReset)
	return nil
}

// onRecover 官方 @0x275E30：计数表回载。
func (m *IncomeReduceModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}

// ApplyIncome 收益递减面：按 ReduceCountRec 计数档返回收益比例
//（3/6/9 档：100%/50%/25%/0——档位系数为 Go 缝合，官方经 Level 曲线）。
func (m *IncomeReduceModule) ApplyIncome(k *Kernel, self PersistID, col int32) float64 {
	o := k.GetGameObj(self)
	if o == nil {
		return 1
	}
	r := o.Record("ReduceCountRec")
	if r == nil || r.Rows() == 0 {
		return 1
	}
	n := r.Row(0).Int(int(col))
	switch {
	case n <= 3:
		return 1
	case n <= 6:
		return 0.5
	case n <= 9:
		return 0.25
	default:
		m.notices = append(m.notices, "sys_eventment_0002")
		return 0
	}
}

// CountIncome 收益事件计数（+1 后取比例）。
func (m *IncomeReduceModule) CountIncome(k *Kernel, self PersistID, col int32) float64 {
	o := k.GetGameObj(self)
	if o == nil {
		return 1
	}
	r := o.Record("ReduceCountRec")
	if r == nil {
		return 1
	}
	if r.Rows() == 0 {
		r.AppendRow(VarList{int32(0), int32(0), int32(0)})
	}
	row := r.Row(0)
	row[int(col)] = row.Int(int(col)) + 1
	r.ModifyRow(0, row)
	return m.ApplyIncome(k, self, col)
}

// HB_ReduceReset 官方 @0x276320：计数重置（周期面 + sys_eventment_0003）。
func (m *IncomeReduceModule) hbReduceReset(k *Kernel, self PersistID, beats int32) {
	if o := k.GetGameObj(self); o != nil {
		if r := o.Record("ReduceCountRec"); r != nil && r.Rows() > 0 && r.Row(0).Int(0) > 0 {
			r.ModifyRow(0, VarList{int32(0), int32(0), int32(0)})
			m.notices = append(m.notices, "sys_eventment_0003")
		}
	}
}

// Notices 公告流水（测试面）。
func (m *IncomeReduceModule) Notices() []string { return m.notices }
