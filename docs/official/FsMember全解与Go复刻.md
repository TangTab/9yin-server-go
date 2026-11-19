# FsMember.dll 全解与 Go 复刻

> 权威快照：`E:\Workspace\project\game\九阴真经ol\BGserverCJ\Debug\FsMember.dll`
>（1,380,352 B，2014-03-22）+ `FsMember.pdb`（3.67MB，12,975 pub 符号）；
> 进程壳 `game_member.exe`（589,824 B，md5 195e1ce1…，与 game_entry/lister/
> world.exe 同一构建改名部署）。同源单机形态 `FsSoloMember.dll`（2.79MB）
> 带 `.map`（全符号），类结构以 map 定案、消息表以 FsMember.dll 反汇编
>（capstone + PDB 符号交叉注释，工具 `tmp_research/fsmember/fm_dis.py`）
> 逐函数对齐。宿主脚本 `BGserverCJ\Lua\GameMember\main.lua` + `echo.lua`。
> Go 复刻：`internal/fsmember` + `cmd/member`（§11 为验收清单）。

## 1. 进程定位与部署形态

Member 是集群的**场景服务器**（scene host）：持有全部 CGameScene/CGamePlayer
游戏对象、驱动心跳与 AOI、对接 entry（客户端流转发）与 world/public/
store（集群控制面）。`[Main] MemberID` 即场景服编号（一个区服多台 member）。

| 启动方式 | ini | 形态 |
|---|---|---|
| `GameMember.bat` → `FsLoader.exe GameMember.ini` | 大写系集群 | FsKit+FsComm+**FsMember**+FsCollide，MainEntity=MainConsole，脚本 `..\Lua\GameMember`，逻辑模块 FsGame.dll |
| `game_member.exe`（裸跑，`<exe基名>.ini` 约定） | `game_member.ini` 小写系 | fskit+fscomm+fscollide+**fssolomember**，MainEntity=SoloMember，脚本 `..\lua\solomember` |

装载器五步序列与 game_lister/entry/world.exe 完全同构（见
《FsLoader全解与Go复刻.md》§3；md5 定案四 exe 同构建）。
**同名多角色的 Go 复刻**：`build/game_loader.exe` 复制为
`game_member.exe` 裸跑（`<exe基名>.ini` 发现序）或 argv 传
`GameMember.ini`，即按 ini [Module] 表装载不同模块集——
fskit-only=纯装载器 / +fssolomember=SoloMember 主实体+solo 形态 /
+FsMember=MainConsole+集群形态；未注册模块报官方文案
"(CDllManager::Load)module interface not found"。Go 侧模块=
编译期注册表适配器（官方 CRT 静态构造注册 creator 的进程内等价，
Go plugin 不支持 Windows 且原生 C++ interop 不取）。

### 1.1 GameMember.ini 官方方言（集群形态全文）

```ini
[System]
Sleep=1                    // 主循环休眠毫秒（<0 不休眠）
MainEntity=MainConsole
MainScript=main
[Module]
FsKit.dll / FsComm.dll / FsMember.dll / FsCollide.dll
[Script] Path=..\Lua\GameMember  Reload=1
[Resource] Path=..\Res
[Main]
DistrictID=1               // 游戏分区编号
ServerID=11                // 游戏服务器编号
MemberID=0                 // 场景服务器编号
CollideRootPath            // （lua 读，可缺省）碰撞信息根目录
[Logic] Modules=1 Module0=FsGame.dll   // Path 空=当前目录
[Inner]    // Addr=127.0.0.1 Port=5001（注释态；缺省走 main.lua 公式）
[Broadcast] // Addr=127.255.255.255 Port=4001（注释态）
```

main.lua 里的**缺省公式**（ini 未配置时）：Inner `16300+MemberID`、
Broadcast `127.255.255.255:15000`、Echo 维护口 `15300+MemberID`。

## 2. 模块构成

标准 Fx 模块接口 8 导出（FxModule_GetType/GetVersion/GetSpace/Init/
GetEntCreator/GetIntCreator/GetLogicCreator/GetFuncCreator），注册实体类
**GameMember**（EntCreator）。FsSoloMember.map 的 obj 布局（同源定案）：

| obj（FsSoloMember.map） | 职责 |
|---|---|
| SoloMember.obj（635 符号） | 服务类本体：Start/Stop/Execute(=DoWait)/属性 getter/setter/CheckName/Dump* |
| SoloMemberWorldMsg.obj | world 消息 17 型处理器 |
| SoloMemberPublicMsg.obj | public 数据面 14 型处理器 |
| SoloMemberStoreMsg.obj | store 存档面 4 型 + Manage 发布 |
| SoloMemberScene.obj / SoloMemberState.obj | 场景管理与状态机 |
| SoloMemberAuctionMsg.obj | 拍卖行转发（solo 特有） |
| MemberPlayers.obj / MemberScenes | 玩家注册表 / 场景注册表 |
| Kernel*.obj（~1500 符号） | CKernel：游戏对象内核（Create/Place/Switch/Store…） |
| GameObject*.obj / GameScene / GamePlayer / GameNpc / GameItem / GameHelper / GameVisible / GameMotion / SceneBeat / ClassSet / Resource / EventCmdMnger / Reactor(内嵌) / InnerHook / AppRecord / EchoHook | 对象系统全家桶（自带 Reactor 线程泵） |

## 3. main.lua 装配序列（集群形态）

```
LogPool(log\member 前缀, TraceMax=256) → 读 GameMember.ini
→ LogicLoader.Load(FsGame.dll) → CollideManager
→ EchoSock(15300+M) ── bind 脚本 echo.lua
→ BroadSock(Bind=InnerAddr, 广播=127.255.255.255:15000)
→ AppManager("member_<S>_<M>", AppAddress=echo 地址)   // 共享内存注册表
→ Adapter(InnerAddr:16300+M, ConnectMax=64, WorkThreads=1,
          InBufLen=OutBufLen=1MiB)
→ GameMember 实体：23 属性 6 方法（§3.1）
   ├ LoadNameFilter(Res\filter.txt)    // 名字过滤
   ├ LoadLocations(Res\location.ini)   // 全局落地点
   └ Start()
→ AppManager.SetAppState(WAPS_STARTING→WAPS_OPENED)
→ 顺序启动 AppManager→Adapter→BroadSock→EchoSock→GameMember
```

关闭序（close()）：BroadSock→Adapter→EchoSock→GameMember→AppManager→
LogPool（广播先停、监控最后摘牌）。

### 3.1 GameMember 属性/方法面（PDB prop/func_struct 定案）

属性（getter/setter 全对）：DistrictId、ServerId、MemberId、State、
MaxLetterNum、MaxRunningTime、PlayerSingleBuffer、DebugHalt、
AppearAtOnce、HideEmptyProperty、ScriptReload、TaskReload、SightOptimize、
ScriptPath、TaskPath、CollideRootPath、Console、EchoSock、BroadSock、
AppManager、Adapter、LogicLoader、CollideManager。

方法：Start()、Stop()、LoadNameFilter(path)、LoadLocations(path)、
DumpScene(file)、DumpPerformance(file)。

Solo 形态另有：BindCore、UseThread、UseQuickLog、UseAppRecord、BeatTime、
ScanTime、FastScanTime、SlowScanTime、SceneGridSize、MergeMoving、
MergeProperty、MergeObjects、LogFrameTime。

## 4. 内网通讯与 ident 命名空间（核心增量）

Member 同时持有**两个通讯面**：

1. **Adapter（受面）**：内网 TCP 侦听 16300+M，`[u32 len][帧]` 分帧
  （FsComm §3.1），帧体 = `{src u32, dst u32, type u32} + payload`（12B 头）。
2. **InterConnect（发面）**：CInterConnect（FsComm）主动外连池，
   SendMsg/SendMsg2 按 comm_id 寻路（`GameMember::GetInterConnect`，
   this+0xd8）。

### 4.1 ident（comm_id）命名空间（ReceiveMsg 路由定案）

`CAdapterHook::ReceiveMsg`（RVA 0x313c0）逐分支实证：

| src ident | 路由 | 队列（this+） |
|---|---|---|
| type==0xFFFF | **注册帧**：绑定连接↔ident（vtbl+0x98） | — |
| 0 | world | PutWorldMessage → [0xb8] |
| 1 | public | PutPublicMessage → [0xc0] |
| 4..7 | store | PutStoreMessage → [0xbc] |
| 8..0x17 | entry（8+EntryIdx，16 槽） | PutEntryMessage（直投，见 §7） |
| 其余 | `"(CAdapterHook::ReceiveMsg)message source error"` 丢弃 | — |

长度门：`<0xC → "message size error"`。三队列全走 CMsgDeque2
（SwapMsgVector/Peek 双缓冲，IOCP 线程生产、逻辑线程消费）。

**集群 ident 全景**（与 entry/world 侧一致）：world=0、public=1、
store=4..7、entry=8..0x17、member=0x18..0x27（member 连入 entry 时
以 0x18+MemberIdx 注册——见 entry/inner.go validSource 白名单）。

### 4.2 DoWait 主循环（GameMemberProc→DoWait，RVA 0x39910）

```
每帧：SwapMsgVector 后逐条 Peek——
  world 队列  → ProcessWorldMsg(ident, type, data, size)
  store 队列  → ProcessStoreMsg(...)
  public 队列 → ProcessPublicMsg(...)
  → 遍历场景表 [0xf4..0xf8]（vector<CGameScene*>）：CGameScene::DoWait
  → MemberStateMachine（限时 MaxRunningTime 保护）
  → BroadcastMemberInfo（§8 周期广播）
```

单帧超时由 MaxRunningTime 保护（DebugHalt 时挂起观察）。

## 5. world 消息表（ProcessWorldMsg，跳表 RVA 0x424d0/0x42518 定案）

| type | 处理器 | payload 语义（实证偏移） |
|---|---|---|
| 0x0 | CommandCreateScene | 场景创建指令（scene_id/类型/名字/配置） |
| 0x1 | CommandCreateClone | 克隆场景创建 |
| 0x2 | CommandDeleteClone | 克隆场景删除（§6 生命周期断言串） |
| 0x3 | CommandCheckScene | 场景存在性应答 |
| 0x4 | CommandSceneAll | 全量场景上报（memmove 拼包回 world） |
| 0x5 | CommandSceneAdd | 增量场景上报 |
| 0x6 | CommandSceneRemove | 场景移除上报 |
| 0x7 | CommandPlayerAll | 全量玩家上报 |
| 0x8 | CommandPlayerAdd | `{scene@0, ?@4, ?@8, name[36]wchar@0xC, account@0x54}` → CMemberPlayers::AddPlayer |
| 0x9 | CommandPlayerRemove | `{scene@0, ?@4, name@8}` → GetPlayer→RemovePlayer |
| 0xA | CommandPlayerChange | 玩家换场景（ChangePlayerSceneId） |
| 0xB | CommandRoleInfo | 角色档案下发（落 RoleInfo 表） |
| 0xC | CommandRoleAdd | 角色新增 |
| 0xD | CommandRoleRemove | 角色删除 |
| 0x2D | CommandItemInfo | 商品表下发（GetItemInfo 缓存） |
| 0x37 | CommandCloseMember | 关服指令（→状态机 Closing） |
| 0x38 | CommandDumpPerformance | DumpPerformance(perf.txt) |

（solo 形态另有 RoleRecreate/ClearPerformance/MemoryRole —— 快照年龄差异。）

## 6. 状态机与场景生命周期

`MemberStateMachine`（RVA 0x40410）按 `this+0x24` 分派：
**0=Unknown（启动中）→ 1=Opened → 2=Closing → 3=Closed**。

- Opened 收 CloseMember → 置 Closing；`"(GameMember::MemberStateClosing)
  scene has player"`——**场景清空才允许落 Closed**（踢线→存档→摘场景）。
- 进程级另有 AppManager 的 WAPS_*（STARTING/OPENED/CLOSING/CLOSED…），
  两层状态并行：WAPS 对监控面、member 状态对集群面。

场景面（GameMember::CreateScene/CloneScene/DeleteClone/CanCreateScene/
IsPrototypeScene，断言串全存）：

- 场景分**原型（prototype）与克隆（clone）**：CloneScene 找原型
  （`"prototype not found/not created/error"`）→ 复制；DeleteClone 仅对
  clone 型（`"scene type not clone"`）且场景无玩家（`"scene has player"`）
  才可删；CanCreateScene 是配额/合法性预检（`"create scene failed"`）。
- `(GameMember::CreateScene)no scene creator / scene creator type error /
  scene load error / scene initialize error`——创建器来自 FsGame 逻辑
  模块注册的 scene creator。

## 7. entry 消息与客户端流（PutEntryMessage，RVA 0x37f50）

entry（src 8..0x17）消息**不进队列**，直投两路：

- `type==3（entry_forw set_client_online）` 且 PlayerSingleBuffer 开：
  `{ident,serial}` 剥 8B → CPlayerBuffers::PutMessage（玩家缓冲池，
  进场流程用）。
- 其余 → `SendToScene(scene, 0x10=CommandClientMessage, data, size)`
  ——场景命令命名空间（37 型：AddPlayer/SwitchPlayer/RemovePlayer/
  SendLetter/QueryLetter/RecvLetter/LookLetter/CleanLetter/RejectLetter/
  SceneBroad/TimerSaving/Disconnect/Continue/QueryGift/GetGift/SellCard/
  UnsellCard/BuyCard/BuyItem/GetPoints/ItemInfo/Debug/GmNews/
  PublicComplete/ServerClose/SceneCommand/NamedCommand/PlayerCommand/
  GroupCommand/CreateRole/Warning/FromCharge/FromStat/SysStatInfo/
  FromManage/FromExtra + MessageSelect/Ready/RequestMove/Speech/Custom
  客户端消息五型）。

**下行**（GameMember::SendToClient，RVA 0x39630）：

```
serial 高 16 位 = EntryIdx → dst ident = 8 + EntryIdx
CInterConnect::SendMsg2(dst, 0, type=7 /*entry_forw send_to_client*/,
                        hdr={ident,serial}(8B), data, size)
```

即 member → entry 的客户端下发**就是 entry_forw 7 型**（与
entry/forward.go 定案的 16 型控制面完全对接）；serial 的
`(entryIdx<<16)|序号` 编码在此定案。SendToWorld/SendToPublic/
SendToStorage(4..7 各发=SendToAllStorage) 同为 InterConnect 发面。

## 8. BroadcastMemberInfo（RVA 0x37fa0，UDP 0x28B 包）

经 BroadSock 周期广播（DoWait 每帧触发、内部门限节流），payload：

```
+0x00 u32 kind = 3                 // member 信息包（entry=1/world=2 同族）
+0x04 u32 magic = 0x300            // 与 lister 注册包同 magic（分区门）
+0x08 …                            // DistrictId/ServerId/MemberId（this+0x28/2C/30）
+0x10 char ip[16]                  // CopyString 定长（通告地址）
+0x1c …                            // 端口等（BroadSock getter）
共 0x28 字节
```

（kind/magic 与 FsLister 侧 RecvEntryServerInfo/RecvWorldServerInfo 报文
同族——lister 聚合 entry/world，member 包供集群发现。）

## 9. public 数据面（ProcessPublicMsg，0x3ED18 跳表定案）

public 服务器把全服共享数据镜像推给每个 member，member 侧本地只读副本：

| type | 处理器 | 语义 |
|---|---|---|
| 0x0 | PublicSpaces | 空间声明清单（`"space name repeat"`） |
| 0x1 | PublicData | 空间→数据对象建立 |
| 0x2 | PublicAdd | 数据加入（`"data already exists"`） |
| 0x3 | PublicRemove | 数据移除 |
| 0x4/0x5 | PublicAddAttr/RemoveAttr | 属性列增删 |
| 0x6/0x7 | PublicAddRec/RemoveRec | 记录表增删（列类型校验） |
| 0x8/0x9/0xA | PublicRecAddRow/RemoveRow/Clear | 行级操作 |
| 0xB/0xC | PublicRecChange/RecSetRow | 行改写 |
| 0xD | PublicMessage | 透传场景消息（`"scene not active/no scene"`） |

存储结构：`FindPubSpace/GetPubSpace/GetPubSpaceCount/GetPubSpaceList`
（CPubSpace + PubDataImpl），每 space 下 data→{attr 集, rec 集}。

## 10. store 存档面（ProcessStoreMsg，type-9 基址 0x40A04 定案）

| type | 处理器 | 语义 |
|---|---|---|
| 0x9 | PublishChunkInfo | 存档块清单（`"no scene chunks/scene id error"`） |
| 0xA | PublishLoadChunk | 块加载（`"chunks amount error"`） |
| 0xB | PublishSaveChunk | 块保存应答 |
| 0xC | PublishDeleteChunk | 块删除 |

块=场景分片存档单位；另有 PublishManageInfo/Base/Update/Remove
（Manage 表下发，UpdateManageTableAllConfig/get xml info）。

## 11. Go 复刻（internal/fsmember + cmd/member）

| 官方件 | Go 等价 | 状态 |
|---|---|---|
| GameMember.ini 方言 | `fsmember.Config` + `ParseGameMemberINI`（缺省公式 16300/15300+ID、广播 15000） | ✅ |
| Adapter 受面 + ReceiveMsg 路由 | `fsmember.Inner`（fscomm.Adapter + ident 路由 0/1/4-7/8-0x17 + 0xFFFF 注册） | ✅ |
| InterConnect 发面 | `fsmember.Outer`（fscomm.MultiSock 池，按 ident 寻路 SendToWorld/Public/Storage/Client） | ✅ |
| world 消息 17 型 | `worldHandlers` 表 + 场景/玩家注册表落盘 | ✅ |
| public 14 型 | `publicSpaces` 镜像（space→data→attr/rec） | ✅ |
| store 4 型 | `chunkHandlers`（清单/加载/保存/删除状态记录） | ✅ |
| 状态机 0-3 | `fsmember.State*` + CloseMember 门（场景清空才 Closed） | ✅ |
| 场景注册表 | `SceneRegistry`（prototype/clone、Create/Clone/Delete/CanCreate、官方断言串） | ✅ |
| 玩家注册表 | `PlayerRegistry`（AddPlayer/RemovePlayer/FindByName/FindByAccount/ChangeScene/CleanConnect） | ✅ |
| entry 流 | PutEntryMessage 双路（online→PlayerBuffers；else→scene 0x10）+ SendToClient(7 型, serial→8+entryIdx) | ✅ |
| BroadcastMemberInfo | 0x28B kind=3/0x300 周期包（fscomm.BroadSock） | ✅ |
| 维护面 echo.lua | fscomm.EchoSock（welcome/quit/end/perf + 实体调试回调面） | ✅ |
| AppManager | fscomm.AppManager（member_S_M + WAPS） | ✅ |
| **同名多角色装载**（game_*.exe 一壳多 ini） | `fsmember.Assemble`（官方 main.lua 装配段）+ cmd/loader 集群模块注册（fscomm/fscollide/fsmember/fssolomember）+ 主实体钩子装配 | ✅ |
| 名字过滤/落地点 | `NameFilter`（HasInvalidChar/Filter 双口径）+ `Locations`（location.ini） | ✅ |
| CKernel/GameObject 全家桶 | 不在本次（ECS 内核已由 internal/ecs 承担，见任务清单 §一） | 🔜 后续 |

验收：`go test ./internal/fsmember`（路由/状态机/场景生命周期/玩家表/
消息表/jump-table 对拍 golden）。

## 12. 未尽事项

- CKernel 对象内核（1500+ 符号）与 FsGame.dll 玩法层：属玩法域，
  由 ECS + Lua 行为层路线承接（里程碑 3）。
- PublishManage* XML 表与 Manage 面细节、拍卖行转发（solo 形态
  AuctionMessage）待 world/public 侧对拍时补。
- CPlayerBuffers 进场缓冲与 CGameScene::CommandAddPlayer 的
  serialize player data 细节：等 persist 角色存档（nx_chunks）合流。
