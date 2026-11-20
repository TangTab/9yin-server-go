package fsgame

import (
	"encoding/xml"
	"fmt"
	"io"
)

// PropType 复刻 struct XML 的属性类型方言：
// byte/word/int/int64/float/double/string/widestr/object（+record 列型 dword）。
type PropType string

// 属性类型常量（对齐 struct_template.xml 注释）。
const (
	TypeByte     PropType = "byte"
	TypeWord     PropType = "word"
	TypeDWord    PropType = "dword"
	TypeInt      PropType = "int"
	TypeInt64    PropType = "int64"
	TypeFloat    PropType = "float"
	TypeDouble   PropType = "double"
	TypeString   PropType = "string"
	TypeWideStr  PropType = "widestr"
	TypeObject   PropType = "object"
	TypeBinary   PropType = "binary"
	TypeBool     PropType = "bool"
	TypeUnkPrec6 PropType = "prec6" // struct 抽样中出现，按 double 承载
)

// PropDef 属性定义（<property name type public private save realtime desc/>）。
type PropDef struct {
	Name     string
	Type     PropType
	Public   bool // 公共可视（AOI 下发面）
	Private  bool // 私有可视（本人下发面）
	Save     bool // 存档面（save_data chunk 收录）
	Realtime bool // 实时下发
	Desc     string
}

// ColDef 记录列定义。
type ColDef struct {
	Type PropType
	Desc string
}

// RecordDef 记录定义（<record name cols maxrows save ...><column .../></record>）。
type RecordDef struct {
	Name    string
	Cols    int
	MaxRows int
	Public  bool
	Private bool
	Save    bool
	Desc    string
	Columns []ColDef
}

// ClassDef 逻辑类定义：logic_class.xml 的类树节点 + struct/<class>.xml 的
// 属性/记录 schema（等价官方 createclass_from_xml(kernel, classID, path)）。
type ClassDef struct {
	Name   string
	Parent string
	Type   string // TYPE_PLAYER/TYPE_SCENE/TYPE_NPC/TYPE_ITEM（可空）
	Desc   string

	// internals 段：引擎内部属性（默认不可视不保存，XML 可改写其标志）。
	// 与 properties 合并存于 Props，此处仅记录顺序来源。
	Props       []PropDef
	propIndex   map[string]int
	Records     []RecordDef
	recordIndex map[string]int
}

// Prop 按名取属性定义。
func (c *ClassDef) Prop(name string) (PropDef, bool) {
	if c == nil || c.propIndex == nil {
		return PropDef{}, false
	}
	i, ok := c.propIndex[name]
	if !ok {
		return PropDef{}, false
	}
	return c.Props[i], true
}

// Record 按名取记录定义。
func (c *ClassDef) Record(name string) (RecordDef, bool) {
	if c == nil || c.recordIndex == nil {
		return RecordDef{}, false
	}
	i, ok := c.recordIndex[name]
	if !ok {
		return RecordDef{}, false
	}
	return c.Records[i], true
}

func (c *ClassDef) seal() {
	c.propIndex = make(map[string]int, len(c.Props))
	for i, p := range c.Props {
		c.propIndex[p.Name] = i
	}
	c.recordIndex = make(map[string]int, len(c.Records))
	for i, r := range c.Records {
		c.recordIndex[r.Name] = i
	}
}

// ---- logic_class.xml ----

type xmlLogicClass struct {
	XMLName xml.Name       `xml:"logicclass"`
	Classes []xmlClassNode `xml:"class"`
}

type xmlClassNode struct {
	ID       string         `xml:"id,attr"`
	Type     string         `xml:"type,attr"`
	Desc     string         `xml:"desc,attr"`
	Children []xmlClassNode `xml:"class"`
}

// ParseLogicClass 解析 logic_class.xml，返回 (类名 → 父类名/类型/描述) 平表。
// 官方树：player→role、scene→NormalScene/CloneScene/…、npc→HiddenNpc/VisibleNpc→…
func ParseLogicClass(r io.Reader) (map[string]ClassDef, error) {
	var root xmlLogicClass
	if err := xml.NewDecoder(r).Decode(&root); err != nil {
		return nil, fmt.Errorf("fsgame: logic_class.xml: %w", err)
	}
	out := make(map[string]ClassDef)
	var walk func(nodes []xmlClassNode, parent string)
	walk = func(nodes []xmlClassNode, parent string) {
		for _, n := range nodes {
			c := ClassDef{Name: n.ID, Parent: parent, Type: n.Type, Desc: n.Desc}
			c.seal()
			out[n.ID] = c
			walk(n.Children, n.ID)
		}
	}
	walk(root.Classes, "")
	return out, nil
}

// ---- struct/<class>.xml ----

type xmlStructObject struct {
	XMLName    xml.Name        `xml:"object"`
	Internals  []xmlPropNode   `xml:"internals>property"`
	Properties []xmlPropNode   `xml:"properties>property"`
	Records    []xmlRecordNode `xml:"records>record"`
	Includes   []xmlIncludeNode `xml:"includes>path"`
}

type xmlIncludeNode struct {
	Name string `xml:"name,attr"` // 官方为反斜杠相对 Res 根路径
	Desc string `xml:"desc,attr"`
}

type xmlPropNode struct {
	Name     string `xml:"name,attr"`
	Type     string `xml:"type,attr"`
	Public   string `xml:"public,attr"`
	Private  string `xml:"private,attr"`
	Save     string `xml:"save,attr"`
	Realtime string `xml:"realtime,attr"`
	Desc     string `xml:"desc,attr"`
}

type xmlRecordNode struct {
	Name    string       `xml:"name,attr"`
	Cols    int          `xml:"cols,attr"`
	MaxRows int          `xml:"maxrows,attr"`
	Public  string       `xml:"public,attr"`
	Private string       `xml:"private,attr"`
	Save    string       `xml:"save,attr"`
	Desc    string       `xml:"desc,attr"`
	Columns []xmlColNode `xml:"column"`
}

type xmlColNode struct {
	Type string `xml:"type,attr"`
	Desc string `xml:"desc,attr"`
}

func xmlBool(s string) bool { return s == "true" || s == "1" }

// ParseStructXML 解析 struct/<class>.xml（object/properties/records 模板，
// 见 testdata/struct/struct_template.xml 注释），把 schema 并入类定义。
// internals 段在前（引擎属性），properties 段随后；重复属性以首次定义为准。
// 返回值为本文件 <includes> 段声明的子 schema 路径（相对 Res 根，反斜杠方言），
// 由 Kernel.LoadStruct 递归装载（官方 createclass_from_xml 的 includes 递归面）。
func ParseStructXML(r io.Reader, into *ClassDef) ([]string, error) {
	var root xmlStructObject
	dec := xml.NewDecoder(r)
	dec.CharsetReader = gbCharsetReader
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("fsgame: struct xml %s: %w", into.Name, err)
	}
	if into.propIndex == nil {
		into.seal()
	}
	addProp := func(n xmlPropNode, internal bool) {
		if _, dup := into.propIndex[n.Name]; dup {
			return
		}
		p := PropDef{
			Name: n.Name, Type: PropType(n.Type),
			Public: xmlBool(n.Public), Private: xmlBool(n.Private),
			Save: xmlBool(n.Save), Realtime: xmlBool(n.Realtime),
			Desc: n.Desc,
		}
		if internal {
			// 引擎内部属性默认不可视不保存（模板注释），XML 显式改写才开。
		}
		into.Props = append(into.Props, p)
		into.propIndex[p.Name] = len(into.Props) - 1
	}
	for _, n := range root.Internals {
		addProp(n, true)
	}
	for _, n := range root.Properties {
		addProp(n, false)
	}
	for _, n := range root.Records {
		if _, dup := into.recordIndex[n.Name]; dup {
			continue
		}
		rd := RecordDef{
			Name: n.Name, Cols: n.Cols, MaxRows: n.MaxRows,
			Public: xmlBool(n.Public), Private: xmlBool(n.Private),
			Save: xmlBool(n.Save), Desc: n.Desc,
		}
		for _, c := range n.Columns {
			rd.Columns = append(rd.Columns, ColDef{Type: PropType(c.Type), Desc: c.Desc})
		}
		if rd.Cols == 0 {
			rd.Cols = len(rd.Columns)
		}
		into.Records = append(into.Records, rd)
		into.recordIndex[rd.Name] = len(into.Records) - 1
	}
	includes := make([]string, 0, len(root.Includes))
	for _, inc := range root.Includes {
		if inc.Name != "" {
			includes = append(includes, inc.Name)
		}
	}
	return includes, nil
}

// EngineInternals 引擎内部属性（struct_template.xml internals 段定案：
// 每个游戏对象默认持有；默认不可视不保存，可被类 XML 改写标志）。
// 真档 player 存档首属性 Name 即此面（widestr）。
var EngineInternals = []PropDef{
	{Name: "name", Type: TypeWideStr, Desc: "名称"},
	{Name: "fixed", Type: TypeByte, Realtime: true, Desc: "固定位"},
}

// withEngineInternals 类定义前置引擎内部属性（子类不重复）。
func withEngineInternals(c *ClassDef) {
	if c.Parent != "" {
		return // 祖先已有；合并面沿链带上
	}
	props := append([]PropDef(nil), EngineInternals...)
	c.Props = append(props, c.Props...)
}
