package main

import (
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

func registerTestCombatNPC(t *testing.T, lifecycle *sceneLifecycle, id uint32) {
	t.Helper()
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_test"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.x, npc.y, npc.z = 1, 0, 0
	if err := lifecycle.registerNPCOnRun(id, npc); err != nil {
		t.Fatal(err)
	}
}

// placeTestNPC 把已注册的测试 NPC 摆到指定 X/Z。注册点固定为
// (1,0,0)，因此显式写全两个坐标与原先只改 transform.X 的戳点
// 行为一致。
func placeTestNPC(t *testing.T, lifecycle *sceneLifecycle, id uint32, x, z float32) {
	t.Helper()
	transform, ok := lifecycle.npcTransform(worldcore.EntityID(id))
	if !ok {
		t.Fatalf("NPC %d has no transform", id)
	}
	transform.X, transform.Z = x, z
	lifecycle.writeNpcTransform(worldcore.EntityID(id), transform)
}

func TestCombatTickAggroesAndDamagesNearbyPlayer(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 81)
	player := newPlayerActor("tester", 0)
	before := player.stateSnapshot().HP
	result := lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, time.Unix(1000, 0))
	if !result.playerStateChanged {
		t.Fatal("nearby hostile did not produce a player state update")
	}
	if after := player.stateSnapshot().HP; after >= before {
		t.Fatalf("NPC attack HP=%d, want below %d", after, before)
	}
}

func TestCombatTickHonorsFreshActiveParry(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 82)
	player := newPlayerActor("tester", 0)
	now := time.Unix(2000, 0)
	player.setActiveParry(activeParryRequest{Subtype: 0, At: now})
	before := player.stateSnapshot().HP
	result := lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, now)
	if after := player.stateSnapshot().HP; after != before {
		t.Fatalf("parry HP=%d, want %d", after, before)
	}
	if !result.playerStateChanged || player.stateSnapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("fresh parry must still enter combat, result=%+v actor=%+v", result, player.stateSnapshot())
	}
}

func TestCombatThreatInterruptsSitcrossAndClearsAfterLeash(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 83)
	player := newPlayerActor("tester", 0)
	player.setLogicState(entity.LogicStateSitcross)
	now := time.Unix(2500, 0)
	result := lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, now)
	if !result.playerStateChanged || player.stateSnapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("aggro state result=%+v actor=%+v", result, player.stateSnapshot())
	}
	result = lifecycle.combatTickForTest(player, role.Position{X: 100, Y: 0, Z: 0}, now.Add(time.Second))
	if !result.playerStateChanged || player.stateSnapshot().LogicState != entity.LogicStateNormal {
		t.Fatalf("leash state result=%+v actor=%+v", result, player.stateSnapshot())
	}
	lifecycle.close()
}

func TestCombatTickRespawnsDeadHostileAtHome(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 83)
	id := worldcore.EntityID(83)
	vitals, ok := lifecycle.run.combat.Vitals(id)
	if !ok {
		t.Fatal("combat NPC must have vitals")
	}
	vitals.ApplyDamage(999999)
	lifecycle.run.combat.WriteVitals(id, vitals)
	brain, ok := lifecycle.run.combat.Brain(id)
	if !ok {
		t.Fatal("combat NPC must have a brain")
	}
	now := time.Unix(3000, 0)
	brain.RespawnAt = now
	lifecycle.run.combat.WriteBrain(id, brain)
	player := newPlayerActor("tester", 0)
	lifecycle.combatTickForTest(player, role.Position{X: 999, Y: 0, Z: 999}, now)
	if revived, _ := lifecycle.run.combat.Vitals(id); revived.HP != revived.MaxHP || revived.Logic != entity.LogicStateNormal {
		t.Fatalf("respawn vitals=%+v", revived)
	}
	frames := conn.Frames()
	if len(frames) != 2 || frames[0][0] != 0x20 || frames[1][0] != 0x10 {
		t.Fatalf("respawn frames=%x", frames)
	}
}

func TestCombatMoveDoesNotImmediatelyConfirmLocation(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 84)
	player := newPlayerActor("tester", 0)
	lifecycle.combatTickForTest(player, role.Position{X: 10, Y: 0, Z: 0}, time.Unix(4000, 0))
	frames := conn.Frames()
	if len(frames) != 1 || frames[0][0] != 0x20 {
		t.Fatalf("first chase tick frames=%x, want only ServerMoving", frames)
	}
	lifecycle.close()
}
