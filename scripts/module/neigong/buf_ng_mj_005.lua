-- 明王宝策（buf_ng_mj_005）｜明教｜受击触发
-- 源：内功特效清单.txt 行402-406（5 层）。
-- 不动明王：7 秒不攻击后受击，7 秒内招式伤害 -10-30%（30 秒回气）。
-- 大轮明王：N 次出招后 7 秒伤害 +10%（TODO：临时伤害%）。层5 如来多变 TODO。
-- v1：不动明王以"7 秒未出招 → 受击减免窗口"实现。
local M = {}
M.meta = { id = "buf_ng_mj_005", name = "明王宝策", book = "ng_mj_005",
           category = "victim", max_tier = 5 }
M.levels = {
  [1] = { reduce = 0.10 },
  [2] = { reduce = 0.15 },
  [3] = { reduce = 0.20 },
  [4] = { reduce = 0.25 },
  [5] = { reduce = 0.30 },
}
M.STILL_MS = 7000
M.SHIELD_MS = 7000
M.CD_MS = 30000

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_attack(ctx)
  return { state = { last_attack_at = ctx.now or 0 } }
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.guard_until and now < s.guard_until then
    return { reduce_pct = p.reduce }
  end
  if s.cd_until and now < s.cd_until then return nil end
  local last = s.last_attack_at or 0
  if now - last < M.STILL_MS then return nil end
  return { reduce_pct = p.reduce,
           state = { guard_until = now + M.SHIELD_MS, cd_until = now + M.CD_MS } }
end

return M
