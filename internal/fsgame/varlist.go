package fsgame

// PersistID 复刻官方 UPERSISTID：对象弱引用句柄（低 32 位 id + 高 32 位
// serial 防悬挂），贯穿全部回调签名。0 = NullPersistID。
type PersistID uint64

// NullPersistID 空 ID（官方 PERSISTID() 默认构造）。
const NullPersistID PersistID = 0

// ID 返回对象编号（低 32 位）。
func (p PersistID) ID() uint32 { return uint32(p) }

// Serial 返回防悬挂序号（高 32 位）。
func (p PersistID) Serial() uint32 { return uint32(p >> 32) }

// Valid ID 是否可用。
func (p PersistID) Valid() bool { return p != NullPersistID }

// MakePersistID 由编号与序号组装。
func MakePersistID(id, serial uint32) PersistID {
	return PersistID(id) | (PersistID(serial) << 32)
}

// Var 是回调/属性承载值的联合面（对应官方 IVar 的九型：
// int/int64/float/double/string/widestr/object/binary + bool 辅助）。
type Var interface{}

// VarList 复刻官方 TVarList<8>/IVarList：回调与命令的参数列。
// 官方语义：值类型自描述（GetType/GetCount），流式追加（operator<<）。
type VarList []Var

// Append 追加一个参数（等价 IVarList::operator<<）。
func (l *VarList) Append(v Var) *VarList {
	*l = append(*l, v)
	return l
}

// Len 参数个数。
func (l VarList) Len() int { return len(l) }

// Int 取第 i 个参数为 int32（越界/类型不符返回 0，等价官方缺省值语义）。
func (l VarList) Int(i int) int32 {
	if i < 0 || i >= len(l) {
		return 0
	}
	switch v := l[i].(type) {
	case int32:
		return v
	case int64:
		return int32(v)
	case int:
		return int32(v)
	case float64:
		return int32(v)
	case float32:
		return int32(v)
	}
	return 0
}

// Int64 取第 i 个参数为 int64。
func (l VarList) Int64(i int) int64 {
	if i < 0 || i >= len(l) {
		return 0
	}
	switch v := l[i].(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	}
	return 0
}

// Float 取第 i 个参数为 float32。
func (l VarList) Float(i int) float32 {
	if i < 0 || i >= len(l) {
		return 0
	}
	switch v := l[i].(type) {
	case float32:
		return v
	case float64:
		return float32(v)
	case int32:
		return float32(v)
	}
	return 0
}

// Double 取第 i 个参数为 float64。
func (l VarList) Double(i int) float64 {
	if i < 0 || i >= len(l) {
		return 0
	}
	switch v := l[i].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int32:
		return float64(v)
	}
	return 0
}

// String 取第 i 个参数为 string。
func (l VarList) String(i int) string {
	if i < 0 || i >= len(l) {
		return ""
	}
	if s, ok := l[i].(string); ok {
		return s
	}
	return ""
}

// Object 取第 i 个参数为 PersistID。
func (l VarList) Object(i int) PersistID {
	if i < 0 || i >= len(l) {
		return NullPersistID
	}
	if p, ok := l[i].(PersistID); ok {
		return p
	}
	return NullPersistID
}
