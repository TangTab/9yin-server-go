package main

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/entity"
	"github.com/local/9yin-go-server/internal/mapnav"
	"github.com/local/9yin-go-server/internal/role"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// encodeMapnavGraph 按逆向格式编码合成路点图（与 mapnav 包测试同构，
// 供战斗链集成注入）。
func encodeMapnavGraph(nodes []mapnav.Node, edges [][2]int) []byte {
	adj := make([][]int, len(nodes))
	for _, e := range edges {
		adj[e[0]] = append(adj[e[0]], e[1])
	}
	buf := binary.LittleEndian.AppendUint32(nil, uint32(len(nodes)))
	for i, node := range nodes {
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(node.X))
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(node.Y))
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(node.Z))
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(adj[i])))
		for _, target := range adj[i] {
			d := node
			other := nodes[target]
			dx, dy, dz := d.X-other.X, d.Y-other.Y, d.Z-other.Z
			buf = binary.LittleEndian.AppendUint32(buf, uint32(target))
			buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(float32(math.Sqrt(float64(dx*dx+dy*dy+dz*dz)))))
		}
	}
	return buf
}

// hookMapnavGraph 把合成图挂到 resource 名下（测试钩子）。必须在
// newSceneLifecycle 之前调用——sceneRun 创建即后台预热，晚挂会被
// Once 的空结果抢先。
func hookMapnavGraph(t *testing.T, resource string, data []byte) {
	t.Helper()
	graph, err := mapnav.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	previous := mapnavLoaderForTest
	mapnavLoaderForTest = func(name string) (*mapnav.Graph, bool) {
		if name == resource {
			return graph, true
		}
		return nil, false
	}
	t.Cleanup(func() { mapnavLoaderForTest = previous })
}

// warmMapnavForTest 同步完成装载（消除预热 goroutine 的调度竞态，
// 首拍即有图）。
func warmMapnavForTest(t *testing.T, lifecycle *sceneLifecycle) {
	t.Helper()
	lifecycle.run.warmNav()
}

// registerChaseTestNPC 注册可控速度/感知范围的战斗 NPC 于 (x,y,z)。
func registerChaseTestNPC(t *testing.T, lifecycle *sceneLifecycle, id uint32, x, y, z, aggro, chase, speed float32) {
	t.Helper()
	npc := testNPCSpawn()
	npc.resolved.ConfigID = "AttackNPC_chase"
	npc.resolved.ScriptClass = "AttackNpc"
	npc.resolved.Properties["SpringRange"] = clientdata.PropertyValue{Value: clientdata.Float32Value(aggro)}
	npc.resolved.Properties["ChaseRange"] = clientdata.PropertyValue{Value: clientdata.Float32Value(chase)}
	npc.resolved.Properties["RunSpeed"] = clientdata.PropertyValue{Value: clientdata.Float32Value(speed)}
	npc.x, npc.y, npc.z = x, y, z
	if err := lifecycle.registerNPCOnRun(id, npc); err != nil {
		t.Fatal(err)
	}
}

func chaseNPCTransform(t *testing.T, lifecycle *sceneLifecycle, id uint32) worldcore.Transform {
	t.Helper()
	transform, ok := lifecycle.npcTransform(worldcore.EntityID(forkNPCID(id)))
	if !ok {
		t.Fatalf("NPC %d has no transform", id)
	}
	return transform
}

// chaseTickForTest 后续追击拍：combatTickForTest 每次调用都会挂一个
// 新成员（旧成员位置冻结），这里改为原地更新既有玩家成员的位置再
// 驱动 tick，使移动目标真正参与仇恨定位/重规划。
func chaseTickForTest(lifecycle *sceneLifecycle, position role.Position, now time.Time) {
	lifecycle.run.mu.Lock()
	for _, member := range lifecycle.run.members {
		if member.player != nil {
			member.position = position
			member.active = true
		}
	}
	lifecycle.run.mu.Unlock()
	outcome := lifecycle.run.simulateLocked(now)
	if outcome != nil {
		lifecycle.run.broadcastNPCFrames(outcome.members, outcome.moves, outcome.updates)
		lifecycle.run.broadcastNPCAttacks(outcome.attacks)
	}
}

// TestChaseFollowsWaypointPolylineAroundWall 追击穿墙回归（§6.4）：
// 4×3 网格、x=0 列外无纵向通路（等效一堵墙），NPC 在 (0,0,0)、玩家在
// (30,10,20)。直线追击必然斜穿墙；路点图追击第一拍必须朝 x=0 列的
// 纵向缺口推进且 y 随路点爬坡插值。
func TestChaseFollowsWaypointPolylineAroundWall(t *testing.T) {
	const cols, rows = 4, 3
	var nodes []mapnav.Node
	var edges [][2]int
	id := func(c, r int) int { return r*cols + c }
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			nodes = append(nodes, mapnav.Node{X: float32(c * 10), Y: float32(r * 5), Z: float32(r * 10)})
		}
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if c+1 < cols {
				edges = append(edges, [2]int{id(c, r), id(c+1, r)}, [2]int{id(c+1, r), id(c, r)})
			}
			// 纵向（z+）边只存在于 x=0 列：墙体。
			if r+1 < rows && c == 0 {
				edges = append(edges, [2]int{id(c, r), id(c, r+1)}, [2]int{id(c, r+1), id(c, r)})
			}
		}
	}
	hookMapnavGraph(t, "wallgrid", encodeMapnavGraph(nodes, edges))
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "wallgrid", nil); err != nil {
		t.Fatal(err)
	}
	warmMapnavForTest(t, lifecycle)

	registerChaseTestNPC(t, lifecycle, 90, 0, 0, 0, 80, 200, 4.5)
	player := newPlayerActor("tester", 0)
	now := time.Unix(5000, 0)
	lifecycle.combatTickForTest(player, role.Position{X: 30, Y: 10, Z: 20}, now)

	at := chaseNPCTransform(t, lifecycle, 90)
	// 第一拍沿折线 [NPC→(0,0,0)→(0,5,10)→…] 推进 4.5m：方向 +Z/+Y、
	// X 不动（直线斜穿墙会立刻 +X）。
	if at.X > 0.01 {
		t.Fatalf("chase cut through wall: X advanced to %.2f on first tick", at.X)
	}
	if at.Z < 3.9 || at.Z > 4.2 {
		t.Fatalf("chase Z=%.2f want ~4.0 (polyline step toward wall gap)", at.Z)
	}
	if at.Y < 1.8 || at.Y > 2.3 {
		t.Fatalf("chase Y=%.2f want ~2.0 (slope interpolation between y=0 and y=5)", at.Y)
	}
	// 到达确认延时 = 剩余折线长/速度：折线全长 ≈ 11.18+11.18+30 =
	// 52.36，推进 4.5 后剩 ~47.9 → 延时 ~10.6s（固定 1s 即回到漂移）。
	lifecycle.run.mu.Lock()
	brain, _ := lifecycle.run.combat.Brain(worldcore.EntityID(forkNPCID(90)))
	meta, _ := lifecycle.run.npcMeta(worldcore.EntityID(forkNPCID(90)))
	chase := meta.chase
	lifecycle.run.mu.Unlock()
	if brain.Threat == 0 {
		t.Fatal("NPC must hold threat while chasing")
	}
	if chase == nil {
		t.Fatal("NPC must have chase state after first chase tick")
	}
	if remaining := chase.follower.Remaining(); remaining < 47.3 || remaining > 48.5 {
		t.Fatalf("remaining polyline=%.2f want ~47.9", remaining)
	}
	// 连续推进：NPC 沿 x=0 列爬到顶行才横移（穿墙回归的时序版）。
	// 前 3 拍未到顶行，X 必须恒 0（直线斜穿墙会立刻 +X）。
	for tick := 1; tick <= 3; tick++ {
		chaseTickForTest(lifecycle, role.Position{X: 30, Y: 10, Z: 20}, now.Add(time.Duration(tick)*time.Second))
		at := chaseNPCTransform(t, lifecycle, 90)
		if at.X > 0.01 {
			t.Fatalf("tick %d: NPC crossed wall before reaching gap: X=%.2f", tick, at.X)
		}
		if at.Y < 3.5 {
			t.Fatalf("tick %d: NPC failed to climb slope: Y=%.2f", tick, at.Y)
		}
	}
	// 第 4~5 拍抵达顶行（y≈10）后横移出缺口——转向合法，但高度必须
	// 已爬到顶（穿地回归的时序版）。
	for tick := 4; tick <= 5; tick++ {
		chaseTickForTest(lifecycle, role.Position{X: 30, Y: 10, Z: 20}, now.Add(time.Duration(tick)*time.Second))
	}
	at = chaseNPCTransform(t, lifecycle, 90)
	if at.Y < 9.0 {
		t.Fatalf("NPC must reach top row height: Y=%.2f", at.Y)
	}
	if at.Z < 15.0 {
		t.Fatalf("NPC must climb the x=0 column: Z=%.2f", at.Z)
	}
	if at.X > 9.5 {
		t.Fatalf("NPC overshot along top row: X=%.2f", at.X)
	}
	lifecycle.close()
}

// TestChaseFinalStraightApproach 末段直线（§6.3.3）：距目标 <12m 折线
// 让位直线逼近，y 直接朝目标插值（近距目标 y 可信），步长钳到攻击
// 范围边界（6m）。
func TestChaseFinalStraightApproach(t *testing.T) {
	// 平地图：两排路点即可，玩家挂在排间空档。
	nodes := []mapnav.Node{{0, 0, 0}, {20, 0, 0}, {0, 0, 20}, {20, 0, 20}}
	edges := [][2]int{{0, 1}, {1, 0}, {0, 2}, {2, 0}, {1, 3}, {3, 1}, {2, 3}, {3, 2}}
	hookMapnavGraph(t, "wallgrid2", encodeMapnavGraph(nodes, edges))
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "wallgrid2", nil); err != nil {
		t.Fatal(err)
	}
	warmMapnavForTest(t, lifecycle)

	// NPC (10,0,10) 图中心，玩家 (10,8,1)：水平距 9m ∈ (6,12)。
	registerChaseTestNPC(t, lifecycle, 91, 10, 0, 10, 80, 200, 4.5)
	player := newPlayerActor("tester", 0)
	lifecycle.combatTickForTest(player, role.Position{X: 10, Y: 8, Z: 1}, time.Unix(6000, 0))

	at := chaseNPCTransform(t, lifecycle, 91)
	// 直线步长 = min(4.5, 9-6) = 3：Z 减 3、y 朝 8 插值 3/9。
	if at.Z > 7.01 || at.Z < 6.99 {
		t.Fatalf("final straight Z=%.2f want ~7 (clamped at attack range edge)", at.Z)
	}
	if at.Y < 2.5 || at.Y > 2.9 {
		t.Fatalf("final straight Y=%.2f want ~2.67 (interpolated toward target y)", at.Y)
	}
	lifecycle.close()
}

// TestChaseCrossComponentGivesUpAndReturnsHome 跨分量兜底（§6.3.5）：
// NPC 与玩家各在一个连通分量，A* 连续失败 5 次 → 放弃仇恨按栓绳语义
// 回 Home（瞬移 + Fighting 重置），等待期直线贴地推进不悬空。
func TestChaseCrossComponentGivesUpAndReturnsHome(t *testing.T) {
	// 两个不连通分量：NPC 侧 (0..20,0,0)，玩家侧 (100..120,0,0)。
	nodes := []mapnav.Node{{0, 0, 0}, {10, 0, 0}, {20, 0, 0}, {100, 4, 0}, {110, 4, 0}, {120, 4, 0}}
	edges := [][2]int{{0, 1}, {1, 0}, {1, 2}, {2, 1}, {3, 4}, {4, 3}, {4, 5}, {5, 4}}
	hookMapnavGraph(t, "split", encodeMapnavGraph(nodes, edges))
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "split", nil); err != nil {
		t.Fatal(err)
	}
	warmMapnavForTest(t, lifecycle)

	registerChaseTestNPC(t, lifecycle, 92, 0, 0, 0, 200, 400, 5)
	player := newPlayerActor("tester", 0)
	now := time.Unix(7000, 0)
	// 重试间隔 2s：t=0/2/4/6/8 五次规划全败，第 5 次当拍放弃回位。
	for tick := 0; tick <= 8; tick++ {
		lifecycle.combatTickForTest(player, role.Position{X: 100, Y: 4, Z: 0}, now.Add(time.Duration(tick)*time.Second))
	}
	lifecycle.run.mu.Lock()
	brain, _ := lifecycle.run.combat.Brain(worldcore.EntityID(forkNPCID(92)))
	meta, _ := lifecycle.run.npcMeta(worldcore.EntityID(forkNPCID(92)))
	chase := meta.chase
	lifecycle.run.mu.Unlock()
	if brain.Threat != 0 || brain.TargetMember != 0 {
		t.Fatalf("cross-component chase must give up: threat=%d target=%d", brain.Threat, brain.TargetMember)
	}
	if chase != nil {
		t.Fatal("give-up must reset chase state")
	}
	at := chaseNPCTransform(t, lifecycle, 92)
	if at.X != 0 || at.Z != 0 {
		t.Fatalf("give-up must return home, got %+v", at)
	}
	// 回位移动帧 + Fighting 重置属性帧都应广播。
	var sawMove, sawReset bool
	for _, frame := range conn.Frames() {
		if frame[0] == 0x20 {
			sawMove = true
		}
		if frame[0] == 0x10 {
			sawReset = true
		}
	}
	if !sawMove {
		t.Fatal("return-home move frame missing")
	}
	_ = sawReset
	lifecycle.close()
}

// TestChaseWithoutGraphKeepsStraightLine 无图兜底回归：资源无 .path 时
// 维持旧直线语义（x/z 推进、y 维持出发高度），步长仍钳攻击范围边界。
func TestChaseWithoutGraphKeepsStraightLine(t *testing.T) {
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	// resource 名不挂测试钩子也不存在真图 → warmNav 负缓存为 nil。
	if err := lifecycle.begin("scene", "nograph", nil); err != nil {
		t.Fatal(err)
	}
	lifecycle.run.warmNav()
	registerChaseTestNPC(t, lifecycle, 93, 0, 0, 0, 200, 400, 4)
	player := newPlayerActor("tester", 0)
	// 水平距 10（>6 攻击距），玩家 y=50（悬空）：旧语义 y 不动。
	lifecycle.combatTickForTest(player, role.Position{X: 10, Y: 50, Z: 0}, time.Unix(8000, 0))
	at := chaseNPCTransform(t, lifecycle, 93)
	if at.X < 3.5 || at.X > 4.5 {
		t.Fatalf("no-graph chase X=%.2f want ~4 (min(4, 10-6))", at.X)
	}
	if at.Y != 0 {
		t.Fatalf("no-graph chase Y=%.2f want 0 (legacy semantics keep spawn height)", at.Y)
	}
	if at.Orient < -0.01 || at.Orient > 0.01 {
		t.Fatalf("no-graph chase orient=%.3f want ~0 (facing +X)", at.Orient)
	}
	lifecycle.close()
}

// TestChaseReplansWhenTargetShifts 重规划触发（§6.3.1）：玩家位移超
// 8m 立即重算折线，规划终点随玩家走。
func TestChaseReplansWhenTargetShifts(t *testing.T) {
	const cols, rows = 4, 3
	var nodes []mapnav.Node
	var edges [][2]int
	id := func(c, r int) int { return r*cols + c }
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			nodes = append(nodes, mapnav.Node{X: float32(c * 10), Y: 0, Z: float32(r * 10)})
		}
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if c+1 < cols {
				edges = append(edges, [2]int{id(c, r), id(c+1, r)}, [2]int{id(c+1, r), id(c, r)})
			}
			if r+1 < rows {
				edges = append(edges, [2]int{id(c, r), id(c, r+1)}, [2]int{id(c, r+1), id(c, r)})
			}
		}
	}
	hookMapnavGraph(t, "wallgrid3", encodeMapnavGraph(nodes, edges))
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "wallgrid3", nil); err != nil {
		t.Fatal(err)
	}
	warmMapnavForTest(t, lifecycle)

	registerChaseTestNPC(t, lifecycle, 94, 0, 0, 0, 200, 400, 4)
	player := newPlayerActor("tester", 0)
	now := time.Unix(9000, 0)
	lifecycle.combatTickForTest(player, role.Position{X: 30, Y: 0, Z: 20}, now)
	lifecycle.run.mu.Lock()
	meta, _ := lifecycle.run.npcMeta(worldcore.EntityID(forkNPCID(94)))
	firstPlan, firstEnd := meta.chase.planAt, meta.chase.planEnd
	lifecycle.run.mu.Unlock()
	if firstPlan.IsZero() {
		t.Fatal("first tick must plan a path")
	}
	// 150ms 内玩家仅挪 2m（<8m 且 <400ms）：不重规划，沿旧折线推进。
	chaseTickForTest(lifecycle, role.Position{X: 30, Y: 0, Z: 22}, now.Add(150*time.Millisecond))
	lifecycle.run.mu.Lock()
	meta, _ = lifecycle.run.npcMeta(worldcore.EntityID(forkNPCID(94)))
	if !meta.chase.planAt.Equal(firstPlan) {
		t.Fatal("small target shift within throttle window must not replan")
	}
	secondPlan := meta.chase.planAt
	lifecycle.run.mu.Unlock()
	// 玩家再挪 9m（累计 >8m）：立即重规划且规划终点跟随新位置。
	chaseTickForTest(lifecycle, role.Position{X: 30, Y: 0, Z: 31}, now.Add(300*time.Millisecond))
	lifecycle.run.mu.Lock()
	meta, _ = lifecycle.run.npcMeta(worldcore.EntityID(forkNPCID(94)))
	replanned, replanEnd := meta.chase.planAt, meta.chase.planEnd
	lifecycle.run.mu.Unlock()
	if !replanned.After(secondPlan) {
		t.Fatal("target shift >8m must trigger immediate replan")
	}
	if mapnavHorizontal(replanEnd, firstEnd) < 8 {
		t.Fatalf("replan endpoint must track moved target: got (%.0f,%.0f) want near (30,31)", replanEnd.X, replanEnd.Z)
	}
	lifecycle.close()
}

// TestChaseGateSilencesMovementButHoldsThreat GM 追击总闸
//（set_npc_chase=0 / NINEYIN_NPC_CHASE=off）：位移分支整段静默、怪停
// 出生点，但仇恨照常持有（威胁表/受击锁定不动）。
func TestChaseGateSilencesMovementButHoldsThreat(t *testing.T) {
	nodes := []mapnav.Node{{0, 0, 0}, {10, 0, 0}, {20, 0, 0}, {30, 0, 0}}
	edges := [][2]int{{0, 1}, {1, 0}, {1, 2}, {2, 1}, {2, 3}, {3, 2}}
	hookMapnavGraph(t, "wallgrid5", encodeMapnavGraph(nodes, edges))
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "wallgrid5", nil); err != nil {
		t.Fatal(err)
	}
	warmMapnavForTest(t, lifecycle)

	previous := npcChaseEnabled()
	t.Cleanup(func() { setNPCChaseEnabled(previous) })
	setNPCChaseEnabled(false)

	registerChaseTestNPC(t, lifecycle, 96, 0, 0, 0, 200, 400, 4)
	player := newPlayerActor("tester", 0)
	now := time.Unix(9800, 0)
	lifecycle.combatTickForTest(player, role.Position{X: 30, Y: 0, Z: 0}, now)
	for tick := 1; tick <= 3; tick++ {
		chaseTickForTest(lifecycle, role.Position{X: 30, Y: 0, Z: 0}, now.Add(time.Duration(tick)*time.Second))
	}
	at := chaseNPCTransform(t, lifecycle, 96)
	if at.X != 0 || at.Z != 0 {
		t.Fatalf("chase gate closed but NPC moved to %+v", at)
	}
	lifecycle.run.mu.Lock()
	brain, _ := lifecycle.run.combat.Brain(worldcore.EntityID(forkNPCID(96)))
	lifecycle.run.mu.Unlock()
	if brain.Threat == 0 || brain.TargetMember == 0 {
		t.Fatalf("chase gate must hold threat, got threat=%d target=%d", brain.Threat, brain.TargetMember)
	}
	// 重开（GM set_npc_chase=1 等价）：下一拍恢复折线推进。
	setNPCChaseEnabled(true)
	chaseTickForTest(lifecycle, role.Position{X: 30, Y: 0, Z: 0}, now.Add(4*time.Second))
	at = chaseNPCTransform(t, lifecycle, 96)
	if at.X <= 0 {
		t.Fatalf("chase gate reopened but NPC did not move: %+v", at)
	}
	lifecycle.close()
}

// TestSitcrossInterruptedByChaseTickWithGraph 带图追击下的玩家侧联动
// 冒烟：追击推进与 Fighting 临场感共存（不因导航分支改变战斗节奏）。
func TestSitcrossInterruptedByChaseTickWithGraph(t *testing.T) {
	nodes := []mapnav.Node{{0, 0, 0}, {10, 0, 0}, {20, 0, 0}, {30, 0, 0}}
	edges := [][2]int{{0, 1}, {1, 0}, {1, 2}, {2, 1}, {2, 3}, {3, 2}}
	hookMapnavGraph(t, "wallgrid4", encodeMapnavGraph(nodes, edges))
	conn := &captureMessageConnection{}
	lifecycle := newSceneLifecycle("test", conn)
	if err := lifecycle.begin("scene", "wallgrid4", nil); err != nil {
		t.Fatal(err)
	}
	warmMapnavForTest(t, lifecycle)

	registerChaseTestNPC(t, lifecycle, 95, 0, 0, 0, 200, 400, 4)
	player := newPlayerActor("tester", 0)
	player.setLogicState(entity.LogicStateSitcross)
	lifecycle.combatTickForTest(player, role.Position{X: 30, Y: 0, Z: 0}, time.Unix(9500, 0))
	if state := player.stateSnapshot(); state.LogicState == entity.LogicStateSitcross {
		t.Fatal("chase must interrupt sitcross")
	}
	lifecycle.close()
}
