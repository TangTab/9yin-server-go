-- 噬月神鉴（buf_ng_jl_006）｜极乐谷｜受击触发
-- 源：内功特效清单.txt 行290-294（5 层）。
-- 孤狼吞月：短时间内连击同一目标 6 次，下 3 次攻击其吸血 30-60%。
-- 噬光留影：>70% 血攻击几率获"凝血"（出招内力-2%/层）；<30% 血受击消耗全部凝血回血
-- （TODO：内力减免；v1：凝血每层回血近似）。
local M = {}
M.meta = { id = "buf_ng_jl_006", name = "噬月神鉴", book = "ng_jl_006",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20, window_ms = 10000 }
M.levels = {
  [1] = { drain = 0.30, heal_per = 30,  max_ning = 8 },
  [2] = { drain = 0.36, heal_per = 38,  max_ning = 9 },
  [3] = { drain = 0.42, heal_per = 47,  max_ning = 10 },
  [4] = { drain = 0.50, heal_per = 58,  max_ning = 12 },
  [5] = { drain = 0.60, heal_per = 70,  max_ning = 15 },
}
M.COMBO_NEED = 6
M.DRAIN_HITS = 3

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
  -- 孤狼吞月：同目标连击计数
  if s.drain_hits and s.drain_hits > 0 then
    return { heal = math.floor(ctx.damage * p.drain),
             state = { drain_hits = s.drain_hits - 1, ning = s.ning, combo = 0 } }
  end
  local combo, target = s.combo or 0, s.target
  if target ~= ctx.target.oid or (s.combo_at and now - s.combo_at > M.defaults.window_ms) then
    combo = 0
  end
  combo = combo + 1
  local state = { combo = combo, combo_at = now, target = ctx.target.oid, ning = s.ning or 0 }
  local out = {}
  if combo >= M.COMBO_NEED then
    state.combo = 0
    state.drain_hits = M.DRAIN_HITS
  end
  -- 噬光留影：高血量获凝血
  if ctx.self.hp > ctx.self.max_hp * 0.70 and roll(ctx, M.defaults.proc) then
    state.ning = math.min((s.ning or 0) + 1, p.max_ning)
  end
  out.state = state
  return out
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  if ctx.self.hp < ctx.self.max_hp * 0.30 and (s.ning or 0) > 0 then
    return { heal = s.ning * p.heal_per, state = { ning = 0, combo = s.combo } }
  end
  return nil
end

return M
