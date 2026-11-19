# FsSoloGlobal.dll 全解与 Go 复刻

> 权威事实源：`BGserverCJ\Debug\FsSoloGlobal.dll`（208 KB，2014-03-22 快照，
> VC8 Debug）+ `FsSoloGlobal.pdb`（1316 公共符号，S_PUB32 无尺寸）+
> `game_global.ini` + `Lua\sologlobal\main.lua`（官方主控脚本原文）。
> 符号偏移注意：PDB 公共符号基于提交前布局，最终 PE 中函数实际
> VA = `0x10000000 + 符号偏移 + 0x12000`（.text RVA；经导出表 thunk
> 目标校验，如 `_FxModule_GetEntCreator` 0x17b30 → 0x10029b30）。

## 一、定位

**全局名字服务器**（global name server）：solo 单机集群里唯一负责
跨服唯一名（角色名/帮会名等）登记与释放的数据库前置服务。

```text
game_global.exe（通用 solo 壳 FsSoloLoader，按自身文件名读 game_global.ini，
                  装载 [Module] 列表：fskit/fssql/fscomm/fssologlobal.dll，
                  MainEntity=SoloGlobal，MainScript=main）
 └ FsSoloGlobal.dll（本模块）＝ SoloGlobal 实体
     ├ CReactor（IOCP 反应器；Go 等价 internal/reactor）
     ├ 外网受面 TCP 监听（[Outer] Addr:Port，缺省 127.0.0.1:4003，
     │  快照 ini 为 192.168.33.70:4005）—— 行协议命令通道
     ├ CEchoHook（维护口 TCP，[Echo] 缺省 127.0.0.1:15012）
     ├ CAppRecord（跨进程运行记录，UseAppRecord=true 时）
     ├ 状态机 Unknown→Opened→Closed（sologlobalstate.cpp）
     ├ 域表 domains[8]（name[0x24]+table[0x24]，this+0xE0 起 0x240）
     └ CAccess → DBSystem 实体（main.lua: nx_create("MysqlDBSystem")，
        fscomm/fssql 提供）→ SQL：SELECT/INSERT/DELETE
```

## 二、SoloGlobal 实体布局（反汇编定案）

### 成员偏移（this+，0x1002bc30 ctor / 各 getter 实证）

| 偏移 | 类型 | 成员 | 依据 |
|---|---|---|---|
| 0x00 | vtbl | IEntity 虚表（0x1003b784） | ctor |
| 0x28 | IWinSocket* | WinSocket 组件（`GetCore()->CreateComponent("WinSocket")`，vtbl[0x68]） | Init@0x1002c8f0；Shut 释放 |
| 0x2c | bool | UseQuickLog | Start 分支一 |
| 0x2d | bool | UseAppRecord | Start 分支二 |
| 0x2e | bool | LogDBPerformance | CAccess ctor 第二参 |
| 0x30 | int | m_State（0=Unknown/1=Opened/2=Closed） | GlobalStateMachine |
| 0x34 | TFastStr | ConnectString（[Main]） | SetConnectString@0x1002ca80 |
| 0x54 | CAccess* | m_Access（Start 尾 new CAccess(0xC)） | Start@0x1002d389 |
| 0x58 | TFastStr | OuterAddr | Start CreateListener |
| 0x78 | int | OuterPort | 同上（[ebp+0x78] 入参） |
| 0x7c | TFastStr | EchoAddr | Start echo 分支 |
| 0x9c | int | EchoPort | AppRecord "%s:%d" |
| 0xa0 | TFastStr | AppName（"mutex_%s" 单实例名） | Start@0x1002cee5 |
| 0xc0 | TFastStr | LogLauncher | Start 快日志分支 |
| 0xe0 | domain[8] | `{char name[0x24]; char table[0x24]}` 共 0x240 | SetDomain@0x1002ccc0 |
| 0x320 | TArrayPod<client_t*> | m_Clients | AddClient@0x1002dfa0 |
| 0x334 | int | 连接计数 | AddClient/RemoveClient |
| 0x338 | IEntity* | m_pDBSystem（SetDBSystemID 经内核解析） | GetDBSystem@0x100284c0 |
| 0x33c | CReactor* | ctor new(0xE4) | Execute/Start |
| 0x340 | CEchoHook* | ctor new(0x414)，(this, reactor) | SendEchoLine@0x1002dee0 |
| 0x344 | CLogShareMem* | 快日志（UseQuickLog） | Start/Stop |
| 0x348 | CAppRecord* | 进程记录（UseAppRecord） | TimerCheck 状态行 |
| 0x34c | int | OuterListenerId（-1 初始） | Start/Stop |
| 0x350 | int | TimerId（-1 初始，1.0s） | Start/Stop |
| 0x354 | CLockVal | DatabaseBreak（"Break"/"OK"） | TimerCheck@0x1002f06d |
| 0x358 | CLockVal | DatabaseError 计数 | TimerCheck db_err |

`client_t`（new 0x40C + memset 0）：`+0` 未用/保留，`+8 char buf[0x400]`，
`+0x408 int len`（行缓冲，上限 0x3FF）。

### FxModule 导出（0x10029a30 系）

`GetType()="FxModule"`、`GetVersion()=0x3EB(1003)`、`GetSpace()=""`、
`GetEntCreator()=g_pEntCreator`、Int/Logic/FuncCreator=NULL、
`Init(p)=全局 0x10048e88=p`。

### 属性/方法注册表（prop_struct 系符号 + rdata 名串）

属性：`State UseQuickLog UseAppRecord LogDBPerformance ConnectString
OuterAddr OuterPort EchoAddr EchoPort AppName LogLauncher DBSystem
DatabaseBreak`；方法：`Break SetDomain Start Stop SetAppState
SetConsoleColor WriteLine SendEchoLine CloseEcho DumpReactor
ClearDatabaseError`。`DBSystem` setter：IVar→PERSISTID→SetDBSystemID。

## 三、生命周期

### Start()@0x1002cea0（顺序实证）

1. `OpenMutex("mutex_"+AppName)` 失败 → `printf("...exclusive failed\r\n")` + `exit(0)`
2. UseQuickLog → new CLogShareMem：SetAppId(0x38)、Start、SetLogPrefix(AppName)、SetCreateNew(1)、挂全局日志、GetExists()==0 且 LogLauncher 非空 → `Port_Execute(LogLauncher)`（拉起 game_log）
3. UseAppRecord → new CAppRecord：SetAppId(0x38)、SetAppName(AppName)、SetAppAddr(`"%s:%d"`=EchoAddr:EchoPort)、Start
4. `CReactor::Start()`
5. `CreateListener(OuterAddr, OuterPort, accept=100, 0x400/0x400/0x4000, on_accept=AddClient, on_close=RemoveClient, on_recv=ReceiveClientMsg, this, 1, 0)` → m_OuterListenerId
6. `CEchoHook::Start(EchoAddr, EchoPort)`
7. `CreateTimer(cb=TimerCheck, this, 1.0f)` → m_TimerId
8. 旧 echo 口（this+0x338 误复用检查）……实际为 m_pDBSystem 非空时 Shutdown 前置
9. `m_Access = new CAccess(this, LogDBPerformance)`；`SetConnectString(ConnectString)`；`Start()` 失败仅 trace（"start access failed"+连接串），不中断
10. `GlobalStateMachine(1,0,0)`（Unknown→Opened）
11. `CORE_TRACE("SoloGlobal pointer is 0x%p")`；返回 true

### Stop()@0x1002da50

删监听→删定时器→CEchoHook::Stop→CReactor::Stop→CAppRecord
停+释放→快日志还原+释放→`CAccess::Stop()` 失败仅 trace
（"stop access failed"+连接串）+释放→m_pDBSystem->vtbl[0x24]。

### Execute()@0x1002ca20 = `CReactor::EventLoop()`；Init()@0x1002c8f0 =
`GetCore()->CreateComponent("WinSocket")`；Shut()@0x1002c990 = 释放 WinSocket。

## 四、外网命令通道（行协议）

### ReceiveClientMsg@0x1002e370

逐字节：`\r`/`\n` 任一 = 行终止；行缓冲非空才分发
（`client->buf[len]=0` → `ReceiveCommand(index, buf, len)`，len 清 0）；
超 0x3FF 丢字节。空行不分发（与 echo hook 同方言）。

### ReceiveCommand@0x1002e570（命令分派）

```text
断言 index>=0、pdata!=NULL；m_Clients[index]==NULL → trace "state error"、false
text_decode_msg(pdata,size) 失败 → "decode message error"、false（无应答）
GetCount()<2 → "message too short"、false（无应答）
arg0=StringVal(0)（回显标记） cmd=StringVal(1)
应答 vl = [arg0, cmd]
cmd=="add_name"（strcmp@0x1002e725）：
  GetCount()<6 → "need 6 arguments"、false（无应答）
  domain=StringVal(2) name=WideStrVal(3) owner=WideStrVal(4) server=IntVal(5)
  table=GetDomainTable(domain)；空 → "no domain table"+domain、false（无应答）
  应答 += [table(str), name(widestr)]；ok=m_Access->AddName(table,name,owner,server)
  应答 += AddInt(ok?1:0)
cmd=="remove_name"（strcmp@0x1002e8c1）：
  GetCount()<4 → "need 4 arguments"、false（无应答）
  domain=StringVal(2) name=WideStrVal(3)
  table 空 → 同上；应答 += [table, name]；ok=m_Access->RemoveName(table,name)
  应答 += AddInt(ok?1:0)
其余 cmd：应答仅 [arg0, cmd]（空结果也回包）
text_encode_msg(应答, buf 0x3FE) <0 → "encode msg failed"、false
buf += "\r\n"（0x0D 0x0A 两字节追加）
CReactor::Send(conn=index, buf, len, 0) 失败 → "send msg failed"、false；否则 true
```

编码即 `internal/charge` 的 VarList 文本方言（text_encode_msg/
text_decode_msg 同源：空格分隔、`$` 窄串、`#` 宽串、%d/%f）。

### AddClient@0x1002dfa0 / RemoveClient@0x1002e1e0

`new client_t(0x40C)` → `m_Clients.Resize(index+1)`（空槽扩展）→ 断言
槽空 → 存入 → 计数++ → trace `"address %s:%d connected, total %d"`；
移除：delete + 置空 + 计数-- → `"address %s:%d closed, total %d"`。

## 五、域表与 SQL（access.cpp）

### SetDomain@0x1002ccc0 / GetDomainTable@0x1002cdc0

```c
bool SetDomain(int index, const char* name, const char* table)
    // index>=8 → false；strncpy(dst=this+0xE0+index*0x48, name, 0x24)
    //                            +0x24 → table, 0x24（定长 0x24 两槽）
const char* GetDomainTable(const char* name)
    // 线性扫 8 槽 strcmp(name) 命中返回 table 槽；否则返回 ""
```

main.lua 实配：`SetDomain(0,"role","nx_role_names")`、
`SetDomain(1,"guild","nx_guild_names")`。

### CAccess（this 0xC：+0 pOwner / +4 IDBAccess* / +8 失败计数）

`ctor(pOwner, logPerf)`：`m_pDB = pOwner->GetDBSystem()->vtbl[0x28](logPerf)`
（DBSystem 实体派发 DB 访问对象——Go 复刻为 DBAccess 接口注入）。
`SetConnectString→vtbl[0x14]`、`Start: vtbl[0xC] 连接 + vtbl[0x30] 校验
（失败→SetDatabaseBreak(1)）`、`Stop: vtbl[0x34]+vtbl[0x10]`、
`CheckConnect: !vtbl[0x3C]→IncDatabaseError；返回 vtbl[0x38]`、
`FailCanRetry: vtbl[0x40]; m_FailCount<2`。

### AddName@0x100289b0 / RemoveName@0x10028f50（同一重连循环骨架）

```c
// 名/主先 Port_WideStrToUTF8（0x80 缓冲）再拼 SQL
while (true) {
    CheckConnect();
    if (ConnectFailed()) {                 // 连接已断
        if (!FailCanRetry()) { trace "...connection error"; break; } // ≥2 次放弃
        m_FailCount++; continue;           // 重试
    }
    snprintf(sql,0x400,"SELECT name FROM %s WHERE name = '%s' ",table,name8);
    rs = m_pDB->vtbl[0x48](sql);           // 查询
    eof = rs ? rs->vtbl[0xC]() : (异常路径)；rs->vtbl[0x4]() 释放
    AddName:    if (eof)  { INSERT; ok=true; }   // 名不存在 → 登记
    RemoveName: if (!eof) { DELETE; ok=true; }   // 名存在 → 释放
    查询异常 → trace "(CAccess::AddName|RemoveName)failed" + CExceptMsg::GetMsg
             + vtbl[0x2C] 错误串 → ConnectFailed 分支同上（计数重试）
    break;
}
```

SQL 原文（rdata 0x1003b498/0x1003b504/0x1003b5a4）：

```sql
SELECT name FROM %s WHERE name = '%s'
INSERT INTO %s ( name, owner, server_id, create_time ) VALUES ( '%s', '%s', %d, NOW() )
DELETE FROM %s WHERE name = '%s'
```

快照连接串（game_global.ini）为 SQLOLEDB 方言
（`Provider=SQLOLEDB;...;Initial Catalog=nx_base1;...`），main.lua 创建
的是 `MysqlDBSystem`——同名表族见 Res/mysql（nx_role_names 等）；
Go 复刻按 MySQL 落地（NOW() 原样），表名/列名/顺序零改动。

## 六、状态机（sologlobalstate.cpp）

```c
GlobalStateMachine(event,a1,a2)          // this+0x30 分派
  Unknown:  event==1 → m_State=Opened（Start 调用）；event==0 → 忽略；
            其他 → trace "GlobalStateUnknown)error input %d"
  Opened:   event==0 → 忽略（TimerCheck 每秒）；其他 → trace error
  Closed:   event==0 → 忽略；其他 → trace error
GetState() → "Unknown"/"Opened"/"Closed"
```

Closed 无官方迁入路径（仅枚举/呈现）——Go 复刻保持同构。

## 七、TimerCheck@0x1002f010（1.0s）

```c
if (m_pAppRecord) {
    g_counter++;
    if (g_counter > 5) { g_counter = 0;
        db = CLockVal(+0x354) ? "Break" : "OK";
        snprintf(buf,0x100,
          "<info state=\"%s\" database=\"%s\" db_err=\"%d\" tick=\"%d\" />",
          GetState(), db, CLockVal(+0x358), Port_GetTickCount());
        m_pAppRecord->SetAppInfo(buf);     // 写共享内存运行记录
    }
}
GlobalStateMachine(0,0,0);                 // tick 事件
```

## 八、echo 维护口（CEchoHook + main.lua）

行协议与外网通道同方言（\r/\n 终止、1023 上限、空行不分发；
SendLine 原样发送）。main.lua 回调：连接开 →
`WriteLine("echo client addr:port connected")` +
`SendEchoLine("welcome global server\r")`；命令 `quit`（回应
"server will shutdown..." + close()：WriteLine("global server
shutdown...")→Stop()→nx_quit()）与 `end`（CloseEcho）；
另有 getp/setp/getc/setc/method/listp/listc/listm/listg/function/
debugger 通用实体自省族（FxCore 脚本层标准件）。

## 九、Go 复刻（internal/fssologlobal + cmd/global）

线上字节一致面：外网行协议（text_encode_msg+CRLF）、命令语义与应答
五元组、SQL 三句原文、域表 8 槽 0x24 定长语义（GBK→UTF-8 落地差异见
下）、状态机三态、echo 口欢迎语与 quit/end。

进程内等价面（原则同 fssoloauction）：

| 官方 | Go 等价 |
|---|---|
| CReactor(IOCP) | internal/reactor |
| CEchoHook | fssoloauction.EchoConsole 模式（本包自带 echo.go） |
| CAppRecord/CLogShareMem 共享内存 | 日志行呈现（`<info …/>` 每 6s） |
| DBSystem 实体 + CAccess vtbl 面 | `DBAccess` 接口：SQLStore（database/sql MySQL）或 LocalStore（内存+文件快照，无库时兜底） |
| mutex_%s 单实例互斥 | 不复刻（Go 进程管理由部署面承担） |
| Port_Execute(LogLauncher) 拉起 game_log | 不复刻（日志由进程自身承担） |
| OLEDB 连接串 | MySQL DSN（-mysql-dsn / [Main] ConnectString 原样读取；检测到 Provider= 前缀时告警并要求 -mysql-dsn） |

cmd/global 旗标：`-config`（game_global.ini 方言：[System]/[Module]/
[Script]/[Main]/[Outer]/[Echo]）、`-outer`、`-echo`、`-mysql-dsn`、
`-local-store`（默认 data/global 名字快照兜底）、`-run`。

## 十、验证

- `go test ./internal/fssologlobal`：ini 方言（官方 game_global.ini 原文
  回归+缺省值表）/域表 8 槽边界与查名/编解码往返（charge 方言复用）/
  状态机全迁移+非法事件/TimerCheck 状态行（含 Break/OK 双态）/
  LocalStore 登记-冲突-释放语义/真 TCP 全链（连接欢迎语、add_name
  首登/重名/越权域/remove_name 存在/不存在、未知命令空应答、
  \r 与 \n 双终止、超长行丢弃、多客户端并发槽位）
- 实机冒烟：`./build/game_global.exe`（净目录释放内置 game_global.ini
  → LocalStore 兜底）+ `game_probe` 风格手工会话见任务日志
