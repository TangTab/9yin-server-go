-- 紫气天罗（buf_ng_hs_002）｜华山派｜招架触发
-- 源：内功特效清单.txt 行128-134（7 层）。
-- 天罗劲：攻击未招架目标几率附阴伤（当前 NPC 恒未招架）；紫旋气：消减怒气 TODO；
-- 层5+ 气运无息：攻/受几率清除冷却 TODO。天罗劲窗口 5-11 秒后进冷却（v1：触发窗口+内置 CD）。
local M = {}
M.meta = { id = "buf_ng_hs_002", name = "紫气天罗", book = "ng_hs_002",
           category = "parry", max_tier = 7 }
M.defaults = { proc = 0.25, cd_ms = 15000 }
M.levels = {
  [1] = { dur_ms = 5000,  yin = 40 },
  [2] = { dur_ms = 6000,  yin = 50 },
  [3] = { dur_ms = 7000,  yin = 61 },
  [4] = { dur_ms = 8000,  yin = 74 },
  [5] = { dur_ms = 9000,  yin = 88 },
  [6] = { dur_ms = 10000, yin = 104 },
  [7] = { dur_ms = 11000, yin = 122 },
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
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then
    return { extra_damage = { amount = p.yin, kind = "yin" } }
  end
  if s.cd_until and now < s.cd_until then return nil end
  if roll(ctx, M.defaults.proc) then
    return { extra_damage = { amount = p.yin, kind = "yin" },
             state = { until_at = now + p.dur_ms, cd_until = now + p.dur_ms + M.defaults.cd_ms } }
  end
  return nil
end

return M
