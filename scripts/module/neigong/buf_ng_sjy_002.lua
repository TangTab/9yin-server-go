-- 鬼谷玄功录（buf_ng_sjy_002）｜圣军｜受击触发
-- 源：内功特效清单.txt 行447-453（7 层）。
-- 鬼谋霸纵：攻击几率目标闪避 -50-100% + 受伤 +7-30%（TODO：debuff）。
-- 玄兵冲横：受击几率 3-6 秒受伤 -10-40%（地面翻倍 TODO；v1：直接取上限近似半程）。
-- 鬼谷纵横：结束时随机控制 TODO。
local M = {}
M.meta = { id = "buf_ng_sjy_002", name = "鬼谷玄功录", book = "ng_sjy_002",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { reduce = 0.10, dur_ms = 3000 },
  [2] = { reduce = 0.15, dur_ms = 3000 },
  [3] = { reduce = 0.20, dur_ms = 4000 },
  [4] = { reduce = 0.25, dur_ms = 4000 },
  [5] = { reduce = 0.30, dur_ms = 5000 },
  [6] = { reduce = 0.35, dur_ms = 5000 },
  [7] = { reduce = 0.40, dur_ms = 6000 },
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
