package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// gmGBK 把 UTF-8 样本编成 GBK，模拟资源树里的真实 ini 文件
// （gmScanINI 装载路径先做 GBK 解码）。
func gmGBK(t *testing.T, text string) []byte {
	t.Helper()
	data, _, err := transform.Bytes(simplifiedchinese.GBK.NewEncoder(), []byte(text))
	if err != nil {
		t.Fatalf("encode GBK sample: %v", err)
	}
	return data
}

func TestGMParseCloneRows(t *testing.T) {
	sample := `
//注释节不进目录
[//又一个注释]
[ini\scene\clone003]
Desc=暮色之村
EasyPowerLevel=11
DifficultPowerLevel=200
HellPowerLevel=26

[ini\scene\clone003_yx]
Desc=暮色之村
EasyPowerLevel=70

[ini\scene\clone777]
Desc=无难度键的畸形行
`
	rows := parseGMCloneRows(gmGBK(t, sample))
	if len(rows) != 3 {
		t.Fatalf("rows=%d want 3: %+v", len(rows), rows)
	}
	first := rows[0]
	if first.Config != `ini\scene\clone003` || first.Description != "暮色之村" {
		t.Fatalf("first=%+v", first)
	}
	if len(first.Difficulties) != 3 || first.Difficulties[0] != 1 || first.Difficulties[2] != 3 {
		t.Fatalf("first difficulties=%v want [1 2 3]", first.Difficulties)
	}
	hero := rows[1]
	if hero.Config != `ini\scene\clone003_yx` || len(hero.Difficulties) != 1 || hero.Difficulties[0] != 1 {
		t.Fatalf("hero variant=%+v", hero)
	}
	if hero.Description != "暮色之村" {
		t.Fatalf("hero description=%q (GBK 解码失败?)", hero.Description)
	}
	odd := rows[2]
	if len(odd.Difficulties) != 1 || odd.Difficulties[0] != 1 {
		t.Fatalf("odd row difficulties=%v want [1] 兜底", odd.Difficulties)
	}
	if rows := parseGMCloneRows(nil); len(rows) != 0 {
		t.Fatalf("empty data rows=%d want 0", len(rows))
	}
}

func TestGMParseSceneRows(t *testing.T) {
	sample := `
[17]
Config=ini\scene\city04_LuoYang

[3]
Config=ini\scene\school09_YiHuaGong

[3]
Config=ini\scene\repeat_Kept_First
`
	rows := parseGMSceneRows(gmGBK(t, sample))
	if len(rows) != 2 {
		t.Fatalf("rows=%d want 2 (重复 id 首见优先): %+v", len(rows), rows)
	}
	if rows[0].ID != 3 || rows[0].Config != `ini\scene\school09_YiHuaGong` || rows[0].Resource != "school09_YiHuaGong" {
		t.Fatalf("rows[0]=%+v", rows[0])
	}
	if rows[1].ID != 17 || rows[1].Resource != "city04_LuoYang" {
		t.Fatalf("rows[1]=%+v（id 升序）", rows[1])
	}
}

func TestGMParseTitleRows(t *testing.T) {
	sample := `
[1003]
TitleType=1

[15]
TitleType=2

[not_a_number]
TitleType=1

[2000]
TitleType=3
NotTitle=ignore
`
	rows := parseGMTitleRows(gmGBK(t, sample))
	if len(rows) != 3 {
		t.Fatalf("rows=%d want 3: %+v", len(rows), rows)
	}
	if rows[0].ID != 15 || rows[0].Type != 2 || rows[2].ID != 2000 || rows[2].Type != 3 {
		t.Fatalf("rows=%+v", rows)
	}
}

func TestGMParseMountRows(t *testing.T) {
	sample := `
[MOUNT_6_003]
Des=1
name=ui_MOUNT_6_003
type=cy

[mount_2_001qd]
name=ui_mount_2_001qd

[orphan_mount]
name=ui_orphan_mount
`
	// 物品目录：MOUNT_6_003 有完整行（名字走 idres 直键）；mount_2_001qd
	// 缺席（验证目录缺失条目保留、名链回退）。
	itemsByConfig := map[string]gmItemRow{
		"MOUNT_6_003": {ConfigID: "MOUNT_6_003", Name: "坐骑·河套马", ArtPack: "9507"},
	}
	names := map[string]string{
		"ui_mount_2_001qd": "庸马（ui 键）",
		"orphan_mount":     "孤骑（configID 直键）",
	}
	rows := parseGMMountRows(gmGBK(t, sample), itemsByConfig, names)
	if len(rows) != 3 {
		t.Fatalf("rows=%d want 3: %+v", len(rows), rows)
	}
	if rows[0].ConfigID != "MOUNT_6_003" || rows[0].Name != "坐骑·河套马" || rows[0].ArtPack != "9507" {
		t.Fatalf("rows[0]=%+v（物品目录优先）", rows[0])
	}
	if rows[1].Name != "庸马（ui 键）" {
		t.Fatalf("rows[1]=%+v（ui 键回退）", rows[1])
	}
	if rows[2].Name != "孤骑（configID 直键）" {
		t.Fatalf("rows[2]=%+v（configID 直键回退）", rows[2])
	}
}

func TestGMParseIdleStances(t *testing.T) {
	sample := `
[b_stand_taichi]
Skeleton=role_stand
[b_walk_fast]
Skeleton=role_walk
[b_stand_jian]
skeleton=role_stand_sword
`
	rows := parseGMIdleStances(gmGBK(t, sample))
	if len(rows) != 2 {
		t.Fatalf("rows=%d want 2 (仅 stand 节): %+v", len(rows), rows)
	}
	if rows[0].Action != "b_stand_taichi" || rows[0].Skeleton != "role_stand" {
		t.Fatalf("rows[0]=%+v", rows[0])
	}
	if rows[1].Skeleton != "role_stand_sword" {
		t.Fatalf("rows[1]=%+v（skeleton 键大小写不敏感）", rows[1])
	}
}

func TestGMParseRecastAffixes(t *testing.T) {
	sample := `
[blade_50302]
r=MeleeDamage,13040,13041,,,13042
r=AIPhyDef,13234,13236

[blade_50401]
r=MeleeDamage,13040,13043
`
	result := parseGMRecastAffixes(gmGBK(t, sample))
	if !result.Loaded {
		t.Fatalf("loaded=false want true")
	}
	if len(result.Prop) != 6 {
		t.Fatalf("prop=%d want 6 (13040 跨武器去重): %+v", len(result.Prop), result.Prop)
	}
	first := result.Prop[0]
	if first.ID != 13040 || first.Name != "MeleeDamage" {
		t.Fatalf("first=%+v", first)
	}
	if first.Source != "blade_50302 blade_50401" {
		t.Fatalf("first source=%q want 合并去重", first.Source)
	}
	if len(result.Skill) != 0 {
		t.Fatalf("skill=%d want 0（当前资源无招式词条源）", len(result.Skill))
	}
	if result := parseGMRecastAffixes(gmGBK(t, "[no_data]\nx=1\n")); result.Loaded || len(result.Prop) != 0 {
		t.Fatalf("空表应 loaded=false: %+v", result)
	}
}

// TestGMCatalogDirsEndpoints 面板路由 smoke：此前 404 的 11 个端点全部
// 注册成功，GET 一律 200 + JSON（资源缺失时按约定降级为空数组/loaded
// 状态，不 5xx）；minimal-login 的 POST 取反生效。
func TestGMCatalogDirsEndpoints(t *testing.T) {
	server := gmPanelTestServer(t)
	for _, path := range []string{
		"/api/clones", "/api/scenes", "/api/giftboxes", "/api/titles",
		"/api/mounts", "/api/idle-stances", "/api/recast-affixes",
		"/api/combat-dummy/taolus", "/api/repute/status", "/api/origin/status",
		"/api/minimal-login",
	} {
		body, contentType, status := gmGetBody(t, server.URL+path)
		if status != http.StatusOK {
			t.Fatalf("GET %s status=%d want=200", path, status)
		}
		if !strings.HasPrefix(contentType, "application/json") {
			t.Fatalf("GET %s content-type=%q want application/json", path, contentType)
		}
		var value any
		if err := json.Unmarshal([]byte(body), &value); err != nil {
			t.Fatalf("GET %s body not JSON: %v (%s)", path, err, body)
		}
	}
	// minimal-login：GET→POST 取反→GET 验证→POST 还原。
	body, _, _ := gmGetBody(t, server.URL+"/api/minimal-login")
	state := map[string]bool{}
	if err := json.Unmarshal([]byte(body), &state); err != nil {
		t.Fatalf("minimal-login body=%s: %v", body, err)
	}
	if state["enabled"] {
		// 还原到 false 基线再测翻转。
		response, err := http.Post(server.URL+"/api/minimal-login", "", nil)
		if err != nil {
			t.Fatalf("post minimal-login: %v", err)
		}
		response.Body.Close()
	}
	response, err := http.Post(server.URL+"/api/minimal-login", "", nil)
	if err != nil {
		t.Fatalf("post minimal-login: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("post minimal-login status=%d", response.StatusCode)
	}
	toggled := map[string]bool{}
	if err := json.NewDecoder(response.Body).Decode(&toggled); err != nil || !toggled["enabled"] {
		t.Fatalf("post body=%v err=%v want enabled=true", toggled, err)
	}
	response.Body.Close()
	// 还原全局开关，避免影响其它测试。
	response, err = http.Post(server.URL+"/api/minimal-login", "", nil)
	if err != nil {
		t.Fatalf("post minimal-login restore: %v", err)
	}
	response.Body.Close()
	// 非 GET 的目录端点 405。
	response, err = http.Post(server.URL+"/api/clones", "", nil)
	if err != nil {
		t.Fatalf("post clones: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("post clones status=%d want 405", response.StatusCode)
	}
}

// TestGMCloneAndRecastFromRealResources 资源树在位时的装载冒烟：副本
// 与洗练词条目录取到真数据（无资源环境下自动降级为空，仅打日志跳过）。
func TestGMCloneAndRecastFromRealResources(t *testing.T) {
	clones := loadGMCloneRows()
	if len(clones) == 0 {
		t.Log("cloneinfo.ini 未装载（资源树缺席），跳过真数据断言")
	} else {
		found := false
		for _, row := range clones {
			if row.Config == `ini\scene\clone003` && row.Description != "" && len(row.Difficulties) == 3 {
				found = true
			}
		}
		if !found {
			t.Fatalf("clone003 未按预期装载: %+v", clones[:min(3, len(clones))])
		}
	}
	affixes := loadGMRecastAffixes()
	if affixes.Loaded {
		if len(affixes.Prop) < 10 {
			t.Fatalf("senior_weapon_dire.ini 词条过少: %d", len(affixes.Prop))
		}
		if affixes.Prop[0].Name == "" {
			t.Fatalf("首词条缺属性名: %+v", affixes.Prop[0])
		}
	}
}
