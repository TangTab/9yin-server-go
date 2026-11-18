package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// PropDef 属性定义（internals 与 properties 同构，Internal 标记来源）。
type PropDef struct {
	Name, Type, Desc                 string
	Internal, Public, Private        bool
	Save, Realtime                   bool
}

// ColDef 表格列定义（列无名，身份是位置）。
type ColDef struct {
	Type, Desc string
}

// RecordDef 表格（Record）定义。
type RecordDef struct {
	Name                     string
	Cols, MaxRows            int
	Public, Private, Save    bool
	Desc                     string
	Columns                  []ColDef
}

// ObjectDef 一个 struct XML 文件的对象定义。
type ObjectDef struct {
	File      string // 相对 struct 根，如 player/Achievement.xml
	Group     string // 一级目录名；根文件为 ""
	GoName    string // 组件类型名（全路径分段驼峰，全局去重后分配）
	Internals []PropDef
	Props     []PropDef
	Records   []RecordDef
	Includes  []string // 已归一化为 struct 根内相对路径
}

type xmlObject struct {
	Internals struct {
		Property []xmlProp `xml:"property"`
	} `xml:"internals"`
	Properties struct {
		Property []xmlProp `xml:"property"`
	} `xml:"properties"`
	Records struct {
		Record []xmlRecord `xml:"record"`
	} `xml:"records"`
	Includes struct {
		Path []struct {
			Name string `xml:"name,attr"`
		} `xml:"path"`
	} `xml:"includes"`
}

type xmlProp struct {
	Name     string `xml:"name,attr"`
	Type     string `xml:"type,attr"`
	Public   string `xml:"public,attr"`
	Private  string `xml:"private,attr"`
	Save     string `xml:"save,attr"`
	Realtime string `xml:"realtime,attr"`
	Desc     string `xml:"desc,attr"`
}

type xmlRecord struct {
	Name    string   `xml:"name,attr"`
	Cols    int      `xml:"cols,attr"`
	MaxRows int      `xml:"maxrows,attr"`
	Public  string   `xml:"public,attr"`
	Private string   `xml:"private,attr"`
	Save    string   `xml:"save,attr"`
	Desc    string   `xml:"desc,attr"`
	Column  []xmlCol `xml:"column"`
}

type xmlCol struct {
	Type string `xml:"type,attr"`
	Desc string `xml:"desc,attr"`
}

func parseBool(s string) bool { return strings.EqualFold(s, "true") || s == "1" }

func toProp(p xmlProp, internal bool) PropDef {
	return PropDef{
		Name: p.Name, Type: p.Type, Desc: p.Desc,
		Internal:  internal,
		Public:    parseBool(p.Public),
		Private:   parseBool(p.Private),
		Save:      parseBool(p.Save),
		Realtime:  parseBool(p.Realtime),
	}
}

// stripComments 容错剥除 XML 注释：原版文件存在注释内含 "--" 的非法写法
// （如 player/PetShow.xml），encoding/xml 会拒绝；注释对生成无意义，直接丢弃。
func stripComments(b []byte) []byte {
	var out []byte
	for {
		i := bytes.Index(b, commentOpen)
		if i < 0 {
			break
		}
		j := bytes.Index(b[i:], commentClose)
		if j < 0 { // 未闭合注释：丢弃尾部
			b = nil
			break
		}
		out = append(out, b[:i]...)
		b = b[i+j+len(commentClose):]
	}
	return append(out, b...)
}

var (
	commentOpen  = []byte("<!--")
	commentClose = []byte("-->")
)

// attrFixup 修复未加引号的数字属性值（如 transtool.xml 的 maxrows=1，
// 原版解析器容忍此写法）。带空白/标签边界约束，避免误伤引号内文本。
var attrFixup = regexp.MustCompile(`(\s)([A-Za-z_][A-Za-z0-9_]*)=([0-9]+)([\s/>])`)

// parseObject 解析单个 struct XML：GBK 转码 → 剥注释 → 修属性 → 解码。
func parseObject(path string) (*ObjectDef, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	utf8, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: GBK 转码失败: %w", path, err)
	}
	clean := attrFixup.ReplaceAll(stripComments(utf8), []byte(`$1$2="$3"$4`))

	// 已转码为 UTF-8，声明里的 gb2312 直通即可。
	dec := xml.NewDecoder(bytes.NewReader(clean))
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	var xo xmlObject
	if err := dec.Decode(&xo); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	obj := &ObjectDef{}
	for _, p := range xo.Internals.Property {
		obj.Internals = append(obj.Internals, toProp(p, true))
	}
	for _, p := range xo.Properties.Property {
		obj.Props = append(obj.Props, toProp(p, false))
	}
	for _, r := range xo.Records.Record {
		rd := RecordDef{
			Name: r.Name, Cols: r.Cols, MaxRows: r.MaxRows,
			Public: parseBool(r.Public), Private: parseBool(r.Private),
			Save: parseBool(r.Save), Desc: r.Desc,
		}
		for _, c := range r.Column {
			rd.Columns = append(rd.Columns, ColDef{Type: c.Type, Desc: c.Desc})
		}
		obj.Records = append(obj.Records, rd)
	}
	for _, inc := range xo.Includes.Path {
		obj.Includes = append(obj.Includes, normInclude(inc.Name))
	}
	return obj, nil
}

// normInclude "struct\rule\fight.xml" → "rule/fight.xml"。
func normInclude(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	return strings.TrimPrefix(p, "struct/")
}
