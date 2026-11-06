-- 天游诀（buf_ng_tyhg_002）｜天宇门｜受击触发
-- 源：内功特效清单.txt 行572-578（7 层）。
-- 春露之滋：受击几率 N 秒每秒回 1-1.5% 气血（v1：单次回复近似；持续 TODO）。
-- 秋霜之烈：攻击几率伤害 +10-30%（TODO：临时伤害%）。春露秋霜联动 TODO。
local M = {}
M.meta = { id = "buf_ng_tyhg_002", name = "天游诀", book = "ng_tyhg_002",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { hp_pct = 0.010 },
  [2] = { hp_pct = 0.010 },
  [3] = { hp_pct = 0.010 },
  [4] = { hp_pct = 0.010 },
  [5] = { hp_pct = 0.015 },
  [6] = { hp_pct = 0.015 },
  [7] = { hp_pct = 0.015 },
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
  if not roll(ctx, M.defaults.proc) then return nil end
  return { heal = math.floor(ctx.self.max_hp * lv(ctx).hp_pct) }
end

return M
