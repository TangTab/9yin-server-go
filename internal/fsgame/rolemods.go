package fsgame

import "fmt"

// CreateRoleModule 复刻官方 createrolemodule（Init @0x1B4430，玩法表
// Init 序 133；符号 1505，建角/角色信息底座）。
//
// 官方 Init 注册面：
//
//	AddEventCallback ("role","OnCreateRole",@0x1B47F0)  建角主流程
//	AddIntCommandHook ("player",0x375=885,@0x1B47A0)    客户端建角命令
//	记录：skill / shortcut_rec / normal_attack（@0x1BB430 登记初值行）
//
// OnCreateRole 主流程（@0x1B47F0 反汇编定案）：
//
//	Trace×4 → GetServerId → IsCanCreate(server, camp, race, job) 校验 →
//	SetInt×7（ServerID/Camp/Race/Job/Level=1/…）→ ClearRecord →
//	SetString(名字) → SetInt×4 → RoleExterior::SetRoleArtKey/UpdateExterior/
//	SetRoleExposed/UpdateHeadDetails → get_camp_race_job_id。
//
// 命令 0x375（@0x1B47A0）→ SaveRoleInfo；PlayerBase OnStore 也调 SaveRoleInfo。
type CreateRoleModule struct {
	BaseModule
	// Created 建角流水（角色名 → 持久句柄）。
	Created []string
	// Saved SaveRoleInfo 调用数（0x375 命令 + OnStore 链）。
	Saved int32
}

// Name 模块名。
func (m *CreateRoleModule) Name() string { return "CreateRoleModule" }

// Init 注册期（对齐官方 0x1B4430）。
func (m *CreateRoleModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("role", "OnCreateRole", m.onCreateRole)
	b.OnCommand("player", 0x375, m.cmdCreateRole)
	return nil
}

// LoadResource 官方读 ini/CreateRole/RoleBaseProp.xml（建角基础属性表；
// testdata 已有同构副本）。
func (m *CreateRoleModule) LoadResource(k *Kernel) error {
	k.Tracef("CreateRoleModule: load ini/CreateRole/RoleBaseProp.xml")
	return nil
}

// IsCanCreate 官方 ?IsCanCreate@CreateRoleModule@@QBE_N(IKernel*,H,H,H)：
// 阵营/种族/职业组合合法性。官方表驱动（RoleBaseProp.xml）；Go 面先收
// 组合非零校验，表接线随静态数据里程碑。
func (m *CreateRoleModule) IsCanCreate(camp, race, job int32) bool {
	return camp != 0 && race != 0 && job != 0
}

// cmdCreateRole 官方 @0x1B47A0（命令 0x375）：SaveRoleInfo。
func (m *CreateRoleModule) cmdCreateRole(k *Kernel, self, from PersistID, args VarList) {
	if err := m.SaveRoleInfo(k, self); err != nil {
		k.Tracef("CreateRoleModule: SaveRoleInfo: %v", err)
	}
}

// onCreateRole 官方 @0x1B47F0 建角主流程（role 对象上的属性初始化链）。
// args 方言（官方 OnCreateRole 回调参数）：名字/阵营/种族/职业/外观串。
func (m *CreateRoleModule) onCreateRole(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	name := args.String(0)
	camp, race, job := args.Int(1), args.Int(2), args.Int(3)
	if !m.IsCanCreate(camp, race, job) {
		k.Tracef("CreateRoleModule: IsCanCreate 拒绝 camp=%d race=%d job=%d", camp, race, job)
		return
	}
	// 官方 SetInt 链（属性名对齐 struct/player/player.xml schema）。
	for prop, v := range map[string]int32{
		"Camp": camp, "Race": race, "Job": job, "Level": 1,
	} {
		if _, ok := o.Class().Prop(prop); ok {
			_ = k.SetInt(self, prop, v)
		}
	}
	// 官方引擎内部属性 name（struct_template.xml internals，widestr）；
	// 存档真档以 Name 面呈现，此处两种定名都写（存在才写）。
	if _, ok := o.Class().Prop("name"); ok {
		_ = k.SetString(self, "name", name)
	}
	if _, ok := o.Class().Prop("Name"); ok {
		_ = k.SetString(self, "Name", name)
	}
	if _, ok := o.Class().Prop("RoleCreateTime"); ok {
		_ = k.SetString(self, "RoleCreateTime", fmt.Sprintf("%d", unixNow()))
	}
	// 官方 ClearRecord + skill/shortcut_rec/normal_attack 初值行（@0x1BB430）。
	for _, rec := range []string{"skill", "shortcut_rec", "normal_attack"} {
		if r := o.Record(rec); r != nil {
			r.Clear()
		}
	}
	m.Created = append(m.Created, name)
	k.Tracef("CreateRoleModule: OnCreateRole %s camp=%d race=%d job=%d", name, camp, race, job)
}

// SaveRoleInfo 官方 ?SaveRoleInfo@CreateRoleModule@@QAE_N(IKernel*,PERSISTID)：
// 角色信息落存档（PlayerBase OnStore 复用此入口）。
func (m *CreateRoleModule) SaveRoleInfo(k *Kernel, role PersistID) error {
	m.Saved++
	return k.SavePlayerData(role, "player")
}

// LevelModule 复刻官方 levelmodule（Init @0x221E80，玩法表 Init 序 157）。
//
// 官方 Init 注册面：
//
//	AddEventCallback ("player","OnRecover",@0x223590)   回载后经验/等级核对
//	DeclareCritical ("player","Exp",C_OnExpChanged@0x222CE0)
//	DeclareCritical ("player","Level",C_OnLevelChanged@0x222E60)
//
// 静态表：ini/Level/Level.xml（经验曲线）。
type LevelModule struct {
	BaseModule
	// LevelUps 升级事件计数（C_OnLevelChanged 观测）。
	LevelUps int32
	// ExpTable 经验曲线（Level.xml：等级 → 升级所需经验；官方 GetLevelExp 查询面）。
	ExpTable map[int32]int64
}

// Name 模块名。
func (m *LevelModule) Name() string { return "LevelModule" }

// Init 注册期（对齐官方 0x221E80）。
func (m *LevelModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onRecover)
	b.Critical("player", "PlayerExp", m.onExpChanged)
	b.Critical("player", "Level", m.onLevelChanged)
	return nil
}

// LoadResource 官方读 ini/Level/Level.xml。
func (m *LevelModule) LoadResource(k *Kernel) error {
	m.ExpTable = DefaultLevelExpTable
	k.Tracef("LevelModule: load ini/Level/Level.xml (%d 档)", len(m.ExpTable))
	return nil
}

// DefaultLevelExpTable 内置经验曲线（缺档兜底；官方表以 Level.xml 为准）。
var DefaultLevelExpTable = map[int32]int64{
	1: 100, 2: 260, 3: 480, 4: 780, 5: 1180,
}

// onRecover 官方 @0x223590：存档回载后核对 Exp/Level 一致性（溢出升级补齐）。
func (m *LevelModule) onRecover(k *Kernel, self, from PersistID, args VarList) {
	m.settleExp(k, self)
}

// onExpChanged 官方 C_OnExpChanged @0x222CE0：经验变更 → 跨档升级判定。
func (m *LevelModule) onExpChanged(k *Kernel, self PersistID, prop string, old Var) {
	m.settleExp(k, self)
}

// settleExp 经验结算：PlayerExp 达到 PlayerUpgradeExp（属性面，Level.xml
// 装载；零值回落内置曲线）则 Level+1（触发 Level 临界，形成
// PlayerBase::C_OnPlayerLevelUpChange 联动——官方链路）。
func (m *LevelModule) settleExp(k *Kernel, self PersistID) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if _, ok := o.Class().Prop("PlayerExp"); !ok {
		return
	}
	for {
		lv := o.GetInt("Level")
		need := int64(o.GetInt("PlayerUpgradeExp"))
		if need == 0 {
			need = m.ExpTable[lv]
		}
		if need == 0 {
			return
		}
		exp := varToInt64(o.Get("PlayerExp"))
		if exp < need {
			return
		}
		_ = k.SetInt(self, "Level", lv+1) // 临界联动 PlayerBase/升级规则
		_ = k.SetProp(self, "PlayerExp", exp-need)
	}
}

// varToInt64 属性值宽容读（exp.xml 定型 int，代码面按 int64 算）。
func varToInt64(v Var) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int32:
		return int64(x)
	case int:
		return int64(x)
	case float64:
		return int64(x)
	}
	return 0
}

// onLevelChanged 官方 C_OnLevelChanged @0x222E60。
func (m *LevelModule) onLevelChanged(k *Kernel, self PersistID, prop string, old Var) {
	m.LevelUps++
	k.Tracef("LevelModule: LevelUp → %d", k.GetGameObj(self).GetInt("Level"))
}

// RequestModule 复刻官方 requestmodule（Init @0x877180，玩法表 Init 序 180；
// 符号 1193，玩家请求/开关族）。
//
// 官方 Init 注册面（命令号 0x62..0x6E 族）：
//
//	AddEventCallback ("player","OnLeaveScene",@0x87C360)
//	AddIntCustomHook 98=@0x87A500 99=@0x87A840 100=@0x878960 101=@0x878A00
//	                 102=@0x8790D0 103=@0x87AB80 104=@0x87AE90
//	AddIntCommandHook 100=@0x879260 101=@0x87A250 102=@0x879800
//
// 处理器共性（反汇编定性）：Trace + GetMemberId（→请求挂载/回执面）。
type RequestModule struct {
	BaseModule
	// handled 命令号流水（观测面）。
	handled []int32
}

// Name 模块名。
func (m *RequestModule) Name() string { return "RequestModule" }

// Init 注册期（对齐官方 0x877180）。
func (m *RequestModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnLeaveScene", func(k *Kernel, self, from PersistID, args VarList) {
		k.Tracef("RequestModule: OnLeaveScene player=%v", self.ID())
	})
	for _, cmd := range []int32{98, 99, 100, 101, 102, 103, 104, 0x6D} {
		c := cmd
		b.OnCommand("player", c, func(k *Kernel, self, from PersistID, args VarList) {
			m.requestCmd(k, self, c)
		})
	}
	return nil
}

// requestCmd 官方族处理器：Trace + GetMemberId。
func (m *RequestModule) requestCmd(k *Kernel, self PersistID, cmd int32) {
	m.handled = append(m.handled, cmd)
	k.Tracef("RequestModule: cmd %d player=%v member=%d", cmd, self.ID(), k.MemberID)
}

// Handled 命令流水（测试面）。
func (m *RequestModule) Handled() []int32 { return m.handled }
