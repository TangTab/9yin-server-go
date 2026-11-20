// creator.go 场景摆放转换：share/creator/npc_creator/<场景>/*.xml →
// 每场景一档 testdata/ini/scene/<场景>/npc_creator.xml（服务端方言）。
// 展开逻辑移植自 9yin-go-server2 internal/clientdata/npc_creator.go：
// 静态 item 直收；randomcreator 的 creator+item 属性合并、已 authored 的
// position 子元素逐点展开（No=组:项:序）；未给 position 的随机项只保留
// Amount/中心/半径，不发明坐标——运行时按量生成。
package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// creatorInstance 一条摆放实例（已展开到点，或保留量的随机组）。
type creatorInstance struct {
	Attrs []xml.Attr // 输出属性（保持来源序）
}

// convertCreatorFile 转换单个 creator XML 文件为实例列表。
func convertCreatorFile(path string) ([]creatorInstance, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data, err := toUTF8(raw)
	if err != nil {
		return nil, fmt.Errorf("解码: %w", err)
	}
	fileBase := filepath.Base(path)
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out []creatorInstance
	var randomCreator []xml.Attr // 当前 <creator> 属性；nil 表示不在随机段
	for {
		token, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("解析: %w", err)
		}
		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "creator":
				randomCreator = append([]xml.Attr(nil), element.Attr...)
			case "item":
				if randomCreator != nil {
					// 随机项：内部连 position 一起消费到 </item>。
					out = append(out, expandRandomItem(dec, randomCreator, element.Attr, fileBase)...)
				} else {
					out = append(out, creatorInstance{Attrs: withFile(element.Attr, fileBase)})
				}
			}
		case xml.EndElement:
			if element.Name.Local == "creator" {
				randomCreator = nil
			}
		}
	}
	return out, nil
}

// expandRandomItem 展开一个随机 item：读取其 position 子元素（读到 item
// 结束为止），逐点生成实例；无 position 的保留单行（Amount 留给运行时）。
func expandRandomItem(dec *xml.Decoder, creator, item []xml.Attr, fileBase string) []creatorInstance {
	positions := readPositions(dec, "item")
	creatorNo := attrValue(creator, "No")
	itemNo := attrValue(item, "No")
	merged := withoutAttr(mergeAttrs(creator, item), "No") // 组合 No 取代原值，避免同名属性
	if len(positions) == 0 {
		attrs := withFile(merged, fileBase)
		attrs = append(attrs,
			xml.Attr{Name: xml.Name{Local: "Random"}, Value: "1"},
			xml.Attr{Name: xml.Name{Local: "CreatorNo"}, Value: creatorNo},
			xml.Attr{Name: xml.Name{Local: "No"}, Value: creatorNo + ":" + itemNo})
		return []creatorInstance{{Attrs: attrs}}
	}
	out := make([]creatorInstance, 0, len(positions))
	for i, pos := range positions {
		attrs := withFile(mergeAttrs(merged, pos), fileBase)
		attrs = append(attrs,
			xml.Attr{Name: xml.Name{Local: "Random"}, Value: "1"},
			xml.Attr{Name: xml.Name{Local: "CreatorNo"}, Value: creatorNo},
			xml.Attr{Name: xml.Name{Local: "No"}, Value: fmt.Sprintf("%s:%s:%d", creatorNo, itemNo, i+1)})
		out = append(out, creatorInstance{Attrs: attrs})
	}
	return out
}

// readPositions 读到 endElement 为止，收集 position 子元素属性。
func readPositions(dec *xml.Decoder, until string) [][]xml.Attr {
	var positions [][]xml.Attr
	for depth := 1; depth > 0; {
		token, err := dec.Token()
		if err != nil {
			return positions
		}
		switch element := token.(type) {
		case xml.StartElement:
			depth++
			if element.Name.Local == "position" {
				positions = append(positions, append([]xml.Attr(nil), element.Attr...))
			}
		case xml.EndElement:
			depth--
			if element.Name.Local == until && depth == 0 {
				return positions
			}
		}
	}
	return positions
}

// withFile 副本 + File 溯源属性置首。
func withFile(attrs []xml.Attr, fileBase string) []xml.Attr {
	out := make([]xml.Attr, 0, len(attrs)+1)
	out = append(out, xml.Attr{Name: xml.Name{Local: "File"}, Value: fileBase})
	out = append(out, attrs...)
	return out
}

// mergeAttrs 多层属性合并（后者覆盖前者同名）。
func mergeAttrs(layers ...[]xml.Attr) []xml.Attr {
	seen := map[string]int{}
	var out []xml.Attr
	for _, layer := range layers {
		for _, a := range layer {
			if idx, ok := seen[a.Name.Local]; ok {
				out[idx] = a
				continue
			}
			seen[a.Name.Local] = len(out)
			out = append(out, a)
		}
	}
	return out
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// withoutAttr 剔除同名属性（保持其余顺序）。
func withoutAttr(attrs []xml.Attr, name string) []xml.Attr {
	out := attrs[:0:0]
	for _, a := range attrs {
		if a.Name.Local != name {
			out = append(out, a)
		}
	}
	return out
}

// convertScene 一个场景目录 → npc_creator.xml 字节 + 统计。
// eventtrigger.xml 是场景事件表（triggercreator），不是 NPC 摆放，排除。
// 单文件损坏（提取源带坏档）只跳过该文件并记警告，不废整个场景。
func convertScene(sceneDir string) ([]byte, int, []string, error) {
	var files []string
	entries, err := os.ReadDir(sceneDir)
	if err != nil {
		return nil, 0, nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".xml") {
			continue
		}
		if strings.EqualFold(e.Name(), "eventtrigger.xml") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)
	w := newObjectWriter()
	var warnings []string
	count := 0
	for _, name := range files {
		instances, err := convertCreatorFile(filepath.Join(sceneDir, name))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("跳过损坏文件 %s: %v", name, err))
			continue
		}
		for _, inst := range instances {
			attrs := make([][2]string, 0, len(inst.Attrs))
			for _, a := range inst.Attrs {
				attrs = append(attrs, [2]string{a.Name.Local, a.Value})
			}
			w.AddProperty(attrs)
			count++
		}
	}
	return w.Bytes(), count, warnings, nil
}
