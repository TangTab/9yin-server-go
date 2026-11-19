# FsLoader.exe 官方全解与 Go 复刻

> 权威快照：`E:\Workspace\project\game\九阴真经ol\BGserverCJ\Debug\FsLoader.exe`
>（585,728 B，2014-03-22）+ `FsLoader.pdb`（2.2MB）+ 同目录 `FsLoader.ini`。
> 同源兄弟 `FsSoloLoader.exe`（589,824 B）带 `.map`（3188 行符号表），
> 装载器代码结构以 map 定案、FsLoader.exe 以反汇编（capstone，工具
> `/tmp/fsloader_analysis/disas.py`）逐段对齐。内核侧 `FxCore.dll`
>（1.47MB + 5.16MB pdb）以导出表 + 字符串面 + PDB 符号定性。
> Go 复刻：`internal/fsloader` + `cmd/loader`（§7 为验收清单）。

## 1. 进程定位与部署形态

FsLoader.exe 是 Fs 框架的**通用进程装载器**：一个 16KB 级 main.cpp 薄壳
（源码路径串定案：`d:\blackgoldserverengine\serverengine\branches\
server(20120807)\fsloader\main.cpp`），本身不含任何业务；它装载
FxCore.dll（引擎内核），把 ini 配置交给内核，然后跑泵循环。

部署上**一套装载器多角色复用**：

| 启动方式 | ini | 角色 |
|---|---|---|
| `FsLoader.exe`（裸跑） | FsLoader.ini | 默认形态（FsKit 单模块） |
| `FsLoader.exe GameLister.ini`（GameLister.bat） | GameLister.ini | 大写系集群 lister |
| `FsLoader.exe GameEntry.ini`（GameEntry.bat） | GameEntry.ini | 大写系集群 entry |
| `game_lister.exe`（裸跑，独立构建） | game_lister.ini | 小写系 solo lister |

md5 定案：`game_lister.exe` == `game_entry.exe`（195e1ce1…，同一构建改名
部署）；FsSoloLoader.exe（c2f2df73…）与 FsLoader.exe（10a98d43…）为
各自独立构建。**ini 名即角色**——argv[1] 覆盖是官方 bat 的标准用法。

## 2. 装载器构成（FsSoloLoader.map 定案）

| obj | 函数 | 角色 |
|---|---|---|
| Main.obj | `_main`、`app_load_core_dll`、`app_load_core_ini`、`CCoreConfig`、`console_ctrl_handler`、`DisableSetUnhandledExceptionFilter`、`Port_Dll*`、`_AssertionFail` | 五步序列主体 |
| AppTool.obj | `app_get_ini_name`、`app_get_work_path`、`app_show_fatal_error`、`app_set_console_color` | 工具 |
| Catcher.obj | `DumpStack`、`ErrorReporter` | 崩溃栈 |
| MiniDump.obj | `Ca_UnhandledExceptionFilter{,Full}`、`InitMiniDump{,FullMemory}` | dbghelp 崩溃转储 |
| IniFile.obj | CIniFile 全套（与 FsKit.dll 内嵌同源，见 FsKit 全解 §2） | 官方 INI 方言 |

静态链接 CRT Debug（TVarList/TFastStr 内嵌），导入表仅 KERNEL32/
USER32/ADVAPI32/dbghelp/PSAPI。

## 3. _main 五步序列（FsLoader.exe 0x4449b0 反汇编定案）

```c
int main(int argc, char* argv[]) {
    // ① 控制台 Ctrl 处理：SetConsoleCtrlHandler(handler, TRUE)，失败即退
    // ② 工作目录锚定：app_get_work_path + 尾 '\\' + Port_SetCurrentDir
    //    （装载器以 exe 所在目录为 cwd，ini/模块/脚本相对路径的锚点）
    // ③ 装载内核：app_load_core_dll("FxCore.dll")，失败 →
    //    app_show_fatal_error("Load core dll failed") + exit
    // ④ ini 发现：
    //    app_get_ini_name(argv[0], buf, 0x100)   // "<exe基名>.ini"
    //    if (buf空) strncpy(buf, "FsLoader.ini") // 回退字面量
    //    if (argc > 1) strncpy(buf, argv[1])     // ★ argv[1] 覆盖
    //    CCoreConfig config;
    //    for (i=2; i<argc; i++) config追加(argv[i])  // argv[2..] 附加参数
    //    if (!app_load_core_ini(buf, &config)) fatal("Load core ini failed")
    // ⑤ 生命周期：
    //    g_core = FxModule_CoreInitialize(&config)   // 失败 "Core initialize failed"
    //    if (!FxModule_CoreStartup(g_core, …)) fatal("Core startup failed")
    //    while (!g_quit && g_core->vtbl_0x4c()) {}   // ★ 泵循环在装载器内
    //        // Sleep 节流：config.sleep > 0 时每拍 Sleep(ms)（<0 不休眠）
    //    FxModule_CoreShutdown(g_core)
}
```

- `g_quit`（0x4c4338）由 `console_ctrl_handler` 置位：CTRL_C/BREAK/CLOSE/
  LOGOFF/SHUTDOWN → 优雅停机（switch 跳表定案，非白名单事件忽略）。
- 泵方法 = ICore vtable+0x4c（第 19 槽），返回 false 亦退出循环。
- Core 导出解析地址存全局槽：`0x4c433c`=Initialize、`0x4c4340`=Startup、
  `0x4c4344`=Shutdown；`0x4c4334` = g_core 实例指针。

## 4. app_load_core_dll（0x443840）——内核双门

```c
HMODULE h = LoadLibrary(name);                    // "FxCore.dll"
if (!h)                     → "(LoadCoreDll)open dll failed"
fn = GetProcAddress(h,"FxModule_GetType");
if (!fn)                    → "(LoadCoreDll)FxModule_GetType not found"
if (strcmp(fn(), "FxCore")) → "(LoadCoreDll)dll type not FxCore"
fn = GetProcAddress(h,"FxModule_GetVersion");
if (!fn)                    → "(LoadCoreDll)FxModule_GetVersion not found"
if (fn() != 0x3EB /*1003*/) → "(LoadCoreDll)core version not match"
// 解析 FxModule_CoreInitialize/CoreStartup/CoreShutdown 三槽
// 任一缺失 → "(LoadCoreDll)FxModule_Core* not found" + FreeLibrary
```

**版本常量 1003（0x3EB）与类型串 "FxCore" 是装载器↔内核的硬契约。**
（模块侧契约对照：FsKit.dll 等业务模块导出 FxModule_Init/GetEntCreator
等 8 导出、类型串 "FxModule"、版本 100——由内核的 CDllManager 校验，
见 FsKit 全解 §1。）

## 5. app_load_core_ini（0x443ff0）——CCoreConfig 装配

官方 CIniFile 方言（= fskit.IniFile 逐字节同方言）解析，写入序：

| 段.键 | 缺省 | 语义 |
|---|---|---|
| System.DebugLog | 1 | 调试日志系统开关（bool） |
| System.Sleep | 0 | 主循环休眠毫秒（<0=不休眠） |
| Script.Reload | 1 | 脚本即时重载开关（bool） |
| Script.Path | — | 脚本目录；**非空必须存在**（GetFullPathName 归一 + Port_DirExists），失败 fatal `"Script path not found, %s"`；通过后**追加尾 '\\'** |
| Resource.Path | — | 同上，fatal `"Resource path not found, %s"` |
| Module（裸行表） | 空=全部 | FindSection 命中后 GetItemList 取整行 key=模块 DLL 名，**以 ';'(0x3b) 连接**存单串 |
| System.FileSystem | 空 | 文件系统名（空=默认） |
| System.FileConfig | 空 | 文件系统配置 |
| System.MainEntity | 空 | 主实体名（FsLoader.ini=MainCon / game_lister.ini=SoloLister） |
| System.MainScript | 空 | 启动脚本名（main → main.lua） |
| System.MainLogic | 空 | 启动逻辑类（生产配置均空） |

ini 文件本身装载失败 → fatal `"Load core ini failed"`。

[Module] 之外的段（[Main]/[Outer]/[Inner]/[Broadcast] 等）**装载器不读**——
它们是角色模块（如 FsSoloLister）在 Startup 后自行回读 ini 的方言。

## 6. 内核侧 FxCore.dll（装载器的对端）

FxCore.dll = CCore 运行时（PDB 符号 + 错误串定案）：

- **管理器全家**：CEntManager（实体）、CFuncManager（扩展函数）、
  CGameTimer（帧时钟）、CIntManager（接口）、CLogTool、CLogicManager、
  CLuaScript（内嵌 Lua）、CScpDebugger、CHookManager、CDllManager（模块）、
  CAsyncProc（脚本异步过程）
- **CCore::Startup(CCoreConfig&) 序列**（错误串即断言序）：
  1. 文件系统：`(CCore::Startup)file system %s not found` / `load config %s fail`
  2. 模块装载（CDllManager::Load 逐个：`module interface not found` /
     `module version not match` / `module already loaded` / `repeat entity|
     function|interface|logic name` / `load module failed`）——
     [Module] 空 = 全部（Lua 搜索路径 `.\?.dll;!\?.dll;!\loadall.dll`
     印证 loadall.dll 语义；`!` 前缀 = VFS 内挂载点）
  3. 主实体：`(CCore::Startup)create main entity failed`（CEntFactory::Create，
     全局 main_id）
  4. 主逻辑：`(CCore::Startup)bind main logic failed`（可选）
  5. 主脚本：`(CCore::Startup)run main script failed`（CLuaScript 装载
     ScriptPath\<MainScript>.lua 并执行）
- **Lua 面**：nx_* API 全家（nx_create/nx_bind_script/nx_bind_logic/
  nx_call/nx_execute/nx_property/nx_method/…），脚本可创建实体/绑逻辑/
  注册定时器——**装配逻辑住在 Lua**（testdata/lua 361 脚本）。
- **泵方法**（vtable+0x4c）：驱动 GameTimer、CAsyncProc、脚本热重载
  （[Script] Reload=1 → mtime 轮询）、实体帧事件。
- 崩溃面：`ERROR%04d%02d%02d%02d%02d%02d%03d.log`（本地时间 yyyymmdd
  HHMMSSmmm）+ `DUMP…dmp`（MiniDumpWriteDump）+ 寄存器/栈转储
  （EAX..EIP 格式串）+ `mem_fx_core.log`/`allocator.log` 泄漏检查。

## 7. Go 复刻对齐清单（internal/fsloader + cmd/loader）

| 官方 | Go | 对齐点 |
|---|---|---|
| FsLoader.exe 壳 | `Loader.Run`（loader.go） | 五步序列、退出码、fatal 文案 |
| argv 约定 | `args[0]=ini 覆盖 / args[1..]=附加` | argv[1] 覆盖 ini 名；ExtraArgs 进配置 |
| app_get_ini_name | `IniNameFromExe` | "<exe基名>.ini"；回退字面量 FsLoader.ini |
| LoadCoreDll 双门 | `Core.Type/Version` + CoreType/CoreVersion 常量 | "FxCore" / 1003(0x3EB)；三失败文案逐字 |
| app_load_core_ini | `LoadCoreConfig`（fskit.IniFile 官方方言） | 键/缺省/写入序逐项；目录存在性 fatal 文案；尾分隔符 |
| CCoreConfig | `CoreConfig` | 全 12 字段 + ExtraArgs |
| CDllManager | `ModuleRegistry` | 四重校验文案（not found/not match/already/repeat *）；空=全部 |
| FxModule 8 导出 | `Module` 接口 + `DeclModule` 适配 | Type/Version/Space/Init/Entities/Funcs |
| CCore::Startup 五段 | `EngineCore.Startup` | VFS→模块→主实体→主逻辑→主脚本；段失败文案逐字 |
| ICore 泵方法 | `EngineCore.Process`（TickOnce 一拍） | dt 实测钳制 0.25s |
| 装载器泵循环 | `Loader.pump` | `while(!quit && Process())` + Sleep>0 毫秒节流 |
| SetConsoleCtrlHandler/g_quit | `signal.NotifyContext` → quit 通道 | Ctrl+C 优雅停机 → CoreShutdown |
| CoreShutdown | `EngineCore.Shutdown` | 停机标志幂等 + 钩子逆序 + Bus 关闭 |
| ERROR*.log 崩溃面 | `WriteErrorLog`（crashlog.go） | 官方命名格式串（测试对拍 ERROR20261119080507123.log）；panic 值+goroutine 栈（DUMP*.dmp 为 Win32 专有，不复刻） |
| Port_SetCurrentDir(exe 目录) | `ChdirToExeDir` | cwd 锚定（cmd 可选采用） |

**已知延迟项**（依赖后续里程碑，非遗漏）：

- Lua 行为层（nx_* + main 脚本执行）→ 里程碑二（gopher-lua）；
  当前 `ScriptEnv` 定位主脚本并报告延迟，`CoreHooks` 以原生钩子
  承接装配位（cmd/loader 演示：模块报告/帧报/停机报）。
- CGameTimer/异步脚本/脚本热重载 → 随 Lua 层落地；当前泵循环以
  Scheduler tick 驱动等价帧语义。
- 集群角色装配（entry/lister/...）继续由各自 cmd 承担；loader 是
  通用装载面，与官方"一套 exe 多角色"形态对应的关系是 cmd 命名
  game_loader.exe + ini 名区分。

**验收记录**（2026-11-19）：

- `go test ./internal/fsloader/`：8 用例（官方方言解析/缺目录 fatal/
  ini 名派生/注册表四重校验/泵循环与停机/版本门/Startup 序列与主实体
  物化/缺脚本 fatal/日志命名）全绿。
- 端到端（build/game_loader.exe）：无参 → 释放内置官方方言 ini →
  配置装载（Sleep=1/DebugLog/MainCon/main/1 模块）→ fskit.dll 装载
  （4 实体类型+6 扩展函数）→ 主实体物化 → 泵 400ms 257 tick（Sleep=1
  节流 ✓）→ CoreShutdown 优雅退出码 0；ini 缺失 → "Load core ini
  failed" 退出码 1（官方致命路径 ✓）。
