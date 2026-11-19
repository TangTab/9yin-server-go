package fspublic

import (
	"fmt"
	"strings"
)

// VType 是 CPropData/CRecData 的值类型枚举（is_valid_data_type 0xB2FF0
// 跳表七案定案：合法域 [2,8]，与 proto.Var* 线路码同基）。
type VType byte

// 官方 VTYPE_* 常量。
const (
	VTypeInt     VType = 2
	VTypeInt64   VType = 3
	VTypeFloat   VType = 4
	VTypeDouble  VType = 5
	VTypeString  VType = 6
	VTypeWideStr VType = 7
	VTypeObject  VType = 8
)

// IsValidVType 官方 is_valid_data_type。
func IsValidVType(t VType) bool { return t >= VTypeInt && t <= VTypeObject }

// String 类型名（日志面）。
func (t VType) String() string {
	switch t {
	case VTypeInt:
		return "int"
	case VTypeInt64:
		return "int64"
	case VTypeFloat:
		return "float"
	case VTypeDouble:
		return "double"
	case VTypeString:
		return "string"
	case VTypeWideStr:
		return "widestr"
	case VTypeObject:
		return "object"
	}
	return fmt.Sprintf("VType(%d)", byte(t))
}

// VTypeSize 定长型的线宽（string/widestr 变长，返回 0）。
func VTypeSize(t VType) int {
	switch t {
	case VTypeInt, VTypeFloat:
		return 4
	case VTypeInt64, VTypeDouble, VTypeObject:
		return 8
	}
	return 0
}

// Value 类型化属性值（CPropData 等价：word type + union value）。
type Value struct {
	Type VType
	I64  int64
	F64  float64
	Text string  // String=GBK 字节；WideStr=UTF-16 文本
	Obj  [2]uint32
}

// 快捷构造。
func IntValue(v int32) Value      { return Value{Type: VTypeInt, I64: int64(v)} }
func Int64Value(v int64) Value    { return Value{Type: VTypeInt64, I64: v} }
func FloatValue(v float32) Value  { return Value{Type: VTypeFloat, F64: float64(v)} }
func DoubleValue(v float64) Value { return Value{Type: VTypeDouble, F64: v} }
func StringValue(s string) Value  { return Value{Type: VTypeString, Text: s} }
func WideValue(s string) Value    { return Value{Type: VTypeWideStr, Text: s} }
func ObjectValue(lo, hi uint32) Value {
	return Value{Type: VTypeObject, Obj: [2]uint32{lo, hi}}
}

// PutArchive 按型写入归档（WriteValue 语义）。
func (v Value) PutArchive(a *StoreArchive) error {
	switch v.Type {
	case VTypeInt:
		return a.WriteInt(int32(v.I64))
	case VTypeInt64:
		return a.WriteInt64(v.I64)
	case VTypeFloat:
		return a.WriteFloat(float32(v.F64))
	case VTypeDouble:
		return a.WriteDouble(v.F64)
	case VTypeString:
		return a.WriteString(v.Text)
	case VTypeWideStr:
		return a.WriteWideStr(v.Text)
	case VTypeObject:
		return a.WriteObject(v.Obj[0], v.Obj[1])
	}
	return fmt.Errorf("(Value::PutArchive)invalid type %d", byte(v.Type))
}

// GetArchive 按型读取（ReadValue 语义）。
func GetArchive(a *LoadArchive, t VType) (Value, error) {
	switch t {
	case VTypeInt:
		n, err := a.ReadInt()
		return IntValue(n), err
	case VTypeInt64:
		n, err := a.ReadInt64()
		return Int64Value(n), err
	case VTypeFloat:
		n, err := a.ReadFloat()
		return FloatValue(n), err
	case VTypeDouble:
		n, err := a.ReadDouble()
		return DoubleValue(n), err
	case VTypeString:
		s, err := a.ReadString()
		return StringValue(s), err
	case VTypeWideStr:
		s, err := a.ReadWideStr()
		return WideValue(s), err
	case VTypeObject:
		lo, hi, err := a.ReadObject()
		return ObjectValue(lo, hi), err
	}
	return Value{}, fmt.Errorf("(Value::GetArchive)invalid type %d", byte(t))
}

// Record 是 CRecData 等价：定类型列 + 行主序网格 + 键列索引。
type Record struct {
	RowMax  int     // 官方 row_max（0=无上限）
	Cols    int     // 官方 col_max（[1,0x100]）
	ColType []VType // 每列类型
	Rows    [][]Value
	Keys    []RecKey // 键列（官方 key_data_t{col, enabled}）
}

// RecKey 键列声明。
type RecKey struct {
	Col     int
	Enabled bool
}

// NewRecord 官方 CRecData::CRecData(rowMax, cols)。
func NewRecord(rowMax, cols int) *Record {
	rc := &Record{RowMax: rowMax, Cols: cols}
	rc.ColType = make([]VType, cols)
	return rc
}

// ValidateCols 官方断言域：(cols>0)&&(cols<=MAX_PUBREC_COLS=0x100)。
func ValidateCols(cols int) error {
	if cols <= 0 || cols > 0x100 {
		return fmt.Errorf("(cols > 0) && (cols <= MAX_PUBREC_COLS)")
	}
	return nil
}

// AddRow 追加行（值数必须等于列数，类型须匹配）。
func (r *Record) AddRow(vals []Value) error {
	if len(vals) != r.Cols {
		return fmt.Errorf("(CRecData::AddRowValue)value number error")
	}
	for i, v := range vals {
		if v.Type != r.ColType[i] {
			return fmt.Errorf("(CRecData::AddRowValue)value type not %s", r.ColType[i])
		}
	}
	if r.RowMax > 0 && len(r.Rows) >= r.RowMax {
		return fmt.Errorf("(CPubData::AddRecRow)row number exceed")
	}
	r.Rows = append(r.Rows, append([]Value(nil), vals...))
	return nil
}

// Row 行访问（越界=官方 "row error"）。
func (r *Record) Row(row int) ([]Value, error) {
	if row < 0 || row >= len(r.Rows) {
		return nil, fmt.Errorf("row error")
	}
	return r.Rows[row], nil
}

// SetGrid 单格写（类型匹配门）。
func (r *Record) SetGrid(row, col int, v Value) error {
	if row < 0 || row >= len(r.Rows) {
		return fmt.Errorf("(CRecData::SetRowValue)row error")
	}
	if col < 0 || col >= r.Cols {
		return fmt.Errorf("(CRecData::SetRowValue)column number error")
	}
	if v.Type != r.ColType[col] {
		return fmt.Errorf("(CRecData::SetRowValue)column type not match")
	}
	r.Rows[row][col] = v
	return nil
}

// PubData 是 CPubData 等价：权威公共数据对象（属性 + 记录表）。
type PubData struct {
	Name   string // wchar 名（镜像侧 space→data 主键之一）
	Key    string // char key[0x21]
	UID    string // guid（AddPubData 时 create_guid）
	Save   bool   // 持久化标记（saveOnly 快照筛选用）
	Modify bool   // Closing 批量回存筛选用（GetSave && GetModify）

	attrs   []AttrInfo
	recs    []*RecInfo
	attrIdx map[string]int
	recIdx  map[string]int
}

// AttrInfo 属性条目（attr_info_t）。
type AttrInfo struct {
	Name string
	Type VType
	Save bool
	Val  Value
}

// RecInfo 记录表条目（rec_info_t）。
type RecInfo struct {
	Name string
	Save bool
	Rec  *Record
}

// NewPubData 官方 ctor（name/key 由空间侧赋予）。
func NewPubData(name, key string) *PubData {
	return &PubData{
		Name: name, Key: key,
		attrIdx: make(map[string]int),
		recIdx:  make(map[string]int),
	}
}

// GetAttrCount/GetAttr 官方查询面。
func (d *PubData) GetAttrCount() int { return len(d.attrs) }

func (d *PubData) GetAttr(name string) (*AttrInfo, bool) {
	if i, ok := d.attrIdx[name]; ok {
		return &d.attrs[i], true
	}
	return nil, false
}

// SetAttr upsert（官方 AddAttr 与全部 SetAttr* 复用同一线格式 0x4，
// 权威侧 Add→有则改型改值，无则插）。
func (d *PubData) SetAttr(name string, t VType, v Value, save bool) error {
	if v.Type != t {
		return fmt.Errorf("(CPubData::SetAttr)type not match")
	}
	if i, ok := d.attrIdx[name]; ok {
		d.attrs[i].Type = t
		d.attrs[i].Val = v
		d.attrs[i].Save = save
		return nil
	}
	d.attrIdx[name] = len(d.attrs)
	d.attrs = append(d.attrs, AttrInfo{Name: name, Type: t, Save: save, Val: v})
	return nil
}

// RemoveAttr 删除属性（官方 RemoveAttr）。
func (d *PubData) RemoveAttr(name string) bool {
	i, ok := d.attrIdx[name]
	if !ok {
		return false
	}
	d.attrs = append(d.attrs[:i], d.attrs[i+1:]...)
	delete(d.attrIdx, name)
	for n, idx := range d.attrIdx {
		if idx > i {
			d.attrIdx[n] = idx - 1
		}
	}
	return true
}

// Attrs 全量属性视图（顺序稳定）。
func (d *PubData) Attrs() []AttrInfo { return append([]AttrInfo(nil), d.attrs...) }

// AddRec 建表（重名=官方 "already exists"）。
func (d *PubData) AddRec(name string, rowMax, cols int, save bool) (*Record, error) {
	if err := ValidateCols(cols); err != nil {
		return nil, err
	}
	if _, ok := d.recIdx[name]; ok {
		return nil, fmt.Errorf("(CPubData::AddRec)already exists: %s", name)
	}
	if rowMax < 0 {
		return nil, fmt.Errorf("row_max >= 0")
	}
	rc := NewRecord(rowMax, cols)
	d.recIdx[name] = len(d.recs)
	d.recs = append(d.recs, &RecInfo{Name: name, Save: save, Rec: rc})
	return rc, nil
}

// RemoveRec 删表。
func (d *PubData) RemoveRec(name string) bool {
	i, ok := d.recIdx[name]
	if !ok {
		return false
	}
	d.recs = append(d.recs[:i], d.recs[i+1:]...)
	delete(d.recIdx, name)
	for n, idx := range d.recIdx {
		if idx > i {
			d.recIdx[n] = idx - 1
		}
	}
	return true
}

// GetRec 查表。
func (d *PubData) GetRec(name string) (*RecInfo, bool) {
	if i, ok := d.recIdx[name]; ok {
		return d.recs[i], true
	}
	return nil, false
}

// Recs 全量记录视图。
func (d *PubData) Recs() []RecInfo {
	out := make([]RecInfo, len(d.recs))
	for i, ri := range d.recs {
		out[i] = *ri
	}
	return out
}

// SetRecKey 官方 SetRecKey 门序：列类型须 string/widestr → 列未按键 →
// 表无数据（"rec data exists" 官方串反语义：有数据禁改键）。
func (d *PubData) SetRecKey(recName string, cols []int) error {
	ri, ok := d.GetRec(recName)
	if !ok {
		return fmt.Errorf("(CPubData::SetRecKey)rec no data")
	}
	if len(ri.Rec.Rows) > 0 {
		return fmt.Errorf("(CPubData::SetRecKey)rec data exists")
	}
	for _, c := range cols {
		if c < 0 || c >= ri.Rec.Cols {
			return fmt.Errorf("(CPubData::SetRecKey)column error")
		}
		if t := ri.Rec.ColType[c]; t != VTypeString && t != VTypeWideStr {
			return fmt.Errorf("(CPubData::SetRecKey)column type not string")
		}
	}
	for _, c := range cols {
		ri.Rec.Keys = append(ri.Rec.Keys, RecKey{Col: c, Enabled: true})
	}
	return nil
}

// SetRecColType 官方 SetRecColType（表须为空；列上限门）。
func (d *PubData) SetRecColType(recName string, col int, t VType) error {
	ri, ok := d.GetRec(recName)
	if !ok {
		return fmt.Errorf("(CPubData::SetRecColType)rec no data")
	}
	if len(ri.Rec.Rows) > 0 {
		return fmt.Errorf("(CPubData::SetRecColType)rec data exists")
	}
	if col < 0 || col >= ri.Rec.Cols {
		return fmt.Errorf("(CPubData::SetRecColType)column exceed")
	}
	if !IsValidVType(t) {
		return fmt.Errorf("(CPubData::SetRecColType)type error")
	}
	ri.Rec.ColType[col] = t
	return nil
}

// --- PUB4 blob（CPubData::StoreArchive/LoadArchive）---

// blobMagic 官方 0x50554234 = "PUB4"（与 FsAuctionLogic 持久化同源）。
const blobMagic uint32 = 0x50554234

// StoreArchiveBlob 官方 CPubData::StoreArchive(arc, flag)：
// flag=false 全量（member 快照）；true 只含 save 标记项（store 持久化）。
func (d *PubData) StoreArchiveBlob(saveOnly bool) ([]byte, error) {
	arc := NewStoreArchive(0x400)
	if err := arc.WriteInt(int32(blobMagic)); err != nil {
		return nil, err
	}
	attrPos := arc.Len()
	if err := arc.WriteInt(0); err != nil { // attrCount 占位
		return nil, err
	}
	recPos := arc.Len()
	if err := arc.WriteInt(0); err != nil { // recCount 占位
		return nil, err
	}
	attrN := 0
	for _, at := range d.attrs {
		if saveOnly && !at.Save {
			continue
		}
		if err := arc.WriteString(at.Name); err != nil {
			return nil, err
		}
		if err := arc.WriteByte(byte(at.Type)); err != nil {
			return nil, err
		}
		if err := at.Val.PutArchive(arc); err != nil {
			return nil, err
		}
		attrN++
	}
	recN := 0
	for _, ri := range d.recs {
		if saveOnly && !ri.Save {
			continue
		}
		rc := ri.Rec
		if err := arc.WriteString(ri.Name); err != nil {
			return nil, err
		}
		if err := arc.WriteInt(int32(rc.RowMax)); err != nil {
			return nil, err
		}
		if err := arc.WriteInt(int32(len(rc.Rows))); err != nil {
			return nil, err
		}
		if err := arc.WriteInt(int32(rc.Cols)); err != nil {
			return nil, err
		}
		if err := arc.WriteInt(int32(len(rc.Keys))); err != nil {
			return nil, err
		}
		for _, t := range rc.ColType {
			if err := arc.WriteByte(byte(t)); err != nil {
				return nil, err
			}
		}
		for _, k := range rc.Keys {
			if err := arc.WriteInt(int32(k.Col)); err != nil {
				return nil, err
			}
			b := byte(0)
			if k.Enabled {
				b = 1
			}
			if err := arc.WriteByte(b); err != nil {
				return nil, err
			}
		}
		for _, row := range rc.Rows {
			for c, v := range row {
				if err := v.PutArchive(arc); err != nil {
					return nil, err
				}
				_ = c
			}
		}
		recN++
	}
	// 回填占位
	if err := arc.PatchInt(attrPos, int32(attrN)); err != nil {
		return nil, err
	}
	if err := arc.PatchInt(recPos, int32(recN)); err != nil {
		return nil, err
	}
	return arc.Bytes(), nil
}

// LoadArchiveBlob 官方 CPubData::LoadArchive：校验序 magic→列域→
// 列型合法→键列型→已有表全等否则 LoadRecDiscard 整块跳过。
func (d *PubData) LoadArchiveBlob(blob []byte) error {
	a := NewLoadArchive(blob)
	magic, err := a.ReadInt()
	if err != nil {
		return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
	}
	if uint32(magic) != blobMagic {
		return fmt.Errorf("(CPubData::LoadArchive)version not match")
	}
	attrN, err := a.ReadInt()
	if err != nil {
		return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
	}
	recN, err := a.ReadInt()
	if err != nil {
		return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
	}
	for i := 0; i < int(attrN); i++ {
		name, err := a.ReadString()
		if err != nil {
			return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
		}
		tb, err := a.ReadByte()
		if err != nil {
			return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
		}
		t := VType(tb)
		if !IsValidVType(t) {
			return fmt.Errorf("(CPubData::LoadArchive)attr type error")
		}
		if at, ok := d.GetAttr(name); ok && at.Type != t {
			return fmt.Errorf("(CPubData::LoadArchive)attr type not match")
		}
		v, err := GetArchive(a, t)
		if err != nil {
			return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
		}
		save := false
		if at, ok := d.GetAttr(name); ok {
			save = at.Save
		}
		if err := d.SetAttr(name, t, v, save); err != nil {
			return fmt.Errorf("(CPubData::LoadArchive)add attr error")
		}
	}
	for i := 0; i < int(recN); i++ {
		name, err := a.ReadString()
		if err != nil {
			return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
		}
		rowMax, err := a.ReadInt()
		if err != nil {
			return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
		}
		rows, err := a.ReadInt()
		if err != nil {
			return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
		}
		cols, err := a.ReadInt()
		if err != nil {
			return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
		}
		if cols < 1 || cols > 0x100 {
			return fmt.Errorf("(CPubData::LoadArchive)rec size error")
		}
		keyN, err := a.ReadInt()
		if err != nil {
			return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
		}
		if int(keyN) > int(cols) {
			return fmt.Errorf("(CPubData::LoadArchive)rec size error")
		}
		colTypes := make([]VType, cols)
		for c := range colTypes {
			tb, err := a.ReadByte()
			if err != nil {
				return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
			}
			if !IsValidVType(VType(tb)) {
				return fmt.Errorf("(CPubData::LoadArchive)rec type error")
			}
			colTypes[c] = VType(tb)
		}
		keys := make([]RecKey, keyN)
		for k := range keys {
			col, err := a.ReadInt()
			if err != nil {
				return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
			}
			en, err := a.ReadByte()
			if err != nil {
				return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
			}
			if t := colTypes[col]; t != VTypeString && t != VTypeWideStr {
				return fmt.Errorf("(CPubData::LoadArchive)col key not string")
			}
			keys[k] = RecKey{Col: int(col), Enabled: en != 0}
		}
		ri, exists := d.GetRec(name)
		if exists {
			rc := ri.Rec
			if rc.RowMax != int(rowMax) || rc.Cols != int(cols) || len(rc.Keys) != int(keyN) {
				if err := discardRecRows(a, colTypes, int(rows)); err != nil {
					return fmt.Errorf("(CPubData::LoadRecDiscard)rec type invalid")
				}
				continue
			}
			for c := range colTypes {
				if rc.ColType[c] != colTypes[c] {
					if err := discardRecRows(a, colTypes, int(rows)); err != nil {
						return fmt.Errorf("(CPubData::LoadRecDiscard)rec type invalid")
					}
					continue
				}
			}
		} else {
			save := false
			rc, err := d.AddRec(name, int(rowMax), int(cols), save)
			if err != nil {
				return fmt.Errorf("(CPubData::LoadArchive)add rec error: %v", err)
			}
			copy(rc.ColType, colTypes)
			rc.Keys = keys
			ri, _ = d.GetRec(name)
		}
		rc := ri.Rec
		for r := 0; r < int(rows); r++ {
			row := make([]Value, cols)
			for c := 0; c < int(cols); c++ {
				v, err := GetArchive(a, colTypes[c])
				if err != nil {
					return fmt.Errorf("(CPubData::LoadArchive)loading error: %v", err)
				}
				row[c] = v
			}
			if rc.RowMax > 0 && len(rc.Rows) >= rc.RowMax {
				return fmt.Errorf("(CPubData::LoadArchive)rec size not match")
			}
			rc.Rows = append(rc.Rows, row)
		}
	}
	return nil
}

// discardRecRows 官方 LoadRecDiscard：按列型跳过 rows 行不读。
func discardRecRows(a *LoadArchive, colTypes []VType, rows int) error {
	for r := 0; r < rows; r++ {
		for _, t := range colTypes {
			if _, err := GetArchive(a, t); err != nil {
				return err
			}
		}
	}
	return nil
}

// Summary 调试摘要。
func (d *PubData) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "data %q key=%q uid=%s attrs=%d recs=%d",
		d.Name, d.Key, d.UID, len(d.attrs), len(d.recs))
	return b.String()
}
