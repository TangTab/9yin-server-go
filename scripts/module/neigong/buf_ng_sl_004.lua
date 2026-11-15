-- 旃檀心经（buf_ng_sl_004）｜少林派｜受击触发
-- 源：内功特效清单.txt 行466-470（5 层）。
-- 受击几率减免该次伤害 15-25% 并获旃檀香气；化瘀（死血转淤血 TODO）；
-- 活血：5(4) 缕后将 30-50% 淤血恢复为气血（bruise_restore percent 官方值）。
-- 层5：受击几率移除昏迷封招 TODO。
local M = {}
M.meta = { id = "buf_ng_sl_004", name = "旃檀心经", book = "ng_sl_004",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { reduce = 0.15, need = 5, bruise = 0.30 },
  [2] = { reduce = 0.17, need = 5, bruise = 0.34 },
  [3] = { reduce = 0.20, need = 5, bruise = 0.38 },
  [4] = { reduce = 0.22, need = 5, bruise = 0.42 },
  [5] = { reduce = 0.25, need = 4, bruise = 0.50 },
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

function M.on_victim(ctx)
  if not roll(ctx, M.defaults.proc) then return nil end
  local p = lv(ctx)
  local s = ctx.state or {}
  local aroma = (s.aroma or 0) + 1
  if aroma >= p.need then
    -- 活血：叠层满后把官方比例的淤血层转为气血（受层厚上限）。
    return { reduce_pct = p.reduce, bruise_restore = { percent = p.bruise }, state = { aroma = 0 } }
  end
  return { reduce_pct = p.reduce, state = { aroma = aroma } }
end

return M
