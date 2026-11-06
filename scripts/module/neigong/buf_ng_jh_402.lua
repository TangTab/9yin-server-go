-- 天魔秘（buf_ng_jh_402）｜ng_jh｜periodic
-- 源：res1 string_buff_desc desc_buf_ng_jh_402_<层>（1 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_jh_402", name = "天魔秘", book = "ng_jh_402",
           category = "periodic", max_tier = 1 }
M.levels = {
  [1] = { desc = "天魔秘每隔30秒，下次出招可使周围空间气场的塌陷，牵扯周围目标靠近自己，并造成大量刚伤害。天魔真气：使牵扯的目标在一段时间内神智不清，按键错乱,持续3秒。", duration_sec = { 30, 3 } },
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
  return nil
end

function M.on_periodic(ctx)
  local p = lv(ctx)
  -- TODO(二期)：按描述机制返回效果指令（字段见 M.levels 同名参数）
  return nil
end

function M.passive(ctx)
  local p = lv(ctx)
  -- TODO(二期)：内力上限/罡气等被动加成接线
  return nil
end

return M
