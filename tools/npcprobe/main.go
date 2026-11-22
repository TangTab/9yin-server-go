// npcprobe NPC 目录装载探针：离线验证数据根（正式 $root\Res / 测试
// testdata）的摆放装载、模板联查与出生点筛选——起服前排查数据问题的
// 只读工具（不占互斥体）。
//
// 用法：go run ./tools/npcprobe -root <dir> [-scene born03] [-radius 500] [-limit 200]
//   <dir> 即数据根（其下有 ini/Npc/NpcConfig 与 ini/Scene/<场景>）。
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/lizhanyin/9yin-server-go/internal/npcdata"
)

func main() {
	root := flag.String("root", "", "数据根（下含 ini/Npc、ini/Scene；空=报用法）")
	scene := flag.String("scene", "born03", "场景键（找 <scene>_* 摆放目录）")
	radius := flag.Float64("radius", 500, "出生点半径（与 fssoloworld npcDefaultRadius 同缺省）")
	limit := flag.Int("limit", 200, "总量上限")
	flag.Parse()
	if *root == "" {
		fmt.Fprintln(os.Stderr, "用法: go run ./tools/npcprobe -root <数据根>（如 F:/Game/Online/9yin/9yin-server-go2/Res）")
		os.Exit(2)
	}

	npcs, err := npcdata.LoadSceneNPCs(*root, sceneDirOf(*root, *scene))
	if err != nil {
		fmt.Fprintf(os.Stderr, "装载失败: %v\n", err)
		os.Exit(1)
	}
	byFile := map[string]int{}
	for _, n := range npcs {
		byFile[n.File]++
	}
	fmt.Printf("场景 %s 静态白名单项 %d（类别分布：\n", *scene, len(npcs))
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		fmt.Printf("  %-22s %d\n", f, byFile[f])
	}
	fmt.Println(")")

	tpls, err := npcdata.LoadTemplates(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "模板装载失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("NpcConfig 模板 %d 个\n", len(tpls))
	hit := npcdata.Join(npcs, tpls)
	fmt.Printf("联查命中 %d/%d（%.0f%%；未命中项跳过不下发）\n", hit, len(npcs), 100*float64(hit)/float64(len(npcs)))

	kept := npcdata.Filter(npcs, 482.5, 1008, *radius, *limit) // born03 出生点 BornCJ000003
	shown := 0
	for _, n := range kept {
		if !n.Hit {
			continue
		}
		if shown < 5 {
			fmt.Printf("  [%d] %s  res=%s  (%.1f, %.1f, %.1f) ay=%.3f  name=%q scale=%q\n",
				shown+1, n.ID, n.Tpl.Resource, n.X, n.Y, n.Z, n.AY, n.Tpl.Name, n.Tpl.Scale)
			shown++
		}
	}
	fmt.Printf("筛选（半径 %.0f 上限 %d，出生点 482.5/1008）→ 可下发 %d 条\n", *radius, *limit, countHit(kept))
}

func countHit(npcs []npcdata.SceneNPC) int {
	n := 0
	for _, x := range npcs {
		if x.Hit {
			n++
		}
	}
	return n
}

// sceneDirOf 找 <scene>_* 摆放目录（fssoloworld findSceneDir 同口径的
// 只读版）。
func sceneDirOf(root, scene string) string {
	entries, err := os.ReadDir(root + "/ini/Scene")
	if err != nil {
		return scene // 缺目录时装载层报错
	}
	prefix := scene + "_"
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() && (name == scene || (len(name) > len(prefix) && equalFold(name[:len(prefix)], prefix))) {
			return name
		}
	}
	return scene
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 32
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
