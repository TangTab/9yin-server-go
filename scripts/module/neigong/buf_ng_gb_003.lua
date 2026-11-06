-- 降龙心法（buf_ng_gb_003）｜丐帮｜攻击触发
-- 源：内功特效清单.txt 行87-90（4 层）。
-- 龙吟：攻击几率造成目标当前气血 10% 的阳伤害（上限 458-983）。
-- 调息：打坐回轻功/解酒 TODO。
local M = {}
M.meta = { id = "buf_ng_gb_003", name = "降龙心法", book = "ng_gb_003",
           category = "attack", max_tier = 4 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { cap = 458 },
  [2] = { cap = 614 },
  [3] = { cap = 786 },
  [4] = { cap = 983 },
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
  local amount = math.floor((ctx.target.hp or 0) * 0.10)
  return { extra_damage = { amount = math.min(amount, p.cap), kind = "yang" } }
end

return M
