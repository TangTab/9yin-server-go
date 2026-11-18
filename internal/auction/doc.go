// Package auction 复刻官方 FsAuctionLogic.dll 的拍卖行数据服务器业务核心。
//
// 逆向事实源：F:\Game\Online\9yin\BGserverCJ\Debug\FsAuctionLogic.dll
// （全符号 .map/.pdb，2014-03-22 快照），权威文档见
// docs/official/fsauctionlogic-拍卖行数据服务器.md。
//
// # 官方架构位置
//
//	game_auction.exe → FsSoloAuction.dll（框架宿主，inner :16003/echo :16001/
//	broadcast :15000，Lua 主控 main.lua）→ LogicLoader 加载 fsauctionlogic.dll
//	→ auction_callee（IAuctionCallee 八槽接口）。
//
// 本包 = FsAuctionLogic 的业务等价翻译（记录层+消息层），不含框架宿主
// （socket/持久化由 solo 载体与 fscomm/persist 承担）。
//
// # 数据模型
//
//   - MainRecord：全量拍卖行表，每行 18 列（Row），空行复用链
//     （官方 IsRecFull 上限 = 构造器常量 100000/0x186a0）
//   - SortedRecord：按 大类(1..4=Weapon/Armor/Consume/Other) ×
//     排序键(1..6=Quality/RestTime/Level/CurPrice/FixedPrice/SellerName)
//     的 24 个排序视图，行号有序数组；单大类上限 25000/0x61A8
//   - ItemType→BigType 映射：ini/trade/AuctionHouse/ItemType.xml
//     （LoadResource 装载，缺失仅告警）
//
// # 消息方言（内网 VarList 帧，元素 [0]=int 消息号）
//
//	game→auction：1 寄售 / 2 取消寄售 / 3 竞拍出价 / 4 一口价 /
//	               5 重建排序视图(GM) / 6 清空数据(GM)
//	auction→game：7 寄售回执 / 8 取消回执 / 9 竞拍回执 /
//	               10 一口价成交（兼流拍退回，按元素数区分）/
//	               11 竞拍到期成交（超时定时器 HCheckDueTime 广播 target=0）
//
// 校验失败静默丢弃（官方 Check*Args 只返回码不回帧——前端校验在
// FsGame AuctionHouseModule 完成，拍卖服是可信后端）。
//
// 配置：官方 game_auction.ini 方言（auctionconfig.go），内置默认
//（game_auction.ini，//go:embed 释放式，与 game_entry/game_lister 同范式）；
// 接线 DistrictID/ServerID/Inner.Addr+Port（缺省回落 main.lua 16003）/
// Resource.Path（Go 扩展，ItemType.xml 装载根）。
package auction
