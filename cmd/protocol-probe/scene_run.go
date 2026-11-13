package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/ecs"
	"github.com/local/9yin-go-server/internal/luascript"
	"github.com/local/9yin-go-server/internal/mapnav"
	"github.com/local/9yin-go-server/internal/npcfunc"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/sim"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// sceneRun 是一个场景的共享权威运行时：sim 世界、可视权威 Scene、
// 定时器与中央战斗 tick 都归它所有，多个连接以 sceneLifecycle
// 视图的形式挂靠同一份 NPC 模拟。本轮玩家之间互不可见；
// NPC 仇恨目标取"最近存活成员（粘性）"。
//
// 锁纪律：run.mu 即全局锁序中的 scene.mu（外层），p.mu 为内层。
// 视图自身的状态（armed/selected*/viewport 等）只被其属主连接
// goroutine 访问，不需要锁；run 的定时器与 tick goroutine 只触
// run 状态与各视图的 conn（transport 写帧自带互斥）。
type sceneRun struct {
	mu             sync.Mutex
	key            sceneKey
	combat         *sim.World
	scene          *worldcore.Scene
	members        map[uint32]*sceneMember
	epoch          uint64
	timers         []*time.Timer
	portalCooldown time.Time
	patrolIDs      []uint32 // 在巡 NPC 的场景 ID（populate 按目录序即 ID 序追加）
	closed         bool
	ticking        bool
	// managed：副本实例标记。置位后末成员离开不再自动 shutdown
	//（实例在打开期间常驻，供 reenter_clone 回来），销毁由
	// dungeonManager 显式执行。noRespawn：副本怪杀完不复活——
	// combatTick 的每怪复活分支与 refreshTick 的批量/重注册扫描
	// 一并跳过（尸体 TTL 清除照常）。
	managed   bool
	noRespawn bool
	// 波次域（dungeon_wave.go）：副本实例的 NpcProgress 逐波揭示。
	// waveMode 由 populate 时目录含波次数据开启；waveAlive 清空即
	// 波次结算点（Lua dungeon_wave_cleared 可接管推进节奏）。
	waveMode         bool
	currentWave      int                  // 当前在场波次值（0=无波次数据）
	waveGroups       map[int][]int        // 波次值 → 目录下标集
	pendingWaves     []int                // 尚未注册的波次值（升序）
	waveAlive        map[uint32]struct{}  // 当前波在场 NPC 的 wire ID
	waveClearedFired int                  // 已发过 wave_cleared 的波次值（防重发）
	dungeonKills     int                  // 副本内击杀累计
	dungeonDifficulty int                 // 难度档（attachRun 后由 GM 路径回填）
	finished         bool                 // 末波全灭（收尾已排程）
	viewsDirty       bool                 // 有新波物化，待 tick 外 reconcile 全体视口
	scripts          *luascript.Host      // 副本事件宿主（enter 时注入；nil=无脚本走默认引擎）
	finishHook       func()               // 收尾回调（dungeonManager.markFinished；epoch 防护后锁外执行）
	tickStop       chan struct{}
	tickDone       chan struct{}
	npcFuncs       *npcfunc.Registry
	nextWireID     uint32
	// 刷新域（npc_refresh.go）：目录保留供尸体清除后的重注册；
	// deadNPCs 登记等待批量/定时重生的尸体；lootBoxes 是在场掉落宝箱。
	catalog             []npcSpawn
	deadNPCs            map[uint32]deadNPCRecord
	lootBoxes           map[uint32]*lootBoxRuntime
	nextSweepAt         time.Time
	pendingObjectFrames []pendingObjectFrame
	// npcLastLocation 是 NPC 0x1F 位置广播的节流水位（追击期间客户端
	// 认知保鲜，combat_ai.throttledNPCLocationBroadcast 读写）。
	npcLastLocation map[uint32]npcLocationBroadcast
	// fightRelation 是 0x13 fight_relation_rec 单格累计伤害的权威表：
	// 「玩家对象 → 受害 NPC」的逐击累加（抓包 143→210→277 每击 +67）。
	// NPC 死亡清除，重生后从零累计。run.mu 保护。
	fightRelation map[uint32]int32
	// npcMoveState 是 NPC 行走状态水位（patrolTick 读写）：id → 最近一次
	// 下发的 moving 标志。检出翻转时发 0x10 State(183)="walk"/"stand"
	// （官服 NPC 行为通道：起走发 walk、驻停发 stand，与 0x21 路点帧
	// 成对出现；docs/官服登录帧对照与缺口清单.md §13.5）。实体注销时
	// 连带清除。run.mu 保护。
	npcMoveState map[uint32]bool
	// 路点图导航域（mapnav_chase.go）：nav/navPlanner/navLoaded 由
	// warmNav（sync.Once，创建时后台预热）在 run.mu 内写入；追击分支
	// 在 run.mu 内经 navGraphLocked() 取用，未就绪本拍走直线兜底。
	nav        *mapnav.Graph
	navPlanner *mapnav.Planner
	navLoaded  bool
	navOnce    sync.Once
}

// sceneMember 是挂靠场景的一个连接成员。position 是场景侧的
// 权威位置（由 0x0A/传送路径更新），active 在就绪屏障
// （clientActivity）之后置位，NPC 仇恨只考虑 active 成员。
//
// objectID 是该玩家在场景内的 wire 身份（其他客户端看到的远程
// 对象）；selfID 恒为 1——本机客户端自己的对象必须是 1-1
// （stage_main 开图与 0x11 记录绑定）。带 player 的成员会被
// 实体化进可视权威 Scene（Kind=EntityPlayer，两组件
// Transform+sceneEntity，不带 Vitals——因此被所有 NPC 遍历与
// PvP 选靶闸门天然排除）。
type sceneMember struct {
	objectID uint32 // 场景内 wire objectID（其他客户端可见）
	selfID   uint32 // 本机 1-1 身份
	player   *playerActor
	view     *sceneLifecycle
	position role.Position
	active   bool
	visual   roleVisual
	// patrolReadyAt：clientActivity 后的 0x1F 重放窗口（250ms..5s）内
	// 不向该成员发巡逻 0x21，防重放帧把走动 NPC 拽回物化位置。
	patrolReadyAt time.Time
}

// sceneKey 是场景运行时的注册表键。instance 区分共享世界场景与副本
// 实例：空 = 共享世界场景（同 config 全服一份运行时），非空 = 副本
// 实例 ID（同 config 可并存多个独立运行时，由 dungeonManager 生命周期
// 托管）。
type sceneKey struct {
	config, resource string
	instance         string
}

var (
	sceneRegistryMu sync.Mutex
	sceneRegistry   = map[sceneKey]*sceneRun{}
)

// acquireSceneRun 返回 (config, resource) 对应的共享场景运行时，
// 首次进入时按目录一次性注册全部合格 NPC。目录排除 EventNpc：
// 它们没有任务归属者，绝不应进入共享世界。
func acquireSceneRun(key sceneKey, catalog []npcSpawn, funcs *npcfunc.Registry) (*sceneRun, error) {
	sceneRegistryMu.Lock()
	run, exists := sceneRegistry[key]
	if !exists {
		run = newSceneRun(key, funcs)
		sceneRegistry[key] = run
	}
	sceneRegistryMu.Unlock()
	if !exists {
		if err := run.populate(catalog); err != nil {
			sceneRegistryMu.Lock()
			delete(sceneRegistry, key)
			sceneRegistryMu.Unlock()
			run.mu.Lock()
			run.closed = true
			run.mu.Unlock()
			return nil, err
		}
		log.Printf("scene run created config=%s resource=%s npcs=%d", key.config, key.resource, run.combat.CombatCount())
	}
	return run, nil
}

func newSceneRun(key sceneKey, funcs *npcfunc.Registry) *sceneRun {
	run := &sceneRun{
		key:       key,
		combat:    sim.New(),
		scene:     worldcore.NewScene(),
		members:   make(map[uint32]*sceneMember),
		npcFuncs:  funcs,
		deadNPCs:  make(map[uint32]deadNPCRecord),
		lootBoxes: make(map[uint32]*lootBoxRuntime),
		// fightRelation 惰性建表即可，但预分配免去首击路径的判空。
		fightRelation: make(map[uint32]int32),
	}
	// 后台预热路点图（scene14 解析 ~百毫秒级），避免首个追击 tick
	// 在 run.mu 内同步付这笔钱。
	go run.warmNav()
	return run
}

// populate 一次性注册全量目录。实体 ID = 目录下标 + 2，
// 与单连接时代的分配规则保持一致（0/1 保留给玩家）。
// 副本实例（key.instance 非空）且目录带 NpcProgress 波次数据时走
// 逐波揭示：首波之外的全部留待波次推进注册（dungeon_wave.go）。
func (r *sceneRun) populate(catalog []npcSpawn) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.scene.Begin(r.key.config, r.key.resource); err != nil {
		return err
	}
	registered := 0
	// 目录保留在运行时上：尸体 TTL 清除后的重生需要按原下标重注册。
	r.catalog = catalog
	if r.key.instance != "" {
		r.initWaveGroupsLocked(catalog)
	}
	for index, npc := range catalog {
		if !npcSpawnEligible(npc) {
			continue
		}
		if r.waveMode {
			if wave, hasWave := npcWaveValue(npc); hasWave && wave != r.currentWave {
				continue
			}
		}
		id := (uint32(index) + 2) | 0x11000000
		if err := r.registerNPCLocked(id, npc); err != nil {
			return fmt.Errorf("register NPC %s: %w", npc.resolved.ConfigID, err)
		}
		if r.waveMode {
			if _, hasWave := npcWaveValue(npc); hasWave {
				r.trackWaveNPCLocked(id)
			}
		}
		registered++
	}
	// NPC 占用 0x11000002..0x11000000|len(catalog)+1（fork 版 ID 方案，
	// 与玩家共用 0x11000000 类型位前缀）；远程玩家 wireID 仍从无前缀的
	// 低位空间自增，与 NPC 的前缀空间和玩家保留 ID 0x11000001 永不碰撞。
	r.nextWireID = uint32(len(catalog)) + 2
	log.Printf("scene run populated config=%s resource=%s registered=%d skipped=%d", r.key.config, r.key.resource, registered, len(catalog)-registered)
	return nil
}

// npcSpawnEligible 前移了原按视口注册时的 EventNpc 闸门：
// EventNpc = 任务/事件触发器（触发某个任务才刷新的任务怪口径），
// 与 Java 参考 NpcViewport.visibleInLocalViewport 同规则——不注册、
// 不物化、无 Vitals/Brain，因此也永不进入死亡→重生刷新循环。
// eventtrigger.xml（任务触发生成器）另在 npc_catalog.go 装载期排除。
//
// 子场景闸门（2026-11-08）：creator 条目带 SubSceneNo（成都实测
// AttackNpc.xml 219 条中 135 条，如夜探守卫 newworld_nangong_yetan_*
// SubSceneNo=8961）是活动/分线态内容——官服只在子场景激活时由脚本
// 物化，基础场景从不出现。战斗脚本类（AttackNpc/BossNpc）带此标记
// 一律不注册：否则白名活动守卫会被静态摆进大场景主动咬路过玩家。
// 平民带 SubSceneNo 的照常注册（站桩无害，避免误伤活动对话内容）。
func npcSpawnEligible(npc npcSpawn) bool {
	if strings.EqualFold(strings.TrimSpace(npc.resolved.ScriptClass), "EventNpc") {
		return false
	}
	if subScene, exists := npc.resolved.Extensions["creator.SubSceneNo"]; exists && subScene != "" {
		switch strings.ToLower(strings.TrimSpace(npc.resolved.ScriptClass)) {
		case "attacknpc", "bossnpc":
			return false
		}
	}
	return true
}

// ---- 成员管理 ----

// attachMember 登记成员并按需启动场景 tick；带 player 的成员会被
// 实体化进可视权威 Scene，从而对其他客户端可见。重复 attach 时
// 沿用已分配的 wireID。返回该成员的 wire objectID。
func (r *sceneRun) attachMember(member *sceneMember) uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return member.objectID
	}
	if member.objectID != 0 {
		if existing, ok := r.members[member.objectID]; ok {
			member = existing
		}
	} else {
		if r.nextWireID == 0 {
			r.nextWireID = 2
		}
		// 远程玩家 wireID 留在无前缀低位空间（docs/id-system-analysis.md
		// §6.2 ID 布局）：0x11 前缀段专属玩家化身与 NPC 物化，低位段给
		// 动态挂接成员，两空间由前缀位天然隔离、永不碰撞。
		member.objectID = r.nextWireID
		r.nextWireID++
		r.members[member.objectID] = member
		if member.player != nil {
			r.materializeMemberLocked(member)
		}
	}
	r.startTickLocked()
	return member.objectID
}

// materializeMemberLocked 把成员物化为场景实体（0x0D payload 一次
// 性物化；后续 HP 变化经战斗 tick 的远程属性帧传播）。
// 调用方必须已持有 r.mu。
func (r *sceneRun) materializeMemberLocked(member *sceneMember) {
	wireID := member.objectID
	entityID := worldcore.EntityID(wireID)
	transform := worldcore.Transform{X: member.position.X, Y: member.position.Y, Z: member.position.Z, Orient: member.position.Orient}
	payload := member.player.addObjectAs(wireID, 1, member.position, member.visual)
	meta := sceneEntity{
		id:          wireID,
		description: "player:" + member.player.stateSnapshot().Name,
		ownerID:     1,
		configID:    "player",
		scriptClass: "Player",
		// 远程玩家可选中（LastObject/选靶前提），但绝不弹 NPC 菜单。
		interaction: npcInteractionSelectOnly,
		payload:     payload,
	}
	_ = r.scene.Add(worldcore.Entity{ID: entityID, OwnerID: 1, Kind: worldcore.EntityPlayer, Transform: transform})
	raw := r.combat.Raw()
	ecs.Write(raw, ecs.Id(entityID), ecs.C(transform), ecs.C(meta))
}

// detachMember 注销成员并移除其实体（其他客户端的 0x0E 由各自
// 下一次 reconcile 结算）；末成员离开时关闭整个运行时。
func (r *sceneRun) detachMember(objectID uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.members[objectID]; !exists {
		return
	}
	delete(r.members, objectID)
	entityID := worldcore.EntityID(objectID)
	if _, err := r.scene.Get(entityID); err == nil {
		_ = r.scene.Remove(entityID)
	}
	r.combat.Remove(entityID)
	if len(r.members) == 0 && !r.closed && !r.managed {
		r.shutdownLocked()
	}
}

// updateMemberPosition 同步场景侧权威位置：成员表、Transform 组件、
// 可视权威 Scene（revision，供新进入视口者对账），并向可见该玩家
// 的其他成员直发 0x20 移动帧。
func (r *sceneRun) updateMemberPosition(objectID uint32, position role.Position) {
	r.mu.Lock()
	defer r.mu.Unlock()
	member, exists := r.members[objectID]
	if !exists {
		return
	}
	member.position = position
	transform := worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}
	entityID := worldcore.EntityID(objectID)
	r.writeNpcTransform(entityID, transform)
	if current, err := r.scene.Get(entityID); err == nil {
		current.Entity.Transform = transform
		_ = r.scene.Replace(current.Entity)
	}
	_ = r.broadcastNPCLocked(objectID, serverMoving(objectID, 1, motionFromDest(transform)))
}

func (r *sceneRun) activateMember(objectID uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if member, exists := r.members[objectID]; exists {
		member.active = true
		// 巡逻首拍延迟：clientActivity 刚做完视口对账并排布 0x1F 重放
		// （250ms..5s），5s 后才开始向该成员广播 0x21。
		member.patrolReadyAt = time.Now().Add(patrolFirstBroadcastDelay)
	}
}

// shutdownLocked 停止 tick 与全部定时器，并把自己从注册表摘除。
// 调用方必须已持有 r.mu。
func (r *sceneRun) shutdownLocked() {
	r.closed = true
	r.epoch++
	for _, timer := range r.timers {
		timer.Stop()
	}
	r.timers = nil
	if r.ticking {
		close(r.tickStop)
		r.ticking = false
	}
	sceneRegistryMu.Lock()
	if sceneRegistry[r.key] == r {
		delete(sceneRegistry, r.key)
	}
	sceneRegistryMu.Unlock()
}

// ---- 中央 tick ----

func (r *sceneRun) startTickLocked() {
	if r.ticking || r.closed {
		return
	}
	r.ticking = true
	r.tickStop = make(chan struct{})
	r.tickDone = make(chan struct{})
	go func() {
		defer close(r.tickDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-r.tickStop:
				return
			case now := <-ticker.C:
				r.tickOnce(now)
			}
		}
	}()
}

// tickOnce 是每秒一次的权威 NPC 模拟入口：战斗结算、巡逻采样与刷新
// 周期（尸体 TTL/批量重生/宝箱过期）各为一段，均在 run.mu 内完成，
// 帧写入在锁外进行（transport.WriteFrame 线程安全；跨写者的相对顺序
// 不保证，这是本轮已接受的权衡）。
func (r *sceneRun) tickOnce(now time.Time) {
	r.combatTick(now)
	r.patrolTick(now)
	r.refreshTick(now)
	r.waveTick(now)
	r.mu.Lock()
	dirty := r.viewsDirty
	r.mu.Unlock()
	if dirty {
		r.reconcileMemberViews()
	}
}

// ---- NPC 注册（自 sceneLifecycle.registerNPC 迁入）----

// registerNPCLocked 注册单个 NPC：可视权威实体 + 四组件一次成型。
// 调用方必须已持有 r.mu。
// forkNPCID 把裸场景索引映射到 fork 对象 id 段（0x1100_0000|idx，
// docs/fork-object-id-migration.md）。已是大号 id（宝箱实体等）直通。
func forkNPCID(id uint32) uint32 {
	if id == 0 || id >= 0x01000000 {
		return id
	}
	return 0x11000000 | id
}

// forkNPCIndex 反解 fork id 的目录行索引（forkNPCID 的逆；裸 id 直通）。
func forkNPCIndex(id uint32) int {
	if id >= 0x11000000 {
		return int(id & 0xFFFFFF)
	}
	return int(id)
}

func (r *sceneRun) registerNPCLocked(id uint32, npc npcSpawn) error {
	id = forkNPCID(id)
	if id == 0 || id == 1 || id == playerObjectID {
		return fmt.Errorf("NPC scene identity %d collides with reserved player identity", id)
	}
	if npc.schedule != nil && !patrolCombatScriptClass(npc.resolved.ScriptClass) {
		// 巡逻 NPC 按共同时钟采样当前时刻位置：0x0D 载荷与初始
		// Transform 即时间表当前位置（场景运行时重建后续走）。
		position := npc.schedule.positionAt(time.Now().UnixMilli())
		npc.x, npc.y, npc.z, npc.orient = position.x, position.y, position.z, position.orient
	}
	orderedProperties, err := npc.resolved.OrderedProperties()
	if err != nil {
		return fmt.Errorf("resolve NPC %s properties: %w", npc.resolved.ConfigID, err)
	}
	payload, err := npcAddObject(id, npc, 0)
	if err != nil {
		return fmt.Errorf("encode NPC %s: %w", npc.resolved.ConfigID, err)
	}
	properties := worldcore.PropertyArchive{}
	for _, property := range orderedProperties {
		value, err := worldValue(property.Value)
		if err != nil {
			return fmt.Errorf("convert NPC %s property %s: %w", npc.resolved.ConfigID, property.Name, err)
		}
		if err := properties.Set(property.Name, value); err != nil {
			return err
		}
	}
	entityID := worldcore.EntityID(id)
	// NPC owner 官服化（npcOwnerID 注释；owner=1 是 fork 方言的 NPC 位，
	// 官服客户端把它当玩家化身→PvP 分支不自走）。
	if err := r.scene.Add(worldcore.Entity{
		ID:         entityID,
		OwnerID:    worldcore.EntityID(npcOwnerID(id)),
		Kind:       worldcore.EntityNPC,
		Transform:  worldcore.Transform{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient},
		Properties: properties,
	}); err != nil {
		return err
	}
	services := modernNPCServices(npc, taskCatalog)
	portal, hasPortal, portalErr := doorPortalRoute(npc)
	if portalErr != nil {
		return fmt.Errorf("resolve portal %s: %w", npc.resolved.ConfigID, portalErr)
	}
	funcs := r.npcFuncs.FuncsForNPC(npc.resolved.ConfigID)
	interaction := npcInteractionFor(npc.resolved.ScriptClass, services)
	meta := sceneEntity{
		id:          id,
		description: "npc:" + npc.resolved.ConfigID,
		ownerID:     npcOwnerID(id),
		configID:    npc.resolved.ConfigID,
		scriptClass: npc.resolved.ScriptClass,
		interaction: interaction,
		funcs:       funcs,
		services:    services,
		properties:  cloneIndexedProperties(orderedProperties),
		payload:     payload,
	}
	// msg24 战斗帧素材：Material/NpcNickName 缺列时保持空串（帧仍发）。
	if property, ok := npc.resolved.Properties["Material"]; ok {
		meta.model = property.Value.Text
	}
	if property, ok := npc.resolved.Properties["NpcNickName"]; ok {
		meta.wideName = property.Value.Text
	}
	// NPC 攻击招式（msg20 动画串）：NormalSkill 列（attacknpc col81）
	// 的值须是 zhaoshi_npc.ini 的现有节名才采用（表里混有模型名/其它
	// 命名空间的值，如 chuanyunsheri_nan 不在招式表）；否则攻击帧回落
	// jn_mon053_02 常量（Java 同款兜底）。
	if property, ok := npc.resolved.Properties["NormalSkill"]; ok && npcSkillActionKnown(property.Value.Text) {
		meta.normalSkill = property.Value.Text
	}
	if hasPortal {
		meta.portal = &portal
	}
	meta.schedule = npc.schedule
	if meta.schedule != nil {
		// 幂等：尸体 TTL 清除后重注册会再次走到这里，在巡表不得重复。
		onPatrol := false
		for _, existing := range r.patrolIDs {
			if existing == id {
				onPatrol = true
				break
			}
		}
		if !onPatrol {
			r.patrolIDs = append(r.patrolIDs, id)
		}
	}
	maxHP, maxMP := int32(1000), int32(100)
	level, powerNo := int32(1), int32(0)
	for _, property := range orderedProperties {
		switch property.Name {
		case "MaxHP":
			if property.Value.I32 > 0 {
				maxHP = property.Value.I32
			}
		case "MaxMP":
			if property.Value.I32 > 0 {
				maxMP = property.Value.I32
			}
		case "Level":
			if property.Value.I32 > 0 {
				level = property.Value.I32
			}
		case "PowerNo":
			powerNo = property.Value.I32
		}
	}
	// 实力包守方数值推导（2026-11-08 NPC 属性块迁移）：Base×Level 规则
	// 集中在 deriveNPCCombatStats；封劲暂不推导（见其注释）。
	stats := deriveNPCCombatStats(powerNo, level)
	meta.combatStats = &stats
	transform := worldcore.Transform{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient}
	vitals := sim.NewVitals(maxHP, maxMP)
	if interaction == npcInteractionCombat {
		brain := newNPCCombatState(npc, transform)
		r.spawnSceneEntityLocked(entityID, vitals, transform, meta, &brain)
	} else {
		r.spawnSceneEntityLocked(entityID, vitals, transform, meta, nil)
	}
	return nil
}

// ---- 组件访问器（自 sceneLifecycle 迁入，供视图委托）----

func (r *sceneRun) spawnSceneEntityLocked(id worldcore.EntityID, vitals sim.Vitals, transform worldcore.Transform, meta sceneEntity, brain *sim.Brain) {
	raw := r.combat.Raw()
	if brain != nil {
		ecs.Write(raw, ecs.Id(id), ecs.C(vitals), ecs.C(transform), ecs.C(meta), ecs.C(*brain))
		return
	}
	ecs.Write(raw, ecs.Id(id), ecs.C(vitals), ecs.C(transform), ecs.C(meta))
}

func (r *sceneRun) npcMeta(id worldcore.EntityID) (sceneEntity, bool) {
	return ecs.Read[sceneEntity](r.combat.Raw(), ecs.Id(id))
}

func (r *sceneRun) writeNpcMeta(id worldcore.EntityID, meta sceneEntity) {
	ecs.Write(r.combat.Raw(), ecs.Id(id), ecs.C(meta))
}

func (r *sceneRun) npcTransform(id worldcore.EntityID) (worldcore.Transform, bool) {
	return ecs.Read[worldcore.Transform](r.combat.Raw(), ecs.Id(id))
}

// npcVitals 读取 NPC 当前生命组件（调用方必须已持有 r.mu，与
// npcMeta/npcTransform 同契约）。重放 0x0D 后用它判断尸体状态。
func (r *sceneRun) npcVitals(id worldcore.EntityID) (sim.Vitals, bool) {
	return ecs.Read[sim.Vitals](r.combat.Raw(), ecs.Id(id))
}

func (r *sceneRun) writeNpcTransform(id worldcore.EntityID, transform worldcore.Transform) {
	ecs.Write(r.combat.Raw(), ecs.Id(id), ecs.C(transform))
}

func (r *sceneRun) unregisterNPC(id worldcore.EntityID) error {
	if err := r.scene.Remove(id); err != nil {
		return err
	}
	r.combat.Remove(id)
	return nil
}

// sortedMembersLocked 返回按 objectID 升序的成员切片。
// 调用方必须已持有 r.mu。
func (r *sceneRun) sortedMembersLocked() []*sceneMember {
	members := make([]*sceneMember, 0, len(r.members))
	for _, member := range r.members {
		members = append(members, member)
	}
	sort.Slice(members, func(i, j int) bool { return members[i].objectID < members[j].objectID })
	return members
}

type npcEntry struct {
	id        worldcore.EntityID
	meta      sceneEntity
	transform worldcore.Transform
}

// eachNPCLocked 以 EntityID 升序回调全部已注册 NPC（平民与敌对
// 均含）。回调收到的是组件副本，修改后经 writeNpcMeta/
// writeNpcTransform 写回。调用方必须已持有 r.mu。
func (r *sceneRun) eachNPCLocked(fn func(id worldcore.EntityID, meta *sceneEntity, transform *worldcore.Transform)) {
	entries := make([]npcEntry, 0)
	ecs.Query3[sceneEntity, worldcore.Transform, sim.Vitals](r.combat.Raw()).MapId(func(id ecs.Id, meta *sceneEntity, transform *worldcore.Transform, _ *sim.Vitals) {
		entries = append(entries, npcEntry{id: worldcore.EntityID(id), meta: *meta, transform: *transform})
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].id < entries[j].id })
	for i := range entries {
		fn(entries[i].id, &entries[i].meta, &entries[i].transform)
	}
}

type combatNPCEntry struct {
	id        worldcore.EntityID
	vitals    *sim.Vitals
	brain     *sim.Brain
	transform *worldcore.Transform
	meta      *sceneEntity
}

// eachCombatNPCFullLocked 以 EntityID 升序一次遍历敌对 NPC 的全部
// 四个组件（Vitals/Brain/Transform/sceneEntity），回调收到直指组件
// 存储的指针——就地修改即生效，无需逐 ID Read。
// 契约同 sim.World.EachCombatNPC：回调期间禁止一切结构性 ECS 操作；
// 对已有组件的值写入会使指针所指的值更新。
// 调用方必须已持有 r.mu。
func (r *sceneRun) eachCombatNPCFullLocked(fn func(id worldcore.EntityID, vitals *sim.Vitals, brain *sim.Brain, transform *worldcore.Transform, meta *sceneEntity)) {
	entries := make([]combatNPCEntry, 0)
	ecs.Query4[sim.Vitals, sim.Brain, worldcore.Transform, sceneEntity](r.combat.Raw()).MapId(func(id ecs.Id, vitals *sim.Vitals, brain *sim.Brain, transform *worldcore.Transform, meta *sceneEntity) {
		entries = append(entries, combatNPCEntry{id: worldcore.EntityID(id), vitals: vitals, brain: brain, transform: transform, meta: meta})
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].id < entries[j].id })
	for i := range entries {
		fn(entries[i].id, entries[i].vitals, entries[i].brain, entries[i].transform, entries[i].meta)
	}
}

// ---- 帧扇出 ----

// broadcastNPCLocked 把单帧扇出给可见该 NPC 的成员。
// 调用方必须已持有 r.mu。
func (r *sceneRun) broadcastNPCLocked(id uint32, frame []byte) error {
	for _, member := range r.sortedMembersLocked() {
		if member.view == nil || !member.view.npcVisible(id) {
			continue
		}
		if err := member.view.conn.WriteFrame(frame); err != nil {
			return err
		}
	}
	return nil
}
