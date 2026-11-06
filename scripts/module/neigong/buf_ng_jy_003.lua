-- 七杀心经（buf_ng_jy_003）｜君子堂｜攻击触发
-- 源：内功特效清单.txt 行306-309（4 层）。
-- 肃杀一击：攻击几率积蓄杀气（臂力 +9-18 TODO），七次后释放：造成臂力等额外功伤，
-- 气血低于 20-30% 的敌人额外受其损失生命 8% 功伤（人马如一 TODO）。
local M = {}
M.meta = { id = "buf_ng_jy_003", name = "七杀心经", book = "ng_jy_003",
           category = "attack", max_tier = 4 }
M.defaults = { proc = 0.30 }
M.levels = {
  [1] = { threshold = 0.20 },
  [2] = { threshold = 0.20 },
  [3] = { threshold = 0.20 },
  [4] = { threshold = 0.30 },
}
M.RELEASE_NEED = 7

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
  local s = ctx.state or {}
  if not roll(ctx, M.defaults.proc) then
    return s.stacks and s.stacks > 0 and { state = s } or nil
  end
  local stacks = (s.stacks or 0) + 1
  if stacks >= M.RELEASE_NEED then
    local amount = ctx.self.attack or 150
    local hp_pct = ctx.target.max_hp > 0 and ctx.target.hp / ctx.target.max_hp or 1
    if hp_pct < p.threshold then
      amount = amount + math.floor((ctx.target.max_hp - ctx.target.hp) * 0.08)
    end
    return { extra_damage = { amount = amount, kind = "external" },
             state = { stacks = 0 } }
  end
  return { state = { stacks = stacks } }
end

return M
