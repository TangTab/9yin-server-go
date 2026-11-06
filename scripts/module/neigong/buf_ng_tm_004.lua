-- 太素玄阴经（buf_ng_tm_004）｜唐门｜攻击触发
-- 源：内功特效清单.txt 行522-526（5 层）。
-- 攻击几率传入"阴毒"（16 秒微量伤 TODO：DoT）；阴毒爆发：几率引爆毒素总和
-- （毒素不消失）+ 硬直 + 损失 18-30% 瘀血 + 减速禁轻功 TODO。
-- v1：注入记层数，爆发时按层数结算阴伤。
local M = {}
M.meta = { id = "buf_ng_tm_004", name = "太素玄阴经", book = "ng_tm_004",
           category = "attack", max_tier = 5 }
M.defaults = { inject_proc = 0.25, burst_proc = 0.10 }
M.levels = {
  [1] = { per_dose = 30 },
  [2] = { per_dose = 38 },
  [3] = { per_dose = 47 },
  [4] = { per_dose = 57 },
  [5] = { per_dose = 68 },
}
M.MAX_DOSES = 8

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
  if (s.doses or 0) > 0 and roll(ctx, M.defaults.burst_proc) then
    return { extra_damage = { amount = s.doses * p.per_dose, kind = "yin" },
             state = { doses = s.doses } }
  end
  if roll(ctx, M.defaults.inject_proc) then
    return { state = { doses = math.min((s.doses or 0) + 1, M.MAX_DOSES) } }
  end
  return nil
end

return M
