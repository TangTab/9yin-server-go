package fssoloworld

// npc_test.go 场景 NPC 帧族测试（构建产物在仓根 testdata/ini，cwd 探测
// ../../testdata 命中；npcfeed 数据根同链）。

import (
	"os"
	"testing"
)

func newNpcTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadPort = 0
	cfg.EchoAddr = ""
	return New(Options{Config: cfg, ManualTick: true, Logf: func(format string, args ...any) {
		t.Logf(format, args...)
	}})
}

func TestSceneNPCFramesBorn03(t *testing.T) {
	s := newNpcTestServer(t)
	frames := s.sceneNPCFrames("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(frames) == 0 {
		t.Fatal("born03 NPC 帧族为空（真档缺席或装载失败——见日志）")
	}
	for i, f := range frames {
		if f[0] != 0x28 || f[1] != 0x46 {
			t.Fatalf("帧 %d 形态异常 op=%#x（默认应 0x28 mode=0x46）", i, f[0])
		}
		if seg := uint32(f[10]) | uint32(f[11])<<8 | uint32(f[12])<<16 | uint32(f[13])<<24; seg>>24 != 0x11 {
			t.Fatalf("帧 %d objectID 段异常 %x", i, seg)
		}
	}
	// 进程缓存：二次调用同规模。
	again := s.sceneNPCFrames("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(again) != len(frames) {
		t.Errorf("缓存未命中（二次装载规模 %d ≠ %d）", len(again), len(frames))
	}
}

func TestSceneNPCFramesRadiusLimit(t *testing.T) {
	s := newNpcTestServer(t)
	t.Setenv("NINEYIN_NPC_RADIUS", "50")
	t.Setenv("NINEYIN_NPC_LIMIT", "5")
	frames := s.sceneNPCFrames("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(frames) == 0 || len(frames) > 5 {
		t.Fatalf("半径 50/上限 5 过滤后帧数 = %d", len(frames))
	}
}

func TestSceneNPCPushOff(t *testing.T) {
	s := newNpcTestServer(t)
	t.Setenv("NINEYIN_NPC_PUSH", "off")
	if got := s.sceneNPCFrames("born03", [4]float64{482, 13, 1008, 1}); got != nil {
		t.Fatalf("PUSH=off 应静默，得 %d 帧", len(got))
	}
}

func TestNpcEnvHelpers(t *testing.T) {
	t.Setenv("NINEYIN_NPC_0X28", "off")
	if npcAs0x28Enabled() {
		t.Error("NINEYIN_NPC_0X28=off 应切裸 0x0D")
	}
	os.Unsetenv("NINEYIN_NPC_0X28")
	if !npcAs0x28Enabled() {
		t.Error("缺省应 0x28 官方言")
	}
	if npcRadius() != 0 || npcLimit() != 0 {
		t.Error("缺省半径/上限应为 0（全量）")
	}
}
