package fssoloworld

// npc_test.go 场景 NPC 全量物化目录测试（真档在仓根 testdata，cwd 探测
// ../../testdata 命中）。

import (
	"os"
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

func TestSceneNPCObjects0DBorn03(t *testing.T) {
	s := newNpcTestServer(t)
	objects := s.sceneNPCObjects("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(objects) == 0 {
		t.Fatal("born03 NPC 物化目录为空（真档缺席或装载失败——见日志）")
	}
	// 全量物化：联查命中 born03 定数 1759。
	if len(objects) != 1759 {
		t.Errorf("全量条数 = %d, want 1759", len(objects))
	}
	seen := map[uint32]bool{}
	for i, o := range objects {
		if o.ObjectID>>24 != 0x11 {
			t.Fatalf("条 %d objectID %x 不在 fork NPC 0x11 段", i, o.ObjectID)
		}
		if o.OwnerID>>16 != 0x0066 {
			t.Fatalf("条 %d ownerID %x 不在 0x66xxxx 段", i, o.OwnerID)
		}
		if seen[o.ObjectID] {
			t.Fatalf("objectID %x 重复", o.ObjectID)
		}
		seen[o.ObjectID] = true
		if o.Template["Resource"] == "" && o.Template["Material"] == "" {
			t.Fatalf("条 %d (%s) 模板无 Resource/Material", i, o.ConfigID)
		}
	}
	// 帧生成闭环（0x28 mode=0x46 官方言）。
	frames, err := entergen.NpcMaterializeFrames(objects, true)
	if err != nil {
		t.Fatalf("NpcMaterializeFrames: %v", err)
	}
	if len(frames) != len(objects) {
		t.Fatalf("帧数 %d ≠ 条数 %d", len(frames), len(objects))
	}
	for i, f := range frames {
		if f[0] != 0x28 || f[1] != 0x46 || f[13] != 0x11 {
			t.Fatalf("帧 %d 形态异常 op=%#x mode=%#x obj段=%#x", i, f[0], f[1], f[13])
		}
	}
	// 判别形态（裸 0x0D 大档）可切换。
	legacy, err := entergen.NpcMaterializeFrames(objects[:2], false)
	if err != nil || legacy[0][0] != 0x0D {
		t.Fatalf("裸 0x0D 判别形态异常: %v", err)
	}
	// 进程缓存：二次调用同集。
	again := s.sceneNPCObjects("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(again) != len(objects) || &again[0] != &objects[0] {
		t.Error("缓存未命中（重复装载）")
	}
}

func TestSceneNPCMarks229Retired(t *testing.T) {
	s := newNpcTestServer(t)
	t.Setenv("NINEYIN_NPC_MARK_229", "1")
	if got := s.sceneNPCMarks("born03", [4]float64{482.5, 13.7, 1008, 1.56}); got != nil {
		t.Fatalf("m229 已退役恒空（got %d 条）", len(got))
	}
}

func TestSceneNPCPushOff(t *testing.T) {
	t.Setenv("NINEYIN_NPC_PUSH", "off")
	s := newNpcTestServer(t)
	if got := s.sceneNPCObjects("born03", [4]float64{482.5, 13.7, 1008, 1.56}); got != nil {
		t.Fatalf("NINEYIN_NPC_PUSH=off 仍下发 %d 条", len(got))
	}
}

func TestResolveDataRootPrefersRes(t *testing.T) {
	t.Setenv("NINEYIN_DATA_ROOT", "")
	dir := t.TempDir()
	for _, sub := range []string{"Res", "testdata"} {
		if err := os.MkdirAll(dir+"/"+sub+"/ini/Npc/NpcConfig", 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	if got := resolveDataRoot(); got != "Res" {
		t.Fatalf("resolveDataRoot = %q, want Res（正式 $root Res 优先于 testdata）", got)
	}
}

func TestResolveDataRootFallbackTestdata(t *testing.T) {
	t.Setenv("NINEYIN_DATA_ROOT", "")
	dir := t.TempDir()
	if err := os.MkdirAll(dir+"/testdata/ini/Npc/NpcConfig", 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if got := resolveDataRoot(); got != "testdata" {
		t.Fatalf("resolveDataRoot = %q, want testdata（无 Res 时兜底）", got)
	}
}
