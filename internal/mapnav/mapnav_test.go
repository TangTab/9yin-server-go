package mapnav

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// encodeGraph 按逆向格式（docs/map-path-waypoint-graph.md §2）编码一张
// 合成图：int32 节点数，逐节点 float32 x/y/z + int32 边数，逐边
// int32 目标 + float32 欧氏距离。
func encodeGraph(t *testing.T, nodes []Node, edges [][2]int) []byte {
	t.Helper()
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
			buf = binary.LittleEndian.AppendUint32(buf, uint32(target))
			buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(distance3D(node, nodes[target])))
		}
	}
	return buf
}

func TestParseEmptyGraph(t *testing.T) {
	// battle02_main.path 实测 4 字节（N=0）——空图必须合法。
	g, err := Parse([]byte{0, 0, 0, 0})
	if err != nil {
		t.Fatalf("empty graph rejected: %v", err)
	}
	if g.NodeCount() != 0 || g.EdgeCount() != 0 || g.Nearest(Node{}) != -1 {
		t.Fatalf("empty graph state wrong: nodes=%d edges=%d", g.NodeCount(), g.EdgeCount())
	}
}

func TestParseRoundTripAndValidation(t *testing.T) {
	nodes := []Node{{0, 0, 0}, {10, 2, 0}, {10, 4, 10}, {0, 6, 10}}
	edges := [][2]int{{0, 1}, {1, 0}, {1, 2}, {2, 1}, {2, 3}, {3, 2}}
	g, err := Parse(encodeGraph(t, nodes, edges))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if g.NodeCount() != 4 || g.EdgeCount() != 6 {
		t.Fatalf("counts nodes=%d edges=%d want 4/6", g.NodeCount(), g.EdgeCount())
	}
	if mismatches, maxDeviation := g.ValidateEdgeDistances(1e-4); mismatches != 0 || maxDeviation > 1e-4 {
		t.Fatalf("validation mismatches=%d maxDev=%f", mismatches, maxDeviation)
	}
	if got := g.NodeAt(2); got != (Node{10, 4, 10}) {
		t.Fatalf("node 2 = %+v", got)
	}
}

func TestParseRejectsMalformed(t *testing.T) {
	cases := map[string][]byte{
		"truncated header": {0, 0},
		"node count overflow": func() []byte {
			b := binary.LittleEndian.AppendUint32(nil, 1<<20)
			return b
		}(),
		"truncated node": func() []byte {
			b := binary.LittleEndian.AppendUint32(nil, 1)
			b = append(b, 0, 0, 0, 0, 0, 0) // 缺 z + 边数
			return b
		}(),
		"target out of range": func() []byte {
			b := binary.LittleEndian.AppendUint32(nil, 1)
			b = binary.LittleEndian.AppendUint32(b, 0)  // x
			b = binary.LittleEndian.AppendUint32(b, 0)  // y
			b = binary.LittleEndian.AppendUint32(b, 0)  // z
			b = binary.LittleEndian.AppendUint32(b, 1)  // 1 条边
			b = binary.LittleEndian.AppendUint32(b, 99) // 目标越界
			b = binary.LittleEndian.AppendUint32(b, 0)  // dist
			return b
		}(),
		"trailing bytes": func() []byte {
			b := encodeGraph(t, []Node{{0, 0, 0}}, nil)
			return append(b, 0xDE, 0xAD)
		}(),
	}
	for name, data := range cases {
		if _, err := Parse(data); err == nil {
			t.Errorf("%s: accepted malformed input", name)
		}
	}
}

// 真实文件样本（F 盘部署资源抽取）：字节精确 + 距离校验 + 索引规模
// 与逆向报告（§2.1/§3）同数量级。
func TestParseRealSamples(t *testing.T) {
	cases := []struct {
		name  string
		nodes int
	}{
		{"adv031.path", 513},
		{"adv031_main.path", 32},
		{"battle02.path", 33},
		{"battle02_main.path", 0},
		{"scene04.path", 6133},
		{"school01.path", 15271},
	}
	for _, c := range cases {
		data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mapnav", c.name))
		if err != nil {
			t.Fatalf("read %s: %v", c.name, err)
		}
		g, err := Parse(data)
		if err != nil {
			t.Fatalf("parse %s: %v", c.name, err)
		}
		if g.NodeCount() != c.nodes {
			t.Errorf("%s nodes=%d want %d", c.name, g.NodeCount(), c.nodes)
		}
		// 数学级校验：存储边距与坐标欧氏距离一致。官方文件实测最大
		// 偏差 ~0.033m（原工具量化的系统性误差，135 万边无一超 0.1m），
		// 容差取 0.1m——位级解释错误的偏差会是米级而非厘米级。
		if mismatches, maxDeviation := g.ValidateEdgeDistances(0.1); mismatches != 0 {
			t.Errorf("%s distance mismatches=%d maxDev=%.4f", c.name, mismatches, maxDeviation)
		}
	}
}

func TestNearestPrefersCorrectFloor(t *testing.T) {
	// 多层图：同 x/z 两层路点，3D 最近必须命中同层。
	nodes := []Node{
		{0, 0, 0}, {10, 0, 0},
		{0, 20, 0}, {10, 20, 0}, // 二层
	}
	g, err := Parse(encodeGraph(t, nodes, [][2]int{{0, 1}, {1, 0}, {2, 3}, {3, 2}}))
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Nearest(Node{X: 9, Y: 19, Z: 1}); got != 3 {
		t.Fatalf("nearest = %d want 3 (upper floor)", got)
	}
	if got := g.Nearest(Node{X: 9, Y: 1, Z: 1}); got != 1 {
		t.Fatalf("nearest = %d want 1 (ground floor)", got)
	}
	// 查询点远在图界之外也必须返回最近者（环形扩张越界剪枝不误杀）。
	if got := g.Nearest(Node{X: 500, Y: 0, Z: 500}); got < 0 {
		t.Fatalf("nearest outside bounds = %d want >=0", got)
	}
}

func TestAStarGridDetour(t *testing.T) {
	// 4×3 网格，中竖墙断开 1..2 列间全部通路只留顶行缺口：
	// 直线穿墙必须被 A* 绕行。
	const cols, rows = 4, 3
	var nodes []Node
	var edges [][2]int
	id := func(c, r int) int { return r*cols + c }
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			nodes = append(nodes, Node{X: float32(c * 10), Y: float32(r), Z: float32(r * 10)})
		}
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			for _, d := range [][2]int{{1, 0}, {0, 1}} {
				nc, nr := c+d[0], r+d[1]
				if nc >= cols || nr >= rows {
					continue
				}
				if nc == 1 && c == 0 && r != 0 {
					continue // 墙：仅顶行 (0,0)-(1,0) 可通
				}
				edges = append(edges, [2]int{id(c, r), id(nc, nr)}, [2]int{id(nc, nr), id(c, r)})
			}
		}
	}
	g, err := Parse(encodeGraph(t, nodes, edges))
	if err != nil {
		t.Fatal(err)
	}
	p := NewPlanner(g)
	path := p.ShortestPath(id(0, 2), id(3, 2)) // 左下 → 右下
	if path == nil {
		t.Fatal("path must exist around wall")
	}
	if path[0] != id(0, 2) || path[len(path)-1] != id(3, 2) {
		t.Fatalf("path endpoints wrong: %v", path)
	}
	// 路径长 ≥ 欧氏距离；且路径不穿过墙体边（(0,1)-(1,1) 等不存在边）。
	length := PathLength(g, path)
	straight := distance3D(nodes[id(0, 2)], nodes[id(3, 2)])
	if length < straight {
		t.Fatalf("path length %.2f < straight %.2f", length, straight)
	}
	for i := 1; i < len(path); i++ {
		a, b := nodes[path[i-1]], nodes[path[i]]
		if a.X == 0 && b.X == 10 && a.Z != 0 && b.Z != 0 {
			t.Fatalf("path crosses wall at %v→%v", a, b)
		}
	}
	// 重复查询复用工作区（代章懒失效）结果一致。
	again := p.ShortestPath(id(0, 2), id(3, 2))
	if len(again) != len(path) {
		t.Fatalf("repeated query differs: %d vs %d", len(again), len(path))
	}
}

func TestAStarUnreachableReturnsNil(t *testing.T) {
	g, err := Parse(encodeGraph(t,
		[]Node{{0, 0, 0}, {10, 0, 0}, {100, 0, 100}},
		[][2]int{{0, 1}, {1, 0}}))
	if err != nil {
		t.Fatal(err)
	}
	p := NewPlanner(g)
	if path := p.ShortestPath(0, 1); path == nil {
		t.Fatal("connected pair must have a path")
	}
	if path := p.ShortestPath(0, 2); path != nil {
		t.Fatalf("cross-component path = %v want nil", path)
	}
	if path := p.ShortestPath(1, 1); len(path) != 1 || path[0] != 1 {
		t.Fatalf("self path = %v", path)
	}
	if path := p.ShortestPath(-1, 0); path != nil {
		t.Fatal("out of range must return nil")
	}
}

// TestAStarRandomPairsOnRealGraph 对应 §6.4：随机起终点对断言
// 存在率 ≥ 主连通分量占比、路径长 ≥ 欧氏距离。school01 是 439 分量、
// 最大分量 53% 的难样本（报告 §3），正好压测兜底语义。
func TestAStarRandomPairsOnRealGraph(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mapnav", "school01.path"))
	if err != nil {
		t.Fatal(err)
	}
	g, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	// 连通分量占比（BFS 一遍）。
	component := make([]int32, g.NodeCount())
	var sizes []int
	for seed := 0; seed < g.NodeCount(); seed++ {
		if component[seed] != 0 {
			continue
		}
		label := int32(len(sizes) + 1)
		size := 0
		queue := []int{seed}
		component[seed] = label
		for len(queue) > 0 {
			at := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			size++
			for e := g.edgeStart[at]; e < g.edgeStart[at+1]; e++ {
				if next := g.edgeTo[e]; component[next] == 0 {
					component[next] = label
					queue = append(queue, int(next))
				}
			}
		}
		sizes = append(sizes, size)
	}
	largest := 0
	for _, s := range sizes {
		if s > largest {
			largest = s
		}
	}
	share := float64(largest) / float64(g.NodeCount())
	// 随机点对的成功概率是 Σ(分量占比)²（起终点须同分量）；这既是
	// 主分量占比的下界（share²），也是无偏差的期望值。
	expected := 0.0
	for _, s := range sizes {
		expected += float64(s) / float64(g.NodeCount()) * float64(s) / float64(g.NodeCount())
	}

	random := newSplitMix64(0x5EED)
	p := NewPlanner(g)
	const pairs = 500
	existing := 0
	for i := 0; i < pairs; i++ {
		from, to := int(random.next()%uint64(g.NodeCount())), int(random.next()%uint64(g.NodeCount()))
		path := p.ShortestPath(from, to)
		if component[from] != component[to] {
			if path != nil {
				t.Fatalf("pair %d cross-component yielded a path", i)
			}
			continue
		}
		if path == nil {
			t.Fatalf("pair %d same-component (%d=%d) has no path", i, component[from], component[to])
		}
		existing++
		if length := PathLength(g, path); length < distance3D(g.NodeAt(from), g.NodeAt(to))-0.05 {
			t.Fatalf("pair %d path %.2f shorter than straight line", i, length)
		}
	}
	if rate := float64(existing) / pairs; rate < expected*0.9 || rate < share*share*0.9 {
		t.Fatalf("success rate %.3f below expectation %.3f (share=%.3f)", rate, expected, share)
	}
	t.Logf("school01 pairs=%d success=%d expected=%.3f largestComponent=%.3f (%d components)",
		pairs, existing, expected, share, len(sizes))
}

type splitMix64 struct{ state uint64 }

func newSplitMix64(seed uint64) *splitMix64 { return &splitMix64{state: seed} }

func (s *splitMix64) next() uint64 {
	s.state += 0x9E3779B97F4A7C15
	z := s.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

func TestFollowerAdvancesWithYInterpolation(t *testing.T) {
	// 坡地折线：每段爬升，推进后 y 必须落在两路点高度的线性插值上，
	// 无跳变（§6.4 跟随测试）。
	pts := []Node{{0, 0, 0}, {10, 5, 0}, {20, 10, 0}, {30, 10, 0}}
	f := &Follower{}
	f.SetPath(pts)
	// 段长 11.18+11.18+10 = 32.36（y 计入折线长度）。
	if got := f.Remaining(); got < 32.3 || got > 32.4 {
		t.Fatalf("remaining = %.2f want ~32.36", got)
	}
	prev := f.Peek().Y
	for step := 0; step < 20 && !f.Done(); step++ {
		at := f.Advance(2)
		if at.Y < prev-0.001 || at.Y > prev+1.01 {
			t.Fatalf("y jumped: %.3f → %.3f", prev, at.Y)
		}
		if at.X < -0.001 || at.X > 30.001 {
			t.Fatalf("x out of polyline: %.2f", at.X)
		}
		prev = at.Y
	}
	if !f.Done() {
		t.Fatal("20×2m must finish a ~41m polyline")
	}
	final := f.Advance(5)
	if final != (Node{30, 10, 0}) {
		t.Fatalf("over-advance = %+v want endpoint", final)
	}
}

func TestFollowerDegeneratePaths(t *testing.T) {
	var f Follower
	f.SetPath(nil)
	if got := f.Advance(5); got != (Node{}) {
		t.Fatalf("empty path advance = %+v", got)
	}
	f.SetPath([]Node{{1, 2, 3}})
	if got := f.Advance(5); got != (Node{1, 2, 3}) {
		t.Fatalf("single point advance = %+v", got)
	}
	// 零长段（重复点）不除零。
	f.SetPath([]Node{{1, 2, 3}, {1, 2, 3}, {4, 2, 3}})
	if got := f.Advance(1); got.X < 0.99 || got.X > 2.01 {
		t.Fatalf("zero-length segment advance = %+v", got)
	}
}

func TestStraightSteps(t *testing.T) {
	graph, err := Parse(encodeGraph(t, []Node{{0, 0, 0}, {10, 0, 0}}, [][2]int{{0, 1}, {1, 0}}))
	if err != nil {
		t.Fatal(err)
	}
	// 无图：维持旧 stepToward 语义 + y 向目标插值。
	got := LerpOrientFreeStep(Node{0, 0, 0}, Node{10, 4, 0}, 2.5)
	if got != (Node{2.5, 1, 0}) {
		t.Fatalf("lerp step = %+v", got)
	}
	// 贴地：目标 y 乱值（悬空）也不被带走——向地面高度渐进。
	grounded := StepTowardGround(graph, Node{0, 0, 0}, Node{10, 50, 0}, 2.5)
	if grounded.Y > 1.0 {
		t.Fatalf("grounded step y=%.2f drifted toward airborne target", grounded.Y)
	}
	if grounded.Y < -0.001 {
		t.Fatalf("grounded step y=%.2f sank below ground", grounded.Y)
	}
	// 负步长钳 0：只贴地不倒退（攻防分位 3D 化后水平可先于步长归零）。
	clamped := StepTowardGround(graph, Node{0, 3, 0}, Node{10, 50, 0}, -2)
	if clamped.X != 0 || clamped.Y > 3.001 || clamped.Y < 0 {
		t.Fatalf("negative step = %+v want x frozen, y eased toward ground", clamped)
	}
	if o := OrientToward(Node{0, 0, 0}, Node{0, 0, 10}); math.Abs(float64(o-math.Pi/2)) > 1e-5 {
		t.Fatalf("orient = %f want pi/2", o)
	}
}

// TestGroundFollowStepClampsVertical 悬空目标回归（追击飞天/瞬移根因）：
// x/z 照常推进，y 限速向落点地面收敛、绝不朝目标 y 插值；零水平步长
// （step<=0 钳 0）时 y 仍按 vstep 落地/爬出，不依赖水平推进。
func TestGroundFollowStepClampsVertical(t *testing.T) {
	graph, err := Parse(encodeGraph(t, []Node{{0, 0, 0}, {10, 0, 0}}, [][2]int{{0, 1}, {1, 0}}))
	if err != nil {
		t.Fatal(err)
	}
	// 目标悬空 y=30：x/z 推进 2.5，y 已在地面则纹丝不动。
	got := GroundFollowStep(graph, Node{0, 0, 0}, Node{10, 30, 0}, 2.5, 4.5)
	if got != (Node{2.5, 0, 0}) {
		t.Fatalf("ground-follow step = %+v want {2.5 0 0}", got)
	}
	// 出发悬空（异常态）+ 零水平步长：y 仍按 vstep=4.5 限速落地。
	settling := GroundFollowStep(graph, Node{10, 12, 0}, Node{10, 30, 0}, -1, 4.5)
	if settling.X != 10 || settling.Z != 0 {
		t.Fatalf("zero step must freeze x/z: %+v", settling)
	}
	if settling.Y < 7.4 || settling.Y > 7.6 {
		t.Fatalf("settling y=%.2f want 7.5 (12-4.5 clamped descent)", settling.Y)
	}
	// 地面在上方（坑底）：限速爬升而非一步登天。
	climbing := GroundFollowStep(graph, Node{0, -6, 0}, Node{10, -6, 0}, 0, 4.5)
	if climbing.X != 0 {
		t.Fatalf("zero step must freeze x: %+v", climbing)
	}
	if climbing.Y < -1.6 || climbing.Y > -1.4 {
		t.Fatalf("climbing y=%.2f want -1.5 (-6+4.5 clamped climb)", climbing.Y)
	}
	// 无图退化为旧语义：y 维持出发高度。
	legacy := GroundFollowStep(nil, Node{0, 5, 0}, Node{10, 30, 0}, 2.5, 4.5)
	if legacy != (Node{2.5, 5, 0}) {
		t.Fatalf("no-graph ground-follow = %+v want y kept at 5", legacy)
	}
}

func TestLibraryCachesAndNegativeCaches(t *testing.T) {
	dir := t.TempDir()
	// 写一个合法图文件；另一个名字故意缺失。
	data := encodeGraph(t, []Node{{0, 0, 0}}, nil)
	if err := os.WriteFile(filepath.Join(dir, "have.path"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	lib := NewLibrary(dir)
	calls := 0
	inner := lib.loader
	lib.loader = func(name string) (*Graph, error) {
		calls++
		return inner(name)
	}
	have, ok := lib.Graph("have")
	if !ok || have.NodeCount() != 1 {
		t.Fatalf("have graph ok=%t nodes=%d", ok, have.NodeCount())
	}
	if again, ok := lib.Graph("have"); !ok || again != have {
		t.Fatalf("library did not cache: %p vs %p", again, have)
	}
	if _, ok := lib.Graph("missing"); ok {
		t.Fatal("missing graph must not resolve")
	}
	if _, ok := lib.Graph("missing"); ok {
		t.Fatal("missing graph must not resolve on retry")
	}
	if calls != 2 {
		t.Fatalf("loader called %d times want 2 (have+missing once each, negative cache)", calls)
	}
	var nilLib *Library
	if _, ok := nilLib.Graph("have"); ok {
		t.Fatal("nil library must resolve nothing")
	}
	if _, ok := lib.Graph(""); ok {
		t.Fatal("empty name must resolve nothing")
	}
}
