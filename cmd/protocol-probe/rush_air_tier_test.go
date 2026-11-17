package main

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	worldcore "github.com/local/9yin-go-server/internal/world"
)

// TestRushDashAirTierDrive 技能4.pcapng 分档（2026-11-15）：空中/高对低冲刺
// 0x21 s5=AirRushSpeed(15)（地面 25）+ 空中授予补 m707[29]（+40~80ms）+
// 冲刺已发后的远带 212=到位申报照发释放（#78 悬空 212 官服 m26+40ms，零取消）。
func TestRushDashAirTierDrive(t *testing.T) {
	conn := &captureMessageConnection{}
	world := newSceneLifecycle("test", conn)
	if err := world.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, world, 91)
	placeTestNPC(t, world, 91, 20, 0)
	player := newPlayerActor("空袭侠", 0)
	definition := combatSkillDefinition{id: "CS_rush_air", canRush: true, script: "skilllock",
		bindDistance: defaultRushBindDistance, castRange: 3,
		groundRushAction: defaultGroundRushAction, airRushAction: "qg_fastruning_s_2h_01"}
	player.mu.Lock()
	player.learnedSkills[definition.id] = 1
	player.mu.Unlock()
	if err := handleSelectTargetCustom(world, player, warningToggleMessage(clientCustomSelectTarget,
		clientCustomValue{Type: 8, Raw: packTestObject(forkNPCID(91), npcTestOwner(91))}), "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := stageRushCast(conn, player, world, definition, forkNPCID(91), npcTestOwner(91),
		worldcore.Transform{X: 20, Y: 0, Z: 0}, worldcore.Transform{X: 0, Y: 3, Z: 0}, 20, "test"); err != nil {
		t.Fatal(err)
	}

	// 空中上报（y=3 高出 NPC 平面 1.5+）进入冲刺带 (3,5]。
	base := len(conn.Frames())
	if _, err := handleMovementUpload(conn, player, world,
		walkUploadMessage(-1610608895, 7, 16, 3, 0, 20, 0, 4.5, 0), "test"); err != nil {
		t.Fatal(err)
	}
	frames := conn.Frames()[base:]
	var sawC61Air bool
	for _, f := range frames {
		if f[0] == 0x1E {
			values := decodeCustomValuesFrame(t, f)
			if len(values) >= 8 && values[0].Type == 2 && values[0].Int32 == 24 &&
				values[2].Type == 2 && values[2].Int32 == msgRushCommand &&
				values[6].Type == 2 && values[6].Int32 == 2 {
				sawC61Air = true
			}
		}
	}
	// v31（2026-11-17 去 0x21）：空中档不再有 0x21 驱动条目——c61 授予即
	// 冲锋（客户端自走，技能5.pcapng 官方模型）。
	for _, f := range frames {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 1 {
			t.Fatalf("v31 client self-run must not emit 0x21 drive entries: %x", f)
		}
	}
	if !sawC61Air {
		t.Fatalf("air grant must be c61 flag=2, frames=%x", frames)
	}

	// m707[29]：空中授予后 +60ms 补发。
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
		t.Fatal("air grant must follow with m707[29]")
	}

	// 冲刺已发后的远带 212 = 到位申报照发释放（旧方言=cancel，技能4 #78 作废）：
	// 释放走落地簇（带 0x1F 锚），取消路径无 0x1F。
	if player.peekPendingCast(time.Now()) == nil {
		t.Fatal("pending must survive until 212/ETA")
	}
	if _, err := handleCastDashLanding(conn, player, world,
		warningToggleMessage(clientCustomCastDashLanding,
			clientCustomValue{Type: 4, Float32: 10}, clientCustomValue{Type: 4, Float32: 3},
			clientCustomValue{Type: 4, Float32: 0}, clientCustomValue{Type: 4, Float32: 0},
			clientCustomValue{Type: 2, Int32: 1}), "test"); err != nil {
		t.Fatal(err)
	}
	sawAnchor := false
	for _, f := range conn.Frames() {
		if f[0] == 0x1F {
			sawAnchor = true
		}
	}
	if !sawAnchor {
		t.Fatal("far 212 with dash issued must release (landing cluster 0x1F), not cancel")
	}
	if player.peekPendingCast(time.Now()) != nil {
		t.Fatal("far 212 release must consume pending")
	}

	// 2026-11-16「穿云纵后续有概率用不出」修复：212 抢先于 ETA 的释放
	// 出口必须带轻功绑定释放帧（0x21 空置条目 speed=0/intent=0xA0000700）
	// ——旧版该帧只在 ETA 路径发，早退后轻功状态残留锁死后续轻功。
	sawQGRelease := false
	for _, f := range conn.Frames() {
		if f[0] == 0x21 && len(f) == 47 && binary.LittleEndian.Uint32(f[43:]) == 0xA0000700 {
			if got := math.Float32frombits(binary.LittleEndian.Uint32(f[27:])); got == 0 {
				sawQGRelease = true
			}
		}
	}
	if !sawQGRelease {
		t.Fatal("212-wins race release must carry qinggong release frame (0x21 idle, speed=0, intent=0xA0000700)")
	}
}
