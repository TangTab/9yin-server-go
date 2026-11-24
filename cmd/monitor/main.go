// monitor 是反作弊监控服务器进程，对应原版 game_monitor.exe +
// FsSoloMonitor.dll（小写系 solo 体系，BGserverCJ\Debug）。
//
// 官方形态：game_monitor.exe 通用 solo 宿主壳按自身文件名读
// game_monitor.ini 装载 [Module] 列表（MainEntity=SoloMonitor），
// FsSoloMonitor.dll 承载外网命令受面（[Outer] :4008，charge 同源
// 文本行协议）+ 多客户端 echo 维护口（[Echo] 缺省 :15012）+ 7 张
// 监控表（Res\MonitorLimit.ini 限额）+ CAccess→nx_monitor 告警落库。
// 本进程 = 上述全链的 Go 等价（internal/fssolomonitor）。
//
// 权威文档：docs/official/FsSoloMonitor全解与Go复刻.md
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"

	"github.com/lizhanyin/9yin-server-go/internal/fssolomonitor"
	"github.com/lizhanyin/9yin-server-go/internal/logline"
)

const version = "0.1.0-monitor"

func main() {
	var (
		configPath = flag.String("config", "", "game_monitor.ini 配置路径（默认：当前目录 game_monitor.ini，没有则释放内置默认并采用；官方方言）")
		limitPath  = flag.String("limit-ini", "../Res/MonitorLimit.ini", "MonitorLimit.ini 限额配置路径（官方固定 ..\\Res\\MonitorLimit.ini，即 bin 工作目录上一级 Res\\；不存在则释放内置官方快照）")
		outer      = flag.String("outer", "", "外网受面 addr:port（官方 [Outer]；缺省 127.0.0.1:4008；off=关闭）")
		echoAddr   = flag.String("echo", "127.0.0.1:15012", "echo 维护口 addr:port（官方 [Echo] 缺省 15012；off=关闭）")
		mysqlDSN   = flag.String("mysql-dsn", "", "MySQL DSN（官方 [Main] ConnectString 为 SQLOLEDB 方言，Go 复刻经此接真库；空=内存兜底，告警仅日志/echo）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C）")
	)
	flag.Parse()
	logline.Install("monitor")
	defer logline.Flush()

	fmt.Printf("9yin-server-go monitor %s（game_monitor.exe + FsSoloMonitor.dll 复刻）\n", version)

	cfgData := []byte(nil)
	released := false
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			fatal(err)
		}
		cfgData = data
	} else {
		_, data, created, err := fssolomonitor.LoadGameMonitorINI("game_monitor.ini")
		if err != nil {
			fatal(err)
		}
		cfgData, released = data, created
	}
	cfg, err := fssolomonitor.ParseGameMonitorINI(cfgData)
	if err != nil {
		fatal(err)
	}
	if released {
		fmt.Println("当前目录未找到 game_monitor.ini，已释放内置默认配置（修改后重启生效）")
	}

	// 限额配置（官方 LoadSystemConfig）。
	limits, _, limitCreated, err := fssolomonitor.LoadMonitorLimitINI(*limitPath)
	if err != nil {
		fatal(err)
	}
	if limitCreated {
		fmt.Printf("未找到 %s，已释放内置官方快照（键值与 BGserverCJ\\Res\\MonitorLimit.ini 等价）\n", *limitPath)
	}
	if cfg.IsOLEDB() {
		fmt.Println("注意：[Main] ConnectString 为官方 OLEDB 方言，Go 复刻无法直连；")
		fmt.Println("      未提供 -mysql-dsn 时告警仅落日志与 echo 广播（内存兜底）。")
	}

	given := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { given[f.Name] = true })

	outerAddr, outerPort := cfg.OuterAddr, cfg.OuterPort
	if given["outer"] {
		if *outer == "off" || *outer == "" {
			outerAddr, outerPort = "", 0
		} else {
			a, p, err := splitHostPort(*outer)
			if err != nil {
				fatal(err)
			}
			outerAddr, outerPort = a, p
		}
	}
	echoListen := cfg.EchoAddr
	echoPort := cfg.EchoPort
	if given["echo"] {
		if *echoAddr == "off" || *echoAddr == "" {
			echoListen = ""
		} else {
			a, p, err := splitHostPort(*echoAddr)
			if err != nil {
				fatal(err)
			}
			echoListen, echoPort = a, p
		}
	}

	ctx := context.Background()
	var sink fssolomonitor.AlertSink
	if *mysqlDSN != "" {
		sqlSink, err := fssolomonitor.OpenMySQLAlertSink(ctx, *mysqlDSN, logf)
		if err != nil {
			fatal(err)
		}
		sink = sqlSink
		fmt.Println("告警库：MySQL nx_monitor（-mysql-dsn）")
	} else {
		sink = fssolomonitor.NewMemoryAlertSink()
		fmt.Println("告警库：内存兜底（告警落日志与 echo 广播；-mysql-dsn 接真库）")
	}

	srv := fssolomonitor.New(fssolomonitor.Options{
		AppName:       cfg.AppName(),
		OuterAddr:     outerAddr,
		OuterPort:     outerPort,
		EchoAddr:      echoListen,
		EchoPort:      echoPort,
		ServerID:      cfg.ServerID,
		ConnectString: cfg.ConnectString,
		Limits:        limits,
		Sink:          sink,
		UseAppRecord:  true, // 官方 main.lua 实配
		Logf:          logf,
	})

	if *runFor > 0 {
		ctx, cancel := context.WithTimeout(ctx, *runFor)
		defer cancel()
		go func() {
			<-ctx.Done()
			srv.Stop()
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println("\n收到中断信号，关停…")
		srv.Stop()
	}()

	if err := srv.Run(ctx); err != nil {
		fatal(err)
	}
	fmt.Println("monitor 进程退出")
}

func splitHostPort(s string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return "", 0, fmt.Errorf("addr:port 形式非法 %q: %w", s, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("端口须为整数：%q", portStr)
	}
	return host, port, nil
}

func logf(format string, args ...any) {
	fmt.Printf("[monitor] "+format+"\n", args...)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "[monitor] 致命错误：%v\n", err)
	logline.Exit(1)
}
