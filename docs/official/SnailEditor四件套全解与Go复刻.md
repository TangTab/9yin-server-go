# SnailEditor 四件套全解与 Go 复刻

> 权威文档：`SharingMemory.dll` / `SnailEditorNet.dll` / `SnailEditor.dll` / `SnailEditorModule.dll`
> 快照：`F:\Game\Online\9yin\BGserverCJ\Debug`（2014-03-22，VS2005 编译，含 .pdb/.map）
> 源码路径（自断言串）：`e:\work\SnailEditor\Client\Exec\bin\`（编辑器客户端 GUI + Net 库）、
> `e:\work\snaileditor\tools\snaileditornet\snaileditornet\`（Net 库工程）、
> `d:\bg_source\server\snaileditor\`、`d:\bg_source\server\fsgame\utili*`（服务端模块）
> 2026-11-19 逆向定案；Go 复刻见 `internal/snailsharemem` / `internal/snaileditornet` / `internal/snaileditor` / `cmd/snaileditor`

---

## 1. 体系定位：服务端实时场景编辑器管线

四件套构成蜗牛"编辑器 ↔ 运行中游戏服"的实时协同管线（GM/策划在编辑器 GUI 里改场景、
脚本、配置，运行中的服务器热重载；文件走 SVN（TortoiseSVN）同步）：

```
┌─────────────────────┐   TCP (FMSG 帧, 8B头)   ┌───────────────────────────────┐
│ SnailEditor GUI     │◄───────────────────────►│ 游戏服宿主进程 (FsMember 等)    │
│ (编辑器客户端,       │  ServerSession/Listen    │  ├─ SnailEditor(.Module).dll  │
│  链 SnailEditorNet) │  ClientInit/ClientConnect│  │    (FxGameLogic 逻辑模块)   │
└─────────────────────┘                         │  └─ SnailEditorNet.dll (动态)  │
                                                │       └─ SharingMemory.dll(动态)│
                                                └───────────────────────────────┘
                                                     进程内: MsgManager 队列
                                                     跨线程: 共享内存 SNAILEDITOR_*
```

- **SharingMemory.dll**：COM 风格共享内存 IPC 库（单帧信箱，读端创建/写端打开）。
- **SnailEditorNet.dll**：编辑器网络库（IOCP TCP + 文件传输 + SVN 调用 + 共享内存轮询），
  双向双角色：编辑器 GUI 侧用 Server 端，游戏服侧用 Client 端；**被 SnailEditor.dll 在运行时
  `LoadLibraryA("SnailEditorNet.dll")` 动态加载**（非静态导入）。
- **SnailEditor.dll / SnailEditorModule.dll**：FxCore 逻辑模块（`FxModule_Init` /
  `FxGameLogic_GetCreator` 五导出面，与 fsgame.dll 同接口），部署上二选一；
  Module 版是超集（多 varprop/extend/custom/geometry 命令面 + CXmlFile）。

### 导出面

| DLL | 导出 | 说明 |
|---|---|---|
| SharingMemory.dll | `GetFactory` | 返回 `SHARINGMEMORY::fac`（CFactory 单例，0x100b3a00） |
| SnailEditorNet.dll | `GetSnailEditorNet` | 返回 `&g_SnailEditorNet`（0x10110f20，ISnailEditorNet*） |
| SnailEditor.dll | `FxModule_GetType/FxModule_Init/FxGameLogic_GetVersion/GetCreator/GetModuleCreator` | FxCore 逻辑模块面（同 fsgame.dll） |
| SnailEditorModule.dll | 同上五导出 | 同上 |

加载方式：`game_member.ini` 的 `[Logic] Module0=fsgame.dll` 位替换为
`snaileditor.dll`/`snaileditormodule.dll`（或追加 Module1=）由 FsMember 宿主加载。

---

## 2. SharingMemory.dll 全解

### 2.1 类体系（PUB 符号 + RTTI vtable 定案）

```
IMyUnknown           纯接口: [0]=virtual dtor, [1]=AddRef, [2]=Release   ← 无 QueryInterface!
 └ CUnknown          引用计数实现 (InterlockedIncrement/Decrement)
IFactory / CFactory  fac 单例; CreateInstance(名字) → IMyUnknown*
ISharingMemoryRead   读端 7 虚方法（声明序 = 下表 +0x0C..）
 └ CSharingDataExRead    持内部 CSharingData*（this+0x21C）+ 自身 readerID（this+0x220）
ISharingMemoryWrite  写端 6 虚方法
 └ CSharingDataExWrite   持内部 CSharingData*（this+0x21C）
CSharingMemory<T>    模板: 名字缓冲(this+4, wchar 0x104) + hMap(this+0x20C) + pView(this+0x210)
CSharingData         共享内存头对象本身（直接位于映射基址）
```

调用约定：CFactory::CreateInstance(L"SharingMemoryRead") / (L"SharingMemoryWrite")；
`CLibraryLoader`（SnailEditorNet 内）封装 `LoadLibraryW(L"SharingMemory.dll")` +
`GetProcAddress("GetFactory")` + CreateInstance 链。

### 2.2 共享内存布局（0x814 字节，定案）

映射名：`swprintf(name_buf, L"SNAILEDITOR_%s", 用户名)`（0x1009ed64 常量）。
实例名："SvnUpdateFinish"（编辑器管线唯一用例）。

```
CSharingData @ 映射基址：
+0x000  vtable（仅写端进程有效，读端不触）
+0x004  data[0x800]        单帧消息缓冲（= GetBufferMaxSize()）
+0x804  uint32 dataLen     当前帧字节数（= GetBufferSize()）
+0x808  byte   valid       1=有新帧（= GetReadable()）；GetWritable() == !valid
+0x80C  uint32 readableBitmap  可读位图（每读者 1 位；写帧时 = registeredBitmap）
+0x810  uint32 registeredBitmap 注册位图（读者 ID 0..0x1F；GetMaxCountReader()=0x20）
合计 0x814 = CreateFileMappingW 的 dwMaximumSizeLow
```

### 2.3 打开路径（两端不对称，定案）

- **读端创建**（CSharingDataExRead::Open → CSharingMemory<T>::OpenRead 0x1003d4d0）：
  `CreateFileMappingW(INVALID_HANDLE_VALUE, NULL, PAGE_READWRITE, 0, 0x814, L"SNAILEDITOR_xxx")`
  → 若 `GetLastError()==ERROR_ALREADY_EXISTS(0xB7)` 直接 `MapViewOfFile(FILE_MAP_ALL_ACCESS)`
  （映射已存在，头部由先创建者初始化）；
  否则（自己是首创建者）MapView 后**栈上构造零初始化 CSharingData 实例 memcpy 进视图**
  （ctor: memset(+4,0,0x800) + dataLen=0 + valid=0 + readable=registered=0），
  随后 `CreateReaderID(&this->m_readerID)`。
- **写端打开**（CSharingDataExWrite::Open → 0x1003dea0）：
  `OpenFileMappingW(FILE_MAP_ALL_ACCESS, FALSE, L"SNAILEDITOR_xxx")` + MapViewOfFile。
  映射不存在则失败（读端必须先起）。

文件掩码坑：两 API 均传 `0xF001F`（= FILE_MAP_ALL_ACCESS / SECTION_ALL_ACCESS）。

### 2.4 单帧信箱协议（定案）

```
写端:                                     读端(每读者独立位):
  while (!GetWritable()) 等待               if (GetReadable()):
  ASSERT(registeredBitmap != 0)               p = Read(readerID)
  len = min(len, 0x800)                        // 前置: valid && 0<=id<32
  dataLen = len                                //        && registeredBitmap[id]
  memcpy(data+4, buf, len)                     //        && readableBitmap[id]
  readableBitmap = registeredBitmap            // → 返回 data+4 指针
  valid = 1                                    len = GetBufferSize()
  return len                                   消费 data[0..len)
                                             ReadEnd(readerID)
                                               // 同前置; readableBitmap &= ~(1<<id)
                                               if (readableBitmap == 0) valid = 0
```

- `CreateReaderID(&id)`：扫 0..0x1F 找 `registeredBitmap` 首个 0 位，置位并返回真；
  满 32 则 `*id=0xFF` 返回假。`FreeReaderID(id)`：清 registeredBitmap 位。
- `CheckIllegalUserID(id)`：`id < 0 || id >= 0x20`。
- 写端 ASSERT 串：`m_bitUserValid != 0`（sharingdata.cpp）——写帧时若无任何读者注册则弹断言。
- 无锁：仅靠 valid/位图字段的原子可见性（单写多读 SPSC 变体，读者间用位图隔离）。

### 2.5 CFactory / GetFactory

`GetFactory()` 返回 `&fac`；`fac.CreateInstance(L"SharingMemoryRead"|"SharingMemoryWrite")`
new 对象（ExRead 0x224 字节含 CSharingMemory 模板成员 @ +8）压入内部 vector 统一 Release。
IMyUnknown 无 QI——工厂直接返回具体接口指针。

---

## 3. SnailEditorNet.dll 全解

### 3.1 结构（符号修正 +0x4E000 = 真 RVA，双点验证）

```
GetSnailEditorNet (导出 0x817D0) → &g_SnailEditorNet (0x10110F20)
g_SnailEditorNet: SnailEditorNet 单例 —— 全部方法转发 g_Con: FileCon (0x10110F28)
g_Con / FileCon:  网络枢纽。ServerInit/ServerListen（IOCP 收连接）、
                  ClientInit/ClientConnect（主动外连编辑器 GUI）、文件收发、SVN 配置
CIOCP:            IOCP TCP 服务器（Listen/AcceptThread/WorkThread/OnAccept/OnRead/OnClose）
ServerSession:    编辑器 GUI 侧多连接管理（PlayerItem 列表; OnAccept/OnRead/OnClose/
                  SendAll/SendAllExceptOne/IsSendFile/ServerSendSvnUpdate）
ClientSession:    游戏服侧外连会话。四线程: IOCP 收/发 + ProcessMsgThread +
                  ReadSharingMemoryThread; 持 MsgManager 复本(0x14C)与双回调(0xD0/0xD8)
FCon:             阻塞式 TCP 客户端（WorkThread; Init/Connect/Send/Recv）——文件通道
_fileworkitem:    文件传输工作项
CLibraryLoader:   LoadLibraryW("SharingMemory.dll")→GetFactory→CreateInstance("SharingMemoryRead")
CSetup:           HKLM 注册表 SetUp（WriteRegValue/ReadRegValue/AssemblyRegValue）
MsgManager/sMsg:  {int type; int x; char* data} deque + CLockUtil(CRITICAL_SECTION)
```

### 3.2 ISnailEditorNet vtable（12 槽，定案）

```
+0x00 ServerInit()                 +0x1C SetStrSvnTortoiseProc(wstring)
+0x04 ServerListen()               +0x20 GetStrSvnTortoiseProc()
+0x08 ClientInit()                 +0x24 SetStrSvnLocalRootPath(wstring)
+0x0C ClientConnect(ip, port)      +0x28 GetStrSvnLocalRootPath()
+0x10 SetDelaySvnUpdate(cb)        +0x2C SetSvnWatchPathes(vector<wstring>&)
+0x14 SetSvnUpdateCompleteCallback(cb)
+0x18 SetSvnConflictChangedCallback(cb)
```

### 3.3 FMSG 线协议（定案）

帧 = `{ u32 totalLen(含头); u32 msgId; char payload[totalLen-8] }`，TCP 直连裸流。
msgId 基址 **0x3E8 (1000)**，ClientSession::OnRead 按 `msgId-0x3E8` switch 0..9：

| id | 名称/语义（自处理体定案） | 处理 |
|---|---|---|
| 0x3E8 | 文件块 ACK/0x200 缓冲回发 | alloc 0x200 → 回读转发 |
| 0x3E9 | 文件传输握手（this+0x34 通道） | 0x1000 缓冲 + 0x1005049F |
| 0x3EA | 发送文件数据块 | Lock + WSASend 族 + +0xC8 状态 |
| 0x3EB | 文件占用冲突 | `printf("[error]%s is used by other peaple!")` |
| 0x3EC | 编辑器命令帧（进游戏队列） | alloc 0x800 → 读载荷 → 0x10050805 入队 |
| 0x3ED | 发送文件数据块(变体) | 同 0x3EA |
| 0x3EE | — | （default 出口） |
| 0x3EF | 发送文件数据块(变体) | 同 0x3EA |
| 0x3F0 | 空分支 | default 出口 |
| 0x3F1 | **FMSG_SERVER_SVN_UPDATE** | `recv FMSG_SERVER_SVN_UPDATE: %s`；'/'→'\\'；
  路径列表 → 延迟 SVN 更新（见 3.5） |

服务端对等：`ServerSession::OnRead` 头 8 字节取 msgId（!=0 校验）→ 0x800 应答缓冲回填
0x3F1 回发；`ServerSendSvnUpdate(payload,len)`：`buf[0]=8+len; buf[4]=0x3F1;
memcpy(buf+8,payload,len); Send(buf, 8+len)`。文件块 0x200/0x1000 分块 + "Start/End Send
File:%s,times:%d!" 计时。

### 3.4 ClientSession 四线程数据流（定案）

```
IOCP WorkThread ──OnRead(FMSG)──┐
                                ├─► [0x14C] MsgManager 队列
ReadSharingMemoryThread ◄───────┘        │
  Load SharingMemory.dll                 ▼
  CreateInstance("SharingMemoryRead")   ProcessMsgThread(100ms 轮询):
  Open(L"SvnUpdateFinish")                sMsg{type,x,data}:
  loop while running(0x124):                type=0 → cb[0xD0](x,data) = FuncDelaySvnUpdateCallback
    if GetReadable():                       type=1 → cb[0xD8](x,data) = SnailEditorServerModule::PushMsg
      p=Read(); len=GetBufferSize()           （两回调均由 SnailEditor.dll 在装配时注入）
      推字符串 → [0x14C] 队列
      ReadEnd()
    Sleep(100)
```

共享内存帧载荷解析：`len-7` 为字符串体（前 4B int + 3B 保留），构造 string 入队。

### 3.5 SVN 更新（TortoiseSVN 集成，定案）

- TortoiseProc 路径：`HKLM\SOFTWARE\TortoiseSVN\ProcPath`（读注册表，StartSnailEditor 做）。
- 命令行：`"<TortoiseProc.exe>" /command:update /path:"<本地根路径>" /notempfile /closeonend:1`
  （字符串 `" /command:update /path:"`、`" /notempfile /closeonend:1"`）。
- 日志串：`start update: %s`、`delay update: %s`、`End Send File:%s,times:%d!`。
- 流程：编辑器 GUI 提交（0x3F1 路径列表）→ 游戏侧 `FuncDelaySvnUpdateCallback` 把路径并入
  `m_vNeedSvnUpdatePathsLowerCase` → 帧边界统一 `SvnUpdate()`（WinExec TortoiseProc +
  等待）→ 完成回调 `SvnUpdateFinish`（写共享内存名即此事件名）→ Net 读线程 → 广播回 GUI。
- `IsInWatchPath`/`AddSvnWatchPathes`：仅 watch 目录下的路径触发更新。

---

## 4. SnailEditor.dll / SnailEditorModule.dll 全解

### 4.1 模块装配（FxCore 宿主面）

`FxModule_GetType` → 逻辑模块类型；`FxGameLogic_GetModuleCreator` → `EditorUpdateModuleCreator`
（`Name()`="EditorUpdate"）→ `Create()` → EditorUpdate(ILogicModule)。
`ILogicModule::BeforeLaunch/LoadResource` 桩。

### 4.2 SnailEditorServerModule::Init(IFxKernel*)（0x100ADE00，定案）

```c
m_pSnailEditorServerModule = this;        // 0x10167220 单例
m_pKernel = p;                            // 0x10167224
p->RegisterEvent("scene", "OnCreate", OnSceneCreate, 0);          // vt+0x18
p->RegisterTimer("SnailEditorServerModule::HB_ProcessMsg",        // vt+0x34
                 HB_ProcessMsg, 0);
p->RegisterEvent("player", 0x13, OnPlayerEvent, 0);               // vt+0x28
```

### 4.3 StartSnailEditor（0x100AE2E0，定案）

```c
if (started) return 1;
// 1. TortoiseProc 路径: HKLM\SOFTWARE\TortoiseSVN\ProcPath → this+0xC（失败 Trace 警告）
// 2. TinyXML 读 ini\SnailEditorServerModule.xml:
//    <root><config SvnLocalRootPath=... SvnServerIP=... SvnServerPort=.../>
//          <watch>…路径列表…</watch></root>
// 3. h = LoadLibraryA("SnailEditorNet.dll");  失败 Trace "load SnailEditorNet.dll failed."
// 4. pfn = GetProcAddress(h, "GetSnailEditorNet"); net = pfn();
// 5. net->SetStrSvnLocalRootPath(路径)          (vt+0x24)
//    net->SetStrSvnTortoiseProc(TortoiseProc)   (vt+0x1C)
//    net->ClientInit()                          (vt+0x08) 失败 Trace "gConnect error"
//    net->ClientConnect(SvnServerIP, SvnServerPort) (vt+0x0C) 失败 Trace "init error"
//    net->SetDelaySvnUpdate(FuncDelaySvnUpdateCallback)      (vt+0x10)
//    net->SetSvnUpdateCompleteCallback(SnailEditorServerModule::PushMsg) (vt+0x14)
//    net->SetSvnWatchPathes(watch 列表)          (vt+0x2C 循环)
// Trace "StartSnailEditor complete."
```

### 4.4 消息闭环（游戏侧，定案）

```c
// Net 回调（cdecl，跨 DLL）:
static PushMsg(len, msg):                  // SnailEditorNet type=1 命令
    wstring wide(msg); string s = UnicodeToANSI(wide);   // 双重编码净化
    if (s.size() >= 0x800) s = 默认串;      // 0x800 上限 = 共享内存帧宽
    strncpy(g_buf /*0x10166A20,0x800*/, s);
    m_pKernel->Trace("(SnailEditorServerModule::PushMsg) buf:%s");
    m_msgManager.PushMsg(0, 0, g_buf);      // 0x10167228

static FuncDelaySvnUpdateCallback(x, path): // Net type=0：并入延迟 SVN 更新
    UpdateNeedSvnUpdatePaths(path) → m_vNeedSvnUpdatePathsLowerCase

// 心跳（每帧）:
bool HB_ProcessMsg(dt):
    sMsg* m = m_msgManager.PopMsg();
    if (!m) return 1;
    string data(m->data);
    vector<string> files = split(data, "*");      // '*' 分隔的路径列表
    foreach f: if (IsFileInReloadPath(f)) files.push(f);   // ClipStringByTag/大小写折叠
    delete m;
    EditorUpdate::Reload(m_pKernel, files);
```

### 4.5 EditorUpdate（命令面，SnailEditor ⊂ SnailEditorModule）

Init 建立 `m_mapUpdate: map<路径, 重载动作>`；`Reload(kernel, files)` 按表分发。
Lua/内核自定义命令入口 `CustomEditor(varlist, …)`（ILogicModule 虚面）分派到：

```
OnCustomEditorStart/End/InfoUpdate        编辑会话生命周期
OnCustomEditorWriteObject{,Base,Ai,Task,Event,CustomEvent}
    → WriteObjectToXml 族（场景对象落 XML; 属性名 Grouping/CustomEvent/Event/Task/
      StaticID/Stage; creator 目录 ini\Scene\creator\npc_creator\*.xml）
OnCustomEditorIniWrite/Del/Add/End        ini 增量写（EditorUpdate 版走 CIniFile）
OnCustomEditorReloadRes                   资源重载（gm.lua / randomcreator / staticcreator /
                                          item *.xml; buff 族 ini\Skill\buff_*.ini +
                                          BufferModule; clear_allnpcEx）
NewXml/SaveXml/Reload
```

SnailEditorModule 额外命令面（对 FxKernel/游戏对象操作，函数名即 API）：

- **varprop_func**：`varprop_load_data / varprop_refresh_data / varprop_refresh_property /
  varprop_query_property×2`（hash_map 属性表查询/刷新）
- **extend_func**：`createclass_from_xml / createclass_add_property`（逻辑类动态创建）、
  `find_firsttype_parent / get_sceneobj_master / run_logic_lua_script / get_random_position /
  save_to_string / save_to_xml / compare_property / extend_warning / convert_*`（类型转换族）
- **custom_func**：CVarList 完整实现（FxKit IVarList 克隆）+ 游戏内效果命令
  `CustomSysInfo{,ByName,ByKen,ByScene,ByGroup,ByChannel,Broadcast}`、
  `CustomBeginMenu/AddTitle/AddMenu/EndMenu/CloseMenu`、`CustomSpeech`、
  `CustomAction{,ByKen}`、`CustomStateAction`、`CustomEffect{,Destory}ByKen`、
  `CustomSetNeedCollide`
- **geometry_func**：`util_is_point_in_sphere / util_is_point_in_cylinder /
  util_random_point_in_circle`
- **XmlFile**：CXmlFile（section/item 视图的 XML 读写器，LoadFromFile/_Rapid/SaveToFile）

---

## 5. Go 复刻

| 包 | 对应 | 要点 |
|---|---|---|
| `internal/snailsharemem` | SharingMemory.dll | 零布局依赖的 `Mailbox`（0x814 帧 + 双位图协议）+
  Windows `CreateFileMapping` 映射层（`SNAILEDITOR_` 前缀、读创建/写打开、0xF001F）；
  名字/掩码/大小与原版逐字节兼容，ctypes 对拍原 DLL 通过 |
| `internal/snaileditornet` | SnailEditorNet.dll | FMSG 帧编解码（8B 头 + 0x3E8 基址）、
  ClientSession 等价（TCP 拨号 + 共享内存轮询读 + 命令队列 + 双回调）、
  ServerSession 等价（多连接 + SendAll + 0x3F1 广播）、文件分块传输、
  TortoiseProc SVN 命令行构造 |
| `internal/snaileditor` | SnailEditor(.Module).dll | 模块装配面（kernel 接口注入）、
  StartSnailEditor 等价（注册表 TortoiseProc 路径 + SnailEditorServerModule.xml 配置）、
  MsgManager（sMsg deque）、HB_ProcessMsg（`*` 分隔路径 → Reload 回调）、
  EditorUpdate 命令分发表（可注册式） |
| `cmd/snaileditor` | 演示宿主 | 单进程跑通：配置读取 → TCP 客户端/服务端 → 共享内存桥 →
  命令回调打印/SVN 更新 |

测试：
- `snailsharemem`：协议单测（多读者/满帧/非法 ID）+ **Win 下 ctypes 加载原版
  SharingMemory.dll 对拍**（Go 写 → 原生读；原生写 → Go 读，见 `sharingmem_dll_test.go`，
  需 `F:\Game\Online\9yin\BGserverCJ\Debug` 存在，无则跳过）。
- `snaileditornet`：帧编解码 + 回环 TCP（client/server 互通 + 0x3F1 广播）。
- `snaileditor`：MsgManager/HB 解析/装配配置。

## 6. 未尽事项

- FMSG 0x3E8..0x3F0 各分支的精确命名需编辑器客户端（`SnailEditor.Client`）二进制交叉验证，
  现按处理体语义命名。
- SnailEditorModule 的 varprop/extend/custom 命令面仅做 API 面清单与接口抽象（其实现深耦合
  FxKernel 游戏对象，属里程碑二玩法层）。
- IOCP 每连接缓冲细节（0x200/0x1000 分块窗口）按观察值实现，未逐指令验证序列号语义。
