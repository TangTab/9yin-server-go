-- 太极神功（buf_ng_wd_006）｜武当｜受击触发
-- 源：内功特效清单.txt 行601-605（5 层）。
-- 化气：受击几率化劲吸收 15-30% 伤害并回血；引劲：受击几率卸去 15-30% 并反弹柔伤。
-- 层5 太极为道：触发时招式伤害+10% 可叠 5 层（TODO：临时伤害%指令）。
local M = {}
M.meta = { id = "buf_ng_wd_006", name = "太极神功", book = "ng_wd_006",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { pct = 0.15, heal = 30, rou = 25 },
  [2] = { pct = 0.18, heal = 36, rou = 30 },
  [3] = { pct = 0.21, heal = 42, rou = 35 },
  [4] = { pct = 0.25, heal = 48, rou = 42 },
  [5] = { pct = 0.30, heal = 55, rou = 50 },
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
  local r = ctx.rand.r_proc * 1000
  if math.floor(r) % 2 == 0 then
    -- 化气：吸收 + 回血
    return { reduce_pct = p.pct, heal = p.heal }
  end
  -- 引劲：卸去 + 柔伤反弹
  return { reduce_pct = p.pct,
           reflect = { amount = p.rou + math.floor(ctx.incoming * 0.10), kind = "rou" } }
end

return M
