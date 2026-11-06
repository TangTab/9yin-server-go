-- 寒魄诀（buf_ng_ts_001）｜天山派｜常驻/被动
-- 源：内功特效清单.txt 行540-543（4 层）。
-- 每攻击 8/7/6/5 次，下次攻击额外 97-234 阴伤（近身/内功威力加成 TODO）。
local M = {}
M.meta = { id = "buf_ng_ts_001", name = "寒魄诀", book = "ng_ts_001",
           category = "passive", max_tier = 4 }
M.levels = {
  [1] = { need = 8, yin = 97 },
  [2] = { need = 7, yin = 138 },
  [3] = { need = 6, yin = 184 },
  [4] = { need = 5, yin = 234 },
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
  local p = lv(ctx)
  local s = ctx.state or {}
  local count = (s.count or 0) + 1
  if count >= p.need then
    return { extra_damage = { amount = p.yin, kind = "yin" }, state = { count = 0 } }
  end
  return { state = { count = count } }
end

return M
