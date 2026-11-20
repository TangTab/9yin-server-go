package fsgame

import (
	"fmt"
	"strings"
)

// 九期：事件系统核心五模块（docs/official/FsGame全解与Go复刻.md §8
// 事件系统底座）。官方源码树：fsgame\eventspringmodule\。
//
// 事件弹簧（EventSpring）= 世界事件引擎：事件定义（event_spring.xml 216 行）
// 经六参数包表展开——condition_info（条件 ClassId+Param 包）/behavior_info
//（行为 1505 行：Param_2_4 召怪/Param_2_5 坐标）/reward_info（GoalAwards/
// CriteriaAwards）/switch_info（弹簧开关：SwitchType/SteamPos/等级门）/
// task_info（SteamCriteriaId/GoalCriteriaId）/notice_info（公告 389 行）。
// 玩家侧记录（struct/player/event_spring.xml）：PlayerESRec(4×1024)/
// PlayerESCriteriaRec(6×1024)/PlayerESRewardRec(8×1024)。

// ESState 事件弹簧运行态。
const (
	ESIdle int32 = iota // 未开启
	ESRunning           // 进行中
	ESEnding            // 结算期
)

// PlayerESRec 列序（struct/player/event_spring.xml 定案）。
const (
	esEventID = iota // 0 事件编号
	esState          // 1 参与状态
	esScore          // 2 积分
	esJoinTime       // 3 参与时间
)

// ---- EventSpringModule ----

// EventSpringModule 复刻官方 eventspringmodule\eventspringmodule
// （Init @0xA5AD20 / LoadResource @0xA5DFA0，玩法表 Init 序 109；
// 符号 817，9 命令/3 心跳/15 Lua）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("scene","OnCreate",@0xA5E050) / ("scene",
//	    "OnPublicMessage",@0xA5E200)
//	AddEventCallback ("player","OnRecover",@0xA5E420) / ("player",
//	    "OnStore",@0xA5E9C0) / ("player","OnReady",@0xA5EF80
//	    OnPlayerEntry) / ("player","OnLeave",@0xA5F590) / ("player",
//	    "OnMapChanged",@0xA5FA00)
//	AddEventCallback ("AttackNpc","OnSpring",@0xA63B30) / ("AttackNpc",
//	    "OnEndSpring",@0xA63DA0) / ×2 BattleCarrier
//	AddIntCommandHook ("scene",751,@0xA5FAB0 OnCmdESStart) / (761,
//	    @0xA5FB70 OnCmdPlayerRecover) / (762,@0xA5FE90 OnCmdPlayerStore) /
//	    (759,@0xA57CC0) / (756,@0xA5A5F0) / (757,@0xA5A7F0)
//	AddIntCommandHook ("player",763,@0xA5FFA0 OnCmdPlayerExitES) /
//	    ("player",616,@0xA601A0 OnCmdPlayerBeKilled) / ("AttackNpc",616,
//	    @0xA633A0 OnCmdNpcBeKilled)
//	AddIntCustomHook ("player",661=0x295,@0xA575F0 OnClientMessage)
//	DeclareHeartBeat ×3：HB_CheckCondition@0xA60480 / HB_CheckEvent@
//	    @0xA60800 / HB_EndES@0xA632B0（+CB_ReloadResource 泵）
//	DeclareLuaExt ×15：nx_start_es/nx_end_es/nx_reload_es/nx_get_area_id/
//	    nx_get_event_by_id/nx_not_exist_{behavior,condition,notice,reward,
//	    switch,task}_id/nx_query_es_reward/nx_open/close/write_event_log
type EventSpringModule struct {
	BaseModule

	// Events 事件定义表；Conditions/Behaviors/Rewards/Switches/Tasks/
	// Notices 六参数包表。
	Events     map[string]map[string]string
	Conditions map[string]map[string]string
	Behaviors  map[string]map[string]string
	Rewards    map[string]map[string]string
	Switches   map[string]map[string]string
	Tasks      map[string]map[string]string
	Notices    map[string]map[string]string

	// running 运行事件：eventID → 状态/剩余拍。
	running map[int32]*esRun
	// ops 命令流水（测试面）。
	ops []string
	// firedNotices 公告流水（notice_info TextId）。
	firedNotices []string
}

type esRun struct {
	State    int32
	Duration int32 // 剩余拍（官方 Duration 秒面）
}

// Name 模块名。
func (m *EventSpringModule) Name() string { return "EventSpringModule" }

// Init 注册期（对齐官方 0xA5AD20）。
func (m *EventSpringModule) Init(k *Kernel) error {
	m.Events = make(map[string]map[string]string)
	m.Conditions = make(map[string]map[string]string)
	m.Behaviors = make(map[string]map[string]string)
	m.Rewards = make(map[string]map[string]string)
	m.Switches = make(map[string]map[string]string)
	m.Tasks = make(map[string]map[string]string)
	m.Notices = make(map[string]map[string]string)
	m.running = make(map[int32]*esRun)
	b := k.Bus()
	b.OnEvent("scene", "OnCreate", m.onSceneCreate)
	b.OnEvent("scene", "OnPublicMessage", m.onScenePublicMessage)
	b.OnEvent("player", "OnRecover", m.onPlayerRecover)
	b.OnEvent("player", "OnStore", m.onPlayerStore)
	b.OnEvent("player", "OnReady", m.onPlayerEntry)
	b.OnEvent("player", "OnLeave", m.onPlayerLeave)
	b.OnEvent("player", "OnMapChanged", m.onPlayerMapChanged)
	b.OnEvent("AttackNpc", "OnSpring", m.onNpcSpring)
	b.OnEvent("AttackNpc", "OnEndSpring", m.onNpcEndSpring)
	b.OnEvent("BattleCarrier", "OnSpring", m.onNpcSpring)
	b.OnEvent("BattleCarrier", "OnEndSpring", m.onNpcEndSpring)
	b.OnIntCommand("scene", 751, m.onCmdESStart)
	b.OnIntCommand("scene", 761, m.onCmdPlayerRecover)
	b.OnIntCommand("scene", 762, m.onCmdPlayerStore)
	b.OnIntCommand("scene", 759, m.onSceneCmd)
	b.OnIntCommand("scene", 756, m.onSceneCmd)
	b.OnIntCommand("scene", 757, m.onSceneCmd)
	b.OnIntCommand("player", 763, m.onCmdPlayerExitES)
	b.OnIntCommand("player", 616, m.onCmdPlayerBeKilled)
	b.OnIntCommand("AttackNpc", 616, m.onCmdNpcBeKilled)
	b.OnCommand("player", 661, m.onClientMessage)
	b.HeartBeat("EventSpringModule::HB_CheckCondition", m.hbCheckCondition)
	b.HeartBeat("EventSpringModule::HB_CheckEvent", m.hbCheckEvent)
	b.HeartBeat("EventSpringModule::HB_EndES", m.hbEndES)
	return nil
}

// LoadResource 官方 @0xA5DFA0：ini/EventSpring/ 七表。
func (m *EventSpringModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: EventSpringModule.LoadResource: ResourcePath 未设置")
	}
	for path, into := range map[string]map[string]map[string]string{
		"ini/EventSpring/event_spring.xml":  m.Events,
		"ini/EventSpring/condition_info.xml": m.Conditions,
		"ini/EventSpring/behavior_info.xml":  m.Behaviors,
		"ini/EventSpring/reward_info.xml":    m.Rewards,
		"ini/EventSpring/switch_info.xml":    m.Switches,
		"ini/EventSpring/task_info.xml":      m.Tasks,
		"ini/EventSpring/notice_info.xml":    m.Notices,
	} {
		data, err := readResource(k.ResourcePath, path)
		if err != nil {
			return fmt.Errorf("fsgame: EventSpringModule::LoadResource %s: %w", path, err)
		}
		rows, err := ParsePropTable(data)
		if err != nil {
			return err
		}
		for id, row := range ruleByID(rows) {
			into[id] = row
		}
	}
	k.Tracef("EventSpringModule: LoadResource 事件 %d/条件 %d/行为 %d/奖励 %d/开关 %d/任务 %d/公告 %d",
		len(m.Events), len(m.Conditions), len(m.Behaviors), len(m.Rewards),
		len(m.Switches), len(m.Tasks), len(m.Notices))
	return nil
}

// StartES 启动事件（官方 OnCmdESStart @0xA5FAB0 / nx_start_es）：
// 事件表定义门 → running 登记（Duration 秒）→ 公告下发（NoticeId 逗号串）。
func (m *EventSpringModule) StartES(k *Kernel, eventID int32) bool {
	cfg := m.Events[fmt.Sprint(eventID)]
	if cfg == nil {
		return false
	}
	if m.running[eventID] != nil {
		return false // 已在跑
	}
	m.running[eventID] = &esRun{State: ESRunning, Duration: atoi32(cfg["Duration"])}
	for _, nid := range strings.Split(cfg["NoticeId"], ",") {
		if notice := m.Notices[strings.TrimSpace(nid)]; notice != nil {
			m.firedNotices = append(m.firedNotices, notice["TextId"])
		}
	}
	m.ops = append(m.ops, fmt.Sprintf("start:%d", eventID))
	k.Tracef("EventSpringModule: start ES %d (duration=%ds)", eventID, atoi32(cfg["Duration"]))
	return true
}

// EndES 结束（官方 OnCmdPlayerExitES 面 763 / nx_end_es / HB_EndES 到期）。
func (m *EventSpringModule) EndES(k *Kernel, eventID int32) {
	if _, ok := m.running[eventID]; !ok {
		return
	}
	delete(m.running, eventID)
	m.ops = append(m.ops, fmt.Sprintf("end:%d", eventID))
}

// JoinES 玩家参与（PlayerESRec 追加：事件/状态/积分/时间）。
func (m *EventSpringModule) JoinES(k *Kernel, self PersistID, eventID int32) bool {
	if m.running[eventID] == nil {
		return false
	}
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	r := o.Record("PlayerESRec")
	if r == nil {
		return false
	}
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).Int(esEventID) == eventID {
			return true // 已参与
		}
	}
	if r.Rows() >= r.Def().MaxRows {
		return false
	}
	r.AppendRow(VarList{eventID, ESRunning, int32(0), nowUnix()})
	return true
}

// RunningES 运行事件集（测试面）。
func (m *EventSpringModule) RunningES() map[int32]*esRun { return m.running }

// onCmdESStart 官方 @0xA5FAB0（scene 命令 751=0x2EF）：启动。
func (m *EventSpringModule) onCmdESStart(k *Kernel, self, from PersistID, args VarList) {
	m.StartES(k, args.Int(0))
}

// onCmdPlayerRecover/onCmdPlayerStore 官方 @0xA5FB70/@0xA5FE90（scene 面）。
func (m *EventSpringModule) onCmdPlayerRecover(k *Kernel, self, from PersistID, args VarList) {}
func (m *EventSpringModule) onCmdPlayerStore(k *Kernel, self, from PersistID, args VarList)  {}

// onSceneCmd 官方 759/756/757（scene 侧事件消息族）。
func (m *EventSpringModule) onSceneCmd(k *Kernel, self, from PersistID, args VarList) {}

// onCmdPlayerExitES 官方 @0xA5FFA0（player 命令 763=0x2FB）：退出事件。
func (m *EventSpringModule) onCmdPlayerExitES(k *Kernel, self, from PersistID, args VarList) {}

// onCmdPlayerBeKilled 官方 @0xA601A0（616）：玩家被杀（事件积分面）。
func (m *EventSpringModule) onCmdPlayerBeKilled(k *Kernel, self, from PersistID, args VarList) {}

// onCmdNpcBeKilled 官方 @0xA633A0（616）：NPC 被杀 → 事件积分/条件推进
//（task_info SteamCriteriaId → QuestModule 协作面）。
func (m *EventSpringModule) onCmdNpcBeKilled(k *Kernel, self, from PersistID, args VarList) {
	for id := range m.running {
		if cfg := m.Events[fmt.Sprint(id)]; cfg != nil {
			if t := m.Tasks[cfg["TaskId"]]; t != nil {
				if qm, ok := k.GetLogicModule("QuestModule").(*QuestModule); ok && qm != nil {
					qm.UpdateCriteria(k, from, atoi32(t["SteamCriteriaId"]), 1, false)
				}
			}
		}
	}
}

// onClientMessage 官方 @0xA575F0（cust 661=0x295）：客户端事件消息多路
//（查询/抽奖/兑换——nx_query_es_reward 面）。
func (m *EventSpringModule) onClientMessage(k *Kernel, self, from PersistID, args VarList) {}

// onSceneCreate 官方 @0xA5E050：场景创建（本场景事件装载）。
func (m *EventSpringModule) onSceneCreate(k *Kernel, self, from PersistID, args VarList) {}

// onScenePublicMessage 官方 @0xA5E200：跨 member 事件消息。
func (m *EventSpringModule) onScenePublicMessage(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerRecover 官方 @0xA5E420：PlayerESRec 三表回载。
func (m *EventSpringModule) onPlayerRecover(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerStore 官方 @0xA5E9C0。
func (m *EventSpringModule) onPlayerStore(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerEntry 官方 @0xA5EF80：入场（运行中事件同步）。
func (m *EventSpringModule) onPlayerEntry(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerLeave 官方 @0xA5F590。
func (m *EventSpringModule) onPlayerLeave(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerMapChanged 官方 @0xA5FA00：换图（事件区域判定）。
func (m *EventSpringModule) onPlayerMapChanged(k *Kernel, self, from PersistID, args VarList) {}

// onNpcSpring/onNpcEndSpring 官方 @0xA63B30/@0xA63DA0。
func (m *EventSpringModule) onNpcSpring(k *Kernel, self, from PersistID, args VarList)    {}
func (m *EventSpringModule) onNpcEndSpring(k *Kernel, self, from PersistID, args VarList) {}

// HB_CheckCondition 官方 @0xA60480：条件泵（condition_info ClassId 分派）。
func (m *EventSpringModule) hbCheckCondition(k *Kernel, self PersistID, beats int32) {}

// HB_CheckEvent 官方 @0xA60800：事件泵（开关/进入判定）。
func (m *EventSpringModule) hbCheckEvent(k *Kernel, self PersistID, beats int32) {}

// HB_EndES 官方 @0xA632B0：到期结束（Duration 倒计时 → EndES）。
func (m *EventSpringModule) hbEndES(k *Kernel, self PersistID, beats int32) {
	for id, run := range m.running {
		if run.Duration > 0 {
			run.Duration--
		}
		if run.Duration <= 0 {
			m.EndES(k, id)
		}
	}
}

// NxEndES/NxReloadES/nx_query 族 Go 等价。
func (m *EventSpringModule) NxEndES(k *Kernel, eventID int32)    { m.EndES(k, eventID) }
func (m *EventSpringModule) NxGetEventByID(eventID int32) map[string]string {
	return m.Events[fmt.Sprint(eventID)]
}

// NxNotExistBehaviorID nx_not_exist_behavior_id 族（配置校验面）。
func (m *EventSpringModule) NxNotExistBehaviorID(id string) bool { return m.Behaviors[id] == nil }

// Ops/FiredNotices 流水（测试面）。
func (m *EventSpringModule) Ops() []string          { return m.ops }
func (m *EventSpringModule) FiredNotices() []string { return m.firedNotices }

// ---- TriggerProcesserModule ----

// TriggerProcesserModule 复刻官方 triggerprocessermodule（Init @0xA70260 /
// LoadResource @0xA703C0，玩法表 Init 序 102；符号 458）——触发器处理器
//（失败串定案 "TriggerProcesserModule: LoadResource(): load file error ..."，
// EventSpringProcesser 配置族）。
//
// 官方 Init 注册面：
//
//	AddIntCommandHook ("player",753=0x2F1 / 754=0x2F2)
//	DeclareHeartBeat ("TriggerProcesserModule::HB_WaitProcess")
//	事件 ×3（注册经扫描面）
type TriggerProcesserModule struct {
	BaseModule

	// Processers 处理器表（EventSpringProcesser 配置族）。
	Processers map[string]map[string]string
	// ops 流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *TriggerProcesserModule) Name() string { return "TriggerProcesserModule" }

// Init 注册期（对齐官方 0xA70260）。
func (m *TriggerProcesserModule) Init(k *Kernel) error {
	m.Processers = make(map[string]map[string]string)
	b := k.Bus()
	b.OnIntCommand("player", 753, m.onCmdProcess)
	b.OnIntCommand("player", 754, m.onCmdProcess)
	b.HeartBeat("TriggerProcesserModule::HB_WaitProcess", m.hbWaitProcess)
	return nil
}

// LoadResource 官方 @0xA703C0：EventSpringProcesser 配置族装载。
func (m *TriggerProcesserModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: TriggerProcesserModule.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/EventSpring/point_set.xml")
	if err != nil {
		return fmt.Errorf("fsgame: TriggerProcesserModule: LoadResource(): load file error ...: %w", err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	for id, row := range ruleByID(rows) {
		m.Processers[id] = row
	}
	k.Tracef("TriggerProcesserModule: LoadResource 处理器 %d", len(m.Processers))
	return nil
}

// onCmdProcess 官方（命令 753/754）：触发处理（条件链求值入口）。
func (m *TriggerProcesserModule) onCmdProcess(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, fmt.Sprintf("process:%d", args.Int(0)))
}

// HB_WaitProcess 官方：等待处理泵。
func (m *TriggerProcesserModule) hbWaitProcess(k *Kernel, self PersistID, beats int32) {}

// Ops 流水（测试面）。
func (m *TriggerProcesserModule) Ops() []string { return m.ops }

// ---- RewardExecutor ----

// RewardExecutor 复刻官方 eventspringmodule\rewardexecutor
// （Init @0xA918F0，玩法表 Init 序 106；符号 197）——奖励执行器
//（reward_info：GoalAwards 完成奖/CriteriaAwards 进度奖，逗号串
// "物品,数量,绑定" 族）。
type RewardExecutor struct {
	BaseModule

	// RewardOut 奖励出口（接 ToolBox/Capital）。
	RewardOut func(player PersistID, itemID string, count int32)
	// granted 发放流水（测试面）。
	granted []string
}

// Name 模块名。
func (m *RewardExecutor) Name() string { return "RewardExecutor" }

// Init 注册期（对齐官方 0xA918F0——执行器经 EventSpring 协作面调用，
// 无独立命令注册）。
func (m *RewardExecutor) Init(k *Kernel) error { return nil }

// Execute 奖励执行：reward_info 行 → CriteriaAwards/GoalAwards 逗号段展开
//（"item,count,bind"）→ RewardOut。
func (m *RewardExecutor) Execute(k *Kernel, self PersistID, rewardID string, goal bool) int {
	row := m.rewardRow(k, rewardID)
	if row == nil {
		return 0
	}
	field := row["CriteriaAwards"]
	if goal {
		field = row["GoalAwards"]
	}
	n := 0
	for _, seg := range strings.Split(field, ",") {
		seg = strings.TrimSpace(seg)
		// 官方逗号段语义："物品ID" 或空段跳过（数量绑定经静态表）。
		if seg == "" {
			continue
		}
		if m.RewardOut != nil {
			m.RewardOut(self, seg, 1)
		}
		m.granted = append(m.granted, seg)
		n++
	}
	return n
}

// rewardRow 奖励行（经 EventSpringModule 配置协作）。
func (m *RewardExecutor) rewardRow(k *Kernel, rewardID string) map[string]string {
	if es, ok := k.GetLogicModule("EventSpringModule").(*EventSpringModule); ok && es != nil {
		return es.Rewards[rewardID]
	}
	return nil
}

// Granted 发放流水（测试面）。
func (m *RewardExecutor) Granted() []string { return m.granted }

// ---- SwitchExecutor ----

// SwitchExecutor 复刻官方 switchexecutor（Init @0xA978B0，玩法表 Init 序
// 105；符号 48）——弹簧开关执行器（switch_info：SwitchType/SteamPos/
// MinLevel-MaxLevel 等级门/SceneId/AreaId）。
type SwitchExecutor struct {
	BaseModule

	// entered 进入流水（测试面）。
	entered []string
}

// Name 模块名。
func (m *SwitchExecutor) Name() string { return "SwitchExecutor" }

// Init 注册期（对齐官方 0xA978B0——cust 758=0x2F6）。
func (m *SwitchExecutor) Init(k *Kernel) error {
	b := k.Bus()
	b.OnCommand("player", 758, m.onCustomEnter)
	return nil
}

// onCustomEnter 官方（cust 758）：经开关进入事件
//（switch_info 等级门 → EventSpringModule::JoinES）。
func (m *SwitchExecutor) onCustomEnter(k *Kernel, self, from PersistID, args VarList) {
	switchID := args.String(0)
	sw := m.switchRow(k, switchID)
	o := k.GetGameObj(self)
	if sw == nil || o == nil {
		return
	}
	if lv := o.GetInt("Level"); lv < atoi32(sw["MinLevel"]) || lv > atoi32(sw["MaxLevel"]) {
		return
	}
	if es, ok := k.GetLogicModule("EventSpringModule").(*EventSpringModule); ok && es != nil {
		for id := range es.RunningES() {
			if es.JoinES(k, self, id) {
				m.entered = append(m.entered, fmt.Sprintf("%s→ES%d", switchID, id))
			}
		}
	}
}

// switchRow 开关行（EventSpring 配置协作）。
func (m *SwitchExecutor) switchRow(k *Kernel, id string) map[string]string {
	if es, ok := k.GetLogicModule("EventSpringModule").(*EventSpringModule); ok && es != nil {
		return es.Switches[id]
	}
	return nil
}

// Entered 进入流水（测试面）。
func (m *SwitchExecutor) Entered() []string { return m.entered }

// ---- TaskExecutor ----

// TaskExecutor 复刻官方 eventspringmodule\taskexecutor（Init @0xA9AA10，
// 玩法表 Init 序 104；符号 118）——事件任务执行器（task_info：
// SteamCriteriaId 进度/SteamGoalCriteriaId 目标——QuestModule 协作）。
//
// 官方 Init 注册面：
//
//	AddIntCommandHook ("player",927=0x39F / 936=0x3A8)
//	DeclareHeartBeat ("TaskExecutor::HB_SyncSharedCurNum")
//	DeclareCritical ×2 + 事件 ×1
type TaskExecutor struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *TaskExecutor) Name() string { return "TaskExecutor" }

// Init 注册期（对齐官方 0xA9AA10）。
func (m *TaskExecutor) Init(k *Kernel) error {
	b := k.Bus()
	b.OnIntCommand("player", 927, m.onCmdTask)
	b.OnIntCommand("player", 936, m.onCmdGoal)
	b.HeartBeat("TaskExecutor::HB_SyncSharedCurNum", m.hbSyncSharedCurNum)
	return nil
}

// onCmdTask 官方（命令 927=0x39F）：事件任务进度
//（task_info SteamCriteriaId → QuestModule::UpdateCriteria）。
func (m *TaskExecutor) onCmdTask(k *Kernel, self, from PersistID, args VarList) {
	m.executeCriteria(k, self, args.String(0), args.Int(1))
}

// onCmdGoal 官方（命令 936=0x3A8）：目标进度（SteamGoalCriteriaId）。
func (m *TaskExecutor) onCmdGoal(k *Kernel, self, from PersistID, args VarList) {
	m.executeCriteria(k, self, args.String(0), args.Int(1))
}

// executeCriteria 按 task_info 行推进（goal=false 进度/true 目标）。
func (m *TaskExecutor) executeCriteria(k *Kernel, self PersistID, taskID string, count int32) {
	es, _ := k.GetLogicModule("EventSpringModule").(*EventSpringModule)
	qm, _ := k.GetLogicModule("QuestModule").(*QuestModule)
	if es == nil || qm == nil {
		return
	}
	t := es.Tasks[taskID]
	if t == nil {
		return
	}
	criteriaID := atoi32(t["SteamCriteriaId"])
	if criteriaID != 0 {
		qm.UpdateCriteria(k, self, criteriaID, count, false)
		m.ops = append(m.ops, fmt.Sprintf("task:%s→%d", taskID, criteriaID))
	}
}

// HB_SyncSharedCurNum 官方：共享进度同步泵。
func (m *TaskExecutor) hbSyncSharedCurNum(k *Kernel, self PersistID, beats int32) {}

// Ops 流水（测试面）。
func (m *TaskExecutor) Ops() []string { return m.ops }
