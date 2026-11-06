-- 太极拳·五式：太极五式专用租约槽（不改租约时长、无红霸体）。
-- 原 VERIFIED_OVERLAYS: Overlay(0, BUFF_SLOT_TAIJI_05=8, false, nil)。
local M = {}

function M.compile(def)
  return {
    self_buff_slot = 8,   -- PlayerActor.BUFF_SLOT_TAIJI_05
  }
end

return M
