package main

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

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

// 冲锋链帧字节钉死（官服 7071/7074 线值序，0x1E 外壳）：
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

// 受理簇（官服 g/f 定版首发 5 帧）：CurSkillID → CurSkillTarget →
// msg25[30000] → PauseSkill → msg707[15]。重受理 7 帧：三帧清前导
// CurSkillID=""+msg26+PauseSkill="" 后**不重发 CurSkillTarget**。
func TestStageRushCastStagesPendingAndMarksInFlight(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_stage", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction}

	before := time.Now()
	if handled, err := stageRushCast(conn, player, definition, forkNPCID(91), 1,
		worldcore.Transform{X: 10, Y: 0, Z: 20}, worldcore.Transform{X: 4, Y: 0, Z: 20}, 12, "test"); err != nil || !handled {
		t.Fatalf("stage handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 9 { // 5 帧受理簇 + 状态四连（官服簇尾 +42ms 必发）
		t.Fatalf("staging frames=%d, want 9-frame acceptance cluster+state-ping", len(frames))
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
	if pending == nil || pending.skillID != "CS_rush_stage" || pending.targetID != forkNPCID(91) || pending.targetOwner != 1 || pending.airborne {
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

	// 暂存期再次按下（重受理）：官服 f/g 定版 7 帧——CurSkillID="" +
	// msg26 + PauseSkill="" + CurSkillID + msg25 + PauseSkill + msg707，
	// 不重发 CurSkillTarget。
	if handled, err := stageRushCast(conn, player, definition, forkNPCID(91), 1,
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

// 空中判定禁用回归（53831 真机教训）：y 差再大也走地面链。
func TestStageRushCastAirborneDisabledFallsBackToGroundDialect(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_air", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction}
	if _, err := stageRushCast(conn, player, definition, forkNPCID(91), 1,
		worldcore.Transform{X: 10, Y: 43, Z: 20}, worldcore.Transform{X: 4, Y: 50, Z: 20}, 12, "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 9 { // 5 帧受理簇 + 状态四连
		t.Fatalf("air staging frames=%d, want 9 (airborne disabled)", len(frames))
	}
	if pending := player.peekPendingCast(time.Now()); pending == nil || pending.airborne {
		t.Fatalf("pending cast=%+v, want airborne=false", pending)
	}
}

// 无暂存 212：官服零回应（b 51.4+ 连发 15 条 212[尾0] 全程静默）。
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
	player.markRushDashIssued()
	beforeMP := player.stateSnapshot().MP

	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(6, 0, 7, 0), "test"); err != nil || !handled {
		t.Fatalf("landing handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 18 { // 14 帧落地簇 + 状态四连（官服 416.321 实证）
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

	// 暂存已消费：第二次 212 官服零回应。
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
		if err := handleSelectTargetCustom(world, warningToggleMessage(clientCustomSelectTarget,
			clientCustomValue{Type: 8, Raw: packTestObject(forkNPCID(91), 1)}), "test"); err != nil {
			t.Fatal(err)
		}
		return conn, player, world
	}

	// 接近带：受理簇 5 帧，无冲刺、无 CD 行。
	conn, player, world := build(float32((math.Max(rushDashTriggerMeters(), nearCast) + float64(definition.rushRange)) / 2))
	base := len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("staging cast handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	if len(frames) != 9 || frames[0][0] != 0x10 ||
		int32(binary.LittleEndian.Uint32(frames[2][4:])) != 25 ||
		int32(binary.LittleEndian.Uint32(frames[4][4:])) != 707 {
		t.Fatalf("staging frames=%x, want 9-frame acceptance cluster+state-ping", frames)
	}
	if !player.rushInFlightSnapshot() {
		t.Fatal("staging did not mark rush in flight")
	}

	// 直收带外不远处的按下（直收带+1.5m）：同样只发受理 5 帧。
	conn, player, world = build(float32(nearCast + 1.5))
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
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
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
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
	// CurSkillID（官服 g 直收簇，无 0x11/CD 首帧、无暂存）。
	conn, player, world = build(float32(math.Max(0.5, nearCast-1)))
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
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

// msg15/msg220 接近探测 + 冲刺窗口 (5.0, 6.5]（g 5.54 触发 / b 3.61 不触发）：
// 窗口外零帧；窗口内 c61 冲刺簇；冲刺已发后静默；212 三分兜底保留。
func TestMovementUploadDrivesRushArrival(t *testing.T) {
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
		id: "CS_rush_unlearned", canRush: true, script: "skilllock", castRange: 8,
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction,
	}
	if _, err := stageRushCast(conn, player, definition, forkNPCID(91), 1,
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames()) // 受理簇 5 帧为基数。

	// 距目标 8m（>6.5m 窗口上界）：fork 客户端方言——发官服同形 c61 接近
	// 冲刺（档1 + y+1.0；fork 客户端无自主接近行为，服务端驱动）。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(2, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("far movement handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	if len(frames) != 1 || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 24 ||
		int32(binary.LittleEndian.Uint32(frames[0][18:])) != msgRushCommand ||
		int32(binary.LittleEndian.Uint32(frames[0][38:])) != convertDashLevel {
		t.Fatalf("far movement frames=%x, want c61 approach dash (档1 官服同形)", frames)
	}
	base = len(conn.Frames())

	// 窗口下界以下（距目标 4m ≤5.0）：零帧——官服走近直接 212[尾0] 落地，
	// 不发 c61（b 4.41m 样本 3.61m 上报不触发实证）。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(6, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("below-window movement handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("below-window movement emitted %d frames, want 0", got)
	}

	// 走带外 212（距目标 10m > 认可半径 max(5.0, castRange+1=9)——技能
	// .pcapng 定版：hyjf02(射程6) 212@5.58/5.81 官服认可落地，认可半径
	// 随 castRange 走）：取消方言（f 拉回/62622 暂存期按别的技能）——
	// 安全作废，3 帧释放簇。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(0, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("far landing handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 3 || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 26 ||
		frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][12:]) != propRushPauseSkill ||
		int32(binary.LittleEndian.Uint32(frames[2][4:])) != 337 {
		t.Fatalf("far landing frames=%x, want msg26+PauseSkill-clear+337 cancel", frames)
	}
	if player.rushInFlightSnapshot() || player.peekPendingCast(time.Now()) != nil {
		t.Fatal("far landing kept rush state alive")
	}

	// 自冲落地（上报点距目标 ≤3m，未发 c61=步行落地形态）：
	// PauseSkill+msg26+LogicState 对+fight_relation+0x1F×1+337+c1=40+
	// msg19+CantMove/CantParry=11 帧（步行无 CantMove 清前导、单 0x1F）。
	if _, err := stageRushCast(conn, player, definition, forkNPCID(91), 1,
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 5, Y: 0, Z: 0}, 5, "test"); err != nil {
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

	// 重新暂存后走完整冲刺链：上报点距目标 6m ∈ (5.0, 6.5] → c61 簇
	//（零成本定义 6 帧），终点=目标身前 bind、y+1.0。
	if _, err := stageRushCast(conn, player, definition, forkNPCID(91), 1,
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base = len(conn.Frames())
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(4, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("dash movement handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 6 || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 24 ||
		int32(binary.LittleEndian.Uint32(frames[0][18:])) != msgRushCommand ||
		frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][12:]) != propRushCantMove {
		t.Fatalf("dash frames=%x, want c61 cluster (6 frames)", frames)
	}
	destX := math.Float32frombits(binary.LittleEndian.Uint32(frames[0][23:]))
	destY := math.Float32frombits(binary.LittleEndian.Uint32(frames[0][28:]))
	if destX != 8.5 || destY != 1.0 {
		t.Fatalf("c61 dest=(%.2f,%.2f), want (8.5,1.0)", destX, destY)
	}
	if int32(binary.LittleEndian.Uint32(frames[0][38:])) != convertDashLevel {
		t.Fatalf("c61 dash level=%x", frames[0])
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

	// 212 冲刺落地：14 帧官服簇（定义未习得 → 伤害链不启动）。
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
		{Type: 4, Float32: 4},
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
		id: "CS_rush_unlearned", canRush: true, script: "skilllock", castRange: 8,
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction,
	}
	if _, err := stageRushCast(conn, player, definition, forkNPCID(91), 1,
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames())

	// 距目标 6m ∈ (5.0, 6.5]：c61 冲刺簇（零成本 6 帧）。
	if handled, err := handleRushMovementUpload(conn, player, world, upload220, "test"); err != nil || !handled {
		t.Fatalf("dash 220 handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	if len(frames) != 6 || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 24 ||
		int32(binary.LittleEndian.Uint32(frames[0][18:])) != msgRushCommand {
		t.Fatalf("dash 220 frames=%x, want c61 dash cluster", frames)
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

	// 212 冲刺落地：14 帧官服簇（同 msg15 链）。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("landing after 220 handled=%t err=%v", handled, err)
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
// 为增量、跳帧无朝向——一律 0（技能.pcapng：官服 state7 [15,id,7,x,y,z,
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

// 范围型点地三分带（技能.pcapng 定版）：≤3.5 直收零帧（55.5@1.45 无
// c61）；3.5<d≤6.5 c61 档1 qg_fastrun（1216@5.52/+48ms、1275@3.96）；
// >6.5 档2 qg_fastruning_s（b/e 8.2）。簇=0x1F 锚 + c61（无
// CantMove/CantParry——那是锁定 c61 专属）；+400ms 落地三件套
// msg26+0x1F+337（近似官服 1216.720）。
func TestAreaGroundRushDashTierSelection(t *testing.T) {
	definition := combatSkillDefinition{
		id: "CS_jh_hyjf07", canRush: true, castRange: 15,
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction,
	}
	newCase := func(gx float32) (*captureMessageConnection, worldcore.Transform) {
		conn := &captureMessageConnection{}
		castAt := worldcore.Transform{X: 0, Y: 0, Z: 0}
		issueAreaGroundRushDash(conn, newPlayerActor("点地侠", 0), definition, castAt,
			worldcore.Transform{X: gx, Y: 0, Z: 0}, "test")
		return conn, castAt
	}

	conn, _ := newCase(2) // ≤3.5：直收，主路径处理，零帧。
	if got := len(conn.Frames()); got != 0 {
		t.Fatalf("near-ground emitted %d frames, want 0", got)
	}

	conn, castAt := newCase(5) // 档1 带。
	frames := conn.Frames()
	if len(frames) != 2 || frames[0][0] != 0x1F ||
		int32(binary.LittleEndian.Uint32(frames[1][4:])) != 24 ||
		int32(binary.LittleEndian.Uint32(frames[1][18:])) != msgRushCommand ||
		int32(binary.LittleEndian.Uint32(frames[1][38:])) != 1 {
		t.Fatalf("tier1 frames=%x, want 0x1F + c61 tier1", frames)
	}
	// c61 终点 = ground 身前 bind（官服 1216 dash=dist-1.5）。
	destX := binary.LittleEndian.Uint32(frames[1][23:])
	if math.Abs(float64(math.Float32frombits(destX)-3.5)) > 0.01 {
		t.Fatalf("tier1 dest x = %v, want 3.5 (ground 5 - bind 1.5)", math.Float32frombits(destX))
	}
	_ = castAt

	conn, _ = newCase(8) // 档2 带。
	frames = conn.Frames()
	if len(frames) != 2 || int32(binary.LittleEndian.Uint32(frames[1][38:])) != 2 {
		t.Fatalf("tier2 frames=%x, want 0x1F + c61 tier2", frames)
	}
}
