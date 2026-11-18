# FsEntry.dll 全解与 Go 复刻

> 权威文档。证据源：`F:\Game\Online\9yin\BGserverCJ\Debug` 快照的
> FsEntry.dll + FsEntry.pdb（PDB2.00，全快照唯一）+ Lua/GameEntry 脚本 +
> GameEntry.ini + 127.0.0.1.lic 实物；孪生 FsSoloEntry.dll/.map 提供全量
> 符号反推。复刻落地：`internal/entry` + `cmd/entry`。
>
> 逆向方法注记：FsEntry 无 .map 且 PDB 为 VC6 时代 MSF 2.00（目录区
> 指向 ~16MB 虚拟空间，疑增量链接残骸，pdb_syms.py 不适用），改走
> 「字符串交叉引用定位 + 断言串还原结构体 + FsSoloEntry.map 孪生符号」
> 三源路线。

## 一、进程定位与装配

`GameEntry.bat → FsLoader.exe GameEntry.ini`：

```ini
[Module]
FsKit.dll      ; 框架（实体/属性/Reactor）
FsComm.dll     ; Adapter/BroadSock/EchoSock/AppManager/LogPool/IniDocument
FsEntry.dll    ; GameEntry 本体
[Script] Path=..\Lua\GameEntry   ; main.lua + echo.lua
```

Lua/main.lua 装配（对应 `cmd/entry/main.go`）：

| 实体 | 端口/缺省 | 复刻 |
|---|---|---|
| EchoSock 维护 | 15200+EntryID | fscomm.EchoSock |
| MassSock 外网 | [Outer] :2001 | entry.Gateway |
| BroadSock 广播 | 127.255.255.255:15000 | fscomm.BroadSock |
| Adapter 内网 | 127.0.0.1:16200+EntryID | entry.Inner（fscomm.Adapter） |
| AppManager | AppName=`entry_<sid>_<eid>` | 状态行/进程名 |
| GameEntry | 身份+键+license | entry.Entry |

关键配置键：`IDC/DistrictID/ServerID/EntryID/MaxPlayer/SockThreads/
MessageEncodeKey(hex→u32)/MessageVerifyDll/[Outer]Addr,Port/[Inner]/
[Broadcast]/[Echo]`（复刻 entry.ParseGameEntryINI 兼容 solo 版
MaxConnects 键名）。`License = <OuterAddr>.lic`。

与 solo 变体差异：soloentry Lua 把全部服务装进单实体（Inner 端口 0，
自管），GameEntry 变体是**多进程网关**——world/member 从外部连入内网口。

## 二、外网面（客户端 TCP，CMassSockHook/CClients）

### 线路帧
- EE 转义（OnReceive 状态机）：`EE EE`=帧界（首 EE 已入缓冲，收第二个
  EE 时回退一字节并派发整帧）；数据中 `EE 00`=转义出字面 EE（00 丢弃）；
  单帧缓冲上限 0x1000，prev 字节记忆在 client+0x1048。
- XOR 编码：XorCipher 滚动键（EncodeKey=ConvertEncodeKey(ini)），
  internal/net 已按客户端 fxnet2 逆向复刻（golden 对拍过）。
- 压缩（InnerSendPrepare）：消息 >0x100 且 CompressMessage 开启时
  quicklz 压缩 + op 重映射 {0x27,0x28,0x29,0x2a,0x2c,0x2e,0x30}
  （op-0x0D 索引 0x22 项跳表）。复刻未含（本线客户端不启用）。

### 槽位与身份
- 池：MAX_CLIENT_NUM=0x2000 槽；槽结构 0x4C（网关版）/0x104C（solo 版
  带帧缓冲）。ident = `EntryID<<16 | 槽位`；serial 单调递增且跳 0。
- client 域：bInuse@0 / bOnline@1 / ident@0xC / serial@0x10 /
  member@0x14(-1 未绑) / memberIdent@0x18 / next@0x3C(格链表) /
  x@0x44 / y@0x48。
- OnAccept 流：Closed 闸 → IsBlockIP → 池满检查 → 分配 →
  **向 world 上报 28B accept 通知** {ident,serial,addr[16],port}
  （"send client accept failed"）。

### 转发（纯代理）
ReceiveMessage：op 统计（首字节 1..12；发向客户端统计至 50）→
CheckMessageIllegal（fsverify 钩子，非法→world type 4）→
已绑 member → `SendToMember2(type 3)`，否则 `SendToWorld2(type 3)`；
帧体 = {ident,serial,原始客户端帧}。entry 不解析游戏 op 语义。

### 下行（world/member 驱动）
ClientSend(ident,serial,data) → 编码（XOR+EE）→ TCP。shake 失联回报
world type 2 {ident,serial}。

## 三、内网面（CAdapterHook + CInterConnect）

- 线路：FsComm Adapter（TCP，`[u32 LE len][body]`）。
- 帧头 12B：`{src u32, dst u32, type u32}` + payload。
- **ident 注册**：type=0xFFFF 特例，src 即宣告的连接身份
  （SendAdapterCommIdent 对偶）。
- 源校验（ReceiveMsg）：src ∈ {0=world} ∪ {0x18..0x27=member 0..15}
  （MAX_MEMBER_NUM=16），否则 "message source error"。
- 消息尺寸门：<0xC 即 "message size error"。

### entry_forw_* 16 型（ProcessMsg 跳表 0..0xF）

| type | 结构体（断言原文） | 尺寸 | 语义 |
|---|---|---|---|
| 0 | init_connect_t | ==4 | flag≠0→开放(1)/=0→暂停；世界窗校验 |
| 1 | break_client_t | ==8 | {ident,serial} 踢线 |
| 2 | set_client_inuse_t | ==9 | {ident,serial,bInuse} |
| 3 | set_client_online_t | ==0x10 | {ident,serial,member,memberIdent} 绑定上线 |
| 4 | set_client_posi_t | ==0x28 | {ident,serial,_,scene,x,y,x0,y0,w,h} 落格 |
| 5 | set_client_move_t | ≥0xC | {scene,_,nCount}+n×16{ident,serial,x,y} |
| 6 | send_to_sight_t | ≥0x1C | {ident,serial,_,scene,x,y,radius}+payload |
| 7 | send_to_client_t | ≥8 | {ident,serial}+payload |
| 8 | —— | —— | 全量广播（二级 kind=0xB） |
| 9 | —— | —— | 批组播 MultiSend（2 参形态） |
| 10 | shake_client_t | ==8 | 在线回发 1B {0x06}；失联回报 type 2 |
| 11 | channel_add_t | ==0xC | {ident,serial,channel} |
| 12 | channel_remove_t | ==0xC | 同上 |
| 13 | clear_channel_t | ==4 | {channel} |
| 14 | send_to_channel_t | ≥4 | {channel}+payload |
| 15 | —— | —— | 仅在线广播（二级 kind=2） |

> solo 变体编号（FsSoloEntry ProcessInnerMsg 0..0xE）：0=init,
> 1=break, 2=reset_inuse, 3=online, 4=multi, 5=scene, 6=client,
> 7=online_send, 8=broadcast, 9=shake, 10..13=频道四件套, 14=close
> （SetState(3)）。源校验放宽为 {0,1,2}∪[0x18,0x2F]。
> 8/9/15 三型无断言串，语义由二级分派器 (0,kind,msg,len) 的 kind 值
> {1 开放, 0 暂停, 2 在线播, 0xB 全量播} 反推——置信度中，复刻按此落地。

## 四、场景网格与视野扇出（FsEntry 独有）

ProcClientPosi（set_client_posi）懒建场景网格：

- 场景表 0x24B/项：{x0,y0,w,h,cell,rows,cols,total,grid[]}
  （+0x10 cell、+0x14 rows、+0x18 cols、+0x1C total、+0x20 格数组）。
- **cell 自适应**：从 1 起 ×2，当 `max(w,h)/cell ≤ 0x200`（严格大于才
  继续）或 cell=8 停。
- 格数组 u32/格 = 客户端链表头；client.next@0x3C 串链。
- 维度任一变化 → 释放旧格整场重建（重建后各客户端待 world 逐个重发
  posi 落位——复刻保持此语义）。

ProcSightSend（send_to_sight）：

- 预建**螺旋偏移表** 255×255 项（dx,dy ∈ [-127,127]），
  dist=√(dx²+dy²)，qsort 升序（早退前提）。
- 遍历：`dist×cell ≤ radius+cell` 为格级上限（超即 break）；每格走链
  表，断言 bInuse/bOnline，逐客户端 `(cx-x)²+(cy-y)² ≤ (radius+cell)²`
  精判，命中即 ClientSend。
- 表半径 ±127 格 → 视野硬上限 127×cell 世界单位。

## 五、License（LookupThread 全破）

```
lic = 读 `<OuterAddr>.lic` 恰 256B            （少一字节即 "license file error"）
ip  = 建哑套接字 getsockname → inet_ntoa      （IAT 序号导入 #6/#12）
key = md5( ip字符串 + 盐16 + 0xCC×16 )         （盐 77 56 D3 34 2B 8C E2 2F
                                                 ED AC B8 15 45 9E 5E C4，
                                                 FsEntry.dll 0x1005D827 逐字节）
lic' = 3DES-EDE-ECB(key16 为 K1||K2, 即 K1K2K1) 解密 256B   ← 逐 8B 块
身份域 = 明文头 20B → GameEntry+0xC4..+0xD4（5×u32）
门槛   = 首块魔数对
```

- 快照 `127.0.0.1.lic` 实测魔数对：**0x8E74C52F / 0xEB64EB5C**；
  明文其余为随机填充（仅首块参与判定）。
- solo 变体 TimerCheck 对照魔数：0xE7990369 / 0x87930363
  （FsSoloEntry.dll 0x100A28EB 起，每 tick 复验，失败
  "not licensed server program!!!"）。
- **生成器**：`entry.GenerateLicense(ip, m1, m2)` 可为任意 IP 自签
  （明文=魔数+零填充；与官方随机尾部不等价但同过校验）。
- `cmd/entry -gen-license <ip>` 一键出 lic。

## 六、状态机 / 定时器 / 广播 / 维护面

- 四态：0=Unknown / 1=Opened / 2=Paused / 3=Closed
  （GameEntryState*.cpp；init_connect flag 驱动 1/2，mass_sock.Closed
  闸联动：非 Opened 即拒新连接）。
- TimerCheck：license 复验（失败→Closed）+ 状态机 tick；5 tick 一次
  状态行 `<info state="%s" connect="%d" inuse="%d" tick="%d" />`
  （发往 AppManager/监控通道）。
- BroadcastEntryInfo：UDP 0x48B 包 `{ver=1, msg=0x3001, name1[16],
  u32×4, name2[16], u32, connect_count}`，BroadSock 周期发。
- 维护面行协议：`welcome server <sid> entry <eid>\r`、`>...\r` 应答；
  命令 quit/end + 反射面（getp/setp/getc/setc/method/listp/listc/
  listm/listg/function/debugger）；solo 另有 loadip（ip.txt 黑名单
  重载，`beg,end` 行 → AddBlockIP）。

## 七、Go 复刻清单

| 官方 | 复刻 | 测试 |
|---|---|---|
| CClients 池/绑定/频道 | entry.Clients | TestClientsLifecycle 等 |
| 场景网格+螺旋表 | entry.grid.go | TestPickCell/TestSightTableOrder/TestSceneGridSight |
| ProcessMsg 16 型 | entry.ProcessForward | TestProcessForwardAllTypes |
| ReceiveMsg 头/注册/源校验 | entry.Inner | TestInnerWireCodec + e2e |
| MassSock 外网面 | entry.Gateway | TestGatewayEndToEnd（真 TCP） |
| LookupThread license | entry/license.go | TestLicenseSnapshotGolden（快照实物）+ 回环 |
| BroadcastEntryInfo | Entry.BroadcastInfo | TestBroadcastInfoPacket |
| CBlockIP | entry.BlockIPList | TestBlockIP |
| GameEntry.ini | entry.ParseGameEntryINI | TestParseGameEntryINI |
| game_entry.exe 进程 | cmd/entry | 冒烟：注册→开放→接入→下行 |

冒烟实录（build/entry.exe）：假 world 注册 ident 0 → init_connect(1) →
state=unknown→opened → 客户端接入 → world 收 type 0 通知
`{ident=0x0, serial=1, addr=127.0.0.1, port=51075}` → set_online +
send_to_client → 客户端收到 EE+XOR 帧 `c7 cd ee ee`（=0x0B 0xF6）。

已知未复刻项（记录在案）：
- quicklz 大包压缩 + op 重映射（0x27..0x30）；
- AppManager/AppBook 共享内存注册（fscomm 已有 AppBook，未接）；
- fsverify 每帧校验钩子（Gateway.VerifyMessage 预留位，可接
  internal/fsverify）；
- 二级分派器 kind 0/1 之外的官方语义细节（见 §三注）。

## 八、测试与构建

```
go test ./internal/entry/ -timeout 60s     # 14 用例全绿
go build ./cmd/entry/                      # build/entry.exe
./entry.exe -gen-license <外网IP>          # 自签 lic
./entry.exe                                # 起服（GameEntry.ini 可选）
```
