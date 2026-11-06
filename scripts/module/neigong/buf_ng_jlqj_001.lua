-- 双修诀·奇技（buf_ng_jlqj_001）｜极乐谷｜常驻/被动
-- 源：内功特效清单.txt 行295-297（3 层）。
-- 每攻击 5/4/3 次，下 2/3/4 次攻击附加额外 3/3/4 倍武器伤害（以当次伤害近似武器伤害）。
local M = {}
M.meta = { id = "buf_ng_jlqj_001", name = "双修诀", book = "ng_jlqj_001",
           category = "passive", max_tier = 3 }
M.levels = {
  [1] = { need = 5, charges = 2, mult = 3 },
  [2] = { need = 4, charges = 3, mult = 3 },
  [3] = { need = 3, charges = 4, mult = 4 },
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
  if s.charges and s.charges > 0 then
    return { extra_damage = { amount = math.floor(ctx.damage * p.mult), kind = "external" },
             state = { charges = s.charges - 1 } }
  end
  local count = (s.count or 0) + 1
  if count >= p.need then
    return { state = { count = 0, charges = p.charges } }
  end
  return { state = { count = count } }
end

return M
