package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// npcCombatState is deliberately small and server-owned. The creator/template
// still determines which object is hostile; this state only holds the runtime
// decisions that a live server must not delegate to the client.
type npcCombatState struct {
	home         worldcore.Transform
	aggroRange   float32
	chaseRange   float32
	attackRange  float32
	moveSpeed    float32
	damage       int32
	attackEvery  time.Duration
	respawnAfter time.Duration

	threat          int32
	nextAttack      time.Time
	respawnAt       time.Time
	controlledUntil time.Time
}

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

func newNPCCombatState(npc npcSpawn, home worldcore.Transform) npcCombatState {
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
	return npcCombatState{
		home: home, aggroRange: aggro, chaseRange: chase,
		attackRange: defaultAttackRange, moveSpeed: move, damage: damage,
		attackEvery: defaultAttackEvery, respawnAfter: defaultNPCRespawn,
	}
}

func (n npcSpawn) float32Property(name string) float32 {
	return n.resolved.Properties[name].Value.F32
}

// combatTick runs on the owning connection loop once per second. It only
// writes packets the current client already consumes: NPC 0x20/0x1F movement
// and ordinary 0x10 properties. No obsolete S->C custom message is involved.
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
	playerState := player.actor.Snapshot()
	if playerState.HP <= 0 {
		return result, nil
	}

	s.mu.Lock()
	if s.closed || !s.combatActive || len(s.combatStates) == 0 {
		s.mu.Unlock()
		return result, nil
	}
	ids := make([]int, 0, len(s.combatStates))
	for id := range s.combatStates {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	moves := make([]npcMoveUpdate, 0, combatTickMaxAggressors)
	updates := make([]npcPropertyUpdate, 0, 2)
	aggressors := 0
	threatened := false
	for _, rawID := range ids {
		id := worldcore.EntityID(rawID)
		state, exists := s.combatStates[id]
		if !exists {
			continue
		}
		meta, exists := s.entities[id]
		actor := s.combatActors[id]
		if !exists || actor == nil {
			continue
		}
		actorState := actor.Snapshot()
		if actorState.HP <= 0 {
			if !state.respawnAt.IsZero() && !now.Before(state.respawnAt) {
				actor.Revive()
				state.threat = 0
				state.nextAttack = time.Time{}
				state.respawnAt = time.Time{}
				if err := s.setNPCTransformLocked(id, state.home); err != nil {
					s.mu.Unlock()
					return result, err
				}
				meta = s.entities[id]
				moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: state.home})
				updates = append(updates, npcPropertyUpdate{id: meta.id, ownerID: meta.ownerID, properties: npcCombatProperties(actor.Snapshot())})
			}
			s.combatStates[id] = state
			continue
		}

		distance := horizontalDistance(meta.transform, worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient})
		if state.threat == 0 && distance <= state.aggroRange {
			state.threat = 1
		}
		if state.threat == 0 {
			continue
		}
		if distance > state.chaseRange {
			state.threat = 0
			if horizontalDistance(meta.transform, state.home) > 0.25 {
				if err := s.setNPCTransformLocked(id, state.home); err != nil {
					s.mu.Unlock()
					return result, err
				}
				moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: state.home})
			}
			s.combatStates[id] = state
			continue
		}
		// This NPC retains a valid target, even if it is currently waiting for
		// its attack cooldown or exceeds the per-tick attacker cap.
		threatened = true
		if now.Before(state.controlledUntil) {
			s.combatStates[id] = state
			continue
		}
		if aggressors >= combatTickMaxAggressors {
			s.combatStates[id] = state
			continue
		}
		aggressors++
		if distance > state.attackRange {
			destination := stepToward(meta.transform, worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}, minFloat32(state.moveSpeed, distance-state.attackRange))
			if err := s.setNPCTransformLocked(id, destination); err != nil {
				s.mu.Unlock()
				return result, err
			}
			moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: destination})
			s.combatStates[id] = state
			continue
		}
		if now.Before(state.nextAttack) {
			s.combatStates[id] = state
			continue
		}
		state.nextAttack = now.Add(state.attackEvery)
		outcome := player.resolveNPCAttack(meta.id, state.damage, now)
		if outcome.innerPowerTriggered {
			result.playerStateChanged = true
			log.Printf("%s: inner-power incoming passive triggered id=%d config=%s", s.remote, meta.id, meta.configID)
		}
		if outcome.dodged {
			if outcome.innerPowerHeal > 0 {
				playerState := player.actor.Snapshot()
				log.Printf("%s: NPC attack dodged id=%d config=%s inner-power dodge-heal=%d player_hp=%d/%d", s.remote, meta.id, meta.configID, outcome.innerPowerHeal, playerState.HP, playerState.MaxHP)
			} else {
				log.Printf("%s: NPC attack dodged id=%d config=%s", s.remote, meta.id, meta.configID)
			}
		} else if outcome.stateChanged {
			result.playerStateChanged = true
			playerState := player.actor.Snapshot()
			log.Printf("%s: NPC attack id=%d config=%s damage=%d player_hp=%d/%d", s.remote, meta.id, meta.configID, state.damage, playerState.HP, playerState.MaxHP)
		} else if outcome.parried {
			log.Printf("%s: NPC attack parried id=%d config=%s", s.remote, meta.id, meta.configID)
		}
		if outcome.dead {
			result.playerStateChanged = true
		}
		s.combatStates[id] = state
	}
	if player.setCombatPresence(threatened) {
		result.playerStateChanged = true
	}
	s.mu.Unlock()

	for _, move := range moves {
		if err := s.conn.WriteFrame(serverMoving(move.id, move.ownerID, motionFromDest(move.destination))); err != nil {
			return result, fmt.Errorf("combat move NPC %d: %w", move.id, err)
		}
		// ServerLocation is an arrival confirmation in the current client.  Do
		// not pair it with ServerMoving in the same combat tick: doing so makes
		// the actor snap to the destination instead of playing the move.  A
		// later chase destination invalidates this confirmation by changing the
		// authoritative transform before this timer executes.
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

// scheduleCombatMoveArrival confirms the final location only after the
// client has had one movement interval to animate toward it. The transform
// comparison is the cancellation token: continuous chasing updates it every
// tick, so an obsolete timer never confirms a stale destination.
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
