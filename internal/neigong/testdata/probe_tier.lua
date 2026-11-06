-- 探针：档位门——tier_of 照抄真实脚本，回显生效档位的数值。
local M = {}
M.meta = { id = "probe_tier", name = "探针·档位", book = "ng_probe_tier",
           category = "hit", max_tier = 5 }
M.levels = { [1] = { v = 10 }, [2] = { v = 20 }, [3] = { v = 30 },
             [4] = { v = 40 }, [5] = { v = 50 } }
local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
function M.on_attack(ctx)
  return { extra_damage = { amount = M.levels[tier_of(ctx)].v } }
end
return M
