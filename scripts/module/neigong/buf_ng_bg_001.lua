-- 大巧若拙经（buf_ng_bg_001）｜ng_bg｜hit
-- 源：res1 string_buff_desc desc_buf_ng_bg_001_<层>（5 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_bg_001", name = "大巧若拙经", book = "ng_bg_001",
           category = "hit", max_tier = 5 }
M.levels = {
  [1] = { desc = "大巧若拙经守拙：攻击目标有几率让自己获得1层“拙”（最多可叠加6层），之后每使用招式消耗5%内力便可根据当前的“拙”层数获得相应层数的“刃”（每层“刃”提升3%伤害，最多18%，持续6秒，效果结束进入短暂回气，但“拙”不清除），直至叠加6层“拙”后，进入调息。", damage_up_pct = { 3 }, duration_sec = { 6 }, layers = { 1, 6, 6 } },
  [2] = { desc = "大巧若拙经守拙：攻击目标有几率让自己获得1层“拙”（最多可叠加6层），之后每使用招式消耗5%内力便可根据当前的“拙”层数获得相应层数的“刃”（每层“刃”提升3.5%伤害，最多21%，持续6秒，效果结束进入短暂回气，但“拙”不清除），直至叠加6层“拙”后，进入调息。", damage_up_pct = { 3.5 }, duration_sec = { 6 }, layers = { 1, 6, 6 } },
  [3] = { desc = "大巧若拙经守拙：攻击目标有几率让自己获得1层“拙”（最多可叠加6层），之后每使用招式消耗5%内力便可根据当前的“拙”层数获得相应层数的“刃”（每层“刃”提升4%伤害，最多24%，持续6秒，效果结束进入短暂回气，但“拙”不清除），直至叠加6层“拙”后，进入调息。 ", damage_up_pct = { 4 }, duration_sec = { 6 }, layers = { 1, 6, 6 } },
  [4] = { desc = "大巧若拙经守拙：攻击目标有几率让自己获得1层“拙”（最多可叠加6层），之后每使用招式消耗5%内力便可根据当前的“拙”层数获得相应层数的“刃”（每层“刃”提升4.5%伤害，最多27%，持续6秒，效果结束进入短暂回气，但“拙”不清除），直至叠加6层“拙”后，进入调息。藏锋：当自身气血低于70%，受到攻击有几率获得“巧”（气血值越低，获得几率越高），可令每层“拙”提高自己2%伤害减免，同时“拙”期间，每次攻击造成伤害的2%将转化为自身气血。", damage_up_pct = { 4.5 }, drain_heal_pct = { 2 }, duration_sec = { 6 }, layers = { 1, 6, 6 } },
  [5] = { desc = "大巧若拙经守拙：攻击目标有几率让自己获得1层“拙”（最多可叠加6层），之后每使用招式消耗5%内力便可根据当前的“拙”层数获得相应层数的“刃”（每层“刃”提升5%伤害，最多30%，持续6秒，效果结束进入短暂回气，但“拙”不清除），直至叠加6层“拙”后，进入调息。藏锋：当自身气血低于70%，受到攻击有几率获得“巧”（气血值越低，获得几率越高），可令每层“拙”提高自己3%伤害减免，同时“拙”期间，每次攻击造成伤害的3%将转化为自身气血。", damage_up_pct = { 5 }, drain_heal_pct = { 3 }, duration_sec = { 6 }, layers = { 1, 6, 6 } },
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
  return nil
end

function M.on_be_hit(ctx)
  return nil
end

function M.on_hit(ctx)
  local p = lv(ctx)
  -- TODO(二期)：按描述机制返回效果指令（字段见 M.levels 同名参数）
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
