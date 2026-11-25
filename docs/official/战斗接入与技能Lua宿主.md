# 战斗接入与技能 Lua 宿主定版（2026-11-25）

> 承接 [技能服务端XML构建链与m211受理链](技能服务端XML构建链与m211受理链.md)（下称
> 「受理链」）：那边做到出招为止，本篇接上**命中 / 伤害 / 击杀 / 还手 / 脱战**，并把
> 技能与 buff 接进官方 `nx_*` 契约的 Lua 宿主。
>
> 证据源：
> - 官服抓包 `E:\Game\9yin\9yin-server-go2\资料\`：女号完整引导包（8032 线，197~232s
>   交手段、345~400s 施放段）、挑战包（7071 线，447~484s），逐帧解码
>   `go run ./tools/pcapevents <包> <端口> dumprec`。
> - 官方服务端脚本 `Res/bak1/lua`（数据根 `lua/` 层内是同一语料，361 文件）：钩子契约
>   与 `nx_*` 调用面的事实源。
> - 设计态表 `share/skill`（伤害包 / 公式 / buff 族）与 `ini/action`（招式判定帧）。
> - 2433 线 `docs/combat/` 九篇：数据链与结算系数的出处；流程与帧方言不照搬
>   （理由见受理链 §一）。
>
> 结论先行：
> 1. **命中节拍取招式表**：判定帧 = `trigger_hitbreak ∪ hitback`，30fps，按
>    `action_speed` 折算；女号包五个技能逐拍对上。
> 2. **伤害走公式引擎**：`DamageCalculate → r 行 → Formula` 全链数据驱动，1,275 条
>    公式转译成 Lua 闭包全部装载；新号面板直接套公式取整即得实测值。
> 3. **技能接入 Lua = 官方契约的宿主**（`internal/script`）：49 个 `nx_*` 入口、对象
>    句柄、每份钩子脚本独立的函数环境；出错只禁用出错的那份脚本，服务器不停摆。
> 4. **帧方言按抓包定版**：首次交手、每击、收招与锁定 buff、击杀、闪避、招架、NPC
>    还手、脱战八组帧簇，外加 buff 周期伤害帧，用例逐帧断言。

## 一、数据链：两类新产物

### 1. 招式判定节拍（`tools/datagen action` → `ini/Action/zhaoshi_hits.xml`）

客户端招式表 `ini/action/zhaoshi_*.ini`（`zhaoshi_list.ini [include]` 列 11 档）是动作
分段与触发帧的设计态数据：

```ini
[CS_jh_cqgf01]
1=tzcq_10;0;45;-1;-1;1                        段号=动作名;起始帧;帧长;入混合;出混合;标志
trigger_custom_2=1;trigger_hitback;27;0.6,hurt_f_1_l,,0.2
trigger_custom_7=1;trigger_hitbreak;13;hurt_f_2_s,
trigger_custom_6=1;trigger_action_speed;32; ,2.5,CS_fight_0h_jb_16
```

构建器把每招编译成一行 `<Property ID Actions Frames Hits Length/>`：`Hits` 是各次
命中判定的时刻（毫秒，自招式起算，升序去重），`Length` 是变速折算后的总时长。
12,770 招（11,310 招带判定，27,255 条判定触发），产物 1.3MB。装载器
`internal/actiondata`（进程级缓存；数据根探测与 `skilldata` 同款）。

折算规则（`tools/datagen/action/action.go`）：

- 判定帧 = `trigger_hitbreak / hitback / hitdown / hitout / hitclasp`；同刻多触发只算
  一击。`trigger_action_speed` 是变速，**复位到 1 倍也是变速，不是判定帧**。
- 段内时刻 = `Σ 帧数 × (1000/30) / 当时速度`；多段招式前段时长累加为后段起点。
- 段号越界、帧号超出段长的触发忽略；清单外的同名招式按清单序首见胜出。

对拍女号包（名义时刻 → 实测，差值为链路延迟）：

| 招式 | 段 / 帧 | 判定时刻 | 实测 |
|---|---|---|---|
| `zs_normal_00_01`（普攻） | `ptgj_0h_01;ptgj_0h_05` / 15;29 | 467 / 1000 | 487 / 1051 |
| `CS_jh_cqgf01`（长拳） | `tzcq_10` / 45 | 433 / 600 / 900 | 452 / 682 / 964 |
| `CS_jh_cqgf04` | `CS_fight_0h_jb_16` / 58（28 起 ×0.5、32 起 ×2.5、46 复位） | 1280 | 1277 |
| `CS_jh_cqgf06` | `jb_0h_06;jb_0h_02` / 17;30 | 500 / 1067 | — |

### 2. 命中结算包（并入 `tools/datagen skill`）

与技能对象行同构的引用链，新增 7 个产物（`tools/datagen/skill/mapping.go` 尾段）：

| 源表 `share/skill` | 产物 `ini/Skill` | 行数 | 用途 |
|---|---|---|---|
| `damage_calculate.ini` | `Skill_DamageCalculate.xml` | — | 伤害计算包：一行一条或多条 `r=公式ID,伤害类型,参数…`（多条输出 `r/r_2…`） |
| `formula.ini` | `Skill_Formula.xml` | 1,275 条公式 | `Src1..12/Prop1..12` 给 `P1..P12` 的来源，`Formula` 为表达式 |
| `buff_new.ini` | `Buff_New/Buffer.xml`、`Unknown.xml` | — | buff 对象行（按 `script` 逻辑类分档） |
| `buff_static.ini` | `buff_static.xml` | 29,087 | buff 静态包行（Category / EffectType / LifeTimeType / LuaScript / Min·MaxVarPropNo） |
| `buff_varprop.ini` | `Buff_VarProp/Buffer.xml` | 54,929 | buff 逐级行（LifeTime / EffectTimer / EffectRatio） |
| `skill_static_to_id.ini` | `Skill_StaticToId.xml` | 17,035 | 静态包号 → 技能 ID（NPC 战斗行技能列记静态包号） |

装载面：`internal/skilldata/damage.go`（`DamageCalculate` / `Formula` / `Formulas` /
`Transpile`）、`buff.go`（`Buff`，对象行 → 静态包行 → 逐级行三段联查，越界等级退回
首行）、`skilldata.go`（`SkillIDByStaticData`）。

### 3. 源表脏数据：审定修正（`tableSpec.Fixups`）

`formula.ini [1116]` 的 `Formula=((P1+((P2+P3+P4+P5)*0.065))` 多一个最外层左括号，
不配平，引擎装载期编译失败。处置分两层：

- **转译器**（`skilldata.Transpile`）做括号配平校验，不配平即拒——坏行到不了引擎。
- **构建器**按审定值改写源值，使产物可由重建逐字节复现：`mapping.go` 的
  `Fixups`（节名 → 键 → 审定值 + 理由），命中即改写并进审计报告
  （`skillgen-report.json` 的 `fixed`）。1116 定版为去掉最外层左括号
  `(P1+((P2+P3+P4+P5)*0.065))`（语义不变；2026-11-25 手修定版）。

> 产物是层文件里的生成物，**手改会被下次 `go run ./tools/datagen skill` 覆盖**。手修
> 定版一律落 `Fixups`；复现校验：重建后 `store pack` 应报「去重 N 个、无变化」。

### 4. 归档

产物在数据根的增量层 `testdata/b003.rep`（3.0 MiB）：收录 8 个新产物，另有 12 个
重建后与 `b001` 逐字节相同的既有产物在并入时去重。

## 二、命中模型（`fssoloworld/combathit.go`）

命中计划 `hitPlan{times, hurt}` 的取用顺序：

1. **招式表判定帧**：按 m20 下发的动作名（`skillAction`，连招段按技能各自轮转）查
   `ini/Action` 产物。
2. **流程包**：`HitTime` 起算，`HitCount` 击按 `LeadSepTime` 间隔（缺省均分）——NPC 的
   `default_normal_skill` 即此形（HitTime=1000）。
3. 都没有 → 不结算（非伤害技）。

- 分母击数（公式的 `HurtCount`）= 计划击数。
- 每击重新索敌：范围技以玩家**当前**位为形状中心（2433 定案「技能期间进入范围的
  都要计算伤害」）。形状取 `Skill_HitShape`（1 圆 / 3 扇形 / 4 矩形），高度差上限
  `HeightDiff`。
- 目标须可攻击：摆放脚本 `AttackNpc` / `BossNpc` 且模板未置 `CantBeAttack=1`。

**时钟**：命中、脱战、buff 到期、NPC 行走都排在战斗事件队列上，由 `CombatPump`
驱动（线上 50ms 节拍；测试用可拨动的 `combatNow` 手动推进）。一拍只取开拍时已到期
的事件，事件处理中新排的到点事件留到下一拍。收招定时器取 `castLockMS` 与「末击 +
余量」的较长者，余量 = 泵节拍 + 一帧：命中由泵放行、可能晚于名义时刻，收招若抢到
末击前面，此刻受理的新施放会顶掉序号、末击被丢。

**锁序**：玩家 `combatState.mu` → NPC `npcFighter.mu`；调 Lua 钩子与引擎前一律放锁
（脚本会经 `nx_*` 回到宿主取锁）。

## 三、伤害模型

### 1. 公式参数来源（`formula.ini` 的 `Src` 列）

`0 / 3 / 11 / 15` 攻击者属性（`Prop` = 属性名）· `1` 技能上下文（`HurtCount` 击数 /
`CurTargetCount` 目标数 / `Level`）· `2 / 14` 目标属性（14 = 目标 HP）· `4` 常量 ·
`5` 伤害计算行的位置实参 · `6` 位置实参（`a;b` 区间，取区间内随机数）· `7 / 8` 攻击者
/ 目标身上某 buff 的层数 · `9` 与目标的距离 · `10` 具名变量 · `12 / 13` 攻击者 / 目标
MaxHP。

伤害类型记号映射到 `define.lua` 的 `DAMAGE_TYPE_*`：`PhyDamage` 外功 0 / `StiDamage`
刚 1 / `JujDamage` 柔 2 / `NegDamage` 阴 3 / `MasDamage` 阳 4。一个包的多条 r 行逐条
求值、结果相加。

### 2. 表达式转译（`skilldata.Transpile`）

| 源 | Lua 函数体 |
|---|---|
| `P<n>`（或小写 `p<n>`，n = 1..12） | `PV[n]`（引擎按 `Src/Prop` 绑定取值） |
| `R[a,b]` | `R(a,b)` 区间掷值 |
| `MIN[..]` / `MAX[..]` | `MIN(a,b,…)` / `MAX(…)`（可嵌套） |
| `#属性`、裸标识字 | `Q("属性")` 直接引用 |
| 全角 `（），` | 归一为半角 |

子集之外的记号、越界参数位、括号不配平一律报错；调用方把该公式记为不可转译，
用到它的击走回退面。现产物 1,275 条**全部可转译并装载**。

### 3. 结算管线（`internal/script/lua/combat_formula.lua`）

与 2433 线 `scripts/module/combat/formula.lua`（Java 参考服 CombatFormulas 的移植）
同口径，系数同源：

```
公式求值 → 防御减免 → 未命中 → 取整 → 暴击（整伤 ×2）→ 招架（×0.10）→ 保底 1 点
```

- 防御减免 `20·ln(外防/100) − 44`（%），夹 `[0, 75]`；外防 ≲ 900 不减伤。
- 未命中率 `10% + 15%·(闪避 − 命中)/(闪避 + 1)`，夹 24% 上限。
- 暴击率 `5% + 10%·(暴击值 − 坚韧)/(暴击值 + 1)`，夹 15% 上限。
- **先取整再暴击**：抓包锁定技 8 → 暴击 16，是整伤翻倍。
- 随机数由宿主预掷（`HitContext.Rand`：`[1]` 未命中、`[2]` 暴击、其后区间掷值），
  脚本不掷骰，结果可复现。

技能脚本给了 `get_damage(self, skill, target)` 时，取其返回的 5 系伤害（外功 / 刚 /
柔 / 阴 / 阳）之和为原值，仍走同一结算管线（未命中 / 暴击 / 招架 / 保底由引擎管）。

### 4. 官服对拍（新号面板 MeleePower = 3）

| 技能 | 伤害包 → 公式 | 代入 | 取整 | 抓包实测 |
|---|---|---|---|---|
| 普攻 `zs_normal_00` | 52 → 372 `(P1+((P3+P4)·P2·0.1))/P5` | (5 + 3×0.1) / 2 | 2 | 2（两击） |
| 长拳 `CS_jh_cqgf01` | 2998 → 237 `(P1+((P4+P5)·P2·0.3))/P3` | (25 + 3×0.3) / 3 = 8.63 | 8（暴击 16） | 8 / 16 |
| `CS_jh_cqgf06` | 3789 → 237，实参 16 / 1.1 / 2 | (16 + 3×1.1×0.3) / 2 = 8.50 | 8 | 8 |
| `CS_jh_cqgf04` | 3004 → 237，实参 32 / 1.3 / 1 | 32 + 3×1.3×0.3 = 33.17 | 33 | 33 |

新号的外防 / 闪避 / 暴击对抗项都在「不触发」区间，所以实测值 = 公式原值取整；对抗
项的系数来自上面两份资料，尚无抓包样本能单独印证（见 §八）。

## 四、官服帧方言（抓包定版）

| 场景 | 帧序（抓包时刻） |
|---|---|
| **首次交手**（215.577） | `NPC.LogicState=1` → 玩家 `fight_relation_rec`（rec114）加行 `[对手, word=(now+36860ms)&0xFFFF]` → `玩家.LogicState=1` → `m24[npc,37,"npc_foundplayer"]` |
| **每击**（208.229 / 216.675） | 技能行 `CurTargetCount` → 攻击者 `FP` → `SP`（各 +2）→ 关系行刷新（`0x13`）→ `m24[目标,17,伤害,0,0,技能,逻辑类,目标名,攻击者名,暴击]` → 第 2 击起 `[m24[攻击者,10,累计击数]` + `MaxHits]` → `m205[攻击者,技能,目标…]` |
| **收招**（217.239） | 末击后 `CurSkillID=""`（`CurSkillTarget` 保持）。锁定技末击另挂 buff：`CurSkillID=""` → `m169[0,"buf_<技能>",目标,攻击者,目标名,攻击者名,到期毫秒(1601 纪元),Category]` → `CurSkillTarget=None` → `m189[攻击者]` |
| **击杀**（210.666） | `m24 17` → 关系行删（`0x12`）→ `m24[npc,29,攻击者,npc名,攻击者名]` → `m18[npc,"hurt_half",0,攻击者]` → `m24[npc,13]` |
| **闪避**（220.371 / 226.276） | `m24[目标,7,1,攻击者,技能,目标名,攻击者名,100]`；锁定技另配 `m20[目标,"<技能>_dodge",攻击者]` + `m707[14]`；施放就此收束（`CurSkillID` 与 `CurSkillTarget` 同清），无 FP / SP |
| **招架**（225.883） | `m24[玩家,8,1,npc,技能,玩家名,NPC名]` → 关系行 → 怒 +4 → `ParryRatio` / `ParryValue`（−5）→ `FP` +2 → `HPRatio` / `HP` → `m24 17`（穿透后伤害） |
| **NPC 还手**（218.058 / 223.954） | 玩家 `SP` +1 → `HPRatio` → `HP` → 关系行 → `m24[玩家,17,…,"SkillNormal",玩家名,NPC名,0]` → `m205[npc,技能,玩家]` → NPC `CurSkillID=""` → `CurSkillTarget=None` → `m189[npc]` |
| **脱战**（228.138 → 230.166 / 231.152 / 231.303） | 玩家最后一次战斗活动 3s 后 `LogicState=2`、`FP`/`SP` 清零；5s 后回 0、关系行删；NPC 2s 后 `LogicState=0` 归位 |
| **buff 伤害**（挑战包 5.219 / 8.739） | `m24[目标,22,伤害,0,0,buffID,"Buffer",目标名,施法者名]`——到施法者名为止，**没有** 17 帧那样的末位暴击标记 |

两份抓包的形态统计印证了这几条：`m24 17` 共 282 帧全是 10 值（末位暴击标记），
`m24 22` 共 22 帧全是 9 值。NPC 的 HP **不以属性帧下发**（客户端凭 `m24 17` 自行
扣减）；`HP` / `HPRatio` 帧只发给受击玩家本人。

## 五、运行时（`internal/fssoloworld`）

| 文件 | 职责 |
|---|---|
| `combat.go` | 帧方言文件头、战斗体状态（玩家生存态 / NPC 战斗体 / buff）、节拍常量、战斗事件队列与 `CombatPump`、玩家 / NPC / 技能 buff 辅助对象视图、帧汇 `frameSink` |
| `combatcore.go` | Lua 宿主懒建（`combatLua`）、玩家面板解码、NPC 战斗属性、帧汇的可见性广播 |
| `combathit.go` | 玩家技能的命中计划 / 索敌 / 每击结算 / 帧簇；引擎回退面 `legacyDamage` |
| `combatdamage.go` | 通用伤害落地（脚本 `nx_damage` / `nx_damage_target`、buff 周期伤害、治疗） |
| `combatnpc.go` | NPC 战斗 AI、尸体移除与重生、主动索敌 |
| `combatbuff.go` | buff 挂载 / 到期 / 周期触发、记录表效果（`DirectPropertyRec` / `ContinuePropertyRec`） |
| `combatworld.go` | Lua `World` 适配：对象注册表与 `nx_*` 战斗动作面的落点 |

接入点：受理链在受理帧落地后登记命中计划与收招定时器（`skill.go`、冲锋落地
`skillrush.go`）；`timerCheck` 每秒驱动 `combatTick`（MP / 招架耐力回复）与
`npcAggroTick`。

NPC AI（`combatnpc.go`）：被发现或被击 → `npcReactDelay` 后决策；目标超出攻击距离则
每 `npcChaseStep` 推进一步（`State=run` + `0x1F`）；在距离内且过了 `npcAttackPeriod`
即起手（`CurSkillID` → `CurSkillTarget` → `0x1F` → `m20`）；命中走与玩家对称的结算
（闪避 / 招架 / 命中三种帧序）；空闲超 `npcCalmDelay` 或目标离得太远则脱战归位；
死亡后尸体留 `corpseKeep`，`npcRespawnDelay` 后重生。NPC 的技能取战斗行技能列经
`Skill_StaticToId` 联查。

开关（真机 A/B 用，不必回滚代码）：

| 环境变量 | 作用 |
|---|---|
| `NINEYIN_COMBAT=off` | 关停全部命中结算（受理链照旧，施放仍按出招锁收招） |
| `NINEYIN_NPC_AI=off` | 关停 NPC 战斗 AI（仇恨 / 追击 / 起手 / 脱战）；玩家侧结算照常 |
| `NINEYIN_LUA_DIR=<目录>` | 官方脚本根；缺省 `<数据根>/lua` |

## 六、Lua 宿主（`internal/script`）

### 1. VM

- gopher-lua；每个 `Server` 一份，首次用到时建。脚本按路径键装载：
  `skill\test_hook.lua` 这类反斜杠路径归一为 `skill/test_hook`（大小写不敏感）。
- **每份钩子脚本独立的函数环境**。官方宿主里每份技能 / buff 脚本是一个独立的 Lua
  上下文——脚本都把 `get_damage` / `skill_before_hit` 定义在文件作用域，`AddBuff`
  这类模块同名不同实现。本 VM 给每份钩子脚本一张环境表，`__index` 指向全局表：
  `nx_*` 与内置引擎读得到，写只落在自己环境里；环境里的 `require` / `dofile` /
  `loadfile` 是绑定本环境的版本，模块全局各归各（官方脚本各自 `require("define")`
  正为此）。`Function(script, name)` 只认本脚本环境里定义的函数。
- `require` / `loadfile` / `dofile` 经资源仓（`internal/mpq/res`）读文件——散文件与层
  文件两种布局都能读。
- **错误隔离**：单次调用有预算（缺省 50ms）；脚本装载或运行期出错只禁用**出错的
  那份脚本**（其后返回 `ErrScriptDisabled`），引擎脚本与其余脚本不受影响。公式引擎
  出错只禁用引擎一次，之后全部击回退 Go 面。
- **嵌套调用**（回调里再入）结束后把 VM 的上下文还原成外层那一个——曾经无条件摘
  掉，外层主循环随即对着 nil 上下文崩溃、外层脚本被误禁用
  （`TestNestedCallKeepsOuterContext`）。

### 2. 对象句柄

Lua 侧的对象是 `Handle{Kind, ID, Helper}`（场景 1 / 玩家 2 / NPC 4 / 物品 8 / 辅助对象
16），不暴露 Go 指针。`nx_create_from_config(scene, 配置 ID)` 建技能 / buff 辅助对象；
`nx_type` / `nx_get_config` / `nx_query_property` / `nx_set_property` /
`nx_query_record_value` 等经宿主的 `Object` 接口落到玩家、NPC 或辅助对象（辅助对象
带 `SkillStatic` 静态包属性与记录表）。buff 对象的 `Sender` 属性 = 挂它的施法者
（官方 `buf_testbuff_03.lua` 的写法）。

### 3. 钩子（官方契约；`internal/script/script_test.go` 逐条断言调用约定）

| 钩子 | 宿主调用点 | 状态 |
|---|---|---|
| `skill_before_hit(self, skill) → 1` | 每击结算前（返回 0 = 本击不结算） | 已接 |
| `get_damage(self, skill, target) → 外功,刚,柔,阴,阳` | 每击取伤害原值 | 已接 |
| `skill_after_hit(self, skill, target)` | 每击结算后 | 已接 |
| `buffer_active` / `buffer_deactive(self, buffer)` | buff 挂载 / 摘除 | 已接 |
| `buffer_timer(self, buffer)`（计次型返回 0 = 本次不计） | 周期型 buff 每跳 | 已接 |
| `get_damage(self, buffer, target)` | `nx_damage_target(sender, target, buffer)` 取 buff 伤害原值 | 已接 |
| `refresh_property`、`skill_prepare` / `skill_after_prepare`、`skill_finish`、`skill_break`、`buffer_break`、`adjust_hit / dodge / defend_prob / parry`、`effecttype_compare_value_add`、`AddBuff` 脚本 | — | **未接**（契约见官方 `SKILL/BUFF 常用函数.txt`；`Res/bak1/lua` 语料里除 `skill_break` 一处外无实例，调用点与返回值语义无实证） |

### 4. `nx_*` 调用面（`api.go`，49 个入口）

| 类 | 入口 |
|---|---|
| 转换 | `nx_int` `nx_number` `nx_float` `nx_double` `nx_string` `nx_widestr` `nx_boolean` `nx_int64` `nx_null` `nx_null_obj` `nx_object` |
| 对象 | `nx_exists` `nx_type` `nx_get_scene` `nx_get_config` `nx_get_script` `nx_get_position` `nx_get_child` `nx_get_child_count` |
| 属性 / 记录 | `nx_query_property` `nx_find_property` `nx_set_property` `nx_inc_property` `nx_add_record_row_values` `nx_query_record_value` `nx_get_record_rows` |
| 脚本 / 消息 | `nx_execute`（调另一脚本的函数，结果包成表）`nx_custom` `nx_sysinfo` |
| buff | `nx_create_from_config` `nx_destroy_self` `nx_add_buffer` `nx_remove_buffer` `nx_find_buffer` `nx_damage_target` |
| 战斗 | `nx_damage` `nx_get_around_list` `nx_distance` `nx_distance3d` `nx_break_any_flow` `nx_stop` |
| 依赖宿主可选能力 | `nx_use_skill_to_target` `nx_query_skill_int` `nx_query_skill_string` `nx_find_skill` `nx_add_skill` `nx_get_current_neigong` `nx_find_neigong` `nx_add_countbeat`；`nx_object(数字)` |

末行的入口在战斗宿主里**没有落点**（`SkillCaster` / `SkillFinder` / `NeigongFinder` /
`CountBeater` / `IDResolver` 几个可选能力面未实现，`nx_query_skill_*` 除外），调用返回
false / nil，不报错。未登记的 `nx_*` 名字落全局兜底桩：返回空值并一次性 trace
（`__nx_unsupported`），脚本不致硬崩。

### 5. 再入

Go 回调（`nx_*` 的落点）由 VM 在持锁协程上同步调用；回调里再调脚本必须经
`script.Reentrant`（不加锁的调用面），用普通 `*VM` 会自锁——VM 锁不可重入。

### 6. 公式引擎（`formula.go`）

`NewEngine(vm, specs)` 装载内置 `lua/combat_formula.lua`，把每条公式编译成
`function(PV, R, Q, MIN, MAX) return <函数体> end` 注入；`CalcHit(ctx)` 每击 × 每目标
一次，`Settle(ctx, raw)` 供脚本给出原值时只走结算管线。公式求值包在 `pcall` 里，
单条公式运行期出错只让这一击回退。

## 七、验证

| 包 | 用例 | 覆盖 |
|---|---|---|
| `internal/fssoloworld` | 13 | 普攻两击节拍与帧序、真实时间流逝不影响帧序、锁定技三击 + `m169` 挂载与到期摘除、击杀簇、闪避收束、脱战、NPC 还手、技能脚本钩子（`get_damage` + `skill_after_hit` 里 `nx_*` 建 buff / 读写属性）、`nx_break_any_flow` 打断、buff 脚本周期伤害（`Sender` + `nx_damage_target`，照抄官方 `buf_testbuff_03.lua`）、技能钩子经 buff 脚本再入后仍执行到底、引擎全量装载（1,275 条零跳过）、关停开关 |
| `internal/script` | 7 | 钩子调用约定、反斜杠 `require`、钩子脚本环境隔离、预算只禁用出错脚本、引擎复算官服新号伤害（含 `Settle`）、回调内再入、嵌套调用不破坏外层上下文 |
| `internal/skilldata` | 6 | 伤害包取行、公式参数绑定、转译规则与拒绝集、全量公式可转译 + 1116 审定值、buff 三段联查、静态包号映射 |
| `internal/actiondata` | 2 | 四招节拍对拍官服、进程级缓存 |
| `tools/datagen/action` | 4 | 单段多判定帧、变速折算、多段累计与去重与清单首见、落盘 + 回读 |
| `tools/datagen/skill` | 1 | 审定修正改写与报告 |

全仓 `go test ./... -count=1`：71 包通过；`go vet` 仅 `internal/snailsharemem` 一条
既有告警（本次未触碰）；`internal/fssolopublic TestBootChainOverTCP` 在全仓并行负载
下偶发（该包不链接本次改动的任何包，单跑稳定通过）。

## 八、未覆盖与推断项

**未接**：

- **玩家死亡**：致命伤钳到 1 HP，没有死亡流程（`LS_DIED`、复活）。
- 招式克制（`Rule/skill_new.lua` 的 EffectType 1 克 2、2 克 3、3 克 1 与同型比力道 /
  身法）与 §六.3 表末行的钩子；§六.4 末行的可选能力面。
- `HitHP` / `HitHPRatio` 两帧未发（NPC 打玩家的 `HPRatio` / `HP` 已发）；`m24 17` 的
  两个次级 int 恒 0。
- 抓包里见到但语义未定、未实现的 `m24` 子码：buff 效果的 16 / 19 / 23 / 25（与 22 同形
  或多一个浮点，挑战包各有样本）、`54`（`prepare` / `end_prepare`，NPC 起手读条）、
  `71`、`11`，以及无参的 1 / 33 / 34 / 41 / 56 / 57。
- 脚本写法 `self.Color = "…"`（点号读写属性）不支持，对象上没有 `__index` / `__newindex`
  ——语料 361 份里只有 `buf_poison_04.lua` 一处；主流写法 `nx_set_property`（120 份）
  已支持。
- 钩子里脚本发起的帧（`nx_add_buffer` / `nx_damage` / `nx_break_any_flow`）即时下发，
  排在本击帧簇**之前**；官服序无抓包样本，待定。
- 多人同屏时 NPC 相关帧只广播给看得见该 NPC 的玩家，未经真机多人验证。
- 引导、打断、空中档冲锋、锁定技原地施放变体（承受理链 §七）。

**推断（抓包未直接覆盖，常量在 `combat.go` 并注明）**：尸体停留 5s、重生 15s、NPC
反应延迟 600ms；对抗项系数（防御减免 / 未命中 / 暴击）无抓包样本，只有资料出处；
`fight_relation_rec` 第 2 列的对钟提前量 36860ms 取 49 个样本的中位。

**未验证**：真机（客户端实际表现、177 个挂 Lua 的特殊技能逐个运行）；exe 未重编。

## 九、工具与复跑

```sh
go run ./tools/datagen skill action          # 重建两类产物（散文件落 testdata，verify 回读）
go run ./cmd/mpq store pack testdata -prefix b -prune \
  -exclude mpq -exclude README.md \
  -exclude config-bootstrap -exclude lua-bootstrap -exclude mysql -exclude share.7z
go run ./tools/pcapevents <抓包> <服务端口> dumprec   # 逐帧解码（含记录表行；女号包 8032、挑战包 7071）
go test ./internal/fssoloworld ./internal/script ./internal/skilldata \
  ./internal/actiondata ./tools/datagen/... -count=1
```

> `store pack` 的排除清单须带上 `config-bootstrap/`、`lua-bootstrap/`、`mysql/`、
> `share.7z`——它们是受 git 跟踪的散文件，不在排除内会被吞进层并从工作区删掉。
