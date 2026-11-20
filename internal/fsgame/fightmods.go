package fsgame

import "fmt"

// 三期：战斗域整族五模块（docs/official/FsGame全解与Go复刻.md §10 建议批次）。
// 官方源码树：fsgame\skillmodule\（skillmodule.cpp/fight\fightmodule.cpp/
// buffermodule.cpp/damagecalculatemodule.cpp）。

// DamageType 官方 DAMAGE_TYPE 枚举（SkillDamageCalculate 出参）。
type DamageType int32

// 伤害类型（风1火2雷3土4水5暗6自然7蒸汽8——skilluse.xml DamageType 串方言）。
const (
	DamagePhysical DamageType = iota
	DamageMagic
	DamageWind
	DamageFire
	DamageThunder
	DamageEarth
	DamageWater
	DamageDark
	DamageNature
	DamageSteam
)

// ---- CoolDownModule ----

// CoolDownModule 复刻官方 cooldownmodule（Init @0x36FBF0，玩法表 Init 序 64；
// 符号 72）——冷却底座（纯 Lua 面，无命令/心跳）。
//
// 官方 Init 注册面：
//
//	DeclareLuaExt ×4：nx_is_cooldown@0x36F380 / nx_begin_cooldown@0x36F5E0 /
//	    nx_clear_all_skill_cooldown@0x36F910 / nx_clear_all_cooldown@0x36FA80
//
// 冷却数据落 cooldown_rec（struct/rule/Common/commonfighter.xml，4 列 ×512）：
// 0 冷却分类ID(dword) / 1 开始时间(int64) / 2 结束时间(int64) / 3 冷却大分类ID。
// 技能侧 CoolDownTime/CoolDownTimeAdd/AddCoolDownTime 属性（cooldown_new.xml）。
type CoolDownModule struct {
	BaseModule
	// begun 冷却开启流水（测试面）。
	begun []cooldownBegin
}

type cooldownBegin struct {
	Self     PersistID
	Category uint32
	Start    int64
	End      int64
	Team     uint32
}

// Name 模块名。
func (m *CoolDownModule) Name() string { return "CoolDownModule" }

// Init 注册期（对齐官方 0x36FBF0：仅 Lua 扩展面）。
func (m *CoolDownModule) Init(k *Kernel) error { return nil }

// NxIsCooldown nx_is_cooldown（官方参数校验：object/int）：分类是否在冷却中
//（结束时间 > 当前时刻）。大分类（列 3）同组亦命中（冷却组传递）。
func (m *CoolDownModule) NxIsCooldown(k *Kernel, self PersistID, category uint32) bool {
	o := k.GetGameObj(self)
	if o == nil {
		return false
	}
	r := o.Record("cooldown_rec")
	if r == nil {
		return false
	}
	now := unixNowMilli()
	for i := 0; i < r.Rows(); i++ {
		row := r.Row(i)
		if uint32(row.Int(0)) != category && uint32(row.Int(3)) != category {
			continue
		}
		if row.Int64(2) > now {
			return true
		}
	}
	return false
}

// NxBeginCooldown nx_begin_cooldown（官方参数：object/int/int64/int/int64 →
//（对象, 分类ID, 开始, 结束, 大分类））：同分类行刷新，否则追加。
func (m *CoolDownModule) NxBeginCooldown(k *Kernel, self PersistID, category uint32, start, end int64, team uint32) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	r := o.Record("cooldown_rec")
	if r == nil {
		return
	}
	for i := 0; i < r.Rows(); i++ {
		if uint32(r.Row(i).Int(0)) == category {
			r.ModifyRow(i, VarList{int32(category), start, end, int32(team)})
			m.begun = append(m.begun, cooldownBegin{self, category, start, end, team})
			return
		}
	}
	if r.Rows() < r.Def().MaxRows {
		r.AppendRow(VarList{int32(category), start, end, int32(team)})
		m.begun = append(m.begun, cooldownBegin{self, category, start, end, team})
	}
}

// NxClearAllSkillCooldown nx_clear_all_skill_cooldown：清技能冷却
//（大分类=技能组的行；Go 面清全表，官方按组过滤的组表见 nocooldown 族）。
func (m *CoolDownModule) NxClearAllSkillCooldown(k *Kernel, self PersistID) {
	m.NxClearAllCooldown(k, self)
}

// NxClearAllCooldown nx_clear_all_cooldown：清冷却表。
func (m *CoolDownModule) NxClearAllCooldown(k *Kernel, self PersistID) {
	if o := k.GetGameObj(self); o != nil {
		if r := o.Record("cooldown_rec"); r != nil {
			r.Clear()
		}
	}
}

// Begun 冷却开启流水（测试面）。
func (m *CoolDownModule) Begun() []cooldownBegin { return m.begun }

// ---- DamageCalculateModule ----

// DamageCalculateModule 复刻官方 damagecalculatemodule
// （Init @0x3D3010 / LoadResource @0x3D3870，玩法表 Init 序 57；符号 59）。
//
// 官方 Init 注册面：
//
//	DeclareLuaExt ("nx_reload_damage_calculate_data",@0x3D2EF0)
//
// 核心 API（符号定案）：
//
//	SkillDamageCalculate(kernel, attacker, defender, skill, VarList&, DAMAGE_TYPE&)
//	    → bool    伤害主入口（TargetSkill/RandDmgType 随机浮动面）
//	CalculateFormula(kernel, attacker, defender, ?, int) → float  公式求值
//	CalculateParamValue → float[8] 参数包
//	公式源：m_pStaticDataQueryModule（StaticDataQuery 静态数据域，未复刻批次）。
type DamageCalculateModule struct {
	BaseModule

	// RandDmgType 随机浮动开关（官方 TargetSkill/RandDmgType 面；Go 测试可关）。
	RandDmgType bool
	// FloatRatio 伤害浮动比例（官方随机 ±区间；Go 面固定 0.1=±10%）。
	FloatRatio float64
	// reloads 配置重载计数（nx_reload_damage_calculate_data 面）。
	reloads int
}

// Name 模块名。
func (m *DamageCalculateModule) Name() string { return "DamageCalculateModule" }

// Init 注册期（对齐官方 0x3D3010）。
func (m *DamageCalculateModule) Init(k *Kernel) error {
	m.FloatRatio = 0.1
	return nil
}

// LoadResource 官方 @0x3D3870：公式数据装载（公式本体在 StaticDataQuery
// 静态数据域；Go 面记录装载，公式参数经 CalculateFormula 缝合）。
func (m *DamageCalculateModule) LoadResource(k *Kernel) error {
	k.Tracef("DamageCalculateModule: LoadResource (formula pack)")
	return nil
}

// NxReloadDamageCalculateData nx_reload_damage_calculate_data 的 Go 等价。
func (m *DamageCalculateModule) NxReloadDamageCalculateData() { m.reloads++ }

// SkillDamageCalculate 官方 @0x3D3120：技能伤害主入口。
// 公式面：base = 技能伤害基数（skill 的 Damage 基数或参数）；防御减伤
//（FightModule::NxGetDefend 缝合）后浮动（RandDmgType）。
// 返回（伤害类型, 结算值, 是否有效）。
func (m *DamageCalculateModule) SkillDamageCalculate(k *Kernel, attacker, defender, skill PersistID, args VarList) (DamageType, float64, bool) {
	if k.GetGameObj(attacker) == nil || k.GetGameObj(defender) == nil {
		return DamagePhysical, 0, false
	}
	base := args.Double(0)
	if base == 0 {
		if so := k.GetGameObj(skill); so != nil {
			base = float64(so.GetInt("Skill_Damage"))
		}
	}
	dt := DamageType(args.Int(1))
	defend := 0.0
	if fm, ok := k.GetLogicModule("FightModule").(*FightModule); ok && fm != nil {
		defend = fm.NxGetDefend(k, defender, dt)
	}
	dmg := base - defend
	if dmg < 0 {
		dmg = 0
	}
	if m.RandDmgType {
		dmg *= 1 - m.FloatRatio // Go 面取下浮（官方随机区间；测试可关）
	}
	return dt, dmg, true
}

// CalculateFormula 官方 @0x3D3690：公式求值（StaticDataQuery 公式包的
// Go 缝合面——线性公式：base×level 系数）。
func (m *DamageCalculateModule) CalculateFormula(k *Kernel, attacker, defender PersistID, formulaID, level int32) float64 {
	return float64(formulaID) * float64(level)
}

// Reloads 重载计数（测试面）。
func (m *DamageCalculateModule) Reloads() int { return m.reloads }

// ---- FightModule ----

// FightModule 复刻官方 skillmodule\fight\fightmodule（Init @0x32BBC0，
// 玩法表 Init 序 63；符号 398，12 Lua 面）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnRecover",@0x330AA0) / ("player","OnLeave",
//	    @0x330D10) / ("player","OnReady",@0x3309D0)
//	AddEventCallback ×6（AttackNpc/BattleCarrier/SkillNpc/MultiSkillNpc/
//	    TechnologyNpc/Pet，"OnLoad"，@0x330B50）
//	AddIntCommandHook ("player"/"AttackNpc"/"BattleCarrier",616,@0x330D40)
//	    OnCommandBeKill
//	AddIntCommandHook ("player",609,@0x337670) OnCommandChangeWeaponHardiness
//	AddIntCommandHook ("player",613,@0x3374E0) OnCommandChangeEquipmentHardiness
//	DeclareCritical ×4：C_OnHPChanged@0x330ED0（MaxHP/MaxHPAdd/HPRatio 重算 +
//	    Dead 判定）/ C_OnMPChanged@0x331080 / C_OnCantMoveChange@0x3311F0 /
//	    C_OnExpChanged@0x338000
//	DeclareLuaExt ×12：nx_damage/nx_damage_target/nx_isattacktarget/
//	    nx_canattacktarget/nx_can_damage_target/nx_get_defend/
//	    nx_send_fight_{miss,parry,dodge,vahp,failing,spite}
type FightModule struct {
	BaseModule

	// FightOut 战斗包出口（飘字/命中表现：miss/parry/dodge/vahp/failing/spite；
	// 官方经 Custom 通道下发，sync 层挂接）。
	FightOut func(kind string, attacker, defender PersistID, value float64)

	// dead 死亡判定流水（C_OnHPChanged → Dead=1 事件）。
	dead []PersistID
	// damageLog 伤害流水（nx_damage 面；测试面）。
	damageLog []damageEntry
}

type damageEntry struct {
	Attacker, Defender PersistID
	Type               DamageType
	Base, Actual       float64
}

// Name 模块名。
func (m *FightModule) Name() string { return "FightModule" }

// Init 注册期（对齐官方 0x32BBC0）。
func (m *FightModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onPlayerRecover)
	b.OnEvent("player", "OnLeave", m.onLeave)
	b.OnEvent("player", "OnReady", m.onPlayerReady)
	for _, cls := range []string{"AttackNpc", "BattleCarrier", "SkillNpc", "MultiSkillNpc", "TechnologyNpc", "Pet"} {
		b.OnEvent(cls, "OnLoad", m.onLoad)
	}
	for _, cls := range []string{"player", "AttackNpc", "BattleCarrier"} {
		b.OnIntCommand(cls, 616, m.onCommandBeKill)
	}
	b.OnIntCommand("player", 609, m.onChangeWeaponHardiness)
	b.OnIntCommand("player", 613, m.onChangeEquipmentHardiness)
	b.Critical("player", "HP", m.onHPChanged)
	b.Critical("player", "MP", m.onMPChanged)
	b.Critical("player", "CantMove", m.onCantMoveChange)
	b.Critical("player", "Exp", m.onExpChanged)
	return nil
}

// onPlayerRecover 官方 @0x330AA0：战斗面回载（HP/MP 上限链重算）。
func (m *FightModule) onPlayerRecover(k *Kernel, self, from PersistID, args VarList) {
	m.recalcRatio(k, self)
}

// onLoad 官方 @0x330B50：战斗 NPC 族 OnLoad（战斗面初始化）。
func (m *FightModule) onLoad(k *Kernel, self, from PersistID, args VarList) {
	m.recalcRatio(k, self)
}

// onLeave 官方 @0x330D10：离场（脱离战斗态）。
func (m *FightModule) onLeave(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("FightModule: OnLeave player=%v", self.ID())
}

// onPlayerReady 官方 @0x3309D0。
func (m *FightModule) onPlayerReady(k *Kernel, self, from PersistID, args VarList) {}

// onCommandBeKill 官方 @0x330D40（命令 616=0x268）：击杀通知（战绩/仇恨联动面）。
func (m *FightModule) onCommandBeKill(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("FightModule: BeKill %v", self.ID())
}

// onChangeWeaponHardiness 官方 @0x337670（命令 609=0x261）：武器硬度变更。
func (m *FightModule) onChangeWeaponHardiness(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("FightModule: ChangeWeaponHardiness")
}

// onChangeEquipmentHardiness 官方 @0x3374E0（命令 613=0x265）：装备硬度变更。
func (m *FightModule) onChangeEquipmentHardiness(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("FightModule: ChangeEquipmentHardiness")
}

// C_OnHPChanged 官方 @0x330ED0：HP 变更 → HPRatio 重算（MaxHP/MaxHPAdd 面）
// + 归零判死（Dead=1 → SkillModule C_OnDeadChanged 联动）。
func (m *FightModule) onHPChanged(k *Kernel, self PersistID, prop string, old Var) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	m.recalcRatio(k, self)
	if o.GetInt("HP") <= 0 && o.GetInt("Dead") == 0 {
		_ = k.SetInt(self, "Dead", 1)
		m.dead = append(m.dead, self)
		k.Tracef("FightModule: %v dead", self.ID())
	}
}

// C_OnMPChanged 官方 @0x331080：MP 变更 → MPRatio 重算。
func (m *FightModule) onMPChanged(k *Kernel, self PersistID, prop string, old Var) {
	m.recalcRatio(k, self)
}

// C_OnCantMoveChange 官方 @0x3311F0：禁移状态变更（SkillModule H_CantMove 联动）。
func (m *FightModule) onCantMoveChange(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnExpChanged 官方 @0x338000：经验变更（战斗结算面 → LevelModule 链）。
func (m *FightModule) onExpChanged(k *Kernel, self PersistID, prop string, old Var) {}

// recalcRatio HPRatio/MPRatio 重算（MaxHP(+Add) 面；防御性除零）。
func (m *FightModule) recalcRatio(k *Kernel, self PersistID) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	maxHP := int64(o.GetInt("MaxHP")) + int64(o.GetInt("MaxHPAdd"))
	if maxHP > 0 {
		_ = k.SetInt(self, "HPRatio", int32(int64(o.GetInt("HP"))*100/maxHP))
	}
	maxMP := int64(o.GetInt("MaxMP")) + int64(o.GetInt("MaxMPAdd"))
	if maxMP > 0 {
		_ = k.SetInt(self, "MPRatio", int32(int64(o.GetInt("MP"))*100/maxMP))
	}
}

// NxDamage nx_damage（官方 @0x32A1C0，参数 object/object/int/int）：
// 伤害结算主入口——（攻击者, 防御者, 伤害类型, 基数）→ 扣血并广播 VaHP。
// 返回实际伤害值。
func (m *FightModule) NxDamage(k *Kernel, attacker, defender PersistID, dt DamageType, base float64) float64 {
	o := k.GetGameObj(defender)
	if o == nil {
		return 0
	}
	dmg, ok := m.NxCanDamageTarget(k, attacker, defender, dt, base)
	if !ok {
		return 0
	}
	hp := o.GetInt("HP") - int32(dmg)
	if hp < 0 {
		hp = 0
	}
	_ = k.SetInt(defender, "HP", hp)
	m.damageLog = append(m.damageLog, damageEntry{attacker, defender, dt, base, dmg})
	m.sendFight(k, "vahp", attacker, defender, dmg)
	return dmg
}

// NxCanDamageTarget nx_can_damage_target（官方 @0x32A4D0）：可伤判定
//（Dead=1 / CantBeAttack 面）→（结算值, 是否有效）。
func (m *FightModule) NxCanDamageTarget(k *Kernel, attacker, defender PersistID, dt DamageType, base float64) (float64, bool) {
	o := k.GetGameObj(defender)
	if o == nil {
		return 0, false
	}
	if o.GetInt("Dead") != 0 {
		return 0, false
	}
	if o.GetInt("CantBeAttack") != 0 {
		return 0, false
	}
	if dcm, ok := k.GetLogicModule("DamageCalculateModule").(*DamageCalculateModule); ok && dcm != nil {
		if _, dmg, ok := dcm.SkillDamageCalculate(k, attacker, defender, NullPersistID, VarList{base, int32(dt)}); ok {
			return dmg, true
		}
	}
	return base, true
}

// NxGetDefend nx_get_defend（官方 @0x32A6F0）：防御面（物理/法术分轨；
// 属性源自 playerfightpara 防御族：DefenseLevel(+Add) 防御等级 /
// SAllResistProb 绝对法术减伤率）。
func (m *FightModule) NxGetDefend(k *Kernel, defender PersistID, dt DamageType) float64 {
	o := k.GetGameObj(defender)
	if o == nil {
		return 0
	}
	if dt == DamagePhysical {
		return float64(o.GetInt("DefenseLevel") + o.GetInt("DefenseLevelAdd"))
	}
	return float64(o.GetInt("SAllResistProb"))
}

// NxIsAttackTarget nx_isattacktarget（官方 @0x329DD0）：是否当前攻击目标。
func (m *FightModule) NxIsAttackTarget(k *Kernel, self, target PersistID) bool {
	o := k.GetGameObj(self)
	return o != nil && o.Get("CurSkillTarget") == Var(target)
}

// NxCanAttackTarget nx_canattacktarget（官方 @0x329FC0）：可攻判定
//（Dead/CantBeAttack/阵营面——阵营细则属 FightRelation/PK 域）。
func (m *FightModule) NxCanAttackTarget(k *Kernel, self, target PersistID) bool {
	t := k.GetGameObj(target)
	if t == nil {
		return false
	}
	return t.GetInt("Dead") == 0 && t.GetInt("CantBeAttack") == 0
}

// nx_send_fight_* 六出口（飘字/表现包；官方 @0x32A9F0-0x32B460）。
func (m *FightModule) NxSendFightMiss(k *Kernel, attacker, defender PersistID) {
	m.sendFight(k, "miss", attacker, defender, 0)
}

func (m *FightModule) NxSendFightParry(k *Kernel, attacker, defender PersistID, value float64) {
	m.sendFight(k, "parry", attacker, defender, value)
}

func (m *FightModule) NxSendFightDodge(k *Kernel, attacker, defender PersistID) {
	m.sendFight(k, "dodge", attacker, defender, 0)
}

func (m *FightModule) NxSendFightVaHP(k *Kernel, attacker, defender PersistID, value float64) {
	m.sendFight(k, "vahp", attacker, defender, value)
}

func (m *FightModule) NxSendFightFailing(k *Kernel, attacker, defender PersistID, value float64) {
	m.sendFight(k, "failing", attacker, defender, value)
}

func (m *FightModule) NxSendFightSpite(k *Kernel, attacker, defender PersistID, value float64) {
	m.sendFight(k, "spite", attacker, defender, value)
}

func (m *FightModule) sendFight(k *Kernel, kind string, attacker, defender PersistID, value float64) {
	if m.FightOut != nil {
		m.FightOut(kind, attacker, defender, value)
	}
}

// Dead 死亡流水（测试面）。
func (m *FightModule) Dead() []PersistID { return m.dead }

// DamageLog 伤害流水（测试面）。
func (m *FightModule) DamageLog() []damageEntry { return m.damageLog }

// ---- BufferModule ----

// BufferModule 复刻官方 skillmodule\buffermodule（Init @0x3491A0 /
// LoadResource @0x349DC0，玩法表 Init 序 65；符号 1500）。
//
// 官方 Init 注册面（p2scan 反汇编定案，节选）：
//
//	AddEventCallback ("player","OnRecover",@0x34DAF0) / ("player","OnStore",
//	    @0x34DE70) / ("VisibleNpc","OnLoad",@0x34DAF0) / ("Pet","OnDestroy",
//	    @0x34E2B0)
//	AddClassCallback ("Buffer","OnCreateClass",@0x34D4F0)
//	AddEventCallback ("Buffer","OnLoad",@0x34D540) / ("Buffer","OnRecover",
//	    @0x34D730)
//	AddEventCallback ("BufferContainer","OnCreate",@0x34D910) /
//	    ("BufferContainer","OnAfterAdd",@0x34D970) / ("BufferContainer",
//	    "OnRemove",@0x34DA40)
//	AddEventCallback ("player"/"AttackNpc"/"BattleCarrier","OnSelect",@0x34E050)
//	AddIntCommandHook：616（BeKill）与 600/605/609-613/615/630-632/641
//	    （Buffer 表现族，三类同挂 @0x34E8B0）
//	AddIntCustomHook ("player",1021,@0x34EE30) OnCustomRemoveBuffer /
//	    ("player",1019,@0x34F060) OnCustomAddBuffer
//	DeclareHeartBeat ("BufferModule::H_BufferUpdate",@0x34F410)
//	DeclareCritical ×5：C_OnLogicStateChanged@0x359450 / C_OnVarPropNoChanged@
//	    0x34FF20 / C_OnBufferActive@0x350100 / C_OnBufferCount@0x350240
//	DeclareRecHook（OverLay）→ C_OnOverLayRecordChanged@0x350330
//	DeclareLuaExt ×6：nx_add_buffer/nx_remove_buffer/nx_find_buffer/
//	    nx_refresh_buffer_byvarprop/nx_reload_buffer_resource/nx_reload_buffer_res
//
// Buffer 类 schema（struct/skill/buffer_new.xml）：LifeTime(ms,0=持久)/
// ConfigID/UniqueID/Level/Category/MaxOverlayNumber/OverlayType/RemoveType/
// EffectType/LifeTimeType/IsDamage/CantClear/OffLineClear/DeadClear…
type BufferModule struct {
	BaseModule

	// beatMS 每拍推进的毫秒数（官方按 FxCore 定步泵；Go 测试面 100ms/拍）。
	beatMS int64

	// buffers 挂载表：目标 → Buffer 对象句柄序（BufferContainer 镜像）。
	buffers map[PersistID][]PersistID
	// nextUID Buffer UniqueID 序号。
	nextUID int64
	// added/removed 挂载/移除流水（测试面）。
	added   []string
	removed []string
}

// Name 模块名。
func (m *BufferModule) Name() string { return "BufferModule" }

// Init 注册期（对齐官方 0x3491A0）。
func (m *BufferModule) Init(k *Kernel) error {
	m.beatMS = 100
	m.buffers = make(map[PersistID][]PersistID)
	b := k.Bus()
	b.OnEvent("player", "OnRecover", m.onLoad)
	b.OnEvent("player", "OnStore", m.onStore)
	b.OnEvent("VisibleNpc", "OnLoad", m.onLoad)
	b.OnEvent("Pet", "OnDestroy", m.onPetDestroy)
	b.OnClassEvent("Buffer", "OnCreateClass", func(k *Kernel, class string) {
		k.Tracef("BufferModule: OnCreateClass Buffer (buffer_new.xml)")
	})
	b.OnEvent("Buffer", "OnLoad", m.onBufferLoad)
	b.OnEvent("Buffer", "OnRecover", m.onBufferRecover)
	b.OnEvent("BufferContainer", "OnCreate", m.onBufferContainerCreate)
	b.OnEvent("BufferContainer", "OnAfterAdd", m.onBufferContainerAfterAdd)
	b.OnEvent("BufferContainer", "OnRemove", m.onBufferContainerRemove)
	for _, cls := range []string{"player", "AttackNpc", "BattleCarrier"} {
		b.OnEvent(cls, "OnSelect", m.onObjectSelect)
		b.OnIntCommand(cls, 616, m.onCommandBeKill)
		for _, cmd := range []int32{600, 605, 609, 610, 611, 612, 613, 615, 630, 631, 632, 641} {
			b.OnIntCommand(cls, cmd, m.onBufferCmd)
		}
	}
	b.OnCommand("player", 1021, m.onCustomRemoveBuffer)
	b.OnCommand("player", 1019, m.onCustomAddBuffer)
	b.HeartBeat("BufferModule::H_BufferUpdate", m.hbBufferUpdate)
	b.Critical("player", "LogicState", m.onLogicStateChanged)
	b.Critical("player", "VarPropNo", m.onVarPropNoChanged)
	b.Critical("Buffer", "Active", m.onBufferActive)
	b.Critical("Buffer", "Count", m.onBufferCount)
	b.RecHook("OverLay", m.onOverLayRecordChanged)
	return nil
}

// LoadResource 官方 @0x349DC0：ini/ 下 Buff 配置族
//（Buff_Config.xml/Buff_Modify.xml/buff_static.xml；Go 面按存在性装载）。
func (m *BufferModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: BufferModule.LoadResource: ResourcePath 未设置")
	}
	n := 0
	for _, p := range []string{"ini/Skill/Buff_Config.xml", "ini/Skill/Buff_Modify.xml", "ini/Skill/buff_static.xml"} {
		if _, err := readResource(k.ResourcePath, p); err == nil {
			n++
		}
	}
	k.Tracef("BufferModule: LoadResource ini/Skill buff 族 (%d)", n)
	return nil
}

// onLoad 官方 @0x34DAF0：战斗体回载（Buffer 容器重建）。
func (m *BufferModule) onLoad(k *Kernel, self, from PersistID, args VarList) {}

// onStore 官方 @0x34DE70：存档（OffLineClear 的 Buffer 落档/清除面）。
func (m *BufferModule) onStore(k *Kernel, self, from PersistID, args VarList) {
	m.clearBy(k, self, func(o *GameObject) bool {
		return o.GetInt("OffLineClear") != 0
	})
}

// onPetDestroy 官方 @0x34E2B0：宠物销毁 → 其 Buffer 全清。
func (m *BufferModule) onPetDestroy(k *Kernel, self, from PersistID, args VarList) {
	m.RemoveAll(k, self)
}

// onBufferLoad/onBufferRecover 官方 @0x34D540/@0x34D730：Buffer 对象回载。
func (m *BufferModule) onBufferLoad(k *Kernel, self, from PersistID, args VarList)     {}
func (m *BufferModule) onBufferRecover(k *Kernel, self, from PersistID, args VarList) {}

// onBufferContainerCreate/AfterAdd/Remove 官方 @0x34D910/@0x34D970/@0x34DA40。
func (m *BufferModule) onBufferContainerCreate(k *Kernel, self, from PersistID, args VarList) {}
func (m *BufferModule) onBufferContainerAfterAdd(k *Kernel, self, from PersistID, args VarList) {
}
func (m *BufferModule) onBufferContainerRemove(k *Kernel, self, from PersistID, args VarList) {}

// onObjectSelect 官方 @0x34E050：选中（Buffer 可视面下发）。
func (m *BufferModule) onObjectSelect(k *Kernel, self, from PersistID, args VarList) {}

// onCommandBeKill 官方 @0x34E610（命令 616）：击杀（DeadClear 面清除）。
func (m *BufferModule) onCommandBeKill(k *Kernel, self, from PersistID, args VarList) {}

// onBufferCmd 官方 @0x34E8B0：Buffer 表现族命令（600/605/609-613/615/630-632/641）。
func (m *BufferModule) onBufferCmd(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("BufferModule: buffer cmd %v", args.Int(0))
}

// onCustomAddBuffer 官方 @0x34F060（cust 1019）：加 Buffer（GM/调试面；
// args: 配置ID, 等级, 时长ms）。
func (m *BufferModule) onCustomAddBuffer(k *Kernel, self, from PersistID, args VarList) {
	m.AddBuffer(k, self, args.String(0), args.Int(1), int64(args.Int(2)))
}

// onCustomRemoveBuffer 官方 @0x34EE30（cust 1021）：移除
//（args: 配置ID；IsDamage/CantCustomRemove 校验）。
func (m *BufferModule) onCustomRemoveBuffer(k *Kernel, self, from PersistID, args VarList) {
	if id := m.FindBuffer(k, self, args.String(0)); id.Valid() {
		m.RemoveBuffer(k, self, id)
	}
}

// AddBuffer nx_add_buffer（官方 @0x348270）的 Go 等价：
// 建 Buffer 对象 → 容器挂载（叠加：同 ConfigID 按 MaxOverlayNumber/OverlayType
// 计数——C_OnBufferCount 临界 + OverLay 记录面）。返回 Buffer 句柄。
func (m *BufferModule) AddBuffer(k *Kernel, target PersistID, configID string, level int32, lifeMS int64) PersistID {
	o := k.GetGameObj(target)
	if o == nil {
		return NullPersistID
	}
	// 叠加：同级同配置刷新时长（OverlayType 官方细分；Go 面取刷新语义）。
	for _, bid := range m.buffers[target] {
		if bo := k.GetGameObj(bid); bo != nil && bo.GetString("ConfigID") == configID {
			if lifeMS > 0 {
				_ = k.SetProp(bid, "LifeTime", lifeMS)
			}
			return bid
		}
	}
	id, err := k.CreateObject("Buffer")
	if err != nil {
		return NullPersistID
	}
	bo := k.GetGameObj(id)
	_ = bo.Set("ConfigID", configID)
	_ = bo.Set("Level", level)
	_ = bo.Set("UniqueID", fmt.Sprintf("BUF-%d", m.nextUID))
	m.nextUID++
	if _, ok := bo.Class().Prop("LifeTime"); ok {
		_ = bo.Set("LifeTime", lifeMS)
	}
	m.buffers[target] = append(m.buffers[target], id)
	m.added = append(m.added, configID)
	k.Tracef("BufferModule: +buffer %s → %v (life=%dms)", configID, target.ID(), lifeMS)
	return id
}

// RemoveBuffer nx_remove_buffer（官方 @0x348520）：按句柄移除
//（CantClear=1 拒）。
func (m *BufferModule) RemoveBuffer(k *Kernel, target, buffer PersistID) bool {
	bo := k.GetGameObj(buffer)
	if bo == nil {
		return false
	}
	if bo.GetInt("CantClear") != 0 {
		return false
	}
	list := m.buffers[target]
	for i, bid := range list {
		if bid == buffer {
			m.buffers[target] = append(list[:i:i], list[i+1:]...)
			m.removed = append(m.removed, bo.GetString("ConfigID"))
			k.RemoveObject(buffer)
			return true
		}
	}
	return false
}

// FindBuffer nx_find_buffer（官方 @0x348BE0）：按配置ID取句柄。
func (m *BufferModule) FindBuffer(k *Kernel, target PersistID, configID string) PersistID {
	for _, bid := range m.buffers[target] {
		if bo := k.GetGameObj(bid); bo != nil && bo.GetString("ConfigID") == configID {
			return bid
		}
	}
	return NullPersistID
}

// RemoveAll 全清（宠物销毁面）。
func (m *BufferModule) RemoveAll(k *Kernel, target PersistID) {
	for _, bid := range m.buffers[target] {
		if bo := k.GetGameObj(bid); bo != nil {
			m.removed = append(m.removed, bo.GetString("ConfigID"))
			k.RemoveObject(bid)
		}
	}
	delete(m.buffers, target)
}

// clearBy 条件清除（OnStore OffLineClear / DeadClear 面）。
func (m *BufferModule) clearBy(k *Kernel, target PersistID, keep func(*GameObject) bool) {
	var rest []PersistID
	for _, bid := range m.buffers[target] {
		bo := k.GetGameObj(bid)
		if bo != nil && keep(bo) {
			m.removed = append(m.removed, bo.GetString("ConfigID"))
			k.RemoveObject(bid)
			continue
		}
		rest = append(rest, bid)
	}
	m.buffers[target] = rest
}

// BuffersOf 目标挂载的 Buffer 句柄序（测试面）。
func (m *BufferModule) BuffersOf(target PersistID) []PersistID { return m.buffers[target] }

// H_BufferUpdate 官方 @0x34F410：Buffer 泵——Timer 按 beatMS 推进，
// LifeType 计时至 LifeTime 归零移除（LifeTime=0 持久）；EffectTimer 周期
// 效果面（EffectType 分轨）。官方列面：MainChar/MainBuffer/HaloShapeType/
// TargetShapePara1/2/Timer/EffectType/EffectTimer/LifeTimeType/LifeTime/
// Counter/AddTime。
func (m *BufferModule) hbBufferUpdate(k *Kernel, self PersistID, beats int32) {
	for _, bid := range append([]PersistID(nil), m.buffers[self]...) {
		bo := k.GetGameObj(bid)
		if bo == nil {
			continue
		}
		life := varToInt64(bo.Get("LifeTime"))
		if life <= 0 {
			continue // 持久 Buffer
		}
		timer := varToInt64(bo.Get("Timer")) + m.beatMS
		_ = k.SetProp(bid, "Timer", timer)
		if timer >= life {
			m.RemoveBuffer(k, self, bid)
		}
	}
}

// C_OnLogicStateChanged 官方 @0x359450：逻辑态变更（DeadClear 面清除）。
func (m *BufferModule) onLogicStateChanged(k *Kernel, self PersistID, prop string, old Var) {
	if k.GetGameObj(self).GetInt("LogicState") != 0 {
		m.clearBy(k, self, func(o *GameObject) bool { return o.GetInt("DeadClear") != 0 })
	}
}

// C_OnVarPropNoChanged 官方 @0x34FF20。
func (m *BufferModule) onVarPropNoChanged(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnBufferActive 官方 @0x350100：Buffer 激活切换。
func (m *BufferModule) onBufferActive(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnBufferCount 官方 @0x350240：叠加计数变更。
func (m *BufferModule) onBufferCount(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnOverLayRecordChanged 官方 @0x350330（OverLay 记录钩子）。
func (m *BufferModule) onOverLayRecordChanged(k *Kernel, self PersistID, record string, op, row int32) {}

// Added/Removed 挂载/移除流水（测试面）。
func (m *BufferModule) Added() []string   { return m.added }
func (m *BufferModule) Removed() []string { return m.removed }

// ---- SkillModule ----

// SkillModule 复刻官方 skillmodule\skillmodule（Init @0x382620 /
// LoadResource @0x387310，玩法表 Init 序 68；符号 582，34 心跳）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("player","OnReady",@0x385A70) / ("player","OnRecover",
//	    @0x385050)
//	AddEventCallback ×7（AttackNpc/BattleCarrier/SkillNpc/MultiSkillNpc/
//	    OrganNpc/Pet/SkillEffectNpc，"OnLoad"，@0x385050）
//	AddClassCallback ("Skill","OnCreateClass",@0x385E30) / ("SkillContainer",
//	    "OnCreateClass",@0x385E80)
//	AddEventCallback ("Skill","OnRecover"/"OnLoad"/"OnCreate"，@0x385F10/
//	    @0x385FA0/@0x385EE0)
//	AddIntCommandHook ("player",856,@0x386030) OnCommandSkillLearn /
//	    ("player",870,@0x3860D0) OnCommandSkillForget
//	AddIntCustomHook ("player",211,@0x3861B0) OnCustomUseSkill /
//	    ("player",232,@0x386900) OnCustomCoolDownSkill / ("player",215,
//	    @0x386980) OnCustomGetReplaceSkill / ("player",216,@0x386BA0)
//	    OnCustomInterruptCurrentSkill
//	DeclareCritical ×5：C_OnPropChanged@0x3872C0 / C_OnCurSkillChanged@
//	    0x387AE0 / C_OnCantUseSkillChanged@0x387B30 / C_OnDeadChanged@
//	    0x3883B0 / C_OnCantAttackChanged@0x387CA0
//	DeclareHeartBeat ×17：H_EFFECT1-8（@0x3767A0-0x3769D0 技能效果八通道）/
//	    H_QuequeSkill@0x397600 / H_CantMove@0x3772A0 / H_LockAction@0x3773C0 /
//	    H_HitOutStateClear@0x38AD70 / H_CantRotate@0x377300 / H_CantJump@
//	    0x377360 / H_DelayRemoveBuff@0x387D10 / H_DelayFinish@0x387EA0 /
//	    H_DelayUse@0x3965B0
//
// LoadResource @0x387310：sprintf("ini/skill/skill_new_%d.xml",
// GetMemberId()) → PreloadConfig + ini/Skill/Skill_Normal_Attack.xml。
type SkillModule struct {
	BaseModule

	// skillTable skill_new 表：ConfigID → 行（Name/script/Skill_Main/
	// Skill_Damage/Skill_Effect/Skill_Display）。
	skillTable map[string]map[string]string
	// normalAttack 普攻映射：job_sex → 行（skill_0h/1hR/1hL/…）。
	normalAttack map[string]map[string]string
	// skills 玩家技能表：player → Skill 对象句柄序（SkillContainer 镜像）。
	skills map[PersistID][]PersistID
	// delays 延迟技队列（H_DelayUse 面：DelaySkill 属性族）。
	delays []delayUse
	// nextUUID 技能实例 UUID 序（CurSkillUUID 面）。
	nextUUID int64
	// ops 命令流水（测试面）。
	ops []string
}

type delayUse struct {
	self   PersistID
	skill  PersistID
	dueBeat int32
}

// Name 模块名。
func (m *SkillModule) Name() string { return "SkillModule" }

// Init 注册期（对齐官方 0x382620）。
func (m *SkillModule) Init(k *Kernel) error {
	m.skillTable = make(map[string]map[string]string)
	m.normalAttack = make(map[string]map[string]string)
	m.skills = make(map[PersistID][]PersistID)
	b := k.Bus()
	b.OnEvent("player", "OnReady", m.onReady)
	b.OnEvent("player", "OnRecover", m.onLoad)
	for _, cls := range []string{"AttackNpc", "BattleCarrier", "SkillNpc", "MultiSkillNpc", "OrganNpc", "Pet", "SkillEffectNpc"} {
		b.OnEvent(cls, "OnLoad", m.onLoad)
	}
	b.OnClassEvent("Skill", "OnCreateClass", m.onSkillCreateClass)
	b.OnEvent("Skill", "OnCreate", m.onSkillCreate)
	b.OnEvent("Skill", "OnLoad", m.onSkillLoad)
	b.OnEvent("Skill", "OnRecover", m.onSkillRecover)
	b.OnClassEvent("SkillContainer", "OnCreateClass", m.onSkillContainerCreateClass)
	b.OnIntCommand("player", 856, m.onCommandSkillLearn)
	b.OnIntCommand("player", 870, m.onCommandSkillForget)
	b.OnCommand("player", 211, m.onCustomUseSkill)
	b.OnCommand("player", 232, m.onCustomCoolDownSkill)
	b.OnCommand("player", 215, m.onCustomGetReplaceSkill)
	b.OnCommand("player", 216, m.onCustomInterruptCurrentSkill)
	b.Critical("player", "Prop", m.onPropChanged)
	b.Critical("player", "CurSkill", m.onCurSkillChanged)
	b.Critical("player", "CantUseSkill", m.onCantUseSkillChanged)
	b.Critical("player", "Dead", m.onDeadChanged)
	b.Critical("player", "CantAttack", m.onCantAttackChanged)
	for i := 1; i <= 8; i++ {
		i := i
		b.HeartBeat(fmt.Sprintf("SkillModule::H_EFFECT%d", i), func(k *Kernel, self PersistID, beats int32) {})
	}
	b.HeartBeat("SkillModule::H_QuequeSkill", m.hbQuequeSkill)
	b.HeartBeat("SkillModule::H_CantMove", func(k *Kernel, self PersistID, beats int32) {})
	b.HeartBeat("SkillModule::H_LockAction", func(k *Kernel, self PersistID, beats int32) {})
	b.HeartBeat("SkillModule::H_HitOutStateClear", func(k *Kernel, self PersistID, beats int32) {})
	b.HeartBeat("SkillModule::H_CantRotate", func(k *Kernel, self PersistID, beats int32) {})
	b.HeartBeat("SkillModule::H_CantJump", func(k *Kernel, self PersistID, beats int32) {})
	b.HeartBeat("SkillModule::H_DelayRemoveBuff", m.hbDelayRemoveBuff)
	b.HeartBeat("SkillModule::H_DelayFinish", m.hbDelayFinish)
	b.HeartBeat("SkillModule::H_DelayUse", m.hbDelayUse)
	return nil
}

// LoadResource 官方 @0x387310：sprintf("ini/skill/skill_new_%d.xml",
// IKernel::GetMemberId) → PreloadConfig；另载 ini/Skill/Skill_Normal_Attack.xml
//（普通攻击表）。缺失容忍（官方逐档部署）。
func (m *SkillModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: SkillModule.LoadResource: ResourcePath 未设置")
	}
	p := fmt.Sprintf("ini/skill/skill_new_%d.xml", k.MemberID)
	loaded := false
	for _, path := range []string{p, "ini/skill/skill_new.xml.xml"} {
		data, err := readResource(k.ResourcePath, path)
		if err != nil {
			continue
		}
		loaded = true
		if rows, err := ParsePropTable(data); err == nil {
			for id, row := range ruleByID(rows) {
				m.skillTable[id] = row
			}
		}
		break
	}
	if !loaded {
		k.Tracef("SkillModule: 技能表缺档（官方逐 member 部署）")
	}
	if data, err := readResource(k.ResourcePath, "ini/Skill/Skill_Normal_Attack.xml"); err == nil {
		if rows, err := ParsePropTable(data); err == nil {
			for _, row := range rows {
				if job := row["job_sex"]; job != "" {
					m.normalAttack[job] = row
				}
			}
		}
	}
	k.Tracef("SkillModule: LoadResource 技能表 %d 项 + 普攻映射 %d 项",
		len(m.skillTable), len(m.normalAttack))
	return nil
}

// SkillCfg 查技能配置行（测试面）。
func (m *SkillModule) SkillCfg(configID string) map[string]string { return m.skillTable[configID] }

// NormalAttackCfg 查普攻映射行（job_sex 键；测试面）。
func (m *SkillModule) NormalAttackCfg(jobSex string) map[string]string { return m.normalAttack[jobSex] }

// onReady 官方 @0x385A70：就绪（技能表同步下发）。
func (m *SkillModule) onReady(k *Kernel, self, from PersistID, args VarList) {
	k.Tracef("SkillModule: OnReady player=%v skills=%d", self.ID(), len(m.skills[self]))
}

// onLoad 官方 @0x385050：战斗体回载（技能面初始化）。
func (m *SkillModule) onLoad(k *Kernel, self, from PersistID, args VarList) {}

// onSkillCreateClass 官方 @0x385E30：Skill 类 schema 就绪
//（struct/skill/skillbase.xml + fight includes）。
func (m *SkillModule) onSkillCreateClass(k *Kernel, class string) {
	k.Tracef("SkillModule: OnCreateClass Skill (skillbase.xml)")
}

// onSkillContainerCreateClass 官方 @0x385E80。
func (m *SkillModule) onSkillContainerCreateClass(k *Kernel, class string) {}

// onSkillCreate 官方 @0x385EE0：技能对象物化（ConfigID/CanUse 初值）。
func (m *SkillModule) onSkillCreate(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if _, ok := o.Class().Prop("CanUse"); ok && o.GetInt("CanUse") == 0 {
		_ = o.Set("CanUse", int32(1))
	}
}

// onSkillLoad/onSkillRecover 官方 @0x385FA0/@0x385F10。
func (m *SkillModule) onSkillLoad(k *Kernel, self, from PersistID, args VarList)    {}
func (m *SkillModule) onSkillRecover(k *Kernel, self, from PersistID, args VarList) {}

// onCommandSkillLearn 官方 @0x386030（命令 856=0x358）：学技
//（args: ConfigID）→ 建 Skill 对象入 SkillContainer。
func (m *SkillModule) onCommandSkillLearn(k *Kernel, self, from PersistID, args VarList) {
	m.Learn(k, self, args.String(0))
}

// onCommandSkillForget 官方 @0x3860D0（命令 870=0x366）：忘技。
func (m *SkillModule) onCommandSkillForget(k *Kernel, self, from PersistID, args VarList) {
	m.Forget(k, self, args.String(0))
}

// Learn 学技：按配置行建 Skill 对象（ConfigID/Level/CoolDownTime 面）。
func (m *SkillModule) Learn(k *Kernel, self PersistID, configID string) PersistID {
	if configID == "" {
		return NullPersistID
	}
	id, err := k.CreateObject("Skill")
	if err != nil {
		return NullPersistID
	}
	o := k.GetGameObj(id)
	_ = o.Set("ConfigID", configID)
	if _, ok := o.Class().Prop("Level"); ok {
		_ = o.Set("Level", int32(1))
	}
	m.skills[self] = append(m.skills[self], id)
	m.ops = append(m.ops, "learn:"+configID)
	k.Tracef("SkillModule: learn %s player=%v", configID, self.ID())
	return id
}

// Forget 忘技：摘除技能对象。
func (m *SkillModule) Forget(k *Kernel, self PersistID, configID string) bool {
	for i, sid := range m.skills[self] {
		if so := k.GetGameObj(sid); so != nil && so.GetString("ConfigID") == configID {
			m.skills[self] = append(m.skills[self][:i:i], m.skills[self][i+1:]...)
			k.RemoveObject(sid)
			m.ops = append(m.ops, "forget:"+configID)
			return true
		}
	}
	return false
}

// SkillsOf 玩家技能句柄序（测试面）。
func (m *SkillModule) SkillsOf(self PersistID) []PersistID { return m.skills[self] }

// onCustomUseSkill 官方 @0x3861B0（cust 211=0xD3）：出招主入口。
// args: 技能句柄/配置ID, 目标句柄[, x, y, z]。校验链（官方串定案）：
// recved invalid msg → CantUseSkill → Dead → 冷却（CoolDownModule）→
// 消耗（AConsumeCurMP 族）→ 结算（DamageCalculate→FightModule::NxDamage）→
// 开冷却（CoolDownTime ms）→ CurSkill/CurSkillID/CurSkillTarget 写入
//（C_OnCurSkillChanged 临界）。
func (m *SkillModule) onCustomUseSkill(k *Kernel, self, from PersistID, args VarList) {
	var skill PersistID
	if s := args.Object(0); s.Valid() {
		skill = s
	} else if cfg := args.String(0); cfg != "" {
		skill = m.findSkill(k, self, cfg)
	}
	target := args.Object(1)
	m.UseSkill(k, self, skill, target)
}

// findSkill 按配置ID取玩家技能对象。
func (m *SkillModule) findSkill(k *Kernel, self PersistID, configID string) PersistID {
	for _, sid := range m.skills[self] {
		if so := k.GetGameObj(sid); so != nil && so.GetString("ConfigID") == configID {
			return sid
		}
	}
	return NullPersistID
}

// UseSkill 出招闭环（官方流定案；返回是否成功）。
func (m *SkillModule) UseSkill(k *Kernel, self, skill, target PersistID) bool {
	o := k.GetGameObj(self)
	so := k.GetGameObj(skill)
	if o == nil || so == nil {
		m.ops = append(m.ops, "use:invalid")
		return false // use skill error: recved invalid msg!
	}
	if o.GetInt("CantUseSkill") != 0 {
		m.ops = append(m.ops, "use:cant_use")
		return false // C_OnCantUseSkillChanged 面
	}
	if o.GetInt("Dead") != 0 {
		m.ops = append(m.ops, "use:dead")
		return false
	}
	// 冷却检查（CoolDownModule；分类=技能 QTypeID/ConfigID 哈希面）。
	cd, _ := k.GetLogicModule("CoolDownModule").(*CoolDownModule)
	category := uint32(so.GetInt("QTypeID"))
	if cd != nil && cd.NxIsCooldown(k, self, category) {
		m.ops = append(m.ops, "use:cooldown")
		return false
	}
	// 消耗：AConsumeCurMP（consume.xml 面；不足拒）。
	need := so.GetInt("AConsumeCurMP")
	if mp := o.GetInt("MP"); mp < need {
		m.ops = append(m.ops, "use:no_mp")
		return false
	}
	if need > 0 {
		_ = k.SetInt(self, "MP", o.GetInt("MP")-need)
	}
	// 当前技能面写入（CurSkill 族临界联动）。
	_ = k.SetProp(self, "CurSkill", skill)
	_ = k.SetString(self, "CurSkillID", so.GetString("ConfigID"))
	if target.Valid() {
		_ = k.SetProp(self, "CurSkillTarget", target)
	}
	// 伤害结算：FightModule::NxDamage（DamageCalculate 缝合）。
	if target.Valid() {
		if fm, ok := k.GetLogicModule("FightModule").(*FightModule); ok && fm != nil {
			// 伤害基数：对象属性缺档时取 skill_new 配置行（Skill_Damage 列；
			// 官方经 StaticDataQuery 公式包求值——Go 缝合为直接基数）。
			base := float64(so.GetInt("Skill_Damage"))
			if base == 0 {
				if cfg := m.skillTable[so.GetString("ConfigID")]; cfg != nil {
					base = float64(atoi32(cfg["Skill_Damage"]))
				}
			}
			fm.NxDamage(k, self, target, DamagePhysical, base)
		}
	}
	// 开冷却：CoolDownTime ms（cooldown_new.xml 面）。
	if cd != nil {
		cdMS := int64(so.GetInt("CoolDownTime")) + int64(so.GetInt("AddCoolDownTime"))
		if cdMS == 0 {
			if cfg := m.skillTable[so.GetString("ConfigID")]; cfg != nil {
				cdMS = int64(atoi32(cfg["CoolDownTime"]))
			}
		}
		if cdMS > 0 {
			now := unixNowMilli()
			cd.NxBeginCooldown(k, self, category, now, now+cdMS, category)
		}
	}
	m.ops = append(m.ops, "use:"+so.GetString("ConfigID"))
	k.Tracef("SkillModule: use %s player=%v→%v", so.GetString("ConfigID"), self.ID(), target.ID())
	return true
}

// onCustomCoolDownSkill 官方 @0x386900（cust 232=0xE8）：冷却查询下发。
func (m *SkillModule) onCustomCoolDownSkill(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "cooldown_query")
}

// onCustomGetReplaceSkill 官方 @0x386980（cust 215=0xD7）：替换技查询。
func (m *SkillModule) onCustomGetReplaceSkill(k *Kernel, self, from PersistID, args VarList) {
	m.ops = append(m.ops, "replace_query")
}

// onCustomInterruptCurrentSkill 官方 @0x386BA0（cust 216=0xD8）：
// 打断当前技（CurSkill 清零 → C_OnCurSkillChanged）。
func (m *SkillModule) onCustomInterruptCurrentSkill(k *Kernel, self, from PersistID, args VarList) {
	_ = k.SetProp(self, "CurSkill", NullPersistID)
	_ = k.SetString(self, "CurSkillID", "")
	m.ops = append(m.ops, "interrupt")
}

// C_OnPropChanged 官方 @0x3872C0：属性变更 → 技能面重算
//（MoveSpeed/RunSpeed 冲刺技联动）。
func (m *SkillModule) onPropChanged(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnCurSkillChanged 官方 @0x387AE0：当前技切换（连招/队列面）。
func (m *SkillModule) onCurSkillChanged(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnCantUseSkillChanged 官方 @0x387B30：禁技状态切换。
func (m *SkillModule) onCantUseSkillChanged(k *Kernel, self PersistID, prop string, old Var) {}

// C_OnDeadChanged 官方 @0x3883B0：死亡切换（技能中断 + DelayRemoveBuff 面）。
func (m *SkillModule) onDeadChanged(k *Kernel, self PersistID, prop string, old Var) {
	o := k.GetGameObj(self)
	if o != nil && o.GetInt("Dead") != 0 {
		_ = k.SetProp(self, "CurSkill", NullPersistID)
		_ = k.SetString(self, "CurSkillID", "")
		m.ops = append(m.ops, "dead_break")
	}
}

// C_OnCantAttackChanged 官方 @0x387CA0：禁攻切换（H_EFFECT 通道暂停面）。
func (m *SkillModule) onCantAttackChanged(k *Kernel, self PersistID, prop string, old Var) {}

// hbQuequeSkill 官方 @0x397600：技能队列泵（连招按序释放）。
func (m *SkillModule) hbQuequeSkill(k *Kernel, self PersistID, beats int32) {}

// hbDelayUse 官方 @0x3965B0：延迟技泵（DelaySkill/DelaySkillHandType/
// DelaySkillX/Y/Z/Type/Time 属性族 → 到点释放）。
func (m *SkillModule) hbDelayUse(k *Kernel, self PersistID, beats int32) {
	var fire []delayUse
	rest := m.delays[:0]
	for _, d := range m.delays {
		if d.self != self {
			rest = append(rest, d)
			continue
		}
		if beats >= d.dueBeat {
			fire = append(fire, d)
		} else {
			rest = append(rest, d)
		}
	}
	m.delays = rest
	for _, d := range fire {
		m.UseSkill(k, self, d.skill, NullPersistID)
	}
}

// ScheduleDelayUse 延迟出招登记（DelaySkill 面）。
func (m *SkillModule) ScheduleDelayUse(self, skill PersistID, dueBeat int32) {
	m.delays = append(m.delays, delayUse{self: self, skill: skill, dueBeat: dueBeat})
}

// hbDelayRemoveBuff 官方 @0x387D10：延迟移除 Buffer 泵。
func (m *SkillModule) hbDelayRemoveBuff(k *Kernel, self PersistID, beats int32) {}

// hbDelayFinish 官方 @0x387EA0：延迟收招泵（FinishAction 面）。
func (m *SkillModule) hbDelayFinish(k *Kernel, self PersistID, beats int32) {}

// Ops 命令流水（测试面）。
func (m *SkillModule) Ops() []string { return m.ops }

// unixNowMilli 毫秒时钟（冷却/Buffer 计时面）。
func unixNowMilli() int64 { return unixNow() * 1000 }
