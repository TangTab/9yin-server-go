-- 洗髓经（buf_ng_sl_005）｜少林派｜受击触发
-- 源：内功特效清单.txt 行471-475（5 层）。
-- 无使钟气：攻击几率暴伤 +4% 叠 2-6 层（TODO：临时暴伤）。洗髓还原：受击几率回血
-- （层5：回血并反震 + 易髓净化 90 秒 CD TODO）。v1：受击回血 + 层5 反震。
local M = {}
M.meta = { id = "buf_ng_sl_005", name = "洗髓经", book = "ng_sl_005",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { heal = 35,  reflect = 0 },
  [2] = { heal = 44,  reflect = 0 },
  [3] = { heal = 54,  reflect = 0 },
  [4] = { heal = 66,  reflect = 0 },
  [5] = { heal = 80,  reflect = 60 },
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
  if p.reflect > 0 then
    return { heal = p.heal, reflect = { amount = p.reflect, kind = "external" } }
  end
  return { heal = p.heal }
end

return M
