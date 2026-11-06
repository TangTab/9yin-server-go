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

func TestRageRecoveryTickAddsFivePerSecondRegardlessOfCombatState(t *testing.T) {
	player := newPlayerActor("tester", 0)
	// 出生时默认 SP 为满池,先设为一个已耗尽的已知值。
	if _, _, err := player.setSP(40); err != nil {
		t.Fatalf("stage drained SP: %v", err)
	}
	// 官方模型(rage-recovery-frames.md §5):回复不限战斗态,脱战照样 +5。
	if !player.rageRecoveryTick() {
		t.Fatal("normal-state actor must recover rage")
	}
	player.setLogicState(entity.LogicStateFighting)
	if !player.rageRecoveryTick() {
		t.Fatal("fighting actor should recover rage")
	}
	if got := player.skillSP(); got != 40+2*combatRageRecoveryPerSecond {
		t.Fatalf("SP after two ticks = %d, want %d", got, 40+2*combatRageRecoveryPerSecond)
	}
	player.setLogicState(entity.LogicStateSitcross)
	if !player.rageRecoveryTick() {
		t.Fatal("sitting actor still recovers rage (state-independent)")
	}
	player.setLogicState(entity.LogicStateDied)
	if player.rageRecoveryTick() {
		t.Fatal("dead actor must not recover rage")
	}
}

func TestRageRecoveryTickSaturatesAtMaxPlayerSP(t *testing.T) {
	player := newPlayerActor("tester", 0)
	if _, _, err := player.setSP(maxPlayerSP - 3); err != nil {
		t.Fatalf("stage partial SP: %v", err)
	}
	if !player.rageRecoveryTick() {
		t.Fatal("partial rage should still advance toward the cap")
	}
	if got := player.skillSP(); got != maxPlayerSP {
		t.Fatalf("SP must saturate at %d, got %d", maxPlayerSP, got)
	}
	if player.rageRecoveryTick() {
		t.Fatal("capped rage must not report progress")
	}
}
