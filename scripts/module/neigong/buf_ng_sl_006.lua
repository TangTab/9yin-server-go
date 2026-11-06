-- 灭相禅功（buf_ng_sl_006）｜少林派｜招架触发
-- 源：内功特效清单.txt 行476-480（5 层）。
-- 苦渡众生：攻击几率 N 秒招式伤害与防御减免忽视 +8-15%（TODO：临时属性）。
-- 枯禅独饮：招架成功几率 N 秒持续恢复气血和招架值（v1：招架回血近似；招架值 TODO）。
-- 层5 行者无僵（净化转移）TODO。
local M = {}
M.meta = { id = "buf_ng_sl_006", name = "灭相禅功", book = "ng_sl_006",
           category = "parry", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { heal = 40,  dur_ms = 8000 },
  [2] = { heal = 50,  dur_ms = 9000 },
  [3] = { heal = 62,  dur_ms = 10000 },
  [4] = { heal = 76,  dur_ms = 12000 },
  [5] = { heal = 92,  dur_ms = 15000 },
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

function M.on_parry(ctx)
  if not roll(ctx, M.defaults.proc) then return nil end
  local p = lv(ctx)
  return { heal = p.heal, state = { until_at = (ctx.now or 0) + p.dur_ms } }
end

return M
