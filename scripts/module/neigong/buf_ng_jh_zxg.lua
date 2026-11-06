-- 紫瑞心经（buf_ng_jh_zxg）｜江湖｜常驻/被动
-- 源：内功特效清单.txt 行260-267（8 层）。
-- 攻击造成 0.2-0.85 倍目标所缺失内力的伤害 + 额外阴伤（只对玩家生效；当前战斗管线
-- 无 PvP 结算，NPC 无内力）。v1：占位，待 PvP 接入。
local M = {}
M.meta = { id = "buf_ng_jh_zxg", name = "紫瑞心经", book = "ng_jh_zxg",
           category = "passive", max_tier = 8 }
M.levels = {
  [1] = { missing_mp = 0.20, flat = 0 },
  [2] = { missing_mp = 0.30, flat = 0 },
  [3] = { missing_mp = 0.40, flat = 20 },
  [4] = { missing_mp = 0.55, flat = 50 },
  [5] = { missing_mp = 0.70, flat = 100 },
  [6] = { missing_mp = 0.75, flat = 150 },
  [7] = { missing_mp = 0.80, flat = 500 },
  [8] = { missing_mp = 0.85, flat = 640 },
}

return M
