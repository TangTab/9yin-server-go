package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/skillscript"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

func warningToggleMessage(id int32, extra ...clientCustomValue) clientCustomMessage {
	values := append([]clientCustomValue{{Type: 2, Int32: id}}, extra...)
	return clientCustomMessage{Values: values}
}

func dashLandingMessage(x, y, z, orient float32) clientCustomMessage {
	// fork/Java 守卫均要求 len(values)>=5（朝向值必须到场，tag4 才被读取）。
	return clientCustomMessage{Opcode: 0x1E, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomCastDashLanding},
		{Type: 4, Float32: x},
		{Type: 4, Float32: y},
		{Type: 4, Float32: z},
		{Type: 4, Float32: orient},
	}}
}

func movementUploadMessage(x, y, z, orient float32) clientCustomMessage {
	values := []clientCustomValue{
		{Type: 2, Int32: clientCustomMovementUpload},
		{Type: 2, Int32: -1610592768}, // 0xA0005000 的 int32 下转（客户端 nx_int u32 回绕形态）
		{Type: 2, Int32: 1},
		{Type: 4, Float32: x},
		{Type: 4, Float32: y},
		{Type: 4, Float32: z},
		{Type: 4, Float32: orient},
	}
	return clientCustomMessage{Opcode: 0x1E, Values: values}
}

// 冲锋链帧字节钉死（外服私服 7071/7074 线值序，0x1E 外壳）：
// msg25=[25,30000,"skill_pause"]；msg26=[26]；
// msg337=[337,obj,x,y,z,orient,0]；msg24 c61=[24,obj,61,dest,档1,冲刺动画]。
func TestRushFrameBuildersMatchJavaLayouts(t *testing.T) {
	pause, err := skillPauseFrame(30000)
	if err != nil {
		t.Fatal(err)
	}
	if len(pause) != 30 || pause[0] != 0x1E || binary.LittleEndian.Uint16(pause[1:]) != 3 ||
		pause[3] != 2 || int32(binary.LittleEndian.Uint32(pause[4:])) != 25 ||
		pause[8] != 2 || binary.LittleEndian.Uint32(pause[9:]) != 30000 ||
		pause[13] != 6 || string(pause[18:29]) != "skill_pause" || pause[29] != 0 {
		t.Fatalf("msg25 frame=%x", pause)
	}

	clear, err := skillPauseClearFrame()
	if err != nil {
		t.Fatal(err)
	}
	if len(clear) != 8 || clear[0] != 0x1E || binary.LittleEndian.Uint16(clear[1:]) != 1 ||
		clear[3] != 2 || int32(binary.LittleEndian.Uint32(clear[4:])) != 26 {
		t.Fatalf("msg26 frame=%x", clear)
	}

	sync, err := casterTransformSyncFrame(1.5, -2, 3.25, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(sync) != 42 || sync[0] != 0x1E || binary.LittleEndian.Uint16(sync[1:]) != 7 ||
		sync[3] != 2 || int32(binary.LittleEndian.Uint32(sync[4:])) != 337 ||
		sync[8] != 8 || binary.LittleEndian.Uint32(sync[9:]) != playerObjectID ||
		binary.LittleEndian.Uint32(sync[13:]) != playerOwnerID ||
		sync[17] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(sync[18:])) != 1.5 ||
		sync[22] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(sync[23:])) != -2 ||
		sync[27] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(sync[28:])) != 3.25 ||
		sync[32] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(sync[33:])) != 0.5 ||
		sync[37] != 2 || binary.LittleEndian.Uint32(sync[38:]) != 0 {
		t.Fatalf("msg337 frame=%x", sync)
	}

	dash, err := dashRushFrame(10, 0, 20, convertDashLevel, defaultGroundRushAction)
	if err != nil {
		t.Fatal(err)
	}
	if len(dash) != 64 || dash[0] != 0x1E || binary.LittleEndian.Uint16(dash[1:]) != 8 ||
		dash[3] != 2 || int32(binary.LittleEndian.Uint32(dash[4:])) != 24 ||
		dash[8] != 8 || binary.LittleEndian.Uint32(dash[9:]) != playerObjectID ||
		binary.LittleEndian.Uint32(dash[13:]) != playerOwnerID ||
		dash[17] != 2 || int32(binary.LittleEndian.Uint32(dash[18:])) != msgRushCommand ||
		dash[22] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(dash[23:])) != 10 ||
		dash[27] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(dash[28:])) != 0 ||
		dash[32] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(dash[33:])) != 20 ||
		dash[37] != 2 || int32(binary.LittleEndian.Uint32(dash[38:])) != convertDashLevel ||
		dash[42] != 6 || string(dash[47:63]) != defaultGroundRushAction || dash[63] != 0 {
		t.Fatalf("msg24 rush dash frame=%x", dash)
	}
}

// 受理簇（技能1.pcapng 定版首发 5 帧）：CurSkillID → CurSkillTarget →
// msg25[30000] → PauseSkill → msg707[15]。重受理 7 帧：三帧清前导
// CurSkillID=""+msg26+PauseSkill="" 后**不重发 CurSkillTarget**。
func TestStageRushCastStagesPendingAndMarksInFlight(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off") // 本测试只验受理簇方言，关服务端驱动接近
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_stage", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction}

	before := time.Now()
	if handled, err := stageRushCast(conn, player, nil, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 0, Z: 20}, worldcore.Transform{X: 4, Y: 0, Z: 20}, 12, "test"); err != nil || !handled {
		t.Fatalf("stage handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 9 { // 5 受理簇（m25 已恢复）+ 4 状态四连
		t.Fatalf("staging frames=%d, want 9-frame 簇（受理簇零 m24 且首受理带 PauseSkill）", len(frames))
	}
	if frames[0][0] != 0x10 || binary.LittleEndian.Uint16(frames[0][12:]) != propCurSkillID {
		t.Fatalf("CurSkillID frame=%x", frames[0])
	}
	if frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][12:]) != propCurSkillTarget {
		t.Fatalf("CurSkillTarget frame=%x", frames[1])
	}
	if int32(binary.LittleEndian.Uint32(frames[2][4:])) != 25 {
		t.Fatalf("msg25 frame=%x", frames[2])
	}
	if frames[3][0] != 0x10 || binary.LittleEndian.Uint16(frames[3][12:]) != propRushPauseSkill {
		t.Fatalf("PauseSkill frame=%x", frames[3])
	}
	if int32(binary.LittleEndian.Uint32(frames[4][4:])) != 707 {
		t.Fatalf("msg707 frame=%x", frames[4])
	}
	pending := player.peekPendingCast(time.Now())
	if pending == nil || pending.skillID != "CS_rush_stage" || pending.targetID != forkNPCID(91) || pending.targetOwner != npcTestOwner(91) || pending.airborne {
		t.Fatalf("pending cast=%+v", pending)
	}
	if pending.expiresUTC.Before(before.Add(rushPendingWindow - time.Second)) {
		t.Fatalf("pending window=%s, want ~%s", pending.expiresUTC.Sub(before), rushPendingWindow)
	}
	if !player.rushInFlightSnapshot() {
		t.Fatal("staging did not mark rush in flight")
	}
	if player.rushDashIssuedSnapshot() {
		t.Fatal("staging beyond dash range issued dash")
	}
	if got := player.stateSnapshot().MP; got != player.vitals.MaxMP {
		t.Fatalf("staging spent MP: %d", got)
	}

	// 暂存期再次按下（重受理）：外服私服 f/g 定版 7 帧——CurSkillID="" +
	// msg26 + PauseSkill="" + CurSkillID + msg25 + PauseSkill + msg707，
	// 不重发 CurSkillTarget。
	if handled, err := stageRushCast(conn, player, nil, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 0, Z: 20}, worldcore.Transform{X: 4, Y: 0, Z: 20}, 12, "test"); err != nil || !handled {
		t.Fatalf("rearm stage handled=%t err=%v", handled, err)
	}
	frames3 := conn.Frames()[9:]
	if len(frames3) != 11 ||
		frames3[0][0] != 0x10 || binary.LittleEndian.Uint16(frames3[0][12:]) != propCurSkillID ||
		int32(binary.LittleEndian.Uint32(frames3[1][4:])) != 26 ||
		binary.LittleEndian.Uint16(frames3[2][12:]) != propRushPauseSkill ||
		binary.LittleEndian.Uint16(frames3[3][12:]) != propCurSkillID ||
		int32(binary.LittleEndian.Uint32(frames3[4][4:])) != 25 ||
		int32(binary.LittleEndian.Uint32(frames3[6][4:])) != 707 {
		t.Fatalf("rearm frames=%x, want 11-frame re-arm (7+state-ping) without CurSkillTarget resend", frames3)
	}
}

// 空中判定启用（v29，官服观察「空中奔袭凭空飞过不落地」）：211 自报
// castAt.y 高出目标 y 1.5m+ 即空中档标记（旧 1e9 禁用线撤销——其根因
// 是 NPC 服务端 y 口径时代，冲锋族 geometry 已改 211 自报目标坐标）。
func TestStageRushCastAirborneMarksPendingAir(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off") // 本测试只验受理簇方言，关服务端驱动接近
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_air", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction}
	if _, err := stageRushCast(conn, player, nil, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 43, Z: 20}, worldcore.Transform{X: 4, Y: 50, Z: 20}, 12, "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 10 {
		t.Fatalf("air staging frames=%d, want 10 (stage cluster + CurIsAirSkill, without m24)", len(frames))
	}
	sawAirMark := false
	for _, f := range frames {
		if f[0] == 0x10 && len(f) >= 15 &&
			binary.LittleEndian.Uint16(f[12:]) == propCurIsAirSkill && f[14] == 4 {
			sawAirMark = true // v31 空中受理 CurIsAirSkill=4（技能4/5 受理簇实证）
		}
	}
	if !sawAirMark {
		t.Fatalf("air staging must carry CurIsAirSkill=4, frames=%x", frames)
	}
	if pending := player.peekPendingCast(time.Now()); pending == nil || !pending.airborne {
		t.Fatalf("pending cast=%+v, want airborne=true", pending)
	}
	// 地面按压（y 差 0）仍走地面档。
	if _, err := stageRushCast(conn, player, nil, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 43, Z: 20}, worldcore.Transform{X: 4, Y: 43.4, Z: 20}, 12, "test"); err != nil {
		t.Fatal(err)
	}
	if pending := player.peekPendingCast(time.Now()); pending == nil || pending.airborne {
		t.Fatalf("ground pending cast=%+v, want airborne=false", pending)
	}
}

// per-skill c61 授予方言（技能1.pcapng 2026-11-13 定版，外服私服 16 次授予观测）：
// hyjf02 窗 (7.5,8.5]——6m 窗下零帧（走入直收桶）、8m 授予 G 类 flag1+
// lock 动画；xytf07_sky 窗 (8,11] 授予 A 类 flag2+qg_fastruning_s_0h_01、
// 终点 y=目标 y+1.0；mzjf02 上界 8——13m 零授予（220@13.8 外服私服零应答）。
// 无暂存 212：外服私服零回应（b 51.4+ 连发 15 条 212[尾0] 全程静默）。
func TestDashLandingWithoutPendingIsSilent(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)

	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(4, 0, 5, 1.25), "test"); err != nil || !handled {
		t.Fatalf("cold landing handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("cold landing emitted %d frames, want 0 (official silence)", got)
	}
	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(4, 0, 5, 0), "test"); err != nil || !handled {
		t.Fatalf("repeat landing handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("repeat landing emitted %d frames, want 0", got)
	}
}

// 212 有暂存且冲刺已发出（自增益目标，零成本定义跳过结算）：
// 冲刺落地簇 = CantMove/CantParry 清 + PauseSkill="" + msg26 +
// NPC(自身)LogicState=1 + fight_relation 行0 + LogicState=1 +
// 0x1F×2 + msg337 + c1=40 + msg19(14 值) + CantMove=1 + CantParry=1。
func TestDashLandingResumesPendingSelfSkill(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)
	player.stagePendingCast(&pendingCastState{
		skillID: "CS_yhwq_hsqs05", targetID: playerObjectID, targetOwner: playerOwnerID,
		expiresUTC: time.Now().Add(rushPendingWindow),
	})
	player.markRushInFlight()
	player.markRushDashIssued(false)
	beforeMP := player.stateSnapshot().MP

	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(6, 0, 7, 0), "test"); err != nil || !handled {
		t.Fatalf("landing handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 18 { // 14 帧落地簇 + 状态四连（外服私服 416.321 实证）
		t.Fatalf("landing frames=%d, want 18-frame dash landing cluster+state-ping", len(frames))
	}
	if frames[0][0] != 0x10 || binary.LittleEndian.Uint16(frames[0][12:]) != propRushCantMove {
		t.Fatalf("CantMove clear frame=%x", frames[0])
	}
	if frames[2][0] != 0x10 || binary.LittleEndian.Uint16(frames[2][12:]) != propRushPauseSkill {
		t.Fatalf("PauseSkill clear frame=%x", frames[2])
	}
	if frames[3][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[3][4:])) != 26 {
		t.Fatalf("pause clear frame=%x", frames[3])
	}
	if frames[4][0] != 0x10 || binary.LittleEndian.Uint16(frames[4][12:]) != propRushLogicState {
		t.Fatalf("NPC LogicState frame=%x", frames[4])
	}
	if frames[5][0] != 0x11 || binary.LittleEndian.Uint16(frames[5][10:]) != recordFightRelation {
		t.Fatalf("fight relation row=%x", frames[5])
	}
	if frames[7][0] != 0x1F {
		t.Fatalf("first anchor=%x", frames[7])
	}
	if frames[8][0] != 0x1F {
		t.Fatalf("second anchor=%x", frames[8])
	}
	if frames[9][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[9][4:])) != 337 {
		t.Fatalf("msg337 frame=%x", frames[9])
	}
	if frames[10][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[10][4:])) != 24 ||
		int32(binary.LittleEndian.Uint32(frames[10][18:])) != 40 {
		t.Fatalf("c1=40 precast frame=%x", frames[10])
	}
	if frames[11][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[11][4:])) != 19 {
		t.Fatalf("msg19 open frame=%x", frames[11])
	}
	if frames[12][0] != 0x10 || binary.LittleEndian.Uint16(frames[12][12:]) != propRushCantMove {
		t.Fatalf("CantMove re-set frame=%x", frames[12])
	}
	if got := player.stateSnapshot().MP; got != beforeMP {
		t.Fatalf("zero-cost landing spent MP: %d want %d", got, beforeMP)
	}
	if player.rushInFlightSnapshot() {
		t.Fatal("landing kept rush in flight")
	}
	if player.peekPendingCast(time.Now()) != nil {
		t.Fatal("landing kept pending cast")
	}

	// 暂存已消费：第二次 212 外服私服零回应。
	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(6, 0, 7, 0), "test"); err != nil || !handled {
		t.Fatalf("repeat landing handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 18 {
		t.Fatalf("repeat landing frames=%d, want no extra (official silence)", got)
	}
}

// 211 距离四桶分发（g/f/b 定版）：≤直收带 executeRushDirect（首帧 0x10
// CurSkillID，无暂存）；其余一律暂存受理簇 5 帧（受理与距离解耦）。
func TestOutOfRangeStagedRushFamilySkillStagesViaSkill211(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off") // 本测试只验受理簇方言，关服务端驱动接近
	var definition combatSkillDefinition
	found := false
	for id, candidate := range installedCombatSkillCatalog.levelOne {
		if candidate.stagedRushFamily() && candidate.requiresTarget &&
			rushDirectCastMeters(candidate)+2 < float64(candidate.rushRange) {
			definition = candidate
			_ = id
			found = true
			break
		}
	}
	if !found {
		t.Skip("fixture catalog has no staged rush family selected-target skill with roomy rush range")
	}
	nearCast := rushDirectCastMeters(definition)

	build := func(npcX float32) (*captureMessageConnection, *playerActor, *sceneLifecycle) {
		conn := &captureMessageConnection{}
		world := newSceneLifecycle("test", conn)
		if err := world.begin("scene", "resource", nil); err != nil {
			t.Fatal(err)
		}
		registerTestCombatNPC(t, world, 91)
		placeTestNPC(t, world, 91, npcX, 0)
		player := newPlayerActor("冲锋侠", 0)
		player.mu.Lock()
		player.learnedSkills[definition.id] = 1
		player.mu.Unlock()
		if err := handleSelectTargetCustom(world, player, warningToggleMessage(clientCustomSelectTarget,
			clientCustomValue{Type: 8, Raw: packTestObject(forkNPCID(91), npcTestOwner(91))}), "test"); err != nil {
			t.Fatal(err)
		}
		return conn, player, world
	}

	// 接近带：受理簇 5 帧，无冲刺、无 CD 行（远位锚=GRushRange+Add=6.5
	// 与直收带取大后同 RushRange 取半，仅作「直收带外」摆放）。
	conn, player, world := build(float32((math.Max(rushGRushRangeMeters+1.5, nearCast) + float64(definition.rushRange)) / 2))
	base := len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, nil, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("staging cast handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	if len(frames) != 9 || frames[0][0] != 0x10 ||
		int32(binary.LittleEndian.Uint32(frames[2][4:])) != 25 ||
		binary.LittleEndian.Uint16(frames[3][12:]) != propRushPauseSkill ||
		int32(binary.LittleEndian.Uint32(frames[4][4:])) != 707 {
		t.Fatalf("staging frames=%x, want 9-frame acceptance cluster (stage 无 m24, 首受理带 PauseSkill)", frames)
	}
	if !player.rushInFlightSnapshot() {
		t.Fatal("staging did not mark rush in flight")
	}

	// 直收带外、地面冲锋带外不远处的按下（>方言窗上界）：同样只发受理
	// 5 帧（带内按压现受理即授 c61，见 TestRushGroundBandPressGrantsImmediately）。
	closeBeyond := math.Max(nearCast+1.5, rushDashDialectFor(definition).windowHi+1)
	conn, player, world = build(float32(closeBeyond))
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, nil, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("close cast handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 9 || int32(binary.LittleEndian.Uint32(frames[4][4:])) != 707 {
		t.Fatalf("close cast frames=%x, want acceptance cluster", frames)
	}
	if player.rushDashIssuedSnapshot() {
		t.Fatal("close cast issued dash at staging")
	}

	// 远距（超 RushRange）：受理与距离解耦——照常暂存。
	conn, player, world = build(definition.rushRange + 5)
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, nil, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("far cast handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 9 || int32(binary.LittleEndian.Uint32(frames[4][4:])) != 707 {
		t.Fatalf("far cast frames=%x, want acceptance (distance-decoupled)", frames)
	}
	if player.peekPendingCast(time.Now()) == nil {
		t.Fatal("far cast did not stage a pending skill")
	}

	// 直收带内（≤max(bind+2,3.5)）：executeRushDirect——首帧 0x10
	// CurSkillID（外服私服 g 直收簇，无 0x11/CD 首帧、无暂存）。
	conn, player, world = build(float32(math.Max(0.5, nearCast-1)))
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, nil, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("near cast handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) == 0 || frames[0][0] != 0x10 ||
		binary.LittleEndian.Uint16(frames[0][12:]) != propCurSkillID {
		head := frames
		if len(head) > 4 {
			head = head[:4]
		}
		t.Fatalf("near cast frames=%x, want direct cluster with CurSkillID first", head)
	}
	if player.peekPendingCast(time.Now()) != nil {
		t.Fatal("near cast staged a pending skill")
	}
}

// 冲锋族按压闸（2026-11-11）：个人 CD 在途时按下（远距原本会暂存受理、
// 近距原本会直收出招——commitRushDashCosts 只起 CD 不查）必须整体拒绝：
// 零帧、不挂暂存、不标记 in-flight。冷却结束后按压恢复受理。
func TestRushPressDuringPersonalCooldownRejected(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off") // 本测试只验受理簇方言，关服务端驱动接近
	var definition combatSkillDefinition
	found := false
	for id, candidate := range installedCombatSkillCatalog.levelOne {
		if candidate.stagedRushFamily() && candidate.requiresTarget &&
			rushDirectCastMeters(candidate)+2 < float64(candidate.rushRange) {
			definition = candidate
			_ = id
			found = true
			break
		}
	}
	if !found {
		t.Skip("fixture catalog has no staged rush family selected-target skill with roomy rush range")
	}
	nearCast := rushDirectCastMeters(definition)

	build := func(npcX float32) (*captureMessageConnection, *playerActor, *sceneLifecycle) {
		conn := &captureMessageConnection{}
		world := newSceneLifecycle("test", conn)
		if err := world.begin("scene", "resource", nil); err != nil {
			t.Fatal(err)
		}
		registerTestCombatNPC(t, world, 91)
		placeTestNPC(t, world, 91, npcX, 0)
		player := newPlayerActor("冲锋侠", 0)
		player.mu.Lock()
		player.learnedSkills[definition.id] = 1
		player.skillCooldowns[definition.id] = time.Now().Add(30 * time.Second)
		player.mu.Unlock()
		if err := handleSelectTargetCustom(world, player, warningToggleMessage(clientCustomSelectTarget,
			clientCustomValue{Type: 8, Raw: packTestObject(forkNPCID(91), npcTestOwner(91))}), "test"); err != nil {
			t.Fatal(err)
		}
		return conn, player, world
	}

	for name, npcX := range map[string]float32{
		"far-staging-band": float32(definition.rushRange + 5),
		"near-direct-band": float32(math.Max(0.5, nearCast-1)),
	} {
		conn, player, world := build(npcX)
		base := len(conn.Frames())
		if handled, err := handleSkillCustom(conn, player, world, nil, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
			t.Fatalf("%s press handled=%t err=%v", name, handled, err)
		}
		if frames := conn.Frames()[base:]; len(frames) != 0 {
			t.Fatalf("%s press sent %d frames during cooldown, want zero", name, len(frames))
		}
		if player.peekPendingCast(time.Now()) != nil {
			t.Fatalf("%s press staged a pending cast during cooldown", name)
		}
		if player.rushInFlightSnapshot() {
			t.Fatalf("%s press marked rush in flight during cooldown", name)
		}
	}

	// 冷却结束后（闸时间已过）：恢复受理（远距照常暂存）。
	conn, player, world := build(float32(definition.rushRange + 5))
	player.mu.Lock()
	player.skillCooldowns[definition.id] = time.Now().Add(-time.Second)
	player.mu.Unlock()
	base := len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, nil, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("expired-cooldown press handled=%t err=%v", handled, err)
	}
	if frames := conn.Frames()[base:]; len(frames) != 9 {
		t.Fatalf("expired-cooldown press frames=%d, want 9-frame acceptance cluster (stage 无 m24, 首受理带 PauseSkill)", len(frames))
	}
}

// msg15/msg220 接近探测 + 冲刺窗口 (5.0, 6.5]（g 5.54 触发 / b 3.61 不触发）：
// 窗口外零帧；窗口内 c61 冲刺簇；冲刺已发后静默；212 三分兜底保留。
func TestMovementUploadDrivesRushArrival(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off") // 本测试只验受理簇方言，关服务端驱动接近
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	placeTestNPC(t, world, 91, 10, 0)
	player := newPlayerActor("冲锋侠", 0)

	// 无冲锋在途：静默，零帧。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(1, 0, 1, 0), "test"); err != nil || !handled {
		t.Fatalf("cold movement handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("cold movement emitted %d frames", got)
	}

	definition := combatSkillDefinition{
		id: "CS_rush_unlearned", canRush: true, script: "skilllock", castRange: 3,
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction,
	}
	if _, err := stageRushCast(conn, player, nil, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames()) // 受理簇 5 帧为基数。

	// 距目标 9m（冲锋带 (3.5,5] 外）：外服私服方言——客户端自走（65210 会话
	// 23:02:27 外服私服簇下客户端实发 state7 实证），服务端零帧；旧的
	// c61/0x1F 服务端驱动已撤（瞬移根因）。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(2, 0, 1, 0), "test"); err != nil || !handled {
		t.Fatalf("far movement handled=%t err=%v", handled, err)
	}
	var frames [][]byte
	frames = conn.Frames()[base:]
	if len(frames) != 0 {
		t.Fatalf("far movement emitted %d frames=%x, want 0 (client self-approach)", len(frames), frames)
	}

	// 带外（7m，3m 族冲锋带 (3.5,6.5] 外）：仅官方回显，无驱动帧（含
	// 0x21 s5=25）。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(3, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("out-of-band movement handled=%t err=%v", handled, err)
	}
	for _, f := range conn.Frames()[base:] {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("out-of-band movement must not drive dash: %x", f)
		}
	}

	// 走带外 212 且未发冲刺（2026-11-12 深夜方言更新，Java b 线
	// handleDashLanding 同款）：无条件落地续招——54949 会话实证走带闸
	// 静默忽略致客户端狂发 212×278 无响应。客户端停点即其认定射程内，
	// 服务端按其上报直接 resume（pending 消费、落地簇照发）。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(0, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("far landing handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) == 0 {
		t.Fatal("far landing must resume pending cast (Java dialect: unconditional landing)")
	}
	if player.rushInFlightSnapshot() || player.peekPendingCast(time.Now()) != nil {
		t.Fatal("far landing must consume pending cast")
	}

	// 自冲落地（上报点距目标 ≤3m，未发 c61=步行落地形态）：
	// PauseSkill+msg26+LogicState 对+fight_relation+0x1F×1+337+c1=40+
	// msg19+CantMove/CantParry=11 帧（步行无 CantMove 清前导、单 0x1F）。
	// 按压位 7m=带外受理（带内按压现受理即授 c61，见
	// TestRushGroundBandPressGrantsImmediately）。
	if _, err := stageRushCast(conn, player, nil, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 3, Y: 0, Z: 0}, 7, "test"); err != nil {
		t.Fatal(err)
	}
	base = len(conn.Frames())
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("self-dash landing handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 15 || int32(binary.LittleEndian.Uint32(frames[8][4:])) != 19 {
		t.Fatalf("self-dash frames=%x, want 15-frame walk-in landing cluster+state-ping", frames)
	}
	if frames[0][0] != 0x10 || binary.LittleEndian.Uint16(frames[0][12:]) != propRushPauseSkill {
		t.Fatalf("walk-in landing starts with PauseSkill clear, got %x", frames[0])
	}
	if player.rushInFlightSnapshot() || player.peekPendingCast(time.Now()) != nil {
		t.Fatal("self-dash landing kept rush state alive")
	}

	// 重新暂存后走完整冲锋链：上报点距目标 4m ∈ 3m 族冲锋带 (3.5,5] →
	// v29 c61 授予簇（c61 + CantMove/CantParry=1 + CD/MP + State=run）+
	// 0x21 s5=25 驱动，终点=目标身前 bind（v30 二轮澄清：同层平地带内
	// 照常地面冲锋，高度规则只管空/地混层）。
	if _, err := stageRushCast(conn, player, nil, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base = len(conn.Frames())
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(6, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("dash movement handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	var sawC61, sawCantMove bool
	for _, f := range frames {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("v31 client self-run must not emit 0x21 drive entries: %x", f)
		}
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 3 && values[0].Type == 2 && values[0].Int32 == 24 &&
				values[2].Type == 2 && values[2].Int32 == msgRushCommand {
				sawC61 = true // v31：c61 授予即冲锋（客户端自走，零位移驱动）
			}
		}
		if f[0] == 0x10 && len(f) >= 15 &&
			binary.LittleEndian.Uint16(f[12:]) == propRushCantMove && f[14] == 1 {
			sawCantMove = true // 授予簇 CantMove=1（官方同毫秒对）
		}
	}
	if !sawC61 {
		t.Fatalf("dash frames=%x, want c61 grant (no 0x21 dialect)", frames)
	}
	if !sawCantMove {
		t.Fatalf("dash frames=%x, want grant cluster CantMove=1", frames)
	}
	if !player.rushDashIssuedSnapshot() {
		t.Fatal("approach did not issue dash")
	}
	base = len(conn.Frames())

	// 冲刺在途再上报：静默——落地只认 212。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("in-dash movement handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("in-dash movement emitted %d frames, want 0", got)
	}

	// 212 冲刺落地：14 帧外服私服簇（定义未习得 → 伤害链不启动）。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("landing handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 18 ||
		frames[0][0] != 0x10 || binary.LittleEndian.Uint16(frames[0][12:]) != propRushCantMove ||
		int32(binary.LittleEndian.Uint32(frames[3][4:])) != 26 ||
		frames[7][0] != 0x1F || frames[8][0] != 0x1F ||
		int32(binary.LittleEndian.Uint32(frames[9][4:])) != 337 ||
		int32(binary.LittleEndian.Uint32(frames[10][18:])) != 40 ||
		int32(binary.LittleEndian.Uint32(frames[11][4:])) != 19 {
		t.Fatalf("landing frames=%x, want 18-frame dash landing cluster+state-ping", frames)
	}
	if player.rushInFlightSnapshot() || player.peekPendingCast(time.Now()) != nil {
		t.Fatal("landing kept rush state alive")
	}
}

// msg220 冲锋在途位移上报（4×f32）：窗口内同款 c61 簇；非在途静默。
func TestRushMovementUpload220DrivesArrival(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off") // 本测试只验受理簇方言，关服务端驱动接近
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	placeTestNPC(t, world, 91, 10, 0)
	player := newPlayerActor("冲锋侠", 0)

	upload220 := clientCustomMessage{Opcode: 0x1E, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomRushMoveUpload},
		{Type: 4, Float32: 6},
		{Type: 4, Float32: 0},
		{Type: 4, Float32: 0},
		{Type: 4, Float32: 0.5},
	}}
	// 无冲锋在途：静默。
	if handled, err := handleRushMovementUpload(conn, player, world, upload220, "test"); err != nil || !handled {
		t.Fatalf("cold 220 handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("cold 220 emitted %d frames", got)
	}

	definition := combatSkillDefinition{
		id: "CS_rush_unlearned", canRush: true, script: "skilllock", castRange: 3,
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction,
	}
	if _, err := stageRushCast(conn, player, nil, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames())

	// 距目标 4m ∈ 3m 族冲锋带 (3.5,5]：c61 授予簇 + 0x21 驱动。
	if handled, err := handleRushMovementUpload(conn, player, world, upload220, "test"); err != nil || !handled {
		t.Fatalf("dash 220 handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	var sawC61 bool
	for _, f := range frames {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("v31 client self-run must not emit 0x21 drive entries: %x", f)
		}
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 3 && values[0].Type == 2 && values[0].Int32 == 24 &&
				values[2].Type == 2 && values[2].Int32 == msgRushCommand {
				sawC61 = true
			}
		}
	}
	if !sawC61 {
		t.Fatalf("dash 220 frames=%x, want c61 grant (no 0x21 dialect)", frames)
	}
	if !player.rushDashIssuedSnapshot() {
		t.Fatal("220 approach did not issue dash")
	}
	base = len(conn.Frames())

	// 冲刺在途再上报：静默（落地只认 212）。
	arrive220 := clientCustomMessage{Opcode: 0x1E, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomRushMoveUpload},
		{Type: 4, Float32: 8.5},
		{Type: 4, Float32: 0},
		{Type: 4, Float32: 0},
		{Type: 4, Float32: 0.5},
	}}
	if handled, err := handleRushMovementUpload(conn, player, world, arrive220, "test"); err != nil || !handled {
		t.Fatalf("in-dash 220 handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("in-dash 220 emitted %d frames, want 0", got)
	}

	// 212 冲刺落地：14 帧外服私服簇（同 msg15 链）。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("landing handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 18 || int32(binary.LittleEndian.Uint32(frames[11][4:])) != 19 {
		t.Fatalf("landing after 220 frames=%x, want 18-frame cluster+state-ping", frames)
	}
	if player.rushInFlightSnapshot() || player.peekPendingCast(time.Now()) != nil {
		t.Fatal("landing after 220 kept rush state alive")
	}
}

func packTestObject(objectID, ownerID uint32) [8]byte {
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], uint64(objectID)|uint64(ownerID)<<32)
	return raw
}

// movementUploadOrient：仅 state7 长式 idx9 是绝对朝向；state4 停帧 idx6
// 为增量、跳帧无朝向——一律 0（技能.pcapng：外服私服 state7 [15,id,7,x,y,z,
// destx,desty,4.50,orient,clock]，旧实现取 idx6=destx 属字段错位）。
func TestMovementUploadOrientExtraction(t *testing.T) {
	walk := []clientCustomValue{
		{Type: 2, Int32: clientCustomMovementUpload},
		{Type: 2, Int32: -1610608895}, // 0xA0000F01
		{Type: 2, Int32: 7},
		{Type: 4, Float32: 950}, {Type: 4, Float32: 43}, {Type: 4, Float32: 1029},
		{Type: 4, Float32: 947}, {Type: 4, Float32: 998}, // destx/desty
		{Type: 4, Float32: 4.5}, {Type: 4, Float32: 3.14},
	}
	if got := movementUploadOrient(walk); got != 3.14 {
		t.Fatalf("state7 orient = %v, want 3.14", got)
	}
	stop := []clientCustomValue{
		{Type: 2, Int32: clientCustomMovementUpload},
		{Type: 2, Int32: -1610612736}, // 0xA0000000
		{Type: 2, Int32: 4},
		{Type: 4, Float32: 950}, {Type: 4, Float32: 43}, {Type: 4, Float32: 1029},
		{Type: 4, Float32: -1.07}, // 增量朝向，不可用
	}
	if got := movementUploadOrient(stop); got != 0 {
		t.Fatalf("state4 orient = %v, want 0", got)
	}
}

// useSkillGroundPointTransform：211 短式 idx6..8 地面点（范围型 hyjf07
// 点地形态；锁定式三值为 0 须拒绝）。
func TestUseSkillGroundPointTransform(t *testing.T) {
	withGround := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 211}, {Type: 6, Text: "CS_jh_hyjf07"},
		{Type: 4, Float32: 957}, {Type: 4, Float32: 43}, {Type: 4, Float32: 1045},
		{Type: 4, Float32: 1.76},
		{Type: 4, Float32: 954}, {Type: 4, Float32: 43}, {Type: 4, Float32: 1049},
	}}
	ground, ok := useSkillGroundPointTransform(withGround)
	if !ok || ground.X != 954 || ground.Z != 1049 {
		t.Fatalf("ground = %+v ok=%t, want (954,1049)", ground, ok)
	}
	locked := clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: 211}, {Type: 6, Text: "CS_jh_hyjf02"},
		{Type: 4, Float32: 977}, {Type: 4, Float32: 43}, {Type: 4, Float32: 1029},
		{Type: 4, Float32: -1.61},
		{Type: 4, Float32: 0}, {Type: 4, Float32: 0}, {Type: 4, Float32: 0},
	}}
	if _, ok := useSkillGroundPointTransform(locked); ok {
		t.Fatal("locked-form ground point should be rejected (all zero)")
	}
}

// 范围型点地冲刺（v31 官方自走模型，2026-11-17 用户令「去掉 0x21」）：
// ≤3.5 直收零帧；落点=点选平面点本体；簇=纯 c61（观察者另收 AOI 裸广播）
// ——地面 flag1 / 高→低·低→高 flag2，无 106/State=run/0x21 位移。
func TestAreaGroundRushDashTierSelection(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_SYNC_PAIR", "1") // v31.2 同步条目默认关；本测试验证开态方言
	definition := combatSkillDefinition{
		id: "CS_jh_hyjf07", canRush: true, castRange: 15,
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction,
		airRushAction: "qg_fastruning_s_2h_01",
	}
	newCase := func(castAt, ground worldcore.Transform) *captureMessageConnection {
		conn := &captureMessageConnection{}
		issueAreaGroundRushDash(conn, newPlayerActor("点地侠", 0), nil, definition, castAt, ground, "test")
		return conn
	}
	c61Of := func(t *testing.T, frames [][]byte, wantFlag int32, wantDestX float32, name string) {
		t.Helper()
		var c61 []clientCustomValue
		for _, f := range frames {
			if f[0] != 0x1E {
				continue
			}
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 3 && values[0].Int32 == 24 && values[2].Int32 == msgRushCommand {
				c61 = values
			}
		}
		if c61 == nil {
			t.Fatalf("%s: no c61 in frames=%x", name, frames)
		}
		if c61[6].Type != 2 || c61[6].Int32 != wantFlag {
			t.Fatalf("%s: c61 flag=%d, want %d", name, c61[6].Int32, wantFlag)
		}
		if math.Abs(float64(c61[3].Float32-wantDestX)) > 0.01 {
			t.Fatalf("%s: c61 dest x=%v, want %v (clicked point itself)", name, c61[3].Float32, wantDestX)
		}
	}

	conn := newCase(worldcore.Transform{}, worldcore.Transform{X: 2}) // ≤3.5：直收，主路径处理，零帧。
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("near-ground emitted %d frames, want 0", got)
	}

	// syncOf 校验 v31.2 成对 0x21 冲刺同步条目（47B：intent/s5/dest.x）。
	syncOf := func(t *testing.T, frames [][]byte, wantIntent uint32, wantSpeed, wantDestX float32, name string) {
		t.Helper()
		for _, f := range frames {
			if f[0] != 0x21 || len(f) != 47 {
				continue
			}
			if got := binary.LittleEndian.Uint32(f[43:]); got != wantIntent {
				t.Fatalf("%s: sync intent=0x%X, want 0x%X", name, got, wantIntent)
			}
			if got := math.Float32frombits(binary.LittleEndian.Uint32(f[27:])); math.Abs(float64(got-wantSpeed)) > 0.01 {
				t.Fatalf("%s: sync s5=%v, want %v", name, got, wantSpeed)
			}
			if got := math.Float32frombits(binary.LittleEndian.Uint32(f[11:])); math.Abs(float64(got-wantDestX)) > 0.01 {
				t.Fatalf("%s: sync dest x=%v, want %v", name, got, wantDestX)
			}
			return
		}
		t.Fatalf("%s: no 47B 0x21 sync entry in frames=%x", name, frames)
	}

	// v31.2 地面：c61 flag1（dest.y=点+1.0）+ 成对 0x21 同步条目
	//（s5=35、intent=0x4701、dest.xz=点选点本体、y=平面）。
	conn = newCase(worldcore.Transform{}, worldcore.Transform{X: 5})
	frames := conn.Frames()
	if len(frames) != 2 || frames[0][0] != 0x1E || frames[1][0] != 0x21 || len(frames[1]) != 47 {
		t.Fatalf("area ground frames=%x, want [c61]+[0x21 sync 47B]", frames)
	}
	c61Of(t, frames, 1, 5, "area ground")
	syncOf(t, frames, 0x4701, rushAreaSyncSpeedMeters, 5, "area ground")

	// v31.2 高→低（落差 4m）：c61 flag2 + 同步条 s5=15/intent=0x4704。
	conn = newCase(worldcore.Transform{Y: 4}, worldcore.Transform{X: 8})
	frames2 := conn.Frames()
	if len(frames2) != 2 || frames2[0][0] != 0x1E || frames2[1][0] != 0x21 {
		t.Fatalf("area air frames=%x, want [c61]+[0x21 sync]", frames2)
	}
	c61Of(t, frames2, 2, 8, "area air")
	syncOf(t, frames2, 0x4704, rushDriveAirDashSpeedMeters, 8, "area air")

	// v31.2 低→高（落差 2m，用户补充「范围冲锋也是一样」）：c61 flag2。
	conn = newCase(worldcore.Transform{}, worldcore.Transform{X: 6, Y: 2})
	framesUp := conn.Frames()
	if len(framesUp) != 2 || framesUp[0][0] != 0x1E {
		t.Fatalf("area uplift frames=%x, want [c61]+[0x21 sync]", framesUp)
	}
	c61Of(t, framesUp, 2, 6, "area uplift")
	syncOf(t, framesUp, 0x4704, rushDriveAirDashSpeedMeters, 6, "area uplift")

	// 远距同款（12m）。
	conn = newCase(worldcore.Transform{}, worldcore.Transform{X: 12})
	frames3 := conn.Frames()
	if len(frames3) != 2 || frames3[1][0] != 0x21 || len(frames3[1]) != 47 {
		t.Fatalf("area far frames=%x, want [c61]+[0x21 sync 47B]", frames3)
	}
	c61Of(t, frames3, 1, 12, "area far")
	syncOf(t, frames3, 0x4701, rushAreaSyncSpeedMeters, 12, "area far")
}

// TestRushDashBandDrive v24 终版（用户拍板）：受理簇纯官方 9 帧（服务端
// 不再控制步行）；客户端移动上报进入冲刺带 (3,5] 触发 25m/s 冲刺驱动
// （逃逸门方言 [106]+[0x21]），ETA 到点 bind（1~1.5m）直接释放。
func TestRushDashBandDrive(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_SYNC_PAIR", "1") // v31.2 同步条目默认关；本测试验证开态 91B 方言
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	placeTestNPC(t, world, 91, 20, 0)
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_far", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 3, groundRushAction: defaultGroundRushAction}
	player.mu.Lock()
	player.learnedSkills[definition.id] = 1
	player.mu.Unlock()
	if err := handleSelectTargetCustom(world, player, warningToggleMessage(clientCustomSelectTarget,
		clientCustomValue{Type: 8, Raw: packTestObject(forkNPCID(91), npcTestOwner(91))}), "test"); err != nil {
		t.Fatal(err)
	}

	if _, err := stageRushCast(conn, player, world, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 20, Y: 0, Z: 0}, worldcore.Transform{X: 0, Y: 0, Z: 0}, 20, "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()); got != 12 { // 选靶应答2 + 纯官方受理簇9（步行驱动已撤）
		t.Fatalf("staging frames=%d, want 11", got)
	}

	// 客户端自走进入冲刺带：位置 (16,0,0) → 距目标 4m ∈ (3.5,6.5]（同层
	// 平地；v30 高度规则只管空/地混层）。
	base := len(conn.Frames())
	if _, err := handleMovementUpload(conn, player, world,
		walkUploadMessage(-1610608895, 7, 16, 0, 0, 20, 0, 4.5, 0), "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()[base:]
	var sawC61, sawSync91 bool
	for _, f := range frames {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("v31 client self-run must not emit 0x21 drive entries: %x", f)
		}
		// v31.2 成对 91B 双条目：entry1=s5=25/intent=0x4701、dest=bind xz；
		// entry2（f[87:91]）=空置回显 0xA0000700（dest=按压位 16,0,0）。
		if f[0] == 0x21 && len(f) == 91 && binary.LittleEndian.Uint32(f[43:47]) == 0x4701 &&
			math.Abs(float64(math.Float32frombits(binary.LittleEndian.Uint32(f[27:]))-rushDriveDashSpeedMeters)) < 0.01 &&
			math.Abs(float64(math.Float32frombits(binary.LittleEndian.Uint32(f[11:]))-18.5)) < 0.05 &&
			binary.LittleEndian.Uint32(f[87:91]) == 0xA0000700 {
			sawSync91 = true
		}
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 8 && values[0].Type == 2 && values[0].Int32 == 24 &&
				values[2].Type == 2 && values[2].Int32 == msgRushCommand &&
				values[6].Type == 2 && values[6].Int32 == 1 {
				// 地面档 flag1；dest=目标身前 bind（20-1.5=18.5）。
				if math.Abs(float64(values[3].Float32-18.5)) > 0.05 {
					t.Fatalf("c61 dest x=%v, want bind 18.5", values[3].Float32)
				}
				sawC61 = true
			}
		}
	}
	if !sawC61 {
		t.Fatalf("dash band must grant c61 flag1 dest=bind, frames=%x", frames)
	}
	if !sawSync91 {
		t.Fatalf("dash band must carry v31.2 paired 91B sync (0x4701 s5=25 + idle echo), frames=%x", frames)
	}
	if !player.rushDashIssuedSnapshot() {
		t.Fatal("dash band must mark dash issued")
	}

	// v31：释放主通道=m212 到位申报（客户端自走到 bind 后上报）——落地
	// 簇 0x1F 锚在 212 自报位；呆坐哨（≥1.5s）只是 c61 未消费的兜底，
	// 本测试直接走 212 链。
	if handled, err := handleCastDashLanding(conn, player, world,
		warningToggleMessage(clientCustomCastDashLanding,
			clientCustomValue{Type: 4, Float32: 18.5}, clientCustomValue{Type: 4, Float32: 0},
			clientCustomValue{Type: 4, Float32: 0}, clientCustomValue{Type: 4, Float32: 0.5},
			clientCustomValue{Type: 2, Int32: 1}), "test"); err != nil || !handled {
		t.Fatalf("dash landing handled=%t err=%v", handled, err)
	}
	sawLanding := false
	for _, f := range conn.Frames() {
		if f[0] == 0x1F {
			sawLanding = true
		}
	}
	if !sawLanding {
		t.Fatal("dash 212 must land (0x1F anchors)")
	}
	if player.peekPendingCast(time.Now()) != nil || player.rushInFlightSnapshot() {
		t.Fatal("dash landing must consume pending")
	}
	time.Sleep(50 * time.Millisecond) // 排空 m707/呆坐哨 AfterFunc 检查窗外的残留定时器
}

// v29 分档带族规则（2026-11-16 官服观察定版）：直收带/冲锋带按释放距离
// 分族——3m 族直收 castRange+0.5、冲锋带 (castRange+0.5, GRushRange 5]；
// 6m 族直收 castRange+1、无冲锋带；方言表技能（xytf02 等）表窗优先。
func TestRushChargeBandFamilySelection(t *testing.T) {
	three := combatSkillDefinition{id: "CS_three", canRush: true, castRange: 3, bindDistance: defaultRushBindDistance}
	if got := rushDirectCastMeters(three); math.Abs(got-3.5) > 1e-6 {
		t.Fatalf("3m family direct band=%v, want 3.5", got)
	}
	dialect := rushDashDialectFor(three)
	if math.Abs(dialect.windowLo-3.5) > 1e-6 || math.Abs(dialect.windowHi-rushGroundChargeHiMeters) > 1e-6 {
		t.Fatalf("3m family charge band=%+v, want (3.5, 6.5]", dialect)
	}
	six := combatSkillDefinition{id: "CS_six", canRush: true, castRange: 6, bindDistance: defaultRushBindDistance}
	if got := rushDirectCastMeters(six); math.Abs(got-7) > 1e-6 {
		t.Fatalf("6m family direct band=%v, want 7 (hyjf02 6.95 直收实证)", got)
	}
	if dialect := rushDashDialectFor(six); dialect.windowLo != 0 || dialect.windowHi != 0 {
		t.Fatalf("6m family must have no charge band, got %+v", dialect)
	}
	tabled := combatSkillDefinition{id: "CS_jz_xytf02", canRush: true, castRange: 3, bindDistance: defaultRushBindDistance}
	if dialect := rushDashDialectFor(tabled); math.Abs(dialect.windowLo-5.0) > 1e-6 || math.Abs(dialect.windowHi-7.0) > 1e-6 ||
		dialect.level != 2 || dialect.action != "qg_fastruning_s_0h_01" {
		t.Fatalf("dialect table must take priority, got %+v", dialect)
	}
}

// v29 武技冲锋扩程（2026-11-16）：RushRange 列实据/手勘表把 3m 冲锋扩到
// 20m（落英飞花·落尘 wuji_CS_jz_lyfh05）；lua rush 指令字段级覆盖
//（mergeRushSpec）；6m 族被扩程时下界回落直收带；缺列回退值不触发扩程。
func TestRushChargeWujiOverride(t *testing.T) {
	// 手勘表实据：wuji_lyfh05（INI 缺列 raw=0）→ chargeHi=20。
	wuji := resolveRushChargeOverride("wuji_CS_jz_lyfh05", 3, 0)
	if wuji == nil || math.Abs(wuji.chargeHi-20) > 1e-6 {
		t.Fatalf("wuji seed override = %+v, want chargeHi=20", wuji)
	}
	def := combatSkillDefinition{id: "wuji_CS_jz_lyfh05", canRush: true, castRange: 3,
		bindDistance: defaultRushBindDistance, rushOverride: wuji}
	if dialect := rushDashDialectFor(def); math.Abs(dialect.windowLo-3.5) > 1e-6 || math.Abs(dialect.windowHi-20) > 1e-6 {
		t.Fatalf("wuji dialect = %+v, want band (3.5, 20]", dialect)
	}
	// INI RushRange 列实据直接生效（raw=18）；普通技缺列（回退值）不扩程。
	if o := resolveRushChargeOverride("CS_ranged_wuji", 3, 18); o == nil || math.Abs(o.chargeHi-18) > 1e-6 {
		t.Fatalf("ranged override = %+v, want chargeHi=18", o)
	}
	if o := resolveRushChargeOverride("CS_plain", 3, 0); o != nil {
		t.Fatalf("plain 3m skill must not extend, got %+v", o)
	}
	if o := resolveRushChargeOverride("CS_six_plain", 6, 0); o != nil {
		t.Fatalf("plain 6m skill must not extend, got %+v", o)
	}
	// lua 字段级覆盖（mergeRushSpec 直接验证——宿主装载在 skillscript 包测试）。
	luaOverridden := mergeRushSpec(nil, &skillscript.RushSpec{
		DirectBandM: 2, ChargeLoM: 4, ChargeHiM: 16, Level: 2,
		Action: "qg_fastrun_0h_01", AirAction: "qg_fastruning_s_0h_01",
	})
	if luaOverridden.directBand != 2 || luaOverridden.chargeLo != 4 ||
		luaOverridden.chargeHi != 16 || luaOverridden.level != 2 ||
		luaOverridden.groundAction != "qg_fastrun_0h_01" || luaOverridden.airAction != "qg_fastruning_s_0h_01" {
		t.Fatalf("lua merge = %+v", luaOverridden)
	}
	luaDef := combatSkillDefinition{id: "CS_lua_wuji", canRush: true, castRange: 3,
		bindDistance: defaultRushBindDistance, rushOverride: luaOverridden}
	if got := rushDirectCastMeters(luaDef); math.Abs(got-2) > 1e-6 {
		t.Fatalf("lua direct band = %v, want 2", got)
	}
	if dialect := rushDashDialectFor(luaDef); math.Abs(dialect.windowLo-4) > 1e-6 ||
		math.Abs(dialect.windowHi-16) > 1e-6 || dialect.level != 2 || dialect.action != "qg_fastrun_0h_01" {
		t.Fatalf("lua dialect = %+v, want (4,16] level2 action0h", dialect)
	}
	// 6m 族被扩程：无带底窗扩到 (7, 20]。
	sixDef := combatSkillDefinition{id: "CS_six_wuji", canRush: true, castRange: 6,
		bindDistance: defaultRushBindDistance, rushOverride: &rushChargeOverride{chargeHi: 20}}
	if dialect := rushDashDialectFor(sixDef); math.Abs(dialect.windowLo-7) > 1e-6 || math.Abs(dialect.windowHi-20) > 1e-6 {
		t.Fatalf("6m wuji dialect = %+v, want band (7, 20]", dialect)
	}
}

// 6m 族无冲锋带（v29）：距目标 4m 的移动上报不触发冲刺（无 c61/无 0x21
// 驱动）；停点 212 照常落地续招。
func TestRushSixMeterFamilyNoChargeBand(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off")
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	placeTestNPC(t, world, 91, 20, 0)
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_six", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 6, groundRushAction: defaultGroundRushAction}
	if _, err := stageRushCast(conn, player, nil, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 20, Y: 0, Z: 0}, worldcore.Transform{X: 0, Y: 0, Z: 0}, 20, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames())

	// 距目标 4m（旧全局带 (3,5] 会误触发；v29 6m 族无带）：零驱动帧。
	if _, err := handleMovementUpload(conn, player, world,
		walkUploadMessage(-1610608895, 7, 16, 0, 0, 20, 0, 4.5, 0), "test"); err != nil {
		t.Fatal(err)
	}
	for _, f := range conn.Frames()[base:] {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("6m family must not drive dash: %x", f)
		}
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 3 && values[0].Type == 2 && values[0].Int32 == 24 && values[2].Int32 == msgRushCommand {
				t.Fatalf("6m family must not grant c61: %x", f)
			}
		}
	}
	if player.rushDashIssuedSnapshot() {
		t.Fatal("6m family must not issue dash")
	}

	// 停点 212：走带认可半径 max(5, castRange+1)=7 → 落地续招（消费 pending）。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(16, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("walk-in landing handled=%t err=%v", handled, err)
	}
	if player.peekPendingCast(time.Now()) != nil || player.rushInFlightSnapshot() {
		t.Fatal("walk-in landing must consume pending")
	}
}

// v31 低→高空中奔袭档（2026-11-17 去 0x21）：目标高出施法者 >0.5m 的
// 冲锋走空中档——c61 flag2 + fastruning 动画，终点 y=目标平面（无 +1.0
// 抬升）；零 0x21 驱动帧（客户端自走），空中授予随帧 m707[29]。
func TestRushUpliftChargeAirTier(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off")
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	player := newPlayerActor("低冲高侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_up", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 3,
		groundRushAction: defaultGroundRushAction, airRushAction: "qg_fastruning_s_2h_01"}
	// 带外受理（distance 6 > 直收带 3.5，不触发立即冲锋），目标在 2m 高台。
	if _, err := stageRushCast(conn, player, world, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 2, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 6, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames())

	// 平距 4m ∈ (3.5,5] 带内照常触发；低→高 2m>0.5m → 空中奔袭档。
	if _, err := handleMovementUpload(conn, player, world,
		walkUploadMessage(-1610608895, 7, 6, 0, 0, 10, 0, 4.5, 0), "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()[base:]
	var sawC61Uplift, sawRun bool
	for _, f := range frames {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("v31 client self-run must not emit 0x21 drive entries: %x", f)
		}
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 8 && values[0].Type == 2 && values[0].Int32 == 24 &&
				values[2].Type == 2 && values[2].Int32 == msgRushCommand &&
				values[6].Type == 2 && values[6].Int32 == 2 &&
				values[7].Type == 6 && values[7].Text == "qg_fastruning_s_2h_01" {
				// 终点 y=目标平面 2.0（低→高无 +1.0 抬升）。
				if math.Abs(float64(values[4].Float32-2)) > 0.1 {
					t.Fatalf("uplift c61 dest y=%v, want target plane 2", values[4].Float32)
				}
				sawC61Uplift = true
			}
		}
		if f[0] == 0x10 && len(f) >= 18 &&
			binary.LittleEndian.Uint16(f[12:]) == npcStatePropertyIndex && bytes.Contains(f, []byte("run")) {
			sawRun = true
		}
	}
	if !sawC61Uplift {
		t.Fatalf("uplift charge must grant c61 flag=2 fastruning, frames=%x", frames)
	}
	if sawRun {
		t.Fatal("uplift charge must not carry State=run (aerial, not ground run)")
	}
	// 空中授予随帧 m707[29]（+40~80ms）。
	time.Sleep(200 * time.Millisecond)
	saw707 := false
	for _, f := range conn.Frames()[base:] {
		if f[0] != 0x1E {
			continue
		}
		values := decodeCustomValuesFrame(t, f)
		if len(values) >= 2 && values[0].Type == 2 && values[0].Int32 == 707 &&
			values[1].Type == 2 && values[1].Int32 == 29 {
			saw707 = true
		}
	}
	if !saw707 {
		t.Fatal("uplift air grant must follow with m707[29]")
	}
	time.Sleep(700 * time.Millisecond) // 排空呆坐哨 AfterFunc
}

// v30 近身高差立即冲锋（直发 stageRushCast 层）：直收带内但 Δy>1m——
// 受理簇后立即冲锋（低→高 2m → 空中奔袭 15m/s），ETA 到点释放。
func TestNearHeightGapStagesAndChargesImmediately(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off")
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	player := newPlayerActor("台侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_near", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 3,
		groundRushAction: defaultGroundRushAction, airRushAction: "qg_fastruning_s_2h_01"}
	base := len(conn.Frames())
	// 平距 2m ≤ 直收带 3.5（v29 会直收原地普攻），目标高 2m → 立即冲锋。
	if handled, err := stageRushCast(conn, player, world, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 4, Y: 2, Z: 0}, worldcore.Transform{X: 2, Y: 0, Z: 0}, 2, "test"); err != nil || !handled {
		t.Fatalf("stage handled=%t err=%v", handled, err)
	}
	if !player.rushDashIssuedSnapshot() {
		t.Fatal("near height-gap cast must issue dash at staging")
	}
	sawC61Air := false
	for _, f := range conn.Frames()[base:] {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("v31 client self-run must not emit 0x21 drive entries: %x", f)
		}
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 7 && values[0].Type == 2 && values[0].Int32 == 24 &&
				values[2].Type == 2 && values[2].Int32 == msgRushCommand &&
				values[6].Type == 2 && values[6].Int32 == 2 {
				sawC61Air = true // v31：空中档 c61 flag2 授予即冲锋（客户端自走）
			}
		}
	}
	if !sawC61Air {
		t.Fatal("near height-gap cast must grant aerial c61 flag=2 (uplift tier)")
	}
	// 呆坐哨（≥1.5s，c61 未消费兜底）后在客户端上报位就地释放。
	time.Sleep(1800 * time.Millisecond)
	if player.peekPendingCast(time.Now()) != nil {
		t.Fatal("idle-fallback watchdog must consume pending")
	}
}

// 地面冲锋带按压即授（规格 §二/§三；2026-11-17 真机「地面带无效果」修复）：
// 同层（|Δy|≤混层线）且按压 2D 距离已在 (直收带, GRushRange 5] 内——本服
// 登录方言下客户端无「按下即走」，站立按压零 m15/m220 上报，进带判定须以
// 按压 211 位置即刻完成（官服「3~5m 立即冲锋」）；6m 族同距无带不授。
func TestRushGroundBandPressGrantsImmediately(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off")
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	placeTestNPC(t, world, 91, 8, 0)
	player := newPlayerActor("带侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_band", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 3, groundRushAction: defaultGroundRushAction}
	base := len(conn.Frames())
	// 同层平距 4m ∈ (3.5, 5]：受理簇尾直接授予地面档 c61（flag1、dest=目标
	// 身前 bind 8-1.5=6.5）。
	if handled, err := stageRushCast(conn, player, world, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 8, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 4, "test"); err != nil || !handled {
		t.Fatalf("stage handled=%t err=%v", handled, err)
	}
	if !player.rushDashIssuedSnapshot() {
		t.Fatal("in-band ground press must issue dash at staging")
	}
	if player.rushDashAirborneSnapshot() {
		t.Fatal("same-layer press must grant ground tier, not air")
	}
	sawC61Ground := false
	for _, f := range conn.Frames()[base:] {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("v31 client self-run must not emit 0x21 drive entries: %x", f)
		}
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 7 && values[0].Type == 2 && values[0].Int32 == 24 &&
				values[2].Type == 2 && values[2].Int32 == msgRushCommand &&
				values[6].Type == 2 && values[6].Int32 == 1 {
				if math.Abs(float64(values[3].Float32-6.5)) > 0.05 {
					t.Fatalf("c61 dest x=%v, want bind 6.5", values[3].Float32)
				}
				sawC61Ground = true // 地面档 flag1（受理即带内 → 受理簇尾授予）
			}
		}
	}
	if !sawC61Ground {
		t.Fatal("in-band ground press must grant c61 flag=1 at staging")
	}
	time.Sleep(50 * time.Millisecond) // 排空 m707 AfterFunc（ETA 呆坐哨在测试窗外）

	// 6m 族同距无带（v29）：受理不授（仅 3m 族有地面带）。
	conn6 := &captureMessageConnection{}
	player6 := newPlayerActor("六米侠", 0)
	six := combatSkillDefinition{id: "CS_rush_band6", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 6, groundRushAction: defaultGroundRushAction}
	if _, err := stageRushCast(conn6, player6, world, six, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 8, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 4, "test"); err != nil {
		t.Fatal(err)
	}
	if player6.rushDashIssuedSnapshot() {
		t.Fatal("6m family has no ground band: press must not issue dash")
	}
	for _, f := range conn6.Frames() {
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 3 && values[0].Type == 2 && values[0].Int32 == 24 && values[2].Int32 == msgRushCommand {
				t.Fatalf("6m family must not grant c61 at staging: %x", f)
			}
		}
	}
	time.Sleep(50 * time.Millisecond)

	// 真机回归锚（2026-11-17 22:51「地面还是没有冲锋」）：客户端自动接近
	// 稳定停在 5.19~5.43m 等授予并狂发 212——v29 hi=GRushRange 5.0 把带卡
	// 死在停点外；带扩到 (直收带, GRushRange+Add 6.5] 后停点 m220 即授予。
	connA := &captureMessageConnection{}
	worldA := newSceneLifecycle("test", connA)
	if err := worldA.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, worldA, 91)
	playerA := newPlayerActor("自走侠", 0)
	auto := combatSkillDefinition{id: "CS_rush_auto", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 3, groundRushAction: defaultGroundRushAction}
	if _, err := stageRushCast(connA, playerA, worldA, auto, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 8, Y: 0, Z: 0}, worldcore.Transform{X: 0, Y: 0, Z: 0}, 8, "test"); err != nil {
		t.Fatal(err)
	}
	baseA := len(connA.Frames())
	// 自走停点 5.43m（真机 22:51:45 样本复刻）→ m220 停点上报 ∈ (3.5,6.5]。
	stop220 := clientCustomMessage{Opcode: 0x1E, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomRushMoveUpload},
		{Type: 4, Float32: 2.57}, {Type: 4, Float32: 0}, {Type: 4, Float32: 0},
		{Type: 4, Float32: 0.5},
	}}
	if handled, err := handleRushMovementUpload(connA, playerA, worldA, stop220, "test"); err != nil || !handled {
		t.Fatalf("autopath-stop 220 handled=%t err=%v", handled, err)
	}
	sawStopGrant := false
	for _, f := range connA.Frames()[baseA:] {
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 3 && values[0].Type == 2 && values[0].Int32 == 24 &&
				values[2].Type == 2 && values[2].Int32 == msgRushCommand {
				sawStopGrant = true
			}
		}
	}
	if !sawStopGrant || !playerA.rushDashIssuedSnapshot() {
		t.Fatal("autopath stop at 5.43m must grant c61 (band hi=GRushRange+Add 6.5)")
	}
	time.Sleep(50 * time.Millisecond)

	// 高→低 0.8m（旧 1.5 线下=地面档；2026-11-17 用户拍板双向对称 0.5）：
	// 空中档=按压不即授（m220/m15 驱动），走进带授予 c61 flag2 空中奔袭、
	// dest.y=目标 y+1.0 悬停弧线（唯低→高空中档=平面本体）。
	connS := &captureMessageConnection{}
	worldS := newSceneLifecycle("test", connS)
	if err := worldS.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, worldS, 91)
	playerS := newPlayerActor("浅降侠", 0)
	shallow := combatSkillDefinition{id: "CS_rush_shallow", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 3,
		groundRushAction: defaultGroundRushAction, airRushAction: "qg_fastruning_s_2h_01"}
	if _, err := stageRushCast(connS, playerS, worldS, shallow, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 8, Y: -0.8, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 4, "test"); err != nil {
		t.Fatal(err)
	}
	if playerS.rushDashIssuedSnapshot() {
		t.Fatal("airborne press must not take the ground press-grant path")
	}
	baseS := len(connS.Frames())
	if _, err := handleMovementUpload(connS, playerS, worldS,
		walkUploadMessage(-1610608895, 7, 4, 0, 0, 8, 0, 4.5, 0), "test"); err != nil {
		t.Fatal(err)
	}
	sawC61ShallowAir := false
	for _, f := range connS.Frames()[baseS:] {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("v31 client self-run must not emit 0x21 drive entries: %x", f)
		}
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 7 && values[0].Type == 2 && values[0].Int32 == 24 &&
				values[2].Type == 2 && values[2].Int32 == msgRushCommand {
				if values[6].Int32 != 2 {
					t.Fatalf("shallow high→low 0.8m must grant air tier flag2, got flag=%d", values[6].Int32)
				}
				// 3D 距离 √(3.5²… 报告位(4,0,0)→目标(8,-0.8,0)=√(16+0.64)≈4.08 ∈ (3.5,14.5]；
				// dest.y=目标 y+1.0=0.2（高→低俯冲悬停弧线）。
				if math.Abs(float64(values[4].Float32-0.2)) > 0.05 {
					t.Fatalf("c61 dest y=%v, want target+1.0=0.2", values[4].Float32)
				}
				sawC61ShallowAir = true
			}
		}
	}
	if !sawC61ShallowAir {
		t.Fatal("shallow high→low 0.8m walk-in must grant air-tier c61")
	}
	time.Sleep(50 * time.Millisecond)
}

// v30 211 直收分流高度闸：直收带内但 |出招位 y−目标 y|>1m 不再
// executeRushDirect（原地普攻=隔空打高处/低处目标），改走暂存链并立即冲锋。
func TestNearHeightGap211ChargesInsteadOfDirect(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off")
	var definition combatSkillDefinition
	found := false
	for _, candidate := range installedCombatSkillCatalog.levelOne {
		if candidate.stagedRushFamily() && candidate.requiresTarget {
			definition = candidate
			found = true
			break
		}
	}
	if !found {
		t.Skip("fixture catalog has no staged rush family selected-target skill")
	}
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	placeTestNPC(t, world, 91, float32(math.Max(0.5, rushDirectCastMeters(definition)-1)), 0)
	player := newPlayerActor("冲锋侠", 0)
	player.mu.Lock()
	player.learnedSkills[definition.id] = 1
	player.mu.Unlock()
	if err := handleSelectTargetCustom(world, player, warningToggleMessage(clientCustomSelectTarget,
		clientCustomValue{Type: 8, Raw: packTestObject(forkNPCID(91), npcTestOwner(91))}), "test"); err != nil {
		t.Fatal(err)
	}
	// 平距在直收带内、出招位高出目标 2m：直收被高度闸拦下 → 暂存+立即冲锋。
	if handled, err := handleSkillCustom(conn, player, world, nil, useSkillMessageAt(definition.id, 0, 2, 0), "test"); err != nil || !handled {
		t.Fatalf("height-gap cast handled=%t err=%v", handled, err)
	}
	if player.peekPendingCast(time.Now()) == nil {
		t.Fatal("near height-gap cast must stage, not direct-execute")
	}
	if !player.rushDashIssuedSnapshot() {
		t.Fatal("near height-gap cast must issue dash immediately after staging")
	}
	time.Sleep(700 * time.Millisecond) // 排空 ETA 落地 AfterFunc
}

// v33 冲锋动作 {W}h=套路武器位（2026-11-18 用户定版）：锁表逐技能块优先
//（含官方异武器档动画），武器推导只填缺行；use_limit 解析取 ItemType 行。
func TestRushActionWeaponSlotDerivation(t *testing.T) {
	if got := rushActionForSlot("qg_fastrun", 5); got != "qg_fastrun_5h_01" {
		t.Fatalf("ground construct=%q", got)
	}
	if got := rushActionForSlot("qg_fastruning_s", 8); got != "qg_fastruning_s_8h_01" {
		t.Fatalf("air construct=%q", got)
	}
	// 逐技能块配对：同一 LockType 包内两技能各归其位（旧 iniValue 末值
	// 会把邻技能动作拿给先出招的技能）。
	lock := iniTable{"42": []iniField{
		{key: "HitAction", value: "CS_a_sword"},
		{key: "LockGRushAction", value: "qg_fastrun_2h_01"},
		{key: "LockARushAction", value: "qg_fastruning_s_2h_01"},
		{key: "HitAction", value: "CS_b_fist"},
		{key: "LockGRushAction", value: "qg_fastrun_0h_01"},
		{key: "LockARushAction", value: "qg_fastruning_s_0h_01"},
	}}
	per := buildRushLockActions(lock)
	if got := per["CS_a_sword"]; got.ground != "qg_fastrun_2h_01" || got.air != "qg_fastruning_s_2h_01" {
		t.Fatalf("sword block=%+v", got)
	}
	if got := per["CS_b_fist"]; got.ground != "qg_fastrun_0h_01" || got.air != "qg_fastruning_s_0h_01" {
		t.Fatalf("fist block=%+v", got)
	}
	// 武器位解析：ItemType 行（非 ItemType 忽略、暗器族=不可推导）。
	tables := skillResourceTables{useLimit: iniTable{
		"2": []iniField{{key: "r", value: "0,ItemType,102,==,"}},
		"7": []iniField{
			{key: "r", value: "0,School,3,==,"},
			{key: "r", value: "0,ItemType,105,==,"},
		},
		"9": []iniField{{key: "r", value: "0,ItemType,111|116,==,"}},
	}}
	if got := rushWeaponSlotOf(tables, "2"); got != 2 {
		t.Fatalf("use_limit 2 slot=%d, want 2 (单手剑)", got)
	}
	if got := rushWeaponSlotOf(tables, "7"); got != 5 {
		t.Fatalf("use_limit 7 slot=%d, want 5 (双剑)", got)
	}
	if got := rushWeaponSlotOf(tables, "9"); got != 0 {
		t.Fatalf("暗器族 slot=%d, want 0 不可推导", got)
	}
	if got := rushWeaponSlotOf(tables, "404"); got != 0 {
		t.Fatalf("missing section slot=%d, want 0", got)
	}
	if got := rushWeaponSlotOf(tables, ""); got != 0 {
		t.Fatalf("empty id slot=%d, want 0", got)
	}
}

// hasQinggongRelease 断言辅助：0x21 空置条目 speed=0/intent=0xA0000700
//（writeQinggongReleaseFrame 官方式轻功绑定终结符）。
func hasQinggongRelease(frames [][]byte) bool {
	for _, f := range frames {
		if f[0] == 0x21 && len(f) >= 47 && binary.LittleEndian.Uint32(f[43:]) == 0xA0000700 {
			if got := math.Float32frombits(binary.LittleEndian.Uint32(f[27:])); got == 0 {
				return true
			}
		}
	}
	return false
}

// v31 轻功复位①（2026-11-17 用户令核对）：空中档已授予、212 迟到于
// pending 消费之后——resumePendingCastAt 的 pending==nil 早退分支必须补发
// 轻功终结符 + 授予对（CantMove/CantParry）还原。
func TestAirDashLate212RestoresQinggong(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off")
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	player := newPlayerActor("迟暮侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_late", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 3,
		groundRushAction: defaultGroundRushAction, airRushAction: "qg_fastruning_s_2h_01"}
	if _, err := stageRushCast(conn, player, world, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 2, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 6, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames())
	if _, err := handleMovementUpload(conn, player, world,
		walkUploadMessage(-1610608895, 7, 6, 0, 0, 10, 0, 4.5, 0), "test"); err != nil {
		t.Fatal(err)
	}
	if !player.rushDashIssuedSnapshot() || !player.rushDashAirborneSnapshot() {
		t.Fatal("air dash must be issued (airborne tier)")
	}
	// 模拟 pending 已被过期出口消费（dashIssued 标记仍在——consumePendingCast
	// 不清 rush 标记），随后 212 才迟到。
	player.mu.Lock()
	player.pendingCast = nil
	player.mu.Unlock()
	if _, err := handleCastDashLanding(conn, player, world,
		warningToggleMessage(clientCustomCastDashLanding,
			clientCustomValue{Type: 4, Float32: 9}, clientCustomValue{Type: 4, Float32: 1.2},
			clientCustomValue{Type: 4, Float32: 0}, clientCustomValue{Type: 4, Float32: 0.5},
			clientCustomValue{Type: 2, Int32: 1}), "test"); err != nil {
		t.Fatal(err)
	}
	if !hasQinggongRelease(conn.Frames()[base:]) {
		t.Fatalf("late 212 after pending consumed must restore qinggong binding, frames=%x", conn.Frames()[base:])
	}
	sawCantMoveClear := false
	for _, f := range conn.Frames()[base:] {
		if f[0] == 0x10 && len(f) >= 15 &&
			binary.LittleEndian.Uint16(f[12:]) == propRushCantMove && f[14] == 0 {
			sawCantMoveClear = true
		}
	}
	if !sawCantMoveClear {
		t.Fatalf("late 212 must clear grant pair CantMove=0, frames=%x", conn.Frames()[base:])
	}
}

// v31 轻功复位②：暂存 30s 过期出口（skillPauseReleaseTick）——空中档已
// 授予但 212 未到：轻功终结符 + CantMove/CantParry 清 + rush 在途标记复位
// + pending 槽出清。
func TestExpiredPendingRestoresDashState(t *testing.T) {
	t.Setenv("NINEYIN_RUSH_APPROACH", "off")
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	player := newPlayerActor("过期侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_exp", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 3,
		groundRushAction: defaultGroundRushAction, airRushAction: "qg_fastruning_s_2h_01"}
	if _, err := stageRushCast(conn, player, world, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 2, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 6, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames())
	if _, err := handleMovementUpload(conn, player, world,
		walkUploadMessage(-1610608895, 7, 6, 0, 0, 10, 0, 4.5, 0), "test"); err != nil {
		t.Fatal(err)
	}
	if !player.rushDashIssuedSnapshot() {
		t.Fatal("air dash must be issued")
	}
	// 手动拨过期并停掉看门狗定时器，直接驱动过期出口。
	player.mu.Lock()
	if player.skillPauseTimer != nil {
		player.skillPauseTimer.Stop()
	}
	if player.pendingCast != nil {
		player.pendingCast.expiresUTC = time.Now().UTC().Add(-time.Second)
	}
	player.mu.Unlock()
	player.skillPauseReleaseTick(conn, "test")
	if !hasQinggongRelease(conn.Frames()[base:]) {
		t.Fatalf("expired pending must restore qinggong binding, frames=%x", conn.Frames()[base:])
	}
	for _, probe := range []struct {
		idx  uint16
		want byte
	}{{propRushCantMove, 0}, {propCantParry, 0}} {
		saw := false
		for _, f := range conn.Frames()[base:] {
			if f[0] == 0x10 && len(f) >= 15 &&
				binary.LittleEndian.Uint16(f[12:]) == probe.idx && f[14] == probe.want {
				saw = true
			}
		}
		if !saw {
			t.Fatalf("expired pending must clear grant pair idx=%d, frames=%x", probe.idx, conn.Frames()[base:])
		}
	}
	if player.rushDashIssuedSnapshot() || player.rushInFlightSnapshot() {
		t.Fatal("expired pending must reset dash in-flight flags")
	}
	player.mu.Lock()
	pendingGone := player.pendingCast == nil
	player.mu.Unlock()
	if !pendingGone {
		t.Fatal("expired pending must consume the pending slot")
	}
}

// v31 轻功复位③：范围点地空中档（高→低）落地 AfterFunc 补轻功终结符。
func TestAreaAirLandingReleasesQinggong(t *testing.T) {
	definition := combatSkillDefinition{
		id: "CS_jh_hyjf07", canRush: true, castRange: 15,
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction,
		airRushAction: "qg_fastruning_s_2h_01",
	}
	conn := &captureMessageConnection{}
	player := newPlayerActor("点地侠", 0)
	player.noteClientReport(7.5, 3.6, 0, time.Now()) // 客户端自走后的上报位
	issueAreaGroundRushDash(conn, player, nil, definition,
		worldcore.Transform{Y: 4}, worldcore.Transform{X: 8}, "test")
	// 3D≈8.9/15+0.25≈0.85s；等落地 AfterFunc。
	time.Sleep(1200 * time.Millisecond)
	if !hasQinggongRelease(conn.Frames()) {
		t.Fatalf("area air landing must release qinggong binding, frames=%x", conn.Frames())
	}
}
