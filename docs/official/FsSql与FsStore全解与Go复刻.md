# FsSql.dll / FsStore.dll 全解与 Go 复刻

> 逆向基准：`BGserverCJ/Debug/FsSql.dll`（3,268,608 B + .map 4,791 函数符号全量）
> 与 `FsStore.dll`（1,073,152 B + .pdb，无 .map）+ `GameStore.bat/GameStore.ini`
> + `Lua/GameStore/main.lua / echo.lua`（官方装配蓝图）。
> 源码路径（断言串泄露）：`d:\flexis\client\fsstore\gamestore.cpp`、
> `gamestorestate.cpp`、`access.cpp`、`d:\flexis\client\network\interconnect.h`
> （MAX_MEMBER_NUM=16）、`d:\flexis\client\public\inlines.h`；
> FsSql 侧 `c:\wuyf\svn\serveriii\branches\publish20130508\server\`。
> 工具：`tmp_research/fssqlstore/fss_dis.py`（map/pdb 符号 + capstone 注释反汇编 CLI）。

---

## 〇、总览与分工

| 模块 | 定位 |
|---|---|
| **FsStore.dll** | 网络集群**存储/DB 服务器**（GameStore 实体，MainEntity=MainConsole 由 lua 装配）。FsSoloStore 的网络版同源体：五通道 DB 池 + 41 型消息协议 + 0xEE 转义内网帧 |
| **FsSql.dll** | **DB 抽象层模块**（注册 `MysqlDBSystem`/`SqlDBSystem` 两实体）：libmysqlclient（静态含 yaSSL/TaoCrypt）+ ADO 双后端，实现 IDBSystem/IDBConnection/IDBCommand/IDBRecordset 四层接口 |

两 DLL 均为标准 FxModule 插件（8 导出面），版本 0x3EB=**1003** 族。
装配：`GameStore.bat → FsLoader.exe GameStore.ini → MainEntity=MainConsole`，
模块加载序 `FsKit → FsSql → FsComm → FsStore`，脚本 `..\Lua\GameStore`。

**关键结论（与直觉相反）**：网络版 GameStore 的属性面**没有 DBSystem 槽**
（g_GameStore*_prop_struct 全表无此项），CAccess 直接内嵌 **CAdoConn**（ADO）
连库；lua 里 `game_store.DBSystem = db_system` 实为无效赋值（遗留兼容）。
FsSql 的 DBSystem 实体在 GameStore 进程里只被创建、不被消费——它是
solo 系（FsSoloStore 静态链接同源 IDBSystem 消费方）与未来进程的公共件。

### 与 solo 版（FsSoloStore.dll，见姊妹篇）差异速览

| 维度 | FsSoloStore（solo） | FsStore（网络版） |
|---|---|---|
| 主实体 | SoloStore（MainEntity 直启） | MainConsole + lua 装配 GameStore |
| 内网 | CInnerHook（自建 listener/连接表） | **Adapter 实体**（FsComm）+ CAdapterHook 钩子 |
| 广播 | CReactor::CreateBroadcast | **BroadSock 实体** + CBroadSockHook |
| 监控 | CAppRecord | **AppManager 实体**（WAPS_ 8 态） |
| 维护口 | CEchoHook（内建） | **EchoSock 实体**（echo.lua 绑定） |
| 上行消息 | 0x00..0x42（67 型） | **0x00..0x28（41 型）**（结构尺寸代际不同） |
| Account 族 | 仅连接池 | **5 型消息**（reset/logon/logoff/validate/add → dc_accounts） |
| 回执路由 | wake socket 唤醒主线程 | **DB 线程直接 SendMsg**（Adapter 线程安全） |
| 回执抑制 | — | **StoreId>0 不回发**（分组主从，主 store 独占回包） |
| 分片 | serial%(pools-1)+1 | **hash(name) % pools**（BKDR-5 三变体） |
| DB 访问 | IDBSystem 接口 | **CAdoConn 直接 ADO**（SQL Server 方言） |
| 唤醒 | Port_SocketSend(wake) | DoWait 轮询 + Sleep(1) |

---

## 一、FsStore.dll：GameStore 实体

### 1.1 对象布局（pdb 符号 + 反汇编实证）

```
GameStore@this（IEntity 派生）
+0x24  state(i)          四态 0=Unknown/1=Opened/2=Closing/3=Closed
+0x28  DistrictId        +0x2C ServerId          +0x30 StoreId
+0x34  AccountPools      +0x38 RolePools         +0x3C LogPools
+0x40  PublicPools       +0x44 ManagePools       （int，>0 才建通道）
+0x48  AccountBase       +0x68 RoleBase   +0x88 LogBase           （TFastStr@0x20）
+0xA8  PublicBase        +0xC8 ManageBase
+0xE8  AccountAccess[1]  +0x128 RoleAccess +0x168 LogAccess       （CAccess* 数组）
+0x1A8 PublicAccess      +0x1E8 ManageAccess                      （步长 0x40）
+0x230  BroadSock*(实体)  +0x238 Adapter*(实体)
+0x23C  CThread*(StoreThreadProc)      +0x240 CMsgDeque2(主线程响应队列)
+0x244  CTickTimer(1000ms)             +0x248 CInterConnect
（另存 Console/EchoSock/AppManager/Adapter 的 PERSISTID）
```

lua 可写属性 26 组：DistrictId/ServerId/StoreId/五通道 Base+Pools/Console/
EchoSock/BroadSock/AppManager/Adapter/State(只读)；
方法：Start/Stop/AccessEmpty/AddLogTable/AddLogField/OutputXml。

### 1.2 装配序（main.lua + Start()，实证）

```
lua main(console):
  LogPool(log\store*.log, TraceMax=256) → IniDocument(GameStore.ini)
  EchoSock(15100+StoreID, echo.lua) → BroadSock(bind=inner, 广播=255.255:15000)
  AppManager("store_<server>_<store>") → Adapter(inner:16100+StoreID,
      ConnectMax=64, WorkThreads=1, In/OutBuf=1MB)
  db_system = nx_create(UseMysql ? "MysqlDBSystem" : "SqlDBSystem")   ← 无效赋值
  game_store = nx_create("GameStore") → 属性全注入 → init_custom_log
      (AddLogTable/AddLogField 12 张运营日志表)
  Start 序: AppManager → Adapter → BroadSock → EchoSock → GameStore:Start()
  SetAppState(WAPS_OPENED)（AppManager 8 态 0..7，进程级）

GameStore::Start @0x100A4DF0:
  [this+0x23C] CThread::Start（StoreThreadProc 主循环线程）
  五通道: pools>0 → StartAccess(&slot, pools, base_cstr)
      断言 access!=NULL / num>0 / conn_str!=NULL（access.cpp）
      new CAccess → SetServerId → SetConnectString → CAccess::Start
      失败 trace "start failed" 后换下一池
  CTickTimer::Start(1000)
  StoreStateMachine(1, 0, 0, 0)   → Unknown→Opened
Stop @0x100A5130: CThread::Stop → 五通道 StopAccess（倒序同表）
```

### 1.3 状态机（gamestorestate.cpp，跳表 @0x100A956C）

| 态 | 事件 | 行为 |
|---|---|---|
| Unknown(0) | 1 → Opened(1)；0 → 无；其它 assert | — |
| Opened(1) | 2 → Closing(2)；0 → 无 | — |
| Closing(2) | 0 | `AccessEmpty()` 为真才迁移：构造 `{u32 1, u32 StoreId}` 8B → `CInterConnect::SendToWorld(0, 0x1C, data, 8)`（**通知 world 本 store 已排空**）→ Closed(3) |
| Closed(3) | 0 → 无；其它 assert | — |

主循环 `StoreThreadProc → DoWait @0x100A7F30`：
CMsgDeque2(+0x240) 非空则 SwapMsgVector+Peek 循环（ProcessMessage 为空壳，
消息经 PutMessage/PutMessage2 入队仅作优先级调度）；
CTickTimer 到 1000ms → `StoreStateMachine(0,0,0,0)` + `BroadcastStoreInfo()`。

### 1.4 网络面

#### 1.4.1 内网 Adapter 帧（0xEE 转义 + 12B 头）

CAdapterHook（实现 FsComm 的 IAdapterHook，NewInstance 8B / 每连接上下文 0x100008B）：

- **CreateContext**：分配 1MB+8B：`ctx[0..0x100000)`=收缓冲，`+0x100000`=写长，`+0x100004`=last_byte。
- **转义（MakeSendData/GetSendLength/GetMaxSendSize）**：载荷逐字节复制，遇 `0xEE` 补发 `0x00`（即 0xEE→`EE 00`），帧尾追加 `EE EE`；max_payload=(bufsize-2)/2，len 最坏 2n+2。
- **OnReceive 状态机**：`EE EE`=帧尾（修正已入缓冲的首个 EE，len-1 后投递）；`EE 00`=转义（EE 已入缓冲）；`00 EE` 后跟 00 → 字面 00；缓冲上限 1MB。
- **ReceiveMsg(data,len)**（帧体 ≥12B，否则 "message size error"）：
  - `head[2]==0xFFFF` → 绑定帧：`SetAdapterCommIndex(ic, head[0], ident)`（对端自报 comm）
  - `head[0]==0`（world）→ `SendAccessRequest(store, 0, head[2], data+12, len-12)`
  - `head[0]==1`（public）→ 同上 serial=1
  - `0x18 ≤ head[0] ≤ 0x27`（member 0..15）→ `serial=(head[0]<<16)+head[1]` 后同上
  - 其它 → "message source error"
- **OnAccept/OnConnected**：日志 `Store adapter %s:%d accept|connected, total %d`；
  OnConnected 后**主动发绑定帧** `{store_comm, 0, 0xFFFF}`（SendAdapterCommIdent，
  store_comm=StoreId+4）。
- **OnDisconnect**：GetAdapterCommIdent 日志 + SetAdapterDisconnect。

**头 12B = `{u32 comm, u32 serial, u32 msg}`**；出站同构（SendMsg 拼头后经
Adapter vtbl+0x88 两段发送；SendMsg2 为 7 参变体带附加路由参数）。

#### 1.4.2 CInterConnect（interconnect.h）

两字段：`+0x00 Adapter*`、`+0x04 CommIdent`。

| 方法 | 语义 |
|---|---|
| GetWorldCommIdent() = **0** | GetPublicCommIdent() = **1** |
| GetMemberCommIdent(n) = **0x18+n** | GetStoreCommIdent(n) = **n+4** |
| SendToWorld/SendToPublic/SendToMember(idx<0x10 assert) | 组头 SendMsg |
| SendAdapterCommIdent(ident) | 连入后自报身份帧 |

comm 空间与 solo 全系一致：world=0 / public=1 / auction=2 / entry=3 /
store=store_id+4 / member=0x18+n。**本进程入站只认 0、1、0x18..0x27。**

#### 1.4.3 广播自宣（BroadcastStoreInfo @0x100A7DB0，每秒）

```
0x28B UDP 帧: { 2, 0x300, InnerAddr[16], InnerPort, DistrictId, ServerId, StoreId }
              +0x00 kind=2(store)  +0x04 协议号 0x300（solo 为 0x3001，代际差异）
              +0x08 地址串(窄字符,Adapter vtbl+0x28 取宽字符后转换)
              +0x18 实际端口(Adapter vtbl+0x30)  +0x1C/+0x20/+0x24 三级编号
经 [this+0x230] BroadSock vtbl+0x48 发往 255.255.255.255:15000
```
CBroadSockHook::OnReceive 仅做过滤器：`size≥8 && head+4==0x300` 才认领。

#### 1.4.4 echo 维护口（EchoSock 实体 + echo.lua）

端口 15100+StoreID；命令 `quit`（触发 main.close 全链优雅停机）/
`end`（断开本连接）；欢迎语 `Welcome server %d store %d\r`。
echo_open/echo_close/echo_input 三回调（对比 solo 的 CEchoHook 内建）。

### 1.5 上行消息表（SendAccessRequest @0x100A6060，switch 0x00..0x28）

签名 `SendAccessRequest(this, serial, msg, data, size)`。分片键三函数
（public/inlines.h，BKDR-5）：
- `hashw(wchar*)`：`h = h*5 + c`（宽字符）
- `f7310(char*)`：`h = h*5 + fold[c]`（查表大小写折叠）
- `f73c0(char*)`：`h = h*5 + c`（裸）

| msg | 通道 | 结构（gamestore.cpp 断言） | 尺寸 | 分片键 |
|---|---|---|---|---|
| 0x00 | Role | role_info（无断言） | — | 0（固定槽） |
| 0x01 | Role | role_offline | — | 0 |
| 0x02 | Role | store_forw_load_t | ==0x8F | hashw(data+0x29) |
| 0x03 | Role | store_forw_load_cancel_t | ==0x6B | **PutLoadCancel 原地取消**（不入队） |
| 0x04 | Role | store_forw_save_t | ≥0x7A | hashw(data+8) |
| 0x05 | Role | store_forw_set_online_t | ≥0x6B | hashw(data+0x29) |
| 0x06 | Role | store_forw_set_offline_t | ≥0x6B | hashw(data+0x29) |
| 0x07 | Role | store_forw_illegal_t | ==0x42 | hashw(data+8) |
| 0x08 | Role | store_forw_block_t | ==0x62 | hashw(data+8) |
| 0x09 | Role | store_forw_resume_t | ==0xA2 | hashw(data+8) |
| 0x0A | Role | store_forw_create_t | ≥0x9B | hashw(data+0x2D) |
| 0x0B | Role | store_forw_delete_t | ==0x7F | hashw(data+0x3D) |
| 0x0C | Role | store_forw_roles_t | ==0x2D | f7310(data+0xC)（account 折叠） |
| 0x0D | Log | store_forw_log_data_t | ≥0x44A | hashw(data+8) |
| 0x0E | Log | store_forw_item_log_t | ≥0x106 | hashw(data+0x44) |
| 0x0F | Log | store_forw_chat_log_t | ≥0x488 | hashw(data+0) |
| 0x10 | Log | store_forw_gm_log_t | ≥0x44A | hashw(data+8) |
| 0x11 | Log | store_forw_custom_log_t | ≥0x24 | f73c0(data) |
| 0x12 | Role | （透传，无断言） | — | 0 |
| 0x13 | Role | store_forw_load_chunk_t | ≥0x4E | hashw(data+4) |
| 0x14 | Role | store_forw_save_chunk_t | ≥0x46 | hashw(data+4) |
| 0x15 | Role | store_forw_delete_chunk_t | ≥0x46 | hashw(data+4) |
| 0x16 | Role | store_forw_send_letter_t | ≥0x137 | hashw(data+0xC) |
| 0x17 | Role | store_forw_query_letter_t | ≥0x6B | hashw(data+8) |
| 0x18 | Role | store_forw_recv_letter_t | ≥0x8C | hashw(data+8) |
| 0x19 | Role | store_forw_look_letter_t | ≥0x6B | hashw(data+8) |
| 0x1A | Role | store_forw_clean_letter_t | ≥0xA0 | hashw(data+8) |
| 0x1B | Role | store_forw_reject_letter_t | ≥0x90 | hashw(data+8) |
| 0x1C | Role | store_forw_output_xml_t | ≥0x166 | 无分片 |
| 0x1D | Role | store_forw_input_xml_t | ≥0x166 | 无分片 |
| 0x1E | Public | store_forw_pub_info_t | ≥0x42 | 无分片 |
| 0x1F | Public | store_forw_pub_load_t | ≥0x8C | hashw(data+0x4A) |
| 0x20 | Public | store_forw_pub_save_t | ≥0x84 | hashw(data+0x42) |
| 0x21 | Public | store_forw_pub_add_t | ≥0xA5 | hashw(data+0x42) |
| 0x22 | Public | store_forw_pub_remove_t | ≥0x84 | hashw(data+0x42) |
| 0x23 | Account | account_reset（无断言） | — | 无分片 |
| 0x24 | Account | store_forw_account_logon_t | ==0x72 | f7310(data+0xC) |
| 0x25 | Account | store_forw_account_logoff_t | ==0x56 | f7310(data+0x14) |
| 0x26 | Account | store_forw_account_validate_t | ==0x72 | f7310(data+0xC) |
| 0x27 | Account | account_add（无断言） | — | 无分片 |
| 0x28 | — | default | assert("0") | — |

通道投递：`SendToAccess{Account|Role|Log|Public|Manage}(hash, msg, serial, data, size)`
→ `idx = hash % pools` → `CAccess::PutMessage(slot[idx], ...)`（LogPools==0 时直接丢弃）。

### 1.6 回执协议（CAccess::SendResponse(this, serial, id, data, size)）

DB 线程 handler 完成后**直接调用**（不经队列）：
`SendResponse → GameStore::RecvAccessResponse → CInterConnect::SendToWorld/Public/Member`
（serial==0→world；==1→public；否则 comm=(serial>>16)-0x18, idx=serial&0xFFFF）。
**RecvAccessResponse/2 开头：`StoreId > 0 → 直接 return`**——多 store 分组时仅主
store（StoreID=0）持有回包权。SendResponse2 六参变体走 *2 族（附加路由参数）。

| resp id | handler | 回执体 |
|---|---|---|
| 0x00 | MessageRoleInfo | 0x10B（分片流，节间 Sleep(1)） |
| 0x01 | MessageRoleOffline | — |
| 0x02 | MessageLoad | 0xBFB（存档全量） |
| 0x03 | MessageSave | 0x52B |
| 0x04 | MessageBlock | 0x4EB |
| 0x05 | MessageResume | 0x4EB |
| 0x06 | MessageCreate / MessageRoles | 0x90B |
| 0x07 | MessageDelete | 0x90B |
| 0x09 | MessageChunkInfo | — |
| 0x0A | MessageLoadChunk | 0x52B+ |
| 0x0B | MessageSaveChunk | — |
| 0x0C | MessageDeleteChunk | — |
| 0x0D | MessageSendLetter | — |
| 0x0E | MessageQueryLetter | 0x52B |
| 0x0F | MessageRecvLetter | 0x6FB |
| 0x10 | MessageLookLetter | 0x73B |
| 0x11 | MessageCleanLetter | 0x56B |
| 0x12 | MessageRejectLetter | 0x4EB |
| 0x13 | MessagePubInfo | 0x4AB |
| 0x14 | MessagePubLoad | 0xB1B |
| 0x15/0x16/0x17 | MessagePubSave/Add/Remove | 0x88B |
| 0x18 | MessageAccountReset | — |
| 0x19 | MessageAccountLogon | 0x35B |
| 0x1A | MessageAccountLogoff | 0x31B |
| 0x1B | MessageAccountValidate | 0x35B |
| — | SetOnline/SetOffline/Illegal/LogData/ItemLog/ChatLog/GmLog/CustomLog/OutputXml/InputXml/AccountAdd | 无回执 |

serial 编码与 solo 一致：高 16 位=源 comm、低 16 位=请求方序号（原样带回）。

### 1.7 CAccess 线程模型（access.cpp）

```
GameStore 主线程 SendAccessRequest ──► CAccess::PutMessage ──► CMsgDeque2
CAccess 线程（_beginthread AccessThreadProc）:
  InitConnect: CoInitializeEx(0,0) → CAdoConn::Open(GetConnectString(this+0x10))
  loop: while !quit: DoWait(); Sleep(1)
  DoWait: IsEmpty→SwapMsgVector→Peek→ProcessMsg(CheckConnect 后 switch 0..0x27)
  handler → CAdoConn SQL → SendResponse 直发（线程安全经 Adapter）
Stop: quit=1 → WaitForSingleObject → 句柄清零
```

CAccess 布局：`+0x00 store` `+0x04 线程句柄` `+0x08 quit` `+0x0C ServerId`
`+0x10 连接串(TFastStr)` `+0x30 CMsgDeque2*` `+0x34 CAdoConn*` `+0x38 fail 计数`
`+0x3C..0x123C 0x1200 工作缓冲`。
断链：`CheckConnect = GetConnected + Check(重连)`；失败 `FailCanRetry =
SetConnFailed && fail<2` 重试。**网络版无 wake socket、无压缩网关**
（CompressData/UncompressData 仍在但仅 BLOB 长块本地用）。

### 1.8 SQL 全谱（CAdoConn → SQLOLEDB，SQL Server 方言）

连接串原样取自通道 Base：`Provider=SQLOLEDB;Password=..;User ID=sa;
Initial Catalog=nx_base1;Data Source=..`。全部语句（宽字符 sprintf）：

- **dc_accounts**（Account 通道）：
  - logon: `UPDATE dc_accounts SET logged=%d, lastlog_time=GetDate(), lastlog_addr=N'%S' WHERE name=N'%s' AND status=1 AND (logged=0 OR logged=%d)`
  - logoff: `UPDATE dc_accounts SET logged=0 WHERE name=N'%s' AND logged=%d`
  - validate: `SELECT user_type FROM dc_accounts WHERE name=N'%s' AND password=N'%s' AND status=1`
  - reset: `UPDATE dc_accounts SET logged=0 WHERE status=1 AND logged=%d`
  - add: `INSERT INTO dc_accounts (name,password,create_time,lastlog_time,lastlog_addr,valid_time,user_type,status,logged,charge_mode,points,used_points,comment) VALUES (N'%s%d',N'%s%d',GetDate(),GetDate(),N'',GetDate(),0,1,0,0,0,0,N'')`
- **nx_roles**（Role 通道）：roles(`SELECT name,role_index,sys_flags,save_para1 FROM nx_roles WHERE account=N'%s' AND server_id=%d AND deleted=0`)、
  load(`SELECT * .. WHERE account/name/server_id`)、create（22 列 INSERT，save_data=NULL 初建）、
  delete(`UPDATE SET deleted=1, delete_time=GetDate()`)、save(`UPDATE SET total_sec,letter_max,scene_id,location,save_para1,save_para2 WHERE name`)、
  set online/offline、illegal 计数、block(open_time)/resume(resume_time+resume_info)、
  `UPDATE nx_roles SET online=0 WHERE online=1 AND server_id=%d`（启动清场）、
  save_data 全表/单名 SELECT（OutputXml 用）＋分块 `SELECT save_data FROM %S`。
- **nx_chunks**：按 (name,server_id,scene_id) 增删查；save_data BLOB 参数化。
- **nx_letter**：serial_no(GUID)/msg_send_time/msg_time/msg_name/msg_uid/
  msg_source/msg_source_uid/msg_type/msg_content/msg_appendix；TOP 100/TOP 1 查询、
  拒收/清理按 `GetDate()-msg_send_time > %f` 过期。
- **nx_logdata / nx_itemlog / nx_chatlog / nx_gmlog**：固定 INSERT。
- **动态表 `%S`（Public 通道）**：`SELECT name FROM %S` / `SELECT * FROM %S WHERE name=N'%s'` /
  `INSERT INTO %S (name,uid,create_time,deleted,save_time,save_data)` /
  `DELETE FROM %S WHERE name=N'%s'`——即 nx_base save-data 容器族
  （AddLogTable/AddLogField 运行期建表，12 张运营日志表结构见 main.lua）。

`%S`=宽转窄、`%s`=宽字符原样（SQL Server N'' 前缀）。

### 1.9 序列化

CStoreArchive/CLoadArchive（Write/Read：Byte/Int/Int64/Float/Double/String/
WideStr/Bool/Object + Seek/ExpandBuffer）——save_data BLOB 容器与
`docs/official/nx_base-save_data容器格式.md` 的 OBJ3 同族；GUID 由
`CAccess::CreateGuid`（CoCreateGuid）生成 serial_no。

---

## 二、FsSql.dll：DB 抽象模块

### 2.1 构成

静态链接：**libmysqlclient 全量 C API**（mysql_server_init/real_connect/
stmt_prepare/stmt_bind/escape_string/…）+ **yaSSL/TaoCrypt**（SSL 加密栈，
MySQL 4.1+ SSL 连接用）+ MS CRT undname 等。另含 ADO COM 封装类
（Connection15/Command15/Recordset15/_com_ptr_t 族）。

### 2.2 实体注册

- `MysqlDBSystemEntCreator`：GetName="MysqlDBSystem"，Create→`new MysqlDBSystem`
- `SqlDBSystemEntCreator`：GetName="SqlDBSystem"
- vtbl（IEntity 三纯虚槽落地）：

```
IDBSystem:      Init / Shut / (IEntity: Execute/Display/MsgProc/Release/GetMemoryUsage)
                +0x20 Startup / +0x24 Cleanup / +0x28 CreateConnection
IDBConnection:  Release/GetDBSystemType/Initialize/Shutdown/
                Set+Get ConnectString/ConnectTimeOut/CommandTimeOut/
                GetErrorMessage/Open/Close/Check/GetConnected/SetConnectFailed/
                Execute/OpenRecordset/OpenRecordsetForward/CreateCommand/
                GetDatetimeString/GetLimitPrefix/GetLimitSuffix    （24 槽）
IDBCommand:     AddParam{Int,Float,Double,String,WideStr,Datetime,Binary}/
                GetParamOut{同族+Size}/Execute/OpenRecordset       （19 槽）
IDBRecordset:   GetRecordCount/EndOfFile/MoveFirstRecord/MoveNextRecord/
                GetField{Count,Name,Type,Size,Int,Float,Double,String,WideStr,
                Datetime,Binary} + 全部 ByIndex 变体               （24 槽）
```

方言差异收敛点：`GetDatetimeString`（MySQL `Now()` vs SQLServer `GetDate()`）、
`GetLimitPrefix/Suffix`（MySQL `LIMIT n` 尾缀 vs SQLServer `TOP n` 头缀）。

### 2.3 MysqlDB 后端

- `MysqlDBSystem::Startup = mysql_server_init(-1, 0, 0)`；Cleanup 对应收尾。
- `MysqlDBConnection::Open`：**解析 ADO 风格连接串**——`strstr` 提取
  `Password=`/`User ID=`/`Initial Catalog`/`Data Source`（支持 `host:port`），
  转 `Connect(host, port, user, passwd, db)` → mysql 系连接；失败
  `"MysqlDBConnection::Open connect..."` trace。
- Command 走 **prepared statement**（mysql_stmt_prepare/stmt_bind_result，
  map 中 mysql_stmt_param_count 等实证）；Recordset 走 mysql_fetch_row 族。

### 2.4 SqlDB 后端（ADO）

`SqlDBConnection::Open`：`CoCreateInstance(__uuidof(ADODB.Connection))` →
`PutConnectionTimeout/PutCommandTimeout` → `Connection15::Open(bstrConnStr,
"","",0)`——连接串**原样透传**给 SQLOLEDB。`trace_com_error` 统一异常。

> 即：**同一份 ini 连接串，双后端通吃**——这是 GameStore.ini 只配一列
> ConnectString、UseMysql 一键换后端的设计基础。

---

## 三、Go 复刻方案

### 3.1 包结构（建议）

```
internal/fsdb            ← FsSql 对应物（先做，solo 系也能复用）
    connstr.go           ADO 串解析: SplitAdeoConn("Provider=..;Password=..;User ID=..;Initial Catalog=..;Data Source=host[:port]")
    system.go            DBSystem 接口: Startup/Cleanup/CreateConnection
    connection.go        接口: Open/Close/Check/Execute/OpenRecordset/CreateCommand/Datetime/Limit 方言对
    command.go / recordset.go    参数绑定与行读取（database/sql 之上）
    mysql.go / mssql.go  方言实现（Limit 前后缀/Datetime/占位符转换）
internal/fsstore         ← FsStore 对应物
    gamestore.go         五通道池 + 状态机(4 态) + tick(1s 广播+Closing 检查)
    wire.go              0xEE 转义编解码 + 12B 头 {comm,serial,msg} + 绑定帧 {comm,0,0xFFFF}
    route.go             comm 空间(world=0/public=1/member=0x18+n/store=id+4)
                         入站三分支 / 出站 SendToWorld/Public/Member / StoreId>0 回执抑制
    dispatch.go          41 型上行表 + hash 三变体(BKDR-5) + hash%pools 分片
    access.go            每通道 goroutine 池（CMsgDeque2 → chan）+ 重试(<2)
    handlers_role.go     0x00..0x12 Role 族 + chunk 族 + letter 族
    handlers_pub.go      0x1E..0x22 动态表族
    handlers_account.go  0x23..0x27 dc_accounts 族
    handlers_log.go      0x0D..0x11 日志族 + AddLogTable/AddLogField 建表
    archive.go           CStoreArchive/CLoadArchive（对齐 nx_base OBJ3 容器）
    echo.go              15100+StoreID 维护口(quit/end)
cmd/gamestore/main.go    GameStore.ini 装配（对应 MainConsole+main.lua）
```

### 3.2 关键实现要点

1. **连接串**：单解析器输出 `{Provider,User,Password,Database,Host,Port}`；
   MySQL 走 `user:pass@tcp(host:port)/db`，SQLOLEDB 场景换 driver。
   `%S/%s` 宽窄转换在 SQL 模板层统一（Go 全 UTF-8，直接参数化替代拼接，
   消除注入面并保留语义）。
2. **wire 层**（与 fssolomember/fssoloworld 的 inner 协议族对齐）：
   `EncodeFrame(payload)`：0xEE→`EE 00`，尾 `EE EE`；`DecodeFrame` 状态机
   对齐 OnReceive；头 12B 小端。绑定帧收到即登记 `conn→comm`。
3. **分片**：`hashW(wide) / hashFold(byte) / hashRaw(byte)`（h=h*5+c，
   fold 表按 ASCII 大小写折叠）；`idx = hash % pools`。
4. **回执**：handler 返回 `(respID, body)`；`StoreId > 0` 直接丢弃；
   serial 高 16 位还原 comm、低 16 位透传。
5. **状态机**：Closing 态轮询 `AccessEmpty()`（各通道队列长度和+在途计数），
   排空后向 world 发 `msg=0x1C {1, StoreId}` 再进 Closed。
6. **广播**：每秒 UDP `255.255.255.255:15000` 发 0x28B 自宣帧
   `{2,0x300,addr[16],port,district,server,store}`（注意网络版协议号
   0x300 ≠ solo 0x3001）。
7. **stdout 兼容**：echo 维护口、`Store adapter %s:%d ...` 日志族按原文案输出。

### 3.3 与既有 Go 代码的衔接

- `internal/fscomm` 已有 Adapter/实体协议基础 → wire.go 复用其帧层或按上表独立实现（0xEE 转义若已在 fscomm 实现，直接引用）。
- `docs/official/nx_base-save_data容器格式.md` + `save-data-OBJ3-player-schema.txt` 直接支撑 archive.go 与 save_data 读写。
- FsSoloStore 未建 Go 包；**先落 fsdb（FsSql 对应物）再落 fsstore**，solo 网络面差异（CInnerHook/wake 回路）后续按姊妹篇单独补。

---

## 四、遗留未决（低风险待证）

- PutMessage2/SendMsg2 的附加两参语义（优先级/频道；仅影响 *2 回执路径，主路径不受影响）。
- resp id 0x08 空号（ChunkInfo 与 Roles 具体绑定以实测抓包复核为准）。
- CAdoConn 31 方法中 GetFieldImage/UpdateImage 的 SAFEARRAY 细节（BLOB 通道已由 SQL 模板证实）。
- MysqlDBCommand 参数占位符形态（`?` 与 stmt 绑定映射，复刻直接用 database/sql 占位符即可）。
