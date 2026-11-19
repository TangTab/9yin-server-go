// roles.go 是「同名多角色」装载的模块注册面：官方 game_*.exe 为同一
// 构建改名部署，行为差异全部来自 `<exe基名>.ini`（或 argv[1]）的
// [Module] 模块集——FsLoader.exe 装载器按表 LoadLibrary，模块 CRT
// 静态构造即注册实体/函数 creator。
//
// Go 等价：ModuleRegistry（官方 CDllManager 复刻，大小写不敏感=Windows
// LoadLibrary 语义）编译期注册各模块适配器；主实体钩子
//（官方 main.lua 时机）按已装载模块集装配角色面。本文件注册：
//
//	fscomm.dll       → 通讯面实体（Adapter/EchoSock/BroadSock/AppManager/
//	                   LogPool/LogicLoader，internal/fscomm）
//	fscollide.dll    → CollideManager 实体（internal/fscollide）
//	fsmember.dll     → GameMember 实体 + member 角色装配（集群形态
//	                   GameMember.ini；internal/fsmember）
//	fssolomember.dll → SoloMember 实体 + member 角色装配（小写系
//	                   game_member.ini 裸跑形态）
//	fssoloworld.dll  → SoloWorld 实体 + world 角色装配（小写系
//	                   game_world.ini 裸跑形态）
//	fssolostore.dll  → SoloStore 实体 + store 角色装配（小写系
//	                   game_store.ini 裸跑形态；internal/fssolostore）
//
// 其余角色（fsentry/fssoloentry/fslister/fssololister/fsserver/…）按需
// 增补——装载器本体与发现序零改动（ini 名即角色）。
package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/fscollide"
	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
	"github.com/lizhanyin/9yin-server-go/internal/fskit"
	"github.com/lizhanyin/9yin-server-go/internal/fsloader"
	"github.com/lizhanyin/9yin-server-go/internal/fsmember"
	"github.com/lizhanyin/9yin-server-go/internal/fssolomember"
	"github.com/lizhanyin/9yin-server-go/internal/fssolostore"
	"github.com/lizhanyin/9yin-server-go/internal/fssoloworld"
	"github.com/lizhanyin/9yin-server-go/internal/fxcore"
)

// RegisterClusterModules 把集群家族模块注册进注册表（编译期等价于
// 官方部署目录摆放 DLL；装载时按 ini [Module] 表激活）。
func RegisterClusterModules(registry *fsloader.ModuleRegistry) error {
	for _, m := range []fsloader.Module{
		commModule(),
		collideModule(),
		memberModule("fsmember.dll", "GameMember"),
		memberModule("fssolomember.dll", "SoloMember"),
		worldModule(),
		storeModule(),
	} {
		if err := registry.Register(m); err != nil {
			return err
		}
	}
	return nil
}

// worldModule 复刻 FsSoloWorld.dll（小写系 game_world.exe 世界服）：
// SoloWorld 服务实体 creator（属性面 = §二属性表定案）。角色装配见
// AssembleWorldRole。
func worldModule() fsloader.Module {
	return &fsloader.DeclModule{
		ModuleName:  "fssoloworld.dll",
		ModuleSpace: "fssoloworld",
		ModuleEnts: []fsloader.EntityDecl{{
			TypeName: "SoloWorld",
			New:      func(*fskit.VarList) any { return worldFace{} },
			Props: []string{
				"StateLock", "UseThread", "UseQuickLog", "UseAppRecord",
				"ServerName", "DistrictId", "ServerId", "MaxPlayers",
				"SaveMinutes", "FlushMinutes", "AccountRoles",
				"RoleNameSizeLimit", "RoleProtectBase", "RoleProtectRate",
				"AutoOpen", "PlayerQueue", "VipQueue", "OnlineContinue",
				"OfflineContinue", "PasswordAnyChar", "OriginalPassword",
				"ReallyDeleteRole", "QuitOnClosed", "UseVerifyImage",
				"DenyCreateRole", "DenyEntryScene", "ClearPlayerBatchSize",
				"ClearSceneBatchSize", "GameVersion", "OfflineActiveSeconds",
				"WaitChargeSeconds", "WaitStoreSeconds", "ChargeResendSeconds",
				"LimitRoleSize", "LogInnerMessage", "InnerAddr", "InnerPort",
				"BroadAddr", "BroadPort", "EchoAddr", "EchoPort",
				"AppName", "LogLauncher", "RoleCount", "PlayerCount",
				"ActiveCount", "OnlineCount", "OfflineCount", "LoggingCount",
				"QueueCount", "VipQueueCount", "LockedCount", "State",
			},
			Methods: []string{
				"Start", "Stop", "SetAppState", "SetConsoleColor", "WriteLine",
				"SendEchoLine", "CloseEcho", "SetChargeAddress", "SetGmccAddress",
				"SetManageAddress", "SetExtraAddress", "OpenServer", "CloseServer",
				"LoadCharImage", "LoadNameFilter", "LoadPlayer", "AddScene",
				"AddPrototypeScene", "DumpPlayers", "DumpMemoryPlayers",
				"DumpScenes", "DumpPerformance", "ClearPerformance",
				"DumpReactor", "AddAccount", "ReconnectCharge",
				"debugger_get_scene_list",
			},
		}},
	}
}

// storeModule 复刻 FsSoloStore.dll（小写系 game_store.exe 存储枢纽）：
// SoloStore 服务实体 creator（属性面 = §一对象布局投影）。角色装配见
// AssembleStoreRole。
func storeModule() fsloader.Module {
	return &fsloader.DeclModule{
		ModuleName:  "fssolostore.dll",
		ModuleSpace: "fssolostore",
		ModuleEnts: []fsloader.EntityDecl{{
			TypeName: "SoloStore",
			New:      func(*fskit.VarList) any { return storeFace{} },
			Props: []string{
				"UseThread", "UseQuickLog", "UseAppRecord", "UseLogSerial",
				"LogDBPerformance", "DistrictId", "ServerId", "StoreId",
				"AccountBase", "AccountPools", "RoleBase", "RolePools",
				"LogBase", "LogPools", "PublicBase", "PublicPools",
				"ManageBase", "ManagePools", "DBSystem", "SplitWeek",
				"SqlBatchNum", "InnerAddr", "InnerPort", "BroadAddr",
				"BroadPort", "EchoAddr", "EchoPort", "WakeAddr", "WakePort",
				"AppName", "LogLauncher", "DatabaseBreak",
			},
			Methods: []string{
				"Start", "Stop", "SetAppState", "SetConsoleColor", "WriteLine",
				"SendEchoLine", "CloseEcho", "DumpReactor", "AccessEmpty",
				"AddLogTable", "AddLogField", "AddLogCreator", "OutputXml",
			},
		}},
	}
}

// storeFace SoloStore 实体静态面（装载期声明用；实例由 store 角色承担）。
type storeFace struct{}

func (storeFace) EntityInfo() string { return "SoloStore" }

// commModule 复刻 FsComm.dll：通讯面实体家族（属性面 = 官方实体属性表）。
func commModule() fsloader.Module {
	return &fsloader.DeclModule{
		ModuleName:  "fscomm.dll",
		ModuleSpace: "FsComm",
		ModuleEnts: []fsloader.EntityDecl{
			{TypeName: "Adapter", New: func(*fskit.VarList) any { return adapterFace{} },
				Props: []string{"Address", "Port", "ConnectMax", "WorkThreads", "InBufLen", "OutBufLen", "Closed"}},
			{TypeName: "EchoSock", New: func(*fskit.VarList) any { return echoFace{} },
				Props: []string{"Address", "Port", "Connected"}},
			{TypeName: "BroadSock", New: func(*fskit.VarList) any { return broadFace{} },
				Props: []string{"BindAddress", "BroadcastAddress", "BroadcastPort"}},
			{TypeName: "AppManager", New: func(*fskit.VarList) any { return appFace{} },
				Props: []string{"AppName", "AppAddress"}},
			{TypeName: "LogPool", New: func(*fskit.VarList) any { return logFace{} },
				Props: []string{"FilePrefix", "TraceMax", "EveryDayAlone", "Console"}},
			{TypeName: "LogicLoader", New: func(*fskit.VarList) any { return logicFace{} },
				Props: []string{"Loaded"}},
		},
	}
}

// collideModule 复刻 FsCollide.dll：CollideManager 实体。
func collideModule() fsloader.Module {
	return &fsloader.DeclModule{
		ModuleName:  "fscollide.dll",
		ModuleSpace: "FsCollide",
		ModuleEnts: []fsloader.EntityDecl{
			{TypeName: "CollideManager", New: func(*fskit.VarList) any { return collideFace{} },
				Props: []string{"TotalScenes", "TotalBlocks"}},
		},
	}
}

// memberModule 复刻 FsMember.dll（集群）/FsSoloMember.dll（solo）：
// 服务实体 creator（属性/方法面 = PDB prop/func_struct 定案 §3.1）。
// 角色装配见 AssembleMemberRole。
func memberModule(dllName, entityName string) fsloader.Module {
	return &fsloader.DeclModule{
		ModuleName:  dllName,
		ModuleSpace: strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(dllName), ".dll"), "member"),
		ModuleEnts: []fsloader.EntityDecl{{
			TypeName: entityName,
			New:      func(*fskit.VarList) any { return fsmember.New(1, 1, 0) },
			Props: []string{
				"DistrictId", "ServerId", "MemberId", "State", "MaxLetterNum",
				"MaxRunningTime", "PlayerSingleBuffer", "DebugHalt", "AppearAtOnce",
				"HideEmptyProperty", "ScriptReload", "TaskReload", "SightOptimize",
				"ScriptPath", "TaskPath", "CollideRootPath", "Console", "EchoSock",
				"BroadSock", "AppManager", "Adapter", "LogicLoader", "CollideManager",
			},
			Methods: []string{"Start", "Stop", "LoadNameFilter", "LoadLocations", "DumpScene", "DumpPerformance"},
		}},
	}
}

// ---- 角色装配（官方 main.lua 时机：主实体就绪后） ----

var (
	roleMu    sync.Mutex
	assembled *fsmember.Assembly
	soloSrv   *fssolomember.Server
	worldSrv  *fssoloworld.Server
	storeSrv  *fssolostore.Server
)

// AssembleWorldRole 在主实体钩子处装配 world 角色（fssoloworld.dll 小写系）；
// 模块集不含则返回 nil。幂等。
func AssembleWorldRole(core *fxcore.Core, cfg *fsloader.CoreConfig) *fssoloworld.Server {
	roleMu.Lock()
	defer roleMu.Unlock()
	if worldSrv != nil {
		return worldSrv
	}
	if !hasWorldModule(cfg.Modules) {
		return nil
	}
	logf := func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	wcfg := fssoloworld.DefaultConfig()
	if data, err := os.ReadFile(cfg.IniName); err == nil {
		if parsed, err := fssoloworld.ParseGameWorldINI(data); err == nil {
			wcfg = parsed
		}
	}
	worldSrv = fssoloworld.New(fssoloworld.Options{
		Config: wcfg,
		Logf:   logf,
		OnClosed: func() {
			logf("world server shutdown...")
			core.Quit()
		},
	})
	if err := worldSrv.Start(); err != nil {
		logf("(world role)start failed: %v", err)
		worldSrv = nil
		return nil
	}
	logf("app name %s", wcfg.AppName())
	logf("inner socket %s:%d", wcfg.InnerAddr, worldSrv.InnerPort())
	return worldSrv
}

// AssembleStoreRole 在主实体钩子处装配 store 角色（fssolostore.dll
// 小写系 game_store.ini 裸跑形态）；模块集不含则返回 nil。幂等。
func AssembleStoreRole(core *fxcore.Core, cfg *fsloader.CoreConfig) *fssolostore.Server {
	roleMu.Lock()
	defer roleMu.Unlock()
	if storeSrv != nil {
		return storeSrv
	}
	if !hasStoreModule(cfg.Modules) {
		return nil
	}
	logf := func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	scfg := fssolostore.DefaultConfig()
	if data, err := os.ReadFile(cfg.IniName); err == nil {
		if parsed, err := fssolostore.ParseGameStoreINI(data); err == nil {
			scfg = parsed
		}
	}
	storeSrv = fssolostore.New(fssolostore.Options{
		Config:  scfg,
		Backend: fssolostore.NewMemoryStore("game_store_state.json"),
		Logf:    logf,
		OnStateClosed: func() {
			logf("store server shutdown...")
			core.Quit()
		},
	})
	if err := storeSrv.Start(); err != nil {
		logf("(store role)start failed: %v", err)
		storeSrv = nil
		return nil
	}
	rt := storeSrv.Config()
	logf("app name %s", rt.AppName)
	logf("inner socket %s:%d", rt.InnerAddr, rt.InnerPort)
	return storeSrv
}

// hasStoreModule 模块集是否为 fssolostore.dll（小写系 store 形态）。
func hasStoreModule(modules []string) bool {
	for _, m := range modules {
		if strings.EqualFold(m, "fssolostore.dll") {
			return true
		}
	}
	return false
}

// hasWorldModule 模块集是否为 fssoloworld.dll（小写系 world 形态）。
func hasWorldModule(modules []string) bool {
	for _, m := range modules {
		if strings.EqualFold(m, "fssoloworld.dll") {
			return true
		}
	}
	return false
}

// AssembleMemberRole 在主实体钩子处按已装载模块集装配 member 角色；
// 返回 nil 表示本进程不是 member 角色（模块集不含 fsmember 家族）。
// 幂等：重复调用返回首次装配产物。fssolomember.dll（小写系）→
// internal/fssolomember solo 装配；fsmember.dll（大写系）→ 集群装配。
func AssembleMemberRole(core *fxcore.Core, cfg *fsloader.CoreConfig) *fsmember.Assembly {
	roleMu.Lock()
	defer roleMu.Unlock()
	if assembled != nil {
		return assembled
	}
	if !hasMemberModule(cfg.Modules) {
		return nil
	}
	logf := func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
	if hasSoloMemberModule(cfg.Modules) {
		// 小写系：game_member.ini 裸跑形态（SoloMember 实体）
		scfg := fssolomember.DefaultConfig()
		if data, err := os.ReadFile(cfg.IniName); err == nil {
			if parsed, err := fssolomember.ParseGameMemberINI(data); err == nil {
				scfg = parsed
			}
		}
		soloSrv = fssolomember.New(fssolomember.Options{
			Config: scfg,
			Logf:   logf,
			OnExit: core.Quit,
		})
		if err := soloSrv.Start(); err != nil {
			logf("(member role)solo start failed: %v", err)
			soloSrv = nil
			return nil
		}
		return nil // solo 形态无集群 Assembly（返回值仅集群角色使用）
	}
	// 官方 main.lua 以 IniDocument 二次打开 GameMember.ini；Go 侧等价：
	// ini 文件重读全量方言（Raw 兜底）。
	mcfg := fsmember.ConfigFromRawIni(cfg.Raw)
	if data, err := os.ReadFile(cfg.IniName); err == nil {
		if parsed, err := fsmember.ParseGameMemberINI(data); err == nil {
			mcfg = parsed
		}
	}
	asm, err := fsmember.Assemble(nil, mcfg, fsmember.AssembleOptions{
		Logf:   logf,
		OnQuit: core.Quit, // 官方 nx_quit()：echo quit → 泵循环退出 → 进程终
	})
	if err != nil {
		logf("(member role)assemble failed: %v", err)
		return nil
	}
	if err := asm.Start(); err != nil {
		logf("(member role)start failed: %v", err)
		return nil
	}
	assembled = asm // 停机序经 StopMemberRole 挂进 OnShutdown 钩子
	return asm
}

// StopMemberRole 停 member 角色（官方 close() 停机序；幂等）。
func StopMemberRole() {
	roleMu.Lock()
	asm := assembled
	solo := soloSrv
	world := worldSrv
	roleMu.Unlock()
	if asm != nil {
		asm.Stop()
	}
	if solo != nil {
		solo.Stop()
	}
	if world != nil {
		world.Stop()
	}
}

// hasMemberModule 模块集是否含 member 家族（大小写不敏感，等价
// Windows LoadLibrary("FsMember.dll")=="fsmember.dll"）。
func hasMemberModule(modules []string) bool {
	for _, m := range modules {
		switch strings.ToLower(m) {
		case "fsmember.dll", "fssolomember.dll":
			return true
		}
	}
	return false
}

// hasSoloMemberModule 模块集是否为小写系 solo 形态（fssolomember.dll）。
func hasSoloMemberModule(modules []string) bool {
	for _, m := range modules {
		if strings.EqualFold(m, "fssolomember.dll") {
			return true
		}
	}
	return false
}

// ---- 实体面占位（官方实体属性/方法表的类型面；行为由装配器实例承担） ----

// worldFace SoloWorld 实体类型面（行为由 fssoloworld.Server 装配实例承担）。
type worldFace struct {
	StateLock, UseThread, UseQuickLog, UseAppRecord bool
	ServerName                                      string
	DistrictId, ServerId, MaxPlayers                int
	SaveMinutes, FlushMinutes, AccountRoles         int
	RoleNameSizeLimit                               int
	RoleProtectBase, RoleProtectRate                float64
	AutoOpen, PlayerQueue, VipQueue                 bool
	OnlineContinue, OfflineContinue                 bool
	PasswordAnyChar, OriginalPassword               bool
	ReallyDeleteRole, QuitOnClosed, UseVerifyImage  bool
	DenyCreateRole, DenyEntryScene                  bool
	ClearPlayerBatchSize, ClearSceneBatchSize       int
	GameVersion, OfflineActiveSeconds               int
	WaitChargeSeconds, WaitStoreSeconds             int
	ChargeResendSeconds, LimitRoleSize              int
	LogInnerMessage                                 bool
	InnerAddr, BroadAddr, EchoAddr                  string
	InnerPort, BroadPort, EchoPort                  int
	AppName, LogLauncher                            string
	RoleCount, PlayerCount, ActiveCount             int
	OnlineCount, OfflineCount, LoggingCount         int
	QueueCount, VipQueueCount, LockedCount          int
	State                                           string
}

type adapterFace struct {
	Address                       string
	Port, ConnectMax, WorkThreads int
	InBufLen, OutBufLen           int
	Closed                        bool
}

type echoFace struct {
	Address   string
	Port      int
	Connected bool
}

type broadFace struct {
	BindAddress, BroadcastAddress string
	BroadcastPort                 int
}

type appFace struct {
	AppName, AppAddress string
}

type logFace struct {
	FilePrefix             string
	TraceMax               int
	EveryDayAlone, Console bool
}

type logicFace struct{ Loaded int }

type collideFace struct{ TotalScenes, TotalBlocks int }

// fscomm/fscollide 面引用位（占位类型与真实包同源同构声明）。
var (
	_ = fscomm.AdapterOptions{}
	_ = fscollide.Manager{}
)
