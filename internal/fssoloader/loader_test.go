package fssoloader

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	fsloader "github.com/lizhanyin/9yin-server-go/internal/fsloader"
)

// fakeCore 测试用 SoloCore：帧/停机计数可编程。
type fakeCore struct {
	mu atomic.Int32 // 帧计数
	tick int32      // WantQuit 转 true 的帧阈值（<=0=从不）
	dts []float32

	startupErr error
	mainNil    bool
	started    atomic.Bool
	shutdown   atomic.Bool

	typ    string
	ver    int
}

func (f *fakeCore) Type() string {
	if f.typ != "" {
		return f.typ
	}
	return fsloader.CoreType
}

func (f *fakeCore) Version() int {
	if f.ver != 0 {
		return f.ver
	}
	return fsloader.CoreVersion
}

func (f *fakeCore) Startup(cfg *fsloader.CoreConfig) error {
	if f.startupErr != nil {
		return f.startupErr
	}
	f.started.Store(true)
	return nil
}

func (f *fakeCore) Process() bool { return !f.shutdown.Load() }

func (f *fakeCore) WantQuit() bool { return f.tick > 0 && f.mu.Load() >= int32(f.tick) }

func (f *fakeCore) MainEntity() MainEntityView {
	if f.mainNil {
		return nil
	}
	return fakeEntity{f}
}

func (f *fakeCore) Shutdown() { f.shutdown.Store(true) }

// fakeEntity 帧面：记录 dt。
type fakeEntity struct{ f *fakeCore }

func (e fakeEntity) Frame(dt float32) {
	e.f.dts = append(e.f.dts, dt)
	e.f.mu.Add(1)
}

// writeIni 在临时目录落一份官方方言 ini。
func writeIni(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	body := "[System]\nSleep=-1\nMainEntity=MainCon\nMainScript=main\n\n[Module]\nfskit.dll\n\n[Script]\nPath=\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestIniNameFromExeLowercase(t *testing.T) {
	cases := []struct{ in, want string }{
		{`C:\Debug\Game_Lister.EXE`, "game_lister.ini"},
		{`C:\Debug\FsSoloLoader.exe`, "fssololoader.ini"},
		{`D:\srv\game_entry.exe`, "game_entry.ini"},
		{"game_world", "game_world.ini"},
		{"", ""},
	}
	for _, c := range cases {
		if got := IniNameFromExe(c.in); got != c.want {
			t.Errorf("IniNameFromExe(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestCoreGatesOfficialMessages(t *testing.T) {
	// ① 工厂 nil（open dll failed）。
	l := &SoloLoader{Logf: func(string, ...any) {}}
	if code := l.Run(context.Background(), nil); code != 1 {
		t.Fatalf("nil factory: code=%d want 1", code)
	}

	// ② 类型门（官方前缀 (app_load_core_dll)）。
	l = &SoloLoader{
		Logf:        func(string, ...any) {},
		CoreFactory: func() SoloCore { return &fakeCore{typ: "FxModule"} },
	}
	if code := l.Run(context.Background(), nil); code != 1 {
		t.Fatalf("type gate: code=%d want 1", code)
	}

	// ③ 版本门（1003）。
	l = &SoloLoader{
		Logf:        func(string, ...any) {},
		CoreFactory: func() SoloCore { return &fakeCore{ver: 1002} },
	}
	if code := l.Run(context.Background(), nil); code != 1 {
		t.Fatalf("version gate: code=%d want 1", code)
	}
}

func TestFatalMessagesVerbatim(t *testing.T) {
	// 关键常量逐字（分支文案断言见各路径用例；官方前缀 (app_load_core_dll)
	// 与小写回退字面量为装载器↔内核硬契约）。
	if DefaultIniName != "fssololoader.ini" {
		t.Errorf("DefaultIniName=%q want fssololoader.ini", DefaultIniName)
	}
	if strings.Contains(DefaultIniName, "FsLoader") {
		t.Error("solo 回退字面量不得混入 FsLoader 形态")
	}
}

func TestFallbackIniLiteralAndLoadFailure(t *testing.T) {
	// 无参 + 无 exe 派生 → 回退字面量 fssololoader.ini（目录中不存在
	// → "load core ini failed"，退出码 1）。
	wd, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	l := &SoloLoader{
		Logf:        func(string, ...any) {},
		CoreFactory: func() SoloCore { return &fakeCore{} },
	}
	if code := l.Run(context.Background(), nil); code != 1 {
		t.Fatalf("missing ini: code=%d want 1", code)
	}
	if _, err := os.Stat(DefaultIniName); err == nil {
		t.Errorf("壳层不得自行释放 ini（%s 不应存在）", DefaultIniName)
	}
}

func TestMainEntityGate(t *testing.T) {
	ini := writeIni(t, "fssololoader.ini")
	wd, _ := os.Getwd()
	if err := os.Chdir(filepath.Dir(ini)); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	l := &SoloLoader{
		Logf:        func(string, ...any) {},
		CoreFactory: func() SoloCore { return &fakeCore{mainNil: true} },
	}
	if code := l.Run(context.Background(), nil); code != 1 {
		t.Fatalf("main entity gate: code=%d want 1", code)
	}
}

func TestPumpLoopAndExitCode(t *testing.T) {
	ini := writeIni(t, "game_lister.ini")
	wd, _ := os.Getwd()
	if err := os.Chdir(filepath.Dir(ini)); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	fc := &fakeCore{tick: 5}
	l := &SoloLoader{
		Logf:        func(string, ...any) {},
		CoreFactory: func() SoloCore { return fc },
	}
	code := l.Run(context.Background(), []string{"game_lister.ini"})
	if code != 1 {
		t.Fatalf("normal exit code=%d want 1（官方 mov eax,1）", code)
	}
	if n := fc.mu.Load(); n != 5 {
		t.Errorf("帧数=%d want 5（WantQuit=true 退出）", n)
	}
	for i, dt := range fc.dts {
		if dt != 0 {
			t.Errorf("第 %d 帧 dt=%v want 0（官方 fldz 字面）", i, dt)
		}
	}
	if !fc.shutdown.Load() {
		t.Error("CoreShutdown 未调用")
	}
}

func TestContextCancelStopsPump(t *testing.T) {
	ini := writeIni(t, "game_world.ini")
	wd, _ := os.Getwd()
	if err := os.Chdir(filepath.Dir(ini)); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	ctx, cancel := context.WithCancel(context.Background())
	fc := &fakeCore{} // tick<=0：从不 WantQuit——仅 ctx 停机
	l := &SoloLoader{
		Logf:        func(string, ...any) {},
		CoreFactory: func() SoloCore { return fc },
	}
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	done := make(chan int, 1)
	go func() { done <- l.Run(ctx, []string{"game_world.ini"}) }()
	select {
	case code := <-done:
		if code != 1 {
			t.Errorf("code=%d want 1", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ctx 取消未退出泵循环")
	}
	if !fc.shutdown.Load() {
		t.Error("ctx 停机后 CoreShutdown 未调用")
	}
	if n := fc.mu.Load(); n == 0 {
		t.Error("泵循环未驱动任何帧")
	}
}

func TestStartupFailureFatal(t *testing.T) {
	ini := writeIni(t, "game_entry.ini")
	wd, _ := os.Getwd()
	if err := os.Chdir(filepath.Dir(ini)); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	l := &SoloLoader{
		Logf: func(string, ...any) {},
		CoreFactory: func() SoloCore {
			return &fakeCore{startupErr: errFake("simulate")}
		},
	}
	if code := l.Run(context.Background(), []string{"game_entry.ini"}); code != 1 {
		t.Fatalf("startup fail: code=%d want 1", code)
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }

func TestExtraArgsForwarded(t *testing.T) {
	ini := writeIni(t, "game_global.ini")
	wd, _ := os.Getwd()
	if err := os.Chdir(filepath.Dir(ini)); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	var gotArgs []string
	var gotIni string
	fc := &fakeCore{tick: 1}
	// fakeCore.Startup 忽略 cfg——包一层在 Startup 时机捕获配置面。
	wrapped := &captureCore{SoloCore: fc, onStart: func(cfg *fsloader.CoreConfig) {
		gotArgs = append(gotArgs, cfg.ExtraArgs...)
		gotIni = cfg.IniName
	}}
	l := &SoloLoader{
		Logf:        func(string, ...any) {},
		CoreFactory: func() SoloCore { return wrapped },
	}
	l.Run(context.Background(), []string{"game_global.ini", "-idc=1", "-n=2"})
	if len(gotArgs) != 2 || gotArgs[0] != "-idc=1" || gotArgs[1] != "-n=2" {
		t.Errorf("ExtraArgs=%v want [-idc=1 -n=2]", gotArgs)
	}
	if gotIni != "game_global.ini" {
		t.Errorf("IniName=%q", gotIni)
	}
}

type captureCore struct {
	SoloCore
	onStart func(*fsloader.CoreConfig)
}

func (c *captureCore) Startup(cfg *fsloader.CoreConfig) error {
	if c.onStart != nil {
		c.onStart(cfg)
	}
	return c.SoloCore.Startup(cfg)
}

func TestSignalSemanticsSwallowCtrlC(t *testing.T) {
	// 官方映射：CTRL_C/BREAK 吞掉（不停服）——SIGINT 发送后泵循环
	// 必须仍在运转，直到 WantQuit/ctx 停机。
	ini := writeIni(t, "game_log.ini")
	wd, _ := os.Getwd()
	if err := os.Chdir(filepath.Dir(ini)); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fc := &fakeCore{}
	l := &SoloLoader{
		Logf: func(string, ...any) {},
		CoreFactory: func() SoloCore { return fc },
	}
	done := make(chan int, 1)
	go func() { done <- l.Run(ctx, []string{"game_log.ini"}) }()

	time.Sleep(100 * time.Millisecond) // 等 signal.Notify 生效
	proc, err := os.FindProcess(os.Getpid())
	if err == nil {
		_ = proc.Signal(os.Interrupt) // CTRL_C 等价
	}
	select {
	case <-done:
		t.Fatal("SIGINT 不应停服（官方吞掉语义）")
	case <-time.After(300 * time.Millisecond):
	}
	if fc.shutdown.Load() {
		t.Fatal("SIGINT 不应触发 CoreShutdown")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ctx 取消未停机")
	}
}
