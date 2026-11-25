package action

// action_test.go 构建器测试：招式表方言 → 判定时刻折算（分段累计、
// action_speed 变速、判定集 hitbreak ∪ hitback 同刻去重、变速复位不当
// 判定），以及产物经服务端装载器回读一致。夹具照抄客户端招式表的真实
// 节（GBK + CRLF）。

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/actiondata"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// writeTree 落一份临时客户端资源根（ini/action）；行尾统一成 CRLF。
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	client := t.TempDir()
	for name, body := range files {
		path := filepath.Join(client, "ini", "action", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		body = strings.ReplaceAll(body, "\n", "\r\n")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return client
}

// tableDir 客户端资源根下的招式表目录。
func tableDir(client string) string { return filepath.Join(client, "ini", "action") }

// gbk 客户端表是 GBK 编码：把夹具里的中文编成 GBK 字节串。
func gbk(t *testing.T, s string) string {
	t.Helper()
	out, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// 单段多判定帧（女号抓包锁定技 zhaoshi_player.ini [CS_jh_cqgf01]：判定帧
// 13/18/27 → 433/600/900ms，实测 452/682/964）。无清单时退回目录内全部
// zhaoshi_*.ini。
func TestHitTimesSingleSegment(t *testing.T) {
	dir := tableDir(writeTree(t, map[string]string{"zhaoshi_player.ini": `
[CS_jh_cqgf01]
1=tzcq_10;0;45;-1;-1;1
trigger_custom_1=1;trigger_effect;18;body_hit_3
trigger_custom_2=1;trigger_hitback;27;0.6,hurt_f_1_l,,0.2
trigger_custom_3=1;trigger_effect;28;head_hit_3
trigger_custom_4=1;trigger_play_sound;27;fight\hurt\hurt_0h_009
trigger_custom_5=1;trigger_hitback;18;0.3,hurt_f_1_s,,
trigger_custom_6=1;trigger_play_sound;18;fight\hurt\hurt_0h_007
trigger_custom_7=1;trigger_hitbreak;13;hurt_f_2_s,
trigger_custom_8=1;trigger_effect;13;body_hit_2
trigger_custom_9=1;trigger_play_sound;13;fight\hurt\hurt_0h_006
trigger_custom_10=1;trigger_play_sound;29;atk_05
trigger_custom_11=1;trigger_effect;1;cqgf01_01
Mark=518ffb5e
`}))
	files, warnings := includeFiles(dir)
	if len(files) != 1 || len(warnings) != 0 {
		t.Fatalf("清单文件 %v、告警 %v", files, warnings)
	}
	entries, triggers := parse(files)
	if len(entries) != 1 || triggers != 3 {
		t.Fatalf("招式 %d 条、判定触发 %d: %+v", len(entries), triggers, entries)
	}
	if got := entries[0].hitTimes(); !reflect.DeepEqual(got, []int{433, 600, 900}) {
		t.Errorf("判定时刻 = %v, want [433 600 900]", got)
	}
	if got := entries[0].length(); got != 1500 {
		t.Errorf("总时长 = %d, want 1500", got)
	}
}

// 变速折算（[CS_jh_cqgf04]）：帧 28 起 ×0.5、32 起 ×2.5、46 复位 1 倍，
// 判定帧 38 → 1280ms（实测 1277）；复位触发不是判定帧。
func TestHitTimesFollowActionSpeed(t *testing.T) {
	dir := tableDir(writeTree(t, map[string]string{"zhaoshi_player.ini": `
[CS_jh_cqgf04]
1=CS_fight_0h_jb_16;0;58;-1;-1;1
trigger_custom_1=1;trigger_play_sound;38;fight\hurt\hurt_0h_009
trigger_custom_2=1;trigger_play_sound;28;fight\attack\att_0H_005
trigger_custom_3=1;trigger_play_sound;10;fight\attack\att_0H_014
trigger_custom_4=1;trigger_play_sound;18;fight\attack\att_0H_012
trigger_custom_5=1;trigger_camera_vibrate;41;hit_heavy
trigger_custom_6=1;trigger_action_speed;32; ,2.5,CS_fight_0h_jb_16
trigger_custom_7=1;trigger_action_speed;46; ,1,CS_fight_0h_jb_16
trigger_custom_8=1;trigger_action_speed;28; ,0.5,CS_fight_0h_jb_16
trigger_custom_9=1;trigger_hitbreak;38;,
trigger_custom_10=1;trigger_play_sound;41;atk_03
trigger_custom_11=1;trigger_effect;39;cqgf04_01
trigger_custom_12=1;trigger_effect;33;cqgf04_02
Mark=cd313b3894c3a0a905b314acdb2b1b99
`}))
	entries, triggers := parse([]string{filepath.Join(dir, "zhaoshi_player.ini")})
	if triggers != 1 {
		t.Fatalf("判定触发应为 1（变速复位不计）: %d", triggers)
	}
	if got := entries[0].hitTimes(); !reflect.DeepEqual(got, []int{1280}) {
		t.Errorf("判定时刻 = %v, want [1280]", got)
	}
	// 28/30 + (32−28)/30/0.5 + (46−32)/30/2.5 + (58−46)/30 = 1787ms。
	if got := entries[0].length(); got != 1787 {
		t.Errorf("变速后总时长 = %d, want 1787", got)
	}
}

// 多段累计（zhaoshi_player_2.ini [zs_normal_00_01]，GBK 表头）：前段时长
// 是后段判定帧的起点；同刻的 hitbreak/hitback 只算一击；越界段号与超出
// 帧长的判定忽略；清单序在前的同名招式胜出；清单项缺席只告警。
func TestHitTimesAcrossSegments(t *testing.T) {
	dir := tableDir(writeTree(t, map[string]string{
		"zhaoshi_list.ini": "[include]\nfile1=ini\\action\\zhaoshi_player_2.ini\n" +
			"file2=ini\\action\\zhaoshi_npc.ini\nfile3=ini\\action\\missing.ini\n",
		"zhaoshi_player_2.ini": `
[zs_normal_00_01]
Name=` + gbk(t, "空手普攻01") + `
1=ptgj_0h_01;0;15;-1;0.2175;1
2=ptgj_0h_05;0;29;0.2175;-1;1
trigger_custom_1=1;trigger_hitbreak;14;hurt_l_1_s
trigger_custom_2=1;trigger_effect;14;head_qj_1
trigger_custom_3=2;trigger_hitbreak;15;hurt_f_2_s
trigger_custom_4=2;trigger_effect;15;abdomen_qj_2
trigger_custom_5=1;trigger_play_sound;14;fight\hurt\hurt_0h_006,
trigger_custom_6=2;trigger_play_sound;15;fight\hurt\hurt_0h_009,
trigger_custom_7=3;trigger_hitbreak;1;,
trigger_custom_8=2;trigger_hitback;40;,
trigger_custom_9=2;trigger_hitback;15;0.3,x,,
Mark=989adead203caa5ac0519b1199f92f17

[dupe]
1=a;0;10;-1;-1;1
`,
		"zhaoshi_npc.ini": `
[dupe]
1=b;0;99;-1;-1;1
trigger_custom_1=1;trigger_hitbreak;5;,
`,
	}))
	files, warnings := includeFiles(dir)
	if len(files) != 2 || len(warnings) != 1 || !strings.Contains(warnings[0], "missing.ini") {
		t.Fatalf("清单文件 %d、告警 %v", len(files), warnings)
	}
	entries, triggers := parse(files)
	if len(entries) != 2 {
		t.Fatalf("招式 %d 条: %+v", len(entries), entries)
	}
	// 判定触发按源表计（5 条，含 3 条越界后被忽略的）。
	if triggers != 5 {
		t.Fatalf("判定触发应为 5: %d", triggers)
	}
	basic, dupe := entries[0], entries[1]
	if basic.id != "zs_normal_00_01" {
		t.Fatalf("招式序不符: %+v", entries)
	}
	if got := basic.hitTimes(); !reflect.DeepEqual(got, []int{467, 1000}) {
		t.Errorf("普攻判定时刻 = %v, want [467 1000]", got)
	}
	if got := basic.frameList(); !reflect.DeepEqual(got, []int{15, 29}) {
		t.Errorf("帧长 = %v, want [15 29]", got)
	}
	if basic.length() != 1467 {
		t.Errorf("总时长 = %d, want 1467", basic.length())
	}
	if dupe.source != "zhaoshi_player_2.ini" || !reflect.DeepEqual(dupe.frameList(), []int{10}) || len(dupe.hitTimes()) != 0 {
		t.Errorf("同名招式应取清单序首见: %+v", dupe)
	}
}

// Run：落盘 + verify（产物经 actiondata 回读一致），报告可核；招式表
// 目录不可用或清单为空时报错而非静默产出。
func TestRunWritesVerifiableProduct(t *testing.T) {
	client := writeTree(t, map[string]string{"zhaoshi_player.ini": `
[k_atk3_1]
1=k_atk3_1;0;20;-1;-1;1
trigger_custom_1=1;trigger_hitbreak;10;,
`})
	out := t.TempDir()
	report := filepath.Join(out, "actiongen-report.json")
	if err := Run(Options{Client: client, Out: out, ReportPath: report, Verify: true}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	catalog, err := actiondata.LoadDir(filepath.Join(out, "ini", "Action"))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := catalog.Lookup("k_atk3_1")
	if !ok || !reflect.DeepEqual(got.HitsMS, []int{333}) || got.Length != 667 {
		t.Fatalf("产物回读 = %+v", got)
	}
	if info, err := os.Stat(report); err != nil || info.Size() == 0 {
		t.Fatalf("审计报告缺席: %v", err)
	}
	empty := t.TempDir()
	if err := os.MkdirAll(filepath.Join(empty, "ini", "action"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"目录不可用": filepath.Join(client, "nope"),
		"清单为空":  empty,
	} {
		if err := Run(Options{Client: bad, Out: out, ReportPath: report}); err == nil {
			t.Errorf("%s：应报错", name)
		}
	}
}
