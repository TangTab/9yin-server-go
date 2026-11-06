-- 飞天策（buf_ng_whz_001）｜王室｜攻击触发
-- 源：内功特效清单.txt 行618-622（5 层）。
-- 攻击几率"风阻"减速 20-35%（TODO：减速指令）；对减速目标大几率附伤（依赖减速，
--   v1：触发风阻即视为减速，后续攻击附伤）。层4+ 风行（自身加速+轻功返还）TODO。
local M = {}
M.meta = { id = "buf_ng_whz_001", name = "飞天策", book = "ng_whz_001",
           category = "attack", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { amount = 45 },
  [2] = { amount = 56 },
  [3] = { amount = 68 },
  [4] = { amount = 82 },
  [5] = { amount = 98 },
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
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.marked_until and now < s.marked_until then
    return { extra_damage = { amount = p.amount, kind = "external" },
             state = { marked_until = s.marked_until } }
  end
  if roll(ctx, M.defaults.proc) then
    return { state = { marked_until = now + 8000 } }
  end
  return nil
end

return M
