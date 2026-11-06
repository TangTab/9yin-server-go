-- 探针：state 窗口——首次触发开窗，窗口内叠层，窗口外重开。
local M = {}
M.meta = { id = "probe_state", name = "探针·state", book = "ng_probe_st",
           category = "hit", max_tier = 1 }
function M.on_attack(ctx)
  local s = ctx.state or {}
  local now = ctx.now or 0
  if s.until_at and now < s.until_at then
    return { heal = 5, state = { until_at = s.until_at, count = (s.count or 0) + 1 } }
  end
  return { heal = 1, state = { until_at = now + 10000, count = 1 } }
end
return M
