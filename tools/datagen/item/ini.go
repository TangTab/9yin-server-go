// ini.go 源表读取与输出方言：ini 节解析（表头元数据行、节内重复键、残留
// 合并冲突标记、节外行的容错与告警）、编码自适应，以及 Object/Property XML
// 累加器（npcgen/skillgen 同方言）。包级说明与黑金模型对应关系见 main.go。
package item

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// toUTF8 编码自适应：合法 UTF-8 直通，否则 GBK 转码（与 npcgen/skillgen/
// config 同款）。
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
	// CommentSections `[//说明]` 形态的注释节名（数据之外的分组标题）。
	CommentSections []string
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

// xmlHeader 生成文件头（与 npcgen/skillgen 产物一致；黑金原档 gb2312，本仓
// 产物统一 UTF-8——装载器编码自适应，两条路径均可工作）。
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

func newObjectWriter(comment string) *objectWriter {
	w := &objectWriter{}
	w.sb.WriteString(xmlHeader)
	if comment != "" {
		w.sb.WriteString("<!-- ")
		w.sb.WriteString(comment)
		w.sb.WriteString(" -->\n")
	}
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

// validAttrName XML 属性名合法性（.NameStartChar 的 ASCII 子集；九阴键全是
// ASCII 字母数字下划线，中文/数字开头/@ 开头判非法——@ 键另走审定改名）。
var attrNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.\-]*$`)

func validAttrName(name string) bool { return attrNamePattern.MatchString(name) }

// recordKeyPattern 九阴 `@记录键`（多行成表：@CompensateItemRec 等）。
var recordKeyPattern = regexp.MustCompile(`^@([A-Za-z_][A-Za-z0-9_.\-]*)$`)

// recordAttr 记录键的输出属性名（审定改名）：`@X` → `XRec`，已带 Rec
// 后缀的原名（@CompensateItemRec → CompensateItemRec）——黑金 struct 的
// 记录命名法（dice_member_rec / RaPropertyRec）。不直接去 @：`@ItemType`
// （宝石可镶嵌部位）会与标量 ItemType=73 撞名。非记录键返回 ok=false。
func recordAttr(key string) (string, bool) {
	m := recordKeyPattern.FindStringSubmatch(key)
	if m == nil {
		return "", false
	}
	if strings.HasSuffix(m[1], "Rec") {
		return m[1], true
	}
	return m[1] + "Rec", true
}
