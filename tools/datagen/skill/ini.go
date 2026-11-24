// ini.go 源表读取与输出方言：ini 节解析（残留合并冲突标记、表头元数据
// 行、节内重复键、节外行的容错与告警）、编码自适应，以及 Object/Property
// XML 累加器（npcgen 同方言）。包级说明与黑金模型对应关系见 main.go。
package skill

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// toUTF8 编码自适应：合法 UTF-8 直通，否则 GBK 转码（与 npcgen/config 同款）。
func toUTF8(raw []byte) []byte {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		raw = raw[3:]
	}
	if utf8.Valid(raw) {
		return raw
	}
	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw)
	if err != nil {
		return raw
	}
	return decoded
}

// iniSection ini 一节：键值按出现序保留（允许重复键）。
type iniSection struct {
	Name  string
	Line  int // 节头行号（1 基）
	Keys  []string
	Vals  []string
	Lines []int // 每个键值所在行号
}

// Get 首见键值。
func (s *iniSection) Get(key string) (string, bool) {
	for i, k := range s.Keys {
		if k == key {
			return s.Vals[i], true
		}
	}
	return "", false
}

// iniTable 一张 ini 表的解析结果。
type iniTable struct {
	Source   string
	Sections []*iniSection
	Warnings []string
	// ConflictMarkers 残留合并冲突标记行号（1 基）。
	ConflictMarkers []int
}

// parseINI 解析 `[节]` + `键=值` 方言。注释行（; # //）与空行跳过；
// `<<<<<<<`/`=======`/`>>>>>>>` 开头的合并冲突标记行跳过并记录。
func parseINI(data []byte, source string) *iniTable {
	t := &iniTable{Source: source}
	var cur *iniSection
	line := 1
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		var raw []byte
		if end < 0 {
			raw, data = data, nil
		} else {
			raw, data = data[:end], data[end+1:]
		}
		text := strings.TrimSpace(string(toUTF8(raw)))
		if text != "" {
			switch {
			case strings.HasPrefix(text, "<<<<<<<") || strings.HasPrefix(text, "=======") ||
				strings.HasPrefix(text, ">>>>>>>"):
				t.ConflictMarkers = append(t.ConflictMarkers, line)
			case strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]"):
				cur = &iniSection{Name: text[1 : len(text)-1], Line: line}
				t.Sections = append(t.Sections, cur)
			case strings.HasPrefix(text, ";") || strings.HasPrefix(text, "#") || strings.HasPrefix(text, "//"):
				// 注释
			default:
				if cur != nil {
					key, value := text, ""
					if eq := strings.IndexByte(text, '='); eq >= 0 {
						key, value = text[:eq], text[eq+1:]
					}
					cur.Keys = append(cur.Keys, strings.TrimSpace(key))
					cur.Vals = append(cur.Vals, strings.TrimSpace(value))
					cur.Lines = append(cur.Lines, line)
				} else {
					t.Warnings = append(t.Warnings, fmt.Sprintf("%s:%d: 节外行 %q 忽略", source, line, text))
				}
			}
		}
		line++
	}
	if n := len(t.ConflictMarkers); n > 0 {
		t.Warnings = append(t.Warnings, fmt.Sprintf("%s: 残留合并冲突标记 %d 处（行 %v），标记行跳过、其间数据保留",
			source, n, t.ConflictMarkers))
	}
	return t
}

// xmlHeader 生成文件头（与黑金档/npcgen 一致）。
const xmlHeader = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"

// attrEscaper 属性值转义（npcgen 同款：换行/回车/制表用字符引用）。
var attrEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"\"", "&quot;",
	"\r", "&#13;",
	"\n", "&#10;",
	"\t", "&#9;",
)

// objectWriter Object/Property XML 累加器（npcgen 同方言）。
type objectWriter struct {
	sb strings.Builder
}

func newObjectWriter() *objectWriter {
	w := &objectWriter{}
	w.sb.WriteString(xmlHeader)
	w.sb.WriteString("<Object>\n")
	return w
}

// AddProperty 追加一行（空值项跳过）。
func (w *objectWriter) AddProperty(attrs [][2]string) {
	w.sb.WriteString("\t<Property")
	for _, kv := range attrs {
		if kv[1] == "" {
			continue
		}
		w.sb.WriteString(" ")
		w.sb.WriteString(kv[0])
		w.sb.WriteString("=\"")
		w.sb.WriteString(attrEscaper.Replace(kv[1]))
		w.sb.WriteString("\"")
	}
	w.sb.WriteString("/>\n")
}

// Bytes 收尾。
func (w *objectWriter) Bytes() []byte {
	w.sb.WriteString("</Object>\n")
	return []byte(w.sb.String())
}
