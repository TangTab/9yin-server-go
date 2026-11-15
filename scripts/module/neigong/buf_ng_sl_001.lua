-- 禅定功（buf_ng_sl_001）｜少林派｜常驻/被动
-- 源：内功特效清单.txt 行454-457（4 层）。
-- 层1-3：攻击 15% 几率化解部分淤血（bruise_restore：淤血层即时转气血，
--        比例推断 15%/层——官方未给固定值，percent 形态自适应层厚）。
-- 层4：15% 几率回复少量气血（heal）。
local M = {}
M.meta = { id = "buf_ng_sl_001", name = "禅定功", book = "ng_sl_001",
           category = "passive", max_tier = 4 }
M.defaults = { proc = 0.15 }
M.levels = {
  [1] = { bruise = 0.15 },
  [2] = { bruise = 0.15 },
  [3] = { bruise = 0.15 },
  [4] = { heal = 52 },
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
  local p = lv(ctx)
  if p.bruise then
    return { bruise_restore = { percent = p.bruise } }
  end
  return { heal = p.heal }
end

return M
