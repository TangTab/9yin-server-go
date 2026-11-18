package auction

import (
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

// 内网消息号（元素 [0]=int 消息号；game↔auction 双向共用一号段）。
const (
	MsgConsign       = 1  // game→auction 寄售（19 参负载）
	MsgCancelConsign = 2  // game→auction 取消寄售（1 参）
	MsgBid           = 3  // game→auction 竞拍出价（3 参）
	MsgFixedBid      = 4  // game→auction 一口价（2 参）
	MsgRebuildRecs   = 5  // game→auction 重建排序视图（GM，0 参）
	MsgClearData     = 6  // game→auction 清空数据（GM，0 参）
	MsgConsignAck    = 7  // auction→game 寄售回执
	MsgCancelAck     = 8  // auction→game 取消回执
	MsgBidAck        = 9  // auction→game 竞拍回执
	MsgFixedBidAck   = 10 // auction→game 一口价成交回执
	MsgBidSold       = 11 // auction→game 竞拍到期成交（超时广播）
	MsgUnsoldReturn  = 12 // auction→game 流拍退回（超时广播；FsGame 表 TimeOutUnSoldBack）
)

// CheckResult 校验错误码（官方 Check*Args 返回值；0=通过）。
// 官方行为：非 0 一律静默丢帧（不回错误帧——前端校验在 FsGame 侧）。
const (
	CheckOK       = 0
	CheckBadArgs  = 1  // 参数个数/类型不符
	CheckNoName   = 9  // 空字符串（uid/出价人/买家）
	CheckNoItem   = 5  // FindItem 无此物品
	CheckDupItem  = 11 // 唯一号已存在（0xB）
	CheckRecFull  = 22 // 全表 ≥ 100000（0x16）
	CheckTypeFull = 23 // 单大类 ≥ 25000（0x17）
)

// Outbound 一条 auction→game 出站消息
// （官方 kernel->SendAuctionMessage(target, sub, msg) = vt+0x20 三参）。
type Outbound struct {
	Target int32 // 目标（0 = 广播全部 game 服；官方超时结算即广播）
	Sub    int32 // param2 回传（官方 OnAuctionMessage 的 int2 原样透传）
	Body   *proto.VarList
}

// Logic 对应官方 AuctionLogic 单例：消息分发 + 四业务 + 超时结算。
type Logic struct {
	Mgr *Manager
	now func() time.Time // 可注入时钟（测试用）
}

// NewLogic 建逻辑器。
func NewLogic(m *Manager) *Logic {
	return &Logic{Mgr: m, now: time.Now}
}

// nowUnix 当前秒（官方 util_get_time_64 口径与 OverTime 比较用）。
func (l *Logic) nowUnix() int64 { return l.now().Unix() }

// Dispatch 处理一条入站消息（params 元素 [0]=int 消息号）。
// 返回待发回执（可能多条；校验失败静默返回 nil，官方语义）。
func (l *Logic) Dispatch(target, sub int32, params *proto.VarList) []Outbound {
	if len(params.Values) == 0 {
		return nil
	}
	msgid := int32(params.Values[0].I64)
	rest := &proto.VarList{Values: params.Values[1:]}
	switch msgid {
	case MsgConsign:
		return l.msgConsign(target, sub, rest)
	case MsgCancelConsign:
		return l.msgCancelConsign(target, sub, rest)
	case MsgBid:
		return l.msgBid(target, sub, rest)
	case MsgFixedBid:
		return l.msgFixedBid(target, sub, rest)
	case MsgRebuildRecs:
		l.Mgr.RebuildRecs()
		return nil
	case MsgClearData:
		l.Mgr.ClearData()
		return nil
	}
	return nil
}

// --- 取值助手（官方 IntVal/StringVal/Int64Val/FloatVal 宽容语义） ---

func vInt(v *proto.VarList, i int) int32 {
	if i >= len(v.Values) {
		return 0
	}
	return int32(v.Values[i].I64)
}

func vInt64(v *proto.VarList, i int) int64 {
	if i >= len(v.Values) {
		return 0
	}
	return v.Values[i].I64
}

func vFloat(v *proto.VarList, i int) float32 {
	if i >= len(v.Values) {
		return 0
	}
	return float32(v.Values[i].F64)
}

func vStr(v *proto.VarList, i int) string {
	if i >= len(v.Values) {
		return ""
	}
	return v.Values[i].Text
}

// --- 校验（Check*Args 等价翻译） ---

// checkConsign 官方 CheckConsignArgs：19 参类型闸 + 卖家名非空 +
// 唯一号不重复 + 全表/单大类容量。
func (l *Logic) checkConsign(p *proto.VarList) int {
	if len(p.Values) != 19 {
		return CheckBadArgs
	}
	want := []byte{proto.VarInt64, proto.VarString, proto.VarString, proto.VarInt32,
		proto.VarString, proto.VarString, proto.VarInt64, proto.VarInt64,
		proto.VarFloat, proto.VarInt64, proto.VarInt64, proto.VarString,
		proto.VarInt64, proto.VarInt32, proto.VarInt32, proto.VarWide,
		proto.VarInt32, proto.VarInt32, proto.VarString}
	for i, t := range want {
		if p.Values[i].Type != t {
			return CheckBadArgs
		}
	}
	if vStr(p, 1) == "" { // strlen(卖家名) == 0 → 9
		return CheckNoName
	}
	if l.Mgr.Main.FindItem(vStr(p, 1)) != -1 { // 唯一号重复 → 0xB
		return CheckDupItem
	}
	if l.Mgr.GetItemNum() >= MaxTotalRows { // ≥0x186A0 → 0x16
		return CheckRecFull
	}
	sr := l.Mgr.GetSortedRecord(l.Mgr.GetBigType(vInt(p, 13)), 1)
	if sr != nil && sr.GetItemNum() >= MaxPerBig { // ≥0x61A8 → 0x17
		return CheckTypeFull
	}
	return CheckOK
}

func (l *Logic) checkCancelConsign(p *proto.VarList) int {
	if len(p.Values) != 1 || p.Values[0].Type != proto.VarString {
		return CheckBadArgs
	}
	if vStr(p, 0) == "" {
		return CheckNoName
	}
	if l.Mgr.Main.FindItem(vStr(p, 0)) == -1 {
		return CheckNoItem
	}
	return CheckOK
}

func (l *Logic) checkBid(p *proto.VarList) int {
	if len(p.Values) != 3 || p.Values[0].Type != proto.VarString ||
		p.Values[1].Type != proto.VarString || p.Values[2].Type != proto.VarInt64 {
		return CheckBadArgs
	}
	if vStr(p, 0) == "" || vStr(p, 1) == "" {
		return CheckNoName
	}
	if l.Mgr.Main.FindItem(vStr(p, 0)) == -1 {
		return CheckNoItem
	}
	return CheckOK
}

func (l *Logic) checkFixedBid(p *proto.VarList) int {
	if len(p.Values) != 2 || p.Values[0].Type != proto.VarString ||
		p.Values[1].Type != proto.VarString {
		return CheckBadArgs
	}
	if vStr(p, 0) == "" || vStr(p, 1) == "" {
		return CheckNoName
	}
	if l.Mgr.Main.FindItem(vStr(p, 0)) == -1 {
		return CheckNoItem
	}
	return CheckOK
}

// --- 消息处理（MsgHandler* 等价翻译） ---

// msgConsign 寄售：校验 → DoConsign（入行+挂视图）→ 回 [7, consign_id, itemstring]。
func (l *Logic) msgConsign(target, sub int32, p *proto.VarList) []Outbound {
	if code := l.checkConsign(p); code != CheckOK {
		return nil
	}
	l.doConsign(p)
	ack := &proto.VarList{}
	ack.AddInt32(MsgConsignAck)
	ack.AddInt64(vInt64(p, 0)) // consign_id（FsGame gen_item_unique_id 生成，原样回传）
	ack.AddString(vStr(p, 4))  // itemstring（官方 StringVal(4)）
	return []Outbound{{Target: target, Sub: sub, Body: ack}}
}

// doConsign 官方 DoConsign：AddItem(整负载除 reqid) + 挂 1..6 视图。
func (l *Logic) doConsign(p *proto.VarList) int32 {
	r := Row{
		UniqueID:    vStr(p, 1),
		ConfigID:    vStr(p, 2),
		SortType:    vInt(p, 3),
		ItemString:  vStr(p, 4),
		SellerUid:   vStr(p, 5),
		StartPrice:  vInt64(p, 6),
		FixedPrice:  vInt64(p, 7),
		TransFeeCoe: vFloat(p, 8),
		KeepingFee:  vInt64(p, 9),
		OverTime:    vInt64(p, 10),
		CurBiderUid: vStr(p, 11),
		CurPrice:    vInt64(p, 12),
		State:       vInt(p, 13),
		ItemType:    vInt(p, 14),
		SellerName:  vStr(p, 15),
		Level:       vInt(p, 16),
		Quality:     vInt(p, 17),
		ExtraStr:    vStr(p, 18),
	}
	row := l.Mgr.Main.AddItem(r)
	if row < 0 {
		return -1
	}
	l.Mgr.OnItemAdded(row)
	return row
}

// msgCancelConsign 取消：校验 → DoCancelConsign → 回
// [8, uid, configid, itemstring, selleruid]。
func (l *Logic) msgCancelConsign(target, sub int32, p *proto.VarList) []Outbound {
	if code := l.checkCancelConsign(p); code != CheckOK {
		return nil
	}
	uid := vStr(p, 0)
	row := l.Mgr.Main.FindItem(uid)
	r := l.Mgr.Main.Rows[row]
	l.doCancelConsign(uid)
	ack := &proto.VarList{}
	ack.AddInt32(MsgCancelAck)
	ack.AddString(uid)
	ack.AddString(r.ConfigID)
	ack.AddString(r.ItemString)
	ack.AddString(r.SellerUid)
	return []Outbound{{Target: target, Sub: sub, Body: ack}}
}

// doCancelConsign 官方 DoCancelConsign：记 ItemType → RemoveItem → 摘视图。
func (l *Logic) doCancelConsign(uid string) {
	row := l.Mgr.Main.FindItem(uid)
	if row == -1 {
		return
	}
	it := l.Mgr.Main.Rows[row].ItemType
	l.Mgr.Main.RemoveItem(row)
	l.Mgr.OnItemRemoved(it, row)
}

// msgBid 竞拍：校验 → 读旧出价人/旧价 → DoBid → 回
// [9, uid, configid, bidder, old_bider, old_price]（官方 DoBid 前取值）。
func (l *Logic) msgBid(target, sub int32, p *proto.VarList) []Outbound {
	if code := l.checkBid(p); code != CheckOK {
		return nil
	}
	uid, bidder, price := vStr(p, 0), vStr(p, 1), vInt64(p, 2)
	row := l.Mgr.Main.FindItem(uid)
	oldBider := l.Mgr.Main.Rows[row].CurBiderUid
	oldPrice := l.Mgr.Main.Rows[row].CurPrice
	configID := l.Mgr.Main.Rows[row].ConfigID
	l.doBid(uid, bidder, price)
	ack := &proto.VarList{}
	ack.AddInt32(MsgBidAck)
	ack.AddString(uid)
	ack.AddString(configID)
	ack.AddString(bidder)
	ack.AddString(oldBider)
	ack.AddInt64(oldPrice)
	return []Outbound{{Target: target, Sub: sub, Body: ack}}
}

// doBid 官方 DoBid：SetPrice + SetCurBider + SetState(2) + 视图重定位。
func (l *Logic) doBid(uid, bidder string, price int64) {
	row := l.Mgr.Main.FindItem(uid)
	if row == -1 {
		return
	}
	r := &l.Mgr.Main.Rows[row]
	r.CurPrice = price
	r.CurBiderUid = bidder
	r.State = StateBidded
	l.Mgr.OnItemUpdated(r.ItemType, row)
}

// msgFixedBid 一口价：校验 → 读旧值/成交值 → DoFixedBid → 回
// [10, uid, configid, itemstring, old_bider, old_price, buyer, fixedprice,
// selleruid, feecoe]。
func (l *Logic) msgFixedBid(target, sub int32, p *proto.VarList) []Outbound {
	if code := l.checkFixedBid(p); code != CheckOK {
		return nil
	}
	uid, buyer := vStr(p, 0), vStr(p, 1)
	row := l.Mgr.Main.FindItem(uid)
	r := l.Mgr.Main.Rows[row]
	oldBider, oldPrice := r.CurBiderUid, r.CurPrice
	configID, itemString := r.ConfigID, r.ItemString
	fixedPrice, feeCoe := r.FixedPrice, r.TransFeeCoe
	sellerUid := r.SellerUid
	l.doFixedBid(uid, buyer)
	ack := &proto.VarList{}
	ack.AddInt32(MsgFixedBidAck)
	ack.AddString(uid)
	ack.AddString(configID)
	ack.AddString(itemString)
	ack.AddString(oldBider)
	ack.AddInt64(oldPrice)
	ack.AddString(buyer)
	ack.AddInt64(fixedPrice)
	ack.AddString(sellerUid)
	ack.AddFloat32(feeCoe)
	return []Outbound{{Target: target, Sub: sub, Body: ack}}
}

// doFixedBid 官方 DoFixedBid：SetPrice(FixedPrice)+SetCurBider+SetState(12)
// + 立即 RemoveItem + 摘视图。
func (l *Logic) doFixedBid(uid, buyer string) {
	row := l.Mgr.Main.FindItem(uid)
	if row == -1 {
		return
	}
	r := &l.Mgr.Main.Rows[row]
	r.CurPrice = r.FixedPrice
	r.CurBiderUid = buyer
	r.State = StateFixedSold
	it := r.ItemType
	l.Mgr.Main.RemoveItem(row)
	l.Mgr.OnItemRemoved(it, row)
}

// CheckOverTime 超时扫描（官方 H_CheckOverTime：倒序遍历，now ≥ OverTime
// 即结算）。返回待广播消息（target=0 官方语义）。
//
//	msg 11（有出价成交）：[11, uid, configid, selleruid, itemstring,
//	                       curprice, feecoe, bideruid]
//	msg 12（流拍退回）：  [12, uid, configid, selleruid, itemstring]
//	（分支选择值 = 10+(State==2)，成交分支实发 11、流拍分支实发 12）
func (l *Logic) CheckOverTime() []Outbound {
	var out []Outbound
	now := l.nowUnix()
	m := l.Mgr.Main
	for row := int32(len(m.Rows) - 1); row >= 0; row-- {
		if m.Rows[row].UniqueID == "" {
			continue
		}
		if now < m.Rows[row].OverTime {
			continue
		}
		out = append(out, l.processOverTimeItem(row)...)
	}
	return out
}

// processOverTimeItem 官方 ProcessOverTimeItem：
// 分支选择 10+(State==2)：==11 成交分支多带 curprice/feecoe/bider 三列，
// 否则流拍分支（实发 msg 12 五元素）；取数 → RemoveItem → 摘视图 → 广播。
func (l *Logic) processOverTimeItem(row int32) []Outbound {
	m := l.Mgr.Main
	r := m.Rows[row]
	sold := r.State == StateBidded
	msgID := int32(MsgUnsoldReturn)
	if sold {
		msgID = MsgBidSold
	}
	it := r.ItemType
	m.RemoveItem(row)
	l.Mgr.OnItemRemoved(it, row)

	ack := &proto.VarList{}
	ack.AddInt32(msgID)
	ack.AddString(r.UniqueID)
	ack.AddString(r.ConfigID)
	ack.AddString(r.SellerUid)
	ack.AddString(r.ItemString)
	if sold {
		ack.AddInt64(r.CurPrice)
		ack.AddFloat32(r.TransFeeCoe)
		ack.AddString(r.CurBiderUid)
	}
	return []Outbound{{Target: 0, Sub: 0, Body: ack}}
}
