package entergen

// 基础帧编码器：0x0B/0x0D/0x10/0x11/0x14/0x15/0x17/0x18/0x1F。
// 布局全部 b-line 真机定版（docs/official/外服登录帧对照与缺口清单.md）。

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
)

// IndexedProperty 一条带表索引的属性值（0x10/0x0D/视图行共用载体）。
type IndexedProperty struct {
	Index uint16
	Name  string
	Value PropertyValue
}

// PropertyValue 按线型的单值。编码不自描述——宽度由 0x09 表条目决定。
type PropertyValue struct {
	Byte     byte
	Word     uint16
	Int32    int32
	Int64    int64
	Float32  float32
	Float64  float64
	Text     string
	WideText string
	Obj      uint64
	kind     byte // 1=byte 2=word 3=int32 4=int64 5=f32 6=f64 7=str 8=wide 9=obj
}

func ByteValue(v byte) PropertyValue       { return PropertyValue{kind: 1, Byte: v} }
func WordValue(v uint16) PropertyValue     { return PropertyValue{kind: 2, Word: v} }
func Int32Value(v int32) PropertyValue     { return PropertyValue{kind: 3, Int32: v} }
func Int64Value(v int64) PropertyValue     { return PropertyValue{kind: 4, Int64: v} }
func Float32Value(v float32) PropertyValue { return PropertyValue{kind: 5, Float32: v} }
func Float64Value(v float64) PropertyValue { return PropertyValue{kind: 6, Float64: v} }
func StringValue(v string) PropertyValue   { return PropertyValue{kind: 7, Text: v} }
func WideValue(v string) PropertyValue     { return PropertyValue{kind: 8, WideText: v} }
func ObjectValue(obj, own uint32) PropertyValue {
	return PropertyValue{kind: 9, Obj: uint64(obj) | uint64(own)<<32}
}

// appendProperty 按表线型追加 [idx u16][值]。kind 与表型不符时按表型
// 取对应字段（调用方保证一致——Must*Property 按名解析后直接赋 kind）。
func appendProperty(msg []byte, prop IndexedProperty) []byte {
	msg = binary.LittleEndian.AppendUint16(msg, prop.Index)
	switch prop.Value.kind {
	case 1:
		return append(msg, prop.Value.Byte)
	case 2:
		return binary.LittleEndian.AppendUint16(msg, prop.Value.Word)
	case 3:
		return binary.LittleEndian.AppendUint32(msg, uint32(prop.Value.Int32))
	case 4:
		return binary.LittleEndian.AppendUint64(msg, uint64(prop.Value.Int64))
	case 5:
		return binary.LittleEndian.AppendUint32(msg, math.Float32bits(prop.Value.Float32))
	case 6:
		return binary.LittleEndian.AppendUint64(msg, math.Float64bits(prop.Value.Float64))
	case 7:
		msg = binary.LittleEndian.AppendUint32(msg, uint32(len(prop.Value.Text)+1))
		msg = append(msg, prop.Value.Text...)
		return append(msg, 0)
	case 8:
		// 宽串线长 = UTF-16 码元字节数（按 rune 计，非 UTF-8 字节数）+
		// NUL 终止——52626 #25 idx178 Name wide[10]=4 汉字实证。
		msg = binary.LittleEndian.AppendUint32(msg, uint32(len([]rune(prop.Value.WideText))*2+2))
		for _, r := range prop.Value.WideText {
			msg = binary.LittleEndian.AppendUint16(msg, uint16(r))
		}
		return binary.LittleEndian.AppendUint16(msg, 0)
	case 9:
		return binary.LittleEndian.AppendUint64(msg, prop.Value.Obj)
	}
	return msg
}

// NamedProperty 按名构造 IndexedProperty（缺名 panic——编码期暴露）。
func NamedProperty(name string, value PropertyValue) IndexedProperty {
	entry := clientdata.MustPlayerProperty(name)
	return IndexedProperty{Index: entry.Index, Name: name, Value: value}
}

// maxVisualsValue 场景可视表容量（AddObject 注册闸——客户端按此值建
// 角色渲染数据池；2026-11-22 真机定案：230 全量物化 1674 条时
// CreateRoleUserData 池满拒建 1236 条，改 10000）。
const maxVisualsValue uint16 = 10000

// EntrySceneFrame 0x0B 进场帧（5 属性官方形——golden 实证：Name/MaxVisuals/
// ConfigID/Resource/TiGuanGroupCount）。帧头三 u32：[1:5]=登录会话 serial、
// [5:9]/[9:13]=玩家本次进场的 PERSISTID（官服抓包实证，见 Identity 注释）。
func EntrySceneFrame(id Identity, config, resource string) ([]byte, error) {
	nameEntry := clientdata.MustSceneProperty("Name")
	maxVisuals := clientdata.MustSceneProperty("MaxVisuals")
	configEntry := clientdata.MustSceneProperty("ConfigID")
	resourceEntry := clientdata.MustSceneProperty("Resource")
	tiguanEntry := clientdata.MustSceneProperty("TiGuanGroupCount")
	entry := make([]byte, 15, 104)
	entry[0] = 0x0B
	binary.LittleEndian.PutUint32(entry[1:], id.Sequence)
	binary.LittleEndian.PutUint32(entry[5:], id.ObjectID)
	binary.LittleEndian.PutUint32(entry[9:], id.OwnerID)
	binary.LittleEndian.PutUint16(entry[13:], 5)
	// idx178 Name：8B 遮蔽常量（六包恒定，按表宽串编码）。
	entry = binary.LittleEndian.AppendUint16(entry, nameEntry.Index)
	entry = binary.LittleEndian.AppendUint32(entry, 8)
	entry = append(entry, 0x1b, 0x54, 0x50, 0x5b, 0x02, 0x58, 0x00, 0x00)
	entry = binary.LittleEndian.AppendUint16(entry, maxVisuals.Index)
	entry = binary.LittleEndian.AppendUint16(entry, maxVisualsValue)
	entry = appendProperty(entry, IndexedProperty{Index: configEntry.Index, Name: "ConfigID", Value: StringValue(config)})
	entry = appendProperty(entry, IndexedProperty{Index: resourceEntry.Index, Name: "Resource", Value: StringValue(resource)})
	// idx1451 TiGuanGroupCount：u16=1（golden 第 5 属性真机形）。
	entry = binary.LittleEndian.AppendUint16(entry, tiguanEntry.Index)
	entry = binary.LittleEndian.AppendUint16(entry, 1)
	return entry, nil
}

// ExitSceneFrame 0x0C 退出场景（单字节；b-line encodeExitScene 定版）。
// 客户端原生拆除当前场景对象/记录——服务端不逐个发 0x0E，重入簇整段重推。
func ExitSceneFrame() []byte { return []byte{0x0C} }

// ObjectPropertyFrame 0x10 属性帧（单对象一批）。
func ObjectPropertyFrame(objectID, ownerID uint32, isViewObject byte, props []IndexedProperty) ([]byte, error) {
	if len(props) > 0xffff {
		return nil, fmt.Errorf("object property count %d overflows u16", len(props))
	}
	msg := make([]byte, 12, 12+len(props)*8)
	msg[0] = 0x10
	msg[1] = isViewObject
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], uint16(len(props)))
	for _, prop := range props {
		msg = appendProperty(msg, prop)
	}
	return msg, nil
}

// LocationFrame 0x1F 位置帧（25B 无 mode 形）。
func LocationFrame(objectID, ownerID uint32, x, y, z, orient float32) []byte {
	msg := make([]byte, 25)
	msg[0] = 0x1F
	binary.LittleEndian.PutUint32(msg[1:], objectID)
	binary.LittleEndian.PutUint32(msg[5:], ownerID)
	binary.LittleEndian.PutUint32(msg[9:], math.Float32bits(x))
	binary.LittleEndian.PutUint32(msg[13:], math.Float32bits(y))
	binary.LittleEndian.PutUint32(msg[17:], math.Float32bits(z))
	binary.LittleEndian.PutUint32(msg[21:], math.Float32bits(orient))
	return msg
}

// ---- 记录族 ----

// RecordCell 0x11 行的一格（六型载体，b-line recordCell 同构）。
type RecordCell struct {
	Int      *uint32
	Int64    *int64
	Byte     *uint32
	Double   *float64
	Text     *string
	IsObject bool
}

func RecordInt(v uint32) RecordCell     { return RecordCell{Int: &v} }
func RecordInt64(v int64) RecordCell    { return RecordCell{Int64: &v} }
func RecordByte(v uint32) RecordCell    { return RecordCell{Byte: &v} }
func RecordDouble(v float64) RecordCell { return RecordCell{Double: &v} }
func RecordString(v string) RecordCell  { return RecordCell{Text: &v} }

// RecordAddCellsFrame 0x11 整行加（按官方 schema 列型写满；缺尾列补零/
// 空串，超宽报错——b-line serverRecordAddCells 语义）。行号恒 0；要在
// 指定行号插入用 RecordInsertRowFrame。
func RecordAddCellsFrame(objectID, ownerID uint32, recordIndex uint16, cells []RecordCell) ([]byte, error) {
	return recordAddRowFrame(objectID, ownerID, recordIndex, 0, cells)
}

// RecordInsertRowFrame 0x11 在 row 处插入一行（各列按官方 schema 线型写
// 零值）。官服冷却表方言（女号抓包 cooldown_rec 逐帧定版）：先插入空行，
// 再用 RecordGridFrame 逐格写值——
//
//	11 00 [obj][own] 6d 00 | 行号 u16 | 01 00 | 各列零值
func RecordInsertRowFrame(objectID, ownerID uint32, recordIndex, row uint16) ([]byte, error) {
	return recordAddRowFrame(objectID, ownerID, recordIndex, row, nil)
}

// recordAddRowFrame 0x11 行加帧：[11][00][obj u32][own u32][rec u16]
// [起始行 u16][行数 u16=1][行内各列]。
func recordAddRowFrame(objectID, ownerID uint32, recordIndex, row uint16, cells []RecordCell) ([]byte, error) {
	schema, ok := clientdata.RecordSchemaByIndex(int(recordIndex))
	if !ok {
		return nil, fmt.Errorf("record index %d outside official table", recordIndex)
	}
	colTypes := schema.ColumnTypes
	if len(cells) > len(colTypes) {
		return nil, fmt.Errorf("record %q has %d cells, want at most %d", schema.Name, len(cells), len(colTypes))
	}
	msg := make([]byte, 16)
	msg[0] = 0x11
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], row)
	binary.LittleEndian.PutUint16(msg[14:], 1) // 一整行
	for column := range colTypes {
		if column < len(cells) {
			msg = appendRecordCell(msg, colTypes[column], &cells[column])
		} else {
			msg = appendRecordCell(msg, colTypes[column], nil)
		}
	}
	return msg, nil
}

// appendRecordCell 按 schema 列型追加一格（cell 为 nil 或对应字段为 nil 写零值）。
func appendRecordCell(msg []byte, colType byte, cell *RecordCell) []byte {
	switch colType {
	case 3: // int32
		if cell == nil || cell.Int == nil {
			return binary.LittleEndian.AppendUint32(msg, 0)
		}
		return binary.LittleEndian.AppendUint32(msg, *cell.Int)
	case 2: // word
		if cell == nil || cell.Int == nil {
			return binary.LittleEndian.AppendUint16(msg, 0)
		}
		return binary.LittleEndian.AppendUint16(msg, uint16(*cell.Int&0xFFFF))
	case 1: // byte
		if cell == nil || cell.Byte == nil {
			return append(msg, 0)
		}
		return append(msg, byte(*cell.Byte))
	case 4: // int64
		if cell == nil || cell.Int64 == nil {
			return binary.LittleEndian.AppendUint64(msg, 0)
		}
		return binary.LittleEndian.AppendUint64(msg, uint64(*cell.Int64))
	case 6: // f64
		if cell == nil || cell.Double == nil {
			return binary.LittleEndian.AppendUint64(msg, 0)
		}
		return binary.LittleEndian.AppendUint64(msg, math.Float64bits(*cell.Double))
	case 7: // 窄串
		if cell == nil || cell.Text == nil {
			msg = binary.LittleEndian.AppendUint32(msg, 1)
			return append(msg, 0)
		}
		msg = binary.LittleEndian.AppendUint32(msg, uint32(len(*cell.Text)+1))
		msg = append(msg, (*cell.Text)...)
		return append(msg, 0)
	case 8: // 宽串
		if cell == nil || cell.Text == nil {
			msg = binary.LittleEndian.AppendUint32(msg, 2)
			return binary.LittleEndian.AppendUint16(msg, 0)
		}
		msg = binary.LittleEndian.AppendUint32(msg, uint32(len([]rune(*cell.Text))*2+2))
		for _, r := range *cell.Text {
			msg = binary.LittleEndian.AppendUint16(msg, uint16(r))
		}
		return binary.LittleEndian.AppendUint16(msg, 0)
	case 9: // object
		if cell == nil || cell.Int64 == nil {
			return binary.LittleEndian.AppendUint64(msg, 0)
		}
		return binary.LittleEndian.AppendUint64(msg, uint64(*cell.Int64))
	}
	return msg
}

// RecordGridCell 0x13 的一格：列号 + 值（值字段按 schema 列型取用：
// int32/word/byte 列读 Int，int64/object 列读 Int64，f64 列读 Double）。
type RecordGridCell struct {
	Column int
	Value  RecordCell
}

// GridInt / GridInt64 / GridWord 逐格写值的三种常用构造。
func GridInt(column int, value uint32) RecordGridCell {
	return RecordGridCell{Column: column, Value: RecordInt(value)}
}

func GridInt64(column int, value int64) RecordGridCell {
	return RecordGridCell{Column: column, Value: RecordInt64(value)}
}

func GridWord(column int, value uint16) RecordGridCell {
	return RecordGridCell{Column: column, Value: RecordInt(uint32(value))}
}

// RecordGridFrame 0x13 ServerRecordGrid 逐格写（官服冷却表方言，女号抓包
// 逐帧定版——官服每帧只写任意一列的一格）：
//
//	13 00 [obj u32][own u32] [rec u16] [格数 u16] [行 u16] { [列 u8] [按 schema 列型的值] }
//
// 例：cooldown_rec 行 0 写类别 2012 → 13 00 … 6d 00 01 00 00 00 00 dc 07 00 00
// （16B 头 + 列号 00 + int32）。值宽由官方 schema 列型决定；列越界报错。
func RecordGridFrame(objectID, ownerID uint32, recordIndex, row uint16, cells ...RecordGridCell) ([]byte, error) {
	schema, ok := clientdata.RecordSchemaByIndex(int(recordIndex))
	if !ok {
		return nil, fmt.Errorf("record index %d outside official table", recordIndex)
	}
	if len(cells) == 0 {
		return nil, fmt.Errorf("record %q grid frame needs at least one cell", schema.Name)
	}
	for _, cell := range cells {
		if cell.Column < 0 || cell.Column >= len(schema.ColumnTypes) {
			return nil, fmt.Errorf("record %q grid column %d outside %d columns",
				schema.Name, cell.Column, len(schema.ColumnTypes))
		}
	}
	msg := make([]byte, 16, 16+len(cells)*9)
	msg[0] = 0x13
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], uint16(len(cells)))
	binary.LittleEndian.PutUint16(msg[14:], row)
	for _, cell := range cells {
		msg = append(msg, byte(cell.Column))
		msg = appendRecordCell(msg, schema.ColumnTypes[cell.Column], &cell.Value)
	}
	return msg, nil
}

// RecordDelRowFrame 0x12 记录行删除（14B；末 u16 = 记录内 0 基行号）。
// 快捷栏整表重建方言的清空半边：从尾行到 0 倒序删，行号重编号永不
// 参与——客户端引擎侧无论怎样维护行号都不会错位（9yin_server1 真机
// 抓包镜像方言）。
func RecordDelRowFrame(objectID, ownerID uint32, recordIndex, row uint16) []byte {
	msg := make([]byte, 14)
	msg[0] = 0x12
	msg[1] = 0
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	binary.LittleEndian.PutUint16(msg[12:], row)
	return msg
}

// MountSpec 一条挂载计划项（MountPlan 数据源见 mountplan.go）。
type MountSpec struct {
	Name string
	Sub  byte // 0=玩家域（obj/own=玩家身份），2=全局域（obj/own=0）
}

// RecordMountFrame 0x14 记录挂载帧（12B；golden 48 帧全样本定版：
// [0]=0x14 [1]=sub(0=玩家域, 2=全局) [2:6]=obj [6:10]=own [10:12]=rec u16；
// sub≠0 时 obj/own=0）。
func RecordMountFrame(sub byte, objectID, ownerID uint32, recordIndex uint16) []byte {
	msg := make([]byte, 12)
	msg[0] = 0x14
	msg[1] = sub
	if sub != 0 {
		objectID, ownerID = 0, 0
	}
	binary.LittleEndian.PutUint32(msg[2:], objectID)
	binary.LittleEndian.PutUint32(msg[6:], ownerID)
	binary.LittleEndian.PutUint16(msg[10:], recordIndex)
	return msg
}

// ---- 视图族 ----

// ViewProperty 视图行属性（0x15 归档/0x18 行共用；值按表线型）。
type ViewProperty struct {
	Index uint16
	Value PropertyValue
}

// CreateViewFrame 0x15 建视图：[15][u16 vid][u16 cap][u16 n][归档]。
func CreateViewFrame(viewID, capacity uint16, props []ViewProperty) ([]byte, error) {
	if len(props) > 0xffff {
		return nil, fmt.Errorf("view %d property count overflows u16", viewID)
	}
	msg := make([]byte, 7, 7+len(props)*8)
	msg[0] = 0x15
	binary.LittleEndian.PutUint16(msg[1:], viewID)
	binary.LittleEndian.PutUint16(msg[3:], capacity)
	binary.LittleEndian.PutUint16(msg[5:], uint16(len(props)))
	for _, prop := range props {
		msg = appendProperty(msg, IndexedProperty{Index: prop.Index, Value: prop.Value})
	}
	return msg, nil
}

// ViewPropertyFrame 0x17 视图属性帧：[17][u16 vid][u16 n][归档]。
func ViewPropertyFrame(viewID uint16, props []ViewProperty) ([]byte, error) {
	if len(props) > 0xffff {
		return nil, fmt.Errorf("view %d property count overflows u16", viewID)
	}
	msg := make([]byte, 5, 5+len(props)*8)
	msg[0] = 0x17
	binary.LittleEndian.PutUint16(msg[1:], viewID)
	binary.LittleEndian.PutUint16(msg[3:], uint16(len(props)))
	for _, prop := range props {
		msg = appendProperty(msg, IndexedProperty{Index: prop.Index, Value: prop.Value})
	}
	return msg, nil
}

// ViewAddFrame 0x18 视图加行：[18][u16 vid][u16 idx][u16 n][归档]。
func ViewAddFrame(viewID, objectIndex uint16, props []ViewProperty) ([]byte, error) {
	if len(props) > 0xffff {
		return nil, fmt.Errorf("view %d object %d property count overflows u16", viewID, objectIndex)
	}
	msg := make([]byte, 7, 7+len(props)*8)
	msg[0] = 0x18
	binary.LittleEndian.PutUint16(msg[1:], viewID)
	binary.LittleEndian.PutUint16(msg[3:], objectIndex)
	binary.LittleEndian.PutUint16(msg[5:], uint16(len(props)))
	for _, prop := range props {
		msg = appendProperty(msg, IndexedProperty{Index: prop.Index, Value: prop.Value})
	}
	return msg, nil
}
