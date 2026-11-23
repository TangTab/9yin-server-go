package fssoloworld

// npctask.go NPC 任务绑定（对话框任务行 + 接取/提交受理）：
//
// 数据：taskdata 快照（storytasklist/qianjiu_tasknpc.ini 派生的
// tasks.json，NPC→TID 绑定 + 任务文案）。装载路径：env
// NINEYIN_TASK_SNAPSHOT > <数据根>/taskdata/tasks.json（正式 Res）>
// <数据根>/../artifacts/local/tasks.json（仓内 testdata 布局）。
//
// 协议（form_talk_data.lua g_menu_id 域 + 引导抓包定版）：菜单任务行
// m6 funcid=flag*1000000+TID，行文本双键（首键=行文案、次键 title_
// <TID>=格式化参数）。on_endmenu 只聚合 on_addmenu 收到的行——客户端
// 不自行生成任务行，服务端不发即无任务入口。
//
// 受理：玩家点任务行回 op0x07 FuncID=任务域 mark。官方流（引导抓包）
// 中任务状态同步走 0x29 记录帧（接取/完成/链推进即时推），接取窗与
// 提交窗均由客户端从 REC_TASK_ACCEPT 本地渲染（全程无 m327、无
// task_msg 字符串路由）；提交后服务端回 m8×2 关窗。

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/taskdata"
)

// sortTaskLines 已并入 TasksForNPC 的升序保证（byNPC 绑定按 TID 排序）。

// npcTaskFuncBase g_menu_id.base：funcid = flag*base + TID。
const npcTaskFuncBase = 1000000

// 任务行 flag 域（form_talk_data.lua g_menu_id；「初始账号登陆-完整
// 引导-女」抓包实证）：100 接取 / 200 提交 / 201 提交+奖励单选（lua
// menu_select 的 funcid+singleselect*base 二次确认，c2s 恒随 200 后
// 出现）/ 500 进行中（官方已接任务行，双键 story_menu_<sid>+title_
// <TID>）/ 50N·60N 剧情段落续/离 / 600 离开。本服务端无进度系统：
// 已接任务发 200 可提交行（可玩闭环），500/剧情域仅识别留痕。
const (
	npcTaskFlagAccept       int32 = 100
	npcTaskFlagSubmit       int32 = 200
	npcTaskFlagSubmitChoice int32 = 201
	npcTaskFlagDoing        int32 = 500
)

// npcTaskLine 一条菜单任务行（提交行或接取行）。
type npcTaskLine struct {
	TaskID int
	Submit bool // true=已接发提交行；false=未接发接取行
}

// ---- 快照装载（进程级缓存） ----

var (
	npcTaskMu        sync.Mutex
	npcTaskCatalog   *taskdata.Catalog
	npcTaskRootTried string
)

// taskCatalogForRoot 装载任务目录（按数据根缓存；缺档返回 nil——任务行
// 是增强下发，缺数据不阻塞对话）。
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
	for _, path := range npcTaskSnapshotCandidates(root) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		catalog, err := taskdata.ParseSnapshot(data)
		if err != nil {
			continue
		}
		npcTaskCatalog = catalog
		return catalog
	}
	return nil
}

// npcTaskSnapshotCandidates 快照候选路径序：env 显式 > 数据根 taskdata
// 子目录（正式 Res/taskdata/tasks.json）> 仓内 artifacts/local（testdata
// 布局的兄弟目录）。
func npcTaskSnapshotCandidates(root string) []string {
	var paths []string
	if v := strings.TrimSpace(os.Getenv("NINEYIN_TASK_SNAPSHOT")); v != "" {
		paths = append(paths, filepath.Clean(v))
	}
	paths = append(paths,
		filepath.Join(root, "taskdata", "tasks.json"),
		filepath.Join(root, "..", "artifacts", "local", "tasks.json"),
	)
	return paths
}

// npcTaskMenuLimit NINEYIN_NPC_TASK_LIMIT 单 NPC 任务行上限（缺省 8——
// 客户端菜单每页 4 行翻页，过多任务行淹没服务行；<0 一键关停任务行）。
func npcTaskMenuLimit() int { return int(envFloat("NINEYIN_NPC_TASK_LIMIT", 8)) }

// npcTaskLines NPC 的菜单任务行集：绑定且已加载文案的任务按 TID 升序，
// 已接（TaskAccepted）发提交行、未接发接取行，截断至上限。
func npcTaskLines(catalog *taskdata.Catalog, configID string, accepted map[uint32]bool, limit int) []npcTaskLine {
	if catalog == nil || limit < 0 || strings.TrimSpace(configID) == "" {
		return nil
	}
	tasks := catalog.TasksForNPC(configID)
	lines := make([]npcTaskLine, 0, len(tasks))
	for _, task := range tasks {
		if limit > 0 && len(lines) >= limit {
			break
		}
		lines = append(lines, npcTaskLine{TaskID: task.ID, Submit: accepted[uint32(task.ID)]})
	}
	return lines
}

// npcAcceptedSet 玩家已接任务集合。
func npcAcceptedSet(pl *Player) map[uint32]bool {
	set := make(map[uint32]bool, len(pl.roleState.TaskAccepted))
	for _, id := range pl.roleState.TaskAccepted {
		set[id] = true
	}
	return set
}

// npcTaskMenuFrames 任务行 m6 帧族（服务行之前——客户端 talk_data_add_save_menu
// 按域分区聚合，任务行归 menu_task_main 置顶）。行文本双键（引导抓包
// 定版）：首键=菜单行文案（接取 CanAcceptMenu_<TID> / 提交
// CompleteMenu_<TID>，TaskText.xml 全覆盖），次键 title_<TID> 作
// TextManager 格式化参数。
func npcTaskMenuFrames(lines []npcTaskLine) ([][]byte, error) {
	frames := make([][]byte, 0, len(lines))
	for _, line := range lines {
		flag, textKey := npcTaskFlagAccept, "CanAcceptMenu_"+strconv.Itoa(line.TaskID)
		if line.Submit {
			flag, textKey = npcTaskFlagSubmit, "CompleteMenu_"+strconv.Itoa(line.TaskID)
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

// handleNPCTaskMark op0x07 任务域 mark 受理（引导抓包实证口径）：
//   - 接取（flag 100）：写 TaskAccepted → 0x29 重放 → 重发菜单
//     （接取行翻提交行）；
//   - 提交（flag 200/201——201 为奖励单选二次确认，随 200 后到达）：
//     移除 → 0x29 重放 → m8×2 关窗（官方提交后关菜单，不刷新——
//     玩家再点 NPC 出下一环任务窗）；
//   - 其余任务域 flag（300 放弃/400 共享/500 进行中/50N·60N 剧情/
//     700 未完成）本阶段留痕，走通用关菜单链。
func (s *Server) handleNPCTaskMark(pl *Player, menu *npcActiveMenu, mark int32) bool {
	flag, taskID, ok := npcTaskMarkSplit(mark)
	if !ok {
		return false
	}
	switch flag {
	case npcTaskFlagAccept:
		if !pl.taskAcceptedContains(uint32(taskID)) {
			pl.roleState.TaskAccepted = append(pl.roleState.TaskAccepted, uint32(taskID))
			s.trace("npc task: 接取 %d client=%d（已接 %d）", taskID, pl.Conn1, len(pl.roleState.TaskAccepted))
		}
		s.replayTaskAccepted(pl)
		s.refreshNPCTalkMenu(pl, menu)
	case npcTaskFlagSubmit, npcTaskFlagSubmitChoice:
		pl.roleState.TaskAccepted = pl.removeAcceptedTask(uint32(taskID))
		s.trace("npc task: 提交 %d（flag=%d）client=%d（余 %d）", taskID, flag, pl.Conn1, len(pl.roleState.TaskAccepted))
		s.replayTaskAccepted(pl)
		// 官方提交应答：m8×2 关窗（无菜单刷新——客户端提交窗为本地
		// 渲染，任务日志由 0x29 记录帧同步）。
		for i := 0; i < 2; i++ {
			if end, err := entergen.CustomMessageFrame(8); err == nil {
				s.sendToClient(pl, end)
			}
		}
		s.clearNPCMenu(pl)
	default:
		s.trace("npc task: mark flag=%d tid=%d client=%d（未受理域）", flag, taskID, pl.Conn1)
	}
	return true
}

// replayTaskAccepted 重放 Task_Accepted 0x29 批量帧（m327 sub6 同源编码，
// 全量幂等——客户端记录全刷）。
func (s *Server) replayTaskAccepted(pl *Player) {
	if !pl.haveRoleState {
		return
	}
	frame, err := entergen.TaskAcceptedBatchFrame(entergen.Context{
		Identity: entergen.DefaultIdentity(),
		State:    pl.roleState,
	})
	if err != nil {
		s.trace("npc task: 重放已接任务失败 client=%d: %v", pl.Conn1, err)
		return
	}
	if frame != nil {
		s.sendToClient(pl, frame)
	}
}

// refreshNPCTalkMenu 重发 m4-7 菜单族（接取行翻提交行/移除后菜单即时
// 刷新；选中帧无需重发——选中态未变）。
func (s *Server) refreshNPCTalkMenu(pl *Player, menu *npcActiveMenu) {
	if menu == nil {
		return
	}
	pl.npcMu.Lock()
	scene := pl.npcScene
	pl.npcMu.Unlock()
	spawn, ok := s.findNPCSpawn(scene, menu.objectID)
	if !ok || spawn.OwnerID != menu.ownerID {
		return
	}
	s.sendNPCTalkMenu(pl, spawn)
}

// taskAcceptedContains 已接判定（nil 接收者安全）。
func (p *Player) taskAcceptedContains(id uint32) bool {
	for _, v := range p.roleState.TaskAccepted {
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

// npcTaskCatalogOf 数据根解析 + 快照装载薄封装（trace 一次缺档）。
func (s *Server) npcTaskCatalogOf() *taskdata.Catalog {
	root := npcfeed.ResolveRoot()
	catalog := taskCatalogForRoot(root)
	if catalog == nil {
		s.trace("npc task: 任务快照缺席（NINEYIN_TASK_SNAPSHOT / %s），任务行不下发", filepath.Join(root, "taskdata", "tasks.json"))
	}
	return catalog
}
