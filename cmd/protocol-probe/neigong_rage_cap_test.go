package main

import (
	"bytes"
	"testing"
)

// TestNeiGongRageCapAddTiers 钉死档位表(docs/neigong-beginner-guide.md §六
// × neigong_type.ini 快照;桃花岛=指南"丁氏山庄"+40 槽位,坐忘功=
// ng_jh_001 由 stringname.idres 实证)。
func TestNeiGongRageCapAddTiers(t *testing.T) {
	cases := []struct {
		configID string
		want     int32
	}{
		// starter 目录六本
		{"ng_wd_001", 10}, // 武当一内(两仪护心功)
		{"ng_wd_002", 20}, // 武当二内
		{"ng_em_001", 10}, // 峨眉一内
		{"ng_yh_001", 40}, // 移花宫(势力+40)
		{"ng_th_001", 40}, // 桃花岛(碧波心经,势力槽位)
		{"ng_jh_001", 10}, // 坐忘功
		// 宗派位置档:一内..六内 10..50 封顶,会意 _101=50
		{"ng_sl_001", 10}, {"ng_sl_003", 30}, {"ng_sl_005", 50},
		{"ng_sl_006", 50}, {"ng_xt_006", 50}, {"ng_wxp_004", 40},
		{"ng_wd_101", 50}, {"ng_em_101", 50},
		// 隐世按段内位置:ng_xd/ng_gm 首位是 _002(一内+30)
		{"ng_xd_002", 30}, {"ng_xd_001", 40},
		{"ng_gm_002", 30}, {"ng_gm_001", 40},
		{"ng_cf_001", 30}, {"ng_cf_002", 40},
		{"ng_bg_001", 30}, {"ng_bg_002", 40},
		// 势力固定档
		{"ng_xj_001", 10}, // 徐家庄
		{"ng_sj_001", 30}, // 金针沈家
		{"ng_ws_001", 30}, // 万兽山庄
		{"ng_tl_001", 40}, // 天轮寺
		{"ng_qy_001", 40}, // 青衣阁
		{"ng_wg_001", 40}, // 无根门
		// 江湖:坐忘功 +10,其它(含九阴/九阳真经、造化功、ng_jh_5xx)+40
		{"ng_jh_101", 40}, {"ng_jh_507", 40},
		{"ng_jyzj", 40}, {"ng_jysg", 40}, {"ng_task_001", 40},
		// 指南未覆盖/未知:0(不虚构)
		{"ng_bh_001", 0},   // 三皇劲(天)
		{"ng_slqj_001", 0}, // 气剑系
		{"ng_qfg_001", 0},
		{"", 0}, {"ng_unknown", 0}, {"buf_ng_wd_001", 0},
	}
	for _, tc := range cases {
		if got := neiGongRageCapAdd(tc.configID); got != tc.want {
			t.Errorf("neiGongRageCapAdd(%q)=%d, want %d", tc.configID, got, tc.want)
		}
	}
}

// TestRageCapFollowsEquippedNeiGong 换书改变有效上限:饱和、初始满值
// 与 wire 上的 MaxSPAdd 同步切换。
func TestRageCapFollowsEquippedNeiGong(t *testing.T) {
	player := newPlayerActor("tester", 0)
	// 默认 ng_wd_001(一内+10):登录即满 90,MaxSPAdd=10。
	if got := player.rageCap(); got != maxPlayerSP+10 {
		t.Fatalf("default rageCap=%d, want %d", got, maxPlayerSP+10)
	}
	if got := player.skillSP(); got != player.rageCap() {
		t.Fatalf("login SP=%d, want full %d", got, player.rageCap())
	}
	frame, err := player.loginVitalUpdate()
	if err != nil {
		t.Fatal(err)
	}
	addNeedle := []byte{byte(583 & 0xff), byte(583 >> 8), 10, 0, 0, 0}
	if !bytes.Contains(frame, addNeedle) {
		t.Fatalf("login MaxSPAdd is not 10 (ng_wd_001): %x", frame)
	}

	// 切到移花宫明玉神功(+40):上限 120,存量怒气保持,MaxSPAdd=40。
	if err := player.equipNeiGong("ng_yh_001"); err != nil {
		t.Fatal(err)
	}
	if got := player.rageCap(); got != maxPlayerSP+40 {
		t.Fatalf("ng_yh_001 rageCap=%d, want %d", got, maxPlayerSP+40)
	}
	sp, _, err := player.setSP(999)
	if err != nil || sp != maxPlayerSP+40 {
		t.Fatalf("setSP(999)=%d err=%v, want %d", sp, err, maxPlayerSP+40)
	}
	capFrame, err := player.rageCapUpdate()
	if err != nil {
		t.Fatal(err)
	}
	addNeedle = []byte{byte(583 & 0xff), byte(583 >> 8), 40, 0, 0, 0}
	if !bytes.Contains(capFrame, addNeedle) {
		t.Fatalf("cap frame MaxSPAdd is not 40 (ng_yh_001): %x", capFrame)
	}

	// 切回坐忘功(+10):上限收缩回 90,存量 120 收敛到 90。
	if err := player.equipNeiGong("ng_jh_001"); err != nil {
		t.Fatal(err)
	}
	if got := player.skillSP(); got != maxPlayerSP+10 {
		t.Fatalf("SP after shrinking cap=%d, want %d", got, maxPlayerSP+10)
	}
}
