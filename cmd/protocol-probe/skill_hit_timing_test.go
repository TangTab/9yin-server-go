package main

import (
	"testing"
	"time"
)

// TestCompileSkillActionsHitBreakTiming 锚定 2026-11-11 官服 pcapng 定版的
// 多段判定时序（技能.pcapng + 挑战.pcapng 12 技互证）：
//
//  1. hitbreak 帧优先于 trigger_effect（wuji_CS_jz_lyfh07 无任何
//     trigger_effect，官服仍按 hitbreak 15/45/75/105/135 → 500..4500ms 五段）；
//  2. 帧是段内相对帧，段 N 时刻 = 前序段全时长累加 + 帧号/30fps
//     （CS_jz_lyfh02 seg1@14/seg2@12 → 467/1067ms；官服实测 551/1141，
//     差值=客户端→官服 RTT≈80ms，服务端不补偿）；
//  3. inBlend 只折减动画混融（followup 播放），不折减判定时刻；
//  4. 无 hitbreak 的技能回落 trigger_effect _hurt/核心前缀旧口径（shl08）。
func TestCompileSkillActionsHitBreakTiming(t *testing.T) {
	ms := func(sec float64) time.Duration {
		return time.Duration(sec * float64(time.Second))
	}
	approx := func(name string, got []time.Duration, want []float64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: got %v ticks, want %d", name, got, len(want))
		}
		for i, at := range got {
			if diff := at - ms(want[i]); diff < -2*time.Millisecond || diff > 2*time.Millisecond {
				t.Fatalf("%s tick#%d: got %v, want %.0fms", name, i+1, at, want[i]*1000)
			}
		}
	}

	// CS_jz_lyfh02：两段各带 hitbreak，seg2 inBlend 0.24 不得折减判定。
	lyfh02 := []iniField{
		{key: "1", value: "jb_2h_07;0;20;-1;0.24;1"},
		{key: "2", value: "jb_2h_08;0;28;0.24;-1;1"},
		{key: "trigger_custom_1", value: "1;trigger_hitbreak;14;hurt_f_1_s,"},
		{key: "trigger_custom_5", value: "2;trigger_hitbreak;12;hurt_f_1_l,"},
		{key: "trigger_custom_2", value: "1;trigger_effect;14;body_slash_3"},
		{key: "trigger_custom_6", value: "2;trigger_effect;12;body_slash_3"},
	}
	_, duration, followups, hitFrameMs := compileSkillActions(lyfh02, "CS_jz_lyfh02")
	approx("lyfh02", hitFrameMs, []float64{0.467, 1.067})
	if duration != ms(20.0/30) {
		t.Fatalf("lyfh02 duration: got %v", duration)
	}
	// followup 播放时刻仍用 inBlend 折减（0.667-0.24=0.427s）。
	if len(followups) != 1 || followups[0].at != ms(20.0/30-0.24) {
		t.Fatalf("lyfh02 followup: %+v", followups)
	}

	// wuji_CS_jz_lyfh07：单段 143 帧，五连 hitbreak（30 帧一击）。
	lyfh07 := []iniField{
		{key: "1", value: "att_Jb_2H_11_2;0;143;-1;-1;1"},
		{key: "trigger_custom_28", value: "1;trigger_hitbreak;15; "},
		{key: "trigger_custom_29", value: "1;trigger_hitbreak;45; "},
		{key: "trigger_custom_30", value: "1;trigger_hitbreak;75; "},
		{key: "trigger_custom_31", value: "1;trigger_hitbreak;105; "},
		{key: "trigger_custom_32", value: "1;trigger_hitbreak;135; "},
	}
	_, _, _, hitFrameMs = compileSkillActions(lyfh07, "wuji_CS_jz_lyfh07")
	approx("lyfh07", hitFrameMs, []float64{0.5, 1.5, 2.5, 3.5, 4.5})

	// CS_xy_shl08：无 hitbreak → 核心前缀 trigger_effect 回退（帧即全局）。
	shl08 := []iniField{
		{key: "1", value: "shl_01;0;137;-1;-1;1"},
		{key: "trigger_custom_1", value: "1;trigger_effect;17;shl08_01,"},
		{key: "trigger_custom_2", value: "1;trigger_effect;52;shl08_02,"},
		{key: "trigger_custom_3", value: "1;trigger_effect;65;shl08_03,"},
	}
	_, _, _, hitFrameMs = compileSkillActions(shl08, "CS_xy_shl08")
	approx("shl08", hitFrameMs, []float64{17.0 / 30, 52.0 / 30, 65.0 / 30})

	// 段号失配（引用未定义段 9）：按全局帧兜底，不崩、不丢段。
	orphan := []iniField{
		{key: "1", value: "a;0;30;-1;-1;1"},
		{key: "trigger_custom_1", value: "9;trigger_hitbreak;10;,"},
	}
	_, _, _, hitFrameMs = compileSkillActions(orphan, "CS_x_y")
	approx("orphan-seg", hitFrameMs, []float64{10.0 / 30})

	// CS_wd_tjj07 太极剑舞（挑战.pcapng 2026-11-12 定版）：判定帧 =
	// hitback×6 + hitbreak×2 = 官服 8 跳。段链 41/49/56/38/56 帧全时长
	// 累加，hitbreak-only 只会 2 跳——这是 hitback 合并的回归锚点。
	tjj07 := []iniField{
		{key: "1", value: "wdj_01;0;41;-1;0.765;1"},
		{key: "2", value: "wdj_03;0;49;0.765;0.66;1"},
		{key: "3", value: "wdj_06;0;56;0.66;0.87;1"},
		{key: "4", value: "wdj_08;0;38;0.87;0.48;1"},
		{key: "5", value: "wdj_09;0;56;0.48;-1;1"},
		{key: "trigger_custom_1", value: "1;trigger_hitback;24;1.3,hurt_f_1_s"},
		{key: "trigger_custom_6", value: "2;trigger_hitback;28;0.8,hurt_f_2_l"},
		{key: "trigger_custom_13", value: "3;trigger_hitback;50;0.6,hurt_f_1_l,,"},
		{key: "trigger_custom_14", value: "4;trigger_hitback;11;3,hurt_f_2_l"},
		{key: "trigger_custom_27", value: "5;trigger_hitback;12;0.3,hurt_f_1_s,,"},
		{key: "trigger_custom_28", value: "5;trigger_hitback;35;0.5,hurt_f_2_l,,"},
		{key: "trigger_custom_40", value: "3;trigger_hitbreak;14;hurt_f_1_s,"},
		{key: "trigger_custom_45", value: "4;trigger_hitbreak;33;hurt_f_1_s,"},
	}
	_, _, _, hitFrameMs = compileSkillActions(tjj07, "CS_wd_tjj07")
	approx("tjj07", hitFrameMs, []float64{
		24.0 / 30,               // seg1 hitback
		(41 + 28.0) / 30,        // seg2 hitback
		(90 + 14.0) / 30,        // seg3 hitbreak
		(90 + 50.0) / 30,        // seg3 hitback
		(146 + 11.0) / 30,       // seg4 hitback
		(146 + 33.0) / 30,       // seg4 hitbreak
		(184 + 12.0) / 30,       // seg5 hitback
		(184 + 35.0) / 30,       // seg5 hitback
	})

	// hitbreak 与 hitback 同刻（同段同帧）：合并为一跳。
	sameFrame := []iniField{
		{key: "1", value: "a;0;30;-1;-1;1"},
		{key: "trigger_custom_1", value: "1;trigger_hitbreak;10;,"},
		{key: "trigger_custom_2", value: "1;trigger_hitback;10;1.0,hurt_f_1_s"},
		{key: "trigger_custom_3", value: "1;trigger_hitback;20;1.0,hurt_f_1_s"},
	}
	_, _, _, hitFrameMs = compileSkillActions(sameFrame, "CS_x_y")
	approx("same-frame-dedup", hitFrameMs, []float64{10.0 / 30, 20.0 / 30})
}
