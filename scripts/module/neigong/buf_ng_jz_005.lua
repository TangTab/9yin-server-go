-- 九天凤舞仙诀（buf_ng_jz_005）｜禁地｜受击触发
-- 源：内功特效清单.txt 行345-349（5 层）。
-- 凤舞鸾歌：攻击或被攻击几率 15-20 秒招式伤害 +20-33%（TODO：临时伤害%）+
--   免伤 +22%（v1：免伤窗口实现）。层5 凤舞九天（重力/空中/免疫击飞 TODO）。
local M = {}
M.meta = { id = "buf_ng_jz_005", name = "九天凤舞仙诀", book = "ng_jz_005",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { reduce = 0.22, dur_ms = 15000 },
  [2] = { reduce = 0.22, dur_ms = 16000 },
  [3] = { reduce = 0.22, dur_ms = 17000 },
  [4] = { reduce = 0.22, dur_ms = 18000 },
  [5] = { reduce = 0.22, dur_ms = 20000 },
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

local function on_event(ctx)
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

M.on_attack = on_event
M.on_victim = on_event

return M
