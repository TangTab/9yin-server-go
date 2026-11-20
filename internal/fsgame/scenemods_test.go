package fsgame

import (
	"path/filepath"
	"testing"
)

// assemble7 装配七批三十九模块 + CloneScene/scene schema。
func assemble7(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))
	cs := mustRead(t, filepath.Join(td, "struct", "scene", "clone_scene.xml"))
	sc := mustRead(t, filepath.Join(td, "struct", "scene", "scene.xml"))

	k := NewKernel()
	k.ResourcePath = td
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	for cls, data := range map[string][]byte{"player": px, "CloneScene": cs, "scene": sc} {
		if err := k.LoadStruct(cls, data); err != nil {
			t.Fatalf("LoadStruct %s: %v", cls, err)
		}
	}
	if err := k.LoadStruct("player", ex); err != nil {
		t.Fatalf("LoadStruct player(exp): %v", err)
	}
	var r Registry
	for _, reg := range []func(*Registry) error{
		RegisterFirstBatch, RegisterSecondBatch, RegisterThirdBatch,
		RegisterFourthBatch, RegisterFifthBatch, RegisterSixthBatch,
		RegisterSeventhBatch,
	} {
		if err := reg(&r); err != nil {
			t.Fatal(err)
		}
	}
	mods, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return k, mods
}

// TestSeventhBatchAssemble 七期装配序 + 评分/传送真档。
func TestSeventhBatchAssemble(t *testing.T) {
	k, mods := assemble7(t)
	want := []string{"WarFog", "MapModule", "AreaObjModule", "MapMagModule", "CloneSceneModule"}
	var got []string
	for _, m := range mods {
		switch m.Name() {
		case "WarFog", "MapModule", "AreaObjModule", "MapMagModule", "CloneSceneModule":
			got = append(got, m.Name())
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	cm := module2[*CloneSceneModule](t, k, "CloneSceneModule")
	if g := cm.Grades["101_0"]; g == nil || atoi32(g["TargetPassTime"]) != 10 || atof64(g["ScoreFactor"]) != 0.1 {
		t.Fatalf("CloneSceneGrade 真档异常: %v", cm.Grades["101_0"])
	}
	mm := module2[*MapMagModule](t, k, "MapMagModule")
	if tr := mm.Transports["npc_zqm003_0126"]; tr == nil || atoi32(tr["Scene"]) != 7 {
		t.Fatalf("transport 真档异常: %v", mm.Transports["npc_zqm003_0126"])
	}
}

// TestCloneSceneLifecycle 副本闭环：建副（属性链）→ 进度推进 → Boss 击杀 →
// 死亡计数 → 通关评分（CloneSceneGrade 真档）→ 关闭。
func TestCloneSceneLifecycle(t *testing.T) {
	k, _ := assemble7(t)
	cm := module2[*CloneSceneModule](t, k, "CloneSceneModule")
	leader, _ := k.CreateObject("player")
	_ = k.SetString(leader, "name", "团长")

	// 建副（原型 101 / 难度 0 / 队伍 5 / 拥有者类型 2=小队）。
	_ = k.SetInt(leader, "TeamID", 5)
	uid, clone := cm.CreateClone(k, 101, 0, leader, 2)
	if uid == "" || !clone.Valid() {
		t.Fatal("建副失败")
	}
	co := k.GetGameObj(clone)
	if co.GetString("CloneUniqueID") != uid || co.GetInt("CloneOwnerID") != 5 {
		t.Fatalf("副本属性链 %+v", co)
	}
	if co.GetString("CloneOwnerName") != "团长" {
		t.Fatal("CloneOwnerName 未写")
	}
	// 进度推进（临界联动注册面）。
	cm.SetProgress(k, clone, 50)
	if co.GetInt("CloneProgress") != 50 {
		t.Fatal("进度未写")
	}
	// Boss 击杀 + 玩家死亡（616 多播收令）。
	k.DispatchIntCommand("AttackNpc", 616, leader, leader, VarList{"boss"})
	k.DispatchIntCommand("player", 616, leader, NullPersistID, VarList{})
	if co.GetInt("BossBeKilledCount") != 1 || co.GetInt("PlayerDeadCount") != 1 {
		t.Fatalf("击杀/死亡计数 %d/%d", co.GetInt("BossBeKilledCount"), co.GetInt("PlayerDeadCount"))
	}
	// 通关评分：Grade 101_0（Target=10/Normal=40/DeadDecrease=0）。
	// passTime=8 ≤ Target → 100 分 S（死亡扣 0）。
	if grade := cm.PassClone(k, clone, 101, 8); grade != "S" {
		t.Fatalf("评分 %q 期望 S", grade)
	}
	if co.GetInt("CloneProgress") != 100 {
		t.Fatal("通关未拉满进度")
	}
	// 慢速通关：passTime=50 > Normal=40 → 60 分 B。
	uid2, clone2 := cm.CreateClone(k, 101, 0, leader, 2)
	if grade := cm.PassClone(k, clone2, 101, 50); grade != "B" {
		t.Fatalf("慢速评分 %q 期望 B", grade)
	}
	_ = uid2
	// 关闭。
	cm.CloseClone(k, uid)
	if cm.CloneOf(uid).Valid() || k.GetGameObj(clone) != nil {
		t.Fatal("关闭未销毁副本")
	}
}

// TestMapTransAndExplore 传送 + 探索 + 区域联动 + 战雾。
func TestMapTransAndExplore(t *testing.T) {
	k, _ := assemble7(t)
	mm := module2[*MapMagModule](t, k, "MapMagModule")
	mp := module2[*MapModule](t, k, "MapModule")
	ao := module2[*AreaObjModule](t, k, "AreaObjModule")
	wf := module2[*WarFog](t, k, "WarFog")

	p, _ := k.CreateObject("player")
	k.RunEvent("player", "OnEntry", p, NullPersistID, nil) // MapModule 入场自动探索

	// 战雾默认关：入场不揭示。
	if len(wf.Reveals()) != 0 {
		t.Fatal("战雾未开即揭示")
	}
	wf.EnableWarFog = true
	k.RunEvent("player", "OnEntry", p, NullPersistID, nil)
	if len(wf.Reveals()) != 1 {
		t.Fatal("战雾开启后未揭示")
	}

	// 传送：死亡拒（sys_map_0007）→ 正常传送（OnMapChanged 联动探索+区域）。
	_ = k.SetInt(p, "Dead", 1)
	if mm.Trans(k, p, "npc_zqm003_0126") {
		t.Fatal("死亡可传送")
	}
	if !hasTpl(mm.SysMsgs(), "sys_map_0007") {
		t.Fatal("死亡传送未通知 sys_map_0007")
	}
	_ = k.SetInt(p, "Dead", 0)
	if !mm.Trans(k, p, "npc_zqm003_0126") {
		t.Fatal("传送失败")
	}
	if mm.SceneOf(p) != 7 {
		t.Fatalf("传送后场景 %d 期望 7", mm.SceneOf(p))
	}
	// OnMapChanged → MapModule 自动探索目标场景 + AreaObj 联动。
	if !mp.IsExplored(p, 7) {
		t.Fatal("传送后未自动探索目标场景")
	}
	found := false
	for _, e := range ao.Enters() {
		if e == "map:1" { // 玩家对象 id=1（首个创建）
			found = true
		}
	}
	if !found {
		t.Fatal("AreaObj 未联动 OnMapChanged")
	}
	// 探索 Lua 面。
	mp.NxClearExplored(p)
	if mp.IsExplored(p, 7) {
		t.Fatal("清探索未生效")
	}
	mp.NxSetExplored(p, 7)
	if !mp.IsExplored(p, 7) {
		t.Fatal("标记探索未生效")
	}
	// cust 732 路由。
	_ = k.SetInt(p, "Dead", 0)
	k.DispatchCommand("player", 732, p, NullPersistID, VarList{"npc_zqm003_0126"})
	if mm.SceneOf(p) != 7 {
		t.Fatal("cust 732 传送未路由")
	}
}
