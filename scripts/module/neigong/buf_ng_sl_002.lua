-- 二指禅（buf_ng_sl_002）｜少林派｜攻击触发
-- 源：内功特效清单.txt 行458-461（4 层）。
-- 每次出招起手劲气震伤周围目标（TODO：AoE；层4 臂力加成 TODO；v1：单体阳伤近似）。
local M = {}
M.meta = { id = "buf_ng_sl_002", name = "二指禅", book = "ng_sl_002",
           category = "attack", max_tier = 4 }
M.levels = {
  [1] = { gang = 30 },
  [2] = { gang = 38 },
  [3] = { gang = 47 },
  [4] = { gang = 58 },
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
  return { extra_damage = { amount = lv(ctx).gang, kind = "gang" } }
end

return M
