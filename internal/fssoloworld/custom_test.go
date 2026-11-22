package fssoloworld

// 自定义配置域接收链单测：首传闸（登录期默认转储不覆盖存档）、
// 快捷栏基线物化（starter 打坐条目「所见即所存」、删除不复活）、
// 0x1E/0x0A 双路由入口。下行帧正确性由 entergen.CustomConfigFrames/
// ShortcutRebuildFrames 字节级测试覆盖（nil inner 下 sendToClient 静默）。

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
	state, err := rolestate.BuildRoleState(rolestate.DefaultBirthScene, 1, "", 0)
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

// TestCustomizingFirstUploadGate 首传闸：库存有档时登录期默认转储被跳过
// （回推存档），编辑提交照常收。
func TestCustomizingFirstUploadGate(t *testing.T) {
	s, pl := newCustomTestServer(t)
	pl.roleState.Customizing = "SAVED"
	s.handleCustomizingUpload(pl, customMsg(107, customStr("DEFAULT")))
	if pl.roleState.Customizing != "SAVED" {
		t.Fatalf("login dump overwrote saved config: %q", pl.roleState.Customizing)
	}
	if !pl.customizingSeen {
		t.Fatal("first-upload gate not consumed")
	}
	s.handleCustomizingUpload(pl, customMsg(107, customStr("NEW")))
	if pl.roleState.Customizing != "NEW" {
		t.Fatalf("edit rejected: %q", pl.roleState.Customizing)
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
