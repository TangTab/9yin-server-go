// tasks.go BGserverCJ 任务域与战斗 NPC 模板转换。
//
// 源（-tasks 指向 task 根，缺省仓库兄弟目录 九阴真经ol\task）：
//   - Task/<域>/<前缀>_*.txt   任务各域表（GBK，2 行头：类型+列键；按
//     main/copy/sch 分档，同域合并去重）——HuntList 杀怪、CollectList 采集、
//     Convoy 押镖、Prize 奖励、StoryTaskList 剧情等 12 域
//   - Task/xxx_type_ID.txt     根级散表
//   - task002.txt              主表旧快照（UTF-8 BOM，37 列；与 Task 树
//     ID 互补，单独立档不合并）
//   - task_config.txt          字段字典（含中文枚举说明）
//   - CharEffect.txt           人物特效表（列键为中文，XML 合法保留）
//   - npctemplate.tab          战斗 NPC 模板（6273 行，1 行头）
//
// 这些表本身就是服务端词汇（BGserverCJ 引擎侧），与黑金档无对应关系，
// 不做跨词汇映射：列名原样透传，空 cell 省略，ID 置首。任务表的 NpcId
// 引用（如 monv040100）解析到本工具生成的 NpcConfig 树。
package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// loadTabTable 读 2 行头（类型+列键）或 1 行头（仅列键）的 TAB 表。
// 编码自适应（GBK / UTF-8 BOM）；尾部空列键裁剪；短行补齐；全空行跳过。
func loadTabTable(path string, headerRows int) (*npcTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data, err := toUTF8(raw)
	if err != nil {
		return nil, fmt.Errorf("解码: %w", err)
	}
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = false

	readRecord := func() ([]string, error) {
		record, err := reader.Read()
		if err != nil {
			return nil, err
		}
		return record, nil
	}
	source := filepath.ToSlash(mustRel(path))
	t := &npcTable{Source: source}
	if headerRows >= 1 {
		types, err := readRecord()
		if err != nil {
			return nil, fmt.Errorf("读类型行: %w", err)
		}
		if headerRows >= 2 {
			t.Types = types
		} else {
			t.Columns = types // 1 行头：首行即列键
		}
	}
	if headerRows >= 2 {
		columns, err := readRecord()
		if err != nil {
			return nil, fmt.Errorf("读列键行: %w", err)
		}
		t.Columns = columns
	}
	// 尾部空列键裁剪（HuntList 等表头尾带空 TAB 段）。
	for len(t.Columns) > 0 && strings.TrimSpace(t.Columns[len(t.Columns)-1]) == "" {
		t.Columns = t.Columns[:len(t.Columns)-1]
	}
	if len(t.Columns) == 0 {
		return nil, errors.New("无列键")
	}
	width := len(t.Columns)
	if len(t.Types) > width {
		t.Types = t.Types[:width]
	}
	t.Comments = make([]string, width)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读数据行: %w", err)
		}
		allEmpty := true
		for _, raw := range record {
			if strings.TrimSpace(raw) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}
		if len(record) < width {
			record = append(record, make([]string, width-len(record))...)
		} else if len(record) > width {
			record = record[:width]
		}
		t.Rows = append(t.Rows, record)
	}
	return t, nil
}

// genPassThroughTable 透传表 → XML（列名原样，ID 置首，空 cell 省略）。
func genPassThroughTable(t *npcTable) []byte {
	first := t.firstIndex()
	idIdx, hasID := first["ID"]
	w := newObjectWriter()
	for _, row := range t.Rows {
		if !hasID || row[idIdx] == "" {
			continue
		}
		attrs := [][2]string{{"ID", row[idIdx]}}
		for i, name := range t.Columns {
			if first[name] != i || name == "ID" || name == "" {
				continue
			}
			if row[i] != "" {
				attrs = append(attrs, [2]string{name, row[i]})
			}
		}
		w.AddProperty(attrs)
	}
	return w.Bytes()
}

// domainReport 一个任务域的转换统计。
type domainReport struct {
	Name      string   `json:"name"`
	Files     int      `json:"files"`
	Rows      int      `json:"rows"`
	RepeatIDs int      `json:"repeat_ids"` // 同 ID 多行数（剧情步骤/分档变体，全部保留不去重）
	Notes     []string `json:"notes,omitempty"`
}

// sortDomainFiles 域内档序：_main_ 主档优先（主档是权威版本），其余按
// 文件名字典序。
func sortDomainFiles(paths []string) {
	score := func(p string) int {
		if strings.Contains(filepath.Base(p), "_main_") {
			return 0
		}
		return 1
	}
	sort.SliceStable(paths, func(i, j int) bool { return score(paths[i]) < score(paths[j]) })
}

// mergeDomainFiles 同域分档合并：主档序在前，全部行保留——同 ID 多行是
// 合法形态（StoryTaskList 一条任务多段剧情、SubFunction 多段子功能），
// 按 ID 去重会吞步骤。返回合并表与同 ID 多行计数（仅统计）。
func mergeDomainFiles(paths []string) (*npcTable, []string, int, error) {
	merged := &npcTable{}
	seen := map[string]int{}
	var warnings []string
	repeats := 0
	for _, p := range paths {
		t, err := loadTabTable(p, 2)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("跳过 %s: %v", filepath.Base(p), err))
			continue
		}
		if len(merged.Columns) == 0 {
			merged.Columns = t.Columns
			merged.Types = t.Types
			merged.Source = t.Source
		}
		first := t.firstIndex()
		idIdx, hasID := first["ID"]
		for _, row := range t.Rows {
			merged.Rows = append(merged.Rows, row)
			if hasID && idIdx < len(row) && row[idIdx] != "" {
				seen[row[idIdx]]++
				if seen[row[idIdx]] > 1 {
					repeats++
				}
			}
		}
	}
	if len(merged.Columns) == 0 {
		return nil, warnings, repeats, errors.New("域内无可解析表")
	}
	return merged, warnings, repeats, nil
}

// convertQuest 任务域转换主入口：Task/ 子目录各并一档，根级散表单档。
// 返回域统计与全部输出（路径→期望行数，供 verify）。
func convertQuest(tasksDir, questOut string) ([]domainReport, map[string]int, []string, error) {
	domains := []domainReport{}
	written := map[string]int{}
	var warnings []string

	taskRoot := filepath.Join(tasksDir, "Task")
	entries, err := os.ReadDir(taskRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("npcgen: 读 %s: %w", taskRoot, err)
	}
	var dirs, loose []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		} else if strings.EqualFold(filepath.Ext(e.Name()), ".txt") {
			loose = append(loose, e.Name())
		} else {
			warnings = append(warnings, fmt.Sprintf("Task 根跳过非 txt: %s", e.Name()))
		}
	}
	sort.Strings(dirs)
	sort.Strings(loose)

	for _, dir := range dirs {
		var files []string
		sub := filepath.Join(taskRoot, dir)
		list, err := os.ReadDir(sub)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("域 %s: %v", dir, err))
			continue
		}
		for _, e := range list {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".txt") {
				files = append(files, filepath.Join(sub, e.Name()))
			}
		}
		sort.Strings(files)
		if len(files) == 0 {
			continue
		}
		sortDomainFiles(files)
		merged, dwarns, repeats, err := mergeDomainFiles(files)
		warnings = append(warnings, dwarns...)
		if err != nil {
			domains = append(domains, domainReport{Name: dir, Notes: []string{err.Error()}})
			continue
		}
		outPath := filepath.Join(questOut, dir, dir+".xml")
		if err := writeGenerated(outPath, genPassThroughTable(merged)); err != nil {
			return nil, nil, nil, err
		}
		written[outPath] = countIDRows(merged)
		domains = append(domains, domainReport{Name: dir, Files: len(files), Rows: written[outPath], RepeatIDs: repeats})
	}

	// Task/ 根级散表（xxx_type_ID.txt 等）。
	for _, name := range loose {
		t, err := loadTabTable(filepath.Join(taskRoot, name), 2)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("跳过 %s: %v", name, err))
			continue
		}
		outPath := filepath.Join(questOut, strings.TrimSuffix(name, filepath.Ext(name))+".xml")
		if err := writeGenerated(outPath, genPassThroughTable(t)); err != nil {
			return nil, nil, nil, err
		}
		written[outPath] = countIDRows(t)
		domains = append(domains, domainReport{Name: strings.TrimSuffix(name, filepath.Ext(name)), Files: 1, Rows: written[outPath]})
	}

	// task 根级：task002.txt（主表旧快照，与 Task 树互补不合并）、
	// task_config.txt（字段字典）、CharEffect.txt（特效表）。
	for name, headerRows := range map[string]int{"task002.txt": 2, "task_config.txt": 2, "CharEffect.txt": 2} {
		path := filepath.Join(tasksDir, name)
		if _, err := os.Stat(path); err != nil {
			warnings = append(warnings, fmt.Sprintf("根级表缺失: %s", name))
			continue
		}
		t, err := loadTabTable(path, headerRows)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("跳过 %s: %v", name, err))
			continue
		}
		outPath := filepath.Join(questOut, strings.TrimSuffix(name, filepath.Ext(name))+".xml")
		if err := writeGenerated(outPath, genPassThroughTable(t)); err != nil {
			return nil, nil, nil, err
		}
		written[outPath] = countIDRows(t)
		domains = append(domains, domainReport{Name: strings.TrimSuffix(name, filepath.Ext(name)), Files: 1, Rows: written[outPath]})
	}
	return domains, written, warnings, nil
}

// convertNpcTemplate 战斗 NPC 模板（1 行头 .tab）→ ini/Npc/NpcTemplate.xml。
//
// src 三态：npctemplate.tab 文件本身 / 其所在目录 / glob 模式
// （npctemplate*.tab）。多档合并：npctemplate.tab 主档序前，其余按文件名
// 字典序补充——同 ID 首档胜出（npctemplate2.tab 等空壳/坏档列头解析失败
// 整档 warning 跳过，不废主档）。
func convertNpcTemplate(src, npcOut string) (int, string, []string, error) {
	files, err := resolveNpcTemplateFiles(src)
	if err != nil {
		return 0, "", nil, err
	}
	merged := &npcTable{}
	var warnings []string
	seen := map[string]struct{}{}
	for _, path := range files {
		t, loadErr := loadTabTable(path, 1)
		if loadErr != nil {
			warnings = append(warnings, fmt.Sprintf("战斗模板档跳过 %s: %v", filepath.Base(path), loadErr))
			continue
		}
		if len(merged.Columns) == 0 {
			merged.Columns, merged.Types, merged.Source = t.Columns, t.Types, t.Source
		}
		first := t.firstIndex()
		idIdx, hasID := first["ID"]
		for _, row := range t.Rows {
			id := ""
			if hasID && idIdx < len(row) {
				id = row[idIdx]
			}
			if id != "" {
				if _, dup := seen[id]; dup {
					continue
				}
				seen[id] = struct{}{}
			}
			merged.Rows = append(merged.Rows, row)
		}
	}
	if len(merged.Rows) == 0 {
		return 0, "", warnings, os.ErrNotExist
	}
	outPath := filepath.Join(npcOut, "NpcTemplate.xml")
	if err := writeGenerated(outPath, genPassThroughTable(merged)); err != nil {
		return 0, "", warnings, err
	}
	return countIDRows(merged), outPath, warnings, nil
}

// resolveNpcTemplateFiles 战斗模板源三态解析：文件直用；目录取
// npctemplate*.tab（npctemplate.tab 精确名优先，其余字典序）；含路径
// 通配符按 glob 展开。
func resolveNpcTemplateFiles(src string) ([]string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil, os.ErrNotExist
	}
	if strings.ContainsAny(filepath.Base(src), "*?[") {
		matches, err := filepath.Glob(src)
		if err != nil {
			return nil, fmt.Errorf("npcgen: glob %s: %w", src, err)
		}
		if len(matches) == 0 {
			return nil, os.ErrNotExist
		}
		sortNpcTemplateFiles(matches)
		return matches, nil
	}
	if info, err := os.Stat(src); err == nil && !info.IsDir() {
		return []string{src}, nil
	}
	dir := src
	if info, err := os.Stat(dir); err != nil {
		return nil, os.ErrNotExist
	} else if !info.IsDir() {
		return nil, fmt.Errorf("npcgen: 战斗模板源 %s 非目录/文件", src)
	}
	matches, err := filepath.Glob(filepath.Join(dir, "npctemplate*.tab"))
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, os.ErrNotExist
	}
	sortNpcTemplateFiles(matches)
	return matches, nil
}

// sortNpcTemplateFiles npctemplate.tab 精确名主档序前，其余按字典序
// （npctemplate2.tab 等补充档在后——同 ID 首档胜出的合并序）。
func sortNpcTemplateFiles(paths []string) {
	score := func(p string) int {
		if strings.EqualFold(filepath.Base(p), "npctemplate.tab") {
			return 0
		}
		return 1
	}
	sort.SliceStable(paths, func(i, j int) bool { return score(paths[i]) < score(paths[j]) })
}

// countIDRows 有 ID 的行数（与 genPassThroughTable 输出一致）。
func countIDRows(t *npcTable) int {
	first := t.firstIndex()
	idIdx, hasID := first["ID"]
	n := 0
	for _, row := range t.Rows {
		if hasID && idIdx < len(row) && row[idIdx] != "" {
			n++
		}
	}
	return n
}

// mustRel 相对仓库根的展示路径（失败回退绝对路径）。
func mustRel(path string) string {
	if rel, err := filepath.Rel(repoRoot(), path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return path
}
