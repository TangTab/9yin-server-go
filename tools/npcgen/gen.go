// gen.go 输出面：服务端方言 XML 生成（<Object><Property .../>，UTF-8）
// 与黑金档备份。编码自适应 toUTF8 与 internal/config/objectfile.go 同款。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

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

// backupHeiJin 首跑备份：把 <out>/NpcConfig 与 <out>/NpcConfigAuto 的黑金
// 档整体移入 <out>/heijin_backup/（已存在则视为已备份，跳过）。
// 返回说明文字与错误。
func backupHeiJin(outDir string) (string, error) {
	backupDir := filepath.Join(outDir, "heijin_backup")
	if info, err := os.Stat(backupDir); err == nil && info.IsDir() {
		return "heijin_backup 已存在，跳过备份", nil
	}
	moved := 0
	for _, sub := range []string{"NpcConfig", "NpcConfigAuto"} {
		src := filepath.Join(outDir, sub)
		entries, err := os.ReadDir(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("npcgen: 读 %s: %w", src, err)
		}
		if len(entries) == 0 {
			continue
		}
		dst := filepath.Join(backupDir, sub)
		if err := os.MkdirAll(dst, 0o755); err != nil {
			return "", err
		}
		for _, e := range entries {
			if err := os.Rename(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return "", fmt.Errorf("npcgen: 备份 %s: %w", e.Name(), err)
			}
			moved++
		}
	}
	if moved == 0 {
		return "无黑金档可备份", nil
	}
	return fmt.Sprintf("黑金档 %d 项已移入 heijin_backup/", moved), nil
}

// writeGenerated 写文件（建目录）。
func writeGenerated(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
