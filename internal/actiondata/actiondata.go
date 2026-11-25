// Package actiondata 招式判定节拍产物的只读装载。
//
// 数据源是 tools/datagen action 的构建产物 ini/Action/zhaoshi_hits.xml
// （客户端招式表 ini/action/zhaoshi_*.ini 的离线编译：一招一行）：
//
//	ID      招式名（= 技能流程包 HitAction / m20 下发的动作名）
//	Actions 各段动作名
//	Frames  各段帧长（30fps）
//	Hits    判定时刻（毫秒，自招式起算；hitbreak ∪ hitback 同刻去重）
//	Stagger 与 Hits 逐拍对位：该拍的受击反应是否由服务端下发（判定触发器
//	        不带受击者动画）；旧产物无此列时为空，调用方按「不下发」处理
//	Length  招式总时长（毫秒）
//
// 战斗链按技能的 HitAction 取判定时刻调度每击结算（官服命中节拍的数据
// 源，见 docs/official/战斗接入与技能Lua宿主.md）。产物缺席时全部查询
// 落空，调用方按流程包/缺省节拍回退。
package actiondata

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

// FileName 产物文件名（相对数据根）。
const FileName = "zhaoshi_hits.xml"

// Action 一招的判定节拍。
type Action struct {
	ID      string
	Actions []string
	Frames  []int
	HitsMS  []int
	Stagger []bool
	Length  int
}

// Staggers 第 index 击（0 基）的受击反应是否由服务端下发；下标越界或产物
// 无 Stagger 列均为假。
func (a Action) Staggers(index int) bool {
	return index >= 0 && index < len(a.Stagger) && a.Stagger[index]
}

// Catalog 产物目录（进程级缓存）。
type Catalog struct {
	actions map[string]Action
}

var cache struct {
	once sync.Once
	cat  *Catalog
	err  error
}

// Load 装载产物（数据根候选：NINEYIN_DATA_ROOT / npcfeed 同款探测 → 仓内
// testdata），进程级缓存；全缺返回错误（调用方按无节拍数据回退）。
func Load() (*Catalog, error) {
	cache.once.Do(func() {
		for _, root := range roots() {
			path := root + "/ini/Action/" + FileName
			if _, err := res.Stat(path); err != nil {
				continue
			}
			cat, err := LoadDir(root + "/ini/Action")
			if err == nil {
				cache.cat = cat
				return
			}
			cache.err = err
		}
		if cache.cat == nil && cache.err == nil {
			cache.err = fmt.Errorf("actiondata: 招式节拍产物缺席（go run ./tools/datagen action）")
		}
	})
	return cache.cat, cache.err
}

// LoadDir 装载指定目录（不经进程级缓存；构建器 verify 用）。
func LoadDir(dir string) (*Catalog, error) {
	raw, err := res.ReadFile(dir + "/" + FileName)
	if err != nil {
		return nil, fmt.Errorf("actiondata: 读产物失败: %w", err)
	}
	return parse(raw)
}

// parse 一行一招的 <Property ID=… Actions=… Frames=… Hits=… Stagger=… Length=…/>。
func parse(raw []byte) (*Catalog, error) {
	type row struct {
		ID      string `xml:"ID,attr"`
		Actions string `xml:"Actions,attr"`
		Frames  string `xml:"Frames,attr"`
		Hits    string `xml:"Hits,attr"`
		Stagger string `xml:"Stagger,attr"`
		Length  string `xml:"Length,attr"`
	}
	decoder := xml.NewDecoder(strings.NewReader(string(raw)))
	cat := &Catalog{actions: map[string]Action{}}
	depth := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("actiondata: 产物解析失败: %w", err)
		}
		switch element := token.(type) {
		case xml.StartElement:
			depth++
			if element.Name.Local != "Property" {
				continue
			}
			var r row
			if err := decoder.DecodeElement(&r, &element); err != nil {
				return nil, fmt.Errorf("actiondata: 第 %d 行解析失败: %w", depth, err)
			}
			length, _ := strconv.Atoi(r.Length)
			action := Action{ID: r.ID, Actions: splitList(r.Actions), Frames: splitInts(r.Frames),
				HitsMS: splitInts(r.Hits), Stagger: splitFlags(r.Stagger), Length: length}
			// 列数不齐的产物不采信 Stagger（逐拍对位是它的全部意义）。
			if len(action.Stagger) != len(action.HitsMS) {
				action.Stagger = nil
			}
			if _, dup := cat.actions[r.ID]; !dup {
				cat.actions[r.ID] = action
			}
		case xml.EndElement:
			depth--
		}
	}
	return cat, nil
}

// Lookup 按招式名取节拍。
func (c *Catalog) Lookup(action string) (Action, bool) {
	a, ok := c.actions[action]
	return a, ok
}

// Count 招式数。
func (c *Catalog) Count() int { return len(c.actions) }

// roots 数据根候选：npcfeed.ResolveRoot（NINEYIN_DATA_ROOT / 部署 Res / 仓内
// testdata）之后再补仓内相对候选（skilldata 的 skillDataRoots 同款）。
func roots() []string {
	var out []string
	if root := npcfeed.ResolveRoot(); root != "" {
		out = append(out, root)
	}
	return append(out, "testdata", "../testdata", "../../testdata")
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ";")
}

func splitInts(s string) []int {
	parts := splitList(s)
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

// splitFlags Stagger 列（1/0 逐拍）；空列返回空。
func splitFlags(s string) []bool {
	parts := splitList(s)
	out := make([]bool, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part) == "1")
	}
	return out
}
