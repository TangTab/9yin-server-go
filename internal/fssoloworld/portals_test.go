package fssoloworld

// 传送门单测：doordata 装载（入仓 XML）/resource 推导/圆判定触发执行器
// /冷却与点击专用路线跳过。

import (
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/doordata"
)

func TestDoorDataLoaded(t *testing.T) {
	if doordata.SceneCount() == 0 || doordata.RouteCount() == 0 {
		t.Fatalf("door routes not loaded: scenes=%d routes=%d", doordata.SceneCount(), doordata.RouteCount())
	}
	// 出生场景应有可用路线（born03 烟雨庄——door_data 实测有就近门）。
	routes := doordata.RoutesForScene("born03")
	if len(routes) == 0 {
		t.Fatal("born03 has no door routes")
	}
	proximity := 0
	for _, r := range routes {
		if r.Radius > 0 {
			proximity++
		}
	}
	if proximity == 0 {
		t.Fatal("born03 has no proximity-capable routes")
	}
	if doordata.RoutesForScene("no-such-scene") != nil {
		t.Fatal("unknown scene should yield nil")
	}
}

func TestResourceOfConfig(t *testing.T) {
	cases := map[string]string{
		`ini\scene\city02_SuZhou`:      "city02",
		`ini\scene\born01_MoHeZhen`:    "born01",
		`ini\scene\school10_taohuadao`: "school10",
		`ini\scene\clone001`:           "clone001",       // 副本图全形
		`ini\scene\adv156_tianlun`:     "adv156_tianlun", // 江湖势力图全形
	}
	for config, want := range cases {
		if got := doordata.ResourceOfConfig(config); got != want {
			t.Fatalf("ResourceOfConfig(%q) = %q, want %q", config, got, want)
		}
	}
}

func TestCheckPortalTriggersTransfer(t *testing.T) {
	s, pl := newCustomTestServer(t)
	prevGap := sceneSwitchGap
	sceneSwitchGap = time.Millisecond
	t.Cleanup(func() { sceneSwitchGap = prevGap })

	// 数据驱动：取 born03 首条走近路线（radius>0），把玩家放到触发圆心。
	var route doordata.Route
	for _, r := range doordata.RoutesForScene("born03") {
		if r.Radius > 0 {
			route = r
			break
		}
	}
	if route.ID == "" {
		t.Fatal("born03 has no proximity route for test")
	}
	target := doordata.ResourceOfConfig(route.TargetConfig)
	sameScene := target == pl.roleState.Resource

	s.checkPortal(pl, [4]float64{route.TriggerX, route.TriggerY, route.TriggerZ, 0})
	if sameScene {
		if pl.roleState.Position[0] != route.TargetX || pl.roleState.Position[2] != route.TargetZ {
			t.Fatalf("same-scene portal did not move player: %+v", pl.roleState.Position)
		}
	} else {
		if pl.roleState.Resource != target || !s.sceneSwitchInFlight(pl) {
			t.Fatalf("cross-scene portal not triggered: resource=%s switching=%v",
				pl.roleState.Resource, s.sceneSwitchInFlight(pl))
		}
		// 等切换 goroutine 收尾（gap=1ms），防状态泄漏到其他测试。
		deadline := time.Now().Add(2 * time.Second)
		for s.sceneSwitchInFlight(pl) && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
	}

	// 冷却：冷却窗内再次进圈不重复触发（位置已变/或同场景锁定原位）。
	before := pl.roleState.Position
	pl.portalCoolAt = time.Now().Add(time.Hour) // 显式冷却窗
	s.checkPortal(pl, [4]float64{route.TriggerX, route.TriggerY, route.TriggerZ, 0})
	if pl.roleState.Position != before {
		t.Fatal("portal fired during cooldown")
	}
}
