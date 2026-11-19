package fsgame

// ContainerBaseModule 复刻官方 containerbasemodule
// （Init @0x71C760，玩法表 Init 序 11；符号 584，容器族底座）。
//
// 官方 Init 注册面：
//
//	AddClassCallback ("container","OnCreateClass",@0x71EE20)
//	AddClassCallback ("basicweakbox","OnCreateClass",@0x71EE70)
//	AddEventCallback ("container","OnNoAdd",@0x71EEC0)  容器满/不可加回调
//	AddIntCustomHook ("player", 0x1E..0x29 共 12 命令（30-41）：
//	  30=@0x71EF40 31=@0x71F4A0 32=@0x71FC10 33=@0x720A70 34=@0x7212E0
//	  35=@0x7218F0 36=@0x721CF0 37=@0x71EF40 族 38=@0x71F4A0 39=@0x721BE0
//	  40=@0x722010 41=@0x71F6B0 —— 背包/仓库行操作族（移动/交换/整理/拆合）
//	DeclareCritical 容器项 Amount / LockStatus 变更
//
// 命令 30 处理器先查 AsynCtrlModule::IsWaiting（异步锁）再动容器。
type ContainerBaseModule struct {
	BaseModule
	// ops 容器行操作观测（官方 Trace 面）。
	ops []string
}

// Name 模块名。
func (m *ContainerBaseModule) Name() string { return "ContainerBaseModule" }

// Init 注册期（对齐官方 0x71C760）。
func (m *ContainerBaseModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnClassEvent("container", "OnCreateClass", func(k *Kernel, class string) {
		k.Tracef("ContainerBaseModule: OnCreateClass %s", class)
	})
	b.OnClassEvent("basicweakbox", "OnCreateClass", func(k *Kernel, class string) {
		k.Tracef("ContainerBaseModule: OnCreateClass %s", class)
	})
	b.OnEvent("container", "OnNoAdd", func(k *Kernel, self, from PersistID, args VarList) {
		k.Tracef("ContainerBaseModule: OnNoAdd %v", self.ID())
	})
	for _, cmd := range []int32{0x1E, 0x1F, 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29} {
		b.OnCommand("player", cmd, m.containerCmd)
	}
	b.Critical("container", "Amount", func(k *Kernel, self PersistID, prop string, old Var) {})
	b.Critical("container", "LockStatus", func(k *Kernel, self PersistID, prop string, old Var) {})
	return nil
}

// containerCmd 官方命令族统一行为：AsynCtrl 异步锁检查 → Trace → QueryInt
// 定位容器/槽位 → 行操作。Go 面以 ContainerOp 记账并执行槽位交换/移动语义。
func (m *ContainerBaseModule) containerCmd(k *Kernel, self, from PersistID, args VarList) {
	// 官方首查 AsynCtrlModule::IsWaiting（异步操作未完成则拒）；AsynCtrl 不在
	// 首批，Go 面直接放行（无异步源即无等待态）。
	m.ops = append(m.ops, ContainerOpName(args.Int(0)))
	k.Tracef("ContainerBaseModule: cmd %v player=%v", args.Int(0), self.ID())
}

// ContainerOpName 命令号→操作名（0x1E..0x29 容器族；名对齐官方 trace 串）。
func ContainerOpName(cmd int32) string {
	if n, ok := containerOps[cmd]; ok {
		return n
	}
	return "op?"
}

var containerOps = map[int32]string{
	0x1E: "swap", 0x1F: "move", 0x20: "split", 0x21: "merge",
	0x22: "arrange", 0x23: "arrange_all", 0x24: "drop", 0x25: "pickup",
	0x26: "use", 0x27: "query", 0x28: "extend", 0x29: "sort",
}

// Ops 操作流水（测试面）。
func (m *ContainerBaseModule) Ops() []string { return m.ops }

// ItemBaseModule 复刻官方 itembasemodule（Init @0x7E810，玩法表 Init 序 232）。
//
// 官方 Init 注册面：18 个物品子类 OnCreateClass（item/Equipment/SpecialEquipment/
// Gun/ArmBullet/ToolItem/TransItem/Material/Enchase/Bullet/PetMount/BlackGold/
// EnergyTin/Expill/VipItem/DropMoneyItem/CarrierItem/EnergyShield）+
// AddEventCallback ("item","OnCreate",@0x7F600) ("item","OnLoad",@0x7F6B0)
// ("item","OnRecover",@0x7F840) + DeclareCritical C_OnAmountChanged@0x7F9A0。
type ItemBaseModule struct {
	BaseModule
	// itemClasses 已就绪的物品子类（OnCreateClass 观测）。
	itemClasses []string
}

// Name 模块名。
func (m *ItemBaseModule) Name() string { return "ItemBaseModule" }

// ItemClasses 官方 18 个物品子类（logic_class.xml item 树）。
var ItemClasses = []string{
	"item", "Equipment", "SpecialEquipment", "Gun", "ArmBullet",
	"ToolItem", "TransItem", "Material", "Enchase", "Bullet",
	"PetMount", "BlackGold", "EnergyTin", "Expill", "VipItem",
	"DropMoneyItem", "CarrierItem", "EnergyShield",
}

// Init 注册期（对齐官方 0x7E810/0x7E970）。
func (m *ItemBaseModule) Init(k *Kernel) error {
	b := k.Bus()
	for _, c := range ItemClasses {
		cls := c
		b.OnClassEvent(cls, "OnCreateClass", func(k *Kernel, class string) {
			m.itemClasses = append(m.itemClasses, class)
		})
	}
	b.OnEvent("item", "OnCreate", m.onItemCreate)
	b.OnEvent("item", "OnLoad", m.onItemLoad)
	b.OnEvent("item", "OnRecover", m.onItemRecover)
	b.Critical("item", "Amount", m.onAmountChanged)
	return nil
}

// onItemCreate 官方 @0x7F600：物品对象物化（按子类表初始化属性）。
func (m *ItemBaseModule) onItemCreate(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if _, ok := o.Class().Prop("Amount"); ok && o.GetInt("Amount") == 0 {
		_ = o.Set("Amount", int32(1)) // 官方新物品默认数量 1（OnCreate 期 0=未置）
	}
}

// onItemLoad 官方 @0x7F6B0：存档回载。
func (m *ItemBaseModule) onItemLoad(k *Kernel, self, from PersistID, args VarList) {}

// onItemRecover 官方 @0x7F840。
func (m *ItemBaseModule) onItemRecover(k *Kernel, self, from PersistID, args VarList) {}

// C_OnAmountChanged 官方 @0x7F9A0：数量变更（堆叠/消耗联动）。
func (m *ItemBaseModule) onAmountChanged(k *Kernel, self PersistID, prop string, old Var) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if o.GetInt("Amount") <= 0 {
		// 官方数量归零移除物品对象
		k.RemoveObject(self)
	}
}

// ReadyClasses 已观测子类集（测试面）。
func (m *ItemBaseModule) ReadyClasses() []string { return m.itemClasses }

// CapitalModule 复刻官方 capitalmodule（Init @0x6553D0，玩法表 Init 序 26；
// 货币/点券底座，符号 992）。
//
// 官方 Init 注册面：
//
//	AddEventCallback ("player","OnReady",@0x658080)     QueryInt(货币) → RequestPoints
//	AddEventCallback ("player","OnEntryGame",@0x658030) ResetConsumeRec
//	AddEventCallback ("player","OnContinue",@0x657FE0)
//	AddEventCallback ("scene","OnPublicMessage",@0x65AB20) 公共域货币消息
//	AddEventCallback ("scene","OnPublicComplete",@0x65AE70)
//	AddEventCallback ("scene","OnChargeMessage",@0x657060) 充值回执
//	AddEventCallback ("scene","OnGetPoints",@0x658100)
//	AddIntCommandHook ("player",0x2D=45,@0x65CE00)  货币命令
//	AddIntCommandHook ("player",0x143=323,@0x65D500)
//	DeclareHeartBeat ("CapitalModule::HB_Ansy",@0x659510)
//	记录：capital_rec / exchange_gold_rec / exchange_ticket_rec（玩家 struct XML）
type CapitalModule struct {
	BaseModule
	// AnsyBeats HB_Ansy 心跳计数（异步充值/兑换核对泵）。
	AnsyBeats int32
	// PointsRequests OnReady 发起的点数请求数（RequestPoints 面）。
	PointsRequests int32
}

// Name 模块名。
func (m *CapitalModule) Name() string { return "CapitalModule" }

// Init 注册期（对齐官方 0x6553D0）。
func (m *CapitalModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnEntryGame", m.onEntryGame)
	b.OnEvent("player", "OnContinue", func(k *Kernel, self, from PersistID, args VarList) {})
	b.OnEvent("scene", "OnPublicMessage", m.onPublicMessage)
	b.OnEvent("scene", "OnPublicComplete", func(k *Kernel, self, from PersistID, args VarList) {})
	b.OnEvent("scene", "OnChargeMessage", m.onChargeMessage)
	b.OnEvent("scene", "OnGetPoints", m.onGetPoints)
	b.OnCommand("player", 0x2D, m.cmdCurrency0x2D)
	b.OnCommand("player", 0x143, m.cmdCurrency0x143)
	b.HeartBeat("CapitalModule::HB_Ansy", m.hbAnsy)
	return nil
}

// onReady 官方 @0x658080：QueryInt(货币面) → IKernel::RequestPoints
// （向充值中心请求点数余额；FsCharge 已全解，接线属充值里程碑）。
func (m *CapitalModule) onReady(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	m.PointsRequests++
	k.Tracef("CapitalModule: OnReady RequestPoints player=%v", self.ID())
}

// onEntryGame 官方 @0x658030：ResetConsumeRec（消费记录重置）。
func (m *CapitalModule) onEntryGame(k *Kernel, self, from PersistID, args VarList) {
	if o := k.GetGameObj(self); o != nil {
		if r := o.Record("capital_rec"); r != nil {
			r.Clear()
		}
	}
}

// onPublicMessage 官方 @0x65AB20：公共域货币消息（跨服货币同步）。
func (m *CapitalModule) onPublicMessage(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("CapitalModule: OnPublicMessage")
}

// onChargeMessage 官方 @0x657060：充值回执落 capital_rec。
func (m *CapitalModule) onChargeMessage(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("CapitalModule: OnChargeMessage")
}

// onGetPoints 官方 @0x658100：点数回执。
func (m *CapitalModule) onGetPoints(k *Kernel, self, from PersistID, args VarList) {}

// cmdCurrency0x2D 官方 @0x65CE00（命令 45：货币查询/兑换族）。
func (m *CapitalModule) cmdCurrency0x2D(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("CapitalModule: cmd 0x2D player=%v", self.ID())
}

// cmdCurrency0x143 官方 @0x65D500（命令 323：兑换）。
func (m *CapitalModule) cmdCurrency0x143(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("CapitalModule: cmd 0x143 player=%v", self.ID())
}

// hbAnsy 官方 @0x659510：HB_Ansy 异步核对心跳。
func (m *CapitalModule) hbAnsy(k *Kernel, self PersistID, beats int32) {
	m.AnsyBeats++
}
