# FsSoloStore.dll 全解与 Go 复刻

> 逆向基准：`BGserverCJ/Debug/FsSoloStore.dll`（1,159,168 B，2014-03-22，
> MSVC .map 2,333 符号全量在手）+ `Lua/solostore/main.lua`（33 KB 官方装配
> 蓝图）+ `game_store.ini`。源码路径（断言串泄露）：
> `c:\wuyf\svn\serveriii\branches\publish20130508\server\fssolostore\`
> （solostore.cpp / solostorestate.cpp / accessrole.cpp / accesslog.cpp /
> taskqueue.cpp）。工具：`tmp_research/solostore/store_dis.py`（map 符号 +
> capstone 注释反汇编 CLI，复用 solomember 的 sm_dis.py）。

---

## 〇、总览

FsSoloStore.dll 是 **solo 单机集群的存储/DB 枢纽**（小写系
`game_store.exe` 直启，MainEntity=SoloStore）。集群内一切持久化——
角色存档、场景块、信件、公共数据、管理配置、SNS 关系/动态、交易行、
拍卖行、全部运营日志——都经 inner 网发给它，由它落库。

| 维度 | 定案 |
|---|---|
| 模块版本 | FxModule_GetVersion → 1003 族（solo 同代） |
| 网络面 | 内网 listener（缺省 0=系统分配）+ UDP 广播自宣（15000）+ echo 维护口（**15100+StoreID**）+ wake listener（缺省 0，线程唤醒回路） |
| 广播自宣 | kind=**2**（store），0x28B 帧 `{2, 0x3001, InnerAddr[16], InnerPort, DistrictId, ServerId, StoreId}`（BroadcastStoreInfo @0x100C4D60 实证：+0x08=InnerAddr 串、+0x18=实际内网口、+0x24=StoreId 原值；member 端 `CheckStoreConnect` 断言 <4 后 **+4** 当 comm 用——两侧互证） |
| comm 分配 | `SetCommIdent(StoreId+4)` → **4..7**（world=0/public=1/auction=2/entry=3/store=4+n/member=0x18+n，与 fssolomember wire.go 注释互证） |
| DB 通道 | **5 通道 × N 池**：Account(nx_district)/Role(nx_base)/Log/Public/Manage，每通道 `StartAccess` 建 CAccess 线程池 |
| 命令空间 | 上行 msg 0x00..0x42（67 型），下行回执独立 id 空间（§四） |
| 状态机 | 4 态 Unknown/Opened/Closing/Closed（this+0x34），比 member 少 Loading |
| 分周日志表 | Opened 态每拍 `SHOW TABLES LIKE '%s%s'` 缺则建，suffix=`%04d%02d%02d`（上周某日），广播给所有 CAccess |

## 一、对象布局（SoloStore@this，map 实证）

```
+0x2C  UseThread(b)         +0x30  UseQuickLog(b)      +0x31  UseAppRecord(b)
+0x32  UseLogSerial(b)      +0x34  state(i)  四态
+0x38  DistrictId           +0x3C  ServerId            +0x40  StoreId
+0x44  AccountPools         +0x48  RolePools           +0x4C  LogPools
+0x50  PublicPools          +0x54  ManagePools         +0x5C  SplitWeek?
+0x68  分周维护门(≠0 跳过)
+0x80/0xA0/0xC0/0xE0/0x100  Account/Role/Log/Public/Manage Base (TFastStr[0x40])
+0x120..0x240 step 0x40     五通道 CAccess* 首槽
+0x260  InnerAddr(TFastStr) +0x280  InnerPort(实际)
+0x288  BroadAddr           +0x2A8  BroadPort
+0x2C8  EchoAddr            +0x2CC  EchoPort
+0x2E8  WakeAddr            +0x2F0  WakePort(实际)
+0x330  IDBSystem*          +0x33C  CInnerHook*
+0x344  CAppRecord*         +0x348  CThread*(UseThread)
+0x34C  响应 CTaskQueue      +0x350  log_table_set_t*
+0x354  log_creator_set_t*  +0x358  广播句柄
+0x368  CLockVal database_break +0x36C  CLockVal db_err
```

对外方法面（lua 实配）：Start/Stop/SetAppState/WriteLine/SetConsoleColor/
AddLogTable/AddLogField/AddLogCreator/SendEchoLine/CloseEcho/AccessEmpty/
DumpReactor + 26 组属性 Get/Set（AppRecord 呈现 `<info state database db_err tick/>`）。

## 二、Start()/Stop() 装配序（0x100BFDF0）

1. UseQuickLog → new CLogShareMem → SetAppId/Start/SetLogPrefix/SetCreateNew/
   挂 TraceFunc（失败仅 trace）
2. LogLauncher 非空且 LogShareMem 不存在 → `Port_Execute("game_log game_log.ini daemon")`
3. UseAppRecord → new CAppRecord → SetAppId/SetAppName/SetAppAddr/Start
4. CInnerHook::SetCallback(4 回调) + SetAppName + **SetCommIdent(StoreId+4)**
5. CReactor::Start → CreateBroadcast(BroadAddr:15000)
6. EchoAddr 非空 → CEchoHook::Start（欢迎语 lua 拼装：
   `welcome server %d store %d\r`，quit=关进程 / end=断开）
7. CreateListener(InnerAddr:InnerPort) → GetListenerPort 回写实际口
8. CreateListener(WakeAddr:WakePort) → 回写实际口（DB 线程唤醒回路受面）
9. CreateTimer(每拍) → TimerCheck
10. DBSystem≠nil → vtbl+0x20 调用（池注册）
11. 五通道 `StartAccess(&slot, pools, base, is_log)`：
    - 断言 access/num>0/conn_str；`get_real_connect_string` 解码
    - new CAccess → SetServerId/SetConnectString/SetWakeAddress(唤醒口)/Start
    - Start 失败重试下一池
12. UseThread → 额外 CThread(0x100C1070, this+0x334)+Start（主循环卸载开关）
13. `StoreStateMachine(1,0,0)` → Unknown→Opened

Stop()（0x100C10C0）：StopAccess 五通道（CTaskQueue 唤醒+CThread 停）、
关广播/timer/listener/echo、CAppRecord/LogShareMem Stop。

## 三、CAccess 线程模型（DB 执行链）

```
主线程 ProcessInnerMsg ──PutMessage──► CTaskQueue ──wake字节──► wake socket
                                                                      │
CAccess 线程: AccessInitProc → AccessLoopProc{ DoWait } ◄─────────────┘
  DoWait: IsEmpty? CThreadWaiter::Wait → SwapMsgVector → Peek → ProcessMsg
  ProcessMsg(0x1008D8A0): CheckConnect 后跳表 0x42 上限 67 型 → Message*
  Message* → IDBSystem(SQL) → SendResponse/SendResponse2
  SendResponse: SoloStore::RecvAccessResponse(回调) → 响应 CTaskQueue.Put
              + Port_SocketSend(wake_sock, 1B)   // 唤醒主线程
主线程 wake listener 收字节 → CheckResponse: SwapMsgVector → Peek
  serial==0 → CInnerHook::SendToWorld(0, id, data, size)
  serial==1 → SendToPublic   serial==2 → SendToAuction
  其余      → comm=(serial>>16)-0x18, idx=serial&0xFFFF
             → SendToMember(comm, idx, id, data, size)
```

- **serial 编码**：高 16 位=源 comm（member 连入即 0x18+MemberID；world/
  public/auction 直发 0/1/2），低 16 位=请求方自带序号（回包原样带回）。
- 池>1 时 `SendToAccessRole` 以 `serial % (pools-1) + 1` 分片负载。
- 连接失败：ConnectFailed/FailCanRetry(<2 次)重试；断链计数经
  CLockVal(+0x368/+0x36C) 呈现到 AppRecord `database="Break"`。
- save_data BLOB 走 SQL 参数 `?`；超长块 CompressData/UncompressData
  （GetCompressBufferSize/GetUncompressBufferSize，zlib 族）。
- 连接串混淆：`encode_string/decode_string(key="SnailGameCharge")`
  （charge 同源方言）；明文串解码失败时原样透传（两可）。

## 四、命令协议（67 型全表）

### 上行 SendAccessRequest 跳表（@0x100C4160，switch=msg，0..0x42）

| msg | 通道 | 结构体（尺寸断言） | CAccess 处理器 → 回执 id |
|---|---|---|---|
| 0x00 | Role | role_info(0x258B 定长) | MessageRoleInfo → **0x00**（分片流：res16B+archive，节间 Sleep(1)） |
| 0x01 | Role | — | MessageRoleOffline → 0x01 |
| 0x02 | Role | load==188 | MessageLoad → 0x02（0xD0B） |
| 0x03 | Role | load_cancel==148 | **PutLoadCancel 原地取消**（不入队） |
| 0x04 | Role | save≥160 | MessageSave → 0x03（0x58B） |
| 0x05 | Role | set_online≥148 | MessageSetOnline（UPDATE online=1，无回执） |
| 0x06 | Role | set_offline≥148 | MessageSetOffline（无回执） |
| 0x07 | Role | illegal==72 | MessageIllegal → illegals 计数 |
| 0x08 | Role | block==104 | MessageBlock → 0x04（0x54B） |
| 0x09 | Role | resume==168 | MessageResume → 0x05 |
| 0x0A | Role | create≥212 | MessageCreate → 0x06 |
| 0x0B | Role | delete==172 | MessageDelete → 0x07（0xBCB） |
| 0x0C | Role | revive==168 | MessageRevive → 0x08 |
| 0x0D | Role | roles==96 | MessageRoles → 0x06 |
| 0x0E | Role | edit_role≥76 | MessageEditRole → **0x0B** |
| 0x0F | Role | recreate_role≥148 | MessageRecreateRole → **0x0C**/9 |
| 0x10 | Role | query_account≥72 | MessageQueryAccount → **0x0D** |
| 0x11 | Role | query_role≥76 | MessageQueryRole → **0x0E** |
| 0x12 | Log | log_data≥1104 | MessageLogData |
| 0x13 | Log | item_log≥262 | MessageItemLog |
| 0x14 | Log | chat_log≥1172 | MessageChatLog |
| 0x15 | Log | gm_log≥1104 | MessageGmLog |
| 0x16 | Log | custom_log≥36 | MessageCustomLog（动态表） |
| 0x17 | Role | custom_log≥36 | MessageCustomLog |
| 0x18 | Role | — | （透传） |
| 0x19 | Role | load_chunk≥84 | MessageLoadChunk → **0x10**（0x58B） |
| 0x1A | Role | save_chunk≥76 | MessageSaveChunk → 0x11 |
| 0x1B | Role | delete_chunk≥76 | MessageDeleteChunk → 0x12 |
| 0x1C | Role | send_letter≥364 | MessageSendLetter → **0x13** |
| 0x1D | Role | query_letter≥116 | MessageQueryLetter → **0x14** |
| 0x1E | Role | recv_letter≥152 | MessageRecvLetter → 0x15 |
| 0x1F | Role | look_letter≥116 | MessageLookLetter → 0x16 |
| 0x20 | Role | clean_letter≥172 | MessageCleanLetter → 0x17 |
| 0x21 | Role | reject_letter≥156 | MessageRejectLetter → **0x18** |
| 0x22 | Role | output_xml≥364 | MessageOutputXml |
| 0x23 | Role | input_xml≥364 | MessageInputXml |
| 0x24 | Public | pub_info≥72 | MessagePubInfo → **0x19**（0x50B） |
| 0x25 | Public | pub_load≥152 | MessagePubLoad → **0x1A**（0xC0B） |
| 0x26 | Public | pub_save≥144 | MessagePubSave → 0x1B |
| 0x27 | Public | pub_add≥180 | MessagePubAdd → 0x1C |
| 0x28 | Public | pub_remove≥144 | MessagePubRemove → **0x1D**（0x94B） |
| 0x29 | Manage | manage_info≥4 | MessageManageInfo → 0x1E |
| 0x2A | Manage | manage_base≥4 | MessageManageBase → **0x1F**（0x24B） |
| 0x2B | Manage | manage_update≥4 | MessageManageUpdate → 0x20 |
| 0x2C | Manage | manage_remove≥4 | MessageManageRemove → 0x21 |
| 0x2D | Public | relation_info≥4 | MessageRelationInfo → **0x22**（0x10B） |
| 0x2E | Public | relation_add≥76 | MessageRelationAdd |
| 0x2F | Public | relation_remove≥72 | MessageRelationRemove |
| 0x30 | Public | feed_info≥264 | MessageFeedInfo → 0x23 |
| 0x31 | Public | feed_add≥158 | MessageFeedAdd |
| 0x32 | Public | feed_remove≥36 | MessageFeedRemove |
| 0x33 | Public | feed_good_bad≥44 | MessageFeedGoodBad |
| 0x34 | Public | feed_desc≥38 | MessageFeedDesc |
| 0x35 | Account | — | （透传） |
| 0x36 | Account | account_logon==152 | MessageAccountLogon → **0x25** |
| 0x37 | Account | account_logoff==124 | MessageAccountLogoff → **0x26** |
| 0x38 | Account | — | MessageAccountReset → 0x24 |
| 0x39 | — | — | （空） |
| 0x3A | Public | trade_info≥4 | MessageTradeInfo → 0x28 |
| 0x3B | Public | trade_add≥522 | MessageTradeAdd |
| 0x3C | Public | trade_remove≥36 | MessageTradeRemove |
| 0x3D | Public | trade_update==192 | MessageTradeUpdate |
| 0x3E | — | — | default（仅 log） |
| 0x3F | Public | auction_attr_load≥4 | MessageAuctionAttrLoad → **0x29** |
| 0x40 | Public | auction_record_load≥4 | MessageAuctionRecordLoad → 0x2A/**0x2B**（LoadRecordOk） |
| 0x41 | Public | auction_attr_save≥82 | MessageAuctionAttrSave → 0x2C（与 fssoloauction PUB4 块协议对上） |
| 0x42 | Public | auction_record_save≥172 | MessageAuctionRecordSave → 0x2D |

通道分片：0x00–0x11/0x17–0x23 → Role；0x12–0x16 → Log；
0x24–0x28/0x2D–0x34/0x3A–0x3D/0x3F–0x42 → Public；0x29–0x2C → Manage；
0x35–0x38 → Account。**member 端重映射闭环**（FsSoloMember 双向定案）：
store 回执 id2 0x0B–0x0E → 场景 0x40–0x43（EditRole/RecreateRole/
QueryAccount/QueryRole 四连）、0x13–0x18 → 场景 0x21–0x26（信件族
SendLetter..RejectLetter）、0x0F–0x12 → ProcessStoreMsg chunk 族、
**0x1E–0x21 → ProcessStoreMsg manage 族**（member 字节跳表 @0x102A0398
实证：0x13..0x1D 全为 default 空槽——member 复刻此前把 manage 族记在
0x13..0x16 属误读，已随本次修正）。

### ProcessInnerMsg 路由（0x100C4EA0）

```
arg1=源comm, arg2=请求serial, arg3=msg, data, size
  arg1==0 → SendAccessRequest(0,           msg, …)  // world→Account 面
  arg1==1 → SendAccessRequest(1,           msg, …)
  arg1==2 → SendAccessRequest(2,           msg, …)
  arg1∈[0x18,0x2F] → SendAccessRequest((arg1<<16)|arg2, msg, …)
  其余 → trace "message source error! comm:%d" + return false
```

（0/1/2 分支的 serial 直用常量 → CheckResponse 还给 world/public/auction。）

## 五、SQL 全文（.rdata 0x101532E0–0x1015A0A0，逐字提取）

**Account 库（nx_district.dc_accounts）**
```sql
UPDATE dc_accounts SET logged = 0 WHERE status = 1 AND logged = %d
UPDATE dc_accounts SET logged = %d, lastlog_time = NOW(), lastlog_addr = '%s'
  WHERE name = '%s' AND status = 1 AND (logged = 0 OR logged = %d)     -- logon
SELECT * FROM dc_accounts WHERE name = '%s' AND status = 1 AND (logged = 0 OR logged = %d)
SELECT * FROM dc_accounts WHERE name = '%s' AND status = 1             -- add 查重
UPDATE dc_accounts SET logged = 0 WHERE name = '%s' AND logged = %d    -- logoff
INSERT INTO dc_accounts ( name, password, create_time, lastlog_time,
  lastlog_addr, valid_time, user_type, status, logged, charge_mode,
  points, used_points, comment ) VALUES ( '%s%d', '%s%d', NOW(), NOW(),
  '', NOW(), %d, 1, 0, 0, 0, 0, '' )                                   -- add（前缀号）
-- 同上 '%s' 直名变体；logon 成功 trace: "create account %s password %s …"
```

**Role 库（nx_base*.nx_roles / nx_letter / nx_chunks）**
```sql
-- 引导/角色清单
SELECT name, uid, deleted FROM nx_roles WHERE server_id = %d           -- RoleInfo 流
SELECT COUNT(*) as count FROM nx_roles WHERE server_id = %d
UPDATE nx_roles SET online = 0 WHERE online = 1 AND server_id = %d     -- 开机复位
SELECT name, role_index, sys_flags, save_para1, deleted, delete_time, charge_info
  FROM nx_roles WHERE account = '%s' AND server_id = %d AND deleted = 0  -- 列角（有档）
-- 同列 ORDER BY save_time DESC（无档全量）
SELECT COUNT(*) as count FROM nx_roles WHERE account = '%s' AND server_id = %d AND deleted = 0
-- 存档
SELECT * FROM nx_roles WHERE account='%s' AND name='%s' AND server_id=%d AND deleted=0
SELECT save_data FROM nx_roles WHERE name = '%s'（LIMIT 变体 %s）
INSERT INTO nx_roles ( name, uid, account, create_time, delete_time, open_time,
  server_id, total_sec, deleted, letter_max, sys_flags, illegals, online,
  role_index, scene_id, location, resume_time, resume_info, charge_info,
  save_para1, save_time, save_data ) VALUES ( …, ? )                    -- save_data 参数
UPDATE nx_roles SET total_sec=%d, letter_max=%d, scene_id=%d, location='%s',
  charge_info='%s', save_para1='%s', save_time=NOW() WHERE name='%s'   -- save 元数据
UPDATE nx_roles SET total_sec=%d, … save_data=? WHERE name='%s'        -- save 含档
-- 在线/封禁/复活/删除
UPDATE nx_roles SET online = 1 WHERE name = '%s' AND online <> 1
UPDATE nx_roles SET online = 0 WHERE name = '%s' AND online <> 0
UPDATE nx_roles SET illegals = %d WHERE name = '%s'
SELECT illegals FROM nx_roles WHERE name='%s' AND server_id=%d AND deleted=0
UPDATE nx_roles SET open_time = '%s' WHERE name = '%s'                 -- block
SELECT name FROM nx_roles WHERE name='%s' AND server_id=%d AND deleted=0 AND open_time > NOW()
UPDATE nx_roles SET resume_time='%s', resume_info='%s' WHERE name='%s' -- resume 存挂机
SELECT %s account, name, resume_time, resume_info FROM nx_roles
  WHERE resume_time < '%s' AND server_id=%d AND deleted=0
    AND resume_info <> '' AND online=0 ORDER BY resume_time %s          -- 捞回（scene 变体）
-- 删除/重建
UPDATE nx_roles SET deleted=1, delete_time=NOW() WHERE name='%s' AND account='%s'
DELETE FROM nx_roles WHERE name='%s' AND account='%s'                   -- 删档（超期）
SELECT create_time, save_time, uid FROM nx_roles WHERE name='%s' AND account='%s'
UPDATE nx_roles SET deleted=0, save_time=NOW() WHERE name='%s' AND account='%s' AND deleted=1  -- revive
UPDATE nx_roles SET deleted=0, role_index=%d, save_time=NOW() WHERE … deleted=1  -- recreate
-- 信件
INSERT INTO nx_letter ( serial_no, msg_send_time, msg_time, msg_name, msg_uid,
  msg_source, msg_source_uid, msg_type, msg_content, msg_appendix )
  VALUES ( '%s', NOW(), '%s', '%s', '%s', '%s', '%s', %d, '%s', '%s' )
SELECT %s * FROM nx_letter WHERE msg_name='%s' AND msg_uid='%s' ORDER BY msg_send_time %s
SELECT uid, letter_max FROM nx_roles WHERE name='%s' [AND uid='%s'] AND deleted=0
SELECT COUNT(*) as count FROM nx_letter WHERE msg_name='%s' [AND msg_uid='%s']
UPDATE nx_letter SET msg_name='%s', msg_uid='%s', msg_source='%s',
  msg_source_uid='%s', msg_type=%d WHERE serial_no='%s'                 -- recv/look
DELETE FROM nx_letter WHERE serial_no='%s'                              -- clean
SELECT serial_no, msg_name, msg_uid FROM nx_letter WHERE msg_source='%s'
  AND msg_source_uid='%s' [AND msg_type=%d AND msg_send_time<'%s']      -- reject
-- 场景块
SELECT name FROM nx_chunks WHERE server_id=%d AND scene_id=%d           -- ChunkInfo
SELECT save_data FROM nx_chunks WHERE serial_no='%s' AND server_id=%d   -- LoadChunk
INSERT INTO nx_chunks ( serial_no, name, server_id, scene_id, create_time,
  deleted, save_time, save_data ) VALUES ( '%s','%s',%d,%d,NOW(),0,NOW(),? )
UPDATE nx_chunks SET name='%s', scene_id=%d, save_time=NOW(), save_data=?
  WHERE serial_no='%s' AND server_id=%d
DELETE FROM nx_chunks WHERE serial_no='%s' AND server_id=%d
```

**Public 库（nx_config / 公共空间表 %s）**
```sql
SELECT id, op_name, op_type, op_property, op_status, op_version FROM nx_config WHERE op_name <> ''
SELECT COUNT(*) as count FROM nx_config WHERE op_name <> ''             -- PubInfo
SELECT name FROM %s                                                    -- PubLoad 全量
SELECT * FROM %s WHERE name = '%s'                                     -- PubLoad 单
UPDATE %s SET save_time=NOW(), save_data=? WHERE name='%s'             -- PubSave
SELECT name FROM %s WHERE name='%s'
INSERT INTO %s ( name, uid, create_time, deleted, save_time, save_data )
  VALUES ( '%s','%s',NOW(),0,NOW(),? )                                 -- PubAdd
DELETE FROM %s WHERE name='%s'                                         -- PubRemove
INSERT INTO nx_config ( id, name, type, property, status, op_name, op_type,
  op_property, op_status, op_version ) VALUES ( '%s',?,%d,?,%d,'',0,'',0,0 )
UPDATE nx_config SET name=?, type=%d, property=?, status=%d WHERE id='%s'   -- ManageBase
UPDATE nx_config SET op_name=?, op_type=%d, op_property=?, op_status=%d,
  op_version=%d WHERE id='%s'                                          -- ManageUpdate
UPDATE nx_config SET op_name='',op_type=0,op_property='',op_status=0,op_version=0 WHERE id='%s'
DELETE FROM nx_config WHERE id='%s'                                    -- ManageRemove
```

**Manage/SNS/交易/拍卖**
```sql
SELECT * FROM nx_sns_relation / SELECT COUNT(*) as count FROM nx_sns_relation
UPDATE nx_sns_relation SET range_mask='%d', create_time=NOW() WHERE uid='%s' AND target_uid='%s'
INSERT INTO nx_sns_relation ( serial_no, uid, target_uid, range_mask, create_time )
  VALUES ( '%s','%s','%s',%d,NOW() )
SELECT serial_no FROM nx_sns_relation WHERE uid='%s' AND target_uid='%s'
DELETE FROM nx_sns_relation WHERE uid='%s' AND target_uid='%s'
SELECT * FROM nx_sns_feed ORDER BY create_time / COUNT(*)
DELETE FROM nx_sns_feed WHERE create_time<'%s' AND level=%d
INSERT INTO nx_sns_feed ( feed_id, uid, target_uid, range_mask, level, category,
  desc_info, good_count, bad_count, comment_at, create_time )
  VALUES ( '%s','%s','%s',%d,%d,%d,'%s',0,0,'%s',NOW() )
DELETE FROM nx_sns_feed WHERE feed_id='%s'
UPDATE nx_sns_feed SET good_count=%d, bad_count=%d WHERE feed_id='%s'
UPDATE nx_sns_feed SET desc_info='%s' WHERE feed_id='%s'
SELECT * FROM nx_trade / COUNT(*)
INSERT INTO nx_trade ( serial_no, seller_name, seller_uid, item_type, item_name,
  item_info, item_cat1, item_cat2, item_cat3, status, price_min, price_max,
  time_beg, time_end ) VALUES ( … )
DELETE FROM nx_trade WHERE serial_no='%s'
UPDATE nx_trade SET buyer_name='%s', buyer_uid='%s', buyer_count=%d, status=%d,
  price_deal=%d, time_deal='%s' WHERE serial_no='%s'
SELECT * FROM nx_auction WHERE type = '1'                              -- attr 全量
SELECT * FROM nx_auction WHERE type = '2'                              -- record 全量
INSERT INTO nx_auction ( name, uid, create_time, type, record, version, amount,
  `range`, save_time, save_data ) VALUES ( '%s','%s',NOW(),1,'attr',%d,%d,0,NOW(),? )
UPDATE nx_auction SET version=%d, amount=%d, save_time=NOW(), save_data=? WHERE name='%s'
SELECT name FROM nx_auction WHERE name='%s'
INSERT INTO nx_auction ( …, type, record, …, `range`, … ) VALUES ( '%s_%s_%d','%s',NOW(),'2','%s',%d,%d,'%d',NOW(),? )
UPDATE nx_auction SET amount=%d, `range`=%d, save_time=NOW(), save_data=? WHERE name='%s_%s_%d'
SELECT name FROM nx_auction WHERE name='%s_%s_%d' AND type='2'
```

**日志通道（BatchInsertLog / 各 Message*Log）**
```sql
INSERT INTO %s (%s) VALUES (%s)                       -- CustomLog 动态表+分周后缀
INSERT INTO %s ( log_time, log_name, log_source, log_type, log_content, log_comment )
  VALUES ( NOW(),'%s',%d,%d,'%s','%s' )               -- LogData（serial 变体多一列）
INSERT INTO %s ( item_time, item_id, item_name, item_user, item_op, item_reason, item_comment )
  VALUES ( NOW(),'%s','%s','%s',%d,'%s','%s' )        -- ItemLog
INSERT INTO %s ( chat_time, chat_from, chat_to, chat_type, chat_content, chat_comment )
  VALUES ( NOW(),'%s','%s',%d,'%s','%s' )             -- ChatLog
SELECT save_data FROM %s WHERE name='%s'（三类库全量/单条变体）            -- 捞档
```

## 六、自定义日志表 + 分周分表（main.lua init_custom_log + solostorestate.cpp）

- `AddLogTable(name)`：log_table_set_t（计数上限 **32** 表；入口 0x420B：
  `+0x04 表名`、`+0x24 建表 SQL 模板`）——重复报 "table already exists"。
- `AddLogField(table, field, type, len)`：类型仅 datetime/nvarchar/varchar/
  decimal/int；越界 "field too much"/"table not found"/"field type error"。
- main.lua 实配 **29 表**（item_log/deal_log/game_player_action_log/role_log/
  task_log/instance_log/instance_role_log/instance_npc_log/scene_log/
  punish_log/program_log/total_capital_log/player_exp_log/deed_log/
  role_transmit_log/role_experience_log/role_item_change_log/role_login_log/
  role_scene_log/sys_rank_log/role_task_log/role_guild_log/role_cre_del_log/
  role_upgrade_log/role_item_log/role_auction_log/role_enjoy_log/
  role_illegal_req_log/sys_total_capital_snap）。
- **Opened 态每拍**：Port_GetLocalTime → 若到分周点（SplitWeek）→
  get_real_connect_string(Manage/Role) → CSyncAccess（同步直连）→
  `GetLastWeekDay` → suffix=`sprintf("%04d%02d%02d")` → 逐表
  `SHOW TABLES LIKE '%s%s'` 缺则 `SafeSprintf(建表模板, suffix)` 执行 →
  成功后 **SendSplitSuffix**（PutMessage(MessageSplitSuffix, suffix) 到全部
  CAccess）→ 之后所有日志 INSERT 用 `表名_suffix`。
- MessageCustomLog：`get_vary_message(fieldCount, CLoadArchive, TVarList)`
  读动态值 → 查 log_table_set → 拼 `INSERT INTO %s(%s) VALUES(%s)`；
  未定义表 trace "table not define"。值格式化：`%04d%02d%02d %d:%d:%d` /
  `%d-%d-%d %d:%d:%d` / NOW()。
- BatchInsert：SqlBatchNum 条合批执行（LogDBPerformance 可记录耗时）。

## 七、状态机（solostorestate.cpp，this+0x34）

```
事件1(Start)                          事件2 + AccessEmpty
0 Unknown ──────────► 1 Opened ────────────────► 2 Closing ──► 3 Closed
           事件0=Timer：无迁移          │ 未能 AccessEmpty：留态等待
                                       └ SendToWorld(关闭帧) 失败：
                                         "send close store failed" 重试
```

- Unknown：事件 1 → Opened；其余 trace `(SoloStore::StoreState%s)event %d`。
- Opened：事件 0=TimerCheck → 分周表维护（§六）；事件 2 → Closing。
- Closing：`AccessEmpty()`（全部 CTaskQueue 空）后才迁 Closed，并发
  `SendToWorld(0, 0x1F 族, {1, StoreId?})` 关播（失败重试）。
- Closed：终态，无迁出（GetState 串表 "Unknown/Opened/Closing/Closed"）。
- TimerCheck（0x100C5030，每拍）：CheckResponse → AppRecord（**每 5 拍**
  `g_tick%5==0` 刷新 `<info state="%s" database="%s" db_err="%d" tick="%d"/>`，
  database 取 CLockVal Break/OK）→ BroadcastStoreInfo → StoreStateMachine(0)。

## 八、echo 维护口（Lua\solostore\main.lua echo_*）

行协议（\r 终结）：连入即 `welcome server {ServerId} store {StoreId}\r`；
`quit` → close() → Stop + nx_quit；`end` → CloseEcho；其余
`>undefined command: %s\r`。另有 getp/setp/getc/setc/method/listp/listc/
listm/listg/function/debugger 十一类内省命令（框架通用，与其它 solo 进程
同构，复用 fssoloentry/fssoloauction EchoConsole 模式即可）。

## 九、Go 复刻（internal/fssolostore + cmd/store）

**范围**（对齐既有 solo 复刻取舍——框架/协议/SQL 语义层复刻，DB 以
接口注入）：

1. **reactor 装配**：inner listener + UDP 广播自宣（kind2 0x28B 帧，
   `+0x24=StoreId`）+ echo 控制台（15100+StoreID）+ 每秒 TimerCheck。
2. **命令路由**：ProcessInnerMsg 四分支 + SendAccessRequest 67 型表 +
   serial 编解码（高 16=源 comm-0x18→member idx / 0..2 直连）。
3. **StoreBackend 接口**：SQL 语义面（§五按通道分组的方法集），
   - mysql.go：-mysql-dsn 连 MySQL（原文 SQL 直接可用，NOW() 兼容）；
   - memory.go：内存 + JSON 快照兜底（-local-store；对齐 fssologlobal
     LocalStore 模式），分周 suffix 退化为常量空串。
4. **回执路由**：CheckResponse 等价——backend 异步执行回调经响应队列，
   serial 还源（world/public/auction/member）。
5. **分周表**：AddLogTable/AddLogField 全量登记（29 表 schema 常量），
   CreateLogTable/SendSplitSuffix 语义（后端不支持 SHOW TABLES 时跳过）。
6. **AppRecord XML**：每 5s `<info state database db_err tick/>` 快照文件。
7. **UseThread**：不做官方双线程（Go goroutine 天然并发），保留开关位。

**不复刻**：TStoreArchive/CLoadArchive 对象内核细节（收发两端透传
payload）、md5/md5License 校验链、CThread/CTaskQueue 具体实现（以
goroutine+chan 等价）、SQLOLEDB 方言（Go 侧只出 MySQL/内存两实现）。

## 十、验证口径

- 命令表 67 型与 CAccess::ProcessMsg 跳表逐 id 对齐（两张 0x43 槽表互证）；
- 回执 id 与 fssolomember storeID2ToSceneCmd 重映射（0xB..0xE→0x40..0x43、
  0x13..0x18→0x21..0x26）及 fssoloauction storeBack*（0x29/0x2A/0x2B）闭环；
- 广播帧与 member 端 KindStore 解析闭环（store 发 StoreId@+0x24，member
  `CheckStoreConnect` @0x10291190 断言 `index < MAX_STORE_NUM(4)` 后
  `CheckConnect(index+4)`；fssolomember 复刻已按此修正——直传 comm 原值
  是历史偏差）；
- SQL 全文来自 .rdata 逐字节提取（非重构），Go 侧仅参数化；
- main.lua 29 表 schema 逐字段常量化。
