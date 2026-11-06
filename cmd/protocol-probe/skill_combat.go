package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

const (
	clientCustomUseSkill int32 = 211

	// Current FxGameLogic GameMessageHandler binding constructor stores
	// sub_110120E4 -> sub_129C0330 at pair 55/56, so its modern numeric
	// message is (55-15)/2 == 20. The recovered receiver reads:
	// [2] caster object, [3] skill-action ConfigID, [4] target object,
	// then enters the SkillEffect/CurSkillCanNoLimit continuation. Unlike
	// message 18, this is the skill-action route rather than generic DoAction.
	serverNativeSkillActionMessage int32 = 20

	// Installed buff_new.ini/buff_static.ini:
	// buf_CS_yhwq_hsqs04 -> StaticData 15101, visible self Buff.
	// Its level-one hidden re-trigger lease is 35 seconds (StaticData 15104).
	yiHuaSkill04BuffStaticData uint32 = 15101
	yiHuaSkill04BuffLease             = 35 * time.Second
	// buff_new.ini/buff_static.ini/buff_varprop.ini level one:
	// buf_CS_yhwq_hsqs05 -> StaticData 15105, visible, LifeTime 5000 ms.
	yiHuaSkill05BuffStaticData uint32 = 15105
	yiHuaSkill05BuffLease             = 5 * time.Second

	// Installed 太极拳(古谱) Buff routes. Skill 04 alternates its two
	// persistent visible stance records; skill 05 and 07 use their exact
	// level-one buff_varprop lifetimes.
	taiJiDefenceStanceBuffStaticData uint32 = 6013
	taiJiAttackStanceBuffStaticData  uint32 = 6055
	taiJiSkill05BuffStaticData       uint32 = 6043
	taiJiSkill07BuffStaticData       uint32 = 6050
	taiJiStanceBuffLease                    = 24 * time.Hour
	taiJiSkill05BuffLease                   = 15 * time.Second
	taiJiSkill07BuffLease                   = 8 * time.Second
)

type skillActionSegment struct {
	action   string
	at       time.Duration
	duration time.Duration
}

type skillTargetMode uint8

const (
	skillTargetSelf skillTargetMode = iota
	skillTargetSelected
	skillTargetCasterCircle
	skillTargetCasterSector
	skillTargetCasterRectangle
	skillTargetSelectedArea
)

type skillEffectKind uint8

const (
	skillEffectDamage skillEffectKind = iota
	skillEffectPlayerBuff
	skillEffectTargetControl
	skillEffectRedArmor
	skillEffectStanceCycle
)

type skillBuffDefinition struct {
	configID      string
	staticData    uint32
	level         int32
	lifetime      time.Duration
	preferredSlot uint16
}

type combatSkillEffect struct {
	kind         skillEffectKind
	amount       int32
	buff         skillBuffDefinition
	buffVariants []skillBuffDefinition
}

type combatSkillDefinition struct {
	id               string
	staticData       int32
	script           string
	taoLu            string
	level            int32
	cooldownCategory int32
	cooldownTeam     int32
	personalCD       time.Duration
	publicCD         time.Duration
	mpCost           int32
	spCost           int32
	qgCost           int32
	baseDamage       int32
	requiresTarget   bool
	targetMode       skillTargetMode
	areaRadius       float32
	areaHeight       float32
	sectorAngle      float32
	areaWidth        float32
	actionName       string
	followupActions  []skillActionSegment
	actionDuration   time.Duration
	effects          []combatSkillEffect
}

func (definition combatSkillDefinition) totalActionDuration() time.Duration {
	duration := definition.actionDuration
	for _, segment := range definition.followupActions {
		if segment.at+segment.duration > duration {
			duration = segment.at + segment.duration
		}
	}
	return duration
}

func skillCooldownFrame(definition combatSkillDefinition, now time.Time) ([]byte, error) {
	start := now.UnixMilli()
	end := start + definition.personalCD.Milliseconds()
	return serverRecordAddCells(playerObjectID, playerOwnerID, recordCooldown, []recordCell{
		recordInt(uint32(definition.cooldownCategory)),
		recordInt64(start),
		recordInt64(end),
		recordInt(uint32(definition.cooldownTeam)),
	})
}

func nativeSkillActionFrame(skillID string, targetID, targetOwnerID uint32) ([]byte, error) {
	if skillID == "" {
		return nil, fmt.Errorf("native skill action ID is empty")
	}
	return serverModernCustomIntMessage(
		serverNativeSkillActionMessage,
		customObject(playerObjectID, playerOwnerID),
		customString(skillID),
		customObject(targetID, targetOwnerID),
		// The protected receiver accepts both optional fields. Supplying the
		// complete modern form makes the per-row show runner preserve actor
		// translation and action timing instead of relying on omitted defaults.
		customInt(1),
		customFloat32(1),
	)
}

// skillActionPreparationFrame replays the two native player-action properties
// immediately before ID20. The client registers a real on_actionset callback:
// it enters the fight action service and rebuilds the action prefix/current
// action state synchronously. PlayerAddObject already carries the same values,
// but that initial archive can arrive before the asynchronous role model/action
// component is ready; an explicit object-property refresh after ClientReady
// closes that ordering gap without changing the selected appearance.
func skillActionPreparationFrame() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 18, Name: "ActionSet", Value: clientdata.StringValue("0h")},
		{Index: 19, Name: "State", Value: clientdata.StringValue("stand")},
	})
}

// playerActionStateFrame drives the ordinary player State property callback.
// This is intentionally separate from skill_action: installed life-skill Lua
// uses the same action names (for example interact268 for fishing) against the
// player's action component, so it is a useful standalone reachability probe.
func playerActionStateFrame(action string) ([]byte, error) {
	if action == "" {
		return nil, fmt.Errorf("player action state is empty")
	}
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: 19, Name: "State", Value: clientdata.StringValue(action)},
	})
}

// nativeGenericActionFrame reaches the current client's generic "action"
// command receiver (sub_129C0E90). Its native implementation reads the target
// from TVarList[2], the action from [3], optionally clears the current action,
// and calls the action service. The registration-table slot is not a numeric
// wire ID: the leading TVarList value must be the string command name.
func nativeGenericActionFrame(action string) ([]byte, error) {
	if action == "" {
		return nil, fmt.Errorf("native generic action is empty")
	}
	return serverModernCustomStringMessage(
		"action",
		customObject(playerObjectID, playerOwnerID),
		customString(action),
		customString("1"),
	)
}

func parseUseSkillCustom(custom clientCustomMessage) (combatSkillDefinition, bool) {
	if len(custom.Values) < 2 || custom.Values[0].Type != 2 ||
		custom.Values[0].Int32 != clientCustomUseSkill ||
		custom.Values[1].Type != 6 || custom.Values[1].Text == "" {
		return combatSkillDefinition{}, false
	}
	definition, exists := combatSkills[custom.Values[1].Text]
	return definition, exists
}

func useSkillCastTransform(custom clientCustomMessage) (worldcore.Transform, bool) {
	if len(custom.Values) < 5 ||
		custom.Values[2].Type != 4 ||
		custom.Values[3].Type != 4 ||
		custom.Values[4].Type != 4 {
		return worldcore.Transform{}, false
	}
	transform := worldcore.Transform{
		X: custom.Values[2].Float32,
		Y: custom.Values[3].Float32,
		Z: custom.Values[4].Float32,
	}
	// Modern C2S 211 continues with caster orientation before the optional
	// target position. Sector skills need this value for their facing test.
	if len(custom.Values) > 5 && custom.Values[5].Type == 4 {
		transform.Orient = custom.Values[5].Float32
	}
	return transform, true
}

func (p *playerActor) beginSkillUse(definition combatSkillDefinition, target uint64, now time.Time) (bool, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.skillLockUntil.After(now) {
		return false, fmt.Sprintf("skill locked remaining=%s", p.skillLockUntil.Sub(now).Round(time.Millisecond))
	}
	if ready := p.skillCooldowns[definition.id]; ready.After(now) {
		return false, fmt.Sprintf("personal cooldown remaining=%s", ready.Sub(now).Round(time.Millisecond))
	}
	teamKey := definition.cooldownTeam
	if teamKey == 0 {
		teamKey = -definition.cooldownCategory
	}
	if ready := p.skillTeamCooldowns[teamKey]; ready.After(now) {
		return false, fmt.Sprintf("public cooldown team=%d remaining=%s", teamKey, ready.Sub(now).Round(time.Millisecond))
	}
	state := p.actor.Snapshot()
	if state.LogicState == entity.LogicStateDied || state.HP <= 0 {
		return false, "player is dead"
	}
	if state.MP < definition.mpCost {
		return false, fmt.Sprintf("MP %d < %d", state.MP, definition.mpCost)
	}
	if p.progress.attributes.sp < definition.spCost {
		return false, fmt.Sprintf("SP %d < %d", p.progress.attributes.sp, definition.spCost)
	}
	if definition.qgCost > 0 {
		if cost, ok := p.actor.UseQingGong(definition.qgCost, now); !ok {
			return false, fmt.Sprintf("QingGongPoint cannot pay %d (resolved cost %d)", definition.qgCost, cost)
		}
	}
	if !p.actor.SpendMP(definition.mpCost) {
		return false, "MP spend rejected"
	}
	p.progress.attributes.sp -= definition.spCost
	p.skillCooldowns[definition.id] = now.Add(definition.personalCD)
	p.skillTeamCooldowns[teamKey] = now.Add(definition.publicCD)
	p.currentSkillID = definition.id
	p.currentSkillEffectID = definition.id
	p.currentSkillLevel = definition.level
	p.currentSkillTarget = target
	p.actor.SetLogicState(entity.LogicStateFighting)
	return true, ""
}

func (p *playerActor) skillSP() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.progress.attributes.sp
}

func (s *sceneLifecycle) selectedCombatTarget() (uint32, uint32, bool) {
	if s == nil {
		return 0, 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := worldcore.EntityID(s.selectedObject)
	actor := s.combatActors[id]
	if actor == nil || actor.Snapshot().HP <= 0 {
		return 0, 0, false
	}
	meta, exists := s.entities[id]
	if !exists || meta.ownerID != s.selectedOwner || meta.interaction != npcInteractionCombat {
		return 0, 0, false
	}
	return uint32(id), meta.ownerID, true
}

func (s *sceneLifecycle) applySelectedSkillDamage(amount int32, now time.Time) ([]byte, uint32, int32, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := worldcore.EntityID(s.selectedObject)
	actor := s.combatActors[id]
	meta, exists := s.entities[id]
	if actor == nil || !exists || meta.ownerID != s.selectedOwner || meta.interaction != npcInteractionCombat || actor.Snapshot().HP <= 0 {
		return nil, 0, 0, false, nil
	}
	if !actor.ApplyDamage(amount) {
		return nil, uint32(id), 0, false, nil
	}
	state := actor.Snapshot()
	combat, isCombat := s.combatStates[id]
	if isCombat {
		combat.threat += amount
		if state.HP <= 0 {
			combat.threat = 0
			combat.respawnAt = now.Add(combat.respawnAfter)
		}
		s.combatStates[id] = combat
	}
	frame, err := sceneObjectProperties(uint32(id), meta.ownerID, 0, npcCombatProperties(state))
	loot := int32(0)
	if state.HP <= 0 {
		loot = state.MaxHP / 100
		if loot < 1 {
			loot = 1
		}
		if loot > 99 {
			loot = 99
		}
	}
	return frame, uint32(id), loot, state.HP <= 0, err
}

type areaSkillDamageResult struct {
	frame  []byte
	id     uint32
	loot   int32
	killed bool
}

func (s *sceneLifecycle) applyAreaSkillDamage(center worldcore.Transform, radius, height float32, amount int32, now time.Time) ([]areaSkillDamageResult, error) {
	return s.applyShapedSkillDamage(center, radius, height, 0, 0, amount, now)
}

func (s *sceneLifecycle) applyShapedSkillDamage(center worldcore.Transform, radius, height, sectorAngle, rectangleWidth float32, amount int32, now time.Time) ([]areaSkillDamageResult, error) {
	if s == nil || radius <= 0 || amount <= 0 {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	radiusSquared := radius * radius
	ids := make([]int, 0, len(s.combatActors))
	for id, actor := range s.combatActors {
		meta, exists := s.entities[id]
		if actor == nil || !exists || meta.interaction != npcInteractionCombat || actor.Snapshot().HP <= 0 {
			continue
		}
		dx := meta.transform.X - center.X
		dz := meta.transform.Z - center.Z
		dy := meta.transform.Y - center.Y
		if dy < 0 {
			dy = -dy
		}
		if height > 0 && dy > height {
			continue
		}
		if rectangleWidth > 0 {
			facingX, facingZ := math.Cos(float64(center.Orient)), math.Sin(float64(center.Orient))
			forward := float64(dx)*facingX + float64(dz)*facingZ
			side := -float64(dx)*facingZ + float64(dz)*facingX
			if forward < 0 || forward > float64(radius) || math.Abs(side) > float64(rectangleWidth)/2 {
				continue
			}
		} else if dx*dx+dz*dz > radiusSquared {
			continue
		} else if sectorAngle > 0 {
			targetAngle := math.Atan2(float64(dz), float64(dx))
			delta := math.Atan2(math.Sin(targetAngle-float64(center.Orient)), math.Cos(targetAngle-float64(center.Orient)))
			if math.Abs(delta) > float64(sectorAngle)*math.Pi/360 {
				continue
			}
		}
		{
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)

	results := make([]areaSkillDamageResult, 0, len(ids))
	for _, rawID := range ids {
		id := worldcore.EntityID(rawID)
		actor := s.combatActors[id]
		meta := s.entities[id]
		if !actor.ApplyDamage(amount) {
			continue
		}
		state := actor.Snapshot()
		if combat, exists := s.combatStates[id]; exists {
			combat.threat += amount
			if state.HP <= 0 {
				combat.threat = 0
				combat.respawnAt = now.Add(combat.respawnAfter)
			}
			s.combatStates[id] = combat
		}
		frame, err := sceneObjectProperties(uint32(id), meta.ownerID, 0, npcCombatProperties(state))
		if err != nil {
			return nil, err
		}
		loot := int32(0)
		if state.HP <= 0 {
			loot = state.MaxHP / 100
			if loot < 1 {
				loot = 1
			}
			if loot > 99 {
				loot = 99
			}
		}
		results = append(results, areaSkillDamageResult{
			frame: frame, id: uint32(id), loot: loot, killed: state.HP <= 0,
		})
	}
	return results, nil
}

func (s *sceneLifecycle) selectedCombatTargetTransform() (worldcore.Transform, bool) {
	if s == nil {
		return worldcore.Transform{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := worldcore.EntityID(s.selectedObject)
	meta, exists := s.entities[id]
	actor := s.combatActors[id]
	return meta.transform, exists && actor != nil && actor.Snapshot().HP > 0
}

func (s *sceneLifecycle) applyNPCControl(ids []uint32, lifetime time.Duration, now time.Time) {
	if s == nil || lifetime <= 0 || len(ids) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	until := now.Add(lifetime)
	for _, rawID := range ids {
		id := worldcore.EntityID(rawID)
		state, exists := s.combatStates[id]
		if !exists {
			continue
		}
		if until.After(state.controlledUntil) {
			state.controlledUntil = until
			s.combatStates[id] = state
		}
	}
}

type appliedSkillBuff struct {
	slot       uint16
	staticData uint32
	expires    time.Time
}

func stagePlayerSkillEffects(player *playerActor, definition combatSkillDefinition, now time.Time, remote string) []appliedSkillBuff {
	applied := make([]appliedSkillBuff, 0, 3)
	for _, effect := range definition.effects {
		var buff skillBuffDefinition
		switch effect.kind {
		case skillEffectPlayerBuff:
			buff = effect.buff
		case skillEffectStanceCycle:
			var ok bool
			buff, ok = player.nextSkillBuffVariant(effect.buffVariants, now)
			if !ok {
				continue
			}
		case skillEffectRedArmor:
			buff = skillBuffDefinition{
				configID: "buf_fixed_red", staticData: redSuperArmorStaticData,
				lifetime: definition.totalActionDuration(), preferredSlot: redSuperArmorBuffSlot,
			}
		default:
			continue
		}
		slot, state, expires, ok := player.activateSkillBuff(buff, now)
		if !ok {
			log.Printf("%s: no native BuffInfo slot available skill=%s buff=%s", remote, definition.id, buff.configID)
			continue
		}
		applied = append(applied, appliedSkillBuff{slot: slot, staticData: buff.staticData, expires: expires})
		log.Printf("%s: staged compiled skill Buff skill=%s buff=%s property=%d value=%q expiry=%d",
			remote, definition.id, buff.configID, bufferInfoPropertyIndex(slot), state, expires.UnixMilli())
	}
	return applied
}

func handleSkillCustom(link sceneMessageConnection, player *playerActor, world *sceneLifecycle, custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomUseSkill {
		return false, nil
	}
	definition, valid := parseUseSkillCustom(custom)
	if !valid {
		log.Printf("%s: reject malformed or unlearned USE_SKILL values=%v", remote, custom.Values)
		return true, nil
	}
	if player == nil {
		log.Printf("%s: reject skill before player spawn id=%s", remote, definition.id)
		return true, nil
	}
	level, learned := player.learnedSkillLevel(definition.id)
	if !learned {
		log.Printf("%s: reject unlearned installed skill id=%s", remote, definition.id)
		return true, nil
	}
	resolved, executable := installedCombatSkillCatalog.definition(definition.id, level)
	if !executable {
		log.Printf("%s: reject learned skill without compiled level response id=%s level=%d", remote, definition.id, level)
		return true, nil
	}
	definition = resolved
	targetID := playerObjectID
	targetOwnerID := playerOwnerID
	if definition.requiresTarget {
		var ok bool
		if targetID, targetOwnerID, ok = world.selectedCombatTarget(); !ok {
			log.Printf("%s: reject skill id=%s without living selected target", remote, definition.id)
			return true, nil
		}
	}
	now := time.Now()
	target := uint64(targetID) | uint64(targetOwnerID)<<32
	if ok, reason := player.beginSkillUse(definition, target, now); !ok {
		log.Printf("%s: reject skill id=%s: %s", remote, definition.id, reason)
		return true, nil
	}
	appliedBuffs := stagePlayerSkillEffects(player, definition, now, remote)
	cooldownFrame, err := skillCooldownFrame(definition, now)
	if err != nil {
		return true, fmt.Errorf("encode skill cooldown %s: %w", definition.id, err)
	}
	if err := link.WriteFrame(cooldownFrame); err != nil {
		return true, err
	}
	playerFrame, err := player.vitalUpdate()
	if err != nil {
		return true, err
	}
	if err := link.WriteFrame(playerFrame); err != nil {
		return true, err
	}
	if definition.actionName != "" {
		actionFrame, err := playerActionStateFrame(definition.actionName)
		if err != nil {
			return true, fmt.Errorf("encode skill State action %s (%s): %w", definition.id, definition.actionName, err)
		}
		if err := link.WriteFrame(actionFrame); err != nil {
			return true, err
		}
		if len(definition.followupActions) > 0 {
			scheduleSkillFollowupActions(link, definition)
		}
	}
	log.Printf("%s: started verified State skill action skill=%s action=%s followups=%d total=%s caster=%d-%d target=%d-%d",
		remote, definition.id, definition.actionName, len(definition.followupActions), definition.totalActionDuration(),
		playerObjectID, playerOwnerID, targetID, targetOwnerID)
	damageTargetID := uint32(0)
	damageTargetCount := 0
	hitIDs := make([]uint32, 0, 8)
	var totalLoot int32
	for _, effect := range definition.effects {
		if effect.kind != skillEffectDamage || effect.amount <= 0 || world == nil {
			continue
		}
		switch definition.targetMode {
		case skillTargetSelected:
			targetFrame, id, loot, killed, err := world.applySelectedSkillDamage(effect.amount, now)
			if err != nil {
				return true, err
			}
			if targetFrame != nil {
				damageTargetID, damageTargetCount = id, damageTargetCount+1
				hitIDs = append(hitIDs, id)
				if err := link.WriteFrame(targetFrame); err != nil {
					return true, err
				}
			}
			if killed {
				totalLoot += loot
			}
		case skillTargetCasterCircle, skillTargetCasterSector, skillTargetCasterRectangle, skillTargetSelectedArea:
			center, ok := useSkillCastTransform(custom)
			if definition.targetMode == skillTargetSelectedArea {
				center, ok = world.selectedCombatTargetTransform()
			}
			if !ok {
				log.Printf("%s: compiled area skill id=%s has no center; no targets settled", remote, definition.id)
				continue
			}
			angle := float32(0)
			width := float32(0)
			if definition.targetMode == skillTargetCasterSector {
				angle = definition.sectorAngle
			} else if definition.targetMode == skillTargetCasterRectangle {
				width = definition.areaWidth
			}
			results, err := world.applyShapedSkillDamage(center, definition.areaRadius, definition.areaHeight, angle, width, effect.amount, now)
			if err != nil {
				return true, err
			}
			for _, result := range results {
				if damageTargetID == 0 {
					damageTargetID = result.id
				}
				damageTargetCount++
				hitIDs = append(hitIDs, result.id)
				if err := link.WriteFrame(result.frame); err != nil {
					return true, err
				}
				if result.killed {
					totalLoot += result.loot
				}
			}
			log.Printf("%s: settled compiled area skill id=%s mode=%d center=(%.2f,%.2f,%.2f) radius=%.1f angle=%.1f width=%.1f hits=%d damage_each=%d",
				remote, definition.id, definition.targetMode, center.X, center.Y, center.Z,
				definition.areaRadius, definition.sectorAngle, definition.areaWidth, len(results), effect.amount)
		}
	}
	for _, effect := range definition.effects {
		if effect.kind == skillEffectTargetControl {
			world.applyNPCControl(hitIDs, effect.buff.lifetime, now)
			log.Printf("%s: settled compiled target control skill=%s buff=%s targets=%d lifetime=%s",
				remote, definition.id, effect.buff.configID, len(hitIDs), effect.buff.lifetime)
		}
	}
	if totalLoot > 0 {
		player.addSilver(totalLoot)
		silverFrame, err := player.silverUpdate()
		if err != nil {
			return true, err
		}
		if err := link.WriteFrame(silverFrame); err != nil {
			return true, err
		}
		log.Printf("%s: skill defeated NPCs targets=%d silver_drop=%d", remote, damageTargetCount, totalLoot)
	}
	for _, buff := range appliedBuffs {
		schedulePlayerBuffExpiry(link, player, buff.slot, buff.staticData, buff.expires)
	}
	scheduleCurrentSkillClear(link, player, definition)
	state := player.actor.Snapshot()
	log.Printf("%s: accepted skill id=%s mp=%d sp=%d qg=%d damage=%d target=%d targets=%d personal_cd=%s public_cd=%s",
		remote, definition.id, state.MP, player.skillSP(), state.QingGongPoint, definition.baseDamage, damageTargetID, damageTargetCount, definition.personalCD, definition.publicCD)
	return true, nil
}

func scheduleSkillFollowupActions(conn sceneMessageConnection, definition combatSkillDefinition) {
	for _, scheduled := range definition.followupActions {
		segment := scheduled
		delay := segment.at
		if delay < 0 {
			delay = 0
		}
		time.AfterFunc(delay, func() {
			frame, err := playerActionStateFrame(segment.action)
			if err != nil {
				log.Printf("skill follow-up State encode id=%s action=%s: %v", definition.id, segment.action, err)
				return
			}
			if err := conn.WriteFrame(frame); err != nil {
				log.Printf("skill follow-up State write id=%s action=%s: %v", definition.id, segment.action, err)
				return
			}
			log.Printf("skill follow-up State sent id=%s action=%s at=%s", definition.id, segment.action, delay)
		})
	}
}

func scheduleCurrentSkillClear(conn sceneMessageConnection, player *playerActor, definition combatSkillDefinition) {
	delay := definition.totalActionDuration()
	if delay <= 0 {
		delay = time.Millisecond
	}
	time.AfterFunc(delay, func() {
		if !player.clearCurrentSkill(definition.id) {
			return
		}
		update, err := player.currentSkillClearUpdate()
		if err != nil {
			log.Printf("skill state clear encode id=%s: %v", definition.id, err)
			return
		}
		if err := conn.WriteFrame(update); err != nil {
			log.Printf("skill state clear write id=%s: %v", definition.id, err)
			return
		}
		stand, err := playerActionStateFrame("stand")
		if err != nil {
			log.Printf("skill stand restore encode id=%s: %v", definition.id, err)
			return
		}
		if err := conn.WriteFrame(stand); err != nil {
			log.Printf("skill stand restore write id=%s: %v", definition.id, err)
			return
		}
		log.Printf("skill state cleared and stand restored id=%s total_action=%s", definition.id, delay)
	})
}
