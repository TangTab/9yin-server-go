package entry

import "sort"

// sceneGrid 复刻 CClients 的场景网格（clients.cpp ProcClientPosi）：
// 自适应 cell 尺寸、行主序 u32 格链表、客户端坐标驱动放置。
type sceneGrid struct {
	x0, y0 int32 // 场景原点（send_to_sight 坐标换算基准）
	width  int32 // 场景宽（网格维度来源，msg+0x20）
	height int32 // 场景高（msg+0x24）
	cell   int32 // 格边长 ∈ {1,2,4,8}
	rows   int   // height/cell
	cols   int   // width/cell
	cells  []int // 每格客户端链表头（clientIndex，-1 空）
}

// pickCell 复刻官方自适应算法：cell 从 1 起 ×2，直到
// max(w,h)/cell ≤ 0x200 或 cell 达 8。
func pickCell(width, height int32) int32 {
	m := width
	if height > m {
		m = height
	}
	cell := int32(1)
	for m/cell > 0x200 && cell < 8 {
		cell *= 2
	}
	return cell
}

func newSceneGrid(x0, y0, width, height int32) *sceneGrid {
	cell := pickCell(width, height)
	g := &sceneGrid{
		x0: x0, y0: y0, width: width, height: height, cell: cell,
		rows: int(height / cell), cols: int(width / cell),
	}
	if g.rows < 1 {
		g.rows = 1
	}
	if g.cols < 1 {
		g.cols = 1
	}
	g.cells = make([]int, g.rows*g.cols)
	for i := range g.cells {
		g.cells[i] = -1
	}
	return g
}

// sameDims 官方在四个维度任一变化时整网格重建。
func (g *sceneGrid) sameDims(x0, y0, width, height int32) bool {
	return g != nil && g.x0 == x0 && g.y0 == y0 && g.width == width && g.height == height
}

// cellIndex 由世界坐标求行主序格下标；越界返回 -1。
func (g *sceneGrid) cellIndex(x, y int32) int {
	if x < g.x0 || y < g.y0 {
		return -1
	}
	col := int((x - g.x0) / g.cell)
	row := int((y - g.y0) / g.cell)
	if col < 0 || col >= g.cols || row < 0 || row >= g.rows {
		return -1
	}
	return row*g.cols + col
}

// sightOffset 螺旋偏移表项（官方 12 字节 {dx,dy,dist}，按 dist 升序 qsort）。
type sightOffset struct {
	dx, dy int32
	dist   float64
}

// sightTableRadius 官方表覆盖 dx,dy ∈ [-127,127]，即 255×255 项。
const sightTableRadius = 127

// buildSightTable 复刻 CClients::CreateSightTable：全量 (dx,dy) 组合，
// dist=√(dx²+dy²)，升序排序（ProcSightSend 依赖顺序做早退）。
func buildSightTable() []sightOffset {
	n := (sightTableRadius*2 + 1) * (sightTableRadius*2 + 1)
	table := make([]sightOffset, 0, n)
	for dx := int32(-sightTableRadius); dx <= sightTableRadius; dx++ {
		for dy := int32(-sightTableRadius); dy <= sightTableRadius; dy++ {
			table = append(table, sightOffset{
				dx: dx, dy: dy,
				dist: float64(dx*dx + dy*dy), // 平方距离，排序等价于官方 √ 值
			})
		}
	}
	sort.Slice(table, func(i, j int) bool { return table[i].dist < table[j].dist })
	return table
}
