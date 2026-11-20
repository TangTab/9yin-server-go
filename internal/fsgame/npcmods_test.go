package fsgame

import (
	"path/filepath"
	"testing"
)

// assemble8 装配八批四十五模块 + NPC 类 schema。
func assemble8(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))
	cs := mustRead(t, filepath.Join(td, "struct", "scene", "clone_scene.xml"))
	sc := mustRead(t, filepath.Join(td, "struct", "scene", "scene.xml"))
	cn := mustRead(t, filepath.Join(td, "struct", "npc", "CommonNpc.xml"))
	vn := mustRead(t, filepath.Join(td, "struct", "npc", "VisibleNpc.xml"))

	k := NewKernel()
	k.ResourcePath = td
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	for cls, data := range map[string][]byte{"player": px, "CloneScene": cs,
		"scene": sc, "CommonNpc": cn, "VisibleNpc": vn} {
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
		RegisterSeventhBatch, RegisterEighthBatch,
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

// TestEighthBatchAssemble 八期装配序 + NPC 资源/召唤真档。
func TestEighthBatchAssemble(t *testing.T) {
	k, mods := assemble8(t)
	want := []string{"GivingNpcModule", "FunctionNpcModule", "SummonNpcModule",
		"CommonNpcModule", "VisibleNpcModule", "NpcResourceModule"}
	var got []string
	for _, m := range mods {
		switch m.Name() {
		case "GivingNpcModule", "FunctionNpcModule", "SummonNpcModule",
			"CommonNpcModule", "VisibleNpcModule", "NpcResourceModule":
			got = append(got, m.Name())
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	sm := module2[*SummonNpcModule](t, k, "SummonNpcModule")
	if len(sm.Summons) == 0 {
		t.Fatalf("EventSummon.xml 真档未装载")
	}
	if segs := sm.Summons["6"]; len(segs) == 0 || segs[0].NpcID != "summon_qhm005_0003" {
		t.Fatalf("召唤段解析异常: %v", sm.Summons["6"])
	}
	nrm := module2[*NpcResourceModule](t, k, "NpcResourceModule")
	if nrm.Loaded() == 0 {
		t.Fatal("ini/Npc 树预载计数为零")
	}
	// 九阴真档抽查（tools/npcgen 生成）：成都马六，script=CommonNpc。
	if class := nrm.ClassOf("FuncNpc00601"); class != "CommonNpc" {
		t.Fatalf("FuncNpc00601 登记类 %q，期望 CommonNpc", class)
	}
	if id := nrm.Spawn(k, "FuncNpc00601"); !id.Valid() {
		t.Fatal("九阴真档 NPC 物化失败")
	}
}

// TestNpcLifecycle NPC 闭环：资源登记 → 召唤物化（EventSummon 真档）→
// 交互枢纽（CommonNpc OnSelect 多播）→ 视野挂载。
func TestNpcLifecycle(t *testing.T) {
	k, _ := assemble8(t)
	nrm := module2[*NpcResourceModule](t, k, "NpcResourceModule")
	sm := module2[*SummonNpcModule](t, k, "SummonNpcModule")
	cm := module2[*CommonNpcModule](t, k, "CommonNpcModule")
	vm := module2[*VisibleNpcModule](t, k, "VisibleNpcModule")

	// 资源登记 + 物化（ConfigID 写入）。
	nrm.RegisterNpc("summon_qhm005_0003", "CommonNpc")
	id := nrm.Spawn(k, "summon_qhm005_0003")
	if !id.Valid() {
		t.Fatal("NPC 物化失败")
	}
	if k.GetGameObj(id).GetString("ConfigID") != "summon_qhm005_0003" {
		t.Fatal("ConfigID 未写")
	}
	// 召唤真档：事件 6 → 段展开 ×count。
	if n := sm.Summon(k, "6"); n == 0 {
		t.Fatal("真档召唤未物化任何 NPC")
	}
	// 交互枢纽：CommonNpc OnSelect 多播（Quest/Shop/Strengthen 各域同收）。
	k.RunEvent("CommonNpc", "OnSelect", id, NullPersistID, VarList{"npc_master"})
	if len(cm.Selects()) != 1 || cm.Selects()[0] != "npc_master" {
		t.Fatalf("交互流水 %v", cm.Selects())
	}
	// 视野：玩家入场挂载。
	p, _ := k.CreateObject("player")
	k.RunEvent("player", "OnEntry", p, NullPersistID, nil)
	if len(vm.Viewers()) == 0 {
		t.Fatal("视野挂载未流水")
	}
}

// TestGivingNpcLifecycle 献祭 NPC 闭环：弹出 → 采集开始 → 完成（HP 归零）→
// 拾取（隐身）→ HB 重生。
func TestGivingNpcLifecycle(t *testing.T) {
	k, _ := assemble8(t)
	gm := module2[*GivingNpcModule](t, k, "GivingNpcModule")

	npc, _ := k.CreateObject("player") // 占位 GivingNpc（记录面模块侧承载）
	k.RunEvent("GivingNpc", "OnLoad", npc, NullPersistID, nil)
	k.RunEvent("GivingNpc", "OnSpring", npc, NullPersistID, nil)
	p, _ := k.CreateObject("player")
	// 采集：开始 → 完成（603 GivingNpc 面）。
	k.DispatchIntCommand("player", 600, p, NullPersistID, nil)
	k.DispatchIntCommand("GivingNpc", 603, npc, NullPersistID, nil)
	has := func(ev string) bool {
		for _, e := range gm.Events() {
			if e == ev {
				return true
			}
		}
		return false
	}
	if !has("skill_begin") || !has("skill_finish") {
		t.Fatalf("采集流水 %v", gm.Events())
	}
	// 拾取前无效（未完成态拒）——先验证未完成拒：新 NPC 直接 pickup。
	npc2, _ := k.CreateObject("player")
	k.RunEvent("GivingNpc", "OnLoad", npc2, NullPersistID, nil)
	k.DispatchIntCommand("GivingNpc", 852, npc2, NullPersistID, nil)
	if has("pickup") {
		t.Fatal("未完成目标物可拾取")
	}
	// 完成 → 拾取 → 隐身 → HB 重生。
	k.DispatchIntCommand("GivingNpc", 603, npc, NullPersistID, nil)
	k.DispatchIntCommand("GivingNpc", 852, npc, NullPersistID, nil)
	if !has("pickup") {
		t.Fatal("完成态拾取未流水")
	}
	k.Bus().TickHeartBeats(k, 1)
	if !has("reborn") {
		t.Fatal("HB_ReBorn 未重生")
	}
	// 采集中断面。
	k.DispatchIntCommand("player", 602, p, NullPersistID, nil)
	if !has("skill_break") {
		t.Fatal("中断未流水")
	}
}
