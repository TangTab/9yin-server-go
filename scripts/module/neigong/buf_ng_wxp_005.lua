-- 巫咸经（buf_ng_wxp_005）｜ng_wxp｜be_hit
-- 源：res1 string_buff_desc desc_buf_ng_wxp_005_<层>（5 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_wxp_005", name = "巫咸经", book = "ng_wxp_005",
           category = "be_hit", max_tier = 5 }
M.levels = {
  [1] = { desc = "巫咸经龟甲纹：攻击敌人自动浮现\"龟甲纹\"，每2秒累积1层，上限9层，每层提升自己1%暴击伤害和0.4%内外功防御忽视，叠加到9层后急速消失。卜纹：受到攻击有几率为自己添加“卜纹”，每层提高4%暴击伤害减免，最多叠加3层，叠至3层时，下1击受到伤害减免60%，卜纹持续12秒。", reduce_pct = { 60 }, duration_sec = { 2, 12 }, layers = { 1, 9, 9, 3, 3 } },
  [2] = { desc = "巫咸经龟甲纹：攻击敌人自动浮现\"龟甲纹\"，每2秒累积1层，上限9层，每层提升自己2%暴击伤害和0.5%内外功防御忽视，叠加到9层后急速消失。卜纹：受到攻击有几率为自己添加“卜纹”，每层提高5%暴击伤害减免，最多叠加3层，叠至3层时，下1击受到伤害减免70%，卜纹持续12秒。", reduce_pct = { 70 }, duration_sec = { 2, 12 }, layers = { 1, 9, 9, 3, 3 } },
  [3] = { desc = "巫咸经龟甲纹：攻击敌人自动浮现\"龟甲纹\"，每2秒累积1层，上限9层，每层提升自己3%暴击伤害和0.6%内外功防御忽视，叠加到9层后急速消失。卜纹：受到攻击有几率为自己添加“卜纹”，每层提高6%暴击伤害减免，最多叠加3层，叠至3层时，下1击受到伤害减免80%，卜纹持续12秒。", reduce_pct = { 80 }, duration_sec = { 2, 12 }, layers = { 1, 9, 9, 3, 3 } },
  [4] = { desc = "巫咸经龟甲纹：攻击敌人自动浮现\"龟甲纹\"，每2秒累积1层，上限9层，每层提升自己4%暴击伤害和0.7%内外功防御忽视，叠加到9层后急速消失。卜纹：受到攻击有几率为自己添加“卜纹”，每层提高7%暴击伤害减免，最多叠加3层，叠至3层时，下1击受到伤害减免90%，卜纹持续12秒。", reduce_pct = { 90 }, duration_sec = { 2, 12 }, layers = { 1, 9, 9, 3, 3 } },
  [5] = { desc = "巫咸经龟甲纹：攻击敌人自动浮现\"龟甲纹\"，每2秒累积1层，上限9层，每层提升自己5%暴击伤害和0.8%内外功防御忽视，叠加到9层后急速消失。卜纹：受到攻击有几率为自己添加“卜纹”，每层提高8%暴击伤害减免，最多叠加3层，叠至3层时，下1击受到伤害减免100%，卜纹持续12秒。灵觉·审时度势：：获得“龟甲纹”且未达至上限期间暴击敌人有几率开启灵觉，使“龟甲纹”叠加至9层后缓慢消失，延长5秒持续时间（与愈战愈明共享回气）。灵觉·愈战愈明：：获得“卜纹”期间招架敌人有几率开启灵觉，在“卜纹”结束后，接下来8秒内受到伤害减免50%（与审时度势共享回气）", reduce_pct = { 100, 50 }, duration_sec = { 2, 12, 5, 8 }, layers = { 1, 9, 9, 3, 3, 9 } },
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
