-- 月洛录（buf_ng_ts_002）｜天山派｜受击触发
-- 源：内功特效清单.txt 行544-547（4 层）。
-- 每受击 6/5 次，5 秒内下次受击回复 145-351 气血（+5% 内力）或反弹同值阴伤
-- （25 秒回气；威力加成 TODO）。v1：走回复分支。
local M = {}
M.meta = { id = "buf_ng_ts_002", name = "月洛录", book = "ng_ts_002",
           category = "victim", max_tier = 4 }
M.levels = {
  [1] = { need = 6, amount = 145 },
  [2] = { need = 6, amount = 207 },
  [3] = { need = 6, amount = 276 },
  [4] = { need = 5, amount = 351 },
}
M.WINDOW_MS = 5000
M.CD_MS = 25000

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.ready and s.ready_until and now < s.ready_until then
    return { heal = p.amount,
             mp_restore = tier_of(ctx) >= 4 and math.floor(ctx.self.max_mp * 0.05) or 0,
             state = { ready = false, cd_until = now + M.CD_MS } }
  end
  if s.cd_until and now < s.cd_until then return nil end
  local count = (s.count or 0) + 1
  if count >= p.need then
    return { state = { count = 0, ready = true, ready_until = now + M.WINDOW_MS } }
  end
  return { state = { count = count } }
end

return M
