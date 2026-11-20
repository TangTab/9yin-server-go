package fsgame

import (
	"path/filepath"
	"strings"
	"testing"
)

// assemble11 装配十一批六十八模块 + 玩家余量 schema（player.xml includes
// 链自带 autorestore/state/maxpara/safemode/doselect/ModifyBase；relive/
// playerchange/offline_compensate 三档显式并入 player 类）。
func assemble11(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))
	rv := mustRead(t, filepath.Join(td, "struct", "player", "relive.xml"))
	pc := mustRead(t, filepath.Join(td, "struct", "player", "playerchange.xml"))
	oc := mustRead(t, filepath.Join(td, "struct", "player", "offline_compensate.xml"))
	sc := mustRead(t, filepath.Join(td, "struct", "scene", "scene.xml"))

	k := NewKernel()
	k.ResourcePath = td
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	if err := k.LoadStruct("player", px); err != nil {
		t.Fatalf("LoadStruct player: %v", err)
	}
	for name, data := range map[string][]byte{
		"exp": ex, "relive": rv, "playerchange": pc, "offline_compensate": oc,
	} {
		if err := k.LoadStruct("player", data); err != nil {
			t.Fatalf("LoadStruct player(%s): %v", name, err)
		}
	}
	if err := k.LoadStruct("scene", sc); err != nil {
		t.Fatalf("LoadStruct scene: %v", err)
	}
	var r Registry
	for _, reg := range []func(*Registry) error{
		RegisterFirstBatch, RegisterSecondBatch, RegisterThirdBatch,
		RegisterFourthBatch, RegisterFifthBatch, RegisterSixthBatch,
		RegisterSeventhBatch, RegisterEighthBatch, RegisterNinthBatch,
		RegisterTenthBatch, RegisterEleventhBatch,
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

// TestEleventhBatchAssemble 十一期装配序 + 注册面 + schema 合并抽查。
func TestEleventhBatchAssemble(t *testing.T) {
	k, mods := assemble11(t)
	want := []string{"EnergyModule", "RevertPlayerModule", "SafeModeModule",
		"RoleExteriorModule", "PlayerChangeModule", "RestoreModule", "ReliveModule",
		"DeadModule", "FreshmanModule", "ObjPropRefreshModule",
		"OffLineCompensateModule", "SetPathModule", "PathFindingModule", "DoSelectModule"}
	var got []string
	for _, m := range mods {
		for _, w := range want {
			if m.Name() == w {
				got = append(got, w)
			}
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	// 恢复六泵 + 复活三泵 + 安全模式/变身/寻路心跳。
	for _, h := range []string{"RestoreModule::H_BeginRestore_HP",
		"RestoreModule::H_BeginRestore_DP", "ReliveModule::H_Relive",
		"ReliveModule::H_ReliveCDTimer", "SafeModeModule::HB_CheckSafeTime",
		"PlayerChangeModule::HB_ChangeCost"} {
		found := false
		for _, name := range k.Bus().HeartBeatNames() {
			if name == h {
				found = true
			}
		}
		if !found {
			t.Fatalf("心跳 %s 未注册", h)
		}
	}
	// schema：includes 链 + 三档显式并入。
	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	for _, prop := range []string{"HPUpSpeed", "State", "EY", "LastObject",
		"SafeTime", "RelivePositon", "bRoleChange", "OffLineExp", "HideHat"} {
		if _, ok := o.Class().Prop(prop); !ok {
			t.Fatalf("属性 %s 未随 includes/补档装载", prop)
		}
	}
	if o.Record("PropModifyRec") == nil {
		t.Fatal("PropModifyRec 未装载")
	}
}

// TestDeathReliveLoop 死亡→复活闭环：HP 归零（Fight 临界置 Dead）→ cmd 616
// 结算（State=dead + DeadAvenge 击杀者档案 + 复活倒计时登记）→ 倒计时内
// 请求被拒 → 到位后原地/回城复活（ReliveNoNearPos 强制回城 + RelivePositon
// 落位 + HP/MP 半恢复 + State 复位）。
func TestDeathReliveLoop(t *testing.T) {
	k, _ := assemble11(t)
	dm := module2[*DeadModule](t, k, "DeadModule")
	rm := module2[*ReliveModule](t, k, "ReliveModule")
	rm.ReliveDelay = 60 // 拉长倒计时便于测拒

	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	_ = k.SetInt(p, "MaxHP", 100)
	_ = k.SetInt(p, "MaxMP", 80)
	_ = k.SetInt(p, "HP", 30) // 先离零（初值 0 时同值不触发临界）
	_ = k.SetInt(p, "HP", 0)  // FightModule 临界面置 Dead=1
	if o.GetInt("Dead") != 1 {
		t.Fatal("Fight 临界未置死亡位")
	}
	killer, _ := k.CreateObject("player")
	_ = k.GetGameObj(killer).Set("name", "凶手甲")
	k.DispatchIntCommand("player", 616, p, NullPersistID, VarList{killer})
	if o.GetString("State") != "dead" || len(dm.Events()) == 0 {
		t.Fatalf("死亡结算异常: state=%q events=%v", o.GetString("State"), dm.Events())
	}
	if o.GetString("KillerName") != "凶手甲" {
		t.Fatalf("击杀者档案未写: %q", o.GetString("KillerName"))
	}
	// 倒计时内请求 → 拒。
	k.DispatchCommand("player", 290, p, NullPersistID, VarList{int32(0)})
	if !hasStr(rm.Events(), "reject-cd") {
		t.Fatalf("倒计时内应拒: %v", rm.Events())
	}
	// 到位（直接改 ready——避免等待）+ 禁最近点 → 模式 0 强制回城。
	rm.ready[p] = 1
	_ = o.Set("ReliveNoNearPos", int32(1))
	_ = o.Set("RelivePositon", "[huixian,10,0,20,0]")
	k.DispatchCommand("player", 290, p, NullPersistID, VarList{int32(0)})
	if o.GetInt("Dead") != 0 || o.GetString("State") != "" {
		t.Fatalf("复活未复位: dead=%d state=%q", o.GetInt("Dead"), o.GetString("State"))
	}
	if o.GetInt("HP") != 50 || o.GetInt("MP") != 40 {
		t.Fatalf("半恢复异常: hp=%d mp=%d", o.GetInt("HP"), o.GetInt("MP"))
	}
	if x, _, z := o.Pos(); x != 10 || z != 20 {
		t.Fatalf("回城落位异常: %v,%v", x, z)
	}
	if !hasStr(rm.Events(), "relive") {
		t.Fatalf("复活流水缺失: %v", rm.Events())
	}
}

// TestRestorePumps 恢复泵：HP/MP 按速度回复（死亡停）；能量族速度属性
// 缺省 0 不自然回复。
func TestRestorePumps(t *testing.T) {
	k, _ := assemble11(t)
	rm := module2[*RestoreModule](t, k, "RestoreModule")
	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	_ = k.SetInt(p, "MaxHP", 100)
	_ = k.SetInt(p, "HP", 50)
	_ = k.SetInt(p, "HPUpSpeed", 10)
	_ = k.SetInt(p, "MaxMP", 60)
	_ = k.SetInt(p, "MP", 55)
	_ = k.SetInt(p, "MPUpSpeed", 30)
	_ = k.SetInt(p, "MaxEY", 100)
	_ = k.SetInt(p, "EY", 10)
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("HP") != 60 || o.GetInt("MP") != 60 {
		t.Fatalf("回复异常: hp=%d mp=%d", o.GetInt("HP"), o.GetInt("MP"))
	}
	if o.GetInt("EY") != 10 {
		t.Fatal("能量族无速度属性不应回复")
	}
	if !hasStr(rm.Events(), "hp+10") {
		t.Fatalf("恢复流水缺失: %v", rm.Events())
	}
	// 死亡面停泵。
	_ = k.SetInt(p, "Dead", 1)
	_ = k.SetInt(p, "HP", 50)
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("HP") != 50 {
		t.Fatal("死亡面不应回复")
	}
	// 满值不越。
	_ = k.SetInt(p, "Dead", 0)
	_ = k.SetInt(p, "HP", 98)
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("HP") != 100 {
		t.Fatalf("满值钳制异常: %d", o.GetInt("HP"))
	}
}

// TestSafeModeFlow 安全模式：上线重启保护倒计时 → 密码错误拒 → 正确解锁；
// 无密码面倒计时归零自动解锁。
func TestSafeModeFlow(t *testing.T) {
	k, _ := assemble11(t)
	sm := module2[*SafeModeModule](t, k, "SafeModeModule")
	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	_ = k.SetInt(p, "SafeTime", 2)
	_ = k.SetInt(p, "IsHaveSafeModePsw", 1)
	_ = k.SetString(p, "SafeModePsw", "pw123")
	k.RunEvent("player", "OnReady", p, NullPersistID, nil)
	if o.GetInt("CurSafeTime") != 2 || !sm.SafeModeActive(k, p) {
		t.Fatal("上线应进入保护")
	}
	k.DispatchCommand("player", 261, p, NullPersistID, VarList{"wrong"})
	if !sm.SafeModeActive(k, p) {
		t.Fatal("错误密码不应解锁")
	}
	k.DispatchCommand("player", 261, p, NullPersistID, VarList{"pw123"})
	if sm.SafeModeActive(k, p) {
		t.Fatal("正确密码应解锁")
	}
	// 无密码面：倒计时归零自动解锁。
	p2, _ := k.CreateObject("player")
	o2 := k.GetGameObj(p2)
	_ = k.SetInt(p2, "SafeTime", 1)
	k.RunEvent("player", "OnReady", p2, NullPersistID, nil)
	k.Bus().TickHeartBeats(k, 1)
	if o2.GetInt("CurSafeTime") != 0 || !hasStr(sm.Events(), "auto-unlock") {
		t.Fatalf("无密码自动解锁异常: %d %v", o2.GetInt("CurSafeTime"), sm.Events())
	}
}

// TestPlayerChangeFlow 变身：模型串 + 时限 → 计时消耗 → 到时复原。
func TestPlayerChangeFlow(t *testing.T) {
	k, _ := assemble11(t)
	cm := module2[*PlayerChangeModule](t, k, "PlayerChangeModule")
	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	if !cm.ChangeRole(k, p, "model_yao", 2) {
		t.Fatal("变身失败")
	}
	if o.GetInt("bRoleChange") != 1 || o.GetString("MainRoleChange") != "model_yao" {
		t.Fatal("变身态未置")
	}
	k.Bus().TickHeartBeats(k, 1)
	if o.GetInt("ChangeLeftTime") != 1 || o.GetInt("ChangeCostTime") != 1 {
		t.Fatalf("计时异常: left=%d cost=%d", o.GetInt("ChangeLeftTime"), o.GetInt("ChangeCostTime"))
	}
	k.Bus().TickHeartBeats(k, 1) // left 1→0（消耗中）
	if o.GetInt("ChangeLeftTime") != 0 {
		t.Fatalf("计时异常: left=%d", o.GetInt("ChangeLeftTime"))
	}
	k.Bus().TickHeartBeats(k, 1) // left=0 → 复原
	if o.GetInt("bRoleChange") != 0 || o.GetString("MainRoleChange") != "" {
		t.Fatal("到时未复原")
	}
	if !hasStr(cm.Events(), "revert") {
		t.Fatalf("复原流水缺失: %v", cm.Events())
	}
}

// TestOfflineCompensate 离线补偿：下线落 LogoutTime → 回载按时长结算
// OffLineExp → cmd 884 领取入 PlayerExp。
func TestOfflineCompensate(t *testing.T) {
	k, _ := assemble11(t)
	om := module2[*OffLineCompensateModule](t, k, "OffLineCompensateModule")
	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	_ = o.Set("LogoutTime", unixNow()-7200) // 离线 2h → 2000 exp
	k.RunEvent("player", "OnRecover", p, NullPersistID, nil)
	if o.GetInt("OffLineExp") != 2000 {
		t.Fatalf("离线经验异常: %d", o.GetInt("OffLineExp"))
	}
	before := o.GetInt("PlayerExp")
	k.DispatchIntCommand("player", 884, p, NullPersistID, nil)
	if o.GetInt("OffLineExp") != 0 || o.GetInt("PlayerExp") != before+2000 {
		t.Fatalf("领取异常: off=%d exp=%d→%d", o.GetInt("OffLineExp"), before, o.GetInt("PlayerExp"))
	}
	if !hasStr(om.Events(), "claim") {
		t.Fatalf("领取流水缺失: %v", om.Events())
	}
}

// TestPathSelectAndRefresh 路径装载/寻路简化/选中面/属性修正聚合。
func TestPathSelectAndRefresh(t *testing.T) {
	k, _ := assemble11(t)
	sm := module2[*SetPathModule](t, k, "SetPathModule")
	pf := module2[*PathFindingModule](t, k, "PathFindingModule")
	ds := module2[*DoSelectModule](t, k, "DoSelectModule")
	or := module2[*ObjPropRefreshModule](t, k, "ObjPropRefreshModule")

	// 场景物化 → SceneMapProcess 登记 → 寻路分段。
	scene, err := k.CreateObject("scene")
	if err != nil {
		t.Fatal(err)
	}
	pts := pf.FindPath(k, scene, 0, 0, 12, 0, 5)
	if len(pts) != 3 || pts[len(pts)-1].X != 12 {
		t.Fatalf("寻路分段异常: %v", pts)
	}
	if !hasStr(pf.Ops(), "map") || !hasStr(pf.Ops(), "find") {
		t.Fatalf("寻路流水缺失: %v", pf.Ops())
	}
	// NPC 路径装载（PathID 在 ai_npc_data——需并入）。
	cn := mustRead(t, filepath.Join("..", "..", "testdata", "struct", "npc", "CommonNpc.xml"))
	ai := mustRead(t, filepath.Join("..", "..", "testdata", "struct", "npc_property", "ai_motion.xml"))
	ad := mustRead(t, filepath.Join("..", "..", "testdata", "struct", "npc_property", "ai_npc_data.xml"))
	for name, data := range map[string][]byte{"CommonNpc": cn, "ai_motion": ai, "ai_npc_data": ad} {
		if err := k.LoadStruct("CommonNpc", data); err != nil {
			t.Fatalf("NPC schema 装载失败(%s): %v", name, err)
		}
	}
	npc, _ := k.CreateObject("CommonNpc")
	sm.SetPath(k, npc, "path_7", []PathPoint{{X: 1, Z: 2}, {X: 3, Z: 4}})
	no := k.GetGameObj(npc)
	if no.GetString("PathID") != "path_7" || no.Record("AIFindPathRec").Rows() != 2 {
		t.Fatalf("路径装载异常: %s %d", no.GetString("PathID"), no.Record("AIFindPathRec").Rows())
	}
	// 选中/取消/查询。
	p, _ := k.CreateObject("player")
	ds.Select(k, p, npc)
	if v, _ := k.GetGameObj(p).Get("LastObject").(PersistID); v != npc {
		t.Fatal("选中未写 LastObject")
	}
	k.DispatchCommand("player", 440, p, NullPersistID, nil)
	k.DispatchIntCommand("player", 613, p, NullPersistID, nil)
	if !hasStr(ds.Events(), "select") || !hasStr(ds.Events(), "cancel") {
		t.Fatalf("选中流水缺失: %v", ds.Events())
	}
	// 属性修正聚合：PropModifyRec 行（属性名, double 值）→ Recalc 累加。
	po := k.GetGameObj(p)
	_ = k.SetInt(p, "HP", 50)
	po.Record("PropModifyRec").AppendRow(VarList{"HP", float64(7), int32(0), "test", int32(0), int32(0), int32(0), ""})
	or.Recalc(k, p)
	if po.GetInt("HP") != 57 || or.Refreshes() == 0 {
		t.Fatalf("修正聚合异常: hp=%d refreshes=%d", po.GetInt("HP"), or.Refreshes())
	}
}

// TestRevertSnapshot 还原面：快照 → 改值 → cust 712 回写。
func TestRevertSnapshot(t *testing.T) {
	k, _ := assemble11(t)
	rm := module2[*RevertPlayerModule](t, k, "RevertPlayerModule")
	p, _ := k.CreateObject("player")
	o := k.GetGameObj(p)
	_ = k.SetInt(p, "HP", 80)
	if !rm.NxStartRevert(k, p) {
		t.Fatal("快照失败")
	}
	_ = k.SetInt(p, "HP", 10)
	k.DispatchCommand("player", 712, p, NullPersistID, nil)
	if o.GetInt("HP") != 80 {
		t.Fatalf("还原异常: hp=%d", o.GetInt("HP"))
	}
	k.DispatchCommand("player", 713, p, NullPersistID, nil)
	if !hasStr(rm.Ops(), "revert") {
		t.Fatalf("还原流水缺失: %v", rm.Ops())
	}
}

// hasStr 流水包含（测试面）。
func hasStr(list []string, want string) bool {
	for _, s := range list {
		if strings.Contains(s, want) {
			return true
		}
	}
	return false
}
