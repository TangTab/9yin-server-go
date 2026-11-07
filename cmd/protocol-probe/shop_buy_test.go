package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// 商店购销（msg70/71）：fork shop_buy.go 同源语义。内存模型为主
//（store=nil 纯内存），货币落库用 sqlmock 断言。
// fork 钉死的事实：70=[shopId,page,pos,count] count∈[1,98]、货位
// page+1/pos-1；71=[类别页,槽位1起,数量,configID]、槽位-1、固定售价
// 10 碎银/件、类别映射同 bagViewOf。

func stubShopCatalog(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shop.ini")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write stub shop.ini: %v", err)
	}
	original := defaultShopINIPath
	defaultShopINIPath = path
	t.Cleanup(func() { defaultShopINIPath = original })
}

const testShopINI = `[Shop_test_01]
Type=1
0=potion_test,1,1,50,0,1,0
1=gear_test,1,2,30,0,1,1
`

func buyMsg(shopID string, page, pos, count int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomShopBuy},
		{Type: 6, Text: shopID},
		{Type: 2, Int32: page},
		{Type: 2, Int32: pos},
		{Type: 2, Int32: count},
	}}
}

func sellMsg(category, idx, count int32, configID string) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomShopSell},
		{Type: 2, Int32: category},
		{Type: 2, Int32: idx},
		{Type: 2, Int32: count},
		{Type: 6, Text: configID},
	}}
}

func TestShopBuyChargesSilverAndStacksIntoExistingRow(t *testing.T) {
	stubShopCatalog(t, testShopINI)
	player := playerWithBag(
		bagRow{slot: 1, configID: "potion_test", itemType: 2101, amount: 3, viewID: 1},
	)
	player.addSilver(1000)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleShopBuyCustom(h, buyMsg("Shop_test_01", 0, 1, 2)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.silverBalance(); got != 900 {
		t.Fatalf("silver=%d, want 900 (50x2)", got)
	}
	if got := bagAmountOf(t, player, "potion_test", 1); got != 5 {
		t.Fatalf("stacked amount=%d, want 5", got)
	}
	// 帧序：叠堆行成对帧（remove→0x18 行→镜像 remove）+ 三币 0x10。
	frames := conn.Frames()
	if len(frames) != 6 {
		t.Fatalf("frames=%d: %x", len(frames), frames)
	}
	if frames[0][0] != 0x19 || binary.LittleEndian.Uint16(frames[0][1:]) != bagViewOf(1) {
		t.Fatalf("stack pre-clear=%x", frames[0])
	}
	if frames[1][0] != 0x18 {
		t.Fatalf("stack row frame=%x", frames[1][:1])
	}
	for index := 3; index <= 5; index++ {
		if frames[index][0] != 0x10 {
			t.Fatalf("currency frame %d=%#x", index, frames[index][0])
		}
	}
	// 三币帧 idx 序：416/417/418（金/银/卡）。
	for index, want := range []uint16{416, 417, 418} {
		if got := binary.LittleEndian.Uint16(frames[3+index][12:]); got != want {
			t.Fatalf("currency frame %d idx=%d want %d", index, got, want)
		}
	}
}

func TestShopBuyRejectsInsufficientSilverSilently(t *testing.T) {
	stubShopCatalog(t, testShopINI)
	player := playerWithBag(
		bagRow{slot: 1, configID: "potion_test", itemType: 2101, amount: 3, viewID: 1},
	)
	player.addSilver(10)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleShopBuyCustom(h, buyMsg("Shop_test_01", 0, 1, 2)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.silverBalance(); got != 10 {
		t.Fatalf("silver moved on reject: %d", got)
	}
	if got := bagAmountOf(t, player, "potion_test", 1); got != 3 {
		t.Fatalf("amount moved on reject: %d", got)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("reject emitted %d frames", len(frames))
	}
}

func TestShopBuyRejectsUnknownCellAndMalformedCount(t *testing.T) {
	stubShopCatalog(t, testShopINI)
	player := playerWithBag()
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	// 空货位（page 0 / pos 5 无条目）。
	if handled, err := handleShopBuyCustom(h, buyMsg("Shop_test_01", 0, 5, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// 超过 fork 闸 98 的数量。
	if handled, err := handleShopBuyCustom(h, buyMsg("Shop_test_01", 0, 1, 99)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// 未知商店节。
	if handled, err := handleShopBuyCustom(h, buyMsg("Shop_missing", 0, 1, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("rejects emitted %d frames", len(frames))
	}
}

func TestShopSellFixedPricePerUnitAndRowSync(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 2, configID: "potion_test", itemType: 2101, amount: 5, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	// 出售 3 件：固定 10/件 → +30 碎银，行剩余 2（重推行帧）。
	if handled, err := handleShopSellCustom(h, sellMsg(1, 3, 3, "potion_test")); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.silverBalance(); got != 30 {
		t.Fatalf("silver=%d, want 30", got)
	}
	if got := bagAmountOf(t, player, "potion_test", 1); got != 2 {
		t.Fatalf("remaining=%d, want 2", got)
	}
	frames := conn.Frames()
	if len(frames) != 6 {
		t.Fatalf("partial sell frames=%d", len(frames))
	}
	if frames[1][0] != 0x18 {
		t.Fatalf("remaining row frame=%#x", frames[1][0])
	}

	// 售空剩余 2 件：删行（ViewRemove+镜像）+ 三币帧。
	if handled, err := handleShopSellCustom(h, sellMsg(1, 3, 2, "potion_test")); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.silverBalance(); got != 50 {
		t.Fatalf("silver=%d, want 50", got)
	}
	if bagHas(player, "potion_test", 1) {
		t.Fatal("row should be removed after selling out")
	}
}

func TestShopSellRejectsStaleAndMissingCells(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 2, configID: "potion_test", itemType: 2101, amount: 5, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	// configID 与格位不符（陈旧单元格）。
	if handled, err := handleShopSellCustom(h, sellMsg(1, 3, 1, "gear_test")); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// 无行格位。
	if handled, err := handleShopSellCustom(h, sellMsg(1, 9, 1, "")); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// 超出叠堆数量。
	if handled, err := handleShopSellCustom(h, sellMsg(1, 3, 6, "potion_test")); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.silverBalance(); got != 0 {
		t.Fatalf("silver moved on reject: %d", got)
	}
	if frames := conn.Frames(); len(frames) != 0 {
		t.Fatalf("rejects emitted %d frames", len(frames))
	}
}

func TestShopBuyPersistsCurrencySnapshot(t *testing.T) {
	stubShopCatalog(t, testShopINI)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	// 发放目录查询 + 新行事务（SELECT slot / MAX(seq)+1 / INSERT）。
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items").
		WithArgs("potion_test").
		WillReturnRows(mockGameItemsRows([]string{"potion_test"}))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT slot FROM role_bag_items").WillReturnRows(mockBagSlots(nil))
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1 FROM role_bag_items").WillReturnRows(
		sqlmock.NewRows([]string{"next"}).AddRow(1))
	mock.ExpectExec("INSERT INTO role_bag_items").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectExec("INSERT INTO role_currency").
		WithArgs(901, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	player := playerWithBag()
	player.addSilver(200)
	h := newTestItemSession(&captureMessageConnection{}, player, newInventoryStore(db))
	h.currency = newCurrencyDBStore(db)

	if handled, err := handleShopBuyCustom(h, buyMsg("Shop_test_01", 0, 1, 2)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.silverBalance(); got != 100 {
		t.Fatalf("silver=%d, want 100", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
