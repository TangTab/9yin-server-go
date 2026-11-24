// fscollide-probe：官方碰撞数据探针（独立测试工具，不进 go test 套件）。
//
// 单图模式（默认）：
//
//	go run ./cmd/fscollide-probe -map <地图目录>
//
// 抽样模式（每个类别目录挑 walk 文件最多的一张装载）：
//
//	go run ./cmd/fscollide-probe -sample <server_ter 根目录>
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	fscollide "github.com/lizhanyin/9yin-server-go/internal/fscollide"
	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

func main() {
	root := flag.String("map", `..\..\testdata\map\battle\battle_gmp`, "单图模式：地图目录")
	sample := flag.String("sample", "", "抽样模式：类别/<地图> 两层根目录")
	flag.Parse()

	if *sample != "" {
		sampleProbe(*sample)
		return
	}
	singleProbe(*root)
}

// singleProbe 单图：装载 + 包络采样统计 + 数据带抽查。
func singleProbe(mapDir string) {
	s, err := fscollide.LoadScene3D(mapDir)
	if err != nil {
		fmt.Println("装载失败:", err)
		os.Exit(1)
	}
	nz := 0
	for _, z := range s.Zones {
		if z != nil {
			nz++
		}
	}
	fmt.Printf("zones=%d mapLeft=%.1f mapTop=%.1f sizeX=%.1f sizeZ=%.1f zoneWorld=%.1f cell=%.3f\n",
		nz, s.GetMapLeft(), s.GetMapTop(), s.GetMapSizeX(), s.GetMapSizeZ(), s.ZoneWorld, s.CellSize)
	walkable, blocked := 0, 0
	for x := s.GetMapLeft(); x < s.GetMapLeft()+s.GetMapSizeX(); x += 8 {
		for z := s.GetMapTop(); z < s.GetMapTop()+s.GetMapSizeZ(); z += 8 {
			if s.GetWalkEnable(x, z) {
				walkable++
			} else {
				blocked++
			}
		}
	}
	fmt.Printf("包络采样: 可行走 %d / 阻挡 %d\n", walkable, blocked)
	var sx, sz float32
	found := false
	for x := s.GetMapLeft(); x < s.GetMapLeft()+s.GetMapSizeX() && !found; x += 4 {
		for z := s.GetMapTop(); z < s.GetMapTop()+s.GetMapSizeZ(); z += 4 {
			if s.GetWalkEnable(x, z) {
				sx, sz, found = x, z, true
				break
			}
		}
	}
	if found {
		sy := s.GetWalkHeight(sx, sz)
		r := s.TraceLineWalk(1.0, sx, sy, sz, sx+2, sy, sz+2)
		fmt.Printf("TraceLineWalk(%.1f,%.1f)->(%.1f,%.1f) ok=%v 落点=(%.1f,%.1f,%.1f)\n",
			sx, sz, sx+2, sz+2, r.OK, r.X, r.Y, r.Z)
	}
}

// sampleProbe 抽样：每个类别目录挑 walk 文件最多（最复杂）的一张。
func sampleProbe(root string) {
	entries, err := res.ReadDir(root)
	if err != nil {
		fmt.Println("读目录失败:", err)
		os.Exit(1)
	}
	loaded, failed, zones := 0, 0, 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cat := filepath.Join(root, e.Name())
		maps, _ := res.ReadDir(cat)
		best, bestN := "", -1
		for _, m := range maps {
			if !m.IsDir() {
				continue
			}
			dir := filepath.Join(cat, m.Name())
			if _, err := res.Stat(filepath.Join(dir, "terrain.ini")); err != nil {
				continue
			}
			n := 0
			if ws, err := res.ReadDir(filepath.Join(dir, "walk")); err == nil {
				n = len(ws)
			}
			if n > bestN {
				best, bestN = dir, n
			}
		}
		if best == "" {
			continue
		}
		s, err := fscollide.LoadScene3D(best)
		if err != nil {
			failed++
			fmt.Printf("[FAIL] %-16s %s: %v\n", e.Name(), filepath.Base(best), err)
			continue
		}
		nz := 0
		for _, z := range s.Zones {
			if z != nil {
				nz++
			}
		}
		loaded++
		zones += nz
		fmt.Printf("[OK]   %-16s 样本 %-16s zones=%d\n", e.Name(), filepath.Base(best), nz)
	}
	fmt.Printf("抽样装载 %d 类别 / 失败 %d / zones 合计 %d\n", loaded, failed, zones)
	if failed > 0 {
		os.Exit(1)
	}
}
