-- 峨眉会意功（buf_ng_em_101）｜峨眉（会意）｜受击触发
-- 源：内功特效清单.txt 行74-75（档 1/5，中间档按最高档近似）。
-- 花落无期：被攻击叠 6 层后抵挡 30% 伤害 3 次。花开有时/青莲聚魂：TODO（临时伤害%/
--   致死抵挡/金顶佛光/击退定身均缺指令）。
local M = {}
M.meta = { id = "buf_ng_em_101", name = "峨眉会意功", book = "ng_em_101",
           category = "victim", max_tier = 5 }
M.levels = {
  [1] = { reduce = 0.30 },
  [5] = { reduce = 0.30 },
}
M.STACK_NEED = 6
M.GUARD_HITS = 3

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  if s.guard and s.guard > 0 then
    return { reduce_pct = p.reduce, state = { guard = s.guard - 1 } }
  end
  local stacks = (s.stacks or 0) + 1
  if stacks >= M.STACK_NEED then
    return { state = { guard = M.GUARD_HITS } }
  end
  return { state = { stacks = stacks } }
end

return M
