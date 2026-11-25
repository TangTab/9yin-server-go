package fssoloworld

// taskrecord_test.go 任务记录同步层：按官服引导抓包（初始账号登陆-完整
// 引导-女.pcapng，8032）的帧序对拍——接取簇、剧情窗、自动提交完成簇、交互
// 确认窗、追踪列表回置、提交窗/201 结算——以及模拟客户端走完整条新手
// 引导链。帧摘要里玩家身份取官服抓包的 350000f3/102a，NPC 身份是本服摆放
// 目录的对象号。

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// newTaskTestPlayer 带角色态（真建号种子）的在线玩家，玩家身份取官服抓包
// 的值。用例结束后置离线——接取影像/占位推进的定时器到点即作废，不会往
// 别的用例里发帧。
func newTaskTestPlayer(t *testing.T, conn uint32) *Player {
	t.Helper()
	state, err := rolestate.BuildRoleState(rolestate.DefaultBirthScene, 1, "", 0)
	if err != nil {
		t.Fatalf("build role state: %v", err)
	}
	pl := &Player{Conn1: conn, Conn2: 1, State: PlOnline, roleState: state, haveRoleState: true}
	pl.setWireIdent(wireIdent{0x350000F3, 0x0000102A})
	t.Cleanup(func() {
		pl.State = PlOffline
		npcMenuMu.Lock()
		delete(npcMenuState, conn)
		npcMenuMu.Unlock()
	})
	return pl
}

// taskTestNoTimers 用例内把占位推进/接取影像/链根进场的定时器拉长：用例
// 自己用 npcTaskStubTick 直调推进，定时器不会在别的用例里触发。
func taskTestNoTimers(t *testing.T) {
	t.Helper()
	stub, movie, root := npcTaskStubInterval, npcTaskMovieDelay, npcTaskGuideRootDelay
	npcTaskStubInterval, npcTaskMovieDelay, npcTaskGuideRootDelay = time.Hour, time.Hour, time.Hour
	t.Cleanup(func() { npcTaskStubInterval, npcTaskMovieDelay, npcTaskGuideRootDelay = stub, movie, root })
}

// blankTasks 清空任务三态（官服新号登录时三张任务表为空）。
func blankTasks(pl *Player) {
	pl.roleState.TaskAccepted, pl.roleState.TaskCompleted, pl.roleState.TaskTracks = nil, nil, nil
}

// ---- 帧摘要 ----

var taskTestSchema = map[uint16][]byte{
	29: mustTaskHex("03030703070703030304030107070703070707070707070103070304"),
	30: mustTaskHex("030303030303070707030704030307"),
	31: {3},
}

var taskTestNames = map[uint16]string{29: "Accepted", 30: "Record", 31: "Completed"}

func mustTaskHex(s string) []byte {
	raw, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return raw
}

// taskTestCell 读一个记录列：摘要 + 字节数。
func taskTestCell(b []byte, typ byte) (string, int) {
	switch typ {
	case 1:
		return strconv.Itoa(int(b[0])), 1
	case 3:
		return strconv.FormatInt(int64(int32(binary.LittleEndian.Uint32(b))), 10), 4
	case 4:
		return strconv.FormatInt(int64(binary.LittleEndian.Uint64(b)), 10), 8
	case 7:
		n := int(binary.LittleEndian.Uint32(b))
		if n == 0 {
			return "<null>", 4
		}
		return strconv.Quote(strings.TrimRight(string(b[4:4+n]), "\x00")), 4 + n
	}
	panic("unsupported cell type")
}

// taskRecordLine 记录帧（0x11/0x12/0x13）摘要。
func taskRecordLine(frame []byte) string {
	rec := binary.LittleEndian.Uint16(frame[10:])
	name := taskTestNames[rec]
	switch frame[0] {
	case 0x12:
		return fmt.Sprintf("12 DEL %s row=%d", name, binary.LittleEndian.Uint16(frame[12:]))
	case 0x11:
		row := binary.LittleEndian.Uint16(frame[12:])
		c := make([]string, 0, len(taskTestSchema[rec]))
		b := frame[16:]
		for _, typ := range taskTestSchema[rec] {
			v, n := taskTestCell(b, typ)
			c = append(c, v)
			b = b[n:]
		}
		switch rec {
		case 29:
			return fmt.Sprintf("11 Accepted row=%d id=%s status=%s track=%s step=%s accept=%s submit=%s",
				row, c[0], c[6], c[7], c[8], c[14], c[4])
		case 30:
			return fmt.Sprintf("11 Record row=%d id=%s sub=%s idx=%s order=%s cur=%s max=%s key=%s scene=%s track=%s flags=%s link=%s",
				row, c[0], c[1], c[2], c[3], c[4], c[5], c[6], c[7], c[8], c[9], c[10])
		default:
			return fmt.Sprintf("11 Completed row=%d val=%d", row, binary.LittleEndian.Uint32(frame[len(frame)-4:]))
		}
	default:
		ncell := int(binary.LittleEndian.Uint16(frame[12:]))
		row := binary.LittleEndian.Uint16(frame[14:])
		b := frame[16:]
		parts := make([]string, 0, ncell)
		for i := 0; i < ncell; i++ {
			col := int(b[0])
			b = b[1:]
			if rec == 31 { // 完成位图按无符号打印
				parts = append(parts, fmt.Sprintf("c%d=%d", col, binary.LittleEndian.Uint32(b)))
				b = b[4:]
				continue
			}
			v, n := taskTestCell(b, taskTestSchema[rec][col])
			b = b[n:]
			parts = append(parts, fmt.Sprintf("c%d=%s", col, v))
		}
		return fmt.Sprintf("13 %s row=%d %s", name, row, strings.Join(parts, " "))
	}
}

// taskMessageLine 0x1E 自定义消息摘要：m<号> <值…>。
func taskMessageLine(t *testing.T, frame []byte) string {
	t.Helper()
	msg, err := entergen.ParseCustomMessage(frame)
	if err != nil {
		t.Fatalf("自定义消息解帧失败: %v", err)
	}
	if len(msg.Values) == 0 || msg.Values[0].Type != 2 {
		t.Fatalf("自定义消息首值不是消息号: %+v", msg.Values)
	}
	parts := []string{strconv.Itoa(int(msg.Values[0].Int32))}
	for _, v := range msg.Values[1:] {
		switch v.Type {
		case 2:
			parts = append(parts, strconv.FormatInt(int64(v.Int32), 10))
		case 3:
			parts = append(parts, strconv.FormatInt(v.Int64, 10))
		case 6:
			parts = append(parts, strconv.Quote(v.Text))
		case 8:
			parts = append(parts, fmt.Sprintf("obj(%x/%x)", binary.LittleEndian.Uint32(v.Raw[:]), binary.LittleEndian.Uint32(v.Raw[4:])))
		default:
			parts = append(parts, fmt.Sprintf("<t%d>", v.Type))
		}
	}
	return "m" + strings.Join(parts, " ")
}

// taskSyncLines 任务同步相关帧的摘要序列：三张任务记录帧 + 任务通知/收尾
// 消息（m1、m8、m18、m24、m209）；完成位图的连续零值加行折成一行，NPC
// 标记（m209 15）、选中帧、对话窗不计。
func taskSyncLines(t *testing.T, frames [][]byte) []string {
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
		if f[0] == 0x11 && binary.LittleEndian.Uint16(f[10:]) == 31 && binary.LittleEndian.Uint32(f[16:]) == 0 {
			row := int(binary.LittleEndian.Uint16(f[12:]))
			if runStart < 0 {
				runStart = row
			}
			runEnd = row
			continue
		}
		flush()
		switch f[0] {
		case 0x11, 0x12, 0x13:
			if rec := binary.LittleEndian.Uint16(f[10:]); taskTestNames[rec] != "" {
				out = append(out, taskRecordLine(f))
			}
		case 0x1E:
			line := taskMessageLine(t, f)
			id := line
			if sp := strings.IndexByte(line, ' '); sp >= 0 {
				id = line[:sp]
			}
			if id == "m1" || id == "m8" || id == "m18" || id == "m24" ||
				(id == "m209" && !strings.HasPrefix(line, "m209 15 ")) {
				out = append(out, line)
			}
		}
	}
	flush()
	return out
}

// talkWindows 对话窗摘要：每个 m4 开一段，段内取 m5 标题与 m6 行。
func talkWindows(t *testing.T, frames [][]byte) []string {
	t.Helper()
	var windows []string
	var lines []string
	title := ""
	for _, f := range frames {
		if f[0] != 0x1E {
			continue
		}
		line := taskMessageLine(t, f)
		switch {
		case line == "m4":
			title, lines = "", nil
		case strings.HasPrefix(line, "m5 "):
			title = strings.Trim(strings.TrimPrefix(line, "m5 "), `"`)
		case strings.HasPrefix(line, "m6 "):
			lines = append(lines, strings.TrimPrefix(line, "m6 "))
		case strings.HasPrefix(line, "m7"):
			windows = append(windows, title+" | "+strings.Join(lines, " ; "))
		}
	}
	return windows
}

// assertLines 帧摘要序列逐行相等。
func assertLines(t *testing.T, got, want []string, what string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s 帧数 = %d, want %d:\n got:\n%s\nwant:\n%s", what, len(got), len(want),
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s 第 %d 行不一致:\n got  %s\n want %s", what, i, got[i], want[i])
		}
	}
}

// recordRow 期望行：目标表加行（12 个摘要列全比对）。
func recordRow(row, id, sub, idx, order, cur, max int, key, scene, track string, flags int, link string) string {
	if link == "" {
		link = `""`
	} else {
		link = strconv.Quote(link)
	}
	return fmt.Sprintf("11 Record row=%d id=%d sub=%d idx=%d order=%d cur=%d max=%d key=%q scene=%s track=%q flags=%d link=%s",
		row, id, sub, idx, order, cur, max, key, scene, track, flags, link)
}

// ---- 模拟客户端 ----

// taskTestSpawn 玩家所在场景（born03 真档摆放）里指定配置 ID 的 NPC。
func taskTestSpawn(t *testing.T, s *Server, pl *Player, configID string) npcfeed.Spawn {
	t.Helper()
	spawn, ok := s.npcTaskSpawnOf(pl, configID)
	if !ok {
		t.Fatalf("场景 %s 的摆放目录里没有 NPC %s", pl.npcScene, configID)
	}
	return spawn
}

func taskTestIdent(spawn npcfeed.Spawn) string {
	return fmt.Sprintf("obj(%x/%x)", spawn.ObjectID, spawn.OwnerID)
}

// taskTestClick 模拟客户端 op0x07：funcID=0 点 NPC 开窗，否则点对话行。
// 返回本次应答帧。
func taskTestClick(t *testing.T, s *Server, pl *Player, spawn npcfeed.Spawn, funcID int32, sink *[][]byte) [][]byte {
	t.Helper()
	*sink = nil
	s.handleNPCObjectRequest(pl, npcObjectRequest{Sequence: 1, ObjectID: spawn.ObjectID, OwnerID: spawn.OwnerID, FuncID: funcID})
	return *sink
}

// taskTestTrace 模拟客户端上报追踪列表 m327 [6,"id,…"]，返回应答帧。
func taskTestTrace(t *testing.T, s *Server, pl *Player, list string, sink *[][]byte) [][]byte {
	t.Helper()
	*sink = nil
	s.handleTaskMsg(pl, customMsg(327, customInt(6), customStr(list)))
	return *sink
}

// taskMarkfunc 对话行 funcid。
func taskMarkfunc(flag, taskID int32) int32 { return flag*npcTaskFuncBase + taskID }

const taskTestPlayer = `obj(350000f3/102a)`

// TestTaskSyncOfficialOrder 1340→1341 三环按官服抓包帧序逐步对拍（新号：
// 任务三表为空，行号与官服一致）。
func TestTaskSyncOfficialOrder(t *testing.T) {
	s := newNpcTestServer(t)
	if s.npcTaskCatalogOf() == nil {
		t.Fatal("任务目录缺席")
	}
	taskTestNoTimers(t)
	pl := newTaskTestPlayer(t, 21)
	blankTasks(pl)
	pl.npcMu.Lock()
	pl.npcScene = "born_single_01"
	pl.npcMu.Unlock()
	frames := captureFrames(t)
	guide := taskTestSpawn(t, s, pl, "funnpcborn003031")
	wushu := taskTestSpawn(t, s, pl, "funnpcborn003035")
	wushuIdent := taskTestIdent(wushu)
	traceWrites := []string{"13 Accepted row=0 c7=0", "13 Accepted row=0 c7=1"}

	// 接取 1340（官服 #1537/#1538[0x29 单行批]/#1539/#1550..1552）。
	s.acceptNPCTask(pl, 1340, "进场")
	assertLines(t, taskSyncLines(t, *frames), []string{
		recordRow(0, 1340, 6, 482, 1, 0, 1, "funnpcborn003031", `""`, "storyinfo_482", 0, "1340_6_funnpcborn003031"),
		`11 Accepted row=0 id=1340 status=0 track=1 step=0 accept="funnpcborn003031" submit="funnpcborn003031"`,
		"13 Accepted row=0 c8=1",
		"m209 9 1340",
		"m24 " + taskTestPlayer + ` 40 "task_accept"`,
		`m1 1 "9628" "title_1340"`,
	}, "1340 接取簇")
	assertLines(t, taskSyncLines(t, taskTestTrace(t, s, pl, "0,1340", frames)), traceWrites, "1340 追踪列表回置")

	// 点 NPC：主窗只有剧情行（官服 98.161s）。
	if got := talkWindows(t, taskTestClick(t, s, pl, guide, 0, frames)); len(got) != 1 ||
		got[0] != `task_funnpcborn003031 | 500001340 "story_menu_482" "title_1340" ; 600000000 "ui_exit"` {
		t.Fatalf("主窗 = %q", got)
	}
	// 剧情窗五段：点剧情行开窗 1，续行 50N 开窗 N+1（官服 100.097..101.530s）。
	last := storyWindows(t, s, pl, guide, 1340, 482, 5, frames)
	// 末窗续行（505）：剧情步完成，末步自动提交——完成簇（官服 101.701s，
	// 奖励提示与 NPC 标记不计）。
	taskTestClick(t, s, pl, guide, last, frames)
	assertLines(t, taskSyncLines(t, *frames), []string{
		"13 Record row=0 c4=1",
		"11 Completed rows=0..41 val=0",
		"13 Completed row=41 c0=268435456",
		"12 DEL Record row=0",
		"12 DEL Accepted row=0",
		"m209 8 1340",
		`m1 1 "9631" "title_1340"`,
		recordRow(0, 1347, 3, 558, 1, 0, 1, "funnpcborn003035", `"DialogID_558"`, "interactinfo_558", 0, "1347_3_funnpcborn003035"),
		recordRow(1, 1347, 3, 559, 2, 0, 1, "funnpcborn003035", `"DialogID_559"`, "interactinfo_559", 0, "1347_3_funnpcborn003035"),
		recordRow(2, 1347, 3, 560, 3, 0, 1, "funnpcborn003035", `"DialogID_560"`, "interactinfo_560", 0, "1347_3_funnpcborn003035"),
		recordRow(3, 1347, 3, 561, 4, 0, 1, "funnpcborn003035", `"DialogID_561"`, "interactinfo_561", 0, "1347_3_funnpcborn003035"),
		`11 Accepted row=0 id=1347 status=0 track=1 step=0 accept="funnpcborn003035" submit="funnpcborn003035"`,
		"13 Accepted row=0 c8=1",
		"m209 9 1347",
		"m24 " + taskTestPlayer + ` 40 "task_accept"`,
		`m1 1 "9628" "title_1347"`,
		"m24 " + taskTestPlayer + ` 40 "task_complete"`,
		"m8",
	}, "1340 完成簇（末步自动提交）")
	if pl.taskAcceptedContains(1340) || !pl.taskCompletedContains(1340) || !pl.taskAcceptedContains(1347) {
		t.Fatalf("1340 完成/1347 接取后状态异常: accepted=%+v completed=%+v", pl.roleState.TaskAccepted, pl.roleState.TaskCompleted)
	}
	assertLines(t, taskSyncLines(t, taskTestTrace(t, s, pl, "0,1347", frames)), traceWrites, "1347 追踪列表回置")

	// 点武学 NPC：主窗只有离开，紧接步骤确认窗（官服 103.635s）。
	if got := talkWindows(t, taskTestClick(t, s, pl, wushu, 0, frames)); len(got) != 2 ||
		got[0] != `task_funnpcborn003035 | 600000000 "ui_exit"` ||
		got[1] != `DialogID_558 | 500001347 "okmenu_558" ; 600000000 "ui_form_talk_leave"` {
		t.Fatalf("武学 NPC 对话窗 = %q", got)
	}
	// 四个交互步逐一确认（官服 104.697/112.279/124.171/136.631s）。
	for i, step := range []int{558, 559, 560, 561} {
		last := i == 3
		transition := fmt.Sprintf("13 Accepted row=0 c8=%d", i+2)
		if last {
			transition = "13 Accepted row=0 c6=2"
		}
		want := []string{
			fmt.Sprintf("13 Record row=%d c4=1", i),
			transition,
			fmt.Sprintf("13 Record row=%d c9=32", i),
			fmt.Sprintf("13 Record row=%d c9=48", i),
			"m24 " + taskTestPlayer + ` 40 "task_ok"`,
			fmt.Sprintf("m18 %s \"\"", wushuIdent),
		}
		if last {
			// 末步非自动提交：紧接提交窗参数（官服 1342/1597 的末步同此）。
			want = append(want, fmt.Sprintf(`m209 2 %s 201001347 1347 "complete_1347" "submit_1347" 0 "" 9`, wushuIdent))
		}
		want = append(want, "m8")
		assertLines(t, taskSyncLines(t, taskTestClick(t, s, pl, wushu, taskMarkfunc(500, 1347), frames)), want, fmt.Sprintf("1347 交互步 %d", step))
		if !last {
			// 下一步的确认窗：重新点 NPC 才开（官服 111.247/122.873/135.711s）。
			if got := talkWindows(t, taskTestClick(t, s, pl, wushu, 0, frames)); len(got) != 2 ||
				got[1] != fmt.Sprintf(`DialogID_%d | 500001347 "okmenu_%d" ; 600000000 "ui_form_talk_leave"`, step+1, step+1) {
				t.Fatalf("交互步 %d 走完后再点 NPC 的窗口 = %q", step, got)
			}
		}
	}
	// 四步走完：主窗翻提交行（官服 146.728s）。
	if got := talkWindows(t, taskTestClick(t, s, pl, wushu, 0, frames)); len(got) != 1 ||
		got[0] != `task_funnpcborn003035 | 200001347 "CompleteMenu_1347" "title_1347" ; 600000000 "ui_exit"` {
		t.Fatalf("1347 可提交时主窗 = %q", got)
	}
	// 点提交行：只回提交窗参数（官服 147.593s #2406）。
	assertLines(t, taskSyncLines(t, taskTestClick(t, s, pl, wushu, taskMarkfunc(200, 1347), frames)), []string{
		fmt.Sprintf(`m209 2 %s 201001347 1347 "complete_1347" "submit_1347" 0 "" 9`, wushuIdent),
	}, "提交窗参数")
	if !pl.taskAcceptedContains(1347) {
		t.Fatal("200 不应改任务状态")
	}
	// 窗内确认（201）：完成簇 + 链推 1341（官服 148.094s）。
	assertLines(t, taskSyncLines(t, taskTestClick(t, s, pl, wushu, taskMarkfunc(201, 1347), frames)), []string{
		"11 Completed rows=42..42 val=0", // 位图行 42 客户端还没有：先加零值行再写格（官服 #2413/#2414）
		"13 Completed row=42 c0=8",
		"12 DEL Record row=0", "12 DEL Record row=0", "12 DEL Record row=0", "12 DEL Record row=0",
		"12 DEL Accepted row=0",
		"m209 8 1347",
		`m1 1 "9631" "title_1347"`,
		recordRow(0, 1341, 1, 522, 1, 0, 1, "NpcBorn003011", "<null>", "hunterinfo_522", 0, ""),
		recordRow(1, 1341, 1, 518, 2, 0, 1, "bossborn003001", "<null>", "hunterinfo_518", 0, ""),
		`11 Accepted row=0 id=1341 status=0 track=1 step=0 accept="funnpcborn003036" submit="funnpcborn003036"`,
		"13 Accepted row=0 c8=1",
		"13 Record row=0 c9=1",
		"m209 9 1341",
		"m24 " + taskTestPlayer + ` 40 "task_accept"`,
		`m1 1 "9628" "title_1341"`,
		"m24 " + taskTestPlayer + ` 40 "task_complete"`,
		"m18 " + wushuIdent + ` ""`,
		"m8",
	}, "1347 完成簇（提交确认）")
	assertLines(t, taskSyncLines(t, taskTestTrace(t, s, pl, "0,1341", frames)), traceWrites, "1341 追踪列表回置")

	// 杀怪步占位（官服 210.666s 击杀帧序：flags 写带位 1、下一行置 1、
	// 不关对话）。
	*frames = nil
	s.npcTaskStubTick(pl, 1341)
	assertLines(t, taskSyncLines(t, *frames), []string{
		"13 Record row=0 c4=1",
		"13 Accepted row=0 c8=2",
		"13 Record row=0 c9=33",
		"13 Record row=0 c9=49",
		"13 Record row=1 c9=1",
		"m24 " + taskTestPlayer + ` 40 "task_ok"`,
	}, "1341 第一只")
	// 末只：末步自动提交——curnum 之后直接进完成簇（官服 228.138s）。
	*frames = nil
	s.npcTaskStubTick(pl, 1341)
	assertLines(t, taskSyncLines(t, *frames), []string{
		"13 Record row=1 c4=1",
		"13 Completed row=41 c0=805306368",
		"12 DEL Record row=0", "12 DEL Record row=0",
		"12 DEL Accepted row=0",
		"m209 8 1341",
		`m1 1 "9631" "title_1341"`,
		recordRow(0, 1342, 3, 562, 1, 0, 1, "funnpcborn003032", `"DialogID_562"`, "interactinfo_562", 0, "1342_3_funnpcborn003032"),
		recordRow(1, 1342, 8, 203, 2, 0, 1, "2061", `""`, "specialinfo_2061", 0, "1342_8_2061"),
		recordRow(2, 1342, 6, 484, 3, 0, 1, "funnpcborn003032", `""`, "storyinfo_484", 0, "1342_6_funnpcborn003032"),
		recordRow(3, 1342, 3, 570, 4, 0, 1, "funnpcborn003032", `"DialogID_570"`, "interactinfo_570", 0, "1342_3_funnpcborn003032"),
		recordRow(4, 1342, 3, 139, 5, 0, 1, "funnpcborn003032", `"DialogID_139"`, "interactinfo_139", 0, "1342_3_funnpcborn003032"),
		`11 Accepted row=0 id=1342 status=0 track=1 step=0 accept="funnpcborn003032" submit="funnpcborn003032"`,
		"13 Accepted row=0 c8=1",
		"m209 9 1342",
		"m24 " + taskTestPlayer + ` 40 "task_accept"`,
		`m1 1 "9628" "title_1342"`,
		"m24 " + taskTestPlayer + ` 40 "task_complete"`,
	}, "1341 完成簇（末步自动提交，无对话收尾）")
	if pl.taskAcceptedContains(1341) || !pl.taskAcceptedContains(1342) {
		t.Fatalf("1341 完成/1342 接取后状态异常: %+v", pl.roleState.TaskAccepted)
	}
}

// maskTimes 把已接行接取时刻的逐格写折成占位（时刻取当前时间）。
func maskTimes(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		if strings.HasPrefix(line, "13 Accepted row=") && strings.Contains(line, " c9=") && !strings.Contains(line, " c8=") {
			line = line[:strings.Index(line, " c9=")] + " c9=T"
		}
		out[i] = line
	}
	return out
}

// completedMask 完成位图某一行当前应有的掩码（玩家已交任务集）。
func completedMask(pl *Player, row int) uint32 {
	var mask uint32
	for _, id := range pl.roleState.TaskCompleted {
		if int(id/32) == row {
			mask |= 1 << (id % 32)
		}
	}
	return mask
}

// assertRecordLayout 登录簇的目标表行推与运行期的行号推算同一口径（重登
// 后客户端表与运行期推算的行布局一致）。
func assertRecordLayout(t *testing.T, s *Server, pl *Player) {
	t.Helper()
	rows, err := entergen.MountSequenceFrames(s.enterContext(pl, false))
	if err != nil {
		t.Fatalf("登录簇行推生成失败: %v", err)
	}
	login := 0
	for _, f := range rows {
		if f[0] == 0x11 && binary.LittleEndian.Uint16(f[10:]) == 30 {
			login++
		}
	}
	total := 0
	for _, id := range pl.roleState.TaskAccepted {
		if pl.roleState.TaskTrackOf(id) != nil {
			total += len(s.npcTaskCatalogOf().StepsForTask(int(id)))
		}
	}
	if login != total {
		t.Fatalf("登录簇目标表 %d 行 ≠ 运行期推算 %d 行", login, total)
	}
}

// markLines NPC 任务标记帧摘要（m209 15 → "obj(..) state"）。
func markLines(t *testing.T, frames [][]byte) []string {
	t.Helper()
	var out []string
	for _, f := range frames {
		if f[0] != 0x1E {
			continue
		}
		fields := strings.Fields(taskMessageLine(t, f))
		if len(fields) == 4 && fields[0] == "m209" && fields[1] == "15" {
			out = append(out, fields[2]+" "+fields[3])
		}
	}
	return out
}

// storyWindows 剧情步走点击路径：主窗剧情行 → 窗 1，续行 50N → 窗 N+1
// （共 segments 段，剧情组 group）。返回末窗续行的 funcid。
func storyWindows(t *testing.T, s *Server, pl *Player, npc npcfeed.Spawn, taskID, group, segments int, frames *[][]byte) int32 {
	t.Helper()
	for n := 1; n <= segments; n++ {
		click := taskMarkfunc(500, int32(taskID))
		if n > 1 {
			click = taskMarkfunc(int32(500+n-1), int32(taskID))
		}
		got := talkWindows(t, taskTestClick(t, s, pl, npc, click, frames))
		want := fmt.Sprintf("story_%d_%d | %d%06d %q ; %d%06d %q",
			group, n, 500+n, taskID, fmt.Sprintf("storymenu_%d_%d", group, n), 600+n, taskID, "9607")
		if len(got) != 1 || got[0] != want {
			t.Fatalf("任务 %d 剧情窗 %d = %q, want %q", taskID, n, got, want)
		}
		if len(taskSyncLines(t, *frames)) != 0 {
			t.Fatalf("剧情窗段间不应有同步帧: %v", taskSyncLines(t, *frames))
		}
	}
	return taskMarkfunc(int32(500+segments), int32(taskID))
}

// TestTaskSyncGuideChainToEnd 模拟客户端从 1342 走到引导链末环 1346：交互
// 确认、特殊步到步即过、多段剧情窗、杀怪/事件步占位、提交窗确认，以及
// 建号种子行（1344 已在已接表、1346 已在已交位图）上的链推进。玩家带真
// 建号种子（已接表先有功能门行）。
func TestTaskSyncGuideChainToEnd(t *testing.T) {
	s := newNpcTestServer(t)
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		t.Fatal("任务目录缺席")
	}
	taskTestNoTimers(t)
	pl := newTaskTestPlayer(t, 22)
	pl.npcMu.Lock()
	pl.npcScene = "born_single_01"
	pl.npcMu.Unlock()
	frames := captureFrames(t)
	seedAccepted := len(pl.roleState.TaskAccepted)
	if !pl.taskAcceptedContains(1344) || !pl.taskCompletedContains(1346) {
		t.Fatalf("建号种子应含已接 1344 与已交 1346: %+v / %+v", pl.roleState.TaskAccepted, pl.roleState.TaskCompleted)
	}
	// stepDone 对话窗内完成任务的当前步。
	stepDone := func(taskID int) bool {
		done := false
		s.npcTaskLocked(pl, func() {
			task, track, _, ok := npcTaskCurrent(catalog, pl, uint32(taskID))
			if !ok {
				return
			}
			done = s.npcTaskStepDone(pl, uint32(taskID), npcTaskDialogTail(entergen.Identity{}, task, track.Step-1))
		})
		return done
	}
	track := func(taskID int) *rolestate.TaskTrack { return pl.roleState.TaskTrackOf(uint32(taskID)) }
	acceptedRow := func(taskID int) int {
		row, _ := npcTaskAcceptedRow(pl, uint32(taskID))
		return int(row)
	}
	// 快进到 1341 完成（1340 剧情自动提交 → 1347 四步 + 提交 → 1341 两只）。
	s.acceptNPCTask(pl, 1340, "进场")
	assertRecordLayout(t, s, pl)
	if !stepDone(1340) || !pl.taskAcceptedContains(1347) {
		t.Fatalf("1340 走完应链推 1347: %+v", pl.roleState.TaskAccepted)
	}
	for i := 0; i < 4; i++ {
		if !stepDone(1347) {
			t.Fatalf("1347 第 %d 步未推进", i+1)
		}
	}
	if !s.completeNPCTask(pl, 1347, npcTaskCompleteTail{closeTalk: true}) {
		t.Fatal("1347 提交未受理")
	}
	if acceptedRow(1341) != seedAccepted {
		t.Fatalf("1341 应接在种子行之后（行 %d）: %d", seedAccepted, acceptedRow(1341))
	}
	s.npcTaskStubTick(pl, 1341)
	if track(1341).Step != 2 || track(1341).Cur[0] != 1 {
		t.Fatalf("1341 第一只后进度异常: %+v", track(1341))
	}
	s.npcTaskStubTick(pl, 1341)
	if pl.taskAcceptedContains(1341) || !pl.taskAcceptedContains(1342) {
		t.Fatalf("1341 完成后应链推 1342: %+v", pl.roleState.TaskAccepted)
	}
	assertRecordLayout(t, s, pl)

	// 1342：交互 562 →（特殊步 2061 到步即过）→ 剧情 484 → 交互 570 → 交互 139。
	if !stepDone(1342) {
		t.Fatal("1342 交互步 562 未推进")
	}
	if got := track(1342); got.Step != 3 || len(got.Cur) != 2 || got.Cur[0] != 1 || got.Cur[1] != 1 {
		t.Fatalf("特殊步应到步即过（当前步应是剧情步 484）: %+v", got)
	}
	for i := 0; i < 3; i++ {
		if !stepDone(1342) {
			t.Fatalf("1342 第 %d 步未推进", i+3)
		}
	}
	if !entergen.TaskSubmittable(track(1342), catalogTask(catalog, 1342).Steps) {
		t.Fatalf("1342 五步走完应可提交: %+v", track(1342))
	}
	if !s.completeNPCTask(pl, 1342, npcTaskCompleteTail{closeTalk: true}) {
		t.Fatal("1342 提交未受理")
	}
	if !pl.taskAcceptedContains(1343) {
		t.Fatalf("1342 完成后应链推 1343: %+v", pl.roleState.TaskAccepted)
	}

	// 1343：杀怪 519×3 → 交互 575 → 杀怪 523 → 杀怪 520（末步自动提交）。
	start1343, _, _ := npcTaskRecordRows(pl, catalog, 1343)
	row1343 := acceptedRow(1343)
	for i := 0; i < 3; i++ {
		s.npcTaskStubTick(pl, 1343)
	}
	if got := track(1343); got.Step != 2 || got.Cur[0] != 3 {
		t.Fatalf("1343 三只后进度异常: %+v", got)
	}
	if !stepDone(1343) {
		t.Fatal("1343 交互步 575 未推进")
	}
	*frames = nil
	s.npcTaskStubTick(pl, 1343) // 杀怪 523：非末步
	assertLines(t, taskSyncLines(t, *frames), []string{
		fmt.Sprintf("13 Record row=%d c4=1", start1343+2),
		fmt.Sprintf("13 Accepted row=%d c8=4", row1343),
		fmt.Sprintf("13 Record row=%d c9=33", start1343+2),
		fmt.Sprintf("13 Record row=%d c9=49", start1343+2),
		fmt.Sprintf("13 Record row=%d c9=1", start1343+3),
		"m24 " + taskTestPlayer + ` 40 "task_ok"`,
	}, "1343 杀怪 523")
	// 杀怪 520：末步自动提交；完成簇里 1344 接取——种子已接行不重复加行：
	// 目标行接在表尾（此刻只有它有进度行＝从第 0 行起），已接行逐格写
	// 追踪标志/接取时刻，再置步序（官服 400.440s 的簇在已接表为空时是加
	// 行，帧序其余相同）。
	row1344 := acceptedRow(1344)
	*frames = nil
	s.npcTaskStubTick(pl, 1343)
	assertLines(t, maskTimes(taskSyncLines(t, *frames)), []string{
		fmt.Sprintf("13 Record row=%d c4=1", start1343+3),
		fmt.Sprintf("13 Completed row=41 c0=%d", completedMask(pl, 41)),
		"12 DEL Record row=0", "12 DEL Record row=0", "12 DEL Record row=0", "12 DEL Record row=0",
		fmt.Sprintf("12 DEL Accepted row=%d", row1343),
		"m209 8 1343",
		`m1 1 "9631" "title_1343"`,
		recordRow(0, 1344, 3, 564, 1, 0, 1, "funnpcborn003033", `"DialogID_564"`, "interactinfo_564", 0, "1344_3_funnpcborn003033"),
		recordRow(1, 1344, 3, 565, 2, 0, 1, "funnpcborn003033", `"DialogID_565"`, "interactinfo_565", 0, "1344_3_funnpcborn003033"),
		recordRow(2, 1344, 3, 576, 3, 0, 1, "GatherBorn003001", `"DialogID_576"`, "interactinfo_576", 0, "1344_3_GatherBorn003001"),
		recordRow(3, 1344, 3, 566, 4, 0, 1, "funnpcborn003033", `"DialogID_566"`, "interactinfo_566", 0, "1344_3_funnpcborn003033"),
		recordRow(4, 1344, 3, 567, 5, 0, 1, "funnpcborn003033", `"DialogID_567"`, "interactinfo_567", 0, "1344_3_funnpcborn003033"),
		fmt.Sprintf("13 Accepted row=%d c7=1", row1344),
		fmt.Sprintf("13 Accepted row=%d c9=T", row1344),
		fmt.Sprintf("13 Accepted row=%d c8=1", row1344),
		"m209 9 1344",
		"m24 " + taskTestPlayer + ` 40 "task_accept"`,
		`m1 1 "9628" "title_1344"`,
		"m24 " + taskTestPlayer + ` 40 "task_complete"`,
	}, "1343 完成簇（1344 种子已接行）")
	if pl.taskAcceptedContains(1343) || len(pl.roleState.TaskAccepted) != seedAccepted {
		t.Fatalf("1343 完成后已接表应回到种子行数: %+v", pl.roleState.TaskAccepted)
	}
	assertRecordLayout(t, s, pl)

	// 1344：五个交互步 → 提交 → 1348。
	for i := 0; i < 5; i++ {
		if !stepDone(1344) {
			t.Fatalf("1344 第 %d 步未推进", i+1)
		}
	}
	if !s.completeNPCTask(pl, 1344, npcTaskCompleteTail{closeTalk: true}) || !pl.taskAcceptedContains(1348) {
		t.Fatalf("1344 提交后应链推 1348: %+v", pl.roleState.TaskAccepted)
	}
	assertRecordLayout(t, s, pl)
	// 1348：交互 270 → 特殊步 2091 到步即过 → 末步自动提交 → 1349。
	if !stepDone(1348) {
		t.Fatal("1348 交互步 270 未推进")
	}
	if pl.taskAcceptedContains(1348) || !pl.taskAcceptedContains(1349) {
		t.Fatalf("1348 特殊步到步即过后应自动提交并链推 1349: %+v", pl.roleState.TaskAccepted)
	}
	// 1349：交互 269/271/272 → 杀怪 534（末步自动提交）→ 1597。
	for i := 0; i < 3; i++ {
		if !stepDone(1349) {
			t.Fatalf("1349 第 %d 步未推进", i+1)
		}
	}
	if got := track(1349); got.Step != 4 {
		t.Fatalf("1349 三步后应在杀怪步: %+v", got)
	}
	s.npcTaskStubTick(pl, 1349)
	if !pl.taskAcceptedContains(1597) {
		t.Fatalf("1349 完成后应链推 1597: %+v", pl.roleState.TaskAccepted)
	}
	// 1597：交互 496 → 提交 → 1345。
	if !stepDone(1597) || !s.completeNPCTask(pl, 1597, npcTaskCompleteTail{closeTalk: true}) || !pl.taskAcceptedContains(1345) {
		t.Fatalf("1597 推进/提交后应链推 1345: %+v", pl.roleState.TaskAccepted)
	}
	// 1345：交互 260（末步自动提交）→ 1346（种子里已交：照常重新接取）。
	if !stepDone(1345) {
		t.Fatal("1345 交互步 260 未推进")
	}
	if pl.taskAcceptedContains(1345) || !pl.taskAcceptedContains(1346) || track(1346) == nil {
		t.Fatalf("1345 完成后应重新接取 1346: accepted=%+v", pl.roleState.TaskAccepted)
	}
	assertRecordLayout(t, s, pl)
	// 1346：事件交互步 568（无对话，不出确认窗）与杀怪步 521 走占位 → 剧情步 483。
	if _, _, ok := s.npcTaskStepWindow(catalog, pl, "EventNpcBorn003003"); ok {
		t.Fatal("事件交互步没有对话，不应出确认窗")
	}
	s.npcTaskStubTick(pl, 1346)
	if got := track(1346); got.Step != 2 || got.Cur[0] != 1 {
		t.Fatalf("1346 事件步占位后进度异常: %+v", got)
	}
	s.npcTaskStubTick(pl, 1346)
	if got := track(1346); got.Step != 3 || got.Cur[1] != 1 {
		t.Fatalf("1346 杀怪步占位后进度异常: %+v", got)
	}
	// 剧情步 483（四段）：主窗剧情行 → 窗 1..4 → 末窗续行（504）= 步完成，
	// 末步非自动提交紧接提交窗参数（官服 828.580..830.572s）。
	zhao := taskTestSpawn(t, s, pl, "funnpcborn003034")
	if got := talkWindows(t, taskTestClick(t, s, pl, zhao, 0, frames)); len(got) != 1 ||
		got[0] != `task_funnpcborn003034 | 500001346 "story_menu_483" "title_1346" ; 600000000 "ui_exit"` {
		t.Fatalf("1346 主窗 = %q", got)
	}
	last := storyWindows(t, s, pl, zhao, 1346, 483, 4, frames)
	taskTestClick(t, s, pl, zhao, last, frames)
	zhaoIdent := taskTestIdent(zhao)
	assertLines(t, taskSyncLines(t, *frames), []string{
		"13 Record row=2 c4=1",
		fmt.Sprintf("13 Accepted row=%d c6=2", acceptedRow(1346)),
		"13 Record row=2 c9=32",
		"13 Record row=2 c9=48",
		"m24 " + taskTestPlayer + ` 40 "task_ok"`,
		fmt.Sprintf("m18 %s \"\"", zhaoIdent),
		fmt.Sprintf(`m209 2 %s 201001346 1346 "complete_1346" "submit_1346" 0 "" 9`, zhaoIdent),
		"m8",
	}, "1346 末步（剧情）")
	// 提交确认：链尾（1346 无后继）。
	row1346 := acceptedRow(1346)
	*frames = nil
	if !s.completeNPCTask(pl, 1346, npcTaskCompleteTail{closeTalk: true}) {
		t.Fatal("1346 提交未受理")
	}
	assertLines(t, taskSyncLines(t, *frames), []string{
		fmt.Sprintf("13 Completed row=42 c0=%d", completedMask(pl, 42)),
		"12 DEL Record row=0", "12 DEL Record row=0", "12 DEL Record row=0",
		fmt.Sprintf("12 DEL Accepted row=%d", row1346),
		"m209 8 1346",
		`m1 1 "9631" "title_1346"`,
		"m24 " + taskTestPlayer + ` 40 "task_complete"`,
		"m8",
	}, "1346 完成簇（链尾）")
	// 种子里的 1344 已接行随它真正交掉而离开已接表：其余种子行原样留下。
	if len(pl.roleState.TaskAccepted) != seedAccepted-1 || pl.taskAcceptedContains(1344) || len(pl.roleState.TaskTracks) != 0 {
		t.Fatalf("引导链走完后已接表应只剩种子行（去掉已交的 1344）、无进度行: %+v / %+v", pl.roleState.TaskAccepted, pl.roleState.TaskTracks)
	}
	for _, id := range []uint32{1340, 1341, 1342, 1343, 1344, 1345, 1346, 1348, 1349, 1597} {
		if !pl.taskCompletedContains(id) {
			t.Fatalf("引导链任务 %d 应已交: %+v", id, pl.roleState.TaskCompleted)
		}
	}
	assertRecordLayout(t, s, pl)
}

// TestTaskMarksFollowProgress NPC 任务标记（m209 [15,obj,state]）随进度变化：
// 可接 4 → 剧情步在此 9 → 交互步在此 1 → 可提交 2 → 任务待回此 NPC 3。
func TestTaskMarksFollowProgress(t *testing.T) {
	s := newNpcTestServer(t)
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		t.Fatal("任务目录缺席")
	}
	taskTestNoTimers(t)
	pl := newTaskTestPlayer(t, 23)
	blankTasks(pl)
	pl.npcMu.Lock()
	pl.npcScene = "born_single_01"
	pl.npcVisible = map[uint32]bool{}
	visible := s.sceneNPCCatalog("born_single_01").spawns
	for _, spawn := range visible {
		pl.npcVisible[spawn.ObjectID] = true
	}
	pl.npcMu.Unlock()
	state := func(configID string) uint32 { return npcTaskMarkOf(catalog, pl, configID) }
	if got := state("funnpcborn003031"); got != npcTaskMarkAccept {
		t.Fatalf("接取前引导 NPC 标记 = %d, want %d（可接）", got, npcTaskMarkAccept)
	}
	guide := taskTestSpawn(t, s, pl, "funnpcborn003031")
	frames := captureFrames(t)
	s.pushNPCTaskMarks(pl)
	// 首推同官服：视口内每个 NPC 各一帧（绝大多数为 0），只有可接的带 4。
	if got := markLines(t, *frames); len(got) != len(visible) {
		t.Fatalf("首推标记帧数 = %d, want %d（视口内每个 NPC 一帧）", len(got), len(visible))
	}
	for _, line := range markLines(t, *frames) {
		if line != taskTestIdent(guide)+" 4" && !strings.HasSuffix(line, " 0") {
			t.Fatalf("接取前不应有其它非零标记: %q", line)
		}
	}
	*frames = nil
	s.pushNPCTaskMarks(pl)
	if got := markLines(t, *frames); len(got) != 0 {
		t.Fatalf("标记未变化不应重发: %q", got)
	}
	*frames = nil
	s.acceptNPCTask(pl, 1340, "进场")
	if state("funnpcborn003031") != npcTaskMarkDoing|npcTaskMarkStory {
		t.Fatalf("1340 进行中标记 = %d, want 9", state("funnpcborn003031"))
	}
	if got := markLines(t, *frames); len(got) != 1 || got[0] != taskTestIdent(guide)+" 9" {
		t.Fatalf("接取后标记帧 = %q", got)
	}
	s.npcTaskLocked(pl, func() {
		s.npcTaskStepDone(pl, 1340, npcTaskDialogTail(entergen.Identity{}, catalogTask(catalog, 1340), 0))
	})
	if got := state("funnpcborn003031"); got != 0 {
		t.Fatalf("1340 完成后引导 NPC 标记 = %d, want 0", got)
	}
	if got := state("funnpcborn003035"); got != npcTaskMarkDoing {
		t.Fatalf("1347 交互步在此标记 = %d, want 1", got)
	}
	for i := 0; i < 4; i++ {
		s.npcTaskLocked(pl, func() {
			s.npcTaskStepDone(pl, 1347, npcTaskDialogTail(entergen.Identity{}, catalogTask(catalog, 1347), 0))
		})
	}
	if got := state("funnpcborn003035"); got != npcTaskMarkSubmit {
		t.Fatalf("1347 可提交标记 = %d, want 2", got)
	}
	s.completeNPCTask(pl, 1347, npcTaskCompleteTail{})
	if got := state("funnpcborn003036"); got != npcTaskMarkDoing|npcTaskMarkSubmit {
		t.Fatalf("1341 杀怪步期间任务 NPC 标记 = %d, want 3", got)
	}
	if got := state("funnpcborn003035"); got != 0 {
		t.Fatalf("1347 交后标记 = %d, want 0", got)
	}
}

// TestTaskNormalizeTracksAndSceneRoots 进场整理与链根进场接取：旧流程只记
// 了任务号的已接任务补进度行；建号种子的功能门行不补；引导链根在进场后
// 自动接取，已交过的不再接。
func TestTaskNormalizeTracksAndSceneRoots(t *testing.T) {
	s := newNpcTestServer(t)
	if s.npcTaskCatalogOf() == nil {
		t.Fatal("任务目录缺席")
	}
	pl := newTaskTestPlayer(t, 24)
	// 旧流程留下的已接任务号（无进度行）。
	pl.roleState.TaskAccepted = append(pl.roleState.TaskAccepted, 1340, 1347)
	s.normalizeTaskTracks(pl)
	if track := pl.roleState.TaskTrackOf(1340); track == nil || track.Step != 1 || !track.Tracked {
		t.Fatalf("旧流程的 1340 应补进度行: %+v", track)
	}
	if pl.roleState.TaskTrackOf(1344) != nil {
		t.Fatal("建号种子的功能门行 1344 不应补进度行")
	}
	if len(pl.roleState.TaskTracks) != 2 {
		t.Fatalf("进度行数 = %d, want 2（1340/1347）: %+v", len(pl.roleState.TaskTracks), pl.roleState.TaskTracks)
	}
	// 链根进场接取：born03 里摆放了接取 NPC 的引导链根只有 1340（已接，
	// 不重复）。
	fresh := newTaskTestPlayer(t, 25)
	fresh.roleState.TaskAccepted, fresh.roleState.TaskCompleted = nil, nil
	fresh.npcMu.Lock()
	fresh.npcScene = "born_single_01"
	fresh.npcMu.Unlock()
	s.npcAcceptSceneGuideRoots(fresh)
	if !fresh.taskAcceptedContains(1340) || len(fresh.roleState.TaskAccepted) != 1 {
		t.Fatalf("进场应自动接取引导链根 1340: %+v", fresh.roleState.TaskAccepted)
	}
	s.npcAcceptSceneGuideRoots(fresh)
	if len(fresh.roleState.TaskAccepted) != 1 {
		t.Fatalf("重复进场接取应幂等: %+v", fresh.roleState.TaskAccepted)
	}
}
