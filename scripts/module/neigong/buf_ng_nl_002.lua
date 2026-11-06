-- 惑神宝篆（buf_ng_nl_002）｜念萝芭｜受击触发
-- 源：内功特效清单.txt 行419-425（7 层）。
-- 魔吟：攻击几率啸音使目标 N 秒持续阴伤（TODO：DoT；v1：立即阴伤一跳近似）。
-- 魅舞：受击几率使攻击者 N 秒伤害 -12-30%（TODO：debuff）。天魔幻境 TODO。
local M = {}
M.meta = { id = "buf_ng_nl_002", name = "惑神宝篆", book = "ng_nl_002",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { yin = 40 },
  [2] = { yin = 50 },
  [3] = { yin = 62 },
  [4] = { yin = 76 },
  [5] = { yin = 92 },
  [6] = { yin = 110 },
  [7] = { yin = 130 },
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
  if not roll(ctx, M.defaults.proc) then return nil end
  return { extra_damage = { amount = lv(ctx).yin, kind = "yin" } }
end

return M
