-- 蚕蜕功（buf_ng_wxp_002）｜ng_wxp｜passive
-- 源：res1 string_buff_desc desc_buf_ng_wxp_002_<层>（4 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_wxp_002", name = "蚕蜕功", book = "ng_wxp_002",
           category = "passive", max_tier = 4 }
M.levels = {
  [1] = { desc = "蚕蜕功攻击敌人有几率弹射蛊丝，对目标造成一次性伤害且移动速度缓慢下降直至定身，持续12秒。（定身结束，蛊丝消失）", duration_sec = { 12 } },
  [2] = { desc = "蚕蜕功攻击敌人有几率弹射蛊丝，对目标造成一次性伤害且移动速度缓慢下降直至定身，持续12秒。（定身结束，蛊丝消失）", duration_sec = { 12 } },
  [3] = { desc = "蚕蜕功攻击敌人有几率弹射蛊丝，对目标造成一次性伤害且移动速度缓慢下降直至定身，持续12秒。（定身结束，蛊丝消失）", duration_sec = { 12 } },
  [4] = { desc = "蚕蜕功攻击敌人有几率弹射蛊丝，对目标造成一次性伤害且移动速度急剧下降直至定身，持续12秒。（定身结束，蛊丝不消失。蛊丝缠敌期间敌人轻功值消耗提高8点，同时可操控蛊毒使自己招式伤害提高30%）", duration_sec = { 12 } },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end

function M.passive(ctx)
  local p = lv(ctx)
  -- TODO(二期)：内力上限/罡气等被动加成接线
  return nil
end

return M
