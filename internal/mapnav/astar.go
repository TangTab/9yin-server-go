package mapnav

// Planner 是绑定单张图的 A* 工作区。节点数最大 ~16.7 万（scene14），
// gScore/前向指针/访问标记用代章（generation stamp）懒失效，跨查询
// 复用免清零。Planner 非并发安全：每个场景运行时自持一份，在各自的
// tick 锁内使用。
type Planner struct {
	graph *Graph
	// 每节点懒失效标签：stamp[i] == gen 表示本查询已初始化。
	stamp  []uint32
	gen    uint32
	gScore []float32
	came   []int32
	closed []bool
	heap   nodeHeap
}

// NewPlanner 创建绑定 graph 的 A* 工作区。
func NewPlanner(graph *Graph) *Planner {
	n := graph.NodeCount()
	return &Planner{
		graph:  graph,
		stamp:  make([]uint32, n),
		gScore: make([]float32, n),
		came:   make([]int32, n),
		closed: make([]bool, n),
		heap:   newNodeHeap(n / 4),
	}
}

// ShortestPath 返回 from → to 的节点下标序列（含两端）。不可达（跨
// 连通分量）返回 nil。from == to 返回单元素。启发式为 3D 欧氏距离，
// 边权即存储距离，两者同量纲，启发式可采纳（不高估）。
func (p *Planner) ShortestPath(from, to int) []int {
	n := p.graph.NodeCount()
	if from < 0 || from >= n || to < 0 || to >= n {
		return nil
	}
	if from == to {
		return []int{from}
	}
	p.gen++
	if p.gen == 0 { // 回绕：代章耗尽（约 40 亿次查询）整体重置。
		for i := range p.stamp {
			p.stamp[i] = 0
		}
		p.gen = 1
	}
	goal := p.graph.nodes[to]
	p.stamp[from] = p.gen
	p.gScore[from] = 0
	p.came[from] = -1
	p.closed[from] = false
	p.heap.reset()
	p.heap.push(from, distance3D(p.graph.nodes[from], goal))
	for p.heap.len() > 0 {
		current, _ := p.heap.pop()
		if current == to {
			return p.reconstruct(from, to)
		}
		// 堆中同一节点可能残留过期条目（松弛后重新入堆），弹出时
		// 以代章+closed 双闸跳过。
		if p.stamp[current] != p.gen || p.closed[current] {
			continue
		}
		p.closed[current] = true
		for e := p.graph.edgeStart[current]; e < p.graph.edgeStart[current+1]; e++ {
			next := int(p.graph.edgeTo[e])
			if p.stamp[next] == p.gen && p.closed[next] {
				continue
			}
			tentative := p.gScore[current] + p.graph.edgeDist[e]
			if p.stamp[next] != p.gen || tentative < p.gScore[next] {
				p.stamp[next] = p.gen
				p.gScore[next] = tentative
				p.came[next] = int32(current)
				p.closed[next] = false
				p.heap.push(next, tentative+distance3D(p.graph.nodes[next], goal))
			}
		}
	}
	return nil
}

func (p *Planner) reconstruct(from, to int) []int {
	path := make([]int, 0, 64)
	for at := int32(to); at >= 0; at = p.came[at] {
		path = append(path, int(at))
		if int(at) == from {
			break
		}
	}
	// 反转（reconstruct 沿 came 链倒走）。
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	if path[0] != from {
		return nil
	}
	return path
}

// nodeHeap 是 (节点, f 值) 最小二叉堆，手写避免 container/heap 的
// 接口分发开销。同节点重复入堆合法，弹出端靠代章去重。
type nodeHeap struct {
	nodes []int32
	f     []float32
}

func newNodeHeap(capacity int) nodeHeap {
	if capacity < 16 {
		capacity = 16
	}
	return nodeHeap{nodes: make([]int32, 0, capacity), f: make([]float32, 0, capacity)}
}

func (h *nodeHeap) len() int { return len(h.nodes) }

func (h *nodeHeap) reset() {
	h.nodes = h.nodes[:0]
	h.f = h.f[:0]
}

func (h *nodeHeap) push(node int, f float32) {
	h.nodes = append(h.nodes, int32(node))
	h.f = append(h.f, f)
	i := len(h.nodes) - 1
	for i > 0 {
		parent := (i - 1) / 2
		if h.f[parent] <= h.f[i] {
			break
		}
		h.f[parent], h.f[i] = h.f[i], h.f[parent]
		h.nodes[parent], h.nodes[i] = h.nodes[i], h.nodes[parent]
		i = parent
	}
}

func (h *nodeHeap) pop() (int, float32) {
	last := len(h.nodes) - 1
	node, f := int(h.nodes[0]), h.f[0]
	h.nodes[0], h.f[0] = h.nodes[last], h.f[last]
	h.nodes = h.nodes[:last]
	h.f = h.f[:last]
	i := 0
	for {
		left, right := 2*i+1, 2*i+2
		smallest := i
		if left < len(h.nodes) && h.f[left] < h.f[smallest] {
			smallest = left
		}
		if right < len(h.nodes) && h.f[right] < h.f[smallest] {
			smallest = right
		}
		if smallest == i {
			break
		}
		h.f[smallest], h.f[i] = h.f[i], h.f[smallest]
		h.nodes[smallest], h.nodes[i] = h.nodes[i], h.nodes[smallest]
		i = smallest
	}
	return node, f
}

// PathLength 返回节点序列的折线长度（3D 欧氏逐段求和）。少于两点
// 返回 0。
func PathLength(g *Graph, path []int) float32 {
	var total float32
	for i := 1; i < len(path); i++ {
		total += distance3D(g.nodes[path[i-1]], g.nodes[path[i]])
	}
	return total
}
