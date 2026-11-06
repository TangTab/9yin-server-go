-- 尸衣经（buf_ng_wxp_004）｜ng_wxp｜be_hit
-- 源：res1 string_buff_desc desc_buf_ng_wxp_004_<层>（5 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_wxp_004", name = "尸衣经", book = "ng_wxp_004",
           category = "be_hit", max_tier = 5 }
M.levels = {
  [1] = { desc = "尸衣经被攻击有几率为自身添加“尸衣”；尸衣：受到的伤害降低20%且会对攻击自身的敌人染上“尸药”（最多影响5个目标），持续10秒，且第一个染上的敌人受到影响会令其下次攻击被封招1秒。尸药：每次被攻击会受到10%额外伤害，持续10秒。", duration_sec = { 10, 1, 10 } },
  [2] = { desc = "尸衣经被攻击有几率为自身添加“尸衣”；尸衣：受到的伤害降低22%且会对攻击自身的敌人染上“尸药”（最多影响5个目标），持续10秒，且第一个染上的敌人受到影响会令其下次攻击被封招1.5秒。尸药：每次被攻击会受到15%额外伤害，持续10秒。", duration_sec = { 10, 1.5, 10 } },
  [3] = { desc = "尸衣经被攻击有几率为自身添加“尸衣”；尸衣：受到的伤害降低25%且会对攻击自身的敌人染上“尸药”（最多影响5个目标），持续11秒，且第一个染上的敌人受到影响会令其下次攻击被封招2秒。尸药：每次被攻击会受到20%额外伤害，持续11秒。", duration_sec = { 11, 2, 11 } },
  [4] = { desc = "尸衣经被攻击有几率为自身添加“尸衣”；尸衣：受到的伤害降低27%且会对攻击自身的敌人染上“尸药”（最多影响5个目标），持续11秒，且第一个染上的敌人受到影响会令其下次攻击被封招2.5秒。尸药：每次被攻击会受到25%额外伤害，持续11秒。", duration_sec = { 11, 2.5, 11 } },
  [5] = { desc = "尸衣经被攻击有几率为自身添加“尸衣”；尸衣：受到的伤害降低30%且会对攻击自身的敌人染上“尸药”（最多影响5个目标），持续12秒，且第一个染上的敌人受到影响会令其下次攻击被封招3秒。尸药：每次被攻击会受到30%额外伤害，持续12秒。尸心疯：回气期间攻击未招架敌人有几率触发短暂“尸药”，使其每次被攻击受到100%额外伤害，持续3秒（仅1次）。", duration_sec = { 12, 3, 12, 3 }, times = { 1 } },
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
