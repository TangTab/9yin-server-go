package main

import (
	"math"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// newDummyTestRun 构造空目录 + 单成员视图的共享运行时（训练木桩测试）。
func newDummyTestRun(t *testing.T) (*sceneRun, *sceneLifecycle, *captureMessageConnection) {
	t.Helper()
	conn := &captureMessageConnection{}
	view := newSceneLifecycle("test", conn)
	view.radius = 50
	run := newSceneRun(sceneKey{config: "scene", resource: "resource"}, nil)
	if err := run.populate(nil); err != nil {
		t.Fatal(err)
	}
	view.run = run
	player := newPlayerActor("tester", 0)
	view.attachMember(player, role.Position{X: 1}, roleVisual{})
	run.activateMember(view.wireID)
	view.refreshVisibility(nil, role.Position{X: 1})
	return run, view, conn
}

// TestResolveDummySpawnCarriesCombatShape 木桩模板（资源在场走官方表，
// 缺席走手工回落）必须保住战斗语义与本次参数：AttackNpc 交互类 +
// MaxHP/HP=指定血量 + 显示名。
func TestResolveDummySpawnCarriesCombatShape(t *testing.T) {
	spawn, err := resolveDummySpawn(250000, "训练木桩·2")
	if err != nil {
		t.Fatal(err)
	}
	if spawn.resolved.ScriptClass != "AttackNpc" {
		t.Fatalf("dummy script class = %q, want AttackNpc", spawn.resolved.ScriptClass)
	}
	if got := spawn.resolved.Properties["MaxHP"].Value.I32; got != 250000 {
		t.Fatalf("dummy MaxHP = %d, want 250000", got)
	}
	if got := spawn.resolved.Properties["HP"].Value.I32; got != 250000 {
		t.Fatalf("dummy HP = %d, want 250000", got)
	}
	if got := spawn.resolved.Properties["Name"].Value.Text; got != "训练木桩·2" {
		t.Fatalf("dummy Name = %q, want 训练木桩·2", got)
	}
}

// TestSpawnTrainingDummiesRing 单只摆在面前、多只环形均布；实体注册
// （Vitals 满血 + 阉割 Brain）与清除登记一并到位。
func TestSpawnTrainingDummiesRing(t *testing.T) {
	run, view, _ := newDummyTestRun(t)
	defer view.close()

	center := role.Position{X: 10, Y: 0, Z: 20, Orient: 0}
	ids := run.spawnTrainingDummiesLocked(center, 80000, 1)
	if len(ids) != 1 {
		t.Fatalf("spawned %d dummies, want 1", len(ids))
	}
	entityID := worldcore.EntityID(ids[0])
	transform, ok := run.npcTransform(entityID)
	if !ok {
		t.Fatal("single dummy not registered")
	}
	// 面前 2.5m：orient=0 → +X 方向。
	if d := math.Abs(float64(transform.X - (center.X + 2.5))); d > 0.01 {
		t.Fatalf("single dummy X=%.3f, want ~%.3f (front)", transform.X, center.X+2.5)
	}
	vitals, _ := run.npcVitals(entityID)
	if vitals.MaxHP != 80000 || vitals.HP != 80000 {
		t.Fatalf("dummy vitals = %+v, want full 80000", vitals)
	}
	brain, _ := run.combat.Brain(entityID)
	if !brain.Passive || brain.ChaseRange != 0 || brain.AttackRange >= 0 || brain.AggroRange != 0 {
		t.Fatalf("dummy brain not neutered: %+v", brain)
	}
	if _, tracked := run.gmDummies[ids[0]]; !tracked {
		t.Fatal("dummy not tracked in gmDummies")
	}

	// 多只：环形均布半径 3m，朝向角色（首只对齐角色朝向 orient=0 → +X）。
	multi := run.spawnTrainingDummiesLocked(center, 80000, 4)
	if len(multi) != 4 {
		t.Fatalf("spawned %d dummies, want 4", len(multi))
	}
	seen := map[uint32]bool{}
	for _, id := range multi {
		if seen[id] {
			t.Fatalf("duplicate dummy id %d", id)
		}
		seen[id] = true
		transform, ok := run.npcTransform(worldcore.EntityID(id))
		if !ok {
			t.Fatalf("dummy %d not registered", id)
		}
		dx := transform.X - center.X
		dz := transform.Z - center.Z
		if dist := math.Sqrt(float64(dx*dx + dz*dz)); math.Abs(dist-3) > 0.01 {
			t.Fatalf("ring dummy id=%d dist=%.3f, want 3.0", id, dist)
		}
		// 均布：相邻夹角 90°。
		angle := math.Atan2(float64(dz), float64(dx))
		if math.Abs(angle-math.Round(angle/(math.Pi/2))*(math.Pi/2)) > 0.01 {
			t.Fatalf("ring dummy id=%d angle=%.4f not on quadrant marks", id, angle)
		}
	}
	firstTransform, _ := run.npcTransform(worldcore.EntityID(multi[0]))
	if firstTransform.X < center.X {
		t.Fatalf("first ring dummy should align with facing (+X), got X=%.3f", firstTransform.X)
	}
}

// TestDummyIsPureSandbag 受击掉血但不还手不移动：伤害路径锁定的攻击者
// 在下一 tick 即按栓绳语义清除（ChaseRange=0），木桩坐标不动。
func TestDummyIsPureSandbag(t *testing.T) {
	run, view, conn := newDummyTestRun(t)
	defer view.close()

	center := role.Position{X: 1, Y: 0, Z: 0, Orient: 0}
	ids := run.spawnTrainingDummiesLocked(center, 100000, 1)
	run.flushPendingObjectFrames()
	if len(ids) != 1 {
		t.Fatalf("spawned %d dummies, want 1", len(ids))
	}
	id := ids[0]
	before, _ := run.npcTransform(worldcore.EntityID(id))

	now := time.Unix(10000, 0)
	outcome, err := view.applySkillDamageToNPC(id, run.gmDummies[id], 500, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.killed || outcome.frame == nil {
		t.Fatalf("damage outcome unexpected: %+v", outcome)
	}
	vitals, _ := run.npcVitals(worldcore.EntityID(id))
	if vitals.HP != 100000-500 {
		t.Fatalf("dummy HP after hit = %d, want %d", vitals.HP, 100000-500)
	}
	brain, _ := run.combat.Brain(worldcore.EntityID(id))
	if brain.TargetMember == 0 {
		t.Fatal("damage path did not lock attacker into TargetMember")
	}

	conn.frames = nil
	run.tickOnce(now.Add(time.Second))
	brain, _ = run.combat.Brain(worldcore.EntityID(id))
	if brain.TargetMember != 0 {
		t.Fatalf("dummy still holds target after tick: %+v", brain)
	}
	after, _ := run.npcTransform(worldcore.EntityID(id))
	if after != before {
		t.Fatalf("dummy moved: before=%+v after=%+v", before, after)
	}
	for _, frame := range conn.Frames() {
		// msg337（0x0151 小端首字节 0x51 0x01）是 NPC 出招预备帧，纯沙包
		// 一个都不该出现。
		if len(frame) >= 2 && frame[0] == 0x51 && frame[1] == 0x01 {
			t.Fatalf("dummy emitted attack presentation frame % x", frame[:4])
		}
	}
}

// TestDummyRevivesAfterDeath 打空血走通用重生链：15 秒后原地满血复活。
func TestDummyRevivesAfterDeath(t *testing.T) {
	run, view, _ := newDummyTestRun(t)
	defer view.close()

	ids := run.spawnTrainingDummiesLocked(role.Position{X: 1}, 1000, 1)
	id := ids[0]
	now := time.Unix(10000, 0)
	if outcome, err := view.applySkillDamageToNPC(id, run.gmDummies[id], 1000, nil, now); err != nil || !outcome.killed {
		t.Fatalf("dummy did not die: outcome=%+v err=%v", outcome, err)
	}

	run.tickOnce(now.Add(14 * time.Second))
	if vitals, _ := run.npcVitals(worldcore.EntityID(id)); vitals.HP != 0 {
		t.Fatalf("dummy revived early: %+v", vitals)
	}
	run.tickOnce(now.Add(16 * time.Second))
	vitals, _ := run.npcVitals(worldcore.EntityID(id))
	if vitals.HP != vitals.MaxHP || vitals.MaxHP != 1000 {
		t.Fatalf("dummy not revived full: %+v", vitals)
	}
	if brain, _ := run.combat.Brain(worldcore.EntityID(id)); !brain.Passive || brain.ChaseRange != 0 {
		t.Fatalf("revived dummy brain lost neutering: %+v", brain)
	}
}

// TestClearTrainingDummies 清除全部木桩：实体注销、成员收到 0x0E、
// 重复清除归零。
func TestClearTrainingDummies(t *testing.T) {
	run, view, conn := newDummyTestRun(t)
	defer view.close()

	ids := run.spawnTrainingDummiesLocked(role.Position{X: 1}, 50000, 3)
	run.flushPendingObjectFrames()
	if len(ids) != 3 {
		t.Fatalf("spawned %d dummies, want 3", len(ids))
	}
	conn.frames = nil
	if cleared := run.clearTrainingDummies(); cleared != 3 {
		t.Fatalf("cleared %d dummies, want 3", cleared)
	}
	for _, id := range ids {
		if _, exists := run.npcVitals(worldcore.EntityID(id)); exists {
			t.Fatalf("dummy %d still has vitals after clear", id)
		}
		if _, tracked := run.gmDummies[id]; tracked {
			t.Fatalf("dummy %d still tracked after clear", id)
		}
	}
	removes := 0
	for _, frame := range conn.Frames() {
		if len(frame) > 0 && frame[0] == 0x0E {
			removes++
		}
	}
	if removes != 3 {
		t.Fatalf("member received %d remove frames, want 3", removes)
	}
	if again := run.clearTrainingDummies(); again != 0 {
		t.Fatalf("second clear removed %d, want 0", again)
	}
}
