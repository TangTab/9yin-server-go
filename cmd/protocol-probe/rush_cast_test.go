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

// 冲锋受理簇（官服 e.pcapng lyfh02 +38ms 地面锁定序）：
// CurSkillID → CurSkillTarget → msg25[30000] → PauseSkill(技能) → msg707[15]。
// 距离 >6.5m 只发受理簇（客户端自动接近由簇触发）；≤6.5m 同簇尾接
// 冲刺簇（c61 档1 + CantMove/CantParry）。旧方言的 msg26 首帧与
// CurIsAirSkill=4（重受理/空中方言）不得出现。
func TestStageRushCastStagesPendingAndMarksInFlight(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_stage", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction}

	before := time.Now()
	if handled, err := stageRushCast(conn, player, definition, 91, 1,
		worldcore.Transform{X: 10, Y: 0, Z: 20}, worldcore.Transform{X: 4, Y: 0, Z: 20}, 12, "test"); err != nil || !handled {
		t.Fatalf("stage handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 5 {
		t.Fatalf("staging frames=%d, want 5-frame acceptance cluster", len(frames))
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
	if pending == nil || pending.skillID != "CS_rush_stage" || pending.targetID != 91 || pending.targetOwner != 1 || pending.airborne {
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
	// CD/资源在落地释放时才结：暂存不扣蓝不进 CD。
	if got := player.stateSnapshot().MP; got != player.vitals.MaxMP {
		t.Fatalf("staging spent MP: %d", got)
	}

	// 冲刺距离内按下（≤6.5m）也只发受理簇：c61 一律由 msg15/220 移动
	// 上报触发（官服序——受理即发 c61+CantMove 会把刚起走的客户端冻在
	// 原地，62622 真机教训）。
	conn2 := &captureMessageConnection{}
	player2 := newPlayerActor("冲锋侠", 0)
	if handled, err := stageRushCast(conn2, player2, definition, 91, 1,
		worldcore.Transform{X: 10, Y: 0, Z: 20}, worldcore.Transform{X: 5, Y: 0, Z: 20}, 5, "test"); err != nil || !handled {
		t.Fatalf("close stage handled=%t err=%v", handled, err)
	}
	if got := len(conn2.Frames()); got != 5 {
		t.Fatalf("close staging frames=%d, want acceptance only", got)
	}
	if player2.rushDashIssuedSnapshot() {
		t.Fatal("staging issued dash without movement upload")
	}

	// 暂存期再次按下（重受理）：官服 f 十连按实证前导三帧 CurSkillID=""
	// + msg26 + PauseSkill("") 清旧窗，再重铺受理簇（共 8 帧）。
	if handled, err := stageRushCast(conn, player, definition, 91, 1,
		worldcore.Transform{X: 10, Y: 0, Z: 20}, worldcore.Transform{X: 4, Y: 0, Z: 20}, 12, "test"); err != nil || !handled {
		t.Fatalf("rearm stage handled=%t err=%v", handled, err)
	}
	frames3 := conn.Frames()[5:]
	if len(frames3) != 8 ||
		frames3[0][0] != 0x10 || binary.LittleEndian.Uint16(frames3[0][12:]) != propCurSkillID ||
		int32(binary.LittleEndian.Uint32(frames3[1][4:])) != 26 ||
		binary.LittleEndian.Uint16(frames3[2][12:]) != propRushPauseSkill {
		t.Fatalf("rearm frames=%x, want CurSkillID-clear+msg26+PauseSkill-clear+cluster", frames3)
	}
}

// 空中判定禁用回归（2026-11-08 53831 真机教训）：world 内部 NPC y 与
// 广播 y 语义不一致曾致恒判空中→不发 c61→客户端零反应；禁用后 y 差
// 再大也走地面链（受理簇不含空中方言帧）。
func TestStageRushCastAirborneDisabledFallsBackToGroundDialect(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_air", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction}
	if _, err := stageRushCast(conn, player, definition, 91, 1,
		worldcore.Transform{X: 10, Y: 43, Z: 20}, worldcore.Transform{X: 4, Y: 50, Z: 20}, 12, "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()
	if len(frames) != 5 {
		t.Fatalf("air staging frames=%d, want 5 (airborne disabled)", len(frames))
	}
	if pending := player.peekPendingCast(time.Now()); pending == nil || pending.airborne {
		t.Fatalf("pending cast=%+v, want airborne=false", pending)
	}
}

func TestDashLandingWithoutPendingEchoesTransformOnly(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)

	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(4, 0, 5, 1.25), "test"); err != nil || !handled {
		t.Fatalf("cold landing handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	// 无暂存落地：只回 msg337 位置回显（清零簇不再刷——受理已不置位）。
	if len(frames) != 1 {
		t.Fatalf("cold landing frames=%d, want single msg337 echo", len(frames))
	}
	sync := frames[0]
	if sync[0] != 0x1E || int32(binary.LittleEndian.Uint32(sync[4:])) != 337 ||
		math.Float32frombits(binary.LittleEndian.Uint32(sync[33:])) != 1.25 {
		t.Fatalf("echo frame=%x", sync)
	}
	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(4, 0, 5, 0), "test"); err != nil || !handled {
		t.Fatalf("repeat landing handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 2 {
		t.Fatalf("repeat landing frames=%d, want one more echo", got)
	}
}

// 212 有暂存且冲刺已发出：337 回显 + msg26 解除 + 重组 211 重入受理链
// （自增益招无目标闸：CD 行/vital/State/msg20 全簇）。
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
	// 前导 8 帧（CantMove/CantParry 清+1F+PauseSkill 清+msg26+337+
	// c1=40 预兆+msg19 段起始）+ 重入链 6 帧（CD 双行夹 active_skill
	// 0x11×3 + vital + 337 + msg20）。
	if len(frames) != 14 {
		t.Fatalf("landing frames=%d, want 8 prelude + 6 resume", len(frames))
	}
	if frames[0][0] != 0x10 || binary.LittleEndian.Uint16(frames[0][12:]) != propRushCantMove {
		t.Fatalf("CantMove clear frame=%x", frames[0])
	}
	if frames[3][0] != 0x10 || binary.LittleEndian.Uint16(frames[3][12:]) != propRushPauseSkill {
		t.Fatalf("PauseSkill clear frame=%x", frames[3])
	}
	if frames[4][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[4][4:])) != 26 {
		t.Fatalf("pause clear frame=%x", frames[4])
	}
	if frames[5][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[5][4:])) != 337 {
		t.Fatalf("echo frame=%x", frames[5])
	}
	if frames[6][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[6][4:])) != 24 ||
		int32(binary.LittleEndian.Uint32(frames[6][18:])) != 40 {
		t.Fatalf("c1=40 precast frame=%x", frames[6])
	}
	if frames[7][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[7][4:])) != 19 {
		t.Fatalf("msg19 open frame=%x", frames[7])
	}
	if frames[8][0] != 0x11 || binary.LittleEndian.Uint16(frames[8][10:]) != recordCooldown {
		t.Fatalf("resumed cooldown frame=%x", frames[8])
	}
	if frames[9][0] != 0x11 || binary.LittleEndian.Uint16(frames[9][10:]) != recordActiveSkill {
		t.Fatalf("resumed active skill row=%x", frames[9])
	}
	if frames[10][0] != 0x11 || binary.LittleEndian.Uint16(frames[10][10:]) != recordCooldown {
		t.Fatalf("resumed GCD row=%x", frames[10])
	}
	if frames[13][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[13][4:])) != 20 {
		t.Fatalf("resumed msg20 frame=%x", frames[13])
	}
	if got := player.stateSnapshot().MP; got != beforeMP-11 {
		t.Fatalf("resume MP=%d, want %d (settled on landing)", got, beforeMP-11)
	}
	if player.rushInFlightSnapshot() {
		t.Fatal("landing kept rush in flight")
	}
	if player.peekPendingCast(time.Now()) != nil {
		t.Fatal("landing kept pending cast")
	}
	if got := player.currentSkillSnapshot(); got != "CS_yhwq_hsqs05" {
		t.Fatalf("resumed skill=%q", got)
	}

	// 暂存已消费：第二次 212 只回 msg337（14+1=15）。
	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(6, 0, 7, 0), "test"); err != nil || !handled {
		t.Fatalf("repeat landing handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 15 {
		t.Fatalf("repeat landing frames=%d, want one more echo", got)
	}
}

// 211 冲锋距离模型（官服 b/e.pcapng + f.pcapng 十连按，2026-11-08 修正）：
// 近距（≤max(bind+2,3.5)）直接正常受理（CD 行首帧）；更远一律走暂存
// 冲锋链（受理簇 5 帧）——**受理与距离解耦**（f 实证 14.6~19.8m 全受理
// 零拒绝），不做发起上限拒包。面板射程 castRange 不是冲锋族的可攻击
// 距离（myjf02/03 TSP2=6 时代 ≤6m 全部就地出伤即「把最大射程当可攻击
// 距离」症状）。
func TestOutOfRangeStagedRushFamilySkillStagesViaSkill211(t *testing.T) {
	var definition combatSkillDefinition
	found := false
	// 距离带按所选技能自身阈值相对摆位（bind 因技能而异——空中锁定技
	// bind 可达 10m，其近距阈值随之上移）；要求 rushRange 比近距阈值
	// 明显大，否则接近带摆不下。map 遍历顺序随机，选技须对摆位鲁棒。
	for id, candidate := range installedCombatSkillCatalog.levelOne {
		if candidate.stagedRushFamily() && candidate.requiresTarget &&
			rushStageNearCastMeters(candidate)+2 < float64(candidate.rushRange) {
			definition = candidate
			_ = id
			found = true
			break
		}
	}
	if !found {
		t.Skip("fixture catalog has no staged rush family selected-target skill with roomy rush range")
	}
	nearCast := rushStageNearCastMeters(definition)

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
			clientCustomValue{Type: 8, Raw: packTestObject(91, 1)}), "test"); err != nil {
			t.Fatal(err)
		}
		return conn, player, world
	}

	// 接近带：NPC 摆在 (max(6.5m,近距阈值), RushRange] 中点。受理簇 5 帧，
	// 无冲刺、无 CD 行。（选目标先写 2 帧，取基数后再断言。）
	conn, player, world := build(float32((math.Max(rushDashTriggerMeters(), nearCast) + float64(definition.rushRange)) / 2))
	base := len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("staging cast handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	if len(frames) != 5 || frames[0][0] != 0x10 ||
		int32(binary.LittleEndian.Uint32(frames[2][4:])) != 25 ||
		int32(binary.LittleEndian.Uint32(frames[4][4:])) != 707 {
		t.Fatalf("staging frames=%x, want 5-frame acceptance cluster", frames)
	}
	if !player.rushInFlightSnapshot() {
		t.Fatal("staging did not mark rush in flight")
	}

	// 近距阈值外不远处的按下（近距阈值+1.5m）：同样只发受理 5 帧
	//（c61 由移动上报触发，受理即冲=自创方言，62622 真机已废）。
	conn, player, world = build(float32(nearCast + 1.5))
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("close cast handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 5 || int32(binary.LittleEndian.Uint32(frames[4][4:])) != 707 {
		t.Fatalf("close cast frames=%x, want acceptance only", frames)
	}
	if player.rushDashIssuedSnapshot() {
		t.Fatal("close cast issued dash at staging")
	}

	// 远距（超 RushRange，f 十连按实证 19.8m）：受理与距离解耦——照常
	// 暂存（受理簇 5 帧），不拒包。
	conn, player, world = build(definition.rushRange + 5)
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("far cast handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 5 || int32(binary.LittleEndian.Uint32(frames[4][4:])) != 707 {
		t.Fatalf("far cast frames=%x, want acceptance (distance-decoupled)", frames)
	}
	if player.peekPendingCast(time.Now()) == nil {
		t.Fatal("far cast did not stage a pending skill")
	}

	// 近距（≤近距阈值，摆 max(0.5, 阈值-1)）：直接正常受理（CD 行在首帧）。
	conn, player, world = build(float32(math.Max(0.5, nearCast-1)))
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("near cast handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) == 0 || frames[0][0] != 0x11 {
		t.Fatalf("near cast frames=%x, want cooldown first", frames)
	}
	if player.peekPendingCast(time.Now()) != nil {
		t.Fatal("near cast staged a pending skill")
	}
}

// msg15/msg220 驱动的接近探测（官服新链）：无冲锋静默；远距（>6.5m）
// 零帧（客户端自动接近，服务端无帧）；进入冲刺距离（≤6.5m）发最终
// 冲刺簇（c61+CantMove/CantParry，终点=bind、y=玩家上报 y+1.0——world
// 内部 NPC y 不可用）；冲刺已发出后到位探测静默。212 按上报位置三分：
// ≤3m=落地续招（8 帧官服前导簇）；远处未冲刺=就地执行请求→发 c61
// 命其冲刺；远处已冲刺未执行=安全释放（不出伤害）。
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
	if _, err := stageRushCast(conn, player, definition, 91, 1,
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames()) // 受理簇 5 帧为基数。

	// 距目标 8m（>6.5m 冲刺阈值）：零帧（客户端自动接近中，服务端无帧）。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(2, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("far movement handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("far movement emitted %d frames, want 0", got)
	}

	// 远处执行请求 212（近距站立按下，客户端就地执行请求；上报点距目标
	// 6m > 3m、未发 c61）：发 c61 命其冲刺（3 帧），暂存保留不取消。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(4, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("execute-request landing handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	// c61+CantMove+CantParry + 个人CD行(字面定义无 team/publicCD) + MPRatio/MP/QG = 7 帧。
	if len(frames) != 7 || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 24 ||
		int32(binary.LittleEndian.Uint32(frames[0][18:])) != msgRushCommand ||
		frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][12:]) != propRushCantMove {
		t.Fatalf("execute-request frames=%x, want c61 cluster with settle rows", frames)
	}
	if !player.rushDashIssuedSnapshot() || player.peekPendingCast(time.Now()) == nil {
		t.Fatal("execute-request did not issue dash or dropped pending")
	}
	base = len(conn.Frames())

	// 客户端没执行冲刺、再次远处 212（c61 已发）：安全释放——作废暂存 +
	// msg26 + PauseSkill("") + 337，不出落地簇不出伤害。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(4, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("unexecuted-dash landing handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 3 || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 26 ||
		frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][12:]) != propRushPauseSkill ||
		int32(binary.LittleEndian.Uint32(frames[2][4:])) != 337 {
		t.Fatalf("unexecuted-dash frames=%x, want msg26+PauseSkill-clear+337", frames)
	}
	if player.rushInFlightSnapshot() || player.peekPendingCast(time.Now()) != nil {
		t.Fatal("unexecuted-dash release kept rush state alive")
	}

	// 自冲落地（54992 会话实证：客户端本地直冲后上报）：未发 c61 但上报
	// 点距目标 ≤3m → 按落地续招处理（8 帧前导簇）。
	if _, err := stageRushCast(conn, player, definition, 91, 1,
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 5, Y: 0, Z: 0}, 5, "test"); err != nil {
		t.Fatal(err)
	}
	base = len(conn.Frames())
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("self-dash landing handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 8 || int32(binary.LittleEndian.Uint32(frames[7][4:])) != 19 {
		t.Fatalf("self-dash frames=%x, want 8-frame landing prelude", frames)
	}
	if player.rushInFlightSnapshot() || player.peekPendingCast(time.Now()) != nil {
		t.Fatal("self-dash landing kept rush state alive")
	}

	// 重新暂存后走完整链：接近上传进入冲刺距离（距目标 6m ≤6.5m）→
	// c61 冲刺簇 3 帧，终点=目标身前 bind、y+1.0。
	if _, err := stageRushCast(conn, player, definition, 91, 1,
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base = len(conn.Frames())
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(4, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("dash movement handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 7 || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 24 ||
		int32(binary.LittleEndian.Uint32(frames[0][18:])) != msgRushCommand ||
		frames[1][0] != 0x10 || binary.LittleEndian.Uint16(frames[1][12:]) != propRushCantMove {
		t.Fatalf("dash frames=%x, want c61 cluster with settle rows", frames)
	}
	destX := math.Float32frombits(binary.LittleEndian.Uint32(frames[0][23:]))
	destY := math.Float32frombits(binary.LittleEndian.Uint32(frames[0][28:]))
	// 目标 x=10，上报位置 x=4 → bind=10-1.5=8.5；y=0+1.0。
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

	// 冲刺在途再上报（距目标 1.5m）：静默——落地只认 212，中途 resume
	// 会打断客户端冲刺动画。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("in-dash movement handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("in-dash movement emitted %d frames, want 0", got)
	}

	// 212 落地：官服落地前导簇 8 帧；重组 211 的技能未习得被拒
	//（CantMove/CantParry 清+1F+PauseSkill 清+msg26+337+c1=40+msg19）。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("landing handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 8 || int32(binary.LittleEndian.Uint32(frames[4][4:])) != 26 ||
		int32(binary.LittleEndian.Uint32(frames[5][4:])) != 337 ||
		int32(binary.LittleEndian.Uint32(frames[6][4:])) != 24 ||
		int32(binary.LittleEndian.Uint32(frames[7][4:])) != 19 {
		t.Fatalf("landing frames=%x", frames)
	}
	if player.rushInFlightSnapshot() || player.peekPendingCast(time.Now()) != nil {
		t.Fatal("landing kept rush state alive")
	}
}

// msg220 冲锋在途位移上报（4×f32）：驱动同款接近/冲刺探测；非在途静默。
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
	if _, err := stageRushCast(conn, player, definition, 91, 1,
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 4, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames())

	// 距目标 6m ≤6.5m：c61 冲刺簇 3 帧。
	if handled, err := handleRushMovementUpload(conn, player, world, upload220, "test"); err != nil || !handled {
		t.Fatalf("dash 220 handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	if len(frames) != 7 || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 24 ||
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

	// 212 落地：8 帧落地前导簇（同 msg15 链）。
	if handled, err := handleCastDashLanding(conn, player, world, dashLandingMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("landing after 220 handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 8 || int32(binary.LittleEndian.Uint32(frames[7][4:])) != 19 {
		t.Fatalf("landing after 220 frames=%x", frames)
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
