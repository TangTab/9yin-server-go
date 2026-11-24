// soloentry 是单机形态接入网关进程，对应原版 game_entry.exe +
// FsSoloEntry.dll（小写系 solo 体系，BGserverCJ\Debug）。
//
// 官方形态：game_entry.exe 是通用 solo 宿主壳（与 game_lister.exe/
// game_auction.exe 同一二进制，按自身文件名读 game_entry.ini 装载
// fssoloentry.dll + lua\soloentry\main.lua），SoloEntry 实体承载
// 四通道：外网客户端 TCP（EE+XOR 纯代理）/ 内网 Adapter（world/
// member 反向连入，InnerPort 缺省 0 动态分配）/ echo 维护面 /
// UDP 服务发现广播（0x3001 信息包，license 门），加四态状态机与
// 15 型内网分派。本进程 = internal/fssoloentry + internal/entry 复用
// 组件的 Go 等价装配。
//
// 权威文档：docs/official/FsSoloEntry全解与Go复刻.md
// （网关版孪生见 docs/official/FsEntry全解与Go复刻.md）。
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"

	gonet "net"

	"github.com/lizhanyin/9yin-server-go/internal/entry"
	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloentry"
	"github.com/lizhanyin/9yin-server-go/internal/logline"
	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

const version = "0.1.0-soloentry"

// 默认 game_entry.ini（官方快照方言，solo 装载面原文——fssoloentry.dll）
const defaultINI = `; 单机接入服务器配置
[System]
Sleep=-1
MainEntity=SoloEntry
MainScript=main

[Module]
fskit.dll
fscomm.dll
fssoloentry.dll

[Script]
Path=..\lua\soloentry
Reload=1

[Main]
IDC=1
DistrictID=1
ServerID=11
EntryID=0
MaxConnects=1024
;MessageEncodeKey=A5375CAE
;MessageVerifyDll=fsverify.dll

[Outer]
Addr=127.0.0.1
Port=2001

[Inner]
;Addr=127.0.0.1
;Port=0

[Broadcast]
Addr=127.255.255.255
Port=15000

[Echo]
Addr=127.0.0.1
Port=15200
`

func main() {
	var (
		configPath = flag.String("config", "", "game_entry.ini 路径（默认：当前目录 game_entry.ini，没有则释放内置默认；官方方言）")
		listen     = flag.String("listen", "", "外网客户端监听 addr:port（空=取配置 [Outer]）")
		innerPort  = flag.Int("inner-port", -1, "内网监听端口（-1=取配置；官方缺省 0=动态分配）")
		echoPort   = flag.Int("echo-port", 0, "维护面端口（0=取配置缺省 15200+EntryID）")
		noLicense  = flag.Bool("no-license", false, "跳过 license 门（仅联调用）")
		genLic     = flag.String("gen-license", "", "为本机 IP 生成 solo lic 后退出（值=IP；内部形态 256B）")
		runFor     = flag.Duration("run", 0, "运行时长（0=直到 Ctrl+C；冒烟用）")
	)
	flag.Parse()
	logline.Install("entry")
	defer logline.Flush()

	fmt.Printf("9yin-server-go soloentry %s（game_entry.exe + FsSoloEntry.dll 复刻）\n", version)

	// 配置装载：显式 -config 须存在；默认路径缺失则释放内置默认（不覆盖）
	var cfgData []byte
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			fatal(err)
		}
		cfgData = data
	} else if data, err := os.ReadFile("game_entry.ini"); err == nil {
		cfgData = data
	} else {
		if err := os.WriteFile("game_entry.ini", []byte(defaultINI), 0644); err != nil {
			fatal(err)
		}
		cfgData = []byte(defaultINI)
		fmt.Println("当前目录未找到 game_entry.ini，已释放内置默认配置（修改后重启生效）")
	}
	cfg, err := fssoloentry.ParseSoloINI(cfgData)
	if err != nil {
		fatal(err)
	}
	if *listen != "" {
		host, port, _ := gonet.SplitHostPort(*listen)
		if host != "" {
			cfg.OuterAddr = host
		}
		if p, err := strconv.Atoi(port); err == nil {
			cfg.OuterPort = p
		}
	}
	if *innerPort >= 0 {
		cfg.InnerPort = *innerPort
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

	// license 门（官方 lookup_thread：0x7D0ms 等待，超时 program abort!）
	var lic *fssoloentry.License
	if !*noLicense {
		ip := fssoloentry.LocalIPString(cfg.OuterAddr)
		l, err := fssoloentry.LoadLicenseFile(cfg.License, ip, time.Now())
		if err != nil {
			fmt.Fprintf(os.Stderr, "soloentry: %v未授权（不广播 + 每拍告警）\n", err)
		} else {
			lic = &l
			if l.Licensed() {
				fmt.Printf("license 就绪：%s 魔数对 %#X/%#X\n", cfg.License, l.Fields[0], l.Fields[1])
			} else {
				fmt.Printf("license 解码成功但魔数对失配：%#X/%#X（按未授权处理）\n", l.Fields[0], l.Fields[1])
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

	clients := entry.NewClients()

	// 内网面：solo 语义（源校验 {0,1,2}∪[0x18,0x2F]、自身 ident=
	// EntryId+8、ident 重复保旧踢新、member 24 位、入站 0xFFFE 退出）
	quit := make(chan struct{})
	var svc *fssoloentry.Server
	inner := entry.NewInner(cfg.InnerAddr, cfg.InnerPort, fssoloentry.SoloOwnIdent(cfg.EntryID), entry.InnerHandler{
		OnMessage: func(src, typ uint32, p []byte) {
			if svc != nil {
				if err := svc.ProcessInnerMsg(src, typ, p); err != nil {
					fmt.Printf("soloentry: inner %#x: %v\n", typ, err)
				}
			}
		},
		OnRegister: func(ident uint32) { fmt.Printf("inner: ident %#x 注册\n", ident) },
		OnDrop:     func(ident uint32) { fmt.Printf("inner: ident %#x 断开\n", ident) },
	})
	inner.SourceCheck = fssoloentry.SoloSourceCheck
	inner.KickDuplicate = true
	inner.MemberBase = fssoloentry.SoloMemberBase
	inner.MemberCount = fssoloentry.SoloMemberCount
	inner.OnQuitFrame = func() {
		fmt.Println("soloentry: 收到内网退出帧（官方 Core->Quit(1)）")
		close(quit)
	}
	if err := inner.Start(); err != nil {
		fatal(err)
	}
	if cfg.InnerPort == 0 {
		// 官方：InnerPort=0 时以 GetListenerPort 回写实际端口
		if addr, ok := inner.Addr().(*net.TCPAddr); ok {
			cfg.InnerPort = addr.Port
		}
	}

	// 外网面（COuterHook：EE+XOR 客户端线 + 池 + 黑名单 + accept 上报）
	gateway := entry.NewGateway(gonet.JoinHostPort(cfg.OuterAddr, strconv.Itoa(cfg.OuterPort)),
		cfg.EntryID, clients, inner, netOptions)
	gateway.Logf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	if _, err := gateway.Listen(); err != nil {
		fatal(err)
	}

	// 广播面：UDP 0x3001 信息包（license 魔数对门内置于 Server.TimerCheck）
	broad := fscomm.NewBroadSock(fscomm.BroadSockOptions{
		BindAddress: cfg.InnerAddr, BroadcastAddress: cfg.BroadAddr,
	}, fscomm.BroadSockHandler{})
	if cfg.BroadPort != 0 {
		if err := broad.Start(); err != nil {
			fatal(err)
		}
		defer broad.Stop()
	}

	// 总装（ManualTick=false：1s 定时循环）
	svc = fssoloentry.New(fssoloentry.Options{
		IDC: cfg.IDC, DistrictID: cfg.DistrictID,
		ServerID: cfg.ServerID, EntryID: cfg.EntryID,
		MaxConnects: cfg.MaxConnects,
		InnerAddr:   cfg.InnerAddr, InnerPort: cfg.InnerPort,
		OuterAddr:  cfg.OuterAddr, OuterPort: cfg.OuterPort,
		AppName:    fmt.Sprintf("entry_%d_%d", cfg.ServerID, cfg.EntryID),
		License:    lic,
		StatusSink: func(line string) { fmt.Println(line) },
		BroadcastSink: func(pkt []byte) {
			if cfg.BroadPort != 0 {
				_ = broad.SendBroad(pkt)
			}
		},
		Quit: func() { close(quit) },
		Logf: func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
	}, clients, inner, gateway)
	svc.Start()

	// 维护面：行协议（welcome + quit/loadip/end/state）
	var echo *fscomm.EchoSock
	welcome, onInput := fssoloentry.EchoHandler(cfg.ServerID, uint32(cfg.EntryID),
		gateway.BlockIP(), svc.StatusLine, func() { shutdown(gateway, inner, echo, svc) })
	echo = fscomm.NewEchoSock(cfg.EchoAddr, cfg.EchoPort, fscomm.EchoSockHandler{
		OnOpen: func(addr string, port int) {
			fmt.Printf("echo client %s:%d connected\n", addr, port)
			_ = echo.SendLine(welcome)
		},
		OnClose: func(addr string, port int) {
			fmt.Printf("echo client %s:%d closed\n", addr, port)
		},
		OnInput: func(line string) {
			for _, resp := range onInput(line) {
				_ = echo.SendLine(resp)
			}
		},
	})
	if err := echo.Start(); err != nil {
		fatal(err)
	}

	// 官方启动横幅（main.lua WriteLine 序列）
	fmt.Printf("app name %s\n", fmt.Sprintf("entry_%d_%d", cfg.ServerID, cfg.EntryID))
	fmt.Printf("echo socket %s:%d\n", cfg.EchoAddr, cfg.EchoPort)
	fmt.Printf("inner socket %s:%d\n", cfg.InnerAddr, cfg.InnerPort)
	fmt.Printf("broadcast socket %s:%d\n", cfg.BroadAddr, cfg.BroadPort)
	fmt.Printf("outer socket %s:%d\n", cfg.OuterAddr, cfg.OuterPort)
	fmt.Printf("game server %d entry %d started\n", cfg.ServerID, cfg.EntryID)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	if *runFor > 0 {
		select {
		case <-time.After(*runFor):
		case <-sig:
		case <-quit:
		}
	} else {
		select {
		case <-sig:
		case <-quit:
		}
	}
	fmt.Println("entry server shutdown...")
	shutdown(gateway, inner, echo, svc)
}

func shutdown(gateway *entry.Gateway, inner *entry.Inner, echo *fscomm.EchoSock, svc *fssoloentry.Server) {
	gateway.Stop()
	if svc != nil {
		svc.Stop()
	}
	inner.Stop()
	if echo != nil {
		echo.Stop()
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "soloentry:", err)
	logline.Exit(1)
}
