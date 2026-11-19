package snaileditornet

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/snailsharemem"
)

// 回调面：由游戏侧模块（snaileditor 包）注入，对应官方
// SetDelaySvnUpdate(FuncDelaySvnUpdateCallback) /
// SetSvnUpdateCompleteCallback(SnailEditorServerModule::PushMsg)。
type Callbacks struct {
	// OnDelaySvnUpdate 官方 sMsg type=0：路径并入延迟 SVN 更新列表。
	OnDelaySvnUpdate func(path string)
	// OnEditorCommand 官方 sMsg type=1：编辑器命令帧（PushMsg → MsgManager）。
	OnEditorCommand func(data string)
	// OnSvnUpdateComplete 官方 SetSvnConflictChangedCallback 之外的完成回执面。
	OnSvnUpdateComplete func(paths []string)
	// OnConflict 官方 FMsgFileInUse：文件被占用。
	OnConflict func(file string)
	// OnConnState 连接建立/断开通知（日志面）。
	OnConnState func(up bool, remote string)
}

// Config Net 层配置（官方 ini\SnailEditorServerModule.xml 的 config/watch 段）。
type Config struct {
	SvnServerIP   string   // 编辑器 GUI 地址（ClientConnect 目标）
	SvnServerPort int      // 编辑器 GUI 端口
	SvnLocalRoot  string   // 本地工作副本根（TortoiseProc /path）
	TortoiseProc  string   // TortoiseProc.exe 路径（空=注册表探测）
	WatchPaths    []string // watch 目录（仅其中路径触发更新）
}

// Server 编辑器 GUI 侧（ServerSession+CIOCP 等价）：多连接 + 0x3F1 广播。
type Server struct {
	ln net.Listener

	mu    sync.Mutex
	conns map[net.Conn]struct{}
	wg    sync.WaitGroup

	onConflict func(file string)
	onFrame    func(msgID uint32, payload []byte)
}

// NewServer ServerInit 等价。addr 形如 "0.0.0.0:16010"。
func NewServer(addr string) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &Server{ln: ln, conns: map[net.Conn]struct{}{}}
	s.wg.Add(1)
	go s.acceptLoop()
	return s, nil
}

// Addr 返回实际监听地址。
func (s *Server) Addr() net.Addr { return s.ln.Addr() }

// OnConflict 注册文件占用回调（FMsgFileInUse 转发）。
func (s *Server) OnConflict(f func(file string)) { s.onConflict = f }

// OnFrame 注册原始帧观察面（未知命令/回执帧交给宿主处理）。
func (s *Server) OnFrame(f func(msgID uint32, payload []byte)) { s.onFrame = f }

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns[c] = struct{}{}
		s.mu.Unlock()
		s.wg.Add(1)
		go s.serveConn(c)
	}
}

func (s *Server) serveConn(c net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.conns, c)
		s.mu.Unlock()
		c.Close()
		s.wg.Done()
	}()
	for {
		f, err := ReadFrame(c)
		if err != nil {
			return
		}
		switch f.MsgID {
		case FMsgFileInUse:
			if s.onConflict != nil {
				s.onConflict(string(f.Payload))
			}
		case FMsgEditorCommand, FMsgServerSvnUpdate:
			// 官方 ServerSession::SendAll/SendAllExceptOne：GUI 收到
			// 命令/SVN 通知后广播给其余连接的游戏服。
			s.SendAllExceptOne(f.MsgID, f.Payload, c)
		}
		if s.onFrame != nil {
			s.onFrame(f.MsgID, f.Payload)
		}
	}
}

// SendAll 广播一帧（ServerSession::SendAll 等价）。
func (s *Server) SendAll(msgID uint32, payload []byte) int {
	return s.send(msgID, payload, nil)
}

// SendAllExceptOne 广播但排除一连接（SendAllExceptOne 等价）。
func (s *Server) SendAllExceptOne(msgID uint32, payload []byte, except net.Conn) int {
	return s.send(msgID, payload, except)
}

func (s *Server) send(msgID uint32, payload []byte, except net.Conn) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for c := range s.conns {
		if c == except {
			continue
		}
		if WriteFrame(c, msgID, payload) == nil {
			n++
		}
	}
	return n
}

// Close 关闭监听与全部连接。
func (s *Server) Close() {
	s.ln.Close()
	s.mu.Lock()
	for c := range s.conns {
		c.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// Client 游戏服侧（ClientSession 等价）：外连编辑器 GUI；
// ProcessMsg 线程消费命令队列分派双回调；共享内存读线程轮询
// SNAILEDITOR_SvnUpdateFinish。
type Client struct {
	cfg  Config
	cb   Callbacks
	name string // 共享内存实例名（官方 "SvnUpdateFinish"）

	mu     sync.Mutex
	conn   net.Conn
	queue  []queuedMsg
	closed bool

	wg     sync.WaitGroup
	stopCh chan struct{}
}

type queuedMsg struct {
	typ  int // 官方 sMsg.type：0=延迟SVN更新 1=编辑器命令
	data string
}

// DefaultShareName 官方共享内存实例名（L"SvnUpdateFinish"）。
const DefaultShareName = "SvnUpdateFinish"

// NewClient ClientInit 等价（配置与回调注入）。
func NewClient(cfg Config, cb Callbacks) *Client {
	if cfg.TortoiseProc == "" {
		cfg.TortoiseProc = TortoiseProcPath()
	}
	return &Client{cfg: cfg, cb: cb, name: DefaultShareName, stopCh: make(chan struct{})}
}

// Connect ClientConnect 等价：拨号编辑器 GUI 并起收帧/分发循环。
func (c *Client) Connect(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	if c.cb.OnConnState != nil {
		c.cb.OnConnState(true, conn.RemoteAddr().String())
	}
	c.wg.Add(1)
	go c.readLoop(conn)
	return nil
}

func (c *Client) readLoop(conn net.Conn) {
	defer func() {
		conn.Close()
		if c.cb.OnConnState != nil {
			c.cb.OnConnState(false, "")
		}
		c.wg.Done()
	}()
	for {
		f, err := ReadFrame(conn)
		if err != nil {
			return
		}
		c.dispatch(f)
	}
}

func (c *Client) dispatch(f *Frame) {
	switch f.MsgID {
	case FMsgServerSvnUpdate:
		for _, p := range SplitPaths(string(f.Payload)) {
			if c.IsInWatchPath(p) {
				c.push(0, p) // 官方 type=0 → FuncDelaySvnUpdateCallback
			}
		}
	case FMsgEditorCommand:
		c.push(1, string(f.Payload)) // 官方 type=1 → PushMsg 回调
	case FMsgFileInUse:
		if c.cb.OnConflict != nil {
			c.cb.OnConflict(string(f.Payload))
		}
	}
}

func (c *Client) push(typ int, data string) {
	c.mu.Lock()
	c.queue = append(c.queue, queuedMsg{typ, data})
	c.mu.Unlock()
}

// IsInWatchPath 官方语义：路径落在任一 watch 目录（大小写不敏感）之下。
// watch 列表为空视为全监听。
func (c *Client) IsInWatchPath(path string) bool {
	if len(c.cfg.WatchPaths) == 0 {
		return true
	}
	p := foldSep(lower(path))
	for _, w := range c.cfg.WatchPaths {
		if hasPrefixPath(p, foldSep(lower(w))) {
			return true
		}
	}
	return false
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

func foldSep(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			out[i] = '\\'
		} else {
			out[i] = s[i]
		}
	}
	return string(out)
}

func hasPrefixPath(p, prefix string) bool {
	if len(prefix) == 0 {
		return true
	}
	if !hasPrefix(p, prefix) {
		return false
	}
	// 前缀自身以分隔符结尾时 hasPrefix 即命中目录包含；
	// 否则要求 p 恰等于前缀或后随分隔符（避免 lua2\ 误中 lua\）。
	return prefix[len(prefix)-1] == '\\' || len(p) == len(prefix) || p[len(prefix)] == '\\'
}

func hasPrefix(s, pre string) bool {
	return len(s) >= len(pre) && s[:len(pre)] == pre
}

// ProcessMsg ProcessMsgThread 等价：一次排空命令队列（供测试/宿主手动泵；
// 官方为 100ms 轮询线程）。
func (c *Client) ProcessMsg() {
	for {
		c.mu.Lock()
		if len(c.queue) == 0 {
			c.mu.Unlock()
			return
		}
		m := c.queue[0]
		c.queue = c.queue[1:]
		c.mu.Unlock()
		switch m.typ {
		case 0:
			if c.cb.OnDelaySvnUpdate != nil {
				c.cb.OnDelaySvnUpdate(m.data)
			}
		case 1:
			if c.cb.OnEditorCommand != nil {
				c.cb.OnEditorCommand(m.data)
			}
		}
	}
}

// StartMsgLoop 起 100ms ProcessMsgThread。
func (c *Client) StartMsgLoop() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-c.stopCh:
				return
			case <-t.C:
				c.ProcessMsg()
			}
		}
	}()
}

// StartShareMemLoop ReadSharingMemoryThread 等价：打开
// SNAILEDITOR_<name> 读端（创建+注册），100ms 轮询帧并回调
// OnSvnUpdateComplete；失败返回错误（官方 Trace
// "[ClientSession::ReadSharingMemoryThread] Init SharingMemory failed"）。
// 仅 Windows 可用（命名共享内存）。
func (c *Client) StartShareMemLoop() error {
	mp, err := snailsharemem.OpenRead(c.name)
	if err != nil {
		return fmt.Errorf("Init SharingMemory failed: %w", err)
	}
	id, err := mp.Box().CreateReaderID()
	if err != nil {
		mp.Close()
		return err
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		defer mp.Close()
		for {
			select {
			case <-c.stopCh:
				mp.Box().FreeReaderID(id)
				return
			case <-t.C:
				if b := mp.Box(); b.GetReadable() {
					if data := b.Read(id); data != nil {
						if c.cb.OnSvnUpdateComplete != nil {
							c.cb.OnSvnUpdateComplete(SplitPaths(string(data)))
						}
						b.ReadEnd(id)
					}
				}
			}
		}
	}()
	return nil
}

// Send 原始帧发送。
func (c *Client) Send(msgID uint32, payload []byte) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return io.ErrClosedPipe
	}
	return WriteFrame(conn, msgID, payload)
}

// SendSvnUpdate 通知编辑器 GUI 路径列表已更新（官方 0x3F1 发送方向）。
func (c *Client) SendSvnUpdate(paths []string) error {
	return c.Send(FMsgServerSvnUpdate, []byte(joinPaths(paths)))
}

func joinPaths(ps []string) string {
	out := ""
	for i, p := range ps {
		if i > 0 {
			out += "\n"
		}
		out += NormalizeSvnPath(p)
	}
	return out
}

// Close 停全部循环并断连。
func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	close(c.stopCh)
	if conn != nil {
		conn.Close()
	}
	c.wg.Wait()
}
