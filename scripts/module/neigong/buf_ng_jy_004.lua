-- 混天宝纲（buf_ng_jy_004）｜君子堂｜招架触发
-- 源：内功特效清单.txt 行310-314（5 层）。
-- 追杀：每 25 秒下 7-12 次攻击忽视招架 +16-28%（TODO：忽视招架指令）并附大量刚伤
-- （v1：刚伤窗口实现）。玄天逆鳞：被直接攻击几率受伤 -10-18% 10-16 秒。
local M = {}
M.meta = { id = "buf_ng_jy_004", name = "混天宝纲", book = "ng_jy_004",
           category = "parry", max_tier = 5 }
M.defaults = { proc = 0.20, interval_ms = 25000 }
M.levels = {
  [1] = { hits = 7,  gang = 80,  reduce = 0.10, reduce_ms = 10000 },
  [2] = { hits = 8,  gang = 95,  reduce = 0.12, reduce_ms = 13000 },
  [3] = { hits = 9,  gang = 112, reduce = 0.14, reduce_ms = 14000 },
  [4] = { hits = 10, gang = 130, reduce = 0.16, reduce_ms = 15000 },
  [5] = { hits = 12, gang = 150, reduce = 0.18, reduce_ms = 16000 },
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

function M.on_tick(ctx)
  local since = ctx.since_last_ms or -1
  if since >= 0 and since < M.defaults.interval_ms then return nil end
  return { state = { last_at = ctx.now or 0 } }
end

function M.on_attack(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  if s.last_at and (s.hits or 0) > 0 then
    return { extra_damage = { amount = p.gang, kind = "gang" },
             state = { last_at = s.last_at, hits = s.hits - 1 } }
  end
  if s.last_at then
    return { extra_damage = { amount = p.gang, kind = "gang" },
             state = { last_at = s.last_at, hits = p.hits - 1 } }
  end
  return nil
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.reduce_until and now < s.reduce_until then
    return { reduce_pct = p.reduce }
  end
  if roll(ctx, M.defaults.proc) then
    return { reduce_pct = p.reduce, state = { reduce_until = now + p.reduce_ms } }
  end
  return nil
end

return M
