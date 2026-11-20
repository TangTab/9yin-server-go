package fsgame

import (
	"path/filepath"
	"testing"
)

// assemble6 装配六批三十四模块。
func assemble6(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))
	nb := mustRead(t, filepath.Join(td, "struct", "container", "npcdropbox.xml"))

	k := NewKernel()
	k.ResourcePath = td
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	if err := k.LoadStruct("player", px); err != nil {
		t.Fatalf("LoadStruct player: %v", err)
	}
	if err := k.LoadStruct("NpcDropBox", nb); err != nil {
		t.Fatalf("LoadStruct NpcDropBox: %v", err)
	}
	if err := k.LoadStruct("player", ex); err != nil {
		t.Fatalf("LoadStruct player(exp): %v", err)
	}
	var r Registry
	for _, reg := range []func(*Registry) error{
		RegisterFirstBatch, RegisterSecondBatch, RegisterThirdBatch,
		RegisterFourthBatch, RegisterFifthBatch, RegisterSixthBatch,
	} {
		if err := reg(&r); err != nil {
			t.Fatal(err)
		}
	}
	mods, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return k, mods
}

// TestSixthBatchAssemble 六期装配序 + 商店/拍卖真档装载。
func TestSixthBatchAssemble(t *testing.T) {
	k, mods := assemble6(t)
	want := []string{"AuctionHouseModule", "ShopModule", "IncomeReduceModule", "VipModule", "GiftModule"}
	var got []string
	for _, m := range mods {
		switch m.Name() {
		case "AuctionHouseModule", "ShopModule", "IncomeReduceModule", "VipModule", "GiftModule":
			got = append(got, m.Name())
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	sm := module2[*ShopModule](t, k, "ShopModule")
	if len(sm.Shops) == 0 || len(sm.Items) == 0 {
		t.Fatalf("商店真档装载异常: 店 %d / 商品 %d", len(sm.Shops), len(sm.Items))
	}
	// 真档抽查：PVP 商店含 SellID；伏特加 BuyPrice=-1 非卖。
	if shop := sm.Shops["Shop_PVP_ZQ_ZB1"]; shop == nil || shop["SellID"] == "" {
		t.Fatalf("Shop.xml 真档异常: %v", sm.Shops["Shop_PVP_ZQ_ZB1"])
	}
	if it := sm.Items["Task_00166A"]; it == nil || atoi32(it["BuyPrice"]) != -1 {
		t.Fatalf("ShopItem.xml 真档异常: %v", sm.Items["Task_00166A"])
	}
	ah := module2[*AuctionHouseModule](t, k, "AuctionHouseModule")
	if len(ah.ItemTypes) == 0 || len(ah.ItemTypeChildren("0")) == 0 {
		t.Fatalf("ItemType.xml 分类树异常: %d", len(ah.ItemTypes))
	}
}

// TestShopBuy 商店闭环：在售门（SellID 含商品）→ 非卖拒（BuyPrice=-1）→
// 限购（Maxamount）→ 货币扣减 + 到货。
func TestShopBuy(t *testing.T) {
	k, _ := assemble6(t)
	sm := module2[*ShopModule](t, k, "ShopModule")
	p, _ := k.CreateObject("player")

	// 注入可售商品（真档店 + 合成商品行：价 100/限购 5）。
	sm.Items["Drug_Heal_001"] = map[string]string{
		"ID": "Drug_Heal_001", "ItemID": "Drug_Heal_001", "Maxamount": "5",
		"CurrencyType": "1", "BuyPrice": "100", "isbind": "0", "SellNum": "-1", "CoolTime": "-1",
	}
	shop := sm.Shops["Shop_PVP_ZQ_ZB1"]
	shop["SellID"] = shop["SellID"] + ",Drug_Heal_001"

	var cost int64
	var bought string
	var boughtN int32
	sm.CostOut = func(pl PersistID, ct int32, amount int64) bool {
		cost += amount
		return true
	}
	sm.BuyOut = func(pl PersistID, itemID string, count int32) {
		bought, boughtN = itemID, count
	}
	// 限购 5：要 10 只给 5。
	if !sm.Buy(k, p, "Shop_PVP_ZQ_ZB1", "Drug_Heal_001", 10) {
		t.Fatal("购买失败")
	}
	if bought != "Drug_Heal_001" || boughtN != 5 || cost != 500 {
		t.Fatalf("购买账面 %s×%d cost=%d", bought, boughtN, cost)
	}
	// 非卖品拒（真档 Task_00166A BuyPrice=-1，且不在 SellID——先入表）。
	shop["SellID"] = shop["SellID"] + ",Task_00166A"
	if sm.Buy(k, p, "Shop_PVP_ZQ_ZB1", "Task_00166A", 1) {
		t.Fatal("非卖品（BuyPrice=-1）未被拒")
	}
	// 不在店表拒。
	if sm.Buy(k, p, "Shop_PVP_ZQ_ZB1", "Not_In_Shop", 1) {
		t.Fatal("店外商品未被拒")
	}
	// 货币不足拒。
	sm.CostOut = func(pl PersistID, ct int32, amount int64) bool { return false }
	if sm.Buy(k, p, "Shop_PVP_ZQ_ZB1", "Drug_Heal_001", 1) {
		t.Fatal("货币不足未被拒")
	}
	// cust 70 子命令 0 购买路由。
	sm.CostOut = func(pl PersistID, ct int32, amount int64) bool { return true }
	if !k.DispatchCommand("player", 70, p, NullPersistID,
		VarList{int32(0), "Shop_PVP_ZQ_ZB1", "Drug_Heal_001", int32(1)}) {
		t.Fatal("购物命令 70 未命中")
	}
	if len(sm.Sold()) != 2 {
		t.Fatalf("售卖流水 %v", sm.Sold())
	}
}

// TestVipAndGift VIP 礼包 + 充值礼包闭环。
func TestVipAndGift(t *testing.T) {
	k, _ := assemble6(t)
	vm := module2[*VipModule](t, k, "VipModule")
	gm := module2[*GiftModule](t, k, "GiftModule")
	p, _ := k.CreateObject("player")
	k.RunEvent("player", "OnRecover", p, NullPersistID, nil)

	// 非 VIP 领礼包拒。
	k.DispatchCommand("player", 65, p, NullPersistID, VarList{})
	if len(vm.Ops()) != 0 {
		t.Fatal("非 VIP 可领礼包")
	}
	// VIP 领礼包：每日一次（VipGiftTime 天序号）。
	_ = k.SetInt(p, "IsVip", 1)
	var gifted int
	vm.GiftOut = func(pl PersistID, giftID string, count int32) { gifted++ }
	k.DispatchCommand("player", 65, p, NullPersistID, VarList{})
	if gifted != 1 {
		t.Fatal("VIP 礼包未发放")
	}
	k.DispatchCommand("player", 65, p, NullPersistID, VarList{})
	if gifted != 1 {
		t.Fatal("VIP 礼包每日一次未生效")
	}
	// 充值礼包：scene OnGetGift（sendmail=0 直发 / =1 入邮件待收表）。
	var direct int
	gm.GiftOut = func(pl PersistID, giftID, giftName string, count int32) { direct++ }
	scene, _ := k.CreateObject("player") // 占位 scene 对象
	k.RunEvent("scene", "OnGetGift", scene, p, VarList{int32(1), "GIFT_01", "新手礼包", int32(2), int32(0)})
	if direct != 1 {
		t.Fatal("直发礼包未出口")
	}
	k.RunEvent("scene", "OnGetGift", scene, p, VarList{int32(1), "GIFT_02", "邮件礼包", int32(1), int32(1)})
	if r := k.GetGameObj(p).Record("GiftWaitPostGetRec"); r == nil || r.Rows() != 1 || r.Row(0).String(1) != "邮件礼包" {
		t.Fatalf("邮件待收表异常: %v", r)
	}
	if len(gm.Grants()) != 2 {
		t.Fatalf("发放流水 %v", gm.Grants())
	}
}

// TestAuctionAndIncome 拍卖桥 + 收益递减闭环。
func TestAuctionAndIncome(t *testing.T) {
	k, _ := assemble6(t)
	ah := module2[*AuctionHouseModule](t, k, "AuctionHouseModule")
	ir := module2[*IncomeReduceModule](t, k, "IncomeReduceModule")
	p, _ := k.CreateObject("player")
	k.RunEvent("player", "OnRecover", p, NullPersistID, nil)

	// 拍卖：寄卖/竞价记录 + 上行出口。
	var uplinked string
	ah.AuctionOut = func(pl PersistID, op string, args VarList) { uplinked = op }
	k.DispatchCommand("player", 197, p, NullPersistID, VarList{"consign", "Sword_001", int64(100), int64(200), int32(1)})
	if uplinked != "consign" {
		t.Fatalf("拍卖上行 %q", uplinked)
	}
	if r := k.GetGameObj(p).Record("rec_consign_args"); r == nil || r.Rows() != 1 || r.Row(0).String(0) != "Sword_001" {
		t.Fatalf("寄卖记录异常: %v", r)
	}
	k.DispatchCommand("player", 197, p, NullPersistID, VarList{"bid", "UID-9"})
	if r := k.GetGameObj(p).Record("rec_my_bid"); r == nil || r.Rows() != 1 {
		t.Fatal("竞价记录缺失")
	}
	ah.NxClearAhData(k, p)
	if r := k.GetGameObj(p).Record("rec_consign_args"); r.Rows() != 0 {
		t.Fatal("nx_clear_ah_data 未清")
	}

	// 收益递减：3 次全收益 → 递减档 → 超限零收益（sys_eventment_0002）→
	// HB 重置（0003）。
	var ratio float64
	for i := 0; i < 3; i++ {
		ratio = ir.CountIncome(k, p, 0)
	}
	if ratio != 1 {
		t.Fatalf("前 3 次收益比例 %v", ratio)
	}
	ratio = ir.CountIncome(k, p, 0) // 第 4 次
	if ratio != 0.5 {
		t.Fatalf("第 4 次收益比例 %v 期望 0.5", ratio)
	}
	for i := 0; i < 6; i++ {
		ratio = ir.CountIncome(k, p, 0) // 5..10
	}
	if ratio != 0 {
		t.Fatalf("超限收益比例 %v 期望 0", ratio)
	}
	if !hasTpl(ir.Notices(), "sys_eventment_0002") {
		t.Fatal("递减生效未通知 sys_eventment_0002")
	}
	k.Bus().TickHeartBeats(k, 1)
	if r := k.GetGameObj(p).Record("ReduceCountRec"); r == nil || r.Row(0).Int(0) != 0 {
		t.Fatal("HB_ReduceReset 未清计数")
	}
	if !hasTpl(ir.Notices(), "sys_eventment_0003") {
		t.Fatal("重置未通知 sys_eventment_0003")
	}
}
