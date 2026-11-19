# FsSoloWorld.dll 全解与 Go 复刻

> 权威逆向文档。样本：`F:\Game\Online\9yin\BGserverCJ\Debug\FsSoloWorld.dll`（1,179,648 字节，
> 同目录带 FsSoloWorld.map/PDB；源码路径 `c:\wuyf\svn\serveriii\branches\publish20130508\server\fssoloworld\`）。
> 宿主 `game_world.exe`（FsSoloLoader 体系，game_world.ini：MainEntity=SoloWorld，
> Module=fskit/fscomm/fssoloworld，脚本 `..\lua\soloworld\main.lua`）。
> 模块版本 `_FxModule_GetVersion`=1003 族；Go 复刻=`internal/fssoloworld` + `cmd/soloworld`。
> 工具：`tmp_research/soloworld/w_dis.py`（map 符号+capstone 注释反汇编 CLI）。

FsSoloWorld 是 solo 集群的**世界服**（game world 掌权者）：solo 形态下它一身兼任
集群系的 FsServer（世界调度）+ 登录入口裁决 + 角色名登记（NameUid）+ 在线玩家
状态机 + 场景分布表 + charge/gmcc/manage/monitor 五类外围代理。是 solo 家族中
仅次于 FsSoloMember 的第二大逻辑模块。

---

## 一、模块组成（.map 对象文件布局）

| obj | 符号数 | 职责 |
|---|---|---|
| SoloWorld.obj | 746 | 实体主类：属性表/Start/Stop/TimerCheck/消息分发 |
| SoloWorldMemberMsg.obj | 210 | member → world 消息面（37 case） |
| SoloWorldChargeMsg.obj | 110 | charge 应答面（21 命令 + 安全消息重发） |
| SoloWorldPlayerState.obj | 85 | 玩家状态机（14 态） |
| SoloWorldPlayer.obj | 84 | 玩家管理/存档流 |
| SoloWorldStoreMsg.obj | 80 | store → world 消息面（15 case） |
| SoloWorldInnerMsg.obj | 72 | 内网连接事件 + 同步 + manage/gmcc/extra 注入 |
| SoloWorldState.obj | 68 | 世界状态机（13 态） |
| WorldPlayers.obj | 61 | 玩家容器（0x48）+ PERSISTID 索引 |
| SoloWorldEntryMsg.obj | 56 | entry → world 消息面（6 case + 客户端子分发） |
| SoloWorldGmcc.obj | 52 | gmcc 客服协议（Voyage Century 方言） |
| SoloWorldScene.obj | 43 | 场景状态机（8 态）+ 场景消息 |
| WorldScenes.obj | 26 | 场景容器（0x178） |
| InnerHook.obj | 71 | 内网钩子：listener + 5 类主动连接 + comm 表 |
| ChargeHook/GmccHook/ManageHook/ExtraHook | 29/23/37/20 | 四外围代理 |
| EchoHook.obj | 153 | 维护控制台（行协议） |
| AppRecord.obj | 128 | 进程心跳（nx_app_record 共享内存） |
| Reactor.obj | 243 | 网络反应器（家族同构） |
| NameUid.obj | 39 | 角色名 ↔ uid 双向表 |
| MatrixCode.obj | 45 | 验证码位图（0x14f364 大对象） |
| MsgBuf/Routine/AppTool/md5/tinyxml | — | 支撑件 |

## 二、对象布局（SoloWorld this+偏移，逆向 CTOR 0x100af750）

```
+0x000  vtbl(0x101603d4)
+0x028  UseThread(bool 区域起点) / +0x2C..2E bool 旗
+0x030  ServerName  TFastStr<wchar,0x20>
+0x060  DistrictId            +0x064 ServerId
+0x068  MaxPlayers(默认8192)    +0x06C ?
+0x070  SaveMinutes(10)        +0x074 FlushMinutes(0)
+0x078  AccountRoles(1)        +0x080 LimitRoleSize(100)
+0x084  ?10                   +0x088 ?
+0x08C  WaitStoreSeconds?(60)  +0x090 60  +0x094 180  +0x098 180
+0x09C  0xE1000(921600)
+0x0A0  RoleProtectBase(f32)   +0x0A4 RoleProtectRate(f32)
+0x0A8  AutoOpen(1)  +0xAA PlayerQueue(1) +0xAB VipQueue(1) +0xAC OnlineContinue(1)
+0x0AD  OfflineContinue(0) +0xAE..B4 PasswordAnyChar/OriginalPassword/
       ReallyDeleteRole/QuitOnClosed/UseVerifyImage/DenyCreateRole/DenyEntryScene
+0x0B8  InnerAddr  char[0x20]  +0x0D8 InnerPort
+0x0DC  BroadAddr  char[0x20]  +0x0FC BroadPort
+0x100  EchoAddr   char[0x20]  +0x120 EchoPort
+0x124  AppName    char[0x20]  +0x144 LogLauncher char[0x20]
+0x164  WorldState(int)        ← ==4 即 Opened（对齐 main.lua WAPS_OPENED）
+0x168..0x178 计数器簇
+0x180/0x188 double（性能计时）  +0x190 entry 计数区 … +0x1A4/1A8/1AC charge/gmcc/manage inuse
+0x1B0  CExtraHook*[16] 槽表（monitor=0/dump=2）
+0x1F0  CReactor*   +0x1F4 CEchoHook*   +0x1F8 CInnerHook*(0x1C4)
+0x1FC  CChargeHook* +0x200 CGmccHook* +0x204 CManageHook*（SetXXXAddress 时 new）
+0x208  CWorldPlayers*(0x48) +0x20C CWorldScenes*(0x178)
+0x210  CNameUid*(0x10,桶0x7F) +0x214 CMatrixCode*(0x14F364)
+0x21C  tick 时刻   +0x220 broadcast_id   +0x224 inner listener_id   +0x228 timer_id
+0x244  TStringPod<wchar,double> 角色锁表（超时自动解锁）
+0x254  TStringPod<char,double>  账号锁表
+0x264  TStringPod<char,safe_message_t> charge 安全消息表
```

属性表（Lua 可见名→getter/setter 对，TVar 模板生成）：StateLock/UseThread/
UseQuickLog/UseAppRecord/ServerName/DistrictId/ServerId/MaxPlayers/SaveMinutes/
FlushMinutes/AccountRoles/RoleNameSizeLimit/RoleProtectBase/RoleProtectRate/
AutoOpen/PlayerQueue/VipQueue/OnlineContinue/OfflineContinue/PasswordAnyChar/
OriginalPassword/ReallyDeleteRole/QuitOnClosed/UseVerifyImage/DenyCreateRole/
DenyEntryScene/ClearPlayerBatchSize/ClearSceneBatchSize/GameVersion/
OfflineActiveSeconds/WaitChargeSeconds/WaitStoreSeconds/ChargeResendSeconds/
LimitRoleSize/LogInnerMessage/InnerAddr/InnerPort/BroadAddr/BroadPort/
EchoAddr/EchoPort/AppName/LogLauncher；只读计数：RoleCount/PlayerCount/
ActiveCount/OnlineCount/OfflineCount/LoggingCount/QueueCount/VipQueueCount/
LockedCount/State。

25 个 Lua 方法（`*_mid`）：Start Stop SetAppState SetConsoleColor WriteLine
SendEchoLine CloseEcho SetChargeAddress SetGmccAddress SetManageAddress
SetExtraAddress OpenServer CloseServer LoadCharImage LoadNameFilter
LoadPlayer AddScene AddPrototypeScene DumpPlayers DumpMemoryPlayers
DumpScenes DumpPerformance ClearPerformance DumpReactor AddAccount
ReconnectCharge debugger_get_scene_list。

## 三、Start 装配流程（0x100b1b90）

1. `CMutex::Exclusive("mutex_"+AppName)` 单实例；
2. UseQuickLog→new CLogShareMem（SetAppId/Start/SetLogPrefix/SetCreateNew）；
   GetExists 失败→`Port_Execute(LogLauncher)`（拉起 game_log）；
3. UseAppRecord→new CAppRecord（SetAppId/AppName/AppAddr=`"%s:%d"`/Start）；
4. InnerHook.SetCallback(5 回调)+SetAppName+**SetCommIdent(0)**（world comm=0）；
5. Reactor.Start；
6. `CreateBroadcast(InnerAddr, BroadAddr, BroadPort, 0x400, OnBroadcast@0x100b2960)`→id@+0x220；
7. EchoAddr 非空→EchoHook.Start(EchoAddr, EchoPort)；
8. `CreateListener(InnerAddr, InnerPort=0)`→id@+0x224，`GetListenerPort` 回填真实 InnerPort；
9. CreateTimer（1s）→TimerCheck；
10. m_pCharge/Gmcc/ManageHook 已建→各自 Start；
11. 遍历 ExtraHook*[16]→Start；
12. UseThread→new CThread（工作线程跑 EventLoop）。

主循环 `Execute(elapse)`=Reactor.EventLoop（FsLoader Sleep 节流）。

## 四、TimerCheck（每秒，0x100b5b90）

1. CheckVipQueue+CheckQueue（MaxPlayers 门限放行排队者）；
2. MatrixCode.IncMatrixPos（验证码滚动）；
3. ChargeHook.KeepConnect + SendIdentMsg("keep") + ResendChargeSafeMessage
   （+0x264 安全消息表按 ChargeResendSeconds 重发）；
4. GmccHook.KeepConnect + 心跳 SendMsg(make_msg_system_info)；
5. ExtraHook[i].KeepConnect + SendCustomMsg；
6. 角色锁表(+0x244)/账号锁表(+0x254) 扫描：`data + 0.002315(≈200s,天单位) < now`
   →"role/account %s auto unlocked"→Erase；
7. AppRecord.SetAppInfo(`<info state="%s" scenes= clones= player= online= offline= active= queue="/>`）；
8. **BroadcastWorldInfo**（自宣，见 §五）；
9. WorldStateMachine(0,0,NULL,0)（无事件驱动）。

## 五、广播发现（与家族其余成员互补成环）

### 自宣帧（BroadcastWorldInfo 0x100b55c0，0x84=132B，每秒）

```
+0x00 int 0          +0x04 int 0x3001(ident)
+0x08 char InnerAddr[0x10]   +0x18 int InnerPort(真实监听口)
+0x1C DistrictId     +0x20 ServerId
+0x24 wchar ServerName[0x20]  +0x6C? MaxPlayers(+0x68)
+0x70? this+0x170    +0x74 opened=(WorldState==4)   +0x78 在线计数和(+0x234++0x240)
```

### 接收五分支（0x100b2960，先验 ident==0x3001）

| kind | 帧长 | 判定 | 动作 |
|---|---|---|---|
| 4 | 0x24 | district/server 匹配 | CheckPublicConnect(addr,port) |
| 2 | 0x28 | 同上 | CheckStoreConnect(idx=帧+0x24,addr,port) |
| 3 | 0x28 | 同上 + idx<0x18 | CheckMemberConnect(idx,addr,port) |
| 1 | 0x48 | 同上 + idx<0x10 | CheckEntryConnect(idx,addr,port) |
| 5 | 0x24 | 同上 | CheckAuctionConnect(addr,port) |

帧公共头：`{kind,0x3001,addr char[16],port,district,server(,idx)}`。

## 六、InnerHook 与 COMM 枚举（tools/innerhook.h）

- 内网帧：`[len:u32][id1:u32][serial:u32][id2:u32][payload...]`（len=帧长-4）；
- **标识帧**（OnConnect 0x10080c1b）：`[0xC][comm_id][0][0xFFFF]` 16B——对端
  ReceiveMsg 识别 id2=0xFFFF 登记 `comm_id↔connector`，重复登记告警；
- listener 侧 OnAccept：挂空 comm_t 等对端自宣；
- COMM 枚举（由 SendTo* 反汇编定案）：

```
COMM_WORLD=0  COMM_PUBLIC=1  COMM_AUCTION=2  (3 保留)
COMM_STORE=4+idx   (MAX_STORE_NUM=4)
COMM_ENTRY=8+idx   (MAX_ENTRY_NUM=16)
COMM_MEMBER=0x18+idx (MAX_MEMBER_NUM=24)   ← 与 fssolomember 自宣 comm 同式
COMM_ID_MAX=0x30
```

- 发送原语：SendToStore(idx)/SendToEntry(idx)/SendToMember(idx)/SendToPublic/
  SendToAuction(+SendToAllMember/AllEntry/AllStore 与 *2 变体带第二段 payload)；
- 连接事件：InnerPublicConnected/InnerAuctionConnected/InnerMemberConnected/
  InnerStoreConnected/InnerEntryConnected + Disconnect 对（触发场景重同步）。

## 七、世界状态机（13 态，0x100db340 跳表）

```
Unknown ──SendToStore(load roles 请求)──▶ LoadRoles
LoadRoles ──SyncRoleInfo 同步 NameUid 完──▶ LoadScenes
LoadScenes ──CheckAllScene 全 Running──▶ CanOpen   ("game server ready"+SendOpenConnect)
CanOpen ──OpenServer()/AutoOpen──▶ Opened
Opened：每秒 CheckAllScene+CheckAllPlayer+逐玩家 PlayerStateMachine
关闭链（CloseServer 或 echo close）：
Opened▶CloseEntry(逐 entry SendCloseEntry,"entry server close all")
▶ClearPlayers(逐玩家驱动下线，批量 ClearPlayerBatchSize)
▶ClearScenes(批量 ClearSceneBatchSize)▶CloseMember▶ClosePublic▶CloseAuction▶CloseStore▶Closed
Error：异常分支。Closed+QuitOnClosed→Stop+退出。
```

`SendOpenConnect`=SendToAllEntry(id2=8 开服通知)；State 字符串即 GetState()
返回值（"Unknown"/"LoadRoles"/…，lua `self.State` 比对用）。

## 八、玩家状态机（14 态，0x100d00c0）

态名（rdata 顺序）：Unknown→WaitBreak→Logging→Queuing→Getting→Logged→
Choosing→Creating→Deleting→Reviving→Online→Offline→Saving→Leaving。

- **Unknown**：收到 entry ClientAccept 后 AddPlayer；PlayerStateUnknown 里
  CLoadArchive 解客户端 Login 载荷（account/password 双 ReadString + 校验）→
  SendToStore(load 请求)→Logging；
- **Logging**：CheckRepeatLogin（顶号）→ charge "login"（SendPlayerLogon）；
  WaitChargeSeconds 超时回退；
- **Queuing**：排队（PlayerQueue/VipQueue 属性 + CheckQueue 每秒放行）；
- **Getting**：SendPlayerRoles（角色列表下发）；
- **Logged**：等待 ChooseRole；
- **Choosing**→**Creating**：StoreMsgCreate 推进（NameUid::Add→广播
  member/public/auction→SendSystemStatInfo("create_role",account,name)）；
- **Deleting/Reviving**：对应 store delete/revive 应答推进；
- **Online**：SendToScene 转发 + IncScenePlayers；离线事件→DecScenePlayers/
  BreakClient/SendPlayerLogoff/SetOffline/ErasePlayer 组合；
- **Offline**：SaveMinutes 周期 SavePlayer 或退场 ErasePlayer；
- **Saving/Leaving**：存档中/离开中。

player_t（0x1A0=416B，AddPlayer 0x100de650）：`{conn1,conn2(=PERSISTID
entry 客户端键),member=-1@+8,name wchar 区,state@0x54,login_t double@0x58,
active_t double@0x60,…,account char[16]@0x18C}`。容器：指针数组+空闲槽
复用+TPodHashMap<PERSISTID,uint>；FindByConnect/FindByName/ByAccount/Logged。

## 九、场景体系

CWorldScenes（0x178）：normal+clone 两段；AddScene(index,member,type,?,
script,config)/RemoveScene/GetSceneList(IVarList)。lua 装配：
`world:AddScene(scene_id, member_id, script, config)`（scenes.ini 每节，
Clonable→AddPrototypeScene）。

场景状态机（0x100d8f20）：Unknown→Starting(SendSceneAdd→member)→Running→
(Cloning→Cloned 克隆场)→Deleting→Error。MemberMsgSceneState 推进；
SendSceneRemove/SendDeleteClone 对应删除。Inc/DecScenePlayers 维护计数；
BreakScenePlayers 场景故障时全员下线。

## 十、消息面分发表

### entry → world（ProcessEntryMsg，id2 0..5）
```
0 EntryMsgClientAccept   1 ClientClosed   2 ClientLost
3 ClientMessage          4 ClientIllegal  5 CloseEntry
```
ClientMessage 载荷：`{entry_client_id:i32, serial:i32, msgtype:u8, ...}`，
msgtype 0..0xC 子分发：0 GetVerify/1 Login/2 WorldInfo/3 ChooseRole/
4 CreateRole/5 DeleteRole/6 ReviveRole/…（ClientMsg* 七Handler）。
下行：SendToClient/SendToEntry（重定向 connect/BreakClient 断开）。

### store → world（ProcessStoreMsg，id2 0..0xE 有效）
```
0 RoleInfo 1 Load 2 Save 3 Block 4 Resume 5 Roles 6 Create 7 Delete
8 Revive 9 Recreate A AccountReset B AccountLogon C AccountLogoff D? E CloseStore
```
载荷均为 `store_back_*_t` 定长结构（sizeof 断言校验）。

### member → world（ProcessMemberMsg，id2 0..0x24，37 case）
```
0 CreateClone 1 DeleteClone 2 CloneScene 3 SetDownTime 4 RoleInfo
5 SceneState 6 ToScene 7 ToAllScene 8 BreakPlayer 9 SwitchScene
A NewRole B WorldInfo C SetDelay D QueryGift E GetGift F SellCard
10 UnsellCard 11 BuyCard 12 BuyItem 13 BuyItem2 14 GetPoints 15 GetLimitTime
16 GetAccountInfo 17 GetPlayTime 18 ItemInfo 19 ItemInfo2 1A ToCharge
1B ToGmcc 1C GmCustom 1D ToManage 1E ToExtra 1F CloseMember 20 LeaveGame
21 LockRole 22 UnlockRole 23 LockAccount 24 UnlockAccount
```

### public/auction → world
ProcessPublicMsg/ProcessAuctionMsg（角色增删同步等，id2 表未展开——复刻
以 NameUid 同步 + 角色增删广播对齐）。

### charge（文本行协议，CChargeHook）
- 前缀 `"$s_%d_%d_%d_%d_%s "` + `text_encode_msg(IVarList)`（internal/charge
  同源方言）；OnConnect 发 "register"，每秒 "keep"；
- 应答：text_decode_msg→InnerMessageFromCharge(首串=命令)：Reg/Unreg/Reset/
  Login/Logout/QueryGift/GetGift/SellCard/UnsellCard/BuyCard/BuyItem/BuyItem2/
  RetPoints/RetLimitTime/RetAccountInfo/RetPlayTime/ChargeNotify/ItemInfo/
  ItemInfo2/Custom/SafeCustom（后两者入安全消息表重发）。

### manage（CManageHook=**listener**，world:4002 受面）
OnAccept/ReceiveMsg(text_decode)→InnerMessageFromManage(id,cmd,IVarList)；
SendCustomMsg(connector,id,IVarList) 回包。

### gmcc（CGmccHook=主动连接，"Voyage Century" 注册）
ReceiveMsg 处理 CHA_BACK_*；SendLoginToGmcc/SendLogoutToGmcc/
SendGmccCommand/SendGmccCustom；make_msg_system_info 心跳。

### extra（CExtraHook[16] 主动连接；main.lua：0=monitor:4008、2=dump:4006）
ReceiveMsg(text_decode)→InnerMessageFromExtra(idx,cmd,IVarList)；
SendCustomMsg(IVarList)。

## 十一、EchoHook 维护控制台

EchoAddr 缺省 127.0.0.1:15001（ini 未配则不启）。行协议（\r/\n 终结）：
连接横幅 `welcome server %d\r`；命令由 main.lua echo_input 派发：
`quit`(优雅关) `fquit`(强杀) `end`(断开) `players`(五计数) `open`/`close`
(要求 State==CanOpen/Opened) `status`；调试族 on_input_getp/setp/getc/setc/
method/listp/listc/listm/listg/function/debugger（nx_property/nx_method 引擎
反射，复刻仅保留命令族框架）。

## 十二、与 Go 复刻的对照

| 官方 | Go（internal/fssoloworld） |
|---|---|
| SoloWorld 实体+属性表 | Server struct + Properties（Lua 名对齐） |
| CReactor+CreateListener/Broadcast/Timer | 复用 internal/reactor 家族装配 |
| InnerHook（comm 表+标识帧+SendTo*） | inner.go：COMM 常量/conn 表/帧编解码 |
| 广播自宣 0x84/接收五分支 | broadcast.go |
| WorldStateMachine 13 态 | state.go |
| PlayerStateMachine 14 态+CWorldPlayers | player.go |
| CWorldScenes+SceneStateMachine | scene.go |
| CNameUid | nameuid.go（map[uid]/map[name] 双向+deleted） |
| Process{Entry,Store,Member}Msg | entrymsg.go/storemsg.go/membermsg.go |
| CChargeHook | charge.go（internal/charge text_encode/decode） |
| EchoHook | echo.go（fssologlobal EchoConsole 同模式） |
| CAppRecord/CLogShareMem | 复用家族 AppRecord/QuickLog 实现 |
| Manage/Extra/Gmcc | hook 骨架 + InnerMessageFrom* 回调面 |

**不复刻**（既定边界）：CMatrixCode 验证码位图（LoadCharImage 保留空实现）、
Dump* 工具族、UseThread 工作线程（Go 天然并发）、对象内核（走 ECS+Lua，
SceneRelay 同 fssolomember 路线）。

## 十三、复刻验证要点

1. 单实例互斥名 `mutex_world_<id>`；
2. 广播自宣帧字节级对齐（ident 0x3001/kind 布局）；
3. 标识帧 16B 与 comm 登记（id2=0xFFFF）；
4. 登录链：ClientAccept→Load(account 校验)→charge login→Roles→Choose→
   Create/Online 全态迁移；
5. 关闭链 13 态顺序与 QuitOnClosed 退出；
6. echo 七命令 + players 五计数语义；
7. charge 行 `$s_..` 前缀与 register/keep 心跳。
