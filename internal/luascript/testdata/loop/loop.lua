-- 测试夹具：死循环处理器，用于验证超时取消。
game.on("loop", function(ctx)
  local i = 0
  while true do
    i = i + 1
  end
end)
