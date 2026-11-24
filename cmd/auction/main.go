// auction 是拍卖行数据服务器进程，对应原版 game_auction.exe +
// FsSoloAuction.dll（小写系 solo 体系，BGserverCJ\Debug）。
//
// 官方形态：game_auction.exe 是通用 solo 宿主壳（与 game_lister.exe/
// game_entry.exe 同一二进制，按自身文件名读 game_auction.ini 装载
// fssoloauction.dll），FsSoloAuction.dll 承载三通道
// （inner :16003 / echo :16001 / broadcast :15000）+ 状态机 + store
// 持久化协议 + world 注册表，再经 LogicLoader 装载 fsauctionlogic.dll
// 业务。本进程 = 上述全链的 Go 等价（internal/fssoloauction +
// internal/auction），业务逻辑不经 DLL 装载而是进程内装配。
//
// 权威文档：docs/official/FsSoloAuction全解与Go复刻.md
// （业务面见 docs/official/fsauctionlogic-拍卖行数据服务器.md）。
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/auction"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloauction"
	"github.com/lizhanyin/9yin-server-go/internal/logline"
)

const version = "0.1.0-auction"

func main() {
	var (
		configPath = flag.String("config", "", "game_auction.ini 配置路径（默认：当前目录 game_auction.ini，没有则释放内置默认并采用；官方方言）")
		inner      = flag.String("inner", "", "内网受面 addr:port（官方 [Inner]；缺省 16003，与 main.lua 同源）")
		echo       = flag.String("echo", "127.0.0.1:16001", "echo 调试口 addr:port（官方 16001；off=关闭）")
		broadcast  = flag.String("broadcast", "127.255.255.255:15000", "UDP 服务发现广播 addr:port（官方 15000；off=关闭）")
		storeAddr  = flag.String("store", "", "直连 store addr:port（Go 扩展：不经广播发现直接外连 store 0）")
		localStore = flag.String("local-store", "data/auction", "本地文件 store 垫片目录（Go 扩展：无外部 store 时的 PUB4 落盘与加载兜底；off=关闭，纯官方语义等待真 store）")
		saveMin    = flag.Int("save-minutes", 0, "定时退存间隔分钟（0=取配置/缺省 20，官方 main.lua 同值）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C）")
	)
	flag.Parse()
	logline.Install("auction")
	defer logline.Flush()

	fmt.Printf("9yin-server-go auction %s（game_auction.exe + FsSoloAuction.dll 复刻）\n", version)

	// 配置装载：显式 -config 须存在；默认路径当前目录 game_auction.ini，
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
		cfg, data, created, err := auction.LoadGameAuctionINI("game_auction.ini")
		_ = cfg
		if err != nil {
			fatal(err)
		}
		cfgData, released = data, created
	}
	cfg, err := auction.ParseGameAuctionINI(cfgData)
	if err != nil {
		fatal(err)
	}
	if released {
		fmt.Println("当前目录未找到 game_auction.ini，已释放内置默认配置（修改后重启生效）")
	}

	given := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { given[f.Name] = true })

	opt := fssoloauction.Options{
		AppName:       cfg.MainEntity,
		DistrictID:    uint32(cfg.DistrictID),
		ServerID:      uint32(cfg.ServerID),
		InnerAddr:     cfg.InnerAddr,
		InnerPort:     cfg.InnerPort,
		SaveMinutes:   cfg.SaveMinutes,
		ItemTypeXML:   cfg.ItemTypeXML(),
		ExitForceSave: true,
		Logf:          func(format string, args ...any) { fmt.Printf("[auction] "+format+"\n", args...) },
	}

	// inner：旗标 > 配置 > 缺省 16003（cfg.InnerPort 已带缺省）
	if given["inner"] {
		host, port, err := splitAddr(*inner, opt.InnerPort)
		if err != nil {
			fatal(err)
		}
		opt.InnerAddr, opt.InnerPort = host, port
	} else if cfg.InnerAddr != "" {
		opt.InnerAddr = cfg.InnerAddr
	}

	// echo：显式旗标 > 配置 [Echo] > 旗标缺省 127.0.0.1:16001（官方 16001）
	echoVal := *echo
	if !given["echo"] && (cfg.EchoAddr != "" || cfg.EchoPort != 0) {
		echoVal = net.JoinHostPort(orDefault(cfg.EchoAddr, "127.0.0.1"),
			strconv.Itoa(orDefaultInt(cfg.EchoPort, 16001)))
	}
	if strings.EqualFold(echoVal, "off") {
		opt.EchoAddr, opt.EchoPort = "", 0
	} else if host, port, err := splitAddr(echoVal, 16001); err == nil {
		opt.EchoAddr, opt.EchoPort = host, port
	} else {
		fatal(err)
	}

	// broadcast：显式旗标 > 配置 [Broadcast] > 旗标缺省 …:15000（官方 15000）
	broadVal := *broadcast
	if !given["broadcast"] && (cfg.BroadcastAddr != "" || cfg.BroadcastPort != 0) {
		broadVal = net.JoinHostPort(orDefault(cfg.BroadcastAddr, "127.255.255.255"),
			strconv.Itoa(orDefaultInt(cfg.BroadcastPort, 15000)))
	}
	if strings.EqualFold(broadVal, "off") {
		opt.BroadPort = 0
	} else if host, port, err := splitAddr(broadVal, 15000); err == nil {
		opt.BroadAddr, opt.BroadPort = host, port
	} else {
		fatal(err)
	}

	if given["save-minutes"] {
		opt.SaveMinutes = *saveMin
	}
	if opt.SaveMinutes == 0 {
		opt.SaveMinutes = 20
	}

	if !strings.EqualFold(*localStore, "off") {
		opt.LocalStore = &fssoloauction.LocalStore{
			Dir:  *localStore,
			Logf: func(format string, args ...any) { fmt.Printf("[auction] "+format+"\n", args...) },
		}
	}

	fmt.Printf("已加载配置：DistrictID=%d ServerID=%d 逻辑=%s（Go 内置等价）SaveMinutes=%d\n",
		cfg.DistrictID, cfg.ServerID, orDefault(cfg.AuctionLogicDll, "fsauctionlogic.dll"), opt.SaveMinutes)

	srv := fssoloauction.New(opt)
	if err := srv.Start(); err != nil {
		fatal(err)
	}
	if *storeAddr != "" {
		if err := srv.ConnectStore(*storeAddr); err != nil { // Go 扩展：直连 store
			fatal(err)
		}
	}
	fmt.Printf("inner 就绪：%s:%d（member 0x18..0x2F 连入；identify id2=0xFFFF）\n",
		orDefault(opt.InnerAddr, "0.0.0.0"), opt.InnerPort)
	if opt.EchoPort > 0 {
		fmt.Printf("echo 就绪：%s:%d（help 查看命令；行协议 \\r/\\n 终止）\n",
			orDefault(opt.EchoAddr, "127.0.0.1"), opt.EchoPort)
	} else {
		fmt.Println("echo：未启用")
	}
	if opt.BroadPort > 0 {
		fmt.Printf("broadcast 就绪：%s:%d（announce type=5/magic=0x3001 每 1s；收 world/store 自宣外连）\n",
			opt.BroadAddr, opt.BroadPort)
	} else {
		fmt.Println("broadcast：未启用")
	}
	if opt.LocalStore != nil {
		fmt.Printf("local store：%s（PUB4 落盘；无外部 store 时兜底 0x40→0x29/0x2A/0x2B 加载序）\n", opt.LocalStore.Dir)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if *runFor > 0 {
		ctx, cancel = context.WithTimeout(ctx, *runFor)
		defer cancel()
	}
	start := time.Now()
	srv.Run(ctx)
	fmt.Printf("auction：已停止（运行 %s）\n", time.Since(start).Round(time.Millisecond))
}

// splitAddr 解析 "host:port"/":port"/"port"；缺 host 取缺省端口语义由
// 调用方决定，此处 port 为 0 时用 defPort。
func splitAddr(v string, defPort int) (string, int, error) {
	v = strings.TrimSpace(v)
	if p, err := strconv.Atoi(v); err == nil {
		return "", p, nil
	}
	host, portS, err := net.SplitHostPort(v)
	if err != nil {
		return "", 0, fmt.Errorf("-addr %q 解析失败: %w", v, err)
	}
	port, err := strconv.Atoi(portS)
	if err != nil {
		return "", 0, fmt.Errorf("端口须为整数: %q", portS)
	}
	if port == 0 {
		port = defPort
	}
	return host, port, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func orDefaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "auction:", err)
	logline.Exit(1)
}
