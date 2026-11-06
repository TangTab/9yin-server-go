-- 天阳神功（buf_ng_jl_002）｜极乐谷｜攻击触发
-- 源：内功特效清单.txt 行272-275（4 层）。
-- 每次攻击注入真气，15/12/10/10 股后筋脉紊乱：恍惚 2-3.5 秒（TODO：控制指令；
-- v1：满层爆发阳伤近似"紊乱"）。
local M = {}
M.meta = { id = "buf_ng_jl_002", name = "天阳神功", book = "ng_jl_002",
           category = "attack", max_tier = 4 }
M.levels = {
  [1] = { need = 15, burst = 90 },
  [2] = { need = 12, burst = 115 },
  [3] = { need = 10, burst = 145 },
  [4] = { need = 10, burst = 180 },
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
  local count = (s.count or 0) + 1
  if count >= p.need then
    return { extra_damage = { amount = p.burst, kind = "yang" },
             state = { count = 0 } }
  end
  return { state = { count = count } }
end

return M
