// Package mapnav 提供九阴服务端路点图(.path)的解析、空间索引、A*
// 寻路与折线跟随。数据格式与几何性质见 docs/map-path-waypoint-graph.md：
// 小端、无文件头魔法，int32 节点数后逐节点 float32 x/y/z + int32 邻边数，
// 每边 int32 目标下标 + float32 欧氏距离（邻接表双向互指）。
//
// 该图即可行走面的 ~10m 均匀采样（8 邻接网格，度上限 8），y 为高度；
// 服务端没有碰撞几何，寻路以图距离替代直线穿行。
package mapnav

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
)

// Node 是一个路点的世界坐标（y = 高度）。
type Node struct {
	X, Y, Z float32
}

// Graph 是解析后的路点图。全部字段只读，可被多个场景运行时共享；
// 每运行时自持 Planner/A* 工作区。
type Graph struct {
	nodes []Node
	// 邻接表 CSR 压缩：edgeStart[i]..edgeStart[i+1] 是节点 i 的边在
	// edgeTo/edgeDist 中的区间。距离即文件存储的欧氏边权。
	edgeStart []int32
	edgeTo    []int32
	edgeDist  []float32
	// index 是 XZ 均匀网格空间索引（index.go），Parse 时构建。
	index *gridIndex
}

// NodeCount 返回路点总数。
func (g *Graph) NodeCount() int { return len(g.nodes) }

// NodeAt 返回下标 i 的路点坐标。
func (g *Graph) NodeAt(i int) Node { return g.nodes[i] }

// EdgeCount 返回邻接表总边数（含双向重复计数）。
func (g *Graph) EdgeCount() int { return len(g.edgeTo) }

// Parse 解析 .path 字节流并构建空间索引。字节级校验：解析游标必须
// 精确落到文件尾（多字节/截断/越界下标都报错），坐标-距离的数学级
// 校验用 ValidateEdgeDistances（离线批量验证用，热路径不跑）。
func Parse(data []byte) (*Graph, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("mapnav: truncated header: %d bytes", len(data))
	}
	count := int(int32(binary.LittleEndian.Uint32(data[0:4])))
	if count < 0 || count > len(data)/16 {
		// 每节点至少 16B（12B 坐标 + 4B 边数）；超界即按坏文件拒绝，
		// 防御恶意/错配文件的 OOM。
		return nil, fmt.Errorf("mapnav: node count %d out of range for %d bytes", count, len(data))
	}
	g := &Graph{
		nodes:     make([]Node, count),
		edgeStart: make([]int32, count+1),
	}
	// 两遍扫描：先数边定 CSR 布局，再回填目标与距离。第二遍游标与
	// 第一遍逐字节同步推进，最终 off == len(data) 即格式确认。
	edges := 0
	off := 4
	for i := 0; i < count; i++ {
		off += 12 // x,y,z
		e, err := readEdgeCount(data, &off)
		if err != nil {
			return nil, err
		}
		edges += e
		if edges > len(data)/8 {
			return nil, fmt.Errorf("mapnav: edge count overflow at node %d", i)
		}
		off += e * 8
		if off > len(data) {
			return nil, fmt.Errorf("mapnav: truncated at node %d: off=%d len=%d", i, off, len(data))
		}
	}
	if off != len(data) {
		return nil, fmt.Errorf("mapnav: %d trailing bytes after last node", len(data)-off)
	}
	g.edgeTo = make([]int32, edges)
	g.edgeDist = make([]float32, edges)
	off = 4
	edge := 0
	for i := 0; i < count; i++ {
		g.nodes[i] = Node{
			X: math.Float32frombits(binary.LittleEndian.Uint32(data[off:])),
			Y: math.Float32frombits(binary.LittleEndian.Uint32(data[off+4:])),
			Z: math.Float32frombits(binary.LittleEndian.Uint32(data[off+8:])),
		}
		off += 12
		g.edgeStart[i] = int32(edge)
		e := int(int32(binary.LittleEndian.Uint32(data[off:])))
		off += 4
		for j := 0; j < e; j++ {
			target := int(int32(binary.LittleEndian.Uint32(data[off:])))
			if target < 0 || target >= count {
				return nil, fmt.Errorf("mapnav: node %d edge %d target %d out of range", i, j, target)
			}
			g.edgeTo[edge] = int32(target)
			g.edgeDist[edge] = math.Float32frombits(binary.LittleEndian.Uint32(data[off+4:]))
			edge++
			off += 8
		}
	}
	g.edgeStart[count] = int32(edges)
	g.index = buildGridIndex(g.nodes)
	return g, nil
}

func readEdgeCount(data []byte, off *int) (int, error) {
	if *off+4 > len(data) {
		return 0, fmt.Errorf("mapnav: truncated edge count at off=%d", *off)
	}
	e := int(int32(binary.LittleEndian.Uint32(data[*off:])))
	*off += 4
	if e < 0 || e > (len(data)-*off)/8 {
		return 0, fmt.Errorf("mapnav: edge count %d out of range at off=%d", e, *off)
	}
	return e, nil
}

// ParseFile 读取并解析一个 .path 文件。
func ParseFile(path string) (*Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	graph, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("mapnav: %s: %w", path, err)
	}
	return graph, nil
}

// ValidateEdgeDistances 把每条边存储的距离与两端节点坐标的欧氏距离
// 比对（数学级格式验证）。返回 (超差边数, 最大偏差米)。557 文件实测
// 全部零不匹配（docs/map-path-waypoint-graph.md §2.1）。
func (g *Graph) ValidateEdgeDistances(tolerance float32) (mismatches int, maxDeviation float32) {
	for i, node := range g.nodes {
		for e := g.edgeStart[i]; e < g.edgeStart[i+1]; e++ {
			target := g.nodes[g.edgeTo[e]]
			dx := float64(node.X - target.X)
			dy := float64(node.Y - target.Y)
			dz := float64(node.Z - target.Z)
			deviation := float32(math.Abs(math.Sqrt(dx*dx+dy*dy+dz*dz) - float64(g.edgeDist[e])))
			if deviation > maxDeviation {
				maxDeviation = deviation
			}
			if deviation > tolerance {
				mismatches++
			}
		}
	}
	return mismatches, maxDeviation
}
