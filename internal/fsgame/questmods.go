package fsgame

import (
	"fmt"
	"strings"
)

// 四期：任务域三模块（docs/official/FsGame全解与Go复刻.md §8 任务与成就）。
// 官方源码树：fsgame\questmodule\。

// 任务状态（QuestRecord 列 1 byte）。
const (
	QSActive    int32 = iota // 0 进行中
	QSCanSubmit              // 1 可提交（全部子模板完成）
)

// 子模板状态（QuestCriteriaRecord 列 1 byte）。
const (
	CSDoing int32 = iota // 0 未完成
	CSDone               // 1 完成
)

// QuestRecord 列序（struct/quest/quest.xml 定案）。
const (
	qrID     = iota // 0 任务编号(word)
	qrState         // 1 任务状态(byte)
	qrTrack         // 2 追踪标志(byte)
	qrStep          // 3 当前步骤子模板(word)
)

// QuestCriteriaRecord 列序。
const (
	qcID     = iota // 0 任务子模板编号(word)
	qcState         // 1 子模板状态(byte)
	qcCur           // 2 当前进度(word)
	qcMax           // 3 总进度(word)
	qcParent        // 4 父功能编号(word)
	qcParentType    // 5 父功能类型(byte)
)

// 任务子模板类型（ini/Quest/QuestCriteria.xml CriteriaType 值域分布定案）。
const (
	CriteriaKillNpc    = "1"  // 杀怪（1292 行，最众）
	CriteriaTalkNpc    = "4"  // NPC 对话（InterNpc 串）
	CriteriaCollect    = "5"  // 收集物品（ToolBox 钩子）
	CriteriaUseItem    = "7"  // 使用物品
	CriteriaReachArea  = "18" // 到达区域（CustomTrackPos 坐标串）
)

// AchievementCriteriaProgress 列序（struct/player/Achievement.xml 定案）。
const (
	acID       = iota // 0 成就子模板编号(dword)
	acMain            // 1 归属主成就编号(dword)
	acFlag            // 2 进度标识(byte)
	acCur             // 3 当前进度(dword)
	acMax             // 4 需求进度(dword)
	acAch             // 5 成就编号(dword)
)

// AchievementCompleted 列序。
const (
	amID      = iota // 0 成就编号(dword)
	amClaimed        // 1 领取奖励标识(byte)
	amTime           // 2 完成时间(string)
)

// ---- QuestModule ----

// QuestModule 复刻官方 questmodule（Init @0x99C820，玩法表 Init 序 79；
// 符号 753，8 心跳）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddClassCallback ("NpcDropBox","OnCreateClass",@0x99D8A0)
//	AddEventCallback ("player","OnReady",@0x99DDD0) / ("player","OnEntry",
//	    @0x99D9A0) / ("player","OnRecover",@0x99D8F0) / ("player","OnContinue",
//	    @0x99E0C0) / ("player","OnStore",@0x99E110)
//	AddEventCallback ("CommonNpc","OnLoad",@0x99E230) / ("CommonNpc","OnSelect",
//	    @0x9A34B0 OnNpcSelect)
//	AddIntCommandHook ("player",22,@0x9A3B50) OnMenuSelect
//	AddIntCustomHook ("player",880=0x370,@0x99E2C0) OnCustomQuestMsg
//	DeclareRecHook（QuestCriteriaRecord）→ OnCriteriaRecChanged@0x9AC3E0
//	DeclareRecHook（QuestRecord）→ OnQuestRecChanged@0x9AC940
//	DeclareCritical ×2：C_OnLevelChanged@0x9ACE20 / C_OnForceChanged@0x9AD000
//	DeclareHeartBeat ×4：HB_TimeLimit@0x999DE0 / HB_CheckTime@0x99A5C0 /
//	    HB_RepeatQuest@0x99A270 / HB_RefreshQuestEffect@0x99A730
//
// OnCustomQuestMsg（0x99E2C0）子命令分派（cmp 链定案）：
//
//	1 AcceptQuest / 2 SubmitQuest / 3 CanShareQuest→ShareQuest /
//	4 CanGiveUpQuest→OnGiveUpQuest / 5 IsQuestHaving→SetQuestTrackStatus /
//	8 SubmitQuest（AutoQuestFlag 面）/ 9 ReceiveShareQuest / 10 RefuseShareQuest
//
// 配置（官方经 StaticDataQuery 装载；Go 面 LoadResource 直读）：
//
//	ini/Quest/Quest.xml（任务表：MinLevel/preQuestList/postQuestList/
//	  CriteriaList/AcceptNpc/SubmitNpc/GiveUp/Share/TimeType/AwardID…）
//	ini/Quest/QuestCriteria.xml（子模板表：CriteriaType/InterNpc/
//	  CustomTrackPos/MaxCount…）
//	ini/Quest/Award.xml（奖励表：Exp/Money/DefaultItemList）
type QuestModule struct {
	BaseModule

	// Quests 任务表（ID → 行）；Criteria 子模板表；Awards 奖励表。
	Quests   map[string]map[string]string
	Criteria map[string]map[string]string
	Awards   map[string]map[string]string

	// AwardOut 奖励出口（Exp/Money/ItemString；官方经 AwardModule→
	// Capital/Container 落账，Go 面 sync 层挂接）。
	AwardOut func(player PersistID, exp, money int32, items string)

	// npcAccept/npcSubmit NPC → 任务号索引（九阴任务侧 AcceptNpc/
	// SubmitNpc 反查；OnNpcSelect 分发面）。
	npcAccept map[string][]string
	npcSubmit map[string][]string

	// Texts 任务文案（ini/Text/TaskText.xml：title_<id>/context_<id>/
	// CanAcceptMenu_<id>/story_<id>_<n> 等串键，npcgen 从客户端
	// stringtask_normal/special 提取）。
	Texts map[string]string

	// sysMsgs 系统信息观测（sys_quest_* 模板号）。
	sysMsgs []string
	// ops 命令流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *QuestModule) Name() string { return "QuestModule" }

// Init 注册期（对齐官方 0x99C820）。
func (m *QuestModule) Init(k *Kernel) error {
	m.Quests = make(map[string]map[string]string)
	m.Criteria = make(map[string]map[string]string)
	m.Awards = make(map[string]map[string]string)
	b := k.Bus()
	b.OnClassEvent("NpcDropBox", "OnCreateClass", func(k *Kernel, class string) {})
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnContinue", m.onContinue)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnEvent("CommonNpc", "OnLoad", func(k *Kernel, self, from PersistID, args VarList) {})
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	b.OnCommand("player", 880, m.onCustomQuestMsg)
	b.RecHook("QuestCriteriaRecord", m.onCriteriaRecChanged)
	b.RecHook("QuestRecord", m.onQuestRecChanged)
	b.Critical("player", "Level", m.onLevelChanged)
	b.Critical("player", "Force", m.onForceChanged)
	b.HeartBeat("QuestModule::HB_TimeLimit", m.hbTimeLimit)
	b.HeartBeat("QuestModule::HB_CheckTime", m.hbCheckTime)
	b.HeartBeat("QuestModule::HB_RepeatQuest", m.hbRepeatQuest)
	b.HeartBeat("QuestModule::HB_RefreshQuestEffect", m.hbRefreshQuestEffect)
	return nil
}

// LoadResource 任务域配置装载——九阴任务域（BGserverCJ 侧，npcgen 生成的
// ini/Quest/Task 树 + SubFunction 接线 + 步骤域 + TaskItem），经
// questnineyin.go 合成引擎词汇三表与 NPC 面；黑金三表（Quest/
// QuestCriteria/Award.xml）已移 heijin_backup 留作对照。
func (m *QuestModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: QuestModule.LoadResource: ResourcePath 未设置")
	}
	data, err := loadNineyinQuests(k.ResourcePath)
	if err != nil {
		return fmt.Errorf("fsgame: QuestModule::LoadResource: %w", err)
	}
	m.Quests, m.Criteria, m.Awards = data.Quests, data.Criteria, data.Awards
	m.npcAccept, m.npcSubmit = data.NpcAccept, data.NpcSubmit
	if err := m.loadTexts(k); err != nil {
		return err
	}
	s := data.Stats
	k.Tracef("QuestModule: LoadResource 九阴任务 %d（task002 补 %d）/ 合成子模板 %d / 可推进任务 %d / 奖励行 %d / 文案 %d / 未映射类型 %d / 缺步骤 %d",
		s.Tasks, s.Task002Only, s.Synthesized, s.WithSteps, s.AwardsBuilt, len(m.Texts), s.SkippedType, s.MissingStep)
	return nil
}

// loadTexts 任务文案装载（ini/Text/TaskText.xml）。
func (m *QuestModule) loadTexts(k *Kernel) error {
	rows, err := loadPropRows(k.ResourcePath, "ini/Text/TaskText.xml")
	if err != nil {
		return fmt.Errorf("fsgame: QuestModule::LoadResource: %w", err)
	}
	m.Texts = make(map[string]string, len(rows))
	for _, row := range rows {
		if row["ID"] != "" {
			m.Texts[row["ID"]] = row["Text"]
		}
	}
	return nil
}

// TaskText 任务文案查询（串键 → 文本；缺省空）。
func (m *QuestModule) TaskText(key string) string { return m.Texts[key] }

// onCustomQuestMsg 官方 @0x99E2C0（cust 880=0x370）：任务客户端消息多路
//（args: 子命令, 任务编号[, …]）。
func (m *QuestModule) onCustomQuestMsg(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	sub := args.Int(0)
	qid := args.Int(1)
	switch sub {
	case 1: // AcceptQuest
		m.AcceptQuest(k, self, qid)
	case 2, 8: // SubmitQuest（8=AutoQuestFlag 自动提交面）
		m.SubmitQuest(k, self, qid)
	case 3: // CanShareQuest → ShareQuest（Share=1 配置门）
		if cfg := m.Quests[fmt.Sprint(qid)]; cfg != nil && atoi32(cfg["Share"]) != 0 {
			m.ops = append(m.ops, "share")
		}
	case 4: // CanGiveUpQuest → OnGiveUpQuest（GiveUp=1 配置门）
		if cfg := m.Quests[fmt.Sprint(qid)]; cfg != nil && atoi32(cfg["GiveUp"]) != 0 {
			m.GiveUpQuest(k, self, qid)
		}
	case 5: // IsQuestHaving → SetQuestTrackStatus
		m.SetQuestTrackStatus(k, self, qid, args.Int(2))
	case 9: // ReceiveShareQuest
		m.AcceptQuest(k, self, qid)
	case 10: // RefuseShareQuest
		m.ops = append(m.ops, "refuse_share")
	default:
		k.Tracef("QuestModule: subcmd %d", sub)
	}
}

// AcceptQuest 官方 AcceptQuest：接任务闭环——MinLevel 门 / 前置任务完成位集
//（preQuestList）/ 在办去重 / 承接表容量（25）→ QuestRecord 追加（状态=进行中）+
// QuestCriteriaRecord 按 CriteriaList 初始化（总进度取子模板 MaxCount，缺省 1）。
func (m *QuestModule) AcceptQuest(k *Kernel, self PersistID, questID int32) bool {
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	cfg := m.Quests[fmt.Sprint(questID)]
	if cfg == nil {
		m.sys("sys_quest_0002") // 任务不存在
		return false
	}
	// 等级门。
	if minLv := atoi32(cfg["MinLevel"]); minLv > 0 && o.GetInt("Level") < minLv {
		m.sys("sys_quest_0003")
		return false
	}
	// 前置门：preQuestList（逗号串）全部已完成（QuestRewarded 位集）。
	if pre := cfg["preQuestList"]; pre != "" {
		for _, p := range strings.Split(pre, ",") {
			if !m.isRewarded(k, self, atoi32(strings.TrimSpace(p))) {
				m.sys("sys_quest_0004")
				return false
			}
		}
	}
	qr := o.Record("QuestRecord")
	if qr == nil {
		return false
	}
	if m.questRow(k, self, questID) >= 0 {
		m.sys("sys_quest_0005") // 已承接
		return false
	}
	if qr.Rows() >= qr.Def().MaxRows {
		m.sys("sys_quest_0006") // 任务栏满
		return false
	}
	qr.AppendRow(VarList{questID, QSActive, int32(0), int32(0)})
	// 子模板进度初始化。
	if cr := o.Record("QuestCriteriaRecord"); cr != nil {
		for _, cid := range strings.Split(cfg["CriteriaList"], ",") {
			cid = strings.TrimSpace(cid)
			if cid == "" {
				continue
			}
			ccfg := m.Criteria[cid]
			max := int32(1)
			if ccfg != nil {
				if mc := atoi32(ccfg["MaxCount"]); mc > 0 {
					max = mc
				}
			}
			cr.AppendRow(VarList{atoi32(cid), CSDoing, int32(0), max, questID, int32(0)})
		}
	}
	// 时限任务登记（TimeType≠0 → QuestTimeLimitRecord）。
	if tt := atoi32(cfg["TimeType"]); tt != 0 {
		if tr := o.Record("QuestTimeLimitRecord"); tr != nil {
			tr.AppendRow(VarList{questID, unixNow(), unixNow() + 86400})
		}
	}
	m.ops = append(m.ops, fmt.Sprintf("accept:%d", questID))
	k.Tracef("QuestModule: accept %d (%s) player=%v", questID, cfg["Name"], self.ID())
	return true
}

// SubmitQuest 官方 SubmitQuest：交任务——状态=可提交 → 奖励落账
//（AwardID → Award.xml 行：Exp/Money/DefaultItemList）→ 完成位集置位 →
// 承接/进度行清除 → postQuestList 解锁面。
func (m *QuestModule) SubmitQuest(k *Kernel, self PersistID, questID int32) bool {
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	i := m.questRow(k, self, questID)
	if i < 0 {
		return false
	}
	qr := o.Record("QuestRecord")
	if qr.Row(i).Int(qrState) != QSCanSubmit {
		m.sys("sys_quest_0009") // 未完成不可提交
		return false
	}
	cfg := m.Quests[fmt.Sprint(questID)]
	// 奖励（LibertySubmit=1 任意 NPC 可交；否则 SubmitNpc 门在 OnNpcSelect 面）。
	if cfg != nil {
		if a := m.Awards[cfg["AwardID"]]; a != nil {
			exp, money := atoi32(a["Exp"]), atoi32(a["Money"])
			if m.AwardOut != nil {
				m.AwardOut(self, exp, money, a["DefaultItemList"])
			}
			k.Tracef("QuestModule: award %d exp=%d money=%d items=%s",
				questID, exp, money, a["DefaultItemList"])
		}
	}
	qr.DeleteRow(i)
	m.clearCriteria(k, self, questID)
	m.setRewarded(k, self, questID)
	m.ops = append(m.ops, fmt.Sprintf("submit:%d", questID))
	return true
}

// GiveUpQuest 官方 OnGiveUpQuest：放弃（清行；时限任务同步清）。
func (m *QuestModule) GiveUpQuest(k *Kernel, self PersistID, questID int32) bool {
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	i := m.questRow(k, self, questID)
	if i < 0 {
		return false
	}
	o.Record("QuestRecord").DeleteRow(i)
	m.clearCriteria(k, self, questID)
	if tr := o.Record("QuestTimeLimitRecord"); tr != nil {
		for j := 0; j < tr.Rows(); j++ {
			if tr.Row(j).Int(0) == questID {
				tr.DeleteRow(j)
				break
			}
		}
	}
	m.ops = append(m.ops, fmt.Sprintf("giveup:%d", questID))
	return true
}

// SetQuestTrackStatus 官方：追踪标志（QuestRecord 列 2）。
func (m *QuestModule) SetQuestTrackStatus(k *Kernel, self PersistID, questID, track int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	qr := o.Record("QuestRecord")
	if i := m.questRow(k, self, questID); i >= 0 {
		row := qr.Row(i)
		row[qrTrack] = track
		qr.ModifyRow(i, row)
	}
}

// IsQuestHaving 官方：任务是否在办。
func (m *QuestModule) IsQuestHaving(k *Kernel, self PersistID, questID int32) bool {
	return m.questRow(k, self, questID) >= 0
}

// UpdateCriteria 子模板进度推进（QuestCriteriaModule 引擎的落点）：
// absolute=true 时直接设值，否则累加；达总进度 → 子模板完成
//（子模板状态=1）→ 全完成则任务转可提交（OnCriteriaRecChanged 面 +
// COMPLETEQUEST effect + sys_quest_0031）。
func (m *QuestModule) UpdateCriteria(k *Kernel, self PersistID, criteriaID, count int32, absolute bool) bool {
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	cr := o.Record("QuestCriteriaRecord")
	if cr == nil {
		return false
	}
	for i := 0; i < cr.Rows(); i++ {
		row := cr.Row(i)
		if row.Int(qcID) != criteriaID || row.Int(qcState) == CSDone {
			continue
		}
		cur := row.Int(qcCur)
		if absolute {
			cur = count
		} else {
			cur += count
		}
		if cur > row.Int(qcMax) {
			cur = row.Int(qcMax)
		}
		row[qcCur] = cur
		if cur >= row.Int(qcMax) {
			row[qcState] = CSDone
		}
		cr.ModifyRow(i, row)
		if row.Int(qcState) == CSDone {
			m.checkQuestComplete(k, self, row.Int(qcParent))
		}
		return true
	}
	return false
}

// checkQuestComplete 任务下全部子模板完成 → 状态=可提交。
func (m *QuestModule) checkQuestComplete(k *Kernel, self PersistID, questID int32) {
	o := k.GetGameObj(self)
	cr := o.Record("QuestCriteriaRecord")
	allDone := true
	found := false
	for i := 0; i < cr.Rows(); i++ {
		row := cr.Row(i)
		if row.Int(qcParent) != questID {
			continue
		}
		found = true
		if row.Int(qcState) != CSDone {
			allDone = false
			break
		}
	}
	if !found || !allDone {
		return
	}
	if qr := o.Record("QuestRecord"); qr != nil {
		if i := m.questRow(k, self, questID); i >= 0 {
			row := qr.Row(i)
			row[qrState] = QSCanSubmit
			qr.ModifyRow(i, row)
		}
	}
	m.sys("sys_quest_0031")
	m.ops = append(m.ops, fmt.Sprintf("complete:%d", questID))
	k.Tracef("QuestModule: COMPLETEQUEST %d player=%v", questID, self.ID())
}

// FailCriteria 子模板失败（官方 OnCommandCriteriaFailed 926 面）→ 任务失败清理。
func (m *QuestModule) FailCriteria(k *Kernel, self PersistID, questID int32) {
	m.GiveUpQuest(k, self, questID)
	m.sys("sys_quest_0032")
}

// questRow 承接表行定位。
func (m *QuestModule) questRow(k *Kernel, self PersistID, questID int32) int {
	o := k.GetGameObj(self)
	qr := o.Record("QuestRecord")
	if qr == nil {
		return -1
	}
	for i := 0; i < qr.Rows(); i++ {
		if qr.Row(i).Int(qrID) == questID {
			return i
		}
	}
	return -1
}

// clearCriteria 任务进度行清除。
func (m *QuestModule) clearCriteria(k *Kernel, self PersistID, questID int32) {
	o := k.GetGameObj(self)
	cr := o.Record("QuestCriteriaRecord")
	if cr == nil {
		return
	}
	for i := cr.Rows() - 1; i >= 0; i-- {
		if cr.Row(i).Int(qcParent) == questID {
			cr.DeleteRow(i)
		}
	}
}

// QuestRewarded 位集（word 编号 → dword 行×位）。
func (m *QuestModule) isRewarded(k *Kernel, self PersistID, questID int32) bool {
	o := k.GetGameObj(self)
	r := o.Record("QuestRewarded")
	if r == nil {
		return false
	}
	rowIdx, bit := int(questID)/32, uint32(questID)%32
	if rowIdx >= r.Rows() {
		return false
	}
	return r.Row(rowIdx).Int(0)&(int32(1)<<bit) != 0
}

func (m *QuestModule) setRewarded(k *Kernel, self PersistID, questID int32) {
	o := k.GetGameObj(self)
	r := o.Record("QuestRewarded")
	if r == nil {
		return
	}
	rowIdx, bit := int(questID)/32, uint32(questID)%32
	for r.Rows() <= rowIdx {
		if r.Rows() >= r.Def().MaxRows {
			return
		}
		r.AppendRow(VarList{int32(0)})
	}
	row := r.Row(rowIdx)
	row[0] = row.Int(0) | int32(1)<<bit
	r.ModifyRow(rowIdx, row)
}

// onNpcSelect 官方 @0x9A34B0：NPC 对话——对话类子模板（CriteriaType=4，
// InterNpc 匹配）推进；SubmitNpc 匹配可交任务面。Go 面计算该 NPC 的
// 可接/可交清单（九阴任务侧 AcceptNpc/SubmitNpc 索引），供对话菜单
// （cmd 22）分发。
func (m *QuestModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {
	npcID := args.String(0)
	acc := m.AcceptableQuests(k, from, npcID)
	sub := m.SubmittableQuests(k, from, npcID)
	m.ops = append(m.ops, fmt.Sprintf("npc:%s accept=%v submit=%v", npcID, acc, sub))
	k.Tracef("QuestModule: OnNpcSelect %s 可接 %d / 可交 %d", npcID, len(acc), len(sub))
}

// AcceptableQuests NPC 对话面上可接任务清单：AcceptNpc 匹配 + 等级门 +
// 前置完成位集 + 在办去重。
func (m *QuestModule) AcceptableQuests(k *Kernel, player PersistID, npcID string) []int32 {
	o := k.GetGameObj(player)
	if o == nil {
		return nil
	}
	var out []int32
	for _, tid := range m.npcAccept[npcID] {
		cfg := m.Quests[tid]
		if cfg == nil {
			continue
		}
		if minLv := atoi32(cfg["MinLevel"]); minLv > 0 && o.GetInt("Level") < minLv {
			continue
		}
		if !m.prereqMet(k, player, cfg) || m.questRow(k, player, atoi32(tid)) >= 0 {
			continue
		}
		out = append(out, atoi32(tid))
	}
	return out
}

// SubmittableQuests NPC 对话面上可交任务清单：在办 + 状态=可提交 +
// SubmitNpc 匹配。
func (m *QuestModule) SubmittableQuests(k *Kernel, player PersistID, npcID string) []int32 {
	o := k.GetGameObj(player)
	if o == nil {
		return nil
	}
	qr := o.Record("QuestRecord")
	if qr == nil {
		return nil
	}
	submitSet := make(map[string]struct{}, len(m.npcSubmit[npcID]))
	for _, tid := range m.npcSubmit[npcID] {
		submitSet[tid] = struct{}{}
	}
	var out []int32
	for i := 0; i < qr.Rows(); i++ {
		tid := qr.Row(i).Int(qrID)
		if _, ok := submitSet[fmt.Sprint(tid)]; !ok {
			continue
		}
		if qr.Row(i).Int(qrState) == QSCanSubmit {
			out = append(out, tid)
		}
	}
	return out
}

// prereqMet 前置任务完成位集校验（AcceptQuest 同款门的只读版）。
func (m *QuestModule) prereqMet(k *Kernel, player PersistID, cfg map[string]string) bool {
	for _, p := range strings.Split(cfg["preQuestList"], ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !m.isRewarded(k, player, atoi32(p)) {
			return false
		}
	}
	return true
}

// onMenuSelect 官方 @0x9A3B50（命令 22）：菜单（对话选项 → 接/交任务入口）。
func (m *QuestModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {}

// onReady 官方 @0x99DDD0：任务表同步下发。
func (m *QuestModule) onReady(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("QuestModule: OnReady player=%v", self.ID())
}

// onEntry 官方 @0x99D9A0。
func (m *QuestModule) onEntry(k *Kernel, self, from PersistID, args VarList) {}

// onRecover 官方 @0x99D8F0：回载（Quest_Fail_OnLoad_Rec 下线失败任务清理）。
func (m *QuestModule) onRecover(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if fr := o.Record("Quest_Fail_OnLoad_Rec"); fr != nil {
		for i := fr.Rows() - 1; i >= 0; i-- {
			m.GiveUpQuest(k, self, fr.Row(i).Int(0))
			fr.DeleteRow(i)
		}
	}
}

// onContinue 官方 @0x99E0C0。
func (m *QuestModule) onContinue(k *Kernel, self, from PersistID, args VarList) {}

// onStore 官方 @0x99E110。
func (m *QuestModule) onStore(k *Kernel, self, from PersistID, args VarList) {}

// OnCriteriaRecChanged 官方 @0x9AC3E0（RecHook）：进度表变更联动
//（quest_msg 下发 + LastObject 面记）。
func (m *QuestModule) onCriteriaRecChanged(k *Kernel, self PersistID, record string, op, row int32) {}

// OnQuestRecChanged 官方 @0x9AC940（RecHook）：承接表变更联动。
func (m *QuestModule) onQuestRecChanged(k *Kernel, self PersistID, record string, op, row int32) {}

// C_OnLevelChanged 官方 @0x9ACE20：升级 → 新任务可接面刷新。
func (m *QuestModule) onLevelChanged(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnForceChanged 官方 @0x9AD000：势力变更 → 阵营任务可用性刷新。
func (m *QuestModule) onForceChanged(k *Kernel, self PersistID, prop string, old Var) {}

// HB_TimeLimit 官方 @0x999DE0：时限任务到期 → 失败清理（QuestTimeLimitRecord）。
func (m *QuestModule) hbTimeLimit(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	tr := o.Record("QuestTimeLimitRecord")
	if tr == nil {
		return
	}
	for i := tr.Rows() - 1; i >= 0; i-- {
		if unixNow() >= tr.Row(i).Int64(2) {
			m.GiveUpQuest(k, self, tr.Row(i).Int(0))
		}
	}
}

// HB_CheckTime 官方 @0x99A5C0。
func (m *QuestModule) hbCheckTime(k *Kernel, self PersistID, beats int32) {}

// HB_RepeatQuest 官方 @0x99A270：可重复任务刷新（QuestRepeatRecord 刷新时间）。
func (m *QuestModule) hbRepeatQuest(k *Kernel, self PersistID, beats int32) {}

// HB_RefreshQuestEffect 官方 @0x99A730：任务效果刷新（effect 面）。
func (m *QuestModule) hbRefreshQuestEffect(k *Kernel, self PersistID, beats int32) {}

func (m *QuestModule) sys(tpl string) { m.sysMsgs = append(m.sysMsgs, tpl) }

// SysMsgs 系统信息流水（测试面）。
func (m *QuestModule) SysMsgs() []string { return m.sysMsgs }

// Ops 命令流水（测试面）。
func (m *QuestModule) Ops() []string { return m.ops }

// ---- QuestCriteriaModule ----

// QuestCriteriaModule 复刻官方 questcriteriamodule（Init @0x98B8A0，玩法表
// Init 序 80；符号 280）——任务条件引擎：把游戏事件翻译为子模板进度。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnReady",@0x98BFB0) / ("player","OnRecover",
//	    @0x98BF80) / ("player","OnMapChanged",@0x98BFE0)
//	AddEventCallback ("ToolBox","OnAfterAdd",@0x98C260) / ("ToolBox","OnRemove",
//	    @0x98C410)——物品收集/消耗类
//	AddEventCallback ("BufferContainer","OnAfterAdd",@0x98C0E0)
//	AddIntCommandHook ("AttackNpc"/"BattleCarrier",616,@0x98C800
//	    OnCommandNpcBeKilled) / ("player",616,@0x98CE90 OnCommandPlayerBeKilled)
//	AddIntCommandHook ("player",934,@0x9892D0 OnCommandUpdateCriteria) /
//	    ("player",926,@0x98C5C0 OnCommandCriteriaFailed) / ("player",928,
//	    @0x98C690 OnCommandCheckCriteria) / ("player",22,@0x990280 OnMenuSelect)
//	AddEventCallback ("CommonNpc","OnSelect",@0x98FF20 OnNpcSelect)——对话类
//	DeclareHeartBeat ("QuestCriteriaModule::HB_Convoy",@0x98FAA0)——护送
type QuestCriteriaModule struct {
	BaseModule

	// engine 目标模块缓存（QuestModule 协作）。
	ops []string
}

// Name 模块名。
func (m *QuestCriteriaModule) Name() string { return "QuestCriteriaModule" }

// Init 注册期（对齐官方 0x98B8A0）。
func (m *QuestCriteriaModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnMapChanged", m.onMapChanged)
	b.OnEvent("ToolBox", "OnAfterAdd", m.onToolBoxAfterAdd)
	b.OnEvent("ToolBox", "OnRemove", m.onToolBoxRemove)
	b.OnEvent("BufferContainer", "OnAfterAdd", m.onBufferContainerAfterAdd)
	b.OnIntCommand("AttackNpc", 616, m.onCommandNpcBeKilled)
	b.OnIntCommand("BattleCarrier", 616, m.onCommandNpcBeKilled)
	b.OnIntCommand("player", 616, m.onCommandPlayerBeKilled)
	b.OnIntCommand("player", 934, m.onCommandUpdateCriteria)
	b.OnIntCommand("player", 926, m.onCommandCriteriaFailed)
	b.OnIntCommand("player", 928, m.onCommandCheckCriteria)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.HeartBeat("QuestCriteriaModule::HB_Convoy", m.hbConvoy)
	return nil
}

// questModule 取 QuestModule（协作面）。
func (m *QuestCriteriaModule) questModule(k *Kernel) *QuestModule {
	qm, _ := k.GetLogicModule("QuestModule").(*QuestModule)
	return qm
}

// advanceByType 按子模板类型推进在办任务下同型子模板。
// 剧情段（StepType=story）按 Order 顺序播放：同任务一次只推最前未完成段
//（其余段 InterNpc 相同，不门控会一次对话播完全部剧情）。
func (m *QuestCriteriaModule) advanceByType(k *Kernel, self PersistID, criteriaType string, count int32, key string) {
	qm := m.questModule(k)
	o := k.GetGameObj(self)
	if qm == nil || o == nil {
		return
	}
	cr := o.Record("QuestCriteriaRecord")
	if cr == nil {
		return
	}
	// 收集匹配行 + 求同任务剧情段的最小未完成 Order。
	type candidate struct {
		row   int
		order int32
		story bool
	}
	var matched []candidate
	minStoryOrder := map[int32]int32{} // qcParent → 最小 Order
	for i := 0; i < cr.Rows(); i++ {
		row := cr.Row(i)
		if row.Int(qcState) == CSDone {
			continue
		}
		ccfg := qm.Criteria[fmt.Sprint(row.Int(qcID))]
		if ccfg == nil || ccfg["CriteriaType"] != criteriaType {
			continue
		}
		// 键匹配门（InterNpc/ItemID 等子模板键；空键=通配）。
		if key != "" {
			kk := ccfg["InterNpc"]
			if kk == "" {
				kk = ccfg["ItemID"]
			}
			if kk != "" && kk != key {
				continue
			}
		}
		story := ccfg["StepType"] == "story"
		order := atoi32(ccfg["Order"])
		matched = append(matched, candidate{row: i, order: order, story: story})
		if story {
			parent := row.Int(qcParent)
			if mo, ok := minStoryOrder[parent]; !ok || order < mo {
				minStoryOrder[parent] = order
			}
		}
	}
	for _, c := range matched {
		if c.story && c.order != minStoryOrder[cr.Row(c.row).Int(qcParent)] {
			continue
		}
		cid := cr.Row(c.row).Int(qcID)
		qm.UpdateCriteria(k, self, cid, count, false)
		m.ops = append(m.ops, fmt.Sprintf("advance:%d(type %s)", cid, criteriaType))
	}
}

// onCommandNpcBeKilled 官方 @0x98C800（616）：NPC 被杀 → 杀怪类（type 1）推进。
// args: 被杀者串/击杀者。
func (m *QuestCriteriaModule) onCommandNpcBeKilled(k *Kernel, self, from PersistID, args VarList) {
	m.advanceByType(k, from, CriteriaKillNpc, 1, args.String(0))
}

// onCommandPlayerBeKilled 官方 @0x98CE90（616）：玩家被杀（PK 类任务面）。
func (m *QuestCriteriaModule) onCommandPlayerBeKilled(k *Kernel, self, from PersistID, args VarList) {
	m.advanceByType(k, from, CriteriaKillNpc, 1, "")
}

// onCommandUpdateCriteria 官方 @0x9892D0（命令 934=0x3A6）：通用进度推进
//（args: 子模板编号, 数量[, absolute]）。
func (m *QuestCriteriaModule) onCommandUpdateCriteria(k *Kernel, self, from PersistID, args VarList) {
	if qm := m.questModule(k); qm != nil {
		qm.UpdateCriteria(k, self, args.Int(0), args.Int(1), args.Int(2) != 0)
	}
}

// onCommandCriteriaFailed 官方 @0x98C5C0（命令 926=0x39E）：失败
//（args: 任务编号）。
func (m *QuestCriteriaModule) onCommandCriteriaFailed(k *Kernel, self, from PersistID, args VarList) {
	if qm := m.questModule(k); qm != nil {
		qm.FailCriteria(k, self, args.Int(0))
	}
}

// onCommandCheckCriteria 官方 @0x98C690（命令 928=0x3A0）：完成度查询下发。
func (m *QuestCriteriaModule) onCommandCheckCriteria(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "check")
}

// onMenuSelect 官方 @0x990280（命令 22）。
func (m *QuestCriteriaModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {}

// onNpcSelect 官方 @0x98FF20：NPC 对话 → 对话类（type 4，InterNpc 键）推进。
func (m *QuestCriteriaModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {
	m.advanceByType(k, from, CriteriaTalkNpc, 1, args.String(0))
}

// onToolBoxAfterAdd 官方 @0x98C260：物品入手 → 收集类（type 5）推进。
func (m *QuestCriteriaModule) onToolBoxAfterAdd(k *Kernel, self, from PersistID, args VarList) {
	m.advanceByType(k, from, CriteriaCollect, 1, args.String(0))
}

// onToolBoxRemove 官方 @0x98C410：物品消耗 → 收集类回退（-1）。
func (m *QuestCriteriaModule) onToolBoxRemove(k *Kernel, self, from PersistID, args VarList) {
	m.advanceByType(k, from, CriteriaCollect, -1, args.String(0))
}

// onBufferContainerAfterAdd 官方 @0x98C0E0：状态入手（buff 类条件）。
func (m *QuestCriteriaModule) onBufferContainerAfterAdd(k *Kernel, self, from PersistID, args VarList) {}

// onReady 官方 @0x98BFB0。
func (m *QuestCriteriaModule) onReady(k *Kernel, self, from PersistID, args VarList) {}

// onRecover 官方 @0x98BF80。
func (m *QuestCriteriaModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}

// onMapChanged 官方 @0x98BFE0：换图 → 到达区域类（type 18，LinkArea 键）推进。
func (m *QuestCriteriaModule) onMapChanged(k *Kernel, self, from PersistID, args VarList) {
	m.advanceByType(k, from, CriteriaReachArea, 1, args.String(0))
}

// HB_Convoy 官方 @0x98FAA0：护送泵（Convoy_ProtectObj_Rec / ConvoyEndTime）。
func (m *QuestCriteriaModule) hbConvoy(k *Kernel, self PersistID, beats int32) {}

// Ops 引擎流水（测试面）。
func (m *QuestCriteriaModule) Ops() []string { return m.ops }

// ---- AchievementModule ----

// AchievementModule 复刻官方 achievementmodule（Init @0x9CB500，玩法表
// Init 序 81；符号 572，13 临界）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddIntCommandHook ("player",716,@0x9CC9B0) OnCommandUpdateCriteria
//	AddIntCommandHook ("AttackNpc"/"BattleCarrier",616,@0x9D1BB0) OnNpcBeKilled
//	AddIntCustomHook ("player",340=0x154,@0x9D2040) OnCustomAchievementMsg
//	AddEventCallback ("player","OnRecover",@0x9CF340) / ("player","OnEntry",
//	    @0x9CF550)
//	AddEventCallback ("ToolBox","OnAfterAdd",@0x9D06F0) / ("MountBox",
//	    "OnNoAdd",@0x9D0BE0) / ("MountPetBox","OnAfterAdd",@0x9D0A90) /
//	    ("EquipBox","OnAfterAdd",@0x9D0EA0)
//	DeclareCritical ×7：C_OnPropChanged@0x9D1240 / C_OnLevelChanged@0x9D13A0 /
//	    C_OnLifeJobLevelChanged@0x9D1500 / C_OnLifeJobExpChanged@0x9D1690 /
//	    C_OnEveRecActiveChanged@0x9D1820 / C_OnFightPowerChanged@0x9D1950 /
//	    C_OnGuildPositionChanged@0x9D1AB0
//	DeclareLuaExt ×4：nx_reload_achieve_resource@0x9CB100 /
//	    nx_reset_achievement@0x9CB340 / nx_clear_sole_achievement@0x9CB4B0 /
//	    nx_complete_achievement@0x9CB160（参数校验：object/int）
//
// 记录（struct/player/Achievement.xml，经 player includes 装载）：
// AchievementCriteriaProgress(6) / AchievementCompleted(3) /
// AchievementTotalProgress(4) / AchievementBonus / AchievementTrace +
// AchievePoint 属性。
type AchievementModule struct {
	BaseModule

	// Achievements 成就表（ini/Achievement/Achievement.xml：CriteriaList/
	// Points/Parent/SubType…）；Criteria 成就子模板表。
	Achievements map[string]map[string]string
	Criteria     map[string]map[string]string

	// unlocked 解锁流水（测试面）。
	unlocked []int32
	// reloads 重载计数。
	reloads int
}

// Name 模块名。
func (m *AchievementModule) Name() string { return "AchievementModule" }

// Init 注册期（对齐官方 0x9CB500）。
func (m *AchievementModule) Init(k *Kernel) error {
	m.Achievements = make(map[string]map[string]string)
	m.Criteria = make(map[string]map[string]string)
	b := k.Bus()
	b.OnIntCommand("player", 716, m.onCommandUpdateCriteria)
	b.OnIntCommand("AttackNpc", 616, m.onNpcBeKilled)
	b.OnIntCommand("BattleCarrier", 616, m.onNpcBeKilled)
	b.OnCommand("player", 340, m.onCustomAchievementMsg)
	b.OnEvent("player", "OnRecover", m.onPlayerRecover)
	b.OnEvent("player", "OnEntry", m.onPlayerEntry)
	b.OnEvent("ToolBox", "OnAfterAdd", m.onToolBoxAfterAdd)
	b.OnEvent("MountBox", "OnNoAdd", m.onMountBoxAfterAdd)
	b.OnEvent("MountPetBox", "OnAfterAdd", m.onPetBoxAfterAdd)
	b.OnEvent("EquipBox", "OnAfterAdd", m.onEquipBoxAfterAdd)
	b.Critical("player", "Prop", m.onPropChanged)
	b.Critical("player", "Level", m.onLevelChanged)
	b.Critical("player", "LifeJobLevel", m.onLifeJobLevelChanged)
	b.Critical("player", "LifeJobExp", m.onLifeJobExpChanged)
	b.Critical("player", "EveRecActive", m.onEveRecActiveChanged)
	b.Critical("player", "FightPower", m.onFightPowerChanged)
	b.Critical("player", "GuildPosition", m.onGuildPositionChanged)
	return nil
}

// LoadResource 成就域配置装载（官方经 StaticDataQuery；Go 面直读）。
func (m *AchievementModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: AchievementModule.LoadResource: ResourcePath 未设置")
	}
	for path, into := range map[string]map[string]map[string]string{
		"ini/Achievement/Achievement.xml":         m.Achievements,
		"ini/Achievement/AchievementCriteria.xml": m.Criteria,
	} {
		data, err := readResource(k.ResourcePath, path)
		if err != nil {
			return fmt.Errorf("fsgame: AchievementModule::LoadResource %s: %w", path, err)
		}
		rows, err := ParsePropTable(data)
		if err != nil {
			return err
		}
		for id, row := range ruleByID(rows) {
			into[id] = row
		}
	}
	k.Tracef("AchievementModule: LoadResource 成就 %d / 子模板 %d",
		len(m.Achievements), len(m.Criteria))
	return nil
}

// UpdateAchievementCriteria 成就子模板进度推进（cmd 716/引擎面）：
// AchievementCriteriaProgress 行推进；全部子模板达需求 → 成就完成
//（AchievementCompleted 追加 + AchievePoint 累加 + 完成时间戳）。
func (m *AchievementModule) UpdateAchievementCriteria(k *Kernel, self PersistID, criteriaID, count int32) bool {
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	cr := o.Record("AchievementCriteriaProgress")
	if cr == nil {
		return false
	}
	for i := 0; i < cr.Rows(); i++ {
		row := cr.Row(i)
		if row.Int(acID) != criteriaID {
			continue
		}
		cur := row.Int(acCur) + count
		if max := row.Int(acMax); max > 0 && cur > max {
			cur = max
		}
		row[acCur] = cur
		if row.Int(acMax) > 0 && cur >= row.Int(acMax) {
			row[acFlag] = CSDone
		}
		cr.ModifyRow(i, row)
		m.checkComplete(k, self, row.Int(acAch))
		return true
	}
	return false
}

// checkComplete 成就完成判定：该成就全部子模板 flag=1 → 解锁。
func (m *AchievementModule) checkComplete(k *Kernel, self PersistID, achID int32) {
	o := k.GetGameObj(self)
	cr := o.Record("AchievementCriteriaProgress")
	allDone, found := true, false
	for i := 0; i < cr.Rows(); i++ {
		row := cr.Row(i)
		if row.Int(acAch) != achID {
			continue
		}
		found = true
		if row.Int(acFlag) != CSDone {
			allDone = false
			break
		}
	}
	if !found || !allDone {
		return
	}
	// 已完成去重。
	if done := o.Record("AchievementCompleted"); done != nil {
		for i := 0; i < done.Rows(); i++ {
			if done.Row(i).Int(amID) == achID {
				return
			}
		}
	}
	m.CompleteAchievement(k, self, achID)
}

// CompleteAchievement 解锁闭环：AchievementCompleted 追加（领取标识 0）+
// AchievePoint += Points（配置行）。
func (m *AchievementModule) CompleteAchievement(k *Kernel, self PersistID, achID int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if done := o.Record("AchievementCompleted"); done != nil && done.Rows() < done.Def().MaxRows {
		done.AppendRow(VarList{achID, int32(0), fmt.Sprintf("%d", unixNow())})
	}
	points := int32(0)
	if cfg := m.Achievements[fmt.Sprint(achID)]; cfg != nil {
		points = atoi32(cfg["Points"])
	}
	_ = k.SetInt(self, "AchievePoint", o.GetInt("AchievePoint")+points)
	m.unlocked = append(m.unlocked, achID)
	k.Tracef("AchievementModule: unlock %d (+%d 点) player=%v", achID, points, self.ID())
}

// TrackAchievement 成就追踪（AchievementTrace 表，cust 340 面）。
func (m *AchievementModule) TrackAchievement(k *Kernel, self PersistID, achID int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if tr := o.Record("AchievementTrace"); tr != nil && tr.Rows() < tr.Def().MaxRows {
		tr.AppendRow(VarList{achID})
	}
}

// ClaimAchievementReward 领取成就奖励（AchievementCompleted 列 1 置位）。
func (m *AchievementModule) ClaimAchievementReward(k *Kernel, self PersistID, achID int32) bool {
	o := k.GetGameObj(self)
	done := o.Record("AchievementCompleted")
	if done == nil {
		return false
	}
	for i := 0; i < done.Rows(); i++ {
		if done.Row(i).Int(amID) == achID {
			row := done.Row(i)
			if row.Int(amClaimed) != 0 {
				return false
			}
			row[amClaimed] = int32(1)
			done.ModifyRow(i, row)
			return true
		}
	}
	return false
}

// RegisterAchievement 登记成就子模板进度行（接取面：按成就行 CriteriaList
// 展开；测试/装载面用）。
func (m *AchievementModule) RegisterAchievement(k *Kernel, self PersistID, achID int32) bool {
	o := k.GetGameObj(self)
	cfg := m.Achievements[fmt.Sprint(achID)]
	if o == nil || cfg == nil {
		return false
	}
	cr := o.Record("AchievementCriteriaProgress")
	if cr == nil {
		return false
	}
	for _, cid := range strings.Split(cfg["CriteriaList"], ",") {
		cid = strings.TrimSpace(cid)
		if cid == "" {
			continue
		}
		ccfg := m.Criteria[cid]
		max := int32(1)
		if ccfg != nil {
			if mc := atoi32(ccfg["MaxCount"]); mc > 0 {
				max = mc
			}
		}
		cr.AppendRow(VarList{atoi32(cid), achID, CSDoing, int32(0), max, achID})
	}
	return true
}

// onCommandUpdateCriteria 官方 @0x9CC9B0（命令 716=0x2CC）。
func (m *AchievementModule) onCommandUpdateCriteria(k *Kernel, self, from PersistID, args VarList) {
	m.UpdateAchievementCriteria(k, self, args.Int(0), args.Int(1))
}

// onNpcBeKilled 官方 @0x9D1BB0（616）：杀怪成就面。
func (m *AchievementModule) onNpcBeKilled(k *Kernel, self, from PersistID, args VarList) {}

// onCustomAchievementMsg 官方 @0x9D2040（cust 340=0x154）：成就 UI 消息
//（args: 子命令, 成就编号——查询/追踪/领奖；pub_achievement_rec 下发面）。
func (m *AchievementModule) onCustomAchievementMsg(k *Kernel, self, from PersistID, args VarList) {
	switch args.Int(0) {
	case 1: // 追踪
		m.TrackAchievement(k, self, args.Int(1))
	case 2: // 领奖
		m.ClaimAchievementReward(k, self, args.Int(1))
	default:
		k.Tracef("AchievementModule: subcmd %d", args.Int(0))
	}
}

// onPlayerRecover 官方 @0x9CF340：成就面回载。
func (m *AchievementModule) onPlayerRecover(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerEntry 官方 @0x9CF550。
func (m *AchievementModule) onPlayerEntry(k *Kernel, self, from PersistID, args VarList) {}

// onToolBoxAfterAdd 官方 @0x9D06F0：物品入手成就面。
func (m *AchievementModule) onToolBoxAfterAdd(k *Kernel, self, from PersistID, args VarList) {}

// onMountBoxAfterAdd 官方 @0x9D0BE0（OnNoAdd 面）。
func (m *AchievementModule) onMountBoxAfterAdd(k *Kernel, self, from PersistID, args VarList) {}

// onPetBoxAfterAdd 官方 @0x9D0A90。
func (m *AchievementModule) onPetBoxAfterAdd(k *Kernel, self, from PersistID, args VarList) {}

// onEquipBoxAfterAdd 官方 @0x9D0EA0。
func (m *AchievementModule) onEquipBoxAfterAdd(k *Kernel, self, from PersistID, args VarList) {}

// 七临界官方面（@0x9D1240-0x9D1AB0）：属性/等级/生活职业/事件弹簧/战力/
// 帮会职位 → 对应类型成就子模板推进。
func (m *AchievementModule) onPropChanged(k *Kernel, self PersistID, prop string, old Var)          {}
func (m *AchievementModule) onLevelChanged(k *Kernel, self PersistID, prop string, old Var)         {}
func (m *AchievementModule) onLifeJobLevelChanged(k *Kernel, self PersistID, prop string, old Var)  {}
func (m *AchievementModule) onLifeJobExpChanged(k *Kernel, self PersistID, prop string, old Var)    {}
func (m *AchievementModule) onEveRecActiveChanged(k *Kernel, self PersistID, prop string, old Var)  {}
func (m *AchievementModule) onFightPowerChanged(k *Kernel, self PersistID, prop string, old Var)    {}
func (m *AchievementModule) onGuildPositionChanged(k *Kernel, self PersistID, prop string, old Var) {}

// NxReloadAchieveResource nx_reload_achieve_resource 的 Go 等价。
func (m *AchievementModule) NxReloadAchieveResource() { m.reloads++ }

// NxResetAchievement nx_reset_achievement：清全部成就面
//（进度/完成/点数归零）。
func (m *AchievementModule) NxResetAchievement(k *Kernel, self PersistID) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	for _, rec := range []string{"AchievementCriteriaProgress", "AchievementCompleted", "AchievementTrace"} {
		if r := o.Record(rec); r != nil {
			r.Clear()
		}
	}
	_ = k.SetInt(self, "AchievePoint", 0)
}

// NxClearSoleAchievement nx_clear_sole_achievement：清指定成就。
func (m *AchievementModule) NxClearSoleAchievement(k *Kernel, self PersistID, achID int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if cr := o.Record("AchievementCriteriaProgress"); cr != nil {
		for i := cr.Rows() - 1; i >= 0; i-- {
			if cr.Row(i).Int(acAch) == achID {
				cr.DeleteRow(i)
			}
		}
	}
	if done := o.Record("AchievementCompleted"); done != nil {
		for i := done.Rows() - 1; i >= 0; i-- {
			if done.Row(i).Int(amID) == achID {
				done.DeleteRow(i)
			}
		}
	}
}

// NxCompleteAchievement nx_complete_achievement（官方参数 object/int）：
// 强制完成（GM/Lua 面）。
func (m *AchievementModule) NxCompleteAchievement(k *Kernel, self PersistID, achID int32) {
	m.CompleteAchievement(k, self, achID)
}

// Unlocked 解锁流水（测试面）。
func (m *AchievementModule) Unlocked() []int32 { return m.unlocked }

// Reloads 重载计数（测试面）。
func (m *AchievementModule) Reloads() int { return m.reloads }
