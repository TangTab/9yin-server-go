-- 探针：把收到的 ctx 逐字段回显进 state，供 Go 侧断言投递正确。
local M = {}
M.meta = { id = "probe_ctx", name = "探针·ctx", book = "ng_probe_ctx",
           category = "hit", max_tier = 1 }
function M.on_attack(ctx)
  return { state = {
    book_level = ctx.book_level, book_max_level = ctx.book_max_level,
    story_stage = ctx.story_stage, now = ctx.now, damage = ctx.damage,
    incoming = ctx.incoming, parried = ctx.parried, crit = ctx.crit,
    controlled = ctx.controlled, since_last_ms = ctx.since_last_ms,
    self_hp = ctx.self.hp, self_max_hp = ctx.self.max_hp, self_mp = ctx.self.mp,
    self_max_mp = ctx.self.max_mp, self_attack = ctx.self.attack,
    target_oid = ctx.target and ctx.target.oid or -1,
    target_hp = ctx.target and ctx.target.hp or -1,
    attacker_oid = ctx.attacker and ctx.attacker.oid or -1,
    r_proc = ctx.rand.r_proc,
    has_state = (ctx.state ~= nil),
    state_layers = ctx.state and ctx.state.layers or -1,
  } }
end
return M
