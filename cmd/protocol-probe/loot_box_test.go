package main

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// pinRefreshParams 测试内固定刷新参数并在结束后还原（包级原子全局）。
func pinRefreshParams(t *testing.T, interval, ttl time.Duration) {
	t.Helper()
	prevInterval, prevTTL := npcRespawnBatchInterval(), npcCorpseTTL()
	setNpcRespawnBatchInterval(interval)
	setNpcCorpseTTL(ttl)
	t.Cleanup(func() {
		setNpcRespawnBatchInterval(prevInterval)
		setNpcCorpseTTL(prevTTL)
	})
}

// pinLootBoxTTL 测试内固定宝箱存留参数并在结束后还原（包级原子全局）。
func pinLootBoxTTL(t *testing.T, ttl time.Duration) {
	t.Helper()
	previous := lootBoxTTL()
	setLootBoxTTL(ttl)
	t.Cleanup(func() { setLootBoxTTL(previous) })
}

// newRefreshTestRun 构造双 AttackNpc（实体 2/3）+ 单成员视图的共享运行时。
func newRefreshTestRun(t *testing.T) (*sceneRun, *sceneLifecycle, *captureMessageConnection, *playerActor) {
	t.Helper()
	conn := &captureMessageConnection{}
	view := newSceneLifecycle("test", conn)
	view.radius = 50
	catalog := []npcSpawn{dropTestAttackNPC("mon_a"), dropTestAttackNPC("mon_b")}
	run := newSceneRun(sceneKey{config: "scene", resource: "resource"}, nil)
	if err := run.populate(catalog); err != nil {
		t.Fatal(err)
	}
	view.run = run
	player := newPlayerActor("tester", 0)
	view.attachMember(player, role.Position{X: 1}, roleVisual{})
	run.activateMember(view.wireID)
	view.refreshVisibility(catalog, role.Position{X: 1})
	return run, view, conn, player
}

// dropTestAttackNPC 带掉落绑定的目录 AttackNpc（Extensions 面与生产同形）。
func dropTestAttackNPC(configID string) npcSpawn {
	npc := testNPCSpawn()
	npc.resolved.ConfigID = configID
	npc.resolved.ScriptClass = "AttackNpc"
	npc.resolved.Extensions["template.DropID"] = "NormalMon00101"
	npc.x, npc.y, npc.z = 1, 0, 0
	return npc
}

// killAndRecord 模拟 apply* 的击杀结算（含死亡登记与宝箱生成）。
func killAndRecord(run *sceneRun, id uint32, now time.Time) {
	vitals, _ := run.combat.Vitals(worldcore.EntityID(id))
	vitals.ApplyDamage(999999)
	run.combat.WriteVitals(worldcore.EntityID(id), vitals)
	brain, _ := run.combat.Brain(worldcore.EntityID(id))
	run.recordNPCDeathLocked(id, 1, brain, now)
}

// killOnly 只置死并手工登记尸体（不生成宝箱），供纯刷新断言用。
func killOnly(run *sceneRun, id uint32, rec deadNPCRecord) {
	vitals, _ := run.combat.Vitals(worldcore.EntityID(id))
	vitals.ApplyDamage(999999)
	run.combat.WriteVitals(worldcore.EntityID(id), vitals)
	run.deadNPCs[id] = rec
}

// TestBatchRespawnSweepRevivesDeadAndLeavesAlive 批量扫描：死了的统一
// 重生（满血+回出生点），没死的一律不动（HP/坐标不变、无帧）。
func TestBatchRespawnSweepRevivesDeadAndLeavesAlive(t *testing.T) {
	pinRefreshParams(t, 5*time.Second, 0)
	run, view, conn, _ := newRefreshTestRun(t)
	defer view.close()

	now := time.Unix(10000, 0)
	run.refreshTick(now) // 先空扫一次，落定扫描基线（生产由每秒 tick 自然完成）
	killOnly(run, forkNPCID(2), deadNPCRecord{spawnIndex: 0, diedAt: now})
	aliveBefore, _ := run.combat.Vitals(worldcore.EntityID(forkNPCID(3)))
	aliveTransform, _ := run.npcTransform(worldcore.EntityID(forkNPCID(3)))

	run.refreshTick(now.Add(4 * time.Second))
	if dead, _ := run.combat.Vitals(worldcore.EntityID(forkNPCID(2))); dead.HP != 0 {
		t.Fatalf("NPC revived before sweep deadline: %+v", dead)
	}
	if _, busy := run.deadNPCs[forkNPCID(2)]; !busy {
		t.Fatal("dead record dropped before sweep")
	}

	run.refreshTick(now.Add(5 * time.Second))
	revived, _ := run.combat.Vitals(worldcore.EntityID(forkNPCID(2)))
	if revived.HP != revived.MaxHP {
		t.Fatalf("dead NPC not revived by sweep: %+v", revived)
	}
	if _, busy := run.deadNPCs[forkNPCID(2)]; busy {
		t.Fatal("dead record kept after sweep")
	}
	// 活怪不动：HP 与坐标原样。
	if alive, _ := run.combat.Vitals(worldcore.EntityID(forkNPCID(3))); alive.HP != aliveBefore.HP {
		t.Fatalf("alive NPC HP changed: %d -> %d", aliveBefore.HP, alive.HP)
	}
	if after, _ := run.npcTransform(worldcore.EntityID(forkNPCID(3))); after != aliveTransform {
		t.Fatalf("alive NPC moved: %+v -> %+v", aliveTransform, after)
	}
	// 复活帧组：0x20 回家 + 0x10 属性（与每怪复活同帧方言）。
	opcodes := frameOpcodes(conn.Frames())
	if !opcodes.has(0x20) || !opcodes.has(0x10) {
		t.Fatalf("respawn frames=%v", opcodes)
	}
}

// TestCorpseTTLRemovesAndPerNPCRespawnReRegisters 每怪模式 + 尸体 TTL：
// 超时 0x0E + 注销；到 respawnAt 按目录行重注册（实体号不变）+ 0x0D/0x1F。
func TestCorpseTTLRemovesAndPerNPCRespawnReRegisters(t *testing.T) {
	pinRefreshParams(t, 0, 2*time.Second)
	run, view, conn, _ := newRefreshTestRun(t)
	defer view.close()

	now := time.Unix(11000, 0)
	killOnly(run, forkNPCID(2), deadNPCRecord{spawnIndex: 0, diedAt: now, respawnAt: now.Add(15 * time.Second)})

	run.refreshTick(now.Add(3 * time.Second))
	frames := conn.Frames()
	if len(frames) != 1 || frames[0][0] != 0x0E {
		t.Fatalf("corpse TTL must emit exactly 0x0E, frames=%x", frames)
	}
	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2))); exists {
		t.Fatal("corpse must be unregistered after TTL")
	}
	if rec, ok := run.deadNPCs[forkNPCID(2)]; !ok || !rec.removed {
		t.Fatalf("dead record not marked removed: %+v ok=%t", rec, ok)
	}
	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(3))); !exists {
		t.Fatal("alive NPC must stay registered")
	}

	run.refreshTick(now.Add(15 * time.Second))
	frames = conn.Frames()
	if len(frames) != 3 || frames[1][0] != 0x0D || frames[2][0] != 0x1F {
		t.Fatalf("respawn must rematerialize with 0x0D+0x1F, frames=%x", frames)
	}
	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2))); !exists {
		t.Fatal("removed NPC must re-register under the same id")
	}
	if revived, _ := run.combat.Vitals(worldcore.EntityID(forkNPCID(2))); revived.HP != revived.MaxHP {
		t.Fatalf("re-registered NPC must be full HP: %+v", revived)
	}
	if _, busy := run.deadNPCs[forkNPCID(2)]; busy {
		t.Fatal("dead record kept after per-NPC respawn")
	}
}

// TestBatchRespawnReRegistersRemovedCorpse 批量模式：尸体先被 TTL 清除，
// 扫描点按目录行重注册（同 id）并直发 0x0D/0x1F。
func TestBatchRespawnReRegistersRemovedCorpse(t *testing.T) {
	pinRefreshParams(t, 5*time.Second, 2*time.Second)
	run, view, conn, _ := newRefreshTestRun(t)
	defer view.close()

	now := time.Unix(12000, 0)
	run.refreshTick(now) // 先空扫一次，落定扫描基线
	killOnly(run, forkNPCID(2), deadNPCRecord{spawnIndex: 0, diedAt: now})

	run.refreshTick(now.Add(3 * time.Second))
	if frames := conn.Frames(); len(frames) != 1 || frames[0][0] != 0x0E {
		t.Fatalf("corpse TTL frames=%x", frames)
	}
	run.refreshTick(now.Add(5 * time.Second))
	frames := conn.Frames()
	if len(frames) != 3 || frames[1][0] != 0x0D || frames[2][0] != 0x1F {
		t.Fatalf("sweep re-add frames=%x", frames)
	}
	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2))); !exists {
		t.Fatal("removed corpse must re-register on sweep")
	}
}

// TestKillAttachesLootToCorpse 普通怪击杀：掉落挂尸体本体（官服懒加载
// 方言）——不生成新实体、无 0x0D 物化；死亡中继随发 CanPick=1 +
// PickMember=击杀者名（客户端冒光+拾取光标判据）。
func TestKillAttachesLootToCorpse(t *testing.T) {
	pinRefreshParams(t, 0, 0)
	run, view, conn, _ := newRefreshTestRun(t)
	defer view.close()

	view.selectedObject, view.selectedOwner = forkNPCID(2), 1
	now := time.Unix(13000, 0)
	wireIDBefore := run.nextWireID
	outcome, err := view.applySelectedSkillDamage(999999, now)
	frame, id, killed := outcome.frame, outcome.id, outcome.killed
	if err != nil || !killed || frame == nil {
		t.Fatalf("kill settlement frame=%v id=%d killed=%t err=%v", frame, id, killed, err)
	}
	box, has := run.lootBoxes[forkNPCID(2)]
	if !has {
		t.Fatalf("kill must attach loot to the corpse (id=2), got %d boxes", len(run.lootBoxes))
	}
	if box.standalone {
		t.Fatal("plain monster loot must ride the corpse, not a standalone entity")
	}
	if len(box.contents) == 0 || len(box.taken) != len(box.contents) {
		t.Fatalf("loot contents=%+v", box.contents)
	}
	if run.nextWireID != wireIDBefore { // 挂尸不得消耗实体号（玩家已从该池取号）
		t.Fatalf("corpse loot must not consume a wire id: %d -> %d", wireIDBefore, run.nextWireID)
	}
	meta, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2)))
	if !exists || meta.lootBox != box {
		t.Fatalf("corpse meta must carry the loot runtime: exists=%t", exists)
	}

	view.relayNPCKillFrame(id, frame, "tester")
	opcodes := frameOpcodes(conn.Frames())
	if opcodes.has(0x0D) || opcodes.has(0x1F) {
		t.Fatalf("corpse loot must not materialize a new object, frames=%v", opcodes)
	}
	if !hasCorpsePickFrames(t, conn.Frames(), forkNPCID(2)) {
		t.Fatalf("kill relay must send CanPick/PickMember frames, frames=%v", opcodes)
	}
	if box.picker != "tester" {
		t.Fatalf("corpse loot picker=%q want tester", box.picker)
	}
}

// hasCorpsePickFrames 校验帧流里有尸体 obj 的 CanPick@76=1 与
// PickMember@1592 宽串两帧（0x10 单属性帧）。
func hasCorpsePickFrames(t *testing.T, frames [][]byte, corpseID uint32) bool {
	t.Helper()
	canPick, pickMember := false, false
	for _, frame := range frames {
		if len(frame) < 14 || frame[0] != 0x10 || frame[1] != 0 {
			continue
		}
		obj := binary.LittleEndian.Uint32(frame[2:])
		if obj != corpseID {
			continue
		}
		index := binary.LittleEndian.Uint16(frame[12:])
		switch index {
		case uint16(propCanPick):
			if len(frame) >= 15 && frame[14] == 1 {
				canPick = true
			}
		case uint16(propPickMember):
			if len(frame) >= 22 { // u16 idx + u32 len + utf16 名 + 终止 0
				pickMember = true
			}
		}
	}
	return canPick && pickMember
}

// TestDropBoxOpenPickCloseFlow 官方掉落箱窗口闭环（挂尸形态）：点击
// 尸体开窗（0x15+0x18+show_hide_drop_view(1)）→ 380 拾取入包 + 0x19
// 删行 → 拾空关窗（0x16+关窗 custom）+ 尸体 0x0E 消失并放行重生。
func TestDropBoxOpenPickCloseFlow(t *testing.T) {
	pinRefreshParams(t, 0, 0)
	run, view, conn, player := newRefreshTestRun(t)
	defer view.close()

	view.selectedObject, view.selectedOwner = forkNPCID(2), 1
	now := time.Unix(14000, 0)
	outcome, err := view.applySelectedSkillDamage(999999, now)
	frame, id, killed := outcome.frame, outcome.id, outcome.killed
	if err != nil || !killed {
		t.Fatalf("kill settlement err=%v killed=%t", err, killed)
	}
	view.relayNPCKillFrame(id, frame, "tester")
	box := run.lootBoxes[forkNPCID(2)]
	firstItem := box.contents[0]

	openedBase := len(conn.Frames())
	if _, err := view.objectRequest(clientObjectRequest{Sequence: 7, ObjectID: forkNPCID(2), OwnerID: 1}); err != nil {
		t.Fatalf("open drop box on corpse: %v", err)
	}
	openFrames := frameOpcodes(conn.Frames()[openedBase:])
	if !openFrames.has(0x15) || !openFrames.has(0x18) || !openFrames.has(0x1E) {
		t.Fatalf("open frames=%v", openFrames)
	}
	if view.openDropBox == nil || len(view.openDropBox.slots) != len(box.contents) {
		t.Fatalf("open state=%+v", view.openDropBox)
	}

	// 拾取第一个槽位：game_items 目录命中 + 写穿 INSERT（sqlmock）。
	store, mock := newMockInventoryDB(t)
	rows := mock.NewRows([]string{"item_id", "item_type", "view_id", "item_name", "equip_type", "script"}).
		AddRow(firstItem.configID, 2101, 1, "测试掉落", nil, nil)
	mock.ExpectQuery("SELECT item_id, item_type, view_id, item_name, equip_type, script FROM game_items").WillReturnRows(rows)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT slot FROM role_bag_items").WillReturnRows(mock.NewRows([]string{"slot"}))
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1 FROM role_bag_items").WillReturnRows(
		mock.NewRows([]string{"seq"}).AddRow(1))
	mock.ExpectExec("INSERT INTO role_bag_items").WillReturnResult(newSQLMockResult(1, 1))
	mock.ExpectCommit()
	session := newTestItemSession(conn, player, store)

	pickBase := len(conn.Frames())
	handled, err := handlePickupDropItemCustom(view, session, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomPickupDropItem},
		{Type: 2, Int32: 1},
	}}, "test")
	if err != nil || !handled {
		t.Fatalf("pickup handled=%t err=%v", handled, err)
	}
	if slot := bagSlotOf(t, player, firstItem.configID, 1); slot == 0 {
		t.Fatalf("picked item %s not in bag", firstItem.configID)
	}
	pickFrames := frameOpcodes(conn.Frames()[pickBase:])
	if !pickFrames.has(0x19) {
		t.Fatalf("pickup must remove the view row, frames=%v", pickFrames)
	}
	if len(box.contents) == 1 {
		// 单件拾完：关窗 + 尸体消失（挂尸形态 0x0E 的对象就是尸体），
		// 且尸体登记标 removed（重生放行，防批量扫原地复活吞重生）。
		if !pickFrames.has(0x16) || !pickFrames.has(0x0E) {
			t.Fatalf("emptied box must close the view and despawn, frames=%v", pickFrames)
		}
		if len(run.lootBoxes) != 0 {
			t.Fatalf("emptied box still on scene: %d", len(run.lootBoxes))
		}
		if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2))); exists {
			t.Fatal("emptied corpse must be unregistered")
		}
		if rec, ok := run.deadNPCs[forkNPCID(2)]; !ok || !rec.removed {
			t.Fatalf("emptied corpse must be marked removed for respawn: %+v ok=%t", rec, ok)
		}
		if view.openDropBox != nil {
			t.Fatal("drop box view must be closed after emptying")
		}
	}
}

// TestDropBoxCloseKeepsBox 382 关箱：窗口关闭但尸体与掉落留场（TTL/
// 他人可再开）。
func TestDropBoxCloseKeepsBox(t *testing.T) {
	pinRefreshParams(t, 0, 0)
	run, view, conn, _ := newRefreshTestRun(t)
	defer view.close()

	view.selectedObject, view.selectedOwner = forkNPCID(2), 1
	now := time.Unix(15000, 0)
	outcome, err := view.applySelectedSkillDamage(999999, now)
	frame, id, killed := outcome.frame, outcome.id, outcome.killed
	if err != nil || !killed {
		t.Fatalf("kill settlement err=%v killed=%t", err, killed)
	}
	view.relayNPCKillFrame(id, frame, "tester")
	if _, err := view.objectRequest(clientObjectRequest{Sequence: 8, ObjectID: forkNPCID(2), OwnerID: 1}); err != nil {
		t.Fatalf("open drop box on corpse: %v", err)
	}
	base := len(conn.Frames())
	handled, err := handleCloseDropBoxCustom(view, clientCustomMessage{Values: []clientCustomValue{
		{Type: 2, Int32: clientCustomCloseDropBox},
	}})
	if err != nil || !handled {
		t.Fatalf("close handled=%t err=%v", handled, err)
	}
	closeFrames := frameOpcodes(conn.Frames()[base:])
	if !closeFrames.has(0x16) || !closeFrames.has(0x1E) {
		t.Fatalf("close frames=%v", closeFrames)
	}
	if view.openDropBox != nil {
		t.Fatal("close must clear the open state")
	}
	if len(run.lootBoxes) != 1 {
		t.Fatalf("close must keep the loot on the corpse, got %d", len(run.lootBoxes))
	}
	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2))); !exists {
		t.Fatal("close must keep the corpse registered")
	}
	// 再次打开：行仍在（未拾取）。
	if _, err := view.objectRequest(clientObjectRequest{Sequence: 9, ObjectID: forkNPCID(2), OwnerID: 1}); err != nil {
		t.Fatalf("reopen drop box: %v", err)
	}
	if view.openDropBox == nil || len(view.openDropBox.slots) == 0 {
		t.Fatalf("reopen state=%+v", view.openDropBox)
	}
}

// TestLootBoxExpiresAfterTTL 挂尸掉落超 lootBoxTTL（测试值 60s；生产默认
// 300s，官方 DestroyTime2=60s）直接清尸（0x0E + 注销 + removed 放行
// 重生）。改小参数对在场掉落即刻生效（结算按生成时刻+当前参数推算）；
// 尸体 TTL 先到不掉落未超时不清尸。
func TestLootBoxExpiresAfterTTL(t *testing.T) {
	pinRefreshParams(t, 0, 2*time.Second) // 每怪模式 + 尸体 TTL 2s
	pinLootBoxTTL(t, 60*time.Second)
	run, view, conn, _ := newRefreshTestRun(t)
	defer view.close()

	view.selectedObject, view.selectedOwner = forkNPCID(2), 1
	now := time.Unix(16000, 0)
	outcome, err := view.applySelectedSkillDamage(999999, now)
	frame, id, killed := outcome.frame, outcome.id, outcome.killed
	if err != nil || !killed {
		t.Fatalf("kill settlement err=%v killed=%t", err, killed)
	}
	view.relayNPCKillFrame(id, frame, "tester")

	// 尸体 TTL(2s) 已过但掉落 TTL(60s) 未到：尸体不清除。
	run.refreshTick(now.Add(10 * time.Second))
	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2))); !exists {
		t.Fatal("corpse must survive its own TTL while loot is pending")
	}
	if rec, ok := run.deadNPCs[forkNPCID(2)]; !ok || rec.removed {
		t.Fatalf("corpse must not be removed while loot is pending: %+v ok=%t", rec, ok)
	}
	// 参数即时生效：掉落生成 60s 后把 TTL 调小到 30s，下一次结算即清尸。
	setLootBoxTTL(30 * time.Second)
	run.refreshTick(now.Add(61 * time.Second))
	if len(run.lootBoxes) != 0 {
		t.Fatalf("loot must expire after TTL, got %d", len(run.lootBoxes))
	}
	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2))); exists {
		t.Fatal("expired corpse must be unregistered")
	}
	if rec, ok := run.deadNPCs[forkNPCID(2)]; !ok || !rec.removed {
		t.Fatalf("expired corpse must be marked removed for respawn: %+v ok=%t", rec, ok)
	}
	opcodes := frameOpcodes(conn.Frames())
	if !opcodes.has(0x0E) {
		t.Fatalf("expiry must emit 0x0E, frames=%v", opcodes)
	}
}

// TestLootBoxZeroTTLKeepsBox 0=永不过期：尸体与掉落跨超长时刻仍在场
// （只能被拾空移除），重生全程被推迟。
func TestLootBoxZeroTTLKeepsBox(t *testing.T) {
	pinRefreshParams(t, 0, 0)
	pinLootBoxTTL(t, 0)
	run, view, _, _ := newRefreshTestRun(t)
	defer view.close()

	view.selectedObject, view.selectedOwner = forkNPCID(2), 1
	now := time.Unix(16000, 0)
	outcome, err := view.applySelectedSkillDamage(999999, now)
	if err != nil || !outcome.killed {
		t.Fatalf("kill settlement err=%v killed=%t", err, outcome.killed)
	}
	view.relayNPCKillFrame(outcome.id, outcome.frame, "tester")
	if len(run.lootBoxes) != 1 {
		t.Fatalf("loot must attach on kill, got %d", len(run.lootBoxes))
	}

	run.refreshTick(now.Add(24 * time.Hour))
	if len(run.lootBoxes) != 1 {
		t.Fatalf("loot must survive with ttl=0, got %d", len(run.lootBoxes))
	}
	if vitals, _ := run.combat.Vitals(worldcore.EntityID(forkNPCID(2))); vitals.HP != 0 {
		t.Fatalf("corpse must stay dead while loot is unclaimed: %+v", vitals)
	}
	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2))); !exists {
		t.Fatal("corpse must stay registered with ttl=0 loot")
	}
}

// TestEventNpcNeverEntersRefreshCycle 任务怪护栏：EventNpc 不注册，
// 也就永不进入死亡登记/刷新循环。
func TestEventNpcNeverEntersRefreshCycle(t *testing.T) {
	pinRefreshParams(t, 5*time.Second, 2*time.Second)
	conn := &captureMessageConnection{}
	view := newSceneLifecycle("test", conn)
	view.radius = 50
	event := dropTestAttackNPC("EventNpc_quest_001")
	event.resolved.ScriptClass = "EventNpc"
	catalog := []npcSpawn{dropTestAttackNPC("mon_a"), event}
	run := newSceneRun(sceneKey{config: "scene", resource: "resource"}, nil)
	if err := run.populate(catalog); err != nil {
		t.Fatal(err)
	}
	view.run = run
	view.attachMember(newPlayerActor("tester", 0), role.Position{X: 1}, roleVisual{})
	run.activateMember(view.wireID)
	defer view.close()

	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(3))); exists {
		t.Fatal("EventNpc must never be registered")
	}
	if len(run.deadNPCs) != 0 || len(run.lootBoxes) != 0 {
		t.Fatalf("EventNpc leaked into refresh domain: dead=%d boxes=%d", len(run.deadNPCs), len(run.lootBoxes))
	}
}

// pinAIEditorChests 测试内注入宝箱怪判定表并在结束后还原（包级全局，
// Once 已耗尽时直接替换 map，读侧按 map 现值查）。
func pinAIEditorChests(t *testing.T, chests map[string][]string) {
	t.Helper()
	previous := aiEditorChests
	aiEditorChestsOnce.Do(func() {})
	aiEditorChests = chests
	t.Cleanup(func() { aiEditorChests = previous })
}

// TestCorpseLootDefersRespawn 尸体带未拾掉落时重生被推迟：批量扫描点与
// 每怪 RespawnAt 到点都不复活；拾空清尸（removed）后按重注册路径重生。
func TestCorpseLootDefersRespawn(t *testing.T) {
	pinAIEditorChests(t, map[string][]string{})
	pinRefreshParams(t, 5*time.Second, 0) // 批量模式
	pinLootBoxTTL(t, time.Hour)
	run, view, conn, _ := newRefreshTestRun(t)
	defer view.close()

	now := time.Unix(17000, 0)
	run.refreshTick(now) // 落定扫描基线
	view.selectedObject, view.selectedOwner = forkNPCID(2), 1
	outcome, err := view.applySelectedSkillDamage(999999, now)
	if err != nil || !outcome.killed {
		t.Fatalf("kill settlement err=%v killed=%t", err, outcome.killed)
	}
	view.relayNPCKillFrame(outcome.id, outcome.frame, "tester")

	// 批量扫描点已过：尸体带掉落不复活。
	run.refreshTick(now.Add(6 * time.Second))
	if vitals, _ := run.combat.Vitals(worldcore.EntityID(forkNPCID(2))); vitals.HP != 0 {
		t.Fatalf("corpse must stay dead while loot is pending: %+v", vitals)
	}
	if _, busy := run.deadNPCs[forkNPCID(2)]; !busy {
		t.Fatal("dead record dropped while loot is pending")
	}

	// 每怪模式同理：RespawnAt 到点 simulateLocked 也跳过复活。
	setNpcRespawnBatchInterval(0)
	if brain, has := run.combat.Brain(worldcore.EntityID(forkNPCID(2))); has {
		brain.RespawnAt = now.Add(time.Second)
		run.combat.WriteBrain(worldcore.EntityID(forkNPCID(2)), brain)
	}
	run.combatTick(now.Add(7 * time.Second))
	if vitals, _ := run.combat.Vitals(worldcore.EntityID(forkNPCID(2))); vitals.HP != 0 {
		t.Fatalf("per-NPC respawn must defer while loot is pending: %+v", vitals)
	}

	// 拾空清尸 → removed；下一 tick 按目录行重注册（0x0D+0x1F）。
	run.mu.Lock()
	run.removeLootBoxLocked(nil, forkNPCID(2), 1)
	run.mu.Unlock()
	if rec, ok := run.deadNPCs[forkNPCID(2)]; !ok || !rec.removed {
		t.Fatalf("loot cleanup must mark corpse removed: %+v ok=%t", rec, ok)
	}
	respawnBase := len(conn.Frames())
	// 击杀时每怪重生已排 respawnAt=now+15s，越过后重注册成行。
	run.refreshTick(now.Add(20 * time.Second))
	if _, exists := run.npcMeta(worldcore.EntityID(forkNPCID(2))); !exists {
		t.Fatal("corpse must re-register after loot cleanup")
	}
	if revived, _ := run.combat.Vitals(worldcore.EntityID(forkNPCID(2))); revived.HP != revived.MaxHP {
		t.Fatalf("re-registered NPC must be full HP: %+v", revived)
	}
	opcodes := frameOpcodes(conn.Frames()[respawnBase:])
	if !opcodes.has(0x0D) || !opcodes.has(0x1F) {
		t.Fatalf("respawn frames=%v", opcodes)
	}
}

// TestChestMonsterSpawnsBoxEntity 宝箱怪（AIEditorPackID 命中死亡生箱
// 事件）：死后生成独立宝箱实体（0x0D 物化），尸体本身不挂掉落。
func TestChestMonsterSpawnsBoxEntity(t *testing.T) {
	pinAIEditorChests(t, map[string][]string{"700": {"BossBox021001A"}})
	pinRefreshParams(t, 0, 0)
	pinLootBoxTTL(t, time.Minute)
	run, view, conn, _ := newRefreshTestRun(t)
	defer view.close()

	monster := dropTestAttackNPC("mon_chest")
	monster.resolved.Extensions["template.AIEditorPackID"] = "700"
	run.catalog[0] = monster

	view.selectedObject, view.selectedOwner = forkNPCID(2), 1
	now := time.Unix(18000, 0)
	outcome, err := view.applySelectedSkillDamage(999999, now)
	frame, id, killed := outcome.frame, outcome.id, outcome.killed
	if err != nil || !killed {
		t.Fatalf("kill settlement err=%v killed=%t", err, killed)
	}
	if _, hasCorpseLoot := run.lootBoxes[forkNPCID(2)]; hasCorpseLoot {
		t.Fatal("chest monster corpse must not carry loot itself")
	}
	var chestID uint32
	var chest *lootBoxRuntime
	for chestID, chest = range run.lootBoxes {
	}
	if chestID == forkNPCID(2) || chestID == 0 || !chest.standalone {
		t.Fatalf("chest monster must spawn a standalone entity, id=%d box=%+v", chestID, chest)
	}
	view.relayNPCKillFrame(id, frame, "tester")
	opcodes := frameOpcodes(conn.Frames())
	if !opcodes.has(0x0D) || !opcodes.has(0x1F) {
		t.Fatalf("chest entity materialization frames=%v", opcodes)
	}
	if hasCorpsePickFrames(t, conn.Frames(), forkNPCID(2)) {
		t.Fatal("chest monster corpse must NOT emit CanPick frames")
	}
}

// TestLoadAIEditorChests 双 ini 解析：EventGroup 逗号链 → 死亡事件(3)
// + 创建 NPC(25) 的 Para1 宝箱模板；非死亡/非生箱事件忽略。
func TestLoadAIEditorChests(t *testing.T) {
	packs := strings.NewReader(strings.Join([]string{
		"[700]",
		"EventGroup=265,266",
		"Text=田锋宝箱掉落",
		"[701]",
		"EventGroup=267",
	}, "\n"))
	events := strings.NewReader(strings.Join([]string{
		"[265]",
		"EventType=3",
		"OpEvent=25",
		"Para1=BossBox021001A",
		"[266]",
		"EventType=1",
		"OpEvent=25",
		"Para1=buff_stormbox001",
		"[267]",
		"EventType=3",
		"OpEvent=9",
		"Arg1=1354",
	}, "\n"))
	chests := loadAIEditorChests(packs, events)
	if got := chests["700"]; len(got) != 1 || got[0] != "BossBox021001A" {
		t.Fatalf("pack 700 chests=%v", got)
	}
	if got, ok := chests["701"]; ok && len(got) > 0 {
		t.Fatalf("pack 701 must resolve empty, got %v", got)
	}
	pinAIEditorChests(t, chests)
	probe := dropTestAttackNPC("mon_probe")
	probe.resolved.Extensions["template.AIEditorPackID"] = "700"
	if got := chestSpawnConfigForNPC(probe); len(got) != 1 || got[0] != "BossBox021001A" {
		t.Fatalf("Extensions lookup must hit the injected map, got %v", got)
	}
}

// frameOpcodes 收集帧首字节集合，断言用。
type frameOpcodeSet map[byte]int

func frameOpcodes(frames [][]byte) frameOpcodeSet {
	set := make(frameOpcodeSet, len(frames))
	for _, frame := range frames {
		if len(frame) == 0 {
			continue
		}
		set[frame[0]]++
	}
	return set
}

func (s frameOpcodeSet) has(opcode byte) bool { return s[opcode] > 0 }
