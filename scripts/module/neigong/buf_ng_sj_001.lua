-- 太素脉法（buf_ng_sj_001）｜神圣｜受击触发
-- 源：内功特效清单.txt 行433-441（9 层）。
-- 金关玉锁：被攻击后 7/6 秒内不再受击，则恢复气血最大值 4-7.2% + 内力最大值 3-5.2%。
-- 回生换死：重伤时恢复 16 米友方气血内力（TODO：重伤事件/范围友方）。
local M = {}
M.meta = { id = "buf_ng_sj_001", name = "太素脉法", book = "ng_sj_001",
           category = "victim", max_tier = 9 }
M.levels = {
  [1] = { window_ms = 7000, hp_pct = 0.040, mp_pct = 0.030 },
  [2] = { window_ms = 7000, hp_pct = 0.045, mp_pct = 0.032 },
  [3] = { window_ms = 7000, hp_pct = 0.050, mp_pct = 0.034 },
  [4] = { window_ms = 7000, hp_pct = 0.055, mp_pct = 0.036 },
  [5] = { window_ms = 7000, hp_pct = 0.060, mp_pct = 0.040 },
  [6] = { window_ms = 7000, hp_pct = 0.062, mp_pct = 0.042 },
  [7] = { window_ms = 6000, hp_pct = 0.065, mp_pct = 0.045 },
  [8] = { window_ms = 6000, hp_pct = 0.068, mp_pct = 0.048 },
  [9] = { window_ms = 6000, hp_pct = 0.072, mp_pct = 0.052 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  return { state = { last_hit_at = ctx.now or 0, credited = false } }
end

function M.on_tick(ctx)
  local s = ctx.state or {}
  if not s.last_hit_at or s.credited then return nil end
  local now = ctx.now or 0
  local p = lv(ctx)
  if now - s.last_hit_at < p.window_ms then return nil end
  return { heal = math.floor(ctx.self.max_hp * p.hp_pct),
           mp_restore = math.floor(ctx.self.max_mp * p.mp_pct),
           state = { credited = true } }
end

return M
