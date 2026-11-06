-- 心莲无量功（buf_ng_em_006）｜峨眉｜受击触发
-- 源：内功特效清单.txt 行69-73（5 层）。
-- 花开有时：攻击/被攻击叠"花败"6 层后获"花开"：被击抵挡 15-30%、攻击附 15-30% 柔伤，
-- 攻防合计 3 次后移除（state 状态机）。
local M = {}
M.meta = { id = "buf_ng_em_006", name = "心莲无量功", book = "ng_em_006",
           category = "victim", max_tier = 5 }
M.levels = {
  [1] = { pct = 0.15 },
  [2] = { pct = 0.18 },
  [3] = { pct = 0.21 },
  [4] = { pct = 0.25 },
  [5] = { pct = 0.30 },
}
M.BLOOM_NEED = 6
M.BLOOM_HITS = 3

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
  if s.bloom and s.bloom > 0 then
    return { extra_damage = { amount = math.floor(ctx.damage * p.pct), kind = "rou" },
             state = { bloom = s.bloom - 1 } }
  end
  local stacks = (s.stacks or 0) + 1
  if stacks >= M.BLOOM_NEED then
    return { state = { bloom = M.BLOOM_HITS } }
  end
  return { state = { stacks = stacks } }
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  if s.bloom and s.bloom > 0 then
    return { reduce_pct = p.pct, state = { bloom = s.bloom - 1 } }
  end
  local stacks = (s.stacks or 0) + 1
  if stacks >= M.BLOOM_NEED then
    return { state = { bloom = M.BLOOM_HITS } }
  end
  return { state = { stacks = stacks } }
end

return M
