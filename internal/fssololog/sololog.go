package fssololog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/reactor"
)

// DefaultMutexName 官方单实例互斥名（CMutex::Exclusive("mutex_fs_solo_log")）。
const DefaultMutexName = "mutex_fs_solo_log"

// ErrExclusive 官方互斥失败（Start 步①）：
// printf("(SoloLog::Start)%s exclusive failed\n") + exit(0)。
var ErrExclusive = errors.New("(SoloLog::Start)" + DefaultMutexName + " exclusive failed")

// logStateNames 官方 GetState 查表（m_nState 0..3）。
var logStateNames = [4]string{"Closed", "Opened", "Paused", "Unknown"}

// checkLogLinesPerTick 官方每 tick 每槽取行上限（100）。
const checkLogLinesPerTick = 100

// autoExitTicks 官方 AutoExit 静态计数窗口：每 10 拍（≈1s）一轮，
// 槽 1 消失 5 轮（5 秒）退出。
const (
	autoExitEvery = 10
	autoExitMax   = 5
)

// Options 是进程装配参数（官方 SoloLog Lua 属性 → ini/旗标投影）。
// 缺省值出处：ctor @0x100552e0 定案（UseThread=0/AutoExit=1/
// ExitCount=-1/State=0）+ lua/sololog/main.lua（LogFileDir=工作路径
// log\、EveryDayAlone=true、Echo 127.0.0.1:15011）。
type Options struct {
	AppName string // 呈现名（缺省 SoloLog）

	EchoAddr string // 空=不开 echo 口（官方 m_EchoAddr 非空才 Start）
	EchoPort int    // 官方缺省 15011

	LogFileDir    string // 官方 main.lua：nx_work_path() .. "log\\"
	EveryDayAlone bool   // 跨天换档（main.lua 置 true）
	AutoExit      bool   // 看门狗（ctor 缺省 true）
	UseThread     bool   // 官方双 EventLoop 开关（Go no-op，见包文档）

	MutexName string // 单实例互斥名（缺省 mutex_fs_solo_log；测试可改）
	ShareName string // 共享内存名（缺省 memory_fs_solo_log_0）
	// Share 注入既有共享内存（测试/进程内整合）；nil = Windows 按名
	// 开真映射，非 Windows 建进程内缓冲。
	Share *LogShareMem

	Logf    func(format string, args ...any)
	Printf  func(format string, args ...any) // 官方 printf 面（AutoExit 文案等）
	Now     func() time.Time                 // 注入时钟（测试）
	WorkDir string                           // 相对 LogFileDir 的锚定目录（缺省当前目录）

	// ManualTick 测试钩子：不启动 0.1s 定时循环，由测试逐拍调
	// TimerCheck 驱动（等价官方 CreateTimer(0.1f) 的手动化）。
	ManualTick bool
}

// Server 复刻 SoloLog 实体（sololog.cpp）的进程装配：
// 单实例互斥 + LogShareMem 消费 + Reactor + EchoConsole +
// 0.1s CheckLog 定时器 + AutoExit 看门狗。
type Server struct {
	opt    Options
	mu     sync.Mutex // 官方单线程语义：tick 与 echo 回调互斥
	logf   func(format string, args ...any)
	printf func(format string, args ...any)
	now    func() time.Time

	r     *reactor.Reactor
	echo  *EchoConsole
	quick *LogShareMem

	fileNames [LogSlotCount]string // m_FileNames[64][0x100]（空串=未开档）
	fileTime  time.Time            // m_FileTime[8]（当前文件组开档时刻）
	tick      int                  // CheckLog 拍数（AutoExit 静态计数）
	exitCount int                  // m_nExitCount（初值 -1）
	state     int                  // m_nState

	releaseMu func() // 互斥体释放

	stopOnce     sync.Once
	stopCh       chan struct{} // 关闭=请求停（信号/AutoExit/Stop 共用，幂等）
	teardownOnce sync.Once
	doneOnce     sync.Once
	doneCh       chan struct{} // Stop 收敛完成
	exitFlag     bool          // GetCore()->Shutdown(1) 等价（AutoExit 触发）
}

// New 建服（不含资源建立，Start 时五步装配）。
func New(opt Options) *Server {
	if opt.AppName == "" {
		opt.AppName = "SoloLog"
	}
	if opt.EchoPort == 0 {
		opt.EchoPort = 15011
	}
	if opt.LogFileDir == "" {
		opt.LogFileDir = "log" + string(os.PathSeparator)
	}
	if opt.MutexName == "" {
		opt.MutexName = DefaultMutexName
	}
	if opt.ShareName == "" {
		opt.ShareName = DefaultLogShareName
	}
	if opt.Logf == nil {
		opt.Logf = func(string, ...any) {}
	}
	if opt.Printf == nil {
		opt.Printf = stdPrintf
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	s := &Server{
		opt:       opt,
		logf:      opt.Logf,
		printf:    opt.Printf,
		now:       opt.Now,
		exitCount: -1,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
	s.r = reactor.New()
	s.r.Logf = func(format string, args ...any) { s.logf("[reactor] "+format, args...) }
	s.echo = NewEchoConsole(s.r, s.echoDispatch)
	return s
}

// Start 官方 SoloLog::Start 五步（任一失败 CORE_TRACE 后返回 false）：
//
//	① CMutex::Exclusive 单实例互斥（失败 printf + exit(0)——本复刻
//	   返回 ErrExclusive，宿主打印同文案后退出码 0）
//	② CLogShareMem new + Start（共享内存）
//	③ CReactor.Start（IOCP）
//	④ m_EchoAddr 非空才 EchoHook.Start(addr, port)
//	⑤ CreateTimer(0.1f) + Port_GetLocalTime(&m_FileTime)；
//	  UseThread 再起专职线程（Go no-op）。置 m_nState=1。
func (s *Server) Start() error {
	// ① 单实例互斥。
	if s.opt.MutexName != "" { // 空串=测试关闭互斥
		release, ok := acquireInstanceMutex(s.opt.MutexName)
		if !ok {
			s.printf("(SoloLog::Start)%s exclusive failed\n", s.opt.MutexName)
			return fmt.Errorf("%w", ErrExclusive)
		}
		s.releaseMu = release
	}
	// ② 共享内存（消费侧）。
	if s.opt.Share != nil {
		s.quick = s.opt.Share
	} else if mem, err := OpenLogShareMem(s.opt.ShareName); err == nil {
		s.quick = mem
		s.logf("共享内存：Windows 命名映射 %s（可与官方进程互通）", s.opt.ShareName)
	} else {
		s.quick = NewLogShareMemInMemory()
		s.logf("共享内存：进程内缓冲（%v）", err)
	}
	s.quick.SetPrintf(s.printf)
	// ③ Reactor。
	if err := s.r.Start(); err != nil {
		return fmt.Errorf("start reactor failed: %w", err)
	}
	// ④ echo 口（m_EchoAddr 非空才开）。
	if s.opt.EchoAddr != "" {
		if err := s.echo.Start(s.opt.EchoAddr, s.opt.EchoPort); err != nil {
			return fmt.Errorf("echo start failed: %w", err)
		}
	}
	// ⑤ 0.1s 定时器 + 开档时刻。
	s.mu.Lock()
	s.fileTime = s.now()
	s.state = 1
	s.mu.Unlock()
	if !s.opt.ManualTick {
		go s.tickLoop()
	}
	return nil
}

// tickLoop 官方 CreateTimer(0.1f, TimerProc) 等价。
func (s *Server) tickLoop() {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			s.TimerCheck()
		case <-s.stopCh:
			return
		}
	}
}

// TimerCheck 官方 TimerProc 体：CheckLog + AutoExit（每 10 拍一轮）。
func (s *Server) TimerCheck() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkLog()
	s.tick++
	if s.tick%autoExitEvery == 0 {
		s.autoExit()
	}
}

// checkLog 官方 SoloLog::CheckLog @0x10056bf0 主循环（§5.1）。
func (s *Server) checkLog() {
	if s.quick == nil {
		return // GetLogMemory() 空则直接返回
	}
	now := s.now()
	for i := 0; i < LogSlotCount; i++ {
		if s.quick.WriteCursor(i) == s.quick.ReadCursor(i) {
			continue // 无积压
		}
		if s.opt.EveryDayAlone && !sameDay(s.fileTime, now) {
			s.fileTime = now
			s.fileNames[i] = ""
		}
		if s.quick.SlotCreateNew(i) != 0 {
			s.quick.ClearSlotCreateNew(i)
			s.fileTime = now
			s.fileNames[i] = ""
		}
		if s.fileNames[i] == "" { // 开档
			ft := s.fileTime
			s.fileNames[i] = fmt.Sprintf("%s%d_%s_%04d%02d%02d%02d%02d%02d.log",
				s.logDir(), i, s.quick.SlotPrefix(i),
				ft.Year(), int(ft.Month()), ft.Day(),
				ft.Hour(), ft.Minute(), ft.Second())
		}
		fp, _ := os.OpenFile(filepath.Clean(s.fileNames[i]),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) // 官方 "ab"
		n := 0
		rc := s.quick.ReadCursor(i)
		wc := s.quick.WriteCursor(i)
		for rc != wc && n < checkLogLinesPerTick {
			line := s.quick.Line(i, rc)
			if line != "" && fp != nil {
				fmt.Fprintf(fp, "[%04d-%02d-%02d %02d:%02d:%02d] %s\n",
					now.Year(), int(now.Month()), now.Day(),
					now.Hour(), now.Minute(), now.Second(), line)
			}
			rc = (rc + 1) & LogLineMask
			n++
		}
		s.quick.SetReadCursor(i, rc)
		if fp != nil {
			fp.Close()
		}
	}
}

// autoExit 官方看门狗（§5.2）：槽 1（FsSoloWorld）注册过又消失
// 5 轮（≈5 秒）→ GetCore()->Shutdown(1)。
func (s *Server) autoExit() {
	if !s.opt.AutoExit {
		return
	}
	if s.quick.GetLogUsed(1) {
		s.exitCount = 0
	} else if s.exitCount >= 0 {
		s.exitCount++
		s.printf("before exit count %d\n", s.exitCount)
	}
	if s.exitCount >= autoExitMax {
		// GetCore()->Shutdown(1)：标记后请求停（Run/宿主收敛退出）。
		s.exitFlag = true
		go s.requestStop()
	}
}

// requestStop 幂等请求停（stopCh 关闭一次）。
func (s *Server) requestStop() {
	s.stopOnce.Do(func() { close(s.stopCh) })
}

// logDir LogFileDir（相对路径锚定 WorkDir；官方 main.lua 拼
// nx_work_path() .. "log\\"——目录串带尾分隔符，文件名直接前缀拼接）。
func (s *Server) logDir() string {
	dir := s.opt.LogFileDir
	if s.opt.WorkDir != "" && !filepath.IsAbs(dir) {
		dir = filepath.Join(s.opt.WorkDir, dir) + string(os.PathSeparator)
	}
	return dir
}

// Run Start 后阻塞直至 ctx 取消/Stop/AutoExit 触发，再 Stop 收敛。
func (s *Server) Run(ctx context.Context) error {
	if err := s.Start(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
	case <-s.stopCh:
	}
	s.Stop()
	return nil
}

// Stopped 返回 Stop 收敛完成通道（宿主帧泵等价阻塞口）。
func (s *Server) Stopped() <-chan struct{} { return s.doneCh }

// Stop 官方 SoloLog::Stop 逆序：定时器 → EchoHook → Reactor →
// CLogShareMem（+互斥体释放；官方 Shut 的 WinSocket 释放为 no-op）。
// 幂等，可与 AutoExit/信号并发调用。
func (s *Server) Stop() {
	s.requestStop()
	s.teardownOnce.Do(func() {
		s.echo.Stop()
		s.r.Stop()
		if s.quick != nil && s.opt.Share == nil {
			s.quick.Close() // 注入的 Share 归属调用方
		}
		s.mu.Lock()
		s.state = 0
		s.mu.Unlock()
		if s.releaseMu != nil {
			s.releaseMu()
			s.releaseMu = nil
		}
		s.doneOnce.Do(func() { close(s.doneCh) })
	})
}

// Exiting 报告 AutoExit 是否已触发（GetCore()->Shutdown 等价呈现）。
func (s *Server) Exiting() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.exitFlag
}

// State 官方 GetState 查表呈现。
func (s *Server) State() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state < 0 || s.state >= len(logStateNames) {
		return logStateNames[3]
	}
	return logStateNames[s.state]
}

// —— 实体方法（官方 mid 方法；method 命令与宿主横幅共用）——

// WriteLine 官方 WriteLine(info)：printf("%s\n", info)（只上本进程控制台）。
func (s *Server) WriteLine(info string) {
	if info == "" {
		return // 官方 assert 非空
	}
	s.printf("%s\n", info)
}

// SendEchoLine 官方 SendEchoLine(info)：m_pEchoHook->SendLine(info)。
func (s *Server) SendEchoLine(info string) {
	if info == "" {
		return
	}
	s.echo.SendLine(info)
}

// SetConsoleColor 官方 SetConsoleColor(color)：app_set_console_color
// （非控制台环境 no-op，见包文档取舍）。
func (s *Server) SetConsoleColor(color string) {
	_ = color
}

// CloseEcho 官方 CloseEcho()：断开当前 echo 连接。
func (s *Server) CloseEcho() { s.echo.Close() }

// sameDay 官方 EveryDayAlone 判定（年/月/日任一变化即换档）。
func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
