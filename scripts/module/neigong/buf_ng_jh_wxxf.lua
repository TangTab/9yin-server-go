-- 残阳功诀（buf_ng_jh_wxxf）｜江湖｜攻击触发
-- 源：内功特效清单.txt 行252-259（8 层）。
-- 出招聚集真气获 1-4 秒霸体（免疫控制），27-14 秒 CD（TODO：霸体/免疫控制指令；
-- 层5 起受击也触发、层6+ 30% 打断、层8 低血净化均 TODO）。
-- v1：占位（无可用指令），保留层数参数。
local M = {}
M.meta = { id = "buf_ng_jh_wxxf", name = "残阳功诀", book = "ng_jh_wxxf",
           category = "attack", max_tier = 8 }
M.levels = {
  [1] = { duration_ms = 1000, cd_ms = 27000 },
  [2] = { duration_ms = 1000, cd_ms = 24000 },
  [3] = { duration_ms = 1000, cd_ms = 21000 },
  [4] = { duration_ms = 2000, cd_ms = 18000 },
  [5] = { duration_ms = 2000, cd_ms = 14000 },
  [6] = { duration_ms = 3000, cd_ms = 14000 },
  [7] = { duration_ms = 4000, cd_ms = 14000 },
  [8] = { duration_ms = 4000, cd_ms = 14000 },
}

return M
