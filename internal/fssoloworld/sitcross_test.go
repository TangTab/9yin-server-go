package fssoloworld

// sitcross_test.go 原生打坐测试：240 进出帧簇、重复/打断/战斗期拒绝、
// 每秒回复帧序（含补满末跳 (实际, 溢出) 拆分）、打坐中的道具使用拒绝。
// 断言面是可读化的帧序（sitText：0x10 属性 / 0x1E 自定义，宽串记 w:名）。

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

func sitcrossMessage(flag int32) entergen.CustomInbound {
	return entergen.CustomInbound{Values: []entergen.CustomInboundValue{
		{Type: 2, Int32: clientCustomSitCross}, {Type: 2, Int32: flag},
	}}
}

// sitText 下行帧 → 可读串：属性帧走 describeFrames；自定义帧记
// m<号>(o:obj/own,值…)，宽串记 w:名。
func sitText(t *testing.T, frames [][]byte) []string {
	t.Helper()
	out := make([]string, 0, len(frames))
	for _, frame := range frames {
		if len(frame) == 0 || frame[0] != 0x1E {
			out = append(out, describeFrames(t, [][]byte{frame}, time.Time{})...)
			continue
		}
		message, err := entergen.ParseCustomMessage(frame)
		if err != nil || len(message.Values) < 2 {
			t.Fatalf("自定义帧解码失败: %v", err)
		}
		parts := make([]string, 0, len(message.Values)-1)
		for _, value := range message.Values[1:] {
			switch value.Type {
			case 2:
				parts = append(parts, fmt.Sprintf("i:%d", value.Int32))
			case 3:
				parts = append(parts, fmt.Sprintf("l:%d", value.Int64))
			case 6:
				parts = append(parts, "s:"+value.Text)
			case 7:
				parts = append(parts, "w:"+value.Text)
			case 8:
				parts = append(parts, fmt.Sprintf("o:%d/%d", binary.LittleEndian.Uint32(value.Raw[:]),
					binary.LittleEndian.Uint32(value.Raw[4:])))
			default:
				t.Fatalf("未覆盖的值型 %d", value.Type)
			}
		}
		out = append(out, fmt.Sprintf("m%d(%s)", message.Values[0].Int32, strings.Join(parts, ",")))
	}
	return out
}

func assertSitFrames(t *testing.T, frames [][]byte, want string) {
	t.Helper()
	if got := strings.Join(sitText(t, frames), " | "); got != want {
		t.Errorf("帧序不符:\n got: %s\nwant: %s", got, want)
	}
}

func assertNoFrames(t *testing.T, frames [][]byte, what string) {
	t.Helper()
	if len(frames) != 0 {
		t.Fatalf("%s 应零应答: %s", what, strings.Join(sitText(t, frames), " | "))
	}
}

// 进出簇（官服方言，男号线 #1293/#1294 实证）：进入 = BufferInfo2 槽串
//（buf_baosd_02/1E37）→ m169 挂；退出 = m169 撤簇首 → BufferInfo2 清空；
// 重复 start 与不在坐姿的 stop 空转。
func TestSitcrossToggleFrames(t *testing.T) {
	f := newSkillFixture(t)
	send := func(flag int32) { f.s.dispatchCustom(f.pl, sitcrossMessage(flag)) }

	send(1)
	assertSitFrames(t, f.take(), "BufferInfo2=1E37,1,1,0,0,0 | m169(i:0,s:buf_baosd_02,o:1/1,o:1/1,s:,s:,l:0,i:0)")
	if !f.pl.sitting() {
		t.Fatal("进入后应在打坐")
	}
	// 按住按键期间客户端重发 start：不当切换开关。
	send(1)
	assertNoFrames(t, f.take(), "重复 start")

	send(0)
	assertSitFrames(t, f.take(), "m169(i:1,s:buf_baosd_02,o:1/1,o:1/1,s:,s:,l:0,i:0) | BufferInfo2=")
	if f.pl.sitting() {
		t.Fatal("起立后不应在打坐")
	}
	send(0)
	send(2)
	assertNoFrames(t, f.take(), "不在坐姿的 stop/interrupt")

	// 被敌方打断时客户端发 2 而非 0：须按正常停止受理。
	send(1)
	f.take()
	send(2)
	assertSitFrames(t, f.take(), "m169(i:1,s:buf_baosd_02,o:1/1,o:1/1,s:,s:,l:0,i:0) | BufferInfo2=")

	// 未知取值：状态不动。
	send(1)
	f.take()
	send(3)
	assertNoFrames(t, f.take(), "未知 start_or_stop")
	if !f.pl.sitting() {
		t.Fatal("未知取值不应改动坐姿")
	}
}

// 战斗/脱战过渡期（fightAt 未清）拒绝进入：零应答、状态不动。
func TestSitcrossRejectsWhileInFight(t *testing.T) {
	f := newSkillFixture(t)
	st := &f.pl.combat
	st.mu.Lock()
	f.s.ensureLiveLocked(st, f.pl)
	st.live.logic, st.live.fightAt = logicWarning, combatNow()
	st.mu.Unlock()

	f.s.dispatchCustom(f.pl, sitcrossMessage(1))
	assertNoFrames(t, f.take(), "战斗/脱战过渡中")
	st.mu.Lock()
	logic := st.live.logic
	st.mu.Unlock()
	if logic != logicWarning || f.pl.sitting() {
		t.Fatalf("战斗/脱战过渡中不应进入打坐: logic=%d", logic)
	}
}

// 每秒回复：HP/MP 各 +ceil(上限/10)，帧序 HPRatio → HP → m24 16 → MPRatio →
// MP → m24 18；补满末跳 (实际, 溢出) 拆分；站立 MP 回复不叠算；回复帧簇
// 全程不带 LogicState；受击离开坐姿后回复停。
func TestSitcrossTickRecoversHPAndMP(t *testing.T) {
	f := newSkillFixture(t)
	f.pl.Name = "测试"
	f.pl.roleState.HP = 1640 // 每跳 164；MP 上限沿用夹具的 55，每跳 6
	f.s.dispatchCustom(f.pl, sitcrossMessage(1))
	f.take()
	st := &f.pl.combat
	st.mu.Lock()
	f.s.ensureLiveLocked(st, f.pl)
	maxHP, maxMP := st.live.maxHP, playerMaxMP(f.pl)
	st.mpReady, st.mp, st.mpCredit = true, 5, 0.9 // 残留小数累积：打坐中不参与
	st.live.hp -= 400
	st.mu.Unlock()
	if maxHP != 1640 || maxMP != 55 {
		t.Fatalf("夹具上限 HP=%d MP=%d，须与用例取值一致", maxHP, maxMP)
	}

	var all []string
	tick := func() []string {
		f.advance(time.Second)
		f.s.combatTick()
		got := sitText(t, f.take())
		all = append(all, got...)
		return got
	}
	const floatHP = "m24(o:1/1,i:16,i:%d,i:%d,i:0,s:zs_default_01,s:SkillNormal,w:测试,w:测试)"
	const floatMP = "m24(o:1/1,i:18,i:%d,i:%d,i:0,s:zs_default_01,s:SkillNormal,w:测试,w:测试)"
	form := fmt.Sprintf
	expect := func(step string, want []string) {
		t.Helper()
		if got := tick(); strings.Join(got, " | ") != strings.Join(want, " | ") {
			t.Fatalf("%s帧序不符:\n got: %s\nwant: %s", step, strings.Join(got, " | "), strings.Join(want, " | "))
		}
	}
	assertNoFrames(t, f.take(), "回复前")

	// 第 1 跳：HP +164、MP +6（站立回复会另 +3，这里不叠算）。
	expect("第 1 跳 ", []string{"HPRatio=85", "HP=1404", form(floatHP, 164, 0), "MPRatio=20", "MP=11", form(floatMP, 6, 0)})
	expect("第 2 跳 ", []string{"HPRatio=95", "HP=1568", form(floatHP, 164, 0), "MPRatio=30", "MP=17", form(floatMP, 6, 0)})
	// 第 3 跳 HP 补满末跳：名义 164 = 实际 72 + 溢出 92。
	expect("第 3 跳 ", []string{"HPRatio=100", "HP=1640", form(floatHP, 72, 92), "MPRatio=41", "MP=23", form(floatMP, 6, 0)})
	// HP 已满：只剩 MP 层。
	expect("第 4 跳 ", []string{"MPRatio=52", "MP=29", form(floatMP, 6, 0)})
	expect("第 5 跳 ", []string{"MPRatio=63", "MP=35", form(floatMP, 6, 0)})
	expect("第 6 跳 ", []string{"MPRatio=74", "MP=41", form(floatMP, 6, 0)})
	expect("第 7 跳 ", []string{"MPRatio=85", "MP=47", form(floatMP, 6, 0)})
	expect("第 8 跳 ", []string{"MPRatio=96", "MP=53", form(floatMP, 6, 0)})
	// 第 9 跳 MP 补满末跳：名义 6 = 实际 2 + 溢出 4。
	expect("第 9 跳 ", []string{"MPRatio=100", "MP=55", form(floatMP, 2, 4)})
	expect("回满后 ", nil)
	if joined := strings.Join(all, " | "); strings.Contains(joined, "LogicState") {
		t.Fatalf("回复帧簇不得带 LogicState: %s", joined)
	}
	st.mu.Lock()
	hp, mp, credit, logic := st.live.hp, st.mp, st.mpCredit, st.live.logic
	st.mu.Unlock()
	if hp != maxHP || mp != maxMP || credit != 0 || logic != logicSitcross {
		t.Fatalf("回满后状态 HP=%d/%d MP=%d/%d credit=%v logic=%d", hp, maxHP, mp, maxMP, credit, logic)
	}

	// 受击：逻辑态被顶成战斗态即离开打坐，回复停；客户端补发的 2 空转。
	st.mu.Lock()
	st.live.hp -= 400
	noteFightLocked(st, combatNow())
	st.mu.Unlock()
	if got := tick(); len(got) != 0 {
		t.Fatalf("离开坐姿后不应再出回复帧: %s", strings.Join(got, " | "))
	}
	f.s.dispatchCustom(f.pl, sitcrossMessage(2))
	assertNoFrames(t, f.take(), "已不在坐姿的打断")
}

// 站立 MP 回复（combatTick 非打坐分支）满蓝期间不蓄小数：站满一段时间后
// 花掉的蓝仍按每秒速率回，不会一拍回满；不足 1/秒的速率在未满时照常
// 隔拍累积，回满那一拍清零。
func TestCombatTickStandingMPRegenDoesNotBankCredit(t *testing.T) {
	f := newSkillFixture(t)
	st := &f.pl.combat
	st.mu.Lock()
	st.mp, st.mpReady = 55, true
	st.mu.Unlock()
	for i := 0; i < 20; i++ {
		f.s.combatTick()
	}
	assertSitFrames(t, f.take(), "")
	st.mu.Lock()
	st.mp -= 10
	st.mu.Unlock()
	f.s.combatTick()
	assertSitFrames(t, f.take(), "MP=48")

	f.pl.roleState.MP = 10 // 0.5/秒
	st.mu.Lock()
	st.mp, st.mpCredit = 8, 0
	st.mu.Unlock()
	for _, want := range []string{"", "MP=9", "", "MP=10"} {
		f.s.combatTick()
		assertSitFrames(t, f.take(), want)
	}
	st.mu.Lock()
	credit := st.mpCredit
	st.mu.Unlock()
	if credit != 0 {
		t.Fatalf("回满后小数累积应清零: %v", credit)
	}
}

func TestSitRecoveryPerSecond(t *testing.T) {
	for maximum, want := range map[int]int{1640: 164, 1641: 165, 55: 6, 1: 1, 0: 0, -5: 0} {
		if got := sitRecoveryPerSecond(maximum); got != want {
			t.Errorf("sitRecoveryPerSecond(%d) = %d, want %d", maximum, got, want)
		}
	}
}

// 飘字来源位与 starter 快捷栏条目是同一个原生打坐技能。
func TestSitcrossSkillIDMatchesStarterShortcut(t *testing.T) {
	if sitcrossSkillID != rolestate.StarterSitcrossShortcut.ID {
		t.Fatalf("sitcrossSkillID=%q，starter 条目=%q", sitcrossSkillID, rolestate.StarterSitcrossShortcut.ID)
	}
}

// 打坐中使用道具：m1[2,"9402"]，不读条不扣件，坐姿不动。
func TestUseItemRejectedWhileSitting(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "testdata", "ini", "Item", "path_config.xml"),
		filepath.Join("..", "..", "testdata", "ini", "Skill", "skill_static.xml"),
	} {
		if _, err := res.Stat(path); err != nil {
			t.Skipf("构建产物缺席（go run ./tools/datagen item / ./tools/datagen skill）: %v", err)
		}
	}
	f := newItemUseFixture(t)
	st := &f.pl.roleState
	st.BagItems = []rolestate.BagItem{
		{Slot: 5, ConfigID: "book_CS_base_pgd01", ItemType: 23, ViewID: 1, Amount: 1, MaxAmount: 1, Seq: 1},
	}
	f.s.dispatchCustom(f.pl, sitcrossMessage(1))
	if len(f.take()) != 2 {
		t.Fatal("应先进坐姿")
	}
	f.s.dispatchCustom(f.pl, useMsg(2, 5, 1))
	assertSitFrames(t, f.take(), "m1(i:2,s:9402)")
	if len(st.BagItems) != 1 || len(st.Skills) != 0 || f.busy() || !f.pl.sitting() {
		t.Fatalf("打坐中的使用不应生效: bag=%+v skills=%d busy=%t sitting=%t",
			st.BagItems, len(st.Skills), f.busy(), f.pl.sitting())
	}
}
