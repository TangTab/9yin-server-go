package fsgame

import (
	"fmt"
	"time"
)

// RowOp 记录行操作日志（对应 CGameObject::SendRec* 下发面与
// internal/prop 包的行操作设计：行级增删改不整表重发）。
type RowOp struct {
	Record string
	Op     string // append / modify / delete / clear / setrow
	Row    int
	Values VarList
}

// Record 复刻 CGameObject 的 Record：对象上的二维表，行级操作。
type Record struct {
	def  RecordDef
	rows []VarList
	ops  []RowOp
}

// Def 记录定义。
func (r *Record) Def() RecordDef { return r.def }

// Rows 当前行数。
func (r *Record) Rows() int { return len(r.rows) }

// Row 取整行（越界返回 nil）。
func (r *Record) Row(i int) VarList {
	if i < 0 || i >= len(r.rows) {
		return nil
	}
	return r.rows[i]
}

func (r *Record) log(op string, row int, vals VarList) {
	r.ops = append(r.ops, RowOp{Record: r.def.Name, Op: op, Row: row, Values: vals})
}

// AppendRow 行尾追加（官方 AddRecordRow）。
func (r *Record) AppendRow(vals VarList) int {
	if len(r.rows) >= r.def.MaxRows {
		return -1
	}
	r.rows = append(r.rows, vals)
	i := len(r.rows) - 1
	r.log("append", i, vals)
	return i
}

// ModifyRow 改行（官方 SetRecordRowValue）。
func (r *Record) ModifyRow(i int, vals VarList) bool {
	if i < 0 || i >= len(r.rows) {
		return false
	}
	r.rows[i] = vals
	r.log("modify", i, vals)
	return true
}

// DeleteRow 删行（官方 RemoveRecordRow）。
func (r *Record) DeleteRow(i int) bool {
	if i < 0 || i >= len(r.rows) {
		return false
	}
	r.rows = append(r.rows[:i], r.rows[i+1:]...)
	r.log("delete", i, nil)
	return true
}

// Clear 清表（官方 ClearRecord）。
func (r *Record) Clear() {
	r.rows = nil
	r.log("clear", -1, nil)
}

// DrainOps 取走并清空行操作日志（供 sync 层转协议下发）。
func (r *Record) DrainOps() []RowOp {
	ops := r.ops
	r.ops = nil
	return ops
}

// GameObject 复刻 CGameObject（IGameObj 120 方法）的登录闭环子面：
// 类型化属性表 + Record 集 + 子对象树 + 存档。
type GameObject struct {
	id       PersistID
	class    *ClassDef
	props    map[string]Var
	records  map[string]*Record
	parent   PersistID
	children []PersistID

	created time.Time
}

// ID 对象句柄。
func (o *GameObject) ID() PersistID { return o.id }

// ClassName 类名。
func (o *GameObject) ClassName() string { return o.class.Name }

// Class 类定义。
func (o *GameObject) Class() *ClassDef { return o.class }

// Set 写属性（官方 Set{Int,Int64,Float,Double,String,WideStr,Object} 按类型
// 分槽；Go 面按名走 schema 校验，未知属性报错——官方属 schema 索引越界）。
// 变更经 Kernel.setProp 触发临界回调，勿直用。
func (o *GameObject) Set(name string, v Var) error {
	p, ok := o.class.Prop(name)
	if !ok {
		return fmt.Errorf("fsgame: 对象 %d 类 %s 无属性 %s", uint64(o.id), o.class.Name, name)
	}
	o.props[name] = coerceVar(p.Type, v)
	return nil
}

// Get 读属性（缺省返回类型零值）。
func (o *GameObject) Get(name string) Var {
	p, ok := o.class.Prop(name)
	if !ok {
		return nil
	}
	if v, ok := o.props[name]; ok {
		return v
	}
	return zeroVar(p.Type)
}

// GetInt 读 int 型属性。
func (o *GameObject) GetInt(name string) int32 {
	if v, ok := o.Get(name).(int32); ok {
		return v
	}
	return 0
}

// GetString 读 string 型属性。
func (o *GameObject) GetString(name string) string {
	if v, ok := o.Get(name).(string); ok {
		return v
	}
	return ""
}

// Record 按名取记录（未定义返回 nil）。
func (o *GameObject) Record(name string) *Record {
	return o.records[name]
}

// Records 全部记录名。
func (o *GameObject) RecordNames() []string {
	out := make([]string, 0, len(o.records))
	for _, rd := range o.class.Records {
		if _, ok := o.records[rd.Name]; ok {
			out = append(out, rd.Name)
		}
	}
	return out
}

// Parent 父对象。
func (o *GameObject) Parent() PersistID { return o.parent }

// Children 子对象句柄集。
func (o *GameObject) Children() []PersistID { return o.children }

// Created 物化时间（OnLoad/OnCreate 期）。
func (o *GameObject) Created() time.Time { return o.created }

// Archive 产出存档面快照（save=true 的属性与记录；对应
// StoreArchive/SaveChunk 的 chunk 内容源）。
func (o *GameObject) Archive() (map[string]Var, map[string][]VarList) {
	props := make(map[string]Var)
	for _, p := range o.class.Props {
		if !p.Save {
			continue
		}
		if v, ok := o.props[p.Name]; ok {
			props[p.Name] = v
		}
	}
	recs := make(map[string][]VarList)
	for _, rd := range o.class.Records {
		if !rd.Save {
			continue
		}
		if r := o.records[rd.Name]; r != nil && r.Rows() > 0 {
			rows := make([]VarList, len(r.rows))
			copy(rows, r.rows)
			recs[rd.Name] = rows
		}
	}
	return props, recs
}

// newGameObject 按 schema 物化对象（官方 CreateObject/CreateFromConfig）。
func newGameObject(id PersistID, class *ClassDef) *GameObject {
	o := &GameObject{
		id:      id,
		class:   class,
		props:   make(map[string]Var, len(class.Props)),
		records: make(map[string]*Record, len(class.Records)),
		created: time.Now(),
	}
	for _, rd := range class.Records {
		o.records[rd.Name] = &Record{def: rd}
	}
	return o
}

// coerceVar 把 Go 值收进 schema 类型（宽松转换，等价 IVar 类型系统的赋值面）。
func coerceVar(t PropType, v Var) Var {
	switch t {
	case TypeByte, TypeWord, TypeInt, TypeDWord:
		switch x := v.(type) {
		case int32:
			return x
		case int64:
			return int32(x)
		case int:
			return int32(x)
		case float64:
			return int32(x)
		case float32:
			return int32(x)
		case bool:
			if x {
				return int32(1)
			}
			return int32(0)
		}
		return int32(0)
	case TypeInt64:
		switch x := v.(type) {
		case int64:
			return x
		case int32:
			return int64(x)
		case int:
			return int64(x)
		}
		return int64(0)
	case TypeFloat:
		switch x := v.(type) {
		case float32:
			return x
		case float64:
			return float32(x)
		case int32:
			return float32(x)
		}
		return float32(0)
	case TypeDouble, TypeUnkPrec6:
		switch x := v.(type) {
		case float64:
			return x
		case float32:
			return float64(x)
		case int32:
			return float64(x)
		}
		return float64(0)
	case TypeString, TypeWideStr:
		if s, ok := v.(string); ok {
			return s
		}
		return ""
	case TypeObject:
		if p, ok := v.(PersistID); ok {
			return p
		}
		return NullPersistID
	}
	return v
}

// zeroVar 类型零值（官方 Query 缺省语义）。
func zeroVar(t PropType) Var {
	switch t {
	case TypeInt64:
		return int64(0)
	case TypeFloat:
		return float32(0)
	case TypeDouble, TypeUnkPrec6:
		return float64(0)
	case TypeString, TypeWideStr:
		return ""
	case TypeObject:
		return NullPersistID
	case TypeByte, TypeWord, TypeInt, TypeDWord, TypeBool:
		return int32(0)
	}
	return nil
}
