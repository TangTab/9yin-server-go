// gen_scene_waypoints 场景/驿站表生成器：客户端解包树 × 服务端摆放档 →
// internal/scenedata/scene_waypoints.xml（一次性提取器，数据换代重跑即可）。
//
// 源（客户端资源解包树，散文件）：
//
//	map\ter\*                     世界场景全集（有地形 = 客户端可加载）
//	share\rule\homepoint.ini      驿站表 [HomePoint<resource><字母>]
//	share\relive\sencerelive.ini  各场景复活点行（同址驿站，带朝向）
//	share\relive\relive_info.ini  默认 / 门派复活点行
//	text\chineses\*.idres         文案（scene_<resource>=场景名、ui_<驿站名>；
//	                               stringname.idres 另有按场景配置目录登记的
//	                               ini\scene\<目录名>=场景名）
//
// 源（服务端数据根，资源仓）：
//
//	ini\Scene\*                      场景配置目录名（Config 全路径）
//	ini\Scene\<dir>\npc_creator.xml  摆放档（无驿站场景的落点）
//
// 另读 internal/doordata/door_routes.xml（传送门路线目标）。
//
// 覆盖 = 客户端有地形的全部世界场景 city/born/school/sceneNN。落点选取：
//
//	驿站（坐标取同址复活点行——带朝向；默认落点 = 首个安全驿站）
//	  > 传送门路线目标
//	  > 摆放档功能 NPC（复活药师 > 驿站/车夫/前往 > 路标 > 主场景功能
//	    NPC 质心最近者，X 偏 2m 防与 NPC 模型重叠）。
//
// 用法：go run ./tools/gen_scene_waypoints [-client <解包树>] [-data <数据根>]
//
//	[-out <xml>] [-verify]
package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/npcdata"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	defaultClient = `F:\Game\Online\9yin\九阴资源\resources`
	defaultRel    = `internal/scenedata/scene_waypoints.xml`
	doorsRel      = `internal/doordata/door_routes.xml`

	kindHome = "home" // 驿站 / 复活点（客户端表）
	kindDoor = "door" // 传送门路线目标
	kindNPC  = "npc"  // 摆放档功能 NPC 旁

	// configKeyPrefix 场景配置目录键前缀（stringname.idres：
	// ini\scene\<目录名>=场景名；键不分大小写）。
	configKeyPrefix = "ini" + `\` + "scene" + `\`

	// npcLandingOffset NPC 落点 X 偏移（米）——落在 NPC 正中与模型重叠。
	npcLandingOffset = 2.0
	header           = "<!-- 生成物：go run ./tools/gen_scene_waypoints；勿手改（数据换代重跑，-verify 对拍）。 -->\n"
)

var (
	worldSceneRe    = regexp.MustCompile(`^(city|born|school|scene)(\d+)$`)
	homeTailRe      = regexp.MustCompile(`^(.*\d)([a-z]{1,2})$`)
	schoolTypoRe    = regexp.MustCompile(`^schoo(\d)`)
	reviveKeyPrefix = regexp.MustCompile(`^([a-z]+\d+)_`)
	propRe          = regexp.MustCompile(`<Property\s+[^>]*>`)
	fileAttr        = regexp.MustCompile(`\sFile="([^"]*)"`)
	idAttr          = regexp.MustCompile(`\sid="([^"]*)"`)
	descAttr        = regexp.MustCompile(`\sdesc="([^"]*)"`)
	xAttr           = regexp.MustCompile(`\sx="([^"]*)"`)
	yAttr           = regexp.MustCompile(`\sy="([^"]*)"`)
	zAttr           = regexp.MustCompile(`\sz="([^"]*)"`)
	subAttr         = regexp.MustCompile(`\sSubSceneNo="([^"]*)"`)
	categoryOrder   = map[string]int{"city": 0, "born": 1, "school": 2, "scene": 3}
	transportDescs  = []string{"驿站", "车夫", "码头", "船夫", "前往"}
	landmarkDescs   = []string{"路标"}
	logicObjDescs   = []string{"控制"}
	funcNPCFiles    = []string{"funcnpc.xml", "commonnpc.xml"}
)

// ---------- 产物形态（internal/scenedata 的装载契约） ----------

type waypointXML struct {
	Name   string  `xml:"name,attr"`
	Title  string  `xml:"title,attr,omitempty"`
	X      float64 `xml:"x,attr"`
	Y      float64 `xml:"y,attr"`
	Z      float64 `xml:"z,attr"`
	Orient float64 `xml:"orient,attr"`
	Kind   string  `xml:"kind,attr"`
	Safe   bool    `xml:"safe,attr,omitempty"`
	Source string  `xml:"source,attr,omitempty"` // 复活点行键 / 传送门 id / NPC 档
}

type sceneXML struct {
	Resource string        `xml:"resource,attr"`
	Config   string        `xml:"config,attr"`
	Name     string        `xml:"name,attr,omitempty"`
	Category string        `xml:"category,attr"`
	Landing  string        `xml:"landing,attr"` // 默认落点 = Waypoint[].Name
	Waypoint []waypointXML `xml:"Waypoint"`
}

type docXML struct {
	XMLName xml.Name   `xml:"SceneWaypoints"`
	Scene   []sceneXML `xml:"Scene"`
}

// ---------- 源行 ----------

type homePoint struct {
	name, uiName string
	x, y, z      float64
	safe         bool
}

// revive 复活点行：key=场景号,x,y,z,朝向[,ui名[,驿站名]]。
type revive struct {
	key, ui, home   string
	x, y, z, orient float64
}

type npcRow struct {
	file, id, desc string
	x, y, z        float64
	sub            string
}

type iniSection struct {
	name string
	keys map[string]string
}

type doorRoute struct {
	ID           string  `xml:"id,attr"`
	TargetConfig string  `xml:"targetConfig,attr"`
	TargetX      float64 `xml:"targetX,attr"`
	TargetY      float64 `xml:"targetY,attr"`
	TargetZ      float64 `xml:"targetZ,attr"`
	TargetOrient float64 `xml:"targetOrient,attr"`
}

type doorDoc struct {
	Scene []struct {
		Route []doorRoute `xml:"Route"`
	} `xml:"Scene"`
}

// builtWaypoint 落点 + 是否带朝向（取自复活点行）。
type builtWaypoint struct {
	wp       waypointXML
	oriented bool
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "gen_scene_waypoints:", err)
	os.Exit(1)
}

func main() {
	client := flag.String("client", os.Getenv("NINEYIN_CLIENT_RES"), "客户端资源解包树根（散文件；env NINEYIN_CLIENT_RES）")
	data := flag.String("data", "", "服务端数据根（空 = npcfeed.ResolveRoot 推导：Res / testdata）")
	out := flag.String("out", "", "输出 XML（默认 "+defaultRel+"）")
	verify := flag.Bool("verify", false, "不写文件：与现有产物逐字节比对，不一致 exit 1")
	flag.Parse()

	root, err := repoRoot()
	if err != nil {
		fatal(err)
	}
	if *client == "" {
		*client = defaultClient
	}
	if *data == "" {
		*data = npcfeed.ResolveRoot()
	}
	if *data == "" {
		fatal(errors.New("服务端数据根缺席：-data 指定含 ini/Npc/NpcConfig 的目录"))
	}
	if *out == "" {
		*out = filepath.Join(root, filepath.FromSlash(defaultRel))
	}
	doors, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(doorsRel)))
	if err != nil {
		fatal(err)
	}
	doc, notes, err := generate(*client, *data, doors)
	if err != nil {
		fatal(err)
	}
	for _, note := range notes {
		fmt.Println(note)
	}
	rendered, err := render(doc)
	if err != nil {
		fatal(err)
	}
	if *verify {
		existing, err := os.ReadFile(*out)
		if err != nil {
			fatal(err)
		}
		if !bytes.Equal(bytes.ReplaceAll(existing, []byte("\r\n"), []byte("\n")), rendered) {
			fatal(fmt.Errorf("%s 与源数据不一致：重跑 go run ./tools/gen_scene_waypoints", *out))
		}
		fmt.Printf("verify ok: %s（%d 场景）\n", *out, len(doc.Scene))
		return
	}
	if err := os.WriteFile(*out, rendered, 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %s（%d 场景）\n", *out, len(doc.Scene))
}

// repoRoot 自 cwd 向上找 go.mod 所在目录。
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("仓库根缺席：找不到 go.mod")
		}
		dir = parent
	}
}

// generate 组表。notes = 人读备注（落点来源统计、无中文名、目录歧义等）。
func generate(client, data string, doorsXML []byte) (*docXML, []string, error) {
	var notes []string
	resources, err := clientWorldScenes(client)
	if err != nil {
		return nil, nil, err
	}
	homeText, err := readGBK(filepath.Join(client, "share", "rule", "homepoint.ini"))
	if err != nil {
		return nil, nil, err
	}
	var reviveTexts []string
	for _, rel := range []string{"sencerelive.ini", "relive_info.ini"} {
		text, err := readGBK(filepath.Join(client, "share", "relive", rel))
		if err != nil {
			return nil, nil, err
		}
		reviveTexts = append(reviveTexts, text)
	}
	names, err := loadNames(filepath.Join(client, "text", "chineses"))
	if err != nil {
		return nil, nil, err
	}
	homesBy := map[string][]homePoint{}
	for _, h := range parseHomePoints(homeText) {
		if r, ok := resourceOfHomeName(h.name); ok {
			homesBy[r] = append(homesBy[r], h)
		}
	}
	revivesBy := map[string][]revive{}
	for _, text := range reviveTexts {
		for _, rv := range parseRevives(text) {
			if r, ok := resourceOfRevive(rv); ok {
				revivesBy[r] = append(revivesBy[r], rv)
			}
		}
	}
	dirs, err := serverSceneDirs(data)
	if err != nil {
		return nil, nil, err
	}
	var doors doorDoc
	if err := xml.Unmarshal(doorsXML, &doors); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", doorsRel, err)
	}
	// resource → 首条传送门目标（文档序）
	doorTarget := map[string]doorRoute{}
	for _, sc := range doors.Scene {
		for _, r := range sc.Route {
			r.TargetConfig = singleSlashes(r.TargetConfig)
			key := npcdata.SceneResourceOfBase(lastSegment(r.TargetConfig))
			if _, ok := doorTarget[key]; !ok {
				doorTarget[key] = r
			}
		}
	}

	doc := &docXML{}
	byKind := map[string]int{}
	var unnamed []string
	for _, r := range resources {
		config, dir, note := sceneConfig(r, dirs)
		if _, born := rolestate.FindSceneOption(r); !born {
			config = configSpelling(config, dir, doorTarget[r])
		}
		if note != "" {
			notes = append(notes, "警告: "+note)
		}
		category := worldSceneRe.FindStringSubmatch(r)[1]
		scene := sceneXML{Resource: r, Config: config, Name: sceneName(r, config, names), Category: category}
		built := buildWaypoints(homesBy[r], revivesBy[r], names)
		if len(built) == 0 {
			built = fallbackWaypoints(r, doorTarget[r], dir, data)
		}
		if len(built) == 0 {
			notes = append(notes, fmt.Sprintf("警告: %s（%s）无任何落点来源，未收录", r, config))
			continue
		}
		landing := pickLanding(built)
		scene.Landing = built[landing].wp.Name
		for _, b := range built {
			scene.Waypoint = append(scene.Waypoint, b.wp)
		}
		doc.Scene = append(doc.Scene, scene)
		byKind[built[landing].wp.Kind]++
		if scene.Name == "" {
			unnamed = append(unnamed, r)
		}
	}
	notes = append(notes, fmt.Sprintf("场景 %d：默认落点 驿站 %d / 传送门 %d / 功能 NPC %d",
		len(doc.Scene), byKind[kindHome], byKind[kindDoor], byKind[kindNPC]))
	if len(unnamed) > 0 {
		notes = append(notes, "文案表无场景名（显示回退资源名）: "+strings.Join(unnamed, " "))
	}
	return doc, notes, nil
}

// fallbackWaypoints 无驿站场景：传送门路线目标 > 摆放档功能 NPC。
func fallbackWaypoints(r string, door doorRoute, dir, data string) []builtWaypoint {
	if door.TargetConfig != "" {
		return []builtWaypoint{{wp: waypointXML{
			Name: "door:" + door.ID, X: round3(door.TargetX), Y: round3(door.TargetY), Z: round3(door.TargetZ),
			Orient: round3(door.TargetOrient), Kind: kindDoor, Source: door.ID,
		}, oriented: true}}
	}
	if dir == "" {
		return nil
	}
	placements, err := res.ReadFile(filepath.Join(data, "ini", "Scene", dir, "npc_creator.xml"))
	if err != nil {
		return nil
	}
	row, ok := pickNPC(parseNPCPlacements(placements))
	if !ok {
		return nil
	}
	return []builtWaypoint{{wp: waypointXML{
		Name: "npc:" + row.id, Title: row.desc,
		X: round3(row.x + npcLandingOffset), Y: round3(row.y), Z: round3(row.z),
		Kind: kindNPC, Source: row.file,
	}}}
}

// buildWaypoints 场景驿站序 = homepoint.ini 节序，坐标 / 朝向取同址复活点行
// （复活点是服务端权威落位）。场景无驿站时，归属到它的复活点行充当驿站
// （名 = 行键；有驿站的场景不计孤儿行——变体地形坐标空间的行不混入）。
func buildWaypoints(homes []homePoint, revives []revive, names map[string]string) []builtWaypoint {
	var out []builtWaypoint
	matched := make([]bool, len(revives))
	for _, h := range homes {
		w := waypointXML{Name: h.name, Title: names[strings.ToLower(h.uiName)], X: round3(h.x), Y: round3(h.y), Z: round3(h.z), Safe: h.safe, Kind: kindHome, Source: "homepoint"}
		oriented := false
		for i, rv := range revives {
			if !matched[i] && sameHomeName(h.name, rv.home) {
				w.X, w.Y, w.Z, w.Orient, w.Source = round3(rv.x), round3(rv.y), round3(rv.z), round3(rv.orient), rv.key
				matched[i], oriented = true, true
				break
			}
		}
		out = append(out, builtWaypoint{wp: w, oriented: oriented})
	}
	if len(homes) > 0 {
		return out
	}
	seen := map[[3]float64]bool{}
	used := map[string]int{}
	for i, rv := range revives {
		pos := [3]float64{round3(rv.x), round3(rv.y), round3(rv.z)}
		if matched[i] || seen[pos] {
			continue
		}
		seen[pos] = true
		// 名 = 行键（scene11_01）：孤儿行的驿站名字段不可靠——快活岛 14 行
		// 位置各异，第 7 字段却同为 HomePointscene11A（驿站表无此节）。同行
		// 键异坐标（副本模板跨节登记）加 #2 序号保场景内唯一。
		name := rv.key
		if n := used[name]; n > 0 {
			name = fmt.Sprintf("%s#%d", rv.key, n+1)
		}
		used[rv.key]++
		out = append(out, builtWaypoint{wp: waypointXML{
			Name: name, Title: names[strings.ToLower(rv.ui)], X: pos[0], Y: pos[1], Z: pos[2],
			Orient: round3(rv.orient), Kind: kindHome, Source: rv.key,
		}, oriented: true})
	}
	return out
}

// pickLanding 默认落点：首个安全且带朝向的 > 首个安全 > 首个带朝向 > 首个。
func pickLanding(wps []builtWaypoint) int {
	want := func(pass int, w builtWaypoint) bool {
		switch pass {
		case 0:
			return w.wp.Safe && w.oriented
		case 1:
			return w.wp.Safe
		case 2:
			return w.oriented
		}
		return true
	}
	for pass := 0; pass < 4; pass++ {
		for i, w := range wps {
			if want(pass, w) {
				return i
			}
		}
	}
	return 0
}

// sceneConfig 场景 resource → 服务端 Config（ini\scene\<目录>）与数据根里的
// 实际目录名。目录选取：BornCJ 登记的权威目录（八派正图 school01_JinYiWei，
// 非同前缀变体 _fangong/_mie）> 同名目录 > 字典序首个（歧义告警）。服务端
// 无目录时回落 ini\scene\<resource>、dir 空（进场无 NPC，告警）。
func sceneConfig(r string, dirs []string) (config, dir, note string) {
	if opt, ok := rolestate.FindSceneOption(r); ok {
		return opt.Config, findDir(dirs, lastSegment(opt.Config)), ""
	}
	var candidates []string
	for _, d := range dirs {
		if npcdata.SceneResourceOfBase(d) == r {
			candidates = append(candidates, d)
		}
	}
	if len(candidates) == 0 {
		return `ini\scene\` + r, "", fmt.Sprintf("%s 服务端无场景目录（ini/Scene），进场无 NPC", r)
	}
	sort.Strings(candidates)
	dir, exact := candidates[0], false
	for _, d := range candidates {
		if strings.EqualFold(d, r) {
			dir, exact = d, true
			break
		}
	}
	if len(candidates) > 1 && !exact {
		note = fmt.Sprintf("%s 场景目录多变体，取字典序首个 %s（%s）", r, dir, strings.Join(candidates, " / "))
	}
	return `ini\scene\` + dir, dir, note
}

// configSpelling Config 串的写法（仅 BornCJ 未登记的场景——登记的保持
// BornCJ 原样，那是现网一直在发的写法）：客户端传送门表（door_routes.xml）
// 登记了该场景时沿用其 TargetScene（官服经门进场时下发的写法，如
// school09_YiHuaGong；客户端表内大小写本就不统一，路径比对不区分大小写）；
// 没有登记则用服务端目录名。
func configSpelling(config, dir string, door doorRoute) string {
	if dir == "" || door.TargetConfig == "" || !strings.EqualFold(lastSegment(door.TargetConfig), dir) {
		return config
	}
	return strings.ReplaceAll(door.TargetConfig, "/", `\`)
}

// singleSlashes door_routes.xml 的 targetConfig 是 gen_door_routes 以 %q 写出
// 的，反斜杠成对（ini\\scene\\x）——还原成客户端原表的单反斜杠写法。
func singleSlashes(s string) string {
	return strings.ReplaceAll(s, `\\`, `\`)
}

// pickNPC 摆放档落点：复活药师 > 驿站/车夫/前往 > 路标 > 主场景功能 NPC
// 质心最近者。主场景 = SubSceneNo 空（全在子场景时放宽）。
func pickNPC(rows []npcRow) (npcRow, bool) {
	var main []npcRow
	for _, row := range rows {
		if row.sub == "" || row.sub == "0" || row.sub == "-1" {
			main = append(main, row)
		}
	}
	if len(main) == 0 {
		main = rows
	}
	for _, row := range main {
		if strings.HasPrefix(strings.ToLower(row.id), "relive") {
			return row, true
		}
	}
	for _, words := range [][]string{transportDescs, landmarkDescs} {
		for _, row := range main {
			if descHasAny(row.desc, words) {
				return row, true
			}
		}
	}
	var cx, cy, cz float64
	n := 0
	for _, row := range main {
		if logicObject(row) {
			continue
		}
		cx, cy, cz = cx+row.x, cy+row.y, cz+row.z
		n++
	}
	if n == 0 {
		return npcRow{}, false
	}
	best, bestD := npcRow{}, math.MaxFloat64
	for _, row := range main {
		if logicObject(row) {
			continue
		}
		if d := math.Hypot(row.x-cx/float64(n), row.z-cz/float64(n)); d < bestD {
			best, bestD = row, d
		}
	}
	return best, true
}

// logicObject 质心统计排除项：非功能 NPC 档，或逻辑控制物（不可见）。
func logicObject(row npcRow) bool {
	if !isFuncFile(row.file) || descHasAny(row.desc, logicObjDescs) {
		return true
	}
	return false
}

func isFuncFile(file string) bool {
	for _, f := range funcNPCFiles {
		if strings.EqualFold(file, f) {
			return true
		}
	}
	return false
}

func descHasAny(desc string, words []string) bool {
	for _, w := range words {
		if strings.Contains(desc, w) {
			return true
		}
	}
	return false
}

// parseNPCPlacements npc_creator.xml 摆放行（文档序；只留功能 NPC 档）。
func parseNPCPlacements(xmlText []byte) []npcRow {
	var out []npcRow
	for _, m := range propRe.FindAllString(string(xmlText), -1) {
		file := attrOf(fileAttr, m)
		if !isFuncFile(file) {
			continue
		}
		out = append(out, npcRow{
			file: file, id: attrOf(idAttr, m), desc: attrOf(descAttr, m),
			x: numAttr(xAttr, m), y: numAttr(yAttr, m), z: numAttr(zAttr, m),
			sub: attrOf(subAttr, m),
		})
	}
	return out
}

func attrOf(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

func numAttr(re *regexp.Regexp, s string) float64 {
	v, _ := strconv.ParseFloat(attrOf(re, s), 64)
	return v
}

// parseHomePoints homepoint.ini → 驿站序（节序）。PositonXYZ 是原表拼写。
func parseHomePoints(text string) []homePoint {
	var out []homePoint
	for _, sec := range parseINISections(text) {
		x, y, z := parseXYZ(sec.keys["positonxyz"])
		out = append(out, homePoint{name: sec.name, uiName: sec.keys["name"], x: x, y: y, z: z, safe: sec.keys["safe"] == "1"})
	}
	return out
}

// parseINISections ini → 节序（键小写；节内同键保首见；节前的键丢弃）。
func parseINISections(text string) []iniSection {
	var out []iniSection
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			out = append(out, iniSection{name: strings.TrimSpace(line[1 : len(line)-1]), keys: map[string]string{}})
			continue
		}
		i := strings.Index(line, "=")
		if i <= 0 || len(out) == 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:i]))
		if _, dup := out[len(out)-1].keys[key]; !dup {
			out[len(out)-1].keys[key] = strings.TrimSpace(line[i+1:])
		}
	}
	return out
}

// parseRevives 复活点文件（sencerelive.ini / relive_info.ini）行序：
// key=场景号,x,y,z,朝向[,ui名[,驿站名]]（字段数 6~8 不等；行键可跨节重复）。
func parseRevives(text string) []revive {
	var out []revive
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		i := strings.Index(line, "=")
		if i <= 0 || strings.HasPrefix(line, "//") {
			continue
		}
		f := strings.Split(line[i+1:], ",")
		if len(f) < 5 {
			continue
		}
		if _, err := strconv.Atoi(strings.TrimSpace(f[0])); err != nil {
			continue // 非复活点行
		}
		rv := revive{key: strings.TrimSpace(line[:i])}
		rv.x, _ = strconv.ParseFloat(strings.TrimSpace(f[1]), 64)
		rv.y, _ = strconv.ParseFloat(strings.TrimSpace(f[2]), 64)
		rv.z, _ = strconv.ParseFloat(strings.TrimSpace(f[3]), 64)
		rv.orient, _ = strconv.ParseFloat(strings.TrimSpace(f[4]), 64)
		if len(f) >= 6 {
			rv.ui = strings.TrimSpace(f[5])
		}
		if len(f) >= 7 {
			rv.home = strings.TrimSpace(f[6])
		}
		out = append(out, rv)
	}
	return out
}

// parseXYZ "x,y,z"（多余字段忽略；解析失败的轴为 0）。
func parseXYZ(s string) (x, y, z float64) {
	parts := strings.Split(s, ",")
	vals := []*float64{&x, &y, &z}
	for i := 0; i < 3 && i < len(parts); i++ {
		*vals[i], _ = strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
	}
	return x, y, z
}

// resourceOfHomeName 驿站名 → 场景 resource：HomePointschool09A /
// ui_HomePointschool17A → school09 / school17（尾字母 1~2 个）。非此命名 false。
func resourceOfHomeName(name string) (string, bool) {
	if !strings.Contains(strings.ToLower(name), "homepoint") {
		return "", false
	}
	m := homeTailRe.FindStringSubmatch(normalizeHomeName(name))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// resourceOfRevive 复活点行 → 场景 resource：驿站名 > ui 名 > 行键前缀
// （born03_02 → born03；无数字前缀的键如 newschool_xuedao 不归属）。
func resourceOfRevive(rv revive) (string, bool) {
	if r, ok := resourceOfHomeName(rv.home); ok {
		return r, true
	}
	if r, ok := resourceOfHomeName(rv.ui); ok {
		return r, true
	}
	if m := reviveKeyPrefix.FindStringSubmatch(strings.ToLower(rv.key)); m != nil {
		return m[1], true
	}
	return "", false
}

// sameHomeName 驿站名比对（归一后相等）。
func sameHomeName(a, b string) bool {
	return b != "" && normalizeHomeName(a) == normalizeHomeName(b)
}

// normalizeHomeName 驿站名归一：小写、去 ui_ 前缀、去 HomePoint 前缀、
// 归正 schoo 少 l 的拼写（relive_info.ini 的 HomePointschoo14A）。
func normalizeHomeName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	for _, prefix := range []string{"ui_pos_", "ui_"} {
		s = strings.TrimPrefix(s, prefix)
	}
	return schoolTypoRe.ReplaceAllString(strings.TrimPrefix(s, "homepoint"), "school${1}")
}

// clientWorldScenes 客户端地形目录里的世界场景 resource（city/born/school/
// sceneNN；变体 school01_mie / scene08_2 不计），按类序 + 编号排。
func clientWorldScenes(client string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(client, "map", "ter"))
	if err != nil {
		return nil, fmt.Errorf("客户端地形目录: %w", err)
	}
	seen := map[string]bool{}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m := worldSceneRe.FindStringSubmatch(strings.ToLower(e.Name()))
		if m == nil || seen[m[1]+m[2]] {
			continue
		}
		seen[m[1]+m[2]] = true
		out = append(out, m[1]+m[2])
	}
	if len(out) == 0 {
		return nil, errors.New("客户端地形目录下没有 city/born/school/sceneNN 场景")
	}
	sortResources(out)
	return out, nil
}

// sortResources 原地排：类序（主城 > 出生镇 > 门派 > 江湖）> 编号数值序。
func sortResources(resources []string) {
	sort.Slice(resources, func(i, j int) bool { return resourceLess(resources[i], resources[j]) })
}

// resourceLess 类序（主城 > 出生镇 > 门派 > 江湖）> 编号数值序。
func resourceLess(a, b string) bool {
	ma, mb := worldSceneRe.FindStringSubmatch(a), worldSceneRe.FindStringSubmatch(b)
	if ma[1] != mb[1] {
		return categoryOrder[ma[1]] < categoryOrder[mb[1]]
	}
	na, _ := strconv.Atoi(ma[2])
	nb, _ := strconv.Atoi(mb[2])
	if na != nb {
		return na < nb
	}
	return a < b
}

func serverSceneDirs(data string) ([]string, error) {
	entries, err := res.ReadDir(filepath.Join(data, "ini", "Scene"))
	if err != nil {
		return nil, fmt.Errorf("ini/Scene: %w", err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs, nil
}

// sceneName 场景中文名：scene_<resource> 键；缺席时取 stringname.idres 按
// 场景配置目录登记的 ini\scene\<目录名> 键（school21_shenjiying=神机营、
// scene36=星垂城——这两个场景只有后一种键）。都没有 = 空。
func sceneName(resource, config string, names map[string]string) string {
	if name := names["scene_"+resource]; name != "" {
		return name
	}
	return names[configKeyPrefix+strings.ToLower(lastSegment(config))]
}

// loadNames 文案表（text\chineses\*.idres，UTF-8 BOM，个别 GBK）：scene_* /
// ui_* / ini\scene\* 键（键小写、首见优先）。
func loadNames(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("文案目录: %w", err)
	}
	names := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".idres") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		text := strings.TrimPrefix(string(raw), "\ufeff")
		if !utf8.ValidString(text) {
			decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), raw)
			if err != nil {
				continue
			}
			text = strings.TrimPrefix(string(decoded), "\ufeff")
		}
		for _, line := range strings.Split(text, "\n") {
			i := strings.Index(line, "=")
			if i <= 0 {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(line[:i]))
			if !strings.HasPrefix(key, "scene_") && !strings.HasPrefix(key, "ui_") && !strings.HasPrefix(key, configKeyPrefix) {
				continue
			}
			if _, dup := names[key]; !dup {
				names[key] = strings.TrimSpace(line[i+1:])
			}
		}
	}
	return names, nil
}

// readGBK 客户端表是 GBK。
func readGBK(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	out, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), raw)
	if err != nil {
		return "", fmt.Errorf("%s: GBK 解码: %w", path, err)
	}
	return string(out), nil
}

// lastSegment 路径末段（\ 与 / 皆容）。
func lastSegment(path string) string {
	if i := strings.LastIndexAny(path, `\/`); i >= 0 {
		return path[i+1:]
	}
	return path
}

// findDir 目录名大小写不敏感精确匹配（缺席空）。
func findDir(dirs []string, name string) string {
	for _, d := range dirs {
		if strings.EqualFold(d, name) {
			return d
		}
	}
	return ""
}

// round3 坐标保留 3 位小数（毫米级；源表 2~6 位不等）；-0 归 0。
func round3(v float64) float64 {
	r := math.Round(v*1000) / 1000
	if r == 0 {
		return 0
	}
	return r
}

// render 头注释 + 两空格缩进 XML（LF；行尾入库后由 autocrlf 处理）。
func render(doc *docXML) ([]byte, error) {
	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(header)+len(body)+1)
	out = append(out, header...)
	out = append(out, body...)
	return append(out, '\n'), nil
}
