package fssoloworld

// npctask.go NPC 任务绑定（对话框任务行 + 接取/步进/提交受理）：
//
// 数据：任务目录（taskdata.LoadQuest——数据根带客户端 share/task/task 与
// 文案档时直读，否则读 datagen npc 生成的任务域 XML <数据根>/ini/Quest；
// 任务 ID 与客户端文案键同一体系；Task 主表 + SubFunction×各步骤域）。
//
// 协议（form_talk_data.lua g_menu_id 域 + 官服引导抓包定版）：菜单任务行
// m6 funcid=flag*10⁶+TID，行文本双键（首键=行文案、次键 title_<TID>）。
// on_endmenu 只聚合 on_addmenu 收到的行——客户端不自行生成任务行，服务端
// 不发即无任务入口。
//
// 主窗行规则（见 npcTaskLines）：
//   - 已接且当前步是绑定到本 NPC 的剧情步：500 进行中行（story_menu_<剧情
//     组 ID> + title_<TID>），点击进剧情窗；
//   - 已接且可提交（非自动提交）：200 提交行（CompleteMenu_<TID>）；
//   - 已接的交互步不出主窗行：点 NPC 时在主窗之后追加「步骤确认窗」
//     （m4/m5 <DialogID>/m6 [500×10⁶+TID, okmenu_<步骤号>]/m6 离开/m7）；
//   - 未接：仅当任务有接取文案键（主表 CanAcceptMenu）才发 100 接取行
//     ——客户端没有该键时发了只会显示原始键名；链根任务由服务端自动
//     接取（进场/首谈），主窗不出接取行；
//   - 已交任务不出行。
//
// 受理：玩家点任务行回 op0x07 FuncID=任务域 mark。
//   100 接取（两次点击：首次推接取窗，窗内「接受任务」行同 funcid 二次
//       到达=真接取）；
//   200 只回提交窗参数 m209 [2,…]（客户端弹提交窗，玩家确认后回 201）；
//   201 提交确认=完成簇（taskrecord.go）；
//   500 剧情窗第 1 段／交互步确认（按任务当前步类型分流）；
//   501..599 剧情窗 N 的续行：点击发窗 N+1，末窗续行=剧情步完成；
//   601..699 剧情中断关窗。
//
// 任务状态同步（已接/目标行、步进、完成、追踪）见 taskrecord.go。

import (
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// npcTaskFuncBase g_menu_id.base：funcid = flag*base + TID。
const npcTaskFuncBase = 1000000

// 任务行 flag 域（form_talk_data.lua g_menu_id；官服引导抓包实证）。
const (
	npcTaskFlagAccept       int32 = 100 // 接取
	npcTaskFlagSubmit       int32 = 200 // 提交（回提交窗参数）
	npcTaskFlagSubmitChoice int32 = 201 // 提交窗内确认（lua menu_select 二次到达）
	npcTaskFlagDoing        int32 = 500 // 进行中：主窗剧情行／步骤确认窗确认行
	// 剧情窗 N（N=1..99）：续行 funcid=(500+N)×base+TID——点击发窗 N+1，
	// 末窗的续行=剧情步完成；离行 (600+N)×base+TID。
	npcTaskFlagStoryContinueBase int32 = 500
	npcTaskFlagStoryLeaveBase    int32 = 600
	npcTaskStoryMaxSegments      int32 = 99
	// npcStoryLeaveTextDefault 剧情离行缺省文本键（源表 LeaveTextID 列几乎
	// 恒 9607）。
	npcStoryLeaveTextDefault = "9607"
	// npcTalkLeaveFuncID 对话「离开」行 funcid（官方 600000000+"ui_exit"）。
	npcTalkLeaveFuncID int32 = 600000000
)

// npcTaskLine 一条主窗任务行。
type npcTaskLine struct {
	TaskID    int
	Submit    bool   // 200 可提交行
	StoryID   string // 500 行：当前剧情步的剧情组 ID（story_menu_<组 ID>）
	AcceptKey string // 100 行文案键（任务主表 CanAcceptMenu）
}

// ---- 任务目录装载（进程级缓存） ----

var (
	npcTaskMu        sync.Mutex
	npcTaskCatalog   *taskdata.Catalog
	npcTaskRootTried string
)

// taskCatalogForRoot 装载任务目录（按数据根缓存；缺档返回 nil——任务行
// 是增强下发，缺数据不阻塞对话）。数据源见 taskdata.LoadQuest。
func taskCatalogForRoot(root string) *taskdata.Catalog {
	npcTaskMu.Lock()
	defer npcTaskMu.Unlock()
	if root == npcTaskRootTried {
		return npcTaskCatalog // 含 nil 负面缓存（进程级，不重试）
	}
	npcTaskRootTried = root
	npcTaskCatalog = nil
	if root == "" {
		return nil
	}
	if catalog, err := taskdata.LoadQuest(root); err == nil && catalog.Count() > 0 {
		npcTaskCatalog = catalog
	}
	return npcTaskCatalog
}

// npcTaskMenuLimit NINEYIN_NPC_TASK_LIMIT 单 NPC 任务行上限（缺省 8——
// 客户端菜单每页 4 行翻页，过多任务行淹没服务行；<0 一键关停任务行）。
func npcTaskMenuLimit() int { return int(envFloat("NINEYIN_NPC_TASK_LIMIT", 8)) }

// npcTaskStepAt 任务当前步是否就在这个 NPC 且尚未完成（是则返回该步）。
// 末步记满后步序仍指向它——这时任务已可提交，不再出进行中行/确认窗。
func npcTaskStepAt(task taskdata.Task, track *rolestate.TaskTrack, configID string) (taskdata.Step, bool) {
	if track == nil || track.Step < 1 || track.Step > len(task.Steps) {
		return taskdata.Step{}, false
	}
	step := task.Steps[track.Step-1]
	if npcTaskStepDone(track, step) || !npcTaskStepAtNpc(step, configID) {
		return taskdata.Step{}, false
	}
	return step, true
}

// npcTaskStepDone 当前步是否已记满进度。
func npcTaskStepDone(track *rolestate.TaskTrack, step taskdata.Step) bool {
	k := track.Step - 1
	return k < len(track.Cur) && track.Cur[k] >= npcTaskStepMax(step)
}

// npcTaskStepWindow NPC 当前可确认的交互步：已接任务的当前步是绑定到本
// NPC、带对话的交互步（官服：点 NPC 时主窗之后追加步骤确认窗）。
func (s *Server) npcTaskStepWindow(catalog *taskdata.Catalog, pl *Player, configID string) (taskdata.Task, taskdata.Step, bool) {
	if catalog == nil {
		return taskdata.Task{}, taskdata.Step{}, false
	}
	for _, task := range catalog.TasksForNPC(configID) {
		track := pl.roleState.TaskTrackOf(uint32(task.ID))
		step, ok := npcTaskStepAt(task, track, configID)
		if !ok || step.Subtype != taskdata.StepInteract || step.Scene == "" {
			continue
		}
		return task, step, true
	}
	return taskdata.Task{}, taskdata.Step{}, false
}

// npcTaskLines NPC 的主窗任务行集：绑定且在目录里的任务按 TID 升序——
// 已接且当前步是本 NPC 的剧情步=500 行；已接且可提交（非自动提交）=200
// 行；未接且有接取文案=100 行。已接的交互步、已交任务与无接取文案的未接
// 任务不出行。截断至上限。
func (s *Server) npcTaskLines(catalog *taskdata.Catalog, pl *Player, configID string, limit int) []npcTaskLine {
	if catalog == nil || limit < 0 || strings.TrimSpace(configID) == "" {
		return nil
	}
	tasks := catalog.TasksForNPC(configID)
	lines := make([]npcTaskLine, 0, len(tasks))
	for _, task := range tasks {
		if limit > 0 && len(lines) >= limit {
			break
		}
		tid := uint32(task.ID)
		if pl.taskAcceptedContains(tid) {
			track := pl.roleState.TaskTrackOf(tid)
			if track == nil {
				continue // 建号种子的功能门行/目录外：不是玩家接的任务
			}
			switch {
			case entergen.TaskSubmittable(track, task.Steps):
				if !task.AutoSubmit {
					lines = append(lines, npcTaskLine{TaskID: task.ID, Submit: true})
				}
			default:
				if step, ok := npcTaskStepAt(task, track, configID); ok && step.Subtype == taskdata.StepStory {
					lines = append(lines, npcTaskLine{TaskID: task.ID, StoryID: strconv.Itoa(step.Index)})
				}
			}
			continue
		}
		if pl.taskCompletedContains(tid) {
			continue
		}
		if task.AcceptMenuKey != "" {
			lines = append(lines, npcTaskLine{TaskID: task.ID, AcceptKey: task.AcceptMenuKey})
		}
	}
	return lines
}

// autoAcceptChainStarts 链根任务的「首谈即接」：绑定到该 NPC、无接取文案、
// 未接未交的链根任务由服务端直接接取（车夫抓包：1648@Transcity05D 首次
// 点击即发进行中行；引导链首环 1340 官服是进场自动接取，本服两种触发
// 都有，见 npcAcceptSceneGuideRoots）。返回本次新接的任务号。调用方持
// taskMu。
func (s *Server) autoAcceptChainStarts(catalog *taskdata.Catalog, pl *Player, configID string) []int {
	if catalog == nil || strings.TrimSpace(configID) == "" {
		return nil
	}
	var added []int
	for _, task := range catalog.TasksForNPC(configID) {
		tid := uint32(task.ID)
		if !task.ChainRoot || task.AcceptMenuKey != "" || pl.taskAcceptedContains(tid) || pl.taskCompletedContains(tid) {
			continue
		}
		if s.npcTaskAccept(pl, task.ID, "链根首谈") {
			added = append(added, task.ID)
		}
	}
	return added
}

// npcAcceptedSet 玩家已接任务集合。
func npcAcceptedSet(pl *Player) map[uint32]bool {
	set := make(map[uint32]bool, len(pl.roleState.TaskAccepted))
	for _, id := range pl.roleState.TaskAccepted {
		set[id] = true
	}
	return set
}

// npcCompletedSet 玩家已交任务集合（roleState.TaskCompleted——建号种子与
// 提交受理共同维护）。
func npcCompletedSet(pl *Player) map[uint32]bool {
	set := make(map[uint32]bool, len(pl.roleState.TaskCompleted))
	for _, id := range pl.roleState.TaskCompleted {
		set[id] = true
	}
	return set
}

// npcTaskMenuFrames 任务行 m6 帧族（服务行之前——客户端 talk_data_add_save_menu
// 按域分区聚合，任务行归 menu_task_main 置顶）。行文本双键：首键=菜单行
// 文案（接取 CanAcceptMenu_<TID> / 提交 CompleteMenu_<TID> / 进行中
// story_menu_<剧情组 ID>），次键 title_<TID> 作 TextManager 格式化参数。
func npcTaskMenuFrames(lines []npcTaskLine) ([][]byte, error) {
	frames := make([][]byte, 0, len(lines))
	for _, line := range lines {
		flag, textKey := npcTaskFlagAccept, "CanAcceptMenu_"+strconv.Itoa(line.TaskID)
		switch {
		case line.Submit:
			flag, textKey = npcTaskFlagSubmit, "CompleteMenu_"+strconv.Itoa(line.TaskID)
		case line.StoryID != "":
			flag, textKey = npcTaskFlagDoing, "story_menu_"+line.StoryID
		default:
			if line.AcceptKey != "" {
				textKey = line.AcceptKey
			}
		}
		frame, err := entergen.CustomMessageFrame(6,
			entergen.CustomInt(flag*npcTaskFuncBase+int32(line.TaskID)),
			entergen.CustomString(textKey),
			entergen.CustomString("title_"+strconv.Itoa(line.TaskID)))
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// npcTaskStepWindowFrames 步骤确认窗（官服帧族：点 NPC 时主窗之后紧接着的
// 第二个对话窗）：m4 / m5 <DialogID> / m6 [500×10⁶+TID, okmenu_<步骤号>] /
// m6 [600000000, "ui_form_talk_leave"] / m7。
func npcTaskStepWindowFrames(objectID, ownerID uint32, taskID int, step taskdata.Step) ([][]byte, error) {
	frames := make([][]byte, 0, 5)
	add := func(frame []byte, err error) error {
		if err != nil {
			return err
		}
		frames = append(frames, frame)
		return nil
	}
	if err := add(entergen.CustomMessageFrame(4)); err != nil {
		return nil, err
	}
	if err := add(entergen.CustomMessageFrame(5, entergen.CustomString(step.Scene))); err != nil {
		return nil, err
	}
	if err := add(entergen.CustomMessageFrame(6,
		entergen.CustomInt(npcTaskFlagDoing*npcTaskFuncBase+int32(taskID)),
		entergen.CustomString(step.OkMenu))); err != nil {
		return nil, err
	}
	if err := add(entergen.CustomMessageFrame(6, entergen.CustomInt(npcTalkLeaveFuncID), entergen.CustomString("ui_form_talk_leave"))); err != nil {
		return nil, err
	}
	if err := add(entergen.CustomMessageFrame(7, entergen.CustomObject(objectID, ownerID))); err != nil {
		return nil, err
	}
	return frames, nil
}

// npcTaskMarkSplit 任务域 mark 拆解：flag∈[100,800) 且 TID>0 才是任务行。
func npcTaskMarkSplit(mark int32) (flag int32, taskID int, ok bool) {
	if mark < npcTaskFlagAccept*npcTaskFuncBase || mark >= 800*npcTaskFuncBase {
		return 0, 0, false
	}
	taskID = int(mark % npcTaskFuncBase)
	flag = mark / npcTaskFuncBase
	if taskID <= 0 {
		return 0, 0, false
	}
	return flag, taskID, true
}

// handleNPCTaskMark op0x07 任务域 mark 受理（状态同步帧序见 taskrecord.go）。
func (s *Server) handleNPCTaskMark(pl *Player, menu *npcActiveMenu, mark int32) bool {
	flag, taskID, ok := npcTaskMarkSplit(mark)
	if !ok {
		return false
	}
	catalog := s.npcTaskCatalogOf()
	spawn, _ := s.npcTaskMenuSpawn(pl, menu)
	npc := entergen.Identity{ObjectID: spawn.ObjectID, OwnerID: spawn.OwnerID}
	switch {
	case flag == npcTaskFlagAccept:
		if menu != nil && menu.pendingTask == taskID && !pl.taskAcceptedContains(uint32(taskID)) {
			menu.pendingTask = 0 // 二次确认：真接取
			s.npcTaskLocked(pl, func() {
				if s.npcTaskAccept(pl, taskID, "对话接取") {
					s.refreshNPCTalkMenu(pl, menu)
				}
			})
			return true
		}
		if pl.taskAcceptedContains(uint32(taskID)) {
			return true // 已接：幂等吞掉
		}
		// 首次 100：推接取窗（客户端本地渲染），窗内「接受任务」行同
		// funcid 二次到达才是真接取。
		if menu != nil {
			menu.pendingTask = taskID
		}
		s.sendTaskAcceptForm(pl, menu, taskID)
	case flag == npcTaskFlagSubmit:
		// 只回提交窗参数：客户端弹提交窗，玩家确认后回 201。
		s.npcTaskLocked(pl, func() {
			task, track, _, cur := npcTaskCurrent(catalog, pl, uint32(taskID))
			if !cur || !entergen.TaskSubmittable(track, task.Steps) || task.AutoSubmit || spawn.ConfigID == "" {
				return
			}
			s.sendTaskSubmitForm(pl, npc, task)
		})
	case flag == npcTaskFlagSubmitChoice:
		s.completeNPCTask(pl, taskID, npcTaskCompleteTail{npc: npc, closeTalk: true})
		s.clearNPCMenu(pl)
	case flag == npcTaskFlagDoing:
		s.npcTaskConfirm(pl, menu, taskID, 1)
	case flag > npcTaskFlagStoryContinueBase && flag <= npcTaskFlagStoryContinueBase+npcTaskStoryMaxSegments:
		// 窗 N 的续行（N=flag-500）→ 发窗 N+1；超出段数=剧情步完成。
		s.npcTaskConfirm(pl, menu, taskID, int(flag-npcTaskFlagStoryContinueBase)+1)
	case flag > npcTaskFlagStoryLeaveBase && flag <= npcTaskFlagStoryLeaveBase+npcTaskStoryMaxSegments:
		s.trace("npc task: 剧情中断 tid=%d flag=%d client=%d", taskID, flag, pl.Conn1)
		s.closeNPCTalk(pl)
		s.clearNPCMenu(pl)
	default:
		s.trace("npc task: mark flag=%d tid=%d client=%d（未受理域）", flag, taskID, pl.Conn1)
	}
	return true
}

// npcTaskConfirm 500 域点击：任务当前步是本 NPC 的交互步=确认完成该步；
// 是剧情步=进剧情窗第 segment 段（续行点击 segment=N+1）。
func (s *Server) npcTaskConfirm(pl *Player, menu *npcActiveMenu, taskID, segment int) {
	catalog := s.npcTaskCatalogOf()
	spawn, ok := s.npcTaskMenuSpawn(pl, menu)
	if !ok {
		return
	}
	npc := entergen.Identity{ObjectID: spawn.ObjectID, OwnerID: spawn.OwnerID}
	if task, _, _ := s.npcTaskStepWindow(catalog, pl, spawn.ConfigID); task.ID == taskID {
		s.npcTaskLocked(pl, func() {
			task, track, step, cur := npcTaskCurrent(catalog, pl, uint32(taskID))
			if !cur || step.Subtype != taskdata.StepInteract || !npcTaskStepAtNpc(step, spawn.ConfigID) || step.Scene == "" {
				return
			}
			s.npcTaskStepDone(pl, uint32(taskID), npcTaskDialogTail(npc, task, track.Step-1))
		})
		s.clearNPCMenu(pl)
		return
	}
	s.sendStorySegment(pl, menu, taskID, segment)
}

// sendTaskSubmitForm 提交窗参数 m209 [2, NPC, 201×10⁶+T, T, "complete_T",
// "submit_T", 0, "", Line]（点「完成任务」行后下发；客户端弹提交窗并回
// mark 201）。
func (s *Server) sendTaskSubmitForm(pl *Player, npc entergen.Identity, task taskdata.Task) {
	frame, err := entergen.TaskSubmitFormFrame(npc, task)
	if err != nil {
		s.trace("npc task: 提交窗参数编码失败 %d: %v", task.ID, err)
		return
	}
	s.sendToClient(pl, frame)
}

// npcTaskMenuSpawn 菜单簿记对应的场景摆放（对象已不在场景则 ok=false）。
func (s *Server) npcTaskMenuSpawn(pl *Player, menu *npcActiveMenu) (npcfeed.Spawn, bool) {
	if menu == nil {
		return npcfeed.Spawn{}, false
	}
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	spawn, ok := s.findNPCSpawn(scene, menu.objectID)
	if !ok || spawn.OwnerID != menu.ownerID {
		return npcfeed.Spawn{}, false
	}
	return spawn, true
}

// sendTaskAcceptForm 接取窗推送（task_msg sub1——show_accept_form 15 参
// 族，客户端从参数渲染标题/正文/菜单行「接受任务」/离开；奖励三槽无数据
// 发空串，decMoney/冒险/挑战等门参数零值）。目录里没有该任务则不推窗
// （没有文案无处渲染）。
func (s *Server) sendTaskAcceptForm(pl *Player, menu *npcActiveMenu, taskID int) {
	task, ok := s.npcTaskCatalogOf().TaskByID(taskID)
	if !ok {
		s.trace("npc task: 接取窗无任务文案 %d，不推窗", taskID)
		return
	}
	objectID, ownerID := uint32(0), uint32(0)
	if menu != nil {
		objectID, ownerID = menu.objectID, menu.ownerID
	}
	frame, err := entergen.KeyedMessageFrame("task_msg",
		entergen.CustomInt(1), // submsg=1 on_show_accept_form
		entergen.CustomObject(objectID, ownerID),
		entergen.CustomInt(npcTaskFlagAccept*npcTaskFuncBase+int32(taskID)), // funcid
		entergen.CustomInt(int32(taskID)),                                   // task_id
		entergen.CustomString("context_"+strconv.Itoa(taskID)),              // titleid → get_title_task_context
		entergen.CustomString("task_menu_accept_"+strconv.Itoa(taskID)),     // menuid
		entergen.CustomString(""),                                           // prize_base
		entergen.CustomString(""),                                           // prize_ex
		entergen.CustomString(""),                                           // prize_over
		entergen.CustomInt(0),                                               // decMoney
		entergen.CustomInt(0),                                               // adventure_plot
		entergen.CustomInt(0),                                               // lock_value
		entergen.CustomString(""),                                           // strLimit
		entergen.CustomString(task.Title),                                   // title（富文本原文）
		entergen.CustomInt(0),                                               // line
		entergen.CustomInt(0),                                               // tiaozhan
	)
	if err != nil {
		s.trace("npc task: 接取窗编码失败 %d: %v", taskID, err)
		return
	}
	s.sendToClient(pl, frame)
	s.trace("npc task: 接取窗 %d（%s）client=%d", taskID, taskdata.StripRichText(task.Title), pl.Conn1)
}

// npcTaskStorySegments 剧情步在本 NPC 的剧情段（行序即段序）：只取剧情组
// 与步骤号相符的绑定行。
func npcTaskStorySegments(catalog *taskdata.Catalog, configID string, taskID int, step taskdata.Step) []taskdata.Binding {
	group := strconv.Itoa(step.Index)
	var segments []taskdata.Binding
	for _, binding := range catalog.StoryBindingsFor(configID, taskID) {
		if binding.StoryID == group {
			segments = append(segments, binding)
		}
	}
	return segments
}

// sendStorySegment 剧情对话窗第 segment 段（引导抓包帧族定版）：m4 / m5
// "story_<sid>_N" / m6 [(500+N)×10⁶+TID, "storymenu_<sid>_N"] / m6
// [(600+N)×10⁶+TID, 离行文案] / m7。segment 超出段数＝末窗续行点击：剧情步
// 记一次进度（末步且自动提交即进完成簇）。
func (s *Server) sendStorySegment(pl *Player, menu *npcActiveMenu, taskID, segment int) {
	catalog := s.npcTaskCatalogOf()
	spawn, ok := s.npcTaskMenuSpawn(pl, menu)
	if catalog == nil || !ok {
		return
	}
	_, _, step, cur := npcTaskCurrent(catalog, pl, uint32(taskID))
	if !cur || step.Subtype != taskdata.StepStory || !npcTaskStepAtNpc(step, spawn.ConfigID) {
		s.trace("npc task: 剧情点击但当前步不是本 NPC 的剧情步 tid=%d client=%d", taskID, pl.Conn1)
		return
	}
	npc := entergen.Identity{ObjectID: spawn.ObjectID, OwnerID: spawn.OwnerID}
	segments := npcTaskStorySegments(catalog, spawn.ConfigID, taskID, step)
	if segment < 1 {
		segment = 1
	}
	if segment > len(segments) {
		// 末窗续行：剧情步完成（锁内复核当前步，防重复点击重复记账）。
		s.npcTaskLocked(pl, func() {
			task, track, step, ok := npcTaskCurrent(catalog, pl, uint32(taskID))
			if !ok || step.Subtype != taskdata.StepStory {
				return
			}
			s.npcTaskStepDone(pl, uint32(taskID), npcTaskDialogTail(npc, task, track.Step-1))
		})
		s.clearNPCMenu(pl)
		return
	}
	seg := segments[segment-1]
	frames := make([][]byte, 0, 5)
	add := func(frame []byte, err error) bool {
		if err != nil {
			return false
		}
		frames = append(frames, frame)
		return true
	}
	cont := (npcTaskFlagStoryContinueBase + int32(segment)) * npcTaskFuncBase
	leave := (npcTaskFlagStoryLeaveBase + int32(segment)) * npcTaskFuncBase
	leaveText := seg.LeaveTextID
	if leaveText == "" {
		leaveText = npcStoryLeaveTextDefault
	}
	if !add(entergen.CustomMessageFrame(4)) ||
		!add(entergen.CustomMessageFrame(5, entergen.CustomString(seg.ContextID))) ||
		!add(entergen.CustomMessageFrame(6, entergen.CustomInt(cont+int32(taskID)), entergen.CustomString(seg.TitleID))) ||
		!add(entergen.CustomMessageFrame(6, entergen.CustomInt(leave+int32(taskID)), entergen.CustomString(leaveText))) ||
		!add(entergen.CustomMessageFrame(7, entergen.CustomObject(spawn.ObjectID, spawn.OwnerID))) {
		return
	}
	for _, frame := range frames {
		s.sendToClient(pl, frame)
	}
	s.trace("npc task: 剧情段 tid=%d seg=%d/%d（%s）client=%d", taskID, segment, len(segments), seg.ContextID, pl.Conn1)
}

// refreshNPCTalkMenu 重发主窗帧族（接取后任务行即时翻转；选中帧无需
// 重发——选中态未变）。
func (s *Server) refreshNPCTalkMenu(pl *Player, menu *npcActiveMenu) {
	if menu == nil {
		return
	}
	spawn, ok := s.npcTaskMenuSpawn(pl, menu)
	if !ok {
		return
	}
	s.sendNPCTalkMenu(pl, spawn, npcServicesOf(spawn.Template))
}

// taskAcceptedContains 已接判定。
func (p *Player) taskAcceptedContains(id uint32) bool {
	for _, v := range p.roleState.TaskAccepted {
		if v == id {
			return true
		}
	}
	return false
}

// taskCompletedContains 已交判定。
func (p *Player) taskCompletedContains(id uint32) bool {
	for _, v := range p.roleState.TaskCompleted {
		if v == id {
			return true
		}
	}
	return false
}

// removeAcceptedTask 移除已接任务（保持余序；值语义拷贝安全——roleState
// 为 Player 值字段，切片重建不共享底层数组增长位）。
func (p *Player) removeAcceptedTask(id uint32) []uint32 {
	kept := make([]uint32, 0, len(p.roleState.TaskAccepted))
	for _, v := range p.roleState.TaskAccepted {
		if v != id {
			kept = append(kept, v)
		}
	}
	return kept
}

// npcTaskCatalogOf 数据根解析 + 目录装载薄封装（来源 / 缺席各 trace 一次：
// 任务标记等处每次状态变更都会取目录）。
func (s *Server) npcTaskCatalogOf() *taskdata.Catalog {
	root := npcfeed.ResolveRoot()
	catalog := taskCatalogForRoot(root)
	if catalog == nil {
		npcTaskMissingOnce.Do(func() {
			s.trace("npc task: 任务目录缺席（数据根 %s，来源 %s），任务行不下发", root, taskdata.QuestSourceOf(root))
		})
		return catalog
	}
	npcTaskSourceOnce.Do(func() {
		s.trace("npc task: 任务目录 %d 任务 / %d 绑定，来源 %s（share=直读 share/task/task + 文案档，xml=构建产物）",
			catalog.Count(), catalog.BindingCount(), taskdata.QuestSourceOf(root))
	})
	return catalog
}

// npcTaskSourceOnce / npcTaskMissingOnce 任务目录来源与缺席各 trace 一次。
var (
	npcTaskSourceOnce  sync.Once
	npcTaskMissingOnce sync.Once
)
