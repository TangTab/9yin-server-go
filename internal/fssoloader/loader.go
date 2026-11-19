// Package fssoloader 复刻官方 FsSoloLoader.exe（小写系 solo 集群进程壳）。
//
// 权威快照：F:\Game\Online\9yin\BGserverCJ\Debug\FsSoloLoader.exe
//（c2f2df73…，589,824 B，2013-07-05 构建，源码树
// c:\wuyf\svn\serveriii\branches\publish20130508\server\fssololoader\
// main.cpp + tools\apptool.cpp）+ FsSoloLoader.map（3188 行符号表）。
// 生产部署为改名拷贝 game_{lister,entry,world,member,public,store,log,
// global,auction,unite,monitor}.exe（md5 195e1ce1… 同构构建——_main 序列、
// vtable 槽位、g_running 全局 0x4c41ec 逐点一致），ini 名从 exe 基名派生。
//
// 与 FsLoader.exe（internal/fsloader）同源异代：五步序列同构，差异定案：
//
//	ini 发现      回退字面量 "fssololoader.ini"（小写）+ 派生名转小写
//	              （app_get_ini_name = GetFullPathName→Port_StringLower→
//	              去 .exe→追加 .ini）；FsLoader 为 "FsLoader.ini" 不转小写
//	内核门文案    前缀 "(app_load_core_dll)"（=函数名）；FsLoader 为
//	              "(LoadCoreDll)"。双门常量相同：类型串 "FxCore"、版本 1003
//	致命文案      全小写（"load core dll failed"…）；FsLoader 首字母大写
//	主实体门      Startup 后 core->vtbl[+0x78]() 取主实体（core+0x314），
//	              null → fatal "main entity error"（FsLoader 无此门）
//	泵循环        while (g_running) { if (core->vtbl[+0x40]()) break;
//	              ent->vtbl[+0xc](0.0f); }——壳层不调 +0x4c 泵（FsLoader
//	              为 while(!g_quit && vtbl[+0x4c]())）；主实体逐帧驱动，
//	              dt 恒 0.0（fldz 定案；0=内核实测时差）；循环内无 Sleep
//	Ctrl 事件     映射表 [0,0,1,2,2,1,1]：CTRL_C/BREAK 吞掉（return TRUE
//	              不停服——运维防误触），CLOSE/LOGOFF/SHUTDOWN 置
//	              g_running=0 优雅停机；FsLoader 为全事件停机
//	正常退出码    main 返回 1（mov eax,1 定案）；FsLoader 未定值
//
// 完整分析见 docs/official/FsSoloLoader全解与Go复刻.md。
package fssoloader

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	fsloader "github.com/lizhanyin/9yin-server-go/internal/fsloader"
	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// DefaultIniName 官方回退字面量（app_get_ini_name 空产物时）。
const DefaultIniName = "fssololoader.ini"

// MainEntityView 官方 IEntity 面——装载器只用到帧方法（vtbl+0xc，
// float dt 参；FsSoloLoader 泵循环逐拍以 dt=0 调用）。
type MainEntityView interface {
	Frame(dt float32)
}

// SoloCore 官方 ICore 面（FsSoloLoader 契约）：fsloader.Core 四方法
//（Type/Version/Startup/Process/Shutdown）+ solo 壳层两槽位。
type SoloCore interface {
	fsloader.Core
	// WantQuit 官方 vtbl+0x40（true=装载器退出泵循环——与 FsLoader
	// 泵 +0x4c 的"true=继续"相反）。
	WantQuit() bool
	// MainEntity 官方 vtbl+0x78（core+0x314 主实体；null → 壳层
	// fatal "main entity error"）。
	MainEntity() MainEntityView
}

// SoloLoader 复刻 FsSoloLoader.exe 进程壳（_main 序列）。
// argv 约定与官方一致：args[0]=ini 名（空=从可执行文件名派生，回退
// 字面量 fssololoader.ini），args[1..]=附加参数（argv[2..] 官方位）。
type SoloLoader struct {
	// ExePath 可执行文件路径（官方 argv[0]；空=取 os.Executable）。
	ExePath string
	// CoreFactory 内核工厂（官方 LoadLibrary("FxCore.dll") 的进程内
	// 形态；nil 返回值=装载失败 "(app_load_core_dll)open dll failed"）。
	CoreFactory func() SoloCore
	// Logf 运行日志（nil=标准输出）。
	Logf func(format string, args ...any)
	// CtrlCStops 放开官方"Ctrl+C 吞掉"语义（联调/测试用；官方 false）。
	CtrlCStops bool

	core    SoloCore
	quit    chan struct{}
	quitOnce chan struct{}
}

// Run 复刻 _main 全序列，返回进程退出码（官方正常路径 mov eax,1 → 1）。
// 停机面：ctx 取消（宿主控制）与 SIGTERM（服务停机等价 CTRL_CLOSE/
// LOGOFF/SHUTDOWN）→ g_running=0 → 泵循环退出 → CoreShutdown；
// SIGINT（Ctrl+C/CTRL_BREAK）按官方映射吞掉（CtrlCStops 可放开）。
// panic 由 fsloader.WriteErrorLog 承接（官方 ERROR%04d….log 崩溃面）。
func (l *SoloLoader) Run(ctx context.Context, args []string) (code int) {
	if l.Logf == nil {
		l.Logf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	}
	defer func() {
		if v := recover(); v != nil {
			name := fsloader.WriteErrorLog(v)
			l.Logf("FsSoloLoader: 异常退出（详见 %s）", name)
			code = 1
			l.shutdownCore()
		}
	}()

	// 官方第一步：SetConsoleCtrlHandler——事件映射 [0,0,1,2,2,1,1]
	//（CTRL_C/BREAK 吞掉；CLOSE/LOGOFF/SHUTDOWN → g_running=0）。
	l.quit = make(chan struct{})
	l.quitOnce = make(chan struct{})
	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		for s := range sigCh {
			if s == syscall.SIGTERM || l.CtrlCStops {
				l.requestQuit()
				return
			}
			l.Logf("FsSoloLoader: 忽略 %v（官方 Ctrl+C 语义：吞掉不停服）", s)
		}
	}()
	go func() {
		select {
		case <-ctx.Done():
			l.requestQuit()
		case <-l.quitOnce:
		}
	}()

	// 官方第三步：装载内核（app_load_core_dll 六门之三门可在进程内
	// 形态复现：open/type/version；文案前缀逐字）。
	var core SoloCore
	if l.CoreFactory != nil {
		core = l.CoreFactory()
	}
	if core == nil {
		l.fatal("(app_load_core_dll)open dll failed")
		return 1
	}
	if core.Type() != fsloader.CoreType {
		l.fatal("(app_load_core_dll)dll type not " + fsloader.CoreType)
		return 1
	}
	if core.Version() != fsloader.CoreVersion {
		l.fatal(fmt.Sprintf("(app_load_core_dll)core version not match (%d≠%d)",
			core.Version(), fsloader.CoreVersion))
		return 1
	}
	l.core = core

	// 官方第四步：ini 名发现——args[0] 覆盖 > 小写派生 > 字面量回退。
	iniName := ""
	if len(args) > 0 {
		iniName = args[0]
	}
	if iniName == "" {
		iniName = IniNameFromExe(l.exePath())
	}
	if iniName == "" {
		iniName = DefaultIniName // 官方回退字面量（小写）
	}
	extra := []string{}
	if len(args) > 1 {
		extra = append(extra, args[1:]...)
	}

	// 官方第五步之一：app_load_core_ini（与 FsLoader 逐字节同方言）。
	cfg, err := fsloader.LoadCoreConfig(fskit.NewIniFile(iniName))
	if err != nil {
		l.fatal("load core ini failed: " + err.Error())
		return 1
	}
	cfg.ExtraArgs = extra
	cfg.IniName = iniName
	l.Logf("FsSoloLoader: ini=%s Sleep=%d MainEntity=%s MainScript=%s 模块=%d 附加=%d",
		iniName, cfg.Sleep, cfg.MainEntity, cfg.MainScript,
		len(cfg.Modules), len(extra))

	// 官方第五步之二：CoreInitialize→CoreStartup(g_core, &ExtraArgs)。
	if err := core.Startup(cfg); err != nil {
		l.fatal("core startup failed: " + err.Error())
		return 1
	}

	// ★ solo 独有门：主实体（vtbl[+0x78]() → core+0x314；null 致命）。
	ent := core.MainEntity()
	if ent == nil {
		l.fatal("main entity error")
		return 1
	}

	// ★ solo 泵循环：主实体逐帧驱动（dt=0 官方字面；内核实测时差步进）。
	l.pump(core, ent)

	l.shutdownCore()
	l.Logf("FsSoloLoader: 正常退出")
	return 1 // 官方 mov eax,1（正常路径退出码 1）
}

// pump 复刻装载器主循环：`while (g_running) { if (vtbl[+0x40]()) break;
// ent->vtbl[+0xc](0.0f); }`——无 Sleep 节流（System.Sleep 为内核/
// 角色方言键，壳层不消费）。
func (l *SoloLoader) pump(core SoloCore, ent MainEntityView) {
	for {
		select {
		case <-l.quit: // g_running == 0
			return
		default:
		}
		if core.WantQuit() { // vtbl[+0x40] → true=退出
			return
		}
		ent.Frame(0) // vtbl[+0xc](0.0f)
	}
}

func (l *SoloLoader) requestQuit() {
	select {
	case <-l.quitOnce:
	default:
		close(l.quitOnce)
		close(l.quit) // g_running = 0
	}
}

func (l *SoloLoader) shutdownCore() {
	if l.core != nil {
		l.core.Shutdown()
		l.core = nil
	}
}

func (l *SoloLoader) fatal(msg string) {
	fmt.Fprintln(os.Stderr, "FsSoloLoader:", msg)
}

func (l *SoloLoader) exePath() string {
	if l.ExePath != "" {
		return l.ExePath
	}
	if p, err := os.Executable(); err == nil {
		return p
	}
	return ""
}

// IniNameFromExe 复刻官方 app_get_ini_name：exe 路径 →
// GetFullPathName → Port_StringLower（全串转小写）→ 去 .exe →
// 追加 .ini。派生失败返回空串（壳层回退字面量 fssololoader.ini）。
// game_lister.exe → "game_lister.ini"；FsSoloLoader.exe →
// "fssololoader.ini"。
func IniNameFromExe(exePath string) string {
	if exePath == "" {
		return ""
	}
	base := strings.ToLower(filepath.Base(exePath))
	if ext := filepath.Ext(base); ext != "" {
		base = base[:len(base)-len(ext)]
	}
	if base == "" {
		return ""
	}
	return base + ".ini"
}
