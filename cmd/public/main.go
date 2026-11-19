// public 是公共数据进程（对应原版 FsLoader.exe GamePublic.ini +
// FsPublic.dll + FsPubLogic.dll）：
// 跨场景共享数据空间的权威侧（pubdata/guild），member 持只读镜像；
// world 控制状态、store 持久化（PUB4 blob）、FsPubLogic 玩法域挂载。
//
// 权威文档：docs/official/FsPublic全解与Go复刻.md。
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/lizhanyin/9yin-server-go/internal/fspublic"
)

//go:embed GamePublic.ini
var defaultGamePublicINI []byte

const version = "0.1.0-fspublic-replica"

func main() {
	var (
		configPath = flag.String("config", "GamePublic.ini", "GamePublic.ini 配置路径（不存在时用官方缺省）")
		districtID = flag.Uint("district", 0, "游戏分区编号（覆盖 [Main] DistrictID）")
		serverID   = flag.Uint("server", 0, "服务器编号（覆盖 [Main] ServerID）")
		innerAddr  = flag.String("inner-addr", "", "内网监听地址（覆盖 [Inner] Addr）")
		innerPort  = flag.Int("inner-port", 0, "内网监听端口（覆盖 [Inner] Port，缺省 16003）")
		echoPort   = flag.Int("echo-port", 0, "维护面端口（覆盖 [Echo] Port，缺省 15003）")
		worldAddr  = flag.String("world", "", "world 内网地址（空=不外连；官方由 UDP 广播发现补连）")
		storeAddrs = flag.String("stores", "", "store 内网地址 CSV（空=不外连）")
		noLogic    = flag.Bool("no-logic", false, "不挂载内置逻辑模块（FsPubLogic 等价）")
		tps        = flag.Int("tps", 100, "DoWait 每秒帧数")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go public %s\n", version)

	var cfg fspublic.Config
	if data, err := os.ReadFile(*configPath); err == nil {
		cfg, err = fspublic.ParseGamePublicINI(data)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("已加载配置：%s（District=%d Server=%d Logic=%s）\n",
			*configPath, cfg.DistrictID, cfg.ServerID, cfg.PublicLogicDll)
	} else {
		cfg = fspublic.DefaultConfig()
		fmt.Println("未找到配置文件，采用官方缺省（GamePublic.ini 方言）")
	}
	if *districtID != 0 {
		cfg.DistrictID = uint32(*districtID)
	}
	if *serverID != 0 {
		cfg.ServerID = uint32(*serverID)
	}
	if *innerAddr != "" {
		cfg.InnerAddr = *innerAddr
	}
	if *innerPort != 0 {
		cfg.InnerPort = *innerPort
	}
	if *echoPort != 0 {
		cfg.EchoPort = *echoPort
	}

	var callee fspublic.Callee
	if !*noLogic {
		callee = fspublic.NewDefaultCallee()
		fmt.Printf("load logic module %s (builtin callee, version %#x)\n",
			cfg.PublicLogicDll, fspublic.LogicModuleVersion)
	}

	var stores []string
	if *storeAddrs != "" {
		for _, s := range strings.Split(*storeAddrs, ",") {
			if s = strings.TrimSpace(s); s != "" {
				stores = append(stores, s)
			}
		}
	}

	asm, err := fspublic.Assemble(nil, cfg, callee, fspublic.AssembleOptions{
		WorldAddr:  *worldAddr,
		StoreAddrs: stores,
		TPS:        *tps,
	})
	if err != nil {
		fatal(err)
	}

	if err := asm.Start(); err != nil {
		fatal(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	asm.Stop() // 官方 close() 停机序
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "public:", err)
	os.Exit(1)
}
