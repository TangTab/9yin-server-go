-- 探针：受控域三条指令（规格书《击飞击退击倒机制规格书》§7.2）
-- ——control/break_control/control_immune 的白名单与 ctx 控制字段回显。
local M = {}
M.meta = { id = "probe_control", name = "探针·受控", book = "ng_probe_control",
           category = "hit", max_tier = 1 }
function M.on_attack(ctx)
  return { control = { kind = "push", dur_ms = 900, dist_m = 2.5 } }
end
function M.on_victim(ctx)
  return { break_control = true, control_immune = 2500,
           state = { control_kind = ctx.control_kind or "?",
                     control_remain = ctx.control_remain_ms or -2 } }
end
return M
