package sharedata

// creator.go 场景摆放 creator XML（share/creator/npc_creator/<场景>/*.xml，
// 客户端编辑器原生档）展开为属性行。
//
// 展开规则与构建器（tools/datagen/npc/creator.go，产物
// ini/Scene/<场景>/npc_creator.xml）逐条对齐：
//
//	<object>
//	  <staticcreator><item No=.. x=.. y=.. z=.. ay=.. id=.. …/></staticcreator>
//	  <randomcreator>
//	    <creator No=.. x=.. y=.. z=.. radius=.. …>        随机组（中心 + 半径）
//	      <item No=.. id=.. amount=.. …>                 组内一种 NPC
//	        <position x=.. z=.. ay=.. />                 已摆的点：逐点成实例
//	      </item>
//	    </creator>
//	  </randomcreator>
//
//   - 静态 item：属性原样成一行，另加 File=来源档名（下发类别白名单按它判）。
//   - 随机 item：creator 与 item 属性合并（后者覆盖同名），已给出的 position
//     逐点成实例（坐标并入、No=组:项:序）；没给 position 的只保留一行
//     （Random=1，Amount/中心/半径随行——运行时按量生成）。
//   - 空值属性不收（与 XML 产物一致：产物不写空值）。
//
// eventtrigger.xml 是场景事件表（triggercreator），不是 NPC 摆放，排除。
// 单档损坏（提取源带坏档）只跳过该档并记入 skipped，不废整个场景。

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// ReadCreatorDir 场景摆放目录 → 属性行（各档按文件名排序、档内文档序）。
// files 是读入的档数；目录不存在时 err 非 nil。
func ReadCreatorDir(dir string) (rows []map[string]string, files int, skipped []string, err error) {
	entries, err := res.ReadDir(dir)
	if err != nil {
		return nil, 0, nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".xml") {
			continue
		}
		if strings.EqualFold(e.Name(), "eventtrigger.xml") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		part, err := readCreatorFile(filepath.Join(dir, name))
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		rows = append(rows, part...)
		files++
	}
	return rows, files, skipped, nil
}

// readCreatorFile 单档 → 属性行；任何 XML 错误整档作废（已读出的行一并丢弃）。
func readCreatorFile(path string) ([]map[string]string, error) {
	raw, err := res.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data, err := DecodeText(raw)
	if err != nil {
		return nil, fmt.Errorf("解码: %w", err)
	}
	fileBase := filepath.Base(path)
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out []map[string]string
	var randomCreator []xml.Attr // 当前 <creator> 属性；nil 表示不在随机段
	for {
		token, err := dec.Token()
		if err == io.EOF {
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
					out = append(out, toRow(withFile(element.Attr, fileBase)))
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
func expandRandomItem(dec *xml.Decoder, creator, item []xml.Attr, fileBase string) []map[string]string {
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
		return []map[string]string{toRow(attrs)}
	}
	out := make([]map[string]string, 0, len(positions))
	for i, pos := range positions {
		attrs := withFile(mergeAttrs(merged, pos), fileBase)
		attrs = append(attrs,
			xml.Attr{Name: xml.Name{Local: "Random"}, Value: "1"},
			xml.Attr{Name: xml.Name{Local: "CreatorNo"}, Value: creatorNo},
			xml.Attr{Name: xml.Name{Local: "No"}, Value: fmt.Sprintf("%s:%s:%d", creatorNo, itemNo, i+1)})
		out = append(out, toRow(attrs))
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

// toRow 属性表 → 行（空值不收）。
func toRow(attrs []xml.Attr) map[string]string {
	row := make(map[string]string, len(attrs))
	for _, a := range attrs {
		if a.Value != "" {
			row[a.Name.Local] = a.Value
		}
	}
	return row
}

// withFile 副本 + File 溯源属性置首。
func withFile(attrs []xml.Attr, fileBase string) []xml.Attr {
	out := make([]xml.Attr, 0, len(attrs)+1)
	out = append(out, xml.Attr{Name: xml.Name{Local: "File"}, Value: fileBase})
	return append(out, attrs...)
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
