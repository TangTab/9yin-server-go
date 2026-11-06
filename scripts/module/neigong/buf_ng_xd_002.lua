-- 破世神功（buf_ng_xd_002）｜邪刀门｜攻击触发
-- 源：内功特效清单.txt 行657-661（5 层）。
-- 独武傲世：攻击几率使目标流血 5 秒；攻击流血目标额外 15-30% 外功伤。
-- 侩子手：击杀玩家后下次必流血 TODO。v1：状态机两段实现。
local M = {}
M.meta = { id = "buf_ng_xd_002", name = "破世神功", book = "ng_xd_002",
           category = "attack", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { pct = 0.15 },
  [2] = { pct = 0.18 },
  [3] = { pct = 0.21 },
  [4] = { pct = 0.25 },
  [5] = { pct = 0.30 },
}
M.BLEED_MS = 5000

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
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.bleed_until and now < s.bleed_until then
    return { extra_damage = { amount = math.floor(ctx.damage * p.pct), kind = "external" },
             state = { bleed_until = s.bleed_until } }
  end
  if roll(ctx, M.defaults.proc) then
    return { state = { bleed_until = now + M.BLEED_MS } }
  end
  return nil
end

return M
