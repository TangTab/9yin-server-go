# 战斗重写写死参数审计（2026-11-07）

以 Java 参考服（G:\pojie\ai\9yin-server\nineyin）为权威、F:\Game\Online\9yin\9yin_server1\captures\local（bak 子目录 52905/12141/52795 等）为线上对拍基准，重写 Go 战斗链后对全部写死参数的定性审计。三类：

- **A. 本次替换的"方便占位"**——最初为方便随手写死、本次已按 Java/抓包改为权威值；
- **B. 官方对齐的事实常量**——有抓包/反编译双源出处，**保留不动**；
- **C. 仍存在的本地占位/分歧**——本轮未动或有意保留，附接入点。

---

## A. 本次替换的方便占位

| # | 旧写死值（位置） | 新值 | 权威出处 |
|---|---|---|---|
| A1 | 玩家伤害 = damage_calculate.ini r 段静态和直扣（旧 skill_combat.go `effect.amount` → `ApplyDamage`） | `rollSkillDamage`：attack 20-35 掷值 × ratio（baseDamage/100/damageWaves），10% 未命中（damage=0 整段不发帧）、5% 暴击 ×2、保底 1 点 | Java `CombatFormulas.calc` + `SkillCastHandler.rollDamage`；抓包 52905 实测同招同值/异招异值/偶有 miss |
| A2 | 无多段调度（出招瞬间一次结算） | `trigger_custom_N`（`段号;trigger_effect;帧号;效果名`）编译 hitFrameMs，按 30fps 帧位逐段调度，一招掷一次同值复用、死亡目标跳过 | Java `scheduleMultiHits`/`compileSkillActions`（52795：gxfhj02×4 段、shl08×6 段实证） |
| A3 | msg24 飘字 script 标签恒 `"SkillNormal"` | `scriptLabel()`：锁定族（SkillLock∨CanRush∨TT4∧DT≠0）一律 `"SkillLock"` | Java `CombatSkill.scriptLabel`；抓包 52905 myjf01 系='SkillLock' |
| A4 | msg20 走 0x27 现代壳、尾值 `int 1 / float 1` | 0x1E 旧壳、尾值 `int 0 / float 0.0` | Java `FightMessages.skillAction`；抓包 42 份 msg20 从未出现在 0x27、尾两值恒 0（#3005 逐字节） |
| A5 | 出招簇多发 `State@183=actionName` 帧 + followup State 定时帧 | 全部移除——动作时间线由 msg20 驱动，F 盘 exe 不下发 followup State 帧 | Java `CombatSkill` 注释「followups 编译保留；抓包显示 F exe 不下发」；抓包出招簇 #3000-#3005 无 State 帧 |
| A6 | NPC 攻击伤害 `MaxHP/50`（下限 10、BossNpc ×2）、周期 1500ms、距离 3.5m | 实伤恒 1、周期 2200ms、近身 6m | Java `NpcCombatAi` 三常量（参考服 55866：名义 20 折算恒 1；12141 实测 ~2s 一轮） |
| A7 | NPC 攻击 msg20 技能串 `"SkillNormal"` | `"jn_mon053_02"` | Java `NPC_ATTACK_SKILL`（55866 追风士；12141 另见 jn_mon043_02——按 NPC 模型取招式的官方列未定位，统一走 Java 常量） |
| A8 | NPC 命中无 0x2D | msg24 后补 0x2D（玩家受击后 HPRatio 双写）——官方血条由 0x2D 驱动 | Java `NpcCombatAi.tick`；抓包 12141 #7053-7054 四连 337→20→24→2D |
| A9 | `ModifySkillLockTime=5000`（面板+0x0D 归档两处，注释自认"本地容差兜底"） | 200 | Java `PlayerActor`（exe #2014 反汇编钉死） |
| A10 | 无顶掉补删：连招把上一招收尾定时器顶掉 → 旧 active_skill_rec 行泄漏 | 新招受理时补发旧招 `0x12 + 六属性清除帧`；收尾定时器按 (id, castEpoch) 配对 | Java §D8 对拍（参考服 add/del 恒 1:1，8/8；52905 14/14） |
| A11 | 无 View40 CanUse 行帧（出招后图标态不变化） | 出招置 0、个人 CD 尽时置 1（`0x10|isView=1|40|行号|CanUse(2236) Byte`） | Java `FightMessages.skillRowCanUse`（62433 抓包 2159 帧；不置则真机图标恒灰） |
| A12 | 收招簇发两条 State="stand"（idx183+idx19） | 只发 idx19 一条（0x12 → 六属性清除帧 → State stand，共 3 帧） | 抓包 52905 收招簇 14/14 恰一条 24B State 帧 |
| A13 | beginSkillUse 即 `SetLogicState(Fighting)`（出招面板 LogicState=1） | 出招不切战斗态，命中才进（noteCombatActivity → setCombatPresence） | 抓包 62433：出招 vital 仍 LogicState=0、命中后切 1；Java `enterCombat` 同语义 |
| A14 | 无武技（wuji）连击段、无战斗真气 | wuji_* 首跳摘 `buf_<技能>_01`（169 v0=0）；怒招每跳目标 debuff 169（2s 租约）+ msg24 c1=71 段帧（''/_2/_3/_4）；伤害合计 ×0.1 折真气入当前经脉（日预算 500） | Java `emitWujiSequence`/`feedMeridianZhenQi`/`PlayerMeridian.addCombatZhenQi`（55866 myjf08 实证） |
| A15 | 无 msg294 对钟帧 | 连接级 1s ticker：`[294, i32 运行毫秒, i64 unixMs+FILETIME 偏移, i64 unix 秒, f64 46327.464]` | Java `LoginFlow.timeSyncFrame`；抓包 52905 747 条计数器恒 +1000 |
| A16 | 公共 GCD 门控 = per-team 硬闸（`skillTeamCooldowns[CoolDownTeam]`，同套路 5 招共享 team → 任一出招把全套路**含施法者自己**锁 7s，个人 CD<7s 的技能全被压到 7s；"GCD 变 20s"观感 = 图标按个人 CD 点亮但点击被静默拒 + 同套路轮换互推窗口叠加） | 跨套路切换锁：`publicCDUntil/publicCDTaolu` 单字段，`now<窗口 && 套路≠发起套路` 才拒；同套路连招/自身复读只走个人 CD；每次受理刷新窗口 | bug.txt:8「同套路不在触发公共7秒冷却，不同套路会看到7秒CD转圈回复」；Java `PlayerActor.beginSkillUse`:1061-1089（commit c58025f2 2026-11-02 修过同一 bug）；表实证：玩家套路 skill_new.AddCoolDownTime 全 7000、varprop 全空，无任何 20000 |

## B. 官方对齐的事实常量（保留）

| 常量 | 值 | 出处 |
|---|---|---|
| 0x2D 三常量 | u16@11=2、u16@13=28、u16@19=112，HPRatio 双写 | F 盘反汇编 + 62433/12141 抓包全程不变 |
| msg24 技能形态 c1 | 17 | 抓包恒 17 |
| msg24 NPC 形态 c1/名义攻击力 | 7 / 尾值 20 | 55866 抓包 |
| 0x13 fight_relation 行列 | 恒 1/1，累计=逐击伤害和 | 抓包 143→210→277 |
| GCD 行类别 / 个人行 team / GCD 回落组 | 50000 / 0xFFFF / 50001 | 抓包 + gui.lua SetCommonTeamCoolRange 只认 50001-51000 |
| 记录表下标 | cooldown=109、fight_relation=114、active_skill=446 | 官方 447 schema，init() 自检防漂移 |
| 属性索引族 | 19/183(State)、208/209、221、460/463/464/465/467/469/478/479/578/624/627 等 | 官方属性表按名断言测试钉死 |
| 怒气得怒 | 实招命中未格挡 +2/次（多段每段每目标各算）、被击未格挡 +1/格挡 +2 | 用户钉死官方规则（Java 同值） |
| NPC 面板 9 属性 | {21 Byte=1, 28/30/32, 463, 465/469/578, 464}（57B） | 抓包 52905 #3027 逐字节（Java 5 项子集、Go 旧 6 项含非官方 481，均收敛到此） |
| 冲锋链参数 | 8m/15m/3m 三层距离、30s 暂存窗、BindDistance 回落 1.5、动画名两方言 | Java SkillCastHandler 常量 + 61653 抓包（已有，未动） |
| buff StaticData/租约 | 15101/35s、15105/5s、6013/6055/6043/6050、parry 槽 11/4558 等 | F: 部署 buff_new.ini 实证（已有，未动） |
| 回怒 SP 帧 | 纯 SP 变化 18 字节单帧（idx 467） | 49840 帧 #16 线样 |
| wuji c1=71 双精度对 | f64 30.0 / 0.0（语义未钉死，逐字节复刻） | 55866 抓包 |
| msg294 尾常量 | f64 46327.464 | 官方抓包常量（游戏世界天数口径） |

## C. 仍存在的本地占位 / 有意分歧（接入点）

| # | 项 | 现状 | 说明 |
|---|---|---|---|
| C1 | 攻方区间无装备/内功分量 | `meleeDamageRange()` = 基础 20-35 | Java 叠装备外功+内功 Min/MaxMeleeDamage+PhyDamageAdd；Go 装备词缀层未移植，接入点在 `playerActor.meleeDamageRange()`（公式已备） |
| C2 | 守方(NPC)防御/命中/闪避/暴抗全 0 | 10% 基础未命中、无减伤 | Java rollDamage 同款传参（守方面板缺省）；公式分支已移植，NPC 面板有防后填 `combatFormulaInput` 即启用 |
| C3 | msg19 锁定帧不发 | 锁定族仍走 msg20（带目标对象） | Java 走 c61+0x1F+msg19，但 42 份抓包 msg19 0 次 + Go 曾因发它回归「无冲刺动画」；帧构造器与 bind 计算已在 Java 侧验证，恢复时按 `emitLockCast` 移植 |
| C4 | NPC 名 msg24 c1=7 用宽串 | 与 Java 一致 | 抓包 12141 实测窄串(UTF-8 text)方言；两台参考服分歧，按 Java/官方 55866 口径，真机异常再切窄串 |
| C5 | NPC 仇恨/追击/栓绳模型 | Go 特有（aggro 12m/chase 30m/移动 4.5/tick 上限 4） | Java 只有被动反击（无 aggro）；Go 特性保留，攻击语义已对齐 Java。模板列 SpringRange/ChaseRange/RunSpeed 可覆盖前三个 |
| C6 | 击杀掉银 MaxHP/100 clamp [1,99] | 保留 | Java 无掉银；Go 本地经济特性，用户此前拍板保留 |
| C7 | CanUse 周期保活批未实现 | 仅出招置 0 / CD 尽置 1 | Java LoginFlow 2s ticker（≤17 行）保活；如真机出现图标复灰，按 rage-recovery-frames.md 契约补 ticker |
| C8 | 内功 attacker 附伤钩子未移植 | roll 不加附伤 | Java `NeiGongEffects.onAttack`（如 wd_004 震伤窗口附阳伤）；Go neigong_effects.go 只有受击侧被动 |
| C9 | Lua `on_hit` 运行期倍率钩子未移植 | ratio 固定编译值 | Java `LuaScriptService.skillOnHit`（scripts/skill/<ID>.lua r_adjust）；Go 无该脚本层 |
| C10 | 玩家 LogicState 进战斗帧延迟 ~1s | 命中后下一 combatTick 才推 | Java 命中即推单属性帧；Go 走 setCombatPresence 批量通道，战斗体验无差、如需逐帧对齐可在 writeSkillSettleFrames 内补推 |
| C11 | race 检测未跑 | 本机无 gcc（CGO 不可用） | 多段共享态已用 atomic + 既有锁序（run.mu/p.mu）保证；部署前建议在有 gcc 的环境跑一次 `-race` |

## 对拍基准速查（重写后的完整出招簇）

单段（Java handleValues 同序）：
```
[顶掉补删: 0x12 + 清除帧]         （仅旧招未收尾）
0x11 个人CD → 0x11 active(446) → 0x11 GCD(50000)
0x10 View40 行 CanUse=0           （CD 尽另定时置 1）
0x10 玩家面板
0x1E msg337 施法回显
0x1E msg20 [obj, 技能, obj目标, 0, 0.0]
--- 每命中目标 ---
0x10 NPC 9属性 → 0x13 → msg24(17,scriptLabel) → 0x2D → msg18 → msg205
--- wuji 族附加 ---
169 摘 buf_<技能>_01 →（怒招）每受害 NPC 169 目标debuff + msg24 c71
SP 单帧(命中×2) → 经脉真气 View45 行帧
--- 收尾（动作总时长后）---
0x12 → 六属性清除帧 → State(19)="stand"
```
多段：出招簇同上（无命中段），每 trigger_effect 帧位对存活目标重复全套命中帧 + wuji 段帧（`<LevelupRelevance>_2.._4`）。
NPC 攻击（每 2.2s，6m 内）：`337(NPC) → 20(NPC→玩家, jn_mon053_02) → [命中: msg24(7,1,…,20) + 0x2D]`，格挡零伤害帧、回怒 +1/+2。
