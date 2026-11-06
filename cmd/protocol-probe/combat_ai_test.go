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
	if err := lifecycle.registerNPC(id, npc); err != nil {
		t.Fatal(err)
	}
	lifecycle.combatActive = true
}

func TestCombatTickAggroesAndDamagesNearbyPlayer(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 81)
	player := newPlayerActor("tester", 0)
	before := player.actor.Snapshot().HP
	result, err := lifecycle.combatTick(player, role.Position{X: 1, Y: 0, Z: 0}, time.Unix(1000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !result.playerStateChanged {
		t.Fatal("nearby hostile did not produce a player state update")
	}
	if after := player.actor.Snapshot().HP; after >= before {
		t.Fatalf("NPC attack HP=%d, want below %d", after, before)
	}
}

func TestCombatTickHonorsFreshActiveParry(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 82)
	player := newPlayerActor("tester", 0)
	now := time.Unix(2000, 0)
	player.setActiveParry(activeParryRequest{Subtype: 0, At: now})
	before := player.actor.Snapshot().HP
	result, err := lifecycle.combatTick(player, role.Position{X: 1, Y: 0, Z: 0}, now)
	if err != nil {
		t.Fatal(err)
	}
	if after := player.actor.Snapshot().HP; after != before {
		t.Fatalf("parry HP=%d, want %d", after, before)
	}
	if !result.playerStateChanged || player.actor.Snapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("fresh parry must still enter combat, result=%+v actor=%+v", result, player.actor.Snapshot())
	}
}

func TestCombatThreatInterruptsSitcrossAndClearsAfterLeash(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 83)
	player := newPlayerActor("tester", 0)
	player.actor.SetLogicState(entity.LogicStateSitcross)
	now := time.Unix(2500, 0)
	result, err := lifecycle.combatTick(player, role.Position{X: 1, Y: 0, Z: 0}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !result.playerStateChanged || player.actor.Snapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("aggro state result=%+v actor=%+v", result, player.actor.Snapshot())
	}
	result, err = lifecycle.combatTick(player, role.Position{X: 100, Y: 0, Z: 0}, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !result.playerStateChanged || player.actor.Snapshot().LogicState != entity.LogicStateNormal {
		t.Fatalf("leash state result=%+v actor=%+v", result, player.actor.Snapshot())
	}
	lifecycle.close()
}

func TestCombatTickRespawnsDeadHostileAtHome(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 83)
	id := worldcore.EntityID(83)
	vitals, ok := lifecycle.combat.Vitals(id)
	if !ok {
		t.Fatal("combat NPC must have vitals")
	}
	vitals.ApplyDamage(999999)
	lifecycle.combat.WriteVitals(id, vitals)
	brain, ok := lifecycle.combat.Brain(id)
	if !ok {
		t.Fatal("combat NPC must have a brain")
	}
	now := time.Unix(3000, 0)
	brain.RespawnAt = now
	lifecycle.combat.WriteBrain(id, brain)
	player := newPlayerActor("tester", 0)
	if _, err := lifecycle.combatTick(player, role.Position{X: 999, Y: 0, Z: 999}, now); err != nil {
		t.Fatal(err)
	}
	if revived, _ := lifecycle.combat.Vitals(id); revived.HP != revived.MaxHP || revived.Logic != entity.LogicStateNormal {
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
	if err := lifecycle.begin("scene", "resource"); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 84)
	player := newPlayerActor("tester", 0)
	if _, err := lifecycle.combatTick(player, role.Position{X: 10, Y: 0, Z: 0}, time.Unix(4000, 0)); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 1 || frames[0][0] != 0x20 {
		t.Fatalf("first chase tick frames=%x, want only ServerMoving", frames)
	}
	lifecycle.close()
}
