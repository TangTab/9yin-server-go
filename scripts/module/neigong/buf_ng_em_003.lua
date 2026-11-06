-- 飞弦羽经（buf_ng_em_003）｜峨眉｜攻击触发
-- 源：内功特效清单.txt 行55-58（4 层）。
-- 舒活经脉：每次出招攻击 5 秒内恢复 1-2% 最大气血（v1：单次立即回复近似；HoT TODO）。
-- 凌空换气：穿云纵连招 TODO。层4：几率双倍回复 + 内力上限 +200（罡气 +20 TODO）。
local M = {}
M.meta = { id = "buf_ng_em_003", name = "飞弦羽经", book = "ng_em_003",
           category = "attack", max_tier = 4 }
M.defaults = { double_proc = 0.15 }
M.levels = {
  [1] = { hp_pct = 0.010 },
  [2] = { hp_pct = 0.015 },
  [3] = { hp_pct = 0.020 },
  [4] = { hp_pct = 0.020 },
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
  local p = lv(ctx)
  local mult = 1
  if tier_of(ctx) >= 4 and roll(ctx, M.defaults.double_proc) then
    mult = 2
  end
  return { heal = math.floor(ctx.self.max_hp * p.hp_pct * mult) }
end

function M.passive(ctx)
  if tier_of(ctx) >= 4 then
    return { max_mp_add = 200 }
  end
  return nil
end

return M
