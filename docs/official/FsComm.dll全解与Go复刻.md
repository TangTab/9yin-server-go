# FsComm.dll 全解与 Go 复刻

> 标的：`F:\Game\Online\9yin\BGserverCJ\Debug\FsComm.dll`（798 720 B，
> 2013-07-05 编译，随快照带 FsComm.map 4 536 符号 + FsComm.pdb）
> 方法：tools/dis_entry.py（--dll/--map 参数化，capstone + map 符号直读）
> 产物：`internal/fscomm`（Go 复刻，11 测试全绿）
> 更新：2026-11-19

---

## 一、模块定位

FsComm 是 Fs 系服务器的**共享通讯组件层**：导出标准 FxModule 接口
（8 个：GetEntCreator/GetFuncCreator/GetIntCreator/GetLogicCreator/
GetSpace/GetType/GetVersion/Init），向实体系统注册一批网络/运维
实体类，由 Lua 装配脚本或宿主 C++ 按名 `nx_create("<类名>")` 创建。

源码路径泄露（断言串）：`c:\wuyf\svn\serveriii\branches\publish20130508\server\fscomm\`

**使用方普查**（Debug 全目录二进制 null 结尾字符串命中）：

| 使用方 | 用到的实体类 |
|---|---|
| FsEntry.dll | Adapter, MassSock, EchoSock, BroadSock, AppManager |
| FsMember/FsPublic/FsStore | Adapter, EchoSock, BroadSock, AppManager(±LogicLoader) |
| FsLister.dll | MassSock, EchoSock, BroadSock, AppManager |
| FsServer.dll | Adapter, MassSock, UserSock, EchoSock, BroadSock, AppManager |
| FsWatch.dll | MassSock, UserSock, EchoSock, AppManager |
| FsCharge/FsDumper | MassSock/EchoSock/AppManager 等 |
| **小写 solo 系**（FsSoloEntry/World/…） | **仅 LogicLoader / LogPool** |

结论：FsComm 主要服务**大写 Fs 分布式体系**；solo 系网络走各自内置的
CReactor（见 internal/reactor）。对应本项目「分布式拆分」里程碑
（proc TCP 总线 + 运维件），solo 阶段只有 LogicLoader/LogPool 落地价值。

## 二、组件清单（自有对象 = 1757 符号）

| 官方类 | obj | 职责 | Go 复刻 |
|---|---|---|---|
| Adapter | Adapter2.obj (175) | IOCP 内网 TCP 侦听（**内置 u32 分帧**） | `fscomm.Adapter`（帧模式） |
| MassSock | MassSock.obj (68) | WSAEventSelect 海量接入，serial 身份，读侧**裸字节** | `fscomm.Adapter{Raw:true}` |
| MultiSock | MultiSock.obj (59) | IOCP 客户端，AutoRetry/RetryDelay，四回调 | `fscomm.MultiSock` |
| UserSock | UserSock.obj (52) | 单客户端，Async 同步/异步双路 | `fscomm.UserSock` |
| EchoSock | EchoSock.obj (52) | 维护口 TCP 控制台（select 循环） | `fscomm.EchoSock` |
| EchoSockHook | EchoSockHook.obj (46) | 行分帧 + `on_input_%s` 分发 | 并入 EchoSock |
| BroadSock | BroadSock.obj (42) | UDP 广播口（收发同 socket） | `fscomm.BroadSock` |
| AppManager | AppManager.obj (64) | 跨进程应用注册表 + 单实例锁 + 3s 清扫 | `fscomm.AppManager` |
| CAppBook | AppBook.obj (20) | 共享内存注册表（32 槽） | 并入 AppManager |
| LogPool | LogPool.obj (34) | 按日日志池 | `fscomm.LogPool` |
| LogicLoader | LogicLoader.obj (35) | FxGameLogic 模块装载 | `fscomm.LogicLoader` |
| CSockBuf/SockFIFO | SockBuf/SockFIFO.obj | 有界消息队列 + 收缩 | `fscomm.msgFIFO` |
| WinSocket | WinSocket.obj (35) | WSA 启动壳（IntCreator） | Go 运行时自带，不复刻 |

Lua 属性/方法面（每类一套 getter/setter/prop_struct/func_struct）与
官方错误文案（"(Adapter::Start)bind failed" 等）在 Go 侧以注释锚定。

## 三、线格式（反汇编定案）

### 3.1 内网帧 = [4B 小端长度 N][N 字节帧体]，连续打包

**读**（Adapter::ProcessRead @ 0x10056120）：

```
connect_t（数组基 this+0x70，stride 0xAC）：
  +0x20 socket(-1=无效)  +0x6c inbuf 填充指针  +0x70 inbuf 基址
  +0x74 头部凑字节计数   +0x78 头部 4 字节缓冲
循环：header<4 时逐字节从填充位拷入头部 → msg_len=*(u32*)head
断言：msg_len <= get_max_send_size(m_nOutBufLen)（0x100551a0：= OutBufLen-4）
凑满 msg_len 字节 → memcpy 到 inbuf 头 → hook(index, inbuf, msg_len)
  （虚表 this+0x2c→vtable+0x30）→ 半帧留缓冲等下一轮
```

**写**（Adapter::WriteOutBuf @ 0x10055a60）：

```
total=0
while FIFO.Peek(&p,&len) 且 len+4 <= OutBufLen-total：
    *(u32*)(outbuf+total) = len
    memcpy(outbuf+total+4, p, len)
    total += len+4; FIFO.Pop()
返回 total → ProcessWrite 按剩余量 WSASend
```

即 internal/net.PrefixFramer 的原型（本包复用之）。
MultiSock::WriteOutBuf（0x10075f50）同形；**读路径不同**——
MultiSock/MassSock 的 ProcessRead（0x10076330/0x100708f0）把**裸字节块**
直接投给宿主 hook（(index, buf, len, 剩余量)），成帧/编码留给宿主
（外网客户端线路的 EE+XOR 在 hook 内做）。Go 复刻以 Raw 模式覆盖：
两端同用 PrefixFramer，线上字节与官方一致。

### 3.2 EchoSock 行协议（0x10062790 循环）

- recv 缓冲逐字节扫描：`'\n'(0x0A)` 或 `'\r'(0x0D)` 任一即行终止
- 行缓冲上限 **0x3FF=1023** 字节，超限字节丢弃直至终止符
- 行长 0 的空行不分发
- `EchoSock::SendLine`（0x10062310→Send(info,0)）**原样发送不补终止符**
  （官方 Lua 侧自拼 `"\r"`）

**分发**（CEchoSockHook::OnReceive @ 0x10064140）：

```
切词（空格/制表，token 缓冲 0x400）
首词存在且为字符串 → sprintf("on_input_%s", 首词) → 有该 Lua 回调即独占调用
否则 → "on_input"(实体, 整行, 0) 兜底
```

### 3.3 MassSock serial 身份

MassSock 连接表项带 serial（GenerateSerial @ 0x1006e190），
ClientSend/ClientClose/ResetInuse 均校验 "serial not match"——
槽位复用后陈旧索引的写/关请求被拒。Go 侧以「连接对象从 map 摘除
即失效」表达同一防悬挂语义（idx 不复用）。

## 四、AppManager/CAppBook 跨进程注册表

- **命名**（.data 0x100e9b24/0x100e9b28 前缀指针实证）：
  单实例互斥量 = `"mutex_" + AppName`；注册表共享内存 = `"memory_" + AppName`
- **单实例**：AppManager::Create（0x1005ae70）首步
  `CMutex::Exclusive` 失败即整体失败（"app mutex %s exclusive failed"）
- **布局**：总长 `0x6208` = 8 字节头{count,…} + **32 槽 × 0x310**
  book_data_t{used:bool, name[0x80], path[0x100], addr[0x100],
  info[0x80]（TFastStr 头 8B+数据）, 尾部 state/procid}
- **注册内容**：AppName / exe 全路径 / AppAddress / state(WAPS_*) / pid
- **清扫**（AppManager::DoWait @ 0x1005cb20）：每 **3000ms**（cmp 0xBB8）
  持 AppBookMutex 扫全表，对每槽重建 `mutex_<name>` 探活——
  互斥量消失（进程死亡）即摘除该槽
- Lua 面：AppName/AppAddress/NeedCheckApp 属性 + GetAppCount/
  GetAppList/SetAppState/SetAppInfo/Start/Stop

**Go 等价口径**：注册表=目录下 JSON（原子重写）；单实例=O_EXCL 锁文件
（pid+进程内持锁登记补同进程互斥）；探活=进程存在性
（Windows OpenProcess+STILL_ACTIVE / POSIX kill 0）——
进程死亡即失效，与 named-mutex 存在性判定等价。不做二进制兼容
（Go 服只与 Go 服互通）。

## 五、其余组件语义

- **MultiSock**：非阻塞 connect+select 探活（CheckConnect），断线后
  AutoRetry 按 RetryDelay 退避重连；回调 ClientBegin/ClientConnected/
  ClientConnectFail/ClientClose
- **UserSock**：状态机 this+0x38（0 未连/1 连中/2 已连/3 关闭）；
  Async=false 同步直发，true 走 FIFO（满="send buffer overflow" 且转关闭态）；
  "message too long"=超 OutBufLen
- **BroadSock**：UDP 口 bind + SO_BROADCAST/SO_REUSEADDR/非阻塞 +
  独立线程 select 收包；Send(addr,port,data)/SendBroad；载荷语义宿主定义
  （九阴侧=lister 线路注册 payload）
- **LogPool**：行格式 `"[%04d-%02d-%02d %02d:%02d:%02d] %s"`；按日文件名
  `"%s%04d%02d%02d%02d%02d%02d.log"`（前缀+年月日时分秒=当日首写时刻）；
  FileName 固定文件 / FilePrefix+EveryDayAlone 按日滚动二选一；TraceMax
- **LogicLoader::Load**（0x100690f0）：LoadLibrary → 必须导出
  FxModule_GetType → 类型串必须 `"FxGameLogic"`，否则
  "dll type error" 拒收。game_unite.ini [Logic] 即其装配面
  （Module0=fsgame.dll / Module1=SnailEditor.dll）

## 六、Go 复刻对齐说明

| 官方机制 | Go 落地 | 偏差说明 |
|---|---|---|
| IOCP + WorkThreads | netpoller（Windows 下即 IOCP）+ goroutine | 架构语义对齐，非 syscall 逐字移植（同 reactor 包决策） |
| 共享内存 AppBook | JSON 文件 + 锁文件 | 不做二进制兼容；语义（单实例/32 槽/3s 清扫/pid 探活）对齐 |
| named mutex 探活 | 进程存在性探活 | 判定结果等价 |
| LogicLoader LoadLibrary | 编译期工厂注册表 | Go 无 dlopen 需求；类型门 "FxGameLogic" 保留 |
| ShrinkDataNum/BufLen | 省略 | GC 天然回收 |
| WinSocket（WSA 壳） | 省略 | 运行时自带 |

验证：`go test ./internal/fscomm`（11 用例：FIFO 双上限/帧回环+半帧粘包/
广播+单连关闭/Raw 模式/断线重连生命周期/同步直发/行协议四态+具名命令
独占+超长截断/UDP 回环/注册+单实例+异名共享+死槽清扫/按日命名+固定文件/
类型门+装载），全绿；全仓提交树 build/vet/test 无回归
（login TestFlowEntrySceneOverGateway 为既有已知 flake，见任务与进度 §六）。
