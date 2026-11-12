# 技能1.pcapng 多套路冲锋官服方言（c61 授予协议定版，2026-11-13）

> 数据：`F:\Game\Online\9yin\9yin-server-go\artifacts\技能1.pcapng`（49.4.14.150:7074 官服，
> 1031s/8116 事件/25 次施法；技能2.pcapng 为同连接续段，事件重复无新内容）。
> 套路覆盖：jh_hyjf02（6 米剑）、hs_hsjf02（3 米剑）、jz_xytf02/07（+`_sky` 空中变体，腿法空手）、
> hs_kfkj02（华山·灭剑如虹）、jh_mzjf02/03（剑）、hs_kfkj05 / jz_xytf01(_sky)（范围技）。
> 工具：`tools/rush_gap/`（本文绝对秒数以 pcap_frames 全流锚为准）＋tmp/rush1/cast_context.py。
> 本文修正《冲锋自走触发链官服定版与差集》P0 系列两处关键结论，见 §6。

---

## 一、核心定案：c61（m24 c1=61）是**服务端指令**，按 m220 授予

### 1.1 方向判定（非回显）

每次 c61 与其后首条客户端疾跑上报 m15 state=4（intent `0xA0000700`）的时序：

| 案例 | c61 | m15 state4 | Δ |
|---|---|---|---|
| hyjf02@160.223 | 160.2233 | 160.300 | +77ms（c61 先） |
| hsjf02@185.818 | 185.818 | 185.829 | +11ms（先） |
| hsjf02@199.763 | 199.763 | 199.781 | +18ms（先） |
| xytf02@642.633 | 642.633 | 642.634 | +1ms（先） |
| mzjf02@797.329 | 797.329 | 797.347 | +18ms（先） |

**c61 全部先于客户端疾跑态上报 1~77ms 到达**。若为回显，m15 必须先于 c61 ≥1×RTT
（本流 220→c61 反应延迟 24~132ms），@642.633 的 1ms 间隔物理上排除回显。
→ c61 是服务端下发的冲刺指令，客户端执行（进入 state4 疾跑、途中 212 落地）。

### 1.2 授予触发：m220 进技能冲刺窗，官服 +30~90ms 应答

全流 41 条 m220（五值 `[x,y,z,orient]` 统一、无请求标志位）与 16 次 c61 的配对：

- **窗内 220 → 授予**：220 后 +24~132ms 发 c61（例：hsjf02 220@185.729→c61@185.818 +89ms；
  xytf02 220@642.600→c61@642.633 +33ms；kfkj02 220@749.587→c61@749.620 同毫秒批）。
- **窗外 220 → 零应答**：mzjf02 220@794.734（13.8m，超窗）无 c61，客户端继续走；
  hyjf02 220@6.1m/7.3m（#0/#1）无 c61 → 客户端走入直收（212@5.7/5.6）。
- **在途重按不需再受理**：#15 接近进行中重按 xytf07_sky（220@675.919 先于 211@675.950），
  官服零受理簇、直接 220 授予 c61@675.981 + m707[**2**]。

观测窗口（220 上报时距目标，per-skill）：

| 技能 | 授予窗观测（有 c61） | 窗外观测（无 c61） | 212 落地带 |
|---|---|---|---|
| hyjf02（6 米） | 8.1 | 6.1、7.3 | 5.0~5.7（直收亦此带） |
| hsjf02（3 米） | 6.3、6.4×2、6.5 | — | 2.2~2.6 |
| xytf02 | 6.2 | — | 2.5 |
| xytf07_sky | 9.0、10.2、10.7 | — | 1.9 |
| xytf02_sky | 12.9、14.0 | — | 1.8~2.7 |
| kfkj02 | 10.4 | — | 2.1 |
| mzjf02 | 6.3、7.4 | 13.8 | 2.2、4.4 |
| mzjf03 | 6.4、13.2 | — | 0.9、2.3 |

旧结论**作废**：「hyjf02 六米技 13 施法零 c61」「c61 只属 3 米技窗口 (6.1,6.33]」
（技能.pcapng 样本恰逢窗外；本包 19.1m 按压→220@8.1m 即授予）。
窗口疑似挂技能/兵器配置（登录物理档案 GRushRange=5.0/ARushRange=8.0/WRushRange=8.0、
AirRush(15/12)、LandRush(25/6.5) 与 hyjf02≈8、xytf02_sky≈14 的量级吻合），逐技表待考。

### 1.3 c61 帧字节布局（0x1E 信封）

```
[1e hdr][msgId=24][obj][owner][c1=61][float32 x][float32 y][float32 z][int32 flag][NUL串 anim]
例@160.223: 24, obj(0x1102278c,0x6629d9), 61, (624.669,39.594,270.049), 1, "qg_fastrun_2h_01"
例@199.763: 24, obj同上, 61, (638.798,40.905,257.567), 2, "qg_fastruning_s_2h_01"
```

- **dest=落点前方过冲点**：距目标 ≈1.5~2.3m，比实际 212 落点再向目标多 ~1m（客户端自停于射程带）。
- **两形态**（选择规则未全定案，hsjf02#5 一例跨系）：
  - `flag=1 + qg_fastrun_{0h,2h}_01`——剑系受理簇路径（hyjf02/hsjf02×3/mzjf02/mzjf03）
  - `flag=2 + qg_fastruning_s_{0h,2h}_01`——**Java 方言同款**；空手腿法 xytf 全系、华山灭剑如虹
    kfkj02、hsjf02#5（按压前玩家恰在自动疾跑 state4 中）
  - `0h`=空手（腿法 xytf 全系）、`2h`=持械；与技能武器绑定。
- **伴随帧**：CantMove=1、CantParry=1、MPRatio/MP（授予即扣 MP）；
  **CD 行也在授予时刻写**（0x11[109] 加行+0x13 写格，见 §4）。

### 1.4 冲刺段与落地簇（授予→执行全链，#5 hsjf02 样本）

```
+0.000  m211 + m15 state7×2（同毫秒武装，按下即走）
+0.033  受理簇（§2）
+0.117  m220（6.3m）
+0.149  c61[dest,2,qg_fastruning_s_2h_01] + CantMove=1/CantParry=1 + MPRatio/MP   ← 授予
+0.167  m15 state=4（客户端疾跑态，0xA0000700）
+0.485  m212（落地 2.6m）                       ← 冲刺 318ms / 3.7m（全程 300~670ms / 2.5~7.3m）
+0.518  落地簇：CantMove=0/CantParry=0 → 0x1F → PauseSkill="" → m26 →
        NPC.LogicState=1 → 玩家.LogicState=1 → 0x1F → m337 → m24动作开始 → m19 →
        CantMove=1/CantParry=1（施法动画再锁）
+0.948  伤害飘字 + 击退 + m205 伤害目标列表
+2.414  收招：CantMove=0/CantParry=0 + m914[pos,12,n]
```

sky 系（#9/#12/#15）为空中冲刺：跳跃中（手动 state6/0x2903）发 220、下落中授予、
212 落地带落差可达 4.2m、平距 5.7~7.1m；授予后另发 m707[29]。

---

## 二、受理簇方言修正（对技能.pcapng 版本）

### 2.1 定靶技首按簇（本包 12 例统一）

```
[0x12:109 惰性清过期 CD 行 ×0~2] → CurSkillID=skill → CurSkillTarget=None →
m25[30000,"skill_pause"] → PauseSkill=skill → m707[15]
```

- **CurSkillTarget 全场 48/48 恒 None**——即使 m431 已选靶、m211 十四值带目标三元组，
  官服也从不把目标写入 CurSkillTarget。技能.pcapng「簇帧 CurSkillTarget 带目标 id」
  与本包不一致（两方言并存或旧读有误，落码以恒 None 为准、待第三包仲裁）。
- `_sky` 变体簇内 CurSkillID 后加 `CurIsAirSkill=2`；兵器系（kfkj/mzjf）=4；清除/换技=0
  （语义：0=无/地面清、2=空中技、4=持械系，仅 scene2 角色观测到）。
- 怒招受理即扣 FP：FP=0 + m586/m645/m646/m750 状态族（HP/FP 广播帧，全场 78 次非簇专属）
  + HP/MP 刷新（#1 受理窗实证）。

### 2.2 直收簇（点按已在射程内，#4 hsjf02 @1.8m）

```
CurSkillID → CurSkillTarget=None → m284[]（空值标记帧） → CD 行 → MPRatio/MP →
m24动作开始 → m19 → CantMove=1 / CantParry=**2**（非 1）
```

### 2.3 范围技（无目标 10 值 211：kfkj05 / xytf01_sky）

```
CurSkillID(+CurIsAirSkill) → MPRatio/MP → m20 范围施放标记[技能基名] →（无 m25/m707/PauseSkill）
… 无事发生 ~0.6~1.4s 后 CurSkillID=""（超时清）
```

### 2.4 重按方言（两条路径，按时机分流）

- **受理后即时重按**（#24，243ms）：`CurSkillID="" → m26 → PauseSkill="" →` 再发完整受理簇
  （清旧 pending 再受理，共 7 帧）。
- **接近在途重按**（#15）：零簇，直接 220→c61 授予 + m707[2]。
  → P0e「rearm 每次 211 都回 12 帧大簇」与官服两路径均不符。

### 2.5 m707 子码表（本包全集）

| 子码 | 次数 | 语义 |
|---|---|---|
| 15 | 20 | 受理确认 |
| 2 | 1 | 在途重按/接近期 211 的应答 |
| 28 | 9 | 落地后残留 220 心跳的逐条应答 |
| 29 | 3 | 空中/flag2 授予后的应答（#9/#12/#17） |

---

## 三、距离模型四桶扩点

- 直收桶不变（点按即直收簇；CantParry=2 细节新增）。
- 走入直收桶：hyjf02 220@6.1/7.3 无授予 → 客户端走进 5.6~5.7 落地（窗外下沿路径）。
- **授予冲刺桶**：新增 per-skill 窗口（§1.2 表）——首包确认 6 米技也有 c61、
  sky 技窗可达 14m、mzjf03 窗 ≥13.2m。
- 落地带 per-skill：3 米技 2.2~2.6 / 6 米技 5.0~5.7 / 腿法系 1.8~2.7 / mzjf 0.9~4.4。

---

## 四、CD 行时机（0x11/0x12/0x13 × record 109）

- **dash 路径在授予时刻写行**（c61 同批）；**直收路径在直收簇写行**；换技 0x12 惰性清过期行
  （受理簇前缀）。即 CD 行=技能实际开始执行的时刻。
- 行内容：个人行 `[skillId(CoolDownCategory), start, start+CDms/10, 65535]`
  （hyjf02=8462/6s、hsjf02=14655/6s、xytf07 8s 等）＋ 行3 施法窗格 `[start, start+1000]`（1s）
  ＋ 公共行 `[50000, start, start+7000, X]`。纪元=FILETIME 同 CD 帧族（旧结论维持）。

---

## 五、伤害段补充

- m19 锁定伤害段技能名=**基名**（xytf07_sky → "CS_jz_xytf07"；m20 范围标记同）。
- 双段伤害（397/283、509/142、651）+ 逐段 m205 伤害目标列表 + 击退 + 怒气段[2]
  + m169 buf（buf_CS_jh_hyjf02 / buf_ng_jz_006_01 族）。
- LastNGAttackNpc（内功系 hyjf02/xytf07）/ LastWGAttackNpc（外功系 kfkj02）按系分流——P4 维持证实。

---

## 六、对 P0/P2 系列结论的修正与行动项

1. **P0「官服接近段零帧」作废**：官服在 220 进窗时发 c61+CantMove/CantParry+MP+CD 行
   （技能.pcapng 32.642 案例恰是窗外 walk-in 路径，零帧成立仅限该桶）。
2. **P0c「客户端零消费 m24」归因重开**：我们/Java 发的 `[dest=NPC位, 2, qg_fastruning_s_2h_01]`
   对 hyjf02 类剑系技能 anim+flag 双错（官服=fastrun/1），且时机错（受理期发 vs 220 进窗发）。
   fork 客户端是否消费 c61 应用正确方言重测：**m220 且距目标进窗 → c61[落点前 1.7m, flag, anim]**。
3. **P0a 撤 scheduleRushApproachDash 方向正确**（受理期自动发确非官服），复活时改为 220 触发闸。
4. anim/flag 分流表：剑系 `1/qg_fastrun_2h_01`、空手系 `2/qg_fastruning_s_0h_01`、
   灭剑如虹类 `2/qg_fastruning_s_2h_01`（跨系例外存在，先按套路武器映射）。
5. 落码清单（独立小项）：直收簇 CantParry=2、m284、范围技 m20+超时清 CurSkillID、
   重按两路径、m707[2/28/29]、受理簇 CurSkillTarget 恒 None、CD 行授予时机。

## 七、落码记录（2026-11-13 01:00，已部署 F: 待真机）

`cmd/protocol-probe/rush_cast.go`：

- **撤 stage 期 m24**（P0c/P0e 的受理簇第 4 帧 stage-dash 档2——Java 方言误植，官服受理簇零 m24）；
  受理簇回到 8 帧（CurSkillID→CurSkillTarget→m25→m707→四连），rearm 11 帧。
- **`officialRushDashDialects` per-skill 授予方言表**：§1.2 观测窗 + §1.3 A 类 flag/anim；
  未列技能回落 G 类（skill_lock_data LockGRushAction）+ 默认窗 (GRushRange, +Add]。
  pendingCastState 增 dashWindowLo/Hi/dashLevel/dashAction，tryRushArrival 按 pending 窗判授予。
- **c61 终点 y 改目标基准**：`target.Y + 1.0`（四样本拟合；旧「玩家 y+1.0」系 f 平地巧合）。

测试：TestRushDashDialectPerSkill 新增（hyjf02 6m 零帧/8m flag1、xytf07_sky 9.5m flag2+0h、
mzjf02 13m 零帧）；簇帧数断言 9→8 同步 ×6。build.bat 全绿，deploy.bat 已部署 F: 并起服。

**真机判据**：
1. 剑系（hyjf02/myjf02）：远处按压 → 走近 → 日志 `rush within dash window ... in (7.5, 8.5]`
   → 客户端是否执行冲刺（m15 state4/212）——**fork 客户端消费 c61 的首次正方言测试**。
2. 空手系（GM 发逍遥腿法 xytf02/07）：窗 (5,7]/(8,11]，c61=flag2+qg_fastruning_s_0h_01。
3. 若仍不冲刺：官服方言三要素（时机/flag/anim）已对齐，则 c61 消费问题在客户端协议模式侧
   （与自走缺失同根），转动态调试线。

## 复跑

```bash
cd 本仓库根
python -I tools/rush_gap/pcap_frames.py <技能1.pcapng> artifacts/rush1/jn1.pkl
python -I tools/rush_gap/pcap_events.py  artifacts/rush1/jn1.pkl artifacts/rush1/jn1_events.json
python -I tmp/rush1/cast_context.py artifacts/rush1/jn1.pkl artifacts/rush1/jn1_events.json [技能名] [--tl]
```
