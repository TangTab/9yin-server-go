package fsgame

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 八期：NPC 载体域底座六模块（docs/official/FsGame全解与Go复刻.md §8
// NPC 玩法载体）。官方源码树：fsgame\npcbasemodule\。

// ---- NpcResourceModule ----

// NpcResourceModule 复刻官方 npcresourcemodule（Init @0x8A3F60 /
// LoadResource @0x8A3FD0，玩法表 Init 序 225；符号 106，19 事件经
// OnCreateClass 动态挂载）。NPC 资源预载与物化。
//
// LoadResource @0x8A3FD0：ini/Npc/ 树（NpcConfig/ + NpcConfigAuto/ 按
// member<N>_*.xml 分档预载——Path 驱动）。
type NpcResourceModule struct {
	BaseModule

	// npcs NPC 资源表：npcID → 类名（NpcConfig 物化面）。
	npcs map[string]string
	// loaded 预载文件数。
	loaded int
}

// Name 模块名。
func (m *NpcResourceModule) Name() string { return "NpcResourceModule" }

// Init 注册期（对齐官方 0x8A3F60——OnCreateClass 动态挂载面）。
func (m *NpcResourceModule) Init(k *Kernel) error {
	m.npcs = make(map[string]string)
	return nil
}

// LoadResource 官方 @0x8A3FD0：ini/Npc 树扫描（NpcConfigAuto 按
// member<N>_ 分档优先）。官方 NpcConfig 本就按组子目录分档
// （WorldNpc/BattleNpc/CloneNpc/LifeNpc，见客户端 npcconfig/npc_type.ini
// 的服务端路径映射），Go 面递归装载并按行登记 npcID → script 类。
func (m *NpcResourceModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: NpcResourceModule.LoadResource: ResourcePath 未设置")
	}
	base := filepath.Join(k.ResourcePath, "ini", "Npc")
	for _, sub := range []string{"NpcConfigAuto", "NpcConfig"} {
		dir := filepath.Join(base, sub)
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil // 目录缺失与旧容错一致
				}
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".xml") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				k.Tracef("NpcResourceModule: 读 %s: %v", path, readErr)
				return nil
			}
			rows, parseErr := ParsePropTable(data)
			if parseErr != nil {
				k.Tracef("NpcResourceModule: 解析 %s: %v", path, parseErr)
				return nil
			}
			for _, row := range rows {
				id := row["ID"]
				if id == "" {
					continue
				}
				m.npcs[id] = m.resolveClass(k, row["script"])
			}
			m.loaded++
			return nil
		})
		if err != nil {
			return fmt.Errorf("fsgame: NpcResourceModule.LoadResource: %w", err)
		}
	}
	k.Tracef("NpcResourceModule: LoadResource ini/Npc 树 %d 档 %d NPC", m.loaded, len(m.npcs))
	return nil
}

// resolveClass script 属性 → 物化类名：客户端表 script 值即对象类型；
// 空值或 logic_class 未登记的类名回落 CommonNpc。
func (m *NpcResourceModule) resolveClass(k *Kernel, script string) string {
	if script != "" && k.Class(script) != nil {
		return script
	}
	return "CommonNpc"
}

// ClassOf 查询 NPC 配置登记类（测试面；未登记返回空）。
func (m *NpcResourceModule) ClassOf(npcID string) string { return m.npcs[npcID] }

// RegisterNpc 登记 NPC 资源（npcID → 类）。
func (m *NpcResourceModule) RegisterNpc(npcID, class string) { m.npcs[npcID] = class }

// Spawn 物化 NPC（官方 CreateFromConfig 面）：按资源表建对象。
func (m *NpcResourceModule) Spawn(k *Kernel, npcID string) PersistID {
	class := m.npcs[npcID]
	if class == "" {
		class = "CommonNpc"
	}
	id, err := k.CreateObject(class)
	if err != nil {
		return NullPersistID
	}
	if o := k.GetGameObj(id); o != nil {
		if _, ok := o.Class().Prop("ConfigID"); ok {
			_ = o.Set("ConfigID", npcID)
		}
	}
	return id
}

// Loaded 预载计数（测试面）。
func (m *NpcResourceModule) Loaded() int { return m.loaded }

// ---- VisibleNpcModule ----

// VisibleNpcModule 复刻官方 visiblenpcmodule（Init @0x8B3AD0，玩法表
// Init 序 171；符号 139）——可视 NPC 视野面。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	AddEventCallback ("VisibleNpc","OnLoad",@0x8B3C80) / ("VisibleNpc",
//	    "OnEntry",@0x8B3D50) / ("VisibleNpc","OnLeave",@0x8B41F0)
//	AddEventCallback ("player","OnEntry",@0x8B4460 OnPlayerEntry——玩家
//	    入场视野挂载) / ("player","OnLeave",@0x8B4690)
//	AddIntCommandHook ("VisibleNpc",614,@0x8B4820 OnCanBeKilledCommand)
//	AddIntCustomHook ("player",990=0x3DE,@0x8B4AC0 OnCustomNpcMove)
//	DeclareCritical ×2：C_OnActionChanged@0x8B48F0 /
//	    C_OnCollideTypeChanged@0x8B49E0
//	DeclareLuaExt ("nx_query_preload_npc",@0x8B35D0)
//
// VisibleNpc schema（struct/npc/VisibleNpc.xml）：Level/CollideType/
// FightID/Type/Resource/Scale/ConfigID…
type VisibleNpcModule struct {
	BaseModule

	// viewers 视野挂载流水（player entry/leave 观测）。
	viewers []string
}

// Name 模块名。
func (m *VisibleNpcModule) Name() string { return "VisibleNpcModule" }

// Init 注册期（对齐官方 0x8B3AD0）。
func (m *VisibleNpcModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("VisibleNpc", "OnLoad", m.onNpcLoad)
	b.OnEvent("VisibleNpc", "OnEntry", m.onNpcEntry)
	b.OnEvent("VisibleNpc", "OnLeave", m.onNpcLeave)
	b.OnEvent("player", "OnEntry", m.onPlayerEntry)
	b.OnEvent("player", "OnLeave", m.onPlayerLeave)
	b.OnIntCommand("VisibleNpc", 614, m.onCanBeKilledCommand)
	b.OnCommand("player", 990, m.onCustomNpcMove)
	b.Critical("VisibleNpc", "Action", m.onActionChanged)
	b.Critical("VisibleNpc", "CollideType", m.onCollideTypeChanged)
	return nil
}

// onNpcLoad/onNpcEntry/onNpcLeave 官方 @0x8B3C80/@0x8B3D50/@0x8B41F0。
func (m *VisibleNpcModule) onNpcLoad(k *Kernel, self, from PersistID, args VarList)  {}
func (m *VisibleNpcModule) onNpcEntry(k *Kernel, self, from PersistID, args VarList) {}
func (m *VisibleNpcModule) onNpcLeave(k *Kernel, self, from PersistID, args VarList) {}

// onPlayerEntry 官方 @0x8B4460：玩家入场（AOI 视野内 NPC 下发）。
func (m *VisibleNpcModule) onPlayerEntry(k *Kernel, self, from PersistID, args VarList) {
	m.viewers = append(m.viewers, fmt.Sprintf("entry:%v", self.ID()))
}

// onPlayerLeave 官方 @0x8B4690。
func (m *VisibleNpcModule) onPlayerLeave(k *Kernel, self, from PersistID, args VarList) {
	m.viewers = append(m.viewers, fmt.Sprintf("leave:%v", self.ID()))
}

// onCanBeKilledCommand 官方 @0x8B4820（命令 614=0x266）：可击杀查询。
func (m *VisibleNpcModule) onCanBeKilledCommand(k *Kernel, self, from PersistID, args VarList) {}

// onCustomNpcMove 官方 @0x8B4AC0（cust 990=0x3DE）：NPC 移动（巡逻路径）。
func (m *VisibleNpcModule) onCustomNpcMove(k *Kernel, self, from PersistID, args VarList) {}

// C_OnActionChanged/C_OnCollideTypeChanged 官方 @0x8B48F0/@0x8B49E0。
func (m *VisibleNpcModule) onActionChanged(k *Kernel, self PersistID, prop string, old Var)     {}
func (m *VisibleNpcModule) onCollideTypeChanged(k *Kernel, self PersistID, prop string, old Var) {}

// NxQueryPreloadNpc nx_query_preload_npc 的 Go 等价。
func (m *VisibleNpcModule) NxQueryPreloadNpc(npcID string) bool { return npcID != "" }

// Viewers 视野流水（测试面）。
func (m *VisibleNpcModule) Viewers() []string { return m.viewers }

// ---- CommonNpcModule ----

// CommonNpcModule 复刻官方 commonnpcmodule（Init @0x8B56C0，玩法表
// Init 序 169；符号 23）——通用 NPC 交互枢纽。
//
// 官方 Init 注册面：
//
//	AddEventCallback ("CommonNpc","OnSelect",@0x8B5B40 OnNpcSelect——
//	    对话交互分发：Quest/Shop/Post/Team/Strengthen 各域监听同事件）
//	AddIntCommandHook ("player",22,@0x8B5C70 OnMenuSelect——对话菜单）
type CommonNpcModule struct {
	BaseModule

	// selects 交互流水（测试面）。
	selects []string
}

// Name 模块名。
func (m *CommonNpcModule) Name() string { return "CommonNpcModule" }

// Init 注册期（对齐官方 0x8B56C0）。
func (m *CommonNpcModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("CommonNpc", "OnSelect", m.onNpcSelect)
	b.OnIntCommand("player", 22, m.onMenuSelect)
	return nil
}

// onNpcSelect 官方 @0x8B5B40：NPC 选中（self=NPC、from=玩家）——多播
// 分发面（各域模块监听 CommonNpc OnSelect 协作）。
func (m *CommonNpcModule) onNpcSelect(k *Kernel, self, from PersistID, args VarList) {
	m.selects = append(m.selects, args.String(0))
	k.Tracef("CommonNpcModule: OnSelect %s", args.String(0))
}

// onMenuSelect 官方 @0x8B5C70（命令 22=0x16）。
func (m *CommonNpcModule) onMenuSelect(k *Kernel, self, from PersistID, args VarList) {}

// Selects 交互流水（测试面）。
func (m *CommonNpcModule) Selects() []string { return m.selects }

// ---- GivingNpcModule ----

// GivingNpcModule 复刻官方 givingnpcmodule（Init @0x8DAD80，玩法表
// Init 序 163；符号 579）——献祭/给予 NPC（采集型目标物）。
//
// 官方 Init 注册面（p2scan 反汇编定案）：
//
//	DeclareHeartBeat ("GivingNpcModule::HB_ReBorn",@0x8DE7B0——串定案：
//	    Invisible/DestroyTime1/LifeTime/CreatorItem/BornOrigin/
//	    RefreshTime/PeaceAction/PathID 复活面)
//	AddEventCallback ("GivingNpc","OnSpring",@0x8DB0D0——弹出校验串：
//	    SpringRange/SpringType/NpcType/BombEntryPos) / ("GivingNpc",
//	    "OnSelect",@0x8DB1E0——MaxHP/HPRatio 联动) / ("GivingNpc",
//	    "OnEntry",@0x8DB380) / ("GivingNpc","OnLoad",@0x8DB8C0)
//	AddIntCommandHook ("player",600,@0x8DB8F0 OnSkillBegin) / ("GivingNpc",
//	    603,@0x8DBA60 OnSkillFinish) / ("player",602,@0x8DBFF0
//	    OnSkillBreaked) / ("GivingNpc",852,@0x8DE850 OnPickupNpc)
//	DeclareCritical ("GivingNpcModule::C_OnHPRatioChanged",@0x8DB750)
type GivingNpcModule struct {
	BaseModule

	// states 目标物状态：npc → {HP, Done, RebornAt}。
	states map[PersistID]*givingState
	// events 流水（测试面）。
	events []string
}

type givingState struct {
	HP       int32
	Done     bool
	Invisible bool
}

// Name 模块名。
func (m *GivingNpcModule) Name() string { return "GivingNpcModule" }

// Init 注册期（对齐官方 0x8DAD80）。
func (m *GivingNpcModule) Init(k *Kernel) error {
	m.states = make(map[PersistID]*givingState)
	b := k.Bus()
	b.HeartBeat("GivingNpcModule::HB_ReBorn", m.hbReBorn)
	b.OnEvent("GivingNpc", "OnSpring", m.onSpring)
	b.OnEvent("GivingNpc", "OnSelect", m.onSelect)
	b.OnEvent("GivingNpc", "OnEntry", m.onEntry)
	b.OnEvent("GivingNpc", "OnLoad", m.onLoad)
	b.OnIntCommand("player", 600, m.onSkillBegin)
	b.OnIntCommand("GivingNpc", 603, m.onSkillFinish)
	b.OnIntCommand("player", 602, m.onSkillBreaked)
	b.OnIntCommand("GivingNpc", 852, m.onPickupNpc)
	b.Critical("GivingNpc", "HPRatio", m.onHPRatioChanged)
	return nil
}

func (m *GivingNpcModule) state(k *Kernel, npc PersistID) *givingState {
	st := m.states[npc]
	if st == nil {
		st = &givingState{HP: 100}
		o := k.GetGameObj(npc)
		if o != nil {
			if v := o.GetInt("MaxHP"); v > 0 {
				st.HP = v
			}
		}
		m.states[npc] = st
	}
	return st
}

// onSpring 官方 @0x8DB0D0：目标物弹出（SpringRange/SpringType 校验面）。
func (m *GivingNpcModule) onSpring(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "spring")
}

// onSelect 官方 @0x8DB1E0：选中（进度交互）。
func (m *GivingNpcModule) onSelect(k *Kernel, self, from PersistID, args VarList) {}

// onEntry 官方 @0x8DB380。
func (m *GivingNpcModule) onEntry(k *Kernel, self, from PersistID, args VarList) {}

// onLoad 官方 @0x8DB8C0。
func (m *GivingNpcModule) onLoad(k *Kernel, self, from PersistID, args VarList) {
	m.state(k, self)
}

// onSkillBegin 官方 @0x8DB8F0（命令 600=0x258）：采集开始（player 面）。
func (m *GivingNpcModule) onSkillBegin(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "skill_begin")
}

// onSkillBreaked 官方 @0x8DBFF0（命令 602=0x25A）：采集中断。
func (m *GivingNpcModule) onSkillBreaked(k *Kernel, self, from PersistID, args VarList) {
	m.events = append(m.events, "skill_break")
}

// onSkillFinish 官方 @0x8DBA60（命令 603=0x25B，GivingNpc 面）：采集完成
// → HP 归零 → Done（等待 pickup）。
func (m *GivingNpcModule) onSkillFinish(k *Kernel, self, from PersistID, args VarList) {
	st := m.state(k, self)
	st.HP = 0
	st.Done = true
	if o := k.GetGameObj(self); o != nil {
		if _, ok := o.Class().Prop("HPRatio"); ok {
			_ = k.SetInt(self, "HPRatio", 0) // C_OnHPRatioChanged 临界
		}
	}
	m.events = append(m.events, "skill_finish")
}

// onPickupNpc 官方 @0x8DE850（命令 852=0x354）：拾取产物（PickUpModule/
// DropModule 协作）→ 隐身等待重生。
func (m *GivingNpcModule) onPickupNpc(k *Kernel, self, from PersistID, args VarList) {
	st := m.states[self]
	if st == nil || !st.Done {
		return
	}
	st.Invisible = true
	m.events = append(m.events, "pickup")
}

// HB_ReBorn 官方 @0x8DE7B0：重生泵（Invisible + RefreshTime 面——
// Go 面即刻复活计数归零）。
func (m *GivingNpcModule) hbReBorn(k *Kernel, self PersistID, beats int32) {
	st := m.states[self]
	if st == nil || !st.Invisible {
		return
	}
	o := k.GetGameObj(self)
	if o == nil {
		delete(m.states, self)
		return
	}
	max := o.GetInt("MaxHP")
	if max <= 0 {
		max = 100
	}
	st.HP, st.Done, st.Invisible = max, false, false
	m.events = append(m.events, "reborn")
}

// C_OnHPRatioChanged 官方 @0x8DB750：HP 比例联动。
func (m *GivingNpcModule) onHPRatioChanged(k *Kernel, self PersistID, prop string, old Var) {}

// Events 流水（测试面）。
func (m *GivingNpcModule) Events() []string { return m.events }

// ---- SummonNpcModule ----

// SummonNpcModule 复刻官方 summonnpcmodule（Init @0x8C4CE0 / LoadResource
// @0x8C4D90，玩法表 Init 序 167；符号 331）——事件召唤 NPC。
//
// LoadResource @0x8C4D90：ini/Quest/Convoy/EventSummon.xml（21 行：
// ID → r="npc_id,count,x,z" 多段——真档引号方言容忍解析）。
type SummonNpcModule struct {
	BaseModule

	// Summons 召唤表：事件 ID → 召唤段（npcID/数量/坐标）。
	Summons map[string][]SummonEntry
	// spawned 召唤流水（测试面）。
	spawned []string
}

// SummonEntry 一段召唤（npc/count/x/z）。
type SummonEntry struct {
	NpcID string
	Count int32
	X, Z  int32
}

// Name 模块名。
func (m *SummonNpcModule) Name() string { return "SummonNpcModule" }

// Init 注册期（对齐官方 0x8C4CE0——OnCreateClass 动态挂载面）。
func (m *SummonNpcModule) Init(k *Kernel) error {
	m.Summons = make(map[string][]SummonEntry)
	return nil
}

// summonSeg 召唤段正则（真档 r="npc,count,x,z" 方言）。
var summonSeg = regexp.MustCompile(`([\w]+),(\d+),(-?\d+),(-?\d+)`)

// LoadResource 官方 @0x8C4D90：ini/Quest/Convoy/EventSummon.xml。
func (m *SummonNpcModule) LoadResource(k *Kernel) error {
	if k.ResourcePath == "" {
		return fmt.Errorf("fsgame: SummonNpcModule.LoadResource: ResourcePath 未设置")
	}
	data, err := readResource(k.ResourcePath, "ini/Quest/Convoy/EventSummon.xml")
	if err != nil {
		return fmt.Errorf("fsgame: SummonNpcModule::LoadResource EventSummon.xml: %w", err)
	}
	rows, err := ParsePropTable(data)
	if err != nil {
		return err
	}
	for _, row := range rows {
		id := row["ID"]
		if id == "" {
			continue
		}
		// 真档行内嵌多段 r="npc,count,x,z"（列名被引号吞并——全文扫描）。
		for _, seg := range summonSeg.FindAllStringSubmatch(row[""], -1) {
			m.Summons[id] = append(m.Summons[id], SummonEntry{
				NpcID: seg[1], Count: atoi32(seg[2]), X: atoi32(seg[3]), Z: atoi32(seg[4]),
			})
		}
	}
	// 列名为空的方言：走原始行扫描兜底。
	if len(m.Summons) == 0 {
		raw := string(data)
		for _, line := range strings.Split(raw, "\n") {
			idm := regexp.MustCompile(`ID="(\d+)"`).FindStringSubmatch(line)
			if idm == nil {
				continue
			}
			for _, seg := range summonSeg.FindAllStringSubmatch(line, -1) {
				m.Summons[idm[1]] = append(m.Summons[idm[1]], SummonEntry{
					NpcID: seg[1], Count: atoi32(seg[2]), X: atoi32(seg[3]), Z: atoi32(seg[4]),
				})
			}
		}
	}
	k.Tracef("SummonNpcModule: LoadResource EventSummon %d 事件", len(m.Summons))
	return nil
}

// Summon 召唤闭环（官方 0x8C4CE0 族）：事件 ID → 表段展开 → NpcResource
// 物化 × count。
func (m *SummonNpcModule) Summon(k *Kernel, eventID string) int {
	nrm, _ := k.GetLogicModule("NpcResourceModule").(*NpcResourceModule)
	n := 0
	for _, seg := range m.Summons[eventID] {
		for i := int32(0); i < seg.Count; i++ {
			if nrm != nil {
				if id := nrm.Spawn(k, seg.NpcID); id.Valid() {
					n++
				}
			} else {
				n++
			}
		}
		m.spawned = append(m.spawned, fmt.Sprintf("%s×%d@(%d,%d)", seg.NpcID, seg.Count, seg.X, seg.Z))
	}
	k.Tracef("SummonNpcModule: summon %s → %d", eventID, n)
	return n
}

// Spawned 召唤流水（测试面）。
func (m *SummonNpcModule) Spawned() []string { return m.spawned }

// ---- FunctionNpcModule ----

// FunctionNpcModule 复刻官方 functionnpcmodule（Init @0x8D9EE0，玩法表
// Init 序 166；符号 56）——功能 NPC（商店/传送等功能绑定）。
//
// 官方 Init 注册面：
//
//	AddEventCallback ("FunctionNpc","OnLoad",@0x8DA060) / ("FunctionNpc",
//	    "OnEntry",@0x8DA2E0) / ("FunctionNpc","OnSelect",@0x8DA340) /
//	    ("FunctionNpc","OnEndSpring",@0x8DA660)
type FunctionNpcModule struct {
	BaseModule

	// selects 交互流水（测试面）。
	selects []string
}

// Name 模块名。
func (m *FunctionNpcModule) Name() string { return "FunctionNpcModule" }

// Init 注册期（对齐官方 0x8D9EE0）。
func (m *FunctionNpcModule) Init(k *Kernel) error {
	b := k.Bus()
	b.OnEvent("FunctionNpc", "OnLoad", m.onLoad)
	b.OnEvent("FunctionNpc", "OnEntry", m.onEntry)
	b.OnEvent("FunctionNpc", "OnSelect", m.onSelect)
	b.OnEvent("FunctionNpc", "OnEndSpring", m.onEndSpring)
	return nil
}

func (m *FunctionNpcModule) onLoad(k *Kernel, self, from PersistID, args VarList)  {}
func (m *FunctionNpcModule) onEntry(k *Kernel, self, from PersistID, args VarList) {}

// onSelect 官方 @0x8DA340：功能分发（商店/传送/修理——MapMag/Shop 协作）。
func (m *FunctionNpcModule) onSelect(k *Kernel, self, from PersistID, args VarList) {
	m.selects = append(m.selects, args.String(0))
}

// onEndSpring 官方 @0x8DA660。
func (m *FunctionNpcModule) onEndSpring(k *Kernel, self, from PersistID, args VarList) {}

// Selects 交互流水（测试面）。
func (m *FunctionNpcModule) Selects() []string { return m.selects }
