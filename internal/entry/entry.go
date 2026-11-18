package entry

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// EntryState 官方四态（GameEntryState*.cpp）：
// 0=Unknown（tick 维持）、1=Opened（受理+转发）、2=Paused（拒新连接）、
// 3=Closed（收尾）。
type EntryState int32

const (
	StateUnknown EntryState = 0
	StateOpened  EntryState = 1
	StatePaused  EntryState = 2
	StateClosed  EntryState = 3
)

func (s EntryState) String() string {
	switch s {
	case StateOpened:
		return "opened"
	case StatePaused:
		return "paused"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Options GameEntry 总装参数。
type Options struct {
	EntryID  uint16
	ServerID uint32
	DistrictID uint32
	IDC      uint32
	AppName  string
	// OwnIdent 内网自身标识（官方 SendAdapterCommIdent）
	OwnIdent uint32
	// StatusSink 定时状态行（官方 5 tick 一次 `<info …/>`；nil=仅日志）
	StatusSink func(line string)
	// LicenseInvalid 定期复验钩子（官方 TimerCheck 魔数门槛；nil=放行）
	LicenseInvalid func() bool
	Logf           func(format string, args ...any)
}

// Downlink 是下行面（Gateway 实现；测试可注入桩）。
type Downlink interface {
	ClientSend(ident, serial uint32, payload []byte) error
	ClientClose(ident, serial uint32)
	Shake(ident, serial uint32)
	SendAll(payload []byte)
	SendOnline(payload []byte)
	SendChannel(channel uint32, payload []byte)
	SendSight(ident uint32, sceneID uint32, x, y, radius int32, payload []byte)
	SetClosed(closed bool)
}

// Entry 是 GameEntry 服务总装：状态机 + 内网 16 型分派 + 定时器。
type Entry struct {
	opts     Options
	clients  *Clients
	inner    *Inner
	down     Downlink

	state atomic.Int32
	tick  atomic.Int64
	// 上行统计周期（官方 init_connect 世界窗变化时清零的 8 计数器）
	resetTicker atomic.Int64
	stop        chan struct{}
	stopped     chan struct{}
}

// New 装配服务。down/inner 由宿主创建后注入。
func New(opts Options, clients *Clients, inner *Inner, down Downlink) *Entry {
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	if opts.AppName == "" {
		opts.AppName = fmt.Sprintf("entry_%d_%d", opts.ServerID, opts.EntryID)
	}
	return &Entry{
		opts:    opts,
		clients: clients,
		inner:   inner,
		down:    down,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// Start 进入运行（初始 Unknown；world init_connect 后转 Opened/Paused）。
func (e *Entry) Start() {
	e.state.Store(int32(StateUnknown))
	if e.down != nil {
		e.down.SetClosed(true) // 官方：mass_sock.Closed=true 起步
	}
	go e.run()
}

// Stop 收尾。
func (e *Entry) Stop() {
	close(e.stop)
	<-e.stopped
	e.state.Store(int32(StateClosed))
}

func (e *Entry) run() {
	defer close(e.stopped)
	ticker := time.NewTicker(time.Second / 10) // 官方 Execute 帧率量级
	defer ticker.Stop()
	statTick := 0
	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			e.tick.Add(1)
			e.TimerCheck()
			statTick++
			if statTick >= 10 { // 官方 5 tick 一报；此处 10Hz 主循环折算 ~1s
				statTick = 0
				line := e.StatusLine()
				if e.opts.StatusSink != nil {
					e.opts.StatusSink(line)
				} else {
					e.opts.Logf("%s", line)
				}
			}
		}
	}
}

// State 当前状态。
func (e *Entry) State() EntryState { return EntryState(e.state.Load()) }

// SetState 官方 init_connect 二级分派：flag≠0→Opened(1)，=0→Paused(0)。
func (e *Entry) SetState(open bool) {
	var s EntryState
	if open {
		s = StateOpened
	} else {
		s = StatePaused
	}
	e.state.Store(int32(s))
	if e.down != nil {
		e.down.SetClosed(s != StateOpened)
	}
}

// TimerCheck 官方周期复验（license 门槛 + 状态机 tick）。
// license 失效 → 转 Closed 并拒绝受理（官方 "Not licensed server
// program !!!" 后退出；复刻仅告警一次避免刷屏）。
func (e *Entry) TimerCheck() {
	if e.opts.LicenseInvalid != nil && e.opts.LicenseInvalid() {
		if EntryState(e.state.Load()) != StateClosed {
			e.opts.Logf("Not licensed server program !!!")
		}
		e.state.Store(int32(StateClosed))
		if e.down != nil {
			e.down.SetClosed(true)
		}
	}
}

// StatusLine 官方 `<info state="%s" connect="%d" inuse="%d" tick="%d" />`。
func (e *Entry) StatusLine() string {
	cc, iu, _ := e.clients.Stats()
	return fmt.Sprintf(`<info state="%s" connect="%d" inuse="%d" tick="%d" />`,
		e.State(), cc, iu, e.tick.Load())
}

// ProcessForward 内网 16 型分派（CAdapterHook::ProcessMsg 复刻主体）。
// 返回错误仅用于日志（官方断言路径）。
func (e *Entry) ProcessForward(src uint32, typ uint32, p []byte) error {
	switch typ {
	case ForwInitConnect: // ==4
		if len(p) < 4 {
			return fmt.Errorf("len == sizeof(entry_forw_init_connect_t)")
		}
		e.SetState(binary.LittleEndian.Uint32(p) != 0)
		return nil

	case ForwBreakClient: // ==8
		if len(p) < 8 {
			return fmt.Errorf("len == sizeof(entry_forw_break_client_t)")
		}
		if e.down != nil {
			e.down.ClientClose(le32(p, 0), le32(p, 4))
		}
		return nil

	case ForwSetInuse: // ==9
		if len(p) < 9 {
			return fmt.Errorf("len == sizeof(entry_forw_set_client_inuse_t)")
		}
		return e.clients.SetInuse(le32(p, 0), le32(p, 4), p[8] != 0)

	case ForwSetOnline: // ==16
		if len(p) < 16 {
			return fmt.Errorf("len == sizeof(entry_forw_set_client_online_t)")
		}
		return e.clients.SetOnline(le32(p, 0), le32(p, 4), int32(le32(p, 8)), le32(p, 12))

	case ForwSetPosi: // ==40
		if len(p) < 40 {
			return fmt.Errorf("len == sizeof(entry_forw_set_client_posi_t)")
		}
		return e.clients.SetPosi(le32(p, 0), le32(p, 4), le32(p, 12),
			le32(p, 24), le32(p, 28), le32(p, 32), le32(p, 36),
			int32(le32(p, 16)), int32(le32(p, 20)))

	case ForwSetMove: // >=12 + n*16
		if len(p) < 12 {
			return fmt.Errorf("len >= sizeof(entry_forw_set_client_move_t)")
		}
		n := int(le32(p, 8))
		if len(p) < 12+n*16 {
			return fmt.Errorf("len == sizeof(entry_forw_set_client_move_t) + nCount*16")
		}
		entries := make([][4]uint32, n)
		for i := 0; i < n; i++ {
			o := 12 + i*16
			entries[i] = [4]uint32{le32(p, o), le32(p, o+4), le32(p, o+8), le32(p, o+12)}
		}
		e.clients.Move(le32(p, 0), entries)
		return nil

	case ForwSendSight: // >=28
		if len(p) < 28 {
			return fmt.Errorf("len >= sizeof(entry_forw_send_to_sight_t)")
		}
		if e.down != nil {
			e.down.SendSight(le32(p, 0), le32(p, 12),
				int32(le32(p, 16)), int32(le32(p, 20)), int32(le32(p, 24)), p[28:])
		}
		return nil

	case ForwSendClient: // >=8
		if len(p) < 8 {
			return fmt.Errorf("len >= sizeof(entry_forw_send_to_client_t)")
		}
		if e.down != nil {
			return e.down.ClientSend(le32(p, 0), le32(p, 4), p[8:])
		}
		return nil

	case ForwSendAll:
		if e.down != nil {
			e.down.SendAll(p)
		}
		return nil

	case ForwSendMulti:
		if e.down != nil {
			e.down.SendAll(p)
		}
		return nil

	case ForwShakeClient: // ==8
		if len(p) < 8 {
			return fmt.Errorf("len == sizeof(entry_forw_shake_client_t)")
		}
		if e.down != nil {
			e.down.Shake(le32(p, 0), le32(p, 4))
		}
		return nil

	case ForwChannelAdd: // ==12
		if len(p) < 12 {
			return fmt.Errorf("len == sizeof(entry_forw_channel_add_t)")
		}
		return e.clients.ChannelAdd(le32(p, 0), le32(p, 4), le32(p, 8))

	case ForwChannelRm: // ==12
		if len(p) < 12 {
			return fmt.Errorf("len == sizeof(entry_forw_channel_remove_t)")
		}
		return e.clients.ChannelRemove(le32(p, 0), le32(p, 4), le32(p, 8))

	case ForwClearChan: // ==4
		if len(p) < 4 {
			return fmt.Errorf("len == sizeof(entry_forw_clear_channel_t)")
		}
		e.clients.ClearChannel(le32(p, 0))
		return nil

	case ForwSendChannel: // >=4
		if len(p) < 4 {
			return fmt.Errorf("len >= sizeof(entry_forw_send_to_channel_t)")
		}
		if e.down != nil {
			e.down.SendChannel(le32(p, 0), p[4:])
		}
		return nil

	case ForwSendOnline:
		if e.down != nil {
			e.down.SendOnline(p)
		}
		return nil

	default:
		return fmt.Errorf("(ProcessMsg)msg type error: %d", typ)
	}
}

// BroadcastInfo 复刻 BroadcastEntryInfo：UDP 0x48 字节 0x3001 信息包。
func (e *Entry) BroadcastInfo(name1, name2 string, fields [4]uint32, tail uint32) []byte {
	buf := make([]byte, 0x48)
	binary.LittleEndian.PutUint32(buf[0:], 1)      // 版本
	binary.LittleEndian.PutUint32(buf[4:], 0x3001) // 消息号
	copy(buf[8:], name1)
	copy(buf[0x2c:], name2)
	binary.LittleEndian.PutUint32(buf[0x18:], fields[0])
	binary.LittleEndian.PutUint32(buf[0x1c:], fields[1])
	binary.LittleEndian.PutUint32(buf[0x20:], fields[2])
	binary.LittleEndian.PutUint32(buf[0x24:], fields[3])
	binary.LittleEndian.PutUint32(buf[0x40:], tail)
	cc, _, _ := e.clients.Stats()
	binary.LittleEndian.PutUint32(buf[0x44:], uint32(cc))
	return buf
}

// EchoWelcome/ EchoResponse 官方维护面文案。
func EchoWelcome(serverID, entryID uint32) string {
	return fmt.Sprintf("welcome server %d entry %d\r", serverID, entryID)
}

func EchoResponse(info string) string {
	return ">" + info + "\r"
}

// le32 便捷小端读取。
func le32(p []byte, off int) uint32 { return binary.LittleEndian.Uint32(p[off:]) }

// TrimEchoLine 维护面行规整（\r\n 截断）。
func TrimEchoLine(line string) string { return strings.TrimRight(line, "\r\n") }
