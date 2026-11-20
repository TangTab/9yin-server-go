package fsgame

// 十二期：物品底座余量九模块（docs/official/FsGame全解与Go复刻.md §8
// 物品底座表余量）。官方源码树 itembasemodule\ + commonmodule\ 等。
//
// 真档 schema（struct/item/item.xml 物品基类）：BindType/BindStatus（绑定
// 类型/状态）、ValidTimeType（0 永久 / 1 绝对点 ExpiredTime「年.月.日.时.分」
// / 2 相对秒 ValidTime）+ LeftTime（剩余）、MaxCount/Task/CantDepot/
// CantSell/CantDelete、Amount/MaxAmount、Only（唯一）；equipment.xml：
// PropPack/PropBuilt（固定/强化属性包串）、Hardiness/MaxHardiness、
// EquipType、ResolvePack（分解包）；container 族（addcontainer/depot 等）。

import (
	"fmt"
	"strings"
	"time"
)

// ---- ItemBindModule（Init 序 97）----

// ItemBindModule 复刻官方 ItemBindModule（Init @0x9A6B0；符号 27）——物品
// 绑定（拾取/装备/使用即绑）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("ToolBox"/"EquipBox"/"MountBox","OnAfterAdd")
//	AddEventCallback ("item","OnLoad"/"OnRecover")
//	DeclareCritical ("item","BindStatus",C_OnBindStatusChanged)
type ItemBindModule struct {
	BaseModule

	// events 绑定流水（测试面："bind:<configID>"）。
	events []string
}

// Name 模块名。
func (m *ItemBindModule) Name() string { return "ItemBindModule" }

// Init 注册期（对齐官方 0x9A6B0）。
func (m *ItemBindModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("ToolBox", "OnAfterAdd", m.onAfterAdd)
	b.OnEvent("EquipBox", "OnAfterAdd", m.onAfterAdd)
	b.OnEvent("MountBox", "OnAfterAdd", m.onAfterAdd)
	b.OnEvent("item", "OnLoad", m.onItemLoad)
	b.OnEvent("item", "OnRecover", m.onItemLoad)
	b.Critical("item", "BindStatus", m.onBindStatusChanged)
	return nil
}

// onAfterAdd 官方 @0x9ABA0 族：物品入包/装备——BindType≠0 即绑
// （args[0]=物品对象）。
func (m *ItemBindModule) onAfterAdd(k *Kernel, self, from PersistID, args VarList) {
	if len(args) == 0 {
		return
	}
	item, _ := args[0].(PersistID)
	o := k.GetGameObj(item)
	if o == nil || o.GetInt("BindType") == 0 || o.GetInt("BindStatus") != 0 {
		return
	}
	_ = k.SetInt(item, "BindStatus", 1)
	m.events = append(m.events, fmt.Sprintf("bind:%s", o.GetString("ConfigID")))
}

func (m *ItemBindModule) onItemLoad(k *Kernel, self, from PersistID, args VarList) {}

// onBindStatusChanged 官方临界：绑定态变更（下发/交易禁面）。
func (m *ItemBindModule) onBindStatusChanged(k *Kernel, self PersistID, prop string, old Var) {}

// Events 绑定流水（测试面）。
func (m *ItemBindModule) Events() []string { return m.events }

// ---- ItemValidTimeModule（Init 序 100）----

// ItemValidTimeModule 复刻官方 ItemValidTimeModule（Init @0x8D710；符号 48）——
// 物品时效。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("item","OnLoad"/"OnRecover"/"OnEntry")
//	DeclareHeartBeat ("ItemValidTimeModule::H_ValidTime")
//	DeclareCritical ("item","ValidTime",C_ValidTime)
//	DeclareLuaExt nx_set_leftime
type ItemValidTimeModule struct {
	BaseModule

	// ExpiredOut 到期出口（物品移除由容器域挂接；默认流水）。
	ExpiredOut func(player, item PersistID)
	// events 流水（测试面："expire:<configID>"）。
	events []string
}

// Name 模块名。
func (m *ItemValidTimeModule) Name() string { return "ItemValidTimeModule" }

// Init 注册期（对齐官方 0x8D710）。
func (m *ItemValidTimeModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("item", "OnLoad", m.onItemLoad)
	b.OnEvent("item", "OnRecover", m.onItemRecover)
	b.OnEvent("item", "OnEntry", m.onItemEntry)
	b.HeartBeat("ItemValidTimeModule::H_ValidTime", m.hbValidTime)
	b.Critical("item", "ValidTime", m.onValidTimeChanged)
	if m.ExpiredOut == nil {
		m.ExpiredOut = func(player, item PersistID) {}
	}
	return nil
}

// InitLeftTime 时效初始化（OnLoad/OnEntry 面）：按 ValidTimeType 折算
// LeftTime 秒——1 绝对点（ExpiredTime「年.月.日.时.分」）/ 2 相对秒。
func (m *ItemValidTimeModule) InitLeftTime(k *Kernel, item PersistID) {
	o := k.GetGameObj(item)
	if o == nil {
		return
	}
	switch o.GetInt("ValidTimeType") {
	case 1:
		if t := parseExpiredTime(o.GetString("ExpiredTime")); t > 0 {
			_ = o.Set("LeftTime", int32(t-unixNow()))
		}
	case 2:
		_ = o.Set("LeftTime", o.GetInt("ValidTime"))
	}
}

// hbValidTime 官方 H_ValidTime：LeftTime 递减 → 归零到期
// （ExpiredOut 出口 + 流水）。
func (m *ItemValidTimeModule) hbValidTime(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || o.ClassName() == "" || !isItemClass(k, self) {
		return
	}
	left := o.GetInt("LeftTime")
	if o.GetInt("ValidTimeType") == 0 || left <= 0 {
		return
	}
	left--
	_ = o.Set("LeftTime", left)
	if left == 0 {
		m.ExpiredOut(ownerOf(k, self), self)
		m.events = append(m.events, fmt.Sprintf("expire:%s", o.GetString("ConfigID")))
		k.Tracef("ItemValidTimeModule: expire %s", o.GetString("ConfigID"))
	}
}

// NxSetLeftTime nx_set_leftime 的 Go 等价（剩余时间改写）。
func (m *ItemValidTimeModule) NxSetLeftTime(k *Kernel, item PersistID, seconds int32) {
	if o := k.GetGameObj(item); o != nil {
		_ = o.Set("LeftTime", seconds)
	}
}

func (m *ItemValidTimeModule) onItemLoad(k *Kernel, self, from PersistID, args VarList) {
	m.InitLeftTime(k, self)
}
func (m *ItemValidTimeModule) onItemRecover(k *Kernel, self, from PersistID, args VarList) {
	m.InitLeftTime(k, self)
}
func (m *ItemValidTimeModule) onItemEntry(k *Kernel, self, from PersistID, args VarList) {
	m.InitLeftTime(k, self)
}

func (m *ItemValidTimeModule) onValidTimeChanged(k *Kernel, self PersistID, prop string, old Var) {
	m.InitLeftTime(k, self)
}

// Events 流水（测试面）。
func (m *ItemValidTimeModule) Events() []string { return m.events }

// parseExpiredTime 绝对点解析（「年.月.日.时.分」——item.xml desc 定案）。
func parseExpiredTime(s string) int64 {
	f := strings.Split(strings.TrimSpace(s), ".")
	if len(f) < 5 {
		return 0
	}
	y, m1, d, h, mi := atoi32(f[0]), atoi32(f[1]), atoi32(f[2]), atoi32(f[3]), atoi32(f[4])
	if y < 2000 || m1 < 1 || m1 > 12 || d < 1 || d > 31 {
		return 0
	}
	return time.Date(int(y), time.Month(m1), int(d), int(h), int(mi), 0, 0, time.Local).Unix()
}

// ---- ItemLimited（Init 序 96）----

// ItemLimited 复刻官方 ItemLimited（Init @0x9BDC0；符号 565；零内核注册——
// 物品限制校验 API）。
type ItemLimited struct {
	BaseModule
}

// Name 模块名。
func (m *ItemLimited) Name() string { return "ItemLimited" }

// Init 注册期（官方零注册——API 模块）。
func (m *ItemLimited) Init(k *Kernel) error { return nil }

// CanOwn 拥有上限门（MaxCount；0=不限）。
func (m *ItemLimited) CanOwn(k *Kernel, owned int32, item PersistID) bool {
	o := k.GetGameObj(item)
	if o == nil {
		return true
	}
	max := o.GetInt("MaxCount")
	return max <= 0 || owned < max
}

// CanDepot 入仓门（CantDepot>0 拒；绑定物拒）。
func (m *ItemLimited) CanDepot(k *Kernel, item PersistID) bool {
	o := k.GetGameObj(item)
	return o != nil && o.GetInt("CantDepot") == 0 && o.GetInt("BindStatus") == 0
}

// CanSell 贩卖门（CantSell>0 拒）。
func (m *ItemLimited) CanSell(k *Kernel, item PersistID) bool {
	o := k.GetGameObj(item)
	return o != nil && o.GetInt("CantSell") == 0
}

// CanDelete 删除门（CantDelete>0 拒；任务道具拒）。
func (m *ItemLimited) CanDelete(k *Kernel, item PersistID) bool {
	o := k.GetGameObj(item)
	return o != nil && o.GetInt("CantDelete") == 0 && o.GetInt("Task") == 0
}

// ---- ItemProCompute（Init 序 95）----

// ItemProCompute 复刻官方 ItemProCompute（Init @0xA83C0；符号 479；零内核
// 注册——物品属性计算 API）。
type ItemProCompute struct {
	BaseModule

	// computes 计算流水（测试面）。
	computes []string
}

// Name 模块名。
func (m *ItemProCompute) Name() string { return "ItemProCompute" }

// Init 注册期（官方零注册——API 模块）。
func (m *ItemProCompute) Init(k *Kernel) error { return nil }

// ComputeItemProps 物品属性包计算：PropPack/PropBuilt/PropRanBuilt 串
// （equipment.xml 方言「属性名,值;…」）→ 属性映射（穿戴聚合用）。
func (m *ItemProCompute) ComputeItemProps(k *Kernel, item PersistID) map[string]int32 {
	out := map[string]int32{}
	o := k.GetGameObj(item)
	if o == nil {
		return out
	}
	for _, pack := range []string{"PropPack", "PropBuilt", "PropRanBuilt"} {
		for _, seg := range strings.Split(o.GetString(pack), ";") {
			pair := strings.Split(strings.TrimSpace(seg), ",")
			if len(pair) == 2 && pair[0] != "" {
				out[pair[0]] += atoi32(pair[1])
			}
		}
	}
	m.computes = append(m.computes, fmt.Sprintf("%s:%d", o.GetString("ConfigID"), len(out)))
	return out
}

// Computes 流水（测试面）。
func (m *ItemProCompute) Computes() []string { return m.computes }

// ---- ItemNotify（Init 序 86）----

// ItemNotify 复刻官方 ItemNotify（Init @0x10A8B0；符号 22；零内核注册——
// 物品入/出包通知客户端 API）。
type ItemNotify struct {
	BaseModule

	// NotifyOut 通知出口（sync 层转物品面板方言；默认流水）。
	NotifyOut func(player PersistID, itemID string, count int32, add bool)
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *ItemNotify) Name() string { return "ItemNotify" }

// Init 注册期（官方零注册——API 模块）。
func (m *ItemNotify) Init(k *Kernel) error {
	if m.NotifyOut == nil {
		m.NotifyOut = func(player PersistID, itemID string, count int32, add bool) {
			m.events = append(m.events, fmt.Sprintf("%s:%s×%d", map[bool]string{true: "add", false: "del"}[add], itemID, count))
		}
	}
	return nil
}

// NotifyAdd 入包通知。
func (m *ItemNotify) NotifyAdd(player PersistID, itemID string, count int32) {
	m.NotifyOut(player, itemID, count, true)
}

// NotifyRemove 出包通知。
func (m *ItemNotify) NotifyRemove(player PersistID, itemID string, count int32) {
	m.NotifyOut(player, itemID, count, false)
}

// Events 流水（测试面）。
func (m *ItemNotify) Events() []string { return m.events }

// ---- ItemControlModule（Init 序 197）----

// ItemControlModule 复刻官方 ItemControlModule（Init @0x7C6AF0；符号 9）——
// 物品管控（公共消息面：回收/禁用物品）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("scene","OnPublicMessage")
type ItemControlModule struct {
	BaseModule

	// banned 禁用物品清单（configID → true；OnPublicMessage 收编面）。
	banned map[string]bool
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *ItemControlModule) Name() string { return "ItemControlModule" }

// Init 注册期（对齐官方 0x7C6AF0）。
func (m *ItemControlModule) Init(k *Kernel) error {
	m.banned = make(map[string]bool)
	k.Bus().OnEvent("scene", "OnPublicMessage", m.onPublicMessage)
	return nil
}

// onPublicMessage 官方：公共消息 → 物品管控指令
// （args: "ban:<id>"/"unban:<id>"/"recall:<id>"——Go 收编方言）。
func (m *ItemControlModule) onPublicMessage(k *Kernel, self, from PersistID, args VarList) {
	msg := args.String(0)
	switch {
	case strings.HasPrefix(msg, "ban:"):
		m.banned[strings.TrimPrefix(msg, "ban:")] = true
		m.events = append(m.events, msg)
	case strings.HasPrefix(msg, "unban:"):
		delete(m.banned, strings.TrimPrefix(msg, "unban:"))
		m.events = append(m.events, msg)
	case strings.HasPrefix(msg, "recall:"):
		m.events = append(m.events, msg)
	}
}

// Banned 禁用门。
func (m *ItemControlModule) Banned(itemID string) bool { return m.banned[itemID] }

// Events 流水（测试面）。
func (m *ItemControlModule) Events() []string { return m.events }

// ---- ModifyPackModule（Init 序 230）----

// ModifyPackModule 复刻官方 ModifyPackModule（Init @0x76F2A0；符号 470）——
// 修正包管理（装备/铭文/buff 穿卸 → 属性修正重算联动）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover") + ("Equipment"/"EnergyShield",
//	    "OnLoad"/"OnRecover")
//	AddEventCallback ("EquipBox"/"BufferContainer"/"InscriptionBox",
//	    "OnAfterAdd"/"OnRemove")
//	DeclareCritical ("Equipment","Hardiness",C_OnHardinessChanged)
type ModifyPackModule struct {
	BaseModule

	// events 流水（测试面："equip"/"unequip"）。
	events []string
}

// Name 模块名。
func (m *ModifyPackModule) Name() string { return "ModifyPackModule" }

// Init 注册期（对齐官方 0x76F2A0）。
func (m *ModifyPackModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onPlayerRecover)
	b.OnEvent("Equipment", "OnLoad", m.onEquipLoad)
	b.OnEvent("Equipment", "OnRecover", m.onEquipLoad)
	b.OnEvent("EnergyShield", "OnLoad", m.onEquipLoad)
	b.OnEvent("EnergyShield", "OnRecover", m.onEquipLoad)
	b.OnEvent("EquipBox", "OnAfterAdd", m.onEquipAdd)
	b.OnEvent("EquipBox", "OnRemove", m.onEquipRemove)
	b.OnEvent("BufferContainer", "OnAfterAdd", m.onEquipAdd)
	b.OnEvent("BufferContainer", "OnRemove", m.onEquipRemove)
	b.OnEvent("InscriptionBox", "OnAfterAdd", m.onEquipAdd)
	b.OnEvent("InscriptionBox", "OnRemove", m.onEquipRemove)
	b.Critical("Equipment", "Hardiness", m.onHardinessChanged)
	return nil
}

// recalc 修正重算联动（ObjPropRefreshModule::Recalc——PropModifyRec 聚合）。
func (m *ModifyPackModule) recalc(k *Kernel, owner PersistID) {
	if or, _ := k.GetLogicModule("ObjPropRefreshModule").(*ObjPropRefreshModule); or != nil {
		or.Recalc(k, owner)
	}
}

// onEquipAdd 官方 @0x76FF00 族：穿上（装备/buff/铭文入容器）→ 修正重算。
func (m *ModifyPackModule) onEquipAdd(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "equip")
	m.recalc(k, self)
}

// onEquipRemove 官方 @0x76FFA0 族：卸下 → 修正重算。
func (m *ModifyPackModule) onEquipRemove(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "unequip")
	m.recalc(k, self)
}

func (m *ModifyPackModule) onPlayerRecover(k *Kernel, self, from PersistID, args VarList) {
	m.recalc(k, self)
}

func (m *ModifyPackModule) onEquipLoad(k *Kernel, self, from PersistID, args VarList) {}

// onHardinessChanged 官方临界：耐久变更 → 修正重算（耐久归零属性失效面）。
func (m *ModifyPackModule) onHardinessChanged(k *Kernel, self PersistID, prop string, old Var) {
	if o := k.GetGameObj(self); o != nil && o.GetInt("Hardiness") == 0 {
		m.events = append(m.events, "broken")
	}
}

// Events 流水（测试面）。
func (m *ModifyPackModule) Events() []string { return m.events }

// ---- AddContainerModule（Init 序 237）----

// AddContainerModule 复刻官方 AddContainerModule（Init @0x716720；符号 75）——
// 扩展容器（AddContainer/Bag 类族：背包扩容格）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddClassCallback ("AddContainer"/"Bag","OnCreateClass")
//	AddEventCallback ("AddContainer","OnAfterAdd"/"OnRemove"/"OnNoAdd"/
//	    "OnNoRemove") + ("Bag","OnEntry"/"OnRecover")
//	DeclareCritical ("Bag","NotValid",C_OnNotValidChanged)
type AddContainerModule struct {
	BaseModule

	// caps 扩容容量（玩家 → 已开格数；baglevel_cap_rec 官方面）。
	caps map[PersistID]int32
	// events 流水（测试面："expand:<n>"/"full"）。
	events []string
}

// Name 模块名。
func (m *AddContainerModule) Name() string { return "AddContainerModule" }

// Init 注册期（对齐官方 0x716720）。
func (m *AddContainerModule) Init(k *Kernel) error {
	m.caps = make(map[PersistID]int32)
	b := k.Bus()
	b.OnClassEvent("AddContainer", "OnCreateClass", func(k *Kernel, class string) {})
	b.OnEvent("AddContainer", "OnAfterAdd", m.onAfterAdd)
	b.OnEvent("AddContainer", "OnRemove", m.onRemove)
	b.OnEvent("AddContainer", "OnNoAdd", m.onNoAdd)
	b.OnEvent("AddContainer", "OnNoRemove", m.onNoRemove)
	b.OnClassEvent("Bag", "OnCreateClass", func(k *Kernel, class string) {})
	b.OnEvent("Bag", "OnEntry", m.onBagEntry)
	b.OnEvent("Bag", "OnRecover", m.onBagEntry)
	b.Critical("Bag", "NotValid", m.onNotValidChanged)
	return nil
}

// ExpandBag 背包扩容（baglevel_cap_rec 方言：等级 → 容量；Go 面 +N 格）。
func (m *AddContainerModule) ExpandBag(k *Kernel, player PersistID, slots int32) bool {
	if slots <= 0 {
		return false
	}
	m.caps[player] += slots
	m.events = append(m.events, fmt.Sprintf("expand:%d", slots))
	return true
}

// Capacity 已开格数。
func (m *AddContainerModule) Capacity(player PersistID) int32 { return m.caps[player] }

func (m *AddContainerModule) onAfterAdd(k *Kernel, self, from PersistID, args VarList) {}
func (m *AddContainerModule) onRemove(k *Kernel, self, from PersistID, args VarList)   {}

// onNoAdd 官方 @0x716D60：容量满拒（流水）。
func (m *AddContainerModule) onNoAdd(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "full")
}

func (m *AddContainerModule) onNoRemove(k *Kernel, self, from PersistID, args VarList) {}

func (m *AddContainerModule) onBagEntry(k *Kernel, self, from PersistID, args VarList) {}

// onNotValidChanged 官方临界：扩容格有效期变更。
func (m *AddContainerModule) onNotValidChanged(k *Kernel, self PersistID, prop string, old Var) {}

// Events 流水（测试面）。
func (m *AddContainerModule) Events() []string { return m.events }

// ---- ChargeModule（Init 序 70）----

// ChargeModule 复刻官方 ChargeModule（Init @0x431690；符号 715，11 心跳）——
// 充值商城（异步充值链）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("scene","OnCreate"/"OnItemInfo"/"OnBuyItem2"/
//	    "OnGetRoleInfo"/"OnChargeMessage") + ("player","OnReady")
//	AddIntCustomHook ("player",196=0xC4)
//	DeclareHeartBeat ×5（HB_AsyWait/HB_DelayRequestChargeInfo/
//	    HB_CheckUpdateItem/HB_CheckChargeBoxItem/HB_CheckChargeHistory）
type ChargeModule struct {
	BaseModule

	// BuyOut 购买到货出口（接 ToolBox）。
	BuyOut func(player PersistID, itemID string, count int32)
	// events 流水（测试面："buy:<id>×<n>"）。
	events []string
}

// Name 模块名。
func (m *ChargeModule) Name() string { return "ChargeModule" }

// Init 注册期（对齐官方 0x431690）。
func (m *ChargeModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("scene", "OnCreate", m.onSceneCreate)
	b.OnEvent("scene", "OnItemInfo", m.onItemInfo)
	b.OnEvent("scene", "OnBuyItem2", m.onBuyItem2)
	b.OnEvent("scene", "OnGetRoleInfo", m.onGetRoleInfo)
	b.OnEvent("scene", "OnChargeMessage", m.onChargeMessage)
	b.OnEvent("player", "OnReady", m.onPlayerReady)
	b.OnCommand("player", 196, m.onCustomCharge)
	b.HeartBeat("ChargeModule::HB_AsyWait", m.hbAsyWait)
	b.HeartBeat("ChargeModule::HB_DelayRequestChargeInfo", m.hbDelayRequestChargeInfo)
	b.HeartBeat("ChargeModule::HB_CheckUpdateItem", m.hbCheckUpdateItem)
	b.HeartBeat("ChargeModule::HB_CheckChargeBoxItem", m.hbCheckChargeBoxItem)
	b.HeartBeat("ChargeModule::HB_CheckChargeHistory", m.hbCheckChargeHistory)
	if m.BuyOut == nil {
		m.BuyOut = func(player PersistID, itemID string, count int32) {}
	}
	return nil
}

// onBuyItem2 官方 @0x4332C0：商城购买（args: 物品, 数量）→ 到货出口。
func (m *ChargeModule) onBuyItem2(k *Kernel, self, from PersistID, args VarList) {
	itemID, count := args.String(0), args.Int(1)
	if count <= 0 {
		return
	}
	m.BuyOut(from, itemID, count)
	m.events = append(m.events, fmt.Sprintf("buy:%s×%d", itemID, count))
}

// onCustomCharge 官方 @0x432EF0（cust 196=0xC4）：充值消息
// （args: 子命令——查询/领取面）。
func (m *ChargeModule) onCustomCharge(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, fmt.Sprintf("cust:%d", args.Int(0)))
}

func (m *ChargeModule) onSceneCreate(k *Kernel, self, from PersistID, args VarList) {}
func (m *ChargeModule) onItemInfo(k *Kernel, self, from PersistID, args VarList)    {}
func (m *ChargeModule) onGetRoleInfo(k *Kernel, self, from PersistID, args VarList) {}
func (m *ChargeModule) onChargeMessage(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "charge-msg")
}
func (m *ChargeModule) onPlayerReady(k *Kernel, self, from PersistID, args VarList) {}

// 五心跳：异步充值等待/延迟请求/更新/盒子/历史巡检（官方链——Go 面 trace 级）。
func (m *ChargeModule) hbAsyWait(k *Kernel, self PersistID, beats int32)                {}
func (m *ChargeModule) hbDelayRequestChargeInfo(k *Kernel, self PersistID, beats int32) {}
func (m *ChargeModule) hbCheckUpdateItem(k *Kernel, self PersistID, beats int32)        {}
func (m *ChargeModule) hbCheckChargeBoxItem(k *Kernel, self PersistID, beats int32)     {}
func (m *ChargeModule) hbCheckChargeHistory(k *Kernel, self PersistID, beats int32)     {}

// Events 流水（测试面）。
func (m *ChargeModule) Events() []string { return m.events }

// ---- 共用取用面 ----

// ownerOf 物品属主（官方 GetParent；Go 面父链）。
func ownerOf(k *Kernel, id PersistID) PersistID {
	if o := k.GetGameObj(id); o != nil {
		return o.Parent()
	}
	return NullPersistID
}

// isItemClass 物品类族判断（item 基类链）。
func isItemClass(k *Kernel, id PersistID) bool {
	o := k.GetGameObj(id)
	if o == nil {
		return false
	}
	for c := o.ClassName(); c != ""; {
		if c == "item" {
			return true
		}
		cd := k.Class(c)
		if cd == nil {
			return false
		}
		c = cd.Parent
	}
	return false
}
