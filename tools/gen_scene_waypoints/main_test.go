package main

// 生成器纯函数单测：表解析 / 驿站归属 / 落点优先级 / 场景目录选取。
// 夹具取源表方言的最小样本（节序、拼写、字段数不等等坑各留一例）。

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const homeSample = "\n[HomePointschool09A]\nArea=area_school09_mianlou\nName=ui_HomePointschool09A\nPositonXYZ=344.25,48.0,659.75\nSafe=1\nSceneID=281\nType=2\n\n// 注释行\n[HomePointschool09B]\nName=ui_HomePointschool09B\nPositonXYZ=827.5,-9.8,1237.5\nSafe=0\nSceneID=281\n"

const reviveSample = "//初始复活点\n[init]\nforce=17,911.073,31.192,537.350,1.742,ui_pos_ReliveCJ101702,HomePointcity05B\nnewschool_huashan=571,1561.900,156.214,731.400,-1.900,ui_HomePointschool17A\n\n[281]\nschool09_02=281,827.378,-9.839,1236.628,3.238,ui_HomePointschool09B,HomePointschool09B\nschool09_01=281,340.160,47.979,657.284,2.588,ui_HomePointschool09A,HomePointschool09A\nnot_a_point=abc\nshort=281,1,2\n"

func TestParseHomePoints(t *testing.T) {
	homes := parseHomePoints(homeSample)
	if len(homes) != 2 {
		t.Fatalf("homes = %d, want 2", len(homes))
	}
	a := homes[0]
	if a.name != "HomePointschool09A" || a.uiName != "ui_HomePointschool09A" || !a.safe ||
		a.x != 344.25 || a.y != 48 || a.z != 659.75 {
		t.Fatalf("home A = %+v", a)
	}
	if homes[1].name != "HomePointschool09B" || homes[1].safe {
		t.Fatalf("home B = %+v", homes[1])
	}
}

func TestParseRevives(t *testing.T) {
	revives := parseRevives(reviveSample)
	if len(revives) != 4 {
		t.Fatalf("revives = %d, want 4（非复活点行/字段不足行跳过）", len(revives))
	}
	r := revives[3]
	if r.key != "school09_01" || r.x != 340.16 || r.y != 47.979 || r.z != 657.284 || r.orient != 2.588 ||
		r.ui != "ui_HomePointschool09A" || r.home != "HomePointschool09A" {
		t.Fatalf("revive = %+v", r)
	}
	// 6 字段行（无驿站名）
	if got := revives[1]; got.home != "" || got.ui != "ui_HomePointschool17A" || got.x != 1561.9 {
		t.Fatalf("6-field revive = %+v", got)
	}
}

func TestResourceOfHomeName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"HomePointschool09A", "school09"},
		{"ui_HomePointschool17A", "school17"},
		{"HomePointschoo14A", "school14"}, // relive_info 少写 l
		{"HomePointgroupscene045B", "groupscene045"},
		{"ui_pos_ReliveCJ100501", ""},
		{"born03_02", ""},
		{"", ""},
	}
	for _, c := range cases {
		got, ok := resourceOfHomeName(c.in)
		if c.want == "" {
			if ok {
				t.Errorf("%q resolved to %q, want no match", c.in, got)
			}
			continue
		}
		if !ok || got != c.want {
			t.Errorf("%q → %q ok=%v, want %q", c.in, got, ok, c.want)
		}
	}
}

func TestResourceOfRevive(t *testing.T) {
	cases := []struct {
		revive
		want string
	}{
		{revive{key: "x", home: "HomePointschool09A"}, "school09"},
		{revive{key: "x", ui: "ui_HomePointschool17A"}, "school17"},
		{revive{key: "born03_02"}, "born03"},
		{revive{key: "newschool_xuedao", ui: "ui_pos_ReliveCJ100201"}, ""},
		{revive{key: "force", ui: "ui_pos_ReliveCJ101702", home: "HomePointcity05B"}, "city05"},
	}
	for _, c := range cases {
		got, ok := resourceOfRevive(c.revive)
		if c.want == "" {
			if ok {
				t.Errorf("%+v resolved to %q, want no match", c.revive, got)
			}
			continue
		}
		if !ok || got != c.want {
			t.Errorf("%+v → %q ok=%v, want %q", c.revive, got, ok, c.want)
		}
	}
}

func TestBuildWaypoints(t *testing.T) {
	names := map[string]string{"ui_homepointschool09a": "移花宫眠楼", "ui_homepointschool09b": "雾花林渡口"}
	homes := parseHomePoints(homeSample)
	revives := parseRevives(reviveSample)
	built := buildWaypoints(homes, revives, names)
	if len(built) != 2 {
		t.Fatalf("built = %d, want 2", len(built))
	}
	// 同址复活点行按驿站名匹配（非行序）：坐标与朝向取复活点行（权威
	// 落位），标题取驿站文案。
	a := built[0]
	if a.wp.X != 340.16 || a.wp.Y != 47.979 || a.wp.Z != 657.284 || a.wp.Orient != 2.588 || !a.oriented ||
		!a.wp.Safe || a.wp.Title != "移花宫眠楼" || a.wp.Source != "school09_01" {
		t.Fatalf("waypoint A = %+v", a)
	}
	// B 的同址行排在 A 的之前，仍各归各位。
	b := built[1]
	if b.wp.X != 827.378 || b.wp.Orient != 3.238 || b.wp.Safe || b.wp.Title != "雾花林渡口" {
		t.Fatalf("waypoint B = %+v", b)
	}
	if landing := pickLanding(built); landing != 0 {
		t.Fatalf("landing = %d, want 0（首个安全且带朝向）", landing)
	}
}

func TestBuildWaypointsOrphanRevives(t *testing.T) {
	// 无驿站场景：归属到它的复活点行充当驿站，同坐标去重，名 = 行键。
	revives := []revive{
		{key: "scene11_01", x: 1, y: 2, z: 3, orient: 0.5, ui: "ui_HomePointscene11A"},
		{key: "scene11_01", x: 1, y: 2, z: 3, orient: 0.5}, // 跨节重复行
		{key: "scene11_02", x: 4, y: 5, z: 6, orient: 1.5, home: "HomePointscene11B"},
	}
	built := buildWaypoints(nil, revives, map[string]string{"ui_homepointscene11a": "码头"})
	if len(built) != 2 {
		t.Fatalf("built = %d, want 2", len(built))
	}
	if built[0].wp.Name != "scene11_01" || built[0].wp.Title != "码头" || !built[0].oriented {
		t.Fatalf("orphan = %+v", built[0])
	}
	// 快活岛方言：位置各异的多行共用一个驿站名字段——仍按行键命名，不撞名。
	if built[1].wp.Name != "scene11_02" {
		t.Fatalf("orphan must be named by row key, got %+v", built[1])
	}
	// 同行键异坐标：加序号保场景内唯一。
	spread := []revive{
		{key: "clone003_1", x: 1, y: 1, z: 1}, {key: "clone003_1", x: 2, y: 2, z: 2}, {key: "clone003_1", x: 3, y: 3, z: 3},
	}
	spreadBuilt := buildWaypoints(nil, spread, nil)
	names := []string{spreadBuilt[0].wp.Name, spreadBuilt[1].wp.Name, spreadBuilt[2].wp.Name}
	if !reflect.DeepEqual(names, []string{"clone003_1", "clone003_1#2", "clone003_1#3"}) {
		t.Fatalf("spread names = %v", names)
	}
}

func TestPickLanding(t *testing.T) {
	home := func(name string, safe, oriented bool) builtWaypoint {
		return builtWaypoint{wp: waypointXML{Name: name, Safe: safe}, oriented: oriented}
	}
	cases := []struct {
		name string
		wps  []builtWaypoint
		want int
	}{
		{"安全且带朝向优先于仅安全", []builtWaypoint{home("plain", false, false), home("safe", true, false), home("both", true, true)}, 2},
		{"仅安全优先于仅带朝向", []builtWaypoint{home("plain", false, false), home("safe", true, false), home("ori", false, true)}, 1},
		{"仅带朝向优先于首个", []builtWaypoint{home("plain", false, false), home("ori", false, true)}, 1},
		{"全无标记取首个", []builtWaypoint{home("plain", false, false)}, 0},
	}
	for _, c := range cases {
		if got := pickLanding(c.wps); got != c.want {
			t.Errorf("%s: pickLanding = %d, want %d", c.name, got, c.want)
		}
	}
}

func npc(file, id, desc string, x, y, z float64, sub string) npcRow {
	return npcRow{file: file, id: id, desc: desc, x: x, y: y, z: z, sub: sub}
}

func TestPickNPC(t *testing.T) {
	plain := []npcRow{
		npc("funcnpc.xml", "ev_a", "考古点", 0, 1, 0, ""),
		npc("funcnpc.xml", "ev_b", "考古点", 10, 1, 0, ""),
		npc("funcnpc.xml", "ctl", "BOSS刷新控制", 500, 1, 500, ""),
		npc("attacknpc.xml", "mob", "野狼", 900, 1, 900, ""),
	}
	// 无标识 NPC：质心最近者（逻辑控制物/非功能档不计入质心）——ev_a/ev_b
	// 关于质心 (5,0) 对称，取先到者 ev_a。
	got, ok := pickNPC(plain)
	if !ok || got.id != "ev_a" {
		t.Fatalf("centroid pick = %+v ok=%v", got, ok)
	}
	// 路标 > 质心。
	withLandmark := append([]npcRow{}, plain...)
	withLandmark = append(withLandmark, npc("funcnpc.xml", "sign", "无名洞路标", -300, 9, -300, ""))
	if got, _ := pickNPC(withLandmark); got.id != "sign" {
		t.Fatalf("landmark pick = %+v", got)
	}
	// 驿站/车夫 > 路标。
	withTransport := append([]npcRow{}, withLandmark...)
	withTransport = append(withTransport, npc("commonnpc.xml", "yizhan", "红尘驿站", -500, 9, 500, ""))
	if got, _ := pickNPC(withTransport); got.id != "yizhan" {
		t.Fatalf("transport pick = %+v", got)
	}
	// 复活药师 > 一切。
	withRelive := append([]npcRow{}, withTransport...)
	withRelive = append(withRelive, npc("funcnpc.xml", "Relive_Fhjiangyou", "游方药师", 1, 2, 3, ""))
	if got, _ := pickNPC(withRelive); got.id != "Relive_Fhjiangyou" || got.x != 1 {
		t.Fatalf("relive pick = %+v", got)
	}
	// 只看主场景：复活药师在子场景时不选，主场景无可用行才放宽。
	subOnly := []npcRow{
		npc("funcnpc.xml", "Relive_sub", "游方药师", 1, 1, 1, "3100"),
		npc("funcnpc.xml", "ev_main", "考古点", 5, 1, 5, ""),
	}
	if got, _ := pickNPC(subOnly); got.id != "ev_main" {
		t.Fatalf("sub-scene pick = %+v", got)
	}
	if got, ok := pickNPC([]npcRow{npc("funcnpc.xml", "ctl", "踢人控制器", 1, 1, 1, "")}); ok {
		t.Fatalf("only logic objects must not resolve, got %+v", got)
	}
}

func TestParseNPCPlacements(t *testing.T) {
	xmlText := []byte("<Object>\n<Property File=\"attacknpc.xml\" No=\"X\" x=\"1\" y=\"2\" z=\"3\" id=\"mob\" desc=\"野狼\"/>\n" +
		"<Property File=\"funcnpc.xml\" x=\"-826.24\" y=\"62.86\" z=\"-2940.23\" id=\"Relive_Fhjiangyou\" desc=\"游方药师\" SubSceneNo=\"3100\"/>\n" +
		"<Property File=\"commonnpc.xml\" x=\"10.5\" y=\"1\" z=\"-2\" id=\"a1\" desc=\"码头伙计\"/>\n</Object>")
	rows := parseNPCPlacements(xmlText)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2（只留功能 NPC 档）", len(rows))
	}
	if rows[0].id != "Relive_Fhjiangyou" || rows[0].sub != "3100" || rows[0].x != -826.24 || rows[0].desc != "游方药师" {
		t.Fatalf("row 0 = %+v", rows[0])
	}
	if rows[1].x != 10.5 || rows[1].z != -2 || rows[1].sub != "" {
		t.Fatalf("row 1 = %+v", rows[1])
	}
}

func TestSceneConfig(t *testing.T) {
	// BornCJ 登记的场景取权威目录（八派正图，非 _fangong 变体）。
	config, dir, note := sceneConfig("school01", []string{"school01_fangong", "school01_jinyiwei", "school01_mie"})
	if config != `ini\scene\school01_JinYiWei` || dir != "school01_jinyiwei" || note != "" {
		t.Fatalf("bornCJ scene = %q dir=%q note=%q", config, dir, note)
	}
	// 非 BornCJ：resource 键下唯一目录。
	if config, dir, _ = sceneConfig("school10", []string{"school10_taohuadao"}); config != `ini\scene\school10_taohuadao` || dir != "school10_taohuadao" {
		t.Fatalf("single candidate = %q dir=%q", config, dir)
	}
	// 多变体：同名目录优先，无歧义告警。
	if config, _, note = sceneConfig("scene13", []string{"scene13_1", "scene13"}); config != `ini\scene\scene13` || note != "" {
		t.Fatalf("exact-name candidate = %q note=%q", config, note)
	}
	// 多变体且无同名目录：字典序首个 + 告警。
	if config, _, note = sceneConfig("scene99", []string{"scene99_b", "scene99_a"}); config != `ini\scene\scene99_a` || !strings.Contains(note, "多变体") {
		t.Fatalf("ambiguous candidates = %q note=%q", config, note)
	}
	// 服务端无目录：回落 resource 名 + 告警，dir 空。
	if config, dir, note = sceneConfig("scene98", nil); config != `ini\scene\scene98` || dir != "" || !strings.Contains(note, "无场景目录") {
		t.Fatalf("missing dir = %q dir=%q note=%q", config, dir, note)
	}
}

func TestConfigSpelling(t *testing.T) {
	door := doorRoute{ID: "1126", TargetConfig: `ini\scene\school09_YiHuaGong`}
	if got := configSpelling(`ini\scene\school09_yihuagong`, "school09_yihuagong", door); got != `ini\scene\school09_YiHuaGong` {
		t.Fatalf("door spelling = %q", got)
	}
	// 传送门表登记的是别的目录（变体）或场景无目录：不沿用。
	other := doorRoute{TargetConfig: `ini\scene\scene08_2_DaMoGuiCheng`}
	if got := configSpelling(`ini\scene\scene08`, "scene08", other); got != `ini\scene\scene08` {
		t.Fatalf("unrelated door = %q", got)
	}
	if got := configSpelling(`ini\scene\x`, "", door); got != `ini\scene\x` {
		t.Fatalf("no dir = %q", got)
	}
}

func TestSingleSlashes(t *testing.T) {
	if got := singleSlashes(`ini\\scene\\city02_SuZhou`); got != `ini\scene\city02_SuZhou` {
		t.Fatalf("singleSlashes = %q", got)
	}
	if got := singleSlashes(`ini\scene\x`); got != `ini\scene\x` {
		t.Fatalf("already single = %q", got)
	}
}

func TestLoadNames(t *testing.T) {
	dir := t.TempDir()
	// 个别文案表是 GBK（键内大小写混杂）。
	gbk, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte("Scene_School24=昆仑派\r\nui_HomePointschool24A=昆仑山门\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"interface.idres": gbk,
		// 场景配置目录键在 stringname.idres（UTF-8 BOM）。
		"stringname.idres": []byte("\ufeffini\\scene\\School21_ShenJiYing=神机营\nini\\scene\\scene36=星垂城\nItem_x=物品\n"),
		"notes.txt":        []byte("scene_other=非文案表\n"),
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	names, err := loadNames(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"scene_school24":                        "昆仑派",
		"ui_homepointschool24a":                 "昆仑山门",
		configKeyPrefix + "school21_shenjiying": "神机营",
		configKeyPrefix + "scene36":             "星垂城",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
}

func TestSceneName(t *testing.T) {
	names := map[string]string{
		"scene_school24":                        "昆仑派",
		configKeyPrefix + "school21_shenjiying": "神机营",
		configKeyPrefix + "scene36":             "星垂城",
	}
	cases := []struct {
		resource, config, want string
	}{
		{"school24", `ini\scene\school24_kunlun`, "昆仑派"},     // 资源键在：不读目录键
		{"school21", `ini\scene\school21_shenjiying`, "神机营"}, // 只有目录键
		{"school21", `ini\scene\School21_ShenJiYing`, "神机营"}, // 目录键不分大小写
		{"scene36", `ini\scene\scene36`, "星垂城"},
		{"school99", `ini\scene\nowhere`, ""}, // 两处都无
	}
	for _, c := range cases {
		if got := sceneName(c.resource, c.config, names); got != c.want {
			t.Errorf("sceneName(%q, %q) = %q, want %q", c.resource, c.config, got, c.want)
		}
	}
	// 资源键首见为空值：落到目录键。
	names["scene_school21"] = ""
	if got := sceneName("school21", `ini\scene\school21_shenjiying`, names); got != "神机营" {
		t.Fatalf("empty resource key must fall through to config key, got %q", got)
	}
}

func TestSortResources(t *testing.T) {
	got := []string{"scene02", "school24", "city05", "born01", "school09", "school10", "scene10", "scene09"}
	want := []string{"city05", "born01", "school09", "school10", "school24", "scene02", "scene09", "scene10"}
	sortResources(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	// 编号按数值序（school9 < school10），非字典序。
	if !resourceLess("school9", "school10") || resourceLess("school10", "school9") {
		t.Fatal("numeric ordering")
	}
}

func TestFallbackWaypointsDoor(t *testing.T) {
	// 无驿站场景：传送门目标优先于摆放档。
	door := doorRoute{ID: "6", TargetConfig: `ini\scene\scene25`, TargetX: 671.5, TargetY: 50.7, TargetZ: 675.2, TargetOrient: 2.8}
	got := fallbackWaypoints("scene25", door, "scene25", "testdata-does-not-matter")
	if len(got) != 1 || got[0].wp.Kind != kindDoor || got[0].wp.Name != "door:6" || got[0].wp.Z != 675.2 || !got[0].oriented {
		t.Fatalf("door fallback = %+v", got)
	}
}

func TestFallbackWaypointsNPC(t *testing.T) {
	data := t.TempDir()
	dir := filepath.Join(data, "ini", "Scene", "scene12")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	placements := "<Object><Property File=\"funcnpc.xml\" x=\"461.15\" y=\"10.35\" z=\"2242.20\" id=\"Event_xjh_kg_01\" desc=\"考古点\"/></Object>"
	if err := os.WriteFile(filepath.Join(dir, "npc_creator.xml"), []byte(placements), 0o644); err != nil {
		t.Fatal(err)
	}
	got := fallbackWaypoints("scene12", doorRoute{}, "scene12", data)
	if len(got) != 1 || got[0].wp.Kind != kindNPC || got[0].wp.Title != "考古点" {
		t.Fatalf("npc fallback = %+v", got)
	}
	// NPC 落点 X 偏 2m。
	if got[0].wp.X != 463.15 || got[0].wp.Y != 10.35 || got[0].wp.Z != 2242.2 {
		t.Fatalf("npc landing = %+v", got[0].wp)
	}
	// dir 空（服务端无场景目录）不读盘、不产出。
	if got := fallbackWaypoints("scene12", doorRoute{}, "", data); got != nil {
		t.Fatalf("no dir fallback = %+v", got)
	}
}
