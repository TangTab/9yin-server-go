package sim

import (
	"math"
	"sort"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/entity"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// 本文件对比同一战斗 tick 负载在两套存储上的开销：
//   - ECS：sim.World（Vitals+Brain 组件，EachCombatNPC 迭代）
//   - 旧实现：map[EntityID]*entity.Actor + map[EntityID]状态值，
//     排序 id 后逐个 Snapshot/回写（combat_ai.go 迁移前的形态）
//
// 两侧负载逐行对齐：距离判定、仇恨置位/栓绳清零、攻击冷却推进、
// 组件/状态写回。协议 I/O 与 Scene 对账两侧完全相同，不参与对比。

const benchNPCCount = 500

// legacyState 是迁移前 npcCombatState 的热字段镜像。
type legacyState struct {
	home                   worldcore.Transform
	aggroRange, chaseRange float32
	attackRange, moveSpeed float32
	damage                 int32
	attackEvery            time.Duration
	threat                 int32
	nextAttack, respawnAt  time.Time
	controlledUntil        time.Time
}

func benchTransform(i int) worldcore.Transform {
	angle := float64(i) * 2 * math.Pi / benchNPCCount
	return worldcore.Transform{
		X:      float32(8 * math.Cos(angle)),
		Z:      float32(8 * math.Sin(angle)),
		Orient: float32(angle),
	}
}

func benchBrain(i int) Brain {
	return Brain{
		Home: benchTransform(i), AggroRange: 12, ChaseRange: 30,
		AttackRange: 3.5, MoveSpeed: 4.5, Damage: 20,
		AttackEvery: 1500 * time.Millisecond,
	}
}

func benchHorizontalDistance(a, b worldcore.Transform) float32 {
	dx, dz := a.X-b.X, a.Z-b.Z
	return float32(math.Hypot(float64(dx), float64(dz)))
}

var benchPlayerTransform = worldcore.Transform{X: 0, Y: 0, Z: 0}

// BenchmarkCombatTickECS 是迁移后的存储路径。
func BenchmarkCombatTickECS(b *testing.B) {
	world := New()
	transforms := make([]worldcore.Transform, benchNPCCount)
	for i := 0; i < benchNPCCount; i++ {
		id := worldcore.EntityID(i + 2)
		transforms[i] = benchTransform(i)
		world.SpawnCombatNPC(id, NewVitals(1000, 100), benchBrain(i))
	}
	now := time.Unix(10000, 0)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		world.EachCombatNPC(func(id worldcore.EntityID, v *Vitals, s *Brain) {
			meta := transforms[int(id)-2]
			distance := benchHorizontalDistance(meta, benchPlayerTransform)
			if s.Threat == 0 && distance <= s.AggroRange {
				s.Threat = 1
			}
			if s.Threat == 0 {
				return
			}
			if distance > s.ChaseRange {
				s.Threat = 0
				return
			}
			if now.Before(s.NextAttack) {
				return
			}
			s.NextAttack = now.Add(s.AttackEvery)
			_ = v
		})
	}
}

// BenchmarkCombatTickLegacy 复现迁移前 combatTick 的存储访问形态。
func BenchmarkCombatTickLegacy(b *testing.B) {
	actors := make(map[worldcore.EntityID]*entity.Actor, benchNPCCount)
	states := make(map[worldcore.EntityID]legacyState, benchNPCCount)
	transforms := make([]worldcore.Transform, benchNPCCount)
	for i := 0; i < benchNPCCount; i++ {
		id := worldcore.EntityID(i + 2)
		transforms[i] = benchTransform(i)
		actors[id] = entity.NewActor(uint32(id), 1, "npc", 1000, 100)
		brain := benchBrain(i)
		states[id] = legacyState{
			home: brain.Home, aggroRange: brain.AggroRange, chaseRange: brain.ChaseRange,
			attackRange: brain.AttackRange, moveSpeed: brain.MoveSpeed, damage: brain.Damage,
			attackEvery: brain.AttackEvery,
		}
	}
	now := time.Unix(10000, 0)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ids := make([]int, 0, len(states))
		for id := range states {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		for _, rawID := range ids {
			id := worldcore.EntityID(rawID)
			state := states[id]
			actor := actors[id]
			actorState := actor.Snapshot()
			if actorState.HP <= 0 {
				states[id] = state
				continue
			}
			meta := transforms[int(id)-2]
			distance := benchHorizontalDistance(meta, benchPlayerTransform)
			if state.threat == 0 && distance <= state.aggroRange {
				state.threat = 1
			}
			if state.threat == 0 {
				states[id] = state
				continue
			}
			if distance > state.chaseRange {
				state.threat = 0
				states[id] = state
				continue
			}
			if now.Before(state.nextAttack) {
				states[id] = state
				continue
			}
			state.nextAttack = now.Add(state.attackEvery)
			states[id] = state
		}
	}
}

// 伤害原语的微基准：同样的满血大血条下反复打击（永不死亡，
// 保持稳态），对比 Vitals 值语义与 Actor 互斥锁语义的开销。
func BenchmarkApplyDamageECS(b *testing.B) {
	v := NewVitals(1<<30, 100)
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		v.ApplyDamage(20)
	}
}

func BenchmarkApplyDamageLegacy(b *testing.B) {
	a := entity.NewActor(1, 1, "npc", 1<<30, 100)
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		a.ApplyDamage(20)
	}
}

// 注册风暴：一次性 spawn 500 个战斗 NPC，对比 map 双写与
// ECS 显式 id 写入（含 archetype 分配）的开销。
func BenchmarkPopulateECS(b *testing.B) {
	for n := 0; n < b.N; n++ {
		world := New()
		for i := 0; i < benchNPCCount; i++ {
			world.SpawnCombatNPC(worldcore.EntityID(i+2), NewVitals(1000, 100), benchBrain(i))
		}
	}
}

func BenchmarkPopulateLegacy(b *testing.B) {
	for n := 0; n < b.N; n++ {
		actors := make(map[worldcore.EntityID]*entity.Actor, benchNPCCount)
		states := make(map[worldcore.EntityID]legacyState, benchNPCCount)
		for i := 0; i < benchNPCCount; i++ {
			id := worldcore.EntityID(i + 2)
			actors[id] = entity.NewActor(uint32(id), 1, "npc", 1000, 100)
			brain := benchBrain(i)
			states[id] = legacyState{home: brain.Home, aggroRange: brain.AggroRange}
		}
	}
}
