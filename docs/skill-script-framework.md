# 技能脚本框架（scripts/ Lua 契约）

`scripts/` 与 share 同法经 `ServerPaths`（env `NINEYIN_SERVER_ROOT` → 属性 → cwd 探测）定位；
引擎为 party.iroiro.luajava 4.1.0（lua55 natives，单 Lua 状态 + 全局锁——脚本是纯函数）。
客户端方言为 Lua 5.1，引擎对齐方案（lua55→lua51，4 文件 ~12 行，未执行）见
[lua-dialect-51.md](lua-dialect-51.md)。
**缺目录/natives 失败/执行异常 → 伤害公式回退 Java `CombatFormulas`、技能覆盖回退 INI 默认编译，
出招链路永不因脚本层不可用而拒绝**。

## 目录

| 路径 | 契约 | 状态 |
|---|---|---|
| `combat/formula.lua` | `M.calc(c) → {damage, miss, crit, parried, reduction_pct}` 伤害公式权威实现 | 战斗一期已接 |
| `skill/effects_data.lua` | **全技能效果汇总表**（生成器产出，`E["<技能ID>"] = {damage, self_buffs, target_effects, by_level}`） | 已接（2026-11-05 全技能效果接入，见下节） |
| `skill/<技能ID>.lua` | `M.compile(def) → overlay` 编译期增量覆盖 + `M.on_hit(ctx) → {ratio=...}` 出招倍率钩子 | 战斗一期 6 个覆盖；on_hit 已接线（2026-11-05） |
| `neigong/buf_ng_*.lua` | `M.on_attack/on_victim/on_parry/on_periodic|on_tick(ctx)` 内功 buff 特效运行期钩子 | 已接（战斗二期，2026-11-05，见下节） |
| `item/*.lua` | useitem/box/yaopin 使用物效果 | 已接（msg31 使用链，2026-11-04） |

### 内功脚本与 resources 数据对勘（2026-11-05）

数据源统一为 `E:/Workspace/project/game/resources`（盛大GM版客户端资源）：

- **134 本手写脚本数值与 res1 层级描述（`desc_buf_ng_<书>_<层>`）逐层对勘一致**，无需改动
  （`tools/skill-data/verify_neigong_scripts.py` 输出的差异均为单位换算/管线取舍的既有注释项）。
- **新生成 17 本**（`tools/skill-data/gen_neigong_scripts.py`，幂等可重跑，文件头带生成器标记）：
  八卦 `bg_001/002`、江湖 `jh_216~219/402`、极乐 `jy_101`、君子 `jz_101`、南海 `nh_001`、
  清风观 `qfg_001`、五仙教 `wxp_001~006`。`M.levels` 携带描述原文 + 常见参数解析
  （absorb/reduce_pct/duration_sec/layers 等），机制钩子返回 nil 留 TODO 待二期逐本细化。
- **28 本江湖系列（`jh_101~509`）为 res1 占位条目**（自引用 key、无文本），不生成。
- 4 本手写脚本（`jh_cygj/wwsg/wxxf/zxg`）在 res1 无名字 key，保留原值。

### 技能数据计算（tools/skill-data/）

`compute_skill_catalog.py` 从 res1 全链编译（语义逐条对齐 Java `CombatSkillCatalog`）：

- `docs/text-extract/data/skill_catalog.jsonl.gz`：**17,566 技能 × 等级 = 59,891 行**——
  CD（个人/公共）/消耗（HP/MP/SP/QGP）/编译伤害与倍率/目标模式与形状/冲锋/锁定族判定/
  精确 buff（`buf_<技能ID>` 租约 buff_new→varprop→24h 兜底）/武器门控/动作时间线与多段判定帧/
  武技族标记（`wuji_relevance`/`wuji_combo`）。CD 抽检与抓包实证全对齐
  （tjq05=15000 / tjq07=40000 / jysg01=2000）；`wuji_combo` 全表仅 `CS_jh_myjf08`，
  与 Java 判据（LevelupRelevance=wuji_* ∧ PackCategory=2）一致。
- `docs/text-extract/data/wuji_catalog.json`：武技目录——322 门（wuji_learn）、品质
  （wuji_quality，28 门有显式品质）、65 条激活动作（wuji_action，含多变体 StaticData）、
  槽位类型与重置费用、挂接技能 258（LevelupRelevance=wuji_*）、连击怒招清单。
- `docs/text-extract/data/neigong_effect_index.json`：179 本内功书索引
  （名称/门派/StaticData/层级描述原文/脚本覆盖状态）。
- res1 无 `skill_init.ini`：修炼进度族字段（fill/total/pause_time）不计算，行内为 null。

## formula.lua（伤害公式）

输入 `c`（只读 table，Java 预填）与输出字段见脚本头注释；要点：

- `rand = {r_miss, r_crit}`：**随机数由 Java 预掷**传入（脚本不掷骰 → 可测试、与 Java 兜底逐位一致，
  `LuaFormulaParityTest` 网格对照钉死）。掷骰顺序：未命中 → 暴击（招架不免疫）→ 招架穿透 → 平减 → 保底 1 点。
- `ratio`：技能倍率 = 编译伤害/100/波数（`CombatSkill.damageRatio()`，damage_calculate.ini r 段求和）。
- Java 兜底 `CombatFormulas`（`server/script/`）同常量同顺序；调参改 Lua 后需同步常量区，
  parity 测试会抓不同步。
- 当前守方侧输入（defense/闪避/暴击值等）取缺省 0——属性域补全后由
  `SkillCastHandler.rollDamage` 组装处接线（meleeDamageRange 已从装备+内功取攻方掷值区间）。

## skill/effects_data.lua（全技能效果汇总表，2026-11-05 接线）

**所有技能的效果接入 Lua 的主体路径**：`tools/skill-data/gen_skill_scripts.py` 从
share/skill ini（与 `CombatSkillCatalog` 同一份表，语义逐条复刻
`compileSkillDamage`/`compileExactSkillBuff`）生成单文件汇总表，启动时装载为全局
`SkillEffectsData`（`LuaScriptService.initLocked`，装载失败仅降级 INI 编译不碰公式引擎）。

条目形态（只收录有内容的技术——L1 有编译伤害或有精确 buff；6,697 条 ≈ 2.4MB）：

```lua
E["CS_wd_tjq01"] = {
  damage      = { amount = 112, waves = 1 },   -- 编译伤害/正段数（damage_calculate r 段第 3 段求和）
  target_effects = { { config_id = "buf_CS_wd_tjq01", static_data = 6069,
                       level = 1, life_ms = 10000, control = true } },  -- 有害精确 buff → 目标控制
  by_level    = {                              -- 仅与 L1 有差的级（3,103 技能伤害随级变/675 租约随级变）
    [5] = { damage_amount = 205 },
    [10] = { damage_amount = 341 },
  },
}
-- 有益精确 buff → self_buffs = { { config_id=, static_data=, level=, life_ms= } }
```

- **by_level 字段 nil 判别**：`damage_amount = 0` 是显式有效值（该级无伤害，如
  wx_jh_xl01@2 的 damage_calculate[61] 负段），Java 侧 `readOptionalLong` 区分
  「显式 0」与「缺字段」——缺字段沿用 L1 基准。
- **解析优先级**：逐技能 `<ID>.lua` M.compile（整组取代数据条目）→ effects_data
  条目（含 by_level[level] 差集修补）→ null（INI 默认编译）。缓存键 `id@level`。
- **parity 钉子**：`SkillEffectDataParityTest` 双根对照——同一份 share 表，
  无 scripts 根（INI 兜底编译）vs 本仓 scripts 根（Lua 数据驱动编译），全部技能×
  {1,2,5} 级逐字段 diff 必须为 0（手写覆盖技能除外，其覆盖行为由
  `CombatSkillCatalogTest` 断言）。辅助勘验 `tools/skill-data/verify_skill_scripts.py`
  对 res1 jsonl 交叉核对（结构差异非零退；res1 伤害数值 5 族系统性漂移 ~25% 为
  表版本差异，仅列出）。
- **重新生成**：`python tools/skill-data/gen_skill_scripts.py`（幂等；表更新后重跑
  + `reload_scripts` 或重启生效）。

## skill/<技能ID>.lua（编译覆盖 + on_hit 钩子）

`CombatSkillCatalog` 编译每个技能时调用 `LuaScriptService.skillOverlay(id, level, exactBuff)`；
`def` 输入 = INI 编译出的 `buf_<技能ID>` 精确 buff 快照（`config_id/static_data/level/lifetime_ms`），
返回增量覆盖（对应 Go verifiedSkillEffectOverlays 硬编码的迁移）：

```lua
function M.compile(def)
  return {
    self_buff_lease_ms = 35000,   -- 自 buff 租约覆盖（毫秒）
    self_buff_slot     = 5,       -- 指定 BufferInfo 槽位（0/缺省=自动找空槽）
    red_armor          = true,    -- 红霸体：起手霸体帧，租约=动作总时长（槽1/8682）
    stance_cycle = {              -- 架势轮换：逐次施放循环变体（如太极拳四式 守/攻）
      variants = {
        { id = "buf_CS_wd_tjq04_1", static_data = 6013, level = 1,
          life_ms = 86400000, slot = 7 },
        { id = "buf_CS_wd_tjq04_2", static_data = 6055, level = 1,
          life_ms = 86400000, slot = 7 },
      },
    },
    -- 完整效果集（2026-11-05）：
    damage = { amount = 540, waves = 4 },        -- 伤害/段数覆盖（取整组时覆盖 INI 编译值）
    self_buffs = {                                -- 多自 buff（整组取代精确 buff 的自增益效果）
      { config_id = "buf_CS_xxx", static_data = 6013, level = 1, life_ms = 35000, slot = 5 },
    },
    target_effects = {                            -- 目标效果（整组取代有害精确 buff 的目标控制）
      { config_id = "buf_CS_yyy", static_data = 7001, level = 1,
        life_ms = 8000, control = true, frame = true },
      -- control=true：受控到租约终点（fight.control，同 INI 有害精确 buff 语义）
      -- frame=true：加发 msg169 目标 debuff 帧（武技 169 帧的脚本化泛化，默认关）
    },
  }
end
```

`M.on_hit(ctx)` 运行期倍率钩子（出招每次掷值时调用，`SkillCastHandler.rollDamage`）：
`ctx = {ratio=本波倍率, level=技能等级, rand={r_adjust=Java 预掷}}`（每调用恒 1 枚掷值，
无钩子技能 RNG 序列零扰动；随机数语义同 formula.lua 约定——互斥分支区间法/派生第二掷），
返回 `{ratio = ...}`（夹 [0,100]，nil/非正数=不调整）：

```lua
function M.on_hit(ctx)
  if ctx.level >= 5 and ctx.rand.r_adjust < 0.30 then
    return { ratio = ctx.ratio * 1.5 }   -- 5 级以上 30% 概率本波 ×1.5
  end
end
```

坏脚本隔离：`on_hit` 运行期异常只禁用该技能脚本（`skillBroken` 集，WARN 一次），
不 markBroken 公式引擎，compile 契约不受牵连。

已接覆盖脚本：`CS_yhwq_hsqs04`（35s/槽5）、`CS_yhwq_hsqs05`（槽6+红霸体）、`CS_wd_tjq04`（架势轮换）、
`CS_wd_tjq05`（槽8）、`CS_wd_tjq07`（槽9）、`CS_wd_tjq08`（红霸体）。未列技能走 effects_data → INI 默认编译。

## item/&lt;configId&gt;.lua（使用物效果，msg31）

> 探索取证全录（客户端右键链路/宝箱数据模型/掉落包全景/参考实现盘点/决策记录）：[item-use-msg31.md](item-use-msg31.md)

`LoginFlow.useItemEffect`（msg31 不可穿戴/非卡片分支）调
`LuaScriptService.itemUse(configId, ctx)`；**解析优先级：Lua 脚本 → INI 数据回退
（`server/game/item/ItemUseCatalog`）→ 均无则维持 use-chain not implemented 日志**。
脚本返回 nil / 无脚本 / 引擎坏 → 一律走 INI 回退，使用链不拒绝。

`M.on_use(ctx)` 的 `ctx`（只读，Java 预填）：`item_id/amount/now` +
`self{hp,max_hp,mp,max_mp,silver}` + `rand{r_drop,r_proc}`（随机数 Java 预掷）。返回指令白名单：

```lua
function M.on_use(ctx)
  return {
    heal_hp    = 300,                            -- 回气血（Actor.restore，满血闸掉）
    heal_mp    = 100,                            -- 回内力
    open_box   = {drop_id = "xx", rolls = 1},    -- 按 share/drop/drops_mvp.json 权重开包（1..10 roll）
    grant_items = {{id = "sd_001", amount = 2}}, -- 确定性直接给物品（≤16 条）
    temp_stat  = {attack_add = 25, defense_add = 0, max_hp_add = 0,
                  life_ms = 86400000},           -- 限时属性（已解析未接：仅剩此效果时不消耗）
    consume    = true,                           -- 是否消耗 1 个（默认 true）
  }
end
```

INI 回退（`ItemUseCatalog.resolve(configId, itemType)`，share/item 下三表）：
`giftbox.ini[configId] DropID`（项目自建 UTF-8 表）→ `tool_item.ini` 的
`script=BoxItem`+DropID 开包 / `script=CompensateItem` 每个 `@CompensateItemRec` ref
各开一次（原版 func-56 条件门控未接，简化为全发）→ `item_use_skill_static.ini[itemType]`
回血阶梯（yaopinhp* 无 varprop 数值，L1..L9=100..3000，MP 减半）。掉落包体
`DropPacks`（share/drop/drops_mvp.json）权重抽 1 条/roll、数量 min..max、同物品合并。

落账顺序对齐 bak SceneFlow.useItem：效果先行（`grantBagItem` 容量感知、
回血+`vitalUpdate`），随后消耗 1 个（`consumeBagItemOne` 帧对，归零删行+镜像清位）；
会话级 800ms 节流防连点；随机数取会话 `combatRandom`（测试可换种子）。
已接脚本：`box_chinajoy_2026_giftbox`（开包）、`yaopin_sc_1001`（回血 100）、
`useitemhs_dy_06`（限时属性，效果层待二期）。未列物品无脚本 → INI 回退。

## 热重载

GM 面板 `reload_scripts`（GmRemote 白名单 + applyGmCommand）：重建 Lua 状态并清 overlay/itemUse/neigong 缓存，
下一次出招/使用即生效；重载失败自动回退 Java 管线并回执失败原因。

## neigong/buf_ng_*.lua（内功特效运行期钩子，战斗二期 2026-11-05 接线）

分发器 `world/fight/NeiGongEffects`（装备书 `progress.book(curNeiGong).buffId` → 脚本），
经 `LuaScriptService.neigongHook(buffId, hook, ctx)` 每次事件调用（模块全局仅装载一次）。
155 本（138 手写 + 17 生成桩）；28 本 jh 占位无脚本自然跳过。

**钩子映射**（生成桩的 `on_hit`/`on_be_hit` 按 attacker/defender 侧别名兜底探测）：

| 战斗事件 | 接线点 | 钩子 |
|---|---|---|
| 玩家实中目标（单段逐目标 / 多段逐拍逐目标；miss 不触发） | `SkillCastHandler.settleTargets` / `scheduleMultiHits` | `on_attack`（→`on_hit`） |
| 玩家被 NPC 命中（未格挡） | `NpcCombatAi.tick` | `on_victim`（→`on_be_hit`） |
| 玩家格挡下 NPC 攻击（当前管线格挡零伤，只落 state/heal/mp/shield） | `NpcCombatAi.tick` | `on_parry` |
| 战斗期间周期（1s 节拍；脱战自停，flag 复位待下次 playerEngaged） | `NeiGongEffects.ensureTicker` | `on_periodic` / `on_tick` |

**ctx**（只读，Java 预填；`rand.r_proc` Java 预掷同 formula.lua 约定，脚本不掷骰）：
`book_level/book_max_level/story_stage/now/damage/incoming/parried/crit/since_last_ms` +
`self{hp,max_hp,mp,max_mp,attack}` + `target{oid,hp,max_hp}` + `attacker{oid}` +
`state`（上一次有效触发返回的扁平表，Java 侧 PlayerActor 持有；**无 state 时不设该键，
脚本侧须 `ctx.state or {}` 兜底**）+ `rand{r_proc}`。
档位门（`tier_of`：cap=3+story_stage-1）随 `story_stage` 开档——九阴志系统落地前取
`-Dnineyin.neigong.storyStage`（默认 1）。

**rand 每调用仅 1 枚掷值（`r_proc`），数量按契约写死**（formula=2 `r_miss/r_crit`、
item=2 `r_drop/r_proc`、neigong=1）。Lua 侧无回掷通道：不暴露 Java RNG 桥，`math.random`
虽随 `openLibraries` 可用但脚本层禁用（会破坏可测试/与 Java 兜底逐位一致/无钩子零扰动
三条契约，现有 155 本脚本零使用）；res1 语料 179 本逐本核对亦无一枚以上掷值需求。
**效果需要多枚随机数时的两条路线（2026-11-05 记录，待定——出现真实需求前按 A 兜底）**：

- **路线 A（契约内，不改 Java）**：①互斥分支区间法——互斥结果一枚够用
  （`r<0.10`→效果甲、`r<0.25`→效果乙）；②派生第二掷
  `r2 = (r*9301+49297) % 233280 / 233280`（同 `r_proc` 必同结果，确定性/可测试不破）；
  ③magnitude 不吃随机——从 `book_level/damage/since_last_ms/state` 确定性算（现行脚本的普遍做法）。
- **路线 B（扩契约，改 Java）**：`NeiGongCtx` 加字段 + `buildNeigongCtxTable` 多填 +
  分发器多掷。硬约束：**每个 hook 的消费数量必须固定**（不得按脚本行为动态变，否则共享
  RNG 序列被扰动、"无钩子书零消费"失效、parity 测试无从写），照 formula 固定数组模式
  （如 on_attack 恒 2 枚）。

**返回指令白名单**（非法条目 WARN 丢弃，不打断战斗链）：
`extra_damage{amount,kind}`（并入当次落血值与飘字/真气；kind 现管线统一外功仅透传）、
`reduce_pct`（0..1，折算 incoming）、`shield{amount,life_ms}`（PlayerActor 盾池 FIFO 吸收，
到期即弃；**全吸收=格挡式静默零帧**，被击回怒照常）、`heal`/`mp_restore`（`restore`+0x10 vitalUpdate）、
`reflect`（反打 NPC，击杀走 `onNpcKilled` 常规链）、`state{...}`（扁平表回写，互斥/周期窗口所在；
会话内存态不落 FacultyStore 快照——窗口多为 6~20s，跨重启无意义）、
`parry_reduce{internal,external}`（解析占位：管线格挡零伤无外功/内功之分）。

**降级与隔离**：无书/无脚本/坏脚本/引擎不可用 → 静默零效果，出招与伤害链永不被阻断；
**坏脚本只禁用该 buffId**（per-script 禁用集），绝不 markBroken 公式引擎；
无钩子书不消费随机数（既有 RNG 序列零扰动）。`M.passive`（27 本静态属性，需并入
`CombatFormulas.Input` 守方侧与资源上限计算）与护盾打破回内力（wd_001 层4）留待下一期。

**测试**：`script/LuaNeigongHookTest`（Lua 契约：proc/state 互斥窗/周期门/档位门）、
`world/fight/NeiGongEffectsFlowTest`（管线：盾吸收/附伤窗/ticker/reflect 击杀/被击帧回归；
临时根 junction 拼 GO 仓 resources + 本仓 scripts，AfterAll 还原）。

## 运行期钩子（已接线）

- `M.on_hit(ctx)`：出招倍率钩子（上文 skill 节；`SkillCastHandler.rollDamage` 接线，
  每调用 Java 预掷 1 枚 `r_adjust`，掷骰序固定在 miss/crit/区间掷值之后——无钩子
  技能 RNG 序列零扰动）。CS_wd_tjq04.lua 的注释示例可参考。

## Go 服务端接入状态（2026-11-06；2026-11-09 扩至全部四类并接线战斗分发/技能 buff 编译）

本仓（9yin-go-server1）已接入**全部四类模块脚本层**，其中 **neigong 已接线
战斗分发**（`cmd/protocol-probe/neigong_dispatch.go`，参考服 NeiGongEffects
的 Go 等价物）、**skill 的 effects_data 已接线技能目录编译**（buff 组，
2026-11-09，见下）；item/combat 域中 item 仍为装载 + 调用桥
（combat 公式由 combatcalc 每击引擎单独消费）：

- **脚本入仓**：参考服 `scripts/{combat,item,neigong,skill}` 四类拷贝至本仓
  `scripts/module/` 下；事件脚本迁至 `scripts/host/`（`-scripts` 默认改指该
  目录，见 scripts/README.md）。
- **共享内核**：`internal/modulevm`——模块式装载（捕获 `return M` 表）、
  **运行期坏脚本只禁用该模块**（VM 不重建、指令预算按调用重置——多人下
  一次坏调用最坏一次 ≤500ms 持锁停顿后即隔离）、WarnOnce 防热路径刷屏、
  `EachModule` 域校验回调（读 M.meta 建目录）。与事件宿主
  （`internal/luascript`，game.on/整机重建）分立。
- **四类域包**（均薄封装 modulevm）：
  - `internal/neigong`（`Effects`）：`Hook(buffID, event, ctx) *Effect`
    （别名链 on_attack→on_hit / on_victim→on_be_hit / on_periodic→on_tick；
    返回白名单同上节，reflect 兼容双形态）、`Passive`（扁平数值属性表）、
    `Books/Book/ByBook` 目录。ctx 另含脚本实测消费的 `controlled`；
    `on_attack` 必带 target、`on_victim`/`on_parry` 必带 attacker 的分发
    契约由调用方保证（buf_ng_nl_001 无保护读 `ctx.attacker.oid`）。
  - `internal/itemuse`（`Catalog`）：`Use(itemID, ctx) *Effect` → `M.on_use`
    （heal_hp/heal_mp/open_box/grant_items≤16/temp_stat/consume 缺省 true）；
    `Items/Item` 目录。
  - `internal/skillscript`（`Scripts`）：`Overlay(id, def)` → `M.compile`
    增量覆盖（self_buff_lease_ms/self_buff_slot/red_armor/stance_cycle/
    damage/self_buffs/target_effects）；`HitRatio(id, ctx)` → `M.on_hit`
    （夹 [0,100]，nil=不调整）；`EffectData(id)` → effects_data **装载期
    整表快照**（6433 条；by_level 差集字段实测四种：damage_amount/
    damage_waves/target_life_ms/self_life_ms，显式 0 有效）。
  - `internal/combatformula`（`Formula`）：`Calc(in) *Output` → `M.calc`
    （输入攻守属性 + 预掷 `rand[1]/rand[2]`；nil 时调用方回退内置公式）。
- **接线**：`cmd/protocol-probe` 五 flag（`-scripts` + `-neigong-scripts`/
  `-item-scripts`/`-skill-scripts`/`-combat-scripts`，默认 `scripts/module/`
  对应子目录，空禁用），`scriptHosts` 聚合（script_hosts.go）；启动日志各域
  `ready count=…`；GM `lua_reload` 五类一并热重载并回报各域计数。
- **neigong 战斗分发**（neigong_dispatch.go）：装备书 varprop `buffID` 即
  模块键；`hookNeiGong` 骨架在 p.mu 内取快照（书/自身/state/盾池）、锁外
  调脚本（modulevm 叶子锁，微秒级；坏书只禁用该本）、回锁落账。四个分发点：`resolveNPCAttack`
  命中→`on_victim`（reduce_pct 折减 → 盾池 FIFO 吸收 → 落血；heal/mp_restore
  落账；reflect 经 outcome 反打 NPC 走常规伤亡链）、格挡→`on_parry`
  （零伤，只落 state/heal/mp/shield）；`applySelectedSkillDamage`→`on_attack`
  （extra_damage 并入落血值）；`simulateLocked` 交战成员→`on_periodic`
  （1s 节拍、脱战自停）。state 存 playerActor（会话内存态，换书清空）；
  盾池 FIFO 到期即弃；r_proc 用 combatPercentRoll 确定性预掷（盐 31..34
  与既有判定独立）。九阴志档位门缺省阶段 1（`neigongStoryStage` 常量，
  九阴志系统落地后接全服进度）。宿主 nil（测试/禁用）→ 全部分发点零效果，
  行为与接入前逐位一致。
- **skill effects_data 接线技能目录编译**（2026-11-09）：buff 效果优先级
  对齐参考服 CombatSkillCatalog——手勘覆盖（verifiedSkillEffectOverlays，
  6 技含槽位/红霸体/架势轮换，行为钉死不改）> **effects_data 条目**
  （`skillScripts.EffectData`：self_buffs[] → skillEffectPlayerBuff 原生
  自增益挂槽 11..24、target_effects[] → skillEffectTargetControl 出招结算
  对命中 NPC 施加租约控制；by_level[level] 的 self_life_ms/target_life_ms
  逐级覆盖租约）> INI 单条 exact_buff（buf_<技能ID> 兜底）。条目存在即
  **整组取代** INI（含 damage-only 条目压制 INI 24h 假租约兜底）。
  覆盖增益：INI exact_buff 仅覆盖 45% 玩家技（5951/13062 L1），effects_data
  补齐其中 1233 个无 buff 家族技 + 1145 条多 buff 条目。伤害数值不取
  effects_data（damage_amount/by_level 伤害差集不并入——数值以 combatcalc
  每击公式为权威）。GM `lua_reload` 重载 effects_data 后清技能编译缓存
  （resetCache），下一次出招重编译。宿主 nil → 回落 INI，行为与接入前
  逐位一致。
- **统一以参考服脚本为权威**（2026-11-09）：删除 Go 侧自研的 desc 编译器
  被动引擎（neigong_passives.go——曾从现代客户端描述文本编译幻玉"料敌
  先机"Proc 闪避+闪避治疗，参考服脚本中该机制为 TODO），战斗链内功行为
  全部来自脚本：`M.passive` 静态属性经代数缓存（modulevm.Generation，
  热重载/换书/升级失效）接入受击裁定——坐忘功 `damage_reduction_add` 平减
  （floor 1 点）；其余被动键（max_mp_add 等）缓存待后续属性域消费。
- **本仓白名单/钩子扩展**（把参考服 TODO 机制落进脚本，2026-11-09）：
  - `dodge_add{amount,life_ms}` 指令：闪避加成窗（料敌先机）。Go 侧窗状态
    与盾池同构（到期即弃、Proc 刷新覆盖）；窗激活期间挂客户端可见原生
    Proc Buff（BufferInfo 槽 10 = buff_new.ini [buf_ng_yh_001_5]
    StaticData **13107**，varprop→proppack 链含 DodgeProbAdd=20），战斗
    tick 过期清扫摘除并推 changed 帧。
  - `on_dodge` 钩子（受击裁定闪避分支派发）：成功闪避事件——移花接玉
    （heal/mp/shield/state；闪避无伤故 extra_damage/reduce_pct 不适用）。
  - **幻玉神功 buf_ng_yh_001 完整实现**（数值逐层勘定自 desc2.idres
    desc_mind_buf_ng_yh_001_\<层\>_ngtips）：料敌先机 50% Proc → +20% 闪避
    5-11s（36-26s cd）；移花接玉闪避回 2-5% 最大气血（4s cd）；层 6-7
    下一击 +60% 经 state 充能、on_attack 附伤兑现（ctx.damage×60% 平减取整）。
    原常驻 evasion_add 近似已移除（真实机制取代）。与参考仓同步该文件时
    注意：参考仓版本仍为 TODO 桩。
- **测试**：各域单测 + 真实目录钉子——neigong：155 本 × 4 事件 × 双 r_proc
  全量冒烟零禁用 + 坐忘功 passive/周期盾；item：3 本真实效果（祛风散回血/
  惊鸿礼盒开包/飞云剑匣限时属性）；skill：太极拳四式架势轮换覆盖、
  CS_wd_tjq01 条目与 by_level 逐级差集、karma_jh_cqgf01 显式 0；formula：
  12 组黄金用例（未命中/暴击/对数外防/完全招架/穿透/平减/保底/倍率）手算自
  脚本常量区钉死。
