// Package fssolopublic 复刻官方 FsSoloPublic.dll + game_public.exe
// （小写系 solo 公共数据服务器进程，BGserverCJ\Debug，2013-07 版源
// c:\wuyf\svn\serveriii\branches\publish20130508，全符号 .map/.pdb）。
//
// 权威事实源：FsSoloPublic.map（5,502 符号最终 VA 直读）+
// tmp_research/solopublic/ 反汇编终局（dis_state_*/dis_store_*/
// dis_world_*/dis_ctor/dis_start 等 40 篇，capstone 注释版）+
// 官方装配脚本 BGserverCJ\lua\solopublic\main.lua（全文）+
// docs/official/FsSoloPublic全解与Go复刻.md（本包伴生文档）。
//
// # 官方架构（反汇编定案）
//
//	game_public.exe（通用 solo 壳，按自身文件名读 game_public.ini，
//	                  MainEntity=SoloPublic）
//	 └ FsSoloPublic.dll（本包）
//	    ├ CReactor      → internal/reactor（直接复用）
//	    ├ CInnerHook    → InnerLink（inner TCP，Port 缺省 0=系统分配回填）
//	    ├ CEchoHook     → EchoConsole（echo TCP :15003，欢迎语
//	                       "welcome public server <S>\r"）
//	    ├ UDP :15000    → magic 0x3001 方言：自宣 kind=4（0x24B）；收听
//	                       type2 store / type1 entry（外连幂等补连）/
//	                       type0 world（RoleProtectBase）
//	    ├ 状态机 8 态   → Unknown→LoadRelations→LoadFeeds→LoadTrades→
//	                       LoadSpaces→Opened→Closing→Closed（this+0x34）
//	    ├ CPubSpace×    → 复用 internal/fspublic（PUB4 归档/属性/记录）
//	    ├ CNameUid/CPubPlayers/CPubScenes → world 注册表镜像
//	    ├ CRelation     → 好友关系 + SNS 动态流（feed）
//	    └ CTrade        → 寄售行（Sell/Buy/Search/超时）
//
// # comm 身份命名空间（innerhook.h 断言定案）
//
//	0=World（连入）  1=本服（Start SetCommIdent(1)）
//	4..7=Store（外连）  8..0x17=Entry（外连，solo 特有）
//	0x18..0x2F=Member（连入，MAX_MEMBER_NUM=24）
//
// # 内网帧（CInnerHook::SendMsg/ReceiveMsg，solo 全家同构）
//
//	[u32 N][u32 comm_id][u32 id1][u32 id2][payload]
//	N = 12 + len(payload) ≤ 0x3FFFFC；id2=0xFFFF identify / 0xFFFE 退出。
//
// store 面 type 空间为集群 FsPublic +6：装载 0x24/0x25、保存 0x26/0x27
// （应答 0x19..0x1D）；solo 特有 0x2D 关系请求（→0x22 应答）、0x30 feed
// 存活期配置、0x3A 动态请求（→0x23 应答）、寄售 0x28 应答。world 面 13
// 型（scene/player/role 注册表维护）。member 面 3 型 + 推送 op 0..0xD
// （与集群同构，门 [state]==5）。
//
// # 与官方的差异（原则：线上字节一致，进程内设施取 Go 等价）
//
//   - LogicLoader(fspublogic.dll) 为官方外部 Callee 模块（版本门 0x300F）；
//     Go 版以 Callee 接口注入（复用 internal/fspublic.Callee），默认
//     DefaultCallee（1000..1107 内建修改原语 + 域注册表骨架）。
//   - main.lua 装配序由 Server.Start 吸收（AddPubSpace("domain","nx_domains")
//   - ("guild","nx_guilds") 缺省注册）。
//   - echo 口官方为 Lua 十二调试回调；Go 版为内置命令集
//     （state/spaces/datas/players/scenes/names/relations/trades/help）。
package fssolopublic
