-- 合欢诀（buf_ng_jl_003）｜极乐谷｜常驻/被动
-- 源：内功特效清单.txt 行276-279（4 层）。
-- 惊心动魄：每攻击 6 次，下 3-4 次攻击回复武器伤害 30-50%（层4 有几率 100%）。
-- 合欢密：团练回轻功 TODO。
local M = {}
M.meta = { id = "buf_ng_jl_003", name = "合欢诀", book = "ng_jl_003",
           category = "passive", max_tier = 4 }
M.defaults = { double_proc = 0.20 }
M.levels = {
  [1] = { need = 6, charges = 3, pct = 0.30 },
  [2] = { need = 6, charges = 3, pct = 0.40 },
  [3] = { need = 6, charges = 4, pct = 0.40 },
  [4] = { need = 6, charges = 4, pct = 0.50 },
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
  local s = ctx.state or {}
  if s.charges and s.charges > 0 then
    local pct = p.pct
    if tier_of(ctx) >= 4 and roll(ctx, M.defaults.double_proc) then
      pct = 1.0
    end
    return { heal = math.floor(ctx.damage * pct), state = { charges = s.charges - 1 } }
  end
  local count = (s.count or 0) + 1
  if count >= p.need then
    return { state = { count = 0, charges = p.charges } }
  end
  return { state = { count = count } }
end

return M
