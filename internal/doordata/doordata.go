// Package doordata 门传送路线表（客户端 share/rule/door_data.ini 提取，
// tools/gen_door_routes 生成 door_routes.xml 后 go:embed——服务端零运行
// 时客户端依赖）。
//
// 每条路线：DoorPackageID 节 → 源场景（SceneName=resource 短名）触发圆
// （MoveTargetX/Y/Z + 半径，SpringRange 缺省口径 8）+ 目的地（TargetScene
// 全路径 + TargetX/Y/Z/Orient）。走近触发（portalAt）与门点击共用。
package doordata

import (
	_ "embed"
	"encoding/xml"
	"strings"
)

//go:embed door_routes.xml
var doorRoutesXML []byte

// Route 一条传送路线。
type Route struct {
	ID           string  `xml:"id,attr"`
	TriggerX     float64 `xml:"triggerX,attr"`
	TriggerY     float64 `xml:"triggerY,attr"`
	TriggerZ     float64 `xml:"triggerZ,attr"`
	Radius       float64 `xml:"radius,attr"`
	TargetConfig string  `xml:"targetConfig,attr"`
	TargetX      float64 `xml:"targetX,attr"`
	TargetY      float64 `xml:"targetY,attr"`
	TargetZ      float64 `xml:"targetZ,attr"`
	TargetOrient float64 `xml:"targetOrient,attr"`
}

type sceneRoutesXML struct {
	Name  string  `xml:"name,attr"`
	Route []Route `xml:"Route"`
}

type doorRoutesDocXML struct {
	XMLName xml.Name         `xml:"DoorRoutes"`
	Scene   []sceneRoutesXML `xml:"Scene"`
}

// routesByScene 装载缓存（进程单次）。
var routesByScene = func() map[string][]Route {
	var doc doorRoutesDocXML
	if err := xml.Unmarshal(doorRoutesXML, &doc); err != nil {
		panic("doordata: door_routes.xml parse: " + err.Error())
	}
	out := make(map[string][]Route, len(doc.Scene))
	for _, scene := range doc.Scene {
		out[scene.Name] = append(out[scene.Name], scene.Route...)
	}
	return out
}()

// RoutesForScene 源场景（resource 短名，如 city01）的全部路线（大小写
// 不敏感；无路线 nil）。
func RoutesForScene(resource string) []Route {
	resource = strings.TrimSpace(resource)
	for key, routes := range routesByScene {
		if strings.EqualFold(key, resource) {
			return routes
		}
	}
	return nil
}

// SceneCount 已装载源场景数（诊断/测试面）。
func SceneCount() int { return len(routesByScene) }

// RouteCount 路线总数。
func RouteCount() int {
	n := 0
	for _, routes := range routesByScene {
		n += len(routes)
	}
	return n
}

// ResourceOfConfig 场景配置路径 → resource 短名（2433 resourceForDoorScene
// 规则：city/school/born/scene 前缀截到首个 '_'，city02_SuZhou→city02；
// 其余（clone/adv 副本图）保持基名全形）。
func ResourceOfConfig(config string) string {
	seg := config
	if i := strings.LastIndexAny(seg, `\/`); i >= 0 {
		seg = seg[i+1:]
	}
	if i := strings.IndexByte(seg, '_'); i > 0 {
		head := seg[:i] // city02 / school10 / adv156 …
		for _, prefix := range []string{"city", "school", "born", "scene"} {
			if len(head) > len(prefix) && strings.EqualFold(head[:len(prefix)], prefix) {
				return head
			}
		}
	}
	return seg
}
