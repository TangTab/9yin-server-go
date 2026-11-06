package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// gmFlowHarness 挂一个"假的在线会话"到真实的 gmHub + GM 路由上，
// 在不需要游戏客户端 / MySQL / 客户端资源树的前提下，动态验证
// 浏览器 → HTTP → gmHub → 会话命令通道 → 状态回读 的完整往返。
//
// 它模拟的是 main.go 里 handle() 的两件事：
//  1. gm.attach(...) 在角色进场时登记会话；
//  2. 会话循环从 gmSession.commands 取出命令、执行、complete 回执。
func gmFlowHarness(t *testing.T) (*httptest.Server, *gmSession, *gmHub) {
	t.Helper()
	hub := newGMHub()
	session := hub.attach(7, "127.0.0.1:5555", &role.RoleSnapshot{
		RoleSummary: role.RoleSummary{Name: "测试侠"},
		Location: role.Location{
			Scene:    role.Scene{Config: `ini\scene\city04_LuoYang`, Resource: "city04"},
			Position: role.Position{X: 243.4, Y: 43.3, Z: 403.8, Orient: 5.0},
		},
	}, "少林", 50, 10)
	server := httptest.NewServer(newGMMux(hub))
	t.Cleanup(server.Close)
	return server, session, hub
}

// TestGMFlowCommandRoundTrip 驱动一次真实的 set_silver 命令往返：
// 概览页与控制台读取的 /api/sessions 必须先看到假会话，POST 命令后
// 必须被会话通道原样取走，回执 sent，且状态回读反映会话循环的写入。
func TestGMFlowCommandRoundTrip(t *testing.T) {
	server, session, _ := gmFlowHarness(t)

	// 第 1 步：面板读取在线列表（概览页与控制台 refresh() 都走这里）。
	body, _, status := gmGetBody(t, server.URL+"/api/sessions")
	if status != http.StatusOK {
		t.Fatalf("sessions status=%d", status)
	}
	var states []gmPlayerState
	if err := json.Unmarshal([]byte(body), &states); err != nil {
		t.Fatalf("sessions not JSON: %v", err)
	}
	if len(states) != 1 || states[0].SessionID != 7 || states[0].Name != "测试侠" || states[0].Silver != 50 {
		t.Fatalf("unexpected sessions state: %+v", states)
	}

	// 第 2 步：会话循环消费命令（对应 main.go 的 processGMCommand）。
	gotCommand := make(chan gmCommand, 1)
	go func() {
		command := <-session.commands
		gotCommand <- command
		command.complete(nil)
		// 会话循环执行成功后回写面板状态（main.go 中的 setSilver）。
		session.setSilver(command.silver)
	}()

	// 第 3 步：控制台 command() 发出的同款 POST。
	payload, _ := json.Marshal(map[string]any{"action": "set_silver", "silver": 12345})
	response, err := http.Post(server.URL+"/api/sessions/7/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post command: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("command status=%d", response.StatusCode)
	}
	var receipt map[string]string
	_ = json.NewDecoder(response.Body).Decode(&receipt)
	if receipt["status"] != "sent" {
		t.Fatalf("receipt=%v want sent", receipt)
	}

	// 第 4 步：会话通道必须收到浏览器提交的原值。
	select {
	case command := <-gotCommand:
		if command.action != "set_silver" || command.silver != 12345 {
			t.Fatalf("session received %+v", command)
		}
	case <-time.After(time.Second):
		t.Fatal("session loop never received the command")
	}

	// 第 5 步：状态回读反映会话循环的写入（顶栏碎银/最近操作）。
	body, _, _ = gmGetBody(t, server.URL+"/api/sessions")
	if !strings.Contains(body, `"silver":12345`) || !strings.Contains(body, "碎银设为 12345") {
		t.Fatalf("sessions state not updated by session loop: %s", body)
	}
}

// TestGMFlowRejectsBadCommands 验证动态错误路径与浏览器提示一致：
// 未知动作、非法数值、离线会话分别返回 400/400/409。
func TestGMFlowRejectsBadCommands(t *testing.T) {
	server, session, _ := gmFlowHarness(t)
	post := func(payload string) (string, int) {
		response, err := http.Post(server.URL+"/api/sessions/7/command", "application/json", strings.NewReader(payload))
		if err != nil {
			t.Fatalf("post %s: %v", payload, err)
		}
		defer response.Body.Close()
		raw, _ := io.ReadAll(response.Body)
		return string(raw), response.StatusCode
	}
	// 合法命令需要会话循环消费，否则 750ms 后 enqueue 超时返回 409。
	go func() {
		for command := range session.commands {
			command.complete(nil)
		}
	}()
	cases := []struct {
		payload string
		status  int
	}{
		{`{"action":"learn_all_taolu"}`, http.StatusBadRequest}, // 网关时代的动作，当前服已不支持
		{`{"action":"set_sp","sp":999}`, http.StatusBadRequest},  // 超过 maxPlayerSP
		{`{"action":"set_silver"}`, http.StatusOK},                // silver 缺省 0，合法
	}
	for _, testCase := range cases {
		body, status := post(testCase.payload)
		if status != testCase.status {
			t.Fatalf("POST %s status=%d want=%d body=%s", testCase.payload, status, testCase.status, body)
		}
	}
	// 不在线的会话：命令无法送达，返回 409（控制台 alert 展示 error 字段）。
	offline, status := postOffline(t, server, 999, `{"action":"fill_sp"}`)
	if status != http.StatusConflict || !strings.Contains(offline, "not online") {
		t.Fatalf("offline session status=%d body=%s", status, offline)
	}
}

func postOffline(t *testing.T, server *httptest.Server, id uint64, payload string) (string, int) {
	t.Helper()
	response, err := http.Post(
		server.URL+"/api/sessions/"+strconv.FormatUint(id, 10)+"/command",
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("post offline: %v", err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	return string(raw), response.StatusCode
}
