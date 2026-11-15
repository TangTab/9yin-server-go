-- 探针：bruise_restore 双形态（裸数值 / {amount} / {percent}）。
local M = {}
M.meta = { id = "probe_bruise", name = "探针·回淤血", book = "ng_probe_bruise",
           category = "victim", max_tier = 5 }
function M.on_victim(ctx)
  return { bruise_restore = 88 }
end
function M.on_parry(ctx)
  return { bruise_restore = { amount = 120 } }
end
function M.on_dodge(ctx)
  return { bruise_restore = { percent = 0.3 } }
end
function M.on_periodic(ctx)
  return { bruise_restore = { percent = 1.5 } }
end
return M
