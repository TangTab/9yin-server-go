package fssoloworld

// 场景切换单测：目的地解析/切换状态流（0x0C→gap→0x0B→0x0D→0x09 重入簇）/
// 切换期位移守卫/同场景传送。sceneSwitchGap 测试内调短。

import (
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

func TestResolveSceneDestination(t *testing.T) {
	// resource 短名：born03 → BornCJ 出生点。
	dest, ok := resolveSceneDestination("born03")
	if !ok || dest.Resource != "born03" || dest.Config != `ini\scene\born03_YanYuZhuang` {
		t.Fatalf("born03 resolve = %+v ok=%v", dest, ok)
	}
	if dest.Position == [4]float64{} {
		t.Fatal("born03 position empty")
	}
	// 中文别名：燕京 → city01。
	dest, ok = resolveSceneDestination("燕京")
	if !ok || dest.Resource != "city01" || dest.Config != `ini\scene\city01_YanJing` {
		t.Fatalf("alias resolve = %+v ok=%v", dest, ok)
	}
	// 直填形态：config|resource（坐标零值，调用方须覆盖）。
	dest, ok = resolveSceneDestination(`ini\scene\clone001|clone001`)
	if !ok || dest.Resource != "clone001" || dest.Position != [4]float64{} {
		t.Fatalf("direct resolve = %+v ok=%v", dest, ok)
	}
	if _, ok := resolveSceneDestination("不存在场景"); ok {
		t.Fatal("unknown key should not resolve")
	}
}

func TestSceneSwitchFlow(t *testing.T) {
	s, pl := newCustomTestServer(t)
	prevGap := sceneSwitchGap
	sceneSwitchGap = time.Millisecond
	t.Cleanup(func() { sceneSwitchGap = prevGap })

	frames := captureFrames(t)
	dest, _ := resolveSceneDestination("born01")
	dest.Position = [4]float64{990, 6.66, 532, 4.7}

	if err := s.transferPlayer(pl, dest, "test"); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	// 立即段：0x0C 已发 + 权威态已落（Scene/Resource/Position）。
	if len(*frames) == 0 || (*frames)[0][0] != 0x0C {
		t.Fatalf("first frame op = % x, want 0x0C", opOf(*frames))
	}
	if pl.roleState.Resource != "born01" || pl.roleState.Scene != dest.Config ||
		pl.roleState.Position != dest.Position {
		t.Fatalf("role state not updated: scene=%s resource=%s pos=%v",
			pl.roleState.Scene, pl.roleState.Resource, pl.roleState.Position)
	}

	// 切换在途：旧场景位移上报丢弃（2433 过期 0x0A 坑）。
	s.absorbPlayerPosition(pl, [4]float64{1, 1, 1, 0})
	if pl.roleState.Position != dest.Position {
		t.Fatalf("stale movement leaked during switch: %v", pl.roleState.Position)
	}

	// gap 后 0x0B/0x0D 到位 → 就绪屏障 awaitReady。
	deadline := time.Now().Add(2 * time.Second)
	for s.pendingSceneSwitch(pl) == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.pendingSceneSwitch(pl) == nil {
		t.Fatal("awaitReady not armed after gap")
	}
	ops := opOf(*frames)
	has0B, has0D := false, false
	for _, op := range ops {
		if op == 0x0B {
			has0B = true
		}
		if op == 0x0D {
			has0D = true
		}
	}
	if !has0B || !has0D {
		t.Fatalf("pre-ready frames missing 0x0B/0x0D: % x", ops)
	}

	// 目标场景 0x09 → 重入簇：0x1F + OpenMapRec(0x11 rec51) + 邮件行重放。
	pl.roleState.Letters = []rolestate.Letter{NewGMLetter("切图", "重放", "")}
	mark := len(*frames)
	s.clientReady(pl)
	if s.sceneSwitchInFlight(pl) {
		t.Fatal("switch state not cleared after reentry ready")
	}
	reentryOps := opOf((*frames)[mark:])
	has1F, hasOpenMap, hasLetter := false, false, false
	for i, op := range reentryOps {
		if op == 0x1F {
			has1F = true
		}
		if op == 0x11 {
			switch recordIndexOf((*frames)[mark+i]) {
			case 51: // OpenMapRec
				hasOpenMap = true
			case recvLetterIndex():
				hasLetter = true
			}
		}
	}
	if !has1F || !hasOpenMap {
		t.Fatalf("reentry cluster missing 0x1F/OpenMapRec: % x", reentryOps)
	}
	if !hasLetter {
		t.Fatalf("reentry cluster did not replay letters: % x", reentryOps)
	}

	// 开图位已点亮新场景位（scenes.ini 真表解析，born01 应有全局段号）。
	idx := s.sceneGlobalIndexOf("born01")
	if idx >= 0 {
		row, bit := idx/32, uint(idx%32)
		if len(pl.roleState.OpenMaps) == 24 && pl.roleState.OpenMaps[row]&(1<<bit) == 0 {
			t.Fatalf("open map bit not set for born01 idx=%d", idx)
		}
	}
}

func TestSceneSwitchSameSceneTeleport(t *testing.T) {
	s, pl := newCustomTestServer(t)
	frames := captureFrames(t)
	dest := sceneDestination{
		Config:   pl.roleState.Scene,
		Resource: pl.roleState.Resource,
		Position: [4]float64{100, 7, 200, 1.5},
	}
	if err := s.transferPlayer(pl, dest, "test"); err != nil {
		t.Fatalf("teleport: %v", err)
	}
	if s.sceneSwitchInFlight(pl) {
		t.Fatal("same-scene teleport must not arm scene switch")
	}
	if pl.roleState.Position != dest.Position {
		t.Fatalf("position not updated: %v", pl.roleState.Position)
	}
	if len(*frames) != 1 || (*frames)[0][0] != 0x1F {
		t.Fatalf("teleport frames = % x, want single 0x1F", opOf(*frames))
	}
}

func TestSceneSwitchInProgressGuard(t *testing.T) {
	s, pl := newCustomTestServer(t)
	prevGap := sceneSwitchGap
	sceneSwitchGap = 50 * time.Millisecond // 在途窗口内做二次切换
	t.Cleanup(func() { sceneSwitchGap = prevGap })
	dest, _ := resolveSceneDestination("born01")
	if err := s.transferPlayer(pl, dest, "test"); err != nil {
		t.Fatalf("first transfer: %v", err)
	}
	if err := s.transferPlayer(pl, dest, "again"); err == nil {
		t.Fatal("second transfer during switch should fail")
	}
	time.Sleep(80 * time.Millisecond) // 等 goroutine 结束，防泄漏到下一个测试
}

// recvLetterIndex entergen.RecordRecvLetter 的本包便捷引用。
func recvLetterIndex() uint16 { return entergen.RecordRecvLetter }
