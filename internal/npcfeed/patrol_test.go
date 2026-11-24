package npcfeed

// patrol_test.go 巡逻数据域测试：路线三格式解析（含导航 id 列跳过）、
// offline 排除、去重/高度过滤、时间表采样周期性、BindPatrol 绑定面与
// 速度钳制；短名边界匹配、缓存键带目录、解析统计；真档集成走仓根
// testdata/share/creator/mappath（born03）。

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/mpq/res"
)

func writeRouteFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestLoadPatrolRoutesThreeFormats 三种 r= 行格式：前导空列 / 整数
// 导航 id 列（须跳过——误读即坐标错位）/ 无 id 列。
func TestLoadPatrolRoutesThreeFormats(t *testing.T) {
	ResetPatrolCache()
	dir := t.TempDir()
	writeRouteFile(t, dir, "city01_npc.ini", `[AI_a]
r=,10,1,20,0,200,stand,nil,walk,nil,1,
r=,30,1,40,1.5,0,stand,nil,walk,nil,1,
[AI_b]
r=59,100,2,200,0,5000,stand,nil,walk,nil,1,
r=31,110,2,210,0,0,stand,nil,run,nil,1,
[AI_c]
r=318.19,3.79,632.01,0,4000,stand,nil,walk,nil,1,
r=320.5,3.79,634.5,0,0,stand,nil,walk,nil,1,
`)
	routes := LoadPatrolRoutes(dir, "city01_suzhou")
	if len(routes) != 3 {
		t.Fatalf("路线数 %d ≠ 3", len(routes))
	}
	a := routes["AI_a"]
	if a.Waypoints[0].X != 10 || a.Waypoints[0].Z != 20 || a.Waypoints[0].DwellMs != 200 {
		t.Fatalf("格式1 解析错位: %+v", a.Waypoints[0])
	}
	b := routes["AI_b"]
	if b.Waypoints[0].X != 100 || b.Waypoints[0].Z != 200 || b.Waypoints[0].DwellMs != 5000 {
		t.Fatalf("格式2（导航 id 列）解析错位: %+v", b.Waypoints[0])
	}
	if !b.Waypoints[1].Run {
		t.Fatal("格式2 col8=run 未识别")
	}
	c := routes["AI_c"]
	if c.Waypoints[0].X != 318.19 || c.Waypoints[0].DwellMs != 4000 {
		t.Fatalf("格式3（无 id 列）解析错位: %+v", c.Waypoints[0])
	}
}

// TestLoadPatrolRoutesOfflineExcluded offline 路线与它场景前缀不装载。
func TestLoadPatrolRoutesOfflineExcluded(t *testing.T) {
	ResetPatrolCache()
	dir := t.TempDir()
	writeRouteFile(t, dir, "city01_npc.ini", `[AI_a]
r=,10,1,20,0,0,stand,nil,walk,nil,1,
r=,30,1,40,0,0,stand,nil,walk,nil,1,
`)
	writeRouteFile(t, dir, "city01_npc_offline.ini", `[AI_off]
r=,1,1,2,0,0,stand,nil,walk,nil,1,
r=,3,1,4,0,0,stand,nil,walk,nil,1,
`)
	writeRouteFile(t, dir, "city02_npc.ini", `[AI_other]
r=,5,1,6,0,0,stand,nil,walk,nil,1,
r=,7,1,8,0,0,stand,nil,walk,nil,1,
`)
	routes := LoadPatrolRoutes(dir, "city01_suzhou")
	if _, exists := routes["AI_off"]; exists {
		t.Fatal("offline 路线不应装载")
	}
	if _, exists := routes["AI_other"]; exists {
		t.Fatal("他场景前缀不应装载")
	}
	if _, exists := routes["AI_a"]; !exists {
		t.Fatal("本场景路线缺席")
	}
}

// TestLoadPatrolRoutesDedupeAndShort 相邻去重（<0.5m）后不足 2 点不成
// 路线。
func TestLoadPatrolRoutesDedupeAndShort(t *testing.T) {
	ResetPatrolCache()
	dir := t.TempDir()
	writeRouteFile(t, dir, "city01_npc.ini", `[AI_dup]
r=,10,1,20,0,0,stand,nil,walk,nil,1,
r=,10.2,1,20.1,0,0,stand,nil,walk,nil,1,
r=,30,1,40,0,0,stand,nil,walk,nil,1,
[AI_short]
r=,10,1,20,0,0,stand,nil,walk,nil,1,
`)
	routes := LoadPatrolRoutes(dir, "city01_suzhou")
	if got := len(routes["AI_dup"].Waypoints); got != 2 {
		t.Fatalf("去重后路点 %d ≠ 2", got)
	}
	if _, exists := routes["AI_short"]; exists {
		t.Fatal("单点节不应成路线")
	}
}

// TestLoadPatrolRoutesPrefixBoundary 短名在 '_' 边界处匹配：adv040 不得
// 吞进邻场景的 adv040a_lead.ini；与短名同名的档也收；空短名不匹配任何档。
func TestLoadPatrolRoutesPrefixBoundary(t *testing.T) {
	ResetPatrolCache()
	dir := t.TempDir()
	body := "[AI_x]\nr=,10,1,20,0,0,stand,nil,walk,nil,1,\nr=,30,1,40,0,0,stand,nil,walk,nil,1,\n"
	writeRouteFile(t, dir, "adv040_6nei.ini", strings.Replace(body, "AI_x", "AI_in", 1))
	writeRouteFile(t, dir, "adv040a_lead.ini", strings.Replace(body, "AI_x", "AI_neighbor", 1))
	writeRouteFile(t, dir, "city09.ini", strings.Replace(body, "AI_x", "AI_exact", 1))

	routes := LoadPatrolRoutes(dir, "adv040")
	if routes["AI_in"] == nil {
		t.Fatal("本场景路线缺席")
	}
	if _, exists := routes["AI_neighbor"]; exists {
		t.Fatal("邻场景路线被前缀匹配吞入（adv040 不应收 adv040a_lead.ini）")
	}
	if routes = LoadPatrolRoutes(dir, "adv040a"); routes["AI_neighbor"] == nil {
		t.Fatal("adv040a 自身的路线缺席")
	}
	if routes = LoadPatrolRoutes(dir, "city09_harbor"); routes["AI_exact"] == nil {
		t.Fatal("与短名同名的档未装载")
	}
	if routes = LoadPatrolRoutes(dir, "_noname"); len(routes) != 0 {
		t.Fatalf("空短名不应匹配任何档，得 %d 条", len(routes))
	}
}

// TestLoadPatrolRoutesCacheKeyedByDirectory 缓存键带路线目录：同一场景
// 目录换路线目录（env NINEYIN_MAPPATH_DIR / 测试切换数据根）不得命中
// 另一目录的旧表。
func TestLoadPatrolRoutesCacheKeyedByDirectory(t *testing.T) {
	ResetPatrolCache()
	a, b := t.TempDir(), t.TempDir()
	body := "[AI_x]\nr=,10,1,20,0,0,stand,nil,walk,nil,1,\nr=,30,1,40,0,0,stand,nil,walk,nil,1,\n"
	writeRouteFile(t, a, "city01_npc.ini", strings.Replace(body, "AI_x", "AI_a", 1))
	writeRouteFile(t, b, "city01_npc.ini", strings.Replace(body, "AI_x", "AI_b", 1))

	if routes := LoadPatrolRoutes(a, "city01_yanjing"); routes["AI_a"] == nil {
		t.Fatal("目录 a 的路线缺席")
	}
	routes := LoadPatrolRoutes(b, "city01_yanjing")
	if routes["AI_b"] == nil {
		t.Fatal("目录 b 的路线缺席（命中了目录 a 的缓存）")
	}
	if _, exists := routes["AI_a"]; exists {
		t.Fatal("目录 b 的表混入了目录 a 的路线")
	}
}

// TestParseRouteTableStats 解析统计与服务端读数同源（构建器审计用）：
// 成路线/点数不足/坏行各计其数，路线带 PathId。
func TestParseRouteTableStats(t *testing.T) {
	text := "[AI_ok]\r\n" +
		"r=,10,1,20,0,0,stand,nil,walk,nil,1,\r\n" +
		"r=,10.2,1,20.1,0,0,stand,nil,walk,nil,1,\r\n" + // 与上点 <0.5m，去重
		"r=,30,1,40,1.5,500,stand,nil,run,nil,1,\r\n" +
		"[AI_single]\r\nr=,10,1,20,0,0,stand,nil,walk,nil,1,\r\n" +
		"[AI_bad]\r\nr=,x,1,20,0,0,stand,nil,walk,nil,1,\r\nr=,30,1,40,0,0,stand,nil,walk,nil,1,\r\n"
	routes, stats := ParseRouteTable([]byte(text))
	want := RouteTableStats{Sections: 3, Lines: 6, Rejected: 1, TooShort: 2, Routes: 1, Waypoints: 2}
	if stats != want {
		t.Fatalf("stats=%+v，应为 %+v", stats, want)
	}
	route := routes["AI_ok"]
	if route == nil || route.PathID != "AI_ok" {
		t.Fatalf("AI_ok 未成路线或缺 PathId: %+v", route)
	}
	last := route.Waypoints[len(route.Waypoints)-1]
	if !last.Run || last.DwellMs != 500 || last.X != 30 {
		t.Fatalf("尾点解析错位: %+v", last)
	}
	for _, dropped := range []string{"AI_single", "AI_bad"} {
		if routes[dropped] != nil {
			t.Fatalf("%s 不应成路线", dropped)
		}
	}
}

// TestBuildScheduleSampling 时间表：周期性（t 与 t+cycle 同位置）、行程
// 中位移推进、非法入参返 nil。
func TestBuildScheduleSampling(t *testing.T) {
	route := &Route{PathID: "t", Waypoints: []Waypoint{
		{X: 0, Y: 0, Z: 0}, {X: 100, Y: 0, Z: 0, DwellMs: 1000},
	}}
	s := BuildSchedule(route, 10, 20, 0)
	if s == nil {
		t.Fatal("时间表构建失败")
	}
	// 周期 = 去程 100m/10mps + 停留 1s + 回程 10s + w0 停留 0 = 21s。
	if s.cycleMs != 21_000 {
		t.Fatalf("周期 %d ≠ 21000", s.cycleMs)
	}
	p0 := s.PositionAt(2_000)  // 去程中
	p1 := s.PositionAt(23_000) // 下一周期同相位
	if math.Abs(float64(p0.X-p1.X)) > 0.01 {
		t.Fatalf("周期性破坏: %v vs %v", p0.X, p1.X)
	}
	if p0.X < 15 || p0.X > 25 {
		t.Fatalf("去程 2s 采样 X=%v（应约 20）", p0.X)
	}
	if s.MotionAt(2_000).Speed != 10 || !s.MotionAt(2_000).Moving {
		t.Fatalf("行进动作异常: %+v", s.MotionAt(2_000))
	}
	m := s.MotionAt(10_500) // 停留中
	if m.Moving || m.Speed != 0 {
		t.Fatalf("停留动作异常: %+v", m)
	}
	if BuildSchedule(route, 0, 20, 0) != nil {
		t.Fatal("零速应返 nil")
	}
	if BuildSchedule(&Route{PathID: "s", Waypoints: []Waypoint{{X: 1}}}, 10, 20, 0) != nil {
		t.Fatal("单点路线应返 nil")
	}
}

// TestBuildScheduleDYFilter 高度差 >6m 路点剔除。
func TestBuildScheduleDYFilter(t *testing.T) {
	route := &Route{PathID: "dy", Waypoints: []Waypoint{
		{X: 0, Y: 0, Z: 0}, {X: 10, Y: 50, Z: 0}, {X: 20, Y: 0, Z: 0},
	}}
	s := BuildSchedule(route, 10, 10, 0)
	if s == nil || len(s.points) != 2 {
		t.Fatalf("飞天点未剔除: %v", s)
	}
}

// TestBindPatrolFaces 绑定面：PathId 命中吸附、模板 PathID 缺省、过远
// 站立、敌对脚本排除、速度取模板值。
func TestBindPatrolFaces(t *testing.T) {
	routes := map[string]*Route{
		"AI_x": {PathID: "AI_x", Waypoints: []Waypoint{
			{X: 100, Y: 0, Z: 100}, {X: 130, Y: 0, Z: 100, DwellMs: 500},
		}},
	}
	tpl := func(extra map[string]string) map[string]string {
		m := map[string]string{"ID": "t", "Resource": "npc\\x"}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	spawns := []Spawn{
		{ObjectID: 1, X: 102, Y: 0, Z: 101, PathID: "AI_x", Template: tpl(map[string]string{"WalkSpeed": "2", "RunSpeed": "6"})},
		{ObjectID: 2, X: 100, Y: 0, Z: 100, Template: tpl(map[string]string{"PathID": "AI_x"})},      // 模板缺省
		{ObjectID: 3, X: 500, Y: 0, Z: 500, PathID: "AI_x", Template: tpl(nil)},                      // 过远
		{ObjectID: 4, X: 100, Y: 0, Z: 100, PathID: "nope", Template: tpl(nil)},                      // 路线缺席
		{ObjectID: 5, X: 100, Y: 0, Z: 100, PathID: "AI_x", Script: "AttackNPC", Template: tpl(nil)}, // 敌对
	}
	stats := BindPatrol(spawns, routes, 0)
	if stats.Walkers != 2 {
		t.Fatalf("walkers=%d ≠ 2（stand=%d missing=%d far=%d）", stats.Walkers, stats.Stand, stats.MissingPath, stats.FarOrShort)
	}
	if spawns[0].Schedule == nil || spawns[1].Schedule == nil {
		t.Fatal("前两条应绑定时间表")
	}
	if spawns[2].Schedule != nil || spawns[3].Schedule != nil || spawns[4].Schedule != nil {
		t.Fatal("过远/缺席/敌对不应绑定")
	}
	// 吸附：出生点被改写到最近路点。
	if spawns[0].X != 100 || spawns[0].Z != 100 {
		t.Fatalf("吸附失败: (%v,%v)", spawns[0].X, spawns[0].Z)
	}
	// 速度：模板值传入时间表（行程 = 30m / 2mps = 15s）。
	if got := spawns[0].Schedule.travels[0]; got != 15_000 {
		t.Fatalf("walk 腿行程 %d ≠ 15000", got)
	}
	// 速度钳制上限 15：30m 腿行程 = 30/15 = 2s。
	hot := []Spawn{{ObjectID: 9, X: 100, Y: 0, Z: 100, PathID: "AI_x",
		Template: tpl(map[string]string{"WalkSpeed": "999", "RunSpeed": "999"})}}
	if stats2 := BindPatrol(hot, routes, 0); stats2.Walkers != 1 {
		t.Fatalf("钳制速度绑定失败: %+v", stats2)
	}
	if got := hot[0].Schedule.travels[0]; got != 2_000 {
		t.Fatalf("钳制后行程 %d ≠ 2000", got)
	}
}

// TestLoadPatrolRoutesBorn03 真档集成：born03 路线表装载 + 全目录绑定
// （walkers>0——test 环境无真档时跳过）。
func TestLoadPatrolRoutesBorn03(t *testing.T) {
	root := ResolveRoot()
	if root == "" {
		t.Skip("数据根缺席（真档不随仓）")
	}
	mappathDir := filepath.Join(root, "share", "creator", "mappath")
	if _, err := res.Stat(mappathDir); err != nil {
		t.Skipf("share/creator/mappath 缺席: %v", err)
	}
	ResetPatrolCache()
	routes := LoadPatrolRoutes(mappathDir, "born03_yanyuzhuang")
	if len(routes) == 0 {
		t.Fatal("born03 路线表为空（testdata/share/creator/mappath 缺席？）")
	}
	spawns, _, err := BuildCatalog(root, "born03")
	if err != nil {
		t.Skipf("目录装载失败（真档缺席）: %v", err)
	}
	stats := BindPatrol(spawns, routes, time.Now().UnixMilli())
	if stats.Walkers == 0 {
		t.Fatalf("born03 绑定 0 命中: %+v", stats)
	}
	// 采样健壮性：全 walker 周期内无 NaN/爆表。
	bad := 0
	for i := range spawns {
		if spawns[i].Schedule == nil {
			continue
		}
		for offset := int64(0); offset < 30_000; offset += 1_000 {
			p := spawns[i].Schedule.PositionAt(offset)
			if math.IsNaN(float64(p.X)) || math.Abs(float64(p.X)) > 10000 || math.Abs(float64(p.Z)) > 10000 {
				bad++
				break
			}
		}
	}
	if bad > 0 {
		t.Fatalf("采样爆表/NaN %d 条", bad)
	}
	t.Logf("born03: 路线 %d 条，walkers=%d stand=%d missing=%d far=%d",
		len(routes), stats.Walkers, stats.Stand, stats.MissingPath, stats.FarOrShort)
}
