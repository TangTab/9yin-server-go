-- 燎原神功（buf_ng_mj_004）｜明教｜受击触发
-- 源：内功特效清单.txt 行397-401（5 层）。
-- 攻击种"火种"每秒 73-156 阳伤 8-12 秒，火势随目标出招/受击壮大；"燎原"爆发
-- 220-468 范围阳伤（TODO：DoT/火势层数/AoE/禁轻功；v1：攻击几率立即阳伤一跳近似）。
-- 层5 浴火：燎原后下个怒招 +20% TODO。
local M = {}
M.meta = { id = "buf_ng_mj_004", name = "燎原神功", book = "ng_mj_004",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { yang = 55 },
  [2] = { yang = 68 },
  [3] = { yang = 84 },
  [4] = { yang = 102 },
  [5] = { yang = 122 },
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
  return { extra_damage = { amount = lv(ctx).yang, kind = "yang" } }
end

return M
