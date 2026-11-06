-- 冰心诀（buf_ng_jh_209）｜江湖｜受击触发
-- 源：内功特效清单.txt 行168-175（8 层）。
-- 招架几率触发冰心护体：4-11 秒内招架可恢复气血（恢复量为受击伤害 170-270%，
-- 基础恢复量上限 50-1000，0.5 秒最多一次）。层5+：招架耐力恢复大幅提高 TODO；
-- 层8：震散敌方招架耐力 TODO。
local M = {}
M.meta = { id = "buf_ng_jh_209", name = "冰心诀", book = "ng_jh_209",
           category = "victim", max_tier = 8 }
M.defaults = { proc = 0.25, throttle_ms = 500 }
M.levels = {
  [1] = { ratio = 1.70, cap = 50,   dur_ms = 4000 },
  [2] = { ratio = 1.90, cap = 100,  dur_ms = 5000 },
  [3] = { ratio = 2.10, cap = 170,  dur_ms = 6000 },
  [4] = { ratio = 2.30, cap = 236,  dur_ms = 7000 },
  [5] = { ratio = 2.50, cap = 298,  dur_ms = 8000 },
  [6] = { ratio = 2.50, cap = 388,  dur_ms = 9000 },
  [7] = { ratio = 2.50, cap = 869,  dur_ms = 10000 },
  [8] = { ratio = 2.70, cap = 1000, dur_ms = 11000 },
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

function M.on_parry(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then
    if s.last_heal_at and (now - s.last_heal_at) < M.defaults.throttle_ms then
      return nil
    end
    return { heal = math.min(math.floor(ctx.incoming * p.ratio), p.cap),
             state = { until_at = s.until_at, last_heal_at = now } }
  end
  if roll(ctx, M.defaults.proc) then
    return { heal = math.min(math.floor(ctx.incoming * p.ratio), p.cap),
             state = { until_at = now + p.dur_ms, last_heal_at = now } }
  end
  return nil
end

return M
