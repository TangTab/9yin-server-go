# 物品服务端 XML 构建链与 msg31 使用链定版（2026-11-24）

> 2026-11-24 更新：构建工具已合并为统一入口 `tools/datagen`（子命令 `npc` / `item` / `skill`，
> 无参构建全部），输入收窄为两类——客户端资源解压根（`-client`）与黑金参考资源根
> `testdata/references/h001.rep`。本文中的 `tools/npcgen`、`tools/skillgen`、`tools/itemgen` 命令
> 与数据源路径是定版当时的记录；现状以 `testdata/README.md` 的「构建数据」一节为准。


> 数据源：九阴设计态表 `go2 Res/share/item`（87 档）；黑金样板
> `F:\Game\Online\9yin\BGserverCJ\Res\ini\Item`（原版 37 档）；官服抓包
> `go2 资料\初始账号登陆-完整引导-女.pcapng`（8032 线，14 次 m31）。
> 结论先行：
> 1. **物品配置按黑金格式离线构建为服务端 XML**（`tools/itemgen` →
>    `ini/Item`：22 文件 12.5 万行，其中物品 104,684 件），服务端只读产物
>    （`internal/itemdata`）。
> 2. **msg31 使用道具受理链已接入**（`internal/fssoloworld/itemuse.go`）：
>    读条流帧序与官服抓包逐字节一致；效果面已接武学书学习、钱袋两类，
>    其余按数据拒收不扣件（清单见 §六）。
> 3. 本阶段**未真机验证**——帧序只有单测对拍与官服抓包佐证。

## 一、黑金物品配置格式（样板分析）

黑金原档 `ini/Item/` 三类文件：

| 类 | 文件 | 形态 |
|---|---|---|
| 登记表 | `path_config.xml` | `<Property ID Type Path/>` 逐档登记模板文件（Type=Equipment/Weapon/material/ItemUse/Rune/mount/TaskItemUse/Taskmaterial/add_money…） |
| 模板档 | `Equipment.xml`(4,506) / `Weapon.xml`(624) / `ItemUse.xml`(1,689) / `Material.xml` / `TaskMaterial.xml` / `TaskItemUse.xml` / `Rune.xml` / `ItemMountUse.xml` / `ItemPetUse.xml` / … | 一行一件物品：`<Property ID script ItemType ArtPack LogicPack FuncPack …/>`，`script` = 逻辑类（logic_class.xml 的服务端对象类，决定 struct schema） |
| 静态包 | `ItemArtStatic.xml` / `item_logic_pack.xml` / `item_skill_static.xml` / `item_create_static.xml` / `ItemLimited.xml` / `EquipmentSet.xml`… | 物品行以整型 ID 引用的小表；**前两行是 `ID="-1"`（列序头）/`ID="0"`（列宽头）装载器元数据** |
| 类型树 | `ItemTypeList.xml` | `ID / FatherType / TextID / Desc`，根 55555（全部） |

黑金运行时读本目录（`dump_config.xml` 的道具转储清单即
`ini/Item/Equipment.xml`、`ItemUse.xml`、`Material.xml`、`Weapon.xml`…）。
`go2 Res/ini` 与 `testdata/ini` 原本都没有 Item 目录——九阴线只有设计态
`share/item/*.ini` 随客户端发布，服务端 XML 须重建。

## 二、九阴数据源与构建

构建器：`tools/itemgen`（与技能/NPC 同款数据链：离线构建 → 服务端只读产物）。

### 输入

| 类 | 源（`-in`，默认 `NINEYIN_ITEM_SRC` > `E:\Game\9yin\…\share\item` > `F:\Game\Online\9yin\9yin-server-go2\Res\share\item`） | 节数 |
|---|---|---|
| 模板 | `tool_item.ini` | 76,186（数据 76,178） |
| 模板 | `equipment.ini` | 28,856 |
| 模板 | `tool_item_hw.ini` / `giftbox.ini` | 50 / 1 |
| 静态包 | `item_func_static` / `item_logical_static` / `itemartstatic` / `item_use_skill_static` / `item_learn_skill_static` / `item_task_static` / `item_ingredient_static` / `item_prop_modify_static` / `item_seed_static` | 9 档 |
| 类型树 | 客户端 `九阴资源\resources\lua64\share\itemtype_define.lua`（`-itemtype`，缺席则不产出） | 267 常量 |

九阴与黑金同构：行携 `script`（125 个逻辑类）、`ItemType`、`ViewID`（背包
四页）与静态包引用（`FuncPack`/`LogicPack`/`ArtPack`/`UseSkillPack`/
`LearnSkillPack`/`TaskPack`/`IngredientPack`/`PropModifyPack`/`SeedPack`）；
`[-1]/[0]` 表头与黑金已发布静态包里的 `ID="-1"/"0"` 两行是同一机制。跨档
无重复 ID，合计 105,085 件。

### 产物（`ini/Item`，22 文件 125,003 行，36MB；含静态包表头行 18）

| 产物 | path_config Type | 行数 | 归档规则（优先序，`mapping.go` kindOf） |
|---|---|---|---|
| `Equipment.xml` | Equipment | 21,697 | EquipType 非武器族，或 equipment.ini 的 Equipment/ShotWeapon/FacultyPaint/FacultyBook 行 |
| `Weapon.xml` | Weapon | 6,959 | EquipType ∈ weapon/shotweapon/innerweapon |
| `Material.xml` | material | 1,631 | 背包材料页（ViewID=3） |
| `ItemUse.xml` | ItemUse | 16,322 | 其余全部（ToolItem/CardItem/WuJiItem/LifeExpBook/Bag…） |
| `Rune.xml` | Rune | 187 | script=Enchase |
| `ItemMountUse.xml` | mount | 260 | script ∈ Mount/Sable/Saddle |
| `TaskItemUse.xml` | TaskItemUse | 7,608 | ViewID=4 且带 FuncPack |
| `TaskMaterial.xml` | Taskmaterial | 2,854 | ViewID=4 且无 FuncPack |
| `ItemBox.xml` | ItemBox（九阴新增） | 15,476 | script ∈ BoxItem/ChoiceBoxItem/CompensateItem/GCBoxItem/FurnitureBox/TreasureChest |
| `SkillBook.xml` | SkillBook（九阴新增） | 31,686 | script ∈ SkillBook/SkillPage/SkillScroll/SkillBookTag/SuperBook |
| `add_money.xml` | add_money | 4 | script=DropMoneyItem |
| `path_config.xml` / `ItemTypeList.xml` | — | 11 / 268 | 登记表 / 类型树 |

分档只影响文件归属：装载与使用链一律按行内 `script`/`FuncPack` 取值。

静态包 9 档（列序取 `[-1]` 列序头，表头两行照留）：

| 产物 | 引用键 | 行数 | 说明 |
|---|---|---|---|
| `item_func_static.xml` | FuncPack | 964 | **使用链核心**：`PrepareTime`/`PrepareAction` = 读条时长/动作；`LuaScript` = 官服物品脚本路径 |
| `item_logic_pack.xml` | LogicPack | 83 | 黑金同名档；列集是九阴词汇（BindType/CantDepot…） |
| `ItemArtStatic.xml` | ArtPack | 6,447 | 黑金同名档（图标/男女模型/动作集） |
| `item_use_skill_static.xml` | UseSkillPack | 2,907 | 药品=使用技能（SkillID/SkillLevel） |
| `item_learn_skill_static.xml` | LearnSkillPack | 3,445 | `TragetID`（官方拼写）= 学习目标 |
| `item_task_static.xml` / `item_ingredient_static.xml` / `item_prop_modify_static.xml` / `item_seed_static.xml` | TaskPack / IngredientPack / PropModifyPack / SeedPack | 1,630 / 3,533 / 854 / 159 | |

再生命令：

```
go run ./tools/itemgen -verify                         # 产物 → testdata/ini/Item
go run ./tools/itemgen -out <Res>/ini/Item -verify     # 部署根同步
```

| flag | 语义 | 缺省 |
| --- | --- | --- |
| `-in` | 设计态表根（含 tool_item.ini） | `NINEYIN_ITEM_SRC` > 本机部署路径 |
| `-heijin` | 黑金样本目录（列出处判定，只读） | `NINEYIN_HEIJIN_ITEM` > `F:\Game\Online\9yin\BGserverCJ\Res\ini\Item` |
| `-itemtype` | `itemtype_define.lua` | `NINEYIN_ITEMTYPE_DEFINE` > 本机客户端资源路径 |
| `-struct` / `-lua-define` | struct schema / 静态包列名（列出处判定） | 仓库 testdata / `-in` 上两级 lua/share |
| `-verify` | `config.LoadObjectFile` 回读行数/ID 序/首行属性 | 关 |

基线：22 文件 125,003 行，verify 全过；产物已投 `testdata/ini/Item` 与
`go2 Res/ini/Item` 两处。审计报告 `artifacts/local/itemgen-report.json`。

### 属性词汇

源表键名即九阴运行时词汇，缺省原样直通；每列出处入报告、不改输出——
客户端 lua 按源表键名取物品属性（`form_bag_func.lua`
`item:QueryProp("FuncPack")`），黑金词汇（Desc/Only/CantSell…）不得据其
改名。全产物去重后 450 个属性名：

| 出处 | 属性名数 | 例 |
|---|---|---|
| heijin（黑金样本行属性名） | 35 | script/ItemType/FuncPack/LogicPack/ArtPack/EquipType/MaxAmount/SellPrice1… |
| struct（`struct/item/*.xml`） | 5 | CoolEffectType/ExpiredTime/LockStatus/EffectNo/NeedJob |
| define（`item_static_data_define.lua`） | 27 | PrepareTime/PrepareAction/LuaScript/SkillID/CantMove… |
| rename（审定改名） | 9 | 见下 |
| native（仅九阴） | 374 | ViewID/IsMarketItem/DropID/LearnSkillPack/QTypeID… |

**审定改名一处**：`@记录键` → `<名>Rec`（已带 Rec 后缀的原名）——黑金
struct 的记录命名法（dice_member_rec/RaPropertyRec）。`@X` 多行以 `;`
连成一个属性。不直接去 `@`：`@ItemType`（宝石可镶嵌部位）会与标量
`ItemType=73` 撞名。涉及 AddFacultyRec/PropModifyPackRec/SkillModifyPackRec/
PointPosRec/PointPos2Rec/ItemTypeRec/ChoiceItemRec/CompensateItemRec/
DelayBindItemRec。

### 源表脏数据处理（全部入报告）

| 现象 | 处理 |
|---|---|
| 模板表 `[-1]`/`[0]` 表头 | 丢弃（黑金模板档无表头）；静态包照留 |
| `[//说明]` 注释节（tool_item 8 个） | 跳过 |
| 空节 401 个（运营备用位 `Operation_item_NNNN`/`Operation_equip_NNNN` 等） | 跳过 |
| 非法 XML 属性名键 1 个（`Name金蛇银丝发饰（图纸）`） | 丢弃 |
| 节内重复键 30 处 | 输出 `原名_2` |
| 静态包重复节 24 个（功能包 2 / 使用技能包 7 / 学习包 8 / 任务包 5 / 属性修正包 2；多数内容相同，`item_use_skill_static[5368]`、`item_prop_modify_static[3006]` 两处内容不同） | 行全保留；装载首行胜出 |

### 引用完整性与使用链底盘（报告 `refs` / `use_chain`）

| 引用键 | 引用数 | 包内无行 |
|---|---|---|
| FuncPack | 68,167 | 37 |
| LogicPack | 103,834 | 12（含错位值 `c`/`8012`） |
| ArtPack | 30,298 | 155（含错位值 `FALSE`） |
| UseSkillPack | 2,289 | 15 |
| LearnSkillPack | 31,100 | 0 |
| TaskPack / IngredientPack / PropModifyPack / SeedPack | 1,761 / 1,176 / 722 / 124 | 18 / 0 / 1 / 5 |

带 FuncPack（可使用）的物品 68,167 件，功能包有行 68,130：其中带
`LuaScript`（官服物品脚本，脚本本体未随资源发布）1,489 件、无读条（瞬时）
2,192 件。按逻辑类：SkillPage 24,785 / BoxItem 14,520 / ToolItem 13,142 /
SkillBook 6,100 / CardItem 1,116 / WuJiItem 1,108 / LifeExpBook 943 /
CompensateItem 742…；带 LearnSkillPack 31,100、UseSkillPack 2,289、
DropID 14,613、Capital 23。

未转换 74 项（报告 `skipped`，逐项给理由）：fuse_*（合成 7）、life_*（生活
14）、exchangeitem 族（兑换 5）、senior_/inscription_/propconv 族（精炼铭刻
16）、home_*（家园 2）、treasure 族（宝箱寻宝 7）、randomevent（2）、
huanpi_suits/fashion_suit_resource（换皮 2）、`itemlogicstatic.ini`（旧版
逻辑包）、两个空表、`innerequip/` 目录、其余 15 个外围配置。

## 三、官服 msg31 取证（tools/pcapitem）

```
go run ./tools/pcapitem "go2 资料\初始账号登陆-完整引导-女.pcapng" 8032 0.6 2.4 31
```

上行（0x0A 活动信封）恰 4 值 `[31, view, slot, count]`，view 是线缆视图号。

| # | t | 上行 | 应答 |
|---|---|---|---|
| 1 | 110.323 | `[2,5,1]` | 读条 1500ms `life_ss_01` → 学会 CS_jh_cqgf03（View40 idx 8）→ `0x19(2,5)` |
| 2~3 | 120.325 / 128.538 | `[2,5,1]` | 同形 → CS_jh_cqgf02（idx 9）/ CS_jh_cqgf05（idx 10） |
| 4 | 261.121 | `[2,14,1]` | → zs_default_01（idx 12，打坐） |
| 5~6 | 278.670 / 281.931 | `[2,14,1]` | 仅 `m1[2,"9402"]`（「正在打坐，不能执行该操作。」） |
| 7 | 285.882 | `[2,14,1]` | 暗器书：hw_normal_zl/dw/fd/fb/jk 五招（idx 13~17） |
| 8 | 368.637 | `[125,3,1]` | 任务道具：1000ms `interact020b` → 任务完成簇（`m24[player,40,"task_ok"]` 等） |
| 9 | 420.968 | `[2,14,1]` | 轻功书：读条中途推 MPRatio/MP，结束写 QingGongRec |
| 10 | 454.898 | `[2,14,1]` | 内功书：View43 ng_jh_001 + View45 经脉行 |
| 11~14 | 572.499 / 650.234 / 650.985 / 659.368 | `[2,14,1]`、`[2,6,1]`、`[2,7,1]` | **零应答**（移动中连点） |

受理帧序（#1 原始字节；玩家对象号官服 350000f3/2a100010、本仓 1/1）：

```
S2C 1e 04 00 02 19 00 00 00 02 dc 05 00 00 | 06 0d "flow_prepare" | 06 0d "useitem_flow"
S2C 10 01 02 00 00 00 05 00 00 00 01 00 72 07 01 00 00 00   视图对象(2,5) LockStatus=1
S2C 0x10 玩家 State="stand" → State="life_ss_01"
… PrepareTime …
S2C 1e 01 00 02 1a 00 00 00                               m26（#2 起带；#1 无）
S2C 0x10 玩家 State=""
S2C 10 01 02 00 00 00 05 00 00 00 01 00 72 07 00 00 00 00   视图对象 LockStatus=0
S2C 0x18 ViewAdd view=40 idx=8 …                           效果帧
S2C 0x10 视图对象(40,8) MaxLevel=3 ；CanUse=1 ；m141[技能,"Skill"]；
    m1[2,"9491",技能]（「你学会了{@0:某项技能}！」）；0x11 note_wuxue_rec 行；m98[860]；m371["skill_help",技能]
S2C 19 02 00 05 00                                         ViewRemove(2,5) 扣件
```

要点：读条时长/动作正是 `item_func_static[FuncPack]` 的值（书类 118 =
1500ms `life_ss_01`；菜肴 1269 = 3000ms `interact053`）；`m25` 有时走压缩
载体 `0xCC`（内容同）；官服背包行本身就带 `FuncPack/LearnSkillPack/LogicPack`
等静态包引用（#1 后奖励行的 0x18 十四属性）——印证「模板行 → 对象属性 →
下发」这条黑金格式路线。登录视图容量实测 `View40=800 / View43=512 /
View46=512`（本仓原按登录行数建容量，学会第 8 个武学会越界，已改）。

## 四、装载层（internal/itemdata）

`ini/Item` 36MB/12.5 万行不整表解 map：装载期逐文件扫描建「ConfigID →
(文件, 元素偏移, 长度)」索引后丢弃文件字节（常驻约 10MB），行属性取用时
按偏移回读并进有界缓存（4096 行）；不长期持有文件句柄，产物可在服务
运行期间重生成。静态包整表入内存。实测 104,684 行索引装载约 90ms。

取值面：`Catalog.Definition(id)` → `*Definition`（`Str/Int/Script/ItemType`）
+ 类型化联查 `FuncPack()` / `UseSkill()` / `LearnTargets()` / `Art()` /
`Logic()`；`Catalog.Lookup(id)` 给发放行用的 `Entry`（ItemType/ViewID/
MaxAmount/EquipType/ColorLevel/Photo/Name/Script）。分类页以行内 `ViewID`
为准，缺省/越界（11 = 内功装备页）时装备归装备页、其余道具页。

## 五、服务端接线（internal/fssoloworld）

- **物品目录**（`itemcatalog.go`）：`NINEYIN_ITEM_XML`（产物目录显式覆盖）
  > `NINEYIN_ITEM_DIR`（设计态 ini 显式覆盖，旧口径保留）> 数据根
  `ini/Item` > 仓库 `testdata/ini/Item` > 数据根 `share/item`（设计态回退，
  只有目录面）。旧实现只有设计态回退，且按**文件**归页（任务页物品被归到
  道具页）。
- **入包堆叠上限**（`grantShopItemUnits` / `stackOrAddBagItem` / `grantBagItem`）：
  上限取模板行 `MaxAmount`（未标不设限）；不可堆叠物逐件成行；先容量预检，
  放不下整单则一件不发（购销不扣款——避免部分入包不收费）；`gm item` 未
  指定页时落模板行自己的分类页；行携 `MaxAmount` 入档，行帧下发真值。
- **msg31 受理链**（`itemuse.go`）：上表帧序。**状态在受理时同步落定**
  （学会/入账/扣件/冷却行），读条定时器只补发尾段帧——roleState 的写只
  发生在收包协程，定时器不碰状态（与技能链 finishCast 同纪律）。
- 闸门（命中即零应答）：读条进行中、出招/锁定中、冷却中（行携
  `CoolDownCategory`/`CoolDownTime` 且冷却表该类别未过期）、无功能包、
  效果预检不成立。

## 六、效果面

| 效果 | 状态 | 说明 |
|---|---|---|
| 武学书（LearnSkillPack → 招式） | **已接** | 目标为 SkillNormal/SkillLock 且未学者逐个学会：View40 追加行 + MaxLevel + CanUse + m141 + m1[2,"9491",技能]。目标全已学或都不是招式 → 拒、不扣件 |
| 钱袋（CapitalItem，CapitalType=1 碎银） | **已接** | CapitalValue 入账 + 碎银帧；带冷却的写冷却行。CapitalType=2 未建模 → 拒 |
| 瞬时使用（功能包无 PrepareTime） | **已接** | 无读条流，效果帧即发 |
| 礼盒（BoxItem/CompensateItem，DropID → 掉落包） | 未接 | 官方掉落包体未随客户端发布；`share/drop/drops_mvp.json` 是参考端自制片，不当官方数据接 |
| 药品（UseSkillPack → 回血/回蓝技能） | 未接 | 依赖技能命中/效果结算 |
| 内功书 / 轻功书（View43 / View45 / QingGongRec） | 未接 | 官服 #9/#10 样本已取证，对应容器行生成未做 |
| 任务道具（任务完成簇） | 未接 | 官服 #8：`m24[player,40,"task_ok"]` + Task_Record 改格 |
| 带 LuaScript 的物品（1,489 件） | 未接 | 官服物品脚本本体缺失 |
| 武学书附带帧 | 未发 | `note_wuxue_rec` 来历行、`m98[860]`、`m371` 帮助窗（键规无数据源） |

## 七、未覆盖与开放项

- **未真机验证**：读条流/扣件帧与官服字节一致，但客户端对 `LockStatus`
  上锁、`State` 动作的实际表现只有抓包佐证；帮助窗三帧缺省的影响未知。
- 打坐中使用回 `m1[2,"9402"]` 的门未做（打坐态机未落地）；读条中移动/
  切图/施法是否打断读条未处理（状态已落定，尾段帧照发）。
- 武学书门槛（`UseCondition` → `rule/condition.ini` Type3 前置招式上限区间、
  Type1 门派、Type19 功能态）未门控。
- 部署：`go2 Res/ini/Item` 已同步，但 exe 未重编——需 build + 重启才走新链。

## 八、工具索引

| 工具 | 用途 |
|---|---|
| `tools/itemgen` | 设计态 ini → 黑金格式服务端 XML，审计报告 + `-verify` |
| `tools/pcapitem` | 抓包 msg31 取证：以每条 C2S 消息为锚的时间窗，按官服 0x09 属性表解视图行/属性名（`-time0/-time1` 任意窗） |

复跑：

```sh
go run ./tools/itemgen -verify
go run ./tools/pcapitem "<女号.pcapng>" 8032 0.6 2.4 31
go test ./tools/itemgen ./internal/itemdata ./internal/fssoloworld
```
