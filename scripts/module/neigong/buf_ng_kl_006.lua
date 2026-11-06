-- 昆仑引（buf_ng_kl_006）｜昆仑派｜招架触发
-- 源：内功特效清单.txt 行380-384（5 层）。
-- 攻击未招架目标几率种"昆仑劲"（每秒阴伤+减速 30-50%，引线 10 秒；距离越近几率越高
--   TODO：距离快照）。化劲为气：引线存在时持续回血（v1：窗口回血）。
-- 层3+ 进退维谷（距离判定爆发/定身）TODO。
local M = {}
M.meta = { id = "buf_ng_kl_006", name = "昆仑引", book = "ng_kl_006",
           category = "parry", max_tier = 5 }
M.defaults = { proc = 0.25, dur_ms = 10000 }
M.levels = {
  [1] = { yin = 35, heal = 30 },
  [2] = { yin = 44, heal = 38 },
  [3] = { yin = 54, heal = 47 },
  [4] = { yin = 66, heal = 57 },
  [5] = { yin = 80, heal = 68 },
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
    return { extra_damage = { amount = p.yin, kind = "yin" },
             heal = math.floor(p.heal / 3) }
  end
  if roll(ctx, M.defaults.proc) then
    return { extra_damage = { amount = p.yin, kind = "yin" },
             state = { until_at = now + M.defaults.dur_ms } }
  end
  return nil
end

return M
