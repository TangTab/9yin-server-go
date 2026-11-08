package main

import (
	"strings"
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
	// 三脚猫五件套是基础武学原生动作族（Java CombatSkillCatalog 同口径）：
	// zhaoshi_player* 只有 _NN 连击段子节、无裸 ID 节，均按
	// missing_player_action 不入编译目录（打坐走原生 240，普攻为客户端
	// 本地连击族）——View40/已学表照发，但 211 编译路径不覆盖。
	nativeBasicSkills := map[string]bool{
		"zs_normal_02": true, "hw_normal_fd": true, "zs_default_01": true,
		"zs_suicide": true, "riding_start": true,
	}
	for _, learned := range starterSkillViews {
		if nativeBasicSkills[learned.configID] {
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
	if after.MP != before.MP || player.skillSP() != player.rageCap() || len(conn.Frames()) != 0 {
		t.Fatalf("unlearned skill mutated state before=%+v after=%+v frames=%d", before, after, len(conn.Frames()))
	}
}

// TestUniversalCooldownIsTaoluSwitchLock 钉死官方 GCD 语义（bug.txt:8
// 「同套路不在触发公共7秒冷却，不同套路会看到7秒CD转圈回复」；Java
// PlayerActor.beginSkillUse:1061-1064、2026-11-02 commit c58025f2 修过
// 同一 bug）：公共 GCD 只拦套路切换——同套路连招立即可续、出招者自身
// 复读只等个人 CD、跨套路 7s 内被拒、窗口过后放行。此前 per-team 硬闸
// 把同套路五招（共享 CoolDownTeam）连同施法者锁 7s，个人 CD<7s 的技能
// 全被压到 7s，已废弃。
func TestUniversalCooldownIsTaoluSwitchLock(t *testing.T) {
	player := newPlayerActor("tester", 0)
	now := time.Now()
	flower := combatSkills["CS_yhwq_hsqs04"]      // 个人 CD 1s，套路 CS_yhwq_hsqs
	flowerFirst := combatSkills["CS_yhwq_hsqs01"] // 个人 CD 2s，同套路
	taiJi := combatSkills["CS_wd_tjq04"]          // 个人 CD 1s，套路 CS_wd_tjq
	flower.mpCost, flowerFirst.mpCost, taiJi.mpCost = 0, 0, 0
	if flower.taoLu == taiJi.taoLu {
		t.Fatalf("fixture taolu collision %q", flower.taoLu)
	}
	if ok, reason := player.beginSkillUse(flower, 0, now, false); !ok {
		t.Fatalf("begin flower: %s", reason)
	}
	// 同套路连招：GCD 窗口内立即可续（各走各的个人 CD）。
	if ok, reason := player.beginSkillUse(flowerFirst, 0, now, false); !ok {
		t.Fatalf("same-taolu follow-up blocked by public cooldown: %s", reason)
	}
	// 出招者自身：个人 CD(1s) 尽、GCD(7s) 仍在 → 复读放行。
	if ok, reason := player.beginSkillUse(flower, 0, now.Add(1100*time.Millisecond), false); !ok {
		t.Fatalf("self re-cast after personal CD blocked: %s", reason)
	}
	// 跨套路：GCD 窗口内被拒（套路切换锁）。
	ok, reason := player.beginSkillUse(taiJi, 0, now.Add(1200*time.Millisecond), false)
	if ok || !strings.Contains(reason, "taolu switch") {
		t.Fatalf("cross-taolu cast within GCD ok=%t reason=%q, want taolu switch reject", ok, reason)
	}
	// 窗口过后（最后一次出招 now+1.1s + 7s）放行。
	if ok, reason := player.beginSkillUse(taiJi, 0, now.Add(9*time.Second), false); !ok {
		t.Fatalf("cross-taolu cast after GCD window: %s", reason)
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
