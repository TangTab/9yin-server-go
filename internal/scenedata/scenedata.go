// Package scenedata 场景/驿站表：客户端资源解包树 × 服务端摆放档的离线
// 提取物（tools/gen_scene_waypoints → scene_waypoints.xml，go:embed——world
// 与面板共用，零运行时数据依赖）。
//
// 覆盖 = 客户端有地形的全部世界场景 city/born/school/sceneNN（变体地形
// school01_mie / scene08_2 不计）。GM 场景切换目的地在 BornCJ 出生点表 /
// 补充表查不到时的数据驱动回落：Resource + Config + 默认落点（首个安全
// 驿站；无驿站场景 = 传送门目标 / 摆放档功能 NPC 旁）。驿站序供面板按点
// 传送（gm scene <who> <key> x y z orient）。
package scenedata

import (
	_ "embed"
	"encoding/xml"
	"fmt"
	"math"
	"strings"
)

// 落点类别。
const (
	KindHome = "home" // 驿站 / 复活点（客户端 homepoint.ini 等表）
	KindDoor = "door" // 传送门路线目标
	KindNPC  = "npc"  // 摆放档功能 NPC 旁

	// 类别键（生成表 category 属性；序 = 列表序）。
	CategoryCity   = "city"
	CategoryBorn   = "born"
	CategorySchool = "school"
	CategoryScene  = "scene"
)

//go:embed scene_waypoints.xml
var tableXML []byte

// Waypoint 一个落点。
type Waypoint struct {
	Name   string // 表内名：HomePointcity01A / 复活点行键 / door:<id> / npc:<id>（Scene.Landing 引用键）
	Title  string // 中文名：驿站文案 / NPC 描述（缺席空）
	X      float64
	Y      float64
	Z      float64
	Orient float64 // 朝向（弧度；表里没有则 0）
	Kind   string  // KindHome / KindDoor / KindNPC
	Safe   bool    // 安全区（homepoint.ini Safe=1）
	Source string  // 来源：复活点行键 / 传送门 id / NPC 档
}

// Scene 一个世界场景。
type Scene struct {
	Resource  string // 地形资源名（school09）
	Config    string // 服务端场景配置路径（ini\scene\school09_YiHuaGong）
	Name      string // 中文名（文案表缺席为空）
	Category  string // CategoryCity / CategoryBorn / CategorySchool / CategoryScene
	Landing   Waypoint
	Waypoints []Waypoint // 含 Landing
}

// categories 类序（列表 / 面板分组顺序）与中文名。
var categories = []struct{ key, name string }{
	{CategoryCity, "主城"},
	{CategoryBorn, "出生镇"},
	{CategorySchool, "门派"},
	{CategoryScene, "江湖"},
}

var (
	scenes     = mustLoad(tableXML)
	byResource = func() map[string]Scene {
		out := make(map[string]Scene, len(scenes))
		for _, sc := range scenes {
			out[sc.Resource] = sc
		}
		return out
	}()
	// byName 中文名 → 场景；重名（凌霄城 ×2、念萝坝 ×2、星垂城 ×2）不收录——
	// 按名解析有歧义，须用 resource 短名。
	byName = func() map[string]Scene {
		out, dup := map[string]Scene{}, map[string]bool{}
		for _, sc := range scenes {
			if sc.Name == "" {
				continue
			}
			if _, seen := out[sc.Name]; seen {
				dup[sc.Name] = true
				continue
			}
			out[sc.Name] = sc
		}
		for name := range dup {
			delete(out, name)
		}
		return out
	}()
)

// Scenes 全部场景（类序 + 编号；调用方只读）。
func Scenes() []Scene { return scenes }

// Find 按 resource 短名查（大小写不敏感、首尾空白容忍；缺席 false）。
func Find(resource string) (Scene, bool) {
	sc, ok := byResource[strings.ToLower(strings.TrimSpace(resource))]
	return sc, ok
}

// FindByName 按中文名精确查（重名不命中，见 byName；缺席 false）。
func FindByName(name string) (Scene, bool) {
	sc, ok := byName[name]
	return sc, ok
}

// Categories 类序 [(键, 中文名)]。
func Categories() [][2]string {
	out := make([][2]string, 0, len(categories))
	for _, c := range categories {
		out = append(out, [2]string{c.key, c.name})
	}
	return out
}

// CategoryName 类别键 → 中文名（未知回退键本身）。
func CategoryName(category string) string {
	for _, c := range categories {
		if c.key == category {
			return c.name
		}
	}
	return category
}

// ---------- 装载（生成物契约） ----------

type waypointXML struct {
	Name   string  `xml:"name,attr"`
	Title  string  `xml:"title,attr"`
	X      float64 `xml:"x,attr"`
	Y      float64 `xml:"y,attr"`
	Z      float64 `xml:"z,attr"`
	Orient float64 `xml:"orient,attr"`
	Kind   string  `xml:"kind,attr"`
	Safe   bool    `xml:"safe,attr"`
	Source string  `xml:"source,attr"`
}

type sceneXML struct {
	Resource string        `xml:"resource,attr"`
	Config   string        `xml:"config,attr"`
	Name     string        `xml:"name,attr"`
	Category string        `xml:"category,attr"`
	Landing  string        `xml:"landing,attr"`
	Waypoint []waypointXML `xml:"Waypoint"`
}

type docXML struct {
	Scene []sceneXML `xml:"Scene"`
}

// mustLoad 解析 + 结构校验。产物由 tools/gen_scene_waypoints -verify 与本包
// 测试把关；启动即失败好过运行到 GM 传送时才暴露。
func mustLoad(data []byte) []Scene {
	var doc docXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		panic("scenedata: scene_waypoints.xml parse: " + err.Error())
	}
	if len(doc.Scene) == 0 {
		panic("scenedata: scene_waypoints.xml has no scene")
	}
	known := map[string]bool{}
	for _, c := range categories {
		known[c.key] = true
	}
	seen := map[string]bool{}
	out := make([]Scene, 0, len(doc.Scene))
	for _, sx := range doc.Scene {
		fail := func(format string, args ...any) {
			panic(fmt.Sprintf("scenedata: scene %q: %s", sx.Resource, fmt.Sprintf(format, args...)))
		}
		key := strings.ToLower(sx.Resource)
		switch {
		case key == "":
			fail("empty resource")
		case seen[key]:
			fail("duplicate resource")
		case sx.Config == "":
			fail("empty config")
		case !known[sx.Category]:
			fail("unknown category %q", sx.Category)
		}
		seen[key] = true
		sc := Scene{Resource: key, Config: sx.Config, Name: sx.Name, Category: sx.Category}
		for _, wx := range sx.Waypoint {
			if wx.X == 0 && wx.Y == 0 && wx.Z == 0 {
				fail("waypoint %q at origin（源表缺坐标）", wx.Name)
			}
			for _, v := range [4]float64{wx.X, wx.Y, wx.Z, wx.Orient} {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					fail("waypoint %q has non-finite coordinate", wx.Name)
				}
			}
			sc.Waypoints = append(sc.Waypoints, Waypoint{
				Name: wx.Name, Title: wx.Title, X: wx.X, Y: wx.Y, Z: wx.Z, Orient: wx.Orient,
				Kind: wx.Kind, Safe: wx.Safe, Source: wx.Source,
			})
		}
		landing := -1
		for i, wp := range sc.Waypoints {
			if wp.Name == sx.Landing {
				landing = i
				break
			}
		}
		if landing < 0 {
			fail("landing %q not in waypoints", sx.Landing)
		}
		sc.Landing = sc.Waypoints[landing]
		out = append(out, sc)
	}
	return out
}
