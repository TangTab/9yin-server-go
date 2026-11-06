-- 伤害公式（CombatFormulas.java calc 的逐行移植；调参改这里，GM reload_scripts 热更）。
-- 系数同源：17173《九阴真经伤害计算：暴伤与暴免》《暴击率与暴击值》(2024-11)、
-- 《坑爹的外功防御》对数拟合 (2013-01)、NGA 伤害框架帖 (2023-07)、
-- Age of Wushu Wiki「Accuracy & Evasion」。
--
-- 输入 c（只读 table，字段由 Java 预填）：
--   attack/defense/pierce/internal_conversion      攻方（化劲=internal_conversion）
--   accuracy/evasion/hit_rate_bonus/dodge_rate_bonus
--   crit_value/resilience/crit_rate_bonus/crit_rate_reduction
--   crit_damage_bonus/crit_damage_reduction
--   damage_reduction/block_rating                  守方
--   ratio        技能倍率（编译伤害/100/波数；<=0 按 1.0）
--   parrying     守方是否招架
--   rand = {r_miss, r_crit}   Java 预掷随机数（脚本不掷骰，保证可测试/与 Java 对照）
--
-- 输出：{damage, miss, crit, parried, reduction_pct}
--   掷骰顺序：未命中 → 暴击（招架不免疫暴击）→ 招架穿透 → 平减 → 保底 1 点。

local M = {}

-- 系数常量区（与 CombatFormulas.java 常量一一对应）
local DEFENSE_SLOPE = 20.0        -- 减伤曲线斜率：每 e 倍外防 +20%
local DEFENSE_INTERCEPT = -44.0   -- 20·ln(外防/100) − 44
local DEFENSE_LOG_SCALE = 100.0
local MAX_REDUCTION = 0.75        -- 减伤上限
local CRIT_BASE = 0.05
local CRIT_SLOPE = 0.10
local MAX_BASE_CRIT = 0.15        -- 基础暴击率渐近上限（%加成另计不受此限）
local BASE_CRIT_DAMAGE = 1.0      -- 基础暴伤 100%（双倍）
local MISS_BASE = 0.10
local MISS_SLOPE = 0.15
local MAX_DODGE = 0.24            -- Wiki：Evasion cannot exceed 24%
local REDUCTION_PER_DAMAGE = 10   -- 每 10 点减免属性折 1 点伤害

local function clamp(v, lo, hi)
  return math.max(lo, math.min(hi, v))
end

-- 外防 → 减伤比例 [0, 0.75]；外防 ≤ 0 或低于 ~900 时不减伤
local function defenseReduction(defense)
  if defense <= 0 then
    return 0
  end
  local pct = (DEFENSE_SLOPE * math.log(defense / DEFENSE_LOG_SCALE) + DEFENSE_INTERCEPT) / 100.0
  return clamp(pct, 0, MAX_REDUCTION)
end

-- 未命中率：10% + 15%×(闪避−命中)/(闪避+1)，合计夹 24% 上限后抵扣命中率%
local function missChance(accuracy, evasion, hitRateBonus, dodgeRateBonus)
  local ratio = (evasion - accuracy) / (evasion + 1)
  local dodge = math.min(MAX_DODGE, MISS_BASE + MISS_SLOPE * ratio + dodgeRateBonus)
  return clamp(dodge - hitRateBonus, 0, 1)
end

-- 暴击率：基础 5% + 10%×(暴击值−坚韧)/(暴击值+1)，加算 % 加成后夹 [0,1]
local function critChance(critValue, resilience, critRateBonus, critRateReduction)
  local ratio = (critValue - resilience) / (critValue + 1)
  local base = clamp(CRIT_BASE + CRIT_SLOPE * ratio, 0, MAX_BASE_CRIT)
  return clamp(base + critRateBonus - critRateReduction, 0, 1)
end

-- 暴击倍数：1 + max(0, 基础 100% + 暴伤% − 对手暴伤减免%)
local function critMultiplier(critDamageBonus, critDamageReduction)
  local bonus = math.max(0, BASE_CRIT_DAMAGE + critDamageBonus - critDamageReduction)
  return 1 + bonus
end

-- 招架穿透比 = 1 − 格挡值/化劲，夹 [0,1]；化劲 ≤ 格挡值时完全挡下
local function parryMultiplier(internalConversion, blockRating)
  if internalConversion <= 0 or internalConversion <= blockRating then
    return 0
  end
  return clamp(1 - blockRating / internalConversion, 0, 1)
end

function M.calc(c)
  local rMiss, rCrit = c.rand[1], c.rand[2]
  local miss = missChance(c.accuracy, c.evasion, c.hit_rate_bonus, c.dodge_rate_bonus)
  if miss > 0 and rMiss < miss then
    return { damage = 0, miss = true, crit = false, parried = false,
             reduction_pct = defenseReduction(c.defense) }
  end
  local reduction = defenseReduction(c.defense)
  local raw = (c.attack * (1 - reduction) + c.pierce)
      * (c.ratio <= 0 and 1.0 or c.ratio)
  local crit = false
  local chance = critChance(c.crit_value, c.resilience,
      c.crit_rate_bonus, c.crit_rate_reduction)
  if chance > 0 and rCrit < chance then
    raw = raw * critMultiplier(c.crit_damage_bonus, c.crit_damage_reduction)
    crit = true
  end
  local parried = false
  if c.parrying then
    raw = raw * parryMultiplier(c.internal_conversion, c.block_rating)
    parried = true
  end
  -- 平减：每 10 点减免属性折 1 点（对非负值 floor 与 Java int 除法等价）
  local flat = math.floor(math.max(0, c.damage_reduction) / REDUCTION_PER_DAMAGE)
  local damage = math.floor(raw) - flat
  if parried and parryMultiplier(c.internal_conversion, c.block_rating) == 0 then
    damage = 0   -- 完全招架：无穿透伤害（也不吃保底 1 点）
  else
    damage = math.max(1, damage)
  end
  return { damage = damage, miss = false, crit = crit, parried = parried,
           reduction_pct = reduction }
end

return M
