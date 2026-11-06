-- 无妄神功（buf_ng_jh_wwsg）｜江湖｜攻击触发
-- 源：内功特效清单.txt 行244-251（8 层）。
-- 受到的所有伤害降低 8-25%（v1：受击常驻减免近似；"攻击时移除"互斥 TODO）；
-- 攻击时忽视目标 4-15% 内外功防御减免（TODO：临时忽视指令）+ 层5 起伤害 +5-10% TODO。
-- 层8：暴击伤害减免 25% TODO、暴击伤害 +10% TODO。
local M = {}
M.meta = { id = "buf_ng_jh_wwsg", name = "无妄神功", book = "ng_jh_wwsg",
           category = "attack", max_tier = 8 }
M.levels = {
  [1] = { reduce = 0.08 },
  [2] = { reduce = 0.10 },
  [3] = { reduce = 0.12 },
  [4] = { reduce = 0.16 },
  [5] = { reduce = 0.20 },
  [6] = { reduce = 0.24 },
  [7] = { reduce = 0.24 },
  [8] = { reduce = 0.25 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  return { reduce_pct = lv(ctx).reduce }
end

return M
