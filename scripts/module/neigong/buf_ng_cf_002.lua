-- 凌云心经（buf_ng_cf_002）｜长风镖局｜受击触发
-- 源：内功特效清单.txt 行28-34（7 层）。
-- 云接：受击几率接下来 7 次受伤吸收 12-36%（v1：吸收窗口内每击减免）。
-- 云引：吸收时积蓄劲力，攻击附带全部劲力造成刚伤（近身威力越高越强 TODO：威力加成）。
local M = {}
M.meta = { id = "buf_ng_cf_002", name = "凌云心经", book = "ng_cf_002",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.25 }
M.levels = {
  [1] = { absorb = 0.12 },
  [2] = { absorb = 0.15 },
  [3] = { absorb = 0.18 },
  [4] = { absorb = 0.21 },
  [5] = { absorb = 0.25 },
  [6] = { absorb = 0.30 },
  [7] = { absorb = 0.36 },
}
M.CHARGES = 7

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end
local function roll(ctx, p) return (ctx.rand and ctx.rand.r_proc or 1) < p end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  if s.charges and s.charges > 0 then
    local saved = math.floor(ctx.incoming * p.absorb)
    return { reduce_pct = p.absorb,
             state = { charges = s.charges - 1, jinli = (s.jinli or 0) + math.floor(saved / 2) } }
  end
  if roll(ctx, M.defaults.proc) then
    return { reduce_pct = p.absorb,
             state = { charges = M.CHARGES - 1,
                       jinli = (s.jinli or 0) + math.floor(ctx.incoming * p.absorb / 2) } }
  end
  return nil
end

function M.on_attack(ctx)
  local s = ctx.state or {}
  local jinli = s.jinli or 0
  if jinli <= 0 then return nil end
  return { extra_damage = { amount = jinli, kind = "gang" },
           state = { charges = s.charges, jinli = 0 } }
end

return M
