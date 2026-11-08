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
--
-- ── 每击公式路径（internal/combatcalc 宿主，2026-11-08 数值重写批次）──
--
--   M.install_formulas(formulas, bindings)：装载期注入 formula.ini 转译闭包
--     formulas[formula_id] = function(PV, R, Q) return <转译表达式> end
--     bindings[formula_id][n] = { src=码, prop=名 }（P1..Pn 来源绑定）
--   M.calc_hit(ctx)：战斗链每击×每目标调用一次，按 damage_calculate 的 r 行
--     求单击伤害（P1 基值+P2 波动+威力×系数+分摊，客户端 tips_func_skill.lua
--     同构）：
--     ctx.formula_id / ctx.params[1..n]（r 行第 3 段起原始串）/
--     ctx.hurt_count / ctx.cur_target_count / ctx.props{属性名→值} /
--     ctx.vitals{HP/MaxHP/MP/MaxMP} /
--     ctx.rand{[1]=miss, [2]=crit, [3..]=区间掷序列——Go 预掷，脚本不掷骰}
--     返回 {damage, miss, crit} 或 {fallback=true}（调用方走 legacy 面板×倍率）。
--
--   参数来源码：5=字面量；6="a;b" 区间→掷一点；0=攻方属性；1=计数器
--   (HurtCount/CurTargetCount)；2=生存值。4/7/8/9/10/11/12/13 宿主装载期过滤。
--   结算复用下方管线助手（守方侧当前全零，与参考服一致；接线后两路径同享）。

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

-- ── 每击公式路径 ─────────────────────────────────────────────────────────

local FORMULAS = nil -- [formula_id] = function(PV, R, Q)
local BINDINGS = nil -- [formula_id] = { [n] = {src=, prop=} }

function M.install_formulas(formulas, bindings)
  FORMULAS = formulas
  BINDINGS = bindings
  local count = 0
  for _ in pairs(formulas) do count = count + 1 end
  return count
end

-- next_rand 按求值顺序消费宿主预掷序列（rand[3..]），耗尽复用末位：
-- Go 侧 stub 序列即完全复现（确定性可测）。
local function next_rand(ctx)
  local seq = ctx.rand
  if not seq then return 0.5 end
  local index = (ctx._rand_index or 2) + 1
  ctx._rand_index = index
  local value = seq[index]
  if value == nil then
    value = seq[#seq] or 0.5
  end
  return value
end

-- resolve_param 解析单个参数位（P<n>）的取值。
local function resolve_param(ctx, binding, raw)
  if not binding then
    return tonumber(raw) or 0
  end
  local src = binding.src or 0
  if src == 5 then
    return tonumber(raw) or 0
  elseif src == 6 then
    -- "a;b" 波动区间 → [a,b] 均匀整数
    local text = tostring(raw or "")
    local lo, hi = text:match("^([^;]+);([^;]+)$")
    lo, hi = tonumber(lo), tonumber(hi)
    if not lo or not hi then
      return tonumber(raw) or 0
    end
    if hi < lo then lo, hi = hi, lo end
    return math.floor(lo + (hi - lo + 1) * next_rand(ctx))
  elseif src == 0 then
    local props = ctx.props or {}
    return props[binding.prop or ""] or 0
  elseif src == 1 then
    if binding.prop == "HurtCount" then return ctx.hurt_count or 1 end
    if binding.prop == "CurTargetCount" then return ctx.cur_target_count or 1 end
    return 0
  elseif src == 2 then
    local vitals = ctx.vitals or {}
    return vitals[binding.prop or ""] or 0
  end
  return 0
end

-- query_prop 是 #Prop 直接引用的查询（属性 → 生存值 → 计数器 → 0）。
local function query_prop(ctx, name)
  local value = (ctx.props or {})[name]
  if value == nil then value = (ctx.vitals or {})[name] end
  if value == nil and name == "HurtCount" then value = ctx.hurt_count end
  if value == nil and name == "CurTargetCount" then value = ctx.cur_target_count end
  return value or 0
end

function M.calc_hit(ctx)
  if not FORMULAS or not BINDINGS then
    return {fallback = true}
  end
  local fn = FORMULAS[ctx.formula_id]
  local bind = BINDINGS[ctx.formula_id]
  if not fn or not bind then
    return {fallback = true}
  end

  -- 参数解析：P1..Pmax（绑定表按 1..max 连续注入，位缺即 nil→字面量兜底）
  local params = ctx.params or {}
  local pv = {}
  local max_param = 0
  for n in pairs(bind) do
    if n > max_param then max_param = n end
  end
  for n = 1, max_param do
    pv[n] = resolve_param(ctx, bind[n], params[n])
  end

  local raw = fn(pv, function(lo, hi)
    if hi < lo then lo, hi = hi, lo end
    return math.floor(lo + (hi - lo + 1) * next_rand(ctx))
  end, function(name) return query_prop(ctx, name) end)
  if type(raw) ~= "number" or raw ~= raw or raw == math.huge or raw == -math.huge then
    return {fallback = true}
  end

  -- 结算次序同 M.calc：防御减免 → 未命中 → 暴击 → 保底。攻方暴击值/
  -- 命中值按伤害系取 PhyVa/MagicVa、PhyHit/MagicHit（Go 侧预选）；
  -- 守方（defender）缺省零值——NPC 属性块迁移前与参考服"守方侧缺省"
  -- 口径一致，管线已铺好（§十六 4–7 条）。
  local defender = ctx.defender or {}
  raw = raw * (1 - defenseReduction(defender.defense or 0))
  local seq = ctx.rand or {}
  local miss = missChance(ctx.accuracy or 0, defender.evasion or 0, 0, 0)
  if miss > 0 and (seq[1] or 0) < miss then
    return {damage = 0, miss = true, crit = false}
  end
  local chance = critChance(ctx.crit_value or 0, defender.resilience or 0, 0, 0)
  local crit = false
  if chance > 0 and (seq[2] or 0) < chance then
    raw = raw * critMultiplier(0, 0) -- 基础 2.0 倍率（外部爆伤未接线）
    crit = true
  end
  local damage = math.floor(raw)
  if damage < 1 then damage = 1 end
  return {damage = damage, miss = false, crit = crit}
end

return M
