package fspublic

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
)

// WAPS 进程状态（main.lua 同枚举；AppManager 面）。
const (
	WAPSStarting = 1
	WAPSReady    = 2
	WAPSOpening  = 3
	WAPSOpened   = 4
	WAPSClosing  = 5
	WAPSClosed   = 6
	WAPSQuitting = 7
)

// AssembleOptions 角色装配可选项。
type AssembleOptions struct {
	// Logf 日志面。
	Logf func(format string, args ...any)
	// AppBookDir AppManager 注册表目录（空="run"）。
	AppBookDir string
	// WorldAddr/StoreAddrs 集群外连目标（空=不外连；官方由 UDP 广播
	// 发现后幂等补连——两者可并存）。
	WorldAddr  string
	StoreAddrs []string
	// TPS DoWait 每秒帧数（0=100；官方 WorkerProc Sleep=1ms）。
	TPS int
	// ExtraEchoCommands 追加维护命令（state/end/quit 已内建）。
	ExtraEchoCommands map[string]fscomm.EchoLineHandler
}

// Assembly 是一次角色装配的产物：四件套面 + DoWait 泵 + 服务本体，
// 启停序对齐 main.lua。
type Assembly struct {
	Public *Public
	Config Config
	Inner  *Inner
	Outer  *Outer
	Echo   *fscomm.EchoSock
	Broad  *fscomm.BroadSock
	App    *fscomm.AppManager

	logf       func(format string, args ...any)
	tps        int
	stopOnce   sync.Once
	stop       chan struct{}
	wg         sync.WaitGroup
}

// Assemble 复刻 main.lua 的 GamePublic 装配段（不启动；Start 按官方序
// 拉起）。p 为 nil 时按 cfg 新建；callee 为 nil 时挂 DefaultCallee。
func Assemble(p *Public, cfg Config, callee Callee, opt AssembleOptions) (*Assembly, error) {
	if opt.Logf == nil {
		opt.Logf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	}
	if opt.TPS <= 0 {
		opt.TPS = 100
	}
	if opt.AppBookDir == "" {
		opt.AppBookDir = "run"
	}
	if p == nil {
		p = New(cfg.DistrictID, cfg.ServerID)
	}
	if callee == nil {
		callee = NewDefaultCallee()
		if dc, ok := callee.(*DefaultCallee); ok {
			dc.OnLog = opt.Logf
		}
	}
	p.Logf = opt.Logf
	p.Callee = callee
	p.SetInner(cfg.InnerAddr, cfg.InnerPort)

	// main.lua 空间注册（官方字面；已存在不重复建——测试可预置）
	if _, ok := p.Spaces.Find("pubdata"); !ok {
		_ = p.Spaces.Add("pubdata", "nx_pubdata")
	}
	if _, ok := p.Spaces.Find("guild"); !ok {
		_ = p.Spaces.Add("guild", "nx_guilds")
	}

	a := &Assembly{
		Public: p, Config: cfg,
		logf: opt.Logf, tps: opt.TPS,
		stop: make(chan struct{}),
	}

	// 内网受面（Adapter + ReceiveMsg 路由：world/store/member 三队列）
	a.Inner = NewInner(cfg.InnerAddr, cfg.InnerPort, InnerHandler{
		OnRegister: func(ident uint32) {
			opt.Logf("public adapter register ident %#x", ident)
			switch ident {
			case StoreIdentBase: // store#0
				p.OnStoreConnected()
			case WorldIdent: // world（官方 InnerWorldConnected 空操作）
				p.OnWorldConnected()
			}
		},
		OnDrop: func(ident uint32) {
			opt.Logf("public adapter disconnect ident %#x", ident)
		},
		OnError: func(format string, args ...any) { opt.Logf(format, args...) },
	}, func(src, ident, typ uint32, payload []byte) {
		switch {
		case src == WorldIdent:
			p.PutWorldMessage(src, typ, payload)
		case src >= StoreIdentBase && src < StoreIdentBase+4:
			p.PutStoreMessage(src, typ, payload)
		default: // member 0x18..0x27
			p.PutMemberMessage(src, typ, payload)
		}
	})

	// 维护面（echo.lua：welcome "Welcome public server S"）
	commands := map[string]fscomm.EchoLineHandler{
		"quit": func(args []string) {
			_ = a.Echo.SendLine(">server will shutdown...\r")
			go a.Stop()
		},
		"end": func(args []string) { a.Echo.Stop() },
		"state": func(args []string) {
			_ = a.Echo.SendLine(">" + p.String() + "\r")
		},
	}
	for k, v := range opt.ExtraEchoCommands {
		commands[k] = v
	}
	a.Echo = fscomm.NewEchoSock(cfg.EchoAddr, cfg.EchoPort, fscomm.EchoSockHandler{
		OnOpen: func(addr string, port int) {
			opt.Logf("echo client %s:%d connected", addr, port)
			_ = a.Echo.SendLine(fmt.Sprintf("Welcome public server %d\r", cfg.ServerID))
		},
		OnClose: func(addr string, port int) {
			opt.Logf("echo client %s:%d closed", addr, port)
		},
		OnInput: func(line string) {
			_ = a.Echo.SendLine(">Undefined command: " + line + "\r")
		},
		OnCommand: commands,
	})

	// 广播面（public 信息包 kind=4/magic=0x300；收听 world kind=0 /
	// store kind=2 做幂等补连——CBroadSockHook::OnReceive）
	a.Broad = fscomm.NewBroadSock(fscomm.BroadSockOptions{
		BindAddress:      cfg.InnerAddr,
		BroadcastAddress: cfg.BroadAddr,
		BroadcastPort:    cfg.BroadPort,
	}, fscomm.BroadSockHandler{
		OnReceive: a.onBroadcast,
	})

	// 进程注册（AppManager "public_S"）
	app, err := fscomm.NewAppManager(AppName(cfg.ServerID), opt.AppBookDir)
	if err != nil {
		return nil, err
	}
	a.App = app

	// 外发面（CInterConnect：SendToWorld/SendToStorage/SendToAll\*）
	a.Outer = NewOuter(p.OwnIdent())
	if opt.WorldAddr != "" {
		a.Outer.AddLink(WorldIdent, opt.WorldAddr, func(ident uint32, up bool) {
			opt.Logf("world link %#x %v", ident, up)
		})
	}
	for i, addr := range opt.StoreAddrs {
		if i >= 4 {
			break
		}
		ident := StoreIdentBase + uint32(i)
		a.Outer.AddLink(ident, addr, func(id uint32, up bool) {
			opt.Logf("store link %#x %v", id, up)
		})
	}
	// 发面路由：入站注册优先（world/store 连 public 的官方形态），
	// 外连兜底（UDP 发现补连路径）——官方 Adapter ident 表收发同源。
	sendTo := func(ident, typ uint32, payload []byte) error {
		if a.Inner.Connected(ident) {
			return a.Inner.SendRaw(ident, typ, payload)
		}
		return a.Outer.Send(ident, typ, payload)
	}
	p.SendToWorld = func(typ uint32, payload []byte) error { return sendTo(WorldIdent, typ, payload) }
	p.SendToStorage = func(typ uint32, payload []byte) error { return sendTo(StoreIdentBase, typ, payload) }
	p.SendToAllStore = func(typ uint32, payload []byte) error {
		var firstErr error
		for i := 0; i < 4; i++ {
			if err := sendTo(StoreIdentBase+uint32(i), typ, payload); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}
	p.SendToMember = func(ident, typ uint32, payload []byte) error { return sendTo(ident, typ, payload) }
	p.SendToAllMem = func(typ uint32, payload []byte) error {
		var firstErr error
		for _, id := range a.Inner.MemberIdents() {
			if err := a.Inner.SendRaw(id, typ, payload); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}
	return a, nil
}

// AppName 官方进程名 "public_<S>"。
func AppName(serverID uint32) string {
	return fmt.Sprintf("public_%d", serverID)
}

// onBroadcast 复刻 CBroadSockHook::OnReceive：magic 0x300；kind 0=world
//（0x78B）/kind 2=store（0x28B）；DistrictId/ServerId 匹配本服才补连。
func (a *Assembly) onBroadcast(data []byte, fromAddr string, fromPort int) {
	if len(data) < 0x24 || binary.LittleEndian.Uint32(data[4:]) != 0x300 {
		return
	}
	kind := binary.LittleEndian.Uint32(data[0:])
	switch kind {
	case 0: // world
		if len(data) < 0x78 {
			return
		}
	case 2: // store
		if len(data) < 0x28 {
			return
		}
	default:
		return
	}
	district := binary.LittleEndian.Uint32(data[0x1C:])
	server := binary.LittleEndian.Uint32(data[0x20:])
	if district != a.Config.DistrictID || server != a.Config.ServerID {
		return
	}
	host, _, err := net.SplitHostPort(fromAddr)
	if err != nil {
		return
	}
	innerPort := int(binary.LittleEndian.Uint32(data[0x18:]))
	addr := fmt.Sprintf("%s:%d", host, innerPort)
	var ident uint32
	if kind == 0 {
		ident = WorldIdent
	} else {
		ident = StoreIdentBase
	}
	if !a.Outer.Connected(ident) {
		a.logf("broadcast discover kind=%d %s -> link %#x", kind, addr, ident)
		a.Outer.AddLink(ident, addr, func(id uint32, up bool) {
			a.logf("link %#x %v", id, up)
		})
		// AddLink 建好对象后需启动（发现路径动态加入）
		a.Outer.Start()
	}
}

// Start 复刻 main.lua 启动序：AppManager(WAPS_STARTING) → Adapter →
// BroadSock → EchoSock → GamePublic:Start → WAPS_OPENED → DoWait 泵 +
// 周期广播 + console 五连。
func (a *Assembly) Start() error {
	p, cfg := a.Public, a.Config
	if err := a.App.Register(AppName(cfg.ServerID),
		fmt.Sprintf("%s:%d", cfg.EchoAddr, cfg.EchoPort), WAPSStarting); err != nil {
		return err
	}
	a.App.StartSweeper()
	if err := a.Inner.Start(); err != nil {
		return err
	}
	if err := a.Broad.Start(); err != nil {
		return err
	}
	if err := a.Echo.Start(); err != nil {
		return err
	}
	a.Outer.Start()
	_ = a.App.SetAppState(WAPSOpened)

	a.wg.Add(2)
	go func() { // DoWait 泵（官方 WorkerProc：DoWait+Sleep(1ms)）
		defer a.wg.Done()
		interval := time.Second / time.Duration(a.tps)
		if interval <= 0 {
			interval = time.Millisecond
		}
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-a.stop:
				return
			case <-tick.C:
				p.DoWait()
			}
		}
	}()
	go func() { // BroadcastServerInfo 周期包（官方 DoWait 秒分支）
		defer a.wg.Done()
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-a.stop:
				return
			case <-tick.C:
				_ = a.Broad.SendBroad(p.BuildBroadcast(cfg.InnerAddr, cfg.InnerPort))
			}
		}
	}()

	a.logf("app name %s", AppName(cfg.ServerID))
	a.logf("echo socket %s:%d", cfg.EchoAddr, cfg.EchoPort)
	a.logf("inner socket %s:%d", cfg.InnerAddr, cfg.InnerPort)
	a.logf("broadcast socket %s:%d", cfg.BroadAddr, cfg.BroadPort)
	a.logf("game public server %d started", cfg.ServerID)
	return nil
}

// Stop 复刻 main.lua close() 停机序：广播→内网→维护→服务→监控。
func (a *Assembly) Stop() {
	a.stopOnce.Do(func() {
		close(a.stop)
		a.logf("broad shutdown...")
		a.Broad.Stop()
		a.logf("adapter shutdown...")
		a.Inner.Stop()
		a.Outer.Stop()
		a.logf("echo socket shutdown...")
		a.Echo.Stop()
		a.logf("public server shutdown...")
		_ = a.App.SetAppState(WAPSClosed)
		a.App.Close()
		a.wg.Wait()
	})
}
