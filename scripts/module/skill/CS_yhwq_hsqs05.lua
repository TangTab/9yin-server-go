-- 花神七式·五式：花神五式专用租约槽 + 红霸体（起手霸体帧，租约 = 动作总时长）。
-- 原 VERIFIED_OVERLAYS: Overlay(0, BUFF_SLOT_YIHUA_05=6, true, nil)。
local M = {}

function M.compile(def)
  return {
    self_buff_slot = 6,   -- PlayerActor.BUFF_SLOT_YIHUA_05（不改租约时长）
    red_armor = true,
  }
end

return M
