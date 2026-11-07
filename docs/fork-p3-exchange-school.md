# fork P3 第一批：exchange 商店闭环（msg70 type-3）+ 新手选择（972）+ 门派（973/997）

> 2026-11-07 落地。证据：fork 二进制 `go tool nm/objdump` 行号级反汇编
> （`C:\Users\lzy\AppData\Local\Temp\fork_p3\p3.txt` 等）+ 磁盘资源实证
> （shop.ini / exchangeitem.ini）+ lua 字典（Java 侧四消息全未实现，
> 无参考可抄）。前情：docs/fork-p2-shop-title-jingmai.md。

## 一、msg70 capitalType=3 兑换计价

**缺陷修复**：shop.ini 47818 行商品约 88% 是 capitalType=3（4.2 万行），
此前 `openShopLocked` 遇 type-3 直接报错——客户端拿到**半截商店**，
70 购买也静默拒。

### shop.ini 第 8/9 列（此前未解析）

```
0=signup_schoolpose_12,1,2,100000,0,1,0,13094,543,,
                                │        │    └ col9 conditionID（语义未证，仅留存）
                                │        └ col8 exchangeID = exchangeitem.ini 节名
                                └ col7
```

fork 9/14 的货位匹配内部用 (page-1)*500+position+1 的行坐标（sbu.txt
:124 `IMUL $0x1f4`=500），本服维持与自家 View61 物化一致的 page+1/pos-1
匹配（10/页，Java 同款，实机已验证）。

### exchangeitem.ini（share/item/，4.5 万行 ≈ 10571 节）

节名 = 十进制 exchangeID，键：

| 键 | 样本 | 语义 |
|---|---|---|
| Item | `item_honor_school13,100;stuff_jiebang_01,1` | 按物扣费（分号列表 configID,count） |
| Prop | `CapitalType2,100000` | 按币扣费（CapitalTypeN,数量；可分号多币） |
| Condition/Condition2 | `120958` / `109035` | 任务条件门（本轮不评估仅记日志） |

### View61 type-3 物化

fork 9/14 openShop 的 capitalType switch 支持 0..3（scene_lifecycle.go
:1385）。本服保持已验证的 6 属性行格式，type-3 三价全 0 物化（Java
Shop_menpai_xtc_01 兑换行同为零价形态，客户端兼容）；>3 仍报错。

### 购买链（fork shop_buy.go:148 case 3 → exchangeCatalogFor）

校验全部通过才动账：条件门记日志不拦截（fork "conditions not
evaluated"）→ **Prop 只认服务端跟踪货币** CapitalType0/1/2（其余
"uses untracked currency (refused)"，如 SchoolDanceTotalScore）→
Item 数量 `countBagConfig`（按 configID 全包求和）→ 包容量预检
（可叠堆或空位，杜绝扣完发不出的回滚窗口）→ `consumeBagConfig`
跨行扣物（逐行 consumeBagItemCount，DB 写穿+帧同步）→ 扣币 →
发放。与 fork 的差异：fork 无容量预检（靠失败后日志），本服预检后
仍失败的极端竞态退币并记错误日志。

## 二、msg972 新手选择（handleFreshManChoice，freshman_guide.go:31-47）

反汇编钉死（CMOV 标签选择，串长 17/20）：

```go
if len(values) < 2 { return false, nil }
choice, ok := values[1].exactInt32()   // 非整数也 return false, nil
label := "新手(走引导)"; if choice == 5 { label = "老手(跳过引导)" }
log.Printf("%s: fresh-man choice=%d (%s)", remote, choice, label)
return true, nil                        // 纯日志：无帧、无状态写入
```

引导流程由客户端本地驱动（FinishFreshmanRecStr@225 归客户端管理）。

## 三、msg973 / msg997 门派（school.go:64-99）

两 handler 收敛到同一核心：

```go
joinFaction(id) = lookupFactionByID(id) + player.setFaction(faction)
```

- **973 GUIDE_ADD_SCHOOL** `[973, str schoolKey]`——仅窄串（tag6，
  fork :72 `CMPB $0x6` 单一形态，不收宽串）；玩家未生成记
  "guide add school before player spawn"。
- **997 SCHOOL**：遍历 values[] 取**首个窄串**作为 schoolKey（fork
  :91-93 值扫描循环；lua 侧 Request_Declare_Fight 等多变体共用入口）；
  无玩家返回错误 "school message before player spawn"。
- **lookupFactionByID**（school.go:32-43）：`strings.TrimSpace` → 空 id
  `"empty school id."` → 对 faction **全 ID**（school_shaolin / force_*
  / newschool_*，即 gmFactionPresets 值域）`strings.EqualFold` 匹配 →
  未命中 `"unknown school id %q"`。
- **setFaction** 帧：School@189 / Force / NewSchool 三属性单帧 0x10
  （factionProperties 前缀分流，本仓库既有实现）。
- **会话级不落库**：fork 二进制无 "persist faction" 字符串证据——
  C2S 加入门派重登即回落（门派跨登录存续走 GM set_faction / 角色外观
  appearance[8]），本服对齐 fork。

## 四、文件与测试

新增/修改：

- `exchange_catalog.go`（新）：exchangeitem.ini 解析 + 按路径懒加载缓存
  （`resetExchangeCatalogCache` 测试辅助）。
- `shop_catalog.go`：`shopCatalogItem` 增 exchangeID/conditionID（第
  8/9 列，截尾空列保持 0）。
- `scene_lifecycle.go` openShopLocked：case 3 零价物化（不再报错）。
- `shop_buy.go`：`shopBuyExchange` + `exchangePropBudget`；`item_wear.go`
  增 `countBagConfig`/`consumeBagConfig`（跨行扣物）。
- `freshman_guide.go`（新）972；`school.go`（新）973/997 +
  `lookupFactionByID`/`joinFaction`；`player_actor.go` 增 `factionValue`。
- `main.go`：白名单 if + 两 switch 增 972/973/997（msg70 复用现有 case）。

测试 11 项全绿：目录解析（条目形态/坏行拒绝/真实 4.5 万行冒烟 +
13093/13094 抽检）、shop.ini 第 8/9 列解析、openShop type-3 物化
（0x15+2×0x18 无错）、跨行扣物（slot 清空+余量重推+叠堆、无货币帧）、
Prop 银卡扣费（418 帧序）、四类拒绝（数量不足/untracked 属性/余额
不足/条目缺失）、972 标签与宽容解析、973 EqualFold+TrimSpace+未知拒、
997 首窄串扫描+无玩家错误。全量 `go test ./...` 零回归。

## 五、遗留（后续批次）

861 淬火、933-938 重铸（注意：lua 字典 933=LOCK/934=UNLOCK/935=RANDPROP/
938=SUREPROP，与清单文档「preview/lock/unlock/confirm」顺序有出入，
复刻时以 objdump 为准）、864 生活职业（toolbox 背包扩展）、997 完整
门派操作族（宣战/投票/回山门）、Condition 条件门评估、msg64/79 官方
兑换专用通道、GM 称号指令、fork 的门派出生点传送表。
