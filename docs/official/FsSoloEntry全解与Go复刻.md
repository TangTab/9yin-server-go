# FsSoloEntry.dll 全解与 Go 复刻

> 权威文档。证据源：`BGserverCJ\Debug` 快照的 FsSoloEntry.dll +
> FsSoloEntry.map（4897 符号全量）+ FsSoloEntry.pdb + Lua\soloentry\
> main.lua + game_entry.ini 实物 + 快照 127.0.0.1.lic。反汇编终局存
> `tmp_research/soloentry/`（capstone 注释版，命名 dis_*/oh_*/ih_*）。
> 复刻落地：`internal/fssoloentry` + `cmd/soloentry`（复用 `internal/entry`
> 的 Clients/Gateway/Inner 并叠加 solo 旋钮）。
>
> 网关版孪生 FsEntry.dll 已单独复刻（`internal/entry` +
> docs/official/FsEntry全解与Go复刻.md）；本文只记 solo 差异与实证细节。

## 一、进程定位与装配

`game_entry.exe`（通用 solo 壳，与 game_lister/game_auction 同一二进制）
读 `game_entry.ini`：`[Module] fskit.dll / fscomm.dll / fssoloentry.dll` +
`[Script] Path=..\lua\soloentry`。Lua main.lua 装配（对应
`cmd/soloentry/main.go`）：

| 实体 | 端口/缺省 | 复刻 |
|---|---|---|
| SoloEntry 主实体 | MainEntity=SoloEntry | fssoloentry.Server |
| EchoSock 维护 | [Echo] 127.0.0.1:15200+EntryID | fscomm.EchoSock + EchoHandler |
| 内网 Adapter（DLL 内建） | [Inner] 127.0.0.1:**0 动态** | entry.Inner（solo 旋钮） |
| 外网 MassSock（COuterHook 内建） | [Outer] :2001 | entry.Gateway |
| BroadSock 广播 | [Broadcast] 127.255.255.255:15000 | fscomm.BroadSock |

关键 Lua 语义：`license = outer_addr .. ".lic"`；`entry.InnerPort` 读回
（DLL 把动态分配的实际端口回写字段，Lua 随后打印）；`SetAppState(
WAPS_OPENED=4)`（八态进程壳记录，非服务状态机）；`load_ip(entry)` 读
`ip.txt` 黑名单（分隔符 `,`/TAB/空格依次回退）。维护命令：quit（close()
→ Stop + nx_quit）/ loadip / end（CloseEcho）+ 反射面（getp/setp/getc/
setc/method/listp/listc/listm/listg/function/debugger）。

SoloEntry 字段布局（.map + setter 反汇编锚定）：0x58 IDC / 0x5c
DistrictId / 0x60 ServerId / 0x64 EntryId / 0x68 MaxConnects / 0x6c
BlockOrAllow / 0x70 VerifyDll / 0xb4 InnerAddr / 0xd4 InnerPort / 0xd8
OuterAddr / 0xf8 OuterPort / 0xfc BroadAddr / 0x11c BroadPort / 0x120
EchoAddr / 0x140 EchoPort / 0x144 AppName / 0x164 LogLauncher / 0x18c
CReactor / 0x190 CEchoHook / 0x194 CInnerHook / 0x198 COuterHook / 0x1a0
CAppRecord / 0x1a8 广播口 / 0x1ac 内网监听 / 0x1b0 定时器 / 0x1b4..0x1d0
license 身份域 8×u32 / 0x1d4/0x1d8 license 时间窗。

## 二、Start 装配序（0x1009F4F0）

1. BindCore∈[0,0x40) → Port_SetProcessAffinity
2. `CMutex::Exclusive("mutex_"+AppName)` 失败 → printf + `_exit(0)`
3. UseQuickLog → CLogShareMem（SetLogPrefix(AppName)/SetAppId(EntryId+8)；
   共享内存不存在且 LogLauncher 非空 → Port_Execute 拉起 game_log）
4. UseAppRecord → CAppRecord（SetAppId(EntryId+8)/SetAppName/
   SetAppAddr("EchoAddr:EchoPort")）
5. CInnerHook：SetCallback(收/连入/连出/断开四静态壳) + SetAppName +
   **SetCommIdent(EntryId+8)**（solo 自身内网标识）
6. CReactor::Start → CreateBroadcast(BroadAddr,BroadPort) → 0x1a8
7. EchoAddr 非空 → CEchoHook::Start(EchoAddr,EchoPort)
8. **CreateListener(InnerAddr,InnerPort,10,4M,4M,64M,…)** → 0x1ac
   （"create inner listener failed"）；**InnerPort==0 → GetListenerPort
   回写实际端口**
9. COuterHook：SetInBufferLen/SetOutBufferLen/SetEncodeKey/SetDecodeKey +
   **SetClosed(true)**（起步暂停受理）+ Start(OuterAddr,OuterPort,flag)
   （内部再 CreateListener 外网口，"create outer listener failed"）
10. CreateTimer(1.0s, TimerCheck) → 0x1b0
11. **license 线程**（lookup_thread）+ WaitForSingleObject 0x7D0ms：
    超时 → TerminateThread + "(Start)program abort!" + abort
12. 时间窗有效（0x1d8>0x1d4）→ 打印 "license expired time is
    %d-%d-%d %d:%d:%d"（__time64 反解）
13. VerifyDll 非空 → LoadVerifyDll；SetCurrentEncode
14. UseThread → CThread 工作线程（Execute 循环）
15. state=0（Unknown）；打印 "SoloEntry pointer is 0x%p"

Stop（0x100A0B90）拆解序：CThread → DeleteBroadcast → DeleteListener →
DeleteTimer → CEchoHook::Stop → COuterHook::Stop → CReactor::Stop →
CAppRecord → CLogShareMem。

## 三、license（lookup_thread @0x10089C90 全破）

```
形态判定：fread 尺寸——256B 内部 / 272B 外部（其它 = "license file error"）
内部形态门槛：本机 IP 须 "127.0.0.1" 或前缀 "192.168."
             （否则 "not internal address"）
key 缓冲（32B）＝ memset 0xCC → [0..15]=盐16 → CopyString([16..31],16,IP串)
盐   = 77 56 D3 34 2B 8C E2 2F ED AC B8 15 45 9E 5E C4（与 FsEntry 同盐）
密钥 = md5(key缓冲)                       ← 不同于 FsEntry 的 md5(IP+盐+0xCC×16)
des2key(密钥16, 1)（2-key 3DES，EDE 等价 K1||K2||K1）
外部形态二段密钥：D2des(buf[0x100..0x10F]) 解出明文尾 → 回填 key[16..23]
                  → md5 → 真·会话密钥 → des2key(真密钥, 1)
解密：16 次 D2des（每次 16B=两块）× stride 16 = 全 256B
校验和门：Σ buf[0..0xFB]（字节和） == u32@buf[0xFC]，失配静默返回
外部形态时间窗：buf[0x100..0x107]（明文尾）= {t0,t1}；
  now=__time64(0)：now<t0 → "license too young!"；now>t1 → "license too old!"
落地：buf[0..0x1F] 8×u32 → this+0x1b4..+0x1d0；{t0,t1} → this+0x1d4/+0x1d8
```

TimerCheck 复验（每秒，非致命）：

- `this+0x1b4 == 0xE7990369` → **BroadcastEntryInfo**（授权才广播）
- `this+0x1b8 != 0x87930363` → WriteLine "not licensed server
  program!!!"（仅 printf，服务不退）
- 身份域可被运行时清零（见 §五世界窗）——快照 127.0.0.1.lic 实测属
  网关版（FsEntry 派生下解出 0x8E74C52F/0xEB64EB5C），solo 派生下为
  乱码：同机双网关形态各验各的魔数对

生成器：`fssoloentry.GenerateLicense(ip)`（内部形态）/ 
`GenerateExternalLicense(ip,t0,t1)`——魔数对 + 零填充 + 求和校验回填，
与官方校验序互认。

## 四、内网面（CInnerHook @0x1006B…）

- 线路：FsComm Adapter（u32 前缀帧）之上的 12B 头
  `{src u32, dst u32, type u32}` + payload；<0xC = "message size error"
- **ident 注册**（type=0xFFFF）：src 即宣告身份；comm_id∈[0,0x40)、
  connector 越界断言；**重复 ident → 向新连接回 `{src=自身,dst=0,
  type=0xFFFE}` 12B 帧拒绝其注册（"comm_id duplicate"，保旧踢新）**；
  成功 → trace `"%s connector_id %d comm_id %d received"`
- **type=0xFFFE 入站 → ICore::Quit(1)**（进程退出）
- 正常消息 → 回调 → SoloEntry::ProcessInnerMsg
- 源校验（ProcessInnerMsg 入口）：**{0,1,2} ∪ [0x18,0x2F]**，越界
  "message source error! comm:%d"（比网关版 {0}∪[0x18..0x27] 放宽）
- 自身 ident = EntryId+8；member 24 位（0x18..0x2F）
- Go 复刻：`entry.Inner` 新增四旋钮（SourceCheck/OnQuitFrame/
  KickDuplicate/MemberBase+MemberCount），零值保持网关版语义不变

## 五、ProcessInnerMsg 15 型（0x100A1FE0 跳表直读）

| type | 结构体（断言原文） | 尺寸 | 语义 |
|---|---|---|---|
| 0 | entry_forw_init_connect_t | ==8 | {flag,window}：flag≠0→事件1开/=0→事件2停；window 见下 |
| 1 | entry_forw_break_client_t | ==8 | {ident,serial} 踢线 |
| 2 | entry_forw_reset_client_inuse_t | ==8 | {ident,serial} 槽位回收（连接断后确认） |
| 3 | entry_forw_set_client_online_t | ==0x10 | {ident,serial,member,**scene**} |
| 4 | entry_forw_send_to_multi_t | >=4 | {n}+n×{ident,serial}+payload |
| 5 | entry_forw_send_to_scene_t | >=4 | {scene}+payload |
| 6 | entry_forw_send_to_client_t | >=8 | {ident,serial}+payload |
| 7 | ——（OnlineSend） | —— | payload 仅在线播 |
| 8 | ——（BroadcastSend） | —— | payload 全量播 |
| 9 | entry_forw_shake_client_t | ==8 | 在线回 1B{0x06}；失联回报 world type 2 |
| 10 | entry_forw_channel_add_t | ==0xC | {ident,serial,channel} |
| 11 | entry_forw_channel_remove_t | ==0xC | 同上 |
| 12 | entry_forw_clear_channel_t | ==4 | {channel} |
| 13 | entry_forw_send_to_channel_t | >=4 | {channel}+payload |
| 14 | —— | —— | close → EntryStateMachine(3) |

与网关版编号不同且**无落格/移动/视野三型**（solo 无空间网格；符号表
无任何 sight/grid 资产）。关键语义：

- **set_client_online 第 4 域是 scene**：ClientOnline(ident,serial,a,b)
  写 client+0x14=a（member，路由依据）+ client+0x18=b（scene，
  SceneSend 过滤依据）——网关版第 4 域是 memberIdent
- **MultiSend**（0x10096320）：断言 `(count>0)&&(count<1024)` 与
  `len > header_size`；头 = 4+n×8；`(ident>>16)&0xFFFF == EntryId`
  （本组前缀）才发；InnerSendPrepare 编码一次逐客户端 SendClientEncode
- **SceneSend**（0x10096560）：遍历客户端数组按 client+0x18==scene
  过滤；同码编码一次
- **ResetInuse**（0x10095FE0）：按 (ident,serial) 找槽，校验连接映射
  （"connector not closed"/"connector index not match"），递减 connect
  计数并回收槽位
- **世界窗联动**（type 0）：license 带有效窗（f1d8>f1d4）且
  `window ∉ [f1d4,f1d8]` → **身份域 8×u32 清零**（撤销授权：停广播 +
  每拍告警）——世界侧时间值越出授权有效期即自毁

## 六、外网面（COuterHook @0x10093…）

与网关版同构（EE 转义 + XOR + 纯代理），solo 差异：

- 池上限断言 **0x10000**（网关版 0x2000）；实用上限 = MaxConnects
- OnAccept 序：Closed 闸（"server closed"）→ IsBlockIP（"ip blocked"）
  → 池断言（"client array too big"）→ connect 计数 + trace
  "address %s:%d connected, total %d" → 分配槽（member=-1@0x14）→
  **world accept 通知**（SendToWorld type 1，28B {ident,serial,addr[16],
  port}）
- ReceiveMessage：VerifyMessage（fsverify 钩子）→ op 统计（首字节
  1..12）→ CheckMessageIllegal（失败 → world type 4）→ client+0x14≥0
  → SendToMember2(member,type 3,{ident,serial,原始帧}) 否则
  SendToWorld2(type 3)
- client 布局（solo）：+0x00 状态字 / +0x0C ident / +0x10 serial /
  +0x14 member(-1 未绑) / +0x18 scene / +0x44 帧缓冲 / +0x1044 缓冲长
- ClientConnected(shake 判定) = client+0 ≥ 0；ClientSend 失联回报
  world type 2（"send client lost failed"）
- InnerSendPrepare 压缩 + op 重映射与网关版同为已知未复刻项

## 七、状态机 / 定时器 / 广播

- 四态（this+0x54）：0=Unknown / 1=Opened / 2=Paused / 3=Closed
  （EntryStateMachine @0x100A4F00 跳表 → EntryState{Unknown,Opened,
  Paused,Closed}，soloentrystate.cpp）。事件：0=tick / 1=open /
  2=pause / 3=close。Unknown 态事件2 = "connect server already
  closed"；非法输入 `"(EntryState%s)error input %d"` 返回假
- TimerCheck（1s）：第 6 拍（计数器 >5 归零）上报
  `<info state="%s" connect="%d" inuse="%d" tick="%d" />`（Port_
  GetTickCount）→ license 双门（§三）→ EntryStateMachine(0)
- BroadcastEntryInfo（0x100A1E50）**仅授权时每秒一拍**，0x48B：
  `{u32 ver=1, u32 msg=0x3001, inner_addr[16]（CopyString 截 15+NUL）,
  u32 inner_port, u32 district, u32 server, u32 entry, u32 idc,
  outer_addr[16], u32 max_connects, u32 connect_count}`（Reactor::
  SendBroadcast）
- Execute(elapsed) = Reactor 泵（Go 版组件自管协程等价）

## 八、Go 复刻清单

| 官方 | 复刻 | 测试 |
|---|---|---|
| lookup_thread 双形态 license | fssoloentry/license.go | TestLicense*（往返/地址门槛/时间窗/错 IP 校验和） |
| game_entry.ini solo 方言 | fssoloentry.ParseSoloINI | TestParseSoloINI（官方原文回归 + Echo 偏移） |
| ProcessInnerMsg 15 型 + 世界窗 | fssoloentry/process.go | TestProcessInnerMsgAll15 + TestWorldWindowWipesLicense |
| EntryStateMachine 四态 | fssoloentry.Server | TestProcessInnerMsgAll15（开/停/关） |
| TimerCheck（状态行节拍/授权门） | fssoloentry.Server | TestTimerCheckLicenseGatesAndCadence |
| BroadcastEntryInfo | Server.BroadcastInfo | TestBroadcastInfoGolden（逐域断言） |
| CInnerHook solo 语义 | entry.Inner 四旋钮 | TestSoloSourceCheck + TestInnerKickDuplicate |
| COuterHook 外网面 | entry.Gateway（复用） | TestSoloEndToEnd（真 TCP：注册→开放→接入→accept 通知→下行帧） |
| Lua echo 命令面 | fssoloentry.EchoHandler | TestEchoCommands + TestLoadIPFile |
| game_entry.exe 进程 | cmd/soloentry | 冒烟实录（下） |

冒烟实录（build/game_soloentry.exe，净目录）：

- 自动释放内置 game_entry.ini → `inner socket 127.0.0.1:51778`
  （**动态端口**）→ 官方横幅序五连 → `game server 11 entry 0 started`
- `-gen-license 127.0.0.1` → 256B 内部形态 lic；装载 → "license 就绪
  …魔数对 0XE7990369/0X87930363"，无告警；`-no-license` → 每轮一条
  "not licensed server program!!!" 且不广播（官方非致命语义）
- echo 对拍（15200 口）：`welcome server 11 entry 0` + `>state` 状态行
  + `>reload block ip list.../ended` + `>undefined command: badcmd`
  全中；状态行第 6 拍出线

已知未复刻项（记录在案）：

- quicklz 大包压缩 + op 重映射（0x27..0x30）——与网关版共有的已知缺口
- Lua 反射维护面（getp/…/debugger）以 state/loadip/help 等价，未做
  实体属性反射
- CMessageLog（message.log 收发留痕）未接；CAppRecord/CLogShareMem/
  CShareMem 跨进程运维件以日志呈现等价
- CheckClients 超时清扫的官方节拍未细究（Go 版由网关断连路径等价承载）
