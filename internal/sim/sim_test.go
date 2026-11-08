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
	if settlement := v.RestoreWhileSit(20, 30); settlement.Changed() {
		t.Fatal("normal actor recovered outside sitcross")
	}
	v.SetLogicState(entity.LogicStateSitcross)
	settlement := v.RestoreWhileSit(20, 30)
	if !settlement.Changed() {
		t.Fatal("sitcross actor did not recover")
	}
	if settlement.HPApplied != 20 || settlement.HPWasted != 0 || settlement.HitHPChanged || settlement.MPApplied != 30 || settlement.MPWasted != 0 {
		t.Fatalf("settlement=%+v", settlement)
	}
	if v.HP != 820 || v.MP != 430 || v.HitHP != 1000 {
		t.Fatalf("recovery state=%+v", v)
	}
	// 回复追平受击血条的那一跳：HitHP 低于回复后的 HP 时被抬升，
	// 官服该跳双 Ratio+双值四帧（00:11:54.833）。
	v.HitHP = 700
	settlement = v.RestoreWhileSit(50, 0)
	if !settlement.HitHPChanged || v.HitHP != 870 {
		t.Fatalf("hitHP catch-up settlement=%+v state=%+v", settlement, v)
	}
	// 触顶跳按官服末跳口径拆 (实际, 溢出)：只剩 130 缺口，回复 200
	// 落 130/70（官服补满末跳 162+1478=名义 1640 同式）。
	settlement = v.RestoreWhileSit(200, 0)
	if settlement.HPApplied != 130 || settlement.HPWasted != 70 || !settlement.HitHPChanged {
		t.Fatalf("capped settlement=%+v", settlement)
	}
	if v.HP != 1000 || v.HitHP != 1000 {
		t.Fatalf("capped recovery state=%+v", v)
	}
	v.SetLogicState(entity.LogicStateDied)
	if settlement := v.RestoreWhileSit(20, 30); settlement.Changed() {
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
	// 官方分档回复（Java Actor.recoverQingGongTiered 同款）：余量 130
	// 落在 101..上限 档，一步 +10；满条后不再涨。
	if !q.RecoverTiered() {
		t.Fatal("expected tiered qinggong recovery")
	}
	if q.Point != 140 {
		t.Fatalf("points after recovery=%d, want 140", q.Point)
	}
	q.Point = 150
	if q.RecoverTiered() {
		t.Fatal("recovery must stop at cap")
	}
	if q.Point != 150 {
		t.Fatalf("points above cap=%d, want 150", q.Point)
	}
}

func TestQingGongSuccessfulUseNeverRefills(t *testing.T) {
	// 2026-11-07 对齐 Java：官方轻功回复是分档慢回复，「余量 <50
	// 自动回满」本地规则是真机无限跳的根因，已删除（Java
	// Actor.useQingGong 2026-11-03 同款删除）。
	q := NewQingGong(time.Now())
	start := time.Now()
	for i := 0; i < 5; i++ {
		if _, ok := q.Use(25, start); !ok {
			t.Fatalf("use %d was rejected", i+1)
		}
	}
	if q.Point != 25 {
		t.Fatalf("points after five uses=%d, want 25 (no auto refill)", q.Point)
	}
	// 第六次恰好付得起（剩 0）；第七次才付不起。
	if _, ok := q.Use(25, start); !ok || q.Point != 0 {
		t.Fatalf("sixth use ok=%t point=%d, want true 0", ok, q.Point)
	}
	if _, ok := q.Use(25, start); ok {
		t.Fatal("seventh use must be rejected")
	}
	// 低档回复一步 +3。
	if !q.RecoverTiered() || q.Point != 3 {
		t.Fatalf("low-tier recovery state=%d, want 3", q.Point)
	}
}

func TestQingGongTieredRecoverySteps(t *testing.T) {
	q := NewQingGong(time.Now())
	q.Point = 100
	if !q.RecoverTiered() || q.Point != 105 {
		t.Fatalf("mid tier (51..100) gain state=%d, want 105", q.Point)
	}
	q.Point = 50
	if !q.RecoverTiered() || q.Point != 53 {
		t.Fatalf("low tier (0..50) gain state=%d, want 53", q.Point)
	}
	q.Point = 101
	q.MaxAdd = 49 // cap 199
	if !q.RecoverTiered() || q.Point != 111 {
		t.Fatalf("high tier (>=101) gain state=%d, want 111", q.Point)
	}
}

func TestQingGongDrainForSustainedLease(t *testing.T) {
	// 持续租约扣点（Java Actor.drainQingGong）：不触发回满，
	// 付不起清零并返回 false（调用方强摘租约）。
	q := NewQingGong(time.Now())
	q.Point = 25
	if !q.Drain(7) || q.Point != 18 {
		t.Fatalf("drain state=%d, want 18", q.Point)
	}
	q.Point = 3
	if q.Drain(7) || q.Point != 0 {
		t.Fatalf("exhausted drain state=%d ok, want 0 false", q.Point)
	}
	if !q.Drain(0) || q.Point != 0 {
		t.Fatalf("zero drain on empty pool must succeed without change, state=%d", q.Point)
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

// TestVitalsResetAfterLeashRefillsLivingFighter 栓绳重置只服务活怪：
// 受击残血（含瘀血层）的 Fighting NPC 回家即回满并退回 Normal；
// Died 态不经此路径复活（尸体仍走 Revive/重生批次）。
func TestVitalsResetAfterLeashRefillsLivingFighter(t *testing.T) {
	v := NewVitals(100, 50)
	v.ApplyDamage(60)
	v.SpendMP(30)
	v.ResetAfterLeash()
	if v.HP != 100 || v.HitHP != 100 || v.MP != 50 || v.Logic != entity.LogicStateNormal {
		t.Fatalf("leash reset must restore full normal state, got %+v", v)
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
