package fsmember

import (
	"fmt"
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
	// Logf 日志面（nil=标准输出转 log 前缀）。
	Logf func(format string, args ...any)
	// AppBookDir AppManager 注册表目录（空="run"）。
	AppBookDir string
	// WorldAddr/PublicAddr 外连目标（空=不外连；官方由服务发现）。
	WorldAddr  string
	PublicAddr string
	// TPS DoWait 每秒帧数（0=100）。
	TPS int
	// BroadcastInterval 信息包周期（0=3s）。
	BroadcastInterval time.Duration
	// ExtraEchoCommands 追加维护命令（state/perf/end/quit 已内建）。
	ExtraEchoCommands map[string]fscomm.EchoLineHandler
	// OnQuit 停机序末回调（官方 close() 尾部的 nx_quit()——装载器路径
	// 传 core.Quit 使泵循环退出；独立 cmd 路径无需传）。
	OnQuit func()
}

// Assembly 是一次角色装配的产物：四件套面（Inner/EchoSock/BroadSock/
// AppManager）+ DoWait 泵 + 服务本体，启停序对齐 main.lua。
type Assembly struct {
	Member *Member
	Config Config
	Inner  *Inner
	Outer  *Outer
	Echo   *fscomm.EchoSock
	Broad  *fscomm.BroadSock
	App    *fscomm.AppManager

	logf  func(format string, args ...any)
	tps   int
	broadEvery time.Duration
	onQuit func()

	stopOnce sync.Once
	stop     chan struct{}
	wg       sync.WaitGroup
}

// Assemble 复刻 main.lua 的 GameMember 装配段（不启动；Start 按官方序
// 拉起）。m 为 nil 时按 cfg 新建。
func Assemble(m *Member, cfg Config, opt AssembleOptions) (*Assembly, error) {
	if opt.Logf == nil {
		opt.Logf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	}
	if opt.TPS <= 0 {
		opt.TPS = 100
	}
	if opt.BroadcastInterval <= 0 {
		opt.BroadcastInterval = 3 * time.Second
	}
	if opt.AppBookDir == "" {
		opt.AppBookDir = "run"
	}
	if m == nil {
		m = New(cfg.DistrictID, cfg.ServerID, cfg.MemberID)
	}
	a := &Assembly{
		Member: m, Config: cfg,
		logf: opt.Logf, tps: opt.TPS, broadEvery: opt.BroadcastInterval,
		onQuit: opt.OnQuit, stop: make(chan struct{}),
	}
	m.Logf = opt.Logf
	m.OnStateChange = func(from, to MemberState) {
		opt.Logf("member state %s -> %s", from, to)
	}

	// 内网受面（Adapter + ReceiveMsg 路由）
	a.Inner = NewInner(cfg.InnerAddr, cfg.InnerPort, m.OwnIdent(), InnerHandler{
		OnRegister: func(ident uint32) { opt.Logf("inner: ident %#x 注册", ident) },
		OnDrop:     func(ident uint32) { opt.Logf("inner: ident %#x 断开", ident) },
		OnWorld:    m.PutWorld,
		OnStore:    m.PutStore,
		OnPublic:   m.PutPublic,
		OnEntry:    m.ProcessEntry,
		OnError:    func(format string, args ...any) { opt.Logf(format, args...) },
	})

	// 维护面（echo.lua：welcome/quit/end/perf + 未定义命令）
	commands := map[string]fscomm.EchoLineHandler{
		"quit": func(args []string) {
			_ = a.Echo.SendLine(">server will shutdown...\r")
			go a.Stop()
		},
		"end": func(args []string) { a.Echo.Stop() },
		"perf": func(args []string) {
			_ = a.Echo.SendLine(">dump logic module performance...\r")
			opt.Logf("Dump member logic module performance succeed")
		},
		"state": func(args []string) {
			_ = a.Echo.SendLine(">" + m.String() + "\r")
		},
	}
	for k, v := range opt.ExtraEchoCommands {
		commands[k] = v
	}
	a.Echo = fscomm.NewEchoSock(cfg.EchoAddr, cfg.EchoPort, fscomm.EchoSockHandler{
		OnOpen: func(addr string, port int) {
			opt.Logf("echo client %s:%d connected", addr, port)
			_ = a.Echo.SendLine(fmt.Sprintf("Welcome server %d member %d\r", cfg.ServerID, cfg.MemberID))
		},
		OnClose: func(addr string, port int) {
			opt.Logf("echo client %s:%d closed", addr, port)
		},
		OnInput: func(line string) {
			_ = a.Echo.SendLine(">Undefined command: " + line + "\r")
		},
		OnCommand: commands,
	})

	// 广播面（member 信息包 kind=3/magic=0x300）
	a.Broad = fscomm.NewBroadSock(fscomm.BroadSockOptions{
		BindAddress: cfg.InnerAddr, BroadcastAddress: cfg.BroadAddr,
	}, fscomm.BroadSockHandler{})

	// 进程注册（AppManager "member_S_M"）
	app, err := fscomm.NewAppManager(AppName(cfg.ServerID, cfg.MemberID), opt.AppBookDir)
	if err != nil {
		return nil, err
	}
	a.App = app

	// 外发面（CInterConnect）
	a.Outer = NewOuter(m.OwnIdent())
	if opt.WorldAddr != "" {
		a.Outer.AddLink(WorldIdent, opt.WorldAddr, func(ident uint32, up bool) {
			opt.Logf("world link %#x %v", ident, up)
		})
	}
	if opt.PublicAddr != "" {
		a.Outer.AddLink(PublicIdent, opt.PublicAddr, func(ident uint32, up bool) {
			opt.Logf("public link %#x %v", ident, up)
		})
	}
	m.SendToWorld = func(typ uint32, p []byte) error { return a.Outer.Send(WorldIdent, typ, p) }
	m.SendToPublic = func(typ uint32, p []byte) error { return a.Outer.Send(PublicIdent, typ, p) }
	m.SendToStorage = func(ident uint32, typ uint32, p []byte) error { return a.Outer.Send(ident, typ, p) }
	m.SendToAllStor = func(typ uint32, p []byte) error {
		var firstErr error
		for id := StoreIdentBase; id < StoreIdentBase+4; id++ {
			if err := a.Outer.Send(id, typ, p); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}
	m.SendToEntry = func(ident uint32, typ uint32, p []byte) error { return a.Outer.Send(ident, typ, p) }
	return a, nil
}

// AppName 官方进程名 "member_<S>_<M>"。
func AppName(serverID, memberID uint32) string {
	return fmt.Sprintf("member_%d_%d", serverID, memberID)
}

// Start 复刻 main.lua 启动序：AppManager 注册(WAPS_STARTING) →
// Adapter → BroadSock → EchoSock → GameMember(Start/Open) →
// WAPS_OPENED → DoWait 泵 + 周期广播 + console 四连。
func (a *Assembly) Start() error {
	m, cfg := a.Member, a.Config
	if err := a.App.Register(AppName(cfg.ServerID, cfg.MemberID),
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
	m.Open()
	_ = a.App.SetAppState(WAPSOpened)

	a.wg.Add(2)
	go func() { // DoWait 泵（官方主逻辑帧）
		defer a.wg.Done()
		interval := time.Second / time.Duration(a.tps)
		if interval <= 0 {
			interval = 10 * time.Millisecond
		}
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-a.stop:
				return
			case <-tick.C:
				m.DoWait()
			}
		}
	}()
	go func() { // BroadcastMemberInfo 周期包
		defer a.wg.Done()
		tick := time.NewTicker(a.broadEvery)
		defer tick.Stop()
		for {
			select {
			case <-a.stop:
				return
			case <-tick.C:
				_ = a.Broad.SendBroad(m.BuildBroadcast(cfg.InnerAddr, cfg.InnerPort))
			}
		}
	}()

	a.logf("app name %s", AppName(cfg.ServerID, cfg.MemberID))
	a.logf("echo socket %s:%d", cfg.EchoAddr, cfg.EchoPort)
	a.logf("inner socket %s:%d", cfg.InnerAddr, cfg.InnerPort)
	a.logf("broadcast socket %s:%d", cfg.BroadAddr, cfg.BroadPort)
	a.logf("game server %d member %d started", cfg.ServerID, cfg.MemberID)
	return nil
}

// Stop 复刻 main.lua close() 停机序：广播→内网→维护→服务→监控
//（广播先停、监控最后摘牌）+ WAPS_CLOSED。
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
		a.logf("member server shutdown...")
		_ = a.App.SetAppState(WAPSClosed)
		a.App.Close()
		a.wg.Wait()
		if a.onQuit != nil {
			a.onQuit() // 官方 nx_quit()：装载器泵循环退出位
		}
	})
}

// ConfigFromRawIni 从 CoreConfig.Raw（fskit.IniFile）构建 Config——
// 官方 main.lua 以 IniDocument 二次打开 GameMember.ini 的 Go 等价。
func ConfigFromRawIni(ini interface {
	ReadString(section, key, def string) string
	ReadInteger(section, key string, def int) int
}) Config {
	cfg := DefaultConfig()
	cfg.DistrictID = uint32(ini.ReadInteger("Main", "DistrictID", int(cfg.DistrictID)))
	cfg.ServerID = uint32(ini.ReadInteger("Main", "ServerID", int(cfg.ServerID)))
	cfg.MemberID = uint32(ini.ReadInteger("Main", "MemberID", int(cfg.MemberID)))
	cfg.CollideRootPath = ini.ReadString("Main", "CollideRootPath", "")
	cfg.InnerAddr = ini.ReadString("Inner", "Addr", cfg.InnerAddr)
	cfg.InnerPort = ini.ReadInteger("Inner", "Port", 16300+int(cfg.MemberID))
	cfg.BroadAddr = ini.ReadString("Broadcast", "Addr", cfg.BroadAddr)
	cfg.BroadPort = ini.ReadInteger("Broadcast", "Port", cfg.BroadPort)
	cfg.EchoAddr = ini.ReadString("Echo", "Addr", cfg.EchoAddr)
	cfg.EchoPort = ini.ReadInteger("Echo", "Port", 15300+int(cfg.MemberID))
	for i := 0; ; i++ {
		key := fmt.Sprintf("Module%d", i)
		name := ini.ReadString("Logic", key, "")
		if name == "" {
			break
		}
		cfg.LogicModules = append(cfg.LogicModules, name)
	}
	return cfg
}
