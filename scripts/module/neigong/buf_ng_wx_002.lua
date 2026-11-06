-- 花雨五灵典（buf_ng_wx_002）｜五仙教｜受击触发
-- 源：内功特效清单.txt 行643-649（7 层）。
-- 雨露均沾：攻/受几率中五灵蛊（降五维 6-25%；TODO：属性 debuff）；血浸花寒：攻击
--   中蛊目标附阴伤（蛊数越多越高；v1：以计数近似）。众蛊齐鸣 TODO。
local M = {}
M.meta = { id = "buf_ng_wx_002", name = "花雨五灵典", book = "ng_wx_002",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.25, dur_ms = 8000 }
M.levels = {
  [1] = { per_gu = 25 },
  [2] = { per_gu = 32 },
  [3] = { per_gu = 40 },
  [4] = { per_gu = 49 },
  [5] = { per_gu = 59 },
  [6] = { per_gu = 70 },
  [7] = { per_gu = 82 },
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
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.gus and s.until_at and now < s.until_at then
    return { extra_damage = { amount = s.gus * lv(ctx).per_gu, kind = "yin" },
             state = { gus = s.gus, until_at = s.until_at } }
  end
  if roll(ctx, M.defaults.proc) then
    return { state = { gus = 1, until_at = now + M.defaults.dur_ms } }
  end
  return nil
end

M.on_attack = on_event
M.on_victim = function(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.gus and s.until_at and now < s.until_at then
    return { state = { gus = math.min(s.gus + 1, 5), until_at = s.until_at } }
  end
  if roll(ctx, M.defaults.proc) then
    return { state = { gus = 1, until_at = now + M.defaults.dur_ms } }
  end
  return nil
end

return M
