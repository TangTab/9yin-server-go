-- 玄侠要诀（buf_ng_jh_210）｜江湖｜攻击触发
-- 源：内功特效清单.txt 行176-181（6 层）。
-- 攻击几率对目标造成额外伤害，对非我类者加倍（侠义值系统 TODO：v1 统一按普通倍率）。
-- 玄狂/玄邪/玄恶要诀（211/212/213）同款换皮。
local M = {}
M.meta = { id = "buf_ng_jh_210", name = "玄侠要诀", book = "ng_jh_210",
           category = "attack", max_tier = 6 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { amount = 40 },
  [2] = { amount = 55 },
  [3] = { amount = 72 },
  [4] = { amount = 90 },
  [5] = { amount = 110 },
  [6] = { amount = 132 },
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
  if not roll(ctx, M.defaults.proc) then return nil end
  return { extra_damage = { amount = lv(ctx).amount, kind = "external" } }
end

return M
