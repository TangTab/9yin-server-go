// gen.go 转换面：模板源表 → 按大类分档的 Object/Property XML、静态包 →
// 保留表头行的包档、path_config 登记表、ItemTypeList 类型树，并产出审计
// 报告条目与 verify 期望（行数/ID 序/首行属性）。
package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// builtFile 一个输出文件的字节与 verify 期望。
type builtFile struct {
	RelPath string            // 相对输出根
	Data    []byte            // 文件字节
	IDs     []string          // 期望 ID 序（verify 逐行比对）
	First   map[string]string // 首行非空属性（verify 抽查回读）
}

// columnReport 输出列报告条目。
type columnReport struct {
	Attr  string `json:"attr"`
	Kind  string `json:"kind"`            // heijin | struct | define | rename | native
	Empty int    `json:"empty,omitempty"` // 空值未写出的次数
	Note  string `json:"note,omitempty"`
}

// countEntry 名字 → 次数（script 分布等）。
type countEntry struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// fileReport 一个输出文件的审计条目。
type fileReport struct {
	Title           string         `json:"title"`
	Source          string         `json:"source,omitempty"`
	Output          string         `json:"output"`
	Type            string         `json:"type,omitempty"`
	Rows            int            `json:"rows"`
	Scripts         []countEntry   `json:"scripts,omitempty"`
	Columns         []columnReport `json:"columns"`
	RecordKeys      []string       `json:"record_keys,omitempty"`
	InvalidKeys     []string       `json:"invalid_keys,omitempty"`
	DuplicateIDs    int            `json:"duplicate_ids"`
	DuplicateKey    int            `json:"duplicate_keys"`
	EmptySections   int            `json:"empty_sections,omitempty"`
	CommentSections []string       `json:"comment_sections,omitempty"`
	NoScript        int            `json:"no_script,omitempty"`
	Note            string         `json:"note,omitempty"`
	Warnings        []string       `json:"warnings,omitempty"`
}

// itemRow 一行物品的输出属性（ID 另存）。
type itemRow struct {
	id     string
	values map[string]string
}

// kindBuilder 一个模板档的累加器（多张源表的行汇入同一档，列序取汇入
// 后的首见序）。
type kindBuilder struct {
	spec      kindSpec
	sources   map[string]bool // 汇入本档的源表
	rows      []*itemRow
	order     []string // 输出属性首见序
	seen      map[string]bool
	seenValue map[string]bool // 实际写出过非空值的属性（空值键不成列）
	scripts   map[string]int
	noScript  int
	invalid   map[string]int
	record    map[string]bool
	empty     map[string]int
	dupKeys   int
	dupIDs    int
	emptySec  int
}

func newKindBuilder(spec kindSpec) *kindBuilder {
	return &kindBuilder{
		spec: spec, sources: map[string]bool{}, seen: map[string]bool{}, seenValue: map[string]bool{},
		scripts: map[string]int{}, invalid: map[string]int{}, record: map[string]bool{}, empty: map[string]int{},
	}
}

// addSource 登记汇入源表（审计用）。
func (b *kindBuilder) addSource(source string) { b.sources[source] = true }

// addSection 一节 → 一行（空节跳过）。`@记录键` 多行以 ; 连接成一个属性
// （黑金 RandSerial="1,3;2,6" 同款行串方言）；非法 XML 属性名丢弃；空值
// 不写；同名键重复出现输出 原名_2/_3。
func (b *kindBuilder) addSection(section *iniSection) {
	if len(section.Keys) == 0 {
		b.emptySec++
		return
	}
	row := &itemRow{id: section.Name, values: map[string]string{}}
	occurrence := map[string]int{}
	joined := map[string]bool{}
	for i, key := range section.Keys {
		value := section.Vals[i]
		attr := key
		if name, ok := recordAttr(key); ok {
			attr = name
			if old, exists := row.values[attr]; exists && old != "" {
				value = old + ";" + value
			}
			row.values[attr] = value
			if !joined[attr] {
				joined[attr] = true
				b.note(attr)
				b.record[attr] = true
			}
			b.seenValue[attr] = true
			continue
		}
		if !validAttrName(key) {
			b.invalid[key]++
			continue
		}
		occurrence[attr]++
		if occurrence[attr] > 1 {
			b.dupKeys++
			attr = fmt.Sprintf("%s_%d", key, occurrence[attr])
		}
		b.note(attr)
		if value == "" {
			b.empty[attr]++
			continue
		}
		row.values[attr] = value
		b.seenValue[attr] = true
	}
	if script, ok := row.values["script"]; ok {
		b.scripts[script]++
	} else {
		b.noScript++
	}
	b.rows = append(b.rows, row)
}

// note 登记属性首见序。
func (b *kindBuilder) note(attr string) {
	if !b.seen[attr] {
		b.seen[attr] = true
		b.order = append(b.order, attr)
	}
}

// columnOrder 输出列序：前置列（声明序）→ 其余首见序。
func (b *kindBuilder) columnOrder() []string {
	isLead := map[string]bool{}
	var order []string
	for _, lead := range leadColumns {
		if b.seenValue[lead] && !isLead[lead] {
			isLead[lead] = true
			order = append(order, lead)
		}
	}
	for _, attr := range b.order {
		if b.seenValue[attr] && !isLead[attr] {
			order = append(order, attr)
		}
	}
	return order
}

// finish 写出本档：返回文件与审计条目（重复 ID 只记账，行全保留——同一
// ConfigID 两行是源表事实，装载时首行胜出）。
func (b *kindBuilder) finish(vocab *vocabulary) (*builtFile, *fileReport) {
	w := newObjectWriter(fmt.Sprintf("%s：%s", b.spec.Type, b.spec.Note))
	order := b.columnOrder()
	file := &builtFile{RelPath: b.spec.File + ".xml"}
	idSeen := map[string]bool{}
	for _, row := range b.rows {
		attrs := make([][2]string, 0, len(row.values)+1)
		attrs = append(attrs, [2]string{"ID", row.id})
		for _, attr := range order {
			if value, ok := row.values[attr]; ok {
				attrs = append(attrs, [2]string{attr, value})
			}
		}
		w.AddProperty(attrs)
		if idSeen[row.id] {
			b.dupIDs++
		}
		idSeen[row.id] = true
		file.IDs = append(file.IDs, row.id)
		if file.First == nil {
			file.First = map[string]string{}
			for _, kv := range attrs {
				file.First[kv[0]] = kv[1]
			}
		}
	}
	file.Data = w.Bytes()

	report := &fileReport{
		Title: b.spec.Type, Output: file.RelPath, Type: b.spec.Type,
		Source: strings.Join(sortedKeys(b.sources), ","), Rows: len(b.rows),
		Scripts: topCounts(b.scripts, 15), RecordKeys: sortedKeys(b.record), NoScript: b.noScript,
		InvalidKeys: sortedKeys(b.invalid), DuplicateKey: b.dupKeys, DuplicateIDs: b.dupIDs,
		EmptySections: b.emptySec, Note: b.spec.Note,
	}
	for _, attr := range order {
		column := columnReport{Attr: attr, Empty: b.empty[attr]}
		switch {
		case b.record[attr]:
			column.Kind, column.Note = kindRename, "@记录键改名（recordAttr），多行以 ; 连接"
		default:
			column.Kind = vocab.classify(attr)
		}
		report.Columns = append(report.Columns, column)
	}
	return file, report
}

// buildPack 静态包：[-1]/[0] 两行表头照留（黑金已发布静态包同形），数据行
// 列序取 [-1] 列序头、未登记键按首见序殿后。第三个返回值是数据行属性
// （ID → 属性，@记录键已去 @）——引用完整性/使用链统计用。
func buildPack(t *iniTable, spec packSpec, vocab *vocabulary) (*builtFile, *fileReport, map[string]map[string]string) {
	report := &fileReport{Title: spec.Title, Source: t.Source, Output: spec.Output, Note: spec.Note,
		Warnings: t.Warnings}
	rowsOut := map[string]map[string]string{}
	var order []string
	seen := map[string]bool{}
	note := func(attr string) {
		if !seen[attr] {
			seen[attr] = true
			order = append(order, attr)
		}
	}
	headers := map[string]*iniSection{}
	for _, section := range t.Sections {
		if section.Name == "-1" || section.Name == "0" {
			headers[section.Name] = section
		}
	}
	if h := headers["-1"]; h != nil {
		type slot struct {
			name  string
			index int
		}
		var slots []slot
		for i, key := range h.Keys {
			if index, err := strconv.Atoi(h.Vals[i]); err == nil && validAttrName(key) {
				slots = append(slots, slot{key, index})
			}
		}
		sort.Slice(slots, func(i, j int) bool { return slots[i].index < slots[j].index })
		for _, s := range slots {
			note(s.name)
		}
	} else {
		report.Warnings = append(report.Warnings, "源表无 [-1] 列序头")
	}

	w := newObjectWriter(spec.Title + "：" + spec.Note)
	file := &builtFile{RelPath: spec.Output}
	invalid, record := map[string]int{}, map[string]bool{}
	dupKeys, dupIDs, emptySec := 0, 0, 0
	idSeen := map[string]bool{}
	for _, section := range t.Sections {
		if section.Name == "-1" || section.Name == "0" {
			// 表头行：键值原样（列宽 "0" 也是值）。
			attrs := [][2]string{{"ID", section.Name}}
			for i, key := range section.Keys {
				if validAttrName(key) {
					attrs = append(attrs, [2]string{key, section.Vals[i]})
				}
			}
			w.AddProperty(attrs)
			file.IDs = append(file.IDs, section.Name)
			if file.First == nil {
				file.First = map[string]string{}
				for _, kv := range attrs {
					file.First[kv[0]] = kv[1]
				}
			}
			continue
		}
		if strings.HasPrefix(section.Name, "//") {
			report.CommentSections = append(report.CommentSections, section.Name)
			continue
		}
		if len(section.Keys) == 0 {
			emptySec++
			continue
		}
		row := map[string]string{}
		occurrence, joined := map[string]int{}, map[string]bool{}
		for i, key := range section.Keys {
			value, attr := section.Vals[i], key
			if name, ok := recordAttr(key); ok {
				attr = name
				if old, exists := row[attr]; exists && old != "" {
					value = old + ";" + value
				}
				if !joined[attr] {
					joined[attr] = true
					note(attr)
					record[attr] = true
				}
			} else if !validAttrName(key) {
				invalid[key]++
				continue
			} else {
				occurrence[attr]++
				if occurrence[attr] > 1 {
					dupKeys++
					attr = fmt.Sprintf("%s_%d", key, occurrence[attr])
				}
				note(attr)
			}
			row[attr] = value
		}
		attrs := [][2]string{{"ID", section.Name}}
		for _, attr := range order {
			if value, ok := row[attr]; ok {
				attrs = append(attrs, [2]string{attr, value})
			}
		}
		w.AddProperty(attrs)
		if idSeen[section.Name] {
			dupIDs++
		}
		idSeen[section.Name] = true
		file.IDs = append(file.IDs, section.Name)
		rowsOut[section.Name] = row
		report.Rows++
		if file.First == nil {
			file.First = map[string]string{}
			for _, kv := range attrs {
				file.First[kv[0]] = kv[1]
			}
		}
	}
	file.Data = w.Bytes()
	report.InvalidKeys, report.RecordKeys = sortedKeys(invalid), sortedKeys(record)
	report.DuplicateKey, report.DuplicateIDs, report.EmptySections = dupKeys, dupIDs, emptySec
	for _, attr := range order {
		column := columnReport{Attr: attr}
		if record[attr] {
			column.Kind, column.Note = kindRename, "@记录键改名（recordAttr），多行以 ; 连接"
		} else {
			column.Kind = vocab.classify(attr)
		}
		report.Columns = append(report.Columns, column)
	}
	return file, report, rowsOut
}

// buildPathConfig 黑金 path_config.xml 同构：逐档登记 ID/Type/Path。
func buildPathConfig(kinds []*builtFile) *builtFile {
	w := newObjectWriter("物品模板登记表（黑金 path_config.xml 同构：ID/Type/Path）")
	file := &builtFile{RelPath: "path_config.xml"}
	for i, kind := range kinds {
		attrs := [][2]string{{"ID", strconv.Itoa(i + 1)}, {"Type", kindTypeOf(kind.RelPath)}, {"Path", kind.RelPath}}
		w.AddProperty(attrs)
		file.IDs = append(file.IDs, attrs[0][1])
		if file.First == nil {
			file.First = map[string]string{}
			for _, kv := range attrs {
				file.First[kv[0]] = kv[1]
			}
		}
	}
	file.Data = w.Bytes()
	return file
}

// buildItemTypeList 黑金 ItemTypeList.xml 同构：ID/FatherType/Desc（根 55555
// 殿后）。lua 常量名即 Desc；TextID 九阴无键规不写。
func buildItemTypeList(entries []itemTypeEntry) *builtFile {
	w := newObjectWriter("物品类型总表（黑金 ItemTypeList.xml 同构；源=lua64/share/itemtype_define.lua）")
	file := &builtFile{RelPath: "ItemTypeList.xml"}
	add := func(id, father int, desc string) {
		attrs := [][2]string{
			{"ID", strconv.Itoa(id)}, {"FatherType", strconv.Itoa(father)}, {"Desc", desc},
		}
		w.AddProperty(attrs)
		file.IDs = append(file.IDs, attrs[0][1])
		if file.First == nil {
			file.First = map[string]string{}
			for _, kv := range attrs {
				file.First[kv[0]] = kv[1]
			}
		}
	}
	for _, entry := range entries {
		add(entry.ID, entry.Father, strings.Join(entry.Names, "/"))
	}
	add(itemTypeRoot, itemTypeRoot, "全部")
	file.Data = w.Bytes()
	return file
}

// kindTypeOf 输出文件名 → path_config Type（分档表查）。
func kindTypeOf(file string) string {
	for _, spec := range kindSpecs {
		if spec.File+".xml" == file {
			return spec.Type
		}
	}
	return ""
}

func topCounts(counts map[string]int, limit int) []countEntry {
	entries := make([]countEntry, 0, len(counts))
	for name, count := range counts {
		entries = append(entries, countEntry{name, count})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		return entries[i].Name < entries[j].Name
	})
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
