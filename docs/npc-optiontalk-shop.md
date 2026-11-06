# NPC 对话交互与商店 View61 —— 四期「0x07 对象选择/菜单/商店」实现记录

> 2026-11-04。分支 `9yin-main-new`。数据真值源 `E:\Workspace\project\game\9yin_server1`
> （optiontalk 三件套 + shop.ini + tool_item.ini），wire 真值源 F 服快活岛开商店抓包
> （`tools/smoke/shop_probe_frames.txt` 41396-41450 段）与 Go 源 messages.go:121-193。
> 测试：`NpcDialogFlowTest` 7/7（msg70 购买链真 MySQL）+ `ShopCatalogTest` 5/5。

## 1. 交互总模型（两段式）

```
玩家点击 NPC                    菜单项选择
C2S 0x07 (33B)                  C2S 0x07 (33B)
FuncID@29 == 0                  FuncID@29 == mark（菜单项回发）
        │                              │
        ▼                              ▼
combat NPC? ──是──> selectCombatTarget   dialog.take(mark) 消费即删
        │否                             │
        ▼                              ├── NextPage(id) ──> 下页 0x1C
0x10 LastObject(idx414)                 ├── ChooseOption(shop) ──> 0x1D + 开店帧组
optiontalk_npc 命中?                    ├── ChooseOption(giveitem) ──> 0x1D + 入包
        │是──> 0x1C SERVER_MENU         ├── ChooseOption(task/trans…) ──> 0x1D（日志放行）
        │否──> 放行（记日志）            └── Leave / 未知 ──> 0x1D
```

- 帧布局：`0x1C = op | u32 obj | u32 owner | u16 count | 项(u8 type, u16 mark,
  u32 UTF16 字节长含 NUL, UTF16LE 内容含 NUL)`（Go messages.go:121-193，与
  opcodes.json 0x1C fixed reads 一致）；`0x1D` 整帧 1 字节。
- **mark 是服务端本地路由号**：`NpcDialog` 每次开对话从 1 顺排分配，翻页续排
  不复位（protocol-notes §12：不复用旧 FuncID 数字空间）；`take()` 消费即删防重放。
- 菜单项文本传 **string key**（客户端字串表解析）：选项 key 来自 dialog_pack
  Option 槽 2（如 `biaoti00007`=听说这里出售李白精选诗集？给我来本瞧瞧。→
  desc.idres），「继续」项 = 下页节点 MenuID（`qy_gcd_01_005_1`=是叫我么？），
  「告辞」= `ui_form_talk_leave`（interface.idres，F 服抓包同款）。

## 2. 数据链（全部实测）

```
creator XML id 属性（= NpcSpawn.resolved.configId）
  → share/rule/optiontalk_npc.ini   节名→DialogID（3623 条）
  → share/rule/optiontalk_dialog_pack.ini  节点 {MenuID/TitleID/NullTitleID/
      NextPage=63,64(逗号多值)/Option=描述,描述}
      描述 = 9 槽 `类型|文本key|目标|条件|karma|…`（`|` 切分，尾部空槽）
  → share/rule/optiontalk_operater_type.ini  类型名→枚举（task=1 shop=2 trans=3
      … giveitem=10；本期仅日志参考）
```

- 剧情节点（仅 NextPage）→ 菜单 = 每链一项「继续」；选项节点 → 选项 + 尾部「告辞」。
- giveitem 目标 `Box_qyjl_gcd_009*1` = itemId*count。
- 实测样例链：born04 千灯镇 `FuncNpc00009`(钱渊来) → 69 →
  `shop|biaoti00007|Shop_special_002`；`WorldNpc00002` → 62 → 63/64 双链。
- optiontalk_operate.ini（Menu→Operate 命名菜单）是**并行入口不走**：其引用的
  商店号（Shop_00302 等）在 shop.ini 无对应节（死引用）。
- school09（默认出生场景）实测：Transschool09A→5339（createtrans）、yhg_001..004
  → 各自剧情链；可达子图 37 节点 **无 shop 项**——商店开链真机验证需 born04。

## 3. 商店 View61

- 客户端 **不读 shop.ini**（坑 24），全靠服务端 0x15/0x18 物化
  （form_shop.lua createview 回调直读 101-111）。
- 开店帧序（对齐 F 服抓包）：`0x16 删旧(61)` → `0x15 建(61, cap=100, 头四属性)` →
  逐行 `0x18(61, row 1..N, 行六属性)`。

| 属性 | 号 | 值来源 |
|---|---|---|
| ShopID | 101 | shop.ini 节名原串 |
| PageNum | 102 | 恒 10（抓包常数=行/页；shopconfig.ini `PageNum=500` 是另一口径） |
| PageCount | 103 | PageInfo 项数 |
| ShopType | 104 | 恒 0（**抓包为 0，ini Type=1 不一致，以抓包为准**） |
| ConfigID | 105 | 商品行 f0 |
| Amount | 106 | f1 |
| SellPrice0/1 | 107/108 | 恒 0（金/官银价，抓包口径） |
| SellPrice2 | 109 | f3（碎银价；form_shop.lua：SellPrice0=金 1=银 2=卡/碎银） |
| MaxAmount | 110 | 物品自身堆叠上限（tool_item.ini MaxAmount，缺省 99；非 shop.ini 字段） |

- shop.ini 商品行**键不可信**：`shop_khd_ywb` 节内九行全是 `0=`（§12 坑），
  别的节又 0,1,2 顺排——`ShopCatalog` 一律按文件出现顺序收集（View61 行号=顺序位+1，
  与抓包 9 行一致）。f2/f4-f8 语义未证（绑定/标志/排序/条件族）**不下发**。

## 4. 购买（C2S msg70）

- 参数序 **form_shop.lua buy_shop_item 实证**：`custom_buy_item(shop_id, page, pos, count)`
  = `[70, str, int, int, int]`；`page`=grid.Parent.curpage、`pos`=格位+1、
  行号映射 `row = page*10 + pos`。
- 链路：会话开店态校验（shopId 匹配 + 行号在售）→ 碎银（CapitalType1）余额校验 →
  `ToolItemCatalog.lookup` → `inventoryStore.insertBagItem`（复用 GM 发放链）→
  `pushBagItemFrames`（0x19 摘 + 0x18 落成对帧）→ `setSilver + silverUpdate` 单帧。
- msg345 获得物品提示**未发**：抓包参数 `[345,2,9645,str,2,1]` 语义未证且词典无此 ID，
  少发优于错发（真机补拍后再补）。

## 5. 代码地图

```
world/npc/OptionTalkCatalog.java   optiontalk 三件套加载（GBK，ResourceCache 常驻）
world/npc/ShopCatalog.java         shop.ini 逐行扫描（重复"0"键安全）
world/npc/NpcDialog.java           会话对话路由态（mark 分配/消费/防重放 + 开店态）
protocol/NpcMessages.java          serverMenu(0x1C)/clearMenu(0x1D)
protocol/SceneMessages.java        deleteView(0x16，3B layout_proven)
game/login/LoginFlow.java          onObjectRequest 两段式 / openNpcDialog / sendDialogMenu
                                   dispatchMenuMark / openShop / handleBuyItem(case 70)
tools/smoke/npc_talk_probe.py      活体冒烟（场景无关：数据侧 BFS shop 链走位到开店）
```

## 5.1 活体冒烟口径（npc_talk_probe，10/10）

- 登录用 golden 帧（`verify_move_item.LOGIN_PLAIN` 同款——smoke_client 的极简
  0x02 包过不了密码结构校验；选角 0x04 需 0x54 填充帧）。
- 场景无关：初始视口无对话 NPC 时读 0x0B 场景资源 → creator 目录取对话 NPC 坐标
  → 0x0A motion 走位触发视口物化；优先选 optiontalk 图 BFS 可达 shop 的 NPC。
- 菜单文本 key 与数据侧同源比对（服务端下发即数据 key，不钉死）；
  已验链：city05 FuncNpc00606 → qy_menu_cwqy_monkey_04 → Shop_cwqy_monkeyfood
  （0x16+0x15(61)+0x18 开店帧组 + mark 防重放 + 告辞/未知 mark 0x1D）。
- ⚠ **golden 账号即 role 618**：probe 只点数据侧已知的文本 key（shop 链/告辞），
  盲点首个 mark 曾命中 giveitem 真发 2 个 Box_qycdu_009 进 618 背包
  （2026-11-04 手工 DELETE slot 18/19 复位 35 件）——giveitem 是真发放，勿盲点。

## 6. 真机待验清单

1. **0x1C 渲染**：帧从未在抓包出现（F 服走 custom int 4..8 路由）；若真机不弹菜单，
   备选方案=custom 4..8 序列（shop_probe 41396 段），0x1C 构造器保留。
2. 菜单项 type 字节统一 0 的渲染效果（语义未证）。
3. ShopType 恒 0 / PageNum=10 与商店 UI 分页的一致性。
4. 非 combat 点击 LastObject 用 idx414（F 服抓包是场景域 idx100，两者皆在协商表）。
5. born04 FuncNpc00009 开店+购买全链（含 msg70 参数序真机回验、msg345 提示补拍）。
6. shop.ini f2/f4-f8 语义（拿不准不发；真机对照后补齐）。
