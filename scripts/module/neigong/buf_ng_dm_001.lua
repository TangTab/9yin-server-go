-- 摩诃心经（buf_ng_dm_001）｜大漠鬼城｜受击触发
-- 源：内功特效清单.txt 行35-39（5 层）。
-- 受击几率鼓动真气连续 3 次阳伤害（v1：对攻击者一次结算 3 跳之和；范围伤害 TODO）。
-- 层4+ 摩诃无量：真气消散回淤血（TODO：淤血层指令）。
local M = {}
M.meta = { id = "buf_ng_dm_001", name = "摩诃心经", book = "ng_dm_001",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { per_hit = 40 },
  [2] = { per_hit = 55 },
  [3] = { per_hit = 72 },
  [4] = { per_hit = 92 },
  [5] = { per_hit = 115 },
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
  return { reflect = { amount = lv(ctx).per_hit * 3, kind = "yang" } }
end

return M
