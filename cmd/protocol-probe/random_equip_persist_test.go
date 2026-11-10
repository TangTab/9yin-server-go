package main

// 随机属性持久化审计回归（docs/镇派兵器与娃娃宝物随机属性.md §4/§6）：
// ①upsertEquipSlot 的 uid 携带条件——异款覆盖必须新 uid（否则新宝物/娃娃
// 命中旧物品的 role_item_random_props 快照，属性串号）；②消耗删行时清理
// 快照（防孤儿行与 uid 复用串号）。

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/local/9yin-go-server/internal/role"
)

// uidOtherThan 匹配任意非空且不等于排除值的 uid 串（sqlmock 自定义参数
// matcher：异款覆盖断言 INSERT 落库的是新生成的 uid）。
type uidOtherThan struct{ exclude string }

func (m uidOtherThan) Match(v driver.Value) bool {
	s, ok := v.(string)
	return ok && s != "" && s != m.exclude
}

func (m uidOtherThan) String() string {
	return fmt.Sprintf("uid != %q", m.exclude)
}

// TestUpsertEquipSlotFreshUIDOnDifferentItem 异款覆盖生成新 uid（P1 回归：
// 继承旧 uid 会让新物品命中旧物品的随机属性快照）。
func TestUpsertEquipSlotFreshUIDOnDifferentItem(t *testing.T) {
	store, mock := newMockInventoryDB(t)
	mock.ExpectBegin()
	oldRow := sqlmock.NewRows([]string{"uid", "config_id"}).AddRow("OLDUID", "treasure_old")
	mock.ExpectQuery("SELECT uid, config_id FROM role_equip_items WHERE role_id = \\? AND slot = \\?").
		WithArgs(int64(role.RoleID(901)), int32(30)).
		WillReturnRows(oldRow)
	mock.ExpectExec("DELETE FROM role_equip_items WHERE role_id = \\? AND slot = \\?").
		WithArgs(int64(role.RoleID(901)), int32(30)).
		WillReturnResult(newSQLMockResult(0, 1))
	seqRows := sqlmock.NewRows([]string{"seq"}).AddRow(7)
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1 FROM role_equip_items").
		WillReturnRows(seqRows)
	mock.ExpectExec("INSERT INTO role_equip_items").
		WithArgs(int64(role.RoleID(901)), int64(7), int32(30), uidOtherThan{exclude: "OLDUID"}, "[]", "newtreasure_grow_6_02").
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()
	readback := sqlmock.NewRows([]string{
		"config_id", "item_type", "equip_type", "slot",
		"hardiness", "max_hardiness", "seq", "uid", "affixes",
	}).AddRow("newtreasure_grow_6_02", 193, "NewTreasure", 30, 100, 100, 7, "READBACKUID", "[]")
	mock.ExpectQuery("SELECT config_id, item_type, equip_type, slot, hardiness, max_hardiness, seq, uid, affixes").
		WillReturnRows(readback)

	row, err := store.upsertEquipSlot(context.Background(), role.RoleID(901), 30,
		"newtreasure_grow_6_02", 193, "NewTreasure")
	if err != nil {
		t.Fatalf("upsertEquipSlot: %v", err)
	}
	if row.uid == "OLDUID" || row.uid == "" {
		t.Fatalf("uid=%q, want fresh uid (not inherited OLDUID)", row.uid)
	}
}

// TestUpsertEquipSlotCarriesUIDForSameItem 同款重穿保留实例身份（时装
// 语义/客户端实例稳定）。
func TestUpsertEquipSlotCarriesUIDForSameItem(t *testing.T) {
	store, mock := newMockInventoryDB(t)
	mock.ExpectBegin()
	oldRow := sqlmock.NewRows([]string{"uid", "config_id"}).AddRow("SAMEUID", "fashion_hat_01")
	mock.ExpectQuery("SELECT uid, config_id FROM role_equip_items WHERE role_id = \\? AND slot = \\?").
		WithArgs(int64(role.RoleID(901)), int32(18)).
		WillReturnRows(oldRow)
	mock.ExpectExec("DELETE FROM role_equip_items WHERE role_id = \\? AND slot = \\?").
		WithArgs(int64(role.RoleID(901)), int32(18)).
		WillReturnResult(newSQLMockResult(0, 1))
	seqRows := sqlmock.NewRows([]string{"seq"}).AddRow(7)
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1 FROM role_equip_items").
		WillReturnRows(seqRows)
	mock.ExpectExec("INSERT INTO role_equip_items").
		WithArgs(int64(role.RoleID(901)), int64(7), int32(18), "SAMEUID", "[]", "fashion_hat_01").
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()
	readback := sqlmock.NewRows([]string{
		"config_id", "item_type", "equip_type", "slot",
		"hardiness", "max_hardiness", "seq", "uid", "affixes",
	}).AddRow("fashion_hat_01", 0, "FashionHat", 18, 100, 100, 7, "SAMEUID", nil)
	mock.ExpectQuery("SELECT config_id, item_type, equip_type, slot, hardiness, max_hardiness, seq, uid, affixes").
		WillReturnRows(readback)

	row, err := store.upsertEquipSlot(context.Background(), role.RoleID(901), 18,
		"fashion_hat_01", 0, "FashionHat")
	if err != nil {
		t.Fatalf("upsertEquipSlot: %v", err)
	}
	if row.uid != "SAMEUID" {
		t.Fatalf("uid=%q, want carried SAMEUID for same config re-wear", row.uid)
	}
}

// syntheticRandomPropsCatalog 注入只含 treasure_test 的合成目录（kind 判定
// 用），t.Cleanup 还原旧指针防污染后续用例。
func syntheticRandomPropsCatalog(t *testing.T) {
	t.Helper()
	old := randomEquipCat
	resetRandomEquipCatalogForTest()
	randomEquipCatalogOnce.Do(func() {})
	randomEquipCat = &randomEquipCatalog{
		itemFields: map[string]randomItemFields{
			"treasure_test": {equipType: "Treasure", colorLevel: 6, level: 64, templateID: 7},
		},
	}
	t.Cleanup(func() { randomEquipCat = old })
}

// TestConsumeBagItemDeletesRandomPropsSnapshot 宝物/娃娃行删净（msg34 丢弃/
// 消耗归零）时同步清理 role_item_random_props 快照。
func TestConsumeBagItemDeletesRandomPropsSnapshot(t *testing.T) {
	syntheticRandomPropsCatalog(t)
	player := playerWithBag(
		bagRow{slot: 4, configID: "treasure_test", itemType: 146, amount: 5, viewID: 2,
			equipType: "Treasure", uid: "UID888"},
	)
	store, mock := newMockInventoryDB(t)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)
	h.randomProps = newRandomPropsDBStore(store.db)

	mock.ExpectBegin()
	amountRow := sqlmock.NewRows([]string{"amount"}).AddRow(5)
	mock.ExpectQuery("SELECT amount FROM role_bag_items").
		WillReturnRows(amountRow)
	mock.ExpectExec("DELETE FROM role_bag_items WHERE role_id = \\? AND slot = \\? AND view_id = \\?").
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()
	// 快照清理（P2）：行删净即实例消亡。
	mock.ExpectExec("DELETE FROM role_item_random_props WHERE role_id = \\? AND uid = \\?").
		WithArgs(int64(role.RoleID(901)), "UID888").
		WillReturnResult(newSQLMockResult(0, 1))

	if !h.consumeBagItemCount(player.sessionBag()[0], 5) {
		t.Fatal("consume failed")
	}
	if bagCount := len(player.sessionBag()); bagCount != 0 {
		t.Fatalf("rows=%d, want row deleted", bagCount)
	}
}

// TestConsumeBagItemNoSnapshotCleanupOnPartialOrPlain 部分扣减（行保留）
// 与非随机装备删净（kind 门）都不触发快照清理——sqlmock 严格模式下期望表
// 未列出的 DELETE 会让 ExpectationsWereMet 失败，缺席即断言。
func TestConsumeBagItemNoSnapshotCleanupOnPartialOrPlain(t *testing.T) {
	player := playerWithBag(
		bagRow{slot: 4, configID: "treasure_test", itemType: 146, amount: 5, viewID: 2, uid: "UID888"},
		bagRow{slot: 6, configID: "yaopin_00100", itemType: 2101, amount: 3, viewID: 1, uid: "UID999"},
	)
	store, mock := newMockInventoryDB(t)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)
	h.randomProps = newRandomPropsDBStore(store.db)

	// 部分扣减宝物：UPDATE 分支。
	mock.ExpectBegin()
	amountRow := sqlmock.NewRows([]string{"amount"}).AddRow(5)
	mock.ExpectQuery("SELECT amount FROM role_bag_items").
		WillReturnRows(amountRow)
	mock.ExpectExec("UPDATE role_bag_items SET amount = \\?").
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()
	if !h.consumeBagItemCount(player.sessionBag()[0], 2) {
		t.Fatal("partial consume failed")
	}

	// 删净非随机装备（药）：DELETE 分支但 kind=none 不清快照。
	mock.ExpectBegin()
	amountRow2 := sqlmock.NewRows([]string{"amount"}).AddRow(3)
	mock.ExpectQuery("SELECT amount FROM role_bag_items").
		WillReturnRows(amountRow2)
	mock.ExpectExec("DELETE FROM role_bag_items WHERE role_id = \\? AND slot = \\? AND view_id = \\?").
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()
	if !h.consumeBagItemCount(player.sessionBag()[1], 3) {
		t.Fatal("plain consume failed")
	}
	if bagCount := len(player.sessionBag()); bagCount != 1 {
		t.Fatalf("rows=%d, want 1 (treasure partial + potion deleted)", bagCount)
	}
}
