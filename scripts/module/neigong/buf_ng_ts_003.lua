-- 灵雁录（buf_ng_ts_003）｜天山派｜叠加层数
-- 源：内功特效清单.txt 行548-551（4 层）。
-- 北雁南飞：攻击产生弹射真气（自身伤害 +3-6%/层 或目标受伤 +3-6%/层，叠 5 次，
--   30 秒回气；TODO：弹射/临时属性；v1：以攻击附伤近似受伤增幅）。
-- 身清如雁：内力消耗 -10% TODO。
local M = {}
M.meta = { id = "buf_ng_ts_003", name = "灵雁录", book = "ng_ts_003",
           category = "stack", max_tier = 4 }
M.defaults = { proc = 0.30, cd_ms = 30000 }
M.levels = {
  [1] = { per = 0.03 },
  [2] = { per = 0.04 },
  [3] = { per = 0.05 },
  [4] = { per = 0.06 },
}
M.MAX_LAYERS = 5

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
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.cd_until and now < s.cd_until then return nil end
  if not roll(ctx, M.defaults.proc) then return nil end
  return { extra_damage = { amount = math.floor(ctx.damage * p.per * M.MAX_LAYERS), kind = "external" },
           state = { cd_until = now + M.defaults.cd_ms } }
end

return M
