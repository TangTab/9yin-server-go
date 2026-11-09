package mapnav

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBatchParseAllInstalledPathFiles 离线批测（§6.4）：NINEYIN_SERVER_ROOT
// 指向部署资源根时解析全部 .path 文件。硬闸两项：① 字节精确（解析
// 游标 == 文件长，格式解释错误的强信号）；② 距离量级健全（坐标若被
// 误读为别的排布，偏差会系统性放大到数十米以上且大面积覆盖）。官方
// 语料的真实噪声：508/557 文件在 0.1m 内全净，49 文件含少量官方手编
// 非欧边权（groupscene* 系 52 边/1.5m 恒定、test*/clone* 副本图最高
// ~25m/25% 边）——存储权重即权威代价，A* 照用，此处只记日志不判败。
func TestBatchParseAllInstalledPathFiles(t *testing.T) {
	root := os.Getenv("NINEYIN_SERVER_ROOT")
	if root == "" {
		root = filepath.Join("..", "..")
	}
	dir := filepath.Join(root, "resources", "modern", "map", "path")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("部署 .path 目录不可达（NINEYIN_SERVER_ROOT=%q）：%v", root, err)
	}
	files := 0
	nodes, edges := 0, 0
	deviantFiles := 0
	started := time.Now()
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".path" {
			continue
		}
		files++
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("%s: read: %v", entry.Name(), err)
		}
		g, err := Parse(data)
		if err != nil {
			t.Fatalf("%s: parse: %v", entry.Name(), err)
		}
		mismatches, maxDeviation := g.ValidateEdgeDistances(0.1)
		if mismatches > 0 {
			deviantFiles++
			t.Logf("%s: %d/%d edges deviate >0.1m (max %.3fm) — official authored weights, tolerated",
				entry.Name(), mismatches, g.EdgeCount(), maxDeviation)
			if maxDeviation > 50 || mismatches*2 > g.EdgeCount() {
				t.Errorf("%s: deviation pattern exceeds official-noise envelope (max %.1fm, %d/%d edges)",
					entry.Name(), maxDeviation, mismatches, g.EdgeCount())
			}
		}
		nodes += g.NodeCount()
		edges += g.EdgeCount()
	}
	if files == 0 {
		t.Skipf("%s 无 .path 文件", dir)
	}
	t.Logf("batch parse: files=%d nodes=%d edges=%d deviantFiles=%d in %s",
		files, nodes, edges, deviantFiles, time.Since(started).Round(time.Millisecond))
}

// TestScene14AStarAgainstDeployment §6.4 A* 压测：主城 scene14（16.7 万
// 节点/124 万边）随机 1000 对起终点——存在率不低于 Σ(分量占比)²×0.9
//（主分量 99.8% → ~0.996）、路径长 ≥ 欧氏距离。scene14.path 不在仓库
// testdata（12.6MB），仅在部署资源根可达时生效。
func TestScene14AStarAgainstDeployment(t *testing.T) {
	root := os.Getenv("NINEYIN_SERVER_ROOT")
	if root == "" {
		root = filepath.Join("..", "..")
	}
	graph, err := ParseFile(filepath.Join(root, "resources", "modern", "map", "path", "scene14.path"))
	if err != nil {
		t.Skipf("scene14.path 不可达（NINEYIN_SERVER_ROOT=%q）：%v", root, err)
	}
	// 连通分量占比（BFS 一遍）。
	component := make([]int32, graph.NodeCount())
	var sizes []int
	for seed := 0; seed < graph.NodeCount(); seed++ {
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
			for e := graph.edgeStart[at]; e < graph.edgeStart[at+1]; e++ {
				if next := graph.edgeTo[e]; component[next] == 0 {
					component[next] = label
					queue = append(queue, int(next))
				}
			}
		}
		sizes = append(sizes, size)
	}
	expected := 0.0
	for _, s := range sizes {
		expected += float64(s) / float64(graph.NodeCount()) * float64(s) / float64(graph.NodeCount())
	}

	random := newSplitMix64(0xC0FFEE)
	p := NewPlanner(graph)
	const pairs = 1000
	existing := 0
	started := time.Now()
	for i := 0; i < pairs; i++ {
		from, to := int(random.next()%uint64(graph.NodeCount())), int(random.next()%uint64(graph.NodeCount()))
		path := p.ShortestPath(from, to)
		if component[from] != component[to] {
			if path != nil {
				t.Fatalf("pair %d cross-component yielded a path", i)
			}
			continue
		}
		if path == nil {
			t.Fatalf("pair %d same-component has no path", i)
		}
		existing++
		if length := PathLength(graph, path); length < distance3D(graph.NodeAt(from), graph.NodeAt(to))-0.05 {
			t.Fatalf("pair %d path %.2f shorter than straight line", i, length)
		}
	}
	rate := float64(existing) / pairs
	t.Logf("scene14 A*: nodes=%d edges=%d components=%d expected=%.3f rate=%.3f avg=%.2fms/pair",
		graph.NodeCount(), graph.EdgeCount(), len(sizes), expected, rate,
		float64(time.Since(started).Microseconds())/pairs/1000)
	if rate < expected*0.9 {
		t.Fatalf("success rate %.3f below expectation %.3f", rate, expected)
	}
}
