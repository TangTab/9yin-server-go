# FsLister.dll 全解与 Go 复刻

> 标的：`E:\Workspace\project\game\九阴真经ol\BGserverCJ\Debug\FsLister.dll`
> （602 112 B，随快照带 FsLister.pdb 2.5MB 全符号，**无 .map**）
> 方法：tmp_research/fslister/fl_dis.py（capstone + PDB 符号**节重定位**直读）
> 产物：`internal/fslister`（Go 复刻）+ cmd/lister 集群形态接线
> 更新：2026-11-19
> 姊妹篇：FsSoloLister.dll（小写系）已全解剖落码——见
> FsSoloLister全解与Go复刻.md（internal/fssololister + cmd/sololister）。

---

## 一、模块定位

FsLister.dll 是**大写黑金 Fs 体系**的区服列表服务器实体类模块
（GameLister.bat → FsLoader.exe GameLister.ini → FsKit+FsComm+FsLister 三件套）。
与 solo 系的 FsSoloLister 同源分工，但目标是**多服集群聚合**：
world/entry 进程通过 UDP 广播持续注册自己，客户端连 :4000 要 `svrlist`，
应答从注册表里选每服**负载率最低的线路**并聚合繁忙度。

- 源码路径（断言串）：`d:\flexis\client\fslister\gamelister.cpp`、
  `d:\flexis\client\fslister\broadsockhook.cpp`、`masssockhook.cpp`（3 个 obj）
- 导出：8 个标准 FxModule（GetType=**"FxModule"**/GetVersion=**1000**/
  GetSpace=""/Init=内核指针 → g_pCore 0x100cd758）
- 实体注册：GameListerEntCreator（GetName="GameLister"，GetParent="IEntity"，
  GetSpace=""），Create → `new GameLister`（对象 0xA7C60 字节）

### 实体属性/方法（GameListerEntCreator_Init 静态注册，CRT 自举）

| 名 | 类型码 | 说明 |
|---|---|---|
| IDC / DistrictId / ListerId / MaxConnects | 2(int) | getter/setter 双向 |
| Console / EchoSock / MassSock / BroadSock / AppManager | 8(object) | setter 走 QueryInterface 按类型名接线 |
| Start() / Stop() | 方法 | 脚本可调（game_lister:Start()） |

对象 setter 接线协议（以 MassSock 为例）：`FindEntity(id) →
QueryInterface("MassSock")` 成功 → m_pMassSock=接口指针，且
`CMassSockHook::NewInstance(this)` 创建挂载 hook 并
`MassSock->SetHook(hook)`。Console/EchoSock/BroadSock/AppManager 同款，
仅 BroadSock 再加 `CBroadSockHook::NewInstance`。

### Lua 装配（组合根，Lua\GameLister\main.lua + 官方 GameLister.ini）

```
LogPool(log\lister 前缀,TraceMax 256) → Console(debug 时)
EchoSock([Echo] 默认 127.0.0.1:15004, echo.lua 调试器)
MassSock([Outer] Addr:Port; ConnectMax 8192/WorkThreads=[Main]SockThreads/
         InBufLen 16K/OutBufLen 64K/ListenPendings 20)
BroadSock(BindAddress=[Inner] 默认 127.0.0.1:16004, BroadcastAddress=[Broadcast]
          默认 127.255.255.255, BroadcastPort 默认 15000)
AppManager(AppName="lister_<ListerID>", AppAddress=echo 地址)
GameLister(IDC/DistrictId/ListerId/MaxConnects/五实体指针) → :Start()
```

启动顺序：AppManager:Start → BroadSock:Start → MassSock:Start(Closed=true
暂不接受) → EchoSock:Start → GameLister:Start → `mass_sock.Closed=false`。
关停对称（close()：Closed=true → CloseAll → 等 ConnectCount 归零 → Stop 链）。

ini 默认值（lua ReadXxx 第三参）：IDC=1/DistrictID=1/ListerID=0/
MaxPlayer=1024/Outer 127.0.0.1:2001（快照 ini 写 192.168.106.20:4000）/
Inner 127.0.0.1:16004/Broadcast 127.255.255.255:15000/Echo 127.0.0.1:15004。

## 二、PDB 增量链接残骸的破解（工具沉淀）

FsLister.pdb 可读但符号 VA 全落在 0x0~0x8xxxx（.textbss 虚区间）——
FsEntry.pdb 同款问题的姊妹篇。实证规律：**PDB 存节内偏移**，
按 DLL 节表重定位即可用（tools/pdb_syms.py 已产出 seg，打印侧丢弃）：

| 节 | PDB 偏移 → 实际 RVA | 锚点验证 |
|---|---|---|
| .text (VA 0x39000) | off + 0x39000 | 8 个 FxModule 导出 thunk 全中（4/4） |
| .rdata (VA 0xaf000) | off + 0xaf000 | "svrlist"/"GameLister" 等 5 串全中 |

fl_dis.py 另带：代码符号邻接推断函数长度（本 PDB PROC32 len=0）、
thunk 表（0x10039000 区 jmp rel32 密集区）批量解析、常量池字符串注释。

## 三、GameLister 对象布局（0xA7C60）

```
+0x000  vtable (IEntity 派生)
+0x024  int  IDC                    (SetIDC@10042FB0)
+0x028  int  DistrictId             (UDP 注册包的准入门)
+0x02C  int  ListerId
+0x030  int  MaxConnects = 1000 默认(ctor)
+0x034  void* m_pConsole            (IMainConsole, 日志)
+0x038  void* m_pEchoSock
+0x03C  void* m_pMassSock           (IMassSock)
+0x040  void* m_pBroadSock
+0x044  void* m_pAppManager
+0x048  void* m_pMassSockHook       (SetMassSockID 时 NewInstance)
+0x04C  CThread                     (ListerThreadProc → DoWait 循环)
+0x050  CMsgDeque2                  (server 侧消息队列；PutMessage 入队)
+0x054  CTickTimer                  (GetElapseMillisec 定 1s 节拍)
+0x058  client_t[0x2000] × 0x2C     (8192 客户端槽)
+0x58058 int  client 高水位
+0x5805C server_t[0x100] × 0x4FC    (256 服槽)
+0xA7C5C int  server 计数(高水位)
```

### client_t（0x2C；首 0x18 = CRITICAL_SECTION）

```
+0x18 byte valid      +0x1C int client_id(=槽下标)
+0x20 int conn_id(串号) +0x24 int state +0x28 int counter
```

### server_t（0x4FC；首 0x18 = CRITICAL_SECTION）

```
+0x18 byte valid      +0x19 byte visible(来自 world 包 +0x74)
+0x1A byte from_world +0x1C int server_id(FindServer 键)
+0x20 int world_latch +0x24 int max +0x28 int cur
+0x2C char ip[16]     +0x3C wchar name[32]      ← 应答里的 #宽名
+0x7C wchar name2[256](预留，AddServer/RecvWorld 后清空)
+0x27C line_t[16] × 0x28:
    +0x00 byte valid  +0x04 int type闩锁(1=entry)  +0x08 int line_id
    +0x0C int max     +0x10 int cur   +0x14 int port  +0x18 char ip[16]
```

## 四、:4000 客户端面（CMassSockHook + GameLister 客户端机）

**线路方言（MassSockHook::OnReceive@10048F30 + ReceiveMsg@10049070）**：

- 逐字节状态机：`0x0A '\n'` 终结一行 → 非空则分发；**<0x20 的控制字符
  （含 \r）直接丢弃不缓冲**；行缓冲 0x1000（4 KiB，溢出静默截断）
- 分发前净化：把行内残余控制字符替换为 NUL，再 `strcmp(line,"svrlist")`
  精确匹配——**其余行一概不理**（ReceiveClientMsg@10044160）
- 应答（SendCustomMsg@10049240）：text_encode_msg(VarList, buf, 0x2000)
  → MassSock::Send；**每行尾追 `\r\n`**（MakeSendData@10048B20 手工补
  0x0D 0x0A；GetMaxSendSize=容量-2、GetSendLength=len+2 配套）

**客户端状态机（CheckClient@10047390，5s 扫描周期驱动）**：

```
state 0 连接待请求: counter++ ≥60(60s) → state 2   # 不应答直接进入关闭
state 1 已应答:     立即 SendServerList → state 2   # 由请求处理即时迁移
state 2 关闭中:     counter++ ≥5(5s) → MassSock 关连接 → state 3
```

- ReceiveClientMsg：conn_id 串号校验（过期事件拒收）+ **每连接只应答一次**
  （state≠0 拒绝重复请求）
- OnAccept：AddClient(client_id, conn_id, addr) + 日志
  "Address %s:%d connected, total %d"（总数=MassSock ConnectCount）
- OnClose：conn_id 匹配才 RemoveClient；OnBlockIP 恒 0（不拦——黑名单在
  FsComm MassSock 实体层）；OnError/OnSend 恒 1
- CreateContext：每连接 0x1004 B（0x1000 行缓冲 + 4B 长度）

**应答构造（SendServerList@10044320）**：

```
VarList = [AddString("svrlist")]            # $svrlist（窄串！）
for 每个 valid && visible 的 server:
    best=-1, bestrate=1.0
    for line in 16 槽:
        line.max==0 或 line 无效 → 跳过
        IDC>0 且 line.line_id≠IDC → 跳过        # ⚠ IDC 过滤（非 ListerId）
        rate = line.cur/line.max (float)
        rate ≤ bestrate → best 换新（取最小负载率；相等取后者）
    best<0 → 该服不出现在应答
    busy = server.max>0 ? server.cur*5/server.max : 0
    AddWideStr(server.name)  # 宽名
    AddString(line.ip)       # 通告地址（窄）
    AddInt(line.port) AddInt(busy)
```

即：**应答 = `$svrlist (#名 $ip 端口 busy)×N`**——与 solo golden 方言
逐字节同构（internal/login.EncodeServerList 直接复用）；text_encode_msg
两版本指纹同构（0x20/0x25/0x5c/0x100 同款分支 + 手写转义助手 0x1003ff50）。

## 五、:15000 UDP 注册面（CBroadSockHook + server 表）

**报文门（OnReceive@10040840）**：

```
len < 8 → 丢弃
pkt[+4](u32) ≠ 0x300 → 丢弃            # 魔数/版本
pkt[+0x1C](u32) ≠ GameLister.DistrictId → 丢弃   # 分区准入门
kind = pkt[+0](u32):
  0 = world 信息包，len 必须 == 0x78 (120)
  1 = entry 信息包，len 必须 == 0x48 (72)
```

**world_info（120B）**：`+0x20 server_id`、`+0x24 wchar name[≤32]`、
`+0x08 char ip[16]`、`+0x6C max`、`+0x70 cur`、`+0x74 visible`。
RecvWorldServerInfo@10046BD0：FindServer(id) 未命中 → AddServer（首个
空槽，valid=1、id、memset line[16]；256 满则拒绝）；然后 from_world=1、
world_latch=1、visible=包[+0x74]≠0、cur/max、ip、name 拷入；
新增时 Console 日志 "Add server %s(%s)"（name 的 ToString 窄化 + ip）。

**entry_info（72B）**：`+0x20 server_id`、`+0x24 line_index(<0x10 必须)`、
`+0x28 line_id`、`+0x2C char ip[16]`、`+0x3C port`、`+0x40 max`、`+0x44 cur`。
RecvEntryServerInfo@100471D0：FindServer 未命中→**丢弃**（entry 先于
world 注册不建服）；line_index ≥16 丢弃；然后 line.valid=1、type=1（闩锁）、
line_id/max/cur/port/ip 全量覆盖。**server_id 靠 world 包建立，entry 只补线**。

**生命周期（CheckServer@100475D0，5s 扫描）**：

```
for line in 16: line.valid && line.type==0 → line.valid=0
                else line.type=0            # entry 每 5s 必须重发，否则线死
if from_world:
    world_latch==0 → 服死：日志 "Remove server %s(%s)"、valid=0（槽回收）
    else world_latch=0                      # world 每 5s 必须重发
```

即注册是**双心跳**：world 心跳（服级，丢失一次即除名）+ entry 心跳
（线级，丢失一次即摘线）。主循环（DoWait@100479C0）：消息泵
（CMsgDeque2 swap+peek，ProcessMessage 空实现=纯预留）→
CTickTimer 每 1000ms 一拍 → 每 5 拍全表 CheckServer + CheckClient。

## 六、与小写系 FsSoloLister 的差异清单

> 2026-11-19 勘误补记：本表初版把 Go 静态形态（login.Lister + -extra）
> 误当 FsSoloLister 本体方言（"无注册通道/单线直出/ListerId 过滤/
> 0x3FF 行缓冲/请求即断"皆非 DLL 实情）。FsSoloLister.dll 全解剖后
> 差异表已全面重写，见 **FsSoloLister全解与Go复刻.md §八**；要点：
> solo 系有 UDP 双口中继注册面（魔数 0x3001/world 包 0x84 含
> recommend）、客户端机 30s/5s、行方言 \r\n 皆终结且控制符不丢、
> 线过滤同为 IDC、黑白名单 accept 即拦。本表保留作历史对照。

| 维度 | FsSoloLister（全解剖后实证） | FsLister（本文） |
|---|---|---|
| 形态 | game_lister.exe 直启三件套（内嵌 CReactor） | FsLoader 装载（挂 MassSock/BroadSock 实体） |
| 注册通道 | UDP 双心跳 + 本地环回中继口 | UDP 双心跳（world/entry 分包） |
| 魔数/包长 | 0x3001 / world 0x84（含 recommend） | 0x300 / world 0x78 |
| 选线 | 16 线最小 cur/max（bestrate 10.0，等值取先者） | 16 线最小 cur/max（bestrate 1.0，等值取后者） |
| busy | recommend>10 直通，否则 cur*5/max | 服级 cur*5/max 聚合 |
| 线过滤 | IDC（+0x34 实证） | **IDC**（+0x24 实证） |
| 客户端机 | 30s 请求超时 + 5s 滞留 + 1s 回收 + 单请求 + 版本门 | 60s 请求超时 + 5s 温和关闭 + 单请求 |
| 行缓冲/终结 | 0xFF，\r 与 \n 皆终结，控制符不丢 | 0x1000，仅 \n，<0x20 丢弃 |

**分布式形态清单.md 悬案销项**：① 注册报文头（kind/magic 0x300/
district 门）与两种包长（120/72）——此前只知 payload 偏移；② 选线过滤
字段 = IDC 非 ListerId；③ 扫描周期 = 精确 1s 拍/5s 扫（此前"秒级"估读）。
server_t/line_t 布局与清单记载完全一致（互证）。

## 七、Go 复刻（internal/fslister）

复刻**语义**不复刻 IOCP/CReactor（§14 既定原则；Go netpoller 等价）：

| 官方件 | Go |
|---|---|
| GameLister 对象+表 | `fslister.GameLister`（Server/Line/Client 表，锁粒度=官方 CRITICAL_SECTION 等价） |
| BroadSockHook UDP 门 | `fslister.RegistrationCodec`（ParseWorldInfo/ParseEntryInfo + 魔数/分区/长度校验） |
| RecvWorld/EntryServerInfo | `GameLister.RecvWorldInfo/RecvEntryInfo`（AddServer/FindServer 语义） |
| CheckServer/CheckClient 5s 扫 | `GameLister.Tick()`（幂等，测试可单拍驱动） |
| CMassSockHook 行协议 | `fslister.ClientConn`（逐字节状态机/控制符丢弃/0x1000 截断/单请求/CRLF） |
| SendServerList | 选线最小负载率 + busy 公式 → `login.EncodeServerList`（golden 同源） |
| 60s/5s 超时 | conn 读写 goroutine + deadline |
| Console 日志 | 标准库 log（CORE_TRACE 语义） |
| EchoSock/AppManager/LogPool | 不重复实现——fscomm 已有（§17），单服部署不接线 |

不复刻：CThread/CMsgDeque2/CTickTimer 实现细节（Go goroutine/ticker
等价）、ProcessMessage 空预留、进程监管生态。
