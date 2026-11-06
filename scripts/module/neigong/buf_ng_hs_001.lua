-- 固元经（buf_ng_hs_001）｜华山派｜受击触发
-- 源：内功特效清单.txt 行123-127（5 层）。
-- 受击几率收拢身形：N 秒免伤 8-20%（窗口内每击减免）。
-- 层4+ 抱元守一：触发时锁定对方，攻击其额外 17-20% 柔伤（TODO：目标标记；v1 未实现）。
local M = {}
M.meta = { id = "buf_ng_hs_001", name = "固元经", book = "ng_hs_001",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { reduce = 0.08, dur_ms = 4000 },
  [2] = { reduce = 0.11, dur_ms = 5000 },
  [3] = { reduce = 0.14, dur_ms = 6000 },
  [4] = { reduce = 0.17, dur_ms = 8000 },
  [5] = { reduce = 0.20, dur_ms = 10000 },
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
    return { reduce_pct = p.reduce, state = { until_at = now + p.dur_ms } }
  end
  return nil
end

return M
