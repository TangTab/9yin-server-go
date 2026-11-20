package fsgame

import (
	"fmt"
	"strings"
)

// FriendModule 复刻官方 interactivemodule/friendmodule
// （Init @0x85F390 / LoadResource @0x85FA90，玩法表 Init 序 183；符号 742）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	DeclareLuaExt    ("nx_set_friend_value",@0x85F140)
//	AddEventCallback ("scene","OnGetRoleInfo",@0x863D80)  跨服角色信息回执
//	AddEventCallback ("player","OnLoad",@0x866F80)
//	AddEventCallback ("player","OnRecover",@0x867400)     对象级挂载（见下）
//	AddEventCallback ("player","OnStore",@0x868110)
//	AddIntCustomHook ("player",110=0x6E,@0x86B650)        OnCustomClientMessage
//	AddIntCommandHook ("player",400=0x190,@0x86B160)      OnCustomServerMessage
//	AddIntCommandHook ("player",868,@0x86F5D0)            OnUpdateEachFriendValue
//	AddIntCommandHook ("player",869,@0x86F750)            OnCheckFriendValueUpdate
//	AddIntCommandHook ("player",864,@0x86F490)            OnUpdateFriendValue
//	AddIntCommandHook ("player",706,@0x871570)            OnCommandRefreshRecData
//	AddIntCommandHook ("player",312=0x138,@0x873730)      OnTransDecTicketRes
//	AddIntCommandHook ("player",616=0x268,@0x8610C0)      OnPlayerBeKilled（仇人面）
//	AddIntCommandHook ("AttackNpc",616,@0x8614A0)         OnCommandBeKill
//	DeclareHeartBeat ×5：HB_CheckPlayerFriend@0x866E00 / HB_CheckPlayerBlackList
//	                    @0x866E30 / HB_CheckTeamID@0x8609B0 / HB_CheckOnLine@0x860C60
//	                    / HB_CheckExplore@0x860E30
//	DeclareCritical ×4（OnRecover 对象级挂载，prop 定案）：
//	    Level→C_OnBGLevelChange@0x8601B0 / Job→C_OnBGJobChange@0x8603A0 /
//	    ArrestValue→C_OnArrestValueChange@0x860590 / CurArea→C_OnCurAreaChange@0x860780
//	DeclareRecHook ×2：OnTeamRecChange@0x8718D0（team_rec，好友所在队伍人数）/
//	                    OnOriginRecChange@0x871A40（Origin_Record）
//
// OnCustomClientMessage（0x86B650）子命令跳表 @0x1086B810：
//
//	0 OpenFriend / 1 AddFriend / 2 DelFriend / 3 AddUnfamiliar / 4 DelUnfamiliar /
//	5 AddBlackList / 6 DelBlackList / 7 AddEnemy / 8 DelEnemy / 9 SeeAbout /
//	11 GMAddFriend / 12 RequstSeeAboutNeed
//
// LoadResource：ini/Rule/friendship.xml + ini/Rule/friendvalueevent.xml +
// ini/Rule/enemyexpend.xml。
// 记录（struct/player/friend.xml）：friend_rec(19) / friendvalueevent_rec(21) /
// unfamiliar_rec(8) / blacklist_rec(7) / enemy_rec(8)。
// 系统信息：sys_friend_00xx 族（BG_systeminfo 模板，见 docs）。
type FriendModule struct {
	BaseModule

	// Friendship friendship.xml 主规则（好友度上限/行为上限/等级阈值）。
	Friendship *FriendshipRule
	// FVEvents friendvalueevent.xml 行为奖励表（ID → 规则）。
	FVEvents map[string]FVEventRule
	// EnemyExpend enemyexpend.xml 仇人追踪消耗（USEcash/USEblood/USEtroops）。
	EnemyExpend map[string]string

	// sysMsgs 系统信息观测（sys_friend_* 模板号）。
	sysMsgs []string
	// valueBumps 好友度变更流水（目标, 增量, 行为）。
	valueBumps []fvBump
}

// FriendshipRule friendship.xml <Property ID="FriendValue">。
type FriendshipRule struct {
	MaxValue         int32   // 好友度上限
	FVOlineMax       int32   // 在线行为最大值
	FVChatMax        int32   // 聊天行为最大值
	FVTeamMax        int32   // 组队行为最大值
	FVFunMax         int32   // 娱乐行为最大值
	FVPresentMax     int32   // 送礼行为最大值
	FVPVPMax         int32   // PVP 行为最大值
	OnlineTime       int32   // 在线累计时间（分钟）
	ExploreTime      int32   // 探索行为累计时间
	RidingTime       int32   // 坐骑累计时间
	KillCount        int32   // 组队杀怪个数
	FriendshipLevels []int32 // FriendshipLevel 阈值（逗号串）
}

// FVEventRule friendvalueevent.xml <Property ID="FVOnline" …>。
type FVEventRule struct {
	IncreaseTime  int32 // 行为的时间间隔
	IncreaseCount int32 // 行为的次数
	IncreaseValue int32 // 行为的点数
}

type fvBump struct {
	Target string
	Delta  int32
	Event  string
}

// friendRecCols friend_rec 列序（struct/player/friend.xml 定案）。
const (
	frName = iota // 0 好友名称(widestr)
	frLevel       // 1 好友等级
	frJob         // 2 好友职业
	frOnline      // 3 是否在线
	frScene       // 4 所在场景
	frTeamSize    // 5 所在队伍人数
	frValue       // 6 好友度
	frRelation    // 7 良性关系枚举
	frUID         // 8 好友UID
	frGuildTitle  // 9 好友门派身份
	frArrest      // 10 通缉值
	frKillCount   // 11 共同杀怪数
	frKillTime    // 12 计数有效时间
	frJoinTime    // 13 入队时间
	frExploreTime // 14 开始探索时间
	frRideTime    // 15 乘坐坐骑累计时间
	frArea        // 16 玩家所在区域
)

// Name 模块名。
func (m *FriendModule) Name() string { return "FriendModule" }

// Init 注册期（对齐官方 0x85F390）。
func (m *FriendModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("scene", "OnGetRoleInfo", m.onGetRoleInfo)
	b.OnEvent("player", "OnLoad", func(k *Kernel, self, from PersistID, args VarList) {})
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnCommand("player", 110, m.onCustomClientMessage)
	b.OnIntCommand("player", 400, m.onCustomServerMessage)
	b.OnIntCommand("player", 868, m.onUpdateEachFriendValue)
	b.OnIntCommand("player", 869, m.onCheckFriendValueUpdate)
	b.OnIntCommand("player", 864, m.onUpdateFriendValue)
	b.OnIntCommand("player", 706, m.onCommandRefreshRecData)
	b.OnIntCommand("player", 312, m.onTransDecTicketRes)
	b.OnIntCommand("player", 616, m.onPlayerBeKilled)
	b.OnIntCommand("AttackNpc", 616, m.onCommandBeKill)
	b.HeartBeat("FriendModule::HB_CheckPlayerFriend", m.hbCheckPlayerFriend)
	b.HeartBeat("FriendModule::HB_CheckPlayerBlackList", m.hbCheckPlayerBlackList)
	b.HeartBeat("FriendModule::HB_CheckTeamID", m.hbCheckTeamID)
	b.HeartBeat("FriendModule::HB_CheckOnLine", m.hbCheckOnLine)
	b.HeartBeat("FriendModule::HB_CheckExplore", m.hbCheckExplore)
	b.Critical("player", "Level", m.onBGLevelChange)
	b.Critical("player", "Job", m.onBGJobChange)
	b.Critical("player", "ArrestValue", m.onArrestValueChange)
	b.Critical("player", "CurArea", m.onCurAreaChange)
	b.RecHook("team_rec", m.onTeamRecChange)
	b.RecHook("Origin_Record", m.onOriginRecChange)
	return nil
}

// LoadResource 官方 @0x85FA90：ini/Rule/friendship.xml（失败串
// "[Error] FriendModule::LoadResource file failed!"）+ friendvalueevent.xml +
// enemyexpend.xml。Go 面经 IKernel::PreloadConfig 等价（ResourcePath 根）。
func (m *FriendModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: FriendModule.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/Rule/friendship.xml")
	if err != nil {
		return fmt.Errorf("fsgame: FriendModule::LoadResource file failed!: %w", err)
	}
	rule, err := parseFriendship(data)
	if err != nil {
		return err
	}
	m.Friendship = rule

	if data, err := readResource(k.ResourcePath, "ini/Rule/friendvalueevent.xml"); err == nil {
		if evs, err := parseFVEvents(data); err == nil {
			m.FVEvents = evs
		}
	}
	if data, err := readResource(k.ResourcePath, "ini/Rule/enemyexpend.xml"); err == nil {
		if ex, err := parseEnemyExpend(data); err == nil {
			m.EnemyExpend = ex
		}
	}
	k.Tracef("FriendModule: LoadResource friendship/friendvalueevent/enemyexpend")
	return nil
}

// onGetRoleInfo 官方 @0x863D80：跨服角色信息回执（blacklist_request_role_flag/
// Camp 查询 → 好友行在线/阵营更新）。
func (m *FriendModule) onGetRoleInfo(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("FriendModule: OnGetRoleInfo")
}

// onRecover 官方 @0x867400：对象级临界挂载（Level/Job/ArrestValue/CurArea）+
// 好友度事件表校准。
func (m *FriendModule) onRecover(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("FriendModule: OnRecover player=%v", self.ID())
}

// onStore 官方 @0x868110：Name/player_online 记账 + 存档。
func (m *FriendModule) onStore(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("FriendModule: OnStore player=%v", self.ID())
}

// onCustomClientMessage 官方 @0x86B650（命令 110）：好友操作多路（跳表子命令）。
// args：子命令, 目标名[, …]。
func (m *FriendModule) onCustomClientMessage(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	sub, target := args.Int(0), args.String(1)
	name, _ := o.Get("name").(string)
	switch sub {
	case 0: // OnCustomOpenFriend：打开好友界面（下发四表）
		k.Tracef("FriendModule: OpenFriend player=%v", self.ID())
	case 1: // OnCustomAddFriend
		m.AddFriend(k, self, name, target)
	case 2: // OnCustomDelFriend
		m.DelFriend(k, self, name, target)
	case 3: // OnCustomAddUnfamiliar
		m.addToList(k, self, "unfamiliar_rec", name, target, "sys_friend_0004", "sys_friend_0008")
	case 4: // OnCustomDelUnfamiliar
		m.delFromList(k, self, "unfamiliar_rec", name, target)
	case 5: // OnCustomAddBlackList
		m.AddBlackList(k, self, name, target)
	case 6: // OnCustomDelBlackList
		m.delFromList(k, self, "blacklist_rec", name, target)
	case 7: // OnCustomAddEnemy
		m.addToList(k, self, "enemy_rec", name, target, "sys_friend_0006", "sys_friend_0010")
	case 8: // OnCustomDelEnemy
		m.delFromList(k, self, "enemy_rec", name, target)
	case 9: // OnCustomSeeAbout：查看资料
		k.Tracef("FriendModule: SeeAbout %s→%s", name, target)
	case 11: // OnCustomGMAddFriend：GM 强制加好友（跳过校验）
		if tid := k.SeekRoleName(target); tid.Valid() || target != "" {
			m.addFriendRow(k, self, target, true)
		}
	case 12: // OnCustomRequstSeeAboutNeed
	default:
		k.Tracef("FriendModule: subcmd %d", sub)
	}
}

// AddFriend 官方 OnCustomAddFriend @0x8623E0：加好友闭环。
// 校验序（sys_friend_* 模板定案）：自加(0003)→已在好友(0007)→对方黑名单含我
// (0015)→对方不存在/离线(0002/0001)→好友满(0011)；成对入表后双方 sys_0024。
func (m *FriendModule) AddFriend(k *Kernel, self PersistID, name, target string) bool {
	if name == target {
		m.sys("sys_friend_0003")
		return false
	}
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	if rowByName(o.Record("friend_rec"), target) >= 0 {
		m.sys("sys_friend_0007")
		return false
	}
	tid := k.SeekRoleName(target)
	if !tid.Valid() {
		m.sys("sys_friend_0002")
		return false
	}
	if r := o.Record("friend_rec"); r != nil && r.Rows() >= r.Def().MaxRows {
		m.sys("sys_friend_0011")
		return false
	}
	// 对方黑名单含我：拒发（sys_friend_0018 语义）。
	if t := k.GetGameObj(tid); t != nil {
		if rowByName(t.Record("blacklist_rec"), name) >= 0 {
			m.sys("sys_friend_0018")
			return false
		}
	}
	m.addFriendRow(k, self, target, false)
	if t := k.GetGameObj(tid); t != nil {
		m.addFriendRow(k, tid, name, false)
	}
	m.sys("sys_friend_0024")
	return true
}

// addFriendRow friend_rec 追加一行（Name/Level/Job/Online/Scene/TeamSize/Value…）。
func (m *FriendModule) addFriendRow(k *Kernel, self PersistID, target string, gm bool) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	r := o.Record("friend_rec")
	if r == nil {
		return
	}
	row := make(VarList, r.Def().Cols)
	row[frName] = target
	row[frOnline] = int32(0)
	if tid := k.SeekRoleName(target); tid.Valid() {
		row[frOnline] = int32(1)
		if t := k.GetGameObj(tid); t != nil {
			row[frLevel] = t.GetInt("Level")
			row[frJob] = t.GetInt("Job")
		}
	}
	row[frValue] = int32(0)
	r.AppendRow(row)
	k.Tracef("FriendModule: friend + %s (gm=%v)", target, gm)
}

// DelFriend 官方 OnCustomDelFriend @0x862980：双向删除（0012/0013 模板）。
func (m *FriendModule) DelFriend(k *Kernel, self PersistID, name, target string) bool {
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	r := o.Record("friend_rec")
	i := rowByName(r, target)
	if i < 0 {
		return false
	}
	r.DeleteRow(i)
	m.sys("sys_friend_0012")
	if tid := k.SeekRoleName(target); tid.Valid() {
		if t := k.GetGameObj(tid); t != nil {
			if tr := t.Record("friend_rec"); tr != nil {
				if j := rowByName(tr, name); j >= 0 {
					tr.DeleteRow(j)
					// 对方侧通知 sys_friend_0013（由对方消息面下发）。
				}
			}
		}
	}
	return true
}

// AddBlackList 官方 OnCustomAddBlackList @0x863400：自加(0005)→已在(0009)→
// 满(0029)→敌对阵营(0040)；成功 0033（对方侧同步移出好友表）。
func (m *FriendModule) AddBlackList(k *Kernel, self PersistID, name, target string) bool {
	if name == target {
		m.sys("sys_friend_0005")
		return false
	}
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	if rowByName(o.Record("blacklist_rec"), target) >= 0 {
		m.sys("sys_friend_0009")
		return false
	}
	r := o.Record("blacklist_rec")
	if r == nil || r.Rows() >= r.Def().MaxRows {
		m.sys("sys_friend_0029")
		return false
	}
	row := make(VarList, r.Def().Cols)
	row[0] = target
	r.AppendRow(row)
	// 移出好友/陌生人表（拉黑即绝交）。
	for _, rn := range []string{"friend_rec", "unfamiliar_rec"} {
		if rr := o.Record(rn); rr != nil {
			if i := rowByName(rr, target); i >= 0 {
				rr.DeleteRow(i)
			}
		}
	}
	m.sys("sys_friend_0033")
	return true
}

// addToList 陌生人/仇人通用入表（selfMsg 自加校验 / dupMsg 重复校验模板号）。
func (m *FriendModule) addToList(k *Kernel, self PersistID, rec, name, target, selfMsg, dupMsg string) bool {
	if name == target {
		m.sys(selfMsg)
		return false
	}
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	r := o.Record(rec)
	if r == nil {
		return false
	}
	if rowByName(r, target) >= 0 {
		m.sys(dupMsg)
		return false
	}
	if r.Rows() >= r.Def().MaxRows {
		m.sys("sys_friend_0030")
		return false
	}
	row := make(VarList, r.Def().Cols)
	row[0] = target
	if tid := k.SeekRoleName(target); tid.Valid() {
		row[1] = k.GetGameObj(tid).GetInt("Level")
		row[2] = k.GetGameObj(tid).GetInt("Job")
		row[3] = int32(1)
	}
	r.AppendRow(row)
	if rec == "enemy_rec" {
		m.sys("sys_friend_0034")
	}
	return true
}

// delFromList 通用出表。
func (m *FriendModule) delFromList(k *Kernel, self PersistID, rec, name, target string) bool {
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	r := o.Record(rec)
	i := rowByName(r, target)
	if i < 0 {
		if rec == "enemy_rec" {
			m.sys("sys_friend_0028")
		}
		return false
	}
	r.DeleteRow(i)
	return true
}

// AddFriendValue 好友度增加（官方 nx_set_friend_value @0x85F140 Lua 面的 Go 等价；
// friendvalueevent_rec 行计数 × friendvalueevent.xml 规则门）。
// 行为：FVOnline/FVChat/FVTeamKill/FVTeamTask/FVTeamClone/FVPresent…。
func (m *FriendModule) AddFriendValue(k *Kernel, self PersistID, target, event string, delta int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	r := o.Record("friend_rec")
	i := rowByName(r, target)
	if i < 0 {
		return
	}
	row := r.Row(i)
	// 规则门：friendvalueevent.xml 的 IncreaseValue 覆盖缺省增量。
	if m.FVEvents != nil {
		if ev, ok := m.FVEvents[event]; ok {
			if delta <= 0 {
				delta = ev.IncreaseValue
			}
		}
	}
	cur := row.Int(frValue) + delta
	if m.Friendship != nil && cur > m.Friendship.MaxValue {
		cur = m.Friendship.MaxValue
	}
	row[frValue] = cur
	r.ModifyRow(i, row)
	m.valueBumps = append(m.valueBumps, fvBump{Target: target, Delta: delta, Event: event})
	k.Tracef("FriendModule: friend value %s +%d (%s) → %d", target, delta, event, cur)
}

// FriendshipLevel 好友度等级（friendship.xml FriendshipLevel 阈值序数）。
func (m *FriendModule) FriendshipLevel(value int32) int32 {
	if m.Friendship == nil {
		return 0
	}
	lv := int32(0)
	for _, th := range m.Friendship.FriendshipLevels {
		if value >= th {
			lv++
		}
	}
	return lv
}

// onCustomServerMessage 官方 @0x86B160（命令 400）：服务器侧消息
//（跨服好友请求回执面）。
func (m *FriendModule) onCustomServerMessage(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("FriendModule: OnCustomServerMessage")
}

// onUpdateFriendValue 官方 @0x86F490（命令 864）：好友度更新（定向）。
func (m *FriendModule) onUpdateFriendValue(k *Kernel, self, from PersistID, args VarList) {
	m.AddFriendValue(k, self, args.String(0), args.String(1), args.Int(2))
}

// onUpdateEachFriendValue 官方 @0x86F5D0（命令 868）：全表好友度更新
//（HB_CheckOnLine 的在线奖励入口）。
func (m *FriendModule) onUpdateEachFriendValue(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if r := o.Record("friend_rec"); r != nil {
		for i := 0; i < r.Rows(); i++ {
			name := r.Row(i).String(0)
			m.AddFriendValue(k, self, name, "FVOnline", 0)
		}
	}
}

// onCheckFriendValueUpdate 官方 @0x86F750（命令 869）：好友度事件表核对。
func (m *FriendModule) onCheckFriendValueUpdate(k *Kernel, self, from PersistID, args VarList) {}

// onCommandRefreshRecData 官方 @0x871570（命令 706）：四表重下发。
func (m *FriendModule) onCommandRefreshRecData(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("FriendModule: RefreshRecData player=%v", self.ID())
}

// onTransDecTicketRes 官方 @0x873730（命令 312）：通缉值消耗回执
//（enemyexpend EnemyFind 面）。
func (m *FriendModule) onTransDecTicketRes(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerBeKilled 官方 @0x8610C0（命令 616=0x268）：被杀 → 仇人表追加凶手
//（enemyexpend 消耗门）。
func (m *FriendModule) onPlayerBeKilled(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	killer := args.String(0)
	if killer == "" || rowByName(o.Record("friend_rec"), killer) >= 0 {
		return // 好友互杀不计仇
	}
	m.addToList(k, self, "enemy_rec", "", killer, "sys_friend_0006", "sys_friend_0010")
}

// onCommandBeKill 官方 @0x8614A0（AttackNpc 命令 616）。
func (m *FriendModule) onCommandBeKill(k *Kernel, self, from PersistID, args VarList) {}

// hbCheckPlayerFriend 官方 @0x866E00：好友表巡检（在线/等级/职业回填）。
func (m *FriendModule) hbCheckPlayerFriend(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	r := o.Record("friend_rec")
	if r == nil {
		return
	}
	for i := 0; i < r.Rows(); i++ {
		row := r.Row(i)
		name := row.String(0)
		if tid := k.SeekRoleName(name); tid.Valid() {
			t := k.GetGameObj(tid)
			row[frOnline] = int32(1)
			row[frLevel] = t.GetInt("Level")
			row[frJob] = t.GetInt("Job")
		} else {
			row[frOnline] = int32(0)
		}
		r.ModifyRow(i, row)
	}
}

// hbCheckPlayerBlackList 官方 @0x866E30：黑名单巡检（在线位回填）。
func (m *FriendModule) hbCheckPlayerBlackList(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if r := o.Record("blacklist_rec"); r != nil {
		for i := 0; i < r.Rows(); i++ {
			row := r.Row(i)
			if k.SeekRoleName(row.String(0)).Valid() {
				row[3] = int32(1)
			} else {
				row[3] = int32(0)
			}
			r.ModifyRow(i, row)
		}
	}
}

// hbCheckTeamID 官方 @0x8609B0：好友行“所在队伍人数”同步（team_rec 联动）。
func (m *FriendModule) hbCheckTeamID(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	r := o.Record("friend_rec")
	if r == nil {
		return
	}
	for i := 0; i < r.Rows(); i++ {
		row := r.Row(i)
		if tid := k.SeekRoleName(row.String(0)); tid.Valid() {
			if tm, ok := k.GetLogicModule("TeamModule").(*TeamModule); ok && tm != nil {
				if t := tm.TeamOf(k, tid); t != nil {
					row[frTeamSize] = int32(len(t.Members))
				}
			}
		}
		r.ModifyRow(i, row)
	}
}

// hbCheckOnLine 官方 @0x860C60：好友上/下线通知（0031/0032/0035/0036 模板）。
func (m *FriendModule) hbCheckOnLine(k *Kernel, self PersistID, beats int32) {}

// hbCheckExplore 官方 @0x860E30：探索状态巡检（BattleWarState/Pet/MasterID/
// player_die/BattleCarrier 排除面——非探索态不累计探索好友度）。
func (m *FriendModule) hbCheckExplore(k *Kernel, self PersistID, beats int32) {}

// C_OnBGLevelChange 官方 @0x8601B0：好友行等级回填（friend_rec[1]）。
func (m *FriendModule) onBGLevelChange(k *Kernel, self PersistID, prop string, old Var) {
	m.updateFriendCol(k, self, frLevel, func(o *GameObject) Var { return o.GetInt("Level") })
}

// C_OnBGJobChange 官方 @0x8603A0：好友行职业回填（friend_rec[2]）。
func (m *FriendModule) onBGJobChange(k *Kernel, self PersistID, prop string, old Var) {
	m.updateFriendCol(k, self, frJob, func(o *GameObject) Var { return o.GetInt("Job") })
}

// C_OnArrestValueChange 官方 @0x860590：好友行通缉值回填（friend_rec[10]）。
func (m *FriendModule) onArrestValueChange(k *Kernel, self PersistID, prop string, old Var) {
	m.updateFriendCol(k, self, frArrest, func(o *GameObject) Var { return o.GetInt("ArrestValue") })
}

// C_OnCurAreaChange 官方 @0x860780：好友行区域回填（friend_rec[16]）。
func (m *FriendModule) onCurAreaChange(k *Kernel, self PersistID, prop string, old Var) {
	m.updateFriendCol(k, self, frArea, func(o *GameObject) Var { return o.Get("CurArea") })
}

// updateFriendCol 本玩家属性变更 → 全部好友的好友表内本我行同步
//（官方 OnRecover 挂载的临界语义：A 的 Level 变化推送到 B 的 friend_rec[A]）。
func (m *FriendModule) updateFriendCol(k *Kernel, self PersistID, col int, val func(*GameObject) Var) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	name, _ := o.Get("name").(string)
	if name == "" {
		return
	}
	for _, pid := range k.PlayersSnapshot() {
		if pid == self {
			continue
		}
		if t := k.GetGameObj(pid); t != nil {
			if r := t.Record("friend_rec"); r != nil {
				if i := rowByName(r, name); i >= 0 {
					row := r.Row(i)
					row[col] = val(o)
					r.ModifyRow(i, row)
				}
			}
		}
	}
}

// OnTeamRecChange 官方 @0x8718D0（DeclareRecHook team_rec）：本玩家队伍行变更
// → 好友表 frTeamSize 列联动（HB_CheckTeamID 的即时面）。
func (m *FriendModule) onTeamRecChange(k *Kernel, self PersistID, record string, op, row int32) {
	m.hbCheckTeamID(k, self, 0)
}

// OnOriginRecChange 官方 @0x871A40（DeclareRecHook Origin_Record）：身份进度变更
// → 好友门派身份列（friend_rec[9]）联动。
func (m *FriendModule) onOriginRecChange(k *Kernel, self PersistID, record string, op, row int32) {}

func (m *FriendModule) sys(tpl string) { m.sysMsgs = append(m.sysMsgs, tpl) }

// SysMsgs 系统信息流水（测试面）。
func (m *FriendModule) SysMsgs() []string { return m.sysMsgs }

// ValueBumps 好友度变更流水（测试面）。
func (m *FriendModule) ValueBumps() []fvBump { return m.valueBumps }

// ---- 配置解析（ini/Rule/*.xml，Property 单行方言；复用 resource.go ParsePropTable）----

// ruleByID Property 行列表 → ID 索引表。
func ruleByID(rows []map[string]string) map[string]map[string]string {
	out := make(map[string]map[string]string, len(rows))
	for _, r := range rows {
		if id := r["ID"]; id != "" {
			out[id] = r
		}
	}
	return out
}

func atoi32List(s string) []int32 {
	var out []int32
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, atoi32(p))
		}
	}
	return out
}

// parseFriendship friendship.xml。
func parseFriendship(data []byte) (*FriendshipRule, error) {
	rows, err := ParsePropTable(data)
	if err != nil {
		return nil, fmt.Errorf("fsgame: friendship.xml: %w", err)
	}
	p, ok := ruleByID(rows)["FriendValue"]
	if !ok {
		return nil, fmt.Errorf("fsgame: friendship.xml: 无 FriendValue 行")
	}
	return &FriendshipRule{
		MaxValue:         atoi32(p["MaxValue"]),
		FVOlineMax:       atoi32(p["FVOlineMax"]),
		FVChatMax:        atoi32(p["FVChatMax"]),
		FVTeamMax:        atoi32(p["FVTeamMax"]),
		FVFunMax:         atoi32(p["FVFunMax"]),
		FVPresentMax:     atoi32(p["FVPresentMax"]),
		FVPVPMax:         atoi32(p["FVPVPMax"]),
		OnlineTime:       atoi32(p["OnlineTime"]),
		ExploreTime:      atoi32(p["ExploreTime"]),
		RidingTime:       atoi32(p["RidingTime"]),
		KillCount:        atoi32(p["KillCount"]),
		FriendshipLevels: atoi32List(p["FriendshipLevel"]),
	}, nil
}

// parseFVEvents friendvalueevent.xml。
func parseFVEvents(data []byte) (map[string]FVEventRule, error) {
	rows, err := ParsePropTable(data)
	if err != nil {
		return nil, err
	}
	out := make(map[string]FVEventRule, len(rows))
	for id, p := range ruleByID(rows) {
		out[id] = FVEventRule{
			IncreaseTime:  atoi32(p["IncreaseTime"]),
			IncreaseCount: atoi32(p["IncreaseCount"]),
			IncreaseValue: atoi32(p["IncreaseValue"]),
		}
	}
	return out, nil
}

// parseEnemyExpend enemyexpend.xml（ID → USE* 属性表原样保存）。
func parseEnemyExpend(data []byte) (map[string]string, error) {
	rows, err := ParsePropTable(data)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for id, p := range ruleByID(rows) {
		parts := make([]string, 0, len(p))
		for k, v := range p {
			if k != "ID" {
				parts = append(parts, k+"="+v)
			}
		}
		out[id] = strings.Join(parts, " ")
	}
	return out, nil
}
