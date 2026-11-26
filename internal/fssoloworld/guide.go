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

	// 再战江湖落位：出生地（books.ini 剧本 → location.ini 节，2026-11-26
	// 建号.pcapng 定版）。guideSkipRelive 是男号抓包保底值（#1631
	// RelivePositon；属地复活点走 rolestate.ReliveOfScene）。
	guideSkipRelive = "4,693.908,24.694,404.35,3.611,ui_pos_ReliveCJ100101"

	guideSkipItem       = "item_skipsp_001" // 再战江湖奖励物品（#1695..#1698）
	guideSkipRepute     = "repute_jianghu"  // 江湖声望奖励（#1720..#1722）
	guideSkipReputeFlag = int32(2)          // 跳过奖励的声望行标志列（任务完成奖励是 1）
)

// guideHelpStepOf 出生地 resource → freshmanhelp 帮助链名（归纳文档 §5
// 表 + 官服抓包：男号鸡鸣驿 jimingyi_00 / 女号成都线 chengdu_00）。
// 未收录出生地（洛阳/悟明坡等）→ Freshman_End 直接结链。
func guideHelpStepOf(resource string) string {
	switch resource {
	case "born03":
		return "yanyuzhuang_00"
	case "born04":
		return "qiandengzhen_00"
	case "born01":
		return guideHelpStepSkip // jimingyi_00（官服男号实证）
	case "city05":
		return guideHelpStepGuide // chengdu_00（官服女号线）
	case "city02":
		return "suzhou_00"
	}
	return "Freshman_End"
}

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
	// guideChoiceTimeoutDefault 待选兜底：客户端一直不上报 m972 时按性别
	// 焦点线自动选（男=再战江湖、女=初出茅庐）。官服没有兜底（黑幕窗必
	// 弹）；本服黑幕窗弹不出来（弹出条件在引擎里），20 秒兜底让新号零操
	// 作也能进游戏。
	guideChoiceTimeoutDefault = 20 * time.Second
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

// guideDefaultStep 待选态的引导阶段（黑幕窗默认焦点）：官服女号 #437 发
// 4/5（初出茅庐）、男号（建号.pcapng 42.43s，book2）发 5/5（再战江湖）
// ——两样本与剧本无关、随性别。点选 m972 仍按玩家实际点的受理。
func guideDefaultStep(sex int) int {
	if sex == 2 {
		return guideStepSingle
	}
	return guideStepSkip
}

// guideEntryStep 进场簇下发的引导阶段（entergen.GuideEntryFrames 输入）：
// 已定＝存档 GameStep（重登落定态，官服重登同发）；待选且身在新手副本
// ＝性别默认；其余（副本外的存量号）0＝不进场面簇。
func guideEntryStep(pl *Player) int {
	if !pl.haveRoleState {
		return 0
	}
	if pl.roleState.GameStep != guideStepNone {
		return pl.roleState.GameStep
	}
	if guideSceneOf(pl) {
		return guideDefaultStep(pl.roleState.Sex)
	}
	return 0
}

// guideEntryChoicePending 进场簇的待选判据（guideEntryStep「待选」分支的
// 原条件，无锁——roleContext 的调用链有持 combatState.mu 的路径，进不了
// taskMu 做完整 guideChoicePending 核查）：身在新手副本且引导阶段未定。
// 已定档（含选过初出茅庐/再战江湖）不再发门控属性，避免重登重弹黑幕窗。
func guideEntryChoicePending(pl *Player) bool {
	return pl.haveRoleState && pl.roleState.GameStep == guideStepNone && guideSceneOf(pl)
}

// guideEntryHelpStep 进场簇的 HelpStepID：已入链用存档值（m950 推进过）；
// 待选副本号用出生地属地链（官服男号 book2 实证 idx129="yanyuzhuang_00"）。
// 旧档 Birth 字段缺席回落 guideHelpStepOf 的烟雨庄默认。
func guideEntryHelpStep(pl *Player) string {
	if !pl.haveRoleState {
		return ""
	}
	if pl.roleState.HelpStepID != "" {
		return pl.roleState.HelpStepID
	}
	if guideSceneOf(pl) {
		return guideHelpStepOf(pl.roleState.BirthResource)
	}
	return ""
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

// readyTail 就绪簇收尾（首次进场与切图重入共用）：引导阶段簇（m414/
// BornServerGameStep/HelpStepID）已在进场簇下发（entergen.GuideEntryFrames，
// m655 后、0x0B 前——黑幕窗时机），此处只剩待选态锁输入（官服 49.62s
// idx249=1，NPC 物化之后的就绪簇）+ 所有角色 m390 刷新任务追踪栏（追踪栏
// 窗体进场预载后是隐藏的，只在它之后显示；官服 50.24s）。
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
	pending := false
	s.npcTaskLocked(pl, func() {
		if guideSceneOf(pl) && pl.roleState.GameStep == guideStepNone {
			pending = s.guideChoicePending(pl)
			if pending && pl.roleState.HelpStepID == "" {
				pl.roleState.HelpStepID = guideHelpStepOf(pl.roleState.BirthResource)
			}
		}
		pl.guidePending = pending
	})
	if !pending {
		return
	}
	if frame, err := entergen.InputLockFrame(pl.entergenIdentity(), true); err == nil {
		s.sendToClient(pl, frame)
	}
	timeout := guideChoiceTimeout()
	s.trace("npc task guide: 待选态 client=%d scene=%s（锁输入，等 m972 [4]/[5]；兜底 %s）",
		pl.Conn1, pl.roleState.Resource, timeout)
	if timeout <= 0 {
		return
	}
	sex := pl.roleState.Sex
	time.AfterFunc(timeout, func() {
		if pl.State != PlOnline || !s.guideIsPending(pl) {
			return
		}
		// 2026-11-27：黑幕窗（form_single_blackback，引擎按号弹）在本服
		// 客户端弹不出来，玩家面对锁死的场景（真机「NPC 不物化、任务列
		// 表空」实为待选态卡住）——兜底从 300s 降为 20s，且按性别焦点线
		// 自动选（与 m414 下发的默认焦点一致：男=再战江湖切出副本、女=
		// 初出茅庐进引导），玩家零操作也能进游戏。
		s.trace("npc task guide: 待选态 %s 未收到 m972，按性别焦点（sex=%d → step=%d）推进 client=%d",
			timeout, sex, guideDefaultStep(sex), pl.Conn1)
		s.guideChoose(pl, guideDefaultStep(sex))
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

// guideSkip 再战江湖：跳过簇 + 切出新手副本进出生地（books.ini 剧本 →
// location.ini 出生地节；官服男号抓包 → 鸡鸣驿只是 book5 个案）。
func (s *Server) guideSkip(pl *Player) {
	if !pl.haveRoleState {
		return
	}
	// 出生地：优先档内 Birth 字段（BuildRoleState 建档写入）；旧档缺席
	// 按剧本（外观串）重解析；链尽头回落默认烟雨庄。
	place := rolestate.Birthplace{}
	if pl.roleState.BirthScene != "" {
		place.Section, place.Scene = pl.roleState.BirthSection, pl.roleState.BirthScene
		place.Resource, place.Position = pl.roleState.BirthResource, pl.roleState.BirthPosition
	} else {
		place = rolestate.BirthplaceOf(pl.roleState.Book)
	}
	if place.Scene == "" {
		place = rolestate.BirthplaceOf(rolestate.DefaultBook)
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
			s.guideSkipFrames(pl, place)
			pl.roleState.GameStep, pl.guidePending = guideStepSkip, false
			done = true
		}
	})
	if !done {
		return
	}
	s.trace("npc task guide: 再战江湖 client=%d → 出生地 %s", pl.Conn1, place.Section)
	s.sendRoleSave(pl)
	dest := sceneDestination{Config: place.Scene, Resource: place.Resource}
	dest.Position = place.Position
	if err := s.transferPlayer(pl, dest, "guide skip"); err != nil {
		s.trace("npc task guide: 切出新手副本失败 client=%d: %v", pl.Conn1, err)
	}
}

// guideSkipFrames 跳过簇（改状态、下发；调用方持 taskMu）。帧序按男号抓包：
// 后续任务接取 → 复活点 → 引导链逐个置完成位 → 奖励 → 入江湖属性。
func (s *Server) guideSkipFrames(pl *Player, place rolestate.Birthplace) {
	id := pl.entergenIdentity()
	out := s.taskFrameSink(pl)
	// 1. 后续任务接取（行 + 接取通知，男号 #1612..#1630）。
	for _, taskID := range guideSkipTasks {
		s.npcTaskAcceptFrames(pl, out, taskID)
	}
	// 2. 复活点 → 出生地（#1631 官服男号=鸡鸣驿 ui_pos_ReliveCJ100101；
	//    属地化按 location.ini 同场景 ReliveCJ 节组串，缺席保底抓包值）。
	relive := guideSkipRelive
	if r, ok := rolestate.ReliveOfScene(place.Scene); ok {
		relive = r
	}
	pl.roleState.RelivePositon = relive
	out.add(entergen.ReliveFrame(id, relive))
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
	// 5. 入江湖：帮助链切到出生地对应 freshmanhelp 链 → PowerLevel/
	//    YinLevel → 解锁输入（#1946..#1948；官服男号=鸡鸣驿 jimingyi_00，
	//    链名按出生地 resource 映射，未收录的出生地直接结链 Freshman_End
	//    ——归纳文档 §5「非出生场景进入则置 Freshman_End」同义）。
	step := guideHelpStepOf(place.Resource)
	pl.roleState.HelpStepID = step
	out.add(entergen.HelpStepFrame(id, step))
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
