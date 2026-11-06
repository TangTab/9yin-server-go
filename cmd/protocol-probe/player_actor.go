package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

const (
	playerObjectID uint32 = 1
	playerOwnerID  uint32 = 1

	// These are appended immediately after playerProgressFields (171..207).
	// Their order must stay identical to sceneVisiblePropertyFields because
	// FxNet2 property archives carry only the numeric index, not the name.
	propNeigongPKStatus uint16 = 208
	propDead            uint16 = 209
	propCantUseSkill    uint16 = 210
	propCurSkillID      uint16 = 211
	propPauseTime       uint16 = 212
	// Native main-role UI binds these as ints. They describe the advertised
	// recovery rate; the server remains the source of actual HP/MP changes.
	propHPHeartSpeed    uint16 = 213
	propHPHeartSpeedAdd uint16 = 214
	propMPHeartSpeed    uint16 = 215
	propMPHeartSpeedAdd uint16 = 216
	// form_main_player's recovery tooltip gets the actual restored amount from
	// these separate properties, not from *HeartSpeed.
	propHPUpSpeed    uint16 = 217
	propHPUpSpeedAdd uint16 = 218
	propMPUpSpeed    uint16 = 219
	propMPUpSpeedAdd uint16 = 220
	// The normal server skill state publishes the complete public skill-use
	// tuple.  FxGameLogic also reads ModifySkillLockTime directly while
	// deciding whether a self-target skill may enter its native action.
	propCurSkillEffectID    uint16 = 221
	propCurSkillLevel       uint16 = 222
	propCurSkillTarget      uint16 = 223
	propModifySkillLockTime uint16 = 224
	// Learned skill objects use this byte independently from the server's
	// request acceptance. The modern shortcut UI reads CanUse directly; when
	// it is absent the icon is rendered unavailable even though the guarded
	// combat request path can still reach C2S 211.
	propSkillCanUse uint16 = 225
	// The current client does not represent every affiliation through School.
	// Its role panel chooses Force, then NewSchool, then School.  Keeping these
	// negotiated and explicit is therefore required for the later factions and
	// hidden schools to avoid being rendered as \"无门派\".
	propForce     uint16 = 226
	propNewSchool uint16 = 227

	// BufferListStr is the native gameplay BuffContainer snapshot. BufferInfo
	// slots are the native visual-store projection. Both must describe the same
	// active Buffs: UI-only BufferInfo is insufficient for qinggong_3's native
	// HasBuff("buf_qg_drift") gate.
	bufferListPropertyIndex      uint16 = 143
	bufferInfoFirstPropertyIndex uint16 = bufferListPropertyIndex + 1
	bufferInfoSlotCount          uint16 = 24

	redSuperArmorBuffSlot uint16 = 1
	waterDriftBuffSlot    uint16 = 2
	waterJumpBuffSlot     uint16 = 3
	innerPowerBuffSlot    uint16 = 4
	yiHuaSkillBuffSlot    uint16 = 5
	yiHuaSkill05BuffSlot  uint16 = 6
	taiJiStanceBuffSlot   uint16 = 7
	taiJiSkill05BuffSlot  uint16 = 8
	taiJiSkill07BuffSlot  uint16 = 9
	// 明玉神功“料敌先机”是独立的短期闪避 Buff；它不能覆盖常驻内功
	// Buff，也不能挤占轻功/技能 Buff 的槽位。
	yihuaPassiveBuffSlot uint16 = 10

	redSuperArmorStaticData uint32 = 8682
	waterDriftStaticData    uint32 = 216
	waterJumpStaticData     uint32 = 5786

	// Equipped inner power uses the same native BuffStore, but must not share
	// a temporary combat/qinggong slot.  Its lease is renewed from the
	// authoritative equip state on spawn and every 215 response.
	innerPowerBuffLease = 24 * time.Hour

	// SP is the current modern client rage resource.  The installed skill
	// definitions and live 211 handling use 100 as the full local-test pool.
	// All GM and future combat gain paths go through these helpers so a client
	// property can never become the authority for the resource.
	maxPlayerSP int32 = 100

	// Local-test sitting recovery is settled once per existing server second.
	// User rule: a living actor recovers a full MaxHP/MaxMP pool within ten
	// seconds, regardless of equipment or inner-power-derived totals.
	sitcrossFullRecoverySeconds int32 = 10
	sitcrossHeartMillis         int32 = 1000

	// Combat recovery never restores real HP.  It only reduces the native
	// HitHP-HP bruised-blood layer once per authoritative server second.
	combatBruiseRecoveryPerSecond int32 = 10
)

var nativeBuffSlots = func() []uint16 {
	slots := make([]uint16, bufferInfoSlotCount)
	for index := range slots {
		slots[index] = uint16(index + 1)
	}
	return slots
}()

func bufferInfoPropertyIndex(slot uint16) uint16 {
	if slot == 0 || slot > bufferInfoSlotCount {
		panic(fmt.Sprintf("invalid BufferInfo slot %d", slot))
	}
	return bufferInfoFirstPropertyIndex + slot - 1
}

func bufferInfoPropertyName(slot uint16) string {
	if slot == 0 || slot > bufferInfoSlotCount {
		panic(fmt.Sprintf("invalid BufferInfo slot %d", slot))
	}
	return fmt.Sprintf("BufferInfo%d", slot)
}

// playerActor bridges the authoritative runtime actor to the current client's
// negotiated visible-property schema. Currency is session-authoritative for
// the current local-test server; the purchase transaction will become its
// persistent owner once inventory/currency storage is introduced.
type playerActor struct {
	actor                 *entity.Actor
	silver                int32
	faction               string
	mu                    sync.Mutex
	motion                playerMotion
	buffs                 map[uint16]activePlayerBuff
	activeQingGong        map[string]struct{}
	shortcuts             []playerShortcut
	learnedSkills         map[string]int32
	skillCooldowns        map[string]time.Time
	skillTeamCooldowns    map[int32]time.Time
	currentSkillID        string
	currentSkillEffectID  string
	currentSkillLevel     int32
	currentSkillTarget    uint64
	activeParry           activeParryRequest
	skillLockUntil        time.Time
	reviveAt              time.Time
	innerPowerProcReadyAt time.Time
	innerPowerHealReadyAt time.Time
	progress              playerProgress
}

// playerMotion is deliberately server-owned GM state.  Its values are sent
// over the normal modern object-property receiver, never through S->C 0x1E.
type playerMotion struct {
	moveSpeed float32
	runSpeed  float32
	gravity   float32
}

// activePlayerBuff is the server-side counterpart to one native BufferInfo
// slot.  The client owns rendering, but gameplay decisions must read this
// state instead of trusting a client-side icon.
type activePlayerBuff struct {
	staticData uint32
	expiresUTC time.Time
	configID   string
	level      int32
}

// playerBuffRules mirrors buf_fixed_red -> proppack 9679.  It deliberately
// does not claim damage immunity: the current config grants hit-effect and
// skill-lock immunity only.  Combat/control ingress will consult this state
// when those C2S paths are introduced.
type playerBuffRules struct {
	CantHitEffect     bool
	CantBeSkillLocked bool
}

func newPlayerActor(name string, silver int32) *playerActor {
	learnedSkills := make(map[string]int32, len(starterSkillViews))
	for _, skill := range starterSkillViews {
		learnedSkills[skill.configID] = skill.level
	}
	p := &playerActor{
		actor:              entity.NewPlayer(playerObjectID, playerOwnerID, name),
		silver:             silver,
		faction:            defaultRoleFaction,
		motion:             playerMotion{moveSpeed: 25, runSpeed: 25, gravity: 9.8},
		buffs:              make(map[uint16]activePlayerBuff),
		activeQingGong:     make(map[string]struct{}),
		learnedSkills:      learnedSkills,
		skillCooldowns:     make(map[string]time.Time),
		skillTeamCooldowns: make(map[int32]time.Time),
		progress:           newPlayerProgress(),
	}
	p.syncEquippedNeiGongBuffLocked(time.Now())
	p.syncEquippedNeiGongResourcesLocked()
	return p
}

func (p *playerActor) learnedSkillLevel(configID string) (int32, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	level, exists := p.learnedSkills[configID]
	return level, exists && level > 0
}

func (p *playerActor) learnSkill(configID string, level int32) {
	if configID == "" || level < 1 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.learnedSkills[configID] = level
}

func (p *playerActor) progressProperties() []clientdata.IndexedProperty {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.properties()
}

// setFaction changes the native School property only after the GM whitelist
// and persistent appearance update have succeeded.  The modern client uses
// that one property for original schools, forces and new schools alike.
func (p *playerActor) setFaction(faction string) ([]byte, error) {
	p.mu.Lock()
	p.faction = faction
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, factionProperties(faction))
}

// factionProperties mirrors form_main_player.refresh_role_changed: original
// schools populate School, 江湖势力 populate Force, and later schools populate
// NewSchool.  Empty companion fields deliberately clear a previous GM choice.
func factionProperties(faction string) []clientdata.IndexedProperty {
	school, force, newSchool := faction, "", ""
	switch {
	case strings.HasPrefix(faction, "force_"):
		school, force = "", faction
	case strings.HasPrefix(faction, "newschool_"):
		school, newSchool = "", faction
	}
	return []clientdata.IndexedProperty{
		{Index: 53, Name: "School", Value: clientdata.StringValue(school)},
		{Index: propForce, Name: "Force", Value: clientdata.StringValue(force)},
		{Index: propNewSchool, Name: "NewSchool", Value: clientdata.StringValue(newSchool)},
	}
}

// setActiveParry records a client-originated 218 request for the future
// authoritative incoming-hit resolver.  It has no client-side visual side
// effect by itself: visual parry/hit actions are selected only after an NPC
// or player attack reaches that resolver.
func (p *playerActor) setActiveParry(value activeParryRequest) {
	p.mu.Lock()
	p.activeParry = value
	p.mu.Unlock()
}

func (p *playerActor) activeParrySnapshot() activeParryRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeParry
}

type npcAttackOutcome struct {
	stateChanged        bool
	parried             bool
	dodged              bool
	dead                bool
	innerPowerTriggered bool
	innerPowerHeal      int32
}

// resolveNPCAttack is the server-authoritative incoming-hit decision. The
// current client already supplies the C2S active-parry ingress (218), but its
// exact combat enum is not recovered; for the local server every fresh 218
// request grants one 600ms parry window. Dodge is deterministic per attacker
// and second so a reconnect cannot reroll an accepted outcome.
func (p *playerActor) resolveNPCAttack(attackerID uint32, damage int32, now time.Time) npcAttackOutcome {
	if damage <= 0 || p.actor.Snapshot().HP <= 0 {
		return npcAttackOutcome{}
	}
	parry := p.activeParrySnapshot()
	if !parry.At.IsZero() && now.Sub(parry.At) >= 0 && now.Sub(parry.At) <= 600*time.Millisecond {
		return npcAttackOutcome{parried: true}
	}
	profile, hasProfile := p.equippedInnerPowerPassive()
	activePassive, triggered := false, false
	if hasProfile && profile.mode == innerPowerModeIncomingDodgeHeal {
		activePassive, triggered = p.applyIncomingDodgePassive(profile, attackerID, now)
	}
	dodgeChance := baseLocalDodgeProbability
	if activePassive {
		dodgeChance += profile.dodgeAdd
	}
	if combatPercentRoll(attackerID, now, 18) < dodgeChance {
		healed := int32(0)
		if hasProfile && profile.mode == innerPowerModeIncomingDodgeHeal && !p.innerPowerHealCoolingDown(now) {
			state := p.actor.Snapshot()
			amount := (state.MaxHP*profile.healPercent + 99) / 100
			if p.actor.ApplyHeal(amount) {
				healed = amount
			}
			p.setInnerPowerHealCooldown(now.Add(profile.healCooldown))
		}
		return npcAttackOutcome{stateChanged: triggered || healed > 0, dodged: true, innerPowerTriggered: triggered, innerPowerHeal: healed}
	}
	changed := p.actor.ApplyDamage(damage)
	if !changed {
		return npcAttackOutcome{}
	}
	rules := p.buffRules(now)
	if !rules.CantBeSkillLocked {
		p.mu.Lock()
		p.skillLockUntil = now.Add(500 * time.Millisecond)
		p.mu.Unlock()
	}
	dead := p.actor.Snapshot().HP <= 0
	if dead {
		p.mu.Lock()
		p.reviveAt = now.Add(playerAutoReviveDelay)
		p.mu.Unlock()
	}
	return npcAttackOutcome{stateChanged: true, dead: dead}
}

// combatPercentRoll is deterministic for the same attacker and server tick.
// It keeps local testing reproducible without letting reconnects reroll an
// already-settled NPC attack.
func combatPercentRoll(attackerID uint32, now time.Time, salt uint64) int32 {
	value := uint64(attackerID)*1103515245 + uint64(now.Unix())*2654435761 + salt*97
	return int32(value % 100)
}

// applyIncomingDodgePassive is the generic event-mode executor for an
// inner-power profile compiled from its installed description and child Buff.
// Other passive modes plug into the same pre-hit dispatch point.
func (p *playerActor) applyIncomingDodgePassive(profile innerPowerPassiveProfile, attackerID uint32, now time.Time) (active, triggered bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	current := p.buffs[yihuaPassiveBuffSlot]
	active = current.staticData == profile.procStatic && current.expiresUTC.After(now.UTC())
	if active || p.innerPowerProcReadyAt.After(now) || combatPercentRoll(attackerID, now, 17) >= profile.procChance {
		return active, false
	}
	expires := now.UTC().Add(profile.procLifetime)
	p.buffs[yihuaPassiveBuffSlot] = activePlayerBuff{staticData: profile.procStatic, expiresUTC: expires, configID: profile.procBuffID, level: profile.buffLevel}
	p.innerPowerProcReadyAt = now.Add(profile.procCooldown)
	return true, true
}

func (p *playerActor) innerPowerHealCoolingDown(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.innerPowerHealReadyAt.After(now)
}

func (p *playerActor) setInnerPowerHealCooldown(next time.Time) {
	p.mu.Lock()
	p.innerPowerHealReadyAt = next
	p.mu.Unlock()
}

// expireInnerPowerPassive clears only 料敌先机. The ordinary connection tick
// then sends one normal 0x10 vital archive, removing BufferInfo10 and the
// matching BufferListStr entry together.
func (p *playerActor) expireInnerPowerPassive(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[yihuaPassiveBuffSlot]
	if !ok || buff.expiresUTC.After(now.UTC()) {
		return false
	}
	delete(p.buffs, yihuaPassiveBuffSlot)
	return true
}

func (p *playerActor) skillLocked(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.skillLockUntil.After(now)
}

// reviveWhenDue implements the local-test respawn policy. It intentionally
// uses normal actor properties; the exact official relive UI/teleport route is
// a separate protocol feature and is not fabricated here.
func (p *playerActor) reviveWhenDue(now time.Time) bool {
	p.mu.Lock()
	if p.reviveAt.IsZero() || now.Before(p.reviveAt) {
		p.mu.Unlock()
		return false
	}
	p.reviveAt = time.Time{}
	p.skillLockUntil = time.Time{}
	p.mu.Unlock()
	p.actor.Revive()
	return true
}

func (p *playerActor) progressViewFrames() ([][]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.viewFrames()
}

func (p *playerActor) equipNeiGong(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.progress.equip(id); err != nil {
		return err
	}
	p.syncEquippedNeiGongBuffLocked(time.Now())
	p.syncEquippedNeiGongResourcesLocked()
	return nil
}

// syncEquippedNeiGongResourcesLocked applies the equipped book's resource
// modifiers to the real server Actor. The visible protocol continues to send
// base MaxHP/MaxMP plus raw *Add fields, but all server costs, damage, and
// recovery use Actor.MaxHP/MaxMP (the total derived caps). The current modern
// client refreshes MP as MaxMP + MaxMPAdd directly; there is no multiplier on
// the wire value.
//
// p.mu must be held by the caller.
func (p *playerActor) syncEquippedNeiGongResourcesLocked() {
	hpAdd, mpAddRaw := p.progress.equippedResourceAdds()
	base := p.actor.Snapshot()
	p.actor.SetDerivedMaxima(base.BaseMaxHP+hpAdd, base.BaseMaxMP+mpAddRaw)
}

// syncEquippedNeiGongBuffLocked mirrors the exact installed varprop-derived
// effect of the equipped book into a dedicated native BufferInfo slot.  This
// is deliberately not a server-originated 0x1E custom: FxGameLogic consumes
// BufferInfo and BufferListStr through ordinary object properties.
func (p *playerActor) syncEquippedNeiGongBuffLocked(now time.Time) {
	book := p.progress.book(p.progress.curNeiGong)
	if book == nil || book.buffID == "" || book.buffStaticData == 0 || book.buffLevel <= 0 {
		delete(p.buffs, innerPowerBuffSlot)
		return
	}
	p.buffs[innerPowerBuffSlot] = activePlayerBuff{
		staticData: book.buffStaticData,
		expiresUTC: now.UTC().Add(innerPowerBuffLease),
		configID:   book.buffID,
		level:      book.buffLevel,
	}
}

func (p *playerActor) facultyReady(id string) (uint16, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.facultyReady(id)
}

func (p *playerActor) facultyBegin(style int32) (uint16, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.progress.facultyBegin(style, time.Now()); err != nil {
		return 0, false, err
	}
	// Ordinary inner cultivation is a persistent background conversion and
	// must not force the actor into LS_FACULTY. Only the active 演武 path owns
	// that visible logic/action state.
	if style == facultyStyleAct {
		p.actor.SetLogicState(logicStateFaculty)
	} else {
		p.actor.SetLogicState(0)
	}
	return 0, false, nil
}

func (p *playerActor) facultyExit() {
	p.mu.Lock()
	p.progress.facultyExit()
	p.mu.Unlock()
	p.actor.SetLogicState(0)
}

func (p *playerActor) progressBookFrame(slot uint16) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.bookFrame(slot)
}

func (p *playerActor) facultyTick(now time.Time) (uint16, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot, changed := p.progress.advanceFaculty(now)
	if changed {
		// A settlement may have crossed one or more levels.  The exact
		// varprop record is already re-evaluated by advanceFaculty; publish
		// its native BuffStore state and make the new derived caps authoritative
		// before the caller sends the normal role-property/View refresh.
		p.syncEquippedNeiGongBuffLocked(now)
		p.syncEquippedNeiGongResourcesLocked()
	}
	return slot, changed
}

func (p *playerActor) facultySnapshot() facultyProgressSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.snapshot()
}

func (p *playerActor) restoreFaculty(value facultyProgressSnapshot, now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.progress.restore(value, now)
	if p.progress.facultyState == facultyStateConvert && p.progress.facultyStyle == facultyStyleAct {
		p.actor.SetLogicState(logicStateFaculty)
	} else {
		p.actor.SetLogicState(0)
	}
	p.syncEquippedNeiGongBuffLocked(now)
	p.syncEquippedNeiGongResourcesLocked()
}

// activateQingGong remembers one server-approved ActiveQGSkillRec entry for
// this scene session. The client checks this table before enabling an equipped
// qinggong_gaojie (advanced-qinggong) skill.
func (p *playerActor) activateQingGong(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.activeQingGong[id]; exists {
		return false
	}
	p.activeQingGong[id] = struct{}{}
	return true
}

func (p *playerActor) stateProperties() []clientdata.IndexedProperty {
	state := p.actor.Snapshot()
	hpRecovery := sitcrossRecoveryPerSecond(state.MaxHP)
	mpRecovery := sitcrossRecoveryPerSecond(state.MaxMP)
	currentSkillID, currentSkillEffectID, currentSkillLevel, currentSkillTarget := p.currentSkillStateSnapshot()
	hpRatio := int32(int64(state.HP) * 100 / int64(state.MaxHP))
	hitHPRatio := int32(int64(state.HitHP) * 100 / int64(state.MaxHP))
	mpRatio := int32(int64(state.MP) * 100 / int64(state.MaxMP))
	properties := []clientdata.IndexedProperty{
		{Index: 21, Name: "LogicState", Value: clientdata.ByteValue(state.LogicState)},
		{Index: 28, Name: "HPRatio", Value: clientdata.Int32Value(hpRatio)},
		{Index: 29, Name: "MPRatio", Value: clientdata.Int32Value(mpRatio)},
		// The client derives its displayed totals from these base values plus
		// MaxHPAdd/MaxMPAdd. Actor.Max* remains the server-side total cap.
		{Index: 30, Name: "MaxHP", Value: clientdata.Int32Value(state.BaseMaxHP)},
		{Index: 31, Name: "MaxMP", Value: clientdata.Int32Value(state.BaseMaxMP)},
		{Index: 32, Name: "HP", Value: clientdata.Int32Value(state.HP)},
		{Index: 33, Name: "MP", Value: clientdata.Int32Value(state.MP)},
		{Index: 26, Name: "SpeedRatio", Value: clientdata.Int32Value(state.SpeedRatio)},
		{Index: 112, Name: "HitHP", Value: clientdata.Int32Value(state.HitHP)},
		{Index: 113, Name: "HitHPRatio", Value: clientdata.Int32Value(hitHPRatio)},
		{Index: 114, Name: "QingGongPoint", Value: clientdata.Int32Value(state.QingGongPoint)},
		{Index: 115, Name: "MaxQingGongPoint", Value: clientdata.Int32Value(state.MaxQingGongPoint)},
		{Index: 116, Name: "MaxQingGongPointAdd", Value: clientdata.Int32Value(state.MaxQingGongPointAdd)},
		{Index: propNeigongPKStatus, Name: "NeigongPKStatus", Value: clientdata.Int32Value(0)},
		{Index: propDead, Name: "Dead", Value: clientdata.Int32Value(boolInt32(state.HP <= 0))},
		{Index: propCantUseSkill, Name: "CantUseSkill", Value: clientdata.Int32Value(boolInt32(p.skillLocked(time.Now())))},
		{Index: propCurSkillID, Name: "CurSkillID", Value: clientdata.StringValue(currentSkillID)},
		{Index: propCurSkillEffectID, Name: "CurSkillEffectID", Value: clientdata.StringValue(currentSkillEffectID)},
		{Index: propCurSkillLevel, Name: "CurSkillLevel", Value: clientdata.Int32Value(currentSkillLevel)},
		{Index: propCurSkillTarget, Name: "CurSkillTarget", Value: clientdata.Value{Type: clientdata.WireObject, U64: currentSkillTarget}},
		// Local loopback should never fail the client's skill/action
		// synchronization tolerance check because of a missing property (zero).
		{Index: propModifySkillLockTime, Name: "ModifySkillLockTime", Value: clientdata.Int32Value(5000)},
		{Index: propHPHeartSpeed, Name: "HPHeartSpeed", Value: clientdata.Int32Value(sitcrossHeartMillis)},
		{Index: propHPHeartSpeedAdd, Name: "HPHeartSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: propMPHeartSpeed, Name: "MPHeartSpeed", Value: clientdata.Int32Value(sitcrossHeartMillis)},
		{Index: propMPHeartSpeedAdd, Name: "MPHeartSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: propHPUpSpeed, Name: "HPUpSpeed", Value: clientdata.Int32Value(hpRecovery)},
		{Index: propHPUpSpeedAdd, Name: "HPUpSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: propMPUpSpeed, Name: "MPUpSpeed", Value: clientdata.Int32Value(mpRecovery)},
		{Index: propMPUpSpeedAdd, Name: "MPUpSpeedAdd", Value: clientdata.Int32Value(0)},
		{Index: 36, Name: "Name", Value: clientdata.WideStringValue(state.Name)},
		{Index: 111, Name: "CapitalType1", Value: clientdata.Int32Value(p.silver)},
		{Index: bufferListPropertyIndex, Name: "BufferListStr", Value: clientdata.StringValue(p.bufferListString(time.Now()))},
	}
	properties = append(properties, factionProperties(p.faction)...)
	for _, slot := range nativeBuffSlots {
		properties = append(properties, clientdata.IndexedProperty{
			Index: bufferInfoPropertyIndex(slot), Name: bufferInfoPropertyName(slot), Value: clientdata.StringValue(p.bufferInfo(slot, time.Now())),
		})
	}
	properties = append(properties, baseMotionProperties(p.motionSnapshot())...)
	properties = append(properties, qinggongMotionProperties()...)
	return append(properties, p.progressProperties()...)
}

func boolInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

// baseMotionProperties supplies the player physics that FxGameLogic reads
// while creating the ordinary jump action.  The client has no fallback
// server-side source for Gravity: it only applies it when this negotiated
// property exists.  9.8 is the current client's standard jump gravity.
func baseMotionProperties(motion playerMotion) []clientdata.IndexedProperty {
	return []clientdata.IndexedProperty{
		{Index: 168, Name: "Gravity", Value: clientdata.Float32Value(motion.gravity)},
		{Index: 169, Name: "GravityAdd", Value: clientdata.Float32Value(0)},
		// Modern FxGameLogic FallDownManager queries DropHeightPub from the
		// local player object. Its fallback is only 5.0 world units; use a
		// deliberately high local-test safe threshold so normal terrain
		// descent exits back to the grounded action rather than fall-down.
		{Index: 170, Name: "DropHeightPub", Value: clientdata.Float32Value(100000)},
	}
}

func (p *playerActor) motionSnapshot() playerMotion {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.motion
}

// setMoveSpeed and setGravity return a single normal 0x10 update archive.
// MoveSpeed and RunSpeed are changed together so the GM control has one
// unambiguous meaning for walking and running.
func (p *playerActor) setMoveSpeed(value float32) ([]byte, error) {
	p.mu.Lock()
	p.motion.moveSpeed = value
	p.motion.runSpeed = value
	motion := p.motion
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 23, Name: "MoveSpeed", Value: clientdata.Float32Value(motion.moveSpeed)},
		{Index: 25, Name: "RunSpeed", Value: clientdata.Float32Value(motion.runSpeed)},
	})
}

func (p *playerActor) setGravity(value float32) ([]byte, error) {
	p.mu.Lock()
	p.motion.gravity = value
	motion := p.motion
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 168, Name: "Gravity", Value: clientdata.Float32Value(motion.gravity)},
	})
}

// activateRedSuperArmor begins one authoritative red-super-armor lease. Its
// return value is the native BufferInfo1 payload consumed by FxGameLogic's
// BuffStore setter:
// StaticData,OwnerID,OwnerSerial,Level,EndTimeMs,Count (all hexadecimal).
func (p *playerActor) activateRedSuperArmor(now time.Time) (string, time.Time) {
	return p.activateBuff(redSuperArmorBuffSlot, redSuperArmorStaticData, 20*time.Second, now)
}

func (p *playerActor) activateBuff(slot uint16, staticData uint32, lifetime time.Duration, now time.Time) (string, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	expires := now.UTC().Add(lifetime)
	p.buffs[slot] = activePlayerBuff{staticData: staticData, expiresUTC: expires, level: 1}
	return p.bufferInfoLocked(slot, now), expires
}

func (p *playerActor) activateNamedBuff(slot uint16, staticData uint32, configID string, lifetime time.Duration, now time.Time) (string, time.Time) {
	return p.activateNamedBuffAtLevel(slot, staticData, configID, 1, lifetime, now)
}

func (p *playerActor) activateNamedBuffAtLevel(slot uint16, staticData uint32, configID string, level int32, lifetime time.Duration, now time.Time) (string, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if level < 1 {
		level = 1
	}
	expires := now.UTC().Add(lifetime)
	p.buffs[slot] = activePlayerBuff{
		staticData: staticData,
		expiresUTC: expires,
		configID:   configID,
		level:      level,
	}
	return p.bufferInfoLocked(slot, now), expires
}

// activateSkillBuff projects resource-compiled skill Buffs into the native
// BufferInfo1..24 store. System leases keep slots 1..10; routine Buffs reuse
// their existing ConfigID slot or allocate from 11..24.
func (p *playerActor) activateSkillBuff(definition skillBuffDefinition, now time.Time) (uint16, string, time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	slot := definition.preferredSlot
	if slot == 0 {
		for candidate, active := range p.buffs {
			if active.configID == definition.configID {
				slot = candidate
				break
			}
		}
	}
	if slot == 0 {
		for candidate := yihuaPassiveBuffSlot + 1; candidate <= bufferInfoSlotCount; candidate++ {
			active, occupied := p.buffs[candidate]
			if !occupied || !active.expiresUTC.After(now.UTC()) {
				slot = candidate
				break
			}
		}
	}
	if slot == 0 || slot > bufferInfoSlotCount {
		return 0, "", time.Time{}, false
	}
	lifetime := definition.lifetime
	if lifetime <= 0 {
		lifetime = time.Millisecond
	}
	expires := now.UTC().Add(lifetime)
	level := definition.level
	if level < 1 {
		level = 1
	}
	p.buffs[slot] = activePlayerBuff{
		staticData: definition.staticData,
		expiresUTC: expires,
		configID:   definition.configID,
		level:      level,
	}
	return slot, p.bufferInfoLocked(slot, now), expires, true
}

func (p *playerActor) nextSkillBuffVariant(variants []skillBuffDefinition, now time.Time) (skillBuffDefinition, bool) {
	if len(variants) == 0 {
		return skillBuffDefinition{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	slot := variants[0].preferredSlot
	active, exists := p.buffs[slot]
	if !exists || !active.expiresUTC.After(now.UTC()) {
		return variants[0], true
	}
	for index, variant := range variants {
		if active.staticData == variant.staticData {
			return variants[(index+1)%len(variants)], true
		}
	}
	return variants[0], true
}

func (p *playerActor) bufferInfo(slot uint16, now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bufferInfoLocked(slot, now)
}

func (p *playerActor) bufferInfoLocked(slot uint16, now time.Time) string {
	buff, ok := p.buffs[slot]
	if !ok || !buff.expiresUTC.After(now.UTC()) {
		return ""
	}
	level := buff.level
	if level < 1 {
		level = 1
	}
	return fmt.Sprintf("%X,%X,%X,%X,%X,%X", buff.staticData, playerObjectID, playerOwnerID, level, buff.expiresUTC.UnixMilli(), 1)
}

// bufferListString follows the retained BufferModule::RefreshBuffersInfo
// layout: ConfigID,UniqueID,Level,OwnerIdent-OwnerSerial,EndTime,EffectType,
// OverlayRows|. It is deliberately generated from the same server-authoritative
// leases as BufferInfo, so gameplay and visual Buff state cannot diverge.
func (p *playerActor) bufferListString(now time.Time) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.bufferListStringLocked(now)
}

func (p *playerActor) bufferListStringLocked(now time.Time) string {
	var list strings.Builder
	for _, slot := range nativeBuffSlots {
		buff, ok := p.buffs[slot]
		if !ok || !buff.expiresUTC.After(now.UTC()) {
			continue
		}
		configID := buff.configID
		if configID == "" {
			configID, ok = activeBuffConfigID(buff.staticData)
		}
		if !ok {
			continue
		}
		level := buff.level
		if level < 1 {
			level = 1
		}
		fmt.Fprintf(&list, "%s,local-%d,%d,%d-%d,%d,1,0|", configID, slot, level, playerObjectID, playerOwnerID, buff.expiresUTC.UnixMilli())
	}
	return list.String()
}

func activeBuffConfigID(staticData uint32) (string, bool) {
	switch staticData {
	case redSuperArmorStaticData:
		return "buf_fixed_red", true
	case waterDriftStaticData:
		return "buf_qg_drift", true
	case waterJumpStaticData:
		return "buf_qg_dst", true
	default:
		return "", false
	}
}

func (p *playerActor) hasActiveBuff(slot uint16, staticData uint32, now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	return ok && buff.staticData == staticData && buff.expiresUTC.After(now.UTC())
}

func (p *playerActor) clearBuff(slot uint16, expectedStaticData uint32) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || buff.staticData != expectedStaticData {
		return "", false
	}
	delete(p.buffs, slot)
	return "", true
}

// expireBuff clears only the exact lease that created this timer.  A later
// refresh is never removed by an obsolete timer.
func (p *playerActor) expireBuff(slot uint16, expectedStaticData uint32, expected time.Time, now time.Time) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	buff, ok := p.buffs[slot]
	if !ok || buff.staticData != expectedStaticData || !buff.expiresUTC.Equal(expected) || buff.expiresUTC.After(now.UTC()) {
		return "", false
	}
	delete(p.buffs, slot)
	return "", true
}

func (p *playerActor) buffRules(now time.Time) playerBuffRules {
	red := p.hasActiveBuff(redSuperArmorBuffSlot, redSuperArmorStaticData, now)
	return playerBuffRules{CantHitEffect: red, CantBeSkillLocked: red}
}

// qinggongMotionProperties is sourced from the currently installed
// share/Skill/QingGong/QGDefine.ini.  The client does not derive these values
// from the 216 action request: it reads them from its player object before
// resolving the motion.  Keep base and Add properties separate because Lua
// and FxGameLogic both query the pair.
func qinggongMotionProperties() []clientdata.IndexedProperty {
	const (
		waterSpeed      float32 = 7.5 // qinggong_2 Para1
		waterJumpSpeed  float32 = 6.0 // qinggong_3 Para1
		secondJumpSpeed float32 = 6.0 // qinggong_4 Para1
		thirdJumpSpeed  float32 = 6.0 // qinggong_5 Para1
		airRushSpeed    float32 = 15  // qinggong_6 Para1
		airRushDist     float32 = 12  // qinggong_6 Para2
		// The live test confirms the type-7 Para order is speed, then range:
		// Para1=25 produces the intended fast 雁行功 rush, while Para2=6.5 is
		// its travel range.  This differs from the type-6 air-rush ordering.
		landRushSpeed float32 = 25  // qinggong_7 Para1
		landRushDist  float32 = 6.5 // qinggong_7 Para2
		climbSpeed    float32 = 7.5 // qinggong_11 Para1
		// These are not aliases for the three action-speed values above.
		// FxGameLogic uses share/Skill/rush_range.ini during the rush action
		// gates, including qinggong_3 / 点水跳.  Keeping the historical
		// speed-as-range placeholders here makes that native gate evaluate
		// against the wrong water range.
		groundRushRange    float32 = 5.0 // rush_range.ini GRushRange
		airRushRange       float32 = 8.0 // rush_range.ini ARushRange
		waterRushRange     float32 = 8.0 // rush_range.ini WRushRange
		groundRushRangeAdd float32 = 1.5 // rush_range.ini GRushRangeAdd
		airRushRangeAdd    float32 = 6.5 // rush_range.ini ARushRangeAdd
		waterRushRangeAdd  float32 = 6.5 // rush_range.ini WRushRangeAdd
	)
	f := func(index uint16, name string, value float32) clientdata.IndexedProperty {
		return clientdata.IndexedProperty{Index: index, Name: name, Value: clientdata.Float32Value(value)}
	}
	return []clientdata.IndexedProperty{
		f(117, "RunSpeedAdd", 0), f(118, "JumpSpeedAdd", 0),
		f(119, "DriftSpeed", waterSpeed), f(120, "DriftSpeedAdd", 0),
		f(121, "DriftJumpSpeed", waterJumpSpeed), f(122, "DriftJumpSpeedAdd", 0),
		f(123, "SndJumpSpeed", secondJumpSpeed), f(124, "SndJumpSpeedAdd", 0),
		f(125, "ThdJumpSpeed", thirdJumpSpeed), f(126, "ThdJumpSpeedAdd", 0),
		f(127, "AirRushSpeed", airRushSpeed), f(128, "AirRushSpeedAdd", 0),
		f(129, "AirRushDist", airRushDist), f(130, "AirRushDistAdd", 0),
		f(131, "LandRushSpeed", landRushSpeed), f(132, "LandRushSpeedAdd", 0),
		f(133, "LandRushDist", landRushDist), f(134, "LandRushDistAdd", 0),
		f(135, "ClimbSpeed", climbSpeed), f(136, "ClimbSpeedAdd", 0),
		f(137, "GRushRange", groundRushRange), f(138, "GRushRangeAdd", groundRushRangeAdd),
		f(139, "ARushRange", airRushRange), f(140, "ARushRangeAdd", airRushRangeAdd),
		f(141, "WRushRange", waterRushRange), f(142, "WRushRangeAdd", waterRushRangeAdd),
	}
}

func (p *playerActor) vitalUpdate() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, p.stateProperties())
}

// sitcrossTick advances only the server-owned HP/MP state. The caller sends
// the resulting normal 0x10 archive, so the client never invents resources
// from HPHeartSpeed/MPHeartSpeed by itself.
func (p *playerActor) sitcrossTick() bool {
	state := p.actor.Snapshot()
	return p.actor.RestoreWhileSit(sitcrossRecoveryPerSecond(state.MaxHP), sitcrossRecoveryPerSecond(state.MaxMP))
}

// sitcrossRecoveryPerSecond rounds upward so non-divisible derived maxima
// still reach their full value in no more than ten one-second settlements.
func sitcrossRecoveryPerSecond(maximum int32) int32 {
	if maximum <= 0 {
		return 0
	}
	return (maximum + sitcrossFullRecoverySeconds - 1) / sitcrossFullRecoverySeconds
}

// combatBruiseTick settles the secondary HitHP layer while the actor remains
// in combat. The ordinary vital update already contains HitHP and
// HitHPRatio, so this uses the proven normal 0x10 property route.
func (p *playerActor) combatBruiseTick() bool {
	return p.actor.RecoverBruisedHPWhileFighting(combatBruiseRecoveryPerSecond)
}

// setCombatPresence owns the player-side lifecycle for the local hostile AI.
// Any retained NPC threat interrupts sitting before the first hit is settled;
// ordinary state is restored only after every threat has cleared. This keeps
// recovery, damage and the client LogicState on one authoritative transition.
func (p *playerActor) setCombatPresence(active bool) bool {
	state := p.actor.Snapshot()
	if state.HP <= 0 {
		return false
	}
	if active {
		if state.LogicState == entity.LogicStateFighting {
			return false
		}
		p.actor.SetLogicState(entity.LogicStateFighting)
		return true
	}
	if state.LogicState != entity.LogicStateFighting {
		return false
	}
	p.actor.SetLogicState(entity.LogicStateNormal)
	return true
}

func (p *playerActor) currentSkillSnapshot() string {
	id, _, _, _ := p.currentSkillStateSnapshot()
	return id
}

func (p *playerActor) currentSkillStateSnapshot() (string, string, int32, uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentSkillID, p.currentSkillEffectID, p.currentSkillLevel, p.currentSkillTarget
}

// clearCurrentSkill clears only the exact skill invocation that scheduled the
// timer. A later invocation must not be cancelled by an older timer.
func (p *playerActor) clearCurrentSkill(expected string) bool {
	p.mu.Lock()
	if p.currentSkillID != expected {
		p.mu.Unlock()
		return false
	}
	p.currentSkillID = ""
	p.currentSkillEffectID = ""
	p.currentSkillLevel = 0
	p.currentSkillTarget = 0
	p.mu.Unlock()
	p.actor.SetLogicState(0)
	return true
}

func (p *playerActor) currentSkillClearUpdate() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 21, Name: "LogicState", Value: clientdata.ByteValue(0)},
		{Index: propCurSkillID, Name: "CurSkillID", Value: clientdata.StringValue("")},
		{Index: propCurSkillEffectID, Name: "CurSkillEffectID", Value: clientdata.StringValue("")},
		{Index: propCurSkillLevel, Name: "CurSkillLevel", Value: clientdata.Int32Value(0)},
		{Index: propCurSkillTarget, Name: "CurSkillTarget", Value: clientdata.ObjectValue(0, 0)},
	})
}

// facultyNameResetFrame clears only the client-side FacultyName property.
// Opening form_wuxue_faculty happens after the 1006/1 selection response, so
// its FacultyName binder can miss the original value change.  A 1006/24
// refresh sends this clear frame immediately before the normal vital update;
// restoring the real name then reliably runs set_control_state without
// changing the server-owned selection or pretending that 演武 has started.
func (p *playerActor) facultyNameResetFrame() ([]byte, bool, error) {
	p.mu.Lock()
	hasFacultyName := p.progress.facultyName != ""
	p.mu.Unlock()
	if !hasFacultyName {
		return nil, false, nil
	}
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propFacultyName, Name: "FacultyName", Value: clientdata.StringValue("")},
	})
	return frame, true, err
}

func (p *playerActor) setSilver(value int32) {
	p.mu.Lock()
	p.silver = value
	p.mu.Unlock()
}

func clampPlayerSP(value int32) int32 {
	if value < 0 {
		return 0
	}
	if value > maxPlayerSP {
		return maxPlayerSP
	}
	return value
}

// setSP changes the server-authoritative rage value and immediately returns
// the ordinary 0x10 property update consumed by the main-role UI.
func (p *playerActor) setSP(value int32) (int32, []byte, error) {
	p.mu.Lock()
	p.progress.attributes.sp = clampPlayerSP(value)
	sp := p.progress.attributes.sp
	p.mu.Unlock()
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propSP, Name: "SP", Value: clientdata.Int32Value(sp)},
	})
	return sp, frame, err
}

// addSP is the single future entry point for combat/skill rage awards as well
// as GM adjustments.  Saturating here guarantees that 211 cost checks read
// exactly the value the client was just shown.
func (p *playerActor) addSP(delta int32) (int32, []byte, error) {
	p.mu.Lock()
	next := int64(p.progress.attributes.sp) + int64(delta)
	if next < 0 {
		next = 0
	}
	if next > int64(maxPlayerSP) {
		next = int64(maxPlayerSP)
	}
	p.progress.attributes.sp = int32(next)
	sp := p.progress.attributes.sp
	p.mu.Unlock()
	frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propSP, Name: "SP", Value: clientdata.Int32Value(sp)},
	})
	return sp, frame, err
}

func (p *playerActor) addSilver(amount int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if amount > 0 && p.silver > math.MaxInt32-amount {
		p.silver = math.MaxInt32
	} else {
		p.silver += amount
	}
	return p.silver
}

func (p *playerActor) silverUpdate() ([]byte, error) {
	p.mu.Lock()
	silver := p.silver
	p.mu.Unlock()
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 111, Name: "CapitalType1", Value: clientdata.Int32Value(silver)},
	})
}

func (p *playerActor) addObject(location role.Position, visual roleVisual) []byte {
	state := p.actor.Snapshot()
	motion := p.motionSnapshot()
	msg := make([]byte, 0x3f, 0x120)
	msg[0] = 0x0D
	binary.LittleEndian.PutUint32(msg[1:], state.ObjectID)
	binary.LittleEndian.PutUint32(msg[5:], state.OwnerID)
	binary.LittleEndian.PutUint32(msg[0x09:], math.Float32bits(location.X))
	binary.LittleEndian.PutUint32(msg[0x0D:], math.Float32bits(location.Y))
	binary.LittleEndian.PutUint32(msg[0x11:], math.Float32bits(location.Z))
	binary.LittleEndian.PutUint32(msg[0x15:], math.Float32bits(location.Orient))

	count := uint16(0)
	addByte := func(index uint16, value byte) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = append(msg, value)
		count++
	}
	addInt := func(index uint16, value int32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, uint32(value))
		count++
	}
	addFloat := func(index uint16, value float32) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint32(msg, math.Float32bits(value))
		count++
	}
	addString := func(index uint16, value string) { msg = appendStringProperty(msg, index, value); count++ }
	addObject := func(index uint16, value uint64) {
		msg = binary.LittleEndian.AppendUint16(msg, index)
		msg = binary.LittleEndian.AppendUint64(msg, value)
		count++
	}

	addByte(0, 2) // Type = TYPE_PLAYER
	addByte(1, visual.sex)
	addString(2, visual.photo)
	addInt(3, 0) // Camp
	addInt(4, 0) // Race
	addInt(5, 0) // Job
	addInt(6, 1) // Level
	addFloat(9, location.X)
	addFloat(10, location.Y)
	addFloat(11, location.Z)
	addFloat(12, location.Orient)
	addString(13, visual.hair)
	addString(14, visual.face)
	addString(15, visual.cloth)
	addString(16, visual.pants)
	addString(17, visual.shoes)
	addString(18, visual.actionSet)
	addString(19, "stand")
	addInt(20, 0)
	addByte(21, state.LogicState)
	addFloat(22, 8.0)
	addFloat(23, motion.moveSpeed)
	addFloat(24, 7.5)
	addFloat(25, motion.runSpeed)
	addInt(26, state.SpeedRatio)
	addByte(27, 0)
	addInt(28, int32(int64(state.HP)*100/int64(state.MaxHP)))
	addInt(29, int32(int64(state.MP)*100/int64(state.MaxMP)))
	addInt(30, state.BaseMaxHP)
	addInt(31, state.BaseMaxMP)
	addInt(32, state.HP)
	addInt(33, state.MP)
	addInt(112, state.HitHP)
	addInt(113, int32(int64(state.HitHP)*100/int64(state.MaxHP)))
	addInt(114, state.QingGongPoint)
	addInt(115, state.MaxQingGongPoint)
	addInt(116, state.MaxQingGongPointAdd)
	addInt(propNeigongPKStatus, 0)
	addInt(propDead, boolInt32(state.HP <= 0))
	addInt(propCantUseSkill, boolInt32(p.skillLocked(time.Now())))
	currentSkillID, currentSkillEffectID, currentSkillLevel, currentSkillTarget := p.currentSkillStateSnapshot()
	addString(propCurSkillID, currentSkillID)
	addString(propCurSkillEffectID, currentSkillEffectID)
	addInt(propCurSkillLevel, currentSkillLevel)
	addObject(propCurSkillTarget, currentSkillTarget)
	addInt(propModifySkillLockTime, 5000)
	addString(bufferListPropertyIndex, p.bufferListString(time.Now()))
	for _, property := range baseMotionProperties(motion) {
		addFloat(property.Index, property.Value.F32)
	}
	for _, property := range qinggongMotionProperties() {
		addFloat(property.Index, property.Value.F32)
	}
	addInt(111, p.silver) // CapitalType1: 碎银; negotiated in scene property table.
	for _, slot := range nativeBuffSlots {
		addString(bufferInfoPropertyIndex(slot), p.bufferInfo(slot, time.Now()))
	}
	for _, property := range p.progressProperties() {
		msg = appendNPCProperty(msg, property)
		count++
	}
	addString(35, visual.hair)
	msg = appendWideStringProperty(msg, 36, state.Name)
	count++
	msg = appendObjectProperty(msg, 100, 0, 0)
	count++
	binary.LittleEndian.PutUint16(msg[0x3d:], count)
	return msg
}

func sendPlayerSpawn(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {
	// Preserve any still-active native BuffStore state in a newly created
	// player object. BufferInfo is handled by the client's C++ property binder.
	log.Printf("native Buff spawn player=%d-%d red=%q drift=%q jump=%q neigong=%q", playerObjectID, playerOwnerID, player.bufferInfo(redSuperArmorBuffSlot, time.Now()), player.bufferInfo(waterDriftBuffSlot, time.Now()), player.bufferInfo(waterJumpBuffSlot, time.Now()), player.bufferInfo(innerPowerBuffSlot, time.Now()))
	if err := sendPlayerAddObject(conn, player, location, visual); err != nil {
		return err
	}
	return sendPlayerLocationAndVitals(conn, player, location)
}

// sendPlayerAddObject creates the native player object without issuing the
// separate ServerLocation motion update. Live scene replacement needs this
// split: stage_main requires 1-1 before it opens the target scene, while the
// 0x1F visual relocation must wait until target ClientReady.
func sendPlayerAddObject(conn sceneMessageConnection, player *playerActor, location role.Position, visual roleVisual) error {
	return conn.WriteFrame(player.addObject(location, visual))
}

func sendPlayerLocationAndVitals(conn sceneMessageConnection, player *playerActor, location role.Position) error {
	if err := conn.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(location))); err != nil {
		return err
	}
	vitals, err := player.vitalUpdate()
	if err != nil {
		return err
	}
	return conn.WriteFrame(vitals)
}

func worldTransform(position role.Position) worldcore.Transform {
	return worldcore.Transform{X: position.X, Y: position.Y, Z: position.Z, Orient: position.Orient}
}
