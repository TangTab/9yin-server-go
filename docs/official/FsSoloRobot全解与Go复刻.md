# FsSoloRobot 全解与 Go 复刻

> 分析完成日期:2026-11-19。分析对象:`F:\Game\Online\9yin\BGserverCJ\Debug\FsSoloRobot.dll`(679,936 字节,2014-03-22)
> + `FsSoloRobot.pdb`(2,829,312 字节,**无 .map**)+ 宿主 `game_robot.exe` + `game_robot.ini` + `Lua\solorobot\main.lua`。
> 工具:`tmp_research/solorobot/`(pdbpub.py 自研 PDB publics 提取器 + rb_dis.py 注释反汇编器)。
> 结论先行:**FsSoloRobot 是 solo 集群的压测机器人客户端**——外连网关([Outer],缺省 :2001,即 FsSoloEntry 监听口),
> 每条连接模拟一个完整玩家生命周期:登录 → 选角/建角 → 入景 → 周期随机移动。这是**官方自带的负载发生器**,
> 也是网关客户端线路协议(EE 转义帧 + 单字节 msg id + QuickLZ 压缩)唯一的官方参照实现。

源码路径(来自 PDB 断言字符串,权威蓝图):
```
d:\blackgoldserverengine\serverengine\branches\server(20120807)\fssolorobot\solorobot.cpp
d:\blackgoldserverengine\serverengine\branches\server(20120807)\fssolorobot\solorobotstate.cpp
d:\blackgoldserverengine\serverengine\branches\server(20120807)\tools\echohook.cpp      ← CEchoHook(服务端 echo 控制台)
d:\blackgoldserverengine\serverengine\branches\server(20120807)\tools\reactor.cpp       ← CReactor(IOCP,随 DLL 自带)
d:\blackgoldserverengine\serverengine\branches\server(20120807)\utils\loadarchive.h     ← CLoadArchive(裸读游标)
```

---

## 一、破解方法:PDB-only 模块的通用符号法

FsSoloRobot 与 FsSoloGlobal 一样只有 PDB 没有 .map,但**有 .textbss 段**(增量链接产物),故 +0x12000 恒定偏移法在本 DLL 失效。通用定案(适用于本快照一切 PDB-only 模块):

1. **PDB publics 提取**(`tmp_research/solorobot/pdbpub.py`):极简 MSF 容器解析(超级块 magic 32 字节、字段从 0x20 起)
   → 逐流暴力走 CV 记录,选出"从流头干净走到流尾且 S_PUB32 最多"的流(globals 流,本 DLL 为 stream#8,3487 个 publics)。
   **S_PUB32 经验布局**:`reclen u16 | type u16=0x110E | seg u16 | pad u16 | off u32 | flags u16 | name\0`(name@+14)。
2. **段偏移**(以导出跳板 thunk 实测校验):
   - seg 2(代码)→ VA = `0x10000000 + off + 0x41000`(.text RVA;受 .textbss 前置影响,各 DLL 不同,须每次用导出表验证)
   - seg 0(rdata 字符串/常量)→ VA = `0x10000000 + off + 0xC7000`(.rdata VA)
   - 验证法:导出 RVA 是 `jmp rel32` 跳板,跟随跳板目标与 PDB 偏移相减,8 个导出全部恒差 0x41000 ✓
3. rb_dis.py(注释反汇编器)自动做 thunk(0x10041xxx 渐增区)二级解引用——增量链接 DLL 的虚调用/内部调用都经跳板。

---

## 二、进程形态与装配

### 2.1 宿主与配置

`GameRobot.bat` → `FsLoader.exe GameRobot.ini`(另有 FsRobot.dll 全客户端形态,**非本模块**,勿混淆);
solo 形态 `game_robot.exe` 读 `game_robot.ini`:

```ini
[System]
MainEntity=SoloRobot        ; ← 实体名
MainScript=main
[Module] fskit.dll / fscomm.dll / fssolorobot.dll
[Script] Path=..\lua\solorobot, Reload=1
[Main]
MaxPlayer=2000              ; → robot.MaxConnects(缺省 1000)
ConnectConcurrent=50        ; → 缺省 10
ConnectSpeed=100            ; → 缺省 50
MoveSeconds=5               ; → 缺省 5
; MessageEncodeKey=A5375CAE ; ← 注释掉;且 main.lua 读后从不赋值(DLL 也无对应属性)= 死配置,线路无加密
[Outer] Addr=192.168.33.70  Port=2001   ; 外连目标(网关)
; [Echo] 缺省 127.0.0.1:15009(lua 缺省)
```

### 2.2 Lua 装配(`Lua\solorobot\main.lua`,全文在手)

`main(robot)`:nx_create("IniDocument") 读自身 ini → 15 个属性赋值(MaxConnects/ConnectConcurrent/
ConnectSpeed/MoveSeconds/AccountPrefix(缺省"robot")/PasswordPrefix(缺省"robot")/OuterAddr(缺省127.0.0.1)/
OuterPort(缺省2001)/EchoAddr(缺省127.0.0.1)/EchoPort(缺省15009))→ `robot:Start()` → WriteLine 三连
(`echo socket a:p` / `outer socket a:p` / `game robot started`)。

echo 脚本侧(`echo_init` 系列回调)与其它模块同款调试控制台:
`on_open`(回 `welcome game robot\r`)/`on_close`/`on_input`(quit→关进程、end→断 echo、其它 `undefined command: %s`),
另挂 on_input_getp/setp/getc/setc/method/listp/listc/listm/listg/function/debugger 十一个实体调试命令。

### 2.3 模块注册

- `FxModule_GetType` → "FxModule";`FxModule_GetVersion` → **0x3EB = 1003**(与 FsSoloGlobal 同版);GetSpace → "SoloRobot"。
- SoloRobotEntCreator:Create → `core_mem::CoreNewDebug<SoloRobot>`;GetName/GetParent/GetSpace/Destroy 常规。
- 实体注册 15 属性(4 只读统计 + 11 可写配置)+ 8 方法(Start/Stop/SetConsoleColor/WriteLine/SendEchoLine/CloseEcho/
  GetConnectState/GetConnectWaitSeconds),字符串表 `0x100ca330..0x100ca464`。

---

## 三、SoloRobot 实体结构(this 相对偏移,反汇编实测)

| 偏移 | 类型 | 含义 |
|---|---|---|
| +0x00 | vtable | IEntity 派生(基 ctor 0x1005bb50) |
| +0x28 | void* | WinSocket 核心服务句柄(Init 里 `GetCore()->Create("WinSocket")`,失败→模块 Init 失败) |
| +0x2c | int | **模块状态**(0 Unknown/1 Opened/2 Paused/3 Closed,GetState 返回字符串) |
| +0x30 | int | MaxConnects(ctor 缺省 1000) |
| +0x34 | int | ConnectConcurrent(缺省 10) |
| +0x38 | int | ConnectSpeed(缺省 50) |
| +0x3c | int | MoveSeconds(缺省 5) |
| +0x40 | TFastStr(0x20) | AccountPrefix |
| +0x60 | TFastStr | PasswordPrefix |
| +0x80 | TFastStr | OuterAddr |
| +0xa0 | int | OuterPort |
| +0xa4 | TFastStr | EchoAddr |
| +0xc4 | int | EchoPort |
| +0xc8 | CReactor* | 自带 IOCP 反应器(随 DLL 编入,不走 FsComm) |
| +0xcc | CEchoHook* | echo 控制台(ctor 参数 = this + reactor) |
| +0xd0 | int | 检查定时器 id(ctor -1) |
| +0xd4 | int | 起连定时器 id(ctor -1) |
| +0xd8 | int | **total_raised**:累计发起连接数(单调,不回落) |
| +0xdc | int | connect_count:TCP 建立成功数(ConnectCount 属性) |
| +0xe0 | int | failed_count:连接失败数(FailedCount 属性) |
| +0xe4 | TBufferList | 解压缓冲(qlz 解压目标) |
| +0xe8 | TArrayPod\<connect_t*\> | m_Connects(按 connector_id 索引,槽可空) |

**connect_t(分配 0x1030 字节)**:

| 偏移 | 类型 | 含义 |
|---|---|---|
| +0x00 | int | **index:机器人序号**(账号名 = prefix + index;由 total_raised 赋值) |
| +0x04 | int | 连接级状态 |
| +0x08 | int | wait_seconds(计时秒数;ingame 之外也累加;ingame 时作移动倒计时) |
| +0x10 | PERSISTID(8B) | 自身对象标识 {ident1, ident2}(entry_scene 报文写入,后续按此匹配自身) |
| +0x18/0x1c/0x20 | f32 | 当前坐标 x/y/z |
| +0x24 | 0x1000B | 收包解转义缓冲 |
| +0x1024 | int | recv_len |
| +0x1028 | u8 | last_byte(转义状态机上一字节) |

**连接级状态常量**(GetConnectState 跳转表 0x1005ca4c 实测):

```
0 unknown   1 connected   2 logging   3 creating   4 choosing   5 ingame   6 failed   7 closed
```

**统计口径**:ConnectCount=+0xdc;InGameCount=遍历数 state==5;FailedCount=+0xe0;
`GetConnectWaitSeconds(i)` = m_Connects[i]->wait_seconds;`GetConnectState(i)` 越界/空槽返回 ""(0x100c82db 空串)。

---

## 四、生命周期

### 4.1 Init / Start / Stop / Execute

- `Init(VarList)`:创建 WinSocket 服务存 +0x28,空→false(模块加载失败)。
- `Start()`:
  1. `reactor->Start()` 失败 → trace `(SoloRobot::Start)start reactor failed` → false;
  2. EchoAddr **非空**才 `echohook->Start(EchoAddr.c_str(), EchoPort)`(监听 echo 控制台);
  3. `reactor->AddTimer(1.0f, TimerCheck, this)` → +0xd0;
  4. `reactor->AddTimer(float(ConnectConcurrent)/float(ConnectSpeed), TimerRaise, this)` → +0xd4
     (即恒定 ConnectSpeed 连接/秒的节流;缺省 10/50=0.2s);
  5. `RobotStateMachine(event=1, 0, 0)` → 'connect server opened' → 模块态 Opened。
- `Execute(elapsed)` 每帧:`reactor->EventLoop()`(IOCP 泵,宿主 Sleep=1ms 高频驱动)。
- `Stop()`:DeleteTimer(+0xd0) → DeleteTimer(+0xd4) → echohook->Stop() → reactor->Stop()。
  (注意:Stop **不**迁移模块状态,Closed 态实际不可达,同构保留。)
- `Shut()`:析构清理。
- `WriteLine(s)`:`printf("%s\r\n", s)`;`SetConsoleColor(name)`:app_set_console_color(控制台色);
  `SendEchoLine(s)`→echohook->SendLine;`CloseEcho()`→echohook->Close。

### 4.2 模块状态机(solorobotstate.cpp)

| 状态 \ 事件 | 0=TimerCheck(每秒) | 1=Start/Resume | 2=Pause/Close |
|---|---|---|---|
| Unknown(0) | 忽略 | →Opened,'connect server opened' | trace 'Connect server already closed' |
| Opened(1) | **移动循环(见 §六)** | trace 'Connect server already opened' | →Paused,'connect server paused' |
| Paused(2) | 忽略 | →Opened,'Connect server resumed' | trace 'Connect server already paused' |
| Closed(3) | 忽略 | trace 'error input' | trace 'Connect server close all' |

未知事件:trace `(SoloRobot::RobotState%s)error input %d`(状态名替换 %s)。

### 4.3 TimerRaise(起连调度)

```
while (total_raised < MaxConnects):
    for (i = 0; i < ConnectConcurrent && total_raised < MaxConnects; i++):
        id = reactor->CreateConnector(OuterAddr, OuterPort,
                  send_buf=0x2000, recv_buf=0x1000, ?=0x2000, 0,
                  OnConnect, OnClose, OnReceive, this, 0)
        id<0 → trace 'create connector failed';break
        id≥size → m_Connects.resize(id+1, NULL)
        槽非空 → trace 'connect already exists';reactor->DeleteConnector(id);break
        ct = new(0x1030) connect_t;memset 0;ct->index = total_raised++
        m_Connects[id] = ct
        WriteLine('raise connect address %s:%d, total %d', addr, port, total_raised)
```

### 4.4 OnConnect / OnClose(连接事件)

- **失败**(第 2 参==0):delete connect_t、槽置 NULL、failed_count++、
  `WriteLine('connect address %s:%d failed, total %d')`(total=failed_count)。**无自动重连**——total_raised 不回落,
  每个槽一次性发起;失败即终局(槽空但不再补充)。
- **成功**:connect_count++、`WriteLine('connect address %s:%d succeed, total %d')`(total=connect_count)、
  **send_login**(§五)、SetConnectState(id, 2 logging)。
- **OnClose**:delete connect_t、槽置 NULL、`WriteLine('address %s:%d closed')`(计数不动)。

---

## 五、线路协议(核心定案)

### 5.1 帧法(双向,与 internal/net.EscapeFramer 同款)

- 发送(`SendToServer`):payload ≤0x1FE;逐字节 0xEE → 输出 `EE 00`;末尾追加 `EE EE`;
  转义后总长 ≤0x400(assert `count <= sizeof(buffer)`);`reactor->Send(id, buf, len, true)`。
- 接收(`OnReceive`):逐字节状态机(last_byte@connect+0x1028):
  - b==EE 且 last==EE → **帧完成**:`ReceiveServerMsg(id, buf, len-1)`(去掉终止符首 EE),len=0,last=0;
  - b==00 且 last==EE → 丢弃(转义还原);
  - 否则 len<0x1000 则 buf[len++]=b;
  - last=b。
- **无长度前缀、无加密**(MessageEncodeKey 是注释掉的死配置)。

### 5.2 机器人 → 服务器(首字节 = msg id)

| id | 报文 | 长度 | 布局 |
|---|---|---|---|
| 0x02 | **login** | 0x15+Σ字段 | `[0]=02 [1..5]=int32 0x300(版本)` + 3×`[dword align16(len+1)][ascii+NUL,零垫]`(账号=`prefix+index`、密码=`pwdprefix+index`、登录串="") |
| 0x04 | **choose_char** | 0x54 | `[0]=04 [1..]=UCS2 角色名`(2*len+2≤0x42 否则截 0x40 字节+零字),其余 memset 0 |
| 0x05 | **create_char** | 0x49 | `[0]=05 [1..5]=int32 0 [5..7]=u16 0 [7..0x49)=UCS2 名字(0x42 字段)` 名字=`AccountPrefix+index` |
| 0x09 | **entry_scene 应答** | 1 | `[0]=09` |
| 0x0B | **move** | 0x60 | `[0]=0B [1]=u8 1 [2..4]=u16 6 [4]=f32 x [8]=f32 y [12]=f32 z(当前位置) [16]=f32 dest_x=x+Δ [20]=f32 dest_z=z+Δ [24]=f32 4.0` 其余零;Δ=`rand()*10.0/32767.0-5.0`(x/z 独立) |

login 的字符串字段:长度域= `(strlen+1+15)&~15`(16 字节对齐),数据域含 NUL 零垫到对齐长。
名字 UCS2 = UTF-16LE 含结尾 L'\0'。

### 5.3 服务器 → 机器人(ReceiveServerMsg,switch msg[0]-3 ∈ [0,0x1e])

| id | 报文 | 最短长 | 语义 |
|---|---|---|---|
| 0x03 | error_code | 5 | `int32 code@1`;trace `robot %s%d receive error code %d`;→state 6 failed(终局) |
| 0x04 | login_succeed | 0x29 | `int32 char_count@0x25`;见下 |
| 0x0B | entry_scene | 0xF | `ident1=int32@5 ident2=int32@9` → 写 connect PERSISTID;回发 1 字节 0x09;trace `robot %s%d entry scene`;→state 5 ingame |
| 0x0D | add_object | 0x3F | `ident1@1 ident2@5 x@9 y@0xD z@0x11(f32)`;匹配自身→更新坐标 |
| 0x1F | location | 0x19 | 字段同 add_object;匹配自身→更新 x/y/z |
| 0x20 | moving | 0x2D | 字段同上;匹配自身→只更新 x/z |
| 0x21 | all_dest | 0x2F | `u16 count@1`;条目 0x2C 字节自 @3:`ident1@+0 ident2@+4 x@+8(f32) z@+0x10(f32)`;匹配自身的条目→更新 x/z |
| 0x28 | **压缩包裹** | — | `qlz_size_decompressed(p+1)`≤0xFFFF;`qlz_decompress(p+1, 解压缓冲+1)`;**缓冲[0] 强制 0x0D** → 即:0x28 包裹的解压体是 **add_object 的报文体(无 id 字节)**,回主线分发 |
| 其它 | — | — | 忽略,返回 true(不断链) |

**login_succeed 处理**:
- `char_count > 0`:CLoadArchive.Seek(0x29) 读第一个角色条目:
  `word@0x29, byte, int32, byte, int32, byte, UCS2(角色名: u32 字节长+UTF16LE 含 L'\0'), byte, UCS2`;
  发 0x04 choose_char(该名)→ state 4 choosing。
- `char_count ≤ 0`(无角色):构造名字 `AccountPrefix+index`(TFastStr 宽串→UCS2),发 0x05 create_char → state 3 creating。
- 解析越界/异常(CExceptMsg,如 'Out range'/'UCS2 string no end symbol')→ state 6 failed + trace。

CLoadArchive 读取原语:ReadWord=2B、ReadByte=1B、ReadInt=4B、ReadUCS2=`[u32 字节长][数据]`(要求末尾 wchar==0,否则抛 'UCS2 string no end symbol');Seek=设游标。

### 5.4 协议状态流(单连接)

```
connected(TCP) → [发 0x02] logging → login_succeed:
    有角色 → [发 0x04 choose] choosing ─┐
    无角色 → [发 0x05 create] creating ─┤
                                        ↓
              entry_scene(0x0B, 带自身 PERSISTID) → [发 0x09] ingame
              途中 add_object/location/moving/all_dest 维持坐标
              error_code(0x03) → failed(终局)
```

---

## 六、移动模拟(Opened 态 TimerCheck,每秒)

```
for ct in m_Connects where ct != NULL:
    ct->wait_seconds++
    if ct->state == ingame && ct->wait_seconds > MoveSeconds:
        ct->wait_seconds = 0
        dest_x = ct->x + (rand()*10.0/32767.0 - 5.0)
        dest_z = ct->z + (rand()*10.0/32767.0 - 5.0)
        SendToServer(0x0B move: cur(x,y,z) → dest, 4.0)
```
即:入景后每 MoveSeconds 秒向随机 ±5 米方向走一步(y 恒 4.0,目标点不带 y)。

---

## 七、echo 控制台(CEchoHook,服务端形态)

`tools/echohook.cpp` 家族实现(与 FsSoloLog/sologlobal 文档同模式,本 DLL 自带):
- `Start(addr, port)` → reactor->CreateListener;失败 trace `(CEchoHook::Start)create echo listener failed`。
- **单客户端**:OnAccept 时已有客户端 → DeleteConnector 拒绝后来者;否则回调实体 `on_open(addr, port)`。
- OnReceive:行缓冲,`\r`/`\n` 终结 → `ReceiveInput(line)`。
- ReceiveInput:parse_string/parse_token(引号切换、int→float→宽串→PERSISTID→string 试型)分词;
  回调实体 `on_input_<cmd>`(小写)→ 回退 `on_input`;Lua 侧应答(欢迎语 `welcome game robot\r`、quit/end)。
- SendLine/Close 对应 Lua 的 SendEchoLine/CloseEcho。

---

## 八、Go 复刻设计(internal/fssolorobot + cmd/robot)

**定位**:对内压测/回归工具——为 Go 版 solo 集群(fssoloentry 网关 + soloworld 等)提供官方语义的机器人负载源,
同时是网关客户端线路协议的 Go 参照实现。

### 8.1 复用

| 需求 | 现成件 |
|---|---|
| EE 转义帧(EE 00 转义 / EE EE 终止) | `internal/net.EscapeFramer`(语义逐字一致,robot 上限用 0x1000/0x400 覆写) |
| echo 控制台 | 既有各模块 EchoConsole 模式(fssologlobal/fssoloentry 同款) |
| IOCP/reactor | 不复刻——Go 用 goroutine + net.Dial,每连接一读循环 |
| trace | 沿用各模块 trace 风格(CORE_TRACE 等价物) |

**QuickLZ 解压已随本包落地(internal/fssolorobot/qlz.go,反汇编终局而非猜算法)**——DLL 内嵌的是
**QuickLZ level-2** 位流(_qlz_decompress@0x10058d60 → 核 0x10058e80 + 哈希插入 0x10059250):
- 头两种:`flags&2` → 9B(`[0]=flags [1..5)=csize u32LE [5..9)=dsize u32LE`);否则 3B
  (`[1]=csize u8 [2]=dsize u8`)。`flags&1`=压缩体,否则原样存储(memcpy 自 src+头长)。
- 位流:标签字 cword(=1 时取 src 处 u32LE 续 4 字节);`cword&1==1` → 匹配
  (control=窥视 u32:hash=(c>>4)&0xFFF,nibble≠0→len=nibble+2 消 2 字节,否则 len=src[2] 消 3 字节;
  从哈希表指针按 3 步 u32 前向重叠拷贝);`==0` → 字面量连跑(n=尾零数上限 4,批表
  `[4,0,1,0,2,0,1,0,3,0,1,0,2,0,1,0]`,每 0 位 1 字节)。
- 哈希:`h=((v>>12)^v)&0xFFF`,v=位置 3 字节 u32LE;惰性回填(计数器推到产出位-3,逐位插入)。
  示例:hash("abc")=0x457。
- 距末尾 ≤10B 进尾道(逐字面量;cword==1 时仅 src+=4 且置哨兵 0x80000000)。
- **实现时定案的坑**:标签字位耗尽(cword 移位到恰为 1)会先续读下一标签字再窥 control——
  手工向量必须把跨标签字的令牌排在数据流之外([tag0][data][tag1][control] 布局)。

### 8.2 结构(建议)

```
internal/fssolorobot/
  robot.go       Robot 实体:15 属性、8 方法、模块状态机(4 态 3 事件表驱动)
  conn.go        连接级:state 常量(0..7)、connect{index,state,wait,id,x,y,z,recv状态机}
  sched.go       调度:raise ticker(ConnectConcurrent/ConnectSpeed 节流)+ check ticker(1s 移动循环)
  proto.go       send:login/choose/create/ack/move 五构造器 + recv:7+1 分发器(表驱动 msg id)
  qlz.go         QuickLZ 解压(0x28 → add_object 体)
  echo.go        echo 控制台(欢迎语/quit/end + 调试命令可裁剪)
  robot_test.go  全链:fssoloentry 或假网关上跑 login→create→entry→move 状态流 + EE 帧回归 + qlz 向量
cmd/robot/       -config game_robot.ini 兼容官方键(Main/Outer/Echo 段)
```

### 8.3 保真要点(易错点)

1. **账号名 = AccountPrefix + 连接序号**(不是 connector_id;序号由 total_raised 单调分配)。
   login 报文定长头 0x15(含 [5..0x15) 16 字节零区),三字段自 +0x15 起追加。
2. login 字符串字段 16 字节对齐(`(len+1+15)&~15`),长度域在数据域前。
3. choose/create 名字是 **UCS2(UTF-16LE 含 L'\0')**,定长字段(0x42/0x40+零字),报文定长 0x54/0x49。
4. move 的 Δ 用 `rand()*10.0/32767.0-5.0`(C rand 上限 32767;Go 用 math/rand.Int31n 等价分布即可)。
5. 0x28 压缩包裹只承载 add_object 体;解压后 buf[0]=0x0D 再分发(解压体**不含** id 字节)。
6. error_code → failed 终局、**不重连**;OnClose 只清槽;total_raised 不回落。
7. 双 ticker:check=1s(移动+等待计数),raise=ConnectConcurrent/ConnectSpeed 秒(每次触发循环最多 ConnectConcurrent 个)。
8. 帧发送 payload ≤0x1FE、转义后 ≤0x400;接收缓冲 0x1000;无加密。
9. 统计三数含义:ConnectCount=成功数、FailedCount=失败数、total(d8)=累计发起(InGameCount=遍历统计)。

---

## 九、遗留/未复刻项

- `FsRobot.dll`(GameRobot.ini 全客户端形态,MaxPlayer=1024/MsgEncodeKey 可配)是**另一个模块**(真客户端逻辑),
  本文档不覆盖;若需"官方完整客户端行为"须另行分析。
- CReactor 的 IOCP 细节(overlap_plus/broadcast/UDP 等)不复刻——robot 只用作连接器+监听器两角色,Go 原生覆盖。
- OnAccept 拒后来者的具体包文/时序以 echo 控制台既有实现为准。
- fastlz 也编入了本 DLL但 robot 路径未用(仅 qlz),无需复刻。

## 十、Go 复刻落地记录(2026-11-19)

- `internal/fssolorobot`:qlz.go(QuickLZ L2 解压)/ proto.go(五发报文构造+EE 转义)/
  conn.go(连接级八态+官方解转义状态机)/ recv.go(CLoadArchive+7+1 分发)/ echo.go(单客户端
  echo 控制台)/ robot.go(实体/模块状态机/双定时器调度/移动循环)/ config.go(game_robot.ini
  方言+内置默认)。
- `cmd/robot`:-config/-outer/-echo off/-max/-run;echo quit 触发进程退出(官方 nx_quit 语义)。
- 测试 34 项全绿:qlz 存储态/字面量快速道/尾道/手工匹配向量(hash 0x457)/EE 帧分片与宽容语义/
  golden 报文/分发全分支/net.Pipe 假网关全链(login→create→entry ack→location→move)/
  拨号失败计数/服务端断开清槽。全仓 `go build ./... && go test ./...` 通过。
