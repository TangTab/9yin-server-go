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

## 2026-11-08 数值重写批次：每击公式引擎（combatcalc）

伤害掷骰从「面板攻击区间 × 技能倍率」两参数近似升级为客户端真模型（damage_calculate r 行 × formula.ini 公式），计算移入 Lua：

| 项 | 内容 |
|---|---|
| 新包 | `internal/combatcalc`：transpile（表达式→Lua 闭包体：P<n>/R[a,b]/#Prop/全角括号）+ host（长存 VM、装载期注入公式闭包、ProtectedCall+超时+指令预算、运行期出错整引擎禁用→legacy） |
| 引擎脚本 | `scripts/module/combat/formula.lua`：原 M.calc（Java CombatFormulas 全管线移植，此前未接线）保留，新增 `M.install_formulas`/`M.calc_hit`（参数按 Src 5/6/0/1/2 解析→公式求值→miss 10%→crit 5%×2.0→保底 1；守方侧全零与参考服一致） |
| 目录编译 | `skill_catalog.go`：`compileSkillDamage` 增产 `damageRows`（r 行原始参数），`hitCount`（varprop）入定义；`hurtCount()`=判定帧数＞HitCount＞正 r 段数 |
| 结算口 | `rollSkillHit`（引擎优先→legacy 回退）；**多段从"出招一掷同值复用各段"改为每击×每目标独立掷值**；单段逐目标同前但走新入口；结算发帧链路零改动 |
| 装配 | `-combat-scripts`（默认 scripts/module/combat）+ formula.ini（share/skill/，启动解析注入，非数字节跳过）+ GM `lua_reload` 联动热重载 |
| 回退 | 引擎缺失/公式不支持（转译失败或 Src=4/7/8/9/10/11/12/13）→ `rollSkillDamage` legacy 原样保留，服务器永不因脚本层停摆 |
| 测试 | transpile golden（[1][2][44][494][63全角]）、引擎 golden（太极拳 494=(112+((80+30+20+35)×0.195))/2=72）、fallback 三态、多段独立掷（三段 11/13/17）、开太极 legacy 原样 365 |

已知边界（与 §十六/§十八 待办一致）：守方侧全零（防御/减免/暴击对抗）、外部暴率/爆伤四字段、群妖乱舞怒气暴击（套路 buff，不进基础管线）、多 r 行伤害包（NPC/buff 异形）走 legacy、Min/MaxMeleeDamage 暂用面板攻击区间口径。

### 同批追加：攻方对抗值接入 + 守方管线铺设

- `calc_hit` 的 miss/crit 从硬编码 10%/5% 换为真实公式：`missChance(accuracy, evasion,…)`、`critChance(crit_value, resilience,…)`；攻方暴击/命中按伤害系取 PhyVa/MagicVa、PhyHit/MagicHit（默认 16 暴击值 vs 零封劲 → 暴击率 14.4%，正对 §七 PVE 实测 14–15%）；命中 50 vs 守方零闪避 → PVE 永不 miss（真实口径，低级怪近零闪避）。
- 守方三字段（defense/evasion/resilience）经 `HitInput.Defender` 全程铺通至 Lua `defenseReduction/missChance/critChance`——NPC 属性块（官方 base block 0..98，见 newer-go-server-deployment 记忆）迁移前传 nil 按零值，迁移后只改数据源不改公式。
- 测试：`TestEngineCalcHitAttackerDefenderStats`（14.41% 暴击边界/封劲压制/1 万防→37/闪避对抗/零闪避永不 miss）+ `TestRollSkillHitPicksCritValueByDamageType`（phyVa=0/magicVa=16 两系分选）。

## 2026-11-08 NPC 属性块迁移批次：staticprop 实力包接入守方

| 项 | 内容 |
|---|---|
| 数据源 | `share/npc/staticprop.ini`（745 节＝实力包 PowerNo，含 [-1] 兜底）：五维/HP/MP/五系防/PI·AI 防 17 个成长基数（HPBase 可为浮点如 1.775）。Go/Java/客户端 Lua 此前均未消费；官方 base block 0..98 是场景面板块、不含防御——官服按实力包在服务端推导（视图块 PhyDef@584/Dodge@562/VaDef@573），抓包无攻击怪防御样本 |
| 推导规则 | `属性 = Base × Level`（**推断**，无官方参照；锚＝§七 白狼封劲 20–40 ≈ StaBase1×L30）——集中在 `deriveNPCCombatStats` 一处可调。装于 registerNPCLocked（读模板列 Level@6/PowerNo@60，均场景域字段自动绑定），存 sceneEntity.combatStats |
| 接线 | rollSkillHit 增 ref/world 参：`defenderStatsOf` 按伤害系选防（Phy→外防、Sti/Juj/Neg/Mas→分系）+ Evasion→missChance；两调用点（多段段内逐目标、单段循环）传 ref |
| 风险控制 | **Resilience（封劲）暂不推导**：玩家侧暴击值未成长（默认 16），接 NPC 封劲（≈30）会把暴击率清零——待装备/宝物批抬玩家暴击值后接通。Defense 受对数曲线 ~900 下限天然无害；Evasion（DexBase×Level）vs 命中 50 影响微弱 |
| 装载 | loadChengduNPCSpawns 一并装载；资源仓优先、os.Open 散文件兜底（部署端 resources/modern 为解包树）、双失败→零值守方（不停摆）；启动日志 `packages=745` |
| 测试 | 解析 golden（testdata/staticprop_golden.ini 四节逐字拷自真件，含 1.775 浮点）；推导（×Level/[-1] 兜底/level 缺省/空表全零）；接线（Level50·PhyDefBase200→外防 10000→72×0.519=37；negDef=0 分系不受影响） |

### 附：Go 方言 vs 官方表 0..98 同位异名对照（10 处，纯内部命名卫生，帧按 Index 定位不受影响；本批未改，备查）

| idx | 官方表 | Go 方言 |
|---|---|---|
| 21 | Prop21 | LogicState |
| 26 | Prop26 | SpeedRatio |
| 36 | Prop36 | Name |
| 42 | WuXing | WeaponMode |
| 47 | WeaponName | CollideRadius |
| 48 | WeaponMode | LuaScript |
| 77 | Mount | VisType |
| 82 | CurRiderType | ExtraInfo |
| 85 | NeigongPKStatus | EscortFlag |
| 88 | InParry | WeaponName |

（方言侧 LogicState/SpeedRatio/Name/CollideRadius/LuaScript 在官方表他处存在：463/390/178/179/1464；VisType/ExtraInfo/EscortFlag 官方表完全不存在，已从默认值移除——npc_resolver.go:347 注释。）
