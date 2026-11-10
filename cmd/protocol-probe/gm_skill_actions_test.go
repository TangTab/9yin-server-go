package main

// GM 武学三动作单测：等级档存档读写/原子性、totalFillDisplay 口径、
// 等级行帧编码、登录档值合并（applyStoredSkillLevels）。

import (
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
