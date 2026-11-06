# msg31 道具使用效果链（宝箱开启/使用物）——探索取证与实现

2026-11-04 整理。范围：客户端右键使用的完整语义、宝箱类物品的数据模型与掉落包全景、
参考实现盘点、本服实现决策记录。契约脚本层见
[skill-script-framework.md](skill-script-framework.md) §item；帧格式见
[go-source-protocol-notes.md](go-source-protocol-notes.md) §7.3。

## 1. 客户端侧语义（F:\Game\Online\9yin\九阴资源\lua64_unpacked 取证）

右键使用链路（`lua64\form_stage_main\form_bag_func.lua`）：

- `on_bag_right_click(grid, index)`（:309）先分流特殊 UI（仓库/摆摊/寻宝修复/合成、
  `do_color_egg`、`do_bind`），书本类开预览，其余走 `confirm_use_item(item)`（:1372，
  仅特定类型弹确认：ITEMTYPE_COMPOSE_LILIANYAO、FuncPack==8201 寻宝、带 AddBufferList）。
- 最终都汇到 **`GoodsGrid:ViewGridUseItem(grid)`**（:479/483）——GoodsGrid 是 C++
  引擎对象（`nx_value("GoodsGrid")`），不在 Lua 层定义。
- 绑定后使用（`form_itembind\form_itembind_use.lua:33`）、新背包 UI
  （`form_bag_new_func.lua:183` / `use_item_by_configid:111`）同一出口。
- Shift 连用：`GoodsGrid:IsRepeatedlyUsedItem`（名单在
  `resources\share\item\item_using_repeatedly.ini`）+ 输入框写
  `grid.times_for_using_item_repeatedly` 后同口重放。

**上行 = 原生 msg31（非 custom，`lua64-protocol-dict.md` 里查不到）**：恰好 4 值
`[31, view(int), slot(int), count(int≥1)]`；3/5 值或 count=0 拒绝。出处三重钉死：
menu2.exe 字符串 `reject malformed native item use values=` + `USEITEM fwz card`
（go-source-protocol-notes.md §7.3 :250）、本服 LoginFlow.handleBodyItemUse 实测、
bak 服 useItem 行为。

**下行预期：普通宝箱（锦盒/礼包）没有任何专用握手——纯服务端驱动**。客户端只认：

- 背包行帧 `0x18 ViewAdd` / `0x19 ViewRemove`（视图表 `lua64\share\view_define.lua`；
  背包相关 VIEWPORT_EQUIP=1 / VIEWPORT_TOOL=2 / VIEWPORT_EQUIP_TOOL=121 /
  BOX_CHOICES=155 / DROP_BOX=80 / TREASUREBOX=158）；
- 刷新胶水 `goods_grid.lua` 的 `view_grid_on_update_item/on_delete_item` →
  引擎 `ViewUpdateItem/ViewDeleteItem`；
- 可选文字反馈走命令名寻址 custom 下行 `custom_specific_sysinfo/custom_sysinfo`
  （`custom_handler.lua:87/:264`，tips_type 路由系统条/聊天/居中）。

存在但**均非普通背包宝箱**的箱类 UI（后续做选择箱时才碰）：

| UI | 视图/消息 | lua |
|---|---|---|
| 选择箱（N 选 1） | view 155 物化候选 + C2S 389 `custom_choice(2, pos)` 选取 / `custom_choice(1)` 取消 | `form_pick\form_choices.lua`（容量 16） |
| 寻宝宝箱收集 | S2C `custom_treasure_chest` + C2S 638 | `form_treasure_chest.lua` |
| 运营奖品箱 | S2C `custom_box_event` + C2S 713 | `form_box_event_prize.lua` |
| 邮件箱 | C2S 350（复合箱 603-607） | — |
| 掉落拾取 | view 80 + C2S 380-382 | `form_droppick` |

GM 相关：lua64 无发物品 GM 链（仅 custom_gminfo/msg2）；`custom_give_item`
（msg 80）是**给 NPC 交任务物品**，勿混淆。giveitem 发箱的真源是 NPC optiontalk：
`Option=giveitem|<选项>|Box_qycdu_00N*1|11285|3600|||1`（`share\rule\optiontalk_dialog_pack.ini`，
参数=物品*数量 | sysinfo 11285 | 选项冷却 3600s）。

## 2. 物品数据模型（share/item/tool_item.ini，GBK ~1.2M 行）

- 节名即 configId（**无数值 ID**），键大小写不敏感；grep 必须带 `-a`（GBK 会被当二进制）。
- **`script=` 是行为类**（服务端语义分发键）：SkillPage ~24.8k、ToolItem ~20.1k、
  **BoxItem ~14.7k**、SkillBook ~6.1k、CardItem ~1.1k、WuJiItem ~1.1k、
  **CompensateItem 744**、ExchangeItem ~0.5k（两份 G 拷贝与 F 运行拷贝条数略有出入）。
- `ItemType=` 数值类（`lua64\share\itemtype_define.lua`）：2102=GIVENBOXITEM 礼盒、
  2103=BOXITEM 包裹、2159=奇遇锦盒、34=WUJI_BOX、68=FIND_TREASURE、2156=COLOR_EGG。
  客户端侧"箱性"=ItemType 2102/2103；无 UseType/EffectType 字段。

**BoxItem**（开包类）：`DropID=` 直接指向掉落包名，且惯例**自指**——
`[qy_chengdu_009] Name=包裹 script=BoxItem ItemType=2102 DropID=qy_chengdu_009`
（G 拷贝 :214960 / 9yin_server_new 拷贝 :219765）。

**CompensateItem**（奇遇锦盒类，Box_qycdu_009 真身，:355418）：

```ini
[Box_qycdu_009]
Name=成都奇遇
ItemType=2159
script=CompensateItem
@CompensateItemRec=18658,qy_chengdu_009
@CompensateItemRec=18659,box_qycdu_009
```

- `@CompensateItemRec=<条件Id>,<ref>` 重复键逐行列出奖励引用；Box_qycdu_001..011 均为
  双 ref（18658→qy_chengdu_NNN 奖励包 + 18659→box_qycdu_NNN），016 单 ref。
- **ref 两种落点**：`qy_chengdu_009` 有同名 BoxItem 节（DropID 自指）；
  `box_qycdu_009` 无节、纯掉落包名。因自指惯例，统一"ref 直接当 dropId"即可。
- 条件门控（`share\rule\condition.ini` :167060）：`[18658] Type=19 Func Para1=56
  Para2="==" Para3=2`、`[18659] Para2="<>" Para3=3 Para4=9` —— **玩家 func-56 状态**
  决定走哪个分支（==2 发奖励包；∉3..9 发箱包；本服未接该状态，见 §5 简化）。

**giftbox.ini**（项目自建 UTF-8 表，服务端识别 `script=BoxItem` 用 DropID 指向
drops_mvp.json；仅 `[box_chinajoy_2026_giftbox]` 一条，含 UseCondition=17545/DropBind/过期）。
**item_use_skill_static.ini**（GBK）：`[itemType] SkillID/SkillLevel`——2101→yaopinhp1 L1、
2102→L2、2103→L3、2104→L4；yaoshihp L1-10 在 [11]-[21]；dushiyb [2007]。药品=使用技能。

## 3. 掉落包数据全景（谁在什么位置存奖励）

| 数据源 | 位置 | 形态 | 消费方 |
|---|---|---|---|
| **drops_mvp.json** | `share\drop\`（go-server1 + F:×3，四副本须 md5 全等） | `packages.{id}=[{item,min,max,weight}]`，头注释自述"MVP 自制包承接原版缺失包体"；现 5328 包 | **本服开包唯一权威**（DropPacks） |
| drop_table.json | 9yin_server_public / 9yin_server1 的 `share\item\` | `{"<DropID>":{mode:random|fixed, rolls, entries}}` | C++ 参考树，Java 未用 |
| giftbox.json → MySQL game_gift_boxes | go-server1 `data\`（322/764 箱） | 固定**全发**（weight 无意义） | 仅 gm-web 面板 |
| 道具配置.json → game_item_effects | data-import | heal_hp/heal_mp/consume/effects | 已导入未消费（面板） |

**缺口事实**（2026-11-04 探索钉死）：`qy_chengdu_009`/`box_qycdu_009` 包体在所有
数据源中**原本均不存在**（`qycdu` 全库 0 命中；qy_ 前缀仅 105 包，成都系只有
qy_chengdu_021/022）——官方包体从未随资源发布。本次按 qy_chengdu_021/chinajoy
形制 authored 补齐（yaopin_00100/00300/00500/00700 权重阶梯，箱包偏高档），
四副本 md5 `0d55ea8b…` 全等。**9yin_server_new 原本连 share\drop\ 目录都没有**
（giftbox.ini 也缺，本次补拷；item_use_skill_static.ini 原有）。

## 4. 参考实现盘点

- **bak 树 = 唯一完整参考**（本次移植源）：`9yin-server-bak\...\game\SceneFlow.java:1950`
  `useItem()` 五步（节流→Lua 优先→INI 回退→效果落账→消耗帧对）+
  `ItemUseCatalog` / `DropPacks` / `script\ItemScriptBridge`（指令白名单解析）。
- **Go 活体 exe（cmd/protocol-probe）没有 msg31/30/886-889**——只做登录/心跳/技能/
  快捷栏/打坐/格挡等 custom 面；`OpenBoxFunc`（names_gen.go:1019）只是 NPC 规则
  属性（GivingNpc.xml 世界箱），与背包物品无关。野外开箱在 bak 是 CloneLoot
  （boxnpc NPC + S2C 0x05 worldInfo 文字）。
- **两个"官服"zip 均非官服**：`九阴服务端.zip`（31,546 条目）= go-server1 工作树快照；
  `野猪端.zip`（11,824 条目）= 更旧 Go 树 + mysql + 抓包。全库 0 个服务端 .lua。
  官方 BoxItem.lua/CompensateItem.lua 在磁盘上不可得，语义只能从数据逆推。

## 5. 本服实现（2026-11-04，分支 9yin-main-new）

解析优先级与指令白名单见 skill-script-framework.md §item。要点与决策记录：

- 接线点 = `LoginFlow.handleBodyItemUse` 不可穿戴/非卡片分支 → `useItemEffect`
  （五步对齐 bak）。`consumeBagCard` 改名 `consumeBagItemOne` 通用化（逻辑零改）。
- 新包 `server.game.item`：`ItemUseCatalog`（giftbox.ini + tool_item.ini 只捕
  script/DropID/@CompensateItemRec 三键 + item_use_skill_static 阶梯；ResourceCache.of
  键加前缀防撞）、`DropPacks`（JSON 解析器自带；roll 抽 1 条/次、数量 min..max、
  同物品合并、rolls 钳 1..10）。
- `LuaScriptService.itemUse`：仿 skillOverlay 惯例（ItemModule_ 全局包裹、缓存+NONE
  哨兵、异常 markBroken 返 null 走 INI）。**返回 null=无脚本/nil/引擎坏**——
  与 bak 同语义，使用链永不拒绝。
- **简化决策**：
  - CompensateItem func-56 条件门控未接 → **ref 全发**（单机服口径，日志可追）；
  - temp_stat 已解析未实施（面板限时属性二期）——**仅剩 temp_stat 效果时不消耗防白吃**；
  - grants 部分失败（背包满）仍走消耗（bak 同款，防"半发+不消耗"重复刷取）；
  - 脚本 open_box 抽空包 → 提前返回不消耗（对 bak 的收紧，防 ScriptLock 白吃）；
  - 不发 sysinfo（与 GM 宝箱/发券现状一致，帧即反馈）。
- 节流：会话级 800ms（LoginSession.lastItemUseMs）；随机数取会话 combatRandom（可换种子）。
- 测试：`DropPacksTest`/`ItemUseCatalogTest`/`LuaItemUseTest`/`ItemUseFlowTest`
  15/15；FlowTest 走真 MySQL 临时角色 990033 全链（消耗→入包 DB 行+0x18 帧→重复拒）。
- 部署：jar 20:00 版已上 9yin_server_new（旧份备份 .bak-20261104-2005）；
  scripts/item 三个锚点原已镜像。

## 6. 真机首验根因与缺包兜底（2026-11-04 晚，已修已部署）

真机现象：「物品使用后没有生效」。取证（logs/nineyin-java.log）：

- 链路本身**健康**——21:55–22:01 多次成功：`box_item_xiakedao_haidibaoxiang_3`
  授予 yaopin_00100、`box_ding_0606xsh001` 授予 yaopin_00700×2、ng_book/item_ttf
  走 USE_SKILL 回血消耗；21:58 卡片收录链同源工作。
- 失败集中在 22:05 的 `box_20220401_mpwjb_fc_001`（门派武技包，role618 slot2，
  连点 20 次）——`drop packs [comd_drop_box_20220401_mpwjb_fc_001] empty/missing`：
  **msg31 已到、目录已判箱，但 DropID 引用的包体不在 drops_mvp.json → roll 空返 →
  提前 return 不消耗零帧**（当时设计防白吃，代价是静默）。
- 缺口全景（deployed tool_item.ini × drops_mvp.json diff 钉死）：箱类 15,232 节引用
  **16,903 个缺失包**（官方包体从未随资源发布，5,328 authored 包只覆盖零头）；
  giftbox.json（322 箱）/game_item_effects 均无这些包体。排查中的两个**红鲱鱼**记录：
  ① DB 旧种子行 item_type=0/script NULL——`bagItems()` 读路径的 game_items 富化层
  已经在内存补正（0x10/0x18 帧携带正确 ItemType），**线上无影响**；
  ② grep 窗口渗入邻节造成的「无 script= 行」误判（python 整节解析全都有 script=BoxItem）。

**修复（DropPacks.java，22:35 部署 9yin_server_new）**：缺包 roll 不再静默空返，改走
`FALLBACK_LADDER`（yaopin_00100/00300/00500/00700 权重阶梯 50/30/15/5，形制对齐
qy_chengdu 系 authored 补齐）并 INFO `drop pack {} missing, rolling fallback ladder`
可追；`has()` 仍只认 authored 包。空 dropId / 缺文件容错不变。任何箱现在都可开
（缺包箱给阶梯药），测试：`DropPacksTest.unknownDropIdFallsBackToLadderAndMissingFileTolerated`
+ `ItemUseFlowTest.missingDropPackFallsBackToLadderAndStillOpens`（真 MySQL 缺包箱
开包/消耗/掉落全链）。

## 6.5 按箱子介绍重建内容物对照（2026-11-05，box_contents.json）

**问题**：FALLBACK_LADDER 兜底后"任何箱可开"，但内容全是随机药品阶梯（drops_mvp.json
5,328 包里 5,324 个是合成 yaopin 阶梯，真实包仅 treasure_hht_1..4）——玩家开任何箱子
都与介绍文案无关。

**数据路线**（官方掉落包正文从未随客户端发布，唯一可靠对照=desc 描述文案）：

- 客户端 `text/chineses/{desc,desc2,desc3}.idres`（UTF-8，259,645 键）：14,787 个
  BoxItem 中 13,189 有描述、2,659 带精确清单（`内含菩提子*40、佛经书卷*1`）；
- `stringname.idres`（129,802 条 `configId=物品名`，**混有 desc_* 键须排除**）反查
  名称→configId；时限变体是独立物品名（`时装礼盒（7天）`/`马哨【赤电】(14天)`——
  全角/半角括号两形态都试）；
- 生成器 `tools/boxgen_rebuild.py` 四轮迭代：括号截断修复（`（7天`→`（7天）`）、
  品级/时限/前缀剥离（`百晓生兵录（上等`→`百晓生兵录`→binglu001 系同名任一）、
  手工 ALIAS（错别字 混元易经丹→faculty_event_02 等 12 条）、
  data/giftbox.json 老重建表**仅补缺**（老表 box_hd_wyls_05 漏兵录条目实测，
  新规则覆盖老表）。产物 2,583 箱 / 8,107 条目，未匹配 659 行长尾（NPC 名/家具/
  活跃度等真不存在的为主，`*.unmatched.csv` 可追）。

**接线**（`game.item.BoxContents`，share/drop/box_contents.json 固定全发）：
`useItemEffect` 的 `case BOX` 在 DropPacks roll **之前**先查 `byBox(configId)`——
命中逐条发 max 个（min=max=描述数量）+ INFO `box_contents hit`；未命中回落原链
（drops_mvp authored 包 → FALLBACK_LADDER）。Lua `on_use` 的 openDropId 显式指定包
路径不动。泛型描述箱（"右键点击打开"，~1 万个）无内容可抽，维持兜底。

**数据**：box_contents.json 四副本 md5 全等（go-server1 + F 三服
resources/modern/share/drop/）；resources 不入库，重跑生成器再同步。
**测试**：`BoxContentsTest` 4 + `ItemUseFlowTest.describedBoxGrantsExactContentsFromDescription`
（真 MySQL：box_prize_dmp_drzc_001 开箱 = item_dmp_ptz×40 + item_dmp_fjsj×1、
无 yaopin）。

**二轮增补（2026-11-05 晚，贵重/商人/宝物域界定）**：

- **全局可发过滤**：发放链 ToolItemCatalog = tool_item.ini ∪ equipment.ini；stringname
  有名但两表无节的 configId（武学书 book_wx_*、sns 袋、tre_/hc_ 系）发放时
  unknown config 断链 → 生成器统一剔除，剔空箱走兜底。**grantable 集合必须小写比较**
  （equipment 节名带大写 cjbook_CS_*，大小写敏感曾误剔 41 条，三轮修复）。
- **箱名直配（三轮，396 箱）**：箱名（或剥"礼包/礼盒/宝盒/锦盒/幸运包/包裹/奖励包/
  匣/福袋"后缀）去【】/括号/空格后 = 某非箱可发物品本名 → 开出本体；数量取描述里
  名称后紧贴数字或 ×N（无则 1）。案例：`无妄神功入门篇`（贵重箱）→ ng_book_jh_201_rm
  无妄神功(入门篇)×1；`魅影剑法残卷礼包` → cjbook_cs_jh_myjf【魅影剑法】残卷×10
  （desc"残卷10张"）；`行脚令礼包`→backtown_tool。索引只收非箱+grantable 条目，
  归一化同名冲突键剔除防误配；宝物域 skip 仍优先。
- **礼包商人 41 箱**：描述全泛型无清单，`MANUAL_BOXES` 手工配 33 箱（惊鸿礼盒/海底宝箱
  银金玉/帮会 BOSS 三档/赏钱钱袋/奇遇修炼包/太极剑太极拳/裁缝药师打造台/江南锦盒等，
  物品均实存可发）。
- **IsRarityItem=1 贵重箱**（60 个，多为过期活动礼包）：无任何来源的 50 个统一配
  贵重标准包（碎金×3+真气丹(10万)+玄石散+古玩瓷器），替代随机药（7 个被箱名直配
  精确接走，如无妄神功系列）。
- **宝物域不覆盖（明确边界）**：tdropbox*（未辨识的X，含 4 个数字名宝物池箱）与
  treasure_hht_1..4 的辨识产物 treasure_* 是宝物装备域、无服务端物品表，**走鉴定
  系统另案接入**——开箱对照刻意跳过（含 drops_mvp 里 treasure_hht 真实池其实也发
  不出的存量隐患，鉴定链一并解）。
- 武学书系（某某入门第N篇/进阶/残卷/秘籍）描述重建天然覆盖（87 箱开出书系，
  binglu131 百晓生兵录（进阶一）等品级剥除后正确反查）；仅 6 个书系名未匹配留 report。
  秘籍**使用**（学了而非开箱给书）见 §6.6。
- 三轮终稿 3,053 箱 / 8,616 条目（desc 2,524 + 箱名直配 396 + 手工 32 + 贵重兜底 50
  + 老表 59 + rarity 内含覆盖），不可发剔除后 ungrantable=0，未匹配 659 长尾。

## 6.6 秘籍学习链 script=SkillBook/SkillPage/SkillScroll（2026-11-05）

**等级判定**

篇章与等级的关系，**仅**用于判定秘籍是否可学，**不**用于判定秘籍是否可开。


这些篇章的学习限制主要体现在内功基础层数（必须学完前置层数）以及服务器九阴志阶段（时代封印限制）。具体层数和解锁对应关系如下：

1. 基础至中高阶阶段（1 - 36层）

此阶段通常在服务器处于一内、二内、三内九阴志阶段逐步开放。
• 入门篇：可学习/修炼至 1 ~ 9层。
• 进阶篇：可学习/修炼至 10 ~ 18层（需先将入门篇修满）。
• 中阶篇：可学习/修炼至 19 ~ 27层。
• 高阶篇：可学习/修炼至 28 ~ 36层。

2. 精研至秘传阶段（37 - 49层）

该阶段普遍在三内九阴志（倭寇跳梁）开启后允许进行合成和修炼。
• 精研篇：可学习/修炼至 37 ~ 40层。
• 精修篇：可学习/修炼至 41 ~ 43层。
• 深玄篇：可学习/修炼至 44 ~ 46层。
• 秘传篇：可学习/修炼至 47 ~ 49层。

3. 拾遗篇阶段（50 - 86层）

当江湖内功需要继续突破至64层时（通常对应服务器四内九阴志·魂碎九重前后），需要消耗残卷拼接对应的“拾遗篇”：
• 拾遗篇·一：可学习/修炼至 50 ~ 53层。
• 拾遗篇·二：可学习/修炼至 54 ~ 56层。
• 拾遗篇·三：可学习/修炼至 57 ~ 60层。
• 拾遗篇·四：可学习/修炼至 61 ~ 64层。
• 拾遗篇·五：可学习/修炼至 65 ~ 68层。
• 拾遗篇·六：可学习/修炼至 69 ~ 72层。
• 拾遗篇·七：可学习/修炼至 73 ~ 75层。
• 拾遗篇·八：可学习/修炼至 76 ~ 78层。
• 拾遗篇·九：可学习/修炼至 79 ~ 82层。
• 拾遗篇·十：可学习/修炼至 83 ~ 86层。

(注：64层以上则通常需要拼接“拾遗篇·五”至“拾遗篇·十”来突破至最高的86层。)

（数据核对 2026-11-05：全 6 个江湖内功家族的拾遗书 SkillMaxLevel / 传承 AddMaxLevel
与本表顶层逐项一致——52/56/60/64/68/72/75/78/82/86，服务端按数据驱动逐本生效；
唯一出入=拾遗篇·一官方数据 52（本表 53），已按数据 52 执行。见 §6.7。）

💡 核心学习限制与要点：

1. 等级不可跨越：必须由低到高逐层修满。例如：如果不学满【入门篇】的9层，就无法使用【进阶篇】去读第10层。
2. 九阴志时代限制：低内功时代的服务器即便通过高阶残卷合出了后面的篇章，也会因为服务器九阴志未达成而无法使用或无法合成。
3. 额外效果解锁：特定江湖内功修炼到部分篇章的满层（如49层、64层）且满足当前服务器九阴志要求时，会激活额外的内功特殊条目和特效。

还有就是 某某武学第X页，这个要学习必须需要学了上一层。当然入门篇 和 第一页 无条件学习。

**问题（真机反馈「秘籍类的东西使用后无效果」）**：秘籍没有独立使用分支，
`useItemEffect` 的 INI 回退按 **itemType** 查 `item_use_skill_static.ini`——而书类
ItemType（21/23/24/26/27/28…）在该表**撞型**于 yaoshihp 回血药品档
（[23]=yaoshihp L13…）。满血时 `Actor.restore` 闸掉回血 → 零可见效果零帧，
书却被静默消耗（白吃一本）。

**数据链（share 双表 + MySQL 目录）**：
- `tool_item.ini` 携带 `LearnSkillPack=<packId>` 的三类 script 全量可解析：
  SkillBook 6,116 / SkillPage 24,917 / SkillScroll 498；
- pack 查 `item_learn_skill_static.ini` `[packId] TragetID`（**官方拼写 TragetID**，
  非 TargetID；客户端 lua `get_limit_LearnSkillItem` 同源读法）；目标 1..N 个，
  ',' 与 ';' 混合分隔（大轻功1=`qinggong_20,..,25;QG_JH_gj_001`）；
- 目标 → `game_skills.skill_type` 分发：wuxue 1,489 / qinggong 64 / neigong 89 /
  anqi 28 / zhenfa 42（另有 45 个目录缺席目标，多为 `_hide` 变体，跳过记日志）。

**接线（`LoginFlow.useSkillBook`，handleBodyItemUse 内 useItemEffect 之前拦截）**：
- **wuxue/anqi**：`role_skills` 落库（INSERT IGNORE）+ `learnedSkills/skillRows`
  原地追加（新行号=当前最大行+1，Handler 持同引用即时生效）→ View40 **整视图
  重开**（0x16→0x15→行帧→CanUse 批→心跳批重建，GM forget_all_taolu 真机验证
  同款路径；行序按行号排序，skillRows 是 HashMap 迭代序不可倚）；anqi（暗器
  手法）随 wuxue 同入 View40，登录 `loadDbSkills` 改 `List.of("wuxue","anqi")`；
- **neigong**：`PlayerProgress.addMissingBooks(单本 def)` → View43 整视图重建 →
  `persistFaculty` 快照落库（GM learn_all_neigong 同链裁剪）；
- **qinggong**：`role_skills('qinggong')` 落库 + View46 **追加行**
  （`StarterQingGong.learnedFrames`：行号接 starter 52 行后，qinggong_ 前缀类目
  补 QingGongRec 记录行；不做视图重开——无真机验证且轻功面板按行增量维护）；
  登录 `grantOncePerConnectionRows` qinggong 块重放学得行（starter 集外过滤）；
- **zhenfa**（阵法 42 目标）：暂未接，跳过记日志；
- 条件字段（ConditionID/SchoolNeeded/ZhaoShiNeeded/PropNeeded…）单机服不门控
  （CompensateItem 全发同口径）；全部目标已学/不可解析 → **不消耗**防白吃，
  任一学会即消耗 1 本。

**测试**：`SkillBookCatalogTest` 3（GBK 双表/多分隔符/script 分派）+
`SkillBookFlowTest` 3（真 MySQL：套路书 book_CS_base_pgd01 学会 CS_base_pgd01 =
role_skills 落库 + 0x16/0x15/0x18 帧组 + 书消耗；全已学不消耗；未知 config
itemType=9999 维持 not implemented 口径——**itemType=23 会撞型被当药消耗**，
故 SkillBook 分支必须在 useItemEffect 之前）。坑：`role_skills` 无 roles 外键
（role_bag_items 有 CASCADE），测试 cleanScratch 须显式删行。

## 6.7 篇章/页书分层门槛与修炼上限（2026-11-05 二期，实现注记）

上面（§6.6 头部）的篇章表/规则按**官方数据语义**落地——分层与门槛全在表里，
无需硬编码篇章区间：

- **门槛表** `rule/condition.ini`（40,460 节，GBK；新 `game.item.SkillGateTable`）：
  道具 `UseCondition=<id>` → `Type=3 Name=Skill/NeiGongContainer Para2=MaxLevel
  min/max`＝目标**当前修炼上限**须 ∈ [min,max]。实例：梵心降魔第二页 cond 12193=
  `CS_em_jdmz01 上限=1`（即第一页已用，§6.6「第X页必须学了上一层」的官方编码）；
  气桩功进阶篇 13563=`ng_em_001 上限∈[9,17]`、中阶篇 13579=`[18,29]`、高阶篇
  13595=`[30,35]`——**[min,max] 上界=前一篇章顶层、下界=前篇已完成且本篇未用**，
  恰是"必须由低到高逐篇解锁"。书引用分布：Type3×5,696 + Type1 门派×3,443
  （Para1="==" min=`school_*` 编码，对 `PlayerActor.faction()`）+ Type19 Func×873
  （九阴志时代/功能态，本服未接时代系统→放行）+ 条件表缺席×1,222（放行宁松勿锁）。
- **分层字段**（道具自身而非 pack，`SkillBookCatalog.BookEntry`）：
  `SkillMaxLevel=N`＝解锁修炼上限到 N——第X页/入门第N篇=N、篇章汇总书=该篇顶层
  （门派内功 9/18/30/36；江湖内功各家随表到 86）；29,326/31,531 书携带。
  SkillScroll 传承心法走 `AddLevel=1`+`AddMaxLevel=N`（直升 1 层+上限抬到 N，498 本）。
- **落地面**（`useSkillBook` 重构）：门槛（`checkSkillBookGate`）不过→不消耗；
  wuxue/anqi（无玩家修炼链）＝**直达该层**（learnedSkills 的 level 即上限，
  夹到 `CombatSkillCatalog.maxLevelOf`，落 skillLevelStore）+ View40 整视图重开；
  neigong＝`PlayerProgress.learnOrRaiseBook(def, cap, setLevel)` **只抬修炼上限**
  （层由既有修炼 ticker 自涨，升层沿自然曲线 total+200/maxPower+20）+ View43 重建
  + role_faculty 快照；传承 AddLevel 直升层。全部目标已至该层→不消耗。

**实现语义与 §6.6 规则的两处口径差**（按官方数据）：
1. 官方门槛判的是**上限序列**（书链式解锁）而非训练层数——上限解锁后层仍靠修炼涨，
   不必"修满 9 层"才能读下一本（用户表述"修满"为近似）；
2. 九阴志时代（Type19 Func）单机服未接时代系统，一律放行——低时代合成/使用限制
   不生效；49/64 层额外特效条目未接（特效域二期）。

**测试**：`SkillBookCatalogTest` 5（+分层字段 + SkillGateTable 区间/门派/缺席容错）+
`SkillBookFlowTest` 5（+第二页门槛先拒（不消耗）后放（直升 2 层+落档）、晦明日月鉴
入门篇建书上限 9 非全档、种子内功 36 上限下「进阶第1篇」被门槛拒且上限不破坏）。
坑：武学等级档写在 `GO_ROOT/data/武学等级.json`（skillLevelStore 只存 >1 级），
FlowTest tearDown 须移除本角色档值防污参考仓。



- func-56 玩家状态接入后，CompensateItem 收敛回条件单分支；
- 选择箱（view 155 + msg 389 `custom_choice`，容量 16）——需要 N 选 1 礼盒时做；
- `custom_specific_sysinfo` 文字播报"获得物品"（官方 11285 等 string_id 无 dump，需抓包）；
- temp_stat 面板限时属性（同源替换 + tick 过期 + 面板重算）；
- UseCondition（如 chinajoy 17545）/ValidTime 过期校验目前忽略。

## 8. 坑位备忘

- repo 的 `resources/` 是 gitignored 且**不在仓库内**——测试一律
  `ServerPaths.overrideServerRoot(GO_ROOT)` 指向 go-server1；LuaScriptService 单例可能在
  模块 CWD 下初始化成 broken（formula 缺失），测试类 @BeforeAll 须 override + `reload()` 自愈。
- GBK ini grep 加 `-a`；iconv 大文件遇非法字节静默截断（用 python 整读）。
- drops_mvp.json / box_contents.json 改动必须四副本同步并 md5 校验（9yin_server_new
  的 drop 目录是 2026-11-04 新建，历史上该 SERVER_ROOT 无任何掉落数据）。
- role618 金丝雀（RoleInventoryStoreIntegrationTest）按当期 DB 对齐，真机穿脱即漂移：
  2026-11-04 晚 = 穿戴 8/背包 34/武学 190/内功 48/快捷栏 117/扩容 4。
