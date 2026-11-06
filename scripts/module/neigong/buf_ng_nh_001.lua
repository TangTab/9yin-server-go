-- 蛰龙术（buf_ng_nh_001）｜ng_nh｜be_hit
-- 源：res1 string_buff_desc desc_buf_ng_nh_001_<层>（7 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_nh_001", name = "蛰龙术", book = "ng_nh_001",
           category = "be_hit", max_tier = 7 }
M.levels = {
  [1] = { desc = "蛰龙术龟息蛰藏：受到伤害时，有几率6秒内吸收600点伤害(受内功威力加成，触发后进入30秒调息)水脉通玄：造成伤害时，有几率额外附加30%阴伤害(触发后调息5秒)", absorb = { 600 }, duration_sec = { 6, 30, 5 } },
  [2] = { desc = "蛰龙术龟息蛰藏：受到伤害时，有几率6秒内吸收708点伤害(受内功威力加成，触发后进入30秒调息)水脉通玄：造成伤害时，有几率额外附加30%阴伤害(触发后调息5秒)", absorb = { 708 }, duration_sec = { 6, 30, 5 } },
  [3] = { desc = "蛰龙术龟息蛰藏：受到伤害时，有几率6秒内吸收966点伤害(受内功威力加成，触发后进入30秒调息)水脉通玄：造成伤害时，有几率额外附加30%阴伤害(触发后调息5秒)", absorb = { 966 }, duration_sec = { 6, 30, 5 } },
  [4] = { desc = "蛰龙术龟息蛰藏：受到伤害时，有几率6秒内吸收1099点伤害(受内功威力加成，触发后进入30秒调息)水脉通玄：造成伤害时，有几率额外附加30%阴伤害(触发后调息5秒)", absorb = { 1099 }, duration_sec = { 6, 30, 5 } },
  [5] = { desc = "蛰龙术龟息蛰藏：受到伤害时，有几率6秒内吸收1424点伤害(受内功威力加成，触发后进入30秒调息)水脉通玄：造成伤害时，有几率额外附加30%阴伤害(触发后调息5秒)", absorb = { 1424 }, duration_sec = { 6, 30, 5 } },
  [6] = { desc = "蛰龙术龟息蛰藏：受到伤害时，有几率6秒内吸收1952点伤害(受内功威力加成，触发后进入30秒调息)水脉通玄：造成伤害时，有几率额外附加30%阴伤害。(触发后调息5秒)潜渊蛰龙：触发龟息蛰藏的6秒内每次受到招式攻击时有概率获得一层“潜渊蛰龙”状态，水脉通玄触发时可获得一层“潜渊蛰龙”状态。每层提升2%内外功防御及2%招式伤害。最多叠加至5层后即移除。“潜渊蛰龙”消散后进入45秒调息时间", absorb = { 1952 }, duration_sec = { 6, 30, 5, 6, 45 }, layers = { 5 } },
  [7] = { desc = "蛰龙术龟息蛰藏：受到伤害时，有几率6秒内吸收2781点伤害(受内功威力加成，触发后进入30秒调息)水脉通玄：造成伤害时，有几率额外附加30%阴伤害。(触发后调息5秒)潜渊蛰龙：触发龟息蛰藏的6秒内每次受到招式攻击时有概率获得一层“潜渊蛰龙”状态，水脉通玄触发时可获得一层“潜渊蛰龙”状态。每层提升4%内外功防御及4%招式伤害。最多叠加至5层后即移除。“潜渊蛰龙”消散后进入45秒调息时间", absorb = { 2781 }, duration_sec = { 6, 30, 5, 6, 45 }, layers = { 5 } },
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
