-- 三清真气（buf_ng_xj_001）｜修行会｜受击触发
-- 源：内功特效清单.txt 行662-671（10 层）。
-- 一气化三清：攻击 15% 几率命中大幅提升+免疫缴械封招（TODO：临时属性/免疫）。
-- 冲气为和：受击 15% 几率敌伤害 -1%/层（叠 5-15）+ 自身伤害 +0.5-1.2%/层
-- （TODO：双向临时属性；v1：受击减免 1%×层近似敌方削弱）。
local M = {}
M.meta = { id = "buf_ng_xj_001", name = "三清真气", book = "ng_xj_001",
           category = "victim", max_tier = 10 }
M.defaults = { proc = 0.15, dur_ms = 60000 }
M.levels = {
  [1] = { max = 5 },
  [2] = { max = 6 },
  [3] = { max = 7 },
  [4] = { max = 8 },
  [5] = { max = 10 },
  [6] = { max = 12 },
  [7] = { max = 13 },
  [8] = { max = 15 },
  [9] = { max = 15 },
  [10] = { max = 15 },
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
    return { reduce_pct = 0.01 * (s.layers or 0),
             state = { layers = math.min((s.layers or 0) + 1, p.max), until_at = s.until_at } }
  end
  if roll(ctx, M.defaults.proc) then
    return { reduce_pct = 0.01,
             state = { layers = 1, until_at = now + M.defaults.dur_ms } }
  end
  return nil
end

return M
