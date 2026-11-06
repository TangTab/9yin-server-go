-- 归一气功（buf_ng_xt_004）｜行字辈｜常驻/被动
-- 源：内功特效清单.txt 行696-700（5 层）。
-- 先天元气：受击 20% 几率卸去接下来 5-10 次攻击部分劲道 15-30 秒（v1：触发后
--   窗口内每击固定减免 15%）。元气溃散（范围失明+减速）TODO；凝韵听息（内力减免）TODO。
local M = {}
M.meta = { id = "buf_ng_xt_004", name = "归一气功", book = "ng_xt_004",
           category = "passive", max_tier = 5 }
M.defaults = { proc = 0.20, per = 0.15 }
M.levels = {
  [1] = { hits = 5,  dur_ms = 15000 },
  [2] = { hits = 6,  dur_ms = 18000 },
  [3] = { hits = 7,  dur_ms = 21000 },
  [4] = { hits = 8,  dur_ms = 24000 },
  [5] = { hits = 10, dur_ms = 30000 },
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
  local now = ctx.now or 0
  if s.hits and s.hits > 0 and s.until_at and now < s.until_at then
    return { reduce_pct = M.defaults.per, state = { hits = s.hits - 1, until_at = s.until_at } }
  end
  if roll(ctx, M.defaults.proc) then
    return { reduce_pct = M.defaults.per,
             state = { hits = p.hits - 1, until_at = now + p.dur_ms } }
  end
  return nil
end

return M
