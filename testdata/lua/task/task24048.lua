-----------------------------------------
-- @ 脚本名称: task24048.lua
-- @ 脚本功能: 24048获得门派身份脚本
-- @ 创建者:   fengqw
-- @ 创建日期: 2010.4.17
-----------------------------------------

-- 每个单独的任务脚本可以复制此模板，修改逻辑


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)

	-- 当前场景
	local config_id = "tasklua_24048"

	-- 任务进度表增加一行
	local row = nx_add_record_row_values(player, "Task_Record", nx_int(taskid), nx_int(0),nx_int(0), nx_int(1), "", "", config_id)
	local max_rows = nx_get_record_rows(target, "Task_Record")
	for i = 0, max_rows-1 do
		local id = nx_query_record_value(target, "Task_Record", i, 0)
		local config = nx_query_record_value(target, "Task_Record", i, 6)
		if nx_int(taskid) == nx_int(id) then
			if config_id == config then
				--查询当前进度
				local numnow = nx_query_record_value(target, "Task_Record", i, 2)
				local numneed = nx_query_record_value(target, "Task_Record", i, 3)
				if numnow < numneed then
					return 1
				else
				--提交任务
				nx_command(player, player, 165, taskid)
				end
			end
		end
	return 1
	end
end


-- 任务完成条件判断，在交任务前看是否可以完成任务
function Task_Complete_Condition(player, npc, taskid)


	return 1
end


-- 任务完成后出，处理任务提交后的不通用事件
function Task_After_Complete(player, npc, taskid)

	return 1
end

