-- 三皇劲·人（buf_ng_bh_003）｜八卦门｜受击触发
-- 源：内功特效清单.txt 行16-22（7 层）。同三皇劲·天，附伤为阴、层4+ 额外阳伤。
local M = {}
M.meta = { id = "buf_ng_bh_003", name = "三皇劲（人）", book = "ng_bh_003",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { dur_ms = 6000,  yin = 30,  absorb = 100 },
  [2] = { dur_ms = 7000,  yin = 36,  absorb = 120 },
  [3] = { dur_ms = 8000,  yin = 42,  absorb = 145 },
  [4] = { dur_ms = 9000,  yin = 50,  absorb = 170 },
  [5] = { dur_ms = 10000, yin = 58,  absorb = 200 },
  [6] = { dur_ms = 11000, yin = 66,  absorb = 230 },
  [7] = { dur_ms = 12000, yin = 75,  absorb = 260 },
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
  if s.mode == "tianshi" and s.until_at and now < s.until_at then
    return { extra_damage = { amount = p.yin, kind = "yin" } }
  end
  if s.mode or (s.until_at and now < s.until_at) then return nil end
  if roll(ctx, M.defaults.proc) then
    return { extra_damage = { amount = p.yin, kind = "yin" },
             state = { mode = "tianshi", until_at = now + p.dur_ms } }
  end
  return nil
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.mode == "dili" and s.until_at and now < s.until_at then
    return { shield = { amount = math.floor(p.absorb / 6), life_ms = 1500 } }
  end
  if s.mode or (s.until_at and now < s.until_at) then return nil end
  if roll(ctx, M.defaults.proc) then
    return { shield = { amount = p.absorb, life_ms = p.dur_ms },
             state = { mode = "dili", until_at = now + p.dur_ms } }
  end
  return nil
end

return M
