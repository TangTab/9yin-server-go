package clientdata

import (
	_ "embed"
	"encoding/binary"
	"fmt"
)

//go:embed official_record_table.bin
var officialRecordTableFrame []byte

//go:embed self_home_rec_frame.bin
var homeRecordsMirrorFrame []byte

// RecordSchema 是官方 0x0A 记录表（447 schema）的一项。ColumnTypes 保留
// 官方线型码：1=byte、2=u16、3=int32、4=int64、6=f64、7=窄字符串、
// 8=宽字符串、9=object 引用。
type RecordSchema struct {
	Index       uint16
	Name        string
	ColumnTypes []byte
}

type recordTable struct {
	entries []RecordSchema
	byName  map[string]RecordSchema
}

var officialRecords = parseRecordTableFrame(officialRecordTableFrame)

// parseRecordTableFrame 从 0x0A 帧 [0x0A][u16 N]{name\0 + u16 ncols +
// ncols×colType} 解析 schema 表（官方 447 bin 与 golden 344 帧共用）。
func parseRecordTableFrame(frame []byte) recordTable {
	if len(frame) < 3 || frame[0] != 0x0A {
		panic("clientdata: official record table frame has unexpected header")
	}
	count := int(binary.LittleEndian.Uint16(frame[1:]))
	table := recordTable{entries: make([]RecordSchema, 0, count), byName: make(map[string]RecordSchema, count)}
	offset := 3
	for index := 0; index < count; index++ {
		end := offset
		for end < len(frame) && frame[end] != 0 {
			end++
		}
		if end >= len(frame) {
			panic(fmt.Sprintf("clientdata: official record table truncated at schema %d", index))
		}
		name := string(frame[offset:end])
		columns := int(binary.LittleEndian.Uint16(frame[end+1:]))
		if end+3+columns > len(frame) {
			panic(fmt.Sprintf("clientdata: official record table column overflow at schema %d", index))
		}
		columnTypes := append([]byte(nil), frame[end+3:end+3+columns]...)
		schema := RecordSchema{Index: uint16(index), Name: name, ColumnTypes: columnTypes}
		table.entries = append(table.entries, schema)
		// 与 Java RecordSchemas 相同的首现语义：官方表按名首现绑定。
		if _, exists := table.byName[name]; !exists {
			table.byName[name] = schema
		}
		offset = end + 3 + columns
	}
	if len(table.entries) != count {
		panic("clientdata: official record table schema count mismatch")
	}
	return table
}

// HomeRecordsMirrorFrame 逐字节透传外服登录链 0x0A 镜像帧（446 schema，
// 10429B，c.pcapng 实证形态）。客户端取其帧头 16B 作 C2S 0x0A 活动信封
// 头回显；登录链 recId ≤446 与官方 447 表完全兼容（第 447 项为 fork 追加）。
func HomeRecordsMirrorFrame() []byte {
	return append([]byte(nil), homeRecordsMirrorFrame...)
}

// OfficialRecordTableFrame 逐字节透传官方 0x0A 记录表协商帧
// （与官方服抓包完全一致）。客户端 C++ 原生消费者
// （CoolLogicManager 等）按官方表下标绑定记录，自协商方言
// 会让行帧字节级正确却不上显。
func OfficialRecordTableFrame() []byte {
	return append([]byte(nil), officialRecordTableFrame...)
}

// OfficialRecordSchemas 返回全部声明，帧内位置即 Index。
func OfficialRecordSchemas() []RecordSchema {
	return append([]RecordSchema(nil), officialRecords.entries...)
}

// RecordSchemaByIndex 按官方下标取 schema。
func RecordSchemaByIndex(index int) (RecordSchema, bool) {
	if index < 0 || index >= len(officialRecords.entries) {
		return RecordSchema{}, false
	}
	return officialRecords.entries[index], true
}

// MustRecordIndexByName 按名取官方下标（首现）；缺名 panic——
// 编码期暴露而非线上错绑。
func MustRecordIndexByName(name string) uint16 {
	schema, ok := officialRecords.byName[name]
	if !ok {
		panic(fmt.Sprintf("clientdata: record name absent from official table: %s", name))
	}
	return schema.Index
}
