package gmpanel

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeWorldEcho 模拟 world echo 维护口方言（welcome / status / players /
// plist / kick），返回动态口。
func fakeWorldEcho(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			go func(nc net.Conn) {
				defer nc.Close()
				fmt.Fprint(nc, "welcome server 11\r")
				r := bufio.NewReader(nc)
				for {
					nc.SetReadDeadline(time.Now().Add(5 * time.Second))
					line, err := r.ReadString('\r')
					if err != nil {
						return
					}
					line = strings.TrimSpace(line)
					switch {
					case line == "status":
						fmt.Fprint(nc, ">game server state is Opened\r")
					case line == "players":
						fmt.Fprint(nc, ">player: 2, online: 1, offline: 0, active: 1, queuing: 0\r")
					case line == "plist":
						fmt.Fprint(nc, ">player 2\rtest001|角色甲|Online|3|1700000000|1700000500\rtest002||Logging|0|1700000100|1700000100\r>end\r")
					case strings.HasPrefix(line, "kick "):
						fmt.Fprint(nc, ">kick " + strings.TrimPrefix(line, "kick ") + " sent\r")
					default:
						fmt.Fprint(nc, ">undefined command: " + line + "\r")
					}
				}
			}(nc)
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port
	return port
}

// newTestPanel 动态口面板（HTTP 0 + world echo 注入假口）。
func newTestPanel(t *testing.T, worldPort int, mutate func(*Config)) *Server {
	t.Helper()
	cfg := Config{
		Port: 0, Mode: "cluster", Members: 1,
		BinDir: t.TempDir(), // Start 失败用（本组测试不拉真进程）
		WorldPort: worldPort,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	return s
}

func get(t *testing.T, url string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body
}

func TestEchoCliSingleAndMultiLine(t *testing.T) {
	port := fakeWorldEcho(t)
	c := NewEchoCli("127.0.0.1", port)
	lines, err := c.Command("status")
	if err != nil || len(lines) != 1 || !strings.Contains(lines[0], "Opened") {
		t.Fatalf("status: %v %q", err, lines)
	}
	lines, err = c.Command("plist")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "player 2") {
		t.Fatalf("plist lines %q", lines)
	}
	if !strings.HasPrefix(lines[1], "test001|角色甲|Online|3|") {
		t.Fatalf("plist row %q", lines[1])
	}
	for _, ln := range lines {
		if ln == "end" || ln == ">end" {
			t.Fatalf(">end leaked: %q", lines) // echocli 收尾语义：>end 不外泄
		}
	}
}

func TestSupervisorSpecs(t *testing.T) {
	sup := NewSupervisor(Config{Mode: "cluster", Members: 3})
	if len(sup.Specs()) != 6+3+2 { // log/world/store/public/lister/entry + member0..2 + auction/monitor
		t.Fatalf("specs %d", len(sup.Specs()))
	}
	if sup.Spec("member2") == nil || sup.Spec("member3") != nil {
		t.Fatal("member 命名")
	}
	if sup.Spec("world").StopEcho != "quit" || sup.Spec("member0").StopEcho != "" {
		t.Fatal("优雅停面")
	}
	single := NewSupervisor(Config{Mode: "single"})
	if len(single.Specs()) != 2 || single.Spec("world").Exe != "game_solounite.exe" {
		t.Fatalf("single specs %+v", single.Specs())
	}
}

func TestSupervisorLifecycle(t *testing.T) {
	sys32 := filepath.Join(os.Getenv("SystemRoot"), "System32")
	if _, err := os.Stat(filepath.Join(sys32, "cmd.exe")); err != nil {
		t.Skip("无 System32（非 Windows 环境）")
	}
	sup := NewSupervisor(Config{Mode: "cluster", BinDir: sys32, LogDir: t.TempDir()})
	// 换一个测试 spec（exe 存在、跑得够久、无优雅停面 → 走 Kill）
	sup.specs = []*svcSpec{{Name: "t1", Exe: "cmd.exe",
		Args: []string{"/c", "ping -n 60 -w 1000 127.0.0.1"}, ProbeHost: "127.0.0.1", ProbePort: 1}}
	if _, err := sup.Start("t1"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	st := SvcState{}
	for time.Now().Before(deadline) {
		st = sup.StatusAll()[0]
		if st.State == "running" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if st.State != "running" || st.PID == 0 {
		t.Fatalf("state %+v", st)
	}
	// 日志文件已建
	if sup.LatestLog("t1") == "" {
		t.Fatal("log file not created")
	}
	if _, err := sup.Stop("t1"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if st = sup.StatusAll()[0]; st.State != "stopped" {
		t.Fatalf("after stop %+v", st)
	}
	// 幂等：再停提示未运行
	if msg, err := sup.Stop("t1"); err != nil || !strings.Contains(msg, "未在运行") {
		t.Fatalf("idempotent stop %q %v", msg, err)
	}
}

func TestAPIOverviewAndPlayers(t *testing.T) {
	world := fakeWorldEcho(t)
	s := newTestPanel(t, world, nil)
	base := "http://" + s.Addr().String()

	code, body := get(t, base+"/api/overview")
	if code != 200 {
		t.Fatalf("overview %d", code)
	}
	services, _ := body["services"].([]any)
	if len(services) != 9 { // members=1
		t.Fatalf("services %d", len(services))
	}
	w := body["world"].(map[string]any)
	if w["alive"] != true || !strings.Contains(w["state"].(string), "Opened") {
		t.Fatalf("world %v", w)
	}
	if body["online"] != float64(1) {
		t.Fatalf("online %v", body["online"])
	}

	code, body = get(t, base+"/api/players")
	if code != 200 {
		t.Fatalf("players %d", code)
	}
	players, _ := body["players"].([]any)
	if len(players) != 2 {
		t.Fatalf("players %d", len(players))
	}
	first := players[0].(map[string]any)
	if first["account"] != "test001" || first["name"] != "角色甲" || first["state"] != "Online" {
		t.Fatalf("player[0] %v", first)
	}
	if first["scene"] != float64(3) || first["loginAt"] != float64(1700000000) {
		t.Fatalf("player[0] fields %v", first)
	}
}

func TestAPILogsTail(t *testing.T) {
	dir := t.TempDir()
	world := fakeWorldEcho(t)
	s := newTestPanel(t, world, func(c *Config) { c.LogDir = dir })
	base := "http://" + s.Addr().String()

	lines := make([]string, 500)
	for i := range lines {
		lines[i] = fmt.Sprintf("line-%03d", i)
	}
	if err := os.WriteFile(filepath.Join(dir, "world-20260101.log"), []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	code, body := get(t, base+"/api/logs?name=world&tail=100")
	if code != 200 {
		t.Fatalf("logs %d", code)
	}
	got, _ := body["lines"].([]any)
	if len(got) != 100 || got[0] != "line-400" || got[99] != "line-499" {
		head := "-"
		if len(got) > 0 {
			head = fmt.Sprint(got[0])
		}
		t.Fatalf("tail %d head=%s file=%v", len(got), head, body["file"])
	}
	// 未知服务 400
	if code, _ = get(t, base+"/api/logs?name=hack"); code != 400 {
		t.Fatalf("unknown svc %d", code)
	}
	// 无日志文件：空数组
	if code, body = get(t, base+"/api/logs?name=store"); code != 200 || body["file"] != "" {
		t.Fatalf("missing log %d %v", code, body["file"])
	}
}

func TestAPISvcUnknown(t *testing.T) {
	world := fakeWorldEcho(t)
	s := newTestPanel(t, world, nil)
	base := "http://" + s.Addr().String()
	resp, err := http.Post(base+"/api/svc/start", "application/json",
		strings.NewReader(`{"name":"hack"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("start unknown %d", resp.StatusCode)
	}
	resp2, err := http.Post(base+"/api/svc/bogus", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("bogus action %d", resp2.StatusCode)
	}
}

func TestAPIIndexPage(t *testing.T) {
	world := fakeWorldEcho(t)
	s := newTestPanel(t, world, nil)
	base := "http://" + s.Addr().String()
	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("index %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type %s", ct)
	}
	// 静态 ax 框架可达
	resp2, err := http.Get(base + "/static/ax.css")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Fatalf("ax.css %d", resp2.StatusCode)
	}
}

func TestParsePlayerCounts(t *testing.T) {
	world := map[string]any{}
	n := parsePlayerCounts("player: 2, online: 1, offline: 0, active: 1, queuing: 0", world)
	if n != 1 || world["counts"] == nil {
		t.Fatalf("n=%d %v", n, world)
	}
}
