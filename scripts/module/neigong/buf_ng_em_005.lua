-- 大乘涅磐功（buf_ng_em_005）｜峨眉｜受击触发
-- 源：内功特效清单.txt 行64-68（5 层）。
-- 涅槃化境：攻击或被攻击几率获得护体盾 6-10 秒（周围持续阳伤、降内防 TODO；层5 破盾爆炸 TODO）。
local M = {}
M.meta = { id = "buf_ng_em_005", name = "大乘涅磐功", book = "ng_em_005",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { absorb = 140, dur_ms = 6000 },
  [2] = { absorb = 175, dur_ms = 7000 },
  [3] = { absorb = 215, dur_ms = 8000 },
  [4] = { absorb = 260, dur_ms = 9000 },
  [5] = { absorb = 310, dur_ms = 10000 },
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

local function shield_of(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.cd_until and now < s.cd_until then return nil end
  if roll(ctx, M.defaults.proc) then
    return { shield = { amount = p.absorb, life_ms = p.dur_ms },
             state = { cd_until = now + p.dur_ms } }
  end
  return nil
end

M.on_attack = shield_of
M.on_victim = shield_of

return M
