-- 伏养气功（buf_ng_ws_001）｜王室书塾｜受击触发
-- 源：内功特效清单.txt 行630-637（8 层）。
-- 真气流转：攻击 15% 几率下次出招不耗内力并回 105-378 内力（20 秒 CD；免耗 TODO；
--   层6+ 附带伤害 +12-20% TODO）。真气化生：气血 <15% 被击触发内力承伤
-- （TODO：内力池承伤；v1：触发时减免 50% 该次伤害近似）。
local M = {}
M.meta = { id = "buf_ng_ws_001", name = "伏养气功", book = "ng_ws_001",
           category = "victim", max_tier = 8 }
M.defaults = { proc = 0.15, cd_ms = 20000 }
M.levels = {
  [1] = { mp = 105 },
  [2] = { mp = 131 },
  [3] = { mp = 159 },
  [4] = { mp = 188 },
  [5] = { mp = 218 },
  [6] = { mp = 273 },
  [7] = { mp = 312 },
  [8] = { mp = 378 },
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
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.cd_until and now < s.cd_until then return nil end
  if not roll(ctx, M.defaults.proc) then return nil end
  return { mp_restore = lv(ctx).mp, state = { cd_until = now + M.defaults.cd_ms } }
end

function M.on_victim(ctx)
  if ctx.self.hp >= ctx.self.max_hp * 0.15 then return nil end
  return { reduce_pct = 0.50 }
end

return M
