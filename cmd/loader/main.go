// loader 是进程装载器入口，对应原版 FsLoader.exe + FxCore.dll 内核：
//
//	game_loader.exe [ini名] [附加参数...]
//
// argv 约定与官方逐字一致（GameLister.bat = `start FsLoader.exe
// GameLister.ini`）：args[0] 覆盖 ini 名（缺省从可执行文件名派生，
// 回退字面量 FsLoader.ini），args[1..] 为附加参数。
//
// 装载五步序列/泵循环 Sleep 节流/Ctrl+C 优雅停机/崩溃 ERROR*.log =
// internal/fsloader（FsLoader.exe 复刻）；内核（CCore 运行时：
// Startup 五段/定步泵/实体工厂/事件/脚本热重载/nx_* API 面）=
// internal/fxcore（FxCore.dll 复刻）。默认注册 fskit 模块（官方
// FsLoader.ini [Module] 单模块形态）。
package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
	"github.com/lizhanyin/9yin-server-go/internal/fsloader"
	"github.com/lizhanyin/9yin-server-go/internal/fxcore"
)

// defaultLoaderINI 内置默认配置（官方 FsLoader.ini 方言逐字快照）：
// 未指定 ini 且当前目录没有 FsLoader.ini 时释放并采用
//（已存在的绝不覆盖）——与 solo/lister cmd 的保险策略一致。
//
//go:embed FsLoader.ini
var defaultLoaderINI []byte

const version = "0.2.0-loader-fxcore"

func main() {
	runFor := flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C；测试用）")
	fps := flag.Int("fps", 0, "内核定步上限（>0=定步模式 1/fps 步进每拍最多补 10 帧；0=全速——官方 ctor 缺省）")
	flag.Parse()
	args := flag.Args()

	fmt.Printf("9yin-server-go loader %s（FsLoader.exe + FxCore.dll 复刻）\n", version)

	// 官方 ini 发现序：argv 覆盖 > exe 派生 > FsLoader.ini 回退。
	// 缺省路径无文件时释放内置默认（官方部署目录常备 ini 的等价保障）；
	// 释放名与装载器派生名一致（game_loader.exe → game_loader.ini）。
	if len(args) == 0 {
		defaultINI := "FsLoader.ini"
		if exe, err := os.Executable(); err == nil {
			if derived := fsloader.IniNameFromExe(exe); derived != "" {
				defaultINI = derived
			}
		}
		if _, err := os.Stat(defaultINI); os.IsNotExist(err) {
			if err := os.WriteFile(defaultINI, defaultLoaderINI, 0o644); err != nil {
				fatal(err)
			}
			fmt.Printf("当前目录未找到 %s，已释放内置官方方言默认配置\n", defaultINI)
		}
	}

	// 模块注册表（官方 CDllManager 面）：fskit 为 FsLoader.ini 默认模块。
	registry := fsloader.NewModuleRegistry()
	if err := registry.Register(&fsloader.DeclModule{
		ModuleName:  "fskit.dll",
		ModuleSpace: fskit.ModuleSpace,
		ModuleEnts:  fskit.EntityTypes,
		ModuleFuncs: fskit.ExtFuncs,
	}); err != nil {
		fatal(err)
	}

	// 内核（fxcore.Core = CCore 运行时）。模块装载时实体 creator 自动
	// 收编（ArrayList/IniDocument/FileSearch/MainConsole 四类型可用）。
	var lastReport time.Time
	core := fxcore.New(registry, fxcore.Hooks{
		OnModulesLoaded: func(c *fxcore.Core) error {
			fmt.Printf("模块装载就绪：%v（实体类型 %d、扩展函数 %d）\n",
				c.Registry().Loaded(), len(fskit.EntityTypes), len(fskit.ExtFuncs))
			return nil
		},
		OnMainEntity: func(c *fxcore.Core, e *fxcore.Entity) error {
			fmt.Printf("主实体就绪：%s（类型 %s）\n", e.Name, e.Info.TypeName)
			return nil
		},
		OnProcess: func(c *fxcore.Core) {
			if !c.Config.DebugLog {
				return
			}
			if now := time.Now(); now.Sub(lastReport) >= time.Second {
				lastReport = now
				fmt.Printf("total=%.1fs 实体=%d 异步=%d stats[%s]\n",
					c.TotalTime(), c.Entities.Count(), c.Async.Count(), c.StatsView())
			}
		},
		OnShutdown: func(c *fxcore.Core) {
			fmt.Printf("CoreShutdown：total=%.3fs 实体=%d stats[%s]\n",
				c.TotalTime(), c.Entities.Count(), c.StatsView())
		},
	})
	core.SetFPSCap(*fps)

	// 官方 MainEntity=MainCon 类型来自模块注册——Go 侧以内建 MainCon
	// 承接（主容器实体，域 FxCore）。
	if err := core.Entities.AddCreator(&fxcore.EntInfo{
		TypeName: "MainCon",
		Space:    "FxCore",
		New:      func(*fskit.VarList) any { return map[string]any{} },
	}); err != nil {
		fatal(err)
	}

	loader := &fsloader.Loader{
		CoreFactory: func() fsloader.Core { return core },
	}

	// 官方 SetConsoleCtrlHandler → g_quit：Ctrl+C 触发泵循环退出。
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if *runFor > 0 {
		ctx, cancel = context.WithTimeout(ctx, *runFor)
		defer cancel()
	}

	os.Exit(loader.Run(ctx, args))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "loader:", err)
	os.Exit(1)
}
