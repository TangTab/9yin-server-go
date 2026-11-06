-- 太极拳·七式：太极七式专用租约槽（不改租约时长、无红霸体）。
-- 原 VERIFIED_OVERLAYS: Overlay(0, BUFF_SLOT_TAIJI_07=9, false, nil)。
local M = {}

function M.compile(def)
  return {
    self_buff_slot = 9,   -- PlayerActor.BUFF_SLOT_TAIJI_07
  }
end

return M
