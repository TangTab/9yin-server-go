-- 拍影功（buf_ng_jl_005）｜极乐谷｜常驻/被动
-- 源：内功特效清单.txt 行285-289（5 层）。
-- 系风碟血：攻击几率使目标持续流血 6-10 秒（TODO：DoT；v1：立即阴伤一跳近似）。
-- 捕影嗜血：攻击流血目标吸血 40-200% 武器伤害（0.5 秒 CD）。
local M = {}
M.meta = { id = "buf_ng_jl_005", name = "拍影功", book = "ng_jl_005",
           category = "passive", max_tier = 5 }
M.defaults = { proc = 0.25, throttle_ms = 500 }
M.levels = {
  [1] = { bleed = 40,  drain = 0.40 },
  [2] = { bleed = 52,  drain = 0.80 },
  [3] = { bleed = 66,  drain = 1.20 },
  [4] = { bleed = 82,  drain = 1.60 },
  [5] = { bleed = 100, drain = 2.00 },
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
  local out = {}
  if s.bleed_until and now < s.bleed_until then
    if not s.drain_at or (now - s.drain_at) >= M.defaults.throttle_ms then
      out.heal = math.floor(ctx.damage * p.drain)
      out.state = { bleed_until = s.bleed_until, drain_at = now }
    end
  elseif roll(ctx, M.defaults.proc) then
    out.extra_damage = { amount = p.bleed, kind = "yin" }
    out.state = { bleed_until = now + 6000 + tier_of(ctx) * 1000 }
  end
  if next(out) == nil then return nil end
  return out
end

return M
