# FsSoloMember.dll + game_member.exe 全解与 Go 复刻（单机形态场景服务器）

> 对象：`BGserverCJ\Debug\FsSoloMember.dll`（2 793 472 B，**solo 家族最大模块**，
> 2013-07-05 构建，Debug 全符号 .map（13 689 公符号）+ pdb 9.6MB，
> 源码路径 `c:\wuyf\svn\serveriii\branches\publish20130508\server\fssolomember\`）
> 与宿主 `game_member.exe`（589 824 B，与 game_lister/entry/auction.exe 同构建）。
> 方法：tmp_research/solomember/sm_dis.py（capstone + map 符号 + ILT thunk
> 解析 + .rdata 字符串注释；两级跳表解码器）。
> 日期：2026-11-19　分支：`official`
> 复刻落点：`internal/fssolomember` + `cmd/solomember` + cmd/loader 角色接线。
> 姊妹篇：FsMember全解与Go复刻.md（大写系集群形态——本文 §十为其 solo 增量定案）。

---

## 一、模块定位

FsSoloMember.dll 是**小写系（九阴 solo 体系）的场景服务器实体类模块**：
持有全部 CGameScene/CGamePlayer 游戏对象、驱动场景心跳/扫描、承载客户端
消息流（经 entry 转发）、对接 world（受面连入）/public/store/auction（发面
外连）。与集群形态（FsMember.dll，挂 FsComm Adapter/BroadSock 实体）不同，
solo 形态**自含网络栈**：DLL 内静态链接 CReactor(243 符号)/CEchoHook(153)/
CInnerHook(71)，并内嵌整套游戏对象内核（Kernel*+GameObject*+Game*+Scene*
≈60 obj、Lua 5.0、tinyxml）——这就是它 2.79MB（FsMember 仅 1.38MB）的原因。

- 装配：`game_member.exe` → `[Module] fskit.dll / fscomm.dll / fscollide.dll /
  fssolomember.dll`，`MainEntity=SoloMember`，脚本 `..\lua\solomember`，
  逻辑模块 `[Logic] Module0=fsgame.dll`（玩法层另载）。
- 导出 8 个标准 FxModule（GetType="FxModule"）；实体注册 SoloMemberEntCreator
  （GetName=**"SoloMember"**，GetParent="IEntity"）。

### 模块清单（.map obj 归组 → Go 落点）

| obj 组（符号数） | 官方类 | 职责 | Go |
|---|---|---|---|
| SoloMember (635) | `SoloMember : IEntity` | 服务本体：42 属性 + 28 方法 + Start/Stop/Execute/TimerCheck/TimerBeat/TimerScan | `server.go Server` |
| SoloMemberWorldMsg (234) | ProcessWorldMsg+Command×21 | world 集群面（§六） | `world.go`（复用 fsmember 注册表） |
| SoloMemberPublicMsg (231) | ProcessPublicMsg+Public×15 | public 数据镜像面 | `public.go`（复用 fsmember.PubSpaces） |
| SoloMemberStoreMsg (137) | ProcessStoreMsg+Publish×8 | store 存档/管理面 | `store.go` |
| SoloMemberAuctionMsg (…) | ProcessAuctionMsg | 拍卖行转发（0xD/0xE 两型） | `server.go processAuctionMsg` |
| SoloMemberScene (51+) / SoloMemberState | 场景查询族 / 状态机五处理器 | §五/§七 | `state.go` + `scenes.go` |
| InnerHook(71)/EchoHook(153)/Reactor(243) | CInnerHook/CEchoHook/CReactor | 内网 TCP 受/发面 / 维护台 / IOCP | `inner.go`（reactor 复用既有） |
| Kernel*×12（~1500）/ GameObject*×8 / GameScene±7 / GamePlayer/Npc/Item/Motion/Visible/WeakBox / SceneBeat/Grid/Factory / ClassSet / Resource / EventCmdMenger / MemberPlayers | **游戏对象内核全家桶** | 场景对象运行时（见 §九 survey） | 不复刻（ECS+Lua 路线既定） |
| RecData/KernelRecord/ObjectRecord/PublicRecord/AppRecord/ManageInfo | 行存储/存档/进程记录 | 持久化与运维件 | AppRecord/LogShareMem 不复刻 |
| Lua 全家（lapi/lcode/llex/lbaselib/lauxlib/ldblib）+LuaUtils+KernelLua | 内嵌 Lua 5.0 | 脚本宿主 | Go 侧原生 |
| XmlFile/IniFile/ReadIni/tinyxml | 配置解析 | scenes_member/location 等复用 |

## 二、SoloMember 对象布局与属性面（ctor/getter 实证）

```
+0x000 vtable（IEntity 派生）        +0x118 TFastStr<32> AppName
+0x030 int  BindCore = -1            +0x138 TFastStr<32> LogLauncher
+0x034 int  State（状态机 §七）      +0x160 CReactor*（内嵌构造）
+0x038 int  DistrictId               +0x164 CEchoHook（内嵌）
+0x03C int  ServerId                 +0x168 CInnerHook（内嵌）
+0x040 int  MemberId                 +0x16C CLogShareMem*（UseQuickLog）
+0x044 int  MaxRunningTime = 100     +0x170 CAppRecord*（UseAppRecord）
+0x048 int  MaxLetterNum = 20        +0x19C TArrayPod<CGameScene*> 场景表
+0x04C int  BeatTime = 100 (ms)      +0x1EC int 广播口 id = -1
+0x050 int  ScanTime = 20 (ms)       +0x1F0 int 内网 listener id = -1
+0x054 int  FastScanTime = 200       +0x1F4 int 1s 定时器 id = -1
+0x058 int  SlowScanTime = 1000      +0x1F8 int beat 定时器 id = -1
+0x05C int  SceneGridSize = 12       +0x1FC int scan 定时器 id = -1
+0x060 byte DebugHalt = 1            +0x310 帧耗时串（AppRecord 呈现）
+0x061 byte AppearAtOnce = 0         +0x33x TArrayPod<CPubSpace*> 公共数据镜像
+0x062 byte HideEmptyProperty = 0    +0x35x TArrayPod<CSceneChunks*> 存档块表
+0x063 byte ScriptReload = 1         +0x434 byte 公共数据就绪闩（Loading 门）
+0x064 byte MergeMoving = 1          +0x43C byte 角色档案就绪闩（Loading 门）
+0x065 byte MergeProperty = 0        +0x440 CManageInfo +0x444 byte 管理配置闩
+0x066 byte MergeObjects = 0         +0x45x TArrayPod<module_t> 逻辑模块表
+0x067 byte LogFrameTime = 0
+0x068 byte UseLuaScript = 1
+0x0AC TFastStr<32> InnerAddr        +0x0CC int InnerPort（0=临时口回填）
+0x0D0 TFastStr<32> BroadAddr        +0x0F0 int BroadPort
+0x0F4 TFastStr<32> EchoAddr         +0x114 int EchoPort
```

**42 属性**（prop_struct 静态注册，全文）：AppName/AppearAtOnce/BeatTime/
BindCore/BroadAddr/BroadPort/CollideManager/CollideRootPath/DebugHalt/
DistrictId/EchoAddr/EchoPort/EnableMemPool/EnableObjectPool/FastScanTime/
HideEmptyProperty/InnerAddr/InnerPort/LogFrameTime/LogLauncher/LogicLoader/
MaxLetterNum/MaxRunningTime/MemberId/MergeMoving/MergeObjects/MergeProperty/
ScanTime/SceneGridSize/ScriptPath/ScriptReload/ScriptUseMemPool/ServerId/
SlowScanTime/State(只读)/UseAppRecord/UseLuaScript/UseQuickLog/UseThread。
**28 方法**：Start/Stop/SetAppState/SetConsoleColor/WriteLine/SendEchoLine/
CloseEcho/LoadNameFilter/LoadLocations/DumpScene/DumpScenes/DumpPlayers/
DumpPerformance/DumpReactor/DumpLogicClass/DumpObjectPool/ClearPerformance/
GetChildMax/GetRecordMax/GetPubRecordMax/GetMemPoolSize/GetMemFreeSize +
debugger_get_{scene_list,scene_detail,object_detail,pubspace_list,
pubdata_list,pubdata_detail} 六个 echo 调试器扩展。

> 大写系 GameMember 有而 SoloMember 无：PlayerSingleBuffer/SightOptimize/
> TaskReload/TaskPath/Console/EchoSock/BroadSock/AppManager/Adapter（集群
> 件，solo 由内嵌 Hook 承担）；SoloMember 独有：BindCore/UseThread/
> UseQuickLog/UseAppRecord/BeatTime/ScanTime/FastScanTime/SlowScanTime/
> SceneGridSize/Merge*/LogFrameTime/UseLuaScript/EnableMemPool/
> EnableObjectPool/ScriptUseMemPool + 六个 debugger_* 视图。
> main.lua 引用了不存在的 `member.OuterAddr/OuterPort`（solomember/main.lua
> 原文即有此残留，nil 赋值被属性框架吞掉）——**官方脚本 bug，非隐藏面**。

## 三、Start/Stop/Execute 与定时器（0x1028F0B0 逐 call 实证）

```
Start():
  BindCore ∈ [0,0x40) → Port_SetProcessAffinity
  → CMutex::Exclusive("mutex_" + AppName)   // 单实例；失败 printf + exit(0)
  → UseQuickLog: new CLogShareMem → SetAppId(0x18+MemberId)、SetLogPrefix
    (AppName)、SetCreateNew(1)、core->SetTraceFunc(TraceFunc)；日志服不在
    （GetExists()==false）且 LogLauncher 非空 → Port_Execute(LogLauncher)
    （拉起 game_log）——失败 "(SoloMember::Start)start quick log failed"
  → UseAppRecord: new CAppRecord → SetAppId(0x18+MemberId)、SetAppName
    (AppName)、SetAppAddr("echo:port")、Start
  → CInnerHook::SetCallback(4 回调)/SetAppName/SetCommIdent(**0x18+MemberId**)
  → CReactor::Start
  → CreateBroadcast(InnerAddr, BroadAddr, BroadPort, buf=0x400, cb=on_broadcast)
  → EchoAddr 非空 → CEchoHook::Start(EchoAddr, EchoPort)
  → CreateListener(InnerAddr, InnerPort, pendings=10, in=out=0x400000,
    keep=0x4000000, on_accept/on_close/on_receive)   // InnerPort=0 →
    GetListenerPort 回填实际口
  → CreateTimer(1.0s → TimerCheck)
  → CreateTimer(BeatTime ms → TimerBeat) + CreateTimer(ScanTime ms →
    TimerScan)    // 失败 "(SoloMember::Start)create scene timer failed"
  → InitEventCallbackNames() → CResource::SetPath
  → 逻辑模块链：GetProcAddress("FxModule_Init"/"FxGameLogic_GetVersion"/
    "FxGameLogic_GetCreator"/"FxGameLogic_GetModuleCreator") → 版本门
    （"logic module version not match !"）→ module_t 入表 → 创者入
    CRawCreatorSet/CLogicCreatorSet(ProcessInherit) → CClassSet::AddCreator
  → 内建类核验：CGamePlayer/NPC/Item/Helper/Scene/WeakBox::AddMember
  → 每逻辑创者 RunClassCallback(OnCreateClass，异常保护
    "(SoloMember::Start)exception OnCreateClass %s")
  → CClassSet::GenerateIdentity/CalculateNeedMemory + CKernel::GetBuffer +
    InitTableMessage → 模块 before-launch 回调
  → UseThread → CThread(reactor EventLoop) + Start → "SoloMember pointer is 0x%p"

Stop(): CThread::Stop → DeleteBroadcast → DeleteListener → DeleteTimer×3
  → CEchoHook::Stop → CReactor::Stop → CAppRecord::Stop+del →
  CLogShareMem::Stop+del → 模块 Shut+del → modules.clear()
  （官方 Stop 文案沿用 "(SoloMember::Start)logic module shut failed"）

Execute(elapsed) ≡ CReactor::EventLoop()   // UseThread=false 主循环泵
```

**TimerCheck（1s，@0x10295600）**：AppRecord 每 6s
`<info state="%s" frame="%s" tick="%d" />`（GetTickCount）→
BroadcastMemberInfo()（每拍！）→ MemberStateMachine(0,0,0)。
**TimerBeat（BeatTime ms）**：16 桶帧耗时统计（debug 行
`beat:%.3f,…×16`）→ 遍历场景 CGameScene::TimerBeat。
**TimerScan（ScanTime ms）**：遍历场景 CGameScene::TimerScan（内部按
FastScanTime/SlowScanTime 分档）。

## 四、UDP 15000 服务发现（双向实证）

**自宣 BroadcastMemberInfo（@0x10295130，每 TimerCheck 一拍）**：

```
0x28B { +0x00 u32 kind=3  +0x04 u32 magic=0x3001
        +0x08 char ip[16]=InnerAddr  +0x18 u32 port=InnerPort(实际)
        +0x1C u32 district  +0x20 u32 server  +0x24 u32 member }
```

**收侧 on_broadcast（@0x10290E70）**：len≥8 且 magic==0x3001 才解析，
district(+0x1C)/server(+0x20) 双门后外连（全部**主动外连**）：

| kind | 包长 | 目标 | 动作 |
|---|---|---|---|
| 4 | 0x24 | public | CheckPublicConnect(ip@+8, port@+0x18) |
| 2 | 0x28 | store | CheckStoreConnect(comm@+0x24, ip, port)（comm∈4..7） |
| 1 | 0x48 | entry | CheckEntryConnect(idx@+0x24 (<0x10 断言), ip, port)（comm=8+idx） |
| 5 | 0x24 | auction | CheckAuctionConnect(ip, port)（comm=3？实为 comm 2 槽，见 §五命名空间注） |

kind=0（world 0x84B 自宣）**不处理**——world 是唯一反向角色：收 member
kind=3 自宣后**连入 member 的内网 listener**（受面 OnAccept→CInnerHook::
OnAccept 登记，world 以 identify(comm=0) 自报）。即 solo 拓扑：member 中心
受面 + public/store/entry/auction 发面外连。

## 五、内网 CInnerHook（comm 命名空间与 on_connect 注册）

帧格式/identify(0xFFFF)/exit(0xFFFE) 与 FsSoloAuction §三完全同源
（同一 tools\innerhook.h）。member 专属：

- 自身 comm = **0x18 + MemberId**（Start 实证；member 空间 0x18..0x2F）。
- listener 参数：pendings=**10**、in=out=**4MB**、keep=0x4000000——
  比拍卖（0x400/0x4000）大三个数量级（客户端流量面）。
- **on_connect（@0x10291320）**：外连成功且对端 comm==1（public）→
  SendToPublic(id1=0, id2=0, {DistrictId,ServerId}, 8) **member 注册**；
  失败 trace "(on_inner_connect)send member register failed"。
- SendTo* 族：SendToWorld/SendToWorld2、SendToPublic、SendToStore(idx)/
  SendToAllStore、SendToEntry(idx)/SendToEntry2、SendToAllEntry/
  SendToAllEntry2、SendToAuction、SendToChannel；GetStoreConnected(idx)。

### ProcessInnerMsg 路由（@0x10295270，源 comm 分派）

| src comm | 路由 |
|---|---|
| 0 | ProcessWorldMsg(id1,id2,data,size) |
| 1 | ProcessPublicMsg |
| 2 | ProcessAuctionMsg（id2=0xD→AuctionMessage 转发场景；0xE→AuctionCommand；其余断言） |
| 8..0x17 | entry：**全部** → SendToScene(id1, **0x11**=CommandClientMessage, data, size) |
| 4..7 | store：id2 0xB/0xC/0xD/0xE → SendToScene(id1, **0x40/0x41/0x42/0x43**)（存档块回执入场景）；0xF..0x12 → ProcessStoreMsg；0x13..0x18 → SendToScene(id1, **0x21..0x26**)（信件/礼物/点卡面）；其余 id2 → SendToScene(id1, id2 原样透传) |
| 其它 | 断言失败（跳表 default） |

> 与大写系的两处关键差异：集群 entry 消息有 set_client_online 特殊路
> （CPlayerBuffers），solo 无——entry 流一律 SendToScene 0x11；集群 member
> →entry 用 entry_forw 7 型，solo 用 **id2=6（send_to_client）** 且
> SendToClient/2 显式构造 {serial,type} 8B 头（§八）。

## 六、四张消息表（跳表/断言串逐型钉死）

**ProcessWorldMsg（@0x102A2E50，两级跳表 0x48 槽）**：

| id2 | 处理器 | | id2 | 处理器 |
|---|---|---|---|---|
| 0x00 | CommandCreateScene | | 0x0B | CommandRoleInfo（→+0x43C 闩） |
| 0x01 | CommandCreateClone | | 0x0C | CommandRoleAdd |
| 0x02 | CommandDeleteClone | | 0x0D | CommandRoleRemove |
| 0x03 | CommandCheckScene | | 0x0E | CommandRoleRecreate |
| 0x04 | CommandSceneAll | | 0x33 | CommandItemInfo |
| 0x05 | CommandSceneAdd | | 0x34 | CommandItemInfo2 |
| 0x06 | CommandSceneRemove | | 0x3E | CommandCloseMember（→事件 1） |
| 0x07 | CommandPlayerAll | | 0x3F | CommandDumpPerformance |
| 0x08 | CommandPlayerAdd | | 0x44 | CommandClearPerformance |
| 0x09 | CommandPlayerRemove | | 0x47 | CommandMemoryRole |
| 0x0A | CommandPlayerChange | | 其余 | **默认：遍历场景表 SendToScene(scene, id2 原样)** |

**ProcessPublicMsg（@0x1029CE90，0..0xE 顺序）**：PublicSpaces（→+0x434
闩）/PublicData（同闩）/PublicAdd/PublicRemove/PublicAddAttr/
PublicRemoveAttr/PublicAddRec/PublicRemoveRec/PublicRecAddRow/
PublicRecRemoveRow/PublicRecClear/PublicRecChange/PublicRecSetRow/
PublicMessage/PublicCommand（15 型，比集群多 PublicCommand）。
存储：FindPubSpace/GetPubSpaceCount/GetPubSpaceList/GetPubSpace +
TArrayPod<CPubSpace*>（CPubSpace+PubDataImpl，space→data→{attr,rec}）。

**ProcessStoreMsg（@0x102A0200，sub 0xF 跳表 8 型）**：0xF PublishChunkInfo
（"no scene chunks/scene id error"）/0x10 PublishLoadChunk（"chunks amount
error"）/0x11 PublishSaveChunk（"save failed"）/0x12 PublishDeleteChunk/
0x13 PublishManageInfo（断言 `len >= sizeof(store_back_manage_info_t)`，
CLoadArchive{String,WideStr,Int,WideStr,Int,Int} → CManageInfo@+0x440，
失败清 +0x444 闩）/0x14 PublishManageBase/0x15 PublishManageUpdate/
0x16 PublishManageRemove。

**场景命令命名空间**（CGameScene::ProcessMsg，GameSceneMember.obj 断言串
45+ 型）：0x11 CommandClientMessage（entry 流入口）、0x40..0x43 存档回执、
0x21..0x26 信件/礼物/点卡、CloneScene/CloneReset/AddPlayer/SwitchPlayer/
RemovePlayer/SendLetter…/TimerSaving/Disconnect/Continue/QueryGift/GetGift/
SellCard/UnsellCard/BuyCard/BuyItem/GetPoints/GetLimitTime/GetAccountInfo/
GetPlayTime/ChargeNotify/ItemInfo/FromCharge/Debug/GmCustom/SysStatInfo/
FromManage/FromExtra/FromPublic/PublicCommand/PublicComplete/ServerClose/
EditRole/RecreateRole/QueryAccount/QueryRole/WorldInfo/Warning/
SceneCommand/NamedCommand/PlayerCommand/GroupCommand + 客户端消息五型
（MessageSelect/Ready/RequestMove/Speech/Custom）。**对象内核域，不复刻**
（§九），Go 侧以 SceneRelay 钩子承接。

## 七、状态机（SoloMemberState.obj，state@+0x34 五态）

```
                事件1(CloseMember/世界关服)          场景全空
  0 Unknown ────────────────────────► 3 Closing ───────────► 4 Closed
    │ store0 已连（每 tick 查）          │ 每拍扫场景表：
    │ SendToStore(0, 0, 0x29,           │   GetScenePlayer()->GetCount()>0
    │   {ServerId},4)                    │   → "scene has player" + 场景名（阻塞）
    ▼ "start load manage config data..." ▼ SendToWorld(0,0x1F,{1,MemberId},8)
  1 Loading ──三闩全置──► 2 Opened    （"send close member failed" 重试）
```

- **Loading 三闩**（MemberStateLoading @0x1029F020 逐字节实证）：
  +0x434 公共数据（PublicSpaces/PublicData 置）∧ +0x43C 角色档案
  （CommandRoleInfo 置）∧ +0x444 管理配置（PublishManageInfo 置）
  → Opened。CommandCreateScene 前置检查 +0x434/+0x43C（未就绪拒建场景）。
- GetState（@0x1029EB50）跳表返回串："Unknown"/"Loading"/"Opened"/
  "Closing"/"Closed"（供 echo/AppRecord 呈现）。
- 与大写系差异：集群 4 态（Unknown/Opened/Closing/Closed），solo **插入
  Loading 引导态**（store manage 配置 + public 数据 + 角色档案三源就绪门）。

## 八、entry 客户端流（下行）

```
SendToClient(serial, type, data, size) @0x10294430:
  头 {serial u32, type u32}(8B) + data
  → SendToEntry2(entryIdx=(serial>>16)&0xFFFF, id1=0, id2=6 /*send_to_client*/,
                 hdr+data)     // "(SoloMember::SendToClient)send to entry failed"
SendToClient2 同构（多一个 flag 域）
SendClientBroadcast(data) → SendToAllEntry(id1=0, id2=8 /*broadcast*/)
SendToChannel(channel, data) → SendToAllEntry2(id2=0xD /*send_to_channel*/,
  {channel}+data)
```

serial 的 `(entryIdx<<16)|序号` 编码与大写系一致；id2 枚举与
fssoloentry/process.go 的 SoloForw* 15 型完全对上（member 是这些帧的
**生产方**，entry 是消费方）。entry→member 上行全部走 §五的 0x11 场景中继。

## 九、游戏对象内核 survey（不复刻，ECS/Lua 路线承接）

- **CClassSet**（117 符号）：类注册表 + GenerateIdentity（类标识哈希）+
  CalculateNeedMemory + InitTableMessage；Create(SoloMember*, class) →
  CGameObject。EnableObjectPool 开对象池。
- **CKernel**（198 符号）：对象内核（Create/Place/Switch/Store/Buffer/
  Charge/Gmcc/Lua 桥）；GetBuffer/GetBufferSize（CalculateNeedMemory 分配）。
- **CGameObject 族**：GameObject/Property/Record/Data/Archive/Container/
  Event/Interface + GameScene(130)/GamePlayer(152)/GameNpc/GameItem/
  GameHelper/GameMotion(68)/GameMoveable/GameVisible/GameWeakBox +
  SceneBeat/SceneGrid/SceneFactory/SceneConfig——AOI 网格（SceneGridSize=12）、
  视野合并（MergeMoving/MergeProperty/MergeObjects）、八种运动模式
  （ExecuteMotion/Jump/Fly/Swim/Drift/Climb/Slide/Sink）。
- **持久化**：CSceneChunks（场景分片存档，store 0xB..0xE 回执驱动）+
  CRecData/CPropData（行/值存储）+ CLoadArchive/TStoreArchive（PUB4 同族）。
- **事件**：EventCmdMnger(408)/EventCallback/InitEventCallbackNames。
- **成员变量桥**：CMemberVar（int/int64/float/double/string/widestr/
  object 七型 + 各 Game*::AddMember/SetMember/QueryMember）——Lua 面
  `nx_property`/`nx_custom` 的 C++ 底座。

## 十、与大写系 FsMember.dll 差异总表

| 维度 | FsSoloMember（本文） | FsMember（集群） |
|---|---|---|
| 网络栈 | **内嵌 CReactor+CInnerHook+CEchoHook** | 挂 FsComm Adapter/BroadSock/EchoSock 实体 |
| 对象内核 | **DLL 内嵌全家桶（2.2MB .text）** | 分离（FsGame.dll 装载后注册） |
| 受面 | CreateListener(10, 4MB/4MB) 直连 world | Adapter 内网侦听 16300+M |
| 发面 | Check*Connect 广播发现外连 public/store/entry/auction | CInterConnect 显式配置池 |
| 自身 comm | **0x18+MemberId**（identify 自报） | Adapter 注册帧（dst ident 路由） |
| 状态机 | **5 态 + Loading 三闩引导** | 4 态 |
| 启动引导 | store 0x29 load-manage → Loading → Opened | world CommandCloseMember 直驱 |
| entry 流 | 一律 SendToScene 0x11；下行 id2=6/8/0xD | PutEntryMessage 双路 + entry_forw 7 型 |
| store 面 | id2 0xB..0x18 重映射 0x40..0x43/0x21..0x26 入场景 | 0x9..0xC Publish* 直处理 |
| world 表 | 21 型 + **默认全场景透传**（0x48 槽跳表） | 17 型，无默认透传面 |
| 拍卖行 | **ProcessAuctionMsg 0xD/0xE**（solo 特有） | 无 |
| 单实例 | mutex_%s + exit(0) | AppManager 注册表互斥 |
| 属性面 | 42 属性/28 方法（debugger_* 六视图） | 23 属性/6 方法 |

## 十一、Go 复刻（internal/fssolomember + cmd/solomember）

复刻语义不复刻 IOCP（internal/reactor 既有；EchoHook/AppRecord/LogShareMem
进程监管件按既定原则不复刻，日志呈现等价）：

| 官方件 | Go | 备注 |
|---|---|---|
| game_member.ini 方言 | `Config` + `ParseGameMemberINI`（Main/Inner/Broadcast/Echo/Logic；缺省公式 16300/15300+M、广播 15000） | 集群方言共用解析骨架 |
| SoloMember 实体 + 布局 | `Server`（DistrictId/ServerId/MemberId/BeatTime/ScanTime/… ctor 缺省全对齐 §二） | |
| Start/Stop 序 | `Start/Stop`（单实例 mutex、listener、双 UDP、三定时器等价装配；逻辑模块面=SceneRelay 钩子） | |
| TimerCheck 1s | `Tick()`（AppRecord 行 6s 节流 + 自宣 + 状态机拍） | ManualTick 测试钩 |
| BroadcastMemberInfo | `BuildMemberAnnounce`（0x28B kind=3/0x3001 逐字节） | |
| on_broadcast 四分支 | `OnBroadcast`（len/magic/分区/服门 + Check*Connect 外连） | |
| CInnerHook | `InnerLink`（帧累积/identify/exit + SendToWorld/Public/Store/Entry×2/AllEntry/Auction/Channel + on_connect public 注册） | selfComm=0x18+M |
| ProcessInnerMsg 路由 | `processInnerMsg`（0/1/2/8..0x17/4..7 全分支 + store id2 重映射表） | |
| world 21 型 | `world.go`（场景/玩家/角色注册表复用 internal/fsmember；默认全场景透传） | |
| public 15 型 | `public.go`（fsmember.PubSpaces 镜像 + Loading 闩） | |
| store 8 型 + 0x29 引导 | `store.go`（chunks 表 + ManageInfo 闩 + CLoadArchive 序） | |
| auction 0xD/0xE | `processAuctionMsg`（场景中继钩子） | |
| 状态机五态 | `state.go StateMachine`（0x29 引导、三闩、Closing 空场景门、0x1F 关播） | |
| entry 下行 | `SendToClient/2/Broadcast/Channel`（serial→entryIdx、id2=6/8/0xD） | |
| CGameScene 命令面 | `SceneRelay` 接口（Go 侧场景宿主承接 0x11/0x21../0x40.. 透传） | 对象内核不复刻 |
| 进程载体 | `cmd/solomember`（ini 释放式 + 旗标） | cmd/loader fssolomember.dll 角色同源接线 |

**测试**（fssolomember_test.go）：帧编解码往返与半包、自宣/发现四分支门、
路由全分支（entry 0x11/store 重映射/默认透传）、状态机全流转（0x29→Loading
→三闩→Opened→CloseMember→空场景门→0x1F→Closed）、SendToClient serial
编码、Lua ini 方言解析（快照 game_member.ini 全键）、真 TCP 回环
（world 连入 identify + entry 外连 + public 注册帧）。

## 十二、证据索引（FsSoloMember.dll 内 VA）

| 内容 | VA |
|---|---|
| SoloMember::Start / Stop / Execute | 0x1028F0B0 / 0x10291750 / 0x1028E7C0 |
| TimerCheck / TimerBeat / TimerScan | 0x10295600 / 0x10295740 / 0x102959F0 |
| BroadcastMemberInfo / on_broadcast | 0x10295130 / 0x10290E70 |
| 静态回调：listener_on_accept / on_connect / on_close / on_receive / timer_1s / timer_beat / timer_scan / 线程泵 | 0x102912B0 / 0x10291320 / 0x10291440 / 0x10291520 / 0x10291590 / 0x102915E0 / 0x10291640 / 0x102916A0 |
| ProcessInnerMsg / ProcessWorldMsg / ProcessPublicMsg / ProcessStoreMsg / ProcessAuctionMsg | 0x10295270 / 0x102A2E50 / 0x1029CE90 / 0x102A0200 / 0x10299620 |
| world 两级跳表（0x48 槽索引表/跳表） | 0x102A3250 / 0x102A31F8 |
| store id2 跳表（ProcessInnerMsg 内） | 0x10295508 |
| MemberStateMachine / Unknown / Loading / Opened / Closing / Closed | 0x1029EC40 / 0x1029ED90 / 0x1029F020 / 0x1029F0F0 / 0x1029F180 / 0x1029F320 |
| GetState（五名跳表） | 0x1029EB50 |
| SendToClient / SendToClient2 / SendClientBroadcast / SendToChannel / SendToScene | 0x10294430 / 0x10294520 / 0x10294870 / 0x10294900 / 0x10295080 |
| ctor（缺省值）/ CTOR 内建类核验 | 0x1028C100 段 / 0x1028FE00 段 |
| Loading 三闩写点：PublicSpaces/PublicData/CommandRoleInfo/PublishManageInfo | 0x10299DB1 段 / 0x1029A1FA 段 / 0x102A1B4D 段 / 0x1029FB66 段 |
| "mutex_%s" 单实例 / SetCommIdent(0x18+M) | 0x1028F129 / 0x1028F45E 段 |
| 源码断言：solomember.cpp / solomemberstate.cpp / solomemberstoremsg.cpp / solomemberauctionmsg.cpp / tools\innerhook.h | 0x10361248 / 0x10363488 / 0x10363768 / 0x10362098 / 0x10350830 |
