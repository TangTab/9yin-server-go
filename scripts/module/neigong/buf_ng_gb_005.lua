-- 伏龙诀（buf_ng_gb_005）｜丐帮｜攻击触发
-- 源：内功特效清单.txt 行96-100（5 层）。
-- 青龙出水：每击伤害少量提高（v1：被动攻击 +5-9 近似"少量"）。
-- 擒龙式：出招几率暴击大幅提高+溅射+减速（TODO：临时暴击/溅射指令）；层5 将军令 TODO。
local M = {}
M.meta = { id = "buf_ng_gb_005", name = "伏龙诀", book = "ng_gb_005",
           category = "attack", max_tier = 5 }
M.levels = {
  [1] = { attack_add = 5 },
  [2] = { attack_add = 6 },
  [3] = { attack_add = 7 },
  [4] = { attack_add = 8 },
  [5] = { attack_add = 9 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.passive(ctx)
  return { attack_add = lv(ctx).attack_add }
end

return M
