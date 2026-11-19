package fsunite

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/entry"
	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloauction"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloentry"
	"github.com/lizhanyin/9yin-server-go/internal/fssololog"
	"github.com/lizhanyin/9yin-server-go/internal/fssolomember"
	"github.com/lizhanyin/9yin-server-go/internal/fssolopublic"
	"github.com/lizhanyin/9yin-server-go/internal/fssolostore"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloworld"
	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

// Options 合一进程装配参数（官方由 main.lua 硬编码 + ini 注入）。
type Options struct {
	// License 装载好的授权（nil=未授权形态：entry 不广播 + 每拍告警）。
	License *fssoloentry.License
	// NetOptions 传输面（MessageEncodeKey hex→u32 的 XOR；零值=明文）。
	NetOptions fsnet.Options
	// StoreBackend DB 后端（nil = fssolostore.NewMemoryStore("")）。
	StoreBackend fssolostore.Backend
	// Scenes scenes.ini 装配面（world；可空）。
	Scenes []fssoloworld.SceneSpec
	// Quit force_close 完成后的退出钩（官方 nx_quit；缺省 os.Exit(0)）。
	Quit func()
	// WorkDir logger LogFileDir 相对锚定（官方 nx_work_path()；缺省当前目录）。
	WorkDir string
	// Logf 官方 WriteLine 面的呈现（缺省丢弃）。
	Logf func(format string, args ...any)
}

// Unite 是 game_unite.exe 的七实体编排载体（solounite/main.lua 等价）。
type Unite struct {
	cfg  Config
	opt  Options
	logf func(format string, args ...any)

	logger  *fssololog.Server
	world   *fssoloworld.Server
	store   *fssolostore.Server
	public  *fssolopublic.Server
	auction *fssoloauction.Server
	member  *fssolomember.Server

	// entry 面五件套（cmd/soloentry 同构；实体/echo 在 Start 拿到
	// 动态内网口后总装）
	clients   *entry.Clients
	inner     *entry.Inner
	gateway   *entry.Gateway
	broad     *fscomm.BroadSock
	entryEcho *fscomm.EchoSock
	entrySvc  *fssoloentry.Server

	stopOnce sync.Once // 停机序幂等（Quit 钩/信号/宿主可并发重入）
	quitOnce sync.Once
}

// Assemble 建七实体骨架（不绑定网络；main.lua 14-407 的建实体 +
// 属性赋值 + 逻辑模块装载日志）。Start 按官方序启动。
func Assemble(cfg Config, opt Options) *Unite {
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	if opt.Quit == nil {
		opt.Quit = func() { os.Exit(0) }
	}
	u := &Unite{cfg: cfg, opt: opt, logf: opt.Logf}

	// [Logic]/PublicLogicDll/AuctionLogicDll 装载（呈现层；官方拼写
	// "loigc" 保留——fsgame/SnailEditor 的玩法层与 fspublogic/
	// fsauctionlogic 的数据逻辑层由各 fssolo* 内建承载）。
	for _, dll := range cfg.LogicModules {
		u.logf("load loigc module %s", dll)
	}
	if cfg.PublicLogicDll != "" {
		u.logf("load loigc module %s", cfg.PublicLogicDll)
	}
	if cfg.AuctionLogicDll != "" {
		u.logf("load loigc module %s", cfg.AuctionLogicDll)
	}

	// logger（main.lua 15/19/35-39）
	loggerOpt := cfg.LoggerOptions()
	loggerOpt.WorkDir = opt.WorkDir
	loggerOpt.Logf = u.logf
	loggerOpt.Printf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	u.logger = fssololog.New(loggerOpt)

	// world（main.lua 256-281；AppName "unite_<sid>"）
	u.world = fssoloworld.New(fssoloworld.Options{
		Config:   cfg.WorldConfig(),
		AppName:  cfg.AppName(),
		Scenes:   append([]fssoloworld.SceneSpec(nil), opt.Scenes...),
		OnClosed: func() { go u.quit() }, // 官方 nx_quit（异步防重入 tick 协程）
	})

	// store（main.lua 300-327；29 张日志表 = NewLogTableSet 缺省）
	backend := opt.StoreBackend
	if backend == nil {
		backend = fssolostore.NewMemoryStore("")
	}
	u.store = fssolostore.New(fssolostore.Options{
		Config:  cfg.StoreConfig(),
		Backend: backend,
	})

	// entry 传输面（main.lua 340-366 + cmd/soloentry 同构）
	u.clients = entry.NewClients()
	u.inner = entry.NewInner(cfg.InnerAddr, 0, fssoloentry.SoloOwnIdent(0), entry.InnerHandler{
		OnMessage: func(src, typ uint32, p []byte) {
			if u.entrySvc != nil {
				if err := u.entrySvc.ProcessInnerMsg(src, typ, p); err != nil {
					u.logf("unite entry: inner %#x: %v", typ, err)
				}
			}
		},
		OnRegister: func(ident uint32) { u.logf("unite entry: ident %#x 注册", ident) },
		OnDrop:     func(ident uint32) { u.logf("unite entry: ident %#x 断开", ident) },
	})
	u.inner.SourceCheck = fssoloentry.SoloSourceCheck
	u.inner.KickDuplicate = true
	u.inner.MemberBase = fssoloentry.SoloMemberBase
	u.inner.MemberCount = fssoloentry.SoloMemberCount
	u.inner.OnQuitFrame = func() {
		u.logf("unite entry: 收到内网退出帧（官方 Core->Quit(1)）")
		go u.quit()
	}
	u.gateway = entry.NewGateway(net.JoinHostPort(cfg.OuterAddr, strconv.Itoa(cfg.OuterPort)),
		0, u.clients, u.inner, opt.NetOptions)
	u.gateway.Logf = u.logf
	u.broad = fscomm.NewBroadSock(fscomm.BroadSockOptions{
		BindAddress:      cfg.InnerAddr,
		BindPort:         cfg.BroadPort, // 收发同口：共绑共享广播口
		BroadcastAddress: cfg.BroadAddr,
		BroadcastPort:    cfg.BroadPort,
	}, fscomm.BroadSockHandler{})

	// public（main.lua 283-298；domain/guild 双空间 Start 内置）
	u.public = fssolopublic.New(cfg.PublicConfig(), fssolopublic.Options{
		Logf: u.logf,
	})

	// auction（main.lua 329-337）
	auctionOpt := cfg.AuctionOptions()
	auctionOpt.Logf = u.logf
	u.auction = fssoloauction.New(auctionOpt)

	// member（main.lua 368-391）
	u.member = fssolomember.New(fssolomember.Options{
		Config: cfg.MemberConfig(),
		Logf:   u.logf,
		OnExit: func() { go u.quit() },
	})
	return u
}

// Start 官方启动序（main.lua 408-421）：
// logger → world → store → entry → public → auction → member。
// entry 步内联 cmd/soloentry 总装（动态内网口回填后建实体与 echo）。
func (u *Unite) Start() error {
	if err := u.logger.Start(); err != nil {
		return fmt.Errorf("unite: logger start: %w", err)
	}
	if err := u.world.Start(); err != nil {
		return fmt.Errorf("unite: world start: %w", err)
	}
	if err := u.store.Start(); err != nil {
		return fmt.Errorf("unite: store start: %w", err)
	}
	// entry 面：内网受面（0=动态分配回填）→ 实体 → 外网网关 → 广播口
	if err := u.inner.Start(); err != nil {
		return fmt.Errorf("unite: entry inner start: %w", err)
	}
	innerPort := 0
	if addr, ok := u.inner.Addr().(*net.TCPAddr); ok {
		innerPort = addr.Port
	}
	u.entrySvc = fssoloentry.New(fssoloentry.Options{
		IDC:         0, // main.lua entry.IDC = 0（unite 显式）
		DistrictID:  uint32(u.cfg.DistrictID),
		ServerID:    uint32(u.cfg.ServerID),
		EntryID:     0,
		MaxConnects: 1024, // main.lua 显式
		InnerAddr:   u.cfg.InnerAddr,
		InnerPort:   innerPort, // 官方 GetListenerPort 回填
		OuterAddr:   u.cfg.OuterAddr,
		AppName:     fmt.Sprintf("entry_%d_0", u.cfg.ServerID),
		License:     u.opt.License,
		StatusSink:  func(line string) { u.logf("%s", line) },
		BroadcastSink: func(pkt []byte) {
			if u.cfg.BroadPort != 0 {
				_ = u.broad.SendBroad(pkt)
			}
		},
		Quit: func() { go u.quit() },
	}, u.clients, u.inner, u.gateway)
	if _, err := u.gateway.Listen(); err != nil {
		return fmt.Errorf("unite: entry gateway listen: %w", err)
	}
	if u.cfg.BroadPort != 0 {
		if err := u.broad.Start(); err != nil {
			return fmt.Errorf("unite: entry broadcast start: %w", err)
		}
	}
	u.entrySvc.Start()

	// entry echo 15200（实体 ctor 缺省；main.lua 不设置；测试可覆盖）
	entryEchoPort := 15200
	if u.cfg.testEntryEcho != 0 {
		entryEchoPort = u.cfg.testEntryEcho
	}
	echoWelcome, echoOnInput := fssoloentry.EchoHandler(uint32(u.cfg.ServerID), 0,
		u.gateway.BlockIP(), u.entrySvc.StatusLine, func() { go u.quit() })
	u.entryEcho = fscomm.NewEchoSock("127.0.0.1", entryEchoPort, fscomm.EchoSockHandler{
		OnOpen: func(addr string, port int) {
			u.logf("echo client %s:%d connected", addr, port)
			_ = u.entryEcho.SendLine(echoWelcome)
		},
		OnClose: func(addr string, port int) {
			u.logf("echo client %s:%d closed", addr, port)
		},
		OnInput: func(line string) {
			for _, resp := range echoOnInput(line) {
				_ = u.entryEcho.SendLine(resp)
			}
		},
	})
	if err := u.entryEcho.Start(); err != nil {
		return fmt.Errorf("unite: entry echo start: %w", err)
	}

	if err := u.public.Start(); err != nil {
		return fmt.Errorf("unite: public start: %w", err)
	}
	if err := u.auction.Start(); err != nil {
		return fmt.Errorf("unite: auction start: %w", err)
	}
	if err := u.member.Start(); err != nil {
		return fmt.Errorf("unite: member start: %w", err)
	}
	return nil
}

// Banner 官方启动横幅（main.lua 423-455 WriteLine 序）。
func (u *Unite) Banner() []string {
	c := u.cfg
	lines := []string{
		"app name " + c.AppName(),
		fmt.Sprintf("echo socket %s:%d", c.EchoAddr, c.EchoPort),
		fmt.Sprintf("broadcast socket %s:%d", c.BroadAddr, c.BroadPort),
		fmt.Sprintf("outer socket %s:%d", c.OuterAddr, c.OuterPort),
	}
	if c.ChargeInuse {
		lines = append(lines, fmt.Sprintf("charge socket %s:%d", c.ChargeAddr, c.ChargePort))
	}
	if c.GmccInuse {
		lines = append(lines, fmt.Sprintf("gmcc socket %s:%d", c.GmccAddr, c.GmccPort))
	}
	if c.ManageInuse {
		lines = append(lines, fmt.Sprintf("manage socket %s:%d", c.ManageAddr, c.ManagePort))
	}
	lines = append(lines, fmt.Sprintf("game server %d starting...", c.ServerID))
	return lines
}

// Stop 官方 force_close 停机序（main.lua 917-958；幂等）：
// store → auction → public → entry → member → world → logger。
func (u *Unite) Stop() {
	u.stopOnce.Do(u.stopSequence)
}

func (u *Unite) stopSequence() {
	u.logf("store server shutdown...")
	u.store.Stop()
	u.logf("auction server shutdown...")
	u.auction.Stop()
	u.logf("public server shutdown...")
	u.public.Stop()
	u.logf("entry server shutdown...")
	if u.gateway != nil {
		u.gateway.Stop()
	}
	if u.entrySvc != nil {
		u.entrySvc.Stop()
	}
	u.broad.Stop()
	u.inner.Stop()
	if u.entryEcho != nil {
		u.entryEcho.Stop()
	}
	u.logf("member server shutdown...")
	u.member.Stop()
	u.logf("world server shutdown...")
	u.world.Stop()
	u.logger.Stop()
}

// quit 官方 nx_quit 等价：force_close 一轮后退出（幂等）。
func (u *Unite) quit() {
	u.quitOnce.Do(func() {
		u.Stop()
		u.opt.Quit()
	})
}

// 访问面（测试/横幅）

// World 提供世界服实体（场景/玩家注册表等观测）。
func (u *Unite) World() *fssoloworld.Server { return u.world }

// Store 提供存储服实体。
func (u *Unite) Store() *fssolostore.Server { return u.store }

// Entry 提供接入服实体（Start 前为 nil）。
func (u *Unite) Entry() *fssoloentry.Server { return u.entrySvc }
