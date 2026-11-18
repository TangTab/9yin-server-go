package fsdumper

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// :4006 行缓冲上限：官方 client 结构 0x102C，行缓冲 client+8 起
// 0x1000 字节、长度计数 client+0x1008，超限丢弃（ReceiveClientMsg 实证）。
const outerLineMax = 4096

// GameDumper 四态（GameDumperState.obj，this+0x28）。
const (
	stateUnknown = iota // 0
	stateOpened         // 1
	stateClosing        // 2
	stateClosed         // 3
)

// 状态事件（DumperStateMachine msg 参数语义）。
const (
	stateEventClient = 0 // 客户端接入（Opened 态消费）
	stateEventOpen   = 1 // 服务就绪（Unknown→Opened）
	stateEventClose  = 2 // 开始关停（Opened→Closing）
)

var stateNames = map[int]string{
	stateUnknown: "unknown", stateOpened: "opened",
	stateClosing: "closing", stateClosed: "closed",
}

// Callee 对应 FsDumpLogic 的 dump_callee 回调面（官方为 LogicLoader
// 装载的 DLL；Go 为原生装配）。
type Callee interface {
	OnDumperStart(k *Kernel)
	OnFrameExecute(k *Kernel)
	OnServerClose(k *Kernel)
	OnServerConnected(k *Kernel, clientID int)
	OnRecvGameServerMsg(k *Kernel, clientID int, msg *textVarList)
}

// outerClient :4006 单客户端（官方 client 结构的 Go 投影）。
type outerClient struct {
	id   int
	conn net.Conn
	name string
}

// Server 对应 FsDumper.dll 的 GameDumper 实体+Outer 监听+状态机+主泵。
type Server struct {
	Addr string // 监听地址（[Outer] Addr，空则全部接口）
	Port int    // [Outer] Port；-1 不开外口，0 临时端口（测试），>0 固定

	Kernel *Kernel
	Callee Callee

	Logf func(format string, args ...any) // 官方 console:WriteLine 同位

	mu      sync.Mutex
	ln      net.Listener
	clients map[int]*outerClient
	nextID  int
	state   int
	events  chan int
	closed  chan struct{}
	wg      sync.WaitGroup
	stopOne sync.Once
}

// NewServer 创建服务；callee 必填（业务回调面，通常为 DefaultCallee）。
func NewServer(addr string, port int, callee Callee, logf func(string, ...any)) *Server {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	s := &Server{
		Addr:    addr,
		Port:    port,
		Callee:  callee,
		Logf:    logf,
		clients: map[int]*outerClient{},
		events:  make(chan int, 64),
		closed:  make(chan struct{}),
		state:   stateUnknown,
	}
	s.Kernel = newKernel(s)
	return s
}

// Start 对应 GameDumper::Start：外监听 + 状态 Open 事件 + 主泵 +
// callee 启动（顺序按官方：监听→定时器→业务）。
func (s *Server) Start() error {
	if s.Port >= 0 {
		ln, err := net.Listen("tcp", net.JoinHostPort(s.Addr, fmt.Sprintf("%d", s.Port)))
		if err != nil {
			return fmt.Errorf("fsdumper: outer listen %s:%d: %w", s.Addr, s.Port, err)
		}
		s.mu.Lock()
		s.ln = ln
		s.mu.Unlock()
		s.wg.Add(1)
		go s.acceptLoop(ln)
		s.Logf("Outer socket %s:%d", s.Addr, s.Port)
	}

	// 官方：全部就绪后 Unknown --(msg=1)--> Opened。
	s.postEvent(stateEventOpen)

	s.wg.Add(1)
	go s.pumpLoop()

	s.Callee.OnDumperStart(s.Kernel)
	s.Logf("Game dumper server started")
	return nil
}

// Stop 对应 GameDumper::Stop：Opened --(msg=2)--> Closing，断外口，
// 排空后 Closed。
func (s *Server) Stop() {
	s.stopOne.Do(func() {
		s.postEvent(stateEventClose)
		s.mu.Lock()
		if s.ln != nil {
			s.ln.Close()
		}
		for _, c := range s.clients {
			c.conn.Close()
		}
		s.mu.Unlock()
		close(s.closed)
	})
	s.wg.Wait()
}

func (s *Server) postEvent(ev int) {
	select {
	case s.events <- ev:
	default: // 官方状态事件不排队堆积；丢弃重复事件无害
	}
}

func (s *Server) acceptLoop(ln net.Listener) {
	defer s.wg.Done()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
			}
			s.Logf("(GameDumper::Accept) accept error: %v", err)
			return
		}
		s.mu.Lock()
		if s.state != stateOpened {
			s.mu.Unlock()
			conn.Close()
			continue
		}
		id := s.nextID
		s.nextID++
		cl := &outerClient{id: id, conn: conn}
		s.clients[id] = cl
		s.mu.Unlock()

		s.LogClient(id, conn.RemoteAddr().String(), true)
		// 官方 Opened 态 msg=0：先 callee->OnServerConnected 再 AddClient；
		// Go 此处已入册，顺序等效。
		s.Callee.OnServerConnected(s.Kernel, id)

		s.wg.Add(1)
		go s.serveConn(cl)
	}
}

// serveConn 官方 ReceiveClientMsg：\r/\n 切行、4096 上限、逐行
// ReceiveCommand（text_decode → callee.OnRecvGameServerMsg）。
func (s *Server) serveConn(cl *outerClient) {
	defer s.wg.Done()
	defer func() {
		s.mu.Lock()
		delete(s.clients, cl.id)
		s.mu.Unlock()
		cl.conn.Close()
		s.LogClient(cl.id, "", false)
	}()
	br := bufio.NewReaderSize(cl.conn, 8192)
	line := make([]byte, 0, outerLineMax)
	for {
		b, err := br.ReadByte()
		if err != nil {
			return
		}
		if debugTrace {
			s.Logf("[trace] conn#%d byte %q (%#x) lineLen=%d", cl.id, string(b), b, len(line))
		}
		if b == '\n' || b == '\r' {
			if len(line) > 0 {
				s.receiveCommand(cl.id, string(line))
				line = line[:0]
			}
			continue
		}
		if len(line) < outerLineMax {
			line = append(line, b)
		} // 超限丢弃直至行终止（官方同款）
	}
}

// receiveCommand 官方 0x100758F0：text_decode_msg → callee。
func (s *Server) receiveCommand(clientID int, line string) {
	if debugTrace {
		s.Logf("[trace] receiveCommand #%d %q", clientID, line)
	}
	msg, err := DecodeTextVarList(line)
	if err != nil {
		s.Logf("(GameDumper::ReceiveCommand)decode error: %v", err)
		return
	}
	s.Callee.OnRecvGameServerMsg(s.Kernel, clientID, msg)
}

// SendTo 官方 SendClientMessage/SendServerMsg：按客户端 id（或 -1 广播）
// 写一行。
func (s *Server) SendTo(clientID int, payload []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if clientID >= 0 {
		cl, ok := s.clients[clientID]
		if !ok {
			return false
		}
		_, err := cl.conn.Write(payload)
		return err == nil
	}
	ok := len(s.clients) > 0
	for _, cl := range s.clients {
		if _, err := cl.conn.Write(payload); err != nil {
			ok = false
		}
	}
	return ok
}

// NotifyEcho 维护口日志镜像（官方 SendResponse 走 EchoSock console）。
func (s *Server) NotifyEcho(line string) { s.Logf("%s", line) }

// LogClient 连接开合日志（官方 echo 侧 WriteLine 同义）。
func (s *Server) LogClient(id int, addr string, open bool) {
	if open {
		s.Logf("Outer client #%d %s connected", id, addr)
	} else {
		s.Logf("Outer client #%d closed", id)
	}
}

// pumpLoop 官方 TimerCheck 主泵：状态事件队列 + CDumpTimer 节拍 +
// OnFrameExecute 帧（1000ms 节流窗语义 → 1s tick）。
func (s *Server) pumpLoop() {
	defer s.wg.Done()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-s.closed:
			s.stateMachine(stateEventClose)
			return
		case ev := <-s.events:
			s.stateMachine(ev)
		case <-tick.C:
			s.Kernel.runDue()
			s.Callee.OnFrameExecute(s.Kernel)
		}
	}
}

// stateMachine 官方 DumperStateMachine 四态分派（jmp 表 0x1007961C）。
func (s *Server) stateMachine(ev int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.state {
	case stateUnknown:
		if ev == stateEventOpen {
			s.state = stateOpened
		} else if ev != stateEventClient {
			s.Logf("(DumperStateUnknown)unexpected event %d", ev)
		}
	case stateOpened:
		switch ev {
		case stateEventClose:
			s.state = stateClosing
		case stateEventOpen:
			// 官方 Opened 态对 msg=1 无处理（断言路径）
		default:
			s.Logf("(DumperStateOpened)unexpected event %d", ev)
		}
	case stateClosing:
		if ev == stateEventClient || ev == stateEventClose {
			s.state = stateClosed
			if s.Callee != nil {
				go s.Callee.OnServerClose(s.Kernel)
			}
		}
	case stateClosed:
		// 静默
	}
}

// State 当前状态名（GameDumper::GetState）。
func (s *Server) State() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return stateNames[s.state]
}

// ClientIDs 在册客户端 id 列表（测试/运维面）。
func (s *Server) ClientIDs() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]int, 0, len(s.clients))
	for id := range s.clients {
		ids = append(ids, id)
	}
	return ids
}

// debugTrace 临时诊断开关（TODO: 定位后移除）。
var debugTrace = os.Getenv("NINEYIN_DUMP_TRACE") != ""
