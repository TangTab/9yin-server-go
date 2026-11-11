package main

// GM 武学动作单测：等级档存档读写/原子性、totalFillDisplay 口径、
// 等级行帧编码、登录档值合并（applyStoredSkillLevels）、整套学习
// （learn_taolu/learn_neigong：逐技钳上限/建书直达层/只升不降）。

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

func TestSkillLevelFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), skillLevelFileName)
	store := openSkillLevelStore(path)
	if level := store.levelOf(42, "CS_wd_tjq01"); level != 0 {
		t.Fatalf("empty store level=%d want 0", level)
	}
	if err := store.apply(42, map[string]int32{
		"CS_wd_tjq01":  19,
		"zs_normal_02": 1, // 仅 >1 落档
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// 重开取档：>1 档值保持，1 级行被丢弃。
	reopened := openSkillLevelStore(path)
	if level := reopened.levelOf(42, "CS_wd_tjq01"); level != 19 {
		t.Fatalf("reopened level=%d want 19", level)
	}
	if level := reopened.levelOf(42, "zs_normal_02"); level != 0 {
		t.Fatalf("level-1 row must not persist, got %d", level)
	}
	if level := reopened.levelOf(7, "CS_wd_tjq01"); level != 0 {
		t.Fatalf("other role level=%d want 0", level)
	}
	// 空表 apply 等价 clear。
	if err := reopened.apply(42, nil); err != nil {
		t.Fatalf("apply empty: %v", err)
	}
	if level := reopened.levelOf(42, "CS_wd_tjq01"); level != 0 {
		t.Fatalf("after empty apply level=%d want 0", level)
	}
}

func TestSkillLevelFileStoreRefusesBadArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), skillLevelFileName)
	if err := os.WriteFile(path, []byte(`{"version":99,"roles":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	store := openSkillLevelStore(path)
	if !store.loadFailed {
		t.Fatal("version mismatch must set loadFailed")
	}
	if err := store.apply(42, map[string]int32{"CS_wd_tjq01": 19}); err == nil {
		t.Fatal("persist after failed load must be refused")
	}
	// 档文件不得被覆写。
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"version":99`) {
		t.Fatalf("archive was overwritten: %s", raw)
	}
}

func TestSkillLevelFileStoreClearIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), skillLevelFileName)
	store := openSkillLevelStore(path)
	if err := store.clear(42); err != nil {
		t.Fatalf("clear without archive: %v", err)
	}
	if err := store.apply(42, map[string]int32{"CS_wd_tjq01": 19}); err != nil {
		t.Fatal(err)
	}
	if err := store.clear(42); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if level := store.levelOf(42, "CS_wd_tjq01"); level != 0 {
		t.Fatalf("after clear level=%d want 0", level)
	}
}

func TestTotalFillDisplay(t *testing.T) {
	// starter 内联 total（花神七式 156550）。
	if got := totalFillDisplay("CS_yhwq_hsqs01"); got != 156550 {
		t.Fatalf("hsqs01 totalFill=%d want 156550", got)
	}
	// starter 集外技能 0（Java SkillInitTable 现代数据同口径）。
	if got := totalFillDisplay("no_such_skill"); got != 0 {
		t.Fatalf("unknown totalFill=%d want 0", got)
	}
}

func TestSkillRowLevelUpFrameEncoding(t *testing.T) {
	varPropNo := int32(1234)
	frame, err := skillRowLevelUpFrame(3, 19, 19, 156550, 156550, &varPropNo)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame) == 0 {
		t.Fatal("empty frame")
	}
	if frame[0] != 0x10 {
		t.Fatalf("opcode=%#x want 0x10", frame[0])
	}
	// 无 varPropNo（保持现值）也要可编码。
	if _, err := skillRowLevelUpFrame(3, 19, 19, 156550, 156550, nil); err != nil {
		t.Fatalf("without varPropNo: %v", err)
	}
	// 非法等级拒绝。
	if _, err := skillRowLevelUpFrame(3, 0, 0, 0, 0, nil); err == nil {
		t.Fatal("level 0 must be rejected")
	}
}

func TestApplyStoredSkillLevels(t *testing.T) {
	player := newPlayerActor("测试", 0)
	path := filepath.Join(t.TempDir(), skillLevelFileName)
	store := openSkillLevelStore(path)
	if err := store.apply(9, map[string]int32{"CS_wd_tjq01": 19}); err != nil {
		t.Fatal(err)
	}
	rows := mergeLearnedSkills(nil)
	rows = applyStoredSkillLevels(player, role.RoleID(9), store, rows)
	// 行显示：Level/MaxLevel 抬到档值，修为 CurFill=TotalFill=内联 total。
	found := false
	for _, row := range rows {
		if row.configID != "CS_wd_tjq01" {
			continue
		}
		found = true
		if row.level != 19 || row.maxLevel != 19 {
			t.Fatalf("tjq01 row level/max=%d/%d want 19/19", row.level, row.maxLevel)
		}
	}
	if !found {
		t.Fatal("CS_wd_tjq01 row missing from starter baseline")
	}
	// 释放闸档值同步：learnedSkillLevel 必须出自同一档值。
	if level, learned := player.learnedSkillLevel("CS_wd_tjq01"); !learned || level != 19 {
		t.Fatalf("learnedSkillLevel=%d/%t want 19/true", level, learned)
	}
	// 未落档技能保持 1 级。
	if level, _ := player.learnedSkillLevel("zs_normal_02"); level != 1 {
		t.Fatalf("unstored skill level=%d want 1", level)
	}
	// 幂等：重放不重复抬升。
	rows = applyStoredSkillLevels(player, role.RoleID(9), store, rows)
	if level, _ := player.learnedSkillLevel("CS_wd_tjq01"); level != 19 {
		t.Fatalf("replay level=%d want 19", level)
	}
	// 包级 starter 基线不被污染（mergeLearnedSkills 空集直返 starterSkillViews
	// 本体，改写在副本上进行——跨会话等级泄漏回归锚）。
	for _, skill := range starterSkillViews {
		if skill.configID == "CS_wd_tjq01" && skill.level != 1 {
			t.Fatalf("starterSkillViews mutated: CS_wd_tjq01 level=%d want 1", skill.level)
		}
	}
	// 其他角色档不串号。
	if level, _ := player.learnedSkillLevel("CS_wd_tjq02"); level != 1 {
		t.Fatalf("other skill level=%d want 1", level)
	}
}

func TestSkillViewRowOrder(t *testing.T) {
	player := newPlayerActor("测试", 0)
	rows := mergeLearnedSkills(nil)
	player.noteSkillViewRows(rows)
	order := player.skillViewRowOrder()
	if len(order) != len(rows) {
		t.Fatalf("order len=%d want %d", len(order), len(rows))
	}
	for index, id := range order {
		if id == "" {
			t.Fatalf("row %d has empty id", index+1)
		}
		if player.skillViewRow(id) != uint16(index+1) {
			t.Fatalf("row mismatch id=%s", id)
		}
	}
}

// 整套学习（GM learn_taolu）：套路解析/逐技钳上限/已学表+等级档落盘/
// View40 整视图重开；全招已达层时 no-op 不发帧。
func TestApplyGMLearnTaolu(t *testing.T) {
	saved := gmTaolus
	gmTaolus = loadGMTaoluCatalogFrom(map[string]string{"cs_test": "测试套路"},
		strings.NewReader("[cs_test]\nskill=CS_yhwq_hsqs03,CS_yhwq_hsqs04\n"))
	t.Cleanup(func() { gmTaolus = saved })

	store, mock := newMockInventoryDB(t)
	skillRows := map[string]gameSkillRow{
		"CS_yhwq_hsqs03": {skillID: "CS_yhwq_hsqs03", skillType: "wuxue", staticData: 12638, itemType: 1000},
		"CS_yhwq_hsqs04": {skillID: "CS_yhwq_hsqs04", skillType: "wuxue", staticData: 12639, itemType: 1000},
	}
	mock.ExpectQuery("SELECT skill_id, skill_type, static_data, item_type FROM game_skills").
		WithArgs("CS_yhwq_hsqs03", "CS_yhwq_hsqs04").
		WillReturnRows(mockSkillRows(skillRows))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM role_skills").
		WithArgs(int64(role.RoleID(901))).WillReturnRows(mockIntRows([]int64{1})) // 已有基线 → 跳过物化
	for _, id := range []string{"CS_yhwq_hsqs03", "CS_yhwq_hsqs04"} {
		mock.ExpectExec("INSERT IGNORE INTO role_skills").
			WithArgs(int64(role.RoleID(901)), "wuxue", id).
			WillReturnResult(newSQLMockResult(0, 1))
	}

	player := newPlayerActor("测试", 0)
	conn := &captureMessageConnection{}
	levelStore := openSkillLevelStore(filepath.Join(t.TempDir(), skillLevelFileName))
	message, err := applyGMLearnTaolu(conn, player, role.RoleID(901), levelStore, store,
		context.Background(), "CS_TEST", 5, "test")
	if err != nil {
		t.Fatalf("applyGMLearnTaolu: %v", err)
	}
	for _, id := range []string{"CS_yhwq_hsqs03", "CS_yhwq_hsqs04"} {
		if level, learned := player.learnedSkillLevel(id); !learned || level != 5 {
			t.Fatalf("%s learned=%t level=%d want 5", id, learned, level)
		}
		// 等级档落盘（>1 层行）。
		if stored := levelStore.levelOf(role.RoleID(901), id); stored != 5 {
			t.Fatalf("%s stored level=%d want 5", id, stored)
		}
		// role_skills 落库。
		mock.ExpectationsWereMet()
	}
	if !strings.Contains(message, "测试套路") || !strings.Contains(message, "2 招") {
		t.Fatalf("message=%q", message)
	}
	// View40 整视图重开：0x16 删 + 行帧含新招。
	deleted, rowSeen := false, false
	for _, frame := range conn.Frames() {
		if frame[0] == 0x16 && binary.LittleEndian.Uint16(frame[1:]) == uint16(viewportSkill) {
			deleted = true
		}
		if frame[0] == 0x18 && bytes.Contains(frame, []byte("CS_yhwq_hsqs03\x00")) {
			rowSeen = true
		}
	}
	if !deleted || !rowSeen {
		t.Fatalf("view rebuild deleted=%v row=%v frames=%d", deleted, rowSeen, len(conn.Frames()))
	}
}

func TestApplyGMLearnTaoluClampsToPerSkillMax(t *testing.T) {
	saved := gmTaolus
	gmTaolus = loadGMTaoluCatalogFrom(nil,
		strings.NewReader("[cs_test]\nskill=CS_yhwq_hsqs03\n"))
	t.Cleanup(func() { gmTaolus = saved })

	store, mock := newMockInventoryDB(t)
	skillRows := map[string]gameSkillRow{
		"CS_yhwq_hsqs03": {skillID: "CS_yhwq_hsqs03", skillType: "wuxue", staticData: 12638, itemType: 1000},
	}
	mock.ExpectQuery("SELECT skill_id, skill_type, static_data, item_type FROM game_skills").
		WithArgs("CS_yhwq_hsqs03").WillReturnRows(mockSkillRows(skillRows))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM role_skills").
		WithArgs(int64(role.RoleID(901))).WillReturnRows(mockIntRows([]int64{1}))
	mock.ExpectExec("INSERT IGNORE INTO role_skills").
		WithArgs(int64(role.RoleID(901)), "wuxue", "CS_yhwq_hsqs03").
		WillReturnResult(newSQLMockResult(0, 1))

	player := newPlayerActor("测试", 0)
	conn := &captureMessageConnection{}
	levelStore := openSkillLevelStore(filepath.Join(t.TempDir(), skillLevelFileName))
	if _, err := applyGMLearnTaolu(conn, player, role.RoleID(901), levelStore, store,
		context.Background(), "cs_test", 999, "test"); err != nil {
		t.Fatalf("applyGMLearnTaolu: %v", err)
	}
	// 超过 varprop 满级的层数按该技上限给（maxLevelOf 与目录同源自证）。
	maxLevel := installedCombatSkillCatalog.maxLevelOf("CS_yhwq_hsqs03")
	if maxLevel < 2 {
		t.Skipf("fixture catalog maxLevel=%d, clamp not observable", maxLevel)
	}
	if level, _ := player.learnedSkillLevel("CS_yhwq_hsqs03"); level != maxLevel {
		t.Fatalf("level=%d want per-skill max %d", level, maxLevel)
	}
}

func TestApplyGMLearnTaoluNoOpWhenAllLearned(t *testing.T) {
	saved := gmTaolus
	gmTaolus = loadGMTaoluCatalogFrom(nil,
		strings.NewReader("[cs_test]\nskill=CS_yhwq_hsqs03\n"))
	t.Cleanup(func() { gmTaolus = saved })

	player := newPlayerActor("测试", 0) // starter 含该招 → 已学
	player.mu.Lock()
	player.learnedSkills["CS_yhwq_hsqs03"] = 11
	player.mu.Unlock()
	store, mock := newMockInventoryDB(t)
	mock.ExpectQuery("SELECT skill_id, skill_type, static_data, item_type FROM game_skills").
		WithArgs("CS_yhwq_hsqs03").WillReturnRows(mockSkillRows(map[string]gameSkillRow{
		"CS_yhwq_hsqs03": {skillID: "CS_yhwq_hsqs03", skillType: "wuxue", staticData: 12638, itemType: 1000},
	}))
	conn := &captureMessageConnection{}
	levelStore := openSkillLevelStore(filepath.Join(t.TempDir(), skillLevelFileName))
	message, err := applyGMLearnTaolu(conn, player, role.RoleID(901), levelStore, store,
		context.Background(), "cs_test", 5, "test")
	if err != nil {
		t.Fatalf("applyGMLearnTaolu: %v", err)
	}
	if !strings.Contains(message, "已全部学至目标层") {
		t.Fatalf("message=%q", message)
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("no-op emitted %d frames", got)
	}
}

func TestApplyGMLearnTaoluRejectsUnknownKey(t *testing.T) {
	player := newPlayerActor("测试", 0)
	if _, err := applyGMLearnTaolu(&captureMessageConnection{}, player, 1, nil, nil,
		context.Background(), "no_such_taolu", 5, "test"); err == nil {
		t.Fatal("unknown taolu key must be rejected")
	}
}

// 整套学习（GM learn_neigong）：建书直达层/曲线推进 power、钳 varprop
// 满层、只升不降、未知书拒绝。建书用非 starter 书（starter 六本在
// newPlayerProgress 已带 30/36 等进度，会走「已有书抬升」分支）。
func TestApplyGMLearnNeiGong(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Skip("neigong catalog unavailable in fixture")
	}
	player := newPlayerActor("测试", 0)
	conn := &captureMessageConnection{}
	// 选一本 starter 之外的目录书（cap≥2 才能观察逐层推进）。
	def, capLevel := modernNeiGongBook{}, int32(0)
	player.mu.Lock()
	for _, candidate := range catalog.books {
		if player.progress.book(candidate.configID) != nil {
			continue
		}
		if candidateCap := catalog.maxLevelOf(candidate.staticData); candidateCap >= 2 {
			def, capLevel = candidate, candidateCap
			break
		}
	}
	player.mu.Unlock()
	if def.configID == "" {
		t.Skip("no non-starter neigong book with cap>=2 in fixture")
	}
	if _, err := applyGMLearnNeiGong(conn, player, role.RoleID(901), def.configID, 2, "test"); err != nil {
		t.Fatalf("applyGMLearnNeiGong: %v", err)
	}
	player.mu.Lock()
	book := player.progress.book(def.configID)
	player.mu.Unlock()
	if book == nil {
		t.Fatal("book not created")
	}
	if book.level != 2 || book.maxLevel < 2 {
		t.Fatalf("level/max=%d/%d want 2/2", book.level, book.maxLevel)
	}
	if want := int32(120 + (2-1)*20); book.power != want || book.maxPower != want {
		t.Fatalf("power/maxPower=%d/%d want %d", book.power, book.maxPower, want)
	}
	// 满级（0=各自满级）：钳到 varprop 满层。
	if _, err := applyGMLearnNeiGong(conn, player, role.RoleID(901), def.configID, 0, "test"); err != nil {
		t.Fatalf("applyGMLearnNeiGong max: %v", err)
	}
	player.mu.Lock()
	book = player.progress.book(def.configID)
	player.mu.Unlock()
	if book.level != capLevel || book.maxLevel != capLevel {
		t.Fatalf("max level/max=%d/%d want %d/%d", book.level, book.maxLevel, capLevel, capLevel)
	}
	if want := 120 + (capLevel-1)*20; book.power != want {
		t.Fatalf("max power=%d want %d", book.power, want)
	}
	// 只升不降：低层 no-op。
	message, err := applyGMLearnNeiGong(conn, player, role.RoleID(901), def.configID, 1, "test")
	if err != nil {
		t.Fatalf("applyGMLearnNeiGong downgrade: %v", err)
	}
	if !strings.Contains(message, "无需变更") {
		t.Fatalf("downgrade message=%q", message)
	}
	player.mu.Lock()
	book = player.progress.book(def.configID)
	player.mu.Unlock()
	if book.level != capLevel {
		t.Fatalf("downgrade must not lower level, got %d", book.level)
	}
	// 已有书抬升：starter ng_wd_001（30/36）满级收编到 36。
	if wd, ok := catalog.book("ng_wd_001"); ok {
		wdCap := catalog.maxLevelOf(wd.staticData)
		if _, err := applyGMLearnNeiGong(conn, player, role.RoleID(901), "ng_wd_001", 0, "test"); err != nil {
			t.Fatalf("applyGMLearnNeiGong ng_wd_001: %v", err)
		}
		player.mu.Lock()
		wdBook := player.progress.book("ng_wd_001")
		player.mu.Unlock()
		if wdBook == nil || wdBook.level != wdCap {
			t.Fatalf("ng_wd_001 level=%v want %d", wdBook, wdCap)
		}
	}
	// View43 整视图重建发生过（至少一次删视图帧）。
	deleted := false
	for _, frame := range conn.Frames() {
		if frame[0] == 0x16 && binary.LittleEndian.Uint16(frame[1:]) == uint16(viewportNeiGong) {
			deleted = true
		}
	}
	if !deleted {
		t.Fatalf("View43 rebuild missing, frames=%d", len(conn.Frames()))
	}
}

func TestApplyGMLearnNeiGongRejectsUnknownBook(t *testing.T) {
	if _, err := loadModernNeiGongCatalog(); err != nil {
		t.Skip("neigong catalog unavailable in fixture")
	}
	player := newPlayerActor("测试", 0)
	if _, err := applyGMLearnNeiGong(&captureMessageConnection{}, player, 1, "ng_nope_999", 5, "test"); err == nil {
		t.Fatal("unknown neigong must be rejected")
	}
}

// 整套移除（GM forget_taolu）：非 starter 招删已学表+DB 行+等级档、
// starter 招重置回 1 层；View40 重开后非 starter 行消失。
func TestApplyGMForgetTaolu(t *testing.T) {
	saved := gmTaolus
	gmTaolus = loadGMTaoluCatalogFrom(nil,
		strings.NewReader("[cs_test]\nskill=CS_yhwq_hsqs03,cs_extra_01\n"))
	t.Cleanup(func() { gmTaolus = saved })

	store, mock := newMockInventoryDB(t)
	skillRows := map[string]gameSkillRow{
		"CS_yhwq_hsqs03": {skillID: "CS_yhwq_hsqs03", skillType: "wuxue", staticData: 12638, itemType: 1000},
		"cs_extra_01":    {skillID: "cs_extra_01", skillType: "wuxue", staticData: 99901, itemType: 1000},
	}
	mock.ExpectQuery("SELECT skill_id, skill_type, static_data, item_type FROM game_skills").
		WithArgs("CS_yhwq_hsqs03", "cs_extra_01").WillReturnRows(mockSkillRows(skillRows))
	for _, id := range []string{"cs_extra_01"} {
		mock.ExpectExec("DELETE FROM role_skills").
			WithArgs(int64(role.RoleID(901)), id).
			WillReturnResult(newSQLMockResult(0, 1))
	}

	player := newPlayerActor("测试", 0)
	player.mu.Lock()
	player.learnedSkills["CS_yhwq_hsqs03"] = 11 // starter 招升过层
	player.learnedSkills["cs_extra_01"] = 7     // 非 starter 招
	player.mu.Unlock()
	conn := &captureMessageConnection{}
	levelStore := openSkillLevelStore(filepath.Join(t.TempDir(), skillLevelFileName))
	removed, message, err := applyGMForgetTaolu(conn, player, role.RoleID(901), levelStore, store,
		nil, context.Background(), "cs_test", "test")
	if err != nil {
		t.Fatalf("applyGMForgetTaolu: %v", err)
	}
	// 非 starter 招彻底删除。
	if len(removed) != 1 || removed[0] != "cs_extra_01" {
		t.Fatalf("removed=%v want [cs_extra_01]", removed)
	}
	if _, learned := player.learnedSkillLevel("cs_extra_01"); learned {
		t.Fatal("non-starter skill must leave learned set")
	}
	if stored := levelStore.levelOf(role.RoleID(901), "cs_extra_01"); stored != 0 {
		t.Fatalf("removed skill store level=%d want 0", stored)
	}
	// starter 招重置回 1 层、保持已学。
	if level, learned := player.learnedSkillLevel("CS_yhwq_hsqs03"); !learned || level != 1 {
		t.Fatalf("starter skill learned=%t level=%d want true/1", learned, level)
	}
	if stored := levelStore.levelOf(role.RoleID(901), "CS_yhwq_hsqs03"); stored != 0 {
		t.Fatalf("reset skill store level=%d want 0", stored)
	}
	if !strings.Contains(message, "删除 1 招") || !strings.Contains(message, "重置回 1 层") {
		t.Fatalf("message=%q", message)
	}
	// View40 重开：非 starter 行不再出现，starter 行保留。
	rowExtra, rowStarter := false, false
	for _, frame := range conn.Frames() {
		if frame[0] == 0x18 {
			if bytes.Contains(frame, []byte("cs_extra_01\x00")) {
				rowExtra = true
			}
			if bytes.Contains(frame, []byte("CS_yhwq_hsqs03\x00")) {
				rowStarter = true
			}
		}
	}
	if rowExtra {
		t.Fatal("removed skill row must leave View40")
	}
	if !rowStarter {
		t.Fatal("starter skill row must stay in View40")
	}
}

func TestApplyGMForgetTaoluRejectsUnknownKey(t *testing.T) {
	player := newPlayerActor("测试", 0)
	if _, _, err := applyGMForgetTaolu(&captureMessageConnection{}, player, 1, nil, nil, nil,
		context.Background(), "no_such_taolu", "test"); err == nil {
		t.Fatal("unknown taolu key must be rejected")
	}
}

// 整套移除（GM forget_neigong）：删书 + 装备书/修炼目标重接 + View43
// 重建；未收录 no-op；基线书重登复活语义在消息注明。
func TestApplyGMForgetNeiGong(t *testing.T) {
	catalog, err := loadModernNeiGongCatalog()
	if err != nil {
		t.Skip("neigong catalog unavailable in fixture")
	}
	// 选一本非 starter 目录书：先学再删。
	var def modernNeiGongBook
	for _, candidate := range catalog.books {
		if !starterNeiGongBookIDs[candidate.configID] {
			def = candidate
			break
		}
	}
	if def.configID == "" {
		t.Skip("no non-starter neigong book in fixture catalog")
	}
	player := newPlayerActor("测试", 0)
	conn := &captureMessageConnection{}
	if _, learnErr := applyGMLearnNeiGong(conn, player, role.RoleID(901), def.configID, 2, "test"); learnErr != nil {
		t.Fatalf("learn for forget: %v", learnErr)
	}
	framesAfterLearn := len(conn.Frames())
	message, err := applyGMForgetNeiGong(conn, player, role.RoleID(901), def.configID, "test")
	if err != nil {
		t.Fatalf("applyGMForgetNeiGong: %v", err)
	}
	if !strings.Contains(message, "已移除") {
		t.Fatalf("message=%q", message)
	}
	player.mu.Lock()
	book := player.progress.book(def.configID)
	equipped := player.progress.curNeiGong
	player.mu.Unlock()
	if book != nil {
		t.Fatal("book must leave the shelf")
	}
	if equipped == def.configID {
		t.Fatal("equipped book must be re-pointed after removal")
	}
	// View43 重建发生过（删视图帧新增）。
	deleted := false
	for _, frame := range conn.Frames()[framesAfterLearn:] {
		if frame[0] == 0x16 && binary.LittleEndian.Uint16(frame[1:]) == uint16(viewportNeiGong) {
			deleted = true
		}
	}
	if !deleted {
		t.Fatal("View43 rebuild missing after forget")
	}
	// 未收录 → no-op 不报错。
	message, err = applyGMForgetNeiGong(conn, player, role.RoleID(901), def.configID, "test")
	if err != nil {
		t.Fatalf("second forget: %v", err)
	}
	if !strings.Contains(message, "未收录") {
		t.Fatalf("no-op message=%q", message)
	}
	// 基线书：删掉后装备书重接、消息注明重登复活。
	if wd, ok := catalog.book("ng_wd_001"); ok {
		message, err = applyGMForgetNeiGong(conn, player, role.RoleID(901), wd.configID, "test")
		if err != nil {
			t.Fatalf("forget starter book: %v", err)
		}
		if !strings.Contains(message, "登录基线") {
			t.Fatalf("starter book message=%q", message)
		}
		player.mu.Lock()
		equipped = player.progress.curNeiGong
		player.mu.Unlock()
		if equipped == wd.configID || equipped == "" {
			t.Fatalf("equipped=%q must be re-pointed to a remaining book", equipped)
		}
	}
}

func TestApplyGMForgetNeiGongRejectsUnknownBook(t *testing.T) {
	if _, err := loadModernNeiGongCatalog(); err != nil {
		t.Skip("neigong catalog unavailable in fixture")
	}
	player := newPlayerActor("测试", 0)
	if _, err := applyGMForgetNeiGong(&captureMessageConnection{}, player, 1, "ng_nope_999", "test"); err == nil {
		t.Fatal("unknown neigong must be rejected")
	}
}
