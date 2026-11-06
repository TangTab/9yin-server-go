-- 云梦天香诀（buf_ng_nl_001）｜念萝芭｜受击触发
-- 源：内功特效清单.txt 行414-418（5 层）。
-- 被攻击几率引渡香瘴：N 秒持续流失气血（TODO：DoT）。香销蚀骨：被身中香瘴的目标攻击
--   几率受 10-20% 伤害反噬（层4-5 必定）。v1：标记攻击者，被其再攻击时反弹。
local M = {}
M.meta = { id = "buf_ng_nl_001", name = "云梦天香诀", book = "ng_nl_001",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { pct = 0.10, dur_ms = 4000, certain = false },
  [2] = { pct = 0.12, dur_ms = 5000, certain = false },
  [3] = { pct = 0.14, dur_ms = 6000, certain = false },
  [4] = { pct = 0.17, dur_ms = 7000, certain = true },
  [5] = { pct = 0.20, dur_ms = 8000, certain = true },
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
  local out = {}
  if s.marked == ctx.attacker.oid and s.until_at and now < s.until_at then
    if p.certain or roll(ctx, M.defaults.proc) then
      out.reflect = { amount = math.floor(ctx.incoming * p.pct), kind = "yin" }
    end
    out.state = { marked = s.marked, until_at = s.until_at }
    return next(out) == nil and nil or out
  end
  if roll(ctx, M.defaults.proc) then
    out.state = { marked = ctx.attacker.oid, until_at = now + p.dur_ms }
    return out
  end
  return nil
end

return M
