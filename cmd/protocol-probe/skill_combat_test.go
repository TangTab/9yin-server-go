package main

import (
	"encoding/binary"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/combatcalc"
	"github.com/local/9yin-go-server/internal/entity"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

func useSkillMessage(id string) clientCustomMessage {
	return clientCustomMessage{Opcode: 0x0A, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomUseSkill},
		{Type: 6, Text: id},
	}}
}

func useSkillMessageAt(id string, x, y, z float32) clientCustomMessage {
	message := useSkillMessage(id)
	message.Values = append(message.Values,
		clientCustomValue{Type: 4, Float32: x},
		clientCustomValue{Type: 4, Float32: y},
		clientCustomValue{Type: 4, Float32: z},
	)
	return message
}

func TestHandleSelfSkillSpendsMPAndStartsCooldown(t *testing.T) {
	player := newPlayerActor("tester", 0)
	beforeMP := player.stateSnapshot().MP
	conn := &captureMessageConnection{}
	handled, err := handleSkillCustom(conn, player, nil, nil, useSkillMessage("CS_yhwq_hsqs05"), "test")
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("skill 211 was not handled")
	}
	if got := player.stateSnapshot().MP; got != beforeMP-11 {
		t.Fatalf("MP=%d, want %d", got, beforeMP-11)
	}
	frames := conn.Frames()
	// 出招簇（行0 整行 CD 方言，2026-11-09 真机回退）：个人 CD 行 →
	// GCD 行 → active 行 → 玩家面板 → msg20（0x1E 壳；不再发 State 动作
	// 帧、msg20 不走 0x27——42 份抓包 0 次）。
	if len(frames) != 5 || frames[0][0] != 0x11 || frames[3][0] != 0x10 || frames[4][0] != 0x1E {
		t.Fatalf("frames=%d, want CD rows(0x11 x2, row 0) + active + 0x10 player update + 0x1E msg20", len(frames))
	}
	if got := binary.LittleEndian.Uint16(frames[0][10:]); got != recordCooldown {
		t.Fatalf("cooldown record index=%d, want %d", got, recordCooldown)
	}
	if got := binary.LittleEndian.Uint16(frames[0][12:]); got != 0 {
		t.Fatalf("personal cooldown row=%d, want 0 (row0 whole-row dialect)", got)
	}
	if got := binary.LittleEndian.Uint16(frames[2][10:]); got != recordActiveSkill {
		t.Fatalf("active skill record index=%d, want %d", got, recordActiveSkill)
	}
	if got := player.currentSkillSnapshot(); got != "CS_yhwq_hsqs05" {
		t.Fatalf("CurSkillID=%q, want CS_yhwq_hsqs05", got)
	}
	_, effectID, level, target := player.currentSkillStateSnapshot()
	if effectID != "CS_yhwq_hsqs05" || level != 1 {
		t.Fatalf("skill effect state=%q level=%d", effectID, level)
	}
	if want := uint64(playerObjectID) | uint64(playerOwnerID)<<32; target != want {
		t.Fatalf("CurSkillTarget=%x, want %x", target, want)
	}
	if got := player.bufferInfo(yiHuaSkill05BuffSlot, time.Now()); got == "" {
		t.Fatal("fifth self skill did not stage buf_CS_yhwq_hsqs05")
	}
	if got := player.bufferListString(time.Now()); !strings.Contains(got, "buf_CS_yhwq_hsqs05") {
		t.Fatalf("BufferListStr=%q, want buf_CS_yhwq_hsqs05", got)
	}
	if got := player.bufferInfo(redSuperArmorBuffSlot, time.Now()); got == "" {
		t.Fatal("fifth self skill did not stage action-bound buf_fixed_red")
	}
	if got := player.bufferListString(time.Now()); !strings.Contains(got, "buf_fixed_red") {
		t.Fatalf("BufferListStr=%q, want action-bound buf_fixed_red", got)
	}
	if rules := player.buffRules(time.Now()); !rules.CantHitEffect || !rules.CantBeSkillLocked {
		t.Fatalf("red super armor rules=%+v", rules)
	}
	handled, err = handleSkillCustom(conn, player, nil, nil, useSkillMessage("CS_yhwq_hsqs05"), "test")
	if err != nil || !handled {
		t.Fatalf("second handle handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().MP; got != beforeMP-11 {
		t.Fatalf("cooldown rejection still spent MP: %d", got)
	}
	// 后续 State 帧已随 2026-11-07 战斗重写移除（Java/F 盘 exe 不下发
	// followup State 帧，动作时间线由 msg20 驱动）：CD 拒绝后不再有
	// 延迟 State 帧，帧数保持出招簇 5 条。
	time.Sleep(50 * time.Millisecond)
	frames = conn.Frames()
	if len(frames) != 5 {
		t.Fatalf("frames after follow-up=%d, want no delayed State frames (5)", len(frames))
	}
}

func TestCurrentSkillClearRejectsStaleTimer(t *testing.T) {
	player := newPlayerActor("tester", 0)
	player.currentSkillID = "CS_yhwq_hsqs05"
	if player.clearCurrentSkill("CS_yhwq_hsqs04", player.currentSkillEpochSnapshot()) {
		t.Fatal("stale skill timer cleared a later skill")
	}
	if got := player.currentSkillSnapshot(); got != "CS_yhwq_hsqs05" {
		t.Fatalf("CurSkillID=%q after stale timer", got)
	}
	if !player.clearCurrentSkill("CS_yhwq_hsqs05", player.currentSkillEpochSnapshot()) {
		t.Fatal("matching skill timer did not clear current skill")
	}
	if got := player.currentSkillSnapshot(); got != "" {
		t.Fatalf("CurSkillID=%q after matching timer", got)
	}
}

// TestCurrentSkillClearEchoesCombatLogicState 复刻真机 61512 抓包脱战闪断
// （2026-11-07）：面对面战斗中每招收尾都发出 LogicState=0——旧实现既在
// clearCurrentSkill 里硬编落 0、清除帧又硬编发 0，双双绕过 setCombatPresence
// 权威。官方收招帧（62433 抓包跨招恒 1；exe #2014 六项形态）只回显当前
// 权威值，战斗存在性独归威胁/10s 迟滞窗口管。
func TestCurrentSkillClearEchoesCombatLogicState(t *testing.T) {
	player := newPlayerActor("tester", 0)
	// 命中入战：无威胁但 10s 迟滞窗口内 Fighting。
	player.noteCombatActivity(time.Unix(1, 0))
	if changed := player.setCombatPresence(false, time.Unix(2, 0)); !changed {
		t.Fatal("presence did not enter Fighting after combat activity")
	}
	player.currentSkillID = "CS_jh_myjf03"
	if !player.clearCurrentSkill("CS_jh_myjf03", player.currentSkillEpochSnapshot()) {
		t.Fatal("matching skill timer did not clear current skill")
	}
	// 收招不得改动战斗存在性。
	if got := player.stateSnapshot().LogicState; got != entity.LogicStateFighting {
		t.Fatalf("LogicState=%d after skill clear, want Fighting", got)
	}
	frame, err := player.currentSkillClearUpdate()
	if err != nil {
		t.Fatal(err)
	}
	// 0x10 | isView | u32 obj | u32 owner | u16 n | (u16 idx + 值)*
	if frame[0] != 0x10 {
		t.Fatalf("clear frame opcode=%#x, want 0x10", frame[0])
	}
	if got := binary.LittleEndian.Uint16(frame[10:12]); got != 8 {
		t.Fatalf("clear frame props=%d, want 8 (六项 + CantMove/CantParry 官服 46.175 收尾对)", got)
	}
	wantIndices := []uint16{463, propCurSkillID, propCurSkillTarget, propCurSkillCanNoLimit, propCantBeSkillLocked, propCantBeSkillMustLocked, propRushCantMove, propCantParry}
	off := 12
	for i, want := range wantIndices {
		if got := binary.LittleEndian.Uint16(frame[off : off+2]); got != want {
			t.Fatalf("prop[%d] index=%d, want %d", i, got, want)
		}
		off += 2
		switch want {
		case 463:
			if got := frame[off]; got != entity.LogicStateFighting {
				t.Fatalf("LogicState echo=%d, want Fighting(1)", got)
			}
			off++
		case propCurSkillID:
			// 线型：u32(len+1) + 内容 + NUL；空串即 u32(1)+0x00。
			if got := binary.LittleEndian.Uint32(frame[off : off+4]); got != 1 {
				t.Fatalf("CurSkillID len=%d, want 1 (empty + NUL)", got)
			}
			if got := frame[off+4]; got != 0 {
				t.Fatalf("CurSkillID bytes=%#x, want NUL", got)
			}
			off += 5
		case propCurSkillTarget:
			off += 8
		case propCantParry:
			if got := binary.LittleEndian.Uint32(frame[off : off+4]); got != 0 {
				t.Fatalf("CantParry=%d, want 0", got)
			}
			off += 4
		default:
			if got := frame[off]; got != 0 {
				t.Fatalf("prop %d =%d, want 0", want, got)
			}
			off++
		}
	}
	if off != len(frame) {
		t.Fatalf("clear frame len=%d, parsed=%d (layout drift)", len(frame), off)
	}
	// 脱战后收招同样回显 0：普通态清招不清战斗存在性，也不误报战斗。
	if changed := player.setCombatPresence(false, time.Unix(1+int64(combatPresenceIdleExit/time.Second)+3, 0)); !changed {
		t.Fatal("presence did not exit after idle window")
	}
	frame, err = player.currentSkillClearUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if got := frame[14]; got != entity.LogicStateNormal {
		t.Fatalf("out-of-combat LogicState echo=%d, want 0", got)
	}
}

func TestHandleSkillFourStagesNativeVisibleBuff(t *testing.T) {
	player := newPlayerActor("tester", 0)
	conn := &captureMessageConnection{}
	handled, err := handleSkillCustom(conn, player, nil, nil, useSkillMessage("CS_yhwq_hsqs04"), "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.bufferInfo(yiHuaSkillBuffSlot, time.Now()); got == "" {
		t.Fatal("self skill did not stage native visible Buff")
	}
	if got := player.bufferListString(time.Now()); !strings.Contains(got, "buf_CS_yhwq_hsqs04") {
		t.Fatalf("BufferListStr=%q, want buf_CS_yhwq_hsqs04", got)
	}
	frames := conn.Frames()
	if len(frames) != 5 || frames[0][0] != 0x11 || frames[3][0] != 0x10 || frames[4][0] != 0x1E {
		t.Fatalf("frames=%d, want CD rows(0x11 x2, row 0) + active + 0x10 player update + 0x1E msg20", len(frames))
	}
	if got := binary.LittleEndian.Uint16(frames[0][12:]); got != 0 {
		t.Fatalf("personal cooldown row=%d, want 0 (row0 whole-row dialect)", got)
	}
	if got := binary.LittleEndian.Uint16(frames[2][10:]); got != recordActiveSkill {
		t.Fatalf("active skill record index=%d, want %d", got, recordActiveSkill)
	}
}

func TestNativeSkillActionFrameUsesModernMessage20(t *testing.T) {
	frame, err := nativeSkillActionFrame("CS_yhwq_hsqs05", playerObjectID, playerOwnerID)
	if err != nil {
		t.Fatal(err)
	}
	// 0x1E 壳（抓包 42 份 msg20 从未走 0x27），尾两值官方恒 int 0 /
	// float 0.0（Java FightMessages.skillAction；旧 1/1 是本地占位）。
	if len(frame) != 56 || frame[0] != 0x1E {
		t.Fatalf("native skill action frame=%x", frame)
	}
	if got := binary.LittleEndian.Uint16(frame[1:]); got != 6 {
		t.Fatalf("TVarList count=%d, want 6", got)
	}
	if frame[3] != 2 || int32(binary.LittleEndian.Uint32(frame[4:])) != serverNativeSkillActionMessage {
		t.Fatalf("native message prefix=%x, want int32 message %d", frame[3:8], serverNativeSkillActionMessage)
	}
	if frame[8] != 8 ||
		binary.LittleEndian.Uint32(frame[9:]) != playerObjectID ||
		binary.LittleEndian.Uint32(frame[13:]) != playerOwnerID {
		t.Fatalf("caster object=%x, want %d-%d", frame[8:17], playerObjectID, playerOwnerID)
	}
	if frame[17] != 6 || binary.LittleEndian.Uint32(frame[18:]) != 15 ||
		string(frame[22:36]) != "CS_yhwq_hsqs05" || frame[36] != 0 {
		t.Fatalf("skill action string=%x", frame[17:37])
	}
	if frame[37] != 8 ||
		binary.LittleEndian.Uint32(frame[38:]) != playerObjectID ||
		binary.LittleEndian.Uint32(frame[42:]) != playerOwnerID {
		t.Fatalf("target object=%x, want %d-%d", frame[37:46], playerObjectID, playerOwnerID)
	}
	if frame[46] != 2 || binary.LittleEndian.Uint32(frame[47:]) != 0 ||
		frame[51] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(frame[52:])) != 0 {
		t.Fatalf("optional action fields=%x, want int:0 float:0", frame[46:])
	}
}

func TestStandaloneActionProbeFrames(t *testing.T) {
	stateFrame, err := playerActionStateFrame("interact268")
	if err != nil {
		t.Fatal(err)
	}
	if len(stateFrame) < 16 || stateFrame[0] != 0x10 {
		t.Fatalf("State action frame=%x", stateFrame)
	}

	nativeFrame, err := nativeGenericActionFrame("interact268")
	if err != nil {
		t.Fatal(err)
	}
	if nativeFrame[0] != 0x27 || binary.LittleEndian.Uint16(nativeFrame[1:]) != 4 {
		t.Fatalf("native generic action frame=%x", nativeFrame)
	}
	if nativeFrame[3] != 6 || binary.LittleEndian.Uint32(nativeFrame[4:]) != 7 ||
		string(nativeFrame[8:14]) != "action" || nativeFrame[14] != 0 {
		t.Fatalf("native generic action message=%x, want string action", nativeFrame[3:15])
	}
	if nativeFrame[15] != 8 ||
		binary.LittleEndian.Uint32(nativeFrame[16:]) != playerObjectID ||
		binary.LittleEndian.Uint32(nativeFrame[20:]) != playerOwnerID {
		t.Fatalf("native generic action object=%x", nativeFrame[15:24])
	}
}

func TestFifthSkillTotalActionIncludesFollowupOffset(t *testing.T) {
	definition := yiHuaWuQueCombatSkills["CS_yhwq_hsqs05"]
	wantFollowup := 31*time.Second/30 - 240*time.Millisecond
	if got := definition.followupActions[0].at; got != wantFollowup {
		t.Fatalf("fifth skill followup=%s, want %s", got, wantFollowup)
	}
	want := wantFollowup + 113*time.Second/30
	if got := definition.totalActionDuration(); got != want {
		t.Fatalf("total action duration=%s, want %s", got, want)
	}
}

func TestTaiJiQuanGuPuMetadataAndThreeSegmentAction(t *testing.T) {
	if len(taiJiQuanGuPuCombatSkills) != 8 {
		t.Fatalf("tai ji skill count=%d, want 8", len(taiJiQuanGuPuCombatSkills))
	}
	definition := taiJiQuanGuPuCombatSkills["CS_wd_tjq08"]
	if definition.spCost != 50 || definition.baseDamage != 1304 ||
		definition.personalCD != 10*time.Second || definition.publicCD != 7*time.Second ||
		definition.areaRadius != 5 || definition.areaHeight != 5 {
		t.Fatalf("开太极 metadata=%+v", definition)
	}
	wantSecond := 140*time.Second/30 - 637500*time.Microsecond
	wantThird := wantSecond + 65*time.Second/30 - 120*time.Millisecond
	if len(definition.followupActions) != 2 ||
		definition.followupActions[0].action != "Test0001" ||
		definition.followupActions[0].at != wantSecond ||
		definition.followupActions[0].duration != 65*time.Second/30 ||
		definition.followupActions[1].action != "fight_0h_jb_10" ||
		definition.followupActions[1].at != wantThird ||
		definition.followupActions[1].duration != 58*time.Second/30 {
		t.Fatalf("开太极 action sequence=%+v", definition.followupActions)
	}
	want := wantThird + 58*time.Second/30
	if got := definition.totalActionDuration(); got != want {
		t.Fatalf("开太极 total action=%s, want %s", got, want)
	}
	if definition.requiresTarget {
		t.Fatal("开太极 is a caster-centred area skill, not a selected-target skill")
	}
}

func TestOpenTaiJiStartsWithoutSelectedTargetAndStagesActionBoundRedArmor(t *testing.T) {
	player := newPlayerActor("tester", 0)
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	handled, err := handleSkillCustom(conn, player, world, nil, useSkillMessage("CS_wd_tjq08"), "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got, want := player.skillSP(), player.rageCap()-taiJiQuanGuPuCombatSkills["CS_wd_tjq08"].spCost; got != want {
		t.Fatalf("SP=%d, want %d after 开太极", got, want)
	}
	if got := player.bufferInfo(redSuperArmorBuffSlot, time.Now()); got == "" {
		t.Fatal("开太极 did not stage action-bound buf_fixed_red")
	}
	if got := player.bufferListString(time.Now()); !strings.Contains(got, "buf_fixed_red") {
		t.Fatalf("BufferListStr=%q, want buf_fixed_red", got)
	}
	frames := conn.Frames()
	// 出招簇（行0 整行 CD 方言，不再发 State 动作帧；msg20 走 0x1E 壳）。
	if len(frames) != 5 ||
		frames[0][0] != 0x11 || frames[3][0] != 0x10 || frames[4][0] != 0x1E {
		t.Fatalf("开太极 initial frames=%x", frames)
	}
	if got := binary.LittleEndian.Uint16(frames[0][12:]); got != 0 {
		t.Fatalf("开太极 personal cooldown row=%d, want 0 (row0 whole-row dialect)", got)
	}
	if got := binary.LittleEndian.Uint16(frames[2][10:]); got != recordActiveSkill {
		t.Fatalf("开太极 active skill record=%d, want %d", got, recordActiveSkill)
	}
}

func TestOpenTaiJiDamagesCombatNPCsInsideInstalledFiveMetreCircle(t *testing.T) {
	player := newPlayerActor("tester", 0)
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	registerTestCombatNPC(t, world, 92)

	placeTestNPC(t, world, 91, 4, 0)
	placeTestNPC(t, world, 92, 6, 0)

	// 掷骰伤害（2026-11-07 战斗重写）：attack=20..35 × ratio=baseDamage
	// /100/waves（开太极 1304/100/1=13.04）。注入确定性随机源（0.5/0.5/0.5
	// =命中、非暴击、攻方 20+floor(0.5×16)=28）→ 伤害恒 floor(28×13.04)
	// =365，不再直扣静态 1304。
	// 2026-11-11 hitbreak 时序批次：开太极三段判定 333/6000/7933ms（段链
	// 全时长累加）——首段后验 635，末段（3×365=1095>1000）致死。
	originalRandom := combatRandomFloat64
	combatRandomFloat64 = func() float64 { return 0.5 }
	defer func() { combatRandomFloat64 = originalRandom }()
	handled, err := handleSkillCustom(
		conn,
		player,
		world,
		nil,
		useSkillMessageAt("CS_wd_tjq08", 0, 0, 0),
		"test",
	)
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	waitForHP := func(want func(int32) bool, what string) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for {
			inside, _ := world.run.combat.Vitals(worldcore.EntityID(forkNPCID(91)))
			if want(inside.HP) {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("inside NPC HP=%d, want %s", inside.HP, what)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	waitForHP(func(hp int32) bool { return hp == 1000-365 }, "635 after first 365 damage tick")
	waitForHP(func(hp int32) bool { return hp <= 0 }, "dead after three ticks of 365")
	if outside, _ := world.run.combat.Vitals(worldcore.EntityID(forkNPCID(92))); outside.HP <= 0 {
		t.Fatalf("outside NPC HP=%d, want alive outside five metres", outside.HP)
	}
}

func TestCompiledSectorDamageUsesC2S211CasterOrientation(t *testing.T) {
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 93)
	registerTestCombatNPC(t, world, 94)
	placeTestNPC(t, world, 93, 4, 0)
	placeTestNPC(t, world, 94, -4, 0)

	results, err := world.applyShapedSkillDamage(
		worldcore.Transform{Orient: 0},
		5,
		5,
		120,
		0,
		100,
		time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].id != forkNPCID(93) {
		t.Fatalf("sector results=%+v, want only facing NPC 93", results)
	}
	if front, _ := world.run.combat.Vitals(worldcore.EntityID(forkNPCID(93))); front.HP != 900 {
		t.Fatalf("front NPC HP=%d, want 900", front.HP)
	}
	if back, _ := world.run.combat.Vitals(worldcore.EntityID(forkNPCID(94))); back.HP != 1000 {
		t.Fatalf("back NPC HP=%d, want untouched 1000", back.HP)
	}
}

func TestCompiledRectangleDamageUsesLengthWidthAndOrientation(t *testing.T) {
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint32{95, 96, 97} {
		registerTestCombatNPC(t, world, id)
	}
	placeTestNPC(t, world, 95, 8, 1)
	placeTestNPC(t, world, 96, 8, 3)
	placeTestNPC(t, world, 97, -2, 0)

	results, err := world.applyShapedSkillDamage(
		worldcore.Transform{Orient: 0},
		10,
		5,
		0,
		4,
		100,
		time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].id != forkNPCID(95) {
		t.Fatalf("rectangle results=%+v, want only NPC 95", results)
	}
}

func TestTaiJiStanceAlternatesAndTimedBuffsUseInstalledRecords(t *testing.T) {
	player := newPlayerActor("tester", 0)
	target := uint64(playerObjectID) | uint64(playerOwnerID)<<32
	now := time.Now()
	definition := taiJiQuanGuPuCombatSkills["CS_wd_tjq04"]
	if ok, reason := player.beginSkillUse(definition, target, now, false); !ok {
		t.Fatalf("begin 金鸡独立: %s", reason)
	}
	player.activateNamedBuff(taiJiStanceBuffSlot, taiJiDefenceStanceBuffStaticData, "buf_CS_wd_tjq04_1", taiJiStanceBuffLease, now)
	if !player.hasActiveBuff(taiJiStanceBuffSlot, taiJiDefenceStanceBuffStaticData, now) {
		t.Fatal("defence stance not active")
	}
	player.activateNamedBuff(taiJiStanceBuffSlot, taiJiAttackStanceBuffStaticData, "buf_CS_wd_tjq04_2", taiJiStanceBuffLease, now)
	if !player.hasActiveBuff(taiJiStanceBuffSlot, taiJiAttackStanceBuffStaticData, now) {
		t.Fatal("attack stance did not replace defence stance")
	}
}

func TestSkillCompletionClearsStateAndRestoresStand(t *testing.T) {
	player := newPlayerActor("tester", 0)
	player.currentSkillID = "test_skill"
	player.currentSkillEffectID = "test_skill"
	definition := combatSkillDefinition{id: "test_skill", actionDuration: time.Millisecond}
	conn := &captureMessageConnection{}

	scheduleCurrentSkillClear(conn, player, definition, player.currentSkillEpochSnapshot())
	deadline := time.Now().Add(time.Second)
	// 52905 抓包 14/14 收招簇：0x12 删 active_skill 行 → 六属性清除帧 →
	// 单条 State(idx19)="stand"（24B），共 3 帧（此前双 State 通道比抓包
	// 多一条，2026-11-07 战斗重写收敛到官方簇形）。
	for len(conn.Frames()) < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	frames := conn.Frames()
	if len(frames) != 3 || frames[0][0] != 0x12 || frames[1][0] != 0x10 || frames[2][0] != 0x10 {
		t.Fatalf("completion frames=%d, want 0x12 del row, CurSkill clear, State stand", len(frames))
	}
	if got := binary.LittleEndian.Uint16(frames[0][10:]); got != recordActiveSkill {
		t.Fatalf("delete record=%d, want %d", got, recordActiveSkill)
	}
	if got := player.currentSkillSnapshot(); got != "" {
		t.Fatalf("CurSkillID=%q after completion", got)
	}
}

func TestSkillFourCooldownRecordMatchesInstalledStaticData(t *testing.T) {
	definition := yiHuaWuQueCombatSkills["CS_yhwq_hsqs04"]
	start := time.UnixMilli(123456)
	personal, public, err := skillCooldownFrames(definition, start)
	if err != nil {
		t.Fatal(err)
	}
	// 官方 cooldown_rec [3,4,4,2]：16 头 + 4+8+8+2 = 38 字节，
	// 末列 team 是 u16。双行：个人行 team 恒 0xFFFF；GCD 行类别
	// 固定 50000、team 取 skill_static CoolDownTeam。
	for _, frame := range [][]byte{personal, public} {
		if len(frame) != 38 || frame[0] != 0x11 {
			t.Fatalf("cooldown frame=%x", frame)
		}
		if got := binary.LittleEndian.Uint16(frame[10:]); got != recordCooldown {
			t.Fatalf("record index=%d, want %d", got, recordCooldown)
		}
		begin := int64(binary.LittleEndian.Uint64(frame[20:]))
		if begin != start.UnixMilli() {
			t.Fatalf("cooldown begin=%d", begin)
		}
	}
	if got := binary.LittleEndian.Uint32(personal[16:]); got != 12445 {
		t.Fatalf("personal category=%d, want 12445", got)
	}
	if end := int64(binary.LittleEndian.Uint64(personal[28:])); end-start.UnixMilli() != 1000 {
		t.Fatalf("personal duration=%d ms, want 1000", end-start.UnixMilli())
	}
	if got := binary.LittleEndian.Uint16(personal[36:]); got != 0xFFFF {
		t.Fatalf("personal team=%d, want 0xFFFF", got)
	}
	if got := binary.LittleEndian.Uint32(public[16:]); got != 50000 {
		t.Fatalf("GCD category=%d, want 50000", got)
	}
	if got := binary.LittleEndian.Uint16(public[36:]); got != 50267 {
		t.Fatalf("GCD team=%d, want 50267", got)
	}
}

func TestSkillSevenSpendsSP(t *testing.T) {
	player := newPlayerActor("tester", 0)
	definition := yiHuaWuQueCombatSkills["CS_yhwq_hsqs07"]
	definition.requiresTarget = false
	ok, reason := player.beginSkillUse(definition, uint64(playerObjectID)|uint64(playerOwnerID)<<32, time.Now(), false)
	if !ok {
		t.Fatalf("begin skill seven: %s", reason)
	}
	if got, want := player.progress.attributes.sp, player.rageCap()-definition.spCost; got != want {
		t.Fatalf("SP=%d, want %d", got, want)
	}
}

func TestAttackSkillRequiresSelectedTarget(t *testing.T) {
	player := newPlayerActor("tester", 0)
	beforeMP := player.stateSnapshot().MP
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	handled, err := handleSkillCustom(conn, player, world, nil, useSkillMessage("CS_yhwq_hsqs01"), "test")
	if err != nil || !handled {
		t.Fatalf("handled=%t err=%v", handled, err)
	}
	if got := player.stateSnapshot().MP; got != beforeMP {
		t.Fatalf("target rejection spent MP: %d", got)
	}
	if len(conn.frames) != 0 {
		t.Fatalf("target rejection emitted %d frames", len(conn.frames))
	}
}

// newTestCombatEngine 装载一个仅注入指定规格的每击公式引擎（真实
// scripts/module/combat/formula.lua）。
func newTestCombatEngine(t *testing.T, specs ...combatcalc.FormulaSpec) *combatcalc.Engine {
	t.Helper()
	engine := combatcalc.New(filepath.Join("..", "..", "scripts", "module", "combat"), nil)
	if _, err := engine.Load(specs); err != nil {
		t.Fatalf("combat engine load: %v", err)
	}
	return engine
}

// taijiFormula494 镜像 formula.ini [494]（太极拳系威力系数公式）。
func taijiFormula494() combatcalc.FormulaSpec {
	return combatcalc.FormulaSpec{
		ID:      494,
		Formula: "(P1+((P4+P5+P6+P7)*P2*0.195))/P3",
		Src:     map[int]int{1: 5, 2: 5, 3: 5, 4: 0, 5: 0, 6: 0, 7: 0},
		Prop: map[int]string{
			4: "MeleePower", 5: "MeleePowerAdd",
			6: "MinMeleeDamage", 7: "MaxMeleeDamage",
		},
	}
}

func TestRollSkillHitUsesPerHitFormulaEngine(t *testing.T) {
	original := installedCombatCalc
	installedCombatCalc = newTestCombatEngine(t, taijiFormula494())
	defer func() { installedCombatCalc = original }()

	originalRandom := combatRandomFloat64
	combatRandomFloat64 = func() float64 { return 0.5 }
	defer func() { combatRandomFloat64 = originalRandom }()

	player := newPlayerActor("tester", 0)
	definition := combatSkillDefinition{
		id: "CS_wd_tjq01", script: "SkillNormal",
		damageRows: []skillDamageRow{{formulaID: 494, damageType: "NegDamage", params: []string{"112", "1", "2"}}},
		hitCount:   10,
	}
	// 默认属性（str=30/meleePower=80/武器 20-35）：
	// (112+((80+30+20+35)*1*0.195))/2 = 72.08 → 72（legacy 路径会掷 28×ratio）
	roll := rollSkillHit(player, definition, skillDamageTargetRef{}, nil, 1)
	if roll.damage != 72 || roll.crit {
		t.Fatalf("per-hit roll = %+v, want damage=72 crit=false", roll)
	}
}

func TestRollSkillHitFallsBackWhenFormulaUnsupported(t *testing.T) {
	original := installedCombatCalc
	installedCombatCalc = newTestCombatEngine(t, taijiFormula494())
	defer func() { installedCombatCalc = original }()

	originalRandom := combatRandomFloat64
	combatRandomFloat64 = func() float64 { return 0.5 }
	defer func() { combatRandomFloat64 = originalRandom }()

	player := newPlayerActor("tester", 0)
	// 公式号未注入 → legacy：attack=20+floor(0.5×16)=28，ratio<=0 按 1 → 28
	definition := combatSkillDefinition{
		id: "CS_test", script: "SkillNormal",
		damageRows: []skillDamageRow{{formulaID: 777, damageType: "PhyDamage", params: []string{"100"}}},
	}
	roll := rollSkillHit(player, definition, skillDamageTargetRef{}, nil, 1)
	if roll.damage != 28 {
		t.Fatalf("legacy fallback roll = %+v, want damage=28", roll)
	}
	// 引擎未装配（测试默认态）同样走 legacy
	installedCombatCalc = nil
	roll = rollSkillHit(player, definition, skillDamageTargetRef{}, nil, 1)
	if roll.damage != 28 {
		t.Fatalf("no-engine fallback roll = %+v, want damage=28", roll)
	}
}

// TestMultiHitSkillRollsPerHitIndependently 验证多段技能每击独立掷值
// （2026-11-08 数值重写：此前一掷同值复用各段）。三段各掷 R[10,20]：
// 预掷序列 0.1/0.3/0.7 → 伤害 11/13/17，合计 41。
func TestMultiHitSkillRollsPerHitIndependently(t *testing.T) {
	engine := newTestCombatEngine(t, combatcalc.FormulaSpec{
		ID: 9, Formula: "R[P1,P2]", Src: map[int]int{1: 5, 2: 5},
	})
	originalEngine := installedCombatCalc
	installedCombatCalc = engine
	defer func() { installedCombatCalc = originalEngine }()

	// 计数随机源：每次 rollSkillHit 消费 9 枚（miss/crit/6×extra），
	// miss 与 crit 恒 0.5（不触发），extra 首位按序给 0.1/0.3/0.7。
	rolls := []float64{0.1, 0.3, 0.7, 0.9}
	var index int
	var mu sync.Mutex
	originalRandom := combatRandomFloat64
	combatRandomFloat64 = func() float64 {
		mu.Lock()
		defer mu.Unlock()
		if index%9 < 2 {
			index++
			return 0.5
		}
		value := rolls[(index/9)%len(rolls)]
		index++
		return value
	}
	defer func() { combatRandomFloat64 = originalRandom }()

	player := newPlayerActor("tester", 0)
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 95)
	placeTestNPC(t, world, 95, 4, 0)

	definition := combatSkillDefinition{
		id: "CS_test_multihit", script: "SkillNormal",
		damageRows: []skillDamageRow{{formulaID: 9, damageType: "PhyDamage", params: []string{"10", "20"}}},
		hitFrameMs: []time.Duration{time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond},
	}
	// ownerID 与 resolveSkillDamageTargets 同源（npcMeta 派生）。
	meta, ok := world.npcMeta(worldcore.EntityID(forkNPCID(95)))
	if !ok {
		t.Fatal("npc 95 meta missing")
	}
	scheduleSkillMultiHits(conn, world, definition, player,
		[]skillDamageTargetRef{{id: forkNPCID(95), ownerID: meta.ownerID}}, clientCustomMessage{}, "test")
	time.Sleep(120 * time.Millisecond)

	vitals, ok := world.run.combat.Vitals(worldcore.EntityID(forkNPCID(95)))
	if !ok {
		t.Fatal("NPC 95 missing")
	}
	if got, want := vitals.HP, int32(1000-11-13-17); got != want {
		t.Fatalf("NPC HP=%d, want %d (per-hit 11/13/17)", got, want)
	}
}

// TestShapedMultihitPicksUpNewEntrants 范围型多段逐段重索敌（用户
// 2026-11-12 口径：技能期间进入范围的敌人也要吃伤害）——出招快照
// 只有 NPC95，段间 NPC96 走进范围后两段都命中它。
func TestShapedMultihitPicksUpNewEntrants(t *testing.T) {
	originalEngine := installedCombatCalc
	installedCombatCalc = newTestCombatEngine(t, combatcalc.FormulaSpec{
		ID: 9, Formula: "R[P1,P2]", Src: map[int]int{1: 5, 2: 5},
	})
	defer func() { installedCombatCalc = originalEngine }()
	originalRandom := combatRandomFloat64
	combatRandomFloat64 = func() float64 { return 0.5 }
	defer func() { combatRandomFloat64 = originalRandom }()

	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 95)
	placeTestNPC(t, world, 95, 4, 0)
	registerTestCombatNPC(t, world, 96)
	placeTestNPC(t, world, 96, 50, 0) // 出招时在范围外

	definition := combatSkillDefinition{
		id: "CS_test_shaped", script: "SkillNormal",
		targetMode: skillTargetCasterCircle, areaRadius: 6,
		// 生产目录恒有 skillEffectDamage（compileSkillDamage base>0 时
		// 追加），resolveSkillDamageTargets 的 hasDamage 闸门消费它。
		effects:    []combatSkillEffect{{kind: skillEffectDamage, amount: 10}},
		damageRows: []skillDamageRow{{formulaID: 9, damageType: "PhyDamage", params: []string{"10", "20"}}},
		// 帧位须给「快照后、首跳前」的进场留足余量：命中跳走 time.
		// AfterFunc 墙钟，go test ./... 并行多包抢 CPU 时主协程可能
		// 迟 5ms+ 才写 transform——窄窗（5/20ms）下 NPC96 错过首跳，
		// 断言随机红（2026-11-12 flaky 定位）。150ms 首跳+450ms 次跳
		// 下进场余量 30 倍，仍覆盖「两段都吃到」的语义。
		hitFrameMs: []time.Duration{150 * time.Millisecond, 450 * time.Millisecond},
	}
	custom := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 211},
		{Type: 6, Text: "CS_test_shaped"},
		{Type: 4, Float32: 0}, {Type: 4, Float32: 0}, {Type: 4, Float32: 0},
	}}
	player := newPlayerActor("tester", 0)
	targets := world.resolveSkillDamageTargets(definition, custom, 0, 0)
	if len(targets) != 1 || targets[0].id != forkNPCID(95) {
		t.Fatalf("出招快照=%v, want 仅 NPC95（96 在范围外）", targets)
	}
	scheduleSkillMultiHits(conn, world, definition, player, targets, custom, "test")
	// 技能期间 NPC96 走进范围（段间进场）。
	world.run.mu.Lock()
	world.run.writeNpcTransform(worldcore.EntityID(forkNPCID(96)), worldcore.Transform{X: 1, Z: 0})
	world.run.mu.Unlock()
	time.Sleep(700 * time.Millisecond)

	hp95, _ := world.run.combat.Vitals(worldcore.EntityID(forkNPCID(95)))
	hp96, _ := world.run.combat.Vitals(worldcore.EntityID(forkNPCID(96)))
	if hp95.HP != 1000-2*15 {
		t.Fatalf("NPC95 HP=%d, want %d (两段各 15)", hp95.HP, 1000-2*15)
	}
	if hp96.HP != 1000-2*15 {
		t.Fatalf("NPC96 HP=%d, want %d (进场后两段都命中——旧快照实现恒 1000)", hp96.HP, 1000-2*15)
	}
}

// TestRollSkillHitPicksCritValueByDamageType 验证暴击值按伤害系取用：
// phyVa=0/magicVa=16 时，PhyDamage 行暴击率 5%（0.10 不暴）、内功系行
// 14.4%（0.10 暴击 → 72×2=144）。
func TestRollSkillHitPicksCritValueByDamageType(t *testing.T) {
	original := installedCombatCalc
	installedCombatCalc = newTestCombatEngine(t, taijiFormula494())
	defer func() { installedCombatCalc = original }()

	originalRandom := combatRandomFloat64
	combatRandomFloat64 = func() float64 { return 0.10 }
	defer func() { combatRandomFloat64 = originalRandom }()

	player := newPlayerActor("tester", 0)
	player.mu.Lock()
	player.progress.attributes.phyVa = 0
	player.progress.attributes.magicVa = 16
	player.mu.Unlock()

	external := combatSkillDefinition{
		damageRows: []skillDamageRow{{formulaID: 494, damageType: "NegDamage", params: []string{"112", "1", "2"}}},
	}
	if roll := rollSkillHit(player, external, skillDamageTargetRef{}, nil, 1); !roll.crit || roll.damage != 144 {
		t.Fatalf("内功系应暴击 144，得 %+v", roll)
	}
	physical := combatSkillDefinition{
		damageRows: []skillDamageRow{{formulaID: 494, damageType: "PhyDamage", params: []string{"112", "1", "2"}}},
	}
	if roll := rollSkillHit(player, physical, skillDamageTargetRef{}, nil, 1); roll.crit || roll.damage != 72 {
		t.Fatalf("外功系应不暴击 72，得 %+v", roll)
	}
}
