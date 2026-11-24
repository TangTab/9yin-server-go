// npcprobe：场景 NPC 目录诊断（npcfeed 链）：摆放/联查/战斗命中率、
// 未解析与被闸样本、ident 分布、帧形态抽查（0x28/0x0D）、巡逻绑定。
//
//	npcprobe -root <数据根（含 ini/Npc/NpcConfig 或 share/npc/npcconfig）> -scene born03
//	npcprobe -scene city05 -form 0d   # 帧形态抽查（dual/28/0d）
//	npcprobe -root <Res> -all         # 全场景扫描：逐场景物化数 + 巡逻绑定 + 合计
//
// 数据来源跟随 NINEYIN_DATA_SOURCE（share 直读 / xml 构建产物），输出首部标明。
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/sharedata"
)

func main() {
	root := flag.String("root", "", "数据根（含 ini/Npc/NpcConfig 或 share/npc/npcconfig；空=npcfeed.ResolveRoot 探测）")
	scene := flag.String("scene", "born03", "场景键（born03/city05/...）")
	form := flag.String("form", "dual", "帧形态抽查：dual=双发 0x0D+0x28（默认，与服务端一致）/ 28=仅 0x28 / 0d=仅裸 0x0D")
	dump := flag.Int("dump", 1, "帧 hexdump 条数")
	all := flag.Bool("all", false, "扫描全部场景目录：逐场景物化数 + 巡逻绑定 + 合计")
	flag.Parse()

	dataRoot := *root
	if dataRoot == "" {
		dataRoot = npcfeed.ResolveRoot()
	}
	if dataRoot == "" {
		fmt.Fprintln(os.Stderr, "npcprobe: 数据根缺席（-root 或 NINEYIN_DATA_ROOT/Res/testdata 探测链）")
		os.Exit(1)
	}
	if *all {
		if err := sweep(dataRoot); err != nil {
			fmt.Fprintln(os.Stderr, "npcprobe:", err)
			os.Exit(1)
		}
		return
	}
	spawns, stats, err := npcfeed.BuildCatalog(dataRoot, *scene)
	if err != nil {
		fmt.Fprintln(os.Stderr, "npcprobe:", err)
		os.Exit(1)
	}

	// 类别分布。
	byFile := map[string]int{}
	byScript := map[string]int{}
	for _, s := range spawns {
		byFile[s.File]++
		byScript[s.Script]++
	}
	fmt.Printf("root=%s scene=%s\n", dataRoot, *scene)
	fmt.Printf("数据来源：%s\n", npcfeed.SourcesOf(dataRoot))
	fmt.Printf("摆放 %d | 联查 %d | 战斗命中 %d | 闸拦 %d | 物化 %d\n",
		stats.Placements, stats.Resolved, stats.CombatHit, stats.Skipped, len(spawns))
	fmt.Printf("战斗联查率 %.1f%%\n", pct(stats.CombatHit, stats.Resolved))
	// 巡逻绑定（数据根 share/creator/mappath 路线表；与服务端 sceneNPCCatalog 同链）。
	mappathDir := mappathDirOf(dataRoot)
	bind := npcfeed.BindPatrol(spawns, npcfeed.LoadPatrolRoutes(mappathDir, npcfeed.SceneDirOf(dataRoot, *scene)), time.Now().UnixMilli())
	fmt.Printf("巡逻 walkers %d | 站立 %d | 路线缺席 %d | 离路线过远 %d\n",
		bind.Walkers, bind.Stand, bind.MissingPath, bind.FarOrShort)
	warnMappathMissing(mappathDir)
	// 出生姿态（静息态≠stand：坐/喝/倒地等；物化帧 State=BornAction）。
	poses, poseNames := poseDist(spawns)
	fmt.Printf("姿态 %d（%d 种动作）\n", poses, len(poseNames))
	dumpDist("姿态", poseNames)
	dumpDist("类别", byFile)
	dumpDist("类名", byScript)
	for _, sample := range stats.Samples {
		fmt.Println("  样本:", sample)
	}

	// 帧形态抽查（dual=双发 0x0D+0x28，与服务端缺省一致）。
	var frameForm npcfeed.Form
	switch *form {
	case "0d", "0D":
		frameForm = npcfeed.Form0DOnly
	case "28", "0x28":
		frameForm = npcfeed.Form0x28Only
	default:
		frameForm = npcfeed.FormDual
	}
	frames, err := npcfeed.Frames(spawns, frameForm)
	if err != nil {
		fmt.Fprintln(os.Stderr, "npcprobe: 帧生成:", err)
		os.Exit(1)
	}
	var totalBytes int
	for _, f := range frames {
		totalBytes += len(f)
	}
	fmt.Printf("帧 %d 条（%s 形态）共 %d 字节，均值 %.0f B/条\n",
		len(frames), formName(frameForm), totalBytes, avg(float64(totalBytes), float64(len(frames))))
	for i := 0; i < *dump && i < len(frames) && i < 3; i++ {
		head := frames[i]
		if len(head) > 96 {
			head = head[:96]
		}
		fmt.Printf("帧 %d 前 %d 字节:\n%s\n", i, len(head), hex.Dump(head))
	}
}

// mappathDirOf 巡逻路线目录（与服务端 fssoloworld.resolveMappathDir 同序：
// env NINEYIN_MAPPATH_DIR > <数据根>/share/creator/mappath）。
func mappathDirOf(dataRoot string) string {
	if v := strings.TrimSpace(os.Getenv("NINEYIN_MAPPATH_DIR")); v != "" {
		return v
	}
	return filepath.Join(dataRoot, "share", "creator", "mappath")
}

// warnMappathMissing 路线目录缺席：服务端静默返回空表、NPC 全部站立。
func warnMappathMissing(mappathDir string) {
	if _, err := res.Stat(mappathDir); err != nil {
		fmt.Printf("  警告: 巡逻路线目录缺席 %s —— NPC 全部站立（把客户端 share/creator/mappath 投放数据根同路径，或 go run ./tools/datagen patrol）\n", mappathDir)
	}
}

// sweep 全场景扫描：逐场景（目录装载 + 巡逻绑定，与服务端同链），只列
// 有路线或有巡逻绑定的场景，末行合计。场景清单与数据都跟随来源开关
// （NINEYIN_DATA_SOURCE，首行标明三域各取自哪里）。
func sweep(dataRoot string) error {
	scenes, err := npcfeed.SceneDirs(dataRoot)
	if err != nil {
		return fmt.Errorf("读场景目录: %w", err)
	}
	mappathDir := mappathDirOf(dataRoot)
	fmt.Printf("root=%s mappath=%s\n", dataRoot, mappathDir)
	fmt.Printf("数据来源：%s（NINEYIN_DATA_SOURCE=%q）\n", npcfeed.SourcesOf(dataRoot), strings.TrimSpace(os.Getenv("NINEYIN_DATA_SOURCE")))
	if note := sharedata.StaleShareNote(dataRoot); note != "" {
		fmt.Printf("注意：%s\n", note)
	}
	warnMappathMissing(mappathDir)
	var sceneCount, spawns, withRoutes, patrolled, failed, poses int
	var poseNames map[string]int
	var total npcfeed.PatrolBindStats
	nowMs := time.Now().UnixMilli()
	for _, scene := range scenes {
		sceneCount++
		list, _, err := npcfeed.BuildCatalogDir(dataRoot, scene)
		if err != nil {
			failed++
			continue
		}
		spawns += len(list)
		routes := npcfeed.LoadPatrolRoutes(mappathDir, scene)
		bind := npcfeed.BindPatrol(list, routes, nowMs)
		scenePoses, scenePoseNames := poseDist(list)
		// 合计先于筛列：无路线档的场景里「路线缺席」也要计入。
		total.MissingPath += bind.MissingPath
		total.FarOrShort += bind.FarOrShort
		poses += scenePoses
		for name, n := range scenePoseNames {
			if poseNames == nil {
				poseNames = map[string]int{}
			}
			poseNames[name] += n
		}
		if len(routes) == 0 && bind.Walkers == 0 && scenePoses == 0 {
			continue
		}
		if len(routes) > 0 {
			withRoutes++
		}
		if bind.Walkers > 0 {
			patrolled++
		}
		total.Walkers += bind.Walkers
		fmt.Printf("%-32s 物化 %5d | 路线 %4d | walkers %4d | 路线缺席 %3d | 过远 %3d | 姿态 %4d\n",
			scene, len(list), len(routes), bind.Walkers, bind.MissingPath, bind.FarOrShort, scenePoses)
	}
	fmt.Printf("合计：场景 %d（装载失败 %d）| 物化 %d | 有路线场景 %d | 有巡逻场景 %d | walkers %d | 路线缺席 %d | 过远 %d | 姿态 %d（%d 种动作）\n",
		sceneCount, failed, spawns, withRoutes, patrolled, total.Walkers, total.MissingPath, total.FarOrShort, poses, len(poseNames))
	dumpDist("姿态", poseNames)
	return nil
}

// poseDist 出生姿态统计：静息态≠stand 的 NPC 数与动作名分布（物化帧
// State=BornAction=动作名，见 npcfeed/pose.go）。
func poseDist(spawns []npcfeed.Spawn) (int, map[string]int) {
	names := map[string]int{}
	for _, s := range spawns {
		if rest := s.RestAction(); rest != npcfeed.StandAction {
			names[rest]++
		}
	}
	total := 0
	for _, n := range names {
		total += n
	}
	return total, names
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}

func avg(sum, n float64) float64 {
	if n == 0 {
		return 0
	}
	return sum / n
}

func formName(form npcfeed.Form) string {
	switch form {
	case npcfeed.FormDual:
		return "双发 0x0D+0x28"
	case npcfeed.Form0x28Only:
		return "0x28 mode=0x46"
	default:
		return "0x0D"
	}
}

func dumpDist(label string, m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] })
	fmt.Printf("%s分布（%d 键）:\n", label, len(keys))
	for i, k := range keys {
		if i >= 10 {
			fmt.Printf("  … 余 %d 键\n", len(keys)-i)
			break
		}
		fmt.Printf("  %-24s %d\n", k, m[k])
	}
}
