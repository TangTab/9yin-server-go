package actiondata

// 招式判定节拍产物测试：仓库根 testdata/ini/Action 的产物缺席即跳过。
// 期望值对拍官服女号抓包的命中节拍（tools/pcapevents dumprec）：
//
//	zs_normal_00_01  判定 467/1000ms（实测 487/1051，含链路延迟）
//	CS_jh_cqgf01     判定 433/600/900ms（实测 452/682/964）
//	CS_jh_cqgf04     变速折算 1280ms（实测 1277）

import (
	"path/filepath"
	"reflect"
	"testing"
)

func realCatalog(t *testing.T) *Catalog {
	t.Helper()
	catalog, err := LoadDir(filepath.Join("..", "..", "testdata", "ini", "Action"))
	if err != nil {
		t.Skipf("招式节拍产物缺席（go run ./tools/datagen action）: %v", err)
	}
	return catalog
}

func TestHitScheduleMatchesOfficialCapture(t *testing.T) {
	catalog := realCatalog(t)
	if catalog.Count() < 10000 {
		t.Fatalf("招式数 %d 异常（产物不完整？）", catalog.Count())
	}
	cases := map[string]struct {
		hits   []int
		frames []int
		acts   []string
	}{
		"zs_normal_00_01": {[]int{467, 1000}, []int{15, 29}, []string{"ptgj_0h_01", "ptgj_0h_05"}},
		"CS_jh_cqgf01":    {[]int{433, 600, 900}, []int{45}, []string{"tzcq_10"}},
		"CS_jh_cqgf04":    {[]int{1280}, []int{58}, []string{"CS_fight_0h_jb_16"}},
		"CS_jh_cqgf06":    {[]int{500, 1067}, []int{17, 30}, []string{"jb_0h_06", "jb_0h_02"}},
	}
	for action, want := range cases {
		got, ok := catalog.Lookup(action)
		if !ok {
			t.Errorf("招式 %s 缺席", action)
			continue
		}
		if !reflect.DeepEqual(got.HitsMS, want.hits) || !reflect.DeepEqual(got.Frames, want.frames) ||
			!reflect.DeepEqual(got.Actions, want.acts) {
			t.Errorf("招式 %s = %+v, want hits=%v frames=%v actions=%v",
				action, got, want.hits, want.frames, want.acts)
		}
	}
	// 无判定帧的 NPC 起手动作（流程包给 HitTime）与未知招式。
	if got, ok := catalog.Lookup("atkhit1"); !ok || len(got.HitsMS) != 0 || got.Length != 1500 {
		t.Errorf("atkhit1 = %+v, %t（应无判定帧、总长 1500ms）", got, ok)
	}
	if _, ok := catalog.Lookup("no_such_action"); ok {
		t.Error("未知招式应返回 false")
	}
}

// 进程级入口 Load：数据根在探测链上即返回同一目录（进程级缓存）。
func TestLoadUsesProcessCache(t *testing.T) {
	first, err := Load()
	if err != nil {
		t.Skipf("数据根不在探测链上: %v", err)
	}
	second, err := Load()
	if err != nil || first != second {
		t.Fatalf("Load 二次调用应复用进程级缓存: %v %p %p", err, first, second)
	}
}
