-- 六道凌霄独尊诀（buf_ng_jh_501）｜江湖｜受击触发
-- 源：内功特效清单.txt 行222-229（8 层）。
-- 势破八荒：攻击几率"势惊"10 秒每秒 44-226 阳伤（v1：立即一跳近似；每秒持续 TODO；
-- "体颤"随机控制 TODO）。气贯六合：受击几率"气涌"10 秒每秒回血（v1：单次回复近似）。
-- 层5+ 唯我独尊（体格判定）TODO；层8 内功防御减免忽视 TODO。
local M = {}
M.meta = { id = "buf_ng_jh_501", name = "六道凌霄独尊诀", book = "ng_jh_501",
           category = "victim", max_tier = 8 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = { yang = 44,  heal = 44 },
  [2] = { yang = 56,  heal = 56 },
  [3] = { yang = 70,  heal = 70 },
  [4] = { yang = 84,  heal = 84 },
  [5] = { yang = 103, heal = 103 },
  [6] = { yang = 134, heal = 134 },
  [7] = { yang = 171, heal = 171 },
  [8] = { yang = 226, heal = 226 },
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
  return { extra_damage = { amount = lv(ctx).yang, kind = "yang" } }
end

function M.on_victim(ctx)
  if not roll(ctx, M.defaults.proc) then return nil end
  return { heal = lv(ctx).heal }
end

return M
