# FsCharge.dll 充值计费中心全解与 Go 复刻

> 对象：`F:\Game\Online\9yin\BGserverCJ\Debug\FsCharge.dll`（PE32，2014-03-22
> 官方 Debug 全符号快照，同目录 FsCharge.pdb 全量 PUB32 符号）。
> Go 复刻：`internal/charge`（服务核心）+ `cmd/charge`（进程入口）。
> 逆向工作底稿（逐函数反汇编记录）：`tmp_research/fscharge/`。

## 一、进程形态与接线

- 启动：`GameCharge.bat` → `FsLoader.exe GameCharge.ini`，加载
  FsKit+FsSql+FsComm+FsCharge 四模块，MainEntity=MainConsole，
  Lua 入口 `Lua/GameCharge/main.lua`（接线全解：EchoSock→127.0.0.1:15008
  维护通道、MassSock→[Outer] 192.168.33.70:6000 游戏服接入、
  AppManager("charge") 进程监控、SqlDBSystem Oracle 双库）。
- 客户端 = **FsServer.dll GameWorld**（游戏世界服）：`CChargeSockHook`
  连 charge:6000；`GameWorld::ProcessChargeMsg` 收应答、
  `GameWorld::ChargeMsg{Reg,Unreg,Reset,Login,Logout,Renew,QueryGift,
  GetGift,SellCard,UnsellCard,BuyCard,BuyItem,RetPoints,ItemInfo,Custom}`
  分发；`MemberMsgToCharge` 把玩家消息以 `custom` 命令透传。
- PDB 段基换算：函数 RVA = PUB32 off + 0x4C000（seg2=.text；
  增量链接 thunk 表实证）。`.textbss` 为空段。

## 二、线协议（MassSock 行文本）

| 方向 | 成帧 |
|---|---|
| 发送 | `text_encode_msg` 编码行 + `\r\n`（MakeSendData@0x78500） |
| 接收 | 逐字节累积 `>=0x20` 可见字符（丢弃 `\r` 等控制字节），遇 `\n` 且缓冲非空成帧；行缓冲 0x1000（OnReceive@0x78910） |

元素编码（text_encode_msg@0x56b50 / text_decode_msg@0x55920）：

| 形态 | 元素类型 | 编码 |
|---|---|---|
| 十进制整数 | int（词长<10）/int64 | `%d` / `%I64d` |
| 含 `.` | double | `%f`（6 位定点） |
| `$` 前缀 | 窄串 | `\xHH`（空格/控制/`%`/0x80+）、`\\`→`\` |
| `#` 前缀 | 宽串 | 逐 UTF-16 码元：可打印原样（`\`双写）、`\xHH`、≥0x100 用 `\uHHHH` |
| `*` 前缀 | userdata | 同窄串转义 |
| 裸词 | 窄串（无前缀兼容） | 同上 |

消息 = IVarList：`[0]=回显串（请求方路由标记，应答原样带回）、
[1]=命令名、[2..]=参数`。connid 由 MassSock 分配（0..0xff，
`server_t*[0x100]` 按 connid 直索引）；msgid=accept 频道号（单监听恒 0，
`(connid,msgid)` 对定位连接）。

## 三、命令分发表（ProcessSockMsg@0x70c60，strcmp vl[1]）

18 命令。下发 msgid（subop）与官方池路由：

| 命令 | subop | 官方池 | 请求字段（vl[2..]） | 应答元素（Publish*） |
|---|---|---|---|---|
| register | sync 0 | Access(Sync) | serial?, serverID(+offset), name, ident(MD5), port | [echo,"register",result] |
| unregister | 1 | Access(Sync) | — | [echo,"unregister",result] |
| reset | — | 本地 | — | 无（清本服用户） |
| login | 2 | AccessHash | ?(2), account(3,UP), passwd(4,UP), ip(5), ?(6) | [echo,"login",1,account,result,accID,logonID,"",points,"",chargeType,deadline(f),gmLevel,ext]（14 元素） |
| logout | 3 | AccessHash | useridx(2), sess(3), reason(4), account(5,UP), passwd(6,UP) | [echo,"logout",useridx,result] |
| renew | 4 | AccessHash | useridx(2), sess(3), reason(4), order(5) | [echo,"renew",accID,chargeType,unitType,deadline(f)]（仅类型2回发） |
| query_gift | 5 | AccessHash | useridx(2) | [echo,"query_gift",accID,N,(id,name,amount)×N] |
| get_gift | 6 | AccessHash | useridx(2), giftID(3), serial(4), amount(5) | [echo,"get_gift",accID,result,giftID,serial,amount] |
| sell_card | 7 | CenterHash | useridx(2), cardType(3), cardNo(4,UP), price(5) | [echo,"sell_card",accID,result,cardNo,price,note] |
| unsell_card | 8 | CenterHash | useridx(2,可0=系统), cardNo(3 原文) | [echo,"unsell_card",useridx,result,cardNo] |
| buy_card | 9 | CenterHash | useridx(2), cardType(3), cardNo(4,UP), password(5) | [echo,"buy_card",accID,result,cardNo,fillAccount] |
| buy_item | 0xa | CenterHash | useridx(2), account(3,UP), itemID(4), amount(5), serverID(6) | [echo,"buy_item",useridx,result,item,amount,points] |
| item_info | 0xb | Center(hash0) | query(2,UP) | [echo,"item_info",result,total,start,N,echo2,(name,points)×N] |
| buy_item_give | 0xc | CenterHash | useridx(2), account(3,UP), itemID(4), amount(5), serverID(6), giveTo(7,UP) | [echo,"buy_item_give",useridx,result,item,amount,points,orderID,givee] |
| get_points | 0xd | CenterHash | useridx(2) | [echo,"get_points",useridx,result,points] |
| validate | 0xe | AccessHash | account(2,UP), passwd(3,UP), ip(4), ?(5), mode(6) | [echo,"validate",1,account,result,accID,"",points,"",chargeType,deadline(f),ext] |
| custom | — | 无 | （member 透传） | 官方空实现 |
| keep | — | 本地 | — | 无（刷新活跃时间） |

UP=大写化（Port_StringUpper；账号/密码/卡号域）。useridx==0 在
logout/unsell_card 是"按账号/系统"哨兵。

## 四、游戏服注册状态机（ServerStateMachine@0x6ff80）

state：0 Unknown / 1 Register / 2 Online / 3 Unregister / 4 Break
（gameStateName=GetState@0x74b70）。

| event | 来源 | 语义 |
|---|---|---|
| 1 | ReceiveReg | Unknown 态：FindServer(serverID=vl[2]+ServerIDOffset) 查重（命中回 [echo,"register",0]），否则登记→后端确认→state 2 |
| 2 | PublishRegister | Register 态：result==1 回填 serverID/port/zone→Online；回发 register 应答 |
| 3 | ReceiveUnreg | Online 态：→Unregister，后端注销 |
| 4 | PublishUnregister | Unregister 态：result==1→Break；回发 unregister 应答 |
| 5 | ReceiveReset | Register/Online/Unregister：RemoveUserByServer（无应答） |
| 6 | ReceiveKeep | Online：刷新 lastKeep |
| 7 | OnClose/超时 | 任意态：RemoveServer（不自动清 user） |

服务级状态机（ChargeStateMachine@0x74c60，this+0x1558）：
Unknown/Opened/Closing/Closed，事件 1 开门、2 请求关、3 落闸。

## 五、生命周期与巡检

- OnAccept/OnClose/OnReceive → 队列 → 主循环 DoWait@0x674a0：
  DB 队列（ProcessDbMsg 按 subop 0..0xe 分发 Publish*）→ socket 队列
  （ProcessInnerMsg：type0=AddServer 占位/type1=断连 event7/type2=报文
  /type3,4=dump）→ 每秒 ChargeStateMachine+CheckServers → 每 60s
  CheckDump+CheckUsers。
- CheckServers@0x6fe40：lastKeep 距今 >5 分钟（0.00347 天）断连。
- CheckUsers@0x766b0：每 AwakeTime 秒（ini 默认 60）对计费类型 1/3
  用户推 `["","renew",accID,type,unitType,deadline]`（空回显）。
- 顶号 UniqueAccount：同账号再 login → 旧会话收空回显 renew 踢人帧。
- user_info_t(0xe0)/server_t(0x68)/GameCharge 布局全表见
  tmp_research/fscharge/report_publish.md §18。

## 六、CAccess 存储过程层（Init@0x63fa0，15 sp）

| 键 | Oracle sp |
|---|---|
| REGISTER/UNREGISTER | sn_account.SnailGame.GameServer_Register / GameServer_Logout |
| LOGIN/LOGOUT | Game_UserLogin / Game_UserLogout |
| RENEW | Account_Subtract |
| QUERY_GIFT/GET_GIFT | Query_GamePresents / Take_GamePresents |
| SELL/UNSELL/BUY_CARD | Card_Vend / Card_Cancel / Card_Buy |
| BUY_ITEM/ITEM_INFO/BUY_GIVE | Property_Buy / getPropertyPrice / Property_Buy |
| RET_POINTS/VALIDATE | getAccountBalance / Validate_Account |

Game_UserLogin 参数（RequestLogin@0x5abb0）：入参 psMemAccount/
psMd5Pass/piServerID/piGameID/psClientIP，出参 piResult/piAid/piGMLevel/
psIsFree/piAccPoints/pdAccDlimit/pdLogonTime/psLogonID/psChargeInfo
——即 login 应答 14 元素的字段源。**口令=大写 MD5 hex**（客户端
SendRegister 家族 MD5String 实证）。

时间口径：天数浮点（get_cur_time 家族，宽限 +1.0=一天；从未落账
阈值 epoch+3 天）。

## 七、Go 复刻（internal/charge + cmd/charge）

- `wire.go`：VarList 文本编解码（上述方言全量，含 int64 分档/转义域）。
- `server.go`：会话（connid 0..255、MaxClient）、server FSM、用户注册表
  （index 从 1 起，0=哨兵）、顶号踢人、`[echo,cmd,...]` 应答。
- `handlers.go`：18 命令逐元素复刻（前置校验分层：login 族须 Online；
  buy_item/buy_item_give 只查连接；validate/item_info 无前置）。
- `housekeeper.go`：keep 5 分钟超时 + AwakeTime 主动 renew。
- `backend.go`/`accountbackend.go`：Backend 接口=sp 语义投影；
  FreeBackend（开发免费）与 MySQLBackend（nx_district.dc_accounts
  读路径：MD5 比对/points/charge_mode/valid_time）。
- `cmd/charge`：`GameCharge.ini` 兼容入口（`-dsn` 接 MySQL，缺省免费）。
- 测试：`wire_test.go`（编码 golden+往返）、`server_test.go`（全命令
  TCP 往返、重复注册、未注册拒绝、reset 清人、顶号、keep 超时——全绿）。

官方差异（有意）：Oracle sp 写路径（点数扣减/商城/点卡）暂空转成功，
接口已按 sp 语义预留；异步 DB 队列改同步调用（单连接保序，官方跨
连接本就无序）；EchoSock:15008 维护通道未复刻（echo 方言已由 lister
体系覆盖，需要时按 `Lua/GameCharge/echo.lua` 接入）。
