package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/sim"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

type combatTickResult struct {
	playerStateChanged bool
	playerRevived      bool
}

type npcMoveUpdate struct {
	id          uint32
	ownerID     uint32
	destination worldcore.Transform
}

type npcPropertyUpdate struct {
	id         uint32
	ownerID    uint32
	properties []clientdata.IndexedProperty
}

const (
	combatTickMaxAggressors = 4
	defaultAggroRange       = float32(12)
	defaultChaseRange       = float32(30)
	defaultAttackRange      = float32(3.5)
	defaultNPCMoveSpeed     = float32(4.5)
	defaultAttackEvery      = 1500 * time.Millisecond
	defaultNPCRespawn       = 15 * time.Second
	playerAutoReviveDelay   = 8 * time.Second
)

// newNPCCombatState 从模板属性构造敌对 NPC 的战斗决策组件。
// 模板仍然决定哪个对象是敌对的；此状态只保存运行期决策，而这些决策
// 是正式服务器绝不能下放给客户端的。
func newNPCCombatState(npc npcSpawn, home worldcore.Transform) sim.Brain {
	aggro := npc.float32Property("SpringRange")
	if aggro <= 0 {
		aggro = defaultAggroRange
	}
	chase := npc.float32Property("ChaseRange")
	if chase < aggro {
		chase = defaultChaseRange
	}
	move := npc.float32Property("RunSpeed")
	if move <= 0 {
		move = npc.float32Property("MoveSpeed")
	}
	if move <= 0 {
		move = defaultNPCMoveSpeed
	}
	maxHP := npc.int32Property("MaxHP")
	if maxHP <= 0 {
		maxHP = 1000
	}
	damage := maxHP / 50
	if damage < 10 {
		damage = 10
	}
	if npc.resolved.ScriptClass == "BossNpc" {
		damage *= 2
	}
	return sim.Brain{
		Home: home, AggroRange: aggro, ChaseRange: chase,
		AttackRange: defaultAttackRange, MoveSpeed: move, Damage: damage,
		AttackEvery: defaultAttackEvery, RespawnAfter: defaultNPCRespawn,
	}
}

func (n npcSpawn) float32Property(name string) float32 {
	return n.resolved.Properties[name].Value.F32
}

// combatTick 在所属连接的循环上每秒执行一次。它只
// 发送当前客户端已会消费的包：NPC 0x20/0x1F 移动
// 以及普通的 0x10 属性包。不涉及任何已过时的 S->C 自定义消息。
func (s *sceneLifecycle) combatTick(player *playerActor, position role.Position, now time.Time) (combatTickResult, error) {
	if s == nil || player == nil {
		return combatTickResult{}, nil
	}
	result := combatTickResult{}
	if player.expireInnerPowerPassive(now) {
		result.playerStateChanged = true
	}
	if player.reviveWhenDue(now) {
		result.playerStateChanged = true
		result.playerRevived = true
	}
	playerState := player.stateSnapshot()
	if playerState.HP <= 0 {
		return result, nil
	}

	s.mu.Lock()
	if s.closed || !s.combatActive || s.combat.CombatCount() == 0 {
		s.mu.Unlock()
		return result, nil
	}
	moves := make([]npcMoveUpdate, 0, combatTickMaxAggressors)
	updates := make([]npcPropertyUpdate, 0, 2)
	aggressors := 0
	threatened := false
	// EachCombatNPC 以 EntityID 升序回调并交付组件指针；分支逻辑
	// 与迁移前的 map 版本逐行等价，就地修改即生效。
	var loopErr error
	s.combat.EachCombatNPC(func(id worldcore.EntityID, vitals *sim.Vitals, state *sim.Brain) {
		if loopErr != nil {
			return
		}
		meta, exists := s.entities[id]
		if !exists {
			return
		}
		actorState := vitals.ActorState(meta.id, meta.ownerID)
		if actorState.HP <= 0 {
			if !state.RespawnAt.IsZero() && !now.Before(state.RespawnAt) {
				vitals.Revive()
				state.Threat = 0
				state.NextAttack = time.Time{}
				state.RespawnAt = time.Time{}
				if err := s.setNPCTransformLocked(id, state.Home); err != nil {
					loopErr = err
					return
				}
				meta = s.entities[id]
				moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: state.Home})
				updates = append(updates, npcPropertyUpdate{id: meta.id, ownerID: meta.ownerID, properties: npcCombatProperties(vitals.ActorState(meta.id, meta.ownerID))})
			}
			return
		}

		distance := horizontalDistance(meta.transform, worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient})
		if state.Threat == 0 && distance <= state.AggroRange {
			state.Threat = 1
		}
		if state.Threat == 0 {
			return
		}
		if distance > state.ChaseRange {
			state.Threat = 0
			if horizontalDistance(meta.transform, state.Home) > 0.25 {
				if err := s.setNPCTransformLocked(id, state.Home); err != nil {
					loopErr = err
					return
				}
				moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: state.Home})
			}
			return
		}
		// 该 NPC 仍持有有效目标，即便它此刻正在等待
		// 攻击冷却结束，或已超出本 tick 的攻击者数量上限。
		threatened = true
		if now.Before(state.ControlledUntil) {
			return
		}
		if aggressors >= combatTickMaxAggressors {
			return
		}
		aggressors++
		if distance > state.AttackRange {
			destination := stepToward(meta.transform, worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}, minFloat32(state.MoveSpeed, distance-state.AttackRange))
			if err := s.setNPCTransformLocked(id, destination); err != nil {
				loopErr = err
				return
			}
			moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: destination})
			return
		}
		if now.Before(state.NextAttack) {
			return
		}
		state.NextAttack = now.Add(state.AttackEvery)
		outcome := player.resolveNPCAttack(meta.id, state.Damage, now)
		if outcome.innerPowerTriggered {
			result.playerStateChanged = true
			log.Printf("%s: inner-power incoming passive triggered id=%d config=%s", s.remote, meta.id, meta.configID)
		}
		if outcome.dodged {
			if outcome.innerPowerHeal > 0 {
				playerState := player.stateSnapshot()
				log.Printf("%s: NPC attack dodged id=%d config=%s inner-power dodge-heal=%d player_hp=%d/%d", s.remote, meta.id, meta.configID, outcome.innerPowerHeal, playerState.HP, playerState.MaxHP)
			} else {
				log.Printf("%s: NPC attack dodged id=%d config=%s", s.remote, meta.id, meta.configID)
			}
		} else if outcome.stateChanged {
			result.playerStateChanged = true
			playerState := player.stateSnapshot()
			log.Printf("%s: NPC attack id=%d config=%s damage=%d player_hp=%d/%d", s.remote, meta.id, meta.configID, state.Damage, playerState.HP, playerState.MaxHP)
		} else if outcome.parried {
			log.Printf("%s: NPC attack parried id=%d config=%s", s.remote, meta.id, meta.configID)
		}
		if outcome.dead {
			result.playerStateChanged = true
		}
	})
	if loopErr != nil {
		s.mu.Unlock()
		return result, loopErr
	}
	if player.setCombatPresence(threatened) {
		result.playerStateChanged = true
	}
	s.mu.Unlock()

	for _, move := range moves {
		if err := s.conn.WriteFrame(serverMoving(move.id, move.ownerID, motionFromDest(move.destination))); err != nil {
			return result, fmt.Errorf("combat move NPC %d: %w", move.id, err)
		}
		// ServerLocation 在当前客户端中是到达确认。不要
		// 在同一个战斗 tick 中让它与 ServerMoving 成对发送：那样会使
		// actor 瞬移到目的地而不是播放移动动作。之后
		// 的追击目的地会在该定时器执行之前更改权威
		// transform，从而令此次确认失效。
		s.scheduleCombatMoveArrival(move, time.Second)
	}
	for _, update := range updates {
		frame, err := sceneObjectProperties(update.id, update.ownerID, 0, update.properties)
		if err != nil {
			return result, err
		}
		if err := s.conn.WriteFrame(frame); err != nil {
			return result, fmt.Errorf("combat NPC state %d: %w", update.id, err)
		}
	}
	return result, nil
}

// scheduleCombatMoveArrival 只在客户端拥有一个移动间隔
// 来朝目的地播放动画之后，才确认最终位置。transform
// 比较即是取消令牌：持续追击每 tick 都会更新它，
// 因此过期的定时器绝不会确认一个陈旧的目的地。
func (s *sceneLifecycle) scheduleCombatMoveArrival(move npcMoveUpdate, delay time.Duration) {
	if delay <= 0 {
		delay = time.Second
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	epoch := s.epoch
	timer := time.AfterFunc(delay, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.epoch != epoch {
			return
		}
		meta, exists := s.entities[worldcore.EntityID(move.id)]
		if !exists || meta.ownerID != move.ownerID || !sameCombatTransform(meta.transform, move.destination) {
			return
		}
		if err := s.conn.WriteFrame(serverLocation(meta.id, meta.ownerID, meta.transform)); err != nil {
			log.Printf("%s: confirm combat NPC move id=%d: %v", s.remote, meta.id, err)
		}
	})
	s.timers = append(s.timers, timer)
	s.mu.Unlock()
}

func sameCombatTransform(a, b worldcore.Transform) bool {
	const epsilon = 0.01
	return math.Abs(float64(a.X-b.X)) < epsilon &&
		math.Abs(float64(a.Y-b.Y)) < epsilon &&
		math.Abs(float64(a.Z-b.Z)) < epsilon &&
		math.Abs(float64(a.Orient-b.Orient)) < epsilon
}

func (s *sceneLifecycle) setNPCTransformLocked(id worldcore.EntityID, destination worldcore.Transform) error {
	meta, exists := s.entities[id]
	if !exists {
		return fmt.Errorf("set transform for unknown NPC %d", id)
	}
	current, err := s.scene.Get(id)
	if err != nil {
		return err
	}
	current.Entity.Transform = destination
	for _, property := range []struct {
		name  string
		value float32
	}{{"PosiX", destination.X}, {"PosiY", destination.Y}, {"PosiZ", destination.Z}, {"Orient", destination.Orient}} {
		if err := current.Entity.Properties.Set(property.name, worldcore.Float32(property.value)); err != nil {
			return err
		}
	}
	if err := s.scene.Replace(current.Entity); err != nil {
		return err
	}
	meta.transform = destination
	s.entities[id] = meta
	return nil
}

func npcCombatProperties(state entity.ActorState) []clientdata.IndexedProperty {
	hpRatio := int32(int64(state.HP) * 100 / int64(state.MaxHP))
	return []clientdata.IndexedProperty{
		{Index: 21, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)},
		{Index: 28, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)},
		{Index: 30, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)},
		{Index: 32, Name: "HP", Value: clientdata.Int32Value(state.HP)},
		{Index: 112, Name: "HitHP", Value: clientdata.Int32Value(state.HitHP)},
		{Index: propDead, Name: "Dead", Value: clientdata.Int32Value(boolInt32(state.HP <= 0))},
	}
}

func horizontalDistance(a, b worldcore.Transform) float32 {
	dx, dz := a.X-b.X, a.Z-b.Z
	return float32(math.Hypot(float64(dx), float64(dz)))
}

func stepToward(from, to worldcore.Transform, step float32) worldcore.Transform {
	distance := horizontalDistance(from, to)
	if distance <= 0 || step <= 0 {
		return from
	}
	if step > distance {
		step = distance
	}
	ratio := step / distance
	result := from
	result.X += (to.X - from.X) * ratio
	result.Z += (to.Z - from.Z) * ratio
	result.Orient = float32(math.Atan2(float64(to.Z-from.Z), float64(to.X-from.X)))
	return result
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
