-- 祛风散（yaopin_sc_1001）
-- 来源：stringname/desc.idres——"可以治疗一重风类疾病；由药师配置而成"。
-- 疾病系统服务端未实装前，先落回血效果（对齐药品分度：一重 = 轻伤小回）。
-- INI 回退层（item_use_skill_static 2101 → yaopinhp1 L1 = 回 100）已覆盖同类道具，
-- 此脚本作为 scripts/item 域的锚点样例： heal_hp 直接给值、默认消耗。
local M = {}
M.meta = { id = "yaopin_sc_1001", name = "祛风散" }

function M.on_use(ctx)
  return {
    heal_hp = 100,
    -- 疾病系统实装后在此按 ctx.state 判断风类疾病层数再决定疗效
  }
end

return M
