package fssoloworld

// npc_test.go 场景 NPC 目录装载链（真档在仓根 testdata，cwd 探测
// ../../testdata 命中）。

import (
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
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

func TestSceneNPCCatalogBorn03(t *testing.T) {
	s := newNpcTestServer(t)
	entries := s.sceneNPCCatalog("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(entries) == 0 {
		t.Fatal("born03 NPC 目录为空（真档缺席或装载失败——见日志）")
	}
	if len(entries) > 200 {
		t.Errorf("条数 %d 超上限 200", len(entries))
	}
	seen := map[uint32]bool{}
	named, unnamed := 0, 0
	for i, e := range entries {
		if e.ObjectID>>24 != 0x03 {
			t.Fatalf("条 %d objectID %x 不在 0x03 段", i, e.ObjectID)
		}
		if e.OwnerID < 0x2000 || e.OwnerID >= 0x3000 {
			t.Fatalf("条 %d ownerID %x 不在 0x2xxx 段", i, e.OwnerID)
		}
		if seen[e.ObjectID] {
			t.Fatalf("objectID %x 重复", e.ObjectID)
		}
		seen[e.ObjectID] = true
		if e.Resource == "" {
			t.Fatalf("条 %d 资源空", i)
		}
		if e.Layer != 302 {
			t.Errorf("条 %d Layer=%d, want 302", i, e.Layer)
		}
		if e.Name != "" {
			named++
		} else {
			unnamed++
		}
	}
	if named == 0 || unnamed == 0 {
		t.Errorf("人物/物件混合预期落空：named=%d unnamed=%d", named, unnamed)
	}
	// 帧生成闭环（进场链就绪段形态）。
	frames, err := entergen.NpcListFrames(entries)
	if err != nil {
		t.Fatalf("NpcListFrames: %v", err)
	}
	if len(frames) != len(entries) {
		t.Fatalf("帧数 %d ≠ 条数 %d", len(frames), len(entries))
	}
	// 进程缓存：二次调用同集。
	again := s.sceneNPCCatalog("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(again) != len(entries) || (len(entries) > 0 && &again[0] != &entries[0]) {
		t.Error("缓存未命中（重复装载）")
	}
}

func TestSceneNPCCatalogPushOff(t *testing.T) {
	t.Setenv("NINEYIN_NPC_PUSH", "off")
	s := newNpcTestServer(t)
	if got := s.sceneNPCCatalog("born03", [4]float64{482.5, 13.7, 1008, 1.56}); got != nil {
		t.Fatalf("NINEYIN_NPC_PUSH=off 仍下发 %d 条", len(got))
	}
}
