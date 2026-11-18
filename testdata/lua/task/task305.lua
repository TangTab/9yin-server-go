-----------------------------------------
-- @ 脚本名称: task305.lua
-- @ 脚本功能: 305任务特殊处理逻辑
-- @ 创建者:   wanglei
-- @ 创建日期: 2009.12.17
-----------------------------------------

-- 每个单独的任务脚本可以复制此模板，修改逻辑


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
	-- 开启新手帮助
	nx_set_property(player, "HelpStepID", "lifehelp_00")
	return 1
end
