-- 青壁何天诀（buf_ng_qy_001）｜青云堡｜攻击触发
-- 源：内功特效清单.txt 行426-432（7 层）。
-- 雁行功返轻功值/青衣身法（内外功伤害 +10-30%）/摄心魄（暴击减调息+下一击附伤）/
--   青衣·影（隐匿）——全部依赖轻功与临时伤害%指令（TODO）。v1：占位。
local M = {}
M.meta = { id = "buf_ng_qy_001", name = "青壁何天诀", book = "ng_qy_001",
           category = "attack", max_tier = 7 }
M.levels = {
  [1] = { qg = 1, bonus = 0.10 },
  [2] = { qg = 2, bonus = 0.12 },
  [3] = { qg = 2, bonus = 0.15 },
  [4] = { qg = 3, bonus = 0.18 },
  [5] = { qg = 3, bonus = 0.22 },
  [6] = { qg = 3, bonus = 0.26 },
  [7] = { qg = 4, bonus = 0.30 },
}

return M
