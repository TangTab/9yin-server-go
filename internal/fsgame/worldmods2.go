package fsgame

// 十四期：世界规则与状态十四模块（docs/official/FsGame全解与Go复刻.md §8
// 世界规则与状态表）。官方源码树 commonmodule\ + playerbasemodule\ 等。
//
// 真档 schema（经 player.xml includes 链）：honor.xml（PlayerHonor/
// RankLevel/UpgradeHonor/CanAddHonorPoint/HonorLastRefresh）、title.xml
//（TitleID + title_rec(5)/title_unlock_rec/title_progress_rec/title_total_rec）、
// punisher.xml（RFC 怒火/RFCPara/PunisherState）、exile.xml（ExileStatus/
// ExileLockDate/ExileLockTime 天）、in_water.xml（CurUnderWaterValue 呼吸值/
// UnderWaterSpeed/ModifySpeed）、force_repute.xml（Force/ForceVersion）、
// force_state.xml（ForceState）、unenthrall.xml（UnenthrallTime 累计在线）。

import (
	"fmt"
	"sort"
	"strings"
)

// ---- HonorValueModule（Init 序 119）----

// HonorValueModule 复刻官方 HonorValueModule（Init @0x272D60；符号 579）——
// 荣誉/军阶。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover")
//	AddIntCommandHook ("player",616=0x268 被杀荣誉结算)
//	DeclareCritical ("player","PlayerHonor",C_OnHonorChanged)
//	DeclareHeartBeat ("HonorValueModule::H_UpDateMaxHonor"——每日上限刷新)
type HonorValueModule struct {
	BaseModule

	// events 流水（测试面："honor+<n>"/"rank:<lv>"）。
	events []string
}

// Name 模块名。
func (m *HonorValueModule) Name() string { return "HonorValueModule" }

// Init 注册期（对齐官方 0x272D60）。
func (m *HonorValueModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnIntCommand("player", 616, m.onBeKilled)
	b.Critical("player", "PlayerHonor", m.onHonorChanged)
	b.HeartBeat("HonorValueModule::H_UpDateMaxHonor", m.hbUpDateMaxHonor)
	return nil
}

// AddHonor 荣誉入账（CanAddHonorPoint 每日上限钳）→ 升阶判定
// （PlayerHonor ≥ UpgradeHonor → RankLevel+1）。
func (m *HonorValueModule) AddHonor(k *Kernel, player PersistID, n int32) bool {
	o := k.GetGameObj(player)
	if o == nil || n <= 0 {
		return false
	}
	if limit := o.GetInt("CanAddHonorPoint"); limit > 0 && n > limit {
		n = limit
	}
	_ = k.SetInt(player, "PlayerHonor", o.GetInt("PlayerHonor")+n)
	m.events = append(m.events, fmt.Sprintf("honor+%d", n))
	return true
}

// onHonorChanged 官方临界：荣誉变更 → 升阶（RankLevel 递进 + 新阈值）。
func (m *HonorValueModule) onHonorChanged(k *Kernel, self PersistID, prop string, old Var) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	for o.GetInt("PlayerHonor") >= o.GetInt("UpgradeHonor") && o.GetInt("UpgradeHonor") > 0 {
		_ = k.SetInt(self, "RankLevel", o.GetInt("RankLevel")+1)
		// 升阶阈值：官方按表（军阶级差）；Go 收编线性 +1000。
		_ = k.SetInt(self, "UpgradeHonor", o.GetInt("UpgradeHonor")+1000)
		m.events = append(m.events, fmt.Sprintf("rank:%d", o.GetInt("RankLevel")))
	}
}

// onBeKilled 官方 @0x272F00（616）：被杀荣誉结算（击杀者收益面）。
func (m *HonorValueModule) onBeKilled(k *Kernel, self, from PersistID, args VarList) {
	if len(args) == 0 {
		return
	}
	if killer, ok := args[0].(PersistID); ok && killer.Valid() && killer != self {
		m.AddHonor(k, killer, 10) // 击杀荣誉（官方按表；Go 缺省 10）
	}
}

// hbUpDateMaxHonor 官方心跳：每日上限刷新（HonorLastRefresh 跨日 →
// CanAddHonorPoint 复位——官方按表；Go 收编当日零点锚）。
func (m *HonorValueModule) hbUpDateMaxHonor(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" {
		return
	}
	last := o.GetInt64("HonorLastRefresh")
	today := unixNow() - (unixNow()+8*3600)%86400 // UTC+8 当日零点锚
	if last < today {
		_ = o.Set("HonorLastRefresh", today)
		_ = o.Set("CanAddHonorPoint", 3000)
	}
}

func (m *HonorValueModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}

// Events 流水（测试面）。
func (m *HonorValueModule) Events() []string { return m.events }

// ---- AirportModule（Init 序 120）----

// AirportModule 复刻官方 AirportModule（Init @0x24FBE0；符号 329）——机场
// 航线（乘机传送）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("CommonNpc","OnSelect") + ("player","OnMotion"/
//	    "OnEntry"/"OnReady"/"OnStore")
//	AddIntCommandHook ("player",22 菜单)
//	AddIntCustomHook ("player",553=0x231 航线请求)
//	DeclareLuaExt nx_fly_unlink / nx_active_airport / nx_reload_trans_path
type AirportModule struct {
	BaseModule

	// flights 在飞机票（玩家 → 航线目标坐标）。
	flights map[PersistID][2]float32
	// events 流水（测试面："fly:<x>,<z>"）。
	events []string
}

// Name 模块名。
func (m *AirportModule) Name() string { return "AirportModule" }

// Init 注册期（对齐官方 0x24FBE0）。
func (m *AirportModule) Init(k *Kernel) error {
	m.flights = make(map[PersistID][2]float32)
	b := k.Bus()
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	b.OnEvent("player", "OnMotion", m.onMotion)
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnCommand("player", 553, m.onFlyRequest)
	return nil
}

// onMotion 官方 @0x251C50：乘机移动到位（传送落地事件面）。
func (m *AirportModule) onMotion(k *Kernel, self, from PersistID, args VarList) {}

// NxActiveAirport nx_active_airport 的 Go 等价：激活机场航线
// （args: 机场 NPC/目标坐标——玩家进入乘机态）。
func (m *AirportModule) NxActiveAirport(k *Kernel, player PersistID, tx, tz float32) bool {
	if k.GetGameObj(player) == nil {
		return false
	}
	m.flights[player] = [2]float32{tx, tz}
	m.events = append(m.events, fmt.Sprintf("active:%.0f,%.0f", tx, tz))
	return true
}

// NxFlyUnlink nx_fly_unlink 的 Go 等价：解除乘机。
func (m *AirportModule) NxFlyUnlink(k *Kernel, player PersistID) {
	delete(m.flights, player)
	m.events = append(m.events, "unlink")
}

// onFlyRequest 官方 @0x250530（cust 553=0x231）：请求起飞——乘机态玩家
// 直达目标坐标（传送面：SetPos 落位 + 解除）。
func (m *AirportModule) onFlyRequest(k *Kernel, self, from PersistID, args VarList) {
	target, ok := m.flights[self]
	if !ok {
		return
	}
	if o := k.GetGameObj(self); o != nil {
		_, y, _ := o.Pos()
		o.SetPos(target[0], y, target[1])
	}
	m.NxFlyUnlink(k, self)
	m.events = append(m.events, fmt.Sprintf("fly:%.0f,%.0f", target[0], target[1]))
}

func (m *AirportModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {}
func (m *AirportModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "menu")
}

func (m *AirportModule) onEntry(k *Kernel, self, from PersistID, args VarList) {}
func (m *AirportModule) onReady(k *Kernel, self, from PersistID, args VarList) {}
func (m *AirportModule) onStore(k *Kernel, self, from PersistID, args VarList) {}

// NxReloadTransPath nx_reload_trans_path 的 Go 等价（航线表热载面）。
func (m *AirportModule) NxReloadTransPath() { m.events = append(m.events, "reload-path") }

// Events 流水（测试面）。
func (m *AirportModule) Events() []string { return m.events }

// ---- PunisherModule（Init 序 127）----

// PunisherModule 复刻官方 PunisherModule（Init @0x22E940；符号 562）——惩罚
// 者（怒火 RFC 姿态）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover")
//	AddIntCommandHook ("player",601=0x259)
//	DeclareCritical (RC→C_OnRCChanged / RFC→C_OnRFCChanged /
//	    Dead→C_OnDeadChanged / LogicState→C_OnLogicStateChanged)
//	DeclareHeartBeat ("PunisherModule::H_BeginDEC_RFC"——怒火衰减)
type PunisherModule struct {
	BaseModule

	// events 流水（测试面："rfc+<n>"/"rfc-dec"）。
	events []string
}

// Name 模块名。
func (m *PunisherModule) Name() string { return "PunisherModule" }

// Init 注册期（对齐官方 0x22E940）。
func (m *PunisherModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnIntCommand("player", 601, m.onCommand)
	b.Critical("player", "RC", m.onRCChanged)
	b.Critical("player", "RFC", m.onRFCChanged)
	b.Critical("player", "Dead", m.onDeadChanged)
	b.HeartBeat("PunisherModule::H_BeginDEC_RFC", m.hbBeginDECRFC)
	return nil
}

// onCommand 官方 @0x22EC10（cmd 601=0x259）：怒火增减
// （args: 变化值[, 优先级]——受击/攻击积累怒火面）。
func (m *PunisherModule) onCommand(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	delta := args.Int(0)
	if delta == 0 {
		return
	}
	rfc := o.GetInt("RFC") + delta
	if rfc < 0 {
		rfc = 0
	}
	if max := int32(100); rfc > max {
		rfc = max // 怒火上限（官方按 RFCPara 表；Go 缺省 100）
	}
	_ = k.SetInt(self, "RFC", rfc)
	m.events = append(m.events, fmt.Sprintf("rfc+%d→%d", delta, rfc))
}

// hbBeginDEC_RFC 官方 @0x22EFF0：怒火衰减（离战每拍 -1；死亡清零）。
func (m *PunisherModule) hbBeginDECRFC(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" {
		return
	}
	rfc := o.GetInt("RFC")
	if rfc == 0 {
		return
	}
	if o.GetInt("Dead") != 0 {
		_ = k.SetInt(self, "RFC", 0)
		return
	}
	_ = k.SetInt(self, "RFC", rfc-1)
	m.events = append(m.events, "rfc-dec")
}

func (m *PunisherModule) onRCChanged(k *Kernel, self PersistID, prop string, old Var)   {}
func (m *PunisherModule) onRFCChanged(k *Kernel, self PersistID, prop string, old Var)  {}
func (m *PunisherModule) onDeadChanged(k *Kernel, self PersistID, prop string, old Var) {}
func (m *PunisherModule) onRecover(k *Kernel, self, from PersistID, args VarList)       {}

// Events 流水（测试面）。
func (m *PunisherModule) Events() []string { return m.events }

// ---- UnderWaterModule（Init 序 138）----

// UnderWaterModule 复刻官方 UnderWaterModule（Init @0x19DA90；符号 562）——
// 水下呼吸。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover"/"OnStore"/"OnLeaveScene")
//	AddIntCustomHook ("player",478=0x1DE 入水 / 479=0x1DF 出水)
//	AddIntCommandHook ("player",616)
//	DeclareHeartBeat ×3（H_Enter_Water_Timer / H_Out_Water_Timer /
//	    H_Hurt_Timer——呼吸耗尽持续伤害）
type UnderWaterModule struct {
	BaseModule

	// inWater 水下态（玩家 → true）。
	inWater map[PersistID]bool
	// events 流水（测试面："drown"）。
	events []string
}

// Name 模块名。
func (m *UnderWaterModule) Name() string { return "UnderWaterModule" }

// Init 注册期（对齐官方 0x19DA90）。
func (m *UnderWaterModule) Init(k *Kernel) error {
	m.inWater = make(map[PersistID]bool)
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnEvent("player", "OnLeaveScene", m.onLeaveScene)
	b.OnCommand("player", 478, m.onEnterWater)
	b.OnCommand("player", 479, m.onLeaveWater)
	b.OnIntCommand("player", 616, m.onBeKilled)
	b.HeartBeat("UnderWaterModule::H_Enter_Water_Timer", m.hbEnterWaterTimer)
	b.HeartBeat("UnderWaterModule::H_Out_Water_Timer", m.hbOutWaterTimer)
	b.HeartBeat("UnderWaterModule::H_Hurt_Timer", m.hbHurtTimer)
	return nil
}

// onEnterWater 官方 @0x19DD80（cust 478）：入水——呼吸值开始消耗。
func (m *UnderWaterModule) onEnterWater(k *Kernel, self, from PersistID, args VarList) {
	m.inWater[self] = true
}

// onLeaveWater 官方 @0x19DEC0（cust 479）：出水——呼吸值回复。
func (m *UnderWaterModule) onLeaveWater(k *Kernel, self, from PersistID, args VarList) {
	delete(m.inWater, self)
}

// hbEnterWaterTimer 官方：水下呼吸消耗（Speed×(1+ModifySpeed)/拍）。
func (m *UnderWaterModule) hbEnterWaterTimer(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || !m.inWater[self] {
		return
	}
	speed := o.GetFloat("UnderWaterSpeed") * (1 + o.GetFloat("UnderWaterModifySpeed"))
	cur := o.GetFloat("CurUnderWaterValue") - speed
	if cur < 0 {
		cur = 0
	}
	_ = o.Set("CurUnderWaterValue", cur)
}

// hbOutWaterTimer 官方：出水呼吸回复。
func (m *UnderWaterModule) hbOutWaterTimer(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || m.inWater[self] {
		return
	}
	if cur := o.GetFloat("CurUnderWaterValue"); cur < 100 {
		_ = o.Set("CurUnderWaterValue", cur+5)
	}
}

// hbHurtTimer 官方 @0x19E3E0：呼吸耗尽持续伤害（HP -5/拍；死亡即止）。
func (m *UnderWaterModule) hbHurtTimer(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || !m.inWater[self] || o.GetFloat("CurUnderWaterValue") > 0 || o.GetInt("Dead") != 0 {
		return
	}
	_ = k.SetInt(self, "HP", o.GetInt("HP")-5)
	m.events = append(m.events, "drown")
}

func (m *UnderWaterModule) onBeKilled(k *Kernel, self, from PersistID, args VarList) {}
func (m *UnderWaterModule) onRecover(k *Kernel, self, from PersistID, args VarList)  {}
func (m *UnderWaterModule) onStore(k *Kernel, self, from PersistID, args VarList)    {}
func (m *UnderWaterModule) onLeaveScene(k *Kernel, self, from PersistID, args VarList) {
	delete(m.inWater, self)
}

// Events 流水（测试面）。
func (m *UnderWaterModule) Events() []string { return m.events }

// ---- ReputeModule / CampModule / ForceModule / ForceStateModule ----

// ReputeModule 复刻官方 ReputeModule（Init @0x18EE40；符号 231）——声望
// （force_repute.xml：Force/ForceVersion）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover")
//	DeclareLuaExt nx_add_force / nx_add_repute
type ReputeModule struct {
	BaseModule

	// repute 声望镜像（玩家 → 势力 → 值）。
	repute map[PersistID]map[int32]int32
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *ReputeModule) Name() string { return "ReputeModule" }

// Init 注册期（对齐官方 0x18EE40）。
func (m *ReputeModule) Init(k *Kernel) error {
	m.repute = make(map[PersistID]map[int32]int32)
	k.Bus().OnEvent("player", "OnRecover", m.onRecover)
	return nil
}

// NxAddRepute nx_add_repute 的 Go 等价：声望入账（ForceVersion 递增——
// 势力关系版本变更面）。
func (m *ReputeModule) NxAddRepute(k *Kernel, player PersistID, force, value int32) bool {
	o := k.GetGameObj(player)
	if o == nil {
		return false
	}
	if m.repute[player] == nil {
		m.repute[player] = make(map[int32]int32)
	}
	m.repute[player][force] += value
	if _, ok := o.Class().Prop("ForceVersion"); ok {
		_ = k.SetInt(player, "ForceVersion", o.GetInt("ForceVersion")+1)
	}
	m.events = append(m.events, fmt.Sprintf("repute:%d+%d", force, value))
	return true
}

// ReputeOf 声望查询。
func (m *ReputeModule) ReputeOf(player PersistID, force int32) int32 {
	return m.repute[player][force]
}

// NxAddForce nx_add_force 的 Go 等价：势力归属变更（Force 写 + 版本递增）。
func (m *ReputeModule) NxAddForce(k *Kernel, player PersistID, force int32) bool {
	o := k.GetGameObj(player)
	if o == nil {
		return false
	}
	if _, ok := o.Class().Prop("Force"); !ok {
		return false
	}
	old := o.GetInt("Force")
	if old == force {
		return false
	}
	_ = k.SetInt(player, "Force", force)
	if _, ok := o.Class().Prop("ForceVersion"); ok {
		_ = k.SetInt(player, "ForceVersion", o.GetInt("ForceVersion")+1)
	}
	m.events = append(m.events, fmt.Sprintf("force:%d→%d", old, force))
	return true
}

func (m *ReputeModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}

// Events 流水（测试面）。
func (m *ReputeModule) Events() []string { return m.events }

// CampModule 复刻官方 CampModule（Init @0x251F20；符号 790；零内核注册——
// 阵营 API：Force 派生面）。
type CampModule struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

func (m *CampModule) Name() string         { return "CampModule" }
func (m *CampModule) Init(k *Kernel) error { return nil }

// Camp 阵营查询（Force 同源——阵营为势力派生面）。
func (m *CampModule) Camp(k *Kernel, player PersistID) int32 {
	if o := k.GetGameObj(player); o != nil {
		return o.GetInt("Force")
	}
	return 0
}

// Events 流水（测试面）。
func (m *CampModule) Events() []string { return m.events }

// ForceStateModule 复刻官方 ForceStateModule（Init @0x249D00；符号 724；
// 零内核注册——势力状态 API：force_state.xml ForceState）。
type ForceStateModule struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

func (m *ForceStateModule) Name() string         { return "ForceStateModule" }
func (m *ForceStateModule) Init(k *Kernel) error { return nil }

// SetForceState 势力状态写（ForceState——战争/和平态面）。
func (m *ForceStateModule) SetForceState(k *Kernel, player PersistID, state int32) bool {
	o := k.GetGameObj(player)
	if o == nil {
		return false
	}
	if _, ok := o.Class().Prop("ForceState"); !ok {
		return false
	}
	_ = k.SetInt(player, "ForceState", state)
	m.events = append(m.events, fmt.Sprintf("state:%d", state))
	return true
}

// Events 流水（测试面）。
func (m *ForceStateModule) Events() []string { return m.events }

// ForceModule 复刻官方 ForceModule（Init @0x249D00；符号 1074；零内核注册——
// 势力 API 主面）。
type ForceModule struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

func (m *ForceModule) Name() string         { return "ForceModule" }
func (m *ForceModule) Init(k *Kernel) error { return nil }

// JoinForce 加入势力（ReputeModule::NxAddForce 转发 + 阵营状态复位）。
func (m *ForceModule) JoinForce(k *Kernel, player PersistID, force int32) bool {
	rm, _ := k.GetLogicModule("ReputeModule").(*ReputeModule)
	if rm == nil || !rm.NxAddForce(k, player, force) {
		return false
	}
	if fsm, _ := k.GetLogicModule("ForceStateModule").(*ForceStateModule); fsm != nil {
		fsm.SetForceState(k, player, 0)
	}
	m.events = append(m.events, fmt.Sprintf("join:%d", force))
	return true
}

// Events 流水（测试面）。
func (m *ForceModule) Events() []string { return m.events }

// ---- TitleModule（Init 序 142）----

// TitleModule 复刻官方 TitleModule（Init @0x194720；符号 1382）——称号。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCustomHook ("player",295=0x127)
//	AddEventCallback ("player","OnEntryGame")
//	DeclareLuaExt nx_add_playertitle / nx_remove_playertitle / nx_set_playertitle
//	DeclareRecHook (title_rec → OnTitleRecChanged)
//	DeclareHeartBeat ("TitleModule::HB_CheckTitleOverTime")
//	DeclareCritical ("player","Level",C_OnLevelChanged)
type TitleModule struct {
	BaseModule

	// events 流水（测试面："title+<id>"/"wear:<id>"）。
	events []string
}

// Name 模块名。
func (m *TitleModule) Name() string { return "TitleModule" }

// Init 注册期（对齐官方 0x194720）。
func (m *TitleModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnCommand("player", 295, m.onCustomTitle)
	b.OnEvent("player", "OnEntryGame", m.onEntryGame)
	b.RecHook("title_rec", m.onTitleRecChanged)
	b.HeartBeat("TitleModule::HB_CheckTitleOverTime", m.hbCheckTitleOverTime)
	b.Critical("player", "Level", m.onLevelChanged)
	return nil
}

// NxAddPlayerTitle nx_add_playertitle 的 Go 等价：称号解锁
// （title_rec 追加：{称号, 剩余秒(0=永久), …}——5 列方言）。
func (m *TitleModule) NxAddPlayerTitle(k *Kernel, player PersistID, titleID int32, seconds int32) bool {
	o := k.GetGameObj(player)
	r := o.Record("title_rec")
	if r == nil || r.Rows() >= r.Def().MaxRows {
		return false
	}
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).Int(0) == titleID {
			return false // 已持有
		}
	}
	r.AppendRow(VarList{titleID, seconds, unixNow(), int32(0), int32(0)})
	m.events = append(m.events, fmt.Sprintf("title+%d", titleID))
	return true
}

// NxRemovePlayerTitle nx_remove_playertitle 的 Go 等价：移除（佩戴中同卸）。
func (m *TitleModule) NxRemovePlayerTitle(k *Kernel, player PersistID, titleID int32) bool {
	o := k.GetGameObj(player)
	r := o.Record("title_rec")
	if r == nil {
		return false
	}
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).Int(0) == titleID {
			r.DeleteRow(i)
			if o.GetInt("TitleID") == titleID {
				_ = k.SetInt(player, "TitleID", 0)
			}
			m.events = append(m.events, fmt.Sprintf("title-%d", titleID))
			return true
		}
	}
	return false
}

// NxSetPlayerTitle nx_set_playertitle 的 Go 等价：佩戴（须已持有）。
func (m *TitleModule) NxSetPlayerTitle(k *Kernel, player PersistID, titleID int32) bool {
	o := k.GetGameObj(player)
	r := o.Record("title_rec")
	if r == nil {
		return false
	}
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).Int(0) == titleID {
			_ = k.SetInt(player, "TitleID", titleID)
			m.events = append(m.events, fmt.Sprintf("wear:%d", titleID))
			return true
		}
	}
	return false
}

// hbCheckTitleOverTime 官方：限时称号到期移除（col1 剩余秒递减）。
func (m *TitleModule) hbCheckTitleOverTime(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" {
		return
	}
	r := o.Record("title_rec")
	if r == nil {
		return
	}
	for i := r.Rows() - 1; i >= 0; i-- {
		left := r.Row(i).Int(1)
		if left <= 0 {
			continue // 永久
		}
		left--
		if left == 0 {
			m.NxRemovePlayerTitle(k, self, r.Row(i).Int(0))
			m.events = append(m.events, "overtime")
			continue
		}
		row := r.Row(i)
		row[1] = left
		r.ModifyRow(i, row)
	}
}

// onTitleRecChanged 官方 RecHook：称号表变更联动（佩戴面刷新下发）。
func (m *TitleModule) onTitleRecChanged(k *Kernel, self PersistID, record string, op, row int32) {
	m.events = append(m.events, fmt.Sprintf("rec:%d", op))
}

// onLevelChanged 官方临界：升级 → 等级门称号解锁检查（配置面）。
func (m *TitleModule) onLevelChanged(k *Kernel, self PersistID, prop string, old Var) {}

// onCustomTitle 官方（cust 295=0x127）：称号 UI 消息（args: 子命令, 称号）。
func (m *TitleModule) onCustomTitle(k *Kernel, self, from PersistID, args VarList) {
	switch args.Int(0) {
	case 1: // 佩戴
		m.NxSetPlayerTitle(k, self, args.Int(1))
	case 2: // 卸下
		_ = k.SetInt(self, "TitleID", 0)
	}
}

func (m *TitleModule) onEntryGame(k *Kernel, self, from PersistID, args VarList) {}

// Events 流水（测试面）。
func (m *TitleModule) Events() []string { return m.events }

// ---- WorldRankModule（Init 序 174）----

// WorldRankModule 复刻官方 WorldRankModule（Init @0x8980E0；符号 149）——
// 世界排行。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCustomHook ("player",963=0x3C3)
//	AddEventCallback ("player","OnLeave"/"OnRecover") +
//	    ("scene","OnPublicMessage")
//	AddIntCommandHook ("player",884=0x374 / 800=0x320 提交排行)
//	DeclareCritical ("player","PlayerHonor"/"ArenaSinglePoints")
//	DeclareLuaExt nx_clear_rank
type WorldRankModule struct {
	BaseModule

	// ranks 排行镜像（域 → 玩家 → 分值；域 0=荣誉 1=竞技场）。
	ranks map[int32]map[PersistID]int64
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *WorldRankModule) Name() string { return "WorldRankModule" }

// Init 注册期（对齐官方 0x8980E0）。
func (m *WorldRankModule) Init(k *Kernel) error {
	m.ranks = map[int32]map[PersistID]int64{0: {}, 1: {}}
	b := k.Bus()
	b.OnCommand("player", 963, m.onCustomRank)
	b.OnEvent("player", "OnLeave", m.onLeave)
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("scene", "OnPublicMessage", m.onPublicMessage)
	b.OnIntCommand("player", 884, m.onSubmit)
	b.OnIntCommand("player", 800, m.onSubmit)
	b.Critical("player", "PlayerHonor", m.onHonorChanged)
	b.Critical("player", "ArenaSinglePoints", m.onArenaPointsChanged)
	return nil
}

// onHonorChanged 官方临界：荣誉分值镜像（域 0）。
func (m *WorldRankModule) onHonorChanged(k *Kernel, self PersistID, prop string, old Var) {
	if o := k.GetGameObj(self); o != nil {
		m.ranks[0][self] = int64(o.GetInt("PlayerHonor"))
	}
}

// onArenaPointsChanged 官方临界：竞技场分值镜像（域 1）。
func (m *WorldRankModule) onArenaPointsChanged(k *Kernel, self PersistID, prop string, old Var) {
	if o := k.GetGameObj(self); o != nil {
		m.ranks[1][self] = int64(o.GetInt("ArenaSinglePoints"))
	}
}

// onSubmit 官方 @0x89A990（cmd 800/884）：提交排行（分值快照入榜）。
func (m *WorldRankModule) onSubmit(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, fmt.Sprintf("submit:%d", self.ID()))
}

// Top 域内前 N（分值降序；并列按提交序）。
func (m *WorldRankModule) Top(domain int32, n int) []PersistID {
	type entry struct {
		id PersistID
		v  int64
	}
	var list []entry
	for id, v := range m.ranks[domain] {
		list = append(list, entry{id, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].v > list[j].v })
	if len(list) > n {
		list = list[:n]
	}
	out := make([]PersistID, len(list))
	for i, e := range list {
		out[i] = e.id
	}
	return out
}

// NxClearRank nx_clear_rank 的 Go 等价：清域。
func (m *WorldRankModule) NxClearRank(domain int32) {
	m.ranks[domain] = make(map[PersistID]int64)
	m.events = append(m.events, "clear")
}

func (m *WorldRankModule) onCustomRank(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, fmt.Sprintf("cust:%d", args.Int(0)))
}

// onPublicMessage 官方：公共消息面（跨服榜单同步方言——Go 收编 trace）。
func (m *WorldRankModule) onPublicMessage(k *Kernel, self, from PersistID, args VarList) {}

func (m *WorldRankModule) onLeave(k *Kernel, self, from PersistID, args VarList)   {}
func (m *WorldRankModule) onRecover(k *Kernel, self, from PersistID, args VarList) {}

// Events 流水（测试面）。
func (m *WorldRankModule) Events() []string { return m.events }

// ---- ExileModule（Init 序 193）----

// ExileModule 复刻官方 ExileModule（Init @0x7CF7A0；符号 184）——流放/监禁。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnStore"/"OnRecover"/"OnReady") +
//	    ("CommonNpc","OnSelect")
//	AddIntCommandHook ("player",22 菜单 / 322=0x142)
//	AddIntCustomHook ("player",260=0x104)
//	DeclareCritical (ArrestValue→C_OnArrestValueChanged / Dead)
//	DeclareLuaExt nx_set_status
//	DeclareHeartBeat ("ExileModule::HB_CheckLockTime")
type ExileModule struct {
	BaseModule

	// events 流水（测试面："arrest:<v>"/"release")。
	events []string
}

// Name 模块名。
func (m *ExileModule) Name() string { return "ExileModule" }

// Init 注册期（对齐官方 0x7CF7A0）。
func (m *ExileModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	b.OnIntCommand("player", 322, m.onCommand)
	b.OnCommand("player", 260, m.onCustomExile)
	b.Critical("player", "ArrestValue", m.onArrestValueChanged)
	b.Critical("player", "Dead", m.onDeadChanged)
	b.HeartBeat("ExileModule::HB_CheckLockTime", m.hbCheckLockTime)
	return nil
}

// onArrestValueChanged 官方 @0x7D0120：通缉值变更 → 身份态
// （ExileStatus：0 平民 / >0 通缉）。
func (m *ExileModule) onArrestValueChanged(k *Kernel, self PersistID, prop string, old Var) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	v := o.GetInt("ArrestValue")
	status := int32(0)
	if v > 0 {
		status = 1
	}
	if v >= 100 {
		status = 2 // 重犯（官方按表；Go 收编两档）
	}
	if o.GetInt("ExileStatus") != status {
		_ = k.SetInt(self, "ExileStatus", status)
	}
	m.events = append(m.events, fmt.Sprintf("arrest:%d→%d", v, status))
}

// NxSetStatus nx_set_status 的 Go 等价：监禁执行（ExileLockTime 天 +
// ExileLockDate 落锚）。
func (m *ExileModule) NxSetStatus(k *Kernel, player PersistID, days int32) bool {
	o := k.GetGameObj(player)
	if o == nil || days <= 0 {
		return false
	}
	if _, ok := o.Class().Prop("ExileLockTime"); !ok {
		return false
	}
	_ = o.Set("ExileLockTime", days)
	_ = o.Set("ExileLockDate", float64(unixNow()))
	m.events = append(m.events, fmt.Sprintf("lock:%dd", days))
	return true
}

// hbCheckLockTime 官方 @0x7D1F00：刑期到满释放（ExileStatus 复位 +
// ArrestValue 清零）。
func (m *ExileModule) hbCheckLockTime(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" || o.GetInt("ExileStatus") == 0 {
		return
	}
	lockDate := int64(o.GetDouble("ExileLockDate"))
	days := o.GetInt("ExileLockTime")
	if days <= 0 || unixNow() < lockDate+int64(days)*86400 {
		return
	}
	_ = k.SetInt(self, "ExileStatus", 0)
	_ = o.Set("ExileLockTime", int32(0))
	if _, ok := o.Class().Prop("ArrestValue"); ok {
		_ = k.SetInt(self, "ArrestValue", 0)
	}
	m.events = append(m.events, "release")
}

func (m *ExileModule) onDeadChanged(k *Kernel, self PersistID, prop string, old Var) {}
func (m *ExileModule) onStore(k *Kernel, self, from PersistID, args VarList)         {}
func (m *ExileModule) onRecover(k *Kernel, self, from PersistID, args VarList)       {}
func (m *ExileModule) onReady(k *Kernel, self, from PersistID, args VarList)         {}
func (m *ExileModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList)     {}
func (m *ExileModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList)    {}
func (m *ExileModule) onCommand(k *Kernel, self, from PersistID, args VarList)       {}
func (m *ExileModule) onCustomExile(k *Kernel, self, from PersistID, args VarList)   {}

// Events 流水（测试面）。
func (m *ExileModule) Events() []string { return m.events }

// ---- NewsModule（Init 序 199）----

// NewsModule 复刻官方 NewsModule（Init @0x6FEE60；符号 65；单 Lua 面）——
// 世界新闻/公告发布。
type NewsModule struct {
	BaseModule

	// NewsOut 公告出口（sync 层转公告面板方言；默认流水）。
	NewsOut func(channel int32, text string)
	// events 流水（测试面："news:<ch>:<text>"）。
	events []string
}

// Name 模块名。
func (m *NewsModule) Name() string { return "NewsModule" }

// Init 注册期（官方单 Lua 面 nx_publish_news）。
func (m *NewsModule) Init(k *Kernel) error {
	if m.NewsOut == nil {
		m.NewsOut = func(channel int32, text string) {}
	}
	return nil
}

// NxPublishNews nx_publish_news 的 Go 等价：发布公告。
func (m *NewsModule) NxPublishNews(channel int32, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	m.NewsOut(channel, text)
	m.events = append(m.events, fmt.Sprintf("news:%d:%s", channel, text))
}

// Events 流水（测试面）。
func (m *NewsModule) Events() []string { return m.events }

// ---- MovieModule（Init 序 207）----

// MovieModule 复刻官方 MovieModule（Init @0x6C5330；符号 47）——剧情/过场
// 播放。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCustomHook ("player",806=0x326 / 808=0x328 / 820=0x334 /
//	    821=0x335 / 1009=0x3F1)
//	DeclareHeartBeat ×3（HB_LaterPlay / HB_CantBeAttack / HB_CancelWaitMovie）
//	DeclareLuaExt nx_beigin_movie
type MovieModule struct {
	BaseModule

	// playing 播放中（玩家 → 电影 ID）。
	playing map[PersistID]int32
	// events 流水（测试面："movie:<id>"/"movie-end")。
	events []string
}

// Name 模块名。
func (m *MovieModule) Name() string { return "MovieModule" }

// Init 注册期（对齐官方 0x6C5330）。
func (m *MovieModule) Init(k *Kernel) error {
	m.playing = make(map[PersistID]int32)
	b := k.Bus()
	b.OnCommand("player", 806, m.onCustomMovie)
	b.OnCommand("player", 808, m.onCustomMovie)
	b.OnCommand("player", 820, m.onCustomMovie)
	b.OnCommand("player", 821, m.onCustomMovie)
	b.OnCommand("player", 1009, m.onCustomMovie)
	b.HeartBeat("MovieModule::HB_LaterPlay", m.hbLaterPlay)
	b.HeartBeat("MovieModule::HB_CantBeAttack", m.hbCantBeAttack)
	b.HeartBeat("MovieModule::HB_CancelWaitMovie", m.hbCancelWaitMovie)
	return nil
}

// NxBeiginMovie nx_beigin_movie 的 Go 等价：开始播放（播放态登记 +
// CantBeAttack 保护面）。
func (m *MovieModule) NxBeiginMovie(k *Kernel, player PersistID, movieID int32) bool {
	if k.GetGameObj(player) == nil {
		return false
	}
	m.playing[player] = movieID
	if o := k.GetGameObj(player); o != nil {
		if _, ok := o.Class().Prop("CantBeAttack"); ok {
			_ = k.SetInt(player, "CantBeAttack", 1)
		}
	}
	m.events = append(m.events, fmt.Sprintf("movie:%d", movieID))
	return true
}

// EndMovie 结束播放（保护面解除）。
func (m *MovieModule) EndMovie(k *Kernel, player PersistID) {
	if _, ok := m.playing[player]; !ok {
		return
	}
	delete(m.playing, player)
	if o := k.GetGameObj(player); o != nil {
		if _, ok := o.Class().Prop("CantBeAttack"); ok {
			_ = k.SetInt(player, "CantBeAttack", 0)
		}
	}
	m.events = append(m.events, "movie-end")
}

// hbLaterPlay 官方：延迟播放泵（排队面）。
func (m *MovieModule) hbLaterPlay(k *Kernel, self PersistID, beats int32) {}

// hbCantBeAttack 官方：播放期免打面维持。
func (m *MovieModule) hbCantBeAttack(k *Kernel, self PersistID, beats int32) {}

// hbCancelWaitMovie 官方：取消排队。
func (m *MovieModule) hbCancelWaitMovie(k *Kernel, self PersistID, beats int32) {}

func (m *MovieModule) onCustomMovie(k *Kernel, self, from PersistID, args VarList) {
	switch args.Int(0) {
	case 1: // 结束上报
		m.EndMovie(k, self)
	default:
		m.events = append(m.events, fmt.Sprintf("cust:%d", args.Int(0)))
	}
}

// Events 流水（测试面）。
func (m *MovieModule) Events() []string { return m.events }

// ---- UnenthrallModule（Init 序 213）----

// UnenthrallModule 复刻官方 UnenthrallModule（Init @0x66C040；符号 12）——
// 防沉迷（UnenthrallTime 累计在线）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCustomHook ("player",306=0x132) / AddIntCommandHook (277=0x115)
//	AddEventCallback ("player","OnContinue"/"OnReady") +
//	    ("scene","OnGetPlayTime"/"OnChargeMessage")
//	DeclareHeartBeat ("UnenthrallModule::HB_Unenthrall")
type UnenthrallModule struct {
	BaseModule

	// events 流水（测试面："fatigue:<h>h"）。
	events []string
}

// Name 模块名。
func (m *UnenthrallModule) Name() string { return "UnenthrallModule" }

// Init 注册期（对齐官方 0x66C040）。
func (m *UnenthrallModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnCommand("player", 306, m.onCustomUnenthrall)
	b.OnIntCommand("player", 277, m.onCommand)
	b.OnEvent("player", "OnContinue", m.onContinue)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("scene", "OnGetPlayTime", m.onGetPlayTime)
	b.OnEvent("scene", "OnChargeMessage", m.onChargeMessage)
	b.HeartBeat("UnenthrallModule::HB_Unenthrall", m.hbUnenthrall)
	return nil
}

// hbUnenthrall 官方 @0x66C5A0：累计在线（UnenthrallTime 秒）→ 疲劳档
// （3h/5h 官方面：收益减半/清零——Go 收编档位流水）。
func (m *UnenthrallModule) hbUnenthrall(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() != "player" {
		return
	}
	t := o.GetInt("UnenthrallTime") + 1
	_ = o.Set("UnenthrallTime", t)
	switch t {
	case 3 * 3600:
		m.events = append(m.events, "fatigue:3h")
	case 5 * 3600:
		m.events = append(m.events, "fatigue:5h")
	}
}

// onChargeMessage 官方：充值面（成年人实名解除——Go 收编流水）。
func (m *UnenthrallModule) onChargeMessage(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "charge")
}

// onGetPlayTime 官方：在线时长查询下发。
func (m *UnenthrallModule) onGetPlayTime(k *Kernel, self, from PersistID, args VarList) {}

func (m *UnenthrallModule) onCustomUnenthrall(k *Kernel, self, from PersistID, args VarList) {}
func (m *UnenthrallModule) onCommand(k *Kernel, self, from PersistID, args VarList)          {}
func (m *UnenthrallModule) onContinue(k *Kernel, self, from PersistID, args VarList)         {}
func (m *UnenthrallModule) onReady(k *Kernel, self, from PersistID, args VarList)            {}

// Events 流水（测试面）。
func (m *UnenthrallModule) Events() []string { return m.events }
