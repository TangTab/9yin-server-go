package mapnav

import "math"

// Follower 是一条世界坐标折线的进给器：沿折线推进定长步距，y 取相邻
// 路点高度的线性插值（追击 NPC 的钻地/悬空根因即旧实现只插值 X/Z）。
// 折线点 0 是规划起点（NPC 当前位置），末点通常是追击目标位置。
type Follower struct {
	pts    []Node
	offset float32 // 已沿折线推进的距离
	length float32
}

// SetPath 装入新折线（接管切片所有权）并重置进给。
func (f *Follower) SetPath(pts []Node) {
	f.pts = pts
	f.offset = 0
	f.length = PolylineLength(pts)
}

// Advance 沿折线推进 dist（步距钳到剩余量），返回新位置。折线走完
// 后恒返回末点。
func (f *Follower) Advance(dist float32) Node {
	if len(f.pts) == 0 {
		return Node{}
	}
	if len(f.pts) == 1 {
		return f.pts[0]
	}
	f.offset += dist
	if f.offset >= f.length {
		f.offset = f.length
		return f.pts[len(f.pts)-1]
	}
	return f.positionAt(f.offset)
}

// Peek 返回当前进给位置（不推进）。
func (f *Follower) Peek() Node {
	if len(f.pts) == 0 {
		return Node{}
	}
	if len(f.pts) == 1 || f.length <= 0 {
		return f.pts[0]
	}
	return f.positionAt(f.offset)
}

// Remaining 返回距折线末点的剩余长度。
func (f *Follower) Remaining() float32 {
	if f.length <= f.offset {
		return 0
	}
	return f.length - f.offset
}

// Done 折线是否已走完。
func (f *Follower) Done() bool { return f.offset >= f.length }

func (f *Follower) positionAt(offset float32) Node {
	traveled := float32(0)
	for i := 1; i < len(f.pts); i++ {
		seg := distance3D(f.pts[i-1], f.pts[i])
		if traveled+seg >= offset && seg > 0 {
			ratio := (offset - traveled) / seg
			return lerpNode(f.pts[i-1], f.pts[i], ratio)
		}
		traveled += seg
		if traveled >= offset {
			return f.pts[i]
		}
	}
	return f.pts[len(f.pts)-1]
}

// lerpNode 线性插值两点（x/y/z 全维；y 插值即高度贴合坡面）。
func lerpNode(a, b Node, ratio float32) Node {
	return Node{
		X: a.X + (b.X-a.X)*ratio,
		Y: a.Y + (b.Y-a.Y)*ratio,
		Z: a.Z + (b.Z-a.Z)*ratio,
	}
}

// PolylineLength 返回折线总长（3D 欧氏逐段）。
func PolylineLength(pts []Node) float32 {
	var total float32
	for i := 1; i < len(pts); i++ {
		total += distance3D(pts[i-1], pts[i])
	}
	return total
}

// HorizontalPolylineLength 返回折线的水平投影总长（到达确认延时按
// 移动节奏取水平量更贴近客户端插值；两者仅坡地略有差异）。
func HorizontalPolylineLength(pts []Node) float32 {
	var total float32
	for i := 1; i < len(pts); i++ {
		total += horizontalDistance(pts[i-1], pts[i])
	}
	return total
}

// LerpOrientFreeStep 是直线推进的旧语义保留位（stepToward 对应物）：
// 沿水平方向推进 step，y 朝 targetY 线性插值。供无图兜底与末段直线
// 共用——y 插值消除直线追击的穿地/悬空。
func LerpOrientFreeStep(from, to Node, step float32) Node {
	distance := horizontalDistance(from, to)
	if distance <= 0 || step <= 0 {
		return from
	}
	if step > distance {
		step = distance
	}
	ratio := step / distance
	result := from
	result.X += (to.X - from.X) * ratio
	result.Z += (to.Z - from.Z) * ratio
	result.Y += (to.Y - from.Y) * ratio
	return result
}

// StepTowardGround 与 LerpOrientFreeStep 同为直线推进，但 y 不朝目标
// 插值（目标可能悬空/钻地），而是按本步占全程的比例向“推进落点的
// 最近路点高度”渐进贴地。无图（g == nil）时退化为纯水平推进、
// y 维持出发高度（旧 stepToward 语义）。
func StepTowardGround(g *Graph, from, to Node, step float32) Node {
	// 先只推 x/z（把 y 置平再插值，避免被目标 y 带偏）。
	flat := LerpOrientFreeStep(Node{X: from.X, Z: from.Z}, Node{X: to.X, Z: to.Z}, step)
	next := Node{X: flat.X, Y: from.Y, Z: flat.Z}
	if g == nil {
		return next
	}
	if node := g.Nearest(next); node >= 0 {
		ground := g.nodes[node].Y
		if total := horizontalDistance(from, to); total > 0 {
			ratio := minFloat32(step, total) / total
			next.Y += (ground - next.Y) * ratio
		}
	}
	return next
}

func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// OrientToward 返回面向 to 的朝向（atan2(dz,dx)，与 0x21/朝向语义一致）。
func OrientToward(from, to Node) float32 {
	return float32(math.Atan2(float64(to.Z-from.Z), float64(to.X-from.X)))
}
