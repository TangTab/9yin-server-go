package entry

import (
	"fmt"
	"net"
	"sync"

	gonet "net"

	fsnet "github.com/lizhanyin/9yin-server-go/internal/net"
)

// Gateway 是外网面（CMassSockHook + CClients 复刻）：客户端 TCP 侦听 +
// EE 转义 + XOR 线路 + 槽位池。纯代理语义：收到的每帧按绑定关系转发
// world/member；world 经内网面反向驱动（ClientSend/SightSend/…）。
type Gateway struct {
	Addr     string
	NetOpts  fsnet.Options
	EntryID  uint16 // ident 高 16 位
	MaxFrame int    // 官方单帧缓冲上限 0x1000

	Clients *Clients
	Inner   *Inner // 上行通道（可为 nil：纯侦听模式）
	Logf    func(format string, args ...any)

	// VerifyMessage 可选每帧校验钩子（fsverify 接线位）；返回非 nil 即
	// 非法帧（官方 CheckMessageIllegal → 回报 world type 4）。
	VerifyMessage func(ident, serial uint32, frame []byte) error

	mu       sync.Mutex
	ln       gonet.Listener
	conns    map[int]gonet.Conn   // slotIdx → TCP 连接
	writers  map[int]*fsnet.Connection // slotIdx → 编码连接
	closed   bool                  // 官方 mass_sock.Closed 闸（暂停受理）
	blockIP  *BlockIPList
	msgStats [64]uint32 // op 1..63 计数（官方 this+op*8）
	logSend  bool
	logRecv  bool
}

// NewGateway 装配外网面。
func NewGateway(addr string, entryID uint16, clients *Clients, inner *Inner, opts fsnet.Options) *Gateway {
	return &Gateway{
		Addr:     addr,
		EntryID:  entryID,
		MaxFrame: 0x1000,
		Clients:  clients,
		Inner:    inner,
		NetOpts:  opts,
		conns:    make(map[int]gonet.Conn),
		writers:  make(map[int]*fsnet.Connection),
		blockIP:  NewBlockIPList(),
		Logf:     func(string, ...any) {},
	}
}

// SetClosed 官方 mass_sock.Closed 闸：true=暂停受理新连接。
func (g *Gateway) SetClosed(closed bool) {
	g.mu.Lock()
	g.closed = closed
	g.mu.Unlock()
}

// SetLogReceive 官方 LogReceive：上行帧落 message.log（此处控台输出）。
func (g *Gateway) SetLogReceive(on bool) { g.logRecv = on }

// Listen 绑定端口并进入受理循环。
func (g *Gateway) Listen() (gonet.Addr, error) {
	ln, err := gonet.Listen("tcp", g.Addr)
	if err != nil {
		return nil, fmt.Errorf("entry: gateway listen failed: %w", err)
	}
	g.mu.Lock()
	g.ln = ln
	g.mu.Unlock()
	go g.acceptLoop(ln)
	return ln.Addr(), nil
}

// Stop 关停（官方 close()：Closed=true → CloseAll → Stop）。
func (g *Gateway) Stop() {
	g.mu.Lock()
	g.closed = true
	ln := g.ln
	conns := make([]gonet.Conn, 0, len(g.conns))
	for _, c := range g.conns {
		conns = append(conns, c)
	}
	g.mu.Unlock()
	if ln != nil {
		ln.Close()
	}
	for _, c := range conns {
		c.Close()
	}
}

func (g *Gateway) acceptLoop(ln gonet.Listener) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go g.onAccept(c)
	}
}

func (g *Gateway) onAccept(c gonet.Conn) {
	host, portStr, _ := net.SplitHostPort(c.RemoteAddr().String())
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	g.mu.Lock()
	closed := g.closed
	g.mu.Unlock()
	if closed { // "address %s:%d server closed"
		c.Close()
		return
	}
	if g.blockIP.Blocked(host) { // "address %s:%d ip blocked"
		g.Logf("address %s:%d ip blocked", host, port)
		c.Close()
		return
	}
	idx := g.Clients.Alloc(g.EntryID, host)
	if idx < 0 { // "client array too big"
		g.Logf("(Gateway::onAccept)client array too big")
		c.Close()
		return
	}
	st := g.Clients.Client(idx)
	conn, err := fsnet.NewConnection(c, g.NetOpts)
	if err != nil {
		g.Clients.Release(idx)
		c.Close()
		return
	}
	g.mu.Lock()
	g.conns[idx] = c
	g.writers[idx] = conn
	g.mu.Unlock()
	g.Logf("address %s:%d connected, total %d", host, port, func() int {
		cc, _, _ := g.Clients.Stats()
		return cc
	}())

	// 官方 OnAccept：向 world 上报 28B 通知
	if g.Inner != nil {
		_ = g.Inner.SendToWorld(UpstreamAccept, BuildUpstreamAccept(st.Ident, st.Serial, host, port))
	}

	defer func() {
		// 官方 OnClose：向 world 回报 type 2 连接关闭（entry_back_client_
		// closed_t 8B）——world 侧 entryClientClosed 走 0x37 logoff +
		// playerLeave 清场。先报后释：lost 事件须先于槽位复用入网，
		// 杜绝新连接（serial 单调不复用，槽位会复用）误吃旧键清场。
		if g.Inner != nil {
			_ = g.Inner.SendToWorld(UpstreamLost, BuildUpstreamLost(st.Ident, st.Serial))
		}
		g.mu.Lock()
		delete(g.conns, idx)
		delete(g.writers, idx)
		g.mu.Unlock()
		g.Clients.Release(idx)
		c.Close()
	}()
	for {
		frame, err := conn.ReadMsg()
		if err != nil {
			return
		}
		g.onFrame(idx, frame)
	}
}

// onFrame 复刻 ReceiveMessage：统计 → 校验 → 绑定路由（member/world）。
func (g *Gateway) onFrame(idx int, frame []byte) {
	if len(frame) == 0 {
		return
	}
	st := g.Clients.Client(idx)
	op := int(frame[0])
	if op >= 1 && op < len(g.msgStats) {
		g.msgStats[op]++
	}
	if g.VerifyMessage != nil {
		if err := g.VerifyMessage(st.Ident, st.Serial, frame); err != nil {
			if g.Inner != nil {
				_ = g.Inner.SendToWorld(UpstreamIllegal, BuildUpstreamData(st.Ident, st.Serial, frame))
			}
			return
		}
	}
	if g.logRecv {
		g.Logf("recv %#x/%d op=%#x len=%d", st.Ident, st.Serial, frame[0], len(frame))
	}
	if g.Inner == nil {
		return
	}
	payload := BuildUpstreamData(st.Ident, st.Serial, frame)
	if st.Member >= 0 && st.Member < MaxMemberNum {
		if err := g.Inner.SendToMember(int(st.Member), UpstreamData, payload); err != nil {
			g.Logf("(Gateway::onFrame)send to member failed: %v", err)
		}
		return
	}
	if err := g.Inner.SendToWorld(UpstreamData, payload); err != nil {
		g.Logf("(Gateway::onFrame)send to world failed: %v", err)
	}
}

// clientConn 取槽位对应编码连接。
func (g *Gateway) clientConn(idx int) *fsnet.Connection {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.writers[idx]
}

// ClientSend 复刻 send_to_client：向指定客户端发帧。
func (g *Gateway) ClientSend(ident, serial uint32, payload []byte) error {
	st := g.Clients.Lookup(ident, serial)
	if st == nil {
		return fmt.Errorf("(Gateway::ClientSend)client not exists: %#x/%d", ident, serial)
	}
	conn := g.clientConn(idxOf(ident))
	if conn == nil {
		return fmt.Errorf("(Gateway::ClientSend)client not exists: no connector")
	}
	return conn.WriteMsg(payload)
}

// ClientClose 复刻 break_client：主动踢线。
func (g *Gateway) ClientClose(ident, serial uint32) {
	st := g.Clients.Lookup(ident, serial)
	if st == nil {
		return
	}
	if conn := g.clientConn(idxOf(ident)); conn != nil {
		conn.Close()
	}
}

// Shake 复刻 shake_client：在线即回发 1 字节保活帧 {0x06}，
// 失联向 world 回报 type 2。
func (g *Gateway) Shake(ident, serial uint32) {
	st := g.Clients.Lookup(ident, serial)
	if st == nil {
		if g.Inner != nil {
			_ = g.Inner.SendToWorld(UpstreamLost, BuildUpstreamLost(ident, serial))
		}
		return
	}
	if err := g.ClientSend(ident, serial, []byte{0x06}); err != nil {
		if g.Inner != nil {
			_ = g.Inner.SendToWorld(UpstreamLost, BuildUpstreamLost(ident, serial))
		}
	}
}

// SendAll / SendOnline / SendChannel 复刻三个广播面。
func (g *Gateway) SendAll(payload []byte) { g.sendList(g.Clients.AllMembers(), payload) }

func (g *Gateway) SendOnline(payload []byte) { g.sendList(g.Clients.OnlineMembers(), payload) }

func (g *Gateway) SendChannel(channel uint32, payload []byte) {
	g.sendList(g.Clients.ChannelMembers(channel), payload)
}

// SendSight 复刻 send_to_sight：网格螺旋遍历 + 距离²过滤。
func (g *Gateway) SendSight(ident uint32, sceneID uint32, x, y, radius int32, payload []byte) {
	for _, idx := range g.Clients.SightMembers(ident, sceneID, x, y, radius) {
		st := g.Clients.Client(idx)
		if err := g.ClientSend(st.Ident, st.Serial, payload); err != nil {
			g.Logf("(Gateway::SendSight)send failed: %v", err)
		}
	}
}

func (g *Gateway) sendList(list []int, payload []byte) {
	for _, idx := range list {
		st := g.Clients.Client(idx)
		if err := g.ClientSend(st.Ident, st.Serial, payload); err != nil {
			g.Logf("(Gateway::sendList)send failed: %v", err)
		}
	}
}

// BlockIPList 复刻 CBlockIP：IP 区段黑名单（ip.txt "beg,end" 行）。
type BlockIPList struct {
	mu     sync.Mutex
	ranges [][2]uint32
}

func NewBlockIPList() *BlockIPList { return &BlockIPList{} }

// AddBlockIP 登记区段（点分十进制）；返回是否成功。
func (b *BlockIPList) AddBlockIP(beg, end string) bool {
	lo, ok1 := parseIPv4(beg)
	hi, ok2 := parseIPv4(end)
	if !ok1 || !ok2 || lo > hi {
		return false
	}
	b.mu.Lock()
	b.ranges = append(b.ranges, [2]uint32{lo, hi})
	b.mu.Unlock()
	return true
}

// Clear 清空。
func (b *BlockIPList) Clear() {
	b.mu.Lock()
	b.ranges = b.ranges[:0]
	b.mu.Unlock()
}

// Blocked 命中判定。
func (b *BlockIPList) Blocked(ip string) bool {
	v, ok := parseIPv4(ip)
	if !ok {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, r := range b.ranges {
		if v >= r[0] && v <= r[1] {
			return true
		}
	}
	return false
}

// BlockIP 暴露黑名单（维护面 loadip 复用）。
func (g *Gateway) BlockIP() *BlockIPList { return g.blockIP }

func parseIPv4(s string) (uint32, bool) {
	var a, b, c, d int
	if n, _ := fmt.Sscanf(s, "%d.%d.%d.%d", &a, &b, &c, &d); n != 4 {
		return 0, false
	}
	if a < 0 || a > 255 || b < 0 || b > 255 || c < 0 || c > 255 || d < 0 || d > 255 {
		return 0, false
	}
	return uint32(a)<<24 | uint32(b)<<16 | uint32(c)<<8 | uint32(d), true
}
