package main

import (
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// TestDungeonEntryPosition 入口落点推导：战斗 NPC（attacknpc/bossnpc）
// 质心 + X/Z 各 4m 偏移；非战斗与不合格（EventNpc）不计入；无战斗 NPC
// 回落第一个合格出生点；空目录给零位。
func TestDungeonEntryPosition(t *testing.T) {
	attacker := testNPCSpawn()
	attacker.resolved.ScriptClass = "AttackNpc"
	attacker.x, attacker.y, attacker.z = 0, 10, 0
	boss := testNPCSpawn()
	boss.resolved.ScriptClass = "BossNpc"
	boss.x, boss.y, boss.z = 10, 20, 10
	civilian := testNPCSpawn() // CommonNpc：非战斗，不计入质心
	civilian.x, civilian.y, civilian.z = -100, -100, -100
	event := testNPCSpawn()
	event.resolved.ScriptClass = "EventNpc"
	event.x, event.y, event.z = 999, 999, 999

	entry := dungeonEntryPosition([]npcSpawn{attacker, boss, civilian, event})
	if entry.X != 9 || entry.Y != 15 || entry.Z != 9 {
		t.Fatalf("centroid entry=(%v,%v,%v), want (9,15,9)", entry.X, entry.Y, entry.Z)
	}

	onlyCivilian := dungeonEntryPosition([]npcSpawn{civilian})
	if onlyCivilian.X != -100 || onlyCivilian.Y != -100 {
		t.Fatalf("fallback entry=(%v,%v,%v), want first eligible spawn", onlyCivilian.X, onlyCivilian.Y, onlyCivilian.Z)
	}

	if empty := dungeonEntryPosition(nil); empty != (role.Position{}) {
		t.Fatalf("empty catalog entry=%v, want zero position", empty)
	}
}

// TestCloneSceneOf 副本 config → 场景标识：副本图按完整配置基名取
// 地形名（clone003），城市前缀图仍取首段（city05_ChengDu → city05）。
func TestCloneSceneOf(t *testing.T) {
	scene, err := cloneSceneOf(`ini\scene\clone003`)
	if err != nil {
		t.Fatal(err)
	}
	if scene.Config != `ini\scene\clone003` || scene.Resource != "clone003" {
		t.Fatalf("clone scene=%+v", scene)
	}
	city, err := cloneSceneOf(`ini\scene\city05_ChengDu`)
	if err != nil {
		t.Fatal(err)
	}
	if city.Resource != "city05" {
		t.Fatalf("city scene=%+v", city)
	}
}

// TestDungeonManagerInstanceStateMachine 一人一实例状态机：create/open/
// take 的归属转移、重复 create 拒绝、attachRun 置 managed/noRespawn、
// dropOnDisconnect 幂等清场。
func TestDungeonManagerInstanceStateMachine(t *testing.T) {
	manager := &dungeonManager{byOwner: map[role.RoleID]*dungeonInstance{}}
	owner := role.RoleID(7)
	returnTo := role.Location{Scene: role.Scene{Config: `ini\scene\city05_ChengDu`, Resource: "city05"}}

	instance, err := manager.create(owner, returnTo.Scene, 2, returnTo, role.Position{X: 1})
	if err != nil {
		t.Fatal(err)
	}
	if instance.id == "" || instance.key.instance != instance.id || instance.key.config != returnTo.Scene.Config {
		t.Fatalf("instance key misbuilt: %+v", instance.key)
	}
	if manager.open(owner) != instance {
		t.Fatal("open should return the registered instance")
	}
	if _, err := manager.create(owner, returnTo.Scene, 1, returnTo, role.Position{}); err == nil {
		t.Fatal("duplicate create should be rejected")
	}

	run := newSceneRun(sceneKey{config: "cfg", resource: "res"}, nil)
	if attached := manager.attachRun(owner, run); attached != instance {
		t.Fatal("attachRun should return the instance")
	}
	run.mu.Lock()
	managed, noRespawn := run.managed, run.noRespawn
	run.mu.Unlock()
	if !managed || !noRespawn {
		t.Fatalf("instance run flags managed=%v noRespawn=%v, want both true", managed, noRespawn)
	}

	if taken := manager.take(owner); taken != instance {
		t.Fatal("take should return the instance")
	}
	if manager.open(owner) != nil {
		t.Fatal("taken instance should no longer be open")
	}
	manager.shutdownInstance(instance) // run 未 populate，直接关闭
	run.mu.Lock()
	closed := run.closed
	run.mu.Unlock()
	if !closed {
		t.Fatal("shutdownInstance should close the run")
	}
	manager.shutdownInstance(instance) // 幂等
	manager.dropOnDisconnect(owner)    // 无登记：no-op
}

// TestManagedRunSurvivesLastMemberLeave managed 运行时（副本实例）末成员
// 离开不自动销毁；显式 shutdownInstance 才关闭并摘出注册表。
func TestManagedRunSurvivesLastMemberLeave(t *testing.T) {
	view := newSceneLifecycle("test", discardMessageConnection{})
	view.radius = 50
	catalog := []npcSpawn{dropTestAttackNPC("mon_a")}
	run := newSceneRun(sceneKey{config: "dungeon-scene", resource: "dungeon-res", instance: "dn-test-1"}, nil)
	if err := run.populate(catalog); err != nil {
		t.Fatal(err)
	}
	run.mu.Lock()
	run.managed = true
	run.noRespawn = true
	run.mu.Unlock()
	view.run = run
	player := newPlayerActor("tester", 0)
	view.attachMember(player, role.Position{X: 1}, roleVisual{})

	view.detachMember()
	run.mu.Lock()
	closed, members := run.closed, len(run.members)
	run.mu.Unlock()
	if closed || members != 0 {
		t.Fatalf("managed run closed=%v members=%d after last leave, want alive empty run", closed, members)
	}

	manager := &dungeonManager{byOwner: map[role.RoleID]*dungeonInstance{}}
	instance := &dungeonInstance{id: "dn-test-1", key: run.key, ownerRole: 1, run: run}
	manager.shutdownInstance(instance)
	run.mu.Lock()
	closed = run.closed
	run.mu.Unlock()
	if !closed {
		t.Fatal("explicit shutdown should close the managed run")
	}
}

// TestBeginKeyInstanceIsolation 实例键隔离：同 (config, resource) 不同
// instance 各得独立运行时；空 instance 走共享键，同一键复用同一运行时。
func TestBeginKeyInstanceIsolation(t *testing.T) {
	sharedA := newSharedSceneLifecycle("a", discardMessageConnection{}, nil)
	sharedB := newSharedSceneLifecycle("b", discardMessageConnection{}, nil)
	worldKey := sceneKey{config: "isolate-scene", resource: "isolate-res"}
	if err := sharedA.beginKey(worldKey, nil); err != nil {
		t.Fatal(err)
	}
	if err := sharedB.beginKey(worldKey, nil); err != nil {
		t.Fatal(err)
	}
	if sharedA.run != sharedB.run {
		t.Fatal("shared key should reuse the same run")
	}
	instanceA := newSharedSceneLifecycle("ia", discardMessageConnection{}, nil)
	instanceB := newSharedSceneLifecycle("ib", discardMessageConnection{}, nil)
	if err := instanceA.beginKey(sceneKey{config: worldKey.config, resource: worldKey.resource, instance: "dn-1"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := instanceB.beginKey(sceneKey{config: worldKey.config, resource: worldKey.resource, instance: "dn-2"}, nil); err != nil {
		t.Fatal(err)
	}
	if instanceA.run == instanceB.run || instanceA.run == sharedA.run || instanceB.run == sharedA.run {
		t.Fatal("instance keys must yield independent runs")
	}
	// 清场：共享键与实例键运行时各一次 shutdown（beginKey 经注册表
	// 创建的运行时不会随末成员离开自动关闭时也须可安全关闭）。
	for _, view := range []*sceneLifecycle{sharedA, sharedB, instanceA, instanceB} {
		view.close()
	}
}

// TestNoRespawnSkipsRefreshRevive 副本怪杀完不复活：noRespawn 运行时上
// 批量扫描不复活尸体（对照组：普通运行时批量复活）。攻击 NPC 刻意
// 不带 DropID——挂尸掉落会另行挡住批量复活，干扰对照。
func TestNoRespawnSkipsRefreshRevive(t *testing.T) {
	pinRefreshParams(t, 30*time.Second, 0) // 批量模式 + 尸体常驻

	newRun := func(noRespawn bool) *sceneRun {
		view := newSceneLifecycle("test", &captureMessageConnection{})
		view.radius = 50
		npc := testNPCSpawn()
		npc.resolved.ScriptClass = "AttackNpc"
		npc.resolved.ConfigID = "mon_plain"
		catalog := []npcSpawn{npc}
		run := newSceneRun(sceneKey{config: "scene", resource: "resource"}, nil)
		if err := run.populate(catalog); err != nil {
			t.Fatal(err)
		}
		run.mu.Lock()
		run.noRespawn = noRespawn
		run.mu.Unlock()
		view.run = run
		view.attachMember(newPlayerActor("tester", 0), role.Position{X: 1}, roleVisual{})
		run.activateMember(view.wireID)
		view.refreshVisibility(catalog, role.Position{X: 1})
		return run
	}

	dungeonRun := newRun(true)
	id := forkNPCID(2)
	killOnly(dungeonRun, id, deadNPCRecord{spawnIndex: 0, diedAt: time.Now().Add(-time.Minute)})
	dungeonRun.refreshTick(time.Now())
	if vitals, ok := dungeonRun.combat.Vitals(worldcore.EntityID(id)); ok && vitals.HP > 0 {
		t.Fatalf("noRespawn run revived NPC %d (HP=%d)", id, vitals.HP)
	}

	worldRun := newRun(false)
	killOnly(worldRun, id, deadNPCRecord{spawnIndex: 0, diedAt: time.Now().Add(-time.Minute)})
	worldRun.refreshTick(time.Now())
	if vitals, ok := worldRun.combat.Vitals(worldcore.EntityID(id)); !ok || vitals.HP <= 0 {
		t.Fatalf("normal run should batch-revive NPC %d", id)
	}
}
