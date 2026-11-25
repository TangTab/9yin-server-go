package entergen

// 挂载序列生成器：MountPlan（golden 0x14 帧提取）驱动的 0x14 + 行推
// 交织编排。官方配对模式（golden 48 帧实证）：每条 0x14 挂载后紧跟行推
// ——纯数字列走 0x11、字符串列走 0x29 批量；无数据源的记录挂载即止
//（客户端收到 schema 挂载即为空表，官方语义）。
//
// 行源分级：
//   - RoleState 有数据（Task_Completed/Task_Accepted/Task_Record/
//     QingGongRec）→ 原生行推（挂载后紧跟）；
//   - 其余 → mount-only（数据源接入后在本文件 rowGenerators 注册即可
//     自动生效）。
//
// 任务三表（官服引导抓包方言，编码见 taskrecords.go）：
//
//	Task_Completed  完成位图：行＝任务号/32、位＝任务号%32，从 0 行起
//	                逐行显式行号加（含零值行）
//	Task_Accepted   已接行：一张 0x29 批量帧（从第 0 行起整表），每行
//	                28 列——任务主表列 + 玩家进度（追踪标志/当前步/状态
//	                /接取时刻）
//	Task_Record     目标行：每个流程接取任务的每个步骤一行，按已接
//	                行序拼接、显式行号逐行加

import (
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// mountRowGenerator 一条记录的行推生成器：返回（frames, 是否生成行）。
// nil 生成器 = mount-only。
type mountRowGenerator func(ctx Context, recordIndex uint16) ([][]byte, error)

// rowGenerators 记录名 → 行推生成器（RoleState 可计算子集）。
var rowGenerators = map[string]mountRowGenerator{
	"Task_Completed": taskCompletedRows,
	"Task_Accepted":  taskAcceptedRows,
	"Task_Record":    taskRecordRows,
	"Repute_Record":  reputeRows,
	"QingGongRec":    qingGongRows,
}

// reputeRows Repute_Record 声望行：角色态的声望记录逐行显式行号加（行序
// 即累加序，与运行期 AddRepute 返回的行下标一致）。
func reputeRows(ctx Context, recordIndex uint16) ([][]byte, error) {
	frames := make([][]byte, 0, len(ctx.State.Reputes))
	for i, repute := range ctx.State.Reputes {
		frame, err := TaskReputeRowFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, uint16(i), repute.Name, repute.Count)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// MountSequenceFrames 按官方顺序生成挂载计划的行推族。
// 0x14 挂载帧不发（2026-11-22 晚定案：aow/golden 方言遗物——2433 fork
// 官服重登/切图对拍零 0x14，fork 客户端记录 schema 走 0x0A 全表自足；
// 挂载帧在真机触发 ServerRecordAddRow Out range 越界族（BppEquipRec/
// consign_history/vip_info_rec）＋重登/切图后状态累积卡顿）。行推保留
// （2433 重登照发 0x11 记录行海）。
func MountSequenceFrames(ctx Context) ([][]byte, error) {
	frames := make([][]byte, 0, len(MountPlan)*2)
	for _, spec := range MountPlan {
		recordIndex := clientdata.MustRecordIndexByName(spec.Name)
		if gen, ok := rowGenerators[spec.Name]; ok {
			rows, err := gen(ctx, recordIndex)
			if err != nil {
				return nil, err
			}
			frames = append(frames, rows...)
		}
	}
	return frames, nil
}

// taskCompletedRows Task_Completed 完成位图行：行＝任务号/32、位＝
// 任务号%32（官服映射，见 taskrecords.go），从第 0 行到最大行逐行显式
// 行号加（含零值行——登录簇整表下发后，增量置位才知道哪些行客户端已有；
// 功能门 OpenGate 49634 → 行 1551，共 1552 行）。
func taskCompletedRows(ctx Context, recordIndex uint16) ([][]byte, error) {
	rows := TaskCompletedMask(ctx.State.TaskCompleted)
	if len(rows) == 0 {
		return nil, nil
	}
	maxRow := uint16(0)
	for row := range rows {
		if row > maxRow {
			maxRow = row
		}
	}
	frames := make([][]byte, 0, int(maxRow)+1)
	for row := uint16(0); row <= maxRow; row++ {
		frame, err := RecordAddRowFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, recordIndex, row,
			[]RecordCell{RecordInt(rows[row])})
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// taskAcceptedRows Task_Accepted 已接行：一张 0x29 批量帧（从第 0 行起
// 整表，角色态的已接任务序即客户端行序）。行内容见 TaskAcceptedRowCells。
func taskAcceptedRows(ctx Context, recordIndex uint16) ([][]byte, error) {
	if len(ctx.State.TaskAccepted) == 0 {
		return nil, nil
	}
	rows := make([][]RecordCell, 0, len(ctx.State.TaskAccepted))
	for _, taskID := range ctx.State.TaskAccepted {
		rows = append(rows, TaskAcceptedRowCells(ctx, taskID))
	}
	frame, err := recordBatch0x29(ctx, recordIndex, rows)
	if err != nil {
		return nil, err
	}
	return [][]byte{frame}, nil
}

// TaskAcceptedRowCells 玩家一条已接行的 28 列：任务主表列取自任务目录
// （ctx.Tasks；nil 或目录里没有的任务，除任务号外各列为空——建号种子的
// 功能门行），玩家进度取自角色态的进度行：
//
//	无进度行（不是流程接取的）：不追踪、状态 0、步序 0、接取时刻 0；
//	有进度行：追踪；步序＝当前步序；状态＝可提交（TaskSubmittable）→ 2，
//	否则 0。
func TaskAcceptedRowCells(ctx Context, taskID uint32) []RecordCell {
	task := ctx.TaskByID(taskID)
	view := TaskAcceptedView{Status: TaskStatusDoing}
	if track := ctx.State.TaskTrackOf(taskID); track != nil {
		view.Tracked = track.Tracked
		view.Step = uint32(track.Step)
		view.Accepted = track.Time
		if TaskSubmittable(track, task.Steps) {
			view.Status = TaskStatusCanSubmit
		}
	}
	return TaskAcceptedCells(task, view)
}

// taskRecordRows Task_Record 目标行：每个有进度行的已接任务，其每个步骤
// 一行，按已接任务序拼接、显式行号逐行加（客户端行序须与 fssoloworld
// 推算的行号一致：按已接序、每任务 steps 行）。行内容的进度取自进度行。
func taskRecordRows(ctx Context, recordIndex uint16) ([][]byte, error) {
	var frames [][]byte
	row := uint16(0)
	for _, taskID := range ctx.State.TaskAccepted {
		track := ctx.State.TaskTrackOf(taskID)
		if track == nil {
			continue
		}
		for _, step := range ctx.TaskByID(taskID).Steps {
			frame, err := TaskRecordRowFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, row, taskID, step,
				TaskRecordProgress(track, step), ctx.taskStamp())
			if err != nil {
				return nil, err
			}
			frames = append(frames, frame)
			row++
		}
	}
	return frames, nil
}

// TaskStepsDone 任务的各步是否都已完成。
func TaskStepsDone(track *rolestate.TaskTrack, steps []taskdata.Step) bool {
	if track == nil {
		return false
	}
	for i, step := range steps {
		if i >= len(track.Cur) || track.Cur[i] < step.MaxNum {
			return false
		}
	}
	return true
}

// TaskSubmittable 任务是否可提交：各步都已完成；没有步骤数据的任务没有
// 可达成的目标，接取即可提交（否则这类任务永远交不了）。
func TaskSubmittable(track *rolestate.TaskTrack, steps []taskdata.Step) bool {
	if len(steps) == 0 {
		return track != nil
	}
	return TaskStepsDone(track, steps)
}

// TaskRecordProgress 一条目标行的进度视图：已完成次数取进度行；flags 列
// 该步已完成 → 48（杀怪步 49），杀怪步是当前步 → 1，其余 0（官服帧序
// 的终态）。
func TaskRecordProgress(track *rolestate.TaskTrack, step taskdata.Step) TaskRecordView {
	view := TaskRecordView{}
	if step.Order-1 < len(track.Cur) {
		view.CurNum = uint32(track.Cur[step.Order-1])
	}
	hunter := step.Subtype == taskdata.StepHunter
	switch {
	case int(view.CurNum) >= step.MaxNum:
		view.Flags = TaskRecordFlagSettled
		if hunter {
			view.Flags |= TaskRecordFlagHunterActive
		}
	case hunter && step.Order == track.Step:
		view.Flags = TaskRecordFlagHunterActive
	}
	return view
}

// TaskByID 任务目录里的任务；没有目录（ctx.Tasks 为 nil）或目录里没有
// 该任务时，只有任务号的零值行。
func (ctx Context) TaskByID(taskID uint32) taskdata.Task {
	if ctx.Tasks != nil {
		if task, ok := ctx.Tasks.TaskByID(int(taskID)); ok {
			return task
		}
	}
	return taskdata.Task{ID: int(taskID)}
}

// taskStamp 目标行 stamp 列的计时戳：ctx.Stamp（0＝当前时间）。
func (ctx Context) taskStamp() int64 {
	if ctx.Stamp != 0 {
		return ctx.Stamp
	}
	return time.Now().UnixNano()
}

// qingGongRows QingGongRec 轻功类别行（单列窄串，逐行 0x11 官方形态）。
func qingGongRows(ctx Context, recordIndex uint16) ([][]byte, error) {
	frames := make([][]byte, 0, len(ctx.State.QingGongCategories))
	for _, cat := range ctx.State.QingGongCategories {
		cell := cat
		frame, err := RecordAddCellsFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, recordIndex,
			[]RecordCell{RecordString(cell)})
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
