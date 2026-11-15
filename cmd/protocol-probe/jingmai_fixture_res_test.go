package main

import (
	"path/filepath"
	"testing"
)

// 经脉目录的资源供给回归钉：jingmai.ini/jingmai_static.ini 必须能从
// resstore 夹具仓（本地开发/CI 无散装 resources 时唯一的资源来源）解析
// ——2026-11-07「经脉数据没有下发」的根因即夹具仓缺这两个文件，
// loadJingMaiCatalog 失败 → View45 整域静默降级。
func TestJingMaiCatalogResolvesFromFixtureStore(t *testing.T) {
	catalog, err := loadJingMaiCatalog(
		filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai.ini"),
		filepath.Join(defaultModernShareRoot, "skill", "jingmai", "jingmai_static.ini"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	if got := len(catalog.starterOrder()); got != 21 {
		t.Fatalf("starter order=%d, want 21 (2026-11-15 全套开通：14 官方新号 + 手厥阴/冲/带/维×2/跷×2)", got)
	}
	def, ok := catalog.def("jm_dantian")
	if !ok || def.maxLevel != 180 || def.itemType != 1003 {
		t.Fatalf("jm_dantian def=%+v ok=%v (want maxLevel=180)", def, ok)
	}
	// 穴位地图挂接（UI ini 夹具仓暂缺 → 空表不炸；无穴位图回落目录上限
	// 180——丹田满档前提，2026-11-15）。
	if cap := jingMaiCapFor(def.maxLevel, 0, int32(len(def.xueWei))); cap != 180 {
		t.Fatalf("no-xuewei cap=%d, want 180 (catalog max fallback)", cap)
	}
}
