package luascript

import (
	"strconv"
	"strings"

	"github.com/iceisfun/golua/v2/vm"
)

// goValue 把 Lua 值转为适合 fmt 的 Go 值（用于 game.log）。
func goValue(v vm.Value) any {
	switch {
	case v.IsNil():
		return "nil"
	case v.IsBool():
		return v.AsBool()
	case v.IsInt():
		return v.AsInt()
	case v.IsFloat():
		return v.AsFloat()
	case v.IsString():
		return v.AsString()
	default:
		return v.String()
	}
}

// valueString 把返回值序列化为可读文本（用于 Eval 回显）。表递归到
// 两层、最多 16 项；函数与 userdata 显示占位符。
func valueString(v vm.Value, depth int) string {
	switch {
	case v.IsNil():
		return "nil"
	case v.IsBool():
		return strconv.FormatBool(v.AsBool())
	case v.IsInt():
		return strconv.FormatInt(v.AsInt(), 10)
	case v.IsFloat():
		return strconv.FormatFloat(v.AsFloat(), 'g', -1, 64)
	case v.IsString():
		return v.AsString()
	case v.IsTable():
		if depth >= 2 {
			return "table: ..."
		}
		table := v.AsTable()
		if table == nil {
			return "table"
		}
		return tableString(table, depth)
	default:
		return v.String()
	}
}

func tableString(t vm.LuaTable, depth int) string {
	var parts []string
	count := 0
	key := vm.Nil
	for {
		nextKey, value, err := t.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		if count >= 16 {
			parts = append(parts, "...")
			break
		}
		count++
		parts = append(parts, valueString(nextKey, depth+1)+"="+valueString(value, depth+1))
		key = nextKey
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
