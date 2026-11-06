-- 幽昙秘典·第一重（buf_ng_jh_507）｜江湖｜叠加层数
-- 源：内功特效清单.txt 行230-235（6 层）。
-- 偷移速/穿云纵距离（TODO：移速偷取）；巧捷万端/幽昙真气（TODO：调息/公冷系统）；
-- 层5 至此不败（TODO：免硬直）；层6 未领悟。运功消耗修为 TODO。
-- v1：攻击几率附柔伤占位。
local M = {}
M.meta = { id = "buf_ng_jh_507", name = "幽昙秘典(第一重)", book = "ng_jh_507",
           category = "stack", max_tier = 6 }
M.defaults = { proc = 0.15 }
M.levels = {
  [1] = { rou = 36 },
  [2] = { rou = 46 },
  [3] = { rou = 58 },
  [4] = { rou = 72 },
  [5] = { rou = 88 },
  [6] = { rou = 0 },
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
  local p = lv(ctx)
  if p.rou <= 0 then return nil end   -- 层6 未领悟
  if not roll(ctx, M.defaults.proc) then return nil end
  return { extra_damage = { amount = p.rou, kind = "rou" } }
end

return M
