-- 霄汉玄功（buf_ng_xt_005）｜行字辈｜受击触发
-- 源：内功特效清单.txt 行701-705（5 层）。
-- 气冲霄汉：攻击且气血 <90% 时 8 秒伤害 +10-15%（TODO：临时伤害%），攻击消耗层
--   附加 200-400 外功伤（失血越深越强，最多 +90%；调息 30 秒）。
-- 直落碧霄：受击且 <60% 免伤 10-15%+叠层爆发（TODO：多层状态机）。
-- 碧霄传凤（层5）TODO。v1：气冲霄汉低血附伤实现。
local M = {}
M.meta = { id = "buf_ng_xt_005", name = "霄汉玄功", book = "ng_xt_005",
           category = "victim", max_tier = 5 }
M.defaults = { cd_ms = 30000 }
M.levels = {
  [1] = { base = 200 },
  [2] = { base = 250 },
  [3] = { base = 300 },
  [4] = { base = 350 },
  [5] = { base = 400 },
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
  local now = ctx.now or 0
  if s.cd_until and now < s.cd_until then return nil end
  local hp_pct = ctx.self.max_hp > 0 and ctx.self.hp / ctx.self.max_hp or 1
  if hp_pct >= 0.90 then return nil end
  -- 气血每少 10% 伤害提高 30%，最多 +90%
  local lost = math.min(0.90, math.floor((1 - hp_pct) / 0.10) * 0.10)
  local amount = math.floor(p.base * (1 + lost * 3))
  return { extra_damage = { amount = amount, kind = "external" },
           state = { cd_until = now + M.defaults.cd_ms } }
end

return M
