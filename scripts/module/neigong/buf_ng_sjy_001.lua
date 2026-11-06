-- 定军武略（buf_ng_sjy_001）｜圣军｜攻击触发
-- 源：内功特效清单.txt 行442-446（5 层）。
-- 明心定神：攻击几率定神（伤害 +10-20% 8-10 秒；TODO：临时伤害%）。
-- 处乱不惊：脱离定神 20 秒内首次被暴击反震 300-400（TODO：内功威力加成）。
-- v1：以定神窗口内附伤近似伤害增幅 + 被暴击反震。
local M = {}
M.meta = { id = "buf_ng_sjy_001", name = "定军武略", book = "ng_sjy_001",
           category = "attack", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { pct = 0.10, dur_ms = 8000,  reflect = 0 },
  [2] = { pct = 0.12, dur_ms = 8000,  reflect = 0 },
  [3] = { pct = 0.14, dur_ms = 8000,  reflect = 0 },
  [4] = { pct = 0.17, dur_ms = 8000,  reflect = 300 },
  [5] = { pct = 0.20, dur_ms = 10000, reflect = 400 },
}
M.AFTER_MS = 20000

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
    return { extra_damage = { amount = math.floor(ctx.damage * p.pct), kind = "external" } }
  end
  if roll(ctx, M.defaults.proc) then
    return { extra_damage = { amount = math.floor(ctx.damage * p.pct), kind = "external" },
             state = { until_at = now + p.dur_ms, end_at = now + p.dur_ms } }
  end
  return nil
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if not ctx.crit or p.reflect <= 0 then return nil end
  if not s.end_at or now - s.end_at > M.AFTER_MS or s.reflect_used then return nil end
  return { reflect = { amount = p.reflect, kind = "external" },
           state = { end_at = s.end_at, reflect_used = true } }
end

return M
