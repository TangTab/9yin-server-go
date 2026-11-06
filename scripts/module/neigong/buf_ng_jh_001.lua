-- 坐忘功（buf_ng_jh_001）｜江湖｜常驻/被动
-- 源：内功特效清单.txt 行135-139（5 层）。
-- 有几率忽略 20-250 点伤害（层3 起额外 30-40%）。
-- v1：层1-3 以常驻平减近似"几率忽略"；层4-5 以周期护盾近似"一段时间内忽略"。
local M = {}
M.meta = { id = "buf_ng_jh_001", name = "坐忘功", book = "ng_jh_001",
           category = "passive", max_tier = 5 }
M.levels = {
  [1] = { flat = 20 },
  [2] = { flat = 35 },
  [3] = { flat = 40 },
  [4] = { shield = 200 },
  [5] = { shield = 250 },
}
M.SHIELD_INTERVAL_MS = 30000

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.passive(ctx)
  local p = lv(ctx)
  if p.flat and p.flat > 0 then
    return { damage_reduction_add = p.flat }
  end
  return nil
end

function M.on_tick(ctx)
  local p = lv(ctx)
  if not p.shield then return nil end
  local since = ctx.since_last_ms or -1
  if since >= 0 and since < M.SHIELD_INTERVAL_MS then return nil end
  return { shield = { amount = p.shield, life_ms = M.SHIELD_INTERVAL_MS },
           state = { last_at = ctx.now or 0 } }
end

return M
