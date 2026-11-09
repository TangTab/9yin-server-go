package main

import (
	"log"
	"math"
	"os"
	"time"

	"github.com/local/9yin-go-server/internal/mapnav"
	"github.com/local/9yin-go-server/internal/sim"
	worldcore "github.com/local/9yin-go-server/internal/world"
)

// 路点图追击参数（docs/map-path-waypoint-graph.md §6.3）。
const (
	// chaseReplanEvery 重规划节流：玩家持续移动时不必每拍重算 A*，
	// 300~500ms 一次足够贴身。
	chaseReplanEvery = 400 * time.Millisecond
	// chaseReplanTargetShift 目标偏离上次规划终点超此值立即重算。
	chaseReplanTargetShift = float32(8)
	// chaseFinalStraightMeters 距目标水平小于此值（约一个路点间距）
	// 切直线逼近，视觉无缝。
	chaseFinalStraightMeters = float32(12)
	// chaseRetryEvery A* 失败（跨分量）后的重试间隔——玩家可能走进
	// 另一分量的可达范围。
	chaseRetryEvery = 2 * time.Second
	// chaseMaxPlanFails 连续规划失败上限，超过即放弃仇恨回 Home。
	chaseMaxPlanFails = 5
)

// mapnavLibrary 是进程级路点图装载器（resources/modern/map/path/
// <resource>.path）。Graph 解析后不可变，跨场景运行时共享。
var mapnavLibrary = mapnav.NewLibrary(runtimeProjectPath("resources", "modern", "map", "path"))

// mapnavLoaderForTest 允许测试把场景→路点图的来源替换成合成图（生产
// 恒 nil 走 mapnavLibrary）。同包测试串行执行，无并发读写。
var mapnavLoaderForTest func(resource string) (*mapnav.Graph, bool)

// npcChaseState 是单个 NPC 的追击导航状态（sceneEntity.chase，run.mu
// 保护）。折线由 A* 规划，Follower 按拍推进并做 y 插值。
type npcChaseState struct {
	follower mapnav.Follower
	planAt   time.Time
	planEnd  mapnav.Node
	fails    int
	retryAt  time.Time
}

// warmNav 装载场景地形的路点图（sync.Once，sceneRun 创建时后台预热、
// 首次追击未就绪时兜底触发）。结果（含失败）写回 r.nav*/r.navLoaded，
// 全部在 r.mu 内落盘——追击 tick 在 r.mu 内读，无竞态。
func (r *sceneRun) warmNav() {
	r.navOnce.Do(func() {
		var graph *mapnav.Graph
		if r.key.resource != "" {
			if mapnavLoaderForTest != nil {
				if injected, ok := mapnavLoaderForTest(r.key.resource); ok {
					graph = injected
				}
			} else if loaded, ok := mapnavLibrary.Graph(r.key.resource); ok {
				graph = loaded
			}
		}
		if graph == nil {
			// 部署资源根存在而该图缺失才值得留痕；仓库内开发环境
			//（resources/ 整体不入库）保持静默。
			if r.key.resource != "" && mapnavLoaderForTest == nil {
				if _, err := os.Stat(mapnavLibrary.Dir()); err == nil {
					log.Printf("scene %s/%s: no waypoint graph for resource %q (chase falls back to straight line)",
						r.key.config, r.key.resource, r.key.resource)
				}
			}
		}
		r.mu.Lock()
		r.nav = graph
		if graph != nil {
			r.navPlanner = mapnav.NewPlanner(graph)
		}
		r.navLoaded = true
		r.mu.Unlock()
		if graph != nil {
			log.Printf("scene %s/%s: waypoint graph ready nodes=%d edges=%d",
				r.key.config, r.key.resource, graph.NodeCount(), graph.EdgeCount())
		}
	})
}

// navGraphLocked 返回路点图；装载未完成时返回 nil（追击本拍走直线
// 兜底，下一拍重查——绝不阻塞 tick）。调用方必须已持有 r.mu。
func (r *sceneRun) navGraphLocked() *mapnav.Graph {
	if !r.navLoaded {
		go r.warmNav()
		return nil
	}
	return r.nav
}

// planChasePath 规划 NPC→目标的世界折线：当前坐标 → A* 路点 → 目标
// 坐标。跨分量/无解返回 nil。path[0]（NPC 最近路点）被丢弃——从当前
// 位置直接走向 path[1]：重规划每拍发生时“先折返最近路点”会让 NPC 在
// 两路点间原地振荡（追击穿墙回归实测）；近似弦切不超过一格路点间距
//（~10m），仍沿可走网格拓扑。调用方必须已持有 r.mu。
func (r *sceneRun) planChasePath(graph *mapnav.Graph, from, target mapnav.Node) []mapnav.Node {
	start, goal := graph.Nearest(from), graph.Nearest(target)
	if start < 0 || goal < 0 {
		return nil
	}
	indices := r.navPlanner.ShortestPath(start, goal)
	if indices == nil {
		return nil
	}
	pts := make([]mapnav.Node, 0, len(indices)+2)
	pts = append(pts, from)
	if len(indices) > 1 {
		indices = indices[1:]
	}
	for _, index := range indices {
		pts = append(pts, graph.NodeAt(index))
	}
	pts = append(pts, target)
	return pts
}

// advanceChaseLocked 是追击分支的单拍位移决策（替代旧直线 stepToward）：
// 有图走「A* 折线 + y 插值 + 末段直线」，无图/规划失败走「直线贴地」
// 兜底，跨分量连续失败放弃仇恨。返回推进后的权威坐标与到达确认延时
// （剩余路径长/速度）；gaveUp 时调用方按栓绳语义回位并清仇恨。
//
// 调用方必须已持有 r.mu。
func (r *sceneRun) advanceChaseLocked(meta *sceneEntity, state *sim.Brain, transform *worldcore.Transform, target worldcore.Transform, now time.Time) (destination worldcore.Transform, delay time.Duration, gaveUp bool) {
	from := mapnav.Node{X: transform.X, Y: transform.Y, Z: transform.Z}
	goal := mapnav.Node{X: target.X, Y: target.Y, Z: target.Z}
	distance := horizontalDistance(*transform, target)
	graph := r.navGraphLocked()

	// 无图场景兜底：直线推进但 y 向落点最近路点贴地（穿地/悬空缓解）；
	// 完全无图维持旧语义。
	if graph == nil {
		return chaseStraightLocked(nil, from, goal, distance, state), chaseStraightDelay(distance, state.MoveSpeed), false
	}

	chase := meta.chase
	if chase == nil {
		chase = &npcChaseState{}
		meta.chase = chase
	}

	// 末段直线：距目标 ~一个路点间距内直线逼近，y 直接朝目标插值
	//（近距目标 y 可信），步长钳到攻击范围边界。
	if distance < chaseFinalStraightMeters {
		chase.follower.SetPath(nil)
		step := minFloat32(state.MoveSpeed, distance-state.AttackRange)
		at := mapnav.LerpOrientFreeStep(from, goal, step)
		return orientTowardTransform(from, at, transform.Orient), chaseStraightDelay(step, state.MoveSpeed), false
	}

	needPlan := chase.planAt.IsZero() ||
		now.Sub(chase.planAt) >= chaseReplanEvery ||
		mapnavHorizontal(chase.planEnd, goal) > chaseReplanTargetShift
	if needPlan && !now.Before(chase.retryAt) {
		if pts := r.planChasePath(graph, from, goal); pts != nil {
			chase.follower.SetPath(pts)
			chase.planAt = now
			chase.planEnd = goal
			chase.fails = 0
		} else {
			// 跨分量：A* 失败，直线兜底并周期重试（玩家可能走进另
			// 一分量的可达范围）；连续失败放弃仇恨。
			chase.follower.SetPath(nil)
			chase.fails++
			chase.retryAt = now.Add(chaseRetryEvery)
			if chase.fails >= chaseMaxPlanFails {
				return worldcore.Transform{}, 0, true
			}
		}
	}

	if !chase.follower.Done() {
		at := chase.follower.Advance(state.MoveSpeed)
		delay := time.Duration(chase.follower.Remaining()/state.MoveSpeed*float32(time.Second)) / time.Millisecond * time.Millisecond
		return orientTowardTransform(from, at, transform.Orient), delay, false
	}

	// 折线走完仍在远处（等待重规划/重试窗口）：直线贴地推进。
	return chaseStraightLocked(graph, from, goal, distance, state), chaseStraightDelay(distance, state.MoveSpeed), false
}

// chaseStraightLocked 直线兜底推进：步长钳到攻击范围边界，y 向落点
// 最近路点渐进贴地（graph == nil 时 y 维持出发高度，即旧 stepToward）。
func chaseStraightLocked(graph *mapnav.Graph, from, goal mapnav.Node, distance float32, state *sim.Brain) worldcore.Transform {
	step := minFloat32(state.MoveSpeed, distance-state.AttackRange)
	at := mapnav.StepTowardGround(graph, from, goal, step)
	return worldcore.Transform{X: at.X, Y: at.Y, Z: at.Z, Orient: atan2Orient(from, at)}
}

// chaseStraightDelay 直线推进的到达确认延时：按剩余距离/速度。
func chaseStraightDelay(remaining, speed float32) time.Duration {
	if speed <= 0 || remaining <= 0 {
		return 0
	}
	return time.Duration(remaining/speed*float32(time.Second)) / time.Millisecond * time.Millisecond
}

// orientTowardTransform 折线跟随的朝向：沿本拍实际推进方向；零位移
//（已到折线端点等）保持旧朝向。
func orientTowardTransform(from, at mapnav.Node, fallback float32) worldcore.Transform {
	orient := fallback
	if dx, dz := at.X-from.X, at.Z-from.Z; dx != 0 || dz != 0 {
		orient = atan2Orient(from, at)
	}
	return worldcore.Transform{X: at.X, Y: at.Y, Z: at.Z, Orient: orient}
}

func atan2Orient(from, to mapnav.Node) float32 {
	return float32(math.Atan2(float64(to.Z-from.Z), float64(to.X-from.X)))
}

func mapnavHorizontal(a, b mapnav.Node) float32 {
	dx, dz := a.X-b.X, a.Z-b.Z
	return float32(math.Hypot(float64(dx), float64(dz)))
}
