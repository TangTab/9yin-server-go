-- 地狱换魂经（buf_ng_jy_005）｜君子堂｜受击触发
-- 源：内功特效清单.txt 行315-319（5 层）。
-- 阴魂附体：受击几率 1 秒免疫所有招式伤害（v1：当次 100% 减免近似）；层5：
--   3 秒免疫 + 后 7 秒减 35% 附加伤害（v1：免疫当次 + 7 秒 35% 减免窗口）。
-- 攻击几率提高 12-20% 招式伤害（TODO：临时伤害%指令）。
local M = {}
M.meta = { id = "buf_ng_jy_005", name = "地狱换魂经", book = "ng_jy_005",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.15 }
M.levels = {
  [1] = { after_reduce = 0,    after_ms = 0 },
  [2] = { after_reduce = 0,    after_ms = 0 },
  [3] = { after_reduce = 0,    after_ms = 0 },
  [4] = { after_reduce = 0,    after_ms = 0 },
  [5] = { after_reduce = 0.35, after_ms = 7000 },
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

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.after_until and now < s.after_until then
    return { reduce_pct = p.after_reduce }
  end
  if roll(ctx, M.defaults.proc) then
    local out = { reduce_pct = 1.0 }
    local state = {}
    if p.after_ms > 0 then
      state = { after_until = now + p.after_ms }
    end
    out.state = state
    return out
  end
  return nil
end

return M
