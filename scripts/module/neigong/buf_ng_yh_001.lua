-- 幻玉神功（buf_ng_yh_001）｜移花宫｜受击触发
-- 源：res1 内功特效清单.txt 行711-717；逐层数值勘定自现代客户端
--   desc2.idres desc_mind_buf_ng_yh_001_<层>_ngtips（7 层）。
-- 料敌先机：被攻击时 50% 几率提升自身闪避率 20%，持续 5-11 秒，36-26 秒冷却
--   （on_victim → dodge_add 指令；dodge_add/on_dodge 为本仓对参考服白名单
--   的扩展，参考服脚本中该机制为 TODO，见 docs/skill-script-framework.md）。
-- 移花接玉：成功闪避时恢复 2-5% 最大气血，4 秒冷却（on_dodge）；
--   层 6-7：并使下一击伤害提高 60%（state 充能，on_attack 附伤兑现）。
local M = {}
M.meta = { id = "buf_ng_yh_001", name = "幻玉神功", book = "ng_yh_001",
           category = "victim", max_tier = 7 }
M.defaults = { proc = 0.50, heal_cd_ms = 4000, dodge = 20 }
M.levels = {
  [1] = { dur_ms = 5000,  proc_cd_ms = 36000, heal_pct = 2 },
  [2] = { dur_ms = 6000,  proc_cd_ms = 34000, heal_pct = 2 },
  [3] = { dur_ms = 7000,  proc_cd_ms = 32000, heal_pct = 3 },
  [4] = { dur_ms = 8000,  proc_cd_ms = 30000, heal_pct = 3 },
  [5] = { dur_ms = 9000,  proc_cd_ms = 28000, heal_pct = 4 },
  [6] = { dur_ms = 10000, proc_cd_ms = 27000, heal_pct = 4, next_hit_pct = 60 },
  [7] = { dur_ms = 11000, proc_cd_ms = 26000, heal_pct = 5, next_hit_pct = 60 },
}

local function tier_of(ctx)
  local mt = M.meta.max_tier
  local bl, bm = ctx.book_level or 1, ctx.book_max_level or mt
  -- 九阴志档位门：高层特效档位随全服九阴志阶段开放（阶段1→最高3档，每阶段+1；可调）
  local cap = 3 + (ctx.story_stage or 1) - 1
  return math.max(1, math.min(mt, math.ceil(bl * mt / bm), cap))
end
local function lv(ctx) return M.levels[tier_of(ctx)] or M.levels[M.meta.max_tier] end
local function roll(ctx, p) return (ctx.rand and ctx.rand.r_proc or 1) < p end

-- state 是整体回写：保留仍有效的窗口键（charged 由 on_dodge 充能、
-- on_attack 兑现；proc_until/heal_until 冷却窗跨事件共享）。
local function carry(s, st)
  if s.charged then st.charged = true end
  if s.proc_until then st.proc_until = s.proc_until end
  if s.heal_until then st.heal_until = s.heal_until end
  return st
end

function M.on_victim(ctx)
  local p, s = lv(ctx), ctx.state or {}
  local now = ctx.now or 0
  if s.proc_until and now < s.proc_until then return nil end
  if not roll(ctx, M.defaults.proc) then return nil end
  return {
    dodge_add = { amount = M.defaults.dodge, life_ms = p.dur_ms },
    state = carry(s, { proc_until = now + p.proc_cd_ms }),
  }
end

function M.on_dodge(ctx)
  local p, s = lv(ctx), ctx.state or {}
  local now = ctx.now or 0
  if s.heal_until and now < s.heal_until then return nil end
  local st = carry(s, { heal_until = now + M.defaults.heal_cd_ms })
  if p.next_hit_pct and p.next_hit_pct > 0 then st.charged = true end
  return {
    heal = math.floor((ctx.self.max_hp or 0) * p.heal_pct / 100),
    state = st,
  }
end

function M.on_attack(ctx)
  local s = ctx.state or {}
  if not s.charged then return nil end
  local bonus = math.floor((ctx.damage or 0) * (lv(ctx).next_hit_pct or 0) / 100)
  if bonus <= 0 then return nil end
  -- 兑现充能：清除 charged，保留冷却窗。
  local st = {}
  if s.proc_until then st.proc_until = s.proc_until end
  if s.heal_until then st.heal_until = s.heal_until end
  return { extra_damage = { amount = bonus, kind = "yu" }, state = st }
end

return M
