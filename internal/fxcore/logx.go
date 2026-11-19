package fxcore

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// LogTool 复刻官方 CLogTool/CLogFile：trace.log 追加面 +
// SaveLogList 统计快照。官方 XLog（共享内存 file-mapping，
// XLOG_FILEMAPPING_NAME，供 game_monitor.exe 读取）为 Win32 专有——
// Go 以同结构文本行等价落 trace.log。
type LogTool struct {
	mu       sync.Mutex
	file     string
	enabled  bool
	interval float64 // 官方日志节拍秒（缺省 20s；0=关闭）
	lastLog  float64
	buf      []string // 环形缓冲（官方 LogPool 面的最近 N 行）
	cap      int
}

// NewLogTool 创建（file=""=trace.log；enabled=DebugLog 开关）。
func NewLogTool(file string, enabled bool) *LogTool {
	if file == "" {
		file = "trace.log"
	}
	return &LogTool{file: file, enabled: enabled, interval: 20, cap: 512}
}

// Trace 复刻 CCore::TraceLog：写一行（CRLF——官方 WriteFile 方言）。
func (l *LogTool) Trace(format string, args ...any) {
	if !l.enabled {
		return
	}
	line := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf = append(l.buf, line)
	if len(l.buf) > l.cap {
		l.buf = l.buf[len(l.buf)-l.cap:]
	}
	f, err := os.OpenFile(l.file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "(%s) %s\r\n",
		time.Now().Format("2006-01-02 15:04:05"), line)
}

// Tick 泵日志节拍（interval 秒一次 Dump；返回是否触发）。
func (l *LogTool) Tick(now float64) bool {
	if !l.enabled || l.interval <= 0 {
		return false
	}
	if now-l.lastLog < l.interval {
		return false
	}
	l.lastLog = now
	return true
}

// SaveLogList 复刻 CCore::SaveLogList：缓冲快照（monitor 拉取面）。
func (l *LogTool) SaveLogList() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.buf, "\r\n")
}

// Stats 复刻官方五钟统计（CCore +0x2d0 区：idle/timer/frame/offset/total）。
type Stats struct {
	Idle   float64 // 泵空闲（Wait/Idle）累计秒
	Timer  float64 // GameTimer.Execute 累计
	Frame  float64 // 实体帧执行累计
	Offset float64 // 尾帧余量执行累计
	Total  float64 // Execute 总耗时
}

// String 官方 DumpFile/SaveLogList 呈现面。
func (s Stats) String() string {
	return fmt.Sprintf("idle=%.3f timer=%.3f frame=%.3f offset=%.3f total=%.3f",
		s.Idle, s.Timer, s.Frame, s.Offset, s.Total)
}
