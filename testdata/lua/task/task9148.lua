-----------------------------------------
-- @ 脚本名称: task9148.lua
-- @ 脚本功能: ID为9148的任务执行脚本逻辑
-- @ 创建者:   wwl
-- @ 创建日期: 2009.8.07
-----------------------------------------


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)

	-- 增加标记记录表一行
	local row = nx_add_record_row_values(player, "Task_Record", nx_int(9148), nx_int(8),
				nx_int(0), nx_int(2), nx_string("使用药品"), nx_string("苏州"), nx_widestr("苏州"))
	
	if row < 0 then
		return 0
	end
	
	return 1
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)

	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)

	return 1
end