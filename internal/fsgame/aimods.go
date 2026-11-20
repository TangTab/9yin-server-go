package fsgame

// 十期：AI·世界模拟底座九模块（docs/official/FsGame全解与Go复刻.md §8
// AI·世界模拟底座表）。官方源码树 fsgame\aimodule\。
//
// 配置/记录面（struct/npc_property/ai_npc_data.xml + ai_motion.xml）：
// WarnType（0 被动 / 1 球 / 2 圆柱 / 3 扇柱 / 4 球+扇柱）+ WarnTime +
// SphereRadius/CylinderRadius/Cylinder{Lower,Upper}Height/SectorHalfAngle；
// ChaseRange/ChaseType/ChaseProbability；PatrolMode/PathID/PatrolRange；
// AITemplate/CharacterPackID/ChatTypeID/SkillPackID；CurState/SubState/
// PreState + Born{X,Y,Z}/BF{X,Y,Z}；StopAI/StopPatrol/Resting；
// AIFirstAttacker/LastObject/MasterID/HateTransferScale/CallHelp*；
// AIMotioning/PathFinding/FindPath{X,Z}（ai_motion）。记录：HateList(4)/
// SpringList(1)/AIGroupPosRec/AttackerPosRec/AI_Chat_rec/AI_SpringChat_rec。

import (
	"fmt"
	"math"
	"strings"
)

// CurState 状态机词汇（ai_npc_data CurState int；官方枚举未逐值反汇编，
// Go 收编：0 站桩 → 1 巡逻 → 2 追击 → 3 战斗 → 4 回巢 → 5 死亡）。
const (
	AIStateStand int32 = iota
	AIStatePatrol
	AIStateChase
	AIStateFight
	AIStateReturn
	AIStateDead
)

// AI 类族（OnSpring/OnEntry/OnMotion 注册面五类 + BattleCarrier）。
var aiNpcClasses = []string{"AttackNpc", "Pet", "FormationNpc", "FlyNpc", "SkillEffectNpc"}

// aiWarnClasses 警戒注册面三类（AttackNpc/Pet/SkillEffectNpc——regs_calls
// 定案；OnEntry 仅 AttackNpc/SkillEffectNpc）。
var aiWarnClasses = []string{"AttackNpc", "Pet", "SkillEffectNpc"}

// dist2D 平面距离。
func dist2D(x1, z1, x2, z2 float32) float32 {
	dx, dz := x1-x2, z1-z2
	return float32(math.Sqrt(float64(dx*dx + dz*dz)))
}

// ---- AIModule（Init 序 46）----

// AIModule 复刻官方 AIModule（Init @0x0DCD0；符号 223）——AI 状态机中枢。
//
// 官方 Init 注册面（p2scan；一行 EventCallback 参数为扫描噪声未定位）：
//
//	DeclareRecHook ("play_point_sound") → AIEffect 音效联动
//	DeclareCritical ("AIEffect", C_OnAIEffectChanged)
type AIModule struct {
	BaseModule

	// transitions 状态迁移流水（测试面：npc → "from→to"）。
	transitions []string
}

// Name 模块名。
func (m *AIModule) Name() string { return "AIModule" }

// Init 注册期（对齐官方 0x0DCD0）。
func (m *AIModule) Init(k *Kernel) error {
	b := k.Bus()
	b.RecHook("play_point_sound", m.onPlayPointSound)
	b.Critical("AttackNpc", "AIEffect", m.onAIEffectChanged)
	return nil
}

// Transition 状态迁移（CurState/SubState/PreState 三联动；dead 终态不迁）。
func (m *AIModule) Transition(k *Kernel, npc PersistID, state int32) bool {
	o := k.GetGameObj(npc)
	if o == nil || o.ClassName() == "" {
		return false
	}
	if _, ok := o.Class().Prop("CurState"); !ok {
		return false
	}
	cur := o.GetInt("CurState")
	if cur == state || cur == AIStateDead {
		return false
	}
	_ = o.Set("PreState", cur)
	_ = o.Set("CurState", state)
	m.transitions = append(m.transitions, fmt.Sprintf("%d:%d→%d", npc.ID(), cur, state))
	k.Tracef("AIModule: npc=%v state %d→%d", npc.ID(), cur, state)
	return true
}

// onPlayPointSound AIEffect 点音效联动（RecHook）。
func (m *AIModule) onPlayPointSound(k *Kernel, self PersistID, record string, op, row int32) {}

// onAIEffectChanged AI 特效临界联动。
func (m *AIModule) onAIEffectChanged(k *Kernel, self PersistID, prop string, old Var) {
	k.Tracef("AIModule: AIEffect npc=%v", self.ID())
}

// Transitions 状态迁移流水（测试面）。
func (m *AIModule) Transitions() []string { return m.transitions }

// ---- AICharacterModule（Init 序 39）----

// AICharacterModule 复刻官方 AICharacterModule（Init @0x4D350；符号 14）——
// AI 角色装配面：五类 NPC 进场初始化。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("AttackNpc"/"Pet"/"FormationNpc"/"FlyNpc"/
//	    "SkillEffectNpc","OnEntry",@0x4D490 同一处理器)
type AICharacterModule struct {
	BaseModule

	// inits 装配流水（测试面）。
	inits []string
}

// Name 模块名。
func (m *AICharacterModule) Name() string { return "AICharacterModule" }

// Init 注册期（对齐官方 0x4D350）。
func (m *AICharacterModule) Init(k *Kernel) error {
	for _, cls := range aiNpcClasses {
		k.Bus().OnEvent(cls, "OnEntry", m.onEntry)
	}
	return nil
}

// onEntry 官方 @0x4D490：进场装配——出生点锚定（Born* 空值回落当前坐标）、
// 状态机置 stand、Resting/StopAI 复位、配置包（AITemplate/CharacterPackID/
// SkillPackID/ChatTypeID）登记面。
func (m *AICharacterModule) onEntry(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	x, y, z := o.Pos()
	if _, ok := o.Class().Prop("BornX"); ok && o.GetFloat("BornX") == 0 && o.GetFloat("BornZ") == 0 {
		_ = o.Set("BornX", x)
		_ = o.Set("BornY", y)
		_ = o.Set("BornZ", z)
	}
	if _, ok := o.Class().Prop("CurState"); ok {
		_ = o.Set("CurState", AIStateStand)
		_ = o.Set("SubState", int32(0))
		_ = o.Set("PreState", int32(0))
	}
	if _, ok := o.Class().Prop("Resting"); ok {
		_ = o.Set("Resting", int32(0))
	}
	m.inits = append(m.inits, fmt.Sprintf("char:%d", self.ID()))
	k.Tracef("AICharacterModule: OnEntry npc=%v template=%d pack=%s skill=%s",
		self.ID(), o.GetInt("AITemplate"), o.GetString("CharacterPackID"), o.GetString("SkillPackID"))
}

// Inits 装配流水（测试面）。
func (m *AICharacterModule) Inits() []string { return m.inits }

// ---- WarnModule（Init 序 42）----

// WarnModule 复刻官方 WarnModule（Init @0x76B70；符号 112，LoadResource
// @0xBDFF60）——警戒域：感知几何（球/圆柱/扇柱）检测 + 警戒计时入战。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("AttackNpc"/"Pet"/"SkillEffectNpc","OnSpring",@0x76E00) /
//	    ("AttackNpc"/"Pet","OnEndSpring",@0x77110) /
//	    ("AttackNpc","OnEntry",@0x77280) / ("SkillEffectNpc","OnEntry",@0x775D0)
//	DeclareHeartBeat ×3（HB_AIWarn ×2 + HB_AIWarn_SkillNpc）
//	DeclareLuaExt nx_getlinkpos
type WarnModule struct {
	BaseModule

	// springs 警戒追踪（npc → 目标 → 首见时刻）。
	springs map[PersistID]map[PersistID]int64
	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *WarnModule) Name() string { return "WarnModule" }

// Init 注册期（对齐官方 0x76B70）。
func (m *WarnModule) Init(k *Kernel) error {
	m.springs = make(map[PersistID]map[PersistID]int64)
	b := k.Bus()
	for _, cls := range aiWarnClasses {
		b.OnEvent(cls, "OnSpring", m.onSpring)
	}
	for _, cls := range []string{"AttackNpc", "Pet"} {
		b.OnEvent(cls, "OnEndSpring", m.onEndSpring)
	}
	b.OnEvent("AttackNpc", "OnEntry", m.onNpcEntry)
	b.OnEvent("SkillEffectNpc", "OnEntry", m.onNpcEntry)
	b.HeartBeat("WarnModule::HB_AIWarn", m.hbAIWarn)
	b.HeartBeat("WarnModule::HB_AIWarn_SkillNpc", m.hbAIWarnSkillNpc)
	return nil
}

// onSpring 官方 @0x76E00：目标弹入感知范围（空间层外发；args[0]=目标）。
func (m *WarnModule) onSpring(k *Kernel, self, from PersistID, args VarList) {
	if len(args) == 0 {
		return
	}
	target, _ := args[0].(PersistID)
	if !target.Valid() {
		return
	}
	if m.springs[self] == nil {
		m.springs[self] = make(map[PersistID]int64)
	}
	if _, seen := m.springs[self][target]; !seen {
		m.springs[self][target] = unixNow()
		m.events = append(m.events, fmt.Sprintf("spring:%d", target.ID()))
	}
	m.linkSpringList(k, self, target, true)
}

// onEndSpring 官方 @0x77110：目标弹出感知范围。
func (m *WarnModule) onEndSpring(k *Kernel, self, from PersistID, args VarList) {
	if len(args) == 0 {
		return
	}
	target, _ := args[0].(PersistID)
	delete(m.springs[self], target)
	m.linkSpringList(k, self, target, false)
	m.events = append(m.events, fmt.Sprintf("endspring:%d", target.ID()))
}

// onNpcEntry 官方 @0x77280/@0x775D0：NPC 进场（警戒态复位）。
func (m *WarnModule) onNpcEntry(k *Kernel, self, from PersistID, args VarList) {
	delete(m.springs, self)
}

// hbAIWarn 官方 HB_AIWarn：警戒泵——扫描玩家入感知几何（WarnType 1-4）→
// 登记（等价 OnSpring 自动面）；警戒时满（WarnTime 秒）→ 入战（CurState=
// fight + AIFirstAttacker 最近目标 + HateList 种子仇恨 + AIMotion 追击面）。
func (m *WarnModule) hbAIWarn(k *Kernel, self PersistID, beats int32) {
	m.pump(k, self, false)
}

// hbAIWarnSkillNpc 官方 HB_AIWarn_SkillNpc（SkillEffectNpc 专用泵）。
func (m *WarnModule) hbAIWarnSkillNpc(k *Kernel, self PersistID, beats int32) {
	if o := k.GetGameObj(self); o != nil && o.ClassName() == "SkillEffectNpc" {
		m.pump(k, self, true)
	}
}

// pump 警戒泵主体（skillNpc=true 时跳过状态机面——技能特效 NPC 无战斗态）。
func (m *WarnModule) pump(k *Kernel, self PersistID, skillNpc bool) {
	o := k.GetGameObj(self)
	if o == nil || !isAiClass(k, self) {
		return
	}
	if ts := tempStopAIOf(k); ts != nil {
		ts.EnsureResumed(k, self)
	}
	if o.GetInt("StopAI") != 0 || o.GetInt("CurState") == AIStateDead {
		return
	}
	warnType := o.GetInt("WarnType")
	if warnType == 0 {
		return // 被动怪：入战走 HateModule 被击面
	}
	if m.springs[self] == nil {
		m.springs[self] = make(map[PersistID]int64)
	}
	now := unixNow()
	// 几何扫描：玩家进入感知几何 → 登记；离开 → 除名。
	for _, pid := range k.PlayersSnapshot() {
		in := m.inWarnGeometry(k, o, pid)
		_, seen := m.springs[self][pid]
		switch {
		case in && !seen:
			m.springs[self][pid] = now
			m.linkSpringList(k, self, pid, true)
			m.events = append(m.events, fmt.Sprintf("spring:%d", pid.ID()))
		case !in && seen:
			delete(m.springs[self], pid)
			m.linkSpringList(k, self, pid, false)
			m.events = append(m.events, fmt.Sprintf("endspring:%d", pid.ID()))
		}
	}
	if skillNpc || len(m.springs[self]) == 0 {
		return
	}
	// 警戒时满 → 入战（AIModule 状态机 + 仇恨种子 + 追击）。
	warnTime := int64(o.GetInt("WarnTime"))
	inState := o.GetInt("CurState")
	for target, first := range m.springs[self] {
		if now-first < warnTime {
			continue
		}
		if inState == AIStateStand || inState == AIStatePatrol || inState == AIStateReturn {
			if am := aiModuleOf(k); am != nil {
				am.Transition(k, self, AIStateFight)
			}
		}
		if _, ok := o.Class().Prop("AIFirstAttacker"); ok {
			_ = o.Set("AIFirstAttacker", target)
		}
		if hm := hateModuleOf(k); hm != nil {
			hm.AddHate(k, self, target, 1)
		}
		m.events = append(m.events, fmt.Sprintf("fight:%d", target.ID()))
		break
	}
}

// inWarnGeometry 感知几何检测（WarnType 1 球 / 2 圆柱 / 3 扇柱 / 4 球+扇柱；
// 圆柱/扇柱限高 [BornY+Lower, BornY+Upper]，扇柱以朝向中轴 ±HalfAngle）。
func (m *WarnModule) inWarnGeometry(k *Kernel, o *GameObject, target PersistID) bool {
	to := k.GetGameObj(target)
	if to == nil {
		return false
	}
	nx, ny, nz := o.Pos()
	tx, ty, tz := to.Pos()
	d := dist2D(nx, nz, tx, tz)
	warnType := o.GetInt("WarnType")
	sphere := func() bool { return d <= o.GetFloat("SphereRadius") }
	cylinder := func() bool {
		if d > o.GetFloat("CylinderRadius") {
			return false
		}
		lower, upper := o.GetFloat("CylinderLowerHeight"), o.GetFloat("CylinderUpperHeight")
		return lower == upper || (ty >= ny+lower && ty <= ny+upper)
	}
	sector := func() bool {
		if !cylinder() {
			return false
		}
		half := o.GetFloat("SectorHalfAngle")
		if half <= 0 || half >= 180 {
			return true
		}
		ang := float64(math.Atan2(float64(tz-nz), float64(tx-nx)))
		diff := math.Abs(ang - float64(o.Face()))
		for diff > math.Pi {
			diff = math.Abs(2*math.Pi - diff)
		}
		return diff <= float64(half)*math.Pi/180
	}
	switch warnType {
	case 1:
		return sphere()
	case 2:
		return cylinder()
	case 3:
		return sector()
	case 4:
		return sphere() || sector()
	}
	return false
}

// linkSpringList SpringList 记录联动（进=true 追加 / 出=false 删行）。
func (m *WarnModule) linkSpringList(k *Kernel, npc, target PersistID, in bool) {
	o := k.GetGameObj(npc)
	if o == nil {
		return
	}
	r := o.Record("SpringList")
	if r == nil {
		return
	}
	if in {
		if r.Rows() < r.Def().MaxRows {
			r.AppendRow(VarList{target})
		}
		return
	}
	for i := r.Rows() - 1; i >= 0; i-- {
		if v, ok := r.Row(i)[0].(PersistID); ok && v == target {
			r.DeleteRow(i)
		}
	}
}

// NxGetLinkPos nx_getlinkpos 的 Go 等价（联动对象位置——官方 Lua 面）。
func (m *WarnModule) NxGetLinkPos(k *Kernel, npc PersistID) (float32, float32, float32, bool) {
	o := k.GetGameObj(npc)
	if o == nil {
		return 0, 0, 0, false
	}
	x, y, z := o.Pos()
	return x, y, z, true
}

// Events 警戒流水（测试面）。
func (m *WarnModule) Events() []string { return m.events }

// ---- HateModule（Init 序 43）----

// HateModule 复刻官方 HateModule（Init @0x6E7B0，LoadResource @0x6E960；
// 符号 347）——仇恨列表引擎。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("AttackNpc"/"Pet"/"SkillEffectNpc","OnSpring",@0x6EE80)
//	AddIntCommandHook ("AttackNpc"/"Pet",618,@0x6F100 被击) /
//	    ("player",619,@0x6F850) / ("player",618,@0x6FE00)
type HateModule struct {
	BaseModule

	// events 流水（测试面）。
	events []string
}

// Name 模块名。
func (m *HateModule) Name() string { return "HateModule" }

// Init 注册期（对齐官方 0x6E7B0）。
func (m *HateModule) Init(k *Kernel) error {
	b := k.Bus()
	for _, cls := range aiWarnClasses {
		b.OnEvent(cls, "OnSpring", m.onSpring)
	}
	b.OnIntCommand("AttackNpc", 618, m.onBeAttacked)
	b.OnIntCommand("Pet", 618, m.onBeAttacked)
	b.OnIntCommand("player", 619, m.onPlayerAttack)
	b.OnIntCommand("player", 618, m.onBeAttacked)
	return nil
}

// onSpring 官方 @0x6EE80：目标入感知 → 仇恨种子（主动怪面；被动怪此面
// 不加仇恨——WarnModule.pump 已对主动怪种子，此处幂等）。
func (m *HateModule) onSpring(k *Kernel, self, from PersistID, args VarList) {
	o := k.GetGameObj(self)
	if o == nil || o.GetInt("WarnType") == 0 {
		return // 被动怪：被击才入表
	}
	if len(args) == 0 {
		return
	}
	if target, ok := args[0].(PersistID); ok && target.Valid() {
		m.AddHate(k, self, target, 1)
	}
}

// onBeAttacked 官方 @0x6F100（命令 618=0x26A）：被击 → 仇恨累加
//（args: 攻击者, 伤害值[, 优先级]）+ 入战联动（AIModule fight + 追击）。
func (m *HateModule) onBeAttacked(k *Kernel, self, from PersistID, args VarList) {
	attacker := self
	if len(args) > 0 {
		if v, ok := args[0].(PersistID); ok && v.Valid() {
			attacker = v
		}
	}
	damage := int32(1)
	if len(args) > 1 {
		damage = args.Int(1)
	}
	if !m.AddHate(k, self, attacker, damage) {
		return
	}
	m.events = append(m.events, fmt.Sprintf("hate:%d +%d", attacker.ID(), damage))
	o := k.GetGameObj(self)
	if o == nil {
		return
	}
	if o.GetInt("CurState") != AIStateFight {
		if am := aiModuleOf(k); am != nil {
			am.Transition(k, self, AIStateFight)
		}
		if _, ok := o.Class().Prop("AIFirstAttacker"); ok {
			_ = o.Set("AIFirstAttacker", attacker)
		}
	}
	m.chaseIfFar(k, self)
}

// onPlayerAttack 官方 @0x6F850（命令 619=0x26B）：玩家攻击命中通知
//（args: 目标 NPC, 伤害值）——NPC 侧 618 的玩家来向面。
func (m *HateModule) onPlayerAttack(k *Kernel, self, from PersistID, args VarList) {
	if len(args) == 0 {
		return
	}
	if target, ok := args[0].(PersistID); ok && target.Valid() {
		damage := int32(1)
		if len(args) > 1 {
			damage = args.Int(1)
		}
		if m.AddHate(k, target, self, damage) {
			m.events = append(m.events, fmt.Sprintf("hate:%d +%d", self.ID(), damage))
		}
	}
}

// AddHate 仇恨入表（HateList upsert：col=对象/仇恨值/最近攻击时间/优先级；
// 仇恨转移：MasterID 有效且 HateTransferScale>0 时按比例分流主人）。
func (m *HateModule) AddHate(k *Kernel, npc, target PersistID, hate int32) bool {
	o := k.GetGameObj(npc)
	if o == nil {
		return false
	}
	r := o.Record("HateList")
	if r == nil {
		return false
	}
	if master, ok := o.Get("MasterID").(PersistID); ok && master.Valid() && o.GetFloat("HateTransferScale") > 0 {
		scale := o.GetFloat("HateTransferScale")
		transfer := int32(float32(hate) * scale)
		if transfer > 0 && master != target {
			m.addHateRow(k, npc, master, transfer)
			hate -= transfer
		}
	}
	return m.addHateRow(k, npc, target, hate)
}

// addHateRow 单对象仇恨 upsert。
func (m *HateModule) addHateRow(k *Kernel, npc, target PersistID, hate int32) bool {
	o := k.GetGameObj(npc)
	r := o.Record("HateList")
	for i := 0; i < r.Rows(); i++ {
		if v, ok := r.Row(i)[0].(PersistID); ok && v == target {
			row := r.Row(i)
			row[1] = row.Int(1) + hate
			row[2] = unixNow()
			r.ModifyRow(i, row)
			return true
		}
	}
	if r.Rows() >= r.Def().MaxRows {
		return false
	}
	r.AppendRow(VarList{target, hate, unixNow(), int32(0)})
	return true
}

// Target 最大仇恨目标（LastObject/AIFirstAttacker 双写；无表返 Null）。
func (m *HateModule) Target(k *Kernel, npc PersistID) PersistID {
	o := k.GetGameObj(npc)
	if o == nil {
		return NullPersistID
	}
	r := o.Record("HateList")
	if r == nil {
		return NullPersistID
	}
	best, bestHate := NullPersistID, int32(-1)
	for i := 0; i < r.Rows(); i++ {
		if h := r.Row(i).Int(1); h > bestHate {
			if v, ok := r.Row(i)[0].(PersistID); ok && v.Valid() {
				best, bestHate = v, h
			}
		}
	}
	if best.Valid() {
		if _, ok := o.Class().Prop("LastObject"); ok {
			_ = o.Set("LastObject", best)
		}
		if _, ok := o.Class().Prop("AIFirstAttacker"); ok {
			_ = o.Set("AIFirstAttacker", best)
		}
	}
	return best
}

// ClearHate 清仇恨表（脱战/死亡面）。
func (m *HateModule) ClearHate(k *Kernel, npc PersistID) {
	o := k.GetGameObj(npc)
	if o == nil {
		return
	}
	if r := o.Record("HateList"); r != nil {
		r.Clear()
	}
}

// chaseIfFar 目标超出 ChaseRange → 脱战回巢（AIModule return + 仇恨清 +
// AIMotion 回 Born；ChaseRange=0 不追不脱——站桩怪）。
func (m *HateModule) chaseIfFar(k *Kernel, npc PersistID) {
	o := k.GetGameObj(npc)
	if o == nil {
		return
	}
	chase := o.GetFloat("ChaseRange")
	if chase <= 0 {
		return
	}
	target, _ := o.Get("AIFirstAttacker").(PersistID)
	to := k.GetGameObj(target)
	if to == nil {
		return
	}
	nx, _, nz := o.Pos()
	tx, _, tz := to.Pos()
	bx, bz := o.GetFloat("BornX"), o.GetFloat("BornZ")
	if dist2D(nx, nz, tx, tz) > chase || dist2D(bx, bz, tx, tz) > chase {
		m.ClearHate(k, npc)
		if am := aiModuleOf(k); am != nil {
			am.Transition(k, npc, AIStateReturn)
		}
		if mm := aiMotionOf(k); mm != nil {
			mm.MoveTo(k, npc, bx, bz)
		}
		m.events = append(m.events, "return")
	}
}

// Events 仇恨流水（测试面）。
func (m *HateModule) Events() []string { return m.events }

// ---- AIMotionModule（Init 序 45）----

// AIMotionModule 复刻官方 AIMotionModule（Init @0x42E90；符号 47）——AI
// 移动执行：MoveTo 目标步进 + 巡逻 + 到点回调。
//
// 官方 Init 注册面（regs_calls 定案）：
//
//	AddEventCallback ("AttackNpc"/"BattleCarrier"/"Pet"/"FormationNpc",
//	    "OnMotion",@0x44620) / ("FlyNpc","OnMotion",@0x453A0)
//	DeclareCritical ("SpeedRatio",C_OnSpeedRatioChanged)
//	DeclareHeartBeat ("AIMotionModule::HB_MoveOver",@0x42D80——FlyNpc 面)
type AIMotionModule struct {
	BaseModule

	// arrives 到点流水（测试面：npc → "x,z"）。
	arrives []string
}

// Name 模块名。
func (m *AIMotionModule) Name() string { return "AIMotionModule" }

// Init 注册期（对齐官方 0x42E90）。
func (m *AIMotionModule) Init(k *Kernel) error {
	b := k.Bus()
	for _, cls := range []string{"AttackNpc", "BattleCarrier", "Pet", "FormationNpc"} {
		b.OnEvent(cls, "OnMotion", m.onMotion)
	}
	b.OnEvent("FlyNpc", "OnMotion", m.onFlyMotion)
	b.Critical("AttackNpc", "SpeedRatio", m.onSpeedRatioChanged)
	b.HeartBeat("AIMotionModule::HB_MoveOver", m.hbMoveOver)
	return nil
}

// MoveTo 移动目标设定（FindPathX/Z + AIMotioning=1 + PathFinding 置位）。
func (m *AIMotionModule) MoveTo(k *Kernel, npc PersistID, x, z float32) bool {
	o := k.GetGameObj(npc)
	if o == nil {
		return false
	}
	if o.GetInt("StopAI") != 0 || o.GetInt("StopPatrol") != 0 && o.GetInt("CurState") == AIStatePatrol {
		return false
	}
	if _, ok := o.Class().Prop("FindPathX"); ok {
		_ = o.Set("FindPathX", x)
		_ = o.Set("FindPathZ", z)
	}
	if _, ok := o.Class().Prop("AIMotioning"); ok {
		_ = o.Set("AIMotioning", int32(1))
	}
	return true
}

// onMotion 官方 @0x44620：移动事件（步进完成通知面）。
func (m *AIMotionModule) onMotion(k *Kernel, self, from PersistID, args VarList) {}

// onFlyMotion 官方 @0x453A0（FlyNpc 专用：飞行高度钳制面）。
func (m *AIMotionModule) onFlyMotion(k *Kernel, self, from PersistID, args VarList) {}

// onSpeedRatioChanged 官方临界：移速比变更（步进速度联动）。
func (m *AIMotionModule) onSpeedRatioChanged(k *Kernel, self PersistID, prop string, old Var) {}

// hbMoveOver 官方 HB_MoveOver：移动泵——朝 FindPathX/Z 按 Speed×SpeedRatio
// 步进（Go 收编：每拍一步长）；到达 → AIMotioning=0 + OnMotion 事件外发 +
// 追击中距目标 AttackRange 内转入战（AIModule fight）。回巢到点 → stand。
func (m *AIMotionModule) hbMoveOver(k *Kernel, self PersistID, beats int32) {
	o := k.GetGameObj(self)
	if o == nil || !isAiClass(k, self) {
		return
	}
	if ts := tempStopAIOf(k); ts != nil {
		ts.EnsureResumed(k, self)
	}
	if o.GetInt("AIMotioning") == 0 || o.GetInt("StopAI") != 0 {
		return
	}
	tx, tz := o.GetFloat("FindPathX"), o.GetFloat("FindPathZ")
	x, y, z := o.Pos()
	d := dist2D(x, z, tx, tz)
	step := o.GetFloat("Speed") * o.GetFloat("SpeedRatio")
	if step <= 0 {
		step = 5 // 缺省步长（Speed 未配置面）
	}
	if d <= step {
		o.SetPos(tx, y, tz)
		if _, ok := o.Class().Prop("AIMotioning"); ok {
			_ = o.Set("AIMotioning", int32(0))
		}
		m.arrives = append(m.arrives, fmt.Sprintf("%d:%.0f,%.0f", self.ID(), tx, tz))
		k.Bus().FireEvent(k, o.ClassName(), "OnMotion", self, NullPersistID, nil)
		// 到点态迁移：追击 → 战斗；回巢 → 站桩。
		switch o.GetInt("CurState") {
		case AIStateChase:
			if am := aiModuleOf(k); am != nil {
				am.Transition(k, self, AIStateFight)
			}
		case AIStateReturn:
			if am := aiModuleOf(k); am != nil {
				am.Transition(k, self, AIStateStand)
			}
		}
		return
	}
	nx, nz := x+(tx-x)/d*step, z+(tz-z)/d*step
	o.SetPos(nx, y, nz)
	o.SetFace(float32(math.Atan2(float64(tz-z), float64(tx-x))))
}

// Arrives 到点流水（测试面）。
func (m *AIMotionModule) Arrives() []string { return m.arrives }

// ---- AISkillControlModule（Init 序 40）----

// AISkillControlModule 复刻官方 AISkillControlModule（Init @0x5EAA0；
// 符号 586）——AI 技能选用（战斗态按 SkillPackID 轮选）。
//
// 官方 Init 注册面（regs_calls 定案，三临界）：
//
//	DeclareCritical (HandleState → OnHandleStateChanged @0x5EC20) ×2 行 /
//	    (SkillPack → OnHandleSkillPackChanged @0x5ECB0)
type AISkillControlModule struct {
	BaseModule

	// picks 技能选用流水（测试面：npc → skillID）。
	picks []string
	// rounds 轮选游标（npc → 序）。
	rounds map[PersistID]int
}

// Name 模块名。
func (m *AISkillControlModule) Name() string { return "AISkillControlModule" }

// Init 注册期（对齐官方 0x5EAA0）。
func (m *AISkillControlModule) Init(k *Kernel) error {
	m.rounds = make(map[PersistID]int)
	b := k.Bus()
	b.Critical("AttackNpc", "HandleState", m.onHandleStateChanged)
	b.Critical("AttackNpc", "SkillPackID", m.onHandleSkillPackChanged)
	return nil
}

// PickSkill 战斗态技能选用：SkillPackID（逗号串）轮选——序游标按 NPC 记，
// 死循环技能序稳定；非战斗态或无包返空。
func (m *AISkillControlModule) PickSkill(k *Kernel, npc PersistID) string {
	o := k.GetGameObj(npc)
	if o == nil || o.GetInt("CurState") != AIStateFight {
		return ""
	}
	pack := o.GetString("SkillPackID")
	if pack == "" {
		return ""
	}
	skills := strings.Split(pack, ",")
	i := m.rounds[npc] % len(skills)
	m.rounds[npc] = (m.rounds[npc] + 1) % len(skills)
	skill := strings.TrimSpace(skills[i])
	if skill != "" {
		m.picks = append(m.picks, fmt.Sprintf("%d:%s", npc.ID(), skill))
	}
	return skill
}

// onHandleStateChanged 官方 @0x5EC20：控制态变更（技能面重置游标）。
func (m *AISkillControlModule) onHandleStateChanged(k *Kernel, self PersistID, prop string, old Var) {
	delete(m.rounds, self)
}

// onHandleSkillPackChanged 官方 @0x5ECB0：技能包变更。
func (m *AISkillControlModule) onHandleSkillPackChanged(k *Kernel, self PersistID, prop string, old Var) {
	delete(m.rounds, self)
}

// Picks 选用流水（测试面）。
func (m *AISkillControlModule) Picks() []string { return m.picks }

// ---- AIOnCommandMotion（Init 序 41）----

// AIOnCommandMotion 复刻官方 AIOnCommandMotion（Init @0x45780；符号 57；
// 无事件/命令注册——受控移动 API 面：主人命令/跟随）。AIInstruction 词汇
//（ai_npc_data 主人命令）：0 跟随 / 1 停留 / 2 攻击 / 3 回归。
const (
	AIInstrFollow int32 = iota
	AIInstrStay
	AIInstrAttack
	AIInstrBack
)

// AIOnCommandMotion 受控移动：Pet/召唤物按主人命令移动（跟随超
// FollowProtectRadius 收拢、停留原地、回归主人）。
type AIOnCommandMotion struct {
	BaseModule

	// cmds 命令流水（测试面）。
	cmds []string
}

// Name 模块名。
func (m *AIOnCommandMotion) Name() string { return "AIOnCommandMotion" }

// Init 注册期（官方无注册面——纯 API 模块）。
func (m *AIOnCommandMotion) Init(k *Kernel) error { return nil }

// Command 主人命令（AIInstruction 写面）+ 即时执行（跟随/停留/回归）。
func (m *AIOnCommandMotion) Command(k *Kernel, npc PersistID, instr int32) {
	o := k.GetGameObj(npc)
	if o == nil {
		return
	}
	if _, ok := o.Class().Prop("AIInstruction"); ok {
		_ = o.Set("AIInstruction", instr)
	}
	m.cmds = append(m.cmds, fmt.Sprintf("%d:%d", npc.ID(), instr))
	mm := aiMotionOf(k)
	master, _ := o.Get("MasterID").(PersistID)
	mo := k.GetGameObj(master)
	switch instr {
	case AIInstrFollow:
		if mm != nil && mo != nil {
			mx, _, mz := mo.Pos()
			nx, _, nz := o.Pos()
			if dist2D(nx, nz, mx, mz) > o.GetFloat("FollowProtectRadius") {
				mm.MoveTo(k, npc, mx, mz)
			}
		}
	case AIInstrStay:
		if mm != nil {
			if _, ok := o.Class().Prop("AIMotioning"); ok {
				_ = o.Set("AIMotioning", int32(0))
			}
		}
	case AIInstrBack:
		if mm != nil && mo != nil {
			mx, _, mz := mo.Pos()
			mm.MoveTo(k, npc, mx, mz)
		}
	}
}

// Cmds 命令流水（测试面）。
func (m *AIOnCommandMotion) Cmds() []string { return m.cmds }

// ---- AIChatWithOther（Init 序 44）----

// AIChatWithOther 复刻官方 AIChatWithOther（Init @0x5C110；符号 167；
// 无注册面）——NPC 说话引擎：event_talk_rec（间隔/台词id）台词泵 +
// AI_Chat_rec 说话面。
type AIChatWithOther struct {
	BaseModule

	// chats 说话流水（测试面：npc → "台词@对象"）。
	chats []string
	// next 游标（npc → 行号）。
	next map[PersistID]int
}

// Name 模块名。
func (m *AIChatWithOther) Name() string { return "AIChatWithOther" }

// Init 注册期（官方无注册面——API 泵模块）。
func (m *AIChatWithOther) Init(k *Kernel) error {
	m.next = make(map[PersistID]int)
	return nil
}

// Say 推一句台词（event_talk_rec 行序；对象写入 AI_Chat_rec）。
func (m *AIChatWithOther) Say(k *Kernel, npc, to PersistID) int32 {
	o := k.GetGameObj(npc)
	if o == nil {
		return -1
	}
	r := o.Record("event_talk_rec")
	if r == nil || r.Rows() == 0 {
		return -1
	}
	i := m.next[npc] % r.Rows()
	m.next[npc] = (i + 1) % r.Rows()
	line := r.Row(i).Int(1)
	if cr := o.Record("AI_Chat_rec"); cr != nil {
		if cr.Rows() == 0 {
			cr.AppendRow(VarList{to, fmt.Sprint(line), int32(i)})
		} else {
			cr.ModifyRow(0, VarList{to, fmt.Sprint(line), int32(i)}) // 单槽（maxrows=1）：当前话覆盖
		}
	}
	m.chats = append(m.chats, fmt.Sprintf("%d:%d@%d", npc.ID(), line, to.ID()))
	return line
}

// Chats 说话流水（测试面）。
func (m *AIChatWithOther) Chats() []string { return m.chats }

// ---- TempStopAI（Init 序 48）----

// TempStopAI 复刻官方 TempStopAI（Init @0x3F20C0；符号 19；无注册面）——
// 临时停 AI：StopAI 置位 + 到时恢复（引擎侧恢复时刻——schema 无此属性）。
type TempStopAI struct {
	BaseModule

	// resumes 恢复时刻（npc → unix 秒）。
	resumes map[PersistID]int64
	// ops 流水（测试面）。
	ops []string
}

// Name 模块名。
func (m *TempStopAI) Name() string { return "TempStopAI" }

// Init 注册期（官方无注册面——API 模块）。
func (m *TempStopAI) Init(k *Kernel) error {
	m.resumes = make(map[PersistID]int64)
	return nil
}

// TempStop 临时停 AI（seconds 秒后恢复；AI 泵面经 EnsureResumed 门）。
func (m *TempStopAI) TempStop(k *Kernel, npc PersistID, seconds int32) {
	o := k.GetGameObj(npc)
	if o == nil {
		return
	}
	if _, ok := o.Class().Prop("StopAI"); ok {
		_ = o.Set("StopAI", int32(1))
	}
	m.resumes[npc] = unixNow() + int64(seconds)
	m.ops = append(m.ops, fmt.Sprintf("stop:%d", npc.ID()))
}

// EnsureResumed 恢复检查（各 AI 泵每拍先过此门；到期 → StopAI=0 清记）。
func (m *TempStopAI) EnsureResumed(k *Kernel, npc PersistID) {
	deadline, stopped := m.resumes[npc]
	if !stopped || unixNow() < deadline {
		return
	}
	delete(m.resumes, npc)
	if o := k.GetGameObj(npc); o != nil {
		if _, ok := o.Class().Prop("StopAI"); ok {
			_ = o.Set("StopAI", int32(0))
		}
	}
	m.ops = append(m.ops, fmt.Sprintf("resume:%d", npc.ID()))
}

// Ops 流水（测试面）。
func (m *TempStopAI) Ops() []string { return m.ops }

// ---- 协作取用面 ----

func aiModuleOf(k *Kernel) *AIModule {
	am, _ := k.GetLogicModule("AIModule").(*AIModule)
	return am
}

func hateModuleOf(k *Kernel) *HateModule {
	hm, _ := k.GetLogicModule("HateModule").(*HateModule)
	return hm
}

func aiMotionOf(k *Kernel) *AIMotionModule {
	mm, _ := k.GetLogicModule("AIMotionModule").(*AIMotionModule)
	return mm
}

func tempStopAIOf(k *Kernel) *TempStopAI {
	ts, _ := k.GetLogicModule("TempStopAI").(*TempStopAI)
	return ts
}

// isAiClass 对象类是否 AI 类族（含子类——AttackNpc→CommonNpc 等）。
func isAiClass(k *Kernel, id PersistID) bool {
	o := k.GetGameObj(id)
	if o == nil {
		return false
	}
	for c := o.ClassName(); c != ""; {
		for _, want := range aiNpcClasses {
			if c == want {
				return true
			}
		}
		cd := k.Class(c)
		if cd == nil {
			return false
		}
		c = cd.Parent
	}
	return false
}
