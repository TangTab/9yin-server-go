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

// 冲锋链帧字节钉死（Java FightMessages 值序，0x1E 外壳）：
// msg25=[25,30000,"skill_pause"]；msg26=[26]；
// msg337=[337,obj,x,y,z,orient,0]；msg24 c61=[24,obj,61,dest,档2,疾跑动画]。
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

	dash, err := dashRushFrame(10, 0, 20, stagingDashLevel, stagingDashAction)
	if err != nil {
		t.Fatal(err)
	}
	if len(dash) != 69 || dash[0] != 0x1E || binary.LittleEndian.Uint16(dash[1:]) != 8 ||
		dash[3] != 2 || int32(binary.LittleEndian.Uint32(dash[4:])) != 24 ||
		dash[8] != 8 || binary.LittleEndian.Uint32(dash[9:]) != playerObjectID ||
		binary.LittleEndian.Uint32(dash[13:]) != playerOwnerID ||
		dash[17] != 2 || int32(binary.LittleEndian.Uint32(dash[18:])) != msgRushCommand ||
		dash[22] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(dash[23:])) != 10 ||
		dash[27] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(dash[28:])) != 0 ||
		dash[32] != 4 || math.Float32frombits(binary.LittleEndian.Uint32(dash[33:])) != 20 ||
		dash[37] != 2 || int32(binary.LittleEndian.Uint32(dash[38:])) != stagingDashLevel ||
		dash[42] != 6 || string(dash[47:68]) != stagingDashAction || dash[68] != 0 {
		t.Fatalf("msg24 staging dash frame=%x", dash)
	}
}

// 暂存疾跑三帧：CurSkillID 单属性 0x10 + msg25 + msg24 档2；标记在途。
func TestStageRushCastStagesPendingAndMarksInFlight(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_stage", canRush: true, script: "skilllock"}

	before := time.Now()
	if handled, err := stageRushCast(conn, player, definition, 91, 1,
		worldcore.Transform{X: 10, Y: 0, Z: 20}, 12, "test"); err != nil || !handled {
		t.Fatalf("stage handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 3 {
		t.Fatalf("staging frames=%d, want CurSkillID+msg25+msg24", len(frames))
	}
	curSkill := frames[0]
	if curSkill[0] != 0x10 || binary.LittleEndian.Uint16(curSkill[10:]) != 1 ||
		binary.LittleEndian.Uint16(curSkill[12:]) != propCurSkillID {
		t.Fatalf("CurSkillID frame=%x", curSkill)
	}
	if frames[1][0] != 0x1E || frames[2][0] != 0x1E {
		t.Fatalf("staging custom frames=%x %x", frames[1], frames[2])
	}
	pending := player.peekPendingCast(time.Now())
	if pending == nil || pending.skillID != "CS_rush_stage" || pending.targetID != 91 || pending.targetOwner != 1 {
		t.Fatalf("pending cast=%+v", pending)
	}
	if pending.expiresUTC.Before(before.Add(rushPendingWindow - time.Second)) {
		t.Fatalf("pending window=%s, want ~%s", pending.expiresUTC.Sub(before), rushPendingWindow)
	}
	if !player.rushInFlightSnapshot() {
		t.Fatal("staging did not mark rush in flight")
	}
	// CD/资源在落地释放时才结：暂存不扣蓝不进 CD。
	if got := player.stateSnapshot().MP; got != player.vitals.MaxMP {
		t.Fatalf("staging spent MP: %d", got)
	}
}

// 212 无暂存：仅 msg337 回显（fork/Java 同款），随后静默。
func TestDashLandingWithoutPendingEchoesTransformOnly(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)

	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(4, 0, 5, 1.25), "test"); err != nil || !handled {
		t.Fatalf("cold landing handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
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
		t.Fatalf("repeat landing appended %d frames, want one more echo", got)
	}
}

// 212 有暂存：337 回显 + msg26 解除 + 重组 211 重入受理链
// （自增益招无目标闸：CD 行/vital/State/msg20 全簇）。
func TestDashLandingResumesPendingSelfSkill(t *testing.T) {
	conn := &captureMessageConnection{}
	player := newPlayerActor("冲锋侠", 0)
	player.stagePendingCast(&pendingCastState{
		skillID: "CS_yhwq_hsqs05", targetID: playerObjectID, targetOwner: playerOwnerID,
		expiresUTC: time.Now().Add(rushPendingWindow),
	})
	player.markRushInFlight()
	beforeMP := player.stateSnapshot().MP

	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(6, 0, 7, 0), "test"); err != nil || !handled {
		t.Fatalf("landing handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 6 {
		t.Fatalf("landing frames=%d, want 337+26+cooldown+vital+state+msg20", len(frames))
	}
	if frames[0][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 337 {
		t.Fatalf("echo frame=%x", frames[0])
	}
	if frames[1][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[1][4:])) != 26 {
		t.Fatalf("pause clear frame=%x", frames[1])
	}
	if frames[2][0] != 0x11 || binary.LittleEndian.Uint16(frames[2][10:]) != recordCooldown {
		t.Fatalf("resumed cooldown frame=%x", frames[2])
	}
	if frames[5][0] != 0x27 {
		t.Fatalf("resumed msg20 frame=%x", frames[5])
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

	// 暂存已消费：第二次 212 只回 337。
	if handled, err := handleCastDashLanding(conn, player, nil, dashLandingMessage(6, 0, 7, 0), "test"); err != nil || !handled {
		t.Fatalf("repeat landing handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()); got != 7 {
		t.Fatalf("repeat landing frames=%d, want one more echo", got)
	}
}

// 211 越程三层闸（用真实目录里的冲锋族技能；距离按所选技能的
// castRange 相对取值，兼容夹具/正式服两种表来源）：射程内正常受理；
// castRange<距离≤15m 暂存三帧；>15m 静默拒零帧。
func TestOutOfRangeStagedRushFamilySkillStagesViaSkill211(t *testing.T) {
	var definition combatSkillDefinition
	found := false
	for id, candidate := range installedCombatSkillCatalog.levelOne {
		if candidate.stagedRushFamily() && candidate.requiresTarget {
			definition = candidate
			_ = id
			found = true
			break
		}
	}
	if !found {
		t.Skip("fixture catalog has no staged rush family selected-target skill")
	}

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

	// 暂存带：NPC 摆在 (castRange, 15m] 中点。三帧，无 CD 行。
	//（选目标先写 2 帧，取基数后再断言。）
	conn, player, world := build(definition.castRange + (15-definition.castRange)/2)
	base := len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("staging cast handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	if len(frames) != 3 || frames[0][0] != 0x10 ||
		int32(binary.LittleEndian.Uint32(frames[1][4:])) != 25 ||
		int32(binary.LittleEndian.Uint32(frames[2][4:])) != 24 {
		t.Fatalf("staging frames=%x", frames)
	}
	if !player.rushInFlightSnapshot() {
		t.Fatal("staging did not mark rush in flight")
	}

	// >15m：静默拒（零帧、无暂存）。
	conn, player, world = build(20)
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("far cast handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("far cast emitted %d frames, want silent reject", got)
	}
	if player.peekPendingCast(time.Now()) != nil {
		t.Fatal("far cast staged a pending skill")
	}

	// 射程内：NPC 摆在 castRange-1（保底 0.5m）。正常受理簇（CD 行在首帧）。
	conn, player, world = build(max(0.5, definition.castRange-1))
	base = len(conn.Frames())
	if handled, err := handleSkillCustom(conn, player, world, useSkillMessageAt(definition.id, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("near cast handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) == 0 || frames[0][0] != 0x11 {
		t.Fatalf("near cast frames=%x, want cooldown first", frames)
	}
}

// msg15 驱动的到位探测：无冲锋静默；>3m 未冲刺先转冲刺（msg24 档1+
// 0x1F 锚，每暂存一次）；≤3m 落地续招（337+26+重组 211——未习得技能
// 的重组被受理链拒绝，只余两帧）。
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
		worldcore.Transform{X: 10, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames()) // 暂存三帧为基数。

	// 距目标 6m（>3m、≤8m 射程）：转冲刺 msg24 档1 + 0x1F 终点锚。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(4, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("convert movement handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()[base:]
	if len(frames) != 2 || frames[0][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[0][4:])) != 24 ||
		frames[1][0] != 0x1F {
		t.Fatalf("dash conversion frames=%x", frames)
	}
	// 终点=目标朝玩家方向 bind(1.5m) 处：目标(10,0)，玩家(4,0) → (8.5,0)。
	if got := math.Float32frombits(binary.LittleEndian.Uint32(frames[0][23:])); math.Abs(float64(got)-8.5) > 0.01 {
		t.Fatalf("dash dest X=%.2f, want 8.5", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(frames[1][9:])); math.Abs(float64(got)-8.5) > 0.01 {
		t.Fatalf("anchor X=%.2f, want 8.5", got)
	}

	// 同位置再报：不重复转冲刺。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(4, 0, 0, 0), "test"); err != nil || !handled {
		t.Fatalf("repeat movement handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 2 {
		t.Fatalf("repeat movement appended %d frames", got-2)
	}

	// 到达（距目标 1.5m ≤3m）：337 + 26；重组 211 的技能未习得被拒。
	if handled, err := handleMovementUpload(conn, player, world, movementUploadMessage(8.5, 0, 0, 0.5), "test"); err != nil || !handled {
		t.Fatalf("arrival movement handled=%t err=%v", handled, err)
	}
	frames = conn.Frames()[base:]
	if len(frames) != 4 || int32(binary.LittleEndian.Uint32(frames[2][4:])) != 337 ||
		int32(binary.LittleEndian.Uint32(frames[3][4:])) != 26 {
		t.Fatalf("arrival frames=%x", frames)
	}
	if player.rushInFlightSnapshot() || player.peekPendingCast(time.Now()) != nil {
		t.Fatal("arrival kept rush state alive")
	}
}

func packTestObject(objectID, ownerID uint32) [8]byte {
	var raw [8]byte
	binary.LittleEndian.PutUint64(raw[:], uint64(objectID)|uint64(ownerID)<<32)
	return raw
}
