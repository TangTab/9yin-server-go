-- 明教会意功（buf_ng_mj_101）｜明教（会意）｜受击触发
-- 源：内功特效清单.txt 行412-413（档 1/5，中间档按最高档近似）。
-- 覆手焚海：攻击几率 9 秒每击附 299 阳伤最多 8 次（地面加成 TODO）。
-- 翻手移天：受击几率 5 秒吸收最大血量 15% 伤害（25 秒回气）。移天焚海联动 TODO。
local M = {}
M.meta = { id = "buf_ng_mj_101", name = "明教会意功", book = "ng_mj_101",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20, atk_cd_ms = 20000, vic_cd_ms = 25000 }
M.levels = {
  [1] = { yang = 299, dur_ms = 9000, hits = 8, shield_pct = 0.15 },
  [5] = { yang = 299, dur_ms = 9000, hits = 8, shield_pct = 0.15 },
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
                       atk_cd_until = now + M.defaults.atk_cd_ms } }
  end
  return nil
end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.vic_cd_until and now < s.vic_cd_until then return nil end
  if roll(ctx, M.defaults.proc) then
    return { shield = { amount = math.floor(ctx.self.max_hp * p.shield_pct), life_ms = 5000 },
             state = { vic_cd_until = now + M.defaults.vic_cd_ms } }
  end
  return nil
end

return M
