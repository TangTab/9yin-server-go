-- 雷音神典（buf_ng_ts_006）｜天山派｜受击触发
-- 源：内功特效清单.txt 行562-566（5 层）。
-- 雷声隐隐：攻击几率震伤：8 秒所受伤害 8 秒后爆发 15-35%（TODO：延迟结算）。
-- 神音谐谐：受击几率 6 秒所受伤害转化 30% 为气血（v1：窗口内直接按 30% 回血近似）。
local M = {}
M.meta = { id = "buf_ng_ts_006", name = "雷音神典", book = "ng_ts_006",
           category = "victim", max_tier = 5 }
M.defaults = { proc = 0.20, convert = 0.30 }
M.levels = {
  [1] = { burst = 0.15 },
  [2] = { burst = 0.20 },
  [3] = { burst = 0.25 },
  [4] = { burst = 0.30 },
  [5] = { burst = 0.35 },
}
M.WINDOW_MS = 6000

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
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then
    return { heal = math.floor(ctx.incoming * M.defaults.convert),
             state = { until_at = s.until_at } }
  end
  if roll(ctx, M.defaults.proc) then
    return { heal = math.floor(ctx.incoming * M.defaults.convert),
             state = { until_at = now + M.WINDOW_MS } }
  end
  return nil
end

return M
