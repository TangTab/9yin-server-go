-- 上清无极功（buf_ng_wd_003）｜武当｜受击触发
-- 源：内功特效清单.txt 行587-590（4 层）。
-- 金丹渡劫：气血 <30% 受击时 6-5 秒内快速恢复 30% 最大气血（100-200 秒冷却）。
-- TODO：恢复为立即一次回满 30%（持续回复需 HoT 指令）；扶摇步法起跳速度、切换回气。
local M = {}
M.meta = { id = "buf_ng_wd_003", name = "上清无极功", book = "ng_wd_003",
           category = "victim", max_tier = 4 }
M.levels = {
  [1] = { hp_pct = 0.30, cd_ms = 200000 },
  [2] = { hp_pct = 0.30, cd_ms = 150000 },
  [3] = { hp_pct = 0.30, cd_ms = 120000 },
  [4] = { hp_pct = 0.30, cd_ms = 100000 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_victim(ctx)
  if ctx.self.hp >= ctx.self.max_hp * 0.30 then return nil end
  local p = lv(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.cd_until and now < s.cd_until then return nil end
  return {
    heal = math.floor(ctx.self.max_hp * p.hp_pct),
    state = { cd_until = now + p.cd_ms },
  }
end

return M
