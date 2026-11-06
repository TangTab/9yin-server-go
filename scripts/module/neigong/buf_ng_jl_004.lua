-- 魔相诀（buf_ng_jl_004）｜极乐谷｜攻击触发
-- 源：内功特效清单.txt 行280-284（5 层）。
-- 万象皆魔：暴击时吸取武器伤害 75-150% 的气血（暴伤提高部分 TODO：临时暴伤）。
-- 魔相层数 TODO（暴伤 +18-30%、暴击消层）。
local M = {}
M.meta = { id = "buf_ng_jl_004", name = "魔相诀", book = "ng_jl_004",
           category = "attack", max_tier = 5 }
M.levels = {
  [1] = { drain = 0.75 },
  [2] = { drain = 0.90 },
  [3] = { drain = 1.00 },
  [4] = { drain = 1.20 },
  [5] = { drain = 1.50 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_attack(ctx)
  if not ctx.crit then return nil end
  return { heal = math.floor(ctx.damage * lv(ctx).drain) }
end

return M
