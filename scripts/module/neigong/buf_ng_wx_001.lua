-- 蛊神诀（buf_ng_wx_001）｜五仙教｜受击触发
-- 源：内功特效清单.txt 行638-642（5 层）。
-- 受击几率获 5 秒"覆蛊"反伤状态：每次 48-119 阴伤（威力加成 TODO）。
-- 层4+ 蛊神劫：反伤期间被暴击回 100-119 气血（0.5 秒 CD）。
local M = {}
M.meta = { id = "buf_ng_wx_001", name = "蛊神诀", book = "ng_wx_001",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25, dur_ms = 5000, crit_throttle_ms = 500 }
M.levels = {
  [1] = { reflect = 48 },
  [2] = { reflect = 65 },
  [3] = { reflect = 82 },
  [4] = { reflect = 100, crit_heal = 100 },
  [5] = { reflect = 119, crit_heal = 119 },
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
  local out = {}
  if s.until_at and now < s.until_at then
    out.reflect = { amount = p.reflect, kind = "yin" }
    if ctx.crit and p.crit_heal
        and (not s.crit_at or now - s.crit_at >= M.defaults.crit_throttle_ms) then
      out.heal = p.crit_heal
      out.state = { until_at = s.until_at, crit_at = now }
    else
      out.state = { until_at = s.until_at }
    end
    return out
  end
  if roll(ctx, M.defaults.proc) then
    return { reflect = { amount = p.reflect, kind = "yin" },
             state = { until_at = now + M.defaults.dur_ms } }
  end
  return nil
end

return M
