-- 通慧功（buf_ng_jz_001）｜禁地｜攻击触发
-- 源：内功特效清单.txt 行328-331（4 层）。
-- 攻击 40% 几率恢复内力，内息越高越多（v1：固定基数 + 档位成长；持续恢复 TODO）。
local M = {}
M.meta = { id = "buf_ng_jz_001", name = "通慧功", book = "ng_jz_001",
           category = "attack", max_tier = 4 }
M.defaults = { proc = 0.40 }
M.levels = {
  [1] = { mp = 18 },
  [2] = { mp = 24 },
  [3] = { mp = 30 },
  [4] = { mp = 38 },
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
  return { mp_restore = lv(ctx).mp }
end

return M
