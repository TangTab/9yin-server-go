// Package fssoloentry 复刻官方 FsSoloEntry.dll（BGserverCJ\Debug，
// 2013-07-05 构建，全符号 .map/.pdb）：单机（solo）形态的接入网关，
// 由 game_entry.exe（通用 solo 壳，读 game_entry.ini 装载
// fssoloentry.dll + lua\soloentry\main.lua）承载。
//
// 权威事实源：FsSoloEntry.map（4897 符号直读）+
// tmp_research/soloentry/ 反汇编终局（dis_start/dis_processinner/
// dis_timercheck/dis_lookupthread/oh_*/ih_*，capstone 注释版）+
// Lua/soloentry/main.lua + game_entry.ini 实物；网关版孪生
// FsEntry.dll 已复刻于 internal/entry（docs/official/FsEntry全解与
// Go复刻.md），本包复用其 Clients/Gateway/Inner 并叠加 solo 差异面。
//
// # 官方架构（反汇编定案）
//
//	SoloEntry 实体（soloentry.cpp，源码路径
//	  c:\wuyf\svn\serveriii\branches\publish20130508\server\fssoloentry\）
//	 ├ CReactor          → 复用（fscomm/net 组件自管）
//	 ├ CInnerHook        → entry.Inner + solo 旋钮（innerhook.cpp）
//	 ├ COuterHook        → entry.Gateway（outerhook.cpp）
//	 ├ CEchoHook         → fscomm.EchoSock（Lua echo_* 反射面）
//	 ├ CBlockIP          → entry.BlockIPList（ip.txt 黑名单）
//	 ├ CMessageLog       → 日志等价（LogSend/LogReceive/LogConnect）
//	 ├ CAppRecord/CLogShareMem/CShareMem → 运维件不复刻（日志等价）
//	 └ 状态机/timer/license/广播 → 本包 Server
//
// # 与网关版 FsEntry 的差异（全部字节级实证）
//
//   - 消息编号：ProcessInnerMsg 15 型（0..0xE），无落格/视野/批量移动
//     三型（solo 无空间网格）；4=MultiSend 批组播、5=SceneSend 场景播、
//     7=OnlineSend 仅在线播、8=BroadcastSend 全量播、14=close
//   - set_client_online 第 4 域是 scene（ClientOnline 写 client+0x18，
//     SceneSend 过滤依据），非网关版的 memberIdent
//   - 源校验放宽：{0,1,2} ∪ [0x18,0x2F]（24 个 member 位）
//   - 自身 ident = EntryId+8（SetCommIdent）；ident 重复保旧踢新
//     （回 0xFFFE 帧）；入站 0xFFFE → Core 退出
//   - InnerPort 缺省 0：监听器动态分配后回写字段（Lua 打印实际端口）
//   - license：密钥派生 md5(盐16 + IP串16)（网关版为 md5(IP+盐+0xCC×16)）；
//     双形态——内部 256B（本机 IP 须为 127.0.0.1 或 192.168.*）/外部
//     272B（尾部 16B 二段密钥 + 时间窗 [t0,t1] 校验，now 越界即拒）；
//     解密 3DES-EDE-ECB(K1||K2) 16×16B；校验和 = 前 252 字节求和 ==
//     u32@0xFC；身份域 8×u32@0x00；TimerCheck 复验魔数对
//     0xE7990369/0x87930363（失败仅 WriteLine 告警，非致命；
//     魔数对命中才发 BroadcastEntryInfo）
//   - init_connect 携带世界窗值：越出 license 时间窗 [f1d4,f1d8] →
//     身份域清零（撤销授权——防拆联动）
//   - 广播包 0x48B：{1, 0x3001, inner_addr[16], inner_port, district,
//     server, entry, idc, outer_addr[16], max_connects, connect_count}
//   - 状态行每 6 个 timer tick（1s 定时器）上报一次
//
// # 与官方的差异（原则：线上字节一致，进程内设施取 Go 等价）
//
//   - quicklz 大包压缩 + op 重映射（InnerSendPrepare）与网关版同为
//     已知未复刻项（本线客户端不启用）；扇出按原文透传
//   - Lua 反射维护面（getp/setp/method/list*/function/debugger）以
//     内置命令集等价（state/stat/loadip/help/quit/end）
//   - 命名互斥单实例（mutex_%s）与跨进程共享内存运维件不复刻
//   - UseThread 工作线程、CReactor 定步泵由 Go 运行时等价
package fssoloentry
