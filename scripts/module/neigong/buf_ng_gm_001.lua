-- 凌霄天女诀（buf_ng_gm_001）｜古墓派｜受击触发
-- 源：内功特效清单.txt 行111-117（7 层）。
-- 阴进：攻击几率 N 秒攻击附阴伤（内功威力加成 TODO）；阳退：受击几率 N 秒持续回血
-- （v1：单次回复近似）。层5+ 阴阳同流免疫负面 TODO。
local M = {}
M.meta = { id = "buf_ng_gm_001", name = "凌霄天女诀", book = "ng_gm_001",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { dur_ms = 5000,  yin = 35, heal = 45 },
  [2] = { dur_ms = 6000,  yin = 44, heal = 56 },
  [3] = { dur_ms = 7000,  yin = 54, heal = 68 },
  [4] = { dur_ms = 8000,  yin = 66, heal = 82 },
  [5] = { dur_ms = 9000,  yin = 80, heal = 98 },
  [6] = { dur_ms = 10000, yin = 96, heal = 116 },
  [7] = { dur_ms = 11000, yin = 114, heal = 136 },
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
  if s.yin_until and now < s.yin_until then
    return { extra_damage = { amount = p.yin, kind = "yin" } }
  end
  if roll(ctx, M.defaults.proc) then
    return { extra_damage = { amount = p.yin, kind = "yin" },
             state = { yin_until = now + p.dur_ms } }
  end
  return nil
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.heal_until and now < s.heal_until then
    return { heal = math.floor(p.heal / 3) }
  end
  if roll(ctx, M.defaults.proc) then
    return { heal = p.heal, state = { heal_until = now + p.dur_ms } }
  end
  return nil
end

return M
