package fsgame

import (
	"fmt"
	"strings"
)

// TeamModule 复刻官方 teammodule（Init @0x82A290 / LoadResource @0x840450，
// 玩法表 Init 序 182；符号 1506，53 命令）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallbackInt ×4（CloneScene 类族 int 事件 1/16/17/18：
//	    @0x82FFC0/@0x82F6A0/@0x82E9E0/@0x82E0C0——副本场景进入/销毁联动）
//	AddEventCallback ("player","OnStore",@0x82C920)     TeamID/TeamType/
//	    TeamUniqueID/TeamSceneSign 存档 + TeamBeRequest 清理（sys_request_0009）
//	AddEventCallback ("player","OnRecover",@0x82CDA0)   队伍登记恢复（team_rec 重建）
//	AddEventCallback ("player","OnReady",@0x82D4A0)     队伍数据同步下发
//	AddEventCallback ("scene","OnPublicMessage",@0x82D590) 跨 member 队伍消息
//	AddIntCustomHook ×31（0x12C-0x149/0x1B9-0x1C2，玩家操作族，见 teamCmdNames）
//	AddIntCommandHook ×20（0x6D-0x84/0x313，结果/通知下发族）
//	DeclareCritical ×8（prop 定案）：Photo→C_OnPhotoChanged@0x8398B0 /
//	    HPRatio→C_OnHPRatioChanged@0x839990（MPRatio/EYRatio/AVRatio/RCRatio/
//	    FSRatio 同挂）/ EGRatio→C_OnEGRatioChanged@0x839A40 / Level→
//	    C_OnLevelChanged@0x839B90 / Job→C_OnJobChanged@0x839C70 / LogicState→
//	    C_OnLogicStateChange@0x839D50 / Camp→C_OnCampChange@0x839E30 / CurArea→
//	    C_OnCurAreaChange@0x839F10
//	DeclareHeartBeat ×6：HB_CheckOffLineMember@0x83D4D0 / HB_ReadyAffirmWait@
//	    0x83E3D0 / HB_CheckSummonTime@0x842080 / HB_UpdateMemberPos@0x83ED40 /
//	    HB_UpdateHPRatio@0x83EE20 / HB_UpdateEGRatio@0x83EEE0
//	DeclareRecHook（buffer_rec）→ C_OnBufferRecordChanged@0x83CFE0（team_rec[8]）
//	DeclareLuaExt ×6：nx_is_in_team/nx_is_team_captain/nx_is_team_mate/
//	    nx_get_team_pick_info/nx_get_team_pick_roll/nx_get_captain
//
// LoadResource：ini/Team/summon_cost.xml（副本门召唤消耗：USEcash/blood/
// item_troops|team_%d）。
// 记录（struct/player/team.xml）：team_rec(21) / team_recruit_info_rec(5) /
// team_recruit_rec(11)；属性 TeamID/TeamCaptain/TeamPickMode/TeamPickQuality/
// TeamType/TeamUniqueID/IsReady/CurMembers 等。
// 系统信息：sys_team_00xx 族（BG_systeminfo 模板）。
type TeamModule struct {
	BaseModule

	teams     map[int32]*Team
	nextID    int32
	uniqueSeq int64

	// SummonCost summon_cost.xml（USE → 消耗类型；troops|team_× 费用项）。
	SummonCost map[string]map[string]string

	// ops 命令流水（测试面；官方经 0x6D-0x84 结果命令下发）。
	ops []string
	// sysMsgs 系统信息观测（sys_team_* 模板号）。
	sysMsgs []string
}

// Team 队伍运行态（官方为成员间同步的 team_rec + TeamID 族属性；
// Go 面集中管理、镜像下发）。
type Team struct {
	ID          int32
	Captain     string
	Members     []string // 有序（队长首）
	Type        int32    // 0 普通队 1 团队
	PickMode    int32    // 拾取分配模式
	PickQuality int32    // 拾取分配品质
	PickRoll    int32    // 轮流序号
	Notice      string   // 招募告示
	UniqueID    string
}

// team_rec 列序（struct/player/team.xml 定案）。
const (
	trName = iota // 0 队员名(widestr)
	trPhoto       // 1 Photo
	trScene       // 2 场景编号
	trX           // 3 X
	trY           // 4 Y
	trZ           // 5 Z
	trHPRatio     // 6 HPRatio
	trEGRatio     // 7 EGRatio
	trBuffers     // 8 Buffers
	trFighting    // 9 IsFighting
	trPos         // 10 队伍中位置
	trOffice      // 11 队伍的职位
	trOffline     // 12 上次离线时间
	trMark        // 13 标记字符串
	trReady       // 14 准备确认
	trLevel       // 15 等级
	trJob         // 16 职业
	trCamp        // 17 Camp
	trArea        // 18 玩家所在区域
	trMaxHP       // 19 MaxHP
	trMaxEG       // 20 MaxEG
)

// teamCustomCmds 玩家操作族（AddIntCustomHook；命令号 → 官方处理器名）。
var teamCustomCmds = map[int32]string{
	0x12C: "SmlTeamLeave", 0x12D: "ChangeRequest", 0x12E: "SummonAnswer",
	0x130: "DeleteTeamSceneSign", 0x133: "ChangeLeader", 0x134: "SetMember",
	0x135: "SetTeamType", 0x136: "CreateTeam", 0x137: "KickOut", 0x138: "Leave",
	0x139: "ChangeCaptain", 0x13A: "SetPickQuality", 0x13B: "SetPickMode",
	0x13C: "RequestJoin", 0x13D: "RefreshInfo", 0x13E: "AddInfo",
	0x13F: "RemoveInfo", 0x146: "SetSign", 0x147: "TeamNoticeSet",
	0x148: "SceneSignSet", 0x149: "Summon", 0x1B9: "TeamInfoGet",
	0x1BA: "AddPlayer", 0x1BB: "RemovePlayer", 0x1BC: "Destroy",
	0x1BD: "Invite", 0x1BE: "ReadyAffirm", 0x1BF: "RequestSet",
	0x1C0: "InviteSmlTeamAffirm", 0x1C1: "JoinSmlTeamAffirm", 0x1C2: "FollowSmlTeamLeave",
}

// teamCmdCmds 结果/通知族（AddIntCommandHook）。
var teamCmdCmds = map[int32]string{
	0x6D: "TeamTypeChange", 0x6E: "CreateTeamResult", 0x6F: "JoinTeam",
	0x70: "AddMember", 0x71: "RemoveMember", 0x72: "ClearTeam",
	0x76: "UpdateMemberInfo", 0x77: "UpdateTeamInfo", 0x78: "AddRecruitInfo",
	0x79: "RemoveRecruitInfo", 0x7A: "RequestJoin", 0x7B: "RequestJoinReturn",
	0x7C: "TeamMemberChange", 0x7D: "TeamMemberRecover", 0x7E: "SetTeamMemberListen",
	0x7F: "RequestInviteTeam", 0x80: "RequestInviteReturn", 0x81: "ChangeRequest",
	0x83: "CanBeSummon", 0x84: "CanBeSummonAnswer", 0x313: "TeamTransDecTicketRes",
}

// Name 模块名。
func (m *TeamModule) Name() string { return "TeamModule" }

// Init 注册期（对齐官方 0x82A290；53 命令全量登记）。
func (m *TeamModule) Init(k *Kernel) error {
	m.teams = make(map[int32]*Team)
	m.nextID = 1
	b := k.Bus()
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("scene", "OnPublicMessage", m.onPublicMessage)
	for cmd := range teamCustomCmds {
		b.OnCommand("player", cmd, m.onCustomOp)
	}
	for cmd := range teamCmdCmds {
		b.OnIntCommand("player", cmd, m.onCmdResult)
	}
	for _, prop := range []string{"Photo", "HPRatio", "MPRatio", "EYRatio", "AVRatio", "RCRatio", "FSRatio", "EGRatio", "Level", "Job", "LogicState", "Camp", "CurArea"} {
		p := prop
		b.Critical("player", p, m.onMemberPropChange)
	}
	b.HeartBeat("TeamModule::HB_CheckOffLineMember", m.hbCheckOffLineMember)
	b.HeartBeat("TeamModule::HB_ReadyAffirmWait", m.hbReadyAffirmWait)
	b.HeartBeat("TeamModule::HB_CheckSummonTime", m.hbCheckSummonTime)
	b.HeartBeat("TeamModule::HB_UpdateMemberPos", m.hbUpdateMemberPos)
	b.HeartBeat("TeamModule::HB_UpdateHPRatio", m.hbUpdateHPRatio)
	b.HeartBeat("TeamModule::HB_UpdateEGRatio", m.hbUpdateEGRatio)
	b.RecHook("buffer_rec", m.onBufferRecordChanged)
	return nil
}

// LoadResource 官方 @0x840450：ini/Team/summon_cost.xml。
func (m *TeamModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: TeamModule.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/Team/summon_cost.xml")
	if err != nil {
		return fmt.Errorf("fsgame: CampModule::LoadResource: No found file: %w", err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	m.SummonCost = ruleByID(rows)
	k.Tracef("TeamModule: LoadResource summon_cost (%d 项)", len(m.SummonCost))
	return nil
}

// TeamOf 玩家所在队伍（无队 nil）。
func (m *TeamModule) TeamOf(k *Kernel, self PersistID) *Team {
	o := k.GetGameObj(self)
	if o == nil {
		return nil
	}
	id := o.GetInt("TeamID")
	return m.teams[id]
}

// CreateTeam 官方 OnCustomCreateTeam @0x830630（0x136）：
// 已在队拒（0001）→ 建队 → 队长属性链（TeamID/TeamType/TeamUniqueID/
// TeamCaptain/CurMembers）→ team_rec 自行 → 结果 0x6E。
func (m *TeamModule) CreateTeam(k *Kernel, self PersistID, teamType int32) *Team {
	o := k.GetGameObj(self)
	if o == nil {
		return nil
	}
	if o.GetInt("TeamID") != 0 {
		m.sys("sys_team_0001")
		return nil
	}
	name, _ := o.Get("name").(string)
	t := &Team{
		ID:      m.nextID,
		Captain: name,
		Members: []string{name},
		Type:    teamType,
	}
	m.nextID++
	t.UniqueID = fmt.Sprintf("TEAM-%d-%d", t.ID, m.uniqueSeq)
	m.uniqueSeq++
	m.teams[t.ID] = t
	_ = k.SetInt(self, "TeamID", t.ID)
	_ = k.SetInt(self, "TeamType", teamType)
	_ = k.SetString(self, "TeamUniqueID", t.UniqueID)
	_ = k.SetString(self, "TeamCaptain", name)
	_ = k.SetInt(self, "CurMembers", 1)
	m.syncTeamRec(k, t)
	m.op("CreateTeam")
	k.Tracef("TeamModule: create team %d by %s", t.ID, name)
	return t
}

// Invite 官方 OnCustomInvite @0x830F70（0x1BD）：队长权（0002）→ 队伍满
//（0003/0014）→ 已是成员（0004）→ 不在线（0005/0008）→ 敌对阵营（0009）→
// 在他队（0010）→ 邀请请求下发（0x7F）。
func (m *TeamModule) Invite(k *Kernel, self PersistID, target string) bool {
	t := m.TeamOf(k, self)
	if t == nil {
		return false
	}
	o := k.GetGameObj(self)
	name, _ := o.Get("name").(string)
	if t.Captain != name {
		m.sys("sys_team_0002")
		return false
	}
	if len(t.Members) >= 25 {
		m.sys("sys_team_0003")
		return false
	}
	if tid := k.SeekRoleName(target); tid.Valid() {
		if tt := m.TeamOf(k, tid); tt != nil {
			m.sys("sys_team_0010")
			return false
		}
	} else {
		m.sys("sys_team_0005")
		return false
	}
	m.op("Invite:" + target)
	return true
}

// JoinTeam 官方 OnCustomJoinSmlTeamAffirm 族（0x1C1/0x137 KickOut 相对）：
// 入队闭环——队长在场/队伍在/容量门（0013/0015）→ 入队 → 全员 team_rec 同步 →
// 0x70 AddMember 通知。
func (m *TeamModule) JoinTeam(k *Kernel, self PersistID, teamID int32) bool {
	t := m.teams[teamID]
	if t == nil {
		return false
	}
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	name, _ := o.Get("name").(string)
	if o.GetInt("TeamID") != 0 {
		m.sys("sys_team_0001")
		return false
	}
	if len(t.Members) >= 25 {
		m.sys("sys_team_0015")
		return false
	}
	t.Members = append(t.Members, name)
	_ = k.SetInt(self, "TeamID", t.ID)
	_ = k.SetInt(self, "TeamType", t.Type)
	_ = k.SetString(self, "TeamUniqueID", t.UniqueID)
	_ = k.SetString(self, "TeamCaptain", t.Captain)
	for _, mem := range t.Members {
		if mid := k.SeekRoleName(mem); mid.Valid() {
			_ = k.SetInt(mid, "CurMembers", int32(len(t.Members)))
		}
	}
	m.syncTeamRec(k, t)
	m.op("Join:" + name)
	return true
}

// LeaveTeam 官方 OnCustomLeave @0x832090（0x138）：出队 → 队员则重算
//（队长离队 → 移交首位在线队员，全空解散 0x1BC Destroy）。
func (m *TeamModule) LeaveTeam(k *Kernel, self PersistID) bool {
	t := m.TeamOf(k, self)
	if t == nil {
		return false
	}
	o := k.GetGameObj(self)
	name, _ := o.Get("name").(string)
	m.dropMember(k, t, name)
	m.op("Leave:" + name)
	return true
}

// KickOut 官方 OnCustomKickOut @0x830D20（0x137）：队长权（0026）→ 踢出。
func (m *TeamModule) KickOut(k *Kernel, self PersistID, target string) bool {
	t := m.TeamOf(k, self)
	if t == nil {
		return false
	}
	o := k.GetGameObj(self)
	name, _ := o.Get("name").(string)
	if t.Captain != name {
		m.sys("sys_team_0026")
		return false
	}
	if t.Captain == target {
		return false // 队长不可自踢
	}
	m.dropMember(k, t, target)
	m.op("Kick:" + target)
	return true
}

// ChangeCaptain 官方 OnCustomChangeCaptain @0x832B60（0x139）：移交队长
//（0031 模板；目标不在队 sys_team_0033 语义）。
func (m *TeamModule) ChangeCaptain(k *Kernel, self PersistID, target string) bool {
	t := m.TeamOf(k, self)
	if t == nil {
		return false
	}
	o := k.GetGameObj(self)
	name, _ := o.Get("name").(string)
	if t.Captain != name {
		m.sys("sys_team_0026")
		return false
	}
	if tid := k.SeekRoleName(target); !tid.Valid() || m.TeamOf(k, tid) != t {
		m.sys("sys_team_0033")
		return false
	}
	t.Captain = target
	// 重排：队长置首。
	for i, mn := range t.Members {
		if mn == target {
			t.Members = append([]string{target}, append(t.Members[:i:i], t.Members[i+1:]...)...)
			break
		}
	}
	for _, mem := range t.Members {
		if mid := k.SeekRoleName(mem); mid.Valid() {
			_ = k.SetString(mid, "TeamCaptain", target)
		}
	}
	m.syncTeamRec(k, t)
	m.sys("sys_team_0031")
	m.op("Captain:" + target)
	return true
}

// SetPickMode/SetPickQuality 官方 0x13B/0x13A：拾取分配（战斗中拒改 0038/0039）。
func (m *TeamModule) SetPickMode(k *Kernel, self PersistID, mode int32) bool {
	t := m.TeamOf(k, self)
	if t == nil {
		return false
	}
	t.PickMode = mode
	for _, mem := range t.Members {
		if mid := k.SeekRoleName(mem); mid.Valid() {
			_ = k.SetInt(mid, "TeamPickMode", mode)
		}
	}
	m.op("PickMode")
	return true
}

func (m *TeamModule) SetPickQuality(k *Kernel, self PersistID, quality int32) bool {
	t := m.TeamOf(k, self)
	if t == nil {
		return false
	}
	t.PickQuality = quality
	for _, mem := range t.Members {
		if mid := k.SeekRoleName(mem); mid.Valid() {
			_ = k.SetInt(mid, "TeamPickQuality", quality)
		}
	}
	m.op("PickQuality")
	return true
}

// SetReady 官方 0x1BE ReadyAffirm：准备确认（team_rec[14] 列）。
func (m *TeamModule) SetReady(k *Kernel, self PersistID, ready int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	_ = k.SetInt(self, "IsReady", ready)
	m.op("Ready")
}

// dropMember 出队/踢出公共面：成员表摘除、属性清零；队长离队移交、
// 空队解散（Destroy）、全员 CurMembers 重算与 team_rec 重同步。
func (m *TeamModule) dropMember(k *Kernel, t *Team, name string) {
	idx := -1
	for i, mn := range t.Members {
		if mn == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	t.Members = append(t.Members[:idx:idx], t.Members[idx+1:]...)
	if mid := k.SeekRoleName(name); mid.Valid() {
		_ = k.SetInt(mid, "TeamID", 0)
		_ = k.SetInt(mid, "CurMembers", 0)
		_ = k.SetInt(mid, "IsReady", 0)
		if r := k.GetGameObj(mid).Record("team_rec"); r != nil {
			r.Clear()
		}
	}
	if len(t.Members) == 0 {
		delete(m.teams, t.ID)
		m.op("Destroy")
		return
	}
	if t.Captain == name {
		t.Captain = t.Members[0]
		for _, mem := range t.Members {
			if mid := k.SeekRoleName(mem); mid.Valid() {
				_ = k.SetString(mid, "TeamCaptain", t.Captain)
			}
		}
	}
	for _, mem := range t.Members {
		if mid := k.SeekRoleName(mem); mid.Valid() {
			_ = k.SetInt(mid, "CurMembers", int32(len(t.Members)))
		}
	}
	m.syncTeamRec(k, t)
}

// syncTeamRec 队伍镜像到全体成员的 team_rec（每成员一行，private=true 面）。
func (m *TeamModule) syncTeamRec(k *Kernel, t *Team) {
	for _, mem := range t.Members {
		mid := k.SeekRoleName(mem)
		if !mid.Valid() {
			continue
		}
		o := k.GetGameObj(mid)
		r := o.Record("team_rec")
		if r == nil {
			continue
		}
		r.Clear()
		for _, mn := range t.Members {
			row := make(VarList, r.Def().Cols)
			row[trName] = mn
			row[trPos] = int32(0)
			if mn == t.Captain {
				row[trOffice] = int32(1) // 队长职位
			}
			if tid := k.SeekRoleName(mn); tid.Valid() {
				to := k.GetGameObj(tid)
				row[trLevel] = to.GetInt("Level")
				row[trJob] = to.GetInt("Job")
				row[trScene] = fmt.Sprintf("%d", to.GetInt("SceneID"))
			}
			r.AppendRow(row)
		}
	}
}

// onCustomOp 玩家操作族统一入口（官方各命令独立处理器；Go 面按命令名分派）。
func (m *TeamModule) onCustomOp(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	cmd := args.Int(0)
	name, _ := o.Get("name").(string)
	switch cmd {
	case 0x136: // CreateTeam（args: 队类型）
		m.CreateTeam(k, self, args.Int(1))
	case 0x1BD: // Invite（args: 目标名）
		m.Invite(k, self, args.String(1))
	case 0x1C1: // JoinSmlTeamAffirm（args: 队ID）
		m.JoinTeam(k, self, args.Int(1))
	case 0x138: // Leave
		m.LeaveTeam(k, self)
	case 0x137: // KickOut（args: 目标名）
		m.KickOut(k, self, args.String(1))
	case 0x139: // ChangeCaptain
		m.ChangeCaptain(k, self, args.String(1))
	case 0x13B: // SetPickMode
		m.SetPickMode(k, self, args.Int(1))
	case 0x13A: // SetPickQuality
		m.SetPickQuality(k, self, args.Int(1))
	case 0x1BE: // ReadyAffirm
		m.SetReady(k, self, args.Int(1))
	case 0x1BC: // Destroy：队长解散
		if t := m.TeamOf(k, self); t != nil && t.Captain == name {
			for _, mem := range append([]string(nil), t.Members...) {
				if mid := k.SeekRoleName(mem); mid.Valid() {
					m.dropMember(k, t, mem)
				}
			}
		}
	case 0x147: // TeamNoticeSet
		if t := m.TeamOf(k, self); t != nil && t.Captain == name {
			t.Notice = args.String(1)
			m.op("Notice")
		}
	default:
		m.op(teamCustomCmds[cmd])
	}
}

// onCmdResult 结果/通知族（服务端 → 客户端下发面；Go 侧记流水）。
func (m *TeamModule) onCmdResult(k *Kernel, self, from PersistID, args VarList) {
	m.op("notify:" + teamCmdCmds[args.Int(0)])
}

// onStore 官方 @0x82C920：TeamID/TeamType/TeamUniqueID/TeamSceneSign 落档
//（save=true 属性经 Archive 面）+ TeamBeRequest 清理。
func (m *TeamModule) onStore(k *Kernel, self, from PersistID, args VarList) {}

// onRecover 官方 @0x82CDA0：按 TeamID 归队（team_rec 重建 + HB 登记）。
func (m *TeamModule) onRecover(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if t := m.teams[o.GetInt("TeamID")]; t != nil {
		m.syncTeamRec(k, t)
	}
}

// onReady 官方 @0x82D4A0：队伍数据全量同步（team_rec 下发）。
func (m *TeamModule) onReady(k *Kernel, self, from PersistID, args VarList) {
	if t := m.TeamOf(k, self); t != nil {
		m.syncTeamRec(k, t)
	}
}

// onPublicMessage 官方 @0x82D590：跨 member 队伍消息（跨服组队面）。
func (m *TeamModule) onPublicMessage(k *Kernel, self, from PersistID, args VarList) {}

// onMemberPropChange 官方 C_On*Changed 族：成员属性变更 → 全员 team_rec
// 对应列刷新（Photo[1]/HPRatio[6]/EGRatio[7]/Level[15]/Job[16]/Camp[17]/
// CurArea[18]）。
func (m *TeamModule) onMemberPropChange(k *Kernel, self PersistID, prop string, old Var) {
	t := m.TeamOf(k, self)
	if t == nil {
		return
	}
	o := k.GetGameObj(self)
	name, _ := o.Get("name").(string)
	col := -1
	var v Var
	switch prop {
	case "Photo":
		col, v = trPhoto, o.Get("Photo")
	case "HPRatio", "MPRatio", "EYRatio", "AVRatio", "RCRatio", "FSRatio":
		col, v = trHPRatio, o.Get(prop)
	case "EGRatio":
		col, v = trEGRatio, o.Get(prop)
	case "Level":
		col, v = trLevel, o.Get("Level")
	case "Job":
		col, v = trJob, o.Get("Job")
	case "Camp":
		col, v = trCamp, o.Get("Camp")
	case "CurArea":
		col, v = trArea, o.Get("CurArea")
	case "LogicState":
		col, v = trFighting, o.Get("LogicState")
	}
	if col < 0 {
		return
	}
	for _, mem := range t.Members {
		mid := k.SeekRoleName(mem)
		if !mid.Valid() {
			continue
		}
		if r := k.GetGameObj(mid).Record("team_rec"); r != nil {
			if i := rowByName(r, name); i >= 0 {
				row := r.Row(i)
				row[col] = v
				r.ModifyRow(i, row)
			}
		}
	}
}

// hbCheckOffLineMember 官方 @0x83D4D0：离线成员巡检——team_rec[12] 上次离线
// 时间回填，超时成员移除。
func (m *TeamModule) hbCheckOffLineMember(k *Kernel, self PersistID, beats int32) {
	t := m.TeamOf(k, self)
	if t == nil {
		return
	}
	for _, mem := range append([]string(nil), t.Members...) {
		if k.SeekRoleName(mem).Valid() {
			continue
		}
		if r := k.GetGameObj(self).Record("team_rec"); r != nil {
			if i := rowByName(r, mem); i >= 0 {
				row := r.Row(i)
				row[trOffline] = int64(unixNow())
				r.ModifyRow(i, row)
			}
		}
	}
}

// hbReadyAffirmWait 官方 @0x83E3D0：准备确认等待超时。
func (m *TeamModule) hbReadyAffirmWait(k *Kernel, self PersistID, beats int32) {}

// hbCheckSummonTime 官方 @0x842080：召唤时限（summon_cost 消耗面）。
func (m *TeamModule) hbCheckSummonTime(k *Kernel, self PersistID, beats int32) {}

// hbUpdateMemberPos 官方 @0x83ED40：成员位置同步（team_rec[3..5]）。
func (m *TeamModule) hbUpdateMemberPos(k *Kernel, self PersistID, beats int32) {
	t := m.TeamOf(k, self)
	if t == nil {
		return
	}
	o := k.GetGameObj(self)
	name, _ := o.Get("name").(string)
	for _, mem := range t.Members {
		mid := k.SeekRoleName(mem)
		if !mid.Valid() {
			continue
		}
		if r := k.GetGameObj(mid).Record("team_rec"); r != nil {
			if i := rowByName(r, name); i >= 0 {
				row := r.Row(i)
				row[trX] = o.Get("PosX")
				row[trY] = o.Get("PosY")
				row[trZ] = o.Get("PosZ")
				r.ModifyRow(i, row)
			}
		}
	}
}

// hbUpdateHPRatio 官方 @0x83EE20：HP 比例同步（team_rec[6]）。
func (m *TeamModule) hbUpdateHPRatio(k *Kernel, self PersistID, beats int32) {
	m.onMemberPropChange(k, self, "HPRatio", nil)
}

// hbUpdateEGRatio 官方 @0x83EEE0：EG 比例同步（team_rec[7]）。
func (m *TeamModule) hbUpdateEGRatio(k *Kernel, self PersistID, beats int32) {
	m.onMemberPropChange(k, self, "EGRatio", nil)
}

// C_OnBufferRecordChanged 官方 @0x83CFE0（buffer_rec 钩子）：队员 Buff 变更
// → team_rec[8] Buffers 串同步。
func (m *TeamModule) onBufferRecordChanged(k *Kernel, self PersistID, record string, op, row int32) {
	t := m.TeamOf(k, self)
	if t == nil {
		return
	}
	o := k.GetGameObj(self)
	name, _ := o.Get("name").(string)
	if r := o.Record("buffer_rec"); r != nil {
		var bufs []string
		for i := 0; i < r.Rows(); i++ {
			bufs = append(bufs, r.Row(i).String(0))
		}
		for _, mem := range t.Members {
			mid := k.SeekRoleName(mem)
			if !mid.Valid() {
				continue
			}
			if tr := k.GetGameObj(mid).Record("team_rec"); tr != nil {
				if i := rowByName(tr, name); i >= 0 {
					row := tr.Row(i)
					row[trBuffers] = strings.Join(bufs, ";")
					tr.ModifyRow(i, row)
				}
			}
		}
	}
}

// ---- Lua 扩展面（官方 nx_* 六函数的 Go 等价；DeclareLuaExt 注册）----

// NxIsInTeam nx_is_in_team：是否在队。
func (m *TeamModule) NxIsInTeam(k *Kernel, self PersistID) bool { return m.TeamOf(k, self) != nil }

// NxIsTeamCaptain nx_is_team_captain：是否队长。
func (m *TeamModule) NxIsTeamCaptain(k *Kernel, self PersistID) bool {
	t := m.TeamOf(k, self)
	if t == nil {
		return false
	}
	o := k.GetGameObj(self)
	name, _ := o.Get("name").(string)
	return t.Captain == name
}

// NxIsTeamMate nx_is_team_mate：两人是否同队。
func (m *TeamModule) NxIsTeamMate(k *Kernel, self PersistID, other PersistID) bool {
	t1, t2 := m.TeamOf(k, self), m.TeamOf(k, other)
	return t1 != nil && t1 == t2
}

// NxGetTeamPickInfo nx_get_team_pick_info：拾取模式/品质。
func (m *TeamModule) NxGetTeamPickInfo(k *Kernel, self PersistID) (mode, quality int32) {
	if t := m.TeamOf(k, self); t != nil {
		return t.PickMode, t.PickQuality
	}
	return 0, 0
}

// NxGetTeamPickRoll nx_get_team_pick_roll：轮流序号。
func (m *TeamModule) NxGetTeamPickRoll(k *Kernel, self PersistID) int32 {
	if t := m.TeamOf(k, self); t != nil {
		return t.PickRoll
	}
	return 0
}

// NxGetCaptain nx_get_captain：队长名。
func (m *TeamModule) NxGetCaptain(k *Kernel, self PersistID) string {
	if t := m.TeamOf(k, self); t != nil {
		return t.Captain
	}
	return ""
}

func (m *TeamModule) op(s string)      { m.ops = append(m.ops, s) }
func (m *TeamModule) sys(tpl string)   { m.sysMsgs = append(m.sysMsgs, tpl) }

// Ops 命令流水（测试面）。
func (m *TeamModule) Ops() []string { return m.ops }

// SysMsgs 系统信息流水（测试面）。
func (m *TeamModule) SysMsgs() []string { return m.sysMsgs }

// Teams 队伍快照（测试面）。
func (m *TeamModule) Teams() map[int32]*Team { return m.teams }
