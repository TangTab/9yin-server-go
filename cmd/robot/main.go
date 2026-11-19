// robot 是压测机器人客户端进程，对应原版 game_robot.exe +
// FsSoloRobot.dll（solo 体系，BGserverCJ\Debug）。
//
// 官方形态：game_robot.exe 是通用 solo 宿主壳（按自身文件名读
// game_robot.ini 装载 [Module] 列表，MainEntity=SoloRobot +
// MainScript=main），FsSoloRobot.dll 向 [Outer]（网关，缺省
// 127.0.0.1:2001）发起大量 TCP 连接，每连接模拟一个玩家：
// 登录 → 选角/建角 → 入景 → 周期随机移动；配 echo 维护口
// （[Echo] 缺省 127.0.0.1:15009）。本进程 = 上述全链的 Go 等价
// （internal/fssolorobot），作为 Go 版 solo 集群的官方语义负载源
// 与网关客户端线路协议回归工具。
//
// 权威文档：docs/official/FsSoloRobot全解与Go复刻.md
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fssolorobot"
)

const version = "0.1.0-robot"

func main() {
	var (
		configPath = flag.String("config", "", "game_robot.ini 配置路径（默认：当前目录 game_robot.ini，没有则释放内置默认并采用；官方方言）")
		outer      = flag.String("outer", "", "外连目标 addr:port（官方 [Outer]；缺省 127.0.0.1:2001）")
		echoAddr   = flag.String("echo", "", "echo 维护口 addr:port（官方 [Echo] 缺省 127.0.0.1:15009；off=关闭）")
		maxConn    = flag.Int("max", 0, "连接数量上限（官方 [Main] MaxPlayer；0=用配置）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C）")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go robot %s（game_robot.exe + FsSoloRobot.dll 复刻）\n", version)

	cfgData, released, err := fssolorobot.LoadOrReleaseDefaultINI(*configPath)
	if err != nil {
		fatal(err)
	}
	cfg, err := fssolorobot.ParseGameRobotINI(cfgData)
	if err != nil {
		fatal(err)
	}
	if released {
		fmt.Println("当前目录未找到 game_robot.ini，已释放内置默认配置（修改后重启生效）")
	}

	given := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { given[f.Name] = true })
	if given["outer"] && *outer != "" {
		a, p, err := splitHostPort(*outer)
		if err != nil {
			fatal(err)
		}
		cfg.OuterAddr, cfg.OuterPort = a, p
	}
	if given["echo"] {
		if *echoAddr == "off" || *echoAddr == "" {
			cfg.EchoAddr = ""
		} else {
			a, p, err := splitHostPort(*echoAddr)
			if err != nil {
				fatal(err)
			}
			cfg.EchoAddr, cfg.EchoPort = a, p
		}
	}
	if given["max"] && *maxConn > 0 {
		cfg.MaxConnects = *maxConn
	}

	fmt.Printf("outer %s:%d  echo %s:%d  max=%d concurrent=%d speed=%d move=%ds\n",
		cfg.OuterAddr, cfg.OuterPort, cfg.EchoAddr, cfg.EchoPort,
		cfg.MaxConnects, cfg.ConnectConcurrent, cfg.ConnectSpeed, cfg.MoveSeconds)

	done := make(chan struct{})
	robot := fssolorobot.NewRobot(cfg,
		func(format string, args ...any) { fmt.Printf(format+"\r\n", args...) },
		nil, // 真拨号
		func() { close(done) }, // echo quit → 官方 nx_quit
	)
	if err := robot.Start(); err != nil {
		fatal(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	var expire <-chan time.Time
	if *runFor > 0 {
		timer := time.NewTimer(*runFor)
		expire = timer.C
		defer timer.Stop()
	}

	select {
	case <-sig:
		fmt.Println("\n收到退出信号")
	case <-done:
	case <-expire:
		fmt.Println("运行时长已到")
	}
	robot.Stop()
	fmt.Println("robot exited")
}

func splitHostPort(s string) (string, int, error) {
	var port int
	var host string
	i := lastIndexByte(s, ':')
	if i < 0 {
		return "", 0, fmt.Errorf("地址缺少端口: %q", s)
	}
	host, portStr := s[:i], s[i+1:]
	if j := lastIndexByte(host, ':'); j >= 0 { // IPv6 字面量
		host = host[1 : len(host)-1]
	}
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("非法端口: %q", s)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return host, port, nil
}

func lastIndexByte(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "fatal:", err)
	os.Exit(1)
}
