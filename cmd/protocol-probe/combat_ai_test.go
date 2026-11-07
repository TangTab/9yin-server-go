package main

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

func registerTestCombatNPC(t *testing.T, lifecycle *sceneLifecycle, id uint32) {
	t.Helper()
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_test"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.x, npc.y, npc.z = 1, 0, 0
	if err := lifecycle.registerNPCOnRun(id, npc); err != nil {
		t.Fatal(err)
	}
}

// placeTestNPC 把已注册的测试 NPC 摆到指定 X/Z。注册点固定为
// (1,0,0)，因此显式写全两个坐标与原先只改 transform.X 的戳点
// 行为一致。
func placeTestNPC(t *testing.T, lifecycle *sceneLifecycle, id uint32, x, z float32) {
	t.Helper()
	transform, ok := lifecycle.npcTransform(worldcore.EntityID(id))
	if !ok {
		t.Fatalf("NPC %d has no transform", id)
	}
	transform.X, transform.Z = x, z
	lifecycle.writeNpcTransform(worldcore.EntityID(id), transform)
}

func TestCombatTickAggroesAndDamagesNearbyPlayer(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 81)
	player := newPlayerActor("tester", 0)
	before := player.stateSnapshot().HP
	result := lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, time.Unix(1000, 0))
	if !result.playerStateChanged {
		t.Fatal("nearby hostile did not produce a player state update")
	}
	if after := player.stateSnapshot().HP; after >= before {
		t.Fatalf("NPC attack HP=%d, want below %d", after, before)
	}
	// 三段攻击表现帧（F 盘 writeNPCAttackFrames 同构）：
	// msg337 预备（0x1E）→ msg20 动作（0x27）→ msg24 命中（0x1E）。
	var sawPrep, sawAnim, sawHit bool
	for _, frame := range conn.Frames() {
		switch {
		case frame[0] == 0x1E && int32(binary.LittleEndian.Uint32(frame[4:])) == 337:
			sawPrep = true
		case frame[0] == 0x27:
			sawAnim = true
		case frame[0] == 0x1E && int32(binary.LittleEndian.Uint32(frame[4:])) == 24:
			sawHit = true
		}
	}
	if !sawPrep || !sawAnim || !sawHit {
		t.Fatalf("NPC attack frames incomplete: prep=%t anim=%t hit=%t", sawPrep, sawAnim, sawHit)
	}
}

func TestCombatTickHonorsFreshActiveParry(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 82)
	player := newPlayerActor("tester", 0)
	now := time.Unix(2000, 0)
	player.setActiveParry(activeParryRequest{Subtype: 0, At: now})
	before := player.stateSnapshot().HP
	result := lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, now)
	if after := player.stateSnapshot().HP; after != before {
		t.Fatalf("parry HP=%d, want %d", after, before)
	}
	if !result.playerStateChanged || player.stateSnapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("fresh parry must still enter combat, result=%+v actor=%+v", result, player.stateSnapshot())
	}
}

func TestCombatThreatInterruptsSitcrossAndClearsAfterLeash(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 83)
	player := newPlayerActor("tester", 0)
	player.setLogicState(entity.LogicStateSitcross)
	now := time.Unix(2500, 0)
	result := lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, now)
	if !result.playerStateChanged || player.stateSnapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("aggro state result=%+v actor=%+v", result, player.stateSnapshot())
	}
	// combatTickForTest 每次调用都会登记一个新成员；先跑栓绳 tick（新
	// 成员在场，运行时不因成员清空关闭），再注销留在怪脚边的旧成员，
	// 避免它持续被咬刷新战斗活动时间（真实连接每视图只 attach 一次，
	// 无此现象）。
	lifecycle.mu.Lock()
	staleWireID := lifecycle.wireID
	lifecycle.mu.Unlock()
	// 栓绳后 1s：威胁已消失，但 10s 脱战迟滞窗口内保持 Fighting
	//（击杀清仇恨/栓绳间隙不闪发脱战帧）。
	result = lifecycle.combatTickForTest(player, role.Position{X: 100, Y: 0, Z: 0}, now.Add(time.Second))
	if result.playerStateChanged || player.stateSnapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("hysteresis state result=%+v actor=%+v", result, player.stateSnapshot())
	}
	lifecycle.run.detachMember(staleWireID)
	// 10s 无战斗活动后才真正脱战。
	result = lifecycle.combatTickForTest(player, role.Position{X: 100, Y: 0, Z: 0}, now.Add(combatPresenceIdleExit+time.Second))
	if !result.playerStateChanged || player.stateSnapshot().LogicState != entity.LogicStateNormal {
		t.Fatalf("leash state result=%+v actor=%+v", result, player.stateSnapshot())
	}
	lifecycle.close()
}

// TestCombatPresenceHoldsAfterKillAndExitsAfterIdle 复刻真机反馈：
// 战斗中（含击杀清仇恨、怪未及锁定自己的间隙）不再闪发脱战帧；
// 命中即入战；10s 无战斗活动才单次脱战。
func TestCombatPresenceHoldsAfterKillAndExitsAfterIdle(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 84)
	player := newPlayerActor("tester", 0)
	// 玩家远离怪（无威胁），命中怪一次（noteCombatActivity 同款路径）。
	now := time.Unix(4000, 0)
	player.noteCombatActivity(now)
	result := lifecycle.combatTickForTest(player, role.Position{X: 500, Y: 0, Z: 500}, now)
	if !result.playerStateChanged || player.stateSnapshot().LogicState != entity.LogicStateFighting {
		t.Fatalf("hit-enter state result=%+v actor=%+v", result, player.stateSnapshot())
	}
	// 连续多个无威胁 tick：保持 Fighting，不产生状态变化帧。
	for i := 1; i <= 3; i++ {
		result = lifecycle.combatTickForTest(player, role.Position{X: 500, Y: 0, Z: 500}, now.Add(time.Duration(i)*time.Second))
		if result.playerStateChanged {
			t.Fatalf("tick +%ds must not flip presence (no threat but inside 10s window)", i)
		}
	}
	// 10s 窗口过后单次脱战。
	result = lifecycle.combatTickForTest(player, role.Position{X: 500, Y: 0, Z: 500}, now.Add(combatPresenceIdleExit+2*time.Second))
	if !result.playerStateChanged || player.stateSnapshot().LogicState != entity.LogicStateNormal {
		t.Fatalf("idle exit state result=%+v actor=%+v", result, player.stateSnapshot())
	}
	lifecycle.close()
}

func TestCombatTickRespawnsDeadHostileAtHome(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 83)
	id := worldcore.EntityID(83)
	vitals, ok := lifecycle.run.combat.Vitals(id)
	if !ok {
		t.Fatal("combat NPC must have vitals")
	}
	vitals.ApplyDamage(999999)
	lifecycle.run.combat.WriteVitals(id, vitals)
	brain, ok := lifecycle.run.combat.Brain(id)
	if !ok {
		t.Fatal("combat NPC must have a brain")
	}
	now := time.Unix(3000, 0)
	brain.RespawnAt = now
	lifecycle.run.combat.WriteBrain(id, brain)
	player := newPlayerActor("tester", 0)
	lifecycle.combatTickForTest(player, role.Position{X: 999, Y: 0, Z: 999}, now)
	if revived, _ := lifecycle.run.combat.Vitals(id); revived.HP != revived.MaxHP || revived.Logic != entity.LogicStateNormal {
		t.Fatalf("respawn vitals=%+v", revived)
	}
	frames := conn.Frames()
	if len(frames) != 2 || frames[0][0] != 0x20 || frames[1][0] != 0x10 {
		t.Fatalf("respawn frames=%x", frames)
	}
}

func TestCombatMoveDoesNotImmediatelyConfirmLocation(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 84)
	player := newPlayerActor("tester", 0)
	lifecycle.combatTickForTest(player, role.Position{X: 10, Y: 0, Z: 0}, time.Unix(4000, 0))
	frames := conn.Frames()
	if len(frames) != 1 || frames[0][0] != 0x20 {
		t.Fatalf("first chase tick frames=%x, want only ServerMoving", frames)
	}
	lifecycle.close()
}

// TestNPCRespawnAfterHonorsOfficialRefreshTime 重生间隔消费模板官方字段
// RefreshTime（VisibleNPCModernV1 idx 93，单位秒）：1..3600 生效，空值/
// 越界回落默认 15s。worldnpc/bossnpc 实测该列基本为空，默认路径不变。
func TestNPCRespawnAfterHonorsOfficialRefreshTime(t *testing.T) {
	refreshed := testNPCSpawn()
	refreshed.resolved.ScriptClass = "AttackNpc"
	refreshed.resolved.Properties["RefreshTime"] = clientdata.PropertyValue{Value: clientdata.Int32Value(2)}
	for _, tc := range []struct {
		name string
		npc  npcSpawn
		want time.Duration
	}{
		{"template seconds", refreshed, 2 * time.Second},
		{"absent falls back", testNPCSpawn(), defaultNPCRespawn},
	} {
		if got := npcRespawnAfter(tc.npc); got != tc.want {
			t.Fatalf("%s: respawn=%v want %v", tc.name, got, tc.want)
		}
	}
	outOfRange := refreshed
	outOfRange.resolved.Properties = make(map[string]clientdata.PropertyValue, len(refreshed.resolved.Properties))
	for name, value := range refreshed.resolved.Properties {
		outOfRange.resolved.Properties[name] = value
	}
	outOfRange.resolved.Properties["RefreshTime"] = clientdata.PropertyValue{Value: clientdata.Int32Value(7200)}
	if got := npcRespawnAfter(outOfRange); got != defaultNPCRespawn {
		t.Fatalf("out-of-range refresh time must fall back: %v", got)
	}

	// 注册链路真实消费：AttackNpc 挂 Brain 时带上模板间隔。
	lifecycle := newSceneLifecycle("test", discardMessageConnection{})
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.registerNPCOnRun(85, refreshed); err != nil {
		t.Fatal(err)
	}
	brain, ok := lifecycle.run.combat.Brain(worldcore.EntityID(85))
	if !ok || brain.RespawnAfter != 2*time.Second {
		t.Fatalf("registered brain respawn=%v ok=%t", brain.RespawnAfter, ok)
	}
	lifecycle.close()
}

// TestCombatTickRespawnsAfterTemplateRefreshTime RefreshTime=2 的怪被杀后：
// +1s 仍是尸体，+2s 满血复活——间隔来自模板而非写死 15s。
func TestCombatTickRespawnsAfterTemplateRefreshTime(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_refresh"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.resolved.Properties["RefreshTime"] = clientdata.PropertyValue{Value: clientdata.Int32Value(2)}
	if err := lifecycle.registerNPCOnRun(86, npc); err != nil {
		t.Fatal(err)
	}
	id := worldcore.EntityID(86)
	killTime := time.Unix(6000, 0)
	vitals, ok := lifecycle.run.combat.Vitals(id)
	if !ok {
		t.Fatal("combat NPC must have vitals")
	}
	vitals.ApplyDamage(999999)
	lifecycle.run.combat.WriteVitals(id, vitals)
	brain, _ := lifecycle.run.combat.Brain(id)
	brain.RespawnAt = killTime.Add(brain.RespawnAfter)
	lifecycle.run.combat.WriteBrain(id, brain)

	player := newPlayerActor("tester", 0)
	lifecycle.combatTickForTest(player, role.Position{X: 999, Y: 0, Z: 999}, killTime.Add(time.Second))
	if dead, _ := lifecycle.run.combat.Vitals(id); dead.HP != 0 {
		t.Fatalf("NPC revived before template RefreshTime: %+v", dead)
	}
	lifecycle.combatTickForTest(player, role.Position{X: 999, Y: 0, Z: 999}, killTime.Add(2*time.Second))
	if revived, _ := lifecycle.run.combat.Vitals(id); revived.HP != revived.MaxHP || revived.Logic != entity.LogicStateNormal {
		t.Fatalf("NPC did not revive after template RefreshTime: %+v", revived)
	}
	lifecycle.close()
}
