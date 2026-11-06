package main

import (
	"testing"

	"github.com/local/9yin-go-server/internal/entity"
)

func TestQingGongMotionPropertiesUseModernRushRangeConfig(t *testing.T) {
	want := map[uint16]float32{
		137: 5.0,
		138: 1.5,
		139: 8.0,
		140: 6.5,
		141: 8.0,
		142: 6.5,
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

func TestCombatRageTickAddsFivePerSecondWhileFighting(t *testing.T) {
	player := newPlayerActor("tester", 0)
	// Spawn defaults to a full SP pool, so start from a drained, known value.
	if _, _, err := player.setSP(40); err != nil {
		t.Fatalf("stage drained SP: %v", err)
	}
	if player.combatRageTick() {
		t.Fatal("normal-state actor must not recover rage")
	}
	player.actor.SetLogicState(entity.LogicStateFighting)
	if !player.combatRageTick() {
		t.Fatal("fighting actor should recover rage")
	}
	if got := player.skillSP(); got != 40+combatRageRecoveryPerSecond {
		t.Fatalf("SP after one combat tick = %d, want %d", got, 40+combatRageRecoveryPerSecond)
	}
	player.actor.SetLogicState(entity.LogicStateDied)
	if player.combatRageTick() {
		t.Fatal("dead actor must not recover rage")
	}
	player.actor.SetLogicState(entity.LogicStateSitcross)
	if player.combatRageTick() {
		t.Fatal("sitting actor must not recover rage")
	}
}

func TestCombatRageTickSaturatesAtMaxPlayerSP(t *testing.T) {
	player := newPlayerActor("tester", 0)
	player.actor.SetLogicState(entity.LogicStateFighting)
	if _, _, err := player.setSP(maxPlayerSP - 3); err != nil {
		t.Fatalf("stage partial SP: %v", err)
	}
	if !player.combatRageTick() {
		t.Fatal("partial rage should still advance toward the cap")
	}
	if got := player.skillSP(); got != maxPlayerSP {
		t.Fatalf("SP must saturate at %d, got %d", maxPlayerSP, got)
	}
	if player.combatRageTick() {
		t.Fatal("capped rage must not report progress")
	}
}
