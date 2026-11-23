// tasks.go 客户端任务域与战斗 NPC 模板转换。
//
// 源（-in 资源根下的 share/task——客户端自己在用的任务域树）：任务 ID 与
// 客户端文案键（title_<ID>/context_<ID>…）、官服 0x29 记录帧同一体系（引导链
// 1340→1347→…→1648 的 NPC/后继任务逐项对拍吻合）。2026-11-23 起取代
// BGserverCJ 2014 服务端 Task 树——同一 ID 体系的旧快照：其 4156 任务只
// 覆盖本表的 2886 个同号 ID（引导链等 12549 个任务缺席，1270 个 ID 本表
// 已无），同号任务的 NPC/链与步骤行也大面积漂移。
//   - task/<域>/<前缀>_*.txt   15 个任务域表（GBK/UTF-8 自适应，2 行头：
//     类型+列键）。同域各档列头不统一（主表 83 种列头变体）——按列键并集
//     合并、行按列键对齐；列头须齐含域列键签名才归属本域（挡错置档）。
//     目录内 task_index_file.ini 清单已过期（53 档未登记却带 4730 个独有
//     任务），不作文件集依据。
//   - task_config.txt            字段字典（含中文枚举说明）
//   - chareffect.txt             人物特效表（列键为中文，XML 合法保留）
//   - task/xxx_type_id.txt       根级散表
//   - npctemplate*.tab           战斗 NPC 模板（-npctemplate，缺省同目录）
//
// 这些表本身就是服务端词汇（FsGame 引擎侧 ini/Quest 同构），不做跨词汇
// 映射：列名原样透传（2 行头表的列键裁首尾空白），空 cell 省略，ID 置首。
// 主表 Title 列是设计期备注（与客户端文案七成不一致），输出改存 TitleId
// 键解析出的显示文本，解析不到留空——装载链据此把客户端无文案的任务挡在
// 任务行外。任务表的 NpcId/AcceptNpc 引用解析到本工具生成的 NpcConfig 树。
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
// 编码自适应（GBK / UTF-8 BOM）；2 行头表的列键裁首尾空白；尾部空列键裁剪；
// 短行补齐、长行截断（计入 Truncated）；全空行跳过。
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
		for i := range columns {
			columns[i] = strings.TrimSpace(columns[i])
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
			// 行尾空占位列是排版常态；只有多余字段带值才算截断丢数据。
			for _, extra := range record[width:] {
				if strings.TrimSpace(extra) != "" {
					t.Truncated++
					break
				}
			}
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

// questDomain 一个任务域：客户端目录名 → 输出目录名（沿用既有产物命名）。
type questDomain struct {
	Client string   // share/task/task 下的客户端目录名
	Server string   // 输出目录与文件名 Quest/<Server>/<Server>.xml
	Keys   []string // 列键签名：列头齐含才归属本域
}

// questDomains 任务域清单（输出序）。Keys 取该域全部列头变体的公共键——
// 错置档（storytasklist/ 内的主表档、interactivelist/ 内的奖励档）列头
// 不含签名，按此挡在域外。
var questDomains = []questDomain{
	{"task", "Task", []string{"TitleId", "ContextId", "AcceptNpc", "SubmitNpc"}},
	{"storytasklist", "StoryTaskList", []string{"NpcID", "contextId", "titleId", "TaskID"}},
	{"subfunction", "SubFunction", []string{"Type", "TypeID"}},
	{"huntlist", "HuntList", []string{"NpcId", "Count", "TaskID"}},
	{"collectlist", "CollectList", []string{"ItemId", "NpcId"}},
	{"useitemlist", "UseItemList", []string{"ItemID", "ProCount"}},
	{"interactivelist", "InteractiveList", []string{"NpcId", "DialogID"}},
	{"question", "Question", []string{"QuestionNpc", "QuestionID"}},
	{"refreshmonster", "RefreshMonster", []string{"MonsterID"}},
	{"convoy", "Convoy", []string{"NpcId", "PathID"}},
	{"convoypoint", "Convoypoint", []string{"Xpoint", "Zpoint"}},
	{"special", "Special", []string{"SpecialType"}},
	{"choose", "Choose", []string{"ChooseNpc"}},
	{"taskitem", "TaskItem", []string{"ItemId", "IsGive"}},
	{"prize", "Prize", []string{"Faculty", "CapitalType1"}},
}

// questRootTables 任务域根级散表 → 输出文件名（沿用既有产物命名）。
var questRootTables = []struct{ Source, Output string }{
	{"task_config.txt", "task_config.xml"}, // 字段字典
	{"chareffect.txt", "CharEffect.xml"},   // 人物特效表
	{"task/xxx_type_id.txt", "xxx_type_ID.xml"},
}

// domainReport 一个任务域的转换统计。
type domainReport struct {
	Name      string   `json:"name"`   // 输出域名
	Client    string   `json:"client"` // 客户端目录名
	Files     int      `json:"files"`  // 入表档数
	Rows      int      `json:"rows"`
	Columns   int      `json:"columns"`              // 并集列数
	RepeatIDs int      `json:"repeat_ids"`           // 同 ID 多行数（剧情步骤/分档变体，全部保留不去重）
	TitleText int      `json:"title_text,omitempty"` // 主表 Title 解析到客户端文案的行数（仅 task 域）
	Skipped   []string `json:"skipped,omitempty"`    // 未入表档与原因
	Notes     []string `json:"notes,omitempty"`
}

// mergeAudit 一次同域合并的审计面。
type mergeAudit struct {
	Files   int      // 入表档数
	Repeats int      // 同 ID 多行数
	Skipped []string // 未入表档与原因
	Notes   []string // 源表瑕疵（重复列/无名列/截断行）
}

// mergeDomainFiles 同域各档合并为一张表：列取各档列键并集（首见序），行按
// 列键对齐（缺列留空）。同 ID 多行全部保留不去重——是合法形态（StoryTaskList
// 一条任务多段剧情、SubFunction 多段子功能）。返回合并表（nil＝域内无
// 有效表）。
func mergeDomainFiles(domain questDomain, paths []string) (*npcTable, *mergeAudit) {
	audit := &mergeAudit{}
	var sources []*npcTable
	nameless, duplicateColumnsCount, truncated := 0, 0, 0
	var namelessExample, duplicateExample string
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.Size() == 0 {
			audit.Skipped = append(audit.Skipped, filepath.Base(p)+"（空档）")
			continue
		}
		t, err := loadTabTable(p, 2)
		if err != nil {
			audit.Skipped = append(audit.Skipped, fmt.Sprintf("%s（%v）", filepath.Base(p), err))
			continue
		}
		first := t.firstIndex()
		if _, hasID := first["ID"]; !hasID || !hasColumnKeys(first, domain.Keys) {
			audit.Skipped = append(audit.Skipped, fmt.Sprintf("%s（错置档：列头无 ID/%s 域签名）", filepath.Base(p), domain.Client))
			continue
		}
		sources = append(sources, t)
		audit.Files++
		if blanks := countBlanks(t.Columns); blanks > 0 {
			nameless += blanks
			namelessExample = filepath.Base(p)
		}
		if dups := len(t.Columns) - len(first); dups > 0 {
			duplicateColumnsCount += dups
			duplicateExample = fmt.Sprintf("%s：%s", filepath.Base(p), strings.Join(duplicateColumnNames(t), "、"))
		}
		truncated += t.Truncated
	}
	if len(sources) == 0 {
		return nil, audit
	}
	if nameless > 0 {
		audit.Notes = append(audit.Notes, fmt.Sprintf("无名列 %d 个（值丢弃，如 %s）", nameless, namelessExample))
	}
	if duplicateColumnsCount > 0 {
		audit.Notes = append(audit.Notes, fmt.Sprintf("重复列 %d 处（取首见，如 %s）", duplicateColumnsCount, duplicateExample))
	}
	if truncated > 0 {
		audit.Notes = append(audit.Notes, fmt.Sprintf("超列宽截断行 %d（多余字段丢弃）", truncated))
	}

	// 列并集（首见序）；声明类型取首见该列的档。
	merged := &npcTable{}
	index := make(map[string]int, 256)
	for _, t := range sources {
		for i, name := range t.Columns {
			if name == "" {
				continue
			}
			if _, seen := index[name]; seen {
				continue
			}
			index[name] = len(merged.Columns)
			merged.Columns = append(merged.Columns, name)
			declared := ""
			if i < len(t.Types) {
				declared = t.Types[i]
			}
			merged.Types = append(merged.Types, declared)
		}
	}
	// 行按列键对齐；同名列取首见（与输出取舍一致）。
	idIdx := index["ID"]
	seen := map[string]int{}
	for _, t := range sources {
		first := t.firstIndex()
		for _, row := range t.Rows {
			cells := make([]string, len(merged.Columns))
			for i, name := range t.Columns {
				if name == "" || first[name] != i || i >= len(row) {
					continue
				}
				cells[index[name]] = row[i]
			}
			merged.Rows = append(merged.Rows, cells)
			if id := cells[idIdx]; id != "" {
				seen[id]++
				if seen[id] > 1 {
					audit.Repeats++
				}
			}
		}
	}
	return merged, audit
}

// hasColumnKeys 列头是否齐含签名键。
func hasColumnKeys(first map[string]int, keys []string) bool {
	for _, key := range keys {
		if _, ok := first[key]; !ok {
			return false
		}
	}
	return true
}

// countBlanks 列键中段的无名列数（尾部空列键已裁剪）。
func countBlanks(columns []string) int {
	n := 0
	for _, name := range columns {
		if name == "" {
			n++
		}
	}
	return n
}

// duplicateColumnNames 同名列清单（列键×出现次数，字典序）。
func duplicateColumnNames(t *npcTable) []string {
	counts := map[string]int{}
	for _, name := range t.Columns {
		if name != "" {
			counts[name]++
		}
	}
	var out []string
	for name, n := range counts {
		if n > 1 {
			out = append(out, fmt.Sprintf("%s×%d", name, n))
		}
	}
	sort.Strings(out)
	return out
}

// resolveTaskTitle 主表 Title 改存显示文本：源表 Title 列是设计期备注（与
// 客户端文案七成不一致），显示名以 TitleId 键在 stringtask 文案表的解析为
// 准（多步任务共用一个标题键；TitleId 留空回落 title_<ID>）；解析不到留
// 空。返回解析到文案的行数。
func resolveTaskTitle(t *npcTable, texts map[string]string) int {
	first := t.firstIndex()
	titleIdx, hasTitle := first["Title"]
	if !hasTitle {
		return 0
	}
	titleIDIdx, hasTitleID := first["TitleId"]
	resolved := 0
	for _, row := range t.Rows {
		key := ""
		if hasTitleID && titleIDIdx < len(row) {
			key = strings.TrimSpace(row[titleIDIdx])
		}
		if key == "" {
			key = "title_" + strings.TrimSpace(row[first["ID"]])
		}
		text := ""
		if texts != nil {
			text = strings.TrimSpace(texts[key])
		}
		if text != "" {
			resolved++
		}
		row[titleIdx] = text
	}
	return resolved
}

// convertQuest 任务域转换主入口（<modern>/share/task）：task/ 下各域目录并
// 一档，根级散表单档。texts 为 stringtask 文案表（主表 Title 解析用）。
// 返回域统计与全部输出（路径→期望行数，供 verify）。
func convertQuest(taskRoot string, texts map[string]string, questOut string) ([]domainReport, map[string]int, []string, error) {
	domains := []domainReport{}
	written := map[string]int{}
	var warnings []string

	taskDir := filepath.Join(taskRoot, "task")
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("npcgen: 读 %s: %w", taskDir, err)
	}
	byLower := make(map[string]*questDomain, len(questDomains))
	for i := range questDomains {
		byLower[strings.ToLower(questDomains[i].Client)] = &questDomains[i]
	}
	// 目录名大小写以磁盘为准，域序按 questDomains。
	diskName := map[string]string{} // 小写目录名 → 磁盘目录名
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		diskName[strings.ToLower(e.Name())] = e.Name()
		if _, known := byLower[strings.ToLower(e.Name())]; !known {
			warnings = append(warnings, fmt.Sprintf("任务域未识别，跳过: task/%s", e.Name()))
		}
	}
	for i := range questDomains {
		domain := &questDomains[i]
		name, ok := diskName[strings.ToLower(domain.Client)]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("任务域目录缺席: task/%s", domain.Client))
			continue
		}
		sub := filepath.Join(taskDir, name)
		list, err := os.ReadDir(sub)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("npcgen: 读 %s: %w", sub, err)
		}
		var files []string
		for _, e := range list {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".txt") {
				files = append(files, filepath.Join(sub, e.Name()))
			}
		}
		sort.Strings(files)
		merged, audit := mergeDomainFiles(*domain, files)
		if merged == nil {
			warnings = append(warnings, fmt.Sprintf("任务域 %s 无可解析表", name))
			continue
		}
		report := domainReport{
			Name: domain.Server, Client: name, Files: audit.Files,
			Rows: len(merged.Rows), Columns: len(merged.Columns),
			RepeatIDs: audit.Repeats, Skipped: audit.Skipped, Notes: audit.Notes,
		}
		if domain.Client == "task" {
			report.TitleText = resolveTaskTitle(merged, texts)
		}
		outPath := filepath.Join(questOut, domain.Server, domain.Server+".xml")
		if err := writeGenerated(outPath, genPassThroughTable(merged)); err != nil {
			return nil, nil, nil, err
		}
		written[outPath] = countIDRows(merged)
		domains = append(domains, report)
	}
	if _, ok := written[filepath.Join(questOut, "Task", "Task.xml")]; !ok {
		return nil, nil, nil, fmt.Errorf("npcgen: 任务主表无可解析行（%s/task/task）", taskRoot)
	}

	// 根级散表（字段字典/特效表/xxx_type_ID）。
	for _, table := range questRootTables {
		path := filepath.Join(taskRoot, filepath.FromSlash(table.Source))
		if _, err := os.Stat(path); err != nil {
			warnings = append(warnings, fmt.Sprintf("根级表缺失: %s", table.Source))
			continue
		}
		t, err := loadTabTable(path, 2)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("跳过 %s: %v", table.Source, err))
			continue
		}
		outPath := filepath.Join(questOut, table.Output)
		if err := writeGenerated(outPath, genPassThroughTable(t)); err != nil {
			return nil, nil, nil, err
		}
		written[outPath] = countIDRows(t)
		domains = append(domains, domainReport{
			Name: strings.TrimSuffix(table.Output, filepath.Ext(table.Output)), Client: table.Source,
			Files: 1, Rows: written[outPath], Columns: len(t.Columns),
		})
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
