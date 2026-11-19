# FsSoloLoader.exe 官方全解与 Go 复刻

> 权威快照：`F:\Game\Online\9yin\BGserverCJ\Debug\FsSoloLoader.exe`
>（c2f2df73…，589,824 B，2013-07-05 构建）+ `FsSoloLoader.map`（3188 行
> 符号表）+ `FsSoloLoader.pdb`。源码树（map/断言路径串定案）：
> `c:\wuyf\svn\serveriii\branches\publish20130508\server\fssololoader\main.cpp`
> + `server\tools\apptool.cpp` + `server\utils\inifile.cpp`。
> 反汇编（capstone）逐段对齐；同门 FsLoader.exe 见姊妹篇
> 《FsLoader全解与Go复刻.md》。Go 复刻：`internal/fssoloader` +
> `cmd/sololoader`（§7 为验收清单）。

## 1. 进程定位与部署形态

FsSoloLoader.exe 是小写系 solo 集群的**通用进程壳**：装载 FxCore.dll
内核、交给 ini 配置、跑主实体帧泵。与 FsLoader.exe（大写系，2012 分支）
同源异代——publish20130508 分支重构了壳↔内核契约（§3/§4）。

**装载器族谱**（md5 定案，同目录）：

| exe | md5 | 大小 | 身份 |
|---|---|---|---|
| FsSoloLoader.exe | c2f2df73… | 589,824 | 本篇权威快照（独立构建） |
| game_{lister,entry,world,member,public,store,log,global,auction,unite}.exe（11 份） | 195e1ce1… | 589,824 | **生产部署形态**：同一构建改名拷贝，与 FsSoloLoader 逐点同构（_main 序列、vtable 槽位、g_running@0x4c41ec 全一致） |
| game_monitor.exe | 0a51cad1… | 589,824 | 第三构建（监控角色） |
| FsLoader.exe | 10a98d43… | 585,728 | 大写系姊妹篇（2012 分支） |
| game_robot.exe | 3eeb2dac… | 585,728 | 大写系独立构建 |

**ini 名即角色**：裸跑时 `app_get_ini_name` 从 exe 基名派生
（game_lister.exe → game_lister.ini）；`game_*.ini` 的 [Module] 表
（fskit.dll + fscomm.dll + fssolo\*.dll）与 [System] MainEntity
（SoloLister/SoloEntry/…）决定进程角色。

## 2. 装载器构成（FsSoloLoader.map 定案）

与 FsLoader 同构（Main/AppTool/Catcher/MiniDump/IniFile 五 obj +
静态链接 CRT Debug），差异仅在源码树与字面量（§3）。关键符号：

| 函数 | VA | 角色 |
|---|---|---|
| `app_get_ini_name` | 0x4447b0 | **小写**派生 ini 名 |
| `app_get_work_path` | 0x444c70 | exe 目录锚定 |
| `app_show_fatal_error` | 0x444d90 | 致命文案出口 |
| `app_load_core_dll` | 0x445220 | 内核双门 + 三槽解析 |
| `app_load_core_ini` | 0x445900 | CCoreConfig 装配（12 键白名单） |
| `console_ctrl_handler` | 0x446180 | Ctrl 事件映射（§5） |
| `_main` | 0x446420 | 五步序列 + solo 帧泵 |
| `g_running` | 0x4c41ec | .data 初值 **1**；处理器置 0 |
| 内核函数槽 | 0x4c6338/3c/40 | Initialize/Startup/Shutdown |
| `g_core` | 0x4c6334 | ICore\* 实例 |

## 3. _main 序列（0x446420 反汇编定案）与 FsLoader 差异

```c
int main(int argc, char* argv[]) {
    // ① SetConsoleCtrlHandler(handler, TRUE)——事件映射见 §5（C/BREAK 吞掉）
    // ② app_get_work_path(argv[0]) + 尾 '\\' + Port_SetCurrentDir（exe 目录锚定）
    // ③ app_load_core_dll("FxCore.dll", &Init, &Startup, &Shutdown)
    //    失败 → app_show_fatal_error("load core dll failed")（小写）
    // ④ app_get_ini_name(argv[0], buf, 0x100)：
    //        GetFullPathName → Port_StringLower（全串小写）→ 去 ".exe" → 追加 ".ini"
    //    if (buf空) buf = "fssololoader.ini"          // ★ 小写回退字面量
    //    if (argc > 1) buf = argv[1];                  // argv[1] 覆盖
    //    CCoreConfig cfg; for (i=2; i<argc; i++) cfg.ExtraArgs << argv[i];
    //    if (!app_load_core_ini(buf, &cfg)) fatal("load core ini failed")
    // ⑤ g_core = FxModule_CoreInitialize(&cfg)        // null → "core initialize failed"
    //    if (!FxModule_CoreStartup(g_core, &cfg.ExtraArgs)) fatal("core startup failed")
    //    ~CCoreConfig();
    // ⑥ IEntity* ent = g_core->vtbl[+0x78]();          // ★ 主实体门
    //    if (!ent) fatal("main entity error");         // ★ solo 独有
    // ⑦ while (g_running /*0x4c41ec，初值 1*/) {       // ★ solo 帧泵
    //        if (g_core->vtbl[+0x40]()) break;         // ★ true=退出（与 FsLoader 反向）
    //        ent->vtbl[+0xc](0.0f);                    // ★ dt 恒 0（fldz）
    //    }                                             // （循环内无 Sleep）
    // ⑧ FxModule_CoreShutdown(g_core);
    //    return 1;                                     // ★ mov eax,1（正常退出码 1）
}
```

**与 FsLoader.exe 差异定案**：

| 面 | FsSoloLoader（本篇） | FsLoader |
|---|---|---|
| ini 回退字面量 | `fssololoader.ini`（小写） | `FsLoader.ini` |
| ini 派生 | 全串**转小写**后取基名 | 不转小写 |
| 内核门文案前缀 | `(app_load_core_dll)`（=函数名） | `(LoadCoreDll)` |
| 致命文案 | 全小写（load/core/main 首词） | 首字母大写 |
| Startup 后 | 主实体门 `vtbl[+0x78]` + 帧 | 无 |
| 泵 | `while(g_running){ if(vtbl[+0x40]()) break; ent->vtbl[+0xc](0); }` | `while(!g_quit && vtbl[+0x4c]())` |
| 泵内 Sleep | 无（System.Sleep 为内核/角色方言键） | 无（壳层同样不消费） |
| Ctrl 事件 | C/BREAK **吞掉**；CLOSE/LOGOFF/SHUTDOWN 停机 | 全事件停机 |
| 正常退出码 | 1（mov eax,1） | 未定值 |
| ini 白名单 | 同 12 键（System.DebugLog/Sleep、Script.Reload/Path、Resource.Path、Module、System.FileSystem/FileConfig/MainEntity/MainScript/MainLogic）——逐字节同方言 | 同 |

双门常量两代相同：类型串 `"FxCore"`、版本 `1003 (0x3EB)`
（`cmp eax, 0x3eb` @0x445424 定案）。

## 4. 壳↔内核契约（solo 世代）

装载器与 FxCore.dll 的虚表契约定案（槽位为反汇编直接证据；语义名
为按用法的行为定性）：

| 槽位 | 用法 | 定性 |
|---|---|---|
| ICore vtbl+0x78 | `()` → IEntity\*，非空为装载门 | 取主实体（**core+0x314**——FxCore.dll 0x100d7304 处 `cmp [eax+0x314], 0` 与 CCore::Startup "create main entity failed" 同区交叉定案） |
| ICore vtbl+0x40 | `()` → bool，true → 壳层退出泵 | 停机查询（与 FsLoader 泵 +0x4c "true=继续"**方向相反**——两代契约不可混用） |
| IEntity vtbl+0xc | `(float dt)`，壳层恒传 0.0 | 实体帧方法；dt=0 = 内核以 GameTimer 实测时差步进（官方 GameTimer.Update→Execute(0) 同源） |
| FxModule_CoreStartup | `(core, IVarList* extraArgs)` 两参 | 附加参数随 Startup 二次入核 |

内核侧佐证：FxModule_CoreStartup @0x100e2990 → thunk 0x1008fa22 →
CCore::Startup @0x100d6f40（`(CCore::Startup)create main entity failed`
文案 xref）；FxModule_CoreInitialize 经单例工厂（对象 0x1000 B，
g_core@0x101efd9c）；CCoreConfig 偏移 +0x16=DebugLog、+0x18=ScriptReload
（FxModule_CoreInitialize 内 `movzx byte [eax+0x16/0x18]`）。

## 5. console_ctrl_handler（0x446180）——事件映射定案

双重间接 switch：字节索引表 @0x4461f0 `[0,0,1,2,2,1,1]`（事件 0..6）→
DWORD 目标表 @0x4461e4（三目标）：

| 事件 | 值 | 目标 | 行为 |
|---|---|---|---|
| CTRL_C_EVENT | 0 | case0 @0x4461c4 | `mov eax,1`——**吞掉**（return TRUE，不停服） |
| CTRL_BREAK_EVENT | 1 | case0 | 同上吞掉 |
| CTRL_CLOSE_EVENT | 2 | case1 @0x4461cb | `mov byte [0x4c41ec],0; mov eax,1`——**g_running=0 优雅停机** |
| （保留）3/4 | 3,4 | case2 @0x4461d9 | `xor eax,eax`——忽略（return FALSE） |
| CTRL_LOGOFF_EVENT | 5 | case1 | 停机 |
| CTRL_SHUTDOWN_EVENT | 6 | case1 | 停机 |

运维语义：**控制台 Ctrl+C/_break 被有意吞掉**（生产防误触）——停机走
关窗/注销/关机路径。g_running（.data@0x4c41ec）初值 **1**（文件字节
定案），泵循环以非零为继续条件（与 FsLoader 的 g_quit=0 初值、置 1
停机相反）。

## 6. 崩溃面

与 FsLoader 同源同构（Catcher.obj/MiniDump.obj）：`ERROR%04d%02d%02d%
02d%02d%02d%03d.log`（本地时间 yyyymmddHHMMSSmmm）+ `DUMP…dmp`
（MiniDumpWriteDump）+ 寄存器/栈转储；导出面 KERNEL32/USER32/ADVAPI32/
dbghelp/PSAPI。

## 7. Go 复刻对齐清单（internal/fssoloader + fxcore 增量 + cmd/sololoader）

| 官方 | Go | 对齐点 |
|---|---|---|
| FsSoloLoader.exe 壳 | `SoloLoader.Run`（loader.go） | 五步序列 + 主实体门 + 帧泵；致命文案逐字（小写） |
| 内核门六文案 | `Run` 前置门 | `(app_load_core_dll)open dll failed`/`dll type not FxCore`/`core version not match`（进程内形态三门；FxModule_\* 三门由导出面语义并入工厂） |
| 双门常量 | 复用 `fsloader.CoreType/CoreVersion` | "FxCore" / 1003 |
| app_get_ini_name | `IniNameFromExe` | **全串小写** + 去 .exe + .ini；game_lister.exe→game_lister.ini |
| 回退字面量 | `DefaultIniName` | `fssololoader.ini` |
| app_load_core_ini | 复用 `fsloader.LoadCoreConfig` | 12 键白名单逐字节同方言 |
| CoreStartup(core, extraArgs) | `cfg.ExtraArgs` 随 `Startup(cfg)` | argv[2..] 官方位 |
| vtbl+0x78 主实体门 | `SoloCore.MainEntity() MainEntityView` + fxcore `Core.MainEntity()`（core+0x314 镜像） | null → `main entity error` |
| vtbl+0x40 停机查询 | fxcore `Core.WantQuit()` | true=泵退出（方向与 FsLoader Process 反） |
| IEntity vtbl+0xc 帧 | fxcore `Entity.Frame(dt)`（主实体注入内核反引用） | dt=0 → stepFrame 实测时差步进（Process 主体提取） |
| 帧泵循环 | `SoloLoader.pump` | `while(g_running){ if(WantQuit) break; Frame(0) }`，无 Sleep |
| Ctrl 事件映射 | `Run` 内 signal 面 | SIGINT（Ctrl+C/BREAK）**吞掉**并记日志（`CtrlCStops` 可放开）；SIGTERM/ctx 取消 = CLOSE/LOGOFF/SHUTDOWN 停机位 |
| g_running 初值 1 | quit 通道缺省开 | ctx 取消/SIGTERM → 关闭 |
| ERROR\*.log 崩溃面 | 复用 `fsloader.WriteErrorLog` | 官方命名格式串（DUMP\*.dmp 为 Win32 专有，不复刻） |
| 正常退出码 1 | `Run` 返回 1 | mov eax,1 逐字 |
| game_\*.exe 改名部署 | `cmd/sololoader`（构建名 game-sololoader.exe，可改名） | 派生 ini 名随 exe 名；缺省释放官方方言 ini + 演示 main.lua（绝不覆盖已有） |

**已知延迟项**（依赖后续里程碑，非遗漏）：

- solo 角色模块（fssoloentry/fssololister/…dll）与 Lua 行为层 →
  随复刻进度在 cmd/sololoader 注册表增补（装载器本体零改动）；
  当前缺省注册 fskit（game_\*.ini [Module] 表首行）。
- 实体帧的模块/Lua 回调由 fxcore tick 语义承接（§4 dt=0 定案）。

**验收记录**（2026-11-19）：

- `go test ./internal/fssoloader/`：9 用例全绿（小写派生/三门文案/
  回退字面量与缺 ini fatal/主实体门/帧泵与 WantQuit 退出/ctx 停机/
  Startup 失败 fatal/ExtraArgs 透传/SIGINT 吞掉语义）；
  `go test ./internal/fxcore/` 回归全绿（stepFrame 提取无行为漂移）。
- 端到端（game_lister.exe 改名形态，-run 400ms）：裸跑 → 释放
  game_lister.ini（小写派生 ✓）→ fskit 装载（4 实体类型+6 扩展函数）
  → 主实体 MainCon 物化 → 帧 400ms（total=0.393s 实测时差步进 ✓）
  → CoreShutdown 优雅退出**码 1**（官方 mov eax,1 ✓）；
  `missing.ini` → `load core ini failed`（小写 ✓）退出码 1。
