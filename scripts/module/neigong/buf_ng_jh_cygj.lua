-- 五行心法（buf_ng_jh_cygj）｜江湖｜叠加层数
-- 源：内功特效清单.txt 行236-243（8 层）。
-- 攻击降对方罡气/内息/体魄各 1 点并转化自身内息 30 秒，叠 6-57 次
-- （TODO：目标属性削减；v1：转化内息以回蓝近似，层数越高单次越多）。
-- 层5+：50-80% 几率额外叠 1 次；层8：转化内息越高附柔伤越高。
local M = {}
M.meta = { id = "buf_ng_jh_cygj", name = "五行心法", book = "ng_jh_cygj",
           category = "stack", max_tier = 8 }
M.levels = {
  [1] = { max = 6 },
  [2] = { max = 10 },
  [3] = { max = 15 },
  [4] = { max = 21 },
  [5] = { max = 27 },
  [6] = { max = 35 },
  [7] = { max = 45 },
  [8] = { max = 57 },
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
  local s = ctx.state or {}
  local stacks = math.min((s.stacks or 0) + 1, p.max)
  local mp = 1 + math.floor(stacks / 3)
  local out = { mp_restore = mp, state = { stacks = stacks } }
  if tier_of(ctx) >= 8 and stacks >= p.max * 0.6 then
    out.extra_damage = { amount = math.floor(stacks * 1.5), kind = "rou" }
  end
  return out
end

return M
