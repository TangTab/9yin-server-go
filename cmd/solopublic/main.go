// solopublic 是公共数据服务器进程，对应原版 game_public.exe +
// FsSoloPublic.dll（小写系 solo 体系，BGserverCJ\Debug）。
//
// 官方形态：game_public.exe 通用 solo 宿主壳按自身文件名读
// game_public.ini（MainEntity=SoloPublic），FsSoloPublic.dll 承载
// domain/nx_domains + guild/nx_guilds 两公共数据空间、CRelation 好友
// 关系 + SNS 动态流、CTrade 跨服寄售行、CNameUid 角色名表、world
// scene/player/role 注册表镜像；inner 受面（端口 0=系统分配）、echo
// 维护口 :15003、UDP :15000 自宣 kind=4/magic=0x3001；装载链
// store 0x2D→0x22 / 0x30 / 0x3A→0x23 / 0x28 / 0x24→0x19 / 0x25→0x1A，
// 回存 0x26（全 store）。本进程 = 上述全链的 Go 等价
// （internal/fssolopublic）。
//
// 权威文档：docs/official/FsSoloPublic全解与Go复刻.md
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fssolopublic"
)

const version = "0.1.0-solopublic"

func main() {
	var (
		configPath = flag.String("config", "", "game_public.ini 配置路径（默认：当前目录 game_public.ini，没有则释放内置默认并采用；官方方言）")
		echoAddr   = flag.String("echo", "127.0.0.1:15003", "echo 维护口 addr:port（官方 [Echo] 缺省 15003；off=关闭）")
		innerAddr  = flag.String("inner", "", "inner 受面 addr:port（官方 [Inner]；缺省 ini 值/127.0.0.1:0 系统分配）")
		broadPort  = flag.Int("broad-port", 0, "UDP 广播口（官方 [Broadcast] 缺省 15000；0=用 ini）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C）")
		verbose    = flag.Bool("v", false, "打印官方 trace 面日志")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go solopublic %s（game_public.exe + FsSoloPublic.dll 复刻）\n", version)

	// 配置装载：显式 -config 须存在；默认路径当前目录 game_public.ini，
	// 不存在则释放内置默认（已存在的绝不覆盖）。
	var cfgData []byte
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			fatal(err)
		}
		cfgData = data
	} else {
		_, data, created, err := fssolopublic.LoadGamePublicINI("game_public.ini")
		if err != nil {
			fatal(err)
		}
		cfgData = data
		if created {
			fmt.Println("已释放内置默认配置 game_public.ini")
		}
	}
	cfg, err := fssolopublic.ParseGamePublicINI(cfgData)
	if err != nil {
		fatal(err)
	}
	// 命令行覆写（部署便捷；缺省全走官方方言）
	if *innerAddr != "" {
		host, port, err := splitHostPort(*innerAddr)
		if err != nil {
			fatal(err)
		}
		cfg.InnerAddr, cfg.InnerPort = host, port
	}
	if *broadPort != 0 {
		cfg.BroadPort = *broadPort
	}
	if a := *echoAddr; a != "" && a != "off" {
		host, port, err := splitHostPort(a)
		if err != nil {
			fatal(err)
		}
		cfg.EchoAddr, cfg.EchoPort = host, port
	}

	logf := func(format string, args ...any) {
		if *verbose {
			fmt.Printf("[solopublic] "+format+"\n", args...)
		}
	}
	srv := fssolopublic.New(cfg, fssolopublic.Options{Logf: logf})
	if err := srv.Start(); err != nil {
		fatal(err)
	}
	fmt.Printf("app name %s\n", fssolopublic.AppName(cfg.ServerID))
	fmt.Printf("echo socket %s:%d\n", cfg.EchoAddr, cfg.EchoPort)
	fmt.Printf("inner socket %s:%d（实际端口）\n", cfg.InnerAddr, srv.InnerPort())
	fmt.Printf("broadcast socket %s:%d\n", cfg.BroadAddr, cfg.BroadPort)
	fmt.Printf("game public server %d started\n", cfg.ServerID)

	// TimerCheck 心跳（官方 CReactor::CreateTimer 1s）：自宣 + 状态机 tick
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				srv.Tick()
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	if *runFor > 0 {
		select {
		case <-time.After(*runFor):
		case <-sig:
		}
	} else {
		<-sig
	}
	cancel()
	srv.Stop()
	fmt.Println("public server shutdown...")
}

func splitHostPort(s string) (string, int, error) {
	var host, portStr string
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			host, portStr = s[:i], s[i+1:]
			break
		}
	}
	if portStr == "" {
		return "", 0, fmt.Errorf("addr:port 格式错误：%q", s)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		return "", 0, fmt.Errorf("端口非法：%q", portStr)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return host, port, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "fatal:", err)
	os.Exit(1)
}
