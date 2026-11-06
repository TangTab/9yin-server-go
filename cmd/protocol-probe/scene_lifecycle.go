package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/npcfunc"
	"github.com/local/9yin-go-server/internal/role"
	"github.com/local/9yin-go-server/internal/sim"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

type sceneMessageConnection interface {
	WriteFrame([]byte) error
}

// sceneLifecycle 持有服务器侧的场景对象以及当前关联到单个客户端视口
// 的子集。它刻意与 NPC 行为解耦，使怪物、采集物和远端玩家可以走同
// 一条路径。
type sceneLifecycle struct {
	remote       string
	conn         sceneMessageConnection
	mu           sync.Mutex
	armed        bool
	combatActive bool
	scene        *worldcore.Scene
	viewport     *worldcore.Viewport
	entities     map[worldcore.EntityID]sceneEntity
	// combat 以 ECS 组件存储持有全部已注册 NPC 的模拟态：
	// 平民只有 Vitals，敌对 NPC 另带 Brain。所有访问必须在
	// s.mu 临界区内，纪律与被替换的两张平行 map 完全一致。
	combat         *sim.World
	closed         bool
	epoch          uint64
	timers         []*time.Timer
	npcFuncs       *npcfunc.Registry
	patrolStarted  bool
	selectedObject uint32
	selectedOwner  uint32
	lastMenuObject uint32
	lastMenuOwner  uint32
	lastMenuAt     time.Time
	lastMenuMovie  bool
	portalCooldown time.Time
}

type sceneEntity struct {
	id          uint32
	description string
	payload     func() []byte
	ownerID     uint32
	transform   worldcore.Transform
	properties  []clientdata.IndexedProperty
	patrolSign  float32
	configID    string
	scriptClass string
	interaction npcInteraction
	funcs       []npcfunc.Binding
	services    []npcService
	portal      *scenePortalRoute
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
	return &sceneLifecycle{
		remote:   remote,
		conn:     conn,
		scene:    worldcore.NewScene(),
		viewport: worldcore.NewViewport(),
		entities: make(map[worldcore.EntityID]sceneEntity),
		combat:   sim.New(),
		npcFuncs: funcs,
	}
}

func (s *sceneLifecycle) begin(config, resource string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	if _, err := s.scene.Begin(config, resource); err != nil {
		return err
	}
	clear(s.entities)
	s.combat.Reset()
	s.armed = false
	s.combatActive = false
	s.patrolStarted = false
	s.selectedObject = 0
	s.selectedOwner = 0
	s.lastMenuObject = 0
	s.lastMenuOwner = 0
	s.lastMenuAt = time.Time{}
	s.lastMenuMovie = false
	s.epoch++
	s.stopReplayTimersLocked()
	return nil
}

func (s *sceneLifecycle) registerNPC(id uint32, npc npcSpawn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == 0 || id == 1 {
		return fmt.Errorf("NPC scene identity %d collides with reserved player identity", id)
	}
	orderedProperties, err := npc.resolved.OrderedProperties(clientdata.VisibleNPCModernV1())
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
	if err := s.scene.Add(worldcore.Entity{
		ID:         entityID,
		OwnerID:    1,
		Kind:       worldcore.EntityNPC,
		Transform:  worldcore.Transform{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient},
		Properties: properties,
	}); err != nil {
		return err
	}
	services := modernNPCServices(npc)
	portal, hasPortal, portalErr := doorPortalRoute(npc)
	if portalErr != nil {
		return fmt.Errorf("resolve portal %s: %w", npc.resolved.ConfigID, portalErr)
	}
	funcs := s.npcFuncs.FuncsForNPC(npc.resolved.ConfigID)
	interaction := npcInteractionFor(npc.resolved.ScriptClass, services)
	sceneMeta := sceneEntity{
		id:          id,
		description: "npc:" + npc.resolved.ConfigID,
		ownerID:     1,
		transform:   worldcore.Transform{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient},
		configID:    npc.resolved.ConfigID,
		scriptClass: npc.resolved.ScriptClass,
		interaction: interaction,
		funcs:       funcs,
		services:    services,
		properties:  cloneIndexedProperties(orderedProperties),
		payload: func() []byte {
			return append([]byte(nil), payload...)
		},
	}
	if hasPortal {
		sceneMeta.portal = &portal
	}
	s.entities[entityID] = sceneMeta
	maxHP, maxMP := int32(1000), int32(100)
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
		}
	}
	vitals := sim.NewVitals(maxHP, maxMP)
	if interaction == npcInteractionCombat {
		s.combat.SpawnCombatNPC(entityID, vitals, newNPCCombatState(npc, worldcore.Transform{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient}))
	} else {
		s.combat.SpawnNPC(entityID, vitals)
	}
	return nil
}

// portalRoute 对传入的直接 Select 按活动视口进行校验，只返回有资源
// 支撑的 Door/GotoDoor 路线。它刻意排除 AdventureDoor 和剧情门，这些
// 门对应的副本/任务准入协议尚未还原。
func (s *sceneLifecycle) portalRoute(request clientObjectRequest) (scenePortalRoute, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return scenePortalRoute{}, false, nil
	}
	entity, exists := s.entities[worldcore.EntityID(request.ObjectID)]
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
	defer s.mu.Unlock()
	if s.closed || now.Before(s.portalCooldown) {
		return scenePortalRoute{}, false
	}
	for _, entity := range s.entities {
		if entity.portal == nil {
			continue
		}
		route := *entity.portal
		dx := position.X - route.trigger.X
		dz := position.Z - route.trigger.Z
		if dx*dx+dz*dz > route.triggerRadius*route.triggerRadius {
			continue
		}
		s.portalCooldown = now.Add(2 * time.Second)
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

func (s *sceneLifecycle) unregister(id uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entityID := worldcore.EntityID(id)
	if err := s.scene.Remove(entityID); err != nil {
		return err
	}
	delete(s.entities, entityID)
	s.combat.Remove(entityID)
	return nil
}

// clientReady 只为新一代布防，刻意不写入对象。当前客户端在 stage_main
// 调用 scene_obj:RegisterCallBack 之前就发送 ClientReady。若立即响应，
// 本机连接可能在这段小空窗内创建逻辑对象，却从不调用其视觉回调。
func (s *sceneLifecycle) clientReady() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
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
	defer s.mu.Unlock()
	if s.closed || !s.armed {
		return nil
	}
	log.Printf("%s: post-ready client activity; materializing scene objects", s.remote)
	if err := s.reconcileViewportLocked(); err != nil {
		return err
	}
	s.armed = false
	s.combatActive = true
	return nil
}

func (s *sceneLifecycle) reconcileViewport() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reconcileViewportLocked()
}

func (s *sceneLifecycle) reconcileViewportLocked() error {
	delta := s.viewport.Plan(s.scene.Snapshot())
	for _, update := range delta.Updates {
		entity, exists := s.entities[update.Entity.ID]
		if !exists {
			return fmt.Errorf("missing property encoder for scene object %d", update.Entity.ID)
		}
		// Transform 是目前巡逻循环唯一会改变的状态。在这里重放所有外观/
		// AI 字段会让当前客户端重建它的 Actor2。请使用紧凑的协商坐标更新；
		// sceneObjectProperties 仍留作未来真正变更时的编码器。
		properties := sceneObjectTransformProperties(entity.id, entity.ownerID, entity.transform)
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
		entity, exists := s.entities[id]
		if !exists {
			return fmt.Errorf("missing payload encoder for scene object %d", id)
		}
		payload := entity.payload()
		digest := sha256.Sum256(payload)
		log.Printf("%s: materialize scene object id=%d kind=%s opcode=0x%02X payload_len=%d payload_sha256=%x",
			s.remote, id, entity.description, payload[0], len(payload), digest)
		if err := s.conn.WriteFrame(payload); err != nil {
			return fmt.Errorf("add scene object %d: %w", id, err)
		}
		if entity.ownerID != 0 {
			location := serverLocation(entity.id, entity.ownerID, entity.transform)
			if err := s.conn.WriteFrame(location); err != nil {
				return fmt.Errorf("locate scene object %d: %w", id, err)
			}
			locationReplays = append(locationReplays, sceneLocationReplay{
				id: uint32(id), payload: append([]byte(nil), location...),
			})
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", nil
	}
	entity, exists := s.entities[worldcore.EntityID(request.ObjectID)]
	if !exists || entity.ownerID != request.OwnerID {
		return "", fmt.Errorf("unknown scene object id=%d owner=%d", request.ObjectID, request.OwnerID)
	}
	if entity.id == s.lastMenuObject && entity.ownerID == s.lastMenuOwner && time.Since(s.lastMenuAt) < time.Second {
		log.Printf("%s: coalesced duplicate object request sequence=%d id=%d owner=%d", s.remote, request.Sequence, entity.id, entity.ownerID)
		return entity.description, nil
	}
	if err := s.conn.WriteFrame(sceneObjectObjectProperty(1, 1, 100, entity.id, entity.ownerID)); err != nil {
		return "", fmt.Errorf("set player LastObject=%d-%d: %w", entity.id, entity.ownerID, err)
	}
	// LastObject 是对象选择属性，不是对话属性。它的服务器侧镜像要与
	// lastMenu* 分开保存：攻击目标从不打开对话窗口，但 211 战斗处理仍需
	// 解析同一个被选中的敌对角色。
	s.selectedObject = entity.id
	s.selectedOwner = entity.ownerID
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
		viewportShop    = 61
		shopPageSize    = 10
		shopCapacity    = 100
		propShopID      = 101
		propPageNum     = 102
		propPageCount   = 103
		propShopType    = 104
		propConfigID    = 105
		propAmount      = 106
		propPriceGold   = 107
		propPriceSilver = 108
		propPriceCard   = 109
		propMaxAmount   = 110
		maxBuyAmount    = 99
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
	s.mu.Lock()
	defer s.mu.Unlock()
	entity, exists := s.entities[worldcore.EntityID(request.ObjectID)]
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
		// 菜单对每项现有服务都保持数据驱动。在传送/任务/职业各自的现代
		// 业务包和持久化数据得到验证之前，不要凭空发明客户端动作。
		return fmt.Errorf("NPC %s service %q (mark=%d) is discovered but not yet implemented", entity.configID, service.source, service.mark)
	}
	return nil
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
func modernNPCServices(npc npcSpawn) []npcService {
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
	_, acceptSource := value("table@TaskCanAccept")
	_, submitSource := value("table@TaskCanSubmit")
	if acceptSource != "" || submitSource != "" {
		source := acceptSource
		if source == "" {
			source = submitSource
		}
		items = append(items, npcService{mark: markTask, label: "任务", source: source})
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

// moveNPC 修改权威的世界变换，发出已验证的现代 0x20 目的地并用 0x1F
// 确认到达。motion 尾部保持为零，直到其动画专属语义从现代客户端代码
// 中得到验证；目的地和生命周期都不是猜测。
func (s *sceneLifecycle) moveNPC(id uint32, destination worldcore.Transform, arrival time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	entityID := worldcore.EntityID(id)
	meta, exists := s.entities[entityID]
	if !exists {
		return fmt.Errorf("move unknown scene object %d", id)
	}
	current, err := s.scene.Get(entityID)
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
	s.entities[entityID] = meta
	if err := s.conn.WriteFrame(serverMoving(meta.id, meta.ownerID, motionFromDest(destination))); err != nil {
		return fmt.Errorf("send NPC moving %d: %w", id, err)
	}
	if err := s.reconcileViewportLocked(); err != nil {
		return err
	}
	if arrival <= 0 {
		arrival = 1500 * time.Millisecond
	}
	epoch := s.epoch
	timer := time.AfterFunc(arrival, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.epoch != epoch {
			return
		}
		current, exists := s.entities[entityID]
		if !exists {
			return
		}
		if err := s.conn.WriteFrame(serverLocation(current.id, current.ownerID, current.transform)); err != nil {
			log.Printf("%s: confirm NPC move id=%d: %v", s.remote, current.id, err)
			return
		}
		log.Printf("%s: NPC id=%d arrived at (%.2f, %.2f, %.2f)", s.remote, current.id, current.transform.X, current.transform.Y, current.transform.Z)
	})
	s.timers = append(s.timers, timer)
	log.Printf("%s: NPC id=%d moving to (%.2f, %.2f, %.2f)", s.remote, id, destination.X, destination.Y, destination.Z)
	return nil
}

// startPatrol 运行一组刻意很小且确定性的模拟集合。成都的视口可能包含
// 数百个对象；在行为/路由实现之前全都在动只会让客户端卡顿。可通过
// -npc-patrol-count 传零来禁用。
func (s *sceneLifecycle) startPatrol(count int) {
	if count <= 0 {
		return
	}
	s.mu.Lock()
	if s.closed || s.patrolStarted {
		s.mu.Unlock()
		return
	}
	s.patrolStarted = true
	ids := make([]int, 0, len(s.entities))
	for id, meta := range s.entities {
		// 敌对角色的移动由战斗 tick 掌管。让旧的装饰性巡逻定时器并发地
		// 移动它们会覆盖追击目的地，使它们的仇恨行为变得不确定。
		if meta.interaction != npcInteractionCombat {
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)
	if count > len(ids) {
		count = len(ids)
	}
	selected := make([]uint32, count)
	for index := range selected {
		selected[index] = uint32(ids[index])
	}
	s.mu.Unlock()
	for index, id := range selected {
		s.schedulePatrolStep(id, 3*time.Second+time.Duration(index)*700*time.Millisecond)
	}
	log.Printf("%s: started low-rate patrol for %d nearby NPCs", s.remote, len(selected))
}

func (s *sceneLifecycle) schedulePatrolStep(id uint32, delay time.Duration) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	epoch := s.epoch
	timer := time.AfterFunc(delay, func() {
		s.mu.Lock()
		if s.closed || s.epoch != epoch {
			s.mu.Unlock()
			return
		}
		entityID := worldcore.EntityID(id)
		meta, exists := s.entities[entityID]
		if !exists {
			s.mu.Unlock()
			return
		}
		if meta.patrolSign == 0 {
			meta.patrolSign = 1
		}
		destination := meta.transform
		destination.X += meta.patrolSign * 1.5
		meta.patrolSign = -meta.patrolSign
		s.entities[entityID] = meta
		s.mu.Unlock()
		if err := s.moveNPC(id, destination, 1500*time.Millisecond); err != nil {
			log.Printf("%s: patrol move id=%d: %v", s.remote, id, err)
			return
		}
		s.schedulePatrolStep(id, 5*time.Second)
	})
	s.timers = append(s.timers, timer)
	s.mu.Unlock()
}

func (entity sceneEntity) propertiesForTransform() []clientdata.IndexedProperty {
	properties := cloneIndexedProperties(entity.properties)
	for index := range properties {
		switch properties[index].Name {
		case "PosiX":
			properties[index].Value = clientdata.Float32Value(entity.transform.X)
		case "PosiY":
			properties[index].Value = clientdata.Float32Value(entity.transform.Y)
		case "PosiZ":
			properties[index].Value = clientdata.Float32Value(entity.transform.Z)
		case "Orient":
			properties[index].Value = clientdata.Float32Value(entity.transform.Orient)
		}
	}
	return properties
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
		return worldcore.Value{}, fmt.Errorf("unsupported wire type %d", value.Type)
	}
}

func (s *sceneLifecycle) scheduleLocationReplaysLocked(replays []sceneLocationReplay) {
	epoch := s.epoch
	for _, delay := range actor2LocationReplayDelays {
		delay := delay
		batch := make([]sceneLocationReplay, len(replays))
		for i, replay := range replays {
			batch[i] = sceneLocationReplay{id: replay.id, payload: append([]byte(nil), replay.payload...)}
		}
		timer := time.AfterFunc(delay, func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.closed || s.epoch != epoch {
				return
			}
			sent := 0
			for _, replay := range batch {
				if _, exists := s.entities[worldcore.EntityID(replay.id)]; !exists {
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
		s.timers = append(s.timers, timer)
	}
}

func (s *sceneLifecycle) stopReplayTimersLocked() {
	for _, timer := range s.timers {
		timer.Stop()
	}
	s.timers = nil
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
	defer s.mu.Unlock()
	s.closed = true
	s.epoch++
	s.stopReplayTimersLocked()
}
