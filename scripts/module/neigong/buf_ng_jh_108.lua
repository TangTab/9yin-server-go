-- 混元功（buf_ng_jh_108）｜江湖｜受击触发
-- 源：内功特效清单.txt 行144-151（8 层）。
-- 处于禁轻功/定身/昏迷状态被攻击时 3 秒持续恢复气血（v1：受控受击立即回复一次近似；
-- 持续 HoT TODO）。层5+：几率解除负面 TODO；层8：低血双倍回复 TODO（短 CD）。
local M = {}
M.meta = { id = "buf_ng_jh_108", name = "混元功", book = "ng_jh_108",
           category = "victim", max_tier = 8 }
M.levels = {
  [1] = { heal = 40 },
  [2] = { heal = 52 },
  [3] = { heal = 66 },
  [4] = { heal = 82 },
  [5] = { heal = 100 },
  [6] = { heal = 120 },
  [7] = { heal = 142 },
  [8] = { heal = 168 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  if not ctx.controlled then return nil end
  return { heal = lv(ctx).heal }
end

return M
