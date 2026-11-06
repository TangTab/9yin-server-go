-- 行风典（buf_ng_xt_003）｜行字辈｜受击触发
-- 源：内功特效清单.txt 行692-695（4 层）。
-- 风势：攻击 20% 几率内息 +60-120 10 秒，结束获"风御"层数（每层抵御/提高 10-25%，
--   受击/攻击消层）——依赖临时属性（TODO）。行疾如风（移速）TODO。
-- v1：风御以受击减免层近似（触发后窗口内每次受击减 10-25% 并消层）。
local M = {}
M.meta = { id = "buf_ng_xt_003", name = "行风典", book = "ng_xt_003",
           category = "victim", max_tier = 4 }
M.defaults = { proc = 0.20, dur_ms = 10000 }
M.levels = {
  [1] = { per = 0.10, layers = 4 },
  [2] = { per = 0.15, layers = 5 },
  [3] = { per = 0.20, layers = 6 },
  [4] = { per = 0.25, layers = 7 },
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
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.layers and s.layers > 0 and s.until_at and now < s.until_at then
    return { state = { layers = s.layers - 1, until_at = s.until_at } }
  end
  if roll(ctx, M.defaults.proc) then
    return { state = { layers = p.layers, until_at = now + M.defaults.dur_ms } }
  end
  return nil
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.layers and s.layers > 0 and s.until_at and now < s.until_at then
    return { reduce_pct = p.per, state = { layers = s.layers - 1, until_at = s.until_at } }
  end
  return nil
end

return M
