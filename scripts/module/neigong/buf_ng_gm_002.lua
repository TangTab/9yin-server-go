-- 寒玉诀（buf_ng_gm_002）｜古墓派｜受击触发
-- 源：内功特效清单.txt 行118-122（5 层）。
-- 易水寒：受击几率向攻击者倾注寒气：N 秒减速（TODO）+ 被攻击时额外受 10-20% 阴伤
-- （v1：当次反弹 incoming × pct 近似）。冰清玉洁：玉蜂蛰咬时长降低 TODO。
local M = {}
M.meta = { id = "buf_ng_gm_002", name = "寒玉诀", book = "ng_gm_002",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { pct = 0.10 },
  [2] = { pct = 0.12 },
  [3] = { pct = 0.14 },
  [4] = { pct = 0.17 },
  [5] = { pct = 0.20 },
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
  return { reflect = { amount = math.floor(ctx.incoming * lv(ctx).pct), kind = "yin" } }
end

return M
