-- 贝叶禅经（buf_ng_dm_002）｜大漠鬼城｜受击触发
-- 源：内功特效清单.txt 行40-46（7 层）。
-- 苦经灭生：受击几率 5-12 秒"同死"，结束对对方结算期间受伤 12-30%（TODO：tick 无攻击者，
--   v1 未实现）；枯禅绝死：受击几率"独生"结算回血期间受伤 12-30%。
-- 层5+ 生死同参：同死独生并存免疫击退击倒 TODO。
local M = {}
M.meta = { id = "buf_ng_dm_002", name = "贝叶禅经", book = "ng_dm_002",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { dur_ms = 5000,  pct = 0.12 },
  [2] = { dur_ms = 6000,  pct = 0.15 },
  [3] = { dur_ms = 7000,  pct = 0.18 },
  [4] = { dur_ms = 8000,  pct = 0.21 },
  [5] = { dur_ms = 9000,  pct = 0.25 },
  [6] = { dur_ms = 10000, pct = 0.30 },
  [7] = { dur_ms = 12000, pct = 0.30 },
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
  if s.mode == "dusheng" and s.until_at and now < s.until_at then
    return { state = { mode = "dusheng", until_at = s.until_at, acc = (s.acc or 0) + ctx.incoming } }
  end
  if roll(ctx, M.defaults.proc) then
    return { state = { mode = "dusheng", until_at = now + p.dur_ms, acc = ctx.incoming } }
  end
  return nil
end

function M.on_tick(ctx)
  local s = ctx.state or {}
  if s.mode ~= "dusheng" then return nil end
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then return nil end
  local p = lv(ctx)
  return { heal = math.floor((s.acc or 0) * p.pct), state = {} }
end

return M
