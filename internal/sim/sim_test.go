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
