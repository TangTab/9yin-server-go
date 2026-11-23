// text.go 任务文案提取：modern/text/chineses 的 stringtask_normal/ 与
// stringtask_special/（key=value，UTF-8 BOM，值含富文本标签）→
// testdata/ini/Text/TaskText.xml（服务端方言，ID=串键）。
// 任务表引用的 title_<id>/context_<id>/CanAcceptMenu_<id>/target_<id>/
// story_<id>_<n> 等串键都在这两个目录里（textfiles.ini 只登记单文件
// 域，任务串按档拆目录不在其中）。
//
// npcTalkFiles NPC 对话框文案散档（text/chineses 根级单文件，key=value
// 同方言）→ NpcTalk.xml：m5 对话标题键 task_<ConfigID>（form_talk_data
// on_addtitle 消费）、task_menu_accept_/complete_/submit_<TID> 菜单行
// 键（form_talk_movie show_accept_form 回落链）、task_confirm_ 交任务
// 确认语、randadv_/task_npc_ 闲聊语。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// convertTaskText 扫描两个任务串目录，合并为单档 XML。
// 返回（行数, 键冲突数, 告警）。
func convertTaskText(textRoot, textOut string) (int, int, []string, error) {
	dirs := []string{"stringtask_normal", "stringtask_special"}
	entries := map[string]string{} // 键 → 值（首见胜）
	conflicts := 0
	var warnings []string
	files := 0
	for _, dir := range dirs {
		full := filepath.Join(textRoot, dir)
		list, err := os.ReadDir(full)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("串目录缺失 %s: %v", dir, err))
			continue
		}
		var names []string
		for _, e := range list {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".idres") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			files++
			if err := readIDres(filepath.Join(full, name), entries, &conflicts); err != nil {
				warnings = append(warnings, fmt.Sprintf("跳过 %s/%s: %v", dir, name, err))
			}
		}
	}
	if len(entries) == 0 {
		return 0, conflicts, warnings, fmt.Errorf("npcgen: 任务串目录无内容 %s", textRoot)
	}
	w := newObjectWriter()
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		w.AddProperty([][2]string{{"ID", k}, {"Text", entries[k]}})
	}
	outPath := filepath.Join(textOut, "TaskText.xml")
	if err := writeGenerated(outPath, w.Bytes()); err != nil {
		return 0, conflicts, warnings, err
	}
	_ = files
	return len(entries), conflicts, warnings, nil
}

// npcTalkFiles NPC 对话框文案散档清单（text/chineses 根级，按重要度序；
// stringtalk.idres 与 string_taskmenu.idres 的 task_menu_* 键与
// stringtask_normal 少量重叠——首见胜，先散档后不覆盖）。
var npcTalkFiles = []string{"string_npctalk.idres", "string_taskmenu.idres", "stringtalk.idres"}

// convertNpcTalk NPC 对话框文案散档 → NpcTalk.xml。
// 返回（行数, 键冲突数, 告警）。
func convertNpcTalk(textRoot, textOut string) (int, int, []string, error) {
	entries := map[string]string{} // 键 → 值（首见胜）
	conflicts := 0
	var warnings []string
	read := 0
	for _, name := range npcTalkFiles {
		path := filepath.Join(textRoot, name)
		if _, err := os.Stat(path); err != nil {
			warnings = append(warnings, fmt.Sprintf("对话散档缺失 %s: %v", name, err))
			continue
		}
		read++
		if err := readIDres(path, entries, &conflicts); err != nil {
			warnings = append(warnings, fmt.Sprintf("跳过 %s: %v", name, err))
		}
	}
	if len(entries) == 0 {
		return 0, conflicts, warnings, fmt.Errorf("npcgen: NPC 对话散档无内容 %s", textRoot)
	}
	w := newObjectWriter()
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		w.AddProperty([][2]string{{"ID", k}, {"Text", entries[k]}})
	}
	outPath := filepath.Join(textOut, "NpcTalk.xml")
	if err := writeGenerated(outPath, w.Bytes()); err != nil {
		return 0, conflicts, warnings, err
	}
	_ = read
	return len(entries), conflicts, warnings, nil
}

// readIDres 解析单个 .idres（key=value 行；UTF-8 BOM / GBK 自适应；行内
// 多个 '=' 时首个为界）。首见胜，重复键计数。
func readIDres(path string, into map[string]string, conflicts *int) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data, err := toUTF8(raw)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		if key == "" {
			continue
		}
		if _, exists := into[key]; exists {
			*conflicts++
			continue
		}
		into[key] = value
	}
	return nil
}
