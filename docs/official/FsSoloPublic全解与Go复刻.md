# FsSoloPublic.dll 全解与 Go 复刻

> 权威快照：`F:\Game\Online\9yin\BGserverCJ\Debug\FsSoloPublic.dll`（1,040,384 B，
> 2013-03-22）+ `FsSoloPublic.map`（5,502 符号，最终 VA 直读）+
> `FsSoloPublic.pdb`（4.1MB）。宿主 `game_public.exe`（通用 solo 壳）+
> `game_public.ini`（MainEntity=SoloPublic）；装配脚本
> `BGserverCJ\lua\solopublic\main.lua`（全文在手）。源码路径（assert 串）：
> `c:\wuyf\svn\serveriii\branches\publish20130508\server\fssolopublic\
> {solopublic,solopublicstate,solopublicworldmsg,solopublicstoremsg,
> solopublicmembermsg}.cpp` + `server\tools\innerhook.h`。
>
> 集群形态同源对照：《FsPublic全解与Go复刻.md》（FsPublic.dll 无 map，PDB
> 5,286 符号）。Go 复刻：`internal/fssolopublic` + `cmd/solopublic`；
> 数据层直接复用 `internal/fspublic`（archive/PubData/PubSpace/PUB4）。
> 反汇编底稿：`tmp_research/solopublic/`（disas.py + dis_*.txt 40 篇）。

## 1. 进程定位

solo 集群的**公共数据服务器**：持有跨服共享数据空间（domain/nx_domains +
guild/nx_guilds），并独占三大 **solo 特有**全局子系统（集群 FsPublic.dll 无）：

| 子系统 | 载体 | 语义 |
|---|---|---|
| 好友关系 + SNS 动态流 | CRelation（this+0x130） | 关系表 relation_t + feed 流（发动态/评论/赞踩/过期清理/feed 等级配置） |
| 寄售行 | CTrade（this+0x134） | 跨服寄售：Sell/Unsell/Buy/Search/GainProfit/CheckTimeout |
| 角色↔UID 双向表 | CNameUid（this+0x12c，ctor(0x7f)） | wchar 名 ↔ char UID 哈希双向索引（world 广播维护） |

外加 world 注册表镜像：CPubScenes（this+0x128，普通/克隆场景两栏）+
CPubPlayers（this+0x124，在线/离线玩家、member↔scene 归属）。

对端拓扑：**world(0)/member(0x18..0x2F) 连入**本服 inner 受面；本服**外连**
store(4..7) 与 entry(8..0x17)（UDP 广播发现 + CheckConnect 幂等补连）。

### 1.1 game_public.ini 官方方言（全文）

```ini
[System]
Sleep=-1                  // 主循环不休眠（solo 全家同）
MainEntity=SoloPublic
MainScript=main
[Module]
fskit.dll / fscomm.dll / fssolopublic.dll
[Script] Path=..\lua\solopublic  Reload=1
[Main]
DistrictID=1
ServerID=11
PublicLogicDll=fspublogic.dll    // Callee 逻辑模块（版本门 0x300F）
[Inner]     // Addr=127.0.0.1（注释态；Port 缺省 0=系统分配，Start 后回填）
[Broadcast] // Addr=127.255.255.255 Port=15000（注释态缺省）
```

main.lua 缺省（ini 未配置时）：Inner `127.0.0.1:0`（**端口 0=侦听后
GetListenerPort 回填**）、Broadcast `127.255.255.255:15000`、Echo
`127.0.0.1:15003`、SaveMinutes 20（ctor 缺省 10）。

### 1.2 装配序列（main.lua 定案）

```
SoloPublic.UseQuickLog = true / UseAppRecord = true
SetConsoleColor("LightBlue") → 读 game_public.ini
→ LogicLoader.Load(work_path, PublicLogicDll)
→ nx_bind_script(public, "main", "echo_init")   // echo 回调全家
→ 属性注入：AppName="public_<S>" / DistrictId / ServerId / SaveMinutes /
   LogicLoader / Inner* / Broad* / Echo* / LogLauncher("game_log game_log.ini daemon")
→ AddPubSpace("domain","nx_domains") + AddPubSpace("guild","nx_guilds")
→ public:Start() → 读回 public.InnerPort（实际侦听端口）
→ console 四连（app name / echo / inner / broadcast）
→ "game public server <S> started" → SetAppState(WAPS_OPENED=4)
```

（官方脚本缺陷：`public.OuterAddr = outer_addr` 引用未定义变量→nil，
Outer 面实际不存在。）

## 2. 模块构成（FsSoloPublic.map obj 表）

| obj | 符号数 | 职责 |
|---|---|---|
| SoloPublic.obj | 416 | 实体本体：属性面 24 对 getter/setter、Start/Stop/Execute/TimerCheck、SendCustom* 族、SyncPublic、InnerSavePubData |
| SoloPublicState.obj | 36 | **8 态状态机** |
| SoloPublicWorldMsg.obj | 61 | world 面 13 处理器（注册表维护） |
| SoloPublicStoreMsg.obj | 57 | store 面 9 处理器 |
| SoloPublicMemberMsg.obj | 35 | member 面 3 处理器（同集群） |
| PubKernel.obj | 160 | IPubKernel（Callee 观测槽）+ InnerHook 发送族包装 |
| PubData.obj/PubDataImpl.obj | 139/241 | CPubData（集群同源） |
| PubDataArchive.obj | 55 | PUB4 blob + CLoadArchive |
| PubSpace.obj | 61 | CPubSpace（Add(name,saveOnly)/Remove/Clear/FindIndex） |
| PubTimer.obj | 50 | CPubTimer（集群同源，回调指针式） |
| PubPlayers.obj/PubScenes.obj | 77/26 | world 注册表镜像 |
| NameUid.obj | 39 | wchar 名 ↔ char UID 双向哈希 |
| Relation.obj | 93 | 关系表 + feed 流（NewFeed/LoadFeed/Sync*/IncGoodBad…） |
| Trade.obj | 118 | 寄售行（trade_item_t 超时链 + item_result_t） |
| RecData/PropData/PublicRecord | 272/60/57 | 记录/属性/视图（集群同源） |
| Reactor/InnerHook/EchoHook/AppRecord/LogShareMem/MemPool | — | 基建（solo 全家共享 innerhook.h） |

Fx 模块 8 导出齐备；EntCreator 注册实体类 **SoloPublic**（GetParent=""）。

## 3. SoloPublic 对象布局（ctor @0x100B8980 定案）

```
 +0x00 vptr                    +0x28 (IEntity 尾)
 +0x2c buffer[]（??_U 数组分配）
 +0x30 UseThread(bool)         +0x31 UseQuickLog    +0x32 UseAppRecord
 +0x33 ExitForceSave           +0x34 State(0..7)
 +0x38 DistrictId              +0x3C ServerId
 +0x40 SaveMinutes(缺省 10)    +0x44 SaveBatchSize(缺省 20)
 +0x48 MaxRunningTime(缺省 20)
 +0x4C TFastStr<36> InnerAddr  +0x6C int InnerPort
 +0x70 TFastStr<36> BroadAddr  +0x90 int BroadPort
 +0x94 TFastStr<36> EchoAddr   +0xB4 int EchoPort
 +0xB8 TFastStr<32> AppName    +0xD8 TFastStr<32> LogLauncher
 +0xF8 PERSISTID CalleeID      +0x100 PERSISTID LogicLoaderID
 +0x104 CReactor*              +0x108 CEchoHook*
 +0x10C CInnerHook*            +0x110 CPubKernel*
 +0x120 CPubTimer*             +0x124 CPubPlayers*
 +0x128 CPubScenes*            +0x12C CNameUid*
 +0x130 CRelation*             +0x134 CTrade*
 +0x138 广播 socket id（int）
 +0x144 TArrayPod<CPubSpace*>  +0x158 TStringPod<char,int>（空间名→下标）
 +0x168 已装载空间游标（装载链）+0x16C Closing/周期存盘空间游标
 +0x170 存盘数据游标
 +0x174 relations 装载完       +0x175 feeds 装载完
 +0x176 name-uid 表装载完(world)+0x177 trades 装载完
```

（4 个 bool 缺省全 0；main.lua 只写 UseQuickLog/UseAppRecord=1。）

## 4. comm 身份命名空间（innerhook.h 断言定案）

| comm | 含义 | 证据 |
|---|---|---|
| 0 | world（连入） | ProcessInnerMsg src==0 分支 |
| 1 | **public 自身** | Start@0x100BAD7D `push 1; SetCommIdent` |
| 4..7 | store（外连，MAX_STORE_NUM=4） | SendToStore `add ecx,4` + assert |
| 8..0x17 | **entry（外连，MAX_ENTRY_NUM=16）** | SendToEntry `add ecx,8` + assert（集群无） |
| 0x18..0x2F | member（连入，MAX_MEMBER_NUM=24） | SendToMember `add ecx,0x18` + assert |

内网帧（CInnerHook::SendMsg/ReceiveMsg，solo 全家同构）：
`[u32 N][u32 comm_id][u32 id1][u32 id2][payload]`，N=12+len（≤0x3FFFFC）；
id2=0xFFFF identify / 0xFFFE 退出；public 收帧 id1 恒 0。
路由 ProcessInnerMsg@0x100BED70：src∈0x18..0x2F→member 面 /
4..7→store 面 / 0→world 面 / 其余 trace "(SoloPublic::ProcessInnerMsg)public
message source error"。

## 5. 状态机（solopublicstate.cpp，8 态 @0x100C2090 跳表 + GetStateName 双证）

| 值 | 态 | 语义 |
|---|---|---|
| 0 | Unknown | 启动中 |
| 1 | LoadRelations | 向 store 拉关系表 |
| 2 | LoadFeeds | 拉 feed 流 |
| 3 | LoadTrades | 拉寄售行（双门：trades+nameuid） |
| 4 | LoadSpaces | 逐空间拉公共数据 |
| 5 | Opened | 开放服务（MemberRegister 门 `[+0x34]==5`；SyncPublic 门同） |
| 6 | Closing | 批量回存 |
| 7 | Closed | 终态（tick 空转仍广播） |

事件（arg1）：0=每秒 tick（TimerCheck 驱动）；1=store identify 完成；
2=全空间装载完；3=world 关闭令。**启动装载链（solo 特有五段）**：

```
Unknown ──(1: store identify)──> SendToStore(0,0,0x2D,{ServerId},4) ─┐
  0x22 StoreRelationInfo：{result@0,count@0xC,archive{str,str,int,double}×count}
     → Relation->AddRelation → +0x174=1                            │
LoadRelations ──(tick, +0x174)──> SendToStore(0,0,0x30,             │
     {ServerId; double×GetFeedLevels()},0x108)  // feed 存活期配置回写 ─┐
  0x23 StoreFeedInfo：{result,count@0xC,archive{str×3,int×3,wstr,int×2,str,double}×count}
     → Relation->LoadFeed → +0x175=1                                │
LoadFeeds ──(tick, +0x175)──> SendToStore(0,0,0x3A,{ServerId},4) ─┐
  0x28 StoreTradeInfo：{result,count,archive{str,wstr,str,wstr,str,int,int,wstr×4,int×4,double×3}×count}
     → Trade->Load → +0x177=1                                       │
  world type10 WorldRoleInfo：0x34B {count@8, 名字/UID 项}
     → NameUid Clear+Add → +0x176=1                                 │
LoadTrades ──(tick, +0x176&&+0x177)──> 首空间 SendToStore(0,0,0x24, │
     {space[0x24],table[0x24]},0x48) ─┐                             │
  0x19 StorePubInfo：{space[0x24]@0,table[0x24]@0x24,result@0x48,   │
     count@0x4C,archive{wstr×count}→space->Add(name,saveOnly=1)}    │
     → 立即请求首条 0x25{space,table,wname}；空/完 → 下一空间 0x24   │
  0x1A StorePubLoad（≥0xC0 前缀）：SetUid + LoadArchive(blob) +     │
     SetModify(0) → 下一条 0x25 / 空间完 → OnPubSpaceLoaded          │
LoadSpaces ──(event2)──> [Callee]全空间×数据 OnPubDataLoaded →      │
     SendToAllMember(0,0,archive{str space,str table,int count}×)   │
     → State=5 Opened                                               │
```

- 失败 trace 串：`(SoloPublic::PublicStateUnknown)send failed` /
  `PublicStateRelations` / `PublicStateLoadFeeds` / `PublicStateLoadTrades` /
  `PublicStateLoadSpaces`…（`PublicState%s error input %d` 兜底）。
- OnPubDataLoaded/OnFrameExecute/OnServerClose 均 >MaxRunningTime(秒)
  告警 `"... running exceed %dms"`。

## 6. store 面（solopublicstoremsg.cpp，9 处理器 @0x100C56E0 跳表定案）

| type | 处理器 | 线格式（定案） |
|---|---|---|
| 0x01 | StoreRoleOffline | 下线结果行表 → Callee OnRoleOffline（集群同源） |
| 0x19 | StorePubInfo | `{space[0x24]; table[0x24]; result@0x48; count@0x4C; archive{wstr}×}` ≥0x50 |
| 0x1A | StorePubLoad | `{result; …; space; table; wname; uid; blob}` 前缀 **0xC0**（集群 0x14 前缀 0xB1 的加宽版）→ SetUid+LoadArchive |
| 0x1B | StorePubSave ACK | `{result, space[0x24], table[0x24], wname[0x42]} ==0x88` |
| 0x1C | StorePubAdd ACK | 同上 |
| 0x1D | StorePubRemove ACK | 同上 |
| 0x22 | StoreRelationInfo | `{result@0, ?, ?, count@0xC, archive{str,str,int,double}×count}` ≥0x10 |
| 0x23 | StoreFeedInfo | `{result, count@0xC, archive{LoadFeed 12 字段}×}` ≥0x10 |
| 0x28 | StoreTradeInfo | `{result, count@0xC, archive{CTrade::Load 20 字段}×}` ≥0x10 |

**public→store 请求面**（SendToStore/SendToAllStore，type 空间 = 集群+6）：

| type | 载荷 | 触发 |
|---|---|---|
| 0x24 | `{space[0x24], table[0x24]}` 0x48 | LoadTrades 完成后每空间 PubInfo 请求 |
| 0x25 | `{space[0x24], table[0x24], wname[0x42], index}` | 每条 PubData 拉取（链式） |
| 0x26 | `{space[0x24], table[0x24], wname[0x48], blob@0x90}`（StoreArchive flag=0） | InnerSavePubData（**SendToAllStore** 广播全体 store） |
| 0x27 | 同 0x26 + `uid[0x24]@0x84` | AddPubData 即时持久化 |
| 0x2D | `{ServerId}` 4B | Unknown→关系装载请求 |
| 0x30 | `{ServerId; double×levels}` 0x108 | LoadRelations→feed 存活期配置 |
| 0x3A | `{ServerId}` 4B | LoadFeeds→动态装载请求 |

辅助：RequestRoleOffline（type1 广播全体 store {ServerId,roleID,Y..s,flag}）、
SaveLog（type0xD {wtitle[33],wname[256],wtext[256]} 0x44A，禁 `'`）、
CustomLog（type0x11 {file[32],u32,varlist}）——经 CPubKernel 对 Callee 暴露。

## 7. world 面（solopublicworldmsg.cpp，13 处理器 @0x100C7170 跳表定案）

| type | 处理器 | 动作/线格式 |
|---|---|---|
| 0 | WorldServerState | `public_res_server_state_t` 0x24B → Callee OnServerState（集群同源） |
| 1 | WorldClosePublic | → 状态机 event3 |
| 2 | WorldSceneAll | `member_forw_scene_all_t`（≥头，count + 0x90 项）逐项 AddScene |
| 3 | WorldSceneAdd | `member_forw_scene_add_t` **0x90B**：`{int scene@0,int member@4,int@8,int@0xC,char name[0x40]@0x10,char[0x40]@0x50}` |
| 4 | WorldSceneRemove | `member_forw_scene_remove_t`（int sceneId）→ RemoveScene |
| 5 | WorldPlayerAll | `member_forw_player_all_t`（count + 项）逐项 AddPlayer |
| 6 | WorldPlayerAdd | **0x9CB**：`{int@0,int@4,int@8,int@0xC,wstr name[0x48]@0x10,str[0x44]@0x58}` |
| 7 | WorldPlayerRemove | **0x50B**：`{int@0,int@4,wstr name[0x48]@8}` |
| 8 | WorldPlayerChange | **0x54B**：`{int@0,int@4,?,wstr name@0xC}` → CleanPlayerConnect |
| 9 | WorldRoleInfo | **0x34B**：`{int@0,?,count@8,项}` → NameUid Clear+Add 全量 → +0x176 |
| 10 | WorldRoleAdd | **0x6CB**：`{char name[0x24]@0, char newname[0x24]@0x48}` → NameUid Add |
| 11 | WorldRoleRemove | 0x6CB 同构 → Remove(+Add 兜底) |
| 12 | WorldRoleRecreate | 0x6CB 同构 → Remove+Add |

（member_forw_scene_all/player_all 为变长：`len >= sizeof(头)`，按 count 步进。）

**public→world**：type0 LoadPlayer `{account[0x21],wname[0x21],fromIdent}`
0x67B（LoadPlayer_mid）；type1 RequestServerState（1B）；type2 CloseDone
`{u32 1}`（Closing 收尾，SendToWorld(0,2,…)）。

## 8. member 面（solopublicmembermsg.cpp，3 处理器 cmp 链 @0x100C1E60）

集群三型同构，差异仅门与线长：

| 子类型 | 门/断言 | 行为 |
|---|---|---|
| 0 MemberRegister | len==8；**State==5** 才应答 | 回包（定向 type0）= archive `{str space,str table,int count}×空间` |
| 1 MemberData | len==**0x74**（集群 0x29B，solo 加宽） | 定向 type1 = 0x90 头 `{count,index,space[0x21],wname[0x21],key[0x21],1@0x8C}` + PUB4 blob(flag=0) |
| 2 MemberMessage | len≥8 `{msg_id,argc}` + get_vary_message | → `Callee->OnPublicMessage(kernel, ident, msgid, varlist)` |

**public→member 推送 op 0x0..0xD**：与集群 FsPublic §6 完全同构
（0x0 清单/0x1 快照/0x2 AddData/0x3 Remove/0x4 upsert 属性/0x5 RemoveAttr/
0x6 AddRecord/0x7 RemoveRecord/0x8 AddRow/0x9 RemoveRow/0xA ClearRows/
0xB SetGrid/0xC SetRow/0xD 定向消息）——SyncPublic（门 State==5）→
SendToAllMember(0,0,op,…) 广播含请求者；SendPublicMessage 定向 type0xD。

**solo 特有发送族**（SoloPublic.obj）：SendCustomByName/ByUid/ByWorld/
ByChannel/SendToScene/CommandByName —— 经 CPubPlayers 查名/CPubScenes
路由，向 member/entry/world 定向投递 varlist 消息（回包格式
`{int msg_id, int argc}` + put_vary_message）。

## 9. UDP 广播（magic 0x3001 solo 方言）

- **自宣**（BroadcastServerInfo @0x100BEC40，TimerCheck 每 1s）：
  `{u32 kind=4; u32 magic=0x3001; char ip[0x10]=InnerAddr; u32 port=InnerPort;
  u32 DistrictId; u32 ServerId}` **0x24B**（集群 public kind=4/magic0x300）。
- **收听**（OnBroadcast @0x100BBAF0，DistrictId/ServerId 双匹配门）：
  - type2 store 自宣（≥0x28，[0x24]=store comm 4..7）→ CheckStoreConnect
  - type1 **entry 自宣**（≥0x48，[0x24]<0x10=entry 序号）→ CheckEntryConnect
    （comm=8+index；集群无此面）
  - type0 world 广播（≥0x84）→ SetRoleProtectBase(double@0x7C)
    （角色保护基准；**public 不外连 world，world 连入 public**）
- 重连驱动：无定时器，靠每秒对端自宣幂等补连（CheckConnect 已连即 true）。

## 10. 生命周期（solopublic.cpp）

**Start@0x100BAA10**：mutex `%s exclusive` 单实例门（mutex_AppName）→
UseQuickLog：CLogShareMem(log\ 前缀=AppName) → UseAppRecord：CAppRecord
（SetAppId/SetAppName/SetAppAddr("addr:port"=Echo)/Start，AppRecord 信息
`<info state="%s" tick="%d" />` 每 TimerCheck 更新）→ InnerHook
SetCallback/SetAppName/SetCommIdent(**1**) → CReactor::Start →
CreateBroadcast(InnerAddr, BroadAddr, BroadPort, OnBroadcast, 0x400) →
EchoHook::Start(EchoAddr:EchoPort) → CreateListener(InnerAddr, InnerPort)
（**端口 0=分配后 GetListenerPort 回填 +0x6C**）→ CreateTimer(1s,TimerCheck)
→ LogicLoader：`FxModule_Init`/`FxPublicLogic_GetVersion`/
`FxPublicLogic_CreateCallee`/`FxPublicLogic_DeleteCallee` 四接口齐备 +
**版本 ==0x300F**（快照 Release fspublogic.dll 恰为 0x300F——solo 形态
**无集群的版本断裂**；门错 trace "logic module version error"/
"Public logic module version not match !"）→ Callee->OnPublicStart
（超 MaxRunningTime 告警）→ UseThread：CThread(WorkerProc=Execute 循环)。

**Execute@0x100BA610** = Reactor::EventLoop +（UseThread 时 Sleep(1)）；
主线程亦循环调 Execute（game_public.ini Sleep=-1 不休眠）。

**TimerCheck@0x100BEE70**（1s）：AppRecord SetAppInfo → BroadcastServerInfo
→ `PublicStateMachine(0, &elapsed秒, 4)`。

**Stop@0x100BC1B0**：Thread::Stop → DeleteBroadcast → DeleteListener →
DeleteTimer → EchoHook::Stop → Reactor::Stop → AppRecord::Stop。
ExitForceSave=true 时退出前触发全量回存（属性面）。

## 11. Opened/Closing 行为

- **Opened tick**：Callee->OnFrameExecute（计时告警）→ CPubTimer->Run →
  **周期存盘**：SaveMinutes≠0 且分钟到点 → 游标 {+0x16C 空间,+0x170 数据}
  扫描，`GetModify()` 的 CPubData → InnerSavePubData（0x26 → 全 store），
  每帧至多 SaveBatchSize 条。
- **Opened event3**（world 关闭令）：Callee->OnServerClose → State=6。
- **Closing tick**：同游标批量回存至全空间尽 → `SendToWorld(0,2,{u32 1},4)`
  → State=7 Closed。
- InnerSavePubData@0x100BDA00：TStoreArchive Seek(0x90) 写头
  `{space[0x24],table[0x24],wname[0x48]}` + CPubData::StoreArchive(**flag=0**
  全量；集群 Closing 用 flag=1 saveOnly——solo 差异点) → GetLength>0x1000000
  trace "data too big" → SendToAllStore(0, 0x26, …)。

## 12. echo 维护面（:15003）

标准 EchoHook + lua echo_init：欢迎语 **`welcome public server <S>\r`**
（注意小写 w——sologlobal 是 `welcome global server\r`）；quit→Stop+nx_quit、
end→CloseEcho、未定义→`>undefined command: %s\r`；另有十二调试回调
（getp/setp/getc/setc/method/listp/listc/listm/listg/function/debugger）。
Go 复刻取 fssologlobal/fssoloauction EchoConsole 同模式内置命令集。

## 13. 数据层

CStoreArchive/CLoadArchive/PUB4 blob/VType 枚举与集群逐字节同源
（复用 internal/fspublic，见《FsPublic全解与Go复刻.md》§9）。
CPubSpace vtable `[1]GetSpaceName [2]GetTableName [3]GetCanModify(恒1)
[4]FindPubData [5]GetPubDataCount [7]GetPubData [8]AddPubData
[9]RemovePubData` + `Add(wname, saveOnly)`/`Remove`/`Clear`/`FindIndex`。

## 14. 与集群 FsPublic 差异总表

| 维度 | FsPublic（集群） | FsSoloPublic（solo） |
|---|---|---|
| MainEntity | MainConsole（外装配 Adapter/BroadSock/AppManager） | SoloPublic（属性直挂，自含 Reactor/EchoHook/InnerHook） |
| 状态机 | 5 态 Unknown→LoadSpaces→Opened→Closing→Closed | **8 态**：+LoadRelations/LoadFeeds/LoadTrades |
| Opened 值 | 2 | **5**（Register/SyncPublic 门随迁） |
| world 面 | 3 处理器 | **13 处理器**（scene/player/role 注册表） |
| store type 空间 | 0x1E/0x1F/0x20/0x21/0x22 ↔ 0x13..0x17 | **+6**：0x24/0x25/0x26/0x27 ↔ 0x19..0x1D + 关系/动态/寄售 0x22/0x23/0x28 |
| member data 请求 | 0x29B | **0x74B** |
| store pub load 前缀 | 0xB1 | **0xC0** |
| 逻辑模块版本门 | 0x3001（快照断裂） | **0x300F**（快照匹配） |
| Inner 受面 | CInterConnect，ident 1 | CInnerHook（solo 帧），ident 1，**端口 0 自动分配** |
| entry 面 | 无 | **有**（comm 8..0x17，广播发现外连） |
| 广播 magic | 0x300 | **0x3001**（solo 方言） |
| 空间注册 | pubdata/nx_pubdata + guild/nx_guilds | **domain/nx_domains** + guild/nx_guilds |
| 存盘 blob | flag=1 saveOnly | **flag=0 全量**（GetModify 门控） |
| Relation/Trade/NameUid/Players/Scenes | 无 | **solo 独有** |

## 15. Go 复刻验收清单

- [x] ini 方言 + main.lua 装配序（InnerPort=0 分配回填/SaveMinutes/AppName）
- [x] 内网帧 + comm 命名空间（world0/self1/store4..7/entry8..0x17/member0x18..）
- [x] 8 态状态机 + 五段装载链（0x2D/0x30/0x3A/0x24/0x25 + 0x22/0x23/0x28/0x19/0x1A）
- [x] store 面 9 处理器 + 存盘面（0x26 全 store 广播/0x27）
- [x] world 面 13 处理器（scene/player/role 注册表 + 线格式断言尺寸）
- [x] member 面 3 处理器 + 推送 op 0..0xD（复用 fspublic 编码）
- [x] 广播：自宣 kind4/magic0x3001 0x24B + 收听三型（store/entry/rolebase）
- [x] echo :15003 欢迎语 `welcome public server <S>`
- [x] Relation/Trade/NameUid/Players/Scenes 内存模型 + 装载/查询 API
- [x] 测试（线格式黄金样本/状态机全链 TCP/装载链/广播/寄售搜索）
