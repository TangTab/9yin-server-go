-- 照夜日月轮（buf_ng_whz_002）｜王室｜受击触发
-- 源：内功特效清单.txt 行623-629（7 层）。
-- 沙海燎日（叠层伤害+流金铄石爆发）/塞鸿逐月（叠层抵御+余层回血）——依赖叠层与
--   周期爆发系统（TODO：多层状态机）。层5+ 遮天蔽日（致死抵挡）TODO。
-- v1：塞鸿逐月简化——受击几率获 4-7 层鸿雁，每层抵御 2-4% 招式伤害，15 秒后
--   余层每层回 1-2% 最大气血。
local M = {}
M.meta = { id = "buf_ng_whz_002", name = "照夜日月轮", book = "ng_whz_002",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.25, dur_ms = 15000 }
M.levels = {
  [1] = { layers = 4, per_layer = 0.020, heal_pct = 0.010 },
  [2] = { layers = 4, per_layer = 0.025, heal_pct = 0.010 },
  [3] = { layers = 5, per_layer = 0.027, heal_pct = 0.015 },
  [4] = { layers = 5, per_layer = 0.032, heal_pct = 0.015 },
  [5] = { layers = 6, per_layer = 0.034, heal_pct = 0.018 },
  [6] = { layers = 6, per_layer = 0.038, heal_pct = 0.018 },
  [7] = { layers = 7, per_layer = 0.040, heal_pct = 0.020 },
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
  if s.layers and s.layers > 0 and s.until_at and now < s.until_at then
    return { reduce_pct = p.per_layer * s.layers,
             state = { layers = s.layers - 1, until_at = s.until_at } }
  end
  if roll(ctx, M.defaults.proc) then
    return { reduce_pct = p.per_layer * p.layers,
             state = { layers = p.layers - 1, until_at = now + M.defaults.dur_ms } }
  end
  return nil
end

function M.on_tick(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if not s.until_at or now < s.until_at or not s.layers then return nil end
  local p = lv(ctx)
  return { heal = math.floor(ctx.self.max_hp * p.heal_pct * s.layers),
           state = { layers = 0 } }
end

return M
