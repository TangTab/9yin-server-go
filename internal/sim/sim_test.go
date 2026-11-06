package sim

import (
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/entity"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// 以下原语测试以 internal/entity/actor_test.go 为 oracle：
// NPC 数值逻辑必须与 entity.Actor 等价，直到玩家也迁入组件存储。

func TestNewVitalsClampsAndStartsFull(t *testing.T) {
	v := NewVitals(0, 0)
	if v.MaxHP != 1 || v.MaxMP != 1 {
		t.Fatalf("non-positive maxima must clamp to 1, got %d/%d", v.MaxHP, v.MaxMP)
	}
	if v.HP != v.MaxHP || v.HitHP != v.MaxHP || v.MP != v.MaxMP {
		t.Fatalf("new vitals must start full, got %+v", v)
	}
	if v.Logic != entity.LogicStateNormal {
		t.Fatalf("new vitals must be normal, got %d", v.Logic)
	}
	if v.BaseMaxHP != v.MaxHP || v.BaseMaxMP != v.MaxMP {
		t.Fatalf("base maxima must mirror maxima, got %+v", v)
	}
}

func TestVitalsApplyDamageTransitions(t *testing.T) {
	v := NewVitals(100, 50)
	if !v.ApplyDamage(30) {
		t.Fatal("first damage must change state")
	}
	if v.HP != 70 || v.HitHP != 100 || v.Logic != entity.LogicStateFighting {
		t.Fatalf("bruise layer bookkeeping mismatch: %+v", v)
	}
	if v.ApplyDamage(0) || v.ApplyDamage(-5) {
		t.Fatal("non-positive damage must be rejected")
	}
	if !v.ApplyDamage(80) {
		t.Fatal("killing blow must change state")
	}
	if v.HP != 0 || v.HitHP != 0 || v.Logic != entity.LogicStateDied {
		t.Fatalf("death migration mismatch: %+v", v)
	}
	if v.ApplyDamage(10) {
		t.Fatal("dead vitals must not take further damage")
	}
}

// 以下用例自 internal/entity/actor_test.go 原样移植（玩家路径），
// 与上面的 NPC 子集共同构成原语行为的完整 oracle。

func newPlayerVitals() Vitals { return NewVitals(1000, 500) }

func TestVitalsHealManaAndRevive(t *testing.T) {
	v := newPlayerVitals()
	if !v.ApplyDamage(250) || !v.ApplyHeal(100) {
		t.Fatal("damage then heal must change the vitals")
	}
	if v.HP != 850 || v.HitHP != 1000 || v.Logic != entity.LogicStateFighting {
		t.Fatalf("healed state = %+v", v)
	}
	if !v.SpendMP(500) || v.SpendMP(1) {
		t.Fatal("MP spending must be bounded by the vitals state")
	}
	v.ApplyDamage(5000)
	if v.ApplyHeal(100) {
		t.Fatal("healing must not implicitly revive a dead actor")
	}
	v.Revive()
	if v.HP != 1000 || v.HitHP != 1000 || v.MP != 500 || v.Logic != entity.LogicStateNormal {
		t.Fatalf("revived state = %+v", v)
	}
}

func TestVitalsFightingRecoveryOnlySettlesBruisedBlood(t *testing.T) {
	v := newPlayerVitals()
	v.ApplyDamage(200)
	if changed := v.RecoverBruisedHPWhileFighting(50); !changed {
		t.Fatal("fighting actor did not settle bruised blood")
	}
	if v.HP != 800 || v.HitHP != 950 || v.Logic != entity.LogicStateFighting {
		t.Fatalf("first bruised-blood settlement=%+v", v)
	}
	if changed := v.RecoverBruisedHPWhileFighting(500); !changed {
		t.Fatal("remaining bruised blood did not settle")
	}
	if v.HP != 800 || v.HitHP != 800 {
		t.Fatalf("bruise must clamp at real HP, got=%+v", v)
	}
	if changed := v.RecoverBruisedHPWhileFighting(1); changed {
		t.Fatal("fully settled bruised layer changed again")
	}
	v.SetLogicState(entity.LogicStateNormal)
	if changed := v.RecoverBruisedHPWhileFighting(1); changed {
		t.Fatal("non-fighting actor recovered bruised blood")
	}
}

func TestVitalsRestoreWhileSitOnlyChangesLivingSitcrossActor(t *testing.T) {
	v := newPlayerVitals()
	v.ApplyDamage(200)
	v.SpendMP(100)
	if changed := v.RestoreWhileSit(20, 30); changed {
		t.Fatal("normal actor recovered outside sitcross")
	}
	v.SetLogicState(entity.LogicStateSitcross)
	if changed := v.RestoreWhileSit(20, 30); !changed {
		t.Fatal("sitcross actor did not recover")
	}
	if v.HP != 820 || v.MP != 430 || v.HitHP != 1000 {
		t.Fatalf("recovery state=%+v", v)
	}
	v.SetLogicState(entity.LogicStateDied)
	if changed := v.RestoreWhileSit(20, 30); changed {
		t.Fatal("dead actor recovered while sitcross unavailable")
	}
}

func TestVitalsDerivedMaximaKeepBaseProtocolValuesAndResourceRules(t *testing.T) {
	v := newPlayerVitals()
	if changed := v.SetDerivedMaxima(2630, 1480); !changed {
		t.Fatal("derived maxima did not change")
	}
	if v.BaseMaxHP != 1000 || v.MaxHP != 2630 || v.HP != 2630 || v.BaseMaxMP != 500 || v.MaxMP != 1480 || v.MP != 1480 {
		t.Fatalf("derived full state=%+v", v)
	}
	v.ApplyDamage(1000)
	v.SpendMP(1000)
	v.SetDerivedMaxima(1200, 600)
	if v.HP != 1200 || v.MP != 480 || v.MaxHP != 1200 || v.MaxMP != 600 {
		t.Fatalf("derived clamp state=%+v", v)
	}
}

func TestQingGongCostMultiplierAndRecovery(t *testing.T) {
	q := NewQingGong(time.Now())
	start := time.Now()
	q.SetConsumeMultiplier(-0.2)
	cost, ok := q.Use(25, start)
	if !ok || cost != 20 {
		t.Fatalf("cost=%d ok=%t, want 20 true", cost, ok)
	}
	if q.Point != 130 {
		t.Fatalf("points after use=%d, want 130", q.Point)
	}
	if !q.Restore(start.Add(2 * time.Second)) {
		t.Fatal("expected server-side qinggong recovery")
	}
	if q.Point != 150 {
		t.Fatalf("points after recovery=%d, want 150", q.Point)
	}
}

func TestQingGongSuccessfulUseRefillsBelowFifty(t *testing.T) {
	q := NewQingGong(time.Now())
	start := time.Now()

	// 四次被接受的 25 点动作恰好剩 50 点：阈值是严格小于 50，
	// 所以此时还不应回满。
	for i := 0; i < 4; i++ {
		if _, ok := q.Use(25, start); !ok {
			t.Fatalf("use %d was rejected", i+1)
		}
	}
	if q.Point != 50 {
		t.Fatalf("points at threshold=%d, want 50", q.Point)
	}

	// 下一次被接受的使用剩 25 点，因此本地测试的回满必须
	// 体现在随后的 0x10 更新所发送的同一份状态快照中。
	if cost, ok := q.Use(25, start); !ok || cost != 25 {
		t.Fatalf("below-threshold use cost=%d ok=%t, want 25 true", cost, ok)
	}
	if q.Point != 150 {
		t.Fatalf("points after below-threshold refill=%d, want 150", q.Point)
	}
}

func TestQingGongResetFillsToMaxPlusAdd(t *testing.T) {
	q := NewQingGong(time.Now())
	q.Use(100, time.Now())
	q.MaxAdd = 30
	now := time.Now()
	q.Reset(now)
	if q.Point != 180 || !q.UpdatedAt.Equal(now) {
		t.Fatalf("reset state=%+v", q)
	}
}

func TestMotionDefaultsAndClamp(t *testing.T) {
	m := NewMotion()
	if m.SpeedRatio != 100 {
		t.Fatalf("default speed ratio=%d, want 100", m.SpeedRatio)
	}
	m.SetSpeedRatio(-5)
	if m.SpeedRatio != 0 {
		t.Fatalf("negative ratio must clamp to 0, got %d", m.SpeedRatio)
	}
}

func TestVitalsReviveResetsToFull(t *testing.T) {
	v := NewVitals(100, 50)
	v.ApplyDamage(120)
	v.Revive()
	if v.HP != 100 || v.HitHP != 100 || v.MP != 50 || v.Logic != entity.LogicStateNormal {
		t.Fatalf("revive must restore full normal state, got %+v", v)
	}
}

func TestVitalsActorStateProjection(t *testing.T) {
	v := NewVitals(100, 50)
	v.ApplyDamage(40)
	state := v.ActorState(7, 1)
	if state.ObjectID != 7 || state.OwnerID != 1 {
		t.Fatalf("identity mismatch: %+v", state)
	}
	if state.HP != 60 || state.HitHP != 100 || state.MaxHP != 100 || state.BaseMaxHP != 100 {
		t.Fatalf("hp projection mismatch: %+v", state)
	}
	if state.MP != 50 || state.MaxMP != 50 || state.BaseMaxMP != 50 {
		t.Fatalf("mp projection mismatch: %+v", state)
	}
	if state.LogicState != entity.LogicStateFighting {
		t.Fatalf("logic projection mismatch: %+v", state)
	}
}

func TestWorldSpawnReadAndLayoutSeparation(t *testing.T) {
	world := New()
	world.SpawnNPC(2, NewVitals(80, 40))
	world.SpawnCombatNPC(3, NewVitals(120, 60), Brain{Home: worldcore.Transform{X: 1}, AggroRange: 12})

	if _, ok := world.Brain(2); ok {
		t.Fatal("civilian must not carry a Brain")
	}
	v, ok := world.Vitals(3)
	if !ok || v.MaxHP != 120 {
		t.Fatalf("combat NPC vitals mismatch: %+v ok=%v", v, ok)
	}
	if b, ok := world.Brain(3); !ok || b.Home.X != 1 {
		t.Fatalf("combat NPC brain mismatch: %+v ok=%v", b, ok)
	}
}

func TestWorldEachCombatNPCSortedAfterChurn(t *testing.T) {
	world := New()
	// 以乱序模拟视口反复注册/注销：插入顺序 7,5,3,9。
	for _, id := range []worldcore.EntityID{7, 5, 3, 9} {
		world.SpawnCombatNPC(id, NewVitals(10, 10), Brain{})
	}
	world.SpawnNPC(4, NewVitals(10, 10)) // 平民不得出现在迭代中
	world.Remove(7)
	world.SpawnCombatNPC(7, NewVitals(10, 10), Brain{})

	var seen []worldcore.EntityID
	world.EachCombatNPC(func(id worldcore.EntityID, v *Vitals, b *Brain) {
		seen = append(seen, id)
		v.HP-- // 就地修改：无需写回
	})
	if len(seen) != 4 {
		t.Fatalf("expected 4 combat NPCs, got %d", len(seen))
	}
	for i, want := range []worldcore.EntityID{3, 5, 7, 9} {
		if seen[i] != want {
			t.Fatalf("iteration must be id-ascending: got %v", seen)
		}
		if v, _ := world.Vitals(want); v.HP != 9 {
			t.Fatalf("write-back not visible for %d: %+v", want, v)
		}
	}
}

func TestWorldResetAndCombatCount(t *testing.T) {
	world := New()
	if world.CombatCount() != 0 {
		t.Fatal("fresh world must have no combat NPCs")
	}
	world.SpawnCombatNPC(2, NewVitals(10, 10), Brain{})
	world.SpawnCombatNPC(3, NewVitals(10, 10), Brain{})
	world.SpawnNPC(4, NewVitals(10, 10))
	if world.CombatCount() != 2 {
		t.Fatalf("combat count mismatch: %d", world.CombatCount())
	}
	world.Reset()
	if world.CombatCount() != 0 {
		t.Fatal("reset must drop all entities")
	}
	if _, ok := world.Vitals(2); ok {
		t.Fatal("reset must drop vitals too")
	}
	// 重置后同 id 复用必须正常。
	world.SpawnCombatNPC(2, NewVitals(30, 10), Brain{AttackEvery: time.Second})
	if v, ok := world.Vitals(2); !ok || v.MaxHP != 30 {
		t.Fatalf("respawn after reset mismatch: %+v ok=%v", v, ok)
	}
}

func TestWorldRemoveMissingIsNoop(t *testing.T) {
	world := New()
	world.Remove(42)
	world.SpawnNPC(42, NewVitals(10, 10))
	world.Remove(42)
	world.Remove(42)
	if _, ok := world.Vitals(42); ok {
		t.Fatal("removed entity must be gone")
	}
}
