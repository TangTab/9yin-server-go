package fsloader

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/ecs"
	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// officialLoaderINI 官方 FsLoader.ini 快照（方言回归基准）。
const officialLoaderINI = `[System]
// 主循环休眠毫秒数（小于0则不休眠）
Sleep=1
// 是否使用调试日志系统
DebugLog=1
// 文件系统
FileSystem=
// 主实体名称
MainEntity=MainCon
// 启动脚本名称
MainScript=main
// 启动逻辑类名称
MainLogic=

// 指定加载的组件模块（为空则全部加载）
[Module]
FsKit.dll

// 脚本文件
[Script]
// 所在路径
Path=..\Lua
// 是否即时重新加载
Reload=1

// 资源文件
[Resource]
// 所在路径
Path=..\Res

`

// writeIni 落盘官方方言 ini（CIniFile 从文件装载）。
func writeIni(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadCoreConfigOfficialDialect(t *testing.T) {
	// 官方相对路径方言：..\Lua / ..\Res 相对进程工作目录
	//（Debug 目录）——测试以 <tmp>/debug 为 cwd，Lua/Res 建在其父级。
	root := t.TempDir()
	cwd := filepath.Join(root, "debug")
	for _, d := range []string{cwd, filepath.Join(root, "Lua"), filepath.Join(root, "Res")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	scriptDir := filepath.Join(root, "Lua")
	resDir := filepath.Join(root, "Res")
	oldWd, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	ini := fskit.NewIniFile(writeIni(t, "FsLoader.ini", officialLoaderINI))
	cfg, err := LoadCoreConfig(ini)
	if err != nil {
		t.Fatalf("官方方言解析失败: %v", err)
	}
	if cfg.Sleep != 1 || !cfg.DebugLog || !cfg.ScriptReload {
		t.Errorf("System/Script 键缺省语义错误: %+v", cfg)
	}
	if cfg.MainEntity != "MainCon" || cfg.MainScript != "main" || cfg.MainLogic != "" {
		t.Errorf("Main* 键错误: %+v", cfg)
	}
	if cfg.FileSystem != "" || cfg.FileConfig != "" {
		t.Errorf("FileSystem/FileConfig 应为空: %+v", cfg)
	}
	if len(cfg.Modules) != 1 || cfg.Modules[0] != "FsKit.dll" {
		t.Errorf("[Module] 裸行表应取整行 key: %v", cfg.Modules)
	}
	// 目录归一：尾分隔符 + 绝对化。
	if !strings.HasSuffix(cfg.ScriptPath, string(os.PathSeparator)) {
		t.Errorf("ScriptPath 应保证尾分隔符: %q", cfg.ScriptPath)
	}
	if filepath.Dir(cfg.ScriptPath) != scriptDir {
		t.Errorf("ScriptPath 归一错误: %q 期望 %q", cfg.ScriptPath, scriptDir)
	}
	if filepath.Dir(cfg.ResourcePath) != resDir {
		t.Errorf("ResourcePath 归一错误: %q", cfg.ResourcePath)
	}
}

func TestLoadCoreConfigPathNotFound(t *testing.T) {
	oldWd, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	ini := fskit.NewIniFile(writeIni(t, "FsLoader.ini",
		strings.Replace(officialLoaderINI, `Path=..\Lua`, `Path=..\Nope`, 1)))
	_, err := LoadCoreConfig(ini)
	if err == nil || !strings.Contains(err.Error(), "Script path not found") {
		t.Fatalf("缺目录应为致命错误（官方文案）: %v", err)
	}
}

func TestIniNameFromExe(t *testing.T) {
	cases := map[string]string{
		`D:\srv\FsLoader.exe`: "FsLoader.ini",
		`D:\srv\game_lister.exe`: "game_lister.ini",
		`FsSoloLoader.exe`:     "FsSoloLoader.ini",
		"":                     "",
	}
	for in, want := range cases {
		if got := IniNameFromExe(in); got != want {
			t.Errorf("IniNameFromExe(%q)=%q 期望 %q", in, got, want)
		}
	}
}

//---- 模块注册表（CDllManager 面） ----

func testModule(name string, ents []EntityDecl) Module {
	return &DeclModule{
		ModuleName:  name,
		ModuleSpace: "Test",
		ModuleEnts:  ents,
	}
}

func TestModuleRegistryChecks(t *testing.T) {
	r := NewModuleRegistry()
	if err := r.Register(testModule("fskit.dll", []EntityDecl{{TypeName: "ArrayList"}})); err != nil {
		t.Fatal(err)
	}
	// 重复注册。
	if err := r.Register(testModule("FsKit.dll", nil)); err == nil {
		t.Error("重复注册应报错（大小写不敏感）")
	}
	// 未注册模块装载。
	if _, err := r.Load("fscomm.dll", nil); err == nil ||
		!strings.Contains(err.Error(), "module interface not found") {
		t.Errorf("未注册装载文案错误: %v", err)
	}
	// 装载成功。
	core := NewEngineCore(r, CoreHooks{})
	if _, err := r.Load("fskit.dll", core); err != nil {
		t.Fatal(err)
	}
	// 重复装载。
	if _, err := r.Load("fskit.dll", core); err == nil ||
		!strings.Contains(err.Error(), "already loaded") {
		t.Errorf("重复装载文案错误: %v", err)
	}
	// 实体重名。
	if err := r.Register(testModule("other.dll", []EntityDecl{{TypeName: "arraylist"}})); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Load("other.dll", core); err == nil ||
		!strings.Contains(err.Error(), "repeat entity name") {
		t.Errorf("实体重名文案错误: %v", err)
	}
	// 查找。
	if _, ok := r.FindEntity("ArrayList"); !ok {
		t.Error("FindEntity 应命中已装载实体")
	}
	if _, ok := r.FindEntity("Nope"); ok {
		t.Error("FindEntity 不应命中未注册实体")
	}
}

//---- 内核与装载器壳 ----

// stubCore 泵计数桩（校验装载器 while 语义与退出路径）。
type stubCore struct {
	stopped bool
	ticks   int
}

func (s *stubCore) Type() string { return CoreType }
func (s *stubCore) Version() int { return CoreVersion }
func (s *stubCore) Startup(cfg *CoreConfig) error { return nil }
func (s *stubCore) Process() bool {
	s.ticks++
	return true
}
func (s *stubCore) Shutdown() { s.stopped = true }

func TestLoaderPumpAndShutdown(t *testing.T) {
	core := &stubCore{}
	loader := &Loader{CoreFactory: func() Core { return core }}
	iniPath := writeWorkingConfig(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	code := loader.Run(ctx, []string{filepath.Base(iniPath)})
	if code != 0 {
		t.Fatalf("正常退出码应为 0: %d", code)
	}
	if core.ticks == 0 {
		t.Error("泵循环应驱动内核至少一拍")
	}
	if !core.stopped {
		t.Error("退出后应调用 Shutdown（官方 CoreShutdown 序）")
	}
}

func TestLoaderCoreGate(t *testing.T) {
	// 版本门：官方 "(LoadCoreDll)core version not match"。
	badVer := &stubCore{}
	loader := &Loader{CoreFactory: func() Core {
		return &versionCore{stubCore: badVer, ver: 1002}
	}}
	writeWorkingConfig(t)
	if code := loader.Run(context.Background(), []string{"whatever.ini"}); code != 1 {
		t.Error("版本不符应退出码 1")
	}
}

type versionCore struct {
	*stubCore
	ver int
}

func (v *versionCore) Version() int { return v.ver }

// writeWorkingConfig 落一份最小可启动配置（Sleep=0 全速、无脚本段），
// 并把 cwd 切到 ini 所在目录（官方进程相对路径锚点）。
func writeWorkingConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "FsLoader.ini")
	if err := os.WriteFile(path, []byte("[System]\nSleep=0\nMainEntity=MainCon\nMainScript=\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(oldWd) })
	return path
}

func TestEngineCoreStartupSequence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.lua"), []byte("-- main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewModuleRegistry()
	if err := r.Register(testModule("fskit.dll", nil)); err != nil {
		t.Fatal(err)
	}
	var mainEntity ecs.EntityID
	core := NewEngineCore(r, CoreHooks{
		OnMainEntity: func(c *EngineCore, e ecs.EntityID, cfg *CoreConfig) error {
			mainEntity = e
			return nil
		},
	})
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := core.Startup(&CoreConfig{
		MainEntity: "MainCon", MainScript: "main",
		ScriptPath: dir + string(os.PathSeparator),
		Modules:    []string{"fskit.dll"},
	}); err != nil {
		t.Fatalf("Startup 失败: %v", err)
	}
	if mainEntity.IsNull() || !core.App.World.Alive(mainEntity) {
		t.Error("主实体应物化在 World（官方 nx_create 位）")
	}
	if core.ScriptEnv == nil || core.ScriptEnv.File == "" {
		t.Error("主脚本应定位成功")
	}
	if len(core.Registry.Loaded()) != 1 {
		t.Error("应装载 1 个模块")
	}
	// 泵驱动一拍。
	core.Startup(&CoreConfig{ScriptPath: dir})
	if !core.Process() {
		t.Error("Process 应返回 true")
	}
	if core.App.World.CurrentTick() == 0 {
		t.Error("Process 应推进 Scheduler 帧")
	}
	core.Shutdown()
	core.Shutdown() // 幂等
}

func TestEngineCoreScriptNotFound(t *testing.T) {
	core := NewEngineCore(NewModuleRegistry(), CoreHooks{})
	err := core.Startup(&CoreConfig{
		MainScript: "main",
		ScriptPath: t.TempDir() + string(os.PathSeparator),
	})
	if err == nil || !strings.Contains(err.Error(), "(CCore::Startup)run main script failed") {
		t.Fatalf("缺主脚本应为官方致命文案: %v", err)
	}
}

func TestErrorLogName(t *testing.T) {
	tm := time.Date(2026, 11, 19, 8, 5, 7, 123000000, time.Local)
	if got := errorLogName(tm); got != "ERROR20261119080507123.log" {
		t.Errorf("官方命名格式错误: %q", got)
	}
}
