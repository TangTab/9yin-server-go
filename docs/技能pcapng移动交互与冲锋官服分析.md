# 技能.pcapng：移动交互方言与冲锋距离官服分析

> 抓包：`F:\Game\Online\9yin\9yin-server-go\artifacts\技能.pcapng`（官服 103.143.19.173:7072，
> 2026-11-09，1301s / 18156 帧 / 88 次 msg211 施法）。
> 用户剧本：鼠标点击移动×2 → 地图点击移动×2 → 技能寻路（定点怪跑完 / 移动怪丢目标 /
> 定点怪没跑完+自动选怪+技能×2 / 移动怪没丢目标跑完）→ 冲锋距离观察（3 米技 / 6 米技）。
> 分析产物：`~/.zcode/analysis/`（jn_full.txt 标注时间线、jn_annot.py / jn_rush_pair.py /
> jn_window.py / jn_skill_table.py、jn_c2s.bin / jn_s2c.bin）。

## 一、移动四方式的协议方言

C2S 移动统一走 **msg15**：`[id, state, x, y, z, destx, desty, speed, orient, clock]`
（state 7=走 / 4=停 / 5=跳；id 高 16 位 0xA000=寻路系——鼠标点击地面、地图点击、
技能接近**全用它**，0x3000/0x0000 才是键盘 WASD 手动，f.pcapng 拉回样本）。

| 方式 | 时间锚 | 特征帧 | 与其它方式的区分 |
|---|---|---|---|
| 鼠标点击地面 | 2.594s / 5.378s（5.9m/11m） | `0xA0000F01 state7` **双帧起跑**（同 dest 毫秒×2）→ 到达 `0xA0000000 state4` 单帧 | 无前导 |
| 地图点击 | 10.890s / 14.775s（10.6m/37.8m） | 同上，但起跑前 **~73ms 有 `msg266[4]`** | **msg266[4] 全场仅 2 次、正对两次地图移动；S2C 零回应** |
| 技能寻路自走 | 受理簇同毫秒（如 415.577） | state7 dest=目标 xz，速度 4.5 | 与受理簇（msg25/707[15]）同毫秒起跑 |
| 追移动怪 | 75.849→94.838 | state7 **持续重发、dest 每步更新为怪新位置**（五连改向） | 丢失目标 = **`msg431 obj(0,0)` 清锁**；服务端暂存态零取消、静默挂死（剧本重演 f.pcapng 34s 定论） |

辅助形态：跳跃 = `0xA0005000/0xA0004F00` 双帧各带 32B hex 校验串；出招位移 =
`0xA0001D01` 速度 20.00→0.69 两帧；msg54（349 次）集中在战斗期 = 选怪相关，与移动无关。

**msg220 在途上报只在距目标 ≤~6.2m 起报**（首报点 5.34 / 6.26 / 6.02 三证，
f.pcapng ~6.2 定论再证）→ **第一条 220 本身就是"接近完成"信号**，其后 ~100ms 间隔续报。

**S2C 0x21 ServerAllDest**：每实体 44B = `[obj u32][owner u32][x,y,z f32][δ f32][speed=4.5][orient][?]]`，
帧长 47/91/135/179/223 = 1~5 实体；含玩家自身条目（第二块 = `[msg15 id][obj][…]` 意图结构）。
0x20 全程 0 次（a.pcapng 定论再证）。

## 二、冲锋距离定论（用户观察 vs 帧）

c61 真身 = **`msg24 c1=61`**：`[obj, 61, destx, desty, destz, 档, 动画名]`，全场 20 个。

### 3 米技能（kfkj02/lyfh02/mzjf02/03/wuji_lyfh05，TargetShapePkg=4）

- 服务端 c61 触发实测 **6.09 / 6.10 / 5.91m**；明确不触发 6.33 / 6.36 → **阈值 ∈ (6.1, 6.33]**。
  GRushRange+Add=6.5 是配置上界，实测官服判定比 Go 现行 6.5 紧约一条上报。
- c61 dest = 目标身前 bind（实测距目标 1.74~1.75m ≈ BindDistance 1.54 + 朝向余量），y=玩家上报 y+1.0。
- 冲刺段长 ≈ 4.2m，~300ms，落地 212。
- **用户读数"跑到 4 米冲一下" = 触发瞬间的视觉剩余距离**：中心距 6.1 − 双方模型半径 ≈2m ≈ 4m，
  与冲刺段 4.2m 双重吻合。
- 档2（qg_fastruning_s_2h_01）出现于 wuji_lyfh05 两次 = 该技能个体动画配置，非距离分档
  （锁定冲锋族全为档1）。

### 6 米技能（hyjf02，TargetShapePkg=7）

- **13 次施法零 c61**（容错解码复核 0xCC 伪影后 20 个 c61 全属 3 米族 + hyjf07）。
- **walk-in 直收**：客户端走到 220 首报点（~6.2m）自己停 + 212：
  1210 案例 6.26 首报 → 212@5.81m；1268.6 案例 6.02 → 212@5.58m；267.3 案例 5.34 → 212@4.15m。
  服务端只在 212 后回落地簇（msg26 + CD + 337 + c1=40 + msg19）。
- **受理即直收**（按下已在射程内）：211 当毫秒
  `0x12×2 + msg284(len8) + 0x11 CD×2 + msg337 + c1=40 + msg19`——无受理簇（msg25/707）、
  无 212。样本：hyjf02 **6.95m**@157.793、kfkj02 1.19m@367.928、0.64~0.97m 多例
  → **按下即执行窗口 ≈ castRange+1**。
- 用户观察"射程阈值=最大释放距离"成立：走到 ~6m（castRange）直接出招，不冲锋。

### 范围型点地（hyjf07，RushRange=30，TargetType=1）

- ground 点 ≤6.5 → **立即 c61 档1**（qg_fastrun_2h_01 短冲，非档2）：
  1216.2 案例 ground 5.52m / +48ms；1275.4 案例 ground 3.96m / +42ms。
- b/e 样本（8.2m 远距）为受理即 c61 档2（fastruning_s）→ **档位按距离分档（近档1/远档2）**，
  非按形态分档。

### 三条执行路径（锁定型技能全景）

```
dist ≤ castRange+1            → 受理即直收：msg284 簇（当场扣 CD 出伤，无 212）
castRange+1 < dist, 且 castRange ≥ ~5.5（hyjf02 型）
                              → walk-in 直收：受理簇 → 客户端自走 → 220 首报~6.2 停 + 212 → 落地簇
castRange = 3（kfkj02 族）    → c61 冲锋链：受理簇 → 自走 → 220 跨 ~6.1 → c61+CD/MP 同刻结算
                                 → 冲刺 300ms → 212 → 落地簇 14 帧
```

### 0xCC 伪影警示

kfkj02 358.132 案例的 c61 被 0xCC 伪影吞帧：同毫秒出现 `0xCC len=76` 帧（c61 明文 len=64 +
解码器错位 12B 吻合），伴生簇（0x11+0x13×N+0x10）完整可见。**"无 c61"结论必须先排除 0xCC 区**：
0xCC 帧（1060 个）开头 `cc3b6dfa` = 明文 00000000 = FrameDecoder 相位错位产物，全部是伪影。

## 三、技能表坐标（F: 部署 resources/modern/share/skill/）

castRange = varprop(TargetShapePkg) → attack_targetshape[.].TargetShapePara2：

| 技能 | StaticData | LockType | BindDistance | castRange | RushRange |
|---|---|---|---|---|---|
| CS_hs_kfkj02 | 15256 | 2971 | 1.54 | 3 | 缺列 |
| CS_jz_lyfh02 | 5274 | 1579 | 缺行 | 3 | 缺列 |
| wuji_CS_jz_lyfh05 | 17307 | 3165 | 缺行 | 3 | 20 |
| CS_jh_mzjf02/03 | 10733/10734 | 2286/2287 | 1.5/1 | 3 | 缺列 |
| CS_jh_hyjf02 | 7565 | 1799 | 缺行 | 6 | 缺列 |
| CS_jh_hyjf07 | 7648 | — | — | — | 30（范围型） |

## 四、Go 差距清单与移植批次

### 第一批：移动 / 自动寻路（本文档落地批）

| # | 官服行为 | Go 现状 → 动作 |
|---|---|---|
| A1 | msg15 全形态带位置（state7/4/5 的 idx3..5 恒为 x,y,z），服务端收编（0x21 回显含玩家） | 仅冲锋在途时用 → **非在途也 updateMemberPosition**；orient 现取 values[6]（实为 destx）→ 按 state7 idx9 取 |
| A2 | msg220 带位置 | 同 A1 收编 |
| A3 | msg266[4] 地图寻路标记，S2C 零回应 | 不在白名单（逐条日志噪音）→ **白名单+静默消费** |
| A4 | c61 触发判定为 xz 平面距离（212 三分同口径） | tryRushArrival 用 3D（y 参与误判坡地）→ **改 2D** |
| — | msg431 obj(0,0) 清锁：S2C 回 0x10[417]，暂存态静默挂死 | selection.go 已对齐（416/417 双帧 + 30s 过期兜底优于官服），**不动** |
| — | msg209[15,obj,0] 点选授权 / msg707[15] 受理簇授权 | 已落地（selection.go / rush_cast.go），**不动** |

### 第二批：技能

| # | 官服行为 | Go 现状 → 动作 |
|---|---|---|
| B1 | 受理即执行窗口 ≈ castRange+1（hyjf02 6.95m 直收） | `max(bind+2, 3.5)` → **`max(bind+2, castRange+1, 3.5)`**（g 样本 3.06 直收/4.41 暂存仍相容） |
| B2 | 受理即直收簇 / walk-in 落地簇带 **msg284**（len8 空帧） | 未实现 → 补发 |
| B3 | 范围型近距点地（≤6.5）= c61 **档1** 短冲 | 受理即档2 → 近距改档1 |
| B4 | hyjf02 212@5.58~5.81m 官服认可为走带落地 | 212 走带认可半径=5.0 → **`max(5.0, castRange+1)`** |
| — | c61 触发上界实测 (6.1,6.33] | 维持 6.5（GRushRange+Add 配置语义），注释记录实测带 |

## 五、复跑方法

```bash
python -I ~/.zcode/analysis/pcapng-split.py 技能.pcapng jn_c2s.bin jn_s2c.bin 7072
python -I ~/.zcode/analysis/jn_annot.py   技能.pcapng 7072 > jn_full.txt   # 标注时间线
python -I ~/.zcode/analysis/jn_rush_pair.py                              # 88 次施法配对表
python -I ~/.zcode/analysis/jn_window.py 415.4 417.5                     # 逐帧窗口（grep 需 -a）
```
