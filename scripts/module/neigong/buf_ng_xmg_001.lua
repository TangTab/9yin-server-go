-- 风幻录（buf_ng_xmg_001）｜修罗场｜攻击触发
-- 源：内功特效清单.txt 行672-676（5 层）。
-- 攻击发出"风幻真气"5 秒每秒柔伤（TODO：DoT）；层4+ 真气回馈回血
-- （v1：立即柔伤一跳 + 层4 起附带小额回血）。
local M = {}
M.meta = { id = "buf_ng_xmg_001", name = "风幻录", book = "ng_xmg_001",
           category = "attack", max_tier = 5 }
M.levels = {
  [1] = { rou = 38, heal = 0 },
  [2] = { rou = 48, heal = 0 },
  [3] = { rou = 60, heal = 0 },
  [4] = { rou = 74, heal = 30 },
  [5] = { rou = 90, heal = 38 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_attack(ctx)
  local p = lv(ctx)
  local out = { extra_damage = { amount = p.rou, kind = "rou" } }
  if p.heal > 0 then
    out.heal = p.heal
  end
  return out
end

return M
