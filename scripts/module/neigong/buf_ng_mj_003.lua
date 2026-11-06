-- 炽日心经（buf_ng_mj_003）｜明教｜常驻/被动
-- 源：内功特效清单.txt 行393-396（4 层）。
-- 炽日光耀：怒气招式后伤害 +12-30% 6-12 秒（TODO：怒招判定与临时伤害%）。
-- 炽热：怒气消耗 -20%（TODO：消耗修正）。v1：占位（无可用指令）。
local M = {}
M.meta = { id = "buf_ng_mj_003", name = "炽日心经", book = "ng_mj_003",
           category = "passive", max_tier = 4 }
M.levels = {
  [1] = { bonus = 0.12, dur_ms = 6000 },
  [2] = { bonus = 0.18, dur_ms = 8000 },
  [3] = { bonus = 0.25, dur_ms = 10000 },
  [4] = { bonus = 0.30, dur_ms = 12000 },
}

return M
