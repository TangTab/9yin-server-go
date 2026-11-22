package rolestate

// BirthSceneOptions/FindSceneOption：location.ini BornCJ 出生点表导出
//（GM 场景切换目的地解析数据源）。

import (
	"strings"
	"testing"
)

func TestBirthSceneOptions(t *testing.T) {
	opts := BirthSceneOptions()
	if len(opts) < 20 {
		t.Fatalf("scene options = %d, want >= 20 (BornCJ 全场景)", len(opts))
	}
	born03, ok := FindSceneOption("born03")
	if !ok {
		t.Fatal("born03 missing")
	}
	if born03.Config != `ini\scene\born03_YanYuZhuang` {
		t.Fatalf("born03 config = %q", born03.Config)
	}
	if born03.Position == [4]float64{} {
		t.Fatalf("born03 position empty: %+v", born03.Position)
	}
	// 大小写不敏感。
	if _, ok := FindSceneOption("CITY01"); !ok {
		t.Fatal("case-insensitive lookup failed")
	}
	// 直填形态不含的 clone 副本图也在 BornCJ 节内（clone001 无 '_' 全名键）。
	found := false
	for _, opt := range opts {
		if strings.HasPrefix(opt.Key, "clone") || strings.HasPrefix(opt.Key, "scene") {
			found = true
		}
	}
	if !found {
		t.Fatal("clone/scene keys missing from options")
	}
}
