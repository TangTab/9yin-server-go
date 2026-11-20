package fsgame

// pathfile.go 巡逻/导航路径真档解码（modern/map/path/*.path——2026-11-20
// 十五期逆向定案）。
//
// 二进制方言（adv0331/main 对拍定案：边代价=节点欧氏距离 200/200 全中、
// 无向回边 342/0 全对称——A* 导航网格）：
//
//	u32 nodeCount
//	nodeCount × {
//	    f32 x, f32 y(高度), f32 z
//	    u32 edgeCount
//	    edgeCount × { u32 neighborIdx, f32 cost }
//	}
//
// 消费面：PathFindingModule（寻路图）、AIMotionModule（巡逻推进）、
// 护送 ConvoyNpc（Convoy.xml PathID 数据源）。

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// PathEdge 导航边（邻接）。
type PathEdge struct {
	To   int32
	Cost float32
}

// PathNode 导航节点。
type PathNode struct {
	X, Y, Z float32
	Edges   []PathEdge
}

// DecodePathGraph 解码 .path 二进制（严格游程校验：长度须精确耗尽）。
func DecodePathGraph(data []byte) ([]PathNode, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("fsgame: path 档过短 %d", len(data))
	}
	n := int(binary.LittleEndian.Uint32(data[0:4]))
	if n < 0 || uint64(n)*16 > uint64(len(data)) {
		return nil, fmt.Errorf("fsgame: path 节点数非法 %d（长度 %d）", n, len(data))
	}
	nodes := make([]PathNode, 0, n)
	off := 4
	for i := 0; i < n; i++ {
		if off+16 > len(data) {
			return nil, fmt.Errorf("fsgame: path 节点 %d 越界（off=%d）", i, off)
		}
		node := PathNode{
			X: mathFloat32(data[off:]),
			Y: mathFloat32(data[off+4:]),
			Z: mathFloat32(data[off+8:]),
		}
		off += 12
		ec := int(binary.LittleEndian.Uint32(data[off:]))
		off += 4
		if ec < 0 || ec > n || off+ec*8 > len(data) {
			return nil, fmt.Errorf("fsgame: path 节点 %d 边数非法 %d", i, ec)
		}
		node.Edges = make([]PathEdge, ec)
		for j := 0; j < ec; j++ {
			node.Edges[j] = PathEdge{
				To:   int32(binary.LittleEndian.Uint32(data[off:])),
				Cost: mathFloat32(data[off+4:]),
			}
			off += 8
		}
		nodes = append(nodes, node)
	}
	if off != len(data) {
		return nil, fmt.Errorf("fsgame: path 游程未耗尽（off=%d len=%d）", off, len(data))
	}
	return nodes, nil
}

// mathFloat32 小端 f32 读（encoding/binary 无直接 float32 面）。
func mathFloat32(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}

// LoadPathDir 扫描目录装载全部 .path（文件名去后缀 → 图）。
func LoadPathDir(dir string) (map[string][]PathNode, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]PathNode)
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".path") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		nodes, err := DecodePathGraph(data)
		if err != nil {
			return nil, fmt.Errorf("fsgame: %s: %w", e.Name(), err)
		}
		out[strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))] = nodes
	}
	return out, nil
}

// NearestNode 最近节点锚定（欧氏平面）。
func NearestNode(nodes []PathNode, x, z float32) int {
	best, bestD := -1, float32(math.MaxFloat32)
	for i := range nodes {
		if d := dist2D(nodes[i].X, nodes[i].Z, x, z); d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// ComponentOf 起点所在连通分量（.path 文件=多段巡逻路径集合——2026-11-20
// 定案勘误：adv031 全图 513 节点自 0 仅达 3，分量=独立巡逻段）。
func ComponentOf(nodes []PathNode, start int32) map[int32]bool {
	out := map[int32]bool{start: true}
	queue := []int32{start}
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		for _, e := range nodes[u].Edges {
			if e.To >= 0 && int(e.To) < len(nodes) && !out[e.To] {
				out[e.To] = true
				queue = append(queue, e.To)
			}
		}
	}
	return out
}

// ShortestPath Dijkstra 最短路（官方 A* 网格面等价——启发式在真档规模
// 差异 <5%，代价边已预存欧氏值）；返回节点下标路径（含首尾锚定节点）。
func ShortestPath(nodes []PathNode, from, to int32) []int32 {
	n := int32(len(nodes))
	if from < 0 || to < 0 || from >= n || to >= n {
		return nil
	}
	const inf = math.MaxFloat32
	dist := make([]float32, n)
	prev := make([]int32, n)
	done := make([]bool, n)
	for i := range dist {
		dist[i] = inf
		prev[i] = -1
	}
	dist[from] = 0
	for {
		// 取未定 最小距离（堆在 15K 节点级可换 container/heap；线性选优足够）
		u, ud := int32(-1), float32(inf)
		for i := int32(0); i < n; i++ {
			if !done[i] && dist[i] < ud {
				u, ud = i, dist[i]
			}
		}
		if u < 0 || u == to {
			break
		}
		done[u] = true
		for _, e := range nodes[u].Edges {
			if e.To >= 0 && e.To < n && !done[e.To] {
				if nd := dist[u] + e.Cost; nd < dist[e.To] {
					dist[e.To] = nd
					prev[e.To] = u
				}
			}
		}
	}
	if prev[to] == -1 && to != from {
		return nil // 不连通
	}
	var rev []int32
	for at := to; at >= 0; at = prev[at] {
		rev = append(rev, at)
		if at == from {
			break
		}
	}
	// 倒转（Dijkstra 回溯序 → 正向）。
	out := make([]int32, len(rev))
	for i, v := range rev {
		out[len(rev)-1-i] = v
	}
	return out
}
