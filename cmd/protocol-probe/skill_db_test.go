package main

import (
	"context"
	"errors"
	"math"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/local/9yin-go-server/internal/role"
)

const skillTestRoleID = role.RoleID(618)

func newMockSkillDB(t *testing.T) (*skillDBStore, sqlmock.Sqlmock) {
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
	return newSkillDBStore(db), mock
}

// SQL 与 Java RoleInventoryStore.skills 逐字对齐：wuxue/anqi 域、
// JOIN game_skills 富化 StaticData/ItemType。
func TestSkillDBLearnedSkillsJoinsGameSkills(t *testing.T) {
	store, mock := newMockSkillDB(t)
	mock.ExpectQuery(`SELECT s\.skill_id, s\.skill_type, g\.static_data, g\.item_type` +
		` FROM role_skills s JOIN game_skills g ON g\.skill_id = s\.skill_id` +
		` WHERE s\.role_id = \? AND s\.skill_type IN \('wuxue', 'anqi'\) ORDER BY s\.skill_id`).
		WithArgs(skillTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"skill_id", "skill_type", "static_data", "item_type"}).
			AddRow("CS_by_xhmdl01", "wuxue", 13701, 1000).
			AddRow("CS_gb_dgbf02", "wuxue", 13100, 0). // itemType 0 → Java 同款 1000 兜底
			AddRow("clone001_yuancheng", "anqi", 4150, 1006))
	skills, err := store.learnedSkills(context.Background(), skillTestRoleID)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 3 {
		t.Fatalf("skills=%d, want 3", len(skills))
	}
	first := skills[0]
	if first.configID != "CS_by_xhmdl01" || first.staticData != 13701 || first.itemType != 1000 || first.level != 1 {
		t.Fatalf("first=%+v", first)
	}
	if skills[1].itemType != 1000 {
		t.Fatalf("itemType 0 must normalize to 1000, got %d", skills[1].itemType)
	}
	if skills[2].itemType != 1006 {
		t.Fatalf("anqi itemType must pass through, got %d", skills[2].itemType)
	}
}

func TestSkillDBLearnedSkillsEmptyAndError(t *testing.T) {
	store, mock := newMockSkillDB(t)
	mock.ExpectQuery(`FROM role_skills`).WithArgs(skillTestRoleID).
		WillReturnRows(sqlmock.NewRows([]string{"skill_id", "skill_type", "static_data", "item_type"}))
	skills, err := store.learnedSkills(context.Background(), skillTestRoleID)
	if err != nil || len(skills) != 0 {
		t.Fatalf("empty table: skills=%d err=%v", len(skills), err)
	}

	store2, mock2 := newMockSkillDB(t)
	mock2.ExpectQuery(`FROM role_skills`).WithArgs(skillTestRoleID).
		WillReturnError(errors.New("db down"))
	if _, err := store2.learnedSkills(context.Background(), skillTestRoleID); err == nil {
		t.Fatal("query error must propagate to caller (skip-to-starter path)")
	}
}

// DB 行的 PauseTime 必须与 starter 目录同源（花神七式 01 ≈43 帧，防止
// 两套动作时间线口径漂移）；MaxLevel 刻意 = Level（Java dbSkillViewAdd
// 对 DB 行的 1/1 自洽显示，varprop 区间会高估——已装客户端实证花神 01
// 上限 11 而区间到 19）。
func TestDBSkillViewRowMatchesStarterMetadata(t *testing.T) {
	if installedCombatSkillCatalog == nil {
		t.Skip("combat skill catalog unavailable")
	}
	row := dbSkillViewRow("CS_yhwq_hsqs01", 12636, 1000)
	if row.maxLevel != 1 {
		t.Fatalf("CS_yhwq_hsqs01 MaxLevel=%d, want 1 (Java dbSkillViewAdd parity)", row.maxLevel)
	}
	wantPause := 43 * 1000.0 / 30
	if math.Abs(float64(row.pauseTime)-wantPause) > 2 {
		t.Fatalf("CS_yhwq_hsqs01 PauseTime=%f, want ≈%f", row.pauseTime, wantPause)
	}
	if row.level != 1 || row.fill != 0 || row.total != 0 {
		t.Fatalf("db row progress fields must start at level=1/fill=0/total=0: %+v", row)
	}
}

// 对实库 role_skills JOIN game_skills 的只读校验（不写库）：取已学
// wuxue/anqi 行最多的角色，learnedSkills 读出的每一行 View 40 必填
// 字段非零，且与 starter 合并后超集。NINEYIN_TEST_MYSQL_DSN 未设置
// 时跳过。
func TestSkillDBLearnedSkillsLiveRows(t *testing.T) {
	dsn := os.Getenv("NINEYIN_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("NINEYIN_TEST_MYSQL_DSN not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	repository, err := role.OpenMySQLRepository(ctx, dsn, role.MySQLConfig{})
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer repository.Close()
	store := newSkillDBStore(repository.DB())
	var roleID role.RoleID
	if err := store.db.QueryRowContext(ctx,
		`SELECT role_id FROM role_skills WHERE skill_type IN ('wuxue','anqi') `+
			`GROUP BY role_id ORDER BY COUNT(*) DESC LIMIT 1`).Scan(&roleID); err != nil {
		t.Skipf("no wuxue/anqi rows in live db: %v", err)
	}
	skills, err := store.learnedSkills(ctx, roleID)
	if err != nil {
		t.Fatalf("learned skills role=%d: %v", roleID, err)
	}
	if len(skills) == 0 {
		t.Skip("role has no joined rows")
	}
	for _, skill := range skills {
		if skill.configID == "" || skill.staticData <= 0 || skill.itemType == 0 || skill.level != 1 {
			t.Fatalf("malformed view row: %+v", skill)
		}
	}
	merged := mergeLearnedSkills(skills)
	if len(merged) <= len(starterSkillViews) {
		t.Fatalf("merged=%d must exceed starter=%d when db rows exist", len(merged), len(starterSkillViews))
	}
	t.Logf("live role=%d db_skills=%d merged=%d", roleID, len(skills), len(merged))
}
