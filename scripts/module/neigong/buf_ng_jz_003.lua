-- 忘情诀（buf_ng_jz_003）｜禁地｜攻击触发
-- 源：内功特效清单.txt 行336-339（4 层）。
-- 剑道名家：剑类武器伤害 +13-30（v1：被动攻击加成近似；武器类型判定 TODO）。
-- 剑我两忘：攻击几率进入状态每击 +2-5 点、每 2 秒再提高（TODO：动态成长）。
local M = {}
M.meta = { id = "buf_ng_jz_003", name = "忘情诀", book = "ng_jz_003",
           category = "attack", max_tier = 4 }
M.levels = {
  [1] = { attack_add = 13 },
  [2] = { attack_add = 18 },
  [3] = { attack_add = 24 },
  [4] = { attack_add = 30 },
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
