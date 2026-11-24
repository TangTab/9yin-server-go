package sharedata

// table.go TAB 表读取（TAB 分隔；字段可带双引号、内嵌换行——用 encoding/csv
// 而不按行硬切）。两种表头方言与构建器（tools/datagen/npc）逐条对齐：
//
//	ReadTemplateTable  NPC 模板表 share/npc/npcconfig/**：3 行头（中文注释 /
//	                   类型 / 列键），严格宽度——头宽不一致、数据行窄于列宽
//	                   即报错（整表不可用，调用方跳过）。
//	ReadDomainTable    任务域 share/task/task/<域>/*（2 行头：类型 + 列键）与
//	                   战斗模板 npctemplate*.tab（1 行头：仅列键），宽松宽度——
//	                   短行补齐、超宽截断（多余字段带值才计 Truncated），列键
//	                   裁首尾空白并裁掉尾部空列键。

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// Table 一张 TAB 表。
type Table struct {
	Source    string   // 传入的路径（溯源 / 报错）
	Comments  []string // 列注释行（3 行头表；其余为等长空串）
	Types     []string // 声明类型行（1 行头表为空）
	Columns   []string // 列键
	Rows      [][]string
	Truncated int // 超列宽被截断的行数（仅宽松表）

	first map[string]int
}

// First 列键 → 首次出现的列下标。同名列取首见（源表确有重复列：NPC 模板
// 表的 WeaponName / FreeTalk / KnockTalk / StopTime，任务域的 AcceptBuffer…）。
func (t *Table) First() map[string]int {
	if t.first == nil {
		t.first = make(map[string]int, len(t.Columns))
		for i, name := range t.Columns {
			if _, ok := t.first[name]; !ok {
				t.first[name] = i
			}
		}
	}
	return t.first
}

// Column 列键的首见下标；无此列 -1。
func (t *Table) Column(name string) int {
	if i, ok := t.First()[name]; ok {
		return i
	}
	return -1
}

func newTSV(data []byte) *csv.Reader {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = false
	return reader
}

func allEmpty(record []string) bool {
	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// ReadTemplateTable NPC 模板表：3 行头（中文注释 / 类型 / 列键），严格宽度。
// 空行是排版分隔，跳过。
func ReadTemplateTable(path string) (*Table, error) {
	raw, err := res.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data, err := DecodeText(raw)
	if err != nil {
		return nil, fmt.Errorf("sharedata: 解码 %s: %w", path, err)
	}
	reader := newTSV(data)
	source := filepath.ToSlash(path)
	readHeader := func(n int) ([]string, error) {
		record, err := reader.Read()
		if err != nil {
			return nil, fmt.Errorf("sharedata: 读 %s 头部行 %d: %w", source, n, err)
		}
		return record, nil
	}
	comments, err := readHeader(1)
	if err != nil {
		return nil, err
	}
	declared, err := readHeader(2)
	if err != nil {
		return nil, err
	}
	columns, err := readHeader(3)
	if err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("sharedata: %s 无列键行", source)
	}
	if len(comments) != len(columns) || len(declared) != len(columns) {
		return nil, fmt.Errorf("sharedata: %s 头宽不一致 注释=%d 类型=%d 列键=%d",
			source, len(comments), len(declared), len(columns))
	}
	// 此后每条记录宽度固定为列宽（csv 报 ErrFieldCount）。
	reader.FieldsPerRecord = len(columns)
	t := &Table{Source: source, Comments: comments, Types: declared, Columns: columns}
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("sharedata: 读 %s 数据行: %w", source, readErr)
		}
		if allEmpty(record) {
			continue
		}
		t.Rows = append(t.Rows, record)
	}
	return t, nil
}

// ReadDomainTable 任务域表（headerRows=2：类型 + 列键）或战斗模板表
// （headerRows=1：仅列键），宽松宽度。
func ReadDomainTable(path string, headerRows int) (*Table, error) {
	raw, err := res.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data, err := DecodeText(raw)
	if err != nil {
		return nil, fmt.Errorf("sharedata: 解码 %s: %w", path, err)
	}
	reader := newTSV(data)
	source := filepath.ToSlash(path)
	t := &Table{Source: source}
	readRecord := func(what string) ([]string, error) {
		record, err := reader.Read()
		if err != nil {
			return nil, fmt.Errorf("sharedata: 读 %s %s: %w", source, what, err)
		}
		return record, nil
	}
	if headerRows >= 1 {
		first, err := readRecord("类型行")
		if err != nil {
			return nil, err
		}
		if headerRows >= 2 {
			t.Types = first
		} else {
			t.Columns = first // 1 行头：首行即列键
		}
	}
	if headerRows >= 2 {
		columns, err := readRecord("列键行")
		if err != nil {
			return nil, err
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
		return nil, errors.New("sharedata: " + source + " 无列键")
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
			return nil, fmt.Errorf("sharedata: 读 %s 数据行: %w", source, err)
		}
		if allEmpty(record) {
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
