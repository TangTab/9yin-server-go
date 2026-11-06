-- 太阴玄鉴（buf_ng_wg_001）｜无冠门｜受击触发
-- 源：内功特效清单.txt 行611-617（7 层）。
-- 血穹苍：攻击每击吸血 100-309（累计 6-9 次后 25 秒冷却；层6+ 附 20% 额外伤害）。
-- 玫霞荡：招架时受击回内力 18-56（累计同上）。层5+ 无破天晶（清冷却）TODO。
local M = {}
M.meta = { id = "buf_ng_wg_001", name = "太阴玄鉴", book = "ng_wg_001",
           category = "victim", max_tier = 7 }
M.levels = {
  [1] = { drain = 100, mp = 18, need = 6 },
  [2] = { drain = 118, mp = 22, need = 6 },
  [3] = { drain = 138, mp = 27, need = 7 },
  [4] = { drain = 157, mp = 30, need = 7 },
  [5] = { drain = 178, mp = 36, need = 8 },
  [6] = { drain = 244, mp = 46, need = 8 },
  [7] = { drain = 309, mp = 56, need = 9 },
}
M.CD_MS = 25000

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
  if s.drain_cd_until and now < s.drain_cd_until then return nil end
  local count = (s.drain_count or 0) + 1
  local heal = p.drain
  if tier_of(ctx) >= 6 then
    heal = heal + math.floor(ctx.damage * 0.20)
  end
  if count >= p.need then
    return { heal = heal, state = { drain_count = 0, drain_cd_until = now + M.CD_MS,
                                    mp_count = s.mp_count } }
  end
  return { heal = heal, state = { drain_count = count, mp_count = s.mp_count } }
end

function M.on_parry(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.mp_cd_until and now < s.mp_cd_until then return nil end
  local count = (s.mp_count or 0) + 1
  if count >= p.need then
    return { mp_restore = p.mp, state = { mp_count = 0, mp_cd_until = now + M.CD_MS,
                                          drain_count = s.drain_count } }
  end
  return { mp_restore = p.mp, state = { mp_count = count, drain_count = s.drain_count } }
end

return M
