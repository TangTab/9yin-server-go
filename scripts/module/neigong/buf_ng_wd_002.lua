-- 内丹术（buf_ng_wd_002）｜武当｜常驻/被动
-- 源：内功特效清单.txt 行583-586（4 层）。
-- 层1-3：内力 <40% 时攻击 12-20% 几率回 2% 内力；>40% 时攻击附当前内力 2-4% 伤害（4 秒最多一次）。
-- 层4：攻击 20% 几率回 2% 内力，且每次攻击都附当前内力 4% 伤害（不再限内力区间）。
local M = {}
M.meta = { id = "buf_ng_wd_002", name = "内丹术", book = "ng_wd_002",
           category = "passive", max_tier = 4 }
M.levels = {
  [1] = { proc = 0.12, mp_pct = 0.02, dmg_pct = 0.02, always = false, throttle_ms = 4000 },
  [2] = { proc = 0.16, mp_pct = 0.02, dmg_pct = 0.03, always = false, throttle_ms = 4000 },
  [3] = { proc = 0.20, mp_pct = 0.02, dmg_pct = 0.04, always = false, throttle_ms = 4000 },
  [4] = { proc = 0.20, mp_pct = 0.02, dmg_pct = 0.04, always = true,  throttle_ms = 0 },
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
  local low_mp = ctx.self.max_mp > 0 and ctx.self.mp / ctx.self.max_mp < 0.40
  local out = {}
  if low_mp then
    if roll(ctx, p.proc) then
      out.mp_restore = math.floor(ctx.self.max_mp * p.mp_pct)
    end
  else
    local throttled = p.throttle_ms > 0 and s.last_dmg_at
        and (now - s.last_dmg_at) < p.throttle_ms
    if p.always or not throttled then
      out.extra_damage = { amount = math.floor(ctx.self.mp * p.dmg_pct), kind = "external" }
      if p.throttle_ms > 0 then out.state = { last_dmg_at = now } end
    end
  end
  if next(out) == nil then return nil end
  return out
end

return M
