package main

import (
	"encoding/binary"
	"strings"
	"testing"
)

// P3 第一批：exchange 商店（msg70 type-3）+ 新手选择（972）+ 门派
//（973/997）。fork 证据见 docs/fork-p3-exchange-school.md。

const testExchangeShopINI = `[Shop_ex_01]
Type=1
0=gear_test,1,3,0,0,1,0,13093,543,,
1=potion_test,1,3,0,0,1,1,13094,0,,
`

func stubExchangeShops(t *testing.T) {
	t.Helper()
	stubShopCatalog(t, testExchangeShopINI)
	stubExchangeCatalog(t, `[13093]
Item=pingzheng001,10

[13094]
Prop=CapitalType2,50
`)
}

func exchangeBuyMsg(pos int32) clientCustomMessage {
	return buyMsg("Shop_ex_01", 0, pos, 1)
}

func TestShopCatalogParsesExchangeColumns(t *testing.T) {
	stubShopCatalog(t, testExchangeShopINI)
	items, _, _, err := loadShopCatalogSection(defaultShopINIPath, "Shop_ex_01")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items=%d", len(items))
	}
	if items[0].exchangeID != 13093 || items[0].conditionID != 543 {
		t.Fatalf("row0 exchange cols=%d/%d, want 13093/543", items[0].exchangeID, items[0].conditionID)
	}
	if items[1].exchangeID != 13094 || items[1].conditionID != 0 {
		t.Fatalf("row1 exchange cols=%d/%d, want 13094/0", items[1].exchangeID, items[1].conditionID)
	}
}

func TestOpenShopMaterializesExchangeRowsWithoutError(t *testing.T) {
	stubShopCatalog(t, testExchangeShopINI)
	conn := &captureMessageConnection{}
	lifecycle := &sceneLifecycle{conn: conn}
	if err := lifecycle.openShopLocked("Shop_ex_01"); err != nil {
		t.Fatalf("open exchange shop: %v", err)
	}
	frames := conn.Frames()
	if len(frames) != 3 {
		t.Fatalf("frames=%d, want 1 create + 2 rows", len(frames))
	}
	if frames[0][0] != 0x15 {
		t.Fatalf("container=%#x", frames[0][0])
	}
	for _, row := range frames[1:] {
		if row[0] != 0x18 {
			t.Fatalf("row=%#x", row[0])
		}
	}
}

func TestShopBuyExchangeConsumesItemsAcrossRows(t *testing.T) {
	stubExchangeShops(t)
	player := playerWithBag(
		bagRow{slot: 1, configID: "pingzheng001", itemType: 2101, amount: 5, viewID: 1},
		bagRow{slot: 2, configID: "pingzheng001", itemType: 2101, amount: 7, viewID: 1},
		bagRow{slot: 3, configID: "gear_test", itemType: 2101, amount: 2, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleShopBuyCustom(h, exchangeBuyMsg(1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// 凭证 10 件：slot1 清空（5）+ slot2 剩 2（再扣 5）。
	if bagHas(player, "pingzheng001", 1) != true {
		t.Fatal("pingzheng row should remain")
	}
	if got := bagAmountOf(t, player, "pingzheng001", 1); got != 2 {
		t.Fatalf("remaining凭证=%d, want 2", got)
	}
	// slot1 行应已删除。
	player.mu.Lock()
	slots := map[int32]bool{}
	for _, row := range player.bag {
		slots[row.slot] = true
	}
	player.mu.Unlock()
	if slots[1] {
		t.Fatal("slot 1 should be emptied and removed")
	}
	if got := bagAmountOf(t, player, "gear_test", 1); got != 3 {
		t.Fatalf("gear_test=%d, want 3 (stacked +1)", got)
	}
	// 无 Prop 计价 → 不发货币帧；帧应为纯行同步（删行×2 + 重推×3 + 叠堆×3）。
	for _, frame := range conn.Frames() {
		if frame[0] == 0x10 {
			t.Fatalf("unexpected currency frame %x", frame)
		}
	}
}

func TestShopBuyExchangeChargesTrackedPropCurrency(t *testing.T) {
	stubExchangeShops(t)
	player := playerWithBag(
		bagRow{slot: 1, configID: "potion_test", itemType: 2101, amount: 1, viewID: 1},
	)
	player.addSilverCard(100)
	player.addSilver(999)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	// 13094：Prop=CapitalType2,50 → 银卡 100→50；碎银不动。
	if handled, err := handleShopBuyCustom(h, exchangeBuyMsg(2)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.silverCardBalance(); got != 50 {
		t.Fatalf("silverCard=%d, want 50", got)
	}
	if got := player.silverBalance(); got != 999 {
		t.Fatalf("silver moved on exchange: %d", got)
	}
	if got := bagAmountOf(t, player, "potion_test", 1); got != 2 {
		t.Fatalf("potion=%d, want 2", got)
	}
	// 叠堆行 3 帧 + 三币 0x10 三帧。
	frames := conn.Frames()
	if len(frames) != 6 {
		t.Fatalf("frames=%d", len(frames))
	}
	for index := 3; index <= 5; index++ {
		if frames[index][0] != 0x10 {
			t.Fatalf("currency frame %d=%#x", index, frames[index][0])
		}
	}
	if got := binary.LittleEndian.Uint16(frames[5][12:]); got != 418 {
		t.Fatalf("card frame idx=%d, want 418", got)
	}
}

func TestShopBuyExchangeRefusals(t *testing.T) {
	stubExchangeShops(t)
	// 凭证不足。
	player := playerWithBag(
		bagRow{slot: 1, configID: "pingzheng001", itemType: 2101, amount: 9, viewID: 1},
		bagRow{slot: 2, configID: "gear_test", itemType: 2101, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)
	if handled, err := handleShopBuyCustom(h, exchangeBuyMsg(1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagAmountOf(t, player, "pingzheng001", 1); got != 9 {
		t.Fatalf("insufficient: items moved: %d", got)
	}

	// 未跟踪属性货币。
	stubExchangeCatalog(t, `[13093]
Prop=SchoolDanceTotalScore,20
`)
	player2 := playerWithBag(bagRow{slot: 1, configID: "gear_test", itemType: 2101, amount: 1, viewID: 1})
	conn2 := &captureMessageConnection{}
	h2 := newTestItemSession(conn2, player2, nil)
	if handled, err := handleShopBuyCustom(h2, exchangeBuyMsg(1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if frames := conn2.Frames(); len(frames) != 0 {
		t.Fatalf("untracked currency emitted %d frames", len(frames))
	}

	// 银卡余额不足。
	stubExchangeCatalog(t, `[13094]
Prop=CapitalType2,50
`)
	player3 := playerWithBag(bagRow{slot: 1, configID: "potion_test", itemType: 2101, amount: 1, viewID: 1})
	player3.addSilverCard(10)
	conn3 := &captureMessageConnection{}
	h3 := newTestItemSession(conn3, player3, nil)
	if handled, err := handleShopBuyCustom(h3, exchangeBuyMsg(2)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player3.silverCardBalance(); got != 10 {
		t.Fatalf("card moved on reject: %d", got)
	}

	// 无兑换条目（exchangeID 指向缺失节）。
	stubExchangeCatalog(t, `[42]
Item=other,1
`)
	player4 := playerWithBag(bagRow{slot: 1, configID: "gear_test", itemType: 2101, amount: 1, viewID: 1})
	conn4 := &captureMessageConnection{}
	h4 := newTestItemSession(conn4, player4, nil)
	if handled, err := handleShopBuyCustom(h4, exchangeBuyMsg(1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagAmountOf(t, player4, "gear_test", 1); got != 1 {
		t.Fatalf("missing entry moved rows: %d", got)
	}
}

func TestFreshManChoiceHandledWithoutFrames(t *testing.T) {
	freshMsg := func(choice int32) clientCustomMessage {
		return clientCustomMessage{Values: []clientCustomValue{
			{Type: 2, Int32: clientCustomFreshManChoice},
			{Type: 2, Int32: choice},
		}}
	}
	if label := freshmanChoiceLabel(5); label != "老手(跳过引导)" {
		t.Fatalf("label(5)=%q", label)
	}
	if label := freshmanChoiceLabel(0); label != "新手(走引导)" {
		t.Fatalf("label(0)=%q", label)
	}
	if handled, err := handleFreshManChoice(freshMsg(1), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if handled, err := handleFreshManChoice(freshMsg(5), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// 参数不足 / 非整数 → false（fork :32/:36 同口径）。
	if handled, _ := handleFreshManChoice(clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomFreshManChoice}}}, "test"); handled {
		t.Fatal("short values should not be handled")
	}
	if handled, _ := handleFreshManChoice(clientCustomMessage{Values: []clientCustomValue{{Type: 2, Int32: clientCustomFreshManChoice}, {Type: 6, Text: "x"}}}, "test"); handled {
		t.Fatal("non-int choice should not be handled")
	}
}

func schoolMsg(msgID int32, values ...clientCustomValue) clientCustomMessage {
	return clientCustomMessage{Values: append([]clientCustomValue{{Type: 2, Int32: msgID}}, values...)}
}

func TestGuideAddSchoolJoinsFaction(t *testing.T) {
	player := newPlayerActor("新手侠", 0)
	conn := &captureMessageConnection{}
	if handled, err := handleGuideAddSchoolCustom(conn, player, schoolMsg(clientCustomGuideAddSchool, clientCustomValue{Type: 6, Text: "school_emei"}), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if player.factionValue() != "school_emei" {
		t.Fatalf("faction=%q", player.factionValue())
	}
	frames := conn.Frames()
	if len(frames) != 1 || frames[0][0] != 0x10 {
		t.Fatalf("frames=%d", len(frames))
	}
	if !strings.Contains(string(frames[0]), "school_emei") {
		t.Fatalf("faction frame=%x", frames[0])
	}

	// EqualFold + TrimSpace（fork lookupFactionByID 同义）。
	conn2 := &captureMessageConnection{}
	if handled, err := handleGuideAddSchoolCustom(conn2, player, schoolMsg(clientCustomGuideAddSchool, clientCustomValue{Type: 6, Text: " SCHOOL_WUDANG "}), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if player.factionValue() != "school_wudang" {
		t.Fatalf("faction=%q (EqualFold/Trim)", player.factionValue())
	}

	// 未知门派：拒绝、无帧。
	conn3 := &captureMessageConnection{}
	if handled, err := handleGuideAddSchoolCustom(conn3, player, schoolMsg(clientCustomGuideAddSchool, clientCustomValue{Type: 6, Text: "school_nosuch"}), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if frames := conn3.Frames(); len(frames) != 0 {
		t.Fatalf("unknown school emitted %d frames", len(frames))
	}
	if player.factionValue() != "school_wudang" {
		t.Fatalf("faction moved on reject: %q", player.factionValue())
	}
}

func TestSchoolMessageScansFirstNarrowString(t *testing.T) {
	player := newPlayerActor("门派侠", 0)
	conn := &captureMessageConnection{}
	if handled, err := handleSchoolCustom(conn, player, schoolMsg(clientCustomSchoolMessage,
		clientCustomValue{Type: 2, Int32: 3},
		clientCustomValue{Type: 6, Text: "force_yihua"},
		clientCustomValue{Type: 6, Text: "school_shaolin"}), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if player.factionValue() != "force_yihua" {
		t.Fatalf("faction=%q, want force_yihua (first narrow string wins)", player.factionValue())
	}

	// 无玩家 → 错误（fork :88-89 errorf 同口径）。
	if _, err := handleSchoolCustom(conn, nil, schoolMsg(clientCustomSchoolMessage, clientCustomValue{Type: 6, Text: "school_emei"}), "test"); err == nil {
		t.Fatal("no player should return error")
	}

	// 无字符串值 → 拒绝。
	conn2 := &captureMessageConnection{}
	if handled, err := handleSchoolCustom(conn2, player, schoolMsg(clientCustomSchoolMessage, clientCustomValue{Type: 2, Int32: 7}), "test"); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if frames := conn2.Frames(); len(frames) != 0 {
		t.Fatalf("no-string school message emitted %d frames", len(frames))
	}
}
