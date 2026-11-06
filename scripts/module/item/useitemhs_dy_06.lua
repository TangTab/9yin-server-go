-- 飞云剑匣·金（useitemhs_dy_06）
-- 来源：desc.idres——"右击使用可装备剑匣……使用单剑类武器时武器攻击力提升25点"。
-- 限时属性演示：同源替换（重复使用刷新时长），到期 tick 自清；
-- max_hp_add/defense_add 同理可用；consume 默认 true（这里显式写出便于阅读）。
local M = {}
M.meta = { id = "useitemhs_dy_06", name = "飞云剑匣·金" }

M.bonus = 25
M.life_ms = 24 * 60 * 60 * 1000   -- 24 小时

function M.on_use(ctx)
  return {
    temp_stat = { attack_add = M.bonus, defense_add = 0, max_hp_add = 0,
                  life_ms = M.life_ms },
    consume = true,
  }
end

return M
