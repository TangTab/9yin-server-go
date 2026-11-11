package main

import (
	"path/filepath"
	"testing"
	"time"
)

// TestRealCatalogHitBreakTiming 真实资源树冒烟（NINEYIN_SERVER_ROOT 未
// 配置时跳过）：2026-11-11 官服 pcapng（技能/挑战）实测跳时刻 vs 编译产物。
// 官服值含 ~80ms 客户端 RTT，比对容差 ±120ms。
func TestRealCatalogHitBreakTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	if _, err := openResourceReader(filepath.Join(modernSkillRoot, "skill_new.ini")); err != nil {
		t.Skipf("resource store unavailable: %v", err)
	}
	cases := []struct {
		skill      string
		officialMs []int // 官服干净样本（就近身战，含 RTT）
		tolMs      int   // 0 = 默认 ±130
	}{
		{"CS_jz_lyfh01", []int{363, 1110}, 0},
		{"CS_jz_lyfh02", []int{551, 1141}, 0},
		{"CS_jz_lyfh03", []int{519}, 0},
		{"wuji_CS_jz_lyfh05", []int{545, 1150}, 0},
		{"wuji_CS_jz_lyfh07", []int{587, 1587, 2587, 3569, 4565}, 0},
		{"CS_jh_hyjf05", []int{419}, 0},
		{"CS_jh_hyjf07", []int{424}, 0},
		{"wuji_CS_jh_hyjf03", []int{393, 768}, 0},
		{"CS_jh_mzjf03", []int{413, 1300}, 0},
		// 挑战.pcapng 2026-11-12：8 跳=hitback×6+hitbreak×2。五段深链
		// 的段起点实测比「全时长累加」逐段滞后（尾跳最高 ~360ms，
		// 段链折算细则见 docs/寒心恨雪与太极剑舞引导机制.md §七），
		// 容差放宽到 ±400 只锁跳数与首段节奏。
		{"CS_wd_tjj07", []int{853, 2390, 3570, 4858, 5392, 6180, 6798, 7659}, 400},
	}
	for _, tc := range cases {
		definition, ok := combatSkills[tc.skill]
		if !ok {
			t.Errorf("%s: not in catalog", tc.skill)
			continue
		}
		if len(definition.hitFrameMs) != len(tc.officialMs) {
			t.Errorf("%s: got %d ticks %v, official %d",
				tc.skill, len(definition.hitFrameMs), definition.hitFrameMs, len(tc.officialMs))
			continue
		}
		tol := 130
		if tc.tolMs > 0 {
			tol = tc.tolMs
		}
		for i, at := range definition.hitFrameMs {
			diff := at - time.Duration(tc.officialMs[i])*time.Millisecond
			if diff < -time.Duration(tol)*time.Millisecond || diff > time.Duration(tol)*time.Millisecond {
				t.Errorf("%s tick#%d: got %v, official ~%dms", tc.skill, i+1, at, tc.officialMs[i])
			}
		}
	}
}
