package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeGameBackend 起一个最小 internal API 假后端：记录请求次数与
// 鉴权头，按预设剧本回状态码。驱动的是面板真实代理路径。
type fakeGameBackend struct {
	server   *httptest.Server
	requests atomic.Int64
	authSeen sync.Map // path -> Authorization header

	sessionsStatus int
	sessionsBody   string
	commandStatus  int
	commandBody    string

	catalogETag string // catalog 端点回给面板的 ETag（空=不带）；INM 命中回 304
}

func newFakeGameBackend(t *testing.T) *fakeGameBackend {
	t.Helper()
	backend := &fakeGameBackend{sessionsStatus: http.StatusOK, sessionsBody: `[]`,
		commandStatus: http.StatusOK, commandBody: `{"status":"sent"}`}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/v1/health", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprint(writer, `{"status":"ok","version":"test","uptimeSec":1,"sessions":0}`)
	})
	mux.HandleFunc("/internal/v1/sessions", func(writer http.ResponseWriter, request *http.Request) {
		backend.authSeen.Store("sessions", request.Header.Get("Authorization"))
		backend.requests.Add(1)
		writeRaw(writer, backend.sessionsStatus, backend.sessionsBody)
	})
	mux.HandleFunc("/internal/v1/sessions/", func(writer http.ResponseWriter, request *http.Request) {
		backend.authSeen.Store("command", request.Header.Get("Authorization"))
		backend.requests.Add(1)
		writeRaw(writer, backend.commandStatus, backend.commandBody)
	})
	mux.HandleFunc("/internal/v1/catalog/", func(writer http.ResponseWriter, request *http.Request) {
		backend.authSeen.Store("catalog:"+request.URL.Path, request.Header.Get("Authorization")+"?"+request.URL.RawQuery)
		backend.requests.Add(1)
		if backend.catalogETag != "" {
			writer.Header().Set("ETag", backend.catalogETag)
			if request.Header.Get("If-None-Match") == backend.catalogETag {
				writer.WriteHeader(http.StatusNotModified)
				return
			}
		}
		writeRaw(writer, http.StatusOK, `[{"configId":"x","name":"测试"}]`)
	})
	mux.HandleFunc("/internal/v1/minimal-login", func(writer http.ResponseWriter, request *http.Request) {
		backend.authSeen.Store("minimal-login:"+request.Method, request.Header.Get("Authorization"))
		backend.requests.Add(1)
		writeRaw(writer, http.StatusOK, `{"enabled":true}`)
	})
	backend.server = httptest.NewServer(mux)
	t.Cleanup(backend.server.Close)
	return backend
}

func writeRaw(writer http.ResponseWriter, status int, body string) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_, _ = io.WriteString(writer, body)
}

func newPanel(t *testing.T, backend *fakeGameBackend) *httptest.Server {
	t.Helper()
	panel := &panelServer{gameBase: backend.server.URL, gameAddr: strings.TrimPrefix(backend.server.URL, "http://"),
		token: "tok", client: &http.Client{Timeout: commandTimeout}, catCache: make(map[string]catalogCacheEntry)}
	server := httptest.NewServer(panel.mux())
	t.Cleanup(server.Close)
	return server
}

// TestPanelSessionsProxyAndCache 会话列表透传 + 500ms 微缓存：
// TTL 内的第二个请求不应再打后端。
func TestPanelSessionsProxyAndCache(t *testing.T) {
	backend := newFakeGameBackend(t)
	backend.sessionsBody = `[{"sessionId":1,"name":"测试侠"}]`
	panel := newPanel(t, backend)

	first, status := panelGet(t, panel.URL+"/api/sessions")
	if status != http.StatusOK || !strings.Contains(first, "测试侠") {
		t.Fatalf("first: status=%d body=%s", status, first)
	}
	if _, status := panelGet(t, panel.URL+"/api/sessions"); status != http.StatusOK {
		t.Fatalf("second: status=%d", status)
	}
	if got := backend.requests.Load(); got != 1 {
		t.Fatalf("backend hits=%d want 1 (micro-cache)", got)
	}
	if auth, _ := backend.authSeen.Load("sessions"); auth != "Bearer tok" {
		t.Fatalf("authorization header=%v", auth)
	}
	time.Sleep(sessionsCacheTTL + 50*time.Millisecond)
	if _, status := panelGet(t, panel.URL+"/api/sessions"); status != http.StatusOK {
		t.Fatalf("after TTL: status=%d", status)
	}
	if got := backend.requests.Load(); got != 2 {
		t.Fatalf("backend hits after TTL=%d want 2", got)
	}
}

// TestPanelBackendDown503 后端不在（游戏服重启窗口）：503 + 固定错误体，
// 且该 503 也进微缓存，不叠加打连接。
func TestPanelBackendDown503(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)
	backend.server.Close() // 模拟游戏服下线

	body, status := panelGet(t, panel.URL+"/api/sessions")
	if status != http.StatusServiceUnavailable || !strings.Contains(body, "game server unreachable") {
		t.Fatalf("sessions: status=%d body=%s", status, body)
	}
	body, status = panelPost(t, panel.URL+"/api/sessions/1/command", `{"action":"fill_sp"}`)
	if status != http.StatusServiceUnavailable || !strings.Contains(body, "game server unreachable") {
		t.Fatalf("command: status=%d body=%s", status, body)
	}
	body, status = panelGet(t, panel.URL+"/api/cards")
	if status != http.StatusServiceUnavailable || !strings.Contains(body, "game server unreachable") {
		t.Fatalf("catalog: status=%d body=%s", status, body)
	}
}

// TestPanelCommandPreValidation 非法命令在面板本地被拒（快速 400），
// 根本不产生后端往返；合法命令原样转发并逐字透传回执。
func TestPanelCommandPreValidation(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)

	if body, status := panelPost(t, panel.URL+"/api/sessions/7/command", `{"action":"no_such_action"}`); status != http.StatusBadRequest {
		t.Fatalf("unsupported action should 400 locally: status=%d body=%s", status, body)
	}
	if body, status := panelPost(t, panel.URL+"/api/sessions/7/command", `{"action":"set_sp","sp":9999}`); status != http.StatusBadRequest || !strings.Contains(body, "SP must be between") {
		t.Fatalf("sp bound should 400 locally: status=%d body=%s", status, body)
	}
	if got := backend.requests.Load(); got != 0 {
		t.Fatalf("backend hits=%d want 0 (pre-validation)", got)
	}

	// 合法命令：透传后端回执。
	if body, status := panelPost(t, panel.URL+"/api/sessions/7/command", `{"action":"set_silver","silver":5}`); status != http.StatusOK || !strings.Contains(body, "sent") {
		t.Fatalf("valid command: status=%d body=%s", status, body)
	}

	// 后端 409（离线会话）与 504（帧未确认）逐字透传。
	backend.commandStatus, backend.commandBody = http.StatusConflict, `{"error":"session 7 is not online"}`
	if body, status := panelPost(t, panel.URL+"/api/sessions/7/command", `{"action":"fill_sp"}`); status != http.StatusConflict || !strings.Contains(body, "not online") {
		t.Fatalf("409 passthrough: status=%d body=%s", status, body)
	}
	backend.commandStatus, backend.commandBody = http.StatusGatewayTimeout, `{"error":"game session did not confirm the frame send"}`
	if body, status := panelPost(t, panel.URL+"/api/sessions/7/command", `{"action":"fill_sp"}`); status != http.StatusGatewayTimeout || !strings.Contains(body, "did not confirm") {
		t.Fatalf("504 passthrough: status=%d body=%s", status, body)
	}
}

// TestPanelCatalogProxyQueryPassthrough 目录代理带上查询串与鉴权头
//（/api/items 的 q/view 参数依赖这一点）。
func TestPanelCatalogProxyQueryPassthrough(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)

	if body, status := panelGet(t, panel.URL+"/api/items?q=铁剑&view=1"); status != http.StatusOK || !strings.Contains(body, "测试") {
		t.Fatalf("items: status=%d body=%s", status, body)
	}
	if seen, _ := backend.authSeen.Load("catalog:/internal/v1/catalog/items"); seen != "Bearer tok?q=铁剑&view=1" {
		t.Fatalf("catalog forward=%v", seen)
	}
}

// TestPanelWeatherLocal 天气目录面板本地直出（契约层静态闭名单），
// 后端零请求。
func TestPanelWeatherLocal(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)

	body, status := panelGet(t, panel.URL+"/api/weather")
	if status != http.StatusOK || !strings.Contains(body, "小雨（昼）") {
		t.Fatalf("weather: status=%d body=%s", status, body)
	}
	if got := backend.requests.Load(); got != 0 {
		t.Fatalf("backend hits=%d want 0 (weather is local)", got)
	}
}

// TestPanelPagesServed 页面与静态资源由面板自身直出（gmweb 共享资产）。
func TestPanelPagesServed(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)

	for path, marker := range map[string]string{
		"/":                          "<html",
		"/console.html":              "<html",
		"/static/console-core.js":    "refresh",
		"/static/console-features.js": "command",
		"/static/console.css":        "body",
	} {
		if body, status := panelGet(t, panel.URL+path); status != http.StatusOK || !strings.Contains(body, marker) {
			t.Fatalf("%s: status=%d marker=%q", path, status, marker)
		}
	}
}

func panelGet(t *testing.T, url string) (string, int) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	return string(raw), response.StatusCode
}

func panelPost(t *testing.T, url, payload string) (string, int) {
	t.Helper()
	response, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	return string(raw), response.StatusCode
}

// TestPanelManageStatusShape 探活代理：把免鉴权 health 映射回 Java 网关
// /api/manage/status 的 {online, status:{connected_clients, uptime_sec,
// phase, listen_addr, server_root}} 形状；离线目标回 {online:false}；
// 非法 s 逐字对齐 Java 400 文案。
func TestPanelManageStatusShape(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)

	body, status := panelGet(t, panel.URL+"/api/manage/status?s="+strings.TrimPrefix(backend.server.URL, "http://"))
	if status != http.StatusOK {
		t.Fatalf("status=%d body=%s", status, body)
	}
	var probe struct {
		Online bool `json:"online"`
		Status struct {
			Running         bool   `json:"running"`
			ListenAddr      string `json:"listen_addr"`
			ServerRoot      string `json:"server_root"`
			ConnectedClient int    `json:"connected_clients"`
			UptimeSec       int    `json:"uptime_sec"`
			Phase           string `json:"phase"`
		} `json:"status"`
	}
	if err := json.Unmarshal([]byte(body), &probe); err != nil {
		t.Fatalf("not JSON: %v (%s)", err, body)
	}
	if !probe.Online || !probe.Status.Running || probe.Status.ConnectedClient != 0 ||
		probe.Status.UptimeSec != 1 || probe.Status.Phase != "版本 test" {
		t.Fatalf("unexpected probe shape: %s", body)
	}
}

func TestPanelManageStatusOfflineAndInvalid(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)

	// 未监听端口：{online:false} 而不是 5xx。
	if body, status := panelGet(t, panel.URL+"/api/manage/status?s=127.0.0.1:1"); status != http.StatusOK ||
		!strings.Contains(body, `"online":false`) {
		t.Fatalf("offline: status=%d body=%s", status, body)
	}
	// 非法 s：逐字对齐 Java 网关文案。
	for _, bad := range []string{"", "127.0.0.1", "http://127.0.0.1:19063", "127.0.0.1:19063/x",
		"evil.example.com:19063", "127.0.0.1:0", "127.0.0.1:70000"} {
		if _, status := panelGet(t, panel.URL+"/api/manage/status?s="+url.QueryEscape(bad)); status != http.StatusBadRequest {
			t.Fatalf("s=%q: status=%d want 400", bad, status)
		}
	}
}

// TestPanelInfo 登记册据此区分“本面板所连”与“仅监控”卡片。
func TestPanelInfo(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)
	body, status := panelGet(t, panel.URL+"/api/panel-info")
	if status != http.StatusOK || !strings.Contains(body, `"gameAddr"`) || !strings.Contains(body, `"panelVersion"`) {
		t.Fatalf("panel-info: status=%d body=%s", status, body)
	}
}

// TestPanelSupplementCatalogDirs 补充目录/状态端点（console 页启动即
// fetch 的 11 个 /api/*）代理到 internal API 同名路径，minimal-login
// 的 POST 经 methodProxy 透传方法与鉴权头。
func TestPanelSupplementCatalogDirs(t *testing.T) {
	backend := newFakeGameBackend(t)
	panel := newPanel(t, backend)

	for _, path := range []string{
		"/api/clones", "/api/scenes", "/api/giftboxes", "/api/titles",
		"/api/mounts", "/api/idle-stances", "/api/recast-affixes",
		"/api/combat-dummy/taolus", "/api/repute/status", "/api/origin/status",
	} {
		body, status := panelGet(t, panel.URL+path)
		if status != http.StatusOK || !strings.Contains(body, "测试") {
			t.Fatalf("%s: status=%d body=%s", path, status, body)
		}
		internal := strings.TrimPrefix(path, "/api")
		if seen, _ := backend.authSeen.Load("catalog:/internal/v1/catalog" + internal); seen != "Bearer tok?" {
			t.Fatalf("%s forward=%v", path, seen)
		}
	}

	// minimal-login：GET/POST 都经 methodProxy 透传。
	body, status := panelGet(t, panel.URL+"/api/minimal-login")
	if status != http.StatusOK || !strings.Contains(body, `"enabled":true`) {
		t.Fatalf("minimal-login GET: status=%d body=%s", status, body)
	}
	if body, status := panelPost(t, panel.URL+"/api/minimal-login", ""); status != http.StatusOK || !strings.Contains(body, `"enabled":true`) {
		t.Fatalf("minimal-login POST: status=%d body=%s", status, body)
	}
	if seen, _ := backend.authSeen.Load("minimal-login:POST"); seen != "Bearer tok" {
		t.Fatalf("minimal-login POST forward=%v", seen)
	}

	// GET-only 目录端点在面板本地拒绝 POST（不打后端）。
	before := backend.requests.Load()
	if body, status := panelPost(t, panel.URL+"/api/clones", ""); status != http.StatusMethodNotAllowed {
		t.Fatalf("post clones: status=%d body=%s", status, body)
	}
	if got := backend.requests.Load(); got != before {
		t.Fatalf("post clones hit backend (%d -> %d)", before, got)
	}
}
