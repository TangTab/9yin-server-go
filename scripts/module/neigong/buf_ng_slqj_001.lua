-- 禅定功·奇技（buf_ng_slqj_001）｜少林派｜常驻/被动
-- 源：内功特效清单.txt 行481-483（3 层）。
-- 层1-2：攻击 30% 几率恢复气血；层3：+10% 几率恢复双倍。
local M = {}
M.meta = { id = "buf_ng_slqj_001", name = "禅定功", book = "ng_slqj_001",
           category = "passive", max_tier = 3 }
M.defaults = { proc = 0.30, double_proc = 0.10 }
M.levels = {
  [1] = { heal = 55 },
  [2] = { heal = 70 },
  [3] = { heal = 86 },
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
  local heal = lv(ctx).heal
  if tier_of(ctx) >= 3 and roll(ctx, M.defaults.double_proc) then
    heal = heal * 2
  end
  return { heal = heal }
end

return M
