package fscollide

import (
	"path/filepath"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

// TestModernBattleMaps 现代线战场地图（map_ter_battle_unpacked）全量装载。
func TestModernBattleMaps(t *testing.T) {
	const root = `..\..\testdata\map\battle`
	st, err := res.Stat(root)
	if err != nil || !st.IsDir() {
		t.Skip("战场地图目录不存在")
	}
	entries, _ := res.ReadDir(root)
	loaded := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		s, err := LoadScene3D(dir)
		if err != nil {
			t.Errorf("地图 %s 装载失败: %v", e.Name(), err)
			continue
		}
		nz, w := 0, 0
		for x := s.GetMapLeft(); x < s.GetMapLeft()+s.GetMapSizeX(); x += 4 {
			for z := s.GetMapTop(); z < s.GetMapTop()+s.GetMapSizeZ(); z += 4 {
				if s.GetWalkEnable(x, z) {
					w++
				}
			}
		}
		for _, z := range s.Zones {
			if z != nil {
				nz++
			}
		}
		loaded++
		t.Logf("%-16s zones=%2d 包络=%.0fx%.0f 采样可行走=%d", e.Name(), nz, s.GetMapSizeX(), s.GetMapSizeZ(), w)
	}
	if loaded == 0 {
		t.Fatal("没有地图装载")
	}
	t.Logf("战场地图装载 %d 张", loaded)
}
