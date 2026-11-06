-- 太极拳·四式：架势轮换（守/攻两态循环，24h 租约，太极架势槽 7）。
-- exe 内嵌 buf_CS_wd_tjq04_1/2 ConfigID 佐证双变体仍在。
-- 原 VERIFIED_OVERLAYS: Overlay(0, BUFF_SLOT_TAIJI_STANCE=7, false, stance{6013,6055})。
-- 演示 on_hit：攻态（static_data=6055）下本波倍率 +20%（示例开关，置 nil 关闭）。
local M = {}

local STANCE_SLOT = 7          -- PlayerActor.BUFF_SLOT_TAIJI_STANCE
local STANCE_ATTACK = 6055     -- buf_CS_wd_tjq04_2（攻）

function M.compile(def)
  return {
    stance_cycle = {
      variants = {
        { id = "buf_CS_wd_tjq04_1", static_data = 6013, level = 1,
          life_ms = 24 * 60 * 60 * 1000, slot = STANCE_SLOT },
        { id = "buf_CS_wd_tjq04_2", static_data = 6055, level = 1,
          life_ms = 24 * 60 * 60 * 1000, slot = STANCE_SLOT },
      },
    },
  }
end

-- on_hit 运行期钩子演示（默认关闭，保持与原 Java overlay 行为逐字段等价）。
-- 启用示例：攻态（static_data=6055）下该波倍率 +20%：
--   function M.on_hit(ctx)
--     if ctx.caster.stance_static_data == STANCE_ATTACK then
--       return { ratio = ctx.ratio * 1.2 }
--     end
--     return nil
--   end

return M
