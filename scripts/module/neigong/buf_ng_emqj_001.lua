-- 气桩功·奇技（buf_ng_emqj_001）｜峨眉｜招架触发
-- 源：内功特效清单.txt 行76-78（3 层）。
-- 招架时额外减免 58-90% 内功 / 55-70% 外功伤害（当前管线全外功，取外功比例）。
-- 层3：10% 几率入"气"境：本门招式伤害 +10% 10 秒（TODO：临时伤害%）；内力 +100/罡气 +5 TODO。
local M = {}
M.meta = { id = "buf_ng_emqj_001", name = "气桩功", book = "ng_emqj_001",
           category = "parry", max_tier = 3 }
M.levels = {
  [1] = { reduce = 0.55 },
  [2] = { reduce = 0.65 },
  [3] = { reduce = 0.70 },
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
  return nil   -- 比例减免在 on_victim（parried）生效
end

function M.on_victim(ctx)
  if not ctx.parried then return nil end
  return { reduce_pct = lv(ctx).reduce }
end

function M.passive(ctx)
  if tier_of(ctx) >= 3 then
    return { max_mp_add = 100 }
  end
  return nil
end

return M
