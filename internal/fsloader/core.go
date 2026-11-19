package fsloader

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/ecs"
	"github.com/lizhanyin/9yin-server-go/internal/engine"
	"github.com/lizhanyin/9yin-server-go/internal/proc"
)

// FxCore 内核装载协议常量（app_load_core_dll 双门）。
const (
	// CoreType 官方 FxModule_GetType 须返回 "FxCore"。
	CoreType = "FxCore"
	// CoreVersion 官方装载器期望的内核版本（0x3EB，反汇编 cmp eax, 0x3EB）。
	CoreVersion = 1003
)

// Core 复刻官方 ICore/FxCore.dll 的内核面：装载器只依赖这四个方法
//（对应 FxModule_CoreInitialize/CoreShutdown + CCore::Startup + 泵方法
// vtable+0x4c）。
type Core interface {
	// Type 官方 FxModule_GetType（装载器校验 CoreType）。
	Type() string
	// Version 官方 FxModule_GetVersion（装载器校验 CoreVersion）。
	Version() int
	// Startup 官方 CCore::Startup(config)：VFS→模块→主实体→主逻辑→主脚本。
	Startup(cfg *CoreConfig) error
	// Process 官方泵方法（装载器主循环逐次调用；false=内核请求退出）。
	Process() bool
	// Shutdown 官方 FxModule_CoreShutdown：停机清理。
	Shutdown()
}

// CoreHooks 业务挂载点：集群角色 cmd 在 EngineCore 装配序列的各阶段
// 注入行为（对应官方由 Lua main 脚本完成的装配——Lua 行为层为后续
// 里程碑，Go 侧暂以原生钩子等价）。
type CoreHooks struct {
	// OnModulesLoaded 模块装载完成后（官方 main 脚本运行前的最后装配点）。
	OnModulesLoaded func(core *EngineCore, cfg *CoreConfig) error
	// OnMainEntity 主实体创建后（官方 nx_create(MainEntity) 完成位）。
	OnMainEntity func(core *EngineCore, entity ecs.EntityID, cfg *CoreConfig) error
	// OnMainLogic 主逻辑绑定（官方 BindLogic；MainLogic 非空时必须有钩子，
	// 否则 Startup 报 "bind main logic failed"）。
	OnMainLogic func(core *EngineCore, entity ecs.EntityID, cfg *CoreConfig) error
	// OnMainScript 主脚本阶段：定位成功后调用；返回 error 即官方
	// "run main script failed" 致命位（默认实现仅定位并报告 Lua 层未接线）。
	OnMainScript func(core *EngineCore, scriptPath string, cfg *CoreConfig) error
	// OnProcess 每泵一拍（dt 秒；主实体/网关等周期驱动的挂载位）。
	OnProcess func(core *EngineCore, dt float64)
	// OnShutdown 停机清理（官方 CoreShutdown 序）。
	OnShutdown func(core *EngineCore)
}

// EngineCore 复刻 FxCore.dll 的 CCore 运行时：engine.App（World+
// Scheduler+Bus）装配 + 模块注册表 + 主实体。装载器泵循环逐拍驱动
// Scheduler（官方 GameTimer 驱动语义：Process 一次 = 一帧）。
type EngineCore struct {
	App      *engine.App
	Registry *ModuleRegistry
	Hooks    CoreHooks

	// Config Startup 快照（子进程/模块可回读）。
	Config *CoreConfig
	// MainID 主实体句柄（官方全局 main_id；未创建=ecs.NullEntity）。
	MainID ecs.EntityID

	// ScriptEnv 主脚本解析结果（"未执行"时为延迟说明）。
	ScriptEnv *ScriptEnvironment

	lastTime  time.Time
	stopped   bool
	onStopFns []func()
}

// NewEngineCore 创建内核实例（CoreInitialize 的产物形态）。
func NewEngineCore(registry *ModuleRegistry, hooks CoreHooks) *EngineCore {
	return &EngineCore{
		App:      engine.New(engine.Options{TPS: 0}), // 泵循环由装载器驱动
		Registry: registry,
		Hooks:    hooks,
		MainID:   ecs.NullEntity,
	}
}

// Type 实现 Core。
func (c *EngineCore) Type() string { return CoreType }

// Version 实现 Core。
func (c *EngineCore) Version() int { return CoreVersion }

// OnStop 注册停机钩子（官方 CoreShutdown 清理序的 Go 面）。
func (c *EngineCore) OnStop(f func()) { c.onStopFns = append(c.onStopFns, f) }

// Startup 复刻 CCore::Startup(config) 五段序列；任一段失败返回带
// 官方文案的致命错误（壳层直接终止进程）。
func (c *EngineCore) Startup(cfg *CoreConfig) error {
	c.Config = cfg

	// ① 文件系统：FileSystem 空=内置 OS 文件系统（官方缺省直通）；
	// 显式命名须存在，FileConfig 为其配置文件。
	if err := c.startupFileSystem(cfg); err != nil {
		return err
	}

	// ② 模块装载：[Module] 空=全部（官方 loadall 语义）。
	if _, err := c.Registry.LoadAll(cfg.Modules, c); err != nil {
		return err
	}
	if c.Hooks.OnModulesLoaded != nil {
		if err := c.Hooks.OnModulesLoaded(c, cfg); err != nil {
			return err
		}
	}

	// ③ 主实体：官方 CEntFactory::Create(MainEntity)。
	entity := c.App.World.Create()
	c.MainID = entity
	if c.Hooks.OnMainEntity != nil {
		if err := c.Hooks.OnMainEntity(c, entity, cfg); err != nil {
			return fmt.Errorf("(CCore::Startup)create main entity failed: %w", err)
		}
	}

	// ④ 主逻辑：MainLogic 空=不绑定（FsLoader.ini 生产形态即空）；
	// 非空须由钩子/模块逻辑注册表承接，否则官方 "bind main logic failed"。
	if cfg.MainLogic != "" {
		if c.Hooks.OnMainLogic == nil {
			return fmt.Errorf("(CCore::Startup)bind main logic failed: logic %s not found", cfg.MainLogic)
		}
		if err := c.Hooks.OnMainLogic(c, entity, cfg); err != nil {
			return fmt.Errorf("(CCore::Startup)bind main logic failed: %w", err)
		}
	}

	// ⑤ 主脚本：ScriptPath 下定位 <MainScript>.lua；缺文件=官方致命位。
	if cfg.MainScript != "" {
		scriptPath := filepath.Join(cfg.ScriptPath, cfg.MainScript+".lua")
		if _, err := os.Stat(scriptPath); err != nil {
			return fmt.Errorf("(CCore::Startup)run main script failed: %s", scriptPath)
		}
		c.ScriptEnv = &ScriptEnvironment{
			Name: cfg.MainScript, File: scriptPath,
			Deferred: "Lua behavior layer is a future milestone; located but not executed",
		}
		if c.Hooks.OnMainScript != nil {
			if err := c.Hooks.OnMainScript(c, scriptPath, cfg); err != nil {
				return fmt.Errorf("(CCore::Startup)run main script failed: %w", err)
			}
		}
	}

	c.lastTime = time.Now()
	return nil
}

// ScriptEnvironment 主脚本定位结果（Lua 层未接线期间的状态说明）。
type ScriptEnvironment struct {
	Name     string // 脚本名（如 "main"）
	File     string // 绝对路径
	Deferred string // 延迟执行说明（nil 语义=已由钩子接管执行）
}

// startupFileSystem 复刻 CCore::Startup 文件系统段：
// "(CCore::Startup)file system %s not found" / "load config %s fail"。
func (c *EngineCore) startupFileSystem(cfg *CoreConfig) error {
	if cfg.FileSystem == "" {
		return nil // 内置 OS 文件系统（官方空值直通）
	}
	if fi, err := os.Stat(cfg.FileSystem); err != nil || fi.IsDir() {
		return fmt.Errorf("(CCore::Startup)file system %s not found", cfg.FileSystem)
	}
	if cfg.FileConfig != "" {
		if _, err := os.Stat(cfg.FileConfig); err != nil {
			return fmt.Errorf("(CCore::Startup)file system %s load config %s fail",
				cfg.FileSystem, cfg.FileConfig)
		}
	}
	return nil
}

// Process 复刻官方泵方法（装载器 `while(!quit && core->Process())`）：
// 一拍 = 实测 dt 驱动 Scheduler 一帧（全速语义，Sleep 节流在装载器侧）。
func (c *EngineCore) Process() bool {
	if c.stopped {
		return false
	}
	now := time.Now()
	dt := now.Sub(c.lastTime).Seconds()
	c.lastTime = now
	c.App.TickOnce(math.Min(dt, 0.25))
	if c.Hooks.OnProcess != nil {
		c.Hooks.OnProcess(c, dt)
	}
	return true
}

// Shutdown 复刻 FxModule_CoreShutdown：停机标志 + 钩子逆序。
func (c *EngineCore) Shutdown() {
	if c.stopped {
		return
	}
	c.stopped = true
	if c.Hooks.OnShutdown != nil {
		c.Hooks.OnShutdown(c)
	}
	for i := len(c.onStopFns) - 1; i >= 0; i-- {
		c.onStopFns[i]()
	}
	c.App.Bus.Close()
}

var _ Core = (*EngineCore)(nil)

// NewInprocBus 进程内总线构造暴露（模块/钩子取 Bus 用；
// EngineCore.App.Bus 已由 engine.New 装配，此函数供外部独立组装配）。
func NewInprocBus() proc.Bus { return proc.NewInprocBus() }
