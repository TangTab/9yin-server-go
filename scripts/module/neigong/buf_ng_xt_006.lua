-- 天将神书（buf_ng_xt_006）｜行字辈｜受击触发
-- 源：内功特效清单.txt 行706-710（5 层）。
-- 铜墙铁壁：受击 15% 几率下一个招式伤害减 50-100%，随后"坚如磐石"附加伤 -20-40%
-- 10 秒（TODO：持久减伤；v1：下次受击一次性减免近似）。破敌一击/纵横天下 TODO。
local M = {}
M.meta = { id = "buf_ng_xt_006", name = "天将神书", book = "ng_xt_006",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.15 }
M.levels = {
  [1] = { reduce = 0.50 },
  [2] = { reduce = 0.60 },
  [3] = { reduce = 0.70 },
  [4] = { reduce = 0.80 },
  [5] = { reduce = 1.00 },
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
  local p = lv(ctx)
  local s = ctx.state or {}
  if s.charged then
    return { reduce_pct = p.reduce, state = {} }
  end
  if roll(ctx, M.defaults.proc) then
    return { state = { charged = true } }
  end
  return nil
end

return M
