-- 七绝经（buf_ng_tm_001）｜唐门｜攻击触发
-- 源：内功特效清单.txt 行510-513（4 层）。
-- 每出招 8/7/6/5 次，下次攻击恢复内力上限 5% 内力（层4 击退 TODO：控制指令）。
local M = {}
M.meta = { id = "buf_ng_tm_001", name = "七绝经", book = "ng_tm_001",
           category = "attack", max_tier = 4 }
M.levels = {
  [1] = { need = 8 },
  [2] = { need = 7 },
  [3] = { need = 6 },
  [4] = { need = 5 },
}
M.MP_PCT = 0.05

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
    return { mp_restore = math.floor(ctx.self.max_mp * M.MP_PCT), state = { count = 0 } }
  end
  return { state = { count = count } }
end

return M
