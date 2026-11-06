-- 心斋秘篆（buf_ng_jh_214）｜江湖｜受击触发
-- 源：内功特效清单.txt 行200-207（8 层）。
-- 心剑：受击获得 5-10 层，5-9 秒内每次受击至少反弹 35-103 点外功（威力加成 TODO），
-- 最多反弹 N 次，30 秒 CD。层5+ 气斋：同时获得 5-8 层，5 秒后回复 554-1107 气血，
-- 期间出招 1 次消散 1 层。
local M = {}
M.meta = { id = "buf_ng_jh_214", name = "心斋秘篆", book = "ng_jh_214",
           category = "victim", max_tier = 8 }
M.levels = {
  [1] = { layers = 5, reflect = 35,  dur_ms = 5000, qi = 0,   qi_heal = 0 },
  [2] = { layers = 6, reflect = 43,  dur_ms = 5000, qi = 0,   qi_heal = 0 },
  [3] = { layers = 7, reflect = 53,  dur_ms = 5000, qi = 0,   qi_heal = 0 },
  [4] = { layers = 8, reflect = 59,  dur_ms = 5000, qi = 0,   qi_heal = 0 },
  [5] = { layers = 8, reflect = 68,  dur_ms = 6000, qi = 5,   qi_heal = 554 },
  [6] = { layers = 8, reflect = 79,  dur_ms = 7000, qi = 6,   qi_heal = 719 },
  [7] = { layers = 8, reflect = 91,  dur_ms = 8000, qi = 7,   qi_heal = 906 },
  [8] = { layers = 10, reflect = 103, dur_ms = 9000, qi = 8,  qi_heal = 1107 },
}
M.CD_MS = 30000

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.charges and s.charges > 0 and s.until_at and now < s.until_at then
    local out = { reflect = { amount = p.reflect, kind = "external" },
                  state = { charges = s.charges - 1, until_at = s.until_at,
                            qi = s.qi, qi_at = s.qi_at } }
    return out
  end
  if s.cd_until and now < s.cd_until then return nil end
  local state = { charges = p.layers, until_at = now + p.dur_ms, cd_until = now + M.CD_MS }
  if p.qi > 0 then
    state.qi = p.qi
    state.qi_at = now + 5000
  end
  return { reflect = { amount = p.reflect, kind = "external" }, state = state }
end

function M.on_attack(ctx)
  local s = ctx.state or {}
  if not s.qi or s.qi <= 0 then return nil end
  return { state = { charges = s.charges, until_at = s.until_at, cd_until = s.cd_until,
                     qi = s.qi - 1, qi_at = s.qi_at } }
end

function M.on_tick(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if not s.qi or s.qi <= 0 or not s.qi_at then return nil end
  if now < s.qi_at then return nil end
  local p = lv(ctx)
  return { heal = math.floor(p.qi_heal * s.qi / math.max(1, p.qi)),
           state = { charges = s.charges, until_at = s.until_at, cd_until = s.cd_until,
                     qi = 0 } }
end

return M
