package fsgame

import (
	"path/filepath"
	"strings"
	"testing"
)

// assemble10 装配十批五十四模块 + AI schema（CommonNpc 类合并
// CommonNpc/ai_npc_data/ai_motion 三档——CommonNpc→AttackNpc 继承链
// 上溯收 OnSpring/OnEntry/OnMotion 注册面）。
func assemble10(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))
	cn := mustRead(t, filepath.Join(td, "struct", "npc", "CommonNpc.xml"))
	an := mustRead(t, filepath.Join(td, "struct", "npc", "AttackNpc.xml"))
	ai := mustRead(t, filepath.Join(td, "struct", "npc_property", "ai_npc_data.xml"))
	am := mustRead(t, filepath.Join(td, "struct", "npc_property", "ai_motion.xml"))

	k := NewKernel()
	k.ResourcePath = td
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	for cls, data := range map[string][]byte{"player": px, "CommonNpc": cn, "AttackNpc": an} {
		if err := k.LoadStruct(cls, data); err != nil {
			t.Fatalf("LoadStruct %s: %v", cls, err)
		}
	}
	for _, data := range [][]byte{ai, am} {
		if err := k.LoadStruct("CommonNpc", data); err != nil {
			t.Fatalf("LoadStruct CommonNpc(ai): %v", err)
		}
	}
	if err := k.LoadStruct("player", ex); err != nil {
		t.Fatalf("LoadStruct player(exp): %v", err)
	}
	var r Registry
	for _, reg := range []func(*Registry) error{
		RegisterFirstBatch, RegisterSecondBatch, RegisterThirdBatch,
		RegisterFourthBatch, RegisterFifthBatch, RegisterSixthBatch,
		RegisterSeventhBatch, RegisterEighthBatch, RegisterNinthBatch,
		RegisterTenthBatch,
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

// TestTenthBatchAssemble 十期装配序 + 注册面抽查。
func TestTenthBatchAssemble(t *testing.T) {
	k, mods := assemble10(t)
	want := []string{"AICharacterModule", "AISkillControlModule", "AIOnCommandMotion",
		"WarnModule", "HateModule", "AIChatWithOther", "AIMotionModule", "AIModule", "TempStopAI"}
	var got []string
	for _, m := range mods {
		switch m.Name() {
		case "AICharacterModule", "AISkillControlModule", "AIOnCommandMotion",
			"WarnModule", "HateModule", "AIChatWithOther", "AIMotionModule", "AIModule", "TempStopAI":
			got = append(got, m.Name())
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	// 心跳注册面（Warn 双泵 + AIMotion 移动泵）。
	hbs := k.Bus().HeartBeatNames()
	for _, want := range []string{"WarnModule::HB_AIWarn", "WarnModule::HB_AIWarn_SkillNpc", "AIMotionModule::HB_MoveOver"} {
		found := false
		for _, h := range hbs {
			if h == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("心跳 %s 未注册: %v", want, hbs)
		}
	}
	// schema 合并面：AI 属性/记录随类装载。
	npc, err := k.CreateObject("CommonNpc")
	if err != nil {
		t.Fatal(err)
	}
	o := k.GetGameObj(npc)
	if _, ok := o.Class().Prop("WarnType"); !ok {
		t.Fatal("ai_npc_data 属性未合并")
	}
	if _, ok := o.Class().Prop("AIMotioning"); !ok {
		t.Fatal("ai_motion 属性未合并")
	}
	if o.Record("HateList") == nil || o.Record("SpringList") == nil || o.Record("event_talk_rec") == nil {
		t.Fatal("AI 记录未合并")
	}
}

// TestWarnToFightReturn 主动怪全链：OnEntry 装配（Born 锚定）→ 球警戒
//（SpringList 登记；WarnTime=0 即时入战）→ 仇恨种子 + 战斗态 → 技能轮选
// → 追击界外脱战（return + 仇恨清 + 回巢移动）→ AIMotion 步进到点 → stand。
func TestWarnToFightReturn(t *testing.T) {
	k, _ := assemble10(t)
	am := module2[*AIModule](t, k, "AIModule")
	wm := module2[*WarnModule](t, k, "WarnModule")
	hm := module2[*HateModule](t, k, "HateModule")
	mm := module2[*AIMotionModule](t, k, "AIMotionModule")
	sm := module2[*AISkillControlModule](t, k, "AISkillControlModule")

	npc, _ := k.CreateObject("CommonNpc")
	no := k.GetGameObj(npc)
	no.SetPos(12, 0, 0)
	_ = no.Set("WarnType", int32(1))
	_ = no.Set("SphereRadius", float32(10))
	_ = no.Set("WarnTime", int32(0))
	_ = no.Set("ChaseRange", float32(50))
	_ = no.Set("Speed", float32(5))
	_ = no.Set("SkillPackID", "sk1,sk2")
	k.RunEvent("CommonNpc", "OnEntry", npc, NullPersistID, nil)
	acm := module2[*AICharacterModule](t, k, "AICharacterModule")
	if len(acm.Inits()) == 0 || no.GetFloat("BornX") != 12 {
		t.Fatalf("OnEntry 装配未锚定 Born: %v born=%v", acm.Inits(), no.GetFloat("BornX"))
	}
	if no.GetInt("CurState") != AIStateStand {
		t.Fatalf("进场态=%d 期望 stand", no.GetInt("CurState"))
	}

	// 玩家入球（dist=7<10）→ 警戒 → 即时入战。
	p, _ := k.CreateObject("player")
	k.GetGameObj(p).SetPos(5, 0, 0)
	k.Bus().TickHeartBeats(k, 1)
	if r := no.Record("SpringList"); r.Rows() != 1 {
		t.Fatalf("SpringList 应 1 行: %d", r.Rows())
	}
	if no.GetInt("CurState") != AIStateFight {
		t.Fatalf("警戒后态=%d 期望 fight", no.GetInt("CurState"))
	}
	if v, _ := no.Get("AIFirstAttacker").(PersistID); v != p {
		t.Fatalf("AIFirstAttacker=%v 期望玩家", v)
	}
	if r := no.Record("HateList"); r.Rows() != 1 {
		t.Fatalf("仇恨种子缺失: %d", r.Rows())
	}

	// 战斗态技能轮选（sk1→sk2→sk1）。
	if got := sm.PickSkill(k, npc); got != "sk1" {
		t.Fatalf("首轮技能=%q 期望 sk1", got)
	}
	if got := sm.PickSkill(k, npc); got != "sk2" {
		t.Fatalf("次轮技能=%q 期望 sk2", got)
	}

	// 被击加仇恨 → 玩家拉远再击 → 超追击界脱战回巢。
	k.DispatchIntCommand("CommonNpc", 618, npc, NullPersistID, VarList{p, int32(10)})
	if r := no.Record("HateList"); r.Row(0).Int(1) != 11 {
		t.Fatalf("被击后仇恨=%d 期望 11", r.Row(0).Int(1))
	}
	if hm.Target(k, npc) != p {
		t.Fatal("最大仇恨目标应为玩家")
	}
	k.GetGameObj(p).SetPos(100, 0, 0)
	k.DispatchIntCommand("CommonNpc", 618, npc, NullPersistID, VarList{p, int32(5)})
	if no.GetInt("CurState") != AIStateReturn {
		t.Fatalf("超界后态=%d 期望 return", no.GetInt("CurState"))
	}
	if r := no.Record("HateList"); r.Rows() != 0 {
		t.Fatalf("脱战仇恨未清: %d", r.Rows())
	}
	if no.GetInt("AIMotioning") != 1 || no.GetFloat("FindPathX") != 12 {
		t.Fatalf("回巢移动未挂: motioning=%d target=%v", no.GetInt("AIMotioning"), no.GetFloat("FindPathX"))
	}

	// AIMotion 步进：NPC 漂移到 (30,0) → 每拍 5 → 4 拍回到 (12,0) → stand。
	no.SetPos(30, 0, 0)
	for i := 0; i < 3; i++ {
		k.Bus().TickHeartBeats(k, 1)
	}
	x, _, _ := no.Pos()
	if x != 15 { // 30-5*3 中途未达
		t.Fatalf("步进中途 X=%v 期望 15", x)
	}
	k.Bus().TickHeartBeats(k, 1)
	x, _, _ = no.Pos()
	if x != 12 || no.GetInt("AIMotioning") != 0 {
		t.Fatalf("到点 X=%v motioning=%d", x, no.GetInt("AIMotioning"))
	}
	if no.GetInt("CurState") != AIStateStand {
		t.Fatalf("回巢后态=%d 期望 stand", no.GetInt("CurState"))
	}
	if len(mm.Arrives()) == 0 || len(am.Transitions()) < 3 {
		t.Fatalf("到点/迁移流水缺失: %v %v", mm.Arrives(), am.Transitions())
	}
	// 警戒/仇恨流水：spring + fight（WarnModule）与 return（HateModule
	// 脱战路径）全程可见。
	joined := strings.Join(wm.Events(), ",") + "," + strings.Join(hm.Events(), ",")
	if !strings.Contains(joined, "spring") || !strings.Contains(joined, "fight") || !strings.Contains(joined, "return") {
		t.Fatalf("警戒/仇恨流水不全: %v / %v", wm.Events(), hm.Events())
	}
}

// TestPassiveNpcWarn 被动怪（WarnType=0）：玩家入球不入战；被击才入战。
func TestPassiveNpcWarn(t *testing.T) {
	k, _ := assemble10(t)
	npc, _ := k.CreateObject("CommonNpc")
	no := k.GetGameObj(npc)
	no.SetPos(0, 0, 0)
	_ = no.Set("WarnType", int32(0))
	k.RunEvent("CommonNpc", "OnEntry", npc, NullPersistID, nil)
	p, _ := k.CreateObject("player")
	k.GetGameObj(p).SetPos(3, 0, 0)
	k.Bus().TickHeartBeats(k, 1)
	if no.GetInt("CurState") != AIStateStand || no.Record("SpringList").Rows() != 0 {
		t.Fatal("被动怪不应主动入战")
	}
	k.DispatchIntCommand("CommonNpc", 618, npc, NullPersistID, VarList{p, int32(7)})
	if no.GetInt("CurState") != AIStateFight {
		t.Fatal("被动怪被击应入战")
	}
	if r := no.Record("HateList"); r.Rows() != 1 || r.Row(0).Int(1) != 7 {
		t.Fatalf("被击仇恨异常: %v", r.Row(0))
	}
}

// TestWarnGeometryTypes 警戒几何四型：球 / 圆柱 / 扇柱（朝向内命中、
// 背向不命中）/ 球+扇柱并集。
func TestWarnGeometryTypes(t *testing.T) {
	k, _ := assemble10(t)
	wm := module2[*WarnModule](t, k, "WarnModule")
	mk := func(warnType int32, face float32) (PersistID, *GameObject) {
		id, _ := k.CreateObject("CommonNpc")
		o := k.GetGameObj(id)
		o.SetPos(0, 0, 0)
		_ = o.Set("WarnType", warnType)
		_ = o.Set("WarnTime", int32(0))
		_ = o.Set("SphereRadius", float32(10))
		_ = o.Set("CylinderRadius", float32(10))
		_ = o.Set("CylinderLowerHeight", float32(-2))
		_ = o.Set("CylinderUpperHeight", float32(5))
		_ = o.Set("SectorHalfAngle", float32(45))
		o.SetFace(face)
		k.RunEvent("CommonNpc", "OnEntry", id, NullPersistID, nil)
		return id, o
	}
	p, _ := k.CreateObject("player")
	po := k.GetGameObj(p)

	// 扇柱（type 3）：正面 45° 半角内命中（dx=6,dz=6 → 45°）。
	_, o := mk(3, 0)
	po.SetPos(6, 0, 6)
	if !wm.inWarnGeometry(k, o, p) {
		t.Fatal("扇柱正面应命中")
	}
	po.SetPos(-6, 0, -6) // 背面（±135°）
	if wm.inWarnGeometry(k, o, p) {
		t.Fatal("扇柱背面不应命中")
	}
	// 圆柱（type 2）全向命中 + 高度界外不命中。
	_, o = mk(2, 0)
	po.SetPos(6, 0, -6)
	if !wm.inWarnGeometry(k, o, p) {
		t.Fatal("圆柱全向应命中")
	}
	po.SetPos(6, 9, -6) // y 超上高（NPC y=0，界 [−2,5]）
	if wm.inWarnGeometry(k, o, p) {
		t.Fatal("圆柱高度界外不应命中")
	}
	// 球+扇柱（type 4）：背面但球内命中。
	_, o = mk(4, 0)
	po.SetPos(6, 0, -6)
	if !wm.inWarnGeometry(k, o, p) {
		t.Fatal("球+扇柱并集（球内）应命中")
	}
	// 球（type 1）半径界外。
	_, o = mk(1, 0)
	po.SetPos(11, 0, 0)
	if wm.inWarnGeometry(k, o, p) {
		t.Fatal("球半径界外不应命中")
	}
}

// TestHateTransfer 仇恨转移：宠物 HateTransferScale=0.5 → 半数分流主人。
func TestHateTransfer(t *testing.T) {
	k, _ := assemble10(t)
	hm := module2[*HateModule](t, k, "HateModule")
	pet, _ := k.CreateObject("CommonNpc")
	po := k.GetGameObj(pet)
	master, _ := k.CreateObject("player")
	attacker, _ := k.CreateObject("player")
	_ = po.Set("MasterID", master)
	_ = po.Set("HateTransferScale", float32(0.5))
	if !hm.AddHate(k, pet, attacker, 100) {
		t.Fatal("AddHate 失败")
	}
	r := po.Record("HateList")
	if r.Rows() != 2 {
		t.Fatalf("转移后应两行: %d", r.Rows())
	}
	hate := map[uint32]int32{}
	for i := 0; i < r.Rows(); i++ {
		if v, ok := r.Row(i)[0].(PersistID); ok {
			hate[v.ID()] = r.Row(i).Int(1)
		}
	}
	if hate[attacker.ID()] != 50 || hate[master.ID()] != 50 {
		t.Fatalf("转移比例异常: %v", hate)
	}
}

// TestTempStopAI 临时停 AI：停机期泵面跳过（警戒不推进），恢复后续转。
func TestTempStopAI(t *testing.T) {
	k, _ := assemble10(t)
	ts := module2[*TempStopAI](t, k, "TempStopAI")
	npc, _ := k.CreateObject("CommonNpc")
	no := k.GetGameObj(npc)
	no.SetPos(0, 0, 0)
	_ = no.Set("WarnType", int32(1))
	_ = no.Set("SphereRadius", float32(10))
	_ = no.Set("WarnTime", int32(0))
	k.RunEvent("CommonNpc", "OnEntry", npc, NullPersistID, nil)
	p, _ := k.CreateObject("player")
	k.GetGameObj(p).SetPos(5, 0, 0)

	ts.TempStop(k, npc, 60)
	k.Bus().TickHeartBeats(k, 1)
	if no.GetInt("CurState") != AIStateStand {
		t.Fatal("停机期不应入战")
	}
	if no.GetInt("StopAI") != 1 {
		t.Fatal("StopAI 未置位")
	}
	// 到期恢复（直接改恢复时刻——避免真实等待）。
	ts.resumes[npc] = 1
	k.Bus().TickHeartBeats(k, 1)
	if no.GetInt("StopAI") != 0 {
		t.Fatal("到期未恢复")
	}
	if no.GetInt("CurState") != AIStateFight {
		t.Fatal("恢复后应续转入战")
	}
}

// TestAIChatAndCommand 说话引擎 + 受控移动：event_talk_rec 台词行序推进；
// Pet 跟随超保护半径收拢、停留即停。
func TestAIChatAndCommand(t *testing.T) {
	k, _ := assemble10(t)
	cm := module2[*AIChatWithOther](t, k, "AIChatWithOther")
	om := module2[*AIOnCommandMotion](t, k, "AIOnCommandMotion")

	npc, _ := k.CreateObject("CommonNpc")
	no := k.GetGameObj(npc)
	no.Record("event_talk_rec").AppendRow(VarList{float32(1.5), int32(7)})
	no.Record("event_talk_rec").AppendRow(VarList{float32(2), int32(8)})
	listener, _ := k.CreateObject("player")
	if line := cm.Say(k, npc, listener); line != 7 {
		t.Fatalf("首句=%d 期望 7", line)
	}
	if line := cm.Say(k, npc, listener); line != 8 {
		t.Fatalf("次句=%d 期望 8", line)
	}
	if line := cm.Say(k, npc, listener); line != 7 {
		t.Fatalf("循环句=%d 期望 7", line)
	}
	if cr := no.Record("AI_Chat_rec"); cr.Rows() != 1 {
		t.Fatalf("说话记录单槽应 1 行: %d", cr.Rows())
	}

	master, _ := k.CreateObject("player")
	k.GetGameObj(master).SetPos(0, 0, 0)
	no.SetPos(10, 0, 0)
	_ = no.Set("FollowProtectRadius", float32(3))
	_ = no.Set("MasterID", master)
	om.Command(k, npc, AIInstrFollow)
	if no.GetInt("AIMotioning") != 1 || no.GetFloat("FindPathX") != 0 {
		t.Fatalf("跟随移动未挂: %d/%v", no.GetInt("AIMotioning"), no.GetFloat("FindPathX"))
	}
	om.Command(k, npc, AIInstrStay)
	if no.GetInt("AIMotioning") != 0 {
		t.Fatal("停留应停移动")
	}
}
