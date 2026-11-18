# FsDumper.dll + FsDumpLogic.dll 全解与 Go 复刻

> 权威文档。分析对象：`F:\Game\Online\9yin\BGserverCJ\Debug`（官方 Fs 服 Debug
> 全符号快照，2014-03-22，FsDumper.pdb 3.1MB / FsDumpLogic.pdb 8.2MB +
> .map 双全）。Go 复刻：`internal/fsdumper`（载体 `cmd/gamedumper`）。
> 同源前作：FsComm/FsSoloLister（壳与网络件）、save_data OBJ3（档案格式）。

---

## 〇、一句话定位

**GameDumper = 官方数据转储服务器**：一个独立 Fs 框架服务进程
（`FsLoader.exe GameDumper.ini`），对内连 MySQL（role/public/manage/output
四路 Access），对外开两个口——

| 口 | 端口 | 协议 | 对端 | 用途 |
|---|---|---|---|---|
| EchoSock | 15009（默认，`[Echo]` 可覆写） | 行协议 | 运维 | `echo.lua` 维护命令 |
| Outer | **4006**（`[Outer] Port`） | **行协议** | 游戏服(GameServer) | `dump_role_msg` 双向 |

两大业务（都在 FsDumpLogic.dll）：

1. **DumpRole**：游戏服连上 :4006 后，Dumper 下发角色转储模板版本；
   游戏服按模板回推全服角色快照 → 写 `nx_role_dumpinfo` 表。
2. **DumpRes**：扫描 Res 目录（`dumpconfig/dump_config.xml` 清单），
   跑静态 SQL、转储 LogDefine.h 枚举表、按文件 mtime 变更检测把
   资源 XML 增量写入 `nx_res_dumpinfo`（道具/任务/场景/技能/NPC/掉落/复活点…）。

---

## 一、进程形态与启动链

```
GameDumper.bat ──> FsLoader.exe GameDumper.ini
  [System]  Sleep=1  MainEntity=MainConsole  MainScript=main
  [Module]  FsKit.dll FsSql.dll FsComm.dll FsDumper.dll   ← FsDumper=本服务壳
  [Script]  Path=..\Lua\GameDumper  Reload=1
  [Resource] Path=..\Res                          ← dumpconfig/ 在此
  [Main]    DumpLogicDll=FsDumpLogic.dll          ← 逻辑 DLL
  [Outer]   Addr=127.0.0.1  Port=4006
  [Role]/[Public]/[Manage]/[Output] ConnectString=...
```

`main.lua`（`BGserverCJ/Lua/GameDumper/main.lua`）创建顺序：
LogPool(dumper 前缀, TraceMax=256) → IniDocument 读配置 →
EchoSock(默认 127.0.0.1:15009, 绑 echo.lua) → AppManager(AppName="dumper") →
MysqlDBSystem（`UseMysql` 缺省 **true**）→ GameDumper 实体 →
LogicLoader:Load(FsDumpLogic.dll) → `AddAccess(role/public/manage/output)` →
`SetOutAddr(outer)` → AppManager/EchoSock/GameDumper 依次 Start。

**get_db_config quirk**：output_db 实际读的是 `[Manage]` 段
（main.lua 三处 fallback 全是 `ReadString("Manage",...)`）；空则回退 role_db。
`GameStore.ini` fallback 分支永远走不到（GameDumper.ini 总能加载成功）。

`echo.lua` 维护命令面：`quit`（关服）/`end`（断连）+ `getp/setp/getc/setc/
method/listp/listc/listm/listg/function/debugger`——FsComm EchoSock 同方言，
已在 internal/fscomm 复刻，本文不重复。

GameDumper920.ini 变体：无 `[Outer]`/`[Output]`，逻辑 DLL 为
FsDumpLogic920.dll（920=旧库 nx_base1 形态）。

---

## 二、FsDumper.dll（服务壳）全解

### 2.1 导出与编译单元

导出 8 个标准 Fs 模块口：`FxModule_GetType/GetVersion/GetSpace/Init/
GetEntCreator/GetIntCreator/GetLogicCreator/GetFuncCreator`。

自有编译单元（map 实证）：

| obj | 职责 |
|---|---|
| GameDumper.obj | GameDumper 实体 + EntCreator + Lua 方法/属性桥 |
| DumpKernel.obj | CDumpKernel（IDumpKernel 实现，交给逻辑 DLL 的门面） |
| DumpTimer.obj | CDumpTimer 节拍器（beat 数组轮转） |
| Access.obj | CAccess（每库一条 DB 代理线程） |
| GameDumperDbMsg.obj | 7 个 DB 响应处理器 |
| GameDumperState.obj | 四态状态机 |
| Reactor.obj / MsgDeque2.obj / Routine.obj / TextProtocol.obj | IOCP 网络/消息队列/线程/文本编解码（FsComm 同源件） |
| Parameter.obj | 存储过程参数（int/varchar/date/longvarchar × add/set/get） |

### 2.2 GameDumper 实体（Lua 面）

属性：`State`(只读)/`AccessCount`(只读)/`DBSystem`/`Console`/`EchoSock`/
`AppManager`/`LogicLoader`（getter/setter 桥，GameDumper.obj 符号实证）。

方法（`*_mid`）：`start`/`stop`/`put_command`/`add_access(name, connstr)`/
`access_empty()`/`set_out_addr(addr, port)`/`add_client(id, name, len)`。

### 2.3 Outer :4006 —— 行协议（核心发现）

`GameDumper::Start`（0x10074950）：EchoSock/AppManager Start →
`CreateListener(outer_addr, outer_port, 三回调, 栈 0x400/0x400/0x64, 0x4000)` →
`CreateTimer(TimerCheck, 1s)` → 逐 Access `Start()` → 逻辑侧 Start。

**收包路径**（0x100756f0 `ReceiveClientMsg`）：逐字节扫 `\n`(0x0A)/`\r`(0x0D)，
行缓冲 client+8 起 **0x1000(4096) 字节**（长度计数在 client+0x1008，超限丢弃），
遇行终止符 → NUL 截断 → `ReceiveCommand(id, line, len)`。

**命令路径**（0x100758f0）：`text_decode_msg(line) → IVarList` →
callee->`OnReceiveCommand(callback_id, client_id, varlist)`（虚表+0x30）。
即 **:4006 上的每行 = 一条 text 方言 VarList 消息**（与 svrlist 同一
text_encode_msg/text_decode_msg，FsDumper 内置副本在 0x1005da90/0x1005c610）。

**text 方言语法**（0x1005c780 token 解码器 + svrlist 前作实证）：

```
消息   := 记录 (空格 记录)*
记录   := 数字 | '#'宽串 | '$'窄串
数字   := 首字符 ∈ [0-9.-]；无 '.' 且 len<10 → int32；否则 _atoi64/strtod → int64/double
'#'串  := '\\'→'\'；'\u'HHHH→UTF-16 码元；'\x'HH→字节（% 亦走 \x 转义）
'$'串  := 裸 ASCII
```

**客户端结构**：AddClient 分配 0x102C 字节/客户端（[0]=占用标志，[8..0x1008)
行缓冲，[0x1008] 行长），存 `this+0x498` 数组，按 reactor connector id 索引。

### 2.4 状态机（GameDumperState.obj）

`DumperStateMachine(msg, a, b)` 按 `this+0x28` 分派（jmp 表 0x1007961c）：

| 态 | 值 | 处理 |
|---|---|---|
| Unknown | 0 | msg==1 → Opened；msg==0 忽略；其他断言日志 |
| Opened | 1 | msg==2 → Closing；**msg==0（客户端接入）→ callee->OnServerConnected(id) + AddClient** |
| Closing | 2 | msg==0 → Closed |
| Closed | 3 | 静默 |

### 2.5 IDumpKernel 虚表（17 槽，逻辑 DLL 唯一入口面）

虚表 0x100ECFDC（thunk 已解析到符号）：

| 槽 | 方法 | 语义 |
|---|---|---|
| +0x00 | ~CDumpKernel | |
| +0x04 | FindTimer(name, &param) | |
| +0x08 | AddTimer(name, cb, param, first_ms, interval_ms) | CDumpTimer 节拍 |
| +0x0C | RemoveTimer(name, param) | |
| +0x10 | SendResponse(VarList) | 回 EchoSock |
| +0x14 | RunStoredProc(access, proc, params) | CAccess 代理 |
| +0x18 | ExecuteSql(access, sql_w) | |
| +0x1C | ReadTableInfo(access, table, ?) | |
| +0x20 | ReadBatch(access, table, ?, ?, double, double, n) | |
| +0x24 | ReadRecord(access, table, ?, VarList) | |
| +0x28 | WriteRecord(access, table, ?, wstr, n, int*, char**, VarList) | |
| +0x2C | QuerySql(access, field, cond, table, limit) | |
| +0x30 | SendServerMsg(client_id, VarList) | → :4006 指定客户端（text_encode_msg 行） |
| +0x34 | GetTableSchem(access, table) | 列名探查（回 OnTableColNames） |
| +0x38 | GetResourcePath() | Res 目录 |
| +0x3C | IsAccessIdle() | |
| +0x40 | Quit() | |

CAccess 为每库名一条线程（AccessInitProc/LoopProc/ShutProc），内部走
Request*/Response* 八对消息（StoredProc/ExecuteSql/ReadTableInfo/ReadBatch/
ReadRecord/WriteRecord/QuerySql/TableColNames）与 Store 服通信——
**Go 复刻直接落 MySQL（database/sql），此层不复刻**。

### 2.6 主泵 TimerCheck（0x100768e0）

1. 排空 DB 响应队列（+0x458）→ ProcessMessage → ResponseXxx → callee 回调；
2. 排空命令队列（+0x45c）→ DumperStateMachine / PutCommand 消费；
3. 1000ms 节流窗 + CDumpTimer::Run（节拍触发 AddTimer 注册的回调）。

---

## 三、FsDumpLogic.dll（业务逻辑）全解

### 3.1 入口与单例

导出：`FxModule_GetType` / `FxModule_Init` / `FxDumpLogic_GetVersion` /
**`FxDumpLogic_GetCallee`**（0x100bf680 → dump_callee 单例）。

`dump_callee` 构造（0x100befa0）new 两全局：
`g_dump_role`(0x6C 字节, DumpRole) + `g_dump_res`(0x1A8 字节, DumpRes)。

回调面（vtable 0x1016FC88）：

| 回调 | 行为 |
|---|---|
| OnDumperStart | DumpRole::OnDumperStart + DumpRes::OnDumperStart |
| OnFrameExecute/OnServerClose/OnReceiveCommand | **空实现**（命令不驱动业务） |
| OnRecvGameServerMsg | 首字段 == `"dump_role_msg"` → DumpRes::OnRecvGameServerMsg（内部再转 DumpRole） |
| OnServerConnected | DumpRes::OnServerConnected |

### 3.2 DumpRole（角色转储）

配置（Res 下）：`dumpconfig/dump_role_dbinfo.xml` + `dumpconfig/dump_role_record.xml`。

**模板**（dump_role_dbinfo.xml）：表 `nx_role_dumpinfo`，主键 role_name，
索引 role_uid。每列 `<property desc/col/dbtype/type/prop/child/createsql>`：
- `type="prop"`：直取角色档案属性（Name/Uid/Account/CreateTime/TotalSec/
  Sex/Camp/Race/Job/Level/FightPower/GuildName）
- `type="default"`：固定值（deleted=0、delete_time=1）
- `type="record"`：记录序列化（capital_rec → money_rec varchar(4096)）

dbtype→VarType 映射（0x1016D0A8 常量表）：int→1、varchar→4、nvarchar→5、
datetime→7、decimal→1、longtext→4。

**启动链**（OnDumperStart 0x100ADAA0）：
print_version_info → GetResourcePath → LoadDumpRolePropInfo/RecordInfo(解析两 XML)
→ m_dumpVersion=-1 → `GetTableSchem("output", "nx_role_dumpinfo")`。

**OnTableColNames** → 模板比对 → 建表（逐列 createsql 拼 CREATE）/校验 →
`SendDumpVersionInfo`。

**dump_role_msg 协议**（SendDumpVersionInfo 0x100AF750）：

```
Dumper → 游戏服（SendServerMsg 按 client_id）:
  ["dump_role_msg", 1, version, "nx_role_dumpinfo",
   每列三元组(name, dbtype, prop)...]

游戏服 → Dumper（OnRecvGameServerMsg 0x100AF910）:
  ["dump_role_msg", 1, ver]        # 版本报文：ver==本地 → 走列流程；否则回发模板纠偏
  ["dump_role_msg", 2, 档案串...]  # 角色数据：[3]=字段数，其后按模板列序给值
```

**ParseRoleInfo**（0x100AFDD0）：字段数 ≠ 模板列数 →
`parse role info error, server fields=%d, template fields=%d` 丢弃；否则逐列
分派：type=1(prop) → InitWriteProp 直取；type=4/5(default/record) →
查 record 定义（dump_role_record.xml：capital_rec 三列
capital_name/capital_type/capital_number）→ ParseRoleRecord 展开 →
WriteRecord 落库（`role [%s] has succeed deleted, use time[%d]` 日志实证
upsert 语义）。

档案侧读取器 game_object/game_public（LoadObj/Build + 属性/记录 getter 全套）
= **save_data OBJ3 同源格式**（3JBO 容器；见
`docs/official/nx_base-save_data容器格式.md`），游戏服把角色档案整档推来，
Dumper 用 OBJ3 读取器按 prop 名抽值。

### 3.3 DumpRes（资源转储）

配置：`dumpconfig/dump_config.xml`（gb2312）：

```xml
<config>
  <diction path="dumpconfig\diction\"/>
  <static>  <!-- 逐文件整跑 SQL（;; 双分号分隔） -->
    <sql path="dumpconfig\game_enum.sql"/>
    <sql path="dumpconfig\ratio_action.sql"/>
    <sql path="dumpconfig\dump_res.sql"/>
  </static>
  <logdefine path="ini\logdefine.xml"/>
  <move>   <!-- type: 1道具 5任务 7场景 8技能 10PVP任务 14NPC 15配置包 17掉落 19复活点 -->
    <file bfile="1" type="1" table="nx_res_dumpinfo" path="ini/Item/Equipment.xml"/>
    ...
  </move>
</config>
```

**启动链**（OnDumperStart 0x1007D720）：
`AddTimer("TimerStartDumpConfig", cb, kernel, 1s, 0)` → 定时器到点：

1. `ExecuteSql("Manage", "DELETE FROM nx_config WHERE (op_type = 0 OR ISNULL(op_type))")`
2. `QuerySql("Manage", "nx_config", "id", "id <> ''", 10000000)` — 拉运营配置基线；
3. StartDumpRes(kernel, argc, argv)（echo 亦可触发）：
   - LoadConfigFile(资源根 + `dumpconfig/dump_config.xml`)
   - `AddTimer("TimerCheckDumpConfig", cb, this, 300000ms, 0)` — **5 分钟变更巡检**；
4. 静态 SQL 逐个执行（`All sql run over ...........`）；
5. DumpLogDefine：解析 `dumpconfig/DumpHFile/LogDefine.h`（47KB C 头文件，
   枚举宏表）→ 重建 `log_enumtype`/`log_enumvalue` 两表（DROP+CREATE+INSERT，
   `UPDATE log_enumvalue SET text=... desc=...` 增量修正）；
6. `<move>` 清单逐文件：`FindFileTime`(mtime) vs DB `op_version` → 变更才转储；
   FileAssign 读文件（XML/TextGrid/`*.idres` 字典三格式，book/cell 结构）→
   `DoDumpResource(kernel, type, path, table, bfile)` → UpDataRecord/InsertRecord
   生成 `INSERT INTO nx_res_dumpinfo ... ON DUPLICATE`/UPDATE 方言 →
   `Object dump complete.`；
   - 特型 RuleXxxToXml：Item/Scene/Skill/PvpAction/Npc/Pack/WorldDropSimple/
     Transport/creator 九类把资源转成 XML 属性串（`version="%s"` 头）。
7. `All dump over ..............`。

**nx_res_dumpinfo 行形态**（dump_res.sql 实证）：
`id,name,type,property,status,op_name,op_type,op_property,op_status,op_version,key_id`
（id=类型根行/文件标识、property=XML 属性串、op_version=文件 mtime 基线）。

**变更检测**：`BuildCleanSql`（DELETE 全量型）与 `DeleteType(type)` 支持全清；
巡检定时器每 5 分钟对 `<move>` 全表 FindFileTime 比对，mtime 变化 → 重转储。

### 3.4 工具层

- **TextGrid/Column/Row/Cell/Dictionary**：文本表格引擎（readtxt.h，
  支持字典 diction 目录 `*.idres`）；
- **FileAssign**：XML 清单 → book(文件)/cell(表) 装载（LoadFile/LoadSingleFile/
  LoadCellFromCatalog/LoadCellFromFile，file_format 枚举）；
- **DumpHFile**：C 头文件解析（LoadFromHFile/BuildH/TrimWord）；
- **IniFile/XmlFile/tinyxml**：配置装载；
- **MemPool2/util_\***：杂项；
- **database_operation**：CreateDumpTable_Role/Item/Config 三建表助手。

---

## 四、Go 复刻（internal/fsdumper）

| 官方件 | Go 对应 | 说明 |
|---|---|---|
| FsLoader+GameDumper.ini | `Config`（config.go） | 同段名解析；output←manage quirk 保留 |
| GameDumper 实体+主泵 | `Server`（server.go） | 状态机四态+客户端注册+命令队列 |
| Outer :4006 行协议 | `outerConn` | 4096 行缓冲、\r\n 切行（echosock 同法） |
| TextProtocol | `textvarlist.go` | `#`/`$`/数字三态编解码（本包独立实现，含 \u/\x 转义） |
| IDumpKernel | `Kernel` 接口（kernel.go） | 定时器/DB 面/echo/SendServerMsg |
| CDumpTimer | `Kernel` 内置节拍器 | name+param 定位，first/interval 双参 |
| CAccess→Store | **database/sql 直连 MySQL** | 官方内部协议不复刻 |
| dump_callee | `Callee`（callee.go） | 装配 DumpRole+DumpRes |
| DumpRole | `dumprole.go` | 模板 XML+dump_role_msg+WriteRecord upsert |
| DumpRes | `dumpres.go` | dump_config.xml+静态 SQL+LogDefine.h+变更巡检 |
| EchoSock :15009 | 复用 `internal/fscomm.EchoSock` | |
| echo.lua 命令面 | cmd 载体挂 quit/end | getp/setp 等控制台类命令暂缺（Fs 实体系统无对应） |

设计要点：
1. **两端都是自家 Go 时**，:4006 线格式仍按官方 text 方言落字节——
   保真 + 可与官方端互操作；
2. DumpRole 的 WriteRecord 走 `INSERT ... ON DUPLICATE KEY UPDATE`
   （官方 upsert 语义，主键 role_name）；
3. DumpRes 巡检 5 分钟一轮（官方 0x493E0ms），mtime 用 Unix 秒存 op_version；
4. LogDefine.h 解析器直接吃官方 `Res/dumpconfig/DumpHFile/LogDefine.h`
   原文件（golden 测试）。

## 五、验证

- `go test ./internal/fsdumper/...`（13 用例全过）：text 编解码
  round-trip/golden、配置解析+四库回退、状态机、:4006 行协议+广播、
  E2E TCP→callee→upsert（含官方 15 列模板、双命令同连接）、
  dump_role 模板/建表/upsert SQL golden、dump_config 解析 golden、
  LogDefine.h 解析 golden、字段数不符丢弃。
- **实弹闭环**（本机 MySQL 临时库 nx_dumper_smoke，官方 F: Res 原档）：
  - DumpRes 全量一轮：25 文件 → nx_res_dumpinfo **21934 行**（Equipment
    4507/Quest 1858/八技能 7199/Drop 2807/location 329…）；
    LogDefine.h → log_enumtype **16 组** / log_enumvalue **411 条**；
    nx_config 清理 SQL 正常（空库无表报错与官方空库行为一致）；
  - DumpRole：连接 :4006 → 收模板（`$dump_role_msg 1 286311
    $nx_role_dumpinfo 1 $role_name $Name …`，286311=模板指纹版本）→
    错版本 999 → 收纠偏模板 → 发 op=2 快照（15 字段）→
    `role="测试侠" uid="uid-z1" ct=2026-11-19 14:00:00 … guild="帮会"
    money="黑金,0,100;金币,1,50"` 全列正确落库（GBK→UTF-8 无损）。
- 载体 `cmd/gamedumper`：`gamedumper -c GameDumper.ini [-res DIR]
  [-dsn DSN]`（或 NINEYIN_DUMPER_CONFIG/NINEYIN_DUMPER_DSN）；
  ConnectString 自动 OLEDB→Go DSN 解析；NINEYIN_DUMP_TRACE=1 开
  :4006 字节级诊断。

### 踩坑录（复刻必读）

1. **:4006 行必须以 ``/`
` 结尾才派发**——官方 ReceiveClientMsg
   逐字节扫行终止符；无终止符则缓冲等待。对拍客户端漏发 `
`
   会表现为"服务端静默"（本次实弹排查 40 分钟的真凶）。
2. Go `encoding/xml` 拒收声明为 `gb2312` 的内容——即使已转码 UTF-8，
   声明串本身也会触发 CharsetReader 缺失错误；须改写声明
   （normalizeXML）。
3. 资源转储必须**逐实体行**（`<Property/>` 每行一条），整档塞
   varchar(16384) 会 Error 1406；文件级行（property 空）作 mtime
   基线标记，供 5 分钟巡检跳过。
4. 载体装配顺序：AddAccess 必须先于 Server.Start（官方
   AddAccess→Start 同序），否则 DumpRole EnsureTable 找不到 output。
5. `type="default"` 列值直通 SQL——datetime 列送 `1` 会 MySQL 1292
   （官方同样直通，游戏服侧应送合法值）。
