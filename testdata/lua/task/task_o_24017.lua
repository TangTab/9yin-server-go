-----------------------------------------
-- @ 脚本名称: task_o_24017.lua
-- @ 脚本功能: 24017号任务完成后刷怪
-- @ 创建者:   fengqw
-- @ 创建日期: 2010.3.18
-----------------------------------------

-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)
	return 1
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)
	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)
	local npc_config = "monrv040200"
	--创建npc到场景
	nx_create_obj(npc_config, 1, 91, 11, 381, 0)
	nx_create_obj(npc_config, 1, 95, 10, 383, 0)
	return 1
end

