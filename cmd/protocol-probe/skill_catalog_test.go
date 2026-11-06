package main

import (
	"testing"
	"time"
)

func hasCompiledEffect(definition combatSkillDefinition, kind skillEffectKind) bool {
	for _, effect := range definition.effects {
		if effect.kind == kind {
			return true
		}
	}
	return false
}

func TestInstalledCatalogCompilesEveryLearnedCombatSkill(t *testing.T) {
	if got := len(combatSkills); got < 3000 {
		t.Fatalf("compiled installed level-one combat skills=%d, want full client catalog", got)
	}
	if got := len(installedCombatSkillCatalog.indexed); got < len(combatSkills) {
		t.Fatalf("indexed installed skills=%d, executable level-one=%d", got, len(combatSkills))
	}
	if len(yiHuaWuQueCombatSkills) < 7 ||
		len(taiJiQuanGuPuCombatSkills) < 8 ||
		len(heartBuddhaPalmCombatSkills) < 7 {
		t.Fatalf("compiled families flower=%d taiJi=%d heartBuddha=%d",
			len(yiHuaWuQueCombatSkills), len(taiJiQuanGuPuCombatSkills), len(heartBuddhaPalmCombatSkills))
	}
	for _, learned := range starterSkillViews {
		if learned.configID == "zs_default_01" {
			continue
		}
		definition, exists := combatSkills[learned.configID]
		if !exists {
			t.Errorf("learned skill %s is absent from compiled catalog", learned.configID)
			continue
		}
		if definition.staticData != learned.staticData || definition.level != learned.level {
			t.Errorf("%s compiled static=%d level=%d, want %d/%d",
				learned.configID, definition.staticData, definition.level, learned.staticData, learned.level)
		}
		if definition.actionName == "" || definition.totalActionDuration() <= 0 {
			t.Errorf("%s has no compiled action timeline", learned.configID)
		}
	}
}

func TestCatalogCompilesLearnedSkillAtAuthoritativeLevel(t *testing.T) {
	levelTwo, ok := installedCombatSkillCatalog.definition("CS_yhwq_hsqs01", 2)
	if !ok {
		t.Fatal("花神01 level 2 did not compile")
	}
	if levelTwo.level != 2 || levelTwo.mpCost != 27 || levelTwo.baseDamage != 138 {
		t.Fatalf("花神01 level-two response=%+v", levelTwo)
	}
	levelOne := combatSkills["CS_yhwq_hsqs01"]
	if levelOne.mpCost != 21 || levelOne.baseDamage != 113 {
		t.Fatalf("花神01 cached level-one response=%+v", levelOne)
	}
}

func TestInstalledButUnlearnedSkillCannotUseUniversalExecutor(t *testing.T) {
	player := newPlayerActor("tester", 0)
	var unlearned string
	for id := range combatSkills {
		if _, learned := player.learnedSkillLevel(id); !learned {
			unlearned = id
			break
		}
	}
	if unlearned == "" {
		t.Fatal("full catalog has no unlearned skill for permission test")
	}
	before := player.stateSnapshot()
	conn := &captureMessageConnection{}
	handled, err := handleSkillCustom(conn, player, nil, useSkillMessage(unlearned), "test")
	if err != nil || !handled {
		t.Fatalf("unlearned installed skill handled=%t err=%v", handled, err)
	}
	after := player.stateSnapshot()
	if after.MP != before.MP || player.skillSP() != 100 || len(conn.Frames()) != 0 {
		t.Fatalf("unlearned skill mutated state before=%+v after=%+v frames=%d", before, after, len(conn.Frames()))
	}
}

func TestUniversalCooldownsAreScopedByInstalledCooldownTeam(t *testing.T) {
	player := newPlayerActor("tester", 0)
	now := time.Now()
	flower := combatSkills["CS_yhwq_hsqs04"]
	taiJi := combatSkills["CS_wd_tjq04"]
	flower.mpCost, taiJi.mpCost = 0, 0
	if ok, reason := player.beginSkillUse(flower, 0, now); !ok {
		t.Fatalf("begin flower team %d: %s", flower.cooldownTeam, reason)
	}
	if ok, reason := player.beginSkillUse(taiJi, 0, now); !ok {
		t.Fatalf("different cooldown team %d was blocked by %d: %s", taiJi.cooldownTeam, flower.cooldownTeam, reason)
	}
}

func TestCatalogClassifiesInstalledTargetShapes(t *testing.T) {
	front := combatSkills["CS_wd_tjq01"]
	if front.targetMode != skillTargetCasterSector || front.areaRadius != 5 || front.sectorAngle != 120 || front.requiresTarget {
		t.Fatalf("太极01 target shape=%+v", front)
	}
	open := combatSkills["CS_wd_tjq08"]
	if open.targetMode != skillTargetCasterCircle || open.areaRadius != 5 || open.areaHeight != 5 || open.requiresTarget {
		t.Fatalf("开太极 target shape=%+v", open)
	}
	heartArea := combatSkills["CS_jh_xfz06"]
	if heartArea.targetMode != skillTargetSelectedArea || heartArea.areaRadius != 5 || !heartArea.requiresTarget {
		t.Fatalf("心佛掌06 target shape=%+v", heartArea)
	}
}

func TestCatalogCompilesHeartBuddhaLevelOneResources(t *testing.T) {
	first := combatSkills["CS_jh_xfz01"]
	if first.script != "SkillLock" || first.taoLu != "CS_jh_xfz" ||
		first.mpCost != 16 || first.baseDamage != 119 ||
		first.personalCD != 6*time.Second || first.publicCD != 7*time.Second ||
		first.actionName != "xfz_01" || first.actionDuration != 59*time.Second/30 {
		t.Fatalf("心佛掌01 metadata=%+v", first)
	}
}

func TestCatalogUsesTypedEffectsForVerifiedBuffAndArmorRoutes(t *testing.T) {
	flower := combatSkills["CS_yhwq_hsqs05"]
	if !hasCompiledEffect(flower, skillEffectPlayerBuff) || !hasCompiledEffect(flower, skillEffectRedArmor) {
		t.Fatalf("花神05 effects=%+v", flower.effects)
	}
	open := combatSkills["CS_wd_tjq08"]
	if !hasCompiledEffect(open, skillEffectDamage) ||
		!hasCompiledEffect(open, skillEffectTargetControl) ||
		!hasCompiledEffect(open, skillEffectRedArmor) {
		t.Fatalf("开太极 effects=%+v", open.effects)
	}
	stance := combatSkills["CS_wd_tjq04"]
	if !hasCompiledEffect(stance, skillEffectStanceCycle) {
		t.Fatalf("太极04 effects=%+v", stance.effects)
	}
}

func TestSkillBuffAllocatorUsesNegotiatedDynamicSlots(t *testing.T) {
	player := newPlayerActor("tester", 0)
	now := time.Now()
	for index := uint32(0); index < 14; index++ {
		buff := skillBuffDefinition{
			configID:   "test_dynamic_" + string(rune('a'+index)),
			staticData: 50000 + index,
			lifetime:   time.Minute,
		}
		slot, _, _, ok := player.activateSkillBuff(buff, now)
		if !ok {
			t.Fatalf("dynamic Buff %d could not allocate", index)
		}
		want := uint16(11 + index)
		if slot != want {
			t.Fatalf("dynamic Buff %d slot=%d, want %d", index, slot, want)
		}
	}
	if _, _, _, ok := player.activateSkillBuff(skillBuffDefinition{
		configID: "overflow", staticData: 60000, lifetime: time.Minute,
	}, now); ok {
		t.Fatal("allocator accepted a 15th dynamic Buff beyond BufferInfo24")
	}
}
