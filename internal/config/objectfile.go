package config

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// Row 是 Object/Property XML 中一行 <Property> 的原始属性集（GBK 已转 UTF-8）。
// 值保留字符串原样，类型化取值走辅助方法，未配置的键返回默认值。
type Row map[string]string

// Str 取字符串值，缺省返回 def。
func (r Row) Str(key, def string) string {
	if v, ok := r[key]; ok && v != "" {
		return v
	}
	return def
}

// Int 取整数值（十进制），缺失或非法返回 def。
func (r Row) Int(key string, def int) int {
	v, ok := r[key]
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}

// Bool 取布尔值：原版写法 "1"/"0"，兼容 "true"/"false"。
func (r Row) Bool(key string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(r.Str(key, ""))) {
	case "1", "true":
		return true
	case "0", "false":
		return false
	}
	return def
}

// CSV 取逗号分隔列表，过滤空段（原版存在 "A,,B" 写法）。
func (r Row) CSV(key string) []string {
	v, ok := r[key]
	if !ok {
		return nil
	}
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ObjectDoc 是一个 Object/Property XML 文件的解析结果：若干行。
type ObjectDoc struct {
	Rows []Row
}

var (
	commentOpen  = []byte("<!--")
	commentClose = []byte("-->")
)

// toUTF8 编码自适应：合法 UTF-8 直通，否则 GBK 转码
// （testdata 快照为 UTF-8，NINEYIN_DATA 直读原版为 GBK）。
func toUTF8(raw []byte) ([]byte, error) {
	if utf8.Valid(raw) {
		return raw, nil
	}
	return simplifiedchinese.GBK.NewDecoder().Bytes(raw)
}

// LoadObjectFile 加载通用 Object/Property XML（UTF-8 或 GBK 自适应）。
// 该格式覆盖原版绝大部分数值表与注册表（scenes.xml、ini\**\*.xml）。
func LoadObjectFile(path string) (*ObjectDoc, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	utf8, err := toUTF8(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: GBK 转码失败: %w", path, err)
	}
	clean := stripComments(utf8)

	var xo struct {
		// 根元素即 <Object>，字段路径相对根：直接取子元素 <Property>。
		Property []struct {
			Attrs []xml.Attr `xml:",any,attr"`
		} `xml:"Property"`
	}
	dec := xml.NewDecoder(bytes.NewReader(clean))
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) { return input, nil }
	if err := dec.Decode(&xo); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	doc := &ObjectDoc{Rows: make([]Row, 0, len(xo.Property))}
	for _, p := range xo.Property {
		row := Row{}
		for _, a := range p.Attrs {
			row[a.Name.Local] = a.Value
		}
		doc.Rows = append(doc.Rows, row)
	}
	return doc, nil
}

func stripComments(b []byte) []byte {
	var out []byte
	for {
		i := bytes.Index(b, commentOpen)
		if i < 0 {
			break
		}
		j := bytes.Index(b[i:], commentClose)
		if j < 0 {
			b = nil
			break
		}
		out = append(out, b[:i]...)
		b = b[i+j+len(commentClose):]
	}
	return append(out, b...)
}
