-- 五毒奇经（buf_ng_tm_003）｜唐门｜攻击触发
-- 源：内功特效清单.txt 行518-521（4 层）。
-- 攻击几率向双手积蓄毒素，3 次后下一击注入：每秒 18-39 阴伤 + 结束毒发 144-312 阴伤
-- （TODO：DoT/减速禁轻功；v1：积蓄 3 次后一次性毒发近似；远程威力加成 TODO）。
-- 识毒高手（免疫中毒）TODO。
local M = {}
M.meta = { id = "buf_ng_tm_003", name = "五毒奇经", book = "ng_tm_003",
           category = "attack", max_tier = 4 }
M.defaults = { proc = 0.30 }
M.levels = {
  [1] = { per_sec = 18, burst = 144 },
  [2] = { per_sec = 25, burst = 200 },
  [3] = { per_sec = 31, burst = 248 },
  [4] = { per_sec = 39, burst = 312 },
}
M.CHARGE_NEED = 3

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
  local p = lv(ctx)
  local s = ctx.state or {}
  local charge = (s.charge or 0) + 1
  if charge >= M.CHARGE_NEED then
    return { extra_damage = { amount = p.burst, kind = "yin" }, state = { charge = 0 } }
  end
  return { state = { charge = charge } }
end

return M
