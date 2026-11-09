# 服务端 Lua 脚本

本目录按**脚本模式**分两个子目录，各自有独立的装载器与错误策略：

| 目录 | 契约 | 装载器 | 状态 |
| --- | --- | --- | --- |
| `host/` | 事件型：`game.on(事件, handler)` 注册回调，低频交互 | `internal/luascript`（`-scripts`，默认本目录） | 已接线 `npc_service` |
| `module/` | 模块型：`local M = {} … return M`，暴露 `M.meta/M.levels` 与运行期钩子 | 按类各接（见下） | 四类均已装载；neigong 已接战斗分发，其余待接 |

`module/` 内为自参考服（`9yin-server`）拷贝的四类脚本（更新参考仓后重新拷贝同步，
只拷 `.lua`）：

| 路径 | 内容 | 装载 |
| --- | --- | --- |
| `module/neigong/buf_ng_*.lua` | 155 本内功特效模块 | `internal/neigong`（`-neigong-scripts`，默认本目录） |
| `module/item/*.lua` | 使用物效果 `M.on_use(ctx)`（3 本） | `internal/itemuse`（`-item-scripts`） |
| `module/skill/*.lua` | 技能编译覆盖 + 全技能效果表 effects_data（7 本，6433 条） | `internal/skillscript`（`-skill-scripts`）；effects_data 的 buff 组已接线技能目录编译（skill_catalog.go，优先级：手勘覆盖 > effects_data > INI 兜底） |
| `module/combat/formula.lua` | 伤害公式 `M.calc(c)` | `internal/combatformula`（`-combat-scripts`） |

四类模块宿主共用 `internal/modulevm` 内核（装载软语义、运行期坏脚本只禁用
该模块、指令预算按调用重置）；各查询的 nil 均表示"无脚本数据"，调用方回退
各自的数据层（INI 编译 / 内置公式），链路永不被脚本层拒绝。

引擎均为内嵌 Lua 5.5 运行时（`third_party/golua`，纯 Go）。脚本在沙盒内执行：
**没有 io / os / exec / debug / chan / time 模块**，每次调用有超时与指令预算。

## host/（事件脚本）

`host/` 下的 `*.lua` 在服务端启动时按文件名序加载（`-scripts` 指定目录，
默认即本目录；传空串禁用）。GM 面板的"重载脚本目录"按钮可热重载。
运行期出错只会让当次交互失败，随后 VM 连同脚本目录整体重建，不会影响
服务端进程（低频交互下可接受；这也是内功模块不走事件宿主的原因）。

### 事件注册

```lua
game.on("事件名", function(ctx) ... end)
```

同一事件多次注册以最后一次为准。处理器返回 `false` 表示交回默认路径
（例如维持"服务未实现"）；无返回值或真值视为已处理。

| 事件 | 触发点 |
| --- | --- |
| `npc_service` | 玩家在 NPC 菜单选中一项原生未实现的服务（任务/传送/职业/宝石/帮派/宅邸等 mark） |

### 调试输出

```lua
game.log("格式 %s=%d", "键", 42)   -- 进服务端日志，带 lua: 前缀
```

### 调用上下文 ctx（仅 npc_service）

- `ctx:player()` → `{ id, name, faction, silver, sp, scene, x, y, z }`
- `ctx:npc()` → `{ id, config_id, name }`
- `ctx:service()` → `{ mark, label, value, source }`
- `ctx:tasks()` → 该 NPC 绑定的任务数组（来自 `data/tasks.json` 任务目录，
  纯文本）：`{ id, title, target, can_accept_menu }`
- `ctx:notify(brief, chapter_name?)` → 给玩家发剧情提示卡
  （chapter_name 缺省 `ui_shop`，须是客户端已装的文本 ID；brief 可为任意文本）
- `ctx:bubble(text_id)` → 让该 NPC 头顶冒五秒说话气泡（text_id 须已装）

### GM 面板

"Lua 调试"卡片可在沙盒内执行任意代码并回显返回值（`return 1+2` → `3`），
输入上限 16KB。注意：卡片里没有 ctx，玩家相关副作用请写在事件处理器里。

## module/neigong/（内功特效模块，已接线战斗分发；**特效唯一权威**）

155 本模块（138 手写 + 17 生成桩），契约与参考服一致，详见
[docs/skill-script-framework.md](../docs/skill-script-framework.md) 的 neigong 节。
Go 侧分发点：受击（on_victim）、格挡（on_parry）、出招实中（on_attack，
经 skill 出招链）、战斗周期（on_periodic，1s 节拍/脱战自停）、常驻被动
（M.passive：幻玉 evasion_add 并入闪避、坐忘 damage_reduction_add 平减）；
效果消费见 `cmd/protocol-probe/neigong_dispatch.go`。曾并存的 desc 编译器
被动引擎（幻玉"料敌先机"完整版）已删除——战斗链内功行为统一由脚本决定。
要点：

- **模块形态**：`M.meta{id,name,book,category,max_tier}` + `M.levels` + 钩子；
  档位门 `tier_of` 在脚本内实现（cap = 3 + story_stage - 1）。
- **钩子**（Go 侧经 `neigong.Effects.Hook(buffID, event, ctx)` 调用，事件名
  带别名兜底）：`on_attack`（→`on_hit`）、`on_victim`（→`on_be_hit`）、
  `on_parry`、`on_dodge`（成功闪避事件，本仓扩展——移花接玉）、
  `on_periodic`（→`on_tick`）、`passive`（`Effects.Passive`）。
- **ctx**（Go 预填、只读；随机数 Go 预掷 `rand.r_proc`，脚本不掷骰）：
  `book_level/book_max_level/story_stage/now/damage/incoming/parried/crit/
  controlled/since_last_ms` + `self{hp,max_hp,mp,max_mp,attack}` +
  `target{oid,hp,max_hp}` + `attacker{oid}` + `state` + `rand{r_proc}`。
  **分发契约**：`on_attack` 必带 target；`on_victim`/`on_parry` 必带 attacker
  （个别脚本如 buf_ng_nl_001 无保护读 `ctx.attacker.oid`）。`state` 由 Go 侧
  持有并回写（无 state 时不设键，脚本侧 `ctx.state or {}` 兜底）。
- **返回指令白名单**（白名单外/畸形条目 WARN 丢弃）：`extra_damage{amount,kind}`、
  `reduce_pct`(0..1)、`shield{amount,life_ms}`、`dodge_add{amount,life_ms}`
  （闪避加成窗，本仓扩展——料敌先机；激活期间挂 BufferInfo 槽 10 原生
  Proc Buff 13107）、`heal`、`mp_restore`、`reflect`（数值或 `{amount,kind}`）、
  `state`（扁平标量表）、`parry_reduce{internal,external}`。

## module/item、module/skill、module/combat（使用链/技能目录/伤害公式接线前仅装载）

契约与参考服一致（ctx/返回白名单/RNG 预掷约定见
[docs/skill-script-framework.md](../docs/skill-script-framework.md)），Go 侧调用桥：

- `itemuse.Catalog.Use(itemID, ctx)` → `M.on_use`：heal_hp/heal_mp/
  open_box{drop_id,rolls}/grant_items(≤16)/temp_stat/consume(缺省 true)。
- `skillscript.Scripts`：`Overlay(id, def)` → `M.compile` 增量覆盖
  （self_buff_lease_ms/self_buff_slot/red_armor/stance_cycle/damage/
  self_buffs/target_effects）；`HitRatio(id, ctx)` → `M.on_hit` 倍率
  （夹 [0,100]，nil=不调整）；`EffectData(id)` → effects_data 条目快照
  （damage/self_buffs/target_effects/by_level 四种差集字段，显式 0 有效）。
- `combatformula.Formula.Calc(in)` → `M.calc`：输入攻守属性 + 预掷
  `rand[1]/rand[2]`，输出 `{damage,miss,crit,parried,reduction_pct}`；
  nil 时调用方回退内置公式。

## 热重载与错误策略（module/ 各域）

- 装载期坏文件：跳过该模块并汇总错误，不中断其余。
- 运行期坏脚本：**只禁用该模块**（WARN 一次），VM 不重建——爆炸半径
  =一个模块，调用链永不被脚本层阻断。
- GM `lua_reload` 五类宿主一并热重载，回执各域计数。

## Lua 5.5 注意事项

chunk 内一旦使用 `global x = 1` 声明，该 chunk 就进入严格模式——
所有全局引用（包括 `game`）都必须先声明。除非需要严格检查，
沿用传统的隐式全局即可（本目录示例即如此）。
