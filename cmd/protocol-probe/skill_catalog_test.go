package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/skillscript"
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
	handled, err := handleSkillCustom(conn, player, nil, nil, useSkillMessage(unlearned), "test")
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

// loadEffectsDataHost 用给定 effects_data.lua 内容装载技能脚本宿主并注入
// skill_catalog 的全局；目录缓存随装载/清理重置，避免跨测试残留。
func loadEffectsDataHost(t *testing.T, effectsDataSrc string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "effects_data.lua"), []byte(effectsDataSrc), 0644); err != nil {
		t.Fatal(err)
	}
	host := skillscript.New(dir, nil)
	if _, err := host.Load(); err != nil {
		t.Fatalf("load effects_data host: %v", err)
	}
	previous := skillScripts
	skillScripts = host
	installedCombatSkillCatalog.resetCache()
	t.Cleanup(func() {
		skillScripts = previous
		installedCombatSkillCatalog.resetCache()
	})
}

func effectsByKind(definition combatSkillDefinition, kind skillEffectKind) []combatSkillEffect {
	var out []combatSkillEffect
	for _, effect := range definition.effects {
		if effect.kind == kind {
			out = append(out, effect)
		}
	}
	return out
}

// effects_data 层：self_buffs/target_effects 整组并入编译（优先级：
// 手勘覆盖 > effects_data > INI 兜底）。
func TestEffectsDataBuffCompilation(t *testing.T) {
	loadEffectsDataHost(t, `local E = {}
E["CS_yhwq_hsqs01"] = {
    self_buffs = { { config_id = "buf_probe_a", static_data = 4242, level = 2, life_ms = 5000 } },
    target_effects = { { config_id = "buf_probe_t", static_data = 4343, level = 1, life_ms = 3000, control = true } },
    by_level = { [2] = { self_life_ms = 99000 } },
}
E["CS_yhwq_hsqs03"] = {
    damage = { amount = 100, waves = 1 },
}
return E
`)
	// 自增益 + 目标控制各一条，租约按条目值。
	definition, ok := installedCombatSkillCatalog.definition("CS_yhwq_hsqs01", 1)
	if !ok {
		t.Fatal("CS_yhwq_hsqs01 should compile")
	}
	selfs := effectsByKind(definition, skillEffectPlayerBuff)
	if len(selfs) != 1 || selfs[0].buff.configID != "buf_probe_a" ||
		selfs[0].buff.staticData != 4242 || selfs[0].buff.level != 2 ||
		selfs[0].buff.lifetime != 5*time.Second || selfs[0].buff.preferredSlot != 0 {
		t.Fatalf("self_buffs = %+v", selfs)
	}
	targets := effectsByKind(definition, skillEffectTargetControl)
	if len(targets) != 1 || targets[0].buff.configID != "buf_probe_t" ||
		targets[0].buff.lifetime != 3*time.Second {
		t.Fatalf("target_effects = %+v", targets)
	}

	// by_level[2] 的 self_life_ms 逐级覆盖租约。
	level2, ok := installedCombatSkillCatalog.definition("CS_yhwq_hsqs01", 2)
	if !ok {
		t.Skip("CS_yhwq_hsqs01 无 2 级编译（maxLevel=11 应可编译）")
	}
	selfs2 := effectsByKind(level2, skillEffectPlayerBuff)
	if len(selfs2) != 1 || selfs2[0].buff.lifetime != 99*time.Second {
		t.Fatalf("by_level lease override = %+v", selfs2)
	}

	// 条目无 buff 字段：整组取代（INI exact_buff 不再附加——hsqs03 原有
	// buf_CS_yhwq_hsqs03 的 INI 兜底被 damage-only 条目取代）。
	none, ok := installedCombatSkillCatalog.definition("CS_yhwq_hsqs03", 1)
	if !ok {
		t.Fatal("CS_yhwq_hsqs03 should compile")
	}
	if got := len(effectsByKind(none, skillEffectPlayerBuff)) + len(effectsByKind(none, skillEffectTargetControl)); got != 0 {
		t.Fatalf("damage-only 条目必须整组取代 INI 兜底 buff，实际携带 %d 条", got)
	}
}

// 钉死技能不受 effects_data 层影响：hsqs05 的槽位/红霸体覆盖逐字段保持。
func TestPinnedSkillUnaffectedByEffectsData(t *testing.T) {
	var pinnedWithout, pinnedWith combatSkillDefinition
	var ok bool
	func() {
		loadEffectsDataHost(t, `local E = {}
E["CS_yhwq_hsqs05"] = { self_buffs = { { config_id = "buf_impostor", static_data = 1, life_ms = 60000 } } }
return E
`)
		pinnedWith, ok = installedCombatSkillCatalog.definition("CS_yhwq_hsqs05", 1)
		if !ok {
			t.Fatal("CS_yhwq_hsqs05 should compile")
		}
	}()
	skillScripts = nil
	installedCombatSkillCatalog.resetCache()
	pinnedWithout, ok = installedCombatSkillCatalog.definition("CS_yhwq_hsqs05", 1)
	if !ok {
		t.Fatal("CS_yhwq_hsqs05 should compile without host")
	}
	if len(pinnedWith.effects) != len(pinnedWithout.effects) {
		t.Fatalf("pinned skill effects changed: with=%d without=%d", len(pinnedWith.effects), len(pinnedWithout.effects))
	}
	for i := range pinnedWith.effects {
		a, b := pinnedWith.effects[i], pinnedWithout.effects[i]
		if a.kind != b.kind || a.amount != b.amount || a.buff != b.buff || len(a.buffVariants) != len(b.buffVariants) {
			t.Fatalf("pinned effect[%d] changed: %+v vs %+v", i, a, b)
		}
		for v := range a.buffVariants {
			if a.buffVariants[v] != b.buffVariants[v] {
				t.Fatalf("pinned effect[%d] variant[%d] changed", i, v)
			}
		}
	}
	for _, effect := range pinnedWith.effects {
		if effect.buff.configID == "buf_impostor" {
			t.Fatal("effects_data must not inject buffs into pinned skills")
		}
	}
}

// 回退链：宿主禁用时按 INI exact_buff 编译（与接入前行为一致）。
func TestEffectsDataFallbackToINI(t *testing.T) {
	skillScripts = nil
	installedCombatSkillCatalog.resetCache()
	t.Cleanup(func() { installedCombatSkillCatalog.resetCache() })
	// CS_yhwq_hsqs02 未被钉死：INI exact_buff（buf_CS_yhwq_hsqs02 家族）
	// 决定其 buff 效果的有无——这里只需断言编译成功且不 panic，效果
	// 数与 INI 数据一致（有则为单条，无则为零）。
	definition, ok := installedCombatSkillCatalog.definition("CS_yhwq_hsqs02", 1)
	if !ok {
		t.Fatal("CS_yhwq_hsqs02 should compile")
	}
	total := len(effectsByKind(definition, skillEffectPlayerBuff)) + len(effectsByKind(definition, skillEffectTargetControl))
	if total > 1 {
		t.Fatalf("INI 兜底至多单条 buff，实际 %d 条", total)
	}
}

// 热重载换代：effects_data 快照变化 + resetCache 后重编译生效。
func TestEffectsDataHotReloadRecompiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "effects_data.lua"), []byte(`local E = {}
E["CS_yhwq_hsqs01"] = { self_buffs = { { config_id = "buf_probe_a", static_data = 4242, life_ms = 5000 } } }
return E
`), 0644); err != nil {
		t.Fatal(err)
	}
	host := skillscript.New(dir, nil)
	if _, err := host.Load(); err != nil {
		t.Fatal(err)
	}
	previous := skillScripts
	skillScripts = host
	installedCombatSkillCatalog.resetCache()
	t.Cleanup(func() {
		skillScripts = previous
		installedCombatSkillCatalog.resetCache()
	})

	definition, _ := installedCombatSkillCatalog.definition("CS_yhwq_hsqs01", 1)
	if selfs := effectsByKind(definition, skillEffectPlayerBuff); len(selfs) != 1 || selfs[0].buff.lifetime != 5*time.Second {
		t.Fatalf("before reload = %+v", selfs)
	}
	// 改表 → 宿主重载 → 清缓存（GM lua_reload 链路）→ 重编译取新租约。
	if err := os.WriteFile(filepath.Join(dir, "effects_data.lua"), []byte(`local E = {}
E["CS_yhwq_hsqs01"] = { self_buffs = { { config_id = "buf_probe_a", static_data = 4242, life_ms = 99000 } } }
return E
`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := host.Reload(); err != nil {
		t.Fatal(err)
	}
	installedCombatSkillCatalog.resetCache()
	reloaded, _ := installedCombatSkillCatalog.definition("CS_yhwq_hsqs01", 1)
	if selfs := effectsByKind(reloaded, skillEffectPlayerBuff); len(selfs) != 1 || selfs[0].buff.lifetime != 99*time.Second {
		t.Fatalf("after reload = %+v, want 99s lease", selfs)
	}
}

// 真实 effects_data 冒烟：仓内 scripts/module/skill 快照与目录编译一致
// （可编译的候选技逐个比对 buff 组的条数/类型/staticData/租约）。
func TestRealEffectsDataCatalogSmoke(t *testing.T) {
	host := skillscript.New(filepath.Join("..", "..", "scripts", "module", "skill"), nil)
	if _, err := host.Load(); err != nil {
		t.Fatalf("load real skill scripts: %v", err)
	}
	previous := skillScripts
	skillScripts = host
	installedCombatSkillCatalog.resetCache()
	t.Cleanup(func() {
		skillScripts = previous
		installedCombatSkillCatalog.resetCache()
	})

	carriers := []string{"CS_3times", "CS_base_pgd04", "CS_by_gxcj02", "CS_wd_tjq01", "CS_by_wgl06"}
	compiled := 0
	for _, id := range carriers {
		entry := host.EffectData(id)
		if entry == nil {
			t.Fatalf("%s missing in real effects_data", id)
		}
		definition, ok := installedCombatSkillCatalog.definition(id, 1)
		if !ok {
			continue // 目录跳过的技术（缺动作等）不计
		}
		compiled++
		wantSelf, wantTarget := len(entry.SelfBuffs), len(entry.TargetEffects)
		gotSelf := len(effectsByKind(definition, skillEffectPlayerBuff))
		gotTarget := len(effectsByKind(definition, skillEffectTargetControl))
		if _, pinned := verifiedSkillEffectOverlays[id]; pinned {
			continue // 钉死技走覆盖路径，不与 effects_data 对数
		}
		if gotSelf != wantSelf || gotTarget != wantTarget {
			t.Fatalf("%s buff 组不符：self %d/%d target %d/%d", id, gotSelf, wantSelf, gotTarget, wantTarget)
		}
		for i, spec := range entry.SelfBuffs {
			if buff := effectsByKind(definition, skillEffectPlayerBuff)[i].buff; buff.staticData != uint32(spec.StaticData) ||
				buff.lifetime != time.Duration(spec.LifeMS)*time.Millisecond {
				t.Fatalf("%s self_buff[%d] = %+v, want spec %+v", id, i, buff, spec)
			}
		}
		for i, spec := range entry.TargetEffects {
			if buff := effectsByKind(definition, skillEffectTargetControl)[i].buff; buff.staticData != uint32(spec.StaticData) ||
				buff.lifetime != time.Duration(spec.LifeMS)*time.Millisecond {
				t.Fatalf("%s target_effect[%d] = %+v, want spec %+v", id, i, buff, spec)
			}
		}
	}
	if compiled == 0 {
		t.Fatal("no carrier compiled; smoke tested nothing")
	}
}
