-- 毒蛤经（buf_ng_jh_205）｜江湖｜招架触发
-- 源：内功特效清单.txt 行152-159（8 层）。
-- 未招架对方攻击（当前 NPC 恒未招架）时几率施加蟾毒：9 秒叠 8 层，叠满引爆毒素总和。
-- 层5+：引爆时打断+定身 TODO；层8：暴击额外叠层 TODO。
local M = {}
M.meta = { id = "buf_ng_jh_205", name = "毒蛤经", book = "ng_jh_205",
           category = "parry", max_tier = 8 }
M.defaults = { proc = 0.30 }
M.levels = {
  [1] = { per_stack = 22 },
  [2] = { per_stack = 28 },
  [3] = { per_stack = 35 },
  [4] = { per_stack = 44 },
  [5] = { per_stack = 54 },
  [6] = { per_stack = 66 },
  [7] = { per_stack = 80 },
  [8] = { per_stack = 96 },
}
M.MAX_STACKS = 8

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
  local p = lv(ctx)
  local s = ctx.state or {}
  local stacks = (s.stacks or 0) + 1
  if stacks >= M.MAX_STACKS then
    return { extra_damage = { amount = M.MAX_STACKS * p.per_stack, kind = "yin" },
             state = { stacks = 0 } }
  end
  return { state = { stacks = stacks } }
end

return M
