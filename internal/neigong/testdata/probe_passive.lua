-- 探针：passive 扁平数值属性表（含整型与浮点值）。
local M = {}
M.meta = { id = "probe_passive", name = "探针·被动", book = "ng_probe_pv",
           category = "passive", max_tier = 1 }
function M.passive(ctx)
  return { attack_add = 25, max_mp_add = 100.5, damage_reduction_add = 20 }
end
return M
