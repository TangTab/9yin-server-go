package fssoloworld

// npc_test.go 场景 NPC 帧族测试（构建产物在仓根 testdata/ini，cwd 探测
// ../../testdata 命中；npcfeed 数据根同链）。

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

func newNpcTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadPort = 0
	cfg.EchoAddr = ""
	logf := func(format string, args ...any) {
		t.Logf(format, args...)
	}
	s := New(Options{Config: cfg, ManualTick: true, Logf: logf})
	// 交互链测试要 sendToClient（走 inner 广播）：挂无监听内网链
	//（零连接 → SendToAllEntry 空转安全）。
	s.r = reactor.New()
	s.inner = NewInnerLink(s.r, InnerEvents{}, "npc-test", logf)
	return s
}

func TestSceneNPCFramesBorn03(t *testing.T) {
	s := newNpcTestServer(t)
	frames := s.sceneNPCFrames("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(frames) == 0 {
		t.Fatal("born03 NPC 帧族为空（真档缺席或装载失败——见日志）")
	}
	// 缺省 0x28-only（dual 双发实证挤爆 RoleUserData 池，勿回缺省）。
	for i, f := range frames {
		if f[0] != 0x28 || f[1] != 0x46 {
			t.Fatalf("帧 %d 形态异常 op=%#x（缺省应 0x28 mode=0x46）", i, f[0])
		}
		if seg := binary.LittleEndian.Uint32(f[10:14]); seg>>24 != 0x11 {
			t.Fatalf("帧 %d objectID 段异常 %x", i, seg)
		}
	}
	// 进程缓存：二次调用同规模。
	again := s.sceneNPCFrames("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(again) != len(frames) {
		t.Errorf("缓存未命中（二次装载规模 %d ≠ %d）", len(again), len(frames))
	}
	// dual 显式开启：双发成对（0x0D 在前、0x28 在后）。
	t.Setenv("NINEYIN_NPC_FORM", "dual")
	dual := s.sceneNPCFrames("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	if len(dual) != len(frames)*2 {
		t.Fatalf("dual 帧数 = %d, want %d", len(dual), len(frames)*2)
	}
	for i := 0; i+1 < len(dual); i += 2 {
		if dual[i][0] != 0x0D || dual[i+1][0] != 0x28 {
			t.Fatalf("dual 帧 %d 形态异常 %#x/%#x", i, dual[i][0], dual[i+1][0])
		}
	}
}

func TestSceneNPCFramesRadiusLimit(t *testing.T) {
	s := newNpcTestServer(t)
	t.Setenv("NINEYIN_NPC_RADIUS", "50")
	t.Setenv("NINEYIN_NPC_LIMIT", "5")
	frames := s.sceneNPCFrames("born03", [4]float64{482.5, 13.7, 1008, 1.56})
	// 双发：每条 NPC 两帧（0x0D+0x28）。
	if len(frames) == 0 || len(frames) > 10 {
		t.Fatalf("半径 50/上限 5 过滤后帧数 = %d（双发上限 10）", len(frames))
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
	if npcForm() != npcfeed.Form0DOnly {
		t.Error("NINEYIN_NPC_0X28=off 应切裸 0x0D")
	}
	t.Setenv("NINEYIN_NPC_FORM", "0x28")
	if npcForm() != npcfeed.Form0x28Only {
		t.Error("NINEYIN_NPC_FORM=0x28 应切仅 0x28")
	}
	os.Unsetenv("NINEYIN_NPC_0X28")
	os.Unsetenv("NINEYIN_NPC_FORM")
	if npcForm() != npcfeed.Form0x28Only {
		t.Error("缺省应 0x28-only（dual 双发挤爆池，勿回缺省）")
	}
	if npcRadius() != 0 || npcLimit() != 0 {
		t.Error("缺省半径/上限应为 0（全量）")
	}
}
