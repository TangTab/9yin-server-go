-- 幻玉神功（buf_ng_yh_001）｜移花宫｜受击触发
-- 源：内功特效清单.txt 行711-717（7 层）。
-- 料敌先机：受击 50% 几率闪避率 +20% 持续 5-11 秒（TODO：临时闪避%指令；v1 以常驻
--   闪避值近似）；移花接玉：成功闪避恢复 2-5% 最大气血（TODO：闪避事件钩子）。
-- 层6-7：下一击伤害 +60%（TODO：临时伤害%指令）。
local M = {}
M.meta = { id = "buf_ng_yh_001", name = "幻玉神功", book = "ng_yh_001",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.50 }
M.levels = {
  [1] = { evasion = 6 },
  [2] = { evasion = 8 },
  [3] = { evasion = 10 },
  [4] = { evasion = 12 },
  [5] = { evasion = 14 },
  [6] = { evasion = 16 },
  [7] = { evasion = 20 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.passive(ctx)
  return { evasion_add = lv(ctx).evasion }
end

return M
