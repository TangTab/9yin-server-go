-- 秋老诀（buf_ng_kl_001）｜昆仑派｜受击触发
-- 源：内功特效清单.txt 行358-361（4 层）。
-- 受击几率精气护盾（吸收一定量）。层4：护盾期间攻击者中"衰老"（招式伤害 -50%
--   衰减，TODO：debuff；v1 未实现）。
local M = {}
M.meta = { id = "buf_ng_kl_001", name = "秋老诀", book = "ng_kl_001",
           category = "victim", max_tier = 4 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { absorb = 130 },
  [2] = { absorb = 165 },
  [3] = { absorb = 205 },
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
  return { shield = { amount = lv(ctx).absorb, life_ms = 10000 } }
end

return M
