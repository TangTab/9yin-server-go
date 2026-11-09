package main

// 官服点击移动应答方言测试（技能.pcapng §一 全样本定版）：
// 每帧 0xA000 族 state7/4 回显单条目 0x21（尾三槽 5.5|9.9|msg15 id），
// 走↔停迁移沿发状态四连（586/645/646/750），state5 跳跃与未知态零回显。

import (
	"encoding/binary"
	"math"
	"testing"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// walkUploadMessage 构造官服 82B 走帧形态的 msg15：
// [15, id, state, x, y, z, destX, destZ, speed, orient, clock]。
func walkUploadMessage(intentID int32, state int32, x, y, z, destX, destZ, speed, orient float32) clientCustomMessage {
	return clientCustomMessage{Opcode: 0x1E, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomMovementUpload},
		{Type: 2, Int32: intentID},
		{Type: 2, Int32: state},
		{Type: 4, Float32: x},
		{Type: 4, Float32: y},
		{Type: 4, Float32: z},
		{Type: 4, Float32: destX},
		{Type: 4, Float32: destZ},
		{Type: 4, Float32: speed},
		{Type: 4, Float32: orient},
		{Type: 2, Int32: 3116453},
	}}
}

// stopUploadMessage 构造官服 67B 停帧形态：[15, id, 4, x, y, z, 朝向, clock]。
func stopUploadMessage(intentID int32, x, y, z, facing float32) clientCustomMessage {
	return clientCustomMessage{Opcode: 0x1E, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomMovementUpload},
		{Type: 2, Int32: intentID},
		{Type: 2, Int32: 4},
		{Type: 4, Float32: x},
		{Type: 4, Float32: y},
		{Type: 4, Float32: z},
		{Type: 4, Float32: facing},
		{Type: 2, Int32: 3118718},
	}}
}

// decodePlayerEcho 解码单条目 0x21 回显帧的 11 槽。
func decodePlayerEcho(t *testing.T, frame []byte) (destX, destY, destZ, angle, speed, orient float32, tail [3]uint32) {
	t.Helper()
	if len(frame) != 47 || frame[0] != 0x21 || binary.LittleEndian.Uint16(frame[1:]) != 1 {
		t.Fatalf("echo frame=%x, want 47B single-entry 0x21", frame)
	}
	values := make([]uint32, 9)
	for i := range values {
		values[i] = binary.LittleEndian.Uint32(frame[11+i*4:])
	}
	floats := math.Float32frombits
	destX, destY, destZ = floats(values[0]), floats(values[1]), floats(values[2])
	angle, speed, orient = floats(values[3]), floats(values[4]), floats(values[5])
	tail = [3]uint32{values[6], values[7], values[8]}
	return
}

// 点击地面起跑：状态四连 + 0x21 回显（官服 2.594→2.637/2.701 同构）；
// 同步重发（改向）只回 0x21；到位停帧再发四连 + 停回显；跳跃零回显。
func TestMovementUploadEchoesPlayerIntent(t *testing.T) {
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	player := newPlayerActor("点击侠", 0)

	// 起跑：0xA0000F01 state7（官服 2.594 样本值）。
	handled, err := handleMovementUpload(conn, player, world, walkUploadMessage(
		-1610608895 /* 0xA0000F01 */, 7,
		945.82, 43.89, 1029.98, 946.54, 1040.19, 4.50, 3.14), "test")
	if err != nil || !handled {
		t.Fatalf("walk start handled=%t err=%v", handled, err)
	}
	frames := conn.Frames()
	if len(frames) != 5 {
		t.Fatalf("walk start emitted %d frames, want 4 ping + 1 echo", len(frames))
	}
	for i, id := range []int32{586, 645, 646, 750} {
		if frames[i][0] != 0x1E || int32(binary.LittleEndian.Uint32(frames[i][4:])) != id {
			t.Fatalf("ping[%d]=%x, want msg%d", i, frames[i], id)
		}
	}
	destX, destY, destZ, angle, speed, orient, tail := decodePlayerEcho(t, frames[4])
	if binary.LittleEndian.Uint32(frames[4][3:]) != playerObjectID {
		t.Fatalf("echo obj=%x, want playerObjectID", binary.LittleEndian.Uint32(frames[4][3:]))
	}
	if destX != 946.54 || destY != 43.89 || destZ != 1040.19 || speed != 4.50 || orient != 3.14 {
		t.Fatalf("walk echo dest=(%.2f,%.2f,%.2f) spd=%.2f ori=%.2f", destX, destY, destZ, speed, orient)
	}
	// 移动角 = atan2(destX−x, destZ−z)（官服 0.07 样本钉死）。
	if want := float32(0.0705); math.Abs(float64(angle-want)) > 0.001 {
		t.Fatalf("walk echo angle=%.4f, want %.4f", angle, want)
	}
	if tail != [3]uint32{0x40B00000, 0x411CCCCD, 0xA0000F01} {
		t.Fatalf("walk echo tail=%x, want 5.5|9.9|msg15id", tail)
	}

	// 在途重发（追怪改向同构）：只回 0x21、无 ping、角度按新帧位重算
	//（官服 347.331 样本：961.16→977.77 ang=1.92）。
	base := len(conn.Frames())
	handled, err = handleMovementUpload(conn, player, world, walkUploadMessage(
		-1610608895, 7, 961.16, 43.89, 1035.31, 977.766, 1029.317, 4.50, 3.14), "test")
	if err != nil || !handled {
		t.Fatalf("walk reissue handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 1 {
		t.Fatalf("walk reissue emitted %d frames, want 1 echo only", got)
	}
	_, _, _, angle, _, _, _ = decodePlayerEcho(t, conn.Frames()[base])
	if want := float32(1.917); math.Abs(float64(angle-want)) > 0.001 {
		t.Fatalf("reissue angle=%.4f, want %.4f", angle, want)
	}

	// 到位停帧：0xA0000000 state4（官服 4.871 样本值）——四连 + 停回显
	//（dest=终点、角度=朝向增量、速度 0）。
	base = len(conn.Frames())
	handled, err = handleMovementUpload(conn, player, world, stopUploadMessage(
		-1610612736 /* 0xA0000000 */, 946.54, 43.89, 1040.19, 0.071), "test")
	if err != nil || !handled {
		t.Fatalf("stop handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 5 {
		t.Fatalf("stop emitted %d frames, want 4 ping + 1 echo", got)
	}
	destX, destY, destZ, angle, speed, orient, tail = decodePlayerEcho(t, conn.Frames()[base+4])
	if destX != 946.54 || destZ != 1040.19 || speed != 0 || orient != 0 {
		t.Fatalf("stop echo dest=(%.2f,%.2f,%.2f) spd=%.2f ori=%.2f", destX, destY, destZ, speed, orient)
	}
	if math.Abs(float64(angle-0.071)) > 0.001 {
		t.Fatalf("stop echo angle=%.4f, want facing 0.071", angle)
	}
	if tail[2] != 0xA0000000 {
		t.Fatalf("stop echo tail id=%08x, want a0000000", tail[2])
	}

	// 跳跃双帧（state5）：官服零回显。
	base = len(conn.Frames())
	jump := clientCustomMessage{Opcode: 0x1E, Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomMovementUpload},
		{Type: 2, Int32: -1610592256 /* 0xA0005000 */},
		{Type: 2, Int32: 5},
		{Type: 4, Float32: 946.54}, {Type: 4, Float32: 43.89}, {Type: 4, Float32: 1040.19},
		{Type: 4, Float32: 0},
		{Type: 6, Text: "8A7BEA08F5434671B1EBBB0C9AE92783"},
	}}
	handled, err = handleMovementUpload(conn, player, world, jump, "test")
	if err != nil || !handled {
		t.Fatalf("jump handled=%t err=%v", handled, err)
	}
	if got := len(conn.Frames()) - base; got != 0 {
		t.Fatalf("jump emitted %d frames, want 0", got)
	}

	// 走停态复位验证：跳跃后的下一次起跑仍算迁移（重新四连）。
	base = len(conn.Frames())
	if _, err := handleMovementUpload(conn, player, world, walkUploadMessage(
		-1610608895, 7, 946.54, 43.89, 1040.19, 951.59, 1037.08, 4.50, 3.14), "test"); err != nil {
		t.Fatal(err)
	}
	if got := len(conn.Frames()) - base; got != 5 {
		t.Fatalf("post-jump walk start emitted %d frames, want 5", got)
	}
}

// 负角归一 [0,2π)（官服 -1.076→5.21 样本）与冲锋在途不回显。
func TestMovementEchoAngleWrapAndRushGate(t *testing.T) {
	if got := moveEchoAngle(-9.34, 5.04); math.Abs(float64(got-5.207)) > 0.001 {
		t.Fatalf("moveEchoAngle(-9.34,5.04)=%.4f, want 5.207", got)
	}
	if got := moveEchoAngle(0.72, 10.21); math.Abs(float64(got-0.0705)) > 0.001 {
		t.Fatalf("moveEchoAngle(0.72,10.21)=%.4f, want 0.0705", got)
	}

	// 冲锋在途：回显闸关闭（位移由 tryRushArrival 消费）。
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	placeTestNPC(t, world, 91, 10, 0)
	player := newPlayerActor("冲锋侠", 0)
	definition := combatSkillDefinition{
		id: "CS_rush_echo", canRush: true, script: "skilllock", castRange: 8,
		bindDistance: defaultRushBindDistance, groundRushAction: defaultGroundRushAction,
	}
	if _, err := stageRushCast(conn, player, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 10, Y: 0, Z: 0}, worldcore.Transform{X: 0, Y: 0, Z: 0}, 10, "test"); err != nil {
		t.Fatal(err)
	}
	base := len(conn.Frames())
	if _, err := handleMovementUpload(conn, player, world, walkUploadMessage(
		-1610608895, 7, 2, 0, 0, 9, 0, 4.5, 3.14), "test"); err != nil {
		t.Fatal(err)
	}
	for _, frame := range conn.Frames()[base:] {
		if frame[0] == 0x21 {
			t.Fatal("rush in-flight walk must not emit 0x21 player echo")
		}
	}
}
