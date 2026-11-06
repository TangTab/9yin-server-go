-- 探针：只实现别名钩子 on_hit——验证 on_attack 事件经别名链兜底命中。
local M = {}
M.meta = { id = "probe_alias", name = "探针·别名", book = "ng_probe_alias",
           category = "hit", max_tier = 1 }
function M.on_hit(ctx)
  return { extra_damage = { amount = ctx.damage or 0 } }
end
return M
