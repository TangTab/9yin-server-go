-- 玄冥真罡·第一重（buf_ng_jh_306）｜江湖｜叠加层数
-- 源：内功特效清单.txt 行208-213（6 层）。
-- 攻击几率使目标修为溃散流失内力+减速（TODO：修为/内力流失/减速系统）；玄冥邪法：
-- 吸修为转气血/内力上限+免疫移控（TODO）；运功消耗修为/神融丸 TODO；层6 未领悟。
-- v1：仅保留攻击附阴伤占位，其余待修为系统。
local M = {}
M.meta = { id = "buf_ng_jh_306", name = "玄冥真罡(第一重)", book = "ng_jh_306",
           category = "stack", max_tier = 6 }
M.defaults = { proc = 0.15 }
M.levels = {
  [1] = { yin = 40 },
  [2] = { yin = 52 },
  [3] = { yin = 66 },
  [4] = { yin = 82 },
  [5] = { yin = 100 },
  [6] = { yin = 0 },
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
  local p = lv(ctx)
  if p.yin <= 0 then return nil end   -- 层6 未领悟
  if not roll(ctx, M.defaults.proc) then return nil end
  return { extra_damage = { amount = p.yin, kind = "yin" } }
end

return M
