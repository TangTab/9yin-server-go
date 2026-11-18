# FsAuctionLogic.dll 全解与 Go 复刻（拍卖行数据服务器）

> 对象：`F:\Game\Online\9yin\BGserverCJ\Debug\FsAuctionLogic.dll`（1015808 B，2014-03-22，
> Debug 全符号 .map/.pdb，源码路径 `d:\bg_source\server\fsauctionlogic\`）
> 日期：2026-11-19　分支：`official`　复刻落点：`internal/auction`
> 伴生文档：`tmp_research/auction/soloauction_framework.md`（FsSoloAuction 框架宿主逆向，
> 同目录 `fsgame_relay.md` 为 FsGame 转发链）

---

## 一、官方架构位置

```
game_auction.exe（FsSoloLoader 壳）
 └ FsSoloAuction.dll        框架宿主（Lua 主控 main.lua + inner/echo/broadcast 三通道）
    ├ inner  TCP 16003      game 服(FsMember, comm 0x18..0x2F)/world(0)/store(4..7) 业务内网
    ├ echo   TCP 16001      调试口（逐行 Lua 表达式）
    ├ broadcast UDP 15000   服务发现（world/store 自宣，auction 主动外连）
    └ LogicLoader.Load("fsauctionlogic.dll")
       └ FsAuctionLogic.dll 业务逻辑（本 DLL）：auction_callee + AuctionLogic + AuctionManager
```

DLL 导出 5 个：`FxModule_GetType/Init`（模块契约）、`FxAuctionLogic_GetVersion`、
`FxAuctionLogic_CreateCallee/DeleteCallee`（callee 工厂）。框架取版本校验后 CreateCallee，
之后按事件驱动回调。

## 二、auction_callee（IAuctionCallee，vtable 8 槽 @0x1012F260）

| 槽 | 方法 | 反汇编结论 |
|---|---|---|
| +0x00 | deleting dtor | |
| +0x04 | OnAuctionStart | **no-op**（return 0） |
| +0x08 | OnAuctionDataLoaded | → AuctionManager::OnAuctionDataLoaded（见 §五） |
| +0x0C | OnAuctionMessage(kernel,int1,int2,VarList) | → AuctionLogic::DispatchMessage（§三） |
| +0x10 | OnFrameExecute(kernel,int) | 心跳 no-op（call kernel vt+0x30 后 return 0） |
| +0x14 | OnServerState(kernel,str,5×int) | str=="opened" → 置开启位 |
| +0x18 | OnRoleOffline(kernel,int,VarList) | **no-op**（return 1） |
| +0x1C | OnServerClose | no-op |

## 三、消息方言（DispatchMessage @0x1008FEB0）

读 `VarList[0]`（IntVal）为消息号，负载 = `Append(源, 1, count-1)`（即去掉首元素），
6 槽跳转表 @0x100900A8：

| 消息 | 方向 | handler | 负载（去掉消息号后） |
|---|---|---|---|
| **1 寄售** | game→auction | MsgHandlerConsign | 19 元素（reqid + 18 列行，见 §四） |
| **2 取消寄售** | game→auction | MsgHandlerCancelConsign | 1：`[uid str]` |
| **3 竞拍出价** | game→auction | MsgHandlerBid | 3：`[uid str][bidder str][price i64]` |
| **4 一口价** | game→auction | MsgHandlerFixedBid | 2：`[uid str][buyer str]` |
| **5 重建视图** | game→auction | MsgHandlerRebuildRecs | 0（GM） |
| **6 清空数据** | game→auction | MsgHandlerClearData | 0（GM） |
| **7 寄售回执** | auction→game | ConsignBack | `[7][consign_id i64][itemstring str]` |
| **8 取消回执** | auction→game | CancelConsignBack | `[8][uid][configid][itemstring][selleruid]` |
| **9 竞拍回执** | auction→game | BidBack | `[9][uid][configid][bidder][old_bider][old_price i64]`（旧值取自 DoBid 前） |
| **10 一口价成交回执** | auction→game | FixedBidBack | `[10][uid][configid][itemstring][old_bider][old_price i64][buyer][fixedprice i64][selleruid][feecoe f32]`（10 元素） |
| **11 竞拍到期成交** | auction→game | TimeOutSoldBack | `[11][uid][configid][selleruid][itemstring][curprice i64][feecoe f32][bideruid]`（8 元素） |
| **12 流拍退回** | auction→game | TimeOutUnSoldBack | `[12][uid][configid][selleruid][itemstring]`（5 元素） |

> 超时分支选择值 = `10+(State==2)`（仅用于 ==0xB 判断）：成交分支实发 11、
> 流拍分支实发 12（push 0xC @0x100924E4 实证；与 FsGame OnPublicMsg 入站表
> 7=ConsignBack/8=CancelConsignBack/9=BidBack/10=FixedBidBack/11=TimeOutSoldBack/
> 12=TimeOutUnSoldBack 完全互证）。

- 回执经 kernel vt+0x20 = `SendAuctionMessage(target, sub, VarList)`——target/sub 即
  OnAuctionMessage 的 int1/int2 原样回传（Member 线上封装 `{u32 param2; u32 vary_count; vary[]}`）。
- **超时结算广播 target=0**（ProcessOverTimeItem @0x100925A7 push 0,0）。
- **校验失败静默**：Check*Args 非 0 返回时 MsgHandler 直接 return，不回错误帧
  （前端校验在 FsGame AuctionHouseModule 完成，拍卖服是可信后端）。

### CheckResult 错误码（反汇编实参）

| 码 | 含义 | 出处 |
|---|---|---|
| 1 | 参数个数/类型不符 | 各 Check 首段类型闸 |
| 9 | 空字符串（uid/出价人/买家/卖家名 strlen==0） | CheckConsign(名)/Bid/FixedBid/Cancel |
| 5 | FindItem 无此物品 | CheckBid/FixedBid/Cancel 尾段 |
| 11 (0xB) | 唯一号重复 | CheckConsign |
| 22 (0x16) | 全表 ≥ 100000（0x186A0） | CheckConsign |
| 23 (0x17) | 单大类 ≥ 25000（0x61A8） | CheckConsign |

## 四、MainRecord 行结构（18 列）

列号 = 各 Get*/Set* 实参列下标（反汇编）；寄售负载 = `[consign_id i64] + 18 列原样`
（consign_id = FsGame gen_item_unique_id() 生成，不占行列表列；行 = 负载[1..18]，建表列类型序列 [6,6,2,6,6,3,3,4,3,3,6,3,2,2,7,2,2,6] 构造器 RecConfig 实证）：

| col | 类型 | 字段 | getter 实证 |
|---|---|---|---|
| 0 | string | UniqueID（FindItem 键） | GetUniqueID push 0 |
| 1 | string | ConfigID | push 1 |
| 2 | int | （透传，用途未名） | 类型闸 |
| 3 | string | ItemString（物品序列化串） | push 3 |
| 4 | string | SellerUid | push 4 |
| 5 | i64 | StartPrice 起拍价 | FsGame 寄售帧字段实证 |
| 6 | i64 | FixedPrice | push 6 |
| 7 | f32 | TransFeeCoe | push 7 |
| 8 | i64 | KeepingFee 保管费（GetKeepingFee 计算后随帧透传） | FsGame 寄售帧字段实证 |
| 9 | i64 | OverTime（util_get_time_64 秒） | push 9 |
| 10 | string | CurBiderUid（""=无） | push 0xA |
| 11 | i64 | CurPrice（初始=起拍价） | push 0xB |
| 12 | int | State（1 在拍/2 有出价/12 一口价瞬态） | push 0xC |
| 13 | int | ItemType（→BigType） | push 0xD |
| 14 | widestr | SellerName | 类型闸（wide） |
| 15 | int | Level | Compare 前置读 push 0xF |
| 16 | int | Quality | push 0x10 |
| 17 | string | （透传） | 类型闸 |

空行管理：FindEmptyRow/InitEmptyRowList 空行号栈（LIFO），AddItem 空行写入+摘号，
RemoveItem=ClearRow+回号。

## 五、AuctionManager（单例 @0x1014C428）

- **OnAuctionDataLoaded**（Store 加载完成回调）＝ LoadResource → InitAuctionData
  （AddAuctionData 建数据对象 + AddRecord 建表）→ InitRecords（MainRecord 包装 +
  4×6 SortedRecord）→ InitTimers（`HCheckDueTime` 定时器注册 vt+0x24/+0x2C）。
- **LoadResource**（@0x10093D80）：`resourcePath + "ini/trade/AuctionHouse/" +
  "ItemType.xml"`，逐 `<Property>` 建 `map[ID]=BigType`；缺文件仅 Echo
  `"[Error]AuctionManager::LoadResource: Not found file:"` 继续。
- **GetBigType(itemType)**：查映射表（ItemType.xml 的 BigType 域：0无 1武器 2防具 3消耗 4其他）。
- **排序视图**：`m_vecSortedRecords[1..4][1..6]`，记录名 =
  `{Weapon|Armor|Consume|Other}` + `{SortedFirstByQuality|SortFirstByRestTime|
  SortedFirstByLevel|SortedFirstByCurPrice|SortedFirstByFixedPrice|
  SortedFirstBySellerName}`（构造器名字表 0x1012AFA8/0x1014A364，CRT 动态初始化器
  @0x10124770 填表实证）。比较器全部是值差（CompareQuality=a-b），
  SortedRecord::Compare @0x1008F730 按槽分发。
- **容量**：构造器常量 0x186A0（总）/0x61A8（单大类）。
- **HCheckDueTime**（@0x10090150）：倒序扫 MainRecord，`now ≥ OverTime` →
  ProcessOverTimeItem。`msgid = 10 + (State==2)`，成交分支多带 curprice/feecoe/bider
  三列，随后 RemoveItem + 摘视图 + 广播(0,0)。

## 六、FsGame 侧（客户端链路，符号级；详见 tmp_research/auction/fsgame_relay.md）

**本快照重要事实：FsGame 的客户端拍卖入口是死码**——`OnCustomMsg @0x1042AEC0`
被替换为 0x48 字节 stub：任何 custommsg **0xC5(197)** 都只回
`sys_auction_house_9999`；20 个 CustomMsgHandler / OnPublicMsg 全部无 xref。
活跃链路 = 客户端 → **FsSoloMember 场景服**（game_member.ini [Logic] Module0=
fsgame.dll）`OnAuctionMessage`（事件表 idx 74）→ `CKernel::SendAuctionMessage`
→ inner **16003** → 本服。死码中仍完整保留了历史方言：

- 出站 PubMsg：1 寄售 / 2 取消 / 3 竞拍 / 4 一口价（回包 = +6 → 7/8/9/10；
  帧冠 `["domain", domainName(wstr), msgid, ...]`）。
- 寄售 22 字段全序列（含 int64 consign_id=gen_item_unique_id、save_to_string
  物品串、起拍/一口价、f32 手续费系数、保管费、到期时间、数量、时长档、
  wstr 玩家名）——与 §四 负载类型序列逐位互证。
- 自身校验（CheckConsignArgs）：BindStatus==1→err20；禁拍 buff→err19；
  总量 ≥100000 / 单类 ≥25000→err22（ah_is_full）；个人寄售数 ≥ConsignLimit
  （VIP 走 VipModule::GetAHConsignNumLimit）→err21；保管费 =
  `max(price×coe×数量/100, KeepingFeeMin)`；扣款日志 80018"AHBid"/
  80020"AHFixedBid"。CustomErrorInfo 31 项错误码→sys_auction_house_XXXX 全表
  在 fsgame_relay.md。
- 配置：AHConfig.xml（ConsignLimit=10、KeepingFeeCoe="12,0.1|24,0.2|48,0.4"、
  KeepingFeeMin=1、TransactFeeCoe=0.15、Search/MyConsign/MyBid 每页 20）；
  ItemType.xml 同时建 ItemTypeTree（RecName/Father）。
- 玩家侧记录表（Res/struct/player/auction.xml）：rec_consign_args(4 列×300) /
  rec_search_args(13 列) / rec_my_consign(150) / rec_my_bid(200)；
  搜索/分页/我的寄售/我的竞拍全部在 FsGame 本地 PubData 镜像上执行，
  不走拍卖服。

## 七、Go 复刻对照表（internal/auction）

| 官方 | Go | 备注 |
|---|---|---|
| MainRecord+Row | `record.go MainRecord/Row` | 18 列对齐；free 栈语义同 FindEmptyRow |
| SortedRecord 4×6 | `record.go SortedRecord` | 二分插入；OnItemAdded/Updated/Removed |
| AuctionManager | `manager.go Manager` | GetBigType/OnItem*/Rebuild/Clear/LoadResource |
| LoadResource | `Manager.LoadResource` | 复用 config.LoadObjectFile（GBK 自适应） |
| AuctionLogic.DispatchMessage | `logic.go Logic.Dispatch` | msg 1..6 + 静默校验语义 |
| MsgHandler*→Do* | `msgConsign/msgBid/...` | 回执元素序按反汇编 LIFO 消费序钉死 |
| H_CheckOverTime/ProcessOverTimeItem | `Logic.CheckOverTime` | 11 成交 / 12 流拍 + 广播 target=0 |
| 帧载体 | `internal/proto.VarList` | 线格式与框架 §6 逐字节一致（u8 tag+值） |
| game_auction.ini | `auctionconfig.go` + 内置默认（embed 释放式） | 官方方言；[Inner] Port 官方不写→缺省 16003（main.lua 同源）；[Resource] Path 为 Go 扩展 |

测试：`go test ./internal/auction/`——寄售/竞拍/一口价/取消/超时结算/排序视图/
GM 清表/官方 ItemType.xml 真档装载（80 条）全绿。

## 八、后续挂点（未做，按里程碑推进）

1. **inner 16003 帧接入**：`fscomm`（并行会话 WIP）+ 本包 Dispatch 出入口对接
   `auction_forw_message_t{param2,count,vary}` 封装。
2. **持久化**：Store PUB4 归档协议（attr 0x41/record 0x42 分片存、0x29/0x2A/0x2B 加载），
   与 persist 里程碑 nx_chunks 线合流。
3. **FsGame 转发适配**：AuctionHouseModule 客户端 custommsg 全表 + 校验参数移植
   （AHConfig.xml 数值已在手）。

## 九、证据索引（FsAuctionLogic.dll 内 VA）

| 内容 | VA |
|---|---|
| DispatchMessage / 6 槽跳表 | 0x1008FEB0 / 0x100900A8 |
| Check{Consign|CancelConsign|Bid|FixedBid}Args | 0x10090650 / 0x10090AF0 / 0x10090C20 / 0x10090D90 |
| MsgHandler{Consign|Cancel|Bid|FixedBid|Rebuild|Clear} | 0x10090EE0 / 0x100911B0 / 0x10091550 / 0x10091870 / 0x10091D30 / 0x10091D90 |
| Do{Consign|CancelConsign|Bid|FixedBid} | 0x10091DF0 / 0x10091F20 / 0x10092030 / 0x10092170 |
| ProcessOverTimeItem / H_CheckOverTime | 0x100922E0 / 0x10090150 |
| AuctionManager::{ctor|LoadResource|InitAuctionData|InitRecords|InitTimers|AddRecord} | 0x10092940 / 0x10093D80 / 0x10094250 / 0x10094310 / 0x10094830 / 0x10094930 |
| MainRecord::{AddItem|FindItem|Get*} | 0x1008B520 / 0x1008B4A0 / 0x1008B710… |
| SortedRecord::{Compare|CompareQuality|…|OnItemAdded|OnItemRemoved} | 0x1008F730 / 0x1008FBC0… / 0x1008F410 / 0x1008F6A0 |
| callee vtable / FxModule_Init | 0x1012F260 / 0x100AD790 |
| 排序名前缀表 / 后缀表 / 定时器名 | 0x1012AFA8 / 0x1012A46C(×5 同容) / L"HCheckDueTime" |
