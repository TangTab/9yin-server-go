-- 金刚降魔心诀（buf_ng_sl_003）｜少林派｜攻击触发
-- 源：内功特效清单.txt 行462-465（4 层）。
-- 诸相非相：攻击几率顿悟四相之一（TODO：四相效果系统）。层4：受击几率移除昏迷封招
--   并转移（TODO：净化/转移指令）。v1：占位。
local M = {}
M.meta = { id = "buf_ng_sl_003", name = "金刚降魔心诀", book = "ng_sl_003",
           category = "attack", max_tier = 4 }
M.defaults = { proc = 0.15 }
M.levels = {
  [1] = {},
  [2] = {},
  [3] = {},
  [4] = {},
}

return M
