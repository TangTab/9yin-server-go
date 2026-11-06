-- 双修诀（buf_ng_jl_001）｜极乐谷｜常驻/被动
-- 源：内功特效清单.txt 行268-271（4 层）。
-- 每攻击 6/5/4/4 次，下次攻击附加额外 2/2/2/3 倍武器伤害（以当次伤害近似武器伤害）。
local M = {}
M.meta = { id = "buf_ng_jl_001", name = "双修诀", book = "ng_jl_001",
           category = "passive", max_tier = 4 }
M.levels = {
  [1] = { need = 6, mult = 2 },
  [2] = { need = 5, mult = 2 },
  [3] = { need = 4, mult = 2 },
  [4] = { need = 4, mult = 3 },
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
    return { extra_damage = { amount = math.floor(ctx.damage * p.mult), kind = "external" },
             state = { count = 0 } }
  end
  return { state = { count = count } }
end

return M
