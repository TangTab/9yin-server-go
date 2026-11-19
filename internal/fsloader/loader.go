package fsloader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/fskit"
)

// Loader 复刻官方 FsLoader.exe 进程壳（_main 五步序列）。
// argv 约定与官方一致：args[0]=ini 名（空=从可执行文件名派生），
// args[1..]=附加参数。
type Loader struct {
	// ExePath 可执行文件路径（官方 argv[0]；空=取 os.Executable）。
	ExePath string
	// CoreFactory 内核工厂（官方 LoadLibrary("FxCore.dll") 的进程内形态）。
	CoreFactory func() Core
	// Logf 运行日志（nil=标准输出）。
	Logf func(format string, args ...any)

	core Core
	quit chan struct{} // 官方 g_quit：console_ctrl_handler 置位
}

// Run 复刻 _main 全序列，返回进程退出码。ctx 取消（Ctrl+C/
// SIGINT，等价官方 SetConsoleCtrlHandler）→ 泵循环退出 → CoreShutdown。
// panic 由 crashlog 承接（官方 MiniDump+ERROR*.log 面）。
func (l *Loader) Run(ctx context.Context, args []string) (code int) {
	if l.Logf == nil {
		l.Logf = func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	}
	defer func() {
		if v := recover(); v != nil {
			WriteErrorLog(v)
			l.Logf("FsLoader: 异常退出（详见 %s）", lastErrorLogName())
			code = 1
			l.shutdownCore()
		}
	}()

	// 官方第一步：控制台 Ctrl 处理（失败即退）。Go 以 ctx 取消等价，
	// 注册动作交由调用方 signal.NotifyContext 完成。
	l.quit = make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			close(l.quit) // g_quit = 1
		case <-l.quit:
		}
	}()

	// 官方第二步：装载内核（LoadCoreDll 双门：type=="FxCore"、
	// version==1003；失败文案逐字对齐）。
	if l.CoreFactory == nil {
		l.fatal("Load core dll failed")
		return 1
	}
	core := l.CoreFactory()
	if core.Type() != CoreType {
		l.fatal("(LoadCoreDll)dll type not " + CoreType)
		return 1
	}
	if core.Version() != CoreVersion {
		l.fatal(fmt.Sprintf("(LoadCoreDll)core version not match (%d≠%d)", core.Version(), CoreVersion))
		return 1
	}
	l.core = core

	// 官方第三步：ini 名发现——args[0] 覆盖 > exe 派生 > 字面量回退。
	iniName := ""
	if len(args) > 0 {
		iniName = args[0]
	}
	if iniName == "" {
		iniName = IniNameFromExe(l.exePath())
	}
	if iniName == "" {
		iniName = "FsLoader.ini" // 官方回退字面量
	}
	extra := []string{}
	if len(args) > 1 {
		extra = append(extra, args[1:]...)
	}

	// 官方第四步：app_load_core_ini（官方 CIniFile 方言）。
	cfg, err := LoadCoreConfig(fskit.NewIniFile(iniName))
	if err != nil {
		l.fatal("Load core ini failed: " + err.Error())
		return 1
	}
	cfg.ExtraArgs = extra
	cfg.IniName = iniName
	l.Logf("FsLoader: ini=%s Sleep=%d DebugLog=%t MainEntity=%s MainScript=%s 模块=%d 附加=%d",
		iniName, cfg.Sleep, cfg.DebugLog, cfg.MainEntity, cfg.MainScript,
		len(cfg.Modules), len(extra))

	// 官方第五步：Initialize→Startup→泵循环→Shutdown（CCore::Startup
	// 序：VFS→模块→主实体→主逻辑→主脚本，失败文案见 core.go）。
	if err := core.Startup(cfg); err != nil {
		l.fatal(err.Error())
		return 1
	}
	l.pump(core, cfg)

	l.shutdownCore()
	l.Logf("FsLoader: 正常退出")
	return 0
}

// pump 复刻装载器主循环：`while(!g_quit && core->Process())` +
// Sleep 毫秒节流（<0=不休眠全速；dt 钳制在内核侧）。
func (l *Loader) pump(core Core, cfg *CoreConfig) {
	var sleep time.Duration
	if cfg.Sleep > 0 {
		sleep = time.Duration(cfg.Sleep) * time.Millisecond
	}
	for {
		select {
		case <-l.quit:
			return
		default:
		}
		if !core.Process() {
			return
		}
		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

func (l *Loader) shutdownCore() {
	if l.core != nil {
		l.core.Shutdown()
		l.core = nil
	}
}

func (l *Loader) fatal(msg string) {
	fmt.Fprintln(os.Stderr, "FsLoader:", msg)
}

func (l *Loader) exePath() string {
	if l.ExePath != "" {
		return l.ExePath
	}
	if p, err := os.Executable(); err == nil {
		return p
	}
	return ""
}

// ChdirToExeDir 复刻官方 Port_SetCurrentDir(exe 目录)：装载器以
// 可执行文件所在目录为工作目录（ini/模块/脚本相对路径的锚点）。
func ChdirToExeDir(exePath string) error {
	if exePath == "" {
		return nil
	}
	return os.Chdir(filepath.Dir(exePath))
}
