-- 溪月花香集（buf_ng_jz_006）｜禁地｜受击触发
-- 源：内功特效清单.txt 行350-354（5 层）。
-- 寒月惊魂：攻/受几率获 5 层"寒月"（每层免伤 6-10%，攻/受各消一层，15 秒）。
-- 暗香浸骨：招架层系 TODO（当前 NPC 不招架）。
local M = {}
M.meta = { id = "buf_ng_jz_006", name = "溪月花香集", book = "ng_jz_006",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20, dur_ms = 15000 }
M.levels = {
  [1] = { per_layer = 0.06 },
  [2] = { per_layer = 0.07 },
  [3] = { per_layer = 0.08 },
  [4] = { per_layer = 0.09 },
  [5] = { per_layer = 0.10 },
}
M.LAYERS = 5

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end
local function roll(ctx, p) return (ctx.rand and ctx.rand.r_proc or 1) < p end

local function on_event(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.layers and s.layers > 0 and s.until_at and now < s.until_at then
    return { reduce_pct = p.per_layer * s.layers, state = { layers = s.layers - 1, until_at = s.until_at } }
  end
  if roll(ctx, M.defaults.proc) then
    return { reduce_pct = p.per_layer * M.LAYERS,
             state = { layers = M.LAYERS - 1, until_at = now + M.defaults.dur_ms } }
  end
  return nil
end

M.on_attack = on_event
M.on_victim = on_event

return M
