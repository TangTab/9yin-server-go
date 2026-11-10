package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	}, "少林", 50, 0, 0, 10, 9_999_999)
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
		session.setSilver(command.Silver)
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
		if command.Action != "set_silver" || command.Silver != 12345 {
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
		{`{"action":"learn_all_taolu"}`, http.StatusOK},                       // 武学三动作：学满（会话循环取走）
		{`{"action":"forget_all_taolu"}`, http.StatusOK},                      // 武学三动作：全遗忘
		{`{"action":"learn_all_neigong"}`, http.StatusOK},                     // 武学三动作：内功全收录
		{`{"action":"no_such_action"}`, http.StatusBadRequest},                // 未知动作
		{`{"action":"set_sp","sp":999}`, http.StatusBadRequest},               // 超过 maxPlayerSP
		{`{"action":"set_rage_recovery","value":5}`, http.StatusOK},           // 怒气自动回复速率（官方满档 5）
		{`{"action":"set_rage_recovery","value":3.5}`, http.StatusBadRequest}, // 必须整数
		{`{"action":"set_rage_recovery","value":-1}`, http.StatusBadRequest},  // 负数拒绝
		{`{"action":"set_silver"}`, http.StatusOK},                            // silver 缺省 0，合法

		// —— 货币/属性/状态批次：合法负载过白名单（会话循环侧会取走）——
		{`{"action":"set_guanyin","guanYin":888}`, http.StatusOK},
		{`{"action":"set_power_level","value":126}`, http.StatusOK},
		{`{"action":"set_alignment","justice":10,"evil":0,"freeBase":5}`, http.StatusOK},
		{`{"action":"set_pk_mode","pkMode":3}`, http.StatusOK},
		{`{"action":"set_vitals","ene":80,"maxEne":100,"resumeEne":1,"sat":60,"maxSat":100,"pkValue":0,"drunkLevel":2}`, http.StatusOK},
		{`{"action":"set_faculty","value":123456}`, http.StatusOK},
		{`{"action":"set_five_attributes","attrStr":100,"attrSta":0,"attrDex":80,"attrIng":0,"attrSpi":90}`, http.StatusOK},
		{`{"action":"set_fill_speed","base":200,"multiplier":1.5}`, http.StatusOK},
		{`{"action":"set_act_day_limit","value":500}`, http.StatusOK},
		{`{"action":"add_livepoint","points":99}`, http.StatusOK},
		{`{"action":"set_hold_faculty","value":1234}`, http.StatusOK},
		{`{"action":"set_live_speed_power","value":50}`, http.StatusOK},
		{`{"action":"set_weather","weather":"heavy_rain_501","fadeSec":5}`, http.StatusOK},
		{`{"action":"set_autosave_interval","value":300}`, http.StatusOK}, // 数据存盘：默认周期
		{`{"action":"set_autosave_interval","value":0}`, http.StatusOK},   // 0=关闭合法
		{`{"action":"set_lootbox_ttl","value":300}`, http.StatusOK},       // 宝箱存留：默认 5 分钟
		{`{"action":"set_lootbox_ttl","value":0}`, http.StatusOK},         // 0=保留到拾空合法
		{`{"action":"set_npc_chase","enabled":0}`, http.StatusOK},         // NPC 自动追击总开关
		{`{"action":"set_npc_chase","enabled":1}`, http.StatusOK},
		{`{"action":"spawn_chest"}`, http.StatusOK}, // GM 世界宝箱（无参白名单）
		{`{"action":"popup_gm_panel"}`, http.StatusOK},
		{`{"action":"custom_probe","msgId":517,"subType":899000000}`, http.StatusOK},

		// —— 「所有道具与换装」：grant_item 合法负载（发放+立即穿戴）——
		{`{"action":"grant_item","itemConfig":"blade_60705","amount":2,"wearNow":true}`, http.StatusOK},
		{`{"action":"grant_item","itemConfig":"box_gift_01","amount":0}`, http.StatusOK}, // 数量缺省回落 1

		// —— 界外/缺值负载：照搬 Java GmRemote 的闸门 ——
		{`{"action":"set_guanyin","guanYin":-1}`, http.StatusBadRequest},
		{`{"action":"set_power_level","value":-3}`, http.StatusBadRequest},
		{`{"action":"set_alignment"}`, http.StatusBadRequest}, // 三值全 0=全跳过
		{`{"action":"set_pk_mode","pkMode":9}`, http.StatusBadRequest},
		{`{"action":"set_vitals","ene":-1}`, http.StatusBadRequest},
		{`{"action":"set_vitals","maxEneAdd":50}`, http.StatusBadRequest}, // 仅 Add 字段（官方表无，忽略后全 0）
		{`{"action":"set_faculty","value":3000000000}`, http.StatusBadRequest},
		{`{"action":"set_five_attributes","attrStr":1000001}`, http.StatusBadRequest},
		{`{"action":"set_five_attributes"}`, http.StatusBadRequest},
		{`{"action":"set_fill_speed","base":0,"multiplier":1}`, http.StatusBadRequest},
		{`{"action":"set_fill_speed","base":200,"multiplier":0.05}`, http.StatusBadRequest},
		{`{"action":"set_act_day_limit","value":1000000}`, http.StatusBadRequest},
		{`{"action":"add_livepoint","points":0}`, http.StatusBadRequest}, // 总历练上限 999 点
		{`{"action":"add_livepoint","points":1000}`, http.StatusBadRequest},
		{`{"action":"set_hold_faculty","value":-1}`, http.StatusBadRequest},
		{`{"action":"set_live_speed_power","value":1001}`, http.StatusBadRequest},
		{`{"action":"set_weather","weather":"..\\..\\evil"}`, http.StatusBadRequest}, // 闭名单拒绝路径注入
		{`{"action":"set_weather","weather":"heavy_rain_501","fadeSec":601}`, http.StatusBadRequest},
		{`{"action":"set_autosave_interval","value":-1}`, http.StatusBadRequest},  // 负数拒绝
		{`{"action":"set_autosave_interval","value":3.5}`, http.StatusBadRequest}, // 必须整数
		{`{"action":"set_autosave_interval","value":86401}`, http.StatusBadRequest},
		{`{"action":"set_lootbox_ttl","value":-1}`, http.StatusBadRequest},  // 负数拒绝
		{`{"action":"set_lootbox_ttl","value":3.5}`, http.StatusBadRequest}, // 必须整数
		{`{"action":"set_lootbox_ttl","value":86401}`, http.StatusBadRequest},
		{`{"action":"set_npc_chase","enabled":2}`, http.StatusBadRequest}, // 开关严格 0/1
		{`{"action":"custom_probe","msgId":0,"subType":1}`, http.StatusBadRequest},
		{`{"action":"custom_probe","msgId":800,"subType":1}`, http.StatusBadRequest},
		{`{"action":"grant_item","itemConfig":"   "}`, http.StatusBadRequest}, // 空白 ConfigID
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

// TestGMFlowAutosaveInterval 数据存盘周期是进程级共享原子值：假会话
// 循环执行与 main.go processGMCommand set_autosave_interval 分支相同的
// 写动作，验证 HTTP → 校验 → 会话通道 → setAutosaveInterval 的完整链路。
func TestGMFlowAutosaveInterval(t *testing.T) {
	previous := autosaveInterval()
	t.Cleanup(func() { setAutosaveInterval(previous) })

	server, session, _ := gmFlowHarness(t)
	// 会话循环消费命令：照抄执行器分支的写动作与回执文案。
	go func() {
		for command := range session.commands {
			if command.Action == "set_autosave_interval" {
				seconds := int32(command.Value)
				setAutosaveInterval(time.Duration(seconds) * time.Second)
				if seconds > 0 {
					session.setAction(fmt.Sprintf("自动存盘间隔设为 %d 秒（在线玩家内功/经脉快照周期落库）", seconds))
				} else {
					session.setAction("自动存盘已关闭：仅断线与 Ctrl+C 关服时存")
				}
			}
			command.complete(nil)
		}
	}()

	payload, _ := json.Marshal(map[string]any{"action": "set_autosave_interval", "value": 120})
	response, err := http.Post(server.URL+"/api/sessions/7/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post command: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("command status=%d", response.StatusCode)
	}
	if got := autosaveInterval(); got != 120*time.Second {
		t.Fatalf("autosave interval=%v want 2m", got)
	}
	// 状态回读带上执行器回执文案（顶栏"最近操作"）。
	body, _, _ := gmGetBody(t, server.URL+"/api/sessions")
	if !strings.Contains(body, "自动存盘间隔设为 120 秒") {
		t.Fatalf("sessions state missing autosave receipt: %s", body)
	}

	// 0=关闭语义：共享值清零后 1s tick 的判定分支不再触发。
	payload, _ = json.Marshal(map[string]any{"action": "set_autosave_interval", "value": 0})
	response, err = http.Post(server.URL+"/api/sessions/7/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post disable: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("disable status=%d", response.StatusCode)
	}
	if got := autosaveInterval(); got != 0 {
		t.Fatalf("autosave interval after disable=%v want 0", got)
	}
}

// TestGMFlowLootBoxTTL 宝箱存留时间是进程级共享原子值：假会话循环执行与
// main.go processGMCommand set_lootbox_ttl 分支相同的写动作，验证 HTTP →
// 校验 → 会话通道 → setLootBoxTTL 的完整链路。
func TestGMFlowLootBoxTTL(t *testing.T) {
	previous := lootBoxTTL()
	t.Cleanup(func() { setLootBoxTTL(previous) })

	server, session, _ := gmFlowHarness(t)
	// 会话循环消费命令：照抄执行器分支的写动作与回执文案。
	go func() {
		for command := range session.commands {
			if command.Action == "set_lootbox_ttl" {
				seconds := int32(command.Value)
				setLootBoxTTL(time.Duration(seconds) * time.Second)
				if seconds > 0 {
					session.setAction(fmt.Sprintf("宝箱存留时间设为 %d 秒（未拾超时消失，对在场宝箱即刻生效）", seconds))
				} else {
					session.setAction("宝箱存留已关闭：宝箱保留到被拾空")
				}
			}
			command.complete(nil)
		}
	}()

	payload, _ := json.Marshal(map[string]any{"action": "set_lootbox_ttl", "value": 300})
	response, err := http.Post(server.URL+"/api/sessions/7/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post command: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("command status=%d", response.StatusCode)
	}
	if got := lootBoxTTL(); got != 300*time.Second {
		t.Fatalf("lootbox ttl=%v want 5m", got)
	}
	// 状态回读带上执行器回执文案（顶栏"最近操作"）。
	body, _, _ := gmGetBody(t, server.URL+"/api/sessions")
	if !strings.Contains(body, "宝箱存留时间设为 300 秒") {
		t.Fatalf("sessions state missing lootbox receipt: %s", body)
	}

	// 0=关闭语义：过期结算分支对在场宝箱不再触发。
	payload, _ = json.Marshal(map[string]any{"action": "set_lootbox_ttl", "value": 0})
	response, err = http.Post(server.URL+"/api/sessions/7/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post disable: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("disable status=%d", response.StatusCode)
	}
	if got := lootBoxTTL(); got != 0 {
		t.Fatalf("lootbox ttl after disable=%v want 0", got)
	}
}

// TestGMFlowNPCChase NPC 自动追击总开关是进程级共享原子值（初值=启动
// env NINEYIN_NPC_CHASE）：假会话循环执行与 main.go processGMCommand
// set_npc_chase 分支相同的写动作，验证 HTTP → 校验 → 会话通道 →
// setNPCChaseEnabled → npcChaseEnabled 的完整链路。
func TestGMFlowNPCChase(t *testing.T) {
	previous := npcChaseEnabled()
	t.Cleanup(func() { setNPCChaseEnabled(previous) })

	server, session, _ := gmFlowHarness(t)
	// 会话循环消费命令：照抄执行器分支的写动作与回执文案。
	go func() {
		for command := range session.commands {
			if command.Action == "set_npc_chase" {
				enabled := setNPCChaseEnabled(command.Enabled == 1)
				if enabled {
					session.setAction("NPC 自动追击已开启：怪沿路点图寻路折线逼近（无图场景直线贴地）")
				} else {
					session.setAction("NPC 自动追击已关闭：仇恨照常持有，怪停在出生点（冲锋测试方言）")
				}
			}
			command.complete(nil)
		}
	}()

	payload, _ := json.Marshal(map[string]any{"action": "set_npc_chase", "enabled": 0})
	response, err := http.Post(server.URL+"/api/sessions/7/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post disable: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("disable status=%d", response.StatusCode)
	}
	if npcChaseEnabled() {
		t.Fatal("npc chase must be disabled after GM command")
	}
	// 状态回读带上执行器回执文案（顶栏"最近操作"）。
	body, _, _ := gmGetBody(t, server.URL+"/api/sessions")
	if !strings.Contains(body, "NPC 自动追击已关闭") {
		t.Fatalf("sessions state missing chase receipt: %s", body)
	}

	// 重开语义：追击 tick 恢复位移分支。
	payload, _ = json.Marshal(map[string]any{"action": "set_npc_chase", "enabled": 1})
	response, err = http.Post(server.URL+"/api/sessions/7/command", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post enable: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("enable status=%d", response.StatusCode)
	}
	if !npcChaseEnabled() {
		t.Fatal("npc chase must be re-enabled after GM command")
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
