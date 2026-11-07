package main

import (
	"testing"

	"github.com/local/9yin-go-server/internal/entity"
)

func TestQingGongMotionPropertiesUseModernRushRangeConfig(t *testing.T) {
	want := map[uint16]float32{
		873: 5.0, // GRushRange
		876: 1.5, // GRushRangeAdd
		874: 8.0, // ARushRange
		877: 6.5, // ARushRangeAdd
		875: 8.0, // WRushRange
		878: 6.5, // WRushRangeAdd
	}
	got := make(map[uint16]float32)
	for _, property := range qinggongMotionProperties() {
		if _, needed := want[property.Index]; needed {
			got[property.Index] = property.Value.F32
		}
	}
	for index, expected := range want {
		if actual, ok := got[index]; !ok || actual != expected {
			t.Fatalf("property %d = %v, want %v", index, actual, expected)
		}
	}
}

func TestRageRecoveryTickDisabledByDefaultAndFollowsGMRate(t *testing.T) {
	player := newPlayerActor("tester", 0)
	// 默认 0:不自动回复,满池期/非满池期都不推帧。
	if _, _, err := player.setSP(40); err != nil {
		t.Fatalf("stage drained SP: %v", err)
	}
	if player.rageRecoveryTick() {
		t.Fatal("default rate 0 must not recover rage")
	}
	if got := player.skillSP(); got != 40 {
		t.Fatalf("SP mutated at rate 0: %d", got)
	}
	// GM 设为官方满档 5 后,回复不限战斗态,脱战照样 +5/秒
	// (rage-recovery-frames.md §5)。
	if got := player.setRageRecovery(5); got != 5 {
		t.Fatalf("setRageRecovery=%d, want 5", got)
	}
	if !player.rageRecoveryTick() {
		t.Fatal("normal-state actor must recover rage")
	}
	player.setLogicState(entity.LogicStateFighting)
	if !player.rageRecoveryTick() {
		t.Fatal("fighting actor should recover rage")
	}
	if got := player.skillSP(); got != 40+2*5 {
		t.Fatalf("SP after two ticks = %d, want %d", got, 40+2*5)
	}
	player.setLogicState(entity.LogicStateSitcross)
	if !player.rageRecoveryTick() {
		t.Fatal("sitting actor still recovers rage (state-independent)")
	}
	player.setLogicState(entity.LogicStateDied)
	if player.rageRecoveryTick() {
		t.Fatal("dead actor must not recover rage")
	}
	// 负值护栏:钳到 0。
	if got := player.setRageRecovery(-3); got != 0 {
		t.Fatalf("setRageRecovery(-3)=%d, want 0", got)
	}
}

func TestRageRecoveryTickSaturatesAtMaxPlayerSP(t *testing.T) {
	player := newPlayerActor("tester", 0)
	player.setRageRecovery(5)
	if _, _, err := player.setSP(player.rageCap() - 3); err != nil {
		t.Fatalf("stage partial SP: %v", err)
	}
	if !player.rageRecoveryTick() {
		t.Fatal("partial rage should still advance toward the cap")
	}
	if got := player.skillSP(); got != player.rageCap() {
		t.Fatalf("SP must saturate at %d, got %d", player.rageCap(), got)
	}
	if player.rageRecoveryTick() {
		t.Fatal("capped rage must not report progress")
	}
}

// 释放闸内存表必须与 View40 的 merge 集同口径：DB 新行补齐可施放，
// 与 starter 重叠的 configID 保持 starter 条目（不被 DB 覆盖）。
// 回归背景：登录只把 role_skills 行喂给 View40、没并入
// learnedSkills，导致 DB 学的技能客户端可拖栏但服务端按「未学」拒绝。
func TestMergeDBLearnedSkillsFeedsCastGate(t *testing.T) {
	player := newPlayerActor("tester", 0)
	// 与真实 learnedSkills 查询同口径：非空 ID、level 恒 1（dbSkillViewRow）。
	dbRows := []learnedSkillView{
		{configID: "CS_jh_hyjf01", staticData: 13821, itemType: 1000, level: 1}, // DB 新行
		{configID: "CS_wd_tjq01", staticData: 4057, itemType: 1000, level: 1},   // 与 starter 重叠
		{configID: "CS_jh_tslyz01", staticData: 13851, itemType: 1000, level: 1},
	}
	player.mergeDBLearnedSkills(dbRows)
	if _, ok := player.learnedSkillLevel("CS_jh_hyjf01"); !ok {
		t.Fatal("db row CS_jh_hyjf01 must satisfy the cast gate after merge")
	}
	if level, ok := player.learnedSkillLevel("CS_wd_tjq01"); !ok || level != 1 {
		t.Fatalf("overlap row must keep starter entry, got level=%d ok=%v", level, ok)
	}
	// View40 行数与内存表口径一致性：merge 集去重后每行都应过闸。
	merged := mergeLearnedSkills(dbRows)
	for _, row := range merged {
		if _, ok := player.learnedSkillLevel(row.configID); !ok {
			t.Fatalf("merged view row %s missing from cast gate", row.configID)
		}
	}
	// 防御性边界：空 ID 与非法等级不入闸。
	player.mergeDBLearnedSkills([]learnedSkillView{{configID: "", level: 1}, {configID: "CS_bad", level: 0}})
	if _, ok := player.learnedSkillLevel("CS_bad"); ok {
		t.Fatal("level 0 row must be dropped")
	}
}
