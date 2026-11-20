package fsgame

// 十一期：玩家基础余量十四模块（docs/official/FsGame全解与Go复刻.md §8
// 玩家基础表余量）。官方源码树 playerbasemodule\ + commonmodule\ 等。
//
// 真档 schema（经 player.xml includes 链自动装载）：autorestore（HPUpSpeed/
// MPUpSpeed/Add 四列——Restore 六泵速度面）/ state（State 串）/ maxpara
//（EY/AV/RC/FS/DP 能量族 + Max* + 比例）/ safemode（SafeTime/CurSafeTime/
// SafeModePsw/IsHaveSafeModePsw/IsSafeModeUnLock/IsTempUnLock/LastOffLineTime）/
// doselect（LastObject）/ visequip（HideHat/HideMantle + Hair/Face/Eyes 外观族）/
// ModifyBase（PropModifyRec 等修正记录）；显式补档：relive（RelivePositon/
// ReliveTime/ReliveCdTime/ReliveNoNearPos/RelivePosSelectOpen + 首次三旗）/
// playerchange（MainRoleChange/bRoleChange/ChangeIndex/ChangeCostTime/
// ChangeLeftTime）/ offline_compensate（LogoutTime/OffLineExp）。

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// ---- RestoreModule（Init 序 137）----

// RestoreModule 复刻官方 RestoreModule（Init @0x19ECB0；符号 91，11 心跳）——
// 六恢复泵（HP/MP/EY/AV/RC/DP）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover"/"OnEntry"/"OnLeave"/"OnStore")
//	DeclareHeartBeat ×6（H_BeginRestore_HP/_MP/_EY/_AV/_RC/_DP）
//	DeclareCritical ×6（C_OnHPChanged/_MPChanged/_EYChanged/_AVChanged …——
//	    受击中断恢复、变化重启计时）
type RestoreModule struct {
	BaseModule

	// events 恢复流水（测试面："hp+30" 等）。
	events []string
}

// Name 模块名。
func (m *RestoreModule) Name() string { return "RestoreModule" }

// Init 注册期（对齐官方 0x19ECB0）。
func (m *RestoreModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnEvent("player", "OnLeave", m.onLeave)
	b.OnEvent("player", "OnStore", m.onStore)
	b.HeartBeat("RestoreModule::H_BeginRestore_HP", m.hbHP)
	b.HeartBeat("RestoreModule::H_BeginRestore_MP", m.hbMP)
	b.HeartBeat("RestoreModule::H_BeginRestore_EY", m.hbEnergy("EY", "MaxEY", "EYUpSpeed"))
	b.HeartBeat("RestoreModule::H_BeginRestore_AV", m.hbEnergy("AV", "MaxAV", "AVUpSpeed"))
	b.HeartBeat("RestoreModule::H_BeginRestore_RC", m.hbEnergy("RC", "MaxRC", "RCUpSpeed"))
	b.HeartBeat("RestoreModule::H_BeginRestore_DP", m.hbEnergy("DP", "MaxDP", "DPUpSpeed"))
	b.Critical("player", "HP", m.onHPChanged)
	b.Critical("player", "MP", m.onMPChanged)
	b.Critical("player", "EY", m.onEYChanged)
	b.Critical("player", "AV", m.onAVChanged)
	return nil
}

func (m *RestoreModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}
func (m *RestoreModule) onEntry(k *Kernel, self, from PersistID, args VarList)   {}
func (m *RestoreModule) onLeave(k *Kernel, self, from PersistID, args VarList)   {}
func (m *RestoreModule) onStore(k *Kernel, self, from PersistID, args VarList)   {}

// hbHP 生命泵：HPUpSpeed+HPUpSpeedAdd/拍（死亡面停；满值不越）。
func (m *RestoreModule) hbHP(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" || o.GetInt("Dead") != 0 {
		return
	}
	max := o.GetInt("MaxHP") + o.GetInt("MaxHPAdd")
	cur := o.GetInt("HP")
	if max <= 0 || cur <= 0 || cur >= max {
		return
	}
	up := o.GetInt("HPUpSpeed") + o.GetInt("HPUpSpeedAdd")
	if up <= 0 {
		return
	}
	if cur+up > max {
		up = max - cur
	}
	_ = k.SetInt(self, "HP", cur+up)
	m.events = append(m.events, fmt.Sprintf("hp+%d", up))
}

// hbMP 魔法泵。
func (m *RestoreModule) hbMP(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" || o.GetInt("Dead") != 0 {
		return
	}
	max := o.GetInt("MaxMP") + o.GetInt("MaxMPAdd")
	cur := o.GetInt("MP")
	if max <= 0 || cur < 0 || cur >= max {
		return
	}
	up := o.GetInt("MPUpSpeed") + o.GetInt("MPUpSpeedAdd")
	if up <= 0 {
		return
	}
	if cur+up > max {
		up = max - cur
	}
	_ = k.SetInt(self, "MP", cur+up)
	m.events = append(m.events, fmt.Sprintf("mp+%d", up))
}

// hbEnergy 能量泵工厂（EY/AV/RC/DP——速度属性缺省 0=不自然回复）。
func (m *RestoreModule) hbEnergy(cur, max, speed string) func(*Kernel, PersistID, int32) {
	return func(k *Kernel, self PersistID, beats int32) {
		o := k.GetGameObj(self)
		if o == nil || o.ClassName() != "player" || o.GetInt("Dead") != 0 {
			return
		}
		curV, maxV, up := o.GetInt(cur), o.GetInt(max), o.GetInt(speed)
		if maxV <= 0 || curV < 0 || curV >= maxV || up <= 0 {
			return
		}
		if curV+up > maxV {
			up = maxV - curV
		}
		_ = k.SetInt(self, cur, curV+up)
		m.events = append(m.events, fmt.Sprintf("%s+%d", strings.ToLower(cur), up))
	}
}

// 六临界官方面：受击/变化重启恢复计时（Go 面速度常量泵无需计时重启，
// 保留注册对齐 + trace）。
func (m *RestoreModule) onHPChanged(k *Kernel, self PersistID, prop string, old Var) {}
func (m *RestoreModule) onMPChanged(k *Kernel, self PersistID, prop string, old Var) {}
func (m *RestoreModule) onEYChanged(k *Kernel, self PersistID, prop string, old Var) {}
func (m *RestoreModule) onAVChanged(k *Kernel, self PersistID, prop string, old Var) {}

// Events 恢复流水（测试面）。
func (m *RestoreModule) Events() []string { return m.events }

// ---- DeadModule（Init 序 143）----

// DeadModule 复刻官方 DeadModule（Init @0x1727C0；符号 12——纯死亡结算
// 小模块）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCommandHook ("player",616=0x268,@0x172910 被杀结算)
type DeadModule struct {
	BaseModule

	// events 死亡流水（测试面："dead:<killer>"）。
	events []string
}

// Name 模块名。
func (m *DeadModule) Name() string { return "DeadModule" }

// Init 注册期（对齐官方 0x1727C0）。
func (m *DeadModule) Init(k *Kernel) error {
	k.Bus().OnIntCommand("player", 616, m.onBeKilled)
	return nil
}

// onBeKilled 官方 @0x172910：玩家被杀结算——死亡位/State 落定（HP≤0 面
// 由 FightModule 先置 Dead=1，此处结算击杀者档案 DeadAvenge）+ 复活域
// 联动（ReliveModule 倒计时启动）。
func (m *DeadModule) onBeKilled(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil || o.GetInt("HP") > 0 {
		return // 未达死亡门（616 多播族噪音面）
	}
	_ = k.SetInt(self, "Dead", 1)
	if _, ok := o.Class().Prop("State"); ok {
		_ = k.SetString(self, "State", "dead")
	}
	killer := from
	if len(args) > 0 {
		if v, ok := args[0].(PersistID); ok && v.Valid() {
			killer = v
		}
	}
	m.events = append(m.events, fmt.Sprintf("dead:%d", killer.ID()))
	// 击杀者档案（DeadAvenge 为属性面——KillerName；CurrentTime 是
	// 「复活后经过时间」毫秒，由复活面归零、此处不动）。
	if ko := k.GetGameObj(killer); ko != nil {
		if _, ok := o.Class().Prop("KillerName"); ok {
			_ = k.SetString(self, "KillerName", ko.GetString("name"))
		}
	}
	if rm := reliveModuleOf(k); rm != nil {
		rm.OnPlayerDead(k, self)
	}
	k.Tracef("DeadModule: player=%v dead by=%v", self.ID(), killer.ID())
}

// Events 死亡流水（测试面）。
func (m *DeadModule) Events() []string { return m.events }

// ---- ReliveModule（Init 序 141）----

// ReliveModule 复刻官方 ReliveModule（Init @0x173500；符号 1382，5 心跳）——
// 死亡→复活闭环。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnEntry"/"OnLeave"/"OnReady") /
//	    ("CommonNpc","OnSelect",@0x17B6A0 复活 NPC 对话面)
//	AddIntCommandHook ("player",22 菜单 / 315=0x13B 选择复活点) /
//	    ("AttackNpc"/"BattleCarrier"/"player",616)
//	AddIntCustomHook ("player",290=0x122 请求复活)
//	DeclareHeartBeat ×3（H_Relive/H_ReliveTimer/H_ReliveCDTimer）
type ReliveModule struct {
	BaseModule

	// reliveDelay 死亡倒计时秒（官方按表；Go 缺省 10，可注入）。
	ReliveDelay int32
	// ready 可复活时刻（玩家 → unix 秒；0=未死）。
	ready map[PersistID]int64
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *ReliveModule) Name() string { return "ReliveModule" }

// Init 注册期（对齐官方 0x173500）。
func (m *ReliveModule) Init(k *Kernel) error {
	m.ready = make(map[PersistID]int64)
	if m.ReliveDelay <= 0 {
		m.ReliveDelay = 10
	}
	b := k.Bus()
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnEvent("player", "OnLeave", m.onLeave)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	b.OnIntCommand("AttackNpc", 616, m.onNpcKilled)
	b.OnIntCommand("BattleCarrier", 616, m.onNpcKilled)
	b.OnIntCommand("player", 616, m.onNpcKilled)
	b.OnCommand("player", 290, m.onRequestRelive)
	b.OnIntCommand("player", 315, m.onSelectRelivePos)
	b.HeartBeat("ReliveModule::H_Relive", m.hbRelive)
	b.HeartBeat("ReliveModule::H_ReliveTimer", m.hbReliveTimer)
	b.HeartBeat("ReliveModule::H_ReliveCDTimer", m.hbReliveCDTimer)
	return nil
}

// OnPlayerDead 死亡联动（DeadModule 结算面）：倒计时登记 + ReliveTime 落档。
func (m *ReliveModule) OnPlayerDead(k *Kernel, player PersistID) {
	m.ready[player] = unixNow() + int64(m.ReliveDelay)
	if o := k.GetGameObj(player); o != nil {
		if _, ok := o.Class().Prop("ReliveTime"); ok {
			_ = o.Set("ReliveTime", int32(m.ReliveDelay))
		}
	}
	m.events = append(m.events, fmt.Sprintf("pending:%d", player.ID()))
}

// onNpcKilled 官方 616 面（NPC 被杀——复活 NPC 死亡则清理其对话上下文；
// 玩家面经 DeadModule 主链）。
func (m *ReliveModule) onNpcKilled(k *Kernel, self, from PersistID, args VarList) {}

// onRequestRelive 官方 @0x1750D0（cust 290=0x122）：请求复活
// （args: 模式 0 原地 / 1 回城）。门：倒计时未到拒；ReliveNoNearPos=1
// 禁原地（模式 0）。
func (m *ReliveModule) onRequestRelive(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil || o.GetInt("Dead") == 0 {
		return
	}
	mode := args.Int(0)
	if ready, ok := m.ready[self]; !ok || unixNow() < ready {
		m.events = append(m.events, fmt.Sprintf("reject-cd:%d", self.ID()))
		return
	}
	if mode == 0 && o.GetInt("ReliveNoNearPos") == 1 {
		mode = 1 // 禁最近点 → 回城
	}
	m.relive(k, self, mode)
}

// onMenuSelect 官方 @0x17B810（cmd 22 菜单）：复活 NPC 对话菜单接面
// （args: 动作 1=原地 / 2=回城——菜单上下文经 OnSelect 建立）。
func (m *ReliveModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil || o.GetInt("Dead") == 0 {
		return
	}
	switch args.Int(0) {
	case 1:
		m.relive(k, self, 0)
	case 2:
		m.relive(k, self, 1)
	}
}

// onSelectRelivePos 官方 @0x17C270（cmd 315=0x13B）：选择复活点
// （RelivePosSelectOpen 门）。
func (m *ReliveModule) onSelectRelivePos(k *Kernel, self, from PersistID, args VarList) {
	if o := k.GetGameObj(self); o != nil && o.GetInt("RelivePosSelectOpen") != 0 {
		m.events = append(m.events, fmt.Sprintf("pos-sel:%d", self.ID()))
	}
}

// onNpcSelect 官方 @0x17B6A0：复活 NPC 对话（可复活清单下发面）。
func (m *ReliveModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {
	if o := k.GetGameObj(from); o != nil && o.GetInt("Dead") != 0 {
		m.events = append(m.events, fmt.Sprintf("npc:%d", from.ID()))
	}
}

// relive 复活执行：模式 0 原地（HP/MP 半恢复——官方恢复比按表，Go 缺省
// 50%）/ 1 回城（RelivePositon 串 [name,x,y,z,o] 解析落位）→ Dead=0、
// State 复位、CD 记录、倒计时清。
func (m *ReliveModule) relive(k *Kernel, player PersistID, mode int32) {
	o := k.GetGameObj(player)
	if o == nil {
		return
	}
	if mode == 1 {
		if pos := o.GetString("RelivePositon"); pos != "" {
			// [name,x,y,z,o] 方言（relive.xml desc 定案）。
			inner := strings.Trim(pos, "[]")
			f := strings.Split(inner, ",")
			if len(f) >= 4 {
				o.SetPos(atof32(f[1]), atof32(f[2]), atof32(f[3]))
			}
		}
	}
	maxHP, maxMP := o.GetInt("MaxHP"), o.GetInt("MaxMP")
	_ = k.SetInt(player, "HP", maxHP/2)
	_ = k.SetInt(player, "MP", maxMP/2)
	_ = k.SetInt(player, "Dead", 0)
	if _, ok := o.Class().Prop("State"); ok {
		_ = k.SetString(player, "State", "")
	}
	if _, ok := o.Class().Prop("ReliveCdTime"); ok {
		_ = o.Set("ReliveCdTime", int32(30))
	}
	// 复仇计时面：CurrentTime（复活后经过毫秒）归零重计。
	if _, ok := o.Class().Prop("CurrentTime"); ok {
		_ = o.Set("CurrentTime", int32(0))
	}
	delete(m.ready, player)
	m.events = append(m.events, fmt.Sprintf("relive:%d:%d", player.ID(), mode))
	k.Tracef("ReliveModule: relive player=%v mode=%d", player.ID(), mode)
}

// hbRelive / hbReliveTimer / hbReliveCDTimer 官方三心跳：倒计时到位
// （可复活通知）+ CD 递减。Go 面：到位通知一次（ready→notified 簿记）。
func (m *ReliveModule) hbRelive(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" {
		return
	}
	ready, ok := m.ready[self]
	if !ok || o.GetInt("Dead") == 0 {
		return
	}
	if unixNow() >= ready {
		delete(m.ready, self)
		m.events = append(m.events, fmt.Sprintf("ready:%d", self.ID()))
		if _, ok := o.Class().Prop("ReliveTime"); ok {
			_ = o.Set("ReliveTime", int32(0))
		}
	}
}

func (m *ReliveModule) hbReliveTimer(k *Kernel, self PersistID, beats int32) {}

func (m *ReliveModule) hbReliveCDTimer(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" {
		return
	}
	if cd := o.GetInt("ReliveCdTime"); cd > 0 {
		_ = o.Set("ReliveCdTime", cd-1)
	}
}

func (m *ReliveModule) onEntry(k *Kernel, self, from PersistID, args VarList) {}
func (m *ReliveModule) onLeave(k *Kernel, self, from PersistID, args VarList) {}
func (m *ReliveModule) onReady(k *Kernel, self, from PersistID, args VarList) {}

// Events 复活流水（测试面）。
func (m *ReliveModule) Events() []string { return m.events }

// ---- SafeModeModule（Init 序 129）----

// SafeModeModule 复刻官方 SafeModeModule（Init @0x279140；符号 110）——
// 安全模式（防盗号锁）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnReady"/"OnStore")
//	DeclareHeartBeat ("SafeModeModule::HB_CheckSafeTime")
//	AddIntCustomHook ("player",261=0x105 密码校验)
type SafeModeModule struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *SafeModeModule) Name() string { return "SafeModeModule" }

// Init 注册期（对齐官方 0x279140）。
func (m *SafeModeModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnStore", m.onStore)
	b.HeartBeat("SafeModeModule::HB_CheckSafeTime", m.hbCheckSafeTime)
	b.OnCommand("player", 261, m.onVerifyPsw)
	return nil
}

// onReady 官方 @0x279900：上线——CurSafeTime 复位 SafeTime（保护倒计时
// 重启）+ IsSafeModeUnLock 复位。
func (m *SafeModeModule) onReady(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if _, ok := o.Class().Prop("CurSafeTime"); ok {
		_ = o.Set("CurSafeTime", o.GetInt("SafeTime"))
	}
	if _, ok := o.Class().Prop("IsSafeModeUnLock"); ok {
		_ = o.Set("IsSafeModeUnLock", int32(0))
	}
}

func (m *SafeModeModule) onStore(k *Kernel, self, from PersistID, args VarList) {}

// hbCheckSafeTime 官方 @0x279E40：保护倒计时——CurSafeTime>0 每拍递减，
// 归零即自动解锁（无密码面）/ 维持锁定（有密码面待校验）。
func (m *SafeModeModule) hbCheckSafeTime(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" {
		return
	}
	cur := o.GetInt("CurSafeTime")
	if cur <= 0 {
		return
	}
	_ = o.Set("CurSafeTime", cur-1)
	if cur == 1 && o.GetInt("IsHaveSafeModePsw") == 0 {
		m.events = append(m.events, fmt.Sprintf("auto-unlock:%d", self.ID()))
	}
}

// onVerifyPsw 官方 @0x27A170（cust 261=0x105）：密码校验（args: 密码串）——
// 命中 SafeModePsw → IsSafeModeUnLock=1；未设密码 → 直接过。
func (m *SafeModeModule) onVerifyPsw(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if o.GetInt("IsHaveSafeModePsw") == 0 || args.String(0) == o.GetString("SafeModePsw") {
		_ = o.Set("IsSafeModeUnLock", int32(1))
		m.events = append(m.events, fmt.Sprintf("unlock:%d", self.ID()))
	}
}

// SafeModeActive 安全模式生效门（交易/邮寄/丢弃等敏感操作面调用）：
// 有密码且本次登录未解锁。
func (m *SafeModeModule) SafeModeActive(k *Kernel, player PersistID) bool {
	o := k.GetGameObj(player)
	return o != nil && o.GetInt("IsHaveSafeModePsw") == 1 && o.GetInt("IsSafeModeUnLock") == 0
}

// Events 流水（测试面）。
func (m *SafeModeModule) Events() []string { return m.events }

// ---- PlayerChangeModule（Init 序 135）----

// PlayerChangeModule 复刻官方 PlayerChangeModule（Init @0x238660；符号 137）——
// 变身（模型替换 + 时限消耗）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover"/"OnStore")
//	DeclareLuaExt nx_changemodel_role
//	DeclareCritical (RoleType→C_OnRoleTypeChanged / BRole→C_OnBRoleChanged /
//	    ChangedIndex→C_OnChangedIndex)
//	DeclareHeartBeat ("PlayerChangeModule::HB_ChangeCost")
type PlayerChangeModule struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *PlayerChangeModule) Name() string { return "PlayerChangeModule" }

// Init 注册期（对齐官方 0x238660）。
func (m *PlayerChangeModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnStore", m.onStore)
	b.Critical("player", "RoleChangeType", m.onRoleTypeChanged)
	b.Critical("player", "bRoleChange", m.onBRoleChanged)
	b.Critical("player", "ChangeIndex", m.onChangedIndex)
	b.HeartBeat("PlayerChangeModule::HB_ChangeCost", m.hbChangeCost)
	return nil
}

// ChangeRole 变身（nx_changemodel_role 的 Go 等价）：模型串 + 时限秒
// （ChangeLeftTime 上限 1h——schema desc 定案）→ bRoleChange=1。
func (m *PlayerChangeModule) ChangeRole(k *Kernel, player PersistID, model string, seconds int32) bool {
	o := k.GetGameObj(player)
	if o == nil {
		return false
	}
	if seconds <= 0 || seconds > 3600 {
		seconds = 3600
	}
	_ = o.Set("MainRoleChange", model)
	_ = o.Set("ChangeLeftTime", seconds)
	_ = o.Set("ChangeCostTime", int32(0))
	_ = k.SetInt(player, "bRoleChange", 1)
	m.events = append(m.events, fmt.Sprintf("change:%d:%s", player.ID(), model))
	return true
}

// hbChangeCost 官方 @0x239420：变身时限泵——ChangeCostTime++/ChangeLeftTime--，
// 归零解除变身（bRoleChange=0 + 模型复原空）。
func (m *PlayerChangeModule) hbChangeCost(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" || o.GetInt("bRoleChange") == 0 {
		return
	}
	left := o.GetInt("ChangeLeftTime")
	if left > 0 {
		_ = o.Set("ChangeLeftTime", left-1)
		_ = o.Set("ChangeCostTime", o.GetInt("ChangeCostTime")+1)
		return
	}
	_ = k.SetInt(self, "bRoleChange", 0)
	_ = o.Set("MainRoleChange", "")
	m.events = append(m.events, fmt.Sprintf("revert:%d", self.ID()))
}

func (m *PlayerChangeModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}
func (m *PlayerChangeModule) onStore(k *Kernel, self, from PersistID, args VarList)   {}

// 三临界官方面（变身态联动 trace）。
func (m *PlayerChangeModule) onRoleTypeChanged(k *Kernel, self PersistID, prop string, old Var) {}
func (m *PlayerChangeModule) onBRoleChanged(k *Kernel, self PersistID, prop string, old Var) {
	if o := k.GetGameObj(self); o != nil && o.GetInt("bRoleChange") == 0 {
		m.events = append(m.events, fmt.Sprintf("manual-revert:%d", self.ID()))
	}
}

func (m *PlayerChangeModule) onChangedIndex(k *Kernel, self PersistID, prop string, old Var) {}

// Events 流水（测试面）。
func (m *PlayerChangeModule) Events() []string { return m.events }

// ---- RoleExteriorModule（Init 序 132）----

// RoleExteriorModule 复刻官方 RoleExteriorModule（Init @0x1D4B60；符号 790）——
// 角色外观（visequip 族：HideHat/HideMantle + Hair/Face/Eyes）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover")
//	DeclareCritical (HideHat→C_OnHideHatChanged / HideMantle→C_OnHideMantleChanged)
type RoleExteriorModule struct {
	BaseModule

	// changes 外观变更流水（测试面："hidehat:1"）。
	changes []string
}

// Name 模块名。
func (m *RoleExteriorModule) Name() string { return "RoleExteriorModule" }

// Init 注册期（对齐官方 0x1D4B60）。
func (m *RoleExteriorModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.Critical("player", "HideHat", m.onHideHatChanged)
	b.Critical("player", "HideMantle", m.onHideMantleChanged)
	return nil
}

// ExteriorSnapshot 外观快照（下发面：模型/武器持握/外观串族）。
func (m *RoleExteriorModule) ExteriorSnapshot(k *Kernel, player PersistID) map[string]string {
	o := k.GetGameObj(player)
	if o == nil {
		return nil
	}
	out := map[string]string{}
	for _, p := range []string{"Hair", "Face", "Eyes", "Beard", "Ears", "ArtKey"} {
		if _, ok := o.Class().Prop(p); ok {
			out[p] = o.GetString(p)
		}
	}
	return out
}

func (m *RoleExteriorModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}

func (m *RoleExteriorModule) onHideHatChanged(k *Kernel, self PersistID, prop string, old Var) {
	if o := k.GetGameObj(self); o != nil {
		m.changes = append(m.changes, fmt.Sprintf("hidehat:%d", o.GetInt("HideHat")))
	}
}

func (m *RoleExteriorModule) onHideMantleChanged(k *Kernel, self PersistID, prop string, old Var) {
	if o := k.GetGameObj(self); o != nil {
		m.changes = append(m.changes, fmt.Sprintf("hidemantle:%d", o.GetInt("HideMantle")))
	}
}

// Changes 外观流水（测试面）。
func (m *RoleExteriorModule) Changes() []string { return m.changes }

// ---- EnergyModule（Init 序 52）----

// EnergyModule 复刻官方 EnergyModule（Init @0x40CD10；符号 579；零内核
// 注册——黑金能量（变身货币，item/BlackGold.xml Energy 列）运营面）。
// Go 收编：API 泵（能量查改 + 黑金折算），属性面按类定义守卫。
type EnergyModule struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *EnergyModule) Name() string { return "EnergyModule" }

// Init 注册期（官方零注册——纯 API 模块）。
func (m *EnergyModule) Init(k *Kernel) error { return nil }

// QueryEnergy 能量查询（对象面有 Energy 属性则读，缺省 0）。
func (m *EnergyModule) QueryEnergy(k *Kernel, player PersistID) int32 {
	if o := k.GetGameObj(player); o != nil {
		return o.GetInt("Energy")
	}
	return 0
}

// ConsumeEnergy 能量消耗（不足拒；黑金折算 1:1——item/BlackGold 定档面）。
func (m *EnergyModule) ConsumeEnergy(k *Kernel, player PersistID, amount int32) bool {
	o := k.GetGameObj(player)
	if o == nil || amount <= 0 {
		return false
	}
	cur := o.GetInt("Energy")
	if cur < amount {
		return false
	}
	_ = k.SetInt(player, "Energy", cur-amount)
	m.ops = append(m.ops, fmt.Sprintf("consume:%d:%d", player.ID(), amount))
	return true
}

// Ops 流水（测试面）。
func (m *EnergyModule) Ops() []string { return m.ops }

// ---- RevertPlayerModule（Init 序 122）----

// RevertPlayerModule 复刻官方 RevertPlayerModule（Init @0x2439A0；符号 562）——
// 玩家数据还原（回滚到最近存档态；GM/运维面）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnReady")
//	AddIntCustomHook ("player",712=0x2C8 / 713=0x2C9)
//	DeclareLuaExt nx_start_revert
type RevertPlayerModule struct {
	BaseModule

	// pending 还原登记（玩家 → 存档快照）。
	pending map[PersistID]map[string]Var
	// ops 流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *RevertPlayerModule) Name() string { return "RevertPlayerModule" }

// Init 注册期（对齐官方 0x2439A0）。
func (m *RevertPlayerModule) Init(k *Kernel) error {
	m.pending = make(map[PersistID]map[string]Var)
	b := k.Bus()
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnCommand("player", 712, m.onStartRevert)
	b.OnCommand("player", 713, m.onCancelRevert)
	return nil
}

// NxStartRevert nx_start_revert 的 Go 等价：登记还原点（当前 save=true
// 属性快照）。
func (m *RevertPlayerModule) NxStartRevert(k *Kernel, player PersistID) bool {
	o := k.GetGameObj(player)
	if o == nil {
		return false
	}
	props, _ := o.Archive()
	m.pending[player] = props
	m.ops = append(m.ops, fmt.Sprintf("snapshot:%d", player.ID()))
	return true
}

// onStartRevert 官方 @0x244270（cust 712）：执行还原——快照属性回写。
func (m *RevertPlayerModule) onStartRevert(k *Kernel, self, from PersistID, args VarList) {
	snap, ok := m.pending[self]
	if !ok {
		return
	}
	o := k.GetGameObj(self)
	for name, v := range snap {
		if _, ok := o.Class().Prop(name); ok {
			_ = o.Set(name, v)
		}
	}
	delete(m.pending, self)
	m.ops = append(m.ops, fmt.Sprintf("revert:%d", self.ID()))
}

// onCancelRevert 官方 @0x244150（cust 713）：取消还原。
func (m *RevertPlayerModule) onCancelRevert(k *Kernel, self, from PersistID, args VarList) {
	if _, ok := m.pending[self]; ok {
		delete(m.pending, self)
		m.ops = append(m.ops, fmt.Sprintf("cancel:%d", self.ID()))
	}
}

func (m *RevertPlayerModule) onReady(k *Kernel, self, from PersistID, args VarList) {}

// Ops 流水（测试面）。
func (m *RevertPlayerModule) Ops() []string { return m.ops }

// ---- FreshmanModule（Init 序 144）----

// FreshmanModule 复刻官方 FreshmanModule（Init @0x1F4070；符号 284）——
// 新手引导（NewUserLead 面：IsInNewUserLead/IsGetAward）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddClassCallback ("CopyToolBox","OnCreateClass")
//	AddEventCallback ("player","OnStore"/"OnEntry"/"OnReady")
//	AddIntCustomHook ("player",145=0x91)
//	AddIntCommandHook ("VisibleNpc",616)
//	DeclareCritical (LogicState→C_OnLogicStateChanged)
//	DeclareHeartBeat ×2（HB_DelayClone/HB_ExitClone）
type FreshmanModule struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *FreshmanModule) Name() string { return "FreshmanModule" }

// Init 注册期（对齐官方 0x1F4070）。
func (m *FreshmanModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnClassEvent("CopyToolBox", "OnCreateClass", func(k *Kernel, class string) {
		m.ops = append(m.ops, "copybox:create")
	})
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnCommand("player", 145, m.onFreshmanMsg)
	b.OnIntCommand("VisibleNpc", 616, m.onNpcKilled)
	b.Critical("VisibleNpc", "LogicState", m.onLogicStateChanged)
	b.HeartBeat("FreshmanModule::HB_DelayClone", m.hbDelayClone)
	b.HeartBeat("FreshmanModule::HB_ExitClone", m.hbExitClone)
	return nil
}

// InLead 新手引导期门（IsInNewUserLead 面）。
func (m *FreshmanModule) InLead(k *Kernel, player PersistID) bool {
	if o := k.GetGameObj(player); o != nil {
		return o.GetInt("IsInNewUserLead") != 0
	}
	return false
}

func (m *FreshmanModule) onFreshmanMsg(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, fmt.Sprintf("msg:%d:%d", self.ID(), args.Int(0)))
}

func (m *FreshmanModule) onNpcKilled(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "npc-killed")
}

func (m *FreshmanModule) onLogicStateChanged(k *Kernel, self PersistID, prop string, old Var) {
	m.ops = append(m.ops, "logic-state")
}

func (m *FreshmanModule) hbDelayClone(k *Kernel, self PersistID, beats int32)   {}
func (m *FreshmanModule) hbExitClone(k *Kernel, self PersistID, beats int32)    {}
func (m *FreshmanModule) onStore(k *Kernel, self, from PersistID, args VarList) {}
func (m *FreshmanModule) onEntry(k *Kernel, self, from PersistID, args VarList) {
	if m.InLead(k, self) {
		m.ops = append(m.ops, fmt.Sprintf("lead-entry:%d", self.ID()))
	}
}

func (m *FreshmanModule) onReady(k *Kernel, self, from PersistID, args VarList) {}

// Ops 流水（测试面）。
func (m *FreshmanModule) Ops() []string { return m.ops }

// ---- ObjPropRefreshModule（Init 序 192）----

// ObjPropRefreshModule 复刻官方 ObjPropRefreshModule（Init @0x7D8330；
// 符号 724）——属性修正聚合刷新（ModifyBase 记录族 → 属性面重算）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddClassCallback ("player","OnCreateClass")
//	AddEventCallback ("player","OnRecover"/"OnReady"/"OnStore") +
//	    ("AttackNpc"/"BattleCarrier"/"Pet","OnLoad")
//	DeclareCritical ×3（BindSaveProp/BindProp/BindPropAdd）
//	DeclareRecHook ×2（ModifyPackRec/PropComputeRec）
type ObjPropRefreshModule struct {
	BaseModule

	// PropPacks 修正包真档（share/modifypack/proppack.ini——十五期接入：
	// 节=包号，r=「属性名,值,方式」多行方言，如 [2] r=,StrAdd,2,0）。
	PropPacks map[string][]PropPackEntry
	// refreshes 刷新计数（测试面）。
	refreshes int
}

// PropPackEntry 修正包条目（属性名, 值, 计算方式）。
type PropPackEntry struct {
	Prop  string
	Value int32
	Mode  int32
}

// Name 模块名。
func (m *ObjPropRefreshModule) Name() string { return "ObjPropRefreshModule" }

// Init 注册期（对齐官方 0x7D8330）。
func (m *ObjPropRefreshModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnClassEvent("player", "OnCreateClass", func(k *Kernel, class string) {})
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnEvent("AttackNpc", "OnLoad", m.onLoad)
	b.OnEvent("BattleCarrier", "OnLoad", m.onLoad)
	b.OnEvent("Pet", "OnLoad", m.onLoad)
	b.Critical("player", "BindSaveProp", m.onBindSavePropChanged)
	b.Critical("player", "BindProp", m.onBindPropChanged)
	b.Critical("player", "BindPropAdd", m.onBindPropAddChanged)
	b.RecHook("ModifyPackRec", m.onModifyPackRecChanged)
	b.RecHook("PropComputeRec", m.onPropComputeRecChanged)
	return nil
}

// LoadResource 修正包真档装载（share/modifypack/proppack.ini——fskit
// ParseItems 官方方言；重复 r= 键按序成条目）。
func (m *ObjPropRefreshModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(k.ResourcePath, "share", "modifypack", "proppack.ini"))
	if err != nil {
		return nil // 缺档静默（官方 loadres 非致命面）
	}
	packs := make(map[string][]PropPackEntry)
	err = fskit.ParseItems(data, func(sec, key, value string) error {
		if sec == "" || key != "r" {
			return nil
		}
		f := strings.Split(strings.TrimPrefix(value, ","), ",")
		if len(f) < 2 {
			return nil
		}
		e := PropPackEntry{Prop: strings.TrimSpace(f[0]), Value: atoi32(f[1])}
		if len(f) > 2 {
			e.Mode = atoi32(f[2])
		}
		if e.Prop != "" {
			packs[sec] = append(packs[sec], e)
		}
		return nil
	})
	if err != nil {
		return err
	}
	m.PropPacks = packs
	return nil
}

// PropPack 包查询（缺省 nil）。
func (m *ObjPropRefreshModule) PropPack(id string) []PropPackEntry { return m.PropPacks[id] }

// Recalc 属性修正聚合重算：PropModifyRec 行（属性名, 加成…——ModifyBase
// 方言 col0=属性名）累加 → 目标属性 += 聚合值（重算面：先减旧聚合再加新）。
func (m *ObjPropRefreshModule) Recalc(k *Kernel, id PersistID) {
	o := k.GetGameObj(id)
	if o == nil {
		return
	}
	r := o.Record("PropModifyRec")
	if r == nil {
		return
	}
	for i := 0; i < r.Rows(); i++ {
		row := r.Row(i)
		if row.Len() < 2 {
			continue
		}
		name := row.String(0)
		if name == "" {
			continue
		}
		if _, defined := o.Class().Prop(name); !defined {
			continue
		}
		// col1=修正值（double——ModifyBase 方言）。
		_ = k.SetInt(id, name, o.GetInt(name)+int32(row.Double(1)))
	}
	m.refreshes++
}

func (m *ObjPropRefreshModule) onRecover(k *Kernel, self, from PersistID, args VarList) {
	m.Recalc(k, self)
}
func (m *ObjPropRefreshModule) onReady(k *Kernel, self, from PersistID, args VarList) {}
func (m *ObjPropRefreshModule) onStore(k *Kernel, self, from PersistID, args VarList) {}
func (m *ObjPropRefreshModule) onLoad(k *Kernel, self, from PersistID, args VarList) {
	m.Recalc(k, self)
}

// 三临界官方面（绑定属性变更 → 重算）。
func (m *ObjPropRefreshModule) onBindSavePropChanged(k *Kernel, self PersistID, prop string, old Var) {
	m.Recalc(k, self)
}
func (m *ObjPropRefreshModule) onBindPropChanged(k *Kernel, self PersistID, prop string, old Var) {
	m.Recalc(k, self)
}
func (m *ObjPropRefreshModule) onBindPropAddChanged(k *Kernel, self PersistID, prop string, old Var) {
	m.Recalc(k, self)
}

// 双 RecHook 官方面（修正记录变更 → 重算）。
func (m *ObjPropRefreshModule) onModifyPackRecChanged(k *Kernel, self PersistID, record string, op, row int32) {
	m.Recalc(k, self)
}
func (m *ObjPropRefreshModule) onPropComputeRecChanged(k *Kernel, self PersistID, record string, op, row int32) {
	m.Recalc(k, self)
}

// Refreshes 刷新计数（测试面）。
func (m *ObjPropRefreshModule) Refreshes() int { return m.refreshes }

// ---- OffLineCompensateModule（Init 序 195）----

// OffLineCompensateModule 复刻官方 OffLineCompensateModule（Init @0x7C6D10；
// 符号 184）——离线补偿（offline_compensate.xml：LogoutTime/OffLineExp）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover"/"OnStore")
//	AddIntCommandHook ("player",884=0x374 领取)
type OffLineCompensateModule struct {
	BaseModule

	// ExpPerHour 离线经验速率（官方按表；Go 缺省 1000/时，可注入）。
	ExpPerHour int32
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *OffLineCompensateModule) Name() string { return "OffLineCompensateModule" }

// Init 注册期（对齐官方 0x7C6D10）。
func (m *OffLineCompensateModule) Init(k *Kernel) error {
	if m.ExpPerHour <= 0 {
		m.ExpPerHour = 1000
	}
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnIntCommand("player", 884, m.onClaim)
	return nil
}

// onStore 官方 @0x7C7CC0：下线——LogoutTime 落档。
func (m *OffLineCompensateModule) onStore(k *Kernel, self, from PersistID, args VarList) {
	if o := k.GetGameObj(self); o != nil {
		if _, ok := o.Class().Prop("LogoutTime"); ok {
			_ = o.Set("LogoutTime", unixNow())
		}
	}
}

// onRecover 官方 @0x7C7A90：回载——按离线时长结算 OffLineExp
// （速率 ExpPerHour/时；上限 24h——官方按表，Go 收编缺省）。
func (m *OffLineCompensateModule) onRecover(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	logout := o.GetInt64("LogoutTime")
	if logout <= 0 {
		return
	}
	secs := unixNow() - logout
	if secs > 24*3600 {
		secs = 24 * 3600
	}
	if secs < 60 {
		return
	}
	exp := int32(int64(m.ExpPerHour) * secs / 3600)
	if _, ok := o.Class().Prop("OffLineExp"); ok {
		_ = o.Set("OffLineExp", o.GetInt("OffLineExp")+exp)
	}
	m.events = append(m.events, fmt.Sprintf("accrue:%d:%d", self.ID(), exp))
}

// onClaim 官方 @0x7C8070（cmd 884=0x374）：领取——OffLineExp 清零 + 经验
// 入账（PlayerExp 临界 → LevelModule 升级链）。
func (m *OffLineCompensateModule) onClaim(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil || o.GetInt("OffLineExp") <= 0 {
		return
	}
	exp := o.GetInt("OffLineExp")
	_ = o.Set("OffLineExp", int32(0))
	if _, ok := o.Class().Prop("PlayerExp"); ok {
		_ = k.SetInt(self, "PlayerExp", o.GetInt("PlayerExp")+exp)
	}
	m.events = append(m.events, fmt.Sprintf("claim:%d:%d", self.ID(), exp))
}

// Events 流水（测试面）。
func (m *OffLineCompensateModule) Events() []string { return m.events }

// ---- SetPathModule（Init 序 206）----

// SetPathModule 复刻官方 SetPathModule（Init @0x6CAC00；符号 280；注册面
// 为 StageModule 噪声行——路径设置 API 模块）：NPC 巡逻路径装载
// （PathID → AIFindPathRec 路径点表）。
type SetPathModule struct {
	BaseModule

	// ops 流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *SetPathModule) Name() string { return "SetPathModule" }

// Init 注册期（官方注册行为扫描噪声——API 模块）。
func (m *SetPathModule) Init(k *Kernel) error { return nil }

// PathPoint 巡逻路径点。
type PathPoint struct {
	X, Z float32
	Stop int32 // 停留毫秒（PatrolPointRec Stoptime 方言）
}

// SetPath 路径装载：PathID 属性写 + AIFindPathRec 逐点入表（x/z 两列——
// ai_motion.xml 定案）。
func (m *SetPathModule) SetPath(k *Kernel, npc PersistID, pathID string, points []PathPoint) bool {
	o := k.GetGameObj(npc)
	if o == nil {
		return false
	}
	if _, ok := o.Class().Prop("PathID"); ok {
		_ = o.Set("PathID", pathID)
	}
	r := o.Record("AIFindPathRec")
	if r == nil {
		return true
	}
	r.Clear()
	for _, p := range points {
		if r.Rows() >= r.Def().MaxRows {
			break
		}
		r.AppendRow(VarList{p.X, p.Z})
	}
	m.ops = append(m.ops, fmt.Sprintf("path:%d:%s:%d", npc.ID(), pathID, len(points)))
	return true
}

// Ops 流水（测试面）。
func (m *SetPathModule) Ops() []string { return m.ops }

// ---- PathFindingModule（Init 序 212）----

// PathFindingModule 复刻官方 PathFindingModule（Init @0x66CE40；符号 444）——
// 寻路面（SceneMapProcess 场景地图处理）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("scene","OnCreate")——SceneMapProcess 创建
//	    （失败留痕 "SceneMapProcess create failed: %s ..."）
//	DeclareHeartBeat ×2
type PathFindingModule struct {
	BaseModule

	// maps 场景地图登记（scene 对象 → true）。
	maps map[PersistID]bool
	// graphs 导航图真档（map/path/*.path 文件名 → 节点图——2026-11-20
	// 十五期逆向定案：u32 节点数 + {f32 xyz + 边×{u32 邻居,f32 代价}}，
	// 代价=欧氏距离全中/无向回边全对称）。
	graphs map[string][]PathNode
	// ops 流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *PathFindingModule) Name() string { return "PathFindingModule" }

// Init 注册期（对齐官方 0x66CE40）。
func (m *PathFindingModule) Init(k *Kernel) error {
	m.maps = make(map[PersistID]bool)
	b := k.Bus()
	b.OnEvent("scene", "OnCreate", m.onSceneCreate)
	b.HeartBeat("PathFindingModule::HB_Process", m.hbProcess)
	b.HeartBeat("PathFindingModule::HB_Relax", m.hbRelax)
	return nil
}

// onSceneCreate 官方：场景物化 → SceneMapProcess 登记（地图数据面）。
func (m *PathFindingModule) onSceneCreate(k *Kernel, self, from PersistID, args VarList) {
	m.maps[self] = true
	m.ops = append(m.ops, fmt.Sprintf("map:%d", self.ID()))
}

// LoadResource 导航图真档装载（map/path/*.path；缺目录静默）。
func (m *PathFindingModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return nil
	}
	graphs, err := LoadPathDir(filepath.Join(k.ResourcePath, "map", "path"))
	if err != nil {
		return fmt.Errorf("fsgame: PathFindingModule::LoadResource: %w", err)
	}
	m.graphs = graphs
	return nil
}

// GraphNames 已装载导航图名（测试面）。
func (m *PathFindingModule) GraphNames() []string {
	out := make([]string, 0, len(m.graphs))
	for name := range m.graphs {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// FindPathOnGraph 图上寻路：起点就近锚定其巡逻段（连通分量——.path 文件
// =多段巡逻路径集合，段内 Dijkstra；跨段返 nil）。
func (m *PathFindingModule) FindPathOnGraph(name string, x1, z1, x2, z2 float32) []PathPoint {
	nodes := m.graphs[name]
	if len(nodes) == 0 {
		return nil
	}
	from := int32(NearestNode(nodes, x1, z1))
	comp := ComponentOf(nodes, from)
	to, bestD := int32(-1), float32(math.MaxFloat32)
	for id := range comp {
		if d := dist2D(nodes[id].X, nodes[id].Z, x2, z2); d < bestD {
			to, bestD = id, d
		}
	}
	idx := ShortestPath(nodes, from, to)
	if idx == nil {
		return nil
	}
	out := make([]PathPoint, len(idx))
	for i, v := range idx {
		out[i] = PathPoint{X: nodes[v].X, Z: nodes[v].Z}
	}
	m.ops = append(m.ops, fmt.Sprintf("findg:%s:%d", name, len(out)))
	return out
}

// FindPath 寻路（Go 收编：直线分段简化——A* 网格寻路随碰撞服里程碑后置，
// 官方 SceneMapProcess 网格面）：返回 waypoint 列表（步长 step 分段）。
func (m *PathFindingModule) FindPath(k *Kernel, scene PersistID, x1, z1, x2, z2 float32, step float32) []PathPoint {
	if !m.maps[scene] {
		return nil
	}
	if step <= 0 {
		step = 5
	}
	d := dist2D(x1, z1, x2, z2)
	if d == 0 {
		return []PathPoint{{X: x2, Z: z2}}
	}
	n := int(d/step) + 1
	out := make([]PathPoint, 0, n)
	for i := 1; i <= n; i++ {
		f := float32(i) / float32(n)
		if f > 1 {
			f = 1
		}
		out = append(out, PathPoint{X: x1 + (x2-x1)*f, Z: z1 + (z2-z1)*f})
	}
	m.ops = append(m.ops, fmt.Sprintf("find:%d:%d", scene.ID(), len(out)))
	return out
}

func (m *PathFindingModule) hbProcess(k *Kernel, self PersistID, beats int32) {}
func (m *PathFindingModule) hbRelax(k *Kernel, self PersistID, beats int32)   {}

// Ops 流水（测试面）。
func (m *PathFindingModule) Ops() []string { return m.ops }

// ---- DoSelectModule（Init 序 215）----

// DoSelectModule 复刻官方 DoSelectModule（Init @0x78FBC0；符号 13——选中
// 目标小模块；doselect.xml LastObject）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnDoSelect"/"OnEntry"/"OnMotion")
//	AddIntCommandHook ("player",613=0x265)
//	AddIntCustomHook ("player",440=0x1B8)
type DoSelectModule struct {
	BaseModule

	// events 流水（测试面："select:<id>"）。
	events []string
}

// Name 模块名。
func (m *DoSelectModule) Name() string { return "DoSelectModule" }

// Init 注册期（对齐官方 0x78FBC0）。
func (m *DoSelectModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnDoSelect", m.onDoSelect)
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnEvent("player", "OnMotion", m.onMotion)
	b.OnIntCommand("player", 613, m.onCancelSelect)
	b.OnCommand("player", 440, m.onQuerySelect)
	return nil
}

// Select 选中目标（LastObject 写 + OnDoSelect 外发）。
func (m *DoSelectModule) Select(k *Kernel, player, target PersistID) bool {
	o := k.GetGameObj(player)
	if o == nil {
		return false
	}
	if _, ok := o.Class().Prop("LastObject"); ok {
		_ = o.Set("LastObject", target)
	}
	m.events = append(m.events, fmt.Sprintf("select:%d", target.ID()))
	k.Bus().FireEvent(k, "player", "OnDoSelect", player, NullPersistID, VarList{target})
	return true
}

// onDoSelect 官方 @0x78FD60：选中事件（联动面——技能/攻击目标上下文）。
func (m *DoSelectModule) onDoSelect(k *Kernel, self, from PersistID, args VarList) {}

// onCancelSelect 官方 @0x790450（cmd 613=0x265）：取消选中（LastObject 清）。
func (m *DoSelectModule) onCancelSelect(k *Kernel, self, from PersistID, args VarList) {
	if o := k.GetGameObj(self); o != nil {
		if _, ok := o.Class().Prop("LastObject"); ok {
			_ = o.Set("LastObject", NullPersistID)
		}
		m.events = append(m.events, fmt.Sprintf("cancel:%d", self.ID()))
	}
}

// onQuerySelect 官方 @0x790360（cust 440=0x1B8）：当前选中查询。
func (m *DoSelectModule) onQuerySelect(k *Kernel, self, from PersistID, args VarList) {
	if o := k.GetGameObj(self); o != nil {
		if v, ok := o.Get("LastObject").(PersistID); ok && v.Valid() {
			m.events = append(m.events, fmt.Sprintf("query:%d:%d", self.ID(), v.ID()))
		}
	}
}

func (m *DoSelectModule) onEntry(k *Kernel, self, from PersistID, args VarList) {}
func (m *DoSelectModule) onMotion(k *Kernel, self, from PersistID, args VarList) {
	// 移动打断选中（官方 OnMotion 面：超距清选中——距离门由移动域定，
	// Go 面保留事件面）。
}

// Events 流水（测试面）。
func (m *DoSelectModule) Events() []string { return m.events }

// ---- 协作取用面 ----

func reliveModuleOf(k *Kernel) *ReliveModule {
	rm, _ := k.GetLogicModule("ReliveModule").(*ReliveModule)
	return rm
}
