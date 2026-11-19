package fsgame

// SceneBaseModule 复刻官方 scenebasemodule（Init @0x27C3C0，玩法表 Init 序 126）。
//
// 官方 Init 注册面：
//
//	AddClassCallback ("scene","OnCreateClass",@0x27C550)
//	AddEventCallback ("scene","OnCreate",@0x27C5A0)
//	AddEventCallback ("scene","OnDestroy",@0x27C8E0)
//	AddEventCallback ("player","OnEntry",@0x27CA20)   玩家入场联动
//	AddEventCallback ("VisibleNpc","OnEntry",@0x27CA20)
//	DeclareHeartBeat ("SceneBaseModule::HB_SyncCurrentTime",@0x27CB50)
type SceneBaseModule struct {
	BaseModule
	// SyncTimeBeats 心跳计数（HB_SyncCurrentTime 观测）。
	SyncTimeBeats int32
}

// Name 模块名。
func (m *SceneBaseModule) Name() string { return "SceneBaseModule" }

// Init 注册期（对齐官方 0x27C3C0）。
func (m *SceneBaseModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnClassEvent("scene", "OnCreateClass", func(k *Kernel, class string) {
		k.Tracef("SceneBaseModule: OnCreateClass %s", class)
	})
	b.OnEvent("scene", "OnCreate", m.onSceneCreate)
	b.OnEvent("scene", "OnDestroy", m.onSceneDestroy)
	b.OnEvent("player", "OnEntry", m.onObjectEntry)
	b.OnEvent("VisibleNpc", "OnEntry", m.onObjectEntry)
	b.HeartBeat("SceneBaseModule::HB_SyncCurrentTime", m.hbSyncCurrentTime)
	return nil
}

// onSceneCreate 官方 @0x27C5A0：场景对象物化（场景心跳/时间同步起点）。
func (m *SceneBaseModule) onSceneCreate(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("SceneBaseModule: scene OnCreate %v", self.ID())
}

// onSceneDestroy 官方 @0x27C8E0。
func (m *SceneBaseModule) onSceneDestroy(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("SceneBaseModule: scene OnDestroy %v", self.ID())
}

// onObjectEntry 官方 @0x27CA20：player/VisibleNpc 入场联动（场景侧登记）。
func (m *SceneBaseModule) onObjectEntry(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("SceneBaseModule: object OnEntry %v in scene %d", self.ID(), k.GetSceneId())
}

// hbSyncCurrentTime 官方 @0x27CB50：场景时间同步心跳（对全场下发服务器时间）。
func (m *SceneBaseModule) hbSyncCurrentTime(k *Kernel, self PersistID, beats int32) {
	m.SyncTimeBeats++
}

// LocationModule 复刻官方 locationmodule（Init @0x779A40，玩法表 Init 序 233）。
//
// 官方 Init 定案（0x779A40 反汇编）：
//   - 无事件/命令注册（被动服务模块）；
//   - LoadRes：ReleaseRes + 清 std::map<int,LOCATION_UNIT> + CXmlFile 装载
//     ini/Rule/door_data.xml（门/传送点表）；
//   - Init 缓存跨模块指针：GetLogicModule("AsynCtrlModule")→0x10EAFA24、
//     GetLogicModule("NpcSwitchLocateModule")→0x10EAFA28（RTDynamicCast）。
type LocationModule struct {
	BaseModule
	// doors LOCATION_UNIT 表：编号 → 门/传送点单元（door_data.xml）。
	doors map[int32]LocationUnit
	// asynCtrl/npcSwitch 跨模块缓存面（官方全局 0x10EAFA24/0x10EAFA28）。
	asynCtrl  Module
	npcSwitch Module
}

// LocationUnit ini/Rule/door_data.xml 的传送单元（官方 Property 属性表风格：
// ID/Name/TargetSceneID/Target{X,Y,Z}/TargetOrient/MoveTarget{X,Y,Z}）。
type LocationUnit struct {
	ID                  int32
	Name                string
	Scene               int32   // TargetSceneID
	X, Y, Z             float32 // Target*
	Orient              float32
	MoveX, MoveY, MoveZ float32 // MoveTarget*（缺省回落 Target*）
}

// Name 模块名。
func (m *LocationModule) Name() string { return "LocationModule" }

// Init 官方 @0x779A40：LoadRes + 跨模块指针缓存。
func (m *LocationModule) Init(k *Kernel) error {
	m.doors = make(map[int32]LocationUnit)
	m.asynCtrl = k.GetLogicModule("AsynCtrlModule")
	m.npcSwitch = k.GetLogicModule("NpcSwitchLocateModule")
	return m.LoadResource(k)
}

// LoadResource 官方 LoadRes @0x779B50：先 Release 再装载 door_data.xml。
// DoorData 为官方 XML 原文注入（宿主经 ResourcePath 读取；测试直接给内容）。
func (m *LocationModule) LoadResource(k *Kernel) error {
	m.doors = make(map[int32]LocationUnit)
	if k.ResourcePath == "" {
		return nil // 官方缺文件仅告警；Go 面空资源路径跳过
	}
	data, err := readResource(k.ResourcePath, "ini/Rule/door_data.xml")
	if err != nil {
		k.Tracef("LocationModule: door_data.xml 不可用: %v", err)
		return nil
	}
	units, err := ParseDoorDataXML(data)
	if err != nil {
		return err
	}
	for _, u := range units {
		m.doors[u.ID] = u
	}
	k.Tracef("LocationModule: door_data.xml %d units", len(m.doors))
	return nil
}

// GetSwitchLocation 官方 IKernel::GetSwitchLocation 的模块侧来源：
// 按门编号取传送落点。
func (m *LocationModule) GetSwitchLocation(doorID int32) (LocationUnit, bool) {
	u, ok := m.doors[doorID]
	return u, ok
}

// Doors 全部门单元（场景切换/NpcSwitchLocate 联动面）。
func (m *LocationModule) Doors() map[int32]LocationUnit { return m.doors }

// MotionModule 复刻官方 motionmodule（Init @0x15CAD0，玩法表 Init 序 134）。
//
// 官方 Init 注册面：
//
//	AddEventCallback ("player","OnMotion",@0x15F990)
//	AddEventCallback ("player","OnRotate",@0x15F9F0)
//	AddEventCallback ("player","OnReady",@0x15FA20)
//	AddEventCallback ("player","OnRecover",@0x15FAB0)
//	AddIntCustomHook ("player",0x03,@0x15FD70)  移动命令：QueryInt → ResetMoveSpeed → SetInt
//	AddIntCustomHook ("player",0x0F,@0x15D0C0)
//	AddIntCustomHook ("player",0x37A,@0x1602C0)
//	AddIntCustomHook ("player",0x37B,@0x160B80)
//	DeclareHeartBeat ("MotionModule::H_FLOW_FALL_DOWN",@0x160710)
//	DeclareCritical ("player","MoveType"/"Speed"/"GravityAdd"/"JumpSpeedAdd",
//	                    C_OnMoveTypeChanged@0x15FE20 / C_OnSpeedChanged@0x15FE70 /
//	                    C_OnGravityAddChanged@0x15FF10 / C_OnJumpSpeedAddChanged@0x15FEC0)
type MotionModule struct {
	BaseModule
	// MoveCommands 命令 0x03 观测（ResetMoveSpeed 联动面）。
	MoveCommands int32
}

// Name 模块名。
func (m *MotionModule) Name() string { return "MotionModule" }

// Init 注册期（对齐官方 0x15CAD0）。
func (m *MotionModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnMotion", func(k *Kernel, self, from PersistID, args VarList) {
		k.Tracef("MotionModule: OnMotion player=%v", self.ID())
	})
	b.OnEvent("player", "OnRotate", func(k *Kernel, self, from PersistID, args VarList) {
		k.Tracef("MotionModule: OnRotate player=%v", self.ID())
	})
	b.OnEvent("player", "OnReady", func(k *Kernel, self, from PersistID, args VarList) {})
	b.OnEvent("player", "OnRecover", func(k *Kernel, self, from PersistID, args VarList) {})
	b.OnCommand("player", 0x03, m.cmdMove)
	b.OnCommand("player", 0x0F, m.cmd0x0F)
	b.OnCommand("player", 0x37A, m.cmd0x37A)
	b.OnCommand("player", 0x37B, m.cmd0x37B)
	b.HeartBeat("MotionModule::H_FLOW_FALL_DOWN", func(k *Kernel, self PersistID, beats int32) {})
	b.Critical("player", "MoveType", func(k *Kernel, self PersistID, prop string, old Var) {})
	b.Critical("player", "Speed", func(k *Kernel, self PersistID, prop string, old Var) {})
	b.Critical("player", "GravityAdd", func(k *Kernel, self PersistID, prop string, old Var) {})
	b.Critical("player", "JumpSpeedAdd", func(k *Kernel, self PersistID, prop string, old Var) {})
	return nil
}

// cmdMove 官方 @0x15FD70（命令 0x03）：QueryInt(移动态) → ResetMoveSpeed → SetInt。
func (m *MotionModule) cmdMove(k *Kernel, self, from PersistID, args VarList) {
	m.MoveCommands++
	m.resetMoveSpeed(k, self)
	_ = k.SetInt(self, "MoveType", args.Int(0))
}

// resetMoveSpeed 官方 ?ResetMoveSpeed@MotionModule@@CAX：按属性面重置移速。
func (m *MotionModule) resetMoveSpeed(k *Kernel, self PersistID) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if _, ok := o.Class().Prop("Speed"); ok {
		_ = o.Set("Speed", float32(0)) // 官方按表重算；零值占位对齐"重置"语义
	}
}

// cmd0x0F 官方 @0x15D0C0（命令 15：旋转/朝向类）。
func (m *MotionModule) cmd0x0F(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("MotionModule: cmd 0x0F player=%v", self.ID())
}

// cmd0x37A 官方 @0x1602C0（命令 890）。
func (m *MotionModule) cmd0x37A(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("MotionModule: cmd 0x37A player=%v", self.ID())
}

// cmd0x37B 官方 @0x160B80（命令 891）。
func (m *MotionModule) cmd0x37B(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("MotionModule: cmd 0x37B player=%v", self.ID())
}
