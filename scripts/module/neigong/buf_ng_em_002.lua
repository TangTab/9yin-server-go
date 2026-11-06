-- 五符经（buf_ng_em_002）｜峨眉｜周期触发
-- 源：内功特效清单.txt 行51-54（4 层）。
-- 激发真气：每 20 秒吸收 120-300 点伤害；层4：内力上限 +150（罡气 +10、反伤 TODO）。
local M = {}
M.meta = { id = "buf_ng_em_002", name = "五符经", book = "ng_em_002",
           category = "tick", max_tier = 4 }
M.levels = {
  [1] = { absorb = 120 },
  [2] = { absorb = 180 },
  [3] = { absorb = 260 },
  [4] = { absorb = 300 },
}
M.INTERVAL_MS = 20000

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_tick(ctx)
  local s = ctx.state or {}
  local since = ctx.since_last_ms or -1
  if since >= 0 and since < M.INTERVAL_MS then return nil end
  return { shield = { amount = lv(ctx).absorb, life_ms = M.INTERVAL_MS },
           state = { last_at = ctx.now or 0 } }
end

function M.passive(ctx)
  if tier_of(ctx) >= 4 then
    return { max_mp_add = 150 }
  end
  return nil
end

return M
