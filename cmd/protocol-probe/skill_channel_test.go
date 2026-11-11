package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/combatcalc"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// 通道技能（怒招引导型）波次引擎锚点——挑战.pcapng 2026-11-12 寒心恨雪
// 官服定版（docs/寒心恨雪与太极剑舞引导机制.md §二）：
//  1. 逐波单目标：每波恰 1 个受害者（非 AOE）；
//  2. 每波 SP−10（先扣）+ 命中 +2 回怒（writeSkillSettleFrames）；
//  3. SP 不足扣费即吞波（静默收束，零帧）；
//  4. 无活目标即停（首波击杀后不再补帧）；
//  5. c1=71 每波一帧（帧数=跳数）、c1=10 段序自第 2 波起（v=2/3/..）；
//  6. m914 窗口内更新通道位置并回 0x21 成对停帧，窗口外静默吞。
func channelTestSetup(t *testing.T, sp int32) (*captureMessageConnection, *sceneLifecycle, *playerActor) {
	t.Helper()
	originalEngine := installedCombatCalc
	installedCombatCalc = newTestCombatEngine(t, combatcalc.FormulaSpec{
		ID: 9, Formula: "R[P1,P2]", Src: map[int]int{1: 5, 2: 5},
	})
	t.Cleanup(func() { installedCombatCalc = originalEngine })
	originalRandom := combatRandomFloat64
	combatRandomFloat64 = func() float64 { return 0.5 }
	t.Cleanup(func() { combatRandomFloat64 = originalRandom })

	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 95)
	placeTestNPC(t, world, 95, 4, 0)
	player := newPlayerActor("tester", 0)
	player.mu.Lock()
	player.progress.attributes.sp = sp
	player.mu.Unlock()
	return conn, world, player
}

func channelTestDefinition(waves int, spPerWave int32) combatSkillDefinition {
	return combatSkillDefinition{
		id: "CS_jh_hyjf08", script: "SkillNormal", actionName: "qzj_07_1",
		damageRows: []skillDamageRow{{formulaID: 9, damageType: "PhyDamage", params: []string{"10", "20"}}},
		channel: &channelSkillSpec{
			waves: waves, firstWave: 5 * time.Millisecond, interval: 12 * time.Millisecond, spPerWave: spPerWave,
		},
	}
}

func channelTestCast(origin worldcore.Transform) clientCustomMessage {
	return clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 211},
		{Type: 6, Text: "CS_jh_hyjf08"},
		{Type: 4, Float32: origin.X},
		{Type: 4, Float32: origin.Y},
		{Type: 4, Float32: origin.Z},
	}}
}

// channelFrameStats 统计已发帧里的 msg24 族形态。
func channelFrameStats(t *testing.T, conn *captureMessageConnection) (effect71, combo10 int, comboValues []int32) {
	t.Helper()
	for _, frame := range conn.Frames() {
		if len(frame) == 0 || frame[0] != 0x1E {
			continue
		}
		values, err := parseClientCustomMessage(frame)
		if err != nil || len(values.Values) < 4 || values.Values[0].Type != 2 {
			continue
		}
		if values.Values[0].Int32 != 24 {
			continue
		}
		switch values.Values[2].Int32 {
		case 71:
			effect71++
		case 10:
			combo10++
			comboValues = append(comboValues, values.Values[3].Int32)
		}
	}
	return effect71, combo10, comboValues
}

func TestChannelWavesDrainSPAndEmitPerWaveFrames(t *testing.T) {
	conn, world, player := channelTestSetup(t, 50)
	meta, ok := world.npcMeta(worldcore.EntityID(forkNPCID(95)))
	if !ok {
		t.Fatal("npc 95 meta missing")
	}
	launchChannelWaves(conn, world, channelTestDefinition(3, 10), player,
		channelTestCast(worldcore.Transform{X: 0, Y: 0, Z: 0}), forkNPCID(95), meta.ownerID, "test")
	time.Sleep(80 * time.Millisecond)

	// 伤害 3×15（R[10,20]@0.5）；SP 50 → 每波 −10+2 → 26。
	//（起始 SP 须低于 rageCap——测试角色带初始内功档，超过会被夹。）
	vitals, ok := world.run.combat.Vitals(worldcore.EntityID(forkNPCID(95)))
	if !ok {
		t.Fatal("NPC 95 missing")
	}
	if got, want := vitals.HP, int32(1000-45); got != want {
		t.Fatalf("NPC HP=%d, want %d (3 waves × 15)", got, want)
	}
	if got := player.skillSP(); got != 26 {
		t.Fatalf("SP=%d, want 26 (50 − 3×10 + 3×2)", got)
	}
	effect71, combo10, comboValues := channelFrameStats(t, conn)
	if effect71 != 3 {
		t.Fatalf("c1=71 frames=%d, want 3 (每波一帧)", effect71)
	}
	if combo10 != 2 || len(comboValues) != 2 || comboValues[0] != 2 || comboValues[1] != 3 {
		t.Fatalf("c1=10 段序帧=%d 值=%v, want v=2,3（首波不发）", combo10, comboValues)
	}
	// 官服波簇方言（挑战.pcapng 波2 全帧序）：m169 挂/摘对、m18 受击、
	// Cant* 翻转、BufferInfo 槽 CSV——动画触发链。
	buffApply, buffRemove, hurt18 := 0, 0, 0
	for _, frame := range conn.Frames() {
		if len(frame) == 0 || frame[0] != 0x1E {
			continue
		}
		values, err := parseClientCustomMessage(frame)
		if err != nil || len(values.Values) < 2 || values.Values[0].Type != 2 {
			continue
		}
		switch values.Values[0].Int32 {
		case 18:
			hurt18++
		case 169:
			if len(values.Values) >= 6 && values.Values[2].Type == 6 {
				switch values.Values[1].Int32 {
				case 1:
					buffApply++
				case 0:
					buffRemove++
				}
			}
		}
	}
	if buffApply != 6 || buffRemove != 6 {
		t.Fatalf("m169 apply=%d remove=%d, want 6/6 (每波 victim+self 挂→摘)", buffApply, buffRemove)
	}
	if hurt18 != 3 {
		t.Fatalf("m18 hurt frames=%d, want 3 (每波一帧)", hurt18)
	}
	// c1=71 第 11 值=当波动作名（首波空、波2=qzj_07_2、波3=qzj_07_3）——
	// 真机三轮「无动画」根因修复的锚点。
	waveActions := []string{}
	for _, frame := range conn.Frames() {
		if len(frame) == 0 || frame[0] != 0x1E {
			continue
		}
		values, err := parseClientCustomMessage(frame)
		if err != nil || len(values.Values) < 4 || values.Values[0].Type != 2 {
			continue
		}
		if values.Values[0].Int32 == 24 && values.Values[2].Int32 == 71 && len(values.Values) >= 11 {
			waveActions = append(waveActions, values.Values[10].Text)
		}
	}
	if len(waveActions) != 3 || waveActions[0] != "" || waveActions[1] != "qzj_07_2" || waveActions[2] != "qzj_07_3" {
		t.Fatalf("c1=71 动作名序列=%v, want ['', qzj_07_2, qzj_07_3]", waveActions)
	}
}

func TestChannelStopsWhenSPCannotPayWave(t *testing.T) {
	conn, world, player := channelTestSetup(t, 25)
	meta, _ := world.npcMeta(worldcore.EntityID(forkNPCID(95)))
	launchChannelWaves(conn, world, channelTestDefinition(4, 10), player,
		channelTestCast(worldcore.Transform{X: 0, Y: 0, Z: 0}), forkNPCID(95), meta.ownerID, "test")
	time.Sleep(100 * time.Millisecond)

	// 波1: 25≥10 → 15+2=17；波2: 17≥10 → 7+2=9；波3: 9<10 吞波（静默）。
	effect71, _, _ := channelFrameStats(t, conn)
	if effect71 != 2 {
		t.Fatalf("c1=71 frames=%d, want 2 (SP 只付得起 2 波)", effect71)
	}
	if got := player.skillSP(); got != 9 {
		t.Fatalf("SP=%d, want 9", got)
	}
}

func TestChannelStopsWhenNoLivingTarget(t *testing.T) {
	conn, world, player := channelTestSetup(t, 50)
	npc := worldcore.EntityID(forkNPCID(95))
	meta, _ := world.npcMeta(npc)
	world.run.mu.Lock()
	if vitals, ok := world.run.combat.Vitals(npc); ok {
		vitals.HP = 15 // 首波 15 伤害恰好击杀（Vitals 值类型，须写回）
		world.run.combat.WriteVitals(npc, vitals)
	}
	world.run.mu.Unlock()
	launchChannelWaves(conn, world, channelTestDefinition(3, 10), player,
		channelTestCast(worldcore.Transform{X: 0, Y: 0, Z: 0}), forkNPCID(95), meta.ownerID, "test")
	time.Sleep(80 * time.Millisecond)

	// 官服 1 跳中断形态：首波击杀（c1=29 同毫秒）→ 无活目标即停。
	effect71, _, _ := channelFrameStats(t, conn)
	if effect71 != 1 {
		t.Fatalf("c1=71 frames=%d, want 1 (目标全灭即停)", effect71)
	}
}

// TestChannelRetargetsRandomlyEachWave 双目标随机换靶（用户 2026-11-12
// 口径）：首波=211 锁定目标，后续每波随机换靶（两次施放的伤害分布到
// 两个 NPC 上，单人吃满全波=不换靶才算失败）。
func TestChannelRetargetsRandomlyEachWave(t *testing.T) {
	conn, world, player := channelTestSetup(t, 80) // 8 波×净−8 → 终值 16
	registerTestCombatNPC(t, world, 96)
	placeTestNPC(t, world, 96, -4, 0) // 第二个 NPC，与 95 分列通道原点两侧
	npcA, npcB := worldcore.EntityID(forkNPCID(95)), worldcore.EntityID(forkNPCID(96))
	meta, ok := world.npcMeta(npcA)
	if !ok {
		t.Fatal("npc 95 meta missing")
	}
	launchChannelWaves(conn, world, channelTestDefinition(8, 10), player,
		channelTestCast(worldcore.Transform{X: 0, Y: 0, Z: 0}), forkNPCID(95), meta.ownerID, "test")
	time.Sleep(160 * time.Millisecond)

	vitalsA, _ := world.run.combat.Vitals(npcA)
	vitalsB, _ := world.run.combat.Vitals(npcB)
	damageA, damageB := 1000-vitalsA.HP, 1000-vitalsB.HP
	if damageA+damageB != 8*15 {
		t.Fatalf("total damage=%d+%d, want 8 waves × 15", damageA, damageB)
	}
	if damageB == 0 {
		t.Fatal("8 波全打 NPC95：随机换靶未生效（每波应随机选附近敌人）")
	}
	if damageA == 0 {
		t.Fatal("首波锁定目标 NPC95 未吃到伤害")
	}
}

func TestChannelSyncCustomEchoesInsideWindowOnly(t *testing.T) {
	conn, _, player := channelTestSetup(t, 100)
	spec := &channelSkillSpec{waves: 3, firstWave: 5 * time.Millisecond, interval: 12 * time.Millisecond, spPerWave: 10}
	player.setChannelWindow(time.Now(), spec, worldcore.Transform{}, false)

	sync := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomChannelSync},
		{Type: 4, Float32: 581.6}, {Type: 4, Float32: 38.6}, {Type: 4, Float32: 282.8}, {Type: 4, Float32: -1.17},
	}}
	handled, err := handleChannelSyncCustom(conn, player, sync, "test")
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v, want true nil", handled, err)
	}
	echoes := 0
	for _, frame := range conn.Frames() {
		if len(frame) > 0 && frame[0] == 0x21 {
			echoes++
		}
	}
	if echoes != 2 {
		t.Fatalf("0x21 echo frames=%d, want 2 (成对停帧)", echoes)
	}
	origin := player.channelOriginSnapshot(worldcore.Transform{})
	if origin.X != 581.6 || origin.Orient != -1.17 {
		t.Fatalf("channel transform = %+v, want m914 上报值", origin)
	}

	// 窗口过期：静默吞、无回显。
	player.mu.Lock()
	player.channelUntil = time.Now().Add(-time.Second)
	player.mu.Unlock()
	conn.resetFrames()
	if handled, err = handleChannelSyncCustom(conn, player, sync, "test"); !handled || err != nil {
		t.Fatalf("handled=%v err=%v, want true nil", handled, err)
	}
	for _, frame := range conn.Frames() {
		if len(frame) > 0 && frame[0] == 0x21 {
			t.Fatal("窗口外 m914 不应回显 0x21")
		}
	}
}

// TestChannelSpecsWiredToCatalog 真树冒烟：hyjf08/tqjf08 定义带通道参数
// 且预扣清零（资源树不可用时跳过）。
func TestChannelSpecsWiredToCatalog(t *testing.T) {
	if _, err := openResourceReader(filepath.Join(modernSkillRoot, "skill_new.ini")); err != nil {
		t.Skipf("resource store unavailable: %v", err)
	}
	for _, id := range []string{"CS_jh_hyjf08", "CS_jh_tqjf08"} {
		if _, ok := channelSkillSpecs[id]; !ok {
			t.Fatalf("%s missing from channelSkillSpecs", id)
		}
		definition, ok := installedCombatSkillCatalog.definition(id, 1)
		if !ok {
			t.Skipf("%s not in catalog at level 1", id)
		}
		if definition.channel == nil {
			t.Fatalf("%s compiled without channel spec", id)
		}
		if definition.channel.waves != 8 || definition.channel.spPerWave != 10 {
			t.Fatalf("%s channel spec = %+v, want 8 waves × SP10", id, definition.channel)
		}
		if definition.spCost != 0 {
			t.Fatalf("%s spCost=%d, want 0 (逐波扣无预扣)", id, definition.spCost)
		}
	}
}
