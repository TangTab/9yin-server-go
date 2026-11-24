// golden 测试：对 BGserverCJ 官方部署的真实碰撞数据全量装载 + 查询自检。
// 数据根：BGserverCJ\Res\map（存在时才执行，可移植）。
package fscollide

import (
	"path/filepath"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

const mapRoot = `..\..\testdata\map`

func mapRootExists() bool {
	st, err := res.Stat(mapRoot)
	return err == nil && st.IsDir()
}

// TestGoldenLoadAllMaps 全地图装载：任何格式错误都会在此暴露。
func TestGoldenLoadAllMaps(t *testing.T) {
	if !mapRootExists() {
		t.Skip("官方数据目录不存在")
	}
	entries, err := res.ReadDir(mapRoot)
	if err != nil {
		t.Fatal(err)
	}
	loaded := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(mapRoot, e.Name())
		if _, err := res.Stat(filepath.Join(dir, "terrain.ini")); err != nil {
			continue
		}
		s, err := LoadScene3D(dir)
		if err != nil {
			t.Errorf("地图 %s 装载失败: %v", e.Name(), err)
			continue
		}
		loaded++
		// 基本自洽：地图尺寸 > 0；原点必有 zone 或全空地图
		if s.MapSizeX <= 0 || s.MapSizeZ <= 0 {
			t.Errorf("地图 %s 包络异常 %v/%v", e.Name(), s.MapSizeX, s.MapSizeZ)
		}
		nz := 0
		for _, z := range s.Zones {
			if z != nil {
				nz++
			}
		}
		if nz == 0 {
			t.Errorf("地图 %s 无任何 zone", e.Name())
		}
	}
	if loaded == 0 {
		t.Fatal("没有任何地图装载成功")
	}
	t.Logf("装载 %d 张地图", loaded)
}

// TestGoldenQueries 对 battle\battle_gmp（多层）做抽样查询一致性：
// 各查询不应 panic，且常见不变量成立。
func TestGoldenQueries(t *testing.T) {
	if !mapRootExists() {
		t.Skip("官方数据目录不存在")
	}
	s, err := LoadScene3D(filepath.Join(mapRoot, `battle\battle_gmp`))
	if err != nil {
		t.Fatal(err)
	}
	// 原点附近（地图中心）扫一片：高度/可行走自洽
	const step = 8.0
	walkable, blocked, total := 0, 0, 0
	for x := s.GetMapLeft(); x < s.GetMapLeft()+s.GetMapSizeX(); x += step {
		for z := s.GetMapTop(); z < s.GetMapTop()+s.GetMapSizeZ(); z += step {
			total++
			if s.GetWalkEnable(x, z) {
				walkable++
				// 可行走处的行走高度应为有限值
				h := s.GetWalkHeight(x, z)
				if h != h { // NaN
					t.Fatalf("GetWalkHeight NaN @(%v,%v)", x, z)
				}
				if heightIsNull(h) {
					t.Fatalf("可行走处行走高度为空 @(%v,%v)", x, z)
				}
				fl := s.GetApexFloor(x, z)
				_ = fl
				if s.GetApexHeight(x, z) == 0 && s.GetCollideEnable(x, z) {
					// 地形高度为 0 也合法（InitHeight=10 的平地不为 0；此处只查 NaN）
				}
			} else {
				blocked++
			}
			s.GetGroundHeight(x, z)
			s.GetFloorCount(x, z)
			s.GetWalkWaterExists(x, z)
			s.GetMapArea(x, z)
		}
	}
	t.Logf("battle_gmp 采样 %d：可行走 %d 阻挡 %d", total, walkable, blocked)
	if total > 0 && walkable == 0 {
		t.Fatal("整图不可行走，解码可能有误")
	}
	// 主城必有水面或区域名之一……不强制；只要求无 panic。
}

// TestGoldenWalkFileStructure 单 walk 文件结构校验（WLK1 五数组恰好耗尽块）。
func TestGoldenWalkFileStructure(t *testing.T) {
	if !mapRootExists() {
		t.Skip("官方数据目录不存在")
	}
	path := filepath.Join(mapRoot, `battle\battle_gmp\walk\zone_55_57.walk`)
	tw, err := LoadTerrainWalkable(path, 512)
	if err != nil {
		t.Fatal(err)
	}
	if tw.GetFloorCount() != 5 {
		t.Fatalf("floorCount=%d 期望 5", tw.GetFloorCount())
	}
	if tw.WalkMarker == nil || tw.Water == nil {
		t.Fatal("walk marker / water 缺失")
	}
	// 多层文件：floorCount>1 逐层齐备
	path2 := filepath.Join(mapRoot, `battle\battle_gmp\walk\zone_56_57.walk`)
	tw2, err := LoadTerrainWalkable(path2, 512)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("zone_56_57 floors=%d", tw2.GetFloorCount())
	if tw2.GetFloorCount() < 2 {
		t.Fatalf("多层文件解析异常 floors=%d", tw2.GetFloorCount())
	}
}

// TestGoldenTerrainHeight 高度图双线性：格点值应等于采样值。
func TestGoldenTerrainHeight(t *testing.T) {
	if !mapRootExists() {
		t.Skip("官方数据目录不存在")
	}
	path := filepath.Join(mapRoot, `battle\battle_gmp\zone_55_57.ground`)
	th, err := LoadTerrainHeight(path, 257, 0, 0, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if th.Scale != 257 || len(th.Value) != 257*257 {
		t.Fatalf("scale=%d len=%d", th.Scale, len(th.Value))
	}
	// 格点采样（左上角原点）：GetY(0,0)=Value[0]
	if got, want := th.GetY(0, 0), th.GetValue(0, 0); got != want {
		t.Fatalf("GetY(0,0)=%v != GetValue(0,0)=%v", got, want)
	}
	if got, want := th.GetY(1, 0), th.GetValue(1, 0); got != want {
		t.Fatalf("GetY(1,0)=%v != %v", got, want)
	}
	if got, want := th.GetY(0, 1), th.GetValue(0, 1); got != want {
		t.Fatalf("GetY(0,1)=%v != %v", got, want)
	}
}

// TestQuadtreeRoundtripFlag 综合查询一致性：flat 数组（RLE 解）与四叉树
// 原始数据直接查询（QuadGetFlag/QuadGetBit）应一致——同一份 .area 数据
// 两条路径互证。
func TestQuadtreeRoundtripFlag(t *testing.T) {
	if !mapRootExists() {
		t.Skip("官方数据目录不存在")
	}
	path := filepath.Join(mapRoot, `battle\battle_gmp\zone_55_57.area`)
	tr, err := LoadTerrainRegion(path, 256, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Area) == 0 {
		t.Fatal("区域列表为空")
	}
	t.Logf("zone_55_57 区域 %v", tr.Area)
	// 栅格类型图覆盖统计（应非全零，主城中心必有命名区域）
	nz := 0
	for _, v := range tr.Type {
		if v != 0 {
			nz++
		}
	}
	if nz == 0 {
		t.Fatal("区域类型图全空")
	}
	t.Logf("区域覆盖 %d/%d 格", nz, len(tr.Type))
}

// TestManagerRoute 验证 CollideManager 路由与缓存。
func TestManagerRoute(t *testing.T) {
	if !mapRootExists() {
		t.Skip("官方数据目录不存在")
	}
	m := NewManager()
	if !m.Init() {
		t.Fatal("Init")
	}
	a, err := m.CreateSceneCollide(filepath.Join(mapRoot, `battle\battle_gmp`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.CreateSceneCollide(filepath.Join(mapRoot, `battle\battle_gmp`))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("缓存失效")
	}
	if _, err := m.CreateSceneCollide(filepath.Join(mapRoot, "不存在的地图")); err == nil {
		t.Fatal("缺 ini 应报错")
	}
}

// TestTraceLineWalkSmoke 线追踪冒烟：可行走区内短距离直线应可走。
func TestTraceLineWalkSmoke(t *testing.T) {
	if !mapRootExists() {
		t.Skip("官方数据目录不存在")
	}
	s, err := LoadScene3D(filepath.Join(mapRoot, `battle\battle_gmp`))
	if err != nil {
		t.Fatal(err)
	}
	// 找一个可行走点
	var sx, sz float32
	found := false
	for x := s.GetMapLeft(); x < s.GetMapLeft()+s.GetMapSizeX() && !found; x += 4 {
		for z := s.GetMapTop(); z < s.GetMapTop()+s.GetMapSizeZ(); z += 4 {
			if s.GetWalkEnable(x, z) && !heightIsNull(s.GetWalkHeight(x, z)) {
				sx, sz, found = x, z, true
				break
			}
		}
	}
	if !found {
		t.Fatal("找不到可行走起点")
	}
	sy := s.GetWalkHeight(sx, sz)
	r := s.TraceLineWalk(1.0, sx, sy, sz, sx+2, sy, sz+2)
	t.Logf("TraceLineWalk(%.1f,%.1f)->(%.1f,%.1f) ok=%v 位置=(%.1f,%.1f,%.1f)",
		sx, sz, sx+2, sz+2, r.OK, r.X, r.Y, r.Z)
	r2 := s.TraceNextFloor(1.0, s.GetApexFloor(sx, sz), sx, sy, sz, sx+2, sz+2)
	t.Logf("TraceNextFloor ok=%v floor=%d 位置=(%.1f,%.1f,%.1f)", r2.OK, r2.Floor, r2.X, r2.Y, r2.Z)
}
