package gmpanel

// scenes_test.go /api/gm/scenes：GM 测试台场景下拉数据（scenedata 内嵌表）。

import (
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/scenedata"
)

func TestAPIGMScenes(t *testing.T) {
	world := fakeWorldEcho(t)
	s := newTestPanel(t, world, nil)
	code, body := get(t, "http://"+s.Addr().String()+"/api/gm/scenes")
	if code != 200 {
		t.Fatalf("scenes %d: %v", code, body)
	}
	raw, _ := body["scenes"].([]any)
	table := scenedata.Scenes()
	if len(raw) != len(table) {
		t.Fatalf("scenes = %d, want %d", len(raw), len(table))
	}
	// 逐场景对拍内嵌表（序 = 类序 + 编号）；锚一个场景的字段形态。
	for i, item := range raw {
		m, _ := item.(map[string]any)
		want := table[i]
		if m["key"] != want.Resource || m["config"] != want.Config || m["landing"] != want.Landing.Name ||
			m["categoryName"] != scenedata.CategoryName(want.Category) {
			t.Fatalf("scene %d = %v, want %+v", i, m, want.Resource)
		}
		waypoints, _ := m["waypoints"].([]any)
		if len(waypoints) != len(want.Waypoints) {
			t.Fatalf("%s waypoints = %d, want %d", want.Resource, len(waypoints), len(want.Waypoints))
		}
	}
	first, _ := raw[0].(map[string]any)
	if first["key"] != "city01" || first["name"] != "燕京" || first["category"] != "city" {
		t.Fatalf("first scene = %v", first)
	}
	wp, _ := first["waypoints"].([]any)[0].(map[string]any)
	if wp["name"] == "" || wp["x"] == nil || wp["orient"] == nil {
		t.Fatalf("waypoint shape = %v", wp)
	}
}
