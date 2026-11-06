-- 武当会意功（buf_ng_wd_101）｜武当（会意）｜受击触发
-- 源：内功特效清单.txt 行606-607（档 1/5，中间档按最高档近似）。
-- 化气/引劲 30% + 太极为道（触发时招式伤害 +10% 叠 5 层，TODO：临时伤害%指令）。
local M = {}
M.meta = { id = "buf_ng_wd_101", name = "武当会意功", book = "ng_wd_101",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.30 }
M.levels = {
  [1] = { pct = 0.30, heal = 60, rou = 50 },
  [5] = { pct = 0.30, heal = 60, rou = 50 },
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

function M.on_victim(ctx)
  if not roll(ctx, M.defaults.proc) then return nil end
  local p = lv(ctx)
  if math.floor(ctx.rand.r_proc * 1000) % 2 == 0 then
    return { reduce_pct = p.pct, heal = p.heal }
  end
  return { reduce_pct = p.pct,
           reflect = { amount = p.rou + math.floor(ctx.incoming * 0.10), kind = "rou" } }
end

return M
