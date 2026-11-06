-- 玄元经·奇技（buf_ng_jyqj_001）｜君子堂｜常驻/被动
-- 源：内功特效清单.txt 行325-327（3 层）。
-- 使目标持续伤害 + 降低 50-100% 气血恢复 10 秒；层3 无法回血回蓝（TODO：DoT/
--   治疗封锁 debuff；v1：攻击几率立即阴伤一跳近似）。
local M = {}
M.meta = { id = "buf_ng_jyqj_001", name = "玄元经", book = "ng_jyqj_001",
           category = "passive", max_tier = 3 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { yin = 55 },
  [2] = { yin = 70 },
  [3] = { yin = 88 },
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
