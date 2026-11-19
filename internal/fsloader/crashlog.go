package fsloader

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// 崩溃日志命名复刻官方格式串 "ERROR%04d%02d%02d%02d%02d%02d%03d.log"
//（本地时间 年月日时分秒毫秒；Catcher.obj/MiniDump.obj 面）。
// 官方另有 MiniDumpWriteDump 产 DUMP*.dmp（Win32 专有，Go 以
// goroutine 栈文本等价）。
var lastErrorLog atomic.Value // string

// WriteErrorLog 崩溃承接：panic 值 + goroutine 栈写入 ERROR*.log，
// 返回文件名。官方同目录语义（进程工作目录）。
func WriteErrorLog(v any) string {
	name := errorLogName(time.Now())
	f, err := os.Create(name)
	if err != nil {
		// 退化为标准错误（官方 PopupErrorMsgBox 面）。
		fmt.Fprintf(os.Stderr, "FsLoader: 写崩溃日志失败: %v\n原值: %v\n", err, v)
		return ""
	}
	defer f.Close()
	fmt.Fprintf(f, "FsLoader crash at %s\r\n", time.Now().Format("2006-01-02 15:04:05.000"))
	fmt.Fprintf(f, "panic: %v\r\n\r\ngoroutine stack:\r\n%s", v, debug.Stack())
	lastErrorLog.Store(name)
	return name
}

func errorLogName(t time.Time) string {
	return fmt.Sprintf("ERROR%04d%02d%02d%02d%02d%02d%03d.log",
		t.Year(), int(t.Month()), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1e6)
}

func lastErrorLogName() string {
	if s, ok := lastErrorLog.Load().(string); ok {
		return s
	}
	return "ERROR*.log"
}
