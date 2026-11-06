-- 傩面九转（buf_ng_wxp_001）｜ng_wxp｜parry
-- 源：res1 string_buff_desc desc_buf_ng_wxp_001_<层>（4 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_wxp_001", name = "傩面九转", book = "ng_wxp_001",
           category = "parry", max_tier = 4 }
M.levels = {
  [1] = { desc = "傩面九转攻击未招架目标，有几率激发五感潜能，提升8%基础属性（体魄、臂力、内息、身法、罡气）总和的臂力值，持续4秒。", duration_sec = { 4 } },
  [2] = { desc = "傩面九转攻击未招架目标，有几率激发五感潜能，提升10%基础属性（体魄、臂力、内息、身法、罡气）总和的臂力值，持续5秒。", duration_sec = { 5 } },
  [3] = { desc = "傩面九转攻击未招架目标，有几率激发五感潜能，提升12%基础属性（体魄、臂力、内息、身法、罡气）总和的臂力值，持续6秒。", duration_sec = { 6 } },
  [4] = { desc = "傩面九转攻击未招架目标，有几率激发五感潜能，提升15%基础属性（体魄、臂力、内息、身法、罡气）总和的臂力值，持续7秒。", duration_sec = { 7 } },
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
