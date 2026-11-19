# FsWatch 全解与 Go 复刻

> 对象：`BGserverCJ\Debug\FsWatch.dll`（847,872 B，2014-03-22）+ `FsWatch.pdb`（3,345,408 B）
> 宿主：`FsLoader.exe GameWatch.ini`（`GameWatch.bat`）；脚本 `Lua\GameWatch\main.lua` + `echo.lua`
> 官方源码路径（PDB 断言串）：`d:\blackgoldserverengine\serverengine\branches\server(20120807)\fswatch\gamewatch.cpp`、`querydb.cpp`
> Go 复刻：`internal/fswatch` + `cmd/watch`（2026-11-19 全绿）

FsWatch 是**运维看门狗服务器**（GameWatch）：单进程同时承担
①运维客户端受面（Outer:6000 文本行协议，8 大命令 + 三密码）、
②echo 维护口（Echo:15006，自报心跳）、
③受管服务器进程守护（AppManager 启停/巡活）、
④远程热更新（目录树 diff + 64KB 分块文件下发 + MD5 校验）、
⑤数据库查询代理（CAccess 线程 → MySQL → Publish* 回推）。

---

## 1. 进程形态与装载链

```
GameWatch.bat → FsLoader.exe GameWatch.ini
  [System]  Sleep=1  MainEntity=MainConsole  MainScript=main
  [Module]  FsKit.dll FsSql.dll FsComm.dll FsWatch.dll
  [Script]  Path=..\Lua\GameWatch  Reload=1
```

main.lua 装配（官方依赖注入顺序）：

| 实体 | 来源 DLL | 用途 |
|---|---|---|
| `MainConsole` | FsKit | 控制台 |
| `LogPool` | FsKit | 日志（log\watch 前缀，TraceMax=256） |
| `IniDocument` | FsKit | GameWatch.ini 读取 |
| `EchoSock` | FsComm | 维护连接（127.0.0.1:15006） |
| `MassSock` | FsComm | 运维客户端接入（Outer，MaxClient=16） |
| `AppManager` | FsComm | 进程守护（AppName=watch，NeedCheckApp=true） |
| `SqlDBSystem` | FsSql | 数据库系统 |
| **`GameWatch`** | **FsWatch** | **本 DLL 唯一注册实体** |

装配末序：`mass_sock.Closed=true → Start → echo Start → game_watch:Start() → "Game watch server started" → Closed=false`。

## 2. 自有类清单（PDB seg2 实证）

| 类 | 基类 | 职责 |
|---|---|---|
| `GameWatch` | IEntity | 主实体：8 命令分发/密码/心跳/DoWait 泵 |
| `GameWatchEntCreator` | IEntCreator | 实体注册（name="GameWatch"） |
| `CAccess` | — | DB 访问线程（AccessInitProc/AccessLoopProc/AccessShutProc + Request*×10） |
| `CQueryDb` | — | access.ini 装载 + 多 CAccess 池（GetAccess by app_name） |
| `CMassSockHook` | IMassSockHook | 运维客户端帧钩子（\r\n 编码/控制字符分段/IP 白名单） |
| `CMaintainSockHook` | IUserSockHook | EchoSock 出端钩子（心跳发送） |
| `CUpdate` | — | 目录树扫描/diff/序列化（ReadDirInfo/ReadFileInfo/DiffDirInfo/StoreArchive） |
| `CUpdateFiles`/`CUpdateDirs` | — | diff 产物（待更新文件表/待建目录表） |
| `CCpuRate` | — | PDH CPU 采样 |
| `CTickTimer`/`CThread`/`CThreadWaiter`/`CMsgDeque2`/`CMsgVector2` | — | 基础设施 |
| `CIniFile` | — | access.ini 解析（FsWatch 内嵌一份，非 FsKit IniDocument） |

## 3. 传输层：文本行协议

- **帧格式 = `payload + "\r\n"`**（CMassSockHook::MakeSendData 0x7eb40：memcpy 后补 `0D 0A`；GetSendLength=len+2；CMaintainSockHook 同构 0x7df70）。无长度头、无加密。
- **收向分段**（CMassSockHook::ReceiveMsg 0x7f0b0）：拷贝 len+1 → 末尾置 0 → **遍历把所有 `<0x20` 控制字符替换为 `\0`** → 逐段 `GameWatch::ReceiveClientMsg` → `CMsgDeque2::Put` 入队，主循环 DoWait 再 `ProcessSockMsg`。
- OnAccept 只记日志 `Address %s:%d connected, total %d`（0x7ec70）；**IP 白名单由 SetAllowAddress(0x7f280) 下沉到 FsComm MassSock 层执行**（GameWatch.ini `[Address] AddressN` 列表）。

## 4. text varlist 编解码（协议核心）

`text_decode_msg`（0x5d2d0）：按**空格（0x20 单字符）**分词；`text_append_word`（0x5d440）按首字符推断类型：
- `[0-9]`、`.`、`-` 开头 → 数字（找 `.` 分 int/float）
- `#` 开头 → widestr（逐字 `\xXX`/`\\`/`\uXXXX` 反转义）
- `$` 开头 → string（同上窄串反转义）
- `*` 开头 → userdata（**大写 HEX** 解码；码表 0x1011124c=`0123456789ABCDEF`）
- 其余 → 原样字符串

`text_encode_msg`（0x5e4a0）：两遍（测长/编码），元素间以空格 join：

| varlist 类型 | 文本形态 |
|---|---|
| int | `%d` |
| int64 | `_i64toa` |
| float/double | `%f` |
| string | `$` + 转义体 |
| widestr | `#` + 转义体 |
| userdata | `*` + **大写 HEX**（每字节 2 字符） |

转义体规则（0x5e869 串 case / 0x5e9b0 宽 case）：
- `c<=0x20 || c>=0x7f || c=='%'` → `\x` + 2 位**大写** hex
- `'\\'` → `\\\\`（双写）
- 其余可打印 → 原样

## 5. 八大命令分发表（ProcessSockMsg 0x77170）

首 token = 命令名，`_stricmp` 匹配（顺序即官方比较链）：

| 命令 | 处理器 | 密码位 | 形态 |
|---|---|---|---|
| `servers` | ReceiveServers 0x77630 | 无 | 响应 `servers <name> <cmdline> ...`（server_data_t name(+0)/path(+0x30) 交替） |
| `app_list` | ReceiveAppList 0x77830 | 无 | 转 AppManager vtable+0x3c 取应用列表回推 |
| `query` | ReceiveQuery 0x77b60 | args[2] vs QueryPassword(this+0x5c) | 10 子命令 → CAccess |
| `update` | ReceiveUpdate 0x79170 | args[2] vs UpdatePassword(this+0x64) | 4 子命令 → CUpdate |
| `sys_info` | ReceiveSysInfo 0x7bb30 | **无密码** | `sys_info <cpu> <memload/100> <total_phys> <avail_phys>`（4 个 AddDouble） |
| `command` | ReceiveCommand 0x7be80 | args[2] vs MonitorPassword(this+0x2c) | 见 §8 |
| `start_app` | ReceiveStartApp 0x7c540 | args[2] vs MonitorPassword | 见 §9 |
| `stop_app` | ReceiveStopApp 0x7c900 | args[2] vs MonitorPassword | 见 §9 |

通用错误串（SendResultMsg(cnx,"<cmd>","<err>")，0x6f460 → varlist=`["result",cmd,err]`）：
`error_password` / `error_request` / `error_parameter` / `error_time` / `no_database`。
成功响应普遍以 `result ...` / 命令名开头（ReceiveServers 直接 `["servers", name, path, ...]`）。

`sys_info` 采样（Port_GetMemoryInfo 0x7bd30）：`GlobalMemoryStatus` 后
out[0]=dwMemoryLoad/100.0（常量 0x100f92f0=100.0）、out[8]=dwTotalPhys、out[0x10]=dwAvailPhys；
CPU 由 CCpuRate（PDH `\Processor(_total)\% Processor Time`，0x69910 初始化三连 PdhOpenQuery/PdhAddCounter/PdhCollectQueryData）。

## 6. query 子命令（ReceiveQuery 内 stricmp 链，0x77c80 起）

| 子命令 | CAccess msgid | Publish*（响应） |
|---|---|---|
| `request_roles` | 0 | PublishRoles（`role`，len<0x34 校验） |
| `request_role_data` | 1 | PublishRoleData |
| `request_pub_data` | 2 | PublishPubData |
| `request_chunk_data` | 3 | PublishChunkData |
| `request_log_data` | 4 | PublishLogData（`log`；有 `error_time` 时间参数校验） |
| `request_item_log` | 5 | PublishItemLog |
| `request_chat_log` | 6 | PublishChatLog |
| `request_gm_log` | 7 | PublishGmLog |
| `request_names` | 10 | PublishNames |
| `request_sql_select` | 11 | PublishSqlSelect |

链路：ReceiveQuery 校验密码 → 44 字节（0x2c）请求结构（cnx id + msgid + SQL 参数）→ `CAccess::PutMessage` → Access 线程 `ProcessMsg`（跳转表 0x10061294）→ `RequestXxx`（RequestRoles 模板 `sprintf(sql, fmt, req+8)` 经 this+0x10 IDBSystem vtable+0x48 取结果集）→ `SendResponse(msgid,cnx,data,len)` → GameWatch DB 队列（this+0x130）→ DoWait `ProcessDbMsg`（跳转表 0x1007492c）→ `PublishXxx` → SendCustomMsg。

CAccess 线程（ctor 0x5fc00）：CoreCreateThread(loop=AccessLoopProc, init=AccessInitProc, shut=AccessShutProc)；连接自愈 InitConnect/CheckConnect/ConnectFailed/FailCanRetry。

## 7. update 子命令（ReceiveUpdate，UpdatePassword）

| 子命令 | 行为 | 错误串 |
|---|---|---|
| `get_dir <pwd> <server> <path>` | CUpdate::ReadDirInfo 递归扫目录树 → StoreArchive 序列化 → SendDirData 64KB 分块 | `no dest path`/`no_loop`/`refresh dir info failed`/`send dir data error` |
| `get_file <pwd> <server> <file>` | fopen rb → fseek/ftell → SendCustomFile 64KB 分块 | `file data size error`/`send file error` |
| `set_dir <pwd> <server> <path>` | 建目录（GetDestPath 映射 ExePath/ResPath） | `root path error`/`dir already exists`/`create dir failed` → `succeed` |
| `set_file <pwd> <server> <file> <size> <time> <md5> <*data>` | hex 解码 → 缓冲拼装 → **MD5 校验** → `wb` 落盘 → 恢复 mtime | `file data size error`/`file cache error`/`md5 verify code not match`/`create file failed`/`write file data failed`/`set file time failed` |

分块下发格式（SendCustomFile 0x6f630 / SendDirData 0x6f9d0，两函数同构）：

```
[cmd头串×N…] "succeed" <total_size:int> <offset:int> <chunk_len:int> <chunk:userdata>
```

- chunk 上限 **0x10000（64KB）**；userdata → `*` + 大写 HEX（text_encode）。
- dir_data_t（0x88 字节）：name(+0)/path(+0x30)/dirs(+0x60 TArrayPod<dir_data_t>)/files(+0x74 TArrayPod<file_data_t>)；ReadDirInfo 以 `\` 连接父子、跳 `.`/`..`、CoreNewDebug 分配。
- DiffDirInfo/ParseDiffDirInfo：远端 dir_data 与本地对比（FileEqual：名字+size+mtime）→ CUpdateDirs（缺目录）+ CUpdateFiles（file_info_t：名/size/time）。

## 8. command：运维命令直投

`command <pwd> <addr:port> <命令行>`（ReceiveCommand 0x7be80）：
1. stricmp(args[2], MonitorPassword) → 失败 `result command error_password`；
2. args[1] `addr:port` → strchr(':') 分离、atoi 端口；
3. `GetCore():Create("UserSock")` → SetAddress/SetPort/Connect；错误 `error_no_app`/`error_address`/`error_create_sock`；
4. 发 `args[3] + "\r\n"`（malloc len+2）→ 目标服 echo 维护口（echo.lua 受面：quit/end/getp/setp/getc/setc/method/listp/listc/listm/listg/function/debugger）。

## 9. start_app / stop_app（MonitorPassword）

- `start_app <pwd> <server>`：GetServerPath 查 [Servers] → `no_exe_path`（[ExePath] 无键）→ 已运行 `in_running` → 拉起 → `succeed`/`failed`；日志 `Start server program`。
- `stop_app <pwd> <server>`：找进程 → `not_found` → 终止 → `succeed`/`failed`；日志 `Stop server program`。

## 10. 心跳与主循环（DoWait 0x6fde0）

每轮：`CThreadWaiter.Wait(10ms)` → DB 队列(this+0x130) ProcessDbMsg → 客户端队列(this+0x134) ProcessSockMsg → `CTickTimer(1000ms)`：
- 计数>5（即 ≥5 秒）→ **向所有运维客户端广播 `"\r\n"` KeepAlive**（IMassSock vtable+0x9c）；
- EchoSock 存在且计数>5 → **`<info state="%s" connect="%d" />`**（state=GetState 四态 `Unknown/Opened/Paused/Closed`（this+0x28: 0/1/2/3，跳转表 0x1006d400），connect=MassSock 连接数）经 EchoSock SendLine —— 供中央 AppManager 巡活。

## 11. 配置

**GameWatch.ini**：`[Main] MaxClient/MonitorPassword/QueryPassword/UpdatePassword/LockSystem`（三密码 32 hex=MD5 形态，**明文 stricmp 比对**）、`[Outer] Addr/Port`、`[Echo] Addr/Port`（缺省 127.0.0.1:15006）、`[Servers] key=命令行`、`[ExePath]/[ResPath] key=根目录`、`[Address] AddressN=IP`（白名单）。

**access.ini**（CQueryDb::LoadFromFile 0x7fa90，CIniFile::GetSectionList 逐节）：

```ini
[节名]
app_name = <数据源名>
type = <类型>
connect_string = <连接串>
```

三键 stricmp 匹配（app_name=0x100f9764/type=0x100f975c/connect_string=0x100f9748）→ InnerAddAccess；失败日志 `(CQueryDb::LoadFromFrom)add access failed`。快照 Debug 目录未携带 access.ini。

## 12. Go 复刻

`internal/fswatch`（对齐官方外显行为；AppManager/MassSock/EchoSock/UserSock 由本包内嵌等价实现，不依赖 FsKit/FsComm）：

| 文件 | 对应官方 |
|---|---|
| `varlist.go` | text_encode_msg/text_decode_msg/text_append_word（`$`/`#`/`*`/`\xXX` 大写） |
| `config.go` | GameWatch.ini + access.ini（app_name/type/connect_string） |
| `watch.go` | GameWatch 实体：8 命令分发/三密码/SendResultMsg/servers/command |
| `sysinfo.go` | sys_info（gopsutil 等价采样：cpu/memload/total/avail） |
| `update.go` | CUpdate：目录树/序列化/get_dir/get_file/set_dir/set_file/MD5/64KB 块 |
| `query.go` | CQueryDb/CAccess：10 子命令转发 + QueryBackend（MySQL 可插拔） |
| `proc.go` | start_app/stop_app/app_list（进程表守护） |
| `server.go` | MassSock 文本行受面 + EchoSock 心跳 `<info .../>` + 5 秒 `\r\n` 广播 |
| `fswatch_test.go` | 协议回归 |

`cmd/watch`：`GameWatch.ini` 方言宿主（默认释放内置配置，已存在不覆盖）。

差异声明：①CPU 采样以 Go 运行时等价（非 PDH 原文路径）；②query 的 MySQL 后端可插拔，无库时按子命令回 `no_database`（官方行为）；③set_file 落盘在类 Unix 下 mtime 恢复为 utime 等价。
