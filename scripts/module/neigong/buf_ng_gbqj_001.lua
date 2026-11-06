-- 逍遥心法·奇技（buf_ng_gbqj_001）｜丐帮｜招架触发
-- 源：内功特效清单.txt 行108-110（3 层）。
-- 每隔 15/10/5 秒下次攻击 2-2.5 倍伤害 + 附阳伤（击中未招架敌人定身 TODO：控制指令）。
-- 层3：残废目标使周围敌方加速 TODO。
local M = {}
M.meta = { id = "buf_ng_gbqj_001", name = "逍遥心法", book = "ng_gbqj_001",
           category = "parry", max_tier = 3 }
M.levels = {
  [1] = { interval_ms = 15000, mult = 2.0, yang = 80 },
  [2] = { interval_ms = 10000, mult = 2.0, yang = 100 },
  [3] = { interval_ms = 5000,  mult = 2.5, yang = 125 },
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
  local since = ctx.since_last_ms or -1
  if since >= 0 and since < p.interval_ms then return nil end
  return { state = { last_at = ctx.now or 0, charged = true } }
end

function M.on_attack(ctx)
  local s = ctx.state or {}
  if not s.charged then return nil end
  local p = lv(ctx)
  local extra = math.floor(ctx.damage * (p.mult - 1)) + p.yang
  return { extra_damage = { amount = extra, kind = "yang" },
           state = { charged = false } }
end

return M
