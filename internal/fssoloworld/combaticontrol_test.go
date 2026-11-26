package fssoloworld

// combaticontrol_test.go 任务技能入口（m270）与命中受控：受理帧序与拒绝、
// 施放期间的自身 buff、skill_musejuben_4 的击晕 / 击退 / 打断、位移记录与
// 官服字节对拍。帧序依据见 skill.go（planSkillCast 的 item 分支）与
// combatcontrol.go 的注释（女号包 721~815s）。

import (
	"encoding/hex"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// itemCastMessage 构造 m270 值表 [270, 技能名]。
func itemCastMessage(skill string) entergen.CustomInbound {
	return entergen.CustomInbound{Opcode: 0x0A, Values: []entergen.CustomInboundValue{
		{Type: 2, Int32: clientCustomItemSkill}, {Type: 6, Text: skill}}}
}

// giveSP 让玩家进入战斗态并带上指定战意（任务技能的消耗是 SP）。
func (f *combatFixture) giveSP(sp int) {
	st := &f.pl.combat
	st.mu.Lock()
	f.s.ensureLiveLocked(st, f.pl)
	st.live.sp = sp
	st.mu.Unlock()
}

// stripRows 去掉冷却表行帧（rec109）：受理用例只关心其余帧的次序。
func stripRows(text string) string {
	parts := strings.Split(text, " | ")
	out := parts[:0]
	for _, part := range parts {
		if !strings.HasPrefix(part, "me:rec109") {
			out = append(out, part)
		}
	}
	return strings.Join(out, " | ")
}

// m270 普通类任务技能（musejuben_2，耗 MP 7）：受理帧序 CurSkillID → 冷却行
// → MP → m20，没有 m211 的 0x1F 与 m337（抓包 722.139）；其后每击带受击
// 动作（判定帧 533 / 967ms）。
func TestItemSkillCastNormalSkillHasNoPositionFrames(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "2000"})
	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_2"))
	got := stripRows(wantText(combatText(f.s, f.pl, f.monster, f.take())))
	want := "me:CurSkillID=skill_musejuben_2 | me:MPRatio=87 | me:MP=48 | m20(me,s:CS_jz_xytf03,o:0/0,i:0,f:0.000)"
	if got != want {
		t.Fatalf("m270 受理帧序不符:\n got: %s\nwant: %s", got, want)
	}
	hp := f.hp()
	got = wantText(f.pump(533 * time.Millisecond))
	want = wantText([]string{"view41.1:CurTargetCount=1", "me:FP=2", "me:SP=2", f.aggro(),
		f.hit17("skill_musejuben_2", "SkillNormal", hp-f.hp()), "m18(npc,s:hurt_half,i:0,me)", "me:MaxHits=1",
		"m205(me,s:skill_musejuben_2,npc)"})
	if got != want {
		t.Fatalf("m270 首击帧序不符:\n got: %s\nwant: %s", got, want)
	}
	hp = f.hp()
	got = wantText(f.pump(434 * time.Millisecond))
	want = wantText([]string{"me:FP=4", "me:SP=4", "me:rec114[0.1]=T", f.hit17("skill_musejuben_2", "SkillNormal", hp-f.hp()),
		"m18(npc,s:hurt_half,i:0,me)", "m24(me,i:10,i:2)", "me:MaxHits=2", "m205(me,s:skill_musejuben_2,npc)",
		"me:CurSkillID="})
	if got != want {
		t.Fatalf("m270 末击帧序不符:\n got: %s\nwant: %s", got, want)
	}
}

// m270 锁定类任务技能（musejuben_8，耗 SP 25）：目标回落最近选靶，帧序同
// m211 锁定技直收，但没有 m337、LastWG 位是目标名、m19 技能名位是锁定包
// 的命中动作（抓包 769.504）；命中按该动作的判定帧结算。
func TestItemSkillCastLockSkillUsesLockPackageAction(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "9000"})
	f.giveSP(59)
	st := &f.pl.combat
	st.mu.Lock()
	st.selected = *f.target()
	st.mu.Unlock()
	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_8"))
	got := stripRows(wantText(combatText(f.s, f.pl, f.monster, f.take())))
	want := wantText([]string{"me:CurSkillID=skill_musejuben_8", "me:CurSkillTarget=npc", "m284()", "me:SP=34",
		"loc(me,100.000,5.000,200.000,1.571)", "me:LastWGAttackNpc=test_monster", "m24(npc,i:40,i:1,me)",
		"m19(me,npc,s:CS_slzp_lzs07,i:3666,i:3666,f:104.660,f:5.000,f:200.000,f:1.571,f:4.660,f:106.100," +
			"f:5.000,f:200.000,f:2.617,f:5.100,i:0,f:0.000,i:1,i:1,f:0.000,f:101.000,f:5.000,f:200.000)",
		"me:CantMove=1", "me:CantParry=1"})
	if got != want {
		t.Fatalf("m270 锁定技受理帧序不符:\n got: %s\nwant: %s", got, want)
	}
	// CS_slzp_lzs07 的判定帧 1634 / 2167 / 2767 / 2967 / 3267 / 3667ms 都带受击者
	// 动画（Stagger=0）：命中帧之外不发受击帧（抓包 musejuben_8 4 次命中零受击帧）。
	hp := f.hp()
	got = wantText(f.pump(1634 * time.Millisecond))
	want = wantText([]string{"view41.1:CurTargetCount=1", "me:FP=2", "me:SP=36", f.aggro(),
		f.hit17("skill_musejuben_8", "SkillLock", hp-f.hp()), "me:MaxHits=1", "m205(me,s:skill_musejuben_8,npc)"})
	if got != want {
		t.Fatalf("m270 锁定技首击帧序不符:\n got: %s\nwant: %s", got, want)
	}
}

// m270 的拒绝形态（女号包任务技能段统计）：冷却中 m1[2,"9009"]；出招中
// m707[9033]；锁定技没选靶 m1[2,"9030"] → m707[7]；战意不足 m707[9022]；
// 目标已死亡 m707[9016]。
func TestItemSkillCastRejections(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "9000"})
	f.giveSP(59)
	st := &f.pl.combat
	st.mu.Lock()
	st.selected = *f.target()
	st.mu.Unlock()

	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_2"))
	f.take()
	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_2"))
	if got := wantText(combatText(f.s, f.pl, f.monster, f.take())); got != "m1(i:2,s:9009)" {
		t.Fatalf("冷却中应提示 9009: %s", got)
	}
	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_5"))
	if got := wantText(combatText(f.s, f.pl, f.monster, f.take())); got != "m707(i:9033)" {
		t.Fatalf("出招中应回 9033: %s", got)
	}
	f.pump(time.Second)

	f.pl.combat.mu.Lock()
	f.pl.combat.selected = 0
	f.pl.combat.mu.Unlock()
	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_8"))
	if got := wantText(combatText(f.s, f.pl, f.monster, f.take())); got != "m1(i:2,s:9030) | m707(i:7)" {
		t.Fatalf("锁定技没选靶应提示 9030: %s", got)
	}

	f.giveSP(10)
	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_4"))
	if got := wantText(combatText(f.s, f.pl, f.monster, f.take())); got != "m707(i:9022)" {
		t.Fatalf("战意不足应回 9022: %s", got)
	}

	f.giveSP(59)
	f.pl.combat.mu.Lock()
	f.pl.combat.selected = *f.target()
	f.pl.combat.mu.Unlock()
	f.monster.mu.Lock()
	f.monster.dead = true
	f.monster.mu.Unlock()
	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_8"))
	if got := wantText(combatText(f.s, f.pl, f.monster, f.take())); got != "m707(i:9016)" {
		t.Fatalf("目标已死亡应回 9016: %s", got)
	}
}

// pushPullPlan skill_musejuben_4 的命中计划（命中动作 CS_wd_tjq08 的判定帧
// 333 / 6000 / 7934ms，三拍都是 Stagger 拍）。
func pushPullPlan() hitPlan {
	return hitPlan{times: []time.Duration{333 * time.Millisecond, 6000 * time.Millisecond, 7934 * time.Millisecond},
		stagger: []bool{true, true, true}, action: "CS_wd_tjq08", hurt: 3}
}

// 任务技能 musejuben_4「推—拉」：受理簇带自身霸体（SP 扣 35 → 属性位 →
// BufferInfo3 → m169 buf_fixed_red → m20，抓包 731.152）；首击挂击晕 / 减速
// 与命中硬直 buff 并置硬直态（731.440）；第 2 拍 c56 → c78（N=1）把怪推离
// 到 5.3m、第 3 拍（N=2）拉回身前 0.7m，位移经 AllDest 记录广播；末击收招
// 摘掉自身霸体（738.115）。
func TestQuestPushPullSkillControl(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "9000"})
	f.giveSP(44)
	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_4"))
	got := stripRows(wantText(combatText(f.s, f.pl, f.monster, f.take())))
	want := "me:CurSkillID=skill_musejuben_4 | me:SP=9 | me:CantBeSkillLocked=1 | me:CantBeSkillMustLocked=1 | " +
		"me:BufferInfo3=21EA,1,1,1,C23009C1E20,0 | m169(i:0,s:buf_fixed_red,me,me,s:,s:,i:T64,i:0) | " +
		"m20(me,s:CS_wd_tjq08,o:0/0,i:0,f:0.000)"
	if got != want {
		t.Fatalf("musejuben_4 受理帧序不符:\n got: %s\nwant: %s", got, want)
	}

	def, _ := f.s.skillDefinition(f.pl, "skill_musejuben_4")
	st := &f.pl.combat
	st.mu.Lock()
	st.castSeq++ // 受理时登记的命中 / 收招定时器作废：各拍由本用例直接驱动
	seq := st.castSeq
	st.mu.Unlock()
	cast := skillCast{skill: def.ID, item: true, x: 100, y: 5, z: 200, orient: 1.571}
	plan := pushPullPlan()
	hit := func(beat int) string {
		f.s.resolveSkillHit(f.pl, def, cast, plan, beat, seq)
		return wantText(combatText(f.s, f.pl, f.monster, f.take()))
	}

	hp := f.hp()
	got = hit(1)
	want = wantText([]string{"view41.1:CurTargetCount=1", "me:FP=2", "me:SP=11", f.aggro(),
		f.hit17(def.ID, "SkillNormal", hp-f.hp()), "m18(npc,s:hurt_half,i:0,me)", "me:MaxHits=1",
		"m169(i:0,s:buf_skill_musejuben_41,npc,me,s:test_monster,s:,i:T64,i:134225921)",
		"m169(i:0,s:buf_skill_musejuben_42,npc,me,s:test_monster,s:,i:T64,i:4198405)",
		"npc:State=hurt_stun_2",
		"m169(i:0,s:buf_CS_wd_tjq08,npc,me,s:test_monster,s:,i:T64,i:8201)",
		"m205(me,s:skill_musejuben_4,npc)"})
	if got != want {
		t.Fatalf("首击受控帧序不符:\n got: %s\nwant: %s", got, want)
	}

	hp = f.hp()
	got = hit(2)
	want = wantText([]string{"me:FP=4", "me:SP=13", "me:rec114[0.1]=T", f.hit17(def.ID, "SkillNormal", hp-f.hp()),
		"m18(npc,s:hurt_half,i:0,me)", "m24(me,i:10,i:2)", "me:MaxHits=2",
		"m169(i:0,s:buf_skill_musejuben_41,npc,me,s:test_monster,s:,i:T64,i:134225921)",
		"m169(i:1,s:buf_skill_musejuben_42,npc,me,s:test_monster,s:,i:T64,i:4198405)",
		"m169(i:0,s:buf_skill_musejuben_42,npc,me,s:test_monster,s:,i:T64,i:4198405)",
		"m24(npc,i:56)", "m24(npc,i:78,me,s:,s:,i:1,f:100000000.000)",
		"alldest(npc,101.000,5.000,200.000,o=0.000,spd=2.0,m2=4.0,m3=8.0,f=0x2602)",
		"m205(me,s:skill_musejuben_4,npc)"})
	if got != want {
		t.Fatalf("第 2 拍击退帧序不符:\n got: %s\nwant: %s", got, want)
	}
	// 推离在途：服务端位置还没动；0.85s 后到位，停在离玩家 5.3m 处、朝向转为运动方向。
	if pos := f.npcPos(); pos != [4]float64{101, 5, 200, 0} {
		t.Fatalf("推离在途时服务端位置不应变: %v", pos)
	}
	if got = wantText(f.pump(850 * time.Millisecond)); got != "alldest(npc,105.300,5.000,200.000,o=1.571,spd=0.0,m2=4.0,m3=8.0,f=0xa)" {
		t.Fatalf("推离到位记录不符: %s", got)
	}
	if pos := f.npcPos(); pos[0] < 105.29 || pos[0] > 105.31 || pos[3] < 1.57 || pos[3] > 1.572 {
		t.Fatalf("推离后服务端位置应为 (105.3,5,200) 朝 1.571: %v", pos)
	}

	hp = f.hp()
	got = hit(3)
	want = wantText([]string{"me:FP=6", "me:SP=15", "me:rec114[0.1]=T", f.hit17(def.ID, "SkillNormal", hp-f.hp()),
		"m18(npc,s:hurt_half,i:0,me)", "m24(me,i:10,i:3)", "me:MaxHits=3",
		"m169(i:0,s:buf_skill_musejuben_41,npc,me,s:test_monster,s:,i:T64,i:134225921)",
		"m169(i:1,s:buf_skill_musejuben_42,npc,me,s:test_monster,s:,i:T64,i:4198405)",
		"m169(i:0,s:buf_skill_musejuben_42,npc,me,s:test_monster,s:,i:T64,i:4198405)",
		"m24(npc,i:56)", "m24(npc,i:78,me,s:,s:,i:2,f:100000000.000)",
		"alldest(npc,105.300,5.000,200.000,o=1.571,spd=8.0,m2=7.8,m3=30.4,f=0x2602)",
		"m205(me,s:skill_musejuben_4,npc)", "me:CurSkillID=",
		"m169(i:1,s:buf_fixed_red,me,me,s:,s:,i:T64,i:0)", "me:BufferInfo3=",
		"me:CantBeSkillLocked=0", "me:CantBeSkillMustLocked=0"})
	if got != want {
		t.Fatalf("第 3 拍拉回与收招帧序不符:\n got: %s\nwant: %s", got, want)
	}
	if got = wantText(f.pump(250 * time.Millisecond)); got != "alldest(npc,100.700,5.000,200.000,o=1.571,spd=0.0,m2=7.8,m3=30.4,f=0xa)" {
		t.Fatalf("拉回到位记录不符: %s", got)
	}
	if pos := f.npcPos(); pos[0] < 100.69 || pos[0] > 100.71 {
		t.Fatalf("拉回后服务端位置应在玩家身前 0.7m: %v", pos)
	}
	if buffs := f.s.buffsOf(playerHandle(f.pl)); len(buffs) != 0 {
		t.Fatalf("收招后玩家不应再带自身 buff: %+v", buffs)
	}
}

// 受击者正在出招时被打断：首击 State="hurt_stun_2" → State="" → CurSkillID=""
// → m189 → m24 56（抓包 731.440 的 350001e3）；出招对象清空，硬直接束时
// 决策循环重新接上。
func TestQuestControlInterruptsCastingNpc(t *testing.T) {
	f := newCombatFixture(t, 1, map[string]string{"MaxLife": "9000"})
	f.giveSP(44)
	f.s.handleItemSkillCast(f.pl, itemCastMessage("skill_musejuben_4"))
	f.take()
	def, _ := f.s.skillDefinition(f.pl, "skill_musejuben_4")
	st := &f.pl.combat
	st.mu.Lock()
	seq := st.castSeq
	st.mu.Unlock()
	f.monster.mu.Lock()
	f.monster.cast = &npcCast{skill: def, seq: 7}
	f.monster.mu.Unlock()

	cast := skillCast{skill: def.ID, item: true, x: 100, y: 5, z: 200, orient: 1.571}
	f.s.resolveSkillHit(f.pl, def, cast, pushPullPlan(), 1, seq)
	got := wantText(combatText(f.s, f.pl, f.monster, f.take()))
	want := " | npc:State=hurt_stun_2 | npc:State= | npc:CurSkillID= | m189(npc) | m24(npc,i:56) | " +
		"m169(i:0,s:buf_CS_wd_tjq08,npc,me,s:test_monster,s:,i:T64,i:8201) | m205(me,s:skill_musejuben_4,npc)"
	if !strings.HasSuffix(got, want) {
		t.Fatalf("打断出招的受击者帧序不符:\n got: %s\nwant 尾部: %s", got, want)
	}
	if f.monster.cast != nil {
		t.Fatal("被打断后 NPC 的出招对象应已清空")
	}
}

// npcPos 怪物战斗体的服务端位置快照。
func (f *combatFixture) npcPos() [4]float64 {
	f.monster.mu.Lock()
	defer f.monster.mu.Unlock()
	return f.monster.pos
}

// AllDest 记录与官服逐字节对拍（女号包 757.405 / 758.248 的 0x21 帧：推离
// 的移动记录与到位的停止记录）。
func TestAllDestRecordMatchesOfficialBytes(t *testing.T) {
	const moving = "21 03 00 e8 01 00 35 ca 11 00 00 15 fc 88 c2 bc c8 ed 41 85 37 f4 c2 8b 3a 81 40 00 00 00 40 db 0f c9 c0 00 00 80 40 00 00 00 41 02 26 00 00" +
		" 60 01 00 35 c4 11 00 00 e8 f3 88 c2 bc c8 ed 41 37 8b f2 c2 49 f3 5b 40 00 00 00 00 00 00 00 00 9a 99 f9 40 33 33 f3 41 00 00 00 00" +
		" 60 01 00 35 c4 11 00 00 eb 51 89 c2 bc c8 ed 41 3c c2 f3 c2 2d d9 5b 40 00 00 00 40 db 0f c9 c0 00 00 80 40 00 00 00 41 02 26 00 00"
	const stop = "21 02 00 e8 01 00 35 ca 11 00 00 5a 49 8b c2 bc c8 ed 41 3d 0e f6 c2 e6 94 65 3f 00 00 00 00 00 00 00 00 00 00 80 40 00 00 00 41 0a 00 00 00" +
		" e8 01 00 35 ca 11 00 00 5a 49 8b c2 bc c8 ed 41 3d 0e f6 c2 e6 94 65 3f 00 00 00 00 00 00 00 00 00 00 80 40 00 00 00 41 00 06 00 00"
	for name, text := range map[string]string{"推离移动记录": moving, "到位停止记录": stop} {
		raw, err := hex.DecodeString(strings.ReplaceAll(text, " ", ""))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		// 按布局拆回条目，再用服务端编码器重编：应逐字节还原官服帧。
		var entries []entergen.AllDestEntry
		for at := 3; at+44 <= len(raw); at += 44 {
			e := raw[at:]
			u32 := func(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24 }
			f32 := func(b []byte) float32 { return math.Float32frombits(u32(b)) }
			entries = append(entries, entergen.AllDestEntry{ObjectID: u32(e), OwnerID: u32(e[4:]),
				DestX: f32(e[8:]), DestY: f32(e[12:]), DestZ: f32(e[16:]), DestOrient: f32(e[20:]),
				Speed: f32(e[24:]), Rotate: f32(e[28:]), Motion2: f32(e[32:]), Motion3: f32(e[36:]), MoveFlags: u32(e[40:])})
		}
		if got := hex.EncodeToString(entergen.ServerAllDestFrame(entries)); got != hex.EncodeToString(raw) {
			t.Fatalf("%s: 重编字节与官服不符\n got %s\nwant %s", name, got, hex.EncodeToString(raw))
		}
	}
	// 本实现击退用的运动参数与标志位 = 官服记录里的取值。
	if knockPushSpeed != 2 || knockPushMotion2 != 4 || knockPushMotion3 != 8 || knockMoveFlags != 0x2602 || knockStopFlags != 0xa {
		t.Fatalf("击退运动参数与官服记录不符: %+v", [5]any{knockPushSpeed, knockPushMotion2, knockPushMotion3, knockMoveFlags, knockStopFlags})
	}
}
