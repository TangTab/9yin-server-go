-- 血海刀罡（buf_ng_jh_307）｜江湖｜常驻/被动
-- 源：内功特效清单.txt 行214-221（8 层）。
-- 攻击几率附加额外伤害并使目标 8 秒持续流血（TODO：DoT 指令；v1 立即附伤近似）。
-- 层5+ 致死：流血期间恢复效果降低（TODO：治疗削弱 debuff）。
local M = {}
M.meta = { id = "buf_ng_jh_307", name = "血海刀罡", book = "ng_jh_307",
           category = "passive", max_tier = 8 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { amount = 45 },
  [2] = { amount = 58 },
  [3] = { amount = 73 },
  [4] = { amount = 90 },
  [5] = { amount = 108 },
  [6] = { amount = 128 },
  [7] = { amount = 150 },
  [8] = { amount = 174 },
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
