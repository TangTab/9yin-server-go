package fssoloworld

// skillflash_test.go 闪击通道技（寒心恨雪）：c71 两种线路形态与抓包逐字节对拍；
// 受理 / 逐波 / m914 回显 / 收束的帧序按挑战包 249.640（单跳击杀）、386.312
//（满跳）两次施放定版（docs/combat/寒心恨雪与太极剑舞引导机制.md）。技能取值
// 来自 tools/datagen skill action 构建产物（缺席则跳过）。

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

const flashSkill = "CS_jh_hyjf08"

// 挑战包两条 c71 原帧（tools/pcapevents NINEYIN_PCAP_HEXALL=1 取原始字节）：
// 249.797 首波（12 值）与 386.928 第 2 波（10 值）。施法者 11078F4C/03DC2F97。
const (
	flashFirstWaveHex = "1e0c000218000000" + "0822070b11f0ffdc03" + "0247000000" + "0822070b11f0ffdc03" +
		"060100000000" + "0201000000" + "084c8f0711972fdc03" + "040000f041" + "0400000000" +
		"060100000000" + "0400000000" + "060100000000"
	flashLaterWaveHex = "1e0a000218000000" + "0892660b117f71dd03" + "0247000000" + "086e7a0b118f73dd03" +
		"060100000000" + "0201000000" + "084c8f0711972fdc03" + "040000f041" + "0400000000" +
		"0609000000717a6a5f30375f3200"
)

func objKey(obj, own uint32) uint64 { return uint64(obj) | uint64(own)<<32 }

// c71 的两种线路形态与抓包逐字节一致：首波 [目标,71,目标,”,1,施法者,30,0,
// ”,0,”]，后续波 [上一目标,71,目标,”,1,施法者,30,0,'qzj_07_2']。
func TestFlashAttackFrameMatchesCapture(t *testing.T) {
	caster := wireIdent{objectID: 0x11078f4c, ownerID: 0x03dc2f97}
	target := objKey(0x110b0722, 0x03dcfff0)
	first, err := entergen.CustomMessageFrame(24, flashAttackValues(target, target, caster, 30, flashWaveAction("qzj_07_1", 1))...)
	if err != nil {
		t.Fatalf("首波帧编码失败: %v", err)
	}
	if got := hex.EncodeToString(first); got != flashFirstWaveHex {
		t.Fatalf("首波 c71 与抓包不符:\n got: %s\nwant: %s", got, flashFirstWaveHex)
	}
	later, err := entergen.CustomMessageFrame(24, flashAttackValues(objKey(0x110b6692, 0x03dd717f), objKey(0x110b7a6e, 0x03dd738f),
		caster, 30, flashWaveAction("qzj_07_1", 2))...)
	if err != nil {
		t.Fatalf("后续波帧编码失败: %v", err)
	}
	if got := hex.EncodeToString(later); got != flashLaterWaveHex {
		t.Fatalf("后续波 c71 与抓包不符:\n got: %s\nwant: %s", got, flashLaterWaveHex)
	}
}

// 逐波动作名：首波空，其后 qzj_07_2 … qzj_07_8；异构首段名全波同名。
func TestFlashWaveAction(t *testing.T) {
	for wave, want := range map[int]string{0: "", 1: "", 2: "qzj_07_2", 8: "qzj_07_8"} {
		if got := flashWaveAction("qzj_07_1", wave); got != want {
			t.Errorf("第 %d 波动作 = %q，期望 %q", wave, got, want)
		}
	}
	if got := flashWaveAction("CS_jh_hyjf08", 3); got != "CS_jh_hyjf08" {
		t.Errorf("异构首段名的第 3 波动作 = %q", got)
	}
}

// channelText 帧序文本：去掉冷却表行（冷却模型自有用例），自身 buff 槽串里的
// 时间值归一为 T。
func channelText(s *Server, pl *Player, monster *npcFighter, frames [][]byte) []string {
	cd := fmt.Sprintf("me:rec%d", clientdata.MustRecordIndexByName("cooldown_rec"))
	slot := regexp.MustCompile(`(BufferInfo\d+=233B,[0-9A-F]+,[0-9A-F]+,1,)[0-9A-F]+,0`)
	var out []string
	for _, text := range combatText(s, pl, monster, frames) {
		if strings.HasPrefix(text, cd) {
			continue
		}
		out = append(out, slot.ReplaceAllString(text, "${1}T,0"))
	}
	return out
}

func (f *combatFixture) pumpChannel(d time.Duration) []string {
	f.advance(d)
	f.s.CombatPump()
	return channelText(f.s, f.pl, f.monster, f.take())
}

func (f *combatFixture) setSP(sp int) {
	st := &f.pl.combat
	st.mu.Lock()
	f.s.ensureLiveLocked(st, f.pl)
	st.live.sp = sp
	st.mu.Unlock()
}

func zhaoshiEndMessage(x, y, z, orient float32, kind int32) entergen.CustomInbound {
	return entergen.CustomInbound{Opcode: 0x0A, Values: []entergen.CustomInboundValue{
		{Type: 2, Int32: clientCustomZhaoshiEnd}, {Type: 4, Float32: x}, {Type: 4, Float32: y}, {Type: 4, Float32: z},
		{Type: 4, Float32: orient}, {Type: 2, Int32: kind}, {Type: 2, Int32: 3}}}
}

// 锁态链 / 重挂链 / 收束链的帧序文本（CantParry 与另四位同值：通道自身只占 1 个来源）。
func cantFrames(value string) []string {
	return []string{"me:CantBeSkillLocked=" + value, "me:CantParry=" + value, "me:CantBeSkillMustLocked=" + value,
		"me:CantBeAttack=" + value, "me:CantMove=" + value}
}

const (
	selfBuffDel   = "m169(i:1,s:buf_CS_jh_hyjf08_03,me,me,s:,s:,i:T64,i:0)"
	selfBuffAdd   = "m169(i:0,s:buf_CS_jh_hyjf08_03,me,me,s:,s:,i:T64,i:0)"
	victimBuffDel = "m169(i:1,s:buf_CS_jh_hyjf08,npc,me,s:test_monster,s:,i:T64,i:0)"
	victimBuffAdd = "m169(i:0,s:buf_CS_jh_hyjf08,npc,me,s:test_monster,s:,i:T64,i:0)"
	victimHurt    = "m18(npc,s:hurt_f_1_l_half)"
)

// seq 把几段帧序文本接成一段。
func seq(parts ...[]string) []string {
	var out []string
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func (f *combatFixture) selfSlot() string {
	me := f.pl.wireIdent().orFixed()
	return fmt.Sprintf("me:BufferInfo3=233B,%X,%X,1,T,0", me.objectID, me.ownerID)
}

// 受理簇：CurSkillID → CurSkillTarget → 锁态链 → [冷却行] → 0x1F → m337 →
// m20（动作名 = 招式首段 qzj_07_1）；怒气不预扣。
func TestFlashChannelAcceptFrames(t *testing.T) {
	f := newCombatFixture(t, 3, map[string]string{"MaxLife": "1000000"})
	f.setSP(78)
	f.s.handleSkillCast(f.pl, castMessage(flashSkill, 100, 5, 200, 1.571, f.target(), 103, 5, 200))
	raw := f.take()
	cd := fmt.Sprintf("me:rec%d", clientdata.MustRecordIndexByName("cooldown_rec"))
	rows := 0
	for _, text := range combatText(f.s, f.pl, f.monster, raw) {
		if strings.HasPrefix(text, cd) {
			rows++
		}
	}
	if rows == 0 {
		t.Fatalf("受理簇缺冷却行")
	}
	got := wantText(channelText(f.s, f.pl, f.monster, raw))
	want := wantText([]string{"me:CurSkillID=" + flashSkill, "me:CurSkillTarget=npc",
		"me:CantBeSkillLocked=1", "me:CantParry=1", "me:CantBeSkillMustLocked=1", "me:CantBeAttack=1", "me:CantMove=1",
		f.selfSlot(), selfBuffAdd,
		"loc(me,100.000,5.000,200.000,1.571)", "m337(me,f:100.000,f:5.000,f:200.000,f:1.571,i:0)",
		"m20(me,s:qzj_07_1,npc,i:0,f:0.000)"})
	if got != want {
		t.Fatalf("受理簇帧序不符:\n got: %s\nwant: %s", got, want)
	}
	if sp := f.sp(); sp != 78 {
		t.Fatalf("受理后怒气 = %d，期望不预扣（78）", sp)
	}
}

func (f *combatFixture) sp() int {
	st := &f.pl.combat
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.live.sp
}

func (f *combatFixture) hit17Text(before int) string {
	return fmt.Sprintf("m24(npc,i:17,i:%d,i:0,i:2,s:%s,s:SkillNormal,s:test_monster,s:,i:0)", before-f.hp(), flashSkill)
}

// 满跳：波 1 先结算再重挂链再扣怒，m18 → 挂目标 buff → c71 → m205 → 清
// CurSkillID / CurSkillTarget；波 2 起先摘上一波目标 buff、重挂链、扣怒，m18 →
// 挂目标 buff → c71 在结算帧之前且不再发 m205；第 8 波后一拍只摘目标 buff，
// 末波 + 1s 收束。
func TestFlashChannelFullRun(t *testing.T) {
	f := newCombatFixture(t, 3, map[string]string{"MaxLife": "1000000"})
	f.setSP(78)
	f.s.handleSkillCast(f.pl, castMessage(flashSkill, 100, 5, 200, 1.571, f.target(), 103, 5, 200))
	f.take()

	hp := f.hp()
	got := f.pumpChannel(180 * time.Millisecond)
	want := wantText(seq([]string{"me:FP=2", "me:SP=80", f.aggro(), f.hit17Text(hp), "me:MaxHits=1",
		selfBuffDel, "me:BufferInfo3="},
		cantFrames("0"), cantFrames("1"),
		[]string{f.selfSlot(), selfBuffAdd, "me:SP=70", victimHurt, victimBuffAdd,
			"m24(npc,i:71,npc,s:,i:1,me,f:30.000,f:0.000,s:,f:0.000,s:)",
			"m205(me,s:" + flashSkill + ",npc)", "me:CurSkillID=", "me:CurSkillTarget=o:0/0"}))
	if wantText(got) != want {
		t.Fatalf("波 1 帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}

	hp = f.hp()
	got = f.pumpChannel(440 * time.Millisecond)
	want = wantText(seq([]string{victimBuffDel, selfBuffDel, "me:BufferInfo3="},
		cantFrames("0"), cantFrames("1"),
		[]string{f.selfSlot(), selfBuffAdd, "me:SP=60", victimHurt, victimBuffAdd,
			"m24(npc,i:71,npc,s:,i:1,me,f:30.000,f:0.000,s:qzj_07_2)",
			"me:FP=4", "me:SP=62", "me:rec114[0.1]=T", f.hit17Text(hp), "m24(me,i:10,i:2)", "me:MaxHits=2"}))
	if wantText(got) != want {
		t.Fatalf("波 2 帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}

	// m914 到位上报：位置收编 + 0x21 成对停帧（朝向折 [0, 2π) / 原值）。
	f.s.handleZhaoshiEnd(f.pl, zhaoshiEndMessage(103, 5, 200, -1.1708, zhaoshiEndSkillTranslate))
	got = combatText(f.s, f.pl, f.monster, f.take())
	want = wantText([]string{"alldest(me,103.000,5.000,200.000,o=5.112,spd=0.0,m2=5.5,m3=9.8,f=0xa)",
		"alldest(me,103.000,5.000,200.000,o=-1.171,spd=0.0,m2=5.5,m3=9.8,f=0x600)"})
	if wantText(got) != want {
		t.Fatalf("m914 回显不符:\n got: %s\nwant: %s", wantText(got), want)
	}
	if pos := f.pl.roleState.Position; pos[0] != 103 || float32(pos[3]) != -1.1708 {
		t.Fatalf("m914 位置未收编: %v", pos)
	}
	// 其余类型只收编不回显。
	f.s.handleZhaoshiEnd(f.pl, zhaoshiEndMessage(102, 5, 200, 0.5, 2))
	if left := f.take(); len(left) != 0 {
		t.Fatalf("非技能位移类型的 m914 不应回显: %v", combatText(f.s, f.pl, f.monster, left))
	}

	// 波 3~8：动作名逐波递增；第 8 波后一拍只摘目标 buff。
	got = f.pumpChannel(3080 * time.Millisecond)
	var actions []string
	for _, text := range got {
		if strings.HasPrefix(text, "m24(npc,i:71,") {
			actions = append(actions, strings.TrimSuffix(text, ")"))
		}
	}
	wantActions := []string{"s:qzj_07_3", "s:qzj_07_4", "s:qzj_07_5", "s:qzj_07_6", "s:qzj_07_7", "s:qzj_07_8"}
	if len(actions) != len(wantActions) {
		t.Fatalf("波 3~8 的 c71 数 = %d，期望 %d:\n%s", len(actions), len(wantActions), wantText(got))
	}
	for i, text := range actions {
		if !strings.HasSuffix(text, wantActions[i]) {
			t.Fatalf("第 %d 波 c71 = %s，期望以 %s 结尾", i+3, text, wantActions[i])
		}
	}
	if tail := got[len(got)-1]; tail != victimBuffDel {
		t.Fatalf("收尾拍应只摘目标 buff，末帧 = %s", tail)
	}
	if sp := f.sp(); sp != 14 {
		t.Fatalf("满跳后怒气 = %d，期望 14（78 + 8×2 − 8×10）", sp)
	}

	// 末波（+3260ms）后约 1s 收束：摘自身 buff → 槽串清空 → Cant* 五位清零。
	got = f.pumpChannel(570 * time.Millisecond)
	want = wantText(seq([]string{selfBuffDel, "me:BufferInfo3="}, cantFrames("0")))
	if wantText(got) != want {
		t.Fatalf("收束帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
	f.pl.combat.mu.Lock()
	open, casting := f.pl.combat.channel != nil, f.pl.combat.curSkill != ""
	f.pl.combat.mu.Unlock()
	if open || casting {
		t.Fatalf("收束后通道 / 出招态未清: channel=%t casting=%t", open, casting)
	}
}

// 怒气不足：出过的波照算，到扣不起的那一拍只摘目标 buff 就停，末波 + 1s 收束。
func TestFlashChannelStopsWhenSPIsShort(t *testing.T) {
	f := newCombatFixture(t, 3, map[string]string{"MaxLife": "1000000"})
	f.setSP(30)
	f.s.handleSkillCast(f.pl, castMessage(flashSkill, 100, 5, 200, 1.571, f.target(), 103, 5, 200))
	f.take()
	// 波 1~3：30 → 22 → 14 → 6。
	f.pumpChannel(1060 * time.Millisecond)
	if sp := f.sp(); sp != 6 {
		t.Fatalf("三波后怒气 = %d，期望 6", sp)
	}
	got := f.pumpChannel(440 * time.Millisecond)
	if want := wantText([]string{victimBuffDel}); wantText(got) != want {
		t.Fatalf("怒气不足的收尾拍帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
	// 到末波（+1060ms）+ 1s 之前没有任何帧：既不出波也不补扣。
	if extra := f.pumpChannel(560 * time.Millisecond); len(extra) != 0 {
		t.Fatalf("怒气不足后不应再有帧: %s", wantText(extra))
	}
	got = f.pumpChannel(110 * time.Millisecond)
	want := wantText(seq([]string{selfBuffDel, "me:BufferInfo3="}, cantFrames("0")))
	if wantText(got) != want {
		t.Fatalf("收束帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
}

// 怒气不够扣一次：整条静默吞——零回帧（连冷却清扫都没有），也不起冷却。
func TestFlashChannelSwallowedWithoutSP(t *testing.T) {
	f := newCombatFixture(t, 3, map[string]string{"MaxLife": "1000000"})
	f.setSP(9)
	f.s.handleSkillCast(f.pl, castMessage(flashSkill, 100, 5, 200, 1.571, f.target(), 103, 5, 200))
	if got := f.take(); len(got) != 0 {
		t.Fatalf("怒气不足的施放应零回帧: %v", combatText(f.s, f.pl, f.monster, got))
	}
	if extra := f.pumpChannel(5 * time.Second); len(extra) != 0 {
		t.Fatalf("被吞的施放不应出波: %s", wantText(extra))
	}
	// 同一时刻补足怒气再按：照常受理（没有被上一按的冷却挡住）。
	f.setSP(78)
	f.s.handleSkillCast(f.pl, castMessage(flashSkill, 100, 5, 200, 1.571, f.target(), 103, 5, 200))
	if got := channelText(f.s, f.pl, f.monster, f.take()); len(got) == 0 || got[0] != "me:CurSkillID="+flashSkill {
		t.Fatalf("补足怒气后应受理: %s", wantText(got))
	}
}

// 首跳击杀且再无活目标：波 1 带击杀簇（受击动作只发一次、在结算帧之前），
// 下一拍仍补摘已死目标的 buff，然后收束。
func TestFlashChannelEndsWhenNoTargetLeft(t *testing.T) {
	f := newCombatFixture(t, 3, map[string]string{"MaxLife": "10"})
	f.setSP(78)
	f.s.handleSkillCast(f.pl, castMessage(flashSkill, 100, 5, 200, 1.571, f.target(), 103, 5, 200))
	f.take()
	got := f.pumpChannel(180 * time.Millisecond)
	text := wantText(got)
	for _, part := range []string{"m24(npc,i:17,", " | m24(npc,i:29,me,s:test_monster,s:) | ", " | " + victimHurt + " | " + victimBuffAdd + " | ",
		"m24(npc,i:71,npc,s:,i:1,me,f:30.000,f:0.000,s:,f:0.000,s:)"} {
		if !strings.Contains(text, part) {
			t.Fatalf("首跳击杀的波 1 缺 %q:\n%s", part, text)
		}
	}
	if strings.Count(text, "m18(") != 1 {
		t.Fatalf("击杀波应只发一次受击动作（hurt_f_1_l_half）:\n%s", text)
	}
	if f.monster.isDead() != true {
		t.Fatalf("目标应被击杀")
	}
	got = f.pumpChannel(440 * time.Millisecond)
	if want := wantText([]string{victimBuffDel}); wantText(got) != want {
		t.Fatalf("无目标的收尾拍应补摘已死目标的 buff:\n got: %s\nwant: %s", wantText(got), want)
	}
	got = f.pumpChannel(570 * time.Millisecond)
	want := wantText(seq([]string{selfBuffDel, "me:BufferInfo3="}, cantFrames("0")))
	if wantText(got) != want {
		t.Fatalf("收束帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
}

// 通道被脚本打断（nx_break_any_flow）：下一拍立即收束，不再出波。
func TestFlashChannelClosesEarlyWhenBroken(t *testing.T) {
	f := newCombatFixture(t, 3, map[string]string{"MaxLife": "1000000"})
	f.setSP(78)
	f.s.handleSkillCast(f.pl, castMessage(flashSkill, 100, 5, 200, 1.571, f.target(), 103, 5, 200))
	f.take()
	f.s.breakFlow(playerHandle(f.pl))
	f.take()
	got := f.pumpChannel(180 * time.Millisecond)
	want := wantText(seq([]string{selfBuffDel, "me:BufferInfo3="}, cantFrames("0")))
	if wantText(got) != want {
		t.Fatalf("被打断的收束帧序不符:\n got: %s\nwant: %s", wantText(got), want)
	}
	if extra := f.pumpChannel(5 * time.Second); len(extra) != 0 {
		t.Fatalf("打断后不应再出波: %s", wantText(extra))
	}
	if sp := f.sp(); sp != 78 {
		t.Fatalf("未出波不应扣怒: %d", sp)
	}
}
