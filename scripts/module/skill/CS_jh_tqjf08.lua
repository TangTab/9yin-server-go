-- CS_jh_tqjf08 通道参数｜按 zhaoshi 逐位同构播种（单段 17 帧
-- wj_qzj_07_1 + hitbreak@15 + MovieScene，UseLimit=166 族；hyjf08 为
-- 挑战.pcapng 抓包实证，本技未独立抓包——数值差异仅 self_buff_static）。
-- 机制同 docs/寒心恨雪与太极剑舞引导机制.md §二；调参改这里，GM
-- lua_reload 热更；宿主不可用时回退 Go 离线种子（channelSkillSpecs）。
local M = {}

function M.compile(def)
  return {
    channel = {
      waves = 8,
      first_wave_ms = 180,
      interval_ms = 440,
      sp_per_wave = 10,
      victim_buff = "buf_CS_jh_tqjf08",
      self_buff = "buf_CS_jh_tqjf08_03",
      self_buff_static = 32607,
    },
  }
end

return M
