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

// combatTick 是场景运行时的每秒权威 NPC 模拟。它只发送当前客户端
// 已会消费的包：NPC 0x20/0x1F 移动以及普通的 0x10 属性包。
// 仇恨目标取"最近存活成员"并经 Brain.TargetMember 粘性保持，
// 直到目标死亡或被拉出栓绳距离。
//
// 锁内结算（run.mu → p.mu，锁序与全局规则一致），帧写入在锁外
// 经各成员视图的 conn 完成（transport 写帧线程安全）。
type combatTickOutcome struct {
	members []*sceneMember
	moves   []npcMoveUpdate
	updates []npcPropertyUpdate
	changed map[uint32]bool
	// rageFrames 是受击得怒(格挡被揍 +2/直接被揍 +1)产生的 18 字节
	// SP 单帧,按受击成员收集,锁外写给属主连接。
	rageFrames map[uint32][]byte
}

func (r *sceneRun) combatTick(now time.Time) {
	outcome := r.simulateLocked(now)
	if outcome == nil {
		return
	}
	// 锁外帧写入：受击得怒 SP 单帧、受击成员的 vitalUpdate、
	// 其远程属性帧扇出、以及 NPC 移动/属性帧扇出。
	for _, member := range outcome.members {
		if frame, has := outcome.rageFrames[member.objectID]; has && member.view != nil {
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: publish rage gain: %v", member.view.remote, err)
			}
		}
		if !outcome.changed[member.objectID] {
			continue
		}
		if frame, err := member.player.vitalUpdate(); err == nil {
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: publish combat vitals: %v", member.view.remote, err)
			}
		} else {
			log.Printf("%s: encode combat vitals: %v", member.view.remote, err)
		}
		// 远程可见者收到该玩家的战斗属性子集（掉血/死亡/状态）。
		// 连接循环驱动的状态变化（打坐/技能/复活）的远程广播
		// 属于下一轮范围。
		state := member.player.stateSnapshot()
		state.ObjectID, state.OwnerID = member.objectID, 1
		if frame, err := sceneObjectProperties(member.objectID, 1, 0, npcCombatProperties(state)); err == nil {
			for _, observer := range outcome.members {
				if observer == member || observer.view == nil || !observer.view.npcVisible(member.objectID) {
					continue
				}
				if err := observer.view.conn.WriteFrame(frame); err != nil {
					log.Printf("%s: relay combat vitals for %d: %v", observer.view.remote, member.objectID, err)
				}
			}
		}
	}
	r.broadcastNPCFrames(outcome.members, outcome.moves, outcome.updates)
}

// simulateLocked 是 combatTick 的锁内结算段，独立出来供测试路径
// 复用（测试只消费 NPC 帧，不写玩家 vital 帧）。
func (r *sceneRun) simulateLocked(now time.Time) *combatTickOutcome {
	r.mu.Lock()
	if r.closed || len(r.members) == 0 || r.combat.CombatCount() == 0 {
		r.mu.Unlock()
		return nil
	}
	members := r.sortedMembersLocked()
	memberByID := make(map[uint32]*sceneMember, len(members))
	targets := make(map[uint32]worldcore.Transform, len(members))
	for _, member := range members {
		memberByID[member.objectID] = member
		if !member.active || member.player == nil {
			continue
		}
		// 死亡成员不是仇恨目标；其复活由属主连接循环负责。
		if member.player.stateSnapshot().HP <= 0 {
			continue
		}
		targets[member.objectID] = worldcore.Transform{
			X: member.position.X, Y: member.position.Y, Z: member.position.Z, Orient: member.position.Orient,
		}
	}
	moves := make([]npcMoveUpdate, 0, combatTickMaxAggressors)
	updates := make([]npcPropertyUpdate, 0, 2)
	aggressors := 0
	threatened := make(map[uint32]bool, len(targets))
	var rage map[uint32][]byte
	// eachCombatNPCFullLocked 以 EntityID 升序一次遍历交付全部组件指针；
	// 分支逻辑与单连接时代逐行等价，就地修改即生效。
	r.eachCombatNPCFullLocked(func(id worldcore.EntityID, vitals *sim.Vitals, state *sim.Brain, transform *worldcore.Transform, meta *sceneEntity) {
		actorState := vitals.ActorState(meta.id, meta.ownerID)
		if actorState.HP <= 0 {
			if !state.RespawnAt.IsZero() && !now.Before(state.RespawnAt) {
				vitals.Revive()
				state.Threat = 0
				state.TargetMember = 0
				state.NextAttack = time.Time{}
				state.RespawnAt = time.Time{}
				if err := r.setNPCTransformLocked(id, state.Home); err != nil {
					log.Printf("scene %s/%s: respawn transform id=%d: %v", r.key.config, r.key.resource, meta.id, err)
					return
				}
				moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: state.Home})
				updates = append(updates, npcPropertyUpdate{id: meta.id, ownerID: meta.ownerID, properties: npcCombatProperties(vitals.ActorState(meta.id, meta.ownerID))})
			}
			return
		}

		if state.Threat == 0 || state.TargetMember == 0 {
			state.TargetMember = nearestTargetLocked(targets, *transform, state.AggroRange)
			if state.TargetMember == 0 {
				return
			}
			state.Threat = 1
		}
		position, engaged := targets[state.TargetMember]
		if !engaged {
			// 目标死亡或离场：清除仇恨并按栓绳规则回位。
			state.Threat = 0
			state.TargetMember = 0
			if horizontalDistance(*transform, state.Home) > 0.25 {
				if err := r.setNPCTransformLocked(id, state.Home); err != nil {
					log.Printf("scene %s/%s: leash transform id=%d: %v", r.key.config, r.key.resource, meta.id, err)
					return
				}
				moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: state.Home})
			}
			return
		}
		distance := horizontalDistance(*transform, position)
		if distance > state.ChaseRange {
			state.Threat = 0
			state.TargetMember = 0
			if horizontalDistance(*transform, state.Home) > 0.25 {
				if err := r.setNPCTransformLocked(id, state.Home); err != nil {
					log.Printf("scene %s/%s: leash transform id=%d: %v", r.key.config, r.key.resource, meta.id, err)
					return
				}
				moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: state.Home})
			}
			return
		}
		// 该 NPC 仍持有有效目标，即便它此刻正在等待
		// 攻击冷却结束，或已超出本 tick 的攻击者数量上限。
		threatened[state.TargetMember] = true
		if now.Before(state.ControlledUntil) {
			return
		}
		if aggressors >= combatTickMaxAggressors {
			return
		}
		aggressors++
		target := memberByID[state.TargetMember]
		if distance > state.AttackRange {
			destination := stepToward(*transform, position, minFloat32(state.MoveSpeed, distance-state.AttackRange))
			if err := r.setNPCTransformLocked(id, destination); err != nil {
				log.Printf("scene %s/%s: chase transform id=%d: %v", r.key.config, r.key.resource, meta.id, err)
				return
			}
			moves = append(moves, npcMoveUpdate{id: meta.id, ownerID: meta.ownerID, destination: destination})
			return
		}
		if now.Before(state.NextAttack) {
			return
		}
		state.NextAttack = now.Add(state.AttackEvery)
		outcome := target.player.resolveNPCAttack(meta.id, state.Damage, now)
		if outcome.rageDelta != 0 {
			if _, spFrame, spErr := target.player.addSP(outcome.rageDelta); spErr == nil && spFrame != nil {
				if rage == nil {
					rage = map[uint32][]byte{}
				}
				rage[target.objectID] = spFrame
			}
		}
		remote := target.view.remote
		if outcome.innerPowerTriggered {
			log.Printf("%s: inner-power incoming passive triggered id=%d config=%s", remote, meta.id, meta.configID)
		}
		if outcome.dodged {
			if outcome.innerPowerHeal > 0 {
				playerState := target.player.stateSnapshot()
				log.Printf("%s: NPC attack dodged id=%d config=%s inner-power dodge-heal=%d player_hp=%d/%d", remote, meta.id, meta.configID, outcome.innerPowerHeal, playerState.HP, playerState.MaxHP)
			} else {
				log.Printf("%s: NPC attack dodged id=%d config=%s", remote, meta.id, meta.configID)
			}
		} else if outcome.stateChanged {
			playerState := target.player.stateSnapshot()
			log.Printf("%s: NPC attack id=%d config=%s damage=%d player_hp=%d/%d", remote, meta.id, meta.configID, state.Damage, playerState.HP, playerState.MaxHP)
		} else if outcome.parried {
			log.Printf("%s: NPC attack parried id=%d config=%s", remote, meta.id, meta.configID)
		}
	})
	// 玩家侧战斗临场感（Fighting/Normal 迁移）按成员分别结算。
	changed := make(map[uint32]bool, len(members))
	for _, member := range members {
		if member.player == nil {
			continue
		}
		if member.player.setCombatPresence(threatened[member.objectID]) {
			changed[member.objectID] = true
		}
	}
	r.mu.Unlock()
	return &combatTickOutcome{members: members, moves: moves, updates: updates, changed: changed, rageFrames: rage}
}

// nearestTargetLocked 返回 aggroRange 内最近的存活目标；没有则返回 0。
func nearestTargetLocked(targets map[uint32]worldcore.Transform, from worldcore.Transform, aggroRange float32) uint32 {
	best := uint32(0)
	bestDistance := aggroRange
	for memberID, position := range targets {
		distance := horizontalDistance(from, position)
		if distance <= bestDistance {
			// 同距离时取成员 ID 较小者，保证线上行为确定。
			if distance == bestDistance && best != 0 && memberID > best {
				continue
			}
			best = memberID
			bestDistance = distance
		}
	}
	return best
}

// combatTickForTest 保留单视图独占场景的测试语义：把玩家登记为
// 唯一成员（位置即其坐标）后同步驱动一次场景 tick。
func (s *sceneLifecycle) combatTickForTest(player *playerActor, position role.Position, now time.Time) combatTickResult {
	if s == nil || player == nil {
		return combatTickResult{}
	}
	before := player.stateSnapshot()
	wireID := s.run.attachMember(&sceneMember{selfID: playerObjectID, player: player, view: s, position: position, active: true})
	s.mu.Lock()
	s.wireID = wireID
	s.mu.Unlock()
	// 独占测试视图没有目录驱动的可见集；按旧行为把全部已注册
	// NPC 视为可见（帧无过滤直发）。
	s.run.mu.Lock()
	s.run.eachNPCLocked(func(id worldcore.EntityID, meta *sceneEntity, _ *worldcore.Transform) {
		s.mu.Lock()
		s.visibleNPCs[meta.id] = struct{}{}
		s.mu.Unlock()
	})
	s.run.mu.Unlock()
	outcome := s.run.simulateLocked(now)
	if outcome != nil {
		s.run.broadcastNPCFrames(outcome.members, outcome.moves, outcome.updates)
	}
	after := player.stateSnapshot()
	return combatTickResult{playerStateChanged: after.LogicState != before.LogicState || after.HP != before.HP}
}

// scheduleCombatMoveArrival 只在客户端拥有一个移动间隔
// 来朝目的地播放动画之后，才确认最终位置。transform
// 比较即是取消令牌：持续追击每 tick 都会更新它，
// 因此过期的定时器绝不会确认一个陈旧的目的地。
// 定时器归场景运行时所有（epoch 防护），确认帧扇出给
// 当前视口内可见该 NPC 的成员。
func (r *sceneRun) scheduleCombatMoveArrival(members []*sceneMember, move npcMoveUpdate, delay time.Duration) {
	if delay <= 0 {
		delay = time.Second
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	epoch := r.epoch
	timer := time.AfterFunc(delay, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.closed || r.epoch != epoch {
			return
		}
		meta, exists := r.npcMeta(worldcore.EntityID(move.id))
		if !exists || meta.ownerID != move.ownerID {
			return
		}
		transform, transformOK := r.npcTransform(worldcore.EntityID(move.id))
		if !transformOK || !sameCombatTransform(transform, move.destination) {
			return
		}
		frame := serverLocation(meta.id, meta.ownerID, transform)
		for _, member := range members {
			if member.view == nil || !member.view.npcVisible(move.id) {
				continue
			}
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: confirm combat NPC move id=%d: %v", member.view.remote, meta.id, err)
			}
		}
	})
	r.timers = append(r.timers, timer)
	r.mu.Unlock()
}

// broadcastNPCFrames 在锁外把战斗 tick 产出的 NPC 移动/属性帧
// 扇出给视口内可见对应 NPC 的成员，并排布到达确认定时器。
func (r *sceneRun) broadcastNPCFrames(members []*sceneMember, moves []npcMoveUpdate, updates []npcPropertyUpdate) {
	for _, move := range moves {
		frame := serverMoving(move.id, move.ownerID, motionFromDest(move.destination))
		for _, member := range members {
			if member.view == nil || !member.view.npcVisible(move.id) {
				continue
			}
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: combat move NPC %d: %v", member.view.remote, move.id, err)
			}
		}
		// ServerLocation 在当前客户端中是到达确认。不要
		// 在同一个战斗 tick 中让它与 ServerMoving 成对发送：那样会使
		// actor 瞬移到目的地而不是播放移动动作。之后
		// 的追击目的地会在该定时器执行之前更改权威
		// transform，从而令此次确认失效。
		r.scheduleCombatMoveArrival(members, move, time.Second)
	}
	for _, update := range updates {
		frame, err := sceneObjectProperties(update.id, update.ownerID, 0, update.properties)
		if err != nil {
			log.Printf("scene %s/%s: encode combat NPC state %d: %v", r.key.config, r.key.resource, update.id, err)
			continue
		}
		for _, member := range members {
			if member.view == nil || !member.view.npcVisible(update.id) {
				continue
			}
			if err := member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: combat NPC state %d: %v", member.view.remote, update.id, err)
			}
		}
	}
}

func sameCombatTransform(a, b worldcore.Transform) bool {
	const epsilon = 0.01
	return math.Abs(float64(a.X-b.X)) < epsilon &&
		math.Abs(float64(a.Y-b.Y)) < epsilon &&
		math.Abs(float64(a.Z-b.Z)) < epsilon &&
		math.Abs(float64(a.Orient-b.Orient)) < epsilon
}

// setNPCTransformLocked 更新 NPC 权威坐标：可视权威 Scene 实体
// （revision 抬升，供各视图下一次 reconcile 出 Updates）与
// Transform 组件同步写。调用方必须已持有 r.mu。
func (r *sceneRun) setNPCTransformLocked(id worldcore.EntityID, destination worldcore.Transform) error {
	if _, exists := r.npcMeta(id); !exists {
		return fmt.Errorf("set transform for unknown NPC %d", id)
	}
	current, err := r.scene.Get(id)
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
	if err := r.scene.Replace(current.Entity); err != nil {
		return err
	}
	r.writeNpcTransform(id, destination)
	return nil
}

func npcCombatProperties(state entity.ActorState) []clientdata.IndexedProperty {
	hpRatio := int32(int64(state.HP) * 100 / int64(state.MaxHP))
	return []clientdata.IndexedProperty{
		{Index: 463, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)},
		{Index: 28, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)},
		{Index: 30, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)},
		{Index: 32, Name: "HP", Value: clientdata.Int32Value(state.HP)},
		{Index: 481, Name: "HitHP", Value: clientdata.Int32Value(state.HitHP)},
		{Index: propDead, Name: "Dead", Value: clientdata.ByteValue(boolByte(state.HP <= 0))},
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
