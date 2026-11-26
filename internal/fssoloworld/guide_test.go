package fssoloworld

// guide_test.go 新手引导选择流单测（帧字节级对拍在 entergen/guide_test.go；
// 此处观察帧序与 roleState 落点）。官服样本：女号「初始账号登陆-完整
// 引导-女.pcapng」点初出茅庐 m972 [4]，男号「初始账号登陆-跳过引导-男
// .pcapng」点再战江湖 m972 [5]。点选后的接取／切图在定时器协程里发帧，
// 帧收集器带锁。

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// 官服客户端上行原字节（0x0A 活动信封）：黑幕窗点选与新手帮助步骤。
var (
	guideC2SSingle = guideHex("0a e0 07 ac 32 d6 4c 32 8a 1c 36 31 a3 55 91 f7 b2 27 00 00 00 02 00 02 cc 03 00 00 05 00 00 00 00 00 00 10 40")
	guideC2SSkip   = guideHex("0a 5e 70 81 55 62 ca 9d 86 57 30 34 d0 64 1d b3 3c 6e 01 00 00 02 00 02 cc 03 00 00 05 00 00 00 00 00 00 14 40")
	guideC2SHelp   = guideHex("0a d1 01 73 2b 28 b3 cb 61 bc 6f 33 8b 46 cf c8 3d 58 00 00 00 03 00 02 b6 03 00 00 02 00 00 00 00 06 0b 00 00 00 63 68 65 6e 67 64 75 5f 30 32 00")
)

// pinSceneBorn03 把用例角色钉在烟雨庄：用例本身是 born03 场景数据专属，
// 与默认出生场景无关。
func pinSceneBorn03(t *testing.T, pl *Player) {
	t.Helper()
	pl.roleState.Scene, pl.roleState.Resource = `ini\scene\born03_YanYuZhuang`, "born03"
}

func guideHex(s string) []byte {
	var out []byte
	for _, f := range strings.Fields(s) {
		v, err := strconv.ParseUint(f, 16, 8)
		if err != nil {
			panic(err)
		}
		out = append(out, byte(v))
	}
	return out
}

// guideSink 带锁的下行帧收集器（定时器协程也会发帧）。
type guideSink struct {
	mu     sync.Mutex
	frames [][]byte
}

func (g *guideSink) add(_ *Player, frame []byte) {
	g.mu.Lock()
	g.frames = append(g.frames, frame)
	g.mu.Unlock()
}

// lines 取当前帧的摘要（持锁拷贝后离线摘要）。
func (g *guideSink) lines(t *testing.T) []string {
	t.Helper()
	g.mu.Lock()
	frames := append([][]byte(nil), g.frames...)
	g.mu.Unlock()
	return guideLines(t, frames)
}

func (g *guideSink) reset() {
	g.mu.Lock()
	g.frames = nil
	g.mu.Unlock()
}

// newGuideTestPlayer 新手副本里的新号：任务三表清空（行号与官服一致）、
// 身在 born_single_01；兜底计时不触发。
func newGuideTestPlayer(t *testing.T, conn uint32) (*Server, *Player, *guideSink) {
	t.Helper()
	t.Setenv("NINEYIN_GUIDE_CHOICE_TIMEOUT", "0")
	taskTestNoTimers(t)
	s := newNpcTestServer(t)
	if s.npcTaskCatalogOf() == nil {
		t.Fatal("任务目录缺席")
	}
	pl := newTaskTestPlayer(t, conn)
	blankTasks(pl)
	pl.roleState.Resource, pl.roleState.Scene = guideScene, `ini\scene\born_single_01`
	pl.npcMu.Lock()
	pl.npcScene = guideScene
	pl.npcMu.Unlock()
	// 预热新手副本的 NPC 摆放目录：进程内首次取要现装客户端 NPC 数据（-race
	// 下数秒），留给点选后的延时接取去装会超出用例的等待预算。
	s.sceneNPCCatalog(guideScene)
	sink := &guideSink{}
	prev := clientSendObserver
	clientSendObserver = sink.add
	t.Cleanup(func() { clientSendObserver = prev })
	return s, pl, sink
}

// guideSwitchGapSoon 切图间隙调短（须在发起切图之前调）。
func guideSwitchGapSoon(t *testing.T) {
	t.Helper()
	prev := sceneSwitchGap
	sceneSwitchGap = time.Millisecond
	t.Cleanup(func() { sceneSwitchGap = prev })
}

// guideWaitSwitch 等到目标场景待就绪（0x09 屏障）。
func guideWaitSwitch(t *testing.T, s *Server, pl *Player) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for s.pendingSceneSwitch(pl) == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.pendingSceneSwitch(pl) == nil {
		t.Fatal("切图未进入待就绪态（没有等到目标场景 0x09 屏障）")
	}
}

// waitGuideTrack 等任务进入进度（定时器到点后才有）。
func waitGuideTrack(pl *Player, taskID uint32) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		pl.taskMu.Lock()
		accepted := pl.roleState.TaskTrackOf(taskID) != nil
		pl.taskMu.Unlock()
		if accepted {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

func guidePendingFlag(pl *Player) bool {
	pl.taskMu.Lock()
	defer pl.taskMu.Unlock()
	return pl.guidePending
}

// guideLines 引导流相关帧的摘要序列：玩家单属性帧（含串值）+ 任务四表与
// GiveItemRec 记录帧 + 背包行帧 + 全部自定义消息（NPC 头顶标记 m209 15
// 不计）+ 退出场景 0x0C；完成位图的连续零值加行折成一行。
func guideLines(t *testing.T, frames [][]byte) []string {
	t.Helper()
	var out []string
	runStart, runEnd := -1, -1
	flush := func() {
		if runStart >= 0 {
			out = append(out, fmt.Sprintf("11 Completed rows=%d..%d val=0", runStart, runEnd))
			runStart = -1
		}
	}
	for _, f := range frames {
		if f[0] == 0x11 && recordIndexOf(f) == 31 && binary.LittleEndian.Uint32(f[16:]) == 0 {
			row := int(binary.LittleEndian.Uint16(f[12:]))
			if runStart < 0 {
				runStart = row
			}
			runEnd = row
			continue
		}
		flush()
		switch f[0] {
		case 0x0C:
			out = append(out, "0C")
		case 0x10:
			if line := guidePropertyLine(f); line != "" {
				out = append(out, line)
			}
		case 0x11:
			if recordIndexOf(f) == 0x14a { // GiveItemRec：配置 ID、数量
				row := binary.LittleEndian.Uint16(f[12:])
				b := f[16:]
				n := int(binary.LittleEndian.Uint32(b))
				out = append(out, fmt.Sprintf("11 GiveItem row=%d %q %d", row, string(b[4:4+n-1]), binary.LittleEndian.Uint32(b[4+n:])))
				continue
			}
			fallthrough
		case 0x12, 0x13:
			if taskTestNames[recordIndexOf(f)] != "" {
				out = append(out, taskRecordLine(f))
			}
		case 0x18:
			out = append(out, fmt.Sprintf("18 view=%d idx=%d", binary.LittleEndian.Uint16(f[1:]), binary.LittleEndian.Uint16(f[3:])))
		case 0x19:
			out = append(out, fmt.Sprintf("19 view=%d idx=%d", binary.LittleEndian.Uint16(f[1:]), binary.LittleEndian.Uint16(f[3:])))
		case 0x1E:
			line := taskMessageLine(t, f)
			if !strings.HasPrefix(line, "m209 15 ") {
				out = append(out, line)
			}
		}
	}
	flush()
	return out
}

// guidePropertyLine 玩家单属性帧摘要："10 <属性名>=<值>"（串值带引号）。
func guidePropertyLine(frame []byte) string {
	if len(frame) < 14 || frame[1] != 0 || binary.LittleEndian.Uint16(frame[10:]) != 1 {
		return ""
	}
	if binary.LittleEndian.Uint32(frame[2:]) != 0x350000F3 || binary.LittleEndian.Uint32(frame[6:]) != 0x0000102A {
		return ""
	}
	entry, ok := clientdata.PropertyByIndex(int(binary.LittleEndian.Uint16(frame[12:])))
	if !ok {
		return ""
	}
	var value string
	switch entry.TypeCode {
	case 1:
		value = strconv.Itoa(int(frame[14]))
	case 3:
		value = strconv.FormatInt(int64(int32(binary.LittleEndian.Uint32(frame[14:]))), 10)
	case 4:
		value = strconv.FormatInt(int64(binary.LittleEndian.Uint64(frame[14:])), 10)
	case 7:
		n := int(binary.LittleEndian.Uint32(frame[14:]))
		value = strconv.Quote(strings.TrimRight(string(frame[18:18+n]), "\x00"))
	default:
		return ""
	}
	return "10 " + entry.Name + "=" + value
}

// TestNewRoleBornInGuideScene 新号默认出生在新手副本，落点与 GM「新手引导」
// 传送预设是同一个值（rolestate.GuideSceneLanding，scenepresets.go）。
func TestNewRoleBornInGuideScene(t *testing.T) {
	pl := newTaskTestPlayer(t, 40)
	if pl.roleState.Resource != rolestate.GuideScene || pl.roleState.Scene != rolestate.GuideScenePath {
		t.Fatalf("新号出生场景 = %q / %q", pl.roleState.Scene, pl.roleState.Resource)
	}
	if pl.roleState.Position != rolestate.GuideSceneLanding {
		t.Fatalf("新号落点 = %v, want %v", pl.roleState.Position, rolestate.GuideSceneLanding)
	}
	dest, ok := resolveSceneDestination("新手引导")
	if !ok || dest.Resource != rolestate.GuideScene || dest.Position != rolestate.GuideSceneLanding {
		t.Fatalf("GM 新手引导预设 = %+v（ok=%v）", dest, ok)
	}
}

// TestGuideReadyTailRefreshesTrackerForEveryone 任务追踪栏刷新 m390 [5, 0]
// 对所有角色下发（官服每次进场就绪簇末尾都发）；从没进过引导流的角色
// 不下发引导阶段。
func TestGuideReadyTailRefreshesTrackerForEveryone(t *testing.T) {
	s := newNpcTestServer(t)
	pl := newTaskTestPlayer(t, 41)
	pinSceneBorn03(t, pl) // 用例要的是「从没进过引导流」的普通角色
	frames := captureFrames(t)
	s.readyTail(pl)
	assertLines(t, guideLines(t, *frames), []string{"m390 5 0"}, "普通角色就绪收尾")
	if len(*frames) != 1 || (*frames)[0][0] != 0x1E {
		t.Fatalf("就绪收尾应只发一帧 m390: % x", (*frames))
	}
}

// TestGuideReadyTailPendingLocksInput 新手副本待选态：引导阶段 m414 4/5、
// 出生阶段 45、帮助链 chengdu_00、锁输入，最后刷新追踪栏（官服女号登录：
// #183 BornServerGameStep → #437 m414 → #466 HelpStepID → #881
// CanntOperator=1 → … → #1508 m390）。
func TestGuideReadyTailPendingLocksInput(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 42)
	s.readyTail(pl)
	assertLines(t, sink.lines(t), []string{
		"10 BornServerGameStep=45",
		`m414 "gamestep" 4 5`,
		`10 HelpStepID="chengdu_00"`,
		"10 CanntOperator=1",
		"m390 5 0",
	}, "待选态就绪收尾")
	if !guidePendingFlag(pl) || pl.roleState.GameStep != guideStepNone {
		t.Fatalf("应处于待选态: pending=%v step=%d", guidePendingFlag(pl), pl.roleState.GameStep)
	}
	// 玩家档案帧（重登 0x28）带存档的帮助步骤。
	avatar, _ := entergen.AvatarFrame(s.enterContext(pl, false), false)
	if !strings.Contains(string(avatar), "chengdu_00\x00") {
		t.Error("玩家档案 0x28 应带存档的 HelpStepID chengdu_00")
	}
	// 已点选过的角色再进场：下发落定的阶段，不再锁输入。
	pl.taskMu.Lock()
	pl.roleState.GameStep, pl.guidePending = guideStepSingle, false
	pl.taskMu.Unlock()
	sink.reset()
	s.readyTail(pl)
	assertLines(t, sink.lines(t), []string{
		"10 BornServerGameStep=45",
		`m414 "gamestep" 4 5`,
		`10 HelpStepID="chengdu_00"`,
		"m390 5 0",
	}, "非待选态就绪收尾")
}

// TestGuideEnterSceneDefersAutoAcceptWhenPending 待选态不排链根的延时自动
// 接取（等点选）；非待选态照旧。
func TestGuideEnterSceneDefersAutoAcceptWhenPending(t *testing.T) {
	s, pl, _ := newGuideTestPlayer(t, 43)
	prev := npcTaskGuideRootDelay
	npcTaskGuideRootDelay = 20 * time.Millisecond
	t.Cleanup(func() { npcTaskGuideRootDelay = prev })
	s.readyTail(pl)
	s.npcTaskEnterScene(pl)
	time.Sleep(150 * time.Millisecond)
	pl.taskMu.Lock()
	early := pl.roleState.TaskTrackOf(1340) != nil
	pl.taskMu.Unlock()
	if early {
		t.Fatal("待选态不应自动接取引导链根")
	}
	// 视同已点选初出茅庐（阶段落定）后再进场景：照旧延时接取。
	pl.taskMu.Lock()
	pl.roleState.GameStep, pl.guidePending = guideStepSingle, false
	pl.taskMu.Unlock()
	s.npcTaskEnterScene(pl)
	if !waitGuideTrack(pl, 1340) {
		t.Fatal("非待选态进场景应延时接取引导链根 1340")
	}
}

// TestGuideChoiceStartUnlocksAndAcceptsRoot 初出茅庐（女号 C2S #53 m972
// [4]）：解锁输入（#1532）→ 约 1 秒后引导链根 1340 接取簇（#1537..#1552）。
func TestGuideChoiceStartUnlocksAndAcceptsRoot(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 44)
	prev := guideStartDelay
	guideStartDelay = 20 * time.Millisecond
	t.Cleanup(func() { guideStartDelay = prev })
	s.readyTail(pl)
	sink.reset()
	s.clientActivity(pl, guideC2SSingle)
	// 解锁帧同步发；接取簇在定时器到点后（20ms）——先只看首帧。
	if lines := sink.lines(t); len(lines) == 0 || lines[0] != "10 CanntOperator=0" {
		t.Fatalf("初出茅庐首帧应为解锁: %q", lines)
	}
	if pl.roleState.GameStep != guideStepSingle || guidePendingFlag(pl) {
		t.Fatalf("初出茅庐后状态异常: step=%d pending=%v", pl.roleState.GameStep, guidePendingFlag(pl))
	}
	if !waitGuideTrack(pl, 1340) {
		t.Fatal("点选后应自动接取引导链根 1340")
	}
	want := []string{
		"10 CanntOperator=0",
		recordRow(0, 1340, 6, 482, 1, 0, 1, "funnpcborn003031", `""`, "storyinfo_482", 0, "1340_6_funnpcborn003031"),
		`11 Accepted row=0 id=1340 status=0 track=1 step=0 accept="funnpcborn003031" submit="funnpcborn003031"`,
		"13 Accepted row=0 c8=1",
		"m209 9 1340",
		"m24 " + taskTestPlayer + ` 40 "task_accept"`,
		`m1 1 "9628" "title_1340"`,
	}
	// 接取影像在约半秒后（taskmovie）；本用例的影像延迟是一小时，不会到。
	got := sink.lines(t)
	if len(got) < len(want) || strings.Join(got[:len(want)], "\n") != strings.Join(want, "\n") {
		t.Fatalf("初出茅庐解锁 + 1340 接取簇:\n got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// 再点一次初出茅庐：不在待选态，零帧。
	sink.reset()
	s.clientActivity(pl, guideC2SSingle)
	assertLines(t, sink.lines(t), nil, "重复点选")
}

// TestGuideChoiceSkipOfficialOrder 再战江湖（男号 C2S #94 m972 [5]）：跳过
// 簇帧序对拍男号 #1612..#1948（NPC 标记、内功/经脉对象帧不在本服建模
// 内，不计），随后 0x0C 切出副本进鸡鸣驿。
func TestGuideChoiceSkipOfficialOrder(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 45)
	guideSwitchGapSoon(t)
	s.readyTail(pl)
	sink.reset()
	s.clientActivity(pl, guideC2SSkip)

	player := taskTestPlayer
	want := []string{
		// 1. 后续任务接取（#1612..#1630）
		recordRow(0, 1647, 6, 243, 1, 0, 1, "Transbron01A", `""`, "storyinfo_243", 0, "1647_6_Transbron01A"),
		`11 Accepted row=0 id=1647 status=0 track=1 step=0 accept="" submit="Transbron01A"`,
		"13 Accepted row=0 c8=1",
		"m209 9 1647",
		"m24 " + player + ` 40 "task_accept"`,
		`m1 1 "9628" "title_1647"`,
		recordRow(1, 28111, 3, 1086, 1, 0, 1, "event_jmyzyb_001", `"DialogID_1086"`, "interactinfo_1086", 0, "28111_3_event_jmyzyb_001"),
		`11 Accepted row=1 id=28111 status=0 track=1 step=0 accept="" submit=""`,
		"13 Accepted row=1 c8=1",
		"m209 9 28111",
		"m24 " + player + ` 40 "task_accept"`,
		`m1 1 "9628" "title_28111"`,
		// 2. 复活点 → 鸡鸣驿（#1631）
		`10 RelivePositon="4,693.908,24.694,404.35,3.611,ui_pos_ReliveCJ100101"`,
		// 3. 引导链逐个置完成位（#1674..#1694）
		"11 Completed rows=0..41 val=0",
		"13 Completed row=41 c0=134217728",
		"13 Completed row=41 c0=402653184",
		"13 Completed row=41 c0=939524096",
		"13 Completed row=41 c0=2013265920",
		"13 Completed row=41 c0=4160749568", // 官服按 int32 打印是 -134217728
		"11 Completed rows=42..42 val=0",    // #1679：写行 42 前先补零值行
		"13 Completed row=42 c0=1",
		"13 Completed row=42 c0=3",
		"13 Completed row=42 c0=7",
		"m543",
		"13 Completed row=42 c0=15",
		"13 Completed row=42 c0=31",
		"13 Completed row=42 c0=63",
		"11 Completed rows=43..49 val=0", // #1687..#1693
		"13 Completed row=49 c0=536870912",
		// 4. 奖励（#1695..#1727）
		`11 GiveItem row=0 "item_skipsp_001" 1`,
		"19 view=@view@ idx=@slot@",
		"18 view=@view@ idx=@slot@",
		`m1 1 "9645" "item_skipsp_001" 1`,
	}
	// 各档余额独立（本服建号基线 0：余额＝本档累加值）。
	wantGrooves := [6]int32{}
	for _, gain := range guideSkipLiveGrooves {
		points := gain[1] / 1000
		balance := gain[1]
		wantGrooves[gain[0]] = balance
		want = append(want,
			fmt.Sprintf(`m1 1 "8055" %d`, points),
			fmt.Sprintf("m24 %s 1 %d", player, points),
			fmt.Sprintf(`m24 %s 40 "huodelilian"`, player),
			fmt.Sprintf("10 LiveGroove_%d=%d", gain[0], balance),
		)
	}
	want = append(want,
		`11 Repute row=0 "repute_jianghu" 700 2`,
		"m98 177",
		`m1 1 "9693" "repute_jianghu" 700`,
		`m142 "life_freshmanhelp_caiji"`,
		`m142 "life_freshmanhelp_zhizao"`,
		`m142 "life_freshmanhelp_wenhua"`,
		`m142 "life_freshmanhelp_shijing"`,
		"m98 712",
		// 5. 入江湖（#1946..#1948）
		`10 HelpStepID="jimingyi_00"`,
		"10 PowerLevel=1",
		"10 YinLevel=1",
		"10 CanntOperator=0",
		// 切出新手副本
		"0C",
	)
	assertLines(t, withItemSlot(t, pl, want, sink.lines(t)), want, "再战江湖跳过簇")

	// 状态落点。
	st := pl.roleState
	if st.GameStep != guideStepSkip || guidePendingFlag(pl) {
		t.Fatalf("再战江湖后状态异常: step=%d pending=%v", st.GameStep, guidePendingFlag(pl))
	}
	for _, id := range guideSkipDoneTasks {
		if !pl.taskCompletedContains(uint32(id)) {
			t.Errorf("引导链 %d 未置完成位", id)
		}
	}
	if !pl.taskAcceptedContains(1647) || !pl.taskAcceptedContains(28111) {
		t.Errorf("后续任务未接取: accepted=%v", st.TaskAccepted)
	}
	if st.LiveGrooves != wantGrooves {
		t.Errorf("历练 = %v, want %v", st.LiveGrooves, wantGrooves)
	}
	if len(st.Reputes) != 1 || st.Reputes[0].Name != "repute_jianghu" || st.Reputes[0].Count != 700 {
		t.Errorf("声望 = %+v", st.Reputes)
	}
	if countBagItems(pl, "item_skipsp_001") != 1 {
		t.Errorf("再战江湖物品未入包: %+v", st.BagItems)
	}
	if st.RelivePositon != guideSkipRelive || st.HelpStepID != guideHelpStepSkip || st.Resource != guideSkipScene {
		t.Errorf("入江湖落点异常: relive=%q help=%q resource=%q", st.RelivePositon, st.HelpStepID, st.Resource)
	}
	if pos := st.Position; pos != guideSkipLanding {
		t.Errorf("落点 = %v, want %v", pos, guideSkipLanding)
	}

	// 再点一次再战江湖：已办过，零帧。
	sink.reset()
	s.clientActivity(pl, guideC2SSkip)
	assertLines(t, sink.lines(t), nil, "重复点选")

	// 鸡鸣驿就绪（切图重入）：引导阶段 5/5、帮助链 jimingyi_00、刷新追踪栏。
	guideWaitSwitch(t, s, pl)
	sink.reset()
	s.clientReady(pl)
	lines := sink.lines(t)
	for _, need := range []string{`m414 "gamestep" 5 5`, "10 BornServerGameStep=55", `10 HelpStepID="jimingyi_00"`, "m390 5 0"} {
		found := false
		for _, line := range lines {
			found = found || line == need
		}
		if !found {
			t.Errorf("鸡鸣驿就绪簇缺 %q:\n%s", need, strings.Join(lines, "\n"))
		}
	}
}

// TestGuideChoiceOutsidePendingIgnored 不在待选态的角色点选零帧、状态不
// 变（新手副本之外／引导链已在进行）。
func TestGuideChoiceOutsidePendingIgnored(t *testing.T) {
	s := newNpcTestServer(t)
	// 新手副本之外（烟雨庄新号）。
	pl := newTaskTestPlayer(t, 46)
	pinSceneBorn03(t, pl) // 新手副本之外（烟雨庄新号）
	frames := captureFrames(t)
	s.clientActivity(pl, guideC2SSkip)
	s.clientActivity(pl, guideC2SSingle)
	assertLines(t, guideLines(t, *frames), nil, "副本外点选")
	if pl.roleState.GameStep != guideStepNone || pl.roleState.Resource != "born03" {
		t.Fatalf("副本外点选不应改变状态: step=%d resource=%s", pl.roleState.GameStep, pl.roleState.Resource)
	}
	// 副本内但引导链已在进行：点选零帧。
	_, pl, sink := newGuideTestPlayer(t, 47)
	s.readyTail(pl)
	pl.taskMu.Lock()
	s.npcTaskAcceptFrames(pl, &taskFrames{fail: func(error) {}}, 1340)
	pl.roleState.GameStep, pl.guidePending = guideStepSingle, false
	pl.taskMu.Unlock()
	sink.reset()
	s.clientActivity(pl, guideC2SSkip)
	s.clientActivity(pl, guideC2SSingle)
	assertLines(t, sink.lines(t), nil, "引导进行中点选")
	if pl.roleState.GameStep != guideStepSingle || pl.taskCompletedContains(1340) {
		t.Fatalf("引导进行中不应被跳过: step=%d completed=%v", pl.roleState.GameStep, pl.roleState.TaskCompleted)
	}
}

// TestGuideHelpStepAdvance 新手帮助步骤推进 m950 [0,"chengdu_02"]（女号
// C2S #128 → S2C #1792）：步骤存档并回写属性。
func TestGuideHelpStepAdvance(t *testing.T) {
	s, pl, sink := newGuideTestPlayer(t, 48)
	s.readyTail(pl)
	sink.reset()
	s.clientActivity(pl, guideC2SHelp)
	assertLines(t, sink.lines(t), []string{`10 HelpStepID="chengdu_02"`}, "帮助步骤回写")
	if pl.roleState.HelpStepID != "chengdu_02" {
		t.Fatalf("帮助步骤未存档: %q", pl.roleState.HelpStepID)
	}
	avatar, _ := entergen.AvatarFrame(s.enterContext(pl, false), false)
	if !strings.Contains(string(avatar), "chengdu_02\x00") {
		t.Error("玩家档案 0x28 应带存档的 HelpStepID chengdu_02")
	}
}

// TestGMGuideCommand gm guide <who> <start|skip>：待选态直接走对应分支；
// 非待选态回提示。
func TestGMGuideCommand(t *testing.T) {
	s, pl, _ := newGuideTestPlayer(t, 49)
	prev := guideStartDelay
	guideStartDelay = 20 * time.Millisecond
	t.Cleanup(func() { guideStartDelay = prev })
	s.readyTail(pl)

	ask := func(fields ...string) string {
		t.Helper()
		nc, peer := net.Pipe()
		defer peer.Close()
		go func() {
			s.gmGuideCommand(nc, pl, "测试侠", fields)
			nc.Close()
		}()
		var sb strings.Builder
		buf := make([]byte, 256)
		peer.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			n, err := peer.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				break
			}
		}
		return sb.String()
	}
	if got := ask("guide", "测试侠", "bogus"); !strings.Contains(got, "unknown action") {
		t.Errorf("未知动作应回用法提示: %q", got)
	}
	if got := ask("guide", "测试侠", "start"); !strings.Contains(got, ">gm guide 测试侠: start") {
		t.Errorf("start 应回执行结果: %q", got)
	}
	if !waitGuideTrack(pl, 1340) || pl.roleState.GameStep != guideStepSingle {
		t.Fatalf("gm guide start 应接取引导链根: step=%d", pl.roleState.GameStep)
	}
	if got := ask("guide", "测试侠", "skip"); !strings.Contains(got, "not pending") {
		t.Errorf("非待选态应回提示: %q", got)
	}

	s, pl, _ = newGuideTestPlayer(t, 50)
	guideSwitchGapSoon(t)
	s.readyTail(pl)
	if got := ask("guide", "测试侠", "skip"); !strings.Contains(got, ">gm guide 测试侠: skip") {
		t.Errorf("skip 应回执行结果: %q", got)
	}
	if pl.roleState.GameStep != guideStepSkip {
		t.Fatalf("gm guide skip 应置再战江湖: step=%d", pl.roleState.GameStep)
	}
	guideWaitSwitch(t, s, pl)
}
