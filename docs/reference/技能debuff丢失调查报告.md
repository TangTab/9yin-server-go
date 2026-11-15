# 技能对怪 debuff 功能丢失调查报告

## 一、调查概述

**调查问题**：服务端原本对怪使用技能的时候添加对应技能的 debuff 功能丢失。

**调查范围**：
- 当前端：`h:\9yin-go-server2\cmd\protocol-probe\`
- 目标端（榴莲最新2026年8月25日）：`C:\Users\Administrator\Desktop\9yin-go-server2榴莲最新2026年8月25日\cmd\protocol-probe\`
- 备份版本：`pre_revert_20260826`、`2026-08-09-gm-merge`

**调查方法**：对比分析技能效果类型定义、技能编译逻辑、伤害结算流程、NPC 战斗状态结构、buff 系统等。

---

## 二、当前端已有的对怪效果类型

### 2.1 技能效果类型枚举

文件：`skill_combat.go` 第135-143行

```go
type skillEffectKind uint8

const (
    skillEffectDamage skillEffectKind = iota     // 0: 伤害
    skillEffectPlayerBuff                         // 1: 玩家自身 buff
    skillEffectTargetControl                      // 2: 目标控制（硬控）
    skillEffectRedArmor                           // 3: 红霸体
    skillEffectStanceCycle                        // 4: 姿态切换
)
```

**当前端对怪生效的效果只有 2 种**：
1. **skillEffectDamage**：对目标造成伤害
2. **skillEffectTargetControl**：对目标施加控制效果（硬控，如眩晕、定身等）

### 2.2 skillEffectTargetControl 的实现

文件：`skill_combat.go` 第1401-1419行 `applyNPCControl` 函数

```go
func (s *sceneLifecycle) applyNPCControl(ids []uint32, lifetime time.Duration, now time.Time) {
    // ...
    until := now.Add(lifetime)
    for _, rawID := range ids {
        state.controlledUntil = until
    }
}
```

**仅实现了硬控效果**：设置 NPC 的 `controlledUntil` 时间，让 NPC 在该时间内无法行动。
**没有**：
- 减速效果
- 减防/减攻效果  
- 持续伤害（中毒、流血等）
- 封招效果
- 其他 debuff 属性修改

### 2.3 NPC 战斗状态结构

文件：`combat_ai.go` 第23-48行 `npcCombatState` 结构体

```go
type npcCombatState struct {
    // ... 基础属性 ...
    controlledUntil time.Time  // 唯一的 debuff 相关字段：硬控结束时间
    // ... 其他字段 ...
}
```

**NPC 没有完整的 buff/debuff 系统**，只有一个 `controlledUntil` 字段用于硬控。

---

## 三、目标端有但当前端缺失的效果类型

### 3.1 技能效果类型对比

**结论：目标端与当前端的技能效果类型完全相同，都是 5 种。**

两端都没有 `skillEffectTargetBuff`、`skillEffectTargetDebuff` 等额外的对怪 debuff 类型。

### 3.2 实现方式差异

虽然效果类型相同，但两端的**技能命中系统实现方式有重要差异**：

| 对比项 | 目标端（榴莲版） | 当前端 |
|--------|-----------------|--------|
| 命中时间线 | `hits []zhaoshiHit` | `splitHits` + `hitTimes` + `hitActions` |
| 单目标伤害函数 | `settlePlayerSkillHit` | `scheduleSelectedSkillDamageTicks` |
| 延迟命中调度 | `schedulePlayerSkillHit` | `time.AfterFunc` 内联 |
| 延迟命中中应用 TargetControl | 有（第1059-1063行） | 无（只在技能释放时统一应用） |

### 3.3 关键差异：延迟命中的 TargetControl 应用

**目标端**（`skill_combat.go` 第1059-1063行）：
```go
for _, delayedEffect := range definition.effects {
    if delayedEffect.kind == skillEffectTargetControl {
        world.applyNPCControl(result.hitIDs, delayedEffect.buff.lifetime, time.Now())
    }
}
```

**当前端**：只在 `handleSkillCustom` 末尾统一应用一次（第1789-1795行），延迟命中回调中没有额外应用。

虽然这个差异对硬控效果影响不大（都是技能命中时立即应用），但说明当前端在重构技能命中系统时，**延迟命中的效果处理逻辑被简化了**。

---

## 四、debuff 丢失的具体原因分析

### 4.1 根本原因：NPC 缺少完整的 buff/debuff 系统

这是最核心的问题。当前端（以及目标端）的 NPC 战斗系统中：

1. **没有 NPC buff 列表**：玩家有完整的 24 槽 buff 系统（`player_actor.go` 中的 `buffSlots`），但 NPC 只有一个 `controlledUntil` 字段
2. **没有 debuff 属性修改机制**：无法实现减速、减防、减攻等效果
3. **没有 debuff 持续伤害机制**：无法实现中毒、流血等 DoT 效果
4. **没有客户端 debuff 显示**：NPC 没有类似玩家 `BufferInfo` 的属性更新协议

### 4.2 直接原因：有害 buff 全部被简化为硬控

文件：`skill_catalog.go` 第601-612行 `compileCombatSkill` 函数

```go
if buff, harmful, exists := compileExactSkillBuff(skillID, level, tables); exists {
    kind := skillEffectPlayerBuff
    if harmful {
        kind = skillEffectTargetControl  // 所有有害 buff 都被归类为硬控
    }
    definition.effects = append(definition.effects, combatSkillEffect{kind: kind, buff: buff})
}
```

其中 `harmful` 的判断来自 `compileExactSkillBuff` 函数：

```go
return skillBuffDefinition{...}, iniInt(static, "IsDamage") != 0, true
```

**问题**：
- 只有 `IsDamage = 1` 的 buff 才被认为是"有害的"
- 非伤害类 debuff（减速、减防、封招等）的 `IsDamage` 通常为 0
- 这些 buff 会被**错误归类为 `skillEffectPlayerBuff`**，即施加给玩家自己

### 4.3 技能 buff 的识别方式

文件：`skill_catalog.go` 第463-490行 `compileExactSkillBuff` 函数

```go
func compileExactSkillBuff(skillID string, level int32, tables skillResourceTables) (skillBuffDefinition, bool, bool) {
    configID := "buf_" + skillID  // 固定格式：buf_ + 技能ID
    buffNew := tables.buffNew[configID]
    // ...
}
```

技能对应的 buff 通过 `buf_` + 技能 ID 的命名约定来识别。例如：
- 技能 `CS_yhwq_hsqs04` → buff `buf_CS_yhwq_hsqs04`

**问题**：
- 每个技能只能有一个对应的 buff
- 如果技能有多个 debuff 效果（如同时减速和减防），无法通过此机制实现
- 技能的 buff 配置可能不完整或不存在

### 4.4 内功被动效果中没有对怪 debuff

文件：`neigong_passives.go`

内功被动效果只有两种模式，都是对玩家自身的：
1. `innerPowerModeIncomingDodgeHeal`：被攻击时闪避回血
2. `innerPowerModeOutgoingAttackLifeSteal`：攻击吸血

**没有**对怪物施加 debuff 的内功被动效果。

---

## 五、修复建议

### 5.1 方案概述

要完整实现"对怪使用技能时添加对应技能 debuff"功能，需要以下几个层面的修改：

1. **新增技能效果类型**：区分硬控和普通 debuff
2. **扩展 NPC 战斗状态**：支持 buff/debuff 列表
3. **修改技能编译逻辑**：正确识别和分类 debuff
4. **实现 debuff 应用逻辑**：技能命中时给 NPC 施加 debuff
5. **实现 debuff 效果结算**：减速、减防、持续伤害等
6. **实现客户端显示**：NPC debuff 图标显示

### 5.2 需要修改的文件清单

#### 5.2.1 `skill_combat.go` - 新增效果类型和 debuff 结构

**新增技能效果类型**：
```go
const (
    skillEffectDamage skillEffectKind = iota
    skillEffectPlayerBuff
    skillEffectTargetControl
    skillEffectRedArmor
    skillEffectStanceCycle
    skillEffectTargetDebuff  // 新增：对目标施加 debuff
)
```

**新增 NPC debuff 结构**：
```go
type npcDebuffInstance struct {
    configID   string
    staticData uint32
    level      int32
    expiresAt  time.Time
    // debuff 效果参数（从 buff_varprop 解析）
    slowPercent    int32  // 减速百分比
    defReduction   int32  // 减防数值
    damagePerTick  int32  // 持续伤害每跳
    tickInterval   time.Duration
    // ... 其他效果
}
```

#### 5.2.2 `combat_ai.go` - 扩展 NPC 战斗状态

**修改 `npcCombatState` 结构体**：
```go
type npcCombatState struct {
    // ... 现有字段 ...
    controlledUntil time.Time
    debuffs         []npcDebuffInstance  // 新增：debuff 列表
}
```

**新增 debuff 应用和结算方法**：
```go
func (state *npcCombatState) applyDebuff(buff skillBuffDefinition, now time.Time)
func (state *npcCombatState) tickDebuffs(now time.Time) (damage int32, speedMod float32, defMod float32)
func (state *npcCombatState) cleanupExpiredDebuffs(now time.Time)
```

#### 5.2.3 `skill_catalog.go` - 修改技能编译逻辑

**修改 `compileCombatSkill` 函数**：

当前逻辑（第601-612行）：
```go
if buff, harmful, exists := compileExactSkillBuff(skillID, level, tables); exists {
    kind := skillEffectPlayerBuff
    if harmful {
        kind = skillEffectTargetControl
    }
    definition.effects = append(definition.effects, combatSkillEffect{kind: kind, buff: buff})
}
```

需要修改为更精细的判断：
```go
if buff, buffType, exists := compileExactSkillBuff(skillID, level, tables); exists {
    var kind skillEffectKind
    switch buffType {
    case buffTypeSelf:
        kind = skillEffectPlayerBuff
    case buffTypeControl:
        kind = skillEffectTargetControl
    case buffTypeDebuff:
        kind = skillEffectTargetDebuff
    }
    definition.effects = append(definition.effects, combatSkillEffect{kind: kind, buff: buff})
}
```

同时需要修改 `compileExactSkillBuff` 函数，从 buff_static.ini 中读取更多字段来判断 buff 类型（而不仅仅是 IsDamage）。

#### 5.2.4 `skill_combat.go` - 实现 debuff 应用逻辑

**在 `handleSkillCustom` 中新增 debuff 应用**：
```go
for _, effect := range definition.effects {
    switch effect.kind {
    case skillEffectTargetControl:
        world.applyNPCControl(hitIDs, effect.buff.lifetime, now)
    case skillEffectTargetDebuff:  // 新增
        world.applyNPCDebuff(hitIDs, effect.buff, now)
    }
}
```

**新增 `applyNPCDebuff` 方法**：
```go
func (s *sceneLifecycle) applyNPCDebuff(ids []uint32, buff skillBuffDefinition, now time.Time) {
    s.mu.Lock()
    defer s.mu.Unlock()
    for _, rawID := range ids {
        id := worldcore.EntityID(rawID)
        state, exists := s.combatStates[id]
        if !exists {
            continue
        }
        state.applyDebuff(buff, now)
        s.combatStates[id] = state
    }
}
```

#### 5.2.5 `combat_ai.go` - 实现 debuff 效果结算

在 NPC AI 更新循环中（`combat_ai.go` 的 `tickCombat` 函数附近），添加 debuff 结算：

```go
func (s *sceneLifecycle) tickCombat(now time.Time) {
    // ...
    for id, state := range s.combatStates {
        // 结算 debuff 效果
        dotDamage, speedMod, defMod := state.tickDebuffs(now)
        if dotDamage > 0 {
            actor := s.combatActors[id]
            actor.ApplyDamage(dotDamage)
            // 发送 HP 更新
        }
        // 速度修改应用到移动速度
        // 防御修改应用到战斗公式
        // ...
    }
}
```

#### 5.2.6 客户端 debuff 显示

NPC 的 debuff 显示需要通过属性更新协议发送给客户端。这需要：

1. 在 `npcCombatProperties` 函数中添加 debuff 相关属性
2. 或者使用自定义消息通知客户端 NPC 的 buff 状态

这部分需要根据客户端的协议格式进一步调研。

### 5.3 修复优先级建议

| 优先级 | 功能 | 工作量 | 说明 |
|--------|------|--------|------|
| P0 | 修复 IsDamage 判断，正确分类 debuff | 小 | 确保非伤害类 debuff 不被错误施加给玩家 |
| P1 | 实现 skillEffectTargetDebuff 基础框架 | 中 | 新增效果类型 + NPC debuff 列表 + 应用逻辑 |
| P2 | 实现常见 debuff 效果（减速、减防） | 中 | 修改战斗公式，应用 debuff 属性修改 |
| P3 | 实现持续伤害（DoT）效果 | 中 | 添加 debuff tick 结算机制 |
| P4 | 客户端 debuff 图标显示 | 大 | 需要协议级别的支持 |
| P5 | 封招、沉默等高级 debuff | 大 | 需要修改 NPC AI 和技能系统 |

### 5.4 最小化修复方案（快速恢复基础功能）

如果只需要快速恢复"技能 debuff 能正确作用于怪物"的基础效果，可以先做以下修改：

1. **修改 `compileCombatSkill`**：确保所有有害 buff（不仅是 IsDamage=1 的）都被归类为对目标的效果
2. **扩展 `applyNPCControl`**：除了硬控，还支持减速等基础效果
3. **修改 NPC 战斗公式**：在 `npcStatsFromState` 中考虑 debuff 的属性修改

---

## 六、关键代码位置汇总

| 功能 | 文件 | 行号 | 说明 |
|------|------|------|------|
| 技能效果类型定义 | `skill_combat.go` | 135-143 | 5种效果类型枚举 |
| 技能编译（效果生成） | `skill_catalog.go` | 596-621 | `compileCombatSkill` 函数 |
| 技能 buff 解析 | `skill_catalog.go` | 463-490 | `compileExactSkillBuff` 函数 |
| 硬控应用 | `skill_combat.go` | 1401-1419 | `applyNPCControl` 函数 |
| 技能命中结算 | `skill_combat.go` | 1700-1795 | `handleSkillCustom` 中的效果循环 |
| NPC 战斗状态 | `combat_ai.go` | 23-48 | `npcCombatState` 结构体 |
| NPC 战斗属性 | `combat_resolve.go` | 58-86 | `npcCombatStats` / `npcStatsFromState` |
| 玩家 buff 系统 | `player_actor.go` | 98-166 | BufferInfo 槽位定义 |
| 内功被动效果 | `neigong_passives.go` | 28-57 | 仅两种对玩家自身的模式 |

---

## 七、结论

1. **当前端和目标端的技能效果类型基本一致**，都只有 5 种，都没有完整的 NPC debuff 系统。

2. **debuff 丢失的根本原因**是 NPC 战斗系统设计上只有硬控（`controlledUntil`），没有实现完整的 buff/debuff 机制。

3. **当前端在重构技能命中系统时**（从 `zhaoshi_hits` 模式改为 `splitHits/hitTimes` 模式），延迟命中的效果处理逻辑被简化，可能引入了一些回归问题。

4. **非伤害类 debuff 可能被错误归类**：`IsDamage = 0` 的 debuff 会被当作玩家 buff 处理。

5. **完整修复需要较大的工作量**，建议按照优先级逐步实现，先从修复分类错误和基础 debuff 框架开始。
