package fssoloworld

// 自定义配置域接收链单测：107 上传零应答 + ClientReady 前的登录期转储
// 不落库、快捷栏基线物化（starter 打坐条目「所见即所存」、删除不复活）、
// 0x1E/0x0A 双路由入口。下行帧正确性由 entergen.PreReadyFrames/
// PlayerObjectConfigFrames/ShortcutRebuildFrames 字节级测试覆盖（nil
// inner 下 sendToClient 静默）。

import (
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/entergen"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

func newCustomTestServer(t *testing.T) (*Server, *Player) {
	t.Helper()
	cfg := DefaultConfig()
	cfg.InnerAddr = "127.0.0.1"
	cfg.InnerPort = 0
	cfg.BroadPort = 0
	cfg.EchoAddr = ""
	s := New(Options{Config: cfg, ManualTick: true})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	state, err := rolestate.BuildRoleState(rolestate.DefaultBook, 1, "", 0)
	if err != nil {
		t.Fatalf("build role state: %v", err)
	}
	pl := s.Players().AddPlayer(7, 1, "acct")
	pl.roleState, pl.haveRoleState = state, true
	return s, pl
}

func customMsg(id int32, values ...entergen.CustomInboundValue) entergen.CustomInbound {
	return entergen.CustomInbound{Values: append([]entergen.CustomInboundValue{{Type: 2, Int32: id}}, values...)}
}

func customStr(v string) entergen.CustomInboundValue  { return entergen.CustomInboundValue{Type: 6, Text: v} }
func customInt(v int32) entergen.CustomInboundValue   { return entergen.CustomInboundValue{Type: 2, Int32: v} }
func customDouble(v float64) entergen.CustomInboundValue {
	return entergen.CustomInboundValue{Type: 5, Float64: v}
}

// TestCustomizingLoginDumpIgnored 107 上传零应答（官服口径）；ClientReady
// 之前的登录期转储不落库——有档时全份忽略（旧首传闸只挡首份，第 2/3
// 份把存档冲回旧值），无档时照收作基线；0x09 之后的编辑提交才落库。
func TestCustomizingLoginDumpIgnored(t *testing.T) {
	s, pl := newCustomTestServer(t)
	frames := captureFrames(t)
	pl.roleState.Customizing = "SAVED"
	// 登录期三份转储（本地态 / 应用 705 过程中的中间态）：全部忽略。
	for _, dump := range []string{"LOCAL", "COERCED-1", "COERCED-2"} {
		s.handleCustomizingUpload(pl, customMsg(107, customStr(dump)))
		if pl.roleState.Customizing != "SAVED" {
			t.Fatalf("login-time dump %q overwrote saved config: %q", dump, pl.roleState.Customizing)
		}
	}
	if len(*frames) != 0 {
		t.Fatalf("107 上传须零应答（回声触发 on_parser_customizing 重置链），实发 %d 帧", len(*frames))
	}
	// 0x09 之后的编辑提交：落库，同样零应答。
	pl.entryReady = true
	s.handleCustomizingUpload(pl, customMsg(107, customStr("EDIT")))
	if pl.roleState.Customizing != "EDIT" {
		t.Fatalf("edit after ClientReady rejected: %q", pl.roleState.Customizing)
	}
	if len(*frames) != 0 {
		t.Fatalf("编辑提交同样零应答，实发 %d 帧", len(*frames))
	}
	// 无档（新号）：登录期首份转储即基线。
	s2, pl2 := newCustomTestServer(t)
	s2.handleCustomizingUpload(pl2, customMsg(107, customStr("BASELINE")))
	if pl2.roleState.Customizing != "BASELINE" {
		t.Fatalf("baseline dump on fresh role rejected: %q", pl2.roleState.Customizing)
	}
}

// TestKeybindFirstUploadGate 195 首传闸仅 scheme 0（当前启用转储）；
// scheme 1/2 编辑首传直接落。
func TestKeybindFirstUploadGate(t *testing.T) {
	s, pl := newCustomTestServer(t)
	pl.roleState.Keybind = "1,65;"
	s.handleKeybindUpload(pl, customMsg(195, customInt(0), customStr("9,66;")))
	if pl.roleState.Keybind != "1,65;" {
		t.Fatalf("login dump overwrote saved keybind: %q", pl.roleState.Keybind)
	}
	// 方案一编辑（首传但 scheme≠0）：直接收。
	s.handleKeybindUpload(pl, customMsg(195, customInt(1), customStr("2,33;")))
	if pl.roleState.Keybind1 != "2,33;" {
		t.Fatalf("scheme1 edit rejected: %q", pl.roleState.Keybind1)
	}
	// 非法串（非可打印 ASCII）静默拒。
	s.handleKeybindUpload(pl, customMsg(195, customInt(0), customStr("1,\xff;")))
	if pl.roleState.Keybind != "1,65;" {
		t.Fatalf("invalid keybind accepted: %q", pl.roleState.Keybind)
	}
}

// TestOperateModeAndNeiGongBind 99/199：合法值落 roleState；非法值静默拒。
func TestOperateModeAndNeiGongBind(t *testing.T) {
	s, pl := newCustomTestServer(t)
	s.handleOperateMode(pl, customMsg(99, customInt(1)))
	if !pl.roleState.OperateModeSet || pl.roleState.OperateMode != 1 {
		t.Fatalf("operate mode: %+v", pl.roleState.OperateMode)
	}
	s.handleOperateMode(pl, customMsg(99, customInt(7)))
	if pl.roleState.OperateMode != 1 {
		t.Fatalf("invalid operate mode accepted: %d", pl.roleState.OperateMode)
	}
	s.handleNeiGongBind(pl, customMsg(199, customStr("1;0;2;"+repeatSemicolon(22))))
	if pl.roleState.NeiGongBind == "" {
		t.Fatal("valid neigong bind rejected")
	}
	s.handleNeiGongBind(pl, customMsg(199, customStr("1;2;")))
	if pl.roleState.NeiGongBind == "1;2;" {
		t.Fatal("invalid neigong bind accepted")
	}
}

func repeatSemicolon(n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += "0;"
	}
	return out
}

// TestShortcutEditLifecycle 200/201：首次编辑物化基线（starter 打坐
// 条目入列）→ 替换/删除 → 删除 starter 不复活。
func TestShortcutEditLifecycle(t *testing.T) {
	s, pl := newCustomTestServer(t)
	if got := pl.roleState.EffectiveShortcuts(); len(got) != 1 || got[0].ID != "zs_default_01" {
		t.Fatalf("fresh effective shortcuts: %+v", got)
	}
	// 首次编辑：基线物化 = starter 行 + 新行。
	s.handleShortcutSet(pl, customMsg(200, customInt(0), customStr("skill"), customStr("CS_wd_tjq01")))
	if len(pl.roleState.Shortcuts) != 2 || !pl.roleState.ShortcutsSet {
		t.Fatalf("baseline not materialized: %+v", pl.roleState.Shortcuts)
	}
	// 同值重设：无变更（行数不变）。
	s.handleShortcutSet(pl, customMsg(200, customInt(0), customStr("skill"), customStr("CS_wd_tjq01")))
	if len(pl.roleState.Shortcuts) != 2 {
		t.Fatalf("unchanged set mutated rows: %+v", pl.roleState.Shortcuts)
	}
	// 替换同索引行。
	s.handleShortcutSet(pl, customMsg(200, customInt(0), customStr("skill"), customStr("CS_wd_tjq02")))
	replaced := false
	for _, row := range pl.roleState.Shortcuts {
		if row.Index == 0 {
			replaced = row.ID == "CS_wd_tjq02"
		}
	}
	if !replaced || len(pl.roleState.Shortcuts) != 2 {
		t.Fatalf("replace failed: %+v", pl.roleState.Shortcuts)
	}
	// 删除 starter：不复活。
	s.handleShortcutRemove(pl, customMsg(201, customInt(9)))
	for _, row := range pl.roleState.EffectiveShortcuts() {
		if row.ID == "zs_default_01" {
			t.Fatalf("starter resurrected: %+v", pl.roleState.Shortcuts)
		}
	}
	// 删除缺失索引：静默忽略。
	s.handleShortcutRemove(pl, customMsg(201, customInt(77)))
	if len(pl.roleState.Shortcuts) != 1 {
		t.Fatalf("missing remove mutated rows: %+v", pl.roleState.Shortcuts)
	}
	// 非法输入拒收：越界索引/空 kind/double 枚举索引。
	s.handleShortcutSet(pl, customMsg(200, customInt(5000), customStr("skill"), customStr("x")))
	s.handleShortcutSet(pl, customMsg(200, customDouble(1), customStr("skill"), customStr("x")))
	if len(pl.roleState.Shortcuts) != 1 {
		t.Fatalf("invalid set accepted: %+v", pl.roleState.Shortcuts)
	}
}

// TestClientCustomRoutes 0x1E 直发与 0x0A 活动信封双入口同归分发；
// 畸形帧静默忽略不断会话。
func TestClientCustomRoutes(t *testing.T) {
	s, pl := newCustomTestServer(t)
	frame, err := entergen.CustomMessageFrame(107, entergen.CustomString("VIA1E"))
	if err != nil {
		t.Fatal(err)
	}
	if !s.clientCustom(pl, frame) {
		t.Fatal("0x1E route rejected")
	}
	if pl.roleState.Customizing != "VIA1E" {
		t.Fatalf("0x1E route: %q", pl.roleState.Customizing)
	}
	// 0x0A 信封：21B 头 + 裸类型列表体。
	inner, err := entergen.CustomMessageFrame(99, entergen.CustomInt(1))
	if err != nil {
		t.Fatal(err)
	}
	envelope := append([]byte{0x0A}, make([]byte, 20)...)
	envelope = append(envelope, inner[1:]...)
	if !s.clientActivity(pl, envelope) {
		t.Fatal("0x0A route rejected")
	}
	if !pl.roleState.OperateModeSet || pl.roleState.OperateMode != 1 {
		t.Fatalf("0x0A route: %+v", pl.roleState.OperateMode)
	}
	// 畸形 0x1E：解析失败静默（返回 true 不断会话）。
	if !s.clientCustom(pl, []byte{0x1E, 0xFF, 0xFF}) {
		t.Fatal("malformed frame killed session")
	}
}

// TestCustomRejectedWithoutRoleState 无 roleState（旧档 replay 会话）：
// 编辑拒收不落空账号。
func TestCustomRejectedWithoutRoleState(t *testing.T) {
	s, pl := newCustomTestServer(t)
	pl.haveRoleState = false
	s.handleCustomizingUpload(pl, customMsg(107, customStr("X")))
	s.handleShortcutSet(pl, customMsg(200, customInt(0), customStr("skill"), customStr("x")))
	if pl.roleState.Customizing != "" || len(pl.roleState.Shortcuts) != 0 {
		t.Fatalf("edits accepted without role state: %+v", pl.roleState)
	}
}
// TestCustomizingPoisonedUploadRejected 客户端本地配置被历史坏串污染后，
// 转储里原样带着 U+FFFD——拒收不落库，后续干净上传照常入库；有干净
// 存档时无论登录期还是 ClientReady 之后，坏串都不得覆盖。
func TestCustomizingPoisonedUploadRejected(t *testing.T) {
	s, pl := newCustomTestServer(t)
	poisoned := "\x01e\ufffd\x02\ufffd\ufffd"
	s.handleCustomizingUpload(pl, customMsg(107, customStr(poisoned)))
	if pl.roleState.Customizing != "" {
		t.Fatalf("poisoned dump persisted: %q", pl.roleState.Customizing)
	}
	clean := "\x01e\x01\x02\xa1\xe5"
	s.handleCustomizingUpload(pl, customMsg(107, customStr(clean)))
	if string(pl.roleState.Customizing) != clean {
		t.Fatalf("clean dump after poison rejected: %q", pl.roleState.Customizing)
	}

	state, err := rolestate.BuildRoleState(rolestate.DefaultBook, 1, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	saved := rolestate.BinaryString("\x01SAVED\xa1")
	state.Customizing = saved
	pl2 := s.Players().AddPlayer(8, 2, "acct")
	pl2.roleState, pl2.haveRoleState = state, true
	s.handleCustomizingUpload(pl2, customMsg(107, customStr(poisoned)))
	if pl2.roleState.Customizing != saved {
		t.Fatalf("saved overwritten by poisoned dump: %q", pl2.roleState.Customizing)
	}
	pl2.entryReady = true // ClientReady 之后的坏串编辑同样拒收
	s.handleCustomizingUpload(pl2, customMsg(107, customStr(poisoned)))
	if pl2.roleState.Customizing != saved {
		t.Fatalf("saved overwritten by poisoned edit: %q", pl2.roleState.Customizing)
	}
}
