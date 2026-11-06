-- 常相照（buf_ng_tl_001）｜天龙寺｜受击触发
-- 源：内功特效清单.txt 行503-509（7 层）。
-- 佛性禅心：伤害几率获 N 层，每次伤害附刚伤消一层（v1 实现）。
-- 佛光西来：受击几率护身之气受伤 -20-50% 15 秒（受控/被击飞清除近似：受控不清除 TODO）。
-- 打坐恢复翻倍/内照 TODO。
local M = {}
M.meta = { id = "buf_ng_tl_001", name = "常相照", book = "ng_tl_001",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.20, dur_ms = 15000 }
M.levels = {
  [1] = { layers = 4, gang = 40,  reduce = 0.20 },
  [2] = { layers = 4, gang = 50,  reduce = 0.25 },
  [3] = { layers = 5, gang = 62,  reduce = 0.30 },
  [4] = { layers = 5, gang = 76,  reduce = 0.35 },
  [5] = { layers = 6, gang = 92,  reduce = 0.40 },
  [6] = { layers = 6, gang = 110, reduce = 0.45 },
  [7] = { layers = 7, gang = 130, reduce = 0.50 },
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
  if s.chan and s.chan > 0 then
    return { extra_damage = { amount = p.gang, kind = "gang" },
             state = { chan = s.chan - 1, guard_until = s.guard_until } }
  end
  if roll(ctx, M.defaults.proc) then
    return { extra_damage = { amount = p.gang, kind = "gang" },
             state = { chan = p.layers - 1, guard_until = s.guard_until } }
  end
  return nil
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.guard_until and now < s.guard_until then
    return { reduce_pct = p.reduce, state = { chan = s.chan, guard_until = s.guard_until } }
  end
  if roll(ctx, M.defaults.proc) then
    return { reduce_pct = p.reduce,
             state = { chan = s.chan, guard_until = now + M.defaults.dur_ms } }
  end
  return nil
end

return M
