-- 断脉逆心功（buf_ng_tm_005）｜唐门｜叠加层数
-- 源：内功特效清单.txt 行527-531（5 层）。
-- 攻击几率种"极柔寒毒"：目标出招/轻功时损失气血 30 秒叠 4-5 层
-- （TODO：目标出招事件/层数 debuff；层5 每层降 3% 招式伤害、刷新封招 TODO）。
-- v1：占位（依赖目标侧事件）。
local M = {}
M.meta = { id = "buf_ng_tm_005", name = "断脉逆心功", book = "ng_tm_005",
           category = "stack", max_tier = 5 }
M.defaults = { proc = 0.20 }
M.levels = {
  [1] = {},
  [2] = {},
  [3] = {},
  [4] = {},
  [5] = {},
}

return M
