-- 齐天真罡（buf_ng_wd_005）｜武当｜受击触发
-- 源：内功特效清单.txt 行596-600（5 层）。
-- 号令天下：攻击几率 30% 招式伤害 10-15 秒（TODO：临时伤害%指令）。
-- 谁与争锋：受击几率 30% 免伤 + 持续回血 10-15 秒。v1：免伤窗口 + 单次回血近似。
-- 层5 武林至尊：可同时触发（TODO：号令天下未实现，互斥不生效）。
local M = {}
M.meta = { id = "buf_ng_wd_005", name = "齐天真罡", book = "ng_wd_005",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { reduce = 0.30, heal = 60,  dur_ms = 10000 },
  [2] = { reduce = 0.30, heal = 66,  dur_ms = 11000 },
  [3] = { reduce = 0.30, heal = 72,  dur_ms = 12000 },
  [4] = { reduce = 0.30, heal = 78,  dur_ms = 13000 },
  [5] = { reduce = 0.30, heal = 84,  dur_ms = 15000 },
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
  if s.until_at and now < s.until_at then
    return { reduce_pct = p.reduce }
  end
  if roll(ctx, M.defaults.proc) then
    return { reduce_pct = p.reduce, heal = p.heal,
             state = { until_at = now + p.dur_ms } }
  end
  return nil
end

return M
