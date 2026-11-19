// Package fssoloauction 复刻官方 FsSoloAuction.dll + game_auction.exe
// （小写系 solo 拍卖行数据服务进程，BGserverCJ\Debug，2013-07-05 构建，
// 全符号 .map/.pdb）的框架宿主面。
//
// 权威事实源：FsSoloAuction.map（5244 符号直读）+
// tmp_research/auction/ 反汇编终局（dis_start/dis_membermsg/dis_state_*/
// dis_timercheck/dis_broadcast 等，capstone 注释版）+
// docs/official/FsSoloAuction全解与Go复刻.md（本包伴生文档）。
// 业务核心（fsauctionlogic.dll）不在本包——已复刻于 internal/auction
// （Logic.Dispatch → []Outbound），本包是承载它的宿主进程。
//
// # 官方架构（反汇编定案）
//
//	game_auction.exe（通用 solo 壳，与 game_lister/game_entry 同一二进制，
//	                  按自身文件名读 game_auction.ini）
//	 └ FsSoloAuction.dll（本包）
//	    ├ CReactor      → internal/reactor（已有复刻，直接复用）
//	    ├ CInnerHook    → InnerLink（inner TCP :16003）
//	    ├ CEchoHook     → EchoConsole（echo TCP :16001）
//	    ├ UDP :15000    → 服务发现（world/store 自宣，本服 announce type=5/
//	                       magic=0x3001，每 TimerCheck 1s 发一次）
//	    ├ 状态机        → StateMachine（Unknown→LoadAuctionHall→Opened→
//	                       Closing→Closed，官方 this+0x30 五态索引）
//	    ├ CAuctionScenes/CAuctionPlayers/CNameUid → world 注册表
//	    │  （World 消息 id2 0..0xC 维护）
//	    ├ PUB4 归档     → store 0x41/0x42 存、0x29/0x2A/0x2B 加载
//	    └ LogicLoader("fsauctionlogic.dll") → Go 等价：进程内装配
//	       internal/auction（callee 的 OnAuctionDataLoaded/OnAuctionMessage
//	       → Manager 装配 / Logic.Dispatch）
//
// # comm 身份命名空间（COMM_ID_MAX=64，官方 innerhook.cpp 断言）
//
//	0=World（本服外连）  2=本服（Start@100AAABD SetCommIdent(2)）
//	4..7=Store（本服外连，广播发现）  0x18..0x2F=Member（游戏服，连入）
//
// # 内网帧（CInnerHook::SendMsg/ReceiveMsg @0x1006DE80/0x1006D630）
//
//	[u32 N][u32 comm_id][u32 id1][u32 id2][payload]
//	N = 12 + len(payload) ≤ 0x3FFFFC；id2=0xFFFF identify 握手 /
//	0xFFFE 退出；未知来源 comm → trace "message source error"。
//
// member 业务消息：入站 id2=2、payload = auction_forw_message_t
// {u32 param2; u32 vary_count; vary[]}（vary=无数量前缀元素流，
// proto.EncodeElements）；出站 SendToMember(target,0,0xD,forw)，
// target∈[0,0x18) 是 member 索引（0x18+index=comm）。
//
// # 与官方的差异（原则：线上字节一致，进程内设施取 Go 等价）
//
//   - echo 口（16001）官方为 Lua 表达式求值（on_input）；Go 版为内置
//     命令集（state/stat/items/save/rebuild/clear/roles/scenes/players/help）
//   - CAppRecord/CLogShareMem/CMemPool（跨进程共享内存运维件）不复刻，
//     以日志呈现等价信息；命名互斥单实例不复刻
//   - LocalStore：官方持久化完全依赖外部 Store 进程（comm 4..7）；
//     Go 扩展内置本地文件 store（PUB4 同格式落盘 data/auction/），
//     无外部 store 时兜底完成 0x40→0x29/0x2A/0x2B 加载序（可用性优先，
//     接真 store 时走官方协议原样互通）
//   - World 消息中 SceneAdd(0x90)/PlayerAdd(0x9C)/RoleInfo(archive 流)
//     布局为反汇编实证；ServerState/PlayerChange 等次要消息布局按对称
//     推断（宽容解析，失败仅 trace）
package fssoloauction
