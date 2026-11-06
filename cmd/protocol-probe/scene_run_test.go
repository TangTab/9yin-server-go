package main

import (
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
)

// TestSharedSceneRunServesTwoViews 验证共享场景运行时的核心承诺：
// 同一场景的两个视图共享一份 NPC 模拟——仇恨只结算一次、帧只发给
// 视口内可见该 NPC 的成员、NPC 状态对两个视图一致。
func TestSharedSceneRunServesTwoViews(t *testing.T) {
	connA := &captureMessageConnection{}
	connB := &captureMessageConnection{}
	viewA := newSceneLifecycle("a", connA)
	viewB := newSceneLifecycle("b", connB)
	viewB.memberID = 2
	viewA.radius = 50
	viewB.radius = 50

	catalog := []npcSpawn{testCombatCatalogNPC(2)}
	run := newSceneRun(sceneKey{config: "scene", resource: "resource"}, nil)
	if err := run.populate(catalog); err != nil {
		t.Fatal(err)
	}
	viewA.run = run
	viewB.run = run

	playerA := newPlayerActor("a", 0)
	playerB := newPlayerActor("b", 0)
	// X=10：仇恨半径 12 之内、攻击距离 3.5 之外——首个 tick 是追击，
	// 因此 A 会收到 0x20 而不是立即近身攻击。
	near := role.Position{X: 10, Y: 0, Z: 0}
	far := role.Position{X: 400, Y: 0, Z: 400}
	viewA.attachMember(playerA, near)
	viewB.attachMember(playerB, far)
	// 生产路径由就绪屏障（clientActivity）置位；测试直挂后显式激活。
	run.activateMember(viewA.memberID)
	run.activateMember(viewB.memberID)
	// A 视口内含 NPC；B 距离 400 出半径，不可见。
	viewA.refreshVisibility(catalog, near)
	viewB.refreshVisibility(catalog, far)
	if !viewA.npcVisible(2) {
		t.Fatal("NPC 2 must be visible to the nearby view")
	}
	if viewB.npcVisible(2) {
		t.Fatal("NPC 2 must not be visible to the distant view")
	}

	beforeA := playerA.stateSnapshot().HP
	beforeB := playerB.stateSnapshot().HP
	now := time.Unix(5000, 0)
	// 追击（两步）→ 近身 → 攻击；伤害只应落在最近存活成员 A 身上。
	for step := 0; step < 3; step++ {
		run.combatTick(now.Add(time.Duration(step) * 2 * time.Second))
	}
	if after := playerA.stateSnapshot().HP; after >= beforeA {
		t.Fatalf("nearby member HP=%d, want damage after chase", after)
	}
	if after := playerB.stateSnapshot().HP; after != beforeB {
		t.Fatalf("distant member HP changed: %d -> %d", beforeB, after)
	}
	// 帧只扇出给可见的 A。
	if len(connB.Frames()) != 0 {
		t.Fatalf("distant view received %d frames, want none", len(connB.Frames()))
	}
	if len(connA.Frames()) == 0 {
		t.Fatal("nearby view received no combat frames")
	}

	// 共享状态：A 侧击杀 NPC，B 的下一次对账能看到其权威坐标
	// （复活回 home）而不是各自私有的副本。
	vitals, ok := run.combat.Vitals(2)
	if !ok {
		t.Fatal("NPC 2 must have vitals in the shared run")
	}
	for vitals.HP > 0 {
		vitals.ApplyDamage(500)
		run.combat.WriteVitals(2, vitals)
	}
	brain, _ := run.combat.Brain(2)
	brain.RespawnAt = now.Add(time.Second)
	run.combat.WriteBrain(2, brain)
	run.combatTick(now.Add(2 * time.Second))
	revivedA, _ := run.combat.Vitals(2)
	revivedB, _ := run.combat.Vitals(2)
	if revivedA.HP != revivedA.MaxHP || revivedB.HP != revivedB.MaxHP {
		t.Fatalf("shared respawn state A=%+v B=%+v", revivedA, revivedB)
	}
	if revivedA.Logic != entity.LogicStateNormal {
		t.Fatalf("respawned NPC logic=%d", revivedA.Logic)
	}

	// B 走近后对账：出/入半径由各视图自己的过滤快照结算。
	walkClose := role.Position{X: 2, Y: 0, Z: 0}
	viewB.updateMemberPosition(walkClose, catalog)
	if !viewB.npcVisible(2) {
		t.Fatal("NPC 2 must enter the approaching view's visibility set")
	}
	viewA.close()
	viewB.close()
}

// testCombatCatalogNPC 构造目录下标 idx 的敌对 NPC（实体 ID = idx+2）。
func testCombatCatalogNPC(idx int) npcSpawn {
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_shared"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.x, npc.y, npc.z = 1, 0, 0
	_ = idx
	return npc
}

// BenchmarkScenePopulate 测量共享场景一次性全量注册的真实成本
// （含属性解析、payload 编码与四组件写入）。
func BenchmarkScenePopulate(b *testing.B) {
	base := testNPCSpawn()
	base.resolved.ScriptClass = "AttackNpc"
	catalog := make([]npcSpawn, 500)
	for i := range catalog {
		spawn := base
		catalog[i] = spawn
	}
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		run := newSceneRun(sceneKey{config: "bench", resource: "bench"}, nil)
		if err := run.populate(catalog); err != nil {
			b.Fatal(err)
		}
	}
}
