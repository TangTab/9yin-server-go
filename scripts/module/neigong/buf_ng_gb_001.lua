-- 逍遥游（buf_ng_gb_001）｜丐帮｜周期触发
-- 源：内功特效清单.txt 行79-82（4 层）。
-- 每隔 16/13/10/8 秒，下次攻击 1.5 倍伤害 + 附加阳伤（减速 TODO）。
-- v1：以 extra_damage = 50% 伤害 + 固定阳伤 实现"1.5 倍 + 附伤"。
local M = {}
M.meta = { id = "buf_ng_gb_001", name = "逍遥游", book = "ng_gb_001",
           category = "tick", max_tier = 4 }
M.levels = {
  [1] = { interval_ms = 16000, yang = 60 },
  [2] = { interval_ms = 13000, yang = 75 },
  [3] = { interval_ms = 10000, yang = 92 },
  [4] = { interval_ms = 8000,  yang = 110 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_tick(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local since = ctx.since_last_ms or -1
  if since >= 0 and since < p.interval_ms then return nil end
  return { state = { last_at = ctx.now or 0, charged = true } }
end

function M.on_attack(ctx)
  local s = ctx.state or {}
  if not s.charged then return nil end
  local p = lv(ctx)
  return { extra_damage = { amount = math.floor(ctx.damage * 0.5) + p.yang, kind = "yang" },
           state = { charged = false } }
end

return M
