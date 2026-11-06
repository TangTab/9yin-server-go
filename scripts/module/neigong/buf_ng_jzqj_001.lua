-- 通慧功·奇技（buf_ng_jzqj_001）｜禁地｜叠加层数
-- 源：内功特效清单.txt 行355-357（3 层）。
-- 攻击 40% 几率回内力 + 每次回蓝叠 1 层"悟"（暴击率 +1-2%，最多 5-6 层，
--   TODO：临时暴击；v1：层数记录 + 回蓝）。层3：持续恢复期间暴伤 +10% TODO。
local M = {}
M.meta = { id = "buf_ng_jzqj_001", name = "通慧功", book = "ng_jzqj_001",
           category = "stack", max_tier = 3 }
M.defaults = { proc = 0.40, dur_ms = 30000 }
M.levels = {
  [1] = { mp = 20, max_wu = 5 },
  [2] = { mp = 26, max_wu = 6 },
  [3] = { mp = 32, max_wu = 6 },
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
  if not roll(ctx, M.defaults.proc) then return nil end
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  local wu = s.wu or 0
  if not s.at or (now - s.at) > M.defaults.dur_ms then
    wu = 0
  end
  wu = math.min(wu + 1, p.max_wu)
  return { mp_restore = p.mp, state = { wu = wu, at = now } }
end

return M
