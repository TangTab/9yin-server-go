// sololoader 是小写系 solo 集群进程壳入口，对应原版 FsSoloLoader.exe
//（生产部署形态：game_{lister,entry,world,member,public,store,log,
// global,auction,unite,monitor}.exe 为同一构建的改名拷贝）：
//
//	game_solo.exe [ini名] [附加参数...]
//
// argv 约定与官方逐字一致（裸跑形态）：args[0] 覆盖 ini 名（缺省从
// 可执行文件名**小写**派生——game_lister.exe → game_lister.ini，回退
// 字面量 fssololoader.ini），args[1..] 为附加参数。改名 exe + 对应
// ini 即换角色（官方一套 exe 多角色机制）。
//
// 装载序列/主实体门/主实体帧泵/Ctrl+C 吞掉语义/ERROR*.log 崩溃面 =
// internal/fssoloader（FsSoloLoader.exe 复刻）；内核 =
// internal/fxcore（FxCore.dll 复刻）。默认注册 fskit 模块（官方
// game_*.ini [Module] 表首行即 fskit.dll；solo 角色模块随复刻进度增补
// ——装载器本体零改动，ini 名即角色）。
package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
	fsloader "github.com/lizhanyin/9yin-server-go/internal/fsloader"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloader"
	"github.com/lizhanyin/9yin-server-go/internal/fxcore"
)

// defaultSoloINI 内置缺省配置（官方 game_*.ini 小写系方言快照式演示：
// [System]/[Module]/[Script] 三节 + 官方注释逐类）：未指定 ini 且当前
// 目录没有派生名文件时释放并采用（已存在的绝不覆盖）。
//
//go:embed fssololoader.ini
var defaultSoloINI []byte

// demoMainLua 主脚本占位（fxcore Startup 要求 ScriptPath 下
// <MainScript>.lua 存在；官方由部署目录 ..\lua\solo* 提供）。
//
//go:embed demo_main.lua
var demoMainLua []byte

const version = "0.1.0-fssololoader-replica"

func main() {
	runFor := flag.Duration("run", 0, "运行时长（0=直到停机信号；测试用）")
	fps := flag.Int("fps", 0, "内核定步上限（>0=定步模式；0=全速——官方 ctor 缺省）")
	ctrlC := flag.Bool("ctrl-c", false, "放开官方 Ctrl+C 吞掉语义（发送即停服；官方为忽略）")
	flag.Parse()
	args := flag.Args()

	fmt.Printf("9yin-server-go sololoader %s（FsSoloLoader.exe + FxCore.dll 复刻）\n", version)

	// 官方 ini 发现序：argv 覆盖 > 小写派生 > fssololoader.ini 回退。
	// 派生名缺文件时释放内置缺省（ini + 演示 main.lua），绝不覆盖已有。
	if len(args) == 0 {
		defaultINI := fssoloader.DefaultIniName
		if exe, err := os.Executable(); err == nil {
			if derived := fssoloader.IniNameFromExe(exe); derived != "" {
				defaultINI = derived
			}
		}
		if _, err := os.Stat(defaultINI); os.IsNotExist(err) {
			if err := os.WriteFile(defaultINI, defaultSoloINI, 0o644); err != nil {
				fatal(err)
			}
			if _, err := os.Stat("main.lua"); os.IsNotExist(err) {
				if err := os.WriteFile("main.lua", demoMainLua, 0o644); err != nil {
					fatal(err)
				}
			}
			fmt.Printf("当前目录未找到 %s，已释放内置官方方言缺省配置\n", defaultINI)
		}
	}

	// 模块注册表（官方 CDllManager 面）：fskit 为 game_*.ini [Module]
	// 表首行常客；solo 角色模块（fscomm/fssoloentry/fssololister/…）随
	// 复刻进度在此增补——装载器本体与发现序零改动。
	registry := fsloader.NewModuleRegistry()
	if err := registry.Register(&fsloader.DeclModule{
		ModuleName:  "fskit.dll",
		ModuleSpace: fskit.ModuleSpace,
		ModuleEnts:  fskit.EntityTypes,
		ModuleFuncs: fskit.ExtFuncs,
	}); err != nil {
		fatal(err)
	}

	// 内核（fxcore.Core = CCore 运行时）+ fssoloader 适配（主实体帧面）。
	var lastReport time.Time
	core := fxcore.New(registry, fxcore.Hooks{
		OnModulesLoaded: func(c *fxcore.Core) error {
			fmt.Printf("模块装载就绪：%v（实体类型 %d、扩展函数 %d）\n",
				c.Registry().Loaded(), len(fskit.EntityTypes), len(fskit.ExtFuncs))
			return nil
		},
		OnMainEntity: func(c *fxcore.Core, e *fxcore.Entity) error {
			fmt.Printf("主实体就绪：%s（类型 %s）——solo 帧契约由装载器泵驱动\n", e.Name, e.Info.TypeName)
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

	// 官方 MainEntity 类型来自模块注册——MainCon 内建承接（主容器实体，
	// 域 FxCore；官方 FsKit.dll 注册面）。
	if err := core.Entities.AddCreator(&fxcore.EntInfo{
		TypeName: "MainCon",
		Space:    "FxCore",
		New:      func(*fskit.VarList) any { return map[string]any{} },
	}); err != nil {
		fatal(err)
	}

	loader := &fssoloader.SoloLoader{
		CoreFactory: func() fssoloader.SoloCore { return fssoloader.FxCoreSoloCore(core) },
		CtrlCStops:  *ctrlC,
	}

	// 停机面交给壳层（官方 console_ctrl_handler 语义：Ctrl+C 吞掉、
	// SIGTERM/ctx 停机）——此处不再 NotifyContext(os.Interrupt) 抢占。
	ctx := context.Background()
	if *runFor > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *runFor)
		defer cancel()
	}

	os.Exit(loader.Run(ctx, args))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "sololoader:", err)
	os.Exit(1)
}
