package fssoloentry

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/entry"
)

// solo 内网源校验白名单（ProcessInnerMsg 入口三段门）：
// {0,1,2} ∪ [0x18,0x2F]——比网关版 {0}∪[0x18,0x27] 放宽（world 备份位
// 与 24 个 member 位）。
func SoloSourceCheck(src uint32) bool {
	return src <= 2 || (src >= 0x18 && src <= 0x2F)
}

// SoloOwnIdent 复刻 SetCommIdent(EntryId+8)：solo 接入自身内网标识。
func SoloOwnIdent(entryID uint16) uint32 { return uint32(entryID) + 8 }

// solo member 空间：0x18..0x2F 共 24 位。
const (
	SoloMemberBase  = 0x18
	SoloMemberCount = 24
)

// Downlink 是下行面（entry.Gateway 实现）。
type Downlink = entry.Downlink

// Options SoloEntry 服务总装参数。
type Options struct {
	IDC        uint32
	DistrictID uint32
	ServerID   uint32
	EntryID    uint16
	MaxConnects int

	InnerAddr string // 广播包 inner_addr 域
	InnerPort int    // 广播包 inner_port 域
	OuterAddr string // 广播包 outer_addr 域
	OuterPort int    // 广播包 outer_port 域（网关监听口，客户端接入）

	AppName string // 缺省 entry_<sid>_<eid>

	// License 启动装载的授权（nil=未授权：TimerCheck 每拍告警且不广播）
	License *License
	// StatusSink 状态行接收（官方 CAppRecord::SetAppInfo；nil=仅日志）
	StatusSink func(line string)
	// BroadcastSink 广播包接收（官方 UDP BroadSock；nil=丢弃）
	BroadcastSink func(pkt []byte)
	// Quit 入站 QuitFrame 时的退出钩子（官方 Core->Quit(1)）
	Quit func()

	Logf func(format string, args ...any)
	// ManualTick 测试钩子：不启动 1s 定时循环，由测试逐拍调 TimerCheck
	ManualTick bool
}

// Server 复刻 SoloEntry 实体（soloentry.cpp）：四态状态机 + 内网 15 型
// 分派 + TimerCheck（license 复验/状态行/广播）+ 世界窗联动。
type Server struct {
	opt   Options
	logf  func(format string, args ...any)
	down  Downlink
	inner *entry.Inner

	clients *entry.Clients

	mu    sync.Mutex
	state atomic.Int32
	tick  atomic.Int64
	// license 身份域镜像（官方 this+0x1b4..+0x1d0；世界窗越界时清零）
	fields     [8]uint32
	windowT0   uint32 // 官方 this+0x1d4（外部形态时间窗下界；内部形态恒 0）
	windowT1   uint32 // 官方 this+0x1d8
	licensedSpam bool // 告警一次（官方每拍 WriteLine；复刻防刷屏）

	// sceneOf 客户端场景表（set_client_online 第 4 域 → SceneSend 过滤）
	sceneOf map[uint32]uint32

	stop     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
}

// New 装配服务。down/inner 由宿主创建后注入（down 可为 nil，仅测状态机）。
func New(opts Options, clients *entry.Clients, inner *entry.Inner, down Downlink) *Server {
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	if opts.AppName == "" {
		opts.AppName = fmt.Sprintf("entry_%d_%d", opts.ServerID, opts.EntryID)
	}
	s := &Server{
		opt:     opts,
		logf:    opts.Logf,
		down:    down,
		inner:   inner,
		clients: clients,
		sceneOf: make(map[uint32]uint32),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	if opts.License != nil {
		s.fields = opts.License.Fields
		if !opts.License.Internal {
			s.windowT0 = opts.License.T0
			s.windowT1 = opts.License.T1
		}
	}
	return s
}

// Start 进入运行（初始 Unknown；world init_connect 后转 Opened/Paused）。
func (s *Server) Start() {
	s.state.Store(int32(entry.StateUnknown))
	if s.down != nil {
		s.down.SetClosed(true) // 官方：COuterHook::SetClosed(true) 起步
	}
	if s.opt.ManualTick {
		return
	}
	go s.run()
	s.startDiscoveryWatch()
}

// Stop 收尾（幂等：合一进程内 Quit 钩与宿主停机序可能并发重入）。
func (s *Server) Stop() {
	s.state.Store(int32(entry.StateClosed))
	if s.opt.ManualTick {
		return // 手动拍模式无后台循环
	}
	s.stopOnce.Do(func() { close(s.stop) })
	<-s.stopped
}

func (s *Server) run() {
	defer close(s.stopped)
	ticker := time.NewTicker(time.Second) // 官方 CreateTimer(1.0)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.TimerCheck()
		}
	}
}

// State 当前状态。
func (s *Server) State() entry.EntryState { return entry.EntryState(s.state.Load()) }

// EntryStateMachine 官方四态事件机（事件 0=tick/1=open/2=pause/3=close；
// 未知态：1→Opened 开闸，2→"connect server already closed"，0 静默；
// 其余态由对应 EntryState* 处理，非法输入 "(EntryState%s)error input"）。
func (s *Server) EntryStateMachine(event uint32) bool {
	switch s.State() {
	case entry.StateUnknown:
		switch event {
		case 1:
			if s.down != nil {
				s.down.SetClosed(false)
			}
			s.logf("connect server opened")
			s.state.Store(int32(entry.StateOpened))
			return true
		case 2:
			s.logf("connect server already closed")
			return true
		case 0:
			return true
		default:
			s.logf("(SoloEntry::EntryStateUnknown)error input %d", event)
			return false
		}
	case entry.StateOpened:
		return s.stateOpened(event)
	case entry.StatePaused:
		return s.statePaused(event)
	case entry.StateClosed:
		return s.stateClosed(event)
	default:
		return false
	}
}

func (s *Server) stateOpened(event uint32) bool {
	switch event {
	case 1:
		s.logf("connect server already opened")
		return true
	case 2:
		if s.down != nil {
			s.down.SetClosed(true)
		}
		s.logf("connect server paused")
		s.state.Store(int32(entry.StatePaused))
		return true
	case 3:
		if s.down != nil {
			s.down.SetClosed(true)
		}
		s.logf("connect server closed")
		s.state.Store(int32(entry.StateClosed))
		return true
	case 0:
		return true
	default:
		s.logf("(SoloEntry::EntryStateOpened)error input %d", event)
		return false
	}
}

func (s *Server) statePaused(event uint32) bool {
	switch event {
	case 1:
		if s.down != nil {
			s.down.SetClosed(false)
		}
		s.logf("connect server opened")
		s.state.Store(int32(entry.StateOpened))
		return true
	case 2:
		s.logf("connect server already paused")
		return true
	case 3:
		s.logf("connect server closed")
		s.state.Store(int32(entry.StateClosed))
		return true
	case 0:
		return true
	default:
		s.logf("(SoloEntry::EntryStatePaused)error input %d", event)
		return false
	}
}

func (s *Server) stateClosed(event uint32) bool {
	switch event {
	case 1, 2:
		s.logf("connect server already closed")
		return true
	case 0:
		return true
	default:
		s.logf("(SoloEntry::EntryStateClosed)error input %d", event)
		return false
	}
}

// TimerCheck 官方 1s 定时器：
//
//	状态行（第 6 拍一次，AppRecord 通道）→ license 复验（魔数对命中才
//	BroadcastEntryInfo；第 2 域失配 WriteLine "not licensed server
//	program!!!"——官方非致命仅告警）→ EntryStateMachine(0) tick。
func (s *Server) TimerCheck() {
	s.tick.Add(1)
	if s.tick.Load()%6 == 0 && s.opt.StatusSink != nil {
		s.opt.StatusSink(s.StatusLine())
	}
	s.mu.Lock()
	f1, f2 := s.fields[0], s.fields[1]
	s.mu.Unlock()
	if f1 == OfficialMagic1 {
		pkt := s.BroadcastInfo()
		if s.opt.BroadcastSink != nil {
			s.opt.BroadcastSink(pkt)
		}
	}
	if f2 != OfficialMagic2 {
		if !s.licensedSpam {
			s.licensedSpam = true
			s.logf("not licensed server program!!!")
		}
	}
	s.EntryStateMachine(0)
}

// StatusLine 官方 `<info state="%s" connect="%d" inuse="%d" tick="%d" />`。
func (s *Server) StatusLine() string {
	cc, iu, _ := s.clients.Stats()
	return fmt.Sprintf(`<info state="%s" connect="%d" inuse="%d" tick="%d" />`,
		s.State(), cc, iu, s.tick.Load())
}

// BroadcastInfo 复刻 BroadcastEntryInfo：0x48 字节 UDP 0x3001 信息包
// {ver=1, msg=0x3001, inner_addr[16], inner_port, district, server,
// entry, idc, outer_addr[16], outer_port, max_connects, connect_count}。
// 尾三域与 fssololister.ParseEntryInfo 偏移互证（0x3C=Port/0x40=Max/
// 0x44=Cur——缺 outer_port 会让列表服把 max_connects 当端口下发）。
func (s *Server) BroadcastInfo() []byte {
	buf := make([]byte, 0x48)
	binary.LittleEndian.PutUint32(buf[0:], 1)
	binary.LittleEndian.PutUint32(buf[4:], 0x3001)
	putStr16(buf[8:], s.opt.InnerAddr)
	binary.LittleEndian.PutUint32(buf[0x18:], uint32(s.opt.InnerPort))
	binary.LittleEndian.PutUint32(buf[0x1c:], s.opt.DistrictID)
	binary.LittleEndian.PutUint32(buf[0x20:], s.opt.ServerID)
	binary.LittleEndian.PutUint32(buf[0x24:], uint32(s.opt.EntryID))
	binary.LittleEndian.PutUint32(buf[0x28:], s.opt.IDC)
	putStr16(buf[0x2c:], s.opt.OuterAddr)
	binary.LittleEndian.PutUint32(buf[0x3c:], uint32(s.opt.OuterPort))
	binary.LittleEndian.PutUint32(buf[0x40:], uint32(s.opt.MaxConnects))
	cc, _, _ := s.clients.Stats()
	binary.LittleEndian.PutUint32(buf[0x44:], uint32(cc))
	return buf
}

func putStr16(dst []byte, s string) {
	if len(s) > 15 {
		s = s[:15] // CopyString 截断 + NUL
	}
	copy(dst, s)
}

// Fields 授权身份域镜像（测试与运维观察口）。
func (s *Server) Fields() [8]uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fields
}
