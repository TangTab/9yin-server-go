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
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/npcfunc"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

type sceneMessageConnection interface {
	WriteFrame([]byte) error
}

// sceneLifecycle owns the server-side scene objects and the subset currently
// linked to one client's viewport. It is deliberately independent of NPC
// behaviour so monsters, gatherables and remote players can use the same path.
type sceneLifecycle struct {
	remote         string
	conn           sceneMessageConnection
	mu             sync.Mutex
	armed          bool
	combatActive   bool
	scene          *worldcore.Scene
	viewport       *worldcore.Viewport
	entities       map[worldcore.EntityID]sceneEntity
	combatActors   map[worldcore.EntityID]*entity.Actor
	combatStates   map[worldcore.EntityID]npcCombatState
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

// npcInteraction is a server-side routing decision, derived from the modern
// template ScriptClass plus the business fields that are actually implemented
// by this local server.  It deliberately keeps target selection separate from
// opening an NPC dialogue: combat targets still need LastObject for skill
// targeting, but must never receive a fabricated conversation window.
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

// npcService is derived from the modern template row itself. Unlike the
// legacy Npc_Func table, these values use the exact ConfigID referenced by the
// current creator XML and are therefore valid for the local modern client.
type npcService struct {
	mark   uint16
	label  string
	value  string
	source string
}

// Service marks are server-local routing IDs carried by the current client's
// native 0x1C SERVER_MENU. They deliberately do not reuse the legacy Npc_Func
// integer namespace: a selected mark always resolves against the NPC that
// emitted the menu.
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
		remote:       remote,
		conn:         conn,
		scene:        worldcore.NewScene(),
		viewport:     worldcore.NewViewport(),
		entities:     make(map[worldcore.EntityID]sceneEntity),
		combatActors: make(map[worldcore.EntityID]*entity.Actor),
		combatStates: make(map[worldcore.EntityID]npcCombatState),
		npcFuncs:     funcs,
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
	clear(s.combatActors)
	clear(s.combatStates)
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
	name, maxHP, maxMP := npc.resolved.ConfigID, int32(1000), int32(100)
	for _, property := range orderedProperties {
		switch property.Name {
		case "Name":
			if property.Value.Text != "" {
				name = property.Value.Text
			}
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
	s.combatActors[entityID] = entity.NewActor(id, 1, name, maxHP, maxMP)
	if interaction == npcInteractionCombat {
		s.combatStates[entityID] = newNPCCombatState(npc, worldcore.Transform{X: npc.x, Y: npc.y, Z: npc.z, Orient: npc.orient})
	}
	return nil
}

// portalRoute verifies an incoming direct Select against the live viewport
// and returns only a resource-backed Door/GotoDoor route. It deliberately
// excludes AdventureDoor and plot doors, whose clone/task admission protocol
// has not yet been recovered.
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

// portalAt activates an installed portal by proximity, mirroring the normal
// door walk-through behaviour. It intentionally uses X/Z only: doors on
// slopes and stairways often have a small creator/MoveTarget Y difference.
// A short server-side cooldown prevents a local same-scene endpoint from
// immediately retriggering another nearby portal on the next movement tick.
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

// npcInteractionFor makes ScriptClass the first gate.  The modern city05
// catalog contains AttackNpc and BossNpc alongside CommonNpc, so treating all
// of them as generic talkers is wrong even when the client sends the same
// initial Select packet.  For civilian/service NPCs we only open a dialogue
// if a service can actually be fulfilled locally; otherwise the selection is
// retained for inspection/targeting without inventing an empty "交谈" menu.
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
	delete(s.combatActors, entityID)
	delete(s.combatStates, entityID)
	return nil
}

// clientReady arms a fresh generation but deliberately does not write objects.
// The current client sends ClientReady before stage_main calls
// scene_obj:RegisterCallBack. A localhost response can otherwise create the
// logical object in that small gap without ever invoking its visual callback.
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

// clientActivity is the current-client readiness barrier. The first custom
// message after ClientReady arrives after the Lua entry function has progressed
// past scene_obj:RegisterCallBack.
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
		// Transform is the only state currently changed by the patrol loop.
		// Replaying every appearance/AI field here makes the current client
		// recreate its Actor2. Use the compact, negotiated coordinate update;
		// sceneObjectProperties remains the encoder for actual future changes.
		properties := sceneObjectTransformProperties(entity.id, entity.ownerID, entity.transform)
		if err := s.conn.WriteFrame(properties); err != nil {
			return fmt.Errorf("update scene object %d: %w", entity.id, err)
		}
		log.Printf("%s: updated scene object transform id=%d kind=%s", s.remote, entity.id, entity.description)
	}
	// A generation change means the client has already replaced its native
	// scene/object manager. Sending old-generation removals could target a newly
	// reused ID, so only emit 0x0E inside the same generation.
	if delta.PreviousGeneration == delta.Generation {
		for _, removal := range delta.Removes {
			payload := sceneRemoveObject(uint32(removal.Entity.ID), uint32(removal.Entity.OwnerID))
			if err := s.conn.WriteFrame(payload); err != nil {
				return fmt.Errorf("remove scene object %d: %w", removal.Entity.ID, err)
			}
		}
	}
	// Snapshot order is deterministic, but keep this defensive sort because the
	// wire object order is part of the captured compatibility baseline.
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

// objectRequest accepts C2S 0x07 (Select / object click) with FuncID==0.
// It sets LastObject, then uses the current client's numeric custom-message
// route to open the face-to-face NPC dialog.  A selected dialog entry, not the
// initial NPC click, creates the business view.
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
	// LastObject is an object-selection property, not a dialogue property.
	// Keep its server-side mirror independently from lastMenu*: attack targets
	// never open a talk window, but the 211 combat handler still needs to
	// resolve the same selected hostile actor.
	s.selectedObject = entity.id
	s.selectedOwner = entity.ownerID
	// AttackNpc/BossNpc and unsupported scene objects still become the player's
	// LastObject. That is the target selection used by the combat ingress, but
	// they must not receive the generic cinematic conversation that was formerly
	// sent for every spawned object.
	if entity.interaction != npcInteractionTalk {
		log.Printf("%s: object request sequence=%d id=%d config=%s script_class=%s interaction=%s; selected without dialogue",
			s.remote, request.Sequence, entity.id, entity.configID, entity.scriptClass, entity.interaction)
		return entity.description, nil
	}
	// Creator/template configuration is authoritative for the modern client.
	// The residual Npc_Func table is retained for offline auditing only: its
	// Chengdu bindings do not match the current Creator ConfigIDs, so exposing
	// them as live menu items creates actions this server cannot route.
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

// openDepotLocked starts the retail depot flow.  The current Lua has no
// Lua-level "open depot" receiver: form_depot.on_server_msg only refreshes an
// existing form.  Its form is therefore opened by the client's native handling
// of SERVER_CREATE_VIEW for VIEWPORT_DEPOT (4).  Once open, main_form_open
// emits C2S CLIENT_CUSTOMMSG_DEPOT_MSG (307, submessage 4), which the caller's
// normal frame log records for the next server-side handler step.
func (s *sceneLifecycle) openDepotLocked() error {
	const (
		viewportDepot = 4
		depotCapacity = 18 // current share/Rule/depotbox.ini StartCap
	)
	if err := s.conn.WriteFrame(serverCreateView(serverViewSpec{ID: viewportDepot, Capacity: depotCapacity})); err != nil {
		return fmt.Errorf("create depot view: %w", err)
	}
	return nil
}

// openShopLocked starts the native shop viewport flow. form_shop.lua obtains
// all four settings below from the View=61 createview callback; it does not
// load the NPC template itself. The catalog is therefore parsed server-side
// and every true item line is materialized as a ViewAdd object.
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

// sceneIdent is the client object key used by GetSceneObj / Select / 0x1C header.
func sceneIdent(objectID, ownerID uint32) string {
	return fmt.Sprintf("%d-%d", objectID, ownerID)
}

// sendTalkMenuLocked sends native 0x1C SERVER_MENU for modern open_npc_talk.
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

// encodeCinematicTalkMenu uses the current C++ GameMessageHandler's custom
// receiver table.  The table begins at custom ID 1 (custom_sysinfo); its menu
// entries are therefore 4..8, rather than the raw object-slot offsets 49..53
// that an earlier experiment incorrectly inferred.  String aliases are not
// accepted by this client, so the numeric first TVarList value is required.
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

// selectNPCMenu handles the second C2S Select: a non-zero FuncID/mark emitted
// from a previously sent native menu. It resolves the mark only within that
// NPC's configured modern services, clears the menu, then opens the requested
// business view. This is the same two-stage ownership boundary as the live
// server: click NPC -> SERVER_MENU -> choose mark -> business protocol.
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
		// The menu remains data-driven for every current service. Do not invent
		// a client-side action for transports/tasks/jobs before their respective
		// modern business packets and persisted data are proven.
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

// npcServiceMenu is deliberately derived solely from the selected NPC's
// Npc_Func rows.  The old universal “与我交谈 / 离开” pair made a depot,
// ferryman, trainer and ordinary citizen indistinguishable.
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
		return 805000000 // shop
	case 0x1002:
		return 807000000 // depot
	case 0x1003:
		return 860000022 // transport
	case 0x1004:
		return 806000036 // life job
	case 0x1005:
		return 806000013 // gem service
	case 0x1006:
		return 809000000 // guild
	case 0x1007:
		return 881000001 // home point
	case 0x1008:
		return 100000001 // task offer placeholder until task state is persisted
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

// modernNPCServices extracts business identifiers from the resolved modern
// NPC configuration. Values are retained in Extensions because they are server
// business configuration, not visual 0x09 properties. Creator values take
// precedence over their template counterpart: a creator is an instance-level
// override, while a template is the shared default. The marks deliberately
// occupy a separate range from legacy FuncID values.
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

// playableNPCServices is intentionally narrower than modernNPCServices.
// The latter reports every configured capability; this function exposes only
// capabilities that have both a proven S2C handler and complete local data.
// Pending task/transport/guild configuration remains visible in the audit
// artifact, not as a button that fails after the player clicks it.
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

// effectiveNPCBusinessValue resolves non-visual configuration fields with the
// same precedence as NPC instance configuration: creator override first,
// template default second. Empty strings do not erase a configured default;
// extracted creator XML routinely contains unrelated optional empty fields.
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

// moveNPC changes the authoritative world transform, emits a proven modern
// 0x20 destination and confirms arrival with 0x1F. The motion tail remains
// zero until its animation-specific semantics are proven from modern client
// code; the destination and lifecycle are not speculative.
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

// startPatrol runs a deliberately small, deterministic simulation set. A
// Chengdu viewport can contain hundreds of objects; moving all of them before
// behaviour/routing is implemented would only make the client stutter. Zero
// disables it through -npc-patrol-count.
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
		// Hostile actors own their movement through the combat tick. Letting the
		// old decorative patrol timer move them concurrently would overwrite a
		// chase destination and make their aggro behaviour nondeterministic.
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
