-- 摄神诀（buf_ng_jy_002）｜君子堂｜叠加层数
-- 源：内功特效清单.txt 行302-305（4 层）。
-- 每攻克一个单位 +5 体魄 +3% 近战威力，叠 5-10 层（TODO：击杀事件与临时属性；
-- v1：击杀判定（伤害≥目标血量）记录连杀数，按层提供被动攻击加成近似）。
local M = {}
M.meta = { id = "buf_ng_jy_002", name = "摄神诀", book = "ng_jy_002",
           category = "stack", max_tier = 4 }
M.levels = {
  [1] = { max = 5 },
  [2] = { max = 6 },
  [3] = { max = 8 },
  [4] = { max = 10 },
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
  local stacks = s.stacks or 0
  if (ctx.target.hp or 0) - ctx.damage <= 0 and stacks < p.max then
    return { state = { stacks = stacks + 1 } }
  end
  if stacks > 0 then
    return { extra_damage = { amount = stacks * 4, kind = "external" } }
  end
  return nil
end

return M
