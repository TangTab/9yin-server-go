// Package fssologlobal 复刻官方 FsSoloGlobal.dll + game_global.exe
// （小写系 solo 全局名字服务器，BGserverCJ\Debug，2014-03-22 快照，
// FsSoloGlobal.pdb 1316 公共符号）。
//
// 权威事实源：FsSoloGlobal.pdb 符号直读 + 反汇编终局
// （tmp_research/sologlobal/ 注释版 dis_*.txt）+ 官方 game_global.ini +
// Lua\sologlobal\main.lua 原文；全解沉淀
// docs/official/FsSoloGlobal全解与Go复刻.md（本包伴生文档）。
//
// # 官方架构（反汇编定案）
//
//	game_global.exe（通用 solo 壳，按自身文件名读 game_global.ini，
//	  [Module] 装载 fskit/fssql/fscomm/fssologlobal.dll，
//	  MainEntity=SoloGlobal + MainScript=main）
//	 └ FsSoloGlobal.dll ＝ SoloGlobal 实体（IEntity）
//	    ├ CReactor        → internal/reactor
//	    ├ 外网受面 TCP    → Server（[Outer]，缺省 127.0.0.1:4003）
//	    │   行协议：\r/\n 终止、行缓冲 0x3FF、空行不分发；
//	    │   命令 = text_encode_msg 编码的 IVarList：
//	    │   [回显串, "add_name", 域, 名(宽), 主(宽), server_id] /
//	    │   [回显串, "remove_name", 域, 名(宽)]；
//	    │   应答同构 [回显串, 命令, 表名, 名(宽), 结果int] + CRLF
//	    │   （编解码 = internal/charge 的 VarList 文本方言，同源
//	    │   text_encode_msg/text_decode_msg）
//	    ├ CEchoHook        → EchoConsole（[Echo]，缺省 127.0.0.1:15012，
//	    │   "welcome global server\r" + quit/end，官方 Lua on_input 族）
//	    ├ CAppRecord      → 状态行日志（每 6s
//	    │   `<info state=".." database="OK|Break" db_err=".." tick=".." />`）
//	    ├ 状态机          → StateMachine（Unknown→Opened；Closed 仅枚举）
//	    ├ 域表 domains[8] → DomainTable（name[0x24]/table[0x24] 定长，
//	    │   main.lua 实配 role→nx_role_names / guild→nx_guild_names）
//	    └ CAccess → DBSystem 实体（nx_create("MysqlDBSystem")）
//	       → SQL 三句：SELECT name FROM %s WHERE name='%s' /
//	          INSERT INTO %s ( name, owner, server_id, create_time )
//	          VALUES ( '%s', '%s', %d, NOW() ) /
//	          DELETE FROM %s WHERE name = '%s'
//	       语义：AddName 名不存在才 INSERT；RemoveName 名存在才 DELETE；
//	       连接失败计数重试（FailCanRetry < 2 次）后放弃
//
// # 与官方的差异（原则：线上字节一致，进程内设施取 Go 等价）
//
//   - CAppRecord/CLogShareMem/CMutex（跨进程共享内存运维件、单实例
//     互斥、Port_Execute 拉起 game_log）不复刻，以日志呈现等价信息
//   - DBSystem 实体 + CAccess vtbl 面 → DBAccess 接口：SQLStore
//     （database/sql MySQL，官方连接串为 SQLOLEDB 方言时须改用
//     -mysql-dsn）或 LocalStore（内存 + JSON 快照，无库时兜底）
//   - echo 口官方为 Lua 脚本层（on_input 命令族含 getp/setp/method
//     等实体自省）；Go 版为内置命令集（quit/end/help/state），
//     欢迎语与 quit/end 语义与官方 main.lua 一致
//   - WideStrVal 对窄串元素宽容取值（官方按槽位类型严格返回；
//     客户端实际总是发 `#` 宽串，此为超集兼容）
package fssologlobal
