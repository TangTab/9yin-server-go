-- 幻玉功（buf_ng_jz_002）｜禁地｜常驻/被动
-- 源：内功特效清单.txt 行332-335（4 层）。
-- 提高 18-33% 基础移动速度（TODO：移速指令）；攻击附伤随速度提高（TODO：移速快照；
--   v1：被动命中近似 + 层4 推倒 TODO）。
local M = {}
M.meta = { id = "buf_ng_jz_002", name = "幻玉功", book = "ng_jz_002",
           category = "passive", max_tier = 4 }
M.levels = {
  [1] = { accuracy = 5 },
  [2] = { accuracy = 8 },
  [3] = { accuracy = 11 },
  [4] = { accuracy = 14 },
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
  return { accuracy_add = lv(ctx).accuracy }
end

return M
