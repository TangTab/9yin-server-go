// gen.go 转换面：一张 ini 表 → 一个或多个 Object/Property XML（分组表按
// 组键分文件），并产出审计报告条目与 verify 期望（行数/ID 序/首行属性）。
package skill

import (
	"fmt"
	"sort"
	"strings"
)

// builtFile 一个输出文件的字节与 verify 期望。
type builtFile struct {
	RelPath string            // 相对输出根
	Data    []byte            // 文件字节
	IDs     []string          // 期望 ID 序（verify 逐行比对）
	First   map[string]string // 首行非空属性（verify 抽查回读）
	Columns []columnReport    // 输出列出处（同表各分组共用）
}

// columnReport 输出列报告条目。
type columnReport struct {
	Attr string `json:"attr"`
	Kind string `json:"kind"` // direct | heijin | define | native | rename | dropped
	Note string `json:"note,omitempty"`
}

// tableBuild 一张源表的转换结果。
type tableBuild struct {
	Spec            tableSpec
	Files           []*builtFile
	Rows            int
	HeadersDropped  []string // 实际丢弃的表头元数据行节名
	ConflictMarkers []int    // 源表残留合并冲突标记行号
	InvalidKeys     []string // 非法 XML 属性名而丢弃的源键
	DuplicateKeys   int      // 节内重复键出现次数（输出为 _2/_3 序号）
	DuplicateIDs    int      // 同一输出文件内重复的节名个数
	Dropped         []columnReport
	Warnings        []string
}

// groupOf 分组键取值（缺省 "Unknown"；组键折叠为文件名安全字符）。
func (spec tableSpec) groupOf(section *iniSection) string {
	if spec.Group == "" {
		return ""
	}
	value, ok := section.Get(spec.Group)
	if !ok || value == "" {
		return "Unknown"
	}
	var sb strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			sb.WriteRune(r)
		default:
			sb.WriteByte('_')
		}
	}
	if sb.Len() == 0 {
		return "Unknown"
	}
	return sb.String()
}

// outAttr 源键 → 输出属性名（审定改名优先，否则原名）。
func (spec tableSpec) outAttr(key string) string {
	if target, ok := spec.Rename[key]; ok {
		return target
	}
	return key
}

// sourceOf 审定改名的反向查找（输出属性名 → 源键）。
func (spec tableSpec) sourceOf(attr string) (string, bool) {
	for key, target := range spec.Rename {
		if target == attr {
			return key, true
		}
	}
	return "", false
}

// convertTable 转换一张表。输出列序：前置列（Lead，取源表存在者、按声明
// 序）→ 其余按输出属性名在源表的首见序；空值属性不写（服务端方言同
// npcgen）；节 ID 恒为首个属性。
func convertTable(t *iniTable, spec tableSpec, vocab *vocabulary) *tableBuild {
	b := &tableBuild{Spec: spec, ConflictMarkers: t.ConflictMarkers}
	b.Warnings = append(b.Warnings, t.Warnings...)

	isHeader := func(name string) bool {
		for _, h := range spec.Headers {
			if h == name {
				return true
			}
		}
		return false
	}

	// 表头元数据行与审定丢弃列（只在源表确实存在时入报告）。
	present := map[string]bool{}
	for _, section := range t.Sections {
		present[section.Name] = true
	}
	for _, h := range spec.Headers {
		if present[h] {
			b.HeadersDropped = append(b.HeadersDropped, h)
		}
	}
	for key, reason := range spec.Drop {
		b.Dropped = append(b.Dropped, columnReport{Attr: key, Kind: kindDropped, Note: reason})
	}
	sort.Slice(b.Dropped, func(i, j int) bool { return b.Dropped[i].Attr < b.Dropped[j].Attr })

	// 第一遍：输出属性首见序、非法键、节内重复键变体。
	var firstSeen []string
	seenAttr, invalid, dupAttrs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, section := range t.Sections {
		if isHeader(section.Name) {
			continue
		}
		occurrence := map[string]int{}
		for _, key := range section.Keys {
			if _, drop := spec.Drop[key]; drop {
				continue
			}
			attr := spec.outAttr(key)
			if !validAttrName(attr) {
				invalid[key] = true
				continue
			}
			occurrence[attr]++
			if occurrence[attr] > 1 {
				dupAttrs[fmt.Sprintf("%s_%d", attr, occurrence[attr])] = true
			}
			if !seenAttr[attr] {
				seenAttr[attr] = true
				firstSeen = append(firstSeen, attr)
			}
		}
	}
	for key := range invalid {
		b.InvalidKeys = append(b.InvalidKeys, key)
	}
	sort.Strings(b.InvalidKeys)

	// 列序与列出处：Lead 声明序在前，其余保持首见序；重复键变体列殿后。
	isLead := map[string]bool{}
	var order []string
	for _, lead := range spec.Lead {
		if seenAttr[lead] && !isLead[lead] {
			isLead[lead] = true
			order = append(order, lead)
		}
	}
	for _, attr := range firstSeen {
		if !isLead[attr] {
			order = append(order, attr)
		}
	}
	columns := make([]columnReport, 0, len(order)+len(dupAttrs))
	for _, attr := range order {
		if src, renamed := spec.sourceOf(attr); renamed {
			columns = append(columns, columnReport{Attr: attr, Kind: kindRename, Note: src + "→" + attr})
		} else {
			columns = append(columns, columnReport{Attr: attr, Kind: vocab.classify(attr)})
		}
	}
	tail := len(columns)
	for attr := range dupAttrs {
		columns = append(columns, columnReport{Attr: attr, Kind: kindNative, Note: "节内重复键"})
	}
	sort.Slice(columns[tail:], func(i, j int) bool { return columns[tail+i].Attr < columns[tail+j].Attr })

	// 第二遍：逐节输出。
	writers := map[string]*objectWriter{}
	fileIDs := map[string][]string{}
	fileFirst := map[string]map[string]string{}
	var groups []string
	for _, section := range t.Sections {
		if isHeader(section.Name) {
			continue
		}
		if section.Name == "" {
			b.Warnings = append(b.Warnings, fmt.Sprintf("%s:%d: 空节名跳过", t.Source, section.Line))
			continue
		}
		group := spec.groupOf(section)
		w := writers[group]
		if w == nil {
			w = newObjectWriter()
			writers[group] = w
			groups = append(groups, group)
		}
		// 同一输出属性的多次出现：首次用原名，其后为 原名_2/_3。
		occurrence := map[string]int{}
		rowVals := map[string]string{}
		for i, key := range section.Keys {
			if _, drop := spec.Drop[key]; drop || invalid[key] {
				continue
			}
			attr := spec.outAttr(key)
			occurrence[attr]++
			name := attr
			if occurrence[attr] > 1 {
				b.DuplicateKeys++
				name = fmt.Sprintf("%s_%d", attr, occurrence[attr])
			}
			rowVals[name] = section.Vals[i]
		}
		attrs := make([][2]string, 0, len(order)+4)
		attrs = append(attrs, [2]string{"ID", section.Name})
		for _, attr := range order {
			for n := 1; ; n++ {
				name := attr
				if n > 1 {
					name = fmt.Sprintf("%s_%d", attr, n)
				}
				value, ok := rowVals[name]
				if !ok {
					break
				}
				attrs = append(attrs, [2]string{name, value})
			}
		}
		w.AddProperty(attrs)
		fileIDs[group] = append(fileIDs[group], section.Name)
		if fileFirst[group] == nil {
			fileFirst[group] = map[string]string{}
			for _, kv := range attrs {
				if kv[1] != "" {
					fileFirst[group][kv[0]] = kv[1]
				}
			}
		}
		b.Rows++
	}
	for _, group := range groups {
		idSeen := map[string]bool{}
		for _, id := range fileIDs[group] {
			if idSeen[id] {
				b.DuplicateIDs++
			}
			idSeen[id] = true
		}
		rel := spec.Output
		if spec.Group != "" {
			rel = fmt.Sprintf(spec.Output, group)
		}
		b.Files = append(b.Files, &builtFile{RelPath: rel, Data: writers[group].Bytes(),
			IDs: fileIDs[group], First: fileFirst[group], Columns: columns})
	}
	return b
}
