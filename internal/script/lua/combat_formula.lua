-- 伤害公式引擎（内置；宿主 internal/script/formula.go 装载并注入公式闭包）。
--
-- 管线口径与 2433 线 scripts/module/combat/formula.lua（Java 参考服
-- CombatFormulas 的移植）一致，系数同源：17173《九阴真经伤害计算：暴伤与
-- 暴免》《暴击率与暴击值》、NGA 伤害框架帖、Age of Wushu Wiki
-- 「Accuracy & Evasion」。官服女号抓包（2026-11-25）对拍：新号面板
-- MeleePower=3 直接套公式取整即得实测值——普攻 ⌊(5+0.3)/2⌋=2、锁定
-- ⌊25.9/3⌋=8（暴击 ×2=16）、⌊16.99/2⌋=8、⌊33.17⌋=33。
--
-- 掷骰次序：公式求值 → 防御减免 → 未命中 → 取整 → 暴击（整伤翻倍）→ 招架 → 保底 1 点。
-- 随机数由宿主预掷（ctx.rand，脚本不掷骰），保证可测试。
--
-- M.install(formulas, bindings)：装载期注入
--   formulas[公式号] = function(PV, R, Q, MIN, MAX) return <转译表达式> end
--   bindings[公式号][n] = { src = 来源码, prop = 属性名 }
-- M.calc_hit(ctx)：每击×每目标一次，返回
--   { damage = 整数伤害, miss = bool, crit = bool, parried = bool }
--   或 { fallback = true }（公式缺席/不可求值——调用方走回退面）。

local M = {}

-- 系数常量区
local DEFENSE_SLOPE = 20.0        -- 减伤曲线斜率：每 e 倍外防 +20%
local DEFENSE_INTERCEPT = -44.0   -- 20·ln(外防/100) − 44
local DEFENSE_LOG_SCALE = 100.0
local MAX_REDUCTION = 0.75        -- 减伤上限
local CRIT_BASE = 0.05
local CRIT_SLOPE = 0.10
local MAX_BASE_CRIT = 0.15        -- 基础暴击率渐近上限
local CRIT_MULTIPLIER = 2.0       -- 基础暴伤 200%
local MISS_BASE = 0.10
local MISS_SLOPE = 0.15
local MAX_DODGE = 0.24            -- 闪避上限 24%
local PARRY_FACTOR = 0.10         -- 招架穿透比（女号抓包：9 → 1）

local function clamp(v, lo, hi)
  return math.max(lo, math.min(hi, v))
end

-- 外防 → 减伤比例 [0, 0.75]；外防 ≤ 0 或低于 ~900 时不减伤。
function M.defense_reduction(defense)
  if defense <= 0 then
    return 0
  end
  local pct = (DEFENSE_SLOPE * math.log(defense / DEFENSE_LOG_SCALE) + DEFENSE_INTERCEPT) / 100.0
  return clamp(pct, 0, MAX_REDUCTION)
end

-- 未命中率：10% + 15%×(闪避−命中)/(闪避+1)，夹 24% 上限后抵扣命中率%。
function M.miss_chance(accuracy, evasion, hit_bonus, dodge_bonus)
  local ratio = (evasion - accuracy) / (evasion + 1)
  local dodge = math.min(MAX_DODGE, MISS_BASE + MISS_SLOPE * ratio + dodge_bonus)
  return clamp(dodge - hit_bonus, 0, 1)
end

-- 暴击率：5% + 10%×(暴击值−坚韧)/(暴击值+1)，夹 15% 上限。
function M.crit_chance(crit_value, resilience, bonus, reduction)
  local ratio = (crit_value - resilience) / (crit_value + 1)
  local base = clamp(CRIT_BASE + CRIT_SLOPE * ratio, 0, MAX_BASE_CRIT)
  return clamp(base + bonus - reduction, 0, 1)
end

-- ── 公式装载 ─────────────────────────────────────────────────────────────

local FORMULAS = nil -- [公式号] = function(PV, R, Q, MIN, MAX)
local BINDINGS = nil -- [公式号] = { [n] = {src=, prop=} }

function M.install(formulas, bindings)
  FORMULAS, BINDINGS = formulas, bindings
  local count = 0
  for _ in pairs(formulas) do count = count + 1 end
  return count
end

-- next_rand 按求值顺序消费宿主预掷序列：rand[1] 未命中、rand[2] 暴击由
-- 结算段专用，区间掷值自 rand[3] 起；耗尽复用末位。
local function next_rand(ctx)
  local seq = ctx.rand
  if not seq or #seq == 0 then return 0.5 end
  local index = (ctx._rand_index or 2) + 1
  ctx._rand_index = index
  local value = seq[index]
  if value == nil then
    value = seq[#seq]
  end
  return value
end

-- resolve_param 解析参数位 P<n>。
local function resolve_param(ctx, binding, raw)
  if not binding then
    return tonumber(raw) or 0
  end
  local src = binding.src or 0
  local prop = binding.prop or ""
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
  elseif src == 0 or src == 3 or src == 11 or src == 15 then
    return (ctx.props or {})[prop] or 0
  elseif src == 2 or src == 14 then
    return (ctx.target or {})[prop] or 0
  elseif src == 12 then
    return (ctx.props or {}).MaxHP or 0
  elseif src == 13 then
    return (ctx.target or {}).MaxHP or 0
  elseif src == 1 then
    if prop == "HurtCount" then return ctx.hurt_count or 1 end
    if prop == "CurTargetCount" then return ctx.cur_target_count or 1 end
    if prop == "Level" then return ctx.skill_level or 1 end
    return 0
  elseif src == 4 then
    return tonumber(prop) or 0
  elseif src == 7 then
    return ctx.self_buff and ctx.self_buff(prop) or 0
  elseif src == 8 then
    return ctx.target_buff and ctx.target_buff(prop) or 0
  elseif src == 9 then
    return ctx.distance or 0
  elseif src == 10 then
    return ctx.named and ctx.named(prop) or 0
  end
  return 0
end

-- lookup 是 Q("名字") 直接引用：攻击者属性 → 目标属性 → 计数 → 0。
local function lookup(ctx, name)
  local value = (ctx.props or {})[name]
  if value == nil then value = (ctx.target or {})[name] end
  if value == nil and name == "HurtCount" then value = ctx.hurt_count end
  if value == nil and name == "CurTargetCount" then value = ctx.cur_target_count end
  return value or 0
end

-- M.settle_hit(ctx, raw)：只走结算管线（技能脚本 get_damage 给出原值时
-- 用——脚本算伤害，引擎管未命中/暴击/招架/保底）。
function M.settle_hit(ctx, raw)
  if type(raw) ~= "number" or raw ~= raw then
    return { fallback = true }
  end
  local defender = ctx.defender or {}
  raw = raw * (1 - M.defense_reduction(defender.defense or 0))
  local seq = ctx.rand or {}
  if seq[1] and seq[1] < M.miss_chance(ctx.accuracy or 0, defender.evasion or 0, 0, 0) then
    return { damage = 0, miss = true, crit = false, parried = false }
  end
  local damage = math.floor(raw)
  local crit = false
  if seq[2] and seq[2] < M.crit_chance(ctx.crit_value or 0, defender.resilience or 0, 0, 0) then
    damage = damage * CRIT_MULTIPLIER
    crit = true
  end
  local parried = false
  if ctx.parrying then
    damage = math.floor(damage * PARRY_FACTOR)
    parried = true
  end
  if damage < 1 then damage = 1 end
  return { damage = damage, miss = false, crit = crit, parried = parried }
end

function M.calc_hit(ctx)
  if not FORMULAS or not BINDINGS then
    return { fallback = true }
  end
  local fn = FORMULAS[ctx.formula_id]
  local bind = BINDINGS[ctx.formula_id]
  if not fn or not bind then
    return { fallback = true }
  end

  -- 参数解析：P1..Pmax（位缺即 nil → 按字面量兜底）
  local params = ctx.params or {}
  local pv = {}
  local max_param = 0
  for n in pairs(bind) do
    if n > max_param then max_param = n end
  end
  for n = 1, max_param do
    pv[n] = resolve_param(ctx, bind[n], params[n])
  end

  -- R(a,b)：区间均匀整数掷值（消费宿主预掷序列）
  local function roll(lo, hi)
    if hi < lo then lo, hi = hi, lo end
    return math.floor(lo + (hi - lo + 1) * next_rand(ctx))
  end
  local ok, raw = pcall(fn, pv, roll, function(name) return lookup(ctx, name) end, math.min, math.max)
  if not ok or type(raw) ~= "number" or raw ~= raw or raw == math.huge or raw == -math.huge then
    return { fallback = true }
  end

  -- 结算：防御减免 → 未命中 → 暴击 → 招架 → 保底。
  local defender = ctx.defender or {}
  raw = raw * (1 - M.defense_reduction(defender.defense or 0))
  local seq = ctx.rand or {}
  if seq[1] and seq[1] < M.miss_chance(ctx.accuracy or 0, defender.evasion or 0, 0, 0) then
    return { damage = 0, miss = true, crit = false, parried = false }
  end
  -- 先取整再暴击（抓包：锁定 8 → 暴击 16，是整伤翻倍而非原值翻倍再取整）。
  local damage = math.floor(raw)
  local crit = false
  if seq[2] and seq[2] < M.crit_chance(ctx.crit_value or 0, defender.resilience or 0, 0, 0) then
    damage = damage * CRIT_MULTIPLIER
    crit = true
  end
  local parried = false
  if ctx.parrying then
    damage = math.floor(damage * PARRY_FACTOR)
    parried = true
  end
  if damage < 1 then damage = 1 end
  return { damage = damage, miss = false, crit = crit, parried = parried }
end

__nx_combat = M
return M
