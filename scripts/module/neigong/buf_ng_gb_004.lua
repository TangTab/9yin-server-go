-- 酒雨神功（buf_ng_gb_004）｜丐帮｜攻击触发
-- 源：内功特效清单.txt 行91-95（5 层）。
-- 酒雨：开战几率溅"酒雨"降防闪避+被暴击（TODO：debuff 指令）。
-- 酒醉/酒神：触发酒雨后每次出招消耗一层酒神，剩余层提供附阳伤（v1：层数 × 阳伤近似）。
-- 酒醒：暴击几率 +10%（TODO：临时属性）。
local M = {}
M.meta = { id = "buf_ng_gb_004", name = "酒雨神功", book = "ng_gb_004",
           category = "attack", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { per_stack = 18 },
  [2] = { per_stack = 22 },
  [3] = { per_stack = 27 },
  [4] = { per_stack = 33 },
  [5] = { per_stack = 40 },
}
M.MAX_JIUSHEN = 10

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
  local p = lv(ctx)
  local s = ctx.state or {}
  local stacks = s.jiushen or 0
  if stacks <= 0 then
    if not roll(ctx, M.defaults.proc) then return nil end
    return { state = { jiushen = M.MAX_JIUSHEN } }
  end
  local extra = math.floor(stacks * p.per_stack * 0.5)
  return { extra_damage = { amount = extra, kind = "yang" },
           state = { jiushen = stacks - 1 } }
end

return M
