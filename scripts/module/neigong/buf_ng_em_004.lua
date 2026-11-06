-- 冰肌玉骨功（buf_ng_em_004）｜峨眉｜受击触发
-- 源：内功特效清单.txt 行59-63（5 层）。
-- 寒意侵袭：攻击者打断+硬直（TODO：控制指令）；玉洁冰清：受击几率移除负面（TODO：
--   净化指令）+ 恢复招架耐力（TODO：耐力指令）+ 外功招架/内功化解 +9-18（TODO：临时属性）
--   + 10-15 秒攻击附阴伤（v1：以触发后窗口附伤实现）。
local M = {}
M.meta = { id = "buf_ng_em_004", name = "冰肌玉骨功", book = "ng_em_004",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { yin = 30, dur_ms = 10000 },
  [2] = { yin = 36, dur_ms = 11000 },
  [3] = { yin = 42, dur_ms = 12000 },
  [4] = { yin = 48, dur_ms = 13000 },
  [5] = { yin = 55, dur_ms = 15000 },
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
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then return nil end
  if roll(ctx, M.defaults.proc) then
    return { state = { until_at = now + p.dur_ms } }
  end
  return nil
end

function M.on_attack(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then
    return { extra_damage = { amount = p.yin, kind = "yin" } }
  end
  return nil
end

return M
