package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/luascript"
	"github.com/local/9yin-go-server/internal/npcfunc"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/sim"
	"github.com/local/9yin-go-server/internal/taskdata"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

type sceneMessageConnection interface {
	WriteFrame([]byte) error
}

// sceneLifecycle 持有服务器侧的场景对象以及当前关联到单个客户端视口
// 的子集。它刻意与 NPC 行为解耦，使怪物、采集物和远端玩家可以走同
// 一条路径。
//
// Phase 3a 起 sceneLifecycle 是**每连接视图**：世界（sim 世界、可视
// 权威 Scene、定时器、中央 tick）归属共享的 sceneRun；视图只保留
// 本连接的协议状态。字段分两类锁纪律：
//   - view.mu 保护会被 run 定时器/tick goroutine 跨协程读取的
//     字段（closed、visibleNPCs、center/radius、replayTimers）；
//   - 其余字段（armed、selected*、lastMenu*、viewport）仅被属主
//     连接 goroutine 访问，无锁。
type sceneLifecycle struct {
	remote string
	conn   sceneMessageConnection
	// view.mu 只保护下列四个跨 goroutine 字段；不要用它保护世界
	// 状态——世界一律在 run.mu 下访问。
	mu           sync.Mutex
	closed       bool
	visibleNPCs  map[uint32]struct{}
	center       role.Position
	radius       float32
	replayTimers []*time.Timer

	run      *sceneRun
	registry bool // 生产连接经全局注册表共享场景；测试默认独占

	// scripts 是可选的服务端 Lua 脚本宿主。nil 表示禁用；
	// npc_service 等未实现的原生 NPC 服务会交给脚本处理器。
	scripts *luascript.Host

	armed          bool
	viewport       *worldcore.Viewport
	memberID       uint32 // 本机 1-1 身份（恒 1）
	wireID         uint32 // 场景内远程可见身份（attach 时由 run 分配）
	selectedObject uint32
	selectedOwner  uint32
	// openDropBox 是当前打开的掉落箱窗口（View 80，loot_box.go）；
	// 仅属主连接 goroutine 访问，无需锁。
	openDropBox    *openDropBoxState
	lastMenuObject uint32
	lastMenuOwner  uint32
	lastMenuAt     time.Time
	lastMenuMovie  bool
}

// sceneEntity 是 NPC 的注册元数据组件：模板面（configID/scriptClass/
// interaction/services/portal/funcs）、协议面（payload 物化字节、
// properties）与巡逻面（schedule，目录缓存内共享的绝对时间表；nil 即
// 原地站立）。高频空间态在独立的 worldcore.Transform 组件中。作为
// ECS 组件存于 s.combat，与 Vitals/Brain 同实体。
type sceneEntity struct {
	id          uint32
	description string
	payload     []byte
	ownerID     uint32
	properties  []clientdata.IndexedProperty
	schedule    *patrolSchedule
	configID    string
	scriptClass string
	interaction npcInteraction
	funcs       []npcfunc.Binding
	services    []npcService
	portal      *scenePortalRoute
	// model/wideName 供 msg24 战斗帧携带：模型窄串（Material 列）与
	// 宽串名（NpcNickName 列）。官方表缺列时为空串，帧仍发。
	model    string
	wideName string
	// normalSkill 是该 NPC 的攻击招式节名（NormalSkill 列，经
	// zhaoshi_npc.ini 节名校验；空=用 jn_mon053_02 常量兜底）。
	normalSkill string
	// combatStats 是实力包推导的守方战斗数值（注册期一次，静态；
	// 2026-11-08 NPC 属性块迁移批次）。nil＝未推导（保持零值守方）。
	combatStats *npcCombatStats
	// lootBox 非 nil 表示这是掉落宝箱（点击开 View 80，不走对话/选靶）。
	lootBox *lootBoxRuntime
	// chase 是路点图追击状态（mapnav_chase.go）：A* 折线/重规划节流/
	// 跨分量重试计数。nil＝尚未进入追击。清仇恨/重生时置 nil。
	// 只在 run.mu 内经 eachCombatNPCFullLocked 的组件指针访问。
	chase *npcChaseState
}

// npcCombatStatsOf 取指定 NPC 的守方战斗数值（rollSkillHit 按伤害系选防
// 用）。要求调用方未持 s.run.mu（内部自取；miss 目标返回 nil）。
func (s *sceneLifecycle) npcCombatStatsOf(ref skillDamageTargetRef) *npcCombatStats {
	if s == nil {
		return nil
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	meta, ok := s.run.npcMeta(worldcore.EntityID(ref.id))
	if !ok || meta.ownerID != ref.ownerID {
		return nil
	}
	return meta.combatStats
}

// ---- sceneEntity / Transform 组件访问器（委托场景运行时） ----
// 全部方法要求调用方已持有 s.run.mu；组件存储与 Vitals/Brain
// 同一世界，属 sceneRun 所有。

// spawnSceneEntityLocked 用一次 ecs.Write 写入实体的全部组件
// （Vitals/Transform/sceneEntity，敌对另加 Brain），避免分次补写
// 造成 archetype 搬移。registerNPC 与测试共用。
// 调用方必须已持有 s.run.mu。
func (s *sceneLifecycle) spawnSceneEntityLocked(id worldcore.EntityID, vitals sim.Vitals, transform worldcore.Transform, meta sceneEntity, brain *sim.Brain) {
	s.run.spawnSceneEntityLocked(id, vitals, transform, meta, brain)
}

func (s *sceneLifecycle) npcMeta(id worldcore.EntityID) (sceneEntity, bool) {
	return s.run.npcMeta(id)
}

// writeNpcMeta 写回注册元数据。
// 调用方必须已持有 s.run.mu。
func (s *sceneLifecycle) writeNpcMeta(id worldcore.EntityID, meta sceneEntity) {
	s.run.writeNpcMeta(id, meta)
}

func (s *sceneLifecycle) npcTransform(id worldcore.EntityID) (worldcore.Transform, bool) {
	return s.run.npcTransform(id)
}

// writeNpcTransform 是 NPC 权威坐标的唯一写点；Scene 实体的
// Transform/Posi* 同步由 setNPCTransformLocked/moveNPC 负责对账。
// 调用方必须已持有 s.run.mu。
func (s *sceneLifecycle) writeNpcTransform(id worldcore.EntityID, transform worldcore.Transform) {
	s.run.writeNpcTransform(id, transform)
}

// eachNPC 以 EntityID 升序回调全部已注册 NPC（平民与敌对均含）。
// 回调收到的是组件副本，修改后经 writeNpcMeta/writeNpcTransform 写回。
// 调用方必须已持有 s.run.mu。
func (s *sceneLifecycle) eachNPC(fn func(id worldcore.EntityID, meta *sceneEntity, transform *worldcore.Transform)) {
	s.run.eachNPCLocked(fn)
}

func cloneIndexedProperties(source []clientdata.IndexedProperty) []clientdata.IndexedProperty {
	return append([]clientdata.IndexedProperty(nil), source...)
}

func worldValue(value clientdata.Value) (worldcore.Value, error) {
	switch value.Type {
	case clientdata.WireByte:
		return worldcore.Byte(value.U8), nil
	case clientdata.WireWord:
		return worldcore.Word(value.U16), nil
	case clientdata.WireInt32:
		return worldcore.Int32(value.I32), nil
	case clientdata.WireFloat32:
		return worldcore.Float32(value.F32), nil
	case clientdata.WireString:
		return worldcore.String(value.Text), nil
	case clientdata.WireWideString:
		return worldcore.WideString(value.Text), nil
	default:
		return worldcore.Value{}, fmt.Errorf("unsupported client property kind %d", value.Type)
	}
}

// npcInteraction 是一个服务器侧的路由决策，由现代模板 ScriptClass 加上
// 本地服务器实际实现的业务字段推导而来。它刻意把目标选择与打开 NPC
// 对话分开：战斗目标仍需要 LastObject 来做技能选靶，但绝不能收到一个
// 虚构的对话窗口。
type npcInteraction uint8

const (
	npcInteractionSelectOnly npcInteraction = iota
	npcInteractionTalk
	npcInteractionCombat
)

func (kind npcInteraction) String() string {
	switch kind {
	case npcInteractionTalk:
		return "talk"
	case npcInteractionCombat:
		return "combat"
	default:
		return "select-only"
	}
}

// npcService 直接由现代模板行推导而来。与旧的 Npc_Func 表不同，这些值
// 使用当前 creator XML 引用的确切 ConfigID，因此对本地现代客户端
// 有效。
type npcService struct {
	mark   uint16
	label  string
	value  string
	source string
}

// 服务 mark 是随当前客户端原生 0x1C SERVER_MENU 携带的服务器本地路由
// ID。它们刻意不复用旧 Npc_Func 的整数命名空间：被选中的 mark 总是
// 相对于发出该菜单的 NPC
// 进行解析。
const (
	markShop      uint16 = 0x1001
	markDepot     uint16 = 0x1002
	markTransport uint16 = 0x1003
	markJob       uint16 = 0x1004
	markGem       uint16 = 0x1005
	markGuild     uint16 = 0x1006
	markHome      uint16 = 0x1007
	markTask      uint16 = 0x1008
)

type sceneLocationReplay struct {
	id      uint32
	payload []byte
}

var actor2LocationReplayDelays = []time.Duration{
	250 * time.Millisecond,
	750 * time.Millisecond,
	1500 * time.Millisecond,
	3 * time.Second,
	5 * time.Second,
}

func newSceneLifecycle(remote string, conn sceneMessageConnection, registries ...*npcfunc.Registry) *sceneLifecycle {
	var funcs *npcfunc.Registry
	if len(registries) > 0 {
		funcs = registries[0]
	}
	view := &sceneLifecycle{
		remote:      remote,
		conn:        conn,
		viewport:    worldcore.NewViewport(),
		visibleNPCs: make(map[uint32]struct{}),
		memberID:    playerObjectID,
		run:         newSceneRun(sceneKey{}, funcs),
	}
	return view
}

// newSharedSceneLifecycle 构造走全局注册表的生产视图：
// 同一 (config, resource) 的连接共享同一份场景模拟。
func newSharedSceneLifecycle(remote string, conn sceneMessageConnection, funcs *npcfunc.Registry) *sceneLifecycle {
	view := newSceneLifecycle(remote, conn, funcs)
	view.registry = true
	return view
}

// attachMember 把本视图的玩家登记进场景运行时并启动中央 tick。
// 带 player 的成员会被物化为远程可见实体；visual 来自角色外观，
// 供其他客户端的 0x0D 物化使用。
func (s *sceneLifecycle) attachMember(player *playerActor, position role.Position, visual roleVisual) {
	s.wireID = s.run.attachMember(&sceneMember{
		selfID: playerObjectID, player: player, view: s, position: position,
		active: s.combatActive(), visual: visual,
	})
}

// detachMember 注销成员；末成员离开会关闭整个场景运行时。
func (s *sceneLifecycle) detachMember() {
	if s.wireID != 0 {
		s.run.detachMember(s.wireID)
		s.wireID = 0
	}
}

// updateMemberPosition 同步场景侧权威位置并刷新本视图可见集。
// 返回可见集是否发生变化（调用方据此触发 reconcile）。
func (s *sceneLifecycle) updateMemberPosition(position role.Position, catalog []npcSpawn) bool {
	if s.wireID != 0 {
		s.run.updateMemberPosition(s.wireID, position)
	}
	return s.refreshVisibility(catalog, position)
}

// playerLiveTransform 返回本视图玩家的场景权威位置（成员表条目；
// m15/212 移动上报与冲锋链落点经 updateMemberPosition 持续刷新）。
// 玩家物化进 combat ECS 的键是 wireID 且无 Vitals 组件，按全局
// playerObjectID 走 combatNPCTransform 恒 miss——可移动范围技逐段
// 索敌的实时圆心必须从这里取（2026-11-13 太极剑舞真机修复）。
func (s *sceneLifecycle) playerLiveTransform() (worldcore.Transform, bool) {
	if s == nil || s.wireID == 0 {
		return worldcore.Transform{}, false
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	member, exists := s.run.members[s.wireID]
	if !exists {
		return worldcore.Transform{}, false
	}
	return worldcore.Transform{
		X: member.position.X, Y: member.position.Y, Z: member.position.Z, Orient: member.position.Orient,
	}, true
}

// combatActive 报告本视图是否越过就绪屏障（供成员登记读取）。
func (s *sceneLifecycle) combatActive() bool {
	return s.armed == false && s.runActive()
}

func (s *sceneLifecycle) runActive() bool {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	if s.wireID == 0 {
		return false
	}
	member, exists := s.run.members[s.wireID]
	return exists && member.active
}

// refreshVisibility 重算半径内的 NPC id 集合（纯目录距离扫描，
// 无分配），供帧扇出过滤与 reconcile 触发判据使用。
func (s *sceneLifecycle) refreshVisibility(catalog []npcSpawn, position role.Position) bool {
	radiusSquared := s.radius * s.radius
	next := make(map[uint32]struct{}, len(s.visibleNPCs))
	for index, npc := range catalog {
		if !npcSpawnEligible(npc) {
			continue
		}
		dx := npc.x - position.X
		dz := npc.z - position.Z
		if dx*dx+dz*dz > radiusSquared {
			continue
		}
		next[forkNPCID(uint32(index)+2)] = struct{}{}
	}
	// 其他成员的远程实体与 NPC 共用同一可见集（帧扇出过滤统一）。
	s.run.mu.Lock()
	for wireID, member := range s.run.members {
		if wireID == s.wireID || member.player == nil {
			continue
		}
		dx := member.position.X - position.X
		dz := member.position.Z - position.Z
		if dx*dx+dz*dz <= radiusSquared {
			next[wireID] = struct{}{}
		}
	}
	// 动态实体宝箱（宝箱怪/GM 世界宝箱）共用同一可见集（物化/0x0E 扇出
	// 过滤统一）。挂尸掉落的载体是目录 NPC 本身，已由上方目录距离扫描
	// 覆盖，无需重复注入。
	for id, box := range s.run.lootBoxes {
		if !box.standalone {
			continue
		}
		transform, ok := s.run.npcTransform(worldcore.EntityID(id))
		if !ok {
			continue
		}
		dx := transform.X - position.X
		dz := transform.Z - position.Z
		if dx*dx+dz*dz <= radiusSquared {
			next[id] = struct{}{}
		}
	}
	s.run.mu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.center = position
	if len(next) == len(s.visibleNPCs) {
		same := true
		for id := range next {
			if _, exists := s.visibleNPCs[id]; !exists {
				same = false
				break
			}
		}
		if same {
			return false
		}
	}
	s.visibleNPCs = next
	return true
}

// begin 切换视图到共享世界场景（instance 为空的 sceneKey）。副本
// 实例经 beginKey 携带独立实例键进入。
func (s *sceneLifecycle) begin(config, resource string, catalog []npcSpawn) error {
	return s.beginKey(sceneKey{config: config, resource: resource}, catalog)
}

// beginKey 切换视图到目标场景：离开当前运行时，再（经注册表或
// 独占新建）挂靠目标运行时并一次性注册全量目录。视图自身的
// 就绪屏障与菜单状态随世代重置。key.instance 非空时挂靠的是
// 独立副本实例运行时（同 config 多实例互不干扰）。
func (s *sceneLifecycle) beginKey(key sceneKey, catalog []npcSpawn) error {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return nil
	}
	s.detachMember()
	s.stopReplayTimers()
	var run *sceneRun
	if s.registry {
		acquired, err := acquireSceneRun(key, catalog, s.run.npcFuncs)
		if err != nil {
			return err
		}
		run = acquired
	} else {
		run = newSceneRun(key, s.run.npcFuncs)
		if err := run.populate(catalog); err != nil {
			return err
		}
	}
	s.run = run
	s.armed = false
	s.viewport = worldcore.NewViewport()
	s.mu.Lock()
	s.visibleNPCs = make(map[uint32]struct{})
	s.mu.Unlock()
	s.selectedObject = 0
	s.selectedOwner = 0
	s.openDropBox = nil
	s.lastMenuObject = 0
	s.lastMenuOwner = 0
	s.lastMenuAt = time.Time{}
	s.lastMenuMovie = false
	return nil
}

// registerNPCOnRun 供测试向视图当前运行时注册单个 NPC。
func (s *sceneLifecycle) registerNPCOnRun(id uint32, npc npcSpawn) error {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	return s.run.registerNPCLocked(id, npc)
}

// unregister 供测试移除单个 NPC。
func (s *sceneLifecycle) unregister(id uint32) error {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	return s.run.unregisterNPC(worldcore.EntityID(id))
}

// portalRoute 对传入的直接 Select 按活动视口进行校验，只返回有资源
// 支撑的 Door/GotoDoor 路线。它刻意排除 AdventureDoor 和剧情门，这些
// 门对应的副本/任务准入协议尚未还原。
func (s *sceneLifecycle) portalRoute(request clientObjectRequest) (scenePortalRoute, bool, error) {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return scenePortalRoute{}, false, nil
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	entity, exists := s.run.npcMeta(worldcore.EntityID(request.ObjectID))
	if !exists || entity.ownerID != request.OwnerID {
		return scenePortalRoute{}, false, fmt.Errorf("unknown scene object id=%d owner=%d", request.ObjectID, request.OwnerID)
	}
	if entity.portal == nil {
		return scenePortalRoute{}, false, nil
	}
	return *entity.portal, true, nil
}

// portalAt 按距离激活已安装的传送门，模拟正常的穿门行为。它刻意只使用
// X/Z：坡道和楼梯上的门的 creator/MoveTarget Y 值常有微小差异。一段
// 短暂的服务器侧冷却可防止同场景的本地端点在下一个移动 tick 立即再次
// 触发附近另一个传送门。
func (s *sceneLifecycle) portalAt(position role.Position, now time.Time) (scenePortalRoute, bool) {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return scenePortalRoute{}, false
	}
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	if s.run.closed || now.Before(s.run.portalCooldown) {
		return scenePortalRoute{}, false
	}
	portalFound := false
	var route scenePortalRoute
	s.run.eachNPCLocked(func(_ worldcore.EntityID, meta *sceneEntity, _ *worldcore.Transform) {
		if portalFound || meta.portal == nil {
			return
		}
		dx := position.X - meta.portal.trigger.X
		dz := position.Z - meta.portal.trigger.Z
		if dx*dx+dz*dz > meta.portal.triggerRadius*meta.portal.triggerRadius {
			return
		}
		route = *meta.portal
		portalFound = true
	})
	if portalFound {
		s.run.portalCooldown = now.Add(2 * time.Second)
		return route, true
	}
	return scenePortalRoute{}, false
}

// npcInteractionFor 把 ScriptClass 作为第一道门控。现代 city05 目录在
// CommonNpc 之外还包含 AttackNpc 和 BossNpc，因此即使客户端发送相同的
// 初始 Select 包，把它们全部当作通用对话对象也是错的。对平民/服务类
// NPC，只有当某项服务确实能在本地履约时才打开对话；否则仅保留选择
// 供查看/选靶，而不是凭空造出一个空的“交谈”菜单。
func npcInteractionFor(scriptClass string, services []npcService) npcInteraction {
	switch strings.ToLower(strings.TrimSpace(scriptClass)) {
	case "attacknpc", "bossnpc":
		return npcInteractionCombat
	}
	if len(playableNPCServices(services)) != 0 {
		return npcInteractionTalk
	}
	return npcInteractionSelectOnly
}

// clientReady 只为新一代布防，刻意不写入对象。当前客户端在 stage_main
// 调用 scene_obj:RegisterCallBack 之前就发送 ClientReady。若立即响应，
// 本机连接可能在这段小空窗内创建逻辑对象，却从不调用其视觉回调。
func (s *sceneLifecycle) clientReady() error {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return nil
	}
	s.armed = true
	log.Printf("%s: ClientReady; armed scene objects pending post-ready client activity", s.remote)
	return nil
}

// clientActivity 是当前客户端的就绪屏障。ClientReady 之后的第一条自定义
// 消息到达时，Lua 入口函数已经越过 scene_obj:RegisterCallBack。
func (s *sceneLifecycle) clientActivity() error {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed || !s.armed {
		return nil
	}
	log.Printf("%s: post-ready client activity; materializing scene objects", s.remote)
	if err := s.reconcileViewportLocked(); err != nil {
		return err
	}
	s.armed = false
	if s.wireID != 0 {
		s.run.activateMember(s.wireID)
	}
	return nil
}

func (s *sceneLifecycle) reconcileViewport() error {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	return s.reconcileViewportLocked()
}

func (s *sceneLifecycle) reconcileViewportLocked() error {
	s.mu.Lock()
	center := s.center
	radius := s.radius
	s.mu.Unlock()
	delta := s.viewport.Plan(s.run.scene.SnapshotWithin(center.X, center.Z, radius, worldcore.EntityID(s.wireID)))
	for _, update := range delta.Updates {
		entity, exists := s.npcMeta(update.Entity.ID)
		if !exists {
			return fmt.Errorf("missing property encoder for scene object %d", update.Entity.ID)
		}
		transform, transformOK := s.npcTransform(update.Entity.ID)
		if !transformOK {
			return fmt.Errorf("missing transform for scene object %d", update.Entity.ID)
		}
		// Transform 是目前巡逻循环唯一会改变的状态。在这里重放所有外观/
		// AI 字段会让当前客户端重建它的 Actor2。请使用紧凑的协商坐标更新；
		// sceneObjectProperties 仍留作未来真正变更时的编码器。
		properties := sceneObjectTransformProperties(entity.id, entity.ownerID, transform)
		if err := s.conn.WriteFrame(properties); err != nil {
			return fmt.Errorf("update scene object %d: %w", entity.id, err)
		}
		log.Printf("%s: updated scene object transform id=%d kind=%s", s.remote, entity.id, entity.description)
	}
	// 世代变更意味着客户端已经替换了原生场景/对象管理器。发送旧世代的
	// 移除消息可能命中新近复用的 ID，因此只在同一世代内发出 0x0E。
	if delta.PreviousGeneration == delta.Generation {
		for _, removal := range delta.Removes {
			payload := sceneRemoveObject(uint32(removal.Entity.ID), uint32(removal.Entity.OwnerID))
			if err := s.conn.WriteFrame(payload); err != nil {
				return fmt.Errorf("remove scene object %d: %w", removal.Entity.ID, err)
			}
		}
	}
	// 快照顺序是确定性的，但仍保留这个防御性排序，因为线上对象顺序是
	// 已采集兼容基线的一部分。
	sort.Slice(delta.Adds, func(i, j int) bool { return delta.Adds[i].Entity.ID < delta.Adds[j].Entity.ID })
	if len(delta.Adds) != 0 || len(delta.Removes) != 0 {
		log.Printf("%s: scene viewport delta adds=%d removes=%d", s.remote, len(delta.Adds), len(delta.Removes))
	}
	locationReplays := make([]sceneLocationReplay, 0, len(delta.Adds))
	for _, addition := range delta.Adds {
		id := addition.Entity.ID
		entity, exists := s.npcMeta(id)
		if !exists {
			return fmt.Errorf("missing payload encoder for scene object %d", id)
		}
		transform, transformOK := s.npcTransform(id)
		if !transformOK {
			return fmt.Errorf("missing transform for scene object %d", id)
		}
		// payload 在注册时已物化；防御性拷贝保持重进视口反复 Add 的
		// 语义与原先的闭包完全一致。巡逻 NPC 的 0x0D 落点重打为当前
		// 权威 Transform（巡逻 tick 每秒插值），从时间表当前位置续走。
		payload := append([]byte(nil), entity.payload...)
		if entity.schedule != nil {
			patchAddObjectTransform(payload, transform)
		}
		digest := sha256.Sum256(payload)
		log.Printf("%s: materialize scene object id=%d kind=%s opcode=0x%02X payload_len=%d payload_sha256=%x",
			s.remote, id, entity.description, payload[0], len(payload), digest)
		if err := s.conn.WriteFrame(payload); err != nil {
			return fmt.Errorf("add scene object %d: %w", id, err)
		}
		// 对象-主人绑定伴生帧（op 0x31 ServerLinkPair，2026-11-11 官服
		// 差集定版）：[op][u16 n][n×(obj u32, owner u32)]——官服与 0x28/
		// 0x0D 物化强相关交错（jn 179 条 vs 0x28 279 条；npc.pcapng n=4
		// 批量=0x0E 对象段×0x0A5 玩家 owner 段），是客户端实体-主人表的
		// 喂养通道，本服此前零下发——技能接近引擎的目标校验疑依赖此表
		//（50073 会话受理簇/0x28/选中链逐帧同构仍不自走的剩余差集之首）。
		if entity.ownerID != 0 {
			if err := s.conn.WriteFrame(serverLinkPairFrame(uint32(id), entity.ownerID)); err != nil {
				return fmt.Errorf("link pair %d: %w", id, err)
			}
		}
		// 0x28 详情记录（官服方言 NPC 渲染必需，sparse 二分定案）：
		// NINEYIN_NPC_DETAIL_0X28=1，物化（0x0D）之后下发——两代生成器
		//（=3 模板身份 / v2 本服身份+精确切片）在 0x0D 前下发均真机崩溃，
		// 身份策略迥异故共同点排除身份串、指向时序；官服 #1776→#1777
		// 先 0x28 配对疑为 op29 批量预载对象特例，新对象应先物化后详情
		//（=1 探针首测同教训 c1cd1b9）。
		if npcDetail0x28Enabled() {
			// v3（2026-11-11）：mode=0x46 明文方言程序化构造——0x47 LZ
			// 模板路线四连崩定案退役（根因=模板保真段的池回引 token 依赖
			// 客户端全会话字典状态，见 docs/op28-fxnet2-lz-decoder.md）；
			// 官方 l.pcapng 自有 6 帧 0x46 纯拷贝先例，任何会话状态安全。
			props := officialNPCEntityProperties(entity.properties, entity.model, entity.configID)
			// 走动物化 posB=段末目的地（npc.pcapng 官服 8/8 帧 A≠B）；
			// 停留/静立保持 A=B。
			var target *worldcore.Transform
			if entity.schedule != nil {
				if motion := entity.schedule.motionAt(time.Now().UnixMilli()); motion.moving {
					target = &worldcore.Transform{X: motion.destX, Y: motion.destY, Z: motion.destZ, Orient: motion.destOrient}
				}
			}
			if err := s.conn.WriteFrame(npcDetail0x28V3Frame(uint32(id), entity.ownerID, transform, target, props)); err != nil {
				return fmt.Errorf("npc 0x28 detail %d: %w", id, err)
			}
		}
		// 0x28 详情帧探针已退役（2026-11-10 真机定案）：=3 正帧版实证
		// 客户端消费该通道但记录块身份冲突即崩（模板 NPC 的动作/模型套
		// 在本服对象上）；寻路假设亦证伪（不发 0x28 时寻路正常）。0x28
		// 只走 P3 按对象生成器（见 npc_detail_frame.go 注释）。
		if entity.ownerID != 0 {
			location := serverLocation(entity.id, entity.ownerID, transform)
			if err := s.conn.WriteFrame(location); err != nil {
				return fmt.Errorf("locate scene object %d: %w", entity.id, err)
			}
			locationReplays = append(locationReplays, sceneLocationReplay{
				id: uint32(id), payload: append([]byte(nil), location...),
			})
		}
		// 重放的 0x0D 是注册期快照（满血活怪）。若该 NPC 当前是尸体
		//（被杀后等待重生），紧补一帧 0x10 死亡属性，让迟到的客户端
		// 立即转入死亡表现，而不是渲染一头无法选中的"活怪"。
		if vitals, hasVitals := s.run.npcVitals(id); hasVitals {
			if state := vitals.ActorState(entity.id, entity.ownerID); state.HP <= 0 {
				deadFrame, err := sceneObjectProperties(entity.id, entity.ownerID, 0, npcCombatProperties(state))
				if err != nil {
					return fmt.Errorf("encode dead scene object %d: %w", entity.id, err)
				}
				if err := s.conn.WriteFrame(deadFrame); err != nil {
					return fmt.Errorf("correct dead scene object %d: %w", entity.id, err)
				}
				// 尸体带未拾掉落（挂尸形态）：补发 CanPick/PickMember 冒光
				// 属性，迟到者同样看到尸体发光与拾取光标。
				if entity.lootBox != nil && !entity.lootBox.standalone {
					canPick := sceneObjectByteProperty(entity.id, entity.ownerID, propCanPick, 1)
					pickMember := sceneObjectWideStringProperty(entity.id, entity.ownerID, propPickMember, entity.lootBox.picker)
					for _, frame := range [][]byte{canPick, pickMember} {
						if err := s.conn.WriteFrame(frame); err != nil {
							return fmt.Errorf("send corpse pick property %d: %w", entity.id, err)
						}
					}
				}
			}
		}
		log.Printf("%s: sent modern scene object id=%d via ServerAddObject", s.remote, id)
	}
	if err := s.viewport.Commit(delta); err != nil {
		return err
	}
	if len(locationReplays) != 0 {
		s.scheduleLocationReplaysLocked(locationReplays)
	}
	return nil
}

// objectRequest 接受 FuncID==0 的 C2S 0x07（Select/对象点击）。它先设置
// LastObject，再用当前客户端的数字自定义消息路由打开面对面 NPC 对话。
// 创建业务 View 的是被选中的对话条目，而不是最初的 NPC 点击。
func (s *sceneLifecycle) objectRequest(request clientObjectRequest) (string, error) {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	if s.closed {
		return "", nil
	}
	entity, exists := s.npcMeta(worldcore.EntityID(request.ObjectID))
	if !exists || entity.ownerID != request.OwnerID {
		return "", fmt.Errorf("unknown scene object id=%d owner=%d", request.ObjectID, request.OwnerID)
	}
	if entity.id == s.lastMenuObject && entity.ownerID == s.lastMenuOwner && time.Since(s.lastMenuAt) < time.Second {
		log.Printf("%s: coalesced duplicate object request sequence=%d id=%d owner=%d", s.remote, request.Sequence, entity.id, entity.ownerID)
		return entity.description, nil
	}
	// 选靶查询应答补 0x28 目标详情（技能.pcapng 30.282→30.456 实证：
	// 客户端 op 0x07 查询后官服回 f#0187 mode=0x47 详情帧——客户端接近
	// AI 读目标详情（CollideRadius/位姿等）计算走近停靠，缺此帧即 431
	// 连发重查且冲锋受理后不自走；物化时发过的详情会过期，按需重发，
	// 位置取当前权威 Transform，序=详情先于 LastObject 对）。
	if npcDetail0x28Enabled() {
		if transform, transformOK := s.npcTransform(worldcore.EntityID(entity.id)); transformOK {
			props := officialNPCEntityProperties(entity.properties, entity.model, entity.configID)
			var detailTarget *worldcore.Transform
			if entity.schedule != nil {
				if motion := entity.schedule.motionAt(time.Now().UnixMilli()); motion.moving {
					detailTarget = &worldcore.Transform{X: motion.destX, Y: motion.destY, Z: motion.destZ, Orient: motion.destOrient}
				}
			}
			if err := s.conn.WriteFrame(npcDetail0x28V3Frame(uint32(entity.id), entity.ownerID, transform, detailTarget, props)); err != nil {
				return "", fmt.Errorf("npc 0x28 detail on select %d: %w", entity.id, err)
			}
		}
	}
	// 0x07 点击与 msg431 选目标共用同一条回帧路径：
	// LastObject(100) + AutoSelectObject(230) 双属性帧（49840 抓包
	// 证实客户端点击后服务端回 414/415 两帧）。
	if err := s.writePlayerTargetLocked(entity.id, entity.ownerID); err != nil {
		return "", fmt.Errorf("set player LastObject=%d-%d: %w", entity.id, entity.ownerID, err)
	}
	// LastObject 是对象选择属性，不是对话属性。它的服务器侧镜像要与
	// lastMenu* 分开保存：攻击目标从不打开对话窗口，但 211 战斗处理仍需
	// 解析同一个被选中的敌对角色。
	s.selectedObject = entity.id
	s.selectedOwner = entity.ownerID
	// 掉落宝箱：点击即开掉落箱窗口（View 80），不走对话/选靶语义。
	if entity.lootBox != nil {
		if err := s.openDropBoxLocked(entity); err != nil {
			return "", err
		}
		return entity.description, nil
	}
	// AttackNpc/BossNpc 以及不支持的场景对象仍然会成为玩家的 LastObject。
	// 那是战斗入口使用的目标选择，但这些对象绝不能收到以前对每个出生
	// 对象都发送的
	// 通用剧情对话。
	if entity.interaction != npcInteractionTalk {
		log.Printf("%s: object request sequence=%d id=%d config=%s script_class=%s interaction=%s; selected without dialogue",
			s.remote, request.Sequence, entity.id, entity.configID, entity.scriptClass, entity.interaction)
		return entity.description, nil
	}
	// 对现代客户端而言，creator/模板配置才是权威。遗留的 Npc_Func 表仅
	// 保留用于离线审计：它的成都绑定与当前 Creator ConfigID 不匹配，将其
	// 暴露为可用菜单项会产生
	// 本服务器无法路由的动作。
	items := talkMenuItems(playableNPCServices(entity.services), nil)
	frames, err := encodeCinematicTalkMenu(entity.id, entity.ownerID, "", items)
	if err != nil {
		return "", fmt.Errorf("encode cinematic menu for %s: %w", sceneIdent(entity.id, entity.ownerID), err)
	}
	for _, frame := range frames {
		if err := s.conn.WriteFrame(frame); err != nil {
			return "", fmt.Errorf("send cinematic menu for %s: %w", sceneIdent(entity.id, entity.ownerID), err)
		}
	}
	s.lastMenuObject = entity.id
	s.lastMenuOwner = entity.ownerID
	s.lastMenuAt = time.Now()
	s.lastMenuMovie = true
	log.Printf("%s: object request sequence=%d id=%d config=%s script_class=%s interaction=%s services=%v funcs=%v; sent numeric cinematic talk frames=%d ident=%s",
		s.remote, request.Sequence, entity.id, entity.configID, entity.scriptClass, entity.interaction, entity.services, npcFuncIDs(entity.funcs), len(frames), sceneIdent(entity.id, entity.ownerID))
	return entity.description, nil
}

// openDepotLocked 启动正式服的仓库流程。当前 Lua 没有 Lua 层的“打开
// 仓库”接收器：form_depot.on_server_msg 只会刷新已存在的表单。因此其
// 表单由客户端对 VIEWPORT_DEPOT (4) 的 SERVER_CREATE_VIEW 原生处理来
// 打开。打开后 main_form_open 会发出 C2S CLIENT_CUSTOMMSG_DEPOT_MSG
// （307，子消息 4），调用方的常规帧日志会为下一步服务器处理记录它。
func (s *sceneLifecycle) openDepotLocked() error {
	const (
		viewportDepot = 4
		depotCapacity = 18 // 当前 share/Rule/depotbox.ini 的 StartCap
	)
	if err := s.conn.WriteFrame(serverCreateView(serverViewSpec{ID: viewportDepot, Capacity: depotCapacity})); err != nil {
		return fmt.Errorf("create depot view: %w", err)
	}
	return nil
}

// openShopLocked 启动原生商店视口流程。form_shop.lua 从 View=61 的
// createview 回调获取下面全部四项设置；它自己不加载 NPC 模板。因此
// 目录在服务器侧解析，每条真实物品行都物化为一个 ViewAdd 对象。
func (s *sceneLifecycle) openShopLocked(shopID string) error {
	const (
		viewportShop = 61
		shopPageSize = 10
		shopCapacity = 100
		maxBuyAmount = 99
	)

	// 属性索引按名解析（0x09 表方言门控换表自动跟随；fork 位见各行注记）。
	var (
		propShopID      = clientdata.MustPlayerProperty("ShopID").Index     // ShopID（fork=2219）
		propPageNum     = clientdata.MustPlayerProperty("PageNum").Index    // PageNum（fork=2220）
		propPageCount   = clientdata.MustPlayerProperty("PageCount").Index  // PageCount（fork=2221）
		propShopType    = clientdata.MustPlayerProperty("ShopType").Index   // ShopType（fork=2222）
		propConfigID    = clientdata.MustPlayerProperty("ConfigID").Index   // ConfigID（fork=1440）
		propAmount      = clientdata.MustPlayerProperty("Amount").Index     // Amount（fork=1894）
		propPriceGold   = clientdata.MustPlayerProperty("SellPrice0").Index // SellPrice0（fork=1900）
		propPriceSilver = clientdata.MustPlayerProperty("SellPrice1").Index // SellPrice1（fork=1901）
		propPriceCard   = clientdata.MustPlayerProperty("SellPrice2").Index // SellPrice2（fork=1902）
		propMaxAmount   = clientdata.MustPlayerProperty("MaxAmount").Index  // MaxAmount（fork=1895）
	)
	items, shopType, pageCount, err := loadShopCatalogSection(defaultShopINIPath, shopID)
	if err != nil {
		return err
	}
	frame, err := serverCreateViewWithProperties(serverViewSpec{ID: viewportShop, Capacity: shopCapacity}, []serverViewProperty{
		viewString(propShopID, shopID),
		viewInt(propPageNum, shopPageSize),
		viewInt(propPageCount, pageCount),
		viewInt(propShopType, shopType),
	})
	if err != nil {
		return fmt.Errorf("encode shop view %q: %w", shopID, err)
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("create shop view %q: %w", shopID, err)
	}
	for _, item := range items {
		objectIndex := (item.page-1)*shopPageSize + item.position + 1
		properties := []serverViewProperty{
			viewString(propConfigID, item.configID),
			viewInt(propAmount, item.amount),
			viewInt(propPriceGold, 0),
			viewInt(propPriceSilver, 0),
			viewInt(propPriceCard, 0),
			viewInt(propMaxAmount, maxBuyAmount),
		}
		switch item.capitalType {
		case 0:
			properties[2] = viewInt(propPriceGold, item.price)
		case 1:
			properties[3] = viewInt(propPriceSilver, item.price)
		case 2:
			properties[4] = viewInt(propPriceCard, item.price)
		case 3:
			// 兑换类（shop.ini 约 88% 条目）：三价全 0 物化——成本由
			// msg70 的 exchangeitem.ini 分支校验（fork 9/14 openShop
			// 同样支持 capitalType 0..3；此前这里报错会让客户端拿到
			// 半截商店）。View61 行显示为兑换（Java Shop_menpai_xtc_01
			// 同款零价形态，客户端兼容已验证）。
		default:
			return fmt.Errorf("shop %q item %q has unsupported capital type %d", shopID, item.configID, item.capitalType)
		}
		itemFrame, encodeErr := serverViewAdd(viewportShop, uint16(objectIndex), properties)
		if encodeErr != nil {
			return fmt.Errorf("encode shop %q item %q: %w", shopID, item.configID, encodeErr)
		}
		if writeErr := s.conn.WriteFrame(itemFrame); writeErr != nil {
			return fmt.Errorf("add shop %q item %q: %w", shopID, item.configID, writeErr)
		}
	}
	return nil
}

type talkMenuItem struct {
	funcID int32
	textID string
}

// sceneIdent 是 GetSceneObj / Select / 0x1C 头部使用的客户端对象键。
func sceneIdent(objectID, ownerID uint32) string {
	return fmt.Sprintf("%d-%d", objectID, ownerID)
}

// sendTalkMenuLocked 为现代 open_npc_talk 发送原生 0x1C SERVER_MENU。
func (s *sceneLifecycle) sendTalkMenuLocked(objectID, ownerID uint32, items []serverMenuItem) error {
	frame, err := serverMenu(objectID, ownerID, items)
	if err != nil {
		return fmt.Errorf("encode 0x1C SERVER_MENU for %s: %w", sceneIdent(objectID, ownerID), err)
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("send 0x1C SERVER_MENU for %s: %w", sceneIdent(objectID, ownerID), err)
	}
	return nil
}

// encodeCinematicTalkMenu 使用当前 C++ GameMessageHandler 的自定义接收器
// 表。该表从 custom ID 1（custom_sysinfo）开始；因此其菜单条目是 4..8，
// 而不是早期实验错误推断的原始对象槽位偏移 49..53。该客户端不接受字符
// 串别名，所以第一个 TVarList 值必须使用数字。
func encodeCinematicTalkMenu(objectID, ownerID uint32, titleID string, items []talkMenuItem) ([][]byte, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("talk menu has no items")
	}
	out := make([][]byte, 0, 3+len(items))

	begin, err := serverCustomIntMessage(4)
	if err != nil {
		return nil, err
	}
	out = append(out, begin)

	if titleID != "" {
		title, err := serverCustomIntMessage(5, customString(titleID))
		if err != nil {
			return nil, err
		}
		out = append(out, title)
	}

	for _, item := range items {
		pkt, err := serverCustomIntMessage(6, customInt(item.funcID), customString(item.textID))
		if err != nil {
			return nil, err
		}
		out = append(out, pkt)
	}

	end, err := serverCustomIntMessage(7, customObject(objectID, ownerID))
	if err != nil {
		return nil, err
	}
	out = append(out, end)
	return out, nil
}

// selectNPCMenu 处理第二条 C2S Select：由先前发送的原生菜单产出的非零
// FuncID/mark。它只在该 NPC 已配置的现代服务内解析 mark，清除菜单，然后
// 打开请求的业务 View。这与正式服的两段式所有权边界一致：点击 NPC ->
// SERVER_MENU -> 选择 mark -> 业务协议。
func (s *sceneLifecycle) selectNPCMenu(request clientObjectRequest) error {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	entity, exists := s.npcMeta(worldcore.EntityID(request.ObjectID))
	if !exists || entity.ownerID != request.OwnerID {
		return fmt.Errorf("select unknown scene object id=%d owner=%d", request.ObjectID, request.OwnerID)
	}
	if entity.id != s.lastMenuObject || entity.ownerID != s.lastMenuOwner {
		return fmt.Errorf("select menu mark=%d without active menu for id=%d-%d", request.FuncID, entity.id, entity.ownerID)
	}
	if s.lastMenuMovie {
		return s.selectCinematicTalkLocked(entity, request)
	}
	if request.FuncID < 0 || request.FuncID > int32(^uint16(0)) {
		return fmt.Errorf("invalid menu mark=%d for id=%d-%d", request.FuncID, entity.id, entity.ownerID)
	}
	mark := uint16(request.FuncID)
	if mark == 0 {
		if err := s.clearMenuLocked(); err != nil {
			return err
		}
		s.clearActiveMenuLocked()
		log.Printf("%s: NPC menu dismissed id=%d config=%s", s.remote, entity.id, entity.configID)
		return nil
	}
	service, ok := findNPCService(entity.services, mark)
	if !ok {
		return fmt.Errorf("NPC %s has no implemented modern service for mark=%d", entity.configID, mark)
	}
	if err := s.clearMenuLocked(); err != nil {
		return err
	}
	s.clearActiveMenuLocked()
	if err := s.openSelectedServiceLocked(entity, service); err != nil {
		return err
	}
	log.Printf("%s: NPC menu selection id=%d config=%s mark=%d source=%s completed", s.remote, entity.id, entity.configID, mark, service.source)
	return nil
}

func (s *sceneLifecycle) selectCinematicTalkLocked(entity sceneEntity, request clientObjectRequest) error {
	if request.FuncID == 600000000 { // g_menu_id.leave
		if err := s.closeCinematicTalkLocked(); err != nil {
			return err
		}
		s.clearActiveMenuLocked()
		return nil
	}
	mark, ok := cinematicTalkMark(request.FuncID)
	if !ok {
		return fmt.Errorf("NPC %s has no cinematic service for func_id=%d", entity.configID, request.FuncID)
	}
	service, ok := findNPCService(entity.services, mark)
	if !ok {
		return fmt.Errorf("NPC %s does not advertise cinematic service mark=%d", entity.configID, mark)
	}
	if err := s.closeCinematicTalkLocked(); err != nil {
		return err
	}
	s.clearActiveMenuLocked()
	if err := s.openSelectedServiceLocked(entity, service); err != nil {
		return err
	}
	log.Printf("%s: cinematic NPC menu selection id=%d config=%s func_id=%d source=%s completed", s.remote, entity.id, entity.configID, request.FuncID, service.source)
	return nil
}

func cinematicTalkMark(funcID int32) (uint16, bool) {
	switch funcID {
	case 805000000:
		return markShop, true
	case 807000000:
		return markDepot, true
	default:
		return 0, false
	}
}

func (s *sceneLifecycle) openSelectedServiceLocked(entity sceneEntity, service npcService) error {
	switch service.mark {
	case markShop:
		if service.value == "" {
			return fmt.Errorf("NPC %s shop service has no ShopID", entity.configID)
		}
		if err := s.openShopLocked(service.value); err != nil {
			return err
		}
	case markDepot:
		if err := s.openDepotLocked(); err != nil {
			return err
		}
	default:
		// 菜单对每项现有服务都保持数据驱动。原生实现缺席时先交给
		// Lua npc_service 处理器；脚本也没有接手才维持"未实现"。
		if s.runNpcServiceScriptLocked(entity, service) {
			return nil
		}
		// 在传送/任务/职业各自的现代业务包和持久化数据得到验证之前，
		// 不要凭空发明客户端动作。
		return fmt.Errorf("NPC %s service %q (mark=%d) is discovered but not yet implemented", entity.configID, service.source, service.mark)
	}
	return nil
}

// luaEventNpcService 是 NPC 服务脚本事件名：原生实现缺席的 NPC 菜单
// 服务（markTask 等）会以该事件调用 scripts/ 目录下注册的处理器。
const luaEventNpcService = "npc_service"

// runNpcServiceScriptLocked 把一次 NPC 服务选择交给 Lua npc_service
// 处理器。返回脚本是否接手：处理器返回 false（显式拒绝）或没有注册
// 处理器时返回 false，调用方走默认的"未实现"路径。处理器出错同样
// 视为已接手——菜单已被清除，错误已记录日志，不应再向客户端暴露
// 协议层错误。
//
// 调用方持有 run.mu；脚本侧副作用（notify/bubble）在本锁下直接
// 写帧，与 openShopLocked 的写帧行为一致。
func (s *sceneLifecycle) runNpcServiceScriptLocked(entity sceneEntity, service npcService) bool {
	if s.scripts == nil || !s.scripts.HasHandler(luaEventNpcService) {
		return false
	}
	ec := &luascript.EventContext{
		NPC: luascript.NPCSnapshot{
			ID:       entity.id,
			ConfigID: entity.configID,
			Name:     entity.description,
		},
		Service: luascript.ServiceSnapshot{
			Mark:   service.mark,
			Label:  service.label,
			Value:  service.value,
			Source: service.source,
		},
	}
	// 该 NPC 绑定且有文本的任务（纯文本），供脚本列出任务用。
	for _, task := range taskCatalog.TasksForNPC(entity.configID) {
		ec.Tasks = append(ec.Tasks, luascript.TaskSnapshot{
			ID:            task.ID,
			Title:         taskdata.StripRichText(task.Title),
			Target:        taskdata.StripRichText(task.Target),
			CanAcceptMenu: taskdata.StripRichText(task.CanAcceptMenu),
		})
	}
	if member := s.run.members[s.wireID]; member != nil {
		ec.Player.ID = member.objectID
		ec.Player.Scene = s.run.key.config
		ec.Player.X, ec.Player.Y, ec.Player.Z = float64(member.position.X), float64(member.position.Y), float64(member.position.Z)
		if member.player != nil {
			// 锁序 run.mu -> player.mu 与全仓纪律一致。
			member.player.mu.Lock()
			ec.Player.Name = member.player.name
			ec.Player.Faction = member.player.faction
			ec.Player.Silver = int64(member.player.silver)
			ec.Player.SP = int64(member.player.progress.attributes.sp)
			member.player.mu.Unlock()
		}
	}
	ec.Notify = func(brief, chapterName string) error {
		frame, err := encodeDramaPrompt(dramaPrompt{Chapter: 1, ChapterName: chapterName, Explore: 1, Challenge: 1, Brief: brief})
		if err != nil {
			return err
		}
		if err := s.conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("send drama prompt: %w", err)
		}
		return nil
	}
	ec.Bubble = func(textID string) error {
		frame, err := encodeNPCBubble(entity.id, entity.ownerID, textID)
		if err != nil {
			return err
		}
		if err := s.conn.WriteFrame(frame); err != nil {
			return fmt.Errorf("send NPC bubble: %w", err)
		}
		return nil
	}
	handled, err := s.scripts.Call(luaEventNpcService, ec)
	if err != nil {
		log.Printf("%s: lua npc_service script failed npc=%s mark=%d: %v", s.remote, entity.configID, service.mark, err)
		return true
	}
	if !handled {
		return false
	}
	log.Printf("%s: lua npc_service handled npc=%d config=%s mark=%d source=%s", s.remote, entity.id, entity.configID, service.mark, service.source)
	return true
}

func (s *sceneLifecycle) clearMenuLocked() error {
	if err := s.conn.WriteFrame([]byte{0x1D}); err != nil {
		return fmt.Errorf("send SERVER_CLEAR_MENU: %w", err)
	}
	return nil
}

func (s *sceneLifecycle) clearActiveMenuLocked() {
	s.lastMenuObject = 0
	s.lastMenuOwner = 0
	s.lastMenuAt = time.Time{}
	s.lastMenuMovie = false
}

func (s *sceneLifecycle) closeCinematicTalkLocked() error {
	frame, err := serverCustomIntMessage(8)
	if err != nil {
		return fmt.Errorf("encode cinematic closemenu: %w", err)
	}
	if err := s.conn.WriteFrame(frame); err != nil {
		return fmt.Errorf("send cinematic closemenu: %w", err)
	}
	return nil
}

func findNPCService(services []npcService, mark uint16) (npcService, bool) {
	for _, service := range services {
		if service.mark == mark {
			return service, true
		}
	}
	return npcService{}, false
}

// npcServiceMenu 刻意只从所选 NPC 的 Npc_Func 行推导。旧的那种通用
// “与我交谈 / 离开”组合让仓库、船夫、训练师和普通市民变得无法
// 区分。
func npcServiceMenu(services []npcService, bindings []npcfunc.Binding) []serverMenuItem {
	if len(services) != 0 {
		items := make([]serverMenuItem, 0, len(services)+1)
		for _, service := range services {
			items = append(items, serverMenuItem{Type: 0, Mark: service.mark, Content: service.label})
		}
		return append(items, serverMenuItem{Type: 0, Mark: 0, Content: "离开"})
	}
	seen := make(map[int]struct{}, len(bindings))
	items := make([]serverMenuItem, 0, len(bindings)+1)
	for _, binding := range bindings {
		if _, duplicate := seen[binding.FuncID]; duplicate {
			continue
		}
		seen[binding.FuncID] = struct{}{}
		items = append(items, serverMenuItem{Type: 0, Mark: uint16(binding.FuncID), Content: npcFuncMenuText(binding.FuncID)})
	}
	if len(items) == 0 {
		items = append(items, serverMenuItem{Type: 0, Mark: 1, Content: "此 NPC 暂无 Npc_Func 配置"})
	}
	items = append(items, serverMenuItem{Type: 0, Mark: 0, Content: "离开"})
	return items
}

func talkMenuItems(services []npcService, bindings []npcfunc.Binding) []talkMenuItem {
	items := make([]talkMenuItem, 0, len(services)+len(bindings)+1)
	for _, service := range services {
		items = append(items, talkMenuItem{funcID: modernTalkFuncID(service.mark), textID: modernTalkTextID(service.mark)})
	}
	if len(items) == 0 {
		seen := make(map[int]struct{}, len(bindings))
		for _, binding := range bindings {
			if _, duplicate := seen[binding.FuncID]; duplicate {
				continue
			}
			seen[binding.FuncID] = struct{}{}
			items = append(items, talkMenuItem{funcID: legacyTalkFuncID(binding.FuncID), textID: legacyTalkTextID(binding.FuncID)})
		}
	}
	if len(items) == 0 {
		items = append(items, talkMenuItem{funcID: 860000001, textID: "ui_talk"})
	}
	return append(items, talkMenuItem{funcID: 600000000, textID: "ui_form_talk_leave"})
}

func modernTalkFuncID(mark uint16) int32 {
	switch mark {
	case 0x1001:
		return 805000000 // 商店
	case 0x1002:
		return 807000000 // 仓库
	case 0x1003:
		return 860000022 // 传送
	case 0x1004:
		return 806000036 // 生活职业
	case 0x1005:
		return 806000013 // 宝石服务
	case 0x1006:
		return 809000000 // 帮会
	case 0x1007:
		return 881000001 // 回城点
	case 0x1008:
		return 100000001 // 任务入口占位，待任务状态持久化后替换
	default:
		return 860000000 + int32(mark)
	}
}

func modernTalkTextID(mark uint16) string {
	switch mark {
	case 0x1001:
		return "ui_shop"
	case 0x1002:
		return "ui_depot"
	case 0x1003:
		return "ui_transport"
	case 0x1004:
		return "ui_life_job"
	case 0x1005:
		return "ui_equip_gem"
	case 0x1006:
		return "ui_guild"
	case 0x1007:
		return "ui_home_point"
	case 0x1008:
		return "ui_form_talk_receive_task"
	default:
		return "ui_talk"
	}
}

func legacyTalkFuncID(funcID int) int32 {
	switch funcID {
	case 9, 17, 18:
		return 805000000 + int32(funcID)
	case 20:
		return 807000000
	case 24:
		return 809000000
	case 36:
		return 806000036
	case 48:
		return 881000001
	default:
		return 860000000 + int32(funcID)
	}
}

func legacyTalkTextID(funcID int) string {
	switch funcID {
	case 9:
		return "ui_shop"
	case 20:
		return "ui_depot"
	case 22:
		return "ui_transport"
	case 24:
		return "ui_guild"
	case 36:
		return "ui_life_job"
	case 47:
		return "ui_skill_trainer"
	case 48:
		return "ui_home_point"
	default:
		return "ui_talk"
	}
}

// modernNPCServices 从解析后的现代 NPC 配置中提取业务标识。这些值保留
// 在 Extensions 中，因为它们是服务器业务配置，不是可见的 0x09 属性。
// creator 值优先于模板对应值：creator 是实例级覆盖，而模板是共享默认值。
// 这些 mark 刻意占用与旧 FuncID 值不同的独立区间。
func modernNPCServices(npc npcSpawn, tasks *taskdata.Catalog) []npcService {
	value := func(key string) (string, string) { return effectiveNPCBusinessValue(npc, key) }
	add := func(items []npcService, mark uint16, label, key string) []npcService {
		if configured, source := value(key); configured != "" && configured != "0" {
			return append(items, npcService{mark: mark, label: label, value: configured, source: source})
		}
		return items
	}
	items := make([]npcService, 0, 6)
	items = add(items, markShop, "商店", "ShopID")
	items = add(items, markDepot, "仓库", "DepotID")
	items = add(items, markTransport, "传送", "TransFuncID")
	items = add(items, markGem, "宝石服务", "GemGameNpcID")
	items = add(items, markGuild, "帮会", "GuildFunc")
	items = add(items, markHome, "回城点", "HomePoint")
	if job := npc.int32Property("Job"); job != 0 {
		items = append(items, npcService{mark: markJob, label: "生活职业", value: fmt.Sprintf("%d", job), source: "Job"})
	}
	// 任务绑定来自任务目录（storytasklist/qianjiu 派生）。旧探测键
	// table@TaskCanAccept 在资源中并不存在，已由真实数据取代；value
	// 携带逗号连接的任务 ID，供审计与 Lua npc_service 钩子使用。
	if bindings := tasks.BindingsForNPC(npc.resolved.ConfigID); len(bindings) > 0 {
		ids := make([]string, 0, len(bindings))
		for _, binding := range bindings {
			ids = append(ids, strconv.Itoa(binding.TaskID))
		}
		items = append(items, npcService{
			mark:   markTask,
			label:  "任务",
			value:  strings.Join(ids, ","),
			source: "taskcatalog." + bindings[0].Source,
		})
	}
	return items
}

// playableNPCServices 刻意比 modernNPCServices 更窄。后者报告每一项已
// 配置的能力；本函数只暴露同时具备已验证 S2C 处理器和完整本地数据的
// 能力。待定的任务/传送/帮会配置保留在审计产物中可见，而不是变成一个
// 玩家点了之后才失败的按钮。
func playableNPCServices(services []npcService) []npcService {
	playable := make([]npcService, 0, len(services))
	for _, service := range services {
		switch service.mark {
		case markDepot:
			playable = append(playable, service)
		case markShop:
			if _, _, _, err := loadShopCatalogSection(defaultShopINIPath, service.value); err == nil {
				playable = append(playable, service)
			}
		}
	}
	return playable
}

// effectiveNPCBusinessValue 按 NPC 实例配置的同一优先级解析非可见配置
// 字段：先 creator 覆盖，后模板默认。空字符串不会抹掉已配置的默认值；
// 解包出的 creator XML 经常含有无关的可选空字段。
func effectiveNPCBusinessValue(npc npcSpawn, key string) (value, source string) {
	for _, prefix := range []string{"creator.", "template."} {
		if configured := strings.TrimSpace(npc.resolved.Extensions[prefix+key]); configured != "" {
			return configured, prefix + key
		}
	}
	return "", ""
}

func npcFuncMenuText(funcID int) string {
	labels := map[int]string{
		6: "江湖/PVP", 7: "交谈", 9: "商店", 10: "强化", 12: "合成", 13: "宝石合成", 14: "开孔", 17: "拍卖行", 18: "黑市", 19: "转职", 20: "仓库", 22: "传送", 23: "召唤载具", 24: "帮会", 32: "制造", 35: "物品绑定", 36: "生活职业", 40: "PVP 任务", 41: "PVP 任务", 43: "神秘人", 44: "向导", 45: "典狱官", 46: "监察员", 47: "技能训练", 48: "回城点", 49: "PVP 任务", 50: "载具管理", 51: "赠礼", 111: "个人机械管理",
	}
	if label, ok := labels[funcID]; ok {
		return label
	}
	return fmt.Sprintf("配置功能 #%d", funcID)
}

func npcFuncIDs(bindings []npcfunc.Binding) []int {
	ids := make([]int, 0, len(bindings))
	for _, binding := range bindings {
		ids = append(ids, binding.FuncID)
	}
	return ids
}

// npcVisible 报告该 NPC 是否在本视图视口内（帧扇出过滤）。
func (s *sceneLifecycle) npcVisible(id uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, visible := s.visibleNPCs[id]
	return visible
}

// scheduleLocationReplaysLocked 排布本视图的延迟 ServerLocation
// 重放。定时器归视图所有（只写自己的 conn），世界存在性检查在
// run.mu 下进行。调用方必须已持有 s.run.mu。
func (s *sceneLifecycle) scheduleLocationReplaysLocked(replays []sceneLocationReplay) {
	run := s.run
	for _, delay := range actor2LocationReplayDelays {
		delay := delay
		batch := make([]sceneLocationReplay, len(replays))
		for i, replay := range replays {
			batch[i] = sceneLocationReplay{id: replay.id, payload: append([]byte(nil), replay.payload...)}
		}
		timer := time.AfterFunc(delay, func() {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			run.mu.Lock()
			defer run.mu.Unlock()
			if run.closed {
				return
			}
			sent := 0
			for _, replay := range batch {
				if _, exists := run.npcMeta(worldcore.EntityID(replay.id)); !exists {
					continue
				}
				if err := s.conn.WriteFrame(replay.payload); err != nil {
					log.Printf("%s: delayed ServerLocation id=%d after %s: %v", s.remote, replay.id, delay, err)
					return
				}
				sent++
			}
			if sent != 0 {
				log.Printf("%s: replayed ServerLocation for %d scene objects after %s", s.remote, sent, delay)
			}
		})
		s.mu.Lock()
		s.replayTimers = append(s.replayTimers, timer)
		s.mu.Unlock()
	}
}

func sceneRemoveObject(objectID, ownerID uint32) []byte {
	payload := make([]byte, 9)
	payload[0] = 0x0E
	binary.LittleEndian.PutUint32(payload[1:], objectID)
	binary.LittleEndian.PutUint32(payload[5:], ownerID)
	return payload
}

func (s *sceneLifecycle) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.detachMember()
	s.stopReplayTimers()
}

// stopReplayTimers 取消本视图的延迟 ServerLocation 重放定时器。
// 场景级定时器（巡逻/战斗）归 run 所有，不在此列。
func (s *sceneLifecycle) stopReplayTimers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, timer := range s.replayTimers {
		timer.Stop()
	}
	s.replayTimers = nil
}

// npcMetaByID 供测试按数字 id 读取注册元数据。
func (s *sceneLifecycle) npcMetaByID(id uint32) (sceneEntity, bool) {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	return s.run.npcMeta(worldcore.EntityID(id))
}

// spawnTestNPC 供测试在持锁下注册 NPC 组件（不含可视权威 Scene 实体）。
func (s *sceneLifecycle) spawnTestNPC(id worldcore.EntityID, vitals sim.Vitals, transform worldcore.Transform, meta sceneEntity, brain *sim.Brain) {
	s.run.mu.Lock()
	defer s.run.mu.Unlock()
	s.run.spawnSceneEntityLocked(id, vitals, transform, meta, brain)
}
