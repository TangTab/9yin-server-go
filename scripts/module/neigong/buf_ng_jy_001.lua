-- 玄元经（buf_ng_jy_001）｜君子堂｜常驻/被动
-- 源：内功特效清单.txt 行298-301（4 层）。
-- 使目标受到持续伤害并降低 50-100% 回血速率 20 秒（TODO：DoT/治疗削弱 debuff；
-- v1：攻击几率立即阴伤一跳近似）。
local M = {}
M.meta = { id = "buf_ng_jy_001", name = "玄元经", book = "ng_jy_001",
           category = "passive", max_tier = 4 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { yin = 45 },
  [2] = { yin = 58 },
  [3] = { yin = 73 },
  [4] = { yin = 90 },
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
