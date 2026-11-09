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

// registerPassiveTestCombatNPC 注册 NotPositive=1 的 AttackNpc——城内
// 白名 NPC（掷斧监工/背柴工人/剧情人物）的数据形态：creator 条目属性
// 层写入（NotPositive 官方属性表 idx 62，Int32）。
func registerPassiveTestCombatNPC(t *testing.T, lifecycle *sceneLifecycle, id uint32) {
	t.Helper()
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_passive"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.resolved.Properties["NotPositive"] = clientdata.PropertyValue{Value: clientdata.Int32Value(1)}
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
	transform, ok := lifecycle.npcTransform(worldcore.EntityID(forkNPCID(id)))
	if !ok {
		t.Fatalf("NPC %d has no transform", id)
	}
	transform.X, transform.Z = x, z
	lifecycle.writeNpcTransform(worldcore.EntityID(forkNPCID(id)), transform)
}

// TestNewNPCCombatStateHonorsNotPositive 模板位断言：NotPositive=1 →
// Passive（不主动索敌）；缺省/0（野外怪）→ 主动。creator 属性与模板列
// 同名时 resolver 以 creator 层为最终值（后写覆盖）。
func TestNewNPCCombatStateHonorsNotPositive(t *testing.T) {
	home := worldcore.Transform{X: 1, Y: 0, Z: 0}
	active := newNPCCombatState(testNPCSpawn(), home)
	if active.Passive {
		t.Fatal("absent NotPositive must stay active (wilderness monster)")
	}
	passive := testNPCSpawn()
	passive.resolved.Properties["NotPositive"] = clientdata.PropertyValue{Value: clientdata.Int32Value(1)}
	if state := newNPCCombatState(passive, home); !state.Passive {
		t.Fatal("NotPositive=1 must mark brain passive")
	}
	zero := testNPCSpawn()
	zero.resolved.Properties["NotPositive"] = clientdata.PropertyValue{Value: clientdata.Int32Value(0)}
	if state := newNPCCombatState(zero, home); state.Passive {
		t.Fatal("NotPositive=0 must stay active")
	}
}

// TestPassiveNPCDoesNotAggroNearbyPlayer 复刻真机反馈的根因：玩家走到
// 城内白名 AttackNpc（NotPositive=1）身边，NPC 不追不咬、玩家不入战。
func TestPassiveNPCDoesNotAggroNearbyPlayer(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerPassiveTestCombatNPC(t, lifecycle, 101)
	player := newPlayerActor("tester", 0)
	before := player.stateSnapshot().HP
	now := time.Unix(8000, 0)
	// 连跑多个 tick：被动怪必须始终原地待命。
	for i := 0; i < 3; i++ {
		result := lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, now.Add(time.Duration(i)*time.Second))
		if result.playerStateChanged || result.playerRevived {
			t.Fatalf("passive NPC aggroed idle player at tick %d: %+v", i, result)
		}
	}
	if after := player.stateSnapshot().HP; after != before {
		t.Fatalf("passive NPC damaged idle player: HP=%d want %d", after, before)
	}
	if logic := player.stateSnapshot().LogicState; logic != entity.LogicStateNormal {
		t.Fatalf("passive NPC dragged idle player into combat: LogicState=%d", logic)
	}
	if brain, _ := lifecycle.run.combat.Brain(worldcore.EntityID(forkNPCID(101))); brain.Threat != 0 || brain.TargetMember != 0 {
		t.Fatalf("passive NPC built threat=%d target=%d without being hit", brain.Threat, brain.TargetMember)
	}
	lifecycle.close()
}

// TestPassiveNPCRetaliatesWhenHit 受击反击闭环：白名 NPC 被打一下后锁定
// 攻击者，随后按普通追击/攻击流程反击（Java NpcCombatAi 纯反击模型）。
func TestPassiveNPCRetaliatesWhenHit(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerPassiveTestCombatNPC(t, lifecycle, 102)
	player := newPlayerActor("tester", 0)
	now := time.Unix(8100, 0)
	// 首 tick 登记 s.wireID（受击锁定攻击者要用），被动怪原地待命。
	lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, now)
	// 玩家命中 NPC 一次（applySkillDamageToNPC 真实入口；伤害 1 进威胁表）。
	if _, err := lifecycle.applySkillDamageToNPC(forkNPCID(102), npcTestOwner(102), 1, nil, now); err != nil {
		t.Fatal(err)
	}
	lifecycle.mu.Lock()
	attackerWireID := lifecycle.wireID
	lifecycle.mu.Unlock()
	brain, ok := lifecycle.run.combat.Brain(worldcore.EntityID(forkNPCID(102)))
	if !ok {
		t.Fatal("passive NPC must have a brain")
	}
	if brain.Threat == 0 || brain.TargetMember != attackerWireID {
		t.Fatalf("hit did not lock attacker: threat=%d target=%d attacker=%d", brain.Threat, brain.TargetMember, attackerWireID)
	}
	// 攻击周期 2.2s 后的 tick：受击锁定的成员就在脚边，必须真反击掉血。
	before := player.stateSnapshot().HP
	result := lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, now.Add(npcAttackEvery))
	if after := player.stateSnapshot().HP; after >= before {
		t.Fatalf("passive NPC did not retaliate after being hit: HP=%d want below %d", after, before)
	}
	if !result.playerStateChanged {
		t.Fatal("retaliation must change player state")
	}
	var sawHit bool
	for _, frame := range conn.Frames() {
		if frame[0] == 0x1E && int32(binary.LittleEndian.Uint32(frame[4:])) == 24 {
			sawHit = true
		}
	}
	if !sawHit {
		t.Fatal("retaliation attack frames missing msg24 hit")
	}
	lifecycle.close()
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
	// NPC 攻击表现帧（Java NpcCombatAi 出招簇）：msg337 预备 → msg20
	// 动作 → msg24 命中 → 0x2D 受击特效。msg20 统一 0x1E 壳（42 份抓包
	// 0 次走 0x27，2026-11-07 战斗重写对齐）。
	var sawPrep, sawAnim, sawHit bool
	for _, frame := range conn.Frames() {
		if frame[0] != 0x1E {
			continue
		}
		switch int32(binary.LittleEndian.Uint32(frame[4:])) {
		case 337:
			sawPrep = true
		case 20:
			sawAnim = true
		case 24:
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
	id := worldcore.EntityID(forkNPCID(83))
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
	brain, ok := lifecycle.run.combat.Brain(worldcore.EntityID(forkNPCID(85)))
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
	id := worldcore.EntityID(forkNPCID(86))
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

// TestCombatTickClearsNPCFightingAfterLeash 复刻真机反馈：玩家打怪一下后
// 脱离（死亡/离场/拉出栓绳距离），怪清仇恨回家——旧实现 Vitals.Logic 停在
// Fighting 永不回落（脱战只随重生发生），客户端一直显示战斗态怪。修复后
// 清仇恨 tick 完全重置（HP/瘀血/MP 回满 + Fighting→Normal）并补发 0x10
// 属性帧（463=0），帧形与官方重置帧一致（55866/12141/52905 抓包：NPC 463
// 生命周期 0→1→120→0，0 帧伴随资源回满；栓绳样本无抓包，回满为拍板行为）。
func TestCombatTickClearsNPCFightingAfterLeash(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 87)
	id := worldcore.EntityID(forkNPCID(87))
	vitals, ok := lifecycle.run.combat.Vitals(id)
	if !ok {
		t.Fatal("combat NPC must have vitals")
	}
	vitals.ApplyDamage(1) // 玩家命中：入战 Fighting + 瘀血层
	lifecycle.run.combat.WriteVitals(id, vitals)

	player := newPlayerActor("tester", 0)
	now := time.Unix(7000, 0)
	// 近身 tick：建立仇恨并咬住目标；combatTickForTest 每次调用都
	// 登记一个新成员（旧成员留在怪脚边会被继续咬），先跑脱离 tick
	// 再注销旧成员——顺序颠倒会因成员清空触发场景关闭。
	lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, now)
	lifecycle.mu.Lock()
	staleWireID := lifecycle.wireID
	lifecycle.mu.Unlock()
	lifecycle.combatTickForTest(player, role.Position{X: 500, Y: 0, Z: 500}, now.Add(time.Second))
	lifecycle.run.detachMember(staleWireID)

	// 玩家拉出栓绳距离：清仇恨回位 + 战斗态回落。怪本就在家，
	// 该 tick 唯一新增帧即 463=0 的 NPC 属性帧。
	lifecycle.combatTickForTest(player, role.Position{X: 500, Y: 0, Z: 500}, now.Add(2*time.Second))
	if cleared, _ := lifecycle.run.combat.Vitals(id); cleared.Logic != entity.LogicStateNormal {
		t.Fatalf("NPC LogicState=%d after leash, want Normal", cleared.Logic)
	} else if cleared.HP != cleared.MaxHP || cleared.HitHP != cleared.MaxHP {
		t.Fatalf("leash must refill to full: HP=%d HitHP=%d MaxHP=%d", cleared.HP, cleared.HitHP, cleared.MaxHP)
	}
	if brain, _ := lifecycle.run.combat.Brain(id); brain.Threat != 0 || brain.TargetMember != 0 {
		t.Fatalf("brain threat=%d target=%d after leash, want cleared", brain.Threat, brain.TargetMember)
	}
	var logicState *byte
	for _, frame := range conn.Frames() {
		if frame[0] != 0x10 || binary.LittleEndian.Uint32(frame[2:]) != forkNPCID(87) {
			continue
		}
		// npcCombatProperties 线宽：21/463/464=1B，28/30/32/465/469/578=4B。
		off := 12
		for i := 0; i < int(binary.LittleEndian.Uint16(frame[10:])); i++ {
			idx := binary.LittleEndian.Uint16(frame[off:])
			off += 2
			width := map[uint16]int{21: 1, 463: 1, propDead: 1}[idx]
			if width == 0 {
				width = 4
			}
			if idx == 463 {
				value := frame[off]
				logicState = &value
			}
			off += width
		}
		if off != len(frame) {
			t.Fatalf("NPC property frame len=%d parsed=%d (layout drift)", len(frame), off)
		}
	}
	if logicState == nil {
		t.Fatal("leash tick did not broadcast NPC LogicState frame")
	}
	if *logicState != entity.LogicStateNormal {
		t.Fatalf("NPC LogicState frame=%d, want Normal(0)", *logicState)
	}
	lifecycle.close()
}

// autoSelectFrames 收集受害者连接上的 LastObject(414)/AutoSelectObject(415)
// 单属性 0x10 帧（各 22B：n@10=1、idx@12、target@14/18），返回
// idx→(targetID,targetOwner)。登录全量 0x10 帧属性数远大于 1，天然被过滤。
func autoSelectFrames(conn *captureMessageConnection) map[uint16][2]uint32 {
	found := map[uint16][2]uint32{}
	for _, frame := range conn.Frames() {
		if len(frame) != 22 || frame[0] != 0x10 || binary.LittleEndian.Uint16(frame[10:]) != 1 {
			continue
		}
		idx := binary.LittleEndian.Uint16(frame[12:])
		if idx == 414 || idx == 415 {
			found[idx] = [2]uint32{binary.LittleEndian.Uint32(frame[14:]), binary.LittleEndian.Uint32(frame[18:])}
		}
	}
	return found
}

func TestNPCAttackAutoSelectsAttackerForUntargetedVictim(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 91)
	player := newPlayerActor("tester", 0)
	lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, time.Unix(3000, 0))
	found := autoSelectFrames(conn)
	for _, idx := range []uint16{414, 415} {
		target, ok := found[idx]
		if !ok {
			t.Fatalf("attacker auto-select frame idx=%d missing", idx)
		}
		if target[0] != forkNPCID(91) || target[1] != npcTestOwner(91) {
			t.Fatalf("idx=%d selected %d-%d, want 91-1", idx, target[0], target[1])
		}
	}
	if lifecycle.selectedObject != forkNPCID(91) || lifecycle.selectedOwner != npcTestOwner(91) {
		t.Fatalf("server selection mirror = %d-%d, want 91-1", lifecycle.selectedObject, lifecycle.selectedOwner)
	}
	// 自动选中必须立刻打通 211 选靶路径（skillTargetSelected 族）。
	if id, owner, ok := lifecycle.selectedCombatTarget(); !ok || id != forkNPCID(91) || owner != npcTestOwner(91) {
		t.Fatalf("selectedCombatTarget after auto-select = %d-%d ok=%t", id, owner, ok)
	}
	lifecycle.close()
}

func TestNPCAttackKeepsPlayerManualSelection(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 92)
	registerTestCombatNPC(t, lifecycle, 93)
	placeTestNPC(t, lifecycle, 93, 200, 200) // 仇恨圈外保持存活的手动目标
	lifecycle.selectedObject, lifecycle.selectedOwner = forkNPCID(93), npcTestOwner(93)
	player := newPlayerActor("tester", 0)
	lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, time.Unix(3100, 0))
	if lifecycle.selectedObject != forkNPCID(93) || lifecycle.selectedOwner != npcTestOwner(93) {
		t.Fatalf("manual target overridden by auto-select: now %d-%d", lifecycle.selectedObject, lifecycle.selectedOwner)
	}
	if found := autoSelectFrames(conn); len(found) != 0 {
		t.Fatalf("auto-select frames must not fire while a living manual target exists: %v", found)
	}
	lifecycle.close()
}

func TestNPCAttackReplacesStaleSelection(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "resource", nil); err != nil {
		t.Fatal(err)
	}
	registerTestCombatNPC(t, lifecycle, 94)
	// 已消亡对象占位的陈旧格与空选同权，攻击者应当顶替。
	lifecycle.selectedObject, lifecycle.selectedOwner = forkNPCID(949494), npcTestOwner(949494)
	player := newPlayerActor("tester", 0)
	lifecycle.combatTickForTest(player, role.Position{X: 1, Y: 0, Z: 0}, time.Unix(3200, 0))
	if lifecycle.selectedObject != forkNPCID(94) || lifecycle.selectedOwner != npcTestOwner(94) {
		t.Fatalf("stale selection not replaced: %d-%d", lifecycle.selectedObject, lifecycle.selectedOwner)
	}
	found := autoSelectFrames(conn)
	if _, ok := found[415]; !ok {
		t.Fatal("stale selection replacement must push AutoSelectObject frame")
	}
	lifecycle.close()
}
