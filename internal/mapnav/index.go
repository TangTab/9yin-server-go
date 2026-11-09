package mapnav

import (
	"math"
	"path/filepath"
	"sync"
)

// gridCellMeters 是 XZ 均匀网格的格边长（docs/map-path-waypoint-graph.md
// §6.1 建议 32m）。路点间距 ~10m，一格约含 9~11 个路点，最近点查询
// 只需扫中心格加一环邻格。
const gridCellMeters = float32(32)

// gridIndex 把路点按 (x,z) 均匀分格入桶。查询用环形扩张：从查询点
// 所在格出发逐环外扩，环的水平下界距离超过当前最优 3D 距离即停——
// 下界是纯水平量，恒 ≤ 3D 距离，剪枝不会漏掉多楼层图的正确层。
type gridIndex struct {
	cell   float32
	minX   float32
	minZ   float32
	cols   int
	rows   int
	bucketStart []int32 // (rows+1)*(cols+1) 前缀，桶按行主序
	bucketItems []int32 // 桶内节点下标（按节点序稳定）
}

func buildGridIndex(nodes []Node) *gridIndex {
	if len(nodes) == 0 {
		return &gridIndex{}
	}
	minX, maxX := nodes[0].X, nodes[0].X
	minZ, maxZ := nodes[0].Z, nodes[0].Z
	for _, node := range nodes[1:] {
		if node.X < minX {
			minX = node.X
		}
		if node.X > maxX {
			maxX = node.X
		}
		if node.Z < minZ {
			minZ = node.Z
		}
		if node.Z > maxZ {
			maxZ = node.Z
		}
	}
	cell := gridCellMeters
	index := &gridIndex{
		cell:   cell,
		minX:   minX - cell,
		minZ:   minZ - cell,
		cols:   int((maxX-minX)/cell) + 3,
		rows:   int((maxZ-minZ)/cell) + 3,
	}
	// 节点序入桶（counting sort 前缀和），桶内下标天然升序，查询
	// 结果确定可复现。
	counts := make([]int32, index.cols*index.rows+1)
	cells := make([]int32, len(nodes))
	for i, node := range nodes {
		c := index.cellOf(node.X, node.Z)
		cells[i] = c
		counts[c+1]++
	}
	for c := 0; c < index.cols*index.rows; c++ {
		counts[c+1] += counts[c]
	}
	index.bucketStart = counts
	index.bucketItems = make([]int32, len(nodes))
	cursor := make([]int32, index.cols*index.rows)
	copy(cursor, counts[:index.cols*index.rows])
	for i, c := range cells {
		index.bucketItems[cursor[c]] = int32(i)
		cursor[c]++
	}
	return index
}

// cellOf 返回坐标所在格的行主序下标（含外扩一圈的边界格）。
func (g *gridIndex) cellOf(x, z float32) int32 {
	col := int((x - g.minX) / g.cell)
	row := int((z - g.minZ) / g.cell)
	if col < 0 {
		col = 0
	} else if col >= g.cols {
		col = g.cols - 1
	}
	if row < 0 {
		row = 0
	} else if row >= g.rows {
		row = g.rows - 1
	}
	return int32(row*g.cols + col)
}

// Nearest 返回距离 p 最近的路点下标（3D 距离，含高度——多层建筑图
// 上水平最近点可能在别的楼层）。空图返回 -1。
func (g *Graph) Nearest(p Node) int {
	if len(g.nodes) == 0 {
		return -1
	}
	best := -1
	var bestDist float64
	idx := g.index
	col := int((p.X - idx.minX) / idx.cell)
	row := int((p.Z - idx.minZ) / idx.cell)
	if col < 0 {
		col = 0
	} else if col >= idx.cols {
		col = idx.cols - 1
	}
	if row < 0 {
		row = 0
	} else if row >= idx.rows {
		row = idx.rows - 1
	}
	// maxRing 是仍可能覆盖非空格的最大环：超出即整图扫尽。
	maxRing := col
	if c := idx.cols - 1 - col; c > maxRing {
		maxRing = c
	}
	if r := row; r > maxRing {
		maxRing = r
	}
	if r := idx.rows - 1 - row; r > maxRing {
		maxRing = r
	}
	scanCell := func(c, r int) {
		if c < 0 || c >= idx.cols || r < 0 || r >= idx.rows {
			return
		}
		for e := idx.bucketStart[r*idx.cols+c]; e < idx.bucketStart[r*idx.cols+c+1]; e++ {
			n := g.nodes[idx.bucketItems[e]]
			dx := float64(p.X - n.X)
			dy := float64(p.Y - n.Y)
			dz := float64(p.Z - n.Z)
			if d := dx*dx + dy*dy + dz*dz; best < 0 || d < bestDist {
				bestDist, best = d, int(idx.bucketItems[e])
			}
		}
	}
	scanCell(col, row)
	// 逐环外扩：环 ring 上任意点距查询格 ≥ (ring-1)*cell（水平量，
	// 恒 ≤ 3D 距离），其平方超过当前最优 3D 距离平方即可整环剪掉。
	for ring := 1; ring <= maxRing; ring++ {
		if best >= 0 {
			if bound := float64((float32(ring) - 1) * idx.cell); bound*bound >= bestDist {
				break
			}
		}
		for c := col - ring; c <= col+ring; c++ {
			scanCell(c, row-ring)
			scanCell(c, row+ring)
		}
		for r := row - ring + 1; r <= row+ring-1; r++ {
			scanCell(col-ring, r)
			scanCell(col+ring, r)
		}
	}
	return best
}

// Library 是按资源名缓存路点图的场景级装载器。Graph 解析后不可变，
// 多场景运行时共享同实例；加载失败负缓存（文件缺失/坏文件不重试）。
// 预算参考：scene14（16.7 万节点/124 万边）解析 ~百毫秒级、内存 ~30MB。
type Library struct {
	mu     sync.Mutex
	dir    string
	loader func(name string) (*Graph, error)
	graphs map[string]*Graph
}

// Dir 返回装载器的 .path 目录（诊断用：目录缺失即部署未带地图资源）。
func (l *Library) Dir() string {
	if l == nil {
		return ""
	}
	return l.dir
}

// NewLibrary 创建以 dir 为 .path 目录的装载器（文件名 <name>.path）。
func NewLibrary(dir string) *Library {
	return &Library{
		dir: dir,
		loader: func(name string) (*Graph, error) {
			return ParseFile(filepath.Join(dir, name+".path"))
		},
		graphs: map[string]*Graph{},
	}
}

// Graph 返回资源名对应的路点图；无图（文件不存在/解析失败/空名）返回
// nil, false。结果缓存，缺失只探一次磁盘。
func (l *Library) Graph(name string) (*Graph, bool) {
	if l == nil || name == "" {
		return nil, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if g, ok := l.graphs[name]; ok {
		if g == nil {
			return nil, false
		}
		return g, true
	}
	g, err := l.loader(name)
	if err != nil {
		// 负缓存：无图场景每 tick 都会问，绝不反复打磁盘/日志。
		l.graphs[name] = nil
		return nil, false
	}
	l.graphs[name] = g
	return g, true
}

// horizontalDistance 返回两点的 XZ 平面距离。
func horizontalDistance(a, b Node) float32 {
	dx, dz := a.X-b.X, a.Z-b.Z
	return float32(math.Hypot(float64(dx), float64(dz)))
}

// distance3D 返回两点欧氏距离（与边权同量纲）。
func distance3D(a, b Node) float32 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return float32(math.Sqrt(float64(dx)*float64(dx) + float64(dy)*float64(dy) + float64(dz)*float64(dz)))
}
