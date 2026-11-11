package gmapi

import (
	"encoding/json"
	"strings"
	"testing"
)

// 经脉两动作（set_zhenqi_cap/set_jingmai_tier）的契约层域校验——
// 逐条对齐 Java GmRemote.validateCommand（enabled 0/1 + 阈值 1..10⁷；
// jingMaiId 非空 + 档位 1..216）。

func validateBody(t *testing.T, raw string) (Command, error) {
	t.Helper()
	var body CommandBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return Validate(body)
}

func TestValidateZhenQiCap(t *testing.T) {
	command, err := validateBody(t, `{"action":"set_zhenqi_cap","enabled":1,"value":3000}`)
	if err != nil {
		t.Fatalf("valid cap rejected: %v", err)
	}
	if command.Enabled != 1 || command.Value != 3000 {
		t.Fatalf("command=%+v", command)
	}
	for _, raw := range []string{
		`{"action":"set_zhenqi_cap","enabled":2,"value":3000}`,
		`{"action":"set_zhenqi_cap","enabled":-1,"value":3000}`,
		`{"action":"set_zhenqi_cap","enabled":0,"value":0}`,
		`{"action":"set_zhenqi_cap","enabled":0,"value":10000001}`,
	} {
		if _, err := validateBody(t, raw); err == nil {
			t.Fatalf("invalid cap accepted: %s", raw)
		}
	}
}

func TestValidateLootBoxTTL(t *testing.T) {
	command, err := validateBody(t, `{"action":"set_lootbox_ttl","value":300}`)
	if err != nil {
		t.Fatalf("valid ttl rejected: %v", err)
	}
	if command.Value != 300 {
		t.Fatalf("command=%+v", command)
	}
	for _, raw := range []string{
		`{"action":"set_lootbox_ttl","value":-1}`,
		`{"action":"set_lootbox_ttl","value":86401}`,
		`{"action":"set_lootbox_ttl","value":1.5}`,
	} {
		_, err := validateBody(t, raw)
		if err == nil {
			t.Fatalf("invalid ttl accepted: %s", raw)
		}
		if !strings.Contains(err.Error(), "lootbox ttl") {
			t.Fatalf("error %q should name the offending field", err)
		}
	}
}

func TestValidateJingMaiTier(t *testing.T) {
	command, err := validateBody(t, `{"action":"set_jingmai_tier","jingMaiId":"jm_dantian","jingMaiTier":36}`)
	if err != nil {
		t.Fatalf("valid tier rejected: %v", err)
	}
	if command.JingMaiID != "jm_dantian" || command.JingMaiTier != 36 {
		t.Fatalf("command=%+v", command)
	}
	for _, raw := range []string{
		`{"action":"set_jingmai_tier","jingMaiId":"","jingMaiTier":36}`,
		`{"action":"set_jingmai_tier","jingMaiTier":36}`,
		`{"action":"set_jingmai_tier","jingMaiId":"jm_dantian","jingMaiTier":0}`,
		`{"action":"set_jingmai_tier","jingMaiId":"jm_dantian","jingMaiTier":217}`,
	} {
		_, err := validateBody(t, raw)
		if err == nil {
			t.Fatalf("invalid tier accepted: %s", raw)
		}
		if !strings.Contains(err.Error(), "jingMai") && !strings.Contains(err.Error(), "jingmai") {
			t.Fatalf("error %q should name the offending field", err)
		}
	}
}

// NPC 自动追击总开关（set_npc_chase）：enabled 严格 0/1（与
// set_zhenqi_cap 的开关口径一致），游戏侧进程级覆写。缺省 enabled=0
// 视为关闭（与 set_silver 缺省 0 合法同口径；面板恒显式发送）。
func TestValidateNPCChase(t *testing.T) {
	for _, raw := range []string{
		`{"action":"set_npc_chase","enabled":1}`,
		`{"action":"set_npc_chase","enabled":0}`,
		`{"action":"set_npc_chase"}`,
	} {
		command, err := validateBody(t, raw)
		if err != nil {
			t.Fatalf("valid chase rejected: %v (%s)", err, raw)
		}
		if command.Enabled != 0 && command.Enabled != 1 {
			t.Fatalf("command=%+v (%s)", command, raw)
		}
	}
	for _, raw := range []string{
		`{"action":"set_npc_chase","enabled":2}`,
		`{"action":"set_npc_chase","enabled":-1}`,
	} {
		_, err := validateBody(t, raw)
		if err == nil {
			t.Fatalf("invalid chase accepted: %s", raw)
		}
		if !strings.Contains(err.Error(), "npc chase") {
			t.Fatalf("error %q should name the offending field", err)
		}
	}
}

// 九阴志全局门控四动作（ninjiuzhi_set/event/reset/status）的契约层
// 校验：阶段界值 1..9、事件类别枚举 kill|task、无负载动作直放行。
// 「阶段只增不减」的拒绝语义在游戏侧判定，不在本层。
func TestValidateJiuYinZhi(t *testing.T) {
	command, err := validateBody(t, `{"action":"ninjiuzhi_set","stage":7}`)
	if err != nil {
		t.Fatalf("valid set rejected: %v", err)
	}
	if command.Stage != 7 {
		t.Fatalf("command=%+v", command)
	}
	for _, raw := range []string{
		`{"action":"ninjiuzhi_set","stage":0}`,
		`{"action":"ninjiuzhi_set","stage":10}`,
		`{"action":"ninjiuzhi_set"}`,
	} {
		if _, err := validateBody(t, raw); err == nil {
			t.Fatalf("invalid set accepted: %s", raw)
		} else if !strings.Contains(err.Error(), "stage") {
			t.Fatalf("error %q should name stage", err)
		}
	}
	for _, kind := range []string{"kill", "task"} {
		command, err := validateBody(t, `{"action":"ninjiuzhi_event","eventKind":"`+kind+`"}`)
		if err != nil || command.EventKind != kind {
			t.Fatalf("event kind %q rejected: %v", kind, err)
		}
	}
	if _, err := validateBody(t, `{"action":"ninjiuzhi_event","eventKind":"boss"}`); err == nil {
		t.Fatal("unknown event kind accepted")
	}
	for _, action := range []string{"ninjiuzhi_reset", "ninjiuzhi_status"} {
		if _, err := validateBody(t, `{"action":"`+action+`"}`); err != nil {
			t.Fatalf("%s rejected: %v", action, err)
		}
	}
}

// 任务调试四动作（authored_task_*，数据侧写入口）：接取必填任务 ID；
// 推进/清理/交付允许 0（=最早活跃任务），拒绝负数。
func TestValidateAuthoredTask(t *testing.T) {
	command, err := validateBody(t, `{"action":"authored_task_start","taskId":791}`)
	if err != nil {
		t.Fatalf("valid start rejected: %v", err)
	}
	if command.TaskId != 791 {
		t.Fatalf("command=%+v", command)
	}
	for _, action := range []string{
		"authored_task_submit_ready", "authored_task_remove", "authored_task_reward",
	} {
		command, err := validateBody(t, `{"action":"`+action+`","taskId":0}`)
		if err != nil {
			t.Fatalf("valid %s rejected: %v", action, err)
		}
		if command.TaskId != 0 {
			t.Fatalf("%s command=%+v", action, command)
		}
	}
	for _, raw := range []string{
		`{"action":"authored_task_start","taskId":0}`,
		`{"action":"authored_task_start","taskId":-1}`,
		`{"action":"authored_task_start"}`,
		`{"action":"authored_task_submit_ready","taskId":-7}`,
		`{"action":"authored_task_remove","taskId":-1}`,
		`{"action":"authored_task_reward","taskId":-3}`,
	} {
		if _, err := validateBody(t, raw); err == nil {
			t.Fatalf("invalid task command accepted: %s", raw)
		}
	}
}

// mount_ride（坐骑骑乘）：mountConfig 非空闸（Java GmRemote mount_ride
// 同域；面板 mountConfig 键直传）。
func TestValidateMountRide(t *testing.T) {
	command, err := validateBody(t, `{"action":"mount_ride","mountConfig":"ride_jh_whz_0001"}`)
	if err != nil {
		t.Fatalf("validate mount_ride: %v", err)
	}
	if command.MountConfig != "ride_jh_whz_0001" {
		t.Fatalf("MountConfig=%q", command.MountConfig)
	}
	if _, err := validateBody(t, `{"action":"mount_ride","mountConfig":"  "}`); err == nil {
		t.Fatalf("empty mountConfig should fail")
	}
}

// binglu（兵录设置）：bingluArgs 非空闸（形状由游戏侧执行器校验）。
func TestValidateBinglu(t *testing.T) {
	command, err := validateBody(t, `{"action":"binglu","bingluArgs":"set 1 7 120 15"}`)
	if err != nil {
		t.Fatalf("validate binglu: %v", err)
	}
	if command.BingluArgs != "set 1 7 120 15" {
		t.Fatalf("BingluArgs=%q", command.BingluArgs)
	}
	if _, err := validateBody(t, `{"action":"binglu","bingluArgs":""}`); err == nil {
		t.Fatalf("empty bingluArgs should fail")
	}
}

// learn_taolu/learn_neigong（整套发放）：skillKey 非空闸 + skillLevel
// 0..999 界值（0=各自满级；逐技钳上限在游戏侧）。
func TestValidateLearnSkill(t *testing.T) {
	command, err := validateBody(t, `{"action":"learn_taolu","skillKey":"cs_wd_tjq","skillLevel":36}`)
	if err != nil {
		t.Fatalf("validate learn_taolu: %v", err)
	}
	if command.SkillKey != "cs_wd_tjq" || command.SkillLevel != 36 {
		t.Fatalf("SkillKey/SkillLevel=%q/%d", command.SkillKey, command.SkillLevel)
	}
	// 缺省层数=0（各自满级）放行。
	command, err = validateBody(t, `{"action":"learn_neigong","skillKey":"ng_wd_001"}`)
	if err != nil {
		t.Fatalf("validate learn_neigong default level: %v", err)
	}
	if command.SkillLevel != 0 {
		t.Fatalf("default SkillLevel=%d want 0", command.SkillLevel)
	}
	for _, bad := range []string{
		`{"action":"learn_taolu","skillKey":"  "}`,
		`{"action":"learn_neigong"}`,
		`{"action":"learn_taolu","skillKey":"k","skillLevel":-1}`,
		`{"action":"learn_neigong","skillKey":"k","skillLevel":1000}`,
	} {
		if _, err := validateBody(t, bad); err == nil {
			t.Fatalf("invalid body %s should fail", bad)
		}
	}
}

// forget_taolu/forget_neigong（整套移除）：skillKey 非空闸；层数字段
// 无意义但放行（移除语义在游戏侧）。
func TestValidateForgetSkill(t *testing.T) {
	command, err := validateBody(t, `{"action":"forget_taolu","skillKey":"cs_wd_tjq"}`)
	if err != nil {
		t.Fatalf("validate forget_taolu: %v", err)
	}
	if command.SkillKey != "cs_wd_tjq" {
		t.Fatalf("SkillKey=%q", command.SkillKey)
	}
	if _, err := validateBody(t, `{"action":"forget_neigong","skillKey":"ng_wd_001","skillLevel":5}`); err != nil {
		t.Fatalf("forget_neigong with stray level: %v", err)
	}
	for _, bad := range []string{
		`{"action":"forget_taolu","skillKey":""}`,
		`{"action":"forget_neigong"}`,
	} {
		if _, err := validateBody(t, bad); err == nil {
			t.Fatalf("invalid body %s should fail", bad)
		}
	}
}
