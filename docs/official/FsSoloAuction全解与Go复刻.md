# FsSoloAuction.dll + game_auction.exe 全解与 Go 复刻（拍卖行数据服务器宿主）

> 对象：`BGserverCJ\Debug\FsSoloAuction.dll`（978944 B，2013-07-05 构建，
> Debug 全符号 .map/.pdb，源码路径 `c:\wuyf\svn\serveriii\branches\publish20130508\server\fssoloauction\`）
> 与 `game_auction.exe`（589824 B）
> 日期：2026-11-19　分支：`official`　复刻落点：`internal/fssoloauction` + `cmd/auction`
> 伴生文档：`fsauctionlogic-拍卖行数据服务器.md`（业务核心）；
> 逆向中间产物 `tmp_research/auction/`（soloauction_framework.md 及 dis_*.txt）

---

## 一、宿主 exe 与 DLL 的关系

**game_auction.exe 与 game_lister.exe / game_entry.exe 是同一个文件**
（MD5 `195e1ce1e4a2828d1b26e45608cc3183`；FsSoloLoader.exe 是另一构建
`c2f2df73…`，职责同型）。它是通用 solo 宿主壳：按自身文件名读同名 ini
（`[System] Sleep/MainEntity/MainScript`、`[Module]` 裸行 DLL 列表、
`[Script] Path/Reload`），装载 fssoloauction.dll 等模块后把控制权交给
Lua 主控（`..\lua\soloauction\main.lua`）。

`game_auction.ini` 的 `[Module]` = `fskit.dll / fscomm.dll /
fssoloauction.dll`；`[Main] AuctionLogicDll=fsauctionlogic.dll`。
**Go 复刻不模拟 DLL 装载**——cmd/auction 进程内直接装配等价组件
（与 cmd/lister、cmd/solo 同范式）。

FsSoloAuction.dll 导出 8 个：`FxModule_GetType("FxModule")/GetVersion/
GetSpace/Init/GetEntCreator/GetIntCreator/GetLogicCreator/GetFuncCreator`
（标准 Fx 模块契约）。

## 二、模块清单（.map obj 归组 → Go 落点）

| obj（符号数） | 官方类 | 职责 | Go |
|---|---|---|---|
| SoloAuction (101) | `SoloAuction : IEntity` | 实体本体：20 Lua 属性 + 10 mid 方法 + Start/Stop/Execute | `server.go Server` |
| AuctionKernel (29) | `CAuctionKernel : IAuctionKernel`（28 槽 vtable @0x10126774） | 暴露给逻辑 DLL 的内核 | Server 即内核（出站路由/注册表/Echo） |
| AuctionData(72)+Impl(75) | `CAuctionData : IAuctionData`（72 槽） | 数据对象（attr+多 Record） | `internal/auction`（数据模型内联） |
| AuctionRecord(50)+RecordIndex(18) | `CAuctionRecord`/`CRecordIndex` | 记录与索引 | `internal/auction record.go` |
| RecData(55)+PropData(20) | `CRecData`/`CPropData` | 行存储/值单元 | `internal/auction` + `store.go storeWriter/Reader` |
| AuctionDataArchive(18) | `CLoadArchive`/`TStoreArchive` | 序列化原语 + PUB4 块 | `store.go` |
| Reactor(32) | `CReactor`（IOCP） | socket 引擎 | `internal/reactor`（既有复刻） |
| InnerHook(16) | `CInnerHook`（comm_t） | 内网 TCP 16003 | `inner.go InnerLink` |
| EchoHook(10) | `CEchoHook` | 调试口 16001 | `echo.go EchoConsole` |
| AuctionTimer(15) | `CAuctionTimer` | 逻辑定时器（HCheckDueTime 300s） | `server.go otMS` 累计 |
| AuctionScenes(12)/AuctionPlayers(12)/NameUid(26) | 三注册表 | world 镜像（场景/玩家/角色） | `registry.go` |
| SoloAuctionWorldMsg(14) | `ProcessWorldMsg`+World* | world 连接消息（id2 0..0xC） | `server.go processWorldMsg` |
| SoloAuctionStoreMsg(6) | `ProcessStoreMsg`+StoreAuction* | store 存取（0x29..0x2D） | `server.go processStoreMsg`+`store.go` |
| SoloAuctionState(7) | 状态机五处理器 | Unknown→…→Closed | `state.go StateMachine` |
| AppRecord(17)/LogShareMem(11)/MemPool(10) | 运维件 | 共享内存状态/日志/内存池 | 不复刻（日志呈现等价） |
| Routine(7)/AppTool(4) | 工具 | vary 编解码/路径/控制台色 | `store.go`/cmd |

Lua 属性（prop_struct 注册，getter/setter @0x100A6800–0x100A8300）：
`AppName/InnerAddr/InnerPort/EchoAddr/EchoPort/BroadAddr/BroadPort/
DistrictId/ServerId/SaveMinutes/SaveBatchSize/ExitForceSave/
MaxRunningTime/UseAppRecord/UseQuickLog/UseThread/EnableMemPool/
LogLauncher/LogicLoader/State`(只读)。
mid 方法：`Start/Stop/SetAppState/SetConsoleColor/WriteLine/SendEchoLine/
CloseEcho/DumpReactor/GetMemPoolSize/GetMemFreeSize`。

## 三、内网 16003（CInnerHook，字节级）

### 3.1 TCP 帧（双向同格式，小端）

```
+0x00 u32 N        // 其后字节数 = 12 + len(payload)；N ≤ 0x3FFFFC（0x400000-4）
+0x04 u32 comm_id  // 发送方 comm 身份
+0x08 u32 id1      // 一级路由（member 消息 = 目标 member 索引透传）
+0x0C u32 id2      // 0xFFFF=identify 握手 / 0xFFFE=退出 / 业务消息号
+0x10 payload
```

- **identify**：外连 OnConnect（@0x1006CE40）立即发 `{12, own_comm, 0,
  0xFFFF}`；收侧登记 `m_ConnectorByComm[comm]=conn` 并日志
  `"%s connector_id %d comm_id %d received"`。
- **comm 命名空间**（COMM_ID_MAX=64 断言）：`0=World`、`2=本服`
  （Start @0x100AAABD `SetCommIdent(2)`）、`4..7=Store`、
  `0x18..0x2F=Member`；来源路由（ProcessInnerMsg @0x100AED80）：
  `0→ProcessWorldMsg / 4..7→ProcessStoreMsg / 0x18..0x2F→
  ProcessMemberMsg / 其它→"message source error! comm:%d"`。
- 寻址发送（SendMsg @0x1006DE80）：**comm 寻址非 conn id**；
  `SendToWorld`=comm0、`SendToStore(index<4)`=comm 4+index、
  `SendToMember(index<0x18)`=comm 0x18+index（MAX_MEMBER_NUM 断言）。

### 3.2 member 业务消息（ProcessMemberMsg @0x100B1620）

- `id2=0` **MemberRegister**：断言 len==8
  （`auction_forw_register_t`）；返回 `state==Opened`（1/0）——
  member 侧据此重试直至开市。
- `id2=2` **MemberMessage**（@0x100B0CF0）：payload =
  `auction_forw_message_t{ u32 param2; u32 vary_count; vary[] }`
  （len≥8 断言）；`get_vary_message(count, archive(payload+8), vl)` →
  callee `OnAuctionMessage(kernel, id1, param2, vl)`
  （vtable+0x0C，无状态门——业务分发先于开市也成立）。
- 出站 `SendAuctionMessage(target, sub, vl)`（@0x100AE1F0）：
  target∈[0,0x18) 断言（"member id error"）→ forw 序列化
  （buf[0]=sub、buf[4]=count、Seek(8) 后 put_vary_message）→
  `SendToMember(target, 0, 0xD, forw)`；>1MB 仅 trace 仍发送
  （"message exceed %d bytes"）。
- vary 元素流 = **无数量前缀**的 `u8 tag+值` 序列（tag 2..8 同
  proto.Var*；string=u32 len含NUL+字节、widestr=u32 字节长含L'\0'+UTF16LE、
  object=2×u32）。proto.EncodeElements/DecodeElements 即此格式。

### 3.3 world 消息（ProcessWorldMsg @0x100B4230，id2 0..0xC）

| id2 | 消息 | 负载（实证/推断） |
|---|---|---|
| 0 | WorldServerState | 呈现（布局推断） |
| 1 | WorldClose | 触发关服（evClose） |
| 2 | WorldSceneAll | 0x90 定长循环（全量快照） |
| 3 | WorldSceneAdd | `member_forw_scene_add_t`(0x90)：`+0 server,+4 member,+8 scene,+0xC clone,+0x10 name[0x40],+0x50 resource[0x40]`（实证） |
| 4 | WorldSceneRemove | `{i32 scene_id}`（推断） |
| 5 | WorldPlayerAll | 0x9C 定长循环（全量快照） |
| 6 | WorldPlayerAdd | `member_forw_player_add_t`(0x9C)：`+0 connect,+4 server,+8 player,+0xC online,+0x10 wname[0x48],+0x58 account[0x44]`（实证） |
| 7 | WorldPlayerRemove | `{i32,i32,wname}`（推断） |
| 8 | WorldPlayerChange | 同上，翻转在线位（推断） |
| 9 | WorldRoleInfo | CLoadArchive 全量重建：`Clear` 后循环 `{ReadWideStr,ReadString,ReadBool}`（实证） |
| 0xA/B/C | Role Add/Remove/Recreate | `{widestr,str[,bool]}` archive 流（元素序实证，容器推断） |

### 3.4 store 消息（ProcessStoreMsg @0x100B2890）

拍卖→store：`(0,0x3F,{serverId},4)` 注册（Unknown 态一次性）、
`(0,0x40,{district,server},8)` 加载请求（LoadAuctionHall 态每 tick 重发）、
`(0,0x41,attr块)`/`(0,0x42,record块)` 退存。
store→拍卖：`0x29` 属性块回放 / `0x2A` 记录块回放 / `0x2B` 加载完成
（`{u32 result}`；result!=0 → Opened+OnAuctionDataLoaded，==0 →
"load pub data failed"）/ `0x2C`/`0x2D` 存盘空应答。

## 四、UDP 15000（服务发现）

- 自宣（BroadcastServerInfo @0x100AEC50，每 TimerCheck 1s 发）：
  0x24B `{u32 type=5}{u32 magic=0x3001}{char addr[0x10]=InnerAddr}
  {u32 port=InnerPort}{u32 district}{u32 server}`。
- 收侧（OnBroadcast @0x100AB780）只认 magic 0x3001：
  type1=World 自宣（0x48B，分区/服校验 + `[0x24]<0x10`）→
  `CheckConnect(comm=0, addr, port)` 主动外连；
  type2=Store 自宣（0x28B，`[0x24]∈4..7`）→ 外连该 comm。
  广播通道只做发现，不承载业务。

## 五、状态机（SoloAuctionState.obj，this+0x30 五态索引）

```
Unknown --(store identify 完成=事件1，一次性闩锁 @this+0x13E：发 0x3F 成功)--> LoadAuctionHall
LoadAuctionHall --(每 tick 发 0x40；0x2B result!=0)--> Opened
Opened --(事件3：OnServerClose + SaveIndex=-1)--> Closing
Closing --(每 tick exit=true 批量退存；扫完→SendToWorld(0,1,{1},4) 成功)--> Closed
```

- TimerCheck（1s，@0x100AEEF0）：tick 计数 `>5` 复位（每 6s 一次
  `SetAppInfo("<info state=\"%s\" tick=\"%d\" />")`）→ 每 tick 广播自宣 →
  `AuctionStateMachine(0, elapsed)`。
- Opened tick = callee `OnFrameExecute`（no-op）+ `CAuctionTimer::Run` +
  存盘周期：`g_SaveSeconds++ ≥ SaveMinutes*60` 起，每 tick 按
  `SaveBatchSize` 批量 `InnerSaveAuctionData(p,false)`，
  `SaveIndex` 游标扫完复位 -1（@dis_state_opened 段）。
- **HCheckDueTime 间隔 0x493E0=300s**（FsAuctionLogic InitTimers
  @0x100948B7 push 0x493E0，repeat=0）——超时结算 5 分钟粒度。

## 六、持久化（PUB4 归档）

FsSoloAuction 本身不落盘、无 SQL（导入表/GBK 串扫描定案）。存取全部
经 store：

- attr 块（SaveAttr @0x1008D8D0）：
  `+0 u32 "PUB4"|+4 u32 保留|+8 uid[0x24]|+0x2C name[0x24]|+0x50 u16 0|+0x52 attr流{str|u8 type|value}…`
- record 块（SaveRecordHead @0x1008E470 + Body @0x1008E950）：固定头
  0xAC（`+0x2C 名[0x24]、+0x98 rowMax、+0x9C rows、+0xA0 cols、
  +0xA4 索引数、+0xA8 0`）→ 流：每列 `u8 col_type` → 每索引 `{u32,u8}` →
  `WriteInt(索引数)` → 每索引定义 → 体 `WriteInt(start) WriteInt(end)` +
  每行每列 `{u8 脏标志, value}`；**分片 ≤0x100(256) 行**。
- 加载链：连 store → 0x3F → 0x40（每 tick）→ 0x29 建对象/装属性 →
  0x2A 装行 → 0x2B → Opened + OnAuctionDataLoaded（LoadResource →
  InitAuctionData → InitRecords → InitTimers）。

## 七、Go 复刻对照与差异

| 官方 | Go | 备注 |
|---|---|---|
| game_auction.exe 宿主壳 | `cmd/auction` | ini 释放式（internal/auction 内置默认）；旗标>ini>缺省 |
| CReactor | `internal/reactor` | 既有复刻 |
| CInnerHook | `InnerLink` | 帧格式/identify/comm 路逐字节一致；半包累积语义同 |
| member forw | `encodeForw/decodeForw` + `proto.EncodeElements/DecodeElements` | 无数量前缀元素流 |
| CEchoHook | `EchoConsole` | 行协议一致（\r/\n 皆终止、1023 缓冲、空行不分发、SendLine 不补终止符）；**Lua 表达式→内置命令集**（state/stat/items/save/rebuild/clear/roles/scenes/players/help） |
| 广播发现 | `broadcastServerInfo/onBroadcast` | 0x24 自宣逐字节一致；world/store 自宣外连 |
| 状态机 | `StateMachine` | 五态+事件序/一次性闩锁/0x3F/0x40/关播全对齐 |
| store 协议 | `processStoreMsg/store.go` | PUB4 attr/record 块编解码 + 0x100 分片 + 0x29/0x2A/0x2B 回放 |
| fsauctionlogic.dll | `internal/auction` | 既有复刻（Logic.Dispatch ⇔ OnAuctionMessage） |
| CAuctionTimer/HCheckDueTime | `otMS` 300s 累计 | 官方 0x493E0 |
| world 三注册表 | `registry.go` | SceneAdd/PlayerAdd/RoleInfo 布局实证；次要消息按对称推断 |
| CAppRecord/CLogShareMem/CMemPool/互斥单实例 | 不复刻 | 日志呈现等价（`<info state=… />` 每 6s） |

**Go 扩展（官方无，可用性优先）**：
- `LocalStore`（`-local-store <dir>`，缺省 `data/auction`；`off` 关闭）：
  无外部 store 进程时以同一套 0x29/0x2A/0x2B 处理函数完成加载序，
  PUB4 块原样落盘（`main.attr/main.record`）；接真 store 时网络路径
  优先、本地仅作双写镜像。
- `-store host:port`：不经广播直连 store 0。
- `Options.ManualTick`/`Server.TimerCheck`：测试钩子（官方 1s 定时器
  的手动化）。

测试（`go test ./internal/fssoloauction/`）：帧累积半包/粘包、forw
往返、announce 编解码、状态机全流转（含 0x3F 闩锁/退存重试/关播）、
PUB4 记录块往返+分片边界+Row 互转、**全栈 TCP 回环**（member identify→
寄售→[7] 回执）、300s 超时结算广播（target=0 流拍 [12]）、LocalStore
存盘→重启回放、echo 行协议、未知 comm 拒收。

## 八、证据索引（FsSoloAuction.dll 内 VA）

| 内容 | VA |
|---|---|
| SoloAuction::Start / Stop / Execute / TimerCheck | 0x100AA750 / 0x100ABD80 / 0x100AA350 / 0x100AEEF0 |
| SetCommIdent(2) / CreateBroadcast / CreateListener 调用点 | 0x100AAAC8 / 0x100AAB2F / Start 后段 |
| BroadcastServerInfo / OnBroadcast | 0x100AEC50 / 0x100AB780 |
| CInnerHook::ReceiveMsg / OnReceive / SendMsg / OnConnect / CheckConnect | 0x1006D630 / 0x1006D410 / 0x1006DE80 / 0x1006CE40 / 0x1006DC50 |
| SendToWorld / SendToAllStore / SendToStore / SendToMember / SendToEntry | 0x10097300 / 0x100976C0 / 0x100B1A80 / 0x100AE4E0 / 0x100AD320 |
| SoloAuction::SendAuctionMessage（forw 构造+1MB 告警） | 0x100AE1F0 |
| ProcessInnerMsg / ProcessWorldMsg / ProcessStoreMsg / ProcessMemberMsg / MemberMessage / MemberRegister | 0x100AED80 / 0x100B4230 / 0x100B2890 / 0x100B1620 / 0x100B0CF0 / 0x100B0C60 |
| AuctionStateMachine / Unknown / LoadAuctionHall / Opened / Closing / Closed | 0x100B1800 / 0x100B1950 / 0x100B1BF0 / 0x100B1D00 / 0x100B2040 / 0x100B2270 |
| GetState（五名跳表） / InnerStoreConnected | 0x100B1710 / 0x100AEBF0 |
| StoreAuctionLoadAttr / LoadRecord / LoadRecordOk | 0x100B22F0 / 0x100B2570 / 0x100B2720 |
| InnerSaveAuctionData / SaveAttr / SaveRecordHead / SaveRecordBody | 0x100ADAC0 / 0x1008D8D0 / 0x1008E470 / 0x1008E950 |
| put_vary_message / get_vary_message / put_outer_message | 0x100A53A0 / 0x100A57D0 / 0x100A5BC0 |
| g_SaveSeconds / m_SaveIndex(this+0x138) / 闩锁(this+0x13E) | 0x101494CC / — / — |
| "PUB4" 写入点（0x50554234） | 0x1008D96B |
| HCheckDueTime 间隔 0x493E0（FsAuctionLogic InitTimers） | 0x100948B7 |
