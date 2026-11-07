package main

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// Java 奇偶校验：role_appearance/role_fashion_suits/role_skins 的精确 SQL
//（AppearanceStore.upsertDatabase / RoleInventoryStore.activateFashionSuits /
// SkinStore.upsert 同构）。

func newAppearanceMockStores(t *testing.T) (sqlmock.Sqlmock, appearanceStores) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		_ = db.Close()
	})
	return mock, newAppearanceStores(db)
}

func TestAppearanceStoreSaveUpsertSQL(t *testing.T) {
	mock, stores := newAppearanceMockStores(t)
	mock.ExpectExec(`INSERT INTO role_appearance \(role_id, show_equip_type, version, updated_at\)`+
		` VALUES \(\?, \?, 1, UTC_TIMESTAMP\(6\)\)`+
		` ON DUPLICATE KEY UPDATE show_equip_type = VALUES\(show_equip_type\),`+
		` version = version \+ 1, updated_at = UTC_TIMESTAMP\(6\)`).
		WithArgs(customConfigTestRoleID, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := stores.appearance.Save(customConfigTestRoleID, showEquipFashion); err != nil {
		t.Fatal(err)
	}
	// 越界值拒绝（不触 DB）。
	if err := stores.appearance.Save(customConfigTestRoleID, 5); err == nil {
		t.Fatal("out-of-range save should fail")
	}
}

func TestAppearanceStoreLoadRowAndDegradation(t *testing.T) {
	mock, stores := newAppearanceMockStores(t)
	mock.ExpectQuery(`SELECT show_equip_type FROM role_appearance WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"show_equip_type"}).AddRow(2))
	if pref, saved := stores.appearance.Load(customConfigTestRoleID); !saved || pref != showEquipFashion {
		t.Fatalf("load=(%d,%t)", pref, saved)
	}

	// 无行 = 无存档（默认 0）。
	mock, stores = newAppearanceMockStores(t)
	mock.ExpectQuery(`SELECT show_equip_type FROM role_appearance WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"show_equip_type"}))
	if pref, saved := stores.appearance.Load(customConfigTestRoleID); saved || pref != showEquipGear {
		t.Fatalf("no-row load=(%d,%t), want (0,false)", pref, saved)
	}
}

func TestFashionSuitStoreListAndActivate(t *testing.T) {
	mock, stores := newAppearanceMockStores(t)
	mock.ExpectQuery(`SELECT suit_id FROM role_fashion_suits WHERE role_id = \? ORDER BY suit_id`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"suit_id"}).AddRow("a01").AddRow("b01"))
	suits, ok := stores.fashion.List(customConfigTestRoleID)
	if !ok || len(suits) != 2 || suits[0] != "a01" {
		t.Fatalf("list=%v ok=%t", suits, ok)
	}

	mock, stores = newAppearanceMockStores(t)
	mock.ExpectPrepare(`INSERT IGNORE INTO role_fashion_suits \(role_id, suit_id\) VALUES \(\?, \?\)`)
	for _, suit := range []string{"a01", "b01"} {
		mock.ExpectExec(`INSERT IGNORE INTO role_fashion_suits \(role_id, suit_id\) VALUES \(\?, \?\)`).
			WithArgs(customConfigTestRoleID, suit).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}
	if err := stores.fashion.Activate(customConfigTestRoleID, []string{"a01", "b01"}); err != nil {
		t.Fatal(err)
	}
}

func TestSkinStoreRoundTripSQL(t *testing.T) {
	mock, stores := newAppearanceMockStores(t)
	mock.ExpectQuery(`SELECT collected, mounted FROM role_skins WHERE role_id = \?`).
		WithArgs(customConfigTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"collected", "mounted"}).AddRow("[1,31]", "[2]"))
	state, ok := stores.skins.Load(customConfigTestRoleID)
	if !ok || len(state.collected) != 2 || state.collected[1] != 31 || len(state.mounted) != 1 {
		t.Fatalf("load=%+v ok=%t", state, ok)
	}

	mock, stores = newAppearanceMockStores(t)
	mock.ExpectExec(`INSERT INTO role_skins \(role_id, collected, mounted, version, updated_at\)`+
		` VALUES \(\?, CAST\(\? AS JSON\), CAST\(\? AS JSON\), 1, UTC_TIMESTAMP\(6\)\)`+
		` ON DUPLICATE KEY UPDATE collected = VALUES\(collected\), mounted = VALUES\(mounted\),`+
		` version = version \+ 1, updated_at = UTC_TIMESTAMP\(6\)`).
		WithArgs(customConfigTestRoleID, "[1,2]", "[2]").
		WillReturnResult(sqlmock.NewResult(0, 1))
	if err := stores.skins.Save(customConfigTestRoleID, skinState{collected: []int32{2, 1}, mounted: []int32{2}}); err != nil {
		t.Fatal(err)
	}
}

// JSON 数组形态：升序去重、空集 "[]"。
func TestMarshalSkinIDs(t *testing.T) {
	if got := marshalSkinIDs(nil); got != "[]" {
		t.Fatalf("empty=%q", got)
	}
	if got := marshalSkinIDs([]int32{3, 1, 3, 2}); got != "[1,2,3]" {
		t.Fatalf("dedup=%q", got)
	}
	if parseSkinIDs("") != nil || parseSkinIDs("null") != nil || parseSkinIDs("bad") != nil {
		t.Fatal("malformed JSON should parse to nil")
	}
	if got := parseSkinIDs("[1,2]"); len(got) != 2 || got[1] != 2 {
		t.Fatalf("parse=%v", got)
	}
}

// skinState 行为：收录幂等排序、卸载幂等。
func TestSkinStateOperations(t *testing.T) {
	var state skinState
	state.addCollected(3)
	state.addCollected(1)
	state.addCollected(3)
	if len(state.collected) != 2 || state.collected[0] != 1 {
		t.Fatalf("collected=%v", state.collected)
	}
	state.mounted = []int32{3, 1}
	state.removeMounted(3)
	if len(state.mounted) != 1 || state.mounted[0] != 1 {
		t.Fatalf("mounted=%v", state.mounted)
	}
	state.removeMounted(9) // 幂等
	if len(state.mounted) != 1 {
		t.Fatalf("mounted after noop=%v", state.mounted)
	}
	if !state.collectedContains(3) || state.collectedContains(2) {
		t.Fatal("contains 判定失败")
	}
}
