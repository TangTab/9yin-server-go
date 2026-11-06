-- 玄元控水诀（buf_ng_ssg_001）｜水上漂｜受击触发
-- 源：内功特效清单.txt 行484-488（5 层）。
-- 受击几率凝气成水向攻击者打入水珠造成阴伤（层4+ 多目标穿梭递减 TODO：AoE）。
local M = {}
M.meta = { id = "buf_ng_ssg_001", name = "玄元控水诀", book = "ng_ssg_001",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { yin = 60 },
  [2] = { yin = 76 },
  [3] = { yin = 94 },
  [4] = { yin = 114 },
  [5] = { yin = 136 },
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
  return { reflect = { amount = lv(ctx).yin, kind = "yin" } }
end

return M
