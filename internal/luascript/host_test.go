package luascript

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type logSink struct{ lines []string }

func (s *logSink) printf(format string, args ...any) {
	s.lines = append(s.lines, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func newTestHost(t *testing.T, dir string, sink *logSink) *Host {
	t.Helper()
	host := New(dir, sink.printf)
	if _, err := host.LoadDir(); err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	return host
}

func testEvent() *EventContext {
	return &EventContext{
		Player:  PlayerSnapshot{ID: 7, Name: "测试玩家", Faction: "emey", Silver: 999, SP: 42, Scene: "成都", X: 1.5, Y: 2, Z: 3},
		NPC:     NPCSnapshot{ID: 11, ConfigID: "Relivecity05B", Name: "向导"},
		Service: ServiceSnapshot{Mark: 0x1008, Label: "任务", Value: "", Source: "task"},
	}
}

func TestLoadDirRegistersHandlers(t *testing.T) {
	sink := &logSink{}
	host := newTestHost(t, "testdata", sink)
	for _, event := range []string{"npc_service", "decline", "echo_player"} {
		if !host.HasHandler(event) {
			t.Fatalf("expected handler for %q", event)
		}
	}
	if host.Loaded() != 1 {
		t.Fatalf("Loaded = %d, want 1", host.Loaded())
	}
}

func TestCallNPCService(t *testing.T) {
	sink := &logSink{}
	host := newTestHost(t, "testdata", sink)
	var notified string
	ec := testEvent()
	ec.Notify = func(brief, chapter string) error {
		notified = brief
		return nil
	}
	handled, err := host.Call("npc_service", ec)
	if err != nil || !handled {
		t.Fatalf("Call = (%v, %v), want (true, nil)", handled, err)
	}
	if want := "你好，测试玩家"; notified != want {
		t.Fatalf("notify brief = %q, want %q", notified, want)
	}
	if len(sink.lines) == 0 || !strings.Contains(sink.lines[0], "npc_service npc=Relivecity05B mark=4104") {
		t.Fatalf("unexpected log lines: %q", sink.lines)
	}
}

func TestCallExposesBoundTasks(t *testing.T) {
	sink := &logSink{}
	host := newTestHost(t, "testdata", sink)
	ec := testEvent()
	ec.Tasks = []TaskSnapshot{
		{ID: 75730, Title: "虎啸疑云"},
		{ID: 75731, Title: "旧日仇恨", Target: "击败旧敌"},
	}
	if _, err := host.Call("echo_player", ec); err != nil {
		t.Fatal(err)
	}
	// tasks 为空时 ctx:tasks() 必须返回空表而不是 nil。
	empty := testEvent()
	if _, err := host.Call("echo_player", empty); err != nil {
		t.Fatal(err)
	}
	var taskLines []string
	for _, line := range sink.lines {
		if strings.Contains(line, "tasks=") {
			taskLines = append(taskLines, line)
		}
	}
	if len(taskLines) != 2 {
		t.Fatalf("expected task log lines for both calls, got %q", sink.lines)
	}
	if !strings.Contains(taskLines[0], "tasks=2 first=75730:虎啸疑云") {
		t.Fatalf("tasks line = %q", taskLines[0])
	}
	if !strings.Contains(taskLines[1], "tasks=0") {
		t.Fatalf("empty tasks line = %q", taskLines[1])
	}
}

func TestCallDeclinesAndMissing(t *testing.T) {
	host := newTestHost(t, "testdata", &logSink{})
	if handled, err := host.Call("decline", testEvent()); handled || err != nil {
		t.Fatalf("decline Call = (%v, %v), want (false, nil)", handled, err)
	}
	if handled, err := host.Call("no_such_event", testEvent()); handled || err != nil {
		t.Fatalf("missing Call = (%v, %v), want (false, nil)", handled, err)
	}
	if host.HasHandler("no_such_event") {
		t.Fatal("HasHandler for unknown event should be false")
	}
}

func TestEval(t *testing.T) {
	host := newTestHost(t, "testdata", &logSink{})
	cases := []struct {
		src  string
		want string
	}{
		{"return 1 + 2", "3"},
		{`return "abc", 4`, "abc\t4"},
		{"return { a = 1, 2, 3 }", "{1=2, 2=3, a=1}"},
		{"return type(io)", "nil"},
		{"return type(os)", "nil"},
		{"global t5 = 5 return t5", "5"}, // Lua 5.5 global 声明
	}
	for _, tc := range cases {
		got, err := host.Eval(tc.src)
		if err != nil {
			t.Fatalf("Eval(%q): %v", tc.src, err)
		}
		if got != tc.want {
			t.Fatalf("Eval(%q) = %q, want %q", tc.src, got, tc.want)
		}
	}
}

func TestEvalRecoversAfterError(t *testing.T) {
	host := newTestHost(t, "testdata", &logSink{})
	if _, err := host.Eval("error('boom')"); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected boom error, got %v", err)
	}
	// 运行期错误后 VM 重建，脚本处理器应恢复注册。
	if !host.HasHandler("npc_service") {
		t.Fatal("handlers lost after runtime error")
	}
	if got, err := host.Eval("return 1 + 1"); err != nil || got != "2" {
		t.Fatalf("Eval after error = (%q, %v)", got, err)
	}
	if _, err := host.Eval("retur oops"); err == nil {
		t.Fatal("expected syntax error")
	}
	if got, err := host.Eval("return 'still-alive'"); err != nil || got != "still-alive" {
		t.Fatalf("Eval after syntax error = (%q, %v)", got, err)
	}
}

func TestInfiniteLoopTimesOut(t *testing.T) {
	host := newTestHost(t, "testdata/loop", &logSink{})
	host.timeout = 150 * time.Millisecond
	start := time.Now()
	handled, err := host.Call("loop", testEvent())
	if err == nil {
		t.Fatal("expected timeout error from loop handler")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("loop handler took %v to cancel", elapsed)
	}
	if !handled {
		t.Fatal("a registered handler that errored must report handled=true")
	}
	// 取消后 VM 重建并重新加载脚本目录，处理器应恢复注册。
	if !host.HasHandler("loop") {
		t.Fatal("handlers should be restored after rebuild")
	}
	if got, err := host.Eval("return 'recovered'"); err != nil || got != "recovered" {
		t.Fatalf("Eval after timeout = (%q, %v)", got, err)
	}
}

func TestNotifyErrorSurfaces(t *testing.T) {
	host := newTestHost(t, "testdata", &logSink{})
	ec := testEvent()
	ec.Notify = func(brief, chapter string) error { return errors.New("link closed") }
	if _, err := host.Call("npc_service", ec); err == nil || !strings.Contains(err.Error(), "link closed") {
		t.Fatalf("expected notify error to surface, got %v", err)
	}
}

func TestBadFileDoesNotBlockOthers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a_bad.lua"), []byte("not lua"), 0644); err != nil {
		t.Fatal(err)
	}
	good := `game.on("late", function(ctx) return true end)` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "b_good.lua"), []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	host := New(dir, nil)
	loaded, err := host.LoadDir()
	if loaded != 1 {
		t.Fatalf("loaded = %d, want 1", loaded)
	}
	if err == nil || !strings.Contains(err.Error(), "a_bad.lua") {
		t.Fatalf("expected a_bad.lua in error, got %v", err)
	}
	if !host.HasHandler("late") {
		t.Fatal("good script should still register handlers")
	}
}

func TestDisabledAndClosedHosts(t *testing.T) {
	disabled := New("", nil)
	if n, err := disabled.LoadDir(); n != 0 || err != nil {
		t.Fatalf("disabled LoadDir = (%d, %v)", n, err)
	}
	if handled, err := disabled.Call("npc_service", testEvent()); handled || err != nil {
		t.Fatalf("disabled Call = (%v, %v)", handled, err)
	}
	host := newTestHost(t, "testdata", &logSink{})
	if err := host.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := host.Eval("return 1"); err == nil {
		t.Fatal("Eval after Close should fail")
	}
	if host.HasHandler("npc_service") {
		t.Fatal("handlers should be cleared after Close")
	}
}

func TestReloadPicksUpNewScript(t *testing.T) {
	dir := t.TempDir()
	host := New(dir, nil)
	if _, err := host.LoadDir(); err != nil {
		t.Fatalf("LoadDir empty dir: %v", err)
	}
	if host.HasHandler("fresh") {
		t.Fatal("unexpected handler")
	}
	if err := os.WriteFile(filepath.Join(dir, "fresh.lua"), []byte("game.on(\"fresh\", function() return true end)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if n, err := host.Reload(); err != nil || n != 1 {
		t.Fatalf("Reload = (%d, %v)", n, err)
	}
	if !host.HasHandler("fresh") {
		t.Fatal("reload should pick up new script")
	}
}
