# FsSoloMonitor.dll + game_monitor.exe 全解与 Go 复刻(反作弊监控服务器)

> 对象:`BGserverCJ\Debug\FsSoloMonitor.dll`(946176 B,2013-07-05 构建,
> Debug 全符号 .map/.pdb,源码路径 `c:\wuyf\svn\serveriii\branches\publish20130508\server\fssolomonitor\`)
> 日期:2026-11-19　分支:`official`
> 逆向中间产物:`tmp_research/solomonitor/`(sm_dis.py 注释反汇编器 +
> strings_all.txt + notes_{A_entity,B_business,C_tables,D_infra}.md 四层笔记)

---

## 一、定位与宿主

**game_monitor.exe**(MD5 `0a51cad15bf2bd22184e2822f1744fd8`,与
game_log/lister/entry 壳 `195e1ce1…` 不同构建但同族)按自身文件名读
`game_monitor.ini`:

```ini
[System]
Sleep=-1            ; 主循环不休眠(1.0s reactor 定时器驱动)
MainEntity=SoloMonitor
MainScript=main
ServerId 缺省(→0)
[Module]
fskit.dll
fssql.dll           ; MysqlDBSystem 插件(nx_create("MysqlDBSystem"))
fscomm.dll
FsSoloMonitor.dll
[Script]
Path=..\lua\solomonitor
[Main]
ConnectString=Provider=SQLOLEDB;…Initial Catalog=nx_base1;…   ; 仅呈现,真库见 §七
[Outer]
Addr=127.0.0.1
Port=4008           ; 外网命令受面
; 无 [Echo] 节 → lua 本地缺省 127.0.0.1:15012
```

**FsSoloMonitor.dll 是 solo 集群的防作弊监控服务器**(lua main.lua 首行
注释"全局名字服务器主控脚本"系从 FsSoloGlobal 复制残留,与实现无关):
各 game_world(FsGame.dll,内含 `alert_total_capital_new`、
`first_send_moniter_msg` 等发送侧字符串)与 game_store 经文本行协议把
玩家资金/物品/登录流水上报到 Outer 口;本进程在内存中维护 7 张监控表,
按 `Res\MonitorLimit.ini` 限额判定异常,告警三路出口:CORE_TRACE 日志、
`alert_monitor` echo 广播、MySQL `nx_monitor` 表落库。appid=**0x32(50)**
(LogShareMem/AppRecord 双登记)。

导出 8 个标准 Fx 模块契约(同 FsSoloLog:Type="FxModule"、
Version=0x3EB、Space=""、Init(core) 存 g_pCore,四 creator 仅 EntCreator 非空)。

## 二、模块清单(.map obj 归组)

| obj(符号数) | 官方类/源文件 | 职责 | Go 落点 |
|---|---|---|---|
| SoloMonitor (493) | `SoloMonitor : IEntity`(solomonitor.cpp,尺寸 **0x560**) | 实体本体:19 属性+18 mid 方法+命令分派+TimerCheck | `internal/fssolomonitor` |
| SoloMonitorState | MonitorStateMachine + 三状态函数 | Unknown→Opened 单迁移 | state.go(同 fssologlobal) |
| RoleDataTable (94) | `CRoleDataTable`(roledatatable.cpp) | 玩家资金/物品/交易累计(**核心表**) | roledatatable.go |
| CrateItemTable (75) | `CCreateItemTable`(crateitemtable.cpp) | 系统产粮滚动窗口 | createitemtable.go |
| CrateItemDetailTable (38) | `CCreateItemDetailTable` | uuid 环形查重 | detailring.go(泛型) |
| TradeItemDetailTable (53) | `CTradeItemDetailTable` | 同玩家同编号重复交易 | tradeitemdetail.go |
| LoginDetailTable (39) | `CLoginDetailTable` | 同 MAC 账号数/在线数 | logindetail.go |
| RecvLetterDetailTable | `CRecvLetterDetailTable` | 邮件附件重复提取 | detailring.go |
| GlobalItemDetailTable | `CGlobalItemDetailTable` | 全服物品净数量/堆叠 | globalitemdetail.go |
| Access (67) | `CAccess`(access.cpp) | DB 写入线程 + nx_monitor INSERT | access.go |
| TaskQueue (35) | `CTaskQueue`(taskqueue.cpp) | 双缓冲消息队列(32MB 上限) | chan(等价) |
| Routine (57) | put/get_vary_message(routine.cpp) | IVarList↔二进制 tag 流 | 本模块未上线路径[注1] |
| TextProtocol (25) | text_encode/decode_msg(tools) | 行协议方言(§十) | internal/charge 既有 |
| ReadIni (122) | `CReadIni`(utils) | MonitorLimit.ini 解析 | config.go |
| AppRecord (128) | `CAppRecord/CShareMem` | 跨服进程登记簿共享内存 | 状态行日志等价 |
| EchoHook (152) | `CEchoHook`(echohook.cpp) | echo 维护口(**多客户端版**) | echo.go |
| Reactor (255) | `CReactor`(IOCP) | socket/定时器引擎 | internal/reactor(既有) |
| LogShareMem | `CLogShareMem` | 快速日志生产者 | docs/official/FsSoloLog全解与Go复刻.md |
| AppTool/MemPool/DllMain | 工具/池/泄漏检查 | `mem_fs_solo_mointor.log`(官方 typo) | 不复刻 |

[注1] Routine 的二进制序列化在本 DLL 中无调用点(xref=0),系 tools
公共代码随编译带入;真正上线的是 TextProtocol 文本方言。

## 三、SoloMonitor 实体(布局反汇编定案,ctor 0x1009d850)

`sizeof = 0x560`。五个字符串成员均为 `TFastStr<0x10>`(模板参 0x10,
长串走堆)。

| 偏移 | 成员 | 说明 |
|---|---|---|
| +0x28 | void* m_pWinSocket | Init 经 core vt+0x68 QueryPlugin("WinSocket") |
| +0x2c/2d/2e | bool UseQuickLog / UseAppRecord / LogDBPerformance | main.lua 前二实配 true |
| +0x30 | int m_State | 0=Unknown 1=Opened 2=Closed |
| +0x34 | TFastStr ConnectString | → CAccess |
| +0x54 | CAccess* | Start 建,Stop 删 |
| +0x58 / +0x78 | OuterAddr / OuterPort | |
| +0x7c / +0x9c | EchoAddr / EchoPort | |
| +0xa0 | TFastStr AppName | "monitor_%d"(ServerId);mutex 名与日志 prefix |
| +0xc0 | TFastStr LogLauncher | "game_log game_log.ini daemon" |
| +0xe0..+0x320 | domain[8] × 0x48 `{char name[36]; char table[36]}` | lua:SetDomain(0,"capital","nx_role_capital")、(1,"item","nx_item") |
| +0x320 | TArrayPod<client_t*> m_Clients | 外网口客户端(容量随 index 扩) |
| +0x334 | int 客户端计数 | |
| +0x338 | IDBSystem* | lua DBSystem 属性(PERSISTID 经 core vt+0x7c 解析) |
| +0x33c | CReactor*(堆 0xE4) | Execute=EventLoop |
| +0x340 | CEchoHook*(堆 0x50) | ctor 即建 |
| +0x344 / +0x348 | CLogShareMem* / CAppRecord* | UseQuickLog/UseAppRecord 开启时 Start 建 |
| +0x34c / +0x350 | int listener / timer id | timer 未初始化[!]→Start 写入;-1=未建 |
| +0x354 / +0x358 | CLockVal DatabaseBreak / DatabaseError 计数 | Get/Set/Inc/Clear 四方法 |
| +0x35c..+0x374 | 七表指针(见 §五注册名) | ctor 全部 new + SetWarningCallback(0x1009e580, this) |
| +0x378 / +0x37c | int 当前 nLogType / 当前 client index | ReceiveCommand 每条刷新;OnReportWarning 文本引用 |
| +0x500 | int ServerId | |
| +0x504 / +0x508 | int FaculityLimit / LivepointLimit | |
| +0x50c | int CheckTimerInterval | ctor 缺省 7200;ini 缺省 1200 |
| +0x510 | int64 PlayerDiffCapitalLimit | 玩家间资金差巡检阀值 |
| +0x518..+0x550 | int64 ×8 资金差累加器 | 强化装备/直接交易/乞讨×2/画像/书生/摆摊/合成 |
| +0x558 | int GlobalSwitch | >0 才启用 alert_total_capital 告警 |

`client_t`(0x42C):+0 int state(恒 0,"state error" 分支死码)、
+8 行缓冲 buf[0x400]、+0x408 len、+0x40c "ip:port"。

### 3.1 属性/方法注册(静态 prop_struct/func_struct 链)

19 属性:State(只读串)/UseQuickLog/UseAppRecord/LogDBPerformance/
DatabaseBreak/ConnectString/OuterAddr/EchoAddr/AppName/LogLauncher(串)、
OuterPort/EchoPort/ServerId/RoleTradeCounterLimit/RoleItemCounterLimit/
CheckTimerInterval/GlobalSwitch(int)、DBSystem(PERSISTID)、
PlayerDiffCapitalLimit(int64)。flag:**1=bool 2=int 3=int64 6=string
8=PERSISTID**。

18 mid 方法:SetDomain/Start/Stop/SetAppState/SetConsoleColor/WriteLine/
SendEchoLine/CloseEcho/DumpReactor/ClearDatabaseError/Set+GetItemCreateLimit/
Set+GetTableCapacity/Set+GetRoleCapitalIncLimit/Set+GetRoleItemCounterLimit。

### 3.2 生命周期

- **Init**:`QueryPlugin("WinSocket")` 失败→false;否则
  `LoadSystemConfig(工作目录 + "..\Res\MonitorLimit.ini")`。
- **Execute**:每帧 `CReactor::EventLoop()`。
- **Shut**:仅 Release WinSocket。
- **Start**(0x100a0300,任一失败 CORE_TRACE 后返回 false):
  1. `CMutex::Exclusive("mutex_%s", AppName)` 失败 → printf + **exit(0)**;
  2. UseQuickLog→CLogShareMem(appid 0x32,prefix=AppName,create_new=1,
     SetTraceFunc);`GetExists()==0 && LogLauncher 非空`→`Port_Execute(LogLauncher)` 拉起 game_log;
  3. UseAppRecord→CAppRecord:appid 0x32、AppName、AppAddr="%s:%d"(echo);
  4. `CReactor::Start`;
  5. `CreateListener(OuterAddr, OuterPort, accept=100, in/out=0x400, outmax=0x4000, on_accept=AddClient, on_close=RemoveClient, on_recv=ReceiveClientMsg, this, 1, 0)`→+0x34c;
  6. EchoAddr 非空→`EchoHook::Start(EchoAddr, EchoPort)`;
  7. `CreateTimer(1.0f, TimerCheck, this)`→+0x350;
  8. DBSystem 非空先 vt+0x20(启动);`new CAccess(this, LogDBPerformance)`
     →SetConnectString→**Start 失败仅 trace 不返回 false**;
  9. `MonitorStateMachine(1, NULL, 0)` → Opened。
- **Stop**(0x100a0e40):DeleteListener→DeleteTimer→EchoHook.Stop→
  Reactor.Stop→AppRecord 停删→core vt+0x58(0,0)+LogShareMem 停删→
  Access.Stop(失败 trace 连接串)→DBSystem vt+0x24(停止,不删)。

## 四、外网命令通道(Outer :4008)

`ReceiveClientMsg(index, data, len)`:逐字节入行缓冲(**\r 或 \n 皆终止,
0x3FF 截断,空行不分发**)→`ReceiveCommand(index, line, len)`。

`ReceiveCommand`(0x100a19a0):`text_decode_msg` 解 IVarList(§十方言),
`vl[0]`=命令名,trace `recv message type:%s`;每条置 +0x378=0/+0x37c=index。

| 命令 | 参数(vl 下标) | 处理 |
|---|---|---|
| `createitem` | [2]=itemUid(窄) [3]=role(宽) | `MonitorCreateItem`(§六.4,Debug 遗留基准) |
| `item_log` | 10 元素:role# 配置id# itemUid(窄) 数量(I64) logType(int) itemType(int) maxStack(int) 源role# 备注# 备注2# | `ProcessItemLog`(§六.1) |
| `entry_scene` | [1]=role# [2..4]=三种货币存量(I64) [5..]=逐背包物品数(I64) | `EntryCheckRoleData`→RoleData 表 Update 基线 |
| `leave_scene` | 同上 | `LeaveCheckRoleData`→RoleData 表 Check 比对 |
| `faculty` | [1]=role# [2]=修为(int) | >FaculityLimit → 告警(玩家修为值超过阀值) |
| `alert_total_capital` | 每记录 8 元素×N:(count-1)%8==0 否则 trace format failed;int 资金类型, int 开关, I64×6=系统总可发行/可购买/可奖励/玩家保有/离线/流通中 | trace;GlobalSwitch>0 且 不平衡 → OnReportWarning(type 0) |
| `alert_total_capital_new` | int 告警子型 int 资金类型 I64 当前值 I64 阀值 | 子型 1..7 → 七种中文告警(系统资金<0/可购买<阀值/可奖励<阀值/离线<阀值/用户<1锭/差额>1锭/在途<1锭)→OnReportWarning(0) |
| `loginfo` | mac(窄) addr(窄) acc(窄) role# | `CLoginDetailTable::Login` |
| `logout` | 同上 | `CLoginDetailTable::LoginOut` |
| 其它 | — | trace unknown;返回 true(不断链) |

断言/trace 里的 `(SoloGobal::ReceiveCommand)` 拼写为官方源码原文。
FsGame.dll 侧为发送方(world 进程),FsSoloStore.dll 亦含 item_log。

## 五、监控表体系(7 表)

通用:`SetWarningCallback(fn, ctx)`,表内
`ReportWaring(wsRole, type, fmt, …)` = `vsnprintf(buf,0x100)` 后
`fn(ctx, wsRole, type, buf)` → SoloMonitor::OnReportWarning(§七)。
注册名(SetTableCapacity 分派键,0x1011f0fc 起):

| 注册名 | 表指针 | 源类 | ctor 容量 | 附加容量 |
|---|---|---|---|---|
| CreateItem | +0x35c | CCreateItemTable | 10000 | — |
| CreateItemDetail | +0x360 | CCreateItemDetailTable | 0 | SetCapacity(**1000000**) |
| RoleData | +0x364 | CRoleDataTable | 100000 | — |
| RecvLetterDetail | +0x368 | CRecvLetterDetailTable | 0 | SetCapacity(1000000) |
| TradeItemDetail | +0x36c | CTradeItemDetailTable | 0 | SetCapacity(100000) |
| ItemDetail | +0x370 | CGlobalItemDetailTable | 0 | SetCapacity(**0**=停用) |
| LoginDetail | +0x374 | CLoginDetailTable | 0 | SetCapacity(100000) |

两类容器骨架:
- **detail 环表**(Create/RecvLetter/Trade/Login/Global 五张):
  `capacity 定长记录环 + TRefStringIndex 键索引`;满后 cursor 回绕覆盖
  最老记录并 Remove 旧键;capacity=0 时 Add 直接 false(即 ItemDetail
  默认停用)。
- **大数据表**(RoleData/CreateItem):MemPoolAlloc 变长行 + 指针数组 +
  哈希索引;超 capacity(且 mem_limit==0 或 mem_used≥mem_limit)时
  `ReplaceTimeoutData` 线性找 `time` 最老行换名复用。

### 5.1 CRoleDataTable(核心,0x80 字节对象)

表级:cap=100000 / mem_limit / count / mem_used / 行数组 / 名→下标索引 /
回调 / **capital_inc_limit[8](u64)** / trade_counter_limit(u32) /
item_counter_limit(u32) / 每物品覆盖阈值 pod(128 桶) /
check_timer_interval(**3600s**)。

行 `monitor_role_data`(0x1c0+名长):

```
+0x00 time     最近活动(所有写入刷新;换行依据)
+0x04/+0x08    item_from_system_amount / count(系统取物)
+0x0c pod      核心物品计数(懒建,TimerCheck 清)
+0x10/+0x14    trade_amount / trade_count
+0x18 pod      交易 configId 计数(懒建)
+0x1c pod      已交易 uuid 集合(懒建;BuyBackItem 移除)
+0x20 capital[8] × 0x30:
      +0x00 login_value(上线基准,-1 起)  +0x08 expected_value(推算)
      +0x10 inc_total(窗口增量,清零式)    +0x18 inc_op_count
      +0x20 dec_op_count                  +0x28 dec_total
+0x1a0 bag[3](int64,初始 -1)
+0x1b8 last_check(上次窗口清算)
+0x1bc wchar name[]
```

算法:
- `UpdateCapital`(上线):login≠val 时 diff=|expected−val|≠0 →
  **type 7**"玩家上线资金检查异常";随后 login=expected=val(重置基线)。
- `CheckCapital`(下线):同式 **type 8**"玩家下线资金检查异常"(只读)。
- `IncCapital(role,type,amt,bInc)`:expected+=amt;bInc 时 inc_total+=amt、
  inc_op_count++(item_log 的 30010/30027/35002 等系统邮件类 bInc=false);
  尾调 `TimerCheckRoleData`。
- `DecCapital`:expected−=amt;dec_total+=amt;dec_op_count++(不触发巡检)。
- `AddItem(role,cfg,amt,srcType)`:srcType==28010 跳过核心物品;查表级
  覆盖阈值 pod 有则 `AddCoreItem`(计数>limit → **type 6** 玩家获取核心
  物品超阀值);行计数累加;尾调巡检。
- `TradeItem(role,cfg,uuid,count)`:`TradeItemUuid` 该 uuid 已交易过 →
  **type 2**"同一序列号物品玩家重复交易" 且返 false;否则 configId pod 累计。
- `BuyBackItem`:uuid pod Remove(回购解禁)。
- `UpdateItem/CheckItem`(bag 0..2):上线 **type 0xb**"背包物品校验警告";
  下线只写值。
- `TimerCheckRoleData(row)`(IncCapital/AddItem 尾部惰性触发):距
  last_check ≥ interval 才执行:①8 币种 inc_total>limit → **type 0**
  "玩家资金增长超过阀值"(清零);②trade_count>limit → **type 0xa**
  "交易操作次数超过阀值"(清零);③item_count>limit → **type 5**"从
  系统中获取物品次数超过阀值"(清零);④清核心物品 pod。

### 5.2 其余六表

| 表 | 行/记录 | 判定与告警 |
|---|---|---|
| CCreateItemTable | 行 0x10={窗口起点,窗口累计量,次数}+cfgId | AddItem(cfg,amt,role):滚动窗口(超 interval 清零重开);amount 或 本次增量 > 阈值 → **type 1**"系统创建物品超过上限值";无覆盖阈值的物品只刷时间不告警(无全局默认值);Hour/DayStat 空实现 |
| CCreateItemDetailTable | 43B 定长=uuid[0x2b] | Add:uuid 已存在 → **type 9**"系统创建了重复序列号的物品";环写查重 |
| CTradeItemDetailTable | 行 0x48={wsName[34], configId→detail 子索引(懒建)};detail 0x34={cfg[43], 次数 u8, 累计价格 i32, free 链} | Buy/SellItem(cfg,price,…,wsTrader,…):同玩家同编号已有明细 → 累计价格>0 → **type 0xd**"物品交易获取出错";free-list 复用 detail 结点 |
| CLoginDetailTable | 行 0x9c={mac[0x80], ip[0x10], 在线数 u32, ?, 账号→次数 pod} | Login:pod 记账号次数;不同账号数>RoleNumLimit → **type 0xe**"同一机器登陆账号过多";在线数>OnlineRoleNumLimit → **type 0xf**"同一机器同时登陆账号过多";LoginOut 在线数−1 |
| CRecvLetterDetailTable | 记录 0x2d={计数A u8, 计数B u8, letterId[0x2b]} | Add(letterId,…,attachType):10032/10042→A++;30010/35002→B++;其它→A++;A>1 或 B>1 → **type 4**"重复提取系统邮件附件" |
| CGlobalItemDetailTable | 记录 0x30={cfgId, 净数量 i32} | Buy:净数量+=|n|;Sell:−=|n| 且 n<0 → **type 0xc**"物品操作的数量非法";|净数量|>堆叠上限 → **type 4**"物品增加/减少的数量已经超过堆叠上限" |

### 5.3 告警 type 码汇总(nx_monitor.type 列)

| type | 来源 | 含义 |
|---|---|---|
| 0 | RoleData 巡检 / SoloMonitor | 资金增长超阀值 / 全局资金不平衡、faculty、capital_new 七型 |
| 1 | CreateItem | 系统创建物品超上限 |
| 2 | RoleData | 同序列号物品重复交易 |
| 3 | SoloMonitor TimerCheck | 玩家间资金差 8 类超阀值 |
| 4 | RecvLetter / GlobalItem | 重复提取邮件附件 / 数量超堆叠 |
| 5 | RoleData | 从系统获取物品次数超阀值 |
| 6 | RoleData | 核心物品获取超阀值 |
| 7 / 8 | RoleData | 上线 / 下线资金校验偏差 |
| 9 | CreateItemDetail | 系统创建重复序列号物品 |
| 0xa / 0xb | RoleData | 交易次数超阀值 / 背包校验偏差 |
| 0xc | GlobalItem | 物品操作数量非法 |
| 0xd | TradeItemDetail | 同玩家同编号重复交易 |
| 0xe / 0xf | LoginDetail | 同机器账号数 / 同时在线数过多 |

## 六、业务流水

### 6.1 ProcessItemLog(0x100a4c60)——item_log 分派

10 变元(§四表);`GetCapitalType(configId)`=**取末字符 atoi**
("Capital3"→3;空串→-1)。先记 +0x378=nLogType、
`ProcessPlayerCapital`:35001(货币减少)→+0x520 累加器−=amt、
45001(货币增加)+=amt(全服净流量,配 TimerCheck 61 拍巡检)。

nLogType 开区间分派(边界值 10000/20000/28000/30000/34999/35000/
36029/40000/44999/45000/46029 落空不处理):

```
(10000,20000) 物品获得:
    10031/10032/10035/10037 → RoleData.BuyBackItem(回购解禁)
    否则 CreateItem.AddItem(cfg,amt,srcRole);
         maxStack==1 → CreateItemDetail.Add(cfg,uuid,role)
    RoleData.AddItem(role,cfg,amt,nLogType)
    GlobalItem.BuyItem(cfg,amt,maxStack,role)
(20000,28000) 物品消耗/交易:
    maxStack==1 → RoleData.TradeItem(role,cfg,uuid,amt)
    GlobalItem.SellItem(cfg,amt,maxStack,role)
(30000,34999) 货币增加:  RoleData.IncCapital(role,type,amt, bInc=!(30010||30027))
(35000,36029) 货币增加2: RoleData.IncCapital(role,type,amt, bInc=!(35002||30027))  // 30027 分支死码[官方复制残留]
[40000,44999)∪[45000,46029) 货币减少: RoleData.DecCapital(role,type,amt)
第二趟(明细):
    30010/10042/10032/35002 → RecvLetter.Add(备注,源角色,nLogType,amt)
    10031/35001            → TradeItem.BuyItem(…)
    20058/45001            → TradeItem.SellItem(…)
```

### 6.2 Entry/LeaveCheckRoleData

同构解析(role#+三货币 I64+变长背包数):上线
`UpdateCapital×3 + UpdateItem×N`(建基线,偏差即告警);下线
`CheckCapital×3 + CheckItem×N`(差值核对)。

### 6.3 TimerCheck(1.0s 定时器,0x100a30d0)

1. UseAppRecord 时每 **6 拍**:SetAppInfo
   `<info state="%s" database="OK|Break" db_err="%d" tick="%u" />`;
2. `MonitorStateMachine(0,0,0)`(空转);
3. PlayerDiffCapitalLimit>0 时每 **61 拍**:8 个资金差累加器
   |diff|>limit → OnReportWarning(3,"S",对应文案)。
   CheckTimerInterval(+0x50c)在 TimerCheck 中**未被消费**(消费点疑在
   两表的 3600s 缺省,加载时经 SetCheckTimerInterval 下发,§八)。

### 6.4 MonitorCreateItem(0x100a3de0)——Debug 遗留基准测试

trace 后:wcslen(role)==0 → false;否则 **删掉 +0x360 真实明细表** →
new CCreateItemDetailTable 容量 1000 万 → TStringPod 与表 Add 各跑
1000 万次 `"kdlafj%c"`(26 字母表 rand%24)计时 → 结果不消费 → 删表
置 NULL → **恒返 false**。即生产中收到 `createitem` 命令会摧毁
CreateItemDetail 表(官方缺陷,Go 复刻按"收到即 no-op+日志"处理)。

## 七、告警出口与 DB 落库

### 7.1 OnReportWarning(0x100a3960)

```
bool OnReportWarning(int type, const wchar_t* wsName, const char* reason):
    name = UTF8(wsName)
    有当前 client → trace "[Warning],role:%s type:%d conn addr:%s logtype:%d  %s"
    否则           trace "[Warning],role:%s type:%d logtype:%d  %s"
    CAccess!=NULL → 入库消息(store_forw_insert_table_t):
        +0x00 int type; +0x04 wchar name[33]; +0x46 char reason[](柔性)
        PutMessage(0, 0, buf, 0x47+strlen(reason))    // 队满自旋 Sleep(0)
    vl << "alert_monitor" << wsName << type << reason → SendEchoMsg
        = text_encode_msg(0x800) → EchoHook.SendLine(0,0,…) 广播全部 echo 客户端
```

### 7.2 CAccess(access.cpp,写线程)

布局:+0 owner / +4 CThread / +8 CThreadWaiter / +0xc IDBSystem /
+0x10 CTaskQueue / +0x14 retry。**DLL 内无任何 mysql_\* 符号**——全部
经 IDBSystem 实体 vtable(Connect vt+0xc / SetConnectString +0x14 /
IsConnected +0x30 / CheckConnection +0x3c / Execute(SQL) +0x44 等)。

- 线程体 `DoWait`:空则 `Waiter.Wait(INFINITE)` → `SwapMsgVector` →
  循环 `Peek` → `ProcessMsg`(仅 msgid=0 → MessageInsertTable;失败且
  断连且 retry<2 重试,InitConnect 失败只 trace 不死)。
- `InitConnect`:Connect→IsConnected(失败 `SetDatabaseBreak(1)`)→
  `Execute(CREATE TABLE IF NOT EXISTS nx_monitor (name VARCHAR(128)
  CHARACTER SET utf8 COLLATE utf8_bin, type INT, reason TEXT, time
  DATETIME) ENGINE=InnoDB DEFAULT CHARSET=utf8)`(失败不致命)。
- `MessageInsertTable`:断连接重试判定 → name 宽→UTF8(0x80)、reason
  本地码页→UTF16→UTF8(**未做 SQL 转义,官方原样**)→
  `INSERT INTO nx_monitor VALUES('%s',%d,'%s',NOW())`。
- `SendResponse` **空桩 return 1**:fire-and-forget,无结果回投。

### 7.3 CTaskQueue(32MB 双缓冲)

记录 `{u32 size; i32 ident; i32 msgid}+payload`;生产端加锁 Put
(容量不足倍增扩容,≥32MB 拒收→PutMessage 自旋);消费端 Swap 后无锁
Peek。Go 等价:buffered channel + 单 goroutine。

### 7.4 CAppRecord(跨服进程登记簿)

共享内存 **`memory_fs_solo_app_book_%d`**(book 实例号 ctor=0)=
**64 槽 × 0x310,无文件头**;槽={bInUse, pid, state, nameHash,
AppName[0x80], AppAddr[0x80], AppPath[0x100], AppInfo[0x100]}。
`SetAppId(0x32)` 即槽号(appid 表:World=1/Public=2/Auction=3/
Monitor=0x32/Global=0x38 均 <64 自洽)。`Check()`:扫 64 槽,
`mutex_<AppName>` 命名互斥体消失 → bInUse=0(墓碑清理)。
TimerCheck 每 6 拍写 AppInfo 状态行(§6.3)。LogShareMem 生产者协议
见 FsSoloLog 文档(appid 0x32 → game_log 落 `50_<prefix>_<秒>.log`)。

## 八、配置全解

### 8.1 LoadSystemConfig(0x100a5a80)key→字段映射(CReadIni)

| INI 位置 | key | 缺省 | 落点 |
|---|---|---|---|
| [System] | CheckTimerInterval | 1200 | +0x50c;并下发 RoleData/CreateItem 两表 SetCheckTimerInterval |
| [System] | FaculityLimit | 900000 | +0x504 |
| [system](小写) | FaculityLimit(**key 串复用,官方笔误,本意 LivepointLimit**) | 1000 | +0x508 |
| [system] | DiffCapitalLimit | 15000000 | +0x510(int64) |
| [system] | ExchangeAmountLimit | 0 | RoleData.SetTradeCounterLimit |
| [system] | RoleNumLimit | 10 | LoginDetail.SetAmountLimit |
| [system] | OnlineRoleNumLimit | 3 | LoginDetail.SetOnlineAmountLimit |
| [SysItemLimit] | 逐条 key | atoi | CreateItem.SetItemCounterLimit(cfg,v) |
| [RoleItemLimit] | 逐条 key | atoi | RoleData.SetItemCounterLimit(cfg,v) |
| [RoleCapitalLimit] | CapitalTypeN | atoi→I64 | RoleData.SetCapitalIncLimit(N,v) |

注:CReadIni 哈希**大小写敏感**,"system" 小写节在官方 MonitorLimit.ini
并不存在 → Livepoint/DiffCapital/Exchange/RoleNum/Online 五键实际全部
取缺省值(官方 ini 与代码的双向失配,复刻保持行为一致)。
CheckTimerInterval 注释单位"分钟"而表层按秒消费(60→60s),官方即如此。

### 8.2 官方 MonitorLimit.ini 快照(Res\,2014-03-22)

[System]:60/900000/1000/1500000/0/10/3;[SysItemLimit] 门派内功残卷
ng_cjbook_*_002=20、003~006=1,高级残卷=1,江湖招式残卷=40;
[RoleItemLimit] 江湖武学 book_CS_*01..08=2,残卷 5/15,历练药
growitem_1=500 等,打造台/配方 2~60;[RoleCapitalLimit]
CapitalType0=1000/Type1=3000000/Type2=50000000。完整原文见快照目录。

## 九、状态机与 echo 口

- 状态机(solomonitorstate.cpp):`Unknown --(Start msg1)--> Opened`;
  msg0(每秒 tick)三态皆空转接受;其余事件 trace "error input %d"
  拒绝;**Closed 无迁入路径(保留态)**。GetState 返回
  "Unknown/Opened/Closed"。
- EchoHook(15012):与 FsSoloLog 版同参数同行协议(\r/\n、
  `on_input_<cmd>`→`on_input` 回退);差异全在**多客户端化**:上限
  **64 连接**(池化 client_t+connector 索引+空闲下标栈)、每客户端独立
  0x400 行缓冲、`SendLine(0,0,info)`=广播、`Close()` 批量关闭。
  lua 控制台命令:quit(关停)/end(断开)/getp/setp/getc/setc/method/
  listp/listc/listm/listg/function/debugger(实体属性与调试族)。

## 十、文本协议方言(text_encode/decode_msg)

单空格切分、无引号;token 首字符定型:数字(atoi→int/
StringToInt64→int64 域判定、含 `.`→double)、`#`宽串(`\\`、
`\uXXXX`、`\xXX` 转义)、`@a-b` 对象(PERSISTID 两 u32)、`*`base64
二进制、`$` 强制串、其余裸串。encode:空格分隔,`%d/%I64d/%f/#…/@%u-%u/*…`。
与 internal/charge 的 VarList 文本方言同源(FsSoloGlobal 实证同一套)。
仅两处调用:ReceiveCommand(decode)/SendEchoMsg(encode,0x800 缓冲)。

## 十一、Go 复刻设计(internal/fssolomonitor + cmd/monitor)

```
game_monitor.exe 等价壳 cmd/monitor(-config game_monitor.ini)
 └ internal/fssolomonitor
    ├ Server        — SoloMonitor 实体:装配/命令分派/TimerCheck
    ├ Config        — game_monitor.ini + MonitorLimit.ini(§八映射,含官方缺省失配行为)
    ├ RoleDataTable — 行=map[widestr]*roleData + 环形槽池;8 币种 6 计数器;巡检窗口
    ├ CreateItemTable — map[cfg]*窗口行 + 每物品阈值
    ├ DetailRing[T] — 通用"定长环+键索引"(CreateItemDetail/RecvLetter/Trade/Login/Global 五表参数化)
    ├ Access        — 告警队列(chan)+ 单 goroutine:EnsureTable→INSERT nx_monitor;DatabaseBreak/Error 状态对
    ├ StateMachine  — Unknown→Opened(复用 fssologlobal 形)
    └ EchoConsole   — 多客户端广播版(告警 alert_monitor 下行)
复用:internal/reactor(CreateListener/CreateTimer)、
internal/charge 的 VarList 文本方言、fssologlobal 的状态行/DiffCapital 巡检骨架。
```

要点:
1. **协议面**(必须字节级):行协议+方言编解码+10 命令分派+参数序
   (§四表即测试矩阵);告警下行 `alert_monitor <名#> <type> <reason>+$`。
2. **判定面**(行为级):7 表算法+16 型告警码+窗口清零语义+entry/leave
   基线;MonitorCreateItem 按官方语义降级为 no-op(记录日志)。
3. **落库面**:`CREATE TABLE IF NOT EXISTS nx_monitor` + INSERT(NOW()
   原生);reason 不转义保持官方行为(文档留痕,注入面=运维可控输入)。
4. **不复刻**:AppRecord 共享内存/命名互斥体(状态行走日志等价)、
   LogShareMem 生产者(日志接 slog)、ExitCode exit(0) 路径(返回错误)。
5. 测试计划:ini 回归(含 system 小写失配)/命令全型 golden/
   RoleData 窗口清算/环表回绕覆盖/断线重连计数/真 TCP 行协议端到端。

## 十二、未定事项

- client_t.state 恒 0("state error" 死码);+0x34c ctor 未初始化[!]。
- CheckTimerInterval 实体侧无消费点(仅下发两表;表层缺省 3600s)。
- CTradeItemDetailTable 的 1.5 浮点常量未见使用分支。
- monitor_login_detail 行 +0x94 计数器无读写点。
- GetCapitalType 对 "CapitalType12" 类多位后缀取末位(2)——官方即按
  末字符,复刻保持。
