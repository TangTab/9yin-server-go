// logserver 是日志汇聚服务器进程，对应原版 game_log.exe +
// FsSoloLog.dll（小写系 solo 体系，BGserverCJ\Debug）。
//
// 官方形态：game_log.exe 是通用 solo 宿主壳（与 game_lister.exe/
// game_global.exe 等同一二进制，按自身文件名读 game_log.ini 装载
// fssololog.dll），FsSoloLog.dll 是整个 solo 集群的日志汇聚服：
// 各业务进程（world=1/public=2/auction=3/monitor=0x32/global=0x38…）
// 经 Windows 命名共享内存 memory_fs_solo_log_0 把 CORE_TRACE 日志行
// 投递到各自槽位，本进程 0.1s 一轮把积压行落盘成按进程分文件的
// 日志（每 tick 每槽至多 100 行），附带单连接 TCP 维护口（echo，
// 缺省 127.0.0.1:15011）与 AutoExit 看门狗（world 消失 5s 自退）。
// 本进程 = 上述全链的 Go 等价（internal/fssololog）。
//
// 权威文档：docs/official/FsSoloLog全解与Go复刻.md
package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fssololog"
)

const version = "0.1.0-logserver"

func main() {
	var (
		configPath = flag.String("config", "", "game_log.ini 配置路径（默认：当前目录 game_log.ini，没有则释放内置默认并采用；官方方言）")
		echoAddr   = flag.String("echo", "127.0.0.1:15011", "echo 维护口 addr:port（官方 [Echo] 缺省；off=关闭）")
		logDir     = flag.String("logdir", "log", "日志落盘目录（官方 main.lua LogFileDir=工作路径 log\\）")
		shareName  = flag.String("share", "", "共享内存名（默认：Windows 开官方同名 memory_fs_solo_log_0，可混部互通官方进程；off=进程内缓冲）")
		autoExit   = flag.Bool("auto-exit", true, "AutoExit 看门狗（官方 ctor 缺省开：槽 1 FsSoloWorld 消失 5s 自退）")
		noEveryDay = flag.Bool("no-everyday", false, "关闭 EveryDayAlone 跨天换档（官方 main.lua 置 true）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C）")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go logserver %s（game_log.exe + FsSoloLog.dll 复刻）\n", version)

	// 配置装载：显式 -config 须存在；默认路径当前目录 game_log.ini，
	// 不存在则释放内置默认（已存在的绝不覆盖）。
	var cfgData []byte
	released := false
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			fatal(err)
		}
		cfgData = data
	} else {
		_, data, created, err := fssololog.LoadGameLogINI("game_log.ini")
		if err != nil {
			fatal(err)
		}
		cfgData, released = data, created
	}
	cfg, err := fssololog.ParseGameLogINI(cfgData)
	if err != nil {
		fatal(err)
	}
	if released {
		fmt.Println("当前目录未找到 game_log.ini，已释放内置默认配置（修改后重启生效）")
	}
	fmt.Printf("配置：MainEntity=%s MainScript=%s 模块=%s 脚本=%s\n",
		cfg.MainEntity, cfg.MainScript, join(cfg.Modules, "+"), cfg.ScriptPath)

	// 旗标覆盖（缺省取 ini/官方缺省）。
	given := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { given[f.Name] = true })
	echoListen, echoPort := cfg.EchoAddr, cfg.EchoPort
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
	dir := *logDir
	if !given["logdir"] {
		dir = "log" + string(os.PathSeparator)
	}
	share := *shareName
	if !given["share"] && runtime.GOOS != "windows" {
		share = "off" // 非 Windows 无命名共享内存，直接进程内缓冲
	}

	// main.lua：创建日志目录（log\）。
	if err := os.MkdirAll(filepath.Clean(dir), 0o755); err != nil {
		fatal(fmt.Errorf("创建日志目录 %s: %w", dir, err))
	}

	srv := fssololog.New(fssololog.Options{
		AppName:       "SoloLog",
		EchoAddr:      echoListen,
		EchoPort:      echoPort,
		LogFileDir:    dir,
		EveryDayAlone: !*noEveryDay,
		AutoExit:      *autoExit,
		ShareName:     shareNameOff(share),
		Logf:          logf,
		Printf:        printf,
	})

	if *runFor > 0 {
		time.AfterFunc(*runFor, srv.Stop)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println("\n收到中断信号，关停…")
		srv.Stop()
	}()

	// 官方 SoloLog::Start 五步；互斥失败走官方出口：
	// printf 后 exit(0)（唯一直接退出口）。
	if err := srv.Start(); err != nil {
		if errors.Is(err, fssololog.ErrExclusive) {
			os.Exit(0)
		}
		fatal(err)
	}

	// main.lua 启动横幅（SetConsoleColor + WriteLine 三连）。
	srv.SetConsoleColor("LightBlue")
	srv.WriteLine("log server")
	srv.WriteLine(fmt.Sprintf("echo socket %s:%d", echoListen, echoPort))
	srv.WriteLine("game log server started")

	// 宿主帧泵等价：阻塞直至 Stop（信号/runFor/AutoExit/echo quit）。
	<-srv.Stopped()
	fmt.Println("log server 进程退出")
}

// shareNameOff "off" 归零（Server 回退进程内缓冲）。
func shareNameOff(share string) string {
	if share == "off" {
		return ""
	}
	return share
}

func join(list []string, sep string) string {
	out := ""
	for i, s := range list {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
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
	fmt.Printf("[logserver] "+format+"\n", args...)
}

func printf(format string, args ...any) {
	fmt.Printf(format, args...) // 官方 printf 面：WriteLine/AutoExit 文案
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "[logserver] 致命错误：%v\n", err)
	os.Exit(1)
}
