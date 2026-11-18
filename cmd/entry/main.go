// entry 是接入网关进程（对应原版 game_entry.exe + FsEntry.dll）：
// 外网客户端 TCP（EE+XOR 纯代理）+ 内网 FsComm Adapter（world/member
// 反向连入注册 ident，entry_forw 16 型驱动）+ EchoSock 维护面 +
// BroadSock 0x3001 信息包广播 + license 门。
//
// 权威文档：docs/official/FsEntry全解与Go复刻.md。
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	gonet "net"

	"github.com/lizhanyin/9yin-server-go/internal/entry"
	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

const version = "0.1.0-fsentry-replica"

func main() {
	var (
		configPath = flag.String("config", "GameEntry.ini", "GameEntry.ini 配置路径（不存在时用旗标/缺省值）")
		listen     = flag.String("listen", "", "外网客户端监听（空=取配置 [Outer]）")
		innerPort  = flag.Int("inner-port", 0, "内网监听端口（0=取配置缺省 16200+EntryID）")
		echoPort   = flag.Int("echo-port", 0, "维护面端口（0=取配置缺省 15200+EntryID）")
		noLicense  = flag.Bool("no-license", false, "跳过 license 门（仅联调用）")
		genLic     = flag.String("gen-license", "", "为本机外网 IP 生成 lic 后退出（值=外网 IP）")
		logRecv    = flag.Bool("log-recv", false, "记录上行帧（LogReceive）")
	)
	flag.Parse()

	fmt.Printf("9yin-server-go entry %s\n", version)

	var cfg entry.Config
	if data, err := os.ReadFile(*configPath); err == nil {
		cfg, err = entry.ParseGameEntryINI(data)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("已加载配置：%s（IDC=%d District=%d Server=%d Entry=%d MaxPlayer=%d）\n",
			*configPath, cfg.IDC, cfg.DistrictID, cfg.ServerID, cfg.EntryID, cfg.MaxPlayer)
	} else {
		cfg = entry.DefaultConfig()
		fmt.Println("未找到配置文件，采用官方缺省（GameEntry.ini 方言）")
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
	if *innerPort != 0 {
		cfg.InnerPort = *innerPort
	}
	if *echoPort != 0 {
		cfg.EchoPort = *echoPort
	}
	if cfg.License == "" {
		cfg.License = cfg.OuterAddr + ".lic"
	}

	// license：生成器模式先行
	if *genLic != "" {
		lic := entry.GenerateLicense(*genLic, entry.OfficialMagic1, entry.OfficialMagic2)
		path := *genLic + ".lic"
		if err := os.WriteFile(path, lic, 0644); err != nil {
			fatal(err)
		}
		fmt.Printf("已生成 %s（3DES-EDE 自签，魔数对 %#X/%#X）\n", path, entry.OfficialMagic1, entry.OfficialMagic2)
		return
	}
	licenseOK := true
	if !*noLicense {
		id, err := entry.LoadLicenseFile(cfg.License, cfg.OuterAddr, entry.OfficialMagic1, entry.OfficialMagic2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "entry: %v\n", err)
			licenseOK = false
		} else {
			fmt.Printf("license 就绪：%s 魔数对 %#X/%#X\n", cfg.License, id.Magic1, id.Magic2)
		}
	}

	// 传输 XOR 键（官方 MessageEncodeKey hex→u32；空=内置现代客户端键）
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

	// 内网面：world=0、member=0x18+idx 反向连入注册
	var svc *entry.Entry
	inner := entry.NewInner(cfg.InnerAddr, cfg.InnerPort, 1<<16|uint32(cfg.EntryID), entry.InnerHandler{
		OnMessage: func(src, typ uint32, p []byte) {
			if svc != nil {
				if err := svc.ProcessForward(src, typ, p); err != nil {
					fmt.Printf("entry: forward %#x: %v\n", typ, err)
				}
			}
		},
		OnRegister: func(ident uint32) { fmt.Printf("inner: ident %#x 注册\n", ident) },
		OnDrop:     func(ident uint32) { fmt.Printf("inner: ident %#x 断开\n", ident) },
	})
	if err := inner.Start(); err != nil {
		fatal(err)
	}
	fmt.Printf("inner socket %s:%d\n", cfg.InnerAddr, cfg.InnerPort)

	// 外网面
	gateway := entry.NewGateway(gonet.JoinHostPort(cfg.OuterAddr, strconv.Itoa(cfg.OuterPort)),
		cfg.EntryID, clients, inner, netOptions)
	gateway.Logf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	if *logRecv {
		gateway.SetLogReceive(true)
	}
	addr, err := gateway.Listen()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("outer socket %s\n", addr.String())

	// 维护面：行协议（welcome/quit；OnCommand 具名分发与官方同优先级）
	var echo *fscomm.EchoSock
	echo = fscomm.NewEchoSock(cfg.EchoAddr, cfg.EchoPort, fscomm.EchoSockHandler{
		OnOpen: func(addr string, port int) {
			fmt.Printf("echo client %s:%d connected\n", addr, port)
			_ = echo.SendLine(entry.EchoWelcome(cfg.ServerID, uint32(cfg.EntryID)))
		},
		OnClose: func(addr string, port int) {
			fmt.Printf("echo client %s:%d closed\n", addr, port)
		},
		OnInput: func(line string) {
			_ = echo.SendLine(entry.EchoResponse("undefined command: " + entry.TrimEchoLine(line)))
		},
		OnCommand: map[string]fscomm.EchoLineHandler{
			"quit": func(args []string) {
				_ = echo.SendLine(entry.EchoResponse("server will shutdown..."))
				go func() {
					shutdown(gateway, inner, echo, svc)
					os.Exit(0)
				}()
			},
			"end": func(args []string) { echo.Stop() },
		},
	})
	if err := echo.Start(); err != nil {
		fatal(err)
	}
	fmt.Printf("echo socket %s:%d\n", cfg.EchoAddr, cfg.EchoPort)

	// 广播面：0x3001 信息包（3 秒一拍）
	broad := fscomm.NewBroadSock(fscomm.BroadSockOptions{
		BindAddress: cfg.InnerAddr, BroadcastAddress: cfg.BroadAddr,
	}, fscomm.BroadSockHandler{})
	_ = broad.Start()
	defer broad.Stop()

	// 总装
	svc = entry.New(entry.Options{
		EntryID: cfg.EntryID, ServerID: cfg.ServerID,
		DistrictID: cfg.DistrictID, IDC: cfg.IDC,
		LicenseInvalid: func() bool { return !licenseOK },
		Logf:           func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
	}, clients, inner, gateway)
	svc.Start()
	fmt.Printf("app name entry_%d_%d\n", cfg.ServerID, cfg.EntryID)
	fmt.Printf("game server %d entry %d started\n", cfg.ServerID, cfg.EntryID)

	go func() {
		tick := time.NewTicker(3 * time.Second)
		defer tick.Stop()
		for range tick.C {
			pkt := svc.BroadcastInfo(cfg.OuterAddr, fmt.Sprintf("entry_%d_%d", cfg.ServerID, cfg.EntryID),
				[4]uint32{cfg.IDC, cfg.DistrictID, cfg.ServerID, uint32(cfg.EntryID)}, 0)
			_ = broad.SendBroad(pkt)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	fmt.Println("entry server shutdown...")
	shutdown(gateway, inner, echo, svc)
}

func shutdown(gateway *entry.Gateway, inner *entry.Inner, echo *fscomm.EchoSock, svc *entry.Entry) {
	gateway.Stop()
	if svc != nil {
		svc.Stop()
	}
	inner.Stop()
	echo.Stop()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "entry:", err)
	os.Exit(1)
}
