package main

import (
	"path/filepath"
	"testing"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// TestNPCCatalogLoadsUnderBothDialects 换表工程回归闸（§14.3-P3）：
// NPC 目录在两种 0x09 表方言下都必须可建。2026-11-10 真机教训——
// 官服方言下 VisibleNPCSpawnDefaultsCompatV1 的 ActionSoundMode(int32)
// 撞官服表唯一现 string、placement 层 PosiX/Y/Z/Orient 为 fork-only 名，
// 两层硬错令 5413/5413 creator 实例尽没→选角链断连；修复后按当前表
// 场景域过滤注入。fork 基线与官服模式均须 spawns>0。
func TestNPCCatalogLoadsUnderBothDialects(t *testing.T) {
	reg, err := newSceneNPCRegistry(defaultNPCTableDir, filepath.Dir(defaultNPCCreatorDir))
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	for _, mode := range []string{"fork", "official2433"} {
		t.Run(mode, func(t *testing.T) {
			clientdata.SwapPropertyTableForTest(t, mode)
			spawns, stats, err := reg.catalogFor(role.Scene{Config: "city05", Resource: "city05_chengdu"})
			if err != nil {
				t.Fatalf("catalog: %v", err)
			}
			if len(spawns) == 0 {
				t.Fatal("catalog resolved zero spawns")
			}
			t.Logf("%s: spawns=%d resolved=%d unresolved=%d", mode, len(spawns), stats.Resolved, stats.Unresolved)
			if stats.Resolved == 0 {
				t.Fatal("resolved=0")
			}
		})
	}
}
