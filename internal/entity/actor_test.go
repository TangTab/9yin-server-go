package entity

import (
	"testing"
	"time"
)

func TestActorDamageIsAuthoritativeAndClamped(t *testing.T) {
	a := NewActor(1, 1, "测试", 1000, 500)
	if !a.ApplyDamage(100) {
		t.Fatal("first damage must change actor")
	}
	state := a.Snapshot()
	if state.HP != 900 || state.HitHP != 1000 || state.LogicState != LogicStateFighting {
		t.Fatalf("after first hit = %#v", state)
	}
	if !a.ApplyDamage(5000) {
		t.Fatal("lethal damage must change actor")
	}
	state = a.Snapshot()
	if state.HP != 0 || state.HitHP != 0 || state.LogicState != LogicStateDied {
		t.Fatalf("after lethal hit = %#v", state)
	}
	if a.ApplyDamage(1) || a.ApplyDamage(0) {
		t.Fatal("dead actor and zero damage must not change state")
	}
}

func TestActorHealManaAndRevive(t *testing.T) {
	a := NewPlayer(1, 1, "测试")
	if !a.ApplyDamage(250) || !a.ApplyHeal(100) {
		t.Fatal("damage then heal must change the actor")
	}
	if got := a.Snapshot(); got.HP != 850 || got.HitHP != 1000 || got.LogicState != LogicStateFighting {
		t.Fatalf("healed state = %#v", got)
	}
	if !a.SpendMP(500) || a.SpendMP(1) {
		t.Fatal("MP spending must be bounded by the actor state")
	}
	a.ApplyDamage(5000)
	if a.ApplyHeal(100) {
		t.Fatal("healing must not implicitly revive a dead actor")
	}
	a.Revive()
	if got := a.Snapshot(); got.HP != 1000 || got.HitHP != 1000 || got.MP != 500 || got.LogicState != LogicStateNormal {
		t.Fatalf("revived state = %#v", got)
	}
}

func TestActorFightingRecoveryOnlySettlesBruisedBlood(t *testing.T) {
	a := NewPlayer(1, 1, "测试")
	a.ApplyDamage(200)
	if changed := a.RecoverBruisedHPWhileFighting(50); !changed {
		t.Fatal("fighting actor did not settle bruised blood")
	}
	if got := a.Snapshot(); got.HP != 800 || got.HitHP != 950 || got.LogicState != LogicStateFighting {
		t.Fatalf("first bruised-blood settlement=%+v", got)
	}
	if changed := a.RecoverBruisedHPWhileFighting(500); !changed {
		t.Fatal("remaining bruised blood did not settle")
	}
	if got := a.Snapshot(); got.HP != 800 || got.HitHP != 800 {
		t.Fatalf("bruise must clamp at real HP, got=%+v", got)
	}
	if changed := a.RecoverBruisedHPWhileFighting(1); changed {
		t.Fatal("fully settled bruised layer changed again")
	}
	a.SetLogicState(LogicStateNormal)
	if changed := a.RecoverBruisedHPWhileFighting(1); changed {
		t.Fatal("non-fighting actor recovered bruised blood")
	}
}

func TestActorRestoreWhileSitOnlyChangesLivingSitcrossActor(t *testing.T) {
	a := NewPlayer(1, 1, "测试")
	a.ApplyDamage(200)
	a.SpendMP(100)
	if changed := a.RestoreWhileSit(20, 30); changed {
		t.Fatal("normal actor recovered outside sitcross")
	}
	a.SetLogicState(LogicStateSitcross)
	if changed := a.RestoreWhileSit(20, 30); !changed {
		t.Fatal("sitcross actor did not recover")
	}
	if got := a.Snapshot(); got.HP != 820 || got.MP != 430 || got.HitHP != 1000 {
		t.Fatalf("recovery state=%+v", got)
	}
	a.SetLogicState(LogicStateDied)
	if changed := a.RestoreWhileSit(20, 30); changed {
		t.Fatal("dead actor recovered while sitcross unavailable")
	}
}

func TestActorDerivedMaximaKeepBaseProtocolValuesAndResourceRules(t *testing.T) {
	a := NewPlayer(1, 1, "测试")
	if changed := a.SetDerivedMaxima(2630, 1480); !changed {
		t.Fatal("derived maxima did not change")
	}
	if got := a.Snapshot(); got.BaseMaxHP != 1000 || got.MaxHP != 2630 || got.HP != 2630 || got.BaseMaxMP != 500 || got.MaxMP != 1480 || got.MP != 1480 {
		t.Fatalf("derived full state=%+v", got)
	}
	a.ApplyDamage(1000)
	a.SpendMP(1000)
	a.SetDerivedMaxima(1200, 600)
	if got := a.Snapshot(); got.HP != 1200 || got.MP != 480 || got.MaxHP != 1200 || got.MaxMP != 600 {
		t.Fatalf("derived clamp state=%+v", got)
	}
}

func TestActorQingGongCostMultiplierAndRecovery(t *testing.T) {
	a := NewPlayer(1, 1, "测试")
	start := time.Now()
	a.SetQingGongConsumeMultiplier(-0.2)
	cost, ok := a.UseQingGong(25, start)
	if !ok || cost != 20 {
		t.Fatalf("cost=%d ok=%t, want 20 true", cost, ok)
	}
	if got := a.Snapshot().QingGongPoint; got != 130 {
		t.Fatalf("points after use=%d, want 130", got)
	}
	if !a.RestoreQingGong(start.Add(2 * time.Second)) {
		t.Fatal("expected server-side qinggong recovery")
	}
	if got := a.Snapshot().QingGongPoint; got != 150 {
		t.Fatalf("points after recovery=%d, want 150", got)
	}
}

func TestActorQingGongSuccessfulUseRefillsBelowFifty(t *testing.T) {
	a := NewPlayer(1, 1, "测试")
	start := time.Now()

	// Four accepted 25-point actions leave 50 exactly: the threshold is
	// strictly below 50, so this must not refill yet.
	for i := 0; i < 4; i++ {
		if _, ok := a.UseQingGong(25, start); !ok {
			t.Fatalf("use %d was rejected", i+1)
		}
	}
	if got := a.Snapshot().QingGongPoint; got != 50 {
		t.Fatalf("points at threshold=%d, want 50", got)
	}

	// The next accepted use leaves 25, so the local-test refill must be
	// reflected by the same state snapshot sent in the following 0x10 update.
	if cost, ok := a.UseQingGong(25, start); !ok || cost != 25 {
		t.Fatalf("below-threshold use cost=%d ok=%t, want 25 true", cost, ok)
	}
	if got := a.Snapshot().QingGongPoint; got != 150 {
		t.Fatalf("points after below-threshold refill=%d, want 150", got)
	}
}
