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
//
// 其余角色（fsentry/fssoloentry/fslister/fssololister/fsserver/…）按需
// 增补——装载器本体与发现序零改动（ini 名即角色）。
package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/lizhanyin/9yin-server-go/internal/fscomm"
	"github.com/lizhanyin/9yin-server-go/internal/fscollide"
	"github.com/lizhanyin/9yin-server-go/internal/fskit"
	"github.com/lizhanyin/9yin-server-go/internal/fsloader"
	"github.com/lizhanyin/9yin-server-go/internal/fsmember"
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
	} {
		if err := registry.Register(m); err != nil {
			return err
		}
	}
	return nil
}

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
	roleMu     sync.Mutex
	assembled  *fsmember.Assembly
)

// AssembleMemberRole 在主实体钩子处按已装载模块集装配 member 角色；
// 返回 nil 表示本进程不是 member 角色（模块集不含 fsmember 家族）。
// 幂等：重复调用返回首次装配产物。
func AssembleMemberRole(core *fxcore.Core, cfg *fsloader.CoreConfig) *fsmember.Assembly {
	roleMu.Lock()
	defer roleMu.Unlock()
	if assembled != nil {
		return assembled
	}
	if !hasMemberModule(cfg.Modules) {
		return nil
	}
	// 官方 main.lua 以 IniDocument 二次打开 GameMember.ini；Go 侧等价：
	// ini 文件重读全量方言（Raw 兜底）。
	mcfg := fsmember.ConfigFromRawIni(cfg.Raw)
	if data, err := os.ReadFile(cfg.IniName); err == nil {
		if parsed, err := fsmember.ParseGameMemberINI(data); err == nil {
			mcfg = parsed
		}
	}
	logf := func(format string, args ...any) { fmt.Printf(format+"\n", args...) }
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
	roleMu.Unlock()
	if asm != nil {
		asm.Stop()
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

// ---- 实体面占位（官方实体属性/方法表的类型面；行为由装配器实例承担） ----

type adapterFace struct {
	Address                        string
	Port, ConnectMax, WorkThreads  int
	InBufLen, OutBufLen            int
	Closed                         bool
}

type echoFace struct {
	Address    string
	Port       int
	Connected  bool
}

type broadFace struct {
	BindAddress, BroadcastAddress string
	BroadcastPort                 int
}

type appFace struct {
	AppName, AppAddress string
}

type logFace struct {
	FilePrefix              string
	TraceMax                int
	EveryDayAlone, Console  bool
}

type logicFace struct{ Loaded int }

type collideFace struct{ TotalScenes, TotalBlocks int }

// fscomm/fscollide 面引用位（占位类型与真实包同源同构声明）。
var (
	_ = fscomm.AdapterOptions{}
	_ = fscollide.Manager{}
)
