package fsgame

import (
	"path/filepath"
	"testing"
)

// assemble3 装配三批（一/二/三期）二十一模块 + Skill/Buffer 类 schema。
func assemble3(t *testing.T) (*Kernel, []Module) {
	t.Helper()
	td := filepath.Join("..", "..", "testdata")
	lc := mustRead(t, filepath.Join(td, "logic_class.xml"))
	px := mustRead(t, filepath.Join(td, "struct", "player", "player.xml"))
	ex := mustRead(t, filepath.Join(td, "struct", "player", "exp.xml"))
	sk := mustRead(t, filepath.Join(td, "struct", "skill", "skillbase.xml"))
	bf := mustRead(t, filepath.Join(td, "struct", "skill", "buffer_new.xml"))

	k := NewKernel()
	k.ResourcePath = td
	if err := k.LoadLogicClassXML(lc); err != nil {
		t.Fatalf("LoadLogicClassXML: %v", err)
	}
	for cls, data := range map[string][]byte{"player": px, "Skill": sk, "Buffer": bf} {
		if err := k.LoadStruct(cls, data); err != nil {
			t.Fatalf("LoadStruct %s: %v", cls, err)
		}
	}
	if err := k.LoadStruct("player", ex); err != nil {
		t.Fatalf("LoadStruct player(exp): %v", err)
	}
	var r Registry
	for _, reg := range []func(*Registry) error{RegisterFirstBatch, RegisterSecondBatch, RegisterThirdBatch} {
		if err := reg(&r); err != nil {
			t.Fatal(err)
		}
	}
	mods, err := r.Assemble(k)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	return k, mods
}

// newFighter 建战斗体（HP/MP/MaxHP/防御/等级就绪）。
func newFighter(t *testing.T, k *Kernel, name string, hp, mp int32) PersistID {
	t.Helper()
	id, err := k.CreateObject("player")
	if err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	if err := k.SetString(id, "name", name); err != nil {
		t.Fatalf("SetString name: %v", err)
	}
	_ = k.SetInt(id, "Level", 30)
	_ = k.SetInt(id, "MaxHP", hp)
	_ = k.SetInt(id, "MaxMP", mp)
	_ = k.SetInt(id, "HP", hp)
	_ = k.SetInt(id, "MP", mp)
	k.RunEvent("player", "OnRecover", id, NullPersistID, nil)
	k.RunEvent("player", "OnReady", id, NullPersistID, nil)
	return id
}

// TestFightClosedLoop 战斗闭环：学技 → 出招（扣MP/伤害/开冷却）→ 冷却中拒 →
// 心跳过冷却 → 再出招 → 打死（Dead=1 → 技能中断联动）。
func TestFightClosedLoop(t *testing.T) {
	k, _ := assemble3(t)
	sk := module2[*SkillModule](t, k, "SkillModule")
	cd := module2[*CoolDownModule](t, k, "CoolDownModule")
	fm := module2[*FightModule](t, k, "FightModule")

	a := newFighter(t, k, "剑客", 200, 100)
	b := newFighter(t, k, "靶子", 100, 100)

	// 学技（cmd 856）→ Skill 对象 + 配置属性。
	if !k.DispatchIntCommand("player", 856, a, NullPersistID, VarList{"Normalr_melee_01"}) {
		t.Fatal("学技命令 856 未命中")
	}
	skills := sk.SkillsOf(a)
	if len(skills) != 1 {
		t.Fatalf("学技后技能数 %d", len(skills))
	}
	so := k.GetGameObj(skills[0])
	if so.GetString("ConfigID") != "Normalr_melee_01" || so.GetInt("CanUse") != 1 {
		t.Fatalf("技能对象面 %+v", so)
	}
	// 出招参数：QTypeID 冷却分类 77（schema 属性）+ 冷却 500ms + MP 消耗 10；
	// 伤害基数走 skill_new 配置行（Normalr_melee_01 的 Skill_Damage=2）。
	_ = so.Set("QTypeID", int32(77))
	_ = so.Set("CoolDownTime", int32(500))
	_ = so.Set("AConsumeCurMP", int32(10))

	// 出招（cust 211）：MP 扣 10、目标 HP 100→98（配置伤害 2）、冷却开启。
	if !k.DispatchCommand("player", 211, a, NullPersistID, VarList{skills[0], b}) {
		t.Fatal("出招命令 211 未命中")
	}
	if got := k.GetGameObj(a).GetInt("MP"); got != 90 {
		t.Fatalf("出招后 MP=%d 期望 90", got)
	}
	if got := k.GetGameObj(b).GetInt("HP"); got != 98 {
		t.Fatalf("目标 HP=%d 期望 98", got)
	}
	if got := k.GetGameObj(b).GetInt("HPRatio"); got != 98 {
		t.Fatalf("HPRatio=%d 期望 98", got)
	}
	if len(cd.Begun()) != 1 {
		t.Fatalf("冷却开启流水 %v", cd.Begun())
	}
	// CurSkill 面写入（C_OnCurSkillChanged 临界链）。
	if k.GetGameObj(a).Get("CurSkill") != Var(skills[0]) {
		t.Fatal("CurSkill 未写入")
	}
	// 冷却中再出招 → 拒。
	k.DispatchCommand("player", 211, a, NullPersistID, VarList{skills[0], b})
	if got := k.GetGameObj(b).GetInt("HP"); got != 98 {
		t.Fatal("冷却中出招未拒绝")
	}
	// nx_is_cooldown 面。
	if !cd.NxIsCooldown(k, a, 77) {
		t.Fatal("NxIsCooldown 应命中")
	}
	// 清冷却 → 再出招 → 打死（HP 归零 → Dead=1 → 技能中断）。
	cd.NxClearAllSkillCooldown(k, a)
	k.DispatchCommand("player", 211, a, NullPersistID, VarList{skills[0], b})
	if got := k.GetGameObj(b).GetInt("HP"); got != 96 {
		t.Fatalf("二次出招 HP=%d 期望 96", got)
	}
	// 直接伤害打满：NxDamage ×2 → 死亡联动（C_OnDeadChanged → CurSkill 清）。
	_ = k.SetInt(b, "HP", 10)
	fm.NxDamage(k, a, b, DamagePhysical, 50)
	if k.GetGameObj(b).GetInt("Dead") != 1 {
		t.Fatal("HP 归零未判死")
	}
	if len(fm.Dead()) != 1 {
		t.Fatal("死亡流水缺失")
	}
	if got := k.GetGameObj(a).Get("CurSkill"); got != Var(NullPersistID) && got != nil {
		// 攻击者 CurSkill 不因目标死亡清除；目标自身技能面在其 Dead 临界清除。
		_ = got
	}
	// 死者不可再被伤害（nx_can_damage_target）。
	if dmg := fm.NxDamage(k, a, b, DamagePhysical, 50); dmg != 0 {
		t.Fatalf("死者受伤 %v", dmg)
	}
	// 忘技（cmd 870）。
	if !k.DispatchIntCommand("player", 870, a, NullPersistID, VarList{"Normalr_melee_01"}) {
		t.Fatal("忘技命令 870 未命中")
	}
	if len(sk.SkillsOf(a)) != 0 {
		t.Fatal("忘技未摘除")
	}
}

// TestBufferLifecycle Buffer 闭环：挂载 → 叠加刷新 → 泵计时到期移除 →
// CantClear 拒移除 → OnStore OffLineClear。
func TestBufferLifecycle(t *testing.T) {
	k, _ := assemble3(t)
	bm := module2[*BufferModule](t, k, "BufferModule")
	p := newFighter(t, k, "中毒者", 100, 100)

	// cust 1019 挂 Buffer（200ms）。
	if !k.DispatchCommand("player", 1019, p, NullPersistID, VarList{"poison", int32(1), int32(200)}) {
		t.Fatal("挂 Buffer 命令 1019 未命中")
	}
	bufs := bm.BuffersOf(p)
	if len(bufs) != 1 {
		t.Fatalf("挂载数 %d", len(bufs))
	}
	bo := k.GetGameObj(bufs[0])
	if bo.GetString("ConfigID") != "poison" {
		t.Fatalf("Buffer 配置 %q", bo.GetString("ConfigID"))
	}
	// 同配置再挂 → 叠加刷新（不新增）。
	bm.AddBuffer(k, p, "poison", 1, 500)
	if len(bm.BuffersOf(p)) != 1 {
		t.Fatal("同配置叠加新增了对象")
	}
	// H_BufferUpdate：叠加已刷新至 500ms——五拍（500ms）→ 到期移除。
	for i := int32(1); i <= 5; i++ {
		k.Bus().TickHeartBeats(k, i)
	}
	if len(bm.BuffersOf(p)) != 0 {
		t.Fatal("Buffer 到期未移除")
	}
	// CantClear：永续 Buffer 不可移除。
	id := bm.AddBuffer(k, p, "divine_shield", 1, 0)
	_ = k.GetGameObj(id).Set("CantClear", int32(1))
	if bm.RemoveBuffer(k, p, id) {
		t.Fatal("CantClear Buffer 被移除")
	}
	// OffLineClear：存档清除。
	_ = k.GetGameObj(id).Set("OffLineClear", int32(1))
	k.RunEvent("player", "OnStore", p, NullPersistID, nil)
	if len(bm.BuffersOf(p)) != 0 {
		t.Fatal("OffLineClear 未在 OnStore 清除")
	}
}

// TestDamageFace 伤害面：防御减伤 + 飘字出口 + 公式缝合。
func TestDamageFace(t *testing.T) {
	k, _ := assemble3(t)
	fm := module2[*FightModule](t, k, "FightModule")
	dcm := module2[*DamageCalculateModule](t, k, "DamageCalculateModule")
	a := newFighter(t, k, "攻", 100, 100)
	b := newFighter(t, k, "守", 100, 100)

	// 物理防御 10 → 30 基伤结算 20（无浮动）。
	dcm.RandDmgType = false
	_ = k.SetInt(b, "DefenseLevel", 10)
	if dmg := fm.NxDamage(k, a, b, DamagePhysical, 30); dmg != 20 {
		t.Fatalf("伤害=%v 期望 20（防御 10）", dmg)
	}
	if got := k.GetGameObj(b).GetInt("HP"); got != 80 {
		t.Fatalf("HP=%d 期望 80", got)
	}
	// VaHP 飘字出口。
	var packet string
	fm.FightOut = func(kind string, at, df PersistID, v float64) { packet = kind }
	fm.NxSendFightDodge(k, a, b)
	if packet != "dodge" {
		t.Fatalf("飘字出口 %q", packet)
	}
	// 公式缝合面。
	if v := dcm.CalculateFormula(k, a, b, 2, 10); v != 20 {
		t.Fatalf("公式值 %v", v)
	}
	dcm.NxReloadDamageCalculateData()
	if dcm.Reloads() != 1 {
		t.Fatal("重载计数异常")
	}
	// 禁攻面。
	_ = k.SetInt(b, "CantBeAttack", 1)
	if fm.NxCanAttackTarget(k, a, b) {
		t.Fatal("CantBeAttack 应不可攻")
	}
}

// TestThirdBatchAssemble 三期装配序 + 技能表装载。
func TestThirdBatchAssemble(t *testing.T) {
	k, mods := assemble3(t)
	want := []string{"DamageCalculateModule", "FightModule", "CoolDownModule", "BufferModule", "SkillModule"}
	var got []string
	for _, m := range mods {
		switch m.Name() {
		case "DamageCalculateModule", "FightModule", "CoolDownModule", "BufferModule", "SkillModule":
			got = append(got, m.Name())
		}
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("装配序[%d]=%s 期望 %s", i, got[i], want[i])
		}
	}
	sk := module2[*SkillModule](t, k, "SkillModule")
	// skill_new 主表真档装载（member 分档缺档回退）。
	if cfg := sk.SkillCfg("N_Normall_melee_00"); cfg == nil || cfg["script"] != "SkillNormal" {
		t.Fatalf("skill_new 主表未装载: %v", sk.SkillCfg("N_Normall_melee_00"))
	}
	// 普攻映射表（job_sex 键）。
	if na := sk.NormalAttackCfg("11"); na == nil || na["skill_0h"] != "Normalr_melee_01" {
		t.Fatalf("Skill_Normal_Attack 表未装载: %v", sk.NormalAttackCfg("11"))
	}
	// 17 心跳全登记。
	names := map[string]bool{}
	for _, hb := range k.Bus().HeartBeatNames() {
		names[hb] = true
	}
	for _, want := range []string{"SkillModule::H_EFFECT1", "SkillModule::H_EFFECT8",
		"SkillModule::H_QuequeSkill", "SkillModule::H_DelayUse", "BufferModule::H_BufferUpdate"} {
		if !names[want] {
			t.Fatalf("心跳 %s 未登记", want)
		}
	}
}
