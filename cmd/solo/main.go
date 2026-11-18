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
	"github.com/lizhanyin/9yin-server-go/internal/login"
)

const version = "0.3.0-solo-svrlist"

// milestones 是第一里程碑的串联清单，随实现逐步勾掉：
//   - [x] ecs 内核（World/Entity/Storage/Query/Scheduler，测试+基准通过）
//   - [x] structgen 产出 component + 元数据（208 对象/1666 属性/306 表格）
//   - [x] config 加载 scenes.xml / logic_class.xml（含通用 Object/Property 加载器）
//   - [x] engine tick 循环 + proc/inproc Bus
//   - [x] net/proto 最小集：EE+XOR 线路 + 引擎级消息/自定义消息目录 + 心跳网关
//     （cmd/probe 对拍：golden ping→pong 真 TCP 往返）
//   - [x] svrlist 区服列表分发（login 链第一环；cmd/lister 独立载体）
//   - [ ] login 主干：进场景 + AOI + 属性全量集下发
//   - [ ] persist：DDL 建库 + 角色存取
func main() {
	var (
		scenesPath = flag.String("scenes", "testdata/ini/scenes.xml", "scenes.xml 路径")
		logicPath  = flag.String("logicclass", "testdata/ini/logic_class.xml", "logic_class.xml 路径")
		listen     = flag.String("listen", "127.0.0.1:9600", "客户端网关监听地址（空=不监听）")
		svrlist    = flag.String("svrlist", "127.0.0.1:9601", "区服列表分发监听地址（空=不监听；通告地址/端口跟随 -listen）")
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

	// ---- 网关：客户端线路（EE+XOR），当前只应答心跳 ----
	var gateway *login.Gateway
	if *listen != "" {
		gateway = &login.Gateway{
			Addr: *listen,
			Logf: func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
		}
		app.OnStart(func(*engine.App) {
			addr, err := gateway.Listen()
			if err != nil {
				fatal(err)
			}
			fmt.Printf("网关就绪：%s（EE+XOR，心跳应答；probe 对拍：./build/probe.exe -addr %s）\n",
				addr, addr)
			go gateway.Serve()
		})
		app.OnStop(func(*engine.App) {
			if err := gateway.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "solo: 网关关闭:", err)
			}
		})
	}

	// ---- svrlist：区服列表分发（明文 ASCII 行协议，通告跟随 -listen）----
	if *svrlist != "" {
		svl := &login.Lister{
			Addr:  *svrlist,
			Entry: login.DefaultListerEntry(*listen),
			Logf:  func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
		}
		app.OnStart(func(*engine.App) {
			addr, err := svl.Listen()
			if err != nil {
				fatal(err)
			}
			fmt.Printf("svrlist 就绪：%s → 通告 %s:%d（probe 对拍：./build/probe.exe -svrlist %s）\n",
				addr, svl.Entry.Host, svl.Entry.Port, addr)
			go svl.Serve()
		})
		app.OnStop(func(*engine.App) {
			if err := svl.Close(); err != nil {
				fmt.Fprintln(os.Stderr, "solo: svrlist 关闭:", err)
			}
		})
	}

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
