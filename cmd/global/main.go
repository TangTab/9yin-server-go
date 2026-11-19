// global 是全局名字服务器进程，对应原版 game_global.exe +
// FsSoloGlobal.dll（小写系 solo 体系，BGserverCJ\Debug）。
//
// 官方形态：game_global.exe 是通用 solo 宿主壳（FsSoloLoader 同源，
// 按自身文件名读 game_global.ini 装载 [Module] 列表，
// MainEntity=SoloGlobal + MainScript=main），FsSoloGlobal.dll 承载
// 外网命令受面（[Outer] 缺省 :4003）+ echo 维护口（[Echo] 缺省
// :15012）+ 域表（role→nx_role_names / guild→nx_guild_names）+
// CAccess→MysqlDBSystem 的名字登记/释放 SQL。本进程 = 上述全链的
// Go 等价（internal/fssologlobal）。
//
// 权威文档：docs/official/FsSoloGlobal全解与Go复刻.md
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"

	"github.com/lizhanyin/9yin-server-go/internal/fssologlobal"
)

const version = "0.1.0-global"

func main() {
	var (
		configPath = flag.String("config", "", "game_global.ini 配置路径（默认：当前目录 game_global.ini，没有则释放内置默认并采用；官方方言）")
		outer      = flag.String("outer", "", "外网受面 addr:port（官方 [Outer]；缺省 127.0.0.1:4003；off=关闭）")
		echoAddr   = flag.String("echo", "127.0.0.1:15012", "echo 维护口 addr:port（官方 [Echo] 缺省 15012；off=关闭）")
		mysqlDSN   = flag.String("mysql-dsn", "", "MySQL DSN（官方 [Main] ConnectString 为 SQLOLEDB 方言，Go 复刻经此接真库；空=本地名册兜底）")
		localStore = flag.String("local-store", "data/global/names.json", "本地名册快照（Go 扩展：无 MySQL 时的名字登记兜底；off=纯内存）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C）")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go global %s（game_global.exe + FsSoloGlobal.dll 复刻）\n", version)

	// 配置装载：显式 -config 须存在；默认路径当前目录 game_global.ini，
	// 不存在则释放内置默认（已存在的绝不覆盖）。
	cfgData := []byte(nil)
	released := false
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			fatal(err)
		}
		cfgData = data
	} else {
		data, created, err := loadDefaultINI("game_global.ini")
		if err != nil {
			fatal(err)
		}
		cfgData, released = data, created
	}
	cfg, err := fssologlobal.ParseGameGlobalINI(cfgData)
	if err != nil {
		fatal(err)
	}
	if released {
		fmt.Println("当前目录未找到 game_global.ini，已释放内置默认配置（修改后重启生效）")
	}
	if cfg.IsOLEDB() && *mysqlDSN == "" {
		fmt.Println("注意：[Main] ConnectString 为官方 OLEDB 方言，Go 复刻无法直连；")
		fmt.Println("      未提供 -mysql-dsn，使用本地名册兜底（data/global/names.json）。")
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

	// DB 装配：-mysql-dsn 优先（建缺表）；否则本地名册兜底。
	ctx := context.Background()
	var store fssologlobal.DBAccess
	if *mysqlDSN != "" {
		sqlStore, err := fssologlobal.OpenSQLStore(ctx, *mysqlDSN, logf)
		if err != nil {
			fatal(err)
		}
		for _, nt := range fssologlobal.DefaultDomains().Entries() {
			if err := sqlStore.EnsureTable(ctx, nt[1]); err != nil {
				fatal(fmt.Errorf("建表 %s: %w", nt[1], err))
			}
		}
		store = sqlStore
		fmt.Println("名字库：MySQL（-mysql-dsn）")
	} else {
		path := *localStore
		if path == "off" {
			path = ""
		}
		ls, err := fssologlobal.NewLocalStore(path, logf)
		if err != nil {
			fatal(err)
		}
		store = ls
		fmt.Printf("名字库：本地名册（%s）\n", storePath(path))
	}

	srv := fssologlobal.New(fssologlobal.Options{
		AppName:      "global",
		OuterAddr:    outerAddr,
		OuterPort:    outerPort,
		EchoAddr:     echoListen,
		EchoPort:     echoPort,
		Store:        store,
		UseAppRecord: true, // 官方 main.lua 实配
		Logf:         logf,
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

	if outerAddr == "" && outerPort == 0 {
		// 官方无此形态（外网受面必开）；Go 侧供纯 echo 调试。
		fmt.Println("外网受面：关闭（-outer off）")
	}
	if err := srv.Run(ctx); err != nil {
		fatal(err)
	}
	fmt.Println("global 进程退出")
}

func loadDefaultINI(path string) (data []byte, created bool, err error) {
	_, data, created, err = fssologlobal.LoadGameGlobalINI(path)
	return data, created, err
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

func storePath(p string) string {
	if p == "" {
		return "纯内存"
	}
	return p
}

func logf(format string, args ...any) {
	fmt.Printf("[global] "+format+"\n", args...)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "[global] 致命错误：%v\n", err)
	os.Exit(1)
}
