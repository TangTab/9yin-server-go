-- 沧澜秘谱（buf_ng_ssg_002）｜水上漂｜受击触发
-- 源：内功特效清单.txt 行489-495（7 层）。
-- 惊涛：攻击几率自伤 +12-30% 或敌防 -12-30%（TODO：临时属性）。
-- 骇浪：受击几率自防 +12-30% 或敌伤 -12-30%（v1：受击减免窗口近似）。
-- 沧澜百变（层5+ 双态并存回复/流失）TODO。
local M = {}
M.meta = { id = "buf_ng_ssg_002", name = "沧澜秘谱", book = "ng_ssg_002",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.20, dur_ms = 7000 }
M.levels = {
  [1] = { reduce = 0.12 },
  [2] = { reduce = 0.15 },
  [3] = { reduce = 0.18 },
  [4] = { reduce = 0.21 },
  [5] = { reduce = 0.25 },
  [6] = { reduce = 0.30 },
  [7] = { reduce = 0.30 },
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
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then
    return { reduce_pct = p.reduce }
  end
  if roll(ctx, M.defaults.proc) then
    return { reduce_pct = p.reduce,
             state = { until_at = now + M.defaults.dur_ms + tier_of(ctx) * 1000 } }
  end
  return nil
end

return M
