# FsSoloLister.dll 全解与 Go 复刻

> 标的：`F:\Game\Online\9yin\BGserverCJ\Debug\FsSoloLister.dll`
> （708 608 B，随快照带 **.map 全符号**（1730 个公符号）+ pdb 2.9MB）
> 方法：tmp_research/sololister/sll_dis.py（capstone + map 符号 RVA 直读，
> 调用目标 thunk 解析 + .rdata 常量池字符串注释）
> 产物：`internal/fssololister`（Go 复刻）+ cmd/sololister 进程载体
> 完成日：2026-11-19
> 姊妹篇：FsLister全解与Go复刻.md（大写系集群形态）、分布式形态清单.md
> （本文是其 §12 预研规格的**勘误与终局**——server_t 布局、线过滤字段、
> 中继语义三处旧记有误，见 §八）

---

## 一、模块定位

FsSoloLister.dll 是**小写系（九阴 solo 体系）**的区服列表服务器实体类模块
（game_lister.exe 直启 → fskit.dll + fscomm.dll + fssololister.dll 三件套，
game_lister.ini `[System] MainEntity=SoloLister`）。与大写系 FsLister.dll
同源分工，但**自成一体的进程**：不挂 MassSock/BroadSock 实体，直接内嵌
CReactor IOCP 引擎 + CEchoHook 维护台 + CAppRecord 进程记录 + CBlockIP
黑白名单（四组件随 DLL 静态链接，map 可见 SoloLister/Reactor/EchoHook/
AppRecord/BlockIP/LogShareMem/TextProtocol/AppTool 八个 obj）。

- 源码路径（断言串）：`c:\wuyf\svn\serveriii\branches\publish20130508\
  server\fssololister\sololister.cpp`
- 导出：8 个标准 FxModule（GetType=**"FxModule"**/GetVersion=**1003**
  （0x3EB，大写系为 1000）/GetSpace=""/GetEntCreator…）
- 实体注册：SoloListerEntCreator（GetName=**"SoloLister"**，
  GetParent="IEntity"，GetSpace=""），Create → `new SoloLister`

### 实体属性/方法（CRT 自举静态注册）

| 属性 | 类型 | 方法（脚本可调） |
|---|---|---|
| State（只读 string）/ClientCount（只读 int） | — | Start / Stop |
| UseThread / UseQuickLog / UseAppRecord | bool | SetAppState / SetConsoleColor |
| IDC / DistrictId / ListerId / MaxConnects / ClientVersion / BlockOrAllow | int | WriteLine / SendEchoLine / CloseEcho |
| InnerAddr / OuterAddr / OuterPort / BroadAddr / BroadPort / EchoAddr / EchoPort / AppName / LogLauncher | str/int | AddBlockIP / ClearBlockIP / DumpReactor |

ListerId 走专用 SetListerId@100610A0：**仅 0≤id<8 接受，越界静默忽略**。

## 二、SoloLister 对象布局（构造器/Getter 反汇编实证）

```
+0x000  vtable（IEntity 派生）
+0x028  void*  m_pWinSocket（Init: GetCore→QueryInterface("WinSocket")，WSA 生命周期）
+0x030  int    m_State（GetState 跳表：0=Closed/1=Paused/2=Opened/3=Unknown；Start 尾置 1）
+0x034  int    IDC                ← 线过滤字段（§四，实证）
+0x038  int    DistrictId         ← UDP 分区门
+0x03C  int    ListerId（0..7 钳制）
+0x040  int    MaxConnects = 8000（ctor；lua 以 ini 覆写为 1024）
+0x044  int    ClientVersion（版本门，0=不设）
+0x048  int    BlockOrAllow = -1（-1 不拦/0 黑/1 白）
+0x04C…+0x118 六个 TFastStr<32>：InnerAddr/OuterAddr/BroadAddr/EchoAddr/AppName/LogLauncher
+0x08C  int OuterPort    +0xB0 int BroadPort（双 UDP 口共用；ini 无 [Inner] Port 键）
+0x118  CBlockIP（内嵌）
+0x11C  CReactor* m_pReactor（Start 时 new + Start）
+0x124  CThread*（UseThread 时：reactor EventLoop 跑线程）
+0x128  CLogShareMem*（UseQuickLog 时）
+0x12C  CAppRecord*（UseAppRecord 时）
+0x130  int 本地广播口 id   +0x134  int 配置广播口 id（-1=未建）
+0x138  int 外网 listener id +0x13C  int 1s 定时器 id
+0x140  int  客户端计数
+0x144  TArrayPod<client_t*> m_Clients
+0x158  TArrayPod<server_t*> m_Servers
+0x16C  TArrayPod<unsigned>  空槽索引表（除名下标回收/建服复用）
```

### client_t（0x10C；AddClient new + memset 0）

```
+0x00  int state（0=待请求 1=已应答 2=关闭中）
+0x04  int counter（秒拍计数）
+0x08  char line_buf[0x100]（行缓冲；实际存至 0xFF，末位留 NUL）
+0x108 int line_len
```

### server_t（**0x4E8** = 0x268 头 + line_t[16]×0x28）

```
+0x00  byte visible（world 包 +0x74≠0；应答可见门）
+0x01  byte from_world（world 注册置 1；心跳丢失清 0）
+0x04  int  server_id（FindServer 线性查找键）
+0x08  int  world_latch（world 心跳闩锁：注册置 1，扫描清 0）
+0x0C  int  max    +0x10 int cur    +0x14 int recommend
+0x18  char ip[16]
+0x28  wchar name[0x40]（宽显示名，应答 # 转义来源）
+0x68  wchar name2[0x100]（预留，world 注册时清空）
+0x268 line_t[16] × 0x28：
    +0x00 byte valid  +0x04 int type闩锁（entry 注册置 1，扫描清 0）
    +0x08 int line_id +0x0C int max +0x10 int cur +0x14 int port +0x18 char ip[16]
```

> **勘误**：分布式形态清单.md 旧记「server_t+0x00 char 服名[0x28]」有误——
> +0x00 起是 visible/from_world 标志字节与 server_id，服名在 +0x28 宽串。

## 三、:4000 客户端面（CreateListener + ReceiveClientMsg）

**监听参数**（Start@0x10061280）：CreateListener(OuterAddr, OuterPort,
pendings=**100**, inbuf=**0x400**, outbuf=**0x4000**, on_accept/on_close/
on_receive)；UseThread=false 时主循环 Execute→CReactor::EventLoop 泵。

**行方言**（ReceiveClientMsg@0x10062F40 逐字节状态机）：

- **'\r'(0x0D) 与 '\n'(0x0A) 都是行终结符**；空行（line_len==0）不分发
- 其余字节（**含控制字符**）一律入行缓冲——与大写系「<0x20 丢弃」相反
- 行缓冲存满 **0xFF** 静默丢字节（0x100 缓冲留 NUL 位；大写系 0x1000）
- OnAccept：黑白名单命中即断（日志 `address %s:%d ip blocked`），
  否则 AddClient + 日志 `address %s:%d connected, total %d`
- OnClose → RemoveClient + 日志 `… disconnected, total %d`

**ReceiveCommand@0x10063140**（单请求 + 版本门 + 应答构造）：

```
if ClientVersion > 0:                       # 版本门（大写系无此属性）
    strlen(line) >= 8 且 strncmp(line,"svrlist",7)==0 否则静默 return false
    ver = atoi(line+8)                      # C atoi：前导空格/符号/数字截断
    ver < ClientVersion → 应答 $version_error\r\n → return true
elif strcmp(line,"svrlist") != 0 → return false   # 无门时精确匹配
client = m_Clients[index]; client->state != 0 → return false   # 单请求门
VarList = [AddString("svrlist")]
for server in m_Servers:
    server==NULL || server->visible==0 → 跳过
    best=-1, bestrate=10.0f                 # 官方常量 [0x100d0e78]（大写系 1.0）
    for j in 0..16:
        line = server+0x268+j*0x28
        line->max==0 或 line->valid==0 → 跳过
        IDC>0 且 line->line_id != IDC → 跳过   # ← 过滤字段=IDC（§八勘误）
        rate = (float)cur/max
        rate < bestrate 才换线（严格小于，等值保持先者）
    best<0 → 跳过该服
    busy = recommend>10 ? recommend : (max>0 ? cur*5/max : 0)
    AddWideStr(name); AddString(line.ip); AddInt(line.port); AddInt(busy)
text_encode_msg(VarList, buf, 0x3FE) + "\r\n" → CReactor::Send
发送成功 → client->state=1, counter=0
```

**客户端机**（CheckClient@0x10064140，1s 定时器驱动；ShutdownConnector
温和关闭→DeleteConnector 硬关）：

```
state 0 待请求: counter++ >30（第 31 拍）→ 温和关闭 → state 2
state 1 已应答: counter++ >5  → 温和关闭 → state 2
state 2 关闭中: counter++ >1  → 硬关 + 删表 + 日志 "remove client %d, total %d"
```

即：**30s 请求超时 / 应答后 5s 关闭 / 关闭后 1s 回收**（大写系 60s/5s）。

## 四、UDP 双口注册面（CreateBroadcast × 2 + 中继）

Start 中的两个广播口（**共用 BroadPort**；ini 无 [Inner] Port 键）：

```
本地口（仅 InnerAddr != "127.0.0.1" 时建立）:
    CreateBroadcast("127.0.0.1", "127.255,255,255", BroadPort, 0x400, cb=0x10061F90)
配置口（恒建立）:
    CreateBroadcast(InnerAddr, BroadAddr, BroadPort, 0x400, cb=0x10062030)
```

（"127.255,255,255" 逗号形态由 Port_GetBroadcastBindAddr 解析；Go 侧
127/8 定向回环等价——Linux lo 无 BROADCAST 的既定对策。）

**本地口回调（0x10061F90）——中继器**：

```
len<8 丢弃；pkt[+4] != 0x3001 丢弃；kind=pkt[+0] ∉{0,1} 丢弃
→ SendBroadcast(pkt)：经配置口原样转发到 BroadAddr:BroadPort
（无分区门、无长度门——中继不解析载荷，目的端配置口再全验）
```

同机 world/entry 只需定向 127.0.0.1:BroadPort，中继让注册同时可达跨机
部署的其他进程——**官方双口设计的目的即此**。

**配置口回调（0x10062030）——全门 + 入表**：

```
len<8 丢弃；pkt[+4] != 0x3001 丢弃
kind=0（world）: len==0x84 必须；pkt[+0x1C]==DistrictId 必须
                → RecvWorldServerInfo(pkt,len)
kind=1（entry）: len==0x48 必须；pkt[+0x1C]==DistrictId 必须
                → RecvEntryServerInfo(pkt,len)
```

**报文布局**（收发互证：FsSoloWorld.dll BroadcastWorldInfo@0x100B55C0
构造 0x84 定长、kind 0、魔数 0x3001；FsSoloEntry.dll BroadcastEntryInfo 同构）：

```
world_info（0x84=132B，比大写系 0x78 多 12B）:
  +0x00 kind=0  +0x04 magic=0x3001  +0x1C district_id
  +0x08 char ip[16]  +0x20 server_id  +0x24 wchar name[≤0x40]
  +0x6C max  +0x70 cur  +0x74 visible(u32)  +0x78 recommend（solo 独有）
entry_info（0x48=72B）:
  +0x00 kind=1  +0x04 magic=0x3001  +0x1C district_id
  +0x20 server_id  +0x24 line_index(<0x10)  +0x28 line_id
  +0x2C char ip[16]  +0x3C port  +0x40 max  +0x44 cur
```

> **勘误**：大写系魔数 0x300/包长 0x78，solo 系 **0x3001/0x84**——
> 两系注册报文**不互通**。旧清单未记魔数与 world 包长差异。

**RecvWorldServerInfo@0x10063B60**：FindServer(data[+0x20]) 未命中 →
AddServer（日志 `add server %s(%s)`，宽名窄化 + ip）；然后
from_world=1、world_latch=1、visible=+0x74≠0、max/cur/**recommend**、
ip、name 覆写、name2 清空。RecvEntryServerInfo@0x10063FB0：line_index
≥16 拒；**FindServer 未命中拒（entry 不建服）**；线槽 valid=1、type=1、
line_id/max/cur/port/ip 覆写。

**AddServer@0x10063A00**：new server_t[0x4E8]（memset 0、记 id）→
空槽索引表非空则复用表头下标，否则 push_back。

## 五、表生命周期（TimerCheck@0x10064650）

1s 定时器（CreateTimer(**1.0f**, …)）每拍：

```
每拍：全客户端 CheckClient（§三状态机）
每 5 拍：全服表 CheckServer：
    线级：line.valid && line.type==0 → line.valid=0；else line.type=0
          （entry 每 5s 必重发，丢一拍即摘线）
    服级：from_world && world_latch==0 → 服死：
          日志 "remove server %s(%s)"、delete、槽下标进空槽索引表
          else world_latch=0（world 每 5s 必重发，丢一拍即除名）
每 6 拍（counter>5）：AppRecord 信息行
    '<info state="%s" connect="%d" ip_filter="%s" tick="%d" />'
    （ip_filter：BlockOrAllow 0=black/1=white/其余 none）
```

## 六、IP 黑白名单与运维件

- **CBlockIP**（BlockIP.obj）：AddBlockIP(beg,end) 点分十进制区间
  （reverse_ip 归一 u32 比较）；IsBlockIP + BlockOrAllow 三态
  （-1 不拦/0 黑名单命中拒/1 白名单未命中拒）。**accept 即拦**
  （OnAccept 断连 + 日志）——大写系 FsLister 的 OnBlockIP 恒 0 不拦。
- **黑白名单文件**（lua）：black_ip.txt/white_ip.txt，行格式
  `beg,end`（或 \t/空格分隔），存在即进对应模式（echo 台可动态切换）。
- **EchoHook 维护台**（EchoAddr:15400+ListerID）：welcome 行 + 调试器
  方言（getp/setp/getc/setc/method/listp/listc/listm/listg/function/
  debugger + quit/test/black/open/end）——与 fscomm EchoSock 同源，
  Go 侧不重复实现（分布式形态清单 §四既定）。
- **AppRecord/LogShareMem/LogLauncher**：进程记录/共享内存日志/拉起
  game_log 的官方进程监管生态，不复刻（单进程部署无对拍方）。

## 七、Lua 装配（Lua\sololister\main.lua + game_lister.ini）

```
lister.UseQuickLog=false  UseAppRecord=true  SetConsoleColor("LightCyan")
ini 默认值（ReadXxx 第三参）：
  IDC=1  DistrictID=1  ListerID=0  MaxConnects=1024  ClientVersion=0
  Outer 127.0.0.1:2001  Inner 127.0.0.1  Broadcast 127.255.255.255:15000
  Echo 127.0.0.1:(15400+ListerID)
lister.AppName = "lister_<id>";  LogLauncher = "game_log game_log.ini daemon"
lister:Start() → WriteLine 五连（app name/echo/inner/broadcast/outer）
  → "game lister server district D id I started" → SetAppState(4=OPENED)
black_ip.txt 存在 → BlockOrAllow=0；white_ip.txt 存在 → BlockOrAllow=1
```

快照 game_lister.ini 实值：IDC=0/DistrictID=1/ListerID=0/MaxConnects=
1024/Outer 192.168.33.70:4000/Inner 192.168.33.70/Broadcast
192.168.33.255:15000（注释缺省 127.255.255.255:4001——**以 lua 缺省
15000 为准**，ReadInteger 第三参即 15000）。

## 八、与大写系 FsLister.dll 差异总表（含旧记勘误）

| 维度 | FsSoloLister（本文） | FsLister（大写系） |
|---|---|---|
| 进程 | game_lister.exe 直启三件套 | FsLoader.exe 装载 |
| 模块版本 | **1003** | 1000 |
| 网络栈 | **内嵌 CReactor**（随 DLL 链接） | 挂 FsComm MassSock/BroadSock 实体 |
| 注册魔数 | **0x3001** | 0x300 |
| world 包长 | **0x84（多 recommend@+0x78）** | 0x78 |
| UDP 口 | **双口：本地环回中继 + 配置口** | 单 BroadSock |
| 线过滤字段 | **IDC**（实证 +0x34；~~ListerId~~ 旧记有误） | IDC |
| busy | recommend>10 直通，否则 cur*5/max | cur*5/max（无 recommend） |
| 选线初值 | **10.0f，严格小于（等值取先者）** | 1.0f，≤（等值取后者） |
| 行终结符 | **\r 和 \n 皆终结** | 仅 \n（\r 是控制符丢弃） |
| 行内控制符 | **缓冲不丢** | <0x20 全丢 |
| 行缓冲 | **0xFF**（client_t 内嵌 0x100） | 0x1000（hook 分配） |
| 请求超时 | **30s** | 60s |
| 版本门 | 有（ClientVersion/"svrlist <ver>"） | 无 |
| IP 黑白名单 | 有（accept 即拦） | 无（OnBlockIP 恒 0） |
| server_t | **0x4E8**（标志字节头 + 无锁） | 0x4FC（CRITICAL_SECTION 头） |
| 客户端表 | 动态 TArrayPod（无 0x2000 上限） | client_t[0x2000] 定长槽 |
| 日志方言 | add/remove **小写** | Add/Remove 大写 |
| 表上限 | 无硬上限（动态） | 256 服槽 |
| MaxConnects 默认 | ctor 8000 / lua 1024 | 1000 |

另：solo 系 svrlist 应答编码（text_encode_msg@0x1005AF80）与 golden
钉死的方言同构（internal/login/lister.go 早已按本 DLL 反汇编落地——
§12 的版本门/busy 推荐值/多服连排即源于此 DLL 的早期预研）。

## 九、Go 复刻（internal/fssololister + cmd/sololister）

复刻**语义**不复刻 IOCP/CReactor（internal/reactor 已按本 DLL 的
Reactor.obj 复刻，§14 既定原则）：

| 官方件 | Go |
|---|---|
| SoloLister 对象 + server_t/line_t | `fssololister.SoloLister/Server/Line`（含 Recommend、空槽索引表复用） |
| 双 UDP 口 + 中继 | `fssololister.RegistrationServer`（配置口全门 + 本地口中继；InnerAddr=127.0.0.1 单口，官方同款条件） |
| 收包门/报文 | `ParseRegistration/ParseLoopbackRegistration` + `BuildWorldPacket/BuildEntryPacket`（发送方供 world/entry 进程用） |
| RecvWorld/EntryServerInfo | `RecvWorldInfo/RecvEntryInfo`（entry 不建服/line_index<16/全量覆写） |
| TimerCheck 5s 双心跳 | `CheckServer` + `TickSeconds`（幂等，测试单拍驱动） |
| ReceiveClientMsg 行方言 | `ClientServer.serveConn`（\r\n 皆终结/控制符缓冲/0xFF 截断/空行不发） |
| 版本门 + 单请求 | `CheckVersionGate`（atoi 截断方言）+ `AcquireRequest/MarkAnswered` |
| CheckClient 30/5/1 | 连接 goroutine deadline（30s 请求/5s 滞留/到期即断） |
| SendServerList 选线 | `Entries()`（bestrate 10.0 严格小于 + IDC 过滤 + busy 公式）→ `login.EncodeServerList`（golden 同源） |
| CBlockIP | `AddBlockIP/ClearBlockIP/IsBlockIP`（BlockOrAllow 三态，accept 即拦） |
| Console 日志 | `Logf` 钩子（add/remove/connected/disconnected/ip blocked 官方串） |
| 进程载体 | `cmd/sololister`（game_lister.ini 内置释放 + 静态条目 3s 心跳合成） |

不复刻：EchoHook 调试台方言（fscomm 已有等价物）、AppRecord/LogShareMem
进程监管、LogLauncher、WinSocket 接口壳（Go net 自持）、CThread/
UseThread 分支（goroutine 等价）。

**测试**（fssololister_test.go，15 例全绿 + 实机冒烟）：编解码往返与
门清单（魔数 0x300/包长 0x78 大写系包必须拒）、双心跳除名/空槽复用、
选线（最小负载/等值先者/IDC 过制/busy recommend 直通）、版本门三态、
真 TCP 行方言（\r 终结/控制符入行静默/version_error 占用单请求位）、
30s/5s 超时、黑白名单三态、UDP 定向直达 + 127.0.0.2↔127.0.0.1 中继
全链、ctor 默认值/ListerId 钳制/GetState。实机：sololister.exe 释放
ini → svrlist 多服连排（recommend=99 直通）→ 跨分区包静默，全 PASS。
