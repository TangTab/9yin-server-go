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
	if got := len(catalog.starterOrder()); got != 14 {
		t.Fatalf("starter order=%d, want 14", got)
	}
	def, ok := catalog.def("jm_dantian")
	if !ok || def.maxLevel != 180 || def.itemType != 1003 {
		t.Fatalf("jm_dantian def=%+v ok=%v (want maxLevel=180)", def, ok)
	}
}
