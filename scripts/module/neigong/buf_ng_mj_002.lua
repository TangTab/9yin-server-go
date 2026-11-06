-- 阳炎功（buf_ng_mj_002）｜明教｜受击触发
-- 源：内功特效清单.txt 行389-392（4 层）。
-- 阳炎护体：受击几率 5 秒反弹 194-468 伤害，期间受伤的 15-30% 转化治疗
-- （禁轻功/定身/昏迷下治疗翻倍近似：controlled 时 heal 比例 ×2）。
local M = {}
M.meta = { id = "buf_ng_mj_002", name = "阳炎功", book = "ng_mj_002",
           category = "victim", max_tier = 4 }
M.defaults = { proc = 0.20, dur_ms = 5000 }
M.levels = {
  [1] = { reflect = 194, heal_pct = 0.15 },
  [2] = { reflect = 275, heal_pct = 0.20 },
  [3] = { reflect = 368, heal_pct = 0.25 },
  [4] = { reflect = 468, heal_pct = 0.30 },
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
  local heal_pct = ctx.controlled and p.heal_pct * 2 or p.heal_pct
  if s.until_at and now < s.until_at then
    return { reflect = { amount = p.reflect, kind = "yang" },
             heal = math.floor(ctx.incoming * heal_pct) }
  end
  if roll(ctx, M.defaults.proc) then
    return { reflect = { amount = p.reflect, kind = "yang" },
             heal = math.floor(ctx.incoming * heal_pct),
             state = { until_at = now + M.defaults.dur_ms } }
  end
  return nil
end

return M
