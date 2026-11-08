package main

import (
	worldcore "github.com/local/9yin-go-server/internal/world"

	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// TestParsePatrolWaypoint 覆盖 mappath 三种行格式、停留钳制与 run 标记。
func TestParsePatrolWaypoint(t *testing.T) {
	// 格式1：前导空列（主流）。
	waypoint, ok := parsePatrolWaypoint(",458.7,0.629,637.8,-10,5000,stand,nil,walk,nil,1,")
	if !ok {
		t.Fatal("leading-empty-column row must parse")
	}
	if waypoint.x != 458.7 || waypoint.y != 0.629 || waypoint.z != 637.8 {
		t.Fatalf("coordinates=%+v", waypoint)
	}
	if waypoint.orient != -10 || waypoint.dwellMs != 5000 || waypoint.run {
		t.Fatalf("orient/dwell/run = %v/%d/%v", waypoint.orient, waypoint.dwellMs, waypoint.run)
	}
	if waypoint.hasFacing() {
		t.Fatal("-10 is the follow-direction sentinel, not a facing")
	}

	// 格式2：首列整数导航点 id 必须跳过（旧服误读成 x 的根因）。
	waypoint, ok = parsePatrolWaypoint("59,458.7,0.629,637.8,-10,60000,stand,nil,run,nil,1,")
	if !ok {
		t.Fatal("nav-id row must parse")
	}
	if waypoint.x != 458.7 || waypoint.y != 0.629 || waypoint.z != 637.8 {
		t.Fatalf("coordinates=%+v", waypoint)
	}
	if waypoint.dwellMs != 60000 || !waypoint.run {
		t.Fatalf("dwell/run = %d/%v", waypoint.dwellMs, waypoint.run)
	}

	// 格式3：无 id 列。
	waypoint, ok = parsePatrolWaypoint("318.196625,3.794954,632.016357,0.494,4000,0,stand,nil,walk,nil,1,")
	if !ok {
		t.Fatal("no-id row must parse")
	}
	if waypoint.x != 318.196625 || waypoint.orient != 0.494 || waypoint.dwellMs != 4000 {
		t.Fatalf("waypoint=%+v", waypoint)
	}
	if !waypoint.hasFacing() {
		t.Fatal("0.494 within ±2π is a valid facing")
	}

	// 停留钳制 0~120s。
	waypoint, ok = parsePatrolWaypoint(",1,2,3,0,999999,stand,nil,walk,nil,1,")
	if !ok || waypoint.dwellMs != patrolDwellCapMillis {
		t.Fatalf("dwell clamp=%d ok=%v", waypoint.dwellMs, ok)
	}
	waypoint, ok = parsePatrolWaypoint(",1,2,3,0,-5,stand,nil,walk,nil,1,")
	if !ok || waypoint.dwellMs != 0 {
		t.Fatalf("negative dwell=%d ok=%v", waypoint.dwellMs, ok)
	}

	// 坏行拒绝：非数字首列、列数不足。
	if _, ok := parsePatrolWaypoint("abc,1,2,3"); ok {
		t.Fatal("non-numeric first column must be rejected")
	}
	if _, ok := parsePatrolWaypoint("59"); ok {
		t.Fatal("row without coordinates must be rejected")
	}
}

// TestLoadPatrolRoutes 验证场景短名过滤、offline 排除、去重与
// 不足两点的节不成路线。
func TestLoadPatrolRoutes(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("city05_npc.ini", "[AI_cd_071]\r\n"+
		"r=540,330.44,77.462,844.773,0,0,stand,nil,walk,nil,1,\r\n"+
		"r=130,308.9,77.462,840.9,0,3000,stand,nil,walk,nil,1,\r\n"+
		"; comment line\r\n"+
		"r=,290.0,77.5,830.0,0,0,stand,nil,walk,nil,1,\r\n"+
		"[dup_points]\r\n"+
		"r=,10,0,10,0,0,stand,nil,walk,nil,1,\r\n"+
		"r=,10.05,0,10.05,0,0,stand,nil,walk,nil,1,\r\n"+ // <0.5m 去重后仅 1 点
		"[single]\r\n"+
		"r=,1,2,3,0,0,stand,nil,walk,nil,1,\r\n")
	write("city05_offline.ini", "[AI_off]\r\nr=,1,0,1,0,0,stand,nil,walk,nil,1,\r\nr=,2,0,2,0,0,stand,nil,walk,nil,1,\r\n")
	write("city06_npc.ini", "[AI_other]\r\nr=,1,0,1,0,0,stand,nil,walk,nil,1,\r\nr=,2,0,2,0,0,stand,nil,walk,nil,1,\r\n")

	table := loadPatrolRoutes(dir, "city05_chengdu")
	route, exists := table["AI_cd_071"]
	if !exists {
		t.Fatalf("AI_cd_071 missing, table=%v", keysOfPatrolTable(table))
	}
	if len(route.waypoints) != 3 {
		t.Fatalf("waypoints=%d, want 3", len(route.waypoints))
	}
	first := route.waypoints[0]
	if first.x != 330.44 || first.z != 844.773 || first.dwellMs != 0 {
		t.Fatalf("first waypoint=%+v", first)
	}
	if route.waypoints[1].dwellMs != 3000 {
		t.Fatalf("second dwell=%d", route.waypoints[1].dwellMs)
	}
	for _, banned := range []string{"AI_off", "AI_other", "dup_points", "single"} {
		if _, exists := table[banned]; exists {
			t.Fatalf("section %s must not become a route", banned)
		}
	}

	if table := loadPatrolRoutes(filepath.Join(dir, "missing"), "city05_chengdu"); len(table) != 0 {
		t.Fatal("missing mappath dir must yield an empty table")
	}
}

func keysOfPatrolTable(table map[string]*patrolRoute) []string {
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	return keys
}

// TestPatrolScheduleSampling 验证时间表的几何：插值、停留、run 腿速度、
// 高度过滤与周期回绕。
func TestPatrolScheduleSampling(t *testing.T) {
	route := &patrolRoute{pathID: "t", waypoints: []patrolWaypoint{
		{x: 0, y: 0, z: 0},
		{x: 0, y: 0, z: 10, dwellMs: 3000},
	}}
	// walk 2 m/s：去程/回程各 5000ms，w1 停留 3000ms → 周期 13000ms。
	schedule := buildPatrolSchedule(route, 2, 4, 0)
	if schedule == nil {
		t.Fatal("schedule must build")
	}
	if schedule.cycleMs != 13000 {
		t.Fatalf("cycle=%d, want 13000", schedule.cycleMs)
	}
	position := schedule.positionAt(2500)
	if math.Abs(float64(position.z-5)) > 1e-4 || math.Abs(float64(position.x)) > 1e-6 {
		t.Fatalf("mid-leg position=%+v, want z=5", position)
	}
	motion := schedule.motionAt(2500)
	if motion.destZ != 10 || motion.speed != 2 || !motion.moving {
		t.Fatalf("mid-leg motion=%+v", motion)
	}
	// 停留中：位置=路点，速度 0，flags 语义（moving=false）。
	position = schedule.positionAt(6000)
	if position.z != 10 {
		t.Fatalf("dwelling position=%+v", position)
	}
	motion = schedule.motionAt(6000)
	if motion.speed != 0 || motion.moving || motion.destZ != 10 {
		t.Fatalf("dwelling motion=%+v", motion)
	}
	// 回程中：目的地回到 w0。
	motion = schedule.motionAt(10500)
	if motion.destZ != 0 || motion.speed != 2 || !motion.moving {
		t.Fatalf("return motion=%+v", motion)
	}
	position = schedule.positionAt(10500)
	if math.Abs(float64(position.z-5)) > 1e-4 {
		t.Fatalf("return mid position=%+v", position)
	}
	// 周期回绕与负时刻。
	if got := schedule.positionAt(2500 + 13000); math.Abs(float64(got.z-5)) > 1e-4 {
		t.Fatalf("wrapped position=%+v", got)
	}
	if got := schedule.positionAt(2500 - 13000); math.Abs(float64(got.z-5)) > 1e-4 {
		t.Fatalf("pre-epoch position=%+v", got)
	}

	// run 腿：到达点标记 run → 去程用 runSpeed（2500ms），回程到达
	// w0（walk）用 walkSpeed（5000ms），周期 7500ms。
	runRoute := &patrolRoute{pathID: "t", waypoints: []patrolWaypoint{
		{x: 0, z: 0},
		{x: 0, z: 10, run: true},
	}}
	runSchedule := buildPatrolSchedule(runRoute, 2, 4, 0)
	if runSchedule == nil || runSchedule.cycleMs != 7500 {
		t.Fatalf("run schedule cycle=%d", runSchedule.cycleMs)
	}
	if motion := runSchedule.motionAt(1000); motion.speed != 4 || !motion.moving {
		t.Fatalf("run-leg motion=%+v", motion)
	}

	// 高度差 >6m 的路点剔除：中间飞天点被丢弃，路线仍两点可用。
	dirtyRoute := &patrolRoute{pathID: "t", waypoints: []patrolWaypoint{
		{x: 0, y: 0, z: 0},
		{x: 0, y: 50, z: 5},
		{x: 0, y: 1, z: 10},
	}}
	dirty := buildPatrolSchedule(dirtyRoute, 2, 4, 0)
	if dirty == nil || len(dirty.points) != 2 || dirty.points[1].z != 10 {
		t.Fatalf("dy-filtered schedule=%v", dirty)
	}
	// 剔除后不足两点 → nil。
	lonely := buildPatrolSchedule(&patrolRoute{pathID: "t", waypoints: []patrolWaypoint{
		{x: 0, y: 0, z: 0},
		{x: 0, y: 50, z: 5},
	}}, 2, 4, 0)
	if lonely != nil {
		t.Fatal("single surviving waypoint must not build")
	}
	// 速度非法 → nil。
	if buildPatrolSchedule(route, 0, 4, 0) != nil {
		t.Fatal("zero walk speed must not build")
	}
	// 行程下限钳制：0.1m / 2 m/s = 50ms → 300ms。
	tiny := buildPatrolSchedule(&patrolRoute{pathID: "t", waypoints: []patrolWaypoint{
		{x: 0, z: 0}, {x: 0.1, z: 0},
	}}, 2, 4, 0)
	if tiny == nil || tiny.travels[0] != patrolTravelMinMillis {
		t.Fatalf("tiny-leg travel=%d", tiny.travels[0])
	}
}

// TestBindPatrolRoutes 验证绑定闸门：敌对站立、无路径站立、缺路线、
// 超 35m 不吸附、吸附改写坐标与速度回退/钳制。
func TestBindPatrolRoutes(t *testing.T) {
	routes := map[string]*patrolRoute{
		"AI_x": {pathID: "AI_x", waypoints: []patrolWaypoint{
			{x: 0, y: 0, z: 0, orient: 1.2},
			{x: 0, y: 0, z: 10},
		}},
	}
	combat := testNPCSpawn()
	combat.resolved.ScriptClass = "AttackNpc"
	combat.resolved.Extensions["creator.PathId"] = "AI_x"
	combat.x, combat.z = 1, 1

	talker := testNPCSpawn()
	talker.resolved.Extensions["creator.PathId"] = "AI_x"
	talker.x, talker.z = 1, 1

	noPath := testNPCSpawn()
	noPath.x, noPath.z = 1, 1

	missing := testNPCSpawn()
	missing.resolved.Extensions["creator.PathId"] = "AI_missing"
	missing.x, missing.z = 1, 1

	far := testNPCSpawn()
	far.resolved.Extensions["template.PathID"] = "AI_x"
	far.x, far.z = 100, 100

	spawns := []npcSpawn{combat, talker, noPath, missing, far}
	stats := bindPatrolRoutes(spawns, routes, 0)

	if stats.walkers != 1 || stats.stand != 4 || stats.missingPath != 1 || stats.farOrShort != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	if spawns[0].schedule != nil {
		t.Fatal("combat script class must not patrol")
	}
	if spawns[1].schedule == nil || spawns[1].routeStartIdx != 0 {
		t.Fatalf("talker must bind route at nearest waypoint: %+v", spawns[1])
	}
	if spawns[1].x != 0 || spawns[1].z != 0 || spawns[1].orient != 1.2 {
		t.Fatalf("snap must rewrite spawn transform: %+v", spawns[1])
	}
	// 模板无速度属性 → 回退 1.25/4.0。
	if spawns[1].schedule.walkSpeed != patrolWalkSpeedFallback || spawns[1].schedule.runSpeed != patrolRunSpeedFallback {
		t.Fatalf("fallback speeds=%v/%v", spawns[1].schedule.walkSpeed, spawns[1].schedule.runSpeed)
	}
	if spawns[2].schedule != nil || spawns[3].schedule != nil || spawns[4].schedule != nil {
		t.Fatal("stand cases must not receive schedules")
	}
	if spawns[4].x != 100 || spawns[4].z != 100 {
		t.Fatalf("far spawn must keep its coordinates: %+v", spawns[4])
	}

	// 速度钳制：WalkSpeed=99 → 15。bind 就地改写切片元素。
	fast := []npcSpawn{testNPCSpawn()}
	fast[0].resolved.Extensions["creator.PathId"] = "AI_x"
	fast[0].x, fast[0].z = 1, 1
	fast[0].resolved.Properties["WalkSpeed"] = clientdata.PropertyValue{Value: clientdata.Float32Value(99)}
	bindPatrolRoutes(fast, routes, 0)
	if fast[0].schedule == nil {
		t.Fatal("clamped-speed spawn must bind")
	}
	if fast[0].schedule.walkSpeed != patrolSpeedMax {
		t.Fatalf("clamped walk speed=%v", fast[0].schedule.walkSpeed)
	}
}

// TestServerAllDestEncoding 逐字节核对 0x21 帧布局（3 字节头 + 44B 记录）。
func TestServerAllDestEncoding(t *testing.T) {
	if serverAllDest(nil) != nil {
		t.Fatal("empty entries must not encode")
	}
	frame := serverAllDest([]allDestEntry{{
		objectID: 2, ownerID: 1,
		destX: 1.5, destY: 2.5, destZ: 3.5, destOrient: 0.25,
		speed: 1.25, moveFlags: 1,
	}})
	if len(frame) != 3+44 {
		t.Fatalf("frame length=%d", len(frame))
	}
	if frame[0] != 0x21 || binary.LittleEndian.Uint16(frame[1:]) != 1 {
		t.Fatalf("header=%v", frame[:3])
	}
	if got := binary.LittleEndian.Uint32(frame[3:]); got != 2 {
		t.Fatalf("object id=%d", got)
	}
	if got := binary.LittleEndian.Uint32(frame[7:]); got != 1 {
		t.Fatalf("owner id=%d", got)
	}
	if bits := binary.LittleEndian.Uint32(frame[11:]); math.Float32frombits(bits) != 1.5 {
		t.Fatalf("destX=%v", math.Float32frombits(bits))
	}
	if bits := binary.LittleEndian.Uint32(frame[19:]); math.Float32frombits(bits) != 3.5 {
		t.Fatalf("destZ=%v", math.Float32frombits(bits))
	}
	if bits := binary.LittleEndian.Uint32(frame[27:]); math.Float32frombits(bits) != 1.25 {
		t.Fatalf("speed=%v", math.Float32frombits(bits))
	}
	for _, zero := range frame[31:43] {
		if zero != 0 {
			t.Fatalf("reserved tail must be zero: %v", frame[31:43])
		}
	}
	if got := binary.LittleEndian.Uint32(frame[43:]); got != 1 {
		t.Fatalf("moveFlags=%d", got)
	}
}

// TestSceneRunPatrolTick 端到端：在巡 NPC 的权威 Transform 被时间表
// 插值刷新（可视权威 Scene revision 不动）、0x21 只发给越过重放窗口
// 且视口内可见该 NPC 的成员。
func TestSceneRunPatrolTick(t *testing.T) {
	conn := &captureMessageConnection{}
	suppressedConn := &captureMessageConnection{}
	distantConn := &captureMessageConnection{}
	view := newSceneLifecycle("near", conn)
	suppressedView := newSceneLifecycle("suppressed", suppressedConn)
	distantView := newSceneLifecycle("distant", distantConn)
	view.radius = 50
	suppressedView.radius = 50
	distantView.radius = 50

	spawn := testNPCSpawn() // CommonNpc
	spawn.schedule = buildPatrolSchedule(&patrolRoute{pathID: "t", waypoints: []patrolWaypoint{
		{x: 0, y: 0, z: 0},
		{x: 0, y: 0, z: 10, dwellMs: 3000},
	}}, 2, 4, 0)
	if spawn.schedule == nil {
		t.Fatal("schedule must build")
	}
	catalog := []npcSpawn{spawn}
	run := newSceneRun(sceneKey{config: "scene", resource: "resource"}, nil)
	if err := run.populate(catalog); err != nil {
		t.Fatal(err)
	}
	view.run = run
	suppressedView.run = run
	distantView.run = run

	view.attachMember(newPlayerActor("near", 0), role.Position{X: 0}, roleVisual{})
	suppressedView.attachMember(newPlayerActor("suppressed", 0), role.Position{X: 0}, roleVisual{})
	distantView.attachMember(newPlayerActor("distant", 0), role.Position{X: 400, Z: 400}, roleVisual{})
	run.activateMember(view.wireID)
	run.activateMember(suppressedView.wireID)
	run.activateMember(distantView.wireID)
	// 就绪成员把首拍窗口拨到过去；suppressed 成员保留 now+5s。
	run.mu.Lock()
	run.members[view.wireID].patrolReadyAt = time.UnixMilli(0)
	run.mu.Unlock()
	view.refreshVisibility(catalog, role.Position{X: 0})
	suppressedView.refreshVisibility(catalog, role.Position{X: 0})
	distantView.refreshVisibility(catalog, role.Position{X: 400, Z: 400})

	sceneBefore, err := run.scene.Get(worldcore.EntityID(forkNPCID(2)))
	if err != nil {
		t.Fatal(err)
	}
	nearBefore := len(conn.Frames())
	run.patrolTick(time.UnixMilli(2500)) // 去程中点：z=5

	transform, ok := run.npcTransform(worldcore.EntityID(forkNPCID(2)))
	if !ok || math.Abs(float64(transform.Z-5)) > 1e-4 {
		t.Fatalf("interpolated transform=%+v", transform)
	}
	sceneAfter, err := run.scene.Get(worldcore.EntityID(forkNPCID(2)))
	if err != nil {
		t.Fatal(err)
	}
	if sceneAfter.Revision != sceneBefore.Revision {
		t.Fatal("patrol must not bump the visual scene revision")
	}

	frames := conn.Frames()[nearBefore:]
	var allDest []byte
	for _, frame := range frames {
		if frame[0] == 0x21 {
			allDest = frame
		}
	}
	if allDest == nil {
		t.Fatalf("ready member received no ServerAllDest, frames=%d", len(frames))
	}
	if count := binary.LittleEndian.Uint16(allDest[1:]); count != 1 {
		t.Fatalf("all-dest count=%d", count)
	}
	if id := binary.LittleEndian.Uint32(allDest[3:]); id != forkNPCID(2) {
		t.Fatalf("all-dest object=%d", id)
	}
	if bits := binary.LittleEndian.Uint32(allDest[19:]); math.Float32frombits(bits) != 10 {
		t.Fatalf("all-dest destZ=%v", math.Float32frombits(bits))
	}
	if bits := binary.LittleEndian.Uint32(allDest[27:]); math.Float32frombits(bits) != 2 {
		t.Fatalf("all-dest speed=%v", math.Float32frombits(bits))
	}
	if flags := binary.LittleEndian.Uint32(allDest[43:]); flags != 1 {
		t.Fatalf("all-dest moveFlags=%d", flags)
	}
	if len(suppressedConn.Frames()) != 0 {
		t.Fatalf("member inside replay window got %d frames", len(suppressedConn.Frames()))
	}
	if len(distantConn.Frames()) != 0 {
		t.Fatalf("member out of viewport got %d frames", len(distantConn.Frames()))
	}

	// 停留拍：速度 0、flags 0（驱动客户端停步动画）。
	nearBefore = len(conn.Frames())
	run.patrolTick(time.UnixMilli(6000))
	for _, frame := range conn.Frames()[nearBefore:] {
		if frame[0] != 0x21 {
			continue
		}
		if bits := binary.LittleEndian.Uint32(frame[27:]); math.Float32frombits(bits) != 0 {
			t.Fatalf("dwelling speed=%v", math.Float32frombits(bits))
		}
		if flags := binary.LittleEndian.Uint32(frame[43:]); flags != 0 {
			t.Fatalf("dwelling moveFlags=%d", flags)
		}
	}
	view.close()
	suppressedView.close()
	distantView.close()
}

// TestRegisterPatrolNPCSamplesSchedule 验证注册时坐标按时间表采样：
// 0x0D 载荷与初始 Transform 落在时间表当前位置而非出生点。
func TestRegisterPatrolNPCSamplesSchedule(t *testing.T) {
	spawn := testNPCSpawn()
	spawn.x, spawn.y, spawn.z, spawn.orient = 0, 0, 0, 0
	spawn.schedule = buildPatrolSchedule(&patrolRoute{pathID: "t", waypoints: []patrolWaypoint{
		{x: 0, y: 0, z: 0},
		{x: 0, y: 0, z: 10, dwellMs: 3000},
	}}, 2, 4, 0)
	run := newSceneRun(sceneKey{config: "scene", resource: "resource"}, nil)
	if err := run.populate([]npcSpawn{spawn}); err != nil {
		t.Fatal(err)
	}
	transform, ok := run.npcTransform(worldcore.EntityID(forkNPCID(2)))
	if !ok {
		t.Fatal("NPC must be registered")
	}
	// 时间表位置必在路线包围盒内（注册采样在真实时刻，只需验证
	// 与出生点解耦：z ∈ [0,10] 且当 z>0 时绝不可能来自 spawn 坐标）。
	if transform.Z < -1e-4 || transform.Z > 10+1e-4 {
		t.Fatalf("registered transform=%+v", transform)
	}
	run.mu.Lock()
	meta, _ := run.npcMeta(worldcore.EntityID(forkNPCID(2)))
	run.mu.Unlock()
	if meta.schedule == nil {
		t.Fatal("registered meta must carry the schedule")
	}
	if len(run.patrolIDs) != 1 || run.patrolIDs[0] != forkNPCID(2) {
		t.Fatalf("patrolIDs=%v", run.patrolIDs)
	}
}
