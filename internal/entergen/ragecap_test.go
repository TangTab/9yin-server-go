package entergen

// 怒气上限分档测试（ragecap.go）：档位表逐档锚点、上限跟【运行】内功走
//（人物数据大批 / 基础属性批两处 MaxSP 同源）。

import (
	"strconv"
	"testing"
)

func TestNeiGongRageCapAddTiers(t *testing.T) {
	for _, tc := range []struct {
		configID string
		want     int32
	}{
		// 宗派：一内..六内、会意功（六内封顶 50）。
		{"ng_wd_001", 10}, {"ng_wd_002", 20}, {"ng_wd_003", 30}, {"ng_wd_004", 40},
		{"ng_wd_005", 50}, {"ng_wd_006", 50}, {"ng_wd_101", 50},
		{"ng_sl_003", 30}, {"ng_wxp_101", 50},
		// 隐世：档位按段内位置（ng_xd/ng_gm 段一内是 _002）。
		{"ng_xd_002", 30}, {"ng_xd_001", 40}, {"ng_gm_002", 30},
		{"ng_dm_001", 30}, {"ng_dm_002", 40},
		// 势力。
		{"ng_xj_001", 10}, {"ng_sj_001", 30}, {"ng_ws_001", 30},
		{"ng_tl_001", 40}, {"ng_qy_001", 40}, {"ng_yh_001", 40}, {"ng_wg_001", 40}, {"ng_th_001", 40},
		// 江湖：坐忘功 +10，其余 +40。
		{"ng_jh_001", 10}, {"ng_jh_002", 40}, {"ng_jyzj", 40}, {"ng_jysg", 40}, {"ng_task_001", 40},
		// 无依据：不加成。
		{"", 0}, {"ng_bh_001", 0}, {"skill_001", 0}, {"ng_wd_007", 0}, {"ng_wd_1", 0},
	} {
		if got := neiGongRageCapAdd(tc.configID); got != tc.want {
			t.Errorf("neiGongRageCapAdd(%q) = %d, want %d", tc.configID, got, tc.want)
		}
	}
}

// TestRageCapFollowsRunningNeiGong 上限只看【运行】的那本（非叠加：会的书
// 再多也不累计）；CurNeiGong 悬空时与人物数据大批同法回落首本。
func TestRageCapFollowsRunningNeiGong(t *testing.T) {
	ctx := testContext(t)
	if ctx.State.CurNeiGong != "ng_wd_001" || len(ctx.State.NeiGong) < 2 {
		t.Fatalf("建号运行内功 = %q（会 %d 本）, want 首本 ng_wd_001", ctx.State.CurNeiGong, len(ctx.State.NeiGong))
	}
	for _, tc := range []struct {
		running string
		want    int32
	}{
		{running: "ng_wd_001", want: 90},  // 武当一内：80 + 10
		{running: "ng_yh_001", want: 120}, // 移花宫（势力 +40）
		{running: "ng_jh_001", want: 90},  // 坐忘功
		{running: "ng_none", want: 90},    // 悬空：回落首本
	} {
		ctx.State.CurNeiGong = tc.running
		if got := RageCap(ctx); got != tc.want {
			t.Fatalf("运行 %s：RageCap = %d, want %d", tc.running, got, tc.want)
		}
		vital, err := PlayerVitalBatchFrame(ctx)
		if err != nil {
			t.Fatal(err)
		}
		base, err := BasePropertyFrames(ctx)
		if err != nil || len(base) != 1 {
			t.Fatalf("基础属性批: %d 帧, err=%v", len(base), err)
		}
		want := strconv.Itoa(int(tc.want))
		for name, frame := range map[string][]byte{"人物数据大批": vital, "基础属性批": base[0]} {
			props, _ := decodeVitalProps(t, frame)
			if props["MaxSP"] != want {
				t.Errorf("运行 %s：%s MaxSP = %q, want %s", tc.running, name, props["MaxSP"], want)
			}
		}
	}

	// 无内功：基础位。
	ctx.State.NeiGong, ctx.State.CurNeiGong = nil, ""
	if got := RageCap(ctx); got != baseMaxSP || baseMaxSP != 80 {
		t.Fatalf("无内功 RageCap = %d, want %d", got, baseMaxSP)
	}
}
