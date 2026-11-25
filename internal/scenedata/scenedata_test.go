package scenedata

// 内嵌表单测：产物结构锚点 + 与仓库数据根（ini/Scene 目录）的一致性。
// 产物是否与源数据同步由 tools/gen_scene_waypoints -verify 人工把关（源 =
// 本机客户端解包树，测试环境未必有）。

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
	"github.com/lizhanyin/9yin-server-go/internal/npcfeed"
)

func TestTableAnchors(t *testing.T) {
	scenes := Scenes()
	if len(scenes) < 60 {
		t.Fatalf("scenes = %d, want the full client world-scene set", len(scenes))
	}
	want := map[string]struct {
		config  string
		name    string
		landing Waypoint
	}{
		// BornCJ 场景：权威目录 = 八派正图，非同前缀变体。
		"school01": {config: `ini\scene\school01_JinYiWei`, name: "锦衣卫"},
		"city01":   {config: `ini\scene\city01_YanJing`, name: "燕京"},
		// 移花宫：默认落点 = 安全驿站 HomePointschool09A，坐标取同址复活点行
		// （relive_info.ini force_yihua 同值，场景号 281）。
		"school09": {
			config: `ini\scene\school09_YiHuaGong`, name: "移花宫",
			landing: Waypoint{Name: "HomePointschool09A", Title: "移花宫眠楼", X: 340.16, Y: 47.979, Z: 657.284, Orient: 2.588, Kind: KindHome, Safe: true, Source: "school09_01"},
		},
		// 此前 GM 列表完全没有的场景。
		"school24": {config: `ini\scene\school24_kunlun`, name: "昆仑派"},
		"school21": {config: `ini\scene\school21_shenjiying`, name: "神机营"},
		"scene11":  {config: `ini\scene\scene11_kuaihuodao`, name: "快活岛"},
		"scene36":  {config: `ini\scene\scene36`, name: "星垂城"},
		"scene38":  {config: `ini\scene\scene38`, name: "金乌岛"},
	}
	for resource, w := range want {
		sc, ok := Find(resource)
		if !ok {
			t.Fatalf("%s missing from table", resource)
		}
		if sc.Config != w.config {
			t.Errorf("%s config = %q, want %q", resource, sc.Config, w.config)
		}
		if sc.Name != w.name {
			t.Errorf("%s name = %q, want %q", resource, sc.Name, w.name)
		}
		if w.landing.Name != "" && sc.Landing != w.landing {
			t.Errorf("%s landing = %+v, want %+v", resource, sc.Landing, w.landing)
		}
	}
	// 无驿站场景：落点取摆放档功能 NPC（金乌岛 = 游方药师）。
	if sc, _ := Find("scene38"); sc.Landing.Kind != KindNPC || !strings.HasPrefix(sc.Landing.Name, "npc:Relive") {
		t.Errorf("scene38 landing = %+v, want revive pharmacist NPC", sc.Landing)
	}
}

func TestFind(t *testing.T) {
	if sc, ok := Find("  SCHOOL10 "); !ok || sc.Resource != "school10" {
		t.Fatalf("case-insensitive find = %+v ok=%v", sc, ok)
	}
	if _, ok := Find("school16"); ok { // 客户端无地形
		t.Fatal("school16 must not resolve")
	}
	if _, ok := Find(""); ok {
		t.Fatal("empty resource must not resolve")
	}
	if sc, ok := FindByName("移花宫"); !ok || sc.Resource != "school09" {
		t.Fatalf("FindByName 移花宫 = %+v ok=%v", sc, ok)
	}
	// 重名不收录：凌霄城（scene06 / scene25）、念萝坝（school15 / scene05）、
	// 星垂城（scene35 / scene36）。
	for _, name := range []string{"凌霄城", "念萝坝", "星垂城"} {
		if _, ok := FindByName(name); ok {
			t.Errorf("FindByName(%q) must be ambiguous", name)
		}
	}
	if _, ok := FindByName("不存在场景"); ok {
		t.Fatal("unknown name must not resolve")
	}
}

// TestAllScenesNamed 全表有中文名守卫：面板下拉 / `gm scene` 清单不回退资源名。
// 新场景无名 = 生成器没在文案表找到键（资源键 scene_<resource> 或场景配置
// 目录键 ini\scene\<目录名>）——重跑 gen_scene_waypoints 看它的告警。
func TestAllScenesNamed(t *testing.T) {
	for _, sc := range Scenes() {
		if sc.Name == "" {
			t.Errorf("%s has no Chinese name（面板显示回退资源名）", sc.Resource)
		}
	}
}

func TestSceneOrder(t *testing.T) {
	rank := map[string]int{CategoryCity: 0, CategoryBorn: 1, CategorySchool: 2, CategoryScene: 3}
	scenes := Scenes()
	for i, sc := range scenes {
		if i == 0 {
			continue
		}
		prev := scenes[i-1]
		if rank[sc.Category] < rank[prev.Category] {
			t.Fatalf("category order broken at %d: %s after %s", i, sc.Resource, prev.Resource)
		}
		if sc.Category != prev.Category {
			continue
		}
		// 同类内编号严格递增（数值序，非字典序）。
		if resourceNumber(sc.Resource) <= resourceNumber(prev.Resource) {
			t.Fatalf("numeric order broken at %d: %s after %s", i, sc.Resource, prev.Resource)
		}
	}
	pos := map[string]int{}
	for i, sc := range scenes {
		pos[sc.Resource] = i
	}
	// 字典序会把 school10 排在 school09 前——数值序才是面板想要的。
	if !(pos["school09"] < pos["school10"]) || !(pos["scene09"] < pos["scene10"]) {
		t.Fatal("numeric ordering broken")
	}
}

// resourceNumber 资源名里的编号数值（school09 → 9）。
func resourceNumber(resource string) int {
	n := 0
	for _, c := range resource {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func TestConfigsExistInDataRoot(t *testing.T) {
	root := npcfeed.ResolveRoot()
	if root == "" {
		t.Skip("数据根缺席（Res / testdata）")
	}
	entries, err := res.ReadDir(filepath.Join(root, "ini", "Scene"))
	if err != nil {
		t.Skipf("ini/Scene: %v", err)
	}
	dirs := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			dirs[strings.ToLower(e.Name())] = true
		}
	}
	for _, sc := range Scenes() {
		seg := sc.Config
		if i := strings.LastIndexAny(seg, `\/`); i >= 0 {
			seg = seg[i+1:]
		}
		if !dirs[strings.ToLower(seg)] {
			t.Errorf("%s config %q: 目录不在 %s ini/Scene", sc.Resource, sc.Config, root)
		}
	}
}

func TestWaypointsUniquePerScene(t *testing.T) {
	for _, sc := range Scenes() {
		seen := map[string]bool{}
		for _, wp := range sc.Waypoints {
			if wp.Name == "" {
				t.Errorf("%s has unnamed waypoint", sc.Resource)
				continue
			}
			if seen[wp.Name] {
				t.Errorf("%s duplicate waypoint %q", sc.Resource, wp.Name)
			}
			seen[wp.Name] = true
		}
		if len(sc.Waypoints) == 0 {
			t.Errorf("%s has no waypoint", sc.Resource)
		}
	}
}

func TestCategories(t *testing.T) {
	cats := Categories()
	if len(cats) != 4 || cats[0] != [2]string{CategoryCity, "主城"} || cats[3] != [2]string{CategoryScene, "江湖"} {
		t.Fatalf("categories = %v", cats)
	}
	if CategoryName("nope") != "nope" {
		t.Fatal("unknown category name fallback")
	}
}
