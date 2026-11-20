// table.go 客户端模板表解析（GBK+TAB、3 行头），移植自
// 9yin-go-server2 internal/clientdata/npc_template_table.go 并按本工具
// 输出需求裁剪：本工具不区分逻辑/物理行号，只要列键与数据行。
package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// npcTable 一张客户端 NPC 模板表。
type npcTable struct {
	Source   string   // npcconfig 相对路径（slash）
	Comments []string // 第 1 行：中文列注释（进审计报告）
	Types    []string // 第 2 行：声明类型
	Columns  []string // 第 3 行：英文列键
	Rows     [][]string
}

// firstIndex 列键 → 首次出现的列下标。同名列（WeaponName/FreeTalk/
// KnockTalk/StopTime 各出现两次）取首见，其余在报告标注。
func (t *npcTable) firstIndex() map[string]int {
	m := make(map[string]int, len(t.Columns))
	for i, name := range t.Columns {
		if _, ok := m[name]; !ok {
			m[name] = i
		}
	}
	return m
}

// duplicateColumns 同名列清单（列键,出现次数），供报告标注。
func (t *npcTable) duplicateColumns() []string {
	counts := map[string]int{}
	for _, name := range t.Columns {
		counts[name]++
	}
	var out []string
	for name, n := range counts {
		if n > 1 {
			out = append(out, fmt.Sprintf("%s×%d", name, n))
		}
	}
	sortStrings(out)
	return out
}

// LoadNPCTable 读 GBK TSV 表。encoding/csv 是刻意选择：个别字段带引号
// 内嵌换行，不能按行硬切。
func LoadNPCTable(r io.Reader, source string) (*npcTable, error) {
	if r == nil {
		return nil, errors.New("npcgen: nil 表 reader")
	}
	reader := csv.NewReader(transform.NewReader(r, simplifiedchinese.GBK.NewDecoder()))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = false

	readRecord := func(recordNumber int) ([]string, error) {
		record, err := reader.Read()
		if err != nil {
			return nil, fmt.Errorf("npcgen: 读 %s 头部行 %d: %w", source, recordNumber, err)
		}
		return record, nil
	}
	comments, err := readRecord(1)
	if err != nil {
		return nil, err
	}
	declared, err := readRecord(2)
	if err != nil {
		return nil, err
	}
	columns, err := readRecord(3)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("npcgen: %s 无列键行", source)
	}
	if len(comments) != len(columns) || len(declared) != len(columns) {
		return nil, fmt.Errorf("npcgen: %s 头宽不一致 注释=%d 类型=%d 列键=%d", source, len(comments), len(declared), len(columns))
	}
	reader.FieldsPerRecord = len(columns)

	t := &npcTable{
		Source:   source,
		Comments: comments,
		Types:    declared,
		Columns:  columns,
	}
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("npcgen: 读 %s 数据行: %w", source, readErr)
		}
		// 表内整段空行是排版分隔，不是逻辑记录。
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
		if len(record) < len(columns) {
			return nil, fmt.Errorf("npcgen: %s 数据行宽 %d < 列宽 %d", source, len(record), len(columns))
		}
		t.Rows = append(t.Rows, record)
	}
	return t, nil
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
