-- 两仪护心功·奇技（buf_ng_wdqj_001）｜武当｜受击触发
-- 源：内功特效清单.txt 行608-610（3 层）。
-- 被攻击 10% 几率激发真气：吸收 10-20% 自身最大气血；散去部分回内力（TODO：散去回内力）。
-- 层3：另一部分冲击周围目标定身封招（TODO：范围控制指令）。
local M = {}
M.meta = { id = "buf_ng_wdqj_001", name = "两仪护心功", book = "ng_wdqj_001",
           category = "victim", max_tier = 3 }
M.defaults = { proc = 0.10 }
M.levels = {
  [1] = { absorb_pct = 0.10 },
  [2] = { absorb_pct = 0.15 },
  [3] = { absorb_pct = 0.20 },
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
  if not roll(ctx, M.defaults.proc) then return nil end
  return { shield = { amount = math.floor(ctx.self.max_hp * lv(ctx).absorb_pct),
                      life_ms = 12000 } }
end

return M
