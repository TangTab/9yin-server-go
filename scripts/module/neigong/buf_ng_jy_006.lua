-- 修罗武经（buf_ng_jy_006）｜君子堂｜招架触发
-- 源：内功特效清单.txt 行320-324（5 层）。
-- 修罗临世：攻击几率 N 秒内攻击附 21-42% 阴伤（未招架目标；当前 NPC 恒未招架）。
-- 血祭凶灵：攻击几率"血祭"结束回放期间伤害 15-30% 阴伤（TODO：延迟结算；v1 未实现）。
local M = {}
M.meta = { id = "buf_ng_jy_006", name = "修罗武经", book = "ng_jy_006",
           category = "parry", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { pct = 0.21, dur_ms = 6000 },
  [2] = { pct = 0.25, dur_ms = 7000 },
  [3] = { pct = 0.30, dur_ms = 8000 },
  [4] = { pct = 0.36, dur_ms = 9000 },
  [5] = { pct = 0.42, dur_ms = 10000 },
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
    return { extra_damage = { amount = math.floor(ctx.damage * p.pct), kind = "yin" } }
  end
  if roll(ctx, M.defaults.proc) then
    return { extra_damage = { amount = math.floor(ctx.damage * p.pct), kind = "yin" },
             state = { until_at = now + p.dur_ms } }
  end
  return nil
end

return M
