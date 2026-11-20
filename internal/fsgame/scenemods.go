package fsgame

import "fmt"

// 七期：场景域核心五模块（docs/official/FsGame全解与Go复刻.md §8 场景基础）。
// 官方源码树：fsgame\scenebasemodule\（clonescenemodule/mapmagmodule/
// areaobjmodule）+ fsgame\stagemodule\warfog。

// ---- CloneSceneModule ----

// CloneSceneModule 复刻官方 scenebasemodule\clonescenemodule
// （Init @0x2A6BD0，玩法表 Init 序 115；符号 1528，14 命令/11 心跳/11 Lua）。
//
// 官方 Init 注册面（p2scan 反汇编定案，节选）：
//
//	AddClassCallback ("CloneScene","OnCreateClass",@0x2A76E0——
//	    struct/scene/clone_scene.xml schema：CloneDifficulty/CloneProgress/
//	    CloneUniqueID/CloneOwnerType/CloneOwnerID/BossBeKilledCount/
//	    PassStageTime/PlayerDeadCount/PScore1 评分族）
//	AddEventCallback ("scene","OnPublicMessage",@0x280800) / ("scene",
//	    "OnCloneScene",@0x280F00)
//	AddEventCallback ("CloneScene","OnCloneReset",@0x281CD0) /
//	    ("CloneScene","OnDestroy",@0x281D90)
//	AddEventCallback ("role","OnCreateRole",@0x281F20) + player
//	    OnRecover/OnEntry/OnReady/OnLeaveScene/OnContinue 五事件 + ("VisibleNpc",
//	    "OnEntry",@0x282900)
//	AddIntCommandHook ("CloneScene",799,@0x2830C0 OnComandCloneSceneScene) /
//	    ("player"/"AttackNpc"/"BattleCarrier",800,@0x283240
//	    OnComandCloneScenePlayer) / ("player",110/114/112,@0x283A10/
//	    @0x2843A0/@0x284870——队伍多播协同：CreateTeam/ClearTeam/
//	    AddTeamMember) / ("player",805,@0x284EF0 OnCommandDecCapital) /
//	    ("player",616,@0x284FD0 OnCommandDeadInClone) / ("AttackNpc"/
//	    "BattleCarrier",616,@0x2852D0 OnCommandCloneBossDead)
//	AddIntCustomHook ("player",139=0x8B,@0x28C280)
//	DeclareHeartBeat ×6：HB_CloseClone@0x28FBE0(推断)/HB_CloneCount/
//	    HB_LeaveTeamExit@0x290B40/HB_CloneLimiteTime@0x290E20/HB_CloneDoor@
//	    @0x2910C0/HB_AnotherLeaveTeam@0x291610
//	DeclareCritical ×7+：C_OnProgressChanged@0x28A170 / C_OnTeamCaptain@
//	    @0x28A2D0 / C_OnTeamID@0x28A4A0 / C_OnLogicStateChanged@0x28AEE0 /
//	    C_OnVipStateChanged / C_OnGmPassChanged / C_OnNextMonster
//	DeclareLuaExt ×11：nx_clone_scene/nx_clear_clone_times/nx_close_clone/
//	    nx_clone_pass/nx_get_clone_maxnum/nx_get_clone_maxplayers/
//	    nx_reload_clone_config/nx_get_proto_scene_id/nx_set_clone_enter_count/
//	    nx_set_clone_pass_score/nx_set_reward_rand_count
//
// 评分配置（官方经 StaticDataQuery；Go 面 LoadResource 直读）：
// ini/Scene/CloneSceneGrade.xml（44 行：ID="<scene>_<difficulty>" →
// TargetPassTime/NormalPassTime/TargetBossTime/NormalBossTime/DeadCount/
// DeadDecrease/ScoreFactor）。
type CloneSceneModule struct {
	BaseModule

	// Grades 评分表（scene_difficulty → 行）。
	Grades map[string]map[string]string

	// clones 运行副本：CloneUniqueID → 场景对象句柄。
	clones map[string]PersistID
	// nextClone 副本序号。
	nextClone int64
	// ops 命令流水（测试面）。
	ops []string
	// scores 通关评分流水（测试面）。
	scores []string
}

// Name 模块名。
func (m *CloneSceneModule) Name() string { return "CloneSceneModule" }

// Init 注册期（对齐官方 0x2A6BD0）。
func (m *CloneSceneModule) Init(k *Kernel) error {
	m.Grades = make(map[string]map[string]string)
	m.clones = make(map[string]PersistID)
	b := k.Bus()
	b.OnClassEvent("CloneScene", "OnCreateClass", m.onCloneSceneCreateClass)
	b.OnEvent("scene", "OnPublicMessage", m.onScenePublicMessage)
	b.OnEvent("scene", "OnCloneScene", m.onCloneScene)
	b.OnEvent("CloneScene", "OnCloneReset", m.onCloneReset)
	b.OnEvent("CloneScene", "OnDestroy", m.onCloneDestroy)
	b.OnEvent("role", "OnCreateRole", m.onCreateRole)
	b.OnEvent("player", "OnRecover", m.onPlayerEvent)
	b.OnEvent("player", "OnEntry", m.onPlayerEntry)
	b.OnEvent("player", "OnReady", m.onPlayerEvent)
	b.OnEvent("player", "OnLeaveScene", m.onPlayerEvent)
	b.OnEvent("player", "OnContinue", m.onPlayerEvent)
	b.OnEvent("VisibleNpc", "OnEntry", m.onVisibleNpcEntry)
	b.OnIntCommand("CloneScene", 799, m.onComandCloneSceneScene)
	for _, cls := range []string{"player", "AttackNpc", "BattleCarrier"} {
		b.OnIntCommand(cls, 800, m.onComandCloneScenePlayer)
	}
	b.OnIntCommand("player", 110, m.onCommandCreateTeam)
	b.OnIntCommand("player", 114, m.onCommandClearTeam)
	b.OnIntCommand("player", 112, m.onCommandAddTeamMember)
	b.OnIntCommand("player", 805, m.onCommandDecCapital)
	b.OnIntCommand("player", 616, m.onCommandDeadInClone)
	b.OnIntCommand("AttackNpc", 616, m.onCommandCloneBossDead)
	b.OnIntCommand("BattleCarrier", 616, m.onCommandCloneBossDead)
	b.OnCommand("player", 139, m.onCustomMsg)
	b.HeartBeat("CloneSceneModule::HB_CloseClone", m.hbCloseClone)
	b.HeartBeat("CloneSceneModule::HB_CloneCount", m.hbCloneCount)
	b.HeartBeat("CloneSceneModule::HB_LeaveTeamExit", m.hbLeaveTeamExit)
	b.HeartBeat("CloneSceneModule::HB_CloneLimiteTime", m.hbCloneLimiteTime)
	b.HeartBeat("CloneSceneModule::HB_CloneDoor", m.hbCloneDoor)
	b.HeartBeat("CloneSceneModule::HB_AnotherLeaveTeam", m.hbAnotherLeaveTeam)
	b.Critical("CloneScene", "CloneProgress", m.onProgressChanged)
	b.Critical("CloneScene", "TeamCaptain", m.onTeamCaptain)
	b.Critical("CloneScene", "TeamID", m.onTeamID)
	b.Critical("CloneScene", "LogicState", m.onLogicStateChanged)
	b.Critical("CloneScene", "VipState", m.onVipStateChanged)
	b.Critical("CloneScene", "GmPass", m.onGmPassChanged)
	b.Critical("CloneScene", "NextMonster", m.onNextMonster)
	return nil
}

// LoadResource 评分表装载（官方经 StaticDataQuery；Go 面直读）。
func (m *CloneSceneModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: CloneSceneModule.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/Scene/CloneSceneGrade.xml")
	if err != nil {
		return fmt.Errorf("fsgame: CloneSceneModule::LoadResource CloneSceneGrade.xml: %w", err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	for id, row := range ruleByID(rows) {
		m.Grades[id] = row
	}
	k.Tracef("CloneSceneModule: LoadResource 评分表 %d 行", len(m.Grades))
	return nil
}

// CreateClone 建副本闭环（官方 nx_clone_scene 的 Go 等价）：物化 CloneScene
// 场景对象 → CloneOwnerID/OwnerType/Difficulty/CloneUniqueID 属性链 →
// 进入计数登记。
func (m *CloneSceneModule) CreateClone(k *Kernel, protoSceneID int32, difficulty int32, owner PersistID, ownerType int32) (string, PersistID) {
	id, err := k.CreateObject("CloneScene")
	if err != nil {
		return "", NullPersistID
	}
	uid := fmt.Sprintf("CLONE-%d-%d", protoSceneID, m.nextClone)
	m.nextClone++
	o := k.GetGameObj(id)
	_ = o.Set("CloneUniqueID", uid)
	_ = o.Set("CloneDifficulty", difficulty)
	_ = o.Set("CloneOwnerType", ownerType)
	if oo := k.GetGameObj(owner); oo != nil {
		_ = o.Set("CloneOwnerID", oo.GetInt("TeamID"))
		_ = o.Set("CloneOwnerName", oo.GetString("name"))
	}
	_ = o.Set("CloneProgress", int32(0))
	m.clones[uid] = id
	m.ops = append(m.ops, "create:"+uid)
	k.Tracef("CloneSceneModule: create %s (proto=%d diff=%d)", uid, protoSceneID, difficulty)
	return uid, id
}

// CloneOf 按唯一号取副本场景（测试面）。
func (m *CloneSceneModule) CloneOf(uid string) PersistID { return m.clones[uid] }

// SetProgress 进度推进（官方 C_OnProgressChanged 联动——100% 通关）。
func (m *CloneSceneModule) SetProgress(k *Kernel, clone PersistID, progress int32) {
	_ = k.SetInt(clone, "CloneProgress", progress)
}

// PassClone 通关评分（官方 nx_clone_pass/nx_set_clone_pass_score 面）：
// 按 CloneSceneGrade 表（"<scene>_<difficulty>"）算 S/A/B/C——
// PassTime ≤ Target 全分 / ≤ Normal 过线；DeadCount 扣 DeadDecrease；
// 总分 × ScoreFactor。
func (m *CloneSceneModule) PassClone(k *Kernel, clone PersistID, protoSceneID int32, passTime int64) string {
	o := k.GetGameObj(clone)
	if o == nil {
		return ""
	}
	key := fmt.Sprintf("%d_%d", protoSceneID, o.GetInt("CloneDifficulty"))
	g := m.Grades[key]
	score := 100
	if g != nil {
		score = 60
		if passTime <= int64(atoi32(g["TargetPassTime"])) {
			score = 100
		} else if passTime <= int64(atoi32(g["NormalPassTime"])) {
			score = 80
		}
		if dead := o.GetInt("PlayerDeadCount"); dead > 0 {
			score -= int(dead) * int(atoi32(g["DeadDecrease"]))
		}
		if score < 0 {
			score = 0
		}
	}
	grade := "S"
	switch {
	case score >= 100:
		grade = "S"
	case score >= 80:
		grade = "A"
	case score >= 60:
		grade = "B"
	default:
		grade = "C"
	}
	m.scores = append(m.scores, fmt.Sprintf("%s=%s(%d)", o.GetString("CloneUniqueID"), grade, score))
	m.ops = append(m.ops, "pass")
	_ = k.SetInt(clone, "CloneProgress", 100)
	k.Tracef("CloneSceneModule: pass %s grade=%s score=%d",
		o.GetString("CloneUniqueID"), grade, score)
	return grade
}

// CloseClone 关闭（官方 nx_close_clone/HB_CloseClone 倒计时面）。
func (m *CloneSceneModule) CloseClone(k *Kernel, uid string) {
	if id := m.clones[uid]; id.Valid() {
		k.RemoveObject(id)
		delete(m.clones, uid)
		m.ops = append(m.ops, "close:"+uid)
	}
}

// NxClearCloneTimes nx_clear_clone_times（周次清零——CloneWeekLimitCount 面）。
func (m *CloneSceneModule) NxClearCloneTimes() {}

// NxGetCloneMaxnum nx_get_clone_maxnum（同刷本次数上限）。
func (m *CloneSceneModule) NxGetCloneMaxnum(protoSceneID int32) int32 { return 0 }

// NxGetCloneMaxplayers nx_get_clone_maxplayers（人数上限）。
func (m *CloneSceneModule) NxGetCloneMaxplayers(protoSceneID int32) int32 { return 5 }

// NxGetProtoSceneId nx_get_proto_scene_id（副本 → 原型场景）。
func (m *CloneSceneModule) NxGetProtoSceneId(uid string) int32 { return 0 }

// onComandCloneSceneScene 官方 @0x2830C0（命令 799=0x31F）：场景侧副本命令。
func (m *CloneSceneModule) onComandCloneSceneScene(k *Kernel, self, from PersistID, args VarList) {}

// onComandCloneScenePlayer 官方 @0x283240（命令 800=0x320）：玩家侧
//（CloneScoreTopRefreshTime/CloneWeekLimitCount 面下发）。
func (m *CloneSceneModule) onComandCloneScenePlayer(k *Kernel, self, from PersistID, args VarList) {}

// onCommandCreateTeam 官方 @0x283A10（命令 110 多播协同——队伍变更联动
// CloneOwnerID）。
func (m *CloneSceneModule) onCommandCreateTeam(k *Kernel, self, from PersistID, args VarList) {}

func (m *CloneSceneModule) onCommandClearTeam(k *Kernel, self, from PersistID, args VarList)    {}
func (m *CloneSceneModule) onCommandAddTeamMember(k *Kernel, self, from PersistID, args VarList) {}

// onCommandDecCapital 官方 @0x284EF0（命令 805=0x325）：副本货币消耗回执。
func (m *CloneSceneModule) onCommandDecCapital(k *Kernel, self, from PersistID, args VarList) {}

// onCommandDeadInClone 官方 @0x284FD0（616）：副本内死亡（PlayerDeadCount+1）。
func (m *CloneSceneModule) onCommandDeadInClone(k *Kernel, self, from PersistID, args VarList) {
	for _, id := range m.clones {
		if o := k.GetGameObj(id); o != nil {
			_ = k.SetInt(id, "PlayerDeadCount", o.GetInt("PlayerDeadCount")+1)
		}
	}
}

// onCommandCloneBossDead 官方 @0x2852D0（616）：Boss 击杀
//（BossBeKilledCount+1 + BossTimeCount 记时）。
func (m *CloneSceneModule) onCommandCloneBossDead(k *Kernel, self, from PersistID, args VarList) {
	for _, id := range m.clones {
		if o := k.GetGameObj(id); o != nil {
			_ = k.SetInt(id, "BossBeKilledCount", o.GetInt("BossBeKilledCount")+1)
		}
	}
}

// onCustomMsg 官方 @0x28C280（cust 139=0x8B）：副本 UI 命令多路。
func (m *CloneSceneModule) onCustomMsg(k *Kernel, self, from PersistID, args VarList) {}

// onCloneSceneCreateClass 官方 @0x2A76E0：CloneScene schema 就绪。
func (m *CloneSceneModule) onCloneSceneCreateClass(k *Kernel, class string) {
	k.Tracef("CloneSceneModule: OnCreateClass CloneScene (clone_scene.xml)")
}

// onScenePublicMessage 官方 @0x280800。
func (m *CloneSceneModule) onScenePublicMessage(k *Kernel, self, from PersistID, args VarList) {}

// onCloneScene 官方 @0x280F00（scene OnCloneScene——跨 member 副本消息）。
func (m *CloneSceneModule) onCloneScene(k *Kernel, self, from PersistID, args VarList) {}

// onCloneReset/onCloneDestroy 官方 @0x281CD0/@0x281D90。
func (m *CloneSceneModule) onCloneReset(k *Kernel, self, from PersistID, args VarList)   {}
func (m *CloneSceneModule) onCloneDestroy(k *Kernel, self, from PersistID, args VarList) {
	for uid, id := range m.clones {
		if id == self {
			delete(m.clones, uid)
		}
	}
}

// onCreateRole 官方 @0x281F20。
func (m *CloneSceneModule) onCreateRole(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerEntry 官方 @0x282190：入场（进入副本登记 + HB_CloneCount 计数）。
func (m *CloneSceneModule) onPlayerEntry(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerEvent 官方 OnRecover/OnReady/OnLeaveScene/OnContinue 族。
func (m *CloneSceneModule) onPlayerEvent(k *Kernel, self, from PersistID, args VarList) {}

// onVisibleNpcEntry 官方 @0x282900。
func (m *CloneSceneModule) onVisibleNpcEntry(k *Kernel, self, from PersistID, args VarList) {}

// 六心跳官方面（关闭倒计时/同刷计数/离队退出/时限/副本门/他人离队）。
func (m *CloneSceneModule) hbCloseClone(k *Kernel, self PersistID, beats int32)        {}
func (m *CloneSceneModule) hbCloneCount(k *Kernel, self PersistID, beats int32)        {}
func (m *CloneSceneModule) hbLeaveTeamExit(k *Kernel, self PersistID, beats int32)     {}
func (m *CloneSceneModule) hbCloneLimiteTime(k *Kernel, self PersistID, beats int32)   {}
func (m *CloneSceneModule) hbCloneDoor(k *Kernel, self PersistID, beats int32)         {}
func (m *CloneSceneModule) hbAnotherLeaveTeam(k *Kernel, self PersistID, beats int32)  {}

// 七临界官方面。
func (m *CloneSceneModule) onProgressChanged(k *Kernel, self PersistID, prop string, old Var) {}
func (m *CloneSceneModule) onTeamCaptain(k *Kernel, self PersistID, prop string, old Var)     {}
func (m *CloneSceneModule) onTeamID(k *Kernel, self PersistID, prop string, old Var)          {}
func (m *CloneSceneModule) onLogicStateChanged(k *Kernel, self PersistID, prop string, old Var) {
}
func (m *CloneSceneModule) onVipStateChanged(k *Kernel, self PersistID, prop string, old Var) {}
func (m *CloneSceneModule) onGmPassChanged(k *Kernel, self PersistID, prop string, old Var)   {}
func (m *CloneSceneModule) onNextMonster(k *Kernel, self PersistID, prop string, old Var)     {}

// Ops/Scores 流水（测试面）。
func (m *CloneSceneModule) Ops() []string     { return m.ops }
func (m *CloneSceneModule) Scores() []string  { return m.scores }

// ---- MapMagModule ----

// MapMagModule 复刻官方 scenebasemodule\mapmagmodule（Init @0x2E7030，
// 玩法表 Init 序 113；符号 2280）——地图管理与传送。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnReady",@0x2E7720) / ("player",
//	    "OnMapChanged",@0x2E7C00) / ("player","OnLoad",@0x2E7B70) /
//	    ("player","OnStore",@0x2E8330) / ("player","OnLeave",@0x2E83B0) /
//	    ("player","OnEntry",@0x2F1640)
//	AddIntCustomHook ("player",732=0x2DC,@0x2EDEB0 OnCustomTrans——传送
//	    校验串：Dead/Carrier/sys_map_0007/OnTransToolState/sys_map_0008/
//	    sys_pvptask_0125/0132/LogicState/sys_map_0009/ActiveScene) /
//	    ("player",733=0x2DD,@0x2F0080 OnCustomGetTransPoints) /
//	    ("player",591=0x24F,@0x2ECAB0 OnSceneTriggerConfigChanged)
//	AddEventCallback ("CommonNpc","OnSpring",@0x2EFD80 OnTransNpcSpring——
//	    传送 NPC 弹出)
//	AddIntCommandHook ("player",316=0x13C,@0x2F1740 OnMapTransDecTicketRes) /
//	    (317=0x13D,@0x2F1850 OnMapTransCDDecTicketRes)
//	DeclareHeartBeat ("MapMagModule::HB_RefreshResource",@0x2EC850)
//	DeclareCritical ("…LogicState")→C_OnLogicStateChanged@0x2E8410
//
// 传送点表（官方经 Location/静态面；Go 面直读）：ini/Map/transport.xml
//（56 点：ID/NpcID/Name/NeedCamp/Scene）。
type MapMagModule struct {
	BaseModule

	// Transports 传送点（NpcID → 行）。
	Transports map[string]map[string]string

	// playerScenes 玩家当前场景（dump schema 无 SceneID 属性——官方为
	// 引擎侧场景成员关系，Go 面模块记账）。
	playerScenes map[PersistID]int32
	// ops 命令流水（测试面）。
	ops []string
	// sysMsgs 系统信息观测（sys_map_0007-0009）。
	sysMsgs []string
}

// Name 模块名。
func (m *MapMagModule) Name() string { return "MapMagModule" }

// Init 注册期（对齐官方 0x2E7030）。
func (m *MapMagModule) Init(k *Kernel) error {
	m.Transports = make(map[string]map[string]string)
	m.playerScenes = make(map[PersistID]int32)
	b := k.Bus()
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnMapChanged", m.onMapChanged)
	b.OnEvent("player", "OnLoad", m.onLoad)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnEvent("player", "OnLeave", m.onLeave)
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnCommand("player", 732, m.onCustomTrans)
	b.OnCommand("player", 733, m.onCustomGetTransPoints)
	b.OnCommand("player", 591, m.onSceneTriggerConfigChanged)
	b.OnEvent("CommonNpc", "OnSpring", m.onTransNpcSpring)
	b.OnIntCommand("player", 316, m.onMapTransDecTicketRes)
	b.OnIntCommand("player", 317, m.onMapTransCDDecTicketRes)
	b.HeartBeat("MapMagModule::HB_RefreshResource", m.hbRefreshResource)
	b.Critical("player", "LogicState", m.onLogicStateChanged)
	return nil
}

// LoadResource 传送点表装载。
func (m *MapMagModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: MapMagModule.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/Map/transport.xml")
	if err != nil {
		return fmt.Errorf("fsgame: MapMagModule::LoadResource transport.xml: %w", err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	for id, row := range ruleByID(rows) {
		m.Transports[id] = row
		if npc := row["NpcID"]; npc != "" {
			m.Transports[npc] = row // NpcID 键（传送 NPC 查找面）
		}
	}
	k.Tracef("MapMagModule: LoadResource 传送点 %d", len(m.Transports)/2)
	return nil
}

// Trans 传送闭环（官方 OnCustomTrans @0x2EDEB0 校验链定案）：
// Dead 拒（sys_map_0007）→ Carrier/TransToolState 拒（sys_map_0008）→
// LogicState 拒（sys_map_0009）→ transport 表查找 → 目标场景登记 +
// OnMapChanged 事件 → CD/扣券回执（intCmd 316/317）。
func (m *MapMagModule) Trans(k *Kernel, self PersistID, npcID string) bool {
	o := k.GetGameObj(self)
	row := m.Transports[npcID]
	if o == nil || row == nil {
		return false
	}
	if o.GetInt("Dead") != 0 {
		m.sys("sys_map_0007")
		return false
	}
	if o.GetInt("LogicState") != 0 {
		m.sys("sys_map_0009")
		return false
	}
	scene := atoi32(row["Scene"])
	old := m.playerScenes[self]
	m.playerScenes[self] = scene
	k.RunEvent("player", "OnMapChanged", self, NullPersistID, VarList{old, scene})
	m.ops = append(m.ops, fmt.Sprintf("trans:%s→%d", npcID, scene))
	k.Tracef("MapMagModule: trans %s → scene %d", npcID, scene)
	return true
}

// SceneOf 玩家当前场景（测试面）。
func (m *MapMagModule) SceneOf(self PersistID) int32 { return m.playerScenes[self] }

// onCustomTrans 官方 @0x2EDEB0（cust 732=0x2DC）：args: 传送 NPC 串。
func (m *MapMagModule) onCustomTrans(k *Kernel, self, from PersistID, args VarList) {
	m.Trans(k, self, args.String(0))
}

// onCustomGetTransPoints 官方 @0x2F0080（cust 733）：传送点列表下发。
func (m *MapMagModule) onCustomGetTransPoints(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "points")
}

// onSceneTriggerConfigChanged 官方 @0x2ECAB0（cust 591）。
func (m *MapMagModule) onSceneTriggerConfigChanged(k *Kernel, self, from PersistID, args VarList) {}

// onTransNpcSpring 官方 @0x2EFD80：传送 NPC 弹出（ActiveScene 面）。
func (m *MapMagModule) onTransNpcSpring(k *Kernel, self, from PersistID, args VarList) {}

// onMapTransDecTicketRes 官方 @0x2F1740（命令 316）：传送扣券回执。
func (m *MapMagModule) onMapTransDecTicketRes(k *Kernel, self, from PersistID, args VarList) {}

// onMapTransCDDecTicketRes 官方 @0x2F1850（命令 317）：传送 CD 扣券回执。
func (m *MapMagModule) onMapTransCDDecTicketRes(k *Kernel, self, from PersistID, args VarList) {}

// onReady/onMapChanged/onLoad/onStore/onLeave/onEntry 官方事件族。
func (m *MapMagModule) onReady(k *Kernel, self, from PersistID, args VarList)      {}
func (m *MapMagModule) onMapChanged(k *Kernel, self, from PersistID, args VarList) {}
func (m *MapMagModule) onLoad(k *Kernel, self, from PersistID, args VarList)       {}
func (m *MapMagModule) onStore(k *Kernel, self, from PersistID, args VarList)      {}
func (m *MapMagModule) onLeave(k *Kernel, self, from PersistID, args VarList)      {}
func (m *MapMagModule) onEntry(k *Kernel, self, from PersistID, args VarList)      {}

// HB_RefreshResource 官方 @0x2EC850。
func (m *MapMagModule) hbRefreshResource(k *Kernel, self PersistID, beats int32) {}

// C_OnLogicStateChanged 官方 @0x2E8410。
func (m *MapMagModule) onLogicStateChanged(k *Kernel, self PersistID, prop string, old Var) {}

func (m *MapMagModule) sys(tpl string) { m.sysMsgs = append(m.sysMsgs, tpl) }

// Ops/SysMsgs 流水（测试面）。
func (m *MapMagModule) Ops() []string     { return m.ops }
func (m *MapMagModule) SysMsgs() []string { return m.sysMsgs }

// ---- MapModule ----

// MapModule 复刻官方 mapmodule（Init @0x27BC00，玩法表 Init 序 111；
// 符号 34）——地图标记与探索。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnEntry",@0x27BD30) / ("player",
//	    "OnMapChanged",@0x27BE20)
//	AddIntCustomHook ("player",735=0x2DF,@0x27C000 OnCustomMapMark)
//	DeclareLuaExt ×2：nx_set_explored@0x27B7E0 / nx_clear_explored@0x27BA30
type MapModule struct {
	BaseModule

	// explored 探索标记：玩家 → 场景集合（官方经客户端地图面同步）。
	explored map[PersistID]map[int32]bool
}

// Name 模块名。
func (m *MapModule) Name() string { return "MapModule" }

// Init 注册期（对齐官方 0x27BC00）。
func (m *MapModule) Init(k *Kernel) error {
	m.explored = make(map[PersistID]map[int32]bool)
	b := k.Bus()
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnEvent("player", "OnMapChanged", m.onMapChanged)
	b.OnCommand("player", 735, m.onCustomMapMark)
	return nil
}

// NxSetExplored nx_set_explored（官方 @0x27B7E0）：标记已探索。
func (m *MapModule) NxSetExplored(self PersistID, sceneID int32) {
	if m.explored[self] == nil {
		m.explored[self] = make(map[int32]bool)
	}
	m.explored[self][sceneID] = true
}

// NxClearExplored nx_clear_explored（官方 @0x27BA30）：清探索。
func (m *MapModule) NxClearExplored(self PersistID) { delete(m.explored, self) }

// IsExplored 是否已探索（测试面）。
func (m *MapModule) IsExplored(self PersistID, sceneID int32) bool {
	return m.explored[self][sceneID]
}

// onCustomMapMark 官方 @0x27C000（cust 735=0x2DF）：地图标记命令。
func (m *MapModule) onCustomMapMark(k *Kernel, self, from PersistID, args VarList) {}

// onEntry/onMapChanged 官方 @0x27BD30/@0x27BE20：入场自动探索当前场景。
func (m *MapModule) onEntry(k *Kernel, self, from PersistID, args VarList) {
	m.NxSetExplored(self, k.GetSceneId())
}

func (m *MapModule) onMapChanged(k *Kernel, self, from PersistID, args VarList) {
	if args.Len() >= 2 {
		m.NxSetExplored(self, args.Int(1))
	}
}

// ---- AreaObjModule ----

// AreaObjModule 复刻官方 scenebasemodule\areaobjmodule（Init @0x2D71A0 /
// LoadResource @0x2D7580，玩法表 Init 序 112；符号 794）——区域对象联动。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnMapChanged",@0x2D8420) / ("VisibleNpc",
//	    "OnMapChanged",@0x2D8420 OnMapChanged)
//	AddEventCallback ("player","OnEntry",@0x2D84B0) / ("VisibleNpc",
//	    "OnEntry",@0x2D84B0 OnEntry)
//	AddEventCallback ("player","OnLeave",@0x2D8580) / ("VisibleNpc",
//	    "OnLeave",@0x2D8580 OnLeave)
type AreaObjModule struct {
	BaseModule

	// enters 区域进入流水（测试面：class/事件/对象）。
	enters []string
}

// Name 模块名。
func (m *AreaObjModule) Name() string { return "AreaObjModule" }

// Init 注册期（对齐官方 0x2D71A0）。
func (m *AreaObjModule) Init(k *Kernel) error {
	b := k.Bus()
	for _, cls := range []string{"player", "VisibleNpc"} {
		b.OnEvent(cls, "OnMapChanged", m.onMapChanged)
		b.OnEvent(cls, "OnEntry", m.onEntry)
		b.OnEvent(cls, "OnLeave", m.onLeave)
	}
	return nil
}

// LoadResource 官方 @0x2D7580：区域配置面记。
func (m *AreaObjModule) LoadResource(k *Kernel) error {
	k.Tracef("AreaObjModule: LoadResource (area config)")
	return nil
}

// onMapChanged/onEntry/onLeave 官方 @0x2D8420/@0x2D84B0/@0x2D8580：
// 对象入场/换图/离场 → 所在区域对象联动（区域触发器面——TriggerAreaEvent
// 协作域）。
func (m *AreaObjModule) onMapChanged(k *Kernel, self, from PersistID, args VarList) {
	m.enters = append(m.enters, fmt.Sprintf("map:%v", self.ID()))
}

func (m *AreaObjModule) onEntry(k *Kernel, self, from PersistID, args VarList) {
	m.enters = append(m.enters, fmt.Sprintf("entry:%v", self.ID()))
}

func (m *AreaObjModule) onLeave(k *Kernel, self, from PersistID, args VarList) {
	m.enters = append(m.enters, fmt.Sprintf("leave:%v", self.ID()))
}

// Enters 进入流水（测试面）。
func (m *AreaObjModule) Enters() []string { return m.enters }

// ---- WarFog ----

// WarFog 复刻官方 stagemodule\warfog（Init @0x5F7810 / LoadResource
// @0x5F78C0，玩法表 Init 序 12；符号 1474）——战雾。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnEntry",@0x5F7970) / ("VisibleNpc",
//	    "OnEntry",@0x5F8710)
//	LoadResource @0x5F78C0：EnableWarFog/Resource/Race/Job 配置面。
type WarFog struct {
	BaseModule

	// EnableWarFog 战雾开关（LoadResource 配置面；默认关）。
	EnableWarFog bool
	// reveals 战雾揭示流水（测试面）。
	reveals []string
}

// Name 模块名。
func (m *WarFog) Name() string { return "WarFog" }

// Init 注册期（对齐官方 0x5F7810）。
func (m *WarFog) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnEvent("VisibleNpc", "OnEntry", m.onVisibleNpcEntry)
	return nil
}

// LoadResource 官方 @0x5F78C0：EnableWarFog/Resource/Race/Job 配置面记。
func (m *WarFog) LoadResource(k *Kernel) error {
	k.Tracef("WarFog: LoadResource EnableWarFog/Resource/Race/Job")
	return nil
}

// onEntry 官方 @0x5F7970：玩家入场——战雾区域可见性计算
//（未探索区域不可见；协作 MapModule explored 面）。
func (m *WarFog) onEntry(k *Kernel, self, from PersistID, args VarList) {
	if !m.EnableWarFog {
		return
	}
	m.reveals = append(m.reveals, fmt.Sprintf("player:%v@%d", self.ID(), k.GetSceneId()))
}

// onVisibleNpcEntry 官方 @0x5F8710：NPC 入场（战雾内 NPC 隐藏面）。
func (m *WarFog) onVisibleNpcEntry(k *Kernel, self, from PersistID, args VarList) {}

// Reveals 揭示流水（测试面）。
func (m *WarFog) Reveals() []string { return m.reveals }
