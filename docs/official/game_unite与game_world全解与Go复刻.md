# game_unite 与 game_world 全解与 Go 复刻

> 对象:`BGserverCJ\Debug\game_unite.exe` / `game_world.exe`（+ 各自 ini 与
> `Lua\solounite\main.lua` / `Lua\soloworld\main.lua`）。
> Go 复刻:`internal/fsunite + cmd/solounite`（合一服）与
> `internal/fssoloworld + cmd/soloworld`（世界服,本轮补 scenes 装载）。
> 本文是这两进程的权威分析文档。

## 一、二进制层结论:同体宿主

| 文件 | 大小 | MD5 |
|---|---|---|
| game_unite.exe | 589,824 | `195e1ce1e4a2828d1b26e45608cc3183` |
| game_world.exe | 589,824 | 同上 |
| game_entry.exe / game_lister.exe / game_member.exe / game_public.exe / game_store.exe / game_auction.exe / game_robot.exe / game_monitor.exe / game_log.exe / game_global.exe | 589,824 | 全部同上 |
| FsLoader.exe | 585,728 | `10a98d4363c039151c2e6aac1225c805`（大写系宿主,另一 build） |
| FsSoloLoader.exe | 589,824 | `c2f2df73204e402cce65bcac2644262b`（同名异 build） |

**全部 11 个小写系 game_*.exe 是同一个通用 solo 宿主二进制**——壳行为
（装载五步/主实体帧泵/ini 发现序 `<exe基名>.ini`）即已复刻的 FsSoloLoader
家族（cmd/soloLoader/cmd/loader）。两进程的功能差异**完全**由两件事决定:

1. **ini 方言**（[Module] 装载哪个 fssolo* 模块集 + 配置节）;
2. **Lua 引导脚本**（`[Script] Path` 指向 solounite 还是 soloworld 的
   main.lua——MainEntity=SoloWorld 由 FxCore 创建,`MainScript=main` 起
   `main(world)` 函数）。

即:复刻 game_unite/game_world ≠ 复刻 exe（已有）,而是复刻
**ini 方言 + main.lua 编排**。

## 二、game_world.exe（世界服独立形态）

### ini（快照 `game_world.ini`,GBK 注释）

```
[Module] fskit.dll / fscomm.dll / fssoloworld.dll   ← 只装世界模块
[Script] Path=..\lua\soloworld                      ← 711 行引导
[Main]   DistrictID=3 ServerID=11 RoleNameSizeLimit=18
[Broadcast] 127.255.255.255:4001
[ManageServer] Inuse=true :4002   [Monitor] Inuse=true :4008
```

### main.lua（soloworld\main.lua,711 行）行为序

1. `world.UseQuickLog/UseAppRecord`,控制台色 LightGreen;
2. 读 ini 全键（Main/Inner/Broadcast/Echo/Charge/Gmcc/ManageServer/
   Monitor/DumpServer——缺省回退 `ReadString` 第三参）;
3. 属性赋值:`AppName="world_"..sid`、`AccountRoles=5`、
   `LogLauncher="game_log game_log.ini daemon"`;
4. `LoadNameFilter(res\filter.txt)` + `load_char_image`（17 张验证码 bmp）+
   **`load_scene_config`（res\scenes.ini → AddScene/AddPrototypeScene）**;
5. `world:Start()` → 打印 app/echo/inner/broadcast 横幅 → `SetAppState(WAPS_OPENED)`;
6. echo 维护口（缺省 127.0.0.1:15001）命令集:`quit/fquit/end/players/
   open/close/status` + nx 反射调试族（getp/setp/getc/setc/method/
   listp/listc/listm/listg/function/debugger）;
   - `quit` → `close()`:**先置 `world.QuitOnClosed=true`** 再
     `CloseServer()`——Closed 后自动退出进程;
   - `fquit` → `force_close()`:`world:Stop()` + `nx_quit()`。

### scenes.ini 方言（Res\scenes.ini）

节名 = 场景号;键 `Member`（所在场景服编号）/`Script`（逻辑类名,缺省
`scene`）/`Config`（场景配置名）/`Clonable`（1=true → 副本原型,走
`AddPrototypeScene` 否则 `AddScene`）。快照实配 5 场景。

### Go 复刻（cmd/soloworld,本轮补全）

- 既有:`internal/fssoloworld`（世界 13 态/玩家 14 态/echo/广播五分支）;
- **本轮新增**:`fssoloworld.LoadScenesINI`（包内导出,solounite 共用）+
  cmd `-scenes`（缺省 `..\res\scenes.ini`,缺失静默跳过）;
- **本轮修正**:echo `quit` 补官方 `QuitOnClosed=true` 语义（此前只
  CloseServer 不退出）;
- 冒烟:官方 Debug 目录 + 真 ini/真 scenes.ini → `load scene config: 5 scenes`,
  横幅与官方 WriteLine 序一致。

## 三、game_unite.exe（合一服,本轮主体）

### ini（快照 `game_unite.ini`,2565B）

在 game_world.ini 基础上扩展:

| 节 | 内容 |
|---|---|
| [Module] | **11 模块全量**:fskit/fssql/fscomm/fscollide/fssololog/fssoloentry/fssolostore/fssolomember/fssoloworld/fssolopublic/fssoloauction |
| [Script] | `Path=..\lua\solounite`（1380 行引导） |
| [Main] | + PublicLogicDll=fspublogic.dll / AuctionLogicDll=fsauctionlogic.dll / CollideRootPath=..\Res\map / UseMysql=true / MessageEncodeKey（注释）/ MessageVerifyDll（注释）/ RoleNameSizeLimit=18 |
| [Outer] | **Addr=127.0.0.1 Port=2001**（entry 外网受面——唯一对客户端口） |
| [Logic] | Path= + Modules=2 + Module0=fsgame.dll + Module1=SnailEditor.dll（玩法层） |
| [Account]/[Role]/[Log]/[Public]/[Manage] | 五库 SQLOLEDB 连接串 + ConnectPool |

### main.lua（solounite\main.lua,1380 行)编排核心

**七实体同进程**（`nx_create`）:

```
logger(SoloLog) world(SoloWorld) store(SoloStore) entry(SoloEntry)
public(SoloPublic) auction(SoloAuction) member(SoloMember)
```

关键属性（对照 cmd/solounite/fsunite 投影）:

- 共享:`InnerPort=0`（全部动态分配,注释掉的 +1/+2/+100/+200/+300
  分口公式是分进程集群形态的遗迹）、`BroadAddr/BroadPort` 同口;
- world:`AppName="unite_"..sid`（非 world_!）、echo 15001、
  AccountRoles=5、FlushMinutes、LogLauncher;
- store:`AppName="store_"..sid.."_0"`、StoreId=0、五库串+池、
  `WakeAddr=127.0.0.1 WakePort=0`、LogDBPerformance、
  **`init_custom_log` 注册 29 张日志表**（item_log/deal_log/…/
  sys_total_capital_snap——与 fssolostore.NewLogTableSet 缺省集同源）;
- entry:IDC=0、EntryId=0、MaxConnects=1024、
  **License=`<OuterAddr>.lic`**（即 127.0.0.1.lic,256B 内部形态）、
  `EncodeKey=ConvertEncodeKey(MessageEncodeKey)`、InBufferLen 8K/
  OutBufferLen 32K/OutBufferMax 128K;
- public:`AddPubSpace("domain","nx_domains")`+`AddPubSpace("guild","nx_guilds")`;
- member:MemberId=0、CollideRootPath、ScriptPath=`res\lua\`、
  HideEmptyProperty=true、LoadLocations(location.xml);
- logger:UseThread、LogFileDir=`log\`、**EveryDayAlone=false**（与
  独立 game_log 形态的每日一档相反）;
- world 外连:`SetChargeAddress/SetGmccAddress/SetManageAddress` +
  `SetExtraAddress(0=monitor, 2=dumpserver)`。

**启动序**（408-421 行,缺一不可的依赖序）:

```
logger:Start() → world:Start() → store:Start() → entry:Start() →
public:Start() → auction:Start() → member:Start()
```

**停机序**（force_close,927-952 行）:

```
store → auction → public → entry → member → world → logger → nx_quit()
```

**echo 命令集**:soloworld 全集 + **`perf`**（`DumpPerformance("")`
导出逻辑模块性能统计——unite 独有）。`close()`/`quit` 语义同 soloworld。

### 互发现机制（合一形态的关键)

七实体**不走进程内直连**,仍走官方广播回路:每个实体各持一个 UDP 广播
套接字共绑同一 `BroadAddr:BroadPort`(官方 CCore `CreateBroadcast` =
`SO_REUSEADDR + SO_BROADCAST`),周期自宣(携带 comm id + inner 实际
口),彼此听到后经 127.0.0.1 TCP 回环 `CreateConnector` 内连 + identify
握手——与分进程集群完全同构。**Win11 实测**(internal/reactor
TestBroadcastCoBind):REUSEADDR 多套接字共绑同口,127.255.255.255
定向广播**全量投递到每个套接字(含发送者自收)**。

## 四、Go 复刻(internal/fsunite + cmd/solounite)

### 落点

| 层 | 文件 | 内容 |
|---|---|---|
| 基础 | internal/sockopt(新) | ListenUDPReusable:bind 前置 SO_REUSEADDR+SO_BROADCAST(Windows/其它双形态) |
| 基础 | internal/reactor CreateBroadcast | 改走 sockopt(共绑;fscomm.BroadSock 同修,兑现其注释宣称) |
| 组件 | internal/fssoloworld | Options.AppName 覆盖(unite_N)/echo perf/quit=QuitOnClosed+CloseServer/LoadScenesINI 导出 |
| 组件 | internal/fssolomember | **Stop 死锁修复**(tickDone 1s 超时兜底,ticker.Stop 不关 channel) |
| 组件 | internal/fssoloentry | **Stop 幂等化**(stopOnce;合一进程内 Quit 钩与停机序并发重入) |
| 编排 | internal/fsunite(新) | ParseGameUniteINI 全方言 + 七服务投影(World/Store/Public/Member/Auction/Logger Config)+ Assemble/Start/Stop(官方序,幂等) |
| 壳 | cmd/solounite(新) | 官方 ini 原字节内置释放/license 门+生成器/MessageEncodeKey XOR/-run 冒烟 |

### 验证

- `internal/fsunite` 测试:全方言解析断言(注释键不泄漏/缺省回退/五库/
  Logic 枚举)+ **七服务同进程装配冒烟**:随机口位 + 内部形态 license →
  广播互发现断言(member 外连 `（外连，已发 identify）` + entry 收
  `ident 0x18 注册`)→ echo status/players/perf 对拍 → 停机序六行顺序断言;
- `internal/reactor` TestBroadcastCoBind:双 Reactor 共绑同口双收;
- 活体冒烟(临时目录):`-gen-license 127.0.0.1` → 运行 → echo 探针
  `welcome server 11` / `status → game server state is Opened`(七服务
  集群自动开服)/ `perf → dump logic module performance...`;
  日志可见 member/public/auction 各自外连注册、auction 连 world(comm 0)
  与 member(comm 24);
- cmd/soloworld 冒烟:官方 Debug 目录真 ini/真 scenes.ini →
  `load scene config: 5 scenes`;
- 全仓 `go test ./...` 全绿。

### 已知偏差(呈现层,见包文档)

1. **逻辑层 DLL 不装载**:fsgame.dll(15MB 玩法层)/SnailEditor.dll/
   fspublogic.dll/fsauctionlogic.dll——只打官方日志行
   `load loigc module <dll>`(拼写保留);数据逻辑由 fssolopublic/
   fssoloauction 内建 callee 承载,玩法层属后续里程碑;
2. **DB 后端 = 内存**:五库连接串仅呈现(UseMysql/LogDBPerformance
   照解析),Backend 注入 fssolostore.NewMemoryStore——FsSql 双后端
   IDB 的 Go 落地是独立课题;
3. **logger AutoExit 关闭**:fssolo* 服务无 LogShareMem 生产者
   (world.UseQuickLog 的写入端未复刻),看门狗开则合一起动 ~5s 必被
   误杀;EveryDayAlone/UseThread 照官方;
4. member 的 ScriptPath/HideEmptyProperty/LoadLocations、world 的
   LoadNameFilter/LoadCharImage/LogLauncher 为呈现字段(包现状,
   fssolomember 无 echo 亦同);
5. echo 调试族(getp/setp/…)为框架保留返回 undefined(nx 反射不复刻)。

## 五、相关文档

- FsSoloWorld 全解(internal/fssoloworld 权威):`FsSoloWorld全解与Go复刻.md`
- FsSoloStore 29 表与五通道:`FsSoloStore全解与Go复刻.md`
- FsSoloEntry/license 方言:`FsSoloEntry全解与Go复刻.md`
- 宿主壳/FxCore:`docs/official/`(FsSoloLoader/FxCore 相关)
