// solo 是单进程整合服入口，对应原版 FsSoloLoader + FsSolo* 模块集：
// 把 entry/lister/member/world/public 全部角色装进一个进程跑。
//
// 这是第一里程碑的载体：登录→选角→进场景→AOI→属性同步→落库，
// 跑通即拿到整套架构的 80%；之后再按角色拆出独立 cmd。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/config"
	"github.com/lizhanyin/9yin-server-go/internal/ecs"
	"github.com/lizhanyin/9yin-server-go/internal/engine"
)

const version = "0.1.0-solo-skeleton"

// milestones 是第一里程碑的串联清单，随实现逐步勾掉：
//   - [x] ecs 内核（World/Entity/Storage/Query/Scheduler，测试+基准通过）
//   - [x] structgen 产出 component + 元数据（208 对象/1666 属性/306 表格）
//   - [x] config 加载 scenes.xml / logic_class.xml（含通用 Object/Property 加载器）
//   - [x] engine tick 循环 + proc/inproc Bus
//   - [ ] net/proto 最小集：握手 + 心跳（probe 对拍）
//   - [ ] login 主干：进场景 + AOI + 属性全量集下发
//   - [ ] persist：DDL 建库 + 角色存取
func main() {
	var (
		scenesPath = flag.String("scenes", "testdata/ini/scenes.xml", "scenes.xml 路径")
		logicPath  = flag.String("logicclass", "testdata/ini/logic_class.xml", "logic_class.xml 路径")
		tps        = flag.Int("tps", 60, "每秒 tick 数（0=全速）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C）")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go solo %s\n", version)

	// ---- 装配：配置 + 引擎 ----
	app := engine.New(engine.Options{TPS: *tps})

	var scenes *config.SceneTable
	var classes *config.LogicClassTree
	app.OnStart(func(a *engine.App) {
		var err error
		if scenes, err = config.LoadScenes(*scenesPath); err != nil {
			fatal(err)
		}
		if classes, err = config.LoadLogicClass(*logicPath); err != nil {
			fatal(err)
		}
		clonable := 0
		for _, s := range scenes.All() {
			if classes.IsA(s.Script, "scene") && s.Clonable {
				clonable++
			}
		}
		fmt.Printf("配置就绪：%d 场景（副本 %d）、%d 逻辑类（根 %d）\n",
			scenes.Len(), clonable, classes.Len(), len(classes.Roots()))

		// 演示 System：每秒汇报一次帧号与实体数
		lastReport := time.Now()
		a.Sched.AddFunc("monitor", func(w *ecs.World, dt float64) {
			if now := time.Now(); now.Sub(lastReport) >= time.Second {
				lastReport = now
				fmt.Printf("tick=%d 实体=%d\n", w.CurrentTick(), w.EntityCount())
			}
		})
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if *runFor > 0 {
		ctx, cancel = context.WithTimeout(ctx, *runFor)
		defer cancel()
	}

	start := time.Now()
	app.OnStop(func(*engine.App) {
		fmt.Printf("停止：运行 %s，共 %d tick\n", time.Since(start).Round(time.Millisecond), app.World.CurrentTick())
	})

	app.Run(ctx)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "solo:", err)
	os.Exit(1)
}
