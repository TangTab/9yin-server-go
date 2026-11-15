-- 摩诃心经（buf_ng_dm_001）｜大漠鬼城｜受击触发
-- 源：内功特效清单.txt 行35-39（5 层）。
-- 受击几率鼓动真气连续 3 次阳伤害（v1：对攻击者一次结算 3 跳之和；范围伤害 TODO）。
-- 层4+ 摩诃无量：真气消散时回淤血（bruise_restore percent=10% 层厚，
-- 官方「目标越多回复越多」按目标数放大的范围判定 TODO，v1 单目标近似）。
local M = {}
M.meta = { id = "buf_ng_dm_001", name = "摩诃心经", book = "ng_dm_001",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { per_hit = 40 },
  [2] = { per_hit = 55 },
  [3] = { per_hit = 72 },
  [4] = { per_hit = 92, bruise = 0.10 },
  [5] = { per_hit = 115, bruise = 0.10 },
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
  local p = lv(ctx)
  if p.bruise then
    -- 摩诃无量：真气消散（3 跳结算）同拍回淤血层 10%。
    return { reflect = { amount = p.per_hit * 3, kind = "yang" },
             bruise_restore = { percent = p.bruise } }
  end
  return { reflect = { amount = p.per_hit * 3, kind = "yang" } }
end

return M
