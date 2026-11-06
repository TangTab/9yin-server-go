-- 天龙劲（buf_ng_cf_001）｜长风镖局｜受击触发
-- 源：内功特效清单.txt 行23-27（5 层）。
-- 拳劲：攻击几率近身威力提高 4-12 点叠 10 层（TODO：临时属性指令）。
-- 劲气护体：拳劲叠满后被攻击散去，几率衍生劲气护盾 120-400 点 10 秒。
local M = {}
M.meta = { id = "buf_ng_cf_001", name = "天龙劲", book = "ng_cf_001",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.30 }
M.levels = {
  [1] = { absorb = 120 },
  [2] = { absorb = 180 },
  [3] = { absorb = 240 },
  [4] = { absorb = 320 },
  [5] = { absorb = 400 },
}
M.MAX_STACKS = 10

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
  local s = ctx.state or {}
  if s.full then return nil end
  if not roll(ctx, M.defaults.proc) then return nil end
  local stacks = (s.stacks or 0) + 1
  if stacks >= M.MAX_STACKS then
    return { state = { stacks = M.MAX_STACKS, full = true } }
  end
  return { state = { stacks = stacks } }
end

function M.on_victim(ctx)
  local s = ctx.state or {}
  if not s.full then return nil end
  if not roll(ctx, M.defaults.proc) then return nil end
  return { shield = { amount = lv(ctx).absorb, life_ms = 10000 },
           state = { stacks = 0, full = false } }
end

return M
