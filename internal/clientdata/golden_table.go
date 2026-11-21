package clientdata

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// 真机镜像表源（2026-11-21 用户外服抓包「初始账号登陆.pcapng」提取，
// 与 Blue Dragon 线客户端版本逐字节匹配——2433 项 txt 表与该客户端
// idx 绑定错位会直接崩溃）。帧即下发字节：透传 + 解析共用同一数据。
//
//	property_table_golden.bin  0x09 属性表帧（2086 项，29664B）
//	record_table_golden.bin    0x0A 记录表帧（344 schema，7921B）
//
// 方言选择（2026-11-21 决策）：默认 official2433（fork/7074 线客户端，
// b-line 真机跑通）；NINEYIN_TABLE_DIALECT=golden 切回镜像方言（回退开关）。

//go:embed property_table_golden.bin
var goldenPropertyTableFrame []byte

//go:embed record_table_golden.bin
var goldenRecordTableFrame []byte

// GoldenPropertyTableFrame 0x09 属性表帧逐字节透传（客户端版本对齐源）。
func GoldenPropertyTableFrame() []byte {
	return append([]byte(nil), goldenPropertyTableFrame...)
}

// GoldenRecordTableFrame 0x0A 记录表帧逐字节透传。
func GoldenRecordTableFrame() []byte {
	return append([]byte(nil), goldenRecordTableFrame...)
}

// parseGoldenPropertyTable 从 0x09 帧 [0x09][u16 N]{name\0 + u8 type}×N
// 解析属性行（帧内位置即索引）。
func parseGoldenPropertyTable() []PropertyEntry {
	frame := goldenPropertyTableFrame
	if len(frame) < 3 || frame[0] != 0x09 {
		panic("clientdata: golden property table frame malformed")
	}
	n := int(binary.LittleEndian.Uint16(frame[1:]))
	entries := make([]PropertyEntry, 0, n)
	off := 3
	for i := 0; i < n; i++ {
		end := off
		for end < len(frame) && frame[end] != 0 {
			end++
		}
		if end >= len(frame) {
			panic(fmt.Sprintf("clientdata: golden property table truncated at %d", i))
		}
		code := int(frame[end+1])
		name := string(frame[off:end])
		entries = append(entries, PropertyEntry{
			Index:    uint16(i),
			Name:     name,
			TypeCode: code,
			Type:     OfficialWireTypeOf(code),
		})
		off = end + 2
	}
	if len(entries) != n {
		panic("clientdata: golden property table count mismatch")
	}
	return entries
}

// switchToGoldenTable 以真机镜像表替换默认 2433 方言表（Name→Index 解析
// 与透传帧同源，属性表 + 记录表一并切换）。
func switchToGoldenTable() {
	entries := parseGoldenPropertyTable()
	table := propertyTable{
		entries: entries,
		scene:   map[string]PropertyEntry{},
		player:  map[string]PropertyEntry{},
	}
	for _, entry := range entries {
		if entry.Type == 0 || entry.Name == "None" {
			continue
		}
		if _, exists := table.scene[entry.Name]; !exists {
			table.scene[entry.Name] = entry
			table.sceneFields = append(table.sceneFields, entry)
		}
		table.player[entry.Name] = entry
	}
	officialTable = table
	officialRecords = parseRecordTableFrame(goldenRecordTableFrame)
	goldenTableActive = true
}

// goldenTableActive 当前是否镜像方言（诊断用）。
var goldenTableActive bool

// GoldenTableActive 镜像方言生效态。
func GoldenTableActive() bool { return goldenTableActive }

// init 按 env 门控方言：默认 official2433（fork/7074 线客户端），
// NINEYIN_TABLE_DIALECT=golden 切回真机镜像（Blue Dragon 线客户端回退）。
func init() {
	if strings.TrimSpace(os.Getenv("NINEYIN_TABLE_DIALECT")) == "golden" {
		switchToGoldenTable()
	}
}
