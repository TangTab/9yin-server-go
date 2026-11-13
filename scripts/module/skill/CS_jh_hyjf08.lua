-- 寒心恨雪（CS_jh_hyjf08）通道参数｜挑战.pcapng 2026-11-12 官服定版，
-- docs/寒心恨雪与太极剑舞引导机制.md §二：8 波×440ms、首波 +180ms、
-- 逐波扣 10 怒（SP 不足吞波）、受理 SP=0 整体静默吞。
-- m169 挂/摘对与 BufferInfo 槽 CSV 由服务端波次引擎按此驱动
-- （skill_channel.go，帧序方言不下放脚本）；调参改这里，GM lua_reload
-- 热更。宿主不可用时目录编译回退 Go 离线种子（channelSkillSpecs）。
local M = {}

function M.compile(def)
  return {
    channel = {
      waves = 8,               -- 波数上限（官服满跳实证）
      first_wave_ms = 180,     -- 首波延迟（官服 +166~227ms）
      interval_ms = 440,       -- 波间隔（官服 390~443ms）
      sp_per_wave = 10,        -- 每波扣怒（先扣后结算，无预扣）
      victim_buff = "buf_CS_jh_hyjf08",       -- 每波瞬挂瞬摘的受害者 buff
      self_buff = "buf_CS_jh_hyjf08_03",      -- 自身 buff 对
      self_buff_static = 9019, -- BufferInfo 槽 CSV 首段（0x233B 实证）
    },
  }
end

return M
