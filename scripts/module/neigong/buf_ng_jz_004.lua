-- 葬花玄功录（buf_ng_jz_004）｜禁地｜攻击触发
-- 源：内功特效清单.txt 行340-344（5 层）。
-- 攻击几率注入"飞花真气"潜伏 6 秒后绽放（范围阴伤+打断+经脉受损 TODO：延迟/AoE）。
-- v1：注入时立即柔伤一跳近似"同时造成少量柔伤害"。
local M = {}
M.meta = { id = "buf_ng_jz_004", name = "葬花玄功录", book = "ng_jz_004",
           category = "attack", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { rou = 50, max_marks = 2 },
  [2] = { rou = 62, max_marks = 2 },
  [3] = { rou = 76, max_marks = 2 },
  [4] = { rou = 92, max_marks = 2 },
  [5] = { rou = 110, max_marks = 3 },
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
  if not roll(ctx, M.defaults.proc) then return nil end
  return { extra_damage = { amount = lv(ctx).rou, kind = "rou" } }
end

return M
