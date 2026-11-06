-- 六合经（buf_ng_tm_002）｜唐门｜常驻/被动
-- 源：内功特效清单.txt 行514-517（4 层）。
-- 暴击伤害提高 20-60%。层4：暴击撕裂（身法加成额外伤 TODO：临时暴伤 debuff）。
local M = {}
M.meta = { id = "buf_ng_tm_002", name = "六合经", book = "ng_tm_002",
           category = "passive", max_tier = 4 }
M.levels = {
  [1] = { crit_dmg = 0.20 },
  [2] = { crit_dmg = 0.35 },
  [3] = { crit_dmg = 0.50 },
  [4] = { crit_dmg = 0.60 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.passive(ctx)
  return { crit_damage_bonus = lv(ctx).crit_dmg }
end

return M
