package entergen

// 挂载序列生成器：MountPlan（golden 0x14 帧提取）驱动的 0x14 + 行推
// 交织编排。官方配对模式（golden 48 帧实证）：每条 0x14 挂载后紧跟行推
// ——纯数字列走 0x11、字符串列走 0x29 批量；无数据源的记录挂载即止
//（客户端收到 schema 挂载即为空表，官方语义）。
//
// 行源分级：
//   - RoleState 有数据（Task_Completed 位图/Task_Accepted/QingGongRec）→
//     原生行推（挂载后紧跟）；
//   - 其余 44 条 → mount-only（数据源接入后在本文件 rowGenerators
//     注册即可自动生效）。

import "github.com/lizhanyin/9yin-server-go/internal/clientdata"

// mountRowGenerator 一条记录的行推生成器：返回（frames, 是否生成行）。
// nil 生成器 = mount-only。
type mountRowGenerator func(ctx Context, recordIndex uint16) ([][]byte, error)

// rowGenerators 记录名 → 行推生成器（RoleState 可计算子集）。
var rowGenerators = map[string]mountRowGenerator{
	"Task_Completed": taskCompletedRows,
	"Task_Accepted":  taskAcceptedRows,
	"QingGongRec":    qingGongRows,
}

// MountSequenceFrames 按官方顺序生成挂载计划：每项 0x14（+ 可用行推）。
// 记录名经 MustRecordIndexByName 在当前方言表解析——MountPlan 是官方
// 抓包提取，缺名即方言表错位，panic 编码期暴露。
func MountSequenceFrames(ctx Context) ([][]byte, error) {
	frames := make([][]byte, 0, len(MountPlan)*2)
	for _, spec := range MountPlan {
		recordIndex := clientdata.MustRecordIndexByName(spec.Name)
		frames = append(frames, RecordMountFrame(spec.Sub, ctx.Identity.ObjectID, ctx.Identity.OwnerID, recordIndex))
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

// taskCompletedRows Task_Completed 位图行（串行映射 row=(id-1)/32、
// mask=1<<((id-1)%32)；含 final-row 前全部零值行——b-line
// taskCompletedRows/grantFunctionUnlock 口径，OpenGate 49634 → 1552 行）。
func taskCompletedRows(ctx Context, recordIndex uint16) ([][]byte, error) {
	ids := ctx.State.TaskCompleted
	if len(ids) == 0 {
		return nil, nil
	}
	maxRow := 0
	rows := map[int]uint32{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		row := int((id - 1) / 32)
		rows[row] |= 1 << ((id - 1) % 32)
		if row > maxRow {
			maxRow = row
		}
	}
	frames := make([][]byte, 0, maxRow+1)
	for row := 0; row <= maxRow; row++ {
		v := rows[row]
		frame, err := RecordAddCellsFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, recordIndex,
			[]RecordCell{RecordInt(v)})
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// taskAcceptedRows Task_Accepted 行（28 列字符串记录——0x29 批量帧，
// 0x47 全字面量包装；首 7 列官方方言，尾列按 schema 补零）。
func taskAcceptedRows(ctx Context, recordIndex uint16) ([][]byte, error) {
	if len(ctx.State.TaskAccepted) == 0 {
		return nil, nil
	}
	frame, err := recordBatch0x29(ctx, recordIndex, acceptedTaskRows(ctx.State.TaskAccepted))
	if err != nil {
		return nil, err
	}
	return [][]byte{frame}, nil
}

// TaskAcceptedBatchFrame Task_Accepted 0x29 批量帧（挂载行推同源编码；
// 空表 nil）。m327 sub6 已接任务列表同步的幂等重放通道（fssoloworld）。
func TaskAcceptedBatchFrame(ctx Context) ([]byte, error) {
	if len(ctx.State.TaskAccepted) == 0 {
		return nil, nil
	}
	return recordBatch0x29(ctx, clientdata.MustRecordIndexByName("Task_Accepted"), acceptedTaskRows(ctx.State.TaskAccepted))
}

// acceptedTaskRows 已接任务行集（首 7 列官方方言，尾列由行编码器按
// schema 补零）。
func acceptedTaskRows(ids []uint32) [][]RecordCell {
	rows := make([][]RecordCell, 0, len(ids))
	for _, taskID := range ids {
		rows = append(rows, []RecordCell{
			RecordInt(taskID), RecordInt(0), RecordString(""),
			RecordInt(0), RecordString(""), RecordString(""), RecordInt(0),
		})
	}
	return rows
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
