-- 赤火功（buf_ng_mj_001）｜明教｜攻击触发
-- 源：内功特效清单.txt 行385-388（4 层）。
-- 赤火焚敌：攻击几率 5 秒提升招式伤害 10-30%（怒气越高越高，TODO：怒气快照与临时伤害%）。
-- 层4 怒火攻心：受击几率 5 秒产 10 怒气（30 秒 CD；TODO：怒气指令）。
-- v1：攻击几率附阳伤占位（等效伤害增幅近似）。
local M = {}
M.meta = { id = "buf_ng_mj_001", name = "赤火功", book = "ng_mj_001",
           category = "attack", max_tier = 4 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { yang = 35 },
  [2] = { yang = 45 },
  [3] = { yang = 56 },
  [4] = { yang = 68 },
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
  return { extra_damage = { amount = math.floor(ctx.damage * lv(ctx).yang / 200), kind = "yang" } }
end

return M
