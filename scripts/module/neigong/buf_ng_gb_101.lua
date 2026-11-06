-- 丐帮会意功（buf_ng_gb_101）｜丐帮（会意）｜受击触发
-- 源：内功特效清单.txt 行106-107（档 1/5，中间档按最高档近似）。
-- 焚月归元：攻击几率额外阳伤（石破天惊调息缩减/天元破武 TODO：临时忽视/驱散指令）。
local M = {}
M.meta = { id = "buf_ng_gb_101", name = "丐帮会意功", book = "ng_gb_101",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { yang = 132 },
  [5] = { yang = 132 },
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
  return { extra_damage = { amount = lv(ctx).yang, kind = "yang" } }
end

return M
