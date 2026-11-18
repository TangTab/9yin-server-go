// server.go 充值中心服务核心：TCP 会话、游戏服注册状态机
// （ServerStateMachine@0x6ff80）、用户注册表、18 命令分发表
// （ProcessSockMsg@0x70c60）、keep 超时（CheckServers@0x6fe40，5 分钟）
// 与余额提醒（CheckUsers@0x766b0，AwakeTime 秒，推计费类型 1/3）。
package charge

import (
	"bufio"
	"fmt"
	gonet "net"
	"sync"
	"time"
)

// 会话/状态常量（官方跳表实证）。
const (
	serverStateUnknown    = 0 // 连接占位，未 register
	serverStateRegister   = 1 // 已发 register，等后端确认
	serverStateOnline     = 2 // 在线（业务前置态）
	serverStateUnregister = 3 // 注销中
	serverStateBreak      = 4 // 已断开

	srvEventRegister   = 1 // ReceiveReg
	srvEventRegAck     = 2 // PublishRegister
	srvEventUnreg      = 3 // ReceiveUnreg
	srvEventUnregAck   = 4 // PublishUnregister
	srvEventReset      = 5 // ReceiveReset
	srvEventKeep       = 6 // ReceiveKeep
	srvEventDisconnect = 7 // OnClose/超时

	maxConns     = 256       // m_Servers[0x100]（connid 域）
	lineBufMax   = 0x1000    // OnReceive 行缓冲
	keepTimeout  = 5 * time.Minute
	kickDeadline = 1.0       // 登记时到期宽限（now+1 天，常量@0xee8e8）
	foreverFlush = 3.0       // epoch+3 天内视为从未落账（PublishRenew）
)

// gameStateName 官方 GetState@0x74b70 显示名。
func gameStateName(state int) string {
	switch state {
	case serverStateRegister:
		return "Register"
	case serverStateOnline:
		return "Online"
	case serverStateUnregister:
		return "Unregister"
	case serverStateBreak:
		return "Break"
	}
	return "Unknown"
}

// gameServer 复刻 server_t(0x68)。
type gameServer struct {
	connid   int
	msgid    int // MassSock 频道号（单监听恒 0；GetServer 校验对）
	state    int
	serverID int // 后端确认后回填（+0x14）
	name     string
	addr     string // 对端 ip（accept 记录）
	port     int
	lastKeep time.Time
	zone     int // register 应答 +0x58（[?] 保留域）
	gamePort int // register 应答 +0x54
}

// userInfo 复刻 user_info_t(0xe0) 的运行域（布局无关字段省略）。
type userInfo struct {
	index     int
	account   string // 大写
	passwd    string // 大写（MD5 串）
	address   string
	logonID   string // 票据（login 应答回填）
	accID     int
	hash      uint32
	serverID  int
	connid    int
	msgid     int
	points    int
	chargeTyp int // 计费类型（1/3=时限型 2=点数型）
	unitType  int // renew 应答 unit
	failCnt   int
	kicked    bool
	deadline  float64 // 天数浮点（官方口径）
	lastAwake float64 // CheckUsers 上次主动推 renew 时间
	lastFlush float64 // 最后落账
	loginTime time.Time
}

// Config 是 GameCharge.ini [Main] 投影（main.lua 默认值口径）。
type Config struct {
	GameID         int
	ServerIDOffset int    // serverid = vl[2] + 偏移
	PointMinutes   int    // 每点游戏分钟数（后端扣费换算用）
	AutoDumpMin    int    // 定时导出间隔（分钟；0=关）
	AwakeSec       int    // 余额提醒间隔秒（默认 60）
	FlushErrorMax  int    // renew 连续失败上限（超过转自主判断踢人）
	UserShakeSec   int    // 不活动清理（官方"暂时未用"，保留位）
	FlushLog       bool   // 扣费成功日志开关
	MaxClient      int    // MassSock ConnectMax（默认 16）
	Addr           string // [Outer] 监听
	Port           int
}

// DefaultConfig 官方 main.lua 兜底默认值。
func DefaultConfig() Config {
	return Config{
		GameID:       1,
		PointMinutes: 10,
		AwakeSec:     60,
		MaxClient:    16,
		Addr:         "127.0.0.1",
		Port:         7000,
	}
}

// Server 是充值中心实例。
type Server struct {
	Config  Config
	Backend Backend
	Logf    func(format string, args ...any)

	ln       gonet.Listener
	mu       sync.Mutex
	servers  [maxConns]*gameServer // 按 connid 直索引（m_Servers）
	byID     map[int]*gameServer
	users    []*userInfo // 按 index（m_Users），index 0 起
	freeIdx  []int
	byAcc    map[string]*userInfo
	byAccID  map[int]*userInfo
	sessions map[int]*session
	closed   bool

	nowFunc func() time.Time // 测试可注入
}

// NewServer 构造充值中心（backend 为 nil 时用 FreeBackend）。
func NewServer(cfg Config, backend Backend) *Server {
	if backend == nil {
		backend = FreeBackend{}
	}
	if cfg.MaxClient <= 0 {
		cfg.MaxClient = 16
	}
	return &Server{
		Config:   cfg,
		Backend:  backend,
		Logf:     func(string, ...any) {},
		byID:     map[int]*gameServer{},
		byAcc:    map[string]*userInfo{},
		byAccID:  map[int]*userInfo{},
		sessions: map[int]*session{},
		nowFunc:  time.Now,
	}
}

func (s *Server) logf(format string, args ...any) { s.Logf(format, args...) }

func (s *Server) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
}

// daysSinceEpoch 官方时间口径：自 1899-12-30 起的天数浮点
// （get_cur_time 家族；宽限常量 +1.0 即"一天"）。
var excelEpoch = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)

func daysSinceEpoch(t time.Time) float64 {
	return t.Sub(excelEpoch).Hours() / 24
}

// Listen 绑定监听地址。
func (s *Server) Listen() (gonet.Addr, error) {
	ln, err := gonet.Listen("tcp", gonet.JoinHostPort(s.Config.Addr, fmt.Sprintf("%d", s.Config.Port)))
	if err != nil {
		return nil, err
	}
	s.ln = ln
	return ln.Addr(), nil
}

// Serve 接受循环（阻塞直至 Close）。
func (s *Server) Serve() error {
	if s.ln == nil {
		return fmt.Errorf("charge: 未 Listen")
	}
	stopTicker := make(chan struct{})
	go s.housekeeper(stopTicker)
	defer close(stopTicker)
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return nil
			}
			return err
		}
		go s.handleConn(conn)
	}
}

// Close 停止接受并断开全部会话（幂等）。
func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	var lnErr error
	if s.ln != nil {
		lnErr = s.ln.Close()
	}
	sessions := make([]*session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.mu.Unlock()
	for _, sess := range sessions {
		sess.conn.Close()
	}
	return lnErr
}

// session 一条游戏服 TCP 连接。
type session struct {
	srv    *Server
	conn   gonet.Conn
	connid int
}

// handleConn 承接 OnAccept/OnReceive/OnClose 生命周期。
func (s *Server) handleConn(conn gonet.Conn) {
	s.mu.Lock()
	if len(s.sessions) >= s.Config.MaxClient {
		s.mu.Unlock()
		s.logf("charge: 连接数达上限 %d，拒绝 %s", s.Config.MaxClient, conn.RemoteAddr())
		conn.Close()
		return
	}
	var connid int = -1
	for i := 0; i < maxConns; i++ {
		if s.servers[i] == nil {
			connid = i
			break
		}
	}
	if connid < 0 {
		s.mu.Unlock()
		conn.Close()
		return
	}
	sess := &session{srv: s, conn: conn, connid: connid}
	s.sessions[connid] = sess
	addr, _, _ := gonet.SplitHostPort(conn.RemoteAddr().String())
	s.addServerLocked(connid, addr)
	s.mu.Unlock()
	s.logf("Address %s connected, total %d", conn.RemoteAddr(), len(s.sessions))

	defer func() {
		s.mu.Lock()
		delete(s.sessions, connid)
		sv := s.servers[connid]
		if sv != nil {
			// event7：RemoveServer（关槽位；不自动清 user，官方语义）
			s.removeServerLocked(sv)
		}
		n := len(s.sessions)
		s.mu.Unlock()
		conn.Close()
		s.logf("Address %s closed, total %d", conn.RemoteAddr(), n)
	}()

	r := bufio.NewReaderSize(conn, 8192)
	line := make([]byte, 0, 256)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return
		}
		if b == '\n' {
			if len(line) > 0 {
				s.processLine(sess, line)
				line = line[:0]
			}
			continue
		}
		if b < 0x20 {
			continue // 官方丢弃不可见字符（含 \r）
		}
		if len(line) >= lineBufMax {
			continue // 官方超长静默丢弃
		}
		line = append(line, b)
	}
}

// addServerLocked 连接占位（ProcessInnerMsg type0 → AddServer@0x6fad0）。
func (s *Server) addServerLocked(connid int, addr string) {
	sv := &gameServer{
		connid:   connid,
		addr:     addr,
		state:    serverStateUnknown,
		lastKeep: s.now(),
	}
	s.servers[connid] = sv
}

// removeServerLocked RemoveServer@0x6fc10：清槽位+索引。
func (s *Server) removeServerLocked(sv *gameServer) {
	if s.servers[sv.connid] == sv {
		s.servers[sv.connid] = nil
	}
	if sv.serverID != 0 && s.byID[sv.serverID] == sv {
		delete(s.byID, sv.serverID)
	}
}

// getServerLocked GetServer@0x6fcf0：(connid, msgid) 对校验。
func (s *Server) getServerLocked(connid, msgid int) *gameServer {
	if connid < 0 || connid >= maxConns {
		return nil
	}
	sv := s.servers[connid]
	if sv == nil || sv.msgid != msgid {
		return nil
	}
	return sv
}

// findServerLocked FindServer：按后端 serverID 查重（重复注册路径）。
func (s *Server) findServerLocked(serverID int) *gameServer {
	return s.byID[serverID]
}

// processLine 一行 = 一条消息（text_decode → 分发）。
func (s *Server) processLine(sess *session, line []byte) {
	vl, err := DecodeLine(string(line))
	if err != nil {
		s.logf("(charge)decode msg failed: %v", err)
		return
	}
	if len(vl) < 2 {
		s.logf("(GameCharge::ProcessSockMsg)message too short")
		return
	}
	echo := vl.StringVal(0)
	cmd := vl.StringVal(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	switch cmd {
	case "register":
		s.cmdRegister(sess, echo, vl)
	case "unregister":
		s.cmdUnregister(sess, echo, vl)
	case "reset":
		s.cmdReset(sess, vl)
	case "login":
		s.cmdLogin(sess, echo, vl)
	case "logout":
		s.cmdLogout(sess, echo, vl)
	case "renew":
		s.cmdRenew(sess, echo, vl)
	case "query_gift":
		s.cmdQueryGift(sess, echo, vl)
	case "get_gift":
		s.cmdGetGift(sess, echo, vl)
	case "sell_card":
		s.cmdSellCard(sess, echo, vl)
	case "unsell_card":
		s.cmdUnsellCard(sess, echo, vl)
	case "buy_card":
		s.cmdBuyCard(sess, echo, vl)
	case "buy_item":
		s.cmdBuyItem(sess, echo, vl)
	case "buy_item_give":
		s.cmdBuyItemGive(sess, echo, vl)
	case "item_info":
		s.cmdItemInfo(sess, echo, vl)
	case "get_points":
		s.cmdGetPoints(sess, echo, vl)
	case "validate":
		s.cmdValidate(sess, echo, vl)
	case "custom":
		// 官方 ReceiveCustom@0x747e0：空实现（member 透传通道预留）。
	case "keep":
		s.cmdKeep(sess, vl)
	default:
		s.logf("(GameCharge::ProcessSockMsg)unknown command %q", cmd)
	}
}

// respond 构造应答并回发（SendResponse@0x66910：[echo, cmd, ...] + CRLF）。
func (s *Server) respond(sess *session, vl VarList) {
	data := append([]byte(vl.EncodeLine()), '\r', '\n')
	sess.conn.Write(data) //nolint:errcheck — 单连接写失败由读循环收尸
}

// ---- register 家族（ServerStateMachine） ----

// cmdRegister ReceiveReg → Unknown(event1)。请求：
// [echo, "register", serial(读而未用), serverID, name, ident(md5), port]。
func (s *Server) cmdRegister(sess *session, echo string, vl VarList) {
	sv := s.getServerLocked(sess.connid, 0)
	if sv == nil {
		s.logf("(GameCharge::ReceiveReg)Server not found")
		return
	}
	if sv.state != serverStateUnknown {
		// 官方 Register 态收到 event1 走 ServerStateError；只有 Unknown 态受理。
		s.logf("(GameCharge::ServerState%s)error input %d", gameStateName(sv.state), srvEventRegister)
		return
	}
	serverID := vl.Int(3) + s.Config.ServerIDOffset
	if old := s.findServerLocked(serverID); old != nil {
		s.logf("(GameCharge::ServerStateUnknown)already register")
		out := VarList{}
		out.AddString(echo)
		out.AddString("register")
		out.AddInt(0)
		s.respond(sess, out)
		return
	}
	sv.name = vl.StringVal(4)
	sv.state = serverStateRegister

	reg := ServerReg{
		LogicalID: serverID,
		Name:      sv.name,
		Ident:     vl.StringVal(5),
		Port:      vl.Int(6),
		Addr:      sv.addr,
	}
	result, dbID, gamePort, zone := s.Backend.RegisterServer(reg)
	// PublishRegister → Register(event2)
	if result == 1 {
		sv.serverID = dbID
		sv.port = gamePort
		sv.gamePort = gamePort
		sv.zone = zone
		s.byID[dbID] = sv
		sv.state = serverStateOnline
		s.logf("Server ID: %d Name: %s Address: %s Port: %d register succeed",
			dbID, sv.name, sv.addr, gamePort)
	} else {
		s.logf("Server ID: %d Name: %s register failed result %d", dbID, sv.name, result)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("register")
	out.AddInt(result)
	s.respond(sess, out)
}

// cmdUnregister ReceiveUnreg → Online(event3)。
func (s *Server) cmdUnregister(sess *session, echo string, vl VarList) {
	sv := s.getServerLocked(sess.connid, 0)
	if sv == nil {
		s.logf("(GameCharge::ReceiveUnreg)Server not found")
		return
	}
	if sv.state != serverStateOnline {
		s.logf("(GameCharge::ServerState%s)error input %d", gameStateName(sv.state), srvEventUnreg)
		return
	}
	sv.state = serverStateUnregister
	result := s.Backend.UnregisterServer(sv.serverID)
	if result == 1 {
		sv.state = serverStateBreak
		s.logf("Server ID: %d unregistered", sv.serverID)
	} else {
		s.logf("(GameCharge::ServerStateUnregister)unregister failed result %d", result)
	}
	out := VarList{}
	out.AddString(echo)
	out.AddString("unregister")
	out.AddInt(result)
	s.respond(sess, out)
}

// cmdReset ReceiveReset → Online/Register/Unregister(event5)：
// 清本服全部在线用户，无应答。
func (s *Server) cmdReset(sess *session, vl VarList) {
	sv := s.getServerLocked(sess.connid, 0)
	if sv == nil {
		s.logf("(GameCharge::ReceiveReset)Server not found")
		return
	}
	switch sv.state {
	case serverStateRegister, serverStateOnline, serverStateUnregister:
	default:
		s.logf("(GameCharge::ServerState%s)error input %d", gameStateName(sv.state), srvEventReset)
		return
	}
	s.removeUsersByServerLocked(sv.serverID)
}

// cmdKeep ReceiveKeep：刷新 lastKeep（Online 态），无应答。
func (s *Server) cmdKeep(sess *session, vl VarList) {
	sv := s.getServerLocked(sess.connid, 0)
	if sv == nil {
		s.logf("(GameCharge::ReceiveKeep)Server not found")
		return
	}
	sv.lastKeep = s.now()
}

// ---- 用户注册表 ----

// addUserLocked AddUser@0x6d180：登记返回 index；顶号（UniqueAccount）
// 踢同账号旧会话（向旧连接发空回显 renew 踢人帧）。index 从 1 起
// （0 保留为"按账号查询"哨兵，官方 logout/renew 的 useridx==0 语义）。
func (s *Server) addUserLocked(serverID, connid, msgid int, account, passwd, address string) *userInfo {
	if old, ok := s.byAcc[account]; ok {
		s.uniqueAccountLocked(old)
	}
	if len(s.users) == 0 {
		s.users = append(s.users, nil) // 0 号占位
	}
	idx := len(s.users)
	if n := len(s.freeIdx); n > 0 {
		idx = s.freeIdx[n-1]
		s.freeIdx = s.freeIdx[:n-1]
	}
	u := &userInfo{
		index:     idx,
		account:   account,
		passwd:    passwd,
		address:   address,
		serverID:  serverID,
		connid:    connid,
		msgid:     msgid,
		deadline:  daysSinceEpoch(s.now()) + kickDeadline,
		loginTime: s.now(),
		hash:      hashAccount(account),
	}
	if idx == len(s.users) {
		s.users = append(s.users, u)
	} else {
		s.users[idx] = u
	}
	s.byAcc[account] = u
	return u
}

// uniqueAccountLocked 顶号：官方对旧 user 直接踢（KickOut 帧）。
func (s *Server) uniqueAccountLocked(old *userInfo) {
	if old.kicked {
		return
	}
	old.kicked = true
	if sess := s.sessions[old.connid]; sess != nil {
		out := VarList{}
		out.AddString("")
		out.AddString("renew")
		out.AddInt(old.accID)
		out.AddInt(2)
		out.AddInt(old.unitType)
		out.AddDouble(old.deadline)
		s.respond(sess, out)
	}
	s.logf("Account: %s ID: %d kick out (duplicate login)", old.account, old.accID)
	s.releaseUserLocked(old)
}

// releaseUserLocked ReleaseUser@0x756e0：摘双哈希+回收 index。
func (s *Server) releaseUserLocked(u *userInfo) {
	if s.byAcc[u.account] == u {
		delete(s.byAcc, u.account)
	}
	if u.accID != 0 && s.byAccID[u.accID] == u {
		delete(s.byAccID, u.accID)
	}
	if s.users[u.index] == u {
		s.users[u.index] = nil
		s.freeIdx = append(s.freeIdx, u.index)
	}
}

// removeUsersByServerLocked RemoveUserByServer@0x75600。
func (s *Server) removeUsersByServerLocked(serverID int) {
	for _, u := range s.users {
		if u != nil && u.serverID == serverID {
			s.releaseUserLocked(u)
		}
	}
}

// getUserActiveLocked GetUserActive：serverid+useridx 定位（0 视为缺省）。
func (s *Server) getUserActiveLocked(serverID, useridx, connid, msgid int) *userInfo {
	if useridx <= 0 || useridx >= len(s.users) {
		return nil
	}
	u := s.users[useridx]
	if u == nil || u.serverID != serverID {
		return nil
	}
	_ = connid
	_ = msgid
	return u
}

// getUserByAccountLocked 按大写账号。
func (s *Server) getUserByAccountLocked(account string, connid, msgid int) *userInfo {
	u := s.byAcc[account]
	if u == nil {
		return nil
	}
	_ = connid
	_ = msgid
	return u
}

// hashAccount 复刻 GetHashValue(account)（FNV-1a 32；官方仅用于池内
// 分片，Go 单后端下用于稳定排序语义，不参与线协议）。
func hashAccount(account string) uint32 {
	var h uint32 = 0x811C9DC5
	for i := 0; i < len(account); i++ {
		h ^= uint32(account[i])
		h *= 0x01000193
	}
	return h
}
