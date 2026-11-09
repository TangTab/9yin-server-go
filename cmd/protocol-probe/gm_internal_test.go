package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// gmInternalHarness 起一个真实的 internal API mux（带 token 鉴权）
// 和一个“假的在线会话”，与 gm_panel_flow_test.go 的面板 harness 同款，
// 但面向 gm-panel.exe 将要消费的 /internal/v1 面。
func gmInternalHarness(t *testing.T) (*httptest.Server, *gmSession, *gmHub) {
	t.Helper()
	hub := newGMHub()
	session := hub.attach(9, "127.0.0.1:6666", &role.RoleSnapshot{
		RoleSummary: role.RoleSummary{Name: "内功测试侠"},
		Location: role.Location{
			Scene:    role.Scene{Config: `ini\scene\city04_LuoYang`, Resource: "city04"},
			Position: role.Position{X: 243.4, Y: 43.3, Z: 403.8, Orient: 5.0},
		},
	}, "少林", 50, 0, 0, 10, 9_999_999)
	server := httptest.NewServer(newGMInternalMux("secret-token", hub, gmHealthInfo{
		ListenAddr: "127.0.0.1:19099", ServerRoot: `G:\sandbox`,
	}))
	t.Cleanup(server.Close)
	return server, session, hub
}

func gmInternalGet(t *testing.T, url, token string) (string, int) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	return string(raw), response.StatusCode
}

// TestGMInternalAuthGate 鉴权闸：除 health 外一律 Bearer；错误 token
// 与缺失 token 同罪 401；health 无 token 也要 200。
func TestGMInternalAuthGate(t *testing.T) {
	server, _, _ := gmInternalHarness(t)

	if body, status := gmInternalGet(t, server.URL+"/internal/v1/health", ""); status != http.StatusOK || !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("health without token: status=%d body=%s", status, body)
	}
	if _, status := gmInternalGet(t, server.URL+"/internal/v1/sessions", ""); status != http.StatusUnauthorized {
		t.Fatalf("sessions without token: status=%d want 401", status)
	}
	if _, status := gmInternalGet(t, server.URL+"/internal/v1/sessions", "wrong-token"); status != http.StatusUnauthorized {
		t.Fatalf("sessions with wrong token: status=%d want 401", status)
	}
	if body, status := gmInternalGet(t, server.URL+"/internal/v1/sessions", "secret-token"); status != http.StatusOK || !strings.Contains(body, "内功测试侠") {
		t.Fatalf("sessions with token: status=%d body=%s", status, body)
	}
}

// TestGMInternalCommandRoundTrip 命令端到端：与内嵌面板完全相同的
// 校验 → 入队 → 回执语义，只是路径与鉴权不同。
func TestGMInternalCommandRoundTrip(t *testing.T) {
	server, session, _ := gmInternalHarness(t)

	go func() {
		for command := range session.commands {
			if command.Action == "set_silver" {
				session.setSilver(command.Silver)
			}
			command.complete(nil)
		}
	}()

	post := func(payload string) (string, int) {
		request, err := http.NewRequest(http.MethodPost, server.URL+"/internal/v1/sessions/9/command", strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer secret-token")
		request.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		raw, _ := io.ReadAll(response.Body)
		return string(raw), response.StatusCode
	}

	if body, status := post(`{"action":"set_silver","silver":777}`); status != http.StatusOK || !strings.Contains(body, `"sent"`) {
		t.Fatalf("round trip: status=%d body=%s", status, body)
	}
	if _, status := post(`{"action":"learn_all_taolu"}`); status != http.StatusOK {
		t.Fatalf("learn_all_taolu now valid: status=%d want 200", status)
	}
	if body, status := post(`{"action":"no_such_action"}`); status != http.StatusBadRequest || !strings.Contains(body, "unsupported action") {
		t.Fatalf("unsupported action: status=%d body=%s", status, body)
	}
	if body, status := post(`{"action":"set_sp","sp":999}`); status != http.StatusBadRequest || !strings.Contains(body, "SP must be between") {
		t.Fatalf("sp bound: status=%d body=%s", status, body)
	}

	// 离线会话：internal API 侧同样 409（面板透传）。
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/internal/v1/sessions/404/command", strings.NewReader(`{"action":"fill_sp"}`))
	request.Header.Set("Authorization", "Bearer secret-token")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("offline session: status=%d want 409", response.StatusCode)
	}

	// 状态回读走 internal 面。
	if body, status := gmInternalGet(t, server.URL+"/internal/v1/sessions", "secret-token"); status != http.StatusOK || !strings.Contains(body, `"silver":777`) {
		t.Fatalf("state readback: status=%d body=%s", status, body)
	}
}

// TestGMInternalBodyLimit 64KB 载荷闸：lua_eval 合法 16KB 之上的超限
// body 必须被拒，且不进入校验/入队。
func TestGMInternalBodyLimit(t *testing.T) {
	server, session, _ := gmInternalHarness(t)
	consumed := make(chan gmCommand, 1)
	go func() {
		for command := range session.commands {
			consumed <- command
			command.complete(nil)
		}
	}()
	huge := `{"action":"lua_eval","code":"` + strings.Repeat("x", gmCommandBodyLimit) + `"}`
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/internal/v1/sessions/9/command", bytes.NewReader([]byte(huge)))
	request.Header.Set("Authorization", "Bearer secret-token")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("oversized body: status=%d want 400", response.StatusCode)
	}
	select {
	case <-consumed:
		t.Fatal("oversized body reached the session loop")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestGMInternalCatalogEndpoints 目录端点经 token 可达且与内嵌面板
// 同源（fashion/cards/hair-cards/items 全部 200）。
func TestGMInternalCatalogEndpoints(t *testing.T) {
	server, _, _ := gmInternalHarness(t)
	for _, path := range []string{
		"/internal/v1/catalog/fashion",
		"/internal/v1/catalog/cards",
		"/internal/v1/catalog/hair-cards",
		"/internal/v1/catalog/items",
		// gm_catalog_dirs.go 的补充目录/状态端点镜像（gm-panel 同名
		// /api/* 代理到这里）。
		"/internal/v1/catalog/clones",
		"/internal/v1/catalog/scenes",
		"/internal/v1/catalog/giftboxes",
		"/internal/v1/catalog/titles",
		"/internal/v1/catalog/mounts",
		"/internal/v1/catalog/idle-stances",
		"/internal/v1/catalog/recast-affixes",
		"/internal/v1/catalog/combat-dummy/taolus",
		"/internal/v1/catalog/repute/status",
		"/internal/v1/catalog/origin/status",
		"/internal/v1/minimal-login",
	} {
		if _, status := gmInternalGet(t, server.URL+path, "secret-token"); status != http.StatusOK {
			t.Fatalf("%s: status=%d want 200", path, status)
		}
	}
}

// TestGMInternalHealthShape 面板依赖 health 的四个字段做探活与版本
// 比对，钉死形状。
func TestGMInternalHealthShape(t *testing.T) {
	server, _, _ := gmInternalHarness(t)
	body, status := gmInternalGet(t, server.URL+"/internal/v1/health", "")
	if status != http.StatusOK {
		t.Fatalf("health: status=%d", status)
	}
	var health struct {
		Status     string `json:"status"`
		Version    string `json:"version"`
		UptimeSec  int    `json:"uptimeSec"`
		Sessions   int    `json:"sessions"`
		Running    bool   `json:"running"`
		ListenAddr string `json:"listenAddr"`
		ServerRoot string `json:"serverRoot"`
	}
	if err := json.Unmarshal([]byte(body), &health); err != nil {
		t.Fatalf("health not JSON: %v", err)
	}
	if health.Status != "ok" || health.Version == "" || health.Sessions != 1 || health.UptimeSec < 0 {
		t.Fatalf("unexpected health shape: %s", body)
	}
	// Java GmRemote status 同形字段（管理页卡片消费）。
	if !health.Running || health.ListenAddr != "127.0.0.1:19099" || health.ServerRoot != `G:\sandbox` {
		t.Fatalf("manage status fields missing: %s", body)
	}
}
