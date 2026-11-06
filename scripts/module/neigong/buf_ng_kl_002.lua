-- 骄阳诀（buf_ng_kl_002）｜昆仑派｜攻击触发
-- 源：内功特效清单.txt 行362-365（4 层）。
-- 每次攻击几率注入"骄阳真气"：持续损失气血（TODO：DoT；v1：立即阳伤一跳近似）。
-- 层4：真气溃散至 4 米范围（TODO：AoE/重复注入封锁）。
local M = {}
M.meta = { id = "buf_ng_kl_002", name = "骄阳诀", book = "ng_kl_002",
           category = "attack", max_tier = 4 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { yang = 40 },
  [2] = { yang = 50 },
  [3] = { yang = 62 },
  [4] = { yang = 76 },
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
  return { extra_damage = { amount = lv(ctx).yang, kind = "yang" } }
end

return M
