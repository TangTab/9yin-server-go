-- 溪月花香集(会意)（buf_ng_jz_101）｜ng_jz｜parry
-- 源：res1 string_buff_desc desc_buf_ng_jz_101_<层>（1 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_jz_101", name = "溪月花香集(会意)", book = "ng_jz_101",
           category = "parry", max_tier = 1 }
M.levels = {
  [1] = { desc = "溪月花香集(会意)寒月惊魂：攻击时获得“寒月惊魂”状态，招式伤害提高20%，同时获得三层“寒月”状态：每层使招式伤害提高10%，忽视招架减免提高10%，每0.5秒消除一层。（“寒月惊魂”状态持续5秒，消散后进入回气时间15秒）暗香浸骨：被攻击时获得“暗香浸骨”状态，受到的招式伤害降低20%，同时获得三层“暗香”状态：每层受到的招式伤害降低10%，招架减免提升提高10%，每0.5秒消除一层。（“暗香浸骨”状态持续5秒，消散后进入回气时间20秒）", duration_sec = { 0.5, 5, 15, 0.5, 5, 20 } },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.on_parry(ctx)
  local p = lv(ctx)
  -- TODO(二期)：按描述机制返回效果指令（字段见 M.levels 同名参数）
  return nil
end

function M.on_be_hit(ctx)
  return nil
end

function M.on_hit(ctx)
  return nil
end

function M.on_periodic(ctx)
  return nil
end

function M.passive(ctx)
  local p = lv(ctx)
  -- TODO(二期)：内力上限/罡气等被动加成接线
  return nil
end

return M
