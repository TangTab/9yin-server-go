-- 移天焚海诀（buf_ng_mj_006）｜明教｜受击触发
-- 源：内功特效清单.txt 行407-411（5 层）。
-- 覆手焚海：攻击几率 N 秒每击附 115-299 阳伤（最多 4-8 次；淤血比对加成 TODO）。
-- 翻手移天：受击几率 5 秒吸收淤血值 50-100% 伤害（淤血层未接，v1：固定护盾近似）。
-- 层5 移天焚海：结束时回淤血 TODO。
local M = {}
M.meta = { id = "buf_ng_mj_006", name = "移天焚海诀", book = "ng_mj_006",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20, cd_ms = 20000, victim_cd_ms = 40000 }
M.levels = {
  [1] = { yang = 115, dur_ms = 5000, hits = 4, shield = 150 },
  [2] = { yang = 139, dur_ms = 6000, hits = 5, shield = 180 },
  [3] = { yang = 164, dur_ms = 7000, hits = 6, shield = 210 },
  [4] = { yang = 215, dur_ms = 8000, hits = 7, shield = 240 },
  [5] = { yang = 299, dur_ms = 9000, hits = 8, shield = 280 },
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
  local now = ctx.now or 0
  if s.hits and s.hits > 0 and s.until_at and now < s.until_at then
    return { extra_damage = { amount = p.yang, kind = "yang" },
             state = { hits = s.hits - 1, until_at = s.until_at } }
  end
  if s.atk_cd_until and now < s.atk_cd_until then return nil end
  if roll(ctx, M.defaults.proc) then
    return { extra_damage = { amount = p.yang, kind = "yang" },
             state = { hits = p.hits - 1, until_at = now + p.dur_ms,
                       atk_cd_until = now + M.defaults.cd_ms } }
  end
  return nil
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.vic_cd_until and now < s.vic_cd_until then return nil end
  if roll(ctx, M.defaults.proc) then
    return { shield = { amount = p.shield, life_ms = 5000 },
             state = { vic_cd_until = now + M.defaults.victim_cd_ms } }
  end
  return nil
end

return M
