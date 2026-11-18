// charge 是充值计费中心进程（对应原版 GameCharge.bat → FsLoader.exe
// GameCharge.ini → FsCharge.dll 形态）。承载 internal/charge 的行文本
// 计费协议服务：游戏服（官方 FsServer.dll GameWorld::CChargeSockHook）
// 从 [Outer] 端口接入，走 register/login/logout/renew/... 18 命令方言。
//
// 计费后端两档：-dsn 给出时接本机 MySQL nx_district.dc_accounts
// （口令=大写 MD5 比对、points/charge_mode/valid_time 读路径）；
// 缺省 FreeBackend（开发免费模式：全放行、0 点、类型 2）。
package main

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	gonet "net"
	"os"
	"os/signal"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/lizhanyin/9yin-server-go/internal/charge"
)

//go:embed GameCharge.ini
var defaultGameChargeINI []byte

const version = "0.1.0-charge"

func main() {
	var (
		addr   = flag.String("addr", "", "监听地址（显式给出时优先于配置 [Outer]）")
		config = flag.String("config", "", "配置文件路径（默认：当前目录 GameCharge.ini，没有则释放内置默认；官方 GameCharge.ini 方言）")
		dsn    = flag.String("dsn", "", "MySQL DSN（空=FreeBackend 免费开发模式；例 nineyin:nineyin123@tcp(127.0.0.1:3306)/nx_district）")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go charge %s\n", version)

	var cfgData []byte
	released := false
	if *config != "" {
		data, err := os.ReadFile(*config)
		if err != nil {
			fatal(err)
		}
		cfgData = data
	} else {
		data, created, err := charge.LoadOrCreateGameChargeINI("GameCharge.ini", defaultGameChargeINI)
		if err != nil {
			fatal(err)
		}
		cfgData, released = data, created
	}
	cfg, err := charge.ParseGameChargeINI(cfgData)
	if err != nil {
		fatal(err)
	}
	if released {
		fmt.Println("当前目录未找到 GameCharge.ini，已释放内置默认配置（修改后重启生效）")
	}

	given := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { given[f.Name] = true })
	listen := gonet.JoinHostPort(cfg.Addr, strconv.Itoa(cfg.Port))
	if given["addr"] {
		listen = *addr
	}

	host, portS, _ := gonet.SplitHostPort(listen)
	cfg.Addr, cfg.Port = host, func() int {
		n, err := strconv.Atoi(portS)
		if err != nil {
			fatal(fmt.Errorf("端口解析失败: %w", err))
		}
		return n
	}()

	var backend charge.Backend
	backendName := "free(开发免费模式)"
	if *dsn != "" {
		db, err := sql.Open("mysql", *dsn)
		if err != nil {
			fatal(fmt.Errorf("打开 MySQL: %w", err))
		}
		if err := db.Ping(); err != nil {
			fatal(fmt.Errorf("连接 MySQL: %w", err))
		}
		backend = &charge.MySQLBackend{DB: db}
		backendName = "mysql(nx_district.dc_accounts)"
	}

	srv := charge.NewServer(cfg, backend)
	srv.Logf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	listenAddr, err := srv.Listen()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("已加载配置：GameID=%d ServerIDOffset=%d PointMinutes=%d AwakeTime=%ds FlushErrorMax=%d MaxClient=%d\n",
		cfg.GameID, cfg.ServerIDOffset, cfg.PointMinutes, cfg.AwakeSec, cfg.FlushErrorMax, cfg.MaxClient)
	fmt.Printf("Outer socket %s（backend=%s）\n", listenAddr, backendName)
	fmt.Println("Game charge server started")

	done := make(chan struct{})
	go func() { srv.Serve(); close(done) }()

	<-ctx.Done()
	fmt.Println("charge server shutdown...")
	srv.Close()
	<-done
	fmt.Println("charge: 已停止")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "charge:", err)
	os.Exit(1)
}
