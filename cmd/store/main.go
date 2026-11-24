// store 是 game_store.exe 的 Go 等价进程载体（小写系 solo 存储枢纽）。
//
// 与官方同型：按 `<exe基名>.ini` 发现配置（或 -config 指定），缺省释放
// 内置 game_store.ini 方言；-district/-server/-store/-inner/-broadcast
// 旗标覆盖 ini。DB 落地 -local-store（内存+JSON 快照）或 -mysql-dsn
// （官方 SQL 原文直连）；官方 [Module] DLL 装载不复刻——进程内直接
// 装配 internal/fssolostore（cmd/solomember/cmd/soloentry 同范式）。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lizhanyin/9yin-server-go/internal/fssolostore"
	"github.com/lizhanyin/9yin-server-go/internal/logline"
)

// defaultINI 快照 game_store.ini 方言（BGserverCJ\Debug\game_store.ini）。
const defaultINI = `[System]
Sleep=-1
MainEntity=SoloStore
MainScript=main

[Module]
fskit.dll
fssql.dll
fscomm.dll
fssolostore.dll

[Script]
Path=..\lua\solostore
Reload=1

[Main]
DistrictID=1
ServerID=11
StoreID=0
UseMysql=true

[Inner]
//Addr=127.0.0.1

[Broadcast]
//Addr=127.255.255.255
//Port=4001

[Account]
ConnectString=Provider=SQLOLEDB;Password=abc&4621;User ID=sa;Initial Catalog=nx_district;Data Source=192.168.33.70
ConnectPool=1

[Role]
ConnectString=Provider=SQLOLEDB;Password=abc&4621;User ID=sa;Initial Catalog=nx_base1;Data Source=192.168.33.70
ConnectPool=1

[Log]
ConnectString=
ConnectPool=

[Public]
ConnectString=
ConnectPool=

[Manage]
ConnectString=Provider=SQLOLEDB;Password=abc&4621;User ID=sa;Initial Catalog=nx_base1;Data Source=192.168.33.70
ConnectPool=1
`

func main() {
	var (
		configPath = flag.String("config", "", "game_store.ini 路径（空=内置方言）")
		district   = flag.Uint("district", 0, "覆盖 [Main] DistrictID")
		server     = flag.Uint("server", 0, "覆盖 [Main] ServerID")
		store      = flag.Uint("store", 0, "覆盖 [Main] StoreID")
		innerPort  = flag.Int("inner-port", -1, "覆盖 [Inner] Port")
		broadPort  = flag.Int("broad-port", -1, "覆盖 [Broadcast] Port")
		echoPort   = flag.Int("echo-port", -1, "覆盖 [Echo] Port（缺省 15100+StoreID）")
		mysqlDSN   = flag.String("mysql-dsn", "", "MySQL DSN（空=LocalStore 内存+JSON 快照）")
		localStore = flag.String("local-store", "game_store_state.json", "内存后端 JSON 快照路径（空=纯内存）")
	)
	flag.Parse()
	logline.Install("store")
	defer logline.Flush()

	data := []byte(defaultINI)
	if *configPath != "" {
		b, err := os.ReadFile(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read config: %v\n", err)
			logline.Exit(1)
		}
		data = b
	} else if _, err := os.Stat("game_store.ini"); err == nil {
		if b, err := os.ReadFile("game_store.ini"); err == nil {
			data = b // 官方 <exe基名>.ini 发现序
		}
	}
	cfg, err := fssolostore.ParseGameStoreINI(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse config: %v\n", err)
		logline.Exit(1)
	}
	if *district != 0 {
		cfg.DistrictID = int(*district)
	}
	if *server != 0 {
		cfg.ServerID = int(*server)
	}
	if *store != 0 {
		cfg.StoreID = int(*store)
	}
	if *innerPort >= 0 {
		cfg.InnerPort = *innerPort
	}
	if *broadPort >= 0 {
		cfg.BroadPort = *broadPort
	}
	if *echoPort >= 0 {
		cfg.EchoPort = *echoPort
	}
	cfg.FillDefaults()

	logf := func(f string, a ...any) { fmt.Println(fmt.Sprintf(f, a...)) }
	var backend fssolostore.Backend
	if *mysqlDSN != "" {
		ms, err := fssolostore.OpenMySQLStore(context.Background(), *mysqlDSN, logf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			logline.Exit(1)
		}
		backend = ms
	} else {
		backend = fssolostore.NewMemoryStore(*localStore)
	}

	srv := fssolostore.New(fssolostore.Options{
		Config:  cfg,
		Backend: backend,
		Logf:    logf,
		OnStateClosed: func() {
			logf("store server shutdown...")
			_ = backend.Close()
		},
	})
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		logline.Exit(1)
	}
	rt := srv.Config()
	fmt.Printf("app name %s\n", rt.AppName)
	fmt.Printf("echo socket %s:%d\n", rt.EchoAddr, rt.EchoPort)
	fmt.Printf("inner socket %s:%d\n", rt.InnerAddr, rt.InnerPort)
	fmt.Printf("broadcast socket %s:%d\n", rt.BroadAddr, rt.BroadPort)
	fmt.Printf("game server %d store %d started\n", rt.ServerID, rt.StoreID)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	srv.Stop()
	fmt.Println(strings.Repeat("-", 20))
}
