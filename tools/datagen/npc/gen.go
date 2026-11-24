// gen.go 输出面：服务端方言 XML 生成（<Object><Property .../>，UTF-8）
// 编码自适应 toUTF8 与 internal/config/objectfile.go 同款。
package npc

import (
	"strings"
	"unicode/utf8"

	"github.com/lizhanyin/9yin-server-go/tools/datagen/genutil"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// toUTF8 编码自适应：合法 UTF-8 直通，否则 GBK 转码（客户端提取档为
// GBK，摆放 XML 多为带 BOM 的 UTF-8）。
func toUTF8(raw []byte) ([]byte, error) {
	raw = stripBOM(raw)
	if utf8.Valid(raw) {
		return raw, nil
	}
	return simplifiedchinese.GBK.NewDecoder().Bytes(raw)
}

func stripBOM(raw []byte) []byte {
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return raw[3:]
	}
	return raw
}

// xmlHeader 生成文件头（与黑金档一致的双引号方言）。
const xmlHeader = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"

// escapeAttr 属性值转义。换行/回车/制表用字符引用，规避 XML 属性规范化
// 把字面换行折叠成空格（表内引号字段含内嵌换行）。
var attrEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"\"", "&quot;",
	"\r", "&#13;",
	"\n", "&#10;",
	"\t", "&#9;",
)

// objectWriter Object/Property XML 累加器。
type objectWriter struct {
	sb strings.Builder
}

func newObjectWriter() *objectWriter {
	w := &objectWriter{}
	w.sb.WriteString(xmlHeader)
	w.sb.WriteString("<Object>\n")
	return w
}

// AddProperty 追加一行属性表。attrs 已按期望顺序排列，值为空串的项跳过。
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

// genNpcConfig 一张模板表 → 服务端 XML 字节。
// 行内属性序：ID、script、Name、Desc 前置（Desc=九阴中文名），其余按列序。
func genNpcConfig(t *npcTable, plan *tablePlan) []byte {
	first := t.firstIndex()
	nameIdx, hasName := first["Name"]
	scriptIdx, hasScript := first["script"]
	w := newObjectWriter()
	for _, row := range t.Rows {
		var attrs [][2]string
		id := ""
		if idx, ok := first["ID"]; ok && idx < len(row) {
			id = row[idx]
		}
		if id == "" {
			continue // 无 ID 行（表尾说明/杂项）不进输出
		}
		attrs = append(attrs, [2]string{"ID", id})
		if hasScript && scriptIdx < len(row) {
			attrs = append(attrs, [2]string{"script", row[scriptIdx]})
		}
		name := ""
		if hasName && nameIdx < len(row) {
			name = row[nameIdx]
		}
		if name != "" {
			attrs = append(attrs, [2]string{"Name", name}, [2]string{"Desc", name})
		}
		for _, pc := range plan.Columns {
			if pc.Kind == "dropped" || pc.SourceIndex >= len(row) {
				continue
			}
			switch pc.OutAttr {
			case "ID", "script", "Name", "Desc":
				continue // 前置项已写
			}
			attrs = append(attrs, [2]string{pc.OutAttr, row[pc.SourceIndex]})
		}
		w.AddProperty(attrs)
	}
	return w.Bytes()
}

// writeGenerated 写文件（建目录；行尾跟随数据根里同名现有文件，见 genutil）。
func writeGenerated(path string, data []byte) error {
	return genutil.WriteFile(path, data)
}
