package fsgame

import "testing"

// assemble9 装配九批五十模块（event_spring 记录随 player includes 装载）。
func assemble9(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	k, mods := assemble8(t)
	// 追加第九批（复用八期装配内核：Registry 追加 + 补 Init）。
	var r Registry
	for _, reg := range []func(*Registry) error{RegisterNinthBatch} {
		if err := reg(&r); err != nil {
			t.Fatal(err)
		}
	}
	ninth, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble ninth: %v", err)
	}
	return k, append(mods, ninth...)
}

// TestNinthBatchAssemble 九期装配序 + 事件弹簧七表真档。
func TestNinthBatchAssemble(t *testing.T) {
	k, mods := assemble9(t)
	want := []string{"TriggerProcesserModule", "TaskExecutor", "SwitchExecutor",
		"RewardExecutor", "EventSpringModule"}
	var got []string
	for _, m := range mods {
		switch m.Name() {
		case "TriggerProcesserModule", "TaskExecutor", "SwitchExecutor",
			"RewardExecutor", "EventSpringModule":
			got = append(got, m.Name())
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	es := module2[*EventSpringModule](t, k, "EventSpringModule")
	if len(es.Events) == 0 || len(es.Conditions) == 0 || len(es.Behaviors) == 0 ||
		len(es.Rewards) == 0 || len(es.Switches) == 0 || len(es.Tasks) == 0 ||
		len(es.Notices) == 0 {
		t.Fatalf("七表装载异常: %d/%d/%d/%d/%d/%d/%d", len(es.Events),
			len(es.Conditions), len(es.Behaviors), len(es.Rewards),
			len(es.Switches), len(es.Tasks), len(es.Notices))
	}
	// 真档抽查：事件 1（ConditionId=1/BehaviorId=11,12/Duration=900）。
	if e := es.NxGetEventByID(1); e == nil || atoi32(e["Duration"]) != 900 || e["BehaviorId"] != "11,12" {
		t.Fatalf("event_spring 真档异常: %v", e)
	}
	if es.NxNotExistBehaviorID("11") {
		t.Fatal("behavior 11 应存在")
	}
	// 玩家记录随 includes 装载。
	p, _ := k.CreateObject("player")
	if k.GetGameObj(p).Record("PlayerESRec") == nil {
		t.Fatal("PlayerESRec 未随 player includes 装载")
	}
}

// TestEventSpringLifecycle 事件弹簧闭环：启动（公告下发）→ 开关进入
//（等级门）→ 任务推进（TaskExecutor→QuestModule 协作）→ 奖励执行 →
// 到期结束（HB_EndES）。
func TestEventSpringLifecycle(t *testing.T) {
	k, _ := assemble9(t)
	es := module2[*EventSpringModule](t, k, "EventSpringModule")
	sw := module2[*SwitchExecutor](t, k, "SwitchExecutor")
	te := module2[*TaskExecutor](t, k, "TaskExecutor")
	re := module2[*RewardExecutor](t, k, "RewardExecutor")
	qm := module2[*QuestModule](t, k, "QuestModule")

	p, _ := k.CreateObject("player")
	_ = k.SetInt(p, "Level", 30)
	k.RunEvent("player", "OnRecover", p, NullPersistID, nil)

	// 启动事件 1（真档：NoticeId=100301,100302,100303 → 三公告）。
	if !es.StartES(k, 1) {
		t.Fatal("事件启动失败")
	}
	if len(es.FiredNotices()) != 3 {
		t.Fatalf("公告流水 %v", es.FiredNotices())
	}
	// 重复启动拒。
	if es.StartES(k, 1) {
		t.Fatal("重复启动未被拒")
	}
	// 开关进入：等级门（30 ∈ [1,60] 过）→ JoinES（PlayerESRec 一行）。
	k.DispatchCommand("player", 758, p, NullPersistID, VarList{"1"})
	if len(sw.Entered()) != 1 {
		t.Fatalf("开关进入流水 %v", sw.Entered())
	}
	if r := k.GetGameObj(p).Record("PlayerESRec"); r == nil || r.Rows() != 1 || r.Row(0).Int(esEventID) != 1 {
		t.Fatalf("参与记录异常: %v", r)
	}
	// 等级门外：1 级玩家拒。
	low, _ := k.CreateObject("player")
	_ = k.SetInt(low, "Level", 0)
	k.DispatchCommand("player", 758, low, NullPersistID, VarList{"1"})
	if k.GetGameObj(low).Record("PlayerESRec").Rows() != 0 {
		t.Fatal("等级门外可进入")
	}
	// 任务推进：task 1（真档 SteamCriteriaId=30007）→ QuestModule 进度。
	// 先给玩家挂 30007 进度行（经 QuestModule 接取面）。
	qm.Criteria["30007"] = map[string]string{"ID": "30007", "CriteriaType": "1"}
	qm.Quests["9201"] = map[string]string{"ID": "9201", "MinLevel": "1",
		"CriteriaList": "30007", "AwardID": "x"}
	qm.AcceptQuest(k, p, 9201)
	k.DispatchIntCommand("player", 927, p, NullPersistID, VarList{"1", int32(1)})
	if len(te.Ops()) == 0 {
		t.Fatal("任务执行器未流水")
	}
	cr := k.GetGameObj(p).Record("QuestCriteriaRecord")
	advanced := false
	for i := 0; i < cr.Rows(); i++ {
		if cr.Row(i).Int(qcID) == 30007 && cr.Row(i).Int(qcCur) >= 1 {
			advanced = true
		}
	}
	if !advanced {
		t.Fatal("事件任务未推进 QuestModule 进度")
	}
	// 奖励：reward 1（真档 CriteriaAwards="100,29480,4"）。
	var items []string
	re.RewardOut = func(pl PersistID, itemID string, count int32) { items = append(items, itemID) }
	if n := re.Execute(k, p, "1", false); n == 0 {
		t.Fatal("奖励未发放")
	}
	_ = items
	// 到期结束：真档 Duration=900 拍——直接耗尽。
	for run := es.RunningES()[1]; run != nil && run.Duration > 0; {
		k.Bus().TickHeartBeats(k, 1)
		run = es.RunningES()[1]
	}
	if es.RunningES()[1] != nil {
		t.Fatal("到期未结束")
	}
	// Lua 面：nx_end_es 幂等。
	es.NxEndES(k, 1)
	if !hasTplStr(es.Ops(), "end:1") {
		t.Fatal("结束未流水")
	}
}

func hasTplStr(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
