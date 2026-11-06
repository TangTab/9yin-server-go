-- 影流诀（buf_ng_xmg_002）｜修罗场｜周期触发
-- 源：内功特效清单.txt 行677-683（7 层）。
-- 隐遁：周期获移速 +3%（TODO：移速）。流光破杀：攻击移除隐遁获忽视 +8-20% 8 秒
-- （TODO：临时忽视）。层5+ 和光同尘：受击获层减伤 3%/层叠 5-7（v1：实现受击层减伤）。
local M = {}
M.meta = { id = "buf_ng_xmg_002", name = "影流诀", book = "ng_xmg_002",
           category = "tick", max_tier = 7 }
M.levels = {
  [1] = { layers = 5 },
  [2] = { layers = 6 },
  [3] = { layers = 7 },
  [4] = { layers = 7 },
  [5] = { layers = 7 },
  [6] = { layers = 7 },
  [7] = { layers = 7 },
}
M.PER_LAYER = 0.03
M.DUR_MS = 8000

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if tier_of(ctx) < 5 then return nil end   -- 和光同尘层5+
  if s.until_at and now < s.until_at then
    return { reduce_pct = M.PER_LAYER * (s.layers or 0),
             state = { layers = math.min((s.layers or 0) + 1, p.layers), until_at = s.until_at } }
  end
  return { reduce_pct = M.PER_LAYER, state = { layers = 1, until_at = now + M.DUR_MS } }
end

return M
