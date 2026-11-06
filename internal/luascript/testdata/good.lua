-- 测试夹具：覆盖 game.on 注册、ctx 快照读取与 game.log 格式化。
-- 注意：chunk 内一旦出现 Lua 5.5 的 global 声明就会开启严格模式
-- （所有全局引用都必须声明），因此这里沿用隐式全局。
game.on("npc_service", function(ctx)
  game.log("npc_service npc=%s mark=%d", ctx:npc().config_id, ctx:service().mark)
  ctx:notify("你好，" .. ctx:player().name, "ui_shop")
  return true
end)

game.on("decline", function(ctx)
  return false
end)

game.on("echo_player", function(ctx)
  local p = ctx:player()
  game.log("player=%s silver=%d scene=%s x=%.1f", p.name, p.silver, p.scene, p.x)
  return true
end)
