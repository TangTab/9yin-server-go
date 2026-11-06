-- 争流心经（buf_ng_xt_002）｜行字辈｜招架触发
-- 源：内功特效清单.txt 行688-691（4 层）。
-- "虚实变幻"层随出招类型生效（实招伤害/招架虚弱/怒招返怒/层4 虚招定身）——
-- 依赖出招类型快照与临时属性（TODO）。v1：占位。
local M = {}
M.meta = { id = "buf_ng_xt_002", name = "争流心经", book = "ng_xt_002",
           category = "parry", max_tier = 4 }
M.levels = {
  [1] = { real = 0.15, weak = 0.30 },
  [2] = { real = 0.20, weak = 0.40 },
  [3] = { real = 0.25, weak = 0.40 },
  [4] = { real = 0.30, weak = 0.50 },
}

return M
