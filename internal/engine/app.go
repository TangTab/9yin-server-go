package engine

import (
	"context"
	"math"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/ecs"
	"github.com/lizhanyin/9yin-server-go/internal/proc"
)

// Options 主循环参数。
type Options struct {
	// TPS 每秒 tick 数；0 表示全速循环（对齐原版 Sleep=-1 语义）。
	TPS int
}

// App 是进程引擎壳：World + Scheduler + Bus 的装配与主循环，
// 对应原版 FsLoader/FsKit 的启动装配职责。solo 形态由 cmd/solo 组装；
// 分布式形态各 cmd 组装各自角色的 System 集后共用同一循环。
type App struct {
	World *ecs.World
	Sched *ecs.Scheduler
	Bus   proc.Bus

	opts    Options
	onStart []func(*App)
	onStop  []func(*App)
}

// New 创建引擎：内建 ecs.World、ecs.Scheduler 与 inproc 总线。
// 总线在 Run 返回时关闭。
func New(opts Options) *App {
	return &App{
		World: ecs.New(),
		Sched: ecs.NewScheduler(),
		Bus:   proc.NewInprocBus(),
		opts:  opts,
	}
}

// OnStart 注册启动钩子（主循环开始前按序执行）：加载配置、建场景等装配工作。
func (a *App) OnStart(f func(*App)) { a.onStart = append(a.onStart, f) }

// OnStop 注册停止钩子（主循环退出后按序执行）：落库、清场。
func (a *App) OnStop(f func(*App)) { a.onStop = append(a.onStop, f) }

// Run 驱动主循环直到 ctx 取消，随后按序执行 OnStop 并关闭总线。
// dt 语义：TPS>0 时为 1/TPS 固定步长；全速时为实测间隔（首次 tick 取 0）。
func (a *App) Run(ctx context.Context) {
	defer a.Bus.Close()

	for _, f := range a.onStart {
		f(a)
	}
	defer func() {
		for _, f := range a.onStop {
			f(a)
		}
	}()

	dt := 0.0
	if a.opts.TPS > 0 {
		dt = 1.0 / float64(a.opts.TPS)
	}

	var ticker *time.Ticker
	var ticks <-chan time.Time
	if a.opts.TPS > 0 {
		ticker = time.NewTicker(time.Second / time.Duration(a.opts.TPS))
		defer ticker.Stop()
		ticks = ticker.C
	}

	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if a.opts.TPS > 0 {
			select {
			case <-ctx.Done():
				return
			case <-ticks:
			}
		}
		now := time.Now()
		if dt == 0 {
			dt = math.Min(now.Sub(last).Seconds(), 0.25) // 全速：实际间隔，钳制防大跳
		}
		last = now
		a.Sched.Tick(a.World, dt)
	}
}

// TickOnce 手动驱动一帧（测试与工具用）。
func (a *App) TickOnce(dt float64) { a.Sched.Tick(a.World, dt) }
