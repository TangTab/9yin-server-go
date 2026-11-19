// watch 是运维看门狗进程，对应原版 FsLoader.exe GameWatch.ini +
// FsWatch.dll（BGserverCJ\Debug\GameWatch.bat 形态）。
//
// 官方形态：FsLoader 宿主按 GameWatch.ini 装载 FsKit/FsSql/FsComm/FsWatch
// 四模块 + Lua\GameWatch 脚本；FsWatch 注册唯一实体 GameWatch，承载
// Outer 文本行受面（缺省 :6000，8 命令 + Monitor/Query/Update 三密码）、
// EchoSock 心跳（缺省 127.0.0.1:15006，<info state= connect= />）、
// start_app/stop_app 进程守护、远程热更新（64KB 分块 + 大写 HEX + MD5）
// 与 query 数据库代理。本进程 = 上述全链的 Go 等价（internal/fswatch）。
//
// 权威文档：docs/official/FsWatch全解与Go复刻.md
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fswatch"
)

const version = "0.1.0-watch"

func main() {
	var (
		configPath = flag.String("config", "", "GameWatch.ini 配置路径（默认：当前目录 GameWatch.ini，没有则释放内置默认并采用；官方方言）")
		outer      = flag.String("outer", "", "外网受面 addr:port（官方 [Outer]；缺省 127.0.0.1:6000；off=关闭）")
		echoAddr   = flag.String("echo", "127.0.0.1:15006", "echo 维护口 addr:port（官方 [Echo] 缺省 15006；off=关闭）")
		accessIni  = flag.String("access", "access.ini", "access.ini 数据库配置（官方 LoadAccessConfig；缺省当前目录，不存在=空表）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C）")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go watch %s（FsLoader.exe GameWatch.ini + FsWatch.dll 复刻）\n", version)

	logf := func(f string, a ...any) { fmt.Printf("[watch] "+f+"\n", a...) }

	// 配置装载：显式 -config 须存在；默认路径当前目录 GameWatch.ini，
	// 不存在则释放内置默认（已存在的绝不覆盖）——与 cmd/global 同一范式。
	path := *configPath
	if path == "" {
		path = "GameWatch.ini"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, defaultGameWatchINI, 0o644); err != nil {
				fatal(err)
			}
			fmt.Printf("已释放默认配置 %s（已存在的绝不覆盖）\n", path)
		}
	}
	access := *accessIni
	if access != "" && access != "off" {
		if _, err := os.Stat(access); os.IsNotExist(err) {
			access = ""
		}
	}
	cfg, err := fswatch.LoadConfig(path, access)
	if err != nil {
		fatal(err)
	}

	// 命令行覆盖（缺省回环可用；off=关闭）
	if *outer != "" {
		if *outer == "off" {
			cfg.OuterPort = -1
		} else if err := applyAddr(*outer, &cfg.OuterAddr, &cfg.OuterPort); err != nil {
			fatal(err)
		}
	}
	if *echoAddr == "off" {
		cfg.EchoPort = -1
	} else if *echoAddr != "" {
		if err := applyAddr(*echoAddr, &cfg.EchoAddr, &cfg.EchoPort); err != nil {
			fatal(err)
		}
	}

	if cfg.OuterPort <= 0 {
		fatal(fmt.Errorf("outer 受面已关闭（off），看门狗需要 Outer 口运行"))
	}

	watch := fswatch.NewGameWatch(cfg, nil, logf)
	srv := fswatch.NewServer(cfg, watch, logf)
	if err := srv.Start(); err != nil {
		fatal(err)
	}
	fmt.Printf("Outer socket %s:%d\n", cfg.OuterAddr, cfg.OuterPort)
	fmt.Printf("Echo socket %s:%d\n", cfg.EchoAddr, cfg.EchoPort)
	fmt.Println("Game watch server started")

	if *runFor > 0 {
		time.Sleep(*runFor)
	} else {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig
	}
	fmt.Println("watch server shutdown...")
	srv.Stop()
	fmt.Println("bye")
}

func applyAddr(v string, addr *string, port *int) error {
	host, p, err := splitHostPort(v)
	if err != nil {
		return err
	}
	*addr = host
	*port = p
	return nil
}

func splitHostPort(v string) (string, int, error) {
	for i := len(v) - 1; i >= 0; i-- {
		if v[i] == ':' {
			port := 0
			if _, err := fmt.Sscanf(v[i+1:], "%d", &port); err != nil {
				return "", 0, fmt.Errorf("bad addr %q", v)
			}
			host := v[:i]
			if host == "" {
				host = "127.0.0.1"
			}
			return host, port, nil
		}
	}
	return "", 0, fmt.Errorf("bad addr %q", v)
}

// defaultGameWatchINI 由 internal/fswatch 内嵌的官方方言默认配置提供
// （go:embed game_watch.ini）。
var defaultGameWatchINI = fswatch.DefaultGameWatchINI

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "[watch] %v\n", err)
	os.Exit(1)
}
