package fsgame

import (
	"path/filepath"
	"strings"
	"testing"
)

// assemble4 装配四批（登录/社交/战斗/任务域）二十四模块。
func assemble4(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))

	k := NewKernel()
	k.ResourcePath = td
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	if err := k.LoadStruct("player", px); err != nil {
		t.Fatalf("LoadStruct player: %v", err)
	}
	if err := k.LoadStruct("player", ex); err != nil {
		t.Fatalf("LoadStruct player(exp): %v", err)
	}
	var r Registry
	for _, reg := range []func(*Registry) error{
		RegisterFirstBatch, RegisterSecondBatch, RegisterThirdBatch, RegisterFourthBatch,
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

// questRowOf 测试面：承接表行。
func questRowOf(k *Kernel, self PersistID, qid int32) VarList {
	r := k.GetGameObj(self).Record("QuestRecord")
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).Int(qrID) == qid {
			return r.Row(i)
		}
	}
	return nil
}

// TestFourthBatchAssemble 四期装配序 + 任务/成就配置真档装载。
func TestFourthBatchAssemble(t *testing.T) {
	k, mods := assemble4(t)
	want := []string{"QuestModule", "QuestCriteriaModule", "AchievementModule"}
	var got []string
	for _, m := range mods {
		switch m.Name() {
		case "QuestModule", "QuestCriteriaModule", "AchievementModule":
			got = append(got, m.Name())
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	qm := module2[*QuestModule](t, k, "QuestModule")
	// 九阴真档（npcgen 生成）：22072 谷中考验。
	if cfg := qm.Quests["22072"]; cfg == nil || cfg["Name"] != "谷中考验" ||
		cfg["AcceptNpc"] != "WorldNpc01452" || cfg["preQuestList"] != "22070" {
		t.Fatalf("九阴任务 22072 未装载: %v", qm.Quests["22072"])
	}
	// SubFunction 接线合成：24013 → hunter 4001 → monv040100×3。
	cfg13 := qm.Quests["24013"]
	if cfg13 == nil || cfg13["CriteriaList"] == "" {
		t.Fatalf("24013 未合成步骤: %v", cfg13)
	}
	hunted := false
	for _, cid := range strings.Split(cfg13["CriteriaList"], ",") {
		if c := qm.Criteria[cid]; c != nil && c["CriteriaType"] == CriteriaKillNpc &&
			c["InterNpc"] == "monv040100" && atoi32(c["MaxCount"]) == 3 {
			hunted = true
		}
	}
	if !hunted {
		t.Fatalf("24013 杀怪子模板未合成: %v", cfg13["CriteriaList"])
	}
	// 奖励合成：24241 勇闯挑灯崖 → TaskItem 1117（taskitem_7h×1，IsGive=0）。
	if a := qm.Awards["24241"]; a == nil || a["DefaultItemList"] != "taskitem_7h,1" {
		t.Fatalf("TaskItem 奖励未合成: %v", qm.Awards["24241"])
	}
	// 剧情/文案：22145 → story 3022 两段对话 @ WorldNpc06127；
	// title_22072 文案（idres 版本）。
	cfg145 := qm.Quests["22145"]
	story := 0
	for _, cid := range strings.Split(cfg145["CriteriaList"], ",") {
		if c := qm.Criteria[cid]; c != nil && c["StepType"] == "story" && c["InterNpc"] == "WorldNpc06127" {
			story++
		}
	}
	if story != 2 {
		t.Fatalf("22145 剧情子模板应 2 段: %d (%v)", story, cfg145["CriteriaList"])
	}
	if qm.TaskText("title_22072") != "另有隐情" {
		t.Fatalf("任务文案未装载: %q", qm.TaskText("title_22072"))
	}
	if txt := qm.TaskText("CanAcceptMenu_22001"); !strings.Contains(txt, "新的开始") {
		t.Fatalf("菜单文案未装载: %q", txt)
	}
	// 护送占位：24019 → convoy 4001 → Convoy24019（NpcConfig 命中）。
	c19 := qm.Quests["24019"]
	convoy := 0
	for _, cid := range strings.Split(c19["CriteriaList"], ",") {
		if c := qm.Criteria[cid]; c != nil && c["StepType"] == "convoy" && c["NpcId"] == "Convoy24019" && c["CriteriaType"] == "" {
			convoy++
		}
	}
	if convoy != 1 {
		t.Fatalf("24019 护送占位子模板缺失: %v", c19["CriteriaList"])
	}
	am := module2[*AchievementModule](t, k, "AchievementModule")
	if cfg := am.Achievements["1"]; cfg == nil || atoi32(cfg["Points"]) != 10 {
		t.Fatalf("Achievement.xml 未装载: %v", am.Achievements["1"])
	}
	if c := am.Criteria["92"]; c == nil {
		t.Fatalf("AchievementCriteria.xml 未装载")
	}
	// 任务记录已随 player includes 装载。
	p, _ := k.CreateObject("player")
	if k.GetGameObj(p).Record("QuestRecord") == nil || k.GetGameObj(p).Record("AchievementCompleted") == nil {
		t.Fatal("任务/成就记录未随 includes 装载")
	}
	// 真档全闭环：接 2412 峨嵋剑阵（无前置）→ 各步骤推满 → 可提交。
	cfg12 := qm.Quests["2412"]
	if cfg12 == nil || cfg12["CriteriaList"] == "" {
		t.Fatalf("2412 未合成步骤: %v", cfg12)
	}
	if !qm.AcceptQuest(k, p, 2412) {
		t.Fatal("真档任务 2412 接取失败")
	}
	for _, cid := range strings.Split(cfg12["CriteriaList"], ",") {
		c := qm.Criteria[cid]
		max := atoi32(c["MaxCount"])
		if max <= 0 {
			max = 1
		}
		qm.UpdateCriteria(k, p, atoi32(cid), max, true)
	}
	if !qm.SubmitQuest(k, p, 2412) {
		t.Fatal("真档任务 2412 未达可提交")
	}
	// NPC 面（任务侧索引）：22145 第一回——接 WorldNpc06008 / 交 WorldNpc06127。
	if got := qm.AcceptableQuests(k, p, "WorldNpc06008"); !containsInt(got, 22145) {
		t.Fatalf("WorldNpc06008 可接清单缺 22145: %v", got)
	}
	if !qm.AcceptQuest(k, p, 22145) {
		t.Fatal("真档任务 22145 接取失败")
	}
	if got := qm.AcceptableQuests(k, p, "WorldNpc06008"); containsInt(got, 22145) {
		t.Fatalf("接取后可接清单应去重: %v", got)
	}
	for _, cid := range strings.Split(qm.Quests["22145"]["CriteriaList"], ",") {
		max := atoi32(qm.Criteria[cid]["MaxCount"])
		if max <= 0 {
			max = 1
		}
		qm.UpdateCriteria(k, p, atoi32(cid), max, true)
	}
	if got := qm.SubmittableQuests(k, p, "WorldNpc06127"); !containsInt(got, 22145) {
		t.Fatalf("WorldNpc06127 可交清单缺 22145: %v", got)
	}
	// 剧情顺序播放：22145 的 interact 与 story 两段同 NPC（WorldNpc06127）
	// ——首次对话推 interact+story①（2 条），第二次只推 story②（1 条），
	// 之后 0 条（Order 门控，不一次播完）。
	p2, _ := k.CreateObject("player")
	if !qm.AcceptQuest(k, p2, 22145) {
		t.Fatal("剧情顺序用例接取失败")
	}
	qcm := module2[*QuestCriteriaModule](t, k, "QuestCriteriaModule")
	talk := func() int {
		before := len(qcm.Ops())
		k.RunEvent("CommonNpc", "OnSelect", NullPersistID, p2, VarList{"WorldNpc06127"})
		return len(qcm.Ops()) - before
	}
	for i, want := range []int{2, 1, 0} {
		if got := talk(); got != want {
			t.Fatalf("第 %d 次对话推进数 %d，期望 %d", i+1, got, want)
		}
	}
}

// containsInt 切片包含（测试面）。
func containsInt(list []int32, want int32) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// TestQuestMenuFlow 对话菜单闭环（cmd 22）：OnSelect 建上下文 → 菜单接取
// （NPC 门）→ 步骤推满 → 菜单提交（NPC 门）→ 默认奖励出口流水。
// 用例 24241 勇闯挑灯崖（接 FuncNpc00003 / 交 WorldNpc00027，奖励
// taskitem_7h×1）。
func TestQuestMenuFlow(t *testing.T) {
	k, _ := assemble4(t)
	qm := module2[*QuestModule](t, k, "QuestModule")
	p, _ := k.CreateObject("player")
	qm.setRewarded(k, p, 24240) // 前置完成位

	menu := func(action, questID int32) {
		k.DispatchIntCommand("player", 22, p, NullPersistID, VarList{action, questID})
	}

	// 错误 NPC 接取被拒（先对交任务 NPC 对话再试接）。
	k.RunEvent("CommonNpc", "OnSelect", NullPersistID, p, VarList{"WorldNpc00027"})
	menu(1, 24241)
	if qm.IsQuestHaving(k, p, 24241) {
		t.Fatal("非接取 NPC 的菜单接取未被拒")
	}
	// 正确 NPC：接取 → 步骤推满 → 提交（菜单，NPC 门过）→ 奖励流水。
	k.RunEvent("CommonNpc", "OnSelect", NullPersistID, p, VarList{"FuncNpc00003"})
	menu(1, 24241)
	if !qm.IsQuestHaving(k, p, 24241) {
		t.Fatal("菜单接取失败")
	}
	for _, cid := range strings.Split(qm.Quests["24241"]["CriteriaList"], ",") {
		c := qm.Criteria[cid]
		max := atoi32(c["MaxCount"])
		if max <= 0 {
			max = 1
		}
		qm.UpdateCriteria(k, p, atoi32(cid), max, true)
	}
	k.RunEvent("CommonNpc", "OnSelect", NullPersistID, p, VarList{"WorldNpc00027"})
	menu(2, 24241)
	if qm.IsQuestHaving(k, p, 24241) {
		t.Fatal("菜单提交失败")
	}
	awardSeen := false
	for _, op := range qm.Ops() {
		if strings.HasPrefix(op, "award:") && strings.Contains(op, "item=1") {
			awardSeen = true
		}
	}
	if !awardSeen {
		t.Fatalf("默认奖励出口流水缺失: %v", qm.Ops())
	}
}

// TestConvoyRuntime 护送闭环（黑金官方链路蓝图）：接 24019 茶林遇险
// （前置 24018 先置完成位；三步骤=护送+杀怪+交互）→ 物化 ConvoyNpc
// （LeaderNPC）+ 保护对象登记 → 到达回调推进护送子模板（其余步骤推满）
// → 可提交；超时面走 HB_Convoy 失败清理。装配用八期（NpcResourceModule
// 在八期批次）。
func TestConvoyRuntime(t *testing.T) {
	k, _ := assemble8(t)
	cnData := mustRead(t, filepath.Join("..", "..", "testdata", "struct", "npc", "ConvoyNpc.xml"))
	if err := k.LoadStruct("ConvoyNpc", cnData); err != nil {
		t.Fatalf("LoadStruct ConvoyNpc: %v", err)
	}
	qm := module2[*QuestModule](t, k, "QuestModule")
	qcm := module2[*QuestCriteriaModule](t, k, "QuestCriteriaModule")

	// 接取 → 物化与登记。
	p, _ := k.CreateObject("player")
	qm.setRewarded(k, p, 24018)
	if !qm.AcceptQuest(k, p, 24019) {
		t.Fatal("护送任务 24019 接取失败")
	}
	r := k.GetGameObj(p).Record("Convoy_ProtectObj_Rec")
	if r == nil || r.Rows() != 1 {
		t.Fatalf("保护对象登记异常: rows=%v", r)
	}
	npc, _ := r.Row(0)[4].(PersistID)
	if !npc.Valid() || k.GetGameObj(npc) == nil {
		t.Fatalf("护送 NPC 未物化: %v", npc)
	}
	if got := k.GetGameObj(npc).ClassName(); got != "ConvoyNpc" {
		t.Fatalf("物化类=%q，期望 ConvoyNpc", got)
	}
	if got := k.GetGameObj(npc).GetString("ConfigID"); got != "Convoy24019" {
		t.Fatalf("物化 ConfigID=%q", got)
	}
	criteriaID := k.GetGameObj(npc).GetInt("ConvoyCriteriaID")
	if criteriaID == 0 || k.GetGameObj(p).GetInt("ConvoyCriteriaID") != criteriaID {
		t.Fatalf("ConvoyCriteriaID 未写全: npc=%d player=%d",
			criteriaID, k.GetGameObj(p).GetInt("ConvoyCriteriaID"))
	}

	// 到达（未来移动运行时回调面）→ 护送子模板推进；杀怪/交互步骤推满
	// → 可提交 → 保护对象清。
	qcm.NpcArriveDest(k, p, npc)
	for _, cid := range strings.Split(qm.Quests["24019"]["CriteriaList"], ",") {
		if c := qm.Criteria[cid]; c != nil && c["StepType"] != "convoy" {
			max := atoi32(c["MaxCount"])
			if max <= 0 {
				max = 1
			}
			qm.UpdateCriteria(k, p, atoi32(cid), max, true)
		}
	}
	if !qm.SubmitQuest(k, p, 24019) {
		t.Fatal("护送到达后未达可提交")
	}
	if r.Rows() != 0 {
		t.Fatalf("到达后保护对象未清: %d", r.Rows())
	}

	// 超时失败面（HB_Convoy）。
	p2, _ := k.CreateObject("player")
	qm.setRewarded(k, p2, 24018)
	if !qm.AcceptQuest(k, p2, 24019) {
		t.Fatal("超时用例接取失败")
	}
	if err := k.GetGameObj(p2).Set("ConvoyEndTime", int64(1)); err != nil {
		t.Fatalf("置过期超时: %v", err)
	}
	k.Bus().TickHeartBeats(k, 1)
	if qm.IsQuestHaving(k, p2, 24019) {
		t.Fatal("超时后任务未失败清理")
	}
}

// TestQuestLifecycle 任务闭环：接任务（等级/前置门）→ 引擎推进（杀怪/对话）→
// 全子模板完成转可提交 → 提交（奖励出口+完成位集）→ 后继任务可接。
func TestQuestLifecycle(t *testing.T) {
	k, _ := assemble4(t)
	qm := module2[*QuestModule](t, k, "QuestModule")
	p, _ := k.CreateObject("player")
	_ = k.SetInt(p, "Level", 10)
	k.RunEvent("player", "OnRecover", p, NullPersistID, nil)

	// 合成任务配置：双子模板（对话 4 + 杀怪 1），奖励行。
	qm.Quests["9001"] = map[string]string{
		"ID": "9001", "Name": "初入江湖", "MinLevel": "5", "GiveUp": "1",
		"CriteriaList": "9001,9002", "AwardID": "9901", "postQuestList": "9002", "preQuestList": "",
	}
	qm.Criteria["9001"] = map[string]string{"ID": "9001", "CriteriaType": "4", "InterNpc": "npc_master"}
	qm.Criteria["9002"] = map[string]string{"ID": "9002", "CriteriaType": "1", "MaxCount": "3"}
	qm.Awards["9901"] = map[string]string{"ID": "9901", "Exp": "100", "Money": "50", "DefaultItemList": "Item_001,2"}
	qm.Quests["9002"] = map[string]string{
		"ID": "9002", "Name": "后继", "MinLevel": "1", "preQuestList": "9001",
		"CriteriaList": "9002", "AwardID": "9901", "GiveUp": "1",
	}

	// 等级门：5 级任务，10 级玩家过。
	if !qm.AcceptQuest(k, p, 9001) {
		t.Fatal("接任务失败")
	}
	// 重复接 → 拒（sys_quest_0005）。
	if qm.AcceptQuest(k, p, 9001) {
		t.Fatal("重复接任务未被拒")
	}
	if !hasTpl(qm.SysMsgs(), "sys_quest_0005") {
		t.Fatal("重复接任务未通知")
	}
	// 前置门：9002 需 9001 完成。
	if qm.AcceptQuest(k, p, 9002) {
		t.Fatal("前置未完成可接后继")
	}
	// 承接行 + 进度行。
	if row := questRowOf(k, p, 9001); row == nil || row.Int(qrState) != QSActive {
		t.Fatalf("承接行 %+v", row)
	}
	cr := k.GetGameObj(p).Record("QuestCriteriaRecord")
	if cr.Rows() != 2 || cr.Row(1).Int(qcMax) != 3 {
		t.Fatalf("进度行初始化异常: %v", cr.Row(1))
	}
	// 未完成不可提交。
	if qm.SubmitQuest(k, p, 9001) {
		t.Fatal("未完成可提交")
	}
	// 引擎推进：NPC 对话（type 4，InterNpc=npc_master）。
	k.DispatchIntCommand("player", 934, p, NullPersistID, VarList{int32(9001), int32(1)})
	if cr.Row(0).Int(qcState) != CSDone {
		t.Fatal("对话子模板未完成")
	}
	// 杀怪推进 2 次（还差 1）。
	k.DispatchIntCommand("player", 934, p, NullPersistID, VarList{int32(9002), int32(2)})
	if cr.Row(1).Int(qcCur) != 2 || cr.Row(1).Int(qcState) != CSDoing {
		t.Fatalf("杀怪进度 %+v", cr.Row(1))
	}
	if questRowOf(k, p, 9001).Int(qrState) != QSActive {
		t.Fatal("未全完成即转可提交")
	}
	// 第 3 只 → 全完成 → 可提交（sys_quest_0031 + COMPLETEQUEST）。
	k.DispatchIntCommand("player", 934, p, NullPersistID, VarList{int32(9002), int32(1)})
	if questRowOf(k, p, 9001).Int(qrState) != QSCanSubmit {
		t.Fatal("全完成后未转可提交")
	}
	if !hasTpl(qm.SysMsgs(), "sys_quest_0031") {
		t.Fatal("完成未通知 sys_quest_0031")
	}
	// 奖励出口 + 提交（cust 880 子命令 2）。
	var awarded bool
	qm.AwardOut = func(pl PersistID, exp, money int32, items string) {
		awarded = exp == 100 && money == 50 && items == "Item_001,2"
	}
	if !k.DispatchCommand("player", 880, p, NullPersistID, VarList{int32(2), int32(9001)}) {
		t.Fatal("提交命令 880 未命中")
	}
	if !awarded {
		t.Fatal("奖励出口参数异常")
	}
	if questRowOf(k, p, 9001) != nil || cr.Rows() != 0 {
		t.Fatal("提交后未清行")
	}
	if !qm.isRewarded(k, p, 9001) {
		t.Fatal("完成位集未置位")
	}
	// 后继可接（前置位集已置）。
	if !qm.AcceptQuest(k, p, 9002) {
		t.Fatal("后继任务不可接")
	}
	// 放弃（cust 880 子命令 4，GiveUp=1）。
	if !k.DispatchCommand("player", 880, p, NullPersistID, VarList{int32(4), int32(9002)}) {
		t.Fatal("放弃命令未命中")
	}
	if questRowOf(k, p, 9002) != nil {
		t.Fatal("放弃未清行")
	}
}

// TestQuestCriteriaEngine 条件引擎：事件 → 子模板类型路由
//（NPC 对话 InterNpc 键匹配 / 杀怪 616 面）。
func TestQuestCriteriaEngine(t *testing.T) {
	k, _ := assemble4(t)
	qm := module2[*QuestModule](t, k, "QuestModule")
	qcm := module2[*QuestCriteriaModule](t, k, "QuestCriteriaModule")
	p, _ := k.CreateObject("player")
	_ = k.SetInt(p, "Level", 10)

	// 任务：对话子模板（InterNpc=npc_master）。
	qm.Quests["9101"] = map[string]string{"ID": "9101", "Name": "拜师", "MinLevel": "1",
		"CriteriaList": "9101", "AwardID": "9901"}
	qm.Criteria["9101"] = map[string]string{"ID": "9101", "CriteriaType": "4", "InterNpc": "npc_master"}
	qm.AcceptQuest(k, p, 9101)

	// CommonNpc OnSelect（self=NPC、from=玩家——官方方向）：npc_master 对话 → 完成。
	npcSelf, _ := k.CreateObject("player")
	k.RunEvent("CommonNpc", "OnSelect", npcSelf, p, VarList{"npc_master"})
	cr := k.GetGameObj(p).Record("QuestCriteriaRecord")
	if cr.Row(0).Int(qcState) != CSDone {
		t.Fatal("对话引擎未推进")
	}
	// 键不匹配的 NPC 不推进新任务。
	qm.Quests["9102"] = map[string]string{"ID": "9102", "Name": "杂役", "MinLevel": "1",
		"CriteriaList": "9102", "AwardID": "9901"}
	qm.Criteria["9102"] = map[string]string{"ID": "9102", "CriteriaType": "4", "InterNpc": "npc_other"}
	qm.AcceptQuest(k, p, 9102)
	k.RunEvent("CommonNpc", "OnSelect", npcSelf, p, VarList{"npc_master"})
	cr = k.GetGameObj(p).Record("QuestCriteriaRecord")
	for i := 0; i < cr.Rows(); i++ {
		if cr.Row(i).Int(qcID) == 9102 && cr.Row(i).Int(qcState) == CSDone {
			t.Fatal("InterNpc 键未生效（不匹配的 NPC 也推进了）")
		}
	}
	// 杀怪面：AttackNpc 616。
	qm.Quests["9103"] = map[string]string{"ID": "9103", "Name": "清剿", "MinLevel": "1",
		"CriteriaList": "9103", "AwardID": "9901"}
	qm.Criteria["9103"] = map[string]string{"ID": "9103", "CriteriaType": "1"}
	if !qm.AcceptQuest(k, p, 9103) {
		t.Fatalf("9103 接取失败 sys=%v", qm.SysMsgs())
	}
	npc, _ := k.CreateObject("player") // 占位攻击者（self=NPC、from=玩家）
	k.DispatchIntCommand("AttackNpc", 616, npc, p, VarList{"monster_001"})
	cr = k.GetGameObj(p).Record("QuestCriteriaRecord")
	found := false
	for i := 0; i < cr.Rows(); i++ {
		if cr.Row(i).Int(qcID) == 9103 && cr.Row(i).Int(qcCur) == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("杀怪引擎未推进")
	}
	// 引擎流水。
	if len(qcm.Ops()) < 2 {
		t.Fatal("引擎流水缺失")
	}
}

// TestAchievementLifecycle 成就闭环：登记 → 进度推进 → 解锁（点数累加）→
// 追踪/领奖 → Lua 面（强制完成/重置）。
func TestAchievementLifecycle(t *testing.T) {
	k, _ := assemble4(t)
	am := module2[*AchievementModule](t, k, "AchievementModule")
	p, _ := k.CreateObject("player")
	k.RunEvent("player", "OnRecover", p, NullPersistID, nil)

	// 合成成就：双子模板（92 需 5 次、93 需 1 次），10 点。
	am.Achievements["8001"] = map[string]string{"ID": "8001", "CriteriaList": "8001,8002", "Points": "10"}
	am.Criteria["8001"] = map[string]string{"ID": "8001", "MaxCount": "5"}
	am.Criteria["8002"] = map[string]string{"ID": "8002", "MaxCount": "1"}

	if !am.RegisterAchievement(k, p, 8001) {
		t.Fatal("成就登记失败")
	}
	cr := k.GetGameObj(p).Record("AchievementCriteriaProgress")
	if cr.Rows() != 2 || cr.Row(0).Int(acMax) != 5 {
		t.Fatalf("进度行 %+v", cr.Row(0))
	}
	// 推进 5 次（cmd 716）→ 首子模板完成。
	for i := 0; i < 5; i++ {
		k.DispatchIntCommand("player", 716, p, NullPersistID, VarList{int32(8001), int32(1)})
	}
	if cr.Row(0).Int(acFlag) != CSDone {
		t.Fatal("首子模板未完成")
	}
	if len(am.Unlocked()) != 0 {
		t.Fatal("未全完成即解锁")
	}
	// 次子模板 → 解锁。
	k.DispatchIntCommand("player", 716, p, NullPersistID, VarList{int32(8002), int32(1)})
	if len(am.Unlocked()) != 1 || am.Unlocked()[0] != 8001 {
		t.Fatalf("解锁流水 %v", am.Unlocked())
	}
	if got := k.GetGameObj(p).GetInt("AchievePoint"); got != 10 {
		t.Fatalf("AchievePoint=%d 期望 10", got)
	}
	done := k.GetGameObj(p).Record("AchievementCompleted")
	if done.Rows() != 1 || done.Row(0).Int(amID) != 8001 {
		t.Fatal("完成表未登记")
	}
	// 重复推进不重复解锁。
	k.DispatchIntCommand("player", 716, p, NullPersistID, VarList{int32(8001), int32(1)})
	if len(am.Unlocked()) != 1 {
		t.Fatal("重复解锁")
	}
	// 追踪 + 领奖（cust 340）。
	k.DispatchCommand("player", 340, p, NullPersistID, VarList{int32(1), int32(8001)})
	if k.GetGameObj(p).Record("AchievementTrace").Rows() != 1 {
		t.Fatal("追踪未登记")
	}
	k.DispatchCommand("player", 340, p, NullPersistID, VarList{int32(2), int32(8001)})
	if done.Row(0).Int(amClaimed) != 1 {
		t.Fatal("领奖标识未置位")
	}
	// Lua 面：强制完成 + 重载 + 重置。
	am.NxCompleteAchievement(k, p, 8002)
	if len(am.Unlocked()) != 2 {
		t.Fatal("nx_complete 未生效")
	}
	am.NxReloadAchieveResource()
	if am.Reloads() != 1 {
		t.Fatal("重载计数异常")
	}
	am.NxResetAchievement(k, p)
	if k.GetGameObj(p).Record("AchievementCompleted").Rows() != 0 || k.GetGameObj(p).GetInt("AchievePoint") != 0 {
		t.Fatal("重置未清面")
	}
}
