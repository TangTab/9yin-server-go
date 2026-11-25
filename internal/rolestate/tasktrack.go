package rolestate

// tasktrack.go 流程接取任务的进度行（RoleState.TaskTracks）与建号种子
// 任务集合。
//
// 客户端的两张任务记录表由玩家进度驱动（列名见 task_define.lua）：
//
//	Task_Accepted  已接行：追踪标志（track）为 1 的才进追踪栏；当前步序
//	              （step）、状态（0 进行中 / 2 可提交）、接取时刻
//	Task_Record    目标行：每个任务每个步骤一行，进度 curnum/maxnum
//
// 建号种子的 TaskAccepted / TaskCompleted 是功能门行（b-line
// facultyAcceptedTaskIDs / grantFunctionUnlock 口径），不是玩家做任务
// 的痕迹：不给追踪标志、不下发目标行，链推进遇到它们按 SeedTaskIDs
// 识别。

// TaskTrack 一个流程接取任务的玩家侧进度。
type TaskTrack struct {
	TaskID  uint32 `json:"taskId"`
	Time    int64  `json:"time,omitempty"` // 接取时刻（Unix 秒；旧档 0）
	Tracked bool   `json:"tracked"`        // 追踪标志：接取即置位，客户端上报追踪列表时按列表回置
	Step    int    `json:"step"`           // 当前步序（1 起；0＝未开始）
	Cur     []int  `json:"cur,omitempty"`  // 各步已完成次数（步骤行序）
}

// TaskTrackOf 任务的进度行；没有返回 nil。
func (s *RoleState) TaskTrackOf(taskID uint32) *TaskTrack {
	for i := range s.TaskTracks {
		if s.TaskTracks[i].TaskID == taskID {
			return &s.TaskTracks[i]
		}
	}
	return nil
}

// EnsureTaskTrack 取任务的进度行，没有则追加（接取时刻 now）。
func (s *RoleState) EnsureTaskTrack(taskID uint32, now int64) *TaskTrack {
	if track := s.TaskTrackOf(taskID); track != nil {
		return track
	}
	s.TaskTracks = append(s.TaskTracks, TaskTrack{TaskID: taskID, Time: now})
	return &s.TaskTracks[len(s.TaskTracks)-1]
}

// RemoveTaskTrack 移除任务的进度行（保持余序；重建切片，不与旧档共享
// 底层数组的增长位）。
func (s *RoleState) RemoveTaskTrack(taskID uint32) {
	kept := make([]TaskTrack, 0, len(s.TaskTracks))
	for _, track := range s.TaskTracks {
		if track.TaskID != taskID {
			kept = append(kept, track)
		}
	}
	s.TaskTracks = kept
}

// SeedTaskIDs 建号种子的任务集合：accepted＝种进 Task_Accepted 的功能门
// 行，completed＝种进 Task_Completed 位图的功能门任务（含 0 值占位，与
// BuildRoleState 写入的顺序、内容一致）。
func SeedTaskIDs() (accepted, completed []uint32) {
	g := roleStateAssets.starter.TaskGates
	for id := atou32(g.Guide.First); id != 0 && id <= atou32(g.Guide.Last); id++ {
		accepted = append(accepted, id)
	}
	if v := atou32(g.Unlock.FacultyAlt); v != 0 {
		accepted = append(accepted, v)
	}
	for id := atou32(g.SchoolLead.First); id != 0 && id <= atou32(g.SchoolLead.Last); id++ {
		accepted = append(accepted, id)
	}
	for id := atou32(g.Act.First); id != 0 && id <= atou32(g.Act.Last); id++ {
		accepted = append(accepted, id)
	}
	if v := atou32(g.Unlock.FacultyActAlt); v != 0 {
		accepted = append(accepted, v)
	}
	completed = append(completed, atou32(g.Unlock.FunctionUnlock), atou32(g.Unlock.OpenGate))
	completed = append(completed, splitU32(g.FuncGate.IDs)...)
	return accepted, completed
}
