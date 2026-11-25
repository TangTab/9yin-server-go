package fssoloworld

// taskmovie.go 任务影像（主表 AcceptMovie → m104 [0/0, 影像号, 4]）的下发与
// 生命周期，按官服引导抓包（初始账号登陆-完整引导-女.pcapng，8032）定版。
//
// 引导链的「接取影像」不是一小段动画，而是一场**剧情副本**（客户端
// ini/movie/movie_clone003_<号>.ini → scenario 脚本：玩家被传到副本地点、
// 变成 NPC 演员、禁 ESC、结束后 restore_pos 传回原位；473 开场约 27 秒、
// 474 约 5 秒）。客户端进副本会清掉场景对象，所以官服在影像开始约 0.6 秒
// 后把视口内 NPC 的物化帧整批重发（33.486 m104 → 34.095 六帧 0x28），玩家
// 回到原位时 NPC 还在。只发影像不重发＝副本结束后 NPC 全没了（2026-11-26
// 真机「接取任务后目标 NPC 不见了」）。
//
// 官服时序：
//
//	  T+0.0    接取簇（官服同时置 CantMove/CantBeAttack=1 冻结玩家）
//	  T+0.5    m8 → m104 [0/0, 影像号, 4]
//	  T+1.1    m98 145 → 视口 NPC 物化帧重发
//	  影像结束  C2S m806 [影像号] → CantBeAttack=0 → CantMove=0 → m107
//
// 本服差异：接取时不冻结（客户端不上报结束也不会把玩家卡住；影像期间的
// 位移上报不收编，等价于官服冻结时的服务端口径）；影像在途态有超时兜底，
// 行走态位移（state 7）视作玩家已不在副本里，立即结束在途态。
//
// 自动步（占位推进）在影像期间不推进，影像结束后接着走——官服 1341 的怪
// 物是在 474 影像结束之后才刷出来的。
//
// 开关：NINEYIN_TASK_MOVIE=off 不发接取影像（彻底不进副本）；
// NINEYIN_TASK_MOVIE_REPUSH_MS 影像后重发视口 NPC 的延迟（缺省 600，0＝不重发）。

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// 客户端影像消息（client_custom_define.lua CLIENT_CUSTOMMSG_*_MOVIE）。
const (
	clientCustomBeforeEndMovie int32 = 805
	clientCustomEndMovie       int32 = 806
	clientCustomAgreeMovie     int32 = 807
)

// npcTaskMovieTimeout 影像在途态的超时兜底：客户端一直不上报结束时到点自动
// 收尾（官服最长的 472 约 62 秒，留足余量；测试可调短）。
var npcTaskMovieTimeout = 3 * time.Minute

// taskMovieState 一次任务影像的在途态（Player.taskMovie；nil＝没有影像）。
type taskMovieState struct {
	movie      int
	started    time.Time
	resumeTask uint32 // 影像期间被压后的自动步所属任务（0＝无）
}

// npcTaskMovieEnabled NINEYIN_TASK_MOVIE=off 关停任务影像。
func npcTaskMovieEnabled() bool {
	return strings.ToLower(strings.TrimSpace(os.Getenv("NINEYIN_TASK_MOVIE"))) != "off"
}

// npcTaskMovieRepushDelay 影像开始后重发视口 NPC 的延迟（官服 0.6 秒；0＝不重发）。
func npcTaskMovieRepushDelay() time.Duration {
	value := strings.TrimSpace(os.Getenv("NINEYIN_TASK_MOVIE_REPUSH_MS"))
	if value == "" {
		return 600 * time.Millisecond
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 600 * time.Millisecond
	}
	return time.Duration(n) * time.Millisecond
}

// npcTaskMoviePlaying 玩家是否有任务影像在播。
func (s *Server) npcTaskMoviePlaying(pl *Player) bool {
	pl.taskMu.Lock()
	defer pl.taskMu.Unlock()
	return pl.taskMovie != nil
}

// npcTaskStartMovie 发影像：m8 → m104，置在途态，排视口重发与超时。调用方
// 持 taskMu。
func (s *Server) npcTaskStartMovie(pl *Player, task taskdata.Task) {
	if task.AcceptMovie <= 0 || pl.taskMovie != nil {
		return
	}
	pl.taskMovie = &taskMovieState{movie: task.AcceptMovie, started: time.Now()}
	if closeTalk, err := entergen.TalkCloseFrame(); err == nil {
		s.sendToClient(pl, closeTalk)
	}
	frame, err := entergen.TaskAcceptMovieFrame(task.AcceptMovie)
	if err != nil {
		s.trace("npc task movie: 影像帧编码失败 %d: %v", task.AcceptMovie, err)
		pl.taskMovie = nil
		return
	}
	s.sendToClient(pl, frame)
	s.trace("npc task movie: 任务 %d 影像 %d 开始 client=%d", task.ID, task.AcceptMovie, pl.Conn1)
	if delay := npcTaskMovieRepushDelay(); delay > 0 {
		time.AfterFunc(delay, func() {
			if pl.State != PlOnline {
				return
			}
			s.npcTaskMovieRepush(pl, task.AcceptMovie)
		})
	}
	time.AfterFunc(npcTaskMovieTimeout, func() {
		if pl.State != PlOnline {
			return
		}
		s.npcTaskLocked(pl, func() {
			if st := pl.taskMovie; st != nil && st.movie == task.AcceptMovie && time.Since(st.started) >= npcTaskMovieTimeout {
				s.trace("npc task movie: 任务 %d 影像 %d 超时未收到结束上报，自动收尾 client=%d", task.ID, task.AcceptMovie, pl.Conn1)
				s.npcTaskMovieEnd(pl, "超时")
			}
		})
	})
}

// npcTaskMovieRepush 影像开始后重发视口 NPC 的物化帧：客户端进副本会清掉
// 场景对象，官服在影像开始约 0.6 秒后整批重发。只重发服务端视口簿记里
// 仍可见的 NPC（与 npcReconcile 同一口径），不改簿记。
func (s *Server) npcTaskMovieRepush(pl *Player, movie int) {
	if !s.npcTaskMoviePlaying(pl) {
		return
	}
	pl.npcMu.Lock()
	scene, visible := pl.npcScene, pl.npcVisible
	pl.npcMu.Unlock()
	if scene == "" || len(visible) == 0 {
		return
	}
	catalog := s.sceneNPCCatalog(scene)
	spawns := make([]npcfeed.Spawn, 0, len(visible))
	for _, spawn := range catalog.spawns {
		if visible[spawn.ObjectID] {
			spawns = append(spawns, spawn)
		}
	}
	if len(spawns) == 0 {
		return
	}
	spawns = sampleSpawns(applyResourceOverride(spawns), time.Now().UnixMilli())
	frames, err := npcfeed.Frames(spawns, npcForm())
	if err != nil {
		s.trace("npc task movie: 视口重发编码失败 client=%d: %v", pl.Conn1, err)
		return
	}
	for _, frame := range frames {
		s.sendToClient(pl, frame)
	}
	s.trace("npc task movie: 影像 %d 后重发视口 NPC %d 条 client=%d", movie, len(spawns), pl.Conn1)
}

// npcTaskMovieEnd 影像收尾：清在途态，标志复位两帧 + m107，接上被压后的
// 自动步。调用方持 taskMu。
func (s *Server) npcTaskMovieEnd(pl *Player, how string) {
	st := pl.taskMovie
	if st == nil {
		return
	}
	pl.taskMovie = nil
	id := pl.entergenIdentity()
	for _, name := range []string{"CantBeAttack", "CantMove"} {
		if frame, err := entergen.TaskPlayerFlagFrame(id, name, 0); err == nil {
			s.sendToClient(pl, frame)
		}
	}
	if frame, err := entergen.TaskMovieEndFrame(); err == nil {
		s.sendToClient(pl, frame)
	}
	s.trace("npc task movie: 影像 %d 结束（%s，历时 %s）client=%d", st.movie, how, time.Since(st.started).Round(time.Second), pl.Conn1)
	if st.resumeTask != 0 {
		s.npcTaskAutoSteps(pl, st.resumeTask)
	}
}

// handleMovieMsg C2S 影像消息：m806 [影像号] 影像结束 → 收尾簇；m805 将结束
// / m807 同意播放仅留痕。
func (s *Server) handleMovieMsg(pl *Player, message entergen.CustomInbound, sub int32) {
	movie := int32(0)
	if len(message.Values) >= 2 && message.Values[1].Type == 2 {
		movie = message.Values[1].Int32
	}
	if sub != clientCustomEndMovie {
		s.trace("npc task movie: 客户端影像消息 sub=%d 影像=%d（留痕）client=%d", sub, movie, pl.Conn1)
		return
	}
	s.npcTaskLocked(pl, func() {
		st := pl.taskMovie
		if st == nil {
			s.trace("npc task movie: 收到影像 %d 结束上报但没有在途影像 client=%d", movie, pl.Conn1)
			return
		}
		if movie != 0 && int32(st.movie) != movie {
			s.trace("npc task movie: 影像 %d 的结束上报与在途影像 %d 不符，忽略 client=%d", movie, st.movie, pl.Conn1)
			return
		}
		s.npcTaskMovieEnd(pl, "客户端上报结束")
	})
}

// npcTaskMovieDefer 影像在途时把任务的自动步压到影像结束后（返回 true＝已
// 压后，调用方不要再排推进）。调用方持 taskMu。
func (s *Server) npcTaskMovieDefer(pl *Player, taskID uint32) bool {
	st := pl.taskMovie
	if st == nil {
		return false
	}
	st.resumeTask = taskID
	return true
}
