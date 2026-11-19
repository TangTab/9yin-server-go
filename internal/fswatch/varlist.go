package fswatch

import (
	"fmt"
	"strconv"
	"strings"
)

// VarKind 对齐官方 TVarList 的类型槽（text_encode_msg 跳转表
// 0x1005ed6c 共 9 型，本协议面用到 5 型）。
type VarKind int

const (
	KindInt VarKind = iota + 1
	KindInt64
	KindFloat
	KindString
	KindWideString
	KindUserData
)

// Var 是官方 IVar 的值载体。
type Var struct {
	Kind  VarKind
	Int   int64
	Float float64
	Str   string // KindString/KindWideString
	Data  []byte // KindUserData
}

// VarList 官方 IVarList 的 Go 等价；AddXxx 对应 TVarList::AddInt/AddDouble/
// AddString/AddUserData（thunk 0x514fa/0x5090b/0x513a6/0x50046）。
type VarList struct {
	vars []Var
}

func (l *VarList) Count() int { return len(l.vars) }

func (l *VarList) AddInt(v int64)     { l.vars = append(l.vars, Var{Kind: KindInt, Int: v}) }
func (l *VarList) AddFloat(v float64) { l.vars = append(l.vars, Var{Kind: KindFloat, Float: v}) }
func (l *VarList) AddString(s string) {
	l.vars = append(l.vars, Var{Kind: KindString, Str: s})
}
func (l *VarList) AddUserData(b []byte) {
	l.vars = append(l.vars, Var{Kind: KindUserData, Data: b})
}

func (l *VarList) KindAt(i int) VarKind {
	if i < 0 || i >= len(l.vars) {
		return 0
	}
	return l.vars[i].Kind
}

func (l *VarList) IntAt(i int) int64 {
	if i < 0 || i >= len(l.vars) {
		return 0
	}
	return l.vars[i].Int
}

func (l *VarList) FloatAt(i int) float64 {
	if i < 0 || i >= len(l.vars) {
		return 0
	}
	return l.vars[i].Float
}

func (l *VarList) StringAt(i int) string {
	if i < 0 || i >= len(l.vars) {
		return ""
	}
	return l.vars[i].Str
}

func (l *VarList) DataAt(i int) []byte {
	if i < 0 || i >= len(l.vars) {
		return nil
	}
	return l.vars[i].Data
}

const hexUpper = "0123456789ABCDEF" // 官方码表 0x1011124c 逐字节

// escapeTextBody 官方 text_encode_msg 串/宽串转义体（0x5e869/0x5e9b0）：
// c<=0x20 || c>=0x7f || c=='%' → "\x"+2 位大写 hex；'\\' → "\\\\"；
// 其余可打印原样。
func escapeTextBody(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c <= 0x20 || c >= 0x7f || c == '%':
			b.WriteString(`\x`)
			b.WriteByte(hexUpper[c>>4])
			b.WriteByte(hexUpper[c&0xf])
		case c == '\\':
			b.WriteString(`\\`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// encodeVar 官方 text_encode_msg 单元素编码：int "%d"、float "%f"、
// string "$"+体、widestr "#"+体、userdata "*"+大写HEX（0x5eb68）。
func encodeVar(v Var) string {
	switch v.Kind {
	case KindInt, KindInt64:
		return strconv.FormatInt(v.Int, 10)
	case KindFloat:
		return fmt.Sprintf("%f", v.Float)
	case KindString:
		return "$" + escapeTextBody(v.Str)
	case KindWideString:
		return "#" + escapeTextBody(v.Str)
	case KindUserData:
		var b strings.Builder
		b.WriteByte('*')
		for _, c := range v.Data {
			b.WriteByte(hexUpper[c>>4])
			b.WriteByte(hexUpper[c&0xf])
		}
		return b.String()
	}
	return ""
}

// EncodeText 官方 text_encode_msg：元素以空格 join（两遍测长/编码合一）。
func (l *VarList) EncodeText() string {
	parts := make([]string, 0, len(l.vars))
	for _, v := range l.vars {
		parts = append(parts, encodeVar(v))
	}
	return strings.Join(parts, " ")
}

// hexVal 官方 tohex 反查；容错大小写（官方只产出大写）。
func hexVal(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	}
	return 0, false
}

// unescapeBody 官方 text_append_word 反转义（0x5d6cb 起）：
// "\\"→'\'、"\x"+2hex → 字节、"\u"+4hex → UCS2 码元。
func unescapeBody(s string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i >= len(s) {
			return "", false
		}
		switch s[i] {
		case '\\':
			b.WriteByte('\\')
		case 'x':
			if i+2 >= len(s) {
				return "", false
			}
			hi, ok1 := hexVal(s[i+1])
			lo, ok2 := hexVal(s[i+2])
			if !ok1 || !ok2 {
				return "", false
			}
			b.WriteByte(hi<<4 | lo)
			i += 2
		case 'u':
			if i+4 >= len(s) {
				return "", false
			}
			var r rune
			for k := 1; k <= 4; k++ {
				h, ok := hexVal(s[i+k])
				if !ok {
					return "", false
				}
				r = r<<4 | rune(h)
			}
			b.WriteRune(r)
			i += 4
		default:
			return "", false
		}
	}
	return b.String(), true
}

// decodeHexBody 官方 "*" 前缀 userdata 大写 HEX 解码。
func decodeHexBody(s string) ([]byte, bool) {
	if len(s)%2 != 0 {
		return nil, false
	}
	out := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		hi, ok1 := hexVal(s[i])
		lo, ok2 := hexVal(s[i+1])
		if !ok1 || !ok2 {
			return nil, false
		}
		out[i/2] = hi<<4 | lo
	}
	return out, true
}

// looksNumber 官方 text_append_word 数字首字符判定（0x5d495）：
// [0-9]、'.'、'-'。
func looksNumber(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return (c >= '0' && c <= '9') || c == '.' || c == '-'
}

// decodeWord 官方 text_append_word 单 token 类型分派（0x5d440/0x5d653）。
func decodeWord(w string) (Var, bool) {
	switch w[0] {
	case '$':
		s, ok := unescapeBody(w[1:])
		return Var{Kind: KindString, Str: s}, ok
	case '#':
		s, ok := unescapeBody(w[1:])
		return Var{Kind: KindWideString, Str: s}, ok
	case '*':
		d, ok := decodeHexBody(w[1:])
		return Var{Kind: KindUserData, Data: d}, ok
	}
	if looksNumber(w) {
		if i, err := strconv.ParseInt(w, 10, 64); err == nil {
			return Var{Kind: KindInt, Int: i}, true
		}
		if f, err := strconv.ParseFloat(w, 64); err == nil {
			return Var{Kind: KindFloat, Float: f}, true
		}
	}
	return Var{Kind: KindString, Str: w}, true
}

// DecodeText 官方 text_decode_msg（0x5d2d0）：空格分词 + 类型推断。
func DecodeText(msg string) (*VarList, bool) {
	l := &VarList{}
	for _, w := range strings.Split(msg, " ") {
		if w == "" {
			continue
		}
		v, ok := decodeWord(w)
		if !ok {
			return nil, false
		}
		l.vars = append(l.vars, v)
	}
	return l, true
}
