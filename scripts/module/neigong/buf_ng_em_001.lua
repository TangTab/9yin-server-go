-- 气桩功（buf_ng_em_001）｜峨眉｜招架触发
-- 源：内功特效清单.txt 行47-50（4 层）。
-- 层1-3：招架时额外平减 15/30/45 内功 + 12/16/22 外功伤害（当前管线全外功，取外功值）。
-- 层4：90% 内功 / 70% 外功减免近似为招架时比例减免 + 内力上限 +100（罡气 +5 TODO）。
local M = {}
M.meta = { id = "buf_ng_em_001", name = "气桩功", book = "ng_em_001",
           category = "parry", max_tier = 4 }
M.levels = {
  [1] = { external = 12, internal = 15, reduce = 0 },
  [2] = { external = 16, internal = 30, reduce = 0 },
  [3] = { external = 22, internal = 45, reduce = 0 },
  [4] = { external = 0,  internal = 0,  reduce = 0.70 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_parry(ctx)
  local p = lv(ctx)
  if p.reduce > 0 then
    -- 层4 比例减免在 on_victim（parried 场景）生效更贴近原意
    return nil
  end
  return { parry_reduce = { internal = p.internal, external = p.external } }
end

function M.on_victim(ctx)
  local p = lv(ctx)
  if p.reduce <= 0 or not ctx.parried then return nil end
  return { reduce_pct = p.reduce }
end

function M.passive(ctx)
  local p = lv(ctx)
  if p.reduce > 0 then
    return { max_mp_add = 100 }
  end
  return nil
end

return M
