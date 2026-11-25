package script

// value.go nx_* 值域与对象句柄：Go ↔ Lua 的桥接类型。
//
// 官方引擎的对象是活引用（nx_exists 直接判活）；本实现一律传句柄
// Handle{类别, 宿主标识}，判活 = 经 World.Lookup 查宿主——句柄失效即
// nx_exists 为假，脚本侧语义不变。

import (
	"fmt"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

// Kind 对象类别（define.lua 的 TYPE_* 位值）。
type Kind int

// 对象类别。
const (
	KindNone   Kind = 0
	KindScene  Kind = 1  // TYPE_SCENE
	KindPlayer Kind = 2  // TYPE_PLAYER
	KindNPC    Kind = 4  // TYPE_NPC
	KindItem   Kind = 8  // TYPE_ITEM
	KindHelper Kind = 16 // TYPE_HELPER（技能/buff 等辅助对象）
)

// Helper 辅助对象子类（技能、buff——同为 TYPE_HELPER，以子类区分）。
type Helper int

// 辅助对象子类。
const (
	HelperNone Helper = iota
	HelperSkill
	HelperBuffer
)

// Handle 对象句柄。
type Handle struct {
	Kind   Kind
	ID     uint64
	Helper Helper
}

// NullHandle 空句柄（nx_null / nx_null_obj）。
var NullHandle = Handle{}

// Valid 是否非空句柄。
func (h Handle) Valid() bool { return h.Kind != KindNone || h.ID != 0 }

func (h Handle) String() string {
	return fmt.Sprintf("handle(%d:%d:%d)", h.Kind, h.Helper, h.ID)
}

// ValueKind 值类别。
type ValueKind int

// 值类别。
const (
	KindNil ValueKind = iota
	KindBool
	KindInt
	KindFloat
	KindString
	KindWide
	KindObject
)

// Value nx_* 值：整数/浮点/窄串/宽串/对象/布尔/空。
type Value struct {
	Kind ValueKind
	Flag bool
	Int  int64
	Num  float64
	Text string
	Obj  Handle
}

// 构造器。
func Nil() Value            { return Value{} }
func Bool(b bool) Value     { return Value{Kind: KindBool, Flag: b} }
func Int(v int64) Value     { return Value{Kind: KindInt, Int: v, Num: float64(v)} }
func Float(v float64) Value { return Value{Kind: KindFloat, Num: v} }
func String(s string) Value { return Value{Kind: KindString, Text: s} }
func Wide(s string) Value   { return Value{Kind: KindWide, Text: s} }
func Ref(h Handle) Value    { return Value{Kind: KindObject, Obj: h} }

// Number 数值视图（串按数字解析，失败为 0——nx_number 口径）。
func (v Value) Number() float64 {
	switch v.Kind {
	case KindInt, KindFloat:
		return v.Num
	case KindBool:
		if v.Flag {
			return 1
		}
	case KindString, KindWide:
		n, _ := strconv.ParseFloat(v.Text, 64)
		return n
	}
	return 0
}

// IntValue 整数视图（截断）。
func (v Value) IntValue() int64 { return int64(v.Number()) }

// Truthy Lua 真值（nil/false 为假）。
func (v Value) Truthy() bool {
	return v.Kind != KindNil && !(v.Kind == KindBool && !v.Flag)
}

// ---- Lua 转换 ----

const objectMeta = "nx_object"

// toLua 值 → Lua 栈值。
func toLua(L *lua.LState, v Value) lua.LValue {
	switch v.Kind {
	case KindNil:
		return lua.LNil
	case KindBool:
		return lua.LBool(v.Flag)
	case KindInt:
		return lua.LNumber(v.Int)
	case KindFloat:
		return lua.LNumber(v.Num)
	case KindString, KindWide:
		return lua.LString(v.Text)
	case KindObject:
		ud := L.NewUserData()
		ud.Value = v.Obj
		if mt := L.GetTypeMetatable(objectMeta); mt != lua.LNil {
			ud.Metatable = mt
		}
		return ud
	}
	return lua.LNil
}

// fromLua Lua 栈值 → 值（表不支持，按空处理）。
func fromLua(L *lua.LState, v lua.LValue) Value {
	switch value := v.(type) {
	case lua.LNumber:
		if value == lua.LNumber(float64(int64(value))) {
			return Int(int64(value))
		}
		return Float(float64(value))
	case lua.LString:
		return String(string(value))
	case lua.LBool:
		return Bool(bool(value))
	case *lua.LUserData:
		if h, ok := value.Value.(Handle); ok {
			return Ref(h)
		}
	}
	return Nil()
}

// argValue 取第 idx 个参数（idx>0 自栈底数，idx<0 自栈顶数）。
func argValue(L *lua.LState, idx int) Value {
	if L.GetTop() < abs(idx) {
		return Nil()
	}
	return fromLua(L, L.Get(idx))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
