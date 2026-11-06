-- 花神七式·四式（移花接玉）：自 Buff 租约 35s + 花神租约槽。
-- 原 SkillCatalog.VERIFIED_OVERLAYS 逐字段迁移（Overlay(35_000, BUFF_SLOT_YIHUA=5, false, nil)）。
-- compile(def)：def 为 INI 编译快照（def.exact_buff 即本技能的 buf_CS_yhwq_hsqs04），
-- 返回增量覆盖指令；不返回 effects = 沿用 INI 默认编译效果。
local M = {}

function M.compile(def)
  return {
    self_buff_lease_ms = 35000,
    self_buff_slot = 5,   -- PlayerActor.BUFF_SLOT_YIHUA
  }
end

return M
