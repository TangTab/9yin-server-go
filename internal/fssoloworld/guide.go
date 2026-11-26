package fssoloworld

// guide.go 新手引导选择流（官服女号「完整引导」/ 男号「跳过引导」两份抓包
// 定版；帧编码见 entergen/guide.go）：
//
//	进场（新手副本 born_single_01，就绪簇收尾）
//	      m414 ["gamestep", 主步, 5] → 0x10 HelpStepID → 待选态 0x10
//	      CanntOperator=1（锁输入，等客户端黑幕窗「初出茅庐／再战江湖」）
//	      → m390 [5, 0]（所有角色：刷新任务追踪栏）
//	C2S m972 [4]（初出茅庐）   0x10 CanntOperator=0 → 约 1 秒后自动接取引导链根
//	                          （1340 接取簇 + 接取影像，taskrecord/taskmovie）
//	C2S m972 [5]（再战江湖）   跳过簇（男号 #1612..#1989）：后续任务接取 → 复活点
//	                          → 引导链逐个置完成位 → 奖励（item_skipsp_001、
//	                          历练、江湖声望 700）→ 解锁 → 切出副本进鸡鸣驿
//	C2S m950 [0, 步骤]        新手帮助步骤链推进：回写 0x10 HelpStepID
//
// 待选态只认「身在新手副本 + 引导阶段未定 + 引导链未开始」的角色；其余角色
// 行为不变（链根延时自动接取照旧）。黑幕窗由客户端引擎弹出，服务端只在
// 点选后收到 m972；客户端一直不上报时按兜底计时视同初出茅庐
// （NINEYIN_GUIDE_CHOICE_TIMEOUT 秒，0＝一直等）。

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// 引导阶段（客户端 SwitchManager 主步；C2S m972 回传同一值）。
const (
	guideStepNone   = 0 // 未定（旧档／新号待选）
	guideStepSingle = 4 // 初出茅庐：单机引导
	guideStepSkip   = 5 // 再战江湖：跳过引导、直接入江湖
	// guideStepSub m414 ["gamestep", 主步, 子步] 的子步（官服两号同为 5）。
	guideStepSub int32 = 5

	// guideScene 新手副本（单机引导场景）resource；新号默认出生地。
	guideScene = rolestate.GuideScene
	// guideHelpStepGuide / guideHelpStepSkip 新手帮助链起点（freshmanhelp.ini
	// 节名）：女号 "chengdu_00"（引导线）、男号 "jimingyi_00"（跳过 → 鸡鸣驿）。
	guideHelpStepGuide = "chengdu_00"
	guideHelpStepSkip  = "jimingyi_00"

	// 再战江湖：进入鸡鸣驿 born01。落点与复活点串取男号抓包原值
	// （#2185 0x1F、#1631 RelivePositon）。
	guideSkipScene  = "born01"
	guideSkipRelive = "4,693.908,24.694,404.35,3.611,ui_pos_ReliveCJ100101"

	guideSkipItem       = "item_skipsp_001" // 再战江湖奖励物品（#1695..#1698）
	guideSkipRepute     = "repute_jianghu"  // 江湖声望奖励（#1720..#1722）
	guideSkipReputeFlag = int32(2)          // 跳过奖励的声望行标志列（任务完成奖励是 1）
)

// guideSkipLanding 再战江湖落点（鸡鸣驿，官服 0x1F 原值）。
var guideSkipLanding = [4]float64{728.968, 23.951818, 354.622, 2.828}

// guideSkipReputeCount 江湖声望奖励值。
const guideSkipReputeCount int32 = 700

// guideSkipTasks 跳过后承接的江湖任务（按已接表行序）：1647「初入江湖」车夫
// 剧情、28111「近乡情怯」鸡鸣驿事件（男号 #1612..#1630）。
var guideSkipTasks = []int{1647, 28111}

// guideSkipDoneTasks 跳过时置完成位的任务（男号 #1674..#1694 逐位写序）：
// 1339「江湖名俊」＋新手引导链 1340~1349、1597。
var guideSkipDoneTasks = []int{1339, 1340, 1341, 1342, 1343, 1344, 1345, 1346, 1347, 1348, 1349, 1597}

// guideSkipLiveGrooves 历练奖励：LiveGroove 下标 → 累加值（男号 #1700..#1719；
// 提示点数＝累加值/1000）。官服第 5 档余额落到 33000 是因为新号登录基线已有
// 1000（女 #1367 / 男 #1461）；本服建号基线是 0，余额相应少 1000。
var guideSkipLiveGrooves = [][2]int32{{1, 5000}, {2, 5000}, {3, 10000}, {4, 15000}, {5, 32000}}

var (
	// guideStartDelay m972 [4] 之后到引导链根接取簇的间隔（女号 31.924s 点选
	// → 32.969s 接取簇）。
	guideStartDelay = time.Second
	// guideChoiceTimeoutDefault 待选态兜底：客户端一直不上报 m972 时视同
	// 初出茅庐。官服没有兜底（黑幕窗必弹）；本服黑幕窗的弹出条件在引擎
	// 里，留一道出口免得输入锁死。
	guideChoiceTimeoutDefault = 300 * time.Second
)

// guideChoiceTimeout 待选兜底时长（NINEYIN_GUIDE_CHOICE_TIMEOUT 秒，0＝一直等）。
func guideChoiceTimeout() time.Duration {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv("NINEYIN_GUIDE_CHOICE_TIMEOUT"))); err == nil && v >= 0 {
		return time.Duration(v) * time.Second
	}
	return guideChoiceTimeoutDefault
}

// guideSceneOf 玩家是否身在新手副本。
func guideSceneOf(pl *Player) bool {
	return strings.EqualFold(strings.TrimSpace(pl.roleState.Resource), guideScene)
}

// guideChoicePending 待选态：身在新手副本、引导阶段未定、引导链未开始
// （没有任何引导链任务的进度行，也没有非种子种下的引导链完成位）。
// 调用方持 taskMu。
func (s *Server) guideChoicePending(pl *Player) bool {
	if !pl.haveRoleState || pl.roleState.GameStep != guideStepNone || !guideSceneOf(pl) {
		return false
	}
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		return false
	}
	_, seedCompleted := rolestate.SeedTaskIDs()
	seed := make(map[uint32]bool, len(seedCompleted))
	for _, id := range seedCompleted {
		seed[id] = true
	}
	isGuide := func(id uint32) bool {
		task, ok := catalog.TaskByID(int(id))
		return ok && task.IsGuide
	}
	for _, track := range pl.roleState.TaskTracks {
		if isGuide(track.TaskID) {
			return false
		}
	}
	for _, id := range pl.roleState.TaskCompleted {
		if !seed[id] && isGuide(id) {
			return false
		}
	}
	return true
}

// guideIsPending 会话待选标志（taskMu 保护）。
func (s *Server) guideIsPending(pl *Player) bool {
	pending := false
	s.npcTaskLocked(pl, func() { pending = pl.guidePending })
	return pending
}

// readyTail 就绪簇收尾（首次进场与切图重入共用）：新手副本角色下发引导
// 阶段——待选态锁输入并起兜底；最后给所有角色发 m390 刷新任务追踪栏
// （客户端追踪栏窗体进场预载后是隐藏的，只在它之后显示）。
func (s *Server) readyTail(pl *Player) {
	defer func() {
		if frame, err := entergen.TaskTrackRefreshFrame(); err != nil {
			s.trace("npc task: 追踪栏刷新帧编码失败 client=%d: %v", pl.Conn1, err)
		} else {
			s.sendToClient(pl, frame)
		}
	}()
	if !pl.haveRoleState {
		return
	}
	step, pending := pl.roleState.GameStep, false
	s.npcTaskLocked(pl, func() {
		if guideSceneOf(pl) && step == guideStepNone {
			// 未定的引导阶段按初出茅庐下发（官服女号登录 4/5）；是否待选
			// 另按引导链进度判。
			step = guideStepSingle
			pending = s.guideChoicePending(pl)
			if pending && pl.roleState.HelpStepID == "" {
				pl.roleState.HelpStepID = guideHelpStepGuide
			}
		}
		pl.guidePending = pending
	})
	if step == guideStepNone {
		return // 从没进过引导流的角色：不下发引导阶段
	}
	id := pl.entergenIdentity()
	out := s.taskFrameSink(pl)
	out.add(entergen.BornGameStepFrame(id, int32(step)*10+guideStepSub))
	out.add(entergen.GameStepFrame(int32(step), guideStepSub))
	if pl.roleState.HelpStepID != "" {
		out.add(entergen.HelpStepFrame(id, pl.roleState.HelpStepID))
	}
	if pending {
		out.add(entergen.InputLockFrame(id, true))
	}
	s.sendTaskFrames(pl, out)
	if !pending {
		return
	}
	timeout := guideChoiceTimeout()
	s.trace("npc task guide: 待选态 client=%d scene=%s（锁输入，等 m972 [4]/[5]；兜底 %s）",
		pl.Conn1, pl.roleState.Resource, timeout)
	if timeout <= 0 {
		return
	}
	time.AfterFunc(timeout, func() {
		if pl.State != PlOnline || !s.guideIsPending(pl) {
			return
		}
		s.trace("npc task guide: 待选态 %s 未收到 m972，按初出茅庐推进 client=%d", timeout, pl.Conn1)
		s.guideChoose(pl, guideStepSingle)
	})
}

// guideChoose 引导选择（C2S m972 与 gm guide 共用入口）：4＝初出茅庐，
// 5＝再战江湖。
func (s *Server) guideChoose(pl *Player, step int) {
	switch step {
	case guideStepSingle:
		s.guideBeginSingle(pl)
	case guideStepSkip:
		s.guideSkip(pl)
	default:
		s.trace("npc task guide: 未受理的选择 step=%d client=%d", step, pl.Conn1)
	}
}

// guideBeginSingle 初出茅庐：解锁输入，约 1 秒后自动接取本场景的引导链根。
func (s *Server) guideBeginSingle(pl *Player) {
	if !pl.haveRoleState {
		return
	}
	done := false
	s.npcTaskLocked(pl, func() {
		if !s.guideChoicePending(pl) {
			s.trace("npc task guide: 初出茅庐未受理（不在待选态）client=%d step=%d", pl.Conn1, pl.roleState.GameStep)
			return
		}
		pl.roleState.GameStep, pl.guidePending = guideStepSingle, false
		done = true
		s.trace("npc task guide: 初出茅庐 client=%d", pl.Conn1)
	})
	if !done {
		return
	}
	if frame, err := entergen.InputLockFrame(pl.entergenIdentity(), false); err == nil {
		s.sendToClient(pl, frame) // 官服 #1532：点选后 41ms 解锁
	}
	s.sendRoleSave(pl)
	time.AfterFunc(guideStartDelay, func() {
		if pl.State != PlOnline || !guideSceneOf(pl) {
			return
		}
		s.npcAcceptSceneGuideRoots(pl)
	})
}

// guideSkip 再战江湖：跳过簇 + 切出新手副本进鸡鸣驿。
func (s *Server) guideSkip(pl *Player) {
	if !pl.haveRoleState {
		return
	}
	done := false
	s.npcTaskLocked(pl, func() {
		switch {
		case pl.roleState.GameStep == guideStepSkip:
			s.trace("npc task guide: 再战江湖已办过 client=%d", pl.Conn1)
		case !s.guideChoicePending(pl):
			s.trace("npc task guide: 再战江湖未受理（不在待选态）client=%d step=%d 引导进度 %d 条",
				pl.Conn1, pl.roleState.GameStep, len(pl.roleState.TaskTracks))
		default:
			s.guideSkipFrames(pl)
			pl.roleState.GameStep, pl.guidePending = guideStepSkip, false
			done = true
		}
	})
	if !done {
		return
	}
	s.trace("npc task guide: 再战江湖 client=%d → %s", pl.Conn1, guideSkipScene)
	s.sendRoleSave(pl)
	dest, ok := resolveSceneDestination(guideSkipScene)
	if !ok {
		dest = sceneDestination{Config: `ini\scene\born01_MoHeZhen`, Resource: guideSkipScene}
	}
	dest.Position = guideSkipLanding
	if err := s.transferPlayer(pl, dest, "guide skip"); err != nil {
		s.trace("npc task guide: 切出新手副本失败 client=%d: %v", pl.Conn1, err)
	}
}

// guideSkipFrames 跳过簇（改状态、下发；调用方持 taskMu）。帧序按男号抓包：
// 后续任务接取 → 复活点 → 引导链逐个置完成位 → 奖励 → 入江湖属性。
func (s *Server) guideSkipFrames(pl *Player) {
	id := pl.entergenIdentity()
	out := s.taskFrameSink(pl)
	// 1. 后续任务接取（行 + 接取通知，男号 #1612..#1630）。
	for _, taskID := range guideSkipTasks {
		s.npcTaskAcceptFrames(pl, out, taskID)
	}
	// 2. 复活点 → 鸡鸣驿（#1631）。
	pl.roleState.RelivePositon = guideSkipRelive
	out.add(entergen.ReliveFrame(id, guideSkipRelive))
	// 3. 引导链逐个置完成位（#1674..#1694；1346 置位后官服插一帧 m543）。
	completed := append([]uint32(nil), pl.roleState.TaskCompleted...)
	for n, taskID := range guideSkipDoneTasks {
		if n == 8 {
			out.add(entergen.SingleStepMarkFrame())
		}
		if pl.taskCompletedContains(uint32(taskID)) {
			continue
		}
		out.addAll(entergen.TaskCompletedSetFrames(id.ObjectID, id.OwnerID, uint32(taskID), completed))
		pl.roleState.TaskCompleted = append(pl.roleState.TaskCompleted, uint32(taskID))
		completed = append(completed, uint32(taskID))
	}
	// 4. 奖励：item_skipsp_001（GiveItemRec 行 + 入包 + m1 9645，#1695..#1698）
	//    → 历练（#1700..#1719）→ 江湖声望 700（#1720..#1727）。
	out.add(entergen.GiveItemRowFrame(id.ObjectID, id.OwnerID, 0, guideSkipItem, 1))
	s.npcTaskGrantItems(pl, out, []taskdata.PrizeItem{{ID: guideSkipItem, Num: 1}}, "npc task guide: 再战江湖奖励", false)
	for _, gain := range guideSkipLiveGrooves {
		total, _ := pl.roleState.AddLiveGroove(int(gain[0]), gain[1])
		out.addAll(entergen.LiveGrooveGainFrames(id, int(gain[0]), gain[1], total))
	}
	row, total, added := pl.roleState.AddRepute(guideSkipRepute, guideSkipReputeCount)
	if added {
		out.add(entergen.ReputeRowFrameWithFlag(id.ObjectID, id.OwnerID, uint16(row), guideSkipRepute, total, guideSkipReputeFlag))
	} else {
		out.add(entergen.TaskReputeCountFrame(id.ObjectID, id.OwnerID, uint16(row), total))
	}
	out.add(entergen.FunctionOpenNoticeFrame(177))
	out.add(entergen.TaskReputeNotice(guideSkipRepute, guideSkipReputeCount))
	out.addAll(entergen.LifeHelpUnlockFrames())
	// 5. 入江湖：帮助链切到鸡鸣驿 → PowerLevel/YinLevel → 解锁输入
	//    （#1946..#1948）。
	pl.roleState.HelpStepID = guideHelpStepSkip
	out.add(entergen.HelpStepFrame(id, guideHelpStepSkip))
	out.addAll(entergen.EnterWorldFrames(id))
	out.add(entergen.InputLockFrame(id, false))
	s.sendTaskFrames(pl, out)
}

// handleGuideChoice C2S m972 单机引导步骤：黑幕窗点选回传 [4]（初出茅庐）
// 或 [5]（再战江湖）。Lua 数值到线是 double（ExactInt32 兼容整型）。
func (s *Server) handleGuideChoice(pl *Player, message entergen.CustomInbound) {
	if len(message.Values) < 2 {
		return
	}
	step, ok := message.Values[1].ExactInt32()
	if !ok {
		s.trace("npc task guide: m972 参数非整数 client=%d", pl.Conn1)
		return
	}
	s.guideChoose(pl, int(step))
}

// handleHelpStep C2S m950 [0, "<下一步>"]：新手帮助步骤链推进——步骤名
// 存档并回写 0x10 HelpStepID（官服 #1792）。客户端按新步骤继续放动画、
// 开界面、亮特效（freshmanhelp.ini）。
func (s *Server) handleHelpStep(pl *Player, message entergen.CustomInbound) {
	if !pl.haveRoleState || len(message.Values) < 3 || (message.Values[2].Type != 6 && message.Values[2].Type != 7) {
		return
	}
	step := strings.TrimSpace(message.Values[2].Text)
	if step == "" {
		return
	}
	s.npcTaskLocked(pl, func() {
		pl.roleState.HelpStepID = step
		if frame, err := entergen.HelpStepFrame(pl.entergenIdentity(), step); err == nil {
			s.sendToClient(pl, frame)
		}
	})
	s.trace("npc task guide: 新手帮助步骤 %s client=%d", step, pl.Conn1)
	s.sendRoleSave(pl)
}

// gmGuideCommand gm guide 子命令：gm guide <who> <start|skip>——不等客户端
// 黑幕窗，直接走初出茅庐 / 再战江湖（真机联调入口）。
func (s *Server) gmGuideCommand(nc net.Conn, pl *Player, who string, fields []string) {
	if len(fields) < 3 {
		fmt.Fprint(nc, ">usage: gm guide <who> <start|skip> (start=初出茅庐进引导 skip=再战江湖跳过；需在新手副本待选态)\r")
		return
	}
	var step int
	var effect string
	switch fields[2] {
	case "start":
		step, effect = guideStepSingle, "解锁 → 1s 后接取引导链根"
	case "skip":
		step, effect = guideStepSkip, "跳过簇 → 鸡鸣驿"
	default:
		fmt.Fprintf(nc, ">gm guide: unknown action %q (start|skip)\r", fields[2])
		return
	}
	pending := false
	s.npcTaskLocked(pl, func() { pending = s.guideChoicePending(pl) })
	if !pending {
		fmt.Fprintf(nc, ">gm guide %s: not pending (scene=%s step=%d)——gm scene %s 新手引导 进副本\r",
			who, pl.roleState.Resource, pl.roleState.GameStep, who)
		return
	}
	s.guideChoose(pl, step)
	fmt.Fprintf(nc, ">gm guide %s: %s (%s)\r", who, fields[2], effect)
}
