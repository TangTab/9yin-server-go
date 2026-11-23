package skilldata

// xmltable.go Object/Property XML 的按需索引装载。
//
// 产物合计约 30MB/18 万行，整表解成 map（config.LoadObjectFile 口径）要占
// 数百 MB。这里每个文件只在首次用到时读入，一遍扫描 `<Property` 元素建
// 「ID → 元素偏移」索引（只取 ID，不解析其余属性；同 ID 首行胜出）；行属性
// 在取用时才解析。

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"sync"
)

// xmlTable 一个 Object/Property XML 文件：原始字节 + ID 索引。
type xmlTable struct {
	once    sync.Once
	raw     []byte
	index   map[string]int32 // ID → 元素起始偏移
	missing bool             // 文件缺席或不可读：lookup 恒 false（各包均可选）
}

const propertyOpen = "<Property"

var propertyOpenBytes = []byte(propertyOpen)

// attr 一个属性（取用时 value 已反转义）。
type attr struct{ name, value string }

// row 一行属性。
type row []attr

// ensure 首次使用时装载并建索引。
func (t *xmlTable) ensure(path string) {
	t.once.Do(func() {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.missing = true
			return
		}
		t.raw = raw
		t.index = make(map[string]int32)
		for offset := 0; ; {
			at := indexOfProperty(raw, offset)
			if at < 0 {
				break
			}
			// 元素名后必须是空白 / / / >：<PropertyX 之类伪前缀跳过。
			if !isSpace(raw[at]) && raw[at] != '/' && raw[at] != '>' {
				offset = at
				continue
			}
			end, id := scanAttributes(raw, at, nil)
			if end < 0 {
				break // 畸形元素：索引止于此（产物经 skillgen verify 回读）
			}
			if id != "" {
				if _, duplicate := t.index[id]; !duplicate {
					t.index[id] = int32(at - len(propertyOpen))
				}
			}
			offset = end
		}
	})
}

// indexOfProperty 自 offset 起找下一个 `<Property`，返回元素名之后的偏移
// （无则 -1）。bytes.Index 直接在原字节上找，不拷贝缓冲区。
func indexOfProperty(raw []byte, offset int) int {
	at := bytes.Index(raw[offset:], propertyOpenBytes)
	if at < 0 {
		return -1
	}
	return offset + at + len(propertyOpen)
}

// lookup 取一行（ID 精确匹配；同 ID 首行胜出）。行属性在此解析。
func (t *xmlTable) lookup(id string) (row, bool) {
	if t.missing {
		return nil, false
	}
	offset, ok := t.index[id]
	if !ok {
		return nil, false
	}
	var attrs []attr
	scanAttributes(t.raw, int(offset)+len(propertyOpen), &attrs)
	for i := range attrs {
		attrs[i].value = unescapeAttr(attrs[i].value)
	}
	return attrs, true
}

// scanAttributes 自元素名之后扫描属性直至元素结束。collect 非 nil 时追加
// (属性名, 转义态值)；返回元素结束偏移与 ID 属性值（转义态），end<0 为畸形。
// 方言：name="value"，容忍 name = "value"（黑金档写法）。
func scanAttributes(raw []byte, i int, collect *[]attr) (end int, id string) {
	for i < len(raw) {
		for i < len(raw) && isSpace(raw[i]) {
			i++
		}
		if i >= len(raw) {
			return -1, ""
		}
		switch raw[i] {
		case '>':
			return i + 1, id
		case '/':
			if i+1 < len(raw) && raw[i+1] == '>' {
				return i + 2, id
			}
			return -1, ""
		}
		nameStart := i
		for i < len(raw) && raw[i] != '=' && raw[i] != '>' && raw[i] != '/' && !isSpace(raw[i]) {
			i++
		}
		if i >= len(raw) {
			return -1, ""
		}
		name := raw[nameStart:i]
		for i < len(raw) && isSpace(raw[i]) {
			i++
		}
		if i >= len(raw) || raw[i] != '=' {
			return -1, "" // 裸值属性（struct 方言）不在产物里出现
		}
		i++
		for i < len(raw) && isSpace(raw[i]) {
			i++
		}
		if i >= len(raw) || (raw[i] != '"' && raw[i] != '\'') {
			return -1, ""
		}
		quote := raw[i]
		i++
		valueStart := i
		for i < len(raw) && raw[i] != quote {
			i++
		}
		if i >= len(raw) {
			return -1, ""
		}
		value := raw[valueStart:i]
		if id == "" && string(name) == "ID" {
			id = string(value)
		}
		if collect != nil {
			*collect = append(*collect, attr{string(name), string(value)})
		}
		i++
	}
	return -1, ""
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

// unescapeAttr 属性值反转义（skillgen attrEscaper 的实体 + 十进制/十六进制
// 字符引用；无 & 时零分配直通）。
func unescapeAttr(value string) string {
	if !strings.Contains(value, "&") {
		return value
	}
	var sb strings.Builder
	sb.Grow(len(value))
	for i := 0; i < len(value); {
		if value[i] != '&' {
			j := strings.IndexByte(value[i:], '&')
			if j < 0 {
				sb.WriteString(value[i:])
				break
			}
			sb.WriteString(value[i : i+j])
			i += j
			continue
		}
		semi := strings.IndexByte(value[i:], ';')
		if semi < 0 {
			sb.WriteString(value[i:])
			break
		}
		entity := value[i+1 : i+semi]
		switch entity {
		case "amp":
			sb.WriteByte('&')
		case "lt":
			sb.WriteByte('<')
		case "gt":
			sb.WriteByte('>')
		case "quot":
			sb.WriteByte('"')
		case "apos":
			sb.WriteByte('\'')
		default:
			if !writeCharRef(&sb, entity) {
				sb.WriteString(value[i : i+semi+1]) // 未知实体原样保留
			}
		}
		i += semi + 1
	}
	return sb.String()
}

// writeCharRef 十进制 #NN / 十六进制 #xHH 字符引用。
func writeCharRef(sb *strings.Builder, entity string) bool {
	digits, base := "", 10
	switch {
	case strings.HasPrefix(entity, "#x"), strings.HasPrefix(entity, "#X"):
		digits, base = entity[2:], 16
	case len(entity) > 1 && entity[0] == '#':
		digits = entity[1:]
	default:
		return false
	}
	n, err := strconv.ParseUint(digits, base, 32)
	if err != nil {
		return false
	}
	sb.WriteRune(rune(n))
	return true
}

// ---- 取值助手（线性扫描；行属性 ≤ 71 个） ----

func (r row) str(key string) (string, bool) {
	for _, a := range r {
		if a.name == key {
			return a.value, true
		}
	}
	return "", false
}

// int 整数列（缺席/非法回落 fallback）。
func (r row) int(key string, fallback int) int {
	value, ok := r.str(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return n
}

// float 浮点列（缺席/非法回落 fallback）。
func (r row) float(key string, fallback float64) float64 {
	value, ok := r.str(key)
	if !ok {
		return fallback
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return fallback
	}
	return f
}

// flag 非零整型即真。
func (r row) flag(key string) bool { return r.int(key, 0) != 0 }

// csv 逗号列表（过滤空段）。
func (r row) csv(key string) []string {
	value, ok := r.str(key)
	if !ok {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// text 取串列原值（缺席为空）。
func (r row) text(key string) string {
	value, _ := r.str(key)
	return value
}

// ref 取引用列（数值 ID 串；去首尾空白，缺席为空）。
func (r row) ref(key string) string {
	value, _ := r.str(key)
	return strings.TrimSpace(value)
}
