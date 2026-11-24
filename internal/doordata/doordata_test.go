package doordata

// 门传送路线表装载单测：内嵌 door_routes.xml 的路径形态与规模锚点。

import (
	"strings"
	"testing"
)

// TestRouteTargetConfigSingleSlashes 路线目标路径 = 单反斜杠。源 XML 由
// gen_door_routes 以 %q 写出（反斜杠成对），装载时还原——切图 0x0B 的
// ConfigID 原样下发，形态须与 gm scene（ini\scene\born03_YanYuZhuang）及
// 客户端 door_data.ini 的 TargetScene 一致。
func TestRouteTargetConfigSingleSlashes(t *testing.T) {
	checked, id4 := 0, ""
	for scene, routes := range routesByScene {
		for _, r := range routes {
			checked++
			if strings.Contains(r.TargetConfig, `\\`) || !strings.HasPrefix(r.TargetConfig, `ini\scene\`) {
				t.Errorf("%s route %s targetConfig = %q", scene, r.ID, r.TargetConfig)
			}
			if r.ID == "4" {
				id4 = r.TargetConfig
			}
		}
	}
	if checked == 0 {
		t.Fatal("no routes loaded")
	}
	if id4 != `ini\scene\city02_SuZhou` {
		t.Fatalf("route 4 targetConfig = %q, want ini\\scene\\city02_SuZhou", id4)
	}
}

// TestRouteTableAnchors 规模锚点（door_data.ini 提取：113 条 / 44 场景）。
func TestRouteTableAnchors(t *testing.T) {
	if RouteCount() != 113 || SceneCount() != 44 {
		t.Fatalf("routes = %d scenes = %d, want 113 / 44", RouteCount(), SceneCount())
	}
}
