package fxcore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/engine"
	"github.com/lizhanyin/9yin-server-go/internal/fsloader"
)

// 装载契约常量（与 fsloader 对齐；官方 FxModule_GetType/GetVersion）。
const (
	CoreType    = fsloader.CoreType    // "FxCore"
	CoreVersion = fsloader.CoreVersion // 1003 (0x3EB)
)

// Hooks Go 装配点（官方由 Lua main 脚本承接；里程碑二前的等价面）。
type Hooks struct {
	// OnModulesLoaded 模块装载后（官方 main 脚本运行前的装配位）。
	OnModulesLoaded func(c *Core) error
	// OnMainEntity 主实体创建后。
	OnMainEntity func(c *Core, e *Entity) error
	// OnMainScript 主脚本定位后（Lua VM 未接线期间=登记+mtime 热重载）。
	OnMainScript func(c *Core, s *Script) error
	// OnMainLogic 主逻辑绑定（MainLogic 非空时必须有钩子——官方
	// "bind main logic failed" 语义）。
	OnMainLogic func(c *Core, e *Entity) error
	// OnProcess 每泵一拍后（观测/驱动外设位）。
	OnProcess func(c *Core)
	// OnShutdown 停机清理（官方 FxModule_CoreShutdown 序）。
	OnShutdown func(c *Core)
}

// Core 复刻官方 CCore（FxCore.dll 内核运行时）。
// 实现 fsloader.Core——装载器泵循环逐拍调 Process()（官方 vtable+0x4c
// = CCore::Execute）。
type Core struct {
	Config *fsloader.CoreConfig

	Timer    *GameTimer
	Entities *EntityStore
	Async    *AsyncManager
	Scripts  *ScriptManager
	Log      *LogTool

	registry *fsloader.ModuleRegistry
	hooks    Hooks

	// Main 主实体（官方 core+0x314 / 全局 main_id 镜像）。
	Main *Entity

	// 时钟组（官方 +0x290 frameClock / +0x2a0 total / +0x2a8 last /
	// +0x2b0 virtual）。
	frameClock  float64 // 定步帧钟（fpsCap 模式）
	totalTime   float64 // 累计真实秒
	lastElapsed float64 // 上帧 dt
	virtualTime float64 // 虚拟累计（帧回调后）

	fpsCap int // >0=定步（1/fps 步进，每拍最多补 10 帧）

	lastReal float64 // 真实钟上帧时刻（dt 实测基准）
	stats    Stats

	modules []string // ';' 拆分结果（官方 Initialize 位）
	quit    bool     // 官方 +0x04 停机标志
	started bool
}

// New 创建内核（官方 new CCore + Initialize(cfg) 合一）。
// registry 供模块装载（官方 CDllManager 在 core 内——Go 侧由
// fsloader.ModuleRegistry 承担，Initialize 处拆分 ';'-模块表）。
func New(registry *fsloader.ModuleRegistry, hooks Hooks) *Core {
	return &Core{
		Timer:    NewGameTimer(),
		Entities: NewEntityStore(),
		Async:    NewAsyncManager(),
		Scripts:  NewScriptManager(nil),
		registry: registry,
		hooks:    hooks,
	}
}

// Type 实现 fsloader.Core。
func (c *Core) Type() string { return CoreType }

// Version 实现 fsloader.Core。
func (c *Core) Version() int { return CoreVersion }

// SetFPSCap 定步上限（官方 core+0x284；0=全速——ctor 缺省）。
func (c *Core) SetFPSCap(n int) { c.fpsCap = n }

// FPSCap 当前定步上限。
func (c *Core) FPSCap() int { return c.fpsCap }

// StatsView 统计快照（官方 SaveLogList/DumpFile 面）。
func (c *Core) StatsView() Stats { return c.stats }

// Registry 模块注册表（装配面回读）。
func (c *Core) Registry() *fsloader.ModuleRegistry { return c.registry }

// TotalTime 累计运行秒（官方 +0x2a0 total 面）。
func (c *Core) TotalTime() float64 { return c.totalTime }

// Quit 官方 nx_quit：请求停机（Execute 下一拍返回 false）。
func (c *Core) Quit() { c.quit = true }

// Startup 复刻 CCore::Startup(cfg)（官方序：VFS→脚本系统→模块→
// 主实体→主脚本→主逻辑；错误文案逐字对齐）。
func (c *Core) Startup(cfg *fsloader.CoreConfig) error {
	c.Config = cfg
	c.Log = NewLogTool("trace.log", cfg.DebugLog)
	c.Timer.Reset() // 官方 Initialize 位：时钟起点归零
	c.lastReal = 0

	// ① 文件系统（FileSystem 空=内置 OS 文件系统——官方缺省直通）。
	if cfg.FileSystem != "" {
		if fi, err := os.Stat(cfg.FileSystem); err != nil || fi.IsDir() {
			return fmt.Errorf("(CCore::Startup)file system %s not found", cfg.FileSystem)
		}
		if cfg.FileConfig != "" {
			if _, err := os.Stat(cfg.FileConfig); err != nil {
				return fmt.Errorf("(CCore::Startup)file system %s load config %s fail",
					cfg.FileSystem, cfg.FileConfig)
			}
		}
	}

	// ② 模块装载（官方 Initialize 拆 ';' 表 → Startup 循环，上限 100）。
	if err := c.loadModules(cfg.Modules); err != nil {
		return err
	}
	if c.hooks.OnModulesLoaded != nil {
		if err := c.hooks.OnModulesLoaded(c); err != nil {
			return err
		}
	}

	// ③ 主实体（官方 CreateEntity(MainEntity)→core+0x314）。
	mainName := cfg.MainEntity
	if mainName == "" {
		mainName = "MainCon" // FsLoader.ini 生产缺省
	}
	e, err := c.Entities.Create(mainName, mainName, nil)
	if err != nil {
		return fmt.Errorf("(CCore::Startup)create main entity failed: %s (%v)",
			mainName, err)
	}
	c.Main = e
	c.Entities.SetSingle(e)
	e.core = c // 主实体反引用（官方 IEntity::GetCore→g_pCore；FsSoloLoader 帧契约位）
	if c.hooks.OnMainEntity != nil {
		if err := c.hooks.OnMainEntity(c, e); err != nil {
			return fmt.Errorf("(CCore::Startup)create main entity failed: %w", err)
		}
	}

	// ④ 主脚本（官方默认名 "main"——0x101b3af0 字面量；ScriptPath 下
	// <name>.lua，登记进 CScpManager 供 mtime 热重载）。
	scriptName := cfg.MainScript
	if scriptName == "" {
		scriptName = "main"
	}
	scriptFile := scriptName + ".lua"
	if cfg.ScriptPath != "" {
		scriptFile = filepath.Join(cfg.ScriptPath, scriptFile)
	}
	if _, err := os.Stat(scriptFile); err != nil {
		return fmt.Errorf("(CCore::Startup)run main script failed: %s", scriptFile)
	}
	script, err := c.Scripts.AddScript(scriptName, scriptFile)
	if err != nil {
		return fmt.Errorf("(CCore::Startup)run main script failed: %s", scriptFile)
	}
	if c.hooks.OnMainScript != nil {
		if err := c.hooks.OnMainScript(c, script); err != nil {
			return fmt.Errorf("(CCore::Startup)run main script failed: %w", err)
		}
	}

	// ⑤ 主逻辑（可选；非空须钩子承接——官方 BindLogic）。
	if cfg.MainLogic != "" {
		if c.hooks.OnMainLogic == nil {
			return fmt.Errorf("(CCore::Startup)bind main logic failed: logic %s not found",
				cfg.MainLogic)
		}
		if err := c.hooks.OnMainLogic(c, e); err != nil {
			return fmt.Errorf("(CCore::Startup)bind main logic failed: %w", err)
		}
	}

	c.started = true
	c.Log.Trace("CCore::Startup ok: main=%s script=%s modules=%d",
		mainName, scriptName, len(c.modules))
	return nil
}

// loadModules 复刻官方模块装载（拆 ';' + 循环 + 上限 100；
// 空表=全部注册模块——官方 loadall 语义）。装载成功即把模块实体
// creator 收编进实体表（官方 CDllManager::Load → CEntManager::AddCreator）。
func (c *Core) loadModules(mods []string) error {
	c.modules = nil
	if len(mods) == 0 {
		mods = c.registry.Registered()
	}
	for i, m := range mods {
		if i >= 100 {
			break // 官方 [ebp-0x174] < 0x64 上限
		}
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		mod, err := c.registry.Load(m, c.coreView())
		if err != nil {
			return fmt.Errorf("(CCore::Startup)load module failed: %w", err)
		}
		if err := RegisterEntities(c.Entities, mod.Space(), mod.Entities()); err != nil {
			return fmt.Errorf("(CCore::Startup)load module failed: %w", err)
		}
		c.modules = append(c.modules, m)
	}
	return nil
}

// RegisterEntities 模块实体声明收编（fsloader.EntityDecl=fskit 声明面
// → EntInfo；属性/方法名链转声明表，重名由 AddCreator 拒收）。
func RegisterEntities(store *EntityStore, space string, decls []fsloader.EntityDecl) error {
	for _, d := range decls {
		info := &EntInfo{
			TypeName: d.TypeName,
			Space:    space,
			New:      d.New,
		}
		for _, p := range d.Props {
			info.Props = append(info.Props, PropDecl{Name: p, Type: "string", Get: true, Set: true})
		}
		for _, m := range d.Methods {
			info.Methods = append(info.Methods, MethodDecl{Name: m})
		}
		if err := store.AddCreator(info); err != nil {
			return err
		}
	}
	return nil
}

// Process 复刻 CCore::Execute（官方 vtable+0x4c，泵每拍）：
// 定步/全速双模式 + 异步两遍 + 脚本热重载 + 五钟统计。false=请求退出。
func (c *Core) Process() bool {
	if !c.started || c.quit {
		return false
	}
	c.stepFrame()
	return true
}

// WantQuit 官方 ICore vtbl+0x40（FsSoloLoader 泵循环逐拍查询；
// true=装载器退出泵——与 FsLoader 的 +0x4c 泵"true=继续"相反）。
func (c *Core) WantQuit() bool { return c.quit }

// MainEntity 官方 ICore vtbl+0x78（返回 core+0x314 主实体；
// FsSoloLoader 以非空为装载门，null → "main entity error"）。
func (c *Core) MainEntity() *Entity { return c.Main }

// stepFrame 一拍实体（Process 的主体；实体帧契约（vtbl+0xc）由此承接：
// dt=0 语义 = 内核以 GameTimer 实测时差步进——官方 GameTimer.Update→
// Execute(0) 同源）。
func (c *Core) stepFrame() {
	t0 := time.Now()

	// 真实 dt（官方 GameTimer.Update→Execute(0)；钳制防大跳）。
	raw := c.Timer.Now() - c.lastReal
	c.lastReal += clampFrame(raw)
	dt := c.Timer.Execute(clampFrame(raw))
	if dt < 0 {
		dt = 0
	}

	// 三钟推进（官方 totalTime/virtualTime/lastElapsed）。
	c.totalTime += dt
	c.virtualTime += dt
	c.lastElapsed = dt

	// 异步第一遍（事件派发 + 过程运行）。
	c.Async.ExecutePass1()

	if c.fpsCap > 0 {
		// 定步模式：step=1/fps，虚拟时间驱动帧钟，每拍最多补 10 帧。
		step := 1.0 / float64(c.fpsCap)
		for i := 0; i < 10; i++ {
			if c.frameClock > c.virtualTime {
				break
			}
			tt := time.Now()
			c.Entities.executeFrames(c, step)
			c.stats.Frame += time.Since(tt).Seconds()
			if i == 9 {
				c.frameClock = c.virtualTime // 官方第 10 次强制对齐
			} else {
				c.frameClock += step
			}
		}
		offset := c.virtualTime + step - c.frameClock
		if offset < 0 {
			offset = 0 // 官方 assert(offset_seconds >= 0)
		}
		to := time.Now()
		c.Entities.executeFrames(c, offset) // 尾帧余量
		c.stats.Offset += time.Since(to).Seconds()
	} else {
		// 全速模式：实测 dt 一步。
		tt := time.Now()
		c.Entities.executeFrames(c, dt)
		c.stats.Frame += time.Since(tt).Seconds()
	}

	// 异步第二遍（清除终止过程）+ 实体延迟销毁。
	c.Async.ExecutePass2()
	c.Entities.FlushDeleted()

	// 脚本热重载（[Script] Reload=1 守门；官方每拍 ReloadScript）。
	if c.Config.ScriptReload {
		if n := c.Scripts.Execute(); n > 0 {
			c.Log.Trace("script reload: %d script(s)", n)
		}
	}

	c.stats.Total += time.Since(t0).Seconds()
	if c.hooks.OnProcess != nil {
		c.hooks.OnProcess(c)
	}
}

// Shutdown 复刻 FxModule_CoreShutdown：停机标志 + 清场。
func (c *Core) Shutdown() {
	c.quit = true
	c.Entities.FlushDeleted()
	if c.hooks.OnShutdown != nil {
		c.hooks.OnShutdown(c)
	}
	if c.Log != nil {
		c.Log.Trace("CCore shutdown: total=%.3f stats[%s]", c.totalTime, c.stats)
	}
}

var _ fsloader.Core = (*Core)(nil)

// coreView 供模块 Init 的内核窄面（官方 FxModule_Init(g_core) 注入 ICore*；
// Go 侧暴露 fxcore.Core 本体——模块经 Init 直接可用全部管理器）。
func (c *Core) coreView() *Core { return c }

// Drive 把 engine.App 挂上定步泵（AddFrameExecute→TickOnce(step)；
// 分布式角色的 System 集由此驱动——官方 Lua main 装配的 Go 等价）。
func (c *Core) Drive(app *engine.App, name string) {
	c.Entities.AddFrameExecute(name, func(core *Core, step float64) {
		app.TickOnce(step)
	})
}

// NewEngineApp 便捷构造（官方角色进程 World+Scheduler+Bus 标配）。
func NewEngineApp(tps int) *engine.App { return engine.New(engine.Options{TPS: tps}) }
