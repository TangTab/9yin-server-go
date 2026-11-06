-- 修罗武经(会意)（buf_ng_jy_101）｜ng_jy｜be_hit
-- 源：res1 string_buff_desc desc_buf_ng_jy_101_<层>（1 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_jy_101", name = "修罗武经(会意)", book = "ng_jy_101",
           category = "be_hit", max_tier = 1 }
M.levels = {
  [1] = { desc = "修罗武经(会意)喋血凶灵：攻击时有一定几率使自身10秒内提高20%的招式伤害及10%的暴击伤害（结束后进入20秒回气时间）。凶灵护体：被攻击时有一定几率使自身10秒内减免25%的招式伤害（结束后进入25秒回气时间）。修罗临世：喋血凶灵下攻击将为敌方添加5秒的“血祭”状态并在状态结束时承受“血祭”状态下受到伤害总量30%的阳伤害或在凶灵护体下被攻击则为自己添加5层持续5秒的“修罗”状态：被攻击则回复350点气血（受自身恶行值加成，最多回复1000点，回复一次扣除一层“修罗”）（“血祭”状态与“修罗”状态任一效果触发后进入75秒回气时间）", reduce_pct = { 25 }, duration_sec = { 10, 20, 10, 25, 5, 5, 75 }, layers = { 5 } },
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
  local p = lv(ctx)
  -- TODO(二期)：按描述机制返回效果指令（字段见 M.levels 同名参数）
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
