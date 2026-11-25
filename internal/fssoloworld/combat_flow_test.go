package fssoloworld

// combat_flow_test.go 战斗流程用例：命中节拍与帧序、击杀、闪避、脱战、
// NPC 还手、Lua 钩子、关停开关。数值口径：夹具玩家登录面板
// MeleePower=80、MeleePowerAdd=0，普攻走公式 372 ⌊(5+80×0.1)/2⌋=6，锁定技
// 走公式 237 ⌊(25+80×0.3)/3⌋=16；怪物武器区间 10..20（掷值 20）。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/script"
	"github.com/lizhanyin/9yin-server-go/internal/skilldata"
)

const (
	hitNormal   = "m24(npc,i:17,i:6,i:0,i:0,s:zs_normal_00,s:SkillNormal,s:test_monster,s:,i:0)"
	relationAdd = "me:rec114+row0{c0=obj(%d/%d),c1=T}"
)

// aggro 首次交手簇（抓包 215.577）：怪物进战斗态 → 关系行加行 → 玩家进
// 战斗态 → 发现目标。
func (f *combatFixture) aggro() string {
	return "npc:LogicState=1 | " + fmt.Sprintf(relationAdd, f.monster.spawn.ObjectID, f.monster.spawn.OwnerID) +
		" | me:LogicState=1 | m24(npc,i:37,s:npc_foundplayer)"
}

// 普攻两击：节拍取招式表 zs_normal_00_01 的判定帧（467/1000ms），帧序
// 按抓包 216.675 / 217.239；首击带首次交手簇与得怒，末击收招。
func TestCombatBasicAttackHitsOnZhaoshiBeats(t *testing.T) {
	f := newCombatFixture(t, 1, nil)
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	f.take()

	got := f.pump(467 * time.Millisecond)
	want := "view41.1:CurTargetCount=1 | me:FP=2 | me:SP=2 | " + f.aggro() + " | " + hitNormal +
		" | m205(me,s:zs_normal_00,npc)"
	if strings.Join(got, " | ") != want {
		t.Fatalf("首击帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	if hp := f.hp(); hp != 94 {
		t.Fatalf("首击后怪物 HP = %d, want 94", hp)
	}
	if f.monster.logicState() != logicFighting {
		t.Fatal("受击后怪物应进入战斗态")
	}

	got = f.pump(533 * time.Millisecond)
	want = "me:FP=4 | me:SP=4 | me:rec114[0.1]=T | " + hitNormal +
		" | m24(me,i:10,i:2) | me:MaxHits=2 | m205(me,s:zs_normal_00,npc) | me:CurSkillID="
	if strings.Join(got, " | ") != want {
		t.Fatalf("末击帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	if hp := f.hp(); hp != 88 {
		t.Fatalf("末击后怪物 HP = %d, want 88", hp)
	}
}

// 收招与命中同钟：两次驱动之间真实时间的流逝（机器负载、调度停顿）超过
// 出招锁时长，帧序也不变——收招定时器不走真实时钟。
func TestCombatFramesIgnoreRealTimeStall(t *testing.T) {
	f := newCombatFixture(t, 1, nil)
	def, ok := f.s.skillDefinition(f.pl, basicAttack)
	if !ok {
		t.Fatal("普攻定义缺席")
	}
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	f.take()
	time.Sleep(castLockMS(def) + 200*time.Millisecond)

	got := f.pump(467 * time.Millisecond)
	want := "view41.1:CurTargetCount=1 | me:FP=2 | me:SP=2 | " + f.aggro() + " | " + hitNormal +
		" | m205(me,s:zs_normal_00,npc)"
	if strings.Join(got, " | ") != want {
		t.Fatalf("真实时间流逝后的首击帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
}

// 锁定技直收：判定帧 433/600/900ms（抓包 208.229/208.459/208.741），末击
// 收招后给目标挂 buf_<技能>（m169：+5000ms 到期、Category 73），到期摘除。
func TestCombatLockSkillHitsAndAttachesBuff(t *testing.T) {
	f := newCombatFixture(t, 2, nil)
	f.s.handleSkillCast(f.pl, castMessage(rushSkill, 100, 5, 200, 1.571, f.target(), 102, 5, 200))
	f.take()
	hit := "m24(npc,i:17,i:16,i:0,i:0,s:CS_jh_cqgf01,s:SkillLock,s:test_monster,s:,i:0)"

	got := f.pump(433 * time.Millisecond)
	want := "view41.1:CurTargetCount=1 | me:FP=2 | me:SP=2 | " + f.aggro() + " | " + hit +
		" | m205(me,s:CS_jh_cqgf01,npc)"
	if strings.Join(got, " | ") != want {
		t.Fatalf("首击帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	got = f.pump(167 * time.Millisecond)
	want = "me:FP=4 | me:SP=4 | me:rec114[0.1]=T | " + hit + " | m24(me,i:10,i:2) | me:MaxHits=2 | m205(me,s:CS_jh_cqgf01,npc)"
	if strings.Join(got, " | ") != want {
		t.Fatalf("第 2 击帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	got = f.pump(300 * time.Millisecond)
	want = "me:FP=6 | me:SP=6 | me:rec114[0.1]=T | " + hit + " | m24(me,i:10,i:3) | me:MaxHits=3 | m205(me,s:CS_jh_cqgf01,npc)" +
		" | me:CurSkillID= | m169(i:0,s:buf_CS_jh_cqgf01,npc,me,s:test_monster,s:,i:T64,i:73)" +
		" | me:CurSkillTarget=o:0/0 | m189(me)"
	if strings.Join(got, " | ") != want {
		t.Fatalf("末击帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	if hp := f.hp(); hp != 100-48 {
		t.Fatalf("三击后怪物 HP = %d, want 52", hp)
	}
	// 到期摘除帧（同窗还会有玩家的脱战帧，见 TestCombatCalmDownAfterFight）。
	got = f.pump(5000 * time.Millisecond)
	joined := strings.Join(got, " | ")
	if !strings.Contains(joined, "m169(i:1,s:buf_CS_jh_cqgf01,npc,me,s:test_monster,s:,i:T64,i:73)") {
		t.Fatalf("buff 到期应发摘除帧: %v", got)
	}
	if len(f.s.buffsOf(npcHandle(f.monster))) != 0 {
		t.Fatal("buff 到期后应已摘除")
	}
}

// 击杀：末击致死走击杀簇（关系行删 → m24 29 → m18 → m24 13），尸体停留
// 后重生回满血。
func TestCombatKillEmitsDeathCluster(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "12"})
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	f.take()
	f.pump(467 * time.Millisecond)

	got := f.pump(533 * time.Millisecond)
	want := "me:FP=4 | me:SP=4 | me:rec114[0.1]=T | " + hitNormal +
		" | me:rec114-row0 | m24(npc,i:29,me,s:test_monster,s:) | m18(npc,s:hurt_half,i:0,me) | m24(npc,i:13)" +
		" | m24(me,i:10,i:2) | me:MaxHits=2 | m205(me,s:zs_normal_00,npc) | me:CurSkillID="
	if strings.Join(got, " | ") != want {
		t.Fatalf("击杀帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	if !f.monster.isDead() || f.hp() != 0 {
		t.Fatal("击杀后怪物应死亡且 HP 归零")
	}
	f.pump(corpseKeep)
	f.pump(npcRespawnDelay)
	if f.monster.isDead() || f.hp() != 12 {
		t.Fatalf("重生后应满血存活: dead=%t hp=%d", f.monster.isDead(), f.hp())
	}
}

// 闪避：怪物闪避远高于命中 → m24 7，施放就此收束、不计得怒、不掉血。
func TestCombatDodgeEndsCastWithoutDamage(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"Dodge": "1000"})
	roll := combatRoll
	combatRoll = func() float64 { return 0 }
	t.Cleanup(func() { combatRoll = roll })
	f.s.handleSkillCast(f.pl, castMessage(rushSkill, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	f.take()

	got := f.pump(433 * time.Millisecond)
	want := "view41.1:CurTargetCount=1 | " + f.aggro() +
		" | m24(npc,i:7,i:1,me,s:CS_jh_cqgf01,s:test_monster,s:,i:100) | m20(npc,s:CS_jh_cqgf01_dodge,me)" +
		" | m707(i:14) | me:CurSkillID= | me:CurSkillTarget=o:0/0"
	if strings.Join(got, " | ") != want {
		t.Fatalf("闪避帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	if f.hp() != 100 {
		t.Fatalf("闪避不应掉血: %d", f.hp())
	}
	// 后续击不结算；锁定技的出招锁位到收招定时器才释放（与命中同钟）。
	if got = f.pump(2 * time.Second); strings.Join(got, " | ") != "me:CantMove=0 | me:CantParry=0" {
		t.Fatalf("闪避收束后只应释放出招锁位、后续击不结算: %v", got)
	}
}

// 脱战：最后一次战斗活动 3s 后玩家转警戒态并清 FP/SP，5s 后回常态、
// 关系行删除、怪物同步脱战（抓包 228.138 → 231.152 → 231.303）。
func TestCombatCalmDownAfterFight(t *testing.T) {
	f := newCombatFixture(t, 1, nil)
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	f.take()
	f.pump(467 * time.Millisecond)
	f.pump(533 * time.Millisecond)

	got := f.pump(3000 * time.Millisecond)
	want := "me:FP=0 | me:SP=0 | me:LogicState=2"
	if strings.Join(got, " | ") != want {
		t.Fatalf("脱战帧不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	got = f.pump(2000 * time.Millisecond)
	joined := strings.Join(got, " | ")
	for _, want := range []string{"npc:LogicState=0", "me:LogicState=0", "me:rec114-row0"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("回常态帧缺 %q: %v", want, got)
		}
	}
	if f.monster.logicState() != logicNormal {
		t.Fatal("玩家脱战后怪物应同步脱战")
	}
}

// NPC 还手：受击后经反应延迟起手（CurSkillID/CurSkillTarget/m20），流程包
// HitTime=1000ms 落一击（抓包 217.043 → 218.058 的还手形态）。
func TestCombatNpcRetaliates(t *testing.T) {
	f := newCombatFixture(t, 1, nil)
	t.Setenv("NINEYIN_NPC_AI", "on")
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	f.take()
	f.pump(467 * time.Millisecond)
	f.pump(533 * time.Millisecond)

	got := f.pump(67 * time.Millisecond) // 467 + 反应延迟 600 = 1067ms
	want := "npc:CurSkillID=default_normal_skill | npc:CurSkillTarget=me | loc(npc,101.000,5.000,200.000,0.000)" +
		" | m20(npc,s:atkhit1,me,i:0,f:0.000)"
	if strings.Join(got, " | ") != want {
		t.Fatalf("NPC 起手帧不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	got = f.pump(1000 * time.Millisecond)
	want = "me:SP=5 | me:HPRatio=79 | me:HP=77 | me:rec114[0.1]=T" +
		" | m24(me,i:17,i:20,i:0,i:0,s:default_normal_skill,s:SkillNormal,s:,s:test_monster,i:0)" +
		" | m205(npc,s:default_normal_skill,me) | npc:CurSkillID= | npc:CurSkillTarget=o:0/0 | m189(npc)"
	if strings.Join(got, " | ") != want {
		t.Fatalf("NPC 命中帧不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
}

// luaFixture 落一份临时脚本根并把 NINEYIN_LUA_DIR 指过去：files 为相对根
// 的「路径 → 源」。
func luaFixture(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	for name, source := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("NINEYIN_LUA_DIR", dir)
}

// hookCast 让玩家处于「普攻出招中」并挂上指定脚本的技能对象，返回定义、
// 施放参数与序号——直接驱动 resolveSkillHit 的钩子用例用。
func (f *combatFixture) hookCast(t *testing.T, luaScript string) (skilldata.Definition, skillCast, int) {
	t.Helper()
	def, ok := f.s.skillDefinition(f.pl, basicAttack)
	if !ok {
		t.Fatal("普攻定义缺席")
	}
	def.LuaScript = luaScript
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

// 技能脚本钩子（官方 skill_sample.lua 契约）：get_damage 给出的 5 系伤害
// 走引擎结算管线；skill_after_hit 里经 nx_* 建并挂 buff、读写对象属性。
func TestCombatSkillLuaHooks(t *testing.T) {
	luaFixture(t, map[string]string{
		"define.lua": "TYPE_PLAYER = 2\nTYPE_HELPER = 16\n",
		"skill/test_hook.lua": `require("define")
function skill_before_hit(self, skill)
	if nx_type(self) ~= TYPE_PLAYER or nx_type(skill) ~= TYPE_HELPER then return 0 end
	return 1
end
function get_damage(self, skill, target)
	local power = nx_query_property(self, "MeleePower") + nx_query_property(self, "MeleePowerAdd")
	local level = nx_query_property(skill, "Level")
	return power * level, 0, 0, 0, 0
end
function skill_after_hit(self, skill, target)
	nx_set_property(target, "LogicState", nx_int(1))
	local buffer = nx_create_from_config(nx_get_scene(), "buf_CS_jh_cqgf01")
	if nx_exists(buffer) then
		nx_set_property(buffer, "Level", nx_int(2))
		nx_add_buffer(target, self, buffer)
	end
end
`,
	})
	f := newCombatFixture(t, 1, nil)
	def, cast, seq := f.hookCast(t, `skill\test_hook.lua`)
	f.s.resolveSkillHit(f.pl, def, cast, hitPlan{times: []time.Duration{0}, hurt: 1}, 1, seq)

	got := combatText(f.s, f.pl, f.monster, f.take())
	// nx_add_buffer 在钩子里即时下发 m169 挂载帧；命中簇随后按序落地。
	want := "m169(i:0,s:buf_CS_jh_cqgf01,npc,me,s:test_monster,s:,i:T64,i:73)" +
		" | view41.1:CurTargetCount=1 | me:FP=2 | me:SP=2 | " + f.aggro() +
		" | m24(npc,i:17,i:80,i:0,i:0,s:zs_normal_00,s:SkillNormal,s:test_monster,s:,i:0) | m205(me,s:zs_normal_00,npc)" +
		" | me:CurSkillID="
	if strings.Join(got, " | ") != want {
		t.Fatalf("钩子结算帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	if f.hp() != 20 {
		t.Fatalf("get_damage 给出 80（MeleePower 80 × 等级 1），怪物 HP = %d want 20", f.hp())
	}
	// skill_after_hit 里 nx_set_property(target, "LogicState", 1) 落到怪物上。
	if f.monster.logicState() != logicFighting {
		t.Fatal("skill_after_hit 经 nx_set_property 写入的 LogicState 应落到怪物上")
	}
	// nx_add_buffer 挂上的 buff：脚本没设 LifeTime → 取定义的 5000ms；到期摘除。
	if buffs := f.s.buffsOf(npcHandle(f.monster)); len(buffs) != 1 || buffs[0].id != "buf_CS_jh_cqgf01" || buffs[0].level != 2 {
		t.Fatalf("skill_after_hit 挂的 buff 不符: %+v", buffs)
	}
	got = f.pump(5000 * time.Millisecond)
	// 到期摘除帧（同窗会带玩家的脱战帧，见 TestCombatCalmDownAfterFight）。
	if !strings.Contains(strings.Join(got, " | "), "m169(i:1,s:buf_CS_jh_cqgf01,npc,me,s:test_monster,s:,i:T64,i:73)") {
		t.Fatalf("脚本挂的 buff 到期应发摘除帧: %v", got)
	}
	if len(f.s.buffsOf(npcHandle(f.monster))) != 0 {
		t.Fatal("脚本挂的 buff 到期后应已摘除")
	}
}

// 脚本打断：skill_after_hit 里 nx_break_any_flow(self) 清 CurSkillID 并顶掉
// 施放序号，其后的击不再结算。
func TestCombatScriptBreaksCast(t *testing.T) {
	luaFixture(t, map[string]string{"define.lua": "", "skill/test_break.lua": `require("define")
function skill_after_hit(self, skill, target)
	nx_break_any_flow(self)
end
`})
	f := newCombatFixture(t, 1, nil)
	def, cast, seq := f.hookCast(t, `skill\test_break.lua`)
	plan := hitPlan{times: []time.Duration{467 * time.Millisecond, time.Second}, hurt: 2}
	f.s.resolveSkillHit(f.pl, def, cast, plan, 1, seq)

	// 脚本发起的帧即时下发（同 TestCombatSkillLuaHooks），打断帧先于本击帧簇。
	got := combatText(f.s, f.pl, f.monster, f.take())
	want := "me:CurSkillID= | view41.1:CurTargetCount=1 | me:FP=2 | me:SP=2 | " + f.aggro() + " | " + hitNormal +
		" | m205(me,s:zs_normal_00,npc)"
	if strings.Join(got, " | ") != want {
		t.Fatalf("打断帧序不符:\n got: %s\nwant: %s", strings.Join(got, " | "), want)
	}
	// 第 2 击带的是旧序号：不结算。
	f.s.resolveSkillHit(f.pl, def, cast, plan, 2, seq)
	if left := f.take(); len(left) != 0 || f.hp() != 94 {
		t.Fatalf("打断后后续击不应结算: %v HP=%d", combatText(f.s, f.pl, f.monster, left), f.hp())
	}
}

// buff 脚本钩子（官方 buf_testbuff_03.lua 同款写法）：周期型 buff 每跳
// buffer_timer 取 Sender 经 nx_damage_target 按 get_damage 结算伤害。m24 22
// 到施法者名为止，没有 17 帧那样的末位暴击标记（挑战包 22 帧）；到期摘除
// 后不再跳。
func TestCombatBuffScriptTicksDamage(t *testing.T) {
	luaFixture(t, map[string]string{"define.lua": "", "buf/test_dot.lua": `require("define")
function buffer_timer(self, buffer)
	local sender = nx_query_property(buffer, "Sender")
	nx_damage_target(sender, self, buffer)
	return 1
end
function get_damage(self, buffer, target)
	local buffer_level = nx_query_property(buffer, "Level")
	return 0, 20 + buffer_level * 2, 0, 0, 0
end
`})
	f := newCombatFixture(t, 1, nil)
	aux := f.s.registerAux(&auxObject{kind: script.HelperBuffer, config: "buf_CS_jh_cqgf01", script: `buf\test_dot.lua`,
		owner: playerHandle(f.pl), records: map[string][][]script.Value{},
		props: map[string]script.Value{"Level": script.Int(1), "LifeTime": script.Int(3500),
			"EffectType": script.Int(3), "EffectTimer": script.Int(1000)}})
	if !f.s.attachBuffAux(npcHandle(f.monster), playerHandle(f.pl), aux.ID) {
		t.Fatal("挂 buff 失败")
	}
	if got := f.pump(0); strings.Join(got, " | ") != "m169(i:0,s:buf_CS_jh_cqgf01,npc,me,s:test_monster,s:,i:T64,i:73)" {
		t.Fatalf("挂载应只发 m169 挂帧: %v", got)
	}

	// 1 级：20 + 1×2 = 22；攻击者是 Sender（玩家），100 → 78 → 56 → 34。
	// 每跳处理里才排下一跳，按一跳一步推进时钟（线上泵节拍远小于跳间隔）。
	hit := "m24(npc,i:22,i:22,i:0,i:0,s:buf_CS_jh_cqgf01,s:Buffer,s:test_monster,s:)"
	for n, hp := range []int{78, 56, 34} {
		if got := f.pump(time.Second); strings.Join(got, " | ") != hit || f.hp() != hp {
			t.Fatalf("第 %d 跳 = %v, HP = %d; want [%s], HP %d", n+1, got, f.hp(), hit, hp)
		}
	}
	if got := f.pump(500 * time.Millisecond); strings.Join(got, " | ") !=
		"m169(i:1,s:buf_CS_jh_cqgf01,npc,me,s:test_monster,s:,i:T64,i:73)" {
		t.Fatalf("到期应只发 m169 摘帧: %v", got)
	}
	if got := f.pump(3 * time.Second); len(got) != 0 || f.hp() != 34 {
		t.Fatalf("到期后不应再跳: %v HP=%d", got, f.hp())
	}
}

// 技能钩子里经 nx_damage_target 进 buff 脚本（嵌套调用）后，技能脚本须
// 执行到底且不被禁用。VM 的嵌套调用曾摘掉外层上下文，外层脚本回到主循环
// 即崩溃、被误禁用（internal/script 的 TestNestedCallKeepsOuterContext）。
func TestCombatSkillHookSurvivesNestedBuffCall(t *testing.T) {
	luaFixture(t, map[string]string{"define.lua": "",
		"buf/test_dot.lua": `require("define")
function get_damage(self, buffer, target)
	return 0, 20 + nx_query_property(buffer, "Level") * 2, 0, 0, 0
end
`,
		"skill/test_nested.lua": `require("define")
function skill_after_hit(self, skill, target)
	local buffer = nx_find_buffer(target, "buf_CS_jh_cqgf01")
	nx_damage_target(self, target, buffer)
	nx_set_property(buffer, "Hits", nx_int(nx_query_property(buffer, "Hits") + 1))
end
`})
	f := newCombatFixture(t, 1, nil)
	// 先给怪物挂上带脚本的 buff（不计时），供技能钩子取用。
	dot := f.s.registerAux(&auxObject{kind: script.HelperBuffer, config: "buf_CS_jh_cqgf01", script: `buf\test_dot.lua`,
		owner: playerHandle(f.pl), records: map[string][][]script.Value{},
		props: map[string]script.Value{"Level": script.Int(1), "LifeTime": script.Int(10000), "Hits": script.Int(0)}})
	if !f.s.attachBuffAux(npcHandle(f.monster), playerHandle(f.pl), dot.ID) {
		t.Fatal("挂 buff 失败")
	}
	// 两次出招各一击：钩子里 buff 伤害 22、命中伤害 6。
	for n := 1; n <= 2; n++ {
		def, cast, seq := f.hookCast(t, `skill\test_nested.lua`)
		f.s.resolveSkillHit(f.pl, def, cast, hitPlan{times: []time.Duration{0}, hurt: 2}, 1, seq)
		f.take()
	}
	if got, _ := f.s.auxByHandle(dot.ID).Query("Hits"); got.IntValue() != 2 {
		t.Fatalf("技能钩子应两次都执行到底: Hits = %+v", got)
	}
	if f.hp() != 44 {
		t.Fatalf("两次出招后怪物 HP = %d, want 44（2×（22+6））", f.hp())
	}
}

// 公式引擎全量装载：仓库产物里的每条公式都装成引擎闭包、零跳过——全部
// 伤害包都走公式引擎，没有落回退面的；只有公式缺席才回退。
func TestCombatEngineInstallsAllFormulas(t *testing.T) {
	f := newSkillFixture(t)
	lua := f.s.combatLua()
	if lua.skills == nil || !lua.engine.Enabled() {
		t.Skip("公式引擎产物缺席（go run ./tools/datagen skill）")
	}
	installed, skipped := lua.engine.Installed()
	if want := len(lua.skills.Formulas()); installed != want || skipped != 0 {
		t.Errorf("引擎装载公式 %d 条（跳过 %d）, want %d 条", installed, skipped, want)
	}
	rolls := []float64{0.99, 0.99, 0.99}
	// 普攻公式按官服新号面板（MeleePower 3）复算：⌊(5+0.3)/2⌋=2。
	got := lua.engine.CalcHit(script.HitContext{FormulaID: 372, Params: []string{"5", "1", ""}, HurtCount: 2,
		Attacker: map[string]float64{"MeleePower": 3}, Rand: rolls})
	if got.Fallback || got.Damage != 2 {
		t.Errorf("公式 372 新号面板应得 2: %+v", got)
	}
	// 手修定版的 1116（P1 实参 + 四项强度×0.065）可求值。
	if got := lua.engine.CalcHit(script.HitContext{FormulaID: 1116, Params: []string{"100"}, Rand: rolls}); got.Fallback || got.Damage != 100 {
		t.Errorf("公式 1116 应得 100: %+v", got)
	}
	if got := lua.engine.CalcHit(script.HitContext{FormulaID: 1 << 30, Rand: rolls}); !got.Fallback || !lua.engine.Enabled() {
		t.Errorf("缺席公式应回退且引擎保持可用: %+v", got)
	}
}

// NINEYIN_COMBAT=off：受理链照旧，命中不结算；施放仍按出招锁到期收招
// （否则玩家卡在出招态）。
func TestCombatDisabledSkipsSettlement(t *testing.T) {
	t.Setenv("NINEYIN_COMBAT", "off")
	f := newCombatFixture(t, 1, nil)
	f.s.handleSkillCast(f.pl, castMessage(basicAttack, 100, 5, 200, 1.571, f.target(), 101, 5, 200))
	if len(f.take()) == 0 {
		t.Fatal("关停战斗不应影响受理帧")
	}
	if got := f.pump(2 * time.Second); strings.Join(got, " | ") != "me:CurSkillID=" {
		t.Fatalf("关停后除收招帧外不应有任何帧: %v", got)
	}
	if f.hp() != 100 {
		t.Fatalf("关停后不应掉血: %d", f.hp())
	}
}
