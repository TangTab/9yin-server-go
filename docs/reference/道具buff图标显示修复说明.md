# 道具 buff 图标显示修复说明（纯服务端，客户端已还原）

## 背景
使用道具（内修丹/属性丹/历练玉液等）后右上角不出现 buff 图标。此前怀疑客户端
`buff_static.ini` 的 `Visible=0` 并打过 share.package 补丁——**该判读是错的**，
按你的要求客户端已从 `share.package.bak-20260909` 完整还原，不再替换客户端。

## 真正根因
- 右上角 buff 栏是原生 `form_main_buff`（客户端 Lua 只是壳），数据来源是服务端下发的
  `BufferListStr`（格式 `ConfigID,UniqueID,Level,Owner-Serial,EndTime,EffectType,OverlayRows|`）。
- 官方原版数据里道具 buff 的 `Visible` 其实全是 **1**（如 buf_shuxing_all_01=10965、
  itmbuf_neixiu_07=16334），但图标仍不显示。
- 真正的门禁是 `buff_static.ini` 的 **BelongCategory 列**：
  - `Belong=1`（内修丹系列/历练/宴席/全属性丹/佳肴等"生活增益"）→ 主 buff 栏**不渲染**；
  - `Belong=0/2/7/11…` → 正常渲染（站姿/技能 buff=2，月饼/天机珠/菊花酒/VIP buff=0/7）。
  - 对照实证：同族 `buf_shuxing_tp_01`（体魄丹，无 Belong=0）显示，`buf_shuxing_all_01`（全属性，Belong=1）不显示。
- 官服设计中这些 Belong=1 增益本来就不上主栏（心魔/宴席/内修各有专属 UI）。

## 修复方案：服务端显示别名
`item_buff.go` 新增 `itemBuffDisplayAliasRules` + `itemBuffHiddenStatics`：
catalog 构建时，凡真实 buff 静态行 Belong=1，就把它替换成**官方可见、效果描述最接近**
的 buff 作为图标载体：

| 隐藏 buff | 显示别名（图标+悬停文本） |
|---|---|
| 内修丹系列 / 野味 / 美酒 / 茱萸果 / 宴席 | `itmbuf_vipbuf_01`（内修效率） |
| 玉液 / 茱萸茶 / 野味血（历练类） | `buf_20200902sxjd_fc_001`（历练转换/内修速度） |
| `buf_shuxing_all_01` 全属性丹 | `buf_shuxing_tp_01`（可见同族属性丹） |

- 本轮别名共命中 **66 条**映射（启动日志 `displayAliases=66` 可核对）。
- **计时不受影响**：时长仍按每个道具 `desc2.idres` 描述（持续60分钟/最多延长至20小时等）。
- 折衷说明：图标和悬停文本来自别名 buff（官方数据里不存在"同效果且可见"的孪生 buff，
  已全量扫描属性段与效果包确认）。若观感不满意可换别名池（Mvp_huashan_001、
  buf_add_jhndf_1、buf_yaopinhp 药丸系列等）。
- 已知小坑：悬停 tooltip 键是 `desc_<buff>_1`，部分 buff 该键在官方文本表里是空串，
  悬停可能空白，图标不受影响。

## 验证方法
1. 重启服务端（H:\9yin-go-server2\9yin-game-native-menu.exe，04:07 版）+ 重启客户端。
2. 吃 `itm_neixiu_07`（精致月凝内修丹）或 `shuxing_all_01`（属性丹）→ 右上角应出现图标并倒计时。
3. 对照组：天机珠（`item_qy_tjz_*`，本身 Belong=7，无论有无本修复都应能显示）。
4. 若仍不显示 → 说明 Belong 过滤假设有偏差，需抓帧继续查（此时别名无副作用，行为与旧版一致）。

## 改动文件
- `cmd/protocol-probe/item_buff.go`：别名规则 + Belong=1 判定 + catalog 替换逻辑。
- `cmd/protocol-probe/item_buff_test.go`：断言更新为别名结果（TestItemBuff 全部通过）。
- `C:\游戏蜗牛\9yinjh\res\share.package`：已还原为官方原版（备份保留）。
