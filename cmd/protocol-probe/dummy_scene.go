package main

// 假人系统场景集成层：server1 共享 sceneRun 架构下的安装、每连接视口
// 准入与移动步进（对照 server2 的 per-connection 注册/准入/timer 链）。
//
//   安装：populate 一次性把匹配当前场景的全部假人注册进共享 scene
//   （display-only：EntityPlayer 类实体 + sceneEntity/Transform 两组件，
//   无 Vitals/Brain——被战斗 tick/仇恨/技能选靶天然排除；副本实例不放）。
//   DB 未就绪时挂起，tick 重试（首进图在开服装载完成前后都正确）。
//
//   准入：每连接视口维护 admitted 工作集（预算 110、单次进场 12、半径
//   60±20 迟滞、移动假人优先于站桩——server2 synchronizeDummyViewport
//   同语义）。reconcileViewportLocked 在 Plan 前按工作集过滤快照：未准入
//   假人不进 Plan，退场自然走既有 Removes→0x0E。
//
//   移动：sceneRun 1s tick 推进移动假人步进（固定路点 ping-pong / 随机
//   人流吸附官方 NPC 点），步起向「已准入该假人的成员」广播 0x20
//   serverMoving（真人成员移动同方言），到达广播 0x1F serverLocation；
//   ECS Transform 每拍写插值位置（重进视口 0x0D 补丁续走），可视权威
//   Scene 仅到达时 Replace（revision 不因巡逻抖动，patrolTick 同纪律）。

import (
	"log"
	"math"
	"math/rand"
	"sort"
	"sync/atomic"
	"time"

	"github.com/local/9yin-go-server/internal/ecs"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// ---- GM 安装上限（set_dummy_limit）----

// dummyInstallLimitLimitGM 上限取值域护栏（validate.go 同界）。
const dummyInstallLimitMaxGM = 100_000

// dummyInstallLimitState 进程级安装上限：-1=全量（默认），0=禁用，>0=每
// 场景按配置序取前 N 只安装（确定性：同一 DB 快照裁剪结果稳定）。GM
// set_dummy_limit 写入，installDummiesLocked 与 applyDummyLimitLocked 读取。
var dummyInstallLimitState atomic.Int32

func init() {
	dummyInstallLimitState.Store(-1)
}

func setDummyInstallLimit(limit int32) {
	dummyInstallLimitState.Store(limit)
}

func currentDummyInstallLimit() int32 {
	return dummyInstallLimitState.Load()
}

// clampDummyInstall 按全局上限裁剪待安装清单（installDummiesLocked 用）。
func clampDummyInstall(matched []dummyPlayer) []dummyPlayer {
	if keep := desiredDummyCount(len(matched)); keep < len(matched) {
		return matched[:keep]
	}
	return matched
}

// ---- 注册结构 ----

// registeredDummy 是已注册进场景的一只假人：配置快照 + 静态出生位（站桩
// 测距用）+ 移动权威步进（nil = 站桩）。ID = dummyObjectIDBase + 场景内
// 序号（稳定：重进视口、重连都不换 ID）。
type registeredDummy struct {
	id       uint32
	config   dummyPlayer
	position worldcore.Transform
	motion   *dummyMotionState
}

// dummyMotionState 是移动假人的权威步进状态。
type dummyMotionState struct {
	id   uint32
	pos  worldcore.Transform // 当前权威位置（步起=from，到达时=to）
	from worldcore.Transform
	to   worldcore.Transform
	// moving：[stepStart, stepEnd] 行进至 to；否则站桩 dwell 到 nextMove。
	moving    bool
	stepStart time.Time
	stepEnd   time.Time
	nextMove  time.Time
	speed     float32
	// 固定路点（route != nil）ping-pong；随机人流用 pool 吸附采样。
	route      []worldcore.Transform
	routeIndex int
	routeBack  bool
	rng        *rand.Rand
}

// dummyNodeHold 到站停留（server2 patrolNodeHold 同值）。
const dummyNodeHold = 2 * time.Second

// ---- 安装 ----

// installDummiesLocked 按场景过滤 DB 快照并全量注册。populate（持有
// run.mu）与 tickOnce 的延迟重试共用。副本实例（key.instance 非空）不放
// 人群。
func (r *sceneRun) installDummiesLocked(catalog []npcSpawn) {
	if r.dummiesInstalled || r.key.instance != "" {
		return
	}
	dummies := dummyPlayersSnapshot()
	if dummies == nil {
		// DB 未就绪（或装载失败）：挂起，tick 在装载成功后重试安装。
		if !r.dummiesDeferred {
			r.dummiesDeferred = true
			log.Printf("scene %s/%s: dummy store not ready; deferring crowd install", r.key.config, r.key.resource)
		}
		return
	}
	matched := make([]dummyPlayer, 0, 64)
	for _, dummy := range dummies {
		if dummy.matchesScene(r.key.config, r.key.resource) {
			matched = append(matched, dummy)
		}
	}
	if len(matched) == 0 {
		r.dummiesInstalled = true
		r.dummiesDeferred = false
		return
	}
	now := time.Now()
	matched = clampDummyInstall(matched)
	r.dummies = make([]registeredDummy, 0, len(matched))
	r.dummyMotion = nil
	r.dummyPatrolPoints = dummyPatrolPointsFromCatalog(catalog)
	for index, dummy := range matched {
		id := dummyObjectIDBase + uint32(index)
		if !r.registerDummyAtLocked(id, dummy, now) {
			continue
		}
	}
	r.dummiesInstalled = true
	r.dummiesDeferred = false
	movers := 0
	for _, dummy := range r.dummies {
		if dummy.motion != nil {
			movers++
		}
	}
	log.Printf("scene %s/%s: installed %d dummy player(s) (movers=%d, patrol pool=%d)",
		r.key.config, r.key.resource, len(r.dummies), movers, len(r.dummyPatrolPoints))
}

// registerDummyAtLocked 注册单只假人：预生成 0x0D 归档、加进可视权威
// Scene（EntityPlayer 类、ownerID=1 与远程成员同款）、写 sceneEntity/
// Transform 两组件（无 Vitals/Brain）。移动假人建权威步进状态。
// 调用方必须已持有 r.mu。
func (r *sceneRun) registerDummyAtLocked(id uint32, dummy dummyPlayer, now time.Time) bool {
	entityID := worldcore.EntityID(id)
	if _, exists := r.npcMeta(entityID); exists {
		return false
	}
	payload := dummyAddObject(id, dummy, now)
	transform := worldcore.Transform{
		X: dummy.Pos.X, Y: dummy.Pos.Y, Z: dummy.Pos.Z, Orient: dummy.Pos.Orient,
	}
	if err := r.scene.Add(worldcore.Entity{
		ID:        entityID,
		OwnerID:   1,
		Kind:      worldcore.EntityPlayer,
		Transform: transform,
	}); err != nil {
		log.Printf("dummy players: register %s: %v", dummy.Name, err)
		return false
	}
	entry := registeredDummy{id: id, config: dummy, position: transform}
	meta := sceneEntity{
		id:          id,
		description: "dummy:" + dummy.Name,
		ownerID:     1,
		configID:    "dummy",
		scriptClass: "Dummy",
		// 可选中（LastObject/查看），但绝不弹 NPC 菜单（远程成员同款）。
		interaction: npcInteractionSelectOnly,
		payload:     payload,
	}
	raw := r.combat.Raw()
	ecs.Write(raw, ecs.Id(entityID), ecs.C(transform), ecs.C(meta))
	if dummyMoves(dummy) {
		if r.dummyMotion == nil {
			r.dummyMotion = make(map[uint32]*dummyMotionState)
		}
		motion := &dummyMotionState{
			id:    id,
			pos:   transform,
			speed: dummyMoveSpeed(dummy),
			rng:   rand.New(rand.NewSource(int64(id))),
		}
		if len(dummy.Patrol) >= 2 {
			motion.route = make([]worldcore.Transform, 0, len(dummy.Patrol))
			for _, point := range dummy.Patrol {
				motion.route = append(motion.route, worldcore.Transform{
					X: point.X, Y: point.Y, Z: point.Z, Orient: point.Orient,
				})
			}
		}
		motion.nextMove = now.Add(time.Duration(motion.rng.Int63n(int64(dummyNodeHold))))
		entry.motion = motion
		r.dummyMotion[id] = motion
	}
	r.dummies = append(r.dummies, entry)
	return true
}

// ---- 位置查询（准入/物化共用）----

// dummyPositionOf 返回假人当前权威位置（移动假人为插值位置）。
// 调用方必须已持有 r.mu。
func (r *sceneRun) dummyPositionOf(id uint32, now time.Time) (worldcore.Transform, bool) {
	if motion, exists := r.dummyMotion[id]; exists {
		return motion.interpolated(now), true
	}
	// ID 与序号线性对应（dummyObjectIDBase + index）：反解 O(1)。
	index := int(id - dummyObjectIDBase)
	if index < 0 || index >= len(r.dummies) || r.dummies[index].id != id {
		return worldcore.Transform{}, false
	}
	return r.dummies[index].position, true
}

// interpolated 返回步进插值位置（站桩段= pos）。
func (m *dummyMotionState) interpolated(now time.Time) worldcore.Transform {
	if !m.moving {
		return m.pos
	}
	total := m.stepEnd.Sub(m.stepStart)
	if total <= 0 {
		return m.to
	}
	progress := float64(now.Sub(m.stepStart)) / float64(total)
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	return worldcore.Transform{
		X:      m.from.X + (m.to.X-m.from.X)*float32(progress),
		Y:      m.from.Y + (m.to.Y-m.from.Y)*float32(progress),
		Z:      m.from.Z + (m.to.Z-m.from.Z)*float32(progress),
		Orient: m.pos.Orient,
	}
}

// ---- 每连接视口准入 ----

// 假人距离视口常量（对齐 server2：主城不一次性物化上千玩家模型）。
const (
	dummyViewportObjectBudget        = 110
	dummyViewportAdmissionPerRequest = 12
	dummyViewportRadius              = 60.0
	dummyViewportHysteresis          = 20.0
)

// dummyAdmissionLocked 按本视口玩家位置维护假人 admitted 工作集：
//   - 已活跃假人在 exit 半径内保留，超出即退场（下轮 Plan 生成 0x0E）；
//   - 新入场先吸收**移动假人**（按距离排序），再补站桩——保证可见范围
//     内始终有人在走动；
//   - maxNew 限制单次注册数，配合移动包/1s tick 平滑长出人群。
//
// 返回工作集是否变化（变化时调用方触发 reconcile）。要求调用方已持有
// s.run.mu；工作集经 s.mu 写入。
func (s *sceneLifecycle) dummyAdmissionLocked() bool {
	if s == nil || s.run == nil || len(s.run.dummies) == 0 {
		return false
	}
	now := time.Now()
	s.mu.Lock()
	center := s.center
	current := s.dummyVisible
	s.mu.Unlock()

	enterRadius := dummyViewportRadius
	if len(current) != 0 {
		enterRadius -= dummyViewportHysteresis
		if enterRadius < 0 {
			enterRadius = 0
		}
	}
	exitRadius := dummyViewportRadius + dummyViewportHysteresis
	enterSquared := float32(enterRadius * enterRadius)
	exitSquared := float32(exitRadius * exitRadius)

	type dummyCandidate struct {
		id  uint32
		d2  float32
		dix int
	}
	var retained, moverEntrants, standEntrants []dummyCandidate
	for index := range s.run.dummies {
		entry := &s.run.dummies[index]
		position, ok := s.run.dummyPositionOf(entry.id, now)
		if !ok {
			continue
		}
		dx := position.X - center.X
		dz := position.Z - center.Z
		distanceSquared := dx*dx + dz*dz
		_, alreadyActive := current[entry.id]
		limitSquared := enterSquared
		if alreadyActive {
			limitSquared = exitSquared
		}
		if distanceSquared > limitSquared {
			continue
		}
		candidate := dummyCandidate{id: entry.id, d2: distanceSquared, dix: index}
		if alreadyActive {
			retained = append(retained, candidate)
		} else if entry.motion != nil {
			moverEntrants = append(moverEntrants, candidate)
		} else {
			standEntrants = append(standEntrants, candidate)
		}
	}
	sortCandidates := func(candidates []dummyCandidate) {
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].d2 == candidates[j].d2 {
				return candidates[i].id < candidates[j].id
			}
			return candidates[i].d2 < candidates[j].d2
		})
	}
	sortCandidates(retained)
	sortCandidates(moverEntrants)
	sortCandidates(standEntrants)

	// 预算是稳定工作集：先全额保留仍有效的活跃假人，再按「移动优先、
	// 站桩补齐」吸收新入场。
	desired := make(map[uint32]struct{}, dummyViewportObjectBudget)
	remaining := dummyViewportObjectBudget
	for _, candidate := range retained {
		if remaining == 0 {
			break
		}
		desired[candidate.id] = struct{}{}
		remaining--
	}
	admitted := 0
	admitEntrants := func(candidates []dummyCandidate) {
		for _, candidate := range candidates {
			if remaining == 0 || admitted >= dummyViewportAdmissionPerRequest {
				return
			}
			desired[candidate.id] = struct{}{}
			remaining--
			admitted++
		}
	}
	admitEntrants(moverEntrants)
	admitEntrants(standEntrants)

	changed := len(desired) != len(current)
	if !changed {
		for id := range desired {
			if _, exists := current[id]; !exists {
				changed = true
				break
			}
		}
	}
	if !changed {
		return false
	}
	s.mu.Lock()
	s.dummyVisible = desired
	s.mu.Unlock()
	return true
}

// dummyVisibleOf 报告该假人是否在本视口 admitted 工作集内（帧扇出过滤，
// npcVisible 同款锁纪律）。
func (s *sceneLifecycle) dummyVisibleOf(id uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, visible := s.dummyVisible[id]
	return visible
}

// filterDummySnapshot 把快照里未准入的假人实体剔除：Plan 只看得见工作集
// 内的假人，退场自然走 Removes→0x0E。
func (s *sceneLifecycle) filterDummySnapshot(snapshot worldcore.Snapshot) worldcore.Snapshot {
	if len(s.run.dummies) == 0 {
		return snapshot
	}
	s.mu.Lock()
	visible := s.dummyVisible
	s.mu.Unlock()
	if len(visible) == 0 {
		// 尚无准入假人：整段剔除（fast path）。
		if !snapshotHasDummy(snapshot) {
			return snapshot
		}
		filtered := snapshot
		filtered.Entities = make([]worldcore.VersionedEntity, 0, len(snapshot.Entities))
		for _, entry := range snapshot.Entities {
			if isDummyObjectID(uint32(entry.Entity.ID)) {
				continue
			}
			filtered.Entities = append(filtered.Entities, entry)
		}
		return filtered
	}
	if !snapshotHasDummy(snapshot) {
		return snapshot
	}
	filtered := snapshot
	filtered.Entities = make([]worldcore.VersionedEntity, 0, len(snapshot.Entities))
	for _, entry := range snapshot.Entities {
		if isDummyObjectID(uint32(entry.Entity.ID)) {
			if _, keep := visible[uint32(entry.Entity.ID)]; !keep {
				continue
			}
		}
		filtered.Entities = append(filtered.Entities, entry)
	}
	return filtered
}

func snapshotHasDummy(snapshot worldcore.Snapshot) bool {
	for _, entry := range snapshot.Entities {
		if isDummyObjectID(uint32(entry.Entity.ID)) {
			return true
		}
	}
	return false
}

// ---- 移动步进（中央 tick）----

// dummyTick 是 tickOnce 的假人段：延迟安装重试 → 移动步进 → 各视口
// 准入巡检（变化者锁外 reconcile）。
func (r *sceneRun) dummyTick(now time.Time) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	if r.dummiesDeferred && !r.dummiesInstalled {
		// 安装在锁内做（DB 快照已就绪时）；catalog 为空也能装（仅随机
		// 人流池为空，固定路点不受影响）。
		r.installDummiesLocked(r.catalog)
	}
	if r.dummiesInstalled && len(r.dummies) > 0 && len(r.members) > 0 {
		r.dummyMotionTickLocked(now)
	}
	var changedViews []*sceneLifecycle
	if r.dummiesInstalled && len(r.dummies) > 0 {
		for _, member := range r.members {
			if member.view == nil {
				continue
			}
			if member.view.dummyAdmissionLocked() {
				changedViews = append(changedViews, member.view)
			}
		}
	}
	r.mu.Unlock()
	for _, view := range changedViews {
		if err := view.reconcileViewport(); err != nil {
			log.Printf("dummy viewport reconcile: %v", err)
		}
	}
}

// dummyMotionTickLocked 推进全部移动假人一步。调用方必须已持有 r.mu；
// 帧写入在锁内直发（WriteFrame 线程安全，broadcastNPCLocked 同款）。
func (r *sceneRun) dummyMotionTickLocked(now time.Time) {
	for _, motion := range r.dummyMotion {
		if motion.moving {
			if now.Before(motion.stepEnd) {
				// 行进中：ECS Transform 保持插值位置（重进视口 0x0D 补丁
				// 与准入测距读它），可视权威 Scene 不动。
				r.writeNpcTransform(worldcore.EntityID(motion.id), motion.interpolated(now))
				continue
			}
			// 到达：权威位置落终点，广播 0x1F，进入站桩 dwell。
			motion.pos = motion.to
			motion.moving = false
			motion.nextMove = now.Add(dummyNodeHold)
			r.setDummyTransformLocked(motion.id, motion.pos)
			r.broadcastDummyLocked(motion.id, serverLocation(motion.id, 1, motion.pos))
			continue
		}
		if now.Before(motion.nextMove) {
			continue
		}
		// 起步：挑下一目的地，广播 0x20 serverMoving（真人成员移动方言）。
		destination, duration := r.nextDummyStepLocked(motion, now)
		motion.from = motion.pos
		motion.to = destination
		motion.stepStart = now
		motion.stepEnd = now.Add(duration)
		motion.moving = true
		motion.pos.Orient = destination.Orient
		r.broadcastDummyLocked(motion.id, serverMoving(motion.id, 1, motionFromDest(destination)))
	}
}

// setDummyTransformLocked 到达时同步 ECS 与可视权威 Scene 的权威位置
// （revision 抬升一次；行进中只写 ECS，patrolTick 同纪律）。
// 调用方必须已持有 r.mu。
func (r *sceneRun) setDummyTransformLocked(id uint32, transform worldcore.Transform) {
	entityID := worldcore.EntityID(id)
	r.writeNpcTransform(entityID, transform)
	if current, err := r.scene.Get(entityID); err == nil {
		current.Entity.Transform = transform
		_ = r.scene.Replace(current.Entity)
	}
}

// nextDummyStepLocked 挑下一目的地：固定路点 ping-pong（routePatrolStep
// 同构）；随机人流先均匀随机方向 8~40m、再吸附 15m 内最近的官方 NPC 点
// （randomPatrolStep 同构——池采样会随玩家站位偏聚一侧，均匀方向从根上
// 消除汇聚）。
func (r *sceneRun) nextDummyStepLocked(motion *dummyMotionState, now time.Time) (worldcore.Transform, time.Duration) {
	if motion.route != nil && len(motion.route) >= 2 {
		return dummyRouteStep(motion)
	}
	pool := r.dummyPatrolPoints
	if len(pool) == 0 {
		// 无官方坐标池：原地小憩，避免除零。
		return motion.pos, dummyNodeHold
	}
	from := motion.pos
	angle := motion.rng.Float64() * 2 * math.Pi
	distance := 8 + motion.rng.Float64()*32
	targetX := from.X + float32(math.Cos(angle)*distance)
	targetZ := from.Z + float32(math.Sin(angle)*distance)
	destination := worldcore.Transform{}
	snapped := false
	bestD2 := float32(15 * 15)
	for i := range pool {
		dx := pool[i].X - targetX
		dz := pool[i].Z - targetZ
		if d2 := dx*dx + dz*dz; d2 < bestD2 {
			bestD2 = d2
			destination = pool[i]
			snapped = true
		}
	}
	if !snapped {
		destination = pool[motion.rng.Intn(len(pool))]
		for attempt := 1; attempt < 8; attempt++ {
			if d := horizontalDistance(from, destination); d >= 8 && d <= 90 {
				break
			}
			destination = pool[motion.rng.Intn(len(pool))]
		}
	} else if horizontalDistance(from, destination) < 2 {
		// 吸附点落在脚下（罕见）：回退池采样挑 ≥8m 目的地，避免原地换向。
		for attempt := 0; attempt < 8; attempt++ {
			destination = pool[motion.rng.Intn(len(pool))]
			if horizontalDistance(from, destination) >= 8 {
				break
			}
		}
	}
	destination.Orient = orientTowardTarget(from.X, from.Z, destination.X, destination.Z)
	seconds := horizontalDistance(from, destination) / motion.speed
	if seconds < 1 {
		seconds = 1
	}
	return destination, time.Duration(seconds * float32(time.Second))
}

// dummyRouteStep 固定路点 ping-pong 前进一格（server2 routePatrolStep 同构）。
func dummyRouteStep(motion *dummyMotionState) (worldcore.Transform, time.Duration) {
	nodes := motion.route
	if motion.routeIndex < 0 || motion.routeIndex >= len(nodes) {
		motion.routeIndex = 0
		motion.routeBack = false
	}
	forward := !motion.routeBack
	if forward && motion.routeIndex >= len(nodes)-1 {
		forward = false
	} else if !forward && motion.routeIndex <= 0 {
		forward = true
	}
	target := motion.routeIndex
	if forward {
		target++
	} else {
		target--
	}
	motion.routeIndex = target
	motion.routeBack = !forward
	destination := motion.pos
	destination.X, destination.Y, destination.Z = nodes[target].X, nodes[target].Y, nodes[target].Z
	destination.Orient = orientTowardTarget(motion.pos.X, motion.pos.Z, destination.X, destination.Z)
	dx := destination.X - motion.pos.X
	dz := destination.Z - motion.pos.Z
	dy := destination.Y - motion.pos.Y
	distance := float32(math.Sqrt(float64(dx*dx + dz*dz + dy*dy)))
	seconds := distance / motion.speed
	if seconds < 0.5 {
		seconds = 0.5
	}
	return destination, time.Duration(seconds * float32(time.Second))
}

// broadcastDummyLocked 把单帧扇出给 admitted 工作集含该假人的成员
// （broadcastNPCLocked 同款：锁内直发，WriteFrame 线程安全）。
// 调用方必须已持有 r.mu。
func (r *sceneRun) broadcastDummyLocked(id uint32, frame []byte) {
	for _, member := range r.sortedMembersLocked() {
		if member.view == nil || !member.view.dummyVisibleOf(id) {
			continue
		}
		if err := member.view.conn.WriteFrame(frame); err != nil {
			log.Printf("dummy broadcast %d: %v", id, err)
		}
	}
}

// ---- GM 上限应用 ----

// desiredDummyCount 全局上限下的期望安装量（matched=本场景匹配数）。
func desiredDummyCount(matched int) int {
	limit := currentDummyInstallLimit()
	if limit < 0 || int(limit) >= matched {
		return matched
	}
	return int(limit)
}

// applyDummyLimitLocked 把全局安装上限应用到本场景（GM set_dummy_limit
// 后立即生效）：期望规模与当前已安装不一致时整场重装——卸载者走各成员
// 下一次 reconcile 的 0x0E，补回/重装者走 0x0D（ID 确定性不变）。
// 调用方必须已持有 r.mu。
func (r *sceneRun) applyDummyLimitLocked() {
	if !r.dummiesInstalled {
		return // 未安装：installDummiesLocked 自然按上限裁剪
	}
	dummies := dummyPlayersSnapshot()
	if dummies == nil {
		return
	}
	matched := 0
	for _, dummy := range dummies {
		if dummy.matchesScene(r.key.config, r.key.resource) {
			matched++
		}
	}
	if desiredDummyCount(matched) == len(r.dummies) {
		return
	}
	r.reinstallDummiesLocked()
}

// reinstallDummiesLocked 卸载全部已安装假人并按当前全局上限重装。
// 调用方必须已持有 r.mu。
func (r *sceneRun) reinstallDummiesLocked() {
	for i := range r.dummies {
		_ = r.unregisterNPC(worldcore.EntityID(r.dummies[i].id))
	}
	r.dummies = nil
	r.dummyMotion = nil
	r.dummiesInstalled = false
	r.installDummiesLocked(r.catalog)
}

// applyDummyLimitToAllScenes 把新的全局上限应用到全部共享场景并触发成员
// 视口对账（GM set_dummy_limit 执行器用）。返回（有假人的场景数, 当前
// 安装总数）。
func applyDummyLimitToAllScenes() (int, int) {
	sceneRegistryMu.Lock()
	runs := make([]*sceneRun, 0, len(sceneRegistry))
	for _, run := range sceneRegistry {
		runs = append(runs, run)
	}
	sceneRegistryMu.Unlock()
	scenes, total := 0, 0
	for _, run := range runs {
		var views []*sceneLifecycle
		run.mu.Lock()
		if !run.closed && run.dummiesInstalled {
			run.applyDummyLimitLocked()
			if len(run.dummies) > 0 {
				scenes++
				total += len(run.dummies)
			}
			for _, member := range run.members {
				if member.view != nil {
					views = append(views, member.view)
				}
			}
		}
		run.mu.Unlock()
		for _, view := range views {
			if err := view.reconcileViewport(); err != nil {
				log.Printf("dummy limit reconcile: %v", err)
			}
		}
	}
	return scenes, total
}
