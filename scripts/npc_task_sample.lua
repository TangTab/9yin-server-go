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
	game.log("task service opened by %s at npc %s in scene %s",
		player.name, npc.config_id, player.scene)
	ctx:notify("任务系统尚未开放，侠士且先行游历江湖。", "ui_shop")
	-- 也可以让 NPC 头顶冒气泡（需客户端已装的文本 ID）：
	-- ctx:bubble("ui_talk")
	return true
end)
