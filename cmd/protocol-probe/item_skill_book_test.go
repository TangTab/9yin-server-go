package main

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/local/9yin-go-server/internal/role"

	"github.com/DATA-DOG/go-sqlmock"
)

// 秘籍使用链（Java useSkillBook :4895）：门槛先拒不消耗；wuxue 直达层 +
// View40 重开 + role_skills 落库；neigong 只抬修炼上限 + View43 重建；
// qinggong View46 追加行；全部已学不消耗。

func bookUseMsg(view, slot int32) clientCustomMessage {
	return useItemMsg(view, slot, 1)
}

func testBookCatalog(entries map[string]skillBookEntry) *skillBookCatalog {
	return &skillBookCatalog{entries: entries}
}

func injectBookTables(t *testing.T, books *skillBookCatalog, gates *skillGateTable) {
	t.Helper()
	applyItemTestOverrides(currentDropPacks(), currentBoxContents(), currentItemUseCatalog(), books, gates)
}

func mockSkillRows(rows map[string]gameSkillRow) *sqlmock.Rows {
	result := sqlmock.NewRows([]string{"skill_id", "skill_type", "static_data", "item_type"})
	for _, row := range rows {
		result.AddRow(row.skillID, row.skillType, row.staticData, row.itemType)
	}
	return result
}

func TestUseSkillBookGateRejectsWithoutConsume(t *testing.T) {
	gates := loadSkillGateTableFromTables(iniTable{
		"12193": {
			{key: "Type", value: "3"}, {key: "Para1", value: "CS_em_jdmz01"},
			{key: "Para2", value: "MaxLevel"}, {key: "min", value: "1"}, {key: "max", value: "1"},
		},
	})
	books := testBookCatalog(map[string]skillBookEntry{
		"book_jdmz02": {pack: "p", targets: []string{"CS_em_jdmz01"}, skillMaxLevel: 2, useCondition: 12193},
	})
	injectBookTables(t, books, gates)

	// 行为类分类需 DB 目录（script=SkillBook → useSkillBook → 门槛拒绝）。
	store, mock := newMockInventoryDB(t)
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items").
		WithArgs("book_jdmz02").WillReturnRows(mockGameItemsRowsWithScript("book_jdmz02", "SkillBook"))
	player := playerWithBag(
		bagRow{slot: 1, configID: "book_jdmz02", itemType: 21, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)

	if handled, err := handleBodyItemUse(h, bookUseMsg(2, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// 门槛：目标未学（上限 0），需 ∈[1,1] → 拒（不教不消耗）。
	if _, learned := player.learnedSkillLevel("CS_em_jdmz01"); learned {
		t.Fatal("gate rejected book must not teach skill")
	}
	if got := bagAmountOf(t, player, "book_jdmz02", 1); got != 1 {
		t.Fatal("gate rejected book must not consume")
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("gate rejection emitted %d frames", got)
	}
}

func TestUseSkillBookWuxueTeachesAndRebuildsView40(t *testing.T) {
	books := testBookCatalog(map[string]skillBookEntry{
		"book_new_taolu": {pack: "p", targets: []string{"CS_yhwq_hsqs03"}, skillMaxLevel: 3},
	})
	injectBookTables(t, books, loadSkillGateTableFromTables(nil))

	store, mock := newMockInventoryDB(t)
	skillRows := map[string]gameSkillRow{
		"CS_yhwq_hsqs03": {skillID: "CS_yhwq_hsqs03", skillType: "wuxue", staticData: 12638, itemType: 1000},
	}
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items").
		WithArgs("book_new_taolu").WillReturnRows(mockGameItemsRowsWithScript("book_new_taolu", "SkillBook"))
	mock.ExpectQuery("SELECT skill_id, skill_type, static_data, item_type FROM game_skills").
		WithArgs("CS_yhwq_hsqs03").WillReturnRows(mockSkillRows(skillRows))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM role_skills").
		WithArgs(int64(role.RoleID(901))).WillReturnRows(mockIntRows([]int64{1})) // 已有基线 → 跳过物化
	mock.ExpectExec("INSERT IGNORE INTO role_skills").
		WithArgs(int64(role.RoleID(901)), "wuxue", "CS_yhwq_hsqs03").
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT amount FROM role_bag_items").
		WithArgs(int64(role.RoleID(901)), int32(1), int32(1)).WillReturnRows(mockIntRows([]int64{1}))
	mock.ExpectExec("DELETE FROM role_bag_items").WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()

	player := playerWithBag(
		bagRow{slot: 1, configID: "book_new_taolu", itemType: 21, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)

	if handled, err := handleBodyItemUse(h, bookUseMsg(2, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if level, learned := player.learnedSkillLevel("CS_yhwq_hsqs03"); !learned || level != 3 {
		t.Fatalf("learned=%t level=%d, want level 3 (SkillMaxLevel)", learned, level)
	}
	// View40 整视图重开：0x16 删 → 0x15 容器 → 行帧（starter + 新书行）。
	frames := conn.Frames()
	deleted := false
	created := false
	rowSeen := false
	for _, frame := range frames {
		if frame[0] == 0x16 && binary.LittleEndian.Uint16(frame[1:]) == uint16(viewportSkill) {
			deleted = true
		}
		if frame[0] == 0x15 && binary.LittleEndian.Uint16(frame[1:]) == uint16(viewportSkill) {
			created = true
		}
		if frame[0] == 0x18 && bytes.Contains(frame, []byte("CS_yhwq_hsqs03\x00")) {
			rowSeen = true
		}
	}
	if !deleted || !created || !rowSeen {
		t.Fatalf("view rebuild deleted=%v created=%v row=%v frames=%d", deleted, created, rowSeen, len(frames))
	}
	// 书已消耗。
	if got := len(player.sessionBag()); got != 0 {
		t.Fatalf("book not consumed: %d rows", got)
	}
}

func TestUseSkillBookNeigongRaisesCapOnly(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Skip("neigong catalog unavailable in fixture")
	}
	// 选一本不在 starter 进度里的书（夹具目录里的新内功）。
	starter := map[string]bool{"ng_wd_001": true, "ng_yh_001": true, "ng_th_001": true,
		"ng_wd_002": true, "ng_jh_001": true, "ng_em_001": true}
	var target string
	for id := range catalog.books {
		if !starter[id] {
			target = id
			break
		}
	}
	if target == "" {
		t.Skip("fixture neigong catalog has no non-starter book")
	}
	def, _ := catalog.book(target)
	books := testBookCatalog(map[string]skillBookEntry{
		"book_new_ng": {pack: "p", targets: []string{target}, skillMaxLevel: 9},
	})
	injectBookTables(t, books, loadSkillGateTableFromTables(nil))

	store, mock := newMockInventoryDB(t)
	skillRows := map[string]gameSkillRow{target: {skillID: target, skillType: "neigong", staticData: def.staticData, itemType: def.itemType}}
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items").
		WithArgs("book_new_ng").WillReturnRows(mockGameItemsRowsWithScript("book_new_ng", "SkillBook"))
	mock.ExpectQuery("SELECT skill_id, skill_type, static_data, item_type FROM game_skills").
		WithArgs(target).WillReturnRows(mockSkillRows(skillRows))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT amount FROM role_bag_items").
		WithArgs(int64(role.RoleID(901)), int32(1), int32(1)).WillReturnRows(mockIntRows([]int64{1}))
	mock.ExpectExec("DELETE FROM role_bag_items").WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()

	player := playerWithBag(
		bagRow{slot: 1, configID: "book_new_ng", itemType: 21, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)

	if handled, err := handleBodyItemUse(h, bookUseMsg(2, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	player.mu.Lock()
	book := player.progress.book(target)
	player.mu.Unlock()
	if book == nil {
		t.Fatal("neigong book not created")
	}
	// 只抬修炼上限（层由修炼 ticker 自涨）：level=1、maxLevel=min(9, 目录满级)。
	wantCap := int32(9)
	if catalogMax := catalog.maxLevelOf(def.staticData); catalogMax > 0 && catalogMax < wantCap {
		wantCap = catalogMax
	}
	if book.level != 1 || book.maxLevel != wantCap {
		t.Fatalf("book level=%d maxLevel=%d, want 1/%d (cap-only)", book.level, book.maxLevel, wantCap)
	}
	// View43 整视图重建：0x16 删 + 0x15 容器。
	viewDeleted := false
	for _, frame := range conn.Frames() {
		if frame[0] == 0x16 && binary.LittleEndian.Uint16(frame[1:]) == uint16(viewportNeiGong) {
			viewDeleted = true
		}
	}
	if !viewDeleted {
		t.Fatal("neigong view not rebuilt")
	}
}

func TestUseSkillBookQingGongAppendsView46Rows(t *testing.T) {
	books := testBookCatalog(map[string]skillBookEntry{
		"book_qg": {pack: "p", targets: []string{"qinggong_test_90"}},
	})
	injectBookTables(t, books, loadSkillGateTableFromTables(nil))

	store, mock := newMockInventoryDB(t)
	skillRows := map[string]gameSkillRow{"qinggong_test_90": {skillID: "qinggong_test_90", skillType: "qinggong"}}
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items").
		WithArgs("book_qg").WillReturnRows(mockGameItemsRowsWithScript("book_qg", "SkillBook"))
	mock.ExpectQuery("SELECT skill_id, skill_type, static_data, item_type FROM game_skills").
		WithArgs("qinggong_test_90").WillReturnRows(mockSkillRows(skillRows))
	mock.ExpectExec("INSERT IGNORE INTO role_skills").
		WithArgs(int64(role.RoleID(901)), "qinggong", "qinggong_test_90").
		WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT amount FROM role_bag_items").
		WithArgs(int64(role.RoleID(901)), int32(1), int32(1)).WillReturnRows(mockIntRows([]int64{1}))
	mock.ExpectExec("DELETE FROM role_bag_items").WillReturnResult(newSQLMockResult(0, 1))
	mock.ExpectCommit()

	player := playerWithBag(
		bagRow{slot: 1, configID: "book_qg", itemType: 21, amount: 1, viewID: 1},
	)
	conn := &captureMessageConnection{}
	h := newTestItemSession(conn, player, store)

	if handled, err := handleBodyItemUse(h, bookUseMsg(2, 1)); err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	// QingGongRec 记录行 + View46 追加行（0x18 带 configID）。
	frames := conn.Frames()
	recordSeen := false
	rowSeen := false
	for _, frame := range frames {
		if frame[0] == 0x11 && bytes.Contains(frame, []byte("qinggong_test_90\x00")) {
			recordSeen = true
		}
		if frame[0] == 0x18 && bytes.Contains(frame, []byte("qinggong_test_90\x00")) {
			rowSeen = true
		}
	}
	if !recordSeen || !rowSeen {
		t.Fatalf("qinggong record=%v row=%v frames=%d", recordSeen, rowSeen, len(frames))
	}
	// 重复使用同一本：已学不消耗。
	player2 := playerWithBag(
		bagRow{slot: 1, configID: "book_qg", itemType: 21, amount: 1, viewID: 1},
	)
	player2.mu.Lock()
	player2.qingGongExtra["qinggong_test_90"] = struct{}{}
	player2.mu.Unlock()
	store2, mock2 := newMockInventoryDB(t)
	mock2.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items").
		WithArgs("book_qg").WillReturnRows(mockGameItemsRowsWithScript("book_qg", "SkillBook"))
	mock2.ExpectQuery("SELECT skill_id, skill_type, static_data, item_type FROM game_skills").
		WithArgs("qinggong_test_90").WillReturnRows(mockSkillRows(skillRows))
	conn2 := &captureMessageConnection{}
	h2 := newTestItemSession(conn2, player2, store2)
	if handled, err := handleBodyItemUse(h2, bookUseMsg(2, 1)); err != nil || !handled {
		t.Fatalf("repeat handled=%t err=%v", handled, err)
	}
	if got := bagAmountOf(t, player2, "book_qg", 1); got != 1 {
		t.Fatal("全已学秘籍不应消耗")
	}
	if got := len(conn2.Frames()); got != 0 {
		t.Fatalf("repeat emitted %d frames", got)
	}
}
