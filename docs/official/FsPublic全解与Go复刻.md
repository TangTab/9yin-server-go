# FsPublic.dll + FsPubLogic.dll 全解与 Go 复刻

> 权威快照：`E:\Workspace\project\game\九阴真经ol\BGserverCJ\Debug\FsPublic.dll`
>（970,752 B，2014-03-22）+ `FsPublic.pdb`（4.2MB，5,286 pub 符号，无 .map）；
> 逻辑模块 `FsPubLogic.dll`（3,313,664 B）+ `FsPubLogic.map`（3.1MB，1,987 符号）+
> `FsPubLogic.pdb`（18MB）。同源单机形态 `FsSoloPublic.dll`（1,040,384 B）带
> `.map`（1,724 符号，类结构以 map 定案；服务类 SoloPublic）。宿主脚本
> `BGserverCJ\Lua\GamePublic\main.lua` + `echo.lua`。Go 复刻：
> `internal/fspublic` + `cmd/public`。
>
> 源码路径（assert 串泄露）：`d:\flexis\client\fspublic\{gamepublic,
> gamepublicmembermsg,gamepublicstate,gamepublicstoremsg,gamepublicworldmsg,
> adapterhook,broadsockhook,pubdata,pubdataimpl,pubdataarchive,pubkernel,
> pubspace,pubtimer,publicrecord}.cpp`、`d:\flexis\client\utils\{storearchive,
> loadarchive}.h`、`d:\flexis\client\public\{var,varlist,vartable,inlines,
> converts,exceptmsg}.h`、`d:\flexis\client\network\interconnect.h`、
> `d:\flexis\client\tools\{recdata,propdata}.cpp`。

## 1. 进程定位与部署形态

Public 是集群的**公共数据服务器**（authoritative pub store）：持有全部跨场景
共享数据空间（CPubSpace → CPubData → {CPropData 属性, CRecData 记录表}），
member 进程持有只读镜像并与本服同步；world 控制其状态、store 持久化其数据、
FsPubLogic.dll 在其上挂载玩法域（Domain_*：公会/能量井/竞技场/排行…）。

| 启动方式 | ini | 形态 |
|---|---|---|
| `GamePublic.bat` → `FsLoader.exe GamePublic.ini` | 大写系集群 | FsKit+FsComm+**FsPublic**，MainEntity=MainConsole，脚本 `..\Lua\GamePublic`，逻辑模块 **FsPubLogic.dll**（`[Main] PublicLogicDll`） |
| `game_public.exe`（裸跑，`<exe基名>.ini` 约定） | `game_public.ini` 小写系 | fskit+fscomm+**fssolopublic**，MainEntity=SoloPublic，脚本 `..\lua\solopublic`，逻辑模块 **fspublogic.dll** |

装载器五步序列与 game_member/lister/entry/world.exe 完全同构
（见《FsLoader全解与Go复刻.md》§3）。

### 1.1 GamePublic.ini 官方方言（集群形态全文）

```ini
[System]
Sleep=1                    // 主循环休眠毫秒（<0 不休眠）
MainEntity=MainConsole
MainScript=main
[Module]
FsKit.dll / FsComm.dll / FsPublic.dll
[Script] Path=..\Lua\GamePublic  Reload=1
[Main]
DistrictID=1               // 游戏分区编号
ServerID=11                // 服务器编号
PublicLogicDll=FsPubLogic.dll   // 玩法逻辑模块（Callee 模式）
[Inner]    // Addr=127.0.0.1（注释态；缺省见下）
[Broadcast] // Addr=127.255.255.255 Port=4001（注释态）
```

main.lua 缺省（ini 未配置时）：Inner `127.0.0.1:16003`、Broadcast
`127.255.255.255:15000`、Echo 维护口 `127.0.0.1:15003`——**固定值无公式**
（区别于 member 的 16300+M/15300+M）。

### 1.2 solo 形态差异（Lua/solopublic/main.lua）

- SoloPublic 直挂属性（无独立 EchoSock/AppManager 对象装配）：
  `AppName/DistrictId/ServerId/**SaveMinutes(缺省 20)**/LogicLoader/
  InnerAddr/InnerPort(缺省 0=不监听!)/OuterAddr/OuterPort/BroadAddr/BroadPort/
  EchoAddr/EchoPort/LogLauncher("game_log game_log.ini daemon")`
- 空间注册不同：solo `AddPubSpace("domain","nx_domains") + ("guild","nx_guilds")`；
  集群 `AddPubSpace("pubdata","nx_pubdata") + ("guild","nx_guilds")`
- solo 脚本引用未定义的 outer_addr/outer_port（官方脚本遗留缺陷，nil 赋值）

## 2. 模块构成

标准 Fx 模块接口 8 导出（FsPublic.dll）：FxModule_GetType/GetVersion/GetSpace/
Init/GetEntCreator/GetIntCreator/GetLogicCreator/GetFuncCreator，注册实体类
**GamePublic**（EntCreator）。

FsPubLogic.dll 导出 5 个（**Callee 逻辑模块模式**，非 Fx 实体）：
`FxModule_GetType/FxModule_Init/FxPublicLogic_CreateCallee/
FxPublicLogic_DeleteCallee/FxPublicLogic_GetVersion`；GamePublic::Start 经
LogicLoader 装载后按名取 Callee（`FxPublicLogic_GetCallee` 字串出现在
GetProcAddress 面），版本门对不上报 "Public logic module version not match !"。

FsSoloPublic.map 的 obj 布局（同源定案，类名前缀 C 去 Solo 后缀即集群类）：

| obj（FsSoloPublic.map） | 职责 |
|---|---|
| SoloPublic.obj（104 符号） | 服务类本体：Start/Stop/Execute(=DoWait)/属性面 |
| SoloPublicWorldMsg.obj（14） | world 消息处理器 |
| SoloPublicState.obj（10） | 状态机 |
| SoloPublicStoreMsg.obj（10） | store 持久化面 |
| PubData.obj（77）/ PubDataImpl.obj（74） | CPubData 权威数据对象 + 属性/记录操作实现 |
| PubKernel.obj（60） | CPubKernel：空间/定时器/请求转发枢纽 |
| PubSpace.obj（19） | CPubSpace：命名数据空间 |
| RecData.obj（55）/ PublicRecord.obj（35） | CRecData 记录存储 + CPublicRecord 视图 |
| PropData.obj（20） | CPropData 类型化属性值 |
| PubTimer.obj（15） | CPubTimer 定时器 |
| PubPlayers.obj（12）/ PubScenes.obj（12） | member 侧玩家/场景注册镜像（public 视角） |
| Reactor.obj / InnerHook.obj / EchoHook.obj / AppRecord.obj / LogShareMem.obj / MemPool.obj | 网络线程泵与基建（fscomm 已覆盖） |
| NameUid.obj（26）/ Relation.obj（28）/ Trade.obj（16） | 名字→UID 映射 / 关系表 / 交易（FsPublic.pdb 无此类，solo 特有或裁剪） |

## 3. main.lua 装配序列（集群形态）

```
LogPool(log\public 前缀, TraceMax=256, EveryDayAlone=false) → 读 GamePublic.ini
→ LogicLoader.Load(work_path, PublicLogicDll)   // FsPubLogic.dll
→ EchoSock(echo_addr:echo_port) ── bind 脚本 echo.lua（echo_init）
→ BroadSock(Bind=inner_addr, 广播=broad_addr:broad_port)
→ AppManager("public_<server_id>", AppAddress=echo 地址)
→ Adapter(inner_addr:inner_port=16003, ConnectMax=64, WorkThreads=1,
          InBufLen=OutBufLen=1MiB)
→ GamePublic 实体：DistrictId/ServerId/Console/EchoSock/BroadSock/
   AppManager/Adapter/LogicLoader 属性注入
   ├ AddPubSpace("pubdata", "nx_pubdata")
   └ AddPubSpace("guild",   "nx_guilds")
→ AppManager:Start() → SetAppState(WAPS_STARTING=1)
→ Adapter:Start() → BroadSock:Start() → EchoSock:Start() → GamePublic:Start()
→ console 四连（App name / Echo / Inner / Broadcast）
→ "Game public server <S> started" → SetAppState(WAPS_OPENED=4)
```

关闭序（close()）：BroadSock → Adapter → EchoSock → GamePublic →
AppManager → LogPool（广播先停、监控最后摘牌——与 member 全同构）。

echo.lua 维护面与 member 完全同族（quit/end/未定义命令 +
getp/setp/getc/setc/method/listp/listc/listm/listg/function/debugger 十二回调），
欢迎语 **"Welcome public server \<S\>"**。

（§4 起为反汇编定案：消息面/状态机/数据层/线协议——见后续小节。）

## 4. GamePublic 对象模型（PDB 符号 + 反汇编字段图定案）

```
GamePublic 布局（this+）：
 +0x00 vptr                    +0x24 State（0..4）
 +0x28 DistrictId              +0x2C ServerId
 +0x30 SavePubDataBatchSize（缺省 50，夹紧 1..1000，Lua 属性可写）
 +0x3C BroadSock               +0x44 Adapter
 +0x48 LogicLoader（Start 校验非空） +0x4C Callee（pub_callee*）
 +0x50 CThread（WorkerProc: while{DoWait;Sleep(1)}）
 +0x54/+0x58/+0x5C/+0x60 四个 CMsgDeque2（连接事件/world/store/member）
 +0x64 CTickTimer              +0x68 CInterConnect（SetCommIdent(1)）
 +0x6C CPubKernel              +0x70 CPubTimer
 +0x74 vector<CPubSpace*>      +0x88 TStringPod（空间名→下标）
 +0x8C 已装载空间计数           +0x90/+0x94 Closing 批量回存双游标
```

**线程模型**：Start 只建 1 个线程（WorkerProc 循环 DoWait+Sleep(1ms)）；
DoWait 是单帧：固定顺序泵四队列（IsEmpty→SwapMsgVector→循环 Peek→
handler），随后 `GetElapseMillisec(1000)>=1000` 时 `PublicStateMachine(0,
&elapsed,4)` + `BroadcastServerInfo()`（全 DLL 唯一广播调用点——每秒一包，
无状态门，Closed 后仍广播）。

**ident 命名空间**（与 member 侧全局一致；SetAdapterID 尾部
SetCommIdent(1) 字节级实证）：

| ident | 含义 |
|---|---|
| 0 | world |
| **1** | **public 自身** |
| 4..7 | store（4+i，MAX_STORE_NUM=4；加载协议只用 store#0） |
| 0x18..0x27 | member（0x18+i，MAX_MEMBER_NUM=16） |

内网帧 = fscomm Adapter u32 长度前缀分帧之上的 `{src u32, dst u32,
type u32}` 12B 头 + payload；注册帧 type=0xFFFF（OnConnected 主动发
`{1,0,0xFFFF}` 通告己方 ident；收向绑定连接↔ident）。public 发帧 dst 恒 0
（路由按连接绑定，不按 dst）。

**CAdapterHook::ReceiveMsg(0x1efa0)**：type==0xFFFF→注册；
src∈0x18..0x27→PutMemberMessage；src∈4..7→PutStoreMessage；
src==0→PutWorldMessage；否则 "(CAdapterHook::ReceiveMsg)message source
error"。OnConnected 查本连接绑定 ident：==4（Storage）→PutMessage(0,1)
→InnerStoreConnected；==0（World）→PutMessage(0,0)→InnerWorldConnected
（**空操作**）。

**重连驱动**：无定时器。CBroadSockHook::OnReceive 收 UDP（magic 0x300；
kind 0=world len 0x78、kind 2=store len 0x28；DistrictId/ServerId 匹配
本服才处理）→ CheckWorld/StoreConnect 幂等补连。

## 5. 状态机（gamepublicstate.cpp，5 态）

常量（PublicStateMachine 0x2d1f0 跳表 + GetStateName 字符串 switch 双证）：

| 值 | 态 | 语义 |
|---|---|---|
| 0 | Unknown | 启动中 |
| 1 | LoadSpaces | 向 store 拉取空间数据 |
| 2 | Opened | 开放服务（MemberRegister 的门 `[this+0x24]==2`） |
| 3 | Closing | 批量回存（world 关闭令后） |
| 4 | Closed | 终态（tick 空转，仍广播） |

事件（arg1）：0=每秒 tick；1=store 连接；2=全部空间装载完；3=world 关闭令。

```
Unknown ──(store连)──> LoadSpaces ──(全空间装完)──> Opened
                                              │ (world 关闭令)
                                              v
Closed <──(回存毕: SendToWorld(2,{1}))──── Closing（每秒 tick 分批回存，
                                           批大小 SavePubDataBatchSize，
                                           筛 GetSave()&&GetModify()）
```

- **Unknown(event1)**：对第一个空间发 `SendToStorage(0, 0x1E,
  {space[0x21],table[0x21]}, 0x42)`，成功即 +0x8C=0、State=1（其余空间
  靠应答链）；失败 trace "(GamePublic::PublicStateUnknown)send error"
- **LoadSpaces(event2)**：组空间清单 `{int count; per space {str space,
  str table, int dataCount}}` → SendToAllMember → State=2
- **Opened(event0)**：`Callee->OnFrameExecute(kernel, elapsed)` +
  `CPubTimer->Run(elapsed)`；(event3) `Callee->OnServerClose` → Closing
- **Closing(event0)**：游标 {空间,数据} 扫描，GetSave()&&GetModify() 的
  CPubData 调 InnerSavePubData（type 0x20），每帧至多 BatchSize 条；
  全空间遍历完 → `SendToWorld(2,{u32 1},4)` → Closed

## 6. member 数据面（gamepublicmembermsg.cpp）

**方向澄清（关键）**：member 侧镜像 14 型（§9 fsmember）是 **public→member
推送方向**；member→public 只有 3 个外层子类型（ProcessMemberMsg cmp 链）：

| 子类型 | 处理器 | 请求线格式 |
|---|---|---|
| 0 | MemberRegister | `public_forw_register_t`（8B，len==断言；内容未读） |
| 1 | MemberData | `public_forw_data_t`（0x29B）={?, u32 index@4, char space[0x21]@8} |
| 2 | MemberMessage | `{int msg_id@0; int argc@4; 标签值列@8}`（get_vary_message） |

- **MemberRegister**：State!=2 静默 return 0；回包（定向
  SendToMember(ident,0,**0**)）= 空间清单（见 §5 LoadSpaces 同构）
- **MemberData**：回包（定向，type **1**）= 0x90 头 {u32 count@0, u32
  index@4, char space[0x21]@8, wchar name[0x21]@0x29, char key[0x21]@0x6b,
  u32 1@0x8c} + `CPubData::StoreArchive(arc, 0)` 全量快照
- **MemberMessage**：非分派器——转发 `Callee->OnPublicMessage(kernel,
  ident, msg_id, varlist)`
- **LoadPlayer**：与数据面无关，组 0x67 包 {char account[0x21], wchar
  name[0x21], u32 fromIdent} → SendToWorld(0, **0**)

**public→member 推送 op 总表**（除 0/1/0xD 定向外全部经
SyncPublic→SendToAllMember 广播全体 member，含请求者）：

公共推送头（0x84B）：`{char space[0x21]; wchar name[0x21]@0x21;
char key[0x21]@0x63}`。

| op | 发送方 | 尾部追加 |
|---|---|---|
| 0x0 | MemberRegister 应答 | `{int 空间数; per {str, str, int}}`（无公共头） |
| 0x1 | MemberData 应答 | 0x90 专用头 + blob(flag=0) |
| 0x2 | CPubSpace::AddPubData | 公共头 + blob(flag=0)；持久化另发 store 0x21 |
| 0x3 | RemovePubData | 头截 0x63（space+name，无 key） |
| 0x4 | CPubData::SendAddAttr | +int type@0x84 + 单值；**AddAttr 与全部 SetAttr\* 复用（upsert）** |
| 0x5 | RemoveAttr | 公共头 |
| 0x6 | AddRecord | +int cols@0x84 + cols×int 列类型 |
| 0x7 | RemoveRecord | 公共头 |
| 0x8 | SendAddRecordRow | +int row@0x84 + 全列值 |
| 0x9 | SendRemoveRecordRow | +int row@0x84 |
| 0xA | SendClearRecordRow | 公共头 |
| 0xB | SendSetRecordGrid | +int row@0x84 +int col@0x88 +byte coltype@0x8C + 单值（SetRec\* 与 CPublicRecord::Set\* 复用） |
| 0xC | SendSetRecordRow | +int row@0x84 + 全列值 |
| 0xD | SendPublicMessage（定向） | `{int msg_id, int argc}` + put_vary_message 值列 |

**取数闭环**：Register 拿清单 → member 按 (space,index) 逐条 MemberData
拉全量快照 → 之后靠广播增量维护镜像。

## 7. store 持久化面（gamepublicstoremsg.cpp）

**装载链**（LoadSpaces 态，只用 store#0/ident 4）：

```
public──0x1E {space[33], table[33]} 0x42B──>store
public<──0x13 {result, space[33], table[33], count, wchar name[33]×count} ≥0x4A
public──0x1F {count, index, space[33], table[33], wname[33]} 0x8CB（链式）
public<──0x14 {result, index@8, space@0xC, table@0x2D, wname@0x4E,
               uid[33]@0x90, PUB4 blob@0xB1} ≥0xB1
       └─ Add 数据对象 + SetUid + LoadArchive(blob) → index+1 取下一条；
          空间完（OnPubSpaceLoaded）→ 下一空间 0x1E；全齐 → event2
```

**保存面**（Closing 态分批 + Add/Remove 即时）：

| 方向 | op | 线格式 |
|---|---|---|
| →store | 0x20 PubSave | {space[33], table[33], wname[33](0x42), blob@0x84}（blob=flag1 saveOnly 快照） |
| →store | 0x21 PubAdd | 0xA5 头（上者 + char uid[33]@0x84）+ blob |
| →store | 0x22 PubRemove | {space[33], table[33], wname[33]} 0x84B |
| ←store | 0x15/0x16/0x17 ACK | {result, space[33], table[33], wname[33](0x42)} ==0x88B |

**辅助面**：StoreRoleOffline（←0x1 {result, count, 行 (str,wstr,double,
str)×} → Callee OnRoleOffline）；CPubKernel::RequestRoleOffline →
`{ServerId, roleID, Y,M,D,h,m,s, flag}` 0x24B SendToAllStore type **1**；
SaveLog → type 0xD {wtitle[33], wname[0x100], wtext[0x100]} 0x44AB（禁
`'`）；CustomLog → type 0x11 {file[32], u32, varlist blob}。

对照 member 侧 store 4 型（0x9..0xC Chunk/nx_chunks）：**两套独立 type
空间**——public 存整块 PubData（nx_pubdata/nx_guilds 表），member 存
玩家 chunk。

## 8. world 面（gamepublicworldmsg.cpp）

| 方向 | type | 线格式 |
|---|---|---|
| ←world | 0 WorldServerState | `public_res_server_state_t` ==0x24B {char[16]@0, 5×int@0x10} → Callee OnServerState |
| ←world | 1 WorldClosePublic | →状态机 event3 |
| →world | 0 LoadPlayer | {account[33], wname[33], fromIdent} 0x67B |
| →world | 1 RequestServerState | 1B（CPubKernel::RequestServerState） |
| →world | 2 CloseDone | {u32 1}（Closing 收尾） |

SyncPublic（逻辑层广播 API）：State==2 才发，SendToAllMember(0, op,
data, len)。

## 9. 数据层线格式（utils/{storearchive,loadarchive}.h）

**CStoreArchive/CLoadArchive**：全定长小端，无 varint。

| 方法 | 编码 |
|---|---|
| WriteInt/ReadInt | 4B LE |
| WriteInt64 | 8B LE |
| WriteFloat | 4B IEEE754 |
| WriteDouble | 8B |
| WriteByte | 1B |
| WriteObject | 8B（PERSISTID{lo,hi} 原样） |
| WriteString | **int32 bytelen + 字节（含 NUL，len=strlen+1）**，GBK 原始字节 |
| WriteWideStr | int32 bytelen（=(码元+1)×2）+ UTF-16LE + 终结码元 |

- 越界/越写 throw "Out range"（容量上限 16MiB）；ReadString 校验末字节
  NUL（"String no end symbol"）；官方 ReadWideStr 的 NUL 校验按码元语义
  错位（原生 bug），Go 侧统一字节语义。
- **VType 枚举**（is_valid_data_type 合法域 [2,8]，与 proto.Var* 线路码
  同基）：2=int、3=int64、4=float、5=double、6=string、7=widestr、8=object。

**CPubData blob（PUB4）**（StoreArchive/LoadArchive；flag=0 全量（member
快照）/1 只含 save 标记项（store 持久化））：

```
int32 magic=0x50554234("PUB4"); int32 attrCount(占位); int32 recCount(占位)
属性区: { string name; byte type; <值按型> }*
记录区: { string name; int32 rowMax; int32 rows; int32 colCount; int32 keyCount;
          byte colTypes[colCount]; {int32 col; byte bEnabled}[keyCount];
          行主序网格 rows×colCount 按 colTypes 编码 }*
末尾 Seek 回填两计数
```

LoadArchive 校验序：magic → colCount∈[1,0x100] → keyCount≤colCount →
列型逐个合法 → 键列型必须 6/7 → 已存在记录要求 rowMax/colCount/keyCount/
colTypes 全等，否则 LoadRecDiscard 按列型跳过整块。

**类结构**：CPropData = word type + 8B union；CRecData = {IRecordAccess\*
回调对象, flags 位图, rowMax, colMax, colTypes[], 行向量(dword 网格),
keys→TStringPod 索引}；CPubSpace = {kernel, name, table, datas, freeUids,
nameIndex}，vtable `[1]GetSpaceName [2]GetTableName [3]GetCanModify(恒1)
[4]FindPubData [5]GetPubDataCount [7]GetPubData [8]AddPubData
[9]RemovePubData`；CPubData vtable `[1]GetName [2]GetUid [3]GetSave
[4]GetCanModify`。

**CPubTimer**：beat_data_t{name, **回调函数指针**, sortKey/间隔, 累计,
funcName, TSharePtr\<IVarList\> data, 上次时刻, deleted}；Execute 调
`cb(kernel, funcName, elapsed, data)`，超 100ms CORE_TRACE
"Timer running exceed %dms"；驱动 = Opened 态每秒 Run(elapsed)。
AddTimer 断言 "timer already exists"/"time > 0"。

## 10. FsPubLogic.dll（Callee 逻辑模块）

导出 5 个（**Callee 模式**）：`FxModule_GetType→"FxGameLogic"`、
`FxModule_Init`、`FxPublicLogic_GetVersion`、`FxPublicLogic_CreateCallee
→new pub_callee`、`FxPublicLogic_DeleteCallee`。FsPublic 侧另找可选导出
`FxPublicLogic_GetCallee`（本 DLL 无，回退 CreateCallee）。

**⚠ 快照版本门断裂（本快照实证）**：GamePublic::Start 校验
`GetVersion()==0x3001`（Debug/Release 双 FsPublic.dll 同门 0x817d4/
text+0x237d7），而快照内 FsPubLogic.dll Debug=**0x300E**（map 时间戳
2013-04-17）、Release=**0x300F**——**实装装载即 "Public logic module
version not match !" Fatal**（快照逻辑模块滞后宿主一年；正式农场应配
0x3001 版）。Go 复刻：门保留 0x3001 常量，内置逻辑模块自报 0x3001。

**pub_callee 是纯转发壳**：ctor `new DomainManager + new GuildManager`
塞全局容器；空间路由 strcmp(space, mgr->GetSpaceName())——
DomainManager→"domain"、GuildManager→"guild"。（注意 lua 集群建的是
"pubdata"/"nx_pubdata" 而 DLL 自报 "domain"——2013/2014 版本差的又一
实证；Go 复刻双名兼容。）

**两级协议**：
- IPubManager（0x103E50A0）：GetSpaceName / OnPublicStart / OnServerClose /
  OnPubSpaceLoaded / OnPubDataLoaded / OnPublicMessage
- CDomainBase（0x103E51D4）：GetDomainName(kernel)→wstring / GetSave() /
  OnCreate(kernel,space) / OnLoad(kernel,space) /
  OnMessage(kernel,space,msgid,int,IVarList&)（后四纯虚）

**公共消息分派**：msgid=IVarList::GetInt(2)；**1000..1999 走
ProcessCommonMessage**（对应 member 推送 op 0x3..0xC 的修改原语）：
1000=SetPubDataProp、1001=IncPubDataProp、1100=ClearPubDataRec、
1101=RemovePubDataRecRow、1102=AddPubDataRecRowValue、1103=
SetPubDataRecValue、1104=IncPubDataRecValue、1105..1107=\*ByKey；
其余按 args[1]（数据名=域名）查域注册表。

**IPubKernel 观测槽**：+0x10 FindPubData、+0x20 AddTimer(回调,"domain",
域名,间隔ms,0)（60s=0xEA60）、+0x30 SendPublicMessage(space,数据名,
msgid,varlist)。

**域清单（22 个，OnPublicStart 时 new）**：

| 域 | msg 区间 | 表依赖（Res\ini） | 备注 |
|---|---|---|---|
| Domain_PublicData | 0x55-59 | — | 锁/掉落修正/核心物品（Server_Id_Record、Core_Item_Control_Record、DROPTIMEMANAGEREC） |
| Domain_Team | 0x18-38 | team_main_rec、team_%d_member_rec | 大团/小队/招募；0x8024B 定时 |
| Domain_ChatRoom | 0x2EE-2FB | chatroom_rec(+%d_member) | GetSave=false |
| Domain_CloneScene | 0x47-56 | scene\{clone_scene,FreshmanSceneConfig}.xml | 3 个 HB 定时器 |
| Domain_CampCollection | 0xC8-D4 | Life\{AreaProp,ResourceInfo}.xml | camp_collection_\*_rec |
| Domain_Arena+ArenaQueue | 0x0-F | Arena\arena_rule.xml | ArenaQueueRefresh 定时；arena_\* 记录族 |
| Domain_ArenaRank | 0x1-7 | Arena\arena_rank_{rec,refresh}.xml | |
| Domain_WorldRank | 0x1-4+ | WorldRank\rank_{prop,refresh,reward}.xml | Refresh/Save 双定时 |
| CDomain_BattleWar+Queue | 0x0-14 | War\battlewar_{config,limit}.xml | battle_queue/fight |
| Domain_EnergyWell | 0x4-18 | EnergyWell\ 下 8 个 xml | 99 符号最大域；HB_EnergyWell |
| Domain_Ranking_%d（ResFight） | 0x0-9 | EWGW 系 | 能量井资源战（注册名带 ServerID） |
| Domain_EnergyWellGuildWar | 0x0-7 | 5 个 EWGW\*.xml | 能量井公会战 |
| Domain_CampWar | — | — | 阵营战 |
| Domain_PvpTask/PvpActivity | 0x0-6/1-2 | PVPTask\{PVPConfig,PVPActivity}.xml | |
| Domain_Achievement | 0x60 | — | |
| Domain_Switch | 0xCB-CC | rule\SwitchManager.xml | |
| Domain_WebExchange_%d | 0xF0 | — | |
| Domain_CapitalMonitor | 0x62 | — | |
| DropItemManage | 0x61 | Item\DropItemManage.xml | |
| EventSpring | 0x3E | — | |
| Domain_Channel | 0xA-B | — | |
| SceneList | 1-2 | — | |

**GuildLogic（GuildManager）**：msg 0x64=建会，0x65..0x9C（101..156）
约 50 操作（成员/职位/仓库/解散/关系/资本）；guild 空间 =
guild_common_pubdata + 每公会一个以公会 int64 ID 命名的 PubData；
3 个 ini\Guild\*.xml；20+ guild_\* 记录。

**工具层**：util_func（随机/时间/编码 GBK↔wide↔UTF8/几何/GUID）、
extend_func（createclass_from_xml、save/load_to_{string,xml}，与 FsGame
同源）、XmlFile（tinyxml+rapidxml 快速路径，GB2312）；assert → assert.log。

## 11. 广播包（BroadcastServerInfo）

0x24B UDP：kind=**4**@0、magic=0x300@4、ip[16]@8、port@0x18、
DistrictId@0x1C、ServerId@0x20（member 是 kind=3、0x28B、ServerId@0x20/
MemberId@0x24——**两包布局不同**）；经 BroadSock vptr+0x48 发 15000。
world 周期包 kind=0（0x78B）、store kind=2（0x28B）——public 收听后幂等
补连（DistrictId/ServerId 匹配门）。

## 12. solo 形态差异（FsSoloPublic.map 定案）

- 状态机 8 态：Unknown→LoadSpaces→LoadRelations→LoadFeeds→
  LoadTrades→Opened→Closing→Closed（多 3 个 solo 特有 store 装载态）
- world 面 16 处理器（member_forw_scene/player/role_\*：world 转发注册表）
  ——**集群 FsPublic.dll 无**（member 直连注册）
- store 面多 StoreRelationInfo/StoreFeedInfo/StoreTradeInfo
- CPubKernel 带 CRelation/CNameUid/CTrade/Feed 系 API（集群裁剪）
- 属性面：UseThread/UseQuickLog/UseAppRecord/ExitForceSave/
  SaveMinutes(20)/MaxRunningTime/Inner\*/Outer\*/Broad\*/Echo\*/AppName/
  LogLauncher；InnerPort 缺省 0=不监听；CInnerHook（集群为 CInterConnect）
- 空间注册："domain"/"nx_domains" + "guild"/"nx_guilds"

## 13. Go 复刻验收清单

- [x] ini 方言（GamePublic.ini：Inner 16003/Echo 15003 固定缺省）
- [x] 归档编解码（CStoreArchive/CLoadArchive 字节级 + PUB4 blob 往返）
- [x] 数据层（VType/PubData/Record/PubSpace 全 API + 官方错误串）
- [x] 线协议（ident/三面消息表/推送 op 头布局/定长 NUL 填充区）
- [x] 服务本体（5 态状态机/四队列/装载链/保存面/广播包 kind=4）
- [x] Callee 协议（DefaultCallee：1000..1107 内建修改原语 + 域注册表骨架）
- [x] 装配（main.lua 序 + DoWait 泵 + echo 维护面 "Welcome public server"）
- [x] 测试（golden/往返/真 TCP 装载链/广播包/状态机断言串）

