package entergen

// taskrecords.go 任务记录的行编码、增量帧与引擎通知帧。方言按官服引导
// 抓包（初始账号登陆-完整引导-女.pcapng，8032）逐列、逐字节定版；列名见
// 客户端 form_task/task_define.lua 的 accept_rec_* / task_rec_* 常量。
//
//	Task_Accepted（已接表，28 列）
//	  id | line | "Server" | submit_scene | submit_npc | title | status |
//	  track | step | time(int64) | plot | can_giveup(byte) | target |
//	  context | accept_npc | accept_scene | submit_npc_ex | prize_base |
//	  prize_ex | prize_over | accept_limit | submit_limit | repute_limit |
//	  vip(byte) | sceneid | prize_condition | 0 | 0
//	  status 0 进行中 / 2 可提交；track 1＝追踪（客户端追踪栏只显示 1 的
//	  行）；step 当前步序（接取后由 0x13 置 1）；除场景/线号外都是键。
//	Task_Record（目标表，15 列）
//	  id | subtype | index | order | curnum | maxnum | key | scene |
//	  trackinfo | flags | hyperlink | stamp(int64) | 0 | 0 | ""
//	  scene 列交互步＝DialogID、剧情/特殊＝空串（长度 1）、计数类目标（杀怪）
//	  ＝0 长度串；hyperlink 列无超链接时是空串；flags 杀怪步成为当前步置
//	  1，一步完成先 +32 再 +48；stamp 低 20 位 = (index & 0x3FF) << 10 |
//	  subtype，高位是服务端计时戳。
//	Task_Completed（完成位图，单列 int32）
//	  行 = 任务号 / 32，位 = 任务号 % 32（官服五处更新一致：1340 → 行 41
//	  位 28、1347 → 行 42 位 3 …）。
//
// 增量协议（客户端两张表按行号管理，服务端逐行增删——不是整表重写）：
//
//	接取 T：0x11 Task_Record × n（row=起始行+i）→ 0x11 Task_Accepted
//	          （row=已接表下标；表空时官服用 0x29 批量帧，等价）→ 0x13
//	          step=1（杀怪步为当前步再 0x13 flags=1）→ 通知簇
//	一步完成：0x13 curnum → 0x13 step=k+1 或 status=2 → 0x13 flags+32
//	          → 0x13 flags+48 → m24 "task_ok" → m18 → m8
//	完成 T：m209 [8, T] → m1 "9631" → 0x12 Task_Record row=起始行 × n
//	          （每删一行后续前移，官服恒删起始行）→ 0x12 Task_Accepted
//	          → 完成位图 → （后继任务接取序列）→ m24 "task_complete"
//	          → m18 → m8
//
// 通知簇（C++ 引擎侧消费，无 lua 处理）：
//
//	接取  m209 [9, T] → m24 [玩家, 40, "task_accept"] → m1 [1, "9628", "title_T"]
//	完成  m209 [8, T] → m1 [1, "9631", "title_T"] … m24 [玩家, 40, "task_complete"]
//	提交窗 m209 [2, NPC, 201×10⁶+T, T, "complete_T", "submit_T", 0, "", Line]
//	      （点「完成任务」行后下发，客户端弹提交窗并回 mark 201）

import (
	"fmt"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// 三张任务记录在官方记录表里的下标。
var (
	recordTaskAccepted  = clientdata.MustRecordIndexByName("Task_Accepted")
	recordTaskRecord    = clientdata.MustRecordIndexByName("Task_Record")
	recordTaskCompleted = clientdata.MustRecordIndexByName("Task_Completed")
)

// Task_Accepted 可变列号（task_define.lua accept_rec_*）。
const (
	TaskColStatus = 6 // 0 进行中 / 1 失败 / 2 可提交
	TaskColTrack  = 7 // 1 追踪
	TaskColStep   = 8 // 当前步序
	TaskColTime   = 9 // 接取时刻（int64）
)

// Task_Record 可变列号（task_define.lua task_rec_*）。
const (
	RecordColCurNum = 4
	RecordColFlags  = 9
)

// 任务状态（task_define.lua task_status_*）。
const (
	TaskStatusDoing     uint32 = 0
	TaskStatusCanSubmit uint32 = 2
)

// TaskRecordFlags 目标行 flags 列的位（官服帧序：杀怪步成为当前步置 1，
// 一步完成先写 |32 再写 |48；杀怪步两位写都带 1：33、49）。
const (
	TaskRecordFlagHunterActive uint32 = 1
	TaskRecordFlagDone         uint32 = 32
	TaskRecordFlagSettled      uint32 = 48
)

// TaskAcceptedView 一条已接行的玩家侧状态。
type TaskAcceptedView struct {
	Status   uint32 // TaskStatusDoing / TaskStatusCanSubmit
	Tracked  bool   // 追踪标志
	Step     uint32 // 当前步序（1 起）
	Accepted int64  // 接取时刻（Unix 秒；未知 0）
}

// TaskAcceptedCells 已接行（28 列）。task 为目录里的任务行；不在目录里
// 的任务（零值）除任务号外各列皆空。
func TaskAcceptedCells(task taskdata.Task, view TaskAcceptedView) []RecordCell {
	track := uint32(0)
	if view.Tracked {
		track = 1
	}
	return []RecordCell{
		RecordInt(uint32(task.ID)), RecordInt(uint32(task.Line)), RecordString("Server"),
		RecordInt(uint32(task.SubmitScene)), RecordString(task.SubmitNpc), RecordString(task.TitleKey),
		RecordInt(view.Status), RecordInt(track), RecordInt(view.Step), RecordInt64(view.Accepted),
		RecordInt(0), RecordByte(0),
		RecordString(task.TargetKey), RecordString(task.ContextKey), RecordString(task.AcceptNpc),
		RecordInt(uint32(task.AcceptScene)), RecordString(task.SubmitNpcEx), RecordString(task.Prize),
		RecordString(""), RecordString(""), RecordString(""), RecordString(""), RecordString(""),
		RecordByte(0), RecordInt(0), RecordString(""), RecordInt(0), RecordInt64(0),
	}
}

// TaskRecordView 一条目标行的玩家侧状态。
type TaskRecordView struct {
	CurNum uint32 // 已完成次数
	Flags  uint32 // 见 taskRecordFlag*
}

// TaskRecordCells 目标行（15 列）。stamp 是服务端计时戳（只有低 20 位
// 以外的部分入列）。
func TaskRecordCells(taskID uint32, step taskdata.Step, view TaskRecordView, stamp int64) []RecordCell {
	scene, hyperlink := RecordString(""), RecordString("")
	switch {
	case step.NullScene:
		scene = RecordNullString()
	case step.Scene != "":
		scene = RecordString(step.Scene)
	}
	if step.HasLink {
		hyperlink = RecordString(TaskRecordHyperlink(taskID, step))
	}
	return []RecordCell{
		RecordInt(taskID), RecordInt(uint32(step.Subtype)), RecordInt(uint32(step.Index)), RecordInt(uint32(step.Order)),
		RecordInt(view.CurNum), RecordInt(uint32(step.MaxNum)),
		RecordString(step.Key), scene, RecordString(step.TrackInfo), RecordInt(view.Flags), hyperlink,
		RecordInt64(TaskRecordStamp(stamp, step)),
		RecordInt(0), RecordInt(0), RecordString(""),
	}
}

// TaskRecordHyperlink 目标行超链接串 "<任务号>_<子类型>_<Key>"。
func TaskRecordHyperlink(taskID uint32, step taskdata.Step) string {
	return fmt.Sprintf("%d_%d_%s", taskID, step.Subtype, step.Key)
}

// TaskRecordStamp 目标行的 stamp 列：计时戳高位 | (步骤号 & 0x3FF) << 10 |
// 子类型。
func TaskRecordStamp(stamp int64, step taskdata.Step) int64 {
	return stamp&^0xFFFFF | int64((step.Index&0x3FF)<<10|step.Subtype)
}

// TaskAcceptedRowFrame 已接表加行（0x11，row=已接表下标）。
func TaskAcceptedRowFrame(objectID, ownerID uint32, row uint16, task taskdata.Task, view TaskAcceptedView) ([]byte, error) {
	return RecordAddRowFrame(objectID, ownerID, recordTaskAccepted, row, TaskAcceptedCells(task, view))
}

// TaskRecordRowFrame 目标表加行（0x11，row=该任务目标行的起始下标 + 步骤序 - 1）。
func TaskRecordRowFrame(objectID, ownerID uint32, row uint16, taskID uint32, step taskdata.Step, view TaskRecordView, stamp int64) ([]byte, error) {
	return RecordAddRowFrame(objectID, ownerID, recordTaskRecord, row, TaskRecordCells(taskID, step, view, stamp))
}

// TaskRecordDeleteFrames 目标表删行（0x12 × count，恒删 start 行：每删
// 一行后续行前移）。
func TaskRecordDeleteFrames(objectID, ownerID uint32, start uint16, count int) [][]byte {
	frames := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		frames = append(frames, RecordDelRowFrame(objectID, ownerID, recordTaskRecord, start))
	}
	return frames
}

// TaskAcceptedDeleteFrame 已接表删行（0x12）。
func TaskAcceptedDeleteFrame(objectID, ownerID uint32, row uint16) []byte {
	return RecordDelRowFrame(objectID, ownerID, recordTaskAccepted, row)
}

// TaskAcceptedGridFrame 已接表逐格写（0x13）。
func TaskAcceptedGridFrame(objectID, ownerID uint32, row uint16, cells ...RecordGridCell) ([]byte, error) {
	return RecordGridFrame(objectID, ownerID, recordTaskAccepted, row, cells...)
}

// TaskRecordGridFrame 目标表逐格写（0x13）。
func TaskRecordGridFrame(objectID, ownerID uint32, row uint16, cells ...RecordGridCell) ([]byte, error) {
	return RecordGridFrame(objectID, ownerID, recordTaskRecord, row, cells...)
}

// TaskMarkFrame NPC 任务标记 m209 [15, obj, state]（引擎侧消费：NPC 头顶
// 任务图标）。state 位：1 当前步在此 NPC、2 可提交/任务待回此 NPC、4 可
// 接取、8 当前步是剧情步——位义按官服引导抓包各簇取值归纳。
func TaskMarkFrame(objectID, ownerID uint32, state uint32) ([]byte, error) {
	return CustomMessageFrame(209, CustomInt(15), CustomObject(objectID, ownerID), CustomInt(int32(state)))
}

// TaskAcceptMovieFrame 接取引导影像 m104 [0/0, 影像号, 4]（主表 AcceptMovie；
// 官服在接取约半秒后、关闭对话 m8 之后下发）。
func TaskAcceptMovieFrame(movie int) ([]byte, error) {
	return CustomMessageFrame(104, CustomObject(0, 0), CustomInt(int32(movie)), CustomInt(4))
}

// TaskStepOkActionFrame 一步完成动作 m24 [玩家, 40, "task_ok"]（m18/提交窗
// 参数/m8 的收尾由调用方按场景拼）。
func TaskStepOkActionFrame(player Identity) ([]byte, error) {
	return CustomMessageFrame(24, CustomObject(player.ObjectID, player.OwnerID), CustomInt(40), CustomString("task_ok"))
}

// TalkCloseFrame 对话结束 m8（剧情窗/提交窗收尾的单发形态）。
func TalkCloseFrame() ([]byte, error) {
	return CustomMessageFrame(8)
}

// TaskTalkEndObjectFrame m18 [NPC, ""]（一步完成/提交收尾里不带 m8 的单帧）。
func TaskTalkEndObjectFrame(npc Identity) ([]byte, error) {
	return CustomMessageFrame(18, CustomObject(npc.ObjectID, npc.OwnerID), CustomString(""))
}

// TaskCompletedRow 任务号 → 完成位图的行号与位掩码。
func TaskCompletedRow(taskID uint32) (row uint16, mask uint32) {
	return uint16(taskID / 32), 1 << (taskID % 32)
}

// TaskCompletedMask 任务号集合 → 完成位图各行的掩码。
func TaskCompletedMask(ids []uint32) map[uint16]uint32 {
	rows := make(map[uint16]uint32)
	for _, id := range ids {
		if id == 0 {
			continue
		}
		row, mask := TaskCompletedRow(id)
		rows[row] |= mask
	}
	return rows
}

// TaskCompletedSetFrames 置一个完成位的增量帧。completed 是**完成之前**
// 的已交任务集：登录簇把它的位图从第 0 行整表下发（到最大行，含零值
// 行），所以小于其最大行的位图行客户端已有——该行只需 0x13 写格；超出
// 的行先 0x11 逐行加（官服 #2413 行 42 先加零值行、#2414 再写格），再
// 写格。
func TaskCompletedSetFrames(objectID, ownerID uint32, taskID uint32, completed []uint32) ([][]byte, error) {
	rows := TaskCompletedMask(completed)
	known := 0
	for row := range rows {
		if int(row) >= known {
			known = int(row) + 1
		}
	}
	row, bit := TaskCompletedRow(taskID)
	var frames [][]byte
	for r := known; r <= int(row); r++ {
		add, err := RecordAddRowFrame(objectID, ownerID, recordTaskCompleted, uint16(r), []RecordCell{RecordInt(0)})
		if err != nil {
			return nil, err
		}
		frames = append(frames, add)
	}
	write, err := RecordGridFrame(objectID, ownerID, recordTaskCompleted, row, GridInt(0, rows[row]|bit))
	if err != nil {
		return nil, err
	}
	return append(frames, write), nil
}

// TaskAcceptNoticeFrames 接取通知：m209 [9, T] → m24 "task_accept" →
// m1 "9628"（「接受任务」系统提示，标题键作参数）。
func TaskAcceptNoticeFrames(player Identity, taskID int) ([][]byte, error) {
	return buildFrames(
		func() ([]byte, error) {
			return CustomMessageFrame(209, CustomInt(9), CustomInt(int32(taskID)))
		},
		func() ([]byte, error) {
			return CustomMessageFrame(24, CustomObject(player.ObjectID, player.OwnerID), CustomInt(40), CustomString("task_accept"))
		},
		func() ([]byte, error) {
			return CustomMessageFrame(1, CustomInt(1), CustomString("9628"), CustomString(fmt.Sprintf("title_%d", taskID)))
		},
	)
}

// TaskCompleteNoticeFrames 完成通知的前半：m209 [8, T] → m1 "9631"
// （「完成任务」系统提示）。
func TaskCompleteNoticeFrames(taskID int) ([][]byte, error) {
	return buildFrames(
		func() ([]byte, error) {
			return CustomMessageFrame(209, CustomInt(8), CustomInt(int32(taskID)))
		},
		func() ([]byte, error) {
			return CustomMessageFrame(1, CustomInt(1), CustomString("9631"), CustomString(fmt.Sprintf("title_%d", taskID)))
		},
	)
}

// TaskCompleteActionFrame 完成动作 m24 [玩家, 40, "task_complete"]（官服
// 置于后继任务的接取通知之后）。
func TaskCompleteActionFrame(player Identity) ([]byte, error) {
	return CustomMessageFrame(24, CustomObject(player.ObjectID, player.OwnerID), CustomInt(40), CustomString("task_complete"))
}

// TaskStepOkFrames 一步完成：m24 [玩家, 40, "task_ok"] → m18 [NPC, ""] → m8。
func TaskStepOkFrames(player, npc Identity) ([][]byte, error) {
	return buildFrames(
		func() ([]byte, error) {
			return CustomMessageFrame(24, CustomObject(player.ObjectID, player.OwnerID), CustomInt(40), CustomString("task_ok"))
		},
		func() ([]byte, error) {
			return CustomMessageFrame(18, CustomObject(npc.ObjectID, npc.OwnerID), CustomString(""))
		},
		func() ([]byte, error) { return CustomMessageFrame(8) },
	)
}

// TaskTalkEndFrames 对话收尾 m18 [NPC, ""] → m8（提交流 after 结算簇）。
func TaskTalkEndFrames(npc Identity) ([][]byte, error) {
	return buildFrames(
		func() ([]byte, error) {
			return CustomMessageFrame(18, CustomObject(npc.ObjectID, npc.OwnerID), CustomString(""))
		},
		func() ([]byte, error) { return CustomMessageFrame(8) },
	)
}

// buildFrames 按序编码一组帧；任一失败即整簇失败（通知半发比不发更难
// 排障，调用方按错误丢弃整簇）。
func buildFrames(builders ...func() ([]byte, error)) ([][]byte, error) {
	frames := make([][]byte, 0, len(builders))
	for _, build := range builders {
		frame, err := build()
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// TaskSubmitFormFrame 提交窗参数 m209 [2, NPC, 201×10⁶+T, T, "complete_T",
// "submit_T", 0, "", Line]：正文/提交文案键取主表 CompleteDialogId
// （"complete_T;submit_T"），缺席回落 complete_<T> / submit_<T>。
func TaskSubmitFormFrame(npc Identity, task taskdata.Task) ([]byte, error) {
	complete, submit := fmt.Sprintf("complete_%d", task.ID), fmt.Sprintf("submit_%d", task.ID)
	if keys := strings.Split(task.CompleteDialog, ";"); keys[0] != "" {
		complete = keys[0]
		if len(keys) > 1 && keys[1] != "" {
			submit = keys[1]
		}
	}
	return CustomMessageFrame(209,
		CustomInt(2),
		CustomObject(npc.ObjectID, npc.OwnerID),
		CustomInt(int32(201*1000000+task.ID)),
		CustomInt(int32(task.ID)),
		CustomString(complete),
		CustomString(submit),
		CustomInt(0),
		CustomString(""),
		CustomInt(int32(task.Line)),
	)
}
