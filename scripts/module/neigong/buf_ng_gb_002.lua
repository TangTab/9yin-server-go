-- 玉阳神功（buf_ng_gb_002）｜丐帮｜常驻/被动
-- 源：内功特效清单.txt 行83-86（4 层）。
-- 攻击吸收精气：叠 9/7/6/5 层后下次攻击散去，震倒目标并造成臂力数值的外功伤害
-- （v1：面板攻击近似臂力；震倒 TODO）。层4：招架几率获得精气 TODO（NPC 不招架）。
local M = {}
M.meta = { id = "buf_ng_gb_002", name = "玉阳神功", book = "ng_gb_002",
           category = "passive", max_tier = 4 }
M.levels = {
  [1] = { need = 9 },
  [2] = { need = 7 },
  [3] = { need = 6 },
  [4] = { need = 5 },
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
  local s = ctx.state or {}
  local stacks = (s.stacks or 0) + 1
  if stacks >= p.need then
    return { extra_damage = { amount = ctx.self.attack or 150, kind = "external" },
             state = { stacks = 0 } }
  end
  return { state = { stacks = stacks } }
end

return M
