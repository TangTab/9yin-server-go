// Package taskdata 从客户端资源加载任务文本与 NPC↔任务绑定：
//
//   - 任务文本来自 text/chineses/stringtask_normal/*.idres，按
//     share/task/pre_file.txt 声明的前缀规则以任务 ID（TID）组织；
//   - NPC 绑定来自 share/task/task/storytasklist/*.txt（剧情任务对话表，
//     行含 NpcID/SceneID/TaskID）与 share/task/qianjiu_tasknpc.ini 的
//     [TaskNpc] 段；
//   - 任务链元数据来自 share/task/task/task_drama_cite.ini。
//
// 派生结果经 Snapshot 以版本化 JSON 持久化，服务端启动时可直接读快照，
// 不必每次重扫 500+ 个资源文件。
package taskdata

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Task 是一个任务的客户端文本快照。字段保留客户端富文本原文
// （<font>/<br> 标记），日志与脚本展示用 StripRichText 取纯文本。
type Task struct {
	ID             int    `json:"id"`
	Title          string `json:"title,omitempty"`
	Target         string `json:"target,omitempty"`
	Context        string `json:"context,omitempty"`
	Menu           string `json:"menu,omitempty"`
	CanAcceptMenu  string `json:"can_accept_menu,omitempty"`
	CompleteMenu   string `json:"complete_menu,omitempty"`
	UncompleteMenu string `json:"uncomplete_menu,omitempty"`
	Complete       string `json:"complete,omitempty"`
	Next           string `json:"next,omitempty"`

	// 接取/链进字段（ini/Quest 任务主表行带出；旧 JSON 快照链只有文案，
	// 接取键按 CanAcceptMenu 文案有无折算，其余为零值）。
	//
	// AcceptMenuKey 接取行文案键——空＝客户端没有接取文案，官服不发 100
	// 接取行（多样本抓包实证 `100081264 CanAcceptMenu_81264 title_81264`
	// 只出现在有该键的任务上），只能服务端自动接取：链首任务首谈即接、
	// 前序任务提交后按 AutoAcceptID 链式接取。
	AcceptMenuKey string `json:"accept_menu_key,omitempty"`
	// ChainRoot 链根任务：无前置任务、无接取文案，且不是任何任务的
	// AutoAcceptID 目标——服务端在首次对话时直接接取（官服引导链首环
	// 1340、车夫的 1648 均属此类）。
	ChainRoot bool `json:"chain_root,omitempty"`
	// AutoAcceptID 本任务提交后服务端自动接取的任务。
	AutoAcceptID int `json:"auto_accept_id,omitempty"`
	// AutoSubmit 目标达成即自动提交（剧情任务＝剧情走完）。
	AutoSubmit bool `json:"auto_submit,omitempty"`
}

// Binding 记录一条 NPC↔任务关联。Source 标记数据来源：
// "storytasklist" 或 "qianjiu"。storytasklist 的**每行即剧情的一段**
// （同 StoryID 多行按行序为段序——引导抓包 story_482_1..5 与源表
// 5 行互证）：StoryID=剧情行 ID；ContextID/TitleID=该段窗的 m5 标题键
// （story_<sid>_N）与续行文本键（storymenu_<sid>_N）；LeaveTextID=离行
// 文本键（9607）。键本身，非文本。
type Binding struct {
	TaskID       int    `json:"task_id"`
	NPCConfigID  string `json:"npc_config_id"`
	SceneID      string `json:"scene_id,omitempty"`
	StoryID      string `json:"story_id,omitempty"`
	ContextID    string `json:"context_id,omitempty"`
	TitleID      string `json:"title_id,omitempty"`
	LeaveTextID  string `json:"leave_text_id,omitempty"`
	Source       string `json:"source,omitempty"`
}

// DramaCite 是 task_drama_cite.ini 的任务链条目。
type DramaCite struct {
	DramaName     string `json:"drama_name,omitempty"`
	Chapter       int    `json:"chapter,omitempty"`
	Round         int    `json:"round,omitempty"`
	NextTaskLimit int    `json:"next_task_limit,omitempty"`
}

// Catalog 是去重并索引后的任务目录。零值与 nil 接收者安全，
// 查询返回空结果（与 npcfunc.Registry 的约定一致）。
type Catalog struct {
	tasks      map[int]Task
	bindings   []Binding
	byNPC      map[string][]Binding
	byTask     map[int][]Binding
	dramaCites map[int]DramaCite
}

// NewCatalog 组装目录：绑定按（规范化后的 NPC 配置 ID, 任务 ID,
// 剧情行 ID, 段上下文键）去重——同一剧情的多段行（storytasklist 合法
// 形态，story_<sid>_N）全部保留、行序即段序，同一 NPC 的绑定按任务
// ID 升序（稳定排序保段序）。非法条目（非正任务 ID、空 NPC ID）丢弃。
func NewCatalog(tasks map[int]Task, bindings []Binding, cites map[int]DramaCite) *Catalog {
	c := &Catalog{
		tasks:      make(map[int]Task, len(tasks)),
		bindings:   make([]Binding, 0, len(bindings)),
		byNPC:      make(map[string][]Binding),
		byTask:     make(map[int][]Binding),
		dramaCites: make(map[int]DramaCite, len(cites)),
	}
	for id, task := range tasks {
		if id <= 0 {
			continue
		}
		task.ID = id
		if task.AcceptMenuKey == "" && task.CanAcceptMenu != "" {
			task.AcceptMenuKey = "CanAcceptMenu_" + strconv.Itoa(id)
		}
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
	for id, cite := range cites {
		if id > 0 {
			c.dramaCites[id] = cite
		}
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

// BindingsForNPC 返回该 NPC（按配置 ID，大小写不敏感）的全部绑定，
// 按任务 ID 升序。
func (c *Catalog) BindingsForNPC(configID string) []Binding {
	if c == nil {
		return nil
	}
	source := c.byNPC[normalize(configID)]
	return append([]Binding(nil), source...)
}

// TasksForNPC 返回该 NPC 绑定且已加载到文本的任务，按任务 ID 升序。
// 只有绑定、没有文本的任务不计入（见 BindingsForNPC）；同一任务多条
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

// Count 返回已加载文本的任务数。
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

func (c *Catalog) DramaCiteByID(id int) (DramaCite, bool) {
	if c == nil {
		return DramaCite{}, false
	}
	cite, ok := c.dramaCites[id]
	return cite, ok
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
