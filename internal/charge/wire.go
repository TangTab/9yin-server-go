// Package charge 复刻官方充值计费中心（BGserverCJ\Debug\FsCharge.dll +
// GameCharge.ini，FsLoader 形态独立进程 GameCharge.exe）。规格全部由
// 2014-03-22 官方全符号快照反汇编定案（逆向工作底稿：
// tmp_research/fscharge/{report_receive,report_publish}.md，权威沉淀
// docs/official/FsCharge充值中心协议全解.md）：
//
//   - 传输：TCP 行文本。发送 = text_encode_msg 编码行 + CRLF；
//     接收 = 逐字节累积可见字符（>=0x20），遇 '\n' 且缓冲非空即成帧
//     （CMassSockHook::OnReceive@0x78910，行缓冲 0x1000）。
//   - 元素编码（text_encode_msg@0x56b50 / text_decode_msg@0x55920）：
//     空格分隔；int=%d、int64=十进制、float/double=%f、窄串 `$` 前缀、
//     宽串 `#` 前缀（逐 UTF-16 码元转义）、userdata `*` 前缀；
//     解码侧裸词（无前缀）按窄串处理（官方 svrlist 请求同方言）。
//   - 消息 = IVarList：[0]=回显串（请求方路由标记，原样带回）、
//     [1]=命令名、[2..]=参数；应答同构（Publish* 构造）。
package charge

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Kind 是 varlist 元素的官方类型枚举（TVarList GetType 槽位族）。
type Kind uint8

const (
	KindInt       Kind = iota // 32 位整数（%d）
	KindInt64                 // 64 位整数（%I64d，十进制）
	KindDouble                // 浮点（%f 定点 6 位小数）
	KindString                // 窄串（`$` 前缀）
	KindWideString            // 宽串（`#` 前缀）
	KindUserData              // 用户数据（`*` 前缀）
)

// Value 是单个 varlist 元素。窄串/宽串共用 S（宽串按 UTF-16 码元转义）；
// 整数族共用 I；浮点用 D。
type Value struct {
	Kind Kind
	I    int64
	D    float64
	S    string
}

// VarList 是一条线消息的元素序列。
type VarList []Value

// Int 返回第 i 个元素构造的 int 值（整数族取整、浮点截断、串 atoi——
// 与官方 IntVal 容错语义一致）；越界返回 0。
func (v VarList) Int(i int) int {
	if i < 0 || i >= len(v) {
		return 0
	}
	switch v[i].Kind {
	case KindString, KindWideString:
		n, _ := strconv.Atoi(strings.TrimSpace(v[i].S))
		return n
	case KindDouble:
		return int(v[i].D)
	default:
		return int(v[i].I)
	}
}

// StringVal 返回第 i 个元素的串值（宽串返回其 UTF-8 形态）；越界空串。
func (v VarList) StringVal(i int) string {
	if i < 0 || i >= len(v) {
		return ""
	}
	return v[i].S
}

// DoubleVal 返回第 i 个元素的浮点值（整数族提升）；越界 0。
func (v VarList) DoubleVal(i int) float64 {
	if i < 0 || i >= len(v) {
		return 0
	}
	switch v[i].Kind {
	case KindDouble:
		return v[i].D
	case KindString, KindWideString:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v[i].S), 64)
		return f
	default:
		return float64(v[i].I)
	}
}

// AddInt 追加一个 int 元素。
func (v *VarList) AddInt(n int) { *v = append(*v, Value{Kind: KindInt, I: int64(n)}) }

// AddDouble 追加一个浮点元素。
func (v *VarList) AddDouble(f float64) { *v = append(*v, Value{Kind: KindDouble, D: f}) }

// AddString 追加一个窄串元素。
func (v *VarList) AddString(s string) { *v = append(*v, Value{Kind: KindString, S: s}) }

// AddWideStr 追加一个宽串元素（S 存 UTF-8，编码时按 UTF-16 码元转义）。
func (v *VarList) AddWideStr(s string) { *v = append(*v, Value{Kind: KindWideString, S: s}) }

// EncodeLine 按官方 text_encode_msg 方言编码为单行（不含 CRLF）。
// 元素间单个空格；窄串/宽串转义规则：可打印（0x20<u<0x7f）且非 '%''
// 原样（'\' 双写），其余 \xHH；宽串码元 >=0x100 用 \uHHHH。
func (v VarList) EncodeLine() string {
	parts := make([]string, 0, len(v))
	for _, e := range v {
		switch e.Kind {
		case KindInt:
			parts = append(parts, strconv.FormatInt(e.I, 10))
		case KindInt64:
			parts = append(parts, strconv.FormatInt(e.I, 10))
		case KindDouble:
			parts = append(parts, fmt.Sprintf("%f", e.D))
		case KindString:
			parts = append(parts, "$"+escapeNarrow(e.S))
		case KindWideString:
			parts = append(parts, "#"+escapeWide(e.S))
		case KindUserData:
			parts = append(parts, "*"+escapeNarrow(e.S))
		}
	}
	return strings.Join(parts, " ")
}

// escapeNarrow 官方窄串转义（字节域）。
func escapeNarrow(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c > 0x20 && c < 0x7f && c != '%':
			b.WriteByte(c)
			if c == '\\' {
				b.WriteByte('\\')
			}
		default:
			fmt.Fprintf(&b, "\\x%02X", c)
		}
	}
	return b.String()
}

// escapeWide 官方宽串转义（逐 UTF-16 码元；与 lister 侧同源规格，
// FsSoloLister.dll text_encode_msg 宽串分支终局背书）。
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

// unescapeNarrow 官方窄串解码：`\\`→'\'，`\xHH`→字节，其余原样。
func unescapeNarrow(s string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(s) {
			return "", fmt.Errorf("charge: 窄串悬挂转义@%d", i)
		}
		switch s[i+1] {
		case '\\':
			b.WriteByte('\\')
			i++
		case 'x', 'X':
			if i+3 >= len(s) {
				return "", fmt.Errorf("charge: \\x 转义缺两位十六进制@%d", i)
			}
			n, err := strconv.ParseUint(s[i+2:i+4], 16, 8)
			if err != nil {
				return "", fmt.Errorf("charge: \\x 转义非法@%d: %w", i, err)
			}
			b.WriteByte(byte(n))
			i += 3
		default:
			return "", fmt.Errorf("charge: 窄串未知转义@%d: \\%c", i, s[i+1])
		}
	}
	return b.String(), nil
}

// DecodeLine 按官方 text_decode_msg 方言解码一行（不含换行符）。
// 空格分词；`$`窄串 / `#`宽串 / `*`userdata / 其余裸词按窄串；
// 首字符数字、'.'、'-' 的词按数字（含 '.' 为 double；长度 <10 走
// atoi 的 int 域，否则 int64）。
func DecodeLine(line string) (VarList, error) {
	var out VarList
	tok := ""
	flush := func() error {
		if tok == "" {
			return nil
		}
		v, err := decodeToken(tok)
		if err != nil {
			return err
		}
		out = append(out, v)
		tok = ""
		return nil
	}
	for _, c := range []byte(line) {
		if c == ' ' {
			if err := flush(); err != nil {
				return nil, err
			}
			continue
		}
		tok += string(rune(c))
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeToken(tok string) (Value, error) {
	if tok == "" {
		return Value{}, fmt.Errorf("charge: 空 token")
	}
	c := tok[0]
	if (c >= '0' && c <= '9') || c == '.' || c == '-' {
		if strings.Contains(tok, ".") {
			f, err := strconv.ParseFloat(tok, 64)
			if err != nil {
				return Value{}, fmt.Errorf("charge: 浮点 %q: %w", tok, err)
			}
			return Value{Kind: KindDouble, D: f}, nil
		}
		if len(tok) < 10 {
			n, err := strconv.Atoi(tok)
			if err != nil {
				return Value{}, fmt.Errorf("charge: 整数 %q: %w", tok, err)
			}
			return Value{Kind: KindInt, I: int64(n)}, nil
		}
		n, err := strconv.ParseInt(tok, 10, 64)
		if err != nil {
			return Value{}, fmt.Errorf("charge: int64 %q: %w", tok, err)
		}
		return Value{Kind: KindInt64, I: n}, nil
	}
	switch c {
	case '$':
		s, err := unescapeNarrow(tok[1:])
		return Value{Kind: KindString, S: s}, err
	case '*':
		s, err := unescapeNarrow(tok[1:])
		return Value{Kind: KindUserData, S: s}, err
	case '#':
		s, err := unescapeWide(tok[1:])
		return Value{Kind: KindWideString, S: s}, err
	default:
		s, err := unescapeNarrow(tok)
		return Value{Kind: KindString, S: s}, err
	}
}

// unescapeWide 官方宽串解码：`\\`→'\'，`\xHH`→BMP 码元，`\uHHHH`→码元；
// 结果按 UTF-16 码元重组为 UTF-8。
func unescapeWide(s string) (string, error) {
	var units []uint16
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' {
			units = append(units, uint16(c))
			continue
		}
		if i+1 >= len(s) {
			return "", fmt.Errorf("charge: 宽串悬挂转义@%d", i)
		}
		switch s[i+1] {
		case '\\':
			units = append(units, '\\')
			i++
		case 'x', 'X':
			if i+3 >= len(s) {
				return "", fmt.Errorf("charge: 宽串 \\x 缺两位@%d", i)
			}
			n, err := strconv.ParseUint(s[i+2:i+4], 16, 8)
			if err != nil {
				return "", fmt.Errorf("charge: 宽串 \\x 非法@%d: %w", i, err)
			}
			units = append(units, uint16(n))
			i += 3
		case 'u', 'U':
			if i+5 >= len(s) {
				return "", fmt.Errorf("charge: 宽串 \\u 缺四位@%d", i)
			}
			n, err := strconv.ParseUint(s[i+2:i+6], 16, 16)
			if err != nil {
				return "", fmt.Errorf("charge: 宽串 \\u 非法@%d: %w", i, err)
			}
			units = append(units, uint16(n))
			i += 5
		default:
			return "", fmt.Errorf("charge: 宽串未知转义@%d", i)
		}
	}
	return string(utf16.Decode(units)), nil
}
