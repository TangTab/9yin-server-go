# 技能描述 IR（受控词表）——desc_effects.lua 权威定义

> 生成器：`tools/skill-data/gen_desc_effects.py`（幂等；`--selftest` 金样本断言）。
> 数据源：`<server>/resources/modern/text/chineses/string_skill_desc.idres`
> （`desc_<技能ID>_<等级>`，2,538 玩家技能 / 30,603 条分级描述）。
> 产物：`scripts/module/skill/desc_effects.lua`（2,208 条目；纯数据模块，
> 经 `internal/skillscript` 装载快照，查询 `DescEffects(skillID)`）。
> **旁路数据层**：不进 effects_data 优先级链（手写 > effects_data > INI），
> 当前无引擎消费——消费接线按下方"路线"分期落地。

## 一、设计原则

1. **描述文本即策划语义**：官服把机制写进描述（含 HTML 语义标注），
   解析产物是"这套技能**该有什么**行为"的最全索引。
2. **不猜**：含关键词但无模板命中的段落整段进 `unparsed` 留痕；
   数值未标注（如"一段时间内"）按 0 记录不发明。
3. **颜色语义**：`#f9fc00` 黄=动作词（击飞/眩晕/倒地/拉近/霸体）、
   `#FFAE00` 橙/`#FF3030` 红=状态名（"惊雪""孽"，反查 buff 配置的权威来源）。
4. **分级差集**：全部等级各自解析，L1 为基准，数值变化进 `by_level`
   （点路径键，如 `["compile.max_hits"]`——寒心恨雪 L1 斩 3 次 → L6 起斩 8 次）。

## 二、动词三分类

### 1. 控制状态 `control{kind, dur_ms}` —— 改变目标行为状态

| kind | 中文触发词 | | kind | 中文触发词 |
|---|---|---|---|---|
| `stun` | 眩晕 | | `silence` | 封招 |
| `coma` | 昏迷 | | `ban_qinggong` | 禁轻功/无法使用轻功 |
| `dizzy` | 击晕 | | `slow` | 减速/移动速度减缓 |
| `launch` | 击飞 | | `stiffen` | 僵直 |
| `knockdown` | 击倒/倒地 | | `root` | 定身 |

### 2. 位移 `displacement{kind, distance_m}` —— 改变目标**位置**

- `push` 击退（**受害者位置后移**，远离施法者）
- `pull` 拉近（向施法者）；`distance_m=0` = 描述未标距离

### 3. 资源削减/回复 —— 作用于数值池

- `mp_drain` 流失内力（**目标减蓝**）｜`mp_restore` 恢复内力（0=未标数值）
- `heal` 恢复气血｜`dot` 持续伤害/内伤（生命持续损耗）

## 三、条目字段

```lua
E["<技能ID>"] = {
    compile = {                          -- 编译期静态轴（无触发条件）
        aoe = { aoe_shape = "circle|sector|rect", aoe_radius_m = 20, aoe_width_m = 5 },
        max_hits = 8,                    -- 最多斩击/连击 N 次（双击=2）
        cast_armor = "red|yellow",       -- 出招过程为红/黄霸体状态
        can_move = false,                -- （出招时不可移动）显式标注才出现
        can_qinggong = true, rush_note = true,   -- （可利用轻功快速接近）
        air = "only|never",              -- 只能针对空中目标 / 无法在空中施展
    },
    on_hit         = { { per_hit = true,           -- 每击
                         condition = { unparried = true, parry_broken = false,
                                       target_state = "X", unparryable = true },
                         control_kind = "stun", control_dur_ms = 2000, control_within_ms = 6000,
                         displacement_kind = "push", displacement_distance_m = 0,
                         mp_drain = true, mp_drain_dur_ms = 0, mp_restore = 0, heal = false,
                         apply_buff = "buf_…", apply_resolved = true,
                         apply_stacks = 2, apply_life_ms = 10000,
                         burst_at = 3, burst_kind = "coma", burst_dur_ms = 2000 } },
    on_parry       = { … },  -- 招架成功后（太极剑 06 恢复内力/旋刀 06 血刃）
    on_parry_break = { … },  -- 击破敌人招架时（旋刀 05 封招；破防技标记）
    on_victim      = { { stack_buff = "…", stack_per_hit = 1 } },  -- 感染族：被攻击则叠加
    on_periodic    = { { periodic_every_ms = 2000, periodic_aoe = true } }, -- 每 N 秒周围
    self_state     = { … },  -- 自身进入"X"状态（横行无忌光环）
    by_level = { [6] = { ["compile.max_hits"] = 8 } },   -- 数值差集（点路径键）
    unparsed = { "（防反招式…" },   -- 未解析段落留痕
}
```

Go 类型（`internal/skillscript`）：`DescEntry{Compile, OnHit/OnParry/
OnParryBreak/OnVictim/OnPeriodic/SelfState []*DescRule, ByLevel, Unparsed}`；
规则平铺字段与生成器键一一对应（`MPRestoreSet` 区分"恢复内力未标数值"的
显式 0）。状态名 → `buf_` 配置反查：`string_buff_desc.idres`，歧义按
技能族前缀（`buf_CS_by_xhmdl` > `buf_CS_`）消歧；无解 `apply_resolved=false`。

## 四、覆盖率快照（2026-11-13 首跑）

| 指标 | 数 | | 指标 | 数 |
|---|---|---|---|---|
| 技能总数 | 2,538 | | on_parry | 57 |
| Lua 条目 | 2,208 | | on_parry_break | 150 |
| on_hit 规则 | 1,264 (49.8%) | | on_periodic / on_victim | 1 / 1 |
| aoe | 738 (29.1%) | | self_state | 75 |
| cast_armor | 254 (10.0%) | | 状态反查 | 348/667 |
| air | 28 | | 含 unparsed | 677 (26.7%) |

unparsed 高频（v2 模板候选）：防反招式族（~59）、"招式层数反击伤害"（23）、
"射程增加/打断蓄力/驱散/打散怒气/回气时间"。金样本回归：`--selftest`
（旋刀 8 + 太极剑 5 + 寒心恨雪，13 技逐字段钉死）。

## 五、路线（引擎接线分期）

- **二期**：on_hit per_hit apply_state 端到端（分发点并联
  `dispatchNeiGongAttack` 同位置 skill_combat.go:1004；Effect 落账段
  neigong_dispatch.go:161-189 抽参共享）；`HitRatio`（M.on_hit）接进
  rollSkillHit；typed control——读 `buff_static.EffectType`（1=眩晕、
  3=击飞）区分控制类型，m169 目标帧接常规出招链。
- **三期**：on_parry / on_parry_break / on_victim / on_periodic 四分发点
  （player_actor.go:872-893 格挡三叉、combat_ai.go:574 周期）；
  **位移**（击退=NPC 位置后移帧——需官服 pcap 取证 0x21 vs 瞬移方言）；
  mp_drain 落账；无敌/DoT 指令；`chance` 预掷（rand 纪律同内功）。
- **取证后**：生成物『魔影』（召唤帧）、对空（NPC 腾空态协议位）、
  封招/禁轻功对目标下发（Cant* 目标方言）。
