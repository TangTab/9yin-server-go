// npcprobe：场景 NPC 目录诊断（npcfeed 链）：摆放/联查/战斗命中率、
// 未解析与被闸样本、ident 分布、帧形态抽查（0x28/0x0D）。
//
//	npcprobe -root <数据根（含 ini/Npc/NpcConfig）> -scene born03
//	npcprobe -scene city05 -form 0d   # 帧形态抽查（dual/28/0d）
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

func main() {
	root := flag.String("root", "", "数据根（含 ini/Npc/NpcConfig；空=npcfeed.ResolveRoot 探测）")
	scene := flag.String("scene", "born03", "场景键（born03/city05/...）")
	form := flag.String("form", "dual", "帧形态抽查：dual=双发 0x0D+0x28（默认，与服务端一致）/ 28=仅 0x28 / 0d=仅裸 0x0D")
	dump := flag.Int("dump", 1, "帧 hexdump 条数")
	flag.Parse()

	dataRoot := *root
	if dataRoot == "" {
		dataRoot = npcfeed.ResolveRoot()
	}
	if dataRoot == "" {
		fmt.Fprintln(os.Stderr, "npcprobe: 数据根缺席（-root 或 NINEYIN_DATA_ROOT/Res/testdata 探测链）")
		os.Exit(1)
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
	fmt.Printf("摆放 %d | 联查 %d | 战斗命中 %d | 闸拦 %d | 物化 %d\n",
		stats.Placements, stats.Resolved, stats.CombatHit, stats.Skipped, len(spawns))
	fmt.Printf("战斗联查率 %.1f%%\n", pct(stats.CombatHit, stats.Resolved))
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
