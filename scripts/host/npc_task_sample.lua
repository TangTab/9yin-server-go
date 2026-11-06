-- npc_service 事件示例：接管"任务"服务（mark 0x1008）。
-- 原生已实现的服务（商店 0x1001 / 仓库 0x1002）不会进入脚本；
-- 其余原生未实现的服务在这里显式交回（return false → 维持"未实现"）。

local MARK_TASK = 0x1008

game.on("npc_service", function(ctx)
	local service = ctx:service()
	if service.mark ~= MARK_TASK then
		return false
	end
	local player = ctx:player()
	local npc = ctx:npc()
	local tasks = ctx:tasks()
	game.log("task service opened by %s at npc %s in scene %s: %d bound task(s)",
		player.name, npc.config_id, player.scene, #tasks)
	for i = 1, #tasks do
		local task = tasks[i]
		game.log("  task %d: %s", task.id, task.title)
	end
	ctx:notify(string.format("侠士眼前有 %d 桩江湖事，接取尚未开放。", #tasks), "ui_shop")
	-- 也可以让 NPC 头顶冒气泡（需客户端已装的文本 ID）：
	-- ctx:bubble("ui_talk")
	return true
end)
