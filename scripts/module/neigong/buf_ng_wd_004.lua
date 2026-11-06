-- 纯阳无极功（buf_ng_wd_004）｜武当｜受击触发
-- 源：内功特效清单.txt 行591-595（5 层）。
-- 震伤：攻击时几率使目标"震伤"（每次被攻击额外阳伤害，持续 11-16 秒）。
-- v1 近似：触发后窗口内目标每次被本角色命中附加阳伤。TODO：内伤加重（出招计数封招禁轻功）。
local M = {}
M.meta = { id = "buf_ng_wd_004", name = "纯阳无极功", book = "ng_wd_004",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { extra = 30, dur_ms = 11000 },
  [2] = { extra = 40, dur_ms = 12000 },
  [3] = { extra = 50, dur_ms = 13000 },
  [4] = { extra = 60, dur_ms = 14000 },
  [5] = { extra = 70, dur_ms = 16000 },
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
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then
    return { extra_damage = { amount = p.extra, kind = "yang" } }
  end
  if roll(ctx, M.defaults.proc) then
    return { extra_damage = { amount = p.extra, kind = "yang" },
             state = { until_at = now + p.dur_ms } }
  end
  return nil
end

return M
