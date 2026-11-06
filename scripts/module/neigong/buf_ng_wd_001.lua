-- 两仪护心功（buf_ng_wd_001）｜武当｜受击触发
-- 源：内功特效清单.txt 行579-582（4 层）。
-- 层1-3：被攻击 10% 几率激发真气，吸收 100/150/210 点伤害。
-- 层4：吸收 250 点；真气散去部分回内力（TODO：散去回内力，需护盾打破事件）。
local M = {}
M.meta = { id = "buf_ng_wd_001", name = "两仪护心功", book = "ng_wd_001",
           category = "victim", max_tier = 4 }
M.defaults = { proc = 0.10 }
M.levels = {
  [1] = { absorb = 100 },
  [2] = { absorb = 150 },
  [3] = { absorb = 210 },
  [4] = { absorb = 250 },
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

function M.on_victim(ctx)
  if not roll(ctx, M.defaults.proc) then return nil end
  local p = lv(ctx)
  return { shield = { amount = p.absorb, life_ms = 12000 } }
end

return M
