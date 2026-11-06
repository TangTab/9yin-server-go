-- 邪刀煞劲（buf_ng_xd_001）｜邪刀门｜攻击触发
-- 源：内功特效清单.txt 行650-656（7 层）。
-- 暴走：攻击几率 N 秒伤害 +5-30% + 移速（TODO：临时伤害%/移速）。蚀体：目标恢复
--   效果 -25-50%（TODO：治疗削弱）。层5+ 血魔附体（低血吸血）TODO：临时吸血。
-- v1：以附伤近似伤害增幅。
local M = {}
M.meta = { id = "buf_ng_xd_001", name = "邪刀煞劲", book = "ng_xd_001",
           category = "attack", max_tier = 7 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { pct = 0.05 },
  [2] = { pct = 0.10 },
  [3] = { pct = 0.15 },
  [4] = { pct = 0.20 },
  [5] = { pct = 0.25 },
  [6] = { pct = 0.30 },
  [7] = { pct = 0.30 },
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
  return { extra_damage = { amount = math.floor(ctx.damage * lv(ctx).pct), kind = "external" } }
end

return M
