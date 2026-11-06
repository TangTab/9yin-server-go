-- 逆天邪功（buf_ng_jh_208）｜江湖｜招架触发
-- 源：内功特效清单.txt 行160-167（8 层）。
-- 攻击未被招架获"邪"气：每层 +1.5% 攻击伤害（v1：按伤害百分比附伤近似），叠 8-18 层。
-- 攻击被招架失去一层（NPC 恒不招架，v1 不触发）。逆天邪力 TODO（临时防御+附柔伤）。
local M = {}
M.meta = { id = "buf_ng_jh_208", name = "逆天邪功", book = "ng_jh_208",
           category = "parry", max_tier = 8 }
M.levels = {
  [1] = { max = 8 },
  [2] = { max = 9 },
  [3] = { max = 10 },
  [4] = { max = 11 },
  [5] = { max = 13 },
  [6] = { max = 15 },
  [7] = { max = 15 },
  [8] = { max = 18 },
}
M.PER_STACK = 0.015

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
  local stacks = math.min((s.stacks or 0) + 1, p.max)
  local extra = math.floor(ctx.damage * M.PER_STACK * stacks)
  if extra <= 0 then return nil end
  return { extra_damage = { amount = extra, kind = "external" },
           state = { stacks = stacks } }
end

return M
