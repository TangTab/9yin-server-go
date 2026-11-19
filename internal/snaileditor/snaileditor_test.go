package snaileditor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMsgManager(t *testing.T) {
	m := NewMsgManager()
	if !m.Empty() {
		t.Fatal("初始应空")
	}
	m.PushMsg(0, 1, "a*b")
	m.PushMsg(0, 2, "c")
	if m.Len() != 2 || m.Empty() {
		t.Fatal("入队不符")
	}
	s, ok := m.PopMsg()
	if !ok || s.Data != "a*b" || s.X != 1 {
		t.Fatalf("弹出不符: %+v", s)
	}
	m.PopMsg()
	if _, ok := m.PopMsg(); ok {
		t.Fatal("空队应 false")
	}
}

func TestLoadModuleConfig(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SnailEditorServerModule.xml")
	xml := `<root>
<config SvnLocalRootPath="D:\ws\server" SvnServerIP="127.0.0.1" SvnServerPort="16010"/>
<watch>
  <path value="lua\"/>
  <path value="INI\Scene\"/>
</watch>
</root>`
	if err := os.WriteFile(p, []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadModuleConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Config.SvnServerIP != "127.0.0.1" || cfg.Config.SvnServerPort != 16010 ||
		cfg.Config.SvnLocalRootPath != `D:\ws\server` {
		t.Fatalf("config: %+v", cfg.Config)
	}
	wp := cfg.WatchPaths()
	if len(wp) != 2 || wp[0] != `lua\` || wp[1] != `ini\scene\` {
		t.Fatalf("watch: %q", wp)
	}
}

func TestHBProcessMsgReloadFlow(t *testing.T) {
	m := New()
	var got []string
	m.SetReloadHandler(`lua\`, func(ps []string) { got = append(got, "lua:"+strings.Join(ps, "|")) })
	m.SetReloadHandler(`ini\`, func(ps []string) { got = append(got, "ini:"+strings.Join(ps, "|")) })

	// PushMsg（Net 回调面）→ 心跳泵
	m.PushMsg(0, "lua\\quest\\a.lua*ini\\scene\\city.xml*res\\pack.bin*ini\\skill\\s.xml")
	if m.msgs.Len() != 1 {
		t.Fatal("入队不符")
	}
	if !m.HB_ProcessMsg() {
		t.Fatal("应处理消息")
	}
	if m.HB_ProcessMsg() {
		t.Fatal("空队不应处理")
	}
	// res\pack.bin 未注册 → 过滤；其余按前缀分组合并调用（lua 1 次 + ini 1 次）
	if len(got) != 2 {
		t.Fatalf("分发改动: %v", got)
	}
	joined := strings.Join(got, ";")
	if !strings.Contains(joined, "lua:lua\\quest\\a.lua") {
		t.Fatalf("lua 组: %v", got)
	}
	if !strings.Contains(joined, "ini:ini\\scene\\city.xml|ini\\skill\\s.xml") {
		t.Fatalf("ini 组应合并: %v", got)
	}
	if strings.Contains(joined, "pack.bin") {
		t.Fatalf("未注册路径泄漏: %v", got)
	}
}

func TestDelaySvnUpdateDedup(t *testing.T) {
	m := New()
	m.FuncDelaySvnUpdateCallback(0, `Lua\A.XML`)
	m.FuncDelaySvnUpdateCallback(0, "lua/a.xml") // 折叠后重复
	m.FuncDelaySvnUpdateCallback(0, "ini\\b.xml")
	got := m.TakeNeedSvnUpdates()
	if len(got) != 2 || got[0] != `lua\a.xml` || got[1] != `ini\b.xml` {
		t.Fatalf("%q", got)
	}
	if left := m.TakeNeedSvnUpdates(); len(left) != 0 {
		t.Fatalf("取出后应清空: %q", left)
	}
}

func TestCommandDispatch(t *testing.T) {
	m := New()
	if out, ok := m.RunCommand("CustomEditorStart", nil); !ok || out != "start" {
		t.Fatalf("start: %q %v", out, ok)
	}
	out, ok := m.RunCommand("CustomEditorWriteObjectBase", []string{"7", "Stage=1"})
	if !ok || !strings.Contains(out, "base") || !strings.Contains(out, "Stage=1") {
		t.Fatalf("writeobj: %q", out)
	}
	if _, ok := m.RunCommand("NoSuch", nil); ok {
		t.Fatal("未注册命令应 false")
	}
	// CustomEditorReloadRes 把路径串注入命令队列（官方 PushMsg 通路）
	m.RunCommand("CustomEditorReloadRes", []string{"lua\\x.lua", "lua\\y.lua"})
	if !m.HB_ProcessMsg() {
		t.Fatal("ReloadRes 应入队")
	}
}

func TestStartSnailEditorLoopback(t *testing.T) {
	// 不依赖网络：StartSnailEditor 需要 addr；起一个回环 Server 模拟 GUI。
	srv, err := startTestGUI()
	if err != nil {
		t.Skipf("回环监听失败: %v", err)
	}
	defer srv.Close()

	m := New()
	m.Trace = func(string, ...any) {} // 静音
	cfg := ModuleXML{}
	cfg.Config.SvnServerIP = "127.0.0.1"
	cfg.Config.SvnServerPort = 16010
	ok, err := m.StartSnailEditor(cfg, srv.Addr().String())
	if err != nil || !ok {
		t.Fatalf("StartSnailEditor: %v %v", ok, err)
	}
	defer m.Shut()
	if m.NetClient() == nil {
		t.Fatal("Net 未装配")
	}
}
