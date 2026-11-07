package main

import (
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"

	"github.com/DATA-DOG/go-sqlmock"
)

// msg31 五步链（Java useItemEffect :5188）：节流→Lua 钩子→INI 回退→
// 效果落账→消耗帧对。目录经 applyItemTestOverrides 注入合成表。

func useItemMsg(view, slot, count int32) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomUseItem},
		{Type: 2, Int32: view},
		{Type: 2, Int32: slot},
		{Type: 2, Int32: count},
	}}
}

func injectItemTables(t *testing.T, packs *dropPacks, contents *boxContents, use *itemUseCatalog) {
	t.Helper()
	applyItemTestOverrides(packs, contents, use, currentSkillBookCatalog(), currentSkillGateTable())
}

func TestUseItemSkillLadderHealsAndConsumes(t *testing.T) {
	injectItemTables(t, mustDropPacks(t, []byte(`{"packages":{}}`)), &boxContents{},
		loadItemUseCatalogFromTables(nil, nil, iniTable{
			"2101": {{key: "SkillID", value: "yaopinhp1"}, {key: "SkillLevel", value: "1"}},
		}))
	player := playerWithBag(
		bagRow{slot: 2, configID: "yaopin_00100", itemType: 2101, amount: 3, viewID: 1},
	)
	player.mu.Lock()
	player.vitals.HP = 100
	player.vitals.MP = 100 // 留出回蓝空间（满值会被结算闸掉）
	beforeHP := player.vitals.HP
	beforeMP := player.vitals.MP
	player.mu.Unlock()
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleBodyItemUse(h, useItemMsg(2, 2, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// L1 阶梯：+100 HP / +50 MP（未满则入账）。
	player.mu.Lock()
	afterHP, afterMP := player.vitals.HP, player.vitals.MP
	player.mu.Unlock()
	if afterHP-beforeHP != 100 {
		t.Fatalf("HP delta=%d, want 100", afterHP-beforeHP)
	}
	if afterMP-beforeMP != 50 {
		t.Fatalf("MP delta=%d, want 50", afterMP-beforeMP)
	}
	// 消耗 1：amount 3→2（remove+add+remove 三帧）+ vital 0x10。
	if got := bagAmountOf(t, player, "yaopin_00100", 1); got != 2 {
		t.Fatalf("amount=%d, want 2", got)
	}
	frames := conn.Frames()
	vitalSeen := false
	for _, frame := range frames {
		if frame[0] == 0x10 {
			vitalSeen = true
		}
	}
	if !vitalSeen {
		t.Fatalf("heal vital frame missing: %x", frames)
	}
}

func TestUseItemThrottledWithin800ms(t *testing.T) {
	injectItemTables(t, mustDropPacks(t, []byte(`{"packages":{}}`)), &boxContents{},
		loadItemUseCatalogFromTables(nil, nil, iniTable{
			"2101": {{key: "SkillID", value: "yaopinhp1"}},
		}))
	player := playerWithBag(
		bagRow{slot: 2, configID: "yaopin_00100", itemType: 2101, amount: 5, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleBodyItemUse(h, useItemMsg(2, 2, 1)); err != nil || !handled {
		t.Fatalf("first handled=%t err=%v", handled, err)
	}
	base := len(conn.Frames())
	// now() 固定 → 第二次在 800ms 窗口内：静默。
	if handled, err := handleBodyItemUse(h, useItemMsg(2, 2, 1)); err != nil || !handled {
		t.Fatalf("second handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("throttled use emitted %d frames", got)
	}
	if got := bagAmountOf(t, player, "yaopin_00100", 1); got != 4 {
		t.Fatalf("amount=%d, want 4 (first use only)", got)
	}
	// 窗口过后放行。
	h.now = func() time.Time { return time.Unix(1700000001, 0).UTC() }
	if handled, err := handleBodyItemUse(h, useItemMsg(2, 2, 1)); err != nil || !handled {
		t.Fatalf("third handled=%t err=%v", handled, err)
	}
	if got := bagAmountOf(t, player, "yaopin_00100", 1); got != 3 {
		t.Fatalf("amount=%d, want 3", got)
	}
}

func TestUseItemBoxGrantsDescribedContentsAndConsumes(t *testing.T) {
	// box_contents 命中优先于 DropPacks roll（authored 包会被忽略）。
	packs := mustDropPacks(t, []byte(`{"packages": {"box_hit": [{"item": "yaopin_00700", "min": 1, "max": 1, "weight": 10}]}}`))
	contents, err := loadBoxContents([]byte(`{"boxes": {"box_hit": [{"item": "item_ptz", "min": 2, "max": 2}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	toolItem := iniTable{
		"box_hit": {{key: "script", value: "BoxItem"}, {key: "DropID", value: "box_hit"}},
	}
	store, mock := newMockInventoryDB(t)
	// ① 行为类分类查询（box_hit → script=BoxItem）② 发放目录查询（item_ptz）
	// ③ insertBagItem 序列（SELECT slot / MAX(seq) / INSERT）。
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items").
		WithArgs("box_hit").WillReturnRows(mockGameItemsRowsWithScript("box_hit", "BoxItem"))
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items").
		WithArgs("item_ptz").WillReturnRows(mockGameItemsRows([]string{"item_ptz"}))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT slot FROM role_bag_items").WillReturnRows(mockBagSlots([]int32{5}))
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1 FROM role_bag_items").
		WillReturnRows(mockIntRows([]int64{11}))
	mock.ExpectExec("INSERT INTO role_bag_items").
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()
	// 消耗（事务）：SELECT amount → 剩余 0 → DELETE。
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT amount FROM role_bag_items").WithArgs(int64(role.RoleID(901)), int32(5), int32(1)).
		WillReturnRows(mockIntRows([]int64{1}))
	mock.ExpectExec("DELETE FROM role_bag_items").WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()

	player := playerWithBag(
		bagRow{slot: 5, configID: "box_hit", itemType: 2102, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)
	injectItemTables(t, packs, contents, loadItemUseCatalogFromTables(nil, toolItem, nil))

	if handled, err := handleBodyItemUse(h, useItemMsg(2, 5, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagAmountOf(t, player, "item_ptz", 1); got != 2 {
		t.Fatalf("granted item_ptz amount=%d, want 2", got)
	}
	player.mu.Lock()
	bagCount := len(player.bag)
	player.mu.Unlock()
	if bagCount != 1 {
		t.Fatalf("bag rows=%d, want 1 (box consumed, item granted)", bagCount)
	}
}

func TestUseItemLuaHookGrantsAndSkipConsume(t *testing.T) {
	previous := luaItemUseHook
	luaItemUseHook = func(configID string, amount int32) *luaItemUseInstruction {
		return &luaItemUseInstruction{healHP: 100, consume: false}
	}
	t.Cleanup(func() { luaItemUseHook = previous })

	player := playerWithBag(
		bagRow{slot: 1, configID: "yaopin_sc_1001", itemType: 2101, amount: 1, viewID: 1},
	)
	player.mu.Lock()
	player.vitals.HP = 50
	player.mu.Unlock()
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleBodyItemUse(h, useItemMsg(2, 1, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	player.mu.Lock()
	hp := player.vitals.HP
	player.mu.Unlock()
	if hp != 150 {
		t.Fatalf("HP=%d, want 150", hp)
	}
	if got := bagAmountOf(t, player, "yaopin_sc_1001", 1); got != 1 {
		t.Fatalf("consume=false 仍被消耗: amount=%d", got)
	}
}

func TestUseItemTempStatOnlySkipsConsume(t *testing.T) {
	previous := luaItemUseHook
	luaItemUseHook = func(configID string, amount int32) *luaItemUseInstruction {
		return &luaItemUseInstruction{tempStat: true}
	}
	t.Cleanup(func() { luaItemUseHook = previous })

	player := playerWithBag(
		bagRow{slot: 1, configID: "buff_stone_01", itemType: 2105, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleBodyItemUse(h, useItemMsg(2, 1, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := bagAmountOf(t, player, "buff_stone_01", 1); got != 1 {
		t.Fatal("temp_stat-only 效果不应消耗（防白吃）")
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("temp_stat-only emitted %d frames", got)
	}
}

func TestUseItemRejectsMalformedValues(t *testing.T) {
	player := newPlayerActor("背包侠", 0)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	// 3 值（缺 count）/count=0：拒绝零帧。
	short := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomUseItem},
		{Type: 2, Int32: 2},
		{Type: 2, Int32: 1},
	}}
	if handled, err := handleBodyItemUse(h, short); err != nil || !handled {
		t.Fatalf("short handled=%t err=%v", handled, err)
	}
	if handled, err := handleBodyItemUse(h, useItemMsg(2, 1, 0)); err != nil || !handled {
		t.Fatalf("zero count handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("malformed emitted %d frames", got)
	}
}

func TestUseItemWearableBranchWears(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 2, configID: "hat_test_001", itemType: 101, amount: 1, viewID: 2, equipType: "Hat"},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, nil)

	if handled, err := handleBodyItemUse(h, useItemMsg(121, 2, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if _, _, ok := player.equipAtSlot(1); !ok {
		t.Fatal("wearable use should wear hat to slot 1")
	}
}

// ---- sqlmock 行构造辅助 ----

func mockGameItemsRows(configIDs []string) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"item_id", "item_type", "view_id", "item_name", "equip_type", "script"})
	for _, id := range configIDs {
		rows.AddRow(id, 1, 1, "测试物品", "", "ToolItem")
	}
	return rows
}

func mockGameItemsRowsWithScript(configID, script string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"item_id", "item_type", "view_id", "item_name", "equip_type", "script"}).
		AddRow(configID, 1, 1, "测试物品", "", script)
}

func mockBagSlots(slots []int32) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"slot"})
	for _, slot := range slots {
		rows.AddRow(slot)
	}
	return rows
}

func mockIntRows(values []int64) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"value"})
	for _, value := range values {
		rows.AddRow(value)
	}
	return rows
}
