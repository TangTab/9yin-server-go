package auction

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/proto"
)

// consignParams 按官方 19 参负载顺序拼寄售帧（[0]=reqid 后接 18 列行）。
func consignParams(reqid int64, uid string, itemType int32, price, fixed, overtime int64) *proto.VarList {
	v := &proto.VarList{}
	v.AddInt64(reqid)           // p0 reqid
	v.AddString(uid)            // p1 UniqueID
	v.AddString("cfg_10001")    // p2 ConfigID
	v.AddInt32(0)               // p3 SortType
	v.AddString("ITEMBLOB")     // p4 ItemString
	v.AddString("seller-uid-1") // p5 SellerUid
	v.AddInt64(1700000000)      // p6 ConsignTime
	v.AddInt64(fixed)           // p7 FixedPrice
	v.AddFloat32(0.15)          // p8 TransFeeCoe
	v.AddInt64(0)               // p9 ExtraI64
	v.AddInt64(overtime)        // p10 OverTime
	v.AddString("")             // p11 CurBiderUid
	v.AddInt64(price)           // p12 CurPrice
	v.AddInt32(StateConsigned)  // p13 State
	v.AddInt32(itemType)        // p14 ItemType
	v.AddWide("卖家甲")            // p15 SellerName
	v.AddInt32(30)              // p16 Level
	v.AddInt32(5)               // p17 Quality
	v.AddString("")             // p18 ExtraStr
	return v
}

func dispatch(t *testing.T, l *Logic, msgid int32, payload *proto.VarList) []Outbound {
	t.Helper()
	frame := &proto.VarList{}
	frame.AddInt32(msgid)
	frame.Values = append(frame.Values, payload.Values...)
	return l.Dispatch(7, 0xD, frame)
}

func TestConsignFlow(t *testing.T) {
	m := NewManager()
	l := NewLogic(m)
	m.itemTypes[10001] = BigTypeWeapon

	out := dispatch(t, l, MsgConsign, consignParams(42, "uid-A", 10001, 100, 500, time.Now().Add(time.Hour).Unix()))
	if len(out) != 1 || out[0].Target != 7 || out[0].Sub != 0xD {
		t.Fatalf("ack routing: %+v", out)
	}
	b := out[0].Body.Values
	if b[0].I64 != MsgConsignAck || b[1].I64 != 42 || b[2].Text != "ITEMBLOB" {
		t.Fatalf("ack7 = %+v", b)
	}
	if m.Main.GetItemNum() != 1 || m.Main.FindItem("uid-A") != 0 {
		t.Fatalf("main record state: num=%d find=%d", m.Main.GetItemNum(), m.Main.FindItem("uid-A"))
	}
	// 挂进 4 大类 × 6 视图
	for key := 1; key <= 6; key++ {
		if m.Sorted[BigTypeWeapon][key].GetItemNum() != 1 {
			t.Fatalf("weapon view key=%d empty", key)
		}
	}

	// 重复 uid → 静默
	if out := dispatch(t, l, MsgConsign, consignParams(43, "uid-A", 10001, 100, 500, 0)); out != nil {
		t.Fatalf("dup uid must be silent, got %+v", out)
	}
	// 参数类型坏 → 静默
	bad := &proto.VarList{}
	bad.AddInt64(1)
	for i := 0; i < 18; i++ {
		bad.AddString("x")
	}
	if out := dispatch(t, l, MsgConsign, bad); out != nil {
		t.Fatalf("bad args must be silent, got %+v", out)
	}
}

func TestBidAndFixedBid(t *testing.T) {
	m := NewManager()
	l := NewLogic(m)
	m.itemTypes[10001] = BigTypeArmor
	dispatch(t, l, MsgConsign, consignParams(1, "uid-B", 10001, 100, 500, time.Now().Add(time.Hour).Unix()))

	// 竞拍：ack9 = [9, uid, configid, bidder, old_bider, old_price]
	bid := &proto.VarList{}
	bid.AddString("uid-B")
	bid.AddString("bidder-1")
	bid.AddInt64(150)
	out := dispatch(t, l, MsgBid, bid)
	if len(out) != 1 {
		t.Fatalf("bid ack missing")
	}
	b := out[0].Body.Values
	if b[0].I64 != MsgBidAck || b[1].Text != "uid-B" || b[2].Text != "cfg_10001" ||
		b[3].Text != "bidder-1" || b[4].Text != "" || b[5].I64 != 100 {
		t.Fatalf("ack9 = %+v", b)
	}
	row := m.Main.FindItem("uid-B")
	if m.Main.Rows[row].State != StateBidded || m.Main.Rows[row].CurPrice != 150 ||
		m.Main.Rows[row].CurBiderUid != "bidder-1" {
		t.Fatalf("bid row = %+v", m.Main.Rows[row])
	}

	// 一口价：ack10 十元素，行即刻消失
	fb := &proto.VarList{}
	fb.AddString("uid-B")
	fb.AddString("buyer-9")
	out = dispatch(t, l, MsgFixedBid, fb)
	if len(out) != 1 {
		t.Fatalf("fixedbid ack missing")
	}
	b = out[0].Body.Values
	if len(b) != 10 || b[0].I64 != MsgFixedBidAck || b[4].Text != "bidder-1" ||
		b[5].I64 != 150 || b[6].Text != "buyer-9" || b[7].I64 != 500 {
		t.Fatalf("ack10 = %+v", b)
	}
	if m.Main.FindItem("uid-B") != -1 || m.Main.GetItemNum() != 0 {
		t.Fatalf("fixed bid must remove row")
	}
	for key := 1; key <= 6; key++ {
		if m.Sorted[BigTypeArmor][key].GetItemNum() != 0 {
			t.Fatalf("views must be empty after fixedbid")
		}
	}
}

func TestCancelConsign(t *testing.T) {
	m := NewManager()
	l := NewLogic(m)
	m.itemTypes[10001] = BigTypeConsume
	dispatch(t, l, MsgConsign, consignParams(1, "uid-C", 10001, 10, 50, time.Now().Add(time.Hour).Unix()))

	// 无此物品 → 静默
	ghost := &proto.VarList{}
	ghost.AddString("nope")
	if out := dispatch(t, l, MsgCancelConsign, ghost); out != nil {
		t.Fatalf("cancel ghost must be silent")
	}

	cc := &proto.VarList{}
	cc.AddString("uid-C")
	out := dispatch(t, l, MsgCancelConsign, cc)
	b := out[0].Body.Values
	if b[0].I64 != MsgCancelAck || b[1].Text != "uid-C" || b[2].Text != "cfg_10001" ||
		b[3].Text != "ITEMBLOB" || b[4].Text != "seller-uid-1" {
		t.Fatalf("ack8 = %+v", b)
	}
	if m.Main.GetItemNum() != 0 {
		t.Fatalf("cancel must remove row")
	}
}

func TestOverTimeSettlement(t *testing.T) {
	now := time.Unix(1700000000, 0)
	m := NewManager()
	l := NewLogic(m)
	l.now = func() time.Time { return now }
	m.itemTypes[10001] = BigTypeOther

	// 未到期 → 无输出
	dispatch(t, l, MsgConsign, consignParams(1, "uid-D", 10001, 10, 50, now.Add(time.Minute).Unix()))
	if out := l.CheckOverTime(); len(out) != 0 {
		t.Fatalf("not due yet")
	}

	// 到期无出价 → msg12 五元素广播 target=0
	m2 := NewManager()
	l2 := NewLogic(m2)
	l2.now = func() time.Time { return now }
	m2.itemTypes[10001] = BigTypeOther
	dispatch(t, l2, MsgConsign, consignParams(1, "uid-E", 10001, 10, 50, now.Unix()))
	out := l2.CheckOverTime()
	if len(out) != 1 || out[0].Target != 0 || out[0].Sub != 0 {
		t.Fatalf("unsold broadcast routing: %+v", out)
	}
	b := out[0].Body.Values
	if b[0].I64 != MsgUnsoldReturn || len(b) != 5 || b[1].Text != "uid-E" || b[4].Text != "ITEMBLOB" {
		t.Fatalf("unsold msg12 = %+v", b)
	}
	if m2.Main.GetItemNum() != 0 {
		t.Fatalf("overtime must remove row")
	}

	// 到期有出价 → msg11 八元素（含价/费/买家）
	m3 := NewManager()
	l3 := NewLogic(m3)
	l3.now = func() time.Time { return now }
	m3.itemTypes[10001] = BigTypeOther
	dispatch(t, l3, MsgConsign, consignParams(1, "uid-F", 10001, 10, 50, now.Unix()))
	bid := &proto.VarList{}
	bid.AddString("uid-F")
	bid.AddString("winner-1")
	bid.AddInt64(88)
	dispatch(t, l3, MsgBid, bid)
	out = l3.CheckOverTime()
	b = out[0].Body.Values
	if b[0].I64 != MsgBidSold || len(b) != 8 || b[5].I64 != 88 || b[7].Text != "winner-1" {
		t.Fatalf("sold msg11 = %+v", b)
	}
}

func TestSortedViewsOrdering(t *testing.T) {
	m := NewManager()
	l := NewLogic(m)
	m.itemTypes[10001] = BigTypeWeapon
	// 三行不同品质/等级/价
	dispatch(t, l, MsgConsign, consignParams(1, "q3", 10001, 30, 0, 300))
	dispatch(t, l, MsgConsign, consignParams(2, "q1", 10001, 10, 0, 100))
	dispatch(t, l, MsgConsign, consignParams(3, "q2", 10001, 20, 0, 200))

	items := m.Sorted[BigTypeWeapon][SortByCurPrice].Items()
	var got []string
	for _, r := range items {
		got = append(got, m.Main.Rows[r].UniqueID)
	}
	if got[0] != "q1" || got[1] != "q2" || got[2] != "q3" {
		t.Fatalf("curprice ascending = %v", got)
	}
	// 竞拍改价后重定位
	bid := &proto.VarList{}
	bid.AddString("q1")
	bid.AddString("b")
	bid.AddInt64(99)
	dispatch(t, l, MsgBid, bid)
	items = m.Sorted[BigTypeWeapon][SortByCurPrice].Items()
	if m.Main.Rows[items[2]].UniqueID != "q1" {
		t.Fatalf("re-sort after bid failed: %v", items)
	}

	// RebuildRecs 全量重建等价
	m.RebuildRecs()
	if m.Sorted[BigTypeWeapon][SortByCurPrice].GetItemNum() != 3 {
		t.Fatalf("rebuild lost rows")
	}
}

func TestLoadResourceItemTypeXML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ItemType.xml")
	xml := `<?xml version="1.0" encoding="gb2312"?>
<Object>
	<Property ID="10001" Name="剑" BigType="1" Father="10000"/>
	<Property ID="20001" Name="布甲" BigType="2" Father="11000"/>
	<Property ID="30001" Name="药剂" BigType="3" Father="16000"/>
</Object>`
	if err := os.WriteFile(path, []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	if err := m.LoadResource(path); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int32]int32{10001: 1, 20001: 2, 30001: 3} {
		if got := m.GetBigType(id); got != want {
			t.Fatalf("GetBigType(%d)=%d want %d", id, got, want)
		}
	}
	if m.GetBigType(99999) != BigTypeOther {
		t.Fatalf("unknown type must fall back to Other")
	}
	// 文件缺失仅报错不 panic
	if err := m.LoadResource(filepath.Join(dir, "missing.xml")); err == nil {
		t.Fatalf("missing file must error")
	}
}

func TestClearDataGM(t *testing.T) {
	m := NewManager()
	l := NewLogic(m)
	m.itemTypes[10001] = BigTypeWeapon
	dispatch(t, l, MsgConsign, consignParams(1, "uid-G", 10001, 10, 50, 0))
	if out := dispatch(t, l, MsgClearData, &proto.VarList{}); out != nil {
		t.Fatalf("clear must be silent")
	}
	if m.Main.GetItemNum() != 0 {
		t.Fatalf("clear must empty main")
	}
}
