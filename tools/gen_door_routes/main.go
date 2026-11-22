// gen_door_routes 客户端 door_data.ini → internal/doordata/door_routes.xml
// 一次性提取器（数据换代重跑即可）。
//
// 源：share/rule/door_data.ini（GBK；所需键全 ASCII 数值/路径，无需解码
// Name 串）。节=DoorPackageID：SceneName（源场景 resource 短名）+
// MoveTargetX/Y/Z（触发点）+ TargetScene/TargetX/Y/Z/Orient（目的地）。
// 半径用 SpringRange 缺省口径 8（门 NPC 模板粒度数据不在本表，取默认）。
//
// 用法：go run ./tools/gen_door_routes -src <door_data.ini> [-out <xml>]
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type route struct {
	id           string
	scene        string
	radius       float64
	triggerX     float64
	triggerY     float64
	triggerZ     float64
	targetConfig string
	targetX      float64
	targetY      float64
	targetZ      float64
	targetOrient float64
}

func main() {
	src := flag.String("src", `G:\pojie\ai\9yin_server_public\resources\modern\share\rule\door_data.ini`, "客户端 door_data.ini 路径")
	out := flag.String("out", "", "输出 XML（默认 internal/doordata/door_routes.xml）")
	flag.Parse()
	if *out == "" {
		exe, err := os.Getwd() // go run 时 cwd=仓库根
		if err != nil {
			fatal(err)
		}
		*out = filepath.Join(exe, "internal", "doordata", "door_routes.xml")
	}

	f, err := os.Open(*src)
	if err != nil {
		fatal(err)
	}
	defer f.Close()

	byScene := map[string][]route{}
	scanned, kept := 0, 0
	section := ""
	cur := route{}
	flush := func() {
		if section == "" || cur.targetConfig == "" {
			return
		}
		// 过滤：无源场景（副本包裹门——点击/任务制，无场景归属）丢弃；
		// 触发点全零的路线保留但 radius=0（仅点击路径可用，走近圆判定
		// 跳过——防 0,0,0 半径 8 在世界原点误触发）。
		cur.scene = normalizeSceneName(cur.scene)
		if cur.scene == "" {
			return
		}
		if cur.triggerX == 0 && cur.triggerY == 0 && cur.triggerZ == 0 {
			cur.radius = 0
		}
		cur.id = section
		byScene[cur.scene] = append(byScene[cur.scene], cur)
		kept++
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimSuffix(scanner.Text(), "\r"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			scanned++
			section, cur = strings.Trim(line, "[]"), route{radius: 8}
			continue
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 || section == "" {
			continue
		}
		key, val := strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
		switch key {
		case "SceneName":
			cur.scene = val
		case "MoveTargetX":
			cur.triggerX = atof(val)
		case "MoveTargetY":
			cur.triggerY = atof(val)
		case "MoveTargetZ":
			cur.triggerZ = atof(val)
		case "TargetScene":
			cur.targetConfig = val
		case "TargetX":
			cur.targetX = atof(val)
		case "TargetY":
			cur.targetY = atof(val)
		case "TargetZ":
			cur.targetZ = atof(val)
		case "TargetOrient":
			cur.targetOrient = atof(val)
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		fatal(err)
	}

	scenes := make([]string, 0, len(byScene))
	for name := range byScene {
		scenes = append(scenes, name)
	}
	sort.Strings(scenes)

	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	b.WriteString("<!-- 客户端 share/rule/door_data.ini 提取（tools/gen_door_routes）；请勿手改 -->\n")
	fmt.Fprintf(&b, "<DoorRoutes scenes=\"%d\" routes=\"%d\">\n", len(scenes), kept)
	for _, name := range scenes {
		fmt.Fprintf(&b, "  <Scene name=%q>\n", name)
		for _, r := range byScene[name] {
			fmt.Fprintf(&b, "    <Route id=%q triggerX=\"%s\" triggerY=\"%s\" triggerZ=\"%s\" radius=\"%s\""+
				" targetConfig=%q targetX=\"%s\" targetY=\"%s\" targetZ=\"%s\" targetOrient=\"%s\"/>\n",
				r.id, fnum(r.triggerX), fnum(r.triggerY), fnum(r.triggerZ), fnum(r.radius), r.targetConfig,
				fnum(r.targetX), fnum(r.targetY), fnum(r.targetZ), fnum(r.targetOrient))
		}
		b.WriteString("  </Scene>\n")
	}
	b.WriteString("</DoorRoutes>\n")

	if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("door routes: %d sections scanned, %d routes across %d scenes → %s\n", scanned, kept, len(scenes), *out)
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// normalizeSceneName 源场景名归一：单段中文后缀形态（city01_YanJing →
// city01）截成 resource 短名（与 rolestate.BirthSceneOptions 键域一致）；
// 多段变体名（scene04_2_XiangYangPoBei 是独立变体图）保持全形——不与本
// 线可达场景匹配即天然惰性，绝不误绑到基图。
func normalizeSceneName(name string) string {
	if name == "" {
		return ""
	}
	i := strings.IndexByte(name, '_')
	if i <= 0 {
		return name
	}
	head, rest := name[:i], name[i+1:]
	for _, prefix := range []string{"city", "school", "born", "scene"} {
		if len(head) > len(prefix) && strings.EqualFold(head[:len(prefix)], prefix) {
			if strings.ContainsRune(rest, '_') {
				return name // 多段变体（scene04_2_…）：独立图，保持全形
			}
			return head // city01_YanJing → city01
		}
	}
	return name
}

// fnum 数值串原样精度（去尾零浮点）。
func fnum(v float64) string {
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

func fatal(v any) {
	fmt.Fprintln(os.Stderr, "gen_door_routes:", v)
	os.Exit(1)
}
