-- 淇奥诀（buf_ng_kl_004）｜昆仑派｜受击触发
-- 源：内功特效清单.txt 行370-374（5 层）。
-- 失调音律：攻击几率对目标及其 4 米范围敌阴伤（TODO：AoE；v1：单体阴伤 + 短调息）。
-- 悦耳音律：受击几率自身与队友回血（TODO：队友；v1：自身回血 + 短调息）。
-- 层5 琴瑟调和（联动增强）TODO。
local M = {}
M.meta = { id = "buf_ng_kl_004", name = "淇奥诀", book = "ng_kl_004",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.25, cd_ms = 8000 }
M.levels = {
  [1] = { yin = 45,  heal = 45 },
  [2] = { yin = 56,  heal = 56 },
  [3] = { yin = 69,  heal = 69 },
  [4] = { yin = 84,  heal = 84 },
  [5] = { yin = 100, heal = 100 },
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

local function gated(ctx, key)
  local s = ctx.state or {}
  local now = ctx.now or 0
  local k = "cd_" .. key
  if s[k] and now < s[k] then return nil, s end
  if not roll(ctx, M.defaults.proc) then return nil, s end
  local state = {}
  for kk, vv in pairs(s) do state[kk] = vv end
  state[k] = now + M.defaults.cd_ms
  return state
end

function M.on_attack(ctx)
  local state = gated(ctx, "atk")
  if not state then return nil end
  return { extra_damage = { amount = lv(ctx).yin, kind = "yin" }, state = state }
end

function M.on_victim(ctx)
  local state = gated(ctx, "vic")
  if not state then return nil end
  return { heal = lv(ctx).heal, state = state }
end

return M
