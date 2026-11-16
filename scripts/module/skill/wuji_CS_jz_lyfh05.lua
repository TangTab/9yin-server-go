-- 落英飞花·落尘（wuji_CS_jz_lyfh05）武技冲锋扩程｜v29 2026-11-16。
-- 官服 技能4.pcapng 实证：13m 外按压即受理冲锋（skill_static.RushRange=20，
-- 手勘表 rushDashChargeRangeSeeds 同值兜底）——基础 3m 冲锋被武技扩到
-- 20m 远冲。本模块演示 rush 指令（字段级覆盖，零值=回落 INI/族规则；
-- GM lua_reload 热更）：扩程值实际由 RushRange 实据自动生效，此处的
-- charge_range_m 是显式等值声明（部署侧 INI 缺行时接管）。
-- 字段白名单：
--   direct_band_m  直收带上界（米）        charge_lo_m    冲锋带下界（米）
--   charge_range_m 冲锋带上界（米=扩程值）  level          c61 档 1=地面G/2=空中A
--   action / air_action 动作名覆盖（qg_/fi_/swim_ 前缀）
local M = {}

function M.compile(def)
  return {
    rush = {
      charge_range_m = 20,                      -- 落尘：3m 冲锋 → 20m
      air_action = "qg_fastruning_s_2h_01",     -- 空中奔袭（技能4 全样本 flag2 动作）
    },
  }
end

return M
