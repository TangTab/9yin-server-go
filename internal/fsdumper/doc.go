// Package fsdumper 复刻官方数据转储服务器 FsDumper.dll（服务壳）+
// FsDumpLogic.dll（业务逻辑）。
//
// 逆向事实源：F:\Game\Online\9yin\BGserverCJ\Debug 全符号快照
// （FsDumper.map/.pdb + FsDumpLogic.map/.pdb，2014-03-22），启动脚本
// GameDumper.bat/GameDumper.ini，Lua 主控 Lua/GameDumper/{main,echo}.lua，
// 转储配置 Res/dumpconfig/。权威文档：
// docs/official/FsDumper与FsDumpLogic全解与Go复刻.md。
//
// # 官方架构位置
//
//	GameDumper.bat → FsLoader.exe GameDumper.ini → FsDumper.dll（Fs 模块壳）
//	  ├─ EchoSock :15009（默认）运维口（fscomm 复刻件）
//	  ├─ Outer    :4006（[Outer] Port）游戏服接入口 —— 行协议
//	  ├─ CAccess×4（role/public/manage/output）DB 代理线程
//	  ├─ CDumpKernel（IDumpKernel 17 槽虚表）→ LogicLoader 加载
//	  │   FsDumpLogic.dll → dump_callee（DumpRole + DumpRes 两单例）
//	  └─ GameDumperState：Unknown(0)→Opened(1)→Closing(2)→Closed(3)
//
// # :4006 线路方言（0x100756F0 ReceiveClientMsg 实证）
//
//	行协议：'\r'/'\n' 切行，行缓冲 4096 字节超限丢弃；
//	每行 = 一条 text 方言 VarList（与 svrlist 同一 text_encode_msg）：
//	  数字（首字符 [0-9.-]；无 '.' 且 <10 字 → int，否则 int64/double）
//	  '#'宽串（'\\' → '\'，'\u'HHHH，'\x'HH）
//	  '$'窄串（裸 ASCII）
//
// # dump_role_msg 协议（DumpRole 模块）
//
//	Dumper→游戏服：["dump_role_msg", 1, version, table, (kind,col,prop)*]
//	游戏服→Dumper：["dump_role_msg", 1, ver]（版本报，不符回发模板）
//	                  ["dump_role_msg", 2, fields..., 值...]（角色快照）
//
// # DumpRes 流程
//
//	TimerStartDumpConfig(1s) → DELETE FROM nx_config WHERE op_type=0 →
//	静态 SQL（game_enum/ratio_action/dump_res）→ DumpLogDefine（解析
//	DumpHFile/LogDefine.h → log_enumtype/log_enumvalue）→ <move> 清单
//	按 mtime 变更检测增量写 nx_res_dumpinfo → 每 5 分钟巡检。
//
// # 与官方的差异（复刻决策）
//
//   - CAccess→Store 内部协议不复刻，DB 面 = database/sql 直连 MySQL；
//   - echo.lua 的 getp/setp/listp 等控制台命令依赖 Fs 实体系统，暂缺；
//   - 记录序列化（capital_rec）取「行间 ';'、列间 ','」方言（官方未证）。
package fsdumper
