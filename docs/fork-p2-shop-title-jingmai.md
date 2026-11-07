# fork P2 复刻：商店购销（70/71）、称号（295）、经脉（150/161）

> 2026-11-07 落地。证据三方交叉：fork 二进制（`F:\Game\Online\9yin\9yin_server1\build\9yin-game-native-menu.exe`，`go tool objdump` 行号级反汇编）+ Java 参考服（`G:\pojie\ai\9yin-server`，msg70 全链权威蓝本）+ 官方属性表（`official_property_table.txt`）。
> 复现产物：`C:\Users\lzy\AppData\Local\Temp\fork_p2\{sbu,sse,jma,jmc,tst,scm}.txt`（objdump 导出）。

## 一、msg70 商店购买（handleShopBuyCustom，shop_buy.go:89-…）

fork 反汇编钉死（sbu.txt）：

- 入参 `[70, str shopId(tag6/7), i32 page, i32 pos, i32 count]`；
  **count ∈ [1,98]**（`CMPL count, 0x62`；View61 行 MaxAmount=99 的服务端闸）。
- **fork 不查开店会话态**，直接以消息里的 shopId 装载
  `share/trade/shop.ini` 小节（Java 则要求 openShop 态——本服从 fork）。
- 货位匹配 `item.page == page+1 && item.position == pos-1`，与
  `openShopLocked` 的行索引 `(page-1)*10+position+1` 同一坐标系。
- 计价 `cost = price × count`；按 capitalType 扣款（fork 按属性名
  `"CapitalType0/1/2/4"` 字符串分派：金/银/卡/银票；余额不足静默拒）。
- capitalType=3 为 exchange 计价（`exchangeCatalogFor`+`consumeBagConfig`
  按物品付费）——本服 shop.ini 有 43004 行 type3，**本期不做**，拒绝并记日志。
- 发放 `grantBagItemUnits`；成功回帧 `currenciesUpdate`（三币各一帧 0x10），
  落库 persistBagEquip + currency store（`SELECT snapshot FROM role_currency`）。
- msg345 获得物品提示**不发**（Java 实证）。

本服实现（`shop_buy.go`）：扣款成功但发放失败（包满）时**回滚扣款**
（fork 无此路径，纯防吞钱）；同类同页行内叠堆优先
（`stackTargetFor` + `increaseBagItem`），否则 `lookupGrantable` + `grantBagItem`。

## 二、msg71 商店出售（handleShopSellCustom，shop_buy.go:27-…）

sse.txt 钉死：

- 入参 `[71, i32 类别页, i32 槽位(1 起), i32 数量, str configID]`
  （5 值；values[3].Type∈{6,7} 校验）。
- 类别→线缆视图映射内联（bag_store.go:183-187）：`2→121、3→123、
  4→125、缺省→2`——即 `bagViewOf` 同映射；**槽位取 idx-1**。
- **出售价 = 10 碎银 × 数量**（0x140455cd9：`ADDL CX,CX; LEAL (CX)(CX*4)`
  = 5×2×count），无逐物品价格表。
- decrementBagItem → 删净 `serverViewRemove(view,slot)` / 剩余重推行 →
  `addSilver` → `currenciesUpdate` → persistBagEquip + currency persist。

## 三、msg295 称号（handleSetTitleCustom，title_system.go:189-202）

tst.txt 钉死——fork 的 C2S 路径**极简**：

```go
if len(values) < 2 { return false, nil }
titleID := values[1].Int32
p.setTitleID(titleID)            // 互斥锁直写，无所有权校验、不授予
frame := titleIDUpdateFrame(titleID)
link.WriteFrame(frame)
```

- 回帧：0x10 单属性 `TitleID@360`。fork 以 `MOVW $0x168`（u16）写值——
  那是 fork 自协商表的线型；**本服走官方表（360 TitleID wire3=i32）**，
  用 Int32Value。
- fork 的 title_rec 记录行（`titleRecAddRow→serverRecordAddRow` 5 列）与
  重放（`titleRecWorldEntryFrames`）属授予/进世界链路，不在 C2S handler 内。
- "titleId must be within 0-500000" 范围闸在 fork 辅助函数内。

本服实现（`title_system.go`）：保留范围闸 0..500000；佩戴位落
`role_titles`（Java PanelDataStore 同表形：clear-all-worn + upsert 目标行，
未拥有即补行——与 fork 宽松语义配套）；`title_rec` schema 5 列
`[3,3,6,1,6]`（数值列语义未校准零填充，FashionSuitRec 同例）进世界重放。

## 四、msg150 经脉修炼（handleJingMaiCultivateCustom，jingmai_progress.go:280-…）

jmc.txt 钉死 + Java JingMaiCustomHandler 全链对齐（2026-11-07 重对齐，
以 Java 为准）：

- `[150, i32 sub, ...]`：**sub=1** 设修炼（len≥3 且 values[2] 为字符串；
  `setCurJingMai`：cur→last、新 id 入 cur）；**sub=2** 真气刷新（仅重推
  面板帧）；其余子令 "unhandled" 日志。
- 修炼闸=**已学**（Java「未学经脉静默拒」：客户端不弹错，服务端只记
  日志）。已学集合 = starter 14 条（`starterJingMaiIDs`，Java
  MeridianCatalog.STARTER_MERIDIANS：丹田+十二正经+阴跷·隐/阳跷·隐，
  F 盘 52795 抓包 View45 行序）首次进世界种子、每条 1 级 0 真气
  （`ensureJingMaiLearned` ≡ Java PlayerMeridian.create）。目录其余条目
  （逆脉 _ni/金脉 _jin，共 35 条）默认未学：不发 View45 行、不可修炼。
- **首次开修自动补 active_jingmai_rec 行**（Java setCultivating：此前
  未修炼且无激活行 → 激活位=当前修炼经脉，jmActCount=1）。
- 分支后发专用 8 字段帧（CurJingMai@849 str / LastJingMai@850 str /
  JingMaiTotalLevel@819 / jmActCount@851=1 / ZhenQiDayValue@852=500 /
  ZQPLValue@855=今日已耗 / ZQActlValue@856=0 / ZQUnUsedValue@857=0）。
  该 8 属性同时并入 0x0D 对象档与 stateProperties 大批（Java addObject/
  stateProperties 同位）——面板首开即有值，不必等 ClientReady 补推。

数值模型（Java MeridianCatalog 同源）：**750 真气/周天、43 真气/分钟、
日预算 ZhenQiDayValue=500**；MaxLevel = `jingmai_static.ini` 的
`MaxVarPropNo-MinVarPropNo+1`（丹田 StaticData=41 → 141001..141180 = 180、
正经 216、跷脉 225；目录 35 条）。满级截断（不再耗真气）。
View45 行 `TotalFillValue` = 每周天所需真气 **750**（Java
totalFillPerLevel 常量），不是历史累计。

## 五、msg161 经脉激活（handleJingMaiActiveCustom，jingmai_progress.go:226-264）

jma.txt 钉死：

- `[161, i32 sub(1=激活/2=去活), str jingmaiId]`（len≥3、sub∈{1,2}、
  id 非空窄串，否则 malformed 拒）。
- 激活：`activateJingMaiRecord`（已存在→"already present" 日志，不加行）；
  去活：`deactivateJingMaiRecord` 删行（**7/26 源码快照无此分支，9/14
  二进制新增**）。两者后发 `vitalUpdate`。
- 记录名 `active_jingmai_rec`（官方下标 162），单 string 列。

本服：`serverRecordAddString`（idx=9）增行 / `serverRecordDelRow`(0x12)
删 0 基行；150/1 首修自动激活走 Java 语义（见 §四）。

## 六、记录表扩展（位置协商，只许末尾追加）

`qingGongRecordSchemas` 9→11：

| idx | name | 列型 | 来源 |
|---|---|---|---|
| 9 | active_jingmai_rec | [7] | 官方 162 号记录，单 string |
| 10 | title_rec | [3,3,6,1,6] | 官方 5 列 u32,u32,f64,u8,f64 |

`serverRecordAddCells` 新增 recordColumnByte(1)/recordColumnDouble(6)
两种列型编解码（FxNet2：类型 1=1 字节、6=IEEE-754 double 8 字节）。
`TestQingGongRecordSchemaAndRowLayout` 的表头计数守卫同步 9→11。

## 七、持久化与登录链

- **role_currency**（migrations/0014）：快照 JSON `{Silver}`（fork 二进制
  字段名对齐）；金/银卡会话级不落库（fork 快照仅 Silver）。买/卖后
  `persistCurrency`；登录 `Load` 覆盖 `-starter-silver` 初值。
- **role_titles**（migrations/0014）：拥有+单佩戴位；登录 `restoreTitles`
  （is_worn 行回填 TitleID）。
- **role_meridians**（migrations/0004 已有）：快照 {cur,last,learned,
  activated,progress,zhenQi}（learned=已学授予序，旧档缺节点回填
  starter+进度键序）；150/161 后与 ticker 变化时 `persistJingMai`
  （persistFaculty 同模式，defer 兜底）。
- 服务器运行时**不做 DDL**（仓库铁律），缺表一律降级会话态只记日志；
  建表权威 = migrations/ 重放（deploy.bat 第 3 步）。
- **资源供给**（2026-11-07「经脉数据没有下发」根因）：目录读
  `resources/modern/share/skill/jingmai/{jingmai,jingmai_static}.ini`，
  经三层资源仓（本地 > res-patch > res-base）；本地开发/CI 无散装
  resources 时必须由 testdata/resstore 夹具仓供给（`mpq add` 已注入，
  `TestJingMaiCatalogResolvesFromFixtureStore` 回归钉）。缺目录时
  msg150/161 整域静默降级。
- 登录重放挂在 `grantStarterProgress`（ClientReady 与 0x0C 重入两路都过，
  Java LoginFlow meridian 授予块同位：46/43/40 之后）：View45 容器+
  已学行（learned 授予序，非目录全量）→ active_jingmai_rec 行（cur
  非空即补行，noteMeridianActivated 同位）→ 经脉面板帧 → title_rec
  行 + 佩戴位 0x10。
- ticker 挂 facultyTicker（1s）内 `jingMaiCultivateTick`：整分钟结算、
  变化才发 View45 行帧（升级帧带 Level@1465 byte + TotalFillValue@765=750，
  否则单 CurFillValue@764，均 isView=1 obj=45 owner=已学行号）。

## 八、货币帧

| 币 | 属性 | idx | 线型 | 持久化 |
|---|---|---|---|---|
| 金币 CapitalType0 | 416 | i64 | 会话级 |
| 碎银 CapitalType1 | 417 | i64 | role_currency |
| 银卡 CapitalType2 | 418 | i64 | 会话级 |

`currenciesUpdate()` 三币各一帧 0x10（fork 买/卖成功路径统一回帧）。

## 九、测试与验证

- `shop_buy_test.go` 6 项：叠堆扣款+帧序（含三币 idx 416/417/418）、
  余额不足静默拒、空货位/超 98/未知商店拒、售出固定价 10/件+售空删行、
  陈旧 configID/超量拒、sqlmock 购买→role_currency 落库。
- `title_system_test.go` 5 项：TitleID@360 帧字节+落库、取消仅清位、
  范围/缺参拒、title_rec 5 列编码（41 字节帧）、重放 2 行+佩戴帧。
- `jingmai_progress_test.go` 8 项：目录区间宽（180/225 家族）、cur/last
  迁移+停修、未知 id/未知子令拒、161 增删行（0x11 idx9 / 0x12 row0）、
  ticker 结算（86/2min、750 周天升级、日预算 1 点补齐、满级零消耗）、
  View45 帧组、快照 JSON 往返、重放 5 帧序。
- `p2_db_integration_test.go` 3 项：NINEYIN_TEST_MYSQL_DSN 门控（缺表
  skip 提示重放 migrations；借用 roles 行并恢复原状，不落测试残留）。
- 全量 `go test ./...` 零回归；唯一守卫更新 = 记录表计数 9→11（有意扩展）。

## 十、遗留（后续）

- exchange 计价商店（capitalType=3，43004 行货单）与 fork 的
  `consumeBagConfig` 物品付费链。
- GM 面板称号指令（grant/wear/cancel——fork `grantAndSetTitle` 通道）。
- 经脉数值加成链（jingmai_varprop→PropModifyPackRec）与一步经脉 172。
- fork 的官方数据预取（api.nineyinmanual.com `meridians_official.json`）。
- 真机联调（部署 F:\Game\Online\9yin\9yin-server-go 后验帧序）。

## 十一、邮件/坐骑登录重放（2026-11-07 第二批，Java LoginFlow 对齐）

背景：Java ClientReady 共 14 个授予块，Go 补齐前缺 邮件/坐骑/声望/身份/
兵录 五域；共享库实测只有 role_letters（1 行）与 role_mounts（1 行）有
数据（role 618 官服迁移档：1 封系统邮件 + 2 只坐骑），故本批移植此两域，
其余三域空表零帧（记录下标已探明备用：Repute_Record=44 [7,3,3]、
Origin_Active=45 [3]、Origin_Prized=48 [3]、binglu_rank_rec=178 [2,2,2]）。

- **邮件**（mail_records.go）：role_letters.snapshot `{"letters":[…]}`
  （snake_case）→ RecvLetterRec **319** 十六列行（官方列型
  [8,7,3,8,8,7,7,7,4,3,7,7,3,4,3,6]，col0/3/4 宽串）。空邮箱零帧；
  ClientReady 推送位于 43/40 之前（Java lettersGranted 同位）；0x0C
  重入整表重放。已读/删信 C2S 变更路径暂缺（重放只读）。
- **坐骑**（mount_records.go）：role_mounts.snapshot 裸数组（字段即
  NewRideRec 列镜像）→ NewRideRec **310** `[3,3,6,1]` 拥有行（col2=f64
  ValidTime、col3=Selected 位），整刷先 0x12 倒序清旧行；随后骑乘态
  单帧 0x10：Mount@77 空串（重登步行口径）+ NewRideID@1333 当前包 +
  CurRiderType@82。无存档零帧。
- **坐骑目录**（CurRiderType 查表）：tool_item.ini script=Mount 段
  （MountType 主源/ArtPack）+ itemartstatic.ini MaleModel（npc\ride_XXX，
  无 MaleModel 不可骑）+ share/ride/new_ride.ini MountType 兜底——Java
  MountCatalog 三表同序合并，真表 72 只可骑。夹具仓已注入 tool_item.ini
  （20MB）+ new_ride.ini（此前 share\item\ 下只有 equipment/
  itemartstatic，本地 CurRiderType 整域静默降 0）。
- **线型 8 宽串**：serverRecordAddCells 新增 case（u32(UTF-16LE 字节数
  +2) + 宽串 + 尾 u16 NUL，Java RecordMessages case 8 同构）；init 自检
  挂 NewRideRec/RecvLetterRec。
- 测试：mail_records_test（宽串编码/快照方言/空邮箱零帧）、
  mount_records_test（清行+重推+骑乘态/三表合并 4 分支/快照方言）、
  mount_fixture_res_test（真表回归钉 41523/9513）、p2 集成 2 项
  （role_letters/role_mounts 快照往返，Java 运行时建表故缺表 skip）。
