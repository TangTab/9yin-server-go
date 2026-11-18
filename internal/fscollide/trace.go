// 行走追踪三件套（scenecollide3d.cpp 内联静态 + 三个 Trace 方法）。
//
// 公共骨架：把 (起点→终点) 直线按碰撞格中心离散成一串采样点
// （calc_line_points：起点所在格中心起步，按"出边方向"逐步挪格，
// 斜向走 L 形三脚），再沿采样点逐格做地板判定。
//
// 出边方向分类器（0x1005B1B0，逐指令复刻；eps=FLT_EPSILON 1.1920929e-7）：
//	dir bit1=西(x-cell) bit2=北(z-cell) bit4=东(x+cell) bit8=南(z+cell)
//	dx<0 时考察西缘：直线近似平行于西缘且起点在格内(ex≤ez-eps 且 ex≤-eps)
//	→ 直接西行；否则两端点均在缘东侧(ex,ez≤0)时求直线与 x=left 交点的 z，
//	∈[top,bottom] → 西行。其余三角对称。
//
// 组合 dir∈{1,2,4,8} 单轴移一格；{3,6,9,12} 斜向（L 形：先横/纵各一格、
// 再斜角一格，共 3 个采样点）；{5,7,10,11} 对直线不可能，官方断言。
package fscollide

import "math"

const traceEps = 1.1920928955078125e-7

// exitDir 复刻方向分类器；坐标均为 f32（比较按 f64 语义）。
// 每个方向的两个分支（反汇编逐条对应）：
//   - 贴缘平行（|ex-ez|≤eps 且 |ex|≤eps）→ 直接该向
//   - 否则两端点均在缘内侧（ex≤0 且 ez≤0）时，求直线延展与该缘所在
//     线的交点，落在对边范围内 → 该向
func exitDir(left, top, right, bottom, x1, z1, x2, z2 float32) int {
	dir := 0
	dx := float64(x2 - x1)
	dz := float64(z2 - z1)
	const e = traceEps
	axis := func(edgeA, edgeB float64, d2 float64, crossBase float32, lo, hi float64, bit int) {
		ex := edgeA
		ez := edgeB
		parallel := ez+e >= ex && ex >= ez-e && e >= ex && ex >= -e
		if parallel {
			dir |= bit
		} else if ex <= 0 && ez <= 0 {
			t := float32(ex/(ex-ez))*float32(d2) + crossBase
			if float64(t) >= lo && float64(t) <= hi {
				dir |= bit
			}
		}
	}
	// 西（dx<0）：缘=left，交点在 z 向
	if dx < 0 {
		axis(float64(left-x1), float64(left-x2), dz, z1, float64(top), float64(bottom), 1)
	} else { // 东：缘=right（ex=x1-right）
		axis(float64(x1-right), float64(x2-right), dz, z1, float64(top), float64(bottom), 4)
	}
	// 北（dz<0）：缘=top，交点在 x 向
	if dz < 0 {
		axis(float64(top-z1), float64(top-z2), dx, x1, float64(left), float64(right), 2)
	} else { // 南：缘=bottom（ex=z1-bottom）
		axis(float64(z1-bottom), float64(z2-bottom), dx, x1, float64(left), float64(right), 8)
	}
	return dir
}

// linePoints 复刻 0x1005A980 格心离散；返回 (x,z) 采样点对（扁平 f32）。
// cap 为原分配上限（浮点数计），超限报错（官方为断言）。
func linePoints(x1, z1, x2, z2, cell, invCell float32, cap int) ([]float32, error) {
	var out []float32
	emit := func(x, z float32) { out = append(out, x, z) }
	cx := int(x1 * invCell) // FloatToInt 截断
	cz := int(z1 * invCell)
	if x1 > 0 {
		cx--
	}
	if z1 > 0 {
		cz--
	}
	curX := (float32(cx) + 0.5) * cell
	curZ := (float32(cz) + 0.5) * cell
	emit(curX, curZ)
	for {
		left := curX - cell*0.5
		top := curZ - cell*0.5
		right := left + cell
		bottom := top + cell
		d := exitDir(left, top, right, bottom, x1, z1, x2, z2)
		if d == 0 {
			break
		}
		switch d {
		case 1:
			curX -= cell
			emit(curX, curZ)
		case 2:
			curZ -= cell
			emit(curX, curZ)
		case 4:
			curX += cell
			emit(curX, curZ)
		case 8:
			curZ += cell
			emit(curX, curZ)
		case 3, 6, 9, 12: // 斜向 L 形：(新x,curZ)(curX,新z)(新x,新z)
			var nx, nz float32
			if d&1 != 0 {
				nx = curX - cell
			} else {
				nx = curX + cell
			}
			if d&2 != 0 {
				nz = curZ - cell
			} else {
				nz = curZ + cell
			}
			emit(nx, curZ)
			emit(curX, nz)
			emit(nx, nz)
			curX, curZ = nx, nz
		default: // 5/7/10/11：直线段不可能的组合（官方断言）
			return nil, errTraceDir
		}
		if len(out) > cap {
			return nil, errTraceCap
		}
	}
	return out, nil
}

type traceError string

func (e traceError) Error() string { return string(e) }

const (
	errTraceDir = traceError("fscollide: trace impossible direction combo")
	errTraceCap = traceError("fscollide: trace points overflow")
)

// traceFloorAt 复刻单采样点的地板判定（三 Trace 共用）：
// 自顶层向下找 bit0 层、|h-lastY|≤step、bit1 可行走；命中返回层号与高度。
// 无可行层返回 ok=false（含 zone/walk 缺失）。
func (s *SceneCollide3D) traceFloorAt(px, pz, lastY, step float32) (floor int, h float32, ok bool) {
	zone := s.GetInZone(px, pz)
	if zone == nil || zone.Walk == nil {
		return -1, 0, false
	}
	col, row := s.zoneCell(zone, px, pz)
	for f := zone.Walk.GetFloorCount() - 1; f >= 0; f-- {
		marker := zone.Walk.GetFloorMarker(row, col, f)
		if marker&1 == 0 {
			continue
		}
		h := zone.Walk.GetFloorHeight(row, col, f)
		if heightIsNull(h) {
			h = zone.Height.GetY(px, pz)
		}
		if lastY+step < h || h < lastY-step {
			continue
		}
		if marker&2 == 0 {
			return -1, 0, false // 存在且高度合适但不可走 → 直接判死（不试低层）
		}
		return f, h, true
	}
	return -1, 0, false
}

// TraceResult 是三个 Trace 的公共输出。
type TraceResult struct {
	OK    bool
	X     float32
	Y     float32
	Z     float32
	Floor int
}

// prepTrace 公共前置：距离闸 + 采样点。ok=false 时 result 已填起点。
func (s *SceneCollide3D) prepTrace(x1, y1, z1, x2, z2 float32) (pts []float32, res TraceResult, ok bool) {
	dxi := int(x2 - x1)
	dzi := int(z2 - z1)
	if dxi < 0 {
		dxi = -dxi
	}
	if dzi < 0 {
		dzi = -dzi
	}
	if dxi+dzi > 0x100 {
		return nil, TraceResult{X: x1, Y: y1, Z: z1}, false
	}
	pts, err := linePoints(x1, z1, x2, z2, s.CellSize, s.InvCell, (dxi+dzi+8)*3)
	if err != nil {
		return nil, TraceResult{X: x1, Y: y1, Z: z1}, false
	}
	if len(pts) < 2 {
		return nil, TraceResult{X: x1, Y: y1, Z: z1}, false
	}
	return pts, TraceResult{X: x1, Y: y1, Z: z1}, true
}

// TraceLineWalk 复刻：沿线可走判定；不可走时输出最后合法点。
func (s *SceneCollide3D) TraceLineWalk(step, x1, y1, z1, x2, y2, z2 float32) TraceResult {
	pts, res, ok := s.prepTrace(x1, y1, z1, x2, z2)
	if !ok {
		return res
	}
	if len(pts) == 2 { // 相邻两格：直达
		return TraceResult{OK: true, X: x2, Y: y1, Z: z2}
	}
	curX, curY, curZ := x1, y1, z1
	lastY := y1
	for i := 2; i < len(pts); i += 2 {
		px, pz := pts[i], pts[i+1]
		f, h, found := s.traceFloorAt(px, pz, lastY, step)
		if !found {
			if i == 2 {
				return TraceResult{X: x1, Y: y1, Z: z1}
			}
			return TraceResult{X: curX, Y: curY, Z: curZ}
		}
		_ = f
		lastY = h
		curX, curY, curZ = px, h, pz
	}
	return TraceResult{OK: true, X: x2, Y: curY, Z: z2}
}

// TraceNextFloor 复刻：沿线下一步地板（跨层/落点报告）。
// 与 TraceLineWalk 的差别：中途地板消失＝成功落点（报告最后合法点），
// 并输出层号；首点即无地板才失败。
func (s *SceneCollide3D) TraceNextFloor(step float32, floorIn int, x1, y1, z1, x2, z2 float32) TraceResult {
	pts, res, ok := s.prepTrace(x1, y1, z1, x2, z2)
	if !ok {
		return res
	}
	if len(pts) == 2 {
		return TraceResult{OK: true, Floor: floorIn, X: x2, Y: y1, Z: z2}
	}
	curX, curY, curZ := x1, y1, z1
	lastY := y1
	curFloor := floorIn
	for i := 2; i < len(pts); i += 2 {
		px, pz := pts[i], pts[i+1]
		f, h, found := s.traceFloorAt(px, pz, lastY, step)
		if !found {
			if i == 2 {
				return TraceResult{X: x1, Y: y1, Z: z1}
			}
			return TraceResult{OK: true, Floor: curFloor, X: curX, Y: curY, Z: curZ}
		}
		lastY = h
		curFloor = f
		curX, curY, curZ = px, h, pz
	}
	return TraceResult{OK: true, Floor: curFloor, X: x2, Y: curY, Z: z2}
}

// TraceNextSlide 复刻：TraceNextFloor + 地形兜底（滑落）。
// 无地板时：地形高 ≤ lastY+step → 落地形；地形高 > lastY+2*step → 失败；
// 其间 → 继续滑（不落）。注意：此函数按官方结构级复刻（含 2×step 滑
// 落窗），细节待 oracle 对拍后收紧。
func (s *SceneCollide3D) TraceNextSlide(step float32, floorIn int, x1, y1, z1, x2, z2 float32) TraceResult {
	pts, res, ok := s.prepTrace(x1, y1, z1, x2, z2)
	if !ok {
		return res
	}
	if len(pts) == 2 {
		return TraceResult{OK: true, Floor: floorIn, X: x2, Y: y1, Z: z2}
	}
	curX, curY, curZ := x1, y1, z1
	lastY := y1
	curFloor := floorIn
	for i := 2; i < len(pts); i += 2 {
		px, pz := pts[i], pts[i+1]
		f, h, found := s.traceFloorAt(px, pz, lastY, step)
		if found {
			lastY = h
			curFloor = f
			curX, curY, curZ = px, h, pz
			continue
		}
		// 地形兜底
		zone := s.GetInZone(px, pz)
		terrain := float32(0)
		if zone != nil && zone.Height != nil {
			terrain = zone.Height.GetY(px, pz)
		}
		if lastY+step >= terrain { // 落地
			lastY = terrain
			curX, curY, curZ = px, terrain, pz
			continue
		}
		if terrain > lastY+2*step { // 悬空过高 → 失败
			if i == 2 {
				return TraceResult{X: x1, Y: y1, Z: z1}
			}
			return TraceResult{OK: true, Floor: curFloor, X: curX, Y: curY, Z: curZ}
		}
		// 滑落窗口：继续向下一采样点
	}
	return TraceResult{OK: true, Floor: curFloor, X: x2, Y: curY, Z: z2}
}

var _ = math.Sqrt // 保留 math 引用（LineCanWalk 使用）
