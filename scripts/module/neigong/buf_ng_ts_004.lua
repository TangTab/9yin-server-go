-- 梅影抄（buf_ng_ts_004）｜天山派｜受击触发
-- 源：内功特效清单.txt 行552-556（5 层）。
-- 寒梅护雪：受击后每秒获 1 层护雪寒气（减伤 1-3%/层，叠 6 层 12 秒；转移减速 TODO）。
-- 梅影相顾：状态结束后 6-8 秒攻击附阴伤（寒气越重附伤越高；威力加成 TODO）。
local M = {}
M.meta = { id = "buf_ng_ts_004", name = "梅影抄", book = "ng_ts_004",
           category = "victim", max_tier = 5 }
M.levels = {
  [1] = { per_layer = 0.010, yin = 73 },
  [2] = { per_layer = 0.015, yin = 91 },
  [3] = { per_layer = 0.020, yin = 110 },
  [4] = { per_layer = 0.025, yin = 129 },
  [5] = { per_layer = 0.030, yin = 156 },
}
M.MAX_LAYERS = 6
M.SNOW_MS = 12000
M.BLOOM_MS = 6000

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
  if s.snow_until and now < s.snow_until then
    local layers = math.min((s.layers or 0) + 1, M.MAX_LAYERS)
    return { reduce_pct = p.per_layer * layers,
             state = { layers = layers, snow_until = s.snow_until,
                       bloom_until = s.snow_until + M.BLOOM_MS } }
  end
  if not s.bloom_until or now >= (s.bloom_until or 0) then
    -- 新一轮寒气
    return { reduce_pct = p.per_layer,
             state = { layers = 1, snow_until = now + M.SNOW_MS,
                       bloom_until = now + M.SNOW_MS + M.BLOOM_MS } }
  end
  return nil
end

function M.on_attack(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.bloom_until and now < s.bloom_until and (s.layers or 0) > 0 then
    return { extra_damage = { amount = lv(ctx).yin, kind = "yin" } }
  end
  return nil
end

return M
