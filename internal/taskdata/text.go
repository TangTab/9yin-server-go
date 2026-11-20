package taskdata

import (
	"sort"
	"strconv"
	"strings"
)

// textPrefixes 把 stringtask 文本键前缀映射到 Task 字段，按长度降序
// 尝试以保证最长前缀优先。该白名单来自 share/task/pre_file.txt 中按
// TID 组织、落在 stringtask_normal 的条目；stringtask_special 的 FID
// 前缀（story_/Dialog_/convoyinfo_ 等子任务文本）待接取阶段再引入。
var textPrefixes = func() []struct{ prefix, field string } {
	entries := []struct{ prefix, field string }{
		{"title_", "Title"},
		{"target_", "Target"},
		{"context_", "Context"},
		{"menu_", "Menu"},
		{"CanAcceptMenu_", "CanAcceptMenu"},
		{"CompleteMenu_", "CompleteMenu"},
		{"UncompleteMenu_", "UncompleteMenu"},
		{"complete_", "Complete"},
		{"next_", "Next"},
	}
	sort.Slice(entries, func(i, j int) bool { return len(entries[i].prefix) > len(entries[j].prefix) })
	return entries
}()

// splitTextKey 把形如 "title_75730" 的键拆成（字段, 任务 ID）。
// 不在白名单内或 TID 非正整数的键返回 ok=false。
func splitTextKey(key string) (field string, taskID int, ok bool) {
	trimmed := strings.TrimSpace(key)
	for _, entry := range textPrefixes {
		rest, found := strings.CutPrefix(trimmed, entry.prefix)
		if !found {
			continue
		}
		id, err := strconv.Atoi(rest)
		if err != nil || id <= 0 {
			return "", 0, false
		}
		return entry.field, id, true
	}
	return "", 0, false
}

// ApplyTaskText 把一条已解析的任务文本写进 Task 对应字段。
// 未知字段返回 false。
func ApplyTaskText(task *Task, field, text string) bool {
	switch field {
	case "Title":
		task.Title = text
	case "Target":
		task.Target = text
	case "Context":
		task.Context = text
	case "Menu":
		task.Menu = text
	case "CanAcceptMenu":
		task.CanAcceptMenu = text
	case "CompleteMenu":
		task.CompleteMenu = text
	case "UncompleteMenu":
		task.UncompleteMenu = text
	case "Complete":
		task.Complete = text
	case "Next":
		task.Next = text
	default:
		return false
	}
	return true
}

// ParseTaskText 扫描一份 stringtask idres（UTF-8 可带 BOM，key=value，
// // 为注释），把白名单前缀的条目按（字段, 任务 ID, 文本）回调 emit。
// 无法识别的行（其他前缀、非数字 TID、注释、空行）一律跳过——这些表
// 里混有大量与任务无关的键。
func ParseTaskText(raw []byte, emit func(field string, taskID int, text string)) {
	content := strings.TrimPrefix(string(raw), "\uFEFF")
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		field, taskID, ok := splitTextKey(key)
		if !ok {
			continue
		}
		emit(field, taskID, value)
	}
}

// CollectTaskText 把一份 idres 的解析结果累积进任务表；同一键后写
// 覆盖先写（idres 的覆盖语义）。
func CollectTaskText(raw []byte, tasks map[int]*Task) {
	ParseTaskText(raw, func(field string, taskID int, text string) {
		task := tasks[taskID]
		if task == nil {
			task = &Task{ID: taskID}
			tasks[taskID] = task
		}
		ApplyTaskText(task, field, text)
	})
}
