-- 云天谱（buf_ng_ts_005）｜天山派｜受击触发
-- 源：内功特效清单.txt 行557-561（5 层）。
-- 豪气冲天：攻击几率打入 5 层"豪气断脉"气劲持续刚伤（TODO：DoT/反弹机制）。
-- 正气干云：受击几率 N 秒持续回复气血（v1：单次回复近似；威力加成 TODO）。
local M = {}
M.meta = { id = "buf_ng_ts_005", name = "云天谱", book = "ng_ts_005",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { gang = 45,  heal = 48 },
  [2] = { gang = 56,  heal = 60 },
  [3] = { gang = 68,  heal = 74 },
  [4] = { gang = 82,  heal = 90 },
  [5] = { gang = 98,  heal = 108 },
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

function M.on_attack(ctx)
  if not roll(ctx, M.defaults.proc) then return nil end
  return { extra_damage = { amount = lv(ctx).gang, kind = "gang" } }
end

function M.on_victim(ctx)
  if not roll(ctx, M.defaults.proc) then return nil end
  return { heal = lv(ctx).heal }
end

return M
