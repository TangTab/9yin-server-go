package fssoloworld

// taskmovie_test.go 任务影像生命周期：接取影像（m104）是一场客户端剧情副本，
// 开始帧与结束收尾簇逐字节对拍官服引导抓包（#1559/#1560、#1789..#1791），
// 副本期间位移不收编、视口 NPC 重发、自动步顺延，以及开关与超时兜底。

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// movieTask1340 官服 1340「武林高手」：接取影像 473（开场剧情副本）。
func movieTask1340() taskdata.Task {
	return taskdata.Task{ID: 1340, AcceptMovie: 473}
}

// taskMovieTestScene 玩家站在新手场景（born_single_01）里，视口=场景全量
// 摆放（与 npcReconcile 的簿记同口径）。返回视口 NPC 数。
func taskMovieTestScene(t *testing.T, s *Server, pl *Player) int {
	t.Helper()
	taskTestNoTimers(t)
	spawns := s.sceneNPCCatalog("born_single_01").spawns
	pl.npcMu.Lock()
	pl.npcScene = "born_single_01"
	pl.npcVisible = map[uint32]bool{}
	for _, spawn := range spawns {
		pl.npcVisible[spawn.ObjectID] = true
	}
	pl.npcMu.Unlock()
	return len(spawns)
}

// waitFrames 等下行帧数达到 n（定时器异步发帧；超时返回当前帧）。
func waitFrames(t *testing.T, frames *[][]byte, n int) [][]byte {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for len(*frames) < n && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	return *frames
}

func taskHex(t *testing.T, s string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// movieFrameCount 下行帧里的影像帧（m104）条数。
func movieFrameCount(frames [][]byte) int {
	n := 0
	for _, f := range frames {
		if len(f) >= 5 && f[0] == 0x1E && f[4] == 104 {
			n++
		}
	}
	return n
}

func TestTaskMovieStartMatchesOfficial(t *testing.T) {
	// 开始：m8 → m104 [0/0, 473, 4]（官服 33.486s 两帧原字节）。
	s := newNpcTestServer(t)
	pl := newTaskTestPlayer(t, 31)
	frames := captureFrames(t)
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	got := waitFrames(t, frames, 2)
	if len(got) != 2 || !bytes.Equal(got[0], taskHex(t, "1e01000208000000")) ||
		!bytes.Equal(got[1], taskHex(t, "1e0400026800000008000000000000000002d90100000204000000")) {
		t.Fatalf("影像开始帧与官服不一致: %x", got)
	}
	if !s.npcTaskMoviePlaying(pl) {
		t.Fatal("影像开始后应在途")
	}
	// 同一玩家不叠播。
	*frames = nil
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	if len(*frames) != 0 {
		t.Fatalf("影像在途时不应再起一场: %x", *frames)
	}
	// 没有影像号的任务不起影像。
	s.npcTaskLocked(pl, func() { s.npcTaskMovieEnd(pl, "测试") })
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, taskdata.Task{ID: 1347}) })
	if s.npcTaskMoviePlaying(pl) {
		t.Fatal("无 AcceptMovie 的任务不应起影像")
	}
}

func TestTaskMovieEndMatchesOfficial(t *testing.T) {
	// 客户端上报 m806 [473] → CantBeAttack=0 → CantMove=0 → m107（官服
	// 60.385s 三帧原字节）。
	s := newNpcTestServer(t)
	pl := newTaskTestPlayer(t, 32)
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	frames := captureFrames(t)
	// 影像号的结束上报与在途影像不符：忽略。
	s.handleMovieMsg(pl, customMsg(806, customInt(474)), 806)
	if len(*frames) != 0 || !s.npcTaskMoviePlaying(pl) {
		t.Fatalf("别的影像的结束上报不应收尾: frames=%x", *frames)
	}
	// 将结束/同意播放只留痕。
	s.handleMovieMsg(pl, customMsg(805, customInt(473)), 805)
	s.handleMovieMsg(pl, customMsg(807), 807)
	if len(*frames) != 0 {
		t.Fatalf("m805/m807 不应下发: %x", *frames)
	}
	s.handleMovieMsg(pl, customMsg(806, customInt(473)), 806)
	want := []string{"1000f30000352a1000000100cd0000", "1000f30000352a1000000100cb0000", "1e0100026b000000"}
	if len(*frames) != len(want) {
		t.Fatalf("影像收尾帧数 = %d, want %d: %x", len(*frames), len(want), *frames)
	}
	for i, w := range want {
		if !bytes.Equal((*frames)[i], taskHex(t, w)) {
			t.Fatalf("收尾帧 %d 与官服不一致:\n got %x\nwant %s", i, (*frames)[i], w)
		}
	}
	if s.npcTaskMoviePlaying(pl) {
		t.Fatal("结束上报后影像应不在途")
	}
	// 没有在途影像的结束上报：静默。
	*frames = nil
	s.handleMovieMsg(pl, customMsg(806, customInt(473)), 806)
	if len(*frames) != 0 {
		t.Fatalf("无在途影像不应回帧: %x", *frames)
	}
}

func TestTaskMovieMovementAndRepush(t *testing.T) {
	// 副本期间：位移不收编；视口 NPC 整批重发（官服影像开始 0.6 秒后）；行走
	// 态位移视作玩家已离开副本，立即收尾并照常收编。
	s := newNpcTestServer(t)
	pl := newTaskTestPlayer(t, 33)
	visible := taskMovieTestScene(t, s, pl)
	if visible == 0 {
		t.Fatal("born_single_01 摆放缺席")
	}
	t.Setenv("NINEYIN_TASK_MOVIE_REPUSH_MS", "30")
	frames := captureFrames(t)
	origin := pl.roleState.Position
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	// 副本里的站立态上报（官服 36.157s：state 5、坐标同出生点）：位置不收编。
	s.handleMovementUpload(pl, customMsg(15, customInt(1), customInt(5),
		customFloat32(197.5), customFloat32(4.7), customFloat32(-108.6)))
	if pl.roleState.Position != origin {
		t.Fatalf("影像期间的位移不应收编: %v", pl.roleState.Position)
	}
	// 等重发：m8、m104 之后全部是 NPC 物化帧（0x28），条数=视口 NPC 数。
	got := waitFrames(t, frames, 2+visible)
	op28 := 0
	for _, f := range got[2:] {
		if f[0] == 0x28 {
			op28++
		}
	}
	if op28 != visible {
		t.Fatalf("视口重发 0x28 帧 %d 条, want %d（共 %d 帧）", op28, visible, len(got))
	}
	// 视口簿记不变（重发不是对账）。
	pl.npcMu.Lock()
	kept := len(pl.npcVisible)
	pl.npcMu.Unlock()
	if kept != visible {
		t.Fatalf("重发不应改视口簿记: %d → %d", visible, kept)
	}
	// 行走态上报：玩家已不在副本里——影像收尾（标志复位 + m107），位移照常收编。
	*frames = nil
	s.handleMovementUpload(pl, customMsg(15, customInt(1), customInt(7),
		customFloat32(201), customFloat32(5), customFloat32(-100), customFloat32(0), customFloat32(0), customFloat32(0), customFloat32(1.5)))
	if s.npcTaskMoviePlaying(pl) {
		t.Fatal("行走态位移后影像应收尾")
	}
	if pl.roleState.Position == origin || pl.roleState.Position[0] != 201 {
		t.Fatalf("影像收尾后位移应照常收编: %v", pl.roleState.Position)
	}
	if len(*frames) < 3 {
		t.Fatalf("玩家移动收尾应发标志复位两帧 + m107: %x", *frames)
	}
	if !bytes.Equal((*frames)[0], taskHex(t, "1000f30000352a1000000100cd0000")) ||
		!bytes.Equal((*frames)[1], taskHex(t, "1000f30000352a1000000100cb0000")) {
		t.Fatalf("玩家移动收尾的标志复位帧异常: %x", (*frames)[:2])
	}
	// 影像结束后重发定时器到点不再重发（在途态已清）。
	*frames = nil
	time.Sleep(80 * time.Millisecond)
	for _, f := range *frames {
		if f[0] == 0x28 {
			t.Fatal("影像已结束不应再重发视口")
		}
	}
}

func TestTaskMovieDefersAutoSteps(t *testing.T) {
	// 自动步在影像期间不推进：影像结束（客户端上报）后接着走。
	s := newNpcTestServer(t)
	if s.npcTaskCatalogOf() == nil {
		t.Fatal("任务目录缺席")
	}
	pl := newTaskTestPlayer(t, 34)
	blankTasks(pl)
	taskTestNoTimers(t)
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	s.acceptNPCTask(pl, 1341, "测试")
	track := pl.roleState.TaskTrackOf(1341)
	if track == nil || track.Step != 1 {
		t.Fatalf("1341 应已接取在杀怪步: %+v", track)
	}
	// 影像期间排一次推进：只顺延，不记账。
	s.npcTaskLocked(pl, func() { s.npcTaskAutoSteps(pl, 1341) })
	s.npcTaskStubTick(pl, 1341)
	if track.Cur != nil && len(track.Cur) != 0 {
		t.Fatalf("影像期间占位推进不应记账: %+v", track)
	}
	// 影像结束：压后的自动步接上（定时器被拉长，不立即推进；之后一拍照常）。
	s.handleMovieMsg(pl, customMsg(806, customInt(473)), 806)
	if s.npcTaskMoviePlaying(pl) || len(track.Cur) != 0 {
		t.Fatalf("影像结束后不应立即推进: movie=%v track=%+v", s.npcTaskMoviePlaying(pl), track)
	}
	s.npcTaskStubTick(pl, 1341)
	if len(track.Cur) != 1 || track.Cur[0] != 1 {
		t.Fatalf("影像结束后占位推进应恢复: %+v", track)
	}
}

func TestTaskMovieScheduleSwitches(t *testing.T) {
	// 接取影像经定时器下发（接取约半秒后）；任务已不在进度中则到点不发；
	// NINEYIN_TASK_MOVIE=off 不发。
	s := newNpcTestServer(t)
	taskTestNoTimers(t)
	restore := npcTaskMovieDelay
	npcTaskMovieDelay = 20 * time.Millisecond
	t.Cleanup(func() { npcTaskMovieDelay = restore })

	pl := newTaskTestPlayer(t, 35)
	blankTasks(pl)
	s.acceptNPCTask(pl, 1340, "进场")
	frames := captureFrames(t)
	s.npcTaskLocked(pl, func() { s.npcTaskScheduleMovie(pl, movieTask1340()) })
	got := waitFrames(t, frames, 2)
	if len(got) != 2 || movieFrameCount(got) != 1 || !bytes.Equal(got[0], taskHex(t, "1e01000208000000")) {
		t.Fatalf("接取影像应经定时器下发 m8 + m104(473): %x", got)
	}
	// 任务已提交（不在进度中）：到点不发。
	s.npcTaskLocked(pl, func() { s.npcTaskMovieEnd(pl, "测试") })
	blankTasks(pl)
	*frames = nil
	s.npcTaskLocked(pl, func() { s.npcTaskScheduleMovie(pl, movieTask1340()) })
	time.Sleep(80 * time.Millisecond)
	if movieFrameCount(*frames) != 0 {
		t.Fatalf("任务已不在进度中不应再发影像: %x", *frames)
	}
	// 开关：off 不发。
	t.Setenv("NINEYIN_TASK_MOVIE", "off")
	pl2 := newTaskTestPlayer(t, 36)
	blankTasks(pl2)
	s.acceptNPCTask(pl2, 1340, "进场")
	*frames = nil
	s.npcTaskLocked(pl2, func() { s.npcTaskScheduleMovie(pl2, movieTask1340()) })
	time.Sleep(80 * time.Millisecond)
	if len(*frames) != 0 || s.npcTaskMoviePlaying(pl2) {
		t.Fatalf("NINEYIN_TASK_MOVIE=off 不应发影像: %x", *frames)
	}
}

func TestTaskMovieTimeout(t *testing.T) {
	// 客户端一直不上报结束：到点自动收尾（标志复位 + m107），在途态清除。
	s := newNpcTestServer(t)
	taskTestNoTimers(t)
	restore := npcTaskMovieTimeout
	npcTaskMovieTimeout = 30 * time.Millisecond
	t.Cleanup(func() { npcTaskMovieTimeout = restore })
	pl := newTaskTestPlayer(t, 37)
	s.npcTaskLocked(pl, func() { s.npcTaskStartMovie(pl, movieTask1340()) })
	frames := captureFrames(t)
	deadline := time.Now().Add(3 * time.Second)
	for s.npcTaskMoviePlaying(pl) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if s.npcTaskMoviePlaying(pl) {
		t.Fatal("影像超时后应自动收尾")
	}
	if len(*frames) != 3 || !bytes.Equal((*frames)[2], taskHex(t, "1e0100026b000000")) {
		t.Fatalf("超时收尾应发标志复位两帧 + m107: %x", *frames)
	}
}
