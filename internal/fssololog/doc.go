// Package fssololog 复刻官方 FsSoloLog.dll（game_log.exe 角色）：
// solo 集群的日志汇聚服务器。
//
// 官方形态（docs/official/FsSoloLog全解与Go复刻.md）：
//
//	game_log.exe（通用 solo 宿主壳）→ game_log.ini
//	  MainEntity=SoloLog + fskit.dll/fscomm.dll/fssololog.dll
//
// FsSoloLog.dll 导出 8 个标准 Fx 模块契约（"FxModule"/0x3EB/空 Space），
// 实体 SoloLog（0x40B8）承载三件事：
//
//   - CLogShareMem：Windows 命名共享内存 memory_fs_solo_log_0
//     （64 appid 槽 × 0x808C，128×256B 环形行仓）——各业务进程
//     （world=1/public=2/auction=3/monitor=0x32/global=0x38…）把
//     CORE_TRACE 日志行投递到各自槽位；
//   - CheckLog 消费循环（0.1s 定时器）：逐槽取行（每 tick 每槽至多
//     100 行）落盘 `目录 appid_前缀_时间戳.log`，EveryDayAlone 跨天
//     换档、create_new 生产者主动换档；行时间戳取落盘时刻；
//   - AutoExit 看门狗：槽 1（FsSoloWorld）注册过又消失 5 秒 → 整进程
//     退出（ctor +0x2D 定案缺省开）；echo 维护口（CEchoHook 单连接
//     TCP 控制台，缺省 127.0.0.1:15011）。
//
// Go 落点（§7 设计）：本包 sololog.go（Server 实体装配）+
// logsharemem.go（槽协议）+ echo.go（单连接控制台）+ parse.go
// （parse_string/parse_token）+ commands.go（main.lua 内置命令集）+
// config.go（game_log.ini 方言）。进程载体 cmd/logserver。
//
// 已知取舍（官方 vs Go）：
//   - 共享内存：Windows 上按官方同名 CreateFileMapping 真映射
//     （可与官方 game_*.exe 生产进程互操作）；非 Windows 回退进程内
//     缓冲（布局/游标协议逐字对齐，仅无跨进程能力）。
//   - UseThread 双 EventLoop、CLeakChecker、SetConsoleColor 不复刻
//     （Go 运行时等价/非控制台 no-op）。
//   - getp/setc 族反射命令映射到本实体属性/方法表（与 soloentry 同
//     "等价反射面"原则）。
package fssololog
