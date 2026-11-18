-----------------------------------------
-- @ 脚本名称: task525.lua
-- @ 脚本功能: 525任务特殊处理逻辑
-- @ 创建者:   fengqw
-- @ 创建日期: 2009.12.17
-----------------------------------------

-- 每个单独的任务脚本可以复制此模板，修改逻辑


-- 任务承接条件判断，在接任务前看是否有接任务的条件
function Task_Accept_Condition(player, npc, taskid)
	return 1
end


-- 任务接受后，处理任务接受后各种不通用的事件
function Task_After_Accept(player, npc, taskid)

	-- 当前场景
	local scene = nx_get_scene()
	local scene_id = nx_query_property(scene, "ConfigID")
	local origin_id = "sta_slin0102"
	local config_id = "tasklua_525" 

	-- 如果拥有这个身份直接刷新进度
	if nx_is_owned_origin(player, origin_id) then
		-- 任务进度表增加一行
		local row = nx_add_record_row_values(player, "Task_Record", nx_int(taskid), nx_int(0),
					   nx_int(1), nx_int(1), origin_id, nx_string(scene_id), config_id)

		if row < 0 then
			return 0
		end

		--刷新任务进度
		nx_refresh_task_flag(player)

	else
		-- 任务进度表增加一行
		local row = nx_add_record_row_values(player, "Task_Record", nx_int(taskid), nx_int(0),
					   nx_int(0), nx_int(1), origin_id, nx_string(scene_id), config_id)

		if row < 0 then
			return 0
		end

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

