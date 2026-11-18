package fsdumper

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

// textValue 是 text 方言 VarList 的单元素。
type textValue struct {
	kind byte // 'i' int / 'l' int64 / 'd' double / 's' 窄串 / 'w' 宽串
	n    int64
	f    float64
	s    string
}

// textVarList 是 :4006 行协议的载荷（官方 IVarList 的 text 形态）。
// 逆向事实源：FsDumper.dll text_decode_msg@0x1005C610 /
// token 解码器@0x1005C780 / text_encode_msg@0x1005DA90，方言与
// FsSoloLister（svrlist）同源。
type textVarList struct {
	vals []textValue
}

func newTextVarList() *textVarList { return &textVarList{} }

func (v *textVarList) Len() int { return len(v.vals) }

func (v *textVarList) addInt(n int64) *textVarList {
	v.vals = append(v.vals, textValue{kind: 'i', n: n})
	return v
}

func (v *textVarList) addDouble(f float64) *textVarList {
	v.vals = append(v.vals, textValue{kind: 'd', f: f})
	return v
}

// addString 加窄串（线路 '$' 形态）。
func (v *textVarList) addString(s string) *textVarList {
	v.vals = append(v.vals, textValue{kind: 's', s: s})
	return v
}

// addWide 加宽串（线路 '#' 转义形态）。
func (v *textVarList) addWide(s string) *textVarList {
	v.vals = append(v.vals, textValue{kind: 'w', s: s})
	return v
}

// Int 取第 i 个元素按整数解释（int/int64 直取，double 截断）。
func (v *textVarList) Int(i int) int64 {
	e := v.vals[i]
	if e.kind == 'd' {
		return int64(e.f)
	}
	return e.n
}

// Double 取第 i 个元素按浮点解释。
func (v *textVarList) Double(i int) float64 {
	e := v.vals[i]
	if e.kind == 'd' {
		return e.f
	}
	return float64(e.n)
}

// String 取第 i 个字符串值（窄/宽统一返回解码后文本）。
func (v *textVarList) String(i int) string { return v.vals[i].s }

// Kind 返回第 i 个元素类型标记。
func (v *textVarList) Kind(i int) byte { return v.vals[i].kind }

// EncodeText 把 VarList 编成官方 text 方言行（不含行终止符）。
// 数字按 int 原样、double 带 '.'；窄串 '$' 前缀；宽串 '#' + 逐
// UTF-16 码元转义（\xHH / \uHHHH / '\\' 双写，'%' 走 \x——
// svrlist 前作 golden 同款）。
func (v *textVarList) EncodeText() string {
	parts := make([]string, 0, len(v.vals))
	for _, e := range v.vals {
		switch e.kind {
		case 'i', 'l':
			parts = append(parts, strconv.FormatInt(e.n, 10))
		case 'd':
			s := strconv.FormatFloat(e.f, 'g', -1, 64)
			if !strings.ContainsAny(s, ".eE") {
				s += ".0"
			}
			parts = append(parts, s)
		case 's':
			parts = append(parts, "$"+e.s)
		case 'w':
			parts = append(parts, "#"+escapeWide(e.s))
		}
	}
	return strings.Join(parts, " ")
}

// escapeWide 官方 text_encode_msg 宽串分支：可打印非 '%' 原样（'\' 双写）、
// u>=0x100 → \uHHHH、其余（含空格/控制符/'%'/0x80~0xFF）→ \xHH。
func escapeWide(s string) string {
	var b strings.Builder
	for _, u := range utf16.Encode([]rune(s)) {
		switch {
		case u == '%':
			fmt.Fprintf(&b, "\\x%02X", u)
		case u > 0x20 && u < 0x7f:
			b.WriteByte(byte(u))
			if u == '\\' {
				b.WriteByte('\\')
			}
		case u >= 0x100:
			fmt.Fprintf(&b, "\\u%04X", u)
		default:
			fmt.Fprintf(&b, "\\x%02X", u)
		}
	}
	return b.String()
}

// unescapeWide 官方 text_decode_msg '#' 串分支逆操作（0x1005C996 起）：
// '\\'→'\'；'\u'+4hex → UTF-16 码元；'\x'+2hex → 码元；其余原样。
func unescapeWide(s string) (string, error) {
	var units []uint16
	for i := 0; i < len(s); {
		c := s[i]
		if c != '\\' {
			units = append(units, uint16(c))
			i++
			continue
		}
		i++
		if i >= len(s) {
			return "", fmt.Errorf("fsdumper: text varlist 悬空反斜杠")
		}
		switch s[i] {
		case '\\':
			units = append(units, '\\')
			i++
		case 'u':
			if i+4 >= len(s) {
				return "", fmt.Errorf("fsdumper: text varlist \\u 序列截断")
			}
			n, err := strconv.ParseUint(s[i+1:i+5], 16, 32)
			if err != nil {
				return "", fmt.Errorf("fsdumper: text varlist \\u 序列非法: %w", err)
			}
			units = append(units, uint16(n))
			i += 5
		case 'x':
			if i+2 >= len(s) {
				return "", fmt.Errorf("fsdumper: text varlist \\x 序列截断")
			}
			n, err := strconv.ParseUint(s[i+1:i+3], 16, 32)
			if err != nil {
				return "", fmt.Errorf("fsdumper: text varlist \\x 序列非法: %w", err)
			}
			units = append(units, uint16(n))
			i += 3
		default:
			return "", fmt.Errorf("fsdumper: text varlist 未知转义 \\%c", s[i])
		}
	}
	return string(utf16.Decode(units)), nil
}

// DecodeTextVarList 解析一行 text 方言消息。
// 官方解码器语义：空格分隔；首字符 [0-9.-] → 数字（无 '.' 且长度<10 →
// int32，否则 int64/strtod）；'#' → 宽串；'$' → 窄串；其余拒绝。
func DecodeTextVarList(line string) (*textVarList, error) {
	v := newTextVarList()
	for _, tok := range strings.Fields(line) {
		if tok == "" {
			continue
		}
		c := tok[0]
		switch {
		case (c >= '0' && c <= '9') || c == '.' || c == '-':
			if !strings.Contains(tok, ".") && len(tok) < 10 {
				n, err := strconv.ParseInt(tok, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("fsdumper: text varlist int 非法 %q: %w", tok, err)
				}
				v.addInt(n)
			} else {
				// 官方 _atoi64 优先、失败转 strtod；Go 同序尝试。
				if n, err := strconv.ParseInt(tok, 10, 64); err == nil {
					v.addInt(n)
				} else {
					f, err := strconv.ParseFloat(tok, 64)
					if err != nil {
						return nil, fmt.Errorf("fsdumper: text varlist 数字非法 %q", tok)
					}
					v.addDouble(f)
				}
			}
		case c == '#':
			s, err := unescapeWide(tok[1:])
			if err != nil {
				return nil, err
			}
			v.addWide(s)
		case c == '$':
			v.addString(tok[1:])
		default:
			return nil, fmt.Errorf("fsdumper: text varlist 未知 token %q", tok)
		}
	}
	return v, nil
}

// AppendLine 给编码结果追加官方 CRLF 行终止（SendServerMsg 线路形态）。
func (v *textVarList) AppendLine() []byte { return []byte(v.EncodeText() + "\r\n") }
