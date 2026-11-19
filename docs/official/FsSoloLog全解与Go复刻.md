# FsSoloLog.dll + game_log.exe 全解与 Go 复刻(日志汇聚服务器宿主)

> 对象:`BGserverCJ\Debug\FsSoloLog.dll`(659456 B,2013-07-05 构建,
> Debug 全符号 .map/.pdb,源码路径 `c:\wuyf\svn\serveriii\branches\publish20130508\server\fssololog\`)
> 日期:2026-11-19　分支:`official`
> 逆向中间产物:`tmp_research/sologlog/`(sl_dis2.py 注释反汇编器 + dis_*.txt)

---

## 一、宿主与定位

**game_log.exe 与 game_lister/game_entry/game_auction 等是同一个文件**
(MD5 `195e1ce1e4a2828d1b26e45608cc3183`),通用 solo 宿主壳按自身文件名读
`game_log.ini`:

```ini
[System]
Sleep=-1            ; 主循环不休眠(SoloLog 自带 0.1s reactor 定时器)
MainEntity=SoloLog
MainScript=main
[Module]
fskit.dll
fscomm.dll
fssololog.dll
[Script]
Path=..\lua\sololog
Reload=1
```

**FsSoloLog.dll 是整个 solo 集群的日志汇聚服务器**:各业务进程
(FsSoloWorld/Member/Public/Auction/Entry/Lister/Store/Monitor/Global,
即 game_*.exe 家族)通过 **Windows 命名共享内存** 把 CORE_TRACE 日志行
投递到各自槽位;本进程 0.1s 一轮把积压行落盘成按进程分文件的日志,
并附带一个 TCP 维护口(echo 控制台)。

FsSoloGlobal 的启动逻辑(docs/official/FsSoloGlobal全解与Go复刻.md §86)
实证了拉起关系:UseQuickLog 时若 `GetExists()==0` 且 LogLauncher 非空 →
`Port_Execute(LogLauncher)` 拉起 game_log.exe。**日志服是被动运维件,
由业务进程按需带起。**

导出 8 个标准 Fx 模块契约:

| 导出 | 值/行为 |
|---|---|
| FxModule_GetType | `"FxModule"` |
| FxModule_GetVersion | `0x3EB`(1003) |
| FxModule_GetSpace | `""`(空) |
| FxModule_Init(core) | `g_pCore = core`,返回 1 |
| GetEntCreator/GetIntCreator/GetLogicCreator/GetFuncCreator | 返回全局 creator(仅 EntCreator 非空) |

## 二、模块清单(.map obj 归组)

| obj(符号数) | 官方类 | 职责 | Go 落点 |
|---|---|---|---|
| SoloLog (82) | `SoloLog : IEntity`(实体尺寸 0x40B8) | 实体本体:7 Lua 属性 + 8 mid 方法 + CheckLog 消费循环 | `internal/fssololog` |
| EchoHook (46) | `CEchoHook` | echo 维护口 TCP 监听(默认 15011) | `echo.go EchoConsole` 同型 |
| LogShareMem (9) | `CLogShareMem : CShareMem` | 共享内存创建/读写(生产+消费同码) | `logsharemem.go`(**本模块的核心业务**) |
| Reactor (122) | `CReactor`(IOCP) | socket/定时器引擎 | `internal/reactor`(既有复刻) |
| AppTool (87) | 工具函数 | 路径/控制台色/SafeSprintf | 各处内联 |
| DllMain (6) | `CLeakChecker` | 退出时泄漏报告写 `mem_fs_solo_log.log` | 不复刻 |
| f(1577) | FxComm/FsKit 静态库并入 | IVarList/TFastStr/Port_* | `internal/proto` 等既有 |

源文件(sololog.cpp 之外的都在公共 tools/ 下,被各 solo DLL 共享编译):
`server\fssololog\sololog.cpp`、`server\tools\{echohook,logsharemem,reactor,apptool}.cpp`、
`server\system\winsharemem.h`。

## 三、SoloLog 实体(对象布局,反汇编定案)

```
+0x00  IEntity vptr
+0x28  void*  m_pWinSocket      ; Init 时 QueryPlugin("WinSocket"),Shut 释放
+0x2C  bool   m_bUseThread      ; 属性 UseThread(ctor 缺省 0)
+0x2D  bool   m_bAutoExit       ; 属性 AutoExit(ctor 缺省 1 ★看门狗缺省开)
+0x30  int    m_nExitCount      ; AutoExit 空闲秒计数(初值 -1,见 §5.3)
+0x34  int    m_nState          ; 0=Closed 1=Opened 2=Paused 3=Unknown
+0x38  TFastStr<0x100> m_EchoAddr
+0x58  int    m_nEchoPort
+0x60  TFastStr<0x100> m_LogFileDir
+0x80  CReactor m_Reactor       ; 内嵌
+0x84  CEchoHook* m_pEchoHook   ; ctor 里 new(this, &m_Reactor)
+0x88  CLogShareMem* m_pQuickLog
+0x8C  CThread* m_pThread
+0x90  int    m_nTimer          ; CheckLog 定时器 id(初值 -1)
+0x94  port_date_time_t m_FileTime[8] ; 当前文件组开档时刻 y/m/dow/d/h/min/s/ms
+0xB4  char   m_FileNames[64][0x100]  ; 每 appid 槽的当前日志文件名
                                  ;    (空串=未开档;首字节即开档标志)
```

**ctor 缺省值定案**（??0SoloLog@@QAE@XZ @0x100552e0 反汇编）：
`+0x2C←0`、`+0x2D←1`、`+0x30←0xFFFFFFFF`、`+0x34←0`——
**AutoExit 看门狗缺省开**（main.lua 不设该属性也不影响：solo 形态
日志服随 world 生灭的设计即由此承载）。

### 3.1 生命周期

- **Init(args)**:`m_pWinSocket = GetCore()->QueryPlugin("WinSocket")`(vt+0x68);
  NULL → false。(实测仅用于确保 WSA 初始化,Shut 即释放。)
- **Execute(frame)**:`m_Reactor.EventLoop()` —— UseThread=false 时由宿主
  主循环驱动;UseThread=true 时另有专职线程跑同一 EventLoop(见 Start)。
- **Start()**(五步,任一失败 CORE_TRACE 后返回 false):
  1. `CMutex::Exclusive("mutex_fs_solo_log")` 单实例互斥;失败
     `printf("(SoloLog::Start)%s exclusive failed\n") + exit(0)`(唯一会
     直接 exit 的路径)。
  2. `m_pQuickLog = new CLogShareMem; Start()` —— 创建共享内存(§4)。
  3. `m_Reactor.Start()` —— IOCP 完成端口。
  4. `m_EchoAddr` 非空才 `m_pEchoHook->Start(addr, port)`(§6)。
  5. `m_nTimer = m_Reactor.CreateTimer(0.1f, TimerProc, this)`;随后
     `Port_GetLocalTime(&m_FileTime)`;`m_bUseThread` 为真再
     `new CThread(ThreadProc, &m_Reactor)` 并 Start(ThreadProc 体仅一句
     `reactor->EventLoop()`)。置 m_nState=1。
- **Stop()**:Thread::Stop+delete → DeleteTimer(-1) → EchoHook::Stop →
  Reactor::Stop → CLogShareMem::Stop+delete。
- **Shut()**:释放 m_pWinSocket(vt+0x18),返回 true。
- **GetState()**:按 m_nState 查表返回 "Closed"/"Opened"/"Paused"/"Unknown"。

### 3.2 Lua 属性与 mid 方法(prop_struct/func_struct 链表注册,ctor 自挂)

属性 7 个:`UseThread(bool)` `AutoExit(bool)` `EchoAddr(str)` `EchoPort(int)`
`EveryDayAlone(bool)` `LogFileDir(str)` `State`(只读)。
方法 8 个:`Start` `Stop` `SetConsoleColor(color)` `WriteLine(line)`
`SendEchoLine(line)` `CloseEcho`。

- `WriteLine(info)`:assert 非空 → `printf("%s\n", info)`(只上本进程控制台)。
- `SendEchoLine(info)`:`m_pEchoHook->SendLine(info)`(推给 echo 客户端)。
- `SetConsoleColor(color)`:`app_set_console_color`(fskit 的
  FOREGROUND_* 颜色名,如 "LightBlue")。
- `CloseEcho()`:`m_pEchoHook->Close()`。

## 四、共享内存日志通道(CLogShareMem,本模块核心)

### 4.1 载体

`Create(name="memory_fs_solo_log_%d", size=0x202300, &ptr)`。块 id 恒 0
→ 名字固定 `memory_fs_solo_log_0`;大小 `0x808C × 0x40` **整除无头部**:
64 个 appid 槽,每槽 0x808C 字节。生产、消费两侧共用本类(生产侧见
FsSoloGlobal 等同款代码,SetAppId 选择自己写的槽)。

### 4.2 槽布局(每 appid,stride 0x808C)

```
+0x000  char   prefix[0x80]   ; SetLogPrefix 写入的进程名;[0]!=0 ⟺ 槽在用
+0x080  u32    create_new     ; SetCreateNew(v):非 0 请求消费侧换新文件
+0x084  u32    write_cursor   ; 生产者游标(单调增,&0x7F 取行号)
+0x088  u32    read_cursor    ; 消费者游标(SoloLog 推进)
+0x08C  char   lines[0x80][0x100] ; 128 行 × 256B 环形行仓
```

### 4.3 生产者接口(业务进程侧)

- `SetAppId(v)`:断言 `0 <= v < 0x40`;写 this+4(槽选择)。
- `SetLogPrefix(s)`:CopyString 到槽 +0(≤0x80)。**即"注册本槽"**。
- `SetCreateNew(v)`:写槽 +0x80。
- `ClearLogUsed()`:槽 +0 清零(**注销本槽**)。
- `GetLogUsed(idx)`:槽 idx 的 prefix[0] != 0。
- `TraceLog(info)`:assert 非空;
  `next = (write_cursor+1)&0x7F`;`next == read_cursor` → 环满丢行返回
  false;否则 CopyString 行仓 `lines[write_cursor&0x7F]`(≤0x100)、
  `write_cursor = next`、`printf("%s\n", info)`(本地控制台同步回显)、
  返回 true。静态 `TraceFunc(ctx, msg)` 即 `((CLogShareMem*)ctx)->TraceLog(msg)`
  ——挂到全局日志钩子后,业务进程所有 CORE_TRACE 自动入槽。

### 4.4 已实证的 appid 分配(SetAppId 立即数)

| appid | 进程 | 依据 |
|---|---|---|
| 1 | FsSoloWorld(game_world) | push 1 @0x100B1CBE |
| 2 | FsSoloPublic | push 2 @0x100BAB42 |
| 3 | FsSoloAuction | push 3 @0x100AA882 |
| 0x32 | FsSoloMonitor | push 0x32 @0x100A042E |
| 0x38 | FsSoloGlobal | FsSoloGlobal 文档 §86 |
| — | Entry/Lister/Member/Store | 运行期计算,未取到立即数 |

文件名里直接可见槽号(§5.1),线上日志目录可反查各家 id。

## 五、消费循环(SoloLog::CheckLog,0.1s 定时器)

### 5.1 主循环(逐槽)

```
mem = m_pQuickLog->GetLogMemory();            // 空则直接返回
for (i = 0; i < 64; i++) {
    slot = mem + i*0x808C;
    if (slot->write_cursor == slot->read_cursor) continue;   // 无积压
    fname = &m_FileNames[i][0];                // 首字节==0 ⟺ 需(重)开档
    Port_GetLocalTime(&now);
    if (m_bEveryDayAlone &&                    // 按天分文件
        (m_FileTime.day != now.day || month || year 变化)) {
        m_FileTime = now; fname[0] = 0;
    }
    if (slot->create_new) {                    // 生产者请求换档
        slot->create_new = 0; m_FileTime = now; fname[0] = 0;
    }
    if (fname[0] == 0)                         // 开档
        sprintf(fname, 0x100, "%s%d_%s_%04d%02d%02d%02d%02d%02d.log",
                m_LogFileDir, i, slot->prefix,
                m_FileTime.年月日时分秒);       // e.g. log\1_FsSoloWorld_20130705105322.log
    fp = Port_FileOpen(fname, "ab");
    n = 0;
    while (slot->read_cursor != slot->write_cursor && n < 100) {
        line = &slot->lines[slot->read_cursor & 0x7F];
        if (line[0] && fp)
            fprintf(fp, "[%04d-%02d-%02d %02d:%02d:%02d] %s\n",
                    now.年月日时分秒, line);     // 时间戳取落盘时刻,非生产时刻
        slot->read_cursor = (slot->read_cursor+1) & 0x7F;
        n++;
    }
    slot->read_cursor 回写; if (fp) fclose(fp);
}
```

要点:
- **每 tick 每槽至多 100 行**(0.1s 节拍 → 每槽上限 1000 行/秒;持续
  超速则环满丢行,生产行为见 §4.3)。
- 文件名 = `目录 + appid + _ + 进程前缀 + _ + 开档时刻(到秒) + .log`;
  `EveryDayAlone` 跨天自动换新档,`create_new` 供生产者主动换档
  (FsSoloGlobal 每次启动 SetCreateNew(1) → 每次重启一个新文件)。
- 行仓空行(line[0]==0)跳过不写;fp 打不开仍推进游标(不堵管道)。

### 5.2 AutoExit(看门狗)

64 槽扫描完后,`m_bAutoExit` 为真才执行;静态计数器每 10 拍(≈1s)一轮:

```
if (GetLogUsed(1)) m_nExitCount = 0;           // appid=1(FsSoloWorld)在线
else if (m_nExitCount >= 0) {                  // 曾在线过才开始计
    m_nExitCount++;
    printf("before exit count %d\n", m_nExitCount);
}
if (m_nExitCount >= 5) GetCore()->Shutdown(1); // world 消失 5 秒 → 整进程退出
```

`m_nExitCount` 初值 -1:**只有 world(槽 1)注册过又消失才倒计时**,
从未在线则永不自退。solo 形态下日志服随游戏服而生、随游戏服而灭。

### 5.3 与其它模块 LogShareMem 的关系

其它 solo 模块文档里 LogShareMem 标注"不复刻(日志呈现等价)",是因为
它们只是**生产者**;FsSoloLog 是**唯一消费者**,共享内存协议在本包是
必须复刻的主业务(§七 Go 落点的取舍见 7.3)。

## 六、echo 维护口(CEchoHook,单客户端 TCP 控制台)

### 6.1 监听与连接模型

`Start(addr, port)`:assert `addr != NULL && port > 0`;
`CreateListener(addr, port, 1, 0x400, 0x400, 0x4000, OnAccept, OnClose,
OnReceive, this, 1, 0)`(监听 id 存 this+8)。**单连接模型**:新连接
到来先 `DeleteConnector` 旧连接,`m_nConnector = 新 id`,行缓冲清零
(内嵌 0x400 行缓冲 + 0x410 长度)。

- **OnAccept(id, ip, port)**:换主后触发脚本事件 `on_open(ip, port)`。
- **OnClose(id)**:仅当 id == m_nConnector 时触发 `on_close`,连接号置 -1。
- **OnReceive(id, data, len)**:仅处理主连接;逐字节扫描,
  `\n`(0x0A) 或 `\r`(0x0D) 任一即行终止;行缓冲非空才
  NUL 终结 → `ReceiveInput(buf)` 并清缓冲。空行不分发。
- `SendLine(info)`:assert;m_nConnector<0 → false;`memcpy + "\r\n"`
  (TAutoMem len+2)→ `Reactor::Send`。Lua 侧回显串自带 `\r`,
  叠加后客户端每命令见双换行(官方行为)。
- `Close()`:ShutdownConnector(优雅半关闭)。

### 6.2 命令分发(ReceiveInput)

```
args = parse_string(line)                     // §6.3
if (args 空 || args[0] 非字符串)      → on_input(line 原文)
cmd = lower(args[0])
if (脚本注册了 "on_input_"+cmd)       → on_input_<cmd>(args…)
else                                  → on_input(line 原文)
```

事件存在性经 core vt+0xCC 查询,派发经 vt+0xD0(与 on_open/on_close 同路)。

### 6.3 行解析(parse_string/parse_token,echohook.cpp 静态函数)

- `parse_string`:副本上逐字符;空格/Tab 且**不在引号内**切 token;
  `"` 翻转引号态且**引号字符本身保留进 token**;行尾冲刷。
- `parse_token`:按序试型——整型(`atoi`,如 `123`)→ 浮点(`atof`,
  如 `1.5`)→ 宽串 → PERSISTID(`name:id` 形)→ 字符串;入 IVarList。
  故 `setp foo Bar 42` 进 Lua 时已是 (str,str,int) 三参。

### 6.4 控制台命令集(lua/sololog/main.lua,随包发布)

`main(logger)`:建 `log\` 目录、`LogFileDir = 工作路径\log\`、
`EveryDayAlone=true`、`SetConsoleColor("LightBlue")`、读
`nx_main_config()` 的 `[Echo] Addr/Port`(默认 127.0.0.1:15011)、
`Start()`、`WriteLine("log server"/"echo socket …"/"game log server started")`。

| 命令 | 行为 |
|---|---|
| `quit` | 回 "server will shutdown..." → close()(WriteLine+Stop+nx_quit) |
| `end` | CloseEcho() 断开本连接 |
| 其它 | `>undefined command: <line>\r` |
| `getp/setp <实体> <属性> [值]` | 查/改实体属性(运行期反射) |
| `getc/setc <类> <属性> [值]` | 查/改全局类属性 |
| `method <实体> <方法> [参…]` | 调实体方法 |
| `listp/listc/listm/listg <名>` | 列属性/类/方法/全局表 |
| `function` | 列全局函数 |
| `debugger` | 调试器开关 |

回显统一 `echo_response`:`SendEchoLine(">"+info.."\r")` + WriteLine
(控制台同步留痕)。on_open 即推 `welcome log server`。

## 七、Go 复刻设计

### 7.1 落点（2026-11-19 已实现全绿）

- `internal/fssololog/`:本模块(logsharemem.go + sololog.go + echo.go
  + parse.go + commands.go + config.go + winsharemem_windows.go)
- `cmd/logserver`:进程载体(内置官方 game_log.ini 方言,缺省释放)
- 复用:`internal/reactor`(CReactor 既有复刻)

### 7.2 组件映射

| 官方 | Go | 备注 |
|---|---|---|
| CReactor | reactor.Reactor | CreateTimer(100ms)/CreateListener 直用 |
| CLogShareMem(消费侧) | LogShareMem | 见 7.3 跨进程取舍 |
| SoloLog::CheckLog | Server.checkLog | 逐槽 100 行上限、文件名/翻卷/时间戳格式逐字对齐 |
| AutoExit | Server.autoExit | 槽 1(FsSoloWorld)==0 且曾在线 → 5s 退出 |
| CEchoHook | EchoConsole(fssoloauction 同型) | 单连接、\r/\n 行协议、SendLine 不补终结符 |
| parse_string/parse_token | parse.go | 引号保留语义 + int/float/string 试型 |
| Lua 命令集 | 内置命令表 | quit/end/getp/setp…(对齐 soloauction 的"Lua→内置命令集"原则) |
| CMutex 单实例 | 端口占用/锁文件 | 失败打印同文案后退出 |
| CLeakChecker/WinSocket 查询 | 不复刻 | Go 运行时等价 |

### 7.3 关键取舍:共享内存 vs 单进程

官方用 Windows named shared memory 是因为**生产者是独立 exe 进程**。
Go 复刻若维持多进程形态(与 cmd/auction 等并立),消费侧有两个选择:

1. **保真方案**:消费侧用 `golang.org/x/sys/windows` CreateFileMapping
   打开同名 `memory_fs_solo_log_0`(0x202300 字节),槽布局/游标协议
   逐字对齐——此时**生产者也必须是官方原版**才有意义,适用于混合部署
   (官方 game_*.exe + Go 日志服)的对拍场景。
2. **纯 Go 方案**(默认):生产者改为进程内 `TraceLog` 直连(或本地
   UDP/命名管道汇聚),槽语义(64 appid/prefix/换档/环)保留为内部
   结构;`cmd/solo` 单进程整合服里即内存 channel 直通。

**落地决策（已实现）**：LogShareMem 直接承载官方槽协议（[]byte 布局
逐字对齐），Windows 缺省开真命名映射 `memory_fs_solo_log_0`
（**保真方案为缺省**，可与官方 game_*.exe 生产进程互操作）；非
Windows 回退进程内缓冲（协议同构，仅无跨进程能力，`-share off`
可强制）。Go 侧业务进程作生产者时同用本包（SetAppId/SetLogPrefix/
TraceLog），不再需要 SlotSink 抽象。

### 7.4 与官方的已知差异

- **Win11 24H2+ 映射掩码坑（实测定案）**：MapViewOfFile 取经典
  FILE_MAP_ALL_ACCESS(0xF0010) 时视图**写入即 AV**（CreateFileMappingW/
  MapViewOfFile 双双"成功"但首页不可写；掩码矩阵 0xF0010✗/0x4✗/
  0x2✓/0x6✓）；复刻取 FILE_MAP_READ|FILE_MAP_WRITE(0x6)，
  winsharemem_windows_test.go 为哨兵回归。

- UseThread 双 EventLoop(主循环+线程各一)在 Go 里由 reactor 自身
  事件循环承担,不再暴露线程开关属性。
- TraceLog 的 printf 本地回显保留为 stdout;SetConsoleColor 在非控制台
  环境为 no-op。
- 行时间戳:官方取**落盘时刻**(非生产时刻,§5.1),纯 Go 方案有条件
  改为生产时刻——保真方案必须维持落盘时刻。

## 八、复核清单

- [x] 导出表/模块类型/版本(0x3EB)/空间("")
- [x] SoloLog 对象布局 0x40B8 全字段
- [x] Start 五步 + 单实例互斥名 + 定时器 0.1s + UseThread 线程体
- [x] CheckLog:槽 stride 0x808C、游标语义、100 行/tick、文件名格式、
      "ab"、时间戳格式、EveryDayAlone/create_new 翻卷
- [x] AutoExit:槽 1、10 拍窗口、-1 起始、5 拍退出、printf 文案
- [x] CLogShareMem 全 11 方法语义 + 槽布局 + TraceLog 满环丢行
- [x] CEchoHook:单连接替换、行协议、SendLine(len+2, memcpy+\r\n 由
      TAutoMem 尺寸推定)、ReceiveInput 分发、parse_string 引号语义
- [x] Lua 属性/方法注册表 + main.lua 命令全集
- [x] 生产者 appid 实证(World=1/Public=2/Auction=3/Monitor=0x32/Global=0x38)
- [ ] FsSoloEntry/Lister/Member/Store 的 appid(运行期计算,待运行态取证)

## 九、Go 复刻验收记录（2026-11-19）

- `go test ./internal/fssololog/`：14 用例全绿——槽协议注册/注销、
  TraceLog 环满丢行（128 环 127 待取）、文件名 `目录 appid_前缀_
  时间戳.log` 与行格式 `[YYYY-MM-DD HH:MM:SS] 行`（落盘时刻）、
  每 tick 每槽 100 行上限（127→100+27 同档 append）、EveryDayAlone
  跨天换档、create_new 换档+消费侧清位、AutoExit（从未在线不自退/
  在线跨越一轮后消失 5 轮退出/printf 序列 1..5）、互斥体二实例
  `(SoloLog::Start)mutex_fs_solo_log exclusive failed` 文案、echo
  单连接顶替+getp/setp/只读拒绝/未定义命令/end 断连、quit 停服、
  parse_string 引号保留与 int/float 试型、ini 方言（官方快照缺
  [Echo] → main.lua 缺省 127.0.0.1:15011）、真命名映射双视图互通
  （winsharemem_windows_test.go 哨兵）。
- 端到端（cmd/logserver，Windows）：裸跑释放 game_log.ini → 真命名
  映射打开 → echo 探针 welcome/`>Opened\r\r\n`（双换行官方行为）/
  setp ok/help/quit → WriteLine+优雅退出；**跨进程对拍**：ctypes
  伪造生产进程 attach 同名映射、注册槽 2"PythonProd"、写 3 行推进
  write_cursor → 0.1s 拍落盘 `log\2_PythonProd_*.log`（官方命名与
  行格式 ✓）。
- 新增定案：ctor @0x100552e0（AutoExit 缺省 **1**）；Win11 24H2+
  FILE_MAP_ALL_ACCESS 掩码坑（§7.4）。
