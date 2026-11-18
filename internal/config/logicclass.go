package config

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

// LogicClass 是服务器逻辑对象类型树的节点（logic_class.xml）。
// 基类带引擎类型（TYPE_PLAYER/TYPE_SCENE/TYPE_NPC），子类按继承链挂接，
// scenes.xml 的 Script 字段引用的就是这棵树的 class id。
type LogicClass struct {
	ID       string
	Type     string
	Desc     string
	Parent   *LogicClass
	Children []*LogicClass
}

// LogicClassTree 类型树 + id 索引。
type LogicClassTree struct {
	roots []*LogicClass
	byID  map[string]*LogicClass
}

type xmlClass struct {
	ID       string     `xml:"id,attr"`
	Type     string     `xml:"type,attr"`
	Desc     string     `xml:"desc,attr"`
	Children []xmlClass `xml:"class"`
}

// LoadLogicClass 解析 logic_class.xml（UTF-8/GBK 自适应，嵌套任意深度）。
func LoadLogicClass(path string) (*LogicClassTree, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	utf8, err := toUTF8(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: GBK 转码失败: %w", path, err)
	}

	var xo struct {
		Classes []xmlClass `xml:"class"`
	}
	dec := xml.NewDecoder(bytes.NewReader(utf8))
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) { return input, nil }
	if err := dec.Decode(&xo); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	t := &LogicClassTree{byID: map[string]*LogicClass{}}
	for _, xc := range xo.Classes {
		root := convert(xc, nil, t)
		t.roots = append(t.roots, root)
	}
	return t, nil
}

func convert(xc xmlClass, parent *LogicClass, t *LogicClassTree) *LogicClass {
	lc := &LogicClass{ID: xc.ID, Type: xc.Type, Desc: xc.Desc, Parent: parent}
	if _, dup := t.byID[lc.ID]; dup {
		// 同名 class 后者覆盖前者（原版未见此情况，防御处理）
		lc.ID = xc.ID + "#dup"
	}
	t.byID[xc.ID] = lc
	for _, c := range xc.Children {
		child := convert(c, lc, t)
		lc.Children = append(lc.Children, child)
	}
	return lc
}

// Get 按 id 取类型节点；不存在返回 nil。
func (t *LogicClassTree) Get(id string) *LogicClass { return t.byID[id] }

// Len 类型总数（含全部层级）。
func (t *LogicClassTree) Len() int { return len(t.byID) }

// Roots 顶层类型（player/scene/npc/...）。
func (t *LogicClassTree) Roots() []*LogicClass { return t.roots }

// IsA 判断 id 类型是否继承自 ancestor（含自身）。
// 例：IsA("CloneScene", "scene") == true。
func (t *LogicClassTree) IsA(id, ancestor string) bool {
	for lc := t.byID[id]; lc != nil; lc = lc.Parent {
		if lc.ID == ancestor {
			return true
		}
	}
	return false
}

// RootsOf 返回 id 的继承链（自身→根）。
func (t *LogicClassTree) RootsOf(id string) []string {
	var chain []string
	for lc := t.byID[id]; lc != nil; lc = lc.Parent {
		chain = append(chain, lc.ID)
	}
	return chain
}
