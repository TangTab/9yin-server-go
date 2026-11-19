package fsgame

// PlayerBaseModule 复刻官方 playerbasemodule（Init @0x160EC0，玩法表 Init 序 94）。
//
// 官方 Init 注册面（反汇编定案）：
//
//	AddClassCallback ("player","OnCreateClass",fn@0x161500 → createclass_from_xml)
//	AddEventCallback ("player","OnRecover",@0x161550)  SetInt 1 属性 + AddCritical 对象级挂载
//	AddEventCallback ("player","OnEntry",@0x1615D0)    落点校正：GetPosi* → GetHeight →
//	                    MoveTo 落地 → 水面判断 → SetLandPoint
//	AddEventCallback ("player","OnReady",@0x161980)    AsynServerTime + Custom 下发 +
//	                    RequestMacAddress + run_logic_lua_script
//	AddEventCallback ("player","OnStore",@0x161B00)    util_get_now/decode → SaveRoleInfo
//	AddEventCallback ("player","OnEntryGame",@0x161C50) FindData/AddData/SetDataInt 登记
//	AddEventCallback ("player","OnDisconnect",@0x161D40) Relive::OnPlayerDisconnect + 场景查询
//	AddIntCustomHook ("player",0x0C,@0x162490)  VarList(player,args) → Lua 桥
//	AddIntCustomHook ("player",0x33E,@0x161350)
//	AddIntCustomHook ("player",0x35C,@0x1613B0)
//	AddIntCustomHook ("player",0x3F7,@0x162A80)
//	DeclareCritical ("player","Level"/"Job"/… → C_OnPlayerLevelUpChange@0x162620、
//	                    C_OnPlayerJobChange@0x162720、C_OnPlayerChange@0x162820、
//	                    C_OnShamDeadChange@0x162920)
//	LoadResource：ini/server_version.xml
type PlayerBaseModule struct {
	BaseModule

	// OnEntryHooks 落点校正观测点（宿主/测试挂接；官方无此面）。
	OnEntryHooks []func(k *Kernel, player PersistID)
}

// Name 模块名（IModuleCreator::Name）。
func (m *PlayerBaseModule) Name() string { return "PlayerBaseModule" }

// Init 注册期（对齐官方 0x160EC0 注册表）。
func (m *PlayerBaseModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnClassEvent("player", "OnCreateClass", m.onCreateClass)
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.OnEvent("player", "OnEntry", m.onEntry)
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnEntryGame", m.onEntryGame)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnEvent("player", "OnDisconnect", m.onDisconnect)
	b.OnCommand("player", 0x0C, m.cmdLuaBridge)
	b.OnCommand("player", 0x33E, m.cmdQuery0x33E)
	b.OnCommand("player", 0x35C, m.cmdQuery0x35C)
	b.OnCommand("player", 0x3F7, m.cmdQuery0x3F7)
	b.Critical("player", "Level", m.onLevelUpChange)
	b.Critical("player", "Job", m.onJobChange)
	b.Critical("player", "Player", m.onPlayerChange)
	b.Critical("player", "ShamDead", m.onShamDeadChange)
	return nil
}

// LoadResource 官方读 ini/server_version.xml（版本串比对；Go 面记录装载即可）。
func (m *PlayerBaseModule) LoadResource(k *Kernel) error {
	k.Tracef("PlayerBaseModule: load ini/server_version.xml")
	return nil
}

// onCreateClass 官方 @0x161500 → createclass_from_xml(kernel, class, xml)：
// player 类 schema 由宿主装载（Kernel.LoadStruct 已触发本回调），此处仅记账。
func (m *PlayerBaseModule) onCreateClass(k *Kernel, class string) {
	k.Tracef("PlayerBaseModule: OnCreateClass %s", class)
}

// onRecover 官方 @0x161550：存档回载后 SetInt 1 项 + AddCritical 对象级挂载。
// Go 面等价：登记回载时间（LoginTime 属性，schema save=false）。
func (m *PlayerBaseModule) onRecover(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	_ = k.SetInt(self, "LoginTime", nowUnix())
	k.Tracef("PlayerBaseModule: OnRecover player=%v", self.ID())
}

// onEntry 官方 @0x1615D0：读 PosX/PosY/PosZ/Orient → GetHeight 校正 → MoveTo
// 落地 → 水面（GetWalkWaterExists/Height）→ 场景类 → SetLandPoint。
func (m *PlayerBaseModule) onEntry(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	// 官方数值细节（GetHeight/水面）依赖碰撞引擎（internal/fscollide 已复刻，
	// 接线属场景里程碑）；此处完成落点记账与 LandPoint 写入。
	if _, ok := o.Class().Prop("LandPoint"); ok {
		_ = o.Set("LandPoint", int32(1))
	}
	k.Tracef("PlayerBaseModule: OnEntry player=%v scene=%d", self.ID(), k.GetSceneId())
	for _, h := range m.OnEntryHooks {
		h(k, self)
	}
	// 场景侧对 player.OnEntry 的联动（SceneBaseModule 官方注册同名事件）。
}

// onReady 官方 @0x161980：AsynServerTime（服务器时间异步下发）→
// Custom("...") → RequestMacAddress（GPK 面）→ run_logic_lua_script。
func (m *PlayerBaseModule) onReady(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	k.Tracef("PlayerBaseModule: OnReady player=%v (AsynServerTime/Custom/MacAddress)", self.ID())
	// 官方以 Custom 通道下发服务器时间（帧号 0x1E 方言），登录方言里程碑接线。
}

// onEntryGame 官方 @0x161C50：FindData/AddData/SetDataInt（进入游戏登记）。
func (m *PlayerBaseModule) onEntryGame(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	k.Tracef("PlayerBaseModule: OnEntryGame player=%v", self.ID())
}

// onStore 官方 @0x161B00：util_get_now/decode → CreateRoleModule::SaveRoleInfo。
func (m *PlayerBaseModule) onStore(k *Kernel, self, from PersistID, args VarList) {
	if cr, ok := k.GetLogicModule("CreateRoleModule").(*CreateRoleModule); ok {
		_ = cr.SaveRoleInfo(k, self)
		return
	}
	// CreateRoleModule 未装配时退化为内核直存（保持存档链闭环）。
	_ = k.SavePlayerData(self, "player")
}

// onDisconnect 官方 @0x161D40：ReliveModule::OnPlayerDisconnect + 场景查询。
// Relive 不在首批，留 LeaveSceneTime 记账。
func (m *PlayerBaseModule) onDisconnect(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	_ = k.SetInt(self, "LeaveSceneTime", nowUnix())
	k.Tracef("PlayerBaseModule: OnDisconnect player=%v", self.ID())
}

// cmdLuaBridge 官方 @0x162490（命令 0x0C）：VarList(player, args) →
// run_logic_lua_script —— 通用 Lua 命令桥。
func (m *PlayerBaseModule) cmdLuaBridge(k *Kernel, self, from PersistID, args VarList) {
	if k.Lua != nil {
		if err := k.Lua("player_cmd_0x0C", append(VarList{self}, args...)); err != nil {
			k.Tracef("PlayerBaseModule: lua bridge: %v", err)
		}
	}
}

// cmdQuery0x33E 官方 @0x161350（命令 0x33E=830）。
func (m *PlayerBaseModule) cmdQuery0x33E(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("PlayerBaseModule: cmd 0x33E player=%v", self.ID())
}

// cmdQuery0x35C 官方 @0x1613B0（命令 0x35C=860）。
func (m *PlayerBaseModule) cmdQuery0x35C(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("PlayerBaseModule: cmd 0x35C player=%v", self.ID())
}

// cmdQuery0x3F7 官方 @0x162A80（命令 0x3F7=1015）。
func (m *PlayerBaseModule) cmdQuery0x3F7(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("PlayerBaseModule: cmd 0x3F7 player=%v", self.ID())
}

// C_OnPlayerLevelUpChange 官方 @0x162620（DeclareCritical "player","Level"）。
func (m *PlayerBaseModule) onLevelUpChange(k *Kernel, self PersistID, prop string, old Var) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	// 升级联动：HP/MP 重算等属 LevelModule/UpgradeRule 玩法域，此处仅观测。
	k.Tracef("PlayerBaseModule: LevelUp %d→%d", old.(int32), o.GetInt("Level"))
}

// C_OnPlayerJobChange 官方 @0x162720。
func (m *PlayerBaseModule) onJobChange(k *Kernel, self PersistID, prop string, old Var) {
	k.Tracef("PlayerBaseModule: JobChange %v", old)
}

// C_OnPlayerChange 官方 @0x162820（玩家综合属性变更）。
func (m *PlayerBaseModule) onPlayerChange(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnShamDeadChange 官方 @0x162920（装死开关）。
func (m *PlayerBaseModule) onShamDeadChange(k *Kernel, self PersistID, prop string, old Var) {
	k.Tracef("PlayerBaseModule: ShamDead %v", old)
}

// nowUnix 秒级时钟（util_get_now 等价）。
func nowUnix() int32 {
	return int32(unixNow())
}
