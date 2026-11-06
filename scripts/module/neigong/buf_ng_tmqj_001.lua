-- 七绝经·奇技（buf_ng_tmqj_001）｜唐门｜攻击触发
-- 源：内功特效清单.txt 行537-539（3 层）。
-- 每出招 5/4/3 次，下次攻击恢复内力上限 5-7% 并忽视 30-80% 内外功防御减免
-- （TODO：临时忽视；层3 定身 TODO）。v1：回内力部分。
local M = {}
M.meta = { id = "buf_ng_tmqj_001", name = "七绝经", book = "ng_tmqj_001",
           category = "attack", max_tier = 3 }
M.levels = {
  [1] = { need = 5, mp_pct = 0.05 },
  [2] = { need = 4, mp_pct = 0.06 },
  [3] = { need = 3, mp_pct = 0.07 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_attack(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local count = (s.count or 0) + 1
  if count >= p.need then
    return { mp_restore = math.floor(ctx.self.max_mp * p.mp_pct), state = { count = 0 } }
  end
  return { state = { count = count } }
end

return M
