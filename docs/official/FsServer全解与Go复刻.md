# FsServer.dll 全解与 Go 复刻

> 官方快照：`BGserverCJ/Debug/FsServer.dll`（962KB）+ `FsServer.pdb`（4.2MB 全符号）
> 分析日期：2026-11-19（工具：tmp_research/fsserver/fs_dis.py capstone+PDB 节重定位 + digest.py 函数摘要器）
> Go 复刻：`internal/fsserver` + `cmd/world`

---

## 一、模块定位

FsServer.dll 是**世界服逻辑模块**（flexis 源码树 `d:\flexis\client\fsserver\`）：
GameWorld.bat → FsLoader.exe GameWorld.ini 加载
`FsKit.dll + FsComm.dll + FsServer.dll`，MainEntity=MainConsole，
组合根脚本 `..\Lua\GameWorld\main.lua`（装配序权威）。GameUnite.ini（unite 单进程
形态）亦包含本模块。ident 命名空间中 **world=0** 的对端实体。

职责：登录链世界段（账号校验/排队/选角/建角/删角）、玩家 14 态状态机、
场景分配与 4 态状态机、世界 13 态开关服状态机、跨服消息中枢
（entry/member/store/public/charge/gmcc/stat/manage/extra 九面）、
矩阵卡密保（CMatrixCode）、区服信息 UDP 广播。

### 实体面（Lua 可见）

GameWorldEntCreator → "GameWorld" 实体（IEntity 子类，flexis `fsserver\gameworld.cpp`）：

- 属性：DistrictId/ServerId/ServerName(wstr)/MaxPlayers(默认 8192，ctor 内置 0x800=2048)/
  SaveMinutes(10)/AccountRoles(1)/RoleHold/HoldRate(2)/AutoOpen/PlayerQueue/
  OnlineContinue/OfflineContinue/PasswordAnyChar/OriginalPassword/
  ClearPlayerBatchSize(100)/ClearSceneBatchSize(10)/PlayerCount/OnlineCount/OfflineCount/
  Console/EchoSock/BroadSock/AppManager/Adapter/ChargeSock/GmccSock/StatSock/ManageSock/ExtraSock
- 方法：Start/Stop/OpenServer/CloseServer/LoadNameFilter/LoadPlayer/AddScene/
  AddPrototypeScene/DumpPlayers/DumpScenes

### 装配序（main.lua 定案）

LogPool(log\world) → IniDocument(GameWorld.ini) → EchoSock(15001，echo.lua 维护面) →
BroadSock(bind 内网, 广播 127.255.255.255:15000) → AppManager("world_<ServerID>") →
Adapter(16001, ConnectMax=64, WorkThreads=1, 1MiB 双缓冲) → GameWorld 服务 →
LoadNameFilter(Res\filter.txt) → load_scene_config（Res\scenes.ini → AddScene/
AddPrototypeScene，键 Member/Script/Config/Clonable）→ 可选 UserSock charge/gmcc/stat
（Async 客户端）+ MassSock manage → Start 序：app_manager → adapter → broad_sock →
echo_sock → game_world → SetAppState(OPENED)。

## 二、内核结构

### 2.1 GameWorld 对象（ctor 0x10079CA0）

IEntity 基座 + wstring ServerName(+0x24) + 配置区：
+0x44 DistrictId、+0x48 ServerId、+0x4C MaxPlayers、+0x50 SaveMinutes、
+0x54 RoleHold、+0x58 HoldRate、+0x5C ClearPlayerBatchSize、+0x60 ClearSceneBatchSize、
+0x74 WorldState。组合根：CThread（Start/Stop/ServerThreadProc→DoWait）+
**6× CMsgDeque2**（inner/store/entry/member/public/charge 六队列）+ CTickTimer +
CInterConnect(+0x138) + CWorldPlayers(+0x13C) + CWorldScenes(+0x140) +
CNameUid(角色名注册) + CMatrixCode + CNameFilter + sock 指针区
（BroadSock +0xBC / Adapter +0xC4 / ChargeSock +0xC8 / GmccSock +0xCC / … /
ExtraSock[16] +0xD8 起）。

### 2.2 DoWait 主泵（0x1007D9A0）

1. 六队列依次 Swap+Peek 排空：inner → store → entry → member → public → charge
2. CheckQueue（登录排队放行）
3. 每 1000ms tick：CMatrixCode::IncMatrixPos（全局计数 >0x1E 门）→
   charge "keep" 心跳（CChargeSockHook::SendIdentMsg）→ gmcc SendMsg(0xFFFF,"") →
   stat 空串保活 → extra[16] 空串保活 → WorldStateMachine(0,0,0) →
   BroadcastServerInfo()

### 2.3 CInterConnect（内网连接注册表，0x1007AA90）

对象={+0 Adapter 指针, +4 自身 ident=0}。ident 纯算术：
`GetMemberCommIdent(idx)=0x18+idx`、`GetEntryCommIdent(idx)=8+idx`、
`GetStoreCommIdent(idx)=4+idx`、`GetPublicCommIdent()=1`、`GetAdapterCommIdent()=0`。
连接注册：Adapter 0xFFFF 特例帧 → SetAdapterCommIdent/SetAdapterCommIndex 绑定
连接 context ↔ 源 ident；断连 OnDisconnect → SetAdapterDisconnect 解绑。

**SendMsg(commIdent, dst, type, data, len)**（0x10084300）：构造 12B 头
`{src=world(0), dst, type}` + payload，经 Adapter::Send 二段写（头 12B + 体）。
SendMsg2 = 无连接时 trace 口径不同的孪生。上限：MAX_MEMBER_NUM=0x10（16）、
MAX_ENTRY_NUM、MAX_STORE_NUM=4（ident 4..7）、EXTRA_SERVER_NUM=16。

发送族：SendToMember/SendToMember2/SendToAllMember(2)、SendToEntry(2)/SendToAllEntry、
SendToStore(2)/SendToAllStore(2)、SendToPublic。**SendToScene(sceneId, dst, type, …)**
（0x10097C90）= GetScene → 场景须 Running(+0x10==2) → SendToMember(scene.memberIdx, dst, …)。

### 2.4 CAdapterHook（受面，0x100714B0 ReceiveMsg）

与 member 版同构：len<0xC 断言 "message length error"；type==0xFFFF → 注册帧；
源校验四分支 → `0x18..0x27→PutMemberMessage`、`8..0x17→PutEntryMessage`、
`4..7→PutStoreMessage`、`1→PutPublicMessage`，其余 "message source error" 丢弃。
OnAccept 仅 trace（"World adapter %s:%d accept, total %d"）。

## 三、世界状态机（13 态，state@+0x74，gameworldstate.cpp）

| # | 态 | 行为（digest 实证） |
|---|---|---|
| 0 | Unknown | SendToStorage(type 0, 8B) 角色名全量请求 → LoadRoles |
| 1 | LoadRoles | SyncAllRoleInfo（收 store RoleInfo 后推进）→ LoadScenes |
| 2 | LoadScenes | CheckAllScene + SendOpenConnect（"Game server ready"）→ CanOpen |
| 3 | CanOpen | SendOpenConnect + CheckAllScene → Opened |
| 4 | Opened | 稳态：CheckAllScene + CheckAllPlayer（每 tick 维护） |
| 5 | CloseEntry | SendCloseEntry 轮询全部 entry → "Entry server close all" → ClearPlayers |
| 6 | ClearPlayers | 遍历玩家表逐批（ClearPlayerBatchSize=100）驱动 PlayerStateMachine → ClearScenes |
| 7 | ClearScenes | 遍历 clone 场景 SendToMember + SendCloseMember → "Member server close all" |
| 8 | CloseMember | SendCloseMember → "Member server close succeed" → ClosePublic |
| 9 | ClosePublic | SendCloseStore（public 无独立应答，直接推进） |
| 10 | CloseStore | SendCloseStore → "Store server close succeed" → **"Game server close succeed"** → Closed |
| 11 | Closed | 终态（空处理） |
| 12 | Error | 终态 |

SendOpenConnect = SendToAllEntry(entry_forw **type 0 init_connect**，4B 旗标
worldState==4?1:0)；SendCloseEntry/Member/Public/Store = 对应 close 消息轮询。

## 四、玩家状态机（14 态，state@player+0xC，gameworldplayer.cpp）

| # | 态 | 驱动事件/行为要点 |
|---|---|---|
| 0 | Unknown | 登录主态：op 消息解析（事件 5=login）→ 名字过滤 → 重复登录/满员检查（OnlineCount≥MaxPlayers → 客户端 {0x03, u32 0x4E92}；重复登录 {0x03, 0x4E91}）→ SendPlayerLogon/Validate（md5(password+matrixStr)）→ Logging/Validate；矩阵卡挑战（GetMatrixString/GetPicture → SendToClient） |
| 1 | WaitBreak | BreakClient + ErasePlayer（顶号善后） |
| 2 | Logging | 排队判定（PlayerQueue 开 → GetQueueCount/AddToQueue → Queuing）或 SendToStorage 账号 logon |
| 3 | Validate | 矩阵卡校验等待（应答 → Logged/Getting） |
| 4 | Queuing | 队列位置通知（GetInQueuePos；"queue error"）→ Getting |
| 5 | Getting | SendPlayerRoles（0x04 角色列表）等待选角 |
| 6 | Logged | 选角/建角/删角处理：名字长度/合法词过滤（CNameFilter::Filter/HasInvalidCharW）、"scene busy"/"not found scene"/"no any scene"/"world info too long" 校验 → SendToScene2 进场 |
| 7 | Choosing | 载入角色数据（"Load player data failed"/"player data too big"/"player already in game"）→ SendToScene2 + SendPlayerSetOnline + IncScenePlayers + SendLoginToGmcc |
| 8 | Creating | 建角：SendToAllStorage2(type 0xA) → store 应答 → SendPlayerRoles |
| 9 | Deleting | 删角：store 应答 → SendPlayerRoles |
| 10 | Online | 稳态：客户端帧（put_vary_message 序列化）SendToScene 透传；"renew warning/renew break"（计时续费）→ Saving/Leaving |
| 11 | Offline | 存档落库：SendPlayerSetOffline + SetPlayerLeaveGame + SendPlayerRemove + SavePlayer + FlushPlayer |
| 12 | Saving | 存档等待（"save player timeout"） |
| 13 | Leaving | 踢线善后（SendToStorage type 0xC 45B + "save player timeout"） |

**PlayerStateMachine(player, msgID, data, len)**（0x1008D740）：msgID 事件号
（ClientMsg* 投入）：4=GetVerify、5=Login、6=WorldInfo、7=ChooseRole、8=CreateRole、
9=DeleteRole；0=外部驱动（tick/断线）。事件按当前态路由到 14 个 PlayerState* 处理器。

### 4.1 player_t（CWorldPlayers::AddPlayer 0x100A2060）

0x170B：+0x00 connect id、+0x04 serial、+0x0C state、+0x3C entryIdx、
+0x40/+0x48/+0x50 f64（登录/活跃/存档时刻）、+0x15C char[0x10] account。
PERSISTID={connect,serial} 哈希索引（+0x2C）+ 空闲槽复用栈（+0x10）。
玩家名区（SetupPlayerName/CleanPlayerName）与角色名注册表联动。

### 4.2 客户端消息面（经 entry 转发）

entry_back_client_message_t = `{ident u32, serial u32}` 8B 头 + 原始客户端帧；
帧首字节 opcode 分派（0x10084640 跳表）：0=GetVerify、2=Login、3=WorldInfo、
4=ChooseRole、5=CreateRole、6=DeleteRole、1/其他=忽略（return 1）。

**SendToClient**（0x10097820）= SendToEntry2(entryIdx, entry_forw **type 7
send_to_client**, {ident,serial}8B+帧)。BreakClient=type 1、ShakeClient=type 10、
SetClientInuse=type 2、SetClientOnline=type 3（§24 forward 表双向闭合）。

**SendPlayerRoles**（0x1008E600，客户端 0x04 方言）：
`[0x04][u32 @1][u32 @5][6×u32 时刻 @9..0x1D][u32 serial @0x21][u32 角色数 @0x25]`
+ 每角色 `{u16 计数=4, tag2 i32, tag2 i32, tag7 wstr 名, tag7 wstr 参数}`
——与本项目 login 0x04（41B 基座+TVarList 追加段）同构，proto.VarList 编码
（u16 计数 + tag 元素）官方同源实证。

### 4.3 矩阵卡（CMatrixCode，matrixcode.cpp）

- ctor：343,000 项 int 数组顺序填 0..9 → srand → 343,000 次随机 SwapValue 洗牌
- GetMaxtrixCode(pos)=matrix[(curPos+pos)%343000]；IncMatrixPos 每 tick 前移
- GetMatrixString：挑战串格式 `"%s%s,%s%s,%s%s"`（列字母 A..G + 行号 1..10 三组）
- GetMatrixPicture：GDI（Times New Roman）渲染 BMP（本地 matrix.bmp，线下发卡用）
- 登录校验：MD5String("%s%s" password, matrixString)（SendPlayerLogon/Validate，
  "%s,%s" 串联变体并存）——挑战应答由客户端按卡面回读

## 五、消息面全景（六张表 + 计费文本协议）

### 5.1 entry 面（ProcessEntryMsg 跳表 0x10084D78，六型）

| type | 结构（断言钉死） | 语义 |
|---|---|---|
| 0 | entry_back_register_t 12B {DistrictId, ServerId, EntryIdx} | entry 注册；校验不符 "entry ident error"；回 entry_forw type 0 init_connect 4B 旗标 |
| 1 | entry_back_client_accept_t 28B {ident, serial, addr char[16], port u32} | 客户端接入 → CWorldPlayers::AddPlayer |
| 2 | entry_back_client_closed_t | 连接关闭 → FindByConnect + PSM 断线事件 |
| 3 | entry_back_client_message_t ≥8B {ident, serial}+帧 | 客户端消息 → opcode 分派（§4.2） |
| 4 | entry_back_client_illegal_t | 非法回报：version not match/serial not match/no encode confirm/message illegal → BreakClient |
| 5 | entry_back_close_entry_t | entry 关闭确认 → WorldStateMachine |

> **§24 纠错**：FsEntry 侧旧载 "0=accept 通知" 有误——实证 accept=type 1、
> type 0 为 register；新增 type 5 close_entry。（本次 FsServer.pdb 断言逐型钉死）

### 5.2 member 面（ProcessMemberMsg 跳表 0x1008BEE4，27 型）

| type | 处理器 | payload 断言 |
|---|---|---|
| 0 | MemberMsgCreateClone | member_back_create_clone_t == |
| 1 | MemberMsgDeleteClone | member_back_delete_clone_t == |
| 2 | MemberMsgCloneScene | member_back_clone_scene_t ==（原型校验三态） |
| 3 | MemberMsgSetDownTime | member_back_set_down_time_t == |
| 4 | MemberMsgSceneState | member_back_scene_state_t ==（场景状态回报→场景状态机） |
| 5 | MemberMsgToScene | member_back_to_scene_t >= |
| 6 | MemberMsgToAllScene | member_back_to_all_scene_t >= |
| 7 | MemberMsgBreakPlayer | member_back_break_player_t == |
| 8 | MemberMsgSwitchScene | member_back_switch_scene_t >=（切场景三校验） |
| 9 | MemberMsgNewRole | member_back_new_role_t >= |
| 10 | MemberMsgWorldInfo | member_back_world_info_t >= |
| 11 | MemberMsgSetDelay | member_back_set_delay_t == |
| 12 | MemberMsgQueryGift | member_back_query_gift_t ==（转发 charge "query_gift"） |
| 13 | MemberMsgGetGift | … get_gift |
| 14 | MemberMsgSellCard | … sell_card |
| 15 | MemberMsgUnsellCard | … unsell_card |
| 16 | MemberMsgBuyCard | … buy_card |
| 17 | MemberMsgBuyItem | … buy_item（含 buy_item_give 变体） |
| 18 | MemberMsgGetPoints | … get_points |
| 19 | MemberMsgItemInfo | … item_info |
| 20 | MemberMsgToCharge | member_back_to_charge_t >=（透传 charge，"custom"） |
| 21 | MemberMsgToGmcc | member_back_to_gmcc_t >= |
| 22 | MemberMsgToStat | member_back_to_stat_t >= |
| 23 | MemberMsgToManage | member_back_to_manage_t >= |
| 24 | MemberMsgToExtra | member_back_to_extra_t >=（nIndex 0..15 断言） |
| 25 | MemberMsgCloseMember | member_back_close_member_t ==（→ WorldStateMachine） |
| 26 | MemberMsgLeaveGame | member_back_leave_game_t == |

### 5.3 store 面（ProcessStoreMsg 跳表 0x1009B040，13 型）

type：0 RoleInfo（角色名全量 → CNameUid 重建 → WorldStateMachine 推进）、
1 Load、2 Save（"player stored/timing stored"）、3 Block、4 Resume、
5 Create（建角应答 → CNameUid::Add + SendToAllMember RoleAdd + "create_role" 统计）、
6 Delete、7 Roles（角色列表应答 → SendPlayerRoles）、8 AccountReset、
9 AccountLogon、10 AccountLogoff、11 AccountValidate、12 CloseStore。
结构名 store_back_*_t 全部断言钉死（gameworldstoremsg.cpp）。

world→store（SendToStorage 族，call-site 常量实证）：type 0 角色名请求(8B)、
2 账号 logon(0x8F=143B)、3 LoadCancel(0x6B)、5 SetOnline(0x6B)、6 SetOffline(0x6B)、
0xA 建角(0x9B，SendToAllStorage2)、0xB 角色列表请求(0x7F)、0xC 载入角色(0x2D=45B)、
0x24 计费 logon(0x72)、0x25 计费 logoff(0x56)、0x26 计费 validate(0x72)。

### 5.4 public 面（ProcessPublicMsg 三型）

public_cmd_load_player_t / public_cmd_server_state_t / public_cmd_close_public_t
（LoadPlayer：public 服代载玩家数据回投 InnerLoadPlayer）。

### 5.5 inner 面（ProcessInnerMsg 跳表 0x10088130，14 型）

0 FromGmcc、1 FromStat、2 FromManage、3 FromExtra、4 StoreConnected、
5 EntryConnected、6 **MemberConnected**（4B {memberIdx} → SyncAllScene +
SyncAllPlayer + SyncAllRoleInfo 三段全量同步）、7 StoreDisconnect、
8 EntryDisconnect、9 MemberDisconnect、10 OpenServer、11 CloseServer、
12 DumpPlayers、13 DumpScenes。

### 5.6 charge 面（文本 VarList 行协议）

CChargeSockHook（UserSock 异步 TCP 客户端）：行 = `$s_%d_%d_%d_%d ` 前缀
（GetSerial 等 4 整）+ text_encode_msg(VarList) + `\r\n`（MakeSendData 手补 0D 0A）。
收侧 text_decode_msg → CVarList → 队列。ProcessChargeMsg strcmp 链 16 命令：
register/unregister/reset/login/logout/renew/query_gift/get_gift/sell_card/
unsell_card/buy_card/buy_item/buy_item_give(同 buy_item)/item_info/get_points/
validate/custom。Keepalive：每秒 "keep"。md5 摘要与矩阵卡联动（§4.3）。

### 5.7 gmcc/stat/manage/extra 面

CGmccSockHook/CStatSockHook（UserSock 客户端，空串保活）、CManageSockHook
（MassSock 服务面：CreateContext/OnBlockIP 等）、CExtraSockHook[16]（自定义扩展
服，SendCustomMsg）。RecvGmccCommand → 场景命令转发（SendToScene type 0x2F）。

## 六、广播与存档原语

### 6.1 BroadcastServerInfo（0x1007D7A0，UDP 0x78=120B）

`[u32 kind=0][u32 magic=0x300][char[16] bind 地址 @8][u32 内网端口 @0x18]
[u32 DistrictId @0x1C][u32 ServerId @0x20][wchar 服务器名 @0x24][… @0x6C]
[u32 PlayerCount @0x70][u32 open 旗标 @0x74]`（open=WorldState==4?1:0）。
经 BroadSock 广播口发送——**与 FsLister :15000 world 120B 建服包逐域互证**
（§25/分布式形态清单），kind=0 即 world 注册心跳。

### 6.2 存档原语（CStoreArchive/CLoadArchive + put/get_vary_message）

WriteByte=1B / WriteBool=1B / WriteWord=2B / WriteInt=4B / WriteInt64=8B /
WriteFloat=f32 / WriteDouble=f64 / WriteString=`[u32 len][bytes]` /
WriteWideStr=`[u32 len][UTF-16LE]` / WriteObject=4B（PERSISTID）。
**put_vary_message**：每元素 `{u8 tag, 值}`，tag：2=int、3=int64、4=float、
5=double、6=string、7=widestr、8=object（无计数前缀；get 侧计数由调用方供给）。
SyncAllScene/Player/RoleInfo 与客户端帧封装共用此原语——存档与线路同一血统。

## 七、Go 复刻（internal/fsserver + cmd/world）

结构对照：

| 官方 | Go |
|---|---|
| CAdapterHook ReceiveMsg 源路由 | innerface.go（fscomm.Adapter 受面，0xFFFF 注册 + 四路队列） |
| 六队列 + DoWait 泵 | world.go（World.Pump：六队列排空 + 1s tick） |
| WorldStateMachine 13 态 | world.go 状态机 + 各 WorldState* 处理器 |
| PlayerStateMachine 14 态 | players.go + playerstate.go（事件 4..9 路由） |
| SceneStateMachine 4 态 | scenes.go（Unknown/Starting/Running/Error） |
| CInterConnect + SendTo* | interconnect.go（ident 算术 + Send 族） |
| CMatrixCode | matrixcode.go（343,000 项 + MSVC rand 洗牌 + 挑战串） |
| CNameUid/CNameFilter | nameuid.go（复用 fsmember.NameFilter 思路） |
| 六消息面 | entryface/storeface/memberface/publicface/innerface/chargeface |
| BroadcastServerInfo 120B | broadcast.go（与 fslister 注册包编码同源对拍） |
| CStoreArchive/vary 原语 | archive.go（put/get VaryMessage + 读写原语） |
| main.lua 装配序 | assemble.go（Assemble/Assembly）+ cmd/world 释放式内置 ini |

范围约定（对齐 §28 粒度）：**结构与线路方言逐型复刻、断言文案逐字保留**；
FsGame 玩法层/store 落库后端/charge 真实对端按接缝留钩子（Store 回应注入、
Chargeface 文本协议完整、TCP 对端可缺省）。

## 八、未尽事项

- 场景级 opcode 表（SendToScene dst 透传后的 scene 消息面，FsGame 域）
- CLoadArchive::ReadString NUL 语义精对拍（Write 侧无 NUL，读侧容错待 oracle）
- manage 面 BlockIP/OnBlockIP 细节（运维件，单服未接线）
- gmcc 命令字典（RecvGmccCommand 分支展开，客服系统域）
