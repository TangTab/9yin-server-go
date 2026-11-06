-- 绛雪心经（buf_ng_jh_105）｜江湖｜受击触发
-- 源：内功特效清单.txt 行140-143（4 层）。
-- 命中注入绛雪真气：被侵扰者每次受击可能激发寒气僵直（TODO：控制/持续状态系统），
-- 寒气存在时每次被攻击受阴伤。v1：命中几率注入标记，标记期间被击反弹阴伤近似。
-- 打坐化解/惊世骇俗直接化解 TODO。
local M = {}
M.meta = { id = "buf_ng_jh_105", name = "绛雪心经", book = "ng_jh_105",
           category = "victim", max_tier = 4 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { yin = 30, dur_ms = 6000 },
  [2] = { yin = 38, dur_ms = 7000 },
  [3] = { yin = 47, dur_ms = 8000 },
  [4] = { yin = 57, dur_ms = 9000 },
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
  if s.marked_until and now < s.marked_until then
    return { extra_damage = { amount = p.yin, kind = "yin" } }
  end
  if roll(ctx, M.defaults.proc) then
    return { state = { marked_until = now + p.dur_ms } }
  end
  return nil
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.marked_until and now < s.marked_until then
    return { reflect = { amount = math.floor(p.yin * 0.6), kind = "yin" } }
  end
  return nil
end

return M
