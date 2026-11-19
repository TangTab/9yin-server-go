package fssololog

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// discardPrintf 静音官方 printf 面（个别用例换成捕获）。
func discardPrintf(string, ...any) {}

// capturePrintf 捕获 printf 面。
func capturePrintf() (*[]string, func(format string, args ...any)) {
	var lines []string
	return &lines, func(format string, args ...any) {
		lines = append(lines, fmt.Sprintf(format, args...))
	}
}

// newTestServer 建 ManualTick 测试服（无 echo、注入共享内存、静音）。
func newTestServer(t *testing.T, share *LogShareMem, now func() time.Time) *Server {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "log") + string(os.PathSeparator)
	if err := os.MkdirAll(filepath.Clean(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	s := New(Options{
		MutexName:     "mutex_fs_solo_log_test_" + t.Name(),
		Share:         share,
		LogFileDir:    dir,
		Now:           now,
		Logf:          func(string, ...any) {},
		Printf:        discardPrintf,
		ManualTick:    true,
		AutoExit:      true,
		EveryDayAlone: true,
	})
	t.Cleanup(s.Stop)
	return s
}

func TestLogShareMemSlotProtocol(t *testing.T) {
	prod := NewLogShareMemInMemory()
	prod.SetPrintf(discardPrintf)
	cons := prod.CloneView() // 另一进程视角

	prod.SetAppId(1)
	prod.SetLogPrefix("FsSoloWorld")
	if !cons.GetLogUsed(1) {
		t.Fatal("槽 1 注册后应在用")
	}
	if got := cons.SlotPrefix(1); got != "FsSoloWorld" {
		t.Fatalf("prefix = %q", got)
	}
	if !prod.TraceLog("hello") || !prod.TraceLog("world") {
		t.Fatal("环未满 TraceLog 应成功")
	}
	if cons.ReadCursor(1) != 0 || cons.WriteCursor(1) != 2 {
		t.Fatalf("游标 w=%d r=%d，期望 2/0", cons.WriteCursor(1), cons.ReadCursor(1))
	}
	if got := cons.Line(1, 0); got != "hello" {
		t.Fatalf("行 0 = %q", got)
	}
	prod.ClearLogUsed()
	if cons.GetLogUsed(1) {
		t.Fatal("注销后槽应不在用")
	}
}

func TestTraceLogRingFull(t *testing.T) {
	prod := NewLogShareMemInMemory()
	prod.SetPrintf(discardPrintf)
	prod.SetAppId(2)
	prod.SetLogPrefix("FsSoloPublic")
	// 环容量 128 → 未消费时最多 127 行待取，第 128 次写应丢行。
	for i := 0; i < 127; i++ {
		if !prod.TraceLog(fmt.Sprintf("line%d", i)) {
			t.Fatalf("第 %d 行不应丢", i)
		}
	}
	if prod.TraceLog("overflow") {
		t.Fatal("环满应丢行返回 false")
	}
	cons := prod.CloneView()
	cons.SetReadCursor(2, 1) // 消费 1 行
	if !prod.TraceLog("after-consume") {
		t.Fatal("消费后应可再写")
	}
	if prod.TraceLog("overflow-again") {
		t.Fatal("再满应再丢")
	}
}

func TestCheckLogFileNameAndLineFormat(t *testing.T) {
	share := NewLogShareMemInMemory()
	share.SetPrintf(discardPrintf)
	base := time.Date(2013, 7, 5, 10, 53, 22, 0, time.Local)
	clock := base
	s := newTestServer(t, share, func() time.Time { return clock })
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	prod := share.CloneView()
	prod.SetPrintf(discardPrintf)
	prod.SetAppId(1)
	prod.SetLogPrefix("FsSoloWorld")
	prod.TraceLog("core startup")
	prod.TraceLog("scene opened")

	s.TimerCheck()

	dir := s.logDir()
	wantName := fmt.Sprintf("%s1_FsSoloWorld_%04d%02d%02d%02d%02d%02d.log",
		dir, base.Year(), int(base.Month()), base.Day(),
		base.Hour(), base.Minute(), base.Second())
	data, err := os.ReadFile(wantName)
	if err != nil {
		t.Fatalf("读官方命名日志 %s: %v", wantName, err)
	}
	want := fmt.Sprintf("[%04d-%02d-%02d %02d:%02d:%02d] core startup\n"+
		"[%04d-%02d-%02d %02d:%02d:%02d] scene opened\n",
		2013, 7, 5, 10, 53, 22, 2013, 7, 5, 10, 53, 22)
	if string(data) != want {
		t.Fatalf("行格式/时间戳（落盘时刻）不符：\n得 %q\n望 %q", data, want)
	}
	if share.ReadCursor(1) != share.WriteCursor(1) {
		t.Fatal("消费后游标应追平")
	}
}

func TestCheckLogCap100PerTick(t *testing.T) {
	share := NewLogShareMemInMemory()
	share.SetPrintf(discardPrintf)
	base := time.Date(2013, 7, 5, 10, 53, 22, 0, time.Local)
	clock := base
	s := newTestServer(t, share, func() time.Time { return clock })
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	prod := share.CloneView()
	prod.SetPrintf(discardPrintf)
	prod.SetAppId(1)
	prod.SetLogPrefix("FsSoloWorld")
	for i := 0; i < 127; i++ { // 环待取上限
		prod.TraceLog(fmt.Sprintf("l%03d", i))
	}

	s.TimerCheck() // 每 tick 每槽至多 100 行
	if got := share.ReadCursor(1); got != 100 {
		t.Fatalf("首拍应取 100 行，read=%d", got)
	}
	s.TimerCheck() // 同档续写（同名文件 append）
	if got := share.ReadCursor(1); got != share.WriteCursor(1) {
		t.Fatalf("次拍应取完，read=%d w=%d", got, share.WriteCursor(1))
	}
	matches, _ := filepath.Glob(s.logDir() + "1_FsSoloWorld_*.log")
	if len(matches) != 1 {
		t.Fatalf("两拍应同档 1 个文件，得 %v", matches)
	}
	data, _ := os.ReadFile(matches[0])
	if n := strings.Count(string(data), "\n"); n != 127 {
		t.Fatalf("总行数 = %d，期望 127", n)
	}
}

func TestEveryDayAloneRollover(t *testing.T) {
	share := NewLogShareMemInMemory()
	share.SetPrintf(discardPrintf)
	day1 := time.Date(2013, 7, 5, 23, 59, 50, 0, time.Local)
	clock := day1
	s := newTestServer(t, share, func() time.Time { return clock })
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	prod := share.CloneView()
	prod.SetPrintf(discardPrintf)
	prod.SetAppId(1)
	prod.SetLogPrefix("FsSoloWorld")
	prod.TraceLog("day1")
	s.TimerCheck()

	clock = day1.Add(2 * time.Hour) // 跨天
	prod.TraceLog("day2")
	s.TimerCheck()

	matches, _ := filepath.Glob(s.logDir() + "1_FsSoloWorld_*.log")
	if len(matches) != 2 {
		t.Fatalf("跨天应换新档 2 个文件，得 %v", matches)
	}
}

func TestCreateNewRollover(t *testing.T) {
	share := NewLogShareMemInMemory()
	share.SetPrintf(discardPrintf)
	base := time.Date(2013, 7, 5, 10, 53, 22, 0, time.Local)
	clock := base
	s := newTestServer(t, share, func() time.Time { return clock })
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	prod := share.CloneView()
	prod.SetPrintf(discardPrintf)
	prod.SetAppId(0x38)
	prod.SetLogPrefix("FsSoloGlobal")
	prod.TraceLog("first")
	s.TimerCheck()

	clock = base.Add(3 * time.Second) // 错开文件名秒段
	prod.SetCreateNew(1)              // 生产者请求换档（Global 每次启动）
	prod.TraceLog("second")
	s.TimerCheck()

	if share.SlotCreateNew(0x38) != 0 {
		t.Fatal("消费侧应清 create_new 位")
	}
	matches, _ := filepath.Glob(s.logDir() + "56_FsSoloGlobal_*.log")
	if len(matches) != 2 {
		t.Fatalf("create_new 应换新档 2 个文件，得 %v", matches)
	}
}

func TestAutoExitWatchdog(t *testing.T) {
	share := NewLogShareMemInMemory()
	share.SetPrintf(discardPrintf)
	base := time.Date(2013, 7, 5, 10, 0, 0, 0, time.Local)
	clock := base
	s := newTestServer(t, share, func() time.Time { return clock })
	printed, printf := capturePrintf()
	s.printf = printf
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	// world（槽 1）从未在线：永不自退。
	for i := 0; i < 100; i++ {
		s.TimerCheck()
	}
	if s.Exiting() {
		t.Fatal("槽 1 从未在线不应自退")
	}

	// world 在线跨越至少一轮（round 采样 GetLogUsed(1) 置 0），
	// 再消失：10 拍一轮 × 5 轮后退出（官方 5 秒看门狗）。
	prod := share.CloneView()
	prod.SetPrintf(discardPrintf)
	prod.SetAppId(1)
	prod.SetLogPrefix("FsSoloWorld")
	prod.TraceLog("world up")
	for i := 0; i < 10; i++ {
		s.TimerCheck()
	}
	prod.ClearLogUsed() // world 消失
	for i := 0; i < 60 && !s.Exiting(); i++ {
		s.TimerCheck()
	}
	if !s.Exiting() {
		t.Fatal("槽 1 消失 5 秒应触发退出")
	}
	var counts []string
	for _, l := range *printed {
		if strings.HasPrefix(l, "before exit count") {
			counts = append(counts, strings.TrimRight(l, "\n"))
		}
	}
	if len(counts) != 5 || counts[len(counts)-1] != "before exit count 5" {
		t.Fatalf("AutoExit printf 序列不符：%v", counts)
	}
}

func TestExclusiveMutex(t *testing.T) {
	name := "mutex_fs_solo_log_test_" + t.Name()
	rel, ok := acquireInstanceMutex(name)
	if !ok {
		t.Fatal("首次获取应成功")
	}
	if _, ok2 := acquireInstanceMutex(name); ok2 {
		t.Fatal("二次获取应互斥失败")
	}
	rel()
	if rel2, ok3 := acquireInstanceMutex(name); !ok3 {
		t.Fatal("释放后应可再获取")
	} else {
		rel2()
	}
}

func TestStartExclusiveErr(t *testing.T) {
	share := NewLogShareMemInMemory()
	share.SetPrintf(discardPrintf)
	clock := time.Date(2013, 7, 5, 10, 0, 0, 0, time.Local)
	mk := func() *Server {
		return New(Options{
			MutexName:  "mutex_fs_solo_log_test_" + t.Name(),
			Share:      share,
			LogFileDir: filepath.Join(t.TempDir(), "log") + string(os.PathSeparator),
			Now:        func() time.Time { return clock },
			Logf:       func(string, ...any) {},
			Printf:     discardPrintf,
			ManualTick: true,
		})
	}
	a := mk()
	if err := a.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.Stop)
	printed, printf := capturePrintf()
	b := mk()
	b.printf = printf
	err := b.Start()
	if err == nil {
		t.Fatal("第二实例应互斥失败")
	}
	if got := (*printed)[0]; got != "(SoloLog::Start)"+b.opt.MutexName+" exclusive failed\n" {
		t.Fatalf("互斥失败文案 = %q", got)
	}
}

func TestParseLine(t *testing.T) {
	args := ParseLine("setp solo_log AutoExit false")
	if len(args) != 4 || args[0].(string) != "setp" || args[3].(string) != "false" {
		t.Fatalf("基本切分/试型：%v", args)
	}
	args = ParseLine("method solo_log WriteLine 42 1.5")
	if args[3] != 42 || args[4] != 1.5 {
		t.Fatalf("int/float 试型：%v", args)
	}
	args = ParseLine(`getp "solo log" State`)
	if args[1].(string) != `"solo log"` {
		t.Fatalf("引号字符应保留进 token：%v", args)
	}
	args = ParseLine("a\tb  c")
	if len(args) != 3 {
		t.Fatalf("Tab/多空格切分：%v", args)
	}
	if len(ParseLine("   ")) != 0 {
		t.Fatal("空行应无参")
	}
}

func TestConfigDialect(t *testing.T) {
	cfg, err := ParseGameLogINI([]byte("[System]\nSleep=-1\nMainEntity=SoloLog\nMainScript=main\n\n[Module]\nfskit.dll\nfscomm.dll\nfssololog.dll\n\n[Script]\nPath=..\\lua\\sololog\nReload=1\n\n[Main]\n\n[Echo]\nAddr=10.0.0.1\nPort=15099\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MainEntity != "SoloLog" || cfg.Sleep != -1 || len(cfg.Modules) != 3 || cfg.ScriptPath != `..\lua\sololog` {
		t.Fatalf("方言投影：%+v", cfg)
	}
	if cfg.EchoAddr != "10.0.0.1" || cfg.EchoPort != 15099 {
		t.Fatalf("Echo 覆盖：%+v", cfg)
	}
	// 官方快照（无 [Echo]）→ main.lua 缺省 127.0.0.1:15011。
	cfg2, err := ParseGameLogINI(defaultGameLogINI)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.EchoAddr != "127.0.0.1" || cfg2.EchoPort != 15011 {
		t.Fatalf("缺省回退：%+v", cfg2)
	}
}

// —— echo 维护口（真实 TCP）——

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

// readLine 经共享 bufio.Reader 读一行（每连接一个 Reader，避免重复
// 包裹丢缓冲）。
func readLine(t *testing.T, c net.Conn, br *bufio.Reader) string {
	t.Helper()
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	line, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		t.Fatalf("读 echo 行：%v", err)
	}
	return line
}

func TestEchoSingleConnectorAndCommands(t *testing.T) {
	share := NewLogShareMemInMemory()
	share.SetPrintf(discardPrintf)
	port := freePort(t)
	s := New(Options{
		MutexName:  "mutex_fs_solo_log_test_" + t.Name(),
		Share:      share,
		EchoAddr:   "127.0.0.1",
		EchoPort:   port,
		LogFileDir: filepath.Join(t.TempDir(), "log") + string(os.PathSeparator),
		Logf:       func(string, ...any) {},
		Printf:     discardPrintf,
		ManualTick: true,
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)

	c1, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	br1 := bufio.NewReader(c1)
	if got := readLine(t, c1, br1); got != "welcome log server\r\n" {
		t.Fatalf("欢迎行 = %q", got)
	}

	// 单连接模型：第二连接顶替第一连接。
	c2, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	br2 := bufio.NewReader(c2)
	if got := readLine(t, c2, br2); got != "welcome log server\r\n" {
		t.Fatalf("c2 欢迎行 = %q", got)
	}
	c1.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(c1, make([]byte, 1)); err == nil {
		t.Fatal("旧主连接应被顶替关闭")
	}
	c1.Close()

	// getp：应答 ">" + 信息 + "\r"，SendLine 再叠 "\r\n"（双换行官方行为）。
	fmt.Fprintf(c2, "getp solo_log State\r\n")
	if got := readLine(t, c2, br2); got != ">Opened\r\r\n" {
		t.Fatalf("getp 应答 = %q", got)
	}
	fmt.Fprintf(c2, "setp solo_log AutoExit false\r\n")
	if got := readLine(t, c2, br2); got != ">ok\r\r\n" {
		t.Fatalf("setp 应答 = %q", got)
	}
	fmt.Fprintf(c2, "setp solo_log State true\r\n")
	if got := readLine(t, c2, br2); got != ">property is read-only: State\r\r\n" {
		t.Fatalf("只读属性应答 = %q", got)
	}
	fmt.Fprintf(c2, "blah\r\n")
	if got := readLine(t, c2, br2); got != ">undefined command: blah\r\r\n" {
		t.Fatalf("未定义命令应答 = %q", got)
	}

	// end：仅断开本维护连接。
	fmt.Fprintf(c2, "end\r\n")
	c2.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(c2, make([]byte, 1)); err == nil {
		t.Fatal("end 后应断开连接")
	}
	c2.Close()
}

func TestEchoQuitStopsServer(t *testing.T) {
	share := NewLogShareMemInMemory()
	share.SetPrintf(discardPrintf)
	port := freePort(t)
	s := New(Options{
		MutexName:  "mutex_fs_solo_log_test_" + t.Name(),
		Share:      share,
		EchoAddr:   "127.0.0.1",
		EchoPort:   port,
		LogFileDir: filepath.Join(t.TempDir(), "log") + string(os.PathSeparator),
		Logf:       func(string, ...any) {},
		Printf:     discardPrintf,
		ManualTick: true,
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(c)
	readLine(t, c, br) // welcome
	fmt.Fprintf(c, "quit\r\n")
	if got := readLine(t, c, br); got != ">server will shutdown...\r\r\n" {
		t.Fatalf("quit 应答 = %q", got)
	}
	select {
	case <-s.Stopped():
	case <-time.After(3 * time.Second):
		t.Fatal("quit 应触发停服")
	}
}
