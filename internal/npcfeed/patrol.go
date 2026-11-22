package npcfeed

// patrol.go NPC 巡逻：mappath 路线表装载 + 绝对时间表 + 采样 + 目录绑定。
// 配方移植 2433 线 cmd/protocol-probe/patrol.go（真机验证配方，2026-11-22
// 备份目录 npc-aoi-patrol 再适配 npcfeed.Spawn）：
//
//   - 路线表：mappath/<场景短名>*.ini（GBK），节名=creator PathId，每行
//     r= 一个路点（三种列格式：前导空列 / 整数导航点 id 列 / 无 id 列
//     ——旧服误读 id 成 x 即「上天入地」根因；列序 x,y,z,orient,dwellMs，
//     尾部 col8 walk/run 标记）。
//   - 时间表：路线展开为「行程+停留」绝对时间线（往返一周期），同场景
//     跨玩家一致、重进视口从中断处续走。
//   - 广播语义（fssoloworld.npcPatrolTick 消费）：0x21 ServerAllDest 每秒
//     一拍——行进 spd=WalkSpeed/RunSpeed 且 flags=1（rotate 恒 +2π）；
//     停留 spd=0 且 flags=0；State(183) walk/stand 翻转帧先于 0x21。

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	// patrolSnapMaxMeters 出生点离路线最近路点的吸附上限（米）。
	patrolSnapMaxMeters = float32(35)
	// 速度缺省/钳制（m/s，与 npcconfig 城市表典型值一致）。
	patrolWalkSpeedFallback = float32(1.25)
	patrolRunSpeedFallback  = float32(4)
	patrolSpeedMin          = float32(0.5)
	patrolSpeedMax          = float32(15)
	// patrolDedupeMeters 相邻路点去重阈值（米）。
	patrolDedupeMeters = float32(0.5)
	// patrolDwellCapMillis 停留列上限（毫秒，数据实测 0~60000）。
	patrolDwellCapMillis = int64(120_000)
	// 行程时长钳制（毫秒）：下限防零碎腿刷帧，上限防脏数据长腿挂死。
	patrolTravelMinMillis = int64(300)
	patrolTravelMaxMillis = int64(90_000)
	// patrolMaxLegDYMeters 高度差钳制（米）——防飞天脏数据。
	patrolMaxLegDYMeters = float32(6)
)

// Waypoint 单个路点；Run=true 跑 RunSpeed；DwellMs=到达本点后的停留。
type Waypoint struct {
	X, Y, Z float32
	Orient  float32
	Run     bool
	DwellMs int64
}

// hasFacing 数据朝向哨兵（0=未设、-10=按行进方向）：非零且 |o|≤2π 才有效。
func (w Waypoint) hasFacing() bool {
	return w.Orient != 0 && math.Abs(float64(w.Orient)) <= math.Pi*2
}

// Route mappath 巡逻路线（不可变，场景缓存内共享）：节名=PathId。
type Route struct {
	PathID    string
	Waypoints []Waypoint
}

// Schedule 巡逻时间表：往返一周期的绝对时间线，任意时刻采样当前位置
// 与 0x21 广播动作。epochMs 为绑定时刻（场景目录缓存加载），同场景不同
// 玩家共享同一份 → NPC 位置跨连接一致、服务器重启前续走。
type Schedule struct {
	pathID    string
	epochMs   int64
	cycleMs   int64
	points    []Waypoint
	starts    []int64
	travels   []int64
	dwells    []int64
	fromIdx   []int
	toIdx     []int
	walkSpeed float32
	runSpeed  float32
}

// Position 采样位置（行程中按比例插值，停留中即路点坐标）。
type Position struct {
	X, Y, Z, Orient float32
}

// Motion 0x21 广播动作：段目标坐标/朝向、当前速度（行进=该腿速度，
// 停留=0）、移动标志（官方每秒一拍全量广播，含停留中的实体）。
type Motion struct {
	DestX, DestY, DestZ, DestOrient float32
	Speed                           float32
	Moving                          bool
}

// PositionAt 时刻 nowMs 的当前位置（行程中插值；朝向=目标朝向或行进方向）。
func (s *Schedule) PositionAt(nowMs int64) Position {
	k := s.segmentAt(nowMs)
	a, b := s.points[s.fromIdx[k]], s.points[s.toIdx[k]]
	offset := s.offsetOf(nowMs)
	travelEnd := s.starts[k] + s.travels[k]
	if offset >= travelEnd {
		return Position{X: b.X, Y: b.Y, Z: b.Z, Orient: patrolFacingOf(a, b)}
	}
	frac := float32(offset-s.starts[k]) / float32(s.travels[k])
	return Position{
		X:      a.X + (b.X-a.X)*frac,
		Y:      a.Y + (b.Y-a.Y)*frac,
		Z:      a.Z + (b.Z-a.Z)*frac,
		Orient: patrolFacingOf(a, b),
	}
}

// MotionAt 时刻 nowMs 的 0x21 广播动作：目的地=段末路点，速度=该腿速度
// （run 腿取 runSpeed），停留中速度 0。
func (s *Schedule) MotionAt(nowMs int64) Motion {
	k := s.segmentAt(nowMs)
	a, b := s.points[s.fromIdx[k]], s.points[s.toIdx[k]]
	offset := s.offsetOf(nowMs)
	moving := offset < s.starts[k]+s.travels[k]
	speed := float32(0)
	if moving {
		speed = s.walkSpeed
		if b.Run {
			speed = s.runSpeed
		}
	}
	return Motion{DestX: b.X, DestY: b.Y, DestZ: b.Z, DestOrient: patrolFacingOf(a, b),
		Speed: speed, Moving: moving}
}

// patrolFacingOf 朝向：路点有效朝向优先，否则按段行进方向。
func patrolFacingOf(from, to Waypoint) float32 {
	if to.hasFacing() {
		return to.Orient
	}
	return float32(math.Atan2(float64(to.X-from.X), float64(to.Z-from.Z)))
}

// segmentAt 段区间 [starts[k], starts[k]+travel+dwell)：倒序首命中。
func (s *Schedule) segmentAt(nowMs int64) int {
	offset := s.offsetOf(nowMs)
	for k := len(s.starts) - 1; k >= 0; k-- {
		if offset >= s.starts[k] {
			return k
		}
	}
	return 0
}

func (s *Schedule) offsetOf(nowMs int64) int64 {
	offset := (nowMs - s.epochMs) % s.cycleMs
	if offset < 0 {
		offset += s.cycleMs
	}
	return offset
}

// BuildSchedule 构建时间表（不可用返回 nil：高度过滤后不足 2 点或速度
// 非法）。walkSpeed/runSpeed 为该 NPC 的同值下发属性（m/s——客户端动画
// 据此播放，不同步会到点橡皮筋）。
func BuildSchedule(route *Route, walkSpeed, runSpeed float32, epochMs int64) *Schedule {
	if !patrolFiniteSpeed(walkSpeed) || !patrolFiniteSpeed(runSpeed) || walkSpeed <= 0 || runSpeed <= 0 {
		return nil
	}
	kept := filterWaypointsByDY(route.Waypoints)
	if len(kept) < 2 {
		return nil
	}
	segments := (len(kept) - 1) * 2
	schedule := &Schedule{
		pathID: route.PathID, epochMs: epochMs, points: kept,
		starts:    make([]int64, segments),
		travels:   make([]int64, segments),
		dwells:   make([]int64, segments),
		fromIdx:   make([]int, segments),
		toIdx:     make([]int, segments),
		walkSpeed: walkSpeed,
		runSpeed:  runSpeed,
	}
	cursor := int64(0)
	k := 0
	for i := 0; i+1 < len(kept); i++ { // 去程 w0→…→w(n-1)
		cursor = schedule.appendSegment(i, i+1, cursor, k)
		k++
	}
	for i := len(kept) - 1; i-1 >= 0; i-- { // 回程 w(n-1)→…→w0
		cursor = schedule.appendSegment(i, i-1, cursor, k)
		k++
	}
	schedule.cycleMs = cursor
	return schedule
}

func patrolFiniteSpeed(speed float32) bool {
	return !math.IsNaN(float64(speed)) && !math.IsInf(float64(speed), 0)
}

// filterWaypointsByDY 剔除与前一保留点高度差 >6m 的路点（构建时一次完成，
// 全连接一致）。
func filterWaypointsByDY(waypoints []Waypoint) []Waypoint {
	kept := make([]Waypoint, 0, len(waypoints))
	for _, point := range waypoints {
		if len(kept) > 0 && math.Abs(float64(point.Y-kept[len(kept)-1].Y)) > float64(patrolMaxLegDYMeters) {
			continue
		}
		kept = append(kept, point)
	}
	return kept
}

// appendSegment 追加一段行程+停留（停留归属到达行），返回推进后的游标。
func (s *Schedule) appendSegment(from, to int, cursor int64, k int) int64 {
	a, b := s.points[from], s.points[to]
	dx, dz := b.X-a.X, b.Z-a.Z
	distXz := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	speed := s.walkSpeed
	if b.Run {
		speed = s.runSpeed
	}
	travel := int64(float64(distXz) / float64(speed) * 1000)
	travel = max64(patrolTravelMinMillis, min64(patrolTravelMaxMillis, travel))
	s.starts[k] = cursor
	s.travels[k] = travel
	s.dwells[k] = b.DwellMs
	s.fromIdx[k] = from
	s.toIdx[k] = to
	return cursor + travel + b.DwellMs
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// ---- 路线表装载（进程级缓存） ----

var (
	patrolCacheMu sync.Mutex
	patrolCache   = map[string]map[string]*Route{} // sceneFolder → pathID → route
)

// LoadPatrolRoutes 场景路线表：mappath 目录下 stem 以场景短名（首个 '_'
// 前）开头且不含 offline 的 .ini（GBK）。目录缺失/无匹配 → 空表（该场景
// 无巡逻，不报错）。
func LoadPatrolRoutes(mappathDir, sceneFolder string) map[string]*Route {
	patrolCacheMu.Lock()
	defer patrolCacheMu.Unlock()
	if cached, ok := patrolCache[sceneFolder]; ok {
		return cached
	}
	table := make(map[string]*Route)
	entries, err := os.ReadDir(mappathDir)
	if err != nil {
		patrolCache[sceneFolder] = table
		return table
	}
	prefix := strings.ToLower(patrolSceneShortName(sceneFolder))
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".ini") {
			continue
		}
		stem := strings.ToLower(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		if strings.HasPrefix(stem, prefix) && !strings.Contains(stem, "offline") {
			files = append(files, filepath.Join(mappathDir, entry.Name()))
		}
	}
	sort.Strings(files)
	for _, file := range files {
		loadRouteFile(file, table)
	}
	patrolCache[sceneFolder] = table
	return table
}

// ResetPatrolCache 清路线缓存（测试面）。
func ResetPatrolCache() {
	patrolCacheMu.Lock()
	patrolCache = map[string]map[string]*Route{}
	patrolCacheMu.Unlock()
}

// patrolSceneShortName 场景摆放目录名 → 短名（born03_yanyuzhuang →
// born03；路线文件 born03_npc.ini / born03_mst.ini）。
func patrolSceneShortName(sceneFolder string) string {
	if us := strings.IndexByte(sceneFolder, '_'); us >= 0 {
		return sceneFolder[:us]
	}
	return sceneFolder
}

// loadRouteFile 解析单个 mappath ini：节名=PathId，每行 r= 一个路点。
// GBK 解码（GB18030 超集；失败回落原字节——路线坐标全 ASCII 时无差）。
func loadRouteFile(path string, table map[string]*Route) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(data); err == nil {
		data = decoded
	}
	section := ""
	var points []Waypoint
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flushRouteSection(table, section, points)
			section = line[1 : len(line)-1]
			points = nil
			continue
		}
		eq := strings.IndexByte(line, '=')
		if section == "" || eq <= 0 || (line[0] != 'r' && line[0] != 'R') {
			continue
		}
		if waypoint, ok := parseWaypoint(line[eq+1:]); ok {
			points = append(points, waypoint)
		}
	}
	flushRouteSection(table, section, points)
}

// parseWaypoint r= 载荷三格式（对齐尾部 stand,nil,walk 列）：
//  1. ",<x>,<y>,<z>,…" 前导空列（主流）
//  2. "<导航点id>,<x>,<y>,<z>,…" 首列整数 id 须跳过（旧服误读成 x 即
//     坐标整体错位的根因）
//  3. "<x>,<y>,<z>,…" 无 id 列（少量）
//
// 列序 x,y,z,orient,dwellMs；walk/run 标记在尾部固定列（col8）。
func parseWaypoint(payload string) (Waypoint, bool) {
	parts := strings.Split(payload, ",")
	base := coordinateBase(parts)
	if base < 0 || len(parts) < base+3 {
		return Waypoint{}, false
	}
	parse := func(field string) (float64, bool) {
		value, err := strconv.ParseFloat(strings.TrimSpace(field), 32)
		return value, err == nil
	}
	x, ok := parse(parts[base])
	if !ok {
		return Waypoint{}, false
	}
	y, ok := parse(parts[base+1])
	if !ok {
		return Waypoint{}, false
	}
	z, ok := parse(parts[base+2])
	if !ok {
		return Waypoint{}, false
	}
	orient := float64(0)
	if len(parts) > base+3 && strings.TrimSpace(parts[base+3]) != "" {
		if orient, ok = parse(parts[base+3]); !ok {
			return Waypoint{}, false
		}
	}
	// 停留列：朝向后第一个数值列（A/B 格式 field5、无 id 格式 field4）。
	dwellMs := float64(0)
	if len(parts) > base+4 && strings.TrimSpace(parts[base+4]) != "" {
		if dwellMs, ok = parse(parts[base+4]); !ok {
			return Waypoint{}, false
		}
	}
	dwellMs = math.Max(0, math.Min(float64(patrolDwellCapMillis), dwellMs))
	run := len(parts) > 8 && strings.EqualFold(strings.TrimSpace(parts[8]), "run")
	return Waypoint{X: float32(x), Y: float32(y), Z: float32(z),
		Orient: float32(orient), Run: run, DwellMs: int64(dwellMs)}, true
}

// coordinateBase 坐标起始列：首列空（格式1）或纯整数导航点 id（格式2）
// → x 在第 1 列；首列浮点（格式3，无 id 列）→ 第 0 列；其余 -1。
func coordinateBase(parts []string) int {
	if len(parts) == 0 {
		return -1
	}
	first := strings.TrimSpace(parts[0])
	if first == "" {
		return 1
	}
	if strings.ContainsRune(first, '.') {
		if _, err := strconv.ParseFloat(first, 32); err == nil {
			return 0
		}
		return -1
	}
	if _, err := strconv.ParseInt(first, 10, 64); err == nil {
		return 1
	}
	return -1
}

// flushRouteSection 节收尾：相邻点去重（<0.5m）后 ≥2 点才成路线；同名
// 节首个生效。
func flushRouteSection(table map[string]*Route, section string, points []Waypoint) {
	if section == "" || len(points) < 2 {
		return
	}
	cleaned := make([]Waypoint, 0, len(points))
	for _, point := range points {
		if len(cleaned) > 0 && routeDistanceSq(cleaned[len(cleaned)-1], point) <
			patrolDedupeMeters*patrolDedupeMeters {
			continue
		}
		cleaned = append(cleaned, point)
	}
	if len(cleaned) < 2 {
		return
	}
	if _, exists := table[section]; exists {
		return
	}
	table[section] = &Route{PathID: section, Waypoints: cleaned}
}

func routeDistanceSq(a, b Waypoint) float32 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return dx*dx + dy*dy + dz*dz
}

// PatrolBindStats bind 统计（trace/测试读取）。
type PatrolBindStats struct {
	Walkers     int
	Stand       int
	MissingPath int
	FarOrShort  int
}

// BindPatrol 把路线表绑到场景目录 spawns 上（就地吸附改写坐标 + 建时间
// 表）：PathId → 路线 → 最近路点吸附（≤35m）→ 按该 NPC 模板速度构建
// 时间表。不满足条件（无 PathId/无路线/过远/时间表不可用/敌对脚本类）
// → 站立（Schedule 保持 nil）。
func BindPatrol(spawns []Spawn, routes map[string]*Route, epochMs int64) PatrolBindStats {
	var stats PatrolBindStats
	for i := range spawns {
		spawn := &spawns[i]
		if isCombatScript(spawn.Script) {
			stats.Stand++
			continue
		}
		pathID := spawn.PathID
		if pathID == "" {
			pathID = strings.TrimSpace(spawn.Template["PathID"])
		}
		if pathID == "" {
			stats.Stand++
			continue
		}
		route, exists := routes[pathID]
		if !exists {
			stats.MissingPath++
			stats.Stand++
			continue
		}
		best, bestD2 := -1, float32(math.MaxFloat32)
		for j, waypoint := range route.Waypoints {
			dx, dy, dz := waypoint.X-spawn.X, waypoint.Y-spawn.Y, waypoint.Z-spawn.Z
			if d2 := dx*dx + dy*dy + dz*dz; d2 < bestD2 {
				bestD2 = d2
				best = j
			}
		}
		if best < 0 || bestD2 > patrolSnapMaxMeters*patrolSnapMaxMeters {
			stats.FarOrShort++
			stats.Stand++
			continue
		}
		snap := route.Waypoints[best]
		spawn.X, spawn.Y, spawn.Z = snap.X, snap.Y, snap.Z
		if snap.hasFacing() {
			spawn.Orient = snap.Orient
		}
		walk := patrolClampedSpeed(spawn.Template["WalkSpeed"], patrolWalkSpeedFallback)
		run := patrolClampedSpeed(spawn.Template["RunSpeed"], patrolRunSpeedFallback)
		schedule := BuildSchedule(route, walk, run, epochMs)
		if schedule == nil {
			stats.FarOrShort++
			stats.Stand++
			continue
		}
		spawn.Schedule = schedule
		stats.Walkers++
	}
	return stats
}

// patrolClampedSpeed 模板速度属性（缺失/非法回退缺省），钳 0.5..15 m/s。
func patrolClampedSpeed(raw string, fallback float32) float32 {
	speed := fallback
	if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 32); err == nil && v > 0 && patrolFiniteSpeed(float32(v)) {
		speed = float32(v)
	}
	if speed < patrolSpeedMin {
		return patrolSpeedMin
	}
	if speed > patrolSpeedMax {
		return patrolSpeedMax
	}
	return speed
}

// isCombatScript 敌对脚本类（巡逻与战斗互斥——敌对 AI 另期；与
// join.eligible 的 SubSceneNo 战斗类判据同源）。
func isCombatScript(script string) bool {
	normalized := strings.ToLower(strings.TrimSpace(script))
	return normalized == "attacknpc" || normalized == "bossnpc"
}
