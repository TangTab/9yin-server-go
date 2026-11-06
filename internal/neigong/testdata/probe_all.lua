-- 探针：返回全部白名单指令，并把触发事件写进 state。
local M = {}
M.meta = { id = "probe_all", name = "探针·全指令", book = "ng_probe_all",
           category = "hit", max_tier = 5 }
function M.on_attack(ctx)
  return {
    extra_damage = { amount = 77, kind = "yang" },
    reduce_pct = 0.25,
    shield = { amount = 200, life_ms = 30000 },
    heal = 60,
    mp_restore = 30,
    reflect = { amount = 45, kind = "yin" },
    parry_reduce = { internal = 0.1, external = 0.2 },
    state = { until_at = (ctx.now or 0) + 6000, seen = "on_attack" },
  }
end
return M
