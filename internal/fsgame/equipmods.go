package fsgame

// 十三期：装备与物品玩法余量十九模块（docs/official/FsGame全解与Go复刻.md
// §8 养成·装备与物品玩法表余量）。官方源码树 itembasemodule\ +
// playerbasemodule\ 等。
//
// 真档 schema：shortcut_rec（序号/参数1/参数2/类型——快捷栏 4 列）、
// state.xml FashionState（时装状态）、equipment.xml ResolvePack（分解包）/
// PropPack 族、tool_item.xml FuncPack/WealthPack/FormulaPack/
// FragmentComposePack/TriggerTaskID、enchase.xml JewelLevel/CantCompound、
// container 族 depot.xml。

import (
	"fmt"
	"strings"
)

// ---- FashionModule（Init 序 83）----

// FashionModule 复刻官方 FashionModule（Init @0x10A8B0；符号 223）——时装。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover"/"OnReady")
//	AddEventCallback ("FashionBox","OnCreate"/"OnAfterAdd"/"OnRemove"/
//	    "OnNoAdd"/"OnNoRemove")
//	AddIntCustomHook ("player",69=0x45)
type FashionModule struct {
	BaseModule

	// events 流水（测试面："wear:<id>"/"strip:<id>"）。
	events []string
}

// Name 模块名。
func (m *FashionModule) Name() string { return "FashionModule" }

// Init 注册期（对齐官方 0x10A8B0）。
func (m *FashionModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("FashionBox", "OnCreate", m.onFashionCreate)
	b.OnEvent("FashionBox", "OnAfterAdd", m.onAfterAdd)
	b.OnEvent("FashionBox", "OnRemove", m.onRemove)
	b.OnEvent("FashionBox", "OnNoAdd", m.onNoAdd)
	b.OnEvent("FashionBox", "OnNoRemove", m.onNoRemove)
	b.OnCommand("player", 69, m.onCustomFashion)
	return nil
}

// onAfterAdd 官方 @0x10BB30：穿时装 → FashionState 联动（外观域）+ 流水。
func (m *FashionModule) onAfterAdd(k *Kernel, self, from PersistID, args VarList) {
	if len(args) > 0 {
		if it, ok := args[0].(PersistID); ok {
			if o := k.GetGameObj(it); o != nil {
				m.events = append(m.events, fmt.Sprintf("wear:%s", o.GetString("ConfigID")))
			}
		}
	}
	if o := k.GetGameObj(self); o != nil {
		if _, ok := o.Class().Prop("FashionState"); ok {
			_ = k.SetInt(self, "FashionState", 1)
		}
	}
}

// onRemove 官方 @0x10BC30：脱时装 → FashionState 复位。
func (m *FashionModule) onRemove(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "strip")
	if o := k.GetGameObj(self); o != nil {
		if _, ok := o.Class().Prop("FashionState"); ok {
			_ = k.SetInt(self, "FashionState", 0)
		}
	}
}

func (m *FashionModule) onFashionCreate(k *Kernel, self, from PersistID, args VarList) {}
func (m *FashionModule) onNoAdd(k *Kernel, self, from PersistID, args VarList)         {}
func (m *FashionModule) onNoRemove(k *Kernel, self, from PersistID, args VarList)      {}
func (m *FashionModule) onRecover(k *Kernel, self, from PersistID, args VarList)       {}
func (m *FashionModule) onReady(k *Kernel, self, from PersistID, args VarList)         {}

// onCustomFashion 官方（cust 69=0x45）：时装 UI 消息。
func (m *FashionModule) onCustomFashion(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, fmt.Sprintf("cust:%d", args.Int(0)))
}

// Events 流水（测试面）。
func (m *FashionModule) Events() []string { return m.events }

// ---- ItemLaveModule（Init 序 84）----

// ItemLaveModule 复刻官方 ItemLaveModule（Init @0x80FF0；符号 369）——物品
// 分解（NPC 对话面）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCustomHook ("player",738=0x2E2)
//	AddEventCallback ("CommonNpc","OnSelect") + ("player","OnLeaveScene")
//	AddIntCommandHook ("player",22 菜单)
type ItemLaveModule struct {
	BaseModule

	// LaveOut 分解产出出口（ResolvePack → 货币/物品；接 Capital/ToolBox）。
	LaveOut func(player PersistID, itemID string, value int32)
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *ItemLaveModule) Name() string { return "ItemLaveModule" }

// Init 注册期（对齐官方 0x80FF0）。
func (m *ItemLaveModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnCommand("player", 738, m.onCustomLave)
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.OnEvent("player", "OnLeaveScene", m.onLeaveScene)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	if m.LaveOut == nil {
		m.LaveOut = func(player PersistID, itemID string, value int32) {}
	}
	return nil
}

// LaveItem 分解：ResolvePack（分解包值）→ 产出出口（args 物品对象）。
func (m *ItemLaveModule) LaveItem(k *Kernel, player, item PersistID) bool {
	o := k.GetGameObj(item)
	if o == nil {
		return false
	}
	value := o.GetInt("ResolvePack")
	if value <= 0 {
		return false
	}
	m.LaveOut(player, o.GetString("ConfigID"), value)
	m.events = append(m.events, fmt.Sprintf("lave:%s→%d", o.GetString("ConfigID"), value))
	return true
}

// onCustomLave 官方（cust 738=0x2E2）：请求分解（args: 物品）。
func (m *ItemLaveModule) onCustomLave(k *Kernel, self, from PersistID, args VarList) {
	if len(args) > 0 {
		if it, ok := args[0].(PersistID); ok {
			m.LaveItem(k, self, it)
		}
	}
}

func (m *ItemLaveModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "npc")
}

func (m *ItemLaveModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {}
func (m *ItemLaveModule) onLeaveScene(k *Kernel, self, from PersistID, args VarList) {}

// Events 流水（测试面）。
func (m *ItemLaveModule) Events() []string { return m.events }

// ---- ToolItemModule（Init 序 99）----

// ToolItemModule 复刻官方 ToolItemModule（Init @0x8E7C0；符号 918）——使用
// 物品主链（功能包分派）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCommandHook ("ToolItem",10=0xA) / ("player",613/603/602/1012)
//	AddEventCallback ("player","OnReady") + ("UsingItemSkill","OnCreate")
type ToolItemModule struct {
	BaseModule

	// EffectOut 使用效果出口（FuncPack 分派产物）。
	EffectOut func(player PersistID, itemID string, funcPack int32)
	// events 流水（测试面："use:<id> pack=<n>"）。
	events []string
}

// Name 模块名。
func (m *ToolItemModule) Name() string { return "ToolItemModule" }

// Init 注册期（对齐官方 0x8E7C0）。
func (m *ToolItemModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnIntCommand("ToolItem", 10, m.onUseItem)
	b.OnIntCommand("player", 613, m.onInterrupt)
	b.OnIntCommand("player", 603, m.onSkillFinish)
	b.OnIntCommand("player", 602, m.onSkillBreaked)
	b.OnIntCommand("player", 1012, m.onCollect)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("UsingItemSkill", "OnCreate", m.onUsingSkillCreate)
	if m.EffectOut == nil {
		m.EffectOut = func(player PersistID, itemID string, funcPack int32) {}
	}
	return nil
}

// UseItem 使用物品：FuncPack（公用功能包）/ TriggerTaskID（触发任务）/
// WealthPack（宝箱）三面分派。
func (m *ToolItemModule) UseItem(k *Kernel, player, item PersistID) bool {
	o := k.GetGameObj(item)
	if o == nil {
		return false
	}
	pack := o.GetInt("FuncPack")
	if pack != 0 {
		m.EffectOut(player, o.GetString("ConfigID"), pack)
	}
	// 触发任务面（TriggerTaskID → QuestModule 接取联动）。
	if tid := o.GetInt("TriggerTaskID"); tid != 0 {
		if qm, _ := k.GetLogicModule("QuestModule").(*QuestModule); qm != nil {
			qm.AcceptQuest(k, player, tid)
		}
	}
	m.events = append(m.events, fmt.Sprintf("use:%s pack=%d", o.GetString("ConfigID"), pack))
	return true
}

// onUseItem 官方 @0x8EB70（ToolItem cmd 10=0xA）：使用通知
// （args[0]=物品，from=使用者）。
func (m *ToolItemModule) onUseItem(k *Kernel, self, from PersistID, args VarList) {
	if len(args) > 0 {
		if it, ok := args[0].(PersistID); ok {
			m.UseItem(k, from, it)
		}
	}
}

func (m *ToolItemModule) onInterrupt(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "interrupt")
}

func (m *ToolItemModule) onSkillFinish(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "finish")
}

func (m *ToolItemModule) onSkillBreaked(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "breaked")
}

func (m *ToolItemModule) onCollect(k *Kernel, self, from PersistID, args VarList) {}

func (m *ToolItemModule) onReady(k *Kernel, self, from PersistID, args VarList) {}

// onUsingSkillCreate 官方 @0x8CF90：使用物品临时技能物化（锁定面）。
func (m *ToolItemModule) onUsingSkillCreate(k *Kernel, self, from PersistID, args VarList) {}

// Events 流水（测试面）。
func (m *ToolItemModule) Events() []string { return m.events }

// ---- CompoundItem（Init 序 94）----

// CompoundItem 复刻官方 CompoundItem（Init @0x9DC80；符号 581；零内核注册——
// 合成 API：FormulaPack/FragmentComposePack 方言）。
type CompoundItem struct {
	BaseModule

	// CompoundOut 合成产物出口（接 ToolBox）。
	CompoundOut func(player PersistID, itemID string, count int32)
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *CompoundItem) Name() string { return "CompoundItem" }

// Init 注册期（官方零注册——API 模块）。
func (m *CompoundItem) Init(k *Kernel) error {
	if m.CompoundOut == nil {
		m.CompoundOut = func(player PersistID, itemID string, count int32) {}
	}
	return nil
}

// Compound 碎片合成：FragmentComposePack 是 int 包编号（产物/需求经配置
// 面由调用方解析），碎片足数则产出。
func (m *CompoundItem) Compound(k *Kernel, player, item PersistID, fragments int32, productID string, need int32) bool {
	o := k.GetGameObj(item)
	if o == nil || fragments <= 0 || o.GetInt("FragmentComposePack") == 0 {
		return false
	}
	if fragments < need || need <= 0 {
		return false
	}
	m.CompoundOut(player, productID, 1)
	m.events = append(m.events, fmt.Sprintf("compound:%s×1 (碎片 %d/%d)", productID, fragments, need))
	return true
}

// Events 流水（测试面）。
func (m *CompoundItem) Events() []string { return m.events }

// ---- Enchase（Init 序 198）----

// Enchase 复刻官方 Enchase（Init @0x7616E0；符号 142；零内核注册——镶嵌
// API：enchase.xml JewelLevel/CantCompound）。
type Enchase struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *Enchase) Name() string { return "Enchase" }

// Init 注册期（官方零注册——API 模块）。
func (m *Enchase) Init(k *Kernel) error { return nil }

// EnchaseItem 镶嵌：宝石（Enchase 类）→ 装备 PropPack 追加。宝石属性串
// 官方在配置表（enchase.xml struct 无 PropPack 列——JewelLevel 仅档位），
// 由调用方按 configID 查表后传入 gemProps。
func (m *Enchase) EnchaseItem(k *Kernel, equip, gem PersistID, gemProps string) bool {
	eo, go2 := k.GetGameObj(equip), k.GetGameObj(gem)
	if eo == nil || go2 == nil || gemProps == "" {
		return false
	}
	cur := eo.GetString("PropPack")
	if cur != "" && !strings.HasSuffix(cur, ";") {
		cur += ";"
	}
	_ = k.SetString(equip, "PropPack", cur+gemProps)
	m.events = append(m.events, fmt.Sprintf("enchase:%s←%s(lv%d)",
		eo.GetString("ConfigID"), go2.GetString("ConfigID"), go2.GetInt("JewelLevel")))
	return true
}

// Events 流水（测试面）。
func (m *Enchase) Events() []string { return m.events }

// ---- DropCapitalItemModule（Init 序 98）----

// DropCapitalItemModule 复刻官方 DropCapitalItemModule（Init @0x7A1EC0；
// 符号 165）——掉落金钱。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddClassCallback ("DropMoneyItem","OnCreateClass")
//	AddIntCommandHook ("player",851=0x353 拾取)
type DropCapitalItemModule struct {
	BaseModule

	// MoneyOut 拾钱出口（接 Capital）。
	MoneyOut func(player PersistID, amount int32)
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *DropCapitalItemModule) Name() string { return "DropCapitalItemModule" }

// Init 注册期（对齐官方 0x7A1EC0）。
func (m *DropCapitalItemModule) Init(k *Kernel) error {
	k.Bus().OnClassEvent("DropMoneyItem", "OnCreateClass", func(k *Kernel, class string) {
		m.events = append(m.events, "drop-money")
	})
	k.Bus().OnIntCommand("player", 851, m.onPickMoney)
	if m.MoneyOut == nil {
		m.MoneyOut = func(player PersistID, amount int32) {}
	}
	return nil
}

// onPickMoney 官方 @0x7A880（cmd 851=0x353）：拾取掉落金钱
// （args: DropMoneyItem 对象 → Amount 面值）。
func (m *DropCapitalItemModule) onPickMoney(k *Kernel, self, from PersistID, args VarList) {
	if len(args) == 0 {
		return
	}
	drop, _ := args[0].(PersistID)
	o := k.GetGameObj(drop)
	if o == nil {
		return
	}
	amount := o.GetInt("Amount")
	if amount <= 0 {
		return
	}
	m.MoneyOut(self, amount)
	m.events = append(m.events, fmt.Sprintf("pick:%d", amount))
}

// Events 流水（测试面）。
func (m *DropCapitalItemModule) Events() []string { return m.events }

// ---- ArmorChangeModule（Init 序 126）----

// ArmorChangeModule 复刻官方 ArmorChangeModule（Init @0x2300C0；符号 790）——
// 装甲换装（ArmorItem/ArmorBox 容器族）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddClassCallback ("ArmorItem","OnCreateClass")
//	AddEventCallback ("player","OnRecover"/"OnReady")
//	AddEventCallback ("ArmorBox","OnCreate"/"OnAfterAdd"/"OnRemove"/"OnNoAdd"/
//	    "OnNoRemove") + ("EquipBox","OnRemove")
type ArmorChangeModule struct {
	BaseModule

	// events 流水（测试面："armor-on"/"armor-off"）。
	events []string
}

// Name 模块名。
func (m *ArmorChangeModule) Name() string { return "ArmorChangeModule" }

// Init 注册期（对齐官方 0x2300C0）。
func (m *ArmorChangeModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnClassEvent("ArmorItem", "OnCreateClass", func(k *Kernel, class string) {})
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("ArmorBox", "OnCreate", m.onArmorCreate)
	b.OnEvent("ArmorBox", "OnAfterAdd", m.onArmorAdd)
	b.OnEvent("ArmorBox", "OnRemove", m.onArmorRemove)
	b.OnEvent("ArmorBox", "OnNoAdd", m.onNoAdd)
	b.OnEvent("ArmorBox", "OnNoRemove", m.onNoRemove)
	b.OnEvent("EquipBox", "OnRemove", m.onEquipRemove)
	return nil
}

func (m *ArmorChangeModule) onArmorAdd(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "armor-on")
}

func (m *ArmorChangeModule) onArmorRemove(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "armor-off")
}

// onEquipRemove 官方 @0x234110：装备栏卸下 → 装甲联动重算。
func (m *ArmorChangeModule) onEquipRemove(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "equip-off")
}

func (m *ArmorChangeModule) onArmorCreate(k *Kernel, self, from PersistID, args VarList) {}
func (m *ArmorChangeModule) onNoAdd(k *Kernel, self, from PersistID, args VarList)       {}
func (m *ArmorChangeModule) onNoRemove(k *Kernel, self, from PersistID, args VarList)    {}
func (m *ArmorChangeModule) onRecover(k *Kernel, self, from PersistID, args VarList)     {}
func (m *ArmorChangeModule) onReady(k *Kernel, self, from PersistID, args VarList)       {}

// Events 流水（测试面）。
func (m *ArmorChangeModule) Events() []string { return m.events }

// ---- TransformJobModule（Init 序 136）----

// TransformJobModule 复刻官方 TransformJobModule（Init @0x1A22D0；符号 289）——
// 转职（黑金职业转换）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover"/"OnReady") +
//	    ("CommonNpc","OnSelect",@0x1A3560 转职 NPC 对话)
//	AddIntCustomHook ("player",1023=0x3FF)
//	AddIntCommandHook ("player",22 菜单)
//	DeclareLuaExt nx_trans_job
type TransformJobModule struct {
	BaseModule

	// events 流水（测试面："trans:<job>"）。
	events []string
}

// Name 模块名。
func (m *TransformJobModule) Name() string { return "TransformJobModule" }

// Init 注册期（对齐官方 0x1A22D0）。
func (m *TransformJobModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.OnCommand("player", 1023, m.onCustomTrans)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	return nil
}

// NxTransJob nx_trans_job 的 Go 等价：转职——Job 变更 + 等级/技能面板
// 重置面（TransPre 复位——equipment.xml 转职前旗）。
func (m *TransformJobModule) NxTransJob(k *Kernel, player PersistID, job int32) bool {
	o := k.GetGameObj(player)
	if o == nil || job < 0 {
		return false
	}
	if _, ok := o.Class().Prop("Job"); !ok {
		return false
	}
	old := o.GetInt("Job")
	if old == job {
		return false
	}
	_ = k.SetInt(player, "Job", job)
	m.events = append(m.events, fmt.Sprintf("trans:%d→%d", old, job))
	k.Tracef("TransformJobModule: player=%v job %d→%d", player.ID(), old, job)
	return true
}

// onCustomTrans 官方 @0x1A32C0（cust 1023=0x3FF）：转职请求（args: 目标职
// 业——菜单上下文经转职 NPC 建立）。
func (m *TransformJobModule) onCustomTrans(k *Kernel, self, from PersistID, args VarList) {
	m.NxTransJob(k, self, args.Int(0))
}

func (m *TransformJobModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "npc")
}

func (m *TransformJobModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {}
func (m *TransformJobModule) onRecover(k *Kernel, self, from PersistID, args VarList)    {}
func (m *TransformJobModule) onReady(k *Kernel, self, from PersistID, args VarList)      {}

// Events 流水（测试面）。
func (m *TransformJobModule) Events() []string { return m.events }

// ---- DepotModule（Init 序 145）----

// DepotModule 复刻官方 DepotModule（Init @0x16E0D0；符号 889）——仓库。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("player","OnRecover"/"OnReady"/"OnDoSelect"/"OnLeaveScene")
//	AddIntCustomHook ("player",860=0x35C / 220=0xDC / 223=0xDF / 910=0x38E)
type DepotModule struct {
	BaseModule

	// depot 仓库存量（玩家 → 物品ID → 数量）。
	depot map[PersistID]map[string]int32
	// events 流水（测试面："depot:<id>×<n>"/"withdraw:<id>×<n>"）。
	events []string
}

// Name 模块名。
func (m *DepotModule) Name() string { return "DepotModule" }

// Init 注册期（对齐官方 0x16E0D0）。
func (m *DepotModule) Init(k *Kernel) error {
	m.depot = make(map[PersistID]map[string]int32)
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnDoSelect", m.onDoSelect)
	b.OnEvent("player", "OnLeaveScene", m.onLeaveScene)
	b.OnCommand("player", 860, m.onDepotItem)
	b.OnCommand("player", 220, m.onDepotItem)
	b.OnCommand("player", 223, m.onWithdrawItem)
	b.OnCommand("player", 910, m.onQueryDepot)
	return nil
}

// Depository 存仓（CantDepot/绑定物拒——ItemLimited 门）。
func (m *DepotModule) Depository(k *Kernel, player, item PersistID) bool {
	if il, _ := k.GetLogicModule("ItemLimited").(*ItemLimited); il != nil && !il.CanDepot(k, item) {
		return false
	}
	o := k.GetGameObj(item)
	if o == nil {
		return false
	}
	if m.depot[player] == nil {
		m.depot[player] = make(map[string]int32)
	}
	m.depot[player][o.GetString("ConfigID")] += o.GetInt("Amount")
	m.events = append(m.events, fmt.Sprintf("depot:%s×%d", o.GetString("ConfigID"), o.GetInt("Amount")))
	return true
}

// Withdraw 取仓（存量扣减）。
func (m *DepotModule) Withdraw(k *Kernel, player PersistID, itemID string, count int32) bool {
	d := m.depot[player]
	if d == nil || d[itemID] < count {
		return false
	}
	d[itemID] -= count
	if d[itemID] == 0 {
		delete(d, itemID)
	}
	m.events = append(m.events, fmt.Sprintf("withdraw:%s×%d", itemID, count))
	return true
}

// Stock 存量查询。
func (m *DepotModule) Stock(player PersistID, itemID string) int32 { return m.depot[player][itemID] }

// onDepotItem 官方（cust 860/220）：存仓请求（args: 物品[, 数量]）。
func (m *DepotModule) onDepotItem(k *Kernel, self, from PersistID, args VarList) {
	if len(args) > 0 {
		if it, ok := args[0].(PersistID); ok {
			m.Depository(k, self, it)
		}
	}
}

// onWithdrawItem 官方（cust 223=0xDF）：取仓请求（args: 物品ID, 数量）。
func (m *DepotModule) onWithdrawItem(k *Kernel, self, from PersistID, args VarList) {
	m.Withdraw(k, self, args.String(0), args.Int(1))
}

// onQueryDepot 官方（cust 910=0x38E）：仓库清单查询（下发面）。
func (m *DepotModule) onQueryDepot(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "query")
}

func (m *DepotModule) onDoSelect(k *Kernel, self, from PersistID, args VarList)   {}
func (m *DepotModule) onLeaveScene(k *Kernel, self, from PersistID, args VarList) {}
func (m *DepotModule) onRecover(k *Kernel, self, from PersistID, args VarList)    {}
func (m *DepotModule) onReady(k *Kernel, self, from PersistID, args VarList)      {}

// Events 流水（测试面）。
func (m *DepotModule) Events() []string { return m.events }

// ---- ShortCutModule（Init 序 146）----

// ShortCutModule 复刻官方 ShortCutModule（Init @0x166100；符号 473）——快捷栏
// （shortcut_rec：序号/参数1/参数2/类型）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCustomHook ("player",199/200/201/208/163/209)
//	AddEventCallback ("SkillContainer","OnAfterAdd"/"OnRemove")
type ShortCutModule struct {
	BaseModule

	// events 流水（测试面："bind:<idx>"）。
	events []string
}

// Name 模块名。
func (m *ShortCutModule) Name() string { return "ShortCutModule" }

// Init 注册期（对齐官方 0x166100）。
func (m *ShortCutModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnCommand("player", 199, m.onBind)
	b.OnCommand("player", 200, m.onUnbind)
	b.OnCommand("player", 201, m.onSwap)
	b.OnCommand("player", 208, m.onBind)
	b.OnCommand("player", 163, m.onClear)
	b.OnCommand("player", 209, m.onQuery)
	b.OnEvent("SkillContainer", "OnAfterAdd", m.onSkillAdd)
	b.OnEvent("SkillContainer", "OnRemove", m.onSkillRemove)
	return nil
}

// BindShortcut 绑定（shortcut_rec upsert：序号行 → 参数/类型）。
func (m *ShortCutModule) BindShortcut(k *Kernel, player PersistID, index int32, arg1, arg2 string, typ int32) bool {
	o := k.GetGameObj(player)
	r := o.Record("shortcut_rec")
	if r == nil {
		return false
	}
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).Int(0) == index {
			r.ModifyRow(i, VarList{index, arg1, arg2, typ})
			m.events = append(m.events, fmt.Sprintf("bind:%d", index))
			return true
		}
	}
	if r.Rows() < r.Def().MaxRows {
		r.AppendRow(VarList{index, arg1, arg2, typ})
		m.events = append(m.events, fmt.Sprintf("bind:%d", index))
		return true
	}
	return false
}

// UnbindShortcut 解绑（类型>0 锁定项不可解——官方「不能从快捷栏删除」列）。
func (m *ShortCutModule) UnbindShortcut(k *Kernel, player PersistID, index int32) bool {
	o := k.GetGameObj(player)
	r := o.Record("shortcut_rec")
	if r == nil {
		return false
	}
	for i := 0; i < r.Rows(); i++ {
		if r.Row(i).Int(0) == index {
			if r.Row(i).Int(3) > 0 {
				return false
			}
			r.DeleteRow(i)
			m.events = append(m.events, fmt.Sprintf("unbind:%d", index))
			return true
		}
	}
	return false
}

// onBind 官方 @0x167300 族（cust 199/208）：绑定（args: 序号, 参数1, 参数2,
// 类型）。
func (m *ShortCutModule) onBind(k *Kernel, self, from PersistID, args VarList) {
	m.BindShortcut(k, self, args.Int(0), args.String(1), args.String(2), args.Int(3))
}

// onUnbind 官方（cust 200）：解绑（args: 序号）。
func (m *ShortCutModule) onUnbind(k *Kernel, self, from PersistID, args VarList) {
	m.UnbindShortcut(k, self, args.Int(0))
}

// onSwap 官方（cust 201）：交换两栏。
func (m *ShortCutModule) onSwap(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	r := o.Record("shortcut_rec")
	if r == nil {
		return
	}
	a, b := args.Int(0), args.Int(1)
	var ra, rb VarList
	for i := 0; i < r.Rows(); i++ {
		switch r.Row(i).Int(0) {
		case a:
			ra = r.Row(i)
		case b:
			rb = r.Row(i)
		}
	}
	if ra == nil || rb == nil {
		return
	}
	ra[0], rb[0] = b, a
	m.events = append(m.events, fmt.Sprintf("swap:%d↔%d", a, b))
}

// onClear 官方（cust 163=0xA3）：清栏。
func (m *ShortCutModule) onClear(k *Kernel, self, from PersistID, args VarList) {
	if o := k.GetGameObj(self); o != nil {
		if r := o.Record("shortcut_rec"); r != nil {
			r.Clear()
			m.events = append(m.events, "clear")
		}
	}
}

// onQuery 官方（cust 209）：快捷栏查询下发。
func (m *ShortCutModule) onQuery(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "query")
}

func (m *ShortCutModule) onSkillAdd(k *Kernel, self, from PersistID, args VarList)    {}
func (m *ShortCutModule) onSkillRemove(k *Kernel, self, from PersistID, args VarList) {}

// Events 流水（测试面）。
func (m *ShortCutModule) Events() []string { return m.events }

// ---- ItemAppendModule（Init 序 89）----

// ItemAppendModule 复刻官方 ItemAppendModule（Init @0xDCB70；符号 565）——
// 装备附加（时限道具）。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddIntCommandHook ("player",962=0x3C2)
//	AddEventCallback ("Equipment","OnRecover"/"OnEntry")
//	DeclareHeartBeat ("ItemAppendModule::HB_ItemTimeLimit")
type ItemAppendModule struct {
	BaseModule

	// events 流水（测试面："append:<id>"）。
	events []string
}

// Name 模块名。
func (m *ItemAppendModule) Name() string { return "ItemAppendModule" }

// Init 注册期（对齐官方 0xDCB70）。
func (m *ItemAppendModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnIntCommand("player", 962, m.onAppend)
	b.OnEvent("Equipment", "OnRecover", m.onEquip)
	b.OnEvent("Equipment", "OnEntry", m.onEquip)
	b.HeartBeat("ItemAppendModule::HB_ItemTimeLimit", m.hbItemTimeLimit)
	return nil
}

// onAppend 官方 @0xDD560（cmd 962=0x3C2）：附加请求（args: 装备, 附加物）。
func (m *ItemAppendModule) onAppend(k *Kernel, self, from PersistID, args VarList) {
	if len(args) < 2 {
		return
	}
	eq, _ := args[0].(PersistID)
	app, _ := args[1].(PersistID)
	eo, ao := k.GetGameObj(eq), k.GetGameObj(app)
	if eo == nil || ao == nil {
		return
	}
	m.events = append(m.events, fmt.Sprintf("append:%s←%s",
		eo.GetString("ConfigID"), ao.GetString("ConfigID")))
}

func (m *ItemAppendModule) onEquip(k *Kernel, self, from PersistID, args VarList) {}

// hbItemTimeLimit 官方心跳：附加物时限递减（ValidTime 族联动）。
func (m *ItemAppendModule) hbItemTimeLimit(k *Kernel, self PersistID, beats int32) {}

// Events 流水（测试面）。
func (m *ItemAppendModule) Events() []string { return m.events }

// ---- 无注册 API 族（七模块）----

// CInscriptionModule 复刻官方 CInscriptionModule（Init @0xB3E10；符号 122；
// 零内核注册）——铭文（rule/Inscription.xml 经 player includes 装载）。
type CInscriptionModule struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

func (m *CInscriptionModule) Name() string         { return "CInscriptionModule" }
func (m *CInscriptionModule) Init(k *Kernel) error { return nil }

// Inscribe 铭刻：铭文物 → 装备（PropPack 追加——镶嵌同构，铭文位；
// 铭文属性串同由配置面传入）。
func (m *CInscriptionModule) Inscribe(k *Kernel, equip, inscription PersistID, props string) bool {
	eo := k.GetGameObj(equip)
	if eo == nil || props == "" {
		return false
	}
	cur := eo.GetString("PropPack")
	if cur != "" && !strings.HasSuffix(cur, ";") {
		cur += ";"
	}
	_ = k.SetString(equip, "PropPack", cur+props)
	m.events = append(m.events, fmt.Sprintf("inscribe:%s", eo.GetString("ConfigID")))
	return true
}

// Events 流水（测试面）。
func (m *CInscriptionModule) Events() []string { return m.events }

// UpgradeRuleModule 复刻官方 UpgradeRuleModule（Init @0xB5570；符号 766；
// 零内核注册）——升级规则（强化/升级成功率面）。
type UpgradeRuleModule struct {
	BaseModule

	// SuccessRate 成功率（千分比；可注入——官方按表）。
	SuccessRate int32
	// events 流水（测试面）。
	events []string
}

func (m *UpgradeRuleModule) Name() string { return "UpgradeRuleModule" }
func (m *UpgradeRuleModule) Init(k *Kernel) error {
	if m.SuccessRate <= 0 {
		m.SuccessRate = 1000 // 千分比满值（官方按表；可注入）
	}
	return nil
}

// TryUpgrade 升级尝试（千分比判定——确定性伪随机由调用面注入）。
func (m *UpgradeRuleModule) TryUpgrade(k *Kernel, item PersistID, roll int32) bool {
	ok := roll < m.SuccessRate
	m.events = append(m.events, fmt.Sprintf("upgrade:%t", ok))
	return ok
}

// Events 流水（测试面）。
func (m *UpgradeRuleModule) Events() []string { return m.events }

// ToolItemFunc 复刻官方 ToolItemFunc（Init @0xC18C0；符号 219；零内核注册）——
// 工具物品功能包解释器。
type ToolItemFunc struct {
	BaseModule

	// calls 流水（测试面）。
	calls []string
}

func (m *ToolItemFunc) Name() string         { return "ToolItemFunc" }
func (m *ToolItemFunc) Init(k *Kernel) error { return nil }

// Dispatch 功能包分派（FuncPack → 功能方言：1 回城/2 传送/3 变身…——
// 编号语义由配置表定，此处路由面）。
func (m *ToolItemFunc) Dispatch(k *Kernel, player, item PersistID) int32 {
	o := k.GetGameObj(item)
	if o == nil {
		return 0
	}
	pack := o.GetInt("FuncPack")
	if pack != 0 {
		m.calls = append(m.calls, fmt.Sprintf("func:%d", pack))
	}
	return pack
}

// Calls 流水（测试面）。
func (m *ToolItemFunc) Calls() []string { return m.calls }

// ItemTreasure 复刻官方 ItemTreasure（Init @0xC0DC0；符号 565；零内核注册）——
// 宝箱（WealthPack）。
type ItemTreasure struct {
	BaseModule

	// TreasureOut 开箱产物出口。
	TreasureOut func(player PersistID, itemID string, count int32)
	// events 流水（测试面）。
	events []string
}

func (m *ItemTreasure) Name() string { return "ItemTreasure" }
func (m *ItemTreasure) Init(k *Kernel) error {
	if m.TreasureOut == nil {
		m.TreasureOut = func(player PersistID, itemID string, count int32) {}
	}
	return nil
}

// OpenTreasure 开箱：WealthPack 是 int 包编号（tool_item.xml 定案——产物
// 表经配置面），调用方按 configID 查包后传产物；此处验包（非零）+ 产出。
func (m *ItemTreasure) OpenTreasure(k *Kernel, player, item PersistID, productID string, count int32) bool {
	o := k.GetGameObj(item)
	if o == nil || o.GetInt("WealthPack") == 0 || productID == "" || count <= 0 {
		return false
	}
	m.TreasureOut(player, productID, count)
	m.events = append(m.events, fmt.Sprintf("treasure:%s×%d", productID, count))
	return true
}

// Events 流水（测试面）。
func (m *ItemTreasure) Events() []string { return m.events }

// InheritEnhance 复刻官方 InheritEnhance（Init @0xB9140；符号 766；零内核
// 注册）——继承强化（旧装备属性 → 新装备）。
type InheritEnhance struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

func (m *InheritEnhance) Name() string         { return "InheritEnhance" }
func (m *InheritEnhance) Init(k *Kernel) error { return nil }

// Inherit 继承：源装备 PropBuilt/PropRanBuilt（强化属性）→ 目标装备
// （固定属性 PropPack 不随迁——强化成果继承）。
func (m *InheritEnhance) Inherit(k *Kernel, from, to PersistID) bool {
	fo, to2 := k.GetGameObj(from), k.GetGameObj(to)
	if fo == nil || to2 == nil {
		return false
	}
	built := fo.GetString("PropBuilt")
	if built == "" {
		return false
	}
	_ = k.SetString(to, "PropBuilt", built)
	_ = k.SetString(from, "PropBuilt", "")
	m.events = append(m.events, fmt.Sprintf("inherit:%s→%s",
		fo.GetString("ConfigID"), to2.GetString("ConfigID")))
	return true
}

// Events 流水（测试面）。
func (m *InheritEnhance) Events() []string { return m.events }

// SlotItem 复刻官方 SlotItem（Init @0xC1390；符号 219；零内核注册）——槽位
// 物品。
type SlotItem struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

func (m *SlotItem) Name() string         { return "SlotItem" }
func (m *SlotItem) Init(k *Kernel) error { return nil }

// MountSlot 入槽（args: 槽位物品 → 装备；槽位方言由配置面）。
func (m *SlotItem) MountSlot(k *Kernel, equip, slot PersistID) bool {
	eo, so := k.GetGameObj(equip), k.GetGameObj(slot)
	if eo == nil || so == nil {
		return false
	}
	m.events = append(m.events, fmt.Sprintf("slot:%s←%s", eo.GetString("ConfigID"), so.GetString("ConfigID")))
	return true
}

// Events 流水（测试面）。
func (m *SlotItem) Events() []string { return m.events }

// EnchaseCompound 复刻官方 EnchaseCompound（Init @0xBFA40；符号 565；零内核
// 注册）——宝石合成（JewelLevel/CantCompound）。
type EnchaseCompound struct {
	BaseModule

	// CompoundOut 合成产物出口。
	CompoundOut func(player PersistID, itemID string, count int32)
	// events 流水（测试面）。
	events []string
}

func (m *EnchaseCompound) Name() string { return "EnchaseCompound" }
func (m *EnchaseCompound) Init(k *Kernel) error {
	if m.CompoundOut == nil {
		m.CompoundOut = func(player PersistID, itemID string, count int32) {}
	}
	return nil
}

// CompoundGem 宝石合成：同级×3 → 下一级（CantCompound 拒；等级方言——
// 产物 configID 由调用面按表换算，此处产出出口 + 流水）。
func (m *EnchaseCompound) CompoundGem(k *Kernel, player, gem PersistID, nextID string, count int32) bool {
	o := k.GetGameObj(gem)
	if o == nil || o.GetInt("CantCompound") != 0 || count < 3 {
		return false
	}
	m.CompoundOut(player, nextID, 1)
	m.events = append(m.events, fmt.Sprintf("gem:%s(lv%d)×%d→%s",
		o.GetString("ConfigID"), o.GetInt("JewelLevel"), count, nextID))
	return true
}

// Events 流水（测试面）。
func (m *EnchaseCompound) Events() []string { return m.events }

// FightBag 复刻官方 FightBag（Init @0x7CC1F0；符号 9；零内核注册——战斗包
// 小模块）。
type FightBag struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

func (m *FightBag) Name() string         { return "FightBag" }
func (m *FightBag) Init(k *Kernel) error { return nil }

// OpenFightBag 开战斗包（战斗中可用物品容器面）。
func (m *FightBag) OpenFightBag(k *Kernel, player PersistID) {
	m.events = append(m.events, fmt.Sprintf("open:%d", player.ID()))
}

// Events 流水（测试面）。
func (m *FightBag) Events() []string { return m.events }
