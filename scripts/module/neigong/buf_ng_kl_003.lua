-- 清谈诀（buf_ng_kl_003）｜昆仑派｜攻击触发
-- 源：内功特效清单.txt 行366-369（4 层）。
-- 用内力抵御自身：几率震伤攻击者"步履蹒跚"（出招震伤自己/移动减速 TODO：debuff）。
-- 矫健身姿：轻功返还轻功值 TODO。v1：受击反弹阴伤占位近似"震伤"。
local M = {}
M.meta = { id = "buf_ng_kl_003", name = "清谈诀", book = "ng_kl_003",
           category = "attack", max_tier = 4 }
M.defaults = { proc = 0.20, throttle_ms = 2000 }
M.levels = {
  [1] = { yin = 30 },
  [2] = { yin = 38 },
  [3] = { yin = 47 },
  [4] = { yin = 57 },
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
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.at and (now - s.at) < M.defaults.throttle_ms then return nil end
  if not roll(ctx, M.defaults.proc) then return nil end
  return { reflect = { amount = lv(ctx).yin, kind = "yin" },
           state = { at = now } }
end

return M
