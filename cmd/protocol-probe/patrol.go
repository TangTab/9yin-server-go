package main

// mappath 官方路线巡逻：把 share/creator/mappath 的路线表绑到场景出生点上，
// 运行时按共享的绝对时间表采样位置，经 0x21 ServerAllDest 每秒广播。
// 对译 Java 参考服 PatrolRoutes/PatrolSchedule/NpcMessages.serverAllDest。

import (
	"encoding/binary"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	worldcore "github.com/local/9yin-go-server/internal/world"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const (
	// patrolSnapMaxMeters 出生点离路线最近路点的吸附上限（米）。
	patrolSnapMaxMeters = float32(35)
	// 速度缺省（m/s）：与 npcconfig 城市表典型值一致。
	patrolWalkSpeedFallback = float32(1.25)
	patrolRunSpeedFallback  = float32(4)
	// patrolDedupeMeters 相邻路点去重阈值（米）。
	patrolDedupeMeters = float32(0.5)
	// patrolDwellCapMillis 停留列上限（毫秒）：数据实测 0~60000，留余量防脏值拖死路线。
	patrolDwellCapMillis = int64(120_000)
	// 行程时长钳制（毫秒）：下限防零碎腿刷帧，上限防脏数据长腿挂死。
	patrolTravelMinMillis = int64(300)
	patrolTravelMaxMillis = int64(90_000)
	// patrolMaxLegDYMeters 高度差钳制（米）：与前一保留点 |dy| 超过的路点剔除。
	patrolMaxLegDYMeters = float32(6)
	// 速度钳制（m/s）。
	patrolSpeedMin = float32(0.5)
	patrolSpeedMax = float32(15)
	// patrolBroadcastBatch 单帧 0x21 记录上限（官方抓包最多 5 条，留余量）。
	patrolBroadcastBatch = 8
	// patrolFirstBroadcastDelay 巡逻首拍延迟：clientActivity 后的 0x1F
	// 重放窗口（250ms..5s）内不发 0x21，防重放帧把走动 NPC 拽回物化位置。
	patrolFirstBroadcastDelay = 5 * time.Second
)

// ---- 路线数据（PatrolRoute） ----

// patrolWaypoint 单个路点；run=false 走 WalkSpeed，true 跑 RunSpeed；
// dwellMs = 到达本点后的停留（mappath 停留列，0~120s 钳制）。
type patrolWaypoint struct {
	x, y, z float32
	orient  float32
	run     bool
	dwellMs int64
}

// hasFacing 数据朝向哨兵（0=未设、-10=按行进方向）：非零且 |o|≤2π 才有效。
func (w patrolWaypoint) hasFacing() bool {
	return w.orient != 0 && math.Abs(float64(w.orient)) <= math.Pi*2
}

// patrolRoute mappath 巡逻路线（不可变，场景目录缓存内跨运行时共享）：
// 节名 = creator PathId，路点按行序往返。
type patrolRoute struct {
	pathID    string
	waypoints []patrolWaypoint
}

// ---- 路线表加载（PatrolRoutes.load） ----

// patrolSceneShortName 场景 creator 目录名 → 短名（首个 '_' 前，如
// city02_suzhou → city02）。
func patrolSceneShortName(sceneFolder string) string {
	if us := strings.IndexByte(sceneFolder, '_'); us >= 0 {
		return sceneFolder[:us]
	}
	return sceneFolder
}

// loadPatrolRoutes 加载场景路线表：mappath 目录下 stem 以场景短名开头且不含
// offline 的 .ini（GBK）。目录缺失/无匹配 → 空表（该场景无巡逻，不报错）。
func loadPatrolRoutes(mappathDir, sceneFolder string) map[string]*patrolRoute {
	table := make(map[string]*patrolRoute)
	entries, err := readResourceDirEntries(mappathDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("mappath scan error: %s: %v", mappathDir, err)
		} else {
			log.Printf("mappath dir missing: %s", mappathDir)
		}
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
		loadPatrolRouteFile(file, table)
	}
	log.Printf("mappath table scene=%s files=%d named_paths=%d", prefix, len(files), len(table))
	return table
}

// loadPatrolRouteFile 解析单个 mappath ini：节名 = PathId，每行 r= 一个路点。
func loadPatrolRouteFile(path string, table map[string]*patrolRoute) {
	data, err := readResourceBytes(path)
	if err != nil {
		log.Printf("mappath read failed: %s: %v", path, err)
		return
	}
	if decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), data); err == nil {
		data = decoded
	}
	section := ""
	var points []patrolWaypoint
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flushPatrolSection(table, section, points)
			section = line[1 : len(line)-1]
			points = nil
			continue
		}
		eq := strings.IndexByte(line, '=')
		if section == "" || eq <= 0 || (line[0] != 'r' && line[0] != 'R') {
			continue
		}
		if waypoint, ok := parsePatrolWaypoint(line[eq+1:]); ok {
			points = append(points, waypoint)
		}
	}
	flushPatrolSection(table, section, points)
}

// parsePatrolWaypoint 解析 r= 载荷。行有三种格式（对齐尾部 stand/nil/walk 列）：
//  1. ",<x>,<y>,<z>,…"（前导空列，主流）
//  2. "<导航点id>,<x>,<y>,<z>,…"（首列整数 id，须跳过——旧服误读成 x，
//     坐标整体错位，即"上天入地/穿墙"的根因）
//  3. "<x>,<y>,<z>,…"（无 id 列，少量）
//
// 列序：x,y,z,orient,dwellMs,…；walk/run 标记在尾部固定列。
func parsePatrolWaypoint(payload string) (patrolWaypoint, bool) {
	parts := strings.Split(payload, ",")
	base := patrolCoordinateBase(parts)
	if base < 0 || len(parts) < base+3 {
		return patrolWaypoint{}, false
	}
	parse := func(field string) (float64, bool) {
		value, err := strconv.ParseFloat(strings.TrimSpace(field), 32)
		return value, err == nil
	}
	x, ok := parse(parts[base])
	if !ok {
		return patrolWaypoint{}, false
	}
	y, ok := parse(parts[base+1])
	if !ok {
		return patrolWaypoint{}, false
	}
	z, ok := parse(parts[base+2])
	if !ok {
		return patrolWaypoint{}, false
	}
	orient := float64(0)
	if len(parts) > base+3 && strings.TrimSpace(parts[base+3]) != "" {
		if orient, ok = parse(parts[base+3]); !ok {
			return patrolWaypoint{}, false
		}
	}
	// 停留列：朝向后第一个数值列（A/B 格式 field5、无 id 格式 field4）。
	dwellMs := float64(0)
	if len(parts) > base+4 && strings.TrimSpace(parts[base+4]) != "" {
		if dwellMs, ok = parse(parts[base+4]); !ok {
			return patrolWaypoint{}, false
		}
	}
	dwellMs = math.Max(0, math.Min(float64(patrolDwellCapMillis), dwellMs))
	run := len(parts) > 8 && strings.EqualFold(strings.TrimSpace(parts[8]), "run")
	return patrolWaypoint{
		x: float32(x), y: float32(y), z: float32(z),
		orient: float32(orient), run: run, dwellMs: int64(dwellMs),
	}, true
}

// patrolCoordinateBase 坐标起始列：首列为空（格式1）或纯整数导航点 id（格式2）
// → x 在第 1 列；首列为浮点（格式3，无 id 列）→ x 在第 0 列；其余 -1。
func patrolCoordinateBase(parts []string) int {
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
	if _, err := strconv.ParseInt(first, 10, 64); err != nil {
		return -1
	}
	return 1
}

// flushPatrolSection 节收尾：相邻点去重（<0.5m）后 ≥2 点才成路线；同名节首个生效。
func flushPatrolSection(table map[string]*patrolRoute, section string, points []patrolWaypoint) {
	if section == "" || len(points) < 2 {
		return
	}
	cleaned := make([]patrolWaypoint, 0, len(points))
	for _, point := range points {
		if len(cleaned) > 0 && patrolDistanceSq(cleaned[len(cleaned)-1], point) < patrolDedupeMeters*patrolDedupeMeters {
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
	table[section] = &patrolRoute{pathID: section, waypoints: cleaned}
}

func patrolDistanceSq(a, b patrolWaypoint) float32 {
	dx, dy, dz := a.x-b.x, a.y-b.y, a.z-b.z
	return dx*dx + dy*dy + dz*dz
}

// ---- 巡逻时间表（PatrolSchedule） ----

// patrolSchedule 巡逻时间表：把路线展开成"行程段 + 停留"的绝对时间线
// （往返一个周期），任意时刻可采样当前位置与 0x21 广播动作。同场景目录
// 内共享一份（绑定时刻构建，epochMs 为共同时钟起点），因此同场景不同
// 玩家看到一致的 NPC 位置，视口重进时从中断处继续而非重走。
//
// 段序：w0→w1→…→w(n-1)→w(n-2)→…→w0；每段 = 行程(距离/速度) + 到达
// 路点停留（mappath 停留列归属到达行）。构建时剔除与前一保留点高度差
// >6m 的路点（脏数据防飞天，全连接一致）。
type patrolSchedule struct {
	pathID    string
	epochMs   int64
	cycleMs   int64
	points    []patrolWaypoint
	starts    []int64
	travels   []int64
	dwells    []int64
	fromIdx   []int
	toIdx     []int
	walkSpeed float32
	runSpeed  float32
}

// patrolPosition 采样位置（行程中按比例插值，停留中即路点坐标）。
type patrolPosition struct {
	x, y, z, orient float32
}

// patrolMotion 官方 ServerAllDest（0x21）语义的广播动作：段目标坐标/
// 朝向、当前速度（行进=该腿速度，停留=0）、移动标志。官方真机抓包：
// 移动中 spd=WalkSpeed 且 tail=1，到达停留 spd=0 且 tail=0，每秒一拍全量广播。
type patrolMotion struct {
	destX, destY, destZ, destOrient float32
	speed                           float32
	moving                          bool
}

// buildPatrolSchedule 构建（不可用返回 nil：高度过滤后不足 2 点或速度非法）。
// walkSpeed/runSpeed 为该 NPC 的 0x0D 同值属性（m/s）。
func buildPatrolSchedule(route *patrolRoute, walkSpeed, runSpeed float32, epochMs int64) *patrolSchedule {
	if !patrolFiniteSpeed(walkSpeed) || !patrolFiniteSpeed(runSpeed) || walkSpeed <= 0 || runSpeed <= 0 {
		return nil
	}
	kept := filterPatrolWaypointsByDY(route.waypoints)
	if len(kept) < 2 {
		return nil
	}
	segments := (len(kept) - 1) * 2
	schedule := &patrolSchedule{
		pathID:    route.pathID,
		epochMs:   epochMs,
		points:    kept,
		starts:    make([]int64, segments),
		travels:   make([]int64, segments),
		dwells:    make([]int64, segments),
		fromIdx:   make([]int, segments),
		toIdx:     make([]int, segments),
		walkSpeed: walkSpeed,
		runSpeed:  runSpeed,
	}
	cursor := int64(0)
	k := 0
	// 去程 w0→…→w(n-1)，回程 w(n-1)→…→w0；停留归属到达行。
	for i := 0; i+1 < len(kept); i++ {
		cursor = schedule.appendSegment(i, i+1, cursor, k)
		k++
	}
	for i := len(kept) - 1; i-1 >= 0; i-- {
		cursor = schedule.appendSegment(i, i-1, cursor, k)
		k++
	}
	schedule.cycleMs = cursor
	return schedule
}

func patrolFiniteSpeed(speed float32) bool {
	return !math.IsNaN(float64(speed)) && !math.IsInf(float64(speed), 0)
}

// filterPatrolWaypointsByDY 剔除与前一保留点高度差 >6m 的路点（防飞天脏数据）。
func filterPatrolWaypointsByDY(waypoints []patrolWaypoint) []patrolWaypoint {
	kept := make([]patrolWaypoint, 0, len(waypoints))
	for _, point := range waypoints {
		if len(kept) > 0 && math.Abs(float64(point.y-kept[len(kept)-1].y)) > float64(patrolMaxLegDYMeters) {
			continue
		}
		kept = append(kept, point)
	}
	return kept
}

// appendSegment 追加一段行程+停留，返回推进后的游标。
func (s *patrolSchedule) appendSegment(from, to int, cursor int64, k int) int64 {
	a, b := s.points[from], s.points[to]
	dx, dz := b.x-a.x, b.z-a.z
	distXz := float32(math.Sqrt(float64(dx*dx + dz*dz)))
	speed := s.walkSpeed
	if b.run {
		speed = s.runSpeed
	}
	travel := int64(float64(distXz) / float64(speed) * 1000)
	travel = max64(patrolTravelMinMillis, min64(patrolTravelMaxMillis, travel))
	s.starts[k] = cursor
	s.travels[k] = travel
	s.dwells[k] = b.dwellMs
	s.fromIdx[k] = from
	s.toIdx[k] = to
	return cursor + travel + b.dwellMs
}

// positionAt 时刻 nowMs 的当前位置（行程中插值；朝向=目标朝向或行进方向）。
func (s *patrolSchedule) positionAt(nowMs int64) patrolPosition {
	k := s.segmentAt(nowMs)
	a, b := s.points[s.fromIdx[k]], s.points[s.toIdx[k]]
	offset := s.offsetOf(nowMs)
	travelEnd := s.starts[k] + s.travels[k]
	if offset >= travelEnd {
		return patrolPosition{x: b.x, y: b.y, z: b.z, orient: patrolFacingOf(a, b)}
	}
	frac := float32(offset-s.starts[k]) / float32(s.travels[k])
	return patrolPosition{
		x:      a.x + (b.x-a.x)*frac,
		y:      a.y + (b.y-a.y)*frac,
		z:      a.z + (b.z-a.z)*frac,
		orient: patrolFacingOf(a, b),
	}
}

// motionAt 时刻 nowMs 的 0x21 广播动作：目的地=段末路点，速度=该腿速度
// （run 腿取 runSpeed），停留中速度 0。官方真机节拍每秒全量广播（含
// 停留中的实体）。
func (s *patrolSchedule) motionAt(nowMs int64) patrolMotion {
	k := s.segmentAt(nowMs)
	a, b := s.points[s.fromIdx[k]], s.points[s.toIdx[k]]
	offset := s.offsetOf(nowMs)
	moving := offset < s.starts[k]+s.travels[k]
	speed := float32(0)
	if moving {
		speed = s.walkSpeed
		if b.run {
			speed = s.runSpeed
		}
	}
	return patrolMotion{
		destX: b.x, destY: b.y, destZ: b.z, destOrient: patrolFacingOf(a, b),
		speed: speed, moving: moving,
	}
}

// patrolFacingOf 朝向：路点有效朝向优先，否则按段行进方向。
func patrolFacingOf(from, to patrolWaypoint) float32 {
	if to.hasFacing() {
		return to.orient
	}
	return float32(math.Atan2(float64(to.x-from.x), float64(to.z-from.z)))
}

func (s *patrolSchedule) segmentAt(nowMs int64) int {
	offset := s.offsetOf(nowMs)
	for k := len(s.starts) - 1; k >= 0; k-- {
		// 段区间 [starts[k], starts[k]+travel+dwell)；倒序首命中。
		if offset >= s.starts[k] {
			return k
		}
	}
	return 0
}

func (s *patrolSchedule) offsetOf(nowMs int64) int64 {
	offset := (nowMs - s.epochMs) % s.cycleMs
	if offset < 0 {
		offset += s.cycleMs
	}
	return offset
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

// ---- 出生点绑定（PatrolRoutes.bind） ----

// patrolBindStats bind 统计（日志/测试读取）。
type patrolBindStats struct {
	walkers     int
	stand       int
	missingPath int
	farOrShort  int
}

// bindPatrolRoutes 把路线表绑到场景 spawns 上（就地吸附改写坐标）：
// PathId → 路线 → 最近路点吸附 → 按该 NPC 速度属性构建巡逻时间表
// （目录缓存内共享的绝对时间线）。不满足条件（无 PathId/无路线/过远/
// 时间表不可用/敌对脚本）→ 站立。
func bindPatrolRoutes(spawns []npcSpawn, routes map[string]*patrolRoute, epochMs int64) patrolBindStats {
	var stats patrolBindStats
	for index := range spawns {
		spawn := &spawns[index]
		spawn.route = nil
		spawn.routeStartIdx = 0
		spawn.schedule = nil
		if patrolCombatScriptClass(spawn.resolved.ScriptClass) {
			stats.stand++
			continue
		}
		pathID := patrolPathIDOf(*spawn)
		if pathID == "" {
			stats.stand++
			continue
		}
		route, exists := routes[pathID]
		if !exists {
			stats.missingPath++
			stats.stand++
			continue
		}
		best, bestD2 := -1, float32(math.MaxFloat32)
		for i, waypoint := range route.waypoints {
			dx, dy, dz := waypoint.x-spawn.x, waypoint.y-spawn.y, waypoint.z-spawn.z
			if d2 := dx*dx + dy*dy + dz*dz; d2 < bestD2 {
				bestD2 = d2
				best = i
			}
		}
		if best < 0 || bestD2 > patrolSnapMaxMeters*patrolSnapMaxMeters {
			stats.farOrShort++
			stats.stand++
			continue
		}
		snap := route.waypoints[best]
		spawn.x, spawn.y, spawn.z = snap.x, snap.y, snap.z
		if snap.hasFacing() {
			spawn.orient = snap.orient
		}
		// 时间表：速度与 0x0D 下发属性同值（客户端动画同步的关键）。
		walk := patrolClampedSpeed(*spawn, "WalkSpeed", patrolWalkSpeedFallback)
		run := patrolClampedSpeed(*spawn, "RunSpeed", patrolRunSpeedFallback)
		schedule := buildPatrolSchedule(route, walk, run, epochMs)
		if schedule == nil {
			stats.farOrShort++
			stats.stand++
			continue
		}
		spawn.route = route
		spawn.routeStartIdx = best
		spawn.schedule = schedule
		stats.walkers++
	}
	log.Printf("patrol bind: walkers=%d stand=%d missingPath=%d farOrShort=%d named_paths=%d",
		stats.walkers, stats.stand, stats.missingPath, stats.farOrShort, len(routes))
	return stats
}

// patrolClampedSpeed NPC 速度属性（缺失/非法回退缺省），钳 0.5..15 m/s。
func patrolClampedSpeed(spawn npcSpawn, property string, fallback float32) float32 {
	speed := spawn.float32Property(property)
	if speed <= 0 || !patrolFiniteSpeed(speed) {
		speed = fallback
	}
	if speed < patrolSpeedMin {
		return patrolSpeedMin
	}
	if speed > patrolSpeedMax {
		return patrolSpeedMax
	}
	return speed
}

// patrolPathIDOf creator PathId 优先（PathId/PathID 两种拼法），模板 PathID 缺省。
func patrolPathIDOf(spawn npcSpawn) string {
	for _, key := range []string{"creator.PathId", "creator.PathID", "template.PathID", "template.PathId"} {
		if value := strings.TrimSpace(spawn.resolved.Extensions[key]); value != "" {
			return value
		}
	}
	return ""
}

// patrolCombatScriptClass 与 npcInteractionFor 的战斗判定同口径（bind 时
// 排除敌对 actor：敌对巡逻与战斗互斥，敌对 AI 另期）。
func patrolCombatScriptClass(scriptClass string) bool {
	normalized := strings.ToLower(strings.TrimSpace(scriptClass))
	return normalized == "attacknpc" || normalized == "bossnpc"
}

// ---- 0x21 ServerAllDest（NpcMessages.serverAllDest） ----

// allDestEntry 官方 ServerAllDest（0x21，0x2C 同义）的单条移动记录：
// id u32 | owner u32 | 目的地 x/y/z | orient | 当前速度 | 3×0 | 移动标志 u32。
type allDestEntry struct {
	objectID   uint32
	ownerID    uint32
	destX      float32
	destY      float32
	destZ      float32
	destOrient float32
	speed      float32
	moveFlags  uint32
}

// serverAllDest 编码移动实体的批量广播帧：`op | u16 count | count×44B`。
// 速度与移动标志驱动客户端走/停动画（行进 spd=WalkSpeed/RunSpeed 且
// flags=1；到达停留 spd=0 且 flags=0），官方真机每秒一拍全量广播。
// 0x20 ServerMoving 为参考服未验证帧型（官方抓包从未出现），巡逻移动
// 一律走本帧。
func serverAllDest(entries []allDestEntry) []byte {
	if len(entries) == 0 {
		return nil
	}
	payload := make([]byte, 3, 3+len(entries)*44)
	payload[0] = 0x21
	binary.LittleEndian.PutUint16(payload[1:], uint16(len(entries)))
	for _, entry := range entries {
		payload = binary.LittleEndian.AppendUint32(payload, entry.objectID)
		payload = binary.LittleEndian.AppendUint32(payload, entry.ownerID)
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(entry.destX))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(entry.destY))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(entry.destZ))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(entry.destOrient))
		payload = binary.LittleEndian.AppendUint32(payload, math.Float32bits(entry.speed))
		payload = binary.LittleEndian.AppendUint32(payload, 0)
		payload = binary.LittleEndian.AppendUint32(payload, 0)
		payload = binary.LittleEndian.AppendUint32(payload, 0)
		payload = binary.LittleEndian.AppendUint32(payload, entry.moveFlags)
	}
	return payload
}

// patchAddObjectTransform 把已物化 0x0D 载荷头部的坐标四槽重打为当前
// 权威 Transform（偏移与 npcAddObject 一致：0x09/0x0D/0x11/0x15）。
// 巡逻 NPC 重进视口时由此落到时间表当前位置，避免从出生点闪现。
func patchAddObjectTransform(payload []byte, transform worldcore.Transform) {
	if len(payload) < 0x19 {
		return
	}
	binary.LittleEndian.PutUint32(payload[0x09:], math.Float32bits(transform.X))
	binary.LittleEndian.PutUint32(payload[0x0D:], math.Float32bits(transform.Y))
	binary.LittleEndian.PutUint32(payload[0x11:], math.Float32bits(transform.Z))
	binary.LittleEndian.PutUint32(payload[0x15:], math.Float32bits(transform.Orient))
}

// ---- 中央 tick 的巡逻段 ----

// patrolTick 对全部在巡实体按时刻采样目的地/速度/移动标志，按 ID 序组成
// 0x21 批量帧（超出单帧上限分帧）；行进中实体的服务端权威坐标同步为时间
// 表插值位置。写的是 ECS Transform 组件而非 setNPCTransformLocked——
// 不抬升可视权威 Scene 的 revision，视口对账不因巡逻产生 0x1F 更新
// （与官方一致）；重进视口的 0x0D/0x1F 采样自然续走。
// 结算在 run.mu 内完成，帧写入在锁外进行（与 combatTick 同节奏）。
// clientActivity 后 5s 内（0x1F 重放窗口）的成员不收巡逻帧。
func (r *sceneRun) patrolTick(now time.Time) {
	r.mu.Lock()
	if r.closed || len(r.patrolIDs) == 0 || len(r.members) == 0 {
		r.mu.Unlock()
		return
	}
	nowMs := now.UnixMilli()
	samples := make([]allDestEntry, 0, len(r.patrolIDs))
	kept := r.patrolIDs[:0]
	for _, id := range r.patrolIDs {
		entityID := worldcore.EntityID(id)
		meta, exists := r.npcMeta(entityID)
		if !exists || meta.schedule == nil {
			continue // 实体已注销（测试路径）：移出在巡表
		}
		kept = append(kept, id)
		motion := meta.schedule.motionAt(nowMs)
		position := meta.schedule.positionAt(nowMs)
		r.writeNpcTransform(entityID, worldcore.Transform{X: position.x, Y: position.y, Z: position.z, Orient: position.orient})
		flags := uint32(0)
		if motion.moving {
			flags = 1
		}
		samples = append(samples, allDestEntry{
			objectID: meta.id, ownerID: meta.ownerID,
			destX: motion.destX, destY: motion.destY, destZ: motion.destZ,
			destOrient: motion.destOrient, speed: motion.speed, moveFlags: flags,
		})
	}
	r.patrolIDs = kept
	if len(samples) == 0 {
		r.mu.Unlock()
		return
	}
	type patrolDelivery struct {
		member *sceneMember
		frames [][]byte
	}
	deliveries := make([]patrolDelivery, 0, len(r.members))
	for _, member := range r.sortedMembersLocked() {
		if member.view == nil || member.patrolReadyAt.IsZero() || now.Before(member.patrolReadyAt) {
			continue
		}
		batch := make([]allDestEntry, 0, len(samples))
		for _, sample := range samples {
			if !member.view.npcVisible(sample.objectID) {
				continue
			}
			batch = append(batch, sample)
		}
		if len(batch) == 0 {
			continue
		}
		frames := make([][]byte, 0, (len(batch)+patrolBroadcastBatch-1)/patrolBroadcastBatch)
		for start := 0; start < len(batch); start += patrolBroadcastBatch {
			end := start + patrolBroadcastBatch
			if end > len(batch) {
				end = len(batch)
			}
			if frame := serverAllDest(batch[start:end]); frame != nil {
				frames = append(frames, frame)
			}
		}
		deliveries = append(deliveries, patrolDelivery{member: member, frames: frames})
	}
	r.mu.Unlock()
	for _, delivery := range deliveries {
		for _, frame := range delivery.frames {
			if err := delivery.member.view.conn.WriteFrame(frame); err != nil {
				log.Printf("%s: patrol ServerAllDest: %v", delivery.member.view.remote, err)
			}
		}
	}
}
