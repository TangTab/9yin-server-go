-- 惊鸿礼盒（box_chinajoy_2026_giftbox）
-- 来源：share/item/giftbox.ini——script=BoxItem、DropID=comb_ai_drop_box_chinajoy_2026_giftbox。
-- open_box 指令演示：Java 侧按 drops_mvp.json 权重抽取（随机数 Java 预掷，脚本不掷骰）。
-- 注：无此脚本时 INI 回退层同样按 giftbox.ini 开包——脚本存在则可覆盖次数/叠加额外效果。
local M = {}
M.meta = { id = "box_chinajoy_2026_giftbox", name = "惊鸿礼盒" }

M.drop_id = "comb_ai_drop_box_chinajoy_2026_giftbox"

function M.on_use(ctx)
  return {
    open_box = { drop_id = M.drop_id, rolls = 1 },
  }
end

return M
