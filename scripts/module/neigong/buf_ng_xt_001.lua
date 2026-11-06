-- 入战诀（buf_ng_xt_001）｜行字辈｜叠加层数
-- 源：内功特效清单.txt 行684-687（4 层）。
-- 战意：入战每 3/2/1.5/1 秒 +1 层（伤害 +0.7-1%/层，最多 10-16 层）；
-- 攻/受消层；脱战每秒 -1 层（层4 不脱层）。
-- TODO：入战/脱战事件与临时伤害%；v1：受击叠层减伤近似（每层 1% 减伤）。
local M = {}
M.meta = { id = "buf_ng_xt_001", name = "入战诀", book = "ng_xt_001",
           category = "stack", max_tier = 4 }
M.levels = {
  [1] = { max = 10 },
  [2] = { max = 12 },
  [3] = { max = 14 },
  [4] = { max = 16 },
}
M.PER_LAYER = 0.01
M.DUR_MS = 15000

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then
    return { reduce_pct = M.PER_LAYER * (s.layers or 0),
             state = { layers = math.min((s.layers or 0) + 1, p.max), until_at = s.until_at } }
  end
  return { reduce_pct = M.PER_LAYER, state = { layers = 1, until_at = now + M.DUR_MS } }
end

return M
