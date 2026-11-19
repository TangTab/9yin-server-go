# FxCore.dll 官方全解与 Go 复刻

> 权威快照：`E:\Workspace\project\game\九阴真经ol\BGserverCJ\Debug\FxCore.dll`
>（1,470,464 B，2014-03-22）+ `FxCore.pdb`（5.16MB）。源码树（断言路径串
> 定案）：`d:\flexis\client\fxcore\{core,dllmain}.cpp`——flexis 引擎血统。
> 方法面：PUB32 符号 `__LINE__Var` 断言镜像全量提取（24 类 124 断言位）；
> CCore 虚表 .rdata 0x101b60d4（80 槽）+ capstone 反汇编（工具
> `tmp_research/fxcore/`：vt.py/disas.py/find_vtref.py/extract_classes.py）。
> Go 复刻：`internal/fxcore`（§7 验收清单）；装载壳=internal/fsloader
>（FsLoader 全解姊妹篇）。

## 1. 定位与导出协议

FxCore.dll = Fs 框架**引擎内核**（CCore 运行时）：装载器（FsLoader.exe）
经它启动一切业务模块。仅 5 导出（装载器↔内核硬契约）：

| 导出 | 行为 |
|---|---|
| FxModule_GetType | 返回 "FxCore"（类型门） |
| FxModule_GetVersion | 返回 1003（0x3EB 版本门） |
| FxModule_CoreInitialize(CCoreConfig&) | 构造 CCore + Initialize（返回 ICore*） |
| FxModule_CoreStartup(ICore*, …) | CCore::Startup 五段启动 |
| FxModule_CoreShutdown(ICore*) | 停机 |

与模块侧契约（FsKit 等 8 导出、类型 "FxModule"、版本 100）相对——
内核自己也是一个 FxModule，但类型串/版本不同（CDllManager 收编
业务模块；装载器只认 Core 类型）。

## 2. 类面（PDB 断言镜像提取，24 类）

| 类 | 方法面（断言覆盖处） | 职责 |
|---|---|---|
| **CCore** | 30 虚方法：Initialize/Startup/**Execute**（泵）/Shutdown + 实体（Create/CreateArgs/CreateSameSpace/Lookup/LookupMore/GetEntInfo/SetSingle/GetSingle/SetCanDelByScript）+ 属性方法（GetProperty/SetProperty/InvokeMethod）+ 脚本（RunScript/FindScript/BindScript/RunFunction）+ 逻辑（BindLogic/RunLogicFunc/FindCallback/ExecCallback）+ 杂项（TraceLog/DumpFile/SaveLogList/CreateAsyncProc/GetInterface*） | 内核门面 = ICore 接口 |
| CEntManager | AddCreator/AddFrameExecute/RemoveFrameExecute | 实体注册+帧执行表 |
| CEntFactory | Create/CreateScript/Delete/DeleteScript/Destroy/Dump/Find/FindMore/AddMsgProc/RemoveMsgProc/SetCanDelByScript/SetDeleted | 实体工厂（标记删除+延迟销毁+消息处理器） |
| CEntInfo / CEntInfoList | AddPropertyLink/AddMethodLink/IsKindOf/IsKindSameSpace/SetCreator/SetEntityName/SetParentName/SetSpaceName | 类型描述（creator+继承链+属性/方法链） |
| CIntManager | AddCreator/FindIndex/GetCreator/GetInterface/GetName/GetRefCount/GetReference/ReleaseAll/ReleaseReference/AddFrameExecute/RemoveFrameExecute | 接口注册表（引用计数+帧执行） |
| CFuncManager | AddCreator/GetCreator/GetName | 全局扩展函数 |
| CLogicManager/CLogicInfo(/List) | AddCreator；AddCallbackLink/SetCreator/SetLogicName/SetSpaceName | 逻辑类（回调链） |
| CGameTimer | （无断言；Execute/GetTime 由反汇编定） | 帧时钟 |
| CAsyncManager/CAsyncProc | Add/DeleteAsyncProc/DeleteNode/EraseNode/Find/**GenEvent**/Kill；AddWaitEvent/FindWaitEvent/GetWaitEventObject/Pause | 异步过程+事件等待 |
| CScpManager | AddScript/FindScript/**ReloadScript** | 脚本注册表+热重载 |
| CLuaScript/CScpDebugger | DeleteThread/FindFunc；Execute/Initialize/Receive/Send/SendEntityData | Lua 宿主+调试器 |
| CHookManager | AddHook/RemoveHook/RemoveAll | 钩子 |
| CIdManager | GetId | PERSISTID 发号 |
| CLogTool/CLogFile | Initialize；SaveLog/SetFileName | 日志（trace.log） |
| CMemoryPool/CAllocator | Alloc/ChunkAlloc | 块分配+泄漏转储（allocator.log） |
| CIniFile | 24 方法（与 FsKit.dll 内嵌同源） | 内核自用 INI 引擎 |

管理器存取器（PDB 符号）：GetEntManager/GetFuncManager/GetGameTimer/
GetIntManager/GetLogTool/GetLogicManager/GetLuaScript/GetScpDebugger。

## 3. CCore 对象布局（构造器 0x100cf440 定案）

```
+0x000 vtable（80 槽）
+0x004 byte 停机标志（Execute 首查 → return false）
+0x008 int  （cfg[+0x10]=Sleep 拷贝；Execute 尾段日志节拍门）
+0x00c/+0x10 int  cfg[+0]/cfg[+4] 拷贝
+0x014/+0xa4/+0x134 TFastStr ×3（fileSystem/scriptPath/resourcePath）
+0x1c4..+0x264 列表 ×5（fileConfig/mainEntity/mainScript/mainLogic/modules）
+0x284 int  fpsCap（0=全速；>0=定步 1/fps）
+0x288 int  日志间隔秒
+0x290 double frameClock（定步帧钟）
+0x298 double logClock
+0x2a0 double totalTime（累计）
+0x2a8 double lastElapsed（初值 [0x101b60c0]）
+0x2b0 double virtualTime（虚拟累计）
+0x2b8 byte 帧回调启用=1   +0x2b9 byte 事件帧速=0
+0x2bc HANDLE 帧事件       +0x2c0 默认帧回调 0x100d0330
+0x2d0 统计[5×double]（idle/timer/frame/offset/total）
+0x300..+0x318/0x320 管理器指针区（+0x314=主实体！+0x318=VFS）
+0x320 CXLog（共享内存日志，XLOG_FILEMAPPING_NAME——game_monitor 面）
+0x33c CGameTimer  +0x348 CScpDebugger
+0x350 定步管理器A  +0x360 CAsyncManager  +0x364 定步管理器B  +0x368 CScpManager
+0x304 CIniFile（内核自用）  +0x324 CIdManager  +0x370 末位管理器
```

Initialize（0x100d67e0）：cfg 全量拷贝（含 **cfg+0x2ec ';'-模块表拆分**）→
时钟 Reset → logClock=GetTime → **DebugLog&&门下建 CXLog**（共享内存，
SetMapping(cfg[+8],cfg[+0xc])）→ 模块拆分入表。另：dllmain 读环境旋钮
（"ScriptDebug"/"Wait"），CoreInitialize 尾部开 **LFH 低碎片堆**
（"enable low fragmentation heap"）。

## 4. CCore::Startup（0x100d6f40）五段序

错误串即断言序（反汇编逐一对应）：

1. **VFS**：`file system %s not found` / `file system %s load config %s fail`
2. **脚本系统**：Lua 宿主 + "ScriptDebug"/"Wait" 旋钮 + CScpDebugger
3. **模块循环**：';' 表逐个 CDllManager::Load（**上限 100**）；
   空表=全部（loadall 语义）
4. **主实体**：CreateEntity(cfg.MainEntity) → core+0x314；
   失败 `create main entity failed`
5. **主脚本**：RunScript(cfg.MainScript **缺省字面量 "main"**，
   ScriptPath\<name>.lua)；失败 `run main script failed`
6. **主逻辑**（可选）：BindLogic；失败 `bind main logic failed`

（注意官方序：**主脚本先于主逻辑**；[Main]/[Outer] 等角色段内核不读，
由模块 Startup 后自取。）

## 5. CCore::Execute（0x100d7b90，vtable+0x4c 泵方法）——本包核心复刻物

```c
bool CCore::Execute() {
    if (this->shutdown[+4]) return false;
    dt = GameTimer.Update();                    // 真实 dt
    ScpDebugger 前置拍
    if (帧回调启用 && [ +0x2c0 ]) dt' = FrameCallback(last, virtual, &total);
    else                          dt' = dt;
    totalTime += dt; virtualTime += dt'; lastElapsed = dt';
    AsyncManager 第一遍（0x100fcef0）           // 事件派发+过程运行
    if (fpsCap > 0) {                           // ★ 定步模式
        step = 1.0/fpsCap;
        for (i = 0; i < 10; i++) {              // 每拍最多补 10 帧
            if (frameClock > virtualTime) break;
            Idle(); TimerExec(step);            // [+0x350] 定步驱动
            FrameExec(step);                    // [+0x364] 实体帧
            if (i == 9) frameClock = virtualTime;   // 补满强制对齐
            else        frameClock += step;
            // 每段耗时分别累计（idle/timer/frame 三钟）
        }
        offset = virtualTime + step - frameClock;
        assert(offset >= 0);                    // core.cpp 断言
        FrameExec(offset);                      // 尾帧余量拍
    } else {                                    // 全速模式
        Idle(); TimerExec(dt); FrameExec(dt); FrameExec(0);
    }
    AsyncManager 第二遍（0x100fcfc0）           // 终止过程清理
    ScpManager::Execute()                       // 逐脚本 ReloadScript（mtime）
    统计 total 累计；日志节拍（间隔秒）触发转储
    if (事件帧速启用 && 帧事件 && 余量ms>0)      // WaitForSingleObjectEx 挂帧
    return true;
}
```

语义要点（测试锁死）：

- **定步补帧**：虚拟时间领先帧钟 N 步 → 一次 Execute 连跑 N 步（≤10），
  第 10 步强制对齐；余量 offset 走尾帧拍（offset<0 视为 0——断言防护）
- **首拍即推进一步**（帧钟 0 ≤ 虚拟时间，非零即步进）
- **双管理器定步**：+0x350（TimerExec）与 +0x364（实体帧）各按 step 驱动
- **CScpManager::Execute**（0x100e6ea0 定案）：遍历脚本表逐个
  ReloadScript——内部 mtime 比对，变更才真重载；[Script] Reload=1 守门
- **事件系统**：CAsyncManager.GenEvent 入队 → 第一遍按目标实体匹配唤醒
  AddWaitEvent 的等待者；CAsyncProc.Body 协程步进；Kill/Pause(strict)
  由第二遍清理
- **帧统计五钟**：idle/timer/frame/offset/total（SaveLogList 呈现面）

## 6. nx_* Lua API 面（字符串表全量 82 名）

类别分布：实体 17（create/destroy/lookup/name/type/space/single/
set_value/custom×5/path×3/is_valid/is_null/id_equal）、类型 2、属性 3、
方法 3、事件/异步 6（gen_event/wait_event/pause/strict_pause/kill/running）、
时钟 4、路径/环境 6、日志/系统 9、脚本/逻辑 8（**Lua VM 依赖**）、值构造
12（int/int64/float/double/boolean/string/widestr/null/object/number/
function/value_list/decimals——Lua 类型面）、宽串 4、全局 2
（nx_main/nx_main_config——装载器注入）。

Lua 宿主（CLuaScript）与脚本执行=本项目**里程碑二**（gopher-lua）；
Go 桥先行实现框架域子集（实体/事件/时钟/路径/日志/系统），值构造组与
脚本组声明待接线。

## 7. Go 复刻对齐清单（internal/fxcore）

| 官方 | Go | 对齐点 |
|---|---|---|
| CCore | `Core`（core.go，实现 fsloader.Core） | Type/Version/Startup 五段/Execute/Shutdown |
| CCore::Execute | `Process` | 定步补帧（≤10+对齐）/尾帧余量/双遍异步/热重载/五钟统计/首拍即步进 |
| CGameTimer | `GameTimer`（timer.go） | Reset/Now/Execute(step)/SimTime；dt 钳制 0.25 |
| CEntManager+CEntFactory+CEntInfo | `EntityStore`/`Entity`/`EntInfo`（entity.go） | AddCreator 重名拒收/命名查找（大小写不敏感）/前缀查找/标记删除+帧尾销毁/代际句柄/单例表/IsKindOf 继承链/帧执行表（注册序稳定+可移除） |
| CDllManager::Load→AddCreator | `RegisterEntities` + `loadModules` | 模块装载即收编实体 creator（官方装载序）；上限 100；空=全部 |
| CAsyncManager/CAsyncProc | `AsyncManager`/`AsyncProc`（event.go） | GenEvent 队列→Pass1 目标匹配唤醒；Pass2 清理；Pause(strict)/Kill；单线程泵驱动（无 goroutine——官方协程语义确定性） |
| CScpManager | `ScriptManager`（script.go） | AddScript/FindScript/Execute=mtime 轮询热重载（Loads 计数）；ScriptHost 接口留 Lua 里程碑二 |
| nx_* 82 名 | `APITable`/`APICall`（api.go） | **名表与官方逐名对拍**（测试锁死零缺失零混入）；框架域 56 函数 Go 桥已实现、25 函数 Lua 依赖声明在表 |
| CLogTool/CLogFile | `LogTool`（logx.go） | trace.log CRLF 追加；SaveLogList 缓冲快照；Stats 五钟 String() |
| CXLog 共享内存 | LogTool 文本等价 | Win32 file-mapping 专有（game_monitor 面待分布式里程碑） |
| core+0x314 主实体 | `Core.Main` | Startup 创建+单例登记 |
| engine.App 桥 | `Core.Drive(app)` | AddFrameExecute→TickOnce(step)（分布式角色 System 集驱动位） |

**已知延迟项**（依赖后续里程碑，非遗漏）：Lua VM（nx_call/nx_bind_script
等 8+12 函数与脚本执行）、CXLog 共享内存、CScpDebugger 网面、
CIntManager 接口域、CHookManager（当前无使用方）。

## 8. 验收记录（2026-11-19）

- `go test ./internal/fxcore/ -count=3` 全绿：**12 用例**——
  API 名表对拍（82 名零缺失零混入）/实体生命周期（重名拒收/大小写/
  前缀/继承链/单例/标记删除→帧尾销毁/悬挂句柄）/帧执行序与移除/
  事件唤醒（不匹配不唤醒/匹配唤醒携带事件/两遍清理）/Kill/严格暂停/
  脚本热重载（mtime 轮询+Loads 计数）/Startup 五段（缺脚本与主逻辑
  官方致命文案）/定步泵（首拍即步进/补 5 帧/上限 10+余量拍）/全速+Quit/
  engine.App Drive 一帧推进/API 桥全链（create→lookup→name→type→
  gen_event 唤醒→quit 停机；nx_call Lua 依赖拒调）/GameTimer。
- 全仓 `go build ./... && go vet ./...` 全绿（20 包测试无回归）。
- 实机 `build/game_loader.exe`：全速模式（Sleep=1 泵 500ms 优雅退出）/
  定步模式（-fps 50）/**热重载**（运行中 touch main.lua → trace.log
  `script reload: 1 script(s)` 恰一次）全 PASS；trace.log CRLF 方言
  Startup/Shutdown/重载三行齐备。
