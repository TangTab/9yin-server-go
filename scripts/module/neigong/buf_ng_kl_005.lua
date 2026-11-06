-- 风起诀（buf_ng_kl_005）｜昆仑派｜叠加层数
-- 源：内功特效清单.txt 行375-379（5 层）。
-- 清风拂面：暴击/被暴击获层（受暴伤减免 5-10%/层，最多 5 层，15 秒后转"意气风发"）。
-- 意气风发：每层招式与暴伤 +5-10%，暴击消层（TODO：临时暴伤/暴击减免指令；
--   v1：受击免伤窗口近似）。层5 仙风道骨（调息缩减）TODO。
local M = {}
M.meta = { id = "buf_ng_kl_005", name = "风起诀", book = "ng_kl_005",
           category = "stack", max_tier = 5 }
M.defaults = { dur_ms = 15000 }
M.levels = {
  [1] = { per_layer = 0.05 },
  [2] = { per_layer = 0.06 },
  [3] = { per_layer = 0.07 },
  [4] = { per_layer = 0.08 },
  [5] = { per_layer = 0.10 },
}
M.MAX_LAYERS = 5

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  if not ctx.crit then return nil end
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  local layers = s.layers or 0
  if not s.until_at or now >= s.until_at then
    layers = 0
  end
  layers = math.min(layers + 1, M.MAX_LAYERS)
  return { reduce_pct = p.per_layer * layers,
           state = { layers = layers, until_at = now + M.defaults.dur_ms } }
end

return M
