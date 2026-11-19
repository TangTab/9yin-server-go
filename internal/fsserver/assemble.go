package fsserver

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/charge"
)

// Assembly 复刻 main.lua 装配序（§1）：EchoSock 维护面 + Adapter 内网
// 受面 + BroadSock 广播 + DoWait 泵；可选 charge TCP 客户端。
type Assembly struct {
	World *World
	Inner *Inner
	Cfg   Config

	echoLn    net.Listener
	broadConn *net.UDPConn
	broadAddr *net.UDPAddr

	chargeConn net.Conn
	charge     *ChargeClient

	pumpMu  sync.Mutex
	stopCh  chan struct{}
	stopped chan struct{}
	tps     int
}

// AssembleOptions …
type AssembleOptions struct {
	TPS        int                                 // DoWait 每秒帧数（缺省 100）
	ChargeDial func(addr string) (net.Conn, error) // charge 连接器（测试注入）
}

// Assemble 装配（不启动——Start 序见 Assembly.Start）。
func Assemble(cfg Config, opt AssembleOptions) (*Assembly, error) {
	w := NewWorld()
	w.DistrictID = cfg.DistrictID
	w.ServerID = cfg.ServerID
	w.ServerName = cfg.ServerName
	if cfg.MaxPlayers > 0 {
		w.MaxPlayers = cfg.MaxPlayers
	}
	w.Logf = func(format string, args ...any) {
		fmt.Printf("[world] "+format+"\n", args...)
	}

	a := &Assembly{
		World:   w,
		Cfg:     cfg,
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
		tps:     opt.TPS,
	}
	if a.tps <= 0 {
		a.tps = 100
	}

	in := NewInner(w, cfg.InnerAddr, cfg.InnerPort)
	a.Inner = in

	// InterConnect 发送族接线
	w.IC = &InterConnect{
		Send: func(ident, dst, typ uint32, payload []byte) error {
			return in.sendIdent(ident, dst, typ, payload)
		},
		Connected: in.Connected,
	}
	w.SendToEntryFn = func(entryIdx, dst, typ uint32, payload []byte) error {
		return in.sendIdent(EntryIdent(entryIdx), dst, typ, payload)
	}

	// charge 可选
	if cfg.ChargeInuse && cfg.ChargeAddr != "" {
		a.charge = NewChargeClient(cfg.DistrictID, cfg.ServerID)
		a.charge.OnVarList = func(v charge.VarList) { w.PutChargeVarList(v) }
		dial := opt.ChargeDial
		if dial == nil {
			dial = func(addr string) (net.Conn, error) { return net.DialTimeout("tcp", addr, 3*time.Second) }
		}
		if conn, err := dial(net.JoinHostPort(cfg.ChargeAddr, fmt.Sprint(cfg.ChargePort))); err == nil {
			a.chargeConn = conn
			a.charge.SendLine = func(line string) error {
				_, err := conn.Write([]byte(line))
				return err
			}
			_ = a.charge.SendRegister(fmt.Sprintf("world_%d", cfg.ServerID))
			go a.chargeReadLoop(conn)
		}
	}
	return a, nil
}

// Start 启动序（main.lua：app_manager → adapter → broad_sock → echo_sock →
// game_world）。任一面失败即返回错误（官方致命口径）。
func (a *Assembly) Start() error {
	w := a.World
	// 内网受面
	if err := a.Inner.Start(); err != nil {
		return fmt.Errorf("adapter: %w", err)
	}
	// 广播口（UDP；发送侧专用——BroadSock 双口收发，world 只发）
	if conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP(a.Cfg.BroadAddr),
		Port: a.Cfg.BroadPort,
	}); err == nil {
		a.broadConn = conn
	}
	// 维护面
	ln, err := net.Listen("tcp", net.JoinHostPort(a.Cfg.EchoAddr, fmt.Sprint(a.Cfg.EchoPort)))
	if err != nil {
		return fmt.Errorf("echo socket: %w", err)
	}
	a.echoLn = ln
	go a.echoAccept()

	// DoWait 泵
	go a.pumpLoop()

	w.logf("App name world_%d", a.Cfg.ServerID)
	w.logf("Echo socket %s:%d", a.Cfg.EchoAddr, a.Cfg.EchoPort)
	w.logf("Inner socket %s:%d", a.Cfg.InnerAddr, a.Cfg.InnerPort)
	w.logf("Broadcast socket %s:%d", a.Cfg.BroadAddr, a.Cfg.BroadPort)
	w.logf("Game server %d starting...", a.Cfg.ServerID)
	return nil
}

// Stop 停机序（close()：broad → adapter → echo → game_world）。
func (a *Assembly) Stop() {
	select {
	case <-a.stopCh:
		return // 幂等
	default:
	}
	close(a.stopCh)
	select {
	case <-a.stopped:
	case <-time.After(2 * time.Second):
	}
	if a.broadConn != nil {
		_ = a.broadConn.Close()
	}
	a.Inner.Stop()
	if a.echoLn != nil {
		_ = a.echoLn.Close()
	}
	if a.chargeConn != nil {
		_ = a.chargeConn.Close()
	}
}

// pumpLoop DoWait 主泵 + 1s tick（TPS 节流）。
func (a *Assembly) pumpLoop() {
	defer close(a.stopped)
	interval := time.Second / time.Duration(a.tps)
	tick := time.NewTicker(interval)
	secTick := time.NewTicker(time.Second)
	defer tick.Stop()
	defer secTick.Stop()
	for {
		select {
		case <-a.stopCh:
			return
		case <-tick.C:
			a.World.Pump()
		case <-secTick.C:
			a.World.TickSeconds()
			a.broadcastInfo()
			if a.charge != nil {
				_ = a.charge.Keepalive()
			}
		}
	}
}

// broadcastInfo 120B 区服信息包（§6.1）。
func (a *Assembly) broadcastInfo() {
	if a.broadConn == nil {
		return
	}
	w := a.World
	pkt := BuildBroadcastPacket(
		a.Cfg.InnerAddr, uint32(a.Cfg.InnerPort),
		w.DistrictID, w.ServerID, w.ServerName,
		w.Players.Count(), w.State == WStateOpened,
	)
	_, _ = a.broadConn.Write(pkt)
}

// echoAccept 维护面（echo.lua：welcome/quit/players/state…）。
func (a *Assembly) echoAccept() {
	for {
		conn, err := a.echoLn.Accept()
		if err != nil {
			return
		}
		go a.echoServe(conn)
	}
}

func (a *Assembly) echoServe(conn net.Conn) {
	defer conn.Close()
	fmt.Fprintf(conn, "Welcome server %d", a.Cfg.ServerID)
	// 官方 EchoSock 行方言：'\r'/'\n' 任一终止（§17）；行缓冲 0x3FF 截断
	buf := make([]byte, 0, 512)
	chunk := make([]byte, 256)
	for {
		n, err := conn.Read(chunk)
		if n > 0 {
			for i := 0; i < n; i++ {
				c := chunk[i]
				if c == '\r' || c == '\n' {
					line := strings.TrimSpace(string(buf))
					buf = buf[:0]
					if line != "" && !a.echoCommand(conn, line) {
						return
					}
					continue
				}
				if len(buf) < 0x3FF {
					buf = append(buf, c)
				}
			}
		}
		if err != nil {
			return
		}
	}
}

// echoCommand 单命令分发；返回 false 表示连接结束（end/quit）。
func (a *Assembly) echoCommand(conn net.Conn, cmd string) bool {
	switch cmd {
	case "quit":
		fmt.Fprintln(conn, "server will shutdown...")
		go a.shutdown()
		return false
	case "end":
		return false
	case "players":
		w := a.World
		fmt.Fprintf(conn, "player: %d, online: %d", w.Players.Count(), w.OnlineCount())
	case "state":
		fmt.Fprintf(conn, "state: %s", a.World.State)
	case "scenes":
		n, c := a.World.Scenes.Counts()
		fmt.Fprintf(conn, "scene: %d, clone: %d", n, c)
	default:
		fmt.Fprintf(conn, "unknown command %q", cmd)
	}
	return true
}

func (a *Assembly) shutdown() {
	// 官方 close() 序：WorldStateMachine CloseEntry 起步（本停机为进程级）
	a.Stop()
}

// chargeReadLoop 收侧行循环（UserSock OnReceive 语义：\r/\n 切行）。
func (a *Assembly) chargeReadLoop(conn net.Conn) {
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 4096), 4096)
	for sc.Scan() {
		if a.charge != nil {
			a.charge.ReceiveLine(sc.Text())
		}
	}
}
