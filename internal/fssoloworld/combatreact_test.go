package fssoloworld

// combatreact_test.go 命中簇的受击侧帧：受击动作 / 受击特效（Stagger 拍 +
// 技能事件包）、连击计数与 MaxHits、击杀簇的摘 buff 与 m24 13。帧序依据见
// combatreact.go 与 hitNPC 的注释（女号包 208.229 / 357.896 / 358.469 /
// 730.436、挑战包 442.392）。

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/script"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"
)

// castAs 让玩家处于「技能出招中」并挂好技能对象（同 hookCast，技能可指定）。
func (f *combatFixture) castAs(t *testing.T, skill string) (skilldata.Definition, skillCast, int) {
	t.Helper()
	def, ok := f.s.skillDefinition(f.pl, skill)
	if !ok {
		t.Fatalf("技能 %s 定义缺席", skill)
	}
	st := &f.pl.combat
	st.mu.Lock()
	st.curSkill, st.castSeq = def.ID, st.castSeq+1
	st.skillObj = f.s.registerAux(&auxObject{kind: script.HelperSkill, config: def.ID, script: def.LuaScript,
		def: def, owner: playerHandle(f.pl), props: map[string]script.Value{}, records: map[string][][]script.Value{}}).ID
	seq := st.castSeq
	st.mu.Unlock()
	return def, skillCast{skill: def.ID, x: 100, y: 5, z: 200, orient: 1.571, targetID: f.monster.spawn.ObjectID,
		targetOwner: f.monster.spawn.OwnerID, hasTarget: true}, seq
}

// hit17 对怪物的 m24 17 帧：伤害取怪物实际掉血，第 5 值 = 技能静态包的
// DamageType（五个抓包 36 个技能逐一吻合：任务技能 musejuben 族 = 3、外功 = 0）。
func (f *combatFixture) hit17(skill, class string, dealt int) string {
	damageType := 0
	if def, ok := f.s.skillDefinition(f.pl, skill); ok {
		damageType = def.DamageType
	}
	return fmt.Sprintf("m24(npc,i:17,i:%d,i:0,i:%d,s:%s,s:%s,s:test_monster,s:,i:0)", dealt, damageType, skill, class)
}

func wantText(parts []string) string { return strings.Join(parts, " | ") }

// Stagger 拍（cqgf04 判定帧 1280ms，硬直触发器不带受击者动画）的受击帧：
// 首次交手 m24 17 → m18 hurt_half → MaxHits=1 → m24 40 body_hit_3_self
// （抓包 357.896）；已交手的怪再挨打只发 m24 40（384.503 / 388.619），
// 连击超窗归零后首击不推 MaxHits（平纪录不推）。
func TestCombatStaggerBeatHitReaction(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "400"})
	cast := func() {
		f.s.handleSkillCast(f.pl, castMessage(areaSkill, 100, 5, 200, 1.571, nil, 0, 0, 0))
		f.take()
	}
	cast()
	hp := f.hp()
	got := f.pump(1280 * time.Millisecond)
	want := wantText([]string{"view41.1:CurTargetCount=1", "me:FP=2", "me:SP=2", f.aggro(),
		f.hit17(areaSkill, "SkillNormal", hp-f.hp()), "m18(npc,s:hurt_half,i:0,me)", "me:MaxHits=1",
		"m24(npc,i:40,s:body_hit_3_self,me)", "m205(me,s:" + areaSkill + ",npc)", "me:CurSkillID="})
	if wantText(got) != want {
		t.Fatalf("首次交手的受击帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
	// 冷却（3000ms）与连击窗（2000ms）都过去：脱战帧在这一次驱动里落地。
	f.pump(3200 * time.Millisecond)
	cast()
	hp = f.hp()
	got = f.pump(1280 * time.Millisecond)
	want = wantText([]string{"view41.1:CurTargetCount=1", "me:FP=2", "me:SP=2", "me:rec114[0.1]=T", "me:LogicState=1",
		f.hit17(areaSkill, "SkillNormal", hp-f.hp()), "m24(npc,i:40,s:body_hit_3_self,me)", "m205(me,s:" + areaSkill + ",npc)", "me:CurSkillID="})
	if wantText(got) != want {
		t.Fatalf("再次命中的受击帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
}

// 不在技能事件包复原表里的 Stagger 拍技能（任务技能 musejuben_2，判定帧
// 533/967ms，每击 150 点）：每击 m18 hurt_half、无特效（抓包 730.436 / 730.885）。
func TestCombatDefaultStaggerSendsHurtActionEveryHit(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "2000"})
	const skill = "skill_musejuben_2"
	def, cast, seq := f.castAs(t, skill)
	plan := hitPlan{times: []time.Duration{533 * time.Millisecond, 967 * time.Millisecond}, stagger: []bool{true, true}, hurt: 2}
	hp := f.hp()
	f.s.resolveSkillHit(f.pl, def, cast, plan, 1, seq)
	got := combatText(f.s, f.pl, f.monster, f.take())
	want := wantText([]string{"view41.1:CurTargetCount=1", "me:FP=2", "me:SP=2", f.aggro(),
		f.hit17(skill, "SkillNormal", hp-f.hp()), "m18(npc,s:hurt_half,i:0,me)", "me:MaxHits=1",
		"m205(me,s:" + skill + ",npc)"})
	if wantText(got) != want {
		t.Fatalf("第 1 击帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
	hp = f.hp()
	f.s.resolveSkillHit(f.pl, def, cast, plan, 2, seq)
	got = combatText(f.s, f.pl, f.monster, f.take())
	want = wantText([]string{"me:FP=4", "me:SP=4", "me:rec114[0.1]=T",
		f.hit17(skill, "SkillNormal", hp-f.hp()), "m18(npc,s:hurt_half,i:0,me)", "m24(me,i:10,i:2)", "me:MaxHits=2",
		"m205(me,s:" + skill + ",npc)", "me:CurSkillID="})
	if wantText(got) != want {
		t.Fatalf("第 2 击帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
}

// 连击跨招延续：cqgf04 的命中 0.8s 后接普攻，连击从 2 起算（女号包 208.741
// → 210.666：cqgf01 三击后 1.9s 接 cqgf04，连击 3 → 4）。
func TestCombatComboCarriesAcrossCasts(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "400"})
	f.s.handleSkillCast(f.pl, castMessage(areaSkill, 100, 5, 200, 1.571, nil, 0, 0, 0))
	f.take()
	f.pump(1280 * time.Millisecond)
	f.pump(320 * time.Millisecond)
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	f.take()
	got := f.pump(467 * time.Millisecond)
	want := wantText([]string{"view41.1:CurTargetCount=1", "me:FP=4", "me:SP=4", "me:rec114[0.1]=T", hitNormal,
		"m24(me,i:10,i:2)", "me:MaxHits=2", "m205(me,s:zs_normal_00,npc)"})
	if wantText(got) != want {
		t.Fatalf("接招首击帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
	got = f.pump(533 * time.Millisecond)
	want = wantText([]string{"me:FP=6", "me:SP=6", "me:rec114[0.1]=T", hitNormal,
		"m24(me,i:10,i:3)", "me:MaxHits=3", "m205(me,s:zs_normal_00,npc)", "me:CurSkillID="})
	if wantText(got) != want {
		t.Fatalf("接招末击帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
}

// 连击超窗归零：两击间隔超过 2s 后重新从 1 计，首击不发 m24 10；连击数
// 没超过既有最高纪录时不推 MaxHits。
func TestCombatComboResetsAfterWindow(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "400"})
	cast := func() {
		f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
		f.take()
	}
	cast()
	f.pump(time.Second)
	f.pump(3200 * time.Millisecond) // 冷却（1200ms）、连击窗（2000ms）、脱战（3000ms）都过去
	cast()
	got := f.pump(467 * time.Millisecond)
	want := wantText([]string{"view41.1:CurTargetCount=1", "me:FP=2", "me:SP=2", "me:rec114[0.1]=T", "me:LogicState=1",
		hitNormal, "m205(me,s:zs_normal_00,npc)"})
	if wantText(got) != want {
		t.Fatalf("超窗后的首击帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
	got = f.pump(533 * time.Millisecond)
	want = wantText([]string{"me:FP=4", "me:SP=4", "me:rec114[0.1]=T", hitNormal,
		"m24(me,i:10,i:2)", "m205(me,s:zs_normal_00,npc)", "me:CurSkillID="})
	if wantText(got) != want {
		t.Fatalf("超窗后的末击帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
}

// 被闪避打断连击：锁定技命中一只怪（连击 1），紧接着对高闪避的怪落空（连击
// 归零）；此后的命中从 1 重新起算（挑战包 442.392：3 4 5 → 被闪避 → 1 2）。
func TestCombatDodgeBreaksCombo(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "400"})
	catalog := f.s.sceneNPCCatalog(f.scene)
	dodger := npcfeed.Spawn{ObjectID: 0x0E000101, OwnerID: 0x1000, X: 102, Y: 5, Z: 200,
		ConfigID: "test_dodger", Script: "AttackNpc", Template: map[string]string{},
		Combat: &npcfeed.CombatRow{ID: "test_dodge", Attrs: map[string]string{"MaxLife": "400", "Level": "5",
			"MeleeWeaponDamageBase": "10", "MeleeWeaponDamageRand": "10", "Dodge": "100000", "PhysicsAttackHit": "50"}}}
	catalog.combatMu.Lock()
	catalog.spawns = append(catalog.spawns, dodger)
	catalog.combatMu.Unlock()
	one := hitPlan{times: []time.Duration{0}, hurt: 1}

	def, cast, seq := f.castAs(t, rushSkill)
	f.s.resolveSkillHit(f.pl, def, cast, one, 1, seq)
	if got := wantText(combatText(f.s, f.pl, f.monster, f.take())); !strings.Contains(got, "me:MaxHits=1") {
		t.Fatalf("首击应记连击 1: %s", got)
	}
	def, cast, seq = f.castAs(t, rushSkill)
	cast.targetID, cast.targetOwner = dodger.ObjectID, dodger.OwnerID
	// 随机源拨到 0：命中判定落空（同 TestCombatDodgeEndsCastWithoutDamage）。
	roll := combatRoll
	combatRoll = func() float64 { return 0 }
	t.Cleanup(func() { combatRoll = roll })
	f.s.resolveSkillHit(f.pl, def, cast, one, 1, seq)
	combatRoll = roll
	got := wantText(combatText(f.s, f.pl, f.monster, f.take()))
	if !strings.Contains(got, "m24(o:234881281/4096,i:7,i:1,me,s:"+rushSkill+",s:test_dodger,s:,i:100)") {
		t.Fatalf("高闪避的怪应落空: %s", got)
	}

	def, cast, seq = f.castAs(t, basicAttack)
	f.s.resolveSkillHit(f.pl, def, cast, one, 1, seq)
	got = wantText(combatText(f.s, f.pl, f.monster, f.take()))
	if strings.Contains(got, "m24(me,i:10,") || strings.Contains(got, "me:MaxHits=") {
		t.Fatalf("被闪避后首次命中应从 1 起算（不发 m24 10 / MaxHits）: %s", got)
	}
	hp := f.hp()
	_, cast, seq = f.castAs(t, basicAttack)
	f.s.resolveSkillHit(f.pl, def, cast, one, 1, seq)
	got = wantText(combatText(f.s, f.pl, f.monster, f.take()))
	want := wantText([]string{"view41.1:CurTargetCount=1", "me:FP=6", "me:SP=6", "me:rec114[0.1]=T",
		f.hit17(basicAttack, "SkillNormal", hp-f.hp()), "m24(me,i:10,i:2)", "me:MaxHits=2",
		"m205(me,s:zs_normal_00,npc)", "me:CurSkillID="})
	if got != want {
		t.Fatalf("被闪避后的第 2 次命中帧序不符:\n got: %s\nwant: %s", got, want)
	}
}

// 击杀摘掉受击者身上的 buff：摘帧在 m24 17 之后、关系行删之前（抓包
// 358.469 摘 buf_CS_jh_cqgf01）；普攻击杀不带 m24 13（228.138）。致死的是
// 普攻首击，收招帧随之后的末击落地，不在这批里。
func TestCombatKillStripsBuffs(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "52"})
	f.s.handleSkillCast(f.pl, castMessage(rushSkill, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	f.take()
	f.pump(900 * time.Millisecond)  // 三击 48 点 + 末击挂 buf_CS_jh_cqgf01（5000ms）
	f.pump(1600 * time.Millisecond) // 连击窗（2000ms）过去：此后的命中从 1 起算
	if buffs := f.s.buffsOf(npcHandle(f.monster)); len(buffs) != 1 {
		t.Fatalf("锁定技后怪物应带 1 个 buff: %+v", buffs)
	}
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	f.take()
	hp := f.hp()
	got := f.pump(467 * time.Millisecond)
	want := wantText([]string{"view41.1:CurTargetCount=1", "me:FP=8", "me:SP=8", "me:rec114[0.1]=T",
		f.hit17(basicAttack, "SkillNormal", hp-f.hp()),
		"m169(i:1,s:buf_CS_jh_cqgf01,npc,me,s:test_monster,s:,i:T64,i:73)", "me:rec114-row0",
		"m24(npc,i:29,me,s:test_monster,s:)", "m18(npc,s:hurt_half,i:0,me)", "m205(me,s:zs_normal_00,npc)"})
	if wantText(got) != want {
		t.Fatalf("击杀帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
	if f.hp() != 0 || !f.monster.isDead() {
		t.Fatalf("怪物应已死亡: HP=%d", f.hp())
	}
	// buff 已在击杀时摘掉：到期时刻不再发摘帧。
	if got = f.pump(4500 * time.Millisecond); strings.Contains(wantText(got), "m169(") {
		t.Fatalf("击杀后 buff 到期不应再发摘帧: %s", wantText(got))
	}
	if buffs := f.s.buffsOf(npcHandle(f.monster)); len(buffs) != 0 {
		t.Fatalf("击杀后怪物不应带 buff: %+v", buffs)
	}
}

// 技能事件包带死亡标记的技能（musejuben_7）：击杀簇末尾另发 m24 13（抓包
// 820.649）。
func TestCombatDeathMarkSkillEmitsCorpseState(t *testing.T) {
	const skill = "skill_musejuben_7"
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "50"})
	def, cast, seq := f.castAs(t, skill)
	f.s.resolveSkillHit(f.pl, def, cast, hitPlan{times: []time.Duration{0}, hurt: 50}, 1, seq)
	got := wantText(combatText(f.s, f.pl, f.monster, f.take()))
	want := wantText([]string{"view41.1:CurTargetCount=1", "me:FP=2", "me:SP=2", f.aggro(),
		f.hit17(skill, "SkillLock", 50), "me:rec114-row0", "m24(npc,i:29,me,s:test_monster,s:)",
		"m18(npc,s:hurt_half,i:0,me)", "me:MaxHits=1", "m24(npc,i:13)", "m205(me,s:" + skill + ",npc)",
		"me:CurSkillID=", "me:CurSkillTarget=o:0/0", "m189(me)"})
	if got != want {
		t.Fatalf("死亡标记技能的击杀帧序不符:\n got: %s\nwant: %s", got, want)
	}
}

// 多目标一拍：得怒与连击都按受击目标计——每个目标一对 FP/SP（+2）、一段
// 自己的命中簇，连击逐目标 +1（抓包 730.436：一拍五目标五对 FP/SP，
// m24 10 连发 5…9），m205 收全部目标。
func TestCombatMultiTargetRageAndComboPerTarget(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "400"})
	catalog := f.s.sceneNPCCatalog(f.scene)
	other := npcfeed.Spawn{ObjectID: 0x0E000101, OwnerID: 0x1000, X: 102, Y: 5, Z: 200,
		ConfigID: "test_monster", Script: "AttackNpc", Template: map[string]string{},
		Combat: &npcfeed.CombatRow{ID: "test", Attrs: map[string]string{"MaxLife": "400", "Level": "5",
			"MeleeWeaponDamageBase": "10", "MeleeWeaponDamageRand": "10", "Dodge": "3", "PhysicsAttackHit": "50"}}}
	catalog.combatMu.Lock()
	catalog.spawns = append(catalog.spawns, other)
	catalog.combatMu.Unlock()
	f.s.handleSkillCast(f.pl, castMessage(areaSkill, 100, 5, 200, 1.571, nil, 0, 0, 0))
	f.take()
	got := f.pump(1280 * time.Millisecond)
	const o = "o:234881281/4096"
	want := wantText([]string{"view41.1:CurTargetCount=2",
		"me:FP=2", "me:SP=2", "npc:LogicState=1", "me:rec114+row0{c0=obj(234881280/4096),c1=T}", "me:LogicState=1",
		"m24(npc,i:37,s:npc_foundplayer)", f.hit17(areaSkill, "SkillNormal", 63), "m18(npc,s:hurt_half,i:0,me)",
		"me:MaxHits=1", "m24(npc,i:40,s:body_hit_3_self,me)",
		"me:FP=4", "me:SP=4", o + ":LogicState=1", "me:rec114+row1{c0=obj(234881281/4096),c1=T}",
		"m24(" + o + ",i:37,s:npc_foundplayer)", "m24(" + o + ",i:17,i:63,i:0,i:0,s:" + areaSkill + ",s:SkillNormal,s:test_monster,s:,i:0)",
		"m18(" + o + ",s:hurt_half,i:0,me)", "m24(me,i:10,i:2)", "me:MaxHits=2",
		"m24(" + o + ",i:40,s:body_hit_3_self,me)",
		"m205(me,s:" + areaSkill + ",npc," + o + ")", "me:CurSkillID="})
	if wantText(got) != want {
		t.Fatalf("多目标命中帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
}
