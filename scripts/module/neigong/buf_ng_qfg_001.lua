-- 古仙籍（buf_ng_qfg_001）｜ng_qfg｜be_hit
-- 源：res1 string_buff_desc desc_buf_ng_qfg_001_<层>（7 层）。
-- 生成：tools/skill-data/gen_neigong_scripts.py；数值自描述解析，机制钩子待战斗二期逐本细化。
local M = {}
M.meta = { id = "buf_ng_qfg_001", name = "古仙籍", book = "ng_qfg_001",
           category = "be_hit", max_tier = 7 }
M.levels = {
  [1] = { desc = "古仙籍超脱境界：被攻击时，有30%几率提升自身内外功防御8%，持续10秒（叠加层数不刷新持续时间，效果结束后，进入45秒回气时间），叠加3层将恢复204点气血（受内功威力加成，最多400点，防御提升效果持续时间内仅能触发一次回血）。仙人合一：攻击时，有30%几率使10秒内自身的伤害提高10%（45秒冷却）。", duration_sec = { 10, 45, 10, 45 }, layers = { 3 } },
  [2] = { desc = "古仙籍超脱境界：被攻击时，有30%几率提升自身内外功防御9%，持续10秒（叠加层数不刷新持续时间，效果结束后，进入40秒回气时间），叠加3层将恢复345点气血（受内功威力加成，最多675点，防御提升效果持续时间内仅能触发一次回血）。仙人合一：攻击时，有30%几率使11秒内自身的伤害提高12%（40秒冷却）。", duration_sec = { 10, 40, 11, 40 }, layers = { 3 } },
  [3] = { desc = "古仙籍超脱境界：被攻击时，有30%几率提升自身内外功防御10%，持续10秒（叠加层数不刷新持续时间，效果结束后，进入40秒回气时间），叠加3层将恢复563点气血（受内功威力加成，最多1100点，防御提升效果持续时间内仅能触发一次回血）。仙人合一：攻击时，有30%几率使12秒内自身的伤害提高14%（40秒冷却）。", duration_sec = { 10, 40, 12, 40 }, layers = { 3 } },
  [4] = { desc = "古仙籍超脱境界：被攻击时，有30%几率提升自身内外功防御11%，持续10秒（叠加层数不刷新持续时间，效果结束后，进入35秒回气时间），叠加3层将恢复844点气血（受内功威力加成，最多1650点，防御提升效果持续时间内仅能触发一次回血）。仙人合一：攻击时，有30%几率使13秒内自身的伤害提高16%（35秒冷却）。", duration_sec = { 10, 35, 13, 35 }, layers = { 3 } },
  [5] = { desc = "古仙籍超脱境界：被攻击时，有30%几率提升自身内外功防御12%，持续10秒（叠加层数不刷新持续时间，效果结束后，进入35秒回气时间），叠加3层将恢复1105点气血（受内功威力加成，最多2160点，防御提升效果持续时间内仅能触发一次回血）。仙人合一：攻击时，有30%几率使14秒内自身的伤害提高18%（35秒冷却）。", duration_sec = { 10, 35, 14, 35 }, layers = { 3 } },
  [6] = { desc = "古仙籍超脱境界：被攻击时，有30%几率提升自身内外功防御13%，持续10秒（叠加层数不刷新持续时间，效果结束后，进入30秒回气时间），叠加3层将恢复1331点气血（受内功威力加成，最多2600点，防御提升效果持续时间内仅能触发一次回血）。仙人合一：攻击时，有30%几率使15秒内自身的伤害提高20%、内外功命中率提升15%（30秒冷却）。", duration_sec = { 10, 30, 15, 30 }, layers = { 3 } },
  [7] = { desc = "古仙籍超脱境界：被攻击时，有30%几率提升自身内外功防御14%，持续10秒（叠加层数不刷新持续时间，效果结束后，进入30秒回气时间），叠加3层将恢复1792点气血（受内功威力加成，最多3500点，防御提升效果持续时间内仅能触发一次回血）。仙人合一：攻击时，有30%几率使15秒内自身的伤害提高22%、内外功命中率提升30%（30秒冷却）。", duration_sec = { 10, 30, 15, 30 }, layers = { 3 } },
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
