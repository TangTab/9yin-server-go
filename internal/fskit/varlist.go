package fskit

import "strconv"

// VarKind 对齐官方 IVar 的类型槽（Debug 注册表里属性 type=6 为 string）。
type VarKind int

const (
	VarInt    VarKind = 1
	VarFloat  VarKind = 2
	VarString VarKind = 6
)

// Var 是官方 IVar 的值载体。
type Var struct {
	Kind   VarKind
	Int    int64
	Float  float64
	String string
}

// VarList 复刻官方 IVarList（public\varlist.h）的脚本绑定形态：
// mid 函数按位取参、按位追加结果。vtable 对应关系见全解 §6.3。
type VarList struct {
	vars []Var
}

func (l *VarList) Count() int { return len(l.vars) }

func (l *VarList) Clear() { l.vars = l.vars[:0] }

func (l *VarList) AddInt(v int64)      { l.vars = append(l.vars, Var{Kind: VarInt, Int: v}) }
func (l *VarList) AddFloat(v float64)  { l.vars = append(l.vars, Var{Kind: VarFloat, Float: v}) }
func (l *VarList) AddString(s string)  { l.vars = append(l.vars, Var{Kind: VarString, String: s}) }

// GetKind 官方 TVarGetter<T> 前置类型检查用。
func (l *VarList) GetKind(i int) VarKind {
	if i < 0 || i >= len(l.vars) {
		return 0
	}
	return l.vars[i].Kind
}

func (l *VarList) GetInt(i int) int64 {
	if i < 0 || i >= len(l.vars) {
		return 0
	}
	return l.vars[i].Int
}

func (l *VarList) GetFloat(i int) float64 {
	if i < 0 || i >= len(l.vars) {
		return 0
	}
	return l.vars[i].Float
}

func (l *VarList) GetString(i int) string {
	if i < 0 || i >= len(l.vars) {
		return ""
	}
	return l.vars[i].String
}

// Assign 复刻官方 vtable+0x08（拷贝 other[start, start+count)）。越界静默收敛。
func (l *VarList) Assign(other *VarList, start, count int) {
	l.vars = l.vars[:0]
	if other == nil {
		return
	}
	if start < 0 {
		start = 0
	}
	if start+count > len(other.vars) {
		count = len(other.vars) - start
	}
	if count <= 0 {
		return
	}
	l.vars = append(l.vars, other.vars[start:start+count]...)
}

// cAtoi 复刻 CRT atoi：跳过前导空白、可选符号、解析最长数字前缀，无有效数字为 0。
func cAtoi(s string) int {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\v' || s[i] == '\f' || s[i] == '\r') {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	var v int64
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		v = v*10 + int64(s[i]-'0')
		i++
	}
	if neg {
		v = -v
	}
	return int(v)
}

// cAtof 复刻 CRT atof：最长浮点前缀（含 inf/nan 的近似处理从简，走 strconv 失败回退 atoi）。
func cAtof(s string) float64 {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\v' || s[i] == '\f' || s[i] == '\r') {
		i++
	}
	j := i
	if j < len(s) && (s[j] == '+' || s[j] == '-') {
		j++
	}
	for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.') {
		j++
	}
	// 指数部分
	if j < len(s) && (s[j] == 'e' || s[j] == 'E') {
		k := j + 1
		if k < len(s) && (s[k] == '+' || s[k] == '-') {
			k++
		}
		for k < len(s) && s[k] >= '0' && s[k] <= '9' {
			k++
		}
		if k > j+1 && s[j+1] >= '0' && s[j+1] <= '9' || (k > j+2 && (s[j+1] == '+' || s[j+1] == '-')) {
			j = k
		}
	}
	if j == i {
		return 0
	}
	if v, err := strconv.ParseFloat(s[i:j], 64); err == nil {
		return v
	}
	return float64(cAtoi(s[i:j]))
}

// formatG 复刻 printf "%g"（六位有效数字，去多余零）。
func formatG(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 32)
}
