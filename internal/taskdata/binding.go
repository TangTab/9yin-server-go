package taskdata

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// LoadStoryTaskListFromBytes 解析一份 storytasklist_*.txt（TAB 分隔，
// 两行头：类型行 + 列名行；列 ID/NpcID/contextId/titleId/LeaveTextID/
// LeaveResult/TrackInfoID/SceneID/TaskID）。与 npcfunc 相同，这里有意
// 不做 GBK 转码：消费的列（NpcID/SceneID/TaskID）均为 ASCII。
// 同一（NPC, 任务）的多条对话行由 Catalog 统一去重。
func LoadStoryTaskListFromBytes(raw []byte, source string) ([]Binding, error) {
	var columns map[string]int
	var bindings []Binding
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if columns == nil {
			if strings.TrimSpace(fields[0]) != "ID" {
				continue // 类型行（STRING…）或其他前言
			}
			columns = make(map[string]int, len(fields))
			for i, name := range fields {
				columns[strings.TrimSpace(name)] = i
			}
			for _, required := range []string{"NpcID", "TaskID"} {
				if _, ok := columns[required]; !ok {
					return nil, fmt.Errorf("taskdata: %s storytasklist header lacks %s column", source, required)
				}
			}
			continue
		}
		npcID := storyColumn(fields, columns, "NpcID")
		taskText := storyColumn(fields, columns, "TaskID")
		if npcID == "" || taskText == "" {
			continue
		}
		taskID, err := strconv.Atoi(taskText)
		if err != nil || taskID <= 0 {
			continue
		}
		bindings = append(bindings, Binding{
			TaskID:      taskID,
			NPCConfigID: npcID,
			SceneID:     storyColumn(fields, columns, "SceneID"),
			Source:      "storytasklist",
		})
	}
	if columns == nil {
		return nil, fmt.Errorf("taskdata: %s has no storytasklist column row", source)
	}
	return bindings, nil
}

func storyColumn(fields []string, columns map[string]int, name string) string {
	index, ok := columns[name]
	if !ok || index >= len(fields) {
		return ""
	}
	return strings.TrimSpace(fields[index])
}

// parseIniSections 是任务表专用的极简 INI 扫描：[节名] 下的 key=value
// 收进 map。注释行（//、;、#）、节外行与无法切分的行跳过；值的解释
// 留给调用方。编码同上——只消费 ASCII。
func parseIniSections(raw []byte) map[string]map[string]string {
	sections := make(map[string]map[string]string)
	current := ""
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(line[1 : len(line)-1])
			if current != "" && sections[current] == nil {
				sections[current] = make(map[string]string)
			}
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || current == "" {
			continue
		}
		sections[current][strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return sections
}

// LoadTaskNPCIniFromBytes 解析 [TaskNpc] 段的
// taskid=npcid,conditionid 绑定表（如 share/task/qianjiu_tasknpc.ini）。
func LoadTaskNPCIniFromBytes(raw []byte, source string) ([]Binding, error) {
	section := parseIniSections(raw)["TaskNpc"]
	if section == nil {
		return nil, fmt.Errorf("taskdata: %s has no [TaskNpc] section", source)
	}
	bindings := make([]Binding, 0, len(section))
	for key, value := range section {
		taskID, err := strconv.Atoi(key)
		if err != nil || taskID <= 0 {
			continue
		}
		npcID, _, _ := strings.Cut(value, ",")
		if npcID = strings.TrimSpace(npcID); npcID == "" {
			continue
		}
		bindings = append(bindings, Binding{TaskID: taskID, NPCConfigID: npcID, Source: "qianjiu"})
	}
	sort.SliceStable(bindings, func(i, j int) bool { return bindings[i].TaskID < bindings[j].TaskID })
	return bindings, nil
}

// LoadDramaCiteFromBytes 解析 task_drama_cite.ini 的 [TID] 段
// （DramaName/Chapter/Round/NextTaskLimit）。节名非正整数的段跳过。
func LoadDramaCiteFromBytes(raw []byte) map[int]DramaCite {
	cites := make(map[int]DramaCite)
	for name, entries := range parseIniSections(raw) {
		id, err := strconv.Atoi(name)
		if err != nil || id <= 0 {
			continue
		}
		cites[id] = DramaCite{
			DramaName:     entries["DramaName"],
			Chapter:       iniInt(entries["Chapter"]),
			Round:         iniInt(entries["Round"]),
			NextTaskLimit: iniInt(entries["NextTaskLimit"]),
		}
	}
	return cites
}

func iniInt(value string) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return number
}
