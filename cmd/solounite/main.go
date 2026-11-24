// solounite 是 game_unite.exe 的 Go 等价进程载体（合一服）。
//
// 官方形态：game_unite.exe 是通用 solo 宿主壳（与 game_world.exe 等
// 全部 game_*.exe 同一二进制），按 game_unite.ini 装载 11 个 solo 模块
// 全量 + MainEntity=SoloWorld + lua\solounite\main.lua——单一进程内
// 创建七个实体（SoloLog/SoloWorld/SoloStore/SoloEntry/SoloPublic/
// SoloAuction/SoloMember）共享广播口互发现，对外只露 [Outer] 2001 与
// echo 15001。本进程 = internal/fsunite（main.lua 编排的等价装配）。
//
// 权威文档：docs/official/game_unite与game_world全解与Go复刻.md。
package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fssoloentry"
	"github.com/lizhanyin/9yin-server-go/internal/fssolostore"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloworld"
	"github.com/lizhanyin/9yin-server-go/internal/fsunite"
	"github.com/lizhanyin/9yin-server-go/internal/logline"
	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

//go:embed game_unite.ini
var defaultINI []byte // 官方方言原字节快照（GBK 注释 + CRLF；键面 ASCII）

const version = "0.1.0-solounite"

func main() {
	var (
		configPath = flag.String("config", "", "game_unite.ini 路径（空=当前目录发现，缺则释放内置官方快照）")
		scenesINI  = flag.String("scenes", `..\res\scenes.ini`, "场景配置（scenes.ini；空=不装载）")
		district   = flag.Int("district", 0, "覆盖 [Main] DistrictID")
		server     = flag.Int("server", 0, "覆盖 [Main] ServerID")
		broadPort  = flag.Int("broad-port", 0, "覆盖 [Broadcast] Port")
		outerPort  = flag.Int("outer-port", 0, "覆盖 [Outer] Port")
		echoPort   = flag.Int("echo-port", 0, "覆盖 [Echo] Port")
		noLicense  = flag.Bool("no-license", false, "跳过 license 门（仅联调用）")
		genLic     = flag.String("gen-license", "", "为本机 IP 生成 solo lic 后退出（值=IP；内部形态 256B）")
		mysqlDSN   = flag.String("mysql-dsn", "", "MySQL DSN（空=查 NINEYIN_MYSQL_DSN，仍空=内存库；启用后账号落 nx_district.dc_accounts、角色落 nx_roles）")
		captureDir = flag.String("capture-dir", "", "客户端线路双向抓取落盘目录（{ip}_{port}_c2s/s2c.bin，EE+XOR wire 原样；空=查 NINEYIN_CAPTURE_DIR）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C；冒烟用）")
	)
	flag.Parse()
	logline.Install("solounite")
	defer logline.Flush()

	fmt.Printf("9yin-server-go solounite %s\n", version)

	// 配置发现序：-config > ./game_unite.ini > 释放内置官方快照（不覆盖）
	data := defaultINI
	if *configPath != "" {
		b, err := os.ReadFile(*configPath)
		if err != nil {
			fatal(err)
		}
		data = b
	} else if _, err := os.Stat("game_unite.ini"); err == nil {
		if b, err := os.ReadFile("game_unite.ini"); err == nil {
			data = b
		}
	} else {
		if err := os.WriteFile("game_unite.ini", defaultINI, 0644); err != nil {
			fatal(err)
		}
		fmt.Println("当前目录未找到 game_unite.ini，已释放内置官方快照（修改后重启生效）")
	}
	cfg, err := fsunite.ParseGameUniteINI(data)
	if err != nil {
		fatal(err)
	}
	if *district != 0 {
		cfg.DistrictID = *district
	}
	if *server != 0 {
		cfg.ServerID = *server
	}
	if *broadPort != 0 {
		cfg.BroadPort = *broadPort
	}
	if *outerPort != 0 {
		cfg.OuterPort = *outerPort
	}
	if *echoPort != 0 {
		cfg.EchoPort = *echoPort
	}

	// license 生成器模式先行（solo 方言：md5(盐+IP) 派生 + 求和校验）
	if *genLic != "" {
		path := *genLic + ".lic"
		if err := os.WriteFile(path, fssoloentry.GenerateLicense(*genLic), 0644); err != nil {
			fatal(err)
		}
		fmt.Printf("已生成 %s（内部形态 256B，魔数对 %#X/%#X）\n",
			path, fssoloentry.OfficialMagic1, fssoloentry.OfficialMagic2)
		return
	}

	// license 门（官方 lookup_thread：0x7D0ms 等待，超时 program abort!；
	// license = <OuterAddr>.lic）
	var lic *fssoloentry.License
	if !*noLicense {
		ip := fssoloentry.LocalIPString(cfg.OuterAddr)
		l, err := fssoloentry.LoadLicenseFile(cfg.LicensePath(), ip, time.Now())
		if err != nil {
			fmt.Fprintf(os.Stderr, "solounite: %v未授权（不广播 + 每拍告警）\n", err)
		} else {
			lic = &l
			if l.Licensed() {
				fmt.Printf("license 就绪：%s 魔数对 %#X/%#X\n",
					cfg.LicensePath(), l.Fields[0], l.Fields[1])
			} else {
				fmt.Printf("license 解码成功但魔数对失配：%#X/%#X（按未授权处理）\n",
					l.Fields[0], l.Fields[1])
			}
		}
	}

	// 传输 XOR 键（官方 MessageEncodeKey hex→u32）
	var netOptions fsnet.Options
	if cfg.MessageEncodeKey != "" {
		keyValue, err := strconv.ParseUint(cfg.MessageEncodeKey, 16, 32)
		if err != nil {
			fatal(fmt.Errorf("MessageEncodeKey 非法: %w", err))
		}
		key := fsnet.NewKey(uint32(keyValue))
		netOptions = fsnet.Options{
			ReadCipher:  fsnet.NewXorCipher(key),
			WriteCipher: fsnet.NewXorCipher(key),
		}
	}

	logf := func(format string, args ...any) { fmt.Printf(format+"\n", args...) }

	// 客户端线路抓取（fsnet captureStream：gateway 每连接双向 wire 落盘，
	// 与外服对拍基准同构，可直接喂解帧/对拍工具链）
	if dir := *captureDir; dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fatal(err)
		}
		_ = os.Setenv("NINEYIN_CAPTURE_DIR", dir)
		fmt.Printf("线路抓取：客户端双向字节落盘 %s（*_c2s.bin/*_s2c.bin）\n", dir)
	}


	// 存储后端：-mysql-dsn > env NINEYIN_MYSQL_DSN（env.server）>
	// game_unite.ini [Role] ConnectString（官方 SQLOLEDB 方言公共解析）
	// > 内存库（重启即丢）——公共解析方法 cfg.MySQLDSN()
	dsn := *mysqlDSN
	if dsn == "" {
		dsn = cfg.MySQLDSN()
	}
	var backend fssolostore.Backend
	if dsn != "" {
		mb, err := fssolostore.OpenMySQLStore(context.Background(), dsn, logline.Tagged("store", logf))
		if err != nil {
			fatal(fmt.Errorf("mysql 后端不可用: %w", err))
		}
		backend = mb
		fmt.Printf("存储后端：MySQL（%s）——账号 nx_district.dc_accounts / 角色 nx_roles\n", dsn)
	} else {
		fmt.Println("存储后端：内存库（重启即丢；-mysql-dsn 或 NINEYIN_MYSQL_DSN 启用持久化）")
	}

	var scenes []fssoloworld.SceneSpec
	if *scenesINI != "" {
		if specs, err := fssoloworld.LoadScenesINI(*scenesINI); err == nil && len(specs) > 0 {
			// scenes.ini 是集群快照（场景分给 member 1..N）——合一形态
			// 只承载 member 0，全部重映射，否则场景永远送不到、world 不 Open。
			for i := range specs {
				specs[i].Member = 0
			}
			scenes = specs
			fmt.Println(fssoloworld.FormatScenesCount(len(specs)))
		}
	}
	u := fsunite.Assemble(cfg, fsunite.Options{
		License:      lic,
		NetOptions:   netOptions,
		Scenes:       scenes,
		StoreBackend: backend,
		Logf:         logf,
		Quit:         func() { fmt.Println("unite server shutdown..."); logline.Exit(0) },
	})
	if err := u.Start(); err != nil {
		fatal(err)
	}
	for _, line := range u.Banner() {
		fmt.Println(line)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	if *runFor > 0 {
		select {
		case <-time.After(*runFor):
		case <-sig:
		}
	} else {
		<-sig
	}
	fmt.Println("unite server shutdown...")
	u.Stop()
	fmt.Println(strings.Repeat("-", 20))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "solounite:", err)
	logline.Exit(1)
}
