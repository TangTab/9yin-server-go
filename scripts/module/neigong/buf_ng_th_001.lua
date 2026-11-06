-- 碧波心经（buf_ng_th_001）｜唐门｜受击触发
-- 源：内功特效清单.txt 行496-502（7 层）。
-- 潮起：攻击 50% 几率造成额外 30-65% + 241-736 点柔伤，25 秒冷却（层6-7 另减速禁轻功 TODO）。
-- 潮退：被攻击 50% 几率令攻击者定身 3-4 秒（TODO：控制指令）。
local M = {}
M.meta = { id = "buf_ng_th_001", name = "碧波心经", book = "ng_th_001",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.50, cd_ms = 25000 }
M.levels = {
  [1] = { pct = 0.30, flat = 241 },
  [2] = { pct = 0.35, flat = 295 },
  [3] = { pct = 0.40, flat = 355 },
  [4] = { pct = 0.45, flat = 412 },
  [5] = { pct = 0.50, flat = 475 },
  [6] = { pct = 0.60, flat = 603 },
  [7] = { pct = 0.65, flat = 736 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end
local function roll(ctx, p) return (ctx.rand and ctx.rand.r_proc or 1) < p end

function M.on_attack(ctx)
  if not roll(ctx, M.defaults.proc) then return nil end
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.cd_until and now < s.cd_until then return nil end
  local p = lv(ctx)
  return {
    extra_damage = { amount = math.floor(ctx.damage * p.pct) + p.flat, kind = "rou" },
    state = { cd_until = now + M.defaults.cd_ms },
  }
end

return M
