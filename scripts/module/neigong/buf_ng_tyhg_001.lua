-- 碧游诀（buf_ng_tyhg_001）｜天宇门｜受击触发
-- 源：内功特效清单.txt 行567-571（5 层）。
-- 游云：受击几率获 5-10 秒"游云"（移速 +12-30% TODO），期间再被击回复 48-119 气血
-- （最多 5 次；威力加成 TODO）。层4+ 惊龙（伤害 +17-20% 8-10 秒）TODO：临时伤害%。
local M = {}
M.meta = { id = "buf_ng_tyhg_001", name = "碧游诀", book = "ng_tyhg_001",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { heal = 48,  dur_ms = 5000 },
  [2] = { heal = 65,  dur_ms = 6000 },
  [3] = { heal = 82,  dur_ms = 7000 },
  [4] = { heal = 100, dur_ms = 8000 },
  [5] = { heal = 119, dur_ms = 10000 },
}
M.MAX_HEALS = 5

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
  if s.until_at and now < s.until_at and (s.heals or 0) > 0 then
    return { heal = p.heal, state = { until_at = s.until_at, heals = s.heals - 1 } }
  end
  if roll(ctx, M.defaults.proc) then
    return { heal = p.heal, state = { until_at = now + p.dur_ms, heals = M.MAX_HEALS - 1 } }
  end
  return nil
end

return M
