package fssoloworld

// taskrecord.go 任务记录同步层：玩家任务进度（roleState.TaskAccepted /
// TaskTracks / TaskCompleted）↔ 客户端三张任务记录表的行级同步。帧序按
// 官服引导抓包（初始账号登陆-完整引导-女.pcapng，8032）逐簇定版；行编码
// 见 internal/entergen/taskrecords.go。
//
// 客户端行布局（服务端按同一规则推算行号；登录簇 entergen.taskRecordRows
// 同源）：
//
//	Task_Accepted  行号＝roleState.TaskAccepted 下标
//	Task_Record    每个有进度行的已接任务占 len(Steps) 行，按已接序拼接
//
// 事件簇（帧序即下发序）：
//
//	接取 T       0x11 Record row=start+i × n → 0x11 Accepted row=a（已接
//	            表已有该行——建号种子的功能门行——改为逐格写 track/time）
//	            → 0x13 step=1 →（首步杀怪）0x13 Record 首行 flags=1 →
//	            NPC 标记 → m209 [9,T] → m24 "task_accept" → m1 "9628"
//	一步完成    0x13 curnum →（非末步）0x13 step=k+1 ／（末步且非自动
//	            提交）0x13 status=2 → 0x13 flags=32+h → 0x13 flags=48+h →
//	            （下一步杀怪）0x13 下一行 flags=1 → NPC 标记 → m24 "task_ok"
//	            → m18 [NPC,""] →（末步非自动提交）m209 提交窗参数 → m8
//	完成 T       完成位图（缺行先 0x11 加行，再 0x13 写格）→ 0x12 Record
//	            row=start × n → 0x12 Accepted row=a → m209 [8,T] → m1 "9631"
//	            → NPC 标记 → 后继任务接取簇 → m24 "task_complete" → m18 → m8
//	追踪列表    客户端 m327 [6,"id,…"] → 列表内的已接行 0x13 track=0 →
//	            track=1；不在列表里的追踪行 0x13 track=0
//
// h＝该步是杀怪步（flags 的两次写带位 1：33、49）。末步且自动提交的任务
// 不发 step/status/flags 写与 task_ok：curnum 之后直接进完成簇；完成簇的
// 收尾按场景取：对话窗内只发 m8，提交确认（201）发 m18 → m8，杀怪不发。
//
// 官服新号登录时三张任务表为空（登录簇不发任务行）；每环接取各发一条
// 已接行（小行明文 0x11、稍大走 LZ 0x29 单行批——两者等价，这里恒用
// 明文行）。奖励提示（m1 1303/9693/8055/9645…）与奖励发放未实现，不发。
//
// 占位推进：服务端没有 NPC 战斗/伤害、打坐/经脉、场景事件这些通路，
// 杀怪步、特殊步、无对话的事件交互步无从达成——NINEYIN_TASK_STUB =
// guide（缺省，只占位新手引导链任务）｜all｜off。

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// NPC 任务标记位（m209 [15, obj, state]；官服引导抓包各簇取值归纳）。
const (
	npcTaskMarkDoing  uint32 = 1 // 当前步在此 NPC
	npcTaskMarkSubmit uint32 = 2 // 各步已完成待提交／任务待回此 NPC（当前步在别处）
	npcTaskMarkAccept uint32 = 4 // 此 NPC 有可接取的任务
	npcTaskMarkStory  uint32 = 8 // 当前步是此 NPC 的剧情步（与 1 同现＝9）
)

var (
	// npcTaskStubInterval 占位推进的间隔（杀怪逐只、事件步延迟；官服击杀
	// 间隔数秒到数十秒，这里取能看出追踪栏变化的短间隔）。
	npcTaskStubInterval = 1500 * time.Millisecond
	// npcTaskMovieDelay 接取引导影像的下发延迟（官服约半秒后、m8 之后）。
	npcTaskMovieDelay = 500 * time.Millisecond
	// npcTaskGuideRootDelay 引导链根进场自动接取的延迟（官服 1340 在进场
	// 约 15 秒后自动接取；本服 NPC 物化后起算）。
	npcTaskGuideRootDelay = 5 * time.Second
)

// ---- 行号推算（须与 entergen 登录簇的行推一致） ----

// npcTaskRecordRows 任务在客户端目标表里的起始行与行数；没有进度行的
// 任务不占行（ok=false）。
func npcTaskRecordRows(pl *Player, catalog *taskdata.Catalog, taskID uint32) (start uint16, count int, ok bool) {
	if catalog == nil {
		return 0, 0, false
	}
	for _, id := range pl.roleState.TaskAccepted {
		if pl.roleState.TaskTrackOf(id) == nil {
			continue
		}
		n := len(catalog.StepsForTask(int(id)))
		if id == taskID {
			return start, n, true
		}
		start += uint16(n)
	}
	return 0, 0, false
}

// npcTaskAcceptedRow 任务在客户端已接表里的行号（＝已接序下标）。
func npcTaskAcceptedRow(pl *Player, taskID uint32) (uint16, bool) {
	for i, id := range pl.roleState.TaskAccepted {
		if id == taskID {
			return uint16(i), true
		}
	}
	return 0, false
}

// npcTaskCurrent 任务的当前步：在目录里、有进度行、步序在范围内。
func npcTaskCurrent(catalog *taskdata.Catalog, pl *Player, taskID uint32) (task taskdata.Task, track *rolestate.TaskTrack, step taskdata.Step, ok bool) {
	if catalog == nil {
		return taskdata.Task{}, nil, taskdata.Step{}, false
	}
	task, inCatalog := catalog.TaskByID(int(taskID))
	track = pl.roleState.TaskTrackOf(taskID)
	if !inCatalog || track == nil || track.Step < 1 || track.Step > len(task.Steps) {
		return task, track, taskdata.Step{}, false
	}
	return task, track, task.Steps[track.Step-1], true
}

// npcTaskStepAtNpc 步骤的目标是否就是这个 NPC（NPC 目标类步骤按配置 ID
// 大小写不敏感比较）。
func npcTaskStepAtNpc(step taskdata.Step, configID string) bool {
	switch step.Subtype {
	case taskdata.StepInteract, taskdata.StepStory, taskdata.StepQuestion, taskdata.StepConvoy:
		return step.Key != "" && strings.EqualFold(strings.TrimSpace(step.Key), strings.TrimSpace(configID))
	}
	return false
}

// npcTaskStepMax 目标次数（缺省 1）。
func npcTaskStepMax(step taskdata.Step) int {
	if step.MaxNum > 0 {
		return step.MaxNum
	}
	return 1
}

// npcTaskSpawnOf 玩家当前场景里指定配置 ID 的首个摆放（没有返回 ok=false）。
func (s *Server) npcTaskSpawnOf(pl *Player, configID string) (npcfeed.Spawn, bool) {
	configID = strings.TrimSpace(configID)
	if configID == "" {
		return npcfeed.Spawn{}, false
	}
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	if scene == "" {
		return npcfeed.Spawn{}, false
	}
	for _, spawn := range s.sceneNPCCatalog(scene).spawns {
		if strings.EqualFold(spawn.ConfigID, configID) {
			return spawn, true
		}
	}
	return npcfeed.Spawn{}, false
}

// npcTaskSpawnIdentity 摆放对象的 wire 身份（没有摆放＝零值）。
func npcTaskSpawnIdentity(spawn npcfeed.Spawn, ok bool) entergen.Identity {
	if !ok {
		return entergen.Identity{}
	}
	return entergen.Identity{ObjectID: spawn.ObjectID, OwnerID: spawn.OwnerID}
}

// ---- 帧汇集 ----

// taskFrames 一次同步操作要下发的帧（按序）。编码失败只留痕：编码器错误
// 是程序缺陷，不在运行期分支。
type taskFrames struct {
	list [][]byte
	fail func(error)
}

func (t *taskFrames) add(frame []byte, err error) {
	if err != nil {
		if t.fail != nil {
			t.fail(err)
		}
		return
	}
	t.list = append(t.list, frame)
}

// extend 追加已编码好的帧（无错误形态的编码器，如删行帧）。
func (t *taskFrames) extend(frames [][]byte) {
	t.list = append(t.list, frames...)
}

func (t *taskFrames) addAll(frames [][]byte, err error) {
	if err != nil {
		t.add(nil, err)
		return
	}
	t.list = append(t.list, frames...)
}

func (s *Server) taskFrameSink(pl *Player) *taskFrames {
	return &taskFrames{fail: func(err error) {
		s.trace("npc task: 帧编码失败 client=%d: %v", pl.Conn1, err)
	}}
}

// sendTaskFrames 按序下发帧簇。
func (s *Server) sendTaskFrames(pl *Player, out *taskFrames) {
	for _, frame := range out.list {
		s.sendToClient(pl, frame)
	}
}

// npcTaskLocked 串行化任务状态变更（消息线程与占位/影像定时器都会碰任务
// 进度）；下面各 *Frames 函数一律要求调用方持这把锁。
func (s *Server) npcTaskLocked(pl *Player, fn func()) {
	pl.taskMu.Lock()
	defer pl.taskMu.Unlock()
	fn()
}

// ---- NPC 任务标记 ----

// npcTaskMarkOf NPC 的任务标记（位义见 npcTaskMark* 常量）。
func npcTaskMarkOf(catalog *taskdata.Catalog, pl *Player, configID string) uint32 {
	configID = strings.TrimSpace(configID)
	if catalog == nil || configID == "" {
		return 0
	}
	same := func(npc string) bool {
		return npc != "" && strings.EqualFold(strings.TrimSpace(npc), configID)
	}
	var state uint32
	for _, id := range pl.roleState.TaskAccepted {
		track := pl.roleState.TaskTrackOf(id)
		if track == nil {
			continue
		}
		task, ok := catalog.TaskByID(int(id))
		if !ok {
			continue
		}
		home := same(task.SubmitNpc)
		if entergen.TaskSubmittable(track, task.Steps) {
			if home && !task.AutoSubmit {
				state |= npcTaskMarkSubmit
			}
			continue
		}
		if track.Step < 1 || track.Step > len(task.Steps) {
			continue
		}
		step := task.Steps[track.Step-1]
		switch at := npcTaskStepAtNpc(step, configID); {
		case at && step.Subtype == taskdata.StepStory:
			state |= npcTaskMarkDoing | npcTaskMarkStory
		case at:
			state |= npcTaskMarkDoing
		case home:
			state |= npcTaskMarkDoing | npcTaskMarkSubmit
		}
	}
	for _, task := range catalog.TasksForNPC(configID) {
		tid := uint32(task.ID)
		if pl.taskAcceptedContains(tid) || pl.taskCompletedContains(tid) {
			continue
		}
		if task.ChainRoot || task.AcceptMenuKey != "" {
			state |= npcTaskMarkAccept
		}
	}
	return state
}

// npcTaskMarkFrames 玩家可见 NPC 的任务标记：只发与上次下发不同的 NPC。
func (s *Server) npcTaskMarkFrames(pl *Player, out *taskFrames) {
	catalog := s.npcTaskCatalogOf()
	pl.npcMu.Lock()
	scene, visible := pl.npcScene, pl.npcVisible
	pl.npcMu.Unlock()
	if catalog == nil || visible == nil {
		return
	}
	type update struct {
		objectID, ownerID, state uint32
	}
	var updates []update
	for _, spawn := range s.sceneNPCCatalog(scene).spawns {
		if !visible[spawn.ObjectID] {
			continue
		}
		state := npcTaskMarkOf(catalog, pl, spawn.ConfigID)
		pl.npcMu.Lock()
		last, known := pl.taskMarks[spawn.ObjectID]
		if known && last == state {
			pl.npcMu.Unlock()
			continue
		}
		if pl.taskMarks == nil {
			pl.taskMarks = make(map[uint32]uint32)
		}
		pl.taskMarks[spawn.ObjectID] = state
		pl.npcMu.Unlock()
		updates = append(updates, update{spawn.ObjectID, spawn.OwnerID, state})
	}
	for _, u := range updates {
		out.add(entergen.TaskMarkFrame(u.objectID, u.ownerID, u.state))
	}
}

// pushNPCTaskMarks 任务状态变更之外的时点（NPC 新进视口）补发标记。
func (s *Server) pushNPCTaskMarks(pl *Player) {
	if !pl.haveRoleState {
		return
	}
	s.npcTaskLocked(pl, func() {
		out := s.taskFrameSink(pl)
		s.npcTaskMarkFrames(pl, out)
		s.sendTaskFrames(pl, out)
	})
}

// ---- 接取 ----

// npcTaskAcceptFrames 接取簇（改状态、追加帧）。任务已在进度中返回
// false。已在已接表但没有进度行（建号种子的功能门行，如 1344）不重复
// 加行，改逐格写追踪标志/接取时刻。
func (s *Server) npcTaskAcceptFrames(pl *Player, out *taskFrames, taskID int) bool {
	catalog := s.npcTaskCatalogOf()
	task, inCatalog := catalog.TaskByID(taskID)
	tid := uint32(taskID)
	if !inCatalog || taskID <= 0 {
		return false
	}
	if pl.roleState.TaskTrackOf(tid) != nil {
		return false
	}
	present := pl.taskAcceptedContains(tid)
	track := pl.roleState.EnsureTaskTrack(tid, time.Now().Unix())
	track.Tracked, track.Step, track.Cur = true, 1, nil
	if !present {
		pl.roleState.TaskAccepted = append(pl.roleState.TaskAccepted, tid)
	}

	id := pl.entergenIdentity()
	start, _, _ := npcTaskRecordRows(pl, catalog, tid)
	stamp := time.Now().UnixNano()
	// 新接任务的目标行是零进度行（官服加行 flags/curnum 皆 0，「杀怪步成为
	// 当前步」的 flags=1 随后由逐格写置位）；重登簇才按进度回填。
	for i, step := range task.Steps {
		out.add(entergen.TaskRecordRowFrame(id.ObjectID, id.OwnerID, start+uint16(i), tid, step,
			entergen.TaskRecordView{}, stamp+int64(i)*1024))
	}
	rowA, _ := npcTaskAcceptedRow(pl, tid)
	if present {
		out.add(entergen.TaskAcceptedGridFrame(id.ObjectID, id.OwnerID, rowA, entergen.GridInt(entergen.TaskColTrack, 1)))
		out.add(entergen.TaskAcceptedGridFrame(id.ObjectID, id.OwnerID, rowA, entergen.GridInt64(entergen.TaskColTime, track.Time)))
	} else {
		out.add(entergen.TaskAcceptedRowFrame(id.ObjectID, id.OwnerID, rowA, task,
			entergen.TaskAcceptedView{Status: entergen.TaskStatusDoing, Tracked: true, Step: 0, Accepted: track.Time}))
	}
	if len(task.Steps) > 0 {
		out.add(entergen.TaskAcceptedGridFrame(id.ObjectID, id.OwnerID, rowA, entergen.GridInt(entergen.TaskColStep, 1)))
		if task.Steps[0].Subtype == taskdata.StepHunter {
			out.add(entergen.TaskRecordGridFrame(id.ObjectID, id.OwnerID, start,
				entergen.GridInt(entergen.RecordColFlags, entergen.TaskRecordFlagHunterActive)))
		}
	} else {
		// 没有步骤数据的任务没有可达成的目标：接取即可提交。
		out.add(entergen.TaskAcceptedGridFrame(id.ObjectID, id.OwnerID, rowA, entergen.GridInt(entergen.TaskColStatus, entergen.TaskStatusCanSubmit)))
	}
	s.npcTaskMarkFrames(pl, out)
	out.addAll(entergen.TaskAcceptNoticeFrames(id, taskID))
	return true
}

// acceptNPCTask 接取任务并下发接取簇（链根进场景/首谈、100 行二次确认；
// AutoAcceptID 链推进在完成簇内走 npcTaskAcceptFrames）；返回是否新接。
func (s *Server) acceptNPCTask(pl *Player, taskID int, how string) bool {
	if !pl.haveRoleState {
		return false
	}
	accepted := false
	s.npcTaskLocked(pl, func() { accepted = s.npcTaskAccept(pl, taskID, how) })
	return accepted
}

// npcTaskAccept 接取并下发接取簇，接自动步（占位）与引导影像。调用方持
// taskMu。
func (s *Server) npcTaskAccept(pl *Player, taskID int, how string) bool {
	out := s.taskFrameSink(pl)
	if !s.npcTaskAcceptFrames(pl, out, taskID) {
		return false
	}
	s.sendTaskFrames(pl, out)
	task, _ := s.npcTaskCatalogOf().TaskByID(taskID)
	s.trace("npc task: 接取 %d（%s，%d 步）client=%d（已接 %d）", taskID, how, len(task.Steps), pl.Conn1, len(pl.roleState.TaskAccepted))
	s.sendRoleSave(pl)
	s.npcTaskAutoSteps(pl, uint32(taskID))
	// 接取引导影像只在官服有样本的两种触发后播（进场自动接取、完成簇内的
	// 链式接取）；对话中接取不播——半秒后的 m8 会把刚开的对话窗关掉。
	if how != "链根首谈" && how != "对话接取" {
		s.npcTaskScheduleMovie(pl, task)
	}
	return true
}

// npcTaskScheduleMovie 接取引导影像（主表 AcceptMovie）：官服在接取约半秒
// 后先 m8 再 m104。到点时任务已不在进度中／玩家已离线则不发。
func (s *Server) npcTaskScheduleMovie(pl *Player, task taskdata.Task) {
	if task.AcceptMovie <= 0 {
		return
	}
	movie, taskID := task.AcceptMovie, uint32(task.ID)
	time.AfterFunc(npcTaskMovieDelay, func() {
		if pl.State != PlOnline {
			return
		}
		s.npcTaskLocked(pl, func() {
			if pl.State != PlOnline || pl.roleState.TaskTrackOf(taskID) == nil {
				return
			}
			if closeTalk, err := entergen.TalkCloseFrame(); err == nil {
				s.sendToClient(pl, closeTalk)
			}
			if frame, err := entergen.TaskAcceptMovieFrame(movie); err == nil {
				s.sendToClient(pl, frame)
			}
		})
	})
}

// ---- 步骤推进 ----

// npcTaskStepTail 一步完成后 task_ok 之后的收尾帧。
type npcTaskStepTail struct {
	npc       entergen.Identity // m18 [NPC,""] 的对象（零值不发 m18）
	submit    bool              // 末步且非自动提交：m18 后下发提交窗参数
	closeTalk bool              // 结尾 m8
}

// npcTaskDialogTail 对话窗内完成一步的收尾（剧情窗/交互确认窗）：末步且
// 非自动提交时紧接提交窗参数。
func npcTaskDialogTail(npc entergen.Identity, task taskdata.Task, stepIndex int) npcTaskStepTail {
	return npcTaskStepTail{npc: npc, submit: stepIndex == len(task.Steps)-1 && !task.AutoSubmit, closeTalk: true}
}

// npcTaskAdvanceStep 当前步记一次进度。进度未满只发 curnum（杀怪逐只
// 计数）；记满按官服帧序翻步／置可提交并发 task_ok 收尾；末步且自动
// 提交则 curnum 之后直接进完成簇。返回是否记了进度。
func (s *Server) npcTaskAdvanceStep(pl *Player, taskID uint32, tail npcTaskStepTail) bool {
	if !pl.haveRoleState {
		return false
	}
	catalog := s.npcTaskCatalogOf()
	task, track, step, ok := npcTaskCurrent(catalog, pl, taskID)
	if !ok || npcTaskStepDone(track, step) {
		return false // 当前步已记满（末步翻可提交后步序仍指向它）：重复点击不重复记账
	}
	k := track.Step - 1
	for len(track.Cur) <= k {
		track.Cur = append(track.Cur, 0)
	}
	track.Cur[k]++
	start, _, _ := npcTaskRecordRows(pl, catalog, taskID)
	rowA, _ := npcTaskAcceptedRow(pl, taskID)
	id := pl.entergenIdentity()
	out := s.taskFrameSink(pl)
	out.add(entergen.TaskRecordGridFrame(id.ObjectID, id.OwnerID, start+uint16(k),
		entergen.GridInt(entergen.RecordColCurNum, uint32(track.Cur[k]))))
	if track.Cur[k] < npcTaskStepMax(step) {
		s.sendTaskFrames(pl, out)
		return true
	}
	if k == len(task.Steps)-1 && task.AutoSubmit {
		s.sendTaskFrames(pl, out)
		return s.npcTaskCompleteFrames(pl, taskID, npcTaskCompleteTail{closeTalk: tail.closeTalk})
	}
	last := k == len(task.Steps)-1
	hunter := uint32(0)
	if step.Subtype == taskdata.StepHunter {
		hunter = entergen.TaskRecordFlagHunterActive
	}
	if !last {
		track.Step = k + 2
		out.add(entergen.TaskAcceptedGridFrame(id.ObjectID, id.OwnerID, rowA, entergen.GridInt(entergen.TaskColStep, uint32(track.Step))))
	} else {
		out.add(entergen.TaskAcceptedGridFrame(id.ObjectID, id.OwnerID, rowA, entergen.GridInt(entergen.TaskColStatus, entergen.TaskStatusCanSubmit)))
	}
	out.add(entergen.TaskRecordGridFrame(id.ObjectID, id.OwnerID, start+uint16(k),
		entergen.GridInt(entergen.RecordColFlags, entergen.TaskRecordFlagDone|hunter)))
	out.add(entergen.TaskRecordGridFrame(id.ObjectID, id.OwnerID, start+uint16(k),
		entergen.GridInt(entergen.RecordColFlags, entergen.TaskRecordFlagSettled|hunter)))
	if !last && task.Steps[k+1].Subtype == taskdata.StepHunter {
		out.add(entergen.TaskRecordGridFrame(id.ObjectID, id.OwnerID, start+uint16(k+1),
			entergen.GridInt(entergen.RecordColFlags, entergen.TaskRecordFlagHunterActive)))
	}
	s.npcTaskMarkFrames(pl, out)
	out.add(entergen.TaskStepOkActionFrame(id))
	if tail.npc.ObjectID != 0 {
		out.add(entergen.TaskTalkEndObjectFrame(tail.npc))
	}
	if tail.submit {
		out.add(entergen.TaskSubmitFormFrame(tail.npc, task))
	}
	if tail.closeTalk {
		out.add(entergen.TalkCloseFrame())
	}
	s.sendTaskFrames(pl, out)
	s.trace("npc task: 步进 %d step=%d/%d（%s）client=%d", taskID, k+1, len(task.Steps), step.TrackInfo, pl.Conn1)
	s.sendRoleSave(pl)
	return true
}

// npcTaskStepDone 记一次当前步进度并接自动步（对话/剧情/占位各入口共用；
// 调用方持 taskMu）。返回是否记了进度。
func (s *Server) npcTaskStepDone(pl *Player, taskID uint32, tail npcTaskStepTail) bool {
	if !s.npcTaskAdvanceStep(pl, taskID, tail) {
		return false
	}
	s.npcTaskAutoSteps(pl, taskID)
	return true
}

// ---- 完成 ----

// npcTaskCompleteTail 完成簇的收尾。
type npcTaskCompleteTail struct {
	npc       entergen.Identity // 提交确认：m24 task_complete 后 m18 [NPC,""]
	closeTalk bool              // 对话中（剧情窗/交互确认窗/提交窗）：m8
}

// completeNPCTask 提交受理并下发完成簇（201 提交确认；末步自动提交走
// npcTaskAdvanceStep 内部）。任务不在已接表或尚不可提交（目标未完成时
// 客户端不该回 201）返回 false。
func (s *Server) completeNPCTask(pl *Player, taskID int, tail npcTaskCompleteTail) bool {
	if !pl.haveRoleState {
		return false
	}
	done := false
	s.npcTaskLocked(pl, func() {
		catalog := s.npcTaskCatalogOf()
		task, track, _, _ := npcTaskCurrent(catalog, pl, uint32(taskID))
		_, accepted := npcTaskAcceptedRow(pl, uint32(taskID))
		if !accepted || !entergen.TaskSubmittable(track, task.Steps) {
			s.trace("npc task: 提交 %d 被拒（未接取或目标未完成）client=%d", taskID, pl.Conn1)
			return
		}
		done = s.npcTaskCompleteFrames(pl, uint32(taskID), tail)
	})
	return done
}

// npcTaskCompleteFrames 完成簇（改状态、下发、存档）。任务不在已接表
// 返回 false。
func (s *Server) npcTaskCompleteFrames(pl *Player, taskID uint32, tail npcTaskCompleteTail) bool {
	if !pl.haveRoleState {
		return false
	}
	catalog := s.npcTaskCatalogOf()
	rowA, accepted := npcTaskAcceptedRow(pl, taskID)
	if !accepted {
		return false
	}
	task, _ := catalog.TaskByID(int(taskID))
	start, count, _ := npcTaskRecordRows(pl, catalog, taskID)
	id := pl.entergenIdentity()
	completedBefore := append([]uint32(nil), pl.roleState.TaskCompleted...)

	pl.roleState.TaskAccepted = pl.removeAcceptedTask(taskID)
	pl.roleState.RemoveTaskTrack(taskID)
	if !pl.taskCompletedContains(taskID) {
		pl.roleState.TaskCompleted = append(pl.roleState.TaskCompleted, taskID)
	}

	out := s.taskFrameSink(pl)
	out.addAll(entergen.TaskCompletedSetFrames(id.ObjectID, id.OwnerID, taskID, completedBefore))
	out.extend(entergen.TaskRecordDeleteFrames(id.ObjectID, id.OwnerID, start, count))
	out.add(entergen.TaskAcceptedDeleteFrame(id.ObjectID, id.OwnerID, rowA), nil)
	out.addAll(entergen.TaskCompleteNoticeFrames(int(taskID)))
	s.npcTaskMarkFrames(pl, out)
	next := s.npcTaskAcceptChain(pl, out, task)
	out.add(entergen.TaskCompleteActionFrame(id))
	if tail.npc.ObjectID != 0 {
		out.add(entergen.TaskTalkEndObjectFrame(tail.npc))
	}
	if tail.closeTalk {
		out.add(entergen.TalkCloseFrame())
	}
	s.sendTaskFrames(pl, out)
	if next != 0 {
		s.trace("npc task: 完成 %d → 链推进接取 %d client=%d（余 %d）", taskID, next, pl.Conn1, len(pl.roleState.TaskAccepted))
	} else {
		s.trace("npc task: 完成 %d client=%d（余 %d）", taskID, pl.Conn1, len(pl.roleState.TaskAccepted))
	}
	s.sendRoleSave(pl)
	if next != 0 {
		s.npcTaskAutoSteps(pl, uint32(next))
		if nextTask, ok := catalog.TaskByID(next); ok {
			s.npcTaskScheduleMovie(pl, nextTask)
		}
	}
	return true
}

// npcTaskAcceptChain 任务的 AutoAcceptID 后继接取簇（追加到 out）。后继已在
// 进度中或目录里没有则不发；已交的后继（建号种子的功能门行，如引导链末环
// 1346）照常重新接取。返回新接的后继任务号（无则 0）。
func (s *Server) npcTaskAcceptChain(pl *Player, out *taskFrames, task taskdata.Task) int {
	next := task.AutoAcceptID
	if next <= 0 {
		return 0
	}
	if _, ok := s.npcTaskCatalogOf().TaskByID(next); !ok {
		return 0
	}
	if pl.roleState.TaskTrackOf(uint32(next)) != nil {
		return 0
	}
	if !s.npcTaskAcceptFrames(pl, out, next) {
		return 0
	}
	return next
}

// ---- 自动步与占位推进 ----

// npcTaskStub NINEYIN_TASK_STUB：guide（缺省，只占位新手引导链任务）｜all
// ｜off。占位对象＝服务端无从达成的步骤：杀怪步（无战斗通路）、特殊步
// （打坐/经脉等条件）、无对话的事件交互步。
func npcTaskStub() string {
	switch value := strings.ToLower(strings.TrimSpace(os.Getenv("NINEYIN_TASK_STUB"))); value {
	case "all":
		return "all"
	case "off", "0", "false":
		return "off"
	default:
		return "guide"
	}
}

// npcTaskStubbed 任务的自动步是否走占位。
func npcTaskStubbed(task taskdata.Task) bool {
	switch npcTaskStub() {
	case "all":
		return true
	case "off":
		return false
	}
	return task.IsGuide
}

// npcTaskAutoStepKind 当前步的自动推进方式：特殊步到步即过；杀怪步与无
// 对话的事件交互步定时占位。两者都受占位开关约束。
func npcTaskAutoStepKind(task taskdata.Task, step taskdata.Step) (immediate, scheduled bool) {
	if !npcTaskStubbed(task) {
		return false, false
	}
	switch {
	case step.Subtype == taskdata.StepSpecial:
		return true, false
	case step.Subtype == taskdata.StepHunter:
		return false, true
	case step.Subtype == taskdata.StepInteract && step.Scene == "":
		return false, true
	}
	return false, false
}

// npcTaskAutoSteps 自动步推进（接取/步进后调用）：到步即过的特殊步连推到
// 第一个需要玩家操作的步；定时类排一次占位后返回。
func (s *Server) npcTaskAutoSteps(pl *Player, taskID uint32) {
	for {
		catalog := s.npcTaskCatalogOf()
		task, inCatalog := catalog.TaskByID(int(taskID))
		track := pl.roleState.TaskTrackOf(taskID)
		if !inCatalog || track == nil {
			return
		}
		if len(task.Steps) == 0 {
			// 没有步骤数据的自动提交任务：接取即可提交。
			if task.AutoSubmit {
				s.npcTaskCompleteFrames(pl, taskID, npcTaskCompleteTail{})
			}
			return
		}
		if track.Step < 1 || track.Step > len(task.Steps) {
			return
		}
		step := task.Steps[track.Step-1]
		if npcTaskStepDone(track, step) {
			return // 末步已记满、等玩家提交：没有可自动推进的步
		}
		immediate, scheduled := npcTaskAutoStepKind(task, step)
		if scheduled {
			s.npcTaskScheduleStub(pl, taskID)
			return
		}
		if !immediate {
			return
		}
		home := npcTaskSpawnIdentity(s.npcTaskSpawnOf(pl, task.SubmitNpc))
		if !s.npcTaskAdvanceStep(pl, taskID, npcTaskDialogTail(home, task, track.Step-1)) {
			return
		}
	}
}

// npcTaskScheduleStub 占位定时器：到点时当前步仍是占位步才记一次进度
// （玩家离线/任务已变则作废），步进后再排下一步。
func (s *Server) npcTaskScheduleStub(pl *Player, taskID uint32) {
	time.AfterFunc(npcTaskStubInterval, func() {
		if pl.State != PlOnline {
			return
		}
		s.npcTaskStubTick(pl, taskID)
	})
}

// npcTaskStubTick 占位推进一拍（定时器到点；测试直调）：当前步仍是占位步
// 才记一次进度，之后接自动步（多只怪／下一个占位步）。
func (s *Server) npcTaskStubTick(pl *Player, taskID uint32) {
	s.npcTaskLocked(pl, func() {
		if pl.State != PlOnline {
			return
		}
		task, track, step, ok := npcTaskCurrent(s.npcTaskCatalogOf(), pl, taskID)
		if !ok || npcTaskStepDone(track, step) {
			return
		}
		if _, scheduled := npcTaskAutoStepKind(task, step); !scheduled {
			return
		}
		s.trace("npc task: 占位推进 %d step=%d（%s）client=%d", taskID, step.Order, step.TrackInfo, pl.Conn1)
		if step.Subtype == taskdata.StepHunter {
			// 杀怪：官服帧序 task_ok → m18 [被杀目标]，不关对话。
			s.npcTaskStepDone(pl, taskID, npcTaskStepTail{npc: npcTaskSpawnIdentity(s.npcTaskSpawnOf(pl, step.Key))})
		} else {
			// 事件交互步：同对话窗收尾（m18 [任务 NPC] → m8）。
			home := npcTaskSpawnIdentity(s.npcTaskSpawnOf(pl, task.SubmitNpc))
			s.npcTaskStepDone(pl, taskID, npcTaskDialogTail(home, task, step.Order-1))
		}
	})
}

// ---- 追踪列表 ----

// npcTaskTraceList 解析客户端追踪列表 "id,id,…"（lua 拼串里会混进一个 0
// 占位，忽略）。
func npcTaskTraceList(list string) map[uint32]bool {
	tracked := make(map[uint32]bool)
	for _, field := range strings.Split(list, ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(field)); err == nil && id > 0 {
			tracked[uint32(id)] = true
		}
	}
	return tracked
}

// handleTaskTraceList 客户端追踪列表（m327 [6,"id,…"]）回置追踪标志：列表
// 内的已接行按官服连写 track=0、track=1；原先追踪而不在列表里的写 0。
func (s *Server) handleTaskTraceList(pl *Player, list string) {
	if !pl.haveRoleState {
		return
	}
	s.npcTaskLocked(pl, func() {
		tracked := npcTaskTraceList(list)
		id := pl.entergenIdentity()
		out := s.taskFrameSink(pl)
		for _, taskID := range pl.roleState.TaskAccepted {
			track := pl.roleState.TaskTrackOf(taskID)
			if track == nil {
				continue
			}
			row, _ := npcTaskAcceptedRow(pl, taskID)
			if tracked[taskID] {
				track.Tracked = true
				out.add(entergen.TaskAcceptedGridFrame(id.ObjectID, id.OwnerID, row, entergen.GridInt(entergen.TaskColTrack, 0)))
				out.add(entergen.TaskAcceptedGridFrame(id.ObjectID, id.OwnerID, row, entergen.GridInt(entergen.TaskColTrack, 1)))
			} else if track.Tracked {
				track.Tracked = false
				out.add(entergen.TaskAcceptedGridFrame(id.ObjectID, id.OwnerID, row, entergen.GridInt(entergen.TaskColTrack, 0)))
			}
		}
		if len(out.list) == 0 {
			return
		}
		s.sendTaskFrames(pl, out)
		s.sendRoleSave(pl)
	})
}

// ---- 进场整理与链根接取 ----

// normalizeTaskTracks 进场时给「已接但没有进度行」的流程任务补进度行：
// 旧流程只往已接表追加了任务号（没有目标行/追踪标志，任务行也不出），
// 补行后对话行与追踪栏恢复。建号种子的功能门行（SeedTaskIDs）不是玩家
// 接的任务，不补。
func (s *Server) normalizeTaskTracks(pl *Player) {
	if !pl.haveRoleState {
		return
	}
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		return
	}
	s.npcTaskLocked(pl, func() {
		seedAccepted, _ := rolestate.SeedTaskIDs()
		gate := make(map[uint32]bool, len(seedAccepted))
		for _, id := range seedAccepted {
			gate[id] = true
		}
		fixed := 0
		for _, id := range pl.roleState.TaskAccepted {
			if gate[id] || pl.roleState.TaskTrackOf(id) != nil {
				continue
			}
			if _, ok := catalog.TaskByID(int(id)); !ok {
				continue
			}
			track := pl.roleState.EnsureTaskTrack(id, 0)
			track.Tracked, track.Step = true, 1
			fixed++
		}
		if fixed > 0 {
			s.trace("npc task: 进场补任务进度行 %d 条 client=%d", fixed, pl.Conn1)
		}
	})
}

// npcAcceptSceneGuideRoots 新手引导链根的进场自动接取（官服：1340 进场约
// 15 秒后自动接取）。只认 IsGuide 链根里接取 NPC 摆放在当前场景的任务；
// 其余链根仍走首谈即接。
func (s *Server) npcAcceptSceneGuideRoots(pl *Player) {
	if !pl.haveRoleState {
		return
	}
	catalog := s.npcTaskCatalogOf()
	if catalog == nil {
		return
	}
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	if scene == "" {
		return
	}
	placed := make(map[string]bool)
	for _, spawn := range s.sceneNPCCatalog(scene).spawns {
		placed[strings.ToLower(spawn.ConfigID)] = true
	}
	for _, task := range catalog.ChainRootsFor(true) {
		if !placed[strings.ToLower(task.AcceptNpc)] {
			continue
		}
		if s.acceptNPCTask(pl, task.ID, "链根进场") {
			s.trace("npc task: 引导链根 %d 进场自动接取@%s client=%d", task.ID, scene, pl.Conn1)
		}
	}
}

// npcTaskEnterScene 进场景（NPC 物化之后）：补发 NPC 任务标记；延时接取本
// 场景的新手引导链根（官服 1340 在进场约 15 秒后自动接取；到点时按玩家
// 当时所在场景判定）。
func (s *Server) npcTaskEnterScene(pl *Player) {
	if pl.State != PlOnline || !pl.haveRoleState {
		return
	}
	s.pushNPCTaskMarks(pl)
	time.AfterFunc(npcTaskGuideRootDelay, func() {
		if pl.State != PlOnline {
			return
		}
		s.npcAcceptSceneGuideRoots(pl)
	})
}
