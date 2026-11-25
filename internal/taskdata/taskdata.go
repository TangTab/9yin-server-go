// Package taskdata 装载任务域（数据根带客户端 share/task 与文案档时直读，
// 否则读 tools/datagen npc 生成的 <数据根>/ini/Quest XML）并建立任务目录：
// 任务主表给出任务行与接取/交付 NPC，SubFunction × 各步骤域给出步骤行
// （Task_Record 目标行的来源），SubFunction × StoryTaskList 给出剧情段绑定。
// 任务 ID 与客户端文案键同一体系。装载入口 LoadQuest。
package taskdata

import (
	"regexp"
	"sort"
	"strings"
)

// 任务线号与引导链（task_define.lua task_line_*）。
const (
	// LineGuide 引导线（accept_rec_line）：新手引导链 1340→…→1597/1648。
	LineGuide = 9
	// GuideSearchID 新手引导链的 SearchID（1340~1349、1597 同属此链；官服
	// 进场约 6 秒后自动接取链首 1340）。
	GuideSearchID = "s980301"
)

// 步骤子类型（Task_Record.task_rec_subtype；task_define.lua subtype_*）。
const (
	StepHunter   = 1 // 杀怪
	StepCollect  = 2 // 采集
	StepInteract = 3 // 交互（对话）
	StepConvoy   = 4 // 护送
	StepQuestion = 5 // 问答
	StepStory    = 6 // 剧情
	StepUseItem  = 7 // 用物
	StepSpecial  = 8 // 特殊（服务端条件，成为当前步即完成）
	StepUnmapped = 0 // 无子类型常量（choose 等）——不出目标行
)

// Step 任务的一步：客户端 Task_Record 目标行的来源（任务追踪栏按步骤行
// 显示目标文案与进度 curnum/maxnum）。字段对应 task_define.lua 的
// task_rec_* 列；官服引导抓包杀怪/交互/剧情/特殊四类行逐列定版，其余
// 类型按同构类推。
type Step struct {
	Subtype   int    // 子类型（StepHunter…StepSpecial）
	Index     int    // 步骤行 ID（剧情＝剧情组 ID；特殊＝SpecialType）
	Order     int    // 步骤序（1 起，SubFunction.Order）
	MaxNum    int    // 目标次数（杀怪 Count／采集 Count／用物 ProCount；其余 1）
	Key       string // NPC／怪物／物品配置 ID
	Scene     string // 交互步＝DialogID 键；其余空
	NullScene bool   // scene 列是 0 长度串（杀怪类）；其余为空串或 DialogID 键
	TrackInfo string // 目标文案键（hunterinfo_…／interactinfo_…／storyinfo_…）
	OkMenu    string // 交互步确认行文案键（InteractiveList.OkMenu，okmenu_<步骤号>）
	HasLink   bool   // 带超链接串 "<任务号>_<子类型>_<Key>"（无则空串）
	// 步骤奖励／扣除（InteractiveList.GetItems / LostItems；"物品-数量|…"
	// 或 "物品;物品"，见 ParseItemSpec）。交互步记满时发放。
	GetItems  string
	LostItems string
}

// Task 是目录里的一个任务（任务主表行）。
type Task struct {
	ID    int
	Title string // 客户端显示文本（构建时按 TitleId 解析）；没有文案的行不进目录

	// 主表列——Task_Accepted 已接记录行的来源（task_define.lua
	// accept_rec_* 列；除场景/线号外都是客户端文案键，不是文本）。
	Line           int // 任务线（accept_rec_line）
	SearchID       string
	TitleKey       string // TitleId
	TargetKey      string // TaskTargetId
	ContextKey     string // ContextId
	AcceptNpc      string // 接取 NPC
	SubmitNpc      string // 交付 NPC
	AcceptScene    int    // 接取场景
	SubmitScene    int    // 交付场景
	SubmitNpcEx    string // SubmitNpcEx 键
	CompleteDialog string // CompleteDialogId（"complete_<ID>;submit_<ID>"）
	Prize          string // 奖励串（Prize 域同号行；accept_rec_prize_base）
	Steps          []Step // 步骤行（nil＝无步骤数据）
	AcceptMovie    int    // AcceptMovie：接取后客户端播放的引导影像号（官服 m104 [0, 影像号, 4]；0＝无）

	// IsGuide 新手引导链任务（SearchID＝GuideSearchID）。
	IsGuide bool

	// AcceptMenuKey 接取行文案键——空＝客户端没有接取文案，官服不发 100
	// 接取行（多样本抓包实证 `100081264 CanAcceptMenu_81264 title_81264`
	// 只出现在有该键的任务上），只能服务端自动接取：链根任务进场景/
	// 首谈即接、前序任务提交后按 AutoAcceptID 链式接取。
	AcceptMenuKey string
	// ChainRoot 链根任务：无前置任务、无接取文案，且不是任何任务的
	// AutoAcceptID 目标——服务端自动接取（官服引导链首环 1340 在进场
	// 约 15 秒后自动接取、车夫的 1648 进城即接；本服另保留首谈即接）。
	ChainRoot bool
	// AutoAcceptID 本任务提交后服务端自动接取的任务。
	AutoAcceptID int
	// AutoSubmit 目标达成即自动提交（剧情任务＝剧情走完）。
	AutoSubmit bool
}

// Binding 记录一条 NPC↔任务关联。Source 标记数据来源：主表的接取/提交
// NPC 为 "quest-accept" / "quest-submit"；剧情段为 "storytasklist"。
// storytasklist 的**每行即剧情的一段**
// （同 StoryID 多行按行序为段序——引导抓包 story_482_1..5 与源表
// 5 行互证）：StoryID=剧情行 ID；ContextID/TitleID=该段窗的 m5 标题键
// （story_<sid>_N）与续行文本键（storymenu_<sid>_N）；LeaveTextID=离行
// 文本键（9607）。键本身，非文本。
type Binding struct {
	TaskID      int
	NPCConfigID string
	SceneID     string
	StoryID     string
	ContextID   string
	TitleID     string
	LeaveTextID string
	Source      string
}

// Catalog 是去重并索引后的任务目录。零值与 nil 接收者安全，
// 查询返回空结果（与 npcfunc.Registry 的约定一致）。
type Catalog struct {
	tasks    map[int]Task
	bindings []Binding
	byNPC    map[string][]Binding
	byTask   map[int][]Binding
}

// NewCatalog 组装目录：绑定按（规范化后的 NPC 配置 ID, 任务 ID,
// 剧情行 ID, 段上下文键）去重——同一剧情的多段行（storytasklist 合法
// 形态，story_<sid>_N）全部保留、行序即段序，同一 NPC 的绑定按任务
// ID 升序（稳定排序保段序）。非法条目（非正任务 ID、空 NPC ID）丢弃。
func NewCatalog(tasks map[int]Task, bindings []Binding) *Catalog {
	c := &Catalog{
		tasks:    make(map[int]Task, len(tasks)),
		bindings: make([]Binding, 0, len(bindings)),
		byNPC:    make(map[string][]Binding),
		byTask:   make(map[int][]Binding),
	}
	for id, task := range tasks {
		if id <= 0 {
			continue
		}
		task.ID = id
		c.tasks[id] = task
	}
	seen := make(map[Binding]bool, len(bindings))
	for _, binding := range bindings {
		binding.NPCConfigID = strings.TrimSpace(binding.NPCConfigID)
		binding.StoryID = strings.TrimSpace(binding.StoryID)
		binding.ContextID = strings.TrimSpace(binding.ContextID)
		if binding.TaskID <= 0 || binding.NPCConfigID == "" {
			continue
		}
		key := Binding{TaskID: binding.TaskID, NPCConfigID: normalize(binding.NPCConfigID),
			StoryID: binding.StoryID, ContextID: binding.ContextID}
		if seen[key] {
			continue
		}
		seen[key] = true
		c.bindings = append(c.bindings, binding)
		c.byNPC[key.NPCConfigID] = append(c.byNPC[key.NPCConfigID], binding)
		c.byTask[binding.TaskID] = append(c.byTask[binding.TaskID], binding)
	}
	for _, list := range c.byNPC {
		sort.SliceStable(list, func(i, j int) bool { return list[i].TaskID < list[j].TaskID })
	}
	return c
}

func (c *Catalog) TaskByID(id int) (Task, bool) {
	if c == nil {
		return Task{}, false
	}
	task, ok := c.tasks[id]
	return task, ok
}

// StepsForTask 任务的步骤行（步骤序升序）；无步骤数据或目录里没有该
// 任务返回 nil。
func (c *Catalog) StepsForTask(taskID int) []Step {
	if c == nil {
		return nil
	}
	return c.tasks[taskID].Steps
}

// BindingsForNPC 返回该 NPC（按配置 ID，大小写不敏感）的全部绑定，
// 按任务 ID 升序。
func (c *Catalog) BindingsForNPC(configID string) []Binding {
	if c == nil {
		return nil
	}
	source := c.byNPC[normalize(configID)]
	return append([]Binding(nil), source...)
}

// TasksForNPC 返回该 NPC 绑定的任务，按任务 ID 升序。
// 只有绑定、目录里没有该任务的行不计入（见 BindingsForNPC）；同一任务多条
// 剧情行只返回一次。
func (c *Catalog) TasksForNPC(configID string) []Task {
	if c == nil {
		return nil
	}
	bindings := c.byNPC[normalize(configID)]
	tasks := make([]Task, 0, len(bindings))
	seen := make(map[int]bool, len(bindings))
	for _, binding := range bindings {
		if seen[binding.TaskID] {
			continue
		}
		if task, ok := c.tasks[binding.TaskID]; ok {
			tasks = append(tasks, task)
			seen[binding.TaskID] = true
		}
	}
	return tasks
}

// StoryBindingFor 返回该 NPC 该任务的剧情绑定（storytasklist 来源、
// 首个带 StoryID 的行；无剧情行返回 ok=false）。
func (c *Catalog) StoryBindingFor(configID string, taskID int) (Binding, bool) {
	if c == nil {
		return Binding{}, false
	}
	for _, binding := range c.byNPC[normalize(configID)] {
		if binding.TaskID == taskID && binding.StoryID != "" {
			return binding, true
		}
	}
	return Binding{}, false
}

// StoryBindingsFor 返回该 NPC 该任务的全部剧情段（行序即段序；
// 无剧情行返回空）。
func (c *Catalog) StoryBindingsFor(configID string, taskID int) []Binding {
	if c == nil {
		return nil
	}
	var segments []Binding
	for _, binding := range c.byNPC[normalize(configID)] {
		if binding.TaskID == taskID && binding.StoryID != "" {
			segments = append(segments, binding)
		}
	}
	return segments
}

// ChainRootsFor 可由服务端自动接取的链根任务，按任务 ID 升序：无前置、
// 无接取文案、不是链式接取目标（有接取文案的任务走对话行由玩家接取）。
// guideOnly 只取新手引导链。nil 接收者安全。
func (c *Catalog) ChainRootsFor(guideOnly bool) []Task {
	if c == nil {
		return nil
	}
	tasks := make([]Task, 0, 16)
	for _, task := range c.tasks {
		if !task.ChainRoot || task.AcceptMenuKey != "" || (guideOnly && !task.IsGuide) {
			continue
		}
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks
}

// NPCsForTask 返回绑定到该任务的 NPC 配置 ID（保留原文大小写）。
func (c *Catalog) NPCsForTask(id int) []string {
	if c == nil {
		return nil
	}
	bindings := c.byTask[id]
	npcs := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		npcs = append(npcs, binding.NPCConfigID)
	}
	return npcs
}

// Count 返回目录里的任务数。
func (c *Catalog) Count() int {
	if c == nil {
		return 0
	}
	return len(c.tasks)
}

// BindingCount 返回去重后的绑定数。
func (c *Catalog) BindingCount() int {
	if c == nil {
		return 0
	}
	return len(c.bindings)
}

// BoundNPCCount 返回有任务绑定的 NPC 数。
func (c *Catalog) BoundNPCCount() int {
	if c == nil {
		return 0
	}
	return len(c.byNPC)
}

var (
	richTextTag   = regexp.MustCompile(`<[^>]*>`)
	whitespaceRun = regexp.MustCompile(`\s+`)
)

// StripRichText 去除客户端富文本标记（<font …>、</font>、<br> 等）并
// 压缩空白，供日志与脚本展示；原文仍保留在 Task 字段里。
func StripRichText(text string) string {
	return strings.TrimSpace(whitespaceRun.ReplaceAllString(richTextTag.ReplaceAllString(text, " "), " "))
}

// normalize 与 npcfunc 相同：NPC 配置 ID 大小写不敏感。
func normalize(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
